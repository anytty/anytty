package remoteauth

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/filelock"
	"github.com/anytty/anytty/shared/securefs"
)

const (
	accessStoreFile             = "remote_client_access.json"
	accessStoreLockFile         = "remote_client_access.lock"
	accessStoreVersion          = 4
	accessStateSignatureDomain  = "anytty.remoteauth.access-state.v3"
	pairingReceiptDomain        = "anytty.remoteauth.pairing-receipt.v1"
	defaultDeliveryGrace        = 24 * time.Hour
	expiredTicketRetention      = 24 * time.Hour
	expiredGrantRecordRetention = 30 * 24 * time.Hour
)

// PairingExchangeResult 是 daemon 已原子持久化的 ticket redemption 结果。
// Grant 与 DeliveryReceipt 在相同 ticket/ClientAccessIdentity 的 delivery grace 重试中必须逐字节稳定；只有 Grant 会进入后续 capability handshake。
type PairingExchangeResult struct {
	TicketID              string
	Bundle                []byte
	Grant                 string
	GrantID               string
	DeliveryReceipt       string
	SubjectKeyFingerprint string
	Scope                 Scope
	ExpiresAt             time.Time
	CloudRouteGrant       []byte
	CloudEdgeLocator      []byte
}

// ClientAccessRecord 是 local owner 或 ManageClientAccess session 可读取的脱敏授权投影。
// 它不包含 grant body、ticket、public key bytes 或 private material；RevokedAt 为零表示当前未撤销。
type ClientAccessRecord struct {
	GrantID               string
	RevocationID          string
	SubjectKeyFingerprint string
	ClientLabel           string
	Scope                 Scope
	IssuedAt              time.Time
	ExpiresAt             time.Time
	RevokedAt             time.Time
}

// AccessStoreOptions 只为 deterministic harness 注入时间与随机源。
// 生产调用保持零值，使用 UTC 当前时间和 crypto/rand；这些选项不允许来自 Cloud 或远端请求。
type AccessStoreOptions struct {
	Now    func() time.Time
	Random io.Reader
}

// AccessStore 是 owning daemon 的 PairingTicket 摘要、客户端 key 绑定、grant claims、delivery receipt 摘要与撤销唯一持久真值。
// 一个 state 目录只允许一个进程 owner；普通 capability 验证只读内存且不写磁盘，低频 mutation 才批量 compact 并原子替换签名 state。
type AccessStore struct {
	mu                           sync.RWMutex
	pairingClaimsMu              sync.Mutex
	path                         string
	identity                     Identity
	tickets                      map[string]storedPairingTicket
	grants                       map[string]storedAccessGrant
	pairingClaims                map[[sha256.Size]byte]storedPairingClaim
	now                          func() time.Time
	random                       io.Reader
	owner                        *filelock.Lock
	writeFile                    func(string, []byte) error
	closed                       bool
	accessProjectionRevision     uint64
	changes                      chan struct{}
	managedRouteGrantIssuer      func(ed25519.PublicKey, uint32, time.Time, time.Time) ([]byte, []byte, error)
	managedPairingBootstrapIssue func() (*remoteauthpb.PairingManagedRouteSeed, error)
	clientGrants                 map[string]map[string]struct{}
}

// AccessSnapshot 是不取得 daemon mutation owner lock 的只读撤销快照。
// 它只供离线诊断和测试验证已签名 state；生产 remote ingress 必须持有 AccessStore，不能用 snapshot 代替 owner。
type AccessSnapshot struct {
	grants map[string]storedAccessGrant
}

type storedAccessState struct {
	Version                  uint32                         `json:"version"`
	IssuerDeviceID           string                         `json:"issuer_device_id"`
	IssuerFingerprint        string                         `json:"issuer_fingerprint"`
	Tickets                  map[string]storedPairingTicket `json:"tickets"`
	Grants                   map[string]storedAccessGrant   `json:"grants"`
	StateSignature           string                         `json:"state_signature"`
	AccessProjectionRevision uint64                         `json:"access_projection_revision"`
}

type storedPairingTicket struct {
	Claims                 PairingTicketClaims `json:"claims"`
	TicketDigest           string              `json:"ticket_digest"`
	SubjectKeyFingerprint  string              `json:"subject_key_fingerprint,omitempty"`
	ClientLabel            string              `json:"client_label,omitempty"`
	GrantID                string              `json:"grant_id,omitempty"`
	ResultGrantDigest      string              `json:"result_grant_digest,omitempty"`
	DeliveryReceiptDigest  string              `json:"delivery_receipt_digest,omitempty"`
	RedeemedAt             time.Time           `json:"redeemed_at,omitempty"`
	DeliveryGraceUntil     time.Time           `json:"delivery_grace_until,omitempty"`
	CloudRouteGrantDigest  string              `json:"cloud_route_grant_digest,omitempty"`
	CloudEdgeLocatorDigest string              `json:"cloud_edge_locator_digest,omitempty"`
}

type storedAccessGrant struct {
	Claims           Claims    `json:"claims"`
	ClientLabel      string    `json:"client_label,omitempty"`
	RevokedAt        time.Time `json:"revoked_at,omitempty"`
	CloudRouteGrant  []byte    `json:"cloud_route_grant,omitempty"`
	CloudEdgeLocator []byte    `json:"cloud_edge_locator,omitempty"`
}

// LoadAccessStore 加载或创建 daemon-local client access store，并取得该目录的唯一进程 owner lock。
// 第二个 daemon、损坏 state、签名错误或 issuer identity 不一致必须阻止 remote ingress，不能按空 store 或独立 revocation set 继续运行。
func LoadAccessStore(dir string, identity Identity, options AccessStoreOptions) (*AccessStore, error) {
	if err := identity.Validate(); err != nil {
		return nil, fmt.Errorf("load client access store: %w", err)
	}
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, fmt.Errorf("client access store requires directory")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create client access directory: %w", err)
	}
	if err := securefs.SecureDirectory(dir); err != nil {
		return nil, fmt.Errorf("secure client access directory: %w", err)
	}
	owner, err := filelock.Acquire(filepath.Join(dir, accessStoreLockFile), true)
	if err != nil {
		return nil, fmt.Errorf("acquire client access owner lock: %w", err)
	}
	store := &AccessStore{
		path: filepath.Join(dir, accessStoreFile), identity: identity, owner: owner,
		tickets: map[string]storedPairingTicket{}, grants: map[string]storedAccessGrant{}, pairingClaims: map[[sha256.Size]byte]storedPairingClaim{}, changes: make(chan struct{}, 1), clientGrants: make(map[string]map[string]struct{}),
		now: options.Now, random: options.Random, writeFile: writePrivateFile,
	}
	if store.now == nil {
		store.now = func() time.Time { return time.Now().UTC() }
	}
	if store.random == nil {
		store.random = rand.Reader
	}
	state, found, err := loadStoredAccessState(store.path, identity)
	if err != nil {
		_ = owner.Close()
		return nil, err
	}
	if found {
		store.tickets = state.Tickets
		store.grants = state.Grants
		store.accessProjectionRevision = state.AccessProjectionRevision
		store.rebuildClientGrantIndexLocked()
		if err := store.validateLoadedState(); err != nil {
			_ = owner.Close()
			return nil, err
		}
		if err := securefs.SecureFile(store.path); err != nil {
			_ = owner.Close()
			return nil, fmt.Errorf("secure client access store: %w", err)
		}
	}
	return store, nil
}

// ConfigureManagedRouteGrantIssuer 安装 daemon Cloud owner 的可选 grant issuer。
// issuer 只在 PairingExchange 原子事务内调用，必须返回 DeviceIdentity 签名的 Cloud Proto bytes；重复配置会失败。
func (store *AccessStore) ConfigureManagedRouteGrantIssuer(issuer func(ed25519.PublicKey, uint32, time.Time, time.Time) ([]byte, []byte, error)) error {
	if store == nil || issuer == nil {
		return errors.New("managed Route grant issuer is required")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.ensureOpenLocked(); err != nil {
		return err
	}
	if store.managedRouteGrantIssuer != nil {
		return errors.New("managed Route grant issuer is already configured")
	}
	store.managedRouteGrantIssuer = issuer
	return nil
}

// ConfigureManagedPairingBootstrapIssuer 安装紧凑 Cloud pairing 入口投影。
// issuer 只能返回公开 Edge 入口与 CA pin，不能读取 claim 或签发 bearer grant。
func (store *AccessStore) ConfigureManagedPairingBootstrapIssuer(issuer func() (*remoteauthpb.PairingManagedRouteSeed, error)) error {
	if store == nil || issuer == nil {
		return errors.New("managed pairing bootstrap issuer is required")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.ensureOpenLocked(); err != nil {
		return err
	}
	if store.managedPairingBootstrapIssue != nil {
		return errors.New("managed pairing bootstrap issuer is already configured")
	}
	store.managedPairingBootstrapIssue = issuer
	return nil
}

// DisableManagedCloudRoute removes the current Cloud-only pairing and route issuers.
// Direct and SSH grants remain owned by the same AccessStore.
func (store *AccessStore) DisableManagedCloudRoute() error {
	if store == nil {
		return errors.New("client access store is required")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.ensureOpenLocked(); err != nil {
		return err
	}
	store.managedRouteGrantIssuer = nil
	store.managedPairingBootstrapIssue = nil
	return nil
}

// AccessProjectionRevision 返回 grant 管理投影的持久单调 revision。
// pairing ticket 本身不改变该 revision；grant 新增、撤销或 retention 清理才推进。
func (store *AccessStore) AccessProjectionRevision() uint64 {
	if store == nil {
		return 0
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	if store.closed {
		return 0
	}
	return store.accessProjectionRevision
}

// AccessChanges 返回 grant 管理投影的有界变更通知。
// 通知可以合并；consumer 每次必须重新读取完整 projection 与 revision。
func (store *AccessStore) AccessChanges() <-chan struct{} {
	if store == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return store.changes
}

// LoadAccessSnapshot 验证并读取 daemon-local state 的只读撤销快照，不竞争 mutation owner lock。
// snapshot 不提供 pairing、签发、撤销或 core ingress；文件缺失表示空快照，state 损坏仍 fail closed。
func LoadAccessSnapshot(dir string, identity Identity) (*AccessSnapshot, error) {
	if err := identity.Validate(); err != nil {
		return nil, fmt.Errorf("load client access snapshot: %w", err)
	}
	state, found, err := loadStoredAccessState(filepath.Join(strings.TrimSpace(dir), accessStoreFile), identity)
	if err != nil {
		return nil, err
	}
	if !found {
		return &AccessSnapshot{grants: map[string]storedAccessGrant{}}, nil
	}
	store := &AccessStore{identity: identity, tickets: state.Tickets, grants: state.Grants, accessProjectionRevision: state.AccessProjectionRevision}
	if err := store.validateLoadedState(); err != nil {
		return nil, err
	}
	return &AccessSnapshot{grants: cloneGrantRecords(state.Grants)}, nil
}

// Close 释放 AccessStore 的唯一进程 owner lock。
// close 后所有验证均 fail closed，mutation 返回错误；daemon 必须在 listener 与 remote session 全部停止后调用。
func (store *AccessStore) Close() error {
	if store == nil {
		return nil
	}
	store.mu.Lock()
	if store.closed {
		store.mu.Unlock()
		return nil
	}
	store.closed = true
	store.pairingClaimsMu.Lock()
	clear(store.pairingClaims)
	store.pairingClaimsMu.Unlock()
	owner := store.owner
	store.owner = nil
	store.mu.Unlock()
	return owner.Close()
}

// Available 返回当前 AccessStore 是否仍持有 daemon-local mutation owner lock。
// remote ingress 只可在 true 时发送 DeviceHello；false 表示 daemon 正在关闭或 owner truth 已丢失，必须在协议开始前 fail closed。
func (store *AccessStore) Available() bool {
	if store == nil {
		return false
	}
	store.mu.RLock()
	available := !store.closed && store.owner != nil
	store.mu.RUnlock()
	return available
}

// IssuePairingBundle 原子登记并返回一个短期一次性 PairingTicket bootstrap。
// 持久 state 只保存 canonical bundle digest 与 claims；签发成功前 bundle 不对调用方可见，持久化失败不会留下可兑换记录。
func (store *AccessStore) IssuePairingBundle(options PairingIssueOptions) (*PairingBundle, PairingTicketClaims, error) {
	if store == nil {
		return nil, PairingTicketClaims{}, fmt.Errorf("client access store is nil")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.ensureOpenLocked(); err != nil {
		return nil, PairingTicketClaims{}, err
	}
	if options.Now.IsZero() {
		options.Now = store.now().UTC()
	}
	if options.Random == nil {
		options.Random = store.random
	}
	bundle, claims, err := issuePairingBundle(store.identity, options)
	if err != nil {
		return nil, PairingTicketClaims{}, err
	}
	payload, err := EncodePairingBundle(bundle)
	if err != nil {
		return nil, PairingTicketClaims{}, err
	}
	if err := store.persistPairingBundleLocked(payload, claims, options.Now.UTC()); err != nil {
		return nil, PairingTicketClaims{}, err
	}
	return bundle, claims, nil
}

func (store *AccessStore) persistPairingBundleLocked(payload []byte, claims PairingTicketClaims, now time.Time) error {
	if _, exists := store.tickets[claims.TicketID]; exists {
		return fmt.Errorf("pairing ticket id collision")
	}
	oldTickets, oldGrants := cloneTicketRecords(store.tickets), cloneGrantRecords(store.grants)
	oldRevision := store.accessProjectionRevision
	if store.compactLocked(now.UTC()) {
		store.accessProjectionRevision++
	}
	store.tickets[claims.TicketID] = storedPairingTicket{Claims: claims, TicketDigest: payloadDigest(payload)}
	if err := store.persistLocked(); err != nil {
		if !privateFileWritePublished(err) {
			store.tickets, store.grants, store.accessProjectionRevision = oldTickets, oldGrants, oldRevision
		}
		return err
	}
	if store.accessProjectionRevision != oldRevision {
		store.notifyAccessChangedLocked()
	}
	return nil
}

// RedeemPairingBundle 原子消费 canonical bootstrap、绑定 client public key、签发 CapabilityGrant v2 并保存稳定结果摘要。
// 相同 bundle 与同 key 只在短 delivery grace 内重建原结果且不写磁盘；不同 key、错误 issuer、过期 ticket 或持久化失败全部 fail closed。
func (store *AccessStore) RedeemPairingBundle(payload []byte, clientPublicKey ed25519.PublicKey, clientLabel string, now time.Time) (PairingExchangeResult, error) {
	return store.redeemPairingBundle(payload, clientPublicKey, clientLabel, 0, now)
}

// RedeemPairingBundleForProduct 在普通 capability 原子事务中一并保存可选 CloudRouteGrant。
// shared remoteauth 不解释 Cloud schema，只把 Proto ClientProduct 数值交给 daemon Cloud owner。
func (store *AccessStore) RedeemPairingBundleForProduct(payload []byte, clientPublicKey ed25519.PublicKey, clientLabel string, product uint32, now time.Time) (PairingExchangeResult, error) {
	return store.redeemPairingBundle(payload, clientPublicKey, clientLabel, product, now)
}

func (store *AccessStore) redeemPairingBundle(payload []byte, clientPublicKey ed25519.PublicKey, clientLabel string, product uint32, now time.Time) (PairingExchangeResult, error) {
	if store == nil {
		return PairingExchangeResult{}, fmt.Errorf("client access store is nil")
	}
	if len(clientPublicKey) != ed25519.PublicKeySize {
		return PairingExchangeResult{}, fmt.Errorf("%w: client public key is invalid", ErrPairingTicketMalformed)
	}
	if now.IsZero() {
		now = store.now().UTC()
	} else {
		now = now.UTC()
	}
	clientLabel, err := normalizePairingLabel(clientLabel, "pairing client label")
	if err != nil {
		return PairingExchangeResult{}, err
	}
	bundle, claims, err := ParsePairingBundleForExchange(payload)
	if err != nil {
		return PairingExchangeResult{}, err
	}
	if claims.IssuerDeviceID != store.identity.DeviceID ||
		subtle.ConstantTimeCompare([]byte(bundle.GetIdentity().GetDeviceFingerprint()), []byte(store.identity.Fingerprint)) != 1 {
		return PairingExchangeResult{}, ErrGrantFingerprintMismatch
	}
	canonicalPayload, err := EncodePairingBundle(bundle)
	if err != nil {
		return PairingExchangeResult{}, err
	}
	ticketDigest := payloadDigest(canonicalPayload)
	subjectFingerprint := Fingerprint(clientPublicKey)
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.ensureOpenLocked(); err != nil {
		return PairingExchangeResult{}, err
	}
	record, ok := store.tickets[claims.TicketID]
	if !ok || subtle.ConstantTimeCompare([]byte(record.TicketDigest), []byte(ticketDigest)) != 1 {
		return PairingExchangeResult{}, ErrPairingTicketMalformed
	}
	if record.GrantID != "" {
		if record.SubjectKeyFingerprint != subjectFingerprint {
			return PairingExchangeResult{}, ErrPairingTicketConsumed
		}
		if record.DeliveryGraceUntil.IsZero() || !now.Before(record.DeliveryGraceUntil) {
			return PairingExchangeResult{}, ErrPairingTicketConsumed
		}
		return store.pairingResultFromStored(record)
	}
	if err := validatePairingTicketTime(claims, now); err != nil {
		return PairingExchangeResult{}, err
	}
	grantID, err := randomIdentifier(store.random, "grant-")
	if err != nil {
		return PairingExchangeResult{}, fmt.Errorf("generate bound grant id: %w", err)
	}
	if _, exists := store.grants[grantID]; exists {
		return PairingExchangeResult{}, fmt.Errorf("bound grant id collision")
	}
	grantNonce, err := randomIdentifier(store.random, "")
	if err != nil {
		return PairingExchangeResult{}, fmt.Errorf("generate bound grant nonce: %w", err)
	}
	grantClaims := normalizeClaims(Claims{
		Version: 2, GrantID: grantID, IssuerDeviceID: store.identity.DeviceID, IssuerDeviceFingerprint: store.identity.Fingerprint,
		SubjectKeyFingerprint: subjectFingerprint, Scope: claims.ScopeCeiling, IssuedAt: now,
		NotBefore: now.Add(-ClockSkewTolerance), ExpiresAt: now.Add(time.Duration(claims.GrantLifetimeSeconds) * time.Second),
		RevocationID: grantID, Nonce: grantNonce,
	})
	boundGrant, err := Issue(store.identity.PrivateKey, grantClaims)
	if err != nil {
		return PairingExchangeResult{}, err
	}
	receipt := pairingDeliveryReceipt(store.identity.PrivateKey, claims.TicketID, subjectFingerprint, grantID)
	var cloudRouteGrant, cloudEdgeLocator []byte
	hasManagedRoute := pairingBundleHasManagedRoute(bundle)
	if hasManagedRoute {
		if store.managedRouteGrantIssuer == nil {
			return PairingExchangeResult{}, errors.New("managed Route grant issuer is unavailable")
		}
		cloudRouteGrant, cloudEdgeLocator, err = store.managedRouteGrantIssuer(append(ed25519.PublicKey(nil), clientPublicKey...), product, now, grantClaims.ExpiresAt)
		if err != nil {
			return PairingExchangeResult{}, fmt.Errorf("issue managed Route grant: %w", err)
		}
		if len(cloudRouteGrant) == 0 || len(cloudEdgeLocator) == 0 {
			return PairingExchangeResult{}, errors.New("managed Route grant issuer returned incomplete route material")
		}
	}
	updatedTicket := record
	updatedTicket.SubjectKeyFingerprint = subjectFingerprint
	updatedTicket.ClientLabel = clientLabel
	updatedTicket.GrantID = grantID
	updatedTicket.ResultGrantDigest = payloadDigest([]byte(boundGrant))
	updatedTicket.DeliveryReceiptDigest = payloadDigest([]byte(receipt))
	updatedTicket.RedeemedAt = now
	updatedTicket.DeliveryGraceUntil = now.Add(defaultDeliveryGrace)
	if len(cloudRouteGrant) != 0 {
		updatedTicket.CloudRouteGrantDigest = payloadDigest(cloudRouteGrant)
		updatedTicket.CloudEdgeLocatorDigest = payloadDigest(cloudEdgeLocator)
	}
	grantRecord := storedAccessGrant{Claims: grantClaims, ClientLabel: clientLabel, CloudRouteGrant: append([]byte(nil), cloudRouteGrant...), CloudEdgeLocator: append([]byte(nil), cloudEdgeLocator...)}
	oldTickets, oldGrants := cloneTicketRecords(store.tickets), cloneGrantRecords(store.grants)
	oldRevision := store.accessProjectionRevision
	store.compactLocked(now)
	store.tickets[claims.TicketID] = updatedTicket
	store.grants[grantID] = grantRecord
	store.indexGrantLocked(subjectFingerprint, grantID)
	store.accessProjectionRevision++
	if err := store.persistLocked(); err != nil {
		if !privateFileWritePublished(err) {
			store.tickets, store.grants, store.accessProjectionRevision = oldTickets, oldGrants, oldRevision
			store.rebuildClientGrantIndexLocked()
		}
		return PairingExchangeResult{}, err
	}
	store.notifyAccessChangedLocked()
	return PairingExchangeResult{
		TicketID: claims.TicketID, Grant: boundGrant, GrantID: grantID, DeliveryReceipt: receipt,
		SubjectKeyFingerprint: subjectFingerprint, Scope: grantClaims.Scope, ExpiresAt: grantClaims.ExpiresAt, CloudRouteGrant: append([]byte(nil), cloudRouteGrant...), CloudEdgeLocator: append([]byte(nil), cloudEdgeLocator...),
	}, nil
}

// AllowsClientPublicKey 是 AgentGateway 处理 Cloud offer 前的 daemon-local AccessStore 预检查。
// 索引是持久 grants 的内存派生值；不存在、过期或全部撤销均 fail closed。
func (store *AccessStore) AllowsClientPublicKey(publicKey ed25519.PublicKey, now time.Time) bool {
	if store == nil || len(publicKey) != ed25519.PublicKeySize {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	fingerprint := Fingerprint(publicKey)
	store.mu.RLock()
	defer store.mu.RUnlock()
	if store.closed {
		return false
	}
	for grantID := range store.clientGrants[fingerprint] {
		record, ok := store.grants[grantID]
		if ok && record.RevokedAt.IsZero() && record.Claims.NotBefore.Before(now.Add(ClockSkewTolerance)) && record.Claims.ExpiresAt.After(now.Add(-ClockSkewTolerance)) {
			return true
		}
	}
	return false
}

// ListClientAccess 返回按 GrantID 排序的脱敏 client-bound grant 投影。
// 该只读操作不刷新时间戳、不清理过期记录且不写磁盘，便于 daemon 在 Cloud/数据库不可用时继续离线管理。
func (store *AccessStore) ListClientAccess() []ClientAccessRecord {
	if store == nil {
		return nil
	}
	store.mu.RLock()
	if store.closed {
		store.mu.RUnlock()
		return nil
	}
	records := make([]ClientAccessRecord, 0, len(store.grants))
	for _, grant := range store.grants {
		records = append(records, clientAccessRecordFromStored(grant))
	}
	store.mu.RUnlock()
	sort.Slice(records, func(i, j int) bool { return records[i].GrantID < records[j].GrantID })
	return records
}

// GrantActive 是 core transport admission 与 Direct preauth 共用的窄只读查询。
// 请求必须携带已验证 claims 的规范 GrantID 和绝对期限；未知、撤销、过期或期限不匹配都 fail closed。
func (store *AccessStore) GrantActive(grantID string, expiresAt, now time.Time) bool {
	if store == nil {
		return false
	}
	grantID = strings.TrimSpace(grantID)
	if grantID == "" || expiresAt.IsZero() {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	store.mu.RLock()
	record, ok := store.grants[grantID]
	active := !store.closed && ok && record.RevokedAt.IsZero() &&
		record.Claims.GrantID == grantID && record.Claims.ExpiresAt.Equal(expiresAt.UTC()) && now.UTC().Before(record.Claims.ExpiresAt)
	store.mu.RUnlock()
	return active
}

// RevokeGrant 原子撤销指定 client-bound grant，重启后仍由同一 AccessStore 拒绝普通 capability handshake。
// 重复撤销保持幂等且不重写 state；未知 GrantID 返回错误，调用方不能用删除客户端本地 ref 冒充 daemon 撤销。
func (store *AccessStore) RevokeGrant(grantID string) (ClientAccessRecord, error) {
	if store == nil {
		return ClientAccessRecord{}, fmt.Errorf("client access store is nil")
	}
	grantID = strings.TrimSpace(grantID)
	if grantID == "" {
		return ClientAccessRecord{}, fmt.Errorf("client access revoke requires grant_id")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.ensureOpenLocked(); err != nil {
		return ClientAccessRecord{}, err
	}
	record, ok := store.grants[grantID]
	if !ok {
		return ClientAccessRecord{}, fmt.Errorf("client access grant not found")
	}
	if !record.RevokedAt.IsZero() {
		return clientAccessRecordFromStored(record), nil
	}
	oldTickets, oldGrants := cloneTicketRecords(store.tickets), cloneGrantRecords(store.grants)
	oldRevision := store.accessProjectionRevision
	now := store.now().UTC()
	store.compactLocked(now)
	record.RevokedAt = now
	store.grants[grantID] = record
	store.accessProjectionRevision++
	if err := store.persistLocked(); err != nil {
		if !privateFileWritePublished(err) {
			store.tickets, store.grants, store.accessProjectionRevision = oldTickets, oldGrants, oldRevision
		}
		return ClientAccessRecord{}, err
	}
	store.notifyAccessChangedLocked()
	return clientAccessRecordFromStored(record), nil
}

// Revoked 实现普通 capability handshake 的 daemon-local 只读撤销查询。
// 未登记 ID 或已关闭 store 一律 fail closed；已知有效记录只读内存，不访问 Cloud、文件或数据库，也不产生写入。
func (store *AccessStore) Revoked(revocationID string) bool {
	if store == nil {
		return true
	}
	store.mu.RLock()
	record, ok := store.grants[strings.TrimSpace(revocationID)]
	closed := store.closed
	store.mu.RUnlock()
	return closed || !ok || !record.RevokedAt.IsZero()
}

// Revoked 返回只读 snapshot 中的撤销结论；未知 ID 仍 fail closed。
func (snapshot *AccessSnapshot) Revoked(revocationID string) bool {
	if snapshot == nil {
		return true
	}
	record, ok := snapshot.grants[strings.TrimSpace(revocationID)]
	return !ok || !record.RevokedAt.IsZero()
}

func (store *AccessStore) ensureOpenLocked() error {
	if store.closed || store.owner == nil {
		return fmt.Errorf("client access store is closed")
	}
	return nil
}

func (store *AccessStore) persistLocked() error {
	state := storedAccessState{
		Version: accessStoreVersion, IssuerDeviceID: store.identity.DeviceID, IssuerFingerprint: store.identity.Fingerprint,
		Tickets: store.tickets, Grants: store.grants, AccessProjectionRevision: store.accessProjectionRevision,
	}
	signingBytes, err := accessStateSigningBytes(state)
	if err != nil {
		return err
	}
	state.StateSignature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(store.identity.PrivateKey, signingBytes))
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode client access store: %w", err)
	}
	if err := store.writeFile(store.path, append(payload, '\n')); err != nil {
		return fmt.Errorf("persist client access store: %w", err)
	}
	return nil
}

func (store *AccessStore) notifyAccessChangedLocked() {
	select {
	case store.changes <- struct{}{}:
	default:
	}
}

func loadStoredAccessState(path string, identity Identity) (storedAccessState, bool, error) {
	payload, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return storedAccessState{}, false, nil
	}
	if err != nil {
		return storedAccessState{}, false, fmt.Errorf("read client access store: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var state storedAccessState
	if err := decoder.Decode(&state); err != nil {
		return storedAccessState{}, false, fmt.Errorf("decode client access store: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return storedAccessState{}, false, fmt.Errorf("decode client access store: trailing data")
	}
	if state.Version != accessStoreVersion || state.IssuerDeviceID != identity.DeviceID || state.IssuerFingerprint != identity.Fingerprint {
		return storedAccessState{}, false, fmt.Errorf("client access store version or issuer identity is invalid")
	}
	signature, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(state.StateSignature))
	if err != nil || len(signature) != ed25519.SignatureSize {
		return storedAccessState{}, false, fmt.Errorf("client access store signature is invalid")
	}
	signingBytes, err := accessStateSigningBytes(state)
	if err != nil || !ed25519.Verify(identity.PublicKey, signingBytes, signature) {
		return storedAccessState{}, false, fmt.Errorf("client access store signature does not match daemon identity")
	}
	if state.Tickets == nil {
		state.Tickets = map[string]storedPairingTicket{}
	}
	if state.Grants == nil {
		state.Grants = map[string]storedAccessGrant{}
	}
	return state, true, nil
}

func accessStateSigningBytes(state storedAccessState) ([]byte, error) {
	return accessStateSigningBytesWithDomain(state, accessStateSignatureDomain)
}

func accessStateSigningBytesWithDomain(state storedAccessState, domain string) ([]byte, error) {
	state.StateSignature = ""
	payload, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("encode client access state signature input: %w", err)
	}
	return append([]byte(domain+"\x00"), payload...), nil
}

func (store *AccessStore) validateLoadedState() error {
	if store.accessProjectionRevision == 0 && len(store.grants) != 0 {
		return fmt.Errorf("client access store projection revision is invalid")
	}
	for ticketID, record := range store.tickets {
		claims := normalizePairingTicketClaims(record.Claims)
		if ticketID != claims.TicketID || claims.IssuerDeviceID != store.identity.DeviceID || claims.IssuerDeviceFingerprint != store.identity.Fingerprint ||
			validatePairingTicketClaims(claims) != nil || !validPayloadDigest(record.TicketDigest) {
			return fmt.Errorf("client access store ticket %q is invalid", ticketID)
		}
		if record.GrantID == "" {
			if record.SubjectKeyFingerprint != "" || record.ClientLabel != "" || record.ResultGrantDigest != "" ||
				record.DeliveryReceiptDigest != "" || record.CloudRouteGrantDigest != "" || record.CloudEdgeLocatorDigest != "" || !record.RedeemedAt.IsZero() || !record.DeliveryGraceUntil.IsZero() {
				return fmt.Errorf("client access store ticket %q has partial redemption", ticketID)
			}
			continue
		}
		grant, ok := store.grants[record.GrantID]
		if !ok || grant.Claims.GrantID != record.GrantID || grant.Claims.SubjectKeyFingerprint != record.SubjectKeyFingerprint ||
			grant.ClientLabel != record.ClientLabel || !record.RedeemedAt.Equal(grant.Claims.IssuedAt) ||
			!record.DeliveryGraceUntil.After(record.RedeemedAt) || !validPayloadDigest(record.ResultGrantDigest) || !validPayloadDigest(record.DeliveryReceiptDigest) {
			return fmt.Errorf("client access store ticket %q grant link is invalid", ticketID)
		}
		hasManagedRouteMaterial := len(grant.CloudRouteGrant) != 0 || len(grant.CloudEdgeLocator) != 0 || record.CloudRouteGrantDigest != "" || record.CloudEdgeLocatorDigest != ""
		if hasManagedRouteMaterial && (len(grant.CloudRouteGrant) == 0 || len(grant.CloudEdgeLocator) == 0 || !validPayloadDigest(record.CloudRouteGrantDigest) || !validPayloadDigest(record.CloudEdgeLocatorDigest)) {
			return fmt.Errorf("client access store ticket %q managed route material is incomplete", ticketID)
		}
		result, err := store.pairingResultFromStored(record)
		if err != nil || result.Scope != claims.ScopeCeiling {
			return fmt.Errorf("client access store ticket %q result is invalid", ticketID)
		}
	}
	for grantID, record := range store.grants {
		claims := normalizeClaims(record.Claims)
		if grantID != claims.GrantID || claims.RevocationID != grantID || claims.IssuerDeviceID != store.identity.DeviceID ||
			claims.IssuerDeviceFingerprint != store.identity.Fingerprint || validateClaims(claims) != nil {
			return fmt.Errorf("client access store grant %q is invalid", grantID)
		}
		grant, err := Issue(store.identity.PrivateKey, claims)
		if err != nil {
			return fmt.Errorf("client access store grant %q cannot be reconstructed", grantID)
		}
		parsed, err := verifyGrantEnvelope(grant, store.identity.Fingerprint)
		if err != nil || parsed != claims {
			return fmt.Errorf("client access store grant %q reconstruction is invalid", grantID)
		}
	}
	return nil
}

func (store *AccessStore) pairingResultFromStored(ticket storedPairingTicket) (PairingExchangeResult, error) {
	grant, ok := store.grants[ticket.GrantID]
	if !ok {
		return PairingExchangeResult{}, fmt.Errorf("stored pairing grant is missing")
	}
	boundGrant, err := Issue(store.identity.PrivateKey, grant.Claims)
	if err != nil {
		return PairingExchangeResult{}, fmt.Errorf("stored pairing grant cannot be reconstructed")
	}
	if subtle.ConstantTimeCompare([]byte(payloadDigest([]byte(boundGrant))), []byte(ticket.ResultGrantDigest)) != 1 {
		return PairingExchangeResult{}, fmt.Errorf("stored pairing grant digest mismatch")
	}
	receipt := pairingDeliveryReceipt(store.identity.PrivateKey, ticket.Claims.TicketID, ticket.SubjectKeyFingerprint, ticket.GrantID)
	if subtle.ConstantTimeCompare([]byte(payloadDigest([]byte(receipt))), []byte(ticket.DeliveryReceiptDigest)) != 1 {
		return PairingExchangeResult{}, fmt.Errorf("stored pairing receipt digest mismatch")
	}
	if ticket.CloudRouteGrantDigest != "" && subtle.ConstantTimeCompare([]byte(payloadDigest(grant.CloudRouteGrant)), []byte(ticket.CloudRouteGrantDigest)) != 1 {
		return PairingExchangeResult{}, fmt.Errorf("stored managed Route grant digest mismatch")
	}
	if ticket.CloudEdgeLocatorDigest != "" && subtle.ConstantTimeCompare([]byte(payloadDigest(grant.CloudEdgeLocator)), []byte(ticket.CloudEdgeLocatorDigest)) != 1 {
		return PairingExchangeResult{}, fmt.Errorf("stored managed Edge locator digest mismatch")
	}
	return PairingExchangeResult{
		TicketID: ticket.Claims.TicketID, Grant: boundGrant, GrantID: ticket.GrantID, DeliveryReceipt: receipt,
		SubjectKeyFingerprint: ticket.SubjectKeyFingerprint, Scope: grant.Claims.Scope, ExpiresAt: grant.Claims.ExpiresAt.UTC(), CloudRouteGrant: append([]byte(nil), grant.CloudRouteGrant...), CloudEdgeLocator: append([]byte(nil), grant.CloudEdgeLocator...),
	}, nil
}

func (store *AccessStore) rebuildClientGrantIndexLocked() {
	store.clientGrants = make(map[string]map[string]struct{})
	for grantID, record := range store.grants {
		store.indexGrantLocked(record.Claims.SubjectKeyFingerprint, grantID)
	}
}

func (store *AccessStore) indexGrantLocked(fingerprint, grantID string) {
	if store.clientGrants[fingerprint] == nil {
		store.clientGrants[fingerprint] = make(map[string]struct{})
	}
	store.clientGrants[fingerprint][grantID] = struct{}{}
}

func (store *AccessStore) compactLocked(now time.Time) bool {
	changed := false
	for ticketID, record := range store.tickets {
		if record.GrantID == "" {
			if !now.Before(record.Claims.ExpiresAt.Add(expiredTicketRetention)) {
				delete(store.tickets, ticketID)
			}
			continue
		}
		if !record.DeliveryGraceUntil.IsZero() && !now.Before(record.DeliveryGraceUntil) {
			delete(store.tickets, ticketID)
		}
	}
	for grantID, record := range store.grants {
		if !now.Before(record.Claims.ExpiresAt.Add(expiredGrantRecordRetention)) {
			delete(store.grants, grantID)
			changed = true
		}
	}
	if changed {
		store.rebuildClientGrantIndexLocked()
	}
	return changed
}

func pairingBundleHasManagedRoute(bundle *PairingBundle) bool {
	if bundle == nil {
		return false
	}
	for _, route := range bundle.GetRoutes() {
		if route.GetManagedWebrtc() != nil {
			return true
		}
	}
	return false
}

func pairingDeliveryReceipt(privateKey ed25519.PrivateKey, ticketID, subjectFingerprint, grantID string) string {
	return pairingDeliveryReceiptWithFormat(privateKey, pairingReceiptDomain, "anytty-pairing-receipt-v1", ticketID, subjectFingerprint, grantID)
}

func pairingDeliveryReceiptWithFormat(privateKey ed25519.PrivateKey, domain, prefix, ticketID, subjectFingerprint, grantID string) string {
	digest := sha256.Sum256([]byte(domain + "\x00" + ticketID + "\x00" + subjectFingerprint + "\x00" + grantID))
	signature := ed25519.Sign(privateKey, digest[:])
	return prefix + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func payloadDigest(payload []byte) string {
	digest := sha256.Sum256(payload)
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func validPayloadDigest(value string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
	return err == nil && len(decoded) == sha256.Size
}

func cloneTicketRecords(source map[string]storedPairingTicket) map[string]storedPairingTicket {
	cloned := make(map[string]storedPairingTicket, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneGrantRecords(source map[string]storedAccessGrant) map[string]storedAccessGrant {
	cloned := make(map[string]storedAccessGrant, len(source))
	for key, value := range source {
		value.CloudRouteGrant = append([]byte(nil), value.CloudRouteGrant...)
		value.CloudEdgeLocator = append([]byte(nil), value.CloudEdgeLocator...)
		cloned[key] = value
	}
	return cloned
}

func clientAccessRecordFromStored(record storedAccessGrant) ClientAccessRecord {
	return ClientAccessRecord{
		GrantID: record.Claims.GrantID, RevocationID: record.Claims.RevocationID, SubjectKeyFingerprint: record.Claims.SubjectKeyFingerprint,
		ClientLabel: record.ClientLabel, Scope: record.Claims.Scope, IssuedAt: record.Claims.IssuedAt.UTC(),
		ExpiresAt: record.Claims.ExpiresAt.UTC(), RevokedAt: record.RevokedAt.UTC(),
	}
}

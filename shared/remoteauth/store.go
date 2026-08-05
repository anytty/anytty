package remoteauth

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/shared/filelock"
	"github.com/anytty/anytty/shared/securefs"
)

const clientCredentialVersion = 3

var grantRefPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

var (
	// ErrGrantScopeExpansion 表示新 grant 扩大了同一 Endpoint 已保存的能力边界，必须由用户显式确认。
	ErrGrantScopeExpansion = errors.New("remote capability grant expands existing scope")
)

// ClientAccessCredential 是客户端 secure store 中一个 Endpoint 的完整授权真值。
// Identity private key 与 CapabilityGrant 必须原子保存在 owner-only 文件；普通 endpoint registry 只能保存 ref，不能保存本结构任何 secret 字段。
type ClientAccessCredential struct {
	Version                uint32
	EndpointID             string
	Identity               ClientAccessIdentity
	CapabilityGrant        string
	CloudRouteGrant        []byte
	CloudEdgeLocator       []byte
	LastPairingClaimDigest string
	UpdatedAt              time.Time
}

// Ready 返回 credential 是否已经取得与当前 ClientAccessIdentity 绑定的非空 grant。
// false 表示 pairing 尚未完成或响应尚未恢复，调用方不能尝试 bearer-only 连接。
func (credential ClientAccessCredential) Ready() bool {
	return strings.TrimSpace(credential.CapabilityGrant) != ""
}

// CredentialStore 是桌面客户端 per-Endpoint ClientAccessIdentity 与 bound grant 的文件型 secure store。
// 它不拥有 endpoint registry 或 daemon revocation truth；每个 ref 对应一个 0600 JSON 文件，写入使用同目录原子 rename。
type CredentialStore struct {
	dir string
	mu  sync.Mutex
}

// BindGrantOptions 控制把新 grant 写入既有 Endpoint credential 的高风险策略。
// AllowScopeExpansion 只能来自明确用户确认；二维码、Cloud discovery 或静默重试不得自行设置。
type BindGrantOptions struct {
	AllowScopeExpansion bool
}

type storedClientAccessCredential struct {
	Version                uint32    `json:"version"`
	EndpointID             string    `json:"endpoint_id"`
	PrivateKey             string    `json:"private_key"`
	CapabilityGrant        string    `json:"capability_grant,omitempty"`
	CloudRouteGrant        []byte    `json:"cloud_route_grant,omitempty"`
	CloudEdgeLocator       []byte    `json:"cloud_edge_locator,omitempty"`
	LastPairingClaimDigest string    `json:"last_pairing_claim_digest,omitempty"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// NewCredentialStore 创建桌面客户端 secure credential store。
// dir 必须位于当前用户安全域，不能与 daemon DeviceIdentity、AccessStore、Cloud session 或普通 endpoints.yaml 混用。
func NewCredentialStore(dir string) *CredentialStore {
	return &CredentialStore{dir: strings.TrimSpace(dir)}
}

// LoadOrCreateIdentity 加载 ref 已绑定的 ClientAccessIdentity，或在首次 pairing 前生成并立即持久化同一 Endpoint 的 key。
// 先持久化 key 是响应丢失幂等恢复的前提；一旦 ref 存在，endpoint_id 不匹配必须 fail closed，不能静默换 key。
func (store *CredentialStore) LoadOrCreateIdentity(ref string, endpointID string, random io.Reader) (ClientAccessCredential, error) {
	if err := store.validate(ref); err != nil {
		return ClientAccessCredential{}, err
	}
	endpointID = strings.TrimSpace(endpointID)
	if endpointID == "" {
		return ClientAccessCredential{}, fmt.Errorf("client access credential requires endpoint_id")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	lock, err := store.acquireRefLockLocked(context.Background(), ref)
	if err != nil {
		return ClientAccessCredential{}, err
	}
	defer lock.Close()
	return store.loadOrCreateIdentityLocked(ref, endpointID, random)
}

// PairAndBind 在同一 credential ref 的跨进程锁内完成 scope 预检、PairingExchange 回调、grant 验签与原子落盘。
// 相同 claim 已成功绑定时仍用同一 client key 执行 exchange，由 daemon 幂等返回完整响应，供 registry 写失败后的恢复使用。
// exchange 只接收当前 Endpoint 的 ClientAccessIdentity，必须通过 daemon-local PairingExchange 返回 client-bound grant，不能访问 registry 或写入其他 ref。
func (store *CredentialStore) PairAndBind(
	ctx context.Context,
	ref string,
	endpointID string,
	pairingClaimOffer []byte,
	now func() time.Time,
	random io.Reader,
	options BindGrantOptions,
	exchange func(ClientAccessIdentity) (PairingExchangeResult, error),
) (ClientAccessCredential, error) {
	if err := store.validate(ref); err != nil {
		return ClientAccessCredential{}, err
	}
	offer, err := ParsePairingClaimOfferForExchange(pairingClaimOffer)
	if err != nil {
		return ClientAccessCredential{}, err
	}
	expectedDeviceID := offer.GetDeviceId()
	expectedFingerprint := Fingerprint(ed25519.PublicKey(offer.GetDevicePublicKey()))
	if exchange == nil {
		return ClientAccessCredential{}, fmt.Errorf("client access pairing requires exchange callback")
	}
	endpointID = strings.TrimSpace(endpointID)
	if endpointID == "" {
		return ClientAccessCredential{}, fmt.Errorf("client access credential requires endpoint_id")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if now == nil {
		now = time.Now
	}
	claimDigest := payloadDigest(pairingClaimOffer)
	store.mu.Lock()
	defer store.mu.Unlock()
	lock, err := store.acquireRefLockLocked(ctx, ref)
	if err != nil {
		return ClientAccessCredential{}, err
	}
	defer lock.Close()
	credential, err := store.loadOrCreateIdentityLocked(ref, endpointID, random)
	if err != nil {
		return ClientAccessCredential{}, err
	}
	if err := ctx.Err(); err != nil {
		return ClientAccessCredential{}, err
	}
	if credential.Ready() {
		currentClaims, currentErr := verifyGrantEnvelope(credential.CapabilityGrant, expectedFingerprint)
		if currentErr != nil {
			return ClientAccessCredential{}, fmt.Errorf("verify existing client access grant: %w", currentErr)
		}
		if currentClaims.SubjectKeyFingerprint != credential.Identity.Fingerprint {
			return ClientAccessCredential{}, ErrGrantSubjectMismatch
		}
		if credential.LastPairingClaimDigest == claimDigest {
			if _, verifyErr := Verify(credential.CapabilityGrant, expectedFingerprint, now().UTC(), nil); verifyErr != nil {
				return ClientAccessCredential{}, verifyErr
			}
		}
	}
	result, err := exchange(credential.Identity)
	if err != nil {
		return ClientAccessCredential{}, err
	}
	responseNow := now().UTC()
	responseBundle, ticketClaims, err := ParsePairingBundleForExchange(result.Bundle)
	if err != nil || responseBundle.GetIdentity().GetDeviceId() != expectedDeviceID || responseBundle.GetIdentity().GetDeviceFingerprint() != expectedFingerprint {
		return ClientAccessCredential{}, fmt.Errorf("pairing exchange returned an invalid bundle: %w", err)
	}
	claims, err := Verify(result.Grant, expectedFingerprint, responseNow, nil)
	if err != nil {
		return ClientAccessCredential{}, err
	}
	if claims.IssuerDeviceID != expectedDeviceID {
		return ClientAccessCredential{}, ErrGrantFingerprintMismatch
	}
	if claims.SubjectKeyFingerprint != credential.Identity.Fingerprint || result.SubjectKeyFingerprint != credential.Identity.Fingerprint {
		return ClientAccessCredential{}, ErrGrantSubjectMismatch
	}
	if !scopeContains(ticketClaims.ScopeCeiling, claims.Scope) {
		return ClientAccessCredential{}, fmt.Errorf("%w: pairing exchange grant exceeds ticket scope", ErrGrantScopeInvalid)
	}
	if credential.Ready() {
		currentClaims, currentErr := verifyGrantEnvelope(credential.CapabilityGrant, expectedFingerprint)
		if currentErr != nil {
			return ClientAccessCredential{}, fmt.Errorf("verify existing client access grant: %w", currentErr)
		}
		if !scopeContains(currentClaims.Scope, claims.Scope) && !options.AllowScopeExpansion {
			return ClientAccessCredential{}, ErrGrantScopeExpansion
		}
	}
	credential.CapabilityGrant = strings.TrimSpace(result.Grant)
	credential.CloudRouteGrant = append([]byte(nil), result.CloudRouteGrant...)
	credential.CloudEdgeLocator = append([]byte(nil), result.CloudEdgeLocator...)
	credential.LastPairingClaimDigest = claimDigest
	credential.UpdatedAt = responseNow
	if err := store.persistLocked(ref, credential); err != nil {
		return ClientAccessCredential{}, err
	}
	return credential, nil
}

// BindGrant 验证 daemon issuer、ClientAccessIdentity subject 与 grant v2 后，把 grant 原子绑定到既有 ref。
// 该操作不会更换 private key；校验或写入失败时旧 credential 保持不变，调用方可用同一 key 重试 PairingExchange。
func (store *CredentialStore) BindGrant(ref string, grant string, expectedDaemonFingerprint string, now time.Time, options BindGrantOptions) (ClientAccessCredential, error) {
	if err := store.validate(ref); err != nil {
		return ClientAccessCredential{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	lock, err := store.acquireRefLockLocked(context.Background(), ref)
	if err != nil {
		return ClientAccessCredential{}, err
	}
	defer lock.Close()
	credential, err := store.resolveLocked(ref)
	if err != nil {
		return ClientAccessCredential{}, err
	}
	claims, err := Verify(grant, expectedDaemonFingerprint, now, nil)
	if err != nil {
		return ClientAccessCredential{}, err
	}
	if claims.SubjectKeyFingerprint != credential.Identity.Fingerprint {
		return ClientAccessCredential{}, ErrGrantSubjectMismatch
	}
	if credential.Ready() {
		currentClaims, currentErr := verifyGrantEnvelope(credential.CapabilityGrant, expectedDaemonFingerprint)
		if currentErr != nil {
			return ClientAccessCredential{}, fmt.Errorf("verify existing client access grant: %w", currentErr)
		}
		if !scopeContains(currentClaims.Scope, claims.Scope) && !options.AllowScopeExpansion {
			return ClientAccessCredential{}, ErrGrantScopeExpansion
		}
		if strings.TrimSpace(credential.CapabilityGrant) == strings.TrimSpace(grant) {
			return credential, nil
		}
	}
	credential.CapabilityGrant = strings.TrimSpace(grant)
	credential.UpdatedAt = time.Now().UTC()
	if err := store.persistLocked(ref, credential); err != nil {
		return ClientAccessCredential{}, err
	}
	return credential, nil
}

// Resolve 读取一个已经完成 pairing 的 ClientAccessCredential。
// 缺失、损坏、endpoint key 不一致或 grant 为空都必须失败；调用方不得回退到旧 bearer token、registry 字段或其他 Endpoint 的 key。
func (store *CredentialStore) Resolve(ref string) (ClientAccessCredential, error) {
	return store.ResolveContext(context.Background(), ref)
}

// ResolveContext 等价于 Resolve，但等待同一 credential ref 的跨进程锁时响应调用方取消。
// 带根 deadline 的 CLI/runtime 入口应传递自己的 context，避免 pairing 事务阻塞普通连接退出。
func (store *CredentialStore) ResolveContext(ctx context.Context, ref string) (ClientAccessCredential, error) {
	if err := store.validate(ref); err != nil {
		return ClientAccessCredential{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	lock, err := store.acquireRefLockLocked(ctx, ref)
	if err != nil {
		return ClientAccessCredential{}, err
	}
	defer lock.Close()
	credential, err := store.resolveLocked(ref)
	if err != nil {
		return ClientAccessCredential{}, err
	}
	if !credential.Ready() {
		return ClientAccessCredential{}, fmt.Errorf("client access credential %q is awaiting pairing", ref)
	}
	return credential, nil
}

// UpdateCloudEdgeLocator 原子替换同一 credential 的公开 Edge locator；grant、identity 和权限保持不变。
func (store *CredentialStore) UpdateCloudEdgeLocator(ctx context.Context, ref, endpointID string, locator []byte) error {
	if err := store.validate(ref); err != nil {
		return err
	}
	if strings.TrimSpace(endpointID) == "" || len(locator) == 0 {
		return errors.New("Cloud Edge locator update is incomplete")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	lock, err := store.acquireRefLockLocked(ctx, ref)
	if err != nil {
		return err
	}
	defer lock.Close()
	credential, err := store.resolveLocked(ref)
	if err != nil {
		return err
	}
	if credential.EndpointID != strings.TrimSpace(endpointID) || !credential.Ready() || len(credential.CloudRouteGrant) == 0 {
		return errors.New("Cloud Edge locator credential binding is invalid")
	}
	credential.CloudEdgeLocator = append([]byte(nil), locator...)
	credential.UpdatedAt = time.Now().UTC()
	return store.persistLocked(ref, credential)
}

// Delete 删除本地 credential ref，且不修改 daemon AccessStore 或撤销远端 grant。
// 该操作只用于显式忘记 Endpoint；删除已消费 ticket 对应的唯一 private key 会使该 grant 永久不可用。
func (store *CredentialStore) Delete(ref string) error {
	if err := store.validate(ref); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	lock, err := store.acquireRefLockLocked(context.Background(), ref)
	if err != nil {
		return err
	}
	defer lock.Close()
	err = os.Remove(filepath.Join(store.dir, credentialFileName(ref)))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete client access credential %q: %w", ref, err)
	}
	return nil
}

func (store *CredentialStore) validate(ref string) error {
	if store == nil || strings.TrimSpace(store.dir) == "" {
		return fmt.Errorf("remote credential store directory is not configured")
	}
	return validateGrantRef(ref)
}

func (store *CredentialStore) resolveLocked(ref string) (ClientAccessCredential, error) {
	path := filepath.Join(store.dir, credentialFileName(ref))
	payload, err := os.ReadFile(path)
	if err != nil {
		return ClientAccessCredential{}, fmt.Errorf("resolve client access credential %q: %w", ref, err)
	}
	if err := securefs.SecureFile(path); err != nil {
		return ClientAccessCredential{}, fmt.Errorf("secure client access credential %q: %w", ref, err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	var stored storedClientAccessCredential
	if err := decoder.Decode(&stored); err != nil {
		return ClientAccessCredential{}, fmt.Errorf("decode client access credential %q: %w", ref, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ClientAccessCredential{}, fmt.Errorf("decode client access credential %q: trailing data", ref)
	}
	if stored.Version != clientCredentialVersion || strings.TrimSpace(stored.EndpointID) == "" || stored.UpdatedAt.IsZero() {
		return ClientAccessCredential{}, fmt.Errorf("client access credential %q is incomplete or unsupported", ref)
	}
	if stored.LastPairingClaimDigest != "" && !validPayloadDigest(strings.TrimSpace(stored.LastPairingClaimDigest)) {
		return ClientAccessCredential{}, fmt.Errorf("client access credential %q has invalid pairing claim digest", ref)
	}
	privateKey, err := base64.RawURLEncoding.DecodeString(stored.PrivateKey)
	if err != nil || len(privateKey) != ed25519.PrivateKeySize {
		return ClientAccessCredential{}, fmt.Errorf("client access credential %q has invalid private key", ref)
	}
	identity, err := NewClientAccessIdentity(stored.EndpointID, ed25519.PrivateKey(privateKey))
	if err != nil {
		return ClientAccessCredential{}, fmt.Errorf("client access credential %q: %w", ref, err)
	}
	return ClientAccessCredential{
		Version: stored.Version, EndpointID: stored.EndpointID, Identity: identity,
		CapabilityGrant: strings.TrimSpace(stored.CapabilityGrant), LastPairingClaimDigest: strings.TrimSpace(stored.LastPairingClaimDigest),
		CloudRouteGrant:  append([]byte(nil), stored.CloudRouteGrant...),
		CloudEdgeLocator: append([]byte(nil), stored.CloudEdgeLocator...),
		UpdatedAt:        stored.UpdatedAt.UTC(),
	}, nil
}

func (store *CredentialStore) loadOrCreateIdentityLocked(ref string, endpointID string, random io.Reader) (ClientAccessCredential, error) {
	credential, err := store.resolveLocked(ref)
	if err == nil {
		if credential.EndpointID != endpointID {
			return ClientAccessCredential{}, fmt.Errorf("client access credential %q belongs to endpoint %q", ref, credential.EndpointID)
		}
		return credential, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return ClientAccessCredential{}, err
	}
	identity, err := GenerateClientAccessIdentity(endpointID, random)
	if err != nil {
		return ClientAccessCredential{}, err
	}
	credential = ClientAccessCredential{Version: clientCredentialVersion, EndpointID: endpointID, Identity: identity, UpdatedAt: time.Now().UTC()}
	if err := store.persistLocked(ref, credential); err != nil {
		return ClientAccessCredential{}, err
	}
	return credential, nil
}

func (store *CredentialStore) persistLocked(ref string, credential ClientAccessCredential) error {
	if err := credential.Identity.Validate(); err != nil {
		return err
	}
	if credential.Version != clientCredentialVersion || credential.EndpointID != credential.Identity.EndpointID {
		return fmt.Errorf("client access credential identity binding is invalid")
	}
	if err := os.MkdirAll(store.dir, 0o700); err != nil {
		return fmt.Errorf("create remote credential directory: %w", err)
	}
	if err := securefs.SecureDirectory(store.dir); err != nil {
		return fmt.Errorf("secure remote credential directory: %w", err)
	}
	stored := storedClientAccessCredential{
		Version: credential.Version, EndpointID: credential.EndpointID,
		PrivateKey:             base64.RawURLEncoding.EncodeToString(credential.Identity.PrivateKey),
		CapabilityGrant:        strings.TrimSpace(credential.CapabilityGrant),
		CloudRouteGrant:        append([]byte(nil), credential.CloudRouteGrant...),
		CloudEdgeLocator:       append([]byte(nil), credential.CloudEdgeLocator...),
		LastPairingClaimDigest: strings.TrimSpace(credential.LastPairingClaimDigest), UpdatedAt: credential.UpdatedAt.UTC(),
	}
	payload, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("encode client access credential: %w", err)
	}
	if err := writePrivateFile(filepath.Join(store.dir, credentialFileName(ref)), append(payload, '\n')); err != nil {
		return fmt.Errorf("persist client access credential %q: %w", ref, err)
	}
	return nil
}

func (store *CredentialStore) acquireRefLockLocked(ctx context.Context, ref string) (*filelock.Lock, error) {
	if err := os.MkdirAll(store.dir, 0o700); err != nil {
		return nil, fmt.Errorf("create remote credential directory: %w", err)
	}
	if err := securefs.SecureDirectory(store.dir); err != nil {
		return nil, fmt.Errorf("secure remote credential directory: %w", err)
	}
	lock, err := filelock.AcquireContext(ctx, filepath.Join(store.dir, credentialFileName(ref)+".lock"), false)
	if err != nil {
		return nil, fmt.Errorf("lock client access credential %q: %w", ref, err)
	}
	return lock, nil
}

func scopeContains(current Scope, candidate Scope) bool {
	baseContains := false
	switch {
	case current.AllowDaemon:
		baseContains = candidate.AllowDaemon || candidate.TerminalID != "" || candidate.MachineEventsOnly
	case current.TerminalID != "":
		baseContains = candidate.TerminalID == current.TerminalID
	case current.MachineEventsOnly:
		baseContains = candidate.MachineEventsOnly
	}
	return baseContains &&
		(!candidate.FileReadMetadata || current.FileReadMetadata) &&
		(!candidate.FileReadContent || current.FileReadContent) &&
		(!candidate.FileWriteContent || current.FileWriteContent) &&
		(!candidate.FileMutate || current.FileMutate) &&
		(!candidate.ManageClientAccess || current.ManageClientAccess)
}

func validateGrantRef(ref string) error {
	ref = strings.TrimSpace(ref)
	if !grantRefPattern.MatchString(ref) {
		return fmt.Errorf("invalid remote grant_ref %q", ref)
	}
	return nil
}

func credentialFileName(ref string) string {
	return "access_" + base64.RawURLEncoding.EncodeToString([]byte(ref)) + ".json"
}

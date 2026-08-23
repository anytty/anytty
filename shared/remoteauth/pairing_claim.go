package remoteauth

import (
	"bytes"
	"compress/flate"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"google.golang.org/protobuf/proto"
)

const (
	// PairingClaimOfferVersion 是二维码和手工输入使用的紧凑 claim schema 版本。
	PairingClaimOfferVersion    uint32 = 2
	pairingClaimBytes                  = 16
	maxPairingClaimOfferBytes          = 4 * 1024
	maxPairingClaimRoutes              = 4
	pairingClaimEnvelopeRaw     byte   = 0
	pairingClaimEnvelopeDeflate byte   = 1
	// PairingClaimCodePrefix 标识可以直接粘贴到无摄像头客户端的 portable claim code。
	PairingClaimCodePrefix = "MXP2-"
)

var (
	// ErrPairingClaimMalformed 表示 claim offer 不是 canonical protobuf，或缺少身份、路由和 128-bit claim。
	ErrPairingClaimMalformed = errors.New("remote pairing claim malformed")
	// ErrPairingClaimUnavailable 表示 claim 不属于当前 daemon 进程，常见于错误 daemon 或 daemon 重启后。
	ErrPairingClaimUnavailable = errors.New("remote pairing claim unavailable")
)

// PairingClaimIssueResult 是 daemon 签发短码时返回给 API Layer 的完整本地结果。
// OfferPayload 可以进入二维码；BundlePayload 只留在 daemon 内存，直到端到端 PairingAccepted 返回给目标客户端。
type PairingClaimIssueResult struct {
	Offer         *remoteauthpb.PairingClaimOffer
	OfferPayload  []byte
	ClaimCode     string
	BundlePayload []byte
	Claims        PairingTicketClaims
}

type storedPairingClaim struct {
	BundlePayload         []byte
	ExpiresAt             time.Time
	SubjectKeyFingerprint string
}

// IssuePairingClaim 原子签发持久 PairingTicket，并仅在当前 daemon 内存登记对应 128-bit claim。
// daemon 重启只使尚未兑换的 claim 失效，不改变已经持久化的 grant、撤销或 ticket 消费真值。
func (store *AccessStore) IssuePairingClaim(options PairingIssueOptions) (PairingClaimIssueResult, error) {
	if store == nil {
		return PairingClaimIssueResult{}, fmt.Errorf("client access store is nil")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.ensureOpenLocked(); err != nil {
		return PairingClaimIssueResult{}, err
	}
	if len(options.Routes) == 0 {
		return PairingClaimIssueResult{}, fmt.Errorf("%w: claim requires at least one Direct, SSH, or Cloud pairing Route", ErrPairingClaimMalformed)
	}
	if options.Now.IsZero() {
		options.Now = store.now().UTC()
	}
	if options.Random == nil {
		options.Random = store.random
	}
	claim, digest, err := store.reservePairingClaim(options.Random)
	if err != nil {
		return PairingClaimIssueResult{}, err
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		store.pairingClaimsMu.Lock()
		delete(store.pairingClaims, digest)
		store.pairingClaimsMu.Unlock()
	}()

	bundle, claims, err := issuePairingBundle(store.identity, options)
	if err != nil {
		return PairingClaimIssueResult{}, err
	}
	bundlePayload, err := EncodePairingBundle(bundle)
	if err != nil {
		return PairingClaimIssueResult{}, err
	}
	seeds := pairingClaimRouteSeeds(bundle.GetRoutes())
	if len(seeds) == 0 {
		return PairingClaimIssueResult{}, fmt.Errorf("%w: claim requires at least one Direct, SSH, or Cloud pairing Route", ErrPairingClaimMalformed)
	}
	managedPairingIssuer := store.managedPairingBootstrapIssue
	for _, seed := range seeds {
		managed := seed.GetManagedWebrtc()
		if managed == nil {
			continue
		}
		if managedPairingIssuer == nil {
			return PairingClaimIssueResult{}, errors.New("managed pairing bootstrap issuer is unavailable")
		}
		bootstrap, bootstrapErr := managedPairingIssuer()
		if bootstrapErr != nil {
			return PairingClaimIssueResult{}, fmt.Errorf("issue managed pairing bootstrap: %w", bootstrapErr)
		}
		if bootstrap == nil {
			return PairingClaimIssueResult{}, errors.New("managed pairing bootstrap issuer returned no route material")
		}
		seed.Route = &remoteauthpb.PairingRouteSeed_ManagedWebrtc{ManagedWebrtc: proto.Clone(bootstrap).(*remoteauthpb.PairingManagedRouteSeed)}
	}
	offer := &remoteauthpb.PairingClaimOffer{
		SchemaVersion:     PairingClaimOfferVersion,
		Claim:             append([]byte(nil), claim...),
		DeviceId:          store.identity.DeviceID,
		DevicePublicKey:   append([]byte(nil), store.identity.PublicKey...),
		ExpiresAtUnixNano: claims.ExpiresAt.UnixNano(),
		Routes:            seeds,
	}
	offerPayload, err := EncodePairingClaimOffer(offer)
	if err != nil {
		return PairingClaimIssueResult{}, err
	}
	if err := store.persistPairingBundleLocked(bundlePayload, claims, options.AccessLabel, options.Now.UTC()); err != nil {
		return PairingClaimIssueResult{}, err
	}
	store.pairingClaimsMu.Lock()
	store.pairingClaims[digest] = storedPairingClaim{BundlePayload: append([]byte(nil), bundlePayload...), ExpiresAt: claims.ExpiresAt}
	store.pairingClaimsMu.Unlock()
	committed = true
	return PairingClaimIssueResult{
		Offer: offer, OfferPayload: offerPayload, ClaimCode: EncodePairingClaimCode(offerPayload),
		BundlePayload: bundlePayload, Claims: claims,
	}, nil
}

func (store *AccessStore) reservePairingClaim(randomSource io.Reader) ([]byte, [sha256.Size]byte, error) {
	if randomSource == nil {
		return nil, [sha256.Size]byte{}, fmt.Errorf("generate pairing claim: random source is nil")
	}
	for attempt := 0; attempt < 8; attempt++ {
		claim := make([]byte, pairingClaimBytes)
		if _, err := io.ReadFull(randomSource, claim); err != nil {
			return nil, [sha256.Size]byte{}, fmt.Errorf("generate pairing claim: %w", err)
		}
		digest := sha256.Sum256(claim)
		store.pairingClaimsMu.Lock()
		if store.pairingClaims == nil {
			store.pairingClaims = make(map[[sha256.Size]byte]storedPairingClaim)
		}
		if _, exists := store.pairingClaims[digest]; !exists {
			store.pairingClaims[digest] = storedPairingClaim{}
			store.pairingClaimsMu.Unlock()
			return claim, digest, nil
		}
		store.pairingClaimsMu.Unlock()
	}
	return nil, [sha256.Size]byte{}, fmt.Errorf("generate pairing claim: collision limit reached")
}

// ResolvePairingClaimForExchange 校验 offer 归属和客户端绑定，并返回 daemon 内存中的完整签名 bundle。
// 过期后仅允许已经成功绑定的同一客户端在 delivery grace 内恢复丢失响应。
func (store *AccessStore) ResolvePairingClaimForExchange(payload []byte, clientPublicKey ed25519.PublicKey, now time.Time) ([]byte, error) {
	if store == nil || len(clientPublicKey) != ed25519.PublicKeySize {
		return nil, ErrPairingClaimMalformed
	}
	offer, err := ParsePairingClaimOfferForExchange(payload)
	if err != nil {
		return nil, err
	}
	if offer.GetDeviceId() != store.identity.DeviceID || !bytes.Equal(offer.GetDevicePublicKey(), store.identity.PublicKey) {
		return nil, ErrPairingClaimUnavailable
	}
	if now.IsZero() {
		now = store.now().UTC()
	}
	digest := sha256.Sum256(offer.GetClaim())
	subject := Fingerprint(clientPublicKey)
	store.pairingClaimsMu.Lock()
	defer store.pairingClaimsMu.Unlock()
	store.compactPairingClaimsLocked(now)
	record, exists := store.pairingClaims[digest]
	if !exists || len(record.BundlePayload) == 0 {
		return nil, ErrPairingClaimUnavailable
	}
	if record.SubjectKeyFingerprint != "" && record.SubjectKeyFingerprint != subject {
		return nil, ErrPairingTicketConsumed
	}
	if now.After(record.ExpiresAt) && record.SubjectKeyFingerprint == "" {
		return nil, ErrPairingTicketExpired
	}
	return append([]byte(nil), record.BundlePayload...), nil
}

// AllowsPairingClaimDigest 是 Cloud signaling 的只读 precheck。
// 它只证明 daemon 当前仍持有该 claim；真正的消费、客户端 proof 和 grant 签发仍由 PairingExchange 完成。
func (store *AccessStore) AllowsPairingClaimDigest(digest []byte, clientPublicKey ed25519.PublicKey, now time.Time) bool {
	if store == nil || len(digest) != sha256.Size || len(clientPublicKey) != ed25519.PublicKeySize || !store.Available() {
		return false
	}
	if now.IsZero() {
		now = store.now().UTC()
	} else {
		now = now.UTC()
	}
	var key [sha256.Size]byte
	copy(key[:], digest)
	subject := Fingerprint(clientPublicKey)
	store.pairingClaimsMu.Lock()
	defer store.pairingClaimsMu.Unlock()
	store.compactPairingClaimsLocked(now)
	record, ok := store.pairingClaims[key]
	if !ok || len(record.BundlePayload) == 0 || record.ExpiresAt.IsZero() {
		return false
	}
	if record.SubjectKeyFingerprint != "" {
		return record.SubjectKeyFingerprint == subject && now.Before(record.ExpiresAt.Add(defaultDeliveryGrace))
	}
	return !now.After(record.ExpiresAt)
}

// PairingClaimActive 是 Direct signaling 的显式 pairing preauth 查询。
// digest、公钥和 offer 绝对期限必须同时匹配；真正兑换仍由 DataChannel 内的 PairingOpen 完成。
func (store *AccessStore) PairingClaimActive(digest []byte, clientPublicKey ed25519.PublicKey, expiresAt, now time.Time) bool {
	if store == nil || len(digest) != sha256.Size || len(clientPublicKey) != ed25519.PublicKeySize || expiresAt.IsZero() || !store.Available() {
		return false
	}
	if now.IsZero() {
		now = store.now().UTC()
	} else {
		now = now.UTC()
	}
	var key [sha256.Size]byte
	copy(key[:], digest)
	subject := Fingerprint(clientPublicKey)
	store.pairingClaimsMu.Lock()
	defer store.pairingClaimsMu.Unlock()
	store.compactPairingClaimsLocked(now)
	record, ok := store.pairingClaims[key]
	if !ok || len(record.BundlePayload) == 0 || !record.ExpiresAt.Equal(expiresAt.UTC()) {
		return false
	}
	if record.SubjectKeyFingerprint == "" {
		return !now.After(record.ExpiresAt)
	}
	return record.SubjectKeyFingerprint == subject && now.Before(record.ExpiresAt.Add(defaultDeliveryGrace))
}

// RedeemPairingClaim 复用 AccessStore 的 PairingTicket 原子消费事务，并在成功后把内存 claim 固定到同一客户端 key。
func (store *AccessStore) RedeemPairingClaim(offerPayload []byte, clientPublicKey ed25519.PublicKey, clientLabel string, now time.Time) (PairingExchangeResult, []byte, error) {
	return store.redeemPairingClaim(offerPayload, clientPublicKey, clientLabel, 0, now)
}

// RedeemPairingClaimForProduct 把官方客户端产品类型交给 daemon Cloud owner 签发 CloudRouteGrant。
func (store *AccessStore) RedeemPairingClaimForProduct(offerPayload []byte, clientPublicKey ed25519.PublicKey, clientLabel string, product uint32, now time.Time) (PairingExchangeResult, []byte, error) {
	return store.redeemPairingClaim(offerPayload, clientPublicKey, clientLabel, product, now)
}

func (store *AccessStore) redeemPairingClaim(offerPayload []byte, clientPublicKey ed25519.PublicKey, clientLabel string, product uint32, now time.Time) (PairingExchangeResult, []byte, error) {
	bundlePayload, err := store.ResolvePairingClaimForExchange(offerPayload, clientPublicKey, now)
	if err != nil {
		return PairingExchangeResult{}, nil, err
	}
	result, err := store.RedeemPairingBundleForProduct(bundlePayload, clientPublicKey, clientLabel, product, now)
	if err != nil {
		return PairingExchangeResult{}, nil, err
	}
	offer, err := ParsePairingClaimOfferForExchange(offerPayload)
	if err != nil {
		return PairingExchangeResult{}, nil, err
	}
	digest := sha256.Sum256(offer.GetClaim())
	store.pairingClaimsMu.Lock()
	record := store.pairingClaims[digest]
	record.SubjectKeyFingerprint = result.SubjectKeyFingerprint
	store.pairingClaims[digest] = record
	store.pairingClaimsMu.Unlock()
	return result, bundlePayload, nil
}

func (store *AccessStore) compactPairingClaimsLocked(now time.Time) {
	for digest, record := range store.pairingClaims {
		if !record.ExpiresAt.IsZero() && now.After(record.ExpiresAt.Add(defaultDeliveryGrace)) {
			delete(store.pairingClaims, digest)
		}
	}
}

// EncodePairingClaimOffer 严格校验并确定性编码二维码 claim；输出不得包含 bundle、grant、scope 或 credential。
func EncodePairingClaimOffer(offer *remoteauthpb.PairingClaimOffer) ([]byte, error) {
	if err := validatePairingClaimOffer(offer, time.Time{}, false); err != nil {
		return nil, err
	}
	payload, err := (proto.MarshalOptions{Deterministic: true}).Marshal(offer)
	if err != nil || len(payload) > maxPairingClaimOfferBytes {
		return nil, fmt.Errorf("%w: encoded offer is invalid", ErrPairingClaimMalformed)
	}
	return payload, nil
}

// ParsePairingClaimOffer 校验 canonical offer 和当前有效期，供二维码导入前建立临时 Endpoint pin 与 pairing Route。
func ParsePairingClaimOffer(payload []byte, now time.Time) (*remoteauthpb.PairingClaimOffer, error) {
	offer, err := parsePairingClaimOffer(payload)
	if err != nil {
		return nil, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if err := validatePairingClaimOffer(offer, now, true); err != nil {
		return nil, err
	}
	return offer, nil
}

// ParsePairingClaimOfferForExchange 校验 canonical offer，但把过期后的同 key 响应恢复交给 owning AccessStore 判断。
func ParsePairingClaimOfferForExchange(payload []byte) (*remoteauthpb.PairingClaimOffer, error) {
	offer, err := parsePairingClaimOffer(payload)
	if err != nil {
		return nil, err
	}
	if err := validatePairingClaimOffer(offer, time.Time{}, false); err != nil {
		return nil, err
	}
	return offer, nil
}

func parsePairingClaimOffer(payload []byte) (*remoteauthpb.PairingClaimOffer, error) {
	if len(payload) == 0 || len(payload) > maxPairingClaimOfferBytes {
		return nil, ErrPairingClaimMalformed
	}
	offer := &remoteauthpb.PairingClaimOffer{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, offer); err != nil || len(offer.ProtoReflect().GetUnknown()) != 0 {
		return nil, ErrPairingClaimMalformed
	}
	canonical, err := (proto.MarshalOptions{Deterministic: true}).Marshal(offer)
	if err != nil || !bytes.Equal(payload, canonical) {
		return nil, ErrPairingClaimMalformed
	}
	return offer, nil
}

func validatePairingClaimOffer(offer *remoteauthpb.PairingClaimOffer, now time.Time, requireFresh bool) error {
	if offer == nil || offer.GetSchemaVersion() != PairingClaimOfferVersion || len(offer.GetClaim()) != pairingClaimBytes || strings.TrimSpace(offer.GetDeviceId()) == "" || len(offer.GetDevicePublicKey()) != ed25519.PublicKeySize || offer.GetExpiresAtUnixNano() <= 0 {
		return ErrPairingClaimMalformed
	}
	if len(offer.GetRoutes()) == 0 || len(offer.GetRoutes()) > maxPairingClaimRoutes {
		return ErrPairingClaimMalformed
	}
	seenRouteIDs := make(map[string]struct{}, len(offer.GetRoutes()))
	for _, seed := range offer.GetRoutes() {
		if seed == nil || strings.TrimSpace(seed.GetRouteId()) == "" || strings.TrimSpace(seed.GetRouteId()) != seed.GetRouteId() || len(seed.GetRouteId()) > 64 || strings.TrimSpace(seed.GetDisplayName()) != seed.GetDisplayName() || len(seed.GetDisplayName()) > 128 {
			return ErrPairingClaimMalformed
		}
		if _, duplicate := seenRouteIDs[seed.GetRouteId()]; duplicate {
			return ErrPairingClaimMalformed
		}
		seenRouteIDs[seed.GetRouteId()] = struct{}{}
		switch route := seed.GetRoute().(type) {
		case *remoteauthpb.PairingRouteSeed_DirectWebrtcTcp:
			if route.DirectWebrtcTcp == nil || strings.TrimSpace(route.DirectWebrtcTcp.GetSignalingAddress()) == "" || strings.TrimSpace(route.DirectWebrtcTcp.GetIceTcpAddress()) == "" {
				return ErrPairingClaimMalformed
			}
		case *remoteauthpb.PairingRouteSeed_ManagedWebrtc:
			managed := route.ManagedWebrtc
			if managed == nil || strings.TrimSpace(managed.GetDaemonId()) == "" || strings.TrimSpace(managed.GetEdgeId()) == "" || strings.TrimSpace(managed.GetPublicEndpoint()) == "" || strings.TrimSpace(managed.GetServerName()) == "" || len(managed.GetCaCertificateDerSha256()) != sha256.Size {
				return ErrPairingClaimMalformed
			}
			host, port, splitErr := net.SplitHostPort(managed.GetPublicEndpoint())
			if splitErr != nil || strings.TrimSpace(host) == "" || strings.TrimSpace(port) == "" || strings.Contains(managed.GetPublicEndpoint(), "://") || strings.TrimSpace(managed.GetServerName()) != managed.GetServerName() {
				return ErrPairingClaimMalformed
			}
		case *remoteauthpb.PairingRouteSeed_SshWebrtcTcp:
			ssh := route.SshWebrtcTcp
			if ssh == nil || strings.TrimSpace(ssh.GetHost()) == "" || ssh.GetPort() == 0 || ssh.GetPort() > 65535 || strings.TrimSpace(ssh.GetUser()) == "" || len(ssh.GetHostKeyFingerprints()) == 0 || strings.TrimSpace(ssh.GetRemoteSignalingAddress()) == "" || strings.TrimSpace(ssh.GetRemoteIceTcpAddress()) == "" {
				return ErrPairingClaimMalformed
			}
			switch ssh.GetCredentialKind() {
			case remoteauthpb.EndpointCredentialKind_ENDPOINT_CREDENTIAL_KIND_SSH_AGENT,
				remoteauthpb.EndpointCredentialKind_ENDPOINT_CREDENTIAL_KIND_SSH_PRIVATE_KEY,
				remoteauthpb.EndpointCredentialKind_ENDPOINT_CREDENTIAL_KIND_SSH_PASSWORD:
			default:
				return ErrPairingClaimMalformed
			}
		default:
			return ErrPairingClaimMalformed
		}
	}
	if requireFresh && !time.Unix(0, offer.GetExpiresAtUnixNano()).UTC().After(now.UTC().Add(-ClockSkewTolerance)) {
		return ErrPairingTicketExpired
	}
	return nil
}

func pairingClaimRouteSeeds(routes []*remoteauthpb.EndpointRouteConfigV1) []*remoteauthpb.PairingRouteSeed {
	seeds := make([]*remoteauthpb.PairingRouteSeed, 0, min(len(routes), maxPairingClaimRoutes))
	for _, route := range routes {
		if route == nil || !route.GetEnabled() || len(seeds) == maxPairingClaimRoutes {
			continue
		}
		priority := int32(len(seeds) * 10)
		seed := &remoteauthpb.PairingRouteSeed{RouteId: route.GetRouteId(), DisplayName: route.GetDisplayName(), Priority: &priority}
		if direct := route.GetDirectWebrtcTcp(); direct != nil && len(direct.GetSignalingAddresses()) > 0 && len(direct.GetIceTcpAddresses()) > 0 {
			seed.Route = &remoteauthpb.PairingRouteSeed_DirectWebrtcTcp{DirectWebrtcTcp: &remoteauthpb.PairingDirectRouteSeed{
				SignalingAddress: direct.GetSignalingAddresses()[0], IceTcpAddress: direct.GetIceTcpAddresses()[0], ServerName: direct.GetServerName(),
			}}
			seeds = append(seeds, seed)
			continue
		}
		if managed := route.GetManagedWebrtc(); managed != nil && strings.TrimSpace(managed.GetTargetDeviceId()) != "" {
			seed.Route = &remoteauthpb.PairingRouteSeed_ManagedWebrtc{ManagedWebrtc: &remoteauthpb.PairingManagedRouteSeed{}}
			seeds = append(seeds, seed)
			continue
		}
		if ssh := route.GetSshWebrtcTcp(); ssh != nil && len(ssh.GetHostKeyFingerprints()) > 0 {
			kind := remoteauthpb.EndpointCredentialKind_ENDPOINT_CREDENTIAL_KIND_SSH_PRIVATE_KEY
			if descriptor := ssh.GetCredentialDescriptor(); descriptor != nil {
				kind = descriptor.GetKind()
			}
			seed.Route = &remoteauthpb.PairingRouteSeed_SshWebrtcTcp{SshWebrtcTcp: &remoteauthpb.PairingSSHRouteSeed{
				Host: ssh.GetHost(), Port: ssh.GetPort(), User: ssh.GetUser(), HostKeyFingerprints: append([]string(nil), ssh.GetHostKeyFingerprints()...),
				ProxyJump: ssh.GetProxyJump(), RemoteSignalingAddress: ssh.GetRemoteSignalingAddress(), RemoteIceTcpAddress: ssh.GetRemoteIceTcpAddress(), CredentialKind: kind,
			}}
			seeds = append(seeds, seed)
		}
	}
	return seeds
}

// PairingClaimEndpointCandidate 把紧凑 offer 投影为仅用于建立 pairing peer 的临时 Endpoint candidate。
// 完整 label、Route 列表和授权配置必须以 PairingAccepted 返回的签名 bundle 为准。
func PairingClaimEndpointCandidate(offer *remoteauthpb.PairingClaimOffer) (endpointdomain.EndpointCandidate, error) {
	if err := validatePairingClaimOffer(offer, time.Time{}, false); err != nil {
		return endpointdomain.EndpointCandidate{}, err
	}
	identity := endpointdomain.DaemonIdentity{DeviceID: offer.GetDeviceId(), DeviceFingerprint: Fingerprint(ed25519.PublicKey(offer.GetDevicePublicKey()))}
	routes := make([]endpointdomain.AccessRoute, 0, len(offer.GetRoutes()))
	for _, seed := range offer.GetRoutes() {
		route := endpointdomain.AccessRoute{ID: endpointdomain.RouteID(seed.GetRouteId()), DisplayName: seed.GetDisplayName(), Enabled: true, Source: endpointdomain.SourceBootstrap, PolicySource: endpointdomain.SourceUser}
		if seed.Priority != nil {
			priority := int(seed.GetPriority())
			route.Priority = &priority
		}
		switch value := seed.GetRoute().(type) {
		case *remoteauthpb.PairingRouteSeed_DirectWebrtcTcp:
			route.Kind = endpointdomain.RouteDirectWebRTCTCP
			route.SignalingAddresses = []string{value.DirectWebrtcTcp.GetSignalingAddress()}
			route.ICETCPAddresses = []string{value.DirectWebrtcTcp.GetIceTcpAddress()}
			route.ServerName = value.DirectWebrtcTcp.GetServerName()
		case *remoteauthpb.PairingRouteSeed_ManagedWebrtc:
			route.Kind = endpointdomain.RouteManagedWebRTC
			route.Source = endpointdomain.SourceCloud
			route.TargetDeviceID = offer.GetDeviceId()
			route.AccountProfileRef = "default"
			route.RelayMode = endpointdomain.RelayAuto
		case *remoteauthpb.PairingRouteSeed_SshWebrtcTcp:
			ssh := value.SshWebrtcTcp
			route.Kind = endpointdomain.RouteSSHWebRTCTCP
			route.Host = ssh.GetHost()
			route.Port = uint16(ssh.GetPort())
			route.User = ssh.GetUser()
			route.ProxyJump = ssh.GetProxyJump()
			route.HostKeyFingerprints = append([]string(nil), ssh.GetHostKeyFingerprints()...)
			route.RemoteSignalingAddress = ssh.GetRemoteSignalingAddress()
			route.RemoteICETCPAddress = ssh.GetRemoteIceTcpAddress()
			route.CredentialDescriptor = &endpointdomain.CredentialDescriptor{DescriptorID: "pairing-" + seed.GetRouteId(), Kind: pairingCredentialKind(ssh.GetCredentialKind())}
		default:
			return endpointdomain.EndpointCandidate{}, ErrPairingClaimMalformed
		}
		routes = append(routes, route)
	}
	return endpointdomain.EndpointCandidate{Source: endpointdomain.SourceBootstrap, Identity: identity, SuggestedLabel: offer.GetDeviceId(), Routes: routes}, nil
}

func pairingCredentialKind(value remoteauthpb.EndpointCredentialKind) endpointdomain.CredentialKind {
	switch value {
	case remoteauthpb.EndpointCredentialKind_ENDPOINT_CREDENTIAL_KIND_SSH_AGENT:
		return endpointdomain.CredentialSSHAgent
	case remoteauthpb.EndpointCredentialKind_ENDPOINT_CREDENTIAL_KIND_SSH_PASSWORD:
		return endpointdomain.CredentialSSHPassword
	default:
		return endpointdomain.CredentialSSHPrivateKey
	}
}

// EncodePairingClaimCode 把 canonical offer 编码为相机扫码和手工输入共用的紧凑文本。
// raw-DEFLATE 只有在包含 marker 后仍更短时才使用，避免小 payload 因压缩头部反而膨胀。
func EncodePairingClaimCode(payload []byte) string {
	envelope := append([]byte{pairingClaimEnvelopeRaw}, payload...)
	if compressed, err := deflatePairingClaim(payload); err == nil && len(compressed) < len(payload) {
		envelope = append([]byte{pairingClaimEnvelopeDeflate}, compressed...)
	}
	return PairingClaimCodePrefix + base64.RawURLEncoding.EncodeToString(envelope)
}

// DecodePairingClaimCode 解码 portable claim code；canonical protobuf 和有效期仍由 ParsePairingClaimOffer 校验。
func DecodePairingClaimCode(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, PairingClaimCodePrefix) {
		return nil, ErrPairingClaimMalformed
	}
	envelope, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, PairingClaimCodePrefix))
	if err != nil || len(envelope) < 2 {
		return nil, ErrPairingClaimMalformed
	}
	body := envelope[1:]
	switch envelope[0] {
	case pairingClaimEnvelopeRaw:
		if len(body) > maxPairingClaimOfferBytes {
			return nil, ErrPairingClaimMalformed
		}
		if compressed, compressErr := deflatePairingClaim(body); compressErr == nil && len(compressed) < len(body) {
			return nil, ErrPairingClaimMalformed
		}
		return append([]byte(nil), body...), nil
	case pairingClaimEnvelopeDeflate:
		reader := flate.NewReader(bytes.NewReader(body))
		decoded, readErr := io.ReadAll(io.LimitReader(reader, maxPairingClaimOfferBytes+1))
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || len(decoded) == 0 || len(decoded) > maxPairingClaimOfferBytes {
			return nil, ErrPairingClaimMalformed
		}
		canonical, compressErr := deflatePairingClaim(decoded)
		if compressErr != nil || len(canonical) >= len(decoded) || !bytes.Equal(canonical, body) {
			return nil, ErrPairingClaimMalformed
		}
		return decoded, nil
	default:
		return nil, ErrPairingClaimMalformed
	}
}

func deflatePairingClaim(payload []byte) ([]byte, error) {
	var compressed bytes.Buffer
	writer, err := flate.NewWriter(&compressed, flate.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err = writer.Write(payload); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	return compressed.Bytes(), nil
}

package remoteauth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"google.golang.org/protobuf/proto"
)

const (
	// PairingBundleVersion 与 connection owner 冻结的 EndpointBootstrapBundleV2 版本一致。
	// remoteauth 不再维护第二套 JSON bundle 或独立 ticket envelope。
	PairingBundleVersion     = endpointdomain.EndpointBootstrapBundleVersion
	pairingTicketVersion     = endpointdomain.PortableSignatureVersion
	defaultPairingTicketTTL  = 10 * time.Minute
	defaultPairingGrantTTL   = 90 * 24 * time.Hour
	maxPairingTicketTTL      = 24 * time.Hour
	maxPairingGrantTTL       = 365 * 24 * time.Hour
	maxPairingLabelBytes     = 128
	pairingScopeDaemon       = "base:daemon"
	pairingScopeMachineEvent = "base:machine_events"
	pairingScopeTerminal     = "base:terminal:"
	pairingScopeFileMetadata = "file:read_metadata"
	pairingScopeFileRead     = "file:read_content"
	pairingScopeFileWrite    = "file:write_content"
	pairingScopeFileMutate   = "file:mutate"
	pairingScopeManageAccess = "manage:client_access"
)

var (
	// ErrPairingTicketMalformed 表示 canonical protobuf、签名输入或必填 claims 不符合当前协议。
	ErrPairingTicketMalformed = errors.New("remote pairing ticket malformed")
	// ErrPairingTicketExpired 表示 ticket 尚未生效或已经过期。
	ErrPairingTicketExpired = errors.New("remote pairing ticket expired")
	// ErrPairingTicketConsumed 表示 ticket 已绑定其他 ClientAccessIdentity，不能再次兑换。
	ErrPairingTicketConsumed = errors.New("remote pairing ticket consumed")
)

// PairingTicketClaims 是 DeviceIdentity 签名的一次性配对授权上限。
// TicketID、scope ceiling、短有效期、nonce 与 MaxRedemptions=1 由 daemon-local AccessStore 持久化；ticket 本身不能访问 terminal protocol。
type PairingTicketClaims struct {
	Version                 uint32    `json:"version"`
	TicketID                string    `json:"ticket_id"`
	IssuerDeviceID          string    `json:"issuer_device_id"`
	IssuerDeviceFingerprint string    `json:"issuer_device_fingerprint"`
	ScopeCeiling            Scope     `json:"scope_ceiling"`
	IssuedAt                time.Time `json:"issued_at"`
	NotBefore               time.Time `json:"not_before"`
	ExpiresAt               time.Time `json:"expires_at"`
	GrantLifetimeSeconds    int64     `json:"grant_lifetime_seconds"`
	Nonce                   string    `json:"nonce"`
	MaxRedemptions          uint32    `json:"max_redemptions"`
}

// PairingBundle 是 CONN001 冻结的 deterministic protobuf EndpointBootstrapBundleV2。
// 二维码、owner-only 文件、CLI 与 App 只能传递这一份签名 wire，不再存在 JSON pairing bundle 或独立 ticket 字符串。
type PairingBundle = remoteauthpb.EndpointBootstrapBundleV2

// PairingIssueOptions 描述 local owner 或 ManageClientAccess 签发一次性 ticket 的输入。
// TicketTTL 只控制兑换窗口；GrantLifetime 控制成功兑换后 client-bound grant 的有效期，两者都不能被 Cloud 账号扩大。
type PairingIssueOptions struct {
	Label         string
	Platform      string
	Scope         Scope
	TicketTTL     time.Duration
	GrantLifetime time.Duration
	Now           time.Time
	Random        io.Reader
	Routes        []*remoteauthpb.EndpointRouteConfigV1
}

func issuePairingBundle(identity Identity, options PairingIssueOptions) (*PairingBundle, PairingTicketClaims, error) {
	if err := identity.Validate(); err != nil {
		return nil, PairingTicketClaims{}, fmt.Errorf("issue pairing ticket: %w", err)
	}
	if err := validateScope(options.Scope); err != nil {
		return nil, PairingTicketClaims{}, err
	}
	label, err := normalizePairingLabel(options.Label, "pairing bundle label")
	if err != nil {
		return nil, PairingTicketClaims{}, err
	}
	now := options.Now.UTC()
	if options.Now.IsZero() {
		now = time.Now().UTC()
	}
	ticketTTL := options.TicketTTL
	if ticketTTL == 0 {
		ticketTTL = defaultPairingTicketTTL
	}
	grantLifetime := options.GrantLifetime
	if grantLifetime == 0 {
		grantLifetime = defaultPairingGrantTTL
	}
	if ticketTTL <= 0 || ticketTTL > maxPairingTicketTTL || grantLifetime <= 0 || grantLifetime > maxPairingGrantTTL {
		return nil, PairingTicketClaims{}, fmt.Errorf("pairing ticket lifetime must be at most 24 hours and grant lifetime at most one year")
	}
	randomSource := options.Random
	if randomSource == nil {
		randomSource = rand.Reader
	}
	ticketID, err := randomIdentifier(randomSource, "ticket-")
	if err != nil {
		return nil, PairingTicketClaims{}, fmt.Errorf("generate pairing ticket id: %w", err)
	}
	bundleID, err := randomIdentifier(randomSource, "bundle-")
	if err != nil {
		return nil, PairingTicketClaims{}, fmt.Errorf("generate pairing bundle id: %w", err)
	}
	nonce := make([]byte, 18)
	if _, err := io.ReadFull(randomSource, nonce); err != nil {
		return nil, PairingTicketClaims{}, fmt.Errorf("generate pairing ticket nonce: %w", err)
	}
	scopeCeiling, err := pairingScopeStrings(options.Scope)
	if err != nil {
		return nil, PairingTicketClaims{}, err
	}
	ticket := &remoteauthpb.PairingTicketDescriptor{
		TicketId: ticketID, ScopeCeiling: scopeCeiling, IssuedAtUnixNano: now.UnixNano(),
		ExpiresAtUnixNano: now.Add(ticketTTL).UnixNano(), Nonce: append([]byte(nil), nonce...),
		MaxRedemptions: 1, GrantLifetimeSeconds: int64(grantLifetime / time.Second),
	}
	wireIdentity := &remoteauthpb.EndpointDaemonIdentity{
		DeviceId: identity.DeviceID, DevicePublicKey: append([]byte(nil), identity.PublicKey...), DeviceFingerprint: identity.Fingerprint,
	}
	ticketSigningBytes, err := endpointdomain.PairingTicketSigningBytes(wireIdentity, ticket)
	if err != nil {
		return nil, PairingTicketClaims{}, fmt.Errorf("build pairing ticket signature input: %w", err)
	}
	ticket.Signature = ed25519.Sign(identity.PrivateKey, ticketSigningBytes)
	bundle := &remoteauthpb.EndpointBootstrapBundleV2{
		SchemaVersion: PairingBundleVersion, BundleId: bundleID, Identity: wireIdentity, SuggestedLabel: label,
		Platform:         strings.ToLower(strings.TrimSpace(options.Platform)),
		Routes:           clonePairingRoutes(options.Routes),
		Authorization:    &remoteauthpb.EndpointAuthorizationBootstrap{Payload: &remoteauthpb.EndpointAuthorizationBootstrap_PairingTicket{PairingTicket: ticket}},
		IssuedAtUnixNano: now.UnixNano(), ExpiresAtUnixNano: ticket.GetExpiresAtUnixNano(),
	}
	bundleSigningBytes, err := endpointdomain.EndpointBootstrapSigningBytes(bundle)
	if err != nil {
		return nil, PairingTicketClaims{}, fmt.Errorf("build pairing bundle signature input: %w", err)
	}
	bundle.BundleSignature = ed25519.Sign(identity.PrivateKey, bundleSigningBytes)
	claims, err := pairingTicketClaimsFromBundle(bundle)
	if err != nil {
		return nil, PairingTicketClaims{}, err
	}
	return bundle, claims, nil
}

func clonePairingRoutes(routes []*remoteauthpb.EndpointRouteConfigV1) []*remoteauthpb.EndpointRouteConfigV1 {
	cloned := make([]*remoteauthpb.EndpointRouteConfigV1, 0, len(routes))
	for _, route := range routes {
		if route != nil {
			cloned = append(cloned, proto.Clone(route).(*remoteauthpb.EndpointRouteConfigV1))
		}
	}
	return cloned
}

// EncodePairingBundle 严格校验并以 deterministic protobuf 编码一次性 bootstrap bundle。
// 返回 bytes 可以进入静态二维码；调用方不得附加 grant、SSH credential、Cloud token 或客户端私钥。
func EncodePairingBundle(bundle *PairingBundle) ([]byte, error) {
	return endpointdomain.MarshalEndpointBootstrapBundle(bundle)
}

// ParsePairingBundle 严格解析 bundle 并验证两层 daemon signature、identity、scope ceiling 与短期有效期。
// 成功只证明 ticket 来源和当前可兑换，不代表 client 已经持有 terminal capability；长期 grant 必须经 PairingExchange 取得。
func ParsePairingBundle(payload []byte, now time.Time) (*PairingBundle, PairingTicketClaims, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	bundle, err := endpointdomain.ParseEndpointBootstrapBundleForExchange(payload)
	if err != nil {
		return nil, PairingTicketClaims{}, fmt.Errorf("%w: %v", ErrPairingTicketMalformed, err)
	}
	claims, err := pairingTicketClaimsFromBundle(bundle)
	if err != nil {
		return nil, PairingTicketClaims{}, err
	}
	if err := validatePairingTicketTime(claims, now); err != nil {
		return nil, PairingTicketClaims{}, err
	}
	return bundle, claims, nil
}

// ParsePairingBundleForExchange 严格解析并验签配对 bundle，但把兑换时效判断留给 owning daemon 的 AccessStore。
// 该入口只供持久化 ClientAccessIdentity 后的 PairingExchange 使用：首次兑换仍必须在 ticket 有效期内，只有 daemon 已原子绑定的同一 key 才能在 delivery grace 内取回原结果。
func ParsePairingBundleForExchange(payload []byte) (*PairingBundle, PairingTicketClaims, error) {
	bundle, err := endpointdomain.ParseEndpointBootstrapBundleForExchange(payload)
	if err != nil {
		return nil, PairingTicketClaims{}, fmt.Errorf("%w: %v", ErrPairingTicketMalformed, err)
	}
	claims, err := pairingTicketClaimsFromBundle(bundle)
	if err != nil {
		return nil, PairingTicketClaims{}, err
	}
	return bundle, claims, nil
}

// VerifyPairingTicket 验证完整 canonical bootstrap bytes 与期望 daemon fingerprint，并返回其中的一次性 ticket claims。
// daemon 仍必须查询 AccessStore 中的 TicketID 摘要并原子消费；仅验签不能作为兑换成功或 terminal authorization。
func VerifyPairingTicket(payload []byte, expectedFingerprint string, now time.Time) (PairingTicketClaims, error) {
	bundle, claims, err := ParsePairingBundle(payload, now)
	if err != nil {
		return PairingTicketClaims{}, err
	}
	if subtle.ConstantTimeCompare([]byte(bundle.GetIdentity().GetDeviceFingerprint()), []byte(strings.TrimSpace(expectedFingerprint))) != 1 {
		return PairingTicketClaims{}, ErrGrantFingerprintMismatch
	}
	return claims, nil
}

func pairingTicketClaimsFromBundle(bundle *PairingBundle) (PairingTicketClaims, error) {
	if bundle == nil || bundle.GetIdentity() == nil || bundle.GetAuthorization() == nil || bundle.GetAuthorization().GetPairingTicket() == nil {
		return PairingTicketClaims{}, fmt.Errorf("%w: bootstrap does not contain a pairing ticket", ErrPairingTicketMalformed)
	}
	ticket := bundle.GetAuthorization().GetPairingTicket()
	scope, err := pairingScopeFromStrings(ticket.GetScopeCeiling())
	if err != nil {
		return PairingTicketClaims{}, err
	}
	issuedAt := time.Unix(0, ticket.GetIssuedAtUnixNano()).UTC()
	claims := normalizePairingTicketClaims(PairingTicketClaims{
		Version: pairingTicketVersion, TicketID: ticket.GetTicketId(), IssuerDeviceID: bundle.GetIdentity().GetDeviceId(),
		IssuerDeviceFingerprint: bundle.GetIdentity().GetDeviceFingerprint(), ScopeCeiling: scope,
		IssuedAt: issuedAt, NotBefore: issuedAt, ExpiresAt: time.Unix(0, ticket.GetExpiresAtUnixNano()).UTC(),
		GrantLifetimeSeconds: ticket.GetGrantLifetimeSeconds(), Nonce: base64.RawURLEncoding.EncodeToString(ticket.GetNonce()),
		MaxRedemptions: ticket.GetMaxRedemptions(),
	})
	if err := validatePairingTicketClaims(claims); err != nil {
		return PairingTicketClaims{}, err
	}
	return claims, nil
}

func validatePairingTicketTime(claims PairingTicketClaims, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	if claims.NotBefore.After(now.Add(ClockSkewTolerance)) || !claims.ExpiresAt.After(now.Add(-ClockSkewTolerance)) {
		return ErrPairingTicketExpired
	}
	return nil
}

func normalizePairingTicketClaims(claims PairingTicketClaims) PairingTicketClaims {
	claims.TicketID = strings.TrimSpace(claims.TicketID)
	claims.IssuerDeviceID = strings.TrimSpace(claims.IssuerDeviceID)
	claims.IssuerDeviceFingerprint = strings.TrimSpace(claims.IssuerDeviceFingerprint)
	claims.ScopeCeiling.TerminalID = strings.TrimSpace(claims.ScopeCeiling.TerminalID)
	claims.IssuedAt = claims.IssuedAt.UTC()
	claims.NotBefore = claims.NotBefore.UTC()
	claims.ExpiresAt = claims.ExpiresAt.UTC()
	claims.Nonce = strings.TrimSpace(claims.Nonce)
	return claims
}

func validatePairingTicketClaims(claims PairingTicketClaims) error {
	if claims.Version != pairingTicketVersion || claims.TicketID == "" || claims.IssuerDeviceID == "" || claims.IssuerDeviceFingerprint == "" || claims.Nonce == "" {
		return fmt.Errorf("%w: incomplete claims", ErrPairingTicketMalformed)
	}
	if claims.MaxRedemptions != 1 || claims.GrantLifetimeSeconds <= 0 || claims.GrantLifetimeSeconds > int64(maxPairingGrantTTL/time.Second) {
		return fmt.Errorf("%w: ticket must allow exactly one bounded redemption", ErrPairingTicketMalformed)
	}
	if err := validateScope(claims.ScopeCeiling); err != nil {
		return err
	}
	if claims.IssuedAt.IsZero() || claims.NotBefore.IsZero() || claims.ExpiresAt.IsZero() || claims.NotBefore.Before(claims.IssuedAt) ||
		!claims.ExpiresAt.After(claims.NotBefore) || claims.ExpiresAt.Sub(claims.IssuedAt) > maxPairingTicketTTL {
		return fmt.Errorf("%w: invalid time window", ErrPairingTicketMalformed)
	}
	return nil
}

func pairingScopeStrings(scope Scope) ([]string, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	values := make([]string, 0, 6)
	switch {
	case scope.AllowDaemon:
		values = append(values, pairingScopeDaemon)
	case scope.TerminalID != "":
		values = append(values, pairingScopeTerminal+base64.RawURLEncoding.EncodeToString([]byte(strings.TrimSpace(scope.TerminalID))))
	case scope.MachineEventsOnly:
		values = append(values, pairingScopeMachineEvent)
	}
	if scope.FileReadMetadata {
		values = append(values, pairingScopeFileMetadata)
	}
	if scope.FileReadContent {
		values = append(values, pairingScopeFileRead)
	}
	if scope.FileWriteContent {
		values = append(values, pairingScopeFileWrite)
	}
	if scope.FileMutate {
		values = append(values, pairingScopeFileMutate)
	}
	if scope.ManageClientAccess {
		values = append(values, pairingScopeManageAccess)
	}
	sort.Strings(values)
	return values, nil
}

func pairingScopeFromStrings(values []string) (Scope, error) {
	var scope Scope
	previous := ""
	for _, value := range values {
		if value == "" || value != strings.TrimSpace(value) || (previous != "" && previous >= value) {
			return Scope{}, fmt.Errorf("%w: pairing scope list is not canonical", ErrGrantScopeInvalid)
		}
		previous = value
		switch value {
		case pairingScopeDaemon:
			scope.AllowDaemon = true
		case pairingScopeMachineEvent:
			scope.MachineEventsOnly = true
		case pairingScopeFileMetadata:
			scope.FileReadMetadata = true
		case pairingScopeFileRead:
			scope.FileReadContent = true
		case pairingScopeFileWrite:
			scope.FileWriteContent = true
		case pairingScopeFileMutate:
			scope.FileMutate = true
		case pairingScopeManageAccess:
			scope.ManageClientAccess = true
		default:
			if !strings.HasPrefix(value, pairingScopeTerminal) {
				return Scope{}, fmt.Errorf("%w: unknown pairing scope %q", ErrGrantScopeInvalid, value)
			}
			terminalID, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, pairingScopeTerminal))
			if err != nil || len(terminalID) == 0 || string(terminalID) != strings.TrimSpace(string(terminalID)) {
				return Scope{}, fmt.Errorf("%w: terminal pairing scope is invalid", ErrGrantScopeInvalid)
			}
			scope.TerminalID = string(terminalID)
		}
	}
	if err := validateScope(scope); err != nil {
		return Scope{}, err
	}
	return scope, nil
}

func normalizePairingLabel(value string, field string) (string, error) {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) || len(value) > maxPairingLabelBytes {
		return "", fmt.Errorf("%s must be valid UTF-8 and at most %d bytes", field, maxPairingLabelBytes)
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return "", fmt.Errorf("%s cannot contain control characters", field)
		}
	}
	return value, nil
}

func randomIdentifier(reader io.Reader, prefix string) (string, error) {
	raw := make([]byte, 18)
	if _, err := io.ReadFull(reader, raw); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

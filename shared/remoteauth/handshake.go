package remoteauth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/transport"
)

const (
	authSessionBytes = 18
	authNonceBytes   = 32
	defaultClockSkew = 2 * time.Minute
)

// ClientHandshakeRequest 是 client-bound capability 认证的本地输入。
// Endpoint pin 来自 SavedEndpoint，Credential 来自平台 secure store，ChannelBinding 来自当前实际 TLS/DTLS adapter；三者不能由 Cloud 结果替代。
type ClientHandshakeRequest struct {
	ExpectedDeviceID          string
	ExpectedDeviceFingerprint string
	Credential                ClientAccessCredential
	Signer                    ClientAccessSigner
	ChannelBinding            ChannelBinding
}

// ClientPairingRequest 是 PairingTicket 受限兑换状态机的本地输入。
// Identity 必须在发起请求前持久化到当前 Endpoint credential ref，以便响应丢失后用同一 key 幂等恢复。
type ClientPairingRequest struct {
	ExpectedDeviceID          string
	ExpectedDeviceFingerprint string
	PairingClaimOffer         []byte
	Identity                  ClientAccessIdentity
	Signer                    ClientAccessSigner
	ClientLabel               string
	ClientProduct             uint32
	ChannelBinding            ChannelBinding
}

// ClientHandshake 是公开客户端在 anytty protocol 前执行的 DeviceHello/CapabilityOpen 状态机。
// Random 和 Now 仅用于 deterministic harness；零值使用 crypto/rand 与 UTC 当前时间，不能注入云侧授权结果。
type ClientHandshake struct {
	Random io.Reader
	Now    func() time.Time
}

// Authenticate 完成 daemon pin、actual channel binding、grant issuer/subject、ClientAccessIdentity possession 和 accepted scope 验证。
// 成功后 transport 才能切换到 anytty protocol；失败时调用方必须关闭当前 transport，不能回退到 v1 bearer/HMAC。
func (handshake ClientHandshake) Authenticate(ctx context.Context, connection transport.Transport, request ClientHandshakeRequest) (Claims, error) {
	if connection == nil {
		return Claims{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "remote auth transport is nil", nil)
	}
	if err := validateClientSigner(request.Credential.Identity, request.Signer); err != nil || !request.Credential.Ready() {
		return Claims{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH, "client access credential is incomplete", err)
	}
	if err := request.ChannelBinding.Validate(); err != nil {
		return Claims{}, err
	}
	credentialNow := handshake.now()
	claims, err := Verify(request.Credential.CapabilityGrant, request.ExpectedDeviceFingerprint, credentialNow, nil)
	if err != nil {
		return Claims{}, mapGrantError(err)
	}
	if strings.TrimSpace(claims.IssuerDeviceID) != strings.TrimSpace(request.ExpectedDeviceID) {
		return Claims{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH, "capability issuer device does not match endpoint", nil)
	}
	if claims.SubjectKeyFingerprint != request.Credential.Identity.Fingerprint {
		return Claims{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH, "capability subject does not match ClientAccessIdentity", ErrGrantSubjectMismatch)
	}
	helloEnvelope, err := receiveAuthEnvelope(ctx, connection)
	if err != nil {
		return Claims{}, err
	}
	hello := helloEnvelope.GetDeviceHello()
	if hello == nil {
		return Claims{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "daemon did not start with DeviceHello", nil)
	}
	if err := verifyDeviceHello(helloEnvelope.GetAuthSessionId(), hello, request.ExpectedDeviceID, request.ExpectedDeviceFingerprint, request.ChannelBinding, handshake.now()); err != nil {
		return Claims{}, err
	}
	clientNonce, err := randomBytes(handshake.random(), authNonceBytes)
	if err != nil {
		return Claims{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_INTERNAL, "generate client nonce", err)
	}
	proof, err := signClientProof(ctx, request.Credential.Identity, request.Signer, remoteauthpb.AuthOpenKind_AUTH_OPEN_KIND_CAPABILITY,
		[]byte(request.Credential.CapabilityGrant), helloEnvelope.GetAuthSessionId(), hello.GetServerNonce(), clientNonce, request.ChannelBinding)
	if err != nil {
		return Claims{}, err
	}
	open := &remoteauthpb.AuthEnvelope{
		Protocol: AuthProtocol, Version: AuthVersion, AuthSessionId: helloEnvelope.GetAuthSessionId(),
		Payload: &remoteauthpb.AuthEnvelope_CapabilityOpen{CapabilityOpen: &remoteauthpb.CapabilityOpen{
			Grant:           strings.TrimSpace(request.Credential.CapabilityGrant),
			ClientPublicKey: append([]byte(nil), request.Credential.Identity.PublicKey...), ClientNonce: clientNonce, Proof: proof,
		}},
	}
	if err := sendAuthEnvelope(ctx, connection, open); err != nil {
		return Claims{}, err
	}
	result, err := receiveAuthEnvelope(ctx, connection)
	if err != nil {
		return Claims{}, err
	}
	if err := validateResultSession(result, helloEnvelope.GetAuthSessionId()); err != nil {
		return Claims{}, err
	}
	if rejected := result.GetCapabilityRejected(); rejected != nil {
		return Claims{}, rejectionError(rejected)
	}
	accepted := result.GetCapabilityAccepted()
	if accepted == nil {
		return Claims{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "daemon did not complete capability authorization", nil)
	}
	if accepted.GetGrantId() != claims.GrantID || accepted.GetSubjectKeyFingerprint() != claims.SubjectKeyFingerprint || !scopeSummaryMatches(accepted.GetScope(), claims.Scope) {
		return Claims{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SCOPE_INVALID, "daemon accepted a different capability subject or scope", nil)
	}
	return claims, nil
}

// ClientPairingHandshake 是客户端只允许 PairingOpen/PairingAccepted 的受限状态机。
// 它不创建 anytty protocol session；成功结果仍须先写入平台 secure credential store，之后普通连接重新执行 capability handshake。
type ClientPairingHandshake struct {
	Random io.Reader
	Now    func() time.Time
}

// Redeem 验证 DeviceHello 后，用目标 ClientAccessIdentity 签名 ticket challenge，并验证返回 grant 的 issuer、subject、scope 与 receipt。
// 相同 ticket/key 可用于响应丢失恢复；服务端返回其他 subject、空 receipt 或 bearer v1 时一律拒绝。
func (handshake ClientPairingHandshake) Redeem(ctx context.Context, connection transport.Transport, request ClientPairingRequest) (PairingExchangeResult, error) {
	if connection == nil {
		return PairingExchangeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "pairing transport is nil", nil)
	}
	if err := validateClientSigner(request.Identity, request.Signer); err != nil {
		return PairingExchangeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH, "ClientAccessIdentity is invalid", err)
	}
	if err := request.ChannelBinding.Validate(); err != nil {
		return PairingExchangeResult{}, err
	}
	if len(request.PairingClaimOffer) == 0 {
		return PairingExchangeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "pairing credential is empty", nil)
	}
	offer, err := ParsePairingClaimOfferForExchange(request.PairingClaimOffer)
	if err != nil {
		return PairingExchangeResult{}, mapPairingError(err)
	}
	fingerprint := Fingerprint(ed25519.PublicKey(offer.GetDevicePublicKey()))
	if subtle.ConstantTimeCompare([]byte(fingerprint), []byte(strings.TrimSpace(request.ExpectedDeviceFingerprint))) != 1 || offer.GetDeviceId() != strings.TrimSpace(request.ExpectedDeviceID) {
		return PairingExchangeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH, "pairing claim issuer does not match endpoint", nil)
	}
	credential := request.PairingClaimOffer
	helloEnvelope, err := receiveAuthEnvelope(ctx, connection)
	if err != nil {
		return PairingExchangeResult{}, err
	}
	hello := helloEnvelope.GetDeviceHello()
	if hello == nil {
		return PairingExchangeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "daemon did not start with DeviceHello", nil)
	}
	if err := verifyDeviceHello(helloEnvelope.GetAuthSessionId(), hello, request.ExpectedDeviceID, request.ExpectedDeviceFingerprint, request.ChannelBinding, handshake.now()); err != nil {
		return PairingExchangeResult{}, err
	}
	clientNonce, err := randomBytes(handshake.random(), authNonceBytes)
	if err != nil {
		return PairingExchangeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_INTERNAL, "generate pairing nonce", err)
	}
	proof, err := signClientProof(ctx, request.Identity, request.Signer, remoteauthpb.AuthOpenKind_AUTH_OPEN_KIND_PAIRING,
		credential, helloEnvelope.GetAuthSessionId(), hello.GetServerNonce(), clientNonce, request.ChannelBinding)
	if err != nil {
		return PairingExchangeResult{}, err
	}
	if err := sendAuthEnvelope(ctx, connection, &remoteauthpb.AuthEnvelope{
		Protocol: AuthProtocol, Version: AuthVersion, AuthSessionId: helloEnvelope.GetAuthSessionId(),
		Payload: &remoteauthpb.AuthEnvelope_PairingOpen{PairingOpen: &remoteauthpb.PairingOpen{
			PairingClaimOffer: append([]byte(nil), request.PairingClaimOffer...), ClientPublicKey: append([]byte(nil), request.Identity.PublicKey...),
			ClientLabel: strings.TrimSpace(request.ClientLabel), ClientProduct: request.ClientProduct, ClientNonce: clientNonce, Proof: proof,
		}},
	}); err != nil {
		return PairingExchangeResult{}, err
	}
	response, err := receiveAuthEnvelope(ctx, connection)
	if err != nil {
		return PairingExchangeResult{}, err
	}
	if err := validateResultSession(response, helloEnvelope.GetAuthSessionId()); err != nil {
		return PairingExchangeResult{}, err
	}
	if rejected := response.GetCapabilityRejected(); rejected != nil {
		return PairingExchangeResult{}, rejectionError(rejected)
	}
	accepted := response.GetPairingAccepted()
	if accepted == nil || strings.TrimSpace(accepted.GetDeliveryReceipt()) == "" {
		return PairingExchangeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "daemon did not complete PairingExchange", nil)
	}
	// 新签发 grant 必须按 PairingAccepted 实际接收时间验证；DeviceHello 时刻不能让已过期结果进入 secure store。
	responseNow := handshake.now()
	responseBundle, ticketClaims, err := ParsePairingBundleForExchange(accepted.GetPairingBundle())
	if err != nil || responseBundle.GetIdentity().GetDeviceId() != request.ExpectedDeviceID || responseBundle.GetIdentity().GetDeviceFingerprint() != request.ExpectedDeviceFingerprint {
		return PairingExchangeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PAIRING_TICKET_INVALID, "daemon returned an invalid pairing bundle", err)
	}
	claims, err := Verify(accepted.GetGrant(), request.ExpectedDeviceFingerprint, responseNow, nil)
	if err != nil {
		return PairingExchangeResult{}, mapGrantError(err)
	}
	if claims.IssuerDeviceID != request.ExpectedDeviceID || claims.SubjectKeyFingerprint != request.Identity.Fingerprint ||
		accepted.GetSubjectKeyFingerprint() != request.Identity.Fingerprint || !scopeSummaryMatches(accepted.GetScope(), claims.Scope) || claims.Scope != ticketClaims.ScopeCeiling {
		return PairingExchangeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH, "pairing response grant subject or scope is invalid", nil)
	}
	if pairingBundleHasManagedRoute(responseBundle) && len(accepted.GetCloudRouteGrant()) == 0 {
		return PairingExchangeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "daemon omitted the managed Route discovery grant", nil)
	}
	if pairingBundleHasManagedRoute(responseBundle) && len(accepted.GetCloudEdgeLocator()) == 0 {
		return PairingExchangeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "daemon omitted the managed Edge locator", nil)
	}
	return PairingExchangeResult{
		TicketID: ticketClaims.TicketID, Grant: strings.TrimSpace(accepted.GetGrant()), GrantID: claims.GrantID,
		DeliveryReceipt: strings.TrimSpace(accepted.GetDeliveryReceipt()), SubjectKeyFingerprint: claims.SubjectKeyFingerprint,
		Scope: claims.Scope, ExpiresAt: claims.ExpiresAt, Bundle: append([]byte(nil), accepted.GetPairingBundle()...), CloudRouteGrant: append([]byte(nil), accepted.GetCloudRouteGrant()...), CloudEdgeLocator: append([]byte(nil), accepted.GetCloudEdgeLocator()...),
	}, nil
}

func validateClientSigner(identity ClientAccessIdentity, signer ClientAccessSigner) error {
	if signer != nil {
		return identity.ValidatePublic()
	}
	return identity.Validate()
}

// ServerHandshakeMode 区分成功握手后的唯一下一步。
// capability 模式可以进入 anytty protocol；pairing 模式已经返回 grant 后必须关闭 transport，不能复用为业务 session。
type ServerHandshakeMode uint8

const (
	// ServerHandshakeModeCapability 表示 client-bound capability 已验证，可进入 scoped anytty protocol。
	ServerHandshakeModeCapability ServerHandshakeMode = iota + 1
	// ServerHandshakeModePairing 表示 PairingExchange 已完成，当前 transport 必须在响应后关闭。
	ServerHandshakeModePairing
)

// ServerHandshakeResult 是 daemon auth 状态机的成功结果。
// Mode 决定消息链路；Claims 只在 capability 模式有效，Pairing 只在 pairing 模式有效，调用方不得自行推断或合并两者。
type ServerHandshakeResult struct {
	Mode    ServerHandshakeMode
	Claims  Claims
	Pairing PairingExchangeResult
}

// ServerHandshake 是 daemon 在同一 transport 上证明 DeviceIdentity，并验证 capability 或受限 PairingExchange 的状态机。
// Identity、AccessStore、revocation truth 和 scope mapping 全部属于 daemon；Random/Now 注入只服务 deterministic harness。
type ServerHandshake struct {
	Identity    Identity
	AccessStore *AccessStore
	// PairingOnly 把 owner-only listener 限制为 PairingOpen；CapabilityOpen 会在任何 accepted 响应或 core 切换前拒绝。
	// managed/direct ingress 保持 false，以允许未配对 client 兑换 ticket 或已配对 client 建立业务 session。
	PairingOnly bool
	Random      io.Reader
	Now         func() time.Time
}

// Accept 发送绑定当前 actual channel 的 DeviceHello，并只接受一个 CapabilityOpen 或 PairingOpen。
// 返回前不会调用 core；失败会尽力发送脱敏 rejection，调用方随后必须关闭 transport，且不得尝试另一种 auth mode。
func (handshake ServerHandshake) Accept(ctx context.Context, connection transport.Transport, binding ChannelBinding) (ServerHandshakeResult, error) {
	if connection == nil {
		return ServerHandshakeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "remote auth transport is nil", nil)
	}
	if err := handshake.Identity.Validate(); err != nil {
		return ServerHandshakeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_INTERNAL, "daemon DeviceIdentity is invalid", err)
	}
	if err := binding.Validate(); err != nil {
		return ServerHandshakeResult{}, err
	}
	if handshake.AccessStore == nil || !handshake.AccessStore.Available() {
		return ServerHandshakeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_INTERNAL, "daemon client access store is unavailable", nil)
	}
	if binding.Kind == remoteauthpb.ChannelBindingKind_CHANNEL_BINDING_KIND_LOCAL_UNIX && !handshake.PairingOnly {
		return ServerHandshakeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "local Unix auth is pairing-only", nil)
	}
	authSessionRaw, err := randomBytes(handshake.random(), authSessionBytes)
	if err != nil {
		return ServerHandshakeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_INTERNAL, "generate auth session id", err)
	}
	authSessionID := base64.RawURLEncoding.EncodeToString(authSessionRaw)
	serverNonce, err := randomBytes(handshake.random(), authNonceBytes)
	if err != nil {
		return ServerHandshakeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_INTERNAL, "generate server nonce", err)
	}
	now := handshake.now()
	hello := &remoteauthpb.DeviceHello{
		DeviceId: handshake.Identity.DeviceID, DevicePublicKey: append([]byte(nil), handshake.Identity.PublicKey...),
		DeviceFingerprint: handshake.Identity.Fingerprint, ServerNonce: serverNonce,
		ChannelBinding: channelBindingToProto(binding), IssuedAtUnixNano: now.UnixNano(),
	}
	signingBytes, err := DeviceHelloSigningBytes(authSessionID, hello)
	if err != nil {
		return ServerHandshakeResult{}, err
	}
	hello.Signature = ed25519.Sign(handshake.Identity.PrivateKey, signingBytes)
	if err := sendAuthEnvelope(ctx, connection, &remoteauthpb.AuthEnvelope{
		Protocol: AuthProtocol, Version: AuthVersion, AuthSessionId: authSessionID,
		Payload: &remoteauthpb.AuthEnvelope_DeviceHello{DeviceHello: hello},
	}); err != nil {
		return ServerHandshakeResult{}, err
	}
	openEnvelope, err := receiveAuthEnvelope(ctx, connection)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ServerHandshakeResult{}, ctxErr
		}
		return ServerHandshakeResult{}, handshake.reject(ctx, connection, authSessionID, err)
	}
	if subtle.ConstantTimeCompare([]byte(openEnvelope.GetAuthSessionId()), []byte(authSessionID)) != 1 {
		err := newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_REPLAYED, "auth open used a different auth session", nil)
		return ServerHandshakeResult{}, handshake.reject(ctx, connection, authSessionID, err)
	}
	// credential 的有效期以收到并验证 open 的时刻为准；challenge 发出时刻不能冻结后续授权时间。
	openNow := handshake.now()
	switch payload := openEnvelope.GetPayload().(type) {
	case *remoteauthpb.AuthEnvelope_CapabilityOpen:
		if handshake.PairingOnly {
			err := newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "pairing-only transport rejects CapabilityOpen", nil)
			return ServerHandshakeResult{}, handshake.reject(ctx, connection, authSessionID, err)
		}
		result, err := handshake.acceptCapability(ctx, connection, authSessionID, serverNonce, binding, openNow, payload.CapabilityOpen)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ServerHandshakeResult{}, ctxErr
			}
			return ServerHandshakeResult{}, handshake.reject(ctx, connection, authSessionID, err)
		}
		return result, nil
	case *remoteauthpb.AuthEnvelope_PairingOpen:
		result, err := handshake.acceptPairing(ctx, connection, authSessionID, serverNonce, binding, openNow, payload.PairingOpen)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ServerHandshakeResult{}, ctxErr
			}
			return ServerHandshakeResult{}, handshake.reject(ctx, connection, authSessionID, err)
		}
		return result, nil
	default:
		err := newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "client did not send a supported auth open", nil)
		return ServerHandshakeResult{}, handshake.reject(ctx, connection, authSessionID, err)
	}
}

func (handshake ServerHandshake) acceptCapability(ctx context.Context, connection transport.Transport, authSessionID string, serverNonce []byte, binding ChannelBinding, now time.Time, open *remoteauthpb.CapabilityOpen) (ServerHandshakeResult, error) {
	if open == nil || len(open.GetClientPublicKey()) != ed25519.PublicKeySize || len(open.GetClientNonce()) != authNonceBytes || len(open.GetProof()) != ed25519.SignatureSize {
		return ServerHandshakeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "client did not send a valid CapabilityOpen", nil)
	}
	claims, err := Verify(open.GetGrant(), handshake.Identity.Fingerprint, now, handshake.AccessStore)
	if err != nil {
		return ServerHandshakeResult{}, mapGrantError(err)
	}
	if claims.IssuerDeviceID != handshake.Identity.DeviceID || claims.IssuerDeviceFingerprint != handshake.Identity.Fingerprint {
		return ServerHandshakeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_INVALID, "capability issuer does not match daemon identity", nil)
	}
	clientPublicKey := ed25519.PublicKey(open.GetClientPublicKey())
	if claims.SubjectKeyFingerprint != Fingerprint(clientPublicKey) {
		return ServerHandshakeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH, "capability subject does not match client public key", ErrGrantSubjectMismatch)
	}
	canonical, err := ClientProofSigningBytes(remoteauthpb.AuthOpenKind_AUTH_OPEN_KIND_CAPABILITY, []byte(open.GetGrant()), clientPublicKey, authSessionID, serverNonce, open.GetClientNonce(), binding)
	if err != nil || !ed25519.Verify(clientPublicKey, canonical, open.GetProof()) {
		return ServerHandshakeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_PROOF_INVALID, "ClientAccessIdentity proof does not match current challenge", err)
	}
	summary, err := scopeSummary(claims.Scope)
	if err != nil {
		return ServerHandshakeResult{}, err
	}
	if err := sendAuthEnvelope(ctx, connection, &remoteauthpb.AuthEnvelope{
		Protocol: AuthProtocol, Version: AuthVersion, AuthSessionId: authSessionID,
		Payload: &remoteauthpb.AuthEnvelope_CapabilityAccepted{CapabilityAccepted: &remoteauthpb.CapabilityAccepted{
			GrantId: claims.GrantID, Scope: summary, SubjectKeyFingerprint: claims.SubjectKeyFingerprint,
		}},
	}); err != nil {
		return ServerHandshakeResult{}, err
	}
	return ServerHandshakeResult{Mode: ServerHandshakeModeCapability, Claims: claims}, nil
}

func (handshake ServerHandshake) acceptPairing(ctx context.Context, connection transport.Transport, authSessionID string, serverNonce []byte, binding ChannelBinding, now time.Time, open *remoteauthpb.PairingOpen) (ServerHandshakeResult, error) {
	if open == nil || len(open.GetPairingClaimOffer()) == 0 || len(open.GetClientPublicKey()) != ed25519.PublicKeySize || len(open.GetClientNonce()) != authNonceBytes || len(open.GetProof()) != ed25519.SignatureSize {
		return ServerHandshakeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "client did not send a valid PairingOpen", nil)
	}
	clientPublicKey := ed25519.PublicKey(open.GetClientPublicKey())
	credential := open.GetPairingClaimOffer()
	bundlePayload, err := handshake.AccessStore.ResolvePairingClaimForExchange(credential, clientPublicKey, now)
	if err != nil {
		return ServerHandshakeResult{}, mapPairingError(err)
	}
	bundle, ticketClaims, err := ParsePairingBundleForExchange(bundlePayload)
	if err != nil {
		return ServerHandshakeResult{}, mapPairingError(err)
	}
	if subtle.ConstantTimeCompare([]byte(bundle.GetIdentity().GetDeviceFingerprint()), []byte(handshake.Identity.Fingerprint)) != 1 || ticketClaims.IssuerDeviceID != handshake.Identity.DeviceID {
		return ServerHandshakeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PAIRING_TICKET_INVALID, "pairing ticket issuer does not match daemon", nil)
	}
	canonical, err := ClientProofSigningBytes(remoteauthpb.AuthOpenKind_AUTH_OPEN_KIND_PAIRING, credential, clientPublicKey, authSessionID, serverNonce, open.GetClientNonce(), binding)
	if err != nil || !ed25519.Verify(clientPublicKey, canonical, open.GetProof()) {
		return ServerHandshakeResult{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_PROOF_INVALID, "ClientAccessIdentity pairing proof does not match current challenge", err)
	}
	result, bundlePayload, err := handshake.AccessStore.RedeemPairingClaimForProduct(open.GetPairingClaimOffer(), clientPublicKey, open.GetClientLabel(), open.GetClientProduct(), now)
	if err != nil {
		return ServerHandshakeResult{}, mapPairingError(err)
	}
	summary, err := scopeSummary(result.Scope)
	if err != nil {
		return ServerHandshakeResult{}, err
	}
	if err := sendAuthEnvelope(ctx, connection, &remoteauthpb.AuthEnvelope{
		Protocol: AuthProtocol, Version: AuthVersion, AuthSessionId: authSessionID,
		Payload: &remoteauthpb.AuthEnvelope_PairingAccepted{PairingAccepted: &remoteauthpb.PairingAccepted{
			Grant: result.Grant, DeliveryReceipt: result.DeliveryReceipt,
			SubjectKeyFingerprint: result.SubjectKeyFingerprint, Scope: summary, PairingBundle: append([]byte(nil), bundlePayload...), CloudRouteGrant: append([]byte(nil), result.CloudRouteGrant...), CloudEdgeLocator: append([]byte(nil), result.CloudEdgeLocator...),
		}},
	}); err != nil {
		return ServerHandshakeResult{}, err
	}
	return ServerHandshakeResult{Mode: ServerHandshakeModePairing, Pairing: result}, nil
}

func verifyDeviceHello(authSessionID string, hello *remoteauthpb.DeviceHello, expectedDeviceID string, expectedFingerprint string, expectedBinding ChannelBinding, now time.Time) error {
	issuedAt := time.Unix(0, hello.GetIssuedAtUnixNano()).UTC()
	if hello.GetIssuedAtUnixNano() <= 0 || issuedAt.Before(now.Add(-defaultClockSkew)) || issuedAt.After(now.Add(defaultClockSkew)) {
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH, "DeviceHello is outside the accepted time window", nil)
	}
	publicKey := ed25519.PublicKey(hello.GetDevicePublicKey())
	if len(publicKey) != ed25519.PublicKeySize || len(hello.GetServerNonce()) != authNonceBytes || len(hello.GetSignature()) != ed25519.SignatureSize {
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH, "DeviceHello key, nonce or signature is malformed", nil)
	}
	calculatedFingerprint := Fingerprint(publicKey)
	if subtle.ConstantTimeCompare([]byte(calculatedFingerprint), []byte(hello.GetDeviceFingerprint())) != 1 ||
		subtle.ConstantTimeCompare([]byte(calculatedFingerprint), []byte(strings.TrimSpace(expectedFingerprint))) != 1 {
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH, "DeviceHello fingerprint does not match endpoint pin", nil)
	}
	if strings.TrimSpace(hello.GetDeviceId()) != strings.TrimSpace(expectedDeviceID) {
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH, "DeviceHello device id does not match endpoint", nil)
	}
	helloBinding, err := channelBindingFromProto(hello.GetChannelBinding())
	if err != nil || !channelBindingsEqual(helloBinding, expectedBinding) {
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH, "DeviceHello is not bound to the current transport", err)
	}
	signingBytes, err := DeviceHelloSigningBytes(authSessionID, hello)
	if err != nil || !ed25519.Verify(publicKey, signingBytes, hello.GetSignature()) {
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH, "DeviceHello signature is invalid", err)
	}
	return nil
}

func (handshake ServerHandshake) reject(ctx context.Context, connection transport.Transport, authSessionID string, cause error) error {
	code := HandshakeCodeOf(cause)
	rejected := &remoteauthpb.AuthEnvelope{
		Protocol: AuthProtocol, Version: AuthVersion, AuthSessionId: authSessionID,
		Payload: &remoteauthpb.AuthEnvelope_CapabilityRejected{CapabilityRejected: &remoteauthpb.CapabilityRejected{
			Code: code, Message: rejectionMessage(code),
		}},
	}
	if err := sendAuthEnvelope(ctx, connection, rejected); err != nil {
		return newHandshakeError(code, "authorization failed and rejection could not be sent", errors.Join(cause, err))
	}
	return cause
}

func sendAuthEnvelope(ctx context.Context, connection transport.Transport, envelope *remoteauthpb.AuthEnvelope) error {
	frame, err := MarshalAuthEnvelope(envelope)
	if err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		_ = connection.Close()
		return err
	}
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- connection.Send(frame)
	}()
	select {
	case <-ctx.Done():
		// Transport.Send 没有 context；未认证握手拥有当前 transport，取消时必须关闭实际连接，
		// 让 socket/DataChannel backpressure 退出，不能把阻塞写留给上层锁或 timeout 兜底。
		_ = connection.Close()
		return ctx.Err()
	case sendErr := <-resultCh:
		if ctxErr := ctx.Err(); ctxErr != nil {
			_ = connection.Close()
			return ctxErr
		}
		if sendErr != nil {
			return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "send remote auth frame", sendErr)
		}
		return nil
	}
}

func receiveAuthEnvelope(ctx context.Context, connection transport.Transport) (*remoteauthpb.AuthEnvelope, error) {
	type result struct {
		frame []byte
		err   error
	}
	resultCh := make(chan result, 1)
	go func() {
		frame, err := connection.Recv()
		resultCh <- result{frame: frame, err: err}
	}()
	select {
	case <-ctx.Done():
		_ = connection.Close()
		return nil, ctx.Err()
	case received := <-resultCh:
		if received.err != nil {
			return nil, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "receive remote auth frame", received.err)
		}
		return UnmarshalAuthEnvelope(received.frame)
	}
}

func scopeSummary(scope Scope) (*remoteauthpb.ScopeSummary, error) {
	summary := &remoteauthpb.ScopeSummary{ManageClientAccess: scope.ManageClientAccess}
	switch {
	case scope.AllowDaemon && scope.TerminalID == "" && !scope.MachineEventsOnly:
		summary.Kind = remoteauthpb.ScopeKind_SCOPE_KIND_DAEMON
	case !scope.AllowDaemon && scope.TerminalID != "" && !scope.MachineEventsOnly:
		summary.Kind = remoteauthpb.ScopeKind_SCOPE_KIND_TERMINAL
		summary.TerminalId = scope.TerminalID
	case !scope.AllowDaemon && scope.TerminalID == "" && scope.MachineEventsOnly:
		summary.Kind = remoteauthpb.ScopeKind_SCOPE_KIND_MACHINE_EVENTS
	default:
		return nil, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SCOPE_INVALID, "capability scope cannot map to core-v2", ErrGrantScopeInvalid)
	}
	return summary, nil
}

func scopeSummaryMatches(summary *remoteauthpb.ScopeSummary, scope Scope) bool {
	want, err := scopeSummary(scope)
	return err == nil && summary != nil && summary.GetKind() == want.GetKind() && summary.GetTerminalId() == want.GetTerminalId() && summary.GetManageClientAccess() == want.GetManageClientAccess()
}

func mapGrantError(err error) *HandshakeError {
	switch {
	case errors.Is(err, ErrGrantExpired), errors.Is(err, ErrGrantNotActive):
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_EXPIRED, "capability time window is invalid", err)
	case errors.Is(err, ErrGrantRevoked):
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_REVOKED, "capability is revoked", err)
	case errors.Is(err, ErrGrantFingerprintMismatch):
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH, "capability issuer fingerprint does not match endpoint pin", err)
	case errors.Is(err, ErrGrantSubjectMismatch):
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH, "capability subject key is invalid", err)
	case errors.Is(err, ErrGrantScopeInvalid):
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SCOPE_INVALID, "capability scope is invalid", err)
	default:
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_INVALID, "capability verification failed", err)
	}
}

func mapPairingError(err error) *HandshakeError {
	switch {
	case errors.Is(err, ErrPairingTicketExpired):
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PAIRING_TICKET_EXPIRED, "pairing ticket time window is invalid", err)
	case errors.Is(err, ErrPairingTicketConsumed):
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PAIRING_TICKET_CONSUMED, "pairing ticket is bound to another client", err)
	case errors.Is(err, ErrGrantScopeInvalid):
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SCOPE_INVALID, "pairing scope ceiling is invalid", err)
	default:
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PAIRING_TICKET_INVALID, "pairing ticket verification failed", err)
	}
}

func validateResultSession(result *remoteauthpb.AuthEnvelope, expected string) error {
	if result == nil || subtle.ConstantTimeCompare([]byte(result.GetAuthSessionId()), []byte(expected)) != 1 {
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_REPLAYED, "daemon changed auth session", nil)
	}
	return nil
}

func rejectionError(rejected *remoteauthpb.CapabilityRejected) error {
	code := rejected.GetCode()
	if code == remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_UNSPECIFIED {
		code = remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL
	}
	return newHandshakeError(code, rejected.GetMessage(), nil)
}

func randomBytes(reader io.Reader, size int) ([]byte, error) {
	if reader == nil {
		reader = rand.Reader
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (handshake ClientHandshake) random() io.Reader {
	if handshake.Random != nil {
		return handshake.Random
	}
	return rand.Reader
}

func (handshake ClientHandshake) now() time.Time {
	if handshake.Now != nil {
		return handshake.Now().UTC()
	}
	return time.Now().UTC()
}

func (handshake ClientPairingHandshake) random() io.Reader {
	if handshake.Random != nil {
		return handshake.Random
	}
	return rand.Reader
}

func (handshake ClientPairingHandshake) now() time.Time {
	if handshake.Now != nil {
		return handshake.Now().UTC()
	}
	return time.Now().UTC()
}

func (handshake ServerHandshake) random() io.Reader {
	if handshake.Random != nil {
		return handshake.Random
	}
	return rand.Reader
}

func (handshake ServerHandshake) now() time.Time {
	if handshake.Now != nil {
		return handshake.Now().UTC()
	}
	return time.Now().UTC()
}

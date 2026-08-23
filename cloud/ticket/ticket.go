// Package ticket 实现 Controller binding 与 Cloud Gateway stream proof 的唯一签名边界。
package ticket

import (
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/shared/remoteauth"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	daemonBindingDomain    = "anytty.cloud.daemon-binding.v2\x00"
	agentProofDomain       = "anytty.cloud.agent-hello-proof.v2\x00"
	cloudRouteGrantDomain  = "anytty.cloud.route-grant.v1\x00"
	clientRouteProofDomain = "anytty.cloud.client-route-proof.v1\x00"
	clientHelloProofDomain = "anytty.cloud.gateway-client-hello-proof.v2\x00"

	EdgeChallengeNonceSize = 32
	EdgeChallengeLifetime  = 10 * time.Second
	// ClockSkewTolerance is the shared validity-window tolerance for Cloud protocol peers.
	ClockSkewTolerance = remoteauth.ClockSkewTolerance
	// EdgeChallengeClockSkew is retained for callers that name the challenge-specific boundary.
	EdgeChallengeClockSkew = ClockSkewTolerance
	// MaxKeyBundleTTL 限制 Edge 可离线信任 Controller binding keyset 的最长窗口。
	MaxKeyBundleTTL = 24 * time.Hour
)

var verificationKeyIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// KeySet 是 Edge 当前有效 KeyBundle 中的 daemon binding 公钥集合。
type KeySet map[string]ed25519.PublicKey

// ValidateKeyBundle 严格校验 revision、有效期和规范化 Ed25519 keyset，并返回只读验签快照。
func ValidateKeyBundle(bundle *cloudv1.KeyBundle) (KeySet, error) {
	if bundle == nil || len(bundle.ProtoReflect().GetUnknown()) != 0 || bundle.GetRevision() == 0 ||
		bundle.GetIssuedAt() == nil || bundle.GetExpiresAt() == nil || bundle.GetIssuedAt().CheckValid() != nil || bundle.GetExpiresAt().CheckValid() != nil {
		return nil, errors.New("binding key bundle is invalid")
	}
	issuedAt, expiresAt := bundle.GetIssuedAt().AsTime(), bundle.GetExpiresAt().AsTime()
	if !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > MaxKeyBundleTTL {
		return nil, errors.New("binding key bundle validity window is invalid")
	}
	return parseVerificationKeys(bundle.GetKeys())
}

// KeyBundleUsableAt 要求 bundle 在统一跨设备时钟容差内已生效且尚未过期。
func KeyBundleUsableAt(bundle *cloudv1.KeyBundle, now time.Time) bool {
	if now.IsZero() {
		return false
	}
	if _, err := ValidateKeyBundle(bundle); err != nil {
		return false
	}
	now = now.UTC()
	return !bundle.GetIssuedAt().AsTime().After(now.Add(remoteauth.ClockSkewTolerance)) &&
		bundle.GetExpiresAt().AsTime().After(now.Add(-remoteauth.ClockSkewTolerance))
}

// CanonicalKeyBundle 深拷贝并按 key ID 排序，确保持久 payload 和 keyset 比较稳定。
func CanonicalKeyBundle(bundle *cloudv1.KeyBundle) (*cloudv1.KeyBundle, KeySet, error) {
	keys, err := ValidateKeyBundle(bundle)
	if err != nil {
		return nil, nil, err
	}
	canonical := proto.Clone(bundle).(*cloudv1.KeyBundle)
	sort.Slice(canonical.Keys, func(i, j int) bool { return canonical.Keys[i].GetKeyId() < canonical.Keys[j].GetKeyId() })
	return canonical, keys, nil
}

// SameKeySet 比较 KeyBundle 的规范化 key ID、算法和公钥，不比较 revision 或有效期。
func SameKeySet(first, second *cloudv1.KeyBundle) bool {
	if first == nil || second == nil || len(first.GetKeys()) != len(second.GetKeys()) {
		return false
	}
	left, _, leftErr := CanonicalKeyBundle(first)
	right, _, rightErr := CanonicalKeyBundle(second)
	if leftErr != nil || rightErr != nil {
		return false
	}
	for index := range left.GetKeys() {
		if !proto.Equal(left.GetKeys()[index], right.GetKeys()[index]) {
			return false
		}
	}
	return true
}

func parseVerificationKeys(keys []*cloudv1.VerificationKey) (KeySet, error) {
	result := make(KeySet, len(keys))
	publicKeys := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if key == nil || len(key.ProtoReflect().GetUnknown()) != 0 {
			return nil, errors.New("binding verification key is invalid")
		}
		id := key.GetKeyId()
		if id != strings.TrimSpace(id) || !verificationKeyIDPattern.MatchString(id) || key.GetAlgorithm() != "Ed25519" || len(key.GetPublicKey()) != ed25519.PublicKeySize {
			return nil, errors.New("binding verification key is invalid")
		}
		if _, exists := result[id]; exists {
			return nil, errors.New("binding verification key ID is duplicated")
		}
		encodedPublicKey := string(key.GetPublicKey())
		if _, exists := publicKeys[encodedPublicKey]; exists {
			return nil, errors.New("binding verification public key is duplicated")
		}
		publicKeys[encodedPublicKey] = struct{}{}
		result[id] = append(ed25519.PublicKey(nil), key.GetPublicKey()...)
	}
	if len(result) == 0 {
		return nil, errors.New("binding verification key set is empty")
	}
	return result, nil
}

// SignDaemonBinding 对确定性 DaemonBindingClaims 做 domain-separated 签名。
func SignDaemonBinding(keyID string, privateKey ed25519.PrivateKey, claims *cloudv1.DaemonBindingClaims) (*cloudv1.SignedEnvelope, error) {
	keyID = strings.TrimSpace(keyID)
	if keyID == "" || len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("daemon binding signer is invalid")
	}
	if err := validateDaemonBinding(claims, time.Time{}, 0); err != nil {
		return nil, err
	}
	payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(claims)
	if err != nil {
		return nil, err
	}
	return &cloudv1.SignedEnvelope{KeyId: keyID, Payload: payload, Signature: ed25519.Sign(privateKey, signingBytes(daemonBindingDomain, payload))}, nil
}

// VerifyDaemonBinding 在 Edge 本地验签并校验 target、期限和时钟偏差。
func VerifyDaemonBinding(envelope *cloudv1.SignedEnvelope, keys KeySet, edgeID string, now time.Time, skew time.Duration) (*cloudv1.DaemonBindingClaims, error) {
	if envelope == nil || len(envelope.GetSignature()) != ed25519.SignatureSize {
		return nil, errors.New("daemon binding envelope is invalid")
	}
	publicKey := keys[strings.TrimSpace(envelope.GetKeyId())]
	if len(publicKey) != ed25519.PublicKeySize || !ed25519.Verify(publicKey, signingBytes(daemonBindingDomain, envelope.GetPayload()), envelope.GetSignature()) {
		return nil, errors.New("daemon binding signature is invalid")
	}
	claims := &cloudv1.DaemonBindingClaims{}
	if err := proto.Unmarshal(envelope.GetPayload(), claims); err != nil {
		return nil, errors.New("daemon binding payload is invalid")
	}
	if err := validateDaemonBinding(claims, now, skew); err != nil {
		return nil, err
	}
	if strings.TrimSpace(claims.GetEdgeId()) != strings.TrimSpace(edgeID) {
		return nil, errors.New("daemon binding targets another Edge")
	}
	return claims, nil
}

// ValidateEdgeChallenge 验证 Edge 单次 challenge 的目标、结构、固定 10 秒窗口和统一时钟容差。
func ValidateEdgeChallenge(challenge *cloudv1.EdgeChallenge, target cloudv1.EdgeChallengeTarget, now time.Time) error {
	if challenge == nil || len(challenge.ProtoReflect().GetUnknown()) != 0 || len(challenge.GetNonce()) != EdgeChallengeNonceSize ||
		strings.TrimSpace(challenge.GetEdgeId()) == "" || strings.TrimSpace(challenge.GetEdgeBootId()) == "" || strings.TrimSpace(challenge.GetStreamId()) == "" ||
		challenge.GetTarget() != target || target == cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_UNSPECIFIED ||
		challenge.GetIssuedAt() == nil || challenge.GetExpiresAt() == nil || challenge.GetIssuedAt().CheckValid() != nil || challenge.GetExpiresAt().CheckValid() != nil || now.IsZero() {
		return errors.New("Edge challenge is invalid")
	}
	issuedAt, expiresAt := challenge.GetIssuedAt().AsTime(), challenge.GetExpiresAt().AsTime()
	now = now.UTC()
	if expiresAt.Sub(issuedAt) != EdgeChallengeLifetime {
		return errors.New("Edge challenge lifetime is invalid")
	}
	if issuedAt.After(now.Add(EdgeChallengeClockSkew)) {
		return errors.New("Edge challenge was issued in the future")
	}
	if !expiresAt.After(now.Add(-EdgeChallengeClockSkew)) {
		return errors.New("Edge challenge expired")
	}
	return nil
}

// SignAgentHelloProof 把 v2 AgentHello transcript 绑定到 DeviceIdentity。
func SignAgentHelloProof(identity remoteauth.Identity, challenge *cloudv1.EdgeChallenge, event *cloudv1.AgentEvent, now time.Time) ([]byte, error) {
	if err := identity.Validate(); err != nil {
		return nil, err
	}
	canonical, err := AgentHelloProofBytes(challenge, event, now)
	if err != nil {
		return nil, err
	}
	return ed25519.Sign(identity.PrivateKey, canonical), nil
}

// VerifyAgentHelloProof 证明 AgentHello 持有 binding 内 DeviceIdentity 私钥。
func VerifyAgentHelloProof(publicKey, proof []byte, challenge *cloudv1.EdgeChallenge, event *cloudv1.AgentEvent, now time.Time) error {
	if len(publicKey) != ed25519.PublicKeySize || len(proof) != ed25519.SignatureSize {
		return errors.New("AgentHello proof key or signature is invalid")
	}
	canonical, err := AgentHelloProofBytes(challenge, event, now)
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), canonical, proof) {
		return errors.New("AgentHello DeviceIdentity proof is invalid")
	}
	return nil
}

// SignCloudRouteGrant 使用 daemon DeviceIdentity 对只含发现信息的 grant 做 domain-separated 签名。
func SignCloudRouteGrant(identity remoteauth.Identity, claims *cloudv1.CloudRouteGrantClaims) (*cloudv1.SignedEnvelope, error) {
	if err := identity.Validate(); err != nil {
		return nil, err
	}
	if err := validateCloudRouteGrant(claims, time.Time{}); err != nil {
		return nil, err
	}
	payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(claims)
	if err != nil {
		return nil, err
	}
	return &cloudv1.SignedEnvelope{KeyId: identity.Fingerprint, Payload: payload, Signature: ed25519.Sign(identity.PrivateKey, signingBytes(cloudRouteGrantDomain, payload))}, nil
}

// VerifyCloudRouteGrant 使用 Controller 持久化的 daemon 公钥验签，不读取或保存在线拓扑。
func VerifyCloudRouteGrant(envelope *cloudv1.SignedEnvelope, daemonPublicKey ed25519.PublicKey, expectedDaemonID string, now time.Time) (*cloudv1.CloudRouteGrantClaims, error) {
	if envelope == nil || len(daemonPublicKey) != ed25519.PublicKeySize || len(envelope.GetSignature()) != ed25519.SignatureSize ||
		strings.TrimSpace(envelope.GetKeyId()) != remoteauth.Fingerprint(daemonPublicKey) {
		return nil, errors.New("CloudRouteGrant signature is invalid")
	}
	if !ed25519.Verify(daemonPublicKey, signingBytes(cloudRouteGrantDomain, envelope.GetPayload()), envelope.GetSignature()) {
		return nil, errors.New("CloudRouteGrant signature is invalid")
	}
	claims := &cloudv1.CloudRouteGrantClaims{}
	if err := proto.Unmarshal(envelope.GetPayload(), claims); err != nil {
		return nil, errors.New("CloudRouteGrant payload is invalid")
	}
	if err := validateCloudRouteGrant(claims, now); err != nil {
		return nil, err
	}
	if claims.GetDaemonId() != strings.TrimSpace(expectedDaemonID) {
		return nil, errors.New("CloudRouteGrant targets another daemon")
	}
	return claims, nil
}

// ClientRouteProofBytes 返回 ClientAccessIdentity 对 Controller challenge 的 canonical 签名输入。
func ClientRouteProofBytes(challengeID string, challenge []byte, grant *cloudv1.SignedEnvelope, requestID string) ([]byte, error) {
	if strings.TrimSpace(challengeID) == "" || len(challenge) == 0 || grant == nil || strings.TrimSpace(requestID) == "" {
		return nil, errors.New("client route proof input is incomplete")
	}
	digest := sha256.Sum256(grant.GetPayload())
	payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(&cloudv1.ClientRouteProofInput{
		ChallengeId: strings.TrimSpace(challengeID), Challenge: append([]byte(nil), challenge...), GrantPayloadSha256: digest[:], RequestId: strings.TrimSpace(requestID),
	})
	if err != nil {
		return nil, err
	}
	return signingBytes(clientRouteProofDomain, payload), nil
}

// VerifyClientRouteProof 证明发起解析的客户端持有 grant 内 ClientAccessIdentity 私钥。
func VerifyClientRouteProof(publicKey []byte, signature []byte, canonical []byte) error {
	if len(publicKey) != ed25519.PublicKeySize || len(signature) != ed25519.SignatureSize || len(canonical) == 0 || !ed25519.Verify(ed25519.PublicKey(publicKey), canonical, signature) {
		return errors.New("client route proof is invalid")
	}
	return nil
}

// AgentHelloProofBytes 返回覆盖 challenge、完整 binding 摘要和全部 AgentHello 安全字段的 canonical 输入。
func AgentHelloProofBytes(challenge *cloudv1.EdgeChallenge, event *cloudv1.AgentEvent, now time.Time) ([]byte, error) {
	if err := ValidateEdgeChallenge(challenge, cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_AGENT_GATEWAY, now); err != nil {
		return nil, err
	}
	if event == nil || event.GetHello() == nil || len(event.ProtoReflect().GetUnknown()) != 0 || len(event.GetHello().ProtoReflect().GetUnknown()) != 0 || event.GetProtocolVersion() == 0 || strings.TrimSpace(event.GetMessageId()) == "" ||
		strings.TrimSpace(event.GetSenderId()) == "" || strings.TrimSpace(event.GetBootId()) == "" || strings.TrimSpace(event.GetConnectionId()) == "" || event.GetStreamSeq() == 0 ||
		event.GetSentAt() == nil || event.GetSentAt().CheckValid() != nil || event.GetHello().GetDaemonBinding() == nil ||
		strings.TrimSpace(event.GetHello().GetSoftwareVersion()) == "" || event.GetHello().GetAttemptGeneration() == 0 {
		return nil, errors.New("AgentHello proof input is incomplete")
	}
	bindingDigest, err := deterministicDigest(event.GetHello().GetDaemonBinding())
	if err != nil {
		return nil, err
	}
	payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(&cloudv1.AgentHelloProofInput{
		BindingEnvelopeSha256: bindingDigest,
		DaemonId:              event.GetSenderId(),
		DaemonBootId:          event.GetBootId(),
		DaemonSessionId:       event.GetConnectionId(),
		Challenge:             proto.Clone(challenge).(*cloudv1.EdgeChallenge),
		ProtocolVersion:       event.GetProtocolVersion(),
		MessageId:             event.GetMessageId(),
		StreamSeq:             event.GetStreamSeq(),
		SentAt:                proto.Clone(event.GetSentAt()).(*timestamppb.Timestamp),
		SoftwareVersion:       event.GetHello().GetSoftwareVersion(),
		AttemptGeneration:     event.GetHello().GetAttemptGeneration(),
	})
	if err != nil {
		return nil, err
	}
	return signingBytes(agentProofDomain, payload), nil
}

// ClientHelloProofBytes 对 capability 与 pairing 使用同一个 v2 transcript，不保留旧 proof 分支。
func ClientHelloProofBytes(challenge *cloudv1.EdgeChallenge, event *cloudv1.ClientSignal, now time.Time) ([]byte, error) {
	if err := ValidateEdgeChallenge(challenge, cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY, now); err != nil {
		return nil, err
	}
	if event == nil || event.GetHello() == nil || len(event.ProtoReflect().GetUnknown()) != 0 || len(event.GetHello().ProtoReflect().GetUnknown()) != 0 || event.GetProtocolVersion() == 0 || strings.TrimSpace(event.GetMessageId()) == "" ||
		strings.TrimSpace(event.GetSenderId()) == "" || strings.TrimSpace(event.GetBootId()) == "" || strings.TrimSpace(event.GetConnectionId()) == "" || event.GetStreamSeq() == 0 ||
		event.GetSentAt() == nil || event.GetSentAt().CheckValid() != nil || len(event.GetHello().GetClientPublicKey()) != ed25519.PublicKeySize ||
		event.GetHello().GetProduct() < cloudv1.ClientProduct_CLIENT_PRODUCT_TUI || event.GetHello().GetProduct() > cloudv1.ClientProduct_CLIENT_PRODUCT_DESKTOP_GUI ||
		strings.TrimSpace(event.GetHello().GetSoftwareVersion()) == "" || event.GetHello().GetAttemptGeneration() == 0 ||
		event.GetHello().GetRelayPreference() < cloudv1.RelayPreference_RELAY_PREFERENCE_AUTO || event.GetHello().GetRelayPreference() > cloudv1.RelayPreference_RELAY_PREFERENCE_RELAY_ONLY {
		return nil, errors.New("ClientHello proof input is incomplete")
	}
	var authorization proto.Message
	var accessMode cloudv1.CloudClientAccessMode
	switch value := event.GetHello().GetAuthorization().(type) {
	case *cloudv1.ClientHello_CloudRouteGrant:
		if value.CloudRouteGrant == nil {
			return nil, errors.New("ClientHello authorization is missing")
		}
		authorization = value.CloudRouteGrant
		accessMode = cloudv1.CloudClientAccessMode_CLOUD_CLIENT_ACCESS_MODE_CAPABILITY
	case *cloudv1.ClientHello_PairingAdmission:
		if value.PairingAdmission == nil {
			return nil, errors.New("ClientHello authorization is missing")
		}
		authorization = value.PairingAdmission
		accessMode = cloudv1.CloudClientAccessMode_CLOUD_CLIENT_ACCESS_MODE_PAIRING
	default:
		return nil, errors.New("ClientHello authorization is missing")
	}
	authorizationDigest, err := deterministicDigest(authorization)
	if err != nil {
		return nil, err
	}
	payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(&cloudv1.GatewayClientHelloProofInput{
		Challenge:           proto.Clone(challenge).(*cloudv1.EdgeChallenge),
		AuthorizationSha256: authorizationDigest,
		AccessMode:          accessMode,
		ProtocolVersion:     event.GetProtocolVersion(),
		MessageId:           event.GetMessageId(),
		ClientId:            event.GetSenderId(),
		ClientBootId:        event.GetBootId(),
		ClientSessionId:     event.GetConnectionId(),
		StreamSeq:           event.GetStreamSeq(),
		SentAt:              proto.Clone(event.GetSentAt()).(*timestamppb.Timestamp),
		ClientPublicKey:     append([]byte(nil), event.GetHello().GetClientPublicKey()...),
		Product:             event.GetHello().GetProduct(),
		SoftwareVersion:     event.GetHello().GetSoftwareVersion(),
		AttemptGeneration:   event.GetHello().GetAttemptGeneration(),
		RelayPreference:     event.GetHello().GetRelayPreference(),
		PresenceProbe:       event.GetHello().GetPresenceProbe(),
	})
	if err != nil {
		return nil, err
	}
	return signingBytes(clientHelloProofDomain, payload), nil
}

// VerifyClientHelloProof 证明 ClientHello 持有 authorization 绑定的 ClientAccessIdentity 私钥。
func VerifyClientHelloProof(publicKey, proof []byte, challenge *cloudv1.EdgeChallenge, event *cloudv1.ClientSignal, now time.Time) error {
	canonical, err := ClientHelloProofBytes(challenge, event, now)
	if err != nil {
		return err
	}
	if len(publicKey) != ed25519.PublicKeySize || len(proof) != ed25519.SignatureSize || !ed25519.Verify(ed25519.PublicKey(publicKey), canonical, proof) {
		return errors.New("ClientHello proof is invalid")
	}
	return nil
}

func deterministicDigest(message proto.Message) ([]byte, error) {
	payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(message)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(payload)
	return digest[:], nil
}

func validateDaemonBinding(claims *cloudv1.DaemonBindingClaims, now time.Time, skew time.Duration) error {
	if claims == nil || strings.TrimSpace(claims.GetBindingId()) == "" || strings.TrimSpace(claims.GetDaemonId()) == "" ||
		strings.TrimSpace(claims.GetAccountId()) == "" || strings.TrimSpace(claims.GetEdgeId()) == "" || strings.TrimSpace(claims.GetDeviceId()) == "" ||
		len(claims.GetDevicePublicKey()) != ed25519.PublicKeySize || len(claims.GetEdgeLocatorSha256()) != sha256.Size || claims.GetIssuedAt() == nil || claims.GetExpiresAt() == nil ||
		claims.GetIssuedAt().CheckValid() != nil || claims.GetExpiresAt().CheckValid() != nil || !claims.GetExpiresAt().AsTime().After(claims.GetIssuedAt().AsTime()) {
		return errors.New("daemon binding claims are incomplete")
	}
	if len(claims.GetCapabilities()) == 0 {
		return errors.New("daemon binding has no capability")
	}
	if !now.IsZero() {
		now = now.UTC()
		if claims.GetIssuedAt().AsTime().After(now.Add(skew)) || !claims.GetExpiresAt().AsTime().After(now.Add(-skew)) {
			return fmt.Errorf("daemon binding is outside its validity window")
		}
	}
	return nil
}

func validateCloudRouteGrant(claims *cloudv1.CloudRouteGrantClaims, now time.Time) error {
	if claims == nil || strings.TrimSpace(claims.GetGrantId()) == "" || strings.TrimSpace(claims.GetDaemonId()) == "" || len(claims.GetClientPublicKey()) != ed25519.PublicKeySize ||
		!validCloudRouteGrantProduct(claims.GetProduct()) || claims.GetIssuedAt() == nil || claims.GetIssuedAt().CheckValid() != nil ||
		(claims.GetExpiresAt() != nil && (claims.GetExpiresAt().CheckValid() != nil || !claims.GetExpiresAt().AsTime().After(claims.GetIssuedAt().AsTime()) || claims.GetExpiresAt().AsTime().Sub(claims.GetIssuedAt().AsTime()) > 365*24*time.Hour)) {
		return errors.New("CloudRouteGrant claims are incomplete")
	}
	if !now.IsZero() && (claims.GetIssuedAt().AsTime().After(now.UTC().Add(remoteauth.ClockSkewTolerance)) ||
		(claims.GetExpiresAt() != nil && !claims.GetExpiresAt().AsTime().After(now.UTC().Add(-remoteauth.ClockSkewTolerance)))) {
		return errors.New("CloudRouteGrant is outside its validity window")
	}
	return nil
}

func validCloudRouteGrantProduct(product cloudv1.ClientProduct) bool {
	return product == cloudv1.ClientProduct_CLIENT_PRODUCT_UNSPECIFIED ||
		(product >= cloudv1.ClientProduct_CLIENT_PRODUCT_TUI && product <= cloudv1.ClientProduct_CLIENT_PRODUCT_DESKTOP_GUI)
}

func signingBytes(domain string, payload []byte) []byte {
	result := make([]byte, 0, len(domain)+len(payload))
	result = append(result, domain...)
	return append(result, payload...)
}

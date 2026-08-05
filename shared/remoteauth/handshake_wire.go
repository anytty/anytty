package remoteauth

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/anytty/anytty/proto/remoteauthpb"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	// AuthProtocol 是 direct TLS/DTLS DataChannel 切换到 anytty protocol 前的独立协议标识。
	AuthProtocol = "anytty-remote-auth"
	// AuthVersion 是 client-bound CapabilityGrant 与 PairingExchange canonical contract 的版本。
	// v1 bearer/HMAC envelope 不再接受，也没有版本回退。
	AuthVersion uint32 = 2
	// MaxAuthFrameSize 限制单条授权 envelope，避免 grant、ticket 或恶意 protobuf 无界占用内存。
	MaxAuthFrameSize = 64 * 1024
)

var authFrameMagic = [4]byte{'T', 'X', 'R', 'A'}

// ChannelBinding 是 transport adapter 对当前安全 channel 的 SHA-256 证明。
// Kind 与 Hash 共同进入 daemon/client signature；route 地址、SDP 或 Cloud signaling metadata 不能构造或替代该值。
type ChannelBinding struct {
	Kind remoteauthpb.ChannelBindingKind
	Hash [sha256.Size]byte
}

// Validate 拒绝未知 binding kind 或全零 hash。
// 失败表示 transport adapter 没有从实际 TLS/DTLS/local Unix channel 提供可信 binding，调用方必须关闭当前 transport。
func (binding ChannelBinding) Validate() error {
	switch binding.Kind {
	case remoteauthpb.ChannelBindingKind_CHANNEL_BINDING_KIND_DIRECT_TLS,
		remoteauthpb.ChannelBindingKind_CHANNEL_BINDING_KIND_DTLS,
		remoteauthpb.ChannelBindingKind_CHANNEL_BINDING_KIND_LOCAL_UNIX:
	default:
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "unsupported channel binding kind", nil)
	}
	var zero [sha256.Size]byte
	if subtle.ConstantTimeCompare(binding.Hash[:], zero[:]) == 1 {
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "channel binding hash is empty", nil)
	}
	return nil
}

// DirectTLSChannelBinding 从实际 TLS peer certificate DER bytes 生成 direct TLS binding。
// certificateDER 必须来自当前 tls.ConnectionState，不能使用 server_name、配置 pin 或证书文本替代。
func DirectTLSChannelBinding(certificateDER []byte) (ChannelBinding, error) {
	if len(certificateDER) == 0 {
		return ChannelBinding{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "direct TLS certificate is empty", nil)
	}
	return ChannelBinding{Kind: remoteauthpb.ChannelBindingKind_CHANNEL_BINDING_KIND_DIRECT_TLS, Hash: sha256.Sum256(certificateDER)}, nil
}

// DTLSChannelBinding 从 Pion 读取的实际远端 DTLS certificate fingerprint 生成 canonical binding。
// 输入只接受 SHA-256 fingerprint；解析后使用原始 32-byte digest，避免字符串大小写或冒号格式形成跨平台分叉。
func DTLSChannelBinding(fingerprint string) (ChannelBinding, error) {
	normalized, err := NormalizeDTLSCertificateFingerprint(fingerprint)
	if err != nil {
		return ChannelBinding{}, err
	}
	raw, err := hex.DecodeString(strings.ReplaceAll(strings.TrimPrefix(normalized, "sha-256:"), ":", ""))
	if err != nil || len(raw) != sha256.Size {
		return ChannelBinding{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "daemon DTLS fingerprint is malformed", err)
	}
	var hash [sha256.Size]byte
	copy(hash[:], raw)
	return ChannelBinding{Kind: remoteauthpb.ChannelBindingKind_CHANNEL_BINDING_KIND_DTLS, Hash: hash}, nil
}

// LocalUnixChannelBinding 为 owner-only pairing socket 提供本地 transport domain binding。
// 它不把 Unix socket 提升为网络 TLS；该 kind 只允许本机 PairingExchange harness/CLI，不能用于普通 remote capability session。
func LocalUnixChannelBinding(socketPath string) (ChannelBinding, error) {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return ChannelBinding{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "local Unix channel path is empty", nil)
	}
	canonical := filepath.Clean(socketPath)
	return ChannelBinding{
		Kind: remoteauthpb.ChannelBindingKind_CHANNEL_BINDING_KIND_LOCAL_UNIX,
		Hash: sha256.Sum256([]byte("anytty-local-unix-binding-v1\x00" + canonical)),
	}, nil
}

// HandshakeError 是端到端 remote auth 状态机的稳定错误。
// Code 可用于 endpoint 局部状态；Detail 只留在当前公开进程，发送给对端时会替换为固定脱敏消息。
type HandshakeError struct {
	Code   remoteauthpb.AuthErrorCode
	Detail string
	Cause  error
}

// Error 返回稳定错误码和本地诊断，不包含原始 grant、ticket 或私钥。
func (err *HandshakeError) Error() string {
	if err == nil {
		return "<nil>"
	}
	if err.Detail == "" {
		return err.Code.String()
	}
	return fmt.Sprintf("%s: %s", err.Code.String(), err.Detail)
}

// Unwrap 返回本地底层错误，供调用方用 errors.Is 区分 expiry、revoke、ticket consume 等 daemon-local truth。
func (err *HandshakeError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

// HandshakeCodeOf 返回错误链中的稳定 remote auth code。
// 非状态机错误映射为 INTERNAL；调用方不得据此 fallback 到 bearer、其他 endpoint 或旧 transport。
func HandshakeCodeOf(err error) remoteauthpb.AuthErrorCode {
	var handshakeErr *HandshakeError
	if errors.As(err, &handshakeErr) {
		return handshakeErr.Code
	}
	return remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_INTERNAL
}

// MarshalAuthEnvelope 使用 `TXRA` magic 和 deterministic protobuf 编码一条 pre-protocol 授权帧。
// 输入必须是 v2、单一 payload 且无 unknown field；成功后的 bytes 可以作为一条完整 transport message 发送。
func MarshalAuthEnvelope(envelope *remoteauthpb.AuthEnvelope) ([]byte, error) {
	if err := validateAuthEnvelope(envelope); err != nil {
		return nil, err
	}
	payload, err := deterministicMarshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal remote auth envelope: %w", err)
	}
	if len(payload) > MaxAuthFrameSize-len(authFrameMagic) {
		return nil, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "remote auth frame exceeds limit", nil)
	}
	frame := make([]byte, len(authFrameMagic)+len(payload))
	copy(frame, authFrameMagic[:])
	copy(frame[len(authFrameMagic):], payload)
	return frame, nil
}

// UnmarshalAuthEnvelope 解码并严格校验一条 v2 pre-protocol 授权帧。
// anytty protocol frame、错误 magic、v1 envelope、unknown field 或多义 payload都会返回 PROTOCOL，不会尝试旧 bearer 格式。
func UnmarshalAuthEnvelope(frame []byte) (*remoteauthpb.AuthEnvelope, error) {
	if len(frame) <= len(authFrameMagic) || len(frame) > MaxAuthFrameSize || subtle.ConstantTimeCompare(frame[:min(len(frame), len(authFrameMagic))], authFrameMagic[:]) != 1 {
		return nil, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "invalid remote auth frame", nil)
	}
	envelope := &remoteauthpb.AuthEnvelope{}
	payload := frame[len(authFrameMagic):]
	if err := validateSingleAuthPayloadWire(payload); err != nil {
		return nil, err
	}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, envelope); err != nil {
		return nil, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "decode remote auth envelope", err)
	}
	if err := validateAuthEnvelope(envelope); err != nil {
		return nil, err
	}
	return envelope, nil
}

// DeviceHelloSigningBytes 返回 DeviceIdentity Ed25519 签名使用的跨平台 canonical bytes。
// signature 字段不会进入输入；channel binding kind/hash 与全部 identity 字段由 deterministic protobuf 固定。
func DeviceHelloSigningBytes(authSessionID string, hello *remoteauthpb.DeviceHello) ([]byte, error) {
	if hello == nil {
		return nil, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "device hello is nil", nil)
	}
	binding, err := channelBindingFromProto(hello.GetChannelBinding())
	if err != nil {
		return nil, err
	}
	input := &remoteauthpb.DeviceHelloSignatureInput{
		Protocol: AuthProtocol, Version: AuthVersion, AuthSessionId: strings.TrimSpace(authSessionID),
		DeviceId: strings.TrimSpace(hello.GetDeviceId()), DevicePublicKey: append([]byte(nil), hello.GetDevicePublicKey()...),
		DeviceFingerprint: strings.TrimSpace(hello.GetDeviceFingerprint()), ServerNonce: append([]byte(nil), hello.GetServerNonce()...),
		ChannelBinding: channelBindingToProto(binding), IssuedAtUnixNano: hello.GetIssuedAtUnixNano(),
	}
	if input.AuthSessionId == "" || input.DeviceId == "" || input.DeviceFingerprint == "" || len(input.ServerNonce) == 0 {
		return nil, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "device hello canonical input is incomplete", nil)
	}
	return deterministicMarshal(input)
}

// ClientProofSigningBytes 返回 ClientAccessIdentity 对 capability/pairing open 签名的 canonical bytes。
// credential 只以 SHA-256 进入签名输入；grant/ticket 原文仍仅位于端到端 auth envelope，不会进入日志或 Cloud DTO。
func ClientProofSigningBytes(openKind remoteauthpb.AuthOpenKind, credential []byte, clientPublicKey ed25519.PublicKey, authSessionID string, serverNonce []byte, clientNonce []byte, binding ChannelBinding) ([]byte, error) {
	if openKind != remoteauthpb.AuthOpenKind_AUTH_OPEN_KIND_CAPABILITY && openKind != remoteauthpb.AuthOpenKind_AUTH_OPEN_KIND_PAIRING {
		return nil, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "client proof open kind is invalid", nil)
	}
	if len(credential) == 0 || strings.TrimSpace(authSessionID) == "" || len(serverNonce) != authNonceBytes || len(clientNonce) != authNonceBytes || len(clientPublicKey) != ed25519.PublicKeySize {
		return nil, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "client proof input is incomplete", nil)
	}
	if err := binding.Validate(); err != nil {
		return nil, err
	}
	credentialHash := sha256.Sum256(credential)
	input := &remoteauthpb.ClientProofInput{
		Protocol: AuthProtocol, Version: AuthVersion, AuthSessionId: strings.TrimSpace(authSessionID),
		ServerNonce: append([]byte(nil), serverNonce...), ClientNonce: append([]byte(nil), clientNonce...),
		ChannelBinding: channelBindingToProto(binding), CredentialSha256: credentialHash[:],
		ClientPublicKey: append([]byte(nil), clientPublicKey...), OpenKind: openKind,
	}
	return deterministicMarshal(input)
}

// SignClientProof 使用 ClientAccessIdentity private key 签名当前 capability 或 pairing challenge。
// proof 绑定 auth session、双方 nonce、actual channel binding、credential hash 与 client public key，复制 grant/ticket 文本不能产生有效签名。
func SignClientProof(identity ClientAccessIdentity, openKind remoteauthpb.AuthOpenKind, credential []byte, authSessionID string, serverNonce []byte, clientNonce []byte, binding ChannelBinding) ([]byte, error) {
	if err := identity.Validate(); err != nil {
		return nil, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH, "ClientAccessIdentity is invalid", err)
	}
	signer, err := NewPrivateClientAccessSigner(identity)
	if err != nil {
		return nil, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH, "ClientAccessIdentity signer is invalid", err)
	}
	return signClientProof(context.Background(), identity, signer, openKind, credential, authSessionID, serverNonce, clientNonce, binding)
}

func signClientProof(ctx context.Context, identity ClientAccessIdentity, signer ClientAccessSigner, openKind remoteauthpb.AuthOpenKind, credential []byte, authSessionID string, serverNonce []byte, clientNonce []byte, binding ChannelBinding) ([]byte, error) {
	if signer == nil {
		native, err := NewPrivateClientAccessSigner(identity)
		if err != nil {
			return nil, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH, "ClientAccessIdentity signer is invalid", err)
		}
		signer = native
	}
	if err := identity.ValidatePublic(); err != nil {
		return nil, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH, "ClientAccessIdentity signer is invalid", err)
	}
	canonical, err := ClientProofSigningBytes(openKind, credential, identity.PublicKey, authSessionID, serverNonce, clientNonce, binding)
	if err != nil {
		return nil, err
	}
	proof, err := signer.Sign(ctx, canonical)
	if err != nil {
		return nil, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_INTERNAL, "ClientAccessIdentity signing failed", err)
	}
	if len(proof) != ed25519.SignatureSize || !ed25519.Verify(identity.PublicKey, canonical, proof) {
		return nil, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH, "ClientAccessIdentity signer used a different key", nil)
	}
	return append([]byte(nil), proof...), nil
}

// validateSingleAuthPayloadWire 在 protobuf oneof 解码前检查原始 wire 中恰好出现一个 payload field。
// Go protobuf 会保留最后一个 oneof 值，因此必须在 unmarshal 前拒绝重复或多个 payload，避免攻击者制造解释分叉。
func validateSingleAuthPayloadWire(payload []byte) error {
	payloadFields := 0
	for len(payload) > 0 {
		fieldNumber, wireType, tagLength := protowire.ConsumeTag(payload)
		if tagLength < 0 {
			return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "decode remote auth envelope tag", protowire.ParseError(tagLength))
		}
		fieldLength := protowire.ConsumeFieldValue(fieldNumber, wireType, payload[tagLength:])
		if fieldLength < 0 {
			return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "decode remote auth envelope field", protowire.ParseError(fieldLength))
		}
		if fieldNumber >= 4 && fieldNumber <= 9 {
			payloadFields++
		}
		payload = payload[tagLength+fieldLength:]
	}
	if payloadFields != 1 {
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "remote auth envelope must contain exactly one payload field", nil)
	}
	return nil
}

// NormalizeDTLSCertificateFingerprint 校验并规范化 daemon SHA-256 DTLS certificate fingerprint。
// 输入格式固定为 `sha-256:aa:bb:...`；不接受 SDP 字段替代实际 adapter 读取的 certificate bytes。
func NormalizeDTLSCertificateFingerprint(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	const prefix = "sha-256:"
	if !strings.HasPrefix(value, prefix) {
		return "", newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "daemon DTLS fingerprint must use sha-256", nil)
	}
	rawHex := strings.ReplaceAll(strings.TrimPrefix(value, prefix), ":", "")
	decoded, err := hex.DecodeString(rawHex)
	if err != nil || len(decoded) != sha256.Size {
		return "", newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "daemon DTLS fingerprint is malformed", err)
	}
	parts := make([]string, len(decoded))
	for index, octet := range decoded {
		parts[index] = fmt.Sprintf("%02x", octet)
	}
	return prefix + strings.Join(parts, ":"), nil
}

func channelBindingToProto(binding ChannelBinding) *remoteauthpb.ChannelBinding {
	return &remoteauthpb.ChannelBinding{Kind: binding.Kind, BindingHash: append([]byte(nil), binding.Hash[:]...)}
}

func channelBindingFromProto(message *remoteauthpb.ChannelBinding) (ChannelBinding, error) {
	if message == nil || len(message.GetBindingHash()) != sha256.Size {
		return ChannelBinding{}, newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "channel binding is malformed", nil)
	}
	var hash [sha256.Size]byte
	copy(hash[:], message.GetBindingHash())
	binding := ChannelBinding{Kind: message.GetKind(), Hash: hash}
	return binding, binding.Validate()
}

func channelBindingsEqual(left ChannelBinding, right ChannelBinding) bool {
	return left.Kind == right.Kind && subtle.ConstantTimeCompare(left.Hash[:], right.Hash[:]) == 1
}

func validateAuthEnvelope(envelope *remoteauthpb.AuthEnvelope) error {
	if envelope == nil || envelope.GetProtocol() != AuthProtocol || envelope.GetVersion() != AuthVersion {
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "unsupported remote auth envelope", nil)
	}
	sessionID := strings.TrimSpace(envelope.GetAuthSessionId())
	if len(sessionID) < 16 || len(sessionID) > 128 {
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "invalid remote auth session id", nil)
	}
	for _, char := range sessionID {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' && char != '_' {
			return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "invalid remote auth session id", nil)
		}
	}
	if envelope.GetPayload() == nil || messageHasUnknown(envelope.ProtoReflect()) {
		return newHandshakeError(remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, "ambiguous remote auth envelope", nil)
	}
	return nil
}

func deterministicMarshal(message proto.Message) ([]byte, error) {
	return (proto.MarshalOptions{Deterministic: true}).Marshal(message)
}

func messageHasUnknown(message protoreflect.Message) bool {
	if len(message.GetUnknown()) != 0 {
		return true
	}
	found := false
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if field.Kind() != protoreflect.MessageKind && field.Kind() != protoreflect.GroupKind {
			return true
		}
		if field.IsList() {
			list := value.List()
			for index := 0; index < list.Len(); index++ {
				if messageHasUnknown(list.Get(index).Message()) {
					found = true
					return false
				}
			}
			return !found
		}
		found = messageHasUnknown(value.Message())
		return !found
	})
	return found
}

func newHandshakeError(code remoteauthpb.AuthErrorCode, detail string, cause error) *HandshakeError {
	if code == remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_UNSPECIFIED {
		code = remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_INTERNAL
	}
	return &HandshakeError{Code: code, Detail: detail, Cause: cause}
}

func rejectionMessage(code remoteauthpb.AuthErrorCode) string {
	switch code {
	case remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH:
		return "device identity verification failed"
	case remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_EXPIRED:
		return "capability is expired or not active"
	case remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_REVOKED:
		return "capability has been revoked"
	case remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_PROOF_INVALID:
		return "client access proof verification failed"
	case remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH:
		return "client access identity does not match capability subject"
	case remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PAIRING_TICKET_INVALID:
		return "pairing ticket verification failed"
	case remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PAIRING_TICKET_EXPIRED:
		return "pairing ticket is expired or not active"
	case remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PAIRING_TICKET_CONSUMED:
		return "pairing ticket is already bound"
	case remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SCOPE_INVALID:
		return "capability scope is invalid"
	case remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_REPLAYED:
		return "authorization challenge is no longer valid"
	case remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_INVALID:
		return "capability verification failed"
	case remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL:
		return "remote authorization protocol error"
	default:
		return "remote authorization failed"
	}
}

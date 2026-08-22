package remoteauth

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anytty/anytty/proto/remoteauthpb"
	"google.golang.org/protobuf/proto"
)

const (
	// DirectSignalingSchemaVersion 是 embedded signaling request/answer 当前唯一接受的 schema 版本。
	DirectSignalingSchemaVersion uint32 = 2
	// DirectSignalingMaxTTL 限制公开 signaling admission 的最大有效窗口。
	DirectSignalingMaxTTL   = 30 * time.Second
	directSignalingProtocol = "anytty.direct-signaling.v2"
)

// SignDirectSignalingAnswer 使用 daemon-local DeviceIdentity 对短期 Direct answer 签名。
// answer 必须已经包含当前 daemon public identity、request correlation 和有效期；本函数不会修正上层字段。
func SignDirectSignalingAnswer(identity Identity, answer *remoteauthpb.DirectSignalingAnswerV2) error {
	if err := identity.Validate(); err != nil {
		return fmt.Errorf("sign direct signaling answer with invalid identity: %w", err)
	}
	if answer == nil {
		return errors.New("direct signaling answer is required")
	}
	canonical, err := directSignalingAnswerBytes(answer)
	if err != nil {
		return err
	}
	answer.Signature = ed25519.Sign(identity.PrivateKey, canonical)
	return nil
}

// VerifyDirectSignalingAnswer 校验短期 answer 的 Endpoint pin、有效期与 daemon Ed25519 签名。
// signaling 地址、SDP 和 candidate 都不能替代 expected DeviceID/fingerprint；失败时调用方必须关闭 peer。
func VerifyDirectSignalingAnswer(answer *remoteauthpb.DirectSignalingAnswerV2, requestID, expectedDeviceID, expectedFingerprint string, now time.Time) error {
	if answer == nil || answer.GetIdentity() == nil {
		return errors.New("direct signaling answer identity is required")
	}
	if answer.GetSchemaVersion() != DirectSignalingSchemaVersion || strings.TrimSpace(answer.GetRequestId()) == "" || answer.GetRequestId() != requestID {
		return errors.New("direct signaling answer correlation is invalid")
	}
	identity := answer.GetIdentity()
	if identity.GetDeviceId() != strings.TrimSpace(expectedDeviceID) || identity.GetDeviceFingerprint() != strings.TrimSpace(expectedFingerprint) {
		return errors.New("direct signaling answer does not match endpoint pin")
	}
	if len(identity.GetDevicePublicKey()) != ed25519.PublicKeySize || Fingerprint(ed25519.PublicKey(identity.GetDevicePublicKey())) != identity.GetDeviceFingerprint() {
		return errors.New("direct signaling answer public identity is invalid")
	}
	issuedAt := time.Unix(0, answer.GetIssuedAtUnixNano()).UTC()
	expiresAt := time.Unix(0, answer.GetExpiresAtUnixNano()).UTC()
	now = now.UTC()
	if answer.GetIssuedAtUnixNano() <= 0 || answer.GetExpiresAtUnixNano() <= 0 || issuedAt.After(now.Add(ClockSkewTolerance)) ||
		!expiresAt.After(now.Add(-ClockSkewTolerance)) || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > DirectSignalingMaxTTL {
		return errors.New("direct signaling answer is expired or has invalid lifetime")
	}
	if strings.TrimSpace(answer.GetAnswerSdp()) == "" || len(answer.GetSignature()) != ed25519.SignatureSize {
		return errors.New("direct signaling answer is incomplete")
	}
	canonical, err := directSignalingAnswerBytes(answer)
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(identity.GetDevicePublicKey()), canonical, answer.GetSignature()) {
		return errors.New("direct signaling answer signature is invalid")
	}
	return nil
}

func directSignalingAnswerBytes(answer *remoteauthpb.DirectSignalingAnswerV2) ([]byte, error) {
	clone := proto.Clone(answer).(*remoteauthpb.DirectSignalingAnswerV2)
	clone.Signature = nil
	return (proto.MarshalOptions{Deterministic: true}).Marshal(&remoteauthpb.DirectSignalingAnswerSignatureInput{
		Protocol: directSignalingProtocol, Version: DirectSignalingSchemaVersion, Answer: clone,
	})
}

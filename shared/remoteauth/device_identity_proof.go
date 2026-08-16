package remoteauth

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"strings"

	"github.com/anytty/anytty/proto/remoteauthpb"
	"google.golang.org/protobuf/proto"
)

const (
	// DeviceIdentityChallengeBytes 固定 fresh proof challenge 的长度，避免空 challenge 或可控超大 payload 进入签名边界。
	DeviceIdentityChallengeBytes = 32
	deviceIdentityProofDomain    = "anytty.device-identity-proof.v1"
)

// SignDeviceIdentityProof 使用 daemon-local DeviceIdentity 私钥签署本次 application session 的随机 challenge。
// 返回值只能作为 identity possession proof；local/SSH transport authorization 仍由对应 adapter 独立负责。
func SignDeviceIdentityProof(identity Identity, challenge []byte) ([]byte, error) {
	if err := identity.Validate(); err != nil {
		return nil, err
	}
	canonical, err := deviceIdentityProofSigningBytes(challenge, identity.DeviceID, identity.Fingerprint, identity.PublicKey)
	if err != nil {
		return nil, err
	}
	return ed25519.Sign(identity.PrivateKey, canonical), nil
}

// VerifyDeviceIdentityProof 校验 response identity、原样 challenge 与 Ed25519 proof，成功即证明当前响应方持有对应 DeviceIdentity 私钥。
func VerifyDeviceIdentityProof(challenge []byte, deviceID, fingerprint string, publicKey, proof []byte) error {
	if len(publicKey) != ed25519.PublicKeySize || len(proof) != ed25519.SignatureSize {
		return errors.New("device identity proof key or signature length is invalid")
	}
	if Fingerprint(ed25519.PublicKey(publicKey)) != strings.TrimSpace(fingerprint) {
		return errors.New("device identity public key fingerprint mismatch")
	}
	canonical, err := deviceIdentityProofSigningBytes(challenge, deviceID, fingerprint, publicKey)
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), canonical, proof) {
		return errors.New("device identity proof signature is invalid")
	}
	return nil
}

func deviceIdentityProofSigningBytes(challenge []byte, deviceID, fingerprint string, publicKey []byte) ([]byte, error) {
	if len(challenge) != DeviceIdentityChallengeBytes {
		return nil, fmt.Errorf("device identity challenge must be %d bytes", DeviceIdentityChallengeBytes)
	}
	if strings.TrimSpace(deviceID) == "" || strings.TrimSpace(fingerprint) == "" || len(publicKey) != ed25519.PublicKeySize {
		return nil, errors.New("device identity proof input is incomplete")
	}
	return proto.MarshalOptions{Deterministic: true}.Marshal(&remoteauthpb.DeviceIdentityProofInput{
		Domain: deviceIdentityProofDomain, Challenge: append([]byte(nil), challenge...), DeviceId: strings.TrimSpace(deviceID),
		DeviceFingerprint: strings.TrimSpace(fingerprint), DevicePublicKey: append([]byte(nil), publicKey...),
	})
}

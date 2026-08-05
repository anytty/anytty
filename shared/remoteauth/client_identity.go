package remoteauth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"io"
	"strings"
)

// ClientAccessIdentity 是客户端针对单个 Endpoint 持有的长期 Ed25519 访问身份。
// PrivateKey 的 truth 只存在于当前平台 secure credential store；Fingerprint 被 owning daemon 写入 CapabilityGrant v2，
// 同一客户端访问不同 Endpoint 时必须使用不同 key，不能复用 Cloud installation identity。
type ClientAccessIdentity struct {
	EndpointID  string
	Fingerprint string
	PublicKey   ed25519.PublicKey
	PrivateKey  ed25519.PrivateKey
}

// ClientAccessSigner 是平台 secure key 对 remote-auth canonical proof 的异步签名边界。
// Android Keystore/WebCrypto 实现不得导出私钥；返回签名必须由调用方使用 Identity.PublicKey 再验证，防止 signer ref 指向其他 endpoint key。
type ClientAccessSigner interface {
	// Sign 对 Go remote-auth 提供的 canonical proof bytes 签名；实现必须响应 context 取消并且不得记录 payload。
	Sign(context.Context, []byte) ([]byte, error)
}

// Validate 校验 Endpoint 绑定、Ed25519 key pair 与 fingerprint 是否一致。
// 失败表示客户端 secure store 损坏或调用方把其他 Endpoint 的 key 混入当前连接，调用方必须停止授权且不能生成临时替代 key。
func (identity ClientAccessIdentity) Validate() error {
	if err := identity.ValidatePublic(); err != nil || len(identity.PrivateKey) != ed25519.PrivateKeySize {
		return fmt.Errorf("incomplete ClientAccessIdentity")
	}
	if !identity.PublicKey.Equal(identity.PrivateKey.Public()) {
		return fmt.Errorf("ClientAccessIdentity key and fingerprint mismatch")
	}
	return nil
}

// ValidatePublic 校验不含可导出私钥的 ClientAccessIdentity public projection。
// WebCrypto/Keystore signer 使用该入口；EndpointID、public key 与 fingerprint 任一不一致都必须在 Cloud 或握手前失败。
func (identity ClientAccessIdentity) ValidatePublic() error {
	if strings.TrimSpace(identity.EndpointID) == "" || len(identity.PublicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("incomplete ClientAccessIdentity public key")
	}
	if Fingerprint(identity.PublicKey) != strings.TrimSpace(identity.Fingerprint) {
		return fmt.Errorf("ClientAccessIdentity public key and fingerprint mismatch")
	}
	return nil
}

type privateClientAccessSigner struct {
	privateKey ed25519.PrivateKey
}

func (signer privateClientAccessSigner) Sign(ctx context.Context, payload []byte) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return ed25519.Sign(signer.privateKey, payload), nil
	}
}

// NewPrivateClientAccessSigner 把 native secure store 已加载的 identity 适配为 signer contract。
// 返回 signer 持有私钥副本且只能留在当前 Go 进程；Android/Web 应提供不可导出平台实现而不是调用本函数。
func NewPrivateClientAccessSigner(identity ClientAccessIdentity) (ClientAccessSigner, error) {
	if err := identity.Validate(); err != nil {
		return nil, err
	}
	return privateClientAccessSigner{privateKey: append(ed25519.PrivateKey(nil), identity.PrivateKey...)}, nil
}

// NewClientAccessIdentity 从平台 secure store 已持有的私钥构造一个 Endpoint 专用访问身份。
// 该函数复制 key material；返回值只能留在公开客户端进程并用于 PairingExchange 或 capability channel-bound signature。
func NewClientAccessIdentity(endpointID string, privateKey ed25519.PrivateKey) (ClientAccessIdentity, error) {
	endpointID = strings.TrimSpace(endpointID)
	if endpointID == "" {
		return ClientAccessIdentity{}, fmt.Errorf("ClientAccessIdentity requires endpoint_id")
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return ClientAccessIdentity{}, fmt.Errorf("ClientAccessIdentity requires ed25519 private key")
	}
	publicKey := append(ed25519.PublicKey(nil), privateKey.Public().(ed25519.PublicKey)...)
	identity := ClientAccessIdentity{
		EndpointID:  endpointID,
		Fingerprint: Fingerprint(publicKey),
		PublicKey:   publicKey,
		PrivateKey:  append(ed25519.PrivateKey(nil), privateKey...),
	}
	return identity, identity.Validate()
}

// GenerateClientAccessIdentity 使用调用方提供的 CSPRNG 生成 Endpoint 专用访问身份。
// reader 仅允许测试注入；生产 nil 使用 crypto/rand，生成失败时不得回退到 daemon key、Cloud identity 或其他 Endpoint 的 key。
func GenerateClientAccessIdentity(endpointID string, reader io.Reader) (ClientAccessIdentity, error) {
	if reader == nil {
		reader = rand.Reader
	}
	_, privateKey, err := ed25519.GenerateKey(reader)
	if err != nil {
		return ClientAccessIdentity{}, fmt.Errorf("generate ClientAccessIdentity: %w", err)
	}
	return NewClientAccessIdentity(endpointID, privateKey)
}

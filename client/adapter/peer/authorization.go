// Package peer 提供 Direct、SSH 与 Cloud WebRTC connector 共用的客户端认证事务。
// 本包不建立网络、不选择 Route，也不拥有 session generation；它只冻结 endpoint-bound credential 并执行 DTLS-bound remote auth。
package peer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport"
)

// CredentialSource 从当前平台 secure store 解析单个 endpoint-bound capability credential。
// 返回值只在 Go remote-auth adapter 内短暂使用；实现不得从其他 endpoint、Cloud token 或临时生成 key 回退。
type CredentialSource interface {
	// ResolveClientCredential 解析 endpoint/ref 对应的 grant 与 public identity projection。
	ResolveClientCredential(context.Context, string, string) (remoteauth.ClientAccessCredential, error)
}

// CloudEdgeLocatorStore 只更新 managed Route 的公开位置缓存，不得修改 identity 或授权 grant。
type CloudEdgeLocatorStore interface {
	UpdateCloudEdgeLocator(context.Context, string, string, []byte) error
}

// SignerSource 从 Android Keystore、WebCrypto 或其他平台 secure key owner 解析不可导出 signer。
// identity 只含 endpoint/public key/fingerprint 投影；实现必须返回同一 key 的 signer，remote-auth 会对签名结果再次验签。
type SignerSource interface {
	// ResolveClientSigner 返回与 identity public key 匹配的异步 signer；失败时不得生成替代 key。
	ResolveClientSigner(context.Context, string, string, remoteauth.ClientAccessIdentity) (remoteauth.ClientAccessSigner, error)
}

// PreparedAuthorization 是当前 endpoint/route 已完成本地 credential 校验后的单次认证事务。
// Authenticate 只能绑定当前 peer 的实际 DTLS certificate；失败后调用方必须关闭 peer，不能切换旧授权路径。
type PreparedAuthorization interface {
	GrantID() string
	GrantExpiresAt() time.Time
	// Authenticate 使用当前 peer certificate fingerprint 完成 DataChannel 内 capability proof。
	Authenticate(context.Context, transport.Transport, string) (remoteauth.Claims, error)
}

// PreparedSignalingAuthorization 是 managed Cloud connector 在发送任何网络请求前冻结的客户端身份材料。
// CloudRouteGrant 只授权发现；Sign 仍由同一平台 secure signer 执行，private key 不离开 owner。
type PreparedSignalingAuthorization interface {
	PreparedAuthorization
	ClientIdentity() remoteauth.ClientAccessIdentity
	CloudRouteGrant() []byte
	CloudEdgeLocator() []byte
	StoreCloudEdgeLocator(context.Context, []byte) error
	Sign(context.Context, []byte) ([]byte, error)
}

// Authorizer 在任何 signaling 请求前验证 endpoint-bound credential，并冻结本次认证事务。
// 平台 secure store 或 signer 适配位于实现侧，connector 不读取、持久化或记录私钥。
type Authorizer interface {
	// Prepare 在网络副作用前冻结 endpoint-bound credential/signer 事务。
	Prepare(context.Context, clientruntime.AttemptRequest) (PreparedAuthorization, error)
}

// CapabilityAuthorizer 使用共享 Go remote-auth 状态机完成 WebRTC DataChannel 授权。
// Credentials 是平台 secure-store adapter；Random/Now 仅用于 deterministic harness，生产零值使用安全默认值。
type CapabilityAuthorizer struct {
	Credentials CredentialSource
	Signers     SignerSource
	Random      io.Reader
	Now         func() time.Time
}

// Prepare 在 signaling 之前验证 credential 的 endpoint、issuer、subject 和有效期。
// 成功结果冻结本次 attempt 使用的 credential，防止连接过程中 secure-store ref 被替换后混入其他身份。
func (authorizer CapabilityAuthorizer) Prepare(ctx context.Context, request clientruntime.AttemptRequest) (PreparedAuthorization, error) {
	if authorizer.Credentials == nil {
		return nil, fmt.Errorf("peer endpoint credential source is required")
	}
	route := request.Route()
	credential, err := authorizer.Credentials.ResolveClientCredential(ctx, string(request.EndpointID()), route.CredentialRef)
	if err != nil {
		return nil, err
	}
	var signer remoteauth.ClientAccessSigner
	if authorizer.Signers != nil {
		if err := credential.Identity.ValidatePublic(); err != nil {
			return nil, fmt.Errorf("peer endpoint ClientAccessIdentity public projection is invalid: %w", err)
		}
		signer, err = authorizer.Signers.ResolveClientSigner(ctx, string(request.EndpointID()), route.CredentialRef, credential.Identity)
		if err != nil {
			return nil, err
		}
		if signer == nil {
			return nil, fmt.Errorf("peer endpoint signer source returned no signer")
		}
	} else {
		signer, err = remoteauth.NewPrivateClientAccessSigner(credential.Identity)
		if err != nil {
			return nil, fmt.Errorf("peer endpoint ClientAccessIdentity is invalid: %w", err)
		}
	}
	endpointID := string(request.EndpointID())
	if strings.TrimSpace(credential.EndpointID) != endpointID || credential.Identity.EndpointID != endpointID {
		return nil, fmt.Errorf("peer endpoint credential belongs to endpoint %q, not %q", credential.EndpointID, endpointID)
	}
	if !credential.Ready() {
		return nil, fmt.Errorf("peer endpoint capability credential is awaiting pairing")
	}
	now := time.Now().UTC()
	if authorizer.Now != nil {
		now = authorizer.Now().UTC()
	}
	claims, err := remoteauth.Verify(credential.CapabilityGrant, request.DaemonIdentity().DeviceFingerprint, now, nil)
	if err != nil {
		return nil, fmt.Errorf("verify peer endpoint capability grant: %w", err)
	}
	if claims.IssuerDeviceID != request.DaemonIdentity().DeviceID {
		return nil, fmt.Errorf("peer endpoint device mismatch: grant %q registry %q", claims.IssuerDeviceID, request.DaemonIdentity().DeviceID)
	}
	if claims.SubjectKeyFingerprint != credential.Identity.Fingerprint {
		return nil, fmt.Errorf("peer endpoint capability subject does not match ClientAccessIdentity")
	}
	return &preparedCapabilityAuthorization{
		credential: credential, identity: request.DaemonIdentity(), claims: claims, signer: signer, random: authorizer.Random, now: authorizer.Now,
		credentialRef: route.CredentialRef, locatorStore: locatorStore(authorizer.Credentials),
	}, nil
}

type preparedCapabilityAuthorization struct {
	credential    remoteauth.ClientAccessCredential
	identity      endpoint.DaemonIdentity
	claims        remoteauth.Claims
	signer        remoteauth.ClientAccessSigner
	random        io.Reader
	now           func() time.Time
	credentialRef string
	locatorStore  CloudEdgeLocatorStore
}

func (authorization *preparedCapabilityAuthorization) GrantID() string {
	return authorization.claims.GrantID
}

func (authorization *preparedCapabilityAuthorization) GrantExpiresAt() time.Time {
	return authorization.claims.ExpiresAt
}

func (authorization *preparedCapabilityAuthorization) Authenticate(ctx context.Context, connection transport.Transport, certificateFingerprint string) (remoteauth.Claims, error) {
	binding, err := remoteauth.DTLSChannelBinding(certificateFingerprint)
	if err != nil {
		return remoteauth.Claims{}, fmt.Errorf("bind peer endpoint DTLS certificate: %w", err)
	}
	handshake := remoteauth.ClientHandshake{Random: authorization.random, Now: authorization.now}
	return handshake.Authenticate(ctx, connection, remoteauth.ClientHandshakeRequest{
		ExpectedDeviceID: authorization.identity.DeviceID, ExpectedDeviceFingerprint: authorization.identity.DeviceFingerprint,
		Credential: authorization.credential, Signer: authorization.signer, ChannelBinding: binding,
	})
}

// ClientIdentity 返回当前 attempt 已冻结的公开 ClientAccessIdentity 投影。
func (authorization *preparedCapabilityAuthorization) ClientIdentity() remoteauth.ClientAccessIdentity {
	identity := authorization.credential.Identity
	identity.PublicKey = append([]byte(nil), identity.PublicKey...)
	identity.PrivateKey = nil
	return identity
}

// CloudRouteGrant 返回 pairing 时与同一 credential 原子保存的 DeviceIdentity 签名发现 grant。
func (authorization *preparedCapabilityAuthorization) CloudRouteGrant() []byte {
	return append([]byte(nil), authorization.credential.CloudRouteGrant...)
}

func (authorization *preparedCapabilityAuthorization) CloudEdgeLocator() []byte {
	return append([]byte(nil), authorization.credential.CloudEdgeLocator...)
}

func (authorization *preparedCapabilityAuthorization) StoreCloudEdgeLocator(ctx context.Context, locator []byte) error {
	if authorization.locatorStore == nil {
		return errors.New("Cloud Edge locator store is unavailable")
	}
	return authorization.locatorStore.UpdateCloudEdgeLocator(ctx, authorization.credential.EndpointID, authorization.credentialRef, append([]byte(nil), locator...))
}

// Sign 委托当前 frozen platform signer 对 Cloud challenge/hello canonical bytes 签名。
func (authorization *preparedCapabilityAuthorization) Sign(ctx context.Context, payload []byte) ([]byte, error) {
	return authorization.signer.Sign(ctx, append([]byte(nil), payload...))
}

func locatorStore(source CredentialSource) CloudEdgeLocatorStore {
	store, _ := source.(CloudEdgeLocatorStore)
	return store
}

var _ Authorizer = CapabilityAuthorizer{}

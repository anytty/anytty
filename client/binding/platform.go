package binding

import (
	"context"
	"fmt"
	"sync"

	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/bindingpb"
	"google.golang.org/protobuf/proto"
)

const defaultPlatformRequestCapacity = 64

// PairingHost 是 binding 可选的 portable pairing bootstrap owner。
// 实现必须使用 generated Proto 验证 bundle，并通过平台 credential signer/store port 准备身份；不得把私钥返回 binding。
type PairingHost interface {
	// ImportPairing 验证一次性 bootstrap 并返回非秘密 endpoint metadata。
	ImportPairing(context.Context, *bindingpb.ImportPairingRequest) (*bindingpb.ImportPairingResult, error)
}

// CredentialHost 是 binding 可选的平台 credential lifecycle owner。
// 删除只移除本地 credential，不代表 daemon grant 已撤销。
type CredentialHost interface {
	// DeleteCredential 删除 request 指定的平台 credential ref。
	DeleteCredential(context.Context, *bindingpb.DeleteCredentialRequest) error
}

// EndpointRegistryHost 是 binding 对 Go-owned Endpoint registry 的唯一业务入口。
// 平台只能持久化 opaque Proto bytes；读取、校验、identity 冲突、更新和删除事务必须由实现持有。
type EndpointRegistryHost interface {
	// GetEndpointRegistry 返回当前 generation 读取到的规范化 registry projection。
	GetEndpointRegistry(context.Context, *bindingpb.EndpointRegistryGetRequest) (*bindingpb.EndpointRegistryGetResult, error)
	// UpsertEndpoint 校验并原子写入一个 generated EndpointConfigV1，禁止替换已 pin 的 daemon identity。
	UpsertEndpoint(context.Context, *bindingpb.EndpointUpsertRequest) (*bindingpb.EndpointUpsertResult, error)
	// DeleteEndpoint 原子移除 endpoint 配置，并在提交后清理不再引用的本地 credential。
	DeleteEndpoint(context.Context, *bindingpb.EndpointDeleteRequest) (*bindingpb.EndpointDeleteResult, error)
}

// ConnectionPolicyHost 是 binding 对 Endpoint 网络选择策略的窄业务入口。
// 实现必须复用 Go planner 的平台、凭据和 Cloud eligibility 真值；UI 不得直接改写 registry 字段。
type ConnectionPolicyHost interface {
	// GetConnectionPolicy 返回持久策略及当前 generation 可证明的 Route kind 可用性。
	GetConnectionPolicy(context.Context, *bindingpb.ConnectionPolicyGetRequest) (*bindingpb.ConnectionPolicyGetResult, error)
	// ApplyConnectionPolicy 原子持久化用户策略，并返回提交后的 planner 可用性投影。
	ApplyConnectionPolicy(context.Context, *bindingpb.ConnectionPolicyApplyRequest) (*bindingpb.ConnectionPolicyApplyResult, error)
}

// CloudPresenceHost performs a read-only, endpoint-authenticated probe against
// the cached Edge. It must not resolve through the Controller or open a session.
type CloudPresenceHost interface {
	GetEndpointCloudPresence(context.Context, *bindingpb.EndpointCloudPresenceGetRequest) (*bindingpb.EndpointCloudPresenceGetResult, error)
}

// SessionInvalidationHost removes one exact endpoint generation after a
// confirmed platform network change. Closing a binding session is intentionally
// weaker because normal UI consumers must not tear down a shared P2P session.
type SessionInvalidationHost interface {
	InvalidateSession(context.Context, clientruntime.EndpointSessionStamp) error
}

// EndpointDisconnectionHost owns explicit endpoint-wide user disconnects.
// It must serialize behind an in-flight OpenSession and fence its generation.
type EndpointDisconnectionHost interface {
	DisconnectEndpoint(context.Context, endpoint.EndpointID) error
}

// EndpointShareHost 是 binding 对一次性 Endpoint share 的两阶段入口。
// Receive 只返回 Go 计算的 diff 并持有 generation-local token；Commit 才能原子更新 registry。
type EndpointShareHost interface {
	// ReceiveEndpointShare 完成 TLS pin、receiver proof 和 bundle 校验，但不持久化配置。
	ReceiveEndpointShare(context.Context, *bindingpb.EndpointShareReceiveRequest) (*bindingpb.EndpointShareReceiveResult, error)
	// CommitEndpointShare 提交当前 generation 内尚未过期的 import token。
	CommitEndpointShare(context.Context, *bindingpb.EndpointShareCommitRequest) (*bindingpb.EndpointShareCommitResult, error)
}

// SSHCredentialHost 是 binding 对 Go-owned SSH Route credential provisioning 的业务入口。
// 平台只持有不可导出 signer；Endpoint/Route 选择、credential ref 绑定和 registry 事务由实现负责。
type SSHCredentialHost interface {
	// ProvisionSSHCredential 为指定 SSH Route 准备平台 signer，并返回更新后的 registry 与公开 SSH key。
	ProvisionSSHCredential(context.Context, *bindingpb.SSHCredentialProvisionRequest) (*bindingpb.SSHCredentialProvisionResult, error)
}

// PlatformBroker 是 Go Client Engine 与 Android/WASM platform adapter 之间的 request/response owner。
// 请求和响应均为 bindingpb，平台通过 NextRequest/Complete 驱动；broker 不解释 Cloud 或 credential 字段。
type PlatformBroker struct {
	ctx    context.Context
	cancel context.CancelFunc

	mu      sync.Mutex
	closed  bool
	nextID  uint64
	pending map[uint64]chan *bindingpb.PlatformResponse
	queue   chan []byte
}

// NewPlatformBroker 创建有界平台请求 broker。
// 队列满时 Exchange 施加背压；关闭会取消全部等待方，不能静默丢弃签名或 signaling 请求。
func NewPlatformBroker() *PlatformBroker {
	ctx, cancel := context.WithCancel(context.Background())
	return &PlatformBroker{
		ctx: ctx, cancel: cancel, pending: make(map[uint64]chan *bindingpb.PlatformResponse),
		queue: make(chan []byte, defaultPlatformRequestCapacity),
	}
}

// Exchange 发布一条平台请求并等待 request_id 匹配的响应。
// 调用方 context 取消后会移除 pending；迟到响应返回 invalid request，不能命中新请求。
func (broker *PlatformBroker) Exchange(ctx context.Context, request *bindingpb.PlatformRequest) (*bindingpb.PlatformResponse, error) {
	if broker == nil || request == nil || request.GetRequest() == nil {
		return nil, fmt.Errorf("platform request is incomplete")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	broker.mu.Lock()
	if broker.closed || broker.nextID == ^uint64(0) {
		broker.mu.Unlock()
		return nil, ErrClosed
	}
	broker.nextID++
	requestID := broker.nextID
	responseChannel := make(chan *bindingpb.PlatformResponse, 1)
	broker.pending[requestID] = responseChannel
	broker.mu.Unlock()

	snapshot := proto.Clone(request).(*bindingpb.PlatformRequest)
	snapshot.RequestId = requestID
	payload, err := (proto.MarshalOptions{Deterministic: true}).Marshal(snapshot)
	if err != nil {
		broker.removePending(requestID)
		return nil, fmt.Errorf("encode platform request: %w", err)
	}
	if err := ctx.Err(); err != nil {
		broker.removePending(requestID)
		return nil, err
	}
	select {
	case <-ctx.Done():
		broker.removePending(requestID)
		return nil, ctx.Err()
	case <-broker.ctx.Done():
		broker.removePending(requestID)
		return nil, ErrClosed
	case broker.queue <- payload:
	}
	select {
	case <-ctx.Done():
		broker.removePending(requestID)
		return nil, ctx.Err()
	case <-broker.ctx.Done():
		broker.removePending(requestID)
		return nil, ErrClosed
	case response := <-responseChannel:
		return response, nil
	}
}

// NextRequest 阻塞读取下一条 serialized bindingpb.PlatformRequest。
// 返回独立副本，JNI/WASM wrapper 必须沿用显式 buffer ownership。
func (broker *PlatformBroker) NextRequest(ctx context.Context) ([]byte, error) {
	if broker == nil {
		return nil, ErrClosed
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-broker.ctx.Done():
		return nil, ErrClosed
	case payload := <-broker.queue:
		return append([]byte(nil), payload...), nil
	}
}

// Complete 解析并交付一条 PlatformResponse。
// request_id 缺失、未知或重复完成都会失败，平台错误必须写入 response.error 而不是吞掉响应。
func (broker *PlatformBroker) Complete(payload []byte) error {
	if err := validatePayload(payload); err != nil {
		return err
	}
	response := &bindingpb.PlatformResponse{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, response); err != nil {
		return fmt.Errorf("decode platform response: %w", err)
	}
	if response.GetRequestId() == 0 {
		return fmt.Errorf("platform response request_id is required")
	}
	broker.mu.Lock()
	channel := broker.pending[response.GetRequestId()]
	if channel != nil {
		delete(broker.pending, response.GetRequestId())
	}
	broker.mu.Unlock()
	if channel == nil {
		return ErrInvalidHandle
	}
	channel <- proto.Clone(response).(*bindingpb.PlatformResponse)
	return nil
}

// Close 取消全部平台请求并禁止新 Exchange。
// 方法幂等，适用于 Android process teardown 和 WASM worker disposal。
func (broker *PlatformBroker) Close() error {
	if broker == nil {
		return nil
	}
	broker.mu.Lock()
	if broker.closed {
		broker.mu.Unlock()
		return nil
	}
	broker.closed = true
	broker.pending = make(map[uint64]chan *bindingpb.PlatformResponse)
	broker.mu.Unlock()
	broker.cancel()
	return nil
}

func (broker *PlatformBroker) removePending(requestID uint64) {
	broker.mu.Lock()
	delete(broker.pending, requestID)
	broker.mu.Unlock()
}

package runtime

import (
	"context"
	"errors"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
)

// EndpointPlanSnapshot 是 application runtime 建连前取得的不可变 endpoint/planner 输入。
// ConfigKey 必须覆盖会改变连接结果的 endpoint、route policy、platform capability 与 credential ref 索引，但不能包含 credential body。
type EndpointPlanSnapshot struct {
	Endpoint    endpoint.Endpoint
	Environment RoutePlanEnvironment
	ConfigKey   string
}

// EndpointPlanSource 按 EndpointID 读取当前 registry 与平台 planner 环境快照。
// source 只读取配置和 capability 索引，不建立网络连接、不分配 generation，也不缓存 ReadyPeerSession。
type EndpointPlanSource interface {
	Snapshot(context.Context, endpoint.EndpointID) (EndpointPlanSnapshot, error)
}

// ClientRuntime 是 CLI、TUI 与跨语言 binding 共用的连接控制面 application facade。
// SessionOwner 仍是 generation/current winner 唯一真值；本类型只解析 ConnectRequest、取得 snapshot 并调用 planner race。
type ClientRuntime struct {
	owner   *SessionOwner
	source  EndpointPlanSource
	clock   port.Clock
	dialers PeerConnectorResolver
}

// NewClientRuntime 装配共享 runtime facade；依赖缺失时返回错误，不能构造会在连接时 fallback 的半可用 runtime。
func NewClientRuntime(owner *SessionOwner, source EndpointPlanSource, clock port.Clock, dialers PeerConnectorResolver) (*ClientRuntime, error) {
	if owner == nil || source == nil || clock == nil || dialers == nil {
		return nil, runtimeError(ErrorInvalidRequest, "session owner, endpoint source, clock, and route dialers are required", nil)
	}
	return &ClientRuntime{owner: owner, source: source, clock: clock, dialers: dialers}, nil
}

// EnsureSession 复用匹配 config key 的当前 winner，否则执行一次新的 planner race。
// 返回 lease 只携带 winner stamp；具体 command/event 仍回到同一个 SessionOwner generation fence。
func (runtime *ClientRuntime) EnsureSession(ctx context.Context, request ConnectRequest) (SessionLease, error) {
	if runtime == nil {
		return SessionLease{}, runtimeError(ErrorUnavailable, "client runtime is unavailable", nil)
	}
	if ctx == nil {
		return SessionLease{}, runtimeError(ErrorInvalidRequest, "connect context is required", nil)
	}
	if err := request.Validate(); err != nil {
		return SessionLease{}, err
	}
	runtime.owner.publishEndpointEvent(EndpointEvent{EndpointID: request.EndpointID, Phase: EndpointPhaseResolving})
	snapshot, err := runtime.source.Snapshot(ctx, request.EndpointID)
	if err != nil {
		resolvedErr := sourceRuntimeError(err)
		runtime.owner.publishEndpointEvent(EndpointEvent{EndpointID: request.EndpointID, Phase: EndpointPhaseOffline, ErrorCode: CodeOf(resolvedErr), Message: errorMessage(resolvedErr)})
		return SessionLease{}, resolvedErr
	}
	if snapshot.Endpoint.ID != request.EndpointID || snapshot.ConfigKey == "" {
		return SessionLease{}, runtimeError(ErrorInvalidRequest, "endpoint plan snapshot does not match connect request", nil)
	}
	return runtime.owner.EnsurePlanned(ctx, snapshot.Endpoint, request.RouteOverride, request.Intent, snapshot.ConfigKey, snapshot.Environment, runtime.clock, runtime.dialers)
}

// AcquireSession 为一个 application consumer 获取 endpoint-scoped 共享 lease。
// 同 Endpoint、同 config key 的 ready winner 会被复用；Close 只释放当前 consumer，
// 不会提前关闭仍被其它 TUI/CLI consumer 使用的连接。
func (runtime *ClientRuntime) AcquireSession(ctx context.Context, request ConnectRequest) (ApplicationReadyPeerSession, error) {
	if runtime == nil {
		return nil, runtimeError(ErrorUnavailable, "client runtime is unavailable", nil)
	}
	if ctx == nil {
		return nil, runtimeError(ErrorInvalidRequest, "connect context is required", nil)
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	runtime.owner.publishEndpointEvent(EndpointEvent{EndpointID: request.EndpointID, Phase: EndpointPhaseResolving})
	snapshot, err := runtime.source.Snapshot(ctx, request.EndpointID)
	if err != nil {
		resolvedErr := sourceRuntimeError(err)
		runtime.owner.publishEndpointEvent(EndpointEvent{EndpointID: request.EndpointID, Phase: EndpointPhaseOffline, ErrorCode: CodeOf(resolvedErr), Message: errorMessage(resolvedErr)})
		return nil, resolvedErr
	}
	if snapshot.Endpoint.ID != request.EndpointID || snapshot.ConfigKey == "" {
		return nil, runtimeError(ErrorInvalidRequest, "endpoint plan snapshot does not match connect request", nil)
	}
	return runtime.owner.AcquirePlanned(ctx, snapshot.Endpoint, request.RouteOverride, request.Intent, snapshot.ConfigKey, snapshot.Environment, runtime.clock, runtime.dialers)
}

// Disconnect 按完整 generation stamp 释放当前 winner；stale request 不能关闭后续 session。
func (runtime *ClientRuntime) Disconnect(ctx context.Context, request DisconnectRequest) error {
	if runtime == nil {
		return runtimeError(ErrorUnavailable, "client runtime is unavailable", nil)
	}
	return runtime.owner.Disconnect(ctx, request)
}

// WatchEndpoint 返回 SessionOwner 的 bounded lifecycle mailbox，不建立连接或隐式订阅 application event。
func (runtime *ClientRuntime) WatchEndpoint(ctx context.Context, endpointID endpoint.EndpointID) (<-chan EndpointEvent, error) {
	if runtime == nil {
		return nil, runtimeError(ErrorUnavailable, "client runtime is unavailable", nil)
	}
	return runtime.owner.WatchEndpoint(ctx, endpointID)
}

// PlanSnapshot 返回 EndpointPlanSource 当前可证明的 registry/planner 环境快照，供原生 Go 客户端展示策略与可用性。
// 该读取不分配 generation、不建立连接、不暴露 credential body；TUI/App 不得把快照缓存成第二份连接真值。
func (runtime *ClientRuntime) PlanSnapshot(ctx context.Context, endpointID endpoint.EndpointID) (EndpointPlanSnapshot, error) {
	if runtime == nil || runtime.source == nil {
		return EndpointPlanSnapshot{}, runtimeError(ErrorUnavailable, "client runtime plan source is unavailable", nil)
	}
	if ctx == nil {
		return EndpointPlanSnapshot{}, runtimeError(ErrorInvalidRequest, "endpoint plan snapshot context is required", nil)
	}
	return runtime.source.Snapshot(ctx, endpointID)
}

// Close 关闭唯一 SessionOwner 及其全部 winner、consumer lease 与 lifecycle watcher。
func (runtime *ClientRuntime) Close() error {
	if runtime == nil {
		return nil
	}
	return runtime.owner.Close()
}

var _ Runtime = (*ClientRuntime)(nil)
var _ ApplicationRuntime = (*ClientRuntime)(nil)

func sourceRuntimeError(err error) error {
	var value *Error
	if errors.As(err, &value) {
		return err
	}
	return runtimeError(ErrorUnavailable, "resolve endpoint plan snapshot", err)
}

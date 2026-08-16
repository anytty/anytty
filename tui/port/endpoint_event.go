package port

import (
	"context"

	"github.com/anytty/anytty/tui/state"
)

// EndpointRuntimeEvent 是共享 client runtime 发布给 TUI adapter 的连接生命周期事件。
// 它只描述某个 EndpointID 的 transport/protocol 状态，不携带 terminal lifecycle truth；
// TUI reducer 应把它投影到对应 pane/manager/picker，而不是升级成全局 toast。
type EndpointRuntimeEvent struct {
	EndpointID state.EndpointID
	// RouteID 与 Generation 原样来自 Go SessionOwner stamp；两者只用于展示和拒绝迟到旧事件。
	// TUI 不得用 registry 当前值补造 winner route，也不得分配 generation。
	RouteID    string
	Generation uint64
	Status     state.EndpointStatusKind
	ErrorKind  state.EndpointErrorKind
	// Phase 是 managed WebRTC 当前 resolving/signaling/connecting/authorizing/connected/failed 阶段。
	// local/SSH 保持空值；它只用于 reducer 展示，不能替代 Status、授权结果或实际 ObservedPath。
	Phase state.EndpointConnectionPhase
	// ObservedPath 是 managed WebRTC 已建立连接的 direct/single_relay/relay_mesh 运行时投影。
	// 它不参与 endpoint 路由或授权，空值表示 local/SSH 或尚未观测到 candidate path。
	ObservedPath string
	// RouteSelectionReason 是 SmartRoute 的稳定公开诊断，不包含候选分数、成本或内部权重。
	RouteSelectionReason string
	Message              string
	Err                  error
}

// EndpointEventSource 提供 endpoint-scoped 生命周期事件订阅。
// 该接口用于主动侦测 transport 关闭；订阅者只能通过 message path 回写 reducer state。
type EndpointEventSource interface {
	WatchEndpointEvents(context.Context) (<-chan EndpointRuntimeEvent, error)
}

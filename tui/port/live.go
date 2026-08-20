package port

import (
	"context"
	"time"

	"github.com/anytty/anytty/tui/state"
)

// NativeScreenSource 是 TUI live render loop 拉取 core latest native screen 的唯一接口。
// 它不返回 history token、scrollback 或 lifecycle truth；调用方只能把结果用于当前实时显示。
type NativeScreenSource interface {
	LiveSurface(context.Context, TerminalSurfaceRequest) (TerminalSurfaceResult, error)
}

// LiveScreenSource 提供 one-shot latest live screen 请求。
// 每次请求最多返回一份基于 observed revision 的 latest 行变化，不排队保存中间帧。
type LiveScreenSource interface {
	LiveScreenNext(context.Context, TerminalSurfaceRequest) (TerminalSurfaceResult, error)
}

// TerminalEventSource streams daemon-authoritative attachment count and
// resize-owner projections for one endpoint.
type TerminalEventSource interface {
	WatchTerminalEvents(context.Context, state.EndpointID) (<-chan TerminalLiveEvent, error)
}

// TerminalSurfaceResult 返回当前 live native screen projection，不包含 committed history truth。
type TerminalSurfaceResult struct {
	Snapshot state.LiveSurfaceSnapshot
	Ready    bool
	// 中文说明：只标记这一次 service result 来自 core lifecycle 查询；不要写进 TUI store。
	LifecycleKnown bool
}

// TerminalSurfaceRequest 请求 owning daemon 的最新 live surface snapshot。
type TerminalSurfaceRequest struct {
	EndpointID       state.EndpointID
	TerminalID       string
	Cols             int
	Rows             int
	ObservedRevision uint64
}

// TerminalLiveEvent 是 daemon 事件总线进入 TUI effect 的 lifecycle/metadata 投影。
// 普通 Refresh 只作为被动失效提示；连续 live screen 由 LiveScreenSource 独立拉取。
type TerminalLiveEvent struct {
	EndpointID state.EndpointID
	TerminalID string
	Snapshot   state.LiveSurfaceSnapshot
	Refresh    bool
	// 中文说明：只标记这一次 event 承载 core lifecycle 变化；reducer 用完即丢。
	LifecycleKnown       bool
	Exited               bool
	ExitCode             int
	ExitedAt             time.Time
	Command              []string
	Reason               string
	Tags                 map[string]string
	Metadata             bool
	AttachmentProjection bool
	AttachmentCount      int
	OwnerSurfaceID       string
	OwnerViewID          string
	ResizeEpoch          uint64
	SizeLocked           bool
	Cols                 int
	Rows                 int
	Err                  error
	Ready                bool
}

package port

import (
	"context"
	"errors"
	"time"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
)

// TerminalService 是 TUI terminal control effect 使用的 application port。
// 实现必须把请求路由到 owning endpoint runtime，不能持有 terminal lifecycle 或隐式选择其它 endpoint。
type TerminalService interface {
	Attach(context.Context, TerminalAttachRequest) (TerminalAttachResult, error)
	Detach(context.Context, TerminalDetachRequest) error
	List(context.Context, TerminalListRequest) (TerminalListResult, error)
	Create(context.Context, TerminalCreateRequest) (TerminalCreateResult, error)
	Restart(context.Context, TerminalRestartRequest) error
	Reconnect(context.Context, TerminalReconnectRequest) (TerminalAttachResult, error)
	Kill(context.Context, TerminalKillRequest) error
	Remove(context.Context, TerminalRemoveRequest) error
	EditMetadata(context.Context, TerminalEditMetadataRequest) error
	EditTags(context.Context, TerminalEditTagsRequest) error
	SendInput(context.Context, TerminalInputRequest) error
	Resize(context.Context, TerminalResizeRequest) (TerminalResizeResult, error)
	ResizeLock(context.Context, TerminalResizeLockRequest) (TerminalResizeResult, error)
}

// PathService 是 TUI/client 对 endpoint daemon 文件系统只读查询的 service 边界。
// TerminalPoolItem 是 owning daemon terminal list 的 TUI application projection。
// EndpointID 由 client runtime adapter 回填，lifecycle 字段仍以 daemon 返回结果为准。
type TerminalPoolItem struct {
	EndpointID      state.EndpointID
	TerminalID      string
	Title           string
	State           string
	CWD             string
	Command         []string
	Tags            map[string]string
	ExitCode        *int
	ExitedAt        time.Time
	Cols            int
	Rows            int
	AttachmentCount int
	Resources       TerminalResourceUsage
	ResourceHistory []TerminalResourceUsage
}

// TerminalResourceUsage 是 protocol adapter 从 core protocol 映射来的 terminal 资源诊断投影；
// 真值来自 core-v2 TerminalProcess 的 OS 采样，adapter 不缓存也不推断进程状态。
type TerminalResourceUsage struct {
	PID            int
	CPUPercentX100 int
	MemoryBytes    uint64
	SampledAt      time.Time
}

// TerminalListRequest 请求指定 endpoint 的 daemon terminal inventory。
type TerminalListRequest struct {
	EndpointID state.EndpointID
}

// TerminalListResult 返回单 endpoint terminal inventory，不合并其它 endpoint 的缓存状态。
type TerminalListResult struct {
	Items []TerminalPoolItem
}

// TerminalCreateRequest 描述在 owning endpoint daemon 创建 terminal 的用户意图。
type TerminalCreateRequest struct {
	EndpointID state.EndpointID
	TerminalID string
	Title      string
	Command    []string
	CWD        string
	Tags       map[string]string
	Cols       int
	Rows       int
}

// TerminalCreateResult 返回 daemon 创建的 terminal identity 和初始尺寸。
type TerminalCreateResult struct {
	EndpointID state.EndpointID
	TerminalID string
	State      string
}

// TerminalAttachRequest 描述一次 view/surface 到 daemon terminal 的 attach 请求。
type TerminalAttachRequest struct {
	EndpointID   state.EndpointID
	TerminalID   string
	Cols         int
	Rows         int
	Mode         string
	ResizePolicy string
	SurfaceID    string
	ViewID       string
	OperationID  string
}

// TerminalDetachRequest 精确释放既有 channel/view binding，不得触发 lazy reconnect。
type TerminalDetachRequest struct {
	EndpointID  state.EndpointID
	TerminalID  string
	Channel     uint16
	SurfaceID   string
	ViewID      string
	Session     *apipb.EndpointSessionStamp
	OperationID string
}

// TerminalAttachResult 是 daemon 确认后的 attachment projection；channel 和 resize control 都来自 owning daemon。
type TerminalAttachResult struct {
	EndpointID      state.EndpointID
	TerminalID      string
	Channel         uint16
	Cols            int
	Rows            int
	CanResize       bool
	SizeLocked      bool
	ControlReason   string
	OwnerSurfaceID  string
	OwnerViewID     string
	ResizeEpoch     uint64
	ResizePolicy    string
	SurfaceID       string
	ViewID          string
	AttachmentCount int
	Session         *apipb.EndpointSessionStamp
	OperationID     string
}

// TerminalRestartRequest 请求 owning daemon 按其 lifecycle 规则重启已退出 terminal。
type TerminalRestartRequest struct {
	EndpointID state.EndpointID
	TerminalID string
}

// TerminalReconnectRequest 为已有 terminal view 建立新的 attachment，不改变 terminal identity。
type TerminalReconnectRequest struct {
	EndpointID   state.EndpointID
	TerminalID   string
	Cols         int
	Rows         int
	Mode         string
	ResizePolicy string
	SurfaceID    string
	ViewID       string
	OperationID  string
}

// TerminalKillRequest 请求 owning daemon 终止 terminal process。
type TerminalKillRequest struct {
	EndpointID state.EndpointID
	TerminalID string
}

// TerminalRemoveRequest 请求 owning daemon 删除已退出 terminal 记录。
type TerminalRemoveRequest struct {
	EndpointID state.EndpointID
	TerminalID string
}

// TerminalEditMetadataRequest 请求 owning daemon 更新 terminal title/tags metadata。
type TerminalEditMetadataRequest struct {
	EndpointID state.EndpointID
	TerminalID string
	Title      string
	Tags       map[string]string
}

// TerminalEditTagsRequest 请求 owning daemon 原子替换 terminal tags。
type TerminalEditTagsRequest struct {
	EndpointID state.EndpointID
	TerminalID string
	Tags       map[string]string
}

// TerminalInputRequest 携带用户输入 bytes 和原始 attachment identity；失败后不得自动重放。
type TerminalInputRequest struct {
	EndpointID  state.EndpointID
	TerminalID  string
	Channel     uint16
	SurfaceID   string
	ViewID      string
	Event       input.InputEvent
	Bytes       []byte
	Session     *apipb.EndpointSessionStamp
	OperationID string
}

// TerminalResizeRequest 携带 owner view 的 resize intent；最终尺寸与 ownership 由 daemon 确认。
type TerminalResizeRequest struct {
	EndpointID         state.EndpointID
	TerminalID         string
	Channel            uint16
	Cols               int
	Rows               int
	ResizePolicy       string
	TakeOwnership      bool
	ExpectedOwnerEpoch uint64
	SurfaceID          string
	ViewID             string
	Session            *apipb.EndpointSessionStamp
	OperationID        string
}

type TerminalResizeLockRequest struct {
	EndpointID  state.EndpointID
	TerminalID  string
	Channel     uint16
	Locked      bool
	SurfaceID   string
	ViewID      string
	Session     *apipb.EndpointSessionStamp
	OperationID string
}

// TerminalResizeResult 返回 daemon 对 resize ownership、epoch 和最终尺寸的确认。
type TerminalResizeResult struct {
	EndpointID      state.EndpointID
	TerminalID      string
	Cols            int
	Rows            int
	Resized         bool
	CanResize       bool
	SizeLocked      bool
	ControlReason   string
	OwnerSurfaceID  string
	OwnerViewID     string
	ResizeEpoch     uint64
	ResizePolicy    string
	SurfaceID       string
	ViewID          string
	AttachmentCount int
	Session         *apipb.EndpointSessionStamp
	OperationID     string
}

var ErrMissingTerminalClient = errors.New("missing terminal client")

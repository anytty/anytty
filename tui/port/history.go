package port

import (
	"context"
	"errors"

	"github.com/anytty/anytty/tui/state"
)

// RequestID 标识一个异步 service 请求。
type RequestID uint64

// Valid 表示 request id 能否对应一个 in-flight 请求。
func (id RequestID) Valid() bool {
	return id != 0
}

// HistoryLatestRequest 请求 owning daemon 建立或读取最新 authoritative history window。
type HistoryLatestRequest struct {
	EndpointID state.EndpointID
	RequestID  RequestID
	PaneID     string
	ViewID     string
	TerminalID string
	Cols       int
	Rows       int
	// GenerationBoundary 是显式请求历史 generation 上界的可选参数；
	// copy 入口不能用 live surface revision 填它，frozen snapshot 边界由 core 建立。
	GenerationBoundary uint64
}

// HistoryOlderRequest 使用 core 返回的 token/cursor 向更旧方向分页，不允许 TUI 自行推断历史总量。
type HistoryOlderRequest struct {
	EndpointID state.EndpointID
	RequestID  RequestID
	PaneID     string
	ViewID     string
	TerminalID string
	Cols       int
	Rows       int
	Token      string
	Generation uint64
	Cursor     state.HistoryCursor
	Boundary   state.HistoryBoundary
}

// HistoryNewerRequest 使用同一 frozen generation 向较新方向分页。
type HistoryNewerRequest struct {
	EndpointID state.EndpointID
	RequestID  RequestID
	PaneID     string
	ViewID     string
	TerminalID string
	Cols       int
	Rows       int
	Token      string
	Generation uint64
	Cursor     state.HistoryCursor
	Boundary   state.HistoryBoundary
}

// HistoryOldestRequest 在既有 frozen history boundary 内定位最旧窗口。
type HistoryOldestRequest struct {
	EndpointID state.EndpointID
	RequestID  RequestID
	PaneID     string
	ViewID     string
	TerminalID string
	Cols       int
	Rows       int
	Token      string
	Generation uint64
	Boundary   state.HistoryBoundary
}

// HistoryReleaseRequest 显式释放 owning daemon 持有的 history token 资源。
type HistoryReleaseRequest struct {
	EndpointID state.EndpointID
	TerminalID string
	Token      string
}

// HistoryCopyRangeRequest 请求 core 按 authoritative logical position 复制文本，不从 render rows 反推内容。
type HistoryCopyRangeRequest struct {
	EndpointID state.EndpointID
	TerminalID string
	Cols       int
	Token      string
	Generation uint64
	Boundary   state.HistoryBoundary
	Start      state.CopyLogicalPosition
	End        state.CopyLogicalPosition
}

type HistorySearchDirection string

const (
	HistorySearchForward  HistorySearchDirection = "forward"
	HistorySearchBackward HistorySearchDirection = "backward"
)

type HistorySearchRequest struct {
	EndpointID state.EndpointID
	RequestID  RequestID
	TerminalID string
	Cols       int
	Rows       int
	Token      string
	Generation uint64
	Query      string
	Mode       state.HistorySearchMode
	Direction  HistorySearchDirection
	Start      state.CopyLogicalPosition
}

// HistoryResult 把异步 request identity 与 authoritative history window 一起回投 reducer。
type HistoryResult struct {
	RequestID RequestID
	Window    state.HistoryWindow
}

// HistoryCopyRangeResult 是 core 对 copy range 的稳定文本投影。
type HistoryCopyRangeResult struct {
	Text string
}

type HistorySearchResult struct {
	RequestID RequestID
	Found     bool
	Start     state.CopyLogicalPosition
	End       state.CopyLogicalPosition
	Window    state.HistoryWindow
	Wrapped   bool
}

// CoreClient 是 TUI copy/history effect 使用的 application port。
// 实现必须路由到 owning endpoint runtime，并保持 core-v2 HistoryWindow 为唯一历史真值。
type CoreClient interface {
	HistoryLatest(context.Context, HistoryLatestRequest) (HistoryResult, error)
	HistoryOlder(context.Context, HistoryOlderRequest) (HistoryResult, error)
	HistoryNewer(context.Context, HistoryNewerRequest) (HistoryResult, error)
	HistoryOldest(context.Context, HistoryOldestRequest) (HistoryResult, error)
	HistoryCopyRange(context.Context, HistoryCopyRangeRequest) (HistoryCopyRangeResult, error)
	HistorySearch(context.Context, HistorySearchRequest) (HistorySearchResult, error)
	ReleaseHistory(context.Context, HistoryReleaseRequest) error
}

var (
	ErrMissingHistoryResponse   = errors.New("missing history response")
	ErrUnexpectedHistoryCall    = errors.New("unexpected history call")
	ErrStaleHistoryWindow       = errors.New("stale history window")
	ErrHistoryResourceExhausted = errors.New("history resources are exhausted")
	ErrHistoryWindowTooLarge    = errors.New("history line is too large to display")
	ErrHistoryCopyTooLarge      = errors.New("history selection is too large to copy")
)

// Package terminal 定义 access 路由层使用的 terminal provider 契约。
//
// access 终结客户端 access wire 后，把终端能力路由给一个 Provider。当前实现是
// pool provider（owner-only 本地 socket）；tmux/zellij 等翻译型 provider 以后
// 实现同一契约。Provider 不持有 DeviceIdentity/AccessStore，也不做鉴权：调用方
// （access/server）已经完成本地信任边界或 remoteauth。
//
// 契约直接运输 providerv1 domain DTO：Create/List/.../History*/LiveScreen/
// Events 是 typed facade；access/server 负责 apipb ↔ providerv1 投影与
// access-issued token 重签。
package terminal

import (
	"context"
	"errors"

	providerv1 "github.com/anytty/anytty/proto/provider/v1"
)

// ErrUnsupported 表示 provider 不支持该能力（例如未实现的 tmux provider）。
// access/server 必须把它映射为 typed unavailable，而不是内部错误。
var ErrUnsupported = errors.New("terminal provider capability is unsupported")

// typed 领域错误：provider 实现把低层 wire 错误投影成这三个哨兵，access/server
// 再映射为既有 ApiErrorCode，不新增客户端可见错误码。
var (
	// ErrNotFound 表示 terminal/attachment/history token 不存在或已失效。
	ErrNotFound = errors.New("terminal provider: terminal not found")
	// ErrConflict 表示生命周期状态冲突（例如名字占用、epoch fence 失败）。
	ErrConflict = errors.New("terminal provider: conflict")
	// ErrUnavailable 表示 provider 连接/执行面暂时不可用；可重试。
	ErrUnavailable = errors.New("terminal provider: unavailable")
)

// StreamSyncLostError 表示 provider 报告 PTY 输出有缺口；调用方转发 sync-lost
// 后应继续读取，流可能随即以 StreamClosedError 结束。
type StreamSyncLostError struct {
	// DroppedBytes 是 provider 报告的丢弃字节数。
	DroppedBytes uint64
}

func (err *StreamSyncLostError) Error() string {
	return "terminal provider: attachment output was lost"
}

// StreamClosedError 表示 provider 关闭了 PTY 流，并携带退出码；ExitCode 为 -1
// 时表示 provider 未报告。调用方应把它投影为 stream closed 帧后停止读取。
type StreamClosedError struct {
	// ExitCode 是 provider 报告的终端退出码。
	ExitCode int
}

func (err *StreamClosedError) Error() string {
	return "terminal provider: attachment stream closed"
}

// CodedError 保留 provider wire 的错误码与可重试标记。access/server 用它映射
// 既有 ApiErrorCode，从而不需要依赖 pool 的 provider 实现包。
type CodedError struct {
	// Code 是 provider wire 错误码（internal/providerproto 的 Error* 常量）。
	Code uint32
	// Message 是 provider 返回的可读信息。
	Message string
	// Retryable 表示同一请求可以重试。
	Retryable bool
}

func (err *CodedError) Error() string {
	if err == nil {
		return ""
	}
	return err.Message
}

// Info 描述 provider 实例的静态身份。
type Info struct {
	// Kind 是 provider 类型标识，例如 "pool"、"tmux"。
	Kind string
	// Version 是 provider 实现/协议代际，例如 "provider/v1"。
	Version string
	// EndpointID 是 provider endpoint 标识（诊断用；pool 实现为本地 socket 路径）。
	EndpointID string
}

// Capabilities 声明 provider 支持的能力面。access/server 在路由前用它做
// fail-fast；未声明能力的 typed 方法应返回 ErrUnsupported。
type Capabilities struct {
	Lifecycle  bool
	Metadata   bool
	Attach     bool
	ResizeLock bool
	History    bool
	Live       bool
	Events     bool
	Paths      bool
}

// DialFunc 为每个客户端会话建立一个 Provider。conn 生命周期与返回的
// Provider 一致：会话结束时由 access/server 调用 Close。
type DialFunc func(ctx context.Context) (Provider, error)

// Duplex 是 attachment 的 PTY 双向字节流：Receive 返回 PTY 输出，Send 写入
// PTY 输入。bootstrap/ready 等 provider 流控帧对调用方不可见（WaitReady 可显式
// 等待建流完成）；流缺口以 StreamSyncLostError 报告，provider 关闭以
// StreamClosedError 报告（含退出码）。Close 只停止当前输出流，attachment
// resource 由 Attachment.Detach 释放。
type Duplex interface {
	// WaitReady 等待 bootstrap/ready 完成；幂等。Receive/Send 也会隐式完成握手。
	WaitReady(ctx context.Context) error
	Receive(ctx context.Context) ([]byte, error)
	Send(ctx context.Context, data []byte) error
	Close() error
}

// Attachment 是一个已建立的 provider 附件。Handle/Size/OwnerEpoch/ResizeControl
// 返回当前 authoritative 快照；Stream 返回 PTY 字节流；Resize 协调 size
// ownership；ResizeLock 修改 owner size lock。
type Attachment interface {
	Handle() *providerv1.AttachmentHandle
	Size() *providerv1.Size
	OwnerEpoch() uint64
	ResizeControl() *providerv1.ResizeControl
	Stream() Duplex
	Resize(ctx context.Context, size *providerv1.Size, policy providerv1.ResizePolicy, takeOwnership bool, expectedEpoch uint64) (*providerv1.TerminalResizeResult, error)
	ResizeLock(ctx context.Context, locked bool) (*providerv1.TerminalResizeResult, error)
	Detach(ctx context.Context) error
}

// MetadataPatch 是 SetMetadata 的字段级更新载荷：nil 字段保留 provider 当前值，
// 非 nil 字段与当前值一起原子替换（provider wire 的 SetMetadata 是整体更新，
// adapter 先读取当前快照补齐缺省字段）。
type MetadataPatch struct {
	Name *string
	Tags *map[string]string
}

// TagsPatch 是 SetTags 的载荷：默认在现有 tags 上先按 Remove 删除、再合并 Set；
// Replace 为 true 时忽略现有值，直接把 Set 作为完整 tag map 提交（客户端
// terminal.set-tags 的整体替换语义）。两种形态最终都走 provider wire 的 SetTags。
type TagsPatch struct {
	Set     map[string]string
	Remove  []string
	Replace bool
}

// AttachRequest 是 Attach 的 typed 参数。Size/ExpectedOwnerEpoch 目前只是契约
// 预留：provider wire 的 TerminalAttachCommand 不承载它们，pool adapter 会忽略；
// 需要 initial size 时在 attach 后调用 Attachment.Resize。
type AttachRequest struct {
	TerminalID         string
	Mode               providerv1.AttachmentMode
	ResizePolicy       providerv1.ResizePolicy
	Size               *providerv1.Size
	SurfaceID          string
	ViewID             string
	ExpectedOwnerEpoch uint64
}

// HistoryRequest 是 HistoryWindow 的 typed 参数，字段与 providerv1
// HistoryWindowCommand 对齐（terminal 单独传，range 命名为 Window）。
type HistoryRequest struct {
	TerminalID          string
	Mode                providerv1.HistoryWindowMode
	Limit               int32
	Cols                int32
	Token               string
	HistoryGeneration   uint64
	BeforeCursor        *providerv1.HistoryCursor
	AfterCursor         *providerv1.HistoryCursor
	BoundaryFirstLineID uint64
	BoundaryLastLineID  uint64
	Window              *providerv1.HistoryRange
}

// Command 把 typed 参数投影为 provider wire 命令；TerminalID 始终以本结构为准。
func (request HistoryRequest) Command() *providerv1.HistoryWindowCommand {
	return &providerv1.HistoryWindowCommand{
		Terminal:            &providerv1.TerminalRef{TerminalId: request.TerminalID},
		Mode:                request.Mode,
		Limit:               request.Limit,
		Cols:                request.Cols,
		Token:               request.Token,
		HistoryGeneration:   request.HistoryGeneration,
		BeforeCursor:        request.BeforeCursor,
		AfterCursor:         request.AfterCursor,
		BoundaryFirstLineId: request.BoundaryFirstLineID,
		BoundaryLastLineId:  request.BoundaryLastLineID,
		Range:               request.Window,
	}
}

// HistoryRequestFromCommand 从 provider wire 命令还原 typed 参数。
func HistoryRequestFromCommand(command *providerv1.HistoryWindowCommand) HistoryRequest {
	if command == nil {
		return HistoryRequest{}
	}
	return HistoryRequest{
		TerminalID:          command.GetTerminal().GetTerminalId(),
		Mode:                command.GetMode(),
		Limit:               command.GetLimit(),
		Cols:                command.GetCols(),
		Token:               command.GetToken(),
		HistoryGeneration:   command.GetHistoryGeneration(),
		BeforeCursor:        command.GetBeforeCursor(),
		AfterCursor:         command.GetAfterCursor(),
		BoundaryFirstLineID: command.GetBoundaryFirstLineId(),
		BoundaryLastLineID:  command.GetBoundaryLastLineId(),
		Window:              command.GetRange(),
	}
}

// HistoryCopyRequest 是 HistoryCopy 的 typed 参数。Window 为 nil 时按 provider
// 默认窗口复制；非 nil 时忽略其 TerminalID，以本结构的 TerminalID 为准。
type HistoryCopyRequest struct {
	TerminalID string
	Window     *HistoryRequest
	MaxLines   int32
	MaxBytes   int32
}

// HistoryCopyRequestFromCommand 从 provider wire 命令还原 typed 参数。
func HistoryCopyRequestFromCommand(command *providerv1.HistoryCopyCommand) HistoryCopyRequest {
	if command == nil {
		return HistoryCopyRequest{}
	}
	var window *HistoryRequest
	if command.GetWindow() != nil {
		converted := HistoryRequestFromCommand(command.GetWindow())
		window = &converted
	}
	return HistoryCopyRequest{
		TerminalID: command.GetTerminal().GetTerminalId(),
		Window:     window,
		MaxLines:   command.GetMaxLines(),
		MaxBytes:   command.GetMaxBytes(),
	}
}

// HistorySearchRequest 是 HistorySearch 的 typed 参数，字段与 providerv1
// HistorySearchCommand 对齐。
type HistorySearchRequest struct {
	TerminalID        string
	Token             string
	HistoryGeneration uint64
	Query             string
	Direction         providerv1.HistorySearchDirection
	Cols              int32
	Limit             int32
	Start             *providerv1.HistoryTextPosition
	Mode              providerv1.HistorySearchMode
	ContextBefore     int32
	Scan              bool
	MaxMatches        int32
}

// Command 把 typed 参数投影为 provider wire 命令；TerminalID 始终以本结构为准。
func (request HistorySearchRequest) Command() *providerv1.HistorySearchCommand {
	return &providerv1.HistorySearchCommand{
		Terminal:          &providerv1.TerminalRef{TerminalId: request.TerminalID},
		Token:             request.Token,
		HistoryGeneration: request.HistoryGeneration,
		Query:             request.Query,
		Direction:         request.Direction,
		Cols:              request.Cols,
		Limit:             request.Limit,
		Start:             request.Start,
		Mode:              request.Mode,
		ContextBefore:     request.ContextBefore,
		Scan:              request.Scan,
		MaxMatches:        request.MaxMatches,
	}
}

// HistorySearchRequestFromCommand 从 provider wire 命令还原 typed 参数。
func HistorySearchRequestFromCommand(command *providerv1.HistorySearchCommand) HistorySearchRequest {
	if command == nil {
		return HistorySearchRequest{}
	}
	return HistorySearchRequest{
		TerminalID:        command.GetTerminal().GetTerminalId(),
		Token:             command.GetToken(),
		HistoryGeneration: command.GetHistoryGeneration(),
		Query:             command.GetQuery(),
		Direction:         command.GetDirection(),
		Cols:              command.GetCols(),
		Limit:             command.GetLimit(),
		Start:             command.GetStart(),
		Mode:              command.GetMode(),
		ContextBefore:     command.GetContextBefore(),
		Scan:              command.GetScan(),
		MaxMatches:        command.GetMaxMatches(),
	}
}

// Provider 是一条 provider 会话。实现由 DialFactory 为每个客户端会话建立，
// 会话生命周期与客户端连接一致；一个 Provider 只服务一个 access client session。
type Provider interface {
	// Info 返回 provider 实例身份；Capabilities 返回能力声明。
	Info(ctx context.Context) (Info, error)
	Capabilities() Capabilities

	// ---- typed facade（providerv1 domain DTO）--------------------------------

	// Create 创建 terminal 并返回 authoritative 快照。
	Create(ctx context.Context, spec *providerv1.TerminalCreateSpec) (*providerv1.TerminalInfo, error)
	// List 返回 terminal inventory。
	List(ctx context.Context) ([]*providerv1.TerminalInfo, error)
	// Get 返回单个 terminal 快照。
	Get(ctx context.Context, terminalID string) (*providerv1.TerminalInfo, error)
	// Restart 重启 terminal 进程并返回重启后的快照。
	Restart(ctx context.Context, terminalID string) (*providerv1.TerminalInfo, error)
	// Kill 终止 terminal 进程（保留 record）。
	Kill(ctx context.Context, terminalID string) error
	// Remove 删除 terminal record。
	Remove(ctx context.Context, terminalID string) error
	// SetMetadata 按 patch 更新 name/tags 并返回更新后的快照。
	SetMetadata(ctx context.Context, terminalID string, patch MetadataPatch) (*providerv1.TerminalInfo, error)
	// SetTags 按 patch 更新 tags 并返回更新后的快照。
	SetTags(ctx context.Context, terminalID string, patch TagsPatch) (*providerv1.TerminalInfo, error)
	// Attach 建立 attachment；bootstrap/ready 在 Attachment.Stream 内完成。
	Attach(ctx context.Context, request AttachRequest) (Attachment, error)
	// HistoryWindow 读取 history window 并签发 frozen token。
	HistoryWindow(ctx context.Context, request HistoryRequest) (*providerv1.HistoryWindowResult, error)
	// HistoryCopy 复制 frozen history token 的文本。
	HistoryCopy(ctx context.Context, request HistoryCopyRequest) (*providerv1.HistoryCopyResult, error)
	// HistorySearch 搜索 frozen history token。
	HistorySearch(ctx context.Context, request HistorySearchRequest) (*providerv1.HistorySearchResult, error)
	// HistoryRelease 释放 frozen history token。
	HistoryRelease(ctx context.Context, terminalID, token string) error
	// HistoryBacklogStatus 返回 history backlog 诊断投影。
	HistoryBacklogStatus(ctx context.Context, terminalID string) (*providerv1.HistoryBacklogStatusResult, error)
	// LiveScreen 等待 observedRevision 之后的 native screen delta。
	LiveScreen(ctx context.Context, terminalID string, observedRevision uint64) (*providerv1.NativeScreenResult, error)
	// Subscribe 建立 provider event 订阅；Release 由调用方按 token 负责。
	Subscribe(ctx context.Context, command *providerv1.EventSubscribeCommand) (*providerv1.EventSubscriptionResult, error)
	// EventRelease 释放 Subscribe 返回的 provider event 订阅 token。
	EventRelease(ctx context.Context, token []byte) error
	// Events 返回该 provider session 的 raw terminal event 流。
	Events(ctx context.Context) (<-chan *providerv1.TerminalEvent, error)
	// TerminalDefaults 返回 owning pool 机器的 shell/cwd 默认值。
	TerminalDefaults(ctx context.Context) (*providerv1.TerminalDefaults, error)
	// ListDirectories 返回 path completion 窗口；limit<=0 用 provider 默认值。
	ListDirectories(ctx context.Context, path string, limit int32) (*providerv1.PathListDirectoriesResult, error)

	// Done 在 provider 连接终止时关闭；Err 返回终止原因。
	Done() <-chan struct{}
	Err() error
	// Close 释放 provider 连接；必须幂等。
	Close() error
}

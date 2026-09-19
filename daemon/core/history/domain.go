package history

// ScreenSessionID 是旧 protocol/window cursor 兼容字段。linehist 默认路径不再维护
// screen session journal，普通 logical rows 会保持零值。
type ScreenSessionID uint64

// ScreenFrameID 是旧 protocol/window cursor 兼容字段。linehist 默认路径不再维护
// fixed-grid frame journal，普通 logical rows 会保持零值。
type ScreenFrameID uint64

// HistoryToken 标识 core-v2 创建的 history window 或 frozen copy boundary。
// client 可以原样传回，但不能从 token 反推出 history truth。
type HistoryToken string

// LineKind 是 authoritative history row 的语义种类，不是 TUI 渲染样式。
type LineKind string

const (
	LineKindOrdinary            LineKind = "ordinary"
	LineKindScreenFrame         LineKind = "screen-frame"
	LineKindArchivedScreenFrame LineKind = "archived-screen-frame"
	LineKindAltScreenFrame      LineKind = "alt-screen-frame"
)

// ScreenFrame 是 frozen snapshot 的旧 wire 兼容字段。linehist 默认实现不再产出
// screen-backed frame payload，但字段保留以维持协议结构。
type ScreenFrame struct {
	ID         ScreenFrameID
	SessionID  ScreenSessionID
	Kind       LineKind
	Rows       [][]Cell
	ScreenCols int
	ScreenRows int
	SourceSeq  uint64
}

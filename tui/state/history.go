package state

import (
	"errors"
	"sort"
	"strings"
	"time"

	xansi "github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

// RequestID 是 TUI 本地请求 id，只用于关联 async response。
type RequestID uint64

// HistoryRequestKind 区分 latest replace、oldest replace 和 older prepend。
type HistoryRequestKind string

const (
	HistoryRequestLatest HistoryRequestKind = "latest"
	HistoryRequestOldest HistoryRequestKind = "oldest"
	HistoryRequestOlder  HistoryRequestKind = "older"
	HistoryRequestNewer  HistoryRequestKind = "newer"
)

// HistoryWindowOp 是 core-v2 authoritative window 合并方式。
type HistoryWindowOp string

const (
	HistoryWindowReplace HistoryWindowOp = "replace"
	HistoryWindowPrepend HistoryWindowOp = "prepend"
	HistoryWindowAppend  HistoryWindowOp = "append"
)

const (
	HistoryRowKindScreenFrame         = "screen-frame"
	HistoryRowKindArchivedScreenFrame = "archived-screen-frame"
	HistoryRowKindAltScreenFrame      = "alt-screen-frame"
)

const (
	HistoryCursorSegmentCommitted            = "committed"
	HistoryCursorSegmentCurrentPrimaryFrame  = "current-primary-frame"
	HistoryCursorSegmentArchivedPrimaryFrame = "archived-primary-frame"
	HistoryCursorSegmentCurrentAltFrame      = "current-alt-frame"
)

// HistoryCursor 是 copy/history 分页的 logical boundary。
// BeforeLineID 是 core authoritative logical-line 坐标；BeforeRowInLine 只描述
// 当前本地 reflow 行内位置。
type HistoryCursor struct {
	Valid           bool
	BeforeLineID    uint64
	BeforeRowInLine int
	Segment         string
}

// HistoryBoundary 绑定 window 的 logical line 边界。
type HistoryBoundary struct {
	FirstLineID uint64
	LastLineID  uint64
}

type HistoryViewportAnchor struct {
	TopLineID     uint64
	TopCellOffset int
	AtEnd         bool
	ScreenCols    int
	ScreenRows    int
	Valid         bool
}

// HistoryPendingRequest 保存 reducer 接纳 response 所需的本地 pending 状态。
type HistoryPendingRequest struct {
	ID              RequestID
	Kind            HistoryRequestKind
	PaneID          string
	ViewID          string
	EndpointID      EndpointID
	TerminalID      string
	Cols            int
	Token           string
	Generation      uint64
	Cursor          HistoryCursor
	Boundary        HistoryBoundary
	BoundCopyModeID uint64
	// 分页返回后继续消费用户越过当前本地窗口、但尚未完成的滚动行数。
	DeferredScrollRows int
}

// HistoryRow 是 authoritative HistoryWindow 的 visual row 投影。
type HistoryRow struct {
	Text         string
	Cells        []HistoryCell
	TailFill     *HistoryCellStyle
	LineID       uint64
	RowInLine    int
	Kind         string
	Segment      string
	SessionID    uint64
	FrameID      uint64
	FixedGrid    bool
	ScreenCols   int
	ScreenRow    int
	ScreenRowSet bool
	ClippedStart bool
	ClippedEnd   bool
	LiveTail     bool
	UpdatedAt    time.Time
}

// HistoryCell 是 core-v2 authoritative HistoryWindow 的 styled cell 投影。
// Text 仍保留在 HistoryRow 上作为搜索/复制派生视图，渲染优先消费 Cells。
type HistoryCell struct {
	Text       string
	Width      int
	Style      HistoryCellStyle
	LinkURL    string
	LinkParams string
}

type HistoryCellStyle struct {
	FG            string
	BG            string
	Bold          bool
	Italic        bool
	Underline     bool
	Blink         bool
	Reverse       bool
	Strikethrough bool
}

// HistoryLogicalLine 是 copy/history 冻结快照里的单条 logical line payload。
// TUI 只把它当作 authoritative source，再按当前 pane 宽度本地重排成 Rows。
type HistoryLogicalLine struct {
	Text         string
	Cells        []HistoryCell
	LineID       uint64
	Kind         string
	Segment      string
	SessionID    uint64
	FrameID      uint64
	FixedGrid    bool
	ScreenCols   int
	ScreenRow    int
	ScreenRowSet bool
	TailFill     *HistoryCellStyle
	LiveTail     bool
	UpdatedAt    time.Time
	// clipped 标记表达这条 frozen source 是否只是 authoritative logical line
	// 的局部片段；本地 reflow 时必须继续保留，不能把 partial 片段误当成完整行。
	ClippedBefore bool
	ClippedAfter  bool
}

// HistoryLineSpan 是 authoritative window 中 logical line 到 visual rows 的映射。
type HistoryLineSpan struct {
	LineID        uint64
	StartRow      int
	EndRow        int
	Kind          string
	Segment       string
	SessionID     uint64
	FrameID       uint64
	FixedGrid     bool
	ScreenCols    int
	ScreenRow     int
	ScreenRowSet  bool
	ClippedBefore bool
	ClippedAfter  bool
	UpdatedAt     time.Time
}

// HistoryWindow 是 state 层使用的 core-v2 authoritative window DTO。
type HistoryWindow struct {
	ViewID         string
	PaneID         string
	EndpointID     EndpointID
	TerminalID     string
	Token          string
	Op             HistoryWindowOp
	Cols           int
	SourceLines    []HistoryLogicalLine
	Rows           []HistoryRow
	Lines          []HistoryLineSpan
	Cursor         HistoryCursor
	HasMore        bool
	Generation     uint64
	Boundary       HistoryBoundary
	ViewportAnchor HistoryViewportAnchor
	LoadedLines    int
	TotalLines     int
	ResponseKind   HistoryRequestKind
}

// HistoryStore 只保存 authoritative window、请求状态和 exhausted marker。
type HistoryStore struct {
	ViewID         string
	PaneID         string
	EndpointID     EndpointID
	TerminalID     string
	Token          string
	Cols           int
	SourceLines    []HistoryLogicalLine
	Rows           []HistoryRow
	Lines          []HistoryLineSpan
	Cursor         HistoryCursor
	Generation     uint64
	Boundary       HistoryBoundary
	ViewportAnchor HistoryViewportAnchor
	HasMore        bool
	Exhausted      ExhaustedMarker
	Pending        *HistoryPendingRequest
	LoadedLines    int
	TotalLines     int
}

// ExhaustedMarker 表示某次 older response 证明对应 cursor 已 exhausted。
type ExhaustedMarker struct {
	Valid     bool
	RequestID RequestID
	Token     string
	Cols      int
	Cursor    HistoryCursor
	Boundary  HistoryBoundary
}

// OlderRequestState 汇总 authoritative window 与 pending/exhausted guard 下的 older 请求状态。
type OlderRequestState string

const (
	OlderRequestReady     OlderRequestState = "ready"
	OlderRequestPending   OlderRequestState = "pending"
	OlderRequestExhausted OlderRequestState = "exhausted"
	OlderRequestMissing   OlderRequestState = "missing"
)

// NewerRequestState 汇总本地窗口尾部是否已经被回收，以及能否从 frozen backend 拉回。
type NewerRequestState string

const (
	NewerRequestReady   NewerRequestState = "ready"
	NewerRequestPending NewerRequestState = "pending"
	NewerRequestMissing NewerRequestState = "missing"
)

// CopyModePhase 是 TUI copy mode 的 reducer-owned 阶段。
// entering 只记录 pending request 和输入拦截；frozen 拥有 core frozen token，
// 可用于 older/newer/search/backend copy 合同。
type CopyModePhase string

const (
	CopyModePhaseNone       CopyModePhase = ""
	CopyModeEnteringPending CopyModePhase = "entering-pending"
	CopyModeFrozenHistory   CopyModePhase = "frozen-history"
)

// CopyModeStore 只保存 copy mode 交互态。
type CopyModeStore struct {
	Active   bool
	Entering bool
	// Phase 是 copy/history reducer 的显式阶段；旧测试或恢复态未填时由 PhaseKind 按兼容字段推导。
	Phase      CopyModePhase
	PaneID     string
	ViewID     string
	EndpointID EndpointID
	TerminalID string
	// latest 飞行期间只累计净滚动行数；不能保存输入事件队列或历史文本副本。
	EnteringScrollDelta  int
	ViewportTop          int
	ViewportTailRows     int
	ViewRows             int
	Cursor               CopyPosition
	Mark                 *CopyPosition
	Selection            *CopySelection
	Query                string
	SearchMode           HistorySearchMode
	SearchCursor         int
	Matches              []CopyMatch
	ActiveMatch          int
	SearchEditing        bool
	SearchPending        bool
	SearchScanPending    bool
	SearchScanComplete   bool
	SearchRequestID      RequestID
	SearchMatchStart     CopyLogicalPosition
	SearchMatchEnd       CopyLogicalPosition
	SearchResults        []CopyLogicalMatch
	SearchWrapped        bool
	SearchError          string
	CopyPending          bool
	CopyRequestID        RequestID
	CopyExitAfterSuccess bool
	BoundToken           string
	BoundCols            int
	RequestID            RequestID
	Empty                bool
}

type HistorySearchMode string

const (
	HistorySearchModeText  HistorySearchMode = "text"
	HistorySearchModeGlob  HistorySearchMode = "glob"
	HistorySearchModeRegex HistorySearchMode = "regex"
)

type CopyPosition struct {
	Row int
	// Col 是 authoritative visual row 内的 display cell column，不是 rune index。
	Col int
}

type CopyLogicalPosition struct {
	Valid  bool
	LineID uint64
	// Col 是 logical line 内的 display cell column，用于窗口被裁剪后回 core/backend 拉文本。
	Col int
}

type CopyMatch struct {
	// Start/End 使用 authoritative row + display cell column，允许同一 logical line
	// 的匹配跨越本地 reflow 产生的多个 visual row。
	StartRow int
	StartCol int
	EndRow   int
	EndCol   int
}

type CopyLogicalMatch struct {
	Start CopyLogicalPosition
	End   CopyLogicalPosition
}

type CopySelection struct {
	Anchor        CopyPosition
	Focus         CopyPosition
	LogicalAnchor CopyLogicalPosition
	LogicalFocus  CopyLogicalPosition
}

// HistoryTrimResult 描述一次 TUI 本地 history window 裁剪。
type HistoryTrimResult struct {
	DroppedRowsBefore  int
	DroppedRowsAfter   int
	DroppedLinesBefore int
	DroppedLinesAfter  int
}

var (
	ErrHistoryRequestPending = errors.New("history request pending")
	ErrStaleHistoryResponse  = errors.New("stale history response")
	ErrHistoryWindowMismatch = errors.New("history window mismatch")
	ErrHistorySearchOrder    = errors.New("history search results are not strictly ordered")
)

func historyRowsToLogicalLines(rows []HistoryRow, spans []HistoryLineSpan) []HistoryLogicalLine {
	if len(rows) == 0 {
		return nil
	}
	lines := make([]HistoryLogicalLine, 0, len(rows))
	for index, row := range rows {
		if len(lines) > 0 && sameHistoryRowLogicalLineSource(lines[len(lines)-1], row) {
			lines[len(lines)-1].Text += row.Text
			lines[len(lines)-1].Cells = append(lines[len(lines)-1].Cells, cloneHistoryCells(row.Cells)...)
			if row.TailFill != nil {
				lines[len(lines)-1].TailFill = cloneHistoryCellStyle(row.TailFill)
			}
			if row.Segment != "" {
				lines[len(lines)-1].Segment = row.Segment
			}
			lines[len(lines)-1].LiveTail = lines[len(lines)-1].LiveTail || row.LiveTail
			if row.UpdatedAt.After(lines[len(lines)-1].UpdatedAt) {
				lines[len(lines)-1].UpdatedAt = row.UpdatedAt
			}
			continue
		}
		span, hasSpan := historySpanForRow(spans, index, row)
		lines = append(lines, HistoryLogicalLine{
			Text:          row.Text,
			Cells:         cloneHistoryCells(row.Cells),
			LineID:        row.LineID,
			Kind:          row.Kind,
			Segment:       row.Segment,
			SessionID:     row.SessionID,
			FrameID:       row.FrameID,
			FixedGrid:     row.FixedGrid,
			ScreenCols:    row.ScreenCols,
			ScreenRow:     row.ScreenRow,
			ScreenRowSet:  row.ScreenRowSet,
			TailFill:      cloneHistoryCellStyle(row.TailFill),
			LiveTail:      row.LiveTail,
			UpdatedAt:     row.UpdatedAt,
			ClippedBefore: hasSpan && span.ClippedBefore,
			ClippedAfter:  hasSpan && span.ClippedAfter,
		})
	}
	return lines
}

func historySpanForRow(spans []HistoryLineSpan, rowIndex int, row HistoryRow) (HistoryLineSpan, bool) {
	for _, span := range spans {
		if span.LineID != 0 && row.LineID != 0 && span.LineID != row.LineID {
			continue
		}
		if rowIndex < span.StartRow || rowIndex > span.EndRow {
			continue
		}
		if span.Kind != "" && span.Kind != row.Kind {
			continue
		}
		if span.Segment != "" && span.Segment != row.Segment {
			continue
		}
		if span.SessionID != 0 && span.SessionID != row.SessionID {
			continue
		}
		if span.FrameID != 0 && span.FrameID != row.FrameID {
			continue
		}
		if span.FixedGrid != row.FixedGrid {
			continue
		}
		if span.FixedGrid && span.ScreenCols != 0 && span.ScreenCols != row.ScreenCols {
			continue
		}
		return span, true
	}
	return HistoryLineSpan{}, false
}

func sameHistoryRowLogicalLineSource(line HistoryLogicalLine, row HistoryRow) bool {
	return line.LineID != 0 &&
		line.LineID == row.LineID &&
		line.Kind == row.Kind &&
		line.Segment == row.Segment &&
		line.SessionID == row.SessionID &&
		line.FrameID == row.FrameID &&
		line.FixedGrid == row.FixedGrid &&
		(!line.FixedGrid || line.ScreenCols == row.ScreenCols)
}

func (store HistoryStore) EnsureSourceLines() HistoryStore {
	if len(store.SourceLines) > 0 || len(store.Rows) == 0 {
		return store
	}
	store.SourceLines = historyRowsToLogicalLines(store.Rows, store.Lines)
	store.LoadedLines = len(store.SourceLines)
	return store
}

func (store HistoryStore) TrimRows(startRow int, endRow int) (HistoryStore, HistoryTrimResult) {
	totalRows := len(store.Rows)
	if totalRows == 0 {
		return store, HistoryTrimResult{}
	}
	if startRow < 0 {
		startRow = 0
	}
	if endRow >= totalRows {
		endRow = totalRows - 1
	}
	if startRow <= 0 && endRow >= totalRows-1 {
		return store, HistoryTrimResult{}
	}
	if startRow > endRow {
		return store, HistoryTrimResult{}
	}
	store = store.EnsureSourceLines()
	if len(store.SourceLines) == 0 {
		return store, HistoryTrimResult{}
	}
	startLine := historySourceLineIndexForRow(store.SourceLines, store.Rows[startRow])
	endLine := historySourceLineIndexForRow(store.SourceLines, store.Rows[endRow])
	if startLine < 0 || endLine < startLine {
		return store, HistoryTrimResult{}
	}
	firstKeptRow := historyFirstRowIndexForSource(store.Rows, store.SourceLines[startLine])
	lastKeptRow := historyLastRowIndexForSource(store.Rows, store.SourceLines[endLine])
	if firstKeptRow < 0 || lastKeptRow < firstKeptRow {
		return store, HistoryTrimResult{}
	}
	result := HistoryTrimResult{
		DroppedRowsBefore:  firstKeptRow,
		DroppedRowsAfter:   totalRows - lastKeptRow - 1,
		DroppedLinesBefore: startLine,
		DroppedLinesAfter:  len(store.SourceLines) - endLine - 1,
	}
	boundaryLast := store.Boundary.LastLineID
	// 中文说明：copy/history 的本地 rows 只是当前窗口投影，不是历史 truth。
	// 裁剪时重新分配 slice，确保被丢弃窗口的 backing array 和 cell payload 可以被 GC。
	source := cloneHistoryLogicalLines(store.SourceLines[startLine : endLine+1])
	store.SourceLines = source
	store.Rows, store.Lines = ReflowHistoryLogicalLines(source, store.Cols)
	store.LoadedLines = len(source)
	if len(store.Rows) > 0 {
		store.Boundary.FirstLineID = store.Rows[0].LineID
		if boundaryLast == 0 {
			boundaryLast = store.Rows[len(store.Rows)-1].LineID
		}
		store.Boundary.LastLineID = boundaryLast
		// 中文说明：本地裁剪后 older cursor 只表达 retained logical line。
		// visual row 是本地 reflow 派生值，不能回写成全局 history 坐标。
		store.Cursor = cursorBeforeFirstLocalRow(store.Rows, store.Cursor, result.DroppedLinesBefore)
	}
	return store, result
}

func cursorBeforeFirstLocalRow(rows []HistoryRow, previous HistoryCursor, droppedRowsBefore int) HistoryCursor {
	if len(rows) == 0 {
		return HistoryCursor{}
	}
	first := rows[0]
	if first.LineID == 0 {
		return HistoryCursor{}
	}
	cursor := previous
	cursor.BeforeLineID = first.LineID
	cursor.BeforeRowInLine = first.RowInLine
	cursor.Valid = first.LineID > 1
	if first.Segment != "" {
		cursor.Segment = first.Segment
	}
	return cursor
}

func historyFirstRowIndexForSource(rows []HistoryRow, source HistoryLogicalLine) int {
	for index, row := range rows {
		if sameHistoryRowLogicalLineSource(source, row) {
			return index
		}
	}
	return -1
}

func historyLastRowIndexForSource(rows []HistoryRow, source HistoryLogicalLine) int {
	for index := len(rows) - 1; index >= 0; index-- {
		if sameHistoryRowLogicalLineSource(source, rows[index]) {
			return index
		}
	}
	return -1
}

func historySourceLineIndexForRow(lines []HistoryLogicalLine, row HistoryRow) int {
	for index, line := range lines {
		if sameHistoryRowLogicalLineSource(line, row) {
			return index
		}
	}
	return -1
}

func (store CopyModeStore) BindLatest(paneID string, viewID string, terminalID string, requestID RequestID, cols int, rows int) CopyModeStore {
	return store.BindLatestRef(paneID, viewID, LocalTerminalRef(terminalID), requestID, cols, rows)
}

// BindLatestRef 进入指定 TerminalRef 的 copy/history latest pending 阶段。
// endpoint 是 copy/history token 的路由边界；同名 terminal 在不同 endpoint 下不能共享 pending、token 或 selection。
func (store CopyModeStore) BindLatestRef(paneID string, viewID string, ref TerminalRef, requestID RequestID, cols int, rows int) CopyModeStore {
	ref = ref.Normalize()
	// 中文说明：进入 copy/history 分两步。latest 请求飞行期间只拦截输入，
	// 不读取 live surface，也不把 pane 内容切成 pending 占位；authoritative window 回来后才 Active。
	store.Active = false
	store.Entering = true
	store.Phase = CopyModeEnteringPending
	store.PaneID = paneID
	store.ViewID = viewID
	store.EndpointID = ref.EndpointID
	store.TerminalID = ref.TerminalID
	store.EnteringScrollDelta = 0
	store.ViewportTailRows = 0
	store.RequestID = requestID
	store.BoundCols = cols
	store.ViewRows = rows
	store.Empty = true
	return store
}

// PhaseKind 返回 copy mode 的有效阶段。
// 旧测试和旧恢复态可能只填 Active/Entering/BoundToken，因此这里做只读推导；
// 新写入路径仍必须显式设置 Phase，避免 entering/frozen 语义混淆。
func (store CopyModeStore) PhaseKind() CopyModePhase {
	if store.Phase != CopyModePhaseNone {
		return store.Phase
	}
	if store.Entering {
		return CopyModeEnteringPending
	}
	if store.Active && store.BoundToken != "" {
		return CopyModeFrozenHistory
	}
	if store.Active {
		return CopyModeFrozenHistory
	}
	return CopyModePhaseNone
}

// InputActive 表示 copy mode 是否应该吞掉当前 view 输入。
// entering/frozen 都属于 copy 交互上下文；只有 none 可以把输入继续交给 terminal。
func (store CopyModeStore) InputActive() bool {
	return store.PhaseKind() != CopyModePhaseNone || store.Active || store.Entering
}

// HistoryRenderable 表示 copy-history renderer 可以消费 HistoryStore rows。
// entering 仍走 pending 分支；frozen 必须先通过绑定校验。
func (store CopyModeStore) HistoryRenderable() bool {
	return store.PhaseKind() == CopyModeFrozenHistory
}

// CanSelect 表示当前 frozen window 允许在已接纳 rows 内移动 cursor/选区。
func (store CopyModeStore) CanSelect() bool {
	return store.PhaseKind() == CopyModeFrozenHistory && store.Active
}

// CanCopy 表示当前 frozen window 允许复制本地已接纳 rows。
// frozen token 若需要跨本地裁剪窗口回 backend，仍由 app 层 SelectionNeedsBackend 继续校验 token。
func (store CopyModeStore) CanCopy() bool {
	return store.PhaseKind() == CopyModeFrozenHistory && store.Active
}

// CanSearch 表示当前 frozen window 允许本地搜索已接纳 rows。
func (store CopyModeStore) CanSearch() bool {
	return store.PhaseKind() == CopyModeFrozenHistory && store.Active
}

// CanPageHistory 表示当前 frozen window 允许使用 authoritative older/newer/oldest 合同。
func (store CopyModeStore) CanPageHistory() bool {
	return store.PhaseKind() == CopyModeFrozenHistory && store.Active && store.BoundToken != ""
}

func (store CopyModeStore) acceptHistoryRows(window HistoryWindow, cols int, totalRows int, pendingScroll int) CopyModeStore {
	store.PaneID = window.PaneID
	store.ViewID = window.ViewID
	store.EndpointID = NormalizeEndpointID(window.EndpointID)
	store.TerminalID = window.TerminalID
	if cols <= 0 {
		cols = window.Cols
	}
	store.BoundCols = cols
	if totalRows <= 0 {
		totalRows = len(window.Rows)
	}
	if totalRows > 0 {
		visibleRows := copyVisibleRowsForStore(store)
		cursorRow := totalRows - 1
		viewportTop := maxCopyInt(0, totalRows-visibleRows)
		store.ViewportTailRows = 0
		if anchoredTop, ok := historyViewportAnchorTop(window.Rows, window.ViewportAnchor); ok {
			viewportTop = clampCopyInt(anchoredTop, 0, totalRows)
			visibleHistoryRows := totalRows - viewportTop
			if visibleHistoryRows < visibleRows {
				store.ViewportTailRows = visibleRows - visibleHistoryRows
			}
			if cursorRow >= viewportTop+visibleRows {
				cursorRow = viewportTop + visibleRows - 1
			}
		}
		store.Cursor = CopyPosition{Row: cursorRow}
		store.ViewportTop = viewportTop
		if pendingScroll != 0 {
			store = store.ScrollCursor(pendingScroll, totalRows)
		}
	} else {
		store.ViewportTop = 0
		store.ViewportTailRows = 0
		store.Cursor = CopyPosition{}
	}
	store.Matches = nil
	store.ActiveMatch = 0
	store.Empty = totalRows == 0
	return store
}

func historyViewportAnchorTop(rows []HistoryRow, anchor HistoryViewportAnchor) (int, bool) {
	if !anchor.Valid {
		return 0, false
	}
	if anchor.AtEnd {
		return len(rows), true
	}
	remaining := maxCopyInt(0, anchor.TopCellOffset)
	found := false
	for index, row := range rows {
		if row.LineID != anchor.TopLineID {
			if found {
				break
			}
			continue
		}
		found = true
		if remaining == 0 {
			return index, true
		}
		width := HistoryRowDisplayWidth(row)
		if remaining < width {
			return index, true
		}
		remaining -= width
	}
	return 0, false
}

func (store CopyModeStore) AcceptLatest(window HistoryWindow, cols int, totalRows int) CopyModeStore {
	store.Active = true
	store.Entering = false
	store.Phase = CopyModeFrozenHistory
	pendingScroll := store.EnteringScrollDelta
	store.EnteringScrollDelta = 0
	store.RequestID = 0
	store.BoundToken = window.Token
	return store.acceptHistoryRows(window, cols, totalRows, pendingScroll)
}

func (store CopyModeStore) AcceptOldest(window HistoryWindow, cols int, totalRows int) CopyModeStore {
	store.Active = true
	store.Entering = false
	store.Phase = CopyModeFrozenHistory
	store.RequestID = 0
	store.PaneID = window.PaneID
	store.ViewID = window.ViewID
	store.EndpointID = NormalizeEndpointID(window.EndpointID)
	store.TerminalID = window.TerminalID
	store.BoundToken = window.Token
	if cols <= 0 {
		cols = window.Cols
	}
	store.BoundCols = cols
	// 中文说明：oldest 是用户显式跳到最老页，必须从第 0 行开始显示；
	// 不能复用 latest 的尾部定位，否则 `g` 会跳过真正最老的第一屏。
	store.ViewportTop = 0
	store.ViewportTailRows = 0
	store.Cursor = CopyPosition{}
	store.Matches = nil
	store.ActiveMatch = 0
	store.Empty = totalRows == 0
	return store
}

func (store CopyModeStore) AcceptOlder(insertedRows int, before HistoryStore, after HistoryStore, window HistoryWindow, cols int) CopyModeStore {
	store.Phase = CopyModeFrozenHistory
	store.Active = true
	store.Entering = false
	if store.Query == "" && copyModeCanShiftSimpleOlderPrepend(insertedRows, before, after) {
		store = store.shiftAfterOlderPrepend(insertedRows)
	} else if len(before.Rows) > 0 && len(after.Rows) > 0 {
		// 中文说明：older prepend 可能不仅是“顶部多了几行”，也可能在 boundary overlap
		// 处把同一 logical line 的 partial source 合并成新的本地 rows。这里统一按
		// before/after frozen history 做内容重绑，避免 cursor/selection 只按行号平移后偏到错误内容。
		store = store.RebindToReflowedHistory(before, after)
	} else if insertedRows > 0 {
		store = store.shiftAfterOlderPrepend(insertedRows)
	}
	store.BoundToken = window.Token
	if cols <= 0 {
		cols = window.Cols
	}
	store.BoundCols = cols
	store.Empty = false
	return store
}

// RestoreViewportTail restores the frozen live viewport's virtual blank rows
// once newer pagination reaches the original viewport anchor again.
func (store CopyModeStore) RestoreViewportTail(history HistoryStore) CopyModeStore {
	if len(history.Rows) > 0 {
		last := history.Rows[len(history.Rows)-1]
		if last.LineID != history.Boundary.LastLineID || last.ClippedEnd {
			return store
		}
	} else if !history.ViewportAnchor.AtEnd {
		return store
	}
	anchoredTop, ok := historyViewportAnchorTop(history.Rows, history.ViewportAnchor)
	if !ok {
		return store
	}
	visibleHistoryRows := len(history.Rows) - anchoredTop
	store.ViewportTailRows = maxCopyInt(0, copyVisibleRowsForStore(store)-visibleHistoryRows)
	return store
}

// AtFrozenBottom reports whether the cursor and viewport have reached the
// bottom of the frozen snapshot, including its viewport tail.
func (store CopyModeStore) AtFrozenBottom(history HistoryStore) bool {
	if len(history.Rows) == 0 {
		return history.ViewportAnchor.Valid && history.ViewportAnchor.AtEnd
	}
	last := history.Rows[len(history.Rows)-1]
	if last.LineID != history.Boundary.LastLineID || last.ClippedEnd {
		return false
	}
	return store.Cursor.Row >= len(history.Rows)-1 &&
		store.ViewportTop >= copyModeViewportMaxTop(store, len(history.Rows))
}

func copyModeCanShiftSimpleOlderPrepend(insertedRows int, before HistoryStore, after HistoryStore) bool {
	if insertedRows <= 0 || len(before.Rows) == 0 || len(after.Rows) == 0 {
		return false
	}
	if before.Cols != after.Cols || insertedRows+len(before.Rows) != len(after.Rows) {
		return false
	}
	return historyRowsSameAnchor(before.Rows[0], after.Rows[insertedRows]) &&
		historyRowsSameAnchor(before.Rows[len(before.Rows)-1], after.Rows[len(after.Rows)-1])
}

func historyRowsSameAnchor(left HistoryRow, right HistoryRow) bool {
	return left.LineID == right.LineID &&
		left.RowInLine == right.RowInLine &&
		left.Kind == right.Kind &&
		left.Segment == right.Segment &&
		left.SessionID == right.SessionID &&
		left.FrameID == right.FrameID &&
		left.FixedGrid == right.FixedGrid &&
		(!left.FixedGrid || left.ScreenCols == right.ScreenCols) &&
		left.Text == right.Text &&
		left.ClippedStart == right.ClippedStart &&
		left.ClippedEnd == right.ClippedEnd &&
		historyTailFillSameAnchor(left.TailFill, right.TailFill) &&
		historyCellsSameAnchor(left.Cells, right.Cells)
}

func historyTailFillSameAnchor(left *HistoryCellStyle, right *HistoryCellStyle) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func historyCellsSameAnchor(left []HistoryCell, right []HistoryCell) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func (store CopyModeStore) shiftAfterOlderPrepend(insertedRows int) CopyModeStore {
	if insertedRows <= 0 {
		return store
	}
	store.ViewportTop += insertedRows
	store.Cursor.Row += insertedRows
	if store.Mark != nil {
		mark := *store.Mark
		mark.Row += insertedRows
		store.Mark = &mark
	}
	if store.Selection != nil {
		selection := *store.Selection
		selection.Anchor.Row += insertedRows
		selection.Focus.Row += insertedRows
		store.Selection = &selection
	}
	return store
}

func (store CopyModeStore) ApplyDeferredOlderScroll(rows int, totalRows int) CopyModeStore {
	if rows <= 0 {
		return store.FollowCursor(totalRows)
	}
	// older prepend 已经把原 visible top 重新锚到同一条 logical line；
	// 这里继续消费用户在等待分页时还想向上的行数。浏览语义必须先移动
	// copy cursor，只有 cursor 到达可见区边缘时 viewport 才跟随。
	return store.ScrollCursor(-rows, totalRows)
}

func (store CopyModeStore) ApplyHistoryTrim(trim HistoryTrimResult, totalRows int) CopyModeStore {
	if trim.DroppedRowsBefore <= 0 && trim.DroppedRowsAfter <= 0 {
		return store.Scroll(0, totalRows)
	}
	shift := trim.DroppedRowsBefore
	if shift > 0 {
		store.ViewportTop -= shift
	}
	if trim.DroppedRowsAfter > 0 {
		store.ViewportTailRows = 0
	}
	store.Cursor = shiftCopyPositionAfterTrim(store.Cursor, shift, totalRows)
	if store.Mark != nil {
		mark := shiftCopyPositionAfterTrim(*store.Mark, shift, totalRows)
		store.Mark = &mark
	}
	if store.Selection != nil {
		selection := *store.Selection
		selection.Anchor = shiftCopyPositionAfterTrim(selection.Anchor, shift, totalRows)
		selection.Focus = shiftCopyPositionAfterTrim(selection.Focus, shift, totalRows)
		store.Selection = &selection
	}
	store.Matches = shiftCopyMatchesAfterTrim(store.Matches, shift, totalRows)
	if store.ActiveMatch >= len(store.Matches) {
		store.ActiveMatch = maxCopyInt(0, len(store.Matches)-1)
	}
	store.ViewportTop = clampCopyInt(store.ViewportTop, 0, copyModeViewportMaxTop(store, totalRows))
	return store.FollowCursor(totalRows)
}

func shiftCopyPositionAfterTrim(pos CopyPosition, droppedBefore int, totalRows int) CopyPosition {
	pos.Row -= droppedBefore
	if totalRows <= 0 {
		return CopyPosition{}
	}
	pos.Row = clampCopyInt(pos.Row, 0, totalRows-1)
	return pos
}

func shiftCopyMatchesAfterTrim(matches []CopyMatch, droppedBefore int, totalRows int) []CopyMatch {
	if len(matches) == 0 || totalRows <= 0 {
		return nil
	}
	out := make([]CopyMatch, 0, len(matches))
	for _, match := range matches {
		match.StartRow -= droppedBefore
		match.EndRow -= droppedBefore
		if match.EndRow < 0 || match.StartRow >= totalRows {
			continue
		}
		match.StartRow = clampCopyInt(match.StartRow, 0, totalRows-1)
		match.EndRow = clampCopyInt(match.EndRow, 0, totalRows-1)
		out = append(out, match)
	}
	return out
}

func (store CopyModeStore) Resize(cols int, rows int) CopyModeStore {
	store.BoundCols = cols
	store.ViewRows = rows
	return store
}

// RebindToReflowedHistory 在 frozen source 按新 cols 本地重排后，把 copy mode
// 的交互态重新映射回同一段 logical-line 内容，而不是继续沿用旧 visual row/col。
func (store CopyModeStore) RebindToReflowedHistory(before HistoryStore, after HistoryStore) CopyModeStore {
	store.BoundCols = after.Cols
	if len(after.Rows) == 0 {
		store.ViewportTop = 0
		store.Cursor = CopyPosition{}
		store.Mark = nil
		store.Selection = nil
		store.Matches = nil
		store.ActiveMatch = 0
		store.Empty = true
		return store
	}
	store.Empty = false
	store.ViewportTop = reflowViewportTop(before, after, store.ViewportTop)
	reflowedCursor := reflowCopyPosition(before, after, store.Cursor)
	store.Cursor = reflowedCursor
	if store.Mark != nil {
		mark := reflowCopyPosition(before, after, *store.Mark)
		store.Mark = &mark
	}
	if store.Selection != nil {
		selection := *store.Selection
		selection.Anchor = reflowCopyPosition(before, after, selection.Anchor)
		selection.Focus = reflowCopyPosition(before, after, selection.Focus)
		store.Selection = &selection
	}
	store = store.RefreshSearchMatch(after)
	store.Cursor = reflowedCursor
	return store
}

func (store CopyModeStore) SetViewRows(rows int) CopyModeStore {
	store.ViewRows = rows
	return store
}

func (store CopyModeStore) MoveCursor(pos CopyPosition) CopyModeStore {
	store.Cursor = pos
	if store.Mark != nil {
		selection := CopySelection{Anchor: *store.Mark, Focus: pos}
		if store.Selection != nil && store.Selection.Anchor == *store.Mark {
			selection.LogicalAnchor = store.Selection.LogicalAnchor
		}
		store.Selection = &selection
	}
	return store
}

// ScrollCursor 只表达显式 cursor 移动或选择扩展。
// cursor 还在可见区内时只移动 cursor；到达边缘后，viewport 才跟随移动。
func (store CopyModeStore) ScrollCursor(delta int, totalRows int) CopyModeStore {
	if totalRows <= 0 {
		store.ViewportTop = 0
		store.Cursor = CopyPosition{}
		return store
	}
	nextRow := clampCopyInt(store.Cursor.Row+delta, 0, totalRows-1)
	store = store.MoveCursor(CopyPosition{Row: nextRow, Col: store.Cursor.Col})
	return store.FollowCursor(totalRows)
}

// ScrollNewer moves through real history rows first, then scrolls the viewport
// across the frozen live screen's virtual blank tail.
func (store CopyModeStore) ScrollNewer(delta int, totalRows int) (CopyModeStore, int) {
	if delta <= 0 {
		return store, 0
	}
	beforeCursor := store.Cursor.Row
	store = store.ScrollCursor(delta, totalRows)
	consumed := store.Cursor.Row - beforeCursor
	remaining := delta - consumed
	if remaining <= 0 {
		return store, consumed
	}
	beforeTop := store.ViewportTop
	store = store.ScrollViewport(remaining, totalRows)
	return store, consumed + store.ViewportTop - beforeTop
}

// ScrollViewport 只表达显式 viewport 重定位或内部锚点修正。
// 普通用户浏览输入必须优先走 ScrollCursor，避免页面先跳而 cursor 失去 tmux 式位置感。
func (store CopyModeStore) ScrollViewport(delta int, totalRows int) CopyModeStore {
	if totalRows <= 0 {
		store.ViewportTop = 0
		store.Cursor = CopyPosition{}
		return store
	}
	height := copyVisibleRowsForStore(store)
	if height <= 0 {
		height = 1
	}
	maxTop := copyModeViewportMaxTop(store, totalRows)
	offset := store.Cursor.Row - store.ViewportTop
	if offset < 0 {
		offset = 0
	}
	if offset >= height {
		offset = height - 1
	}
	nextTop := clampCopyInt(store.ViewportTop+delta, 0, maxTop)
	store.ViewportTop = nextTop
	store.Cursor = CopyPosition{Row: clampCopyInt(nextTop+offset, 0, totalRows-1), Col: store.Cursor.Col}
	return store.Scroll(0, totalRows)
}

func (store CopyModeStore) FollowCursor(totalRows int) CopyModeStore {
	if totalRows <= 0 {
		store.ViewportTop = 0
		return store
	}
	height := copyVisibleRowsForStore(store)
	if store.Cursor.Row < store.ViewportTop {
		store.ViewportTop = store.Cursor.Row
	}
	if store.Cursor.Row >= store.ViewportTop+height {
		store.ViewportTop = store.Cursor.Row - height + 1
	}
	return store.Scroll(0, totalRows)
}

func (store CopyModeStore) SetMark(pos CopyPosition) CopyModeStore {
	store.Mark = &pos
	store.Selection = &CopySelection{Anchor: pos, Focus: pos}
	return store
}

func (store CopyModeStore) ClearSelection() CopyModeStore {
	store.Mark = nil
	store.Selection = nil
	store.CopyExitAfterSuccess = false
	return store
}

func (store CopyModeStore) RefreshLogicalSelection(history HistoryStore) CopyModeStore {
	if store.Selection == nil {
		return store
	}
	selection := *store.Selection
	selection.LogicalAnchor = CopyLogicalPositionForPosition(history, selection.Anchor)
	selection.LogicalFocus = CopyLogicalPositionForPosition(history, selection.Focus)
	store.Selection = &selection
	return store
}

func (store CopyModeStore) EnsureLogicalSelection(history HistoryStore) CopyModeStore {
	if store.Selection == nil {
		return store
	}
	selection := *store.Selection
	if !selection.LogicalAnchor.Valid {
		selection.LogicalAnchor = CopyLogicalPositionForPosition(history, selection.Anchor)
	}
	if !selection.LogicalFocus.Valid {
		selection.LogicalFocus = CopyLogicalPositionForPosition(history, selection.Focus)
	}
	store.Selection = &selection
	return store
}

func (store CopyModeStore) RefreshLogicalSelectionFocus(history HistoryStore) CopyModeStore {
	if store.Selection == nil {
		return store
	}
	selection := *store.Selection
	selection.LogicalFocus = CopyLogicalPositionForPosition(history, selection.Focus)
	store.Selection = &selection
	return store
}

func (store CopyModeStore) SelectionLogicalRange(history HistoryStore) (CopyLogicalPosition, CopyLogicalPosition, bool) {
	if store.Selection == nil {
		return CopyLogicalPosition{}, CopyLogicalPosition{}, false
	}
	start := store.Selection.LogicalAnchor
	if !start.Valid {
		start = CopyLogicalPositionForPosition(history, store.Selection.Anchor)
	}
	end := store.Selection.LogicalFocus
	if !end.Valid {
		end = CopyLogicalPositionForPosition(history, store.Selection.Focus)
	}
	if !start.Valid || !end.Valid {
		return CopyLogicalPosition{}, CopyLogicalPosition{}, false
	}
	return start, end, true
}

func (store CopyModeStore) SelectionNeedsBackend(history HistoryStore) bool {
	if store.Selection == nil || !store.Selection.LogicalAnchor.Valid || !store.Selection.LogicalFocus.Valid {
		return false
	}
	currentAnchor := CopyLogicalPositionForPosition(history, store.Selection.Anchor)
	currentFocus := CopyLogicalPositionForPosition(history, store.Selection.Focus)
	return !copyLogicalPositionSame(currentAnchor, store.Selection.LogicalAnchor) ||
		!copyLogicalPositionSame(currentFocus, store.Selection.LogicalFocus)
}

func (store CopyModeStore) SetQuery(query string, matches []CopyMatch) CopyModeStore {
	if store.Query != query {
		store.SearchMatchStart = CopyLogicalPosition{}
		store.SearchMatchEnd = CopyLogicalPosition{}
		store.SearchWrapped = false
		store.SearchResults = nil
		store.SearchScanPending = false
		store.SearchScanComplete = false
	}
	store.Query = query
	store.SearchCursor = clampCopyInt(store.SearchCursor, 0, len([]rune(query)))
	store.SearchError = ""
	store.Matches = cloneCopyMatches(matches)
	store.ActiveMatch = 0
	if len(store.Matches) > 0 {
		store.Cursor = CopyPosition{Row: store.Matches[0].StartRow, Col: store.Matches[0].StartCol}
	}
	return store
}

func (store CopyModeStore) SetSearchCursor(cursor int) CopyModeStore {
	store.SearchCursor = clampCopyInt(cursor, 0, len([]rune(store.Query)))
	return store
}

func (store CopyModeStore) MoveSearchCursor(delta int) CopyModeStore {
	return store.SetSearchCursor(store.SearchCursor + delta)
}

func (store CopyModeStore) InsertSearchText(text string) CopyModeStore {
	if text == "" {
		return store
	}
	runes := []rune(store.Query)
	cursor := clampCopyInt(store.SearchCursor, 0, len(runes))
	insert := []rune(text)
	next := make([]rune, 0, len(runes)+len(insert))
	next = append(next, runes[:cursor]...)
	next = append(next, insert...)
	next = append(next, runes[cursor:]...)
	store = store.SetQuery(string(next), nil)
	store.SearchCursor = cursor + len(insert)
	return store
}

func (store CopyModeStore) DeleteSearchBackward() CopyModeStore {
	runes := []rune(store.Query)
	cursor := clampCopyInt(store.SearchCursor, 0, len(runes))
	if cursor == 0 {
		return store
	}
	next := make([]rune, 0, len(runes)-1)
	next = append(next, runes[:cursor-1]...)
	next = append(next, runes[cursor:]...)
	store = store.SetQuery(string(next), nil)
	store.SearchCursor = cursor - 1
	return store
}

func (store CopyModeStore) DeleteSearchForward() CopyModeStore {
	runes := []rune(store.Query)
	cursor := clampCopyInt(store.SearchCursor, 0, len(runes))
	if cursor >= len(runes) {
		return store
	}
	next := make([]rune, 0, len(runes)-1)
	next = append(next, runes[:cursor]...)
	next = append(next, runes[cursor+1:]...)
	store = store.SetQuery(string(next), nil)
	store.SearchCursor = cursor
	return store
}

func (store CopyModeStore) NormalizedSearchMode() HistorySearchMode {
	switch store.SearchMode {
	case HistorySearchModeGlob, HistorySearchModeRegex:
		return store.SearchMode
	default:
		return HistorySearchModeText
	}
}

func (store CopyModeStore) CycleSearchMode() CopyModeStore {
	switch store.NormalizedSearchMode() {
	case HistorySearchModeText:
		store.SearchMode = HistorySearchModeGlob
	case HistorySearchModeGlob:
		store.SearchMode = HistorySearchModeRegex
	default:
		store.SearchMode = HistorySearchModeText
	}
	store.Matches = nil
	store.ActiveMatch = 0
	store.SearchMatchStart = CopyLogicalPosition{}
	store.SearchMatchEnd = CopyLogicalPosition{}
	store.SearchResults = nil
	store.SearchScanPending = false
	store.SearchScanComplete = false
	store.SearchWrapped = false
	store.SearchError = ""
	return store
}

func (store CopyModeStore) SearchBarVisible() bool {
	return store.Active && (store.SearchEditing || store.SearchPending || store.Query != "" || store.SearchError != "")
}

func (store CopyModeStore) CloseSearch() CopyModeStore {
	store.Query = ""
	store.SearchCursor = 0
	store.Matches = nil
	store.ActiveMatch = 0
	store.SearchEditing = false
	store.SearchPending = false
	store.SearchRequestID = 0
	store.SearchMatchStart = CopyLogicalPosition{}
	store.SearchMatchEnd = CopyLogicalPosition{}
	store.SearchResults = nil
	store.SearchScanPending = false
	store.SearchScanComplete = false
	store.SearchWrapped = false
	store.SearchError = ""
	return store
}

func (store CopyModeStore) RefreshSearchMatch(history HistoryStore) CopyModeStore {
	return store.RefreshSearchMatches(history)
}

func (store CopyModeStore) RefreshSearchMatches(history HistoryStore) CopyModeStore {
	store.Matches = nil
	store.ActiveMatch = 0
	activeVisual := -1
	for _, visible := range visibleHistorySearchRanges(history, store) {
		start := sort.Search(len(store.SearchResults), func(index int) bool {
			result := store.SearchResults[index]
			return result.Start.LineID > visible.LineID || result.Start.LineID == visible.LineID && result.Start.Col >= visible.From
		})
		if start > 0 {
			previous := store.SearchResults[start-1]
			if previous.Start.LineID == visible.LineID && previous.End.Col > visible.From {
				start--
			}
		}
		for index := start; index < len(store.SearchResults); index++ {
			result := store.SearchResults[index]
			if result.Start.LineID != visible.LineID || result.Start.Col >= visible.To {
				break
			}
			if result.End.Col <= visible.From {
				continue
			}
			match, ok := copyMatchForLogicalRange(history, result.Start, result.End)
			if !ok {
				continue
			}
			if copyLogicalPositionSame(result.Start, store.SearchMatchStart) && copyLogicalPositionSame(result.End, store.SearchMatchEnd) {
				activeVisual = len(store.Matches)
			}
			store.Matches = append(store.Matches, match)
		}
	}
	if store.SearchMatchStart.Valid && store.SearchMatchEnd.Valid && activeVisual < 0 {
		if match, ok := copyMatchForLogicalRange(history, store.SearchMatchStart, store.SearchMatchEnd); ok {
			activeVisual = len(store.Matches)
			store.Matches = append(store.Matches, match)
		}
	}
	if activeVisual >= 0 {
		store.ActiveMatch = activeVisual
		store.Cursor = CopyPositionForLogicalPosition(history, store.SearchMatchStart)
	}
	return store
}

// AppendSearchResults consumes the strict chronological scan contract. It only
// compares the append boundary, avoiding the quadratic whole-catalog dedupe
// that made high-frequency searches block the reducer.
func (store CopyModeStore) AppendSearchResults(history HistoryStore, results []CopyLogicalMatch) (CopyModeStore, error) {
	for _, result := range results {
		if !result.Start.Valid || !result.End.Valid || compareCopyLogicalPosition(result.Start, result.End) > 0 {
			return store, ErrHistorySearchOrder
		}
		if len(store.SearchResults) > 0 {
			order := compareCopyLogicalMatch(store.SearchResults[len(store.SearchResults)-1], result)
			if order > 0 {
				return store, ErrHistorySearchOrder
			}
			if order == 0 {
				continue
			}
		}
		store.SearchResults = append(store.SearchResults, result)
	}
	return store.RefreshSearchMatches(history), nil
}

type historySearchVisibleRange struct {
	LineID uint64
	From   int
	To     int
}

func visibleHistorySearchRanges(history HistoryStore, store CopyModeStore) []historySearchVisibleRange {
	if len(history.Rows) == 0 {
		return nil
	}
	top := clampCopyInt(store.ViewportTop, 0, len(history.Rows))
	bottom := minCopyInt(len(history.Rows), top+copyVisibleRowsForStore(store))
	ranges := make([]historySearchVisibleRange, 0, bottom-top)
	for rowIndex := top; rowIndex < bottom; rowIndex++ {
		row := history.Rows[rowIndex]
		if row.LineID == 0 {
			continue
		}
		from := historyLogicalOffsetForPosition(history, CopyPosition{Row: rowIndex}).col
		to := from + HistoryRowDisplayWidth(row)
		if len(ranges) > 0 && ranges[len(ranges)-1].LineID == row.LineID {
			ranges[len(ranges)-1].To = maxCopyInt(ranges[len(ranges)-1].To, to)
			continue
		}
		ranges = append(ranges, historySearchVisibleRange{LineID: row.LineID, From: from, To: to})
	}
	return ranges
}

func copyMatchForLogicalRange(history HistoryStore, startLogical CopyLogicalPosition, endLogical CopyLogicalPosition) (CopyMatch, bool) {
	if !startLogical.Valid || !endLogical.Valid {
		return CopyMatch{}, false
	}
	start := CopyPositionForLogicalPosition(history, startLogical)
	end := CopyPositionForLogicalPosition(history, endLogical)
	if !copyLogicalPositionSame(CopyLogicalPositionForPosition(history, start), startLogical) ||
		!copyLogicalPositionSame(CopyLogicalPositionForPosition(history, end), endLogical) {
		return CopyMatch{}, false
	}
	return CopyMatch{StartRow: start.Row, StartCol: start.Col, EndRow: end.Row, EndCol: end.Col}, true
}

func (store CopyModeStore) SearchResultPosition() (int, int) {
	if !store.SearchMatchStart.Valid || !store.SearchMatchEnd.Valid {
		return 0, len(store.SearchResults)
	}
	target := CopyLogicalMatch{Start: store.SearchMatchStart, End: store.SearchMatchEnd}
	index := sort.Search(len(store.SearchResults), func(index int) bool {
		return compareCopyLogicalMatch(store.SearchResults[index], target) >= 0
	})
	if index < len(store.SearchResults) && compareCopyLogicalMatch(store.SearchResults[index], target) == 0 {
		return index + 1, len(store.SearchResults)
	}
	return 0, len(store.SearchResults)
}

func compareCopyLogicalMatch(left CopyLogicalMatch, right CopyLogicalMatch) int {
	if order := compareCopyLogicalPosition(left.Start, right.Start); order != 0 {
		return order
	}
	return compareCopyLogicalPosition(left.End, right.End)
}

func compareCopyLogicalPosition(left CopyLogicalPosition, right CopyLogicalPosition) int {
	if left.LineID < right.LineID {
		return -1
	}
	if left.LineID > right.LineID {
		return 1
	}
	if left.Col < right.Col {
		return -1
	}
	if left.Col > right.Col {
		return 1
	}
	return 0
}

func historyLineSpansForSearch(history HistoryStore) []HistoryLineSpan {
	if len(history.Lines) > 0 {
		return cloneHistoryLineSpans(history.Lines)
	}
	if len(history.Rows) == 0 {
		return nil
	}
	spans := make([]HistoryLineSpan, 0, len(history.Rows))
	start := 0
	for row := 1; row < len(history.Rows); row++ {
		if sameHistoryRowsSource(history.Rows[start], history.Rows[row]) {
			continue
		}
		spans = append(spans, historyLineSpanForLocalRows(history.Rows, start, row-1))
		start = row
	}
	spans = append(spans, historyLineSpanForLocalRows(history.Rows, start, len(history.Rows)-1))
	return spans
}

func sameHistoryRowsSource(left HistoryRow, right HistoryRow) bool {
	return left.LineID != 0 &&
		left.LineID == right.LineID &&
		left.Kind == right.Kind &&
		left.Segment == right.Segment &&
		left.SessionID == right.SessionID &&
		left.FrameID == right.FrameID &&
		left.FixedGrid == right.FixedGrid &&
		(!left.FixedGrid || left.ScreenCols == right.ScreenCols)
}

func historyLineSpanForLocalRows(rows []HistoryRow, start int, end int) HistoryLineSpan {
	if start < 0 || start >= len(rows) {
		return HistoryLineSpan{}
	}
	row := rows[start]
	updatedAt := row.UpdatedAt
	for index := start + 1; index <= end && index < len(rows); index++ {
		if rows[index].UpdatedAt.After(updatedAt) {
			updatedAt = rows[index].UpdatedAt
		}
	}
	return HistoryLineSpan{
		LineID:       row.LineID,
		StartRow:     start,
		EndRow:       end,
		Kind:         row.Kind,
		Segment:      row.Segment,
		SessionID:    row.SessionID,
		FrameID:      row.FrameID,
		FixedGrid:    row.FixedGrid,
		ScreenCols:   row.ScreenCols,
		ScreenRow:    row.ScreenRow,
		ScreenRowSet: row.ScreenRowSet,
		UpdatedAt:    updatedAt,
	}
}

func HistoryRowGraphemeDisplayColumns(row HistoryRow) []int {
	if len(row.Cells) == 0 {
		return textClusterDisplayColumns(textGraphemeClusters(row.Text))
	}
	columns := []int{0}
	cursor := 0
	for _, cell := range row.Cells {
		clusters := textGraphemeClusters(cell.Text)
		if len(clusters) == 0 {
			cursor += HistoryCellDisplayWidth(cell)
			continue
		}
		natural := textClusterDisplayColumns(clusters)
		cellWidth := HistoryCellDisplayWidth(cell)
		naturalWidth := natural[len(natural)-1]
		for index := 1; index < len(natural); index++ {
			columns = append(columns, cursor+scaleDisplayColumn(natural[index], naturalWidth, cellWidth))
		}
		cursor += cellWidth
	}
	return columns
}

func HistoryRowDisplayWidth(row HistoryRow) int {
	if len(row.Cells) == 0 {
		return textDisplayWidth(row.Text)
	}
	width := 0
	for _, cell := range row.Cells {
		width += HistoryCellDisplayWidth(cell)
	}
	return width
}

func HistoryCellDisplayWidth(cell HistoryCell) int {
	if cell.Width > 0 {
		return cell.Width
	}
	return textDisplayWidth(cell.Text)
}

func HistoryCellDisplayWidthSum(cells []HistoryCell) int {
	width := 0
	for _, cell := range cells {
		width += HistoryCellDisplayWidth(cell)
	}
	return width
}

func HistoryRowSliceDisplay(row HistoryRow, from int, to int) string {
	width := HistoryRowDisplayWidth(row)
	from = clampCopyInt(from, 0, width)
	to = clampCopyInt(to, from, width)
	if to <= from {
		return ""
	}
	if len(row.Cells) == 0 {
		return textSliceDisplay(row.Text, from, to)
	}
	var builder strings.Builder
	cursor := 0
	for _, cell := range row.Cells {
		cellWidth := HistoryCellDisplayWidth(cell)
		next := cursor + cellWidth
		if rangesOverlap(cursor, next, from, to) {
			builder.WriteString(historyCellSliceDisplay(cell, from-cursor, to-cursor))
		}
		cursor = next
	}
	return builder.String()
}

func historyCellSliceDisplay(cell HistoryCell, from int, to int) string {
	width := HistoryCellDisplayWidth(cell)
	from = clampCopyInt(from, 0, width)
	to = clampCopyInt(to, from, width)
	if to <= from {
		return ""
	}
	textWidth := textDisplayWidth(cell.Text)
	textPart := ""
	if from < textWidth {
		textEnd := to
		if textEnd > textWidth {
			textEnd = textWidth
		}
		textPart = textSliceDisplay(cell.Text, from, textEnd)
	}
	pad := to - maxCopyInt(from, textWidth)
	if pad <= 0 {
		return textPart
	}
	return textPart + strings.Repeat(" ", pad)
}

func textClusterDisplayColumns(clusters []string) []int {
	columns := make([]int, len(clusters)+1)
	for index, cluster := range clusters {
		columns[index+1] = columns[index] + textDisplayWidth(cluster)
	}
	return columns
}

func scaleDisplayColumn(column int, naturalWidth int, authoritativeWidth int) int {
	if naturalWidth <= 0 || naturalWidth == authoritativeWidth {
		return column
	}
	return (column*authoritativeWidth + naturalWidth - 1) / naturalWidth
}

func textGraphemeClusters(text string) []string {
	if text == "" {
		return nil
	}
	clusters := make([]string, 0, len([]rune(text)))
	graphemes := uniseg.NewGraphemes(text)
	for graphemes.Next() {
		clusters = append(clusters, graphemes.Str())
	}
	return clusters
}

func textDisplayWidth(text string) int {
	return xansi.StringWidth(strings.ReplaceAll(text, "\n", " "))
}

func textSliceDisplay(text string, from int, to int) string {
	if to <= from {
		return ""
	}
	var builder strings.Builder
	cursor := 0
	for _, cluster := range textGraphemeClusters(text) {
		width := textDisplayWidth(cluster)
		next := cursor + width
		if width == 0 {
			next = cursor
		}
		if rangesOverlap(cursor, maxCopyInt(next, cursor+1), from, to) {
			builder.WriteString(cluster)
		}
		cursor += width
	}
	return builder.String()
}

func rangesOverlap(leftFrom int, leftTo int, rightFrom int, rightTo int) bool {
	return leftFrom < rightTo && rightFrom < leftTo
}

func (store CopyModeStore) Scroll(delta int, totalRows int) CopyModeStore {
	maxTop := copyModeViewportMaxTop(store, totalRows)
	store.ViewportTop = clampCopyInt(store.ViewportTop+delta, 0, maxTop)
	return store
}

func copyModeViewportMaxTop(store CopyModeStore, totalRows int) int {
	return maxCopyInt(0, totalRows+maxCopyInt(0, store.ViewportTailRows)-copyVisibleRowsForStore(store))
}

func cloneHistoryLogicalLines(lines []HistoryLogicalLine) []HistoryLogicalLine {
	if len(lines) == 0 {
		return nil
	}
	cloned := make([]HistoryLogicalLine, len(lines))
	copy(cloned, lines)
	for i := range cloned {
		cloned[i].Cells = cloneHistoryCells(lines[i].Cells)
		cloned[i].TailFill = cloneHistoryCellStyle(lines[i].TailFill)
	}
	return cloned
}

func cloneHistoryCellStyle(style *HistoryCellStyle) *HistoryCellStyle {
	if style == nil {
		return nil
	}
	cloned := *style
	return &cloned
}

func cloneHistoryCells(cells []HistoryCell) []HistoryCell {
	if len(cells) == 0 {
		return nil
	}
	cloned := make([]HistoryCell, len(cells))
	copy(cloned, cells)
	return cloned
}

func cloneHistoryLineSpans(spans []HistoryLineSpan) []HistoryLineSpan {
	if len(spans) == 0 {
		return nil
	}
	cloned := make([]HistoryLineSpan, len(spans))
	copy(cloned, spans)
	return cloned
}

// ReflowHistoryLogicalLines 把冻结 logical-line source 重新排成本地可见 rows。
// 这是 TUI copy/history 的唯一重排路径；它不创造新的历史 truth，只产出 view rows。
func ReflowHistoryLogicalLines(lines []HistoryLogicalLine, cols int) ([]HistoryRow, []HistoryLineSpan) {
	if len(lines) == 0 {
		return nil, nil
	}
	if cols <= 0 {
		cols = 80
	}
	rows := make([]HistoryRow, 0, len(lines))
	spans := make([]HistoryLineSpan, 0, len(lines))
	for _, line := range lines {
		lineRows := reflowHistoryLogicalLine(line, cols)
		start := len(rows)
		rows = append(rows, lineRows...)
		end := len(rows) - 1
		if end < start {
			end = start
		}
		spans = append(spans, HistoryLineSpan{
			LineID:        line.LineID,
			StartRow:      start,
			EndRow:        end,
			Kind:          line.Kind,
			Segment:       line.Segment,
			SessionID:     line.SessionID,
			FrameID:       line.FrameID,
			FixedGrid:     line.FixedGrid,
			ScreenCols:    line.ScreenCols,
			ScreenRow:     line.ScreenRow,
			ScreenRowSet:  line.ScreenRowSet,
			ClippedBefore: line.ClippedBefore,
			ClippedAfter:  line.ClippedAfter,
			UpdatedAt:     line.UpdatedAt,
		})
	}
	return rows, spans
}

func reflowHistoryLogicalLine(line HistoryLogicalLine, cols int) []HistoryRow {
	cells := cloneHistoryCells(line.Cells)
	if isFixedGridHistoryRowKind(line.Kind) {
		rows := fixedGridHistoryRows(line, cells)
		applyHistoryLineClipFlags(rows, line)
		return rows
	}
	if len(cells) == 0 && line.Text != "" {
		rows := reflowPlainHistoryLogicalLine(line, cols)
		applyHistoryLineClipFlags(rows, line)
		return rows
	} else if len(cells) > 0 {
		cells = normalizeHistoryLogicalLineCells(cells, cols)
	}
	if len(cells) == 0 {
		rows := []HistoryRow{{LineID: line.LineID, Kind: line.Kind, Segment: line.Segment, SessionID: line.SessionID, FrameID: line.FrameID, FixedGrid: line.FixedGrid, ScreenCols: line.ScreenCols, ScreenRow: line.ScreenRow, ScreenRowSet: line.ScreenRowSet, LiveTail: line.LiveTail, UpdatedAt: line.UpdatedAt}}
		applyHistoryLineClipFlags(rows, line)
		return rows
	}
	rows := make([]HistoryRow, 0, 1)
	current := make([]HistoryCell, 0, len(cells))
	width := 0
	flush := func() {
		row := HistoryRow{
			Text:         historyCellsPlainTextForState(current),
			Cells:        cloneHistoryCells(current),
			LineID:       line.LineID,
			RowInLine:    len(rows),
			Kind:         line.Kind,
			Segment:      line.Segment,
			SessionID:    line.SessionID,
			FrameID:      line.FrameID,
			FixedGrid:    line.FixedGrid,
			ScreenCols:   line.ScreenCols,
			ScreenRow:    line.ScreenRow,
			ScreenRowSet: line.ScreenRowSet,
			LiveTail:     line.LiveTail,
			UpdatedAt:    line.UpdatedAt,
		}
		rows = append(rows, row)
		current = current[:0]
		width = 0
	}
	for _, cell := range cells {
		cellWidth := HistoryCellDisplayWidth(cell)
		if cellWidth <= 0 {
			continue
		}
		parts := []HistoryCell{cell}
		if cellWidth > cols || (width > 0 && width+cellWidth > cols) {
			// 中文说明：cell 小于 pane 宽但放不进当前行剩余列时，也要按
			// grapheme/padding 切开填满当前行；否则 ls 多列输出会提前换行。
			parts = splitHistoryCellForWrap(cell)
		}
		for _, part := range parts {
			partWidth := HistoryCellDisplayWidth(part)
			if partWidth <= 0 {
				continue
			}
			if width > 0 && width+partWidth > cols {
				flush()
			}
			current = append(current, part)
			width += partWidth
			if width >= cols {
				flush()
			}
		}
	}
	if len(current) > 0 || len(rows) == 0 {
		flush()
	}
	applyTailFillToLastHistoryRow(rows, line.TailFill)
	applyHistoryLineClipFlags(rows, line)
	return rows
}

func applyHistoryLineClipFlags(rows []HistoryRow, line HistoryLogicalLine) {
	if len(rows) == 0 {
		return
	}
	if line.ClippedBefore {
		rows[0].ClippedStart = true
	}
	if line.ClippedAfter {
		rows[len(rows)-1].ClippedEnd = true
	}
}

func isFixedGridHistoryRowKind(kind string) bool {
	return kind == HistoryRowKindScreenFrame || kind == HistoryRowKindArchivedScreenFrame || kind == HistoryRowKindAltScreenFrame
}

func fixedGridHistoryRows(line HistoryLogicalLine, cells []HistoryCell) []HistoryRow {
	if len(cells) == 0 && line.Text != "" {
		cells = []HistoryCell{{Text: line.Text, Width: textDisplayWidth(line.Text)}}
	}
	if len(cells) > 0 {
		cells = normalizeHistoryLogicalLineCells(cells, HistoryCellDisplayWidthSum(cells))
	}
	row := HistoryRow{
		Text:         historyCellsPlainTextForState(cells),
		Cells:        cloneHistoryCells(cells),
		LineID:       line.LineID,
		Kind:         line.Kind,
		Segment:      line.Segment,
		SessionID:    line.SessionID,
		FrameID:      line.FrameID,
		FixedGrid:    line.FixedGrid,
		ScreenCols:   line.ScreenCols,
		ScreenRow:    line.ScreenRow,
		ScreenRowSet: line.ScreenRowSet,
		LiveTail:     line.LiveTail,
		UpdatedAt:    line.UpdatedAt,
		TailFill:     cloneHistoryCellStyle(line.TailFill),
	}
	if row.Text == "" && line.Text != "" {
		row.Text = line.Text
	}
	return []HistoryRow{row}
}

func reflowPlainHistoryLogicalLine(line HistoryLogicalLine, cols int) []HistoryRow {
	if singleWidthASCIIText(line.Text) {
		return reflowPlainASCIIHistoryLogicalLine(line, cols)
	}
	clusters := textGraphemeClusters(line.Text)
	if len(clusters) == 0 {
		return []HistoryRow{{LineID: line.LineID, Kind: line.Kind, Segment: line.Segment, SessionID: line.SessionID, FrameID: line.FrameID, FixedGrid: line.FixedGrid, ScreenCols: line.ScreenCols, ScreenRow: line.ScreenRow, ScreenRowSet: line.ScreenRowSet, LiveTail: line.LiveTail, UpdatedAt: line.UpdatedAt}}
	}
	rows := make([]HistoryRow, 0, 1)
	var builder strings.Builder
	width := 0
	flush := func() {
		row := HistoryRow{
			Text:         builder.String(),
			LineID:       line.LineID,
			RowInLine:    len(rows),
			Kind:         line.Kind,
			Segment:      line.Segment,
			SessionID:    line.SessionID,
			FrameID:      line.FrameID,
			FixedGrid:    line.FixedGrid,
			ScreenCols:   line.ScreenCols,
			ScreenRow:    line.ScreenRow,
			ScreenRowSet: line.ScreenRowSet,
			LiveTail:     line.LiveTail,
			UpdatedAt:    line.UpdatedAt,
		}
		rows = append(rows, row)
		builder.Reset()
		width = 0
	}
	for _, cluster := range clusters {
		clusterWidth := textDisplayWidth(cluster)
		if clusterWidth <= 0 {
			continue
		}
		if width > 0 && width+clusterWidth > cols {
			flush()
		}
		builder.WriteString(cluster)
		width += clusterWidth
		if width >= cols {
			flush()
		}
	}
	if builder.Len() > 0 || len(rows) == 0 {
		flush()
	}
	applyTailFillToLastHistoryRow(rows, line.TailFill)
	return rows
}

func reflowPlainASCIIHistoryLogicalLine(line HistoryLogicalLine, cols int) []HistoryRow {
	if line.Text == "" {
		return []HistoryRow{{LineID: line.LineID, Kind: line.Kind, Segment: line.Segment, SessionID: line.SessionID, FrameID: line.FrameID, FixedGrid: line.FixedGrid, ScreenCols: line.ScreenCols, ScreenRow: line.ScreenRow, ScreenRowSet: line.ScreenRowSet, LiveTail: line.LiveTail, UpdatedAt: line.UpdatedAt}}
	}
	if cols <= 0 {
		cols = 80
	}
	rows := make([]HistoryRow, 0, (len(line.Text)+cols-1)/cols)
	for start := 0; start < len(line.Text); start += cols {
		end := start + cols
		if end > len(line.Text) {
			end = len(line.Text)
		}
		rows = append(rows, HistoryRow{
			Text:         line.Text[start:end],
			LineID:       line.LineID,
			RowInLine:    len(rows),
			Kind:         line.Kind,
			Segment:      line.Segment,
			SessionID:    line.SessionID,
			FrameID:      line.FrameID,
			FixedGrid:    line.FixedGrid,
			ScreenCols:   line.ScreenCols,
			ScreenRow:    line.ScreenRow,
			ScreenRowSet: line.ScreenRowSet,
			LiveTail:     line.LiveTail,
			UpdatedAt:    line.UpdatedAt,
		})
	}
	applyTailFillToLastHistoryRow(rows, line.TailFill)
	return rows
}

func singleWidthASCIIText(text string) bool {
	if text == "" {
		return true
	}
	for i := 0; i < len(text); i++ {
		if text[i] < 0x20 || text[i] >= 0x7f {
			return false
		}
	}
	return true
}

func applyTailFillToLastHistoryRow(rows []HistoryRow, fill *HistoryCellStyle) {
	if len(rows) == 0 || fill == nil {
		return
	}
	tail := *fill
	rows[len(rows)-1].TailFill = &tail
}

func normalizeHistoryLogicalLineCells(cells []HistoryCell, cols int) []HistoryCell {
	if len(cells) == 0 {
		return nil
	}
	out := make([]HistoryCell, 0, len(cells))
	for _, cell := range cells {
		parts := splitHistoryCell(cell, cols)
		if len(parts) == 0 {
			continue
		}
		out = append(out, parts...)
	}
	return out
}

func splitHistoryCell(cell HistoryCell, cols int) []HistoryCell {
	width := HistoryCellDisplayWidth(cell)
	if width <= 0 {
		return nil
	}
	if cols <= 0 {
		cols = 80
	}
	if cell.Text == "" {
		return splitEmptyHistoryCellFootprint(cell, cols)
	}
	clusters := textGraphemeClusters(cell.Text)
	if len(clusters) == 0 {
		return nil
	}
	naturalWidth := textClusterDisplayColumns(clusters)[len(clusters)]
	if width <= cols && naturalWidth == width {
		return []HistoryCell{cell}
	}
	if width <= cols && naturalWidth != width {
		return []HistoryCell{cell}
	}
	if len(clusters) == 1 && naturalWidth >= width {
		next := cell
		next.Width = width
		return []HistoryCell{next}
	}
	return splitHistoryCellForWrap(cell)
}

func splitHistoryCellForWrap(cell HistoryCell) []HistoryCell {
	width := HistoryCellDisplayWidth(cell)
	if width <= 0 {
		return nil
	}
	if cell.Text == "" {
		return splitEmptyHistoryCellFootprint(cell, 1)
	}
	clusters := textGraphemeClusters(cell.Text)
	if len(clusters) == 0 {
		return nil
	}
	naturalWidth := textClusterDisplayColumns(clusters)[len(clusters)]
	if len(clusters) == 1 && naturalWidth == width {
		return []HistoryCell{cell}
	}
	out := make([]HistoryCell, 0, len(clusters)+1)
	for _, cluster := range clusters {
		next := cell
		next.Text = cluster
		next.Width = textDisplayWidth(cluster)
		if next.Width <= 0 {
			continue
		}
		out = append(out, next)
	}
	if pad := width - naturalWidth; pad > 0 {
		padding := cell
		padding.Text = strings.Repeat(" ", pad)
		padding.Width = pad
		padding.LinkURL = ""
		padding.LinkParams = ""
		out = append(out, padding)
	}
	return out
}

func splitEmptyHistoryCellFootprint(cell HistoryCell, cols int) []HistoryCell {
	width := HistoryCellDisplayWidth(cell)
	if width <= 0 {
		return nil
	}
	if cols <= 0 || cols > width {
		cols = width
	}
	out := make([]HistoryCell, 0, (width+cols-1)/cols)
	for remaining := width; remaining > 0; {
		partWidth := remaining
		if partWidth > cols {
			partWidth = cols
		}
		next := cell
		// 中文说明：空文本 terminal cell 仍有真实列宽，copy/history 需要把它投影成带样式空格。
		next.Text = strings.Repeat(" ", partWidth)
		next.Width = partWidth
		next.LinkURL = ""
		next.LinkParams = ""
		out = append(out, next)
		remaining -= partWidth
	}
	return out
}

func historyCellsPlainTextForState(cells []HistoryCell) string {
	var builder strings.Builder
	for _, cell := range cells {
		if cell.Text == "" {
			continue
		}
		builder.WriteString(cell.Text)
		if pad := HistoryCellDisplayWidth(cell) - textDisplayWidth(cell.Text); pad > 0 {
			builder.WriteString(strings.Repeat(" ", pad))
		}
	}
	return builder.String()
}

func cloneCopyMatches(matches []CopyMatch) []CopyMatch {
	if len(matches) == 0 {
		return nil
	}
	cloned := make([]CopyMatch, len(matches))
	copy(cloned, matches)
	return cloned
}

type historyLogicalOffset struct {
	lineID    uint64
	rowInLine int
	col       int
}

func reflowViewportTop(before HistoryStore, after HistoryStore, top int) int {
	if len(before.Rows) == 0 || len(after.Rows) == 0 {
		return 0
	}
	offset := historyLogicalOffsetForPosition(before, CopyPosition{Row: top, Col: 0})
	position := positionForHistoryLogicalOffset(after, offset)
	return clampCopyInt(position.Row, 0, maxCopyInt(0, len(after.Rows)-1))
}

func reflowCopyPosition(before HistoryStore, after HistoryStore, pos CopyPosition) CopyPosition {
	if len(after.Rows) == 0 {
		return CopyPosition{}
	}
	offset := historyLogicalOffsetForPosition(before, pos)
	offset.col += historyLogicalPrependedPrefixWidth(before, after, offset.lineID)
	return positionForHistoryLogicalOffset(after, offset)
}

func CopyLogicalPositionForPosition(history HistoryStore, pos CopyPosition) CopyLogicalPosition {
	offset := historyLogicalOffsetForPosition(history, pos)
	if offset.lineID == 0 {
		return CopyLogicalPosition{}
	}
	return CopyLogicalPosition{Valid: true, LineID: offset.lineID, Col: offset.col}
}

func CopyPositionForLogicalPosition(history HistoryStore, pos CopyLogicalPosition) CopyPosition {
	if !pos.Valid {
		return CopyPosition{}
	}
	return positionForHistoryLogicalOffset(history, historyLogicalOffset{lineID: pos.LineID, col: pos.Col})
}

func copyLogicalPositionSame(left CopyLogicalPosition, right CopyLogicalPosition) bool {
	return left.Valid == right.Valid && left.LineID == right.LineID && left.Col == right.Col
}

func historyLogicalOffsetForPosition(history HistoryStore, pos CopyPosition) historyLogicalOffset {
	if len(history.Rows) == 0 {
		return historyLogicalOffset{}
	}
	row := clampCopyInt(pos.Row, 0, len(history.Rows)-1)
	current := history.Rows[row]
	col := clampCopyInt(pos.Col, 0, HistoryRowDisplayWidth(current))
	offset := col
	for cursor := row - 1; cursor >= 0; cursor-- {
		previous := history.Rows[cursor]
		if previous.LineID != current.LineID {
			break
		}
		offset += HistoryRowDisplayWidth(previous)
	}
	return historyLogicalOffset{
		lineID:    current.LineID,
		rowInLine: current.RowInLine,
		col:       offset,
	}
}

func positionForHistoryLogicalOffset(history HistoryStore, offset historyLogicalOffset) CopyPosition {
	if len(history.Rows) == 0 {
		return CopyPosition{}
	}
	if offset.lineID == 0 {
		return CopyPosition{}
	}
	remaining := maxCopyInt(0, offset.col)
	fallback := -1
	for rowIndex, row := range history.Rows {
		if row.LineID != offset.lineID {
			continue
		}
		if fallback < 0 {
			fallback = rowIndex
		}
		width := HistoryRowDisplayWidth(row)
		if remaining <= width {
			return CopyPosition{Row: rowIndex, Col: remaining}
		}
		remaining -= width
	}
	if fallback >= 0 {
		return CopyPosition{Row: fallback, Col: clampCopyInt(offset.col, 0, HistoryRowDisplayWidth(history.Rows[fallback]))}
	}
	last := len(history.Rows) - 1
	return CopyPosition{Row: last, Col: HistoryRowDisplayWidth(history.Rows[last])}
}

func historyLogicalPrependedPrefixWidth(before HistoryStore, after HistoryStore, lineID uint64) int {
	if lineID == 0 {
		return 0
	}
	beforeLine, ok := historyLogicalLineByID(before, lineID)
	if !ok || !beforeLine.ClippedBefore {
		return 0
	}
	afterLine, ok := historyLogicalLineByID(after, lineID)
	if !ok {
		return 0
	}
	return historyLogicalLinePrependedPrefixWidth(beforeLine, afterLine)
}

func historyLogicalLineByID(history HistoryStore, lineID uint64) (HistoryLogicalLine, bool) {
	lines := history.SourceLines
	if len(lines) == 0 && len(history.Rows) > 0 {
		lines = historyRowsToLogicalLines(history.Rows, history.Lines)
	}
	for _, line := range lines {
		if line.LineID == lineID {
			return line, true
		}
	}
	return HistoryLogicalLine{}, false
}

func historyLogicalLinePrependedPrefixWidth(before HistoryLogicalLine, after HistoryLogicalLine) int {
	if before.Text != "" && after.Text != "" && strings.HasSuffix(after.Text, before.Text) {
		prefix := strings.TrimSuffix(after.Text, before.Text)
		return textDisplayWidth(prefix)
	}
	if len(before.Cells) == 0 || len(after.Cells) == 0 {
		return 0
	}
	beforeIndex := len(before.Cells) - 1
	afterIndex := len(after.Cells) - 1
	for beforeIndex >= 0 && afterIndex >= 0 && historyCellsEqual(before.Cells[beforeIndex], after.Cells[afterIndex]) {
		beforeIndex--
		afterIndex--
	}
	width := 0
	for index := 0; index <= afterIndex; index++ {
		width += HistoryCellDisplayWidth(after.Cells[index])
	}
	return width
}

func historyCellsEqual(left HistoryCell, right HistoryCell) bool {
	return left == right
}

func clampCopyInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func maxCopyInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func minCopyInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func copyVisibleRowsForStore(store CopyModeStore) int {
	if store.ViewRows > 0 {
		return maxCopyInt(1, store.ViewRows)
	}
	return 8
}

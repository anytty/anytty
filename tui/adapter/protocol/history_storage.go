package protocoladapter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/shared/perftrace"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
	xansi "github.com/charmbracelet/x/ansi"
)

// ProtocolPathServiceAdapter 把 endpoint path completion service 映射到 core-v2 protocol。
// adapter 不读取本地文件系统；Prefix 直接交给当前 protocol session 所属 daemon，
// 保证 SSH/hub endpoint 的目录候选来自远端机器。
type ProtocolPathServiceAdapter struct {
	Application *clientruntime.ApplicationSession
}

// NewProtocolPathServiceAdapter 复用 terminal application session 查询 owning endpoint 的路径和默认值。
// 该构造函数不接受裸 protocol client，防止 adapter 丢失 route/generation fence。
func NewProtocolPathServiceAdapter(application *clientruntime.ApplicationSession) (ProtocolPathServiceAdapter, error) {
	if application == nil {
		return ProtocolPathServiceAdapter{}, fmt.Errorf("missing path application session")
	}
	return ProtocolPathServiceAdapter{Application: application}, nil
}

// ListDirectories 返回当前 daemon 文件系统中的目录候选。
// EndpointID 已由 client runtime adapter 剥离；如果 Client 缺失，调用方会在 prompt 内展示失败。
func (adapter ProtocolPathServiceAdapter) ListDirectories(ctx context.Context, req port.PathListDirectoriesRequest) (port.PathListDirectoriesResult, error) {
	if adapter.Application == nil {
		return port.PathListDirectoriesResult{}, fmt.Errorf("missing path client")
	}
	result, err := adapter.Application.PathListDirectories(ctx, &apipb.PathListDirectoriesCommand{Prefix: req.Prefix, Limit: int32(req.Limit)})
	if err != nil {
		return port.PathListDirectoriesResult{}, err
	}
	out := port.PathListDirectoriesResult{
		EndpointID: req.EndpointID, BasePath: result.GetBasePath(), Missing: result.GetMissing(), Truncated: result.GetTruncated(),
		Entries: make([]port.PathDirectoryEntry, 0, len(result.GetEntries())),
	}
	for _, entry := range result.GetEntries() {
		out.Entries = append(out.Entries, port.PathDirectoryEntry{Name: entry.GetName(), Path: entry.GetPath()})
	}
	return out, nil
}

// Defaults 返回当前 daemon 进程所在机器的创建默认值。
// EndpointID 已由 client runtime adapter 剥离；adapter 只消费 protocol 投影，不读取 TUI 本地环境。
func (adapter ProtocolPathServiceAdapter) Defaults(ctx context.Context, req port.PathDefaultsRequest) (port.PathDefaultsResult, error) {
	if adapter.Application == nil {
		return port.PathDefaultsResult{}, fmt.Errorf("missing path client")
	}
	result, err := adapter.Application.TerminalDefaults(ctx, &apipb.TerminalDefaultsCommand{})
	if err != nil {
		return port.PathDefaultsResult{}, err
	}
	return port.PathDefaultsResult{
		EndpointID: req.EndpointID, DefaultCommand: append([]string(nil), result.GetDefaults().GetDefaultCommand()...), DefaultCWD: result.GetDefaults().GetDefaultCwd(),
	}, nil
}

// ProtocolCoreClientAdapter 是真实 protocol history.window 的 service adapter。
type ProtocolCoreClientAdapter struct {
	Application *clientruntime.ApplicationSession
}

func (adapter ProtocolCoreClientAdapter) HistoryLatest(ctx context.Context, req port.HistoryLatestRequest) (port.HistoryResult, error) {
	window, err := adapter.historyWindow(ctx, &apipb.HistoryWindowCommand{
		Terminal: &apipb.TerminalRef{EndpointId: string(req.EndpointID), TerminalId: req.TerminalID},
		Mode:     apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST, Limit: int32(req.Rows), Cols: int32(req.Cols), HistoryGeneration: req.GenerationBoundary,
	}, true)
	if err != nil {
		return port.HistoryResult{RequestID: req.RequestID}, err
	}
	return port.HistoryResult{RequestID: req.RequestID, Window: window}, nil
}

func (adapter ProtocolCoreClientAdapter) HistoryOlder(ctx context.Context, req port.HistoryOlderRequest) (port.HistoryResult, error) {
	window, err := adapter.historyWindow(ctx, &apipb.HistoryWindowCommand{
		Terminal: &apipb.TerminalRef{EndpointId: string(req.EndpointID), TerminalId: req.TerminalID}, Mode: apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDER,
		Limit: int32(req.Rows), Cols: int32(req.Cols), Token: req.Token, HistoryGeneration: req.Generation,
		BeforeCursor: historyCursorToProto(req.Cursor), BoundaryFirstLineId: req.Boundary.FirstLineID, BoundaryLastLineId: req.Boundary.LastLineID,
	}, false)
	if err != nil {
		return port.HistoryResult{RequestID: req.RequestID}, err
	}
	return port.HistoryResult{RequestID: req.RequestID, Window: window}, nil
}

func (adapter ProtocolCoreClientAdapter) HistoryNewer(ctx context.Context, req port.HistoryNewerRequest) (port.HistoryResult, error) {
	window, err := adapter.historyWindow(ctx, &apipb.HistoryWindowCommand{
		Terminal: &apipb.TerminalRef{EndpointId: string(req.EndpointID), TerminalId: req.TerminalID}, Mode: apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_NEWER,
		Limit: int32(req.Rows), Cols: int32(req.Cols), Token: req.Token, HistoryGeneration: req.Generation,
		AfterCursor: historyCursorToProto(req.Cursor), BoundaryFirstLineId: req.Boundary.FirstLineID, BoundaryLastLineId: req.Boundary.LastLineID,
	}, false)
	if err != nil {
		return port.HistoryResult{RequestID: req.RequestID}, err
	}
	return port.HistoryResult{RequestID: req.RequestID, Window: window}, nil
}

func (adapter ProtocolCoreClientAdapter) HistoryOldest(ctx context.Context, req port.HistoryOldestRequest) (port.HistoryResult, error) {
	window, err := adapter.historyWindow(ctx, &apipb.HistoryWindowCommand{
		Terminal: &apipb.TerminalRef{EndpointId: string(req.EndpointID), TerminalId: req.TerminalID}, Mode: apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDEST,
		Limit: int32(req.Rows), Cols: int32(req.Cols), Token: req.Token, HistoryGeneration: req.Generation,
		BoundaryFirstLineId: req.Boundary.FirstLineID, BoundaryLastLineId: req.Boundary.LastLineID,
	}, false)
	if err != nil {
		return port.HistoryResult{RequestID: req.RequestID}, err
	}
	return port.HistoryResult{RequestID: req.RequestID, Window: window}, nil
}

func (adapter ProtocolCoreClientAdapter) ReleaseHistory(ctx context.Context, req port.HistoryReleaseRequest) error {
	if adapter.Application == nil || req.Token == "" {
		return nil
	}
	return adapter.Application.HistoryRelease(ctx, &apipb.HistoryReleaseCommand{Terminal: &apipb.TerminalRef{EndpointId: string(req.EndpointID), TerminalId: req.TerminalID}, Token: req.Token})
}

func (adapter ProtocolCoreClientAdapter) HistoryCopyRange(ctx context.Context, req port.HistoryCopyRangeRequest) (port.HistoryCopyRangeResult, error) {
	if !req.Start.Valid || !req.End.Valid {
		return port.HistoryCopyRangeResult{}, port.ErrHistoryCopyTooLarge
	}
	const maxMaterializedCopyBytes = 64 << 20
	current := req.Start
	var text strings.Builder
	firstChunk := true
	for {
		window := &apipb.HistoryWindowCommand{
			Token: req.Token, Cols: int32(req.Cols), HistoryGeneration: req.Generation,
			BoundaryFirstLineId: req.Boundary.FirstLineID, BoundaryLastLineId: req.Boundary.LastLineID,
			Range: &apipb.HistoryRange{StartLineId: current.LineID, StartCol: int32(current.Col), EndLineId: req.End.LineID, EndCol: int32(req.End.Col)},
		}
		result, err := adapter.Application.HistoryCopy(ctx, &apipb.HistoryCopyCommand{
			Terminal: &apipb.TerminalRef{EndpointId: string(req.EndpointID), TerminalId: req.TerminalID},
			Window:   window, MaxLines: 8192, MaxBytes: 512 << 10,
		})
		if err != nil {
			return port.HistoryCopyRangeResult{}, normalizeProtocolHistoryCopyError(err)
		}
		separatorBytes := 0
		if !firstChunk {
			separatorBytes = 1
		}
		if text.Len()+separatorBytes+len(result.GetText()) > maxMaterializedCopyBytes {
			return port.HistoryCopyRangeResult{}, port.ErrHistoryCopyTooLarge
		}
		if separatorBytes != 0 {
			text.WriteByte('\n')
		}
		text.WriteString(result.GetText())
		if result.GetDone() {
			return port.HistoryCopyRangeResult{Text: text.String()}, nil
		}
		next := result.GetNext()
		if next.GetLineId() == 0 || next.GetLineId() <= current.LineID {
			return port.HistoryCopyRangeResult{}, port.ErrMissingHistoryResponse
		}
		current = state.CopyLogicalPosition{Valid: true, LineID: next.GetLineId(), Col: int(next.GetCol())}
		firstChunk = false
	}
}

func (adapter ProtocolCoreClientAdapter) HistorySearch(ctx context.Context, req port.HistorySearchRequest) (port.HistorySearchResult, error) {
	direction := apipb.HistorySearchDirection_HISTORY_SEARCH_DIRECTION_FORWARD
	if req.Direction == port.HistorySearchBackward {
		direction = apipb.HistorySearchDirection_HISTORY_SEARCH_DIRECTION_BACKWARD
	}
	command := &apipb.HistorySearchCommand{
		Terminal:          &apipb.TerminalRef{EndpointId: string(req.EndpointID), TerminalId: req.TerminalID},
		Token:             req.Token,
		HistoryGeneration: req.Generation,
		Query:             req.Query,
		Direction:         direction,
		Cols:              int32(req.Cols),
		Limit:             int32(req.Rows),
	}
	if req.Start.Valid {
		command.Start = &apipb.HistoryTextPosition{LineId: req.Start.LineID, Col: int32(req.Start.Col)}
	}
	result, err := adapter.Application.HistorySearch(ctx, command)
	if err != nil {
		return port.HistorySearchResult{RequestID: req.RequestID}, normalizeProtocolHistoryWindowError(err)
	}
	out := port.HistorySearchResult{RequestID: req.RequestID, Found: result.GetFound(), Wrapped: result.GetWrapped()}
	if !out.Found {
		return out, nil
	}
	out.Start = state.CopyLogicalPosition{Valid: true, LineID: result.GetMatch().GetStartLineId(), Col: int(result.GetMatch().GetStartCol())}
	out.End = state.CopyLogicalPosition{Valid: true, LineID: result.GetMatch().GetEndLineId(), Col: int(result.GetMatch().GetEndCol())}
	out.Window = historyWindowFromProto(result.GetWindow(), req.Cols)
	return out, nil
}

func (adapter ProtocolCoreClientAdapter) historyWindow(ctx context.Context, command *apipb.HistoryWindowCommand, terminalResponse bool) (state.HistoryWindow, error) {
	if adapter.Application == nil {
		return state.HistoryWindow{}, fmt.Errorf("missing history application session")
	}
	mode := strings.ToLower(strings.TrimPrefix(command.GetMode().String(), "HISTORY_WINDOW_MODE_"))
	finishRPC := perftrace.Measure("tui.protocol.history_window." + mode + ".rpc")
	var window *apipb.HistoryWindowResult
	var err error
	if terminalResponse {
		var result *apipb.ResultEnvelope
		result, err = adapter.Application.ExecuteTerminal(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_HistoryWindow{HistoryWindow: command}})
		if result != nil {
			window = result.GetHistoryWindow()
		}
		if err == nil && window == nil {
			err = fmt.Errorf("history_window returned no result")
		}
	} else {
		window, err = adapter.Application.HistoryWindow(ctx, command)
	}
	if window != nil {
		finishRPC(len(window.GetRows()))
		perftrace.Count("tui.protocol.history_window."+mode+".rows", len(window.GetRows()))
	} else {
		finishRPC(0)
	}
	if err != nil {
		return state.HistoryWindow{}, normalizeProtocolHistoryWindowError(err)
	}
	if terminalResponse && ctx.Err() != nil {
		if window.GetToken() != "" {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			_ = adapter.Application.HistoryRelease(cleanupCtx, &apipb.HistoryReleaseCommand{Terminal: command.GetTerminal(), Token: window.GetToken(), HistoryGeneration: window.GetHistoryGeneration()})
			cancel()
		}
		return state.HistoryWindow{}, ctx.Err()
	}
	finishConvert := perftrace.Measure("tui.protocol.history_window." + mode + ".convert")
	converted := historyWindowFromProto(window, int(command.GetCols()))
	finishConvert(len(converted.Rows))
	return converted, nil
}

func historyCursorToProto(cursor state.HistoryCursor) *apipb.HistoryCursor {
	if !cursor.Valid {
		return nil
	}
	return &apipb.HistoryCursor{LineId: cursor.BeforeLineID, RowInLine: int32(cursor.BeforeRowInLine), Segment: historySegmentToProto(cursor.Segment)}
}

func historySegmentToProto(segment string) apipb.HistoryCursorSegment {
	switch segment {
	case "committed":
		return apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_COMMITTED
	case "current-primary-frame":
		return apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_CURRENT_PRIMARY_FRAME
	case "archived-primary-frame":
		return apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_ARCHIVED_PRIMARY_FRAME
	case "current-alt-frame":
		return apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_CURRENT_ALT_FRAME
	default:
		return apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_UNSPECIFIED
	}
}

func normalizeProtocolHistoryWindowError(err error) error {
	return normalizeProtocolHistoryError(err, port.ErrHistoryWindowTooLarge)
}

func normalizeProtocolHistoryCopyError(err error) error {
	return normalizeProtocolHistoryError(err, port.ErrHistoryCopyTooLarge)
}

func normalizeProtocolHistoryError(err error, nonRetryableLimitError error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, port.ErrStaleHistoryWindow) {
		return err
	}
	if clientruntime.CodeOf(err) == clientruntime.ErrorStaleResource {
		return fmt.Errorf("%w: %v", port.ErrStaleHistoryWindow, err)
	}
	if clientruntime.CodeOf(err) == clientruntime.ErrorResourceExhausted {
		if clientruntime.IsRetryable(err) {
			return fmt.Errorf("%w: %v", port.ErrHistoryResourceExhausted, err)
		}
		return fmt.Errorf("%w: %v", nonRetryableLimitError, err)
	}
	return err
}

func historyWindowFromProto(window *apipb.HistoryWindowResult, requestedCols int) state.HistoryWindow {
	if window == nil {
		return state.HistoryWindow{}
	}
	cols := requestedCols
	if cols <= 0 {
		cols = int(window.GetSize().GetCols())
	}
	sourceLines := historySourceLinesFromProto(window)
	rows, lines := historyRowsFromProto(window, sourceLines, cols)
	cursor := window.GetCursor()
	anchor := window.GetViewportAnchor()
	return state.HistoryWindow{
		TerminalID: window.GetTerminal().GetTerminalId(), Token: window.GetToken(), Op: historyOperationFromProto(window.GetOperation()), Cols: cols,
		SourceLines: sourceLines, Rows: rows, Lines: lines,
		Cursor:  state.HistoryCursor{Valid: cursor != nil, BeforeLineID: cursor.GetLineId(), BeforeRowInLine: int(cursor.GetRowInLine()), Segment: historySegmentFromProto(cursor.GetSegment())},
		HasMore: window.GetHasMore(), Generation: window.GetHistoryGeneration(),
		Boundary: state.HistoryBoundary{FirstLineID: window.GetFirstLineId(), LastLineID: window.GetLastLineId()},
		ViewportAnchor: state.HistoryViewportAnchor{
			TopLineID: anchor.GetTopLineId(), TopCellOffset: int(anchor.GetTopCellOffset()), AtEnd: anchor.GetAtEnd(),
			ScreenCols: int(anchor.GetScreenCols()), ScreenRows: int(anchor.GetScreenRows()), Valid: anchor != nil,
		},
		LoadedLines: int(window.GetLoadedLines()), TotalLines: int(window.GetLogicalTotal()),
	}
}

func historySourceLinesFromProto(window *apipb.HistoryWindowResult) []state.HistoryLogicalLine {
	lines := make([]state.HistoryLogicalLine, 0, len(window.GetRows()))
	for _, row := range window.GetRows() {
		text, cells := historyTextAndCellsFromProto(row.GetRow())
		next := state.HistoryLogicalLine{Text: text, Cells: cells, LineID: row.GetLogicalLineId(), Kind: row.GetRowKind(), Segment: historySegmentFromProto(row.GetSegment()), SessionID: row.GetSessionId(), FrameID: row.GetFrameId(), FixedGrid: row.GetFixedGrid(), ScreenCols: int(row.GetScreenCols()), ScreenRow: int(row.GetScreenRows()), ScreenRowSet: row.GetScreenRowSet(), TailFill: historyStyleFromProto(row.GetRow().GetTailFill()), LiveTail: row.GetOwnership() == apipb.RowOwnership_ROW_OWNERSHIP_LIVE_TAIL_LIVE, UpdatedAt: historyTimeFromUnixNano(row.GetTimestampUnixNano())}
		if len(lines) > 0 && sameProtoHistorySource(lines[len(lines)-1], next) {
			appendProtoHistorySegment(&lines[len(lines)-1], next)
			continue
		}
		lines = append(lines, next)
	}
	for _, span := range window.GetLines() {
		for index := int(span.GetStartRow()); index <= int(span.GetEndRow()) && index < len(lines); index++ {
			if index >= 0 {
				lines[index].ClippedBefore = span.GetClippedBefore()
				lines[index].ClippedAfter = span.GetClippedAfter()
			}
		}
	}
	return lines
}

func historyRowsFromProto(window *apipb.HistoryWindowResult, sourceLines []state.HistoryLogicalLine, cols int) ([]state.HistoryRow, []state.HistoryLineSpan) {
	if cols != int(window.GetSize().GetCols()) {
		return state.ReflowHistoryLogicalLines(sourceLines, cols)
	}
	rows := make([]state.HistoryRow, 0, len(window.GetRows()))
	for _, row := range window.GetRows() {
		text, cells := historyTextAndCellsFromProto(row.GetRow())
		rows = append(rows, state.HistoryRow{Text: text, Cells: cells, TailFill: historyStyleFromProto(row.GetRow().GetTailFill()), LineID: row.GetLogicalLineId(), RowInLine: int(row.GetRowInLine()), Kind: row.GetRowKind(), Segment: historySegmentFromProto(row.GetSegment()), SessionID: row.GetSessionId(), FrameID: row.GetFrameId(), FixedGrid: row.GetFixedGrid(), ScreenCols: int(row.GetScreenCols()), ScreenRow: int(row.GetScreenRows()), ScreenRowSet: row.GetScreenRowSet(), LiveTail: row.GetOwnership() == apipb.RowOwnership_ROW_OWNERSHIP_LIVE_TAIL_LIVE, UpdatedAt: historyTimeFromUnixNano(row.GetTimestampUnixNano())})
	}
	lines := make([]state.HistoryLineSpan, 0, len(window.GetLines()))
	for _, span := range window.GetLines() {
		segment := ""
		screenRow := 0
		screenRowSet := false
		if index := int(span.GetStartRow()); index >= 0 && index < len(rows) {
			segment, screenRow, screenRowSet = rows[index].Segment, rows[index].ScreenRow, rows[index].ScreenRowSet
		}
		lines = append(lines, state.HistoryLineSpan{LineID: span.GetLogicalLineId(), StartRow: int(span.GetStartRow()), EndRow: int(span.GetEndRow()), Kind: span.GetRowKind(), Segment: segment, SessionID: span.GetSessionId(), FrameID: span.GetFrameId(), FixedGrid: span.GetFixedGrid(), ScreenCols: int(span.GetScreenCols()), ScreenRow: screenRow, ScreenRowSet: screenRowSet, ClippedBefore: span.GetClippedBefore(), ClippedAfter: span.GetClippedAfter(), UpdatedAt: historyTimeFromUnixNano(span.GetTimestampEndUnixNano())})
	}
	if len(lines) == 0 {
		lines = protoLineSpansFromRows(rows, sourceLines)
	}
	return rows, lines
}

func historyTextAndCellsFromProto(row *apipb.ScreenRow) (string, []state.HistoryCell) {
	if row == nil || len(row.GetCells()) == 0 {
		return "", nil
	}
	cells := make([]state.HistoryCell, 0, len(row.GetCells()))
	for _, cell := range row.GetCells() {
		cells = append(cells, state.HistoryCell{Text: cell.GetContent(), Width: int(cell.GetWidth()), Style: historyCellStyleFromProto(cell.GetStyle()), LinkURL: cell.GetLinkUrl(), LinkParams: cell.GetLinkParams()})
	}
	return protoHistoryCellsPlainText(cells), cells
}

func historyCellStyleFromProto(style *apipb.CellStyle) state.HistoryCellStyle {
	if style == nil {
		return state.HistoryCellStyle{}
	}
	return state.HistoryCellStyle{FG: style.GetForeground(), BG: style.GetBackground(), Bold: style.GetBold(), Italic: style.GetItalic(), Underline: style.GetUnderline(), Blink: style.GetBlink(), Reverse: style.GetReverse(), Strikethrough: style.GetStrikethrough()}
}

func historyStyleFromProto(style *apipb.CellStyle) *state.HistoryCellStyle {
	if style == nil {
		return nil
	}
	value := historyCellStyleFromProto(style)
	if value == (state.HistoryCellStyle{}) {
		return nil
	}
	return &value
}

func historyOperationFromProto(operation apipb.HistoryWindowOperation) state.HistoryWindowOp {
	switch operation {
	case apipb.HistoryWindowOperation_HISTORY_WINDOW_OPERATION_PREPEND:
		return state.HistoryWindowPrepend
	case apipb.HistoryWindowOperation_HISTORY_WINDOW_OPERATION_APPEND:
		return state.HistoryWindowAppend
	default:
		return state.HistoryWindowReplace
	}
}

func historySegmentFromProto(segment apipb.HistoryCursorSegment) string {
	switch segment {
	case apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_COMMITTED:
		return "committed"
	case apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_CURRENT_PRIMARY_FRAME:
		return "current-primary-frame"
	case apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_ARCHIVED_PRIMARY_FRAME:
		return "archived-primary-frame"
	case apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_CURRENT_ALT_FRAME:
		return "current-alt-frame"
	default:
		return ""
	}
}

func sameProtoHistorySource(left, right state.HistoryLogicalLine) bool {
	return left.LineID != 0 && left.LineID == right.LineID && left.Kind == right.Kind && left.Segment == right.Segment && left.SessionID == right.SessionID && left.FrameID == right.FrameID && left.FixedGrid == right.FixedGrid && (!left.FixedGrid || left.ScreenCols == right.ScreenCols)
}

func appendProtoHistorySegment(line *state.HistoryLogicalLine, next state.HistoryLogicalLine) {
	line.Text += next.Text
	line.Cells = append(line.Cells, next.Cells...)
	if next.UpdatedAt.After(line.UpdatedAt) {
		line.UpdatedAt = next.UpdatedAt
	}
}

func protoLineSpansFromRows(rows []state.HistoryRow, sourceLines []state.HistoryLogicalLine) []state.HistoryLineSpan {
	if len(rows) == 0 {
		return nil
	}
	spans := make([]state.HistoryLineSpan, 0, len(rows))
	start := 0
	for index := 1; index <= len(rows); index++ {
		if index < len(rows) && rows[index].LineID == rows[start].LineID && rows[index].Kind == rows[start].Kind && rows[index].Segment == rows[start].Segment && rows[index].SessionID == rows[start].SessionID && rows[index].FrameID == rows[start].FrameID && rows[index].FixedGrid == rows[start].FixedGrid {
			continue
		}
		row := rows[start]
		spans = append(spans, state.HistoryLineSpan{LineID: row.LineID, StartRow: start, EndRow: index - 1, Kind: row.Kind, Segment: row.Segment, SessionID: row.SessionID, FrameID: row.FrameID, FixedGrid: row.FixedGrid, ScreenCols: row.ScreenCols, ScreenRow: row.ScreenRow, ScreenRowSet: row.ScreenRowSet, UpdatedAt: row.UpdatedAt})
		start = index
	}
	return spans
}

func historyTimeFromUnixNano(value int64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.Unix(0, value).UTC()
}

func protoHistoryCellsPlainText(cells []state.HistoryCell) string {
	var builder strings.Builder
	for _, cell := range cells {
		builder.WriteString(cell.Text)
		if cell.Text != " " {
			if padding := state.HistoryCellDisplayWidth(cell) - xansi.StringWidth(strings.ReplaceAll(cell.Text, "\n", " ")); padding > 0 {
				builder.WriteString(strings.Repeat(" ", padding))
			}
		}
	}
	return builder.String()
}

type ProtocolWorkbenchStorageAdapter struct {
	Application *clientruntime.ApplicationSession
}

func (adapter ProtocolWorkbenchStorageAdapter) LoadWorkbench(ctx context.Context, ref state.WorkbenchStorageRef) (port.WorkbenchStorageLoadResult, error) {
	result, err := adapter.Application.StorageGet(ctx, &apipb.StorageGetCommand{Key: storageKeyToProto(ref.AppID, ref.Scope, ref.OwnerID, ref.Key)})
	if err != nil {
		if isStorageNotFound(err) {
			return port.WorkbenchStorageLoadResult{Found: false}, nil
		}
		return port.WorkbenchStorageLoadResult{}, err
	}
	entry := result.GetEntry()
	if entry == nil || len(entry.GetValue()) == 0 {
		return port.WorkbenchStorageLoadResult{Found: false}, nil
	}
	snapshot, err := state.DecodeWorkbenchStorageSnapshot(entry.GetValue())
	if err != nil {
		return port.WorkbenchStorageLoadResult{}, err
	}
	return port.WorkbenchStorageLoadResult{
		Snapshot: snapshot,
		Version:  entry.GetVersion(),
		Found:    true,
	}, nil
}

type ProtocolClipboardStorageAdapter struct {
	Application *clientruntime.ApplicationSession
}

func (adapter ProtocolClipboardStorageAdapter) LoadClipboard(ctx context.Context, ref state.ClipboardStorageRef) (port.ClipboardStorageLoadResult, error) {
	result, err := adapter.Application.StorageGet(ctx, &apipb.StorageGetCommand{Key: storageKeyToProto(ref.AppID, ref.Scope, ref.OwnerID, ref.Key)})
	if err != nil {
		if isStorageNotFound(err) {
			return port.ClipboardStorageLoadResult{Found: false}, nil
		}
		return port.ClipboardStorageLoadResult{}, err
	}
	entry := result.GetEntry()
	if entry == nil || len(entry.GetValue()) == 0 {
		return port.ClipboardStorageLoadResult{Found: false}, nil
	}
	snapshot, err := state.DecodeClipboardStorageSnapshot(entry.GetValue())
	if err != nil {
		return port.ClipboardStorageLoadResult{}, err
	}
	return port.ClipboardStorageLoadResult{
		Snapshot: snapshot,
		Version:  entry.GetVersion(),
		Found:    true,
	}, nil
}

func (adapter ProtocolWorkbenchStorageAdapter) SaveWorkbench(ctx context.Context, req port.WorkbenchStorageSaveRequest) (port.WorkbenchStorageSaveResult, error) {
	value, err := state.EncodeWorkbenchStorageSnapshotValue(req.Snapshot)
	if err != nil {
		return port.WorkbenchStorageSaveResult{}, err
	}
	result, err := adapter.Application.StoragePut(ctx, &apipb.StoragePutCommand{Key: storageKeyToProto(req.Ref.AppID, req.Ref.Scope, req.Ref.OwnerID, req.Ref.Key), Value: value, Version: &apipb.StorageVersionFence{CheckVersion: req.CheckVersion, ExpectedVersion: req.ExpectedVersion}})
	if err != nil {
		if isStorageVersionConflict(err) {
			return port.WorkbenchStorageSaveResult{}, fmt.Errorf("%w: %v", port.ErrWorkbenchStorageConflict, err)
		}
		return port.WorkbenchStorageSaveResult{}, err
	}
	entry := result.GetEntry()
	return port.WorkbenchStorageSaveResult{
		Ref:     req.Ref.WithVersion(entry.GetVersion()),
		Version: entry.GetVersion(),
	}, nil
}

func (adapter ProtocolClipboardStorageAdapter) SaveClipboard(ctx context.Context, req port.ClipboardStorageSaveRequest) (port.ClipboardStorageSaveResult, error) {
	value, err := state.EncodeClipboardStorageSnapshotValue(req.Snapshot)
	if err != nil {
		return port.ClipboardStorageSaveResult{}, err
	}
	result, err := adapter.Application.StoragePut(ctx, &apipb.StoragePutCommand{Key: storageKeyToProto(req.Ref.AppID, req.Ref.Scope, req.Ref.OwnerID, req.Ref.Key), Value: value, Version: &apipb.StorageVersionFence{CheckVersion: req.CheckVersion, ExpectedVersion: req.ExpectedVersion}})
	if err != nil {
		if isStorageVersionConflict(err) {
			return port.ClipboardStorageSaveResult{}, fmt.Errorf("%w: %v", port.ErrClipboardStorageConflict, err)
		}
		return port.ClipboardStorageSaveResult{}, err
	}
	entry := result.GetEntry()
	return port.ClipboardStorageSaveResult{
		Ref:     req.Ref.WithVersion(entry.GetVersion()),
		Version: entry.GetVersion(),
	}, nil
}

func isStorageVersionConflict(err error) bool {
	return err != nil && strings.Contains(err.Error(), "storage version conflict")
}

func isStorageNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "storage entry not found")
}

func (adapter ProtocolWorkbenchStorageAdapter) WatchWorkbench(ctx context.Context, ref state.WorkbenchStorageRef) (<-chan port.WorkbenchStorageEvent, error) {
	_, events, err := adapter.Application.EventSubscribe(ctx, &apipb.EventSubscribeCommand{Types: []apipb.ApplicationEventType{apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_STORAGE_CHANGED}, StorageAppId: ref.AppID, StorageScope: storageScopeToProto(ref.Scope), StorageOwnerId: ref.OwnerID, StorageKeyPrefix: ref.KeyPrefix()})
	if err != nil {
		return nil, err
	}
	out := make(chan port.WorkbenchStorageEvent, 16)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				storage := event.GetStorageChanged()
				if storage == nil {
					continue
				}
				changed := port.WorkbenchStorageEvent{
					Ref: state.WorkbenchStorageRef{
						AppID: storage.GetKey().GetAppId(), Scope: storageScopeFromProto(storage.GetKey().GetScope()), OwnerID: storage.GetKey().GetOwnerId(), Key: storage.GetKey().GetKey(), Version: storage.GetVersion(),
					},
					Version: storage.GetVersion(), Op: storage.GetOperation(),
				}
				select {
				case out <- changed:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

func (adapter ProtocolClipboardStorageAdapter) WatchClipboard(ctx context.Context, ref state.ClipboardStorageRef) (<-chan port.ClipboardStorageEvent, error) {
	_, events, err := adapter.Application.EventSubscribe(ctx, &apipb.EventSubscribeCommand{Types: []apipb.ApplicationEventType{apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_STORAGE_CHANGED}, StorageAppId: ref.AppID, StorageScope: storageScopeToProto(ref.Scope), StorageOwnerId: ref.OwnerID, StorageKeyPrefix: ref.KeyPrefix()})
	if err != nil {
		return nil, err
	}
	out := make(chan port.ClipboardStorageEvent, 16)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				storage := event.GetStorageChanged()
				if storage == nil {
					continue
				}
				changed := port.ClipboardStorageEvent{
					Ref: state.ClipboardStorageRef{
						AppID: storage.GetKey().GetAppId(), Scope: storageScopeFromProto(storage.GetKey().GetScope()), OwnerID: storage.GetKey().GetOwnerId(), Key: storage.GetKey().GetKey(), Version: storage.GetVersion(),
					},
					Version: storage.GetVersion(), Op: storage.GetOperation(),
				}
				select {
				case out <- changed:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

func storageKeyToProto(appID, scope, ownerID, key string) *apipb.StorageKey {
	return &apipb.StorageKey{AppId: appID, Scope: storageScopeToProto(scope), OwnerId: ownerID, Key: key}
}

func storageScopeToProto(scope string) apipb.StorageScope {
	if strings.EqualFold(strings.TrimSpace(scope), "private") {
		return apipb.StorageScope_STORAGE_SCOPE_PRIVATE
	}
	return apipb.StorageScope_STORAGE_SCOPE_PUBLIC
}

func storageScopeFromProto(scope apipb.StorageScope) string {
	if scope == apipb.StorageScope_STORAGE_SCOPE_PRIVATE {
		return "private"
	}
	return "public"
}

package apimapping

import (
	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/proto/apipb"
	vterm "github.com/anytty/anytty/vterm/vterm"
)

// ValidateHistoryLiveCommand 校验 history/live command 的 terminal identity、token 和窗口边界。
func ValidateHistoryLiveCommand(command *apipb.CommandEnvelope) error {
	requestContext := RequestContextForCommand(command)
	if err := ValidateRequestContext(requestContext); err != nil {
		return err
	}
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_HistoryWindow:
		if err := validateTerminalRefForContext(value.HistoryWindow.GetTerminal(), requestContext); err != nil {
			return err
		}
		if value.HistoryWindow.GetLimit() < 1 || value.HistoryWindow.GetLimit() > history.MaxHistoryWindowLines {
			return validation("history_window.limit", "must be between 1 and 512")
		}
		if value.HistoryWindow.GetCols() < 0 {
			return validation("history_window.cols", "must not be negative")
		}
		switch value.HistoryWindow.GetMode() {
		case apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_UNSPECIFIED,
			apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST:
		case apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDER,
			apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_NEWER,
			apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDEST:
			if value.HistoryWindow.GetToken() == "" {
				return validation("history_window.token", "frozen history token is required for pagination")
			}
		default:
			return validation("history_window.mode", "unsupported history window mode")
		}
	case *apipb.CommandEnvelope_HistoryCopy:
		if err := validateTerminalRefForContext(value.HistoryCopy.GetTerminal(), requestContext); err != nil {
			return err
		}
		if value.HistoryCopy.GetWindow() == nil || value.HistoryCopy.GetWindow().GetToken() == "" {
			return validation("history_copy.window.token", "frozen history token is required")
		}
		if windowTerminal := value.HistoryCopy.GetWindow().GetTerminal(); windowTerminal != nil && (windowTerminal.GetEndpointId() != value.HistoryCopy.GetTerminal().GetEndpointId() || windowTerminal.GetTerminalId() != value.HistoryCopy.GetTerminal().GetTerminalId()) {
			return validation("history_copy.window.terminal", "must match history_copy.terminal")
		}
		if selection := value.HistoryCopy.GetWindow().GetRange(); selection != nil {
			if selection.GetStartLineId() == 0 || selection.GetEndLineId() == 0 {
				return validation("history_copy.window.range", "line ids are required")
			}
			if selection.GetStartCol() < 0 || selection.GetEndCol() < 0 {
				return validation("history_copy.window.range", "columns must not be negative")
			}
			if selection.GetStartLineId() > selection.GetEndLineId() || (selection.GetStartLineId() == selection.GetEndLineId() && selection.GetStartCol() > selection.GetEndCol()) {
				return validation("history_copy.window.range", "start must not follow end")
			}
		}
		if value.HistoryCopy.GetMaxLines() != 0 || value.HistoryCopy.GetMaxBytes() != 0 {
			if value.HistoryCopy.GetMaxLines() < 1 || value.HistoryCopy.GetMaxLines() > history.MaxHistoryCopyChunkLines {
				return validation("history_copy.max_lines", "must be between 1 and 8192")
			}
			if value.HistoryCopy.GetMaxBytes() < 1 || value.HistoryCopy.GetMaxBytes() > history.MaxHistoryCopyChunkBytes {
				return validation("history_copy.max_bytes", "must be between 1 and 524288")
			}
		}
	case *apipb.CommandEnvelope_HistorySearch:
		if err := validateTerminalRefForContext(value.HistorySearch.GetTerminal(), requestContext); err != nil {
			return err
		}
		if value.HistorySearch.GetToken() == "" {
			return validation("history_search.token", "frozen history token is required")
		}
		if value.HistorySearch.GetQuery() == "" {
			return validation("history_search.query", "must not be empty")
		}
		if value.HistorySearch.GetCols() < 0 {
			return validation("history_search.cols", "must not be negative")
		}
		if value.HistorySearch.GetLimit() < 1 || value.HistorySearch.GetLimit() > history.MaxHistoryWindowLines {
			return validation("history_search.limit", "must be between 1 and 512")
		}
		if value.HistorySearch.GetStart().GetCol() < 0 {
			return validation("history_search.start.col", "must not be negative")
		}
		if value.HistorySearch.GetContextBefore() < 0 || value.HistorySearch.GetContextBefore() >= value.HistorySearch.GetLimit() {
			return validation("history_search.context_before", "must be non-negative and smaller than limit")
		}
		switch value.HistorySearch.GetMode() {
		case apipb.HistorySearchMode_HISTORY_SEARCH_MODE_UNSPECIFIED,
			apipb.HistorySearchMode_HISTORY_SEARCH_MODE_TEXT,
			apipb.HistorySearchMode_HISTORY_SEARCH_MODE_GLOB,
			apipb.HistorySearchMode_HISTORY_SEARCH_MODE_REGEX:
		default:
			return validation("history_search.mode", "must be text, glob, or regex")
		}
		switch value.HistorySearch.GetDirection() {
		case apipb.HistorySearchDirection_HISTORY_SEARCH_DIRECTION_FORWARD,
			apipb.HistorySearchDirection_HISTORY_SEARCH_DIRECTION_BACKWARD:
		default:
			return validation("history_search.direction", "must be forward or backward")
		}
	case *apipb.CommandEnvelope_HistoryRelease:
		if err := validateTerminalRefForContext(value.HistoryRelease.GetTerminal(), requestContext); err != nil {
			return err
		}
		if value.HistoryRelease.GetToken() == "" {
			return validation("history_release.token", "frozen history token is required")
		}
	case *apipb.CommandEnvelope_HistoryBacklogStatus:
		return validateTerminalRefForContext(value.HistoryBacklogStatus.GetTerminal(), requestContext)
	case *apipb.CommandEnvelope_LiveScreenNext:
		return validateTerminalRefForContext(value.LiveScreenNext.GetTerminal(), requestContext)
	default:
		return validation("command", "history or live command is required")
	}
	return nil
}

// HistoryWindowRequestFromProto 把 generated Proto window 转为 core authoritative history 查询。
func HistoryWindowRequestFromProto(command *apipb.HistoryWindowCommand) history.HistoryWindowRequest {
	if command == nil {
		return history.HistoryWindowRequest{}
	}
	request := history.HistoryWindowRequest{
		TerminalID: command.GetTerminal().GetTerminalId(),
		Mode:       historyWindowModeFromProto(command.GetMode()),
		Cols:       int(command.GetCols()),
		Limit:      int(command.GetLimit()),
		Token:      history.HistoryToken(command.GetToken()),
		Cursor:     historyCursorFromProto(command.GetBeforeCursor(), command.GetHistoryGeneration(), command.GetToken()),
		Boundary: history.HistoryBoundary{
			FirstLineID: history.LogicalLineID(command.GetBoundaryFirstLineId()),
			LastLineID:  history.LogicalLineID(command.GetBoundaryLastLineId()),
		},
	}
	if command.GetAfterCursor() != nil {
		request.Cursor = historyCursorFromProto(command.GetAfterCursor(), command.GetHistoryGeneration(), command.GetToken())
	}
	request.Boundary.Cursor = request.Cursor
	if request.Mode == "" {
		request.Mode = history.HistoryWindowModeLatest
	}
	if request.Cols <= 0 {
		request.Cols = 80
	}
	return request
}

// HistoryCopyRequestFromProto maps the explicit start-inclusive/end-exclusive
// display-cell range without reusing pagination cursor coordinates.
func HistoryCopyRequestFromProto(command *apipb.HistoryCopyCommand) history.HistoryCopyRequest {
	window := command.GetWindow()
	request := history.HistoryCopyRequest{
		TerminalID: command.GetTerminal().GetTerminalId(),
		Token:      history.HistoryToken(window.GetToken()),
		Cols:       int(window.GetCols()),
	}
	if value := window.GetRange(); value != nil {
		request.Range = &history.HistoryCopyRange{
			Start: history.HistoryCopyPosition{LineID: history.LogicalLineID(value.GetStartLineId()), Col: int(value.GetStartCol())},
			End:   history.HistoryCopyPosition{LineID: history.LogicalLineID(value.GetEndLineId()), Col: int(value.GetEndCol())},
		}
	}
	return request
}

func HistoryCopyChunkRequestFromProto(command *apipb.HistoryCopyCommand) history.HistoryCopyChunkRequest {
	return history.HistoryCopyChunkRequest{
		HistoryCopyRequest: HistoryCopyRequestFromProto(command),
		MaxLines:           int(command.GetMaxLines()),
		MaxBytes:           int(command.GetMaxBytes()),
	}
}

func HistorySearchRequestFromProto(command *apipb.HistorySearchCommand) history.HistorySearchRequest {
	if command == nil {
		return history.HistorySearchRequest{}
	}
	direction := history.HistorySearchForward
	if command.GetDirection() == apipb.HistorySearchDirection_HISTORY_SEARCH_DIRECTION_BACKWARD {
		direction = history.HistorySearchBackward
	}
	return history.HistorySearchRequest{
		TerminalID:    command.GetTerminal().GetTerminalId(),
		Token:         history.HistoryToken(command.GetToken()),
		Cols:          int(command.GetCols()),
		Limit:         int(command.GetLimit()),
		Query:         command.GetQuery(),
		Mode:          historySearchModeFromProto(command.GetMode()),
		Direction:     direction,
		ContextBefore: int(command.GetContextBefore()),
		Start: history.HistoryCopyPosition{
			LineID: history.LogicalLineID(command.GetStart().GetLineId()),
			Col:    int(command.GetStart().GetCol()),
		},
	}
}

func historySearchModeFromProto(mode apipb.HistorySearchMode) history.HistorySearchMode {
	switch mode {
	case apipb.HistorySearchMode_HISTORY_SEARCH_MODE_GLOB:
		return history.HistorySearchModeGlob
	case apipb.HistorySearchMode_HISTORY_SEARCH_MODE_REGEX:
		return history.HistorySearchModeRegex
	default:
		return history.HistorySearchModeText
	}
}

// HistoryWindowToProto 把 core authoritative window 投影为 generated Proto。
func HistoryWindowToProto(endpointID string, window history.HistoryWindow) *apipb.HistoryWindowResult {
	result := &apipb.HistoryWindowResult{
		Terminal:          &apipb.TerminalRef{EndpointId: endpointID, TerminalId: window.TerminalID},
		Token:             string(window.Token),
		Operation:         historyWindowOperationToProto(window.Op),
		Size:              &apipb.TerminalSize{Cols: uint32(window.Cols)},
		LoadedRows:        int32(len(window.Rows)),
		TotalRows:         int32(window.LogicalTotal),
		LoadedLines:       int32(len(window.Lines)),
		LogicalTotal:      int32(window.LogicalTotal),
		HasMore:           window.HasMore,
		HistoryGeneration: uint64(window.Generation),
		FirstLineId:       uint64(window.Boundary.FirstLineID),
		LastLineId:        uint64(window.Boundary.LastLineID),
		Cursor:            historyCursorToProto(window.Boundary.Cursor),
		TimestampUnixNano: window.Timestamp.UnixNano(),
	}
	if window.ViewportAnchor.Valid {
		result.ViewportAnchor = &apipb.HistoryViewportAnchor{
			TopLineId: uint64(window.ViewportAnchor.TopLineID), TopCellOffset: int32(window.ViewportAnchor.TopCellOffset),
			AtEnd: window.ViewportAnchor.AtEnd, ScreenCols: uint32(window.ViewportAnchor.ScreenCols), ScreenRows: uint32(window.ViewportAnchor.ScreenRows),
		}
	}
	for _, row := range window.Rows {
		result.Rows = append(result.Rows, historyRowToProto(row))
	}
	for _, line := range window.Lines {
		result.Lines = append(result.Lines, historyLineToProto(line, window.Rows))
	}
	return result
}

// HistoryCopyToProto 把 core frozen-history copy 文本包装为公共 API result。
func HistoryCopyToProto(text string) *apipb.HistoryCopyResult {
	return &apipb.HistoryCopyResult{Text: text, Done: true}
}

func HistoryCopyChunkToProto(result history.HistoryCopyChunkResult) *apipb.HistoryCopyResult {
	response := &apipb.HistoryCopyResult{Text: result.Text, Done: result.Done}
	if !result.Done && result.Next.LineID != 0 {
		response.Next = &apipb.HistoryTextPosition{LineId: uint64(result.Next.LineID), Col: int32(result.Next.Col)}
	}
	return response
}

func HistorySearchToProto(endpointID string, result history.HistorySearchResult) *apipb.HistorySearchResult {
	response := &apipb.HistorySearchResult{Found: result.Found, Wrapped: result.Wrapped}
	if !result.Found {
		return response
	}
	response.Match = &apipb.HistoryRange{
		StartLineId: uint64(result.Match.Start.LineID), StartCol: int32(result.Match.Start.Col),
		EndLineId: uint64(result.Match.End.LineID), EndCol: int32(result.Match.End.Col),
	}
	response.Window = HistoryWindowToProto(endpointID, result.Window)
	return response
}

// AcknowledgeToProto 返回无附加 payload 的成功确认。
func AcknowledgeToProto() *apipb.AcknowledgeResult {
	return &apipb.AcknowledgeResult{}
}

// HistoryBacklogToProto 返回不包含 history payload 的诊断投影。
func HistoryBacklogToProto(endpointID string, status corev2.HistoryBacklogStatus) *apipb.HistoryBacklogStatusResult {
	return &apipb.HistoryBacklogStatusResult{
		Terminal: &apipb.TerminalRef{EndpointId: endpointID, TerminalId: status.TerminalID}, HistoryEnabled: status.HistoryEnabled,
		OutputBufferPolicy: string(status.OutputBufferPolicy), BufferCapacityBytes: status.BufferCapacityBytes,
		ResidentBytes: status.ResidentBytes, AggregateResidentBytes: status.AggregateResidentBytes,
		AggregateBudgetBytes: status.AggregateBudgetBytes, DroppedBytes: status.DroppedBytes,
		GapCount: status.GapCount, OutputBufferWaitNanos: status.OutputBufferWaitNanos,
		Unavailable: status.Unavailable, UnavailableReason: status.UnavailableReason, Closed: status.Closed,
	}
}

// NativeScreenToProto 把 latest-only core native screen 转为公共 Proto projection。
func NativeScreenToProto(endpointID string, snapshot corev2.NativeScreenSnapshot) *apipb.NativeScreenResult {
	result := &apipb.NativeScreenResult{
		Terminal: &apipb.TerminalRef{EndpointId: endpointID, TerminalId: snapshot.TerminalID}, LiveRevision: uint64(snapshot.Revision),
		Size: &apipb.TerminalSize{Cols: uint32(snapshot.Size.Cols), Rows: uint32(snapshot.Size.Rows)}, AlternateScreen: snapshot.AltScreen,
		Cursor: cursorToProto(snapshot.Cursor), Modes: modesToProto(snapshot.Modes), TimestampUnixNano: snapshot.Timestamp.UnixNano(),
		BaseRevision: uint64(snapshot.BaseRevision), FullReplace: snapshot.FullReplace,
	}
	for _, rowCopy := range snapshot.RowCopies {
		result.RowCopies = append(result.RowCopies, &apipb.ScreenRowCopy{
			SourceRow: int32(rowCopy.SourceRow), DestinationRow: int32(rowCopy.DestinationRow), Count: int32(rowCopy.Count),
		})
	}
	for _, row := range snapshot.Rows {
		result.RowReplacements = append(result.RowReplacements, &apipb.ScreenRowReplace{RowIndex: int32(row.Index), Row: vtermRowToProto(row.Cells)})
	}
	return result
}

func historyWindowModeFromProto(mode apipb.HistoryWindowMode) history.HistoryWindowMode {
	switch mode {
	case apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST:
		return history.HistoryWindowModeLatest
	case apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDER:
		return history.HistoryWindowModeOlder
	case apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_NEWER:
		return history.HistoryWindowModeNewer
	case apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDEST:
		return history.HistoryWindowModeOldest
	default:
		return ""
	}
}

func historyWindowOperationToProto(operation history.HistoryWindowOp) apipb.HistoryWindowOperation {
	switch operation {
	case history.HistoryWindowPrepend:
		return apipb.HistoryWindowOperation_HISTORY_WINDOW_OPERATION_PREPEND
	case history.HistoryWindowAppend:
		return apipb.HistoryWindowOperation_HISTORY_WINDOW_OPERATION_APPEND
	default:
		return apipb.HistoryWindowOperation_HISTORY_WINDOW_OPERATION_REPLACE
	}
}

func historyCursorFromProto(cursor *apipb.HistoryCursor, generation uint64, token string) history.HistoryCursor {
	if cursor == nil {
		return history.HistoryCursor{}
	}
	return history.HistoryCursor{Segment: historySegmentFromProto(cursor.GetSegment()), LineID: history.LogicalLineID(cursor.GetLineId()), RowInLine: int(cursor.GetRowInLine()), Generation: history.Generation(generation), Token: history.HistoryToken(token), Valid: true}
}

func historyCursorToProto(cursor history.HistoryCursor) *apipb.HistoryCursor {
	if !cursor.Valid {
		return nil
	}
	return &apipb.HistoryCursor{LineId: uint64(cursor.LineID), RowInLine: int32(cursor.RowInLine), Segment: historySegmentToProto(cursor.Segment)}
}

func historySegmentFromProto(segment apipb.HistoryCursorSegment) history.HistorySegment {
	switch segment {
	case apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_COMMITTED:
		return history.HistorySegmentCommitted
	case apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_CURRENT_PRIMARY_FRAME:
		return history.HistorySegmentCurrentPrimaryFrame
	case apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_ARCHIVED_PRIMARY_FRAME:
		return history.HistorySegmentArchivedPrimaryFrame
	case apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_CURRENT_ALT_FRAME:
		return history.HistorySegmentCurrentAltFrame
	default:
		return ""
	}
}

func historySegmentToProto(segment history.HistorySegment) apipb.HistoryCursorSegment {
	switch segment {
	case history.HistorySegmentCommitted:
		return apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_COMMITTED
	case history.HistorySegmentCurrentPrimaryFrame:
		return apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_CURRENT_PRIMARY_FRAME
	case history.HistorySegmentArchivedPrimaryFrame:
		return apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_ARCHIVED_PRIMARY_FRAME
	case history.HistorySegmentCurrentAltFrame:
		return apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_CURRENT_ALT_FRAME
	default:
		return apipb.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_UNSPECIFIED
	}
}

func historyRowToProto(row history.HistoryRow) *apipb.HistoryRow {
	return &apipb.HistoryRow{
		Row: historyCellsToProto(row.Cells), RowKind: string(row.Kind), Wrapped: row.Wrapped,
		Ownership: historyRowOwnershipToProto(row), Segment: historySegmentToProto(row.Segment),
		SessionId: uint64(row.SessionID), FrameId: uint64(row.FrameID), FixedGrid: row.FixedGrid,
		ScreenCols: int32(row.ScreenCols), ScreenRows: int32(row.ScreenRow), ScreenRowSet: row.ScreenRowSet,
		LogicalLineId: uint64(row.LineID), RowInLine: int32(row.RowInLine),
		TimestampUnixNano: unixNanoOrZero(row.Timestamp),
	}
}

func historyLineToProto(line history.HistoryLineSpan, rows []history.HistoryRow) *apipb.HistoryLineSpan {
	end := line.EndRow
	if end > line.StartRow {
		end--
	}
	fixedGrid := false
	screenCols := 0
	if line.StartRow >= 0 && line.StartRow < len(rows) {
		fixedGrid = rows[line.StartRow].FixedGrid
		screenCols = rows[line.StartRow].ScreenCols
	}
	return &apipb.HistoryLineSpan{StartRow: int32(line.StartRow), EndRow: int32(end), RowKind: string(line.Kind), LogicalLineId: uint64(line.LogicalLineID), SessionId: uint64(line.SessionID), FrameId: uint64(line.FrameID), FixedGrid: fixedGrid, ScreenCols: int32(screenCols), TimestampStartUnixNano: unixNanoOrZero(line.TimestampStart), TimestampEndUnixNano: unixNanoOrZero(line.TimestampEnd), ClippedBefore: line.ClippedBefore, ClippedAfter: line.ClippedAfter}
}

func historyCellsToProto(cells []history.Cell) *apipb.ScreenRow {
	row := &apipb.ScreenRow{}
	for index, cell := range cells {
		if last := lastScreenCell(row); last != nil && historyCellsShareRun(cells[index-1], cell) {
			last.Content += cell.Text
			last.Width += int32(cell.Width)
			continue
		}
		row.Cells = append(row.Cells, &apipb.ScreenCell{Content: cell.Text, Width: int32(cell.Width), Style: historyStyleToProto(cell.Style), LinkUrl: cell.LinkURL, LinkParams: cell.LinkParams})
	}
	return row
}

func historyCellsShareRun(left, right history.Cell) bool {
	// Web history reflows each run by grapheme count, so a run may only contain
	// graphemes with the same authoritative terminal width.
	return left.Width == right.Width && left.Style == right.Style && left.LinkURL == right.LinkURL && left.LinkParams == right.LinkParams
}

func historyStyleToProto(style history.CellStyle) *apipb.CellStyle {
	return &apipb.CellStyle{Foreground: style.FG, Background: style.BG, Bold: style.Bold, Italic: style.Italic, Underline: style.Underline, Blink: style.Blink, Reverse: style.Reverse, Strikethrough: style.Strikethrough}
}

func historyRowOwnershipToProto(row history.HistoryRow) apipb.RowOwnership {
	if row.Segment == history.HistorySegmentCommitted && row.Committed {
		return apipb.RowOwnership_ROW_OWNERSHIP_PERSISTED
	}
	if row.Segment == history.HistorySegmentCommitted {
		return apipb.RowOwnership_ROW_OWNERSHIP_LIVE_TAIL_LIVE
	}
	return apipb.RowOwnership_ROW_OWNERSHIP_SCREEN
}

func vtermRowToProto(cells []vterm.Cell) *apipb.ScreenRow {
	row := &apipb.ScreenRow{}
	var previous vterm.Cell
	hasPrevious := false
	for _, cell := range cells {
		// A width-zero empty cell is the continuation column already occupied by
		// the preceding wide glyph. It has no independent visual content or style.
		if cell.Content == "" && cell.Width == 0 {
			continue
		}
		if last := lastScreenCell(row); last != nil && hasPrevious && vtermCellsShareRun(previous, cell) {
			last.Content += cell.Content
			last.Width += int32(cell.Width)
			previous = cell
			continue
		}
		row.Cells = append(row.Cells, &apipb.ScreenCell{Content: cell.Content, Width: int32(cell.Width), Style: vtermStyleToProto(cell.Style), LinkUrl: cell.LinkURL, LinkParams: cell.LinkParams})
		previous = cell
		hasPrevious = true
	}
	return row
}

func lastScreenCell(row *apipb.ScreenRow) *apipb.ScreenCell {
	if row == nil || len(row.Cells) == 0 {
		return nil
	}
	return row.Cells[len(row.Cells)-1]
}

func vtermCellsShareRun(left, right vterm.Cell) bool {
	return left.Style == right.Style && left.LinkURL == right.LinkURL && left.LinkParams == right.LinkParams
}

func vtermStyleToProto(style vterm.CellStyle) *apipb.CellStyle {
	return &apipb.CellStyle{Foreground: style.FG, Background: style.BG, Bold: style.Bold, Italic: style.Italic, Underline: style.Underline, Blink: style.Blink, Reverse: style.Reverse, Strikethrough: style.Strikethrough}
}

func cursorToProto(cursor vterm.CursorState) *apipb.TerminalCursor {
	shape := apipb.CursorShape_CURSOR_SHAPE_UNSPECIFIED
	switch cursor.Shape {
	case vterm.CursorBlock:
		shape = apipb.CursorShape_CURSOR_SHAPE_BLOCK
	case vterm.CursorUnderline:
		shape = apipb.CursorShape_CURSOR_SHAPE_UNDERLINE
	case vterm.CursorBar:
		shape = apipb.CursorShape_CURSOR_SHAPE_BAR
	}
	return &apipb.TerminalCursor{Row: int32(cursor.Row), Col: int32(cursor.Col), Visible: cursor.Visible, Shape: shape, Blink: cursor.Blink}
}

func modesToProto(modes vterm.TerminalModes) *apipb.TerminalModes {
	return &apipb.TerminalModes{AlternateScreen: modes.AlternateScreen, AlternateScroll: modes.AlternateScroll, MouseTracking: modes.MouseTracking, MouseX10: modes.MouseX10, MouseNormal: modes.MouseNormal, MouseButtonEvent: modes.MouseButtonEvent, MouseAnyEvent: modes.MouseAnyEvent, MouseSgr: modes.MouseSGR, BracketedPaste: modes.BracketedPaste, ApplicationCursor: modes.ApplicationCursor, AutoWrap: modes.AutoWrap}
}

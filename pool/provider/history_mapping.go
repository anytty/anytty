package provider

import (
	corev2 "github.com/anytty/anytty/pool/core"
	"github.com/anytty/anytty/pool/core/history"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
	vterm "github.com/anytty/anytty/vterm/vterm"
)

// 本文件把 core history/live DTO 映射为 provider-native 消息；语义与
// api_mapping 的 apipb 映射一致，但 provider 不携带 endpoint/session correlation。

func HistoryWindowRequestFromProto(command *providerv1.HistoryWindowCommand) history.HistoryWindowRequest {
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
func HistoryCopyRequestFromProto(command *providerv1.HistoryCopyCommand) history.HistoryCopyRequest {
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

func HistoryCopyChunkRequestFromProto(command *providerv1.HistoryCopyCommand) history.HistoryCopyChunkRequest {
	return history.HistoryCopyChunkRequest{
		HistoryCopyRequest: HistoryCopyRequestFromProto(command),
		MaxLines:           int(command.GetMaxLines()),
		MaxBytes:           int(command.GetMaxBytes()),
	}
}

func HistorySearchRequestFromProto(command *providerv1.HistorySearchCommand) history.HistorySearchRequest {
	if command == nil {
		return history.HistorySearchRequest{}
	}
	direction := history.HistorySearchForward
	if command.GetDirection() == providerv1.HistorySearchDirection_HISTORY_SEARCH_DIRECTION_BACKWARD {
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
		Scan:          command.GetScan(),
		MaxMatches:    int(command.GetMaxMatches()),
		Start: history.HistoryCopyPosition{
			LineID: history.LogicalLineID(command.GetStart().GetLineId()),
			Col:    int(command.GetStart().GetCol()),
		},
	}
}

func historySearchModeFromProto(mode providerv1.HistorySearchMode) history.HistorySearchMode {
	switch mode {
	case providerv1.HistorySearchMode_HISTORY_SEARCH_MODE_GLOB:
		return history.HistorySearchModeGlob
	case providerv1.HistorySearchMode_HISTORY_SEARCH_MODE_REGEX:
		return history.HistorySearchModeRegex
	default:
		return history.HistorySearchModeText
	}
}

// HistoryWindowToProto 把 core authoritative window 投影为 generated Proto。
func historyWindowToProto(_ string, window history.HistoryWindow) *providerv1.HistoryWindowResult {
	result := &providerv1.HistoryWindowResult{
		Terminal:          &providerv1.TerminalRef{TerminalId: window.TerminalID},
		Token:             string(window.Token),
		Operation:         historyWindowOperationToProto(window.Op),
		Size:              &providerv1.Size{Cols: uint32(window.Cols)},
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
		result.ViewportAnchor = &providerv1.HistoryViewportAnchor{
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
func HistoryCopyToProto(text string) *providerv1.HistoryCopyResult {
	return &providerv1.HistoryCopyResult{Text: text, Done: true}
}

func HistoryCopyChunkToProto(result history.HistoryCopyChunkResult) *providerv1.HistoryCopyResult {
	response := &providerv1.HistoryCopyResult{Text: result.Text, Done: result.Done}
	if !result.Done && result.Next.LineID != 0 {
		response.Next = &providerv1.HistoryTextPosition{LineId: uint64(result.Next.LineID), Col: int32(result.Next.Col)}
	}
	return response
}

func HistorySearchToProto(endpointID string, result history.HistorySearchResult) *providerv1.HistorySearchResult {
	response := &providerv1.HistorySearchResult{Found: result.Found, Wrapped: result.Wrapped, ScanDone: result.Done}
	for _, match := range result.Matches {
		response.ScanMatches = append(response.ScanMatches, &providerv1.HistoryRange{
			StartLineId: uint64(match.Start.LineID), StartCol: int32(match.Start.Col),
			EndLineId: uint64(match.End.LineID), EndCol: int32(match.End.Col),
		})
	}
	if result.Next.LineID != 0 {
		response.ScanNext = &providerv1.HistoryTextPosition{LineId: uint64(result.Next.LineID), Col: int32(result.Next.Col)}
	}
	if len(result.Matches) > 0 {
		return response
	}
	if !result.Found {
		return response
	}
	response.Match = &providerv1.HistoryRange{
		StartLineId: uint64(result.Match.Start.LineID), StartCol: int32(result.Match.Start.Col),
		EndLineId: uint64(result.Match.End.LineID), EndCol: int32(result.Match.End.Col),
	}
	response.Window = historyWindowToProto(endpointID, result.Window)
	return response
}

// AcknowledgeToProto 返回无附加 payload 的成功确认。
func historyBacklogToProto(_ string, status corev2.HistoryBacklogStatus) *providerv1.HistoryBacklogStatusResult {
	return &providerv1.HistoryBacklogStatusResult{
		Terminal: &providerv1.TerminalRef{TerminalId: status.TerminalID}, HistoryEnabled: status.HistoryEnabled,
		OutputBufferPolicy: string(status.OutputBufferPolicy), BufferCapacityBytes: status.BufferCapacityBytes,
		ResidentBytes: status.ResidentBytes, AggregateResidentBytes: status.AggregateResidentBytes,
		AggregateBudgetBytes: status.AggregateBudgetBytes, DroppedBytes: status.DroppedBytes,
		GapCount: status.GapCount, OutputBufferWaitNanos: status.OutputBufferWaitNanos,
		Unavailable: status.Unavailable, UnavailableReason: status.UnavailableReason, Closed: status.Closed,
	}
}

// NativeScreenToProto 把 latest-only core native screen 转为公共 Proto projection。
func nativeScreenToProto(_ string, snapshot corev2.NativeScreenSnapshot) *providerv1.NativeScreenResult {
	result := &providerv1.NativeScreenResult{
		Terminal: &providerv1.TerminalRef{TerminalId: snapshot.TerminalID}, LiveRevision: uint64(snapshot.Revision),
		Size: &providerv1.Size{Cols: uint32(snapshot.Size.Cols), Rows: uint32(snapshot.Size.Rows)}, AlternateScreen: snapshot.AltScreen,
		Cursor: cursorToProto(snapshot.Cursor), Modes: modesToProto(snapshot.Modes), TimestampUnixNano: snapshot.Timestamp.UnixNano(),
		BaseRevision: uint64(snapshot.BaseRevision), FullReplace: snapshot.FullReplace,
	}
	for _, rowCopy := range snapshot.RowCopies {
		result.RowCopies = append(result.RowCopies, &providerv1.ScreenRowCopy{
			SourceRow: int32(rowCopy.SourceRow), DestinationRow: int32(rowCopy.DestinationRow), Count: int32(rowCopy.Count),
		})
	}
	for _, row := range snapshot.Rows {
		result.RowReplacements = append(result.RowReplacements, &providerv1.ScreenRowReplace{
			RowIndex: int32(row.Index), Row: vtermRowToProto(row.Cells, row.Wrapped),
		})
	}
	return result
}

func historyWindowModeFromProto(mode providerv1.HistoryWindowMode) history.HistoryWindowMode {
	switch mode {
	case providerv1.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST:
		return history.HistoryWindowModeLatest
	case providerv1.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDER:
		return history.HistoryWindowModeOlder
	case providerv1.HistoryWindowMode_HISTORY_WINDOW_MODE_NEWER:
		return history.HistoryWindowModeNewer
	case providerv1.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDEST:
		return history.HistoryWindowModeOldest
	default:
		return ""
	}
}

func historyWindowOperationToProto(operation history.HistoryWindowOp) providerv1.HistoryWindowOperation {
	switch operation {
	case history.HistoryWindowPrepend:
		return providerv1.HistoryWindowOperation_HISTORY_WINDOW_OPERATION_PREPEND
	case history.HistoryWindowAppend:
		return providerv1.HistoryWindowOperation_HISTORY_WINDOW_OPERATION_APPEND
	default:
		return providerv1.HistoryWindowOperation_HISTORY_WINDOW_OPERATION_REPLACE
	}
}

func historyCursorFromProto(cursor *providerv1.HistoryCursor, generation uint64, token string) history.HistoryCursor {
	if cursor == nil {
		return history.HistoryCursor{}
	}
	return history.HistoryCursor{Segment: historySegmentFromProto(cursor.GetSegment()), LineID: history.LogicalLineID(cursor.GetLineId()), RowInLine: int(cursor.GetRowInLine()), Generation: history.Generation(generation), Token: history.HistoryToken(token), Valid: true}
}

func historyCursorToProto(cursor history.HistoryCursor) *providerv1.HistoryCursor {
	if !cursor.Valid {
		return nil
	}
	return &providerv1.HistoryCursor{LineId: uint64(cursor.LineID), RowInLine: int32(cursor.RowInLine), Segment: historySegmentToProto(cursor.Segment)}
}

func historySegmentFromProto(segment providerv1.HistoryCursorSegment) history.HistorySegment {
	switch segment {
	case providerv1.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_COMMITTED:
		return history.HistorySegmentCommitted
	case providerv1.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_CURRENT_PRIMARY_FRAME:
		return history.HistorySegmentCurrentPrimaryFrame
	case providerv1.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_ARCHIVED_PRIMARY_FRAME:
		return history.HistorySegmentArchivedPrimaryFrame
	case providerv1.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_CURRENT_ALT_FRAME:
		return history.HistorySegmentCurrentAltFrame
	default:
		return ""
	}
}

func historySegmentToProto(segment history.HistorySegment) providerv1.HistoryCursorSegment {
	switch segment {
	case history.HistorySegmentCommitted:
		return providerv1.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_COMMITTED
	case history.HistorySegmentCurrentPrimaryFrame:
		return providerv1.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_CURRENT_PRIMARY_FRAME
	case history.HistorySegmentArchivedPrimaryFrame:
		return providerv1.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_ARCHIVED_PRIMARY_FRAME
	case history.HistorySegmentCurrentAltFrame:
		return providerv1.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_CURRENT_ALT_FRAME
	default:
		return providerv1.HistoryCursorSegment_HISTORY_CURSOR_SEGMENT_UNSPECIFIED
	}
}

func historyRowToProto(row history.HistoryRow) *providerv1.HistoryRow {
	screenRow := historyCellsToProto(row.Cells)
	screenRow.Wrapped = row.Wrapped
	return &providerv1.HistoryRow{
		Row: screenRow, RowKind: string(row.Kind), Wrapped: row.Wrapped,
		Ownership: historyRowOwnershipToProto(row), Segment: historySegmentToProto(row.Segment),
		SessionId: uint64(row.SessionID), FrameId: uint64(row.FrameID), FixedGrid: row.FixedGrid,
		ScreenCols: int32(row.ScreenCols), ScreenRows: int32(row.ScreenRow), ScreenRowSet: row.ScreenRowSet,
		LogicalLineId: uint64(row.LineID), RowInLine: int32(row.RowInLine),
		TimestampUnixNano: unixNanoOrZero(row.Timestamp),
	}
}

func historyLineToProto(line history.HistoryLineSpan, rows []history.HistoryRow) *providerv1.HistoryLineSpan {
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
	return &providerv1.HistoryLineSpan{StartRow: int32(line.StartRow), EndRow: int32(end), RowKind: string(line.Kind), LogicalLineId: uint64(line.LogicalLineID), SessionId: uint64(line.SessionID), FrameId: uint64(line.FrameID), FixedGrid: fixedGrid, ScreenCols: int32(screenCols), TimestampStartUnixNano: unixNanoOrZero(line.TimestampStart), TimestampEndUnixNano: unixNanoOrZero(line.TimestampEnd), ClippedBefore: line.ClippedBefore, ClippedAfter: line.ClippedAfter}
}

func historyCellsToProto(cells []history.Cell) *providerv1.ScreenRow {
	row := &providerv1.ScreenRow{}
	for index, cell := range cells {
		if last := lastScreenCell(row); last != nil && historyCellsShareRun(cells[index-1], cell) {
			last.Content += cell.Text
			last.Width += int32(cell.Width)
			continue
		}
		row.Cells = append(row.Cells, &providerv1.ScreenCell{Content: cell.Text, Width: int32(cell.Width), Style: historyStyleToProto(cell.Style), LinkUrl: cell.LinkURL, LinkParams: cell.LinkParams})
	}
	return row
}

func historyCellsShareRun(left, right history.Cell) bool {
	// Web history reflows each run by grapheme count, so a run may only contain
	// graphemes with the same authoritative terminal width.
	return left.Width == right.Width && left.Style == right.Style && left.LinkURL == right.LinkURL && left.LinkParams == right.LinkParams
}

func historyStyleToProto(style history.CellStyle) *providerv1.CellStyle {
	return &providerv1.CellStyle{Foreground: style.FG, Background: style.BG, Bold: style.Bold, Italic: style.Italic, Underline: style.Underline, Blink: style.Blink, Reverse: style.Reverse, Strikethrough: style.Strikethrough}
}

func historyRowOwnershipToProto(row history.HistoryRow) providerv1.RowOwnership {
	if row.Segment == history.HistorySegmentCommitted && row.Committed {
		return providerv1.RowOwnership_ROW_OWNERSHIP_PERSISTED
	}
	if row.Segment == history.HistorySegmentCommitted {
		return providerv1.RowOwnership_ROW_OWNERSHIP_LIVE_TAIL_LIVE
	}
	return providerv1.RowOwnership_ROW_OWNERSHIP_SCREEN
}

func vtermRowToProto(cells []vterm.Cell, wrapped bool) *providerv1.ScreenRow {
	row := &providerv1.ScreenRow{Wrapped: wrapped}
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
		row.Cells = append(row.Cells, &providerv1.ScreenCell{Content: cell.Content, Width: int32(cell.Width), Style: vtermStyleToProto(cell.Style), LinkUrl: cell.LinkURL, LinkParams: cell.LinkParams})
		previous = cell
		hasPrevious = true
	}
	return row
}

func lastScreenCell(row *providerv1.ScreenRow) *providerv1.ScreenCell {
	if row == nil || len(row.Cells) == 0 {
		return nil
	}
	return row.Cells[len(row.Cells)-1]
}

func vtermCellsShareRun(left, right vterm.Cell) bool {
	return left.Style == right.Style && left.LinkURL == right.LinkURL && left.LinkParams == right.LinkParams
}

func vtermStyleToProto(style vterm.CellStyle) *providerv1.CellStyle {
	return &providerv1.CellStyle{Foreground: style.FG, Background: style.BG, Bold: style.Bold, Italic: style.Italic, Underline: style.Underline, Blink: style.Blink, Reverse: style.Reverse, Strikethrough: style.Strikethrough}
}

func cursorToProto(cursor vterm.CursorState) *providerv1.TerminalCursor {
	shape := providerv1.CursorShape_CURSOR_SHAPE_UNSPECIFIED
	switch cursor.Shape {
	case vterm.CursorBlock:
		shape = providerv1.CursorShape_CURSOR_SHAPE_BLOCK
	case vterm.CursorUnderline:
		shape = providerv1.CursorShape_CURSOR_SHAPE_UNDERLINE
	case vterm.CursorBar:
		shape = providerv1.CursorShape_CURSOR_SHAPE_BAR
	}
	return &providerv1.TerminalCursor{Row: int32(cursor.Row), Col: int32(cursor.Col), Visible: cursor.Visible, Shape: shape, Blink: cursor.Blink}
}

func modesToProto(modes vterm.TerminalModes) *providerv1.TerminalModes {
	return &providerv1.TerminalModes{AlternateScreen: modes.AlternateScreen, AlternateScroll: modes.AlternateScroll, MouseTracking: modes.MouseTracking, MouseX10: modes.MouseX10, MouseNormal: modes.MouseNormal, MouseButtonEvent: modes.MouseButtonEvent, MouseAnyEvent: modes.MouseAnyEvent, MouseSgr: modes.MouseSGR, BracketedPaste: modes.BracketedPaste, ApplicationCursor: modes.ApplicationCursor, AutoWrap: modes.AutoWrap}
}

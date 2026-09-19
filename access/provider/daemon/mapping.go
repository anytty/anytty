package daemon

import (
	"github.com/anytty/anytty/proto/access/apipb"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
)

// ---- terminal projections ----------------------------------------------------

func terminalCreateSpecToProvider(spec *apipb.TerminalCreateSpec) *providerv1.TerminalCreateSpec {
	if spec == nil {
		return nil
	}
	return &providerv1.TerminalCreateSpec{
		TerminalId: spec.GetTerminalId(),
		Name:       spec.GetName(),
		Command:    append([]string(nil), spec.GetCommand()...),
		Cwd:        spec.GetCwd(),
		Env:        append([]string(nil), spec.GetEnv()...),
		Tags:       cloneStringMap(spec.GetTags()),
		Size:       sizeToProviderFromAPI(spec.GetSize()),
		Scrollback: &providerv1.TerminalScrollback{
			Size:         uint32(spec.GetScrollbackRows()),
			MaxBytes:     spec.GetScrollbackMaxBytes(),
			MaxAgeMillis: spec.GetScrollbackMaxAgeSeconds() * 1000,
		},
	}
}

func terminalInfoToAPI(endpointID string, info *providerv1.TerminalInfo) *apipb.TerminalInfo {
	if info == nil {
		return nil
	}
	projection := &apipb.TerminalInfo{
		Ref:                  &apipb.TerminalRef{EndpointId: endpointID, TerminalId: info.GetTerminalId()},
		Name:                 info.GetName(),
		Command:              append([]string(nil), info.GetCommand()...),
		Tags:                 cloneStringMap(info.GetTags()),
		Size:                 sizeToAPI(info.GetSize()),
		State:                terminalStateToAPI(info.GetState()),
		Cwd:                  info.GetCwd(),
		LiveCwd:              info.GetLiveCwd(),
		CreatedAtUnixNano:    info.GetCreatedAtUnixNano(),
		ExitedAtUnixNano:     info.GetExitedAtUnixNano(),
		ForegroundProcess:    info.GetForegroundProcess(),
		ForegroundCwd:        info.GetForegroundCwd(),
		LastOutputAtUnixNano: info.GetLastOutputAtUnixNano(),
		AttachmentCount:      info.GetAttachmentCount(),
	}
	// provider 只报告 exit code 是否为 0 之外的值；显式 0 也需要投影成指针。
	if info.GetExitCode() != 0 || info.GetState() == "exited" {
		code := info.GetExitCode()
		projection.ExitCode = &code
	}
	return projection
}

func terminalStateToAPI(state string) apipb.TerminalState {
	switch state {
	case "created":
		return apipb.TerminalState_TERMINAL_STATE_CREATED
	case "running":
		return apipb.TerminalState_TERMINAL_STATE_RUNNING
	case "exited":
		return apipb.TerminalState_TERMINAL_STATE_EXITED
	case "removed":
		return apipb.TerminalState_TERMINAL_STATE_REMOVED
	default:
		return apipb.TerminalState_TERMINAL_STATE_UNSPECIFIED
	}
}

func sizeToAPI(size *providerv1.Size) *apipb.TerminalSize {
	if size == nil {
		return nil
	}
	return &apipb.TerminalSize{Cols: size.GetCols(), Rows: size.GetRows()}
}

func sizeToProviderFromAPI(size *apipb.TerminalSize) *providerv1.Size {
	if size == nil {
		return nil
	}
	return &providerv1.Size{Cols: size.GetCols(), Rows: size.GetRows()}
}

// ---- attachments/resize ------------------------------------------------------

func attachmentModeToProvider(mode apipb.AttachmentMode) providerv1.AttachmentMode {
	switch mode {
	case apipb.AttachmentMode_ATTACHMENT_MODE_OBSERVER:
		return providerv1.AttachmentMode_ATTACHMENT_MODE_OBSERVER
	case apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR:
		return providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR
	default:
		return providerv1.AttachmentMode_ATTACHMENT_MODE_UNSPECIFIED
	}
}

func resizePolicyToProvider(policy apipb.ResizePolicy) providerv1.ResizePolicy {
	switch policy {
	case apipb.ResizePolicy_RESIZE_POLICY_OWNER:
		return providerv1.ResizePolicy_RESIZE_POLICY_OWNER
	case apipb.ResizePolicy_RESIZE_POLICY_FOLLOWER:
		return providerv1.ResizePolicy_RESIZE_POLICY_FOLLOWER
	case apipb.ResizePolicy_RESIZE_POLICY_OBSERVER:
		return providerv1.ResizePolicy_RESIZE_POLICY_OBSERVER
	default:
		return providerv1.ResizePolicy_RESIZE_POLICY_UNSPECIFIED
	}
}

func resizePolicyToAPI(policy providerv1.ResizePolicy) apipb.ResizePolicy {
	switch policy {
	case providerv1.ResizePolicy_RESIZE_POLICY_OWNER:
		return apipb.ResizePolicy_RESIZE_POLICY_OWNER
	case providerv1.ResizePolicy_RESIZE_POLICY_FOLLOWER:
		return apipb.ResizePolicy_RESIZE_POLICY_FOLLOWER
	case providerv1.ResizePolicy_RESIZE_POLICY_OBSERVER:
		return apipb.ResizePolicy_RESIZE_POLICY_OBSERVER
	default:
		return apipb.ResizePolicy_RESIZE_POLICY_UNSPECIFIED
	}
}

func attachmentModeToAPI(mode providerv1.AttachmentMode) apipb.AttachmentMode {
	switch mode {
	case providerv1.AttachmentMode_ATTACHMENT_MODE_OBSERVER:
		return apipb.AttachmentMode_ATTACHMENT_MODE_OBSERVER
	case providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR:
		return apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR
	default:
		return apipb.AttachmentMode_ATTACHMENT_MODE_UNSPECIFIED
	}
}

func resizeControlToAPI(control *providerv1.ResizeControl) *apipb.ResizeControl {
	if control == nil {
		return nil
	}
	projection := &apipb.ResizeControl{
		CanResize:      control.GetCanResize(),
		Reason:         resizeReasonToAPI(control.GetReason()),
		SizeLocked:     control.GetSizeLocked(),
		SurfaceId:      control.GetSurfaceId(),
		OwnerSurfaceId: control.GetOwnerSurfaceId(),
		OwnerViewId:    control.GetOwnerViewId(),
	}
	if ownership := control.GetResizeOwnership(); ownership != nil {
		projection.Ownership = &apipb.ResizeOwnership{
			OwnerAttachmentId: ownership.GetOwnerAttachmentId(),
			OwnerSurfaceId:    ownership.GetOwnerSurfaceId(),
			OwnerViewId:       ownership.GetOwnerViewId(),
			Size:              sizeToAPI(ownership.GetSize()),
			SizeLocked:        ownership.GetSizeLocked(),
			Epoch:             ownership.GetEpoch(),
		}
	}
	return projection
}

func resizeReasonToAPI(reason providerv1.ResizeReason) apipb.ResizeControlReason {
	switch reason {
	case providerv1.ResizeReason_RESIZE_REASON_OWNER:
		return apipb.ResizeControlReason_RESIZE_CONTROL_REASON_OWNER
	case providerv1.ResizeReason_RESIZE_REASON_OBSERVER:
		return apipb.ResizeControlReason_RESIZE_CONTROL_REASON_OBSERVER
	case providerv1.ResizeReason_RESIZE_REASON_SIZE_LOCKED:
		return apipb.ResizeControlReason_RESIZE_CONTROL_REASON_SIZE_LOCKED
	default:
		return apipb.ResizeControlReason_RESIZE_CONTROL_REASON_FOLLOWER
	}
}

// ---- history requests --------------------------------------------------------

func historyWindowRequestToProvider(command *apipb.HistoryWindowCommand) *providerv1.HistoryWindowCommand {
	if command == nil {
		return nil
	}
	request := &providerv1.HistoryWindowCommand{
		Terminal:            &providerv1.TerminalRef{TerminalId: command.GetTerminal().GetTerminalId()},
		Mode:                providerv1.HistoryWindowMode(command.GetMode()),
		Limit:               command.GetLimit(),
		Cols:                command.GetCols(),
		Token:               command.GetToken(),
		HistoryGeneration:   command.GetHistoryGeneration(),
		BeforeCursor:        historyCursorToProvider(command.GetBeforeCursor()),
		AfterCursor:         historyCursorToProvider(command.GetAfterCursor()),
		BoundaryFirstLineId: command.GetBoundaryFirstLineId(),
		BoundaryLastLineId:  command.GetBoundaryLastLineId(),
		Range:               historyRangeToProvider(command.GetRange()),
	}
	return request
}

func historyCursorToProvider(cursor *apipb.HistoryCursor) *providerv1.HistoryCursor {
	if cursor == nil {
		return nil
	}
	return &providerv1.HistoryCursor{
		LineId: cursor.GetLineId(), RowInLine: cursor.GetRowInLine(),
		Segment: providerv1.HistoryCursorSegment(cursor.GetSegment()),
	}
}

func historyRangeToProvider(value *apipb.HistoryRange) *providerv1.HistoryRange {
	if value == nil {
		return nil
	}
	return &providerv1.HistoryRange{
		StartLineId: value.GetStartLineId(), StartCol: value.GetStartCol(),
		EndLineId: value.GetEndLineId(), EndCol: value.GetEndCol(),
	}
}

func historyCopyCommandToProvider(command *apipb.HistoryCopyCommand) *providerv1.HistoryCopyCommand {
	if command == nil {
		return nil
	}
	return &providerv1.HistoryCopyCommand{
		Terminal: &providerv1.TerminalRef{TerminalId: command.GetTerminal().GetTerminalId()},
		Window:   historyWindowRequestToProvider(command.GetWindow()),
		MaxLines: command.GetMaxLines(),
		MaxBytes: command.GetMaxBytes(),
	}
}

func historySearchCommandToProvider(command *apipb.HistorySearchCommand) *providerv1.HistorySearchCommand {
	if command == nil {
		return nil
	}
	return &providerv1.HistorySearchCommand{
		Terminal:          &providerv1.TerminalRef{TerminalId: command.GetTerminal().GetTerminalId()},
		Token:             command.GetToken(),
		HistoryGeneration: command.GetHistoryGeneration(),
		Query:             command.GetQuery(),
		Direction:         providerv1.HistorySearchDirection(command.GetDirection()),
		Cols:              command.GetCols(),
		Limit:             command.GetLimit(),
		Start:             historyTextPositionToProvider(command.GetStart()),
		Mode:              providerv1.HistorySearchMode(command.GetMode()),
		ContextBefore:     command.GetContextBefore(),
		Scan:              command.GetScan(),
		MaxMatches:        command.GetMaxMatches(),
	}
}

func historyTextPositionToProvider(position *apipb.HistoryTextPosition) *providerv1.HistoryTextPosition {
	if position == nil {
		return nil
	}
	return &providerv1.HistoryTextPosition{LineId: position.GetLineId(), Col: position.GetCol()}
}

// ---- history results ---------------------------------------------------------

func historyWindowToAPI(endpointID string, window *providerv1.HistoryWindowResult) *apipb.HistoryWindowResult {
	if window == nil {
		return nil
	}
	result := &apipb.HistoryWindowResult{
		Terminal:          &apipb.TerminalRef{EndpointId: endpointID, TerminalId: window.GetTerminal().GetTerminalId()},
		Token:             window.GetToken(),
		Operation:         apipb.HistoryWindowOperation(window.GetOperation()),
		Size:              sizeToAPI(window.GetSize()),
		LoadedRows:        window.GetLoadedRows(),
		TotalRows:         window.GetTotalRows(),
		LoadedLines:       window.GetLoadedLines(),
		LogicalTotal:      window.GetLogicalTotal(),
		HasMore:           window.GetHasMore(),
		HistoryGeneration: window.GetHistoryGeneration(),
		FirstRowId:        window.GetFirstRowId(),
		LastRowId:         window.GetLastRowId(),
		FirstLineId:       window.GetFirstLineId(),
		LastLineId:        window.GetLastLineId(),
		Cursor:            historyCursorToAPI(window.GetCursor()),
		TimestampUnixNano: window.GetTimestampUnixNano(),
		ViewportAnchor:    historyViewportAnchorToAPI(window.GetViewportAnchor()),
	}
	for _, row := range window.GetRows() {
		result.Rows = append(result.Rows, historyRowToAPI(row))
	}
	for _, line := range window.GetLines() {
		result.Lines = append(result.Lines, historyLineToAPI(line))
	}
	return result
}

func historyRowToAPI(row *providerv1.HistoryRow) *apipb.HistoryRow {
	if row == nil {
		return nil
	}
	projection := &apipb.HistoryRow{
		TimestampUnixNano: row.GetTimestampUnixNano(),
		RowKind:           row.GetRowKind(),
		Wrapped:           row.GetWrapped(),
		Ownership:         apipb.RowOwnership(row.GetOwnership()),
		Segment:           apipb.HistoryCursorSegment(row.GetSegment()),
		SessionId:         row.GetSessionId(),
		FrameId:           row.GetFrameId(),
		FixedGrid:         row.GetFixedGrid(),
		ScreenCols:        row.GetScreenCols(),
		ScreenRows:        row.GetScreenRows(),
		ScreenRowSet:      row.GetScreenRowSet(),
		LogicalLineId:     row.GetLogicalLineId(),
		RowInLine:         row.GetRowInLine(),
	}
	if row.GetRow() != nil {
		projection.Row = screenRowToAPI(row.GetRow())
	}
	return projection
}

func historyLineToAPI(line *providerv1.HistoryLineSpan) *apipb.HistoryLineSpan {
	if line == nil {
		return nil
	}
	return &apipb.HistoryLineSpan{
		StartRow: line.GetStartRow(), EndRow: line.GetEndRow(), RowKind: line.GetRowKind(),
		LogicalLineId: line.GetLogicalLineId(), SessionId: line.GetSessionId(), FrameId: line.GetFrameId(),
		FixedGrid: line.GetFixedGrid(), ScreenCols: line.GetScreenCols(),
		TimestampStartUnixNano: line.GetTimestampStartUnixNano(), TimestampEndUnixNano: line.GetTimestampEndUnixNano(),
		ClippedBefore: line.GetClippedBefore(), ClippedAfter: line.GetClippedAfter(),
	}
}

func historyViewportAnchorToAPI(anchor *providerv1.HistoryViewportAnchor) *apipb.HistoryViewportAnchor {
	if anchor == nil {
		return nil
	}
	return &apipb.HistoryViewportAnchor{
		TopLineId: anchor.GetTopLineId(), TopCellOffset: anchor.GetTopCellOffset(), AtEnd: anchor.GetAtEnd(),
		ScreenCols: anchor.GetScreenCols(), ScreenRows: anchor.GetScreenRows(),
	}
}

func screenRowToAPI(row *providerv1.ScreenRow) *apipb.ScreenRow {
	if row == nil {
		return nil
	}
	projection := &apipb.ScreenRow{Wrapped: row.GetWrapped(), TailFill: cellStyleToAPI(row.GetTailFill())}
	for _, cell := range row.GetCells() {
		projection.Cells = append(projection.Cells, &apipb.ScreenCell{
			Content: cell.GetContent(), Width: cell.GetWidth(), Style: cellStyleToAPI(cell.GetStyle()),
			LinkUrl: cell.GetLinkUrl(), LinkParams: cell.GetLinkParams(),
		})
	}
	return projection
}

func cellStyleToAPI(style *providerv1.CellStyle) *apipb.CellStyle {
	if style == nil {
		return nil
	}
	return &apipb.CellStyle{
		Foreground: style.GetForeground(), Background: style.GetBackground(), Bold: style.GetBold(),
		Italic: style.GetItalic(), Underline: style.GetUnderline(), Blink: style.GetBlink(),
		Reverse: style.GetReverse(), Strikethrough: style.GetStrikethrough(),
	}
}

func historyCursorToAPI(cursor *providerv1.HistoryCursor) *apipb.HistoryCursor {
	if cursor == nil {
		return nil
	}
	return &apipb.HistoryCursor{
		LineId: cursor.GetLineId(), RowInLine: cursor.GetRowInLine(),
		Segment: apipb.HistoryCursorSegment(cursor.GetSegment()),
	}
}

func historyCopyResultToAPI(result *providerv1.HistoryCopyResult) *apipb.HistoryCopyResult {
	if result == nil {
		return nil
	}
	projection := &apipb.HistoryCopyResult{Text: result.GetText(), Done: result.GetDone()}
	if result.GetNext() != nil {
		projection.Next = &apipb.HistoryTextPosition{LineId: result.GetNext().GetLineId(), Col: result.GetNext().GetCol()}
	}
	return projection
}

func historySearchResultToAPI(endpointID string, result *providerv1.HistorySearchResult) *apipb.HistorySearchResult {
	if result == nil {
		return nil
	}
	projection := &apipb.HistorySearchResult{
		Found:    result.GetFound(),
		Wrapped:  result.GetWrapped(),
		Window:   historyWindowToAPI(endpointID, result.GetWindow()),
		ScanDone: result.GetScanDone(),
	}
	if match := result.GetMatch(); match != nil {
		projection.Match = &apipb.HistoryRange{
			StartLineId: match.GetStartLineId(), StartCol: match.GetStartCol(),
			EndLineId: match.GetEndLineId(), EndCol: match.GetEndCol(),
		}
	}
	for _, match := range result.GetScanMatches() {
		projection.ScanMatches = append(projection.ScanMatches, &apipb.HistoryRange{
			StartLineId: match.GetStartLineId(), StartCol: match.GetStartCol(),
			EndLineId: match.GetEndLineId(), EndCol: match.GetEndCol(),
		})
	}
	if next := result.GetScanNext(); next != nil {
		projection.ScanNext = &apipb.HistoryTextPosition{LineId: next.GetLineId(), Col: next.GetCol()}
	}
	return projection
}

func historyBacklogToAPI(endpointID string, status *providerv1.HistoryBacklogStatusResult) *apipb.HistoryBacklogStatusResult {
	if status == nil {
		return nil
	}
	return &apipb.HistoryBacklogStatusResult{
		Terminal:               &apipb.TerminalRef{EndpointId: endpointID, TerminalId: status.GetTerminal().GetTerminalId()},
		HistoryEnabled:         status.GetHistoryEnabled(),
		OutputBufferPolicy:     status.GetOutputBufferPolicy(),
		BufferCapacityBytes:    status.GetBufferCapacityBytes(),
		ResidentBytes:          status.GetResidentBytes(),
		AggregateResidentBytes: status.GetAggregateResidentBytes(),
		AggregateBudgetBytes:   status.GetAggregateBudgetBytes(),
		DroppedBytes:           status.GetDroppedBytes(),
		GapCount:               status.GetGapCount(),
		OutputBufferWaitNanos:  status.GetOutputBufferWaitNanos(),
		Unavailable:            status.GetUnavailable(),
		UnavailableReason:      status.GetUnavailableReason(),
		Closed:                 status.GetClosed(),
	}
}

func nativeScreenToAPI(endpointID string, snapshot *providerv1.NativeScreenResult) *apipb.NativeScreenResult {
	if snapshot == nil {
		return nil
	}
	result := &apipb.NativeScreenResult{
		Terminal:          &apipb.TerminalRef{EndpointId: endpointID, TerminalId: snapshot.GetTerminal().GetTerminalId()},
		LiveRevision:      snapshot.GetLiveRevision(),
		Size:              sizeToAPI(snapshot.GetSize()),
		AlternateScreen:   snapshot.GetAlternateScreen(),
		TimestampUnixNano: snapshot.GetTimestampUnixNano(),
		BaseRevision:      snapshot.GetBaseRevision(),
		FullReplace:       snapshot.GetFullReplace(),
	}
	if cursor := snapshot.GetCursor(); cursor != nil {
		result.Cursor = &apipb.TerminalCursor{
			Row: cursor.GetRow(), Col: cursor.GetCol(), Visible: cursor.GetVisible(),
			Shape: apipb.CursorShape(cursor.GetShape()), Blink: cursor.GetBlink(),
		}
	}
	if modes := snapshot.GetModes(); modes != nil {
		result.Modes = &apipb.TerminalModes{
			AlternateScreen: modes.GetAlternateScreen(), AlternateScroll: modes.GetAlternateScroll(),
			MouseTracking: modes.GetMouseTracking(), MouseX10: modes.GetMouseX10(), MouseNormal: modes.GetMouseNormal(),
			MouseButtonEvent: modes.GetMouseButtonEvent(), MouseAnyEvent: modes.GetMouseAnyEvent(), MouseSgr: modes.GetMouseSgr(),
			BracketedPaste: modes.GetBracketedPaste(), ApplicationCursor: modes.GetApplicationCursor(),
			AutoWrap: modes.GetAutoWrap(),
		}
	}
	for _, replacement := range snapshot.GetRowReplacements() {
		result.RowReplacements = append(result.RowReplacements, &apipb.ScreenRowReplace{
			RowIndex: replacement.GetRowIndex(), Row: screenRowToAPI(replacement.GetRow()),
		})
	}
	for _, copyRow := range snapshot.GetRowCopies() {
		result.RowCopies = append(result.RowCopies, &apipb.ScreenRowCopy{
			SourceRow: copyRow.GetSourceRow(), DestinationRow: copyRow.GetDestinationRow(), Count: copyRow.GetCount(),
		})
	}
	return result
}

// ---- path defaults/dirs ------------------------------------------------------

func terminalDefaultsToAPI(defaults *providerv1.TerminalDefaults) *apipb.TerminalDefaultsResult {
	if defaults == nil {
		return &apipb.TerminalDefaultsResult{}
	}
	return &apipb.TerminalDefaultsResult{Defaults: &apipb.TerminalDefaults{
		DefaultCommand: append([]string(nil), defaults.GetCommand()...),
		DefaultCwd:     defaults.GetCwd(),
		Platform:       defaults.GetPlatform(),
	}}
}

func pathDirectoriesToAPI(directories *providerv1.PathListDirectoriesResult) *apipb.PathListDirectoriesResult {
	if directories == nil {
		return &apipb.PathListDirectoriesResult{}
	}
	result := &apipb.PathListDirectoriesResult{
		BasePath: directories.GetBasePath(), Missing: directories.GetMissing(), Truncated: directories.GetTruncated(),
	}
	for _, entry := range directories.GetEntries() {
		result.Entries = append(result.Entries, &apipb.PathDirectoryEntry{Name: entry.GetName(), Path: entry.GetPath()})
	}
	return result
}

// ---- helpers -----------------------------------------------------------------

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

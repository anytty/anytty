// Package terminalmap 是 access 消费方持有的 apipb ↔ providerv1 投影层。
//
// access/server 终结 access wire 后，把 terminal family 命令翻译为 typed
// providerv1 DTO 交给 terminal provider，并把 provider 结果投影回同一套 apipb
// result/event envelope。投影只做字段搬运，不解释领域语义；wire 行为由
// access/server 的调用顺序决定。
package terminalmap

import (
	"github.com/anytty/anytty/proto/access/apipb"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
)

// ---- terminal projections ----------------------------------------------------

// TerminalCreateSpecToProvider 把 apipb 建终端载荷投影为 provider wire spec。
func TerminalCreateSpecToProvider(spec *apipb.TerminalCreateSpec) *providerv1.TerminalCreateSpec {
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
		Size:       SizeFromAPI(spec.GetSize()),
		Scrollback: &providerv1.TerminalScrollback{
			Size:         uint32(spec.GetScrollbackRows()),
			MaxBytes:     spec.GetScrollbackMaxBytes(),
			MaxAgeMillis: spec.GetScrollbackMaxAgeSeconds() * 1000,
		},
	}
}

// TerminalInfoToAPI 把 provider terminal 快照投影为客户端可见 apipb 快照。
func TerminalInfoToAPI(endpointID string, info *providerv1.TerminalInfo) *apipb.TerminalInfo {
	if info == nil {
		return nil
	}
	projection := &apipb.TerminalInfo{
		Ref:                  &apipb.TerminalRef{EndpointId: endpointID, TerminalId: info.GetTerminalId()},
		Name:                 info.GetName(),
		Command:              append([]string(nil), info.GetCommand()...),
		Tags:                 cloneStringMap(info.GetTags()),
		Size:                 SizeToAPI(info.GetSize()),
		State:                TerminalStateToAPI(info.GetState()),
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

// TerminalStateToAPI 投影 provider 状态字符串为 apipb 枚举。
func TerminalStateToAPI(state string) apipb.TerminalState {
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

// SizeToAPI 投影 provider size 为 apipb size。
func SizeToAPI(size *providerv1.Size) *apipb.TerminalSize {
	if size == nil {
		return nil
	}
	return &apipb.TerminalSize{Cols: size.GetCols(), Rows: size.GetRows()}
}

// SizeFromAPI 投影 apipb size 为 provider size。
func SizeFromAPI(size *apipb.TerminalSize) *providerv1.Size {
	if size == nil {
		return nil
	}
	return &providerv1.Size{Cols: size.GetCols(), Rows: size.GetRows()}
}

// ---- attachments/resize ------------------------------------------------------

// AttachmentModeToProvider 投影 apipb attachment mode。
func AttachmentModeToProvider(mode apipb.AttachmentMode) providerv1.AttachmentMode {
	switch mode {
	case apipb.AttachmentMode_ATTACHMENT_MODE_OBSERVER:
		return providerv1.AttachmentMode_ATTACHMENT_MODE_OBSERVER
	case apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR:
		return providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR
	default:
		return providerv1.AttachmentMode_ATTACHMENT_MODE_UNSPECIFIED
	}
}

// ResizePolicyToProvider 投影 apipb resize policy。
func ResizePolicyToProvider(policy apipb.ResizePolicy) providerv1.ResizePolicy {
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

// ResizePolicyToAPI 投影 provider resize policy。
func ResizePolicyToAPI(policy providerv1.ResizePolicy) apipb.ResizePolicy {
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

// AttachmentModeToAPI 投影 provider attachment mode。
func AttachmentModeToAPI(mode providerv1.AttachmentMode) apipb.AttachmentMode {
	switch mode {
	case providerv1.AttachmentMode_ATTACHMENT_MODE_OBSERVER:
		return apipb.AttachmentMode_ATTACHMENT_MODE_OBSERVER
	case providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR:
		return apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR
	default:
		return apipb.AttachmentMode_ATTACHMENT_MODE_UNSPECIFIED
	}
}

// ResizeControlToAPI 投影 provider resize control。
func ResizeControlToAPI(control *providerv1.ResizeControl) *apipb.ResizeControl {
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
			Size:              SizeToAPI(ownership.GetSize()),
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

// HistoryWindowRequestToProvider 投影 apipb history window 请求。
func HistoryWindowRequestToProvider(command *apipb.HistoryWindowCommand) *providerv1.HistoryWindowCommand {
	if command == nil {
		return nil
	}
	return &providerv1.HistoryWindowCommand{
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

// HistoryCopyCommandToProvider 投影 apipb history copy 请求。
func HistoryCopyCommandToProvider(command *apipb.HistoryCopyCommand) *providerv1.HistoryCopyCommand {
	if command == nil {
		return nil
	}
	return &providerv1.HistoryCopyCommand{
		Terminal: &providerv1.TerminalRef{TerminalId: command.GetTerminal().GetTerminalId()},
		Window:   HistoryWindowRequestToProvider(command.GetWindow()),
		MaxLines: command.GetMaxLines(),
		MaxBytes: command.GetMaxBytes(),
	}
}

// HistorySearchCommandToProvider 投影 apipb history search 请求。
func HistorySearchCommandToProvider(command *apipb.HistorySearchCommand) *providerv1.HistorySearchCommand {
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

// HistoryWindowToAPI 投影 provider history window 结果。
func HistoryWindowToAPI(endpointID string, window *providerv1.HistoryWindowResult) *apipb.HistoryWindowResult {
	if window == nil {
		return nil
	}
	result := &apipb.HistoryWindowResult{
		Terminal:          &apipb.TerminalRef{EndpointId: endpointID, TerminalId: window.GetTerminal().GetTerminalId()},
		Token:             window.GetToken(),
		Operation:         apipb.HistoryWindowOperation(window.GetOperation()),
		Size:              SizeToAPI(window.GetSize()),
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
		projection.Row = ScreenRowToAPI(row.GetRow())
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

// ScreenRowToAPI 投影 provider screen row。
func ScreenRowToAPI(row *providerv1.ScreenRow) *apipb.ScreenRow {
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

// HistoryCopyResultToAPI 投影 provider history copy 结果。
func HistoryCopyResultToAPI(result *providerv1.HistoryCopyResult) *apipb.HistoryCopyResult {
	if result == nil {
		return nil
	}
	projection := &apipb.HistoryCopyResult{Text: result.GetText(), Done: result.GetDone()}
	if result.GetNext() != nil {
		projection.Next = &apipb.HistoryTextPosition{LineId: result.GetNext().GetLineId(), Col: result.GetNext().GetCol()}
	}
	return projection
}

// HistorySearchResultToAPI 投影 provider history search 结果。
func HistorySearchResultToAPI(endpointID string, result *providerv1.HistorySearchResult) *apipb.HistorySearchResult {
	if result == nil {
		return nil
	}
	projection := &apipb.HistorySearchResult{
		Found:    result.GetFound(),
		Wrapped:  result.GetWrapped(),
		Window:   HistoryWindowToAPI(endpointID, result.GetWindow()),
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

// HistoryBacklogToAPI 投影 provider history backlog 状态。
func HistoryBacklogToAPI(endpointID string, status *providerv1.HistoryBacklogStatusResult) *apipb.HistoryBacklogStatusResult {
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

// NativeScreenToAPI 投影 provider live screen 结果。
func NativeScreenToAPI(endpointID string, snapshot *providerv1.NativeScreenResult) *apipb.NativeScreenResult {
	if snapshot == nil {
		return nil
	}
	result := &apipb.NativeScreenResult{
		Terminal:          &apipb.TerminalRef{EndpointId: endpointID, TerminalId: snapshot.GetTerminal().GetTerminalId()},
		LiveRevision:      snapshot.GetLiveRevision(),
		Size:              SizeToAPI(snapshot.GetSize()),
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
			RowIndex: replacement.GetRowIndex(), Row: ScreenRowToAPI(replacement.GetRow()),
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

// TerminalDefaultsToAPI 投影 provider shell/cwd 默认值。
func TerminalDefaultsToAPI(defaults *providerv1.TerminalDefaults) *apipb.TerminalDefaultsResult {
	if defaults == nil {
		return &apipb.TerminalDefaultsResult{}
	}
	return &apipb.TerminalDefaultsResult{Defaults: &apipb.TerminalDefaults{
		DefaultCommand: append([]string(nil), defaults.GetCommand()...),
		DefaultCwd:     defaults.GetCwd(),
		Platform:       defaults.GetPlatform(),
	}}
}

// PathDirectoriesToAPI 投影 provider path completion 窗口。
func PathDirectoriesToAPI(directories *providerv1.PathListDirectoriesResult) *apipb.PathListDirectoriesResult {
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

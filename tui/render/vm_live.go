package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/anytty/anytty/tui/state"
)

func buildLiveContentVM(surface state.TerminalSurfaceStore, session state.TerminalSessionStore) ContentVM {
	return buildLiveContentVMWithSelection(surface, session, state.TerminalViewBinding{}, state.EndpointItem{}, 0)
}

func buildLiveContentVMWithSelection(surface state.TerminalSurfaceStore, session state.TerminalSessionStore, binding state.TerminalViewBinding, endpoint state.EndpointItem, selectedIndex int) ContentVM {
	lines := terminalLiveLineVMs(surface)
	content := ContentVM{
		Kind:   ContentTerminalLive,
		Lines:  lines,
		Status: liveStatus(surface, session),
		Cursor: liveContentCursor(surface, session, lines),
		Meta: ContentMetaVM{
			LiveEndpointID: string(surface.EndpointID),
			LiveTerminalID: surface.TerminalID,
			LiveRevision:   surface.Revision,
		},
	}
	if len(lines) > 0 {
		content.Extent = liveContentExtent(surface, session)
	}
	if binding.AttachPending {
		return liveConnectingContent(surface, session, binding, endpoint, content.Lines)
	}
	if session.LastError != "" {
		content.Error = session.LastError
	} else if surface.Err != "" {
		content.Error = surface.Err
	}
	if liveContentIsDisconnected(surface, session) {
		return liveDisconnectedContent(surface, session, binding, endpoint, content.Lines, selectedIndex)
	}
	if len(content.Lines) == 0 {
		if session.State == state.TerminalLiveExited || surface.State == state.TerminalLiveExited {
			content = liveExitedContent(surface, session, nil, selectedIndex)
		} else if surface.Ready {
			content.Lines = []Line{NewLine("live surface empty")}
			content.Empty = true
			content.Cursor = liveEmptySurfaceCursor(surface, session)
		} else {
			content.Lines = []Line{NewLine("live surface pending")}
			content.Pending = true
			content.Cursor = Cursor{}
		}
	} else if session.State == state.TerminalLiveExited || surface.State == state.TerminalLiveExited {
		// 退出态是 terminal 内容流的尾部；viewport 负责看尾部，避免覆盖最后一屏历史。
		content = liveExitedContent(surface, session, content.Lines, selectedIndex)
	}
	return content
}

func liveContentIsDisconnected(surface state.TerminalSurfaceStore, session state.TerminalSessionStore) bool {
	if session.State == state.TerminalLiveExited || surface.State == state.TerminalLiveExited {
		return false
	}
	kind := state.ClassifyEndpointErrorText(liveDisconnectedReason(surface, session))
	if kind == state.EndpointErrorUnknown || kind == state.EndpointErrorUnavailable {
		return false
	}
	if strings.TrimSpace(session.LastError) != "" && !session.Attached {
		return true
	}
	return strings.TrimSpace(surface.Err) != "" && surface.State == state.TerminalLiveError
}

func liveConnectingContent(surface state.TerminalSurfaceStore, session state.TerminalSessionStore, binding state.TerminalViewBinding, endpoint state.EndpointItem, previous []Line) ContentVM {
	ref := binding.TerminalRef()
	if ref.Empty() {
		ref = session.TerminalRef()
	}
	if ref.Empty() {
		ref = surface.TerminalRef()
	}
	lines := append([]Line(nil), previous...)
	if len(lines) > 0 {
		lines = append(lines, NewLine(""))
	}
	title := "◌ Connecting terminal"
	if len(previous) > 0 || strings.TrimSpace(surface.Err) != "" || strings.TrimSpace(session.LastError) != "" {
		title = "◌ Reconnecting terminal"
	}
	lines = append(lines,
		Line{Cells: []Cell{styledCell(title, StyleWarning)}},
		Line{Cells: []Cell{styledCell("Opening a new endpoint session.", StyleMuted)}},
		Line{Cells: []Cell{styledCell("Input resumes after attach succeeds.", StyleMuted)}},
		NewLine(""),
		Line{Cells: []Cell{styledCell("endpoint  ", StyleMuted), styledCell(endpointDisplayLabel(endpoint, ref.EndpointID), StyleAccent)}},
		Line{Cells: []Cell{styledCell("transport ", StyleMuted), styledCell(endpointTransportLabel(endpoint), StyleForeground)}},
		Line{Cells: []Cell{styledCell("terminal  ", StyleMuted), styledCell(ref.TerminalID, StyleForeground)}},
		NewLine(""),
		Line{Cells: []Cell{styledCell("Please keep this pane open.", StyleMuted)}},
	)
	return ContentVM{Kind: ContentTerminalLive, Lines: lines, Status: "connecting: input paused", Pending: true, Cursor: Cursor{}}
}

func liveDisconnectedContent(surface state.TerminalSurfaceStore, session state.TerminalSessionStore, binding state.TerminalViewBinding, endpoint state.EndpointItem, previous []Line, selectedIndex int) ContentVM {
	if selectedIndex < 0 || selectedIndex >= len(liveDisconnectedActions()) {
		selectedIndex = 0
	}
	ref := session.TerminalRef()
	if ref.Empty() {
		ref = surface.TerminalRef()
	}
	reason := liveDisconnectedReason(surface, session)
	kind := state.ClassifyEndpointErrorText(reason)
	reason = endpointErrorMessageWithoutKind(kind, reason)
	lines := make([]Line, 0, len(previous)+6)
	if len(previous) > 0 {
		lines = append(lines, previous...)
		lines = append(lines, NewLine(""))
	}
	lines = append(lines,
		Line{Cells: []Cell{styledCell("● Connection interrupted", StyleDanger)}},
		Line{Cells: []Cell{styledCell("The last terminal frame is preserved.", StyleMuted)}},
		Line{Cells: []Cell{styledCell("Input is paused.", StyleMuted)}},
		NewLine(""),
		Line{Cells: []Cell{styledCell("endpoint  ", StyleMuted), styledCell(endpointDisplayLabel(endpoint, ref.EndpointID), StyleAccent)}},
		Line{Cells: []Cell{styledCell("transport ", StyleMuted), styledCell(endpointTransportLabel(endpoint), StyleForeground)}},
		Line{Cells: []Cell{styledCell("terminal  ", StyleMuted), styledCell(ref.TerminalID, StyleForeground)}},
	)
	lines = append(lines, Line{Cells: []Cell{styledCell("issue     ", StyleMuted), styledCell(endpointIssueLabel(kind), StyleWarning)}})
	if reason != "" {
		lines = append(lines, Line{Cells: []Cell{styledCell("detail    ", StyleMuted), styledCell(reason, StyleForeground)}})
	}
	lines = append(lines,
		NewLine(""),
		Line{Cells: []Cell{styledCell("next step ", StyleMuted), styledCell(endpointRecoveryHint(kind), StyleForeground)}},
		NewLine(""),
	)
	actionOffset := len(lines)
	actionLines, regions := liveDisconnectedActionLines(selectedIndex)
	lines = append(lines, actionLines...)
	for index := range regions {
		regions[index].Rect.Y += actionOffset
	}
	errorLabel := endpointErrorLabel(kind, reason)
	if errorLabel == "" {
		errorLabel = "endpoint disconnected"
	}
	return ContentVM{
		Kind:       ContentTerminalLive,
		Lines:      lines,
		Status:     "connection interrupted: reconnect or detach",
		Error:      errorLabel,
		Cursor:     Cursor{},
		HitRegions: regions,
	}
}

func endpointIssueLabel(kind state.EndpointErrorKind) string {
	switch state.NormalizeEndpointErrorKind(kind) {
	case state.EndpointErrorAuth:
		return "Authentication failed"
	case state.EndpointErrorHostKey:
		return "Remote host identity changed"
	case state.EndpointErrorRemoteDaemon:
		return "Remote anytty daemon unavailable"
	case state.EndpointErrorTransportClosed:
		return "Transport connection closed"
	case state.EndpointErrorTransportDial:
		return "Endpoint unreachable"
	case state.EndpointErrorProtocol:
		return "Protocol session ended"
	case state.EndpointErrorConfig:
		return "Endpoint configuration invalid"
	case state.EndpointErrorEntitlement:
		return "Cloud entitlement unavailable"
	case state.EndpointErrorRelayPlan:
		return "Relay is not included in the Cloud plan"
	case state.EndpointErrorRelayQuota:
		return "Relay traffic quota exhausted"
	case state.EndpointErrorRelayConcurrency:
		return "Relay connection limit reached"
	case state.EndpointErrorSubscription:
		return "Cloud subscription inactive"
	case state.EndpointErrorRelayRegion:
		return "Relay unavailable in this region"
	default:
		return "Endpoint unavailable"
	}
}

func endpointRecoveryHint(kind state.EndpointErrorKind) string {
	switch state.NormalizeEndpointErrorKind(kind) {
	case state.EndpointErrorAuth:
		return "Check endpoint credentials, then reconnect."
	case state.EndpointErrorHostKey:
		return "Review the remote host identity before reconnecting."
	case state.EndpointErrorRemoteDaemon:
		return "Check that the remote anytty daemon is running, then reconnect."
	case state.EndpointErrorTransportClosed, state.EndpointErrorTransportDial:
		return "Check the network or remote host, then reconnect."
	case state.EndpointErrorProtocol:
		return "Reconnect to open a new protocol session."
	case state.EndpointErrorConfig:
		return "Review the endpoint configuration before reconnecting."
	case state.EndpointErrorEntitlement:
		return "Review the Cloud plan and usage, or use Direct or SSH."
	case state.EndpointErrorRelayPlan:
		return "Use Direct or SSH, or ask the device owner to change the Cloud plan."
	case state.EndpointErrorRelayQuota:
		return "Use P2P, Direct, or SSH, or wait for the next Cloud period."
	case state.EndpointErrorRelayConcurrency:
		return "Close another Relay connection, or use P2P, Direct, or SSH."
	case state.EndpointErrorSubscription:
		return "Ask the device owner to restore the Cloud subscription, or use Direct or SSH."
	case state.EndpointErrorRelayRegion:
		return "Select another Relay region, or use P2P, Direct, or SSH."
	default:
		return "Reconnect to open a new endpoint session."
	}
}

func endpointDisplayLabel(endpoint state.EndpointItem, endpointID state.EndpointID) string {
	if endpoint.ID != "" {
		return endpoint.DisplayLabel() + " (" + string(endpoint.ID) + ")"
	}
	return string(state.NormalizeEndpointID(endpointID))
}

func endpointTransportLabel(endpoint state.EndpointItem) string {
	if endpoint.Transport == "" {
		return "unknown"
	}
	return string(endpoint.Transport)
}

func liveDisconnectedReason(surface state.TerminalSurfaceStore, session state.TerminalSessionStore) string {
	if strings.TrimSpace(session.LastError) != "" {
		return strings.TrimSpace(session.LastError)
	}
	return strings.TrimSpace(surface.Err)
}

func endpointErrorMessageWithoutKind(kind state.EndpointErrorKind, message string) string {
	kind = state.NormalizeEndpointErrorKind(kind)
	message = strings.TrimSpace(message)
	if kind == state.EndpointErrorUnknown || message == "" {
		return message
	}
	prefix := string(kind) + ":"
	if strings.HasPrefix(message, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(message, prefix))
	}
	return message
}

func liveExitedContent(surface state.TerminalSurfaceStore, session state.TerminalSessionStore, previous []Line, selectedIndex int) ContentVM {
	exitLines, regions := liveExitedContentLines(surface, session, selectedIndex)
	if liveLinesContainExitMarker(previous, surface, session) {
		exitLines, regions = liveExitedActionLines(selectedIndex)
	}
	lines := make([]Line, 0, len(previous)+len(exitLines)+1)
	if len(previous) > 0 {
		lines = append(lines, previous...)
		if len(exitLines) > 0 {
			lines = append(lines, NewLine(""))
		}
	}
	actionOffset := len(lines)
	lines = append(lines, exitLines...)
	if actionOffset > 0 {
		for index := range regions {
			regions[index].Rect.Y += actionOffset
		}
	}
	return ContentVM{
		Kind:       ContentExitedPane,
		Lines:      lines,
		Status:     liveStatus(surface, session),
		Empty:      len(previous) == 0,
		Cursor:     Cursor{},
		HitRegions: regions,
	}
}

func liveExitedContentLines(surface state.TerminalSurfaceStore, session state.TerminalSessionStore, selectedIndex int) ([]Line, []HitRegion) {
	lines := []Line{liveExitedLine(surface, session)}
	if exitedAt := liveExitedAt(surface, session); !exitedAt.IsZero() {
		lines = append(lines, NewLine("exited at: "+exitedAt.UTC().Format(time.RFC3339)))
	}
	if command := liveExitCommand(surface, session); command != "" {
		lines = append(lines, NewLine("command: "+command))
	}
	actions := liveExitedActions()
	if selectedIndex < 0 || selectedIndex >= len(actions) {
		selectedIndex = 0
	}
	regions := make([]HitRegion, 0, len(actions))
	for index, action := range actions {
		selected := index == selectedIndex
		text := emptyPaneActionLabel(action.Label, selected)
		line := centeredStyledLine(text, action.Style)
		lines = append(lines, line)
		regions = append(regions, HitRegion{Kind: HitRegionContentAction, Rect: Rect{Y: len(lines) - 1, W: DisplayWidth(text), H: 1}, ActionID: action.ID.String(), Invocation: invocationForProjection(action.ID), TargetMode: HitTargetExplicit})
	}
	return lines, regions
}

func liveExitedActionLines(selectedIndex int) ([]Line, []HitRegion) {
	actions := liveExitedActions()
	if selectedIndex < 0 || selectedIndex >= len(actions) {
		selectedIndex = 0
	}
	lines := make([]Line, 0, len(actions))
	regions := make([]HitRegion, 0, len(actions))
	for index, action := range actions {
		selected := index == selectedIndex
		text := emptyPaneActionLabel(action.Label, selected)
		line := centeredStyledLine(text, action.Style)
		lines = append(lines, line)
		regions = append(regions, HitRegion{Kind: HitRegionContentAction, Rect: Rect{Y: len(lines) - 1, W: DisplayWidth(text), H: 1}, ActionID: action.ID.String(), Invocation: invocationForProjection(action.ID), TargetMode: HitTargetExplicit})
	}
	return lines, regions
}

func liveDisconnectedActionLines(selectedIndex int) ([]Line, []HitRegion) {
	actions := liveDisconnectedActions()
	if selectedIndex < 0 || selectedIndex >= len(actions) {
		selectedIndex = 0
	}
	lines := make([]Line, 0, len(actions))
	regions := make([]HitRegion, 0, len(actions))
	for index, action := range actions {
		selected := index == selectedIndex
		text := emptyPaneActionLabel(action.Label, selected)
		line := centeredStyledLine(text, action.Style)
		lines = append(lines, line)
		regions = append(regions, HitRegion{Kind: HitRegionContentAction, Rect: Rect{Y: len(lines) - 1, W: DisplayWidth(text), H: 1}, ActionID: action.ID.String(), Invocation: invocationForProjection(action.ID), TargetMode: HitTargetExplicit})
	}
	return lines, regions
}

func liveLinesContainExitMarker(lines []Line, surface state.TerminalSurfaceStore, session state.TerminalSessionStore) bool {
	if len(lines) == 0 {
		return false
	}
	prefix := "terminal exited"
	if terminalID := liveTerminalID(surface, session); terminalID != "" {
		prefix += ": " + terminalID
	}
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line.PlainString()), prefix) {
			return true
		}
	}
	return false
}

func liveContentExtent(surface state.TerminalSurfaceStore, session state.TerminalSessionStore) ContentExtent {
	cols, rows := liveStatusSize(surface, session)
	if cols <= 0 || rows <= 0 {
		return ContentExtent{}
	}
	return ContentExtent{Known: true, Cols: cols, Rows: rows}
}

func liveStatus(surface state.TerminalSurfaceStore, session state.TerminalSessionStore) string {
	status := "live"
	if surface.TerminalID != "" {
		status = "live: " + surface.TerminalID
	} else if session.TerminalID != "" {
		status = "live: " + session.TerminalID
	}
	if session.Attached || surface.State == state.TerminalLiveAttached {
		cols, rows := liveStatusSize(surface, session)
		if cols > 0 && rows > 0 {
			status += fmt.Sprintf(" attached %dx%d", cols, rows)
		} else {
			status += " attached"
		}
	}
	if session.State == state.TerminalLiveExited || surface.State == state.TerminalLiveExited {
		status = "exited: " + liveTerminalID(surface, session)
		if code, ok := liveExitCode(surface, session); ok {
			status += fmt.Sprintf(" code:%d", code)
		}
		if reason := liveExitReason(surface, session); reason != "" {
			status += " " + reason
		}
	}
	if session.LastError != "" {
		status = "error: " + session.LastError
	} else if surface.Err != "" {
		status = "error: " + surface.Err
	}
	return status
}

func liveExitedLine(surface state.TerminalSurfaceStore, session state.TerminalSessionStore) Line {
	text := "terminal exited"
	if terminalID := liveTerminalID(surface, session); terminalID != "" {
		text += ": " + terminalID
	}
	if code, ok := liveExitCode(surface, session); ok {
		text += fmt.Sprintf(" code:%d", code)
	}
	if reason := liveExitReason(surface, session); reason != "" {
		text += " " + reason
	}
	return Line{Cells: []Cell{styledCell(text, StyleWarning)}}
}

func liveTerminalID(surface state.TerminalSurfaceStore, session state.TerminalSessionStore) string {
	if surface.TerminalID != "" {
		return surface.TerminalID
	}
	return session.TerminalID
}

func liveExitCode(surface state.TerminalSurfaceStore, session state.TerminalSessionStore) (int, bool) {
	switch {
	case session.State == state.TerminalLiveExited:
		return session.ExitCode, true
	case surface.State == state.TerminalLiveExited:
		return surface.ExitCode, true
	default:
		return 0, false
	}
}

func liveExitReason(surface state.TerminalSurfaceStore, session state.TerminalSessionStore) string {
	if session.ExitReason != "" {
		return session.ExitReason
	}
	return surface.ExitReason
}

func liveExitedAt(surface state.TerminalSurfaceStore, session state.TerminalSessionStore) time.Time {
	if !session.ExitedAt.IsZero() {
		return session.ExitedAt
	}
	return surface.ExitedAt
}

func liveExitCommand(surface state.TerminalSurfaceStore, session state.TerminalSessionStore) string {
	if len(session.Command) > 0 {
		return strings.Join(session.Command, " ")
	}
	return strings.Join(surface.Command, " ")
}

func liveStatusSize(surface state.TerminalSurfaceStore, session state.TerminalSessionStore) (int, int) {
	if surface.Cols > 0 && surface.Rows > 0 {
		return surface.Cols, surface.Rows
	}
	return session.Cols, session.Rows
}

func liveContentCursor(surface state.TerminalSurfaceStore, session state.TerminalSessionStore, lines []Line) Cursor {
	if session.LastError != "" || surface.Err != "" {
		return Cursor{}
	}
	if surface.Cursor.Visible {
		return Cursor{
			Visible: true,
			Row:     maxInt(0, surface.Cursor.Row),
			Col:     maxInt(0, surface.Cursor.Col),
			Shape:   liveCursorShape(surface.Cursor.Shape),
		}
	}
	if surface.Cursor.Shape != "" {
		return Cursor{
			Anchor: true,
			Row:    maxInt(0, surface.Cursor.Row),
			Col:    maxInt(0, surface.Cursor.Col),
			Shape:  liveCursorShape(surface.Cursor.Shape),
		}
	}
	// 中文说明：live terminal 的光标位置只能来自 core/protocol 的 surface cursor。
	// restart 会保留旧 live tail 但清掉旧进程 cursor；不能按文本尾部臆造新光标。
	return Cursor{}
}

func liveEmptySurfaceCursor(surface state.TerminalSurfaceStore, session state.TerminalSessionStore) Cursor {
	if session.LastError != "" || surface.Err != "" || !session.Attached {
		return Cursor{}
	}
	if surface.Cursor.Visible {
		return Cursor{Visible: true, Row: maxInt(0, surface.Cursor.Row), Col: maxInt(0, surface.Cursor.Col), Shape: liveCursorShape(surface.Cursor.Shape)}
	}
	return Cursor{Visible: true, Row: 0, Col: 0, Shape: CursorShapeBlock}
}

func liveCursorShape(shape string) CursorShape {
	switch shape {
	case string(CursorShapeBar):
		return CursorShapeBar
	default:
		return CursorShapeBlock
	}
}

func terminalLiveLineVMs(surface state.TerminalSurfaceStore) []Line {
	var out []Line
	if len(surface.Screen) > 0 {
		out = terminalLiveLineVMsFromCells(surface.Screen)
	} else if len(surface.Lines) > 0 {
		out = make([]Line, len(surface.Lines))
		for i, line := range surface.Lines {
			out[i] = terminalLiveLineFromANSI(line)
		}
	} else {
		return nil
	}
	// Full snapshots can briefly retain backing rows/cells from the previous PTY size.
	// The declared live extent is authoritative; overflow chrome must describe that
	// extent versus the pane, not stale payload beyond it.
	if surface.Rows > 0 && len(out) > surface.Rows {
		out = out[:surface.Rows]
	}
	if surface.Cols > 0 {
		for index, line := range out {
			if line.Width() > surface.Cols {
				out[index] = contentViewportLineWindow(line, 0, surface.Cols)
			}
		}
	}
	return out
}

func terminalLiveLineVMsFromCells(rows [][]state.LiveCell) []Line {
	if len(rows) == 0 {
		return nil
	}
	out := make([]Line, len(rows))
	for rowIndex, row := range rows {
		out[rowIndex] = terminalLiveLineFromCells(row)
	}
	return out
}

func terminalLiveLineFromCells(row []state.LiveCell) Line {
	if len(row) == 0 {
		return Line{}
	}
	cells := make([]Cell, 0, len(row))
	for _, liveCell := range row {
		text := SafeLine(liveCell.Text)
		width := liveCell.Width
		if width <= 0 {
			width = DisplayWidth(text)
		}
		if width <= 0 {
			continue
		}
		cells = append(cells, Cell{
			Text:            text,
			Width:           width,
			ANSIStyle:       terminalLiveANSIStyle(liveCell),
			LinkURL:         liveCell.LinkURL,
			LinkParams:      liveCell.LinkParams,
			TerminalContent: true,
			Safe:            true,
		})
	}
	return Line{Cells: cells}
}

// TerminalLiveRowANSI 把一行 reducer-owned live cells 投影成固定宽度 ANSI 行。
// runtime 的 live patch 只在完整帧提供了稳定内容区时调用它。
func TerminalLiveRowANSI(row []state.LiveCell, width int, theme Theme) string {
	line := contentViewportFitLine(terminalLiveLineFromCells(row), width)
	return ensureANSIReset(line.ANSIString(theme.WithFallback()))
}

func terminalLiveANSIStyle(cell state.LiveCell) ANSICellStyle {
	return ANSICellStyle{
		FG:            cell.FG,
		BG:            cell.BG,
		Bold:          cell.Bold,
		Italic:        cell.Italic,
		Underline:     cell.Underline,
		Blink:         cell.Blink,
		Reverse:       cell.Reverse,
		Strikethrough: cell.Strikethrough,
	}
}

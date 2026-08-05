package app

import (
	"strconv"
	"strings"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
	xansi "github.com/charmbracelet/x/ansi"
)

type mouseDragState struct {
	Active             bool
	Kind               mouseDragKind
	PaneID             string
	FloatingID         string
	Direction          state.PaneResizeDirection
	SplitPath          string
	ResizeBeforePaneID string
	ResizeAfterPaneID  string
	ResizeBeforeCells  int
	ResizeAfterCells   int
	ResizeGroup        []state.PaneResizeGroupItem
	StartCol           int
	StartRow           int
	LastDelta          int
	LastCol            int
	LastRow            int
}

type mouseActionClickState struct {
	Kind                render.HitRegionKind
	InvocationSignature string
	PaneID              string
	Floating            bool
	Row                 int
	Col                 int
}

type mouseHitResolution struct {
	Foreground    render.HitRegion
	HasForeground bool
	HistoryRow    render.HitRegion
	HasHistoryRow bool
	FocusOwner    render.HitRegion
	HasFocusOwner bool
}

type mouseDragKind string

const (
	mouseDragPaneResize       mouseDragKind = "pane-resize"
	mouseDragFloatingMove     mouseDragKind = "floating-move"
	mouseDragFloatingResize   mouseDragKind = "floating-resize"
	mouseDragClipboardDivider mouseDragKind = "clipboard-divider"
)

func (runtime *AppRuntime) dispatchMouseHitRegion(msg Msg) Msg {
	inputMsg, ok := msg.(InputMsg)
	if !ok {
		return msg
	}
	runtime.clearStaleMouseDrag(inputMsg.Event)
	if inputMsg.Event.Kind != input.EventKindMouse {
		return msg
	}
	if dragMsg, handled := runtime.dispatchMouseDrag(inputMsg.Event); handled {
		return dragMsg
	}
	resolution := resolveMouseHitRegions(runtime.lastHitRegions, inputMsg.Event)
	if inputMsg.Event.Mouse != input.MouseLeft {
		if msg, ok := runtime.overlayMouseSelectMsg(inputMsg.Event, resolution); ok {
			return msg
		}
		// 中文说明：前台程序启用 mouse tracking 后，raw 鼠标事件归子进程所有；
		// 只有未被 terminal 接管的滚轮才作为 AnyTTY infinite history 入口。
		if inputMsg.Event.RawSeq != "" {
			if passthroughMsg, ok := runtime.mousePassthroughInputMsg(inputMsg, resolution); ok {
				return passthroughMsg
			}
		}
		if msg, ok := runtime.copyModeMouseWheelEnterMsg(inputMsg.Event, resolution); ok {
			return msg
		}
		if msg, ok := runtime.copyModeMouseWheelMsg(inputMsg.Event, resolution); ok {
			return msg
		}
		if inputMsg.Event.RawSeq != "" {
			if runtime.mouseWheelCanRouteToCopyMode(inputMsg.Event, resolution) {
				return msg
			}
			return NoopMsg{}
		}
		if runtime.mouseEventHitsUI(inputMsg.Event, resolution) {
			return NoopMsg{}
		}
		return msg
	}
	if !resolution.HasForeground {
		return msg
	}
	region := resolution.Foreground
	if region.Kind == render.HitRegionPaneResize {
		if drag, ok := paneResizeDragState(region, inputMsg.Event); ok {
			runtime.mouseDrag = drag
			return NoopMsg{}
		}
	}
	if drag, ok := runtime.floatingDragState(region, inputMsg.Event); ok {
		runtime.mouseDrag = drag
		return ShellFloatingCommandMsg{Command: state.FloatingCommand{
			Action:   state.FloatingCommandFocusRaise,
			TargetID: drag.FloatingID,
			Source:   state.PaneCommandSourceMouse,
		}}
	}
	if drag, ok := clipboardHistoryDividerDragState(region, inputMsg.Event); ok {
		runtime.mouseDrag = drag
		return NoopMsg{}
	}
	if region.Kind == render.HitRegionPaneAction && region.Invocation.ID == "panel.take_owner" {
		if !runtime.consumeTakeResizeOwnerDoubleClick(region, inputMsg.Event) {
			return ShellArmOwnerConfirmMsg{ViewID: terminalViewIDForOwnerRegion(runtime.state, region)}
		}
		return shortcutSurfaceActionMessage(region)
	}
	if region.Kind == render.HitRegionPaneAction && region.Invocation.ID == "panel.size_lock" {
		return shortcutSurfaceActionMessage(region)
	}
	if (region.Kind == render.HitRegionPaneAction || region.Kind == render.HitRegionPaneChrome) && region.Invocation.ID != "" {
		return shortcutSurfaceActionMessage(region)
	}
	switch region.Kind {
	case render.HitRegionToastClose:
		return ShellCloseCurrentToastMsg{}
	case render.HitRegionToast:
		return NoopMsg{}
	case render.HitRegionOverlay:
		return ShellCloseOverlayMsg{}
	}
	if command, ok := PaneCommandFromHitRegion(region); ok && command.Action != state.PaneCommandFocus {
		runtime.fillMousePaneCommandDefaults(&command)
		if command.Action == state.PaneCommandSplit {
			return ShellWorkbenchCommandMsg{Command: state.WorkbenchCommand{
				Action: state.WorkbenchCommandPaneSplit,
				Pane:   command,
				Source: state.PaneCommandSourceMouse,
			}}
		}
		return ShellPaneCommandMsg{Command: command}
	}
	if passthroughMsg, ok := runtime.mousePassthroughInputMsg(inputMsg, resolution); ok {
		return passthroughMsg
	}
	if activationMsg, ok := runtime.terminalInputActivationMsg(region); ok {
		return activationMsg
	}
	// 中文说明：history row 是内容前景命中区；点击非 active pane 的历史文本时必须先切焦点，
	// 不能直接进入 copy selection，否则文本区域会吞掉 panel focus。
	if resolution.HasHistoryRow {
		if focusMsg, shouldFocus := runtime.historyRowFocusMsg(resolution); shouldFocus {
			return focusMsg
		}
		if !runtime.copyModeMouseSelectAllowed(resolution.HistoryRow) {
			return NoopMsg{}
		}
		col := historyHitRegionDisplayColumn(inputMsg.Event, resolution.HistoryRow)
		return CopyModeMouseSelectMsg{Position: state.CopyPosition{Row: resolution.HistoryRow.Row, Col: col}, PaneID: resolution.HistoryRow.PaneID, ViewID: runtime.copyHistoryViewIDForRegion(resolution.HistoryRow)}
	}
	if command, ok := PaneCommandFromHitRegion(region); ok {
		runtime.fillMousePaneCommandDefaults(&command)
		return ShellPaneCommandMsg{Command: command}
	}
	switch region.Kind {
	case render.HitRegionContentAction:
		if region.Invocation.ID != "" {
			return shortcutSurfaceActionMessage(region)
		}
		return NoopMsg{}
	default:
		return msg
	}
}

func shortcutSurfaceActionMessage(region render.HitRegion) ShellShortcutActionMsg {
	return ShellShortcutActionMsg{
		Invocation: region.Invocation,
		Surface: &ShortcutSurfaceContext{
			ExplicitTarget: region.TargetMode == render.HitTargetExplicit,
			PaneID:         region.PaneID,
			Floating:       region.Floating,
			Row:            region.Row,
			HasRow:         region.HasRow,
		},
	}
}

func (runtime *AppRuntime) terminalInputActivationMsg(region render.HitRegion) (Msg, bool) {
	shell := runtime.state.Shell.EnsureDefaults()
	switch region.Kind {
	case render.HitRegionPaneContent:
		if region.PaneID == "" {
			return nil, false
		}
		if shell.ActivePaneID == region.PaneID && shell.ActiveFloatingID() == "" && shell.InteractionMode == state.InteractionModeNormal {
			return nil, false
		}
		return ShellActivateTerminalInputMsg{PaneID: region.PaneID}, true
	case render.HitRegionContentAction:
		if region.Invocation.ID != actiondomain.ActionFloatingRaise || !region.Floating {
			return nil, false
		}
		floatingID, ok := runtime.floatingIDForPaneID(region.PaneID)
		if !ok {
			return nil, false
		}
		if shell.ActiveFloatingID() == floatingID && shell.InteractionMode == state.InteractionModeNormal {
			return nil, false
		}
		return ShellActivateTerminalInputMsg{PaneID: region.PaneID, FloatingID: floatingID}, true
	default:
		return nil, false
	}
}

func (runtime *AppRuntime) mouseWheelCanRouteToCopyMode(event input.InputEvent, resolution mouseHitResolution) bool {
	if event.Kind != input.EventKindMouse {
		return false
	}
	switch event.Mouse {
	case input.MouseWheelUp:
	case input.MouseWheelDown:
	default:
		return false
	}
	// 中文说明：带 RawSeq 的普通鼠标事件默认会被 terminal mouse tracking 吞掉；上滑需要保留
	// 进入 copy/history 的入口，已进入 copy/history 后下滑也必须继续交给 copy reducer。
	if !resolution.HasForeground {
		return event.Mouse == input.MouseWheelUp || copyModeInputContext(runtime.state.CopyMode)
	}
	switch resolution.Foreground.Kind {
	case render.HitRegionPaneContent, render.HitRegionHistoryRow:
		if event.Mouse == input.MouseWheelUp {
			return true
		}
		_, copyMode := runtime.state.CopyHistorySessionForView(runtime.copyHistoryViewIDForRegion(resolution.Foreground))
		return copyModeInputContext(copyMode)
	default:
		return false
	}
}

func (runtime *AppRuntime) overlayMouseSelectMsg(event input.InputEvent, resolution mouseHitResolution) (Msg, bool) {
	if !runtime.state.Shell.EnsureDefaults().Overlay.Open || !resolution.HasForeground {
		return nil, false
	}
	if resolution.Foreground.Kind != render.HitRegionOverlay && resolution.Foreground.Kind != render.HitRegionContentAction {
		return nil, false
	}
	switch event.Mouse {
	case input.MouseWheelUp:
		return ShellOverlayMouseSelectMsg{Delta: -1}, true
	case input.MouseWheelDown:
		return ShellOverlayMouseSelectMsg{Delta: 1}, true
	default:
		return nil, false
	}
}

func (runtime *AppRuntime) clearStaleMouseDrag(event input.InputEvent) {
	if !runtime.mouseDrag.Active {
		return
	}
	if event.Kind != input.EventKindMouse || event.Mouse == input.MouseLeft {
		// macOS 终端的 Option 原生选择可能吞掉 release，新输入必须能恢复 UI 鼠标状态。
		runtime.mouseDrag = mouseDragState{}
	}
}

func terminalViewIDForOwnerRegion(root state.Root, region render.HitRegion) string {
	if region.Floating {
		if floatingID, ok := root.Shell.EnsureDefaults().FloatingIDForPaneID(region.PaneID); ok {
			if binding, ok := root.TerminalViews.FloatingBinding(floatingID); ok {
				return binding.ViewID
			}
		}
		return ""
	}
	if binding, ok := root.TerminalViews.PaneBinding(region.PaneID); ok {
		return binding.ViewID
	}
	if binding, ok := root.TerminalViews.FloatingBinding(region.PaneID); ok {
		return binding.ViewID
	}
	return ""
}

func (runtime *AppRuntime) consumeTakeResizeOwnerDoubleClick(region render.HitRegion, event input.InputEvent) bool {
	current := mouseActionClickState{Kind: region.Kind, InvocationSignature: region.Invocation.Signature(), PaneID: region.PaneID, Floating: region.Floating, Row: event.Row, Col: event.Col}
	if runtime.lastMouseAction == current {
		runtime.lastMouseAction = mouseActionClickState{}
		return true
	}
	runtime.lastMouseAction = current
	return false
}

func historyHitRegionDisplayColumn(event input.InputEvent, region render.HitRegion) int {
	col := event.Col - 1
	if event.Col <= 0 {
		col = event.Col
	}
	col -= region.Rect.X
	if col < 0 {
		return 0
	}
	return col
}

func (runtime *AppRuntime) historyRowFocusMsg(resolution mouseHitResolution) (Msg, bool) {
	if !resolution.HasHistoryRow {
		return nil, false
	}
	region := resolution.HistoryRow
	if msg, ok := runtime.focusMsgForOwnerRegion(region); ok {
		return msg, true
	}
	if region.PaneID != "" {
		return nil, false
	}
	if !resolution.HasFocusOwner {
		return nil, false
	}
	return runtime.focusMsgForOwnerRegion(resolution.FocusOwner)
}

func (runtime *AppRuntime) focusMsgForOwnerRegion(region render.HitRegion) (Msg, bool) {
	if region.Floating {
		floatingID, ok := runtime.floatingIDForPaneID(region.PaneID)
		if !ok {
			return nil, false
		}
		shell := runtime.state.Shell.EnsureDefaults()
		if shell.ActiveFloatingID() == floatingID && shell.InteractionMode == state.InteractionModeNormal {
			return nil, false
		}
		return ShellActivateTerminalInputMsg{PaneID: region.PaneID, FloatingID: floatingID}, true
	}
	return runtime.focusMsgForOwner(region.PaneID)
}

func (runtime *AppRuntime) focusMsgForOwner(ownerID string) (Msg, bool) {
	if ownerID == "" {
		return nil, false
	}
	shell := runtime.state.Shell.EnsureDefaults()
	if _, ok := shell.PaneByID(ownerID); ok {
		if shell.ActivePaneID == ownerID && shell.ActiveFloatingID() == "" && shell.InteractionMode == state.InteractionModeNormal {
			return nil, false
		}
		return ShellActivateTerminalInputMsg{PaneID: ownerID}, true
	}
	for _, floating := range shell.ActiveFloatings() {
		if floating.ID != ownerID {
			continue
		}
		if shell.ActiveFloatingID() == ownerID && shell.InteractionMode == state.InteractionModeNormal {
			return nil, false
		}
		return ShellActivateTerminalInputMsg{PaneID: floating.Pane.ID, FloatingID: ownerID}, true
	}
	return nil, false
}

func (runtime *AppRuntime) copyModeMouseSelectAllowed(region render.HitRegion) bool {
	viewID := runtime.copyHistoryViewIDForRegion(region)
	_, copyMode := runtime.state.CopyHistorySessionForView(viewID)
	if !copyMode.Active {
		return false
	}
	if region.PaneID == "" {
		return true
	}
	if copyMode.PaneID == region.PaneID {
		return true
	}
	if region.Floating {
		if floatingID, ok := runtime.floatingIDForPaneID(region.PaneID); ok && copyMode.ViewID == runtime.state.TerminalViews.FloatingViewID(floatingID) {
			return true
		}
	}
	if copyMode.ViewID == runtime.state.TerminalViews.FloatingViewID(region.PaneID) {
		return true
	}
	shell := runtime.state.Shell.EnsureDefaults()
	if shell.ActivePaneID == region.PaneID && shell.ActiveFloatingID() == "" && copyMode.PaneID == "" && copyMode.ViewID == "" {
		return true
	}
	return false
}

func (runtime *AppRuntime) copyModeMouseWheelEnterMsg(event input.InputEvent, resolution mouseHitResolution) (Msg, bool) {
	if event.Kind != input.EventKindMouse || event.Mouse != input.MouseWheelUp {
		return nil, false
	}
	region, ok := copyModeWheelTargetRegion(resolution)
	if !ok {
		return nil, false
	}
	if _, copyMode := runtime.state.CopyHistorySessionForView(runtime.copyHistoryViewIDForRegion(region)); copyModeInputContext(copyMode) {
		return nil, false
	}
	binding, ok := runtime.terminalViewBindingForMouseRegion(region)
	if !ok || binding.TerminalID == "" {
		return nil, false
	}
	rect := region.Rect
	if rect.W <= 0 || rect.H <= 0 {
		if fallback, ok := terminalViewContentRect(runtime.state, runtimeViewportRect(runtime.state.Viewport), binding); ok {
			rect = fallback
		}
	}
	if rect.W <= 0 {
		return nil, false
	}
	// 中文说明：滚轮上滑是 copy/history 入口，必须绑定鼠标命中的 TerminalView；
	// 不能先丢给 active pane，否则非 active sibling 要等下一次事件才进入 copy。
	return CopyModeEnterViewMsg{Binding: binding, Cols: rect.W, Rows: rect.H, InitialScrollDelta: -copyModeLineScrollRows()}, true
}

func (runtime *AppRuntime) copyModeMouseWheelMsg(event input.InputEvent, resolution mouseHitResolution) (Msg, bool) {
	if event.Kind != input.EventKindMouse || !mouseEventIsWheel(event) {
		return nil, false
	}
	region, ok := copyModeWheelTargetRegion(resolution)
	if !ok {
		return nil, false
	}
	viewID := runtime.copyHistoryViewIDForRegion(region)
	if viewID == "" {
		return nil, false
	}
	_, copyMode := runtime.state.CopyHistorySessionForView(viewID)
	if !copyModeInputContext(copyMode) {
		return nil, false
	}
	// 中文说明：鼠标滚轮命中的是具体 TerminalView，不能回退到 active pane；
	// floating/pane 的 copy history 会话必须各自滚动、各自退出。
	return CopyModeWheelMsg{Event: event, ViewID: viewID}, true
}

func (runtime *AppRuntime) copyHistoryViewIDForRegion(region render.HitRegion) string {
	if binding, ok := runtime.terminalViewBindingForMouseRegion(region); ok {
		return binding.ViewID
	}
	if region.Floating {
		if floatingID, ok := runtime.floatingIDForPaneID(region.PaneID); ok {
			return runtime.state.TerminalViews.FloatingViewID(floatingID)
		}
	}
	if region.PaneID != "" {
		return runtime.state.TerminalViews.PaneViewID(region.PaneID)
	}
	return ""
}

func copyModeWheelTargetRegion(resolution mouseHitResolution) (render.HitRegion, bool) {
	if resolution.HasHistoryRow {
		return resolution.HistoryRow, true
	}
	if !resolution.HasForeground {
		return render.HitRegion{}, false
	}
	switch resolution.Foreground.Kind {
	case render.HitRegionPaneContent, render.HitRegionHistoryRow, render.HitRegionContentAction:
		return resolution.Foreground, true
	default:
		return render.HitRegion{}, false
	}
}

func (runtime *AppRuntime) terminalViewBindingForMouseRegion(region render.HitRegion) (state.TerminalViewBinding, bool) {
	if region.Floating {
		if floatingID, ok := runtime.floatingIDForPaneID(region.PaneID); ok {
			return runtime.state.TerminalViews.FloatingBinding(floatingID)
		}
	}
	if binding, ok := runtime.state.TerminalViews.PaneBinding(region.PaneID); ok {
		return binding, true
	}
	if binding, ok := runtime.state.TerminalViews.FloatingBinding(region.PaneID); ok {
		return binding, true
	}
	return state.TerminalViewBinding{}, false
}

func runtimeViewportRect(viewport state.ViewportStore) render.Rect {
	if !viewport.Valid {
		return render.Rect{}
	}
	return render.Rect{W: viewport.Cols, H: viewport.Rows}
}

func (runtime *AppRuntime) mouseEventCanPassthrough(event input.InputEvent, resolution mouseHitResolution) bool {
	if event.RawSeq == "" || !resolution.HasFocusOwner || !resolution.HasForeground {
		return false
	}
	shell := runtime.state.Shell.EnsureDefaults()
	if shell.Overlay.Open || copyModeInputContext(runtime.state.CopyMode) {
		return false
	}
	if shell.InteractionMode != state.InteractionModeNormal {
		return false
	}
	if !mouseForegroundAllowsTerminalPassthrough(resolution.Foreground) {
		return false
	}
	return runtime.mouseTrackingEnabledForRegion(resolution.FocusOwner)
}

func (runtime *AppRuntime) mousePassthroughInputMsg(msg InputMsg, resolution mouseHitResolution) (InputMsg, bool) {
	if !runtime.mouseEventCanPassthrough(msg.Event, resolution) {
		return InputMsg{}, false
	}
	binding, ok := runtime.terminalViewBindingForMouseRegion(resolution.FocusOwner)
	if !ok || binding.ViewID == "" {
		return InputMsg{}, false
	}
	modes := runtime.state.Surface.SurfaceForTerminalRef(binding.TerminalRef()).Modes
	msg.Event = encodeMouseEventForTerminal(msg.Event, resolution.Foreground.Rect, modes)
	msg.TerminalMousePassthrough = true
	msg.TerminalMouseTargetViewID = binding.ViewID
	return msg, true
}

func encodeMouseEventForTerminal(event input.InputEvent, rect render.Rect, modes state.LiveTerminalModes) input.InputEvent {
	if event.Kind != input.EventKindMouse || event.RawSeq == "" || rect.W <= 0 || rect.H <= 0 {
		return event
	}
	localCol := event.Col - rect.X
	localRow := event.Row - rect.Y
	if localCol < 1 {
		localCol = 1
	}
	if localRow < 1 {
		localRow = 1
	}
	if localCol > rect.W {
		localCol = rect.W
	}
	if localRow > rect.H {
		localRow = rect.H
	}
	if modes.MouseSGR {
		if raw, ok := rewriteSGRMouseRawSeq(event.RawSeq, localCol, localRow); ok {
			// 中文说明：子进程 mouse tracking 的 truth source 是它自己的 PTY 网格，
			// AnyTTY chrome/header/footer 只属于外层 TUI，透传前必须改成内容区 local 坐标。
			event.RawSeq = raw
			event.Col = localCol
			event.Row = localRow
		}
		return event
	}
	if raw, ok := encodeLegacyMouseRawSeq(event, localCol, localRow); ok {
		// 中文说明：子进程 mouse tracking 的 truth source 是它自己的 PTY 网格，
		// legacy mouse 程序没有请求 SGR 时必须收到 X10 编码，不能把宿主 SGR 原样转发。
		event.RawSeq = raw
		event.Col = localCol
		event.Row = localRow
	}
	return event
}

func rewriteSGRMouseRawSeq(raw string, col int, row int) (string, bool) {
	if col <= 0 || row <= 0 || !strings.HasPrefix(raw, "\x1b[<") || len(raw) < len("\x1b[<0;1;1M") {
		return "", false
	}
	final := raw[len(raw)-1]
	if final != 'M' && final != 'm' {
		return "", false
	}
	parts := strings.Split(raw[3:len(raw)-1], ";")
	if len(parts) != 3 {
		return "", false
	}
	if _, err := strconv.Atoi(parts[0]); err != nil {
		return "", false
	}
	if _, err := strconv.Atoi(parts[1]); err != nil {
		return "", false
	}
	if _, err := strconv.Atoi(parts[2]); err != nil {
		return "", false
	}
	return "\x1b[<" + parts[0] + ";" + strconv.Itoa(col) + ";" + strconv.Itoa(row) + string(final), true
}

func encodeLegacyMouseRawSeq(event input.InputEvent, col int, row int) (string, bool) {
	button, motion, ok := legacyMouseButton(event.Mouse)
	if !ok || col <= 0 || row <= 0 {
		return "", false
	}
	if col > 223 {
		col = 223
	}
	if row > 223 {
		row = 223
	}
	code := xansi.EncodeMouseButton(button, motion, event.Shift, event.Alt, event.Ctrl)
	if code == 0xff {
		return "", false
	}
	return xansi.MouseX10(code, col-1, row-1), true
}

func legacyMouseButton(button input.MouseButton) (xansi.MouseButton, bool, bool) {
	switch button {
	case input.MouseWheelUp:
		return xansi.MouseWheelUp, false, true
	case input.MouseWheelDown:
		return xansi.MouseWheelDown, false, true
	case input.MouseLeft:
		return xansi.MouseLeft, false, true
	case input.MouseMiddle:
		return xansi.MouseMiddle, false, true
	case input.MouseRight:
		return xansi.MouseRight, false, true
	case input.MouseLeftDrag:
		return xansi.MouseLeft, true, true
	case input.MouseMiddleDrag:
		return xansi.MouseMiddle, true, true
	case input.MouseRightDrag:
		return xansi.MouseRight, true, true
	case input.MouseMove:
		return xansi.MouseNone, true, true
	case input.MouseLeftUp, input.MouseMiddleUp, input.MouseRightUp, input.MouseRelease:
		return xansi.MouseNone, false, true
	default:
		return xansi.MouseNone, false, false
	}
}

func (runtime *AppRuntime) mouseEventHitsUI(event input.InputEvent, resolution mouseHitResolution) bool {
	if !resolution.HasForeground {
		return false
	}
	if event.Kind == input.EventKindMouse && mouseEventIsWheel(event) {
		return false
	}
	region := resolution.Foreground
	switch region.Kind {
	case render.HitRegionPaneContent, render.HitRegionHistoryRow:
		return false
	case render.HitRegionContentAction:
		return region.Invocation.ID != actiondomain.ActionFloatingRaise
	default:
		return true
	}
}

func mouseEventIsWheel(event input.InputEvent) bool {
	return event.Mouse == input.MouseWheelUp || event.Mouse == input.MouseWheelDown
}

func mouseForegroundAllowsTerminalPassthrough(region render.HitRegion) bool {
	switch region.Kind {
	case render.HitRegionPaneContent:
		return true
	case render.HitRegionContentAction:
		return region.Invocation.ID == actiondomain.ActionFloatingRaise
	default:
		return false
	}
}

func (runtime *AppRuntime) mouseTrackingEnabledForRegion(region render.HitRegion) bool {
	binding, ok := runtime.terminalViewBindingForMouseRegion(region)
	if !ok || binding.TerminalID == "" {
		return false
	}
	surface := runtime.state.Surface.SurfaceForTerminalRef(binding.TerminalRef())
	return surface.Modes.MousePassthroughEnabled()
}

func (runtime *AppRuntime) fillMousePaneCommandDefaults(command *state.PaneCommand) {
	if command == nil || command.Action != state.PaneCommandSplit || command.NewPane.ID != "" {
		return
	}
	command.NewPane = state.PaneState{ID: nextKeyboardPaneID(runtime.state.Shell), Title: "pane", Kind: state.PaneEmpty}
}

func (runtime *AppRuntime) dispatchMouseDrag(event input.InputEvent) (Msg, bool) {
	switch event.Mouse {
	case input.MouseLeftUp:
		if runtime.mouseDrag.Active {
			runtime.mouseDrag = mouseDragState{}
			return NoopMsg{}, true
		}
		return nil, false
	case input.MouseLeftDrag:
		if !runtime.mouseDrag.Active {
			return nil, false
		}
		switch runtime.mouseDrag.Kind {
		case mouseDragPaneResize:
			delta := mouseDragResizeDelta(runtime.mouseDrag, event)
			step := delta - runtime.mouseDrag.LastDelta
			if step == 0 {
				return NoopMsg{}, true
			}
			runtime.mouseDrag.LastDelta = delta
			runtime.mouseDrag.LastCol = event.Col
			runtime.mouseDrag.LastRow = event.Row
			return ShellPaneCommandMsg{Command: state.PaneCommand{
				Action:           state.PaneCommandResize,
				Target:           state.PaneCommandTarget{PaneID: runtime.mouseDrag.PaneID},
				ResizeDirection:  runtime.mouseDrag.Direction,
				ResizeSplitPath:  runtime.mouseDrag.SplitPath,
				ResizeGroupCells: mouseDragResizeGroupCells(runtime.mouseDrag, delta),
				Delta:            step,
				Source:           state.PaneCommandSourceMouse,
			}}, true
		case mouseDragFloatingMove:
			deltaX := event.Col - runtime.mouseDrag.LastCol
			deltaY := event.Row - runtime.mouseDrag.LastRow
			if deltaX == 0 && deltaY == 0 {
				return NoopMsg{}, true
			}
			runtime.mouseDrag.LastCol = event.Col
			runtime.mouseDrag.LastRow = event.Row
			return ShellFloatingCommandMsg{Command: state.FloatingCommand{
				Action:   state.FloatingCommandMove,
				TargetID: runtime.mouseDrag.FloatingID,
				DeltaX:   deltaX,
				DeltaY:   deltaY,
				Source:   state.PaneCommandSourceMouse,
			}}, true
		case mouseDragFloatingResize:
			deltaW := event.Col - runtime.mouseDrag.LastCol
			deltaH := event.Row - runtime.mouseDrag.LastRow
			if deltaW == 0 && deltaH == 0 {
				return NoopMsg{}, true
			}
			runtime.mouseDrag.LastCol = event.Col
			runtime.mouseDrag.LastRow = event.Row
			return ShellFloatingCommandMsg{Command: state.FloatingCommand{
				Action:   state.FloatingCommandResize,
				TargetID: runtime.mouseDrag.FloatingID,
				DeltaW:   deltaW,
				DeltaH:   deltaH,
				Source:   state.PaneCommandSourceMouse,
			}}, true
		case mouseDragClipboardDivider:
			delta := event.Col - runtime.mouseDrag.LastCol
			if delta == 0 {
				return NoopMsg{}, true
			}
			runtime.mouseDrag.LastCol = event.Col
			runtime.mouseDrag.LastRow = event.Row
			return ShellMoveClipboardHistoryDividerMsg{Delta: delta}, true
		default:
			return NoopMsg{}, true
		}
	default:
		return nil, false
	}
}

func clipboardHistoryDividerDragState(region render.HitRegion, event input.InputEvent) (mouseDragState, bool) {
	if region.Kind != render.HitRegionContentAction || region.Invocation.ID != actiondomain.ActionClipboardHistoryDividerDrag {
		return mouseDragState{}, false
	}
	return mouseDragState{
		Active:  true,
		Kind:    mouseDragClipboardDivider,
		LastCol: event.Col,
		LastRow: event.Row,
	}, true
}

func paneResizeDragState(region render.HitRegion, event input.InputEvent) (mouseDragState, bool) {
	if region.Kind != render.HitRegionPaneResize || region.Invocation.ID != actiondomain.ActionPanelResizeDrag {
		return mouseDragState{}, false
	}
	direction, ok := paneResizeDirectionFromHitRegion(region)
	if !ok || region.PaneID == "" {
		return mouseDragState{}, false
	}
	return mouseDragState{
		Active:             true,
		Kind:               mouseDragPaneResize,
		PaneID:             region.PaneID,
		Direction:          direction,
		SplitPath:          region.SplitPath,
		ResizeBeforePaneID: region.ResizeBeforePaneID,
		ResizeAfterPaneID:  region.ResizeAfterPaneID,
		ResizeBeforeCells:  region.ResizeBeforeCells,
		ResizeAfterCells:   region.ResizeAfterCells,
		ResizeGroup:        paneResizeGroupFromHitRegion(region),
		StartCol:           event.Col,
		StartRow:           event.Row,
		LastCol:            event.Col,
		LastRow:            event.Row,
	}, true
}

func paneResizeGroupFromHitRegion(region render.HitRegion) []state.PaneResizeGroupItem {
	if len(region.ResizeGroup) < 3 {
		return nil
	}
	out := make([]state.PaneResizeGroupItem, 0, len(region.ResizeGroup))
	for _, item := range region.ResizeGroup {
		out = append(out, state.PaneResizeGroupItem{PaneID: item.PaneID, Cells: item.Cells, DeltaSign: item.DeltaSign})
	}
	return out
}

func mouseDragResizeGroupCells(drag mouseDragState, delta int) []state.PaneResizeGroupItem {
	if len(drag.ResizeGroup) == 0 || drag.ResizeBeforePaneID == "" || drag.ResizeAfterPaneID == "" {
		return nil
	}
	beforeCells := drag.ResizeBeforeCells + delta
	afterCells := drag.ResizeAfterCells - delta
	if beforeCells <= 0 || afterCells <= 0 {
		return nil
	}
	out := make([]state.PaneResizeGroupItem, len(drag.ResizeGroup))
	for i, item := range drag.ResizeGroup {
		out[i] = item
		if item.DeltaSign != 0 {
			out[i].Cells = item.Cells + delta*item.DeltaSign
			if out[i].Cells <= 0 {
				return nil
			}
			continue
		}
		switch item.PaneID {
		case drag.ResizeBeforePaneID:
			out[i].Cells = beforeCells
		case drag.ResizeAfterPaneID:
			out[i].Cells = afterCells
		}
	}
	return out
}

func (runtime *AppRuntime) floatingDragState(region render.HitRegion, event input.InputEvent) (mouseDragState, bool) {
	if region.Kind != render.HitRegionContentAction || region.PaneID == "" || !region.Floating {
		return mouseDragState{}, false
	}
	var kind mouseDragKind
	switch region.Invocation.ID {
	case actiondomain.ActionFloatingMoveDrag:
		kind = mouseDragFloatingMove
	case actiondomain.ActionFloatingResizeDrag:
		kind = mouseDragFloatingResize
	default:
		return mouseDragState{}, false
	}
	floatingID, ok := runtime.floatingIDForPaneID(region.PaneID)
	if !ok {
		return mouseDragState{}, false
	}
	return mouseDragState{
		Active:     true,
		Kind:       kind,
		PaneID:     region.PaneID,
		FloatingID: floatingID,
		LastCol:    event.Col,
		LastRow:    event.Row,
	}, true
}

func (runtime *AppRuntime) floatingIDForPaneID(paneID string) (string, bool) {
	if paneID == "" {
		return "", false
	}
	return runtime.state.Shell.EnsureDefaults().FloatingIDForPaneID(paneID)
}

func paneResizeDirectionFromHitRegion(region render.HitRegion) (state.PaneResizeDirection, bool) {
	switch region.Direction {
	case string(state.PaneResizeLeft):
		return state.PaneResizeLeft, true
	case string(state.PaneResizeRight), "":
		return state.PaneResizeRight, true
	case string(state.PaneResizeUp):
		return state.PaneResizeUp, true
	case string(state.PaneResizeDown):
		return state.PaneResizeDown, true
	default:
		return "", false
	}
}

func mouseDragResizeDelta(drag mouseDragState, event input.InputEvent) int {
	switch drag.Direction {
	case state.PaneResizeLeft, state.PaneResizeRight:
		return event.Col - drag.StartCol
	case state.PaneResizeUp, state.PaneResizeDown:
		return event.Row - drag.StartRow
	default:
		return 0
	}
}

func resolveMouseHitRegions(regions []render.HitRegion, event input.InputEvent) mouseHitResolution {
	col, row := mouseEventPoint(event)
	resolution := mouseHitResolution{}
	for _, region := range regions {
		if !pointInRect(col, row, region.Rect) {
			continue
		}
		if !resolution.HasForeground {
			resolution.Foreground = region
			resolution.HasForeground = true
			if region.Kind == render.HitRegionHistoryRow {
				resolution.HistoryRow = region
				resolution.HasHistoryRow = true
			}
		}
		if !resolution.HasFocusOwner && mouseFocusOwnerRegion(region) {
			resolution.FocusOwner = region
			resolution.HasFocusOwner = true
		}
	}
	return resolution
}

func mouseFocusOwnerRegion(region render.HitRegion) bool {
	switch region.Kind {
	case render.HitRegionPaneContent:
		return region.PaneID != ""
	case render.HitRegionContentAction:
		return region.PaneID != "" && region.Invocation.ID == actiondomain.ActionFloatingRaise
	default:
		return false
	}
}

func mouseEventPoint(event input.InputEvent) (int, int) {
	col := event.Col - 1
	row := event.Row - 1
	if event.Col <= 0 {
		col = event.Col
	}
	if event.Row <= 0 {
		row = event.Row
	}
	return col, row
}

func pointInRect(col int, row int, rect render.Rect) bool {
	return col >= rect.X && col < rect.X+rect.W && row >= rect.Y && row < rect.Y+rect.H
}

func cloneRenderHitRegions(regions []render.HitRegion) []render.HitRegion {
	if len(regions) == 0 {
		return nil
	}
	cloned := make([]render.HitRegion, len(regions))
	copy(cloned, regions)
	for i := range cloned {
		if len(cloned[i].ResizeGroup) > 0 {
			cloned[i].ResizeGroup = append([]render.ResizeGroupItem(nil), cloned[i].ResizeGroup...)
		}
	}
	return cloned
}

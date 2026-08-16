package app

import (
	"context"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/state"
)

// reduceCanonicalSurfaceAction 执行带点击目标上下文的 canonical action。
// action identity 来自 tui/action，PaneID/Floating/Row 只描述命中的 surface；这里不读取 render projection，
// 也不在缺少目标时猜测其他 pane、terminal 或 endpoint。
func reduceCanonicalSurfaceAction(root state.Root, msg ShellShortcutActionMsg) (state.Root, []Effect, bool) {
	if msg.Surface == nil {
		return root, nil, false
	}
	target := *msg.Surface
	if surfaceActionRequiresRow(msg.Invocation.ID) {
		if !target.HasRow || !surfaceActionRowValid(root, msg.Invocation.ID, target.Row) {
			return root, nil, true
		}
	}
	if surfaceActionRequiresPaneTarget(msg.Invocation.ID) {
		if target.PaneID == "" {
			return root, nil, true
		}
		if target.Floating && floatingTargetIDForSurface(root, target.PaneID, true) == "" {
			return root, nil, true
		}
	}
	if target.Floating {
		if next, effects, handled := reduceFloatingPanelSurfaceAction(root, msg, target); handled {
			return next, effects, true
		}
	}
	switch msg.Invocation.ID {
	case actiondomain.ActionPanelFocus:
		if target.PaneID == "" {
			return root, nil, true
		}
		next, effects := reducePaneCommand(root, state.PaneCommand{Action: state.PaneCommandFocus, Target: state.PaneCommandTarget{PaneID: target.PaneID}, Source: state.PaneCommandSourceMouse})
		return next, effects, true
	case "panel.close":
		if target.PaneID == "" {
			return root, nil, true
		}
		next, effects := reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandPaneClose, Target: state.PaneCommandTarget{PaneID: target.PaneID}, Source: shortcutSurfaceCommandSource(msg)})
		return next, effects, true
	case "panel.detach":
		next, effects := reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandPaneDetach, Target: state.PaneCommandTarget{PaneID: target.PaneID}, Source: shortcutSurfaceCommandSource(msg)})
		return next, effects, true
	case "panel.reconnect":
		ref := terminalRefForSurface(root, target.PaneID, false)
		if ref.Empty() {
			next, effects := shortcutUnavailable(root, "pane.reconnect", "terminal unavailable")
			return next, effects, true
		}
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolReconnectRequestMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, TargetPaneID: target.PaneID, LocalError: true}
		}}}, true
	case "panel.restart":
		ref := terminalRefForSurface(root, target.PaneID, false)
		if ref.Empty() {
			next, effects := shortcutUnavailable(root, "terminal.restart", "no active terminal")
			return next, effects, true
		}
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolRestartRequestMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID}
		}}}, true
	case "panel.kill", "panel.kill_and_close":
		ref := terminalRefForSurface(root, target.PaneID, false)
		if ref.Empty() {
			next, effects := shortcutUnavailable(root, "pane.kill", "terminal unavailable")
			return next, effects, true
		}
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolKillRequestMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, PaneID: target.PaneID, CloseOnSuccess: msg.Invocation.ID == "panel.kill_and_close"}
		}}}, true
	case "panel.split_down", "panel.split_right":
		if target.PaneID == "" {
			return root, nil, true
		}
		direction := state.SplitDirectionHorizontal
		if msg.Invocation.ID == "panel.split_right" {
			direction = state.SplitDirectionVertical
		}
		pane := state.PaneState{ID: nextKeyboardPaneID(root.Shell), Title: "pane", Kind: state.PaneEmpty}
		next, effects := reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandPaneSplit, Pane: state.PaneCommand{Action: state.PaneCommandSplit, Target: state.PaneCommandTarget{PaneID: target.PaneID}, SplitDirection: direction, NewPane: pane, Source: state.PaneCommandSourceMouse}, Source: state.PaneCommandSourceMouse})
		return next, effects, true
	case "panel.toggle_zoom":
		if target.PaneID == "" {
			return root, nil, true
		}
		next, effects := reducePaneCommand(root, state.PaneCommand{Action: state.PaneCommandToggleZoom, Target: state.PaneCommandTarget{PaneID: target.PaneID}, Source: state.PaneCommandSourceMouse})
		return next, effects, true
	case "panel.take_owner":
		if target.PaneID == "" && !target.Floating {
			return root, nil, true
		}
		if target.Floating {
			next, effects := requestFloatingResizeOwnerWithConfirm(root, floatingTargetIDForSurface(root, target.PaneID, true))
			return next, effects, true
		}
		root.Shell = root.Shell.ClearOwnerConfirm(0)
		next, effects := requestPaneResizeOwner(root, target.PaneID)
		return next, effects, true
	case "panel.size_lock":
		if target.PaneID == "" && !target.Floating {
			return root, nil, true
		}
		root = focusCanonicalSurfaceTarget(root, msg)
		return root, []Effect{handledEffect{}, terminalSizeLockToggleEffect()}, true
	case actiondomain.ActionTabSelect:
		next, effects := reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandTabSwitch, TargetID: target.PaneID, Source: state.PaneCommandSourceMouse})
		return next, effects, true
	case "tab.close":
		next, effects := reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandTabClose, TargetID: target.PaneID, Source: state.PaneCommandSourceMouse})
		return next, effects, true
	case "tab.create":
		shell := root.Shell.EnsureDefaults()
		next, effects := reduceWorkbenchCommandWithOptions(root, state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: nextTabName(shell), Source: state.PaneCommandSourceMouse}, workbenchCommandOptions{OpenPickerAfterOK: true})
		return next, effects, true
	case actiondomain.ActionTerminalPoolSelect:
		items := state.TerminalPoolPageItems(root)
		root.Shell = root.Shell.SetTerminalPoolSelectedIndex(target.Row, len(items))
		return root.Advance(), []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg { return TerminalPoolPreviewRefreshMsg{} }}}, true
	case actiondomain.ActionWorkbenchTreeSelect:
		items := state.WorkbenchTreeItems(root)
		root.Shell = root.Shell.SetWorkbenchTreeSelectedIndex(target.Row, len(items))
		return root.Advance(), []Effect{handledEffect{}}, true
	case actiondomain.ActionClipboardHistorySelect:
		items := state.ClipboardHistoryItems(root)
		root.Shell = root.Shell.SetClipboardHistorySelectedIndex(target.Row, len(items))
		return root.Advance(), []Effect{handledEffect{}}, true
	case actiondomain.ActionTerminalPickerNew:
		root = focusCanonicalSurfaceTarget(root, msg)
		poolTarget := terminalPoolTargetForOverlay(root)
		endpointID := terminalCreateEndpointIDFromPickerSelection(root, target.Row)
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg {
			return ShellOpenPromptMsg{Prompt: createTerminalPromptForTargetEndpoint(root, poolTarget, endpointID)}
		}}}, true
	case actiondomain.ActionEmptyAttach, actiondomain.ActionExitedReconnect:
		root = focusCanonicalSurfaceTarget(root, msg)
		root.Shell = root.Shell.OpenTerminalPicker()
		return root.Advance(), []Effect{handledEffect{}, terminalPickerListRequestEffect()}, true
	case actiondomain.ActionEmptyManager:
		root = focusCanonicalSurfaceTarget(root, msg)
		root.Shell = root.Shell.OpenTerminalPool()
		return root.Advance(), []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg { return TerminalPoolListRequestMsg{} }}}, true
	case actiondomain.ActionEmptyCreate:
		root = focusCanonicalSurfaceTarget(root, msg)
		poolTarget := terminalPoolTargetForSurface(root, target.PaneID, target.Floating)
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg {
			return ShellOpenPromptMsg{Prompt: createTerminalPromptForTarget(root, poolTarget)}
		}}}, true
	case actiondomain.ActionEmptyClose, actiondomain.ActionExitedClose:
		root = focusCanonicalSurfaceTarget(root, msg)
		return reduceCanonicalSurfaceClose(root, msg)
	case actiondomain.ActionExitedRestart:
		root = focusCanonicalSurfaceTarget(root, msg)
		ref := terminalRefForSurface(root, target.PaneID, target.Floating)
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolRestartIfExitedRequestMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID}
		}}}, true
	case actiondomain.ActionDisconnectedReconnect:
		root = focusCanonicalSurfaceTarget(root, msg)
		ref := terminalRefForSurface(root, target.PaneID, target.Floating)
		if ref.Empty() {
			next, effects := shortcutUnavailable(root, "pane.reconnect", "terminal unavailable")
			return next, effects, true
		}
		poolTarget := terminalPoolTargetForSurface(root, target.PaneID, target.Floating)
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolReconnectRequestMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, TargetPaneID: poolTarget.PaneID, TargetFloatingID: poolTarget.FloatingID, LocalError: true}
		}}}, true
	case actiondomain.ActionDisconnectedDisconnect:
		root = focusCanonicalSurfaceTarget(root, msg)
		if target.Floating {
			next, effects := reduceFloatingCommand(root, state.FloatingCommand{Action: state.FloatingCommandClose, TargetID: floatingTargetIDForSurface(root, target.PaneID, true), Source: state.PaneCommandSourceMouse})
			return next, effects, true
		}
		next, effects := reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandPaneDetach, Target: state.PaneCommandTarget{PaneID: target.PaneID}, Source: state.PaneCommandSourceMouse})
		return next, effects, true
	case actiondomain.ActionFloatingRaise:
		next, effects := reduceFloatingCommand(root, state.FloatingCommand{Action: state.FloatingCommandFocusRaise, TargetID: floatingTargetIDForSurface(root, target.PaneID, target.Floating), Source: state.PaneCommandSourceMouse})
		return next, effects, true
	case actiondomain.ActionFloatingResize:
		if target.PaneID == "" {
			return root, nil, true
		}
		next, effects := reduceFloatingCommand(root, state.FloatingCommand{Action: state.FloatingCommandResize, TargetID: floatingTargetIDForSurface(root, target.PaneID, target.Floating), DeltaW: 2, DeltaH: 1, Source: state.PaneCommandSourceMouse})
		return next, effects, true
	case "floating.center", "floating.collapse", "floating.close":
		if target.PaneID == "" {
			return root, nil, true
		}
		action := state.FloatingCommandCenter
		if msg.Invocation.ID == "floating.collapse" {
			action = state.FloatingCommandToggleCollapse
		} else if msg.Invocation.ID == "floating.close" {
			action = state.FloatingCommandClose
		}
		next, effects := reduceFloatingCommand(root, state.FloatingCommand{Action: action, TargetID: floatingTargetIDForSurface(root, target.PaneID, target.Floating), Source: state.PaneCommandSourceMouse})
		return next, effects, true
	default:
		return root, nil, false
	}
}

func reduceActiveSurfaceShortcut(root state.Root, invocation actiondomain.Invocation) (state.Root, []Effect, bool) {
	if !activeSurfaceShortcutAction(invocation.ID) {
		return root, nil, false
	}
	target, ok := root.Shell.ActiveSurfaceTarget()
	if !ok || !target.Floating {
		return root, nil, false
	}
	return reduceCanonicalSurfaceAction(root, ShellShortcutActionMsg{
		Invocation: invocation,
		Surface: &ShortcutSurfaceContext{
			PaneID:   target.PaneID,
			Floating: true,
			Row:      -1,
		},
	})
}

func activeSurfaceShortcutAction(id actiondomain.ID) bool {
	switch id {
	case "panel.close", "panel.detach", "panel.reconnect", "panel.restart", "panel.take_owner", "panel.size_lock",
		"panel.split_right", "panel.split_down", "panel.kill", "panel.kill_and_close", "panel.toggle_zoom",
		"panel.balance", "panel.presentation_card", "panel.presentation_split_line", "panel.focus_next", "panel.focus_prev",
		"resize.left", "resize.right", "resize.up", "resize.down", "resize.left_large", "resize.right_large", "resize.up_large", "resize.down_large",
		"resize.pan_left", "resize.pan_right", "resize.pan_up", "resize.pan_down",
		"resize.align_left", "resize.align_right", "resize.align_top", "resize.align_bottom",
		"resize.center", "resize.center_x", "resize.center_y", "resize.layout_toggle", "resize.layout_reset":
		return true
	default:
		return false
	}
}

func reduceFloatingPanelSurfaceAction(root state.Root, msg ShellShortcutActionMsg, target ShortcutSurfaceContext) (state.Root, []Effect, bool) {
	if !activeSurfaceShortcutAction(msg.Invocation.ID) {
		return root, nil, false
	}
	floatingID := floatingTargetIDForSurface(root, target.PaneID, true)
	if floatingID == "" {
		return root, nil, true
	}
	switch msg.Invocation.ID {
	case "panel.close":
		next, effects := reduceFloatingCommand(root, state.FloatingCommand{Action: state.FloatingCommandClose, TargetID: floatingID, Source: shortcutSurfaceCommandSource(msg)})
		return next, effects, true
	case "panel.detach":
		next, effects := reduceFloatingDetach(root, floatingID)
		return next, effects, true
	case "panel.reconnect":
		ref := terminalRefForSurface(root, target.PaneID, true)
		if ref.Empty() {
			next, effects := shortcutUnavailable(root, "pane.reconnect", "terminal unavailable")
			return next, effects, true
		}
		poolTarget := terminalPoolTargetForSurface(root, target.PaneID, true)
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolReconnectRequestMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, TargetPaneID: poolTarget.PaneID, TargetFloatingID: poolTarget.FloatingID, LocalError: true}
		}}}, true
	case "panel.restart":
		ref := terminalRefForSurface(root, target.PaneID, true)
		if ref.Empty() {
			next, effects := shortcutUnavailable(root, "terminal.restart", "no active terminal")
			return next, effects, true
		}
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolRestartRequestMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID}
		}}}, true
	case "panel.take_owner":
		if msg.Surface != nil && msg.Surface.ExplicitTarget {
			next, effects := requestFloatingResizeOwnerWithConfirm(root, floatingID)
			return next, effects, true
		}
		next, effects := requestFloatingResizeOwner(root, floatingID)
		return next, append([]Effect{handledEffect{}}, effects...), true
	case "panel.size_lock":
		root = focusCanonicalSurfaceTarget(root, msg)
		return root, []Effect{handledEffect{}, terminalSizeLockToggleEffect()}, true
	case "panel.kill", "panel.kill_and_close":
		ref := terminalRefForSurface(root, target.PaneID, true)
		if ref.Empty() {
			next, effects := shortcutUnavailable(root, "pane.kill", "terminal unavailable")
			return next, effects, true
		}
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolKillRequestMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, FloatingID: floatingID, CloseOnSuccess: msg.Invocation.ID == "panel.kill_and_close"}
		}}}, true
	default:
		if command, ok := floatingPositionCommandForResizeAction(root, msg.Invocation.ID, floatingID); ok {
			next, effects := reduceFloatingCommand(root, command)
			return next, append([]Effect{handledEffect{}}, effects...), true
		}
		next, effects := shortcutUnavailable(root, msg.Invocation.ID.String(), "not available for active floating")
		return next, effects, true
	}
}

func floatingPositionCommandForResizeAction(root state.Root, id actiondomain.ID, floatingID string) (state.FloatingCommand, bool) {
	command := state.FloatingCommand{
		TargetID: floatingID,
		Source:   state.PaneCommandSourceKeyboard,
		BoundsW:  root.Viewport.Cols,
		BoundsH:  root.Viewport.Rows,
	}
	switch id {
	case "resize.left", "resize.pan_left":
		command.Action, command.DeltaX = state.FloatingCommandMove, -2
	case "resize.right", "resize.pan_right":
		command.Action, command.DeltaX = state.FloatingCommandMove, 2
	case "resize.up", "resize.pan_up":
		command.Action, command.DeltaY = state.FloatingCommandMove, -1
	case "resize.down", "resize.pan_down":
		command.Action, command.DeltaY = state.FloatingCommandMove, 1
	case "resize.left_large":
		command.Action, command.DeltaX = state.FloatingCommandMove, -6
	case "resize.right_large":
		command.Action, command.DeltaX = state.FloatingCommandMove, 6
	case "resize.up_large":
		command.Action, command.DeltaY = state.FloatingCommandMove, -3
	case "resize.down_large":
		command.Action, command.DeltaY = state.FloatingCommandMove, 3
	case "resize.align_left":
		command.Action, command.PositionX = state.FloatingCommandPosition, state.TerminalViewAlignStart
	case "resize.align_right":
		command.Action, command.PositionX = state.FloatingCommandPosition, state.TerminalViewAlignEnd
	case "resize.align_top":
		command.Action, command.PositionY = state.FloatingCommandPosition, state.TerminalViewAlignStart
	case "resize.align_bottom":
		command.Action, command.PositionY = state.FloatingCommandPosition, state.TerminalViewAlignEnd
	case "resize.center", "resize.layout_reset":
		command.Action = state.FloatingCommandCenter
	case "resize.center_x":
		command.Action, command.PositionX = state.FloatingCommandPosition, state.TerminalViewAlignCenter
	case "resize.center_y":
		command.Action, command.PositionY = state.FloatingCommandPosition, state.TerminalViewAlignCenter
	default:
		return state.FloatingCommand{}, false
	}
	return command, true
}

func reduceFloatingDetach(root state.Root, floatingID string) (state.Root, []Effect) {
	binding, ok := root.TerminalViews.FloatingBinding(floatingID)
	if !ok || binding.TerminalID == "" {
		return shortcutUnavailable(root, "pane.detach", "terminal unavailable")
	}
	effects := workbenchPersistEffects(string(state.WorkbenchCommandPaneDetach))
	effects = append(effects, copyHistoryCleanupEffectsForView(root, binding.ViewID)...)
	if request, ok := terminalDetachRequestFromBinding(binding); ok {
		effects = append(effects, terminalDetachEffect(request))
	}
	root = root.WithoutCopyHistorySession(binding.ViewID)
	root.TerminalViews = root.TerminalViews.DetachFloating(floatingID)
	root.Shell = root.Shell.DetachFloatingTerminal(floatingID).AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: string(state.WorkbenchCommandPaneDetach), Body: floatingID})
	return root.Advance(), append([]Effect{handledEffect{}}, effects...)
}

func shortcutSurfaceCommandSource(msg ShellShortcutActionMsg) state.PaneCommandSource {
	if msg.Surface != nil && msg.Surface.ExplicitTarget {
		return state.PaneCommandSourceMouse
	}
	return state.PaneCommandSourceKeyboard
}

func surfaceActionRowValid(root state.Root, id actiondomain.ID, row int) bool {
	switch id {
	case actiondomain.ActionTerminalPoolSelect:
		return row >= 0 && row < len(state.TerminalPoolPageItems(root))
	case actiondomain.ActionWorkbenchTreeSelect:
		return row >= 0 && row < len(state.WorkbenchTreeItems(root))
	case actiondomain.ActionClipboardHistorySelect:
		return row >= 0 && row < len(state.ClipboardHistoryItems(root))
	case actiondomain.ActionTerminalPickerNew:
		items := state.TerminalPickerItems(root)
		return row >= 0 && row < len(items) && items[row].CreateNew && items[row].EndpointID != ""
	case "terminal_picker.attach":
		items := state.TerminalPickerItems(root)
		return row >= 0 && row < len(items) && !items[row].CreateNew && items[row].TerminalID != ""
	case "floating_overview.open":
		return row >= 0 && row < len(state.FloatingOverviewItems(root))
	case "workbench_tree.open":
		items := state.WorkbenchTreeItems(root)
		// Workbench detail/preview producer 用显式 -1 表示当前 selected；其他负数或越界值不允许 clamp。
		return len(items) > 0 && (row == -1 || row >= 0 && row < len(items))
	default:
		return false
	}
}

func surfaceActionRequiresRow(id actiondomain.ID) bool {
	switch id {
	case actiondomain.ActionTerminalPoolSelect,
		actiondomain.ActionWorkbenchTreeSelect,
		actiondomain.ActionClipboardHistorySelect,
		actiondomain.ActionTerminalPickerNew,
		"terminal_picker.attach", "workbench_tree.open", "floating_overview.open":
		return true
	default:
		return false
	}
}

func surfaceActionRequiresPaneTarget(id actiondomain.ID) bool {
	switch id {
	case actiondomain.ActionPanelFocus,
		"panel.close", "panel.detach", "panel.reconnect", "panel.restart", "panel.split_down", "panel.split_right", "panel.kill", "panel.kill_and_close", "panel.toggle_zoom", "panel.take_owner", "panel.size_lock",
		"panel.balance", "panel.presentation_card", "panel.presentation_split_line", "panel.focus_next", "panel.focus_prev",
		"resize.left", "resize.right", "resize.up", "resize.down", "resize.left_large", "resize.right_large", "resize.up_large", "resize.down_large",
		"resize.pan_left", "resize.pan_right", "resize.pan_up", "resize.pan_down",
		"resize.align_left", "resize.align_right", "resize.align_top", "resize.align_bottom",
		"resize.center", "resize.center_x", "resize.center_y", "resize.layout_toggle", "resize.layout_reset",
		actiondomain.ActionTabSelect, "tab.close",
		actiondomain.ActionEmptyClose,
		actiondomain.ActionExitedRestart, actiondomain.ActionExitedReconnect, actiondomain.ActionExitedClose,
		actiondomain.ActionDisconnectedReconnect, actiondomain.ActionDisconnectedDisconnect,
		actiondomain.ActionFloatingRaise, actiondomain.ActionFloatingResize,
		"floating.center", "floating.collapse", "floating.close":
		return true
	default:
		return false
	}
}

func focusCanonicalSurfaceTarget(root state.Root, msg ShellShortcutActionMsg) state.Root {
	if msg.Surface == nil {
		return root
	}
	target := *msg.Surface
	if target.Floating {
		if id := floatingTargetIDForSurface(root, target.PaneID, true); id != "" {
			root.Shell, _ = root.Shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandFocusRaise, TargetID: id, Source: state.PaneCommandSourceMouse})
		}
	} else if target.PaneID != "" {
		root.Shell = root.Shell.FocusPane(state.PaneCommandTarget{PaneID: target.PaneID})
	}
	return root
}

func reduceCanonicalSurfaceClose(root state.Root, msg ShellShortcutActionMsg) (state.Root, []Effect, bool) {
	if msg.Surface == nil {
		return root, nil, false
	}
	target := *msg.Surface
	if target.PaneID == "" && !target.Floating {
		return root, nil, true
	}
	if target.Floating {
		next, effects := reduceFloatingCommand(root, state.FloatingCommand{Action: state.FloatingCommandClose, TargetID: floatingTargetIDForSurface(root, target.PaneID, true), Source: state.PaneCommandSourceMouse})
		return next, effects, true
	}
	next, effects := reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandPaneClose, Target: state.PaneCommandTarget{PaneID: target.PaneID}, Source: state.PaneCommandSourceMouse})
	return next, effects, true
}

func floatingTargetIDForSurface(root state.Root, paneID string, floating bool) string {
	shell := root.Shell.EnsureDefaults()
	if paneID == "" {
		return shell.ActiveFloatingID()
	}
	if floating {
		if floatingID, ok := shell.FloatingIDForPaneID(paneID); ok {
			return floatingID
		}
	}
	for _, candidate := range shell.ActiveFloatings() {
		if candidate.ID == paneID {
			return candidate.ID
		}
	}
	return ""
}

func terminalRefForSurface(root state.Root, paneID string, floating bool) state.TerminalRef {
	if floating {
		if binding, ok := root.TerminalViews.FloatingBinding(floatingTargetIDForSurface(root, paneID, true)); ok {
			return binding.TerminalRef()
		}
		return state.TerminalRef{}
	}
	return terminalRefForContentAction(root, paneID)
}

func terminalPoolTargetForSurface(root state.Root, paneID string, floating bool) terminalPoolTarget {
	if floating {
		if id := floatingTargetIDForSurface(root, paneID, true); id != "" {
			if target, ok := terminalPoolTargetForID(root.Shell.EnsureDefaults(), id); ok {
				target.ViewID = root.TerminalViews.FloatingViewID(id)
				return target
			}
		}
	}
	if paneID != "" {
		return terminalPoolTarget{PaneID: paneID, ViewID: root.TerminalViews.PaneViewID(paneID)}
	}
	return terminalPoolTargetForActive(root)
}

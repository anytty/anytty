package app

import (
	"context"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
)

// reduceAppShortcutAction 执行不能降解为通用 input intent 的 canonical keyboard action。
// handler 只按 action.ID 选择业务步骤；选中行来自 reducer state 或显式点击上下文，绝不查询 render projection。
func reduceAppShortcutAction(root state.Root, invocation actiondomain.Invocation, row int) (state.Root, []Effect) {
	switch invocation.ID {
	case "terminal_picker.endpoint_previous":
		root = moveTerminalPickerEndpoint(root, -1)
		return root.Advance(), []Effect{handledEffect{}}
	case "terminal_picker.endpoint_next":
		root = moveTerminalPickerEndpoint(root, 1)
		return root.Advance(), []Effect{handledEffect{}}
	case "terminal_picker.status_previous":
		root = moveTerminalPickerStatus(root, -1)
		return root.Advance(), []Effect{handledEffect{}}
	case "terminal_picker.status_next":
		root = moveTerminalPickerStatus(root, 1)
		return root.Advance(), []Effect{handledEffect{}}
	case "terminal_picker.select_previous":
		root = moveTerminalPickerSelection(root, -1)
		return root.Advance(), []Effect{handledEffect{}}
	case "terminal_picker.select_next":
		root = moveTerminalPickerSelection(root, 1)
		return root.Advance(), []Effect{handledEffect{}}
	case "terminal_picker.tags":
		if terminalPickerTagsOpen(root) {
			root.Shell = root.Shell.CloseTerminalPickerTags()
		} else {
			root.Shell = root.Shell.OpenTerminalPickerTags()
		}
		return root.Advance(), []Effect{handledEffect{}}
	case "terminal_picker.tag_toggle":
		if terminalPickerTagsOpen(root) {
			root = toggleTerminalPickerTag(root, row)
		}
		return root.Advance(), []Effect{handledEffect{}}
	case "terminal_picker.attach":
		items := state.TerminalPickerItems(root)
		if row >= 0 {
			root.Shell = root.Shell.SetTerminalPickerSelectedIndex(row, len(items))
			items = state.TerminalPickerItems(root)
		}
		return reduceTerminalPickerConfirm(root, items)
	case "terminal_picker.split":
		selected, ok := terminalPickerItemAt(state.TerminalPickerItems(root), row)
		if !ok || selected.TerminalID == "" {
			return shortcutUnavailable(root, "picker.split", "no terminal")
		}
		return reduceTerminalPickerSplitAttach(root, selected.EndpointID, selected.TerminalID)
	case "terminal_picker.edit":
		return reduceTerminalPickerEdit(root, row)
	case "terminal_picker.kill", "terminal_picker.delete":
		selected, ok := terminalPickerItemAt(state.TerminalPickerItems(root), row)
		if !ok || selected.TerminalID == "" {
			return shortcutUnavailable(root, invocation.ID.String(), "no terminal")
		}
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg {
			switch invocation.ID {
			case "terminal_picker.kill":
				return TerminalPoolKillRequestMsg{EndpointID: selected.EndpointID, TerminalID: selected.TerminalID}
			default:
				return TerminalPoolRemoveRequestMsg{EndpointID: selected.EndpointID, TerminalID: selected.TerminalID}
			}
		}}}
	case "terminal_picker.close", "terminal_pool.close", "connections.close", "workbench_tree.close", "clipboard_history.close", "floating_overview.close", "help.close":
		root.Shell = root.Shell.CloseOverlay()
		return root.Advance(), []Effect{handledEffect{}}
	case "help.previous", "help.next", "help.page_up", "help.page_down", "help.first", "help.last":
		itemCount := len(input.ShortcutEntriesForHelp(root.Config.Shortcuts, root.HostCapabilities.KeyboardDisambiguation))
		switch invocation.ID {
		case "help.previous":
			root.Shell = root.Shell.MoveHelpSelection(-1, itemCount)
		case "help.next":
			root.Shell = root.Shell.MoveHelpSelection(1, itemCount)
		case "help.page_up":
			root.Shell = root.Shell.MoveHelpSelection(-8, itemCount)
		case "help.page_down":
			root.Shell = root.Shell.MoveHelpSelection(8, itemCount)
		case "help.first":
			root.Shell = root.Shell.SetHelpSelection(0, itemCount)
		case "help.last":
			root.Shell = root.Shell.SetHelpSelection(itemCount-1, itemCount)
		}
		return root.Advance(), []Effect{handledEffect{}}
	case "terminal_pool.attach":
		items := state.TerminalPoolPageItems(root)
		if row >= 0 {
			root.Shell = root.Shell.SetTerminalPoolSelectedIndex(row, len(items))
			items = state.TerminalPoolPageItems(root)
		}
		return reduceTerminalPoolPageAttach(root, items)
	case "connections.edit":
		return reduceConnectionsEdit(root, row)
	case "connections.toggle":
		return reduceConnectionsToggle(root, row)
	case "connections.refresh":
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg { return ConnectionsLoadRequestMsg{} }}}
	case "terminal_pool.attach_tab", "terminal_pool.attach_float", "terminal_pool.restart", "terminal_pool.edit", "terminal_pool.kill", "terminal_pool.delete":
		selected, ok := terminalPoolPageItemForAction(root, row)
		if !ok {
			return shortcutUnavailable(root, invocation.ID.String(), "no terminal")
		}
		switch invocation.ID {
		case "terminal_pool.attach_tab":
			return reduceTerminalPoolAttachTab(root, selected.EndpointID, selected.TerminalID)
		case "terminal_pool.attach_float":
			return reduceTerminalPoolAttachFloating(root, selected.EndpointID, selected.TerminalID)
		case "terminal_pool.edit":
			root.Shell = root.Shell.OpenPrompt(terminalEditPrompt(selected))
			return root.Advance(), []Effect{handledEffect{}}
		default:
			return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg {
				if invocation.ID == "terminal_pool.restart" {
					return TerminalPoolRestartRequestMsg{EndpointID: selected.EndpointID, TerminalID: selected.TerminalID}
				}
				if invocation.ID == "terminal_pool.kill" {
					return TerminalPoolKillRequestMsg{EndpointID: selected.EndpointID, TerminalID: selected.TerminalID}
				}
				return TerminalPoolRemoveRequestMsg{EndpointID: selected.EndpointID, TerminalID: selected.TerminalID}
			}}}
		}
	case "workbench_tree.open":
		items := state.WorkbenchTreeItems(root)
		if row >= 0 {
			root.Shell = root.Shell.SetWorkbenchTreeSelectedIndex(row, len(items))
			items = state.WorkbenchTreeItems(root)
		}
		if selected, ok := workbenchTreeSelectedItem(items); row < 0 && ok && selected.Expandable {
			root.Shell = root.Shell.ToggleWorkbenchTreeItem(selected)
			return root.Advance(), []Effect{handledEffect{}}
		}
		return reduceWorkbenchTreeOpen(root, items)
	case "workbench_tree.new":
		return reduceWorkbenchTreeNew(root, state.WorkbenchTreeItems(root))
	case "workbench_tree.rename":
		return reduceWorkbenchTreeRename(root, state.WorkbenchTreeItems(root))
	case "workbench_tree.delete":
		return reduceWorkbenchTreeDelete(root, state.WorkbenchTreeItems(root))
	case "workbench_tree.detach":
		return reduceWorkbenchTreeDetach(root, state.WorkbenchTreeItems(root))
	case "workbench_tree.zoom":
		return reduceWorkbenchTreeZoom(root, state.WorkbenchTreeItems(root))
	case "clipboard_history.paste":
		return reduceClipboardHistoryPaste(root)
	case "clipboard_history.new":
		return reduceClipboardHistoryNew(root)
	case "clipboard_history.edit":
		return reduceClipboardHistoryEdit(root)
	case "clipboard_history.delete":
		return reduceClipboardHistoryDelete(root)
	case "floating_overview.open":
		items := state.FloatingOverviewItems(root)
		if row >= 0 {
			root.Shell = root.Shell.SetFloatingOverviewSelectedIndex(row, len(items))
			items = state.FloatingOverviewItems(root)
		}
		return reduceFloatingOverviewOpen(root, items)
	case "floating_overview.show_all":
		return reduceFloatingCommand(root, state.FloatingCommand{Action: state.FloatingCommandShowAll, Source: state.PaneCommandSourceKeyboard})
	case "floating_overview.collapse_all":
		return reduceFloatingCommand(root, state.FloatingCommand{Action: state.FloatingCommandCollapseAll, Source: state.PaneCommandSourceKeyboard})
	case "prompt.submit":
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg { return ShellPromptSubmitMsg{} }}}
	case "prompt.cancel":
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg { return ShellPromptCancelMsg{} }}}
	default:
		return shortcutUnavailable(root, invocation.ID.String(), "handler missing")
	}
}

func reduceTerminalPickerEdit(root state.Root, row int) (state.Root, []Effect) {
	selected, ok := terminalPickerItemAt(state.TerminalPickerItems(root), row)
	if !ok || selected.TerminalID == "" {
		return shortcutUnavailable(root, "terminal_picker.edit", "no terminal")
	}
	item, ok := terminalPoolItemRef(root.TerminalPool, state.NewTerminalRef(selected.EndpointID, selected.TerminalID))
	if !ok {
		return shortcutUnavailable(root, "terminal_picker.edit", "terminal metadata unavailable")
	}
	root.Shell = root.Shell.OpenPrompt(terminalPickerEditPrompt(state.TerminalPoolPageItem{
		EndpointID: item.EndpointID, TerminalID: item.TerminalID, Title: item.Title, Tags: item.Tags,
	}))
	return root.Advance(), []Effect{handledEffect{}}
}

func reduceTerminalPickerSplitAttach(root state.Root, endpointID state.EndpointID, terminalID string) (state.Root, []Effect) {
	shell := root.Shell.EnsureDefaults()
	targetPaneID := nextKeyboardPaneID(shell)
	next, effects := reducePaneCommand(root, state.PaneCommand{
		Action: state.PaneCommandSplit, Target: state.PaneCommandTarget{PaneID: shell.ActivePaneID}, SplitDirection: state.SplitDirectionVertical,
		NewPane: state.PaneState{ID: targetPaneID, Title: "pane", Kind: state.PaneEmpty}, Source: state.PaneCommandSourceKeyboard,
	})
	if pane, ok := next.Shell.Pane(state.PaneCommandTarget{PaneID: targetPaneID}); !ok || pane.ID != targetPaneID {
		return next, effects
	}
	return next, append(effects, FuncEffect{Run: func(context.Context) Msg {
		return TerminalPoolAttachRequestMsg{EndpointID: endpointID, TerminalID: terminalID, TargetPaneID: targetPaneID}
	}})
}

func reduceTerminalPoolAttachTab(root state.Root, endpointID state.EndpointID, terminalID string) (state.Root, []Effect) {
	beforeTabID := root.Shell.EnsureDefaults().Workspace.ActiveTabID
	next, effects := reduceWorkbenchCommand(root, state.WorkbenchCommand{
		Action: state.WorkbenchCommandTabCreate, Name: nextTabName(root.Shell.EnsureDefaults()), Source: state.PaneCommandSourcePalette,
	})
	after := next.Shell.EnsureDefaults()
	if after.Workspace.ActiveTabID == beforeTabID || after.ActivePaneID == "" {
		return next, effects
	}
	if _, ok := after.Pane(state.PaneCommandTarget{PaneID: after.ActivePaneID}); !ok {
		return next, effects
	}
	targetPaneID := after.ActivePaneID
	return next, append(effects, FuncEffect{Run: func(context.Context) Msg {
		return TerminalPoolAttachRequestMsg{EndpointID: endpointID, TerminalID: terminalID, TargetPaneID: targetPaneID}
	}})
}

func reduceTerminalPoolAttachFloating(root state.Root, endpointID state.EndpointID, terminalID string) (state.Root, []Effect) {
	floatingID := nextFloatingID(root.Shell)
	next, effects := reduceFloatingCommand(root, state.FloatingCommand{
		Action: state.FloatingCommandCreate, TargetID: floatingID,
		Pane:  state.PaneState{ID: nextFloatingPaneID(root.Shell), Title: "floating", Kind: state.PaneEmpty},
		Title: "floating", Source: state.PaneCommandSourceKeyboard,
	})
	if _, ok := next.Shell.FloatingByID(floatingID); !ok {
		return next, effects
	}
	return next, append(effects, FuncEffect{Run: func(context.Context) Msg {
		return TerminalPoolAttachRequestMsg{EndpointID: endpointID, TerminalID: terminalID, TargetFloatingID: floatingID}
	}})
}

func shortcutUnavailable(root state.Root, title, body string) (state.Root, []Effect) {
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: title, Body: body})
	return root.Advance(), []Effect{handledEffect{}}
}

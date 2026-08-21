package app

import (
	"strconv"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/shortcut"
)

func shortcutIntentOwnedByCopy(intent input.Intent) bool {
	switch intent.Kind {
	case input.IntentEnterCopyMode, input.IntentRequestOlder, input.IntentRequestNewer,
		input.IntentCopyCommand:
		return true
	default:
		return false
	}
}

type actionHandler func(actiondomain.Invocation, input.InputEvent) (input.Intent, bool)

var actionHandlerRegistry = buildActionHandlerRegistry()

// shortcutIntentForInvocation 通过 canonical action handler registry 生成 reducer intent。
// registry 只以 tui/action ID 为键；alias、scene、展示和 source string 均不能参与执行选择。
func shortcutIntentForInvocation(invocation actiondomain.Invocation, event input.InputEvent) (input.Intent, bool) {
	handler, ok := actionHandlerRegistry[invocation.ID]
	if !ok {
		return input.Intent{}, false
	}
	return handler(invocation, event)
}

func shortcutInvocationHasHandler(invocation actiondomain.Invocation) bool {
	_, ok := actionHandlerRegistry[invocation.ID]
	return ok
}

func shortcutInvocationAvailableFromCommand(invocation actiondomain.Invocation) bool {
	policy, ok := shortcut.Policies()[invocation.ID]
	if !ok {
		return false
	}
	for _, sceneID := range policy.AllowedScenes {
		scene, declared := shortcut.SceneByName(string(sceneID))
		if declared && scene.Routable && scene.ID != shortcut.SceneCopy {
			return true
		}
	}
	return false
}

func buildActionHandlerRegistry() map[actiondomain.ID]actionHandler {
	registry := map[actiondomain.ID]actionHandler{}
	register := func(canonical actiondomain.ID, handler actionHandler) {
		if _, exists := registry[canonical]; exists {
			panic("duplicate app action handler " + canonical.String())
		}
		registry[canonical] = handler
	}
	for _, scene := range shortcut.Scenes() {
		if scene.MenuAction == "" {
			continue
		}
		register(scene.MenuAction, func(invocation actiondomain.Invocation, event input.InputEvent) (input.Intent, bool) {
			return shortcutMenuIntent(invocation.ID, input.Intent{Event: event, Invocation: invocation})
		})
	}
	direct := map[string]input.IntentKind{
		"copy.request_older":     input.IntentRequestOlder,
		"copy.request_newer":     input.IntentRequestNewer,
		"clipboard.paste_latest": input.IntentPasteLastCopy,
		"clipboard.paste_system": input.IntentPasteClipboard,
	}
	for id, kind := range direct {
		kind := kind
		register(actiondomain.ID(id), func(invocation actiondomain.Invocation, event input.InputEvent) (input.Intent, bool) {
			return input.Intent{Kind: kind, Event: event, Invocation: invocation}, true
		})
	}
	for _, id := range []string{"copy.line_start", "copy.line_end", "copy.cursor_left", "copy.cursor_right", "copy.cursor_down", "copy.cursor_up", "copy.accept", "copy.oldest", "copy.newest", "copy.half_page_older", "copy.half_page_newer", "copy.mark", "copy.copy_selection", "copy.search_start", "copy.search_mode", "copy.search_next", "copy.search_previous"} {
		id := id
		register(actiondomain.ID(id), func(invocation actiondomain.Invocation, event input.InputEvent) (input.Intent, bool) {
			return input.Intent{Kind: input.IntentCopyCommand, Command: id, Event: event, Invocation: invocation}, true
		})
	}
	for _, spec := range actiondomain.Specs() {
		id := string(spec.ID)
		if _, exists := registry[spec.ID]; exists {
			continue
		}
		if _, ok := shortcutPaneCommand(id); ok {
			register(spec.ID, paneCommandActionHandler)
			continue
		}
		if _, ok := shortcutWorkbenchCommand(actiondomain.Invocation{ID: spec.ID, Params: defaultActionParams(spec)}); ok {
			register(spec.ID, workbenchCommandActionHandler)
			continue
		}
		if _, _, ok := shortcutShellAction(actiondomain.Invocation{ID: spec.ID, Params: defaultActionParams(spec)}); ok {
			register(spec.ID, shellActionHandler)
		}
	}
	for _, id := range []actiondomain.ID{
		"terminal_picker.attach", "terminal_picker.split", "terminal_picker.edit", "terminal_picker.kill", "terminal_picker.delete", "terminal_picker.close",
		"terminal_pool.attach", "terminal_pool.attach_tab", "terminal_pool.attach_float", "terminal_pool.restart", "terminal_pool.edit", "terminal_pool.kill", "terminal_pool.delete", "terminal_pool.close",
		"connections.edit", "connections.toggle", "connections.refresh", "connections.close",
		"workbench_tree.open", "workbench_tree.new", "workbench_tree.rename", "workbench_tree.delete", "workbench_tree.detach", "workbench_tree.zoom", "workbench_tree.close",
		"clipboard_history.paste", "clipboard_history.new", "clipboard_history.edit", "clipboard_history.delete", "clipboard_history.close",
		"floating_overview.open", "floating_overview.show_all", "floating_overview.collapse_all", "floating_overview.close",
		"prompt.submit", "prompt.cancel", "help.previous", "help.next", "help.page_up", "help.page_down", "help.first", "help.last", "help.close",
	} {
		register(id, func(invocation actiondomain.Invocation, event input.InputEvent) (input.Intent, bool) {
			return input.Intent{Kind: input.IntentAppAction, Event: event, Invocation: invocation}, true
		})
	}
	return registry
}

func defaultActionParams(spec actiondomain.Spec) map[string]int {
	if spec.Param == nil {
		return nil
	}
	return map[string]int{spec.Param.Name: spec.Param.Min}
}

func paneCommandActionHandler(invocation actiondomain.Invocation, event input.InputEvent) (input.Intent, bool) {
	command, ok := shortcutPaneCommand(string(invocation.ID))
	return input.Intent{Kind: input.IntentPaneCommand, Command: command, Event: event, Invocation: invocation}, ok
}

func workbenchCommandActionHandler(invocation actiondomain.Invocation, event input.InputEvent) (input.Intent, bool) {
	command, ok := shortcutWorkbenchCommand(invocation)
	return input.Intent{Kind: input.IntentWorkbenchCommand, Command: command, Event: event, Invocation: invocation}, ok
}

func shellActionHandler(invocation actiondomain.Invocation, event input.InputEvent) (input.Intent, bool) {
	action, reason, ok := shortcutShellAction(invocation)
	return input.Intent{Kind: input.IntentShellAction, Action: action, Reason: reason, Event: event, Invocation: invocation}, ok
}

func shortcutMenuIntent(id actiondomain.ID, intent input.Intent) (input.Intent, bool) {
	switch id {
	case "menu.copy":
		intent.Kind = input.IntentEnterCopyMode
	case "menu.terminal_picker":
		intent.Kind = input.IntentOpenTerminalPicker
	case "menu.terminal_pool":
		intent.Kind, intent.Action = input.IntentShellAction, input.ShellActionOpenPool
	case "menu.connections":
		intent.Kind, intent.Action = input.IntentShellAction, input.ShellActionOpenConnections
	case "menu.workbench_tree":
		intent.Kind, intent.Action = input.IntentShellAction, input.ShellActionOpenTree
	case "menu.clipboard_history":
		intent.Kind, intent.Action = input.IntentShellAction, input.ShellActionOpenClipboardHistory
	case "menu.floating_overview":
		intent.Kind, intent.Action = input.IntentShellAction, input.ShellActionFloatingOverview
	case "menu.prompt":
		intent.Kind, intent.Action = input.IntentShellAction, input.ShellActionOpenPrompt
	case "menu.help":
		intent.Kind, intent.Action = input.IntentShellAction, input.ShellActionOpenHelp
	default:
		scene, ok := shortcut.SceneByMenuAction(id)
		if !ok {
			return input.Intent{}, false
		}
		mode, ok := input.InteractionModeForShortcutScene(scene.ID)
		if !ok {
			return input.Intent{}, false
		}
		intent.Kind, intent.Mode = input.IntentSetInteractionMode, mode
	}
	return intent, true
}

func shortcutPaneCommand(id string) (string, bool) {
	commands := map[string]string{
		"panel.split_right": "pane split-right", "panel.split_down": "pane split-down", "panel.toggle_zoom": "pane toggle-zoom", "panel.balance": "pane balance", "panel.presentation_card": "pane presentation card", "panel.presentation_split_line": "pane presentation split-line", "panel.focus_next": "pane focus-next", "panel.focus_prev": "pane focus-prev",
		"panel.kill": "pane kill confirm=accepted", "panel.kill_and_close": "pane close-kill confirm=accepted",
		"resize.left": "pane resize left delta=2", "resize.right": "pane resize right delta=2", "resize.up": "pane resize up delta=2", "resize.down": "pane resize down delta=2", "resize.left_large": "pane resize left delta=6", "resize.right_large": "pane resize right delta=6", "resize.up_large": "pane resize up delta=6", "resize.down_large": "pane resize down delta=6",
	}
	command, ok := commands[id]
	return command, ok
}

func shortcutWorkbenchCommand(invocation actiondomain.Invocation) (string, bool) {
	id := string(invocation.ID)
	commands := map[string]string{
		"panel.close": "pane close", "panel.detach": "pane detach", "panel.reconnect": "pane reconnect", "panel.restart": "pane restart", "panel.take_owner": "pane take-owner", "panel.size_lock": "terminal size lock",
		"resize.layout_toggle": "terminal layout toggle", "resize.pan_left": "terminal layout pan-left", "resize.pan_right": "terminal layout pan-right", "resize.pan_up": "terminal layout pan-up", "resize.pan_down": "terminal layout pan-down", "resize.align_left": "terminal layout align-left", "resize.align_right": "terminal layout align-right", "resize.align_top": "terminal layout align-top", "resize.align_bottom": "terminal layout align-bottom", "resize.center": "terminal layout center", "resize.center_x": "terminal layout center-x", "resize.center_y": "terminal layout center-y", "resize.layout_reset": "terminal layout reset",
		"floating.take_owner": "floating take-owner", "tab.create": "tab create", "tab.next": "tab next", "tab.previous": "tab previous", "tab.rename": "tab rename", "tab.close": "tab close", "tab.kill": "tab kill confirm=accepted", "workspace.create": "workspace create", "workspace.next": "workspace next", "workspace.previous": "workspace previous", "workspace.rename": "workspace rename", "workspace.delete": "workspace delete confirm=accepted",
	}
	if id == "tab.jump" {
		index, ok := invocation.Param("index")
		return "tab jump " + strconv.Itoa(index), ok
	}
	command, ok := commands[id]
	return command, ok
}

func shortcutShellAction(invocation actiondomain.Invocation) (input.ShellAction, string, bool) {
	id := string(invocation.ID)
	plain := map[string]input.ShellAction{
		"system.toggle_header": input.ShellActionToggleHeader, "system.toggle_footer": input.ShellActionToggleFooter, "system.clear_toasts": input.ShellActionClearToasts, "system.close_toast": input.ShellActionCloseToast, "system.toggle_shortcut_lock": input.ShellActionToggleShortcutLock, "system.quit": input.ShellActionQuit,
		"floating.new": input.ShellActionFloatingNew,
	}
	if action, ok := plain[id]; ok {
		return action, "", true
	}
	reasons := map[string]struct {
		action input.ShellAction
		reason string
	}{
		"floating.close": {input.ShellActionFloatingCtrl, "close"}, "floating.collapse": {input.ShellActionFloatingCtrl, "collapse"}, "floating.center": {input.ShellActionFloatingCtrl, "center"}, "floating.toggle_all": {input.ShellActionFloatingGroup, "toggle-all"}, "floating.fit": {input.ShellActionFloatingGroup, "fit"}, "floating.auto_fit": {input.ShellActionFloatingGroup, "toggle-auto-fit"}, "floating.move_left": {input.ShellActionFloatingMove, "left"}, "floating.move_right": {input.ShellActionFloatingMove, "right"}, "floating.move_up": {input.ShellActionFloatingMove, "up"}, "floating.move_down": {input.ShellActionFloatingMove, "down"}, "floating.narrow": {input.ShellActionFloatingSize, "narrow"}, "floating.wide": {input.ShellActionFloatingSize, "wide"}, "floating.short": {input.ShellActionFloatingSize, "short"}, "floating.tall": {input.ShellActionFloatingSize, "tall"},
	}
	if mapped, ok := reasons[id]; ok {
		return mapped.action, mapped.reason, true
	}
	if id == "floating.summon" {
		index, ok := invocation.Param("index")
		return input.ShellActionFloatingSummon, strconv.Itoa(index), ok
	}
	return "", "", false
}

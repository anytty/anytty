package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
)

type ShellSetInteractionModeMsg struct {
	Mode state.InteractionMode
}

func (ShellSetInteractionModeMsg) isMsg() {}

type ShellExitInteractionModeMsg struct{}

func (ShellExitInteractionModeMsg) isMsg() {}

type ShellInteractionModeTimeoutMsg struct {
	Mode state.InteractionMode
	Seq  uint64
}

func (ShellInteractionModeTimeoutMsg) isMsg() {}

type ShellShortcutPassthroughTimeoutMsg struct {
	Kind string
	Seq  uint64
}

func (ShellShortcutPassthroughTimeoutMsg) isMsg() {}

const defaultStickyInteractionModeTimeoutMS = 3000
const defaultShortcutPassthroughIntervalMS = 1000

const stickyInteractionModeTimeoutToken CancelToken = "shell.interaction-mode.timeout"
const shellShortcutPassthroughTimeoutToken CancelToken = "shell.shortcut-passthrough.timeout"

const shortcutPassthroughKindCopy = "copy"

func NewUIInputReducer() Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		if timeoutMsg, ok := msg.(ShellInteractionModeTimeoutMsg); ok {
			return reduceInteractionModeTimeout(root, timeoutMsg)
		}
		if timeoutMsg, ok := msg.(ShellShortcutPassthroughTimeoutMsg); ok {
			return reduceShortcutPassthroughTimeout(root, timeoutMsg)
		}
		if mouseMsg, ok := msg.(ShellOverlayMouseSelectMsg); ok {
			return reduceOverlayMouseSelect(root, mouseMsg)
		}
		inputMsg, ok := msg.(InputMsg)
		if !ok {
			return root, nil
		}
		shell := root.Shell.EnsureDefaults()
		if inputMsg.TerminalMousePassthrough {
			return root, nil
		}
		if shell.Overlay.Open && shell.Overlay.Kind == state.OverlayTerminalPicker {
			return reduceTerminalPickerInput(root, inputMsg.Event)
		}
		if shell.Overlay.Open && shell.Overlay.Kind == state.OverlayTerminalPool {
			return reduceTerminalPoolPageInput(root, inputMsg.Event)
		}
		if shell.Overlay.Open && shell.Overlay.Kind == state.OverlayConnections {
			return reduceConnectionsInput(root, inputMsg.Event)
		}
		if shell.Overlay.Open && shell.Overlay.Kind == state.OverlayWorkbenchTree {
			return reduceWorkbenchTreeInput(root, inputMsg.Event)
		}
		if shell.Overlay.Open && shell.Overlay.Kind == state.OverlayClipboardHistory {
			return reduceClipboardHistoryInput(root, inputMsg.Event)
		}
		if shell.Overlay.Open && shell.Overlay.Kind == state.OverlayFloatingOverview {
			return reduceFloatingOverviewInput(root, inputMsg.Event)
		}
		if shell.Overlay.Open && shell.Overlay.Kind == state.OverlayPrompt {
			return reducePromptInput(root, inputMsg.Event)
		}
		if shell.Overlay.Open && shell.Overlay.Kind == state.OverlayHelp {
			return reduceHelpInput(root, inputMsg.Event)
		}
		if handled, next, effects := reduceEmptyPaneCTAInput(root, inputMsg.Event); handled {
			return next, effects
		}
		if handled, next, effects := reduceDisconnectedPaneCTAInput(root, inputMsg.Event); handled {
			return next, effects
		}
		if handled, next, effects := reduceExitedPaneCTAInput(root, inputMsg.Event); handled {
			return next, effects
		}
		if root.CopyMode.Entering && copyModeOwnsActiveInput(root) {
			return root, nil
		}
		if next, ok := shortcutPassthroughInput(root, inputMsg.Event); ok {
			return next, nil
		}
		intent := input.RouteWithOptions(inputMsg.Event, input.RouteOptions{
			Mode:           inputMode(root.Shell.ReadonlyDefaults().InteractionMode),
			CopyModeActive: copyModeOwnsActiveInput(root),
			Shortcuts:      root.Config.Shortcuts,
		})
		if intent.Kind == input.IntentShortcutAction {
			invocation := intent.Invocation
			var ok bool
			intent, ok = shortcutIntentForInvocation(intent.Invocation, intent.Event)
			if !ok {
				return root, []Effect{handledEffect{}}
			}
			if next, effects, handled := reduceActiveSurfaceShortcut(root, invocation); handled {
				return finishInteractionModeAfterIntent(next, effects, intent)
			}
		}
		switch intent.Kind {
		case input.IntentOpenTerminalPicker:
			root.Shell = root.Shell.OpenTerminalPicker()
			next := root.Advance()
			return finishInteractionModeAfterIntent(next, []Effect{
				handledEffect{},
				terminalPickerListRequestEffect(),
			}, intent)
		case input.IntentSetInteractionMode:
			root.Shell = root.Shell.SetInteractionMode(stateInteractionMode(intent.Mode))
			root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: string(root.Shell.InteractionMode) + " mode"})
			root, effects := armShortcutPassthroughWindow(root, shortcutPassthroughKindForMode(root.Shell.InteractionMode), []Effect{handledEffect{}})
			return root.Advance(), appendInteractionModeTimeoutEffect(root, effects)
		case input.IntentShellAction:
			next, effects := reduceShellActionIntent(root, intent)
			return finishInteractionModeAfterIntent(next, effects, intent)
		case input.IntentPaneCommand:
			next, effects := reducePaneCommandIntent(root, intent)
			return finishInteractionModeAfterIntent(next, effects, intent)
		case input.IntentWorkbenchCommand:
			next, effects := reduceWorkbenchCommandIntent(root, intent)
			return finishInteractionModeAfterIntent(next, effects, intent)
		default:
			return root, nil
		}
	}
}

func reduceShortcutIntent(root state.Root, intent input.Intent) (state.Root, []Effect) {
	return reduceShortcutIntentWithContext(root, intent, -1)
}

func reduceShortcutIntentWithContext(root state.Root, intent input.Intent, row int) (state.Root, []Effect) {
	switch intent.Kind {
	case input.IntentOpenTerminalPicker:
		root.Shell = root.Shell.OpenTerminalPicker()
		return root.Advance(), []Effect{handledEffect{}, terminalPickerListRequestEffect()}
	case input.IntentSetInteractionMode:
		root.Shell = root.Shell.SetInteractionMode(stateInteractionMode(intent.Mode))
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: string(root.Shell.InteractionMode) + " mode"})
		root, effects := armShortcutPassthroughWindow(root, shortcutPassthroughKindForMode(root.Shell.InteractionMode), []Effect{handledEffect{}})
		return root.Advance(), appendInteractionModeTimeoutEffect(root, effects)
	case input.IntentShellAction:
		return reduceShellActionIntent(root, intent)
	case input.IntentPaneCommand:
		return reducePaneCommandIntent(root, intent)
	case input.IntentWorkbenchCommand:
		return reduceWorkbenchCommandIntent(root, intent)
	case input.IntentAppAction:
		next, effects := reduceAppShortcutAction(root, intent.Invocation, row)
		return next, ensureShortcutHandled(effects)
	default:
		return root, []Effect{handledEffect{}}
	}
}

func ensureShortcutHandled(effects []Effect) []Effect {
	for _, effect := range effects {
		if _, ok := effect.(handledEffect); ok {
			return effects
		}
	}
	return append([]Effect{handledEffect{}}, effects...)
}

func shortcutPassthroughInput(root state.Root, event input.InputEvent) (state.Root, bool) {
	if event.Kind != input.EventKindKey {
		return root, false
	}
	shell := root.Shell.EnsureDefaults()
	mode := inputMode(shell.InteractionMode)
	if shell.StickyInteractionMode() {
		if _, ok := input.StickyModeEntryShortcutIntentWithShortcuts(event, mode, root.Config.Shortcuts); ok {
			kind := shortcutPassthroughKindForMode(shell.InteractionMode)
			if shell.ShortcutPassthroughWindowMatches(kind) {
				// 中文说明：双击 sticky prefix 的第二击属于用户显式 PTY 输入；
				// UI reducer 只退出 mode 并把同一 InputMsg 交给 terminal reducer，不能异步补发导致乱序。
				root.Shell = shell.ClearShortcutPassthroughWindow(kind).ExitInteractionMode().ArmTerminalInputPassthroughOnce()
				return root.Advance(), true
			}
			return root, false
		}
		if shell.ShortcutPassthroughLocked {
			if _, ok := input.LockableRootShortcutIntentWithShortcuts(event, root.Config.Shortcuts); ok {
				root.Shell = shell.ClearShortcutPassthroughWindow(shortcutPassthroughKindForMode(shell.InteractionMode)).ExitInteractionMode().ArmTerminalInputPassthroughOnce()
				return root.Advance(), true
			}
		}
		return root, false
	}
	if shell.InteractionMode == state.InteractionModeNormal && shell.ShortcutPassthroughLocked {
		if _, ok := input.LockableRootShortcutIntentWithShortcuts(event, root.Config.Shortcuts); ok {
			// 中文说明：shortcut lock 只让 root shortcut 让路；global 入口保留为解锁控制面。
			root.Shell = shell.ArmTerminalInputPassthroughOnce()
			return root.Advance(), true
		}
	}
	return root, false
}

func shortcutPassthroughKindForMode(mode state.InteractionMode) string {
	if mode == state.InteractionModeNormal {
		return ""
	}
	return "mode:" + string(mode)
}

func reduceInteractionModeTimeout(root state.Root, msg ShellInteractionModeTimeoutMsg) (state.Root, []Effect) {
	shell := root.Shell.EnsureDefaults()
	if !shell.StickyInteractionMode() || shell.InteractionMode != msg.Mode || shell.InteractionModeSeq != msg.Seq {
		return root, nil
	}
	root.Shell = shell.ExitInteractionMode()
	return root.Advance(), nil
}

func reduceShortcutPassthroughTimeout(root state.Root, msg ShellShortcutPassthroughTimeoutMsg) (state.Root, []Effect) {
	nextShell, ok := root.Shell.ClearShortcutPassthroughWindowTimeout(msg.Kind, msg.Seq)
	if !ok {
		return root, nil
	}
	root.Shell = nextShell
	return root.Advance(), nil
}

func appendInteractionModeTimeoutEffect(root state.Root, effects []Effect) []Effect {
	shell := root.Shell.EnsureDefaults()
	if !shell.StickyInteractionMode() {
		return effects
	}
	timeout := stickyInteractionModeTimeout(root)
	if timeout <= 0 {
		return effects
	}
	return append(effects, interactionModeTimeoutEffect(shell.InteractionMode, shell.InteractionModeSeq, timeout))
}

func armShortcutPassthroughWindow(root state.Root, kind string, effects []Effect) (state.Root, []Effect) {
	if kind == "" {
		return root, effects
	}
	root.Shell = root.Shell.ArmShortcutPassthroughWindow(kind)
	if seq, ok := root.Shell.ShortcutPassthroughWindow(kind); ok {
		effects = append(effects, shortcutPassthroughTimeoutEffect(kind, seq, shortcutPassthroughInterval(root)))
	}
	return root, effects
}

func rearmInteractionModeTimeout(root state.Root, effects []Effect) (state.Root, []Effect) {
	shell := root.Shell.EnsureDefaults()
	if !shell.StickyInteractionMode() {
		root.Shell = shell
		return root, effects
	}
	root.Shell = shell.RearmInteractionMode()
	return root, appendInteractionModeTimeoutEffect(root, effects)
}

func finishInteractionModeAfterIntent(root state.Root, effects []Effect, intent input.Intent) (state.Root, []Effect) {
	shell := root.Shell.EnsureDefaults()
	if !shell.StickyInteractionMode() {
		root.Shell = shell
		return root, effects
	}
	if interactionIntentKeepsPrefixMode(intent) {
		return rearmInteractionModeTimeout(root, effects)
	}
	// 普通 prefix 命令执行后回到主菜单；只有连续调节类动作会续期留在当前 mode。
	root.Shell = shell.ExitInteractionMode()
	return root, effects
}

func interactionIntentKeepsPrefixMode(intent input.Intent) bool {
	switch intent.Kind {
	case input.IntentPaneCommand:
		return paneIntentKeepsPrefixMode(intent.Command)
	case input.IntentShellAction:
		return shellActionIntentKeepsPrefixMode(intent)
	case input.IntentWorkbenchCommand:
		return workbenchIntentKeepsPrefixMode(intent.Command)
	default:
		return false
	}
}

func paneIntentKeepsPrefixMode(command string) bool {
	if strings.HasPrefix(command, "pane resize ") {
		return true
	}
	switch command {
	case "pane focus-next", "pane focus-prev":
		return true
	default:
		return false
	}
}

func shellActionIntentKeepsPrefixMode(intent input.Intent) bool {
	switch intent.Action {
	case input.ShellActionFloatingMove, input.ShellActionFloatingSize:
		return true
	default:
		return false
	}
}

func workbenchIntentKeepsPrefixMode(command string) bool {
	switch command {
	case "tab next", "tab previous", "workspace next", "workspace previous",
		"terminal layout pan-left", "terminal layout pan-right", "terminal layout pan-up", "terminal layout pan-down":
		return true
	default:
		return false
	}
}

func stickyInteractionModeTimeout(root state.Root) time.Duration {
	ms := root.Config.Interaction.StickyPrefixTimeoutMS
	if root.Config.Version == 0 {
		ms = defaultStickyInteractionModeTimeoutMS
	}
	if ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

func shortcutPassthroughInterval(root state.Root) time.Duration {
	ms := root.Config.Interaction.ShortcutPassthroughIntervalMS
	if root.Config.Version == 0 || ms <= 0 {
		ms = defaultShortcutPassthroughIntervalMS
	}
	return time.Duration(ms) * time.Millisecond
}

func interactionModeTimeoutEffect(mode state.InteractionMode, seq uint64, timeout time.Duration) Effect {
	return FuncEffect{
		Token: stickyInteractionModeTimeoutToken,
		Async: true,
		Run: func(ctx context.Context) Msg {
			// sticky mode 是前缀键提示态；超时只退出 mode，不关闭 overlay/copy 页面。
			timer := time.NewTimer(timeout)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return nil
			case <-timer.C:
				return ShellInteractionModeTimeoutMsg{Mode: mode, Seq: seq}
			}
		},
	}
}

func shortcutPassthroughTimeoutEffect(kind string, seq uint64, timeout time.Duration) Effect {
	return FuncEffect{
		Token: shellShortcutPassthroughTimeoutToken,
		Async: true,
		Run: func(ctx context.Context) Msg {
			// entry 双击窗口只决定“第二次入口键是否透传”，不能退出 sticky/copy 状态。
			timer := time.NewTimer(timeout)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return nil
			case <-timer.C:
				return ShellShortcutPassthroughTimeoutMsg{Kind: kind, Seq: seq}
			}
		},
	}
}

func reduceOverlayMouseSelect(root state.Root, msg ShellOverlayMouseSelectMsg) (state.Root, []Effect) {
	shell := root.Shell.EnsureDefaults()
	if !shell.Overlay.Open || msg.Delta == 0 {
		return root, []Effect{handledEffect{}}
	}
	// 中文说明：overlay 滚轮只移动当前弹层选择，不允许事件继续落到底层 terminal。
	effects := []Effect{handledEffect{}}
	switch shell.Overlay.Kind {
	case state.OverlayTerminalPicker:
		root.Shell = shell.MoveTerminalPickerSelection(msg.Delta, len(state.TerminalPickerItems(root)))
	case state.OverlayTerminalPool:
		root.Shell = shell.MoveTerminalPoolSelection(msg.Delta, len(state.TerminalPoolPageItems(root)))
		effects = append(effects, terminalPoolPreviewRefreshEffect())
	case state.OverlayConnections:
		root.Shell = shell.MoveConnectionsSelection(msg.Delta, len(root.Endpoints.Items))
	case state.OverlayWorkbenchTree:
		root.Shell = shell.MoveWorkbenchTreeSelection(msg.Delta, len(state.WorkbenchTreeItems(root)))
	case state.OverlayClipboardHistory:
		root.Shell = shell.MoveClipboardHistorySelection(msg.Delta, len(state.ClipboardHistoryItems(root)))
	case state.OverlayFloatingOverview:
		root.Shell = shell.MoveFloatingOverviewSelection(msg.Delta, len(state.FloatingOverviewItems(root)))
	default:
		return root, []Effect{handledEffect{}}
	}
	return root.Advance(), effects
}

func reduceEmptyPaneCTAInput(root state.Root, event input.InputEvent) (bool, state.Root, []Effect) {
	if event.Kind != input.EventKindKey {
		return false, root, nil
	}
	shell := root.Shell.EnsureDefaults()
	if shell.InteractionMode != state.InteractionModeNormal {
		return false, root, nil
	}
	pane, floating, ok := activeEmptyPaneCTATarget(root, shell)
	if !ok {
		return false, root, nil
	}
	switch event.Key {
	case input.KeyUp:
		root.Shell = shell.MoveEmptyPaneCTASelection(-1, len(actiondomain.EmptyPaneCTAActions()))
		return true, root.Advance(), []Effect{handledEffect{}}
	case input.KeyDown:
		root.Shell = shell.MoveEmptyPaneCTASelection(1, len(actiondomain.EmptyPaneCTAActions()))
		return true, root.Advance(), []Effect{handledEffect{}}
	case input.KeyEnter:
		actionID, ok := selectedSurfaceAction(actiondomain.EmptyPaneCTAActions(), shell.EmptyPaneCTA.SelectedIndex)
		if !ok {
			return true, root, []Effect{handledEffect{}}
		}
		return true, root, []Effect{
			handledEffect{},
			FuncEffect{Run: func(context.Context) Msg {
				return shellShortcutMessageForSurfaceAction(actionID, pane.ID, floating)
			}},
		}
	default:
		return false, root, nil
	}
}

func activeEmptyPaneCTATarget(root state.Root, shell state.ShellStore) (state.PaneState, bool, bool) {
	shell = shell.ReadonlyDefaults()
	if activeFloatingID := shell.ActiveFloatingID(); activeFloatingID != "" {
		for _, floating := range shell.ActiveFloatings() {
			if floating.ID == activeFloatingID && floating.Pane.Kind == state.PaneEmpty && !floatingHasTerminalBinding(root, floating.ID) {
				return floating.Pane, true, true
			}
		}
		return state.PaneState{}, false, false
	}
	pane, ok := shell.Pane(state.PaneCommandTarget{PaneID: shell.ActivePaneID})
	if !ok || pane.Kind != state.PaneEmpty || paneHasTerminalBinding(root, pane.ID) {
		return state.PaneState{}, false, false
	}
	return pane, false, true
}

func reduceDisconnectedPaneCTAInput(root state.Root, event input.InputEvent) (bool, state.Root, []Effect) {
	if event.Kind != input.EventKindKey {
		return false, root, nil
	}
	shell := root.Shell.EnsureDefaults()
	if shell.InteractionMode != state.InteractionModeNormal {
		return false, root, nil
	}
	pane, floating, ok := activeDisconnectedPaneCTATarget(root, shell)
	if !ok {
		return false, root, nil
	}
	switch event.Key {
	case input.KeyUp:
		root.Shell = shell.MoveExitedPaneCTASelection(-1, len(actiondomain.DisconnectedPaneCTAActions()))
		return true, root.Advance(), []Effect{handledEffect{}}
	case input.KeyDown:
		root.Shell = shell.MoveExitedPaneCTASelection(1, len(actiondomain.DisconnectedPaneCTAActions()))
		return true, root.Advance(), []Effect{handledEffect{}}
	case input.KeyEnter:
		actionID, ok := selectedSurfaceAction(actiondomain.DisconnectedPaneCTAActions(), shell.ExitedPaneCTA.SelectedIndex)
		if !ok {
			return true, root, []Effect{handledEffect{}}
		}
		return true, root, []Effect{
			handledEffect{},
			FuncEffect{Run: func(context.Context) Msg {
				return shellShortcutMessageForSurfaceAction(actionID, pane.ID, floating)
			}},
		}
	default:
		return false, root, nil
	}
}

func activeDisconnectedPaneCTATarget(root state.Root, shell state.ShellStore) (state.PaneState, bool, bool) {
	shell = shell.ReadonlyDefaults()
	if activeFloatingID := shell.ActiveFloatingID(); activeFloatingID != "" {
		for _, floating := range shell.ActiveFloatings() {
			if floating.ID == activeFloatingID && paneHasDisconnectedTerminal(root, floating.Pane.ID, true) {
				return floating.Pane, true, true
			}
		}
		return state.PaneState{}, false, false
	}
	pane, ok := shell.Pane(state.PaneCommandTarget{PaneID: shell.ActivePaneID})
	if !ok || !paneHasDisconnectedTerminal(root, pane.ID, false) {
		return state.PaneState{}, false, false
	}
	return pane, false, true
}

func reduceExitedPaneCTAInput(root state.Root, event input.InputEvent) (bool, state.Root, []Effect) {
	if event.Kind != input.EventKindKey {
		return false, root, nil
	}
	shell := root.Shell.EnsureDefaults()
	if shell.InteractionMode != state.InteractionModeNormal {
		return false, root, nil
	}
	pane, floating, ok := activeExitedPaneCTATarget(root, shell)
	if !ok {
		return false, root, nil
	}
	switch event.Key {
	case input.KeyUp:
		root.Shell = shell.MoveExitedPaneCTASelection(-1, len(actiondomain.ExitedPaneCTAActions()))
		return true, root.Advance(), []Effect{handledEffect{}}
	case input.KeyDown:
		root.Shell = shell.MoveExitedPaneCTASelection(1, len(actiondomain.ExitedPaneCTAActions()))
		return true, root.Advance(), []Effect{handledEffect{}}
	case input.KeyEnter:
		actionID, ok := selectedSurfaceAction(actiondomain.ExitedPaneCTAActions(), shell.ExitedPaneCTA.SelectedIndex)
		if !ok {
			return true, root, []Effect{handledEffect{}}
		}
		return true, root, []Effect{
			handledEffect{},
			FuncEffect{Run: func(context.Context) Msg {
				return shellShortcutMessageForSurfaceAction(actionID, pane.ID, floating)
			}},
		}
	default:
		return false, root, nil
	}
}

func shellShortcutMessageForSurfaceAction(id actiondomain.ID, paneID string, floating bool) Msg {
	return ShellShortcutActionMsg{
		Invocation: actiondomain.Invocation{ID: id, SourceActionID: id.String()},
		Surface:    &ShortcutSurfaceContext{ExplicitTarget: true, PaneID: paneID, Floating: floating, Row: -1},
	}
}

func selectedSurfaceAction(actions []actiondomain.ID, index int) (actiondomain.ID, bool) {
	if index < 0 || index >= len(actions) {
		return "", false
	}
	return actions[index], true
}

func activeExitedPaneCTATarget(root state.Root, shell state.ShellStore) (state.PaneState, bool, bool) {
	shell = shell.ReadonlyDefaults()
	if activeFloatingID := shell.ActiveFloatingID(); activeFloatingID != "" {
		for _, floating := range shell.ActiveFloatings() {
			if floating.ID == activeFloatingID && paneHasExitedTerminal(root, floating.Pane.ID, true) {
				return floating.Pane, true, true
			}
		}
		return state.PaneState{}, false, false
	}
	pane, ok := shell.Pane(state.PaneCommandTarget{PaneID: shell.ActivePaneID})
	if !ok || !paneHasExitedTerminal(root, pane.ID, false) {
		return state.PaneState{}, false, false
	}
	return pane, false, true
}

func paneHasExitedTerminal(root state.Root, paneID string, floating bool) bool {
	ref := state.TerminalRef{}
	if floating {
		if floatingID, ok := root.Shell.ReadonlyDefaults().FloatingIDForPaneID(paneID); ok {
			if binding, ok := root.TerminalViews.FloatingBinding(floatingID); ok {
				ref = binding.TerminalRef()
			}
		}
	} else if binding, ok := root.TerminalViews.PaneBinding(paneID); ok {
		ref = binding.TerminalRef()
	}
	if ref.Empty() {
		return false
	}
	// 退出 CTA 只认当前 TerminalView binding 对应的 reducer lifecycle。
	// surface/session 都是 core/live 消息回投，不从 pane kind 或 workbench storage 推断。
	surface := root.Surface.SurfaceForTerminalRef(ref)
	if surface.State == state.TerminalLiveExited {
		return true
	}
	return root.Session.TerminalRef().Equal(ref) && root.Session.State == state.TerminalLiveExited
}

func paneHasDisconnectedTerminal(root state.Root, paneID string, floating bool) bool {
	binding, ok := terminalBindingForPaneCTA(root, paneID, floating)
	if !ok || binding.TerminalID == "" {
		return false
	}
	ref := binding.TerminalRef()
	surface := root.Surface.SurfaceForTerminalRef(ref)
	if surface.State == state.TerminalLiveExited {
		return false
	}
	if binding.LastError != "" && !binding.Attached {
		return true
	}
	if surface.State == state.TerminalLiveError && surface.Err != "" {
		return true
	}
	return root.Session.TerminalRef().Equal(ref) && root.Session.State == state.TerminalLiveError && root.Session.LastError != ""
}

func terminalBindingForPaneCTA(root state.Root, paneID string, floating bool) (state.TerminalViewBinding, bool) {
	if floating {
		if floatingID, ok := root.Shell.ReadonlyDefaults().FloatingIDForPaneID(paneID); ok {
			return root.TerminalViews.FloatingBinding(floatingID)
		}
		return state.TerminalViewBinding{}, false
	}
	return root.TerminalViews.PaneBinding(paneID)
}

func paneHasTerminalBinding(root state.Root, paneID string) bool {
	binding, ok := root.TerminalViews.PaneBinding(paneID)
	return ok && binding.TerminalID != ""
}

func floatingHasTerminalBinding(root state.Root, floatingID string) bool {
	binding, ok := root.TerminalViews.FloatingBinding(floatingID)
	return ok && binding.TerminalID != ""
}

func reduceTerminalPickerInput(root state.Root, event input.InputEvent) (state.Root, []Effect) {
	if event.Kind != input.EventKindKey {
		return root, nil
	}
	items := state.TerminalPickerItems(root)
	if entry, ok := input.ShortcutEntryForEvent(root.Config.Shortcuts, "terminal_picker", event); ok {
		return reduceTerminalPickerShortcut(root, entry, items, event)
	}
	switch event.Key {
	case input.KeyUp:
		root.Shell = root.Shell.MoveTerminalPickerSelection(-1, len(items))
		return root.Advance(), []Effect{handledEffect{}}
	case input.KeyDown:
		root.Shell = root.Shell.MoveTerminalPickerSelection(1, len(items))
		return root.Advance(), []Effect{handledEffect{}}
	case input.KeyBackspace, input.KeyDelete:
		root.Shell = root.Shell.SetTerminalPickerQuery(trimLastRune(root.Shell.EnsureDefaults().Overlay.Query))
		return root.Advance(), []Effect{handledEffect{}}
	case input.KeyChar:
		if isBackspaceEvent(event) {
			root.Shell = root.Shell.SetTerminalPickerQuery(trimLastRune(root.Shell.EnsureDefaults().Overlay.Query))
			return root.Advance(), []Effect{handledEffect{}}
		}
		if event.Ctrl || event.Char == "" {
			return root, []Effect{handledEffect{}}
		}
		root.Shell = root.Shell.SetTerminalPickerQuery(root.Shell.EnsureDefaults().Overlay.Query + event.Char)
		return root.Advance(), []Effect{handledEffect{}}
	default:
		return root, []Effect{handledEffect{}}
	}
}

func reduceTerminalPickerShortcut(root state.Root, entry input.ShortcutEntry, items []state.TerminalPickerItem, event input.InputEvent) (state.Root, []Effect) {
	return reduceOverlayShortcutAction(root, entry, event)
}

func reduceTerminalPickerConfirm(root state.Root, items []state.TerminalPickerItem) (state.Root, []Effect) {
	if len(items) == 0 {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.attach", Body: "no terminal"})
		return root.Advance(), []Effect{handledEffect{}}
	}
	selected := items[0]
	for _, item := range items {
		if item.Selected {
			selected = item
			break
		}
	}
	if selected.CreateNew {
		target := terminalPoolTargetForOverlay(root)
		return root, []Effect{
			handledEffect{},
			FuncEffect{Run: func(context.Context) Msg {
				return ShellOpenPromptMsg{Prompt: createTerminalPromptForTargetEndpoint(root, target, selected.EndpointID)}
			}},
		}
	}
	if selected.TerminalID == "" {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.attach", Body: "no terminal"})
		return root.Advance(), []Effect{handledEffect{}}
	}
	target := terminalPoolTargetForOverlay(root)
	return root, []Effect{
		handledEffect{},
		FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolAttachRequestMsg{EndpointID: selected.EndpointID, TerminalID: selected.TerminalID, TargetPaneID: target.PaneID, TargetFloatingID: target.FloatingID}
		}},
	}
}

func reduceTerminalPoolPageInput(root state.Root, event input.InputEvent) (state.Root, []Effect) {
	if event.Kind != input.EventKindKey {
		return root, nil
	}
	items := state.TerminalPoolPageItems(root)
	if entry, ok := input.ShortcutEntryForEvent(root.Config.Shortcuts, "terminal_pool", event); ok {
		return reduceTerminalPoolShortcut(root, entry, items, event)
	}
	switch event.Key {
	case input.KeyUp:
		root.Shell = root.Shell.MoveTerminalPoolSelection(-1, len(items))
		return root.Advance(), terminalPoolPageHandledEffects()
	case input.KeyDown:
		root.Shell = root.Shell.MoveTerminalPoolSelection(1, len(items))
		return root.Advance(), terminalPoolPageHandledEffects()
	case input.KeyBackspace, input.KeyDelete:
		root.Shell = root.Shell.SetTerminalPoolQuery(trimLastRune(root.Shell.EnsureDefaults().Overlay.Query))
		return root.Advance(), terminalPoolPageHandledEffects()
	case input.KeyChar:
		if isBackspaceEvent(event) {
			root.Shell = root.Shell.SetTerminalPoolQuery(trimLastRune(root.Shell.EnsureDefaults().Overlay.Query))
			return root.Advance(), terminalPoolPageHandledEffects()
		}
		if event.Ctrl || event.Char == "" {
			return root, []Effect{handledEffect{}}
		}
		root.Shell = root.Shell.SetTerminalPoolQuery(root.Shell.EnsureDefaults().Overlay.Query + event.Char)
		return root.Advance(), terminalPoolPageHandledEffects()
	default:
		return root, []Effect{handledEffect{}}
	}
}

func reduceTerminalPoolShortcut(root state.Root, entry input.ShortcutEntry, items []state.TerminalPoolPageItem, event input.InputEvent) (state.Root, []Effect) {
	return reduceOverlayShortcutAction(root, entry, event)
}

func terminalPoolPageHandledEffects() []Effect {
	return []Effect{handledEffect{}, terminalPoolPreviewRefreshEffect()}
}

func terminalPoolPreviewRefreshEffect() Effect {
	return FuncEffect{Run: func(context.Context) Msg { return TerminalPoolPreviewRefreshMsg{} }}
}

func reduceTerminalPoolPageAttach(root state.Root, items []state.TerminalPoolPageItem) (state.Root, []Effect) {
	if len(items) == 0 {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "pool.attach", Body: "no terminal"})
		return root.Advance(), []Effect{handledEffect{}}
	}
	selected := items[0]
	for _, item := range items {
		if item.Selected {
			selected = item
			break
		}
	}
	if selected.TerminalID == "" {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "pool.attach", Body: "no terminal"})
		return root.Advance(), []Effect{handledEffect{}}
	}
	target := terminalPoolTargetForOverlay(root)
	return root, []Effect{
		handledEffect{},
		FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolAttachRequestMsg{EndpointID: selected.EndpointID, TerminalID: selected.TerminalID, TargetPaneID: target.PaneID, TargetFloatingID: target.FloatingID}
		}},
	}
}

func reduceWorkbenchTreeInput(root state.Root, event input.InputEvent) (state.Root, []Effect) {
	if event.Kind != input.EventKindKey {
		return root, nil
	}
	items := state.WorkbenchTreeItems(root)
	if entry, ok := input.ShortcutEntryForEvent(root.Config.Shortcuts, "workbench_tree", event); ok {
		return reduceWorkbenchTreeShortcut(root, entry, items, event)
	}
	switch event.Key {
	case input.KeyUp:
		root.Shell = root.Shell.MoveWorkbenchTreeSelection(-1, len(items))
		return root.Advance(), []Effect{handledEffect{}}
	case input.KeyDown:
		root.Shell = root.Shell.MoveWorkbenchTreeSelection(1, len(items))
		return root.Advance(), []Effect{handledEffect{}}
	case input.KeyLeft:
		if selected, ok := workbenchTreeSelectedItem(items); ok && selected.Expandable && !selected.Collapsed {
			root.Shell = root.Shell.SetWorkbenchTreeItemCollapsed(selected, true)
			return root.Advance(), []Effect{handledEffect{}}
		}
		return root, []Effect{handledEffect{}}
	case input.KeyRight:
		if selected, ok := workbenchTreeSelectedItem(items); ok && selected.Expandable && selected.Collapsed {
			root.Shell = root.Shell.SetWorkbenchTreeItemCollapsed(selected, false)
			return root.Advance(), []Effect{handledEffect{}}
		}
		return root, []Effect{handledEffect{}}
	case input.KeyBackspace, input.KeyDelete:
		root.Shell = root.Shell.SetWorkbenchTreeQuery(trimLastRune(root.Shell.EnsureDefaults().Overlay.Query))
		return root.Advance(), []Effect{handledEffect{}}
	case input.KeyChar:
		if isBackspaceEvent(event) {
			root.Shell = root.Shell.SetWorkbenchTreeQuery(trimLastRune(root.Shell.EnsureDefaults().Overlay.Query))
			return root.Advance(), []Effect{handledEffect{}}
		}
		if event.Ctrl || event.Char == "" {
			return root, []Effect{handledEffect{}}
		}
		root.Shell = root.Shell.SetWorkbenchTreeQuery(root.Shell.EnsureDefaults().Overlay.Query + event.Char)
		return root.Advance(), []Effect{handledEffect{}}
	default:
		return root, []Effect{handledEffect{}}
	}
}

func reduceWorkbenchTreeShortcut(root state.Root, entry input.ShortcutEntry, items []state.WorkbenchTreeItem, event input.InputEvent) (state.Root, []Effect) {
	return reduceOverlayShortcutAction(root, entry, event)
}

func reduceClipboardHistoryInput(root state.Root, event input.InputEvent) (state.Root, []Effect) {
	if event.Kind != input.EventKindKey {
		return root, nil
	}
	items := state.ClipboardHistoryItems(root)
	if entry, ok := input.ShortcutEntryForEvent(root.Config.Shortcuts, "clipboard_history", event); ok {
		return reduceOverlayShortcutAction(root, entry, event)
	}
	switch event.Key {
	case input.KeyUp:
		root.Shell = root.Shell.MoveClipboardHistorySelection(-1, len(items))
		return root.Advance(), []Effect{handledEffect{}}
	case input.KeyDown:
		root.Shell = root.Shell.MoveClipboardHistorySelection(1, len(items))
		return root.Advance(), []Effect{handledEffect{}}
	case input.KeyBackspace, input.KeyDelete:
		root.Shell = root.Shell.SetClipboardHistoryQuery(trimLastRune(root.Shell.EnsureDefaults().Overlay.Query))
		return root.Advance(), []Effect{handledEffect{}}
	case input.KeyChar:
		if isBackspaceEvent(event) {
			root.Shell = root.Shell.SetClipboardHistoryQuery(trimLastRune(root.Shell.EnsureDefaults().Overlay.Query))
			return root.Advance(), []Effect{handledEffect{}}
		}
		if event.Ctrl || event.Char == "" {
			return root, []Effect{handledEffect{}}
		}
		root.Shell = root.Shell.SetClipboardHistoryQuery(root.Shell.EnsureDefaults().Overlay.Query + event.Char)
		return root.Advance(), []Effect{handledEffect{}}
	default:
		return root, []Effect{handledEffect{}}
	}
}

func reduceFloatingOverviewInput(root state.Root, event input.InputEvent) (state.Root, []Effect) {
	if event.Kind != input.EventKindKey {
		return root, nil
	}
	items := state.FloatingOverviewItems(root)
	if entry, ok := input.ShortcutEntryForEvent(root.Config.Shortcuts, "floating_overview", event); ok {
		return reduceFloatingOverviewShortcut(root, entry, items, event)
	}
	switch event.Key {
	case input.KeyUp:
		root.Shell = root.Shell.MoveFloatingOverviewSelection(-1, len(items))
		return root.Advance(), []Effect{handledEffect{}}
	case input.KeyDown:
		root.Shell = root.Shell.MoveFloatingOverviewSelection(1, len(items))
		return root.Advance(), []Effect{handledEffect{}}
	case input.KeyChar:
		return root, []Effect{handledEffect{}}
	default:
		return root, []Effect{handledEffect{}}
	}
}

func reduceFloatingOverviewShortcut(root state.Root, entry input.ShortcutEntry, items []state.FloatingOverviewItem, event input.InputEvent) (state.Root, []Effect) {
	return reduceOverlayShortcutAction(root, entry, event)
}

func reduceFloatingOverviewOpen(root state.Root, items []state.FloatingOverviewItem) (state.Root, []Effect) {
	selected, ok := selectedFloatingOverviewItem(items)
	if !ok {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "floating.open", Body: "no floating"})
		return root.Advance(), []Effect{handledEffect{}}
	}
	return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg {
		return ShellFloatingCommandMsg{Command: state.FloatingCommand{Action: state.FloatingCommandSummon, TargetID: selected.FloatingID, Source: state.PaneCommandSourceKeyboard}}
	}}}
}

func reduceOverlayShortcutAction(root state.Root, entry input.ShortcutEntry, event input.InputEvent) (state.Root, []Effect) {
	invocation, _, err := actiondomain.ParseInvocation(entry.ActionID)
	if err != nil {
		return root, []Effect{handledEffect{}}
	}
	intent, ok := shortcutIntentForInvocation(invocation, event)
	if !ok {
		return root, []Effect{handledEffect{}}
	}
	return reduceShortcutIntent(root, intent)
}

func reducePromptInput(root state.Root, event input.InputEvent) (state.Root, []Effect) {
	if event.Kind == input.EventKindPaste {
		root.Shell = refreshPromptCompletions(root, root.Shell.SetPromptSuggestionFocused(false).InsertPromptText(event.Paste))
		return root.Advance(), promptCompletionHandledEffects(root, false)
	}
	if event.Kind != input.EventKindKey {
		return root, nil
	}
	shell := root.Shell.EnsureDefaults()
	if shell.Overlay.Prompt.SuggestionFocused {
		return reducePromptSuggestionInput(root, event)
	}
	if entry, ok := input.ShortcutEntryForEvent(root.Config.Shortcuts, "prompt", event); ok {
		return reducePromptShortcut(root, entry, event)
	}
	switch event.Key {
	case input.KeyTab:
		root.Shell = refreshPromptCompletions(root, root.Shell)
		if len(root.Shell.EnsureDefaults().Overlay.Prompt.ActiveSuggestionItems()) > 0 {
			root.Shell = root.Shell.SetPromptSuggestionFocused(true)
			return root.Advance(), []Effect{handledEffect{}}
		}
		if effects := promptPathCompletionTriggerEffect(root, true); len(effects) > 0 {
			return root.Advance(), append([]Effect{handledEffect{}}, effects...)
		}
		root.Shell = refreshPromptCompletions(root, root.Shell.MovePromptField(1))
		return root.Advance(), promptCompletionHandledEffects(root, false)
	case input.KeyDown:
		root.Shell = root.Shell.MovePromptField(1)
		root.Shell = refreshPromptCompletions(root, root.Shell)
		return root.Advance(), promptCompletionHandledEffects(root, false)
	case input.KeyUp, input.KeyShiftTab:
		root.Shell = root.Shell.MovePromptField(-1)
		root.Shell = refreshPromptCompletions(root, root.Shell)
		return root.Advance(), promptCompletionHandledEffects(root, false)
	case input.KeyBackspace:
		root.Shell = refreshPromptCompletions(root, root.Shell.DeletePromptBackward())
		return root.Advance(), promptCompletionHandledEffects(root, false)
	case input.KeyDelete:
		root.Shell = refreshPromptCompletions(root, root.Shell.DeletePromptForward())
		return root.Advance(), promptCompletionHandledEffects(root, false)
	case input.KeyLeft:
		root.Shell = refreshPromptCompletions(root, root.Shell.MovePromptCursor(-1))
		return root.Advance(), promptCompletionHandledEffects(root, false)
	case input.KeyRight:
		root.Shell = refreshPromptCompletions(root, root.Shell.MovePromptCursor(1))
		return root.Advance(), promptCompletionHandledEffects(root, false)
	case input.KeyHome:
		root.Shell = refreshPromptCompletions(root, root.Shell.SetPromptCursor(0))
		return root.Advance(), promptCompletionHandledEffects(root, false)
	case input.KeyEnd:
		root.Shell = refreshPromptCompletions(root, root.Shell.SetPromptCursor(len([]rune(promptEditableValue(root.Shell.EnsureDefaults().Overlay.Prompt)))))
		return root.Advance(), promptCompletionHandledEffects(root, false)
	case input.KeyChar:
		if isBackspaceEvent(event) {
			root.Shell = refreshPromptCompletions(root, root.Shell.DeletePromptBackward())
			return root.Advance(), promptCompletionHandledEffects(root, false)
		}
		if event.Ctrl || event.Char == "" {
			return root, []Effect{handledEffect{}}
		}
		root.Shell = refreshPromptCompletions(root, root.Shell.InsertPromptText(event.Char))
		return root.Advance(), promptCompletionHandledEffects(root, false)
	default:
		return root, []Effect{handledEffect{}}
	}
}

func reducePromptShortcut(root state.Root, entry input.ShortcutEntry, event input.InputEvent) (state.Root, []Effect) {
	return reduceOverlayShortcutAction(root, entry, event)
}

func reducePromptSuggestionInput(root state.Root, event input.InputEvent) (state.Root, []Effect) {
	triggerCompletion := false
	focusCompletion := false
	switch event.Key {
	case input.KeyUp:
		root.Shell = root.Shell.MovePromptSuggestionSelection(-1)
	case input.KeyDown:
		root.Shell = root.Shell.MovePromptSuggestionSelection(1)
	case input.KeyEnter:
		root.Shell = refreshPromptCompletions(root, root.Shell.AcceptPromptSuggestion())
		triggerCompletion = true
	case input.KeyRight:
		root.Shell = refreshPromptCompletions(root, root.Shell.EnterPromptSuggestion())
		triggerCompletion = true
		focusCompletion = true
		if len(root.Shell.EnsureDefaults().Overlay.Prompt.ActiveSuggestionItems()) > 0 {
			root.Shell = root.Shell.SetPromptSuggestionFocused(true)
		}
	case input.KeyLeft:
		root.Shell = refreshPromptCompletions(root, root.Shell.LeavePromptSuggestionPath())
		triggerCompletion = true
		focusCompletion = true
		if len(root.Shell.EnsureDefaults().Overlay.Prompt.ActiveSuggestionItems()) > 0 {
			root.Shell = root.Shell.SetPromptSuggestionFocused(true)
		}
	case input.KeyTab:
		root.Shell = root.Shell.MovePromptSuggestionSelection(1)
	case input.KeyShiftTab:
		root.Shell = root.Shell.MovePromptSuggestionSelection(-1)
	case input.KeyBackspace:
		root.Shell = refreshPromptCompletions(root, root.Shell.SetPromptSuggestionFocused(false).DeletePromptBackward())
		triggerCompletion = true
	case input.KeyDelete:
		root.Shell = refreshPromptCompletions(root, root.Shell.SetPromptSuggestionFocused(false).DeletePromptForward())
		triggerCompletion = true
	case input.KeyChar:
		if isBackspaceEvent(event) {
			root.Shell = refreshPromptCompletions(root, root.Shell.SetPromptSuggestionFocused(false).DeletePromptBackward())
			triggerCompletion = true
			break
		}
		if !event.Ctrl && event.Char != "" {
			root.Shell = refreshPromptCompletions(root, root.Shell.SetPromptSuggestionFocused(false).InsertPromptText(event.Char))
			triggerCompletion = true
		}
	default:
		return root, []Effect{handledEffect{}}
	}
	if triggerCompletion {
		return root.Advance(), promptCompletionHandledEffects(root, focusCompletion)
	}
	return root.Advance(), []Effect{handledEffect{}}
}

func promptEditableValue(prompt state.PromptState) string {
	if active := prompt.ActivePromptField(); active != nil {
		return active.Value
	}
	return prompt.Value
}

func reduceHelpInput(root state.Root, event input.InputEvent) (state.Root, []Effect) {
	if event.Kind != input.EventKindKey {
		return root, nil
	}
	if entry, ok := input.ShortcutEntryForEvent(root.Config.Shortcuts, "help", event); ok {
		return reduceOverlayShortcutAction(root, entry, event)
	}
	return root, []Effect{handledEffect{}}
}

func isBackspaceEvent(event input.InputEvent) bool {
	return event.Key == input.KeyBackspace || (event.Key == input.KeyChar && (event.Char == "\x7f" || event.Char == "\b"))
}

func trimLastRune(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	return string(runes[:len(runes)-1])
}

func reducePaneCommandIntent(root state.Root, intent input.Intent) (state.Root, []Effect) {
	command, ok := PaneCommandFromIntent(intent)
	if !ok {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "pane command", Body: "invalid shortcut"})
		return root.Advance(), []Effect{handledEffect{}}
	}
	if command.Action == state.PaneCommandSplit && command.NewPane.ID == "" {
		command.NewPane = state.PaneState{ID: nextKeyboardPaneID(root.Shell), Title: "pane", Kind: state.PaneEmpty}
	}
	if command.Action == state.PaneCommandSplit {
		return root, []Effect{
			handledEffect{},
			FuncEffect{Run: func(context.Context) Msg {
				return ShellWorkbenchCommandMsg{Command: state.WorkbenchCommand{
					Action: state.WorkbenchCommandPaneSplit,
					Pane:   command,
					Source: state.PaneCommandSourceKeyboard,
				}}
			}},
		}
	}
	return root, []Effect{
		handledEffect{},
		FuncEffect{Run: func(context.Context) Msg {
			return ShellPaneCommandMsg{Command: command}
		}},
	}
}

func reduceWorkbenchCommandIntent(root state.Root, intent input.Intent) (state.Root, []Effect) {
	if next, effects, ok := reduceViewWorkbenchShortcut(root, intent.Command); ok {
		return next, effects
	}
	command, prompt, ok := workbenchCommandFromIntent(root, intent)
	if !ok {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench command", Body: "invalid shortcut"})
		return root.Advance(), []Effect{handledEffect{}}
	}
	if prompt.Title != "" {
		return root, []Effect{
			handledEffect{},
			FuncEffect{Run: func(context.Context) Msg { return ShellOpenPromptMsg{Prompt: prompt} }},
		}
	}
	return root, []Effect{
		handledEffect{},
		FuncEffect{Run: func(context.Context) Msg {
			return ShellWorkbenchCommandMsg{Command: command, OpenPickerAfterOK: command.Action == state.WorkbenchCommandTabCreate}
		}},
	}
}

func reduceViewWorkbenchShortcut(root state.Root, command string) (state.Root, []Effect, bool) {
	shell := root.Shell.EnsureDefaults()
	if command == "terminal size lock" {
		return root, []Effect{handledEffect{}, terminalSizeLockToggleEffect()}, true
	}
	if layoutCommand, ok := terminalViewLayoutCommandFromString(command); ok {
		next, effects := applyActiveTerminalViewLayoutCommand(root, layoutCommand)
		return next, append([]Effect{handledEffect{}}, effects...), true
	}
	switch command {
	case "pane take-owner":
		next, effects := requestPaneResizeOwner(root, shell.ActivePaneID)
		return next, append([]Effect{handledEffect{}}, effects...), true
	case "floating take-owner":
		activeFloatingID := shell.ActiveFloatingID()
		if activeFloatingID == "" {
			root.Shell = shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.owner", Body: "no active floating"})
			return root.Advance(), []Effect{handledEffect{}}, true
		}
		next, effects := requestFloatingResizeOwner(root, activeFloatingID)
		return next, append([]Effect{handledEffect{}}, effects...), true
	case "pane reconnect":
		ref := terminalRefForContentAction(root, shell.ActivePaneID)
		if ref.Empty() {
			root.Shell = shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "pane.reconnect", Body: "terminal unavailable"})
			return root.Advance(), []Effect{handledEffect{}}, true
		}
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolReconnectRequestMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, TargetPaneID: shell.ActivePaneID, LocalError: true}
		}}}, true
	case "pane restart":
		ref := terminalRefForContentAction(root, shell.ActivePaneID)
		if ref.Empty() {
			root.Shell = shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.restart", Body: "no active terminal"})
			return root.Advance(), []Effect{handledEffect{}}, true
		}
		return root, []Effect{handledEffect{}, FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolRestartRequestMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID}
		}}}, true
	default:
		return root, nil, false
	}
}

func terminalSizeLockToggleEffect() Effect {
	return FuncEffect{Run: func(context.Context) Msg { return TerminalSizeLockToggleRequestMsg{} }}
}

func applyActiveTerminalViewLayoutCommand(root state.Root, command state.TerminalViewLayoutCommand) (state.Root, []Effect) {
	shell := root.Shell.EnsureDefaults()
	var binding state.TerminalViewBinding
	var ok bool
	if activeFloatingID := shell.ActiveFloatingID(); activeFloatingID != "" {
		root.TerminalViews, binding, ok = root.TerminalViews.ApplyFloatingLayoutCommand(activeFloatingID, command)
	} else {
		root.TerminalViews, binding, ok = root.TerminalViews.ApplyPaneLayoutCommand(shell.ActivePaneID, command)
	}
	if !ok {
		root.Shell = shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.layout", Body: "no active view"})
		return root.Advance(), nil
	}
	root.Shell = shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "terminal.layout", Body: terminalViewLayoutToast(binding.Layout)})
	return root.Advance(), workbenchPersistEffects("terminal.layout")
}

func terminalViewLayoutCommandFromString(command string) (state.TerminalViewLayoutCommand, bool) {
	switch command {
	case "terminal layout lock":
		return state.TerminalViewLayoutCommand{Action: "toggle-lock"}, true
	case "terminal layout toggle":
		return state.TerminalViewLayoutCommand{Action: "toggle-layout"}, true
	case "terminal layout pan-left":
		return state.TerminalViewLayoutCommand{Action: "pan", DeltaX: -2}, true
	case "terminal layout pan-right":
		return state.TerminalViewLayoutCommand{Action: "pan", DeltaX: 2}, true
	case "terminal layout pan-up":
		return state.TerminalViewLayoutCommand{Action: "pan", DeltaY: -1}, true
	case "terminal layout pan-down":
		return state.TerminalViewLayoutCommand{Action: "pan", DeltaY: 1}, true
	case "terminal layout align-left":
		return state.TerminalViewLayoutCommand{Action: "align", AlignX: state.TerminalViewAlignStart}, true
	case "terminal layout align-right":
		return state.TerminalViewLayoutCommand{Action: "align", AlignX: state.TerminalViewAlignEnd}, true
	case "terminal layout align-top":
		return state.TerminalViewLayoutCommand{Action: "align", AlignY: state.TerminalViewAlignStart}, true
	case "terminal layout align-bottom":
		return state.TerminalViewLayoutCommand{Action: "align", AlignY: state.TerminalViewAlignEnd}, true
	case "terminal layout center":
		return state.TerminalViewLayoutCommand{Action: "center"}, true
	case "terminal layout center-x":
		return state.TerminalViewLayoutCommand{Action: "center-x"}, true
	case "terminal layout center-y":
		return state.TerminalViewLayoutCommand{Action: "center-y"}, true
	case "terminal layout reset":
		return state.TerminalViewLayoutCommand{Action: "reset"}, true
	default:
		return state.TerminalViewLayoutCommand{}, false
	}
}

func terminalViewLayoutToast(layout state.TerminalViewLayout) string {
	layout = layout.Normalize()
	lock := "unlocked"
	if layout.SizeLocked {
		lock = "locked"
	}
	return fmt.Sprintf("%s %s pan:%d,%d align:%s/%s", lock, layout.Mode, layout.PanX, layout.PanY, layout.AlignX, layout.AlignY)
}

func reduceShellActionIntent(root state.Root, intent input.Intent) (state.Root, []Effect) {
	var msg Msg
	switch intent.Action {
	case input.ShellActionToggleHeader:
		msg = ShellToggleHeaderVisibleMsg{}
	case input.ShellActionToggleFooter:
		msg = ShellToggleFooterVisibleMsg{}
	case input.ShellActionToggleShortcutLock:
		root.Shell = root.Shell.ToggleShortcutPassthroughLock()
		status := "off"
		if root.Shell.ShortcutPassthroughLocked {
			status = "on"
		}
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "shortcut lock", Body: status})
		return root.Advance(), []Effect{handledEffect{}}
	case input.ShellActionClearToasts:
		msg = ShellClearToastsMsg{}
	case input.ShellActionCloseToast:
		msg = ShellCloseCurrentToastMsg{}
	case input.ShellActionFloatingNew, input.ShellActionFloatingCtrl, input.ShellActionFloatingGroup, input.ShellActionFloatingMove, input.ShellActionFloatingSize:
		command, ok := floatingCommandFromIntent(root, intent)
		if !ok {
			return root, []Effect{handledEffect{}}
		}
		msg = ShellFloatingCommandMsg{Command: command}
	case input.ShellActionFloatingOverview:
		msg = ShellOpenFloatingOverviewMsg{}
	case input.ShellActionFloatingSummon:
		index, ok := floatingSummonIndex(intent.Reason)
		if !ok {
			return root, []Effect{handledEffect{}}
		}
		msg = ShellFloatingCommandMsg{Command: state.FloatingCommand{Action: state.FloatingCommandSummon, Index: index, Source: state.PaneCommandSourceKeyboard}}
	case input.ShellActionOpenPool:
		msg = ShellOpenTerminalPoolMsg{}
	case input.ShellActionOpenConnections:
		msg = ShellOpenConnectionsMsg{}
	case input.ShellActionOpenTree:
		msg = ShellOpenWorkbenchTreeMsg{}
	case input.ShellActionOpenClipboardHistory:
		msg = ShellOpenClipboardHistoryMsg{}
	case input.ShellActionOpenPicker:
		msg = ShellOpenTerminalPickerMsg{}
	case input.ShellActionOpenPrompt:
		msg = ShellOpenPromptMsg{Prompt: actionCommandPrompt()}
	case input.ShellActionOpenHelp:
		msg = ShellOpenHelpMsg{Section: "most-used"}
	case input.ShellActionQuit:
		msg = QuitMsg{}
	default:
		return root, []Effect{handledEffect{}}
	}
	return root, []Effect{
		handledEffect{},
		FuncEffect{Run: func(context.Context) Msg {
			return msg
		}},
	}
}

func actionCommandPrompt() state.PromptState {
	return state.PromptState{Title: "Command Prompt", Context: "Run a canonical AnyTTY action.", Purpose: "action.command", Placeholder: "command"}
}

func floatingSummonIndex(value string) (int, bool) {
	if len(value) != 1 || value[0] < '1' || value[0] > '9' {
		return 0, false
	}
	return int(value[0] - '1'), true
}

func selectedFloatingOverviewItem(items []state.FloatingOverviewItem) (state.FloatingOverviewItem, bool) {
	if len(items) == 0 {
		return state.FloatingOverviewItem{}, false
	}
	selected := items[0]
	for _, item := range items {
		if item.Selected {
			selected = item
			break
		}
	}
	return selected, selected.FloatingID != ""
}

func workbenchCommandFromIntent(root state.Root, intent input.Intent) (state.WorkbenchCommand, state.PromptState, bool) {
	shell := root.Shell.EnsureDefaults()
	command := state.WorkbenchCommand{Source: state.PaneCommandSourceKeyboard}
	if strings.HasPrefix(intent.Command, "tab jump ") {
		command.Action = state.WorkbenchCommandTabSwitch
		command.TargetID = tabJumpTargetID(shell, intent.Command)
		return command, state.PromptState{}, true
	}
	switch intent.Command {
	case "tab create":
		command.Action = state.WorkbenchCommandTabCreate
		command.Name = nextTabName(shell)
		return command, state.PromptState{}, true
	case "tab next":
		command.Action = state.WorkbenchCommandTabNext
		return command, state.PromptState{}, true
	case "tab previous":
		command.Action = state.WorkbenchCommandTabPrevious
		return command, state.PromptState{}, true
	case "tab rename":
		return command, tabRenamePrompt(shell), true
	case "tab close":
		command.Action = state.WorkbenchCommandTabClose
		return command, state.PromptState{}, true
	case "tab kill confirm=accepted":
		command.Action = state.WorkbenchCommandTabKill
		command.Confirm = state.PaneConfirmAccepted
		return command, state.PromptState{}, true
	case "workspace create":
		command.Action = state.WorkbenchCommandWorkspaceCreate
		command.Name = nextWorkspaceName(shell)
		return command, state.PromptState{}, true
	case "workspace next":
		command.Action = state.WorkbenchCommandWorkspaceNext
		return command, state.PromptState{}, true
	case "workspace previous":
		command.Action = state.WorkbenchCommandWorkspacePrevious
		return command, state.PromptState{}, true
	case "workspace rename":
		return command, workspaceRenamePrompt(shell), true
	case "workspace delete confirm=accepted":
		command.Action = state.WorkbenchCommandWorkspaceDelete
		command.Confirm = state.PaneConfirmAccepted
		return command, state.PromptState{}, true
	case "pane close":
		command.Action = state.WorkbenchCommandPaneClose
		command.Target = state.PaneCommandTarget{PaneID: shell.ActivePaneID}
		return command, state.PromptState{}, true
	case "pane detach":
		command.Action = state.WorkbenchCommandPaneDetach
		command.Target = state.PaneCommandTarget{PaneID: shell.ActivePaneID}
		return command, state.PromptState{}, true
	case "pane kill confirm=accepted":
		command.Action = state.WorkbenchCommandPaneKill
		command.Target = state.PaneCommandTarget{PaneID: shell.ActivePaneID}
		command.Confirm = state.PaneConfirmAccepted
		return command, state.PromptState{}, true
	default:
		return state.WorkbenchCommand{}, state.PromptState{}, false
	}
}

func tabJumpTargetID(shell state.ShellStore, command string) string {
	indexText := strings.TrimSpace(strings.TrimPrefix(command, "tab jump "))
	index, err := strconv.Atoi(indexText)
	if err != nil || index < 1 {
		return "__invalid_tab_jump__"
	}
	tabs := shell.EnsureDefaults().Workspace.Tabs
	if index > len(tabs) {
		return "__invalid_tab_jump__"
	}
	return tabs[index-1].ID
}

func floatingCommandFromIntent(root state.Root, intent input.Intent) (state.FloatingCommand, bool) {
	command := state.FloatingCommand{
		Source:  state.PaneCommandSourceKeyboard,
		BoundsW: root.Viewport.Cols,
		BoundsH: root.Viewport.Rows,
	}
	switch intent.Action {
	case input.ShellActionFloatingNew:
		command.Action = state.FloatingCommandCreate
		command.TargetID = nextFloatingID(root.Shell)
		command.Pane = state.PaneState{ID: nextFloatingPaneID(root.Shell), Title: "floating", Kind: state.PaneEmpty}
		command.Title = "floating"
		return command, true
	case input.ShellActionFloatingCtrl:
		switch intent.Reason {
		case "close":
			command.Action = state.FloatingCommandClose
		case "collapse":
			command.Action = state.FloatingCommandToggleCollapse
		case "center":
			command.Action = state.FloatingCommandCenter
		default:
			return state.FloatingCommand{}, false
		}
		return command, true
	case input.ShellActionFloatingMove:
		command.Action = state.FloatingCommandMove
		switch intent.Reason {
		case "left":
			command.DeltaX = -2
		case "right":
			command.DeltaX = 2
		case "up":
			command.DeltaY = -1
		case "down":
			command.DeltaY = 1
		default:
			return state.FloatingCommand{}, false
		}
		return command, true
	case input.ShellActionFloatingSize:
		command.Action = state.FloatingCommandResize
		switch intent.Reason {
		case "narrow":
			command.DeltaW = -2
		case "wide":
			command.DeltaW = 2
		case "short":
			command.DeltaH = -1
		case "tall":
			command.DeltaH = 1
		default:
			return state.FloatingCommand{}, false
		}
		return command, true
	case input.ShellActionFloatingGroup:
		switch intent.Reason {
		case "toggle-all":
			command.Action = state.FloatingCommandToggleAll
		case "fit":
			command.Action = state.FloatingCommandFit
		case "toggle-auto-fit":
			command.Action = state.FloatingCommandToggleAutoFit
		default:
			return state.FloatingCommand{}, false
		}
		return command, true
	default:
		return state.FloatingCommand{}, false
	}
}

func inputMode(mode state.InteractionMode) input.InteractionMode {
	switch mode {
	case state.InteractionModePane:
		return input.InteractionModePane
	case state.InteractionModeResize:
		return input.InteractionModeResize
	case state.InteractionModeGlobal:
		return input.InteractionModeGlobal
	case state.InteractionModeFloating:
		return input.InteractionModeFloating
	case state.InteractionModeTab:
		return input.InteractionModeTab
	case state.InteractionModeWorkspace:
		return input.InteractionModeWorkspace
	default:
		return input.InteractionModeNormal
	}
}

func stateInteractionMode(mode input.InteractionMode) state.InteractionMode {
	switch mode {
	case input.InteractionModePane:
		return state.InteractionModePane
	case input.InteractionModeResize:
		return state.InteractionModeResize
	case input.InteractionModeGlobal:
		return state.InteractionModeGlobal
	case input.InteractionModeFloating:
		return state.InteractionModeFloating
	case input.InteractionModeTab:
		return state.InteractionModeTab
	case input.InteractionModeWorkspace:
		return state.InteractionModeWorkspace
	default:
		return state.InteractionModeNormal
	}
}

func activeTabTitle(shell state.ShellStore) string {
	shell = shell.EnsureDefaults()
	for _, tab := range shell.Workspace.Tabs {
		if tab.ID == shell.Workspace.ActiveTabID {
			return tab.Title
		}
	}
	return "main"
}

func tabRenamePrompt(shell state.ShellStore) state.PromptState {
	return state.PromptState{
		Title:       "Rename Tab",
		Context:     "Rename current tab. Submit applies through workbench command.",
		Purpose:     "tab.rename",
		Value:       activeTabTitle(shell),
		Placeholder: "tab name",
	}
}

func nextTabName(shell state.ShellStore) string {
	shell = shell.EnsureDefaults()
	return fmt.Sprintf("tab %d", len(shell.Workspace.Tabs)+1)
}

func nextWorkspaceName(shell state.ShellStore) string {
	shell = shell.EnsureDefaults()
	return fmt.Sprintf("workspace %d", len(shell.Workspaces)+1)
}

func workspaceRenamePrompt(shell state.ShellStore) state.PromptState {
	shell = shell.EnsureDefaults()
	return state.PromptState{
		Title:       "Rename Workspace",
		Context:     "Rename current workspace. Submit applies through workbench command.",
		Purpose:     "workspace.rename",
		Value:       shell.Workspace.Name,
		Placeholder: "workspace name",
	}
}

func nextKeyboardPaneID(shell state.ShellStore) string {
	shell = shell.EnsureDefaults()
	for i := 2; ; i++ {
		id := fmt.Sprintf("pane-%d", i)
		if !shell.HasPane(state.PaneCommandTarget{PaneID: id}) {
			return id
		}
	}
}

func nextFloatingPaneID(shell state.ShellStore) string {
	shell = shell.EnsureDefaults()
	for i := 1; ; i++ {
		id := fmt.Sprintf("floating-pane-%d", i)
		exists := false
		for _, floating := range shell.ActiveFloatings() {
			if floating.Pane.ID == id {
				exists = true
				break
			}
		}
		if !exists {
			return id
		}
	}
}

func nextFloatingID(shell state.ShellStore) string {
	shell = shell.EnsureDefaults()
	for i := 1; ; i++ {
		id := fmt.Sprintf("floating-%d", i)
		exists := false
		for _, floating := range shell.ActiveFloatings() {
			if floating.ID == id {
				exists = true
				break
			}
		}
		if !exists {
			return id
		}
	}
}

package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

type ShellSetPanelPresentationMsg struct {
	Presentation state.PanelPresentation
}

func (ShellSetPanelPresentationMsg) isMsg() {}

type ShellTogglePanelPresentationMsg struct{}

func (ShellTogglePanelPresentationMsg) isMsg() {}

type ShellSetHeaderVisibleMsg struct {
	Visible bool
}

func (ShellSetHeaderVisibleMsg) isMsg() {}

type ShellToggleHeaderVisibleMsg struct{}

func (ShellToggleHeaderVisibleMsg) isMsg() {}

type ShellSetFooterVisibleMsg struct {
	Visible bool
}

func (ShellSetFooterVisibleMsg) isMsg() {}

type ShellToggleFooterVisibleMsg struct{}

func (ShellToggleFooterVisibleMsg) isMsg() {}

type ShellAddToastMsg struct {
	Toast state.ToastSpec
}

func (ShellAddToastMsg) isMsg() {}

type ShellTickToastsMsg struct {
	Ticks uint64
}

func (ShellTickToastsMsg) isMsg() {}

type ShellCloseCurrentToastMsg struct{}

func (ShellCloseCurrentToastMsg) isMsg() {}

type ShellClearToastsMsg struct{}

func (ShellClearToastsMsg) isMsg() {}

type ShellOpenTerminalPickerMsg struct{}

func (ShellOpenTerminalPickerMsg) isMsg() {}

type ShellOpenTerminalPoolMsg struct{}

func (ShellOpenTerminalPoolMsg) isMsg() {}

type ShellOpenWorkbenchTreeMsg struct{}

func (ShellOpenWorkbenchTreeMsg) isMsg() {}

type ShellOpenFloatingOverviewMsg struct{}

func (ShellOpenFloatingOverviewMsg) isMsg() {}

type ShellOpenPromptMsg struct {
	Prompt state.PromptState
}

func (ShellOpenPromptMsg) isMsg() {}

type ShellOpenHelpMsg struct {
	Section string
}

type ShellOpenClipboardHistoryMsg struct{}

func (ShellOpenClipboardHistoryMsg) isMsg() {}

func (ShellOpenHelpMsg) isMsg() {}

type ShellCloseOverlayMsg struct{}

func (ShellCloseOverlayMsg) isMsg() {}

type ShellPromptSetValueMsg struct {
	Value string
}

func (ShellPromptSetValueMsg) isMsg() {}

type ShellPromptSubmitMsg struct{}

func (ShellPromptSubmitMsg) isMsg() {}

type ShellPromptCancelMsg struct{}

func (ShellPromptCancelMsg) isMsg() {}

// ShortcutSurfaceContext 描述鼠标、drag 或内容 CTA 提供的显式 surface 目标。
// 非 nil 指针同时标记消息来源为 surface；PaneID/Floating/Row 只携带命中上下文，
// 缺失或无效目标必须 fail closed，不能回退到 keyboard 的 active target。
type ShortcutSurfaceContext struct {
	// ExplicitTarget 区分 pane chrome/CTA 的强目标语义与 footer 的 active-target 语义。
	// 为 true 时目标无效必须终止；为 false 时 action 可按 keyboard 规则使用 active target。
	ExplicitTarget bool
	PaneID         string
	Floating       bool
	Row            int
	HasRow         bool
}

// ShellShortcutActionMsg 让键盘和 surface 共享同一个 canonical app action dispatcher。
// Invocation 是执行身份；Surface 为 nil 表示 keyboard/command 来源，非 nil 表示必须遵守显式目标失败语义。
type ShellShortcutActionMsg struct {
	Invocation actiondomain.Invocation
	Surface    *ShortcutSurfaceContext
}

func (ShellShortcutActionMsg) isMsg() {}

type ShellMoveClipboardHistoryDividerMsg struct {
	Delta int
}

func (ShellMoveClipboardHistoryDividerMsg) isMsg() {}

type ShellActivateTerminalInputMsg struct {
	PaneID     string
	FloatingID string
}

func (ShellActivateTerminalInputMsg) isMsg() {}

type ShellOverlayMouseSelectMsg struct {
	Delta int
}

func (ShellOverlayMouseSelectMsg) isMsg() {}

type ShellArmOwnerConfirmMsg struct {
	ViewID string
}

func (ShellArmOwnerConfirmMsg) isMsg() {}

type ShellClearOwnerConfirmMsg struct {
	Seq uint64
}

func (ShellClearOwnerConfirmMsg) isMsg() {}

type HostThemeMsg struct {
	Update state.HostThemeUpdate
}

func (HostThemeMsg) isMsg() {}

// HostCapabilityMsg 把 TerminalHost 已确认的宿主能力回投 reducer-owned state。
type HostCapabilityMsg struct {
	Update state.HostCapabilityUpdate
}

func (HostCapabilityMsg) isMsg() {}

const ownerConfirmDelay = 500 * time.Millisecond

const ownerConfirmClearToken CancelToken = "terminal.owner.confirm.clear"

type ShellSplitActivePaneMsg struct {
	Pane      state.PaneState
	Direction state.SplitDirection
}

func (ShellSplitActivePaneMsg) isMsg() {}

type ShellPaneCommandMsg struct {
	Command state.PaneCommand
}

func (ShellPaneCommandMsg) isMsg() {}

type ShellClosePaneIfTerminalRefMsg struct {
	PaneID      string
	ExpectedRef state.TerminalRef
}

func (ShellClosePaneIfTerminalRefMsg) isMsg() {}

type ShellCloseFloatingIfTerminalRefMsg struct {
	FloatingID  string
	ExpectedRef state.TerminalRef
}

func (ShellCloseFloatingIfTerminalRefMsg) isMsg() {}

type ShellFloatingCommandMsg struct {
	Command state.FloatingCommand
}

func (ShellFloatingCommandMsg) isMsg() {}

type ShellWorkbenchCommandMsg struct {
	Command           state.WorkbenchCommand
	OpenPickerAfterOK bool
}

func (ShellWorkbenchCommandMsg) isMsg() {}

type PaneCommandFeedbackEffect struct {
	Result  state.PaneCommandResult
	Command state.PaneCommand
}

func (PaneCommandFeedbackEffect) isEffect() {}

func NewShellReducer() Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		switch msg := msg.(type) {
		case ShellSetPanelPresentationMsg:
			return reducePaneCommand(root, state.PaneCommand{
				Action:       state.PaneCommandSetPresentation,
				Presentation: msg.Presentation,
				Source:       state.PaneCommandSourceTest,
			})
		case ShellTogglePanelPresentationMsg:
			return reducePaneCommand(root, state.PaneCommand{
				Action: state.PaneCommandTogglePresentation,
				Source: state.PaneCommandSourceTest,
			})
		case ShellSetHeaderVisibleMsg:
			root.Shell = root.Shell.SetHeaderVisible(msg.Visible)
		case ShellToggleHeaderVisibleMsg:
			root.Shell = root.Shell.ToggleHeaderVisible()
		case ShellSetFooterVisibleMsg:
			root.Shell = root.Shell.SetFooterVisible(msg.Visible)
		case ShellToggleFooterVisibleMsg:
			root.Shell = root.Shell.ToggleFooterVisible()
		case ShellAddToastMsg:
			root.Shell = root.Shell.AddToast(msg.Toast)
		case ShellTickToastsMsg:
			root.Shell = root.Shell.TickToasts(msg.Ticks)
		case TickMsg:
			if msg.Token != "" && msg.Token != toastTickToken {
				return root, nil
			}
			if len(root.Shell.Toasts) == 0 {
				return root, nil
			}
			ticks := msg.Ticks
			if ticks == 0 {
				ticks = 1
			}
			root.Shell = root.Shell.TickToasts(ticks)
		case ShellCloseCurrentToastMsg:
			root.Shell = root.Shell.CloseCurrentToast()
		case ShellClearToastsMsg:
			root.Shell = root.Shell.ClearToasts()
		case ShellOpenTerminalPickerMsg:
			root.Shell = root.Shell.OpenTerminalPicker()
			return root.Advance(), []Effect{terminalPickerListRequestEffect()}
		case ShellOpenTerminalPoolMsg:
			root.Shell = root.Shell.OpenTerminalPool()
			return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg { return TerminalPoolListRequestMsg{} }}}
		case ShellOpenConnectionsMsg:
			root.Shell = root.Shell.OpenConnections()
			return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg { return ConnectionsLoadRequestMsg{} }}}
		case ShellOpenWorkbenchTreeMsg:
			root.Shell = openWorkbenchTreeAtActivePane(root)
		case ShellOpenClipboardHistoryMsg:
			root.Shell = root.Shell.OpenClipboardHistory()
			return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg {
				return ClipboardStorageLoadRequestMsg{Reason: "open"}
			}}}
		case ShellOpenFloatingOverviewMsg:
			root.Shell = root.Shell.OpenFloatingOverview()
		case ShellOpenPromptMsg:
			root.Shell = root.Shell.OpenPrompt(msg.Prompt)
			if msg.Prompt.Purpose == "terminal.create" {
				endpointID := currentCreatePromptEndpoint(root)
				if endpointID != "" {
					return root.Advance(), endpointDefaultsRequestEffect(endpointID, false)
				}
			}
		case ShellOpenHelpMsg:
			root.Shell = root.Shell.OpenHelp(msg.Section)
		case ShellCloseOverlayMsg:
			root.Shell = root.Shell.CloseOverlay()
		case ShellPromptSetValueMsg:
			root.Shell = root.Shell.SetPromptValue(msg.Value)
		case ShellPromptSubmitMsg:
			return reducePromptSubmit(root)
		case ShellPromptCancelMsg:
			root.Shell = root.Shell.CancelPrompt().CloseOverlay()
			root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "prompt.cancel", Body: "canceled"})
		case ShellShortcutActionMsg:
			row := -1
			if msg.Surface != nil {
				if msg.Surface.HasRow {
					row = msg.Surface.Row
				}
				if msg.Surface.ExplicitTarget {
					if next, effects, handled := reduceCanonicalSurfaceAction(root, msg); handled {
						return rearmInteractionModeTimeout(next, effects)
					}
				}
			}
			if next, effects, handled := reduceActiveSurfaceShortcut(root, msg.Invocation); handled {
				return rearmInteractionModeTimeout(next, effects)
			}
			intent, ok := shortcutIntentForInvocation(msg.Invocation, input.InputEvent{})
			if !ok {
				return root, nil
			}
			if shortcutIntentOwnedByCopy(intent) {
				return root, nil
			}
			next, effects := reduceShortcutIntentWithContext(root, intent, row)
			if intent.Kind == input.IntentOpenTerminalPicker || intent.Kind == input.IntentSetInteractionMode {
				return next, effects
			}
			return finishInteractionModeAfterIntent(next, effects, intent)
		case ShellMoveClipboardHistoryDividerMsg:
			root.Shell = root.Shell.MoveClipboardHistoryNameWidth(msg.Delta)
		case ShellActivateTerminalInputMsg:
			return reduceShellActivateTerminalInput(root, msg)
		case ShellArmOwnerConfirmMsg:
			root.Shell = root.Shell.ArmOwnerConfirm(msg.ViewID)
			seq := root.Shell.OwnerConfirm.Seq
			return root.Advance(), []Effect{ownerConfirmClearEffect(seq)}
		case ShellClearOwnerConfirmMsg:
			root.Shell = root.Shell.ClearOwnerConfirm(msg.Seq)
		case HostThemeMsg:
			root.HostTheme = root.HostTheme.ApplyUpdate(msg.Update)
		case HostCapabilityMsg:
			root.HostCapabilities = root.HostCapabilities.ApplyUpdate(msg.Update)
		case ShellSplitActivePaneMsg:
			return reduceWorkbenchCommand(root, state.WorkbenchCommand{
				Action: state.WorkbenchCommandPaneSplit,
				Pane: state.PaneCommand{
					Action:         state.PaneCommandSplit,
					SplitDirection: msg.Direction,
					NewPane:        msg.Pane,
					Source:         state.PaneCommandSourceTest,
				},
				Source: state.PaneCommandSourceTest,
			})
		case ShellPaneCommandMsg:
			return reducePaneCommand(root, msg.Command)
		case ShellClosePaneIfTerminalRefMsg:
			return reduceClosePaneIfTerminalRef(root, msg)
		case ShellCloseFloatingIfTerminalRefMsg:
			return reduceCloseFloatingIfTerminalRef(root, msg)
		case ShellFloatingCommandMsg:
			return reduceFloatingCommand(root, msg.Command)
		case ShellWorkbenchCommandMsg:
			return reduceWorkbenchCommandWithOptions(root, msg.Command, workbenchCommandOptions{OpenPickerAfterOK: msg.OpenPickerAfterOK})
		default:
			return root, nil
		}
		return root.Advance(), nil
	}
}

func reduceClosePaneIfTerminalRef(root state.Root, msg ShellClosePaneIfTerminalRefMsg) (state.Root, []Effect) {
	if !paneStillOwnsTerminalRef(root, msg.PaneID, msg.ExpectedRef) {
		return root, nil
	}
	return reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandPaneClose, Target: state.PaneCommandTarget{PaneID: msg.PaneID}, Source: state.PaneCommandSourceKeyboard})
}

func reduceCloseFloatingIfTerminalRef(root state.Root, msg ShellCloseFloatingIfTerminalRefMsg) (state.Root, []Effect) {
	if !floatingStillOwnsTerminalRef(root, msg.FloatingID, msg.ExpectedRef) {
		return root, nil
	}
	return reduceFloatingCommand(root, state.FloatingCommand{Action: state.FloatingCommandClose, TargetID: msg.FloatingID, Source: state.PaneCommandSourceKeyboard})
}

func ownerConfirmClearEffect(seq uint64) Effect {
	return FuncEffect{
		Token: ownerConfirmClearToken,
		Async: true,
		Run: func(ctx context.Context) Msg {
			// owner? 是临时确认提示，超时后必须退回 authoritative follower 展示。
			timer := time.NewTimer(ownerConfirmDelay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return nil
			case <-timer.C:
				return ShellClearOwnerConfirmMsg{Seq: seq}
			}
		},
	}
}

func openWorkbenchTreeAtActivePane(root state.Root) state.ShellStore {
	shell := root.Shell.OpenWorkbenchTree()
	activePaneID := shell.ActivePaneID
	if activePaneID == "" {
		return shell
	}
	root.Shell = shell
	items := state.WorkbenchTreeItems(root)
	for index, item := range items {
		if item.Kind == state.WorkbenchTreeKindPane && item.PaneID == activePaneID {
			return shell.SetWorkbenchTreeSelectedIndex(index, len(items))
		}
	}
	return shell
}

func reduceShellActivateTerminalInput(root state.Root, msg ShellActivateTerminalInputMsg) (state.Root, []Effect) {
	shell := root.Shell.EnsureDefaults()
	if msg.FloatingID != "" {
		command := state.FloatingCommand{
			Action:   state.FloatingCommandFocusRaise,
			TargetID: msg.FloatingID,
			Source:   state.PaneCommandSourceMouse,
		}
		nextShell, result := shell.ApplyFloatingCommand(command)
		root.Shell = addFloatingCommandToast(nextShell, command, result)
		if result.Status == state.FloatingCommandOK {
			// 点击 terminal 内容代表回到输入态，后续键盘输入必须直达 active TerminalView。
			root.Shell = root.Shell.ExitInteractionMode()
		}
		return root.Advance(), nil
	}
	if msg.PaneID == "" {
		root.Shell = shell.ExitInteractionMode()
		return root.Advance(), nil
	}
	command := state.PaneCommand{
		Action: state.PaneCommandFocus,
		Target: state.PaneCommandTarget{
			PaneID: msg.PaneID,
		},
		Source: state.PaneCommandSourceMouse,
	}
	command = command.WithDefaults(shell)
	targetPane, hasTargetPane := shell.Pane(command.Target)
	nextShell, result := shell.ApplyPaneCommand(command)
	if result.Status == state.PaneCommandOK {
		root.Shell = deactivateFloatingAfterPaneCommand(nextShell, command).ExitInteractionMode()
		root = updateTerminalViewsAfterPaneCommand(root, command, targetPane, hasTargetPane)
		root.Shell = addPaneCommandToast(root.Shell, command, result)
		return root.Advance(), paneCommandEffects(root, command, result, targetPane, hasTargetPane)
	}
	root.Shell = addPaneCommandToast(shell, command, result)
	return root.Advance(), []Effect{PaneCommandFeedbackEffect{Result: result, Command: command}}
}

func requestPaneResizeOwner(root state.Root, paneID string) (state.Root, []Effect) {
	if paneID == "" {
		return root, nil
	}
	binding, ok := root.TerminalViews.PaneBinding(paneID)
	if !ok || binding.TerminalID == "" {
		root.Shell = root.Shell.EnsureDefaults().AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.owner", Body: "no terminal view"})
		return root.Advance(), nil
	}
	if binding.Channel != 0 {
		cols := binding.DesiredCols
		rows := binding.DesiredRows
		if rect, ok := terminalViewContentRect(root, render.Rect{}, binding); ok {
			cols = rect.W
			rows = rect.H
		}
		var decision state.TerminalViewResizeDecision
		root.TerminalViews, decision = root.TerminalViews.RequestViewResizeOwner(binding.ViewID, cols, rows)
		if !decision.Allowed {
			return root, nil
		}
		seq := decision.Seq
		return root, []Effect{FuncEffect{Run: func(context.Context) Msg {
			return LiveResizeMsg{EndpointID: binding.EndpointID, TerminalID: binding.TerminalID, Cols: cols, Rows: rows, Seq: seq, ViewID: binding.ViewID, TakeOwnership: true, ExpectedOwnerEpoch: binding.ResizeEpoch}
		}}}
	}
	return root, []Effect{FuncEffect{Run: func(context.Context) Msg {
		return liveAttachMsgForResizeOwner(root, binding)
	}}}
}

func requestFloatingResizeOwner(root state.Root, floatingID string) (state.Root, []Effect) {
	binding, ok := root.TerminalViews.FloatingBinding(floatingID)
	if !ok || binding.TerminalID == "" {
		root.Shell = root.Shell.EnsureDefaults().AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.owner", Body: "no terminal view"})
		return root.Advance(), nil
	}
	if binding.Channel != 0 {
		cols := binding.DesiredCols
		rows := binding.DesiredRows
		if rect, ok := terminalViewContentRect(root, render.Rect{}, binding); ok {
			cols = rect.W
			rows = rect.H
		}
		var decision state.TerminalViewResizeDecision
		root.TerminalViews, decision = root.TerminalViews.RequestViewResizeOwner(binding.ViewID, cols, rows)
		if !decision.Allowed {
			return root, nil
		}
		seq := decision.Seq
		return root, []Effect{FuncEffect{Run: func(context.Context) Msg {
			return LiveResizeMsg{EndpointID: binding.EndpointID, TerminalID: binding.TerminalID, Cols: cols, Rows: rows, Seq: seq, ViewID: binding.ViewID, TakeOwnership: true, ExpectedOwnerEpoch: binding.ResizeEpoch}
		}}}
	}
	return root, []Effect{FuncEffect{Run: func(context.Context) Msg {
		return liveAttachMsgForResizeOwner(root, binding)
	}}}
}

func requestFloatingResizeOwnerWithConfirm(root state.Root, floatingID string) (state.Root, []Effect) {
	binding, ok := root.TerminalViews.FloatingBinding(floatingID)
	if !ok || binding.TerminalID == "" {
		root.Shell = root.Shell.EnsureDefaults().AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.owner", Body: "no terminal view"})
		return root.Advance(), nil
	}
	if binding.ViewID != "" && !binding.HasResizeOwner() && root.Shell.ReadonlyDefaults().OwnerConfirm.ViewID != binding.ViewID {
		// 中文说明：浮动 chrome 的 owner token 走 content action，不经过 pane chrome 的鼠标双击拦截；
		// 首击只设置 UI 确认态，terminal owner truth 必须等第二次确认后才交给 requestFloatingResizeOwner。
		root.Shell = root.Shell.ArmOwnerConfirm(binding.ViewID)
		seq := root.Shell.OwnerConfirm.Seq
		return root.Advance(), []Effect{ownerConfirmClearEffect(seq)}
	}
	root.Shell = root.Shell.ClearOwnerConfirm(0)
	return requestFloatingResizeOwner(root, floatingID)
}

func liveAttachMsgForResizeOwner(root state.Root, binding state.TerminalViewBinding) LiveAttachMsg {
	return LiveAttachMsg{Config: LiveConfig{
		EndpointID:   binding.EndpointID,
		TerminalID:   binding.TerminalID,
		Cols:         binding.DesiredCols,
		Rows:         binding.DesiredRows,
		ResizePolicy: state.TerminalResizeRoleOwner,
		SurfaceID:    runtimeSurfaceID(root),
		ViewID:       binding.ViewID,
	}}
}

func reducePromptSubmit(root state.Root) (state.Root, []Effect) {
	shell := root.Shell.EnsureDefaults()
	if shell.Overlay.Kind != state.OverlayPrompt || !shell.Overlay.Open {
		return root, nil
	}
	before := shell.Overlay.Prompt
	shell = shell.SubmitPrompt()
	after := shell.Overlay.Prompt
	root.Shell = shell
	if after.Submitted {
		body := after.LastResult
		if body == "" {
			body = "(empty)"
		}
		if after.Purpose == "terminal.create" {
			request, err := terminalCreateRequestFromPrompt(root, after)
			if err != nil {
				shell = root.Shell.EnsureDefaults()
				prompt := shell.Overlay.Prompt
				prompt.Submitted = false
				prompt.LastResult = err.Error()
				shell.Overlay.Prompt = prompt
				root.Shell = shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.create", Body: err.Error()})
				return root.Advance(), nil
			}
			root.Shell = rememberTerminalCreateDraft(root.Shell, after, request)
			root.Shell = root.Shell.CloseOverlay()
			return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg { return request }}}
		}
		if after.Purpose == "connections.settings" {
			request, err := connectionSettingsRequestFromPrompt(root, after)
			if err != nil {
				shell = root.Shell.EnsureDefaults()
				prompt := shell.Overlay.Prompt
				prompt.Submitted = false
				prompt.LastResult = err.Error()
				shell.Overlay.Prompt = prompt
				root.Shell = shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "Connection settings", Body: err.Error()})
				return root.Advance(), nil
			}
			return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg { return request }}}
		}
		if after.Purpose == "terminal.rename" {
			request, err := terminalEditRequestFromPrompt(after)
			if err != nil {
				shell = root.Shell.EnsureDefaults()
				prompt := shell.Overlay.Prompt
				prompt.Submitted = false
				prompt.LastResult = err.Error()
				shell.Overlay.Prompt = prompt
				root.Shell = shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.rename", Body: err.Error()})
				return root.Advance(), nil
			}
			root.Shell = root.Shell.OpenTerminalPool()
			return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg { return request }}}
		}
		if after.Purpose == "action.command" {
			invocation, _, err := actiondomain.ParseInvocation(after.LastResult)
			if err == nil {
				if !shortcutInvocationHasHandler(invocation) {
					err = fmt.Errorf("action %q has no executable handler", after.LastResult)
				} else if !shortcutInvocationAvailableFromCommand(invocation) {
					err = fmt.Errorf("action %q is not available from command prompt", after.LastResult)
				}
			}
			if err != nil {
				shell = root.Shell.EnsureDefaults()
				prompt := shell.Overlay.Prompt
				prompt.Submitted = false
				prompt.LastResult = err.Error()
				shell.Overlay.Prompt = prompt
				root.Shell = shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "action.command", Body: err.Error()})
				return root.Advance(), nil
			}
			root.Shell = root.Shell.CloseOverlay()
			return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg {
				return ShellShortcutActionMsg{Invocation: invocation}
			}}}
		}
		if after.Purpose == "clipboard.edit" {
			root.Clipboard = root.Clipboard.ReplaceEntryText(after.TargetID, after.LastResult)
			root.Shell = root.Shell.CloseOverlay()
			return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg {
				return ClipboardStoragePersistRequestMsg{Reason: "edit"}
			}}}
		}
		if after.Purpose == "clipboard.new" {
			root.Clipboard, _ = root.Clipboard.WithCopiedTextLimit(after.LastResult, clipboardHistoryMaxItems(root))
			root.Shell = root.Shell.CloseOverlay()
			return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg {
				return ClipboardStoragePersistRequestMsg{Reason: "new"}
			}}}
		}
		root.Shell = root.Shell.CloseOverlay()
		if command, ok := promptWorkbenchCommand(after); ok {
			return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg {
				return ShellWorkbenchCommandMsg{Command: command}
			}}}
		}
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "prompt.submit", Body: body})
		return root.Advance(), nil
	}
	if after.LastResult != "" && after.LastResult != before.LastResult {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "prompt.confirm", Body: after.LastResult})
	}
	return root.Advance(), nil
}

func clipboardHistoryMaxItems(root state.Root) int {
	if root.Config.Interaction.ClipboardHistory.MaxItems > 0 {
		return root.Config.Interaction.ClipboardHistory.MaxItems
	}
	return state.DefaultClipboardHistoryMaxItems
}

func promptWorkbenchCommand(prompt state.PromptState) (state.WorkbenchCommand, bool) {
	name := prompt.LastResult
	if name == "" {
		name = prompt.Value
	}
	switch prompt.Purpose {
	case "tab.rename":
		return state.WorkbenchCommand{Action: state.WorkbenchCommandTabRename, TargetID: prompt.TargetID, Target: state.PaneCommandTarget{WorkspaceID: prompt.TargetWorkspaceID}, Name: name, Source: state.PaneCommandSourcePalette}, true
	case "workspace.rename":
		return state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceRename, TargetID: prompt.TargetID, Name: name, Source: state.PaneCommandSourcePalette}, true
	case "pane.rename":
		return state.WorkbenchCommand{Action: state.WorkbenchCommandPaneRename, Target: state.PaneCommandTarget{WorkspaceID: prompt.TargetWorkspaceID, TabID: prompt.TargetTabID, PaneID: prompt.TargetID}, Name: name, Source: state.PaneCommandSourcePalette}, true
	default:
		return state.WorkbenchCommand{}, false
	}
}

func reduceClipboardHistoryPaste(root state.Root) (state.Root, []Effect) {
	items := state.ClipboardHistoryItems(root)
	if len(items) == 0 {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "clipboard history", Body: "no clipboard entry"})
		return root.Advance(), nil
	}
	selected := items[0]
	for _, item := range items {
		if item.Selected {
			selected = item
			break
		}
	}
	root.Shell = root.Shell.CloseOverlay()
	return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg {
		return ClipboardPasteTextMsg{Text: selected.Text}
	}}}
}

func reduceClipboardHistoryEdit(root state.Root) (state.Root, []Effect) {
	items := state.ClipboardHistoryItems(root)
	if len(items) == 0 {
		return root, nil
	}
	selected := items[0]
	for _, item := range items {
		if item.Selected {
			selected = item
			break
		}
	}
	root.Shell = root.Shell.OpenPrompt(state.PromptState{
		Title:       "Edit Clipboard Entry",
		Context:     "Edit selected clipboard entry. Submit updates clipboard history storage.",
		Purpose:     "clipboard.edit",
		TargetID:    selected.ID,
		Value:       selected.Text,
		Placeholder: "clipboard text",
	})
	return root.Advance(), nil
}

func reduceClipboardHistoryNew(root state.Root) (state.Root, []Effect) {
	root.Shell = root.Shell.OpenPrompt(state.PromptState{
		Title:       "New Clipboard Entry",
		Context:     "Create a clipboard entry. Submit updates clipboard history storage.",
		Purpose:     "clipboard.new",
		Placeholder: "clipboard text",
	})
	return root.Advance(), nil
}

func reduceClipboardHistoryDelete(root state.Root) (state.Root, []Effect) {
	items := state.ClipboardHistoryItems(root)
	if len(items) == 0 {
		return root, nil
	}
	selected := items[0]
	for _, item := range items {
		if item.Selected {
			selected = item
			break
		}
	}
	root.Clipboard = root.Clipboard.DeleteEntry(selected.ID, selected.Text)
	root.Shell = root.Shell.SetClipboardHistorySelectedIndex(0, len(state.ClipboardHistoryItems(root)))
	return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg {
		return ClipboardStoragePersistRequestMsg{Reason: "delete"}
	}}}
}

func createTerminalPrompt(targetPaneID string) state.PromptState {
	return createTerminalPromptWithEndpoint(state.Root{}, targetPaneID, state.DefaultEndpointID)
}

func createTerminalPromptWithEndpoint(root state.Root, targetPaneID string, endpointID state.EndpointID) state.PromptState {
	endpointID = state.NormalizeEndpointID(endpointID)
	endpointValue := terminalCreateEndpointPromptValue(root, endpointID)
	endpointCursor := len([]rune(endpointValue))
	workdirValue := terminalCreateDefaultWorkdirForEndpoint(root, endpointID)
	defaultCommand := terminalCreateDefaultCommandForEndpoint(root, endpointID)
	commandPlaceholder := terminalCreateCommandDisplay(defaultCommand)
	commandValue := ""
	if draft := root.Shell.ReadonlyDefaults().TerminalCreateDraft; state.NormalizeEndpointID(draft.EndpointID) == endpointID {
		if strings.TrimSpace(draft.Workdir) != "" {
			workdirValue = strings.TrimSpace(draft.Workdir)
		}
		if strings.TrimSpace(draft.Command) != "" && strings.TrimSpace(draft.Command) != commandPlaceholder {
			commandValue = strings.TrimSpace(draft.Command)
		}
	}
	return state.PromptState{
		Title:            "Create Terminal",
		Purpose:          "terminal.create",
		TargetEndpointID: endpointID,
		TargetID:         targetPaneID,
		Command:          defaultCommand,
		Workdir:          workdirValue,
		DefaultName:      terminalCreateDefaultName(defaultCommand),
		Fields: []state.PromptFieldState{
			{Key: "name", Label: "name", Required: true},
			{Key: "command", Label: "command", Value: commandValue, Cursor: len([]rune(commandValue)), Placeholder: commandPlaceholder},
			{Key: "server", Label: "server", Value: endpointValue, Cursor: endpointCursor, Placeholder: "endpoint id or label", Required: true},
			{Key: "workdir", Label: "workdir", Value: workdirValue, Cursor: len([]rune(workdirValue)), Placeholder: "endpoint cwd"},
		},
	}
}

func terminalCreateDefaultCommandForEndpoint(root state.Root, endpointID state.EndpointID) []string {
	endpoint, ok := root.Endpoints.DisplayEndpoint(endpointID)
	if !ok || !endpoint.DefaultsLoaded {
		return nil
	}
	return append([]string(nil), endpoint.DefaultCommand...)
}

func terminalCreateCommandDisplay(command []string) string {
	return strings.Join(command, " ")
}

func terminalCreateDefaultName(command []string) string {
	if len(command) == 0 || strings.TrimSpace(command[0]) == "" {
		return "sh"
	}
	return filepath.Base(command[0])
}

func createTerminalPromptForTarget(root state.Root, target terminalPoolTarget) state.PromptState {
	return createTerminalPromptForTargetEndpoint(root, target, "")
}

func createTerminalPromptForTargetEndpoint(root state.Root, target terminalPoolTarget, endpointID state.EndpointID) state.PromptState {
	if target.PaneID == "" && target.FloatingID == "" {
		target = terminalPoolTargetForActive(root)
	}
	endpointID = terminalCreateDefaultEndpointID(root, target, endpointID)
	if target.FloatingID != "" {
		prompt := createTerminalPromptWithEndpoint(root, target.PaneID, endpointID)
		prompt.Context = "floating"
		prompt.TargetID = target.FloatingID
		return prompt
	}
	if target.PaneID == "" {
		target = terminalPoolTargetForActive(root)
	}
	prompt := createTerminalPromptWithEndpoint(root, target.PaneID, endpointID)
	prompt.Context = "pane"
	prompt.TargetID = target.PaneID
	return prompt
}

func terminalCreateRequestFromPrompt(root state.Root, prompt state.PromptState) (TerminalPoolCreateRequestMsg, error) {
	name := strings.TrimSpace(prompt.FieldValue("name"))
	if name == "" {
		return TerminalPoolCreateRequestMsg{}, fmt.Errorf("name is required")
	}
	endpointID, err := terminalCreateEndpointIDFromPrompt(root, prompt)
	if err != nil {
		return TerminalPoolCreateRequestMsg{}, err
	}
	command, err := parsePromptCommand(prompt.FieldValue("command"))
	if err != nil {
		return TerminalPoolCreateRequestMsg{}, err
	}
	if len(command) == 0 {
		// 中文说明：空 command 表示使用目标 endpoint 默认 shell；提交时重新按 endpoint 解析，
		// 避免用户切换 server 后把其它 endpoint 的默认值发送到 owning daemon。
		command = terminalCreateDefaultCommandForEndpoint(root, endpointID)
	}
	if len(command) == 0 {
		return TerminalPoolCreateRequestMsg{}, terminalCreateDefaultsError(root, endpointID)
	}
	workdir := strings.TrimSpace(prompt.FieldValue("workdir"))
	if workdir == "" {
		workdir = terminalCreateDefaultWorkdirForEndpoint(root, endpointID)
	} else if state.NormalizeEndpointID(prompt.TargetEndpointID) != endpointID &&
		workdir == terminalCreateDefaultWorkdirForEndpoint(root, prompt.TargetEndpointID) {
		// 中文说明：server 字段可能被手写，未经过 refreshPromptCompletions；
		// 提交边界仍要防止把旧 endpoint 自动 cwd 发送到新的 owning daemon。
		workdir = terminalCreateDefaultWorkdirForEndpoint(root, endpointID)
	}
	request := TerminalPoolCreateRequestMsg{EndpointID: endpointID, Title: name, Command: command, CWD: workdir}
	if prompt.Context == "floating" {
		request.TargetFloatingID = prompt.TargetID
	} else {
		request.TargetPaneID = prompt.TargetID
	}
	return request, nil
}

func rememberTerminalCreateDraft(shell state.ShellStore, prompt state.PromptState, request TerminalPoolCreateRequestMsg) state.ShellStore {
	// 中文说明：draft 是下一次 create prompt 的交互默认值；terminal name 是唯一 key，
	// 不自动复用，避免用户直接回车撞上同 endpoint 已存在名称。
	shell.TerminalCreateDraft = state.TerminalCreateDraft{
		EndpointID: state.NormalizeEndpointID(request.EndpointID),
		Command:    strings.TrimSpace(prompt.FieldValue("command")),
		Workdir:    strings.TrimSpace(request.CWD),
	}
	return shell
}

func terminalCreateDefaultEndpointID(root state.Root, target terminalPoolTarget, preferred state.EndpointID) state.EndpointID {
	// 中文说明：create prompt 的 endpoint 只决定 TerminalPoolCreateRequestMsg 路由；
	// 当前 pane/floating binding 仍是 TUI 侧连接意图真值，terminal lifecycle 属于 owning daemon。
	if preferred != "" {
		if id := state.NormalizeEndpointID(preferred); terminalCreateEndpointAvailable(root, id) {
			return id
		}
	}
	if draftEndpointID := state.NormalizeEndpointID(root.Shell.ReadonlyDefaults().TerminalCreateDraft.EndpointID); terminalCreateEndpointAvailable(root, draftEndpointID) {
		return draftEndpointID
	}
	if target.FloatingID != "" {
		if binding, ok := root.TerminalViews.FloatingBinding(target.FloatingID); ok && terminalCreateEndpointAvailable(root, binding.EndpointID) {
			return state.NormalizeEndpointID(binding.EndpointID)
		}
	}
	if target.PaneID != "" {
		if binding, ok := root.TerminalViews.PaneBinding(target.PaneID); ok && terminalCreateEndpointAvailable(root, binding.EndpointID) {
			return state.NormalizeEndpointID(binding.EndpointID)
		}
	}
	if root.Session.TerminalID != "" && terminalCreateEndpointAvailable(root, root.Session.EndpointID) {
		return state.NormalizeEndpointID(root.Session.EndpointID)
	}
	for _, endpoint := range state.TerminalCreateEndpointItems(root) {
		return endpoint.ID
	}
	if preferred != "" {
		return state.NormalizeEndpointID(preferred)
	}
	return state.DefaultEndpointID
}

func terminalCreateEndpointAvailable(root state.Root, endpointID state.EndpointID) bool {
	endpointID = state.NormalizeEndpointID(endpointID)
	for _, endpoint := range state.TerminalCreateEndpointItems(root) {
		if endpoint.ID == endpointID {
			return true
		}
	}
	return false
}

func terminalCreateEndpointPromptValue(root state.Root, endpointID state.EndpointID) string {
	endpointID = state.NormalizeEndpointID(endpointID)
	if endpoint, ok := root.Endpoints.DisplayEndpoint(endpointID); ok {
		label := endpoint.DisplayLabel()
		if label != "" && !strings.EqualFold(label, string(endpointID)) {
			return fmt.Sprintf("%s (%s)", label, endpointID)
		}
	}
	return string(endpointID)
}

func terminalCreateEndpointIDFromPrompt(root state.Root, prompt state.PromptState) (state.EndpointID, error) {
	// 中文说明：server 字段可填 endpoint id、label 或 "label (id)" 展示值；
	// 解析只接受当前 reducer-owned registry 中可创建的 endpoint，避免 disabled/manual 目标被隐式拨号。
	value := strings.TrimSpace(prompt.FieldValue("server"))
	if value == "" {
		value = strings.TrimSpace(prompt.FieldValue("endpoint"))
	}
	if value == "" {
		value = terminalCreateEndpointPromptValue(root, prompt.TargetEndpointID)
	}
	if endpointID, ok := terminalCreateEndpointIDFromValue(root, value); ok {
		return endpointID, nil
	}
	return "", fmt.Errorf("server %q is not available for create", value)
}

func terminalCreateEndpointIDFromValue(root state.Root, value string) (state.EndpointID, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	candidates := []string{value}
	if paren := strings.LastIndex(value, "("); paren >= 0 && strings.HasSuffix(value, ")") {
		candidate := strings.TrimSpace(strings.TrimSuffix(value[paren+1:], ")"))
		if candidate != "" {
			candidates = append(candidates, candidate)
		}
	}
	for _, endpoint := range state.TerminalCreateEndpointItems(root) {
		for _, candidate := range candidates {
			if strings.EqualFold(candidate, string(endpoint.ID)) ||
				strings.EqualFold(candidate, endpoint.DisplayLabel()) ||
				strings.EqualFold(candidate, terminalCreateEndpointPromptValue(root, endpoint.ID)) {
				return endpoint.ID, true
			}
		}
	}
	return "", false
}

func terminalCreateEndpointIDFromPickerSelection(root state.Root, row int) state.EndpointID {
	if selected, ok := terminalPickerItemAt(state.TerminalPickerItems(root), row); ok && selected.CreateNew && selected.EndpointID != "" {
		return selected.EndpointID
	}
	return ""
}

func terminalCreateDefaultWorkdirForEndpoint(root state.Root, endpointID state.EndpointID) string {
	if endpoint, ok := root.Endpoints.DisplayEndpoint(endpointID); ok {
		return strings.TrimSpace(endpoint.DefaultCWD)
	}
	return ""
}

func terminalCreateDefaultsError(root state.Root, endpointID state.EndpointID) error {
	endpointID = state.NormalizeEndpointID(endpointID)
	if endpoint, ok := root.Endpoints.DisplayEndpoint(endpointID); ok && strings.TrimSpace(endpoint.DefaultsError) != "" {
		return fmt.Errorf("endpoint %q defaults unavailable: %s", endpointID, strings.TrimSpace(endpoint.DefaultsError))
	}
	return fmt.Errorf("endpoint %q defaults are not loaded", endpointID)
}

func syncCreatePromptWorkdirForServer(root state.Root, shell state.ShellStore) state.ShellStore {
	shell = shell.EnsureDefaults()
	if shell.Overlay.Kind != state.OverlayPrompt || !shell.Overlay.Open {
		return shell
	}
	prompt := shell.Overlay.Prompt
	if prompt.Purpose != "terminal.create" {
		return shell
	}
	endpointID, ok := terminalCreateEndpointIDFromValue(root, prompt.FieldValue("server"))
	if !ok {
		return shell
	}
	return syncCreatePromptDefaultsForEndpoint(root, shell, endpointID)
}

func syncCreatePromptDefaultsForEndpoint(root state.Root, shell state.ShellStore, endpointID state.EndpointID) state.ShellStore {
	shell = shell.EnsureDefaults()
	if shell.Overlay.Kind != state.OverlayPrompt || !shell.Overlay.Open {
		return shell
	}
	prompt := shell.Overlay.Prompt
	if prompt.Purpose != "terminal.create" {
		return shell
	}
	endpointID = state.NormalizeEndpointID(endpointID)
	previousDefault := terminalCreateDefaultWorkdirForEndpoint(root, prompt.TargetEndpointID)
	nextDefault := terminalCreateDefaultWorkdirForEndpoint(root, endpointID)
	previousCommand := terminalCreateDefaultCommandForEndpoint(root, prompt.TargetEndpointID)
	nextCommand := terminalCreateDefaultCommandForEndpoint(root, endpointID)
	previousCommandValue := strings.TrimSpace(terminalCreateCommandDisplay(previousCommand))
	nextCommandValue := strings.TrimSpace(terminalCreateCommandDisplay(nextCommand))
	for index := range prompt.Fields {
		switch prompt.Fields[index].Key {
		case "command":
			current := strings.TrimSpace(prompt.Fields[index].Value)
			if current == previousCommandValue {
				prompt.Fields[index].Value = ""
				prompt.Fields[index].Cursor = 0
			}
			prompt.Fields[index].Placeholder = nextCommandValue
		case "workdir":
			current := strings.TrimSpace(prompt.Fields[index].Value)
			// 中文说明：只替换 prompt 自动带入的 endpoint 默认 CWD；用户手写路径必须保留。
			if current == "" || current == previousDefault {
				prompt.Fields[index].Value = nextDefault
				prompt.Fields[index].Cursor = len([]rune(nextDefault))
			}
		}
	}
	prompt.Command = nextCommand
	if prompt.DefaultName == "" || prompt.DefaultName == terminalCreateDefaultName(previousCommand) {
		prompt.DefaultName = terminalCreateDefaultName(nextCommand)
	}
	prompt.TargetEndpointID = endpointID
	shell.Overlay.Prompt = prompt
	return shell
}

func terminalEditPrompt(item state.TerminalPoolPageItem) state.PromptState {
	// 中文说明：rename 输入归 Shell Prompt 管，提交后再生成 TerminalPoolEditRequestMsg；
	// 这里不直接修改 reducer-owned pool state，也不在 renderer/service 间绕过主消息链。
	return state.PromptState{
		Title:            "Rename Terminal",
		Purpose:          "terminal.rename",
		TargetEndpointID: item.EndpointID,
		TargetID:         item.TerminalID,
		Value:            item.Title,
		Placeholder:      "terminal name",
		Tags:             cloneStringMap(item.Tags),
	}
}

func terminalEditRequestFromPrompt(prompt state.PromptState) (TerminalPoolEditRequestMsg, error) {
	name := strings.TrimSpace(prompt.LastResult)
	if name == "" {
		name = strings.TrimSpace(prompt.Value)
	}
	if strings.TrimSpace(prompt.TargetID) == "" {
		return TerminalPoolEditRequestMsg{}, fmt.Errorf("terminal id is required")
	}
	if name == "" {
		return TerminalPoolEditRequestMsg{}, fmt.Errorf("name is required")
	}
	return TerminalPoolEditRequestMsg{EndpointID: prompt.TargetEndpointID, TerminalID: prompt.TargetID, Title: name, Tags: cloneStringMap(prompt.Tags)}, nil
}

func parsePromptCommand(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	// 这里只实现 tuiv2 表单需要的轻量 shell-like 拆分，避免为 prompt 引入完整 shell 解析器。
	var (
		args       []string
		current    []rune
		inSingle   bool
		inDouble   bool
		escaped    bool
		quotedPart bool
	)
	flush := func(force bool) {
		if len(current) == 0 && !quotedPart && !force {
			return
		}
		args = append(args, string(current))
		current = current[:0]
		quotedPart = false
	}
	for _, r := range value {
		switch {
		case escaped:
			current = append(current, r)
			escaped = false
		case r == '\\' && !inSingle:
			escaped = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
			quotedPart = true
		case r == '"' && !inSingle:
			inDouble = !inDouble
			quotedPart = true
		case !inSingle && !inDouble && unicode.IsSpace(r):
			flush(false)
		default:
			current = append(current, r)
		}
	}
	if escaped || inSingle || inDouble {
		return nil, fmt.Errorf("invalid command syntax")
	}
	flush(false)
	return args, nil
}

func terminalPickerItemAt(items []state.TerminalPickerItem, row int) (state.TerminalPickerItem, bool) {
	if row >= 0 && row < len(items) {
		return items[row], true
	}
	for _, item := range items {
		if item.Selected {
			return item, true
		}
	}
	return state.TerminalPickerItem{}, false
}

func terminalPoolPageItemForAction(root state.Root, row int) (state.TerminalPoolPageItem, bool) {
	items := state.TerminalPoolPageItems(root)
	if row >= 0 {
		root.Shell = root.Shell.SetTerminalPoolSelectedIndex(row, len(items))
		items = state.TerminalPoolPageItems(root)
	}
	if row >= 0 && row < len(items) {
		return items[row], true
	}
	for _, item := range items {
		if item.Selected {
			return item, true
		}
	}
	return state.TerminalPoolPageItem{}, false
}

func reduceWorkbenchTreeOpen(root state.Root, items []state.WorkbenchTreeItem) (state.Root, []Effect) {
	selected, ok := workbenchTreeSelectedItem(items)
	if !ok {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.open", Body: "no node"})
		return root.Advance(), nil
	}
	switch selected.Kind {
	case state.WorkbenchTreeKindPane:
		root.Shell = root.Shell.FocusPane(state.PaneCommandTarget{WorkspaceID: selected.WorkspaceID, TabID: selected.TabID, PaneID: selected.PaneID})
		root.Shell = root.Shell.CloseOverlay()
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "workbench.open", Body: selected.PaneID})
	case state.WorkbenchTreeKindTab:
		targetPaneID := selected.PaneID
		if targetPaneID == "" {
			targetPaneID = firstPaneIDForTab(root.Shell, selected.WorkspaceID, selected.TabID)
		}
		if targetPaneID != "" {
			root.Shell = root.Shell.FocusPane(state.PaneCommandTarget{WorkspaceID: selected.WorkspaceID, TabID: selected.TabID, PaneID: targetPaneID})
			root.Shell = root.Shell.CloseOverlay()
			root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "workbench.open", Body: selected.TabID})
		} else {
			root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.open", Body: "tab has no pane"})
		}
	case state.WorkbenchTreeKindWorkspace:
		root.Shell = root.Shell.CloseOverlay()
		if selected.WorkspaceID == "" || selected.WorkspaceID == root.Shell.EnsureDefaults().Workspace.ID {
			root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "workbench.open", Body: selected.WorkspaceName})
			return root.Advance(), nil
		}
		return reduceWorkbenchCommand(root, state.WorkbenchCommand{
			Action:   state.WorkbenchCommandWorkspaceSwitch,
			TargetID: selected.WorkspaceID,
			Source:   state.PaneCommandSourceMouse,
		})
	case state.WorkbenchTreeKindFloating:
		if selected.FloatingID == "" {
			root.Shell = root.Shell.OpenFloatingOverview()
			return root.Advance(), nil
		}
		if selected.TabID != "" {
			var switchResult state.WorkbenchCommandResult
			root.Shell, switchResult = root.Shell.ApplyWorkbenchCommand(state.WorkbenchCommand{
				Action:   state.WorkbenchCommandTabSwitch,
				TargetID: selected.TabID,
				Target:   state.PaneCommandTarget{WorkspaceID: selected.WorkspaceID},
				Source:   state.PaneCommandSourceMouse,
			})
			if switchResult.Status != state.WorkbenchCommandOK {
				root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "floating", Body: switchResult.Reason})
				return root.Advance(), nil
			}
		}
		var result state.FloatingCommandResult
		root.Shell, result = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
			Action:   state.FloatingCommandSummon,
			TargetID: selected.FloatingID,
			Source:   state.PaneCommandSourceKeyboard,
		})
		if result.Status != state.FloatingCommandOK {
			root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "floating", Body: result.Reason})
			return root.Advance(), nil
		}
		root.Shell = root.Shell.CloseOverlay()
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "floating", Body: selected.FloatingID})
	default:
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.open", Body: "unknown node"})
	}
	return root.Advance(), nil
}

func reduceWorkbenchTreeRename(root state.Root, items []state.WorkbenchTreeItem) (state.Root, []Effect) {
	selected, ok := workbenchTreeSelectedItem(items)
	if !ok {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.rename", Body: "no node"})
		return root.Advance(), nil
	}
	switch selected.Kind {
	case state.WorkbenchTreeKindWorkspace:
		root.Shell = root.Shell.OpenPrompt(state.PromptState{Title: "Rename Workspace", Purpose: "workspace.rename", TargetWorkspaceID: selected.WorkspaceID, TargetID: selected.WorkspaceID, Value: selected.WorkspaceName, Placeholder: "workspace name"})
	case state.WorkbenchTreeKindTab:
		root.Shell = root.Shell.OpenPrompt(state.PromptState{Title: "Rename Tab", Purpose: "tab.rename", TargetWorkspaceID: selected.WorkspaceID, TargetTabID: selected.TabID, TargetID: selected.TabID, Value: selected.TabTitle, Placeholder: "tab name"})
	case state.WorkbenchTreeKindPane:
		root.Shell = root.Shell.OpenPrompt(state.PromptState{Title: "Rename Pane", Purpose: "pane.rename", TargetWorkspaceID: selected.WorkspaceID, TargetTabID: selected.TabID, TargetID: selected.PaneID, Value: selected.PaneTitle, Placeholder: "pane name"})
	default:
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.rename", Body: "unsupported node"})
	}
	return root.Advance(), nil
}

func reduceWorkbenchTreeNew(root state.Root, items []state.WorkbenchTreeItem) (state.Root, []Effect) {
	selected, ok := workbenchTreeSelectedItem(items)
	if !ok {
		return reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceCreate, Name: nextWorkspaceName(root.Shell), Source: state.PaneCommandSourceMouse})
	}
	switch selected.Kind {
	case state.WorkbenchTreeKindWorkspace:
		return reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceCreate, Name: nextWorkspaceName(root.Shell), Source: state.PaneCommandSourceMouse})
	case state.WorkbenchTreeKindTab:
		return reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Target: state.PaneCommandTarget{WorkspaceID: selected.WorkspaceID}, Name: nextTabNameForWorkspace(root.Shell, selected.WorkspaceID), Source: state.PaneCommandSourceMouse})
	case state.WorkbenchTreeKindPane:
		target := state.PaneCommandTarget{WorkspaceID: selected.WorkspaceID, TabID: selected.TabID, PaneID: selected.PaneID}
		if target.PaneID == "" {
			target.PaneID = firstPaneIDForTab(root.Shell, selected.WorkspaceID, selected.TabID)
		}
		command := state.PaneCommand{
			Action:         state.PaneCommandSplit,
			Target:         target,
			SplitDirection: state.SplitDirectionVertical,
			NewPane:        state.PaneState{ID: nextKeyboardPaneIDForWorkspace(root.Shell, selected.WorkspaceID), Title: "pane", Kind: state.PaneEmpty},
			Source:         state.PaneCommandSourceMouse,
		}
		return reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandPaneSplit, Pane: command, Source: state.PaneCommandSourceMouse})
	default:
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.new", Body: "unsupported node"})
		return root.Advance(), nil
	}
}

func reduceWorkbenchTreeDelete(root state.Root, items []state.WorkbenchTreeItem) (state.Root, []Effect) {
	selected, ok := workbenchTreeSelectedItem(items)
	if !ok {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.delete", Body: "no node"})
		return root.Advance(), nil
	}
	switch selected.Kind {
	case state.WorkbenchTreeKindWorkspace:
		return reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceDelete, TargetID: selected.WorkspaceID, Confirm: state.PaneConfirmAccepted, Source: state.PaneCommandSourceMouse})
	case state.WorkbenchTreeKindTab:
		return reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandTabClose, TargetID: selected.TabID, Target: state.PaneCommandTarget{WorkspaceID: selected.WorkspaceID}, Source: state.PaneCommandSourceMouse})
	case state.WorkbenchTreeKindPane:
		return reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandPaneClose, Target: state.PaneCommandTarget{WorkspaceID: selected.WorkspaceID, TabID: selected.TabID, PaneID: selected.PaneID}, Source: state.PaneCommandSourceMouse})
	default:
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.delete", Body: "unsupported node"})
		return root.Advance(), nil
	}
}

func reduceWorkbenchTreeDetach(root state.Root, items []state.WorkbenchTreeItem) (state.Root, []Effect) {
	selected, ok := workbenchTreeSelectedItem(items)
	if !ok {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.detach", Body: "no node"})
		return root.Advance(), nil
	}
	if selected.Kind != state.WorkbenchTreeKindPane {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.detach", Body: "select a pane"})
		return root.Advance(), nil
	}
	return reduceWorkbenchCommand(root, state.WorkbenchCommand{Action: state.WorkbenchCommandPaneDetach, Target: state.PaneCommandTarget{WorkspaceID: selected.WorkspaceID, TabID: selected.TabID, PaneID: selected.PaneID}, Source: state.PaneCommandSourceKeyboard})
}

func reduceWorkbenchTreeZoom(root state.Root, items []state.WorkbenchTreeItem) (state.Root, []Effect) {
	selected, ok := workbenchTreeSelectedItem(items)
	if !ok {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.zoom", Body: "no node"})
		return root.Advance(), nil
	}
	if selected.Kind != state.WorkbenchTreeKindPane {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.zoom", Body: "select a pane"})
		return root.Advance(), nil
	}
	return reducePaneCommand(root, state.PaneCommand{Action: state.PaneCommandToggleZoom, Target: state.PaneCommandTarget{WorkspaceID: selected.WorkspaceID, TabID: selected.TabID, PaneID: selected.PaneID}, Source: state.PaneCommandSourceKeyboard})
}

func workbenchTreeSelectedItem(items []state.WorkbenchTreeItem) (state.WorkbenchTreeItem, bool) {
	if len(items) == 0 {
		return state.WorkbenchTreeItem{}, false
	}
	for _, item := range items {
		if item.Selected {
			return item, true
		}
	}
	return items[0], true
}

func firstPaneIDForTab(shell state.ShellStore, workspaceID string, tabID string) string {
	workspace := workspaceForWorkbenchAction(shell, workspaceID)
	for _, tab := range workspace.Tabs {
		if tab.ID == tabID && len(tab.Panes) > 0 {
			return tab.Panes[0].ID
		}
	}
	return ""
}

func nextTabNameForWorkspace(shell state.ShellStore, workspaceID string) string {
	workspace := workspaceForWorkbenchAction(shell, workspaceID)
	return fmt.Sprintf("tab %d", len(workspace.Tabs)+1)
}

func nextKeyboardPaneIDForWorkspace(shell state.ShellStore, workspaceID string) string {
	workspace := workspaceForWorkbenchAction(shell, workspaceID)
	for i := 2; ; i++ {
		id := fmt.Sprintf("pane-%d", i)
		if !workspaceHasPaneID(workspace, id) {
			return id
		}
	}
}

func workspaceForWorkbenchAction(shell state.ShellStore, workspaceID string) state.WorkspaceState {
	shell = shell.EnsureDefaults()
	workspace := shell.Workspace
	for _, candidate := range shell.Workspaces {
		if candidate.ID == workspaceID {
			workspace = candidate
			break
		}
	}
	return workspace
}

func workspaceHasPaneID(workspace state.WorkspaceState, paneID string) bool {
	for _, tab := range workspace.Tabs {
		for _, pane := range tab.Panes {
			if pane.ID == paneID {
				return true
			}
		}
	}
	return false
}

func terminalRefForContentAction(root state.Root, paneID string) state.TerminalRef {
	if binding, ok := root.TerminalViews.PaneBinding(paneID); ok && binding.TerminalID != "" {
		return binding.TerminalRef()
	}
	return state.TerminalRef{}
}

func terminalPoolTargetForOverlay(root state.Root) terminalPoolTarget {
	shell := root.Shell.EnsureDefaults()
	if targetID := shell.Overlay.TargetID; targetID != "" {
		if target, ok := terminalPoolTargetForID(shell, targetID); ok {
			return target
		}
	}
	return terminalPoolTargetForActive(root)
}

func terminalPoolTargetForActive(root state.Root) terminalPoolTarget {
	shell := root.Shell.EnsureDefaults()
	if active, ok := shell.ActiveSurfaceTarget(); ok && active.Floating {
		if target, ok := terminalPoolTargetForID(shell, active.FloatingID); ok {
			target.ViewID = root.TerminalViews.FloatingViewID(active.FloatingID)
			return target
		}
	}
	return terminalPoolTarget{PaneID: shell.ActivePaneID, ViewID: root.TerminalViews.PaneViewID(shell.ActivePaneID)}
}

func terminalPoolTargetForID(shell state.ShellStore, id string) (terminalPoolTarget, bool) {
	shell = shell.EnsureDefaults()
	for _, floating := range shell.ActiveFloatings() {
		if floating.ID == id || floating.Pane.ID == id {
			return terminalPoolTarget{PaneID: floating.Pane.ID, FloatingID: floating.ID, ViewID: state.TerminalFloatingViewID(floating.ID)}, true
		}
	}
	if _, ok := shell.Pane(state.PaneCommandTarget{PaneID: id}); ok {
		return terminalPoolTarget{PaneID: id, ViewID: state.TerminalPaneViewID(id)}, true
	}
	return terminalPoolTarget{}, false
}

func reducePaneCommand(root state.Root, command state.PaneCommand) (state.Root, []Effect) {
	command = command.WithDefaults(root.Shell)
	targetPane, hasTargetPane := root.Shell.Pane(command.Target)
	command = inheritSplitTerminalPane(root, command, targetPane, hasTargetPane)
	nextShell, result := root.Shell.ApplyPaneCommand(command)
	if result.Status == state.PaneCommandOK {
		effects := paneCommandEffects(root, command, result, targetPane, hasTargetPane)
		detachEffects := terminalDetachEffectsForPaneCommand(root, command, result)
		if command.Action == state.PaneCommandClose {
			effects = append(effects, copyHistoryCleanupEffectsForView(root, root.TerminalViews.PaneViewID(command.Target.PaneID))...)
		}
		root.Shell = deactivateFloatingAfterPaneCommand(nextShell, command)
		root = updateTerminalViewsAfterPaneCommand(root, command, targetPane, hasTargetPane)
		root.Shell = addPaneCommandToast(root.Shell, command, result)
		effects = append(effects, detachEffects...)
		return root.Advance(), effects
	}
	root.Shell = addPaneCommandToast(root.Shell, command, result)
	return root.Advance(), []Effect{PaneCommandFeedbackEffect{Result: result, Command: command}}
}

func inheritSplitTerminalPane(root state.Root, command state.PaneCommand, targetPane state.PaneState, hasTargetPane bool) state.PaneCommand {
	if command.Action != state.PaneCommandSplit || command.NewPane.TerminalID != "" {
		return command
	}
	if command.NewPane.Kind == state.PaneEmpty {
		return command
	}
	terminalID := ""
	if hasTargetPane {
		terminalID = targetPane.TerminalID
		if binding, ok := root.TerminalViews.PaneBinding(targetPane.ID); ok && binding.TerminalID != "" {
			terminalID = binding.TerminalID
		}
	}
	if terminalID == "" {
		return command
	}
	command.NewPane.TerminalID = terminalID
	command.NewPane.Kind = state.PaneTerminalLive
	if command.NewPane.Title == "" || command.NewPane.Title == command.NewPane.ID || command.NewPane.Title == "pane" {
		command.NewPane.Title = terminalID
	}
	return command
}

func reduceFloatingCommand(root state.Root, command state.FloatingCommand) (state.Root, []Effect) {
	command = withFloatingCommandDefaults(root, command)
	detachEffects := terminalDetachEffectsForFloatingCommand(root, command)
	nextShell, result := root.Shell.ApplyFloatingCommand(command)
	root.Shell = addFloatingCommandToast(nextShell, command, result)
	effects := []Effect{}
	if result.Status == state.FloatingCommandOK && command.Action == state.FloatingCommandClose {
		effects = append(effects, copyHistoryCleanupEffectsForView(root, root.TerminalViews.FloatingViewID(result.ID))...)
		root = invalidateCopyModeForClosedFloating(root, result.ID)
		root.TerminalViews = root.TerminalViews.DetachFloating(result.ID)
		effects = append(effects, detachEffects...)
	}
	if result.Status == state.FloatingCommandOK && command.Action == state.FloatingCommandDeactivate {
		root = invalidateCopyModeForInactiveView(root)
	}
	if result.Status == state.FloatingCommandOK && shouldPersistFloatingCommand(command) {
		effects = append(effects, FuncEffect{Run: func(context.Context) Msg {
			return WorkbenchStoragePersistRequestMsg{Reason: string(result.Action)}
		}})
	}
	return root.Advance(), effects
}

func reduceWorkbenchCommand(root state.Root, command state.WorkbenchCommand) (state.Root, []Effect) {
	return reduceWorkbenchCommandWithOptions(root, command, workbenchCommandOptions{})
}

type workbenchCommandOptions struct {
	OpenPickerAfterOK bool
}

func reduceWorkbenchCommandWithOptions(root state.Root, command state.WorkbenchCommand, options workbenchCommandOptions) (state.Root, []Effect) {
	previousShell := root.Shell
	nextShell, result := root.Shell.ApplyWorkbenchCommand(command)
	root.Shell = addWorkbenchCommandToast(nextShell, result)
	if result.Status != state.WorkbenchCommandOK {
		return root.Advance(), nil
	}
	detachEffects := terminalDetachEffectsForWorkbenchCommand(root, previousShell, command, result)
	copyHistoryEffects := copyHistoryCleanupEffectsForWorkbenchCommand(root, previousShell, command, result)
	killEffects := terminalKillEffectsForWorkbenchResult(root, previousShell, result)
	root = updateTerminalViewsAfterWorkbenchCommand(root, previousShell, command, result)
	if workbenchCommandChangesActiveView(command.Action) {
		root = invalidateCopyModeForInactiveView(root)
	}
	effects := []Effect{FuncEffect{Run: func(context.Context) Msg {
		return WorkbenchStoragePersistRequestMsg{Reason: string(result.Action)}
	}}}
	if options.OpenPickerAfterOK && command.Action == state.WorkbenchCommandTabCreate {
		root, effects = openTerminalPickerForCreatedTab(root, effects)
	}
	effects = append(effects, killEffects...)
	effects = append(effects, detachEffects...)
	effects = append(effects, copyHistoryEffects...)
	return root.Advance(), effects
}

func copyHistoryCleanupEffectsForWorkbenchCommand(root state.Root, previousShell state.ShellStore, command state.WorkbenchCommand, result state.WorkbenchCommandResult) []Effect {
	var viewIDs []string
	switch result.Action {
	case state.WorkbenchCommandPaneDetach, state.WorkbenchCommandPaneClose:
		viewIDs = append(viewIDs, root.TerminalViews.PaneViewID(result.ID))
	case state.WorkbenchCommandTabClose:
		for _, pane := range panesForWorkbenchTarget(previousShell, command.TargetID) {
			viewIDs = append(viewIDs, root.TerminalViews.PaneViewID(pane.ID))
		}
	default:
		return nil
	}
	var effects []Effect
	for _, viewID := range viewIDs {
		effects = append(effects, copyHistoryCleanupEffectsForView(root, viewID)...)
	}
	return effects
}

func terminalKillEffectsForWorkbenchResult(root state.Root, previousShell state.ShellStore, result state.WorkbenchCommandResult) []Effect {
	refs := terminalRefsForWorkbenchResult(root, previousShell, result)
	if len(refs) == 0 {
		return nil
	}
	effects := make([]Effect, 0, len(refs))
	for _, ref := range refs {
		ref := ref.Normalize()
		if ref.Empty() {
			continue
		}
		effects = append(effects, FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolKillRequestMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID}
		}})
	}
	return effects
}

func terminalRefsForWorkbenchResult(root state.Root, previousShell state.ShellStore, result state.WorkbenchCommandResult) []state.TerminalRef {
	if len(result.Killed) == 0 {
		return nil
	}
	// 中文说明：Workbench state 目前只回传 daemon-local TerminalID；
	// app 层必须在 TerminalView 被删除前从 pane/view binding 还原 TerminalRef，避免 close-and-kill 打到 local 同名 terminal。
	switch result.Action {
	case state.WorkbenchCommandPaneKill:
		if ref := terminalRefForContentAction(root, result.ID); !ref.Empty() {
			return []state.TerminalRef{ref}
		}
	case state.WorkbenchCommandTabKill:
		if tab, ok := shellTabByID(previousShell, result.ID); ok {
			refs := terminalRefsForPanes(root, tab.Panes, result.Killed)
			if len(refs) > 0 {
				return refs
			}
		}
	}
	return terminalRefsForTerminalIDs(root, result.Killed)
}

func shellTabByID(shell state.ShellStore, tabID string) (state.TabState, bool) {
	if tabID == "" {
		return state.TabState{}, false
	}
	for _, tab := range shell.Workspace.Tabs {
		if tab.ID == tabID {
			return tab, true
		}
	}
	for _, workspace := range shell.Workspaces {
		for _, tab := range workspace.Tabs {
			if tab.ID == tabID {
				return tab, true
			}
		}
	}
	return state.TabState{}, false
}

func terminalRefsForPanes(root state.Root, panes []state.PaneState, terminalIDs []string) []state.TerminalRef {
	wanted := map[string]struct{}{}
	for _, terminalID := range terminalIDs {
		if terminalID != "" {
			wanted[terminalID] = struct{}{}
		}
	}
	refs := make([]state.TerminalRef, 0, len(wanted))
	seen := map[string]struct{}{}
	for _, pane := range panes {
		if pane.TerminalID == "" {
			continue
		}
		if _, ok := wanted[pane.TerminalID]; !ok {
			continue
		}
		ref := terminalRefForContentAction(root, pane.ID)
		if ref.Empty() {
			ref = state.LocalTerminalRef(pane.TerminalID)
		}
		if key := ref.Key(); key != "" {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			refs = append(refs, ref)
		}
	}
	return refs
}

func terminalRefsForTerminalIDs(root state.Root, terminalIDs []string) []state.TerminalRef {
	refs := make([]state.TerminalRef, 0, len(terminalIDs))
	for _, terminalID := range terminalIDs {
		if terminalID == "" {
			continue
		}
		refs = append(refs, terminalRefForTerminalID(root, terminalID))
	}
	return refs
}

func terminalRefForTerminalID(root state.Root, terminalID string) state.TerminalRef {
	seen := map[string]state.TerminalRef{}
	for _, binding := range root.TerminalViews.Bindings() {
		if binding.TerminalID != terminalID {
			continue
		}
		ref := binding.TerminalRef()
		if key := ref.Key(); key != "" {
			seen[key] = ref
		}
	}
	if len(seen) == 1 {
		for _, ref := range seen {
			return ref
		}
	}
	return state.LocalTerminalRef(terminalID)
}

func openTerminalPickerForCreatedTab(root state.Root, effects []Effect) (state.Root, []Effect) {
	root.Shell = root.Shell.OpenTerminalPicker()
	effects = append(effects, terminalPickerListRequestEffect())
	return root, effects
}

func updateTerminalViewsAfterPaneCommand(root state.Root, command state.PaneCommand, targetPane state.PaneState, hasTargetPane bool) state.Root {
	switch command.Action {
	case state.PaneCommandSplit:
		if command.NewPane.TerminalID == "" && command.NewPane.Kind == state.PaneEmpty {
			return root
		}
		terminalID := command.NewPane.TerminalID
		var targetBinding state.TerminalViewBinding
		hasTargetBinding := false
		if hasTargetPane {
			targetBinding, hasTargetBinding = root.TerminalViews.PaneBinding(targetPane.ID)
			if terminalID == "" && hasTargetBinding {
				terminalID = targetBinding.TerminalID
			}
		}
		if terminalID == "" && hasTargetPane {
			terminalID = targetPane.TerminalID
		}
		if terminalID == "" {
			return root
		}
		binding := state.NewPaneTerminalView(command.NewPane.ID, terminalID, 0, 0, 0, state.TerminalResizeRoleFollower, "", state.TerminalPaneViewID(command.NewPane.ID), false)
		if hasTargetBinding && targetBinding.TerminalID == terminalID {
			binding.SurfaceID = targetBinding.SurfaceID
			binding.DesiredCols = targetBinding.DesiredCols
			binding.DesiredRows = targetBinding.DesiredRows
		}
		root.TerminalViews = root.TerminalViews.BindPane(binding)
	case state.PaneCommandClose:
		root = invalidateCopyModeForClosedPane(root, command.Target.PaneID)
		root.TerminalViews = root.TerminalViews.DetachPane(command.Target.PaneID)
	}
	return root
}

func updateTerminalViewsAfterWorkbenchCommand(root state.Root, previousShell state.ShellStore, command state.WorkbenchCommand, result state.WorkbenchCommandResult) state.Root {
	switch result.Action {
	case state.WorkbenchCommandPaneSplit:
		root = bindWorkbenchSplitTerminalView(root, previousShell, command, result)
	case state.WorkbenchCommandPaneDetach:
		root = invalidateCopyModeForClosedPane(root, result.ID)
		root.TerminalViews = root.TerminalViews.DetachPane(result.ID)
	case state.WorkbenchCommandPaneClose:
		root = invalidateCopyModeForClosedPane(root, result.ID)
		root.TerminalViews = root.TerminalViews.DetachPane(result.ID)
	case state.WorkbenchCommandPaneKill, state.WorkbenchCommandTabKill:
		for _, terminalID := range result.Killed {
			root.TerminalViews = root.TerminalViews.RemoveTerminal(terminalID)
		}
	case state.WorkbenchCommandTabClose:
		for _, pane := range panesForWorkbenchTarget(previousShell, command.TargetID) {
			root = invalidateCopyModeForClosedPane(root, pane.ID)
			root.TerminalViews = root.TerminalViews.DetachPane(pane.ID)
		}
	}
	return root
}

func invalidateCopyModeForClosedPane(root state.Root, paneID string) state.Root {
	if paneID == "" {
		return root
	}
	return root.WithoutCopyHistorySession(root.TerminalViews.PaneViewID(paneID))
}

func invalidateCopyModeForClosedFloating(root state.Root, floatingID string) state.Root {
	if floatingID == "" {
		return root
	}
	return root.WithoutCopyHistorySession(root.TerminalViews.FloatingViewID(floatingID))
}

func invalidateCopyModeForInactiveView(root state.Root) state.Root {
	// 中文说明：history/copy 是每个 TerminalView 的主动交互态；
	// 切换 active pane/floating 不能替其它 view 自动退出。
	return root
}

func workbenchCommandChangesActiveView(action state.WorkbenchCommandAction) bool {
	switch action {
	case state.WorkbenchCommandTabSwitch,
		state.WorkbenchCommandTabNext,
		state.WorkbenchCommandTabPrevious,
		state.WorkbenchCommandTabClose,
		state.WorkbenchCommandWorkspaceSwitch,
		state.WorkbenchCommandWorkspaceNext,
		state.WorkbenchCommandWorkspacePrevious,
		state.WorkbenchCommandWorkspaceDelete:
		return true
	default:
		return false
	}
}

func bindWorkbenchSplitTerminalView(root state.Root, previousShell state.ShellStore, command state.WorkbenchCommand, result state.WorkbenchCommandResult) state.Root {
	if result.ID == "" {
		return root
	}
	if command.Pane.NewPane.TerminalID == "" && command.Pane.NewPane.Kind == state.PaneEmpty {
		return root
	}
	target := command.Pane.Target
	if target.PaneID == "" {
		previous := previousShell.EnsureDefaults()
		target = state.PaneCommandTarget{WorkspaceID: previous.Workspace.ID, TabID: previous.Workspace.ActiveTabID, PaneID: previous.ActivePaneID}
	}
	targetPane, hasTargetPane := previousShell.Pane(target)
	terminalID := command.Pane.NewPane.TerminalID
	var targetBinding state.TerminalViewBinding
	hasTargetBinding := false
	if hasTargetPane {
		targetBinding, hasTargetBinding = root.TerminalViews.PaneBinding(targetPane.ID)
		if terminalID == "" && hasTargetBinding {
			terminalID = targetBinding.TerminalID
		}
		if terminalID == "" {
			terminalID = targetPane.TerminalID
		}
	}
	if terminalID == "" {
		return root
	}
	binding := state.NewPaneTerminalView(result.ID, terminalID, 0, 0, 0, state.TerminalResizeRoleFollower, "", state.TerminalPaneViewID(result.ID), false)
	if hasTargetBinding && targetBinding.TerminalID == terminalID {
		binding.SurfaceID = targetBinding.SurfaceID
		binding.DesiredCols = targetBinding.DesiredCols
		binding.DesiredRows = targetBinding.DesiredRows
	}
	root.TerminalViews = root.TerminalViews.BindPane(binding)
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: result.ID}, terminalID)
	return root
}

func panesForWorkbenchTarget(shell state.ShellStore, tabID string) []state.PaneState {
	shell = shell.EnsureDefaults()
	if tabID == "" {
		tabID = shell.Workspace.ActiveTabID
	}
	for _, tab := range shell.Workspace.Tabs {
		if tab.ID == tabID {
			return tab.Panes
		}
	}
	return nil
}

func withFloatingCommandDefaults(root state.Root, command state.FloatingCommand) state.FloatingCommand {
	if command.BoundsW <= 0 {
		command.BoundsW = root.Viewport.Cols
	}
	if command.BoundsH <= 0 {
		command.BoundsH = root.Viewport.Rows
	}
	if command.BoundsW <= 0 {
		command.BoundsW = 80
	}
	if command.BoundsH <= 0 {
		command.BoundsH = 24
	}
	if command.Action == state.FloatingCommandCreate && command.Pane.ID == "" {
		command.Pane = state.PaneState{ID: "floating-pane", Title: "floating", Kind: state.PaneEmpty}
	}
	if command.Action == state.FloatingCommandFit || command.Action == state.FloatingCommandToggleAutoFit || command.Action == state.FloatingCommandRefreshAutoFit {
		command.FitCols, command.FitRows = floatingCommandFitSize(root, command)
	}
	return command
}

func addFloatingCommandToast(shell state.ShellStore, command state.FloatingCommand, result state.FloatingCommandResult) state.ShellStore {
	if result.Status == state.FloatingCommandOK {
		if shouldSuppressFloatingCommandSuccessToast(command) {
			return shell
		}
		return shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: string(result.Action), Body: result.ID})
	}
	return shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: string(result.Action), Body: result.Reason})
}

func shouldSuppressFloatingCommandSuccessToast(command state.FloatingCommand) bool {
	switch command.Action {
	case state.FloatingCommandFocusRaise, state.FloatingCommandDeactivate, state.FloatingCommandMove, state.FloatingCommandPosition, state.FloatingCommandResize, state.FloatingCommandRefreshAutoFit:
		return true
	default:
		return false
	}
}

func shouldPersistFloatingCommand(command state.FloatingCommand) bool {
	switch command.Action {
	case state.FloatingCommandCreate,
		state.FloatingCommandClose:
		return true
	default:
		return false
	}
}

func floatingCommandFitSize(root state.Root, command state.FloatingCommand) (int, int) {
	binding, ok := floatingCommandBinding(root, command)
	if !ok || binding.TerminalID == "" {
		return 0, 0
	}
	surface := root.Surface.SurfaceForTerminalRef(binding.TerminalRef())
	if surface.Cols > 0 && surface.Rows > 0 {
		return surface.Cols, surface.Rows
	}
	if binding.DesiredCols > 0 && binding.DesiredRows > 0 {
		return binding.DesiredCols, binding.DesiredRows
	}
	if root.Session.TerminalRef().Equal(binding.TerminalRef()) {
		return root.Session.DesiredSize()
	}
	return 0, 0
}

func floatingCommandBinding(root state.Root, command state.FloatingCommand) (state.TerminalViewBinding, bool) {
	if command.TargetID != "" {
		if binding, ok := root.TerminalViews.FloatingBinding(command.TargetID); ok {
			return binding, true
		}
	}
	activeFloatingID := root.Shell.EnsureDefaults().ActiveFloatingID()
	if activeFloatingID == "" {
		return state.TerminalViewBinding{}, false
	}
	return root.TerminalViews.FloatingBinding(activeFloatingID)
}

func deactivateFloatingAfterPaneCommand(shell state.ShellStore, command state.PaneCommand) state.ShellStore {
	switch command.Action {
	case state.PaneCommandFocus, state.PaneCommandSplit, state.PaneCommandClose, state.PaneCommandZoom, state.PaneCommandUnzoom, state.PaneCommandToggleZoom:
		next, _ := shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandDeactivate, Source: command.Source})
		return next
	default:
		return shell
	}
}

func addWorkbenchCommandToast(shell state.ShellStore, result state.WorkbenchCommandResult) state.ShellStore {
	if result.Status == state.WorkbenchCommandOK {
		return shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: string(result.Action), Body: result.ID})
	}
	return shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: string(result.Action), Body: result.Reason})
}

func paneCommandEffects(root state.Root, command state.PaneCommand, result state.PaneCommandResult, targetPane state.PaneState, hasTargetPane bool) []Effect {
	effects := []Effect{PaneCommandFeedbackEffect{Result: result, Command: command}}
	if command.Action != state.PaneCommandKill && command.Action != state.PaneCommandCloseAndKill {
		return effects
	}
	if !hasTargetPane {
		return effects
	}
	ref := terminalRefForContentAction(root, targetPane.ID)
	if ref.Empty() {
		return append(effects, FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolKillResultMsg{PaneID: targetPane.ID, CloseOnSuccess: command.Action == state.PaneCommandCloseAndKill, Err: fmt.Errorf("pane terminal binding unavailable")}
		}})
	}
	effects = append(effects, FuncEffect{Run: func(context.Context) Msg {
		return TerminalPoolKillRequestMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, PaneID: targetPane.ID, CloseOnSuccess: command.Action == state.PaneCommandCloseAndKill}
	}})
	return effects
}

func terminalDetachEffectsForPaneCommand(root state.Root, command state.PaneCommand, result state.PaneCommandResult) []Effect {
	if result.Status != state.PaneCommandOK || command.Action != state.PaneCommandClose {
		return nil
	}
	if binding, ok := root.TerminalViews.PaneBinding(command.Target.PaneID); ok {
		if req, ok := terminalDetachRequestFromBinding(binding); ok {
			return []Effect{terminalDetachEffect(req)}
		}
	}
	return nil
}

func terminalDetachEffectsForFloatingCommand(root state.Root, command state.FloatingCommand) []Effect {
	if command.Action != state.FloatingCommandClose {
		return nil
	}
	binding, ok := floatingCommandBinding(root, command)
	if !ok {
		return nil
	}
	if req, ok := terminalDetachRequestFromBinding(binding); ok {
		return []Effect{terminalDetachEffect(req)}
	}
	return nil
}

func terminalDetachEffectsForWorkbenchCommand(root state.Root, previousShell state.ShellStore, command state.WorkbenchCommand, result state.WorkbenchCommandResult) []Effect {
	if result.Status != state.WorkbenchCommandOK {
		return nil
	}
	var bindings []state.TerminalViewBinding
	switch result.Action {
	case state.WorkbenchCommandPaneDetach, state.WorkbenchCommandPaneClose:
		if binding, ok := root.TerminalViews.PaneBinding(result.ID); ok {
			bindings = append(bindings, binding)
		}
	case state.WorkbenchCommandTabClose:
		for _, pane := range panesForWorkbenchTarget(previousShell, command.TargetID) {
			if binding, ok := root.TerminalViews.PaneBinding(pane.ID); ok {
				bindings = append(bindings, binding)
			}
		}
	default:
		return nil
	}
	effects := make([]Effect, 0, len(bindings))
	for _, binding := range bindings {
		if req, ok := terminalDetachRequestFromBinding(binding); ok {
			effects = append(effects, terminalDetachEffect(req))
		}
	}
	return effects
}

func terminalDetachRequestFromBinding(binding state.TerminalViewBinding) (port.TerminalDetachRequest, bool) {
	if binding.TerminalID == "" || binding.Channel == 0 {
		return port.TerminalDetachRequest{}, false
	}
	return port.TerminalDetachRequest{
		EndpointID:  binding.EndpointID,
		TerminalID:  binding.TerminalID,
		Channel:     binding.Channel,
		SurfaceID:   binding.SurfaceID,
		ViewID:      binding.ViewID,
		Session:     binding.AttachmentSession(),
		OperationID: "detach:" + binding.OperationID,
	}, true
}

func terminalDetachEffect(req port.TerminalDetachRequest) Effect {
	// 中文说明：pane/floating close 删除的是当前 view；core attachment 必须同步释放，
	// 否则 terminal pool 的 xN 和 resize owner 会长期保留僵尸 view。
	return FuncEffect{Run: func(context.Context) Msg {
		return LiveDetachRequestMsg{Request: req}
	}}
}

func addPaneCommandToast(shell state.ShellStore, command state.PaneCommand, result state.PaneCommandResult) state.ShellStore {
	switch result.Status {
	case state.PaneCommandOK:
		if shouldSuppressPaneCommandSuccessToast(command) {
			return shell
		}
		return shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: string(command.Action)})
	case state.PaneCommandNeedsConfirmation:
		return shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: string(command.Action), Body: result.Reason, Pending: true})
	case state.PaneCommandInvalid:
		return shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: string(command.Action), Body: result.Reason})
	default:
		return shell
	}
}

func shouldSuppressPaneCommandSuccessToast(command state.PaneCommand) bool {
	switch command.Action {
	case state.PaneCommandFocus, state.PaneCommandFocusNext, state.PaneCommandFocusPrevious, state.PaneCommandResize,
		state.PaneCommandKill, state.PaneCommandCloseAndKill:
		return true
	default:
		return false
	}
}

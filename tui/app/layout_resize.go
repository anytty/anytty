package app

import (
	"context"

	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

// terminalDefaultResizePolicy 返回 TUI 本地配置决定的 attach resize 策略。
// 默认跟随 daemon 仲裁；开启 auto_take_owner 后，TUI 会主动请求 owner。
func terminalDefaultResizePolicy(root state.Root) string {
	if root.Config.Terminal.AutoTakeOwner {
		return state.TerminalResizeRoleOwner
	}
	return state.TerminalResizeRoleFollower
}

// NewTerminalLayoutResizeReducer 把最新 shell/layout state 投影成所有可见 resize owner 的 content rect。
// 这里不直接调用 terminal service，而是通过 LiveResizeMsg 回到 live reducer，保持 service IO 不越过 message path。
func NewTerminalLayoutResizeReducer() Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		if !terminalLayoutMayNeedResize(root, msg) {
			return root, nil
		}
		if root.Config.Terminal.AutoTakeOwner {
			if effects, ok := autoTakeOwnerEffects(root); ok {
				return root, effects
			}
		}
		if terminalLayoutNeedsAllVisibleOwnerResize(root, msg) {
			if targets, ok := visibleResizeOwnerTargets(root, render.Rect{}); ok && len(targets) > 0 {
				effects := make([]Effect, 0, len(targets))
				for _, target := range targets {
					nextViews, decision := root.TerminalViews.RequestViewResize(target.binding.ViewID, target.rect.W, target.rect.H)
					root.TerminalViews = nextViews
					if !decision.Allowed || !decision.Changed {
						continue
					}
					if root.Session.Attached && root.Session.TerminalRef().Equal(target.binding.TerminalRef()) {
						root.Session = root.Session.RequestResize(target.rect.W, target.rect.H)
					}
					binding := target.binding
					cols, rows, seq := target.rect.W, target.rect.H, decision.Seq
					effects = append(effects, FuncEffect{
						Run: func(context.Context) Msg {
							return LiveResizeMsg{EndpointID: binding.EndpointID, TerminalID: binding.TerminalID, Cols: cols, Rows: rows, Seq: seq, ViewID: binding.ViewID}
						},
					})
				}
				return root, effects
			}
		}
		binding, rect, ok := resizeOwnerTerminalContentRect(root, render.Rect{})
		if !ok {
			return root, nil
		}
		if binding.ViewID != "" {
			if !binding.Attached || binding.Channel == 0 {
				return root, nil
			}
			nextViews, decision := root.TerminalViews.RequestViewResize(binding.ViewID, rect.W, rect.H)
			if !decision.Allowed || !decision.Changed {
				return root, nil
			}
			root.TerminalViews = nextViews
			if root.Session.Attached && root.Session.TerminalRef().Equal(binding.TerminalRef()) {
				root.Session = root.Session.RequestResize(rect.W, rect.H)
			}
			return root, []Effect{FuncEffect{
				Run: func(context.Context) Msg {
					return LiveResizeMsg{EndpointID: binding.EndpointID, TerminalID: binding.TerminalID, Cols: rect.W, Rows: rect.H, Seq: decision.Seq, ViewID: binding.ViewID}
				},
			}}
		}
		if !root.Session.Attached {
			return root, nil
		}
		desiredCols, desiredRows := root.Session.DesiredSize()
		if rect.W == desiredCols && rect.H == desiredRows {
			return root, nil
		}
		cols := rect.W
		rows := rect.H
		root.Session = root.Session.RequestResize(cols, rows)
		seq := root.Session.ResizeRequestSeq
		ref := root.Session.TerminalRef()
		return root, []Effect{FuncEffect{
			Run: func(context.Context) Msg {
				return LiveResizeMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, Cols: cols, Rows: rows, Seq: seq}
			},
		}}
	}
}

func autoTakeOwnerEffects(root state.Root) ([]Effect, bool) {
	binding, ok := activeTerminalViewBinding(root)
	if !ok || binding.TerminalID == "" || !binding.Attached || binding.Channel == 0 || binding.HasResizeOwner() || binding.OwnerAcquirePending {
		return nil, false
	}
	rect, ok := terminalViewContentRect(root, render.Rect{}, binding)
	if !ok || rect.W <= 0 || rect.H <= 0 {
		return nil, false
	}
	var decision state.TerminalViewResizeDecision
	root.TerminalViews, decision = root.TerminalViews.RequestViewResizeOwner(binding.ViewID, rect.W, rect.H)
	if !decision.Allowed {
		return nil, false
	}
	seq := decision.Seq
	return []Effect{FuncEffect{
		Run: func(context.Context) Msg {
			return LiveResizeMsg{EndpointID: binding.EndpointID, TerminalID: binding.TerminalID, Cols: rect.W, Rows: rect.H, Seq: seq, ViewID: binding.ViewID, TakeOwnership: true, ExpectedOwnerEpoch: binding.ResizeEpoch}
		},
	}}, true
}

func terminalLayoutNeedsAllVisibleOwnerResize(root state.Root, msg Msg) bool {
	switch msg := msg.(type) {
	case HostResizeMsg,
		ShellWorkbenchCommandMsg,
		ShellSetPanelPresentationMsg,
		ShellTogglePanelPresentationMsg,
		ShellSetHeaderVisibleMsg,
		ShellToggleHeaderVisibleMsg,
		ShellSetFooterVisibleMsg,
		ShellToggleFooterVisibleMsg,
		ShellSplitActivePaneMsg:
		return true
	case ShellPaneCommandMsg:
		switch msg.Command.Action {
		case state.PaneCommandSplit,
			state.PaneCommandClose,
			state.PaneCommandCloseAndKill,
			state.PaneCommandZoom,
			state.PaneCommandUnzoom,
			state.PaneCommandToggleZoom,
			state.PaneCommandResize,
			state.PaneCommandSetSize,
			state.PaneCommandBalance,
			state.PaneCommandSetPresentation,
			state.PaneCommandTogglePresentation:
			return true
		}
	case ShellFloatingCommandMsg:
		return floatingCommandMayResizeTerminal(root, msg.Command)
	}
	return false
}

type terminalLayoutResizeTarget struct {
	binding state.TerminalViewBinding
	rect    render.Rect
}

func visibleResizeOwnerTargets(root state.Root, fallbackViewport render.Rect) ([]terminalLayoutResizeTarget, bool) {
	plan, ok := terminalLayoutPlan(root, fallbackViewport)
	if !ok {
		return nil, false
	}
	targets := make([]terminalLayoutResizeTarget, 0, len(plan.Panels)+len(plan.Floatings))
	seen := make(map[string]struct{}, len(root.TerminalViews.Views))
	appendTarget := func(binding state.TerminalViewBinding, rect render.Rect) {
		if !binding.Attached || binding.Channel == 0 || !binding.HasAuthoritativeResizeOwner() || binding.ViewID == "" || rect.W <= 0 || rect.H <= 0 {
			return
		}
		if _, exists := seen[binding.ViewID]; exists {
			return
		}
		seen[binding.ViewID] = struct{}{}
		targets = append(targets, terminalLayoutResizeTarget{binding: binding, rect: rect})
	}
	for _, panel := range plan.Panels {
		if binding, found := root.TerminalViews.PaneBinding(panel.Panel.ID); found {
			appendTarget(binding, panel.ContentRect)
		}
	}
	for _, floating := range plan.Floatings {
		if binding, found := root.TerminalViews.FloatingBinding(floating.Floating.ID); found {
			appendTarget(binding, floating.ContentRect)
		}
	}
	return targets, true
}

func resizeOwnerTerminalContentRect(root state.Root, fallbackViewport render.Rect) (state.TerminalViewBinding, render.Rect, bool) {
	if activeBinding, hasActiveBinding := activeTerminalViewBinding(root); hasActiveBinding {
		if activeBinding.HasAuthoritativeResizeOwner() {
			if rect, ok := terminalViewContentRect(root, fallbackViewport, activeBinding); ok {
				return activeBinding, rect, true
			}
			return state.TerminalViewBinding{}, render.Rect{}, false
		}
		if activeBinding.TerminalID != "" {
			if binding, ok := root.TerminalViews.OwnerBindingRef(activeBinding.TerminalRef()); ok {
				if rect, ok := terminalViewContentRect(root, fallbackViewport, binding); ok {
					return binding, rect, true
				}
			}
			return state.TerminalViewBinding{}, render.Rect{}, false
		}
	}
	if binding, ok := resizeOwnerBindingForSessionTerminal(root); ok {
		if rect, ok := terminalViewContentRect(root, fallbackViewport, binding); ok {
			return binding, rect, true
		}
	}
	if !root.Session.TerminalRef().Empty() && len(root.TerminalViews.BindingsForTerminalRef(root.Session.TerminalRef())) > 0 {
		// 中文说明：只要 session terminal 已进入 TerminalView 模型，就不能退回全局 session resize；
		// active pane 为空或 owner 被 size lock 锁住时，fallback 会绕开 owner/follower/lock guard。
		return state.TerminalViewBinding{}, render.Rect{}, false
	}
	rect, ok := activeTerminalContentRect(root, fallbackViewport)
	return state.TerminalViewBinding{}, rect, ok
}

func resizeOwnerBindingForSessionTerminal(root state.Root) (state.TerminalViewBinding, bool) {
	ref := root.Session.TerminalRef()
	if ref.Empty() {
		return state.TerminalViewBinding{}, false
	}
	return root.TerminalViews.OwnerBindingRef(ref)
}

func terminalViewContentRect(root state.Root, fallbackViewport render.Rect, binding state.TerminalViewBinding) (render.Rect, bool) {
	plan, ok := terminalLayoutPlan(root, fallbackViewport)
	if !ok {
		return render.Rect{}, false
	}
	if binding.FloatingID != "" {
		for _, layout := range plan.Floatings {
			if layout.Floating.ID == binding.FloatingID && layout.ContentRect.W > 0 && layout.ContentRect.H > 0 {
				return layout.ContentRect, true
			}
		}
	}
	if binding.PaneID != "" {
		for _, panel := range plan.Panels {
			if panel.Panel.ID == binding.PaneID && panel.ContentRect.W > 0 && panel.ContentRect.H > 0 {
				return panel.ContentRect, true
			}
		}
	}
	return render.Rect{}, false
}

func activeTerminalViewBinding(root state.Root) (state.TerminalViewBinding, bool) {
	shell := root.Shell.ReadonlyDefaults()
	if target, ok := shell.ActiveSurfaceTarget(); ok && target.Floating {
		if binding, ok := root.TerminalViews.FloatingBinding(target.FloatingID); ok {
			return binding, true
		}
	}
	return root.TerminalViews.PaneBinding(shell.ActivePaneID)
}

func terminalLayoutMayNeedResize(root state.Root, msg Msg) bool {
	switch msg := msg.(type) {
	case HostResizeMsg,
		LiveAttachResultMsg,
		LiveResizeResultMsg,
		TerminalPoolAttachResultMsg,
		TerminalSizeLockToggleResultMsg,
		ShellWorkbenchCommandMsg,
		ShellSetPanelPresentationMsg,
		ShellTogglePanelPresentationMsg,
		ShellSetHeaderVisibleMsg,
		ShellToggleHeaderVisibleMsg,
		ShellSetFooterVisibleMsg,
		ShellToggleFooterVisibleMsg,
		ShellSplitActivePaneMsg,
		ShellPaneCommandMsg:
		return true
	case LiveEventMsg:
		// 中文说明：普通 refresh 只是 live surface 失效信号；不能因此构造完整 RenderVM 做布局测量。
		return !msg.isOrdinaryRefresh()
	case ShellFloatingCommandMsg:
		return floatingCommandMayResizeTerminal(root, msg.Command) || terminalHasPendingOwnerResize(root)
	default:
		return false
	}
}

func terminalHasPendingOwnerResize(root state.Root) bool {
	for _, binding := range root.TerminalViews.Bindings() {
		if binding.ResizePending && binding.HasAuthoritativeResizeOwner() {
			return true
		}
	}
	return false
}

func floatingCommandMayResizeTerminal(root state.Root, command state.FloatingCommand) bool {
	switch command.Action {
	case state.FloatingCommandCreate,
		state.FloatingCommandCenter,
		state.FloatingCommandToggleCollapse,
		state.FloatingCommandToggleAll,
		state.FloatingCommandShowAll,
		state.FloatingCommandCollapseAll,
		state.FloatingCommandFit,
		state.FloatingCommandToggleAutoFit,
		state.FloatingCommandRefreshAutoFit,
		state.FloatingCommandMove,
		state.FloatingCommandResize:
		return activeFloatingHasTerminal(root)
	default:
		return false
	}
}

func activeFloatingHasTerminal(root state.Root) bool {
	shell := root.Shell.ReadonlyDefaults()
	activeFloatingID := shell.ActiveFloatingID()
	if activeFloatingID == "" {
		return false
	}
	binding, ok := root.TerminalViews.FloatingBinding(activeFloatingID)
	return ok && binding.TerminalID != ""
}

func liveAttachContentSize(root state.Root, cfg LiveConfig) (int, int) {
	rect, ok := activeTerminalContentRect(root, render.Rect{W: cfg.Cols, H: cfg.Rows})
	if !ok {
		return cfg.Cols, cfg.Rows
	}
	return rect.W, rect.H
}

func activeTerminalContentRect(root state.Root, fallbackViewport render.Rect) (render.Rect, bool) {
	plan, ok := terminalLayoutPlan(root, fallbackViewport)
	if !ok {
		return render.Rect{}, false
	}
	if rect, ok := activeFloatingContentRectFromPlan(root, plan, true); ok {
		return rect, true
	}
	activePaneID := root.Shell.ReadonlyDefaults().ActivePaneID
	for _, panel := range plan.Panels {
		if panel.Panel.ID == activePaneID && panel.ContentRect.W > 0 && panel.ContentRect.H > 0 {
			return panel.ContentRect, true
		}
	}
	return render.Rect{}, false
}

func terminalLayoutPlan(root state.Root, fallbackViewport render.Rect) (render.LayoutPlan, bool) {
	if !root.Viewport.Valid {
		if fallbackViewport.W <= 0 || fallbackViewport.H <= 0 {
			return render.LayoutPlan{}, false
		}
		root.Viewport = state.ViewportStore{Valid: true, Cols: fallbackViewport.W, Rows: fallbackViewport.H}
	}
	vm := render.NewRenderVMBuilder().Build(root)
	plan := render.MeasureLayout(vm.Shell, vm.Shell.Layout.Viewport)
	return plan, true
}

func activeFloatingContentRectFromPlan(root state.Root, plan render.LayoutPlan, requireTerminal bool) (render.Rect, bool) {
	shell := root.Shell.ReadonlyDefaults()
	activeFloatingID := shell.ActiveFloatingID()
	if activeFloatingID == "" {
		return render.Rect{}, false
	}
	if requireTerminal {
		binding, ok := root.TerminalViews.FloatingBinding(activeFloatingID)
		if !ok || binding.TerminalID == "" {
			return render.Rect{}, false
		}
	}
	for _, layout := range plan.Floatings {
		if layout.Floating.ID == activeFloatingID && layout.ContentRect.W > 0 && layout.ContentRect.H > 0 {
			return layout.ContentRect, true
		}
	}
	return render.Rect{}, false
}

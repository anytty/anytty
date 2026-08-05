package app

import (
	"context"

	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

// NewTerminalLayoutResizeReducer 把最新 shell/layout state 投影成 active terminal content rect。
// 这里不直接调用 terminal service，而是通过 LiveResizeMsg 回到 live reducer，保持 service IO 不越过 message path。
func NewTerminalLayoutResizeReducer() Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		if !terminalLayoutMayNeedResize(root, msg) || !root.Session.Attached {
			return root, nil
		}
		binding, rect, ok := resizeOwnerTerminalContentRect(root, render.Rect{})
		if !ok {
			return root, nil
		}
		if binding.ViewID != "" {
			nextViews, decision := root.TerminalViews.RequestViewResize(binding.ViewID, rect.W, rect.H)
			if !decision.Allowed || !decision.Changed {
				return root, nil
			}
			root.TerminalViews = nextViews
			root.Session = root.Session.RequestResize(rect.W, rect.H)
			return root, []Effect{FuncEffect{
				Run: func(context.Context) Msg {
					return LiveResizeMsg{EndpointID: binding.EndpointID, TerminalID: binding.TerminalID, Cols: rect.W, Rows: rect.H, Seq: decision.Seq, ViewID: binding.ViewID}
				},
			}}
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

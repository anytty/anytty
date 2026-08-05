package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anytty/anytty/shared/perftrace"
	"github.com/anytty/anytty/shared/terminalmeta"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

const (
	terminalPoolRefreshToken    CancelToken   = "terminal.pool.refresh"
	terminalPoolRefreshInterval time.Duration = time.Second
)

// TerminalPoolListRequestMsg 请求刷新 Terminal Manager 的 inventory 投影。
// Refresh=true 表示后台诊断刷新：只更新资源/连接数/生命周期，不切换 loading 状态，也不让请求消息本身触发渲染。
type TerminalPoolListRequestMsg struct {
	EndpointID state.EndpointID
	Refresh    bool
}

func (TerminalPoolListRequestMsg) isMsg() {}

// SkipRender 避免后台刷新请求产生空 frame；真正的渲染由 list result 或 live surface result 驱动。
func (msg TerminalPoolListRequestMsg) SkipRender() bool {
	return msg.Refresh
}

func terminalPickerListRequestEffect() Effect {
	// 中文说明：Terminal Picker 打开时用静默刷新校准 daemon list；
	// picker 本身先展示 reducer-owned 现有投影，避免普通 list 把列表切到 loading 中间帧。
	return FuncEffect{Run: func(context.Context) Msg { return TerminalPoolListRequestMsg{Refresh: true} }}
}

// TerminalPoolRefreshTickMsg 是 Terminal Picker/Manager 打开期间的周期性刷新 tick。
// 它只回到 reducer 发起一次后台 list，inventory 真值仍来自 terminal service 的下一次 list result。
type TerminalPoolRefreshTickMsg struct{}

func (TerminalPoolRefreshTickMsg) isMsg() {}

// SkipRender 表示 tick 自身不改变 reducer-owned view-model。
func (TerminalPoolRefreshTickMsg) SkipRender() bool {
	return true
}

// TerminalPoolPreviewRefreshMsg 请求重新拉取当前选中 terminal 的 latest native screen。
// 这个消息只负责把 selection/query 变化接到 live surface effect，不读取 history 或本地 scrollback。
type TerminalPoolPreviewRefreshMsg struct{}

func (TerminalPoolPreviewRefreshMsg) isMsg() {}

// SkipRender 表示 preview 刷新请求本身不产生新画面；LiveSurfaceMsg 回投后才更新右侧 preview。
func (TerminalPoolPreviewRefreshMsg) SkipRender() bool {
	return true
}

// TerminalPoolListResultMsg 是 terminal service inventory 回到 TUI reducer 的结果消息。
// Refresh 记录它是否来自后台刷新，用于静默处理错误并继续刷新循环。
type TerminalPoolListResultMsg struct {
	EndpointID state.EndpointID
	Seq        uint64
	Refresh    bool
	Result     port.TerminalListResult
	Err        error
}

func (TerminalPoolListResultMsg) isMsg() {}

// SkipRender 只跳过后台刷新失败 frame；成功刷新需要让资源/连接数变化立即显示。
func (msg TerminalPoolListResultMsg) SkipRender() bool {
	return msg.Refresh && msg.Err != nil
}

type TerminalPoolAttachRequestMsg struct {
	EndpointID       state.EndpointID
	TerminalID       string
	TargetPaneID     string
	TargetFloatingID string
	ResizePolicy     string
}

func (TerminalPoolAttachRequestMsg) isMsg() {}

type TerminalPoolAttachResultMsg struct {
	EndpointID       state.EndpointID
	TerminalID       string
	TargetPaneID     string
	TargetFloatingID string
	ResizePolicy     string
	OperationID      string
	Result           port.TerminalAttachResult
	Err              error
}

func (TerminalPoolAttachResultMsg) isMsg() {}

type TerminalPoolCreateRequestMsg struct {
	EndpointID       state.EndpointID
	Title            string
	Command          []string
	CWD              string
	Tags             map[string]string
	TargetPaneID     string
	TargetFloatingID string
}

func (TerminalPoolCreateRequestMsg) isMsg() {}

type TerminalPoolCreateResultMsg struct {
	EndpointID       state.EndpointID
	RequestedID      string
	TargetPaneID     string
	TargetFloatingID string
	Result           port.TerminalCreateResult
	Err              error
}

func (TerminalPoolCreateResultMsg) isMsg() {}

type TerminalPoolRestartRequestMsg struct {
	EndpointID state.EndpointID
	TerminalID string
}

func (TerminalPoolRestartRequestMsg) isMsg() {}

type TerminalPoolRestartIfExitedRequestMsg struct {
	EndpointID state.EndpointID
	TerminalID string
}

func (TerminalPoolRestartIfExitedRequestMsg) isMsg() {}

type TerminalPoolRestartIfExitedResultMsg struct {
	EndpointID state.EndpointID
	TerminalID string
	Seq        uint64
	Result     port.TerminalListResult
	Err        error
}

func (TerminalPoolRestartIfExitedResultMsg) isMsg() {}

type TerminalPoolRestartResultMsg struct {
	EndpointID state.EndpointID
	TerminalID string
	Err        error
}

func (TerminalPoolRestartResultMsg) isMsg() {}

type TerminalPoolReconnectRequestMsg struct {
	EndpointID       state.EndpointID
	TerminalID       string
	TargetPaneID     string
	TargetFloatingID string
	// LocalError 表示该 reconnect 来自断线 pane 的本地 CTA。
	// 失败时错误必须写回目标 TerminalView，不能升级成 picker/global toast。
	LocalError bool
}

func (TerminalPoolReconnectRequestMsg) isMsg() {}

type TerminalPoolReconnectResultMsg struct {
	EndpointID       state.EndpointID
	TerminalID       string
	TargetPaneID     string
	TargetFloatingID string
	ResizePolicy     string
	OperationID      string
	Result           port.TerminalAttachResult
	Err              error
	// LocalError 沿用 request 的断线 pane 边界，指导 reducer 进行 view-local 错误投影。
	LocalError bool
}

func (TerminalPoolReconnectResultMsg) isMsg() {}

type TerminalPoolKillRequestMsg struct {
	EndpointID     state.EndpointID
	TerminalID     string
	PaneID         string
	FloatingID     string
	CloseOnSuccess bool
}

func (TerminalPoolKillRequestMsg) isMsg() {}

type TerminalPoolKillResultMsg struct {
	EndpointID     state.EndpointID
	TerminalID     string
	PaneID         string
	FloatingID     string
	CloseOnSuccess bool
	Err            error
}

func (TerminalPoolKillResultMsg) isMsg() {}

type TerminalPoolRemoveRequestMsg struct {
	EndpointID state.EndpointID
	TerminalID string
}

func (TerminalPoolRemoveRequestMsg) isMsg() {}

type TerminalPoolRemoveResultMsg struct {
	EndpointID state.EndpointID
	TerminalID string
	Err        error
}

func (TerminalPoolRemoveResultMsg) isMsg() {}

type TerminalPoolEditRequestMsg struct {
	EndpointID state.EndpointID
	TerminalID string
	Title      string
	Tags       map[string]string
}

func (TerminalPoolEditRequestMsg) isMsg() {}

// TerminalPoolEditResultMsg 是 terminal metadata 编辑服务回到 TUI reducer 的确认消息；
// title/tags 来自已提交请求，用于立即更新 Terminal Manager 的本地列表投影，失败时不得改写列表。
type TerminalPoolEditResultMsg struct {
	EndpointID state.EndpointID
	TerminalID string
	Title      string
	Tags       map[string]string
	Err        error
}

func (TerminalPoolEditResultMsg) isMsg() {}

type TerminalSizeLockToggleRequestMsg struct{}

func (TerminalSizeLockToggleRequestMsg) isMsg() {}

type TerminalSizeLockToggleResultMsg struct {
	EndpointID state.EndpointID
	TerminalID string
	Tags       map[string]string
	Locked     bool
	Err        error
}

func (TerminalSizeLockToggleResultMsg) isMsg() {}

func NewTerminalPoolReducer(deps LiveDeps) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		switch msg := msg.(type) {
		case TerminalPoolListRequestMsg:
			return reduceTerminalPoolListRequest(root, msg, deps)
		case TerminalPoolListResultMsg:
			return reduceTerminalPoolListResult(root, msg, deps)
		case TerminalPoolRefreshTickMsg:
			return reduceTerminalPoolRefreshTick(root, deps)
		case TerminalPoolPreviewRefreshMsg:
			return reduceTerminalPoolPreviewRefresh(root, deps)
		case TerminalPoolAttachRequestMsg:
			return reduceTerminalPoolAttachRequest(root, msg, deps)
		case TerminalPoolAttachResultMsg:
			return reduceTerminalPoolAttachResult(root, msg, deps)
		case TerminalPoolCreateRequestMsg:
			return reduceTerminalPoolCreateRequest(root, msg, deps)
		case TerminalPoolCreateResultMsg:
			return reduceTerminalPoolCreateResult(root, msg)
		case TerminalPoolRestartRequestMsg:
			return reduceTerminalPoolRestartRequest(root, msg, deps)
		case TerminalPoolRestartIfExitedRequestMsg:
			return reduceTerminalPoolRestartIfExitedRequest(root, msg, deps)
		case TerminalPoolRestartIfExitedResultMsg:
			return reduceTerminalPoolRestartIfExitedResult(root, msg, deps)
		case TerminalPoolRestartResultMsg:
			return reduceTerminalPoolRestartResult(root, msg, deps)
		case TerminalPoolReconnectRequestMsg:
			return reduceTerminalPoolReconnectRequest(root, msg, deps)
		case TerminalPoolReconnectResultMsg:
			return reduceTerminalPoolReconnectResult(root, msg, deps)
		case TerminalPoolKillRequestMsg:
			return reduceTerminalPoolKillRequest(root, msg, deps)
		case TerminalPoolKillResultMsg:
			return reduceTerminalPoolKillResult(root, msg)
		case TerminalPoolRemoveRequestMsg:
			return reduceTerminalPoolRemoveRequest(root, msg, deps)
		case TerminalPoolRemoveResultMsg:
			return reduceTerminalPoolRemoveResult(root, msg)
		case TerminalPoolEditRequestMsg:
			return reduceTerminalPoolEditRequest(root, msg, deps)
		case TerminalPoolEditResultMsg:
			return reduceTerminalPoolEditResult(root, msg)
		case TerminalSizeLockToggleRequestMsg:
			return reduceTerminalSizeLockToggleRequest(root, deps)
		case TerminalSizeLockToggleResultMsg:
			return reduceTerminalSizeLockToggleResult(root, msg)
		default:
			return root, nil
		}
	}
}

func reduceTerminalPoolListRequest(root state.Root, msg TerminalPoolListRequestMsg, deps LiveDeps) (state.Root, []Effect) {
	if deps.Terminal == nil {
		if msg.Refresh {
			return root, nil
		}
		root.TerminalPool, _ = root.TerminalPool.ApplyList(root.TerminalPool.RequestSeq, nil, "terminal service missing")
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.pool", Body: "terminal service missing"})
		return root.Advance(), nil
	}
	if msg.Refresh {
		root.TerminalPool = root.TerminalPool.RequestRefresh()
	} else {
		root.TerminalPool = root.TerminalPool.RequestList()
	}
	seq := root.TerminalPool.RequestSeq
	if msg.EndpointID == "" && root.Endpoints.HasItems() {
		targets := terminalPoolListEndpointTargets(root.Endpoints)
		if len(targets) == 0 {
			root.TerminalPool, _ = root.TerminalPool.ApplyList(seq, nil, "")
			return root.Advance(), nil
		}
		effects := make([]Effect, 0, len(targets))
		for _, endpointID := range targets {
			effects = append(effects, terminalPoolListEffect(endpointID, seq, msg.Refresh, deps))
		}
		return root.Advance(), effects
	}
	endpointID := state.NormalizeEndpointID(msg.EndpointID)
	return root.Advance(), []Effect{terminalPoolListEffect(endpointID, seq, msg.Refresh, deps)}
}

func terminalPoolListEffect(endpointID state.EndpointID, seq uint64, refresh bool, deps LiveDeps) Effect {
	endpointID = state.NormalizeEndpointID(endpointID)
	return FuncEffect{
		Async:            true,
		ForceSyncInTests: true,
		Run: func(ctx context.Context) Msg {
			finish := perftrace.Measure("tui.terminal_pool.list.effect")
			result, err := deps.Terminal.List(ctx, port.TerminalListRequest{EndpointID: endpointID})
			finish(len(result.Items))
			return TerminalPoolListResultMsg{EndpointID: endpointID, Seq: seq, Refresh: refresh, Result: result, Err: err}
		},
	}
}

func terminalPoolListEndpointTargets(endpoints state.EndpointStore) []state.EndpointID {
	endpoints = endpoints.Normalize()
	targets := make([]state.EndpointID, 0, len(endpoints.Items))
	for _, endpoint := range endpoints.Items {
		status := endpoint.DisplayStatus()
		switch status {
		case state.EndpointStatusDisabled, state.EndpointStatusManual, state.EndpointStatusReconnectRequired, state.EndpointStatusUnregistered:
			continue
		}
		if !endpoint.Enabled {
			continue
		}
		targets = append(targets, endpoint.ID)
	}
	return targets
}

func reduceTerminalPoolListResult(root state.Root, msg TerminalPoolListResultMsg, deps LiveDeps) (state.Root, []Effect) {
	errText := errorString(msg.Err)
	beforeSurfaceTerminal := root.Surface.TerminalID
	beforeSurfaceState := root.Surface.State
	beforeSessionTerminal := root.Session.TerminalID
	beforeSessionState := root.Session.State
	var missingRefs []state.TerminalRef
	if msg.EndpointID != "" {
		if endpoint, ok := root.Endpoints.Endpoint(msg.EndpointID); ok && !endpoint.Enabled {
			return root, nil
		}
		if root.TerminalPool.IsStale(msg.Seq) {
			return root, nil
		}
		items := terminalPoolItemsFromService(msg.Result.Items)
		holdRuntimeError := false
		if msg.Refresh && errText == "" {
			missingRefs = terminalRefsMissingFromEndpointList(root, msg.EndpointID, items)
			missingRefs = removableMissingTerminalRefs(root, missingRefs)
			holdRuntimeError = shouldHoldEndpointRuntimeErrorOnEmptyRefresh(root, msg.EndpointID, items)
		}
		if holdRuntimeError {
			root.TerminalPool.AppliedSeq = msg.Seq
		} else {
			root = root.ApplyEndpointTerminalList(msg.EndpointID, items, errText)
		}
		if errText != "" && root.TerminalPool.Status == state.TerminalPoolLoading {
			root.TerminalPool.Status = state.TerminalPoolReady
		}
		root.TerminalPool.AppliedSeq = msg.Seq
	} else {
		if msg.Refresh && errText != "" {
			return root, terminalPoolRefreshLoopEffects(root)
		}
		items := terminalPoolItemsFromService(msg.Result.Items)
		holdRuntimeError := false
		if msg.Refresh && errText == "" {
			missingRefs = terminalRefsMissingFromEndpointList(root, state.DefaultEndpointID, items)
			missingRefs = removableMissingTerminalRefs(root, missingRefs)
			holdRuntimeError = shouldHoldEndpointRuntimeErrorOnEmptyRefresh(root, state.DefaultEndpointID, items)
		}
		if holdRuntimeError {
			root.TerminalPool.AppliedSeq = msg.Seq
		} else {
			next, applied := root.TerminalPool.ApplyList(msg.Seq, items, errText)
			if !applied {
				return root, nil
			}
			root.TerminalPool = next
		}
	}
	if len(missingRefs) > 0 {
		for _, ref := range missingRefs {
			root = removeTerminalRefFromRoot(root, ref)
		}
	}
	if msg.Refresh && errText != "" {
		return root, terminalPoolRefreshLoopEffects(root)
	}
	logLifecycleTrace(deps.Logger, "terminal.pool.list",
		"seq", msg.Seq,
		"err", errText,
		"items", lifecyclePoolItemsSummary(root.TerminalPool.Items),
		"active_terminal", lifecycleActiveTerminalID(root),
		"surface_terminal", root.Surface.TerminalID,
		"surface_state", string(root.Surface.State),
		"surface_before", fmt.Sprintf("%s:%s", beforeSurfaceTerminal, beforeSurfaceState),
		"session_terminal", root.Session.TerminalID,
		"session_state", string(root.Session.State),
		"session_before", fmt.Sprintf("%s:%s", beforeSessionTerminal, beforeSessionState),
		"bindings", lifecycleTerminalViewsSummary(root.TerminalViews),
	)
	if errText != "" {
		if !msg.Refresh {
			root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.pool", Body: errText})
		}
	}
	effects := terminalPoolRefreshLoopEffects(root)
	effects = append(effects, terminalPoolPreviewRefreshEffects(root, deps)...)
	if len(missingRefs) > 0 {
		effects = append(effects, workbenchPersistEffects("terminal.inventory-missing")...)
	}
	return root.Advance(), effects
}

func reduceTerminalPoolRefreshTick(root state.Root, deps LiveDeps) (state.Root, []Effect) {
	if !terminalInventoryOverlayOpen(root) {
		return root, nil
	}
	return reduceTerminalPoolListRequest(root, TerminalPoolListRequestMsg{Refresh: true}, deps)
}

func reduceTerminalPoolPreviewRefresh(root state.Root, deps LiveDeps) (state.Root, []Effect) {
	if !terminalPoolOverlayOpen(root) {
		return root, nil
	}
	return root, terminalPoolPreviewRefreshEffects(root, deps)
}

func terminalPoolRefreshLoopEffects(root state.Root) []Effect {
	if !terminalInventoryOverlayOpen(root) {
		return nil
	}
	return []Effect{FuncEffect{
		Token: terminalPoolRefreshToken,
		Async: true,
		Run: func(ctx context.Context) Msg {
			timer := time.NewTimer(terminalPoolRefreshInterval)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return nil
			case <-timer.C:
				return TerminalPoolRefreshTickMsg{}
			}
		},
	}}
}

func terminalPoolPreviewRefreshEffects(root state.Root, deps LiveDeps) []Effect {
	if !terminalPoolOverlayOpen(root) {
		return nil
	}
	selected, ok := selectedTerminalPoolPageItem(root)
	if !ok {
		return nil
	}
	if selected.TerminalID == root.TerminalPool.LastRemovedID {
		// 中文说明：remove 成功后若后台 list 暂时返回旧项，preview 不能拉 live surface 复活已清理的 session/surface。
		return nil
	}
	cols, rows := selected.Cols, selected.Rows
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	// 中文说明：Terminal Manager preview 只拉 core latest native screen；
	// 它是实时展示投影，不从 history/copy 或本地 scrollback 推断内容。
	return liveSurfaceEffectForRef(selected.EndpointID, selected.TerminalID, cols, rows, false, deps)
}

func terminalPoolOverlayOpen(root state.Root) bool {
	shell := root.Shell.ReadonlyDefaults()
	return shell.Overlay.Open && shell.Overlay.Kind == state.OverlayTerminalPool
}

func terminalInventoryOverlayOpen(root state.Root) bool {
	shell := root.Shell.ReadonlyDefaults()
	return shell.Overlay.Open && (shell.Overlay.Kind == state.OverlayTerminalPool || shell.Overlay.Kind == state.OverlayTerminalPicker)
}

func selectedTerminalPoolPageItem(root state.Root) (state.TerminalPoolPageItem, bool) {
	items := state.TerminalPoolPageItems(root)
	if len(items) == 0 {
		return state.TerminalPoolPageItem{}, false
	}
	for _, item := range items {
		if item.Selected {
			return item, item.TerminalID != ""
		}
	}
	return items[0], items[0].TerminalID != ""
}

func reduceTerminalPoolAttachRequest(root state.Root, msg TerminalPoolAttachRequestMsg, deps LiveDeps) (state.Root, []Effect) {
	if msg.TerminalID == "" {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.attach", Body: "missing terminal"})
		return root.Advance(), nil
	}
	if deps.Terminal == nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.attach", Body: "terminal service missing"})
		return root.Advance(), nil
	}
	target, ok := terminalPoolTargetFromRequest(root, msg.TargetPaneID, msg.TargetFloatingID)
	if !ok {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.attach", Body: "missing target panel"})
		return root.Advance(), nil
	}
	cols, rows := terminalPoolAttachSizeForTarget(root, target)
	resizePolicy := msg.ResizePolicy
	if resizePolicy == "" {
		resizePolicy = state.TerminalResizeRoleFollower
	}
	endpointID := state.NormalizeEndpointID(msg.EndpointID)
	surfaceID := runtimeSurfaceID(root)
	binding := state.TerminalViewBinding{ViewID: target.ViewID, SurfaceID: surfaceID, EndpointID: endpointID, TerminalID: msg.TerminalID, DesiredCols: cols, DesiredRows: rows, ResizeRole: resizePolicy, PaneID: target.PaneID, FloatingID: target.FloatingID}
	var candidate state.TerminalAttachCandidate
	root.TerminalViews, candidate = root.TerminalViews.BeginAttach(binding)
	return root, []Effect{FuncEffect{
		Run: func(ctx context.Context) Msg {
			finish := perftrace.Measure("tui.terminal_pool.attach.effect")
			result, err := deps.Terminal.Attach(ctx, port.TerminalAttachRequest{
				EndpointID:   endpointID,
				TerminalID:   msg.TerminalID,
				Cols:         cols,
				Rows:         rows,
				Mode:         "collaborator",
				ResizePolicy: resizePolicy,
				SurfaceID:    surfaceID,
				ViewID:       target.ViewID,
				OperationID:  candidate.OperationID,
			})
			finish(0)
			return TerminalPoolAttachResultMsg{EndpointID: endpointID, TerminalID: msg.TerminalID, TargetPaneID: target.PaneID, TargetFloatingID: target.FloatingID, ResizePolicy: resizePolicy, OperationID: candidate.OperationID, Result: result, Err: err}
		},
	}}
}

func reduceTerminalPoolAttachResult(root state.Root, msg TerminalPoolAttachResultMsg, deps LiveDeps) (state.Root, []Effect) {
	errText := errorString(msg.Err)
	endpointID := state.NormalizeEndpointID(msg.EndpointID)
	target, _ := terminalPoolTargetFromRequest(root, msg.TargetPaneID, msg.TargetFloatingID)
	if errText != "" {
		var applied bool
		root.TerminalViews, applied = root.TerminalViews.FailAttach(target.ViewID, msg.OperationID, errText)
		if !applied {
			return root, nil
		}
		ref := state.NewTerminalRef(endpointID, msg.TerminalID)
		root.TerminalPool = root.TerminalPool.ApplyAttachedRef(ref, errText)
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.attach", Body: errText})
		return root.Advance(), nil
	}
	result := msg.Result
	if result.OperationID == "" {
		result.OperationID = msg.OperationID
	}
	if result.EndpointID == "" {
		result.EndpointID = endpointID
	}
	if result.TerminalID == "" {
		result.TerminalID = msg.TerminalID
	}
	ref := state.NewTerminalRef(result.EndpointID, result.TerminalID)
	current, currentOK := root.TerminalViews.Views[target.ViewID]
	if !currentOK || current.AttachCandidate == nil || current.AttachCandidate.OperationID != msg.OperationID || result.OperationID != msg.OperationID {
		return root, cleanupAttachResultEffects(result)
	}
	root.TerminalPool = root.TerminalPool.ApplyAttachedRef(ref, "")
	root.Endpoints = root.Endpoints.MarkRuntimeStatus(ref.EndpointID, state.EndpointStatusConnected, state.EndpointErrorUnknown, endpointTerminalCount(root, ref.EndpointID), "")
	previous := state.TerminalViewBinding{}
	if current.Attached {
		previous = current
	}
	if result.ViewID == "" {
		result.ViewID = target.ViewID
	}
	if result.SurfaceID == "" {
		result.SurfaceID = runtimeSurfaceID(root)
	}
	if shouldPreserveTerminalPoolAttachResizePolicyRef(root, ref, msg.ResizePolicy) {
		result.ResizePolicy = msg.ResizePolicy
		if msg.ResizePolicy != state.TerminalResizeRoleOwner {
			result.CanResize = false
		}
	}
	result = normalizeTerminalAttachResultForLock(root, result)
	root = applyLiveAttachRuntimeProjection(root, result, result.ViewID)
	root = applyTerminalAttachmentProjectionFromAttach(root, result)
	var copyHistoryEffects []Effect
	if msg.TargetFloatingID != "" {
		paneID := msg.TargetPaneID
		if floating, ok := root.Shell.FloatingByID(msg.TargetFloatingID); ok {
			paneID = floating.Pane.ID
		}
		root, copyHistoryEffects = invalidateCopyModeForTerminalRebindRef(root, paneID, result.ViewID, ref)
		binding := state.NewEndpointFloatingTerminalView(result.EndpointID, msg.TargetFloatingID, paneID, result.TerminalID, result.Channel, result.Cols, result.Rows, result.ResizePolicy, result.SurfaceID, result.ViewID, result.CanResize)
		binding.Session = result.Session
		binding.OperationID = result.OperationID
		root.TerminalViews = root.TerminalViews.BindFloating(binding)
		root.TerminalViews = projectTerminalAttachResultLock(root.TerminalViews, result)
		root.Shell = root.Shell.BindFloatingTerminal(msg.TargetFloatingID, result.TerminalID)
	} else {
		root.Shell = root.Shell.EnsureActiveTabForAttach()
		targetPaneID := msg.TargetPaneID
		if targetPaneID == "" {
			targetPaneID = root.Shell.EnsureDefaults().ActivePaneID
		}
		root, copyHistoryEffects = invalidateCopyModeForTerminalRebindRef(root, targetPaneID, result.ViewID, ref)
		root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: targetPaneID}, result.TerminalID)
		binding := state.NewEndpointPaneTerminalView(result.EndpointID, targetPaneID, result.TerminalID, result.Channel, result.Cols, result.Rows, result.ResizePolicy, result.SurfaceID, result.ViewID, result.CanResize)
		binding.Session = result.Session
		binding.OperationID = result.OperationID
		root.TerminalViews = root.TerminalViews.BindPane(binding)
		root.TerminalViews = projectTerminalAttachResultLock(root.TerminalViews, result)
	}
	root.Shell = root.Shell.CloseOverlay().ExitInteractionMode()
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "picker.attach", Body: result.TerminalID})
	effects := append(copyHistoryEffects, workbenchPersistEffects("terminal.attach")...)
	effects = append(effects, liveEffectsForRef(result.EndpointID, result.TerminalID, result.Cols, result.Rows, deps)...)
	effects = appendPreviousAttachmentCleanup(effects, previous, result)
	return root.Advance(), effects
}

func reduceTerminalPoolCreateRequest(root state.Root, msg TerminalPoolCreateRequestMsg, deps LiveDeps) (state.Root, []Effect) {
	if deps.Terminal == nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.new", Body: "terminal service missing"})
		return root.Advance(), nil
	}
	target, ok := terminalPoolTargetFromRequest(root, msg.TargetPaneID, msg.TargetFloatingID)
	if !ok {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.new", Body: "missing target panel"})
		return root.Advance(), nil
	}
	cols, rows := terminalPoolAttachSizeForTarget(root, target)
	title := strings.TrimSpace(msg.Title)
	if title == "" {
		title = nextTerminalPoolID(root)
	}
	endpointID := state.NormalizeEndpointID(msg.EndpointID)
	terminalID := terminalCreateIDFromName(title)
	if terminalNameExists(root, endpointID, title) {
		err := fmt.Sprintf("terminal name %q already exists on endpoint %q", title, endpointID)
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.new", Body: err})
		return root.Advance(), nil
	}
	command := append([]string(nil), msg.Command...)
	if len(command) == 0 {
		// 中文说明：create request 的默认 command 属于目标 endpoint 语义；
		// 不能继承 TUI 进程本地 SHELL，否则 remote daemon 会收到错误机器的命令。
		command = terminalCreateDefaultCommandForEndpoint(root, endpointID)
	}
	if len(command) == 0 {
		err := terminalCreateDefaultsError(root, endpointID)
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.new", Body: err.Error()})
		return root.Advance(), endpointDefaultsRequestEffect(endpointID, false)
	}
	cwd := strings.TrimSpace(msg.CWD)
	tags := cloneStringMap(msg.Tags)
	return root, []Effect{FuncEffect{
		Run: func(ctx context.Context) Msg {
			finish := perftrace.Measure("tui.terminal_pool.create.effect")
			result, err := deps.Terminal.Create(ctx, port.TerminalCreateRequest{
				EndpointID: endpointID,
				TerminalID: terminalID,
				Title:      title,
				Command:    command,
				CWD:        cwd,
				Tags:       tags,
				Cols:       cols,
				Rows:       rows,
			})
			finish(0)
			return TerminalPoolCreateResultMsg{EndpointID: endpointID, RequestedID: terminalID, TargetPaneID: target.PaneID, TargetFloatingID: target.FloatingID, Result: result, Err: err}
		},
	}}
}

func terminalCreateIDFromName(name string) string {
	return strings.TrimSpace(name)
}

func terminalNameExists(root state.Root, endpointID state.EndpointID, name string) bool {
	endpointID = state.NormalizeEndpointID(endpointID)
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, item := range root.TerminalPool.Items {
		if item.TerminalRef().EndpointID != endpointID {
			continue
		}
		if strings.TrimSpace(item.Title) == name || strings.TrimSpace(item.TerminalID) == name {
			return true
		}
	}
	if root.Session.TerminalRef().Equal(state.NewTerminalRef(endpointID, name)) {
		return true
	}
	for _, binding := range root.TerminalViews.BindingsForTerminalRef(state.NewTerminalRef(endpointID, name)) {
		if strings.TrimSpace(binding.TerminalID) == name {
			return true
		}
	}
	return false
}

func reduceTerminalPoolCreateResult(root state.Root, msg TerminalPoolCreateResultMsg) (state.Root, []Effect) {
	finish := perftrace.Measure("tui.terminal_pool.create.apply")
	defer finish(0)
	errText := errorString(msg.Err)
	endpointID := state.NormalizeEndpointID(msg.EndpointID)
	if msg.Result.EndpointID != "" {
		endpointID = state.NormalizeEndpointID(msg.Result.EndpointID)
	}
	result := msg.Result
	if result.EndpointID == "" {
		result.EndpointID = endpointID
	}
	if result.TerminalID == "" {
		result.TerminalID = msg.RequestedID
	}
	root.TerminalPool = root.TerminalPool.ApplyCreatedRef(state.NewTerminalRef(endpointID, result.TerminalID), errText)
	if errText != "" {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.new", Body: errText})
		return root.Advance(), nil
	}
	if result.TerminalID == "" {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.new", Body: "missing terminal"})
		return root.Advance(), nil
	}
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "picker.new", Body: result.TerminalID})
	effects := []Effect{
		FuncEffect{Run: func(context.Context) Msg { return TerminalPoolListRequestMsg{EndpointID: endpointID} }},
		FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolAttachRequestMsg{EndpointID: endpointID, TerminalID: result.TerminalID, TargetPaneID: msg.TargetPaneID, TargetFloatingID: msg.TargetFloatingID, ResizePolicy: state.TerminalResizeRoleOwner}
		}},
	}
	return root.Advance(), effects
}

func reduceTerminalPoolRestartRequest(root state.Root, msg TerminalPoolRestartRequestMsg, deps LiveDeps) (state.Root, []Effect) {
	if msg.TerminalID == "" || deps.Terminal == nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.restart", Body: "terminal unavailable"})
		return root.Advance(), nil
	}
	ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	logLifecycleTrace(deps.Logger, "terminal.restart.request",
		"endpoint_id", string(ref.EndpointID),
		"terminal_id", msg.TerminalID,
		"surface_state", string(root.Surface.SurfaceForTerminalRef(ref).State),
		"session_terminal", root.Session.TerminalID,
		"session_state", string(root.Session.State),
		"bindings", lifecycleTerminalViewBindingsSummary(root.TerminalViews.BindingsForTerminalRef(ref)),
		"input_channels", lifecycleInputChannelsSummary(root.Session.InputChannels),
	)
	return root, []Effect{FuncEffect{Run: func(ctx context.Context) Msg {
		err := deps.Terminal.Restart(ctx, port.TerminalRestartRequest{EndpointID: ref.EndpointID, TerminalID: msg.TerminalID})
		return TerminalPoolRestartResultMsg{EndpointID: ref.EndpointID, TerminalID: msg.TerminalID, Err: err}
	}}}
}

func reduceTerminalPoolRestartIfExitedRequest(root state.Root, msg TerminalPoolRestartIfExitedRequestMsg, deps LiveDeps) (state.Root, []Effect) {
	if msg.TerminalID == "" || deps.Terminal == nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.restart", Body: "terminal unavailable"})
		return root.Advance(), nil
	}
	root.TerminalPool = root.TerminalPool.RequestList()
	seq := root.TerminalPool.RequestSeq
	ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	logLifecycleTrace(deps.Logger, "terminal.restart_if_exited.request",
		"endpoint_id", string(ref.EndpointID),
		"terminal_id", msg.TerminalID,
		"seq", seq,
		"surface_state", string(root.Surface.SurfaceForTerminalRef(ref).State),
		"session_terminal", root.Session.TerminalID,
		"session_state", string(root.Session.State),
		"bindings", lifecycleTerminalViewBindingsSummary(root.TerminalViews.BindingsForTerminalRef(ref)),
	)
	return root.Advance(), []Effect{FuncEffect{Run: func(ctx context.Context) Msg {
		result, err := deps.Terminal.List(ctx, port.TerminalListRequest{EndpointID: ref.EndpointID})
		return TerminalPoolRestartIfExitedResultMsg{EndpointID: ref.EndpointID, TerminalID: msg.TerminalID, Seq: seq, Result: result, Err: err}
	}}}
}

func reduceTerminalPoolRestartIfExitedResult(root state.Root, msg TerminalPoolRestartIfExitedResultMsg, deps LiveDeps) (state.Root, []Effect) {
	ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	if root.TerminalPool.IsStale(msg.Seq) {
		logLifecycleTrace(deps.Logger, "terminal.restart_if_exited.result",
			"endpoint_id", string(ref.EndpointID),
			"terminal_id", msg.TerminalID,
			"seq", msg.Seq,
			"stale", true,
			"request_seq", root.TerminalPool.RequestSeq,
		)
		return root, nil
	}
	next, effects := reduceTerminalPoolListResult(root, TerminalPoolListResultMsg{EndpointID: msg.EndpointID, Seq: msg.Seq, Result: msg.Result, Err: msg.Err}, deps)
	if msg.Err != nil {
		logLifecycleTrace(deps.Logger, "terminal.restart_if_exited.result",
			"endpoint_id", string(ref.EndpointID),
			"terminal_id", msg.TerminalID,
			"seq", msg.Seq,
			"err", msg.Err.Error(),
			"bindings", lifecycleTerminalViewBindingsSummary(next.TerminalViews.BindingsForTerminalRef(ref)),
		)
		return next, effects
	}
	item, ok := terminalPoolItemRef(next.TerminalPool, ref)
	stateValue := ""
	if ok {
		stateValue = item.State
	}
	logLifecycleTrace(deps.Logger, "terminal.restart_if_exited.result",
		"terminal_id", msg.TerminalID,
		"seq", msg.Seq,
		"state", stateValue,
		"found", ok,
		"surface_state", string(next.Surface.SurfaceForTerminalRef(ref).State),
		"session_terminal", next.Session.TerminalID,
		"session_state", string(next.Session.State),
		"bindings", lifecycleTerminalViewBindingsSummary(next.TerminalViews.BindingsForTerminalRef(ref)),
	)
	if !ok || item.State != string(state.TerminalLiveExited) {
		return next, effects
	}
	effects = append(effects, FuncEffect{Run: func(context.Context) Msg {
		return TerminalPoolRestartRequestMsg{EndpointID: ref.EndpointID, TerminalID: msg.TerminalID}
	}})
	return next, effects
}

func reduceTerminalPoolRestartResult(root state.Root, msg TerminalPoolRestartResultMsg, deps LiveDeps) (state.Root, []Effect) {
	ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	if msg.Err != nil {
		root.TerminalPool = root.TerminalPool.ApplyRestartedRef(ref, msg.Err.Error())
		logLifecycleTrace(deps.Logger, "terminal.restart.result",
			"endpoint_id", string(ref.EndpointID),
			"terminal_id", msg.TerminalID,
			"err", msg.Err.Error(),
			"surface_state", string(root.Surface.SurfaceForTerminalRef(ref).State),
			"session_state", string(root.Session.State),
			"bindings", lifecycleTerminalViewBindingsSummary(root.TerminalViews.BindingsForTerminalRef(ref)),
		)
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.restart", Body: msg.Err.Error()})
		return root.Advance(), nil
	}
	beforeBindings := lifecycleTerminalViewBindingsSummary(root.TerminalViews.BindingsForTerminalRef(ref))
	beforeSurface := root.Surface.SurfaceForTerminalRef(ref)
	root.TerminalPool = root.TerminalPool.ApplyRestartedRef(ref, "")
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "picker.restart", Body: msg.TerminalID})
	root.Surface = root.Surface.RestartPreservingContentRef(ref, root.Surface.Cols, root.Surface.Rows)
	root.TerminalViews = root.TerminalViews.MarkTerminalRefReattaching(ref)
	root.Session = root.Session.ClearInputChannelRef(ref)
	logLifecycleTrace(deps.Logger, "terminal.restart.result",
		"terminal_id", msg.TerminalID,
		"surface_before", string(beforeSurface.State),
		"surface_before_exited_at", lifecycleTimeSummary(beforeSurface.ExitedAt),
		"surface_state", string(root.Surface.State),
		"session_state", string(root.Session.State),
		"active_terminal", lifecycleActiveTerminalID(root),
		"bindings_before", beforeBindings,
		"bindings_after", lifecycleTerminalViewBindingsSummary(root.TerminalViews.BindingsForTerminalRef(ref)),
		"input_channels", lifecycleInputChannelsSummary(root.Session.InputChannels),
	)
	effects := []Effect{FuncEffect{Run: func(context.Context) Msg { return TerminalPoolListRequestMsg{EndpointID: ref.EndpointID} }}}
	effects = append(effects, restartTerminalViewEffectsRef(root, ref)...)
	return root.Advance(), effects
}

func reduceTerminalPoolReconnectRequest(root state.Root, msg TerminalPoolReconnectRequestMsg, deps LiveDeps) (state.Root, []Effect) {
	if msg.TerminalID == "" || deps.Terminal == nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.reconnect", Body: "terminal unavailable"})
		return root.Advance(), nil
	}
	target, ok := terminalPoolTargetFromRequest(root, msg.TargetPaneID, msg.TargetFloatingID)
	if !ok {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.reconnect", Body: "missing target panel"})
		return root.Advance(), nil
	}
	cols, rows := terminalPoolAttachSizeForTarget(root, target)
	surfaceID := runtimeSurfaceID(root)
	ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	binding := state.TerminalViewBinding{ViewID: target.ViewID, SurfaceID: surfaceID, EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, DesiredCols: cols, DesiredRows: rows, ResizeRole: state.TerminalResizeRoleFollower, PaneID: target.PaneID, FloatingID: target.FloatingID}
	// 中文说明：连接中的唯一真值是 view attach pending；renderer 直接消费该投影，
	// 不用定时器、toast 或 transport 猜测伪造连接进度。
	var candidate state.TerminalAttachCandidate
	root.TerminalViews, candidate = root.TerminalViews.BeginAttach(binding)
	root.Endpoints = root.Endpoints.MarkRuntimeStatus(ref.EndpointID, state.EndpointStatusConnecting, state.EndpointErrorUnknown, endpointTerminalCount(root, ref.EndpointID), "")
	return root.Advance(), []Effect{FuncEffect{Run: func(ctx context.Context) Msg {
		result, err := deps.Terminal.Reconnect(ctx, port.TerminalReconnectRequest{EndpointID: ref.EndpointID, TerminalID: msg.TerminalID, Cols: cols, Rows: rows, Mode: "collaborator", ResizePolicy: state.TerminalResizeRoleFollower, SurfaceID: surfaceID, ViewID: target.ViewID, OperationID: candidate.OperationID})
		return TerminalPoolReconnectResultMsg{EndpointID: ref.EndpointID, TerminalID: msg.TerminalID, TargetPaneID: target.PaneID, TargetFloatingID: target.FloatingID, ResizePolicy: state.TerminalResizeRoleFollower, OperationID: candidate.OperationID, Result: result, Err: err, LocalError: msg.LocalError}
	}}}
}

func reduceTerminalPoolReconnectResult(root state.Root, msg TerminalPoolReconnectResultMsg, deps LiveDeps) (state.Root, []Effect) {
	if msg.Err != nil {
		target, _ := terminalPoolTargetFromRequest(root, msg.TargetPaneID, msg.TargetFloatingID)
		var applied bool
		root.TerminalViews, applied = root.TerminalViews.FailAttach(target.ViewID, msg.OperationID, msg.Err.Error())
		if !applied {
			return root, nil
		}
		root, effects := reduceTerminalPoolReconnectLocalError(root, msg)
		if !msg.LocalError {
			root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "picker.reconnect", Body: msg.Err.Error()})
			root = root.Advance()
		}
		return root, effects
	}
	return reduceTerminalPoolAttachResult(root, TerminalPoolAttachResultMsg{EndpointID: msg.EndpointID, TerminalID: msg.TerminalID, TargetPaneID: msg.TargetPaneID, TargetFloatingID: msg.TargetFloatingID, ResizePolicy: msg.ResizePolicy, OperationID: msg.OperationID, Result: msg.Result, Err: msg.Err}, deps)
}

func reduceTerminalPoolReconnectLocalError(root state.Root, msg TerminalPoolReconnectResultMsg) (state.Root, []Effect) {
	errText := errorString(msg.Err)
	ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	errorKind := state.ClassifyEndpointErrorText(errText)
	displayMessage := endpointRuntimeDisplayMessage(errorKind, errText)
	target, ok := terminalPoolTargetFromRequest(root, msg.TargetPaneID, msg.TargetFloatingID)
	if !ok {
		return root.Advance(), nil
	}
	root.TerminalPool = root.TerminalPool.ApplyAttachedRef(ref, errText)
	if errorKind != state.EndpointErrorUnknown && errorKind != state.EndpointErrorUnavailable {
		root.Endpoints = root.Endpoints.MarkRuntimeStatus(ref.EndpointID, state.EndpointStatusOffline, errorKind, endpointTerminalCount(root, ref.EndpointID), errText)
	}
	root.TerminalViews = root.TerminalViews.MarkViewRuntimeError(target.ViewID, displayMessage)
	root.Session = root.Session.ClearInputChannelRef(ref)
	root = applyLiveAttachErrorRef(root, ref, target.ViewID, displayMessage)
	return root.Advance(), nil
}

func reduceTerminalPoolKillRequest(root state.Root, msg TerminalPoolKillRequestMsg, deps LiveDeps) (state.Root, []Effect) {
	if msg.TerminalID == "" || deps.Terminal == nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "pool.kill", Body: "terminal unavailable"})
		return root.Advance(), nil
	}
	ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	return root, []Effect{FuncEffect{Run: func(ctx context.Context) Msg {
		err := deps.Terminal.Kill(ctx, port.TerminalKillRequest{EndpointID: ref.EndpointID, TerminalID: msg.TerminalID})
		return TerminalPoolKillResultMsg{EndpointID: ref.EndpointID, TerminalID: msg.TerminalID, PaneID: msg.PaneID, FloatingID: msg.FloatingID, CloseOnSuccess: msg.CloseOnSuccess, Err: err}
	}}}
}

func reduceTerminalPoolKillResult(root state.Root, msg TerminalPoolKillResultMsg) (state.Root, []Effect) {
	errText := errorString(msg.Err)
	ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	root.TerminalPool = root.TerminalPool.ApplyKilledRef(ref, errText)
	if errText != "" {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "pool.kill", Body: errText})
		return root.Advance(), nil
	}
	// 中文说明：kill 只改变 core terminal lifecycle，不代表用户断开 pane/floating view。
	// 绑定清理只能发生在 remove/disconnect 路径，否则已有浮窗会被误删。
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "pool.kill", Body: msg.TerminalID})
	effects := []Effect{FuncEffect{Run: func(context.Context) Msg { return TerminalPoolListRequestMsg{EndpointID: ref.EndpointID} }}}
	if msg.CloseOnSuccess && paneStillOwnsTerminalRef(root, msg.PaneID, ref) {
		effects = append(effects, FuncEffect{Run: func(context.Context) Msg {
			return ShellClosePaneIfTerminalRefMsg{PaneID: msg.PaneID, ExpectedRef: ref}
		}})
	} else if msg.CloseOnSuccess && floatingStillOwnsTerminalRef(root, msg.FloatingID, ref) {
		effects = append(effects, FuncEffect{Run: func(context.Context) Msg {
			return ShellCloseFloatingIfTerminalRefMsg{FloatingID: msg.FloatingID, ExpectedRef: ref}
		}})
	}
	return root.Advance(), effects
}

func paneStillOwnsTerminalRef(root state.Root, paneID string, ref state.TerminalRef) bool {
	if paneID == "" || ref.Empty() || !root.Shell.HasPane(state.PaneCommandTarget{PaneID: paneID}) {
		return false
	}
	binding, ok := root.TerminalViews.PaneBinding(paneID)
	return ok && binding.TerminalRef().Equal(ref)
}

func floatingStillOwnsTerminalRef(root state.Root, floatingID string, ref state.TerminalRef) bool {
	if floatingID == "" || ref.Empty() {
		return false
	}
	if _, ok := root.Shell.FloatingByID(floatingID); !ok {
		return false
	}
	binding, ok := root.TerminalViews.FloatingBinding(floatingID)
	return ok && binding.TerminalRef().Equal(ref)
}

func reduceTerminalPoolRemoveRequest(root state.Root, msg TerminalPoolRemoveRequestMsg, deps LiveDeps) (state.Root, []Effect) {
	if msg.TerminalID == "" || deps.Terminal == nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "pool.delete", Body: "terminal unavailable"})
		return root.Advance(), nil
	}
	ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	return root, []Effect{FuncEffect{Run: func(ctx context.Context) Msg {
		err := deps.Terminal.Remove(ctx, port.TerminalRemoveRequest{EndpointID: ref.EndpointID, TerminalID: msg.TerminalID})
		return TerminalPoolRemoveResultMsg{EndpointID: ref.EndpointID, TerminalID: msg.TerminalID, Err: err}
	}}}
}

func reduceTerminalPoolRemoveResult(root state.Root, msg TerminalPoolRemoveResultMsg) (state.Root, []Effect) {
	errText := errorString(msg.Err)
	ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	root.TerminalPool = root.TerminalPool.ApplyRemovedRef(ref, errText)
	if errText != "" {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "pool.delete", Body: errText})
		return root.Advance(), nil
	}
	root = removeTerminalRefFromRoot(root, ref)
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "pool.delete", Body: msg.TerminalID})
	effects := workbenchPersistEffects("terminal.delete")
	effects = append(effects, FuncEffect{Run: func(context.Context) Msg { return TerminalPoolListRequestMsg{EndpointID: ref.EndpointID} }})
	return root.Advance(), effects
}

func reduceTerminalPoolEditRequest(root state.Root, msg TerminalPoolEditRequestMsg, deps LiveDeps) (state.Root, []Effect) {
	if msg.TerminalID == "" || deps.Terminal == nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "pool.edit", Body: "terminal unavailable"})
		return root.Advance(), nil
	}
	title := msg.Title
	if title == "" {
		title = msg.TerminalID
	}
	tags := cloneStringMap(msg.Tags)
	ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	return root, []Effect{FuncEffect{Run: func(ctx context.Context) Msg {
		err := deps.Terminal.EditMetadata(ctx, port.TerminalEditMetadataRequest{EndpointID: ref.EndpointID, TerminalID: msg.TerminalID, Title: title, Tags: tags})
		return TerminalPoolEditResultMsg{EndpointID: ref.EndpointID, TerminalID: msg.TerminalID, Title: title, Tags: cloneStringMap(tags), Err: err}
	}}}
}

func reduceTerminalPoolEditResult(root state.Root, msg TerminalPoolEditResultMsg) (state.Root, []Effect) {
	errText := errorString(msg.Err)
	ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	root.TerminalPool = root.TerminalPool.ApplyEditedRef(ref, msg.Title, msg.Tags, errText)
	if errText != "" {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "pool.edit", Body: errText})
		return root.Advance(), nil
	}
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "pool.edit", Body: msg.TerminalID})
	return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg { return TerminalPoolListRequestMsg{EndpointID: ref.EndpointID} }}}
}

func reduceTerminalSizeLockToggleRequest(root state.Root, deps LiveDeps) (state.Root, []Effect) {
	if deps.Terminal == nil {
		root.Shell = root.Shell.EnsureDefaults().AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.size", Body: "terminal service missing"})
		return root.Advance(), nil
	}
	target, ok := activeTerminalSizeLockTarget(root)
	if !ok {
		root.Shell = root.Shell.EnsureDefaults().AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.size", Body: "no active terminal"})
		return root.Advance(), nil
	}
	ref := state.NewTerminalRef(target.EndpointID, target.TerminalID)
	tags, ok := terminalPoolTagsRef(root.TerminalPool, ref)
	if !ok {
		// 中文说明：size-lock 的写入真值是 terminal metadata tags；本地 cache 缺失时必须先拉最新 tags，
		// 再合并写入锁标记，不能停在 pending，也不能用空 map 覆盖用户已有 tags。
		return root, []Effect{FuncEffect{Run: func(ctx context.Context) Msg {
			result, err := deps.Terminal.List(ctx, port.TerminalListRequest{EndpointID: ref.EndpointID})
			if err != nil {
				return TerminalSizeLockToggleResultMsg{EndpointID: ref.EndpointID, TerminalID: target.TerminalID, Err: fmt.Errorf("terminal metadata: %w", err)}
			}
			tags, ok := terminalListTagsRef(result, ref)
			if !ok {
				return TerminalSizeLockToggleResultMsg{EndpointID: ref.EndpointID, TerminalID: target.TerminalID, Err: fmt.Errorf("terminal metadata missing")}
			}
			locked := !terminalmeta.SizeLocked(tags)
			if locked {
				tags[terminalmeta.SizeLockTag] = terminalmeta.SizeLockLock
			} else {
				delete(tags, terminalmeta.SizeLockTag)
			}
			err = deps.Terminal.EditTags(ctx, port.TerminalEditTagsRequest{EndpointID: ref.EndpointID, TerminalID: target.TerminalID, Tags: tags})
			return TerminalSizeLockToggleResultMsg{EndpointID: ref.EndpointID, TerminalID: target.TerminalID, Tags: tags, Locked: locked, Err: err}
		}}}
	}
	locked := !terminalmeta.SizeLocked(tags)
	if locked {
		tags[terminalmeta.SizeLockTag] = terminalmeta.SizeLockLock
	} else {
		delete(tags, terminalmeta.SizeLockTag)
	}
	return root, []Effect{FuncEffect{Run: func(ctx context.Context) Msg {
		err := deps.Terminal.EditTags(ctx, port.TerminalEditTagsRequest{EndpointID: ref.EndpointID, TerminalID: target.TerminalID, Tags: tags})
		return TerminalSizeLockToggleResultMsg{EndpointID: ref.EndpointID, TerminalID: target.TerminalID, Tags: tags, Locked: locked, Err: err}
	}}}
}

func reduceTerminalSizeLockToggleResult(root state.Root, msg TerminalSizeLockToggleResultMsg) (state.Root, []Effect) {
	errText := errorString(msg.Err)
	ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	root.TerminalPool = root.TerminalPool.ApplyTagsEditedRef(ref, msg.Tags, errText)
	if errText != "" {
		root.Shell = root.Shell.EnsureDefaults().AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.size", Body: errText})
		return root.Advance(), nil
	}
	root.TerminalViews = root.TerminalViews.ApplyTerminalRefSizeLock(ref, msg.Locked)
	body := "terminal size lock disabled"
	if msg.Locked {
		body = "terminal size is locked"
	}
	root.Shell = root.Shell.EnsureDefaults().AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "terminal.size", Body: body})
	return root.Advance(), nil
}

func terminalListTagsRef(result port.TerminalListResult, ref state.TerminalRef) (map[string]string, bool) {
	ref = ref.Normalize()
	for _, item := range result.Items {
		if !state.NewTerminalRef(item.EndpointID, item.TerminalID).Equal(ref) {
			continue
		}
		tags := cloneStringMap(item.Tags)
		if tags == nil {
			tags = map[string]string{}
		}
		return tags, true
	}
	return nil, false
}

type terminalSizeLockTarget struct {
	EndpointID state.EndpointID
	TerminalID string
}

type terminalPoolTarget struct {
	PaneID     string
	FloatingID string
	ViewID     string
}

func terminalPoolTargetFromRequest(root state.Root, paneID string, floatingID string) (terminalPoolTarget, bool) {
	shell := root.Shell.EnsureDefaults()
	if floatingID != "" {
		paneID = ""
		if floating, ok := shell.FloatingByID(floatingID); ok {
			paneID = floating.Pane.ID
		}
		return terminalPoolTarget{PaneID: paneID, FloatingID: floatingID, ViewID: root.TerminalViews.FloatingViewID(floatingID)}, true
	}
	if paneID == "" {
		paneID = shell.ActivePaneID
	}
	if paneID == "" {
		return terminalPoolTarget{}, false
	}
	return terminalPoolTarget{PaneID: paneID, ViewID: root.TerminalViews.PaneViewID(paneID)}, true
}

func activeTerminalSizeLockTarget(root state.Root) (terminalSizeLockTarget, bool) {
	if binding, ok := activeTerminalViewBinding(root); ok && binding.TerminalID != "" {
		if binding.HasResizeOwner() {
			return terminalSizeLockTarget{EndpointID: binding.EndpointID, TerminalID: binding.TerminalID}, true
		}
		return terminalSizeLockTarget{}, false
	}
	return terminalSizeLockTarget{}, false
}

func terminalPoolTagsRef(pool state.TerminalPoolStore, ref state.TerminalRef) (map[string]string, bool) {
	ref = ref.Normalize()
	for _, item := range pool.Items {
		if item.TerminalRef().Equal(ref) {
			tags := cloneStringMap(item.Tags)
			if tags == nil {
				tags = map[string]string{}
			}
			return tags, true
		}
	}
	return nil, false
}

func normalizeTerminalAttachResultForLock(root state.Root, result port.TerminalAttachResult) port.TerminalAttachResult {
	if terminalAttachResultSizeLocked(root, result) {
		// 中文说明：terminal size lock 是 terminal 级最高优先级；attach result 即使返回 owner/canResize，
		// 也不能冲掉 metadata 或已有 binding 上的锁，否则新 pane attach 会用自己的尺寸改 PTY。
		result.SizeLocked = true
		result.CanResize = false
		result.ControlReason = "size_locked"
		if result.ResizePolicy == "" {
			result.ResizePolicy = state.TerminalResizeRoleOwner
		}
	}
	return result
}

func projectTerminalAttachResultLock(store state.TerminalViewStore, result port.TerminalAttachResult) state.TerminalViewStore {
	if !result.SizeLocked {
		return store
	}
	return store.ApplyTerminalRefSizeLock(state.NewTerminalRef(result.EndpointID, result.TerminalID), true)
}

func terminalAttachResultSizeLocked(root state.Root, result port.TerminalAttachResult) bool {
	terminalID := result.TerminalID
	if terminalID == "" {
		return result.SizeLocked
	}
	if result.SizeLocked {
		return true
	}
	ref := state.NewTerminalRef(result.EndpointID, terminalID)
	for _, binding := range root.TerminalViews.BindingsForTerminalRef(ref) {
		if binding.SizeLocked {
			return true
		}
	}
	tags, ok := terminalPoolTagsRef(root.TerminalPool, ref)
	return ok && terminalmeta.SizeLocked(tags)
}

func shouldPreserveTerminalPoolAttachResizePolicyRef(root state.Root, ref state.TerminalRef, resizePolicy string) bool {
	if resizePolicy == "" {
		return false
	}
	if resizePolicy == state.TerminalResizeRoleOwner {
		return true
	}
	// 中文说明：同 terminal 已有本地 binding 时，picker/reconnect attach 是新增 follower view，
	// 不能因为 core 旧 owner 状态或 auto-owner 结果让新 pane 抢走 resize authority。
	return len(root.TerminalViews.BindingsForTerminalRef(ref)) > 0
}

func terminalPoolItemsFromService(items []port.TerminalPoolItem) []state.TerminalPoolItem {
	out := make([]state.TerminalPoolItem, len(items))
	for i, item := range items {
		out[i] = state.TerminalPoolItem{
			EndpointID:      item.EndpointID,
			TerminalID:      item.TerminalID,
			Title:           item.Title,
			State:           item.State,
			CWD:             item.CWD,
			Command:         append([]string(nil), item.Command...),
			Tags:            cloneStringMap(item.Tags),
			ExitCode:        cloneIntPointer(item.ExitCode),
			ExitedAt:        item.ExitedAt,
			Cols:            item.Cols,
			Rows:            item.Rows,
			AttachmentCount: item.AttachmentCount,
			Resources: state.TerminalResourceUsage{
				PID:            item.Resources.PID,
				CPUPercentX100: item.Resources.CPUPercentX100,
				MemoryBytes:    item.Resources.MemoryBytes,
				SampledAt:      item.Resources.SampledAt,
			},
		}
	}
	return out
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func lifecyclePoolItemsSummary(items []state.TerminalPoolItem) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if item.TerminalID == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s:%s", item.TerminalID, item.State))
	}
	return strings.Join(parts, ",")
}

func terminalPoolItemRef(pool state.TerminalPoolStore, ref state.TerminalRef) (state.TerminalPoolItem, bool) {
	ref = ref.Normalize()
	if ref.Empty() {
		return state.TerminalPoolItem{}, false
	}
	for _, item := range pool.Items {
		if item.TerminalRef().Equal(ref) {
			return item, true
		}
	}
	return state.TerminalPoolItem{}, false
}

func terminalRefsMissingFromEndpointList(root state.Root, endpointID state.EndpointID, items []state.TerminalPoolItem) []state.TerminalRef {
	endpointID = state.NormalizeEndpointID(endpointID)
	present := map[string]struct{}{}
	for _, item := range items {
		itemRef := state.NewTerminalRef(item.EndpointID, item.TerminalID)
		if item.EndpointID == "" {
			itemRef = state.NewTerminalRef(endpointID, item.TerminalID)
		}
		if itemRef.Empty() || itemRef.EndpointID != endpointID {
			continue
		}
		present[itemRef.Key()] = struct{}{}
	}
	candidates := map[string]state.TerminalRef{}
	addCandidate := func(ref state.TerminalRef) {
		ref = ref.Normalize()
		if ref.Empty() || ref.EndpointID != endpointID {
			return
		}
		if _, ok := present[ref.Key()]; ok {
			return
		}
		candidates[ref.Key()] = ref
	}
	for _, binding := range root.TerminalViews.Bindings() {
		addCandidate(binding.TerminalRef())
	}
	addCandidate(root.Session.TerminalRef())
	addCandidate(root.Surface.TerminalRef())
	for _, snapshot := range root.Surface.Surfaces {
		addCandidate(snapshot.TerminalRef())
	}
	if root.History.TerminalID != "" {
		addCandidate(state.NewTerminalRef(root.History.EndpointID, root.History.TerminalID))
	}
	if root.CopyMode.TerminalID != "" {
		addCandidate(state.NewTerminalRef(root.CopyMode.EndpointID, root.CopyMode.TerminalID))
	}
	for _, history := range root.HistoryByView {
		if history.TerminalID != "" {
			addCandidate(state.NewTerminalRef(history.EndpointID, history.TerminalID))
		}
	}
	for _, copyMode := range root.CopyModeByView {
		if copyMode.TerminalID != "" {
			addCandidate(state.NewTerminalRef(copyMode.EndpointID, copyMode.TerminalID))
		}
	}
	out := make([]state.TerminalRef, 0, len(candidates))
	for _, ref := range candidates {
		out = append(out, ref)
	}
	return out
}

func shouldHoldEndpointRuntimeErrorOnEmptyRefresh(root state.Root, endpointID state.EndpointID, items []state.TerminalPoolItem) bool {
	// 中文说明：后台 inventory 刷新只是 endpoint 下 terminal 列表的投影，不是
	// endpoint lifecycle truth。若 watcher 已经标记 transport/daemon 断线，空列表
	// 很可能是断线竞态里的后到消息，不能把 endpoint 恢复成 connected，也不能删除 pane 的连接意图。
	if len(items) > 0 {
		return false
	}
	return endpointHasRuntimeError(root, endpointID)
}

func removableMissingTerminalRefs(root state.Root, refs []state.TerminalRef) []state.TerminalRef {
	if len(refs) == 0 {
		return nil
	}
	out := make([]state.TerminalRef, 0, len(refs))
	for _, ref := range refs {
		if terminalRefHasEndpointRuntimeError(root, ref) {
			continue
		}
		out = append(out, ref)
	}
	return out
}

func terminalRefHasEndpointRuntimeError(root state.Root, ref state.TerminalRef) bool {
	ref = ref.Normalize()
	if ref.Empty() {
		return false
	}
	if endpointHasRuntimeError(root, ref.EndpointID) {
		return true
	}
	for _, binding := range root.TerminalViews.BindingsForTerminalRef(ref) {
		if endpointRuntimeErrorText(binding.LastError) {
			return true
		}
	}
	surface := root.Surface.SurfaceForTerminalRef(ref)
	if surface.State == state.TerminalLiveError && endpointRuntimeErrorText(surface.Err) {
		return true
	}
	if root.Session.TerminalRef().Equal(ref) && root.Session.State == state.TerminalLiveError && endpointRuntimeErrorText(root.Session.LastError) {
		return true
	}
	return false
}

func endpointHasRuntimeError(root state.Root, endpointID state.EndpointID) bool {
	endpointID = state.NormalizeEndpointID(endpointID)
	if endpoint, ok := root.Endpoints.DisplayEndpoint(endpointID); ok {
		if endpoint.DisplayStatus() == state.EndpointStatusOffline && (endpoint.LastError != "" || endpoint.LastErrorKind != state.EndpointErrorUnknown) {
			return true
		}
		if endpointRuntimeErrorText(endpoint.DisplayErrorLabel()) {
			return true
		}
	}
	for _, binding := range root.TerminalViews.Bindings() {
		if state.NormalizeEndpointID(binding.EndpointID) == endpointID && endpointRuntimeErrorText(binding.LastError) {
			return true
		}
	}
	if root.Session.TerminalRef().EndpointID == endpointID && root.Session.State == state.TerminalLiveError && endpointRuntimeErrorText(root.Session.LastError) {
		return true
	}
	if root.Surface.TerminalRef().EndpointID == endpointID && root.Surface.State == state.TerminalLiveError && endpointRuntimeErrorText(root.Surface.Err) {
		return true
	}
	return false
}

func endpointRuntimeErrorText(message string) bool {
	kind := state.ClassifyEndpointErrorText(message)
	return kind != state.EndpointErrorUnknown && kind != state.EndpointErrorUnavailable
}

func restartTerminalViewEffectsRef(root state.Root, ref state.TerminalRef) []Effect {
	ref = ref.Normalize()
	if ref.Empty() {
		return nil
	}
	bindings := root.TerminalViews.BindingsForTerminalRef(ref)
	if len(bindings) == 0 {
		return nil
	}
	effects := make([]Effect, 0, len(bindings))
	for _, binding := range bindings {
		cols, rows := binding.DesiredCols, binding.DesiredRows
		if cols <= 0 || rows <= 0 {
			surface := root.Surface.SurfaceForTerminalRef(ref)
			cols, rows = surface.Cols, surface.Rows
		}
		if cols <= 0 {
			cols = 80
		}
		if rows <= 0 {
			rows = 24
		}
		resizePolicy := binding.ResizeRole
		if !binding.HasAuthoritativeResizeOwner() {
			resizePolicy = state.TerminalResizeRoleFollower
		}
		surfaceID := binding.SurfaceID
		if surfaceID == "" {
			surfaceID = runtimeSurfaceID(root)
		}
		cfg := LiveConfig{
			EndpointID:   ref.EndpointID,
			TerminalID:   ref.TerminalID,
			Cols:         cols,
			Rows:         rows,
			Mode:         "collaborator",
			ResizePolicy: resizePolicy,
			SurfaceID:    surfaceID,
			ViewID:       binding.ViewID,
		}
		cfgCopy := cfg
		effects = append(effects, FuncEffect{Run: func(context.Context) Msg { return LiveAttachMsg{Config: cfgCopy} }})
	}
	return effects
}

func removeTerminalRefFromRoot(root state.Root, ref state.TerminalRef) state.Root {
	ref = ref.Normalize()
	root.Shell = root.Shell.RemoveTerminalRefBindings(ref, root.TerminalViews)
	root.TerminalViews = root.TerminalViews.RemoveTerminalRef(ref)
	root.Session = root.Session.RemoveTerminalRef(ref)
	root.Surface = root.Surface.RemoveTerminalRef(ref)
	return root.WithoutCopyHistorySessionsForTerminalRef(ref)
}

func terminalPoolAttachSizeForTarget(root state.Root, target terminalPoolTarget) (int, int) {
	if rect, ok := terminalPoolTargetContentRect(root, target, render.Rect{}); ok {
		return rect.W, rect.H
	}
	cols, rows := liveAttachContentSize(root, LiveConfig{Cols: root.Session.Cols, Rows: root.Session.Rows})
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	return cols, rows
}

func terminalPoolTargetContentRect(root state.Root, target terminalPoolTarget, fallbackViewport render.Rect) (render.Rect, bool) {
	plan, ok := terminalLayoutPlan(root, fallbackViewport)
	if !ok {
		return render.Rect{}, false
	}
	if target.FloatingID != "" {
		for _, layout := range plan.Floatings {
			if layout.Floating.ID == target.FloatingID && layout.ContentRect.W > 0 && layout.ContentRect.H > 0 {
				return layout.ContentRect, true
			}
		}
		return render.Rect{}, false
	}
	for _, panel := range plan.Panels {
		if panel.Panel.ID == target.PaneID && panel.ContentRect.W > 0 && panel.ContentRect.H > 0 {
			return panel.ContentRect, true
		}
	}
	return render.Rect{}, false
}

func nextTerminalPoolID(root state.Root) string {
	used := map[string]struct{}{}
	for _, item := range root.TerminalPool.Items {
		used[item.TerminalID] = struct{}{}
	}
	if root.Session.TerminalID != "" {
		used[root.Session.TerminalID] = struct{}{}
	}
	// 中文说明：terminal id 也是 linehist 文件名的一部分。新建 terminal 不能
	// 只按当前 pool 数量复用 term-pool-1/2，否则 daemon 重启后会撞上旧大历史文件。
	base := fmt.Sprintf("term-pool-%d", time.Now().UTC().UnixNano())
	for i := 0; ; i++ {
		id := base
		if i > 0 {
			id = fmt.Sprintf("%s-%d", base, i)
		}
		if _, ok := used[id]; !ok {
			return id
		}
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

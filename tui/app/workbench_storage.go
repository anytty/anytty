package app

import (
	"context"
	"errors"
	"log/slog"
	"sort"

	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

type WorkbenchDeps struct {
	Storage port.WorkbenchStorageService
	Ref     state.WorkbenchStorageRef
	Logger  *slog.Logger
	// root 空 terminal 启动时不能用旧 workbench snapshot 恢复连接意图。
	SkipInitialLoad bool
}

const (
	workbenchStorageLoadToken = CancelToken("workbench.storage.load")
	workbenchStorageSaveToken = CancelToken("workbench.storage.save")
)

type WorkbenchStorageLoadRequestMsg struct{}

func (WorkbenchStorageLoadRequestMsg) isMsg() {}

type WorkbenchStorageWatchRequestMsg struct{}

func (WorkbenchStorageWatchRequestMsg) isMsg() {}

type WorkbenchStorageChangedMsg struct {
	Event port.WorkbenchStorageEvent
	Err   error
}

func (WorkbenchStorageChangedMsg) isMsg() {}

type WorkbenchStorageLoadResultMsg struct {
	Result port.WorkbenchStorageLoadResult
	Err    error
}

func (WorkbenchStorageLoadResultMsg) isMsg() {}

type WorkbenchStoragePersistRequestMsg struct {
	Reason string
}

func (WorkbenchStoragePersistRequestMsg) isMsg() {}

type WorkbenchStoragePersistResultMsg struct {
	Result          port.WorkbenchStorageSaveResult
	Err             error
	Reason          string
	ExpectedVersion uint64
}

func (WorkbenchStoragePersistResultMsg) isMsg() {}

// workbench storage 是 pane/floating 到 terminal 连接关系的持久来源；
// attach、kill、layout 这类本地 view 变更都通过完整 snapshot 委托给 core opaque storage。
func workbenchPersistEffects(reason string) []Effect {
	return []Effect{FuncEffect{Run: func(context.Context) Msg {
		return WorkbenchStoragePersistRequestMsg{Reason: reason}
	}}}
}

func NewWorkbenchStorageReducer(deps WorkbenchDeps) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		switch msg := msg.(type) {
		case WorkbenchStorageWatchRequestMsg:
			return reduceWorkbenchStorageWatchRequest(root, deps)
		case WorkbenchStorageChangedMsg:
			return reduceWorkbenchStorageChanged(root, msg)
		case WorkbenchStorageLoadRequestMsg:
			return reduceWorkbenchStorageLoadRequest(root, deps)
		case WorkbenchStorageLoadResultMsg:
			return reduceWorkbenchStorageLoadResult(root, msg, deps)
		case WorkbenchStoragePersistRequestMsg:
			return reduceWorkbenchStoragePersistRequest(root, msg, deps)
		case WorkbenchStoragePersistResultMsg:
			return reduceWorkbenchStoragePersistResult(root, msg)
		default:
			return root, nil
		}
	}
}

func reduceWorkbenchStorageWatchRequest(root state.Root, deps WorkbenchDeps) (state.Root, []Effect) {
	if deps.Storage == nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.storage", Body: "storage service missing"})
		return root.Advance(), nil
	}
	ref := workbenchStorageRef(root, deps)
	return root, []Effect{StreamEffect{Token: CancelToken("workbench.storage.watch"), Run: func(ctx context.Context, post func(Msg)) {
		events, err := deps.Storage.WatchWorkbench(ctx, ref)
		if err != nil {
			logEffectError(deps.Logger, "workbench.storage.watch", err, "key", ref.Key, "owner_id", ref.OwnerID)
			if isContextLifecycleError(err) {
				return
			}
			post(WorkbenchStorageChangedMsg{Err: err})
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				post(WorkbenchStorageChangedMsg{Event: event})
			}
		}
	}}}
}

func reduceWorkbenchStorageChanged(root state.Root, msg WorkbenchStorageChangedMsg) (state.Root, []Effect) {
	if msg.Err != nil {
		if isContextLifecycleError(msg.Err) {
			return root, nil
		}
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.storage", Body: errorString(msg.Err)})
		return root.Advance(), nil
	}
	if root.WorkbenchSync.ShouldIgnoreEvent(msg.Event.Version) {
		root.WorkbenchSync = root.WorkbenchSync.MarkEvent(msg.Event.Version)
		return root.Advance(), nil
	}
	root.WorkbenchSync = root.WorkbenchSync.MarkEvent(msg.Event.Version)
	return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg {
		return WorkbenchStorageLoadRequestMsg{}
	}}}
}

func reduceWorkbenchStorageLoadRequest(root state.Root, deps WorkbenchDeps) (state.Root, []Effect) {
	if deps.Storage == nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.storage", Body: "storage service missing"})
		return root.Advance(), nil
	}
	ref := workbenchStorageRef(root, deps)
	return root, []Effect{FuncEffect{Token: workbenchStorageLoadToken, Async: true, ForceSyncInTests: true, Run: func(ctx context.Context) Msg {
		result, err := deps.Storage.LoadWorkbench(ctx, ref)
		logEffectError(deps.Logger, "workbench.storage.load", err, "key", ref.Key, "owner_id", ref.OwnerID)
		if isContextLifecycleError(err) {
			return nil
		}
		return WorkbenchStorageLoadResultMsg{Result: result, Err: err}
	}}}
}

func reduceWorkbenchStorageLoadResult(root state.Root, msg WorkbenchStorageLoadResultMsg, deps WorkbenchDeps) (state.Root, []Effect) {
	if msg.Err != nil {
		if isContextLifecycleError(msg.Err) {
			return root, nil
		}
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.storage", Body: errorString(msg.Err)})
		return root.Advance(), nil
	}
	if !msg.Result.Found {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "workbench.storage", Body: "empty"})
		return root.Advance(), nil
	}
	if msg.Result.Version != 0 && msg.Result.Version == root.WorkbenchSync.LastAppliedVersion {
		return root, nil
	}
	shell, err := msg.Result.Snapshot.ToShellStore()
	if err != nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.storage", Body: errorString(err)})
		return root.Advance(), nil
	}
	terminalViews, err := msg.Result.Snapshot.ToTerminalViewStore()
	if err != nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.storage", Body: errorString(err)})
		return root.Advance(), nil
	}
	previousViews := root.TerminalViews
	previousHistoryEndpointID := root.History.EndpointID
	previousHistoryTerminalID := root.History.TerminalID
	terminalViews = preserveWorkbenchRuntimeTerminalViews(previousViews, terminalViews)
	terminalViews = terminalViews.ApplyWorkbenchEndpointResolution(root.Endpoints)
	// 中文说明：storage 不同步 floating 显示态；新 TUI 没有本地 rect 时，
	// 在当前 viewport 内按绑定 terminal 的 size 生成本地初始布局，再允许本地状态覆盖。
	shell = shell.NormalizeRestoredFloatingDisplay(root.Viewport, terminalViews)
	// 外部 workbench snapshot 会整体替换 pane/view 结构；旧 frozen history、
	// pending request 和 copy 绑定都不能跨这次替换继续复用。
	root = root.ClearCopyHistorySessions()
	root.History = state.HistoryStore{EndpointID: previousHistoryEndpointID, TerminalID: previousHistoryTerminalID}
	preserveLocalOperation := root.WorkbenchSync.LastAppliedVersion != 0 || root.WorkbenchSync.LastSavedVersion != 0
	root.Shell = mergeLocalWorkbenchRuntimeState(root.Shell, shell, preserveLocalOperation)
	root.Shell = applyConfiguredShellChrome(root.Shell, root.Config)
	root.TerminalViews = terminalViews
	root.WorkbenchSync = root.WorkbenchSync.MarkApplied(msg.Result.Version)
	restoredAttachBindings := workbenchRestoredTerminalAttachBindings(previousViews, root.TerminalViews.Bindings())
	logLifecycleTrace(deps.Logger, "workbench.restore",
		"version", msg.Result.Version,
		"active_pane", root.Shell.ActivePaneID,
		"active_floating", root.Shell.ActiveFloatingID(),
		"active_pane_kind", lifecycleActivePaneKind(root),
		"active_terminal", lifecycleActiveTerminalID(root),
		"terminal_views", len(root.TerminalViews.Views),
		"bindings", lifecycleTerminalViewsSummary(root.TerminalViews),
		"previous_bindings", lifecycleTerminalViewsSummary(previousViews),
		"reattach_bindings", lifecycleTerminalViewBindingsSummary(restoredAttachBindings),
		"surface_terminal", root.Surface.TerminalID,
		"surface_state", string(root.Surface.State),
		"session_terminal", root.Session.TerminalID,
		"session_state", string(root.Session.State),
	)
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "workbench.storage", Body: "loaded"})
	if len(root.TerminalViews.Views) > 0 {
		effects := []Effect{FuncEffect{Run: func(context.Context) Msg { return TerminalPoolListRequestMsg{} }}}
		effects = append(effects, workbenchRestoredTerminalLifecycleQueryEffects(previousViews, root.TerminalViews.Bindings())...)
		effects = append(effects, workbenchRestoredTerminalAttachEffectsForBindings(root, restoredAttachBindings)...)
		return root.Advance(), effects
	}
	return root.Advance(), nil
}

func mergeLocalWorkbenchRuntimeState(previous state.ShellStore, restored state.ShellStore, preserveLocalOperation bool) state.ShellStore {
	previous = previous.ReadonlyDefaults()
	restored = restored.EnsureDefaults()
	restored = mergeLocalWorkbenchFloatingState(previous, restored)
	if preserveLocalOperation {
		restored = preserveLocalActiveWorkspace(previous, restored)
		restored = preserveLocalActiveTabAndPane(previous, restored)
		restored.ZoomedPaneID = preserveLocalZoomedPane(previous, restored)
		restored.InteractionMode = previous.InteractionMode
		restored.InteractionModeSeq = previous.InteractionModeSeq
		restored.OwnerConfirm = previous.OwnerConfirm
		restored.Overlay = previous.Overlay
		restored.EmptyPaneCTA = previous.EmptyPaneCTA
		restored.ExitedPaneCTA = previous.ExitedPaneCTA
	}
	return restored.EnsureDefaults()
}

func preserveLocalActiveWorkspace(previous state.ShellStore, restored state.ShellStore) state.ShellStore {
	if previous.Workspace.ID == "" {
		return restored
	}
	for _, workspace := range restored.Workspaces {
		if workspace.ID == previous.Workspace.ID {
			restored.Workspace = workspace
			return restored
		}
	}
	return restored
}

func mergeLocalWorkbenchFloatingState(previous state.ShellStore, restored state.ShellStore) state.ShellStore {
	local := localFloatingRuntimeState(previous)
	if len(local) == 0 {
		return restored
	}
	for workspaceIndex := range restored.Workspaces {
		for tabIndex := range restored.Workspaces[workspaceIndex].Tabs {
			workspaceID := restored.Workspaces[workspaceIndex].ID
			tabID := restored.Workspaces[workspaceIndex].Tabs[tabIndex].ID
			restored.Workspaces[workspaceIndex].Tabs[tabIndex] = mergeLocalTabFloatingState(workspaceID, tabID, restored.Workspaces[workspaceIndex].Tabs[tabIndex], local)
		}
		if restored.Workspaces[workspaceIndex].ID == restored.Workspace.ID {
			restored.Workspace = restored.Workspaces[workspaceIndex]
		}
	}
	restored.Workspace = mergeLocalTabFloatingStateForWorkspace(restored.Workspace, local)
	restored.Workspaces = upsertRestoredWorkspace(restored.Workspaces, restored.Workspace)
	return restored
}

type localFloatingKey struct {
	WorkspaceID string
	TabID       string
	FloatingID  string
}

type localFloatingState struct {
	Rect      state.FloatingRect
	Z         int
	Active    bool
	Collapsed bool
	FitMode   state.FloatingFitMode
	AutoFit   state.FloatingAutoFitState
}

func localFloatingRuntimeState(shell state.ShellStore) map[localFloatingKey]localFloatingState {
	out := map[localFloatingKey]localFloatingState{}
	for _, workspace := range shell.Workspaces {
		for _, tab := range workspace.Tabs {
			for _, floating := range tab.Floatings {
				if floating.ID == "" {
					continue
				}
				key := localFloatingKey{WorkspaceID: workspace.ID, TabID: tab.ID, FloatingID: floating.ID}
				out[key] = localFloatingState{
					Rect:      floating.Rect,
					Z:         floating.Z,
					Active:    floating.Active,
					Collapsed: floating.Collapsed,
					FitMode:   floating.FitMode,
					AutoFit:   floating.AutoFit,
				}
			}
		}
	}
	return out
}

func mergeLocalTabFloatingStateForWorkspace(workspace state.WorkspaceState, local map[localFloatingKey]localFloatingState) state.WorkspaceState {
	for tabIndex := range workspace.Tabs {
		workspace.Tabs[tabIndex] = mergeLocalTabFloatingState(workspace.ID, workspace.Tabs[tabIndex].ID, workspace.Tabs[tabIndex], local)
	}
	return workspace
}

func mergeLocalTabFloatingState(workspaceID string, tabID string, tab state.TabState, local map[localFloatingKey]localFloatingState) state.TabState {
	activeID := ""
	for index := range tab.Floatings {
		floating := &tab.Floatings[index]
		localState, ok := local[localFloatingKey{WorkspaceID: workspaceID, TabID: tabID, FloatingID: floating.ID}]
		if !ok {
			floating.Active = false
			continue
		}
		// 中文说明：workbench storage 只同步 floating slot/绑定；显示态按当前 TUI 保留。
		floating.Rect = localState.Rect
		floating.Z = localState.Z
		floating.Active = localState.Active
		floating.Collapsed = localState.Collapsed
		floating.FitMode = localState.FitMode
		floating.AutoFit = localState.AutoFit
		if floating.Active && !floating.Collapsed {
			activeID = floating.ID
		}
	}
	tab.ActiveFloatingID = activeID
	return tab
}

func preserveLocalActiveTabAndPane(previous state.ShellStore, restored state.ShellStore) state.ShellStore {
	activeTabID := restored.Workspace.ActiveTabID
	if tabExistsInWorkspace(restored.Workspace, previous.Workspace.ActiveTabID) {
		activeTabID = previous.Workspace.ActiveTabID
	}
	restored.Workspace.ActiveTabID = activeTabID
	activePaneID := previous.ActivePaneID
	if !paneExistsInWorkspaceTab(restored.Workspace, activeTabID, activePaneID) {
		activePaneID = activePaneIDForTab(restored.Workspace, activeTabID)
	}
	restored.ActivePaneID = activePaneID
	for tabIndex := range restored.Workspace.Tabs {
		if restored.Workspace.Tabs[tabIndex].ID == activeTabID {
			restored.Workspace.Tabs[tabIndex].ActivePaneID = activePaneID
			break
		}
	}
	restored.Workspaces = upsertRestoredWorkspace(restored.Workspaces, restored.Workspace)
	return restored
}

func preserveLocalZoomedPane(previous state.ShellStore, restored state.ShellStore) string {
	if previous.ZoomedPaneID != "" && paneExistsInShell(restored, previous.ZoomedPaneID) {
		return previous.ZoomedPaneID
	}
	return ""
}

func paneExistsInShell(shell state.ShellStore, paneID string) bool {
	if paneID == "" {
		return false
	}
	for _, workspace := range shell.Workspaces {
		for _, tab := range workspace.Tabs {
			for _, pane := range tab.Panes {
				if pane.ID == paneID {
					return true
				}
			}
		}
	}
	return false
}

func tabExistsInWorkspace(workspace state.WorkspaceState, tabID string) bool {
	if tabID == "" {
		return false
	}
	for _, tab := range workspace.Tabs {
		if tab.ID == tabID {
			return true
		}
	}
	return false
}

func paneExistsInWorkspaceTab(workspace state.WorkspaceState, tabID string, paneID string) bool {
	if paneID == "" {
		return false
	}
	for _, tab := range workspace.Tabs {
		if tab.ID != tabID {
			continue
		}
		for _, pane := range tab.Panes {
			if pane.ID == paneID {
				return true
			}
		}
	}
	return false
}

func activePaneIDForTab(workspace state.WorkspaceState, tabID string) string {
	for _, tab := range workspace.Tabs {
		if tab.ID != tabID {
			continue
		}
		if tab.ActivePaneID != "" {
			return tab.ActivePaneID
		}
		if len(tab.Panes) > 0 {
			return tab.Panes[0].ID
		}
		return ""
	}
	return ""
}

func upsertRestoredWorkspace(workspaces []state.WorkspaceState, workspace state.WorkspaceState) []state.WorkspaceState {
	if workspace.ID == "" {
		return workspaces
	}
	out := append([]state.WorkspaceState(nil), workspaces...)
	for index := range out {
		if out[index].ID == workspace.ID {
			out[index] = workspace
			return out
		}
	}
	return append(out, workspace)
}

func preserveWorkbenchRuntimeTerminalViews(previous state.TerminalViewStore, restored state.TerminalViewStore) state.TerminalViewStore {
	if restored.NextOperation < previous.NextOperation {
		restored.NextOperation = previous.NextOperation
	}
	if len(previous.Views) == 0 || len(restored.Views) == 0 {
		return restored
	}
	for viewID, binding := range restored.Views {
		previousBinding, ok := previous.Views[viewID]
		if !ok || !workbenchBindingAlreadyLive(previous, binding) {
			continue
		}
		// storage snapshot 是布局/绑定意图；当前 runtime 已经 live 的同一 view 不能因为 reload
		// 丢掉 channel 和 resize-control truth，否则会触发无意义 reattach/live-surface 风暴。
		binding.Channel = previousBinding.Channel
		binding.SurfaceID = previousBinding.SurfaceID
		binding.ResizeRole = previousBinding.ResizeRole
		binding.Attached = previousBinding.Attached
		binding.CanResize = previousBinding.CanResize
		binding.SizeLocked = previousBinding.SizeLocked
		binding.ControlReason = previousBinding.ControlReason
		binding.OwnerSurfaceID = previousBinding.OwnerSurfaceID
		binding.OwnerViewID = previousBinding.OwnerViewID
		binding.ResizeEpoch = previousBinding.ResizeEpoch
		binding.AttachPending = previousBinding.AttachPending
		binding.AttachCandidate = previousBinding.AttachCandidate
		binding.Session = previousBinding.AttachmentSession()
		binding.OperationID = previousBinding.OperationID
		binding.LastError = previousBinding.LastError
		restored.Views[viewID] = binding
	}
	return restored
}

func lifecycleActivePaneKind(root state.Root) string {
	shell := root.Shell.ReadonlyDefaults()
	if activeFloatingID := shell.ActiveFloatingID(); activeFloatingID != "" {
		for _, floating := range shell.ActiveFloatings() {
			if floating.ID == activeFloatingID {
				return string(floating.Pane.Kind)
			}
		}
	}
	pane, ok := shell.Pane(state.PaneCommandTarget{PaneID: shell.ActivePaneID})
	if !ok {
		return ""
	}
	return string(pane.Kind)
}

func lifecycleActiveTerminalID(root state.Root) string {
	if binding, ok := activeTerminalBinding(root); ok {
		return binding.TerminalID
	}
	return root.Surface.TerminalID
}

func activeTerminalBinding(root state.Root) (state.TerminalViewBinding, bool) {
	shell := root.Shell.ReadonlyDefaults()
	if activeFloatingID := shell.ActiveFloatingID(); activeFloatingID != "" {
		return root.TerminalViews.FloatingBinding(activeFloatingID)
	}
	return root.TerminalViews.PaneBinding(shell.ActivePaneID)
}

func workbenchRestoredTerminalAttachBindings(previous state.TerminalViewStore, bindings []state.TerminalViewBinding) []state.TerminalViewBinding {
	if len(bindings) == 0 {
		return nil
	}
	ordered := orderedRestoredTerminalBindings(bindings)
	out := make([]state.TerminalViewBinding, 0, len(ordered))
	for _, binding := range ordered {
		if binding.TerminalID == "" || binding.ViewID == "" {
			continue
		}
		if binding.Unresolved {
			continue
		}
		if workbenchBindingAlreadyLive(previous, binding) {
			continue
		}
		out = append(out, binding)
	}
	return out
}

func workbenchRestoredTerminalAttachEffects(previous state.TerminalViewStore, bindings []state.TerminalViewBinding) []Effect {
	return workbenchRestoredTerminalAttachEffectsForBindings(state.Root{}, workbenchRestoredTerminalAttachBindings(previous, bindings))
}

func workbenchRestoredTerminalAttachEffectsForBindings(root state.Root, bindings []state.TerminalViewBinding) []Effect {
	if len(bindings) == 0 {
		return nil
	}
	effects := make([]Effect, 0, len(bindings))
	for _, binding := range bindings {
		binding := binding
		cols, rows := binding.DesiredCols, binding.DesiredRows
		if cols <= 0 {
			cols = 80
		}
		if rows <= 0 {
			rows = 24
		}
		// 中文说明：storage 只保存结构/连接意图；恢复时不继承其它 TUI 的 owner。
		resizePolicy := state.TerminalResizeRoleFollower
		cfg := LiveConfig{
			EndpointID:   binding.EndpointID,
			TerminalID:   binding.TerminalID,
			Cols:         cols,
			Rows:         rows,
			Mode:         "collaborator",
			ResizePolicy: resizePolicy,
			SurfaceID:    runtimeSurfaceID(root),
			ViewID:       binding.ViewID,
		}
		cfgCopy := cfg
		effects = append(effects, FuncEffect{Run: func(context.Context) Msg {
			return LiveAttachMsg{Config: cfgCopy}
		}})
	}
	return effects
}

func workbenchRestoredTerminalLifecycleQueryEffects(previous state.TerminalViewStore, bindings []state.TerminalViewBinding) []Effect {
	targets := workbenchRestoredTerminalLifecycleQueryTargets(previous, bindings)
	if len(targets) == 0 {
		return nil
	}
	return []Effect{FuncEffect{Run: func(context.Context) Msg {
		return LiveLifecycleQueryMsg{Reason: "workbench.restore", Targets: targets}
	}}}
}

func workbenchRestoredTerminalLifecycleQueryTargets(previous state.TerminalViewStore, bindings []state.TerminalViewBinding) []LiveLifecycleQueryTarget {
	if len(bindings) == 0 {
		return nil
	}
	ordered := orderedRestoredTerminalBindings(bindings)
	targets := make([]LiveLifecycleQueryTarget, 0, len(ordered))
	seen := map[string]struct{}{}
	for _, binding := range ordered {
		if binding.TerminalID == "" {
			continue
		}
		if binding.Unresolved {
			continue
		}
		if !workbenchBindingAlreadyLive(previous, binding) {
			continue
		}
		refKey := binding.TerminalRef().Key()
		if _, ok := seen[refKey]; ok {
			continue
		}
		seen[refKey] = struct{}{}
		cols, rows := binding.DesiredCols, binding.DesiredRows
		if cols <= 0 {
			cols = 80
		}
		if rows <= 0 {
			rows = 24
		}
		targets = append(targets, LiveLifecycleQueryTarget{EndpointID: binding.EndpointID, TerminalID: binding.TerminalID, Cols: cols, Rows: rows})
	}
	return targets
}

func workbenchBindingAlreadyLive(previous state.TerminalViewStore, binding state.TerminalViewBinding) bool {
	if binding.Unresolved {
		return false
	}
	existing, ok := previous.Views[binding.ViewID]
	if !ok {
		return false
	}
	if existing.TerminalID == "" || !existing.TerminalRef().Equal(binding.TerminalRef()) {
		return false
	}
	if existing.AttachPending {
		return true
	}
	if !existing.Attached {
		return false
	}
	if existing.Channel == 0 {
		return false
	}
	return true
}

func orderedRestoredTerminalBindings(bindings []state.TerminalViewBinding) []state.TerminalViewBinding {
	ordered := append([]state.TerminalViewBinding(nil), bindings...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left := ordered[i]
		right := ordered[j]
		if left.TerminalID != right.TerminalID {
			return left.TerminalID < right.TerminalID
		}
		leftPriority := restoredResizeRolePriority(left.ResizeRole)
		rightPriority := restoredResizeRolePriority(right.ResizeRole)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		return left.ViewID < right.ViewID
	})
	return ordered
}

func restoredResizeRolePriority(role string) int {
	switch role {
	case state.TerminalResizeRoleOwner:
		return 0
	case state.TerminalResizeRoleFollower, "":
		return 1
	case state.TerminalResizeRoleObserver:
		return 2
	default:
		return 1
	}
}

func reduceWorkbenchStoragePersistRequest(root state.Root, msg WorkbenchStoragePersistRequestMsg, deps WorkbenchDeps) (state.Root, []Effect) {
	if deps.Storage == nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.storage", Body: "storage service missing"})
		return root.Advance(), nil
	}
	ref := workbenchStorageRef(root, deps)
	snapshot := state.SnapshotRootWorkbenchForStorage(root)
	expectedVersion := root.WorkbenchSync.SaveVersion()
	return root, []Effect{FuncEffect{Token: workbenchStorageSaveToken, Async: true, ForceSyncInTests: true, Run: func(ctx context.Context) Msg {
		result, err := deps.Storage.SaveWorkbench(ctx, port.WorkbenchStorageSaveRequest{
			Ref:             ref.WithVersion(expectedVersion),
			Snapshot:        snapshot,
			CheckVersion:    true,
			ExpectedVersion: expectedVersion,
		})
		logEffectError(deps.Logger, "workbench.storage.save", err, "key", ref.Key, "owner_id", ref.OwnerID, "expected_version", expectedVersion)
		if isContextLifecycleError(err) {
			return nil
		}
		return WorkbenchStoragePersistResultMsg{Result: result, Err: err, Reason: msg.Reason, ExpectedVersion: expectedVersion}
	}}}
}

func reduceWorkbenchStoragePersistResult(root state.Root, msg WorkbenchStoragePersistResultMsg) (state.Root, []Effect) {
	if msg.ExpectedVersion != root.WorkbenchSync.SaveVersion() {
		return root, nil
	}
	if msg.Err != nil {
		if isContextLifecycleError(msg.Err) {
			return root, nil
		}
		if errors.Is(msg.Err, port.ErrWorkbenchStorageConflict) {
			if root.WorkbenchSync.Conflict && root.WorkbenchSync.ConflictVersion == msg.ExpectedVersion {
				return root, nil
			}
			root.WorkbenchSync = root.WorkbenchSync.MarkConflict(msg.ExpectedVersion)
			root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.storage", Body: "conflict: reloading"})
			return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg {
				return WorkbenchStorageLoadRequestMsg{}
			}}}
		}
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "workbench.storage", Body: errorString(msg.Err)})
		return root.Advance(), nil
	}
	body := msg.Result.Ref.Key
	if body == "" {
		body = state.WorkbenchStorageKeyRoot
	}
	root.WorkbenchSync = root.WorkbenchSync.MarkSaved(msg.Result.Ref, msg.Result.Version)
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "workbench.storage", Body: body})
	return root.Advance(), nil
}

func workbenchStorageRef(root state.Root, deps WorkbenchDeps) state.WorkbenchStorageRef {
	if deps.Ref.AppID != "" || deps.Ref.Key != "" {
		return deps.Ref
	}
	return state.DefaultWorkbenchStorageRef(root.Shell.EnsureDefaults().Workspace.ID)
}

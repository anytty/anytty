package app

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/shared/perftrace"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

const DefaultRuntimeSurfaceID = "tui"

type LiveConfig struct {
	EndpointID   state.EndpointID
	TerminalID   string
	Cols         int
	Rows         int
	Mode         string
	ResizePolicy string
	SurfaceID    string
	ViewID       string
}

type LiveDeps struct {
	Terminal            port.TerminalService
	Path                port.PathService
	EndpointEvents      port.EndpointEventSource
	EndpointConnections port.EndpointConnectionService
	Logger              *slog.Logger
}

const liveScreenNextTokenPrefix = "terminal.live.screen.next:"

const terminalEventWatchTokenPrefix = "terminal.events.watch:"

func runtimeSurfaceID(root state.Root) string {
	if root.RuntimeSurfaceID != "" {
		return root.RuntimeSurfaceID
	}
	return DefaultRuntimeSurfaceID
}

// NewLiveRuntime 组合 live app 主路径：TerminalHost 输入 -> reducer/effect ->
// terminal service -> render VM -> FrameSink。
func NewLiveRuntime(initial state.Root, host TerminalHost, runner EffectRunner, deps LiveDeps) *AppRuntime {
	applyConfiguredPaneChromeGlyphs(initial.Config)
	initial.Shell = applyConfiguredShellChrome(initial.Shell, initial.Config)
	initial.Shell = initial.Shell.EnsureDefaults()
	builder := render.NewRenderVMBuilder()
	renderer := render.NewRenderer(render.DefaultTheme())
	deps = liveDepsWithEndpointEvents(deps)
	runtime := NewAppRuntime(initial, ComposeReducers(NewBackNavigationReducer(CopyModeDeps{}), NewShellReducer(), NewUIInputReducer(), NewEndpointConnectionsReducer(deps), NewEndpointStatusReducer(deps), NewEndpointDefaultsReducer(deps), NewPromptPathCompletionReducer(deps), NewTerminalPoolReducer(deps), NewTerminalInputRouterReducer(deps), NewLiveReducer(deps), NewTerminalLayoutResizeReducer()), hostRenderFunc(host, builder, renderer), host, runner)
	runtime.SetLogger(deps.Logger)
	if deps.EndpointEvents != nil && shouldAutoStartEndpointWatch(runner) {
		runtime.enqueue(EndpointWatchRequestMsg{})
	}
	if _, ok := deps.EndpointConnections.(port.EndpointConnectionWatchService); ok && shouldAutoStartEndpointWatch(runner) {
		runtime.enqueue(ConnectionsWatchRequestMsg{})
	}
	if _, ok := deps.Terminal.(port.TerminalEventSource); ok && shouldAutoStartEndpointWatch(runner) {
		runtime.enqueue(TerminalEventWatchRequestMsg{EndpointID: initial.Session.EndpointID})
	}
	return runtime
}

// NewInteractiveRuntime 组合 live 与 copy mode 主路径。copy mode 会消费
// page up、wheel、selection/copy 等交互；普通 terminal input 仍交给 live path。
func NewInteractiveRuntime(
	initial state.Root,
	host TerminalHost,
	runner EffectRunner,
	live LiveDeps,
	copyMode CopyModeDeps,
) *AppRuntime {
	return NewInteractiveRuntimeWithStorage(initial, host, runner, live, copyMode, WorkbenchDeps{}, ClipboardDeps{})
}

func NewInteractiveRuntimeWithWorkbench(
	initial state.Root,
	host TerminalHost,
	runner EffectRunner,
	live LiveDeps,
	copyMode CopyModeDeps,
	workbench WorkbenchDeps,
) *AppRuntime {
	return NewInteractiveRuntimeWithStorage(initial, host, runner, live, copyMode, workbench, ClipboardDeps{})
}

func NewInteractiveRuntimeWithStorage(
	initial state.Root,
	host TerminalHost,
	runner EffectRunner,
	live LiveDeps,
	copyMode CopyModeDeps,
	workbench WorkbenchDeps,
	clipboard ClipboardDeps,
) *AppRuntime {
	applyConfiguredPaneChromeGlyphs(initial.Config)
	initial.Shell = applyConfiguredShellChrome(initial.Shell, initial.Config)
	initial.Shell = initial.Shell.EnsureDefaults()
	builder := render.NewRenderVMBuilder()
	renderer := render.NewRenderer(render.DefaultTheme())
	live = liveDepsWithEndpointEvents(live)
	clipboardActions := ClipboardActionDeps{Core: copyMode.Core, Clipboard: copyMode.Clipboard, Terminal: live.Terminal}
	runtime := NewAppRuntime(initial, ComposeReducers(NewBackNavigationReducer(copyMode), NewClipboardActionReducer(clipboardActions), NewShellReducer(), NewUIInputReducer(), NewEndpointConnectionsReducer(live), NewEndpointStatusReducer(live), NewEndpointDefaultsReducer(live), NewPromptPathCompletionReducer(live), NewTerminalPoolReducer(live), NewWorkbenchStorageReducer(workbench), NewClipboardStorageReducer(clipboard), NewCopyModeReducer(copyMode), NewCopyModeResizeRebindReducer(copyMode), NewTerminalInputRouterReducer(live), NewLiveReducer(live), NewTerminalLayoutResizeReducer()), hostRenderFunc(host, builder, renderer), host, runner)
	runtime.SetLogger(live.Logger)
	if live.EndpointEvents != nil && shouldAutoStartEndpointWatch(runner) {
		runtime.enqueue(EndpointWatchRequestMsg{})
	}
	if _, ok := live.EndpointConnections.(port.EndpointConnectionWatchService); ok && shouldAutoStartEndpointWatch(runner) {
		runtime.enqueue(ConnectionsWatchRequestMsg{})
	}
	if _, ok := live.Terminal.(port.TerminalEventSource); ok && shouldAutoStartEndpointWatch(runner) {
		runtime.enqueue(TerminalEventWatchRequestMsg{EndpointID: initial.Session.EndpointID})
	}
	if workbench.Storage != nil {
		// 启动时先恢复 core-v2 opaque storage 中的 workbench truth，再订阅后续变化。
		if !workbench.SkipInitialLoad {
			runtime.enqueue(WorkbenchStorageLoadRequestMsg{})
		}
		runtime.enqueue(WorkbenchStorageWatchRequestMsg{})
	}
	if clipboard.Storage != nil {
		// Clipboard history is host-owned materialized text, independent of the active endpoint session.
		runtime.enqueue(ClipboardStorageLoadRequestMsg{Reason: "startup"})
		runtime.enqueue(ClipboardStorageWatchRequestMsg{})
	}
	return runtime
}

func liveDepsWithEndpointEvents(deps LiveDeps) LiveDeps {
	if deps.EndpointEvents == nil {
		if source, ok := deps.Terminal.(port.EndpointEventSource); ok {
			deps.EndpointEvents = source
		}
	}
	return deps
}

func shouldAutoStartEndpointWatch(runner EffectRunner) bool {
	if runner == nil {
		return false
	}
	_, syncRunner := runner.(*SyncEffectRunner)
	return !syncRunner
}

func hostRenderFunc(host TerminalHost, builder render.RenderVMBuilder, renderer render.Renderer) RenderFunc {
	ansiOnly := false
	if host != nil {
		if sink := host.FrameSink(); sink != nil {
			if preference, ok := sink.(render.FrameSinkPreference); ok {
				ansiOnly = !preference.NeedsCompleteFrame()
			}
		}
	}
	return func(root state.Root) render.Frame {
		finishVM := perftrace.Measure("tui.render_vm_build")
		vm := builder.Build(root)
		finishVM(0)
		if ansiOnly {
			// 中文说明：真实 TTY 只消费 ANSI 行；测试 sink 默认保留 plain/styled frame 方便断言。
			finishANSI := perftrace.Measure("tui.render_ansi")
			frame := renderer.RenderANSI(vm)
			finishANSI(frameApproxBytes(frame))
			return frame
		}
		finishStyled := perftrace.Measure("tui.render_styled")
		frame := renderer.Render(vm)
		finishStyled(frameApproxBytes(frame))
		return frame
	}
}

type LiveAttachMsg struct {
	Config LiveConfig
}

func (LiveAttachMsg) isMsg() {}

type LiveAttachResultMsg struct {
	EndpointID            state.EndpointID
	TerminalID            string
	ViewID                string
	RequestedResizePolicy string
	OperationID           string
	Result                port.TerminalAttachResult
	Err                   error
}

func (LiveAttachResultMsg) isMsg() {}

type LiveDetachRequestMsg struct {
	Request port.TerminalDetachRequest
}

func (LiveDetachRequestMsg) isMsg() {}

type LiveDetachResultMsg struct {
	Request port.TerminalDetachRequest
	Err     error
}

func (LiveDetachResultMsg) isMsg() {}

type LiveSurfaceMsg struct {
	Snapshot       state.LiveSurfaceSnapshot
	Err            error
	LifecycleKnown bool
	RequestedCols  int
	RequestedRows  int
}

func (LiveSurfaceMsg) isMsg() {}

// LiveScreenNextResultMsg 是某个 TerminalRef 唯一在途 latest-screen 请求的结果。
// Generation 把 late response 与当前 view demand 隔离；结果先进入 canonical cache，
// 下一次请求要等该 revision 被 renderer 选中提交后才会启动。
type LiveScreenNextResultMsg struct {
	EndpointID state.EndpointID
	TerminalID string
	Generation uint64
	Snapshot   state.LiveSurfaceSnapshot
	Err        error
}

func (LiveScreenNextResultMsg) isMsg() {}

// LiveScreenFrameSelectedMsg 表示一帧已进入 FrameSink submission，不代表物理写出完成。
// Full=true 时 Targets 同时是当前完整画面的 live demand 集合。
type LiveScreenFrameSelectedMsg struct {
	Full    bool
	Targets []render.LiveRenderTarget
}

func (LiveScreenFrameSelectedMsg) isMsg() {}

func (LiveScreenFrameSelectedMsg) SkipRender() bool { return true }

type LiveLifecycleQueryTarget struct {
	EndpointID state.EndpointID
	TerminalID string
	Cols       int
	Rows       int
}

type LiveLifecycleQueryMsg struct {
	Reason  string
	Targets []LiveLifecycleQueryTarget
}

func (LiveLifecycleQueryMsg) isMsg() {}

type LiveEventMsg struct {
	Event port.TerminalLiveEvent
}

type TerminalEventWatchRequestMsg struct {
	EndpointID state.EndpointID
}

func (TerminalEventWatchRequestMsg) isMsg() {}

func (TerminalEventWatchRequestMsg) SkipRender() bool { return true }

func (LiveEventMsg) isMsg() {}

func (msg LiveEventMsg) SkipRender() bool {
	return msg.isOrdinaryRefresh()
}

func (msg LiveEventMsg) isOrdinaryRefresh() bool {
	return ordinaryLiveRefreshEvent(msg.Event)
}

type LiveExitMsg struct {
	EndpointID state.EndpointID
	TerminalID string
	ExitCode   int
	Reason     string
}

func (LiveExitMsg) isMsg() {}

type LiveResizeMsg struct {
	EndpointID         state.EndpointID
	TerminalID         string
	Cols               int
	Rows               int
	Seq                uint64
	ViewID             string
	TakeOwnership      bool
	ExpectedOwnerEpoch uint64
}

func (LiveResizeMsg) isMsg() {}

type LiveResizeResultMsg struct {
	EndpointID    state.EndpointID
	TerminalID    string
	Result        port.TerminalResizeResult
	Cols          int
	Rows          int
	Seq           uint64
	ViewID        string
	Session       *apipb.EndpointSessionStamp
	OperationID   string
	TakeOwnership bool
	Err           error
}

func (LiveResizeResultMsg) isMsg() {}

type LiveInputResultMsg struct {
	EndpointID  state.EndpointID
	TerminalID  string
	ViewID      string
	Channel     uint16
	Session     *apipb.EndpointSessionStamp
	Event       input.InputEvent
	Bytes       []byte
	OperationID string
	Err         error
}

func (LiveInputResultMsg) isMsg() {}

type LiveInputAttachResultMsg struct {
	Target      liveInputTargetInfo
	Event       input.InputEvent
	Bytes       []byte
	OperationID string
	Result      port.TerminalAttachResult
	Err         error
}

func (LiveInputAttachResultMsg) isMsg() {}

func NewLiveReducer(deps LiveDeps) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		switch msg := msg.(type) {
		case LiveAttachMsg:
			return reduceLiveAttach(root, msg, deps)
		case LiveAttachResultMsg:
			return reduceLiveAttachResult(root, msg, deps)
		case LiveDetachRequestMsg:
			return reduceLiveDetachRequest(root, msg, deps)
		case LiveDetachResultMsg:
			if msg.Err != nil {
				root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.detach", Body: msg.Err.Error()})
				return root.Advance(), nil
			}
			return root, nil
		case LiveLifecycleQueryMsg:
			return reduceLiveLifecycleQuery(root, msg, deps)
		case LiveSurfaceMsg:
			finishReduce := perftrace.Measure("tui.reducer.live_surface")
			defer finishReduce(liveSnapshotApproxBytes(msg.Snapshot))
			if msg.Snapshot.EndpointID == "" && msg.Snapshot.TerminalID == root.Surface.TerminalID {
				msg.Snapshot.EndpointID = root.Surface.EndpointID
			}
			msg.Snapshot.EndpointID = state.NormalizeEndpointID(msg.Snapshot.EndpointID)
			snapshotRef := msg.Snapshot.TerminalRef()
			if msg.Err != nil {
				root.Surface = root.Surface.FinishRefreshRef(snapshotRef)
				if next, ok := markTerminalExitedFromErrorRef(root, snapshotRef, msg.Err); ok {
					logLifecycleTrace(deps.Logger, "live.surface.error.exited",
						"endpoint_id", string(snapshotRef.EndpointID),
						"terminal_id", msg.Snapshot.TerminalID,
						"error", msg.Err.Error(),
						"surface_state", string(next.Surface.SurfaceForTerminalRef(snapshotRef).State),
						"session_state", string(next.Session.State),
					)
					return next.Advance(), nil
				}
				root = applyLiveErrorRef(root, snapshotRef, msg.Err.Error())
				root, effects := maybeScheduleDirtyLiveSurfaceRefreshRef(root, snapshotRef, msg.RequestedCols, msg.RequestedRows, deps)
				return root.Advance(), effects
			}
			if ordinaryLiveSurfaceWasInvalidatedWhileInFlight(root, msg) {
				// 中文说明：这张 ordinary live surface 返回时已经被后续 invalidation
				// 标成 dirty，说明它只是中间屏。TUI live truth 是 core latest
				// native screen，不应先渲染过期 snapshot 再补拉下一帧。
				root.Surface = root.Surface.FinishRefreshRef(snapshotRef)
				return maybeScheduleDirtyLiveSurfaceRefreshRef(root, snapshotRef, msg.RequestedCols, msg.RequestedRows, deps)
			}
			root.Surface = root.Surface.ApplySnapshotWithLifecycle(msg.Snapshot, msg.LifecycleKnown)
			if msg.LifecycleKnown && msg.Snapshot.State == state.TerminalLiveAttached && root.Session.TerminalRef().Equal(snapshotRef) {
				root.Session = root.Session.MarkAttachedRef(snapshotRef)
			}
			logLifecycleTrace(deps.Logger, "live.surface",
				"endpoint_id", string(snapshotRef.EndpointID),
				"terminal_id", msg.Snapshot.TerminalID,
				"snapshot_state", string(msg.Snapshot.State),
				"lifecycle_known", msg.LifecycleKnown,
				"snapshot_exit_code", msg.Snapshot.ExitCode,
				"snapshot_exited_at", lifecycleTimeSummary(msg.Snapshot.ExitedAt),
				"snapshot_command", strings.Join(msg.Snapshot.Command, " "),
				"surface_state", string(root.Surface.State),
				"surface_terminal_state", string(root.Surface.SurfaceForTerminalRef(snapshotRef).State),
				"session_state", string(root.Session.State),
				"active_terminal", lifecycleActiveTerminalID(root),
				"bindings", lifecycleTerminalViewBindingsSummary(root.TerminalViews.BindingsForTerminalRef(snapshotRef)),
			)
			next, effects := maybeRefreshFloatingAutoFit(root, msg.Snapshot.TerminalID)
			next, dirtyEffects := maybeScheduleDirtyLiveSurfaceRefreshRef(next, snapshotRef, msg.RequestedCols, msg.RequestedRows, deps)
			return next, append(effects, dirtyEffects...)
		case LiveScreenFrameSelectedMsg:
			return reduceLiveScreenFrameSelected(root, msg, deps)
		case LiveScreenNextResultMsg:
			return reduceLiveScreenNextResult(root, msg, deps)
		case LiveEventMsg:
			finishReduce := perftrace.Measure("tui.reducer.live_event")
			defer finishReduce(liveEventApproxBytes(msg.Event))
			return reduceLiveEvent(root, msg, deps)
		case TerminalEventWatchRequestMsg:
			return reduceTerminalEventWatchRequest(root, msg, deps)
		case LiveExitMsg:
			ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
			root.Session = root.Session.MarkExitedWithMetadataRef(ref, msg.ExitCode, msg.Reason, time.Time{}, nil)
			root.Surface = root.Surface.MarkExitedWithMetadataRef(ref, msg.ExitCode, msg.Reason, time.Time{}, nil)
			return root.Advance(), nil
		case LiveInputResultMsg:
			if msg.Err != nil {
				if isContextLifecycleError(msg.Err) {
					return root, nil
				}
				if !liveInputResultOwnsCurrentBinding(root, msg) {
					return root, nil
				}
				msgRef := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
				if next, ok := markTerminalExitedFromErrorRef(root, msgRef, msg.Err); ok {
					return next.Advance(), nil
				}
				root = setLiveInputViewError(root, msgRef, msg.ViewID, msg.Err.Error())
				return root.Advance(), nil
			}
			return root, nil
		case LiveInputAttachResultMsg:
			return reduceLiveInputAttachResult(root, msg, deps)
		case LiveResizeMsg:
			return reduceLiveResize(root, msg, deps)
		case LiveResizeResultMsg:
			viewScoped := msg.ViewID != ""
			resizeRef := state.TerminalRef{}
			if msg.ViewID != "" {
				binding, ok := root.TerminalViews.Views[msg.ViewID]
				if !ok {
					// view 已经被关闭或解绑后，迟到的 resize 结果不能再回写共享 session/surface；
					// 否则 close pane 后旧 view 的结果会把当前 owner 的尺寸状态顶回去。
					return recoverLatestResizeAfterStaleResult(root, msg)
				}
				if !protoEndpointSessionEqual(binding.Session, msg.Session) {
					return root, nil
				}
				if binding.IsStaleResizeResult(msg.Seq) {
					return recoverLatestResizeAfterStaleResult(root, msg)
				}
				resizeRef = binding.TerminalRef()
			}
			if !viewScoped && root.Session.IsStaleResizeResult(msg.Seq) {
				return recoverLatestResizeAfterStaleResult(root, msg)
			}
			if msg.Err != nil {
				return reduceLiveResizeError(root, msg)
			}
			if !msg.TakeOwnership && shouldRecoverOwnerDesiredAfterResizeResult(root, msg) {
				return recoverLatestResizeAfterStaleResult(root, msg)
			}
			if msg.ViewID != "" {
				binding := root.TerminalViews.Views[msg.ViewID]
				if !msg.TakeOwnership && liveResizeResultConflictsWithLocalOwner(root, binding, msg.Result) {
					return recoverLatestResizeAfterStaleResult(root, msg)
				}
			}
			cols, rows := msg.Cols, msg.Rows
			if msg.Result.Cols > 0 {
				cols = msg.Result.Cols
			}
			if msg.Result.Rows > 0 {
				rows = msg.Result.Rows
			}
			if viewScoped {
				if hasResizeControlResult(msg.Result) {
					root.TerminalViews, _ = root.TerminalViews.ApplyResizeControl(msg.ViewID, resizeControlProjectionFromResult(msg.Result))
				}
				root = applyTerminalAttachmentProjectionFromResize(root, msg.Result)
				root.TerminalViews, _ = root.TerminalViews.ApplyResizeResult(msg.ViewID, msg.Seq, cols, rows, "")
				if msg.TakeOwnership {
					if binding, ok := root.TerminalViews.Views[msg.ViewID]; ok && !binding.HasProjectedResizeOwner() {
						root.Shell = root.Shell.EnsureDefaults().AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.owner", Body: "owner changed; try again"})
					}
				}
				if msg.Result.Resized || !hasResizeControlResult(msg.Result) {
					root.Surface = root.Surface.ResizeRef(resizeRef, cols, rows)
					if root.Session.TerminalRef().Equal(resizeRef) {
						root.Session = root.Session.Resize(cols, rows)
					}
				}
			} else {
				nextSession, applied := root.Session.ApplyResizeResult(msg.Seq, cols, rows)
				if !applied {
					return root, nil
				}
				root.Session = nextSession
				root.Surface = root.Surface.Resize(cols, rows)
			}
			if resizeViewID := msg.ViewID; resizeViewID != "" {
				root = rootWithCopyHistorySessionForView(root, resizeViewID)
				viewRows := rows
				if binding, ok := root.TerminalViews.Views[resizeViewID]; ok {
					if rect, rectOK := terminalViewContentRect(root, render.Rect{}, binding); rectOK {
						viewRows = copyModeVisibleRows(rows, rect.H)
					}
				}
				if root.CopyMode.Active && root.CopyMode.BoundCols != cols {
					root.CopyMode = root.CopyMode.Resize(cols, viewRows)
					root = saveCopyHistorySessionForView(root, resizeViewID)
				} else if root.CopyMode.Active {
					root.CopyMode = root.CopyMode.SetViewRows(viewRows)
					root.CopyMode = root.CopyMode.Scroll(0, len(root.History.Rows))
					root = saveCopyHistorySessionForView(root, resizeViewID)
				}
			}
			return maybeRefreshFloatingAutoFit(root, liveResizeTerminalID(root, msg))
		default:
			return root, nil
		}
	}
}

func reduceLiveDetachRequest(root state.Root, msg LiveDetachRequestMsg, deps LiveDeps) (state.Root, []Effect) {
	if deps.Terminal == nil {
		return root, nil
	}
	req := msg.Request
	if req.TerminalID == "" || (req.Channel == 0 && req.SurfaceID == "" && req.ViewID == "") {
		return root, nil
	}
	return root, []Effect{FuncEffect{
		Async:            true,
		ForceSyncInTests: true,
		Run: func(ctx context.Context) Msg {
			err := deps.Terminal.Detach(ctx, req)
			return LiveDetachResultMsg{Request: req, Err: err}
		},
	}}
}

func reduceLiveAttach(root state.Root, msg LiveAttachMsg, deps LiveDeps) (state.Root, []Effect) {
	if deps.Terminal == nil {
		return setLiveError(root, "terminal service missing"), nil
	}
	cfg := msg.Config
	cfg.EndpointID = state.NormalizeEndpointID(cfg.EndpointID)
	cfg.Cols, cfg.Rows = liveAttachContentSize(root, cfg)
	if cfg.SurfaceID == "" {
		cfg.SurfaceID = runtimeSurfaceID(root)
	}
	if cfg.ViewID == "" {
		cfg.ViewID = liveAttachDefaultViewID(root)
	}
	if cfg.ResizePolicy == "" {
		cfg.ResizePolicy = state.TerminalResizeRoleFollower
	}
	var candidate state.TerminalAttachCandidate
	root, candidate = markLiveAttachPending(root, cfg)
	if candidate.OperationID == "" {
		return root, nil
	}
	logLifecycleTrace(deps.Logger, "live.attach.request",
		"terminal_id", cfg.TerminalID,
		"view_id", cfg.ViewID,
		"surface_id", cfg.SurfaceID,
		"cols", cfg.Cols,
		"rows", cfg.Rows,
		"mode", cfg.Mode,
		"resize_policy", cfg.ResizePolicy,
		"existing_bindings", lifecycleTerminalViewBindingsSummary(root.TerminalViews.BindingsForTerminalRef(state.NewTerminalRef(cfg.EndpointID, cfg.TerminalID))),
		"surface_state", string(root.Surface.SurfaceForTerminalRef(state.NewTerminalRef(cfg.EndpointID, cfg.TerminalID)).State),
		"session_terminal", root.Session.TerminalID,
		"session_state", string(root.Session.State),
	)
	return root, []Effect{FuncEffect{
		Async:            true,
		ForceSyncInTests: true,
		Run: func(ctx context.Context) Msg {
			result, err := deps.Terminal.Attach(ctx, port.TerminalAttachRequest{
				EndpointID:   cfg.EndpointID,
				TerminalID:   cfg.TerminalID,
				Cols:         cfg.Cols,
				Rows:         cfg.Rows,
				Mode:         cfg.Mode,
				ResizePolicy: cfg.ResizePolicy,
				SurfaceID:    cfg.SurfaceID,
				ViewID:       cfg.ViewID,
				OperationID:  candidate.OperationID,
			})
			return LiveAttachResultMsg{EndpointID: cfg.EndpointID, TerminalID: cfg.TerminalID, ViewID: cfg.ViewID, RequestedResizePolicy: cfg.ResizePolicy, OperationID: candidate.OperationID, Result: result, Err: err}
		},
	}}
}

func markLiveAttachPending(root state.Root, cfg LiveConfig) (state.Root, state.TerminalAttachCandidate) {
	if cfg.TerminalID == "" || cfg.ViewID == "" {
		return root, state.TerminalAttachCandidate{}
	}
	target, ok := liveAttachTargetForViewID(root, cfg.ViewID)
	if !ok {
		if len(root.TerminalViews.Views) != 0 {
			return root, state.TerminalAttachCandidate{}
		}
		target.PaneID = root.Shell.EnsureDefaults().ActivePaneID
	}
	role := cfg.ResizePolicy
	if role == "" {
		role = state.TerminalResizeRoleFollower
	}
	binding := state.TerminalViewBinding{
		ViewID:      cfg.ViewID,
		SurfaceID:   cfg.SurfaceID,
		EndpointID:  state.NormalizeEndpointID(cfg.EndpointID),
		TerminalID:  cfg.TerminalID,
		ResizeRole:  role,
		DesiredCols: cfg.Cols,
		DesiredRows: cfg.Rows,
		PaneID:      target.PaneID,
		FloatingID:  target.FloatingID,
	}
	// 中文说明：attach 请求已经发出但 channel 未返回时，先占住当前 view，
	// 避免 storage restore 把同一 view/terminal 再次 attach 成第二个附件。
	var candidate state.TerminalAttachCandidate
	root.TerminalViews, candidate = root.TerminalViews.BeginAttach(binding)
	return root, candidate
}

func reduceLiveAttachResult(root state.Root, msg LiveAttachResultMsg, deps LiveDeps) (state.Root, []Effect) {
	if msg.Err != nil {
		ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
		binding, hadBinding := root.TerminalViews.Views[msg.ViewID]
		if !hadBinding || binding.AttachCandidate == nil || binding.AttachCandidate.OperationID != msg.OperationID {
			return root, nil
		}
		if next, ok := markTerminalExitedFromErrorRef(root, ref, msg.Err); ok {
			next.TerminalViews, _ = next.TerminalViews.FailAttach(msg.ViewID, msg.OperationID, msg.Err.Error())
			logLifecycleTrace(deps.Logger, "live.attach.result.exited",
				"endpoint_id", string(ref.EndpointID),
				"terminal_id", msg.TerminalID,
				"error", msg.Err.Error(),
				"surface_state", string(next.Surface.SurfaceForTerminalRef(ref).State),
				"session_state", string(next.Session.State),
			)
			return next.Advance(), nil
		}
		root.TerminalViews, _ = root.TerminalViews.FailAttach(msg.ViewID, msg.OperationID, msg.Err.Error())
		if hadBinding && binding.Attached {
			return root, nil
		}
		logLifecycleTrace(deps.Logger, "live.attach.result",
			"endpoint_id", string(ref.EndpointID),
			"terminal_id", msg.TerminalID,
			"error", msg.Err.Error(),
			"bindings", lifecycleTerminalViewBindingsSummary(root.TerminalViews.BindingsForTerminalRef(ref)),
		)
		root = applyLiveAttachErrorRef(root, ref, msg.ViewID, msg.Err.Error())
		return root.Advance(), nil
	}
	result := msg.Result
	if result.OperationID == "" {
		result.OperationID = msg.OperationID
	}
	current, currentOK := root.TerminalViews.Views[msg.ViewID]
	if !currentOK || current.AttachCandidate == nil || current.AttachCandidate.OperationID != msg.OperationID || result.OperationID != msg.OperationID {
		return root, cleanupAttachResultEffects(result)
	}
	previous := state.TerminalViewBinding{}
	if current.Attached {
		previous = current
	}
	if result.EndpointID == "" {
		result.EndpointID = state.NormalizeEndpointID(msg.EndpointID)
	}
	if result.TerminalID == "" {
		result.TerminalID = msg.TerminalID
	}
	viewID := result.ViewID
	activePaneID := root.Shell.ReadonlyDefaults().ActivePaneID
	if viewID == "" {
		viewID = liveAttachDefaultViewID(root)
		result.ViewID = viewID
	} else if !liveAttachViewStillPresent(root, viewID) {
		// 外部 reload/restore 替换 pane/view 结构后，旧 view 的迟到 attach result
		// 不能回退绑定到当前 pane/floating；但首次 attach 时 view binding 可能还没建，
		// 只要 shell 结构里仍存在这个目标 view，就必须允许它继续落到当前 truth。
		logLifecycleTrace(deps.Logger, "live.attach.result.stale_view",
			"terminal_id", result.TerminalID,
			"view_id", viewID,
			"channel", result.Channel,
			"bindings", lifecycleTerminalViewsSummary(root.TerminalViews),
		)
		return root, cleanupAttachResultEffects(result)
	}
	target, hasTarget := liveAttachTargetForViewID(root, viewID)
	if hasTarget && target.PaneID != "" {
		activePaneID = target.PaneID
	}
	root = applyLiveAttachRuntimeProjection(root, result, viewID)
	root = applyTerminalAttachmentProjectionFromAttach(root, result)
	if existing, ok := root.TerminalViews.Views[viewID]; ok {
		if existing.PaneID != "" {
			activePaneID = existing.PaneID
		}
		if existing.FloatingID != "" {
			var copyHistoryEffects []Effect
			root, copyHistoryEffects = invalidateCopyModeForTerminalRebindRef(root, existing.PaneID, viewID, state.NewTerminalRef(result.EndpointID, result.TerminalID))
			binding := state.NewEndpointFloatingTerminalView(result.EndpointID, existing.FloatingID, existing.PaneID, result.TerminalID, result.Channel, result.Cols, result.Rows, result.ResizePolicy, result.SurfaceID, viewID, result.CanResize)
			binding.Session = result.Session
			binding.OperationID = result.OperationID
			binding.Layout = existing.Layout
			root.TerminalViews = root.TerminalViews.BindFloating(binding)
			root.TerminalViews = projectTerminalAttachResizeControl(root.TerminalViews, result)
			logLiveAttachApplied(deps, root, result, "floating-existing")
			effects := append(copyHistoryEffects, liveAttachAppliedEffects(result.EndpointID, result.TerminalID, deps)...)
			effects = append(effects, liveEffectsForRef(result.EndpointID, result.TerminalID, result.Cols, result.Rows, deps)...)
			effects = appendPreviousAttachmentCleanup(effects, previous, result)
			return root.Advance(), effects
		}
	}
	if hasTarget && target.FloatingID != "" {
		var copyHistoryEffects []Effect
		root, copyHistoryEffects = invalidateCopyModeForTerminalRebindRef(root, target.PaneID, viewID, state.NewTerminalRef(result.EndpointID, result.TerminalID))
		binding := state.NewEndpointFloatingTerminalView(result.EndpointID, target.FloatingID, target.PaneID, result.TerminalID, result.Channel, result.Cols, result.Rows, result.ResizePolicy, result.SurfaceID, viewID, result.CanResize)
		binding.Session = result.Session
		binding.OperationID = result.OperationID
		root.TerminalViews = root.TerminalViews.BindFloating(binding)
		root.TerminalViews = projectTerminalAttachResizeControl(root.TerminalViews, result)
		root.Shell = root.Shell.BindFloatingTerminal(target.FloatingID, result.TerminalID)
		logLiveAttachApplied(deps, root, result, "floating-target")
		effects := append(copyHistoryEffects, liveAttachAppliedEffects(result.EndpointID, result.TerminalID, deps)...)
		effects = append(effects, liveEffectsForRef(result.EndpointID, result.TerminalID, result.Cols, result.Rows, deps)...)
		effects = appendPreviousAttachmentCleanup(effects, previous, result)
		return root.Advance(), effects
	}
	root.Shell = root.Shell.EnsureActiveTabForAttach()
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: activePaneID}, result.TerminalID)
	var copyHistoryEffects []Effect
	root, copyHistoryEffects = invalidateCopyModeForTerminalRebindRef(root, activePaneID, viewID, state.NewTerminalRef(result.EndpointID, result.TerminalID))
	binding := state.NewEndpointPaneTerminalView(result.EndpointID, activePaneID, result.TerminalID, result.Channel, result.Cols, result.Rows, result.ResizePolicy, result.SurfaceID, viewID, result.CanResize)
	binding.Session = result.Session
	binding.OperationID = result.OperationID
	if existing, ok := root.TerminalViews.Views[viewID]; ok {
		binding.Layout = existing.Layout
	}
	root.TerminalViews = root.TerminalViews.BindPane(binding)
	root.TerminalViews = projectTerminalAttachResizeControl(root.TerminalViews, result)
	logLiveAttachApplied(deps, root, result, "pane")
	effects := append(copyHistoryEffects, liveAttachAppliedEffects(result.EndpointID, result.TerminalID, deps)...)
	effects = append(effects, liveEffectsForRef(result.EndpointID, result.TerminalID, result.Cols, result.Rows, deps)...)
	effects = appendPreviousAttachmentCleanup(effects, previous, result)
	return root.Advance(), effects
}

func cleanupAttachResultEffects(result port.TerminalAttachResult) []Effect {
	if result.Channel == 0 || result.Session == nil || result.OperationID == "" {
		return nil
	}
	return []Effect{terminalDetachEffect(port.TerminalDetachRequest{
		EndpointID: result.EndpointID, TerminalID: result.TerminalID, Channel: result.Channel,
		SurfaceID: result.SurfaceID, ViewID: result.ViewID, Session: result.Session, OperationID: "cleanup:" + result.OperationID,
	})}
}

func appendPreviousAttachmentCleanup(effects []Effect, previous state.TerminalViewBinding, current port.TerminalAttachResult) []Effect {
	if !previous.Attached || previous.Channel == 0 || previous.Session == nil {
		return effects
	}
	if previous.Channel == current.Channel && protoEndpointSessionEqual(previous.Session, current.Session) {
		return effects
	}
	return append(effects, terminalDetachEffect(port.TerminalDetachRequest{
		EndpointID: previous.EndpointID, TerminalID: previous.TerminalID, Channel: previous.Channel,
		SurfaceID: previous.SurfaceID, ViewID: previous.ViewID, Session: previous.AttachmentSession(), OperationID: "cleanup:" + current.OperationID,
	}))
}

func protoEndpointSessionEqual(left, right *apipb.EndpointSessionStamp) bool {
	return left != nil && right != nil && left.GetEndpointId() == right.GetEndpointId() && left.GetRouteId() == right.GetRouteId() && left.GetGeneration() == right.GetGeneration()
}

func liveAttachAppliedEffects(endpointID state.EndpointID, terminalID string, deps LiveDeps) []Effect {
	effects := workbenchPersistEffects("terminal.attach")
	if terminalID == "" {
		return effects
	}
	effects = append(effects, FuncEffect{Run: func(context.Context) Msg {
		return TerminalPoolListRequestMsg{EndpointID: endpointID}
	}})
	effects = append(effects, terminalEventWatchRequestEffects(endpointID, deps)...)
	return effects
}

func terminalEventWatchRequestEffects(endpointID state.EndpointID, deps LiveDeps) []Effect {
	if _, ok := deps.Terminal.(port.TerminalEventSource); !ok {
		return nil
	}
	return []Effect{FuncEffect{Run: func(context.Context) Msg {
		return TerminalEventWatchRequestMsg{EndpointID: endpointID}
	}}}
}

func logLiveAttachApplied(deps LiveDeps, root state.Root, result port.TerminalAttachResult, targetKind string) {
	logLifecycleTrace(deps.Logger, "live.attach.result",
		"target_kind", targetKind,
		"terminal_id", result.TerminalID,
		"view_id", result.ViewID,
		"surface_id", result.SurfaceID,
		"channel", result.Channel,
		"cols", result.Cols,
		"rows", result.Rows,
		"resize_policy", result.ResizePolicy,
		"can_resize", result.CanResize,
		"size_locked", result.SizeLocked,
		"control_reason", result.ControlReason,
		"owner_view_id", result.OwnerViewID,
		"surface_state", string(root.Surface.SurfaceForTerminalRef(state.NewTerminalRef(result.EndpointID, result.TerminalID)).State),
		"session_terminal", root.Session.TerminalID,
		"session_state", string(root.Session.State),
		"bindings", lifecycleTerminalViewBindingsSummary(root.TerminalViews.BindingsForTerminalRef(state.NewTerminalRef(result.EndpointID, result.TerminalID))),
	)
}

func applyLiveAttachRuntimeProjection(root state.Root, result port.TerminalAttachResult, viewID string) state.Root {
	ref := state.NewTerminalRef(result.EndpointID, result.TerminalID)
	if liveAttachResultOwnsActiveView(root, viewID, ref) {
		// 中文说明：全局 Session/Surface 只表达当前前台 view 的 live/input 投影。
		// 后台 restore attach 不能抢走 active pane 的 channel 和实时 surface。
		root.Session = root.Session.AttachRefWithResizeOwner(ref, result.Channel, result.Cols, result.Rows, result.ResizePolicy, result.SurfaceID, viewID)
		root.Surface = root.Surface.AttachRef(ref, result.Cols, result.Rows)
		return root
	}
	root.Session = root.Session.RecordInputChannelRef(ref, result.Channel)
	root.Surface = root.Surface.CacheAttachRef(ref, result.Cols, result.Rows)
	return root
}

func liveAttachResultOwnsActiveView(root state.Root, viewID string, ref state.TerminalRef) bool {
	if viewID == "" {
		return true
	}
	shell := root.Shell.ReadonlyDefaults()
	if activeFloatingID := shell.ActiveFloatingID(); activeFloatingID != "" {
		if viewID == root.TerminalViews.FloatingViewID(activeFloatingID) {
			return true
		}
	} else if shell.ActivePaneID != "" {
		if viewID == root.TerminalViews.PaneViewID(shell.ActivePaneID) {
			return true
		}
	}
	if root.Session.TerminalID == "" && root.Surface.TerminalID == "" {
		return true
	}
	if root.Session.TerminalRef().Equal(ref) && !root.Session.Attached && root.Session.Channel == 0 {
		return true
	}
	return false
}

func liveAttachDefaultViewID(root state.Root) string {
	shell := root.Shell.ReadonlyDefaults()
	// 中文说明：无显式 ViewID 的 root attach 是默认 tiled 入口语义；
	// floating attach 必须由调用方传入 floating binding 的 ViewID。
	return root.TerminalViews.PaneViewID(shell.ActivePaneID)
}

func liveAttachViewStillPresent(root state.Root, viewID string) bool {
	if viewID == "" {
		return false
	}
	if !root.Session.Attached && len(root.TerminalViews.Views) == 0 {
		// 初次 attach 时还没有任何 reducer-owned view binding，显式 ViewID 不能被误判成 stale。
		return true
	}
	if _, ok := root.TerminalViews.Views[viewID]; ok {
		return true
	}
	shell := root.Shell.ReadonlyDefaults()
	for _, tab := range shell.Workspace.Tabs {
		for _, pane := range tab.Panes {
			if liveAttachPaneViewID(root, pane.ID) == viewID {
				return true
			}
		}
	}
	for _, tab := range shell.Workspace.Tabs {
		for _, floating := range tab.Floatings {
			if liveAttachFloatingViewID(root, floating.ID) == viewID {
				return true
			}
		}
	}
	return false
}

type liveAttachViewTarget struct {
	PaneID     string
	FloatingID string
}

func liveAttachTargetForViewID(root state.Root, viewID string) (liveAttachViewTarget, bool) {
	if viewID == "" {
		return liveAttachViewTarget{}, false
	}
	if existing, ok := root.TerminalViews.Views[viewID]; ok {
		return liveAttachViewTarget{PaneID: existing.PaneID, FloatingID: existing.FloatingID}, true
	}
	shell := root.Shell.ReadonlyDefaults()
	for _, tab := range shell.Workspace.Tabs {
		for _, pane := range tab.Panes {
			if liveAttachPaneViewID(root, pane.ID) == viewID {
				return liveAttachViewTarget{PaneID: pane.ID}, true
			}
		}
	}
	for _, tab := range shell.Workspace.Tabs {
		for _, floating := range tab.Floatings {
			if liveAttachFloatingViewID(root, floating.ID) == viewID {
				return liveAttachViewTarget{PaneID: floating.Pane.ID, FloatingID: floating.ID}, true
			}
		}
	}
	return liveAttachViewTarget{}, false
}

func liveAttachPaneViewID(root state.Root, paneID string) string {
	return root.TerminalViews.PaneViewID(paneID)
}

func liveAttachFloatingViewID(root state.Root, floatingID string) string {
	return root.TerminalViews.FloatingViewID(floatingID)
}

func invalidateCopyModeForTerminalRebindRef(root state.Root, paneID string, viewID string, ref state.TerminalRef) (state.Root, []Effect) {
	ref = ref.Normalize()
	if viewID != "" {
		root = rootWithCopyHistorySessionForView(root, viewID)
	}
	if !copyModeInputContext(root.CopyMode) || ref.Empty() || state.NewTerminalRef(root.CopyMode.EndpointID, root.CopyMode.TerminalID).Equal(ref) {
		return root, nil
	}
	sameView := viewID != "" && root.CopyMode.ViewID == viewID
	samePane := paneID != "" && root.CopyMode.PaneID == paneID
	if !sameView && !samePane {
		return root, nil
	}
	effects := copyHistoryCleanupEffectsForView(root, viewID)
	if root.CopyMode.Entering {
		root.History = root.History.InvalidateWindow()
		root.History.EndpointID = ref.EndpointID
		root.History.TerminalID = ref.TerminalID
		root.CopyMode = state.CopyModeStore{}
		root = root.WithoutCopyHistorySession(viewID)
		return root, effects
	}
	// 当前 pane/view 已经重绑到新的 terminal，旧 frozen history 不能继续留在屏幕上。
	root.History = root.History.InvalidateWindow()
	root.History.EndpointID = ref.EndpointID
	root.History.TerminalID = ref.TerminalID
	root.CopyMode.PaneID = paneID
	root.CopyMode.ViewID = viewID
	root.CopyMode.EndpointID = ref.EndpointID
	root.CopyMode.TerminalID = ref.TerminalID
	root.CopyMode.BoundToken = ""
	root.CopyMode.BoundCols = 0
	root.CopyMode.ViewportTop = 0
	root.CopyMode.Cursor = state.CopyPosition{}
	root.CopyMode.Mark = nil
	root.CopyMode.Selection = nil
	root.CopyMode.Query = ""
	root.CopyMode.Matches = nil
	root.CopyMode.ActiveMatch = 0
	root.CopyMode.Empty = true
	root = saveCopyHistorySessionForView(root, viewID)
	return root, effects
}

func liveEffects(terminalID string, cols int, rows int, deps LiveDeps) []Effect {
	return liveEffectsForRef(state.DefaultEndpointID, terminalID, cols, rows, deps)
}

func liveEffectsForRef(endpointID state.EndpointID, terminalID string, cols int, rows int, deps LiveDeps) []Effect {
	return liveSurfaceEffectForRef(endpointID, terminalID, cols, rows, true, deps)
}

func reduceLiveLifecycleQuery(root state.Root, msg LiveLifecycleQueryMsg, deps LiveDeps) (state.Root, []Effect) {
	if len(msg.Targets) == 0 {
		return root, nil
	}
	seen := map[string]struct{}{}
	effects := make([]Effect, 0, len(msg.Targets))
	for _, target := range msg.Targets {
		terminalID := strings.TrimSpace(target.TerminalID)
		if terminalID == "" {
			continue
		}
		ref := state.NewTerminalRef(target.EndpointID, terminalID)
		if _, ok := seen[ref.Key()]; ok {
			continue
		}
		seen[ref.Key()] = struct{}{}
		cols, rows := target.Cols, target.Rows
		if cols <= 0 {
			cols = 80
		}
		if rows <= 0 {
			rows = 24
		}
		// 中文说明：这是按需向 core 查询 terminal lifecycle，不把 running/exited 权威性缓存进 TUI。
		logLifecycleTrace(deps.Logger, "live.lifecycle.query",
			"reason", msg.Reason,
			"endpoint_id", string(target.EndpointID),
			"terminal_id", terminalID,
			"cols", cols,
			"rows", rows,
			"bindings", lifecycleTerminalViewBindingsSummary(root.TerminalViews.BindingsForTerminalRef(ref)),
		)
		effects = append(effects, liveSurfaceEffectForRef(target.EndpointID, terminalID, cols, rows, true, deps)...)
	}
	return root, effects
}

func liveSurfaceEffect(terminalID string, cols int, rows int, knownLifecycle bool, deps LiveDeps) []Effect {
	return liveSurfaceEffectForRef(state.DefaultEndpointID, terminalID, cols, rows, knownLifecycle, deps)
}

func liveSurfaceEffectForRef(endpointID state.EndpointID, terminalID string, cols int, rows int, knownLifecycle bool, deps LiveDeps) []Effect {
	source, ok := deps.Terminal.(port.NativeScreenSource)
	if !ok || terminalID == "" {
		return nil
	}
	endpointID = state.NormalizeEndpointID(endpointID)
	return []Effect{FuncEffect{
		Async:            true,
		ForceSyncInTests: true,
		Run: func(ctx context.Context) Msg {
			finish := perftrace.Measure("tui.live_surface")
			result, err := source.LiveSurface(ctx, port.TerminalSurfaceRequest{
				EndpointID: endpointID,
				TerminalID: terminalID,
				Cols:       cols,
				Rows:       rows,
			})
			finish(liveSurfaceResultApproxBytes(result))
			if err != nil {
				logEffectError(deps.Logger, "live.surface", err, "terminal_id", terminalID)
				if isContextLifecycleError(err) {
					return nil
				}
				return LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{EndpointID: endpointID, TerminalID: terminalID}, Err: err, RequestedCols: cols, RequestedRows: rows}
			}
			result.Snapshot.EndpointID = endpointID
			if result.Snapshot.TerminalID == "" {
				result.Snapshot.TerminalID = terminalID
			}
			if result.Snapshot.Cols == 0 {
				result.Snapshot.Cols = cols
			}
			if result.Snapshot.Rows == 0 {
				result.Snapshot.Rows = rows
			}
			// 中文说明：普通 terminal.changed refresh 只表示 live projection 失效，
			// 不能因为 service 顺手查了 core lifecycle 就伪装成 lifecycle boundary；
			// 否则 app 队列会禁止 latest-only 丢弃旧帧。attach/显式 lifecycle query
			// 仍通过 knownLifecycle 保留权威边界。
			lifecycleKnown := knownLifecycle && result.LifecycleKnown
			return LiveSurfaceMsg{Snapshot: result.Snapshot, LifecycleKnown: lifecycleKnown, RequestedCols: cols, RequestedRows: rows}
		},
	}}
}

func liveScreenNextEffectForRef(request state.LiveScreenRequestState, deps LiveDeps) []Effect {
	source, ok := deps.Terminal.(port.LiveScreenSource)
	if !ok || request.TerminalID == "" {
		return nil
	}
	ref := request.TerminalRef()
	token := liveScreenNextTokenForRef(ref)
	observedRevision := request.SubmittedRevision
	if request.NeedsBootstrap {
		observedRevision = 0
	}
	return []Effect{FuncEffect{
		Token: token,
		Async: true,
		Run: func(ctx context.Context) Msg {
			logLiveScreenTrace(deps.Logger, "next.request",
				"endpoint_id", string(ref.EndpointID),
				"terminal_id", ref.TerminalID,
				"generation", request.Generation,
				"cols", request.Cols,
				"rows", request.Rows,
				"observed_revision", observedRevision,
			)
			result, err := source.LiveScreenNext(ctx, port.TerminalSurfaceRequest{
				EndpointID:       ref.EndpointID,
				TerminalID:       ref.TerminalID,
				Cols:             request.Cols,
				Rows:             request.Rows,
				ObservedRevision: observedRevision,
			})
			if err != nil {
				return LiveScreenNextResultMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, Generation: request.Generation, Err: err}
			}
			result.Snapshot.EndpointID = ref.EndpointID
			if result.Snapshot.TerminalID == "" {
				result.Snapshot.TerminalID = ref.TerminalID
			}
			logLiveScreenTrace(deps.Logger, "next.result",
				"endpoint_id", string(ref.EndpointID),
				"terminal_id", ref.TerminalID,
				"generation", request.Generation,
				"revision", result.Snapshot.Revision,
			)
			perftrace.Count("tui.live_screen_next", liveSnapshotApproxBytes(result.Snapshot))
			return LiveScreenNextResultMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, Generation: request.Generation, Snapshot: result.Snapshot}
		},
	}}
}

func liveScreenNextTokenForRef(ref state.TerminalRef) CancelToken {
	return CancelToken(liveScreenNextTokenPrefix + ref.Normalize().Key())
}

func liveSurfaceRefreshKey(ref state.TerminalRef) string {
	ref = ref.Normalize()
	if ref.Empty() {
		return ""
	}
	if ref.EndpointID == state.DefaultEndpointID {
		return ref.TerminalID
	}
	return ref.Key()
}

func liveSurfaceResultApproxBytes(result port.TerminalSurfaceResult) int {
	return liveSnapshotApproxBytes(result.Snapshot)
}

func liveEventApproxBytes(event port.TerminalLiveEvent) int {
	if event.Ready {
		return liveSnapshotApproxBytes(event.Snapshot)
	}
	if event.Refresh {
		return 1
	}
	return 0
}

func liveSnapshotApproxBytes(snapshot state.LiveSurfaceSnapshot) int {
	total := 0
	for _, line := range snapshot.Lines {
		total += len(line)
	}
	for _, row := range snapshot.Screen {
		for _, cell := range row {
			total += len(cell.Text)
		}
	}
	return total
}

func maybeRefreshFloatingAutoFit(root state.Root, terminalID string) (state.Root, []Effect) {
	if terminalID == "" {
		return root.Advance(), nil
	}
	shell := root.Shell.ReadonlyDefaults()
	for _, floating := range shell.ActiveFloatings() {
		if floating.Pane.TerminalID != terminalID || floating.FitMode != state.FloatingFitAuto {
			continue
		}
		next, effects := reduceFloatingCommand(root, state.FloatingCommand{
			Action:   state.FloatingCommandRefreshAutoFit,
			TargetID: floating.ID,
			Source:   state.PaneCommandSourceTest,
		})
		if len(effects) == 0 {
			return next, nil
		}
		return next, effects
	}
	return root.Advance(), nil
}

func liveResizeTerminalID(root state.Root, msg LiveResizeResultMsg) string {
	if msg.ViewID != "" {
		if binding, ok := root.TerminalViews.Views[msg.ViewID]; ok {
			return binding.TerminalID
		}
	}
	return root.Session.TerminalID
}

func reduceLiveEvent(root state.Root, msg LiveEventMsg, deps LiveDeps) (state.Root, []Effect) {
	event := msg.Event
	if event.TerminalID == "" {
		event.TerminalID = root.Surface.TerminalID
	}
	if event.EndpointID == "" && event.TerminalID == root.Surface.TerminalID {
		event.EndpointID = root.Surface.EndpointID
	}
	event.EndpointID = state.NormalizeEndpointID(event.EndpointID)
	eventRef := state.NewTerminalRef(event.EndpointID, event.TerminalID)
	if ordinaryLiveRefreshEvent(event) {
		// daemon 事件总线中的 terminal.changed 只是失效提示；实时屏幕唯一来源是
		// renderer submission 驱动的 LiveScreenNext，不允许再从被动事件启动第二条拉取链路。
		return root, nil
	}
	if event.Err != nil {
		if isContextLifecycleError(event.Err) {
			return root, nil
		}
		if next, ok := markTerminalExitedFromErrorRef(root, eventRef, event.Err); ok {
			return next.Advance(), nil
		}
		root = applyLiveErrorRef(root, eventRef, event.Err.Error())
		return root.Advance(), nil
	}
	if event.Metadata {
		root.TerminalPool = root.TerminalPool.ApplyTagsEditedRef(eventRef, event.Tags, "")
		if event.AttachmentProjection {
			root.TerminalPool = root.TerminalPool.ApplyAttachmentProjectionRef(eventRef, event.AttachmentCount)
			root.TerminalViews = root.TerminalViews.ApplyTerminalRefResizeControl(eventRef, state.TerminalResizeControlProjection{
				SizeLocked:     event.SizeLocked,
				ControlReason:  terminalResizeControlReason(event.SizeLocked),
				OwnerSurfaceID: event.OwnerSurfaceID,
				OwnerViewID:    event.OwnerViewID,
				ResizeEpoch:    event.ResizeEpoch,
			})
		}
		return root.Advance(), nil
	}
	if event.Exited {
		event.Snapshot.EndpointID = event.EndpointID
		event.Snapshot.State = state.TerminalLiveExited
		event.Snapshot.ExitCode = event.ExitCode
		event.Snapshot.ExitReason = event.Reason
		event.Snapshot.ExitedAt = event.ExitedAt
		event.Snapshot.Command = append([]string(nil), event.Command...)
		root.Surface = root.Surface.MarkExitedWithMetadataRef(eventRef, event.ExitCode, event.Reason, event.ExitedAt, event.Command)
		if root.Session.TerminalRef().Equal(eventRef) {
			root.Session = root.Session.MarkExitedWithMetadataRef(eventRef, event.ExitCode, event.Reason, event.ExitedAt, event.Command)
		}
	}
	if event.Ready {
		if event.Snapshot.EndpointID == "" {
			event.Snapshot.EndpointID = event.EndpointID
		}
		if event.Snapshot.TerminalID == "" {
			event.Snapshot.TerminalID = event.TerminalID
		}
		root.Surface = root.Surface.ApplySnapshotWithLifecycle(event.Snapshot, event.LifecycleKnown)
		snapshotRef := event.Snapshot.TerminalRef()
		if event.LifecycleKnown && event.Snapshot.State == state.TerminalLiveAttached && root.Session.TerminalRef().Equal(snapshotRef) {
			root.Session = root.Session.MarkAttachedRef(snapshotRef)
		}
		logLifecycleTrace(deps.Logger, "live.event",
			"endpoint_id", string(event.EndpointID),
			"terminal_id", event.Snapshot.TerminalID,
			"snapshot_state", string(event.Snapshot.State),
			"lifecycle_known", event.LifecycleKnown,
			"surface_state", string(root.Surface.State),
			"session_state", string(root.Session.State),
			"active_terminal", lifecycleActiveTerminalID(root),
		)
		return maybeRefreshFloatingAutoFit(root, event.Snapshot.TerminalID)
	}
	return root.Advance(), nil
}

func reduceTerminalEventWatchRequest(root state.Root, msg TerminalEventWatchRequestMsg, deps LiveDeps) (state.Root, []Effect) {
	source, ok := deps.Terminal.(port.TerminalEventSource)
	if !ok {
		return root, nil
	}
	endpointID := state.NormalizeEndpointID(msg.EndpointID)
	return root, []Effect{StreamEffect{
		Token: CancelToken(terminalEventWatchTokenPrefix + string(endpointID)),
		Run: func(ctx context.Context, post func(Msg)) {
			events, err := source.WatchTerminalEvents(ctx, endpointID)
			if err != nil {
				if !isContextLifecycleError(err) {
					logEffectError(deps.Logger, "terminal.events.watch", err)
				}
				postTerminalEventWatchRetry(ctx, post, endpointID)
				return
			}
			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-events:
					if !ok {
						postTerminalEventWatchRetry(ctx, post, endpointID)
						return
					}
					post(LiveEventMsg{Event: event})
				}
			}
		},
	}}
}

func postTerminalEventWatchRetry(ctx context.Context, post func(Msg), endpointID state.EndpointID) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
		post(TerminalEventWatchRequestMsg{EndpointID: endpointID})
	}
}

func ordinaryLiveRefreshEvent(event port.TerminalLiveEvent) bool {
	return event.Refresh && event.Err == nil && !event.Exited && !event.LifecycleKnown && !event.Metadata && !event.Ready
}

func ordinaryLiveSurfaceWasInvalidatedWhileInFlight(root state.Root, msg LiveSurfaceMsg) bool {
	if !ordinaryLiveSurfaceResult(msg) {
		return false
	}
	refresh, ok := root.Surface.Refreshes[liveSurfaceRefreshKey(msg.Snapshot.TerminalRef())]
	return ok && refresh.InFlight && refresh.Dirty
}

func terminalResizeControlReason(sizeLocked bool) string {
	if sizeLocked {
		return "size_locked"
	}
	return ""
}

func applyTerminalAttachmentProjectionFromAttach(root state.Root, result port.TerminalAttachResult) state.Root {
	if result.TerminalID == "" || result.AttachmentCount <= 0 {
		return root
	}
	root.TerminalPool = root.TerminalPool.ApplyAttachmentProjectionRef(state.NewTerminalRef(result.EndpointID, result.TerminalID), result.AttachmentCount)
	return root
}

func applyTerminalAttachmentProjectionFromResize(root state.Root, result port.TerminalResizeResult) state.Root {
	if result.TerminalID == "" || result.AttachmentCount <= 0 {
		return root
	}
	root.TerminalPool = root.TerminalPool.ApplyAttachmentProjectionRef(state.NewTerminalRef(result.EndpointID, result.TerminalID), result.AttachmentCount)
	return root
}

func maybeScheduleDirtyLiveSurfaceRefreshRef(root state.Root, ref state.TerminalRef, fallbackCols int, fallbackRows int, deps LiveDeps) (state.Root, []Effect) {
	var cols, rows int
	var shouldFetch bool
	ref = ref.Normalize()
	root.Surface, cols, rows, shouldFetch = root.Surface.ConsumeDirtyRefreshRef(ref)
	if !shouldFetch {
		return root, nil
	}
	perftrace.Count("tui.live_refresh_followup", 0)
	if cols <= 0 || rows <= 0 {
		cols, rows = fallbackCols, fallbackRows
	}
	if cols <= 0 || rows <= 0 {
		cols, rows = liveSurfaceRefreshSizeRef(root, ref)
	}
	return root, liveSurfaceEffectForRef(ref.EndpointID, ref.TerminalID, cols, rows, false, deps)
}

func reduceLiveScreenFrameSelected(root state.Root, msg LiveScreenFrameSelectedMsg, deps LiveDeps) (state.Root, []Effect) {
	if _, ok := deps.Terminal.(port.LiveScreenSource); !ok {
		return root, nil
	}
	refs := make([]state.TerminalRef, 0, len(msg.Targets))
	for _, target := range msg.Targets {
		ref := state.NewTerminalRef(state.EndpointID(target.EndpointID), target.TerminalID)
		if !ref.Empty() {
			refs = append(refs, ref)
		}
	}
	var effects []Effect
	if msg.Full {
		var canceled []state.TerminalRef
		root.Surface, canceled = root.Surface.ReconcileLiveScreenDemand(refs)
		for _, ref := range canceled {
			effects = append(effects, CancelEffect{Token: liveScreenNextTokenForRef(ref)})
		}
	}
	for _, target := range msg.Targets {
		ref := state.NewTerminalRef(state.EndpointID(target.EndpointID), target.TerminalID)
		if ref.Empty() {
			continue
		}
		surface := root.Surface.SurfaceForTerminalRef(ref)
		if surface.State == state.TerminalLiveExited || surface.State == state.TerminalLiveError {
			continue
		}
		cols, rows := liveSurfaceRefreshSizeRef(root, ref)
		root.Surface = root.Surface.SubmitLiveScreenRef(ref, target.Revision, cols, rows)
		var request state.LiveScreenRequestState
		var start bool
		root.Surface, request, start = root.Surface.BeginLiveScreenRequestRef(ref)
		if start {
			effects = append(effects, liveScreenNextEffectForRef(request, deps)...)
		}
	}
	return root, effects
}

func reduceLiveScreenNextResult(root state.Root, msg LiveScreenNextResultMsg, deps LiveDeps) (state.Root, []Effect) {
	ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	if !root.Surface.LiveScreenRequestMatches(ref, msg.Generation) {
		return root, nil
	}
	if msg.Err != nil {
		root.Surface, _ = root.Surface.FinishLiveScreenRequestRef(ref, msg.Generation, 0)
		if isContextLifecycleError(msg.Err) {
			return root, nil
		}
		if next, ok := markTerminalExitedFromErrorRef(root, ref, msg.Err); ok {
			return next.Advance(), nil
		}
		root = applyLiveErrorRef(root, ref, msg.Err.Error())
		return root.Advance(), nil
	}
	msg.Snapshot.EndpointID = ref.EndpointID
	if msg.Snapshot.TerminalID == "" {
		msg.Snapshot.TerminalID = ref.TerminalID
	}
	root.Surface = root.Surface.ApplySnapshot(msg.Snapshot)
	received := root.Surface.SurfaceForTerminalRef(ref).Revision
	if msg.Snapshot.BaseRevision != 0 && !msg.Snapshot.FullReplace && received != msg.Snapshot.Revision {
		root.Surface, _ = root.Surface.RequireLiveScreenBootstrap(ref, msg.Generation)
		logLiveScreenTrace(deps.Logger, "next.bootstrap_required",
			"endpoint_id", string(ref.EndpointID),
			"terminal_id", ref.TerminalID,
			"generation", msg.Generation,
			"base_revision", msg.Snapshot.BaseRevision,
			"received_revision", received,
		)
		return root.Advance(), nil
	}
	root.Surface, _ = root.Surface.FinishLiveScreenRequestRef(ref, msg.Generation, received)
	logLiveScreenTrace(deps.Logger, "next.cached",
		"endpoint_id", string(ref.EndpointID),
		"terminal_id", ref.TerminalID,
		"generation", msg.Generation,
		"received_revision", received,
	)
	return root.Advance(), nil
}

func logLiveScreenTrace(logger *slog.Logger, event string, attrs ...any) {
	if logger == nil || !diagnosticsEnabledFromEnv(tuiDiagnosticsEnv) {
		return
	}
	args := append([]any{"event", event}, attrs...)
	logger.Info("tui-v3 live screen", args...)
}

func liveSurfaceRefreshSizeRef(root state.Root, ref state.TerminalRef) (int, int) {
	ref = ref.Normalize()
	if binding, ok := root.TerminalViews.OwnerBindingRef(ref); ok {
		if binding.DesiredCols > 0 && binding.DesiredRows > 0 {
			return binding.DesiredCols, binding.DesiredRows
		}
	}
	if binding, ok := activeTerminalViewBinding(root); ok && binding.TerminalRef().Equal(ref) {
		if binding.DesiredCols > 0 && binding.DesiredRows > 0 {
			return binding.DesiredCols, binding.DesiredRows
		}
	}
	if root.Session.TerminalRef().Equal(ref) {
		if cols, rows := root.Session.DesiredSize(); cols > 0 && rows > 0 {
			return cols, rows
		}
	}
	surface := root.Surface.SurfaceForTerminalRef(ref)
	if surface.Cols > 0 && surface.Rows > 0 {
		return surface.Cols, surface.Rows
	}
	return 80, 24
}

type liveInputTargetInfo struct {
	PaneID        string
	FloatingID    string
	ViewID        string
	EndpointID    state.EndpointID
	TerminalID    string
	Channel       uint16
	ResizeRole    string
	SurfaceID     string
	DesiredCols   int
	DesiredRows   int
	Floating      bool
	AttachPending bool
	Session       *apipb.EndpointSessionStamp
}

func liveInputTarget(root state.Root) (liveInputTargetInfo, bool) {
	shell := root.Shell.ReadonlyDefaults()
	if activeFloatingID := shell.ActiveFloatingID(); activeFloatingID != "" {
		binding, ok := root.TerminalViews.FloatingBinding(activeFloatingID)
		if !ok || binding.TerminalID == "" || binding.Unresolved {
			return liveInputTargetInfo{}, false
		}
		return liveInputTargetFromBinding(binding), true
	}
	pane, ok := shell.Pane(state.PaneCommandTarget{PaneID: shell.ActivePaneID})
	if !ok {
		return liveInputTargetInfo{}, false
	}
	binding, ok := root.TerminalViews.PaneBinding(pane.ID)
	if !ok || binding.TerminalID == "" || binding.Unresolved {
		return liveInputTargetInfo{}, false
	}
	return liveInputTargetFromBinding(binding), true
}

func liveInputTargetForView(root state.Root, viewID string) (liveInputTargetInfo, bool) {
	if viewID == "" {
		return liveInputTargetInfo{}, false
	}
	binding, ok := root.TerminalViews.Views[viewID]
	if !ok || binding.TerminalID == "" || binding.Unresolved {
		return liveInputTargetInfo{}, false
	}
	return liveInputTargetFromBinding(binding), true
}

func liveInputTargetFromBinding(binding state.TerminalViewBinding) liveInputTargetInfo {
	info := liveInputTargetInfo{
		PaneID:        binding.PaneID,
		FloatingID:    binding.FloatingID,
		ViewID:        binding.ViewID,
		EndpointID:    state.NormalizeEndpointID(binding.EndpointID),
		TerminalID:    binding.TerminalID,
		Channel:       binding.Channel,
		ResizeRole:    binding.ResizeRole,
		SurfaceID:     binding.SurfaceID,
		DesiredCols:   binding.DesiredCols,
		DesiredRows:   binding.DesiredRows,
		Floating:      binding.FloatingID != "",
		AttachPending: binding.AttachPending,
		Session:       binding.AttachmentSession(),
	}
	return info
}

func liveAttachForInputEffect(root state.Root, target liveInputTargetInfo, event input.InputEvent, bytes []byte, deps LiveDeps) (state.Root, Effect) {
	payload := append([]byte(nil), bytes...)
	request := liveInputAttachRequest(root, target)
	binding := state.TerminalViewBinding{
		ViewID: target.ViewID, SurfaceID: request.SurfaceID, EndpointID: target.EndpointID, TerminalID: target.TerminalID,
		ResizeRole: request.ResizePolicy, DesiredCols: request.Cols, DesiredRows: request.Rows, PaneID: target.PaneID, FloatingID: target.FloatingID,
	}
	var candidate state.TerminalAttachCandidate
	root.TerminalViews, candidate = root.TerminalViews.BeginAttach(binding)
	request.OperationID = candidate.OperationID
	return root, FuncEffect{
		Async:            true,
		ForceSyncInTests: true,
		Run: func(ctx context.Context) Msg {
			result, err := deps.Terminal.Attach(ctx, request)
			return LiveInputAttachResultMsg{Target: target, Event: event, Bytes: payload, OperationID: candidate.OperationID, Result: result, Err: err}
		},
	}
}

func liveInputAttachRequest(root state.Root, target liveInputTargetInfo) port.TerminalAttachRequest {
	cols, rows := liveInputAttachSize(root, target)
	resizePolicy := target.ResizeRole
	if resizePolicy == "" {
		resizePolicy = state.TerminalResizeRoleFollower
	}
	surfaceID := target.SurfaceID
	if surfaceID == "" {
		surfaceID = runtimeSurfaceID(root)
	}
	return port.TerminalAttachRequest{
		EndpointID:   target.EndpointID,
		TerminalID:   target.TerminalID,
		Cols:         cols,
		Rows:         rows,
		Mode:         "collaborator",
		ResizePolicy: resizePolicy,
		SurfaceID:    surfaceID,
		ViewID:       target.ViewID,
	}
}

func liveInputAttachSize(root state.Root, target liveInputTargetInfo) (int, int) {
	if rect, ok := terminalPoolTargetContentRect(root, terminalPoolTarget{PaneID: target.PaneID, FloatingID: target.FloatingID, ViewID: target.ViewID}, render.Rect{}); ok {
		return rect.W, rect.H
	}
	if target.DesiredCols > 0 && target.DesiredRows > 0 {
		return target.DesiredCols, target.DesiredRows
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

func reduceLiveInputAttachResult(root state.Root, msg LiveInputAttachResultMsg, deps LiveDeps) (state.Root, []Effect) {
	if msg.Err != nil {
		return reduceLiveAttachResult(root, LiveAttachResultMsg{EndpointID: msg.Target.EndpointID, TerminalID: msg.Target.TerminalID, ViewID: msg.Target.ViewID, RequestedResizePolicy: msg.Target.ResizeRole, OperationID: msg.OperationID, Err: msg.Err}, deps)
	}
	result := msg.Result
	if result.EndpointID == "" {
		result.EndpointID = state.NormalizeEndpointID(msg.Target.EndpointID)
	}
	if result.TerminalID == "" {
		result.TerminalID = msg.Target.TerminalID
	}
	if result.ViewID == "" {
		result.ViewID = msg.Target.ViewID
	}
	if result.SurfaceID == "" {
		result.SurfaceID = msg.Target.SurfaceID
	}
	if result.ResizePolicy == "" {
		result.ResizePolicy = msg.Target.ResizeRole
	}
	if result.ResizePolicy == "" {
		result.ResizePolicy = state.TerminalResizeRoleFollower
	}
	if result.OperationID == "" {
		result.OperationID = msg.OperationID
	}
	if result.Cols <= 0 || result.Rows <= 0 {
		cols, rows := liveInputAttachSize(root, msg.Target)
		if result.Cols <= 0 {
			result.Cols = cols
		}
		if result.Rows <= 0 {
			result.Rows = rows
		}
	}
	next, effects := reduceLiveAttachResult(root, LiveAttachResultMsg{EndpointID: msg.Target.EndpointID, TerminalID: msg.Target.TerminalID, ViewID: msg.Target.ViewID, RequestedResizePolicy: msg.Target.ResizeRole, OperationID: msg.OperationID, Result: result}, deps)
	if !liveInputAttachResultOwnsCurrentBinding(next, result, msg.OperationID) {
		return next, effects
	}
	target, ok := liveInputTargetForView(next, result.ViewID)
	if !ok || target.Channel == 0 {
		return next, effects
	}
	var operationID string
	next.TerminalViews, operationID = next.TerminalViews.NextTerminalOperation(inputOperationKind(msg.Event), target.ViewID)
	effects = append(effects, terminalSendInputEffect(target, msg.Event, msg.Bytes, operationID, deps))
	return next, effects
}

func liveInputAttachResultOwnsCurrentBinding(root state.Root, result port.TerminalAttachResult, operationID string) bool {
	binding, ok := root.TerminalViews.Views[result.ViewID]
	if !ok || !binding.Attached || binding.OperationID != operationID || binding.Channel != result.Channel {
		return false
	}
	if !binding.TerminalRef().Equal(state.NewTerminalRef(result.EndpointID, result.TerminalID)) {
		return false
	}
	return protoEndpointSessionEqual(binding.Session, result.Session)
}

func liveInputResultOwnsCurrentBinding(root state.Root, msg LiveInputResultMsg) bool {
	binding, ok := root.TerminalViews.Views[msg.ViewID]
	if !ok || !binding.Attached || binding.Channel != msg.Channel {
		return false
	}
	if !binding.TerminalRef().Equal(state.NewTerminalRef(msg.EndpointID, msg.TerminalID)) {
		return false
	}
	return protoEndpointSessionEqual(binding.Session, msg.Session)
}

func liveMousePassthroughEnabled(root state.Root, event input.InputEvent, target liveInputTargetInfo) bool {
	if event.Kind != input.EventKindMouse || event.RawSeq == "" {
		return false
	}
	if root.Shell.ReadonlyDefaults().Overlay.Open || copyModeOwnsActiveInput(root) {
		return false
	}
	if target.TerminalID == "" {
		return false
	}
	return root.Surface.SurfaceForTerminalRef(state.NewTerminalRef(target.EndpointID, target.TerminalID)).Modes.MousePassthroughEnabled()
}

func liveResizeShouldPreserveDisconnectedError(root state.Root, msg LiveResizeMsg) bool {
	if root.Session.State == state.TerminalLiveError && root.Session.LastError != "" {
		return true
	}
	ref := state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	if msg.ViewID != "" {
		if binding, ok := root.TerminalViews.Views[msg.ViewID]; ok {
			if binding.LastError != "" && !binding.Attached {
				return true
			}
			ref = binding.TerminalRef()
		}
	}
	surface := root.Surface.SurfaceForTerminalRef(ref)
	// 中文说明：断线后的排队 resize 只能被丢弃，不能把 transport/protocol 错误
	// 覆盖成泛化的 "terminal is not attached"，否则 pane 无法按 endpoint 错误分类展示。
	return surface.State == state.TerminalLiveError && surface.Err != ""
}

func reduceLiveResize(root state.Root, msg LiveResizeMsg, deps LiveDeps) (state.Root, []Effect) {
	if deps.Terminal == nil {
		return setLiveError(root, "terminal service missing"), nil
	}
	if msg.ViewID == "" {
		root, msg = adoptActiveOwnerResizeBinding(root, msg)
	}
	var targetBinding state.TerminalViewBinding
	if msg.ViewID != "" {
		binding, ok := root.TerminalViews.Views[msg.ViewID]
		if !ok {
			// A queued view-scoped resize must never borrow the active session's channel.
			return root, nil
		}
		if !binding.Attached || binding.Channel == 0 {
			return root, nil
		}
		if !msg.TakeOwnership && !binding.HasAuthoritativeResizeOwner() {
			return root, nil
		}
		targetBinding = binding
	} else if !root.Session.Attached {
		if liveResizeShouldPreserveDisconnectedError(root, msg) {
			return root, nil
		}
		return setLiveError(root, "terminal is not attached"), nil
	}
	if msg.Seq == 0 {
		if msg.ViewID != "" {
			msg.Seq = targetBinding.RequestSeq
		} else {
			root.Session = root.Session.RequestResize(msg.Cols, msg.Rows)
			msg.Seq = root.Session.ResizeRequestSeq
		}
	}
	session := root.Session
	var attachmentSession *apipb.EndpointSessionStamp
	if msg.EndpointID != "" {
		session.EndpointID = state.NormalizeEndpointID(msg.EndpointID)
	}
	if msg.ViewID != "" {
		session.EndpointID = targetBinding.EndpointID
		session.TerminalID = targetBinding.TerminalID
		session.Channel = targetBinding.Channel
		session.SurfaceID = targetBinding.SurfaceID
		session.ViewID = targetBinding.ViewID
		// 中文说明：view-scoped resize 的权限身份必须来自目标 binding；
		// restore 多个 view 后，全局 session 可能已被最后一个 follower attach 覆盖。
		session.ResizePolicy = targetBinding.ResizeRole
		if msg.TakeOwnership {
			session.ResizePolicy = state.TerminalResizeRoleOwner
		}
		attachmentSession = targetBinding.AttachmentSession()
	}
	var operationID string
	root.TerminalViews, operationID = root.TerminalViews.NextTerminalOperation("resize", session.ViewID)
	return root, []Effect{FuncEffect{
		Async:            true,
		ForceSyncInTests: true,
		Run: func(ctx context.Context) Msg {
			result, err := deps.Terminal.Resize(ctx, port.TerminalResizeRequest{
				EndpointID:         session.EndpointID,
				TerminalID:         session.TerminalID,
				Channel:            session.Channel,
				Cols:               msg.Cols,
				Rows:               msg.Rows,
				ResizePolicy:       session.ResizePolicy,
				TakeOwnership:      msg.TakeOwnership,
				ExpectedOwnerEpoch: msg.ExpectedOwnerEpoch,
				SurfaceID:          session.SurfaceID,
				ViewID:             session.ViewID,
				Session:            attachmentSession,
				OperationID:        operationID,
			})
			if result.EndpointID == "" {
				result.EndpointID = session.EndpointID
			}
			if result.TerminalID == "" {
				result.TerminalID = session.TerminalID
			}
			return LiveResizeResultMsg{EndpointID: session.EndpointID, TerminalID: session.TerminalID, Result: result, Cols: msg.Cols, Rows: msg.Rows, Seq: msg.Seq, ViewID: msg.ViewID, Session: attachmentSession, OperationID: operationID, TakeOwnership: msg.TakeOwnership, Err: err}
		},
	}}
}

func adoptActiveOwnerResizeBinding(root state.Root, msg LiveResizeMsg) (state.Root, LiveResizeMsg) {
	viewID := root.Session.ViewID
	if viewID == "" {
		return root, msg
	}
	binding, ok := root.TerminalViews.Views[viewID]
	if !ok || binding.TerminalID == "" || binding.TerminalID != root.Session.TerminalID {
		return root, msg
	}
	nextViews, decision := root.TerminalViews.RequestViewResize(viewID, msg.Cols, msg.Rows)
	if !decision.Allowed {
		return root, msg
	}
	root.TerminalViews = nextViews
	msg.EndpointID = binding.EndpointID
	msg.TerminalID = binding.TerminalID
	msg.ViewID = viewID
	// 裸 LiveResizeMsg 如果其实命中当前 owner view，就必须同步 view-local desired size。
	// 否则 resize 结果回来后，stale recovery 还会拿旧 binding 尺寸把 PTY 拉回去。
	if decision.Changed {
		msg.Seq = decision.Seq
	}
	return root, msg
}

func hasResizeControlResult(result port.TerminalResizeResult) bool {
	return result.TerminalID != "" || result.ControlReason != "" || result.SizeLocked || result.ResizeEpoch != 0 || result.OwnerSurfaceID != "" || result.OwnerViewID != "" || result.ResizePolicy != "" || result.SurfaceID != "" || result.ViewID != "" || result.CanResize || result.Resized
}

func liveResizeResultConflictsWithLocalOwner(root state.Root, binding state.TerminalViewBinding, result port.TerminalResizeResult) bool {
	if !hasResizeControlResult(result) || result.TerminalID == "" || result.OwnerViewID != binding.ViewID {
		return false
	}
	if binding.HasResizeOwner() {
		return false
	}
	owner, ok := root.TerminalViews.OwnerBindingRef(binding.TerminalRef())
	if !ok {
		return false
	}
	// 中文说明：旧 owner 的异步 resize result 可能晚于用户 take-owner 返回；
	// 若本地已有另一个 owner，就不能让旧 result 再把 ownership 投影抢回去。
	return owner.ViewID != "" && owner.ViewID != binding.ViewID
}

func resizeControlProjectionFromResult(result port.TerminalResizeResult) state.TerminalResizeControlProjection {
	return state.TerminalResizeControlProjection{
		CanResize:      result.CanResize,
		SizeLocked:     result.SizeLocked,
		ControlReason:  result.ControlReason,
		OwnerSurfaceID: result.OwnerSurfaceID,
		OwnerViewID:    result.OwnerViewID,
		ResizeEpoch:    result.ResizeEpoch,
		ResizeRole:     result.ResizePolicy,
		SurfaceID:      result.SurfaceID,
		ViewID:         result.ViewID,
	}
}

func shouldRecoverOwnerDesiredAfterResizeResult(root state.Root, msg LiveResizeResultMsg) bool {
	if !msg.Result.Resized {
		return false
	}
	ref := staleResizeResultRef(root, msg)
	if ref.Empty() {
		return false
	}
	desiredCols, desiredRows, _, ok := desiredResizeForTerminalRef(root, ref)
	if !ok {
		return false
	}
	cols, rows := resolvedResizeResultSize(msg)
	return cols != desiredCols || rows != desiredRows
}

func recoverLatestResizeAfterStaleResult(root state.Root, msg LiveResizeResultMsg) (state.Root, []Effect) {
	ref := staleResizeResultRef(root, msg)
	if ref.Empty() {
		return root, nil
	}
	desiredCols, desiredRows, viewID, ok := desiredResizeForTerminalRef(root, ref)
	if !ok {
		return root, nil
	}
	actualCols, actualRows := resolvedResizeResultSize(msg)
	if desiredCols == actualCols && desiredRows == actualRows {
		return root, nil
	}
	// 旧 resize 请求可能已经真正把 PTY 改回过期尺寸；
	// stale guard 丢掉结果后，还要立即重申当前 owner 的最新期望尺寸，避免卡在旧 geometry。
	var seq uint64
	if viewID != "" {
		binding, exists := root.TerminalViews.Views[viewID]
		if !exists || !binding.Attached || binding.Channel == 0 {
			return root, nil
		}
		seq = binding.RequestSeq
	} else {
		root.Session = root.Session.RequestResize(desiredCols, desiredRows)
		seq = root.Session.ResizeRequestSeq
	}
	return root, []Effect{FuncEffect{
		Run: func(context.Context) Msg {
			return LiveResizeMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, Cols: desiredCols, Rows: desiredRows, Seq: seq, ViewID: viewID}
		},
	}}
}

func staleResizeResultRef(root state.Root, msg LiveResizeResultMsg) state.TerminalRef {
	if msg.Result.TerminalID != "" {
		endpointID := msg.Result.EndpointID
		if endpointID == "" {
			endpointID = msg.EndpointID
		}
		return state.NewTerminalRef(endpointID, msg.Result.TerminalID)
	}
	if msg.ViewID != "" {
		if binding, ok := root.TerminalViews.Views[msg.ViewID]; ok {
			return binding.TerminalRef()
		}
	}
	if msg.TerminalID != "" {
		return state.NewTerminalRef(msg.EndpointID, msg.TerminalID)
	}
	return root.Session.TerminalRef()
}

func desiredResizeForTerminalRef(root state.Root, ref state.TerminalRef) (int, int, string, bool) {
	ref = ref.Normalize()
	if binding, ok := root.TerminalViews.OwnerBindingRef(ref); ok {
		if binding.DesiredCols > 0 && binding.DesiredRows > 0 {
			return binding.DesiredCols, binding.DesiredRows, binding.ViewID, true
		}
	}
	if !root.Session.TerminalRef().Equal(ref) {
		return 0, 0, "", false
	}
	cols, rows := root.Session.DesiredSize()
	if cols <= 0 || rows <= 0 {
		return 0, 0, "", false
	}
	return cols, rows, root.Session.ViewID, true
}

func resolvedResizeResultSize(msg LiveResizeResultMsg) (int, int) {
	cols, rows := msg.Cols, msg.Rows
	if msg.Result.Cols > 0 {
		cols = msg.Result.Cols
	}
	if msg.Result.Rows > 0 {
		rows = msg.Result.Rows
	}
	return cols, rows
}

func reduceLiveResizeError(root state.Root, msg LiveResizeResultMsg) (state.Root, []Effect) {
	ref := staleResizeResultRef(root, msg)
	if next, ok := markTerminalExitedFromErrorRef(root, ref, msg.Err); ok {
		return next.Advance(), nil
	}
	message := msg.Err.Error()
	if msg.ViewID != "" {
		if binding, ok := root.TerminalViews.Views[msg.ViewID]; ok {
			if msg.TakeOwnership {
				root.TerminalViews, _ = root.TerminalViews.ApplyResizeResult(msg.ViewID, msg.Seq, binding.DesiredCols, binding.DesiredRows, "")
				root.Shell = root.Shell.EnsureDefaults().AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.owner", Body: "take owner failed: " + message})
				return root.Advance(), nil
			}
			root.TerminalViews, _ = root.TerminalViews.ApplyResizeResult(msg.ViewID, msg.Seq, binding.DesiredCols, binding.DesiredRows, message)
			bindingRef := binding.TerminalRef()
			if liveErrorRefOwnsActiveProjection(root, bindingRef) {
				root = applyLiveErrorRefWithActive(root, bindingRef, message, true)
			} else {
				root.Surface = root.Surface.SetErrorRef(bindingRef, message)
			}
			return root.Advance(), nil
		}
	}
	root = applyLiveErrorRef(root, ref, message)
	return root.Advance(), nil
}

func setLiveError(root state.Root, message string) state.Root {
	return applyLiveErrorRef(root, state.TerminalRef{}, message).Advance()
}

func applyLiveAttachErrorRef(root state.Root, ref state.TerminalRef, viewID string, message string) state.Root {
	ref = ref.Normalize()
	if message == "" {
		message = "unknown terminal error"
	}
	if ref.Empty() || liveAttachResultOwnsActiveView(root, viewID, ref) {
		return applyLiveErrorRefWithActive(root, ref, message, true)
	}
	// 中文说明：后台 restore/reattach 失败只属于该 view/ref；前台 live 投影仍由 active view 拥有。
	root.Surface = root.Surface.SetErrorRef(ref, message)
	return root
}

func applyLiveErrorRef(root state.Root, ref state.TerminalRef, message string) state.Root {
	return applyLiveErrorRefWithActive(root, ref, message, liveErrorRefOwnsActiveProjection(root, ref))
}

func applyLiveErrorRefWithActive(root state.Root, ref state.TerminalRef, message string, active bool) state.Root {
	if message == "" {
		message = "unknown terminal error"
	}
	ref = ref.Normalize()
	if ref.Empty() {
		root.Session = root.Session.SetError(message)
		root.Surface = root.Surface.SetError(message)
		return root
	}
	root.TerminalViews = root.TerminalViews.MarkTerminalRefRuntimeError(ref, message)
	root = markEndpointOfflineFromLiveError(root, ref, message)
	if active {
		root.Session = root.Session.SetErrorRef(ref, message)
		root.Surface = root.Surface.SetErrorRef(ref, message)
		return root
	}
	// 中文说明：endpoint/list/live 的局部失败不能升级成全局 terminal 错误。
	// 非 active ref 只更新它自己的 surface 缓存，等待对应 pane/floating 被聚焦时展示。
	root.Surface = root.Surface.SetErrorRef(ref, message)
	return root
}

func markEndpointOfflineFromLiveError(root state.Root, ref state.TerminalRef, message string) state.Root {
	ref = ref.Normalize()
	if ref.Empty() || ref.EndpointID == state.DefaultEndpointID {
		return root
	}
	kind := state.ClassifyEndpointErrorText(message)
	if kind == state.EndpointErrorUnknown || kind == state.EndpointErrorUnavailable {
		return root
	}
	// 中文说明：live screen next/surface/input 的 transport/protocol 错误已经证明
	// owning endpoint 的连接不可用；这里回投到 EndpointStore，保证 picker/manager/workbench
	// 与 pane 使用同一份 endpoint-scoped 错误投影。
	root.Endpoints = root.Endpoints.MarkRuntimeStatus(ref.EndpointID, state.EndpointStatusOffline, kind, endpointTerminalCount(root, ref.EndpointID), message)
	return root
}

func liveErrorRefOwnsActiveProjection(root state.Root, ref state.TerminalRef) bool {
	ref = ref.Normalize()
	if ref.Empty() {
		return true
	}
	if root.Session.TerminalRef().Equal(ref) || root.Surface.TerminalRef().Equal(ref) {
		return true
	}
	if binding, ok := activeTerminalBinding(root); ok && binding.TerminalRef().Equal(ref) {
		return true
	}
	return root.Session.TerminalID == "" && root.Surface.TerminalID == ""
}

func markTerminalExitedFromErrorRef(root state.Root, ref state.TerminalRef, err error) (state.Root, bool) {
	if err == nil {
		return root, false
	}
	ref = ref.Normalize()
	if ref.Empty() {
		ref = root.Session.TerminalRef()
	}
	if ref.Empty() {
		ref = root.Surface.TerminalRef()
	}
	if terminalNotFoundError(err) {
		return removeTerminalRefFromRoot(root, ref), true
	}
	if !strings.Contains(strings.ToLower(err.Error()), "terminal exited") {
		return root, false
	}
	root.Session = root.Session.MarkExitedWithMetadataRef(ref, 0, "", time.Time{}, nil)
	root.Surface = root.Surface.MarkExitedWithMetadataRef(ref, 0, "", time.Time{}, nil)
	return root, true
}

func terminalNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "terminal not found") || strings.Contains(text, "protocol error 404")
}

func setLiveInputViewError(root state.Root, ref state.TerminalRef, viewID, message string) state.Root {
	if message == "" {
		message = "unknown terminal input error"
	}
	if viewID == "" {
		return applyLiveErrorRef(root, ref, message)
	}
	binding, ok := root.TerminalViews.Views[viewID]
	if !ok || !binding.TerminalRef().Equal(ref) {
		return root
	}
	root.TerminalViews = root.TerminalViews.MarkViewRuntimeError(viewID, message)
	root.Surface = root.Surface.SetErrorRef(ref, message)
	if liveAttachResultOwnsActiveView(root, viewID, ref) {
		root.Session = root.Session.SetError(message)
	}
	return root
}

package state

import "time"

type TerminalLiveState string

const (
	TerminalLivePending  TerminalLiveState = "pending"
	TerminalLiveAttached TerminalLiveState = "attached"
	TerminalLiveExited   TerminalLiveState = "exited"
	TerminalLiveError    TerminalLiveState = "error"
)

// TerminalSurfaceStore 保存当前实时 terminal surface 投影，不是历史 truth。
type TerminalSurfaceStore struct {
	EndpointID     EndpointID
	TerminalID     string
	Revision       uint64
	Cols           int
	Rows           int
	Lines          []string
	Screen         [][]LiveCell
	Title          string
	Cursor         LiveCursor
	Modes          LiveTerminalModes
	Ready          bool
	State          TerminalLiveState
	ExitCode       int
	ExitReason     string
	ExitedAt       time.Time
	Command        []string
	Err            string
	ResizeBoundary LiveResizeBoundary
	Surfaces       map[string]LiveSurfaceSnapshot
	Refreshes      map[string]LiveSurfaceRefreshState
	LiveScreens    map[string]LiveScreenRequestState
}

// LiveScreenRequestState 是一个可见 terminal 的 one-shot latest-screen 拉取状态。
// ReceivedRevision > SubmittedRevision 表示唯一一份待选入渲染的最新屏；
// RequestInFlight 保证同一个 TerminalRef 同时最多只有一个网络请求。
type LiveScreenRequestState struct {
	EndpointID        EndpointID
	TerminalID        string
	Demand            bool
	RequestInFlight   bool
	NeedsBootstrap    bool
	Generation        uint64
	ReceivedRevision  uint64
	SubmittedRevision uint64
	Cols              int
	Rows              int
}

func (request LiveScreenRequestState) TerminalRef() TerminalRef {
	return NewTerminalRef(request.EndpointID, request.TerminalID)
}

// LiveResizeBoundary 是一次 content rect resize 后等待匹配 surface 的基线。
type LiveResizeBoundary struct {
	Active       bool
	PreviousCols int
	PreviousRows int
	Cols         int
	Rows         int
}

// LiveSurfaceRefreshState 只服务显式 preview/lifecycle snapshot 刷新，
// 连续 live screen 拉取由 LiveScreenRequestState 独立拥有。
type LiveSurfaceRefreshState struct {
	// InFlight 表示 live.screen.get 已发出但还未回到 reducer。
	InFlight bool
	// Dirty 表示 InFlight 期间又收到 invalidation，当前 fetch 完成后还要再取一次 latest。
	Dirty bool
	// Cols/Rows 保存下一次 live.screen.get 应使用的 owner view 尺寸。
	Cols int
	Rows int
}

// LiveCursor 是 live surface 的 content-local 光标状态。
type LiveCursor struct {
	Visible bool
	Row     int
	Col     int
	Shape   string
}

// LiveTerminalModes 只保存影响 TUI 输入转发的终端模式位。
type LiveTerminalModes struct {
	MouseTracking  bool
	MouseX10       bool
	MouseNormal    bool
	MouseButton    bool
	MouseAny       bool
	MouseSGR       bool
	BracketedPaste bool
}

func (modes LiveTerminalModes) MousePassthroughEnabled() bool {
	return modes.MouseTracking || modes.MouseX10 || modes.MouseNormal || modes.MouseButton || modes.MouseAny || modes.MouseSGR
}

// LiveCell 是 protocol snapshot cell 的 reducer-owned 副本，只服务实时 terminal 展示。
type LiveCell struct {
	Text          string
	Width         int
	FG            string
	BG            string
	Bold          bool
	Italic        bool
	Underline     bool
	Blink         bool
	Reverse       bool
	Strikethrough bool
	LinkURL       string
	LinkParams    string
}

// TerminalSessionStore 保存当前 attach/live path 的 reducer-owned 会话状态。
type TerminalSessionStore struct {
	EndpointID EndpointID
	TerminalID string
	Channel    uint16
	// InputChannels 只保存已 attach terminal 的输入 channel，用于 pane focus 后把输入发回对应 terminal。
	InputChannels map[string]uint16
	Attached      bool
	Cols          int
	Rows          int
	ResizePolicy  string
	SurfaceID     string
	ViewID        string
	// Desired* 只记录已发出的 terminal resize 目标，用于连续 pane command 去重和 stale response guard。
	DesiredCols        int
	DesiredRows        int
	ResizeRequestSeq   uint64
	ResizeConfirmedSeq uint64
	LastError          string
	State              TerminalLiveState
	ExitCode           int
	ExitReason         string
	ExitedAt           time.Time
	Command            []string
}

// LiveSurfaceSnapshot 是 terminal service/event 回投给 reducer 的实时投影。
type LiveSurfaceSnapshot struct {
	EndpointID   EndpointID
	TerminalID   string
	BaseRevision uint64
	Revision     uint64
	FullReplace  bool
	RowCopies    []LiveRowCopy
	ChangedRows  []int
	Cols         int
	Rows         int
	Lines        []string
	Screen       [][]LiveCell
	Title        string
	Cursor       LiveCursor
	Modes        LiveTerminalModes
	State        TerminalLiveState
	ExitCode     int
	ExitReason   string
	ExitedAt     time.Time
	Command      []string
	Err          string
}

type LiveRowCopy struct {
	SourceRow      int
	DestinationRow int
	Count          int
}

// TerminalRef 返回该 live snapshot 的 endpoint-aware terminal 身份。
// 空 EndpointID 只在旧本地调用边界允许出现，返回值会规范化到默认 local endpoint。
func (snapshot LiveSurfaceSnapshot) TerminalRef() TerminalRef {
	return NewTerminalRef(snapshot.EndpointID, snapshot.TerminalID)
}

// TerminalRef 返回当前 active live surface 的 endpoint-aware terminal 身份。
// 该身份只用于 TUI reducer 的实时投影路由，不代表 history truth 或 daemon 安全身份。
func (store TerminalSurfaceStore) TerminalRef() TerminalRef {
	return NewTerminalRef(store.EndpointID, store.TerminalID)
}

// TerminalRef 返回当前 attach/session 的 endpoint-aware terminal 身份。
// input channel、resize owner 和 lifecycle ack 都必须按该身份隔离。
func (store TerminalSessionStore) TerminalRef() TerminalRef {
	return NewTerminalRef(store.EndpointID, store.TerminalID)
}

func (store TerminalSurfaceStore) ApplySnapshot(snapshot LiveSurfaceSnapshot) TerminalSurfaceStore {
	return store.ApplySnapshotWithLifecycle(snapshot, false)
}

// ApplySnapshotWithLifecycle 只供 reducer 处理一次 core lifecycle 消息；lifecycleKnown 不写入 snapshot/store。
func (store TerminalSurfaceStore) ApplySnapshotWithLifecycle(snapshot LiveSurfaceSnapshot, lifecycleKnown bool) TerminalSurfaceStore {
	if snapshot.TerminalID == "" {
		snapshot.TerminalID = store.TerminalID
	}
	if snapshot.EndpointID == "" && snapshot.TerminalID == store.TerminalID {
		snapshot.EndpointID = store.EndpointID
	}
	snapshot.EndpointID = NormalizeEndpointID(snapshot.EndpointID)
	if snapshot.TerminalID == "" {
		return store
	}
	ref := snapshot.TerminalRef()
	if snapshot.State == "" {
		snapshot.State = TerminalLiveAttached
	}
	if store.resizeBoundaryRejects(snapshot) {
		// 中文说明：旧尺寸帧不允许回滚当前展示，但它已经是本次 live.screen.get
		// 的回包；必须释放 TUI 本地 in-flight，否则后续显式刷新会永久认为请求未完成。
		return store.FinishRefreshRef(ref)
	}
	current, hasCurrent := store.snapshotForTerminalRef(ref)
	if hasCurrent {
		if shouldRejectLiveSnapshotWithLifecycle(current, snapshot, lifecycleKnown) {
			// 中文说明：旧 revision / 空白旧帧同样只是不写入 surface truth，
			// 不能继续占用 refresh 背压状态。
			return store.FinishRefreshRef(ref)
		}
		if snapshot.Revision == 0 {
			snapshot.Revision = current.Revision
		}
	}
	if snapshot.BaseRevision != 0 && !snapshot.FullReplace {
		var merged bool
		snapshot, merged = mergeLiveSurfaceDelta(current, snapshot, hasCurrent)
		if !merged {
			return store.FinishRefreshRef(ref)
		}
	} else {
		snapshot.Lines = cloneStrings(snapshot.Lines)
		snapshot.Screen = cloneLiveCellRows(snapshot.Screen)
		snapshot.RowCopies = cloneLiveRowCopies(snapshot.RowCopies)
		snapshot.ChangedRows = cloneInts(snapshot.ChangedRows)
	}
	store.Surfaces = cloneLiveSurfaceSnapshots(store.Surfaces)
	store.Surfaces[liveSurfaceRefKey(ref)] = snapshot
	store = store.FinishRefreshRef(ref)
	if store.TerminalID == "" || store.TerminalRef().Equal(ref) {
		store = store.projectSnapshot(snapshot, true)
		if store.ResizeBoundary.Active && snapshot.Cols == store.ResizeBoundary.Cols && snapshot.Rows == store.ResizeBoundary.Rows {
			store.ResizeBoundary = LiveResizeBoundary{}
		}
		return store
	}
	return store
}

func (store TerminalSurfaceStore) Attach(terminalID string, cols int, rows int) TerminalSurfaceStore {
	return store.AttachRef(LocalTerminalRef(terminalID), cols, rows)
}

// AttachRef 把当前 live surface 投影绑定到指定 TerminalRef。
// Surfaces map 的 key 使用 endpoint-aware 身份，避免不同 daemon 下同名 TerminalID 互相覆盖。
func (store TerminalSurfaceStore) AttachRef(ref TerminalRef, cols int, rows int) TerminalSurfaceStore {
	ref = ref.Normalize()
	store = store.clearRefreshRef(ref)
	if store.TerminalID != "" && !store.TerminalRef().Equal(ref) {
		store.Lines = nil
		store.Screen = nil
		store.Cursor = LiveCursor{}
		store.Modes = LiveTerminalModes{}
		store.Ready = false
	}
	store.EndpointID = ref.EndpointID
	store.TerminalID = ref.TerminalID
	store.Revision = 0
	store.Cols = cols
	store.Rows = rows
	store.State = TerminalLiveAttached
	store.ExitCode = 0
	store.ExitReason = ""
	store.ExitedAt = time.Time{}
	store.Command = nil
	store.Err = ""
	store.ResizeBoundary = LiveResizeBoundary{}
	if !ref.Empty() {
		store.Surfaces = cloneLiveSurfaceSnapshots(store.Surfaces)
		snapshot, ok := store.snapshotForTerminalRef(ref)
		if !ok {
			snapshot = LiveSurfaceSnapshot{}
		}
		wasBoundary := liveSnapshotIsBoundary(snapshot)
		snapshot.EndpointID = ref.EndpointID
		snapshot.TerminalID = ref.TerminalID
		if wasBoundary {
			snapshot.Revision = 0
		}
		if snapshot.Cols == 0 {
			snapshot.Cols = cols
		}
		if snapshot.Rows == 0 {
			snapshot.Rows = rows
		}
		snapshot.State = TerminalLiveAttached
		snapshot.ExitCode = 0
		snapshot.ExitReason = ""
		snapshot.ExitedAt = time.Time{}
		snapshot.Command = nil
		snapshot.Err = ""
		store.Surfaces[liveSurfaceRefKey(ref)] = snapshot
		if len(snapshot.Lines) > 0 || len(snapshot.Screen) > 0 || snapshot.Title != "" || snapshot.Cursor.Visible {
			store = store.projectSnapshot(snapshot, true)
		}
	}
	return store
}

// CacheAttachRef 只把指定 TerminalRef 的 attach metadata 写入 live surface 缓存。
// 该方法服务后台 pane/floating restore：view binding 已经拿到 channel，但当前前台
// surface/session truth 仍属于 active view，不能因为后台 attach 回包而切换全局投影。
func (store TerminalSurfaceStore) CacheAttachRef(ref TerminalRef, cols int, rows int) TerminalSurfaceStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store
	}
	store = store.clearRefreshRef(ref)
	snapshot, ok := store.Surfaces[liveSurfaceRefKey(ref)]
	if !ok {
		snapshot = LiveSurfaceSnapshot{}
	}
	snapshot.EndpointID = ref.EndpointID
	snapshot.TerminalID = ref.TerminalID
	if cols > 0 {
		snapshot.Cols = cols
	}
	if rows > 0 {
		snapshot.Rows = rows
	}
	snapshot.State = TerminalLiveAttached
	snapshot.ExitCode = 0
	snapshot.ExitReason = ""
	snapshot.ExitedAt = time.Time{}
	snapshot.Command = nil
	snapshot.Err = ""
	store.Surfaces = cloneLiveSurfaceSnapshots(store.Surfaces)
	store.Surfaces[liveSurfaceRefKey(ref)] = snapshot
	return store
}

func (store TerminalSurfaceStore) RestartPreservingContent(terminalID string, cols int, rows int) TerminalSurfaceStore {
	return store.RestartPreservingContentRef(LocalTerminalRef(terminalID), cols, rows)
}

// RestartPreservingContentRef 标记指定 TerminalRef 已重启，同时保留该 terminal 既有 live 内容。
// 该操作只清理 lifecycle 元数据，不创建 history，也不影响其他 endpoint 下同名 TerminalID。
func (store TerminalSurfaceStore) RestartPreservingContentRef(ref TerminalRef, cols int, rows int) TerminalSurfaceStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store
	}
	store = store.clearRefreshRef(ref)
	snapshot, ok := store.snapshotForTerminalRef(ref)
	if !ok {
		return store.AttachRef(ref, cols, rows)
	}
	// 中文说明：restart 只重启 terminal process，不能把同 terminal 的 live tail 清空。
	// lifecycle 元数据清掉后，真实 channel/surface 仍等待 per-view reattach 回投。
	snapshot.EndpointID = ref.EndpointID
	snapshot.TerminalID = ref.TerminalID
	if snapshot.Cols == 0 {
		snapshot.Cols = cols
	}
	if snapshot.Rows == 0 {
		snapshot.Rows = rows
	}
	snapshot.State = TerminalLiveAttached
	snapshot.ExitCode = 0
	snapshot.ExitReason = ""
	snapshot.ExitedAt = time.Time{}
	snapshot.Command = nil
	snapshot.Err = ""
	snapshot.Cursor = LiveCursor{}
	snapshot.Modes = LiveTerminalModes{}
	store.Surfaces = cloneLiveSurfaceSnapshots(store.Surfaces)
	store.Surfaces[liveSurfaceRefKey(ref)] = snapshot
	if store.TerminalID != "" && !store.TerminalRef().Equal(ref) {
		store.ResizeBoundary = LiveResizeBoundary{}
		return store
	}
	store = store.projectSnapshot(snapshot, liveSnapshotHasContent(snapshot))
	store.ResizeBoundary = LiveResizeBoundary{}
	return store
}

func (store TerminalSurfaceStore) MarkAttached(terminalID string) TerminalSurfaceStore {
	return store.MarkAttachedRef(LocalTerminalRef(terminalID))
}

// MarkAttachedRef 只把指定 TerminalRef 的 live 投影标记为 attached。
// terminal pool/list 不能调用它伪造生命周期；只有 live surface/event 明确返回 running 才可进入该路径。
func (store TerminalSurfaceStore) MarkAttachedRef(ref TerminalRef) TerminalSurfaceStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store
	}
	store = store.clearRefreshRef(ref)
	snapshot, ok := store.snapshotForTerminalRef(ref)
	if ok {
		// 中文说明：只有 core live surface/event 明确返回 running 时才调用这里；
		// terminal pool/list 不能用它把 running 写进 live 投影缓存。
		snapshot.EndpointID = ref.EndpointID
		snapshot.TerminalID = ref.TerminalID
		snapshot.State = TerminalLiveAttached
		snapshot.ExitCode = 0
		snapshot.ExitReason = ""
		snapshot.ExitedAt = time.Time{}
		snapshot.Command = nil
		snapshot.Err = ""
		store.Surfaces = cloneLiveSurfaceSnapshots(store.Surfaces)
		store.Surfaces[liveSurfaceRefKey(ref)] = snapshot
	}
	if !store.TerminalRef().Equal(ref) {
		return store
	}
	if ok {
		store = store.projectSnapshot(snapshot, store.Ready || liveSnapshotHasContent(snapshot))
	} else {
		store.State = TerminalLiveAttached
		store.ExitCode = 0
		store.ExitReason = ""
		store.ExitedAt = time.Time{}
		store.Command = nil
		store.Err = ""
	}
	store.ResizeBoundary = LiveResizeBoundary{}
	return store
}

func (store TerminalSurfaceStore) MarkExited(terminalID string, exitCode int, reason string) TerminalSurfaceStore {
	return store.MarkExitedWithMetadataRef(LocalTerminalRef(terminalID), exitCode, reason, time.Time{}, nil)
}

func (store TerminalSurfaceStore) MarkExitedWithMetadata(terminalID string, exitCode int, reason string, exitedAt time.Time, command []string) TerminalSurfaceStore {
	return store.MarkExitedWithMetadataRef(LocalTerminalRef(terminalID), exitCode, reason, exitedAt, command)
}

// MarkExitedWithMetadataRef 写入指定 TerminalRef 的 exited lifecycle 边界。
// endpoint 是 lifecycle 投影的隔离边界；远端退出不能把本地同名 terminal 标成 exited。
func (store TerminalSurfaceStore) MarkExitedWithMetadataRef(ref TerminalRef, exitCode int, reason string, exitedAt time.Time, command []string) TerminalSurfaceStore {
	ref = ref.Normalize()
	store = store.clearRefreshRef(ref)
	if !ref.Empty() {
		store.Surfaces = cloneLiveSurfaceSnapshots(store.Surfaces)
		snapshot, ok := store.snapshotForTerminalRef(ref)
		if !ok {
			snapshot = LiveSurfaceSnapshot{}
		}
		snapshot.EndpointID = ref.EndpointID
		snapshot.TerminalID = ref.TerminalID
		snapshot.State = TerminalLiveExited
		snapshot.ExitCode = exitCode
		snapshot.ExitReason = reason
		snapshot.ExitedAt = exitedAt
		snapshot.Command = cloneStrings(command)
		snapshot.Err = ""
		snapshot.Cursor = LiveCursor{}
		snapshot.Modes = LiveTerminalModes{}
		store.Surfaces[liveSurfaceRefKey(ref)] = snapshot
	}
	if ref.Empty() || store.TerminalID == "" || store.TerminalRef().Equal(ref) {
		if !ref.Empty() {
			store.EndpointID = ref.EndpointID
			store.TerminalID = ref.TerminalID
		}
		store.State = TerminalLiveExited
		store.ExitCode = exitCode
		store.ExitReason = reason
		store.ExitedAt = exitedAt
		store.Command = cloneStrings(command)
		store.Err = ""
		store.ResizeBoundary = LiveResizeBoundary{}
		store.Cursor = LiveCursor{}
		store.Modes = LiveTerminalModes{}
	}
	return store
}

func (store TerminalSurfaceStore) SetError(err string) TerminalSurfaceStore {
	ref := store.TerminalRef()
	if ref.Empty() {
		store.Err = err
		store.State = TerminalLiveError
		store.ResizeBoundary = LiveResizeBoundary{}
		return store
	}
	return store.SetErrorRef(ref, err)
}

// SetErrorRef 只把错误生命周期写入指定 TerminalRef 的 live surface。
// 非当前 active ref 的错误只进入 Surfaces 缓存，不能污染前台 terminal 的实时投影。
func (store TerminalSurfaceStore) SetErrorRef(ref TerminalRef, err string) TerminalSurfaceStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store.SetError(err)
	}
	store = store.clearRefreshRef(ref)
	store.Surfaces = cloneLiveSurfaceSnapshots(store.Surfaces)
	snapshot, ok := store.snapshotForTerminalRef(ref)
	if !ok && store.TerminalRef().Equal(ref) {
		snapshot = store.Snapshot()
	}
	snapshot.EndpointID = ref.EndpointID
	snapshot.TerminalID = ref.TerminalID
	snapshot.Err = err
	snapshot.State = TerminalLiveError
	store.Surfaces[liveSurfaceRefKey(ref)] = snapshot
	if store.TerminalID == "" || store.TerminalRef().Equal(ref) {
		store.EndpointID = ref.EndpointID
		store.TerminalID = ref.TerminalID
		store.Err = err
		store.State = TerminalLiveError
		store.ResizeBoundary = LiveResizeBoundary{}
	}
	return store
}

func (store TerminalSurfaceStore) Resize(cols int, rows int) TerminalSurfaceStore {
	ref := store.TerminalRef()
	if !ref.Empty() {
		return store.ResizeRef(ref, cols, rows)
	}
	return store.resizeProjection(cols, rows)
}

// ResizeRef updates one terminal's cached geometry without switching the active surface projection.
func (store TerminalSurfaceStore) ResizeRef(ref TerminalRef, cols int, rows int) TerminalSurfaceStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store.Resize(cols, rows)
	}
	store.Surfaces = cloneLiveSurfaceSnapshots(store.Surfaces)
	snapshot, ok := store.snapshotForTerminalRef(ref)
	if !ok && store.TerminalRef().Equal(ref) {
		snapshot = store.Snapshot()
	}
	snapshot.EndpointID = ref.EndpointID
	snapshot.TerminalID = ref.TerminalID
	snapshot.Cols = cols
	snapshot.Rows = rows
	if snapshot.State == "" {
		snapshot.State = TerminalLiveAttached
	}
	store.Surfaces[liveSurfaceRefKey(ref)] = snapshot
	if !store.TerminalRef().Equal(ref) {
		return store
	}
	return store.resizeProjection(cols, rows)
}

func (store TerminalSurfaceStore) resizeProjection(cols int, rows int) TerminalSurfaceStore {
	previousCols := store.Cols
	previousRows := store.Rows
	store.Cols = cols
	store.Rows = rows
	if previousCols != cols || previousRows != rows {
		store.ResizeBoundary = LiveResizeBoundary{Active: true, PreviousCols: previousCols, PreviousRows: previousRows, Cols: cols, Rows: rows}
	}
	return store
}

func (store TerminalSurfaceStore) SurfaceForTerminal(terminalID string) TerminalSurfaceStore {
	return store.SurfaceForTerminalRef(LocalTerminalRef(terminalID))
}

// SurfaceForTerminalRef 返回指定 TerminalRef 的 live surface 投影。
// 未命中时只返回 pending 占位，不会创建或 fallback 到其他 endpoint 的同名 terminal。
func (store TerminalSurfaceStore) SurfaceForTerminalRef(ref TerminalRef) TerminalSurfaceStore {
	ref = ref.Normalize()
	if ref.Empty() || store.TerminalRef().Equal(ref) {
		return store
	}
	snapshot, ok := store.snapshotForTerminalRef(ref)
	if !ok {
		return TerminalSurfaceStore{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, State: TerminalLivePending}
	}
	return (TerminalSurfaceStore{}).projectSnapshot(snapshot, snapshot.State != TerminalLivePending)
}

func (store TerminalSurfaceStore) Snapshot() LiveSurfaceSnapshot {
	return LiveSurfaceSnapshot{
		EndpointID: store.EndpointID,
		TerminalID: store.TerminalID,
		Revision:   store.Revision,
		Cols:       store.Cols,
		Rows:       store.Rows,
		Lines:      cloneStrings(store.Lines),
		Screen:     cloneLiveCellRows(store.Screen),
		Title:      store.Title,
		Cursor:     store.Cursor,
		Modes:      store.Modes,
		State:      store.State,
		ExitCode:   store.ExitCode,
		ExitReason: store.ExitReason,
		ExitedAt:   store.ExitedAt,
		Command:    cloneStrings(store.Command),
		Err:        store.Err,
	}
}

func (store TerminalSurfaceStore) RemoveTerminal(terminalID string) TerminalSurfaceStore {
	return store.RemoveTerminalRef(LocalTerminalRef(terminalID))
}

// RemoveTerminalRef 删除指定 TerminalRef 的 live surface 和 refresh 状态。
// 删除范围只覆盖该 endpoint + terminal，不能影响其他 endpoint 的同名 surface/session。
func (store TerminalSurfaceStore) RemoveTerminalRef(ref TerminalRef) TerminalSurfaceStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store
	}
	key := liveSurfaceRefKey(ref)
	if len(store.Surfaces) > 0 {
		store.Surfaces = cloneLiveSurfaceSnapshots(store.Surfaces)
		delete(store.Surfaces, key)
	}
	if len(store.Refreshes) > 0 {
		store.Refreshes = cloneLiveSurfaceRefreshStates(store.Refreshes)
		delete(store.Refreshes, key)
	}
	if len(store.LiveScreens) > 0 {
		store.LiveScreens = cloneLiveScreenRequestStates(store.LiveScreens)
		delete(store.LiveScreens, key)
	}
	if !store.TerminalRef().Equal(ref) {
		return store
	}
	store.EndpointID = ""
	store.TerminalID = ""
	store.Revision = 0
	store.Cols = 0
	store.Rows = 0
	store.Lines = nil
	store.Screen = nil
	store.Title = ""
	store.Cursor = LiveCursor{}
	store.Modes = LiveTerminalModes{}
	store.Ready = false
	store.State = ""
	store.ExitCode = 0
	store.ExitReason = ""
	store.ExitedAt = time.Time{}
	store.Command = nil
	store.Err = ""
	store.ResizeBoundary = LiveResizeBoundary{}
	return store
}

// ReconcileLiveScreenDemand 以当前完整帧实际包含的 live terminal 为准更新拉取所有权。
// 返回值只包含从可见变为不可见且需要取消在途请求的 TerminalRef。
func (store TerminalSurfaceStore) ReconcileLiveScreenDemand(refs []TerminalRef) (TerminalSurfaceStore, []TerminalRef) {
	desired := make(map[string]TerminalRef, len(refs))
	for _, ref := range refs {
		ref = ref.Normalize()
		if !ref.Empty() {
			desired[liveSurfaceRefKey(ref)] = ref
		}
	}
	requests := cloneLiveScreenRequestStates(store.LiveScreens)
	var canceled []TerminalRef
	for key, request := range requests {
		if _, ok := desired[key]; ok || !request.Demand {
			continue
		}
		if request.RequestInFlight {
			canceled = append(canceled, request.TerminalRef())
		}
		request.Demand = false
		request.RequestInFlight = false
		request.Generation++
		requests[key] = request
	}
	for key, ref := range desired {
		request := requests[key]
		if request.Demand {
			continue
		}
		request.EndpointID = ref.EndpointID
		request.TerminalID = ref.TerminalID
		request.Demand = true
		request.RequestInFlight = false
		request.Generation++
		requests[key] = request
	}
	store.LiveScreens = requests
	return store, canceled
}

// SubmitLiveScreenRef 标记 canonical latest screen 已被选入 renderer submission。
// 该动作发生在物理写出开始前，因此下一次网络等待可以和当前写出重叠。
func (store TerminalSurfaceStore) SubmitLiveScreenRef(ref TerminalRef, revision uint64, cols int, rows int) TerminalSurfaceStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store
	}
	key := liveSurfaceRefKey(ref)
	request, ok := store.LiveScreens[key]
	if !ok || !request.Demand {
		return store
	}
	if revision > request.ReceivedRevision {
		request.ReceivedRevision = revision
	}
	if revision > request.SubmittedRevision {
		request.SubmittedRevision = revision
	}
	if cols > 0 {
		request.Cols = cols
	}
	if rows > 0 {
		request.Rows = rows
	}
	store.LiveScreens = cloneLiveScreenRequestStates(store.LiveScreens)
	store.LiveScreens[key] = request
	return store
}

// BeginLiveScreenRequestRef 占用该 TerminalRef 唯一的 one-shot request 槽位。
func (store TerminalSurfaceStore) BeginLiveScreenRequestRef(ref TerminalRef) (TerminalSurfaceStore, LiveScreenRequestState, bool) {
	ref = ref.Normalize()
	if ref.Empty() {
		return store, LiveScreenRequestState{}, false
	}
	key := liveSurfaceRefKey(ref)
	request, ok := store.LiveScreens[key]
	if !ok || !request.Demand || request.RequestInFlight || (!request.NeedsBootstrap && request.ReceivedRevision > request.SubmittedRevision) {
		return store, request, false
	}
	request.RequestInFlight = true
	store.LiveScreens = cloneLiveScreenRequestStates(store.LiveScreens)
	store.LiveScreens[key] = request
	return store, request, true
}

// RequireLiveScreenBootstrap releases the matching request and forces its next
// observed revision to zero while keeping the last valid screen visible.
func (store TerminalSurfaceStore) RequireLiveScreenBootstrap(ref TerminalRef, generation uint64) (TerminalSurfaceStore, bool) {
	ref = ref.Normalize()
	if ref.Empty() {
		return store, false
	}
	key := liveSurfaceRefKey(ref)
	request, ok := store.LiveScreens[key]
	if !ok || !request.Demand || !request.RequestInFlight || request.Generation != generation {
		return store, false
	}
	request.RequestInFlight = false
	request.NeedsBootstrap = true
	store.LiveScreens = cloneLiveScreenRequestStates(store.LiveScreens)
	store.LiveScreens[key] = request
	return store, true
}

func (store TerminalSurfaceStore) LiveScreenRequestMatches(ref TerminalRef, generation uint64) bool {
	ref = ref.Normalize()
	request, ok := store.LiveScreens[liveSurfaceRefKey(ref)]
	return ok && request.Demand && request.RequestInFlight && request.Generation == generation
}

// FinishLiveScreenRequestRef 释放 one-shot request，并记录 canonical cache 实际接收的 revision。
// generation 不匹配表示该 view 已隐藏、detach 或重新获得所有权，晚到结果直接丢弃。
func (store TerminalSurfaceStore) FinishLiveScreenRequestRef(ref TerminalRef, generation uint64, receivedRevision uint64) (TerminalSurfaceStore, bool) {
	ref = ref.Normalize()
	if ref.Empty() {
		return store, false
	}
	key := liveSurfaceRefKey(ref)
	request, ok := store.LiveScreens[key]
	if !ok || !request.Demand || !request.RequestInFlight || request.Generation != generation {
		return store, false
	}
	request.RequestInFlight = false
	request.NeedsBootstrap = false
	if receivedRevision > request.ReceivedRevision {
		request.ReceivedRevision = receivedRevision
	}
	store.LiveScreens = cloneLiveScreenRequestStates(store.LiveScreens)
	store.LiveScreens[key] = request
	return store, true
}

func (store TerminalSurfaceStore) LiveScreenRequestRef(ref TerminalRef) (LiveScreenRequestState, bool) {
	ref = ref.Normalize()
	request, ok := store.LiveScreens[liveSurfaceRefKey(ref)]
	return request, ok
}

func (store TerminalSurfaceStore) RequestRefresh(terminalID string, cols int, rows int) (TerminalSurfaceStore, bool) {
	return store.RequestRefreshRef(LocalTerminalRef(terminalID), cols, rows)
}

// RequestRefreshRef 申请拉取指定 TerminalRef 的 latest live surface。
// InFlight/Dirty 是 TUI 本地背压状态，key 必须包含 endpoint，避免跨 daemon 合并刷新。
func (store TerminalSurfaceStore) RequestRefreshRef(ref TerminalRef, cols int, rows int) (TerminalSurfaceStore, bool) {
	ref = ref.Normalize()
	if ref.Empty() {
		return store, false
	}
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	key := liveSurfaceRefKey(ref)
	refresh := store.Refreshes[key]
	refresh.Cols = cols
	refresh.Rows = rows
	if refresh.InFlight {
		// 中文说明：普通 live screen next 只是“当前屏已失效”的合并信号。
		// fetch 飞行期间不能排 N 个 revision，只记录 dirty，下一次仍取 core latest screen。
		refresh.Dirty = true
		store.Refreshes = cloneLiveSurfaceRefreshStates(store.Refreshes)
		store.Refreshes[key] = refresh
		return store, false
	}
	refresh.InFlight = true
	refresh.Dirty = false
	store.Refreshes = cloneLiveSurfaceRefreshStates(store.Refreshes)
	store.Refreshes[key] = refresh
	return store, true
}

func (store TerminalSurfaceStore) FinishRefresh(terminalID string) TerminalSurfaceStore {
	return store.FinishRefreshRef(LocalTerminalRef(terminalID))
}

// FinishRefreshRef 结束指定 TerminalRef 的 latest fetch 状态。
// Dirty follow-up 只在同 endpoint terminal 内保留，不能让其他 endpoint 继承刷新债务。
func (store TerminalSurfaceStore) FinishRefreshRef(ref TerminalRef) TerminalSurfaceStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store
	}
	key := liveSurfaceRefKey(ref)
	refresh, ok := store.Refreshes[key]
	if !ok {
		return store
	}
	store.Refreshes = cloneLiveSurfaceRefreshStates(store.Refreshes)
	if refresh.Dirty {
		// 中文说明：fetch 返回期间又有 invalidation，当前返回值已不是
		// core latest。调用方可选择跳过这张中间屏；此处只释放 in-flight，
		// 让后续 maybeScheduleDirtyLiveSurfaceRefresh 立即再拉一次当前最新屏。
		refresh.InFlight = false
		refresh.Dirty = false
		store.Refreshes[key] = refresh
		return store
	}
	delete(store.Refreshes, key)
	return store
}

// RefreshStateRef 返回指定 TerminalRef 的 live refresh 背压状态。
// 调用方只用于 diagnostics 或 harness 观察 TUI 本地消息链路；该状态不是 core history truth，
// 也不能作为渲染内容来源。
func (store TerminalSurfaceStore) RefreshStateRef(ref TerminalRef) (LiveSurfaceRefreshState, bool) {
	ref = ref.Normalize()
	if ref.Empty() {
		return LiveSurfaceRefreshState{}, false
	}
	refresh, ok := store.Refreshes[liveSurfaceRefKey(ref)]
	return refresh, ok
}

func (store TerminalSurfaceStore) clearRefreshRef(ref TerminalRef) TerminalSurfaceStore {
	ref = ref.Normalize()
	if ref.Empty() || len(store.Refreshes) == 0 {
		return store
	}
	key := liveSurfaceRefKey(ref)
	if _, ok := store.Refreshes[key]; !ok {
		return store
	}
	// 中文说明：exit/error/attach/restart 是 terminal lifecycle 边界，旧 refresh debt
	// 不能穿过边界继续阻塞新的显式刷新；普通 dirty follow-up 仍由 FinishRefreshRef 管。
	store.Refreshes = cloneLiveSurfaceRefreshStates(store.Refreshes)
	delete(store.Refreshes, key)
	return store
}

func (store TerminalSurfaceStore) ConsumeDirtyRefresh(terminalID string) (TerminalSurfaceStore, int, int, bool) {
	return store.ConsumeDirtyRefreshRef(LocalTerminalRef(terminalID))
}

// ConsumeDirtyRefreshRef 取出指定 TerminalRef 的 dirty follow-up 刷新。
// 返回的尺寸来自同 ref 最近一次 invalidation，不能跨 endpoint 复用。
func (store TerminalSurfaceStore) ConsumeDirtyRefreshRef(ref TerminalRef) (TerminalSurfaceStore, int, int, bool) {
	ref = ref.Normalize()
	if ref.Empty() {
		return store, 0, 0, false
	}
	key := liveSurfaceRefKey(ref)
	refresh, ok := store.Refreshes[key]
	if !ok || refresh.InFlight || refresh.Dirty {
		return store, 0, 0, false
	}
	refresh.InFlight = true
	store.Refreshes = cloneLiveSurfaceRefreshStates(store.Refreshes)
	store.Refreshes[key] = refresh
	return store, refresh.Cols, refresh.Rows, true
}

func (store TerminalSessionStore) RemoveTerminal(terminalID string) TerminalSessionStore {
	return store.RemoveTerminalRef(LocalTerminalRef(terminalID))
}

// RemoveTerminalRef 删除指定 TerminalRef 的 attach/session 投影。
// InputChannels 是 endpoint-scoped map；移除远端 terminal 不会清掉 local 同名 input channel。
func (store TerminalSessionStore) RemoveTerminalRef(ref TerminalRef) TerminalSessionStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store
	}
	key := terminalSessionInputKey(ref)
	if len(store.InputChannels) > 0 {
		store.InputChannels = cloneInputChannels(store.InputChannels)
		delete(store.InputChannels, key)
	}
	if !store.TerminalRef().Equal(ref) {
		return store
	}
	store.EndpointID = ""
	store.TerminalID = ""
	store.Channel = 0
	store.Attached = false
	store.ResizePolicy = ""
	store.SurfaceID = ""
	store.ViewID = ""
	store.DesiredCols = 0
	store.DesiredRows = 0
	store.ResizeRequestSeq = 0
	store.ResizeConfirmedSeq = 0
	store.LastError = ""
	store.State = ""
	store.ExitCode = 0
	store.ExitReason = ""
	store.ExitedAt = time.Time{}
	store.Command = nil
	return store
}

func (store TerminalSessionStore) ClearInputChannel(terminalID string) TerminalSessionStore {
	return store.ClearInputChannelRef(LocalTerminalRef(terminalID))
}

// ClearInputChannelRef 清除指定 TerminalRef 的输入 channel。
// restart/reconnect 期间只能清掉同 endpoint terminal 的 channel，避免输入被误路由到别的 daemon。
func (store TerminalSessionStore) ClearInputChannelRef(ref TerminalRef) TerminalSessionStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store
	}
	key := terminalSessionInputKey(ref)
	if len(store.InputChannels) > 0 {
		store.InputChannels = cloneInputChannels(store.InputChannels)
		delete(store.InputChannels, key)
	}
	if store.TerminalRef().Equal(ref) {
		store.Channel = 0
		store.Attached = false
		store.State = TerminalLivePending
		store.ExitCode = 0
		store.ExitReason = ""
		store.ExitedAt = time.Time{}
		store.Command = nil
		store.LastError = ""
	}
	return store
}

func (store TerminalSessionStore) MarkAttached(terminalID string) TerminalSessionStore {
	return store.MarkAttachedRef(LocalTerminalRef(terminalID))
}

// MarkAttachedRef 根据 live lifecycle ack 恢复指定 TerminalRef 的 session attached 状态。
// 非当前 session 的 ack 不能改写 active session，以免远端同名 terminal 影响本地输入链路。
func (store TerminalSessionStore) MarkAttachedRef(ref TerminalRef) TerminalSessionStore {
	ref = ref.Normalize()
	if ref.Empty() || !store.TerminalRef().Equal(ref) {
		return store
	}
	if store.State == TerminalLiveError && !store.Attached {
		return store
	}
	store.ExitCode = 0
	store.ExitReason = ""
	store.ExitedAt = time.Time{}
	store.Command = nil
	store.LastError = ""
	if store.Attached {
		store.State = TerminalLiveAttached
		return store
	}
	store.State = TerminalLivePending
	return store
}

func (store TerminalSurfaceStore) projectSnapshot(snapshot LiveSurfaceSnapshot, ready bool) TerminalSurfaceStore {
	snapshot.EndpointID = NormalizeEndpointID(snapshot.EndpointID)
	store.EndpointID = snapshot.EndpointID
	store.TerminalID = snapshot.TerminalID
	store.Revision = snapshot.Revision
	store.Cols = snapshot.Cols
	store.Rows = snapshot.Rows
	// 中文说明：写入 store 前已经克隆成 reducer-owned payload；active projection
	// 和 Surfaces map 共享同一份不可变帧，避免每个 live frame 再深拷贝一次。
	store.Lines = snapshot.Lines
	store.Screen = snapshot.Screen
	store.Title = snapshot.Title
	store.Cursor = snapshot.Cursor
	store.Modes = snapshot.Modes
	store.Ready = ready
	store.State = snapshot.State
	if store.State == "" {
		store.State = TerminalLiveAttached
	}
	store.ExitCode = snapshot.ExitCode
	store.ExitReason = snapshot.ExitReason
	store.ExitedAt = snapshot.ExitedAt
	store.Command = cloneStrings(snapshot.Command)
	store.Err = snapshot.Err
	return store
}

func mergeLiveSurfaceDelta(current LiveSurfaceSnapshot, incoming LiveSurfaceSnapshot, hasCurrent bool) (LiveSurfaceSnapshot, bool) {
	if !hasCurrent || current.Revision != incoming.BaseRevision || current.Cols != incoming.Cols || current.Rows != incoming.Rows || len(current.Screen) < incoming.Rows || len(incoming.ChangedRows) != len(incoming.Screen) {
		return LiveSurfaceSnapshot{}, false
	}
	merged := current
	merged.Revision = incoming.Revision
	merged.BaseRevision = incoming.BaseRevision
	merged.Cols = incoming.Cols
	merged.Rows = incoming.Rows
	merged.Cursor = incoming.Cursor
	merged.Modes = incoming.Modes
	merged.FullReplace = false
	merged.Lines = current.Lines
	merged.Screen = append([][]LiveCell(nil), current.Screen...)
	if len(merged.Screen) < incoming.Rows {
		merged.Screen = append(merged.Screen, make([][]LiveCell, incoming.Rows-len(merged.Screen))...)
	} else if len(merged.Screen) > incoming.Rows {
		merged.Screen = merged.Screen[:incoming.Rows]
	}
	merged.RowCopies = cloneLiveRowCopies(incoming.RowCopies)
	for _, rowCopy := range incoming.RowCopies {
		if rowCopy.Count <= 0 || rowCopy.SourceRow < 0 || rowCopy.DestinationRow < 0 || rowCopy.SourceRow+rowCopy.Count > incoming.Rows || rowCopy.DestinationRow+rowCopy.Count > incoming.Rows {
			return LiveSurfaceSnapshot{}, false
		}
		for offset := 0; offset < rowCopy.Count; offset++ {
			merged.Screen[rowCopy.DestinationRow+offset] = current.Screen[rowCopy.SourceRow+offset]
		}
	}
	merged.ChangedRows = cloneInts(incoming.ChangedRows)
	for index, rowIndex := range incoming.ChangedRows {
		if rowIndex < 0 || rowIndex >= incoming.Rows {
			return LiveSurfaceSnapshot{}, false
		}
		merged.Screen[rowIndex] = cloneLiveCells(incoming.Screen[index])
	}
	return merged, true
}

func (store TerminalSurfaceStore) resizeBoundaryRejects(snapshot LiveSurfaceSnapshot) bool {
	if !store.ResizeBoundary.Active || store.TerminalID == "" || !snapshot.TerminalRef().Equal(store.TerminalRef()) || !liveSnapshotIsOrdinary(snapshot) {
		return false
	}
	// resize 后只拒绝明确来自旧尺寸基线的普通帧，避免晚到旧帧回滚投影。
	return snapshot.Cols == store.ResizeBoundary.PreviousCols && snapshot.Rows == store.ResizeBoundary.PreviousRows
}

func (store TerminalSurfaceStore) snapshotForTerminalRef(ref TerminalRef) (LiveSurfaceSnapshot, bool) {
	ref = ref.Normalize()
	if ref.Empty() {
		return LiveSurfaceSnapshot{}, false
	}
	if snapshot, ok := store.Surfaces[liveSurfaceRefKey(ref)]; ok {
		return snapshot, true
	}
	if !store.TerminalRef().Equal(ref) {
		return LiveSurfaceSnapshot{}, false
	}
	return LiveSurfaceSnapshot{
		EndpointID: ref.EndpointID,
		TerminalID: ref.TerminalID,
		Revision:   store.Revision,
		Cols:       store.Cols,
		Rows:       store.Rows,
		Lines:      cloneStrings(store.Lines),
		Screen:     cloneLiveCellRows(store.Screen),
		Title:      store.Title,
		Cursor:     store.Cursor,
		Modes:      store.Modes,
		State:      store.State,
		ExitCode:   store.ExitCode,
		ExitReason: store.ExitReason,
		ExitedAt:   store.ExitedAt,
		Command:    cloneStrings(store.Command),
		Err:        store.Err,
	}, true
}

func shouldRejectLiveSnapshotWithLifecycle(current LiveSurfaceSnapshot, incoming LiveSurfaceSnapshot, lifecycleKnown bool) bool {
	if incoming.Revision != 0 && current.Revision != 0 && incoming.Revision < current.Revision && !lifecycleKnown {
		return true
	}
	if incoming.Revision == 0 && liveSnapshotHasContent(current) && !liveSnapshotHasContent(incoming) && !lifecycleKnown {
		return true
	}
	if liveSnapshotIsBoundary(current) && liveSnapshotIsOrdinary(incoming) && !lifecycleKnown {
		return true
	}
	return false
}

func liveSnapshotHasContent(snapshot LiveSurfaceSnapshot) bool {
	return len(snapshot.Lines) > 0 || len(snapshot.Screen) > 0 || snapshot.Title != "" || snapshot.Cursor.Visible
}

func liveSnapshotIsBoundary(snapshot LiveSurfaceSnapshot) bool {
	return snapshot.State == TerminalLiveExited || snapshot.State == TerminalLiveError || snapshot.Err != ""
}

func liveSnapshotIsOrdinary(snapshot LiveSurfaceSnapshot) bool {
	if snapshot.Err != "" || snapshot.ExitCode != 0 || snapshot.ExitReason != "" {
		return false
	}
	return snapshot.State == "" || snapshot.State == TerminalLiveAttached
}

func cloneLiveCellRows(rows [][]LiveCell) [][]LiveCell {
	if len(rows) == 0 {
		return nil
	}
	out := make([][]LiveCell, len(rows))
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		out[i] = cloneLiveCells(row)
	}
	return out
}

func cloneLiveScreenRequestStates(values map[string]LiveScreenRequestState) map[string]LiveScreenRequestState {
	if len(values) == 0 {
		return make(map[string]LiveScreenRequestState)
	}
	out := make(map[string]LiveScreenRequestState, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func cloneLiveCells(cells []LiveCell) []LiveCell {
	if len(cells) == 0 {
		return nil
	}
	return append([]LiveCell(nil), cells...)
}

func cloneInts(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	return append([]int(nil), values...)
}

func (store TerminalSessionStore) Attach(terminalID string, channel uint16, cols int, rows int) TerminalSessionStore {
	return store.AttachRefWithResizeOwner(LocalTerminalRef(terminalID), channel, cols, rows, "", "", "")
}

func (store TerminalSessionStore) AttachWithResizeOwner(terminalID string, channel uint16, cols int, rows int, resizePolicy string, surfaceID string, viewID string) TerminalSessionStore {
	return store.AttachRefWithResizeOwner(LocalTerminalRef(terminalID), channel, cols, rows, resizePolicy, surfaceID, viewID)
}

// AttachRefWithResizeOwner 记录指定 TerminalRef 的 attach 结果和输入 channel。
// EndpointID + TerminalID 是 channel map 的路由真值；同名 terminal 在不同 endpoint 下可以同时 attach。
func (store TerminalSessionStore) AttachRefWithResizeOwner(ref TerminalRef, channel uint16, cols int, rows int, resizePolicy string, surfaceID string, viewID string) TerminalSessionStore {
	ref = ref.Normalize()
	store.EndpointID = ref.EndpointID
	store.TerminalID = ref.TerminalID
	store.Channel = channel
	store.InputChannels = cloneInputChannels(store.InputChannels)
	if !ref.Empty() && channel != 0 {
		store.InputChannels[terminalSessionInputKey(ref)] = channel
	}
	store.Attached = true
	store.Cols = cols
	store.Rows = rows
	store.ResizePolicy = resizePolicy
	store.SurfaceID = surfaceID
	store.ViewID = viewID
	store.DesiredCols = cols
	store.DesiredRows = rows
	store.ResizeRequestSeq = 0
	store.ResizeConfirmedSeq = 0
	store.LastError = ""
	store.State = TerminalLiveAttached
	store.ExitCode = 0
	store.ExitReason = ""
	store.ExitedAt = time.Time{}
	store.Command = nil
	return store
}

// RecordInputChannelRef 只登记指定 TerminalRef 的输入 channel，不切换当前 active session。
// 后台 restore/reattach 的 channel 属于对应 TerminalView binding；只有该 view 成为前台
// 或显式 attach 目标时，AttachRefWithResizeOwner 才能改写全局 session 投影。
func (store TerminalSessionStore) RecordInputChannelRef(ref TerminalRef, channel uint16) TerminalSessionStore {
	ref = ref.Normalize()
	if ref.Empty() || channel == 0 {
		return store
	}
	store.InputChannels = cloneInputChannels(store.InputChannels)
	store.InputChannels[terminalSessionInputKey(ref)] = channel
	return store
}

func (store TerminalSessionStore) InputChannelFor(terminalID string) (uint16, bool) {
	return store.InputChannelForRef(LocalTerminalRef(terminalID))
}

// InputChannelForRef 查询指定 TerminalRef 的输入 channel。
// 调用方必须传入 owning endpoint，避免把键盘输入发给其他 daemon 的同名 terminal。
func (store TerminalSessionStore) InputChannelForRef(ref TerminalRef) (uint16, bool) {
	ref = ref.Normalize()
	if ref.Empty() {
		return 0, false
	}
	if store.TerminalRef().Equal(ref) && store.Channel != 0 {
		return store.Channel, true
	}
	channel, ok := store.InputChannels[terminalSessionInputKey(ref)]
	return channel, ok && channel != 0
}

func (store TerminalSessionStore) Resize(cols int, rows int) TerminalSessionStore {
	store.Cols = cols
	store.Rows = rows
	store.DesiredCols = cols
	store.DesiredRows = rows
	store.ResizeConfirmedSeq = store.ResizeRequestSeq
	store.LastError = ""
	store.State = TerminalLiveAttached
	return store
}

func (store TerminalSessionStore) RequestResize(cols int, rows int) TerminalSessionStore {
	store.ResizeRequestSeq++
	store.DesiredCols = cols
	store.DesiredRows = rows
	store.LastError = ""
	store.State = TerminalLiveAttached
	return store
}

func (store TerminalSessionStore) DesiredSize() (int, int) {
	if store.DesiredCols > 0 && store.DesiredRows > 0 {
		return store.DesiredCols, store.DesiredRows
	}
	return store.Cols, store.Rows
}

func (store TerminalSessionStore) ApplyResizeResult(seq uint64, cols int, rows int) (TerminalSessionStore, bool) {
	if store.IsStaleResizeResult(seq) {
		return store, false
	}
	store.Cols = cols
	store.Rows = rows
	store.DesiredCols = cols
	store.DesiredRows = rows
	if seq > store.ResizeConfirmedSeq {
		store.ResizeConfirmedSeq = seq
	}
	store.LastError = ""
	store.State = TerminalLiveAttached
	return store, true
}

func (store TerminalSessionStore) IsStaleResizeResult(seq uint64) bool {
	return seq != 0 && seq < store.ResizeRequestSeq
}

func (store TerminalSessionStore) SetError(err string) TerminalSessionStore {
	return store.SetErrorRef(store.TerminalRef(), err)
}

// SetErrorRef 把当前前台 attach/session 投影标记为指定 TerminalRef 的错误状态。
// 调用方必须先确认该 ref 拥有 active view；后台 endpoint 失败不得调用它改写全局 session。
func (store TerminalSessionStore) SetErrorRef(ref TerminalRef, err string) TerminalSessionStore {
	ref = ref.Normalize()
	if !ref.Empty() {
		if !store.TerminalRef().Equal(ref) {
			store.Channel = 0
			store.Cols = 0
			store.Rows = 0
			store.DesiredCols = 0
			store.DesiredRows = 0
			store.ResizeRequestSeq = 0
			store.ResizeConfirmedSeq = 0
		}
		store.EndpointID = ref.EndpointID
		store.TerminalID = ref.TerminalID
	}
	store.LastError = err
	store.State = TerminalLiveError
	store.Attached = false
	store.ResizePolicy = ""
	store.SurfaceID = ""
	store.ViewID = ""
	return store
}

func (store TerminalSessionStore) MarkExited(terminalID string, exitCode int, reason string) TerminalSessionStore {
	return store.MarkExitedWithMetadataRef(LocalTerminalRef(terminalID), exitCode, reason, time.Time{}, nil)
}

func (store TerminalSessionStore) MarkExitedWithMetadata(terminalID string, exitCode int, reason string, exitedAt time.Time, command []string) TerminalSessionStore {
	return store.MarkExitedWithMetadataRef(LocalTerminalRef(terminalID), exitCode, reason, exitedAt, command)
}

// MarkExitedWithMetadataRef 把当前 session 标记为指定 TerminalRef 的 exited。
// 如果该 ref 不是当前 session，只清理已记录的 input channel，不改写 active terminal。
func (store TerminalSessionStore) MarkExitedWithMetadataRef(ref TerminalRef, exitCode int, reason string, exitedAt time.Time, command []string) TerminalSessionStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store
	}
	if len(store.InputChannels) > 0 {
		store.InputChannels = cloneInputChannels(store.InputChannels)
		delete(store.InputChannels, terminalSessionInputKey(ref))
	}
	if store.TerminalID != "" && !store.TerminalRef().Equal(ref) {
		return store
	}
	store.EndpointID = ref.EndpointID
	store.TerminalID = ref.TerminalID
	store.Attached = false
	store.State = TerminalLiveExited
	store.ExitCode = exitCode
	store.ExitReason = reason
	store.ExitedAt = exitedAt
	store.Command = cloneStrings(command)
	store.LastError = ""
	store.ResizePolicy = ""
	store.SurfaceID = ""
	store.ViewID = ""
	return store
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func cloneLiveSurfaceSnapshots(values map[string]LiveSurfaceSnapshot) map[string]LiveSurfaceSnapshot {
	cloned := make(map[string]LiveSurfaceSnapshot, len(values)+1)
	for key, value := range values {
		// 中文说明：Surfaces map 只需要结构级 COW；每个 snapshot 在写入 map 前已经
		// 克隆成 reducer-owned payload，更新其它 terminal 时不能重复深拷贝旧 screen。
		cloned[key] = value
	}
	return cloned
}

func cloneLiveSurfaceRefreshStates(values map[string]LiveSurfaceRefreshState) map[string]LiveSurfaceRefreshState {
	cloned := make(map[string]LiveSurfaceRefreshState, len(values)+1)
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func CloneLiveSurfaceSnapshot(value LiveSurfaceSnapshot) LiveSurfaceSnapshot {
	value.Lines = cloneStrings(value.Lines)
	value.Screen = cloneLiveCellRows(value.Screen)
	value.RowCopies = cloneLiveRowCopies(value.RowCopies)
	value.ChangedRows = cloneInts(value.ChangedRows)
	value.Command = cloneStrings(value.Command)
	return value
}

func cloneLiveRowCopies(values []LiveRowCopy) []LiveRowCopy {
	return append([]LiveRowCopy(nil), values...)
}

func liveSurfaceRefKey(ref TerminalRef) string {
	return endpointScopedRuntimeKey(ref)
}

func terminalSessionInputKey(ref TerminalRef) string {
	return endpointScopedRuntimeKey(ref)
}

func endpointScopedRuntimeKey(ref TerminalRef) string {
	ref = ref.Normalize()
	if ref.Empty() {
		return ""
	}
	if ref.EndpointID == DefaultEndpointID {
		// 中文说明：local 保持旧 map key，保证本地单 endpoint 的缓存、测试和调试输出不被格式迁移影响。
		return ref.TerminalID
	}
	return ref.Key()
}

func cloneInputChannels(values map[string]uint16) map[string]uint16 {
	cloned := make(map[string]uint16, len(values)+1)
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

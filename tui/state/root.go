package state

// Root 是 reducer-owned TUI-v3 state root。
type Root struct {
	Generation       uint64
	RuntimeSurfaceID string
	History          HistoryStore
	CopyMode         CopyModeStore
	HistoryByView    map[string]HistoryStore
	CopyModeByView   map[string]CopyModeStore
	Clipboard        ClipboardStore
	Surface          TerminalSurfaceStore
	Session          TerminalSessionStore
	TerminalViews    TerminalViewStore
	TerminalPool     TerminalPoolStore
	Endpoints        EndpointStore
	Viewport         ViewportStore
	Shell            ShellStore
	HostTheme        HostThemeStore
	HostCapabilities HostCapabilityStore
	Config           TUIConfigStore
	WorkbenchSync    WorkbenchSyncStore
}

// Advance 返回 generation 递增后的副本。
func (r Root) Advance() Root {
	r.Generation++
	return r
}

// ApplyEndpointTerminalList 把某个 endpoint 的 terminal list 结果回投到 root。
// TerminalPool 只替换该 endpoint 的条目；EndpointStore 只更新该 endpoint 的状态，确保局部失败不会影响其他 endpoint。
func (r Root) ApplyEndpointTerminalList(endpointID EndpointID, items []TerminalPoolItem, err string) Root {
	endpointID = NormalizeEndpointID(endpointID)
	r.TerminalPool = r.TerminalPool.ApplyEndpointList(endpointID, items, err)
	count := len(items)
	if err != "" {
		count = endpointTerminalRowCount(r.TerminalPool.Items, endpointID)
	}
	r.Endpoints = r.Endpoints.MarkTerminalListResult(endpointID, count, err)
	return r
}

func endpointTerminalRowCount(items []TerminalPoolItem, endpointID EndpointID) int {
	endpointID = NormalizeEndpointID(endpointID)
	count := 0
	for _, item := range items {
		item = normalizeTerminalPoolItem(item)
		if item.EndpointID == endpointID && item.TerminalID != "" {
			count++
		}
	}
	return count
}

func (r Root) CopyHistorySessionForView(viewID string) (HistoryStore, CopyModeStore) {
	if viewID == "" {
		return r.History, r.CopyMode
	}
	if historyStoreMatchesView(r.History, viewID) || copyModeStoreMatchesView(r.CopyMode, viewID) {
		return r.History, r.CopyMode
	}
	history, hasHistory := r.HistoryByView[viewID]
	copyMode, hasCopyMode := r.CopyModeByView[viewID]
	if hasHistory || hasCopyMode {
		return history, copyMode
	}
	return HistoryStore{}, CopyModeStore{}
}

func (r Root) WithCopyHistorySession(viewID string, history HistoryStore, copyMode CopyModeStore) Root {
	if viewID == "" {
		viewID = copyHistorySessionViewID(history, copyMode)
	}
	if viewID == "" {
		r.History = history
		r.CopyMode = copyMode
		return r
	}
	if history.ViewID == "" && historyStoreHasState(history) {
		history.ViewID = viewID
	}
	if copyMode.ViewID == "" && copyModeStoreHasState(copyMode) {
		copyMode.ViewID = viewID
	}
	r.History = history
	r.CopyMode = copyMode
	if !copyHistorySessionHasState(history, copyMode) {
		return r.WithoutCopyHistorySession(viewID)
	}
	r.HistoryByView = cloneHistoryStoreMap(r.HistoryByView)
	r.CopyModeByView = cloneCopyModeStoreMap(r.CopyModeByView)
	r.HistoryByView[viewID] = history
	r.CopyModeByView[viewID] = copyMode
	return r
}

func (r Root) WithoutCopyHistorySession(viewID string) Root {
	if viewID == "" {
		return r
	}
	if historyStoreMatchesView(r.History, viewID) || copyModeStoreMatchesView(r.CopyMode, viewID) {
		// 中文说明：删除 view 级 copy/history 会话时，current root 也不能留下
		// terminal id 空壳，否则 render 侧仍会把它当成 copy/history terminal。
		r.History = HistoryStore{}
		r.CopyMode = CopyModeStore{}
	}
	if _, ok := r.HistoryByView[viewID]; ok {
		r.HistoryByView = cloneHistoryStoreMap(r.HistoryByView)
		delete(r.HistoryByView, viewID)
	}
	if _, ok := r.CopyModeByView[viewID]; ok {
		r.CopyModeByView = cloneCopyModeStoreMap(r.CopyModeByView)
		delete(r.CopyModeByView, viewID)
	}
	return r
}

func (r Root) WithoutCopyHistorySessionsForTerminal(terminalID string) Root {
	return r.WithoutCopyHistorySessionsForTerminalRef(LocalTerminalRef(terminalID))
}

// WithoutCopyHistorySessionsForTerminalRef 删除指定 TerminalRef 绑定的 copy/history 会话。
// endpoint 是 copy/history 交互态的路由边界；本地和远端同名 terminal 不能互相清掉 frozen window。
func (r Root) WithoutCopyHistorySessionsForTerminalRef(ref TerminalRef) Root {
	ref = ref.Normalize()
	if ref.Empty() {
		return r
	}
	if copyHistoryStoreMatchesRef(r.History, r.CopyMode, ref) {
		r.History = HistoryStore{}
		r.CopyMode = CopyModeStore{}
	}
	for viewID, history := range r.HistoryByView {
		copyMode := r.CopyModeByView[viewID]
		if copyHistoryStoreMatchesRef(history, copyMode, ref) {
			r = r.WithoutCopyHistorySession(viewID)
		}
	}
	for viewID, copyMode := range r.CopyModeByView {
		history := r.HistoryByView[viewID]
		if copyHistoryStoreMatchesRef(history, copyMode, ref) {
			r = r.WithoutCopyHistorySession(viewID)
		}
	}
	return r
}

func copyHistoryStoreMatchesRef(history HistoryStore, copyMode CopyModeStore, ref TerminalRef) bool {
	ref = ref.Normalize()
	if ref.Empty() {
		return false
	}
	if history.TerminalID != "" && NewTerminalRef(history.EndpointID, history.TerminalID).Equal(ref) {
		return true
	}
	if copyMode.TerminalID != "" && NewTerminalRef(copyMode.EndpointID, copyMode.TerminalID).Equal(ref) {
		return true
	}
	return false
}

func (r Root) ClearCopyHistorySessions() Root {
	r.History = HistoryStore{}
	r.CopyMode = CopyModeStore{}
	r.HistoryByView = nil
	r.CopyModeByView = nil
	return r
}

func (r Root) HasActiveCopyMode() bool {
	if r.CopyMode.InputActive() {
		return true
	}
	for _, copyMode := range r.CopyModeByView {
		if copyMode.InputActive() {
			return true
		}
	}
	return false
}

func (r Root) CopyHistoryTerminalIDs() []string {
	ids := make(map[string]struct{})
	if r.History.TerminalID != "" {
		ids[r.History.TerminalID] = struct{}{}
	}
	if r.CopyMode.TerminalID != "" {
		ids[r.CopyMode.TerminalID] = struct{}{}
	}
	for _, history := range r.HistoryByView {
		if history.TerminalID != "" {
			ids[history.TerminalID] = struct{}{}
		}
	}
	for _, copyMode := range r.CopyModeByView {
		if copyMode.TerminalID != "" {
			ids[copyMode.TerminalID] = struct{}{}
		}
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	return out
}

func copyHistorySessionViewID(history HistoryStore, copyMode CopyModeStore) string {
	if copyMode.ViewID != "" {
		return copyMode.ViewID
	}
	if history.ViewID != "" {
		return history.ViewID
	}
	if copyMode.PaneID != "" {
		return TerminalPaneViewID(copyMode.PaneID)
	}
	if history.PaneID != "" {
		return TerminalPaneViewID(history.PaneID)
	}
	return ""
}

func copyHistorySessionHasState(history HistoryStore, copyMode CopyModeStore) bool {
	return historyStoreHasState(history) || copyModeStoreHasState(copyMode)
}

func historyStoreHasState(history HistoryStore) bool {
	return history.TerminalID != "" ||
		history.Token != "" ||
		len(history.Rows) > 0 ||
		len(history.SourceLines) > 0 ||
		history.Pending != nil
}

func copyModeStoreHasState(copyMode CopyModeStore) bool {
	return copyMode.InputActive() ||
		copyMode.TerminalID != "" ||
		copyMode.BoundToken != "" ||
		copyMode.RequestID != 0
}

func historyStoreMatchesView(history HistoryStore, viewID string) bool {
	if viewID == "" {
		return false
	}
	if history.ViewID == viewID {
		return true
	}
	return history.ViewID == "" && history.PaneID != "" && TerminalPaneViewID(history.PaneID) == viewID
}

func copyModeStoreMatchesView(copyMode CopyModeStore, viewID string) bool {
	if viewID == "" {
		return false
	}
	if copyMode.ViewID == viewID {
		return true
	}
	return copyMode.ViewID == "" && copyMode.PaneID != "" && TerminalPaneViewID(copyMode.PaneID) == viewID
}

func cloneHistoryStoreMap(in map[string]HistoryStore) map[string]HistoryStore {
	out := make(map[string]HistoryStore, len(in)+1)
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneCopyModeStoreMap(in map[string]CopyModeStore) map[string]CopyModeStore {
	out := make(map[string]CopyModeStore, len(in)+1)
	for key, value := range in {
		out[key] = value
	}
	return out
}

type WorkbenchSyncStore struct {
	Ref                WorkbenchStorageRef
	LastSavedVersion   uint64
	LastAppliedVersion uint64
	LastEventVersion   uint64
	BaseVersion        uint64
	ConflictVersion    uint64
	Conflict           bool
}

func (store WorkbenchSyncStore) MarkSaved(ref WorkbenchStorageRef, version uint64) WorkbenchSyncStore {
	store.Ref = ref.WithVersion(version)
	store.LastSavedVersion = version
	store.BaseVersion = version
	store.Conflict = false
	store.ConflictVersion = 0
	return store
}

func (store WorkbenchSyncStore) MarkEvent(version uint64) WorkbenchSyncStore {
	store.LastEventVersion = version
	return store
}

func (store WorkbenchSyncStore) MarkApplied(version uint64) WorkbenchSyncStore {
	store.LastAppliedVersion = version
	store.BaseVersion = version
	store.Conflict = false
	store.ConflictVersion = 0
	return store
}

func (store WorkbenchSyncStore) ShouldIgnoreEvent(eventVersion uint64) bool {
	return eventVersion != 0 && eventVersion == store.LastSavedVersion
}

func (store WorkbenchSyncStore) SaveVersion() uint64 {
	return store.BaseVersion
}

func (store WorkbenchSyncStore) MarkConflict(version uint64) WorkbenchSyncStore {
	store.Conflict = true
	store.ConflictVersion = version
	return store
}

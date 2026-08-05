package state

// OpenConnections 打开独立连接观察与策略页面。
// 页面只保存选中行；Endpoint/Route policy 和 runtime 状态仍来自 Root.Endpoints 的 Go-owned 投影。
func (store ShellStore) OpenConnections() ShellStore {
	store = store.EnsureDefaults()
	store.Overlay = OverlayState{Kind: OverlayConnections, Open: true, SelectedIndex: 0}
	return store
}

// MoveConnectionsSelection 循环移动 Connections Endpoint 选中行，不改变 registry 或当前 session。
func (store ShellStore) MoveConnectionsSelection(delta int, itemCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayConnections || !store.Overlay.Open || itemCount <= 0 || delta == 0 {
		return store
	}
	next := (store.Overlay.SelectedIndex + delta) % itemCount
	if next < 0 {
		next += itemCount
	}
	store.Overlay.SelectedIndex = next
	return store
}

// SetConnectionsSelectedIndex 设置 Connections 的显式点击/恢复位置，并把越界值约束到当前列表。
func (store ShellStore) SetConnectionsSelectedIndex(index int, itemCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayConnections || !store.Overlay.Open || itemCount <= 0 {
		return store
	}
	if index < 0 {
		index = 0
	}
	if index >= itemCount {
		index = itemCount - 1
	}
	store.Overlay.SelectedIndex = index
	return store
}

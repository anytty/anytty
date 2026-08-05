package state

// ToggleWorkbenchTreeItem 切换 Workbench Navigator 的可折叠节点。
// 折叠状态属于 overlay 交互态；它只过滤当前树投影，不修改真实 workspace/tab/pane 布局。
func (store ShellStore) ToggleWorkbenchTreeItem(item WorkbenchTreeItem) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayWorkbenchTree || !store.Overlay.Open || !item.Expandable {
		return store
	}
	return store.SetWorkbenchTreeItemCollapsed(item, !item.Collapsed)
}

// SetWorkbenchTreeItemCollapsed 设置 Workbench Navigator 单个节点的折叠状态。
// 调用方必须传入当前投影里的 workspace/tab item；pane/floating 叶子节点不会产生状态变化。
func (store ShellStore) SetWorkbenchTreeItemCollapsed(item WorkbenchTreeItem, collapsed bool) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayWorkbenchTree || !store.Overlay.Open || !item.Expandable {
		return store
	}
	key, ok := workbenchTreeCollapseKey(item)
	if !ok {
		return store
	}
	next := cloneWorkbenchTreeCollapsed(store.Overlay.WorkbenchCollapsed)
	if collapsed {
		if next == nil {
			next = map[string]bool{}
		}
		next[key] = true
	} else {
		delete(next, key)
		if len(next) == 0 {
			next = nil
		}
	}
	store.Overlay.WorkbenchCollapsed = next
	return store
}

func cloneWorkbenchTreeCollapsed(collapsed map[string]bool) map[string]bool {
	if len(collapsed) == 0 {
		return nil
	}
	next := make(map[string]bool, len(collapsed))
	for key, value := range collapsed {
		if value {
			next[key] = true
		}
	}
	return next
}

func workbenchTreeCollapseKey(item WorkbenchTreeItem) (string, bool) {
	switch item.Kind {
	case WorkbenchTreeKindWorkspace:
		if item.WorkspaceID == "" {
			return "", false
		}
		return "workspace:" + item.WorkspaceID, true
	case WorkbenchTreeKindTab:
		if item.WorkspaceID == "" || item.TabID == "" {
			return "", false
		}
		return "tab:" + item.WorkspaceID + "/" + item.TabID, true
	default:
		return "", false
	}
}

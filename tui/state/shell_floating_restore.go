package state

// NormalizeRestoredFloatingDisplay 为 workbench storage 恢复出来的 floating slot
// 初始化当前 TUI 的本地显示几何。workbench storage 只保存 slot 和 terminal 连接意图；
// 坐标、大小、z-order、折叠和 auto-fit 都不是跨 TUI 共享 truth。这里只在 restore
// 边界按当前 viewport 与 TerminalViewBinding 的 desired terminal size 生成初始 rect，
// 后续拖拽/缩放仍由当前 TUI 的 reducer-owned floating state 接管。
func (store ShellStore) NormalizeRestoredFloatingDisplay(viewport ViewportStore, views TerminalViewStore) ShellStore {
	store = store.EnsureDefaults()
	boundsW, boundsH := restoredFloatingBounds(viewport)
	store.Workspaces = cloneWorkspaces(store.Workspaces)
	activeWorkspaceSeen := false
	for workspaceIndex := range store.Workspaces {
		workspace := normalizeRestoredWorkspaceFloatingDisplay(store.Workspaces[workspaceIndex], views, boundsW, boundsH)
		store.Workspaces[workspaceIndex] = workspace
		if workspace.ID == store.Workspace.ID {
			store.Workspace = workspace
			activeWorkspaceSeen = true
		}
	}
	if !activeWorkspaceSeen {
		store.Workspace = normalizeRestoredWorkspaceFloatingDisplay(store.Workspace, views, boundsW, boundsH)
	}
	store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
	return store.EnsureDefaults()
}

func restoredFloatingBounds(viewport ViewportStore) (int, int) {
	boundsW, boundsH := viewport.Cols, viewport.Rows
	return floatingPlacementBounds(boundsW, boundsH)
}

func normalizeRestoredWorkspaceFloatingDisplay(workspace WorkspaceState, views TerminalViewStore, boundsW int, boundsH int) WorkspaceState {
	workspace = cloneWorkspace(workspace)
	for tabIndex := range workspace.Tabs {
		workspace.Tabs[tabIndex] = normalizeRestoredTabFloatingDisplay(workspace.Tabs[tabIndex], views, boundsW, boundsH)
	}
	return workspace
}

func normalizeRestoredTabFloatingDisplay(tab TabState, views TerminalViewStore, boundsW int, boundsH int) TabState {
	tab.Floatings = cloneFloatings(tab.Floatings)
	placed := make([]FloatingPaneState, 0, len(tab.Floatings))
	for index := range tab.Floatings {
		floating := &tab.Floatings[index]
		if floating.ID == "" {
			continue
		}
		rect := restoredFloatingInitialRect(*floating, views, boundsW, boundsH)
		floating.Rect = cascadeFloatingRect(rect, placed, boundsW, boundsH)
		if floating.Z <= 0 {
			floating.Z = index + 1
		}
		if floating.FitMode == "" {
			floating.FitMode = FloatingFitManual
		}
		placed = append(placed, *floating)
	}
	return ensureTabFloatingDefaults(tab)
}

func restoredFloatingInitialRect(floating FloatingPaneState, views TerminalViewStore, boundsW int, boundsH int) FloatingRect {
	if binding, ok := views.FloatingBinding(floating.ID); ok && binding.DesiredCols > 0 && binding.DesiredRows > 0 {
		return floatingRectForContentSize(binding.DesiredCols, binding.DesiredRows)
	}
	return defaultFloatingRect(FloatingRect{}, boundsW, boundsH)
}

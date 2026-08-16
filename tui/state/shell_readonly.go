package state

// ReadonlyDefaults 只给 render / projection 这类只读路径使用。
// 已规范化的 ShellStore 会直接复用 slice backing，避免每帧深拷贝 workspace；更新路径仍必须使用 EnsureDefaults。
func (store ShellStore) ReadonlyDefaults() ShellStore {
	if store.readonlyDefaultsReady() {
		return store
	}
	return store.EnsureDefaults()
}

func (store ShellStore) readonlyDefaultsReady() bool {
	if !store.initialized || store.Workspace.ID == "" || store.Workspace.Name == "" || store.PanelPresentation == "" {
		return false
	}
	if !readonlyWorkspacesReady(store.Workspaces, store.Workspace) {
		return false
	}
	if len(store.Workspace.Tabs) == 0 {
		return store.Workspace.ActiveTabID == "" && store.ActivePaneID == "" && store.ZoomedPaneID == ""
	}
	activeTabIndex := -1
	for index, tab := range store.Workspace.Tabs {
		if !readonlyTabDefaultsReady(tab, store.Workspace.ActiveTabID, store.ActivePaneID) {
			return false
		}
		if tab.ID == store.Workspace.ActiveTabID {
			activeTabIndex = index
		}
	}
	if activeTabIndex < 0 {
		return false
	}
	activeTab := store.Workspace.Tabs[activeTabIndex]
	if len(activeTab.Panes) > 0 && !readonlyPaneExists(activeTab.Panes, store.ActivePaneID) {
		return false
	}
	if store.ZoomedPaneID != "" && !readonlyPaneExists(activeTab.Panes, store.ZoomedPaneID) {
		return false
	}
	return true
}

func readonlyWorkspacesReady(workspaces []WorkspaceState, active WorkspaceState) bool {
	if active.ID == "" {
		return false
	}
	for _, workspace := range workspaces {
		if workspace.ID != active.ID {
			continue
		}
		return readonlyWorkspaceSharesActive(workspace, active)
	}
	return false
}

func readonlyWorkspaceSharesActive(workspace WorkspaceState, active WorkspaceState) bool {
	if workspace.ID != active.ID || workspace.Name != active.Name || workspace.ActiveTabID != active.ActiveTabID || len(workspace.Tabs) != len(active.Tabs) {
		return false
	}
	if len(active.Tabs) == 0 {
		return true
	}
	// 当前 workspace 应由 EnsureDefaults/upsertWorkspace 写回 Workspaces；共享 backing 才能证明不是旧投影。
	return &workspace.Tabs[0] == &active.Tabs[0]
}

func readonlyTabDefaultsReady(tab TabState, activeTabID string, activePaneID string) bool {
	if tab.ID == "" || tab.Title == "" {
		return false
	}
	if len(tab.Panes) == 0 {
		if tab.ActivePaneID != "" || tab.RootSplit.PaneID != "" || len(tab.RootSplit.Children) != 0 {
			return false
		}
		return readonlyFloatingsDefaultsReady(tab.Floatings, tab.ActiveFloatingID)
	}
	if tab.ActivePaneID == "" || tab.RootSplit.empty() {
		return false
	}
	tabActive := tab.ID == activeTabID
	for _, pane := range tab.Panes {
		if pane.ID == "" || pane.Kind == "" {
			return false
		}
		if pane.Active != (tabActive && pane.ID == activePaneID) {
			return false
		}
	}
	if tabActive && activePaneID == "" {
		return false
	}
	return readonlyFloatingsDefaultsReady(tab.Floatings, tab.ActiveFloatingID)
}

func readonlyFloatingsDefaultsReady(floatings []FloatingPaneState, activeFloatingID string) bool {
	if len(floatings) == 0 {
		return activeFloatingID == ""
	}
	activeFound := activeFloatingID == ""
	for _, floating := range floatings {
		if floating.ID == "" || floating.Title == "" || floating.Pane.ID == "" || floating.Pane.Title == "" || floating.Pane.Kind == "" {
			return false
		}
		if floating.Rect.W <= 0 || floating.Rect.H <= 0 || floating.Z <= 0 || floating.FitMode == "" {
			return false
		}
		if floating.ID == activeFloatingID && !floating.Collapsed {
			activeFound = true
		}
		if floating.Active != (floating.ID == activeFloatingID) {
			return false
		}
	}
	return activeFound
}

func readonlyPaneExists(panes []PaneState, paneID string) bool {
	if paneID == "" {
		return false
	}
	for _, pane := range panes {
		if pane.ID == paneID {
			return true
		}
	}
	return false
}

func (node SplitNode) empty() bool {
	return node.PaneID == "" && node.Direction == "" && len(node.Children) == 0
}

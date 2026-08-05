package state

func (workspace WorkspaceState) ensureActive(activePaneID string) WorkspaceState {
	for tabIndex := range workspace.Tabs {
		tab := &workspace.Tabs[tabIndex]
		tabActive := tab.ID == workspace.ActiveTabID
		if tab.ActivePaneID == "" && len(tab.Panes) > 0 {
			tab.ActivePaneID = tab.Panes[0].ID
		}
		for paneIndex := range tab.Panes {
			pane := &tab.Panes[paneIndex]
			pane.Active = tabActive && pane.ID == activePaneID
		}
	}
	return workspace
}

func (workspace WorkspaceState) ensureTabDefaults() WorkspaceState {
	for tabIndex := range workspace.Tabs {
		tab := &workspace.Tabs[tabIndex]
		if tab.ID == "" {
			tab.ID = DefaultTabID
		}
		if tab.Title == "" {
			tab.Title = tab.ID
		}
		if tab.ActivePaneID == "" && len(tab.Panes) > 0 {
			tab.ActivePaneID = tab.Panes[0].ID
		}
		if tab.RootSplit.PaneID == "" && tab.RootSplit.Direction == "" && len(tab.RootSplit.Children) == 0 {
			tab.RootSplit = SplitNode{PaneID: tab.ActivePaneID}
		}
	}
	return workspace
}

func (workspace WorkspaceState) ensureActiveTab() WorkspaceState {
	if len(workspace.Tabs) == 0 {
		return workspace
	}
	for _, tab := range workspace.Tabs {
		if tab.ID == workspace.ActiveTabID {
			return workspace
		}
	}
	workspace.ActiveTabID = workspace.Tabs[0].ID
	return workspace
}

func (workspace WorkspaceState) ensureDefaults() WorkspaceState {
	if workspace.ID == "" {
		workspace.ID = DefaultWorkspaceID
	}
	if workspace.Name == "" {
		workspace.Name = workspace.ID
		if workspace.Name == DefaultWorkspaceID {
			workspace.Name = "main"
		}
	}
	workspace = workspace.ensureTabDefaults()
	return workspace.ensureActiveTab()
}

func defaultTabState() TabState {
	return TabState{
		ID:           DefaultTabID,
		Title:        "main",
		ActivePaneID: DefaultPaneID,
		Panes: []PaneState{{
			ID:     DefaultPaneID,
			Title:  "shell",
			Kind:   PaneTerminalLive,
			Active: true,
		}},
		RootSplit: SplitNode{PaneID: DefaultPaneID},
	}
}

func (workspace WorkspaceState) activeTab() TabState {
	for _, tab := range workspace.Tabs {
		if tab.ID == workspace.ActiveTabID {
			return tab
		}
	}
	if len(workspace.Tabs) > 0 {
		return workspace.Tabs[0]
	}
	return TabState{}
}

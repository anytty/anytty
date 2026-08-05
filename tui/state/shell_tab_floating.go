package state

func (store ShellStore) ensureFloatingDefaults() ShellStore {
	for tabIndex := range store.Workspace.Tabs {
		store.Workspace.Tabs[tabIndex] = ensureTabFloatingDefaults(store.Workspace.Tabs[tabIndex])
	}
	return store
}

func ensureTabFloatingDefaults(tab TabState) TabState {
	tab.Floatings = cloneFloatings(tab.Floatings)
	if len(tab.Floatings) == 0 {
		tab.ActiveFloatingID = ""
		return tab
	}
	activeFound := false
	for index := range tab.Floatings {
		floating := &tab.Floatings[index]
		if floating.ID == "" {
			continue
		}
		if floating.Title == "" {
			floating.Title = floating.ID
		}
		if floating.Pane.ID == "" {
			floating.Pane = PaneState{ID: floating.ID + "-pane", Title: floating.Title, Kind: PaneEmpty}
		}
		if floating.Pane.Title == "" {
			floating.Pane.Title = floating.Title
		}
		if floating.Pane.Kind == "" {
			floating.Pane.Kind = PaneEmpty
		}
		if floating.Rect.W <= 0 {
			floating.Rect.W = 40
		}
		if floating.Rect.H <= 0 {
			floating.Rect.H = 10
		}
		if floating.Z <= 0 {
			floating.Z = index + 1
		}
		if floating.FitMode != FloatingFitAuto {
			floating.FitMode = FloatingFitManual
			if floating.AutoFit.Cols < 0 {
				floating.AutoFit.Cols = 0
			}
			if floating.AutoFit.Rows < 0 {
				floating.AutoFit.Rows = 0
			}
		}
		if floating.ID == tab.ActiveFloatingID && !floating.Collapsed {
			activeFound = true
		}
	}
	if tab.ActiveFloatingID != "" && !activeFound {
		tab.ActiveFloatingID = topExpandedFloatingID(tab.Floatings)
	}
	for index := range tab.Floatings {
		tab.Floatings[index].Active = tab.Floatings[index].ID == tab.ActiveFloatingID
	}
	return tab
}

func (store ShellStore) activeFloatings() []FloatingPaneState {
	tabIndex := store.activeTabIndex()
	if tabIndex < 0 {
		return nil
	}
	return store.Workspace.Tabs[tabIndex].Floatings
}

func (store ShellStore) activeFloatingID() string {
	tabIndex := store.activeTabIndex()
	if tabIndex < 0 {
		return ""
	}
	return store.Workspace.Tabs[tabIndex].ActiveFloatingID
}

func (store ShellStore) withActiveTabFloatings(floatings []FloatingPaneState, activeID string) ShellStore {
	tabIndex := store.activeTabIndex()
	if tabIndex < 0 {
		return store
	}
	store.Workspace.Tabs[tabIndex].Floatings = cloneFloatings(floatings)
	store.Workspace.Tabs[tabIndex].ActiveFloatingID = activeID
	store.Workspace.Tabs[tabIndex] = ensureTabFloatingDefaults(store.Workspace.Tabs[tabIndex])
	store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
	return store
}

func (store ShellStore) floatingIndex(id string) int {
	for index, floating := range store.activeFloatings() {
		if floating.ID == id {
			return index
		}
	}
	return -1
}

func (store ShellStore) floatingIndexOrActive(id string) int {
	if id != "" {
		return store.floatingIndex(id)
	}
	if activeID := store.activeFloatingID(); activeID != "" {
		return store.floatingIndex(activeID)
	}
	floatings := store.activeFloatings()
	if len(floatings) == 0 {
		return -1
	}
	topID := topExpandedFloatingID(floatings)
	if topID == "" {
		return -1
	}
	return store.floatingIndex(topID)
}

func (store ShellStore) floatingIndexForToggleCollapse(id string) int {
	if id != "" {
		return store.floatingIndex(id)
	}
	if activeID := store.activeFloatingID(); activeID != "" {
		return store.floatingIndex(activeID)
	}
	floatings := store.activeFloatings()
	if len(floatings) == 0 {
		return -1
	}
	if topID := topExpandedFloatingID(floatings); topID != "" {
		return store.floatingIndex(topID)
	}
	return store.floatingIndex(topFloatingID(floatings))
}

func (store ShellStore) nextFloatingZ() int {
	maxZ := 0
	for _, floating := range store.activeFloatings() {
		if floating.Z > maxZ {
			maxZ = floating.Z
		}
	}
	return maxZ
}

func (store ShellStore) FloatingByID(id string) (FloatingPaneState, bool) {
	store = store.ReadonlyDefaults()
	for _, tab := range store.Workspace.Tabs {
		for _, floating := range tab.Floatings {
			if floating.ID == id {
				return floating, true
			}
		}
	}
	return FloatingPaneState{}, false
}

func (store ShellStore) ActiveFloatingID() string {
	return store.ReadonlyDefaults().activeFloatingID()
}

func (store ShellStore) ActiveFloatings() []FloatingPaneState {
	return cloneFloatings(store.ReadonlyDefaults().activeFloatings())
}

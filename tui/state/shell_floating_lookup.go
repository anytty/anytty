package state

// ActiveSurfaceTarget is the single source of truth for commands that operate
// on the currently focused workbench surface. A focused floating always wins
// over the tiled pane that remains active underneath it.
type ActiveSurfaceTarget struct {
	PaneID     string
	FloatingID string
	Floating   bool
}

func (store ShellStore) ActiveSurfaceTarget() (ActiveSurfaceTarget, bool) {
	store = store.ReadonlyDefaults()
	if floatingID := store.ActiveFloatingID(); floatingID != "" {
		if floating, ok := store.FloatingByID(floatingID); ok && floating.Pane.ID != "" {
			return ActiveSurfaceTarget{PaneID: floating.Pane.ID, FloatingID: floating.ID, Floating: true}, true
		}
		return ActiveSurfaceTarget{}, false
	}
	if store.ActivePaneID == "" {
		return ActiveSurfaceTarget{}, false
	}
	return ActiveSurfaceTarget{PaneID: store.ActivePaneID}, true
}

func (store ShellStore) FloatingByPaneID(paneID string) (FloatingPaneState, bool) {
	if paneID == "" {
		return FloatingPaneState{}, false
	}
	store = store.ReadonlyDefaults()
	for _, tab := range store.Workspace.Tabs {
		for _, floating := range tab.Floatings {
			if floating.Pane.ID == paneID {
				return floating, true
			}
		}
	}
	return FloatingPaneState{}, false
}

func (store ShellStore) FloatingIDForPaneID(paneID string) (string, bool) {
	floating, ok := store.FloatingByPaneID(paneID)
	if !ok || floating.ID == "" {
		return "", false
	}
	return floating.ID, true
}

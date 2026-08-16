package state

import "testing"

func TestRootAdvance(t *testing.T) {
	next := (Root{}).Advance()
	if next.Generation != 1 {
		t.Fatalf("expected generation 1, got %d", next.Generation)
	}
}

func TestWorkbenchSyncStoreTracksSavedEventAndAppliedVersions(t *testing.T) {
	ref := DefaultWorkbenchStorageRef("").WithVersion(3)
	store := (WorkbenchSyncStore{}).MarkSaved(ref, 3).MarkEvent(4).MarkApplied(4)
	if store.Ref.Version != 3 || store.LastSavedVersion != 3 || store.LastEventVersion != 4 || store.LastAppliedVersion != 4 {
		t.Fatalf("unexpected sync store %#v", store)
	}
	if store.SaveVersion() != 4 || store.Conflict {
		t.Fatalf("applied version should become save base and clear conflict, got %#v", store)
	}
	if !store.ShouldIgnoreEvent(3) || store.ShouldIgnoreEvent(4) {
		t.Fatalf("unexpected self-event filtering %#v", store)
	}
	store = store.MarkConflict(store.SaveVersion())
	if !store.Conflict || store.ConflictVersion != 4 {
		t.Fatalf("conflict marker should keep base version, got %#v", store)
	}
}

func TestRootContainsReducerOwnedShellStore(t *testing.T) {
	root := Root{Shell: DefaultShell()}
	if root.Shell.ActivePaneID != DefaultPaneID {
		t.Fatalf("expected root shell active pane, got %#v", root.Shell)
	}
}

func TestRootWithoutCopyHistorySessionDropsCurrentWindowReferences(t *testing.T) {
	viewID := TerminalPaneViewID(DefaultPaneID)
	root := Root{
		History: HistoryStore{
			PaneID:      DefaultPaneID,
			ViewID:      viewID,
			TerminalID:  "term-1",
			Token:       "tok-1",
			SourceLines: []HistoryLogicalLine{{Text: "history", LineID: 1}},
			Rows:        []HistoryRow{{Text: "history", LineID: 1}},
		},
		CopyMode: CopyModeStore{
			Active:     true,
			PaneID:     DefaultPaneID,
			ViewID:     viewID,
			TerminalID: "term-1",
			BoundToken: "tok-1",
		},
	}
	root = root.WithCopyHistorySession(viewID, root.History, root.CopyMode)

	root = root.WithoutCopyHistorySession(viewID)

	if root.History.TerminalID != "" || root.History.Token != "" || len(root.History.Rows) != 0 || len(root.History.SourceLines) != 0 {
		t.Fatalf("current history must not retain released copy window, got %#v", root.History)
	}
	if root.CopyMode.Active || root.CopyMode.TerminalID != "" || root.CopyMode.BoundToken != "" {
		t.Fatalf("current copy mode must be cleared with released view, got %#v", root.CopyMode)
	}
	if _, ok := root.HistoryByView[viewID]; ok {
		t.Fatalf("history session map must drop released view, got %#v", root.HistoryByView)
	}
	if _, ok := root.CopyModeByView[viewID]; ok {
		t.Fatalf("copy session map must drop released view, got %#v", root.CopyModeByView)
	}
	if ids := root.CopyHistoryTerminalIDs(); len(ids) != 0 {
		t.Fatalf("released copy history must not leak terminal ids, got %#v", ids)
	}
}

func TestRootWithoutCopyHistorySessionsForTerminalRefKeepsOtherEndpoint(t *testing.T) {
	localViewID := TerminalPaneViewID("pane-local")
	westViewID := TerminalPaneViewID("pane-west")
	root := Root{
		HistoryByView: map[string]HistoryStore{
			localViewID: {EndpointID: DefaultEndpointID, TerminalID: "term-1", Token: "local-token"},
			westViewID:  {EndpointID: "west", TerminalID: "term-1", Token: "west-token"},
		},
		CopyModeByView: map[string]CopyModeStore{
			localViewID: {EndpointID: DefaultEndpointID, TerminalID: "term-1", BoundToken: "local-token"},
			westViewID:  {EndpointID: "west", TerminalID: "term-1", BoundToken: "west-token"},
		},
	}

	root = root.WithoutCopyHistorySessionsForTerminalRef(NewTerminalRef("west", "term-1"))

	if _, ok := root.HistoryByView[westViewID]; ok {
		t.Fatalf("west history session should be removed")
	}
	if _, ok := root.CopyModeByView[westViewID]; ok {
		t.Fatalf("west copy session should be removed")
	}
	if _, ok := root.HistoryByView[localViewID]; !ok {
		t.Fatalf("local history session must remain")
	}
	if _, ok := root.CopyModeByView[localViewID]; !ok {
		t.Fatalf("local copy session must remain")
	}
}

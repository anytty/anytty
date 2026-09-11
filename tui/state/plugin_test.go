package state

import "testing"

func TestPluginOwnerLifetimeAndSelection(t *testing.T) {
	shell := DefaultShell()
	w := shell.Workspace.ID
	tab := shell.Workspace.ActiveTabID
	shell.Workspace.Tabs[0].Floatings = []FloatingPaneState{{ID: "float-a"}}
	shell.Workspaces[0] = shell.Workspace
	for _, o := range []PluginOwner{{Kind: "workspace", WorkspaceID: w}, {Kind: "tab", WorkspaceID: w, TabID: tab}, {Kind: "panel", WorkspaceID: w, TabID: tab, PaneID: DefaultPaneID}, {Kind: "floating", WorkspaceID: w, TabID: tab, FloatingID: "float-a"}} {
		if !o.Exists(shell) || !o.Visible(shell) {
			t.Fatalf("owner unavailable: %+v", o)
		}
	}
	wrong := PluginOwner{Kind: "floating", WorkspaceID: w, TabID: "other", FloatingID: "float-a"}
	if wrong.Exists(shell) {
		t.Fatal("floating must not resolve through active tab")
	}
	owner := PluginOwner{Kind: "panel", WorkspaceID: w, TabID: tab, PaneID: DefaultPaneID}
	m := PluginMount{ID: "m", PluginID: "p", DaemonID: "d", Owner: owner, Slot: "content", Revision: 1, Nodes: []PluginNode{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}}}
	store, err := (PluginStore{}).Apply(shell, m, 0)
	if err != nil {
		t.Fatal(err)
	}
	selected := store.Mounts["m"].Move(1)
	store = store.Set(selected)
	m.Revision = 2
	m.Nodes = []PluginNode{{ID: "b", Text: "B working"}, {ID: "a", Text: "A"}}
	store, err = store.Apply(shell, m, 1)
	if err != nil || store.Mounts["m"].SelectedID != "b" {
		t.Fatal("reordering changed selected identity")
	}
	m.Revision = 3
	if _, err = store.Apply(shell, m, 1); err == nil {
		t.Fatal("stale revision accepted")
	}
	shell.Workspace.Tabs[0].Panes = nil
	shell.Workspaces[0] = shell.Workspace
	store, removed := store.Prune(shell)
	if len(removed) != 1 || len(store.Mounts) != 0 {
		t.Fatal("closed owner retained mount")
	}
}
func TestPluginSlotBoundary(t *testing.T) {
	if ValidPluginSlot("workspace", "content") || ValidPluginSlot("floating", "sidebar") || !ValidPluginSlot("workspace", "statusbar") {
		t.Fatal("invalid owner slot matrix")
	}
}

func TestDynamicPluginOwnerFollowsActiveTab(t *testing.T) {
	shell := DefaultShell()
	shell.Workspace.Tabs = append(shell.Workspace.Tabs, TabState{ID: "tab-b", Panes: []PaneState{{ID: "pane-b", Kind: PaneTerminalLive}}})
	shell.Workspace.ActiveTabID = shell.Workspace.Tabs[0].ID
	shell.ActivePaneID = DefaultPaneID
	owner := PluginOwner{Kind: "tab", WorkspaceID: shell.Workspace.ID, TabID: shell.Workspace.ActiveTabID}
	store, err := (PluginStore{}).Apply(shell, PluginMount{ID: "surface", PluginID: "plugin", DaemonID: "daemon", Owner: owner, SurfaceID: "plugin.surface", Scope: "active_tab", Slot: "sidebar", Revision: 1, Nodes: []PluginNode{{ID: "row", Text: "row"}}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	shell.Workspace.ActiveTabID = "tab-b"
	shell.ActivePaneID = "pane-b"
	shell.Workspaces[0] = shell.Workspace
	next, changed := store.ReconcileDynamicOwners(shell)
	if len(changed) != 1 || next.Mounts["surface"].Owner.TabID != "tab-b" {
		t.Fatalf("dynamic owner did not follow active tab: changed=%v mount=%+v", changed, next.Mounts["surface"])
	}
}

func TestDynamicPluginSurfaceCanWaitForMissingTab(t *testing.T) {
	shell := DefaultShell()
	shell.Workspace.Tabs = nil
	shell.Workspace.ActiveTabID = ""
	shell.ActivePaneID = ""
	shell.Workspaces[0] = shell.Workspace
	owner := PluginOwner{Kind: "tab", WorkspaceID: shell.Workspace.ID}
	store, err := (PluginStore{}).Apply(shell, PluginMount{ID: "surface", PluginID: "plugin", DaemonID: "daemon", Owner: owner, SurfaceID: "plugin.surface", Scope: "active_tab", Slot: "sidebar", Revision: 1}, 0)
	if err != nil {
		t.Fatal("dynamic surface should be retained while no tab exists:", err)
	}
	shell.Workspace.Tabs = []TabState{{ID: "tab-new", Title: "New", Panes: []PaneState{{ID: "pane-new", Title: "Pane", Kind: PaneTerminalLive}}}}
	shell.Workspace.ActiveTabID = "tab-new"
	shell.ActivePaneID = "pane-new"
	shell.Workspaces[0] = shell.Workspace
	next, changed := store.ReconcileDynamicOwners(shell)
	if len(changed) != 1 || next.Mounts["surface"].Owner.TabID != "tab-new" {
		t.Fatalf("dynamic surface did not bind after tab creation: changed=%v mount=%+v", changed, next.Mounts["surface"])
	}
}

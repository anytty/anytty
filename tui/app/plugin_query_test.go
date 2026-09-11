package app

import (
	"testing"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

func TestPluginSnapshotReplyUsesIngressDaemonAndPublicProjection(t *testing.T) {
	root, deps, service := pluginFixture(t)
	root.Generation = 42
	root.Plugins.FocusedMountID = ""
	root.Plugins.Peers = []state.PluginPeer{{EndpointID: "local", Host: &apipb.PluginAddress{DaemonId: "daemon-a"}}, {EndpointID: "remote", Host: &apipb.PluginAddress{DaemonId: "daemon-b"}}}
	root.Shell.Workspace.Tabs[0].Floatings = []state.FloatingPaneState{{ID: "f", Pane: state.PaneState{ID: "fp", Kind: state.PaneTerminalLive}}}
	root.TerminalViews = root.TerminalViews.BindFloating(state.NewEndpointFloatingTerminalView("remote", "f", "fp", "remote-terminal", 2, 80, 24, state.TerminalResizeRoleFollower, "surface", state.TerminalFloatingViewID("f"), false))
	root.Shell.Workspaces = append(root.Shell.Workspaces, state.WorkspaceState{ID: "inactive", ActiveTabID: "inactive-tab", Tabs: []state.TabState{{ID: "inactive-tab", Panes: []state.PaneState{{ID: "unverified", Kind: state.PaneTerminalLive}}}}})
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView("client-only-alias", "unverified", "not-a-global-id", 3, 80, 24, state.TerminalResizeRoleFollower, "surface", "unverified-view", false))
	request := &apipb.PluginMessage{RequestId: "snapshot-read", TraceId: "trace", Source: &apipb.PluginAddress{DaemonId: "daemon-b", TuiInstanceId: "tui-a", PluginId: "agents", PluginInstanceId: "reader", RegistrationEpoch: 5}, Destination: &apipb.PluginAddress{DaemonId: "daemon-b", TuiInstanceId: "tui-a", PluginId: "agents", PluginInstanceId: "host"}, Body: &apipb.PluginMessage_UiQuery{UiQuery: &apipb.PluginUiQuery{Query: &apipb.PluginUiQuery_Snapshot{Snapshot: &apipb.PluginUiSnapshotQuery{}}}}}
	next, effects := NewPluginReducer(deps)(root, PluginDeliveryMsg{Delivery: port.PluginDelivery{EndpointID: "remote", Message: request}})
	if next.Generation != root.Generation || len(service.messages) != 0 {
		t.Fatal("read mutated state or bypassed daemon effect")
	}
	runPluginEffects(t, effects)
	if len(service.messages) != 1 || service.endpoints[0] != "remote" {
		t.Fatal("snapshot did not return through ingress daemon")
	}
	message := service.messages[0]
	snapshot := message.GetReply().GetUiSnapshot()
	if snapshot == nil || message.GetReply().RequestId != "snapshot-read" || message.Destination.PluginInstanceId != "reader" || snapshot.Revision != 42 {
		t.Fatal("lost correlation or snapshot revision")
	}
	if snapshot.ActiveContext.ContextId != "" {
		t.Fatal("read query minted mutation authority")
	}
	if len(snapshot.Owners) != 7 || len(snapshot.Panels) != 3 {
		t.Fatalf("missing inactive/tab/floating owners: %v", snapshot)
	}
	for _, panel := range snapshot.Panels {
		switch panel.ViewId {
		case state.TerminalPaneViewID(state.DefaultPaneID):
			if panel.Terminal.GetDaemonId() != "daemon-a" || !panel.Focused {
				t.Fatal("active binding/focus wrong", panel)
			}
		case state.TerminalFloatingViewID("f"):
			if panel.Terminal.GetDaemonId() != "daemon-b" || panel.Owner.GetFloating().GetTabId() == "" || panel.Focused {
				t.Fatal("floating lost owner/domain", panel)
			}
		case "unverified-view":
			if panel.Terminal != nil {
				t.Fatal("unverified client alias exposed as daemon identity")
			}
		default:
			t.Fatalf("unexpected panel %v", panel)
		}
	}
	request.Destination.TuiInstanceId = "other-tui"
	_, effects = NewPluginReducer(deps)(root, PluginDeliveryMsg{Delivery: port.PluginDelivery{EndpointID: "remote", Message: request}})
	if len(effects) != 0 {
		t.Fatal("query reached another TUI")
	}
}

func TestPluginSnapshotDistinguishesFloatingAndPluginFocus(t *testing.T) {
	root, deps, _ := pluginFixture(t)
	root.Shell.Workspace.Tabs[0].Floatings = []state.FloatingPaneState{{ID: "f", Pane: state.PaneState{ID: "fp"}}}
	root.Shell.Workspace.Tabs[0].ActiveFloatingID = "f"
	root.Plugins.FocusedMountID = ""
	snapshot := pluginUISnapshot(root, deps)
	if snapshot.ActiveContext.FloatingId != "f" {
		t.Fatal("missing floating focus")
	}
	for _, panel := range snapshot.Panels {
		if panel.Focused != (panel.Owner.GetFloating() != nil) {
			t.Fatal("two content targets focused")
		}
	}
	root.Plugins.FocusedMountID = "agents"
	snapshot = pluginUISnapshot(root, deps)
	if snapshot.ActiveContext.MountId != "agents" {
		t.Fatal("missing plugin focus")
	}
	for _, panel := range snapshot.Panels {
		if panel.Focused {
			t.Fatal("content claimed focus while plugin owns input")
		}
	}
}

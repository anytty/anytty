package app

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/plugins/agents"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

// Exercise the shipped plugin through the real host and renderer: isolated
// fixture tests cannot detect a mismatch in wire slots or terminal ownership.
func TestPluginAgentsBuildMountThroughHostAndRenderer(t *testing.T) {
	for _, kind := range []string{"workspace", "panel"} {
		t.Run(kind, func(t *testing.T) {
			root, deps, service := pluginFixture(t)
			root.Plugins = state.PluginStore{}
			owner := &apipb.PluginMountOwner{Owner: &apipb.PluginMountOwner_Workspace{Workspace: &apipb.PluginWorkspaceOwner{WorkspaceId: root.Shell.Workspace.ID}}}
			if kind == "panel" {
				owner = &apipb.PluginMountOwner{Owner: &apipb.PluginMountOwner_Panel{Panel: &apipb.PluginPanelOwner{WorkspaceId: root.Shell.Workspace.ID, TabId: root.Shell.Workspace.ActiveTabID, PaneId: state.DefaultPaneID}}}
			}
			entries := []agents.Entry{
				{EndpointID: "local", Report: &apipb.PluginAgentReport{Provider: "codex", SessionId: "local-agent", Title: "Local task", State: "working", Terminal: &apipb.PluginTerminalRef{DaemonId: "daemon-a", TerminalId: "old"}}},
				{EndpointID: "remote", Report: &apipb.PluginAgentReport{Provider: "opencode", SessionId: "remote-agent", Title: "Remote task", State: "blocked", Terminal: &apipb.PluginTerminalRef{DaemonId: "daemon-b", TerminalId: "old"}}},
			}
			update := agents.BuildMount(entries, owner, agents.MountID(owner), 1, false)
			source := &apipb.PluginAddress{DaemonId: "daemon-a", TuiInstanceId: deps.TUIInstanceID, PluginId: agents.PluginID, PluginInstanceId: "process", RegistrationEpoch: 1}
			delivery := port.PluginDelivery{EndpointID: "local", Message: &apipb.PluginMessage{RequestId: "real-agent-mount", Source: source, Destination: &apipb.PluginAddress{TuiInstanceId: deps.TUIInstanceID}, Body: &apipb.PluginMessage_MountUpdate{MountUpdate: update}}}
			next, effects := NewPluginReducer(deps)(root, PluginDeliveryMsg{delivery})
			runPluginEffects(t, effects)
			if len(service.messages) != 1 || service.messages[0].GetReply() == nil || service.messages[0].GetReply().GetError() != nil {
				t.Fatalf("real Agent output rejected by host: %v", service.messages)
			}
			vm := render.NewRenderVMBuilder().Build(next)
			if len(vm.Shell.Plugins) != 1 {
				t.Fatalf("real Agent mount missing from render projection: %v", vm.Shell.Plugins)
			}
			rows := vm.Shell.Plugins[0].Mount.Rows()
			if kind == "panel" {
				if len(rows) != 1 || rows[0].TerminalID != "old" || rows[0].EndpointID != "local" || rows[0].Text != "working" {
					t.Fatalf("panel badge must filter same-named remote terminal: %+v", rows)
				}
			} else if len(rows) != 2 {
				t.Fatalf("aggregate sidebar lost Agent rows: %+v", rows)
			}
			rendered := strings.Join(render.NewRenderer(render.DefaultTheme()).RenderResult(vm).Lines(), "\n")
			if kind == "panel" && (!strings.Contains(rendered, "working") || strings.Contains(rendered, "blocked")) || kind == "workspace" && (!strings.Contains(rendered, "Local task") || !strings.Contains(rendered, "Remote task")) {
				t.Fatalf("Agent content rendered incorrectly: %s", rendered)
			}
		})
	}
}

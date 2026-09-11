package host

import (
	"testing"

	"github.com/anytty/anytty/proto/apipb"
)

func TestUIReadRequiresExplicitCapability(t *testing.T) {
	command := &apipb.PluginCommand{Command: &apipb.PluginCommand_Send{Send: &apipb.PluginSendRequest{Message: &apipb.PluginMessage{Body: &apipb.PluginMessage_UiQuery{UiQuery: &apipb.PluginUiQuery{Query: &apipb.PluginUiQuery_Snapshot{Snapshot: &apipb.PluginUiSnapshotQuery{}}}}}}}}
	manifest := Manifest{ID: "example.reader", Capabilities: Capabilities{TUI: []string{"messages.send", "ui.mounts", "ui.notifications"}}}
	if err := authorizeCommand(manifest, "tui", command); err == nil {
		t.Fatal("unrelated UI capabilities granted workspace reads")
	}
	manifest.Capabilities.TUI = append(manifest.Capabilities.TUI, "ui.read")
	if err := authorizeCommand(manifest, "tui", command); err != nil {
		t.Fatal(err)
	}
	if err := authorizeCommand(manifest, "daemon", command); err == nil {
		t.Fatal("TUI grant leaked into daemon component")
	}
}

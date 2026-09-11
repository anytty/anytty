package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	clientprotocol "github.com/anytty/anytty/client/adapter/protocol"
	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/plugin/host"
	"github.com/anytty/anytty/plugin/sdk"
	"github.com/anytty/anytty/plugins/agents"
	"github.com/anytty/anytty/proto/apipb"
)

// A real subprocess uses the same SDK and Agent entry points as the distributed
// binary. Its stdout is exclusively framed protobuf, never test runner output.
func TestPluginAgentProcessHelper(t *testing.T) {
	if os.Getenv("ANYTTY_AGENT_PROCESS_TEST") != "1" {
		return
	}
	client := sdk.NewStdioClient(os.Stdin, os.Stdout)
	var err error
	if os.Getenv("ANYTTY_PLUGIN_MODE") == "daemon" {
		err = agents.RunDaemon(context.Background(), client, "service-test")
	} else {
		var endpoints []string
		_ = json.Unmarshal([]byte(os.Getenv("ANYTTY_PLUGIN_ENDPOINTS")), &endpoints)
		err = agents.RunTUI(context.Background(), client, os.Getenv("ANYTTY_TUI_INSTANCE_ID"), "ui-test", endpoints)
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func TestPluginAgentProcessesTwoDaemonsTwoTUIs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Short, explicitly owned paths avoid macOS UNIX socket limits. Never use
	// the user's ANYTTY_DAEMON_SOCKET or the default daemon path.
	dir, err := os.MkdirTemp("/tmp", "anytty-plugin-e2e-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf("id = %q\nversion = \"0.1.0\"\napi = \"anytty.plugin/1\"\n[daemon]\ncommand = [%q, \"-test.run=^TestPluginAgentProcessHelper$\"]\n[tui]\ncommand = [%q, \"-test.run=^TestPluginAgentProcessHelper$\"]\n", agents.PluginID, executable, executable)
	manifest += "[capabilities]\ndaemon = [\"state.read\", \"state.write\", \"events.subscribe\", \"messages.send\"]\ntui = [\"state.read\", \"events.subscribe\", \"ui.mounts\", \"ui.panes.bind\"]\n[[mounts]]\nid = \"agents.workspace\"\nslot = \"sidebar\"\nrenderer = \"declarative\"\n"
	if err = os.WriteFile(filepath.Join(dir, host.ManifestFile), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	installation := host.Installation{ID: agents.PluginID, Directory: dir, Enabled: true}
	sockets := map[string]string{}
	for _, endpoint := range []string{"local", "remote"} {
		socket := filepath.Join(dir, endpoint+".sock")
		sockets[endpoint] = socket
		server := newCoreV2TestServer(corev2.WithSocketPath(socket), corev2.WithPluginRuntime(endpoint, filepath.Join(dir, endpoint+"-state")))
		serverCtx, stop := context.WithCancel(ctx)
		done := make(chan error, 1)
		go func() { done <- server.ListenAndServe(serverCtx) }()
		t.Cleanup(func() { stop(); _ = server.Shutdown(context.Background()); <-done })
		if err = waitForSocket(socket, 2*time.Second, func() error {
			c, err := dialV3Client(socket)
			if err == nil {
				c.Close()
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	open := func(endpoint string) *clientprotocol.ApplicationClient {
		c, err := openPluginDaemon(ctx, sockets[endpoint])
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { c.Close() })
		return c
	}
	launch := func(options host.ProcessOptions) {
		options.Installation = installation
		options.ExtraEnv = append(options.ExtraEnv, "ANYTTY_AGENT_PROCESS_TEST=1")
		options.Log = io.Discard
		childCtx, stop := context.WithCancel(ctx)
		done := make(chan error, 1)
		go func() { done <- host.Run(childCtx, options) }()
		t.Cleanup(func() {
			stop()
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Error("owned plugin process did not exit")
			}
		})
	}
	for _, endpoint := range []string{"local", "remote"} {
		app := open(endpoint)
		launch(host.ProcessOptions{Mode: "daemon", Executor: pluginExecutor(app.ApplicationSession)})
	}
	type uiHost struct {
		clients   map[string]*sdk.Client
		addresses map[string]*apipb.PluginAddress
		peers     map[string]*apipb.PluginAddress
		revisions map[string]map[string]uint64
	}
	uis := []uiHost{}
	for _, tuiID := range []string{"tui-a", "tui-b"} {
		ui := uiHost{clients: map[string]*sdk.Client{}, addresses: map[string]*apipb.PluginAddress{}, peers: map[string]*apipb.PluginAddress{}, revisions: map[string]map[string]uint64{}}
		executors := map[string]sdk.Executor{}
		for _, endpoint := range []string{"local", "remote"} {
			executors[endpoint] = pluginExecutor(open(endpoint).ApplicationSession)
			client := sdk.NewClient(executors[endpoint])
			reg, err := client.Register(ctx, &apipb.PluginRegisterRequest{Address: &apipb.PluginAddress{TuiInstanceId: tuiID, PluginId: agents.PluginID, PluginInstanceId: "host"}})
			if err != nil {
				t.Fatal(err)
			}
			ui.clients[endpoint], ui.addresses[endpoint] = client, reg.Address
			ui.revisions[endpoint] = map[string]uint64{}
		}
		launch(host.ProcessOptions{Mode: "tui", TUIInstanceID: tuiID, ExtraEnv: []string{`ANYTTY_PLUGIN_ENDPOINTS=["local","remote"]`}, RoutedExecutor: func(ctx context.Context, endpoint string, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
			return executors[endpoint](ctx, command)
		}})
		uis = append(uis, ui)
	}
	owner := &apipb.PluginMountOwner{Owner: &apipb.PluginMountOwner_Workspace{Workspace: &apipb.PluginWorkspaceOwner{WorkspaceId: "workspace"}}}

	acknowledgeMounts := func(ui uiHost, endpoint string, batch *apipb.PluginBatch) {
		for _, message := range batch.Messages {
			if update := message.GetMountUpdate(); update != nil {
				if update.ExpectedRevision != ui.revisions[endpoint][update.MountId] {
					t.Fatalf("mount revision mismatch: %v", update)
				}
				ui.revisions[endpoint][update.MountId] = update.Revision
				if message.RequestId == "" {
					t.Fatal("mount update lacks business acknowledgement request")
				}
				if err := ui.clients[endpoint].Send(ctx, &apipb.PluginMessage{Destination: message.Source, Body: &apipb.PluginMessage_Reply{Reply: &apipb.PluginReply{RequestId: message.RequestId}}}); err != nil {
					t.Fatal(err)
				}
			}
		}
	}

	receive := func(ui uiHost, endpoint string, wait time.Duration) (*apipb.PluginBatch, error) {
		batch, err := ui.clients[endpoint].Receive(ctx, wait)
		if err != nil {
			return nil, err
		}
		for _, m := range batch.Messages {
			if init := m.GetInit(); init != nil {
				ui.peers[endpoint] = m.Source
				if m.RequestId != "" {
					if err := ui.clients[endpoint].Send(ctx, &apipb.PluginMessage{Destination: m.Source, Body: &apipb.PluginMessage_Reply{Reply: &apipb.PluginReply{RequestId: m.RequestId}}}); err != nil {
						return nil, err
					}
				}
				if err := ui.clients[endpoint].Send(ctx, &apipb.PluginMessage{Destination: m.Source, Body: &apipb.PluginMessage_Init{Init: &apipb.PluginUiInit{Owners: []*apipb.PluginMountOwner{owner}, Host: ui.addresses[endpoint], MountRevisions: ui.revisions[endpoint], ReconcileRequestId: init.ReconcileRequestId, Context: &apipb.PluginTargetContext{TuiInstanceId: ui.addresses[endpoint].TuiInstanceId, WorkspaceId: "workspace", PaneId: "pane"}}}}); err != nil {
					return nil, err
				}
			}
		}
		acknowledgeMounts(ui, endpoint, batch)
		return batch, nil
	}
	// Complete the host handshake on both endpoints and both independent TUIs.
	for _, ui := range uis {
		for _, endpoint := range []string{"local", "remote"} {
			for {
				batch, err := receive(ui, endpoint, time.Second)
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, m := range batch.Messages {
					if m.GetInit() != nil {

						found = true
					}
				}
				if found {
					break
				}
			}
		}
	}
	t.Setenv("ANYTTY_PLUGIN_STATE_DIR", filepath.Join(dir, "hooks"))
	t.Setenv("ANYTTY_TERMINAL_ID", "terminal-test")
	for _, endpoint := range []string{"local", "remote"} {
		t.Setenv("ANYTTY_DAEMON_SOCKET", sockets[endpoint])
		payload := fmt.Sprintf(`{"hook_event_name":"UserPromptSubmit","session_id":%q,"cwd":"/test"}`, "session-"+endpoint)
		if err := runAgentHook(ctx, strings.NewReader(payload), "codex"); err != nil {
			t.Fatal(err)
		}
	}
	var firstMount *apipb.PluginUiMountUpdate
	// Both TUIs must independently receive a list aggregating both daemons.
	for i, ui := range uis {
		for {
			batch, err := receive(ui, "local", time.Second)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, message := range batch.Messages {
				mount := message.GetMountUpdate()
				if mount == nil || len(mount.GetRoot().GetChildren()) != 2 {
					continue
				}
				daemons := map[string]bool{}
				for _, row := range mount.Root.Children {
					daemons[row.GetTerminal().GetDaemonId()] = true
				}
				if !daemons["local"] || !daemons["remote"] {
					continue
				}
				if message.Destination.TuiInstanceId != fmt.Sprintf("tui-%c", 'a'+i) {
					t.Fatal("cross-TUI delivery")
				}
				if i == 0 {
					firstMount = mount
				}
				found = true
			}
			if found {
				break
			}
		}
	}
	// Match real TUI routing: a remote row starts a new interaction on the
	// remote daemon. The UI process must accept it even if its mount is local.
	var remoteRow *apipb.PluginUiNode
	for _, row := range firstMount.Root.Children {
		if row.GetTerminal().GetDaemonId() == "remote" {
			remoteRow = row
		}
	}
	a := uis[0]
	err = a.clients["remote"].Send(ctx, &apipb.PluginMessage{RequestId: "remote-click", Destination: a.peers["remote"], Body: &apipb.PluginMessage_Interaction{Interaction: &apipb.PluginUiInteraction{Kind: "activate", MountId: firstMount.MountId, MountRevision: firstMount.Revision, ItemId: remoteRow.ItemId, ActionId: "agents.open", Context: &apipb.PluginTargetContext{ContextId: "host-remote-click", TuiInstanceId: "tui-a", WorkspaceId: "workspace", PaneId: "pane-a", BindingRevision: 7}}}})
	if err != nil {
		t.Fatal(err)
	}
	for {
		batch, err := receive(a, "remote", time.Second)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, m := range batch.Messages {
			if bind := m.GetOperation().GetBind(); bind != nil {
				if bind.Terminal.DaemonId != "remote" || m.GetOperation().Context.PaneId != "pane-a" || m.GetOperation().Context.BindingRevision != 7 || m.Destination.TuiInstanceId != "tui-a" {
					t.Fatalf("remote operation lost its target: %v", m)
				}
				found = true
			}
		}
		if found {
			break
		}
	}
	batch, err := receive(uis[1], "remote", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range batch.Messages {
		if m.GetOperation() != nil {
			t.Fatal("click changed other TUI")
		}
	}

}

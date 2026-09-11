package app

import (
	"context"
	"errors"
	"testing"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
	"github.com/anytty/anytty/tui/testkit"
	"google.golang.org/protobuf/proto"
)

type pluginTestService struct {
	messages  []*apipb.PluginMessage
	endpoints []state.EndpointID
	err       error
}

func (s *pluginTestService) Watch(context.Context) (<-chan port.PluginDelivery, error) {
	return nil, s.err
}
func (s *pluginTestService) Send(_ context.Context, ep state.EndpointID, m *apipb.PluginMessage) error {
	s.messages = append(s.messages, proto.Clone(m).(*apipb.PluginMessage))
	s.endpoints = append(s.endpoints, ep)
	return s.err
}
func (s *pluginTestService) ResolveDaemon(d string) (state.EndpointID, bool) {
	switch d {
	case "daemon-a":
		return "local", true
	case "daemon-b":
		return "remote", true
	}
	return "", false
}
func pluginFixture(t *testing.T) (state.Root, PluginDeps, *pluginTestService) {
	t.Helper()
	root := state.Root{Shell: state.DefaultShell(), Viewport: state.ViewportStore{Cols: 100, Rows: 30, Valid: true}}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView("local", state.DefaultPaneID, "old", 1, 80, 24, state.TerminalResizeRoleFollower, "tui-a", state.TerminalPaneViewID(state.DefaultPaneID), false))
	service := &pluginTestService{}
	deps := PluginDeps{Service: service, TUIInstanceID: "tui-a", Live: LiveDeps{Terminal: &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{Channel: 2}}}}
	update := &apipb.PluginUiMountUpdate{MountId: "agents", Owner: &apipb.PluginMountOwner{Owner: &apipb.PluginMountOwner_Workspace{Workspace: &apipb.PluginWorkspaceOwner{WorkspaceId: root.Shell.Workspace.ID}}}, Slot: "sidebar", Title: "Agents", Revision: 1, Focus: true, Root: &apipb.PluginUiNode{Id: "list", Kind: "list", Children: []*apipb.PluginUiNode{{Id: "a", Kind: "row", Text: "Codex", ItemId: "session-a", ActionId: "open", Terminal: &apipb.PluginTerminalRef{DaemonId: "daemon-b", TerminalId: "new"}}, {Id: "b", Kind: "row", Text: "OpenCode", ItemId: "session-b", ActionId: "open"}}}}
	d := port.PluginDelivery{EndpointID: "local", Message: &apipb.PluginMessage{RequestId: "mount", Source: &apipb.PluginAddress{DaemonId: "daemon-a", TuiInstanceId: "tui-a", PluginId: "agents", PluginInstanceId: "process"}, Destination: &apipb.PluginAddress{DaemonId: "daemon-a", TuiInstanceId: "tui-a", PluginId: "agents", PluginInstanceId: "host"}, Body: &apipb.PluginMessage_MountUpdate{MountUpdate: update}}}
	root, _ = NewPluginReducer(deps)(root, PluginDeliveryMsg{d})
	if len(root.Plugins.Mounts) != 1 {
		t.Fatal("mount not registered")
	}
	return root, deps, service
}
func runPluginEffects(t *testing.T, effects []Effect) {
	t.Helper()
	for _, e := range effects {
		if f, ok := e.(FuncEffect); ok {
			f.Run(context.Background())
		}
	}
}
func TestPluginInteractionRequiresDaemonReturnAndPinsTarget(t *testing.T) {
	root, deps, service := pluginFixture(t)
	reducer := NewPluginReducer(deps)
	next, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	b, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if b.TerminalID != "old" || b.AttachPending {
		t.Fatal("local click must not mutate binding before daemon roundtrip")
	}
	runPluginEffects(t, effects)
	if len(service.messages) != 1 || service.endpoints[0] != "remote" {
		t.Fatal("remote row interaction must use owning daemon connection")
	}
	interaction := service.messages[0].GetInteraction()
	if interaction.GetContext().GetPaneId() != state.DefaultPaneID || interaction.GetContext().GetContextId() == "" {
		t.Fatal("missing pinned target context")
	}
	// Switching focus cannot redirect the delayed operation.
	next.Shell = next.Shell.SplitActivePane(state.PaneState{ID: "other", Kind: state.PaneEmpty}, state.SplitDirectionVertical)
	d := port.PluginDelivery{EndpointID: "remote", Message: &apipb.PluginMessage{RequestId: "bind", Source: &apipb.PluginAddress{DaemonId: "daemon-b", PluginId: "agents"}, Destination: &apipb.PluginAddress{TuiInstanceId: "tui-a"}, Body: &apipb.PluginMessage_Operation{Operation: &apipb.PluginUiOperation{Context: interaction.Context, Operation: &apipb.PluginUiOperation_Bind{Bind: &apipb.PluginPaneBind{Terminal: &apipb.PluginTerminalRef{DaemonId: "daemon-b", TerminalId: "new"}}}}}}}
	pending, bindEffects := reducer(next, PluginDeliveryMsg{d})
	target, _ := pending.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !target.AttachPending || target.AttachCandidate.TerminalID != "new" {
		t.Fatal("routed operation must attach original target")
	}
	if len(bindEffects) == 0 {
		t.Fatal("missing attach effect")
	}
	// A malicious changed context cannot target the newly focused panel.
	forged := proto.Clone(d.Message).(*apipb.PluginMessage)
	forged.GetOperation().Context.PaneId = "other"
	_, reject := reducer(next, PluginDeliveryMsg{port.PluginDelivery{EndpointID: "remote", Message: forged}})
	service.messages = nil
	runPluginEffects(t, reject)
	if service.messages[0].GetReply().GetError().GetCode() != "STALE_CONTEXT" {
		t.Fatal("forged target accepted")
	}
}
func TestPluginBindRejectsRaceAndWrongTUI(t *testing.T) {
	root, deps, service := pluginFixture(t)
	reducer := NewPluginReducer(deps)
	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	runPluginEffects(t, effects)
	c := service.messages[0].GetInteraction().Context
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView("local", state.DefaultPaneID, "replacement", 2, 80, 24, state.TerminalResizeRoleFollower, "tui-a", state.TerminalPaneViewID(state.DefaultPaneID), false))
	p := &apipb.PluginMessage{RequestId: "bind", Source: &apipb.PluginAddress{DaemonId: "daemon-b", PluginId: "agents"}, Destination: &apipb.PluginAddress{TuiInstanceId: "tui-a"}, Body: &apipb.PluginMessage_Operation{Operation: &apipb.PluginUiOperation{Context: c, Operation: &apipb.PluginUiOperation_Bind{Bind: &apipb.PluginPaneBind{Terminal: &apipb.PluginTerminalRef{DaemonId: "daemon-b", TerminalId: "new"}}}}}}
	_, effects = reducer(root, PluginDeliveryMsg{port.PluginDelivery{EndpointID: "remote", Message: p}})
	service.messages = nil
	runPluginEffects(t, effects)
	if service.messages[0].GetReply().GetError().GetCode() != "CONFLICT" {
		t.Fatal("stale binding accepted")
	}
	p.Destination.TuiInstanceId = "tui-b"
	_, effects = reducer(root, PluginDeliveryMsg{port.PluginDelivery{EndpointID: "remote", Message: p}})
	if len(effects) != 0 {
		t.Fatal("other TUI operation accepted")
	}
}
func TestPluginKeyboardMouseAndDisconnect(t *testing.T) {
	root, deps, service := pluginFixture(t)
	reducer := NewPluginReducer(deps)
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyDown}})
	if root.Plugins.Mounts["agents"].SelectedID != "b" {
		t.Fatal("keyboard selection failed")
	}
	root, effects := reducer(root, PluginInputMsg{MountID: "agents", NodeID: "a", Event: input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseLeft}})
	runPluginEffects(t, effects)
	if len(service.messages) != 0 {
		t.Fatal("single click should only select")
	}
	root, effects = reducer(root, PluginInputMsg{MountID: "agents", NodeID: "a", Event: input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseLeft}})
	runPluginEffects(t, effects)
	if len(service.messages) != 1 {
		t.Fatal("double click must route same submit as Enter")
	}
	root, _ = reducer(root, PluginDeliveryMsg{port.PluginDelivery{EndpointID: "local", Err: errors.New("offline")}})
	service.messages = nil
	_, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	runPluginEffects(t, effects)
	if len(service.messages) != 0 {
		t.Fatal("stale mount should not execute")
	}
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}})
	if root.Plugins.FocusedMountID != "" {
		t.Fatal("Escape must restore terminal input")
	}
}
func TestPluginInitAndOwnerCloseTravelThroughDaemon(t *testing.T) {
	root, deps, service := pluginFixture(t)
	root.Plugins.Mounts["agents"] = state.PluginMount{} // use a separate tab-scoped mount below
	root.Plugins.Mounts = nil
	d := port.PluginDelivery{EndpointID: "local", Message: &apipb.PluginMessage{Source: &apipb.PluginAddress{DaemonId: "daemon-a", PluginId: "agents"}, Destination: &apipb.PluginAddress{TuiInstanceId: "tui-a"}, Body: &apipb.PluginMessage_Init{Init: &apipb.PluginUiInit{}}}}
	_, effects := NewPluginReducer(deps)(root, PluginDeliveryMsg{d})
	runPluginEffects(t, effects)
	if len(service.messages) != 1 || len(service.messages[0].GetInit().Owners) < 3 {
		t.Fatal("init must route concrete workspace/tab/panel refs")
	}
}

func TestPluginWorkspaceWatchCannotRetargetOtherTUI(t *testing.T) {
	rootA, _, _ := pluginFixture(t)
	rootB := rootA
	rootA.TerminalViews = rootA.TerminalViews.BindPane(state.NewEndpointPaneTerminalView("remote", state.DefaultPaneID, "agent-remote", 8, 80, 24, state.TerminalResizeRoleFollower, "tui-a", state.TerminalPaneViewID(state.DefaultPaneID), false))
	events := make(chan port.WorkbenchStorageEvent, 1)
	storage := &testkit.FakeWorkbenchStorageService{WatchCh: events, LoadResult: port.WorkbenchStorageLoadResult{Snapshot: state.SnapshotRootWorkbenchForStorage(rootA), Found: true, Version: 7}}
	reducer := NewWorkbenchStorageReducer(WorkbenchDeps{Storage: storage, SkipInitialLoad: true})
	_, effects := reducer(rootB, WorkbenchStorageWatchRequestMsg{})
	if len(effects) != 1 {
		t.Fatal("missing storage watch")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	received := make(chan Msg, 1)
	go effects[0].(StreamEffect).Run(ctx, func(m Msg) { received <- m })
	events <- port.WorkbenchStorageEvent{Version: 7, Op: "put", Ref: state.DefaultWorkbenchStorageRef("")}
	message := <-received
	next, pending := reducer(rootB, message)
	binding, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if binding.TerminalID != "old" || binding.EndpointID != "local" || len(pending) != 0 || len(storage.Loads) != 0 || next.WorkbenchSync.AvailableVersion != 7 {
		t.Fatal("storage watch changed other TUI instead of recording available recovery version")
	}
	close(events)
}

func TestPluginMountShortcutConflictAndOwnerCancellation(t *testing.T) {
	root, deps, service := pluginFixture(t)
	m := root.Plugins.Mounts["agents"]
	m.Actions = []*apipb.PluginUiAction{{Id: "open", DefaultKey: "x", Enabled: true, Scope: "workspace"}}
	root.Plugins = root.Plugins.Set(m)
	another := m
	another.ID = "other"
	root.Plugins = root.Plugins.Set(another)
	next, effects, handled := pluginShortcut(root, input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"}, deps)
	runPluginEffects(t, effects)
	if !handled || next.Plugins.Error == "" || len(service.messages) != 0 {
		t.Fatal("same-scope conflict must disable both actions")
	}
	root.Plugins = root.Plugins.Remove("other")
	root, effects = NewPluginReducer(deps)(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	runPluginEffects(t, effects)
	if len(root.Plugins.Contexts) != 1 {
		t.Fatal("missing context")
	}
	m = root.Plugins.Mounts["agents"]
	m.Owner.WorkspaceID = "closed-workspace"
	root.Plugins = root.Plugins.Set(m)
	next, effects = NewPluginMaintenanceReducer(deps)(root, NoopMsg{})
	service.messages = nil
	runPluginEffects(t, effects)
	if len(next.Plugins.Mounts) != 0 || len(next.Plugins.Contexts) != 0 || len(service.messages) != 1 || service.messages[0].GetInteraction().GetKind() != "close" {
		t.Fatal("owner closure must revoke contexts and route close notification")
	}
}

func TestPluginBindCommitAndLateAttachRace(t *testing.T) {
	for _, kind := range []string{"bound", "empty", "race"} {
		t.Run(kind, func(t *testing.T) {
			root, deps, service := pluginFixture(t)
			if kind == "empty" {
				root.TerminalViews = state.TerminalViewStore{}
			}
			reducer := NewPluginReducer(deps)
			root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
			runPluginEffects(t, effects)
			c := service.messages[0].GetInteraction().Context
			d := port.PluginDelivery{EndpointID: "remote", Message: &apipb.PluginMessage{RequestId: "bind", Source: &apipb.PluginAddress{DaemonId: "daemon-b", PluginId: "agents"}, Destination: &apipb.PluginAddress{TuiInstanceId: "tui-a"}, Body: &apipb.PluginMessage_Operation{Operation: &apipb.PluginUiOperation{Context: c, Operation: &apipb.PluginUiOperation_Bind{Bind: &apipb.PluginPaneBind{Terminal: &apipb.PluginTerminalRef{DaemonId: "daemon-b", TerminalId: "new"}}}}}}}
			pending, effects := reducer(root, PluginDeliveryMsg{d})
			if len(effects) != 1 {
				t.Fatalf("expected attach, got %d", len(effects))
			}
			result := effects[0].(FuncEffect).Run(context.Background())
			if _, ok := result.(PluginBindResultMsg); !ok {
				t.Fatalf("unexpected %T", result)
			}
			if kind == "race" {
				pending.TerminalViews = pending.TerminalViews.BindPane(state.NewEndpointPaneTerminalView("local", state.DefaultPaneID, "replacement", 2, 80, 24, state.TerminalResizeRoleFollower, "tui-a", state.TerminalPaneViewID(state.DefaultPaneID), false))
			}
			next, effects := reducer(pending, result)
			binding, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
			if kind == "race" {
				if binding.TerminalID != "replacement" {
					t.Fatal("late attach replaced newer binding")
				}
			} else if binding.TerminalID != "new" || binding.EndpointID != "remote" || !binding.Attached {
				t.Fatalf("routed bind did not commit: %+v", binding)
			}
		})
	}
}

func TestPluginShortcutUserOverrideDisableAndGlobalOptIn(t *testing.T) {
	root, deps, service := pluginFixture(t)
	m := root.Plugins.Mounts["agents"]
	m.Actions = []*apipb.PluginUiAction{{Id: "attention", DefaultKey: "a", Enabled: true, Scope: "global"}}
	root.Plugins = root.Plugins.Set(m)
	event := input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "a"}
	if _, _, handled := pluginShortcut(root, event, deps); handled {
		t.Fatal("default global shortcut must require user opt-in")
	}
	root.Config.PluginShortcuts = map[string]string{"agents/attention": "x"}
	root.Plugins.FocusedMountID = ""
	if _, _, handled := pluginShortcut(root, event, deps); handled {
		t.Fatal("default key must be replaced")
	}
	_, effects, handled := pluginShortcut(root, input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"}, deps)
	runPluginEffects(t, effects)
	if !handled || len(service.messages) != 1 || service.messages[0].GetInteraction().GetActionId() != "attention" {
		t.Fatal("configured global action did not route")
	}
	root.Config.PluginShortcuts["agents/attention"] = ""
	if _, _, handled := pluginShortcut(root, input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"}, deps); handled {
		t.Fatal("empty override must disable binding")
	}
}
func TestPluginMountRejectsUnsupportedComponentAndBadIDs(t *testing.T) {
	for _, root := range []*apipb.PluginUiNode{{Id: "unknown", Kind: "unknown"}, {Id: "pty", Kind: "pty"}, {Id: "root", Kind: "list", Children: []*apipb.PluginUiNode{{Id: "duplicate", Kind: "text"}, {Id: "duplicate", Kind: "text"}}}} {
		if err := validatePluginMount(&apipb.PluginUiMountUpdate{Root: root}); err == nil {
			t.Fatalf("unsupported mount accepted: %v", root)
		}
	}
}

func TestPluginDefaultKeyboardFocusEntryAndEscape(t *testing.T) {
	root, deps, _ := pluginFixture(t)
	root.Plugins.FocusedMountID = ""
	m := root.Plugins.Mounts["agents"]
	m.ID = "other"
	root.Plugins = root.Plugins.Set(m)
	reducer := ComposeReducers(NewPluginReducer(deps), NewShellReducer(), NewUIInputReducer())
	for _, event := range []input.InputEvent{{Kind: input.EventKindKey, Key: input.KeyChar, Char: "g", Ctrl: true}, {Kind: input.EventKindKey, Key: input.KeyChar, Char: "a"}} {
		queue := []Msg{InputMsg{Event: event}}
		for len(queue) > 0 {
			msg := queue[0]
			queue = queue[1:]
			var effects []Effect
			root, effects = reducer(root, msg)
			queue = append(queue, runDefaultActionEffects(effects)...)
		}
	}
	if root.Plugins.FocusedMountID != "agents" || root.Shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("Ctrl+G a must focus first plugin without mouse: focus=%q mode=%q error=%q", root.Plugins.FocusedMountID, root.Shell.InteractionMode, root.Plugins.Error)
	}
	root = pluginFocusNext(root, 1)
	if root.Plugins.FocusedMountID != "other" {
		t.Fatal("next mount navigation failed")
	}
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}})
	if root.Plugins.FocusedMountID != "" {
		t.Fatal("Escape failed to restore terminal")
	}
}
func TestPluginScopedFailureKeepsOtherPluginOnline(t *testing.T) {
	root, deps, _ := pluginFixture(t)
	m := root.Plugins.Mounts["agents"]
	m.ID = "other"
	m.PluginID = "other"
	root.Plugins = root.Plugins.Set(m)
	root, _ = NewPluginReducer(deps)(root, PluginDeliveryMsg{port.PluginDelivery{EndpointID: "local", PluginID: "agents", Err: errors.New("plugin exited")}})
	if !root.Plugins.Mounts["agents"].Stale || root.Plugins.Mounts["other"].Stale {
		t.Fatal("plugin failure crossed namespace boundary")
	}
}

func TestPluginInitReconcilesOnlyRequestingPluginDaemonMounts(t *testing.T) {
	root, deps, service := pluginFixture(t)
	owned := root.Plugins.Mounts["agents"]
	owned.Revision, owned.Stale = 7, true
	root.Plugins = root.Plugins.Set(owned)
	other := owned
	other.ID, other.PluginID = "other-plugin", "other"
	root.Plugins = root.Plugins.Set(other)
	remote := owned
	remote.ID, remote.DaemonID, remote.EndpointID = "other-daemon", "daemon-b", "remote"
	root.Plugins = root.Plugins.Set(remote)
	source := &apipb.PluginAddress{DaemonId: "daemon-a", TuiInstanceId: "tui-a", PluginId: "agents", PluginInstanceId: "new-process", RegistrationEpoch: 9}
	host := &apipb.PluginAddress{DaemonId: "daemon-a", TuiInstanceId: "tui-a", PluginId: "agents", PluginInstanceId: "host", RegistrationEpoch: 10}
	delivery := port.PluginDelivery{EndpointID: "local", Message: &apipb.PluginMessage{RequestId: "reconcile-9", Source: source, Destination: host, Body: &apipb.PluginMessage_Init{Init: &apipb.PluginUiInit{ReconcileRequestId: "nonce-9"}}}}
	next, effects := NewPluginReducer(deps)(root, PluginDeliveryMsg{delivery})
	runPluginEffects(t, effects)
	if len(service.messages) != 2 || service.endpoints[0] != "local" || !proto.Equal(service.messages[0].Destination, source) {
		t.Fatal("reconciliation must return through the requesting daemon to its exact source")
	}
	if service.messages[0].GetInit().GetReconcileRequestId() != "nonce-9" || service.messages[1].GetReply().GetRequestId() != "reconcile-9" {
		t.Fatal("Init must echo the reconciliation nonce and acknowledge the original request")
	}
	revisions := service.messages[0].GetInit().GetMountRevisions()
	if len(revisions) != 1 || revisions["agents"] != 7 {
		t.Fatalf("expected authoritative own stale mount only, got %v", revisions)
	}
	if !next.Plugins.Mounts["agents"].Stale || next.Plugins.Mounts["agents"].Revision != 7 {
		t.Fatal("Init must neither revive nor modify the existing mount")
	}
	empty := pluginInitForPeer(root, deps.TUIInstanceID, host, &apipb.PluginAddress{DaemonId: "daemon-c", PluginId: "agents"})
	if len(empty.GetMountRevisions()) != 0 {
		t.Fatal("a daemon without mounts must reconcile to zero, not another daemon's revision")
	}
}

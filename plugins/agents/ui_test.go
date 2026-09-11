package agents

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/anytty/anytty/plugin/sdk"
	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

func TestMultiEndpointTUIUsesInteractionDaemonAndPreservesRemoteIdentity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	queues := map[string]chan *apipb.PluginMessage{"local": make(chan *apipb.PluginMessage, 10), "remote": make(chan *apipb.PluginMessage, 10)}
	type sent struct {
		endpoint string
		message  *apipb.PluginMessage
	}
	sends := make(chan sent, 100)
	owner := &apipb.PluginMountOwner{Owner: &apipb.PluginMountOwner_Workspace{Workspace: &apipb.PluginWorkspaceOwner{WorkspaceId: "w"}}}
	go sdk.ServeStdioRouted(ctx, right, right, func(ctx context.Context, endpoint string, cmd *apipb.PluginCommand) (*apipb.PluginResult, error) {
		host := &apipb.PluginAddress{DaemonId: endpoint, TuiInstanceId: "tui-a", PluginId: PluginID, PluginInstanceId: "host", RegistrationEpoch: 1}
		switch {
		case cmd.GetRegister() != nil:
			address := proto.Clone(cmd.GetRegister().Address).(*apipb.PluginAddress)
			address.DaemonId = endpoint
			address.RegistrationEpoch = 1
			return &apipb.PluginResult{Result: &apipb.PluginResult_Registration{Registration: &apipb.PluginRegistration{Address: address, SourceLease: []byte(endpoint), Peers: []*apipb.PluginAddress{host}}}}, nil
		case cmd.GetState() != nil:
			r := (&Event{Agent: "codex", SessionID: "same-session", TerminalID: "same-terminal", Epoch: 1, Sequence: 1, Status: "working"}).Proto(endpoint, time.Now())
			data, _ := proto.Marshal(&apipb.PluginAgentSnapshot{Agents: []*apipb.PluginAgentReport{r}, Revision: 1})
			return &apipb.PluginResult{Result: &apipb.PluginResult_State{State: &apipb.PluginStateSnapshot{Collection: Collection, Revision: 1, Value: &apipb.PluginPayload{Schema: SnapshotSchema, Version: 1, Data: data}}}}, nil
		case cmd.GetReceive() != nil:
			select {
			case message := <-queues[endpoint]:
				return &apipb.PluginResult{Result: &apipb.PluginResult_Batch{Batch: &apipb.PluginBatch{Messages: []*apipb.PluginMessage{message}}}}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		case cmd.GetSend() != nil:
			message := cmd.GetSend().Message
			if init := message.GetInit(); init != nil {
				queues[endpoint] <- &apipb.PluginMessage{Source: host, Body: &apipb.PluginMessage_Init{Init: &apipb.PluginUiInit{Host: host, Owners: []*apipb.PluginMountOwner{owner}, ReconcileRequestId: init.ReconcileRequestId}}}
			} else {
				if mount := message.GetMountUpdate(); mount != nil && len(mount.Root.Children) < 2 {
					queues[endpoint] <- &apipb.PluginMessage{Source: host, Body: &apipb.PluginMessage_Reply{Reply: &apipb.PluginReply{RequestId: message.RequestId}}}
				}
				// Withhold the final two-row mount's business ACK: a remote
				// click can arrive on its own daemon route before that ACK.
				select {
				case sends <- sent{endpoint, message}:
				case <-ctx.Done():
				}
			}
		}
		return &apipb.PluginResult{Result: &apipb.PluginResult_Ack{Ack: &apipb.PluginAck{Delivered: 1}}}, nil
	})
	client := sdk.NewStdioClient(left, left)
	defer client.Close()
	done := make(chan error, 1)
	go func() { done <- RunTUI(ctx, client, "tui-a", "plugin-process", []string{"local", "remote"}) }()
	var control string
	var mount *apipb.PluginUiMountUpdate
	for mount == nil {
		select {
		case outbound := <-sends:
			if update := outbound.message.GetMountUpdate(); update != nil && len(update.Root.Children) == 2 {
				mount = update
				control = outbound.endpoint
			}
		case <-ctx.Done():
			t.Fatal("two daemon feeds not aggregated")
		}
	}
	if mount.Root.Children[0].ItemId == mount.Root.Children[1].ItemId {
		t.Fatal("same named remote resources collided")
	}
	var remote *apipb.PluginUiNode
	for _, node := range mount.Root.Children {
		if node.Terminal.DaemonId != control {
			remote = node
		}
	}
	contextRef := &apipb.PluginTargetContext{ContextId: "fixed-click", TuiInstanceId: "tui-a", WorkspaceId: "w", TabId: "tab", PaneId: "pane-before-focus-change", BindingRevision: 7}
	interactionEndpoint := remote.Terminal.DaemonId
	host := &apipb.PluginAddress{DaemonId: interactionEndpoint, TuiInstanceId: "tui-a", PluginId: PluginID, PluginInstanceId: "host", RegistrationEpoch: 1}
	// A feed may change this session's terminal before the old visible row is
	// clicked. Bind the exact displayed snapshot, not that unseen replacement.
	replacement := (&Event{Agent: "codex", SessionID: "same-session", TerminalID: "unseen-replacement", Epoch: 1, Sequence: 2, Status: "working"}).Proto(interactionEndpoint, time.Now())
	replacementData, _ := proto.Marshal(&apipb.PluginAgentSnapshot{Agents: []*apipb.PluginAgentReport{replacement}, Revision: 2})
	queues[interactionEndpoint] <- &apipb.PluginMessage{Source: host, Body: &apipb.PluginMessage_StateChanged{StateChanged: &apipb.PluginStateSnapshot{Collection: Collection, Revision: 2, Value: &apipb.PluginPayload{Schema: SnapshotSchema, Version: 1, Data: replacementData}}}}
	queues[interactionEndpoint] <- &apipb.PluginMessage{Source: host, Body: &apipb.PluginMessage_Interaction{Interaction: &apipb.PluginUiInteraction{MountId: mount.MountId, MountRevision: mount.Revision, ActionId: "agents.open", ItemId: remote.ItemId, Context: contextRef}}}
	for {
		select {
		case outbound := <-sends:
			if op := outbound.message.GetOperation(); op != nil {
				if outbound.endpoint != interactionEndpoint || op.GetBind().Terminal.DaemonId != interactionEndpoint || op.GetBind().Terminal.TerminalId != "same-terminal" || !proto.Equal(op.Context, contextRef) {
					t.Fatal("click route or original target changed")
				}
				cancel()
				return
			}
		case <-ctx.Done():
			t.Fatal("click did not travel over selected daemon SDK")
		}
	}
}

func TestAgentDisplayTitleUsesSummaryInsteadOfPath(t *testing.T) {
	report := &apipb.PluginAgentReport{Title: "Review API", Cwd: "/Users/example/projects/anytty", Provider: "codex", SessionId: "session-1"}
	if got := agentDisplayTitle(report); got != "Review API" {
		t.Fatalf("report title should win, got %q", got)
	}
	report.Title = ""
	if got := agentDisplayTitle(report); got != "anytty" {
		t.Fatalf("project summary should replace full path, got %q", got)
	}
	report.Cwd = ""
	if got := agentDisplayTitle(report); got != "codex · session-1" {
		t.Fatalf("provider/session fallback is unstable, got %q", got)
	}
}

func TestFullStateClearsCoalescedPermissionDelta(t *testing.T) {
	s := &Store{}
	e := Event{Agent: "opencode", SessionID: "s", TerminalID: "t", Epoch: 1, Sequence: 1, Kind: "status", Status: "working", FullState: true, PendingPermissions: []string{"p1", "p2"}}
	if r, _, err := s.Apply(e, time.Now()); err != nil || r.Status != "blocked" {
		t.Fatal(r, err)
	}
	e.Sequence = 500
	e.PendingPermissions = nil
	e.Status = "idle"
	if r, _, err := s.Apply(e, time.Now()); err != nil || r.Status != "idle" {
		t.Fatal("coalesced complete state must clear answered permissions", r, err)
	}
}

func TestRestoredAgentIsStaleUntilFreshHook(t *testing.T) {
	s := &Store{}
	event := Event{Agent: "codex", SessionID: "s", TerminalID: "t", Epoch: 1, Sequence: 2, Status: "working"}
	if _, _, err := s.Apply(event, time.Now()); err != nil {
		t.Fatal(err)
	}
	restored := &Store{}
	if err := restored.Restore(s.ProtoSnapshot("d")); err != nil {
		t.Fatal(err)
	}
	if !restored.MarkRestoredStale() {
		t.Fatal("restored working status claimed fresh")
	}
	if restored.MarkRestoredStale() {
		t.Fatal("stale transition must be idempotent")
	}
	snapshot := restored.ProtoSnapshot("d")
	if !snapshot.Agents[0].Stale || snapshot.Agents[0].State != "working" {
		t.Fatal("stale must preserve underlying observation")
	}
	if _, changed, _ := restored.Apply(event, time.Now()); changed {
		t.Fatal("old hook cleared stale flag")
	}
	event.Sequence++
	if r, changed, err := restored.Apply(event, time.Now()); err != nil || !changed || r.Stale {
		t.Fatal("fresh report did not clear stale", r, err)
	}
}

package agents

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/plugin/sdk"
	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

// The host applied revision 1 but its business ACK was lost. After a feed
// overflow/re-registration, only the host's authoritative revision can recover.
func TestTUIMountLostACKReconcilesAndRejectsOldInit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	queue := make(chan *apipb.PluginBatch, 20)
	applied := make(chan *apipb.PluginUiMountUpdate, 10)
	host := &apipb.PluginAddress{DaemonId: "daemon", TuiInstanceId: "tui", PluginId: PluginID, PluginInstanceId: "host", RegistrationEpoch: 9}
	owner := &apipb.PluginMountOwner{Owner: &apipb.PluginMountOwner_Workspace{Workspace: &apipb.PluginWorkspaceOwner{WorkspaceId: "workspace"}}}
	var mu sync.Mutex
	var mountedRevision, registration uint64
	firstNonce := ""
	stateVersion := uint64(1)
	stateAt := func(version uint64) *apipb.PluginStateSnapshot {
		title := "first"
		if version == 2 {
			title = "recovered"
		}
		if version == 3 {
			title = "after-old-init"
		}
		report := (&Event{Agent: "codex", SessionID: "s", TerminalID: "t", Epoch: 1, Sequence: version, Status: "working", Title: title}).Proto("daemon", time.Now())
		data, _ := proto.Marshal(&apipb.PluginAgentSnapshot{Revision: version, Agents: []*apipb.PluginAgentReport{report}})
		return &apipb.PluginStateSnapshot{Collection: Collection, Revision: version, Value: &apipb.PluginPayload{Schema: SnapshotSchema, Version: 1, Data: data}}
	}
	enqueue := func(message *apipb.PluginMessage) {
		queue <- &apipb.PluginBatch{Messages: []*apipb.PluginMessage{message}}
	}
	go sdk.ServeStdioRouted(ctx, right, right, func(ctx context.Context, _ string, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
		if command.GetReceive() != nil {
			select {
			case batch := <-queue:
				return &apipb.PluginResult{Result: &apipb.PluginResult_Batch{Batch: batch}}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		mu.Lock()
		defer mu.Unlock()
		if request := command.GetRegister(); request != nil {
			registration++
			if registration == 2 {
				// A restarted daemon may publish a smaller epoch. Trust the
				// newly authenticated registration's peer, not numeric ordering.
				host = proto.Clone(host).(*apipb.PluginAddress)
				host.RegistrationEpoch = 1
			}
			address := proto.Clone(request.Address).(*apipb.PluginAddress)
			address.DaemonId = "daemon"
			address.RegistrationEpoch = registration
			return &apipb.PluginResult{Result: &apipb.PluginResult_Registration{Registration: &apipb.PluginRegistration{Address: address, SourceLease: []byte("fixture"), Peers: []*apipb.PluginAddress{host}}}}, nil
		}
		if command.GetState() != nil {
			return &apipb.PluginResult{Result: &apipb.PluginResult_State{State: stateAt(stateVersion)}}, nil
		}
		if message := command.GetSend().GetMessage(); message != nil {
			if init := message.GetInit(); init != nil {
				if firstNonce == "" && init.ReconcileRequestId != "" {
					firstNonce = init.ReconcileRequestId
				}
				enqueue(&apipb.PluginMessage{Source: host, Body: &apipb.PluginMessage_Init{Init: &apipb.PluginUiInit{Host: host, Owners: []*apipb.PluginMountOwner{owner}, ReconcileRequestId: init.ReconcileRequestId, MountRevisions: map[string]uint64{MountID(owner): mountedRevision}}}})
			}
			if update := message.GetMountUpdate(); update != nil {
				if update.Revision == 2 && registration < 2 {
					t.Error("new snapshot escaped before business ACK or authoritative resync")
				}
				if update.ExpectedRevision != mountedRevision {
					t.Errorf("host revision=%d, plugin expected=%d", mountedRevision, update.ExpectedRevision)
				}
				mountedRevision = update.Revision
				applied <- update
				if mountedRevision == 1 {
					// No business ACK. A newer state waits behind the single flight;
					// then the receiving queue overflows and forces re-registration.
					stateVersion = 2
					enqueue(&apipb.PluginMessage{Source: host, Body: &apipb.PluginMessage_StateChanged{StateChanged: stateAt(2)}})
					queue <- &apipb.PluginBatch{ResyncRequired: true}
				} else {
					enqueue(&apipb.PluginMessage{Source: host, Body: &apipb.PluginMessage_Reply{Reply: &apipb.PluginReply{RequestId: message.RequestId}}})
					if mountedRevision == 2 {
						// An old Init must not roll back the now-applied revision.
						enqueue(&apipb.PluginMessage{Source: host, Body: &apipb.PluginMessage_Init{Init: &apipb.PluginUiInit{Host: host, Owners: []*apipb.PluginMountOwner{owner}, ReconcileRequestId: firstNonce, MountRevisions: map[string]uint64{MountID(owner): 0}}}})
						stateVersion = 3
						enqueue(&apipb.PluginMessage{Source: host, Body: &apipb.PluginMessage_StateChanged{StateChanged: stateAt(3)}})
					}
				}
			}
		}
		return &apipb.PluginResult{Result: &apipb.PluginResult_Ack{Ack: &apipb.PluginAck{Delivered: 1}}}, nil
	})
	client := sdk.NewStdioClient(left, left)
	defer client.Close()
	go RunTUI(ctx, client, "tui", "process", []string{"local"})
	for expected := uint64(1); expected <= 3; expected++ {
		select {
		case update := <-applied:
			if update.Revision != expected || update.ExpectedRevision != expected-1 {
				t.Fatalf("bad mount recovery sequence %+v", update)
			}
			if expected == 2 && !strings.Contains(update.Root.Children[0].Text, "recovered") {
				t.Fatal("latest pending feed snapshot lost during recovery")
			}
			if expected == 3 && !strings.Contains(update.Root.Children[0].Text, "after-old-init") {
				t.Fatal("late Init disturbed latest state")
			}
		case <-ctx.Done():
			t.Fatal("mounts did not recover after lost ACK")
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if registration < 2 {
		t.Fatal("test did not exercise route re-registration")
	}
}

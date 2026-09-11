package agents

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anytty/anytty/plugin/sdk"
	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

type feedUpdate struct {
	endpoint string
	client   *sdk.Client
	state    *apipb.PluginStateSnapshot
	message  *apipb.PluginMessage
	host     *apipb.PluginAddress
	stale    bool
}
type tuiMount struct {
	owner         *apipb.PluginMountOwner
	revision      uint64
	attentionOnly bool
	dirty         bool
	flight        *mountFlight
	snapshots     map[uint64]map[string]*apipb.PluginTerminalRef
}
type mountFlight struct {
	requestID string
	revision  uint64
	expires   time.Time
}

// RunTUI aggregates independently authenticated daemon feeds in one process.
// Rendering uses a fixed control daemon; clicks return through the endpoint
// on which the host minted their interaction context. Resource references
// continue to identify the owning daemon, even for another endpoint's Agent.
func RunTUI(ctx context.Context, root *sdk.Client, tuiID, instanceID string, endpoints []string) error {
	if tuiID == "" || len(endpoints) == 0 {
		return errors.New("TUI instance and endpoints required")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	updates := make(chan feedUpdate, 128)
	for _, endpoint := range endpoints {
		go runFeed(ctx, root.ForEndpoint(endpoint), endpoint, tuiID, instanceID, updates)
	}
	feeds := map[string][]Entry{}
	mounts := map[string]*tuiMount{}
	var control *sdk.Client
	var host *apipb.PluginAddress
	controlEndpoint := ""
	syncing := true
	reconcileID := ""
	var reconcileDeadline time.Time
	var requestSequence uint64
	nextRequest := func(kind string) string {
		requestSequence++
		return fmt.Sprintf("%s-%s-%d", kind, instanceID, requestSequence)
	}
	markDirty := func() {
		for _, mount := range mounts {
			mount.dirty = true
		}
	}
	type pendingAction struct {
		message  *apipb.PluginMessage
		client   *sdk.Client
		endpoint string
		expires  time.Time
	}
	pendingActions := map[string]pendingAction{}
	cleanupTicker := time.NewTicker(time.Second)
	defer cleanupTicker.Stop()
	replyTo := func(client *sdk.Client, message *apipb.PluginMessage, failure *apipb.PluginError) {
		if message.RequestId == "" {
			return
		}
		_ = client.Send(ctx, &apipb.PluginMessage{RequestId: "reply-" + message.RequestId, Destination: message.Source, Body: &apipb.PluginMessage_Reply{Reply: &apipb.PluginReply{RequestId: message.RequestId, Error: failure}}})
	}
	requestSync := func() error {
		if control == nil || host == nil {
			return nil
		}
		syncing = true
		reconcileID = nextRequest("init")
		reconcileDeadline = time.Now().Add(5 * time.Second)
		for _, mount := range mounts {
			mount.flight = nil
			mount.dirty = true
		}
		return control.Send(ctx, &apipb.PluginMessage{RequestId: reconcileID, DeadlineUnixMillis: reconcileDeadline.UnixMilli(), Destination: host, Body: &apipb.PluginMessage_Init{Init: &apipb.PluginUiInit{ReconcileRequestId: reconcileID}}})
	}
	publish := func() error {
		if control == nil || host == nil || syncing {
			return nil
		}
		all := []Entry{}
		for _, entries := range feeds {
			all = append(all, entries...)
		}
		for id, mount := range mounts {
			if !mount.dirty || mount.flight != nil {
				continue
			}
			update := BuildMount(all, mount.owner, id, mount.revision+1, mount.attentionOnly)
			requestID := nextRequest("mount")
			expires := time.Now().Add(5 * time.Second)
			if err := control.Send(ctx, &apipb.PluginMessage{RequestId: requestID, DeadlineUnixMillis: expires.UnixMilli(), Destination: host, Body: &apipb.PluginMessage_MountUpdate{MountUpdate: update}}); err != nil {
				return err
			}
			if mount.snapshots == nil {
				mount.snapshots = map[uint64]map[string]*apipb.PluginTerminalRef{}
			}
			for revision := range mount.snapshots {
				if revision != mount.revision {
					delete(mount.snapshots, revision)
				}
			}
			mount.snapshots[update.Revision] = mountItems(update.Root)
			mount.flight = &mountFlight{requestID: requestID, revision: update.Revision, expires: expires}
			mount.dirty = false
		}
		return nil
	}
	publishLatest := func() {
		if err := publish(); err != nil {
			_ = requestSync()
		}
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-cleanupTicker.C:
			if syncing {
				if !reconcileDeadline.IsZero() && now.After(reconcileDeadline) {
					_ = requestSync()
				}
			} else {
				for _, mount := range mounts {
					if mount.flight != nil && now.After(mount.flight.expires) {
						_ = requestSync()
						break
					}
				}
			}
			for id, action := range pendingActions {
				if now.After(action.expires) {
					replyTo(action.client, action.message, &apipb.PluginError{Code: "DEADLINE_EXCEEDED", Message: "Agent action timed out"})
					delete(pendingActions, id)
				}
			}
		case update := <-updates:
			if update.stale {
				if update.endpoint == controlEndpoint {
					syncing = true
					reconcileID = ""
					reconcileDeadline = time.Time{}
					for _, mount := range mounts {
						mount.flight = nil
					}
				}
				for i := range feeds[update.endpoint] {
					feeds[update.endpoint][i].Stale = true
				}
				markDirty()
				publishLatest()
				continue
			}
			if update.state != nil {
				state := update.state
				if state.Value == nil || len(state.Value.Data) == 0 {
					feeds[update.endpoint] = nil
				} else {
					if state.Value.Schema != SnapshotSchema || state.Value.Version != 1 {
						continue
					}
					var snapshot apipb.PluginAgentSnapshot
					if proto.Unmarshal(state.Value.Data, &snapshot) != nil {
						continue
					}
					entries := make([]Entry, 0, len(snapshot.Agents))
					for _, report := range snapshot.Agents {
						if report.Terminal != nil {
							entries = append(entries, Entry{EndpointID: update.endpoint, Report: report, Stale: report.Stale})
						}
					}
					feeds[update.endpoint] = entries
				}
				markDirty()
				publishLatest()
			}
			message := update.message
			if message == nil {
				continue
			}
			if init := message.GetInit(); init != nil {
				if init.Host == nil || !proto.Equal(init.Host, message.Source) {
					continue
				}
				if !proto.Equal(message.Source, update.host) {
					// Refresh the published peer instead of accepting a stale queued host
					// address or guessing whether epochs increase across daemon restarts.
					_ = update.client.Unregister(ctx)
					continue
				}
				replyTo(update.client, message, nil)
				if controlEndpoint != "" && controlEndpoint != update.endpoint {
					continue
				}
				if control == nil || host == nil || !proto.Equal(host, init.Host) {
					controlEndpoint = update.endpoint
					control = update.client
					host = init.Host
					_ = requestSync()
					continue
				}
				if init.ReconcileRequestId == "" {
					_ = requestSync()
					continue
				}
				if !syncing || init.ReconcileRequestId != reconcileID {
					continue
				}
				syncing = false
				reconcileID = ""
				reconcileDeadline = time.Time{}
				live := map[string]bool{}
				for _, owner := range init.Owners {
					if owner.GetWorkspace() == nil && owner.GetPanel() == nil {
						continue
					}
					id := MountID(owner)
					live[id] = true
					mount := mounts[id]
					if mount == nil {
						mount = &tuiMount{owner: owner}
						mounts[id] = mount
					}
					mount.revision = init.MountRevisions[id]
					mount.flight = nil
					mount.dirty = true
				}
				for id := range mounts {
					if !live[id] {
						delete(mounts, id)
					}
				}
				publishLatest()
				continue
			}
			if reply := message.GetReply(); reply != nil {
				if update.endpoint == controlEndpoint && proto.Equal(message.Source, host) {
					for _, mount := range mounts {
						if mount.flight != nil && mount.flight.requestID == reply.RequestId {
							if reply.Error != nil {
								mount.flight = nil
								mount.dirty = true
								_ = requestSync()
							} else {
								mount.revision = mount.flight.revision
								mount.flight = nil
								publishLatest()
							}
							break
						}
					}
				}

				if original, ok := pendingActions[reply.RequestId]; ok && update.endpoint == original.endpoint {
					replyTo(original.client, original.message, reply.Error)
					delete(pendingActions, reply.RequestId)
				}
				continue
			}
			interaction := message.GetInteraction()
			if interaction == nil || control == nil || message.Source.GetTuiInstanceId() != tuiID || message.Source.GetPluginInstanceId() != "host" || !proto.Equal(message.Source, update.host) {
				continue
			}
			mount := mounts[interaction.MountId]
			if interaction.Kind == "close" {
				delete(mounts, interaction.MountId)
				replyTo(update.client, message, nil)
				continue
			}
			if syncing || mount == nil || interaction.Context == nil || interaction.MountRevision != mount.revision && (mount.flight == nil || interaction.MountRevision != mount.flight.revision) {
				replyTo(update.client, message, &apipb.PluginError{Code: "STALE_CONTEXT", Message: "Agent list changed; select the current row again"})
				continue
			}
			switch interaction.ActionId {
			case "agents.filter":
				mount.attentionOnly = !mount.attentionOnly
				markDirty()
				publishLatest()
				replyTo(update.client, message, nil)
			case "agents.open":
				target := mount.snapshots[interaction.MountRevision][interaction.ItemId]
				if target == nil {
					replyTo(update.client, message, &apipb.PluginError{Code: "UNAVAILABLE", Message: "Agent terminal is no longer available"})
					continue
				}
				requestID := fmt.Sprintf("bind-%s-%d", instanceID, time.Now().UnixNano())
				if err := update.client.Send(ctx, &apipb.PluginMessage{RequestId: requestID, DeadlineUnixMillis: time.Now().Add(25 * time.Second).UnixMilli(), Destination: message.Source, Body: &apipb.PluginMessage_Operation{Operation: &apipb.PluginUiOperation{Context: interaction.Context, Operation: &apipb.PluginUiOperation_Bind{Bind: &apipb.PluginPaneBind{Terminal: target}}}}}); err != nil {
					replyTo(update.client, message, &apipb.PluginError{Code: "UNAVAILABLE", Message: err.Error()})
				} else {
					pendingActions[requestID] = pendingAction{message: message, client: update.client, endpoint: update.endpoint, expires: time.Now().Add(25 * time.Second)}
				}
			default:
				replyTo(update.client, message, &apipb.PluginError{Code: "UNSUPPORTED", Message: "Unknown Agent action"})
			}
		}
	}
}

func runFeed(ctx context.Context, client *sdk.Client, endpoint, tuiID, instanceID string, updates chan<- feedUpdate) {
	var expectedHost *apipb.PluginAddress
	emit := func(update feedUpdate) bool {
		update.endpoint = endpoint
		update.client = client
		update.host = expectedHost
		select {
		case updates <- update:
			return true
		case <-ctx.Done():
			return false
		}
	}
	for ctx.Err() == nil {
		err := func() error {
			reg, err := client.Register(ctx, &apipb.PluginRegisterRequest{Address: &apipb.PluginAddress{TuiInstanceId: tuiID, PluginId: PluginID, PluginInstanceId: instanceID}})
			if err != nil {
				return err
			}
			defer func() {
				cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
				defer cancel()
				_ = client.Unregister(cleanup)
			}()
			expectedHost = nil
			for _, peer := range reg.Peers {
				if peer.PluginInstanceId == "host" && peer.TuiInstanceId == tuiID {
					expectedHost = peer
					break
				}
			}
			if expectedHost == nil {
				return errors.New("TUI host peer unavailable")
			}
			state, err := client.State(ctx, &apipb.PluginStateRequest{Collection: Collection, Operation: &apipb.PluginStateRequest_Watch{Watch: &apipb.PluginStateWatch{}}})
			if err != nil {
				return err
			}
			if !emit(feedUpdate{state: state}) {
				return ctx.Err()
			}
			if err = client.Send(ctx, &apipb.PluginMessage{Destination: expectedHost, Body: &apipb.PluginMessage_Init{Init: &apipb.PluginUiInit{}}}); err != nil {
				return err
			}
			for {
				batch, err := client.Receive(ctx, 20*time.Second)
				if err != nil {
					return err
				}
				if batch.ResyncRequired {
					return errors.New("agent feed requires resync")
				}
				for _, message := range batch.Messages {
					if !emit(feedUpdate{state: message.GetStateChanged(), message: message}) {
						return ctx.Err()
					}
				}
			}
		}()
		if err != nil {
			if !emit(feedUpdate{stale: true}) {
				return
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

// Capture the exact selectable resource references in a displayed revision.
// A newer feed may rebind an Agent session before an older visible row is clicked.
func mountItems(root *apipb.PluginUiNode) map[string]*apipb.PluginTerminalRef {
	items := map[string]*apipb.PluginTerminalRef{}
	var visit func(*apipb.PluginUiNode)
	visit = func(node *apipb.PluginUiNode) {
		if node == nil {
			return
		}
		if node.ItemId != "" && node.Terminal != nil && !node.Disabled {
			items[node.ItemId] = proto.Clone(node.Terminal).(*apipb.PluginTerminalRef)
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(root)
	return items
}

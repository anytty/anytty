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

const Collection = "agents"
const SnapshotSchema = "org.anytty.agents.snapshot"

// RunDaemon is an independent plugin service. Its only state and communication
// access is through the public SDK; stdout belongs exclusively to protobuf IPC.
func RunDaemon(ctx context.Context, client *sdk.Client, instanceID string) error {
	reg, err := client.Register(ctx, &apipb.PluginRegisterRequest{Address: &apipb.PluginAddress{PluginId: PluginID, PluginInstanceId: instanceID}, DaemonService: true})
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		_ = client.Unregister(cleanup)
	}()
	state, err := client.State(ctx, &apipb.PluginStateRequest{Collection: Collection, Operation: &apipb.PluginStateRequest_Get{Get: &apipb.PluginStateGet{}}})
	if err != nil {
		return err
	}
	store := &Store{}
	if value := state.Value; value != nil && len(value.Data) > 0 {
		if value.Schema != SnapshotSchema || value.Version != 1 {
			return errors.New("unsupported stored agents schema")
		}
		var snapshot apipb.PluginAgentSnapshot
		if err = proto.Unmarshal(value.Data, &snapshot); err != nil {
			return err
		}
		if err = store.Restore(&snapshot); err != nil {
			return err
		}
		if store.MarkRestoredStale() {
			data, marshalErr := proto.Marshal(store.ProtoSnapshot(reg.Address.DaemonId))
			if marshalErr != nil {
				return marshalErr
			}
			state, err = client.State(ctx, &apipb.PluginStateRequest{Collection: Collection, Operation: &apipb.PluginStateRequest_Put{Put: &apipb.PluginStatePut{ExpectedRevision: state.Revision, Value: &apipb.PluginPayload{Schema: SnapshotSchema, Version: 1, Data: data}}}})
			if err != nil {
				return err
			}
		}
	}
	for {
		batch, err := client.Receive(ctx, 20*time.Second)
		if err != nil {
			return err
		}
		if batch.ResyncRequired {
			return errors.New("agent service input overflow; restart to restore durable state")
		}
		for _, message := range batch.Messages {
			report := message.GetAgentReport()
			if report == nil {
				continue
			}
			var failure *apipb.PluginError
			if report.GetTerminal().GetDaemonId() != reg.Address.DaemonId {
				failure = &apipb.PluginError{Code: "INVALID_ARGUMENT", Message: "agent terminal belongs to another daemon"}
			} else {
				e, eventErr := EventFromProto(report)
				if eventErr == nil {
					before := store.ProtoSnapshot(reg.Address.DaemonId)
					_, changed, applyErr := store.Apply(e, time.Now())
					eventErr = applyErr
					if eventErr == nil && changed {
						data, marshalErr := proto.Marshal(store.ProtoSnapshot(reg.Address.DaemonId))
						eventErr = marshalErr
						if eventErr == nil {
							next, putErr := client.State(ctx, &apipb.PluginStateRequest{Collection: Collection, Operation: &apipb.PluginStateRequest_Put{Put: &apipb.PluginStatePut{ExpectedRevision: state.Revision, Value: &apipb.PluginPayload{Schema: SnapshotSchema, Version: 1, Data: data}}}})
							eventErr = putErr
							if putErr == nil {
								state = next
							}
						}
						if eventErr != nil {
							_ = store.Restore(before)
						}
					}
				}
				if eventErr != nil {
					failure = &apipb.PluginError{Code: "REPORT_REJECTED", Message: eventErr.Error()}
				}
			}
			if message.RequestId != "" {
				reply := &apipb.PluginMessage{RequestId: fmt.Sprintf("reply-%s", message.RequestId), Destination: message.Source, Body: &apipb.PluginMessage_Reply{Reply: &apipb.PluginReply{RequestId: message.RequestId, Error: failure}}}
				// The reporting hook may have hit its nonblocking deadline and
				// disconnected. Its durable state must survive that lost reply.
				_ = client.Send(ctx, reply)
				if ctx.Err() != nil {
					return ctx.Err()
				}
			}
		}
	}
}

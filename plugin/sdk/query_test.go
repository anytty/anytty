package sdk

import (
	"context"
	"testing"

	"github.com/anytty/anytty/proto/apipb"
)

func TestRequestUISnapshotAlwaysSendsTypedQueryAndCorrelatesReply(t *testing.T) {
	var sent *apipb.PluginMessage
	client := NewClient(func(_ context.Context, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
		if command.GetRegister() != nil {
			return &apipb.PluginResult{Result: &apipb.PluginResult_Registration{Registration: &apipb.PluginRegistration{Address: &apipb.PluginAddress{DaemonId: "remote", TuiInstanceId: "tui", PluginId: "reader", PluginInstanceId: "process"}, SourceLease: []byte("private-lease")}}}, nil
		}
		if send := command.GetSend(); send != nil {
			sent = send.Message
			if string(send.SourceLease) != "private-lease" {
				t.Fatal("missing source lease")
			}
			return &apipb.PluginResult{Result: &apipb.PluginResult_Ack{Ack: &apipb.PluginAck{Delivered: 1}}}, nil
		}
		t.Fatal("query started competing Receive loop")
		return nil, nil
	})
	if _, err := client.Register(context.Background(), &apipb.PluginRegisterRequest{}); err != nil {
		t.Fatal(err)
	}
	host := &apipb.PluginAddress{DaemonId: "remote", TuiInstanceId: "tui", PluginId: "reader", PluginInstanceId: "host", RegistrationEpoch: 9}
	id, err := client.RequestUISnapshot(context.Background(), host)
	if err != nil {
		t.Fatal(err)
	}
	if sent.GetUiQuery().GetSnapshot() == nil || sent.RequestId != id || sent.Destination.RegistrationEpoch != 9 || sent.DeadlineUnixMillis == 0 {
		t.Fatal("incomplete typed query")
	}
	response := &apipb.PluginMessage{Body: &apipb.PluginMessage_Reply{Reply: &apipb.PluginReply{RequestId: id, UiSnapshot: &apipb.PluginUiSnapshot{TuiInstanceId: "tui", Revision: 12}}}}
	if _, _, matched := UISnapshotResult(response, "other"); matched {
		t.Fatal("unrelated reply consumed")
	}
	snapshot, err, matched := UISnapshotResult(response, id)
	if err != nil || !matched || snapshot.Revision != 12 {
		t.Fatal(snapshot, err)
	}
	snapshot.TuiInstanceId = "mutated"
	if response.GetReply().UiSnapshot.TuiInstanceId != "tui" {
		t.Fatal("SDK leaked shared message pointer")
	}
	response.GetReply().Error = &apipb.PluginError{Code: "PERMISSION_DENIED", Message: "ui.read required"}
	if _, err, matched := UISnapshotResult(response, id); err == nil || !matched {
		t.Fatal("query failure hidden")
	}
}

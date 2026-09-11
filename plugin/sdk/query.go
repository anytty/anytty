package sdk

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

// RequestUISnapshot sends a typed read request over this client's daemon route.
// The caller's existing Receive loop handles its reply via UISnapshotResult;
// this method never starts a competing receiver that could steal other events.
func (c *Client) RequestUISnapshot(ctx context.Context, host *apipb.PluginAddress) (string, error) {
	if host == nil || host.DaemonId == "" || host.TuiInstanceId == "" || host.PluginId == "" || host.PluginInstanceId == "" || host.RegistrationEpoch == 0 {
		return "", errors.New("explicit registered TUI host required")
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	requestID := hex.EncodeToString(id[:])
	deadline := time.Now().Add(30 * time.Second)
	if value, ok := ctx.Deadline(); ok && value.Before(deadline) {
		deadline = value
	}
	message := &apipb.PluginMessage{RequestId: requestID, Destination: proto.Clone(host).(*apipb.PluginAddress), DeadlineUnixMillis: deadline.UnixMilli(), Body: &apipb.PluginMessage_UiQuery{UiQuery: &apipb.PluginUiQuery{Query: &apipb.PluginUiQuery_Snapshot{Snapshot: &apipb.PluginUiSnapshotQuery{}}}}}
	if err := c.Send(ctx, message); err != nil {
		return "", err
	}
	return requestID, nil
}

// UISnapshotResult matches a daemon-validated correlated reply. A descriptive
// active_context in this read result is not permission to mutate the UI.
func UISnapshotResult(message *apipb.PluginMessage, requestID string) (*apipb.PluginUiSnapshot, error, bool) {
	reply := message.GetReply()
	if requestID == "" || reply == nil || reply.RequestId != requestID {
		return nil, nil, false
	}
	if failure := reply.Error; failure != nil {
		return nil, &Error{Code: failure.Code, Message: failure.Message}, true
	}
	if reply.UiSnapshot == nil {
		return nil, errors.New("UI snapshot reply missing typed snapshot"), true
	}
	return proto.Clone(reply.UiSnapshot).(*apipb.PluginUiSnapshot), nil, true
}

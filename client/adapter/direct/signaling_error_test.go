package direct_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/anytty/anytty/client/adapter/direct"
	"github.com/anytty/anytty/internal/protocol/directsignal"
	"github.com/anytty/anytty/proto/remoteauthpb"
)

func TestTCPSignalingClientPreservesOverloadedErrorCode(t *testing.T) {
	serverConnection, clientConnection := net.Pipe()
	serverDone := make(chan error, 1)
	go func() {
		defer serverConnection.Close()
		request := &remoteauthpb.DirectSignalingRequestV2{}
		if err := directsignal.ReadMessage(serverConnection, request); err != nil {
			serverDone <- err
			return
		}
		serverDone <- directsignal.WriteMessage(serverConnection, &remoteauthpb.DirectSignalingResponseV2{
			Payload: &remoteauthpb.DirectSignalingResponseV2_Error{Error: &remoteauthpb.DirectSignalingErrorV2{
				Code: remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_OVERLOADED, Message: "direct signaling server is overloaded",
			}},
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := (direct.TCPSignalingClient{Dialer: directSingleConnectionDialer{connection: clientConnection}}).Exchange(ctx, []string{"direct.test:1"}, &remoteauthpb.DirectSignalingRequestV2{RequestId: "overloaded-client-test"})
	var signalingError *direct.SignalingError
	if !errors.As(err, &signalingError) {
		t.Fatalf("Exchange error = %v, want *direct.SignalingError", err)
	}
	if signalingError.Code != remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_OVERLOADED || signalingError.Message != "direct signaling server is overloaded" {
		t.Fatalf("signaling error = %#v", signalingError)
	}
	if serverErr := <-serverDone; serverErr != nil {
		t.Fatal(serverErr)
	}
}

type directSingleConnectionDialer struct {
	connection net.Conn
}

func (dialer directSingleConnectionDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	return dialer.connection, nil
}

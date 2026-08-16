package direct_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/anytty/anytty/client/adapter/direct"
	clientruntime "github.com/anytty/anytty/client/runtime"
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

func TestTCPSignalingClientRacesAddressesBeforeSendingRequest(t *testing.T) {
	serverConnection, clientConnection := net.Pipe()
	slowStarted := make(chan struct{})
	slowCanceled := make(chan struct{})
	dialer := directRacingDialer{connection: clientConnection, slowStarted: slowStarted, slowCanceled: slowCanceled}
	serverDone := make(chan error, 1)
	go func() {
		defer serverConnection.Close()
		request := &remoteauthpb.DirectSignalingRequestV2{}
		if err := directsignal.ReadMessage(serverConnection, request); err != nil {
			serverDone <- err
			return
		}
		if request.GetRequestId() != "race-addresses" {
			serverDone <- errors.New("unexpected request id")
			return
		}
		serverDone <- directsignal.WriteMessage(serverConnection, &remoteauthpb.DirectSignalingResponseV2{
			Payload: &remoteauthpb.DirectSignalingResponseV2_Answer{Answer: &remoteauthpb.DirectSignalingAnswerV2{AnswerSdp: "answer"}},
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	answer, err := (direct.TCPSignalingClient{Dialer: dialer}).Exchange(ctx, []string{"stale.test:1", "reachable.test:2"}, &remoteauthpb.DirectSignalingRequestV2{RequestId: "race-addresses"})
	if err != nil || answer.GetAnswerSdp() != "answer" {
		t.Fatalf("Exchange answer = %#v, err = %v", answer, err)
	}
	if serverErr := <-serverDone; serverErr != nil {
		t.Fatal(serverErr)
	}
	select {
	case <-slowCanceled:
	case <-time.After(time.Second):
		t.Fatal("losing address dial was not canceled")
	}
}

func TestTCPSignalingClientClassifiesUnreachableAddressesAsRetryable(t *testing.T) {
	cause := errors.New("connection refused")
	_, err := (direct.TCPSignalingClient{Dialer: directFailingDialer{cause: cause}}).Exchange(
		context.Background(),
		[]string{"first.test:1", "second.test:2"},
		&remoteauthpb.DirectSignalingRequestV2{RequestId: "unreachable-addresses"},
	)
	var runtimeErr *clientruntime.Error
	if !errors.As(err, &runtimeErr) || runtimeErr.Code != clientruntime.ErrorUnavailable || !runtimeErr.Retryable || !runtimeErr.Attempted {
		t.Fatalf("Exchange error = %#v, want attempted retryable unavailable", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("Exchange error = %v, want wrapped dial cause", err)
	}
}

type directSingleConnectionDialer struct {
	connection net.Conn
}

func (dialer directSingleConnectionDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	return dialer.connection, nil
}

type directRacingDialer struct {
	connection   net.Conn
	slowStarted  chan struct{}
	slowCanceled chan struct{}
}

type directFailingDialer struct{ cause error }

func (dialer directFailingDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	return nil, dialer.cause
}

func (dialer directRacingDialer) DialContext(ctx context.Context, _, address string) (net.Conn, error) {
	switch address {
	case "stale.test:1":
		close(dialer.slowStarted)
		<-ctx.Done()
		close(dialer.slowCanceled)
		return nil, ctx.Err()
	case "reachable.test:2":
		<-dialer.slowStarted
		return dialer.connection, nil
	default:
		return nil, errors.New("unexpected address")
	}
}

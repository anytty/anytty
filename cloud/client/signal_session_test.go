package client

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	cloudprotocol "github.com/anytty/anytty/cloud/protocol"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type signalSessionTestStream struct {
	cloudv1.ClientGateway_ConnectClient
	sent []*cloudv1.ClientSignal
	recv chan signalSessionTestRecv
}

type signalSessionTestRecv struct {
	signal *cloudv1.EdgeSignal
	err    error
}

func (stream *signalSessionTestStream) Recv() (*cloudv1.EdgeSignal, error) {
	result := <-stream.recv
	return result.signal, result.err
}

func (*signalSessionTestStream) CloseSend() error { return nil }

func (stream *signalSessionTestStream) Send(signal *cloudv1.ClientSignal) error {
	stream.sent = append(stream.sent, signal)
	return nil
}

func TestSignalSessionConfirmsSelectedPathOnce(t *testing.T) {
	stream := &signalSessionTestStream{recv: make(chan signalSessionTestRecv, 1)}
	session := &SignalSession{stream: stream, senderID: "client", bootID: "boot", sessionID: "session"}
	if err := session.ConfirmPath(cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_DIRECT); err != nil {
		t.Fatal(err)
	}
	if err := session.ConfirmPath(cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_RELAY); err != nil {
		t.Fatal(err)
	}
	if len(stream.sent) != 1 || stream.sent[0].GetStreamSeq() != 3 || stream.sent[0].GetPathSelected().GetPath() != cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_DIRECT {
		t.Fatalf("path confirmation = %#v", stream.sent)
	}
}

func TestSignalSessionLocalClosePublishesDone(t *testing.T) {
	stream := &signalSessionTestStream{recv: make(chan signalSessionTestRecv)}
	session := &SignalSession{stream: stream, done: make(chan struct{})}

	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-session.Done():
	default:
		t.Fatal("local signaling close did not finish the session")
	}
}

func TestSignalSessionPublishesRemoteStreamTermination(t *testing.T) {
	stream := &signalSessionTestStream{recv: make(chan signalSessionTestRecv, 1)}
	session := &SignalSession{stream: stream, done: make(chan struct{})}
	go session.watch()
	stream.recv <- signalSessionTestRecv{err: io.EOF}
	select {
	case <-session.Done():
	case <-time.After(time.Second):
		t.Fatal("remote signaling close did not finish the session")
	}
	if err := session.Err(); err != nil {
		t.Fatalf("clean stream close error = %v", err)
	}

	failure := context.DeadlineExceeded
	stream = &signalSessionTestStream{recv: make(chan signalSessionTestRecv, 1)}
	session = &SignalSession{stream: stream, done: make(chan struct{})}
	go session.watch()
	stream.recv <- signalSessionTestRecv{err: failure}
	<-session.Done()
	if !errors.Is(session.Err(), failure) {
		t.Fatalf("stream error = %v, want %v", session.Err(), failure)
	}
}

func TestSignalSessionPreservesAdministrativeDisconnect(t *testing.T) {
	stream := &signalSessionTestStream{recv: make(chan signalSessionTestRecv, 1)}
	session := &SignalSession{stream: stream, edgeID: "edge", edgeBootID: "edge-boot", sessionID: "session", done: make(chan struct{})}
	go session.watch()
	stream.recv <- signalSessionTestRecv{signal: administrativeCloseSignal("maintenance")}
	<-session.Done()
	if !IsAdminDisconnect(session.Err()) || session.Err().Error() != "maintenance" {
		t.Fatalf("administrative close error = %v", session.Err())
	}
}

func TestAdministrativeCloseIsTypedBeforeReadyAndBeforeAnswer(t *testing.T) {
	for _, stage := range []string{"before_ready", "before_answer"} {
		t.Run(stage, func(t *testing.T) {
			err, closed := signalSessionCloseError(administrativeCloseSignal("policy update"), "edge", "edge-boot", "session")
			if !closed || !IsAdminDisconnect(err) || err.Error() != "policy update" {
				t.Fatalf("administrative close = (%v, %t)", err, closed)
			}
		})
	}
}

func TestAdministrativeCloseRequiresAuthenticatedEnvelope(t *testing.T) {
	signal := administrativeCloseSignal("maintenance")
	signal.BootId = "another-edge-boot"
	err, closed := signalSessionCloseError(signal, "edge", "edge-boot", "session")
	if !closed || err == nil || IsAdminDisconnect(err) {
		t.Fatalf("unauthenticated close = (%v, %t)", err, closed)
	}
}

func administrativeCloseSignal(message string) *cloudv1.EdgeSignal {
	return &cloudv1.EdgeSignal{
		ProtocolVersion: cloudprotocol.ClientGatewayVersion,
		MessageId:       "close-message",
		SenderId:        "edge",
		BootId:          "edge-boot",
		ConnectionId:    "session",
		StreamSeq:       4,
		SentAt:          timestamppb.Now(),
		Payload: &cloudv1.EdgeSignal_Closed{Closed: &cloudv1.SignalSessionClosed{
			SessionId: "session",
			Code:      cloudv1.SignalSessionCloseCode_SIGNAL_SESSION_CLOSE_CODE_ADMIN_DISCONNECT,
			Message:   message,
		}},
	}
}

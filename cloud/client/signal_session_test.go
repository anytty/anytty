package client

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	cloudprotocol "github.com/anytty/anytty/cloud/protocol"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type signalSessionTestStream struct {
	cloudv1.ClientGateway_ConnectClient
	sentMu    sync.Mutex
	sent      []*cloudv1.ClientSignal
	sentCh    chan *cloudv1.ClientSignal
	recv      chan signalSessionTestRecv
	send      func(*cloudv1.ClientSignal) error
	closeSend func() error
}

type signalSessionTestRecv struct {
	signal *cloudv1.EdgeSignal
	err    error
}

func (stream *signalSessionTestStream) Recv() (*cloudv1.EdgeSignal, error) {
	result := <-stream.recv
	return result.signal, result.err
}

func (stream *signalSessionTestStream) CloseSend() error {
	if stream.closeSend != nil {
		return stream.closeSend()
	}
	return nil
}

func (stream *signalSessionTestStream) Send(signal *cloudv1.ClientSignal) error {
	clone := signal
	stream.sentMu.Lock()
	stream.sent = append(stream.sent, clone)
	stream.sentMu.Unlock()
	if stream.send != nil {
		return stream.send(clone)
	}
	if stream.sentCh != nil {
		stream.sentCh <- clone
	}
	return nil
}

func (stream *signalSessionTestStream) sentSignals() []*cloudv1.ClientSignal {
	stream.sentMu.Lock()
	defer stream.sentMu.Unlock()
	return append([]*cloudv1.ClientSignal(nil), stream.sent...)
}

func newSignalSessionTestHarness() (*SignalSession, *signalSessionTestStream) {
	stream := &signalSessionTestStream{recv: make(chan signalSessionTestRecv, 4), sentCh: make(chan *cloudv1.ClientSignal, 8)}
	session := &SignalSession{
		stream: stream, senderID: "client", bootID: "client-boot", edgeID: "edge", edgeBootID: "edge-boot",
		sessionID: "session", done: make(chan struct{}),
	}
	go session.watch()
	return session, stream
}

func TestSignalSessionConfirmPathWaitsForMatchingAck(t *testing.T) {
	session, stream := newSignalSessionTestHarness()
	result := make(chan error, 1)
	go func() {
		result <- session.ConfirmPath(context.Background(), cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_DIRECT)
	}()
	decision := <-stream.sentCh
	if decision.GetStreamSeq() != 3 || decision.GetPathDecision().GetDecision() != cloudv1.CloudPathDecision_CLOUD_PATH_DECISION_CONFIRM_DIRECT {
		t.Fatalf("path decision = %#v", decision)
	}
	select {
	case err := <-result:
		t.Fatalf("ConfirmPath returned before ACK: %v", err)
	default:
	}
	stream.recv <- signalSessionTestRecv{signal: pathDecisionAckSignal(decision, 4)}
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if !session.PathConfirmed() {
		t.Fatal("confirmed path was not published")
	}
	if err := session.ConfirmPath(context.Background(), cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_DIRECT); err != nil {
		t.Fatalf("idempotent confirmation: %v", err)
	}
	if got := len(stream.sentSignals()); got != 1 {
		t.Fatalf("idempotent confirmation sent %d decisions, want 1", got)
	}
	if err := session.ConfirmPath(context.Background(), cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_RELAY); err == nil {
		t.Fatal("different confirmation unexpectedly replaced committed decision")
	}
	_ = session.Close()
}

func TestSignalSessionRetriesStableDecisionUntilAck(t *testing.T) {
	session, stream := newSignalSessionTestHarness()
	result := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		result <- session.ConfirmPath(ctx, cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_RELAY)
	}()
	first := <-stream.sentCh
	second := <-stream.sentCh
	if first.GetPathDecision().GetDecisionId() == "" || first.GetPathDecision().GetDecisionId() != second.GetPathDecision().GetDecisionId() {
		t.Fatalf("retry decision IDs = %q, %q", first.GetPathDecision().GetDecisionId(), second.GetPathDecision().GetDecisionId())
	}
	if first.GetStreamSeq() != 3 || second.GetStreamSeq() != 4 {
		t.Fatalf("retry sequences = %d, %d", first.GetStreamSeq(), second.GetStreamSeq())
	}
	stream.recv <- signalSessionTestRecv{signal: pathDecisionAckSignal(first, 4)}
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	_ = session.Close()
}

func TestSignalSessionRejectsMismatchedDecisionAck(t *testing.T) {
	session, stream := newSignalSessionTestHarness()
	result := make(chan error, 1)
	go func() {
		result <- session.ConfirmPath(context.Background(), cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_DIRECT)
	}()
	decision := <-stream.sentCh
	ack := pathDecisionAckSignal(decision, 4)
	ack.GetPathDecisionAck().DecisionId = "different-decision"
	stream.recv <- signalSessionTestRecv{signal: ack}
	if err := <-result; err == nil {
		t.Fatal("mismatched ACK unexpectedly confirmed the path")
	}
}

func TestSignalSessionEOFDoesNotAcknowledgeDecision(t *testing.T) {
	session, stream := newSignalSessionTestHarness()
	result := make(chan error, 1)
	go func() {
		result <- session.ConfirmPath(context.Background(), cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_DIRECT)
	}()
	<-stream.sentCh
	stream.recv <- signalSessionTestRecv{err: io.EOF}
	if err := <-result; err == nil {
		t.Fatal("EOF unexpectedly confirmed the path")
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

func TestSignalSessionCloseCancelsBlockedSendBeforeCloseSendExactlyOnce(t *testing.T) {
	sendStarted := make(chan struct{})
	streamCanceled := make(chan struct{})
	var cancelOnce sync.Once
	var closeSendMu sync.Mutex
	closeSendCalls := 0
	stream := &signalSessionTestStream{recv: make(chan signalSessionTestRecv)}
	stream.send = func(*cloudv1.ClientSignal) error {
		close(sendStarted)
		<-streamCanceled
		return status.Error(codes.Canceled, "stream context canceled")
	}
	stream.closeSend = func() error {
		closeSendMu.Lock()
		closeSendCalls++
		closeSendMu.Unlock()
		return status.Error(codes.Canceled, "stream context canceled")
	}
	session := &SignalSession{
		stream: stream, senderID: "client", bootID: "client-boot", sessionID: "session",
		done:   make(chan struct{}),
		cancel: func() { cancelOnce.Do(func() { close(streamCanceled) }) },
	}
	decision := &signalPathDecision{
		id: "decision", decision: cloudv1.CloudPathDecision_CLOUD_PATH_DECISION_ABANDON,
		ack: make(chan struct{}),
	}
	sendResult := make(chan error, 1)
	go func() { sendResult <- session.sendPathDecision(decision) }()
	<-sendStarted

	firstClose := make(chan error, 1)
	secondClose := make(chan error, 1)
	go func() { firstClose <- session.Close() }()
	go func() { secondClose <- session.Close() }()

	for index, result := range []<-chan error{firstClose, secondClose} {
		select {
		case err := <-result:
			if err != nil {
				t.Fatalf("Close call %d returned active-cancel error: %v", index+1, err)
			}
		case <-time.After(time.Second):
			t.Fatalf("Close call %d remained blocked behind Send", index+1)
		}
	}
	if err := <-sendResult; status.Code(err) != codes.Canceled {
		t.Fatalf("blocked Send error = %v, want canceled", err)
	}
	closeSendMu.Lock()
	gotCloseSendCalls := closeSendCalls
	closeSendMu.Unlock()
	if gotCloseSendCalls != 1 {
		t.Fatalf("CloseSend calls = %d, want 1", gotCloseSendCalls)
	}
	select {
	case <-session.Done():
	default:
		t.Fatal("bounded active close did not publish Done")
	}
}

func TestSignalSessionAbandonAndWaitRequiresAckBeforeLocalClose(t *testing.T) {
	session, stream := newSignalSessionTestHarness()
	closeSent := make(chan struct{})
	stream.closeSend = func() error { close(closeSent); return nil }
	closed := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		closed <- session.AbandonAndWait(ctx)
	}()
	decision := <-stream.sentCh
	if decision.GetPathDecision().GetDecision() != cloudv1.CloudPathDecision_CLOUD_PATH_DECISION_ABANDON {
		t.Fatalf("close decision = %v", decision.GetPathDecision().GetDecision())
	}
	select {
	case <-closeSent:
		t.Fatal("CloseSend ran before cleanup ACK")
	default:
	}
	stream.recv <- signalSessionTestRecv{signal: pathDecisionAckSignal(decision, 4)}
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	select {
	case <-closeSent:
	default:
		t.Fatal("CloseSend did not run after cleanup ACK")
	}
}

func TestSignalSessionReleaseAndWaitRequiresConfirmedPathAndCleanupAck(t *testing.T) {
	unconfirmed, _ := newSignalSessionTestHarness()
	if err := unconfirmed.ReleaseAndWait(context.Background()); err == nil {
		t.Fatal("unconfirmed session was released")
	}
	_ = unconfirmed.Close()

	session, stream := newSignalSessionTestHarness()
	confirmed := make(chan error, 1)
	go func() {
		confirmed <- session.ConfirmPath(context.Background(), cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_RELAY)
	}()
	decision := <-stream.sentCh
	stream.recv <- signalSessionTestRecv{signal: pathDecisionAckSignal(decision, 4)}
	if err := <-confirmed; err != nil {
		t.Fatal(err)
	}

	closeSent := make(chan struct{})
	stream.closeSend = func() error { close(closeSent); return nil }
	released := make(chan error, 1)
	go func() {
		released <- session.ReleaseAndWait(context.Background())
	}()
	release := <-stream.sentCh
	if release.GetStreamSeq() != 4 || release.GetSessionRelease().GetReleaseId() == "" {
		t.Fatalf("session release = %#v", release)
	}
	select {
	case <-closeSent:
		t.Fatal("ReleaseAndWait closed locally before release ACK")
	default:
	}
	stream.recv <- signalSessionTestRecv{signal: sessionReleaseAckSignal(release, 5)}
	if err := <-released; err != nil {
		t.Fatal(err)
	}
	select {
	case <-closeSent:
	default:
		t.Fatal("ReleaseAndWait did not close locally after release ACK")
	}
	if err := session.ReleaseAndWait(context.Background()); err != nil {
		t.Fatalf("idempotent ReleaseAndWait: %v", err)
	}
	if got := len(stream.sentSignals()); got != 2 {
		t.Fatalf("confirmed session emitted %d frames, want one decision and exactly one release", got)
	}
}

func TestSuccessfulExchangeScopeTransfersSignalStreamOwnership(t *testing.T) {
	attemptContext, cancelAttempt := context.WithCancel(context.Background())
	streamContext, owner := newSignalStreamOwner(attemptContext)
	if err := owner.Retain(); err != nil {
		t.Fatal(err)
	}

	cancelAttempt() // route-racer cleanup after Dial returns
	select {
	case <-streamContext.Done():
		t.Fatal("successful Dial return left signaling owned by the attempt context")
	case <-time.After(10 * time.Millisecond):
	}

	owner.Close() // physical transport close
	select {
	case <-streamContext.Done():
	case <-time.After(time.Second):
		t.Fatal("physical transport close did not release signaling ownership")
	}
}

func TestSignalKeepaliveCoversIdleActiveStream(t *testing.T) {
	parameters := signalClientKeepaliveParameters()
	if parameters.Time != 15*time.Second || parameters.Timeout != 5*time.Second || parameters.PermitWithoutStream {
		t.Fatalf("signal keepalive parameters = %#v", parameters)
	}
}

func TestSignalStreamEndCausePreservesEOFAndGRPCFailureEvidence(t *testing.T) {
	if got := signalStreamEndCause(io.EOF); got != "eof" {
		t.Fatalf("EOF cause = %q", got)
	}
	if got := signalStreamEndCause(context.Canceled); got != "context_canceled" {
		t.Fatalf("canceled cause = %q", got)
	}
	if got := signalStreamEndCause(context.DeadlineExceeded); got != "deadline_exceeded" {
		t.Fatalf("deadline cause = %q", got)
	}
	if got := signalStreamEndCause(errors.New("received prior GOAWAY")); got != "grpc_error" {
		t.Fatalf("GOAWAY cause = %q", got)
	}
}

func TestSignalSessionEOFDoesNotAcknowledgeRelease(t *testing.T) {
	session, stream := newSignalSessionTestHarness()
	confirmed := make(chan error, 1)
	go func() {
		confirmed <- session.ConfirmPath(context.Background(), cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_DIRECT)
	}()
	decision := <-stream.sentCh
	stream.recv <- signalSessionTestRecv{signal: pathDecisionAckSignal(decision, 4)}
	if err := <-confirmed; err != nil {
		t.Fatal(err)
	}

	released := make(chan error, 1)
	go func() { released <- session.ReleaseAndWait(context.Background()) }()
	<-stream.sentCh
	stream.recv <- signalSessionTestRecv{err: io.EOF}
	if err := <-released; err == nil {
		t.Fatal("EOF unexpectedly acknowledged session release")
	}
}

func TestSignalSessionRetriesStableReleaseUntilAck(t *testing.T) {
	session, stream := newSignalSessionTestHarness()
	confirmed := make(chan error, 1)
	go func() {
		confirmed <- session.ConfirmPath(context.Background(), cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_RELAY)
	}()
	decision := <-stream.sentCh
	stream.recv <- signalSessionTestRecv{signal: pathDecisionAckSignal(decision, 4)}
	if err := <-confirmed; err != nil {
		t.Fatal(err)
	}

	released := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		released <- session.ReleaseAndWait(ctx)
	}()
	first := <-stream.sentCh
	second := <-stream.sentCh
	if first.GetSessionRelease().GetReleaseId() == "" || first.GetSessionRelease().GetReleaseId() != second.GetSessionRelease().GetReleaseId() {
		t.Fatalf("release retry IDs = %q, %q", first.GetSessionRelease().GetReleaseId(), second.GetSessionRelease().GetReleaseId())
	}
	if first.GetStreamSeq() != 4 || second.GetStreamSeq() != 5 {
		t.Fatalf("release retry sequences = %d, %d", first.GetStreamSeq(), second.GetStreamSeq())
	}
	stream.recv <- signalSessionTestRecv{signal: sessionReleaseAckSignal(first, 5)}
	if err := <-released; err != nil {
		t.Fatal(err)
	}
}

func TestSignalSessionCloseAndWaitTimesOut(t *testing.T) {
	recv := make(chan signalSessionTestRecv, 1)
	stream := &signalSessionTestStream{recv: recv, sentCh: make(chan *cloudv1.ClientSignal, 2)}
	session := &SignalSession{
		stream: stream, senderID: "client", bootID: "client-boot", edgeID: "edge", edgeBootID: "edge-boot",
		sessionID: "session", done: make(chan struct{}),
	}
	var cancelOnce sync.Once
	session.cancel = func() { cancelOnce.Do(func() { recv <- signalSessionTestRecv{err: context.Canceled} }) }
	go session.watch()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := session.CloseAndWait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseAndWait error = %v, want deadline exceeded", err)
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

func pathDecisionAckSignal(decision *cloudv1.ClientSignal, sequence uint64) *cloudv1.EdgeSignal {
	return &cloudv1.EdgeSignal{
		ProtocolVersion: cloudprotocol.ClientGatewayVersion,
		MessageId:       "ack-message",
		SenderId:        "edge",
		BootId:          "edge-boot",
		ConnectionId:    "session",
		StreamSeq:       sequence,
		SentAt:          timestamppb.Now(),
		Payload: &cloudv1.EdgeSignal_PathDecisionAck{PathDecisionAck: &cloudv1.EdgePathDecisionAck{
			SessionId:  decision.GetPathDecision().GetSessionId(),
			DecisionId: decision.GetPathDecision().GetDecisionId(),
			Decision:   decision.GetPathDecision().GetDecision(),
		}},
	}
}

func sessionReleaseAckSignal(release *cloudv1.ClientSignal, sequence uint64) *cloudv1.EdgeSignal {
	return &cloudv1.EdgeSignal{
		ProtocolVersion: cloudprotocol.ClientGatewayVersion,
		MessageId:       "release-ack-message",
		SenderId:        "edge",
		BootId:          "edge-boot",
		ConnectionId:    "session",
		StreamSeq:       sequence,
		SentAt:          timestamppb.Now(),
		Payload: &cloudv1.EdgeSignal_SessionReleaseAck{SessionReleaseAck: &cloudv1.EdgeSessionReleaseAck{
			SessionId: release.GetSessionRelease().GetSessionId(),
			ReleaseId: release.GetSessionRelease().GetReleaseId(),
		}},
	}
}

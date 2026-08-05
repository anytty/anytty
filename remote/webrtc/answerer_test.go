package webrtc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anytty/anytty/shared/transport"
	pion "github.com/pion/webrtc/v4"
)

func TestAnswererHandsReliableChannelToAuthorizedHandler(t *testing.T) {
	handler := &recordingAuthorizedHandler{called: make(chan struct{}), result: make(chan error)}
	sessionStarted := make(chan struct{}, 1)
	sessionErrors := make(chan error, 1)
	answerer := Answerer{
		Handler:        handler,
		OnSessionStart: func() { sessionStarted <- struct{}{} },
		OnSessionError: func(err error) { sessionErrors <- err },
	}
	clientPeer, err := pion.NewPeerConnection(pion.Configuration{})
	if err != nil {
		t.Fatalf("create client peer: %v", err)
	}
	defer clientPeer.Close()
	channel, err := clientPeer.CreateDataChannel(protocolChannelLabel, nil)
	if err != nil {
		t.Fatalf("create protocol data channel: %v", err)
	}
	opened := make(chan struct{})
	channel.OnOpen(func() { close(opened) })
	offer := createGatheredOffer(t, clientPeer)
	answer, err := answerer.Answer(context.Background(), &SignalingOffer{
		SessionID: "signal-1", SDP: offer.SDP,
	}, nil)
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if answer.SessionID != "signal-1" || answer.SDP == "" {
		t.Fatalf("answer = %+v", answer)
	}
	if len(answer.Candidates) == 0 {
		t.Fatal("answer must explicitly publish gathered daemon candidates")
	}
	if err := clientPeer.SetRemoteDescription(pion.SessionDescription{Type: pion.SDPTypeAnswer, SDP: answer.SDP}); err != nil {
		t.Fatalf("set remote answer: %v", err)
	}
	for _, candidate := range answer.Candidates {
		if err := clientPeer.AddICECandidate(pion.ICECandidateInit{
			Candidate: candidate.Candidate, SDPMid: stringPointer(candidate.SDPMid),
			SDPMLineIndex:    uint16Pointer(uint16(candidate.SDPMLineIndex)),
			UsernameFragment: stringPointer(candidate.UsernameFragment),
		}); err != nil {
			t.Fatalf("add remote candidate: %v", err)
		}
	}
	select {
	case <-opened:
	case <-time.After(10 * time.Second):
		t.Fatal("anytty protocol data channel did not open")
	}
	select {
	case <-sessionStarted:
	case <-time.After(10 * time.Second):
		t.Fatal("authorized channel start was not reported")
	}
	select {
	case <-handler.called:
	case <-time.After(10 * time.Second):
		t.Fatal("authorized channel handler was not invoked")
	}
	clientFingerprint, err := RemoteCertificateFingerprint(clientPeer)
	if err != nil {
		t.Fatalf("RemoteCertificateFingerprint: %v", err)
	}
	if handler.daemonDTLSFingerprint == "" || handler.daemonDTLSFingerprint != clientFingerprint {
		t.Fatalf("daemon fingerprint = %q, client observed %q", handler.daemonDTLSFingerprint, clientFingerprint)
	}
	sessionErr := errors.New("authorized session failed")
	handler.result <- sessionErr
	select {
	case got := <-sessionErrors:
		if !errors.Is(got, sessionErr) {
			t.Fatalf("session error = %v, want %v", got, sessionErr)
		}
	case <-time.After(time.Second):
		t.Fatal("authorized handler failure was not reported")
	}
}

func stringPointer(value string) *string { return &value }

func uint16Pointer(value uint16) *uint16 { return &value }

func TestAnswererFailsClosedWithoutAuthorizedHandler(t *testing.T) {
	if _, err := (Answerer{}).Answer(context.Background(), &SignalingOffer{SDP: "not-used"}, nil); err == nil {
		t.Fatal("missing authorized handler must fail before WebRTC session creation")
	}
}

func TestAnswererParentCancelClosesPeerWithoutDataChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	handler := &recordingAuthorizedHandler{called: make(chan struct{})}
	closed := make(chan struct{})
	var closeCalls atomic.Int32
	answerer := Answerer{
		Handler: handler,
		OnPeerClosed: func() {
			close(closed)
		},
		closePeerForTest: func(peer *pion.PeerConnection) error {
			closeCalls.Add(1)
			return peer.GracefulClose()
		},
	}
	clientPeer, err := pion.NewPeerConnection(pion.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer clientPeer.Close()
	if _, err := clientPeer.CreateDataChannel("not-protocol", nil); err != nil {
		t.Fatal(err)
	}
	offer := createGatheredOffer(t, clientPeer)
	answer, err := answerer.Answer(ctx, &SignalingOffer{SessionID: "cancel-before-open", SDP: offer.SDP}, nil)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("parent cancellation did not close peer without a DataChannel")
	}
	answer.lifecycle.closeAndWait()
	if got := closeCalls.Load(); got != 1 {
		t.Fatalf("peer close calls = %d, want 1", got)
	}
	select {
	case <-handler.called:
		t.Fatal("handler ran without a protocol DataChannel")
	default:
	}
}

func TestPeerLifecycleWaitsForClaimedHandlerAndClosesExactlyOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	peerClosed := make(chan struct{})
	var closeCalls atomic.Int32
	var callbackCalls atomic.Int32
	lifecycle := newPeerLifecycle(ctx, nil, cancel, func() {
		callbackCalls.Add(1)
		close(peerClosed)
	}, func(*pion.PeerConnection) error {
		closeCalls.Add(1)
		return nil
	})
	if !lifecycle.claimHandler() || lifecycle.claimHandler() {
		t.Fatal("peer lifecycle did not admit exactly one protocol handler")
	}

	var requests sync.WaitGroup
	for range 16 {
		requests.Add(1)
		go func() {
			defer requests.Done()
			lifecycle.requestClose()
		}()
	}
	requests.Wait()
	select {
	case <-peerClosed:
		t.Fatal("peer close callback ran before the claimed handler finished")
	case <-time.After(25 * time.Millisecond):
	}
	lifecycle.finishHandler()
	lifecycle.closeAndWait()
	if closeCalls.Load() != 1 || callbackCalls.Load() != 1 {
		t.Fatalf("close calls=%d callbacks=%d, want 1/1", closeCalls.Load(), callbackCalls.Load())
	}
}

func TestAnswererRecoversDataChannelHandlerPanicAndClosesExactlyOnce(t *testing.T) {
	handler := &panickingAuthorizedHandler{called: make(chan int32, 2), canceled: make(chan int32, 2)}
	peerClosed := make(chan int32, 2)
	sessionErrors := make(chan error, 1)
	var peerCloseCount atomic.Int32
	var underlyingCloseCount atomic.Int32
	answerer := Answerer{
		Handler:        handler,
		OnSessionError: func(err error) { sessionErrors <- err },
		OnPeerClosed: func() {
			peerClosed <- peerCloseCount.Add(1)
			panic("sensitive OnPeerClosed panic")
		},
		closePeerForTest: func(peer *pion.PeerConnection) error {
			underlyingCloseCount.Add(1)
			_ = peer.Close()
			panic("sensitive peer Close panic")
		},
	}

	for session := int32(1); session <= 2; session++ {
		clientPeer, err := pion.NewPeerConnection(pion.Configuration{})
		if err != nil {
			t.Fatalf("create client peer %d: %v", session, err)
		}
		if _, err := clientPeer.CreateDataChannel(protocolChannelLabel, nil); err != nil {
			_ = clientPeer.Close()
			t.Fatalf("create protocol data channel %d: %v", session, err)
		}
		offer := createGatheredOffer(t, clientPeer)
		answer, err := answerer.Answer(context.Background(), &SignalingOffer{SessionID: fmt.Sprintf("panic-%d", session), SDP: offer.SDP}, nil)
		if err != nil {
			_ = clientPeer.Close()
			t.Fatalf("Answer %d: %v", session, err)
		}
		if err := clientPeer.SetRemoteDescription(pion.SessionDescription{Type: pion.SDPTypeAnswer, SDP: answer.SDP}); err != nil {
			_ = clientPeer.Close()
			t.Fatalf("set remote answer %d: %v", session, err)
		}
		for _, candidate := range answer.Candidates {
			if err := clientPeer.AddICECandidate(pion.ICECandidateInit{
				Candidate: candidate.Candidate, SDPMid: stringPointer(candidate.SDPMid),
				SDPMLineIndex: uint16Pointer(uint16(candidate.SDPMLineIndex)), UsernameFragment: stringPointer(candidate.UsernameFragment),
			}); err != nil {
				_ = clientPeer.Close()
				t.Fatalf("add remote candidate %d: %v", session, err)
			}
		}
		select {
		case got := <-handler.called:
			if got != session {
				t.Fatalf("handler call = %d, want %d", got, session)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("panicking handler %d was not invoked", session)
		}
		select {
		case got := <-handler.canceled:
			if got != session {
				t.Fatalf("canceled session = %d, want %d", got, session)
			}
		case <-time.After(time.Second):
			t.Fatalf("panicking handler context %d was not canceled", session)
		}
		select {
		case got := <-peerClosed:
			if got != session {
				t.Fatalf("peer close callback = %d, want %d", got, session)
			}
		case <-time.After(time.Second):
			t.Fatalf("panicking handler peer %d was not closed", session)
		}
		_ = clientPeer.Close()
	}
	if peerCloseCount.Load() != 2 {
		t.Fatalf("peer close callbacks = %d, want 2", peerCloseCount.Load())
	}
	if underlyingCloseCount.Load() != 2 {
		t.Fatalf("underlying peer Close calls = %d, want 2", underlyingCloseCount.Load())
	}
	select {
	case err := <-sessionErrors:
		t.Fatalf("handler panic leaked through session error callback: %v", err)
	default:
	}
}

func createGatheredOffer(t *testing.T, peer *pion.PeerConnection) pion.SessionDescription {
	t.Helper()
	offer, err := peer.CreateOffer(nil)
	if err != nil {
		t.Fatalf("create offer: %v", err)
	}
	gatherComplete := pion.GatheringCompletePromise(peer)
	if err := peer.SetLocalDescription(offer); err != nil {
		t.Fatalf("set local offer: %v", err)
	}
	select {
	case <-gatherComplete:
	case <-time.After(5 * time.Second):
		t.Fatal("client ICE gathering timed out")
	}
	description := peer.LocalDescription()
	if description == nil || description.SDP == "" {
		t.Fatal("client offer has no local SDP")
	}
	return *description
}

type recordingAuthorizedHandler struct {
	called                chan struct{}
	result                chan error
	daemonDTLSFingerprint string
}

type panickingAuthorizedHandler struct {
	calls    atomic.Int32
	called   chan int32
	canceled chan int32
}

func (handler *panickingAuthorizedHandler) ServeDataChannel(ctx context.Context, _ transport.Transport, _ string) error {
	call := handler.calls.Add(1)
	handler.called <- call
	go func() {
		<-ctx.Done()
		handler.canceled <- call
	}()
	panic("sensitive handler panic")
}

func (handler *recordingAuthorizedHandler) ServeDataChannel(ctx context.Context, _ transport.Transport, daemonDTLSFingerprint string) error {
	handler.daemonDTLSFingerprint = daemonDTLSFingerprint
	close(handler.called)
	if handler.result == nil {
		<-ctx.Done()
		return ctx.Err()
	}
	select {
	case err := <-handler.result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

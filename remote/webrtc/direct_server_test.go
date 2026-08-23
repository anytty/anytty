package webrtc

import (
	"container/heap"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anytty/anytty/internal/protocol/directsignal"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport"
	pion "github.com/pion/webrtc/v4"
)

func TestDirectServerPreAuthGlobalLimitRejectsBeforeWorkerAndReusesSlot(t *testing.T) {
	harness := newDirectServerHarness(t)
	var workers atomic.Int32
	harness.server.beforeConnectionWorkerRun = func() { workers.Add(1) }
	harness.start(t)

	clients := make([]net.Conn, 0, directSignalingPreAuthLimit+1)
	for index := 0; index < directSignalingPreAuthLimit; index++ {
		address := net.Addr(directTestAddress(fmt.Sprintf("[2001:db8::%x]:1234", index+1)))
		if index < directSignalingPreAuthPerIPLimit {
			if index%2 == 0 {
				address = directTestAddress(fmt.Sprintf("192.0.2.10:%d", 1200+index))
			} else {
				address = directTestAddress(fmt.Sprintf("[::ffff:192.0.2.10]:%d", 1200+index))
			}
		}
		_, client := harness.acceptPipe(t, address)
		clients = append(clients, client)
	}
	waitDirectServerState(t, harness.server, directSignalingPreAuthLimit, directSignalingPreAuthPerIPLimit)
	waitAtomicInt32(t, &workers, directSignalingPreAuthLimit)

	rejectedServer, rejectedClient := harness.acceptPipe(t, directTestAddress("[2001:db8::ffff]:1234"))
	response := readDirectResponse(t, rejectedClient)
	assertDirectErrorCode(t, response, remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_OVERLOADED)
	if rejectedServer.reads.Load() != 0 {
		t.Fatalf("overloaded connection entered request reader %d times", rejectedServer.reads.Load())
	}
	if got := workers.Load(); got != directSignalingPreAuthLimit {
		t.Fatalf("workers started = %d, want %d", got, directSignalingPreAuthLimit)
	}

	_ = clients[0].Close()
	waitDirectServerState(t, harness.server, directSignalingPreAuthLimit-1, directSignalingPreAuthPerIPLimit-1)
	_, replacement := harness.acceptPipe(t, directTestAddress("[2001:db8::fffe]:1234"))
	clients = append(clients, replacement)
	waitDirectServerState(t, harness.server, directSignalingPreAuthLimit, directSignalingPreAuthPerIPLimit-1)
	waitAtomicInt32(t, &workers, directSignalingPreAuthLimit+1)

	harness.stop(t)
	assertDirectServerSlotsEmpty(t, harness.server)
	for _, connection := range clients {
		_ = connection.Close()
	}
}

func TestDirectServerPreAuthNormalizesSourceIPAndRejectsNinth(t *testing.T) {
	harness := newDirectServerHarness(t)
	var workers atomic.Int32
	harness.server.beforeConnectionWorkerRun = func() { workers.Add(1) }
	harness.start(t)

	clients := make([]net.Conn, 0, directSignalingPreAuthPerIPLimit+1)
	for index := 0; index < directSignalingPreAuthPerIPLimit; index++ {
		var address net.Addr = directTestAddress(fmt.Sprintf("192.0.2.44:%d", 2000+index))
		if index%2 != 0 {
			address = directTestAddress(fmt.Sprintf("[::ffff:192.0.2.44]:%d", 2000+index))
		}
		_, client := harness.acceptPipe(t, address)
		clients = append(clients, client)
	}
	waitDirectServerState(t, harness.server, directSignalingPreAuthPerIPLimit, directSignalingPreAuthPerIPLimit)
	waitAtomicInt32(t, &workers, directSignalingPreAuthPerIPLimit)

	rejectedServer, rejectedClient := harness.acceptPipe(t, &net.TCPAddr{IP: net.ParseIP("192.0.2.44"), Port: 3000})
	assertDirectErrorCode(t, readDirectResponse(t, rejectedClient), remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_OVERLOADED)
	if rejectedServer.reads.Load() != 0 || workers.Load() != directSignalingPreAuthPerIPLimit {
		t.Fatalf("ninth source-IP connection reads=%d workers=%d", rejectedServer.reads.Load(), workers.Load())
	}

	_ = clients[0].Close()
	waitDirectServerState(t, harness.server, directSignalingPreAuthPerIPLimit-1, directSignalingPreAuthPerIPLimit-1)
	_, replacement := harness.acceptPipe(t, directTestAddress("[::ffff:192.0.2.44]:3001"))
	clients = append(clients, replacement)
	waitDirectServerState(t, harness.server, directSignalingPreAuthPerIPLimit, directSignalingPreAuthPerIPLimit)
	waitAtomicInt32(t, &workers, directSignalingPreAuthPerIPLimit+1)

	harness.stop(t)
	assertDirectServerSlotsEmpty(t, harness.server)
	for _, connection := range clients {
		_ = connection.Close()
	}
}

func TestDirectServerFirstRequestTimeoutUsesShortTestHook(t *testing.T) {
	harness := newDirectServerHarness(t)
	harness.server.firstRequestLimit = 25 * time.Millisecond
	harness.start(t)

	started := time.Now()
	_, client := harness.acceptPipe(t, directTestAddress("192.0.2.50:4000"))
	assertDirectErrorCode(t, readDirectResponse(t, client), remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_PROTOCOL)
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("first request timeout took %s", elapsed)
	}
	waitDirectServerState(t, harness.server, 0, 0)
}

func TestDirectServerFirstRequestDeadlineUsesEarlierContextDeadline(t *testing.T) {
	requestDeadline := time.Now().Add(directSignalingFirstRequestLimit)
	contextDeadline := time.Now().Add(time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), contextDeadline)
	defer cancel()
	if got := earlierDeadline(requestDeadline, ctx); !got.Equal(contextDeadline) {
		t.Fatalf("first request deadline = %s, want earlier context deadline %s", got, contextDeadline)
	}
}

func TestDirectServerPeerLimitRejectsImmediatelyAndReleasesPreAuth(t *testing.T) {
	harness := newDirectServerHarness(t)
	for index := 0; index < directSignalingPeerLimit; index++ {
		harness.server.peerSlots <- struct{}{}
	}
	harness.start(t)

	started := time.Now()
	response := harness.exchange(t, directTestAddress("192.0.2.60:5000"), harness.request("peer-full"))
	assertDirectErrorCode(t, response, remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_OVERLOADED)
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("full peer admission waited %s", elapsed)
	}
	waitDirectServerState(t, harness.server, 0, 0)
	if got := len(harness.server.peerSlots); got != directSignalingPeerLimit {
		t.Fatalf("peer slots = %d, want %d", got, directSignalingPeerLimit)
	}
	for index := 0; index < directSignalingPeerLimit; index++ {
		<-harness.server.peerSlots
	}
}

func TestDirectServerAnswerFailuresReleasePeerExactlyOnce(t *testing.T) {
	harness := newDirectServerHarness(t)
	harness.server.peerConnections = func(pion.Configuration) (*pion.PeerConnection, error) {
		return nil, errors.New("injected peer creation failure")
	}
	harness.start(t)

	for index := 0; index < 64; index++ {
		response := harness.exchange(t, directTestAddress(fmt.Sprintf("[2001:db8:1::%x]:6000", index+1)), harness.request(fmt.Sprintf("factory-failure-%d", index)))
		assertDirectErrorCode(t, response, remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_INTERNAL)
		if got := len(harness.server.peerSlots); got != 0 {
			t.Fatalf("peer slots after failure %d = %d", index, got)
		}
	}

	harness.server.peerConnections = pion.NewPeerConnection
	response := harness.exchange(t, directTestAddress("192.0.2.61:6001"), harness.request("invalid-sdp-close-callback"))
	assertDirectErrorCode(t, response, remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_INTERNAL)
	if got := len(harness.server.peerSlots); got != 0 {
		t.Fatalf("peer slots after callback and caller release = %d", got)
	}
	waitDirectServerState(t, harness.server, 0, 0)
}

func TestDirectServerWriteFailureClosesUnhandedPeer(t *testing.T) {
	harness := newDirectServerHarness(t)
	peerClosed := make(chan struct{})
	harness.server.answerForTest = func(ctx context.Context, answerer Answerer, _ *SignalingOffer) (*SignalingAnswer, error) {
		_, cancel := context.WithCancel(ctx)
		releasePeer := answerer.OnPeerClosed
		lifecycle := newPeerLifecycle(ctx, nil, cancel, func() {
			releasePeer()
			close(peerClosed)
		}, func(*pion.PeerConnection) error { return nil })
		return &SignalingAnswer{SessionID: "write-failure", SDP: "unwritten-answer", lifecycle: lifecycle}, nil
	}
	harness.start(t)

	_, client := harness.acceptPipe(t, directTestAddress("192.0.2.62:6100"))
	if err := client.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := directsignal.WriteMessage(client, harness.request("write-failure")); err != nil {
		t.Fatal(err)
	}
	_ = client.Close()
	select {
	case <-peerClosed:
	case <-time.After(time.Second):
		t.Fatal("failed response write retained an unhanded peer")
	}
	if got := len(harness.server.peerSlots); got != 0 {
		t.Fatalf("peer slots after response write failure = %d", got)
	}
}

func TestDirectServerServeWaitsForPeerHandlerJoin(t *testing.T) {
	harness := newDirectServerHarness(t)
	handlerStarted := make(chan struct{})
	handlerCanceled := make(chan struct{})
	releaseHandler := make(chan struct{})
	peerClosed := make(chan struct{})
	harness.server.answerForTest = func(ctx context.Context, answerer Answerer, _ *SignalingOffer) (*SignalingAnswer, error) {
		sessionCtx, cancel := context.WithCancel(ctx)
		releasePeer := answerer.OnPeerClosed
		lifecycle := newPeerLifecycle(ctx, nil, cancel, func() {
			releasePeer()
			close(peerClosed)
		}, func(*pion.PeerConnection) error { return nil })
		if !lifecycle.claimHandler() {
			return nil, errors.New("test peer handler was not admitted")
		}
		go func() {
			close(handlerStarted)
			<-sessionCtx.Done()
			close(handlerCanceled)
			<-releaseHandler
			lifecycle.finishHandler()
		}()
		return &SignalingAnswer{SessionID: "joined-peer", SDP: "joined-answer", lifecycle: lifecycle}, nil
	}
	harness.start(t)
	response := harness.exchange(t, directTestAddress("192.0.2.62:6101"), harness.request("joined-peer"))
	if response.GetAnswer() == nil {
		t.Fatalf("direct response = %#v, want answer", response)
	}
	<-handlerStarted

	harness.stopOnce.Do(func() {
		harness.cancel()
		if err := harness.server.Close(); err != nil {
			t.Fatalf("close direct server: %v", err)
		}
		select {
		case <-handlerCanceled:
		case <-time.After(time.Second):
			t.Fatal("server close did not cancel peer handler")
		}
		select {
		case <-peerClosed:
			t.Fatal("peer closed before its handler joined")
		case err := <-harness.done:
			t.Fatalf("Serve returned before peer handler joined: %v", err)
		case <-time.After(25 * time.Millisecond):
		}
		close(releaseHandler)
		select {
		case <-peerClosed:
		case <-time.After(time.Second):
			t.Fatal("peer did not close after handler joined")
		}
		select {
		case err := <-harness.done:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Fatalf("serve direct server: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("Serve did not return after peer handler joined")
		}
	})
}

func TestDirectServerPreAuthPanicGuardsReleaseAndContinue(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		install func(*DirectServer, func())
	}{
		{name: "handoff", install: func(server *DirectServer, hook func()) { server.afterPreAuthAcquire = hook }},
		{name: "worker", install: func(server *DirectServer, hook func()) { server.beforeConnectionWorkerRun = hook }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newDirectServerHarness(t)
			var panicOnce atomic.Bool
			testCase.install(harness.server, func() {
				if panicOnce.CompareAndSwap(false, true) {
					panic("sensitive pre-auth panic")
				}
			})
			harness.start(t)

			serverConnection, response := harness.exchangeDuringEarlyFailure(t, directTestAddress("192.0.2.62:6100"), harness.request("pre-auth-panic"))
			assertDirectPanicFailure(t, response)
			waitAtomicInt32(t, &serverConnection.closes, 1)
			waitDirectServerState(t, harness.server, 0, 0)
			assertPreAuthSlotsReusable(t, harness.server)

			response = harness.exchange(t, directTestAddress("192.0.2.62:6101"), harness.request("after-pre-auth-panic"))
			assertDirectErrorCode(t, response, remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_INTERNAL)
			waitDirectServerState(t, harness.server, 0, 0)
		})
	}
}

func TestDirectServerPeerFactoryPanicReleasesAndContinues(t *testing.T) {
	harness := newDirectServerHarness(t)
	var calls atomic.Int32
	harness.server.peerConnections = func(pion.Configuration) (*pion.PeerConnection, error) {
		if calls.Add(1) == 1 {
			panic("sensitive peer factory panic")
		}
		return nil, errors.New("factory remains available")
	}
	harness.start(t)

	serverConnection, response := harness.exchangeWithServerConnection(t, directTestAddress("192.0.2.63:6200"), harness.request("peer-factory-panic"))
	assertDirectPanicFailure(t, response)
	waitAtomicInt32(t, &serverConnection.closes, 1)
	waitDirectServerState(t, harness.server, 0, 0)
	assertPeerSlotsReusable(t, harness.server)

	response = harness.exchange(t, directTestAddress("192.0.2.63:6201"), harness.request("after-peer-factory-panic"))
	assertDirectErrorCode(t, response, remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_INTERNAL)
	if calls.Load() != 2 {
		t.Fatalf("peer factory calls = %d, want 2", calls.Load())
	}
}

func TestDirectServerAnswerPanicReleasesBeforeLatePeerClosed(t *testing.T) {
	harness := newDirectServerHarness(t)
	latePeerClosed := make(chan func(), 1)
	var calls atomic.Int32
	harness.server.answerForTest = func(_ context.Context, answerer Answerer, _ *SignalingOffer) (*SignalingAnswer, error) {
		if calls.Add(1) == 1 {
			latePeerClosed <- answerer.OnPeerClosed
			panic("sensitive answer panic")
		}
		return nil, errors.New("answerer remains available")
	}
	harness.start(t)

	serverConnection, response := harness.exchangeWithServerConnection(t, directTestAddress("192.0.2.64:6300"), harness.request("answer-panic"))
	assertDirectPanicFailure(t, response)
	waitAtomicInt32(t, &serverConnection.closes, 1)
	waitDirectServerState(t, harness.server, 0, 0)
	assertPeerSlotsReusable(t, harness.server)

	callback := <-latePeerClosed
	callbackDone := make(chan struct{})
	go func() {
		callback()
		callback()
		close(callbackDone)
	}()
	select {
	case <-callbackDone:
	case <-time.After(time.Second):
		t.Fatal("late OnPeerClosed blocked on an already released peer slot")
	}
	if got := len(harness.server.peerSlots); got != 0 {
		t.Fatalf("late OnPeerClosed changed peer slots to %d", got)
	}

	response = harness.exchange(t, directTestAddress("192.0.2.64:6301"), harness.request("after-answer-panic"))
	assertDirectErrorCode(t, response, remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_INTERNAL)
	if calls.Load() != 2 {
		t.Fatalf("answer calls = %d, want 2", calls.Load())
	}
}

func TestDirectServerAdmissionRejectionsReleasePreAuth(t *testing.T) {
	harness := newDirectServerHarness(t)
	for index := 0; index < directSignalingPeerLimit; index++ {
		harness.server.peerSlots <- struct{}{}
	}
	harness.start(t)

	protocol := harness.request("protocol")
	protocol.OfferSdp = ""
	assertDirectErrorCode(t, harness.exchange(t, directTestAddress("192.0.2.70:7000"), protocol), remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_PROTOCOL)
	missingAuthorization := harness.request("missing-authorization")
	missingAuthorization.GrantId = ""
	missingAuthorization.GrantExpiresAtUnixNano = 0
	assertDirectErrorCode(t, harness.exchange(t, directTestAddress("192.0.2.70:7001"), missingAuthorization), remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_PROTOCOL)

	mismatch := harness.request("identity-mismatch")
	mismatch.ExpectedDeviceFingerprint = "sha256:mismatch"
	assertDirectErrorCode(t, harness.exchange(t, directTestAddress("192.0.2.70:7002"), mismatch), remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_IDENTITY_MISMATCH)

	harness.server.admission = directRejectingAdmission{}
	authorization := harness.request("grant-secret-must-not-leak")
	response := harness.exchange(t, directTestAddress("192.0.2.70:7003"), authorization)
	assertDirectErrorCode(t, response, remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_AUTHORIZATION)
	if got := response.GetError().GetMessage(); got != "direct signaling authorization is unavailable" || strings.Contains(got, authorization.GetGrantId()) {
		t.Fatalf("authorization error leaked identity: %q", got)
	}
	harness.server.admission = directTestHandler{}

	replay := harness.request("replay")
	assertDirectErrorCode(t, harness.exchange(t, directTestAddress("192.0.2.70:7004"), replay), remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_OVERLOADED)
	assertDirectErrorCode(t, harness.exchange(t, directTestAddress("192.0.2.70:7005"), replay), remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_REPLAYED)
	waitDirectServerState(t, harness.server, 0, 0)

	for index := 0; index < directSignalingPeerLimit; index++ {
		<-harness.server.peerSlots
	}
}

func TestDirectServerAdmitsPermanentGrantWithZeroExpiry(t *testing.T) {
	harness := newDirectServerHarness(t)
	harness.start(t)
	request := harness.request("permanent-grant")
	request.GrantExpiresAtUnixNano = 0
	if got := harness.server.admit(request); got != remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("permanent grant admission = %s", got)
	}
}

func TestDirectServerRequestIDRawByteLengthBoundary(t *testing.T) {
	for _, test := range []struct {
		name      string
		requestID string
		want      remoteauthpb.DirectSignalingErrorCode
	}{
		{name: "one byte", requestID: "x", want: remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_UNSPECIFIED},
		{name: "128 bytes", requestID: strings.Repeat("x", directSignalingRequestIDMaxBytes), want: remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_UNSPECIFIED},
		{name: "129 bytes", requestID: strings.Repeat("x", directSignalingRequestIDMaxBytes+1), want: remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_PROTOCOL},
	} {
		t.Run(test.name, func(t *testing.T) {
			harness := newDirectServerHarness(t)
			harness.start(t)
			if got := harness.server.admit(harness.request(test.requestID)); got != test.want {
				t.Fatalf("admit request ID of %d bytes = %s, want %s", len(test.requestID), got, test.want)
			}
			wantConsumed := 0
			if test.want == remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_UNSPECIFIED {
				wantConsumed = 1
			}
			if got := len(harness.server.consumed); got != wantConsumed {
				t.Fatalf("consumed entries = %d, want %d", got, wantConsumed)
			}
		})
	}
}

func TestDirectServerConsumedCapacityAndReplayPriority(t *testing.T) {
	harness := newDirectServerHarness(t)
	harness.start(t)
	expiresAt := harness.now.Add(remoteauth.DirectSignalingMaxTTL)
	for index := 0; index < directSignalingConsumedLimit-1; index++ {
		seedDirectConsumed(harness.server, fmt.Sprintf("used-%04d", index), expiresAt)
	}

	last := harness.request("last-capacity-slot")
	if got := harness.server.admit(last); got != remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("last capacity admission = %s", got)
	}
	if got := len(harness.server.consumed); got != directSignalingConsumedLimit {
		t.Fatalf("consumed entries at boundary = %d, want %d", got, directSignalingConsumedLimit)
	}
	if got := harness.server.admit(harness.request("one-over-capacity")); got != remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_OVERLOADED {
		t.Fatalf("new full-capacity admission = %s, want OVERLOADED", got)
	}
	if got := harness.server.admit(last); got != remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_REPLAYED {
		t.Fatalf("existing full-capacity admission = %s, want REPLAYED", got)
	}
	if got := len(harness.server.consumed); got != directSignalingConsumedLimit {
		t.Fatalf("full consumed entries changed to %d", got)
	}
}

func TestDirectServerHighWaterCleanupInspectsOnlyExpiredHeapEntries(t *testing.T) {
	harness := newDirectServerHarness(t)
	harness.start(t)
	const expiredEntries = 7
	for index := 0; index < directSignalingConsumedLimit; index++ {
		expiresAt := harness.now.Add(remoteauth.DirectSignalingMaxTTL)
		if index < expiredEntries {
			expiresAt = harness.now
		}
		seedDirectConsumed(harness.server, fmt.Sprintf("high-water-%04d", index), expiresAt)
	}

	harness.server.mu.Lock()
	inspected := harness.server.cleanupExpiredConsumed(harness.now)
	remaining := len(harness.server.consumed)
	remainingExpiry := len(harness.server.consumedExpiry)
	harness.server.mu.Unlock()
	if inspected != expiredEntries+1 {
		t.Fatalf("heap entries inspected = %d, want %d for %d expired of %d", inspected, expiredEntries+1, expiredEntries, directSignalingConsumedLimit)
	}
	if remaining != directSignalingConsumedLimit-expiredEntries || remainingExpiry != remaining {
		t.Fatalf("high-water cleanup retained map=%d heap=%d, want %d", remaining, remainingExpiry, directSignalingConsumedLimit-expiredEntries)
	}
	for index := 0; index < expiredEntries; index++ {
		if got := harness.server.admit(harness.request(fmt.Sprintf("high-water-replacement-%d", index))); got != remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_UNSPECIFIED {
			t.Fatalf("replacement admission %d = %s", index, got)
		}
	}
	if got := harness.server.admit(harness.request("high-water-over-capacity")); got != remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_OVERLOADED {
		t.Fatalf("high-water over-capacity admission = %s", got)
	}
	if got := harness.server.admit(harness.request("high-water-0007")); got != remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_REPLAYED {
		t.Fatalf("high-water replay admission = %s", got)
	}
}

func TestDirectServerExpiryHeapLazyDeletionPreservesNewerMapEntry(t *testing.T) {
	harness := newDirectServerHarness(t)
	harness.start(t)
	requestID := "lazy-expiry"
	newExpiry := harness.now.Add(remoteauth.DirectSignalingMaxTTL)
	harness.server.consumed[requestID] = newExpiry
	heap.Push(&harness.server.consumedExpiry, directConsumedExpiry{requestID: requestID, expiresAt: harness.now})
	heap.Push(&harness.server.consumedExpiry, directConsumedExpiry{requestID: requestID, expiresAt: newExpiry})

	harness.server.mu.Lock()
	inspected := harness.server.cleanupExpiredConsumed(harness.now)
	gotExpiry, exists := harness.server.consumed[requestID]
	remainingExpiry := len(harness.server.consumedExpiry)
	harness.server.mu.Unlock()
	if inspected != 2 || !exists || !gotExpiry.Equal(newExpiry) || remainingExpiry != 1 {
		t.Fatalf("lazy cleanup inspected=%d exists=%v expiry=%s heap=%d", inspected, exists, gotExpiry, remainingExpiry)
	}
}

func TestDirectServerExpiredRequestIDCanBeReusedAfterClockAdvance(t *testing.T) {
	harness := newDirectServerHarness(t)
	harness.start(t)
	now := harness.now
	harness.server.now = func() time.Time { return now }
	request := directRequestAt(harness, "reusable-request", now)
	if got := harness.server.admit(request); got != remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("initial admission = %s", got)
	}
	if got := harness.server.admit(request); got != remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_REPLAYED {
		t.Fatalf("immediate replay admission = %s", got)
	}

	now = time.Unix(0, request.GetExpiresAtUnixNano()).UTC()
	if got := harness.server.admit(directRequestAt(harness, request.GetRequestId(), now)); got != remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("admission after consumed expiry = %s, want success", got)
	}
	if got := len(harness.server.consumed); got != 1 {
		t.Fatalf("consumed entries after reuse = %d, want 1", got)
	}
}

func TestDirectServerConcurrentAdmissionNeverExceedsConsumedLimit(t *testing.T) {
	harness := newDirectServerHarness(t)
	harness.start(t)
	const extraRequests = 64
	start := make(chan struct{})
	var admitted atomic.Int32
	var overloaded atomic.Int32
	var wait sync.WaitGroup
	for index := 0; index < directSignalingConsumedLimit+extraRequests; index++ {
		request := harness.request(fmt.Sprintf("concurrent-%04d", index))
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			switch harness.server.admit(request) {
			case remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_UNSPECIFIED:
				admitted.Add(1)
			case remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_OVERLOADED:
				overloaded.Add(1)
			default:
				t.Errorf("unexpected concurrent admission result")
			}
		}()
	}
	close(start)
	wait.Wait()
	if got := admitted.Load(); got != directSignalingConsumedLimit {
		t.Fatalf("concurrent admissions = %d, want %d", got, directSignalingConsumedLimit)
	}
	if got := overloaded.Load(); got != extraRequests {
		t.Fatalf("concurrent overloads = %d, want %d", got, extraRequests)
	}
	if got := len(harness.server.consumed); got != directSignalingConsumedLimit {
		t.Fatalf("concurrent consumed entries = %d, want %d", got, directSignalingConsumedLimit)
	}
}

func TestDirectServerPeerFullDoesNotExceedConsumedLimit(t *testing.T) {
	harness := newDirectServerHarness(t)
	expiresAt := harness.now.Add(remoteauth.DirectSignalingMaxTTL)
	for index := 0; index < directSignalingConsumedLimit-1; index++ {
		seedDirectConsumed(harness.server, fmt.Sprintf("peer-full-used-%04d", index), expiresAt)
	}
	for index := 0; index < directSignalingPeerLimit; index++ {
		harness.server.peerSlots <- struct{}{}
	}
	harness.start(t)

	last := harness.request("peer-full-last-slot")
	assertDirectErrorCode(t, harness.exchange(t, directTestAddress("192.0.2.71:7100"), last), remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_OVERLOADED)
	if got := len(harness.server.consumed); got != directSignalingConsumedLimit {
		t.Fatalf("peer-full consumed entries = %d, want %d", got, directSignalingConsumedLimit)
	}
	assertDirectErrorCode(t, harness.exchange(t, directTestAddress("192.0.2.71:7101"), harness.request("peer-full-over-capacity")), remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_OVERLOADED)
	assertDirectErrorCode(t, harness.exchange(t, directTestAddress("192.0.2.71:7102"), last), remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_REPLAYED)
	if got := len(harness.server.consumed); got != directSignalingConsumedLimit {
		t.Fatalf("peer-full consumed entries after rejections = %d, want %d", got, directSignalingConsumedLimit)
	}
	for index := 0; index < directSignalingPeerLimit; index++ {
		<-harness.server.peerSlots
	}
}

func TestDirectServerCloseBetweenAcquireAndTrackReleasesWithoutWorker(t *testing.T) {
	harness := newDirectServerHarness(t)
	acquired := make(chan struct{})
	resume := make(chan struct{})
	var acquireOnce sync.Once
	var workers atomic.Int32
	harness.server.afterPreAuthAcquire = func() {
		acquireOnce.Do(func() { close(acquired) })
		<-resume
	}
	harness.server.beforeConnectionWorkerRun = func() { workers.Add(1) }
	harness.start(t)

	serverConnection, client := harness.acceptPipe(t, directTestAddress("192.0.2.80:8000"))
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("pre-auth acquisition did not reach interleaving hook")
	}
	if err := harness.server.Close(); err != nil {
		t.Fatal(err)
	}
	close(resume)
	assertDirectErrorCode(t, readDirectResponse(t, client), remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_OVERLOADED)
	harness.stop(t)
	assertDirectServerSlotsEmpty(t, harness.server)
	if workers.Load() != 0 || serverConnection.reads.Load() != 0 {
		t.Fatalf("close interleaving started workers=%d reads=%d", workers.Load(), serverConnection.reads.Load())
	}
}

func TestDirectServerConnectionClosePanicDoesNotInterruptShutdownCleanup(t *testing.T) {
	harness := newDirectServerHarness(t)
	harness.allowedCloseErr = errDirectConnectionClosePanic
	var workers atomic.Int32
	harness.server.beforeConnectionWorkerRun = func() { workers.Add(1) }
	harness.start(t)

	closeStarted := make(chan struct{})
	resumeClose := make(chan struct{})
	serverConnections := make([]*directTestConn, 0, 3)
	clients := make([]net.Conn, 0, 3)
	for index := 0; index < 3; index++ {
		serverSide, clientSide := net.Pipe()
		connection := &directTestConn{
			Conn: serverSide, remoteAddress: directTestAddress(fmt.Sprintf("192.0.2.%d:6400", index+100)),
		}
		if index == 1 {
			connection.closePanic = "sensitive connection close panic"
			connection.closeStarted = closeStarted
			connection.closeResume = resumeClose
		}
		harness.acceptConnection(t, connection)
		serverConnections = append(serverConnections, connection)
		clients = append(clients, clientSide)
	}
	waitDirectServerState(t, harness.server, 3, 1)
	waitAtomicInt32(t, &workers, 3)

	closeDone := make(chan error, 1)
	go func() { closeDone <- harness.server.Close() }()
	select {
	case <-closeStarted:
	case <-time.After(time.Second):
		t.Fatal("panicking connection Close was not reached")
	}
	waitAtomicInt32(t, &serverConnections[1].readReturns, 1)
	close(resumeClose)
	select {
	case err := <-closeDone:
		if !errors.Is(err, errDirectConnectionClosePanic) || err.Error() != errDirectConnectionClosePanic.Error() {
			t.Fatalf("server Close error = %v, want fixed internal close error", err)
		}
		if strings.Contains(err.Error(), "sensitive") {
			t.Fatalf("server Close leaked panic text: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server Close did not finish after connection panic")
	}

	harness.stop(t)
	assertDirectServerSlotsEmpty(t, harness.server)
	for index, connection := range serverConnections {
		if got := connection.closes.Load(); got != 1 {
			t.Fatalf("underlying connection %d Close calls = %d, want 1", index, got)
		}
	}
	if got := harness.mux.closes.Load(); got != 1 {
		t.Fatalf("ICE mux Close calls = %d, want 1", got)
	}
	if got := harness.listener.closes.Load(); got != 1 {
		t.Fatalf("signaling listener Close calls = %d, want 1", got)
	}
	for _, client := range clients {
		_ = client.Close()
	}
}

func TestDirectServerHandlerAndPeerCleanupPanicsReleaseSlotAndContinue(t *testing.T) {
	harness := newDirectServerHarness(t)
	handler := &panickingAuthorizedHandler{called: make(chan int32, 1), canceled: make(chan int32, 1)}
	harness.server.handler = handler
	harness.server.peerConnections = pion.NewPeerConnection
	peerClosed := make(chan struct{}, 1)
	var peerCloseCalls atomic.Int32
	var peerClosedCalls atomic.Int32
	harness.server.answerForTest = func(ctx context.Context, answerer Answerer, offer *SignalingOffer) (*SignalingAnswer, error) {
		releasePeer := answerer.OnPeerClosed
		answerer.OnPeerClosed = func() {
			releasePeer()
			peerClosedCalls.Add(1)
			peerClosed <- struct{}{}
			panic("sensitive OnPeerClosed panic")
		}
		answerer.closePeerForTest = func(peer *pion.PeerConnection) error {
			peerCloseCalls.Add(1)
			_ = peer.Close()
			panic("sensitive peer Close panic")
		}
		return answerer.Answer(ctx, offer, nil)
	}
	harness.start(t)

	clientPeer, err := pion.NewPeerConnection(pion.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer clientPeer.Close()
	if _, err := clientPeer.CreateDataChannel(protocolChannelLabel, nil); err != nil {
		t.Fatal(err)
	}
	offer := createGatheredOffer(t, clientPeer)
	request := harness.request("stacked-handler-cleanup-panics")
	request.OfferSdp = offer.SDP
	response := harness.exchange(t, directTestAddress("192.0.2.110:6500"), request)
	if response.GetAnswer() == nil {
		t.Fatalf("direct response = %#v, want answer", response)
	}
	answer := response.GetAnswer()
	if err := clientPeer.SetRemoteDescription(pion.SessionDescription{Type: pion.SDPTypeAnswer, SDP: answer.GetAnswerSdp()}); err != nil {
		t.Fatal(err)
	}
	for _, candidate := range answer.GetCandidates() {
		if err := clientPeer.AddICECandidate(pion.ICECandidateInit{
			Candidate: candidate.GetCandidate(), SDPMid: stringPointer(candidate.GetSdpMid()),
			SDPMLineIndex: uint16Pointer(uint16(candidate.GetSdpMlineIndex())), UsernameFragment: stringPointer(candidate.GetUsernameFragment()),
		}); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-handler.called:
	case <-time.After(10 * time.Second):
		t.Fatal("panicking direct handler was not invoked")
	}
	select {
	case <-handler.canceled:
	case <-time.After(time.Second):
		t.Fatal("panicking direct handler context was not canceled")
	}
	select {
	case <-peerClosed:
	case <-time.After(time.Second):
		t.Fatal("panicking peer close callback was not invoked")
	}
	if got := peerCloseCalls.Load(); got != 1 {
		t.Fatalf("peer Close calls = %d, want 1", got)
	}
	if got := peerClosedCalls.Load(); got != 1 {
		t.Fatalf("OnPeerClosed calls = %d, want 1", got)
	}
	if got := len(harness.server.peerSlots); got != 0 {
		t.Fatalf("peer slots after stacked panics = %d", got)
	}
	assertPeerSlotsReusable(t, harness.server)

	harness.server.answerForTest = func(context.Context, Answerer, *SignalingOffer) (*SignalingAnswer, error) {
		return nil, errors.New("answerer remains available")
	}
	response = harness.exchange(t, directTestAddress("192.0.2.110:6501"), harness.request("after-stacked-handler-cleanup-panics"))
	assertDirectErrorCode(t, response, remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_INTERNAL)
	if message := response.GetError().GetMessage(); message != "create direct signaling answer failed" || strings.Contains(message, "sensitive") {
		t.Fatalf("follow-up response message = %q", message)
	}
	waitDirectServerState(t, harness.server, 0, 0)
}

type directServerHarness struct {
	server          *DirectServer
	listener        *directTestListener
	mux             *directTestTCPMux
	ctx             context.Context
	cancel          context.CancelFunc
	done            chan error
	now             time.Time
	stopOnce        sync.Once
	allowedCloseErr error
}

func newDirectServerHarness(t *testing.T) *directServerHarness {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := remoteauth.NewIdentity("direct-server-test", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	listener := newDirectTestListener()
	mux := &directTestTCPMux{}
	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now().UTC()
	harness := &directServerHarness{
		listener: listener, mux: mux, ctx: ctx, cancel: cancel, done: make(chan error, 1), now: now,
		server: &DirectServer{
			identity: identity, handler: directTestHandler{}, admission: directTestHandler{}, signalingListener: listener, iceMux: mux,
			peerConnections: func(pion.Configuration) (*pion.PeerConnection, error) { return nil, errors.New("unused peer factory") },
			now:             func() time.Time { return now }, firstRequestLimit: directSignalingFirstRequestLimit,
			consumed: make(map[string]time.Time), conns: make(map[*directConnection]struct{}), preAuthByIP: make(map[string]int),
			peerSlots: make(chan struct{}, directSignalingPeerLimit),
		},
	}
	t.Cleanup(func() { harness.stop(t) })
	return harness
}

func (harness *directServerHarness) start(t *testing.T) {
	t.Helper()
	go func() { harness.done <- harness.server.Serve(harness.ctx) }()
}

func (harness *directServerHarness) stop(t *testing.T) {
	t.Helper()
	harness.stopOnce.Do(func() {
		harness.cancel()
		if err := harness.server.Close(); err != nil && !errors.Is(err, harness.allowedCloseErr) {
			t.Errorf("close direct server: %v", err)
		}
		select {
		case err := <-harness.done:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("serve direct server: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("direct server did not stop")
		}
	})
}

func (harness *directServerHarness) acceptPipe(t *testing.T, remoteAddress net.Addr) (*directTestConn, net.Conn) {
	t.Helper()
	serverSide, clientSide := net.Pipe()
	connection := &directTestConn{Conn: serverSide, remoteAddress: remoteAddress}
	harness.acceptConnection(t, connection)
	return connection, clientSide
}

func (harness *directServerHarness) acceptConnection(t *testing.T, connection net.Conn) {
	t.Helper()
	select {
	case harness.listener.connections <- connection:
	case <-time.After(time.Second):
		t.Fatal("direct test listener did not accept connection")
	}
}

func (harness *directServerHarness) exchange(t *testing.T, remoteAddress net.Addr, request *remoteauthpb.DirectSignalingRequestV2) *remoteauthpb.DirectSignalingResponseV2 {
	t.Helper()
	_, response := harness.exchangeWithServerConnection(t, remoteAddress, request)
	return response
}

func (harness *directServerHarness) exchangeWithServerConnection(t *testing.T, remoteAddress net.Addr, request *remoteauthpb.DirectSignalingRequestV2) (*directTestConn, *remoteauthpb.DirectSignalingResponseV2) {
	t.Helper()
	serverConnection, client := harness.acceptPipe(t, remoteAddress)
	defer client.Close()
	if err := client.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := directsignal.WriteMessage(client, request); err != nil {
		t.Fatal(err)
	}
	response := &remoteauthpb.DirectSignalingResponseV2{}
	if err := directsignal.ReadMessage(client, response); err != nil {
		t.Fatal(err)
	}
	return serverConnection, response
}

func (harness *directServerHarness) exchangeDuringEarlyFailure(t *testing.T, remoteAddress net.Addr, request *remoteauthpb.DirectSignalingRequestV2) (*directTestConn, *remoteauthpb.DirectSignalingResponseV2) {
	t.Helper()
	serverConnection, client := harness.acceptPipe(t, remoteAddress)
	defer client.Close()
	if err := client.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	writeDone := make(chan error, 1)
	go func() { writeDone <- directsignal.WriteMessage(client, request) }()
	response := &remoteauthpb.DirectSignalingResponseV2{}
	if err := directsignal.ReadMessage(client, response); err != nil {
		t.Fatal(err)
	}
	if err := <-writeDone; err != nil && !errors.Is(err, io.ErrClosedPipe) && !errors.Is(err, net.ErrClosed) {
		t.Fatalf("write request during early server failure: %v", err)
	}
	return serverConnection, response
}

func (harness *directServerHarness) request(id string) *remoteauthpb.DirectSignalingRequestV2 {
	return &remoteauthpb.DirectSignalingRequestV2{
		SchemaVersion: remoteauth.DirectSignalingSchemaVersion, RequestId: id,
		ExpectedDeviceId: harness.server.identity.DeviceID, ExpectedDeviceFingerprint: harness.server.identity.Fingerprint,
		OfferSdp: "invalid-sdp-for-admission-only", IssuedAtUnixNano: harness.now.UnixNano(),
		ExpiresAtUnixNano: harness.now.Add(remoteauth.DirectSignalingMaxTTL).UnixNano(),
		GrantId:           "grant-direct-test", GrantExpiresAtUnixNano: harness.now.Add(time.Hour).UnixNano(),
	}
}

func directRequestAt(harness *directServerHarness, id string, now time.Time) *remoteauthpb.DirectSignalingRequestV2 {
	request := harness.request(id)
	request.IssuedAtUnixNano = now.UnixNano()
	request.ExpiresAtUnixNano = now.Add(remoteauth.DirectSignalingMaxTTL).UnixNano()
	request.GrantExpiresAtUnixNano = now.Add(time.Hour).UnixNano()
	return request
}

func seedDirectConsumed(server *DirectServer, requestID string, expiresAt time.Time) {
	server.consumed[requestID] = expiresAt
	heap.Push(&server.consumedExpiry, directConsumedExpiry{requestID: requestID, expiresAt: expiresAt})
}

func readDirectResponse(t *testing.T, connection net.Conn) *remoteauthpb.DirectSignalingResponseV2 {
	t.Helper()
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	response := &remoteauthpb.DirectSignalingResponseV2{}
	if err := directsignal.ReadMessage(connection, response); err != nil {
		t.Fatal(err)
	}
	return response
}

func assertDirectErrorCode(t *testing.T, response *remoteauthpb.DirectSignalingResponseV2, want remoteauthpb.DirectSignalingErrorCode) {
	t.Helper()
	if response.GetError() == nil || response.GetError().GetCode() != want {
		t.Fatalf("direct signaling response = %#v, want error code %s", response, want)
	}
}

func assertDirectPanicFailure(t *testing.T, response *remoteauthpb.DirectSignalingResponseV2) {
	t.Helper()
	assertDirectErrorCode(t, response, remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_INTERNAL)
	if got := response.GetError().GetMessage(); got != "direct signaling server internal failure" {
		t.Fatalf("panic response message = %q", got)
	}
}

func waitDirectServerState(t *testing.T, server *DirectServer, total, perIPMax int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		server.mu.Lock()
		gotTotal := server.preAuthTotal
		gotPerIPMax := 0
		for _, count := range server.preAuthByIP {
			if count > gotPerIPMax {
				gotPerIPMax = count
			}
		}
		server.mu.Unlock()
		if gotTotal == total && gotPerIPMax == perIPMax {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("pre-auth state total=%d max-per-IP=%d, want %d/%d", gotTotal, gotPerIPMax, total, perIPMax)
		}
		time.Sleep(time.Millisecond)
	}
}

func waitAtomicInt32(t *testing.T, value *atomic.Int32, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for value.Load() != int32(want) {
		if time.Now().After(deadline) {
			t.Fatalf("atomic value = %d, want %d", value.Load(), want)
		}
		time.Sleep(time.Millisecond)
	}
}

func assertDirectServerSlotsEmpty(t *testing.T, server *DirectServer) {
	t.Helper()
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.preAuthTotal != 0 || len(server.preAuthByIP) != 0 || len(server.peerSlots) != 0 || len(server.conns) != 0 {
		t.Fatalf("server retained pre-auth=%d per-IP=%v peers=%d conns=%d", server.preAuthTotal, server.preAuthByIP, len(server.peerSlots), len(server.conns))
	}
}

func assertPreAuthSlotsReusable(t *testing.T, server *DirectServer) {
	t.Helper()
	releases := make([]func(), 0, directSignalingPreAuthLimit)
	for index := 0; index < directSignalingPreAuthPerIPLimit; index++ {
		release, acquired := server.tryAcquirePreAuth(directTestAddress(fmt.Sprintf("192.0.2.90:%d", 9000+index)))
		if !acquired {
			t.Fatalf("reacquire same-IP pre-auth slot %d", index)
		}
		releases = append(releases, release)
	}
	if _, acquired := server.tryAcquirePreAuth(directTestAddress("[::ffff:192.0.2.90]:9999")); acquired {
		t.Fatal("reused pre-auth limiter admitted a ninth normalized source IP")
	}
	for index := directSignalingPreAuthPerIPLimit; index < directSignalingPreAuthLimit; index++ {
		release, acquired := server.tryAcquirePreAuth(directTestAddress(fmt.Sprintf("[2001:db8:2::%x]:9000", index)))
		if !acquired {
			t.Fatalf("reacquire global pre-auth slot %d", index)
		}
		releases = append(releases, release)
	}
	if _, acquired := server.tryAcquirePreAuth(directTestAddress("[2001:db8:2::ffff]:9000")); acquired {
		t.Fatal("reused pre-auth limiter admitted a 65th connection")
	}
	for _, release := range releases {
		release()
		release()
	}
	waitDirectServerState(t, server, 0, 0)
}

func assertPeerSlotsReusable(t *testing.T, server *DirectServer) {
	t.Helper()
	releases := make([]func(), 0, directSignalingPeerLimit)
	for index := 0; index < directSignalingPeerLimit; index++ {
		release, acquired := server.tryAcquirePeer()
		if !acquired {
			t.Fatalf("reacquire peer slot %d", index)
		}
		releases = append(releases, release)
	}
	if _, acquired := server.tryAcquirePeer(); acquired {
		t.Fatal("reused peer limiter admitted a 33rd peer")
	}
	for _, release := range releases {
		release()
		release()
	}
	if got := len(server.peerSlots); got != 0 {
		t.Fatalf("peer slots after reuse = %d", got)
	}
}

type directTestHandler struct{}

func (directTestHandler) GrantActive(string, time.Time) bool                { return true }
func (directTestHandler) PairingClaimActive([]byte, []byte, time.Time) bool { return true }

func (directTestHandler) ServeDataChannel(context.Context, transport.Transport, string) error {
	return nil
}

type directRejectingAdmission struct{}

func (directRejectingAdmission) GrantActive(string, time.Time) bool                { return false }
func (directRejectingAdmission) PairingClaimActive([]byte, []byte, time.Time) bool { return false }

type directTestListener struct {
	connections chan net.Conn
	closed      chan struct{}
	closeOnce   sync.Once
	closes      atomic.Int32
}

func newDirectTestListener() *directTestListener {
	return &directTestListener{connections: make(chan net.Conn), closed: make(chan struct{})}
}

func (listener *directTestListener) Accept() (net.Conn, error) {
	select {
	case connection := <-listener.connections:
		return connection, nil
	case <-listener.closed:
		return nil, net.ErrClosed
	}
}

func (listener *directTestListener) Close() error {
	listener.closeOnce.Do(func() {
		listener.closes.Add(1)
		close(listener.closed)
	})
	return nil
}

func (listener *directTestListener) Addr() net.Addr { return directTestAddress("127.0.0.1:0") }

type directTestTCPMux struct {
	closes atomic.Int32
}

func (mux *directTestTCPMux) Close() error {
	mux.closes.Add(1)
	return nil
}
func (*directTestTCPMux) GetConnByUfrag(string, bool, net.IP) (net.PacketConn, error) {
	return nil, net.ErrClosed
}
func (*directTestTCPMux) RemoveConnByUfrag(string) {}

type directTestConn struct {
	net.Conn
	remoteAddress  net.Addr
	reads          atomic.Int32
	readReturns    atomic.Int32
	closes         atomic.Int32
	closePanic     any
	closeStarted   chan struct{}
	closeResume    <-chan struct{}
	closeStartOnce sync.Once
}

func (connection *directTestConn) Read(buffer []byte) (int, error) {
	connection.reads.Add(1)
	count, err := connection.Conn.Read(buffer)
	connection.readReturns.Add(1)
	return count, err
}

func (connection *directTestConn) Close() error {
	connection.closes.Add(1)
	err := connection.Conn.Close()
	if connection.closeStarted != nil {
		connection.closeStartOnce.Do(func() { close(connection.closeStarted) })
	}
	if connection.closeResume != nil {
		<-connection.closeResume
	}
	if connection.closePanic != nil {
		panic(connection.closePanic)
	}
	return err
}

func (connection *directTestConn) RemoteAddr() net.Addr { return connection.remoteAddress }

type directTestAddress string

func (address directTestAddress) Network() string { return "tcp" }
func (address directTestAddress) String() string  { return string(address) }

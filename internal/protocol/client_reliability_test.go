package protocol

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport/memory"
	"google.golang.org/protobuf/proto"
)

func TestApplicationEventsReplaysCapacityDuringConcurrentPublication(t *testing.T) {
	clientTransport, serverTransport := memory.NewPair()
	client := NewClient(clientTransport)
	defer client.Close()
	defer serverTransport.Close()
	completeClientHello(t, client, serverTransport)

	client.mu.Lock()
	for index := range 64 {
		client.pendingApplicationEvents = append(client.pendingApplicationEvents, reliabilityEvent(fmt.Sprintf("pending-%d", index)))
	}
	client.mu.Unlock()

	published := make(chan error, 1)
	go func() {
		for {
			client.mu.Lock()
			registered := len(client.applicationEventSubscribers) > 0
			client.mu.Unlock()
			if registered {
				break
			}
			time.Sleep(time.Millisecond)
		}
		for index := range 64 {
			if err := client.publishApplicationEvent(reliabilityEvent(fmt.Sprintf("concurrent-%d", index))); err != nil {
				published <- err
				return
			}
		}
		published <- nil
	}()
	events, err := client.ApplicationEvents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-published:
		if err != nil {
			t.Fatalf("concurrent publication failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("concurrent publication blocked during event registration")
	}
	for index := range 64 {
		select {
		case event := <-events:
			if event == nil {
				t.Fatalf("replayed event %d is nil", index)
			}
		case <-time.After(time.Second):
			t.Fatalf("event replay blocked at capacity index %d", index)
		}
	}
}

func TestClientApplicationEventOverflowClosesMaliciousPeer(t *testing.T) {
	clientTransport, serverTransport := memory.NewPair()
	client := NewClient(clientTransport)
	defer client.Close()
	defer serverTransport.Close()
	completeClientHello(t, client, serverTransport)
	for index := range 65 {
		payload, err := proto.Marshal(reliabilityEvent(fmt.Sprintf("event-%d", index)))
		if err != nil {
			t.Fatal(err)
		}
		if err := sendTestFrame(serverTransport, 0, wire.TypeEvent, payload); err != nil {
			t.Fatal(err)
		}
	}
	assertPeerFailure(t, client, ProtocolErrorCodeResourceExhausted)
}

func TestClientPendingStreamBoundsCloseMaliciousPeer(t *testing.T) {
	tests := []struct {
		name   string
		limits clientPendingLimits
		frames []pendingTestFrame
	}{
		{
			name:   "channels",
			limits: clientPendingLimits{channels: 2, frames: 8, framesPerChannel: 8, bytes: 64},
			frames: []pendingTestFrame{{channel: 1, payload: []byte("a")}, {channel: 2, payload: []byte("b")}, {channel: 3, payload: []byte("c")}},
		},
		{
			name:   "frames",
			limits: clientPendingLimits{channels: 2, frames: 2, framesPerChannel: 2, bytes: 64},
			frames: []pendingTestFrame{{channel: 1, payload: []byte("a")}, {channel: 1, payload: []byte("b")}, {channel: 1, payload: []byte("c")}},
		},
		{
			name:   "bytes",
			limits: clientPendingLimits{channels: 2, frames: 8, framesPerChannel: 8, bytes: 3},
			frames: []pendingTestFrame{{channel: 1, payload: []byte("ab")}, {channel: 1, payload: []byte("cd")}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientTransport, serverTransport := memory.NewPair()
			client := newClientWithPendingLimits(clientTransport, test.limits)
			defer client.Close()
			defer serverTransport.Close()
			completeClientHello(t, client, serverTransport)
			for _, frame := range test.frames {
				if err := sendTestFrame(serverTransport, frame.channel, wire.TypeFileData, frame.payload); err != nil {
					t.Fatal(err)
				}
			}
			assertPeerFailure(t, client, ProtocolErrorCodeResourceExhausted)
		})
	}
}

func TestClientRejectsDuplicateAndMalformedControlFrames(t *testing.T) {
	t.Run("duplicate hello", func(t *testing.T) {
		clientTransport, serverTransport := memory.NewPair()
		client := NewClient(clientTransport)
		defer client.Close()
		defer serverTransport.Close()
		completeClientHello(t, client, serverTransport)
		hello, err := EncodeHelloPayload(Hello{Version: wire.Version, Server: "duplicate"})
		if err != nil {
			t.Fatal(err)
		}
		if err := sendTestFrame(serverTransport, 0, wire.TypeHello, hello); err != nil {
			t.Fatal(err)
		}
		assertPeerFailure(t, client, ProtocolErrorCodeBadRequest)
	})

	t.Run("duplicate response", func(t *testing.T) {
		clientTransport, serverTransport := memory.NewPair()
		client := NewClient(clientTransport)
		defer client.Close()
		defer serverTransport.Close()
		completeClientHello(t, client, serverTransport)
		client.mu.Lock()
		client.waiters[7] = &responseWaiter{ch: make(chan result, 1)}
		client.mu.Unlock()
		payload, err := EncodeResponsePayload(Response{ID: 7, Result: []byte("ok")})
		if err != nil {
			t.Fatal(err)
		}
		if err := sendTestFrame(serverTransport, 0, wire.TypeResponse, payload); err != nil {
			t.Fatal(err)
		}
		if err := sendTestFrame(serverTransport, 0, wire.TypeResponse, payload); err != nil {
			t.Fatal(err)
		}
		assertPeerFailure(t, client, ProtocolErrorCodeBadRequest)
	})

	t.Run("malformed response", func(t *testing.T) {
		clientTransport, serverTransport := memory.NewPair()
		client := NewClient(clientTransport)
		defer client.Close()
		defer serverTransport.Close()
		completeClientHello(t, client, serverTransport)
		if err := sendTestFrame(serverTransport, 0, wire.TypeResponse, []byte{0xff}); err != nil {
			t.Fatal(err)
		}
		assertPeerFailure(t, client, ProtocolErrorCodeBadRequest)
	})
}

func TestClientStreamOverflowClosesAffectedStreamWithTypedError(t *testing.T) {
	stream := newClientStreamWithConfig(1, 0)
	frames := stream.channel()
	stream.send(StreamFrame{Type: wire.TypeFileData, Payload: []byte("first")})
	waitForStreamQueueLength(t, stream, 0)
	stream.send(StreamFrame{Type: wire.TypeFileData, Payload: []byte("second")})
	stream.send(StreamFrame{Type: wire.TypeFileData, Payload: []byte("overflow")})

	if first := receiveReliabilityStreamFrame(t, frames); first.Type != wire.TypeFileData || string(first.Payload) != "first" {
		t.Fatalf("first stream frame = %#v", first)
	}
	failure := receiveReliabilityStreamFrame(t, frames)
	if failure.Type != wire.TypeError {
		t.Fatalf("overflow frame type = %d", failure.Type)
	}
	message, err := DecodeErrorPayload(failure.Payload)
	if err != nil || message.Error.Code != ProtocolErrorCodeResourceExhausted {
		t.Fatalf("overflow error = %#v, %v", message, err)
	}
	select {
	case _, ok := <-frames:
		if ok {
			t.Fatal("affected stream remained open after overflow")
		}
	case <-time.After(time.Second):
		t.Fatal("affected stream did not close after overflow")
	}
}

func TestClientStreamPayloadBudgetRejectsBeforeRetentionAndReleasesOnHandoff(t *testing.T) {
	shared := newStreamPayloadBudget(64)
	stream := newClientStreamWithLimits(256, 0, 8, shared)
	frames := stream.channel()

	stream.send(StreamFrame{Type: wire.TypeFileData, Payload: []byte("12345678")})
	waitForPayloadBudget(t, shared, 8)
	waitForStreamQueueLength(t, stream, 0)
	stream.send(StreamFrame{Type: wire.TypeFileData, Payload: []byte("overflow")})
	if got := retainedPayloadBytes(shared); got != 8 {
		t.Fatalf("rejected frame changed retained bytes: got=%d want=8", got)
	}

	if first := receiveReliabilityStreamFrame(t, frames); string(first.Payload) != "12345678" {
		t.Fatalf("first retained frame payload = %q", first.Payload)
	}
	waitForPayloadBudget(t, shared, 0)
	failure := receiveReliabilityStreamFrame(t, frames)
	if failure.Type != wire.TypeError {
		t.Fatalf("payload overflow frame type = %d", failure.Type)
	}
	message, err := DecodeErrorPayload(failure.Payload)
	if err != nil || message.Error.Code != ProtocolErrorCodeResourceExhausted {
		t.Fatalf("payload overflow error = %#v, %v", message, err)
	}
}

func TestClientStreamPayloadBudgetQueueResetAndCloseReleaseExactly(t *testing.T) {
	t.Run("queue reset preserves only in-flight reservation", func(t *testing.T) {
		shared := newStreamPayloadBudget(64)
		stream := newClientStreamWithLimits(256, 0, 6, shared)
		frames := stream.channel()
		stream.send(StreamFrame{Type: wire.TypeFileData, Payload: []byte("one")})
		waitForStreamQueueLength(t, stream, 0)
		stream.send(StreamFrame{Type: wire.TypeFileData, Payload: []byte("two")})
		waitForPayloadBudget(t, shared, 6)
		stream.send(StreamFrame{Type: wire.TypeFileData, Payload: []byte("x")})
		waitForPayloadBudget(t, shared, 3)
		if frame := receiveReliabilityStreamFrame(t, frames); string(frame.Payload) != "one" {
			t.Fatalf("in-flight frame payload = %q", frame.Payload)
		}
		waitForPayloadBudget(t, shared, 0)
		if frame := receiveReliabilityStreamFrame(t, frames); frame.Type != wire.TypeError {
			t.Fatalf("queue reset terminal frame type = %d", frame.Type)
		}
	})

	t.Run("close drops in-flight reservation", func(t *testing.T) {
		shared := newStreamPayloadBudget(64)
		stream := newClientStreamWithLimits(256, 0, 8, shared)
		stream.send(StreamFrame{Type: wire.TypeFileData, Payload: []byte("data")})
		waitForPayloadBudget(t, shared, 4)
		stream.close()
		waitForPayloadBudget(t, shared, 0)
		select {
		case _, ok := <-stream.channel():
			if ok {
				t.Fatal("closed stream delivered a dropped frame")
			}
		case <-time.After(time.Second):
			t.Fatal("closed stream channel did not close")
		}
	})
}

func TestClientActiveStreamsSharePayloadBudget(t *testing.T) {
	clientTransport, serverTransport := memory.NewPair()
	client := NewClient(clientTransport)
	defer client.Close()
	defer serverTransport.Close()

	shared := newStreamPayloadBudget(10)
	client.mu.Lock()
	client.streamPayloadBudget = shared
	client.mu.Unlock()
	framesA, _ := client.Stream(1)
	framesB, _ := client.Stream(2)
	client.mu.Lock()
	streamA := client.streams[1]
	streamB := client.streams[2]
	client.mu.Unlock()

	streamA.send(StreamFrame{Type: wire.TypeFileData, Payload: []byte("123456")})
	waitForPayloadBudget(t, shared, 6)
	streamB.send(StreamFrame{Type: wire.TypeFileData, Payload: []byte("12345")})
	if got := retainedPayloadBytes(shared); got != 6 {
		t.Fatalf("shared budget retained bytes = %d want=6", got)
	}
	if frame := receiveReliabilityStreamFrame(t, framesA); string(frame.Payload) != "123456" {
		t.Fatalf("first stream payload = %q", frame.Payload)
	}
	waitForPayloadBudget(t, shared, 0)
	if frame := receiveReliabilityStreamFrame(t, framesB); frame.Type != wire.TypeError {
		t.Fatalf("shared budget overflow frame type = %d", frame.Type)
	}
}

func TestClientPTYByteOverflowCountsTriggeringAndSubsequentFrames(t *testing.T) {
	shared := newStreamPayloadBudget(64)
	stream := newClientStreamWithLimits(256, 0, 5, shared)
	frames := stream.channel()
	stream.send(StreamFrame{Type: wire.TypePTYOutput, Payload: []byte("abc")})
	stream.send(StreamFrame{Type: wire.TypePTYOutput, Payload: []byte("defg")})
	stream.send(StreamFrame{Type: wire.TypePTYOutput, Payload: []byte("hi")})

	if frame := receiveReliabilityStreamFrame(t, frames); string(frame.Payload) != "abc" {
		t.Fatalf("PTY prefix payload = %q", frame.Payload)
	}
	waitForPayloadBudget(t, shared, 0)
	syncLost := receiveReliabilityStreamFrame(t, frames)
	dropped, err := wire.DecodeSyncLostPayload(syncLost.Payload)
	if syncLost.Type != wire.TypeSyncLost || err != nil || dropped != 6 {
		t.Fatalf("PTY sync lost frame = %#v dropped=%d err=%v", syncLost, dropped, err)
	}
}

func TestClientStreamBudgetAccountingIsRaceSafeDuringClose(t *testing.T) {
	shared := newStreamPayloadBudget(64 << 10)
	stream := newClientStreamWithLimits(256, 0, 64<<10, shared)
	drained := make(chan struct{})
	go func() {
		for range stream.channel() {
		}
		close(drained)
	}()
	var senders sync.WaitGroup
	for range 16 {
		senders.Add(1)
		go func() {
			defer senders.Done()
			for range 100 {
				stream.send(StreamFrame{Type: wire.TypeFileData, Payload: make([]byte, 32)})
			}
		}()
	}
	stream.close()
	senders.Wait()
	select {
	case <-drained:
	case <-time.After(time.Second):
		t.Fatal("concurrent stream close did not stop consumer")
	}
	waitForPayloadBudget(t, shared, 0)
	waitForPayloadBudget(t, stream.payloadBudget, 0)
}

func TestClientAbandonedLateResponseReleasesWaiterCapacity(t *testing.T) {
	clientTransport, serverTransport := memory.NewPair()
	client := NewClient(clientTransport)
	defer client.Close()
	defer serverTransport.Close()
	completeClientHello(t, client, serverTransport)

	requestCtx, cancel := context.WithCancel(context.Background())
	requestDone := make(chan error, 1)
	go func() {
		_, err := client.doApplicationRequest(requestCtx, &apipb.CommandEnvelope{}, false)
		requestDone <- err
	}()
	frame, err := serverTransport.Recv()
	if err != nil {
		t.Fatal(err)
	}
	channel, typ, payload, err := wire.DecodeFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	if channel != 0 || typ != wire.TypeRequest {
		t.Fatalf("request frame = channel:%d type:%d", channel, typ)
	}
	request, err := DecodeRequestPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case err := <-requestDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled request error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled request did not return")
	}
	waitForClientAccounting(t, client, func() bool {
		return len(client.waiters) == 0 && len(client.abandonedWaiters) == 1
	})
	cancelFrame, err := serverTransport.Recv()
	if err != nil {
		t.Fatal(err)
	}
	channel, typ, payload, err = wire.DecodeFrame(cancelFrame)
	if err != nil {
		t.Fatal(err)
	}
	if channel != 0 || typ != wire.TypeRequestCancel {
		t.Fatalf("cancel frame = channel:%d type:%d", channel, typ)
	}
	if canceledID, err := DecodeRequestCancelPayload(payload); err != nil || canceledID != request.ID {
		t.Fatalf("cancel request ID = %d err=%v, want %d", canceledID, err, request.ID)
	}
	response, err := EncodeResponsePayload(Response{ID: request.ID, Result: []byte("late")})
	if err != nil {
		t.Fatal(err)
	}
	if err := sendTestFrame(serverTransport, 0, wire.TypeResponse, response); err != nil {
		t.Fatal(err)
	}
	waitForClientAccounting(t, client, func() bool { return len(client.abandonedWaiters) == 0 })
	select {
	case <-client.Done():
		t.Fatalf("late abandoned response closed client: %v", client.Err())
	default:
	}
}

func TestClientPendingAccountingReleasesChannelFrameAndByteCapacity(t *testing.T) {
	clientTransport, serverTransport := memory.NewPair()
	client := newClientWithPendingLimits(clientTransport, clientPendingLimits{channels: 1, frames: 2, framesPerChannel: 2, bytes: 4})
	defer client.Close()
	defer serverTransport.Close()

	client.mu.Lock()
	if err := client.queuePendingFrameLocked(client.pending, 7, wire.TypeFileData, []byte("ab")); err != nil {
		client.mu.Unlock()
		t.Fatal(err)
	}
	frames := client.takePendingFramesLocked(client.pending, 7)
	if len(frames) != 1 || client.pendingFrameCount != 0 || client.pendingByteCount != 0 || len(client.pending) != 0 {
		client.mu.Unlock()
		t.Fatalf("pending accounting after bind = frames:%d bytes:%d channels:%d", client.pendingFrameCount, client.pendingByteCount, len(client.pending))
	}
	if err := client.queuePendingFrameLocked(client.reused, 8, wire.TypeFileData, []byte("cd")); err != nil {
		client.mu.Unlock()
		t.Fatalf("released channel capacity was not reusable: %v", err)
	}
	client.discardPendingFramesLocked(client.reused, 8)
	if client.pendingFrameCount != 0 || client.pendingByteCount != 0 || len(client.reused) != 0 {
		client.mu.Unlock()
		t.Fatalf("pending accounting after discard = frames:%d bytes:%d channels:%d", client.pendingFrameCount, client.pendingByteCount, len(client.reused))
	}
	if err := client.queuePendingFrameLocked(client.pending, 9, wire.TypeFileData, []byte("ef")); err != nil {
		client.mu.Unlock()
		t.Fatal(err)
	}
	client.mu.Unlock()

	client.failAll(errors.New("test peer failure"))
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.pendingFrameCount != 0 || client.pendingByteCount != 0 || len(client.pending) != 0 || len(client.reused) != 0 {
		t.Fatalf("pending accounting after failure = frames:%d bytes:%d pending:%d reused:%d", client.pendingFrameCount, client.pendingByteCount, len(client.pending), len(client.reused))
	}
}

func TestClientDropsLateBrowserFramesAfterStreamClose(t *testing.T) {
	clientTransport, serverTransport := memory.NewPair()
	client := NewClient(clientTransport)
	defer client.Close()
	defer serverTransport.Close()

	const channel = uint16(7)
	client.bindApplicationAttachment(channel, &apipb.ResourceHandle{
		OpaqueToken: []byte{0, byte(channel), 1},
		Kind:        apipb.ResourceKind_RESOURCE_KIND_BROWSER_PROXY,
	})
	_, stop := client.Stream(channel)
	stop()

	if err := sendTestFrame(serverTransport, channel, wire.TypeBrowserData, []byte("late")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-client.Done():
		t.Fatalf("late browser frame closed client: %v", client.Err())
	case <-time.After(100 * time.Millisecond):
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.pending) != 0 || len(client.reused) != 0 || client.pendingFrameCount != 0 || client.pendingByteCount != 0 {
		t.Fatalf("late browser frame was retained: pending=%d reused=%d frames=%d bytes=%d", len(client.pending), len(client.reused), client.pendingFrameCount, client.pendingByteCount)
	}
}

type pendingTestFrame struct {
	channel uint16
	payload []byte
}

func completeClientHello(t *testing.T, client *Client, serverTransport *memory.Transport) {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		done <- client.Hello(context.Background(), Hello{Version: wire.Version, Client: "reliability-test"})
	}()
	if err := expectHelloAndRespond(serverTransport); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("client Hello did not complete")
	}
}

func reliabilityEvent(id string) *apipb.EventEnvelope {
	return &apipb.EventEnvelope{
		EventId: id,
		Event: &apipb.EventEnvelope_TerminalLifecycle{TerminalLifecycle: &apipb.TerminalLifecycleEvent{
			Terminal: &apipb.TerminalInfo{Ref: &apipb.TerminalRef{EndpointId: "local", TerminalId: id}},
		}},
	}
}

func assertPeerFailure(t *testing.T, client *Client, code int) {
	t.Helper()
	select {
	case <-client.Done():
	case <-time.After(time.Second):
		t.Fatal("malicious peer did not close")
	}
	var peerErr *PeerError
	if err := client.Err(); !errors.As(err, &peerErr) || peerErr.Code != code {
		t.Fatalf("client error = %T %v, want peer code %d", err, err, code)
	}
}

func waitForClientAccounting(t *testing.T, client *Client, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		client.mu.Lock()
		complete := ready()
		client.mu.Unlock()
		if complete {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("client accounting did not reach expected state")
}

func waitForStreamQueueLength(t *testing.T, stream *clientStream, length int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		stream.mu.Lock()
		got := len(stream.queue)
		stream.mu.Unlock()
		if got == length {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("stream queue did not reach length %d", length)
}

func receiveReliabilityStreamFrame(t *testing.T, frames <-chan StreamFrame) StreamFrame {
	t.Helper()
	select {
	case frame, ok := <-frames:
		if !ok {
			t.Fatal("stream closed before expected frame")
		}
		return frame
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stream frame")
		return StreamFrame{}
	}
}

func retainedPayloadBytes(budget *streamPayloadBudget) int {
	budget.mu.Lock()
	defer budget.mu.Unlock()
	return budget.retained
}

func waitForPayloadBudget(t *testing.T, budget *streamPayloadBudget, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if got := retainedPayloadBytes(budget); got == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("payload budget retained bytes = %d want=%d", retainedPayloadBytes(budget), want)
}

package datachannel

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/wire"
)

func TestTransportRoundTripsProtocolFrames(t *testing.T) {
	clientChannel, serverChannel := newFakeChannelPair()
	client := New(clientChannel)
	server := New(serverChannel)
	defer client.Close()
	defer server.Close()

	want := []byte("protocol-frame")
	if err := client.Send(want); err != nil {
		t.Fatalf("send protocol frame: %v", err)
	}
	got, err := server.Recv()
	if err != nil {
		t.Fatalf("recv protocol frame: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("protocol frame mismatch: got %q want %q", got, want)
	}
	want[0] = 'X'
	if bytes.Equal(got, want) {
		t.Fatal("transport must copy outbound protocol frames before handing them to the data channel")
	}
}

func TestTransportRemoteCloseUnblocksRecv(t *testing.T) {
	clientChannel, serverChannel := newFakeChannelPair()
	client := New(clientChannel)
	server := New(serverChannel)

	recvDone := make(chan error, 1)
	go func() {
		_, err := server.Recv()
		recvDone <- err
	}()
	if err := client.Close(); err != nil {
		t.Fatalf("close client transport: %v", err)
	}

	select {
	case err := <-recvDone:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("remote close should unblock recv with EOF, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("remote close did not unblock recv")
	}
	select {
	case <-server.Done():
	case <-time.After(time.Second):
		t.Fatal("remote close did not close Done")
	}
}

func TestTransportSendRejectsOversizedLogicalFrameBeforeClone(t *testing.T) {
	clientChannel, _ := newFakeChannelPair()
	transport := New(clientChannel)
	defer transport.Close()
	frame := make([]byte, maxLogicalFrameSize+1)
	var err error
	allocations := testing.AllocsPerRun(100, func() {
		err = transport.Send(frame)
	})
	if !errors.Is(err, wire.ErrFrameTooLarge) {
		t.Fatalf("oversized logical frame error = %v", err)
	}
	clientChannel.mu.Lock()
	sendCalls := clientChannel.sendCalls
	clientChannel.mu.Unlock()
	if sendCalls != 0 {
		t.Fatalf("oversized logical frame reached Channel.Send %d times", sendCalls)
	}
	if allocations != 0 {
		t.Fatalf("oversized logical frame rejection allocated %.2f times", allocations)
	}
}

func TestTransportPreAuthReceiveQueueByteOverflowClosesChannel(t *testing.T) {
	sender, receiverChannel := newFakeChannelPair()
	receiver := New(receiverChannel)
	chunk := make([]byte, maxReceiveQueuedBytes/2)
	if err := sender.Send(chunk); err != nil {
		t.Fatalf("first pre-auth message: %v", err)
	}
	if err := sender.Send(chunk); err != nil {
		t.Fatalf("second pre-auth message: %v", err)
	}
	if got := receiveQueuedBytes(receiver); got != maxReceiveQueuedBytes {
		t.Fatalf("pre-auth queued bytes = %d want=%d", got, maxReceiveQueuedBytes)
	}
	if err := sender.Send([]byte{1}); err != nil {
		t.Fatalf("overflowing pre-auth message delivery: %v", err)
	}
	select {
	case <-receiver.Done():
	case <-time.After(time.Second):
		t.Fatal("pre-auth queue overflow did not close transport")
	}
	receiverChannel.mu.Lock()
	closed := receiverChannel.closed
	receiverChannel.mu.Unlock()
	if !closed {
		t.Fatal("pre-auth queue overflow did not close underlying channel")
	}
	if got := receiveQueuedBytes(receiver); got != 0 {
		t.Fatalf("closed pre-auth queue retained %d bytes", got)
	}
	if _, err := receiver.Recv(); !errors.Is(err, ErrReceiveQueueExhausted) {
		t.Fatalf("Recv after pre-auth overflow = %v", err)
	}
	if err := receiver.Close(); err != nil {
		t.Fatal(err)
	}
	if calls := channelCloseCalls(receiverChannel); calls != 1 {
		t.Fatalf("pre-auth overflow channel close calls = %d want=1", calls)
	}
}

func TestTransportPreAuthReceiveQueueFrameOverflowReturnsStableError(t *testing.T) {
	sender, receiverChannel := newFakeChannelPair()
	receiver := New(receiverChannel)
	for range defaultReceiveQueueCapacity + 1 {
		if err := sender.Send(nil); err != nil {
			t.Fatalf("deliver pre-auth frame: %v", err)
		}
	}
	if _, err := receiver.Recv(); !errors.Is(err, ErrReceiveQueueExhausted) {
		t.Fatalf("Recv after pre-auth frame overflow = %v", err)
	}
	if got := receiveQueuedBytes(receiver); got != 0 {
		t.Fatalf("frame-overflowed pre-auth queue retained %d bytes", got)
	}
	if err := receiver.Close(); err != nil {
		t.Fatal(err)
	}
	if calls := channelCloseCalls(receiverChannel); calls != 1 {
		t.Fatalf("pre-auth frame overflow channel close calls = %d want=1", calls)
	}
}

func TestTransportOversizedInboundFrameClosesBeforeQueueing(t *testing.T) {
	sender, receiverChannel := newFakeChannelPair()
	receiver := New(receiverChannel)
	if err := sender.Send(make([]byte, maxLogicalFrameSize+1)); err != nil {
		t.Fatalf("deliver oversized inbound frame: %v", err)
	}
	select {
	case <-receiver.Done():
	case <-time.After(time.Second):
		t.Fatal("oversized inbound frame did not close transport")
	}
	if got := receiveQueuedBytes(receiver); got != 0 {
		t.Fatalf("oversized inbound frame queued %d bytes", got)
	}
	receiverChannel.mu.Lock()
	closed := receiverChannel.closed
	receiverChannel.mu.Unlock()
	if !closed {
		t.Fatal("oversized inbound frame did not close underlying channel")
	}
	if _, err := receiver.Recv(); !errors.Is(err, wire.ErrFrameTooLarge) {
		t.Fatalf("Recv after oversized inbound frame = %v", err)
	}
	if err := receiver.Close(); err != nil {
		t.Fatal(err)
	}
	if calls := channelCloseCalls(receiverChannel); calls != 1 {
		t.Fatalf("oversized inbound channel close calls = %d want=1", calls)
	}
}

func TestTransportReceiveQueueReleasesBytesOnDequeueAndClose(t *testing.T) {
	t.Run("dequeue", func(t *testing.T) {
		sender, receiverChannel := newFakeChannelPair()
		receiver := New(receiverChannel)
		defer receiver.Close()
		payload := make([]byte, 1<<20)
		if err := sender.Send(payload); err != nil {
			t.Fatal(err)
		}
		if got := receiveQueuedBytes(receiver); got != len(payload) {
			t.Fatalf("queued bytes before dequeue = %d want=%d", got, len(payload))
		}
		if _, err := receiver.Recv(); err != nil {
			t.Fatal(err)
		}
		if got := receiveQueuedBytes(receiver); got != 0 {
			t.Fatalf("queued bytes after dequeue = %d", got)
		}
	})

	t.Run("close", func(t *testing.T) {
		sender, receiverChannel := newFakeChannelPair()
		receiver := New(receiverChannel)
		if err := sender.Send(make([]byte, 1<<20)); err != nil {
			t.Fatal(err)
		}
		if err := receiver.Close(); err != nil {
			t.Fatal(err)
		}
		if got := receiveQueuedBytes(receiver); got != 0 {
			t.Fatalf("queued bytes after close = %d", got)
		}
	})
}

func TestTransportReceiveAccountingIsRaceSafe(t *testing.T) {
	channel, _ := newFakeChannelPair()
	transport := New(channel)
	channel.mu.Lock()
	handler := channel.messageHandler
	channel.mu.Unlock()
	if handler == nil {
		t.Fatal("message handler was not installed")
	}

	var handlers sync.WaitGroup
	for range 64 {
		handlers.Add(1)
		go func() {
			defer handlers.Done()
			handler(make([]byte, 64<<10))
		}()
	}
	recvDone := make(chan struct{})
	go func() {
		for {
			if _, err := transport.Recv(); err != nil {
				close(recvDone)
				return
			}
		}
	}()
	closeDone := make(chan error, 1)
	go func() { closeDone <- transport.Close() }()
	handlers.Wait()
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("concurrent close: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("concurrent close blocked")
	}
	select {
	case <-recvDone:
	case <-time.After(time.Second):
		t.Fatal("concurrent Recv did not unblock")
	}
	if got := receiveQueuedBytes(transport); got != 0 {
		t.Fatalf("concurrent close retained %d bytes", got)
	}
}

func TestTransportSendWaitsForBufferedAmountLow(t *testing.T) {
	clientChannel, serverChannel := newFakeChannelPair()
	clientChannel.setBufferedAmount(defaultSendBufferHigh + 1)
	client := New(clientChannel)
	server := New(serverChannel)
	defer client.Close()
	defer server.Close()

	sendDone := make(chan error, 1)
	go func() {
		sendDone <- client.Send([]byte("frame"))
	}()
	select {
	case err := <-sendDone:
		t.Fatalf("send should wait for buffered amount low, got %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	clientChannel.setBufferedAmount(defaultSendBufferLow)
	clientChannel.signalBufferedAmountLow()
	select {
	case err := <-sendDone:
		if err != nil {
			t.Fatalf("send after drain: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("buffer drain did not resume send")
	}
}

func TestTransportCloseUnblocksInFlightChannelSend(t *testing.T) {
	channel := newBlockingSendChannel()
	transport := New(channel)
	sendDone := make(chan error, 1)
	go func() {
		sendDone <- transport.Send([]byte("blocked-auth-frame"))
	}()
	select {
	case <-channel.sendStarted:
	case <-time.After(time.Second):
		t.Fatal("channel Send did not start")
	}
	closeDone := make(chan error, 1)
	go func() {
		closeDone <- transport.Close()
	}()
	select {
	case err := <-sendDone:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("closed in-flight Send error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("transport Close did not unblock in-flight Channel.Send")
	}
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("transport Close error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("transport Close remained blocked on sendMu")
	}
}

func TestTransportCloseWaitsForInFlightChannelSendToExit(t *testing.T) {
	channel := newBlockingSendChannel()
	sendRelease := make(chan struct{})
	channel.sendRelease = sendRelease
	var releaseOnce sync.Once
	releaseSend := func() { releaseOnce.Do(func() { close(sendRelease) }) }
	t.Cleanup(releaseSend)

	transport := New(channel)
	sendDone := make(chan error, 1)
	go func() {
		sendDone <- transport.Send([]byte("blocked-auth-frame"))
	}()
	select {
	case <-channel.sendStarted:
	case <-time.After(time.Second):
		t.Fatal("channel Send did not start")
	}

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- transport.Close()
	}()
	select {
	case <-channel.sendUnblocked:
	case <-time.After(time.Second):
		t.Fatal("channel Close did not unblock the in-flight Send")
	}
	select {
	case err := <-closeDone:
		t.Fatalf("transport Close returned before the in-flight Send exited: %v", err)
	case <-time.After(30 * time.Millisecond):
	}

	releaseSend()
	select {
	case err := <-sendDone:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("closed in-flight Send error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("in-flight Send did not exit after release")
	}
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("transport Close error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("transport Close did not return after the in-flight Send exited")
	}
}

func TestTransportDrainWaitsForQueuedOutboundMessages(t *testing.T) {
	clientChannel, serverChannel := newFakeChannelPair()
	clientChannel.setBufferedAmount(1)
	client := New(clientChannel)
	server := New(serverChannel)
	defer client.Close()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	drained := make(chan error, 1)
	go func() { drained <- client.Drain(ctx) }()
	select {
	case err := <-drained:
		t.Fatalf("drain returned before outbound buffer emptied: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	clientChannel.setBufferedAmount(0)
	select {
	case err := <-drained:
		if err != nil {
			t.Fatalf("drain queued outbound messages: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("drain did not observe empty outbound buffer")
	}
}

func TestTransportDrainHasInternalDeadlineWithoutCallerDeadline(t *testing.T) {
	channel, _ := newFakeChannelPair()
	channel.setBufferedAmount(1)
	transport := New(channel)
	defer transport.Close()
	transport.drainTimeout = 20 * time.Millisecond
	started := time.Now()
	if err := transport.Drain(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("drain deadline error = %v", err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("drain ignored its internal deadline")
	}
}

type blockingSendChannel struct {
	mu            sync.Mutex
	closeHandler  func()
	sendOnce      sync.Once
	closeOnce     sync.Once
	sendStarted   chan struct{}
	sendUnblocked chan struct{}
	sendRelease   <-chan struct{}
	closed        chan struct{}
}

func newBlockingSendChannel() *blockingSendChannel {
	return &blockingSendChannel{
		sendStarted:   make(chan struct{}),
		sendUnblocked: make(chan struct{}),
		closed:        make(chan struct{}),
	}
}

func (*blockingSendChannel) SetMessageHandler(func([]byte)) {}

func (channel *blockingSendChannel) SetCloseHandler(handler func()) {
	channel.mu.Lock()
	channel.closeHandler = handler
	channel.mu.Unlock()
}

func (*blockingSendChannel) BufferedAmount() uint64 { return 0 }

func (*blockingSendChannel) SetBufferedAmountLowThreshold(uint64) {}

func (*blockingSendChannel) SetBufferedAmountLowHandler(func()) {}

func (channel *blockingSendChannel) Send([]byte) error {
	channel.sendOnce.Do(func() { close(channel.sendStarted) })
	<-channel.closed
	if channel.sendRelease != nil {
		close(channel.sendUnblocked)
		<-channel.sendRelease
	}
	return io.EOF
}

func (channel *blockingSendChannel) Close() error {
	channel.closeOnce.Do(func() {
		close(channel.closed)
		channel.mu.Lock()
		handler := channel.closeHandler
		channel.mu.Unlock()
		if handler != nil {
			handler()
		}
	})
	return nil
}

type fakeChannel struct {
	mu                sync.Mutex
	peer              *fakeChannel
	messageHandler    func([]byte)
	closeHandler      func()
	bufferedLowHandle func()
	bufferedAmount    uint64
	sendCalls         int
	closeCalls        int
	closed            bool
}

func newFakeChannelPair() (*fakeChannel, *fakeChannel) {
	left := &fakeChannel{}
	right := &fakeChannel{}
	left.peer = right
	right.peer = left
	return left, right
}

func (channel *fakeChannel) SetMessageHandler(handler func([]byte)) {
	channel.mu.Lock()
	channel.messageHandler = handler
	channel.mu.Unlock()
}

func (channel *fakeChannel) SetCloseHandler(handler func()) {
	channel.mu.Lock()
	channel.closeHandler = handler
	channel.mu.Unlock()
}

func (channel *fakeChannel) BufferedAmount() uint64 {
	channel.mu.Lock()
	defer channel.mu.Unlock()
	return channel.bufferedAmount
}

func (channel *fakeChannel) SetBufferedAmountLowThreshold(uint64) {}

func (channel *fakeChannel) SetBufferedAmountLowHandler(handler func()) {
	channel.mu.Lock()
	channel.bufferedLowHandle = handler
	channel.mu.Unlock()
}

func (channel *fakeChannel) Send(payload []byte) error {
	channel.mu.Lock()
	channel.sendCalls++
	if channel.closed {
		channel.mu.Unlock()
		return io.EOF
	}
	peer := channel.peer
	channel.mu.Unlock()
	peer.mu.Lock()
	handler := peer.messageHandler
	closed := peer.closed
	peer.mu.Unlock()
	if closed || handler == nil {
		return io.EOF
	}
	handler(append([]byte(nil), payload...))
	return nil
}

func receiveQueuedBytes(transport *Transport) int {
	transport.recvMu.Lock()
	defer transport.recvMu.Unlock()
	return transport.recvQueuedBytes
}

func (channel *fakeChannel) Close() error {
	channel.mu.Lock()
	channel.closeCalls++
	if channel.closed {
		channel.mu.Unlock()
		return nil
	}
	channel.closed = true
	localHandler := channel.closeHandler
	peer := channel.peer
	channel.mu.Unlock()
	if localHandler != nil {
		localHandler()
	}
	peer.mu.Lock()
	peer.closed = true
	peerHandler := peer.closeHandler
	peer.mu.Unlock()
	if peerHandler != nil {
		peerHandler()
	}
	return nil
}

func channelCloseCalls(channel *fakeChannel) int {
	channel.mu.Lock()
	defer channel.mu.Unlock()
	return channel.closeCalls
}

func (channel *fakeChannel) setBufferedAmount(amount uint64) {
	channel.mu.Lock()
	channel.bufferedAmount = amount
	channel.mu.Unlock()
}

func (channel *fakeChannel) signalBufferedAmountLow() {
	channel.mu.Lock()
	handler := channel.bufferedLowHandle
	channel.mu.Unlock()
	if handler != nil {
		handler()
	}
}

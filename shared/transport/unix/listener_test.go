package unix

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport"
	"github.com/klauspost/compress/zstd"
)

func TestListenerDialRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "anytty.sock")
	listener, err := NewListener(path)
	if err != nil {
		t.Fatalf("new listener failed: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	accepted := make(chan transport.Transport, 1)
	go func() {
		conn, err := listener.Accept(ctx)
		if err != nil {
			return
		}
		accepted <- conn
	}()

	client, err := Dial(path)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer client.Close()

	var server transport.Transport
	select {
	case server = <-accepted:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for accept")
	}
	defer server.Close()

	if err := client.Send([]byte("hello")); err != nil {
		t.Fatalf("client send failed: %v", err)
	}
	got, err := server.Recv()
	if err != nil {
		t.Fatalf("server recv failed: %v", err)
	}
	if !bytes.Equal(got, []byte("hello")) {
		t.Fatalf("unexpected server payload: %q", string(got))
	}

	if err := server.Send([]byte("world")); err != nil {
		t.Fatalf("server send failed: %v", err)
	}
	got, err = client.Recv()
	if err != nil {
		t.Fatalf("client recv failed: %v", err)
	}
	if !bytes.Equal(got, []byte("world")) {
		t.Fatalf("unexpected client payload: %q", string(got))
	}

	if err := client.Close(); err != nil {
		t.Fatalf("client close failed: %v", err)
	}
	select {
	case <-client.Done():
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for client done channel")
	}
}

func TestNewListenerPreservesActiveSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "anytty.sock")
	listener, err := NewListener(path)
	if err != nil {
		t.Fatalf("new listener failed: %v", err)
	}
	defer listener.Close()

	if second, err := NewListener(path); err == nil {
		_ = second.Close()
		t.Fatal("second listener replaced an active socket")
	}
	client, err := Dial(path)
	if err != nil {
		t.Fatalf("active listener became unreachable: %v", err)
	}
	_ = client.Close()
}

func TestNewListenerReplacesStaleSocket(t *testing.T) {
	root, err := os.MkdirTemp(shortSocketBaseDir(), "anytty-stale-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	path := filepath.Join(root, "anytty.sock")
	raw, err := net.Listen("unix", path)
	if err != nil {
		t.Fatalf("create stale listener failed: %v", err)
	}
	if unixListener, ok := raw.(*net.UnixListener); ok {
		unixListener.SetUnlinkOnClose(false)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("close stale listener failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stale socket was not preserved: %v", err)
	}

	listener, err := NewListener(path)
	if err != nil {
		t.Fatalf("replace stale socket failed: %v", err)
	}
	defer listener.Close()

	client, err := Dial(path)
	if err != nil {
		t.Fatalf("replacement listener is unreachable: %v", err)
	}
	_ = client.Close()
}

func TestListenerAcceptContextCancel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "anytty.sock")
	listener, err := NewListener(path)
	if err != nil {
		t.Fatalf("new listener failed: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := listener.Accept(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestListenerCanceledAcceptDoesNotStealNextDial(t *testing.T) {
	path := filepath.Join(t.TempDir(), "anytty.sock")
	listener, err := NewListener(path)
	if err != nil {
		t.Fatalf("new listener failed: %v", err)
	}
	defer listener.Close()

	cancelCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := listener.Accept(cancelCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline from canceled accept, got %v", err)
	}

	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	accepted := make(chan transport.Transport, 1)
	go func() {
		conn, err := listener.Accept(ctx)
		if err == nil {
			accepted <- conn
		}
	}()

	client, err := Dial(path)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer client.Close()

	select {
	case server := <-accepted:
		defer server.Close()
	case <-time.After(time.Second):
		t.Fatal("canceled accept stole the next dial")
	}
}

func TestTransportRejectsOversizedPacketHeader(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	server, err := newTransport(serverConn)
	if err != nil {
		t.Fatalf("new transport failed: %v", err)
	}
	defer server.Close()

	writer, err := zstd.NewWriter(
		clientConn,
		zstd.WithEncoderConcurrency(1),
		zstd.WithEncoderLevel(zstd.SpeedFastest),
		zstd.WithWindowSize(zstdTransportEncoderWindowSize),
		zstd.WithLowerEncoderMem(true),
	)
	if err != nil {
		t.Fatalf("new zstd writer failed: %v", err)
	}
	defer writer.Close()
	defer clientConn.Close()

	writeDone := make(chan error, 1)
	go func() {
		var header [5]byte
		header[0] = packetKindFrame
		binary.BigEndian.PutUint32(header[1:], uint32(maxPacketPayloadSize+1))
		if err := writeAll(writer, header[:]); err != nil {
			writeDone <- err
			return
		}
		writeDone <- writer.Flush()
	}()

	if _, err := server.Recv(); err == nil || !strings.Contains(err.Error(), "packet too large") {
		t.Fatalf("expected oversized packet error, got %v", err)
	}
	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("writer failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out writing oversized packet header")
	}
}

func TestTransportSendRejectsOversizedLogicalFrameBeforeClone(t *testing.T) {
	transport := &Transport{}
	frame := make([]byte, maxLogicalFrameSize+1)
	var err error
	allocations := testing.AllocsPerRun(100, func() {
		err = transport.Send(frame)
	})
	if !errors.Is(err, wire.ErrFrameTooLarge) {
		t.Fatalf("oversized logical frame error = %v", err)
	}
	if allocations != 0 {
		t.Fatalf("oversized logical frame rejection allocated %.2f times", allocations)
	}
}

func TestTransportRecvRejectsEndlessFragmentsBeforeOversizedAppend(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	server, err := newTransport(serverConn)
	if err != nil {
		t.Fatalf("new transport failed: %v", err)
	}
	defer server.Close()

	writer, err := zstd.NewWriter(
		clientConn,
		zstd.WithEncoderConcurrency(1),
		zstd.WithEncoderLevel(zstd.SpeedFastest),
		zstd.WithWindowSize(zstdTransportEncoderWindowSize),
		zstd.WithLowerEncoderMem(true),
	)
	if err != nil {
		t.Fatalf("new zstd writer failed: %v", err)
	}
	defer writer.Close()
	defer clientConn.Close()

	writeDone := make(chan error, 1)
	go func() {
		fullPayload := make([]byte, maxPacketPayloadSize)
		fullPackets := maxLogicalFrameSize / maxPacketPayloadSize
		for index := 0; index < fullPackets; index++ {
			kind := byte(packetKindFragmentContinue)
			if index == 0 {
				kind = packetKindFragmentStart
			}
			if err := writeTestPacket(writer, kind, fullPayload); err != nil {
				writeDone <- err
				return
			}
		}
		overflow := make([]byte, maxLogicalFrameSize%maxPacketPayloadSize+1)
		if err := writeTestPacket(writer, packetKindFragmentContinue, overflow); err != nil {
			writeDone <- err
			return
		}
		writeDone <- writer.Flush()
	}()

	if _, err := server.Recv(); !errors.Is(err, wire.ErrFrameTooLarge) {
		t.Fatalf("endless fragmented frame error = %v", err)
	}
	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("fragment writer failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("fragment writer did not finish")
	}
}

func writeTestPacket(writer io.Writer, kind byte, payload []byte) error {
	var header [5]byte
	header[0] = kind
	binary.BigEndian.PutUint32(header[1:], uint32(len(payload)))
	if err := writeAll(writer, header[:]); err != nil {
		return err
	}
	return writeAll(writer, payload)
}

func TestTransportCloseUnblocksBlockedWrite(t *testing.T) {
	conn := newBlockingWriteConn()
	tr, err := newTransport(conn)
	if err != nil {
		t.Fatalf("new transport failed: %v", err)
	}

	sendDone := make(chan error, 1)
	go func() {
		sendDone <- tr.Send(bytes.Repeat([]byte("x"), maxPacketPayloadSize*2))
	}()
	time.Sleep(20 * time.Millisecond)

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- tr.Close()
	}()
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("close failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("transport close did not unblock blocked write")
	}
	select {
	case err := <-sendDone:
		if err == nil {
			t.Fatal("expected send to fail after forced close")
		}
	case <-time.After(time.Second):
		t.Fatal("blocked send did not return after close")
	}
}

func TestListenerSupportsLongSocketPath(t *testing.T) {
	base := filepath.Join(t.TempDir(), strings.Repeat("socket-dir-", 8))
	path := filepath.Join(base, "anytty.sock")
	if len(path) <= maxSocketPathBytes() {
		t.Fatalf("expected long socket path, got len=%d limit=%d", len(path), maxSocketPathBytes())
	}

	listener, err := NewListener(path)
	if err != nil {
		t.Fatalf("new listener failed: %v", err)
	}
	defer listener.Close()

	actualPath, aliasPath := resolveSocketPath(path)
	if aliasPath != "" {
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("expected visible alias at original path: %v", err)
		}
	} else {
		if _, err := os.Lstat(actualPath); err != nil {
			t.Fatalf("expected mapped socket at short path: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	accepted := make(chan transport.Transport, 1)
	go func() {
		conn, err := listener.Accept(ctx)
		if err != nil {
			return
		}
		accepted <- conn
	}()

	client, err := Dial(path)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer client.Close()

	select {
	case server := <-accepted:
		defer server.Close()
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for accept on long socket path")
	}
}

func TestListenerDialRoundTripLargeAndEmptyFrames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "anytty.sock")
	listener, err := NewListener(path)
	if err != nil {
		t.Fatalf("new listener failed: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	accepted := make(chan transport.Transport, 1)
	go func() {
		conn, err := listener.Accept(ctx)
		if err != nil {
			return
		}
		accepted <- conn
	}()

	client, err := Dial(path)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer client.Close()

	var server transport.Transport
	select {
	case server = <-accepted:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for accept")
	}
	defer server.Close()

	large := bytes.Repeat([]byte("anytty-compressible-frame-"), 40*1024)
	if err := client.Send(large); err != nil {
		t.Fatalf("client send large failed: %v", err)
	}
	got, err := server.Recv()
	if err != nil {
		t.Fatalf("server recv large failed: %v", err)
	}
	if !bytes.Equal(got, large) {
		t.Fatalf("unexpected large payload mismatch: got=%d want=%d", len(got), len(large))
	}

	if err := server.Send(nil); err != nil {
		t.Fatalf("server send empty failed: %v", err)
	}
	got, err = client.Recv()
	if err != nil {
		t.Fatalf("client recv empty failed: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty frame payload, got %d bytes", len(got))
	}
}

func TestListenerDialRoundTripManyFrames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "anytty.sock")
	listener, err := NewListener(path)
	if err != nil {
		t.Fatalf("new listener failed: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	accepted := make(chan transport.Transport, 1)
	go func() {
		conn, err := listener.Accept(ctx)
		if err != nil {
			return
		}
		accepted <- conn
	}()

	client, err := Dial(path)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer client.Close()

	var server transport.Transport
	select {
	case server = <-accepted:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for accept")
	}
	defer server.Close()

	for i := 0; i < 128; i++ {
		payload := []byte(fmt.Sprintf("frame-%03d-%s", i, strings.Repeat("x", i%17)))
		if err := client.Send(payload); err != nil {
			t.Fatalf("client send %d failed: %v", i, err)
		}
		got, err := server.Recv()
		if err != nil {
			t.Fatalf("server recv %d failed: %v", i, err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("frame %d mismatch: got %q want %q", i, string(got), string(payload))
		}
	}
}

func TestListenerDialRoundTripFragmentedFrameBoundaries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "anytty.sock")
	listener, err := NewListener(path)
	if err != nil {
		t.Fatalf("new listener failed: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	accepted := make(chan transport.Transport, 1)
	go func() {
		conn, err := listener.Accept(ctx)
		if err != nil {
			return
		}
		accepted <- conn
	}()

	client, err := Dial(path)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer client.Close()

	var server transport.Transport
	select {
	case server = <-accepted:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for accept")
	}
	defer server.Close()

	for _, size := range []int{maxPacketPayloadSize + 1, maxPacketPayloadSize*3 + 17} {
		payload := bytes.Repeat([]byte("z"), size)
		if err := client.Send(payload); err != nil {
			t.Fatalf("client send size=%d failed: %v", size, err)
		}
		got, err := server.Recv()
		if err != nil {
			t.Fatalf("server recv size=%d failed: %v", size, err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("fragmented payload mismatch for size=%d: got=%d want=%d", size, len(got), len(payload))
		}
	}
}

func TestListenerDialRoundTripConcurrentSmallFrames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "anytty.sock")
	listener, err := NewListener(path)
	if err != nil {
		t.Fatalf("new listener failed: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	accepted := make(chan transport.Transport, 1)
	go func() {
		conn, err := listener.Accept(ctx)
		if err != nil {
			return
		}
		accepted <- conn
	}()

	client, err := Dial(path)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer client.Close()

	var server transport.Transport
	select {
	case server = <-accepted:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for accept")
	}
	defer server.Close()

	const frameCount = 32
	payloads := make([][]byte, frameCount)
	for i := 0; i < frameCount; i++ {
		payloads[i] = []byte(fmt.Sprintf("small-%02d-%s", i, strings.Repeat("y", i%5)))
	}

	var wg sync.WaitGroup
	for _, payload := range payloads {
		payload := payload
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := client.Send(payload); err != nil {
				t.Errorf("client send failed: %v", err)
			}
		}()
	}
	wg.Wait()

	received := make(map[string]int, frameCount)
	for i := 0; i < frameCount; i++ {
		got, err := server.Recv()
		if err != nil {
			t.Fatalf("server recv %d failed: %v", i, err)
		}
		received[string(got)]++
	}
	for _, payload := range payloads {
		if received[string(payload)] != 1 {
			t.Fatalf("expected payload %q exactly once, got count=%d", string(payload), received[string(payload)])
		}
	}
}

func TestTransportZstdWindowKeepsConnectionHeapBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("memory smoke is not needed in short mode")
	}
	var before runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	path := filepath.Join(t.TempDir(), "anytty.sock")
	listener, err := NewListener(path)
	if err != nil {
		t.Fatalf("new listener failed: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	accepted := make(chan transport.Transport, 4)
	go func() {
		for i := 0; i < 4; i++ {
			conn, err := listener.Accept(ctx)
			if err != nil {
				return
			}
			accepted <- conn
		}
	}()

	clients := make([]*Transport, 0, 4)
	servers := make([]transport.Transport, 0, 4)
	defer func() {
		for _, client := range clients {
			_ = client.Close()
		}
		for _, server := range servers {
			_ = server.Close()
		}
	}()

	for i := 0; i < 4; i++ {
		client, err := Dial(path)
		if err != nil {
			t.Fatalf("dial %d failed: %v", i, err)
		}
		clients = append(clients, client)
		select {
		case server := <-accepted:
			servers = append(servers, server)
		case <-time.After(3 * time.Second):
			t.Fatal("timed out waiting for accept")
		}
	}

	payload := bytes.Repeat([]byte("anytty-memory-transport-"), 32*1024)
	for i, client := range clients {
		if err := client.Send(payload); err != nil {
			t.Fatalf("client send %d failed: %v", i, err)
		}
		got, err := servers[i].Recv()
		if err != nil {
			t.Fatalf("server recv %d failed: %v", i, err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("payload mismatch for connection %d", i)
		}
	}

	var after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&after)
	var heapGrowth uint64
	if after.HeapAlloc > before.HeapAlloc {
		heapGrowth = after.HeapAlloc - before.HeapAlloc
	}
	const maxTransportHeapGrowth = 16 << 20
	if heapGrowth > maxTransportHeapGrowth {
		t.Fatalf("transport heap grew too much: got=%d want<=%d", heapGrowth, maxTransportHeapGrowth)
	}
}

type blockingWriteConn struct {
	done chan struct{}
	once sync.Once
}

func newBlockingWriteConn() *blockingWriteConn {
	return &blockingWriteConn{done: make(chan struct{})}
}

func (c *blockingWriteConn) Read(_ []byte) (int, error) {
	<-c.done
	return 0, io.EOF
}

func (c *blockingWriteConn) Write(_ []byte) (int, error) {
	<-c.done
	return 0, net.ErrClosed
}

func (c *blockingWriteConn) Close() error {
	c.once.Do(func() {
		close(c.done)
	})
	return nil
}

func (c *blockingWriteConn) LocalAddr() net.Addr {
	return testAddr("local")
}

func (c *blockingWriteConn) RemoteAddr() net.Addr {
	return testAddr("remote")
}

func (c *blockingWriteConn) SetDeadline(time.Time) error {
	return nil
}

func (c *blockingWriteConn) SetReadDeadline(time.Time) error {
	return nil
}

func (c *blockingWriteConn) SetWriteDeadline(time.Time) error {
	return nil
}

type testAddr string

func (a testAddr) Network() string {
	return string(a)
}

func (a testAddr) String() string {
	return string(a)
}

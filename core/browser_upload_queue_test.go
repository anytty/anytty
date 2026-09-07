package core

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport/memory"
)

func TestBrowserUploadQueueBoundsInflightAndWraps(t *testing.T) {
	q := newBrowserUploadQueue(8)
	defer q.close()
	if err := q.enqueue([]byte("abcdefgh")); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 5)
	if n, err := q.peek(buffer); err != nil || n != 5 || string(buffer) != "abcde" {
		t.Fatalf("peek %q %d %v", buffer, n, err)
	}
	if err := q.enqueue([]byte("x")); err == nil {
		t.Fatal("in-flight bytes freed credit before write completion")
	}
	q.consume(5)
	if err := q.enqueue([]byte("ijklm")); err != nil {
		t.Fatal(err)
	}
	buffer = make([]byte, 8)
	if n, err := q.peek(buffer); err != nil || n != 8 || string(buffer) != "fghijklm" {
		t.Fatalf("wrap %q %d %v", buffer, n, err)
	}
	q.consume(8)
	finished := make(chan error, 1)
	go func() { _, err := q.peek(buffer); finished <- err }()
	q.close()
	select {
	case err := <-finished:
		if err != io.EOF {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("queue close did not release worker")
	}
}

func TestBrowserUploadBlockedTargetDoesNotBlockFrameHandler(t *testing.T) {
	client, server := memory.NewPair()
	defer client.Close()
	defer server.Close()
	session := newProtocolSession(NewServer(), server, fullDaemonTransportScope())
	defer session.releaseAllBrowserProxies()
	local, target := net.Pipe()
	defer target.Close()
	const channel = uint16(41)
	proxy := &sessionBrowserProxy{channel: channel, conn: local, uploadQueue: newBrowserUploadQueue(64 << 10)}
	session.browserChannels[channel] = proxy
	finished := make(chan error, 1)
	go func() {
		finished <- session.handleStreamFrame(context.Background(), channel, wire.TypeBrowserData, bytes.Repeat([]byte{1}, 64<<10))
	}()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("target write blocked frame handler")
	}
	otherSent := make(chan error, 1)
	go func() { otherSent <- session.sendFrame(channel+1, wire.TypeBrowserData, []byte("other")) }()
	gotChannel, typ, payload := receiveProtocolFrame(t, client)
	if gotChannel != channel+1 || typ != wire.TypeBrowserData || string(payload) != "other" {
		t.Fatal("other channel was blocked")
	}
	if err := <-otherSent; err != nil {
		t.Fatal(err)
	}
	// The target has consumed nothing, so this byte must exceed the fixed window.
	if err := session.handleStreamFrame(context.Background(), channel, wire.TypeBrowserData, []byte{2}); err == nil {
		t.Fatal("unbounded upload accepted")
	}
	if session.browserProxyForChannel(channel) != nil {
		t.Fatal("overflow retained resource")
	}
	_ = target.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := target.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("target not closed: %v", err)
	}
}

func TestBrowserNegotiatedUploadToRealTCPTarget(t *testing.T) {
	const totalSize = 4 << 20
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	targetBytes := make(chan []byte, 1)
	targetError := make(chan error, 1)
	releaseTarget := make(chan struct{})
	defer close(releaseTarget)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			targetError <- err
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		buffer := make([]byte, totalSize)
		_, err = io.ReadFull(conn, buffer)
		if err != nil {
			targetError <- err
			return
		}
		targetBytes <- buffer
		<-releaseTarget
	}()
	client, server := memory.NewPair()
	defer client.Close()
	defer server.Close()
	session := newProtocolSession(NewServer(), server, fullDaemonTransportScope())
	defer session.releaseAllBrowserProxies()
	resource, err := session.ApplicationBrowserProxyOpen(context.Background(), "127.0.0.1", uint16(listener.Addr().(*net.TCPAddr).Port), 0, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	if resource.SendWindowBytes != 64<<10 {
		t.Fatal("upload window was not negotiated")
	}
	channel := uint16(resource.Token[0])<<8 | uint16(resource.Token[1])
	payload := make([]byte, totalSize)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	for offset := 0; offset < totalSize; offset += 32 << 10 {
		if err := session.handleStreamFrame(context.Background(), channel, wire.TypeBrowserData, payload[offset:offset+32<<10]); err != nil {
			t.Fatal(err)
		}
		gotChannel, typ, frame := receiveProtocolFrame(t, client)
		if gotChannel != channel || typ != wire.TypeFileAck {
			t.Fatalf("expected upload acknowledgement, got channel=%d type=%d", gotChannel, typ)
		}
		ack, err := protocol.DecodeFileTransferAck(frame)
		if err != nil || ack.Offset != int64(offset+32<<10) || ack.WindowBytes != 32<<10 {
			t.Fatalf("ack=%+v err=%v", ack, err)
		}
	}
	select {
	case got := <-targetBytes:
		if !bytes.Equal(got, payload) {
			t.Fatal("uploaded bytes corrupted")
		}
	case err := <-targetError:
		t.Fatal(err)
	case <-time.After(10 * time.Second):
		t.Fatal("target did not receive upload")
	}
}

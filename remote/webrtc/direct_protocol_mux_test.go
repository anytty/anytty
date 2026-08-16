package webrtc

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/anytty/anytty/internal/protocol/directsignal"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/pion/stun/v3"
)

func TestSplitDirectListenerRoutesSignalingWithoutConsumingFrame(t *testing.T) {
	signaling, ice, address := splitDirectTestListeners(t)
	defer signaling.Close()
	defer ice.Close()

	client, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	written := &remoteauthpb.DirectSignalingRequestV2{SchemaVersion: 2, RequestId: "mux-signal"}
	if err := directsignal.WriteMessage(client, written); err != nil {
		t.Fatal(err)
	}

	accepted := acceptDirectTestConnection(t, signaling)
	defer accepted.Close()
	read := &remoteauthpb.DirectSignalingRequestV2{}
	if err := directsignal.ReadMessage(accepted, read); err != nil {
		t.Fatal(err)
	}
	if read.GetRequestId() != written.GetRequestId() {
		t.Fatalf("request id = %q", read.GetRequestId())
	}
}

func TestSplitDirectListenerRoutesRFC4571STUNWithoutConsumingFrame(t *testing.T) {
	signaling, ice, address := splitDirectTestListeners(t)
	defer signaling.Close()
	defer ice.Close()

	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	frame := make([]byte, 2+len(message.Raw))
	binary.BigEndian.PutUint16(frame[:2], uint16(len(message.Raw)))
	copy(frame[2:], message.Raw)
	client, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Write(frame); err != nil {
		t.Fatal(err)
	}

	accepted := acceptDirectTestConnection(t, ice)
	defer accepted.Close()
	got := make([]byte, len(frame))
	if _, err := io.ReadFull(accepted, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != string(frame) {
		t.Fatal("ICE frame changed while routing")
	}
}

func TestSplitDirectListenerCloseUnblocksAnUnclassifiedConnection(t *testing.T) {
	signaling, ice, address := splitDirectTestListeners(t)
	client, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Write([]byte{0, 0}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if err := signaling.Close(); err != nil {
		t.Fatal(err)
	}
	_ = ice.Close()
	if err := client.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 1)
	if _, err := client.Read(buffer); err == nil {
		t.Fatal("unclassified connection remained open after mux close")
	}
}

func splitDirectTestListeners(t *testing.T) (net.Listener, net.Listener, string) {
	t.Helper()
	root, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	signaling, ice, err := SplitDirectListener(root)
	if err != nil {
		t.Fatal(err)
	}
	return signaling, ice, root.Addr().String()
}

func acceptDirectTestConnection(t *testing.T, listener net.Listener) net.Conn {
	t.Helper()
	type result struct {
		connection net.Conn
		err        error
	}
	done := make(chan result, 1)
	go func() { connection, err := listener.Accept(); done <- result{connection, err} }()
	select {
	case value := <-done:
		if value.err != nil {
			t.Fatal(value.err)
		}
		return value.connection
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for routed connection")
		return nil
	}
}

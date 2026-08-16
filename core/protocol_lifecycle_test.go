package core

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport"
	"github.com/anytty/anytty/shared/transport/memory"
)

type recordingTransportObserver struct {
	count atomic.Uint32
	ready chan struct{}
}

func (observer *recordingTransportObserver) HelloAccepted() {
	if observer.count.Add(1) == 1 {
		close(observer.ready)
	}
}

func TestObservedTransportRequiresHelloAndNotifiesOnceAfterResponse(t *testing.T) {
	server := NewServer()
	client, daemon := memory.NewPair()
	observer := &recordingTransportObserver{ready: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		done <- server.ServeScopedTransportObserved(context.Background(), daemon, fullDaemonTransportScope(), observer)
	}()

	requestPayload, err := protocol.EncodeRequestPayload(protocol.Request{ID: 1, Method: "api.execute"})
	if err != nil {
		t.Fatal(err)
	}
	sendProtocolFrame(t, client, wire.TypeRequest, requestPayload)
	_, typ, payload := receiveProtocolFrame(t, client)
	if typ != wire.TypeError {
		t.Fatalf("pre-Hello response type = %d", typ)
	}
	protocolError, err := protocol.DecodeErrorPayload(payload)
	if err != nil || !strings.Contains(protocolError.Error.Message, "Hello is required") {
		t.Fatalf("pre-Hello response = (%#v, %v)", protocolError, err)
	}
	select {
	case <-observer.ready:
		t.Fatal("pre-Hello request notified observer")
	default:
	}

	helloPayload, err := protocol.EncodeHelloPayload(protocol.Hello{Version: wire.Version, Client: "lifecycle-test"})
	if err != nil {
		t.Fatal(err)
	}
	sendProtocolFrame(t, client, wire.TypeHello, helloPayload)
	_, typ, _ = receiveProtocolFrame(t, client)
	if typ != wire.TypeHello {
		t.Fatalf("Hello response type = %d", typ)
	}
	select {
	case <-observer.ready:
	case <-time.After(time.Second):
		t.Fatal("accepted Hello did not notify observer")
	}

	sendProtocolFrame(t, client, wire.TypeHello, helloPayload)
	_, typ, payload = receiveProtocolFrame(t, client)
	if typ != wire.TypeError {
		t.Fatalf("duplicate Hello response type = %d", typ)
	}
	duplicate, err := protocol.DecodeErrorPayload(payload)
	if err != nil || !strings.Contains(duplicate.Error.Message, "already accepted") || observer.count.Load() != 1 {
		t.Fatalf("duplicate Hello response=%#v err=%v observer=%d", duplicate, err, observer.count.Load())
	}

	_ = client.Close()
	select {
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "closed") && !strings.Contains(err.Error(), "EOF") {
			t.Fatalf("observed transport exit = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("observed transport did not stop")
	}
}

func sendProtocolFrame(t *testing.T, connection transport.Transport, typ uint8, payload []byte) {
	t.Helper()
	frame, err := wire.EncodeFrame(0, typ, payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.Send(frame); err != nil {
		t.Fatal(err)
	}
}

func receiveProtocolFrame(t *testing.T, connection transport.Transport) (uint16, uint8, []byte) {
	t.Helper()
	frame, err := connection.Recv()
	if err != nil {
		t.Fatal(err)
	}
	channel, typ, payload, err := wire.DecodeFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	return channel, typ, payload
}

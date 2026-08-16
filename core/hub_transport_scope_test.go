package core

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	clientendpoint "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport/datachannel"
)

func TestHubDataChannelTransportCannotEscapeCapabilityTerminalScope(t *testing.T) {
	expiresAt := time.Now().UTC().Add(time.Hour)
	server := NewServer(WithApplicationExecutorFactory(applicationTestExecutorFactory), WithProcessFactory(newRecordingProcessFactory()), WithClientAccessService(newGrantAccessTestService(map[string]time.Time{"grant-hub": expiresAt})))
	registerScopedTestTerminal(t, server, "allowed")
	registerScopedTestTerminal(t, server, "denied")

	clientChannel, serverChannel := newHubTestChannelPair()
	clientTransport := datachannel.New(clientChannel)
	serverTransport := datachannel.New(serverChannel)
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ServeScopedTransport(context.Background(), serverTransport, TransportScope{GrantID: "grant-hub", GrantExpiresAt: expiresAt, PrincipalID: "subject", TerminalID: "allowed"})
	}()
	client := protocol.NewClient(clientTransport)
	if err := client.Hello(context.Background(), protocol.Hello{Version: wire.Version, Client: "hub-scope-test"}); err != nil {
		t.Fatalf("hello over hub data channel: %v", err)
	}
	defer func() {
		_ = client.Close()
		select {
		case err := <-errCh:
			if err != nil && !strings.Contains(err.Error(), "EOF") {
				t.Fatalf("serve hub transport: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("hub transport server did not stop")
		}
	}()

	application, err := clientruntime.NewApplicationSession(clientruntime.EndpointSessionStamp{EndpointID: clientendpoint.EndpointID("hub"), RouteID: clientendpoint.RouteID("webrtc"), Generation: 1}, client)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.TerminalGet(context.Background(), &apipb.TerminalGetCommand{Terminal: &apipb.TerminalRef{EndpointId: "hub", TerminalId: "allowed"}}); err != nil {
		t.Fatalf("get allowed terminal: %v", err)
	}
	if _, err := application.TerminalGet(context.Background(), &apipb.TerminalGetCommand{Terminal: &apipb.TerminalRef{EndpointId: "hub", TerminalId: "denied"}}); err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("expected capability scope denial, got %v", err)
	}
}

type hubTestChannel struct {
	mu             sync.Mutex
	peer           *hubTestChannel
	messageHandler func([]byte)
	closeHandler   func()
	closed         bool
}

func newHubTestChannelPair() (*hubTestChannel, *hubTestChannel) {
	left := &hubTestChannel{}
	right := &hubTestChannel{}
	left.peer = right
	right.peer = left
	return left, right
}

func (channel *hubTestChannel) SetMessageHandler(handler func([]byte)) {
	channel.mu.Lock()
	channel.messageHandler = handler
	channel.mu.Unlock()
}

func (channel *hubTestChannel) SetCloseHandler(handler func()) {
	channel.mu.Lock()
	channel.closeHandler = handler
	channel.mu.Unlock()
}

func (channel *hubTestChannel) BufferedAmount() uint64 { return 0 }

func (channel *hubTestChannel) SetBufferedAmountLowThreshold(uint64) {}

func (channel *hubTestChannel) SetBufferedAmountLowHandler(func()) {}

func (channel *hubTestChannel) Send(payload []byte) error {
	channel.mu.Lock()
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

func (channel *hubTestChannel) Close() error {
	channel.mu.Lock()
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

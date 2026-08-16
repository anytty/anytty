package loopback

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

const testToken = "abcdefghijklmnopqrstuvwxyz0123456789_-ABCDE"

type fakeEngine struct {
	mu             sync.Mutex
	nextEventCalls int
	events         chan []byte
	requestEntered chan struct{}
	releaseRequest chan struct{}
}

func newFakeEngine() *fakeEngine {
	return &fakeEngine{events: make(chan []byte), requestEntered: make(chan struct{}), releaseRequest: make(chan struct{})}
}

func (engine *fakeEngine) OpenSession(payload []byte) (uint64, error) {
	if string(payload) != "open" {
		return 0, fmt.Errorf("unexpected open payload")
	}
	return 41, nil
}
func (engine *fakeEngine) Execute(handle uint64, payload []byte) (uint64, error) {
	if handle != 41 || string(payload) != "execute" {
		return 0, fmt.Errorf("unexpected execute request")
	}
	return 42, nil
}
func (engine *fakeEngine) OpenResourceStream(uint64, []byte) (uint64, error) { return 43, nil }
func (engine *fakeEngine) SendResourceStreamFrame(uint64, []byte) error      { return nil }
func (engine *fakeEngine) CloseResourceStream(uint64) error                  { return nil }
func (engine *fakeEngine) EngineCommand([]byte) (uint64, error)              { return 44, nil }
func (engine *fakeEngine) Cancel(uint64) error                               { return nil }
func (engine *fakeEngine) CloseSession(uint64) error {
	select {
	case <-engine.requestEntered:
	default:
		close(engine.requestEntered)
	}
	<-engine.releaseRequest
	return nil
}
func (engine *fakeEngine) Release(uint64) error { return nil }
func (engine *fakeEngine) NextEvent(ctx context.Context) ([]byte, error) {
	engine.mu.Lock()
	engine.nextEventCalls++
	engine.mu.Unlock()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case event := <-engine.events:
		return event, nil
	}
}

func TestStartRejectsInvalidToken(t *testing.T) {
	if _, err := Start(newFakeEngine(), "short"); err == nil {
		t.Fatal("Start accepted an invalid token")
	}
}

func TestServerAuthenticatesAndDispatchesOpaqueRequests(t *testing.T) {
	engine := newFakeEngine()
	server := startTestServer(t, engine)
	client := authenticatedClient(t, server)

	if err := websocket.Message.Send(client, requestFrame(opOpenSession, 7, 0, []byte("open"), false)); err != nil {
		t.Fatal(err)
	}
	operation, requestID, handle, payload := receiveResponse(t, client)
	if operation != opAccepted || requestID != 7 || handle != 41 || len(payload) != 0 {
		t.Fatalf("open response = op=%x request=%d handle=%d payload=%q", operation, requestID, handle, payload)
	}

	if err := websocket.Message.Send(client, requestFrame(opExecute, 8, 41, []byte("execute"), true)); err != nil {
		t.Fatal(err)
	}
	operation, requestID, handle, _ = receiveResponse(t, client)
	if operation != opAccepted || requestID != 8 || handle != 42 {
		t.Fatalf("execute response = op=%x request=%d handle=%d", operation, requestID, handle)
	}

	if err := websocket.Message.Send(client, []byte{opExecute}); err != nil {
		t.Fatal(err)
	}
	operation, requestID, _, payload = receiveResponse(t, client)
	if operation != opError || requestID != 0 || len(payload) == 0 {
		t.Fatalf("malformed response = op=%x request=%d payload=%q", operation, requestID, payload)
	}
}

func TestEventsAreNotConsumedWithoutAuthenticatedClient(t *testing.T) {
	engine := newFakeEngine()
	server := startTestServer(t, engine)
	time.Sleep(50 * time.Millisecond)
	engine.mu.Lock()
	calls := engine.nextEventCalls
	engine.mu.Unlock()
	if calls != 0 {
		t.Fatalf("NextEvent called %d times without a client", calls)
	}

	client := authenticatedClient(t, server)
	engine.events <- []byte("event")
	operation, requestID, handle, payload := receiveResponse(t, client)
	if operation != opEvent || requestID != 0 || handle != 0 || string(payload) != "event" {
		t.Fatalf("event response = op=%x request=%d handle=%d payload=%q", operation, requestID, handle, payload)
	}
}

func TestNewAuthenticationReplacesPreviousClient(t *testing.T) {
	server := startTestServer(t, newFakeEngine())
	first := authenticatedClient(t, server)
	second := authenticatedClient(t, server)

	_ = first.SetReadDeadline(time.Now().Add(time.Second))
	var payload []byte
	if err := websocket.Message.Receive(first, &payload); err == nil {
		t.Fatal("replaced client remained readable")
	}
	_ = second.SetReadDeadline(time.Now().Add(time.Second))
}

func TestServerLimitsUpgradedClients(t *testing.T) {
	server := startTestServer(t, newFakeEngine())
	clients := make([]*websocket.Conn, 0, maxUpgradedClients)
	for range maxUpgradedClients {
		client := dialClient(t, server)
		clients = append(clients, client)
	}
	for _, client := range clients {
		defer client.Close()
	}
	overflow := dialClient(t, server)
	defer overflow.Close()
	if err := websocket.Message.Send(overflow, append([]byte{opAuth}, []byte(testToken)...)); err == nil {
		_ = overflow.SetReadDeadline(time.Now().Add(time.Second))
		var response []byte
		if err := websocket.Message.Receive(overflow, &response); err == nil {
			t.Fatal("client beyond the upgrade limit authenticated")
		}
	}
}

func TestStopWaitsForInflightRequest(t *testing.T) {
	engine := newFakeEngine()
	server := startTestServer(t, engine)
	client := authenticatedClient(t, server)
	if err := websocket.Message.Send(client, requestFrame(opCloseSession, 9, 41, nil, true)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-engine.requestEntered:
	case <-time.After(time.Second):
		t.Fatal("request did not enter engine")
	}

	stopped := make(chan struct{})
	go func() {
		server.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
		t.Fatal("Stop returned while an engine request was in flight")
	case <-time.After(50 * time.Millisecond):
	}
	close(engine.releaseRequest)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop did not return after request completed")
	}
}

func TestHandshakeRequiresAllowedOriginAndProtocol(t *testing.T) {
	server := startTestServer(t, newFakeEngine())
	for _, test := range []struct {
		name     string
		origin   string
		protocol []string
	}{
		{name: "origin", origin: "https://example.com", protocol: []string{Protocol}},
		{name: "protocol", origin: "capacitor://localhost", protocol: []string{"other"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			config, err := websocket.NewConfig(fmt.Sprintf("ws://127.0.0.1:%d/", server.Port()), test.origin)
			if err != nil {
				t.Fatal(err)
			}
			config.Protocol = test.protocol
			if client, err := websocket.DialConfig(config); err == nil {
				_ = client.Close()
				t.Fatal("invalid handshake was accepted")
			}
		})
	}
}

func startTestServer(t *testing.T, engine Engine) *Server {
	t.Helper()
	server, err := Start(engine, testToken)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(server.Stop)
	return server
}

func authenticatedClient(t *testing.T, server *Server) *websocket.Conn {
	t.Helper()
	client := dialClient(t, server)
	t.Cleanup(func() { _ = client.Close() })
	auth := append([]byte{opAuth}, []byte(testToken)...)
	if err := websocket.Message.Send(client, auth); err != nil {
		t.Fatal(err)
	}
	operation, requestID, handle, payload := receiveResponse(t, client)
	if operation != opACK || requestID != 0 || handle != 0 || len(payload) != 0 {
		t.Fatalf("auth response = op=%x request=%d handle=%d payload=%q", operation, requestID, handle, payload)
	}
	return client
}

func dialClient(t *testing.T, server *Server) *websocket.Conn {
	t.Helper()
	config, err := websocket.NewConfig(fmt.Sprintf("ws://127.0.0.1:%d/", server.Port()), "capacitor://localhost")
	if err != nil {
		t.Fatal(err)
	}
	config.Protocol = []string{Protocol}
	client, err := websocket.DialConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func requestFrame(operation byte, requestID, handle uint64, payload []byte, withHandle bool) []byte {
	header := 9
	if withHandle {
		header = 17
	}
	frame := make([]byte, header+len(payload))
	frame[0] = operation
	binary.BigEndian.PutUint64(frame[1:9], requestID)
	if withHandle {
		binary.BigEndian.PutUint64(frame[9:17], handle)
	}
	copy(frame[header:], payload)
	return frame
}

func receiveResponse(t *testing.T, client *websocket.Conn) (byte, uint64, uint64, []byte) {
	t.Helper()
	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	var frame []byte
	if err := websocket.Message.Receive(client, &frame); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("timed out waiting for bridge response")
		}
		t.Fatal(err)
	}
	_ = client.SetReadDeadline(time.Time{})
	if len(frame) < responseHeaderBytes {
		t.Fatalf("response length = %d", len(frame))
	}
	length := int(binary.BigEndian.Uint32(frame[17:21]))
	if responseHeaderBytes+length != len(frame) {
		t.Fatalf("response payload length = %d, frame = %d", length, len(frame))
	}
	return frame[0], binary.BigEndian.Uint64(frame[1:9]), binary.BigEndian.Uint64(frame[9:17]), frame[21:]
}

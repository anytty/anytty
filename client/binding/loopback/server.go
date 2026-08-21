// Package loopback owns the authenticated WebView-to-Go binding transport used
// by every native mobile platform. Platform code only starts this server and
// implements requests that require an operating-system API.
package loopback

import (
	"context"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/client/binding"
	"golang.org/x/net/netutil"
	"golang.org/x/net/websocket"
)

const (
	Protocol                 = "anytty.binding.v1"
	MaxMessageBytes          = binding.MaxPayloadBytes
	responseHeaderBytes      = 21
	authTokenBytes           = 43
	authFrameBytes           = authTokenBytes + 1
	maxPhysicalClients       = 8
	maxUpgradedClients       = 4
	authDeadline             = 2 * time.Second
	writeDeadline            = 5 * time.Second
	opAuth              byte = 0x01
	opOpenSession       byte = 0x10
	opExecute           byte = 0x11
	opEngineCommand     byte = 0x12
	opCancel            byte = 0x14
	opCloseSession      byte = 0x15
	opRelease           byte = 0x16
	opOpenStream        byte = 0x17
	opSendStream        byte = 0x18
	opCloseStream       byte = 0x19
	opAccepted          byte = 0x20
	opACK               byte = 0x21
	opError             byte = 0x22
	opEvent             byte = 0x30
)

var allowedOrigins = map[string]struct{}{
	"capacitor://localhost": {},
	"http://localhost":      {},
	"https://localhost":     {},
}

// Engine is the small binding surface consumed by the loopback transport.
// Implementations must keep handles opaque and must not interpret protobufs.
type Engine interface {
	AttachRenderer() (uint64, error)
	DetachRenderer(uint64) error
	OpenSession([]byte) (uint64, error)
	Execute(uint64, []byte) (uint64, error)
	OpenResourceStream(uint64, []byte) (uint64, error)
	SendResourceStreamFrame(uint64, []byte) error
	CloseResourceStream(uint64) error
	EngineCommand([]byte) (uint64, error)
	Cancel(uint64) error
	CloseSession(uint64) error
	Release(uint64) error
	NextEvent(context.Context, uint64) ([]byte, error)
}

// RegistryEngine binds one opaque engine handle to the shared registry.
type RegistryEngine struct {
	Registry *binding.Registry
	Handle   uint64
}

func (engine RegistryEngine) AttachRenderer() (uint64, error) {
	return engine.Registry.AttachRenderer(engine.Handle)
}
func (engine RegistryEngine) DetachRenderer(rendererHandle uint64) error {
	return engine.Registry.DetachRenderer(engine.Handle, rendererHandle)
}

func (engine RegistryEngine) OpenSession(payload []byte) (uint64, error) {
	return engine.Registry.OpenSession(engine.Handle, payload)
}
func (engine RegistryEngine) Execute(handle uint64, payload []byte) (uint64, error) {
	return engine.Registry.Execute(engine.Handle, handle, payload)
}
func (engine RegistryEngine) OpenResourceStream(handle uint64, payload []byte) (uint64, error) {
	return engine.Registry.OpenResourceStream(engine.Handle, handle, payload)
}
func (engine RegistryEngine) SendResourceStreamFrame(handle uint64, payload []byte) error {
	return engine.Registry.SendResourceStreamFrame(engine.Handle, handle, payload)
}
func (engine RegistryEngine) CloseResourceStream(handle uint64) error {
	return engine.Registry.CloseResourceStream(engine.Handle, handle)
}
func (engine RegistryEngine) EngineCommand(payload []byte) (uint64, error) {
	return engine.Registry.EngineCommand(engine.Handle, payload)
}
func (engine RegistryEngine) Cancel(handle uint64) error {
	return engine.Registry.Cancel(engine.Handle, handle)
}
func (engine RegistryEngine) CloseSession(handle uint64) error {
	return engine.Registry.CloseSession(engine.Handle, handle)
}
func (engine RegistryEngine) Release(handle uint64) error {
	return engine.Registry.Release(engine.Handle, handle)
}
func (engine RegistryEngine) NextEvent(ctx context.Context, rendererHandle uint64) ([]byte, error) {
	return engine.Registry.NextRendererEvent(ctx, engine.Handle, rendererHandle)
}

// Server is one engine's loopback transport. Stop is idempotent and waits for
// all in-flight binding calls before returning, so the engine can then close.
type Server struct {
	engine Engine
	token  []byte
	port   uint16

	ctx    context.Context
	cancel context.CancelFunc
	http   *http.Server
	listen net.Listener

	mu          sync.Mutex
	stopping    bool
	active      *client
	clients     map[*client]struct{}
	stateChange chan struct{}
	dispatchMu  sync.Mutex
	stopOnce    sync.Once
	wg          sync.WaitGroup
}

type client struct {
	ws           *websocket.Conn
	renderer     uint64
	writeMu      sync.Mutex
	rendererOnce sync.Once
}

// Start listens only on an ephemeral IPv4 loopback port.
func Start(engine Engine, token string) (*Server, error) {
	if engine == nil {
		return nil, fmt.Errorf("loopback binding engine is required")
	}
	if !validToken(token) {
		return nil, fmt.Errorf("loopback token must be 43-byte base64url")
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen on loopback: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if port <= 0 || port > 65535 {
		_ = listener.Close()
		return nil, fmt.Errorf("invalid loopback port %d", port)
	}
	ctx, cancel := context.WithCancel(context.Background())
	server := &Server{
		engine: engine, token: []byte(token), port: uint16(port), ctx: ctx, cancel: cancel,
		listen: netutil.LimitListener(listener, maxPhysicalClients), clients: make(map[*client]struct{}),
		stateChange: make(chan struct{}),
	}
	websocketServer := websocket.Server{
		Config:    websocket.Config{Protocol: []string{Protocol}},
		Handshake: server.handshake,
		Handler:   server.handleClient,
	}
	server.http = &http.Server{
		Handler:           websocketServer,
		ReadHeaderTimeout: authDeadline,
		MaxHeaderBytes:    16 << 10,
	}
	server.wg.Add(2)
	go func() {
		defer server.wg.Done()
		_ = server.http.Serve(server.listen)
	}()
	go func() {
		defer server.wg.Done()
		server.pumpEvents()
	}()
	return server, nil
}

func (server *Server) Port() uint16 { return server.port }

// Stop closes the listener and every hijacked WebSocket, then drains request
// handlers and the event pump before returning.
func (server *Server) Stop() {
	if server == nil {
		return
	}
	server.stopOnce.Do(func() {
		server.mu.Lock()
		server.stopping = true
		server.signalLocked()
		clients := make([]*client, 0, len(server.clients))
		for current := range server.clients {
			clients = append(clients, current)
		}
		server.mu.Unlock()

		server.cancel()
		_ = server.http.Close()
		_ = server.listen.Close()
		for _, current := range clients {
			_ = current.ws.Close()
		}
		server.wg.Wait()
	})
}

func (server *Server) handshake(config *websocket.Config, request *http.Request) error {
	if request.URL.Path != "/" || request.URL.RawQuery != "" {
		return fmt.Errorf("invalid loopback path")
	}
	if _, ok := allowedOrigins[request.Header.Get("Origin")]; !ok {
		return fmt.Errorf("invalid loopback origin")
	}
	protocols := request.Header.Values("Sec-WebSocket-Protocol")
	matched := false
	for _, value := range protocols {
		for _, protocol := range strings.Split(value, ",") {
			if strings.TrimSpace(protocol) == Protocol {
				matched = true
			}
		}
	}
	if !matched {
		return fmt.Errorf("binding subprotocol is required")
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	if server.stopping {
		return fmt.Errorf("binding upgrade limit reached")
	}
	config.Protocol = []string{Protocol}
	return nil
}

func (server *Server) handleClient(ws *websocket.Conn) {
	current := &client{ws: ws}
	ws.MaxPayloadBytes = MaxMessageBytes

	server.mu.Lock()
	if server.stopping || len(server.clients) >= maxUpgradedClients {
		server.mu.Unlock()
		return
	}
	server.wg.Add(1)
	server.clients[current] = struct{}{}
	server.mu.Unlock()
	defer server.wg.Done()
	defer server.retire(current)

	_ = ws.SetReadDeadline(time.Now().Add(authDeadline))
	var auth []byte
	if err := websocket.Message.Receive(ws, &auth); err != nil || len(auth) != authFrameBytes || auth[0] != opAuth || subtle.ConstantTimeCompare(auth[1:], server.token) != 1 {
		return
	}
	_ = ws.SetReadDeadline(time.Time{})
	server.dispatchMu.Lock()
	if err := server.send(current, opACK, 0, 0, nil); err != nil {
		server.dispatchMu.Unlock()
		return
	}
	renderer, err := server.engine.AttachRenderer()
	if err != nil {
		server.dispatchMu.Unlock()
		return
	}
	current.renderer = renderer

	server.mu.Lock()
	if server.stopping {
		server.mu.Unlock()
		_ = server.engine.DetachRenderer(renderer)
		server.dispatchMu.Unlock()
		return
	}
	replaced := server.active
	server.active = current
	server.signalLocked()
	server.mu.Unlock()
	server.dispatchMu.Unlock()
	if replaced != nil && replaced != current {
		_ = replaced.ws.Close()
	}

	for {
		var frame []byte
		if err := websocket.Message.Receive(ws, &frame); err != nil {
			return
		}
		if len(frame) == 0 || len(frame) > MaxMessageBytes {
			return
		}
		server.mu.Lock()
		admitted := !server.stopping && server.active == current
		server.mu.Unlock()
		if !admitted {
			return
		}
		server.handleRequest(current, frame)
	}
}

func (server *Server) handleRequest(current *client, frame []byte) {
	server.dispatchMu.Lock()
	defer server.dispatchMu.Unlock()
	server.mu.Lock()
	admitted := !server.stopping && server.active == current && current.renderer != 0
	server.mu.Unlock()
	if !admitted {
		return
	}
	requestID := uint64(0)
	if len(frame) >= 9 {
		requestID = binary.BigEndian.Uint64(frame[1:9])
	}
	operation, handle, payload, err := decodeRequest(frame)
	if err != nil {
		_ = server.sendError(current, requestID, err)
		return
	}
	var result uint64
	switch operation {
	case opOpenSession:
		result, err = server.engine.OpenSession(payload)
	case opExecute:
		result, err = server.engine.Execute(handle, payload)
	case opEngineCommand:
		result, err = server.engine.EngineCommand(payload)
	case opCancel:
		err = server.engine.Cancel(handle)
	case opCloseSession:
		err = server.engine.CloseSession(handle)
	case opRelease:
		err = server.engine.Release(handle)
	case opOpenStream:
		result, err = server.engine.OpenResourceStream(handle, payload)
	case opSendStream:
		err = server.engine.SendResourceStreamFrame(handle, payload)
	case opCloseStream:
		err = server.engine.CloseResourceStream(handle)
	default:
		err = fmt.Errorf("unsupported binding operation")
	}
	if err != nil {
		_ = server.sendError(current, requestID, err)
		return
	}
	if operation == opOpenSession || operation == opExecute || operation == opEngineCommand || operation == opOpenStream {
		_ = server.send(current, opAccepted, requestID, result, nil)
		return
	}
	_ = server.send(current, opACK, requestID, 0, nil)
}

func decodeRequest(frame []byte) (operation byte, handle uint64, payload []byte, err error) {
	if len(frame) < 9 {
		return 0, 0, nil, fmt.Errorf("binding request header is truncated")
	}
	operation = frame[0]
	switch operation {
	case opOpenSession, opEngineCommand:
		return operation, 0, frame[9:], nil
	case opExecute, opOpenStream, opSendStream:
		if len(frame) < 17 {
			return 0, 0, nil, fmt.Errorf("binding handle header is truncated")
		}
		return operation, binary.BigEndian.Uint64(frame[9:17]), frame[17:], nil
	case opCancel, opCloseSession, opRelease, opCloseStream:
		if len(frame) != 17 {
			return 0, 0, nil, fmt.Errorf("binding handle request length is invalid")
		}
		return operation, binary.BigEndian.Uint64(frame[9:17]), nil, nil
	default:
		return operation, 0, nil, nil
	}
}

func (server *Server) pumpEvents() {
	for {
		current, err := server.waitForClient()
		if err != nil {
			return
		}
		payload, err := server.engine.NextEvent(server.ctx, current.renderer)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, binding.ErrClosed) {
				return
			}
			if errors.Is(err, binding.ErrInvalidHandle) {
				server.retire(current)
			}
			continue
		}
		if err := server.send(current, opEvent, 0, 0, payload); err != nil {
			server.retire(current)
			continue
		}
	}
}

func (server *Server) waitForClient() (*client, error) {
	for {
		server.mu.Lock()
		if server.stopping {
			server.mu.Unlock()
			return nil, context.Canceled
		}
		if server.active != nil {
			current := server.active
			server.mu.Unlock()
			return current, nil
		}
		changed := server.stateChange
		server.mu.Unlock()
		select {
		case <-server.ctx.Done():
			return nil, server.ctx.Err()
		case <-changed:
		}
	}
}

func (server *Server) sendError(current *client, requestID uint64, failure error) error {
	payload := []byte(failure.Error())
	if len(payload) > MaxMessageBytes-responseHeaderBytes {
		payload = payload[:MaxMessageBytes-responseHeaderBytes]
	}
	return server.send(current, opError, requestID, 0, payload)
}

func (server *Server) send(current *client, operation byte, requestID, handle uint64, payload []byte) error {
	if len(payload) > MaxMessageBytes-responseHeaderBytes {
		return fmt.Errorf("binding response exceeds message limit")
	}
	frame := make([]byte, responseHeaderBytes+len(payload))
	frame[0] = operation
	binary.BigEndian.PutUint64(frame[1:9], requestID)
	binary.BigEndian.PutUint64(frame[9:17], handle)
	binary.BigEndian.PutUint32(frame[17:21], uint32(len(payload)))
	copy(frame[21:], payload)
	current.writeMu.Lock()
	defer current.writeMu.Unlock()
	_ = current.ws.SetWriteDeadline(time.Now().Add(writeDeadline))
	err := websocket.Message.Send(current.ws, frame)
	_ = current.ws.SetWriteDeadline(time.Time{})
	return err
}

func (server *Server) retire(current *client) {
	server.mu.Lock()
	if _, ok := server.clients[current]; !ok {
		server.mu.Unlock()
		return
	}
	delete(server.clients, current)
	if server.active == current {
		server.active = nil
		server.signalLocked()
	}
	server.mu.Unlock()
	current.rendererOnce.Do(func() {
		server.dispatchMu.Lock()
		defer server.dispatchMu.Unlock()
		if current.renderer != 0 {
			_ = server.engine.DetachRenderer(current.renderer)
		}
	})
	_ = current.ws.Close()
}

func (server *Server) signalLocked() {
	close(server.stateChange)
	server.stateChange = make(chan struct{})
}

func validToken(token string) bool {
	if len(token) != authTokenBytes {
		return false
	}
	for index := range token {
		value := token[index]
		if !((value >= 'A' && value <= 'Z') || (value >= 'a' && value <= 'z') || (value >= '0' && value <= '9') || value == '-' || value == '_') {
			return false
		}
	}
	return true
}

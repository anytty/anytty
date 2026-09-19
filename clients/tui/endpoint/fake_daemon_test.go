package endpoint

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/remoteauthpb"
	"github.com/anytty/anytty/proto/access/wire"
	"github.com/anytty/anytty/proto/access/wirepb"
	"github.com/anytty/anytty/shared/remoteauth"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"

	gproto "google.golang.org/protobuf/proto"
)

// fakeDaemon is an in-process minimal server that speaks the same wire
// protocol as core-v2: Hello handshake, api.execute commands and raw PTY
// attachment streams. It implements just enough terminal behavior
// (echo/stty size/exit) to exercise the endpoint client end to end.
//
// newFakeDaemon serves raw frames on a unix socket (the historical test
// transport); newTCPFakeDaemon serves the production zstd framing over tcp,
// which is what the tcp connect mode dials.
type fakeDaemon struct {
	t        *testing.T
	socket   string
	address  string
	ln       net.Listener
	framed   bool
	identity remoteauth.Identity
	stopFn   func()
	mu       sync.Mutex
	terms    map[string]*fakeTerminal
	sessions map[*fakeSession]struct{}
	nextID   int
	closed   bool
}

func newFakeDaemon(t *testing.T) *fakeDaemon {
	t.Helper()
	dir := t.TempDir()
	socket := filepath.Join(dir, "fake-daemon.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("listen fake daemon: %v", err)
	}
	return newFakeDaemonListener(t, socket, "", false, ln)
}

// newTCPFakeDaemon listens on loopback tcp and wraps every accepted
// connection with the production framed transport.
func newTCPFakeDaemon(t *testing.T) *fakeDaemon {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp fake daemon: %v", err)
	}
	return newFakeDaemonListener(t, "", ln.Addr().String(), true, ln)
}

// newFramedFakeDaemon listens on a unix socket with the production
// shared/transport/unix framing, which is what the shared local route adapter
// dials. It is the harness for the real shared connection stack.
func newFramedFakeDaemon(t *testing.T) *fakeDaemon {
	t.Helper()
	dir := t.TempDir()
	socket := filepath.Join(dir, "framed-daemon.sock")
	listener, err := unixtransport.NewListener(socket)
	if err != nil {
		t.Fatalf("listen framed fake daemon: %v", err)
	}
	d := newFakeDaemonListener(t, socket, "", true, nil)
	d.stopFn = func() {
		d.mu.Lock()
		if d.closed {
			d.mu.Unlock()
			return
		}
		d.closed = true
		sessions := make([]*fakeSession, 0, len(d.sessions))
		for session := range d.sessions {
			sessions = append(sessions, session)
		}
		d.mu.Unlock()
		_ = listener.Close()
		for _, session := range sessions {
			_ = session.link.close()
		}
	}
	go func() {
		for {
			transport, err := listener.Accept(context.Background())
			if err != nil {
				return
			}
			d.serveAccepted(&framedFakeStream{transport: transport})
		}
	}()
	return d
}

func newFakeDaemonListener(t *testing.T, socket, address string, framed bool, ln net.Listener) *fakeDaemon {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("fake daemon identity: %v", err)
	}
	identity, err := remoteauth.NewIdentity("device-fake", privateKey)
	if err != nil {
		t.Fatalf("fake daemon identity: %v", err)
	}
	d := &fakeDaemon{t: t, socket: socket, address: address, framed: framed, ln: ln, identity: identity, terms: map[string]*fakeTerminal{}, sessions: map[*fakeSession]struct{}{}}
	if ln != nil {
		go d.acceptLoop()
	}
	t.Cleanup(func() { d.stop() })
	return d
}

func (d *fakeDaemon) acceptLoop() {
	for {
		conn, err := d.ln.Accept()
		if err != nil {
			return
		}
		stream, err := d.wrap(conn)
		if err != nil {
			_ = conn.Close()
			continue
		}
		d.serveAccepted(stream)
	}
}

// serveAccepted registers one accepted connection and starts its session loop.
func (d *fakeDaemon) serveAccepted(stream fakeStream) {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		_ = stream.close()
		return
	}
	session := &fakeSession{daemon: d, link: stream, attachments: map[uint16]*fakeAttach{}}
	d.sessions[session] = struct{}{}
	d.mu.Unlock()
	go session.serve()
}

// wrap builds the session frame stream: raw wire frames over unix, or the
// production zstd framing over tcp.
func (d *fakeDaemon) wrap(conn net.Conn) (fakeStream, error) {
	if !d.framed {
		return &rawFakeStream{conn: conn, decoder: wire.NewDecoder(conn)}, nil
	}
	transport, err := newFramedTransport(conn)
	if err != nil {
		return nil, err
	}
	return &framedFakeStream{transport: transport}, nil
}

// fakeStream is one session's frame boundary. readFrame returns a decoded
// wire frame; writeFrame sends one encoded wire frame.
type fakeStream interface {
	readFrame() (uint16, uint8, []byte, error)
	writeFrame(frame []byte) error
	close() error
}

// rawFakeStream is the uncompressed test framing over a net.Conn.
type rawFakeStream struct {
	conn    net.Conn
	decoder *wire.Decoder
}

func (s *rawFakeStream) readFrame() (uint16, uint8, []byte, error) { return s.decoder.ReadFrame() }
func (s *rawFakeStream) writeFrame(frame []byte) error {
	_, err := s.conn.Write(frame)
	return err
}
func (s *rawFakeStream) close() error { return s.conn.Close() }

// framedFakeStream is the production zstd framing over tcp.
type framedFakeStream struct {
	transport Transport
}

func (s *framedFakeStream) readFrame() (uint16, uint8, []byte, error) {
	frame, err := s.transport.Recv()
	if err != nil {
		return 0, 0, nil, err
	}
	return wire.DecodeFrame(frame)
}
func (s *framedFakeStream) writeFrame(frame []byte) error { return s.transport.Send(frame) }
func (s *framedFakeStream) close() error                  { return s.transport.Close() }

func (d *fakeDaemon) stop() {
	if d.stopFn != nil {
		d.stopFn()
		return
	}
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.closed = true
	sessions := make([]*fakeSession, 0, len(d.sessions))
	for session := range d.sessions {
		sessions = append(sessions, session)
	}
	d.mu.Unlock()
	_ = d.ln.Close()
	for _, session := range sessions {
		_ = session.link.close()
	}
}

// disconnect closes every live connection but keeps terminal state, which is
// how reconnect tests observe health=offline and resubscribe.
func (d *fakeDaemon) disconnect() {
	d.mu.Lock()
	sessions := make([]*fakeSession, 0, len(d.sessions))
	for session := range d.sessions {
		sessions = append(sessions, session)
	}
	d.mu.Unlock()
	for _, session := range sessions {
		_ = session.link.close()
	}
}

func (d *fakeDaemon) forget(session *fakeSession) {
	d.mu.Lock()
	delete(d.sessions, session)
	d.mu.Unlock()
}

func (d *fakeDaemon) terminal(id string) *fakeTerminal {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.terms[id]
}

func (d *fakeDaemon) terminalCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.terms)
}

// rawWireTransport is the test-side Transport: raw framed wire bytes over a
// unix connection (the production local transport compresses packets).
type rawWireTransport struct {
	conn net.Conn
}

func dialRawWire(ctx context.Context, cfg Config) (Transport, error) {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "unix", cfg.Socket)
	if err != nil {
		return nil, err
	}
	return &rawWireTransport{conn: conn}, nil
}

// dialRawSession adapts the raw-wire test client to the sessionConn seam.
// tcp endpoints use the framed test transport (the production path relays
// through the tcp bridge and the shared local route).
func dialRawSession(ctx context.Context, cfg Config) (sessionConn, error) {
	dial := dialRawWire
	if cfg.ConnectModeName() == ConnectDirectTCP {
		dial = func(ctx context.Context, cfg Config) (Transport, error) {
			return dialTCPTransport(ctx, strings.TrimSpace(cfg.Address))
		}
	}
	conn, err := dialClient(ctx, cfg, dial, DefaultDialTimeout)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (t *rawWireTransport) Send(frame []byte) error {
	_, err := t.conn.Write(frame)
	return err
}

func (t *rawWireTransport) Recv() ([]byte, error) {
	header := make([]byte, 7)
	if _, err := io.ReadFull(t.conn, header); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(header[3:])
	payload := make([]byte, length)
	if _, err := io.ReadFull(t.conn, payload); err != nil {
		return nil, err
	}
	return append(header, payload...), nil
}

func (t *rawWireTransport) Close() error { return t.conn.Close() }

// fakeAttach is one server-side attachment.
type fakeAttach struct {
	session  *fakeSession
	terminal *fakeTerminal
	channel  uint16
	surface  string
	view     string
}

// errorEnvelope builds a typed application error result.
func errorEnvelope(requestID string, code apipb.ApiErrorCode, message string) *apipb.ResultEnvelope {
	return &apipb.ResultEnvelope{
		RequestId: requestID,
		Result:    &apipb.ResultEnvelope_Error{Error: &apipb.ApiError{Code: code, Message: message}},
	}
}

type fakeSession struct {
	daemon *fakeDaemon
	link   fakeStream

	encMu sync.Mutex

	mu          sync.Mutex
	attachments map[uint16]*fakeAttach
	nextChannel uint16
}

func (s *fakeSession) serve() {
	defer func() {
		s.daemon.forget(s)
		_ = s.link.close()
	}()
	for {
		channel, typ, payload, err := s.link.readFrame()
		if err != nil {
			return
		}
		if channel == 0 {
			if !s.control(typ, payload) {
				return
			}
			continue
		}
		s.stream(channel, typ, payload)
	}
}

func (s *fakeSession) send(channel uint16, typ uint8, payload []byte) error {
	frame, err := wire.EncodeFrame(channel, typ, payload)
	if err != nil {
		return err
	}
	s.encMu.Lock()
	defer s.encMu.Unlock()
	return s.link.writeFrame(frame)
}

func (s *fakeSession) control(typ uint8, payload []byte) bool {
	switch typ {
	case wire.TypeHello:
		response, _ := gproto.Marshal(&wirepb.Hello{Version: wire.Version, Server: "fake-daemon"})
		_ = s.send(0, wire.TypeHello, response)
		return true
	case wire.TypeRequest:
		request := &wirepb.RequestEnvelope{}
		if err := gproto.Unmarshal(payload, request); err != nil {
			return false
		}
		result := s.execute(request.GetParams())
		raw, err := gproto.Marshal(result)
		if err != nil {
			return false
		}
		envelope, _ := gproto.Marshal(&wirepb.ResponseEnvelope{Id: request.GetId(), Result: raw})
		return s.send(0, wire.TypeResponse, envelope) == nil
	case wire.TypeSessionClose:
		return false
	default:
		return true
	}
}

// execute decodes one command, runs it and stamps every result with the
// request's session generation, exactly like the core API layer does.
func (s *fakeSession) execute(payload []byte) *apipb.ResultEnvelope {
	command := &apipb.CommandEnvelope{}
	if err := gproto.Unmarshal(payload, command); err != nil {
		return errorEnvelope("", apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST, err.Error())
	}
	result := s.executeCommand(command)
	if result != nil && result.GetOriginSession() == nil && command.GetContext().GetSession() != nil {
		result.OriginSession = gproto.Clone(command.GetContext().GetSession()).(*apipb.EndpointSessionStamp)
	}
	return result
}

func (s *fakeSession) executeCommand(command *apipb.CommandEnvelope) *apipb.ResultEnvelope {
	requestID := command.GetContext().GetRequestId()
	if requestID == "" {
		return errorEnvelope("", apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST, "request context is required")
	}
	fail := func(code apipb.ApiErrorCode, message string) *apipb.ResultEnvelope {
		return errorEnvelope(requestID, code, message)
	}
	ack := &apipb.ResultEnvelope{RequestId: requestID, Result: &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}}
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_TerminalList:
		result := &apipb.TerminalListResult{}
		s.daemon.mu.Lock()
		for _, terminal := range s.daemon.terms {
			result.Terminals = append(result.Terminals, terminal.info())
		}
		s.daemon.mu.Unlock()
		return &apipb.ResultEnvelope{RequestId: requestID, Result: &apipb.ResultEnvelope_TerminalList{TerminalList: result}}
	case *apipb.CommandEnvelope_TerminalDefaults:
		return &apipb.ResultEnvelope{RequestId: requestID, Result: &apipb.ResultEnvelope_TerminalDefaults{TerminalDefaults: &apipb.TerminalDefaultsResult{Defaults: &apipb.TerminalDefaults{DefaultCommand: []string{"/bin/sh"}, DefaultCwd: "/"}}}}
	case *apipb.CommandEnvelope_TerminalCreate:
		spec := value.TerminalCreate.GetTerminal()
		if spec.GetTerminalId() == "" || len(spec.GetCommand()) == 0 {
			return fail(apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST, "terminal_id and command are required")
		}
		s.daemon.mu.Lock()
		if _, exists := s.daemon.terms[spec.GetTerminalId()]; exists {
			s.daemon.mu.Unlock()
			return fail(apipb.ApiErrorCode_API_ERROR_CODE_CONFLICT, "terminal already exists")
		}
		terminal := newFakeTerminal(spec.GetTerminalId(), spec.GetName(), spec.GetCommand(), int(spec.GetSize().GetCols()), int(spec.GetSize().GetRows()))
		s.daemon.terms[terminal.id] = terminal
		s.daemon.mu.Unlock()
		return &apipb.ResultEnvelope{RequestId: requestID, Result: &apipb.ResultEnvelope_TerminalCreate{TerminalCreate: &apipb.TerminalCreateResult{Terminal: terminal.info()}}}
	case *apipb.CommandEnvelope_TerminalAttach:
		return s.attach(requestID, value.TerminalAttach)
	case *apipb.CommandEnvelope_TerminalInput:
		attach := s.attachmentFor(value.TerminalInput.GetAttachment())
		if attach == nil {
			return fail(apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND, "attachment is not active")
		}
		attach.terminal.write(value.TerminalInput.GetData())
		return ack
	case *apipb.CommandEnvelope_TerminalResize:
		attach := s.attachmentFor(value.TerminalResize.GetAttachment())
		if attach == nil {
			return fail(apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND, "attachment is not active")
		}
		return s.resize(requestID, attach, value.TerminalResize)
	case *apipb.CommandEnvelope_TerminalDetach:
		attach := s.attachmentFor(value.TerminalDetach.GetAttachment())
		if attach != nil {
			s.dropAttachment(attach)
		}
		return ack
	case *apipb.CommandEnvelope_TerminalKill:
		terminal := s.daemon.terminal(value.TerminalKill.GetTerminal().GetTerminalId())
		if terminal == nil {
			return fail(apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND, "terminal not found")
		}
		terminal.exit(0)
		return ack
	case *apipb.CommandEnvelope_TerminalRestart:
		terminal := s.daemon.terminal(value.TerminalRestart.GetTerminal().GetTerminalId())
		if terminal == nil {
			return fail(apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND, "terminal not found")
		}
		terminal.restart()
		return ack
	case *apipb.CommandEnvelope_TerminalRemove:
		id := value.TerminalRemove.GetTerminal().GetTerminalId()
		s.daemon.mu.Lock()
		terminal := s.daemon.terms[id]
		delete(s.daemon.terms, id)
		s.daemon.mu.Unlock()
		if terminal == nil {
			return fail(apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND, "terminal not found")
		}
		return ack
	case *apipb.CommandEnvelope_LiveScreenNext:
		terminal := s.daemon.terminal(value.LiveScreenNext.GetTerminal().GetTerminalId())
		if terminal == nil {
			return fail(apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND, "terminal not found")
		}
		return &apipb.ResultEnvelope{RequestId: requestID, Result: &apipb.ResultEnvelope_LiveScreen{LiveScreen: terminal.snapshot()}}
	case *apipb.CommandEnvelope_ClientAccessIdentity:
		challenge := value.ClientAccessIdentity.GetChallenge()
		proof, err := remoteauth.SignDeviceIdentityProof(s.daemon.identity, challenge)
		if err != nil {
			return fail(apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST, err.Error())
		}
		return &apipb.ResultEnvelope{RequestId: requestID, Result: &apipb.ResultEnvelope_ClientAccessIdentity{ClientAccessIdentity: &apipb.ClientAccessIdentityResult{
			Identity: &remoteauthpb.ClientAccessIdentityResult{
				DeviceId:          s.daemon.identity.DeviceID,
				DeviceFingerprint: s.daemon.identity.Fingerprint,
				DevicePublicKey:   s.daemon.identity.PublicKey,
			},
			Challenge: append([]byte(nil), challenge...),
			Proof:     proof,
		}}}
	default:
		return fail(apipb.ApiErrorCode_API_ERROR_CODE_UNSUPPORTED_CAPABILITY, "unsupported command")
	}
}

// resize handles the owner CAS.
func (s *fakeSession) resize(requestID string, attach *fakeAttach, command *apipb.TerminalResizeCommand) *apipb.ResultEnvelope {
	terminal := attach.terminal
	terminal.mu.Lock()
	if attach != terminal.owner || (!command.GetTakeOwnership() && command.GetExpectedOwnerEpoch() != terminal.epoch) {
		size := terminal.sizeProtoLocked()
		epoch := terminal.epoch
		terminal.mu.Unlock()
		control := &apipb.ResizeControl{CanResize: false, Reason: apipb.ResizeControlReason_RESIZE_CONTROL_REASON_FOLLOWER, Ownership: &apipb.ResizeOwnership{Size: size, Epoch: epoch}}
		return &apipb.ResultEnvelope{RequestId: requestID, Result: &apipb.ResultEnvelope_TerminalResize{TerminalResize: &apipb.TerminalResizeResult{Size: size, Resized: false, ResizeControl: control}}}
	}
	terminal.cols = int(command.GetSize().GetCols())
	terminal.rows = int(command.GetSize().GetRows())
	size := terminal.sizeProtoLocked()
	control := &apipb.ResizeControl{CanResize: true, Reason: apipb.ResizeControlReason_RESIZE_CONTROL_REASON_OWNER, Ownership: &apipb.ResizeOwnership{Size: size, Epoch: terminal.epoch}}
	terminal.mu.Unlock()
	return &apipb.ResultEnvelope{RequestId: requestID, Result: &apipb.ResultEnvelope_TerminalResize{TerminalResize: &apipb.TerminalResizeResult{Size: size, Resized: true, ResizeControl: control}}}
}

func (s *fakeSession) attach(requestID string, command *apipb.TerminalAttachCommand) *apipb.ResultEnvelope {
	terminal := s.daemon.terminal(command.GetTerminal().GetTerminalId())
	if terminal == nil {
		return errorEnvelope(requestID, apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND, "terminal not found")
	}
	terminal.mu.Lock()
	terminal.owner = nil
	terminal.mu.Unlock()
	s.mu.Lock()
	s.nextChannel++
	channel := s.nextChannel
	attach := &fakeAttach{session: s, terminal: terminal, channel: channel, surface: command.GetSurfaceId(), view: command.GetViewId()}
	s.attachments[channel] = attach
	s.mu.Unlock()
	terminal.mu.Lock()
	terminal.owner = attach
	terminal.epoch++
	epoch := terminal.epoch
	terminal.mu.Unlock()
	token := make([]byte, 8)
	binary.BigEndian.PutUint16(token[:2], channel)
	resource := &apipb.ResourceHandle{
		Kind:        apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT,
		OpaqueToken: token,
		Generation:  1,
	}
	handle := &apipb.AttachmentHandle{
		Resource:  resource,
		Terminal:  &apipb.TerminalRef{TerminalId: terminal.id},
		SurfaceId: command.GetSurfaceId(),
		ViewId:    command.GetViewId(),
	}
	result := &apipb.TerminalAttachResult{
		Attachment:   handle,
		Mode:         command.GetMode(),
		ResizePolicy: command.GetResizePolicy(),
		Size:         terminal.sizeProto(),
		ResizeControl: &apipb.ResizeControl{
			CanResize: true,
			Reason:    apipb.ResizeControlReason_RESIZE_CONTROL_REASON_OWNER,
			Ownership: &apipb.ResizeOwnership{Size: terminal.sizeProto(), Epoch: epoch, OwnerViewId: command.GetViewId()},
		},
	}
	return &apipb.ResultEnvelope{RequestId: requestID, Result: &apipb.ResultEnvelope_TerminalAttach{TerminalAttach: result}}
}

func (s *fakeSession) attachmentFor(resource *apipb.ResourceHandle) *fakeAttach {
	if resource == nil || len(resource.GetOpaqueToken()) < 2 {
		return nil
	}
	channel := binary.BigEndian.Uint16(resource.GetOpaqueToken()[:2])
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attachments[channel]
}

func (s *fakeSession) dropAttachment(attach *fakeAttach) {
	s.mu.Lock()
	delete(s.attachments, attach.channel)
	s.mu.Unlock()
	if attach.terminal != nil {
		attach.terminal.removeSubscriber(s)
	}
	_ = s.send(attach.channel, wire.TypeClosed, wire.EncodeClosedPayload(-1))
}

// stream handles client -> daemon attachment frames.
func (s *fakeSession) stream(channel uint16, typ uint8, payload []byte) {
	s.mu.Lock()
	attach := s.attachments[channel]
	s.mu.Unlock()
	if attach == nil {
		return
	}
	switch typ {
	case wire.TypeBootstrapDone:
		_ = s.send(channel, wire.TypeStreamReady, nil)
		attach.terminal.addSubscriber(s)
	case wire.TypeClosed:
		s.dropAttachment(attach)
	}
}

type fakeTerminal struct {
	mu          sync.Mutex
	id          string
	name        string
	command     []string
	cols        int
	rows        int
	exited      bool
	exitCode    int
	epoch       uint64
	owner       *fakeAttach
	subscribers map[*fakeSession]struct{}
	transcript  []byte
	line        []byte
}

func newFakeTerminal(id, name string, command []string, cols, rows int) *fakeTerminal {
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	return &fakeTerminal{id: id, name: name, command: command, cols: cols, rows: rows, subscribers: map[*fakeSession]struct{}{}}
}

func (t *fakeTerminal) info() *apipb.TerminalInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	info := &apipb.TerminalInfo{
		Ref:     &apipb.TerminalRef{TerminalId: t.id},
		Name:    t.name,
		Command: append([]string(nil), t.command...),
		Size:    t.sizeProtoLocked(),
		State:   apipb.TerminalState_TERMINAL_STATE_RUNNING,
	}
	if t.exited {
		info.State = apipb.TerminalState_TERMINAL_STATE_EXITED
		code := int32(t.exitCode)
		info.ExitCode = &code
	}
	return info
}

func (t *fakeTerminal) sizeProto() *apipb.TerminalSize {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.sizeProtoLocked()
}

func (t *fakeTerminal) sizeProtoLocked() *apipb.TerminalSize {
	return &apipb.TerminalSize{Cols: uint32(t.cols), Rows: uint32(t.rows)}
}

func (t *fakeTerminal) addSubscriber(session *fakeSession) {
	t.mu.Lock()
	if !t.exited {
		t.subscribers[session] = struct{}{}
	}
	t.mu.Unlock()
}

func (t *fakeTerminal) removeSubscriber(session *fakeSession) {
	t.mu.Lock()
	delete(t.subscribers, session)
	t.mu.Unlock()
}

func (t *fakeTerminal) publish(data []byte) {
	t.mu.Lock()
	t.transcript = append(t.transcript, data...)
	sessions := make([]*fakeSession, 0, len(t.subscribers))
	for session := range t.subscribers {
		sessions = append(sessions, session)
	}
	t.mu.Unlock()
	for _, session := range sessions {
		_ = session.send(t.channelOf(session), wire.TypePTYOutput, data)
	}
}

func (t *fakeTerminal) channelOf(session *fakeSession) uint16 {
	session.mu.Lock()
	defer session.mu.Unlock()
	for channel, attach := range session.attachments {
		if attach.terminal == t {
			return channel
		}
	}
	return 0
}

// write emulates a PTY: it echoes the input then interprets whole lines.
func (t *fakeTerminal) write(data []byte) {
	t.publish(data)
	t.mu.Lock()
	for _, b := range data {
		switch b {
		case '\r', '\n':
			line := strings.TrimSpace(string(t.line))
			t.line = t.line[:0]
			cols, rows := t.cols, t.rows
			exit, hasExit := parseExit(line)
			t.mu.Unlock()
			t.respond(line, cols, rows, exit, hasExit)
			t.mu.Lock()
			if t.exited {
				t.mu.Unlock()
				return
			}
		default:
			t.line = append(t.line, b)
		}
	}
	t.mu.Unlock()
}

func (t *fakeTerminal) respond(line string, cols, rows int, exit int, hasExit bool) {
	switch {
	case line == "stty size":
		t.publish([]byte(fmt.Sprintf("%d %d\r\n", rows, cols)))
	case strings.HasPrefix(line, "echo "):
		t.publish([]byte(strings.TrimPrefix(line, "echo ") + "\r\n"))
	case hasExit:
		t.exit(exit)
	}
}

func parseExit(line string) (int, bool) {
	if line == "exit" {
		return 0, true
	}
	if strings.HasPrefix(line, "exit ") {
		if code, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "exit "))); err == nil {
			return code, true
		}
	}
	return 0, false
}

// restart revives an exited terminal with a cleared transcript, matching the
// daemon RestartTerminal contract.
func (t *fakeTerminal) restart() {
	t.mu.Lock()
	t.exited = false
	t.exitCode = 0
	t.transcript = nil
	t.line = nil
	t.mu.Unlock()
}

func (t *fakeTerminal) exit(code int) {
	t.mu.Lock()
	if t.exited {
		t.mu.Unlock()
		return
	}
	t.exited = true
	t.exitCode = code
	sessions := make([]*fakeSession, 0, len(t.subscribers))
	for session := range t.subscribers {
		sessions = append(sessions, session)
	}
	t.subscribers = map[*fakeSession]struct{}{}
	t.mu.Unlock()
	payload := wire.EncodeClosedPayload(code)
	for _, session := range sessions {
		if channel := t.channelOf(session); channel != 0 {
			_ = session.send(channel, wire.TypeClosed, payload)
		}
	}
}

// screenTail returns the last rows lines of the transcript.
func (t *fakeTerminal) screenTailLocked() []string {
	text := strings.ReplaceAll(string(t.transcript), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) > t.rows {
		lines = lines[len(lines)-t.rows:]
	}
	return lines
}

// snapshot builds a full-replace native screen result from the transcript.
func (t *fakeTerminal) snapshot() *apipb.NativeScreenResult {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := &apipb.NativeScreenResult{
		Terminal:     &apipb.TerminalRef{TerminalId: t.id},
		LiveRevision: t.epoch,
		Size:         t.sizeProtoLocked(),
		FullReplace:  true,
	}
	for index, line := range t.screenTailLocked() {
		row := &apipb.ScreenRow{}
		for _, r := range line {
			row.Cells = append(row.Cells, &apipb.ScreenCell{Content: string(r), Width: 1})
		}
		result.RowReplacements = append(result.RowReplacements, &apipb.ScreenRowReplace{RowIndex: int32(index), Row: row})
	}
	return result
}

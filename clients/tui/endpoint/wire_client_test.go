package endpoint

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
	"github.com/anytty/anytty/proto/access/wirepb"

	gproto "google.golang.org/protobuf/proto"
)

// Transport is the framed transport of one endpoint connection. The local
// unix implementation is shared/transport/unix (compressed packets); tests
// inject a raw-frame implementation.
type Transport interface {
	Send(frame []byte) error
	Recv() ([]byte, error)
	Close() error
}

// Dialer opens one endpoint transport.
type Dialer func(ctx context.Context, cfg Config) (Transport, error)

// clientName is the protocol Hello client identity of the tui2 host.
const clientName = "tui2-host"

// callResult is one correlated control response.
type callResult struct {
	result []byte
	err    error
}

// client is one wire connection to a daemon. Control frames are correlated by
// request id; attachment frames are demultiplexed per channel.
type client struct {
	cfg  Config
	sess *apipb.EndpointSessionStamp

	encMu     sync.Mutex
	transport Transport

	mu      sync.Mutex
	nextID  uint64
	nextOp  uint64
	waiters map[uint64]chan callResult
	streams map[uint16]*stream
	done    chan struct{}
	err     error

	instance string
}

// dialClient connects and performs the protocol Hello handshake.
func dialClient(ctx context.Context, cfg Config, dial Dialer, timeout time.Duration) (*client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := cfg.UnsupportedModeError(); err != nil {
		return nil, err
	}
	transport, err := dial(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("endpoint %q: dial %s: %w", cfg.Name, cfg.DialTarget(), err)
	}
	c := &client{
		cfg:       cfg,
		transport: transport,
		waiters:   map[uint64]chan callResult{},
		streams:   map[uint16]*stream{},
		done:      make(chan struct{}),
		sess: &apipb.EndpointSessionStamp{
			EndpointId: cfg.Name,
			RouteId:    cfg.ConnectModeName(),
			Generation: 1,
		},
		instance: fmt.Sprintf("%x", time.Now().UnixNano()),
	}
	go c.readLoop()
	if err := c.hello(ctx, timeout); err != nil {
		_ = c.close()
		return nil, fmt.Errorf("endpoint %q: handshake: %w", cfg.Name, err)
	}
	return c, nil
}

// hello sends the versioned Hello and validates the daemon answer.
func (c *client) hello(ctx context.Context, timeout time.Duration) error {
	payload, err := gproto.Marshal(&wirepb.Hello{Version: uint32(wire.Version), Client: clientName})
	if err != nil {
		return err
	}
	helloCh := make(chan callResult, 1)
	c.mu.Lock()
	c.waiters[0] = helloCh
	c.mu.Unlock()
	if err := c.sendFrame(0, wire.TypeHello, payload); err != nil {
		return err
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case res := <-helloCh:
		return res.err
	case <-ctx.Done():
		return ctx.Err()
	case <-c.done:
		return c.closeErr()
	case <-timer.C:
		return fmt.Errorf("timeout waiting for hello (wire version %d)", wire.Version)
	}
}

func (c *client) closeErr() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	return ErrEndpointClosed
}

// Done is closed when the read loop exits.
func (c *client) Done() <-chan struct{} { return c.done }

// Err reports why the connection ended.
func (c *client) Err() error {
	select {
	case <-c.done:
		return c.closeErr()
	default:
		return nil
	}
}

func (c *client) sendFrame(channel uint16, typ uint8, payload []byte) error {
	frame, err := wire.EncodeFrame(channel, typ, payload)
	if err != nil {
		return err
	}
	c.encMu.Lock()
	defer c.encMu.Unlock()
	return c.transport.Send(frame)
}

// request performs one api.execute control call: marshal the command, send a
// TypeRequest frame, and wait for the correlated response.
func (c *client) request(ctx context.Context, command *apipb.CommandEnvelope) ([]byte, error) {
	params, err := gproto.Marshal(command)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	waiter := make(chan callResult, 1)
	c.waiters[id] = waiter
	c.mu.Unlock()
	envelope, err := gproto.Marshal(&wirepb.RequestEnvelope{Id: id, Method: "api.execute", Params: params})
	if err != nil {
		c.mu.Lock()
		delete(c.waiters, id)
		c.mu.Unlock()
		return nil, err
	}
	if err := c.sendFrame(0, wire.TypeRequest, envelope); err != nil {
		c.mu.Lock()
		delete(c.waiters, id)
		c.mu.Unlock()
		return nil, err
	}
	select {
	case res := <-waiter:
		return res.result, res.err
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.waiters, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	case <-c.done:
		return nil, c.closeErr()
	}
}

// execute runs one application command and decodes the typed ResultEnvelope.
func (c *client) execute(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	command = gproto.Clone(command).(*apipb.CommandEnvelope)
	if command.Context == nil {
		command.Context = c.newContext()
	}
	raw, err := c.request(ctx, command)
	if err != nil {
		return nil, err
	}
	envelope := &apipb.ResultEnvelope{}
	if err := gproto.Unmarshal(raw, envelope); err != nil {
		return nil, fmt.Errorf("endpoint %q: decode application result: %w", c.cfg.Name, err)
	}
	if err := apiErrorFromProto(envelope.GetError()); err != nil {
		return nil, err
	}
	return envelope, nil
}

// newContext builds the request correlation for one command.
func (c *client) newContext() *apipb.RequestContext {
	c.mu.Lock()
	c.nextOp++
	seq := c.nextOp
	c.mu.Unlock()
	return &apipb.RequestContext{
		RequestId:  fmt.Sprintf("tui2-%s-%d", c.instance, seq),
		ApiVersion: &apipb.ApiVersion{Major: 1},
		Session:    c.sessionProto(),
	}
}

func (c *client) sessionProto() *apipb.EndpointSessionStamp {
	return gproto.Clone(c.sess).(*apipb.EndpointSessionStamp)
}

// operation stamps one mutating command with the session fence the daemon
// validates against context.session.
func (c *client) operation(id string) *apipb.OperationStamp {
	return &apipb.OperationStamp{Session: c.sessionProto(), OperationId: id}
}

// readLoop demultiplexes every inbound frame until I/O fails.
func (c *client) readLoop() {
	for {
		frame, err := c.transport.Recv()
		if err != nil {
			// A lost connection is always transient for the manager: it must
			// never look like a terminal exit, so the underlying io.EOF is not
			// wrapped in an errors.Is-compatible way.
			c.fail(fmt.Errorf("endpoint %q: connection lost: %v", c.cfg.Name, err))
			return
		}
		channel, typ, payload, err := wire.DecodeFrame(frame)
		if err != nil {
			c.fail(fmt.Errorf("endpoint %q: bad frame: %v", c.cfg.Name, err))
			return
		}
		if channel == 0 {
			c.handleControl(typ, payload)
			continue
		}
		c.handleStream(channel, typ, payload)
	}
}

func (c *client) handleControl(typ uint8, payload []byte) {
	switch typ {
	case wire.TypeHello:
		hello := &wirepb.Hello{}
		if err := gproto.Unmarshal(payload, hello); err != nil {
			c.deliverHello(callResult{err: err})
			return
		}
		if hello.GetVersion() != uint32(wire.Version) {
			c.deliverHello(callResult{err: fmt.Errorf("unsupported wire version %d (want %d)", hello.GetVersion(), wire.Version)})
			return
		}
		c.deliverHello(callResult{})
	case wire.TypeResponse, wire.TypeResponseBinary:
		envelope := &wirepb.ResponseEnvelope{}
		if err := gproto.Unmarshal(payload, envelope); err != nil {
			return
		}
		c.deliver(envelope.GetId(), callResult{result: envelope.GetResult()})
	case wire.TypeError:
		envelope := &wirepb.ErrorEnvelope{}
		if err := gproto.Unmarshal(payload, envelope); err != nil {
			return
		}
		message := fmt.Sprintf("protocol error %d: %s", envelope.GetError().GetCode(), envelope.GetError().GetMessage())
		c.deliver(envelope.GetId(), callResult{err: errors.New(message)})
	case wire.TypeSessionClose:
		c.fail(fmt.Errorf("endpoint %q: daemon closed the session", c.cfg.Name))
	case wire.TypeEvent:
		// Terminal lifecycle events are not consumed yet: list + attach
		// streams are authoritative for the v2 host (see ENDPOINTS §4).
	default:
	}
}

func (c *client) deliverHello(res callResult) {
	c.mu.Lock()
	waiter := c.waiters[0]
	delete(c.waiters, 0)
	c.mu.Unlock()
	if waiter != nil {
		waiter <- res
	}
}

func (c *client) deliver(id uint64, res callResult) {
	c.mu.Lock()
	waiter := c.waiters[id]
	delete(c.waiters, id)
	c.mu.Unlock()
	if waiter != nil {
		waiter <- res
	}
}

func (c *client) handleStream(channel uint16, typ uint8, payload []byte) {
	c.mu.Lock()
	s := c.streams[channel]
	c.mu.Unlock()
	if s == nil {
		return
	}
	switch typ {
	case wire.TypeStreamReady, wire.TypePTYOutput, wire.TypeSyncLost, wire.TypeClosed:
		s.push(typ, payload)
		if typ == wire.TypeClosed {
			s.close(nil)
		}
	default:
		s.push(typ, payload)
	}
}

// fail closes the connection and wakes every waiter and stream.
func (c *client) fail(err error) {
	c.mu.Lock()
	if c.err == nil {
		c.err = err
	}
	select {
	case <-c.done:
		c.mu.Unlock()
		return
	default:
	}
	close(c.done)
	waiters := c.waiters
	c.waiters = map[uint64]chan callResult{}
	streams := c.streams
	c.streams = map[uint16]*stream{}
	c.mu.Unlock()
	for _, waiter := range waiters {
		waiter <- callResult{err: err}
	}
	for _, s := range streams {
		s.close(err)
	}
}

func (c *client) close() error {
	err := c.transport.Close()
	c.fail(ErrEndpointClosed)
	return err
}

// streamFor returns the demultiplexer of one attachment channel, creating it
// on first use.
func (c *client) streamFor(channel uint16) *stream {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.streams[channel]
	if s == nil {
		s = newStream()
		c.streams[channel] = s
	}
	return s
}

func (c *client) dropStream(channel uint16) {
	c.mu.Lock()
	s := c.streams[channel]
	delete(c.streams, channel)
	c.mu.Unlock()
	if s != nil {
		s.close(nil)
	}
}

// --- typed application commands ---------------------------------------------

func (c *client) list(ctx context.Context) ([]*apipb.TerminalInfo, error) {
	result, err := c.execute(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}})
	if err != nil {
		return nil, err
	}
	return result.GetTerminalList().GetTerminals(), nil
}

func (c *client) defaults(ctx context.Context) (*apipb.TerminalDefaults, error) {
	result, err := c.execute(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalDefaults{TerminalDefaults: &apipb.TerminalDefaultsCommand{}}})
	if err != nil {
		return nil, err
	}
	return result.GetTerminalDefaults().GetDefaults(), nil
}

func (c *client) create(ctx context.Context, spec *apipb.TerminalCreateSpec) (*apipb.TerminalInfo, error) {
	result, err := c.execute(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalCreate{TerminalCreate: &apipb.TerminalCreateCommand{Terminal: spec}}})
	if err != nil {
		return nil, err
	}
	return result.GetTerminalCreate().GetTerminal(), nil
}

func (c *client) kill(ctx context.Context, id string) error {
	_, err := c.execute(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalKill{TerminalKill: &apipb.TerminalKillCommand{Terminal: c.terminalRef(id)}}})
	return err
}

func (c *client) remove(ctx context.Context, id string) error {
	_, err := c.execute(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalRemove{TerminalRemove: &apipb.TerminalRemoveCommand{Terminal: c.terminalRef(id)}}})
	return err
}

func (c *client) restart(ctx context.Context, id string) error {
	_, err := c.execute(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalRestart{TerminalRestart: &apipb.TerminalRestartCommand{Terminal: c.terminalRef(id)}}})
	return err
}

func (c *client) terminalRef(id string) *apipb.TerminalRef {
	return &apipb.TerminalRef{EndpointId: c.cfg.Name, TerminalId: id}
}

// openAttachment reserves one collaborator attachment. The raw PTY stream
// does not start until startStream, so the caller can seed the authoritative
// screen snapshot first without replaying output twice.
func (c *client) openAttachment(ctx context.Context, id, surface, view string, cols, rows int) (*attachment, error) {
	operation := c.operation(fmt.Sprintf("attach-%s-%s", id, c.instance))
	ref := c.terminalRef(id)
	command := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalAttach{TerminalAttach: &apipb.TerminalAttachCommand{
		Terminal:     ref,
		Mode:         apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR,
		ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		SurfaceId:    surface,
		ViewId:       view,
		Operation:    operation,
	}}}
	command.Context = c.newContext()
	ensured, err := c.execute(ctx, command)
	if err != nil {
		return nil, err
	}
	result := ensured.GetTerminalAttach()
	handle := result.GetAttachment()
	if handle == nil || handle.GetResource() == nil {
		return nil, fmt.Errorf("endpoint %q: attach %s returned no attachment resource", c.cfg.Name, id)
	}
	resource := handle.GetResource()
	channel := channelFromResource(resource)
	if channel == 0 {
		if err := c.release(ctx, resource); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("endpoint %q: attach %s returned an invalid channel token", c.cfg.Name, id)
	}
	att := &attachment{
		client:   c,
		terminal: id,
		resource: resource,
		channel:  channel,
		stream:   c.streamFor(channel),
		surface:  surface,
		view:     view,
		mode:     result.GetMode(),
		policy:   result.GetResizePolicy(),
		size:     result.GetSize(),
	}
	if control := result.GetResizeControl(); control.GetOwnership() != nil {
		att.epoch = control.GetOwnership().GetEpoch()
	}
	return att, nil
}

// startStream asks the daemon to begin sending raw PTY output on the
// attachment channel and waits for the ready handshake.
func (c *client) startStream(ctx context.Context, att *attachment, timeout time.Duration) error {
	if err := c.sendFrame(att.channel, wire.TypeBootstrapDone, nil); err != nil {
		c.dropStream(att.channel)
		return err
	}
	readyCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if _, err := waitStreamReady(readyCtx, att.stream); err != nil {
		c.dropStream(att.channel)
		_ = c.release(context.Background(), att.resource)
		return fmt.Errorf("endpoint %q: attach %s stream: %w", c.cfg.Name, att.terminal, err)
	}
	return nil
}

// waitStreamReady consumes frames until the daemon confirms the stream is
// live. PTY output that raced ahead of the ready frame is re-queued ahead of
// newer frames, preserving order.
func waitStreamReady(ctx context.Context, s *stream) (*stream, error) {
	for {
		frame, err := s.recv(ctx)
		if err != nil {
			return nil, err
		}
		switch frame.typ {
		case wire.TypeStreamReady:
			return s, nil
		case wire.TypeClosed:
			return nil, fmt.Errorf("stream closed before ready (code %d)", decodeClosed(frame.payload))
		case wire.TypeSyncLost:
			return nil, ErrStreamSyncLost
		default:
			// A daemon that starts emitting before ready is tolerated: the
			// frame is re-queued ahead of newer frames, preserving order.
			s.pushFront(frame)
			return s, nil
		}
	}
}

func (c *client) input(ctx context.Context, att *attachment, data []byte) error {
	operation := c.operation(fmt.Sprintf("input-%d", time.Now().UnixNano()))
	command := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalInput{TerminalInput: &apipb.TerminalInputCommand{
		Attachment: att.resource,
		Operation:  operation,
		Data:       append([]byte(nil), data...),
	}}}
	command.Context = c.newContext()
	_, err := c.execute(ctx, command)
	return err
}

// resize applies a window size with the daemon resize-owner CAS: the first
// call takes ownership, later calls carry the observed epoch.
func (c *client) resize(ctx context.Context, att *attachment, cols, rows int, takeOwnership bool, expectedEpoch uint64) (*apipb.TerminalResizeResult, error) {
	operation := c.operation(fmt.Sprintf("resize-%d", time.Now().UnixNano()))
	command := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalResize{TerminalResize: &apipb.TerminalResizeCommand{
		Attachment:         att.resource,
		Operation:          operation,
		Size:               &apipb.TerminalSize{Cols: uint32(cols), Rows: uint32(rows)},
		ResizePolicy:       apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		TakeOwnership:      takeOwnership,
		ExpectedOwnerEpoch: expectedEpoch,
	}}}
	command.Context = c.newContext()
	result, err := c.execute(ctx, command)
	if err != nil {
		return nil, err
	}
	envelope := result.GetTerminalResize()
	if envelope == nil {
		return nil, fmt.Errorf("endpoint %q: resize returned no result", c.cfg.Name)
	}
	if control := envelope.GetResizeControl(); control.GetOwnership() != nil {
		if epoch := control.GetOwnership().GetEpoch(); epoch > att.epoch {
			att.epoch = epoch
		}
	}
	return envelope, nil
}

func (c *client) liveScreen(ctx context.Context, id string, observed uint64) (*apipb.NativeScreenResult, error) {
	command := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_LiveScreenNext{LiveScreenNext: &apipb.LiveScreenNextCommand{
		Terminal:         c.terminalRef(id),
		ObservedRevision: observed,
	}}}
	command.Context = c.newContext()
	result, err := c.execute(ctx, command)
	if err != nil {
		return nil, err
	}
	screen := result.GetLiveScreen()
	if screen == nil {
		return nil, fmt.Errorf("endpoint %q: live screen returned no result", c.cfg.Name)
	}
	return screen, nil
}

// detach releases one attachment resource after stopping its stream.
func (c *client) detach(ctx context.Context, att *attachment) error {
	_ = c.sendFrame(att.channel, wire.TypeClosed, nil)
	att.stream.close(nil)
	c.dropStream(att.channel)
	return c.release(ctx, att.resource)
}

func (c *client) release(ctx context.Context, resource *apipb.ResourceHandle) error {
	command := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalDetach{TerminalDetach: &apipb.TerminalDetachCommand{
		Attachment: resource,
		Operation:  c.operation(fmt.Sprintf("detach-%d", time.Now().UnixNano())),
	}}}
	command.Context = c.newContext()
	_, err := c.execute(ctx, command)
	return err
}

// channelFromResource extracts the attachment channel encoded in the opaque
// token prefix chosen by the daemon protocol binding (see core attach).
func channelFromResource(resource *apipb.ResourceHandle) uint16 {
	token := resource.GetOpaqueToken()
	if len(token) < 2 {
		return 0
	}
	return binary.BigEndian.Uint16(token[:2])
}

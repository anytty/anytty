package core

import (
	"errors"
	"io"
	"sync"

	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// Errors returned by the client.
var (
	// ErrNoHello means a view or result was sent before the first HELLO.
	ErrNoHello = errors.New("tui2/sdk: no hello received")
	// ErrNilRoot means Commit was called without a root box.
	ErrNilRoot = errors.New("tui2/sdk: nil root box")
)

// Keys is the routing declaration carried by a view (PROTOCOL §2, §6.5).
type Keys struct {
	// Claim lists the keys the program wants to receive.
	Claim []string
	// All sends every key to the program; the program must then clear
	// focus itself.
	All bool
}

// Handlers receives the host events of one connection. Every field is
// optional; a nil handler drops that event. Handlers run on the Loop
// goroutine, so they may mutate program state and call Emit/Commit directly.
type Handlers struct {
	Hello        func(*pb.Hello)
	Sources      func([]*pb.Source)
	Key          func(*pb.KeyEvent)
	Paste        func(*pb.PasteEvent)
	Mouse        func(*pb.MouseEvent)
	Wheel        func(*pb.WheelEvent)
	Resize       func(cols, rows int)
	Notice       func(level, message string)
	Component    func(*pb.ComponentEvent)
	ViewRejected func(epoch, rev uint64, reason string)
	// Response sees every RESPONSE after its request-specific callback.
	Response func(*pb.Response)
	// Stream receives every STREAM frame for the streams this program opened
	// (access.stream.open): data/close/error. The program allocates the
	// stream id and sends frames back with SendStream.
	Stream func(*pb.StreamFrame)
}

// viewScratch is the reusable Commit envelope: a view and its keys. It lives
// in a pool because Commit runs per frame; the pooled messages are only
// touched while writeMu is held, and Marshal does not read cached sizes.
type viewScratch struct {
	view pb.View
	keys pb.Keys
}

var viewScratchPool = sync.Pool{New: func() any { return &viewScratch{} }}

// resultScratchPool reuses RESULT envelopes for Emit.
var resultScratchPool = sync.Pool{New: func() any { return &pb.Result{} }}

// Client is one layout-program connection: it reads HELLO/EVENT/RESPONSE
// frames and writes VIEW/RESULT frames (PROTOCOL §0). It is safe for
// concurrent Emit/Commit calls from handler goroutines; Loop is the only
// reader and should run once.
type Client struct {
	enc *wire.Encoder
	dec *wire.Decoder
	// w is the raw program->host writer, used by CommitDelta for the
	// size-guarded frame path where one of two pre-marshalled frames is
	// written (wire.Encoder cannot take pre-marshalled bytes).
	w io.Writer

	writeMu sync.Mutex

	mu        sync.Mutex
	handlers  Handlers
	hello     *pb.Hello
	epoch     uint64
	rev       uint64
	requestID uint64
	pending   map[uint64]func(*pb.Response)

	// base is the box tree the host is assumed to have after the last frame
	// this client sent in the current epoch, and baseRev is that frame's
	// rev. A full VIEW sets base to the committed root; a VIEW_DELTA sets it
	// to the post-delta tree. Anything that can desynchronise the host
	// (HELLO, view_rejected) drops it, so the next CommitDelta falls back to
	// a full snapshot. Guarded by mu.
	base    *pb.Box
	baseRev uint64
}

// New returns a Client reading host frames from r and writing program frames
// to w.
func New(r io.Reader, w io.Writer, handlers Handlers) *Client {
	return &Client{
		enc:      wire.NewEncoder(w, wire.RoleProgram, wire.DefaultMaxMessageBytes),
		dec:      wire.NewDecoder(r, wire.RoleProgram, wire.DefaultMaxMessageBytes),
		w:        w,
		handlers: handlers,
		pending:  map[uint64]func(*pb.Response){},
	}
}

// SetHandlers replaces the event handlers. It must be called before Loop.
func (c *Client) SetHandlers(handlers Handlers) {
	c.mu.Lock()
	c.handlers = handlers
	c.mu.Unlock()
}

// Loop reads frames until the host closes the connection. A clean EOF returns
// nil; a protocol or decode error returns it so the process can exit.
func (c *Client) Loop() error {
	for {
		t, payload, err := c.dec.Decode()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		m, err := wire.UnmarshalPayload(t, payload)
		if err != nil {
			return err
		}
		switch t {
		case wire.TypeHello:
			c.dispatchHello(m.(*pb.Hello))
		case wire.TypeEvent:
			c.dispatchEvent(m.(*pb.Event))
		case wire.TypeResponse:
			c.dispatchResponse(m.(*pb.Response))
		case wire.TypeStream:
			c.dispatchStream(m.(*pb.StreamFrame))
		}
	}
}

// SendStream writes one program -> host STREAM frame (stream data, ack,
// close or cancel). The plan program allocates stream ids.
func (c *Client) SendStream(frame *pb.StreamFrame) error {
	if frame == nil {
		return errors.New("sdk: nil stream frame")
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.enc.Encode(wire.TypeStream, frame)
}

func (c *Client) dispatchStream(frame *pb.StreamFrame) {
	if frame == nil {
		return
	}
	c.mu.Lock()
	handlers := c.handlers
	c.mu.Unlock()
	if handlers.Stream != nil {
		handlers.Stream(frame)
	}
}

// Hello returns the last HELLO, or nil before the handshake.
func (c *Client) Hello() *pb.Hello {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hello
}

// Epoch returns the current epoch (0 before the first HELLO).
func (c *Client) Epoch() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.epoch
}

// Supports reports whether the last HELLO advertised feature (PROTOCOL §1).
// It returns false before the handshake and false for an absent feature, so a
// program can treat the protocol default (not supported) safely.
func (c *Client) Supports(feature string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hello.GetFeatures()[feature]
}

// HasBase reports whether a committed view baseline exists in the current
// epoch. CommitDelta can only emit a patch when it does; after DropBase (or a
// HELLO/view_rejected invalidation) the next commit is a full snapshot.
func (c *Client) HasBase() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.base != nil
}

// BaseRev returns the rev of the current baseline, or 0 when there is none.
func (c *Client) BaseRev() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.baseRev
}

// DropBase discards the committed baseline so the next CommitDelta writes a
// full VIEW. It is used when the host may no longer hold that revision
// (view_rejected, epoch reset); callers must never send a patch against a
// dropped base.
func (c *Client) DropBase() {
	c.mu.Lock()
	c.base = nil
	c.baseRev = 0
	c.mu.Unlock()
}

// Rev returns the last committed view revision.
func (c *Client) Rev() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.rev
}

// Pending returns the number of results still waiting for a RESPONSE.
func (c *Client) Pending() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.pending)
}

// Commit sends a full view snapshot: the epoch and rev are filled in and the
// revision advances. Views before the first HELLO are rejected. Commit takes
// writeMu before mu so the rev assigned here is also the write order, and it
// encodes from a pooled envelope (no per-frame View allocation).
func (c *Client) Commit(root *pb.Box, keys Keys) error {
	if root == nil {
		return ErrNilRoot
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.mu.Lock()
	if c.hello == nil {
		c.mu.Unlock()
		return ErrNoHello
	}
	c.rev++
	epoch, rev := c.epoch, c.rev
	c.mu.Unlock()

	err := c.encodeView(root, keys, epoch, rev)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.base = root
	c.baseRev = rev
	c.mu.Unlock()
	return nil
}

// encodeView writes one full VIEW frame from the pooled envelope. The caller
// assigns epoch/rev (Commit advances rev first so write order equals revision
// order).
func (c *Client) encodeView(root *pb.Box, keys Keys, epoch, rev uint64) error {
	scratch := viewScratchPool.Get().(*viewScratch)
	scratch.keys.Claim = append(scratch.keys.Claim[:0], keys.Claim...)
	scratch.keys.All = keys.All
	scratch.view.Epoch = epoch
	scratch.view.Rev = rev
	scratch.view.Keys = &scratch.keys
	scratch.view.Root = root
	err := c.enc.Encode(wire.TypeView, &scratch.view)
	scratch.view.Root = nil
	scratch.view.Keys = nil
	scratch.keys.All = false
	scratch.keys.Claim = scratch.keys.Claim[:0]
	viewScratchPool.Put(scratch)
	return err
}

// Emit sends one method call with a fresh request_id. When onResponse is
// non-nil it is invoked exactly once with the matching RESPONSE; the generic
// Handlers.Response still sees every response. The RESULT envelope is pooled.
func (c *Client) Emit(method string, params *pb.MethodParams, onResponse func(*pb.Response)) (uint64, error) {
	c.mu.Lock()
	if c.hello == nil {
		c.mu.Unlock()
		return 0, ErrNoHello
	}
	c.requestID++
	id := c.requestID
	if onResponse != nil {
		c.pending[id] = onResponse
	}
	epoch := c.epoch
	c.mu.Unlock()

	c.writeMu.Lock()
	result := resultScratchPool.Get().(*pb.Result)
	result.RequestId = id
	result.Epoch = epoch
	result.Method = method
	result.Params = params
	err := c.enc.Encode(wire.TypeResult, result)
	result.Method = ""
	result.Params = nil
	resultScratchPool.Put(result)
	c.writeMu.Unlock()
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return id, err
	}
	return id, nil
}

func (c *Client) dispatchHello(h *pb.Hello) {
	c.mu.Lock()
	c.hello = h
	c.epoch = h.GetEpoch()
	c.rev = 0
	c.pending = map[uint64]func(*pb.Response){}
	// A new HELLO never carries a view baseline (and a new epoch resets the
	// host cache, PROTOCOL §0.5): force the next commit to be a full VIEW.
	c.base = nil
	c.baseRev = 0
	handler := c.handlers.Hello
	c.mu.Unlock()
	if handler != nil {
		handler(h)
	}
}

func (c *Client) dispatchEvent(ev *pb.Event) {
	if ev == nil {
		return
	}
	c.mu.Lock()
	handlers := c.handlers
	rejected := ev.GetViewRejected()
	if rejected != nil {
		// The host rejected the last committed revision (base_mismatch,
		// path_invalid, oversize, max_nodes): its cache no longer matches our
		// baseline, so drop it before the program sees the event. PROTOCOL
		// §2.1 requires the next frame to be a full VIEW.
		c.base = nil
		c.baseRev = 0
	}
	c.mu.Unlock()
	switch {
	case ev.GetKey() != nil && handlers.Key != nil:
		handlers.Key(ev.GetKey())
	case ev.GetPaste() != nil && handlers.Paste != nil:
		handlers.Paste(ev.GetPaste())
	case ev.GetMouse() != nil && handlers.Mouse != nil:
		handlers.Mouse(ev.GetMouse())
	case ev.GetWheel() != nil && handlers.Wheel != nil:
		handlers.Wheel(ev.GetWheel())
	case ev.GetResize() != nil && handlers.Resize != nil:
		handlers.Resize(int(ev.GetResize().GetCols()), int(ev.GetResize().GetRows()))
	case ev.GetSources() != nil && handlers.Sources != nil:
		handlers.Sources(ev.GetSources().GetItems())
	case ev.GetNotice() != nil && handlers.Notice != nil:
		handlers.Notice(ev.GetNotice().GetLevel(), ev.GetNotice().GetMessage())
	case ev.GetComponent() != nil && handlers.Component != nil:
		handlers.Component(ev.GetComponent())
	case ev.GetViewRejected() != nil && handlers.ViewRejected != nil:
		rejected := ev.GetViewRejected()
		handlers.ViewRejected(rejected.GetEpoch(), rejected.GetRev(), rejected.GetReason())
	}
}

func (c *Client) dispatchResponse(resp *pb.Response) {
	if resp == nil {
		return
	}
	c.mu.Lock()
	// request_id is scoped to (epoch, connection) (PROTOCOL §0.5): a response
	// from an older epoch must never fire the callback of a new-epoch request
	// that happens to reuse the id.
	callback := c.pending[resp.GetRequestId()]
	if resp.GetEpoch() != c.epoch {
		callback = nil
	}
	delete(c.pending, resp.GetRequestId())
	generic := c.handlers.Response
	c.mu.Unlock()
	if callback != nil {
		callback(resp)
	}
	if generic != nil {
		generic(resp)
	}
}

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
}

// Client is one layout-program connection: it reads HELLO/EVENT/RESPONSE
// frames and writes VIEW/RESULT frames (PROTOCOL §0). It is safe for
// concurrent Emit/Commit calls from handler goroutines; Loop is the only
// reader and should run once.
type Client struct {
	enc *wire.Encoder
	dec *wire.Decoder

	writeMu sync.Mutex

	mu        sync.Mutex
	handlers  Handlers
	hello     *pb.Hello
	epoch     uint64
	rev       uint64
	requestID uint64
	pending   map[uint64]func(*pb.Response)
}

// New returns a Client reading host frames from r and writing program frames
// to w.
func New(r io.Reader, w io.Writer, handlers Handlers) *Client {
	return &Client{
		enc:      wire.NewEncoder(w, wire.RoleProgram, wire.DefaultMaxMessageBytes),
		dec:      wire.NewDecoder(r, wire.RoleProgram, wire.DefaultMaxMessageBytes),
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
		}
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
// revision advances. Views before the first HELLO are rejected.
func (c *Client) Commit(root *pb.Box, keys Keys) error {
	if root == nil {
		return ErrNilRoot
	}
	c.mu.Lock()
	if c.hello == nil {
		c.mu.Unlock()
		return ErrNoHello
	}
	c.rev++
	view := &pb.View{
		Epoch: c.epoch,
		Rev:   c.rev,
		Keys:  &pb.Keys{Claim: append([]string(nil), keys.Claim...), All: keys.All},
		Root:  root,
	}
	c.mu.Unlock()
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.enc.Encode(wire.TypeView, view)
}

// Emit sends one method call with a fresh request_id. When onResponse is
// non-nil it is invoked exactly once with the matching RESPONSE; the generic
// Handlers.Response still sees every response.
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

	result := &pb.Result{RequestId: id, Epoch: epoch, Method: method, Params: params}
	c.writeMu.Lock()
	err := c.enc.Encode(wire.TypeResult, result)
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

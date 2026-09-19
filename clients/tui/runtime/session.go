package runtime

import (
	"errors"
	"fmt"
	"io"
	"sync"

	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"

	"github.com/anytty/anytty/clients/tui/kernel"
	"github.com/anytty/anytty/clients/tui/render"
	"github.com/anytty/anytty/clients/tui/runtime/keys"
)

// Options configures a Session.
type Options struct {
	// ViewID is the host-assigned connection identifier (HELLO.view_id).
	ViewID string
	// Epoch is the first epoch; 0 means 1.
	Epoch uint64
	// Schema is the protocol schema version; 0 means 1.
	Schema uint32
	// Cols and Rows are the viewport used to solve VIEW trees.
	Cols, Rows int
	// Components is advertised in HELLO.components.
	Components []string
	// Limits default to DefaultLimits when all zero.
	Limits Limits
	// Handler executes RESULT methods; nil means a StubHandler.
	Handler Handler
	// MouseTracking reports whether a terminal source has mouse tracking on
	// (routing input §6.5). Nil means never.
	MouseTracking func(sourceID string) bool
	// InputSink receives host-encoded bytes for DestinationPTY inputs and
	// serves the terminal bracket-paste mode; nil means PTY input fails.
	InputSink InputSink
	// EventSink overrides host -> program event delivery; nil writes EVENT
	// frames through the session encoder.
	EventSink EventSink
}

// Limits mirrors hello.limits (PROTOCOL §1). It is a plain value type so it
// can be copied and compared freely, unlike the generated protobuf message.
type Limits struct {
	MaxNodes            uint32
	MaxMessageBytes     uint32
	MaxPasteBytes       uint32
	MaxInflightRequests uint32
	OwnerLeaseTtlMs     uint32
}

// DefaultLimits returns the PROTOCOL §1 example limits.
func DefaultLimits() Limits {
	return Limits{
		MaxNodes:            4096,
		MaxMessageBytes:     wire.DefaultMaxMessageBytes,
		MaxPasteBytes:       65536,
		MaxInflightRequests: 64,
		OwnerLeaseTtlMs:     15000,
	}
}

func (l Limits) proto() *pb.Limits {
	return &pb.Limits{
		MaxNodes:            l.MaxNodes,
		MaxMessageBytes:     l.MaxMessageBytes,
		MaxPasteBytes:       l.MaxPasteBytes,
		MaxInflightRequests: l.MaxInflightRequests,
		OwnerLeaseTtlMs:     l.OwnerLeaseTtlMs,
	}
}

type inflightRequest struct {
	epoch uint64
}

type rejectedKey struct {
	epoch uint64
	rev   uint64
}

// Session is the host side of one view connection. It owns epoch/view_id
// state, accepted view revisions, claim/focus/capture state, the sources
// snapshot, in-flight RESULT bookkeeping and frame composition.
type Session struct {
	mu sync.Mutex

	viewID       string
	schema       uint32
	epoch        uint64
	limits       Limits
	cols, rows   int
	components   []string
	handler      Handler
	mouseTracker func(string) bool
	inputSink    InputSink
	eventSink    EventSink
	history      *keys.History
	eventSeq     uint64
	screenRev    uint64

	dec *wire.Decoder
	enc *wire.Encoder

	rev       uint64
	view      *pb.View
	root      *kernel.Node
	frame     kernel.Frame
	haveFrame bool
	claim     []string
	keysAll   bool
	focus     *Focus

	sources []*pb.Source

	coreOverlay *kernel.Frame
	captureNode string
	compositor  *Compositor
	lastFrame   *render.Frame

	inflight map[uint64]inflightRequest
	answered map[uint64]bool
	rejected map[rejectedKey]bool
}

// NewSession wires one runtime session to the program pipes: r carries
// program -> host frames (VIEW/RESULT) and w carries host -> program frames
// (HELLO/EVENT/RESPONSE).
func NewSession(opts Options, r io.Reader, w io.Writer) *Session {
	limits := opts.Limits
	if limits.MaxNodes == 0 && limits.MaxMessageBytes == 0 && limits.MaxPasteBytes == 0 &&
		limits.MaxInflightRequests == 0 && limits.OwnerLeaseTtlMs == 0 {
		limits = DefaultLimits()
	}
	epoch := opts.Epoch
	if epoch == 0 {
		epoch = 1
	}
	schema := opts.Schema
	if schema == 0 {
		schema = 1
	}
	handler := opts.Handler
	if handler == nil {
		handler = &StubHandler{}
	}
	return &Session{
		viewID:       opts.ViewID,
		schema:       schema,
		epoch:        epoch,
		limits:       limits,
		cols:         opts.Cols,
		rows:         opts.Rows,
		components:   append([]string(nil), opts.Components...),
		handler:      handler,
		mouseTracker: opts.MouseTracking,
		inputSink:    opts.InputSink,
		eventSink:    opts.EventSink,
		history:      keys.NewHistory(keys.DefaultLimit),
		compositor:   NewCompositor(opts.Cols, opts.Rows),
		dec:          wire.NewDecoder(r, wire.RoleHost, limits.MaxMessageBytes),
		enc:          wire.NewEncoder(w, wire.RoleHost, limits.MaxMessageBytes),
		inflight:     map[uint64]inflightRequest{},
		answered:     map[uint64]bool{},
		rejected:     map[rejectedKey]bool{},
	}
}

// SetTheme is gone: the runtime holds no palette. Programs send explicit
// styles; only builtin chrome resolves the host-internal default palette.

// ViewID returns the connection view id.
func (s *Session) ViewID() string { s.mu.Lock(); defer s.mu.Unlock(); return s.viewID }

// Epoch returns the current epoch.
func (s *Session) Epoch() uint64 { s.mu.Lock(); defer s.mu.Unlock(); return s.epoch }

// Schema returns the advertised protocol schema.
func (s *Session) Schema() uint32 { s.mu.Lock(); defer s.mu.Unlock(); return s.schema }

// Limits returns the advertised limits.
func (s *Session) Limits() Limits { s.mu.Lock(); defer s.mu.Unlock(); return s.limits }

// Rev returns the last accepted view revision (0 before the first view).
func (s *Session) Rev() uint64 { s.mu.Lock(); defer s.mu.Unlock(); return s.rev }

// Hello builds the current HELLO message.
func (s *Session) Hello() *pb.Hello {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.helloLocked()
}

// SendHello writes the HELLO frame for the current epoch.
func (s *Session) SendHello() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enc.Encode(wire.TypeHello, s.helloLocked())
}

// ReadFrame reads one program frame envelope. Oversize frames are returned
// as *wire.Error{Kind: KindOversize} with the frame drained from the stream;
// pass them to HandleOversize and continue.
func (s *Session) ReadFrame() (wire.Type, []byte, error) {
	return s.dec.Decode()
}

// HandleFrame dispatches one frame envelope. Only VIEW and RESULT are legal
// program -> host frames.
func (s *Session) HandleFrame(t wire.Type, payload []byte) error {
	if !wire.RoleHost.CanReceive(t) {
		return &wire.Error{Kind: wire.KindDirection, Type: t}
	}
	m, err := wire.UnmarshalPayload(t, payload)
	if err != nil {
		return err
	}
	switch t {
	case wire.TypeView:
		return s.HandleView(m.(*pb.View))
	case wire.TypeResult:
		return s.HandleResult(m.(*pb.Result))
	default:
		return &wire.Error{Kind: wire.KindDirection, Type: t}
	}
}

// Serve reads and handles program frames until a clean EOF. Envelope errors
// are returned so the caller can disconnect and restart the program; an
// oversize frame is answered (view_rejected / oversize) and the loop
// continues, because a rejected frame must not kill the program.
func (s *Session) Serve() error {
	for {
		t, payload, err := s.dec.Decode()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			var fe *wire.Error
			if errors.As(err, &fe) && fe.Kind == wire.KindOversize {
				if err := s.HandleOversize(fe.Type); err != nil {
					return err
				}
				continue
			}
			return err
		}
		if err := s.HandleFrame(t, payload); err != nil {
			return err
		}
	}
}

// HandleOversize answers a frame that the size limit rejected before its
// payload could be decoded. The rev/request_id inside the payload are
// unknown, so VIEW is answered with view_rejected{rev:0} (deduplicated per
// epoch) and RESULT with RESPONSE{request_id:0, error:"oversize"}.
func (s *Session) HandleOversize(t wire.Type) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch t {
	case wire.TypeView:
		return s.rejectViewLocked(0, "oversize")
	case wire.TypeResult:
		return s.sendResponseLocked(s.epoch, 0, false, nil, "oversize")
	default:
		return &wire.Error{Kind: wire.KindDirection, Type: t}
	}
}

// HandleView applies one VIEW frame: epoch and rev are checked first, then
// the max_nodes limit (answer view_rejected once per (epoch, rev)), then the
// box tree replaces the cached state atomically together with its claim.
// Stale epochs and non-monotonic revs are dropped silently.
func (s *Session) HandleView(v *pb.View) error {
	if v == nil {
		return errors.New("runtime: nil view")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if v.Epoch != s.epoch {
		return nil
	}
	if v.Rev <= s.rev {
		return nil
	}
	if max := s.limits.MaxNodes; max > 0 && uint32(countBoxes(v.Root)) > max {
		return s.rejectViewLocked(v.Rev, "max_nodes")
	}
	root := toNode(v.Root)
	s.rev = v.Rev
	s.view = v
	s.root = root
	s.frame = kernel.Layout(root, s.cols, s.rows)
	s.haveFrame = true
	if v.Keys != nil {
		s.claim = append([]string(nil), v.Keys.Claim...)
		s.keysAll = v.Keys.All
	} else {
		s.claim = nil
		s.keysAll = false
	}
	s.focus = s.focusLocked(root)
	if s.captureNode != "" {
		if _, ok := s.frame.Rect(s.captureNode); !ok {
			s.captureNode = ""
		}
	}
	return nil
}

// HandleResult processes one RESULT: stale epochs are answered "epoch
// reset", the method must be in the §4 registry with valid params, in-flight
// calls over max_inflight_requests are answered "throttled" (never queued),
// and every accepted call gets exactly one RESPONSE carrying its epoch and
// request_id.
func (s *Session) HandleResult(r *pb.Result) error {
	if r == nil {
		return errors.New("runtime: nil result")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.Epoch != s.epoch {
		return s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, "epoch reset")
	}
	if s.answered[r.RequestId] {
		return s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, "duplicate request_id")
	}
	method, ok := LookupMethod(r.Method)
	if !ok {
		return s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, "unknown method: "+r.Method)
	}
	if msg := validateParams(method, r.Params); msg != "" {
		return s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, msg)
	}
	if max := s.limits.MaxInflightRequests; max > 0 && uint32(len(s.inflight)) >= max {
		return s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, "throttled")
	}
	s.answered[r.RequestId] = true
	if method.Name == "input.forward" {
		return s.forwardLocked(r)
	}
	outcome, pending := s.handler.Handle(Request{
		Epoch:     r.Epoch,
		RequestID: r.RequestId,
		Method:    method,
		Params:    r.Params,
	})
	if pending {
		s.inflight[r.RequestId] = inflightRequest{epoch: r.Epoch}
		return nil
	}
	if !outcome.OK && outcome.Error == "" {
		outcome.Error = "rejected"
	}
	return s.sendResponseLocked(r.Epoch, r.RequestId, outcome.OK, outcome.Data, outcome.Error)
}

// Complete delivers the RESPONSE of an accepted RESULT whose Handler
// returned pending=true. It is an error to complete a request that is not in
// flight in the current epoch.
func (s *Session) Complete(requestID uint64, data *pb.MethodData, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.inflight[requestID]
	if !ok {
		return fmt.Errorf("runtime: request %d is not in flight", requestID)
	}
	delete(s.inflight, requestID)
	if req.epoch != s.epoch {
		return fmt.Errorf("runtime: request %d belongs to epoch %d", requestID, req.epoch)
	}
	return s.sendResponseLocked(req.epoch, requestID, errMsg == "", data, errMsg)
}

// Pending returns the number of in-flight RESULT calls.
func (s *Session) Pending() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.inflight)
}

// Reset moves the session to the next epoch: every in-flight call of the
// previous epoch is answered "epoch reset" best effort, then the view cache,
// claim, focus, drag capture and core overlay are cleared. The rev counter
// restarts, so the first VIEW of the new epoch always applies. The caller
// should SendHello afterwards.
func (s *Session) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, req := range s.inflight {
		_ = s.sendResponseLocked(req.epoch, id, false, nil, "epoch reset")
	}
	s.inflight = map[uint64]inflightRequest{}
	s.answered = map[uint64]bool{}
	s.rejected = map[rejectedKey]bool{}
	s.history = keys.NewHistory(keys.DefaultLimit)
	s.eventSeq = 0
	s.epoch++
	s.rev = 0
	s.view = nil
	s.root = nil
	s.frame = kernel.Frame{}
	s.haveFrame = false
	s.claim = nil
	s.keysAll = false
	s.focus = nil
	s.captureNode = ""
	s.coreOverlay = nil
	s.lastFrame = nil
	return nil
}

// SetSources publishes the full sources snapshot (PROTOCOL §5, §9.2): every
// item carries every field on every push; the authoritative state is the
// host's. The snapshot also refreshes the focused source's terminal
// capability.
func (s *Session) SetSources(items []*pb.Source) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*pb.Source, 0, len(items))
	for _, it := range items {
		out = append(out, cloneSource(it))
	}
	s.sources = out
	if s.root != nil {
		s.focus = s.focusLocked(s.root)
	}
	return s.sendEventLocked(&pb.Event{Event: &pb.Event_Sources{Sources: &pb.SourcesEvent{Items: out}}})
}

// SendNotice pushes one transient host notice event (PROTOCOL §3). Level is
// "info", "warning" or "error"; an empty message is ignored. Notices are
// lossy by design: they inform, they never carry state.
func (s *Session) SendNotice(level, message string) error {
	if message == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sendEventLocked(&pb.Event{Event: &pb.Event_Notice{Notice: &pb.NoticeEvent{
		Level:   level,
		Message: message,
	}}})
}

// Sources returns a copy of the last published sources snapshot.
func (s *Session) Sources() []*pb.Source {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*pb.Source, 0, len(s.sources))
	for _, it := range s.sources {
		out = append(out, cloneSource(it))
	}
	return out
}

// View returns the last accepted VIEW, or nil.
func (s *Session) View() *pb.View {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.view
}

// Frame returns the last solved frame.
func (s *Session) Frame() (kernel.Frame, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.frame, s.haveFrame
}

// BoxProps returns the program-declared properties of the box with the given
// node id (content.props), or nil. The runtime never interprets them: they
// are the program -> component attribute channel and are passed through to
// the component factory (PROTOCOL §2, §5).
func (s *Session) BoxProps(nodeID string) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	node := findNode(s.root, nodeID)
	if node == nil || node.Content == nil {
		return nil
	}
	return cloneProps(node.Content.Props)
}

// Resize changes the viewport: the last accepted view is re-solved, the
// compositor is resized, the diff baseline is dropped so the next frame is a
// full repaint, and the program is told with a resize event so it can reflow
// its boxes (PROTOCOL §3: geometry is program policy, the host only says how
// much room there is). An unchanged viewport is a no-op, so repeated
// SIGWINCHes do not spam the program.
func (s *Session) Resize(cols, rows int) {
	if cols <= 0 || rows <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if cols == s.cols && rows == s.rows {
		return
	}
	s.cols, s.rows = cols, rows
	if s.compositor != nil {
		s.compositor.SetSize(s.cols, s.rows)
	}
	if s.root != nil {
		s.frame = kernel.Layout(s.root, s.cols, s.rows)
		s.haveFrame = true
	}
	s.lastFrame = nil
	_ = s.sendEventLocked(&pb.Event{Event: &pb.Event_Resize{Resize: &pb.ResizeEvent{
		Cols: uint32(s.cols),
		Rows: uint32(s.rows),
	}}})
}

// ComposeFrame renders the last accepted view, the given component
// placements, a host notice and the core overlay to one framebuffer.
func (s *Session) ComposeFrame(placements []Placement, notice *kernel.Frame) *render.Frame {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.composeFrameLocked(placements, notice)
}

// FrameBytes composes the current frame and returns the minimal ANSI update
// against the previous composition of this session. When nothing changed it
// returns nil, so no duplicate bytes reach the TTY.
func (s *Session) FrameBytes(placements []Placement, notice *kernel.Frame) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	frame := s.composeFrameLocked(placements, notice)
	prev := s.lastFrame
	s.lastFrame = frame
	return frame.Bytes(prev)
}

func (s *Session) composeFrameLocked(placements []Placement, notice *kernel.Frame) *render.Frame {
	if s.compositor == nil {
		s.compositor = NewCompositor(s.cols, s.rows)
	}
	return s.compositor.Compose(s.frame, placements, notice, s.coreOverlay)
}

// Claim returns the claim of the last accepted view.
func (s *Session) Claim() ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.claim...), s.keysAll
}

// Focus returns the focused content source of the last accepted view.
func (s *Session) Focus() *Focus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.focus
}

// Route applies the §6.5 routing table to the current session state, with
// the focused source's terminal capabilities refreshed from the host
// registry before the mouse/wheel conditions are evaluated.
func (s *Session) Route(ev InputEvent) Destination {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshFocusLocked()
	return Route(ev, RouteState{
		CoreOverlay: s.coreOverlay != nil,
		KeysAll:     s.keysAll,
		Claim:       s.claim,
		Focus:       s.focus,
	})
}

// refreshFocusLocked refreshes the mouse-tracking bit of the cached focus,
// because the program can toggle DEC mouse modes after the view was solved.
// The focus is replaced with a copy, never mutated, so pointers handed out
// by Focus() stay race-free.
func (s *Session) refreshFocusLocked() {
	if s.mouseTracker == nil || s.focus == nil {
		return
	}
	tracked := s.mouseTracker(s.focus.ID)
	if tracked == s.focus.MouseTracking {
		return
	}
	focus := *s.focus
	focus.MouseTracking = tracked
	s.focus = &focus
}

// OpenCoreOverlay installs the host-drawn core overlay frame. While it is
// open the host owns every input (§6.5 priority 2) and it composites last
// (§9.1).
func (s *Session) OpenCoreOverlay(frame kernel.Frame) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := frame
	s.coreOverlay = &f
}

// CloseCoreOverlay removes the core overlay.
func (s *Session) CloseCoreOverlay() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.coreOverlay = nil
}

// CoreOverlayOpen reports whether a core overlay is installed.
func (s *Session) CoreOverlayOpen() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.coreOverlay != nil
}

// Capture starts the implicit drag capture on a non-terminal box
// (PROTOCOL §6.7).
func (s *Session) Capture(nodeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.captureNode = nodeID
}

// ReleaseCapture ends the drag capture.
func (s *Session) ReleaseCapture() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.captureNode = ""
}

// CapturedNode returns the node id owning the current drag capture, or "".
func (s *Session) CapturedNode() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.captureNode
}

func (s *Session) helloLocked() *pb.Hello {
	return &pb.Hello{
		Schema:     s.schema,
		ViewId:     s.viewID,
		Epoch:      s.epoch,
		Cols:       uint32(s.cols),
		Rows:       uint32(s.rows),
		Components: append([]string(nil), s.components...),
		Events:     eventNames(),
		Methods:    MethodNames(),
		Features: map[string]bool{
			"component":  true,
			"state.save": false,
			"state.load": false,
		},
		Limits: s.limits.proto(),
	}
}

func (s *Session) sendEventLocked(ev *pb.Event) error {
	if s.eventSink != nil {
		return s.eventSink.SendEvent(ev)
	}
	return s.enc.Encode(wire.TypeEvent, ev)
}

func (s *Session) sendResponseLocked(epoch, requestID uint64, ok bool, data *pb.MethodData, errMsg string) error {
	if ok {
		errMsg = ""
	} else if errMsg == "" {
		errMsg = "rejected"
	}
	return s.enc.Encode(wire.TypeResponse, &pb.Response{
		RequestId: requestID,
		Epoch:     epoch,
		Ok:        ok,
		Data:      data,
		Error:     errMsg,
	})
}

func (s *Session) rejectViewLocked(rev uint64, reason string) error {
	key := rejectedKey{epoch: s.epoch, rev: rev}
	if s.rejected[key] {
		return nil
	}
	s.rejected[key] = true
	return s.sendEventLocked(&pb.Event{Event: &pb.Event_ViewRejected{
		ViewRejected: &pb.ViewRejectedEvent{Epoch: s.epoch, Rev: rev, Reason: reason},
	}})
}

// focusLocked finds the first focused box with a content source reference
// and annotates it with host-side capability (terminal kind, mouse
// tracking).
func (s *Session) focusLocked(root *kernel.Node) *Focus {
	var found *kernel.Node
	var walk func(*kernel.Node)
	walk = func(n *kernel.Node) {
		if n == nil || found != nil {
			return
		}
		if n.Focused && n.Content != nil && n.Content.Self != "" {
			found = n
			return
		}
		for i := range n.Children {
			walk(&n.Children[i])
			if found != nil {
				return
			}
		}
	}
	walk(root)
	if found == nil {
		return nil
	}
	f := &Focus{ID: found.Content.Self, Input: append([]string(nil), found.Input...)}
	if src := s.sourceLocked(f.ID); src != nil {
		f.IsTerminal = src.GetKind() == "terminal"
	}
	if s.mouseTracker != nil {
		f.MouseTracking = s.mouseTracker(f.ID)
	}
	return f
}

func (s *Session) sourceLocked(id string) *pb.Source {
	for _, src := range s.sources {
		if src.GetId() == id {
			return src
		}
	}
	return nil
}

func cloneSource(src *pb.Source) *pb.Source {
	if src == nil {
		return nil
	}
	return &pb.Source{
		Id:          src.GetId(),
		Kind:        src.GetKind(),
		Title:       src.GetTitle(),
		Endpoint:    src.GetEndpoint(),
		TerminalId:  src.GetTerminalId(),
		Attached:    src.GetAttached(),
		Exited:      src.GetExited(),
		ExitCode:    src.GetExitCode(),
		Health:      src.GetHealth(),
		ResizeOwner: src.GetResizeOwner(),
		OwnerEpoch:  src.GetOwnerEpoch(),
		LastSeenMs:  src.GetLastSeenMs(),
	}
}

func countBoxes(b *pb.Box) int {
	if b == nil {
		return 0
	}
	n := 1
	for _, c := range b.Children {
		n += countBoxes(c)
	}
	return n
}

// findNode looks up a node by id in a solved kernel tree.
func findNode(n *kernel.Node, id string) *kernel.Node {
	if n == nil || id == "" {
		return nil
	}
	if n.ID == id {
		return n
	}
	for i := range n.Children {
		if found := findNode(&n.Children[i], id); found != nil {
			return found
		}
	}
	return nil
}

// cloneProps returns an independent copy of a props map (nil stays nil), so
// components can never mutate runtime state through the shared reference.
func cloneProps(props map[string]string) map[string]string {
	if len(props) == 0 {
		return nil
	}
	out := make(map[string]string, len(props))
	for key, value := range props {
		out[key] = value
	}
	return out
}

func toNode(b *pb.Box) *kernel.Node {
	if b == nil {
		return nil
	}
	n := &kernel.Node{
		ID:      b.GetId(),
		Flow:    kernel.Flow(b.GetFlow()),
		Style:   b.GetStyle(),
		Input:   append([]string(nil), b.Input...),
		Focused: b.GetFocused(),
	}
	if b.Size != nil {
		n.Size = kernel.Size{
			Width:  int(b.Size.GetWidth()),
			Height: int(b.Size.GetHeight()),
			Flex:   int(b.Size.GetFlex()),
		}
	}
	if b.Pos != nil {
		n.Pos = &kernel.Pos{X: int(b.Pos.GetX()), Y: int(b.Pos.GetY())}
	}
	if b.Visible != nil {
		v := b.GetVisible()
		n.Visible = &v
	}
	if b.Content != nil {
		n.Content = &kernel.Content{
			Text:  b.Content.GetText(),
			Lines: append([]string(nil), b.Content.Lines...),
			Self:  b.Content.GetSelf(),
			Props: cloneProps(b.Content.GetProps()),
		}
	}
	if b.Cursor != nil {
		c := &kernel.Cursor{
			Row:   int(b.Cursor.GetRow()),
			Col:   int(b.Cursor.GetCol()),
			Shape: b.Cursor.GetShape(),
		}
		if b.Cursor.Visible != nil {
			v := b.Cursor.GetVisible()
			c.Visible = &v
		}
		n.Cursor = c
	}
	if len(b.Children) > 0 {
		n.Children = make([]kernel.Node, len(b.Children))
		for i, child := range b.Children {
			if c := toNode(child); c != nil {
				n.Children[i] = *c
			}
		}
	}
	return n
}

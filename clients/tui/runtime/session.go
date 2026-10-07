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
	gproto "google.golang.org/protobuf/proto"
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
	// HistoryActive reports whether one pane of a terminal source is currently
	// showing a frozen host-owned history/copy viewport. viewID is the
	// terminal box node id (== pane id); empty means the legacy single view.
	// Nil means never. This is kept separate from MouseTracking because a child
	// may leave DEC tracking on while the TUI owns the current scroll gesture.
	HistoryActive func(sourceID, viewID string) bool
	// InputSink receives host-encoded bytes for DestinationPTY inputs and
	// serves the terminal bracket-paste mode; nil means PTY input fails.
	InputSink InputSink
	// EventSink overrides host -> program event delivery; nil writes EVENT
	// frames through the session encoder.
	EventSink EventSink
	// OnStream handles program -> host STREAM frames (access stream data,
	// ack, close, cancel). Nil rejects them as a direction error.
	OnStream func(*pb.StreamFrame) error
	// OnView is called after an accepted VIEW/VIEW_DELTA is committed. Hosts
	// use it to wake their frame loop immediately instead of waiting for the
	// next tick; it must not block (a buffered signal is enough).
	OnView func()
	// OutputQueue enables a bounded asynchronous host -> program writer. A
	// value of zero preserves the synchronous writer used by small in-memory
	// tests. Production hosts set this so a slow layout program cannot hold
	// the frame loop in an io.Writer call; control frames still wait for space,
	// while notices/sources/stream data coalesce or drop when saturated.
	OutputQueue int
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

	viewID        string
	schema        uint32
	epoch         uint64
	limits        Limits
	cols, rows    int
	components    []string
	handler       Handler
	mouseTracker  func(string) bool
	historyActive func(string, string) bool
	inputSink     InputSink
	eventSink     EventSink
	onStream      func(*pb.StreamFrame) error
	onView        func()
	history       *keys.History
	eventSeq      uint64
	screenRev     uint64

	dec *wire.Decoder
	enc *wire.Encoder
	out *outboundWriter

	rev       uint64
	view      *pb.View
	root      *kernel.Node
	frame     kernel.Frame
	haveFrame bool
	claim     []string
	keysAll   bool
	focus     *Focus

	// nodes maps each program box pointer of the current view to the kernel
	// node built from it. Unchanged (copy-on-write shared) subtrees keep the
	// same *pb.Box pointer across a VIEW_DELTA, so this map is the reuse
	// anchor: a cache hit reuses the exact *kernel.Node instead of rebuilding
	// the subtree. The solved rect/frame for a node live in the node's own
	// kernel cache (kernel.Node.Cached); this map only carries the pointer, so
	// it adds no frame storage.
	//
	// The map is persistent across delta commits: a delta inserts only the new
	// boxes it creates and deletes only the pointers it detached (staleSet), so
	// both building and bookkeeping are O(changed). It is built lazily: a full
	// VIEW shares no pointers with anything, so it neither reads nor writes the
	// map (and pays nothing for it); the next VIEW_DELTA indexes the cached
	// tree once, then every delta maintains it in place. A hard cap
	// (nodeMapCapFactor x box count) falls back to a full indexNodes rebuild if
	// the map ever grows past the live tree (see enforceNodeMapCap).
	nodes map[*pb.Box]*kernel.Node

	// boxCount is the number of boxes in the committed view tree, maintained
	// incrementally by a delta (insert/replace add the patch box's count,
	// remove subtracts the dropped subtree's count) so HandleViewDelta's
	// max_nodes check needs no O(tree) walk. It is -1 when unknown (before the
	// first view, or after a full VIEW / Reset), in which case the check falls
	// back to countBoxes and refreshes the counter. See patchEffect.focusDirty
	// for how focus is kept O(changed) alongside it.
	boxCount int

	sources []*pb.Source

	coreOverlay *kernel.Frame
	captureNode string
	compositor  *Compositor
	lastFrame   *render.Frame
	frameBuffer *render.Frame

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
	s := &Session{
		viewID:        opts.ViewID,
		schema:        schema,
		epoch:         epoch,
		limits:        limits,
		cols:          opts.Cols,
		rows:          opts.Rows,
		components:    append([]string(nil), opts.Components...),
		handler:       handler,
		mouseTracker:  opts.MouseTracking,
		historyActive: opts.HistoryActive,
		inputSink:     opts.InputSink,
		eventSink:     opts.EventSink,
		onStream:      opts.OnStream,
		onView:        opts.OnView,
		history:       keys.NewHistory(keys.DefaultLimit),
		compositor:    NewCompositor(opts.Cols, opts.Rows),
		dec:           wire.NewDecoder(r, wire.RoleHost, limits.MaxMessageBytes),
		enc:           wire.NewEncoder(w, wire.RoleHost, limits.MaxMessageBytes),
		inflight:      map[uint64]inflightRequest{},
		answered:      map[uint64]bool{},
		rejected:      map[rejectedKey]bool{},
		boxCount:      -1,
	}
	if opts.OutputQueue > 0 {
		s.out = newOutboundWriter(w, limits.MaxMessageBytes, opts.OutputQueue)
	}
	return s
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
	return s.writeLocked(wire.TypeHello, s.helloLocked(), outputControl, "hello")
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
	case wire.TypeViewDelta:
		return s.HandleViewDelta(m.(*pb.ViewDelta))
	case wire.TypeResult:
		return s.HandleResult(m.(*pb.Result))
	case wire.TypeStream:
		if s.onStream == nil {
			return &wire.Error{Kind: wire.KindDirection, Type: t}
		}
		return s.onStream(m.(*pb.StreamFrame))
	default:
		return &wire.Error{Kind: wire.KindDirection, Type: t}
	}
}

// SendStream writes one STREAM frame to the program (host -> program). It is
// safe to call from any goroutine; slow programs backpressure the caller
// through the pipe instead of dropping frames.
func (s *Session) SendStream(frame *pb.StreamFrame) error {
	if frame == nil {
		return errors.New("runtime: nil stream frame")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.enc == nil {
		return errors.New("runtime: session is closed")
	}
	class := outputCoalescible
	key := "stream:data"
	if frame.GetKind() != "data" {
		class = outputControl
		key = "stream:control"
	}
	return s.writeLocked(wire.TypeStream, frame, class, key)
}

// Close stops the optional asynchronous writer. It is idempotent and should
// be called by a host before closing the layout program pipes.
func (s *Session) Close() {
	s.mu.Lock()
	out := s.out
	s.out = nil
	s.mu.Unlock()
	if out != nil {
		out.close()
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
// unknown, so VIEW and VIEW_DELTA are answered with view_rejected{rev:0}
// (deduplicated per epoch) and RESULT with RESPONSE{request_id:0,
// error:"oversize"}.
func (s *Session) HandleOversize(t wire.Type) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch t {
	case wire.TypeView, wire.TypeViewDelta:
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
	if v.Epoch != s.epoch {
		s.mu.Unlock()
		return nil
	}
	if v.Rev <= s.rev {
		s.mu.Unlock()
		return nil
	}
	if max := s.limits.MaxNodes; max > 0 && uint32(countBoxes(v.Root)) > max {
		err := s.rejectViewLocked(v.Rev, "max_nodes")
		s.mu.Unlock()
		return err
	}
	s.commitViewLocked(v.Rev, v, v.Root, v.Keys, true, false, staleSet{}, false, -1)
	onView := s.onView
	s.mu.Unlock()
	if onView != nil {
		onView()
	}
	return nil
}

// commitViewLocked atomically installs one accepted revision: rev/view cache,
// the solved layout, focus, drag capture and the claim. applyKeys reports
// whether the frame carried a keys decision: a full VIEW always does (a nil
// Keys resets the claim), while VIEW_DELTA with keys omitted keeps the
// previous claim (§2.1). incremental reports whether the new tree may share
// subtrees with the previous one (VIEW_DELTA) and can therefore reuse the
// node cache; a full VIEW always builds fresh.
//
// stale carries the pointers an incremental commit detached (for cache
// pruning); focusDirty reports whether a patch could have changed the focused
// node (when false the committed focus is reused as-is); boxCount is the
// already-known box count of the new tree, or -1 to signal "unknown, derive
// it".
func (s *Session) commitViewLocked(rev uint64, view *pb.View, rootBox *pb.Box, keys *pb.Keys, applyKeys, incremental bool, stale staleSet, focusDirty bool, boxCount int) {
	root, reused, inexactPrune := s.buildNodeTree(rootBox, incremental, stale)
	s.rev = rev
	s.view = view
	s.root = root
	if reused {
		s.frame = kernel.LayoutCached(root, s.cols, s.rows)
	} else {
		s.frame = kernel.Layout(root, s.cols, s.rows)
	}
	s.haveFrame = true
	if incremental && !focusDirty {
		// No patch could have changed which node is focused or its Input, and
		// Focus{ID, Input} is a value, so the previous focus is still exact.
	} else {
		s.focus = s.focusLocked(root)
	}
	if applyKeys {
		if keys != nil {
			s.claim = append([]string(nil), keys.Claim...)
			s.keysAll = keys.All
		} else {
			s.claim = nil
			s.keysAll = false
		}
	}
	switch {
	case boxCount >= 0:
		s.boxCount = boxCount
	case incremental:
		// The count was not pre-computed (non-growing delta with an unknown
		// base count): derive it once from the committed tree.
		s.boxCount = countBoxes(rootBox)
	default:
		s.boxCount = -1
	}
	if s.captureNode != "" {
		if _, ok := s.frame.Rect(s.captureNode); !ok {
			s.captureNode = ""
		}
	}
	if inexactPrune {
		// s.view/s.root now point at the committed tree, so this reindexes the
		// new tree (the pre-commit attempt would have re-indexed the old one).
		s.indexNodes()
	}
	s.enforceNodeMapCap()
}

// HandleResult processes one RESULT: stale epochs are answered "epoch
// reset", the method must be in the §4 registry with valid params, in-flight
// calls over max_inflight_requests are answered "throttled" (never queued),
// and every accepted call gets exactly one RESPONSE carrying its epoch and
// request_id.
// HandleResult executes one program RESULT (PROTOCOL §4). Validation and the
// response bookkeeping run under the session lock, but the handler itself is
// called without it: a slow handler (an endpoint dial, a daemon restart)
// must never freeze input routing or view handling for the other panes.
// Program calls are serialized by the single frame reader, so handler
// execution stays sequential.
func (s *Session) HandleResult(r *pb.Result) error {
	if r == nil {
		return errors.New("runtime: nil result")
	}
	s.mu.Lock()
	if r.Epoch != s.epoch {
		err := s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, "epoch reset")
		s.mu.Unlock()
		return err
	}
	if s.answered[r.RequestId] {
		err := s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, "duplicate request_id")
		s.mu.Unlock()
		return err
	}
	method, ok := LookupMethod(r.Method)
	if !ok {
		err := s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, "unknown method: "+r.Method)
		s.mu.Unlock()
		return err
	}
	if msg := validateParams(method, r.Params); msg != "" {
		err := s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, msg)
		s.mu.Unlock()
		return err
	}
	if max := s.limits.MaxInflightRequests; max > 0 && uint32(len(s.inflight)) >= max {
		err := s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, "throttled")
		s.mu.Unlock()
		return err
	}
	s.answered[r.RequestId] = true
	if method.Name == "input.forward" {
		err := s.forwardLocked(r)
		s.mu.Unlock()
		return err
	}
	handler := s.handler
	// Reserve before invoking an asynchronous handler. Its worker may finish
	// before Handle returns pending=true.
	s.inflight[r.RequestId] = inflightRequest{epoch: r.Epoch}
	s.mu.Unlock()

	outcome, pending := handler.Handle(Request{
		Epoch:     r.Epoch,
		RequestID: r.RequestId,
		OwnerID:   s.viewID,
		Method:    method,
		Params:    r.Params,
	})

	s.mu.Lock()
	defer s.mu.Unlock()
	if r.Epoch != s.epoch {
		// The epoch advanced while the handler ran: Reset already answered
		// every in-flight call of the old epoch, so drop this completion.
		return nil
	}
	if pending {
		return nil
	}
	delete(s.inflight, r.RequestId)
	if !outcome.OK && outcome.Error == "" {
		outcome.Error = "rejected"
	}
	return s.sendResponseLocked(r.Epoch, r.RequestId, outcome.OK, outcome.Data, outcome.Error)
}

// Complete delivers the RESPONSE of an accepted RESULT whose Handler
// returned pending=true. It is an error to complete a request that is not in
// flight in the current epoch.
func (s *Session) Complete(requestID uint64, data *pb.MethodData, errMsg string) error {
	return s.CompleteForEpoch(0, requestID, data, errMsg)
}

// CompleteForEpoch fences a worker's completion against program restarts.
// Unlike request IDs, epochs are never reused by a Session.
func (s *Session) CompleteForEpoch(epoch, requestID uint64, data *pb.MethodData, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.inflight[requestID]
	if !ok {
		return fmt.Errorf("runtime: request %d is not in flight", requestID)
	}
	if epoch != 0 && req.epoch != epoch {
		return fmt.Errorf("runtime: request %d belongs to epoch %d", requestID, req.epoch)
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
	s.nodes = nil
	s.boxCount = -1
	s.frame = kernel.Frame{}
	s.haveFrame = false
	s.claim = nil
	s.keysAll = false
	s.focus = nil
	s.captureNode = ""
	s.coreOverlay = nil
	s.lastFrame = nil
	s.frameBuffer = nil
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
		s.frame = kernel.LayoutCached(s.root, s.cols, s.rows)
		s.haveFrame = true
	}
	s.lastFrame = nil
	s.frameBuffer = nil
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
	frame := s.composeFrameIntoLocked(s.frameBuffer, placements, notice)
	prev := s.lastFrame
	s.lastFrame = frame
	if prev != nil {
		s.frameBuffer = prev
	} else {
		s.frameBuffer = nil
	}
	if len(placements) > 0 {
		// Component panes can represent a history viewport. Preserve the old
		// TUI's panel-local scroll path: a split panel must not force every
		// full-width row (and its sibling panels) through the diff writer.
		regions := make([]render.ScrollRect, 0, len(placements))
		allowPhysicalScroll := true
		for _, placement := range placements {
			if placement.DisableScrollOptimization {
				allowPhysicalScroll = false
				continue
			}
			r := placement.ContentRect
			if r.Empty() {
				continue
			}
			regions = append(regions, render.ScrollRect{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height})
		}
		if !allowPhysicalScroll {
			// A live PTY owns its own viewport and commonly redraws several
			// rows per output event. Keep the host diff atomic so the terminal
			// never presents the half-written viewport between row updates.
			return frame.BytesWithoutPhysicalScrollSynchronized(prev)
		}
		return frame.BytesWithSynchronizedScrollRegions(prev, regions)
	}
	return frame.Bytes(prev)
}

func (s *Session) composeFrameLocked(placements []Placement, notice *kernel.Frame) *render.Frame {
	if s.compositor == nil {
		s.compositor = NewCompositor(s.cols, s.rows)
	}
	return s.compositor.Compose(s.frame, placements, notice, s.coreOverlay)
}

func (s *Session) composeFrameIntoLocked(dst *render.Frame, placements []Placement, notice *kernel.Frame) *render.Frame {
	if s.compositor == nil {
		s.compositor = NewCompositor(s.cols, s.rows)
	}
	return s.compositor.ComposeInto(dst, s.frame, placements, notice, s.coreOverlay)
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

// refreshFocusLocked refreshes the terminal capability bits of the cached
// focus, because the program can toggle DEC mouse modes and the host can
// enter/leave history after the view was solved. The focus is replaced with a
// copy, never mutated, so pointers handed out by Focus() stay race-free.
func (s *Session) refreshFocusLocked() {
	if s.focus == nil {
		return
	}
	tracked := s.focus.MouseTracking
	if s.mouseTracker != nil {
		tracked = s.mouseTracker(s.focus.ID)
	}
	historyActive := s.focus.HistoryActive
	if s.historyActive != nil {
		historyActive = s.historyActive(s.focus.ID, s.focus.NodeID)
	}
	if tracked == s.focus.MouseTracking && historyActive == s.focus.HistoryActive {
		return
	}
	focus := *s.focus
	focus.MouseTracking = tracked
	focus.HistoryActive = historyActive
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
			"view_delta": true,
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
	class, key := eventOutputClass(ev)
	return s.writeLocked(wire.TypeEvent, ev, class, key)
}

func (s *Session) sendResponseLocked(epoch, requestID uint64, ok bool, data *pb.MethodData, errMsg string) error {
	if ok {
		errMsg = ""
	} else if errMsg == "" {
		errMsg = "rejected"
	}
	return s.writeLocked(wire.TypeResponse, &pb.Response{
		RequestId: requestID,
		Epoch:     epoch,
		Ok:        ok,
		Data:      data,
		Error:     errMsg,
	}, outputControl, "response")
}

func (s *Session) writeLocked(t wire.Type, m gproto.Message, class outputClass, key string) error {
	if s.out != nil {
		return s.out.enqueue(t, m, class, key)
	}
	return s.enc.Encode(t, m)
}

func eventOutputClass(ev *pb.Event) (outputClass, string) {
	switch ev.GetEvent().(type) {
	case *pb.Event_Notice:
		return outputCoalescible, "notice"
	case *pb.Event_Sources:
		return outputCoalescible, "sources"
	case *pb.Event_Key, *pb.Event_Paste, *pb.Event_Resize, *pb.Event_Mouse, *pb.Event_Wheel:
		return outputReliable, "input"
	default:
		return outputControl, "control"
	}
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
	f := &Focus{ID: found.Content.Self, NodeID: found.ID, Input: append([]string(nil), found.Input...)}
	if src := s.sourceLocked(f.ID); src != nil {
		f.IsTerminal = src.GetKind() == "terminal"
	}
	if s.mouseTracker != nil {
		f.MouseTracking = s.mouseTracker(f.ID)
	}
	if s.historyActive != nil {
		f.HistoryActive = s.historyActive(f.ID, f.NodeID)
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
		Id:            src.GetId(),
		Kind:          src.GetKind(),
		Title:         src.GetTitle(),
		Endpoint:      src.GetEndpoint(),
		TerminalId:    src.GetTerminalId(),
		Attached:      src.GetAttached(),
		Exited:        src.GetExited(),
		ExitCode:      src.GetExitCode(),
		Health:        src.GetHealth(),
		ResizeOwner:   src.GetResizeOwner(),
		OwnerEpoch:    src.GetOwnerEpoch(),
		LastSeenMs:    src.GetLastSeenMs(),
		Cols:          src.GetCols(),
		Rows:          src.GetRows(),
		Tags:          src.GetTags(),
		EndpointLabel: src.GetEndpointLabel(),
		LastOutputMs:  src.GetLastOutputMs(),
		// AttachmentCount carries the daemon observer count; SetSources clones
		// every item, so dropping it here would zero the pane badge count.
		AttachmentCount: src.GetAttachmentCount(),
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

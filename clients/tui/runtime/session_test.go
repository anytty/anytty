package runtime

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"testing"

	gproto "google.golang.org/protobuf/proto"

	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"

	"github.com/anytty/anytty/clients/tui/kernel"
)

type harness struct {
	t       *testing.T
	s       *Session
	toHost  *bytes.Buffer
	program *bytes.Buffer
	dec     *wire.Decoder
}

func newHarness(t *testing.T, opts Options) *harness {
	t.Helper()
	toHost := &bytes.Buffer{}
	program := &bytes.Buffer{}
	s := NewSession(opts, toHost, program)
	return &harness{
		t:       t,
		s:       s,
		toHost:  toHost,
		program: program,
		dec:     wire.NewDecoder(program, wire.RoleProgram, 0),
	}
}

func (h *harness) send(typ wire.Type, m gproto.Message) {
	h.t.Helper()
	if err := wire.NewEncoder(h.toHost, wire.RoleProgram, 0).Encode(typ, m); err != nil {
		h.t.Fatalf("encode %v: %v", typ, err)
	}
	got, payload, err := h.s.ReadFrame()
	if err != nil {
		h.t.Fatalf("ReadFrame: %v", err)
	}
	if got != typ {
		h.t.Fatalf("frame type = %v, want %v", got, typ)
	}
	if err := h.s.HandleFrame(got, payload); err != nil {
		h.t.Fatalf("HandleFrame(%v): %v", typ, err)
	}
}

func (h *harness) sendView(v *pb.View)     { h.send(wire.TypeView, v) }
func (h *harness) sendResult(r *pb.Result) { h.send(wire.TypeResult, r) }

func (h *harness) drain() []gproto.Message {
	h.t.Helper()
	var out []gproto.Message
	for {
		_, m, err := h.dec.DecodeMessage()
		if errors.Is(err, io.EOF) {
			return out
		}
		if err != nil {
			h.t.Fatalf("DecodeMessage: %v", err)
		}
		out = append(out, m)
	}
}

func (h *harness) rect(id string) (kernel.Rect, bool) {
	h.t.Helper()
	frame, ok := h.s.Frame()
	if !ok {
		return kernel.Rect{}, false
	}
	return frame.Rect(id)
}

func box(id string) *pb.Box {
	return &pb.Box{Id: id}
}

func textBox(id, text string) *pb.Box {
	return &pb.Box{Id: id, Content: &pb.Content{Text: text}}
}

func focusBox(id, self string, input ...string) *pb.Box {
	return &pb.Box{Id: id, Focused: true, Input: input, Content: &pb.Content{Self: self}}
}

func view(epoch, rev uint64, claim []string, all bool, children ...*pb.Box) *pb.View {
	v := &pb.View{Epoch: epoch, Rev: rev, Root: &pb.Box{Id: "root", Flow: "col", Children: children}}
	if claim != nil || all {
		v.Keys = &pb.Keys{Claim: claim, All: all}
	}
	return v
}

func source(id, kind string) *pb.Source {
	return &pb.Source{Id: id, Kind: kind, Title: id, Endpoint: "local", TerminalId: id}
}

func termSource(id string) *pb.Source {
	return &pb.Source{
		Id:          id,
		Kind:        "terminal",
		Title:       "main",
		Endpoint:    "local",
		TerminalId:  "main",
		Attached:    true,
		Exited:      false,
		ExitCode:    0,
		Health:      "ok",
		ResizeOwner: "view:client-a:1",
		OwnerEpoch:  7,
		LastSeenMs:  123456,
	}
}

func TestEpochRevAndRestart(t *testing.T) {
	h := newHarness(t, Options{ViewID: "view:client-a:1", Cols: 80, Rows: 24})
	h.sendView(view(1, 1, nil, false, textBox("a", "A")))
	if h.s.Rev() != 1 {
		t.Fatalf("rev = %d, want 1", h.s.Rev())
	}
	h.sendView(view(0, 9, nil, false, textBox("x", "X")))
	h.sendView(view(2, 3, nil, false, textBox("y", "Y")))
	if h.s.Rev() != 1 {
		t.Fatalf("stale epoch changed rev to %d", h.s.Rev())
	}
	if _, ok := h.rect("x"); ok {
		t.Fatal("stale epoch view was applied")
	}
	h.sendView(view(1, 1, nil, false, textBox("dup", "D")))
	if _, ok := h.rect("dup"); ok {
		t.Fatal("non-monotonic rev was applied")
	}
	h.sendView(view(1, 5, nil, false, textBox("b", "B")))
	if h.s.Rev() != 5 {
		t.Fatalf("rev = %d, want 5", h.s.Rev())
	}
	h.sendView(view(1, 2, nil, false, textBox("old", "O")))
	if _, ok := h.rect("old"); ok {
		t.Fatal("out-of-order rev was applied")
	}

	if err := h.s.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if h.s.Epoch() != 2 {
		t.Fatalf("epoch = %d, want 2", h.s.Epoch())
	}
	h.sendView(view(1, 100, nil, false, textBox("stale", "S")))
	if h.s.Rev() != 0 {
		t.Fatalf("old epoch after reset applied, rev = %d", h.s.Rev())
	}
	h.sendView(view(2, 1, nil, false, textBox("c", "C")))
	if h.s.Rev() != 1 {
		t.Fatalf("first frame of new epoch dropped, rev = %d", h.s.Rev())
	}
	if _, ok := h.rect("c"); !ok {
		t.Fatal("first frame of new epoch not rendered")
	}
}

func TestResultExactlyOnce(t *testing.T) {
	h := newHarness(t, Options{ViewID: "v", Cols: 80, Rows: 24})
	restart := &pb.Result{
		RequestId: 7,
		Epoch:     1,
		Method:    "terminal.restart",
		Params:    &pb.MethodParams{Endpoint: "local", Id: "main"},
	}
	h.sendResult(restart)
	msgs := h.drain()
	if len(msgs) != 1 {
		t.Fatalf("restart produced %d frames, want 1", len(msgs))
	}
	resp, ok := msgs[0].(*pb.Response)
	if !ok {
		t.Fatalf("frame is %T, want *pb.Response", msgs[0])
	}
	if resp.GetRequestId() != 7 || resp.GetEpoch() != 1 || !resp.GetOk() {
		t.Fatalf("response = %+v, want request 7 epoch 1 ok", resp)
	}
	if resp.Error != "" {
		t.Fatalf("ok response carries error %q", resp.Error)
	}

	h.sendResult(&pb.Result{RequestId: 8, Epoch: 1, Method: "not.a.method"})
	msgs = h.drain()
	if len(msgs) != 1 {
		t.Fatalf("unknown method produced %d frames, want 1", len(msgs))
	}
	resp = msgs[0].(*pb.Response)
	if resp.GetOk() || !strings.Contains(resp.GetError(), "unknown method") {
		t.Fatalf("unknown method response = %+v", resp)
	}

	h.sendResult(&pb.Result{RequestId: 9, Epoch: 1, Method: "terminal.scroll", Params: &pb.MethodParams{Endpoint: "local"}})
	msgs = h.drain()
	if len(msgs) != 1 {
		t.Fatalf("invalid params produced %d frames, want 1", len(msgs))
	}
	resp = msgs[0].(*pb.Response)
	if resp.GetOk() || !strings.Contains(resp.GetError(), "params.id") {
		t.Fatalf("invalid params response = %+v", resp)
	}

	h.sendResult(restart)
	msgs = h.drain()
	if len(msgs) != 1 {
		t.Fatalf("duplicate produced %d frames, want 1", len(msgs))
	}
	resp = msgs[0].(*pb.Response)
	if resp.GetOk() || !strings.Contains(resp.GetError(), "duplicate request_id") {
		t.Fatalf("duplicate response = %+v", resp)
	}

	h.sendResult(&pb.Result{RequestId: 10, Epoch: 99, Method: "system.quit"})
	msgs = h.drain()
	if len(msgs) != 1 {
		t.Fatalf("stale epoch produced %d frames, want 1", len(msgs))
	}
	resp = msgs[0].(*pb.Response)
	if resp.GetOk() || resp.GetError() != "epoch reset" || resp.GetEpoch() != 99 {
		t.Fatalf("stale epoch response = %+v", resp)
	}
}

type pendingHandler struct {
	calls int
}

func (p *pendingHandler) Handle(Request) (Outcome, bool) {
	p.calls++
	return Outcome{}, true
}

func TestThrottleAndComplete(t *testing.T) {
	pending := &pendingHandler{}
	h := newHarness(t, Options{
		ViewID:  "v",
		Cols:    80,
		Rows:    24,
		Limits:  Limits{MaxNodes: 100, MaxMessageBytes: 1 << 20, MaxInflightRequests: 1},
		Handler: pending,
	})
	h.sendResult(&pb.Result{RequestId: 1, Epoch: 1, Method: "system.quit"})
	if h.s.Pending() != 1 {
		t.Fatalf("pending = %d, want 1", h.s.Pending())
	}
	if msgs := h.drain(); len(msgs) != 0 {
		t.Fatalf("pending request answered early: %v", msgs)
	}
	h.sendResult(&pb.Result{RequestId: 2, Epoch: 1, Method: "system.quit"})
	msgs := h.drain()
	if len(msgs) != 1 {
		t.Fatalf("throttled request produced %d frames, want 1", len(msgs))
	}
	resp := msgs[0].(*pb.Response)
	if resp.GetOk() || resp.GetError() != "throttled" || resp.GetRequestId() != 2 {
		t.Fatalf("throttle response = %+v", resp)
	}
	if pending.calls != 1 {
		t.Fatalf("handler calls = %d, want 1 (throttled must not execute)", pending.calls)
	}

	if err := h.s.Complete(1, &pb.MethodData{Text: "done"}, ""); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	msgs = h.drain()
	if len(msgs) != 1 {
		t.Fatalf("completion produced %d frames, want 1", len(msgs))
	}
	resp = msgs[0].(*pb.Response)
	if !resp.GetOk() || resp.GetRequestId() != 1 || resp.GetData().GetText() != "done" {
		t.Fatalf("completion response = %+v", resp)
	}
	if h.s.Pending() != 0 {
		t.Fatalf("pending = %d, want 0", h.s.Pending())
	}
	if err := h.s.Complete(1, nil, ""); err == nil {
		t.Fatal("completing a non-pending request must fail")
	}
}

func TestResetInvalidatesInflight(t *testing.T) {
	pending := &pendingHandler{}
	h := newHarness(t, Options{
		ViewID:  "v",
		Cols:    80,
		Rows:    24,
		Limits:  Limits{MaxNodes: 100, MaxMessageBytes: 1 << 20, MaxInflightRequests: 4},
		Handler: pending,
	})
	h.sendResult(&pb.Result{RequestId: 1, Epoch: 1, Method: "system.quit"})
	if err := h.s.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	msgs := h.drain()
	if len(msgs) != 1 {
		t.Fatalf("reset produced %d frames, want 1", len(msgs))
	}
	resp := msgs[0].(*pb.Response)
	if resp.GetOk() || resp.GetError() != "epoch reset" || resp.GetRequestId() != 1 || resp.GetEpoch() != 1 {
		t.Fatalf("epoch reset response = %+v", resp)
	}
	if h.s.Pending() != 0 {
		t.Fatalf("pending after reset = %d, want 0", h.s.Pending())
	}
	if err := h.s.Complete(1, nil, ""); err == nil {
		t.Fatal("completing an invalidated request must fail")
	}
	h.sendResult(&pb.Result{RequestId: 2, Epoch: 2, Method: "system.quit"})
	if h.s.Pending() != 1 {
		t.Fatalf("new epoch request not pending: %d", h.s.Pending())
	}
	if err := h.s.Complete(2, nil, ""); err != nil {
		t.Fatalf("Complete new epoch: %v", err)
	}
	h.drain()
}

func TestResetClearsSessionState(t *testing.T) {
	h := newHarness(t, Options{
		ViewID:        "v",
		Cols:          80,
		Rows:          24,
		MouseTracking: func(id string) bool { return id == "terminal:local:main" },
	})
	if err := h.s.SetSources([]*pb.Source{termSource("terminal:local:main")}); err != nil {
		t.Fatalf("SetSources: %v", err)
	}
	h.sendView(view(1, 1, []string{"ctrl-p"}, false,
		focusBox("term", "terminal:local:main", "key", "paste", "wheel")))
	h.s.Capture("divider")
	h.drain()

	if got := h.s.Route(InputEvent{Kind: InputKey, Key: "a"}); got != DestinationPTY {
		t.Fatalf("route key = %v, want pty", got)
	}
	if got := h.s.Route(InputEvent{Kind: InputWheel}); got != DestinationPTY {
		t.Fatalf("route wheel = %v, want pty", got)
	}
	if _, _ = h.s.Claim(); len(h.s.Focus().Input) == 0 {
		t.Fatal("focus not extracted")
	}
	if h.s.CapturedNode() != "divider" {
		t.Fatalf("captured = %q, want divider", h.s.CapturedNode())
	}
	h.s.OpenCoreOverlay(kernel.Frame{})
	if got := h.s.Route(InputEvent{Kind: InputKey, Key: "a"}); got != DestinationHost {
		t.Fatalf("route under overlay = %v, want host", got)
	}
	if !h.s.CoreOverlayOpen() {
		t.Fatal("core overlay should be open")
	}

	if err := h.s.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	h.drain()
	if got := h.s.Route(InputEvent{Kind: InputKey, Key: "a"}); got != DestinationProgram {
		t.Fatalf("route after reset = %v, want program", got)
	}
	if claim, all := h.s.Claim(); len(claim) != 0 || all {
		t.Fatalf("claim after reset = %v/%v", claim, all)
	}
	if h.s.Focus() != nil {
		t.Fatal("focus after reset must be nil")
	}
	if h.s.CapturedNode() != "" {
		t.Fatal("capture after reset must be cleared")
	}
	if h.s.CoreOverlayOpen() {
		t.Fatal("core overlay after reset must be closed")
	}
	if _, ok := h.s.Frame(); ok {
		t.Fatal("frame after reset must be empty")
	}
	if h.s.View() != nil {
		t.Fatal("view after reset must be nil")
	}
}

func TestSetSourcesFullSnapshot(t *testing.T) {
	h := newHarness(t, Options{ViewID: "v", Cols: 80, Rows: 24})
	want := termSource("terminal:local:main")
	ownerless := &pb.Source{Id: "terminal:local:two", Kind: "terminal", Endpoint: "local", TerminalId: "two"}
	if err := h.s.SetSources([]*pb.Source{want, ownerless}); err != nil {
		t.Fatalf("SetSources: %v", err)
	}
	msgs := h.drain()
	if len(msgs) != 1 {
		t.Fatalf("snapshot produced %d frames, want 1", len(msgs))
	}
	ev, ok := msgs[0].(*pb.Event)
	if !ok {
		t.Fatalf("frame is %T, want *pb.Event", msgs[0])
	}
	snap := ev.GetSources()
	if snap == nil {
		t.Fatal("event is not a sources snapshot")
	}
	if len(snap.Items) != 2 {
		t.Fatalf("snapshot has %d items, want 2", len(snap.Items))
	}
	if !gproto.Equal(snap.Items[0], want) {
		t.Fatalf("item 0 = %+v, want %+v", snap.Items[0], want)
	}
	if snap.Items[1].GetResizeOwner() != "" || snap.Items[1].GetOwnerEpoch() != 0 {
		t.Fatal("ownerless item must carry empty owner fields, not omit them")
	}

	back := h.s.Sources()
	if len(back) != 2 || !gproto.Equal(back[0], want) {
		t.Fatalf("Sources() = %+v", back)
	}
	back[0].Id = "mutated"
	if h.s.Sources()[0].GetId() != want.GetId() {
		t.Fatal("Sources() must return copies")
	}

	want.OwnerEpoch = 9
	h.s.SetSources([]*pb.Source{want})
	h.drain()
	if h.s.Sources()[0].GetOwnerEpoch() != 9 {
		t.Fatal("snapshot must replace the previous state")
	}
}

func TestViewRejectedOnce(t *testing.T) {
	h := newHarness(t, Options{
		ViewID: "v",
		Cols:   80,
		Rows:   24,
		Limits: Limits{MaxNodes: 2, MaxMessageBytes: 1 << 20, MaxInflightRequests: 4},
	})
	h.sendView(view(1, 1, []string{"ctrl-p"}, false, textBox("a", "A")))
	h.drain()
	if claim, _ := h.s.Claim(); len(claim) != 1 {
		t.Fatalf("claim = %v, want ctrl-p", claim)
	}

	oversized := view(1, 2, nil, false, textBox("a", "A"), textBox("b", "B"))
	h.sendView(oversized)
	msgs := h.drain()
	if len(msgs) != 1 {
		t.Fatalf("max_nodes rejection produced %d frames, want 1", len(msgs))
	}
	ev := msgs[0].(*pb.Event)
	rej := ev.GetViewRejected()
	if rej == nil || rej.GetEpoch() != 1 || rej.GetRev() != 2 || rej.GetReason() != "max_nodes" {
		t.Fatalf("view_rejected = %+v", rej)
	}
	if h.s.Rev() != 1 {
		t.Fatalf("rejected view changed rev to %d", h.s.Rev())
	}
	if claim, _ := h.s.Claim(); len(claim) != 1 || claim[0] != "ctrl-p" {
		t.Fatalf("rejected view changed claim to %v", claim)
	}

	h.sendView(oversized)
	if msgs := h.drain(); len(msgs) != 0 {
		t.Fatalf("same (epoch,rev) rejected twice: %v", msgs)
	}

	h.sendView(view(1, 3, nil, false, textBox("a", "A"), textBox("b", "B")))
	msgs = h.drain()
	if len(msgs) != 1 || msgs[0].(*pb.Event).GetViewRejected().GetRev() != 3 {
		t.Fatalf("new rev must be rejected again: %v", msgs)
	}
}

func TestOversizeFrameHandling(t *testing.T) {
	limits := Limits{MaxNodes: 16, MaxMessageBytes: 64, MaxPasteBytes: 16, MaxInflightRequests: 4}
	valid := view(1, 7, nil, false, textBox("small", "ok"))
	validFrame, err := wire.Marshal(wire.TypeView, valid, 0)
	if err != nil {
		t.Fatalf("Marshal valid: %v", err)
	}

	big := make([]byte, 4+200)
	binary.BigEndian.PutUint32(big[:4], 200)
	big[4] = byte(wire.TypeView)
	for i := 5; i < len(big); i++ {
		big[i] = 0x08
	}

	var stream bytes.Buffer
	stream.Write(big)
	stream.Write(validFrame)
	var out bytes.Buffer
	s := NewSession(Options{ViewID: "v", Cols: 80, Rows: 24, Limits: limits}, &stream, &out)
	if err := s.Serve(); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	dec := wire.NewDecoder(&out, wire.RoleProgram, 0)
	_, m, err := dec.DecodeMessage()
	if err != nil {
		t.Fatalf("decode rejection: %v", err)
	}
	rej := m.(*pb.Event).GetViewRejected()
	if rej == nil || rej.GetReason() != "oversize" || rej.GetRev() != 0 {
		t.Fatalf("oversize view_rejected = %+v", rej)
	}
	if s.Rev() != 7 {
		t.Fatalf("frame after oversize was not processed, rev = %d", s.Rev())
	}

	if err := s.HandleOversize(wire.TypeResult); err != nil {
		t.Fatalf("HandleOversize result: %v", err)
	}
	_, m, err = dec.DecodeMessage()
	if err != nil {
		t.Fatalf("decode oversize response: %v", err)
	}
	resp := m.(*pb.Response)
	if resp.GetError() != "oversize" || resp.GetRequestId() != 0 {
		t.Fatalf("oversize response = %+v", resp)
	}

	if err := s.HandleFrame(wire.TypeHello, nil); !wire.IsKind(err, wire.KindDirection) {
		t.Fatalf("HELLO from program: err = %v, want KindDirection", err)
	}
}

func TestCaptureClearedOnViewChange(t *testing.T) {
	h := newHarness(t, Options{ViewID: "v", Cols: 80, Rows: 24})
	h.sendView(view(1, 1, nil, false, box("divider")))
	h.s.Capture("divider")
	h.sendView(view(1, 2, nil, false, textBox("other", "x")))
	if h.s.CapturedNode() != "" {
		t.Fatal("capture must clear when the node leaves the view")
	}

	h.s.Capture("root")
	h.sendView(view(1, 3, nil, false, textBox("other", "x")))
	if h.s.CapturedNode() != "root" {
		t.Fatal("capture must survive when the node is still in the view")
	}
}

func TestSessionFocusRouting(t *testing.T) {
	h := newHarness(t, Options{
		ViewID:        "v",
		Cols:          80,
		Rows:          24,
		MouseTracking: func(id string) bool { return id == "terminal:local:main" },
	})
	h.sendView(view(1, 1, []string{"ctrl-f"}, false,
		focusBox("term", "terminal:local:main", "key", "paste", "wheel")))
	if err := h.s.SetSources([]*pb.Source{termSource("terminal:local:main")}); err != nil {
		t.Fatalf("SetSources: %v", err)
	}
	h.drain()

	cases := []struct {
		name string
		ev   InputEvent
		want Destination
	}{
		{"key to pty", InputEvent{Kind: InputKey, Key: "a"}, DestinationPTY},
		{"ctrl-c to pty", InputEvent{Kind: InputKey, Key: KeyCtrlC}, DestinationPTY},
		{"claimed key to program", InputEvent{Kind: InputKey, Key: "ctrl-f"}, DestinationProgram},
		{"ctrl-q to host", InputEvent{Kind: InputKey, Key: KeyCtrlQ}, DestinationHost},
		{"paste to pty", InputEvent{Kind: InputPaste}, DestinationPTY},
		{"wheel to pty", InputEvent{Kind: InputWheel}, DestinationPTY},
		{"mouse hit to pty", InputEvent{Kind: InputMouse, HitFocused: true}, DestinationPTY},
		{"mouse miss to program", InputEvent{Kind: InputMouse}, DestinationProgram},
	}
	for _, tc := range cases {
		if got := h.s.Route(tc.ev); got != tc.want {
			t.Fatalf("%s: route = %v, want %v", tc.name, got, tc.want)
		}
	}

	h.s.OpenCoreOverlay(kernel.Frame{})
	if got := h.s.Route(InputEvent{Kind: InputKey, Key: "a"}); got != DestinationHost {
		t.Fatalf("overlay route = %v, want host", got)
	}
	h.s.CloseCoreOverlay()

	h.s.SetSources([]*pb.Source{source("picker:local:1", "picker")})
	h.sendView(view(1, 2, nil, false, focusBox("term", "picker:local:1", "key")))
	if got := h.s.Route(InputEvent{Kind: InputKey, Key: "a"}); got != DestinationComponent {
		t.Fatalf("non-terminal route = %v, want component", got)
	}

	noTracking := newHarness(t, Options{ViewID: "v2", Cols: 80, Rows: 24})
	noTracking.sendView(view(1, 1, nil, false, focusBox("term", "terminal:local:main", "key", "wheel")))
	if got := noTracking.s.Route(InputEvent{Kind: InputWheel}); got != DestinationProgram {
		t.Fatalf("wheel without tracking = %v, want program", got)
	}
}

func TestStubHandlerData(t *testing.T) {
	stub := &StubHandler{Clipboard: "copied", Rows: map[string][]string{"local:main": {"r1", "r2"}}}
	h := newHarness(t, Options{ViewID: "v", Cols: 80, Rows: 24, Handler: stub})

	h.sendResult(&pb.Result{RequestId: 1, Epoch: 1, Method: "terminal.create", Params: &pb.MethodParams{Endpoint: "local"}})
	msgs := h.drain()
	resp := msgs[0].(*pb.Response)
	if !resp.GetOk() || resp.GetData().GetEndpoint() != "local" || resp.GetData().GetId() == "" {
		t.Fatalf("create response = %+v", resp)
	}
	createID := resp.GetData().GetId()

	h.sendResult(&pb.Result{RequestId: 2, Epoch: 1, Method: "terminal.create", Params: &pb.MethodParams{Endpoint: "local"}})
	resp = h.drain()[0].(*pb.Response)
	if resp.GetData().GetId() == createID {
		t.Fatal("stub create ids must be unique")
	}

	h.sendResult(&pb.Result{RequestId: 3, Epoch: 1, Method: "terminal.scroll", Params: &pb.MethodParams{Endpoint: "local", Id: "main", Delta: 5}})
	resp = h.drain()[0].(*pb.Response)
	if !resp.GetOk() || len(resp.GetData().GetRows()) != 2 || resp.GetData().GetRows()[0] != "r1" {
		t.Fatalf("scroll response = %+v", resp)
	}

	h.sendResult(&pb.Result{RequestId: 4, Epoch: 1, Method: "clipboard.read"})
	resp = h.drain()[0].(*pb.Response)
	if !resp.GetOk() || resp.GetData().GetText() != "copied" {
		t.Fatalf("clipboard response = %+v", resp)
	}
}

func TestHelloRegistry(t *testing.T) {
	h := newHarness(t, Options{ViewID: "view:client-a:1", Cols: 120, Rows: 32, Components: []string{"terminal"}})
	hello := h.s.Hello()
	if hello.GetViewId() != "view:client-a:1" || hello.GetEpoch() != 1 || hello.GetSchema() != 1 {
		t.Fatalf("hello identity = %+v", hello)
	}
	if hello.GetCols() != 120 || hello.GetRows() != 32 {
		t.Fatalf("hello viewport = %dx%d", hello.GetCols(), hello.GetRows())
	}
	wantMethods := []string{
		"terminal.attach", "terminal.create", "terminal.restart", "terminal.kill", "terminal.remove",
		"terminal.scroll", "terminal.scrollEnd", "terminal.copy", "history.window",
		"clipboard.read", "input.forward", "system.quit", "endpoint.sync",
	}
	if strings.Join(hello.Methods, ",") != strings.Join(wantMethods, ",") {
		t.Fatalf("hello.methods = %v", hello.Methods)
	}
	for _, name := range []string{"key", "paste", "mouse", "wheel", "resize", "sources", "notice", "component", "view_rejected"} {
		found := false
		for _, ev := range hello.Events {
			if ev == name {
				found = true
			}
		}
		if !found {
			t.Fatalf("hello.events missing %q: %v", name, hello.Events)
		}
	}
	if !hello.GetFeatures()["component"] || hello.GetFeatures()["state.save"] || hello.GetFeatures()["state.load"] {
		t.Fatalf("hello.features = %v", hello.GetFeatures())
	}
	if hello.GetLimits().GetMaxInflightRequests() != 64 || hello.GetLimits().GetMaxMessageBytes() != 1<<20 {
		t.Fatalf("hello.limits = %+v", hello.GetLimits())
	}

	if err := h.s.SendHello(); err != nil {
		t.Fatalf("SendHello: %v", err)
	}
	_, m, err := h.dec.DecodeMessage()
	if err != nil {
		t.Fatalf("decode hello: %v", err)
	}
	if !gproto.Equal(m, hello) {
		t.Fatalf("wire hello = %+v, want %+v", m, hello)
	}
}

func TestMethodRegistry(t *testing.T) {
	want := []struct {
		name    string
		confirm bool
		data    DataKind
	}{
		{"terminal.attach", false, DataNone},
		{"terminal.create", false, DataCreate},
		{"terminal.restart", false, DataNone},
		{"terminal.kill", true, DataNone},
		{"terminal.remove", true, DataNone},
		{"terminal.scroll", false, DataRows},
		{"terminal.scrollEnd", false, DataNone},
		{"terminal.copy", false, DataNone},
		{"history.window", false, DataRows},
		{"clipboard.read", true, DataText},
		{"input.forward", false, DataNone},
		{"system.quit", true, DataNone},
		{"endpoint.sync", false, DataNone},
	}
	got := Methods()
	if len(got) != len(want) {
		t.Fatalf("registry has %d methods, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Name != w.name || got[i].Confirm != w.confirm || got[i].Data != w.data {
			t.Fatalf("method %d = %+v, want %+v", i, got[i], w)
		}
	}
	if _, ok := LookupMethod("terminal.attach"); !ok {
		t.Fatal("terminal.attach must be registered")
	}
	if _, ok := LookupMethod("nope"); ok {
		t.Fatal("unknown method must not be registered")
	}
}

// TestViewContentPropsPassThrough pins the M2 contract: content.props ride
// the accepted view into the kernel node and Session.BoxProps hands them to
// the component factory untouched, as an independent copy.
func TestViewContentPropsPassThrough(t *testing.T) {
	h := newHarness(t, Options{ViewID: "view:local:1", Cols: 40, Rows: 10})
	term := focusBox("slot-1", "terminal:local:main", "key")
	term.Content.Props = map[string]string{
		"chrome.border": "fg:#565f89",
		"vendor.future": "opaque",
	}
	h.sendView(view(1, 1, nil, false, term))

	props := h.s.BoxProps("slot-1")
	if len(props) != 2 || props["chrome.border"] != "fg:#565f89" || props["vendor.future"] != "opaque" {
		t.Fatalf("BoxProps = %v, want the wire map untouched", props)
	}
	props["chrome.border"] = "fg:#000000"
	if again := h.s.BoxProps("slot-1"); again["chrome.border"] != "fg:#565f89" {
		t.Fatalf("BoxProps leaked its backing map: %v", again)
	}
	if h.s.BoxProps("missing") != nil {
		t.Fatal("BoxProps on an unknown node must be nil")
	}
	if h.s.BoxProps("") != nil {
		t.Fatal("BoxProps with an empty id must be nil")
	}
}

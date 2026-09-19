package runtime

import (
	"strings"
	"sync"
	"testing"
	"time"

	pb "github.com/anytty/anytty/proto/ui/protobuf"

	"github.com/anytty/anytty/clients/tui/runtime/keys"
)

// eventRecorder is the testable program delivery interface (M3): it captures
// every EVENT the session would have written to the program pipe.
type eventRecorder struct {
	mu     sync.Mutex
	events []*pb.Event
}

func (r *eventRecorder) SendEvent(ev *pb.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, ev)
	return nil
}

func (r *eventRecorder) count(get func(*pb.Event) bool) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, ev := range r.events {
		if get(ev) {
			n++
		}
	}
	return n
}

func (r *eventRecorder) lastKey(t *testing.T) *pb.KeyEvent {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.events) - 1; i >= 0; i-- {
		if k := r.events[i].GetKey(); k != nil {
			return k
		}
	}
	t.Fatal("no key event recorded")
	return nil
}

func (r *eventRecorder) wheelCount() int {
	return r.count(func(ev *pb.Event) bool { return ev.GetWheel() != nil })
}

func (r *eventRecorder) mouseCount() int {
	return r.count(func(ev *pb.Event) bool { return ev.GetMouse() != nil })
}

func (r *eventRecorder) pasteCount() int {
	return r.count(func(ev *pb.Event) bool { return ev.GetPaste() != nil })
}

// newInputHarness wires a real cat -v PTY, the full session routing state and
// the two host ports (InputSink = TerminalHandler, EventSink = recorder).
func newInputHarness(t *testing.T, claim []string, all bool, inputs ...string) (*harness, *TerminalHandler, *Terminal, *eventRecorder) {
	t.Helper()
	rec := &eventRecorder{}
	h := NewTerminalHandler(TerminalOptions{Cols: 80, Rows: 6, Command: []string{"cat", "-v"}})
	t.Cleanup(h.Close)
	hh := newHarness(t, Options{
		ViewID:    "view:client-a:1",
		Cols:      80,
		Rows:      24,
		EventSink: rec,
	})
	hh.s.SetInputSink(h)
	hh.s.SetMouseTracking(func(id string) bool {
		term, ok := h.TerminalBySource(id)
		return ok && term.Modes().MouseTracking()
	})
	term := attachTerminal(t, h, "main")
	hh.s.SetSources([]*pb.Source{termSource("terminal:local:main")})
	hh.sendView(view(1, 1, claim, all, focusBox("term", "terminal:local:main", inputs...)))
	hh.drain()
	return hh, h, term, rec
}

// TestInputRouteToPTYClaimAndForward is the M3 acceptance: with cat -v
// behind the focused terminal, an unclaimed key echoes, a claimed key does
// not reach the PTY, input.forward pushes it through, and a wheel without
// terminal mouse tracking stays out of the PTY.
func TestInputRouteToPTYClaimAndForward(t *testing.T) {
	hh, _, term, rec := newInputHarness(t, []string{"ctrl-a"}, false, "key", "paste", "wheel")

	if dst, err := hh.s.Input(keys.Event{Kind: keys.KindKey, Key: "ctrl-x"}); err != nil || dst != DestinationPTY {
		t.Fatalf("unclaimed key destination = %v err=%v, want pty", dst, err)
	}
	waitFor(t, "cat -v echo of ctrl-x", func() bool { return screenContains(term, "^X") })

	if dst, err := hh.s.Input(keys.Event{Kind: keys.KindKey, Key: "ctrl-a"}); err != nil || dst != DestinationProgram {
		t.Fatalf("claimed key destination = %v err=%v, want program", dst, err)
	}
	claimed := rec.lastKey(t)
	if claimed.GetKey() != "ctrl-a" || claimed.GetId() == "" {
		t.Fatalf("claimed key event = %+v, want ctrl-a with id", claimed)
	}
	time.Sleep(50 * time.Millisecond)
	if screenContains(term, "^A") {
		t.Fatal("claimed key leaked into the PTY")
	}

	hh.sendResult(&pb.Result{
		RequestId: 1, Epoch: 1, Method: "input.forward",
		Params: &pb.MethodParams{EventId: claimed.GetId(), Source: "terminal:local:main"},
	})
	msgs := hh.drain()
	if len(msgs) != 1 {
		t.Fatalf("forward produced %d frames, want 1", len(msgs))
	}
	resp, ok := msgs[0].(*pb.Response)
	if !ok || !resp.GetOk() {
		t.Fatalf("forward response = %+v, want ok", msgs[0])
	}
	waitFor(t, "cat -v echo after forward", func() bool { return screenContains(term, "^A") })

	hh.sendResult(&pb.Result{
		RequestId: 2, Epoch: 1, Method: "input.forward",
		Params: &pb.MethodParams{EventId: "ev-expired", Source: "terminal:local:main"},
	})
	resp, ok = hh.drain()[0].(*pb.Response)
	if !ok || resp.GetOk() || resp.GetError() != "event expired" {
		t.Fatalf("expired forward response = %+v", resp)
	}

	if dst, err := hh.s.Input(keys.Event{Kind: keys.KindWheel, Delta: 1, X: 3, Y: 2}); err != nil || dst != DestinationProgram {
		t.Fatalf("wheel without mouse tracking = %v err=%v, want program", dst, err)
	}
	if rec.wheelCount() != 1 {
		t.Fatalf("wheel events delivered = %d, want 1", rec.wheelCount())
	}
	if dst, err := hh.s.Input(keys.Event{Kind: keys.KindMouse, Action: keys.ActionPress, Button: keys.ButtonLeft, X: 1, Y: 1, HitFocused: true}); err != nil || dst != DestinationProgram {
		t.Fatalf("mouse without tracking = %v err=%v, want program", dst, err)
	}
	time.Sleep(50 * time.Millisecond)
	if screenContains(term, "<64") || screenContains(term, "<0;") {
		t.Fatal("wheel/mouse without tracking leaked into the PTY")
	}
}

func TestInputKeysAllAndPasteForward(t *testing.T) {
	hh, _, term, rec := newInputHarness(t, nil, true, "key", "wheel")

	if dst, err := hh.s.Input(keys.Event{Kind: keys.KindKey, Char: "Z"}); err != nil || dst != DestinationProgram {
		t.Fatalf("keys.all key destination = %v err=%v, want program", dst, err)
	}
	if got := rec.lastKey(t).GetKey(); got != "Z" {
		t.Fatalf("keys.all key name = %q, want Z", got)
	}
	time.Sleep(50 * time.Millisecond)
	if screenContains(term, "Z") {
		t.Fatal("keys.all key leaked into the PTY")
	}

	// A paste to a focused terminal that does not declare "paste" goes to the
	// program as one bounded, id-carrying chunk and can be forwarded back.
	hh.sendView(view(1, 2, nil, true, focusBox("term", "terminal:local:main", "key", "wheel")))
	if dst, err := hh.s.Input(keys.Event{Kind: keys.KindPaste, Text: "hello\r\nworld"}); err != nil || dst != DestinationProgram {
		t.Fatalf("paste destination = %v err=%v, want program", dst, err)
	}
	if rec.pasteCount() != 1 {
		t.Fatalf("paste events = %d, want 1", rec.pasteCount())
	}
	rec.mu.Lock()
	pasted := rec.events[len(rec.events)-1].GetPaste()
	rec.mu.Unlock()
	if pasted.GetText() != "hello\nworld" {
		t.Fatalf("paste text = %q, want normalized hello\\nworld", pasted.GetText())
	}

	hh.sendResult(&pb.Result{
		RequestId: 5, Epoch: 1, Method: "input.forward",
		Params: &pb.MethodParams{EventId: pasted.GetId(), Source: "terminal:local:main"},
	})
	resp, ok := hh.drain()[0].(*pb.Response)
	if !ok || !resp.GetOk() {
		t.Fatalf("paste forward response = %+v", resp)
	}
	waitFor(t, "cat -v echo of forwarded paste", func() bool {
		return screenContains(term, "hello") && screenContains(term, "world")
	})
}

// TestInputWheelPassesWithMouseTracking covers the positive side of §6.5
// priority 5: the terminal enables tracking after the view was solved, so
// the session must re-probe capabilities at route time before writing.
func TestInputWheelPassesWithMouseTracking(t *testing.T) {
	rec := &eventRecorder{}
	h := NewTerminalHandler(TerminalOptions{
		Cols:    80,
		Rows:    6,
		Command: []string{"sh", "-c", "printf '\\033[?1000h'; cat -v"},
	})
	defer h.Close()
	hh := newHarness(t, Options{ViewID: "v", Cols: 80, Rows: 24, EventSink: rec})
	hh.s.SetInputSink(h)
	hh.s.SetMouseTracking(func(id string) bool {
		term, ok := h.TerminalBySource(id)
		return ok && term.Modes().MouseTracking()
	})
	term := attachTerminal(t, h, "main")
	hh.s.SetSources([]*pb.Source{termSource("terminal:local:main")})
	hh.sendView(view(1, 1, nil, false, focusBox("term", "terminal:local:main", "key", "wheel")))
	waitFor(t, "mouse tracking enabled", func() bool { return term.Modes().MouseTracking() })

	if dst, err := hh.s.Input(keys.Event{Kind: keys.KindWheel, Delta: -1, X: 3, Y: 2}); err != nil || dst != DestinationPTY {
		t.Fatalf("wheel with tracking = %v err=%v, want pty", dst, err)
	}
	waitFor(t, "SGR wheel echo", func() bool { return screenContains(term, "<65") })
	if rec.wheelCount() != 0 {
		t.Fatal("wheel must not go to the program when passthrough conditions hold")
	}
}

func TestForwardRejectsUnknownSource(t *testing.T) {
	hh, _, _, rec := newInputHarness(t, []string{"ctrl-b"}, false, "key")
	if dst, _ := hh.s.Input(keys.Event{Kind: keys.KindKey, Key: "ctrl-b"}); dst != DestinationProgram {
		t.Fatalf("key destination = %v, want program", dst)
	}
	key := rec.lastKey(t)

	hh.sendResult(&pb.Result{
		RequestId: 9, Epoch: 1, Method: "input.forward",
		Params: &pb.MethodParams{EventId: key.GetId(), Source: "terminal:local:missing"},
	})
	resp, ok := hh.drain()[0].(*pb.Response)
	if !ok || resp.GetOk() || !strings.Contains(resp.GetError(), "unknown source") {
		t.Fatalf("unknown source response = %+v", resp)
	}

	detached := termSource("terminal:local:main")
	detached.Attached = false
	hh.s.SetSources([]*pb.Source{detached})
	hh.sendResult(&pb.Result{
		RequestId: 10, Epoch: 1, Method: "input.forward",
		Params: &pb.MethodParams{EventId: key.GetId(), Source: "terminal:local:main"},
	})
	resp, ok = hh.drain()[0].(*pb.Response)
	if !ok || resp.GetOk() || !strings.Contains(resp.GetError(), "not attached") {
		t.Fatalf("detached source response = %+v", resp)
	}
}

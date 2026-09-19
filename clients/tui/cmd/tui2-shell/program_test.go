package main

import (
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

type fakeEmit struct {
	method string
	params *pb.MethodParams
	cb     func(*pb.Response)
}

type fakeEmitter struct {
	hello   *pb.Hello
	epoch   uint64
	commits int
	root    *pb.Box
	keys    sdk.Keys
	emits   []fakeEmit
}

func (f *fakeEmitter) Hello() *pb.Hello { return f.hello }
func (f *fakeEmitter) Epoch() uint64    { return f.epoch }

func (f *fakeEmitter) Commit(root *pb.Box, keys sdk.Keys) error {
	f.commits++
	f.root = root
	f.keys = keys
	return nil
}

func (f *fakeEmitter) Emit(method string, params *pb.MethodParams, cb func(*pb.Response)) (uint64, error) {
	f.emits = append(f.emits, fakeEmit{method: method, params: params, cb: cb})
	return uint64(len(f.emits)), nil
}

func (f *fakeEmitter) answer(t *testing.T, index int, resp *pb.Response) {
	t.Helper()
	if index >= len(f.emits) || f.emits[index].cb == nil {
		t.Fatalf("no callback for emit %d", index)
	}
	f.emits[index].cb(resp)
}

func TestProgramPickerCreatesAndBinds(t *testing.T) {
	m := newModel()
	fake := &fakeEmitter{epoch: 1}
	p := newProgram(m, fake)

	p.onHello(&pb.Hello{ViewId: "view:local:1", Epoch: 1, Cols: 100, Rows: 30})
	if fake.commits != 1 {
		t.Fatalf("commits after hello = %d", fake.commits)
	}
	p.onSources(nil)
	if m.mode != modePicker || fake.commits != 2 {
		t.Fatalf("mode=%v commits=%d", m.mode, fake.commits)
	}

	p.onKey(&pb.KeyEvent{Id: "ev-1", Key: "enter"})
	if len(fake.emits) != 1 || fake.emits[0].method != "terminal.create" {
		t.Fatalf("emits = %+v", fake.emits)
	}
	fake.answer(t, 0, &pb.Response{RequestId: 1, Epoch: 1, Ok: true, Data: &pb.MethodData{Endpoint: "local", Id: "term-1"}})

	if m.focusSlot().sourceID != "terminal:local:term-1" {
		t.Fatalf("slot source = %q", m.focusSlot().sourceID)
	}
	if m.mode != modeNormal {
		t.Fatalf("mode = %v", m.mode)
	}
	slot := findBox(fake.root, m.focusSlot().id)
	if slot == nil || slot.GetContent().GetSelf() != "terminal:local:term-1" {
		t.Fatalf("committed view slot = %+v", slot)
	}
}

func TestProgramRoutesSourcesToPickerAndKeyboard(t *testing.T) {
	m := newModel()
	fake := &fakeEmitter{epoch: 1}
	p := newProgram(m, fake)
	p.onHello(&pb.Hello{ViewId: "view:local:1", Epoch: 1, Cols: 80, Rows: 24})
	p.onSources([]*pb.Source{terminalSource("terminal:local:main", "main", false)})

	if len(fake.emits) != 1 || fake.emits[0].method != "terminal.attach" {
		t.Fatalf("auto-bind emits = %+v", fake.emits)
	}
	if fake.keys.All || len(fake.keys.Claim) == 0 {
		t.Fatalf("NORMAL keys = %+v", fake.keys)
	}

	p.onKey(&pb.KeyEvent{Id: "ev-2", Key: "a", Char: "a"})
	if len(fake.emits) != 1 {
		t.Fatalf("typing must not emit methods, got %+v", fake.emits)
	}

	p.onKey(&pb.KeyEvent{Id: "ev-3", Key: "ctrl-p"})
	if m.mode != modePane || !fake.keys.All {
		t.Fatalf("mode=%v keys=%+v", m.mode, fake.keys)
	}

	p.onWheel(&pb.WheelEvent{Delta: 1, X: 3, Y: 3, Node: m.focusSlot().id})
	if len(fake.emits) != 2 || fake.emits[1].method != "terminal.scroll" {
		t.Fatalf("wheel emits = %+v", fake.emits)
	}

	p.onKey(&pb.KeyEvent{Id: "ev-4", Key: "esc"})
	if m.mode != modeNormal {
		t.Fatalf("mode = %v", m.mode)
	}
	if fake.commits < 5 {
		t.Fatalf("commits = %d, want a commit per state change", fake.commits)
	}
}

func TestProgramClockTickCommitsAfterHello(t *testing.T) {
	m := newModel()
	fake := &fakeEmitter{epoch: 1}
	p := newProgram(m, fake)

	p.tick()
	if fake.commits != 0 {
		t.Fatalf("tick before hello committed %d times", fake.commits)
	}

	p.onHello(&pb.Hello{ViewId: "view:local:1", Epoch: 1, Cols: 80, Rows: 24})
	fake.hello = &pb.Hello{ViewId: "view:local:1", Epoch: 1}
	before := fake.commits
	p.tick()
	if fake.commits != before+1 {
		t.Fatalf("tick commits = %d, want %d", fake.commits, before+1)
	}
}

func TestProgramPasteForwardsWhenUnfocused(t *testing.T) {
	m := newModel()
	fake := &fakeEmitter{epoch: 1}
	p := newProgram(m, fake)
	p.onHello(&pb.Hello{ViewId: "view:local:1", Epoch: 1, Cols: 80, Rows: 24})
	p.onSources([]*pb.Source{terminalSource("terminal:local:main", "main", false)})
	p.onKey(&pb.KeyEvent{Id: "ev-1", Key: "ctrl-p"})

	p.onPaste(&pb.PasteEvent{Id: "ev-9", Text: "echo hi\n"})
	last := fake.emits[len(fake.emits)-1]
	if last.method != "input.forward" || last.params.GetEventId() != "ev-9" || last.params.GetSource() != "terminal:local:main" {
		t.Fatalf("paste forward = %+v", last)
	}
}

// TestProgramPressedIsOneFrame: the press commit highlights the button, the
// redraw right after clears the mark, and a later commit carries the normal
// style (release or redraw both clear it).
func TestProgramPressedIsOneFrame(t *testing.T) {
	m := newModel()
	fake := &fakeEmitter{epoch: 1}
	p := newProgram(m, fake)
	p.onHello(&pb.Hello{ViewId: "view:local:1", Epoch: 1, Cols: 100, Rows: 30})
	p.onSources([]*pb.Source{terminalSource("terminal:local:main", "main", false)})

	node := slotButtonNode(m.focusSlot().id, slotButtonSplitH)
	p.onMouse(&pb.MouseEvent{Action: "press", Button: "left", Node: node, X: 3, Y: 1})
	if got := findBox(fake.root, node); got == nil || got.GetStyle() != m.style("button_pressed") {
		t.Fatalf("pressed commit style = %+v, want theme button_pressed", got)
	}
	if m.pressed != "" {
		t.Fatalf("pressed after the redraw = %q, want empty", m.pressed)
	}
	if len(m.activeTab().slots) != 2 {
		t.Fatalf("split-h slots = %d, want 2", len(m.activeTab().slots))
	}

	p.onMouse(&pb.MouseEvent{Action: "release", Button: "left", Node: node, X: 3, Y: 1})
	fake.hello = &pb.Hello{ViewId: "view:local:1", Epoch: 1}
	p.tick()
	// The split moved the focus to the new slot, so the released commit shows
	// the plain button style for the now-unfocused slot.
	if got := findBox(fake.root, node); got == nil || got.GetStyle() != m.style("button") {
		t.Fatalf("released commit style = %+v, want theme button", got)
	}
}

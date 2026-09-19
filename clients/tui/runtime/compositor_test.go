package runtime

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/kernel"
	"github.com/anytty/anytty/clients/tui/render"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func cellFrame(x, y int, text string) kernel.Frame {
	return kernel.Frame{
		Rects: map[string]kernel.Rect{},
		Lines: []kernel.Line{{X: x, Y: y, Text: text}},
	}
}

func textPlacement(rect kernel.Rect, text string) Placement {
	return Placement{
		Rect:  rect,
		Lines: []render.Line{{X: 0, Y: 0, Text: text, Style: render.TokenDefault}},
	}
}

func TestCompositorZOrder(t *testing.T) {
	comp := NewCompositor(8, 2)
	base := cellFrame(0, 0, "program")
	program := base
	program.OverlayFrames = []kernel.Frame{cellFrame(0, 0, "overlay")}
	placements := []Placement{textPlacement(kernel.Rect{Width: 7, Height: 1}, "component")}
	notice := cellFrame(0, 0, "notice")
	core := cellFrame(0, 0, "core")

	steps := []struct {
		name       string
		program    kernel.Frame
		placements []Placement
		notice     *kernel.Frame
		core       *kernel.Frame
		want       string
	}{
		{"core wins", program, placements, &notice, &core, "c"},
		{"notice over overlay", program, placements, &notice, nil, "n"},
		{"overlay over component", program, placements, nil, nil, "o"},
		{"component covers program", base, placements, nil, nil, "c"},
	}
	for _, step := range steps {
		frame := comp.Compose(step.program, step.placements, step.notice, step.core)
		if got := frame.CellAt(0, 0).Text; got != step.want {
			t.Fatalf("%s: cell = %q, want %q", step.name, got, step.want)
		}
	}
	component := comp.Compose(base, placements, nil, nil)
	if got := component.CellAt(0, 0).Text; got != "c" || component.CellAt(1, 0).Text != "o" {
		t.Fatalf("component layer = %q%q, want component text", got, component.CellAt(1, 0).Text)
	}
}

func TestCompositorComponentInsideOverlaySurvives(t *testing.T) {
	comp := NewCompositor(10, 3)
	root := &kernel.Node{ID: "root", Flow: kernel.FlowStack, Children: []kernel.Node{
		{ID: "term", Pos: &kernel.Pos{X: 2, Y: 1},
			Size:    kernel.Size{Width: 6, Height: 2},
			Content: &kernel.Content{Self: "terminal:local:t"}},
		{ID: "hint", Pos: &kernel.Pos{X: 2, Y: 1},
			Size:    kernel.Size{Width: 4, Height: 1},
			Content: &kernel.Content{Text: "MASK"}},
	}}
	program := kernel.Layout(root, 10, 3)
	placement := textPlacement(kernel.Rect{X: 2, Y: 1, Width: 6, Height: 2}, "hello")
	frame := comp.Compose(program, []Placement{placement}, nil, nil)
	// The terminal's own pos overlay must not erase its component frame...
	if got := frame.CellAt(6, 1).Text; got != "o" {
		t.Fatalf("cell(6,1) = %q, want the component line to survive its overlay", got)
	}
	// ...while a sibling overlay declared later still paints above it.
	if got := frame.CellAt(2, 1).Text; got != "M" {
		t.Fatalf("cell(2,1) = %q, want the hint overlay on top", got)
	}
}

func TestCompositorComponentInsideNestedOverlay(t *testing.T) {
	comp := NewCompositor(12, 3)
	card := kernel.Node{ID: "card", Pos: &kernel.Pos{X: 0, Y: 0},
		Size: kernel.Size{Width: 12, Height: 3}, Flow: kernel.FlowStack}
	card.Children = []kernel.Node{
		{ID: "card-title", Pos: &kernel.Pos{X: 0, Y: 0},
			Size: kernel.Size{Width: 12, Height: 1}, Content: &kernel.Content{Text: "CARD"}},
		{ID: "card-term", Pos: &kernel.Pos{X: 1, Y: 1},
			Size:    kernel.Size{Width: 10, Height: 1},
			Content: &kernel.Content{Self: "terminal:local:t"}},
	}
	root := &kernel.Node{ID: "root", Flow: kernel.FlowStack, Children: []kernel.Node{card}}
	program := kernel.Layout(root, 12, 3)
	placement := textPlacement(kernel.Rect{X: 1, Y: 1, Width: 10, Height: 1}, "content")
	frame := comp.Compose(program, []Placement{placement}, nil, nil)
	if got := frame.CellAt(0, 0).Text; got != "C" {
		t.Fatalf("cell(0,0) = %q, want the card title line", got)
	}
	if got := frame.CellAt(1, 1).Text; got != "c" {
		t.Fatalf("cell(1,1) = %q, want the nested component inside the card", got)
	}
}

func TestCompositorPlacementFollowsTopmostOverlappingOverlay(t *testing.T) {
	comp := NewCompositor(12, 5)
	root := &kernel.Node{ID: "root", Flow: kernel.FlowStack, Children: []kernel.Node{
		{ID: "card-term", Pos: &kernel.Pos{X: 0, Y: 0},
			Size:    kernel.Size{Width: 12, Height: 5},
			Content: &kernel.Content{Self: "terminal:local:card"}},
		{ID: "float-term", Pos: &kernel.Pos{X: 2, Y: 1},
			Size:    kernel.Size{Width: 6, Height: 2},
			Content: &kernel.Content{Self: "terminal:local:float"}},
	}}
	program := kernel.Layout(root, 12, 5)
	placements := []Placement{
		textPlacement(kernel.Rect{X: 0, Y: 0, Width: 12, Height: 5}, "under"),
		textPlacement(kernel.Rect{X: 2, Y: 1, Width: 6, Height: 2}, "float"),
	}
	frame := comp.Compose(program, placements, nil, nil)
	// The later (topmost) overlapping overlay must own the float placement:
	// the earlier card-sized overlay blit would otherwise erase it, and the
	// float's own opaque fill would blank it with nothing to redraw.
	if got := frame.CellAt(2, 1).Text; got != "f" {
		t.Fatalf("cell(2,1) = %q, want the float component inside the card overlay", got)
	}
	if got := frame.CellAt(0, 0).Text; got != "u" {
		t.Fatalf("cell(0,0) = %q, want the card component outside the float", got)
	}
}

func TestCompositorClipsAndClearsPlacement(t *testing.T) {
	comp := NewCompositor(6, 1)
	base := cellFrame(0, 0, "abcd")
	frame := comp.Compose(base, []Placement{textPlacement(kernel.Rect{X: 2, Width: 2, Height: 1}, "WXYZ")}, nil, nil)
	if got := frame.CellAt(2, 0).Text; got != "W" {
		t.Fatalf("cell 2 = %q, want W", got)
	}
	if got := frame.CellAt(3, 0).Text; got != "X" {
		t.Fatalf("cell 3 = %q, want X", got)
	}

	// A placement that only paints its first cell still blanks the rest of
	// its rect: components are opaque.
	frame = comp.Compose(base, []Placement{textPlacement(kernel.Rect{X: 1, Width: 2, Height: 1}, "Q")}, nil, nil)
	if got := frame.CellAt(1, 0).Text; got != "Q" {
		t.Fatalf("cell 1 = %q, want Q", got)
	}
	if !frame.CellAt(2, 0).Blank() {
		t.Fatalf("cell 2 = %+v, want cleared by opaque placement", frame.CellAt(2, 0))
	}
}

func TestCompositorWideClusterAcrossFrames(t *testing.T) {
	comp := NewCompositor(8, 1)
	base := cellFrame(0, 0, "abcdef")
	frame := comp.Compose(base, []Placement{textPlacement(kernel.Rect{X: 1, Width: 4, Height: 1}, "你x")}, nil, nil)
	if cell := frame.CellAt(1, 0); cell.Text != "你" || cell.Width != 2 {
		t.Fatalf("wide head = %+v, want 你 width 2", cell)
	}
	if !frame.CellAt(2, 0).Continuation {
		t.Fatalf("wide continuation missing: %+v", frame.CellAt(2, 0))
	}
	if got := frame.CellAt(3, 0).Text; got != "x" {
		t.Fatalf("cell 3 = %q, want x", got)
	}

	// A wide grapheme that cannot fit at the destination edge is dropped
	// whole, and overwriting the head of a base wide cluster through
	// BlitFrame clears the orphan continuation.
	baseFrame := render.NewFrame(4, 1)
	baseFrame.Blit(render.Line{X: 0, Y: 0, Text: "abcd", Style: render.TokenDefault})
	head := render.NewFrame(2, 1)
	head.Blit(render.Line{X: 0, Y: 0, Text: "你", Style: render.TokenDefault})
	baseFrame.BlitFrame(head, 3, 0)
	if cell := baseFrame.CellAt(3, 0); !cell.Blank() {
		t.Fatalf("wide grapheme split at the edge: %+v", cell)
	}

	wideBase := render.NewFrame(4, 1)
	wideBase.Blit(render.Line{X: 0, Y: 0, Text: "你x", Style: render.TokenDefault})
	overlay := render.NewFrame(1, 1)
	overlay.Blit(render.Line{X: 0, Y: 0, Text: "y", Style: render.TokenDefault})
	wideBase.BlitFrame(overlay, 0, 0)
	if !wideBase.CellAt(1, 0).Blank() {
		t.Fatalf("orphan continuation survived: %+v", wideBase.CellAt(1, 0))
	}
	if got := wideBase.CellAt(2, 0).Text; got != "x" {
		t.Fatalf("cell 2 = %q, want x kept", got)
	}
}

func TestCompositorCursorPriority(t *testing.T) {
	comp := NewCompositor(8, 2)
	program := cellFrame(0, 0, "p")
	program.HasCursor = true
	program.CursorRect = kernel.Rect{X: 1, Y: 0, Width: 1, Height: 1}
	placement := Placement{
		Rect:          kernel.Rect{X: 4, Y: 1, Width: 3, Height: 1},
		Focused:       true,
		CursorVisible: true,
		CursorX:       1,
		CursorY:       0,
	}
	core := cellFrame(0, 0, "core")
	core.HasCursor = true
	core.CursorRect = kernel.Rect{X: 2, Y: 1, Width: 1, Height: 1}

	frame := comp.Compose(program, []Placement{placement}, nil, &core)
	if x, y, visible := frame.Cursor(); !visible || x != 2 || y != 1 {
		t.Fatalf("core cursor = (%d,%d,%v), want (2,1,true)", x, y, visible)
	}

	frame = comp.Compose(program, []Placement{placement}, nil, nil)
	if x, y, visible := frame.Cursor(); !visible || x != 1 || y != 0 {
		t.Fatalf("program cursor = (%d,%d,%v), want (1,0,true)", x, y, visible)
	}

	noCursor := cellFrame(0, 0, "p")
	frame = comp.Compose(noCursor, []Placement{placement}, nil, nil)
	if x, y, visible := frame.Cursor(); !visible || x != 5 || y != 1 {
		t.Fatalf("terminal cursor = (%d,%d,%v), want (5,1,true)", x, y, visible)
	}

	coreNoCursor := cellFrame(0, 0, "core")
	frame = comp.Compose(program, []Placement{placement}, nil, &coreNoCursor)
	if _, _, visible := frame.Cursor(); visible {
		t.Fatal("open core overlay without a cursor must hide the hardware cursor")
	}

	frame = comp.Compose(noCursor, nil, nil, nil)
	if _, _, visible := frame.Cursor(); visible {
		t.Fatal("no cursor source must keep the hardware cursor hidden")
	}
}

func TestSessionFrameBytesAreNotDuplicated(t *testing.T) {
	h := newHarness(t, Options{ViewID: "v", Cols: 24, Rows: 4})
	h.sendView(view(1, 1, nil, false, textBox("body", "hello")))

	first := h.s.FrameBytes(nil, nil)
	if len(first) == 0 || !strings.Contains(string(first), "hello") {
		t.Fatalf("first frame bytes = %q, want the view text", first)
	}
	if second := h.s.FrameBytes(nil, nil); second != nil {
		t.Fatalf("unchanged frame emitted %q, want no bytes", second)
	}

	notice := cellFrame(0, 3, "!")
	if third := h.s.FrameBytes(nil, &notice); len(third) == 0 {
		t.Fatal("notice change must emit bytes")
	}
	if again := h.s.FrameBytes(nil, &notice); again != nil {
		t.Fatalf("stable notice emitted again: %q", again)
	}
}

func TestSessionFrameBytesTrackCursorOnlyChanges(t *testing.T) {
	h := newHarness(t, Options{ViewID: "v", Cols: 24, Rows: 4})
	h.sendView(view(1, 1, nil, false, textBox("body", "hello")))

	placement := Placement{
		Rect:          kernel.Rect{X: 1, Y: 1, Width: 6, Height: 2},
		Lines:         []render.Line{{X: 0, Y: 0, Text: "term", Style: render.TokenDefault}},
		Focused:       true,
		CursorVisible: true,
		CursorX:       2,
		CursorY:       1,
	}
	if bytes := h.s.FrameBytes([]Placement{placement}, nil); len(bytes) == 0 {
		t.Fatal("initial component frame must emit bytes")
	}
	placement.CursorX = 3
	if bytes := h.s.FrameBytes([]Placement{placement}, nil); len(bytes) == 0 {
		t.Fatal("cursor move must emit bytes")
	}
	placement.CursorVisible = false
	if bytes := h.s.FrameBytes([]Placement{placement}, nil); len(bytes) == 0 {
		t.Fatal("cursor hide must emit bytes")
	}
	if bytes := h.s.FrameBytes([]Placement{placement}, nil); bytes != nil {
		t.Fatalf("stable hidden cursor emitted again: %q", bytes)
	}
}

func TestSessionComposeFrameCoreNeverCovered(t *testing.T) {
	h := newHarness(t, Options{ViewID: "v", Cols: 24, Rows: 4})
	h.sendView(view(1, 1, nil, false, textBox("body", "hello")))

	placement := textPlacement(kernel.Rect{Width: 8, Height: 2}, "terminal")
	core := cellFrame(0, 0, "confirm")
	h.s.OpenCoreOverlay(core)

	frame := h.s.ComposeFrame([]Placement{placement}, nil)
	if got := frame.CellAt(0, 0).Text; got != "c" {
		t.Fatalf("cell 0 = %q, want the core overlay on top", got)
	}
}

func TestSessionResizeReflowsAndRepaints(t *testing.T) {
	h := newHarness(t, Options{ViewID: "v", Cols: 24, Rows: 4})
	h.sendView(view(1, 1, nil, false, textBox("body", "hello")))
	if bytes := h.s.FrameBytes(nil, nil); len(bytes) == 0 {
		t.Fatal("initial frame must emit bytes")
	}
	h.s.Resize(12, 2)
	if cols, rows := h.s.compositor.Size(); cols != 12 || rows != 2 {
		t.Fatalf("compositor size = %d×%d, want 12×2", cols, rows)
	}
	if bytes := h.s.FrameBytes(nil, nil); len(bytes) == 0 {
		t.Fatal("resize must force a repaint")
	}
}

// TestSessionResizeEmitsResizeEvent pins the SIGWINCH contract: a real
// viewport change must reach the program as a resize event (PROTOCOL §3, the
// program is the only one who can reflow its boxes), and an unchanged
// viewport must not emit anything.
func TestSessionResizeEmitsResizeEvent(t *testing.T) {
	h := newHarness(t, Options{ViewID: "v", Cols: 24, Rows: 4})
	h.sendView(view(1, 1, nil, false, textBox("body", "hello")))

	h.s.Resize(12, 2)
	var got *pb.ResizeEvent
	for _, m := range h.drain() {
		if ev, ok := m.(*pb.Event); ok && ev.GetResize() != nil {
			got = ev.GetResize()
		}
	}
	if got == nil {
		t.Fatal("Session.Resize must deliver a resize event to the program")
	}
	if got.GetCols() != 12 || got.GetRows() != 2 {
		t.Fatalf("resize event = %dx%d, want 12x2", got.GetCols(), got.GetRows())
	}

	h.s.Resize(12, 2)
	if msgs := h.drain(); len(msgs) != 0 {
		t.Fatalf("unchanged viewport emitted %d frames, want none", len(msgs))
	}
}

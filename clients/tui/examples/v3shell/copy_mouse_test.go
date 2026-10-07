package main

import (
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk/app"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// copyMouse delivers one mouse event straight to the model and discards the
// command it returns. A copy drag arms the 100ms auto-scroll Tick, which must
// not be executed here (the test drives the steps explicitly below).
func copyMouse(m *model, ev *pb.MouseEvent) { _ = m.onMouse(ev) }

// runCopyEdgeStep runs exactly one auto-scroll step (the body of the
// CopyModeMouseAutoScrollMsg tick) and feeds every non-Tick message it produced
// back into the model, so the re-armed 100ms timer is observed but not awaited.
func runCopyEdgeStep(t *testing.T, m *model) {
	t.Helper()
	for _, msg := range app.RunCmd(m.applyCopyEdgeAutoScroll()) {
		if _, isTick := msg.(app.TickMsg); isTick {
			continue
		}
		if next := m.Update(msg); next != nil {
			runCmd(t, m, next)
		}
	}
}

// TestCopyMouseEdgeAutoScrollsOlder ports the legacy copy-drag auto-scroll
// (updateHistoryMouseScrollEdge + CopyModeMouseAutoScrollMsg): holding a marked
// selection against the top content edge steps the frozen window two rows older
// per 100ms tick, parks the cursor on the edge and lets the anchor ride so the
// selection grows. Moving the pointer into the interior disarms it.
func TestCopyMouseEdgeAutoScrollsOlder(t *testing.T) {
	m, fake := boundModel(t)
	window := []string{"l0", "l1", "l2", "l3", "l4", "l5", "l6", "l7", "l8", "l9"}
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		switch method {
		case "history.window":
			return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: window, Offset: 0}}
		case "terminal.scroll":
			// The host prepended two older rows and reports offset 2.
			older := append([]string{"o0", "o1"}, window...)
			return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: older, Offset: 2}}
		}
		return &pb.Response{Ok: true}
	}

	p := m.focusPane()
	runCmd(t, m, key(m, "ctrl-shift-c"))
	st := m.copyFor(p)
	if st == nil {
		t.Fatal("copy session was not opened")
	}
	r, ok := m.copyContentRect(p)
	if !ok {
		t.Fatal("no content rect")
	}

	// Anchor the selection on content row 3 with the copy-mode press (legacy
	// mouse-select intent: press places the cursor AND sets the mark).
	midY := r.y + 1 + 3
	copyMouse(m, &pb.MouseEvent{Action: "press", Node: p.id, X: int32(r.x + 2), Y: int32(midY)})
	if !st.marked || st.markRow != 3 {
		t.Fatalf("mark = %+v, want row 3", st)
	}

	// Drag to the top edge: the direction arms and one tick is scheduled.
	topEdgeY := r.y + 1
	copyMouse(m, &pb.MouseEvent{Action: "drag", Node: p.id, X: int32(r.x + 2), Y: int32(topEdgeY)})
	if m.copyDragDir != -1 || !m.copyDragTick {
		t.Fatalf("top edge did not arm older auto-scroll: dir=%d tick=%v", m.copyDragDir, m.copyDragTick)
	}
	if st.cursorRow != 0 {
		t.Fatalf("cursor not parked on the top edge: row=%d", st.cursorRow)
	}

	fake.calls = nil
	anchor := st.markRow
	// Fire the armed tick once: the view scrolls two rows older.
	runCopyEdgeStep(t, m)
	if !hasCall(fake.calls, "terminal.scroll") {
		t.Fatalf("edge tick did not scroll: %+v", fake.calls)
	}
	if st.offset != 2 {
		t.Fatalf("offset = %d, want 2 (two rows older)", st.offset)
	}
	if st.cursorRow != 0 {
		t.Fatalf("cursor left the top edge: row=%d", st.cursorRow)
	}
	// The anchor row shifts with the prepended rows, so the selection keeps
	// pointing at the same text and now reaches older content.
	if st.markRow <= anchor {
		t.Fatalf("selection did not grow older: mark %d -> %d", anchor, st.markRow)
	}
	if !m.copyDragTick {
		t.Fatal("auto-scroll did not re-arm after a step")
	}

	// Moving the pointer into the interior clears the direction and the
	// pending tick; a later step must not scroll.
	copyMouse(m, &pb.MouseEvent{Action: "drag", Node: p.id, X: int32(r.x + 2), Y: int32(r.y + 1 + 4)})
	if m.copyDragDir != 0 {
		t.Fatalf("interior kept auto-scroll armed: dir=%d", m.copyDragDir)
	}
	fake.calls = nil
	runCopyEdgeStep(t, m)
	if hasCall(fake.calls, "terminal.scroll") {
		t.Fatalf("interior step still scrolled: %+v", fake.calls)
	}
	copyMouse(m, &pb.MouseEvent{Action: "release", Node: p.id})
}

// TestCopyMouseEdgeRequiresMark pins the legacy historyMouseScrollAvailable
// gate: without a MARK the edge drag never arms the timer.
func TestCopyMouseEdgeRequiresMark(t *testing.T) {
	m, _ := boundModel(t)
	p := m.focusPane()
	st := &copyState{rows: []string{"a", "b", "c"}, viewRows: 3, placed: true}
	m.copyPanes[p.id] = st
	r, ok := m.copyContentRect(p)
	if !ok {
		t.Fatal("no content rect")
	}
	m.dragging = "copy:" + p.id
	copyMouse(m, &pb.MouseEvent{Action: "drag", Node: p.id, X: int32(r.x + 2), Y: int32(r.y + 1)})
	if m.copyDragDir != 0 || m.copyDragTick {
		t.Fatalf("edge armed without a mark: dir=%d tick=%v", m.copyDragDir, m.copyDragTick)
	}
}

// TestCopyMouseBottomEdgeScrollsNewerAndDisarmsAtBottom pins the newer half of
// the legacy edge auto-scroll: from a frozen viewport a held bottom-edge pointer
// scrolls newer (the offset shrinks), and it disarms once the viewport reaches
// the live bottom (historyMouseScrollAvailable's AtFrozenBottom rule) so it does
// not keep emitting clamped scroll requests.
func TestCopyMouseBottomEdgeScrollsNewerAndDisarmsAtBottom(t *testing.T) {
	m, fake := boundModel(t)
	p := m.focusPane()
	st := &copyState{rows: []string{"a", "b", "c", "d"}, viewRows: 4, cols: 80, offset: 2, marked: true, markRow: 1, cursorRow: 1, placed: true}
	m.copyPanes[p.id] = st
	r, ok := m.copyContentRect(p)
	if !ok {
		t.Fatal("no content rect")
	}
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "terminal.scroll" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: []string{"c", "d"}, Offset: 0}}
		}
		return &pb.Response{Ok: true}
	}
	// The press that started the drag; the session already carries the mark.
	m.dragging = "copy:" + p.id
	bottomEdgeY := r.y + r.h // one past the last content row is the bottom band
	copyMouse(m, &pb.MouseEvent{Action: "drag", Node: p.id, X: int32(r.x + 2), Y: int32(bottomEdgeY)})
	if m.copyDragDir != 1 || !m.copyDragTick {
		t.Fatalf("bottom edge did not arm newer auto-scroll: dir=%d tick=%v", m.copyDragDir, m.copyDragTick)
	}
	// Dragging to the bottom edge already parked the cursor on the last row,
	// so the first tick crosses the edge and scrolls newer by the frozen offset.
	fake.calls = nil
	runCopyEdgeStep(t, m)
	if !hasCall(fake.calls, "terminal.scroll") {
		t.Fatalf("bottom edge step did not scroll: %+v", fake.calls)
	}
	if st.offset != 0 {
		t.Fatalf("offset = %d, want 0 (back to live bottom)", st.offset)
	}
	// The edge re-armed for the next tick; now that the viewport is at the live
	// bottom the step cannot advance and disarms (legacy
	// historyMouseScrollAvailable) instead of looping.
	fake.calls = nil
	runCopyEdgeStep(t, m)
	if hasCall(fake.calls, "terminal.scroll") {
		t.Fatalf("newer edge still scrolled at the live bottom: %+v", fake.calls)
	}
	if m.copyDragDir != 0 || m.copyDragTick {
		t.Fatalf("newer edge stayed armed at the live bottom: dir=%d tick=%v", m.copyDragDir, m.copyDragTick)
	}
}

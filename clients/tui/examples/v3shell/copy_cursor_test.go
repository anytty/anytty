package main

import (
	"testing"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// terminalBoxWithCursor returns the terminal box the view declares for the
// given pane, or nil when it carries no program cursor.
func terminalBoxWithCursor(root *pb.Box, paneID string) *pb.Box {
	if root == nil {
		return nil
	}
	if root.GetId() == paneID {
		return root
	}
	for _, child := range root.GetChildren() {
		if found := terminalBoxWithCursor(child, paneID); found != nil {
			return found
		}
	}
	return nil
}

// TestCopySessionKeepsVisibleBlockCursor pins the GAP 2 parity rule the user
// reported: entering history/copy mode must not cancel the small white cursor.
// The v3shell declares the copy cursor on the terminal box
// (term.Cursor(st.cursorRow, st.cursorCol, "block")) with the SAME block shape
// the live PTY cursor uses, and rasterizing the frame parks a cursor at the
// absolute copy cell. Live mode declares no program cursor at all, so the host
// paints the single live PTY cursor and copy mode swaps to the single copy
// cursor instead of drawing two.
func TestCopySessionKeepsVisibleBlockCursor(t *testing.T) {
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "history.window" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{
				Rows: []string{"hello world", "short", "another"}, Offset: 0,
			}}
		}
		return &pb.Response{Ok: true}
	}

	// Live mode: the host owns the cursor from the PTY; the view declares none.
	p := m.focusPane()
	if box := terminalBoxWithCursor(m.View(), p.id); box != nil && box.GetCursor() != nil {
		t.Fatalf("live mode must not declare a program cursor: %+v", box.GetCursor())
	}

	// Enter copy mode and place the cursor on a deliberately short row beyond
	// its text (row 1 "short", col 7), i.e. over a blank column. Render once
	// first so the pane content width is known, otherwise the column clamps to
	// the short row's own width.
	runCmd(t, m, key(m, "ctrl-shift-c"))
	st := m.copyFor(p)
	if st == nil {
		t.Fatal("copy session was not opened")
	}
	m.View()
	st.cursorRow, st.cursorCol = 1, 7
	r, ok := m.copyContentRect(p)
	if !ok {
		t.Fatal("no content rect")
	}
	m.clampCopyCursorToViewport(st)
	if st.cursorCol != 7 {
		t.Fatalf("copy cursor col clamped to %d, want 7", st.cursorCol)
	}

	box := terminalBoxWithCursor(m.View(), p.id)
	if box == nil || box.GetCursor() == nil {
		t.Fatalf("copy mode must declare a visible cursor on the terminal box")
	}
	if got := box.GetCursor().GetShape(); got != "block" {
		t.Fatalf("copy cursor shape = %q, want block", got)
	}
	if got := int(box.GetCursor().GetRow()); got != st.cursorRow {
		t.Fatalf("copy cursor row = %d, want %d", got, st.cursorRow)
	}
	if got := int(box.GetCursor().GetCol()); got != st.cursorCol {
		t.Fatalf("copy cursor col = %d, want %d", got, st.cursorCol)
	}

	// Rasterize: the cursor lands at the absolute content cell (the content
	// rect origin plus the viewport cursor).
	lines, _ := m.rasterize(m.View())
	_ = lines
	if m.cursor == nil {
		t.Fatal("copy cursor missing from the rasterized frame")
	}
	wantX, wantY := r.x+st.cursorCol, r.y+st.cursorRow
	if m.cursor.x != wantX || m.cursor.y != wantY {
		t.Fatalf("rasterized copy cursor = %+v, want (%d,%d)", *m.cursor, wantX, wantY)
	}
}

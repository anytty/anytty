package render

import (
	"strings"
	"testing"
)

func TestFrameFullRepaint(t *testing.T) {
	f := NewFrame(4, 2)
	f.BlitLine(Line{X: 0, Y: 0, Text: "hi"})
	f.SetCursor(2, 0)
	want := HideCursor + ResetSGR +
		CursorPosition(0, 0) + ResetSGR + defaultSGR + "hi  " + ResetSGR +
		CursorPosition(0, 1) + ResetSGR + defaultSGR + "    " + ResetSGR +
		CursorPosition(2, 0) + ShowCursor
	if got := string(f.Bytes(nil)); got != want {
		t.Fatalf("full repaint mismatch\n got %q\nwant %q", got, want)
	}
	if got := string(f.FullBytes()); got != want {
		t.Fatalf("FullBytes mismatch\n got %q\nwant %q", got, want)
	}
}

func TestFrameRowStyles(t *testing.T) {
	f := NewFrame(3, 1)
	f.Blit(
		Line{X: 0, Y: 0, Text: "a", Style: TokenAccent},
		Line{X: 1, Y: 0, Text: "b", Style: TokenMuted},
	)
	want := HideCursor + ResetSGR +
		CursorPosition(0, 0) +
		ResetSGR + accentSGR + "a" +
		ResetSGR + mutedSGR + "b" +
		ResetSGR + defaultSGR + " " +
		ResetSGR
	if got := string(f.Bytes(nil)); got != want {
		t.Fatalf("styled row mismatch\n got %q\nwant %q", got, want)
	}
}

func TestFrameDiffSkipsIdenticalFrame(t *testing.T) {
	f := NewFrame(4, 2)
	f.BlitLine(Line{X: 0, Y: 0, Text: "same"})
	f.SetCursor(0, 0)
	if got := f.Bytes(f); got != nil {
		t.Fatalf("identical frame emitted %q, want nil", got)
	}
	g := NewFrame(4, 2)
	g.BlitLine(Line{X: 0, Y: 0, Text: "same"})
	g.SetCursor(0, 0)
	if got := f.Bytes(g); got != nil {
		t.Fatalf("equal frame emitted %q, want nil", got)
	}
	if !f.Equal(g) {
		t.Fatalf("Equal = false for identical frames")
	}
}

func TestFrameDiffOnlyChangedRows(t *testing.T) {
	prev := NewFrame(4, 2)
	prev.BlitLine(Line{X: 0, Y: 0, Text: "ab"})
	curr := NewFrame(4, 2)
	curr.BlitLine(Line{X: 0, Y: 0, Text: "ab"})
	curr.BlitLine(Line{X: 0, Y: 1, Text: "cd"})
	want := HideCursor +
		CursorPosition(0, 1) + ResetSGR + defaultSGR + "cd  " + ResetSGR
	if got := string(curr.Bytes(prev)); got != want {
		t.Fatalf("row diff mismatch\n got %q\nwant %q", got, want)
	}
	if strings.Contains(string(curr.Bytes(prev)), "ab") {
		t.Fatalf("row diff repainted the unchanged row")
	}
}

func TestFrameDiffCursorOnly(t *testing.T) {
	prev := NewFrame(2, 1)
	prev.BlitLine(Line{X: 0, Y: 0, Text: "ab"})
	prev.SetCursor(0, 0)
	curr := NewFrame(2, 1)
	curr.BlitLine(Line{X: 0, Y: 0, Text: "ab"})
	curr.SetCursor(1, 0)
	want := CursorPosition(1, 0) + ShowCursor
	if got := string(curr.Bytes(prev)); got != want {
		t.Fatalf("cursor diff mismatch\n got %q\nwant %q", got, want)
	}
	hidden := NewFrame(2, 1)
	hidden.BlitLine(Line{X: 0, Y: 0, Text: "ab"})
	if got := string(hidden.Bytes(prev)); got != HideCursor {
		t.Fatalf("hide cursor diff = %q, want %q", got, HideCursor)
	}
}

func TestFrameDiffSizeChangeIsFullRepaint(t *testing.T) {
	prev := NewFrame(2, 1)
	curr := NewFrame(3, 1)
	got := string(curr.Bytes(prev))
	if !strings.HasPrefix(got, HideCursor+ResetSGR) {
		t.Fatalf("size change should repaint fully, got %q", got)
	}
}

func TestFrameBlitClipsText(t *testing.T) {
	f := NewFrame(3, 1)
	f.BlitLine(Line{X: 0, Y: 0, Text: "hello"})
	if got := f.CellAt(0, 0).Text; got != "h" {
		t.Fatalf("cell 0 = %q", got)
	}
	if got := f.CellAt(2, 0).Text; got != "l" {
		t.Fatalf("cell 2 = %q", got)
	}
	if !f.CellAt(3, 0).Blank() {
		t.Fatalf("cell outside the grid should be blank")
	}
}

func TestFrameBlitWideGrapheme(t *testing.T) {
	f := NewFrame(6, 1)
	f.BlitLine(Line{X: 0, Y: 0, Text: "你好"})
	if got := f.CellAt(0, 0); got.Text != "你" || got.Width != 2 || got.Continuation {
		t.Fatalf("cell 0 = %+v, want wide head 你", got)
	}
	if got := f.CellAt(1, 0); !got.Continuation || got.Text != "" {
		t.Fatalf("cell 1 = %+v, want continuation", got)
	}
	if got := f.CellAt(2, 0); got.Text != "好" || got.Width != 2 {
		t.Fatalf("cell 2 = %+v, want wide head 好", got)
	}
	if got := f.CellAt(3, 0); !got.Continuation {
		t.Fatalf("cell 3 = %+v, want continuation", got)
	}
}

func TestFrameBlitEmojiClusterOccupiesTwoCells(t *testing.T) {
	f := NewFrame(4, 1)
	f.BlitLine(Line{X: 0, Y: 0, Text: "👨‍👩‍👧‍👦x"})
	if got := f.CellAt(0, 0); got.Text != "👨‍👩‍👧‍👦" || got.Width != 2 {
		t.Fatalf("family cell = %+v, want whole cluster width 2", got)
	}
	if !f.CellAt(1, 0).Continuation {
		t.Fatalf("family continuation missing")
	}
	if got := f.CellAt(2, 0); got.Text != "x" || got.Width != 1 {
		t.Fatalf("cell after family = %+v, want x", got)
	}
}

func TestFrameBlitNeverSplitsWideAtRightEdge(t *testing.T) {
	f := NewFrame(3, 1)
	f.BlitLine(Line{X: 2, Y: 0, Text: "你"})
	if !f.CellAt(2, 0).Blank() {
		t.Fatalf("wide grapheme must not be drawn half-off the grid")
	}
	g := NewFrame(3, 1)
	g.BlitLine(Line{X: 1, Y: 0, Text: "你a"})
	if got := g.CellAt(1, 0); got.Text != "你" {
		t.Fatalf("cell 1 = %+v, want 你", got)
	}
	if !g.CellAt(3, 0).Blank() {
		t.Fatalf("trailing a must be clipped")
	}
}

func TestFrameBlitOverwriteWideClearsPartner(t *testing.T) {
	head := NewFrame(4, 1)
	head.BlitLine(Line{X: 0, Y: 0, Text: "你"})
	head.BlitLine(Line{X: 0, Y: 0, Text: "a"})
	if !head.CellAt(1, 0).Blank() {
		t.Fatalf("overwriting the head must clear the continuation: %+v", head.CellAt(1, 0))
	}

	tail := NewFrame(4, 1)
	tail.BlitLine(Line{X: 0, Y: 0, Text: "你"})
	tail.BlitLine(Line{X: 1, Y: 0, Text: "x"})
	if !tail.CellAt(0, 0).Blank() {
		t.Fatalf("overwriting the continuation must clear the head: %+v", tail.CellAt(0, 0))
	}
	if got := tail.CellAt(1, 0).Text; got != "x" {
		t.Fatalf("cell 1 = %q, want x", got)
	}
}

func TestFrameBlitCombiningMark(t *testing.T) {
	f := NewFrame(2, 1)
	f.BlitLine(Line{X: 0, Y: 0, Text: "e\u0301"})
	if got := f.CellAt(0, 0); got.Text != "e\u0301" || got.Width != 1 {
		t.Fatalf("combining cell = %+v, want cluster e+U+0301 width 1", got)
	}
	if !f.CellAt(1, 0).Blank() {
		t.Fatalf("combining mark must not consume a cell: %+v", f.CellAt(1, 0))
	}

	lead := NewFrame(2, 1)
	lead.BlitLine(Line{X: 0, Y: 0, Text: "\u0301"})
	if !lead.CellAt(0, 0).Blank() {
		t.Fatalf("lone combining mark should not create a cell")
	}
}

func TestFrameBlitNegativeXClips(t *testing.T) {
	f := NewFrame(2, 1)
	f.BlitLine(Line{X: -1, Y: 0, Text: "ab"})
	if got := f.CellAt(0, 0).Text; got != "b" {
		t.Fatalf("cell 0 = %q, want b", got)
	}
	if !f.CellAt(1, 0).Blank() {
		t.Fatalf("cell 1 should be blank")
	}
}

func TestFrameBlitOutOfRangeRowIsIgnored(t *testing.T) {
	f := NewFrame(2, 1)
	f.BlitLine(Line{X: 0, Y: 3, Text: "x"})
	f.BlitLine(Line{X: 0, Y: -1, Text: "y"})
	if got := string(f.Bytes(nil)); strings.Contains(got, "x") || strings.Contains(got, "y") {
		t.Fatalf("out-of-range blit leaked into output: %q", got)
	}
}

func TestFrameClear(t *testing.T) {
	f := NewFrame(2, 1)
	f.BlitLine(Line{X: 0, Y: 0, Text: "ab", Style: TokenAccent})
	f.SetCursor(0, 0)
	f.Clear()
	if !f.CellAt(0, 0).Blank() {
		t.Fatalf("Clear left content: %+v", f.CellAt(0, 0))
	}
	if _, _, visible := f.Cursor(); visible {
		t.Fatalf("Clear left the cursor visible")
	}
}

func TestFrameBlitFrameIsOpaqueAndOffset(t *testing.T) {
	dst := NewFrame(6, 2)
	dst.BlitLine(Line{X: 0, Y: 0, Text: "abcdef", Style: TokenDefault})
	src := NewFrame(3, 2)
	src.BlitLine(Line{X: 1, Y: 0, Text: "XY", Style: TokenAccent})
	dst.BlitFrame(src, 2, 0)

	if !dst.CellAt(2, 0).Blank() {
		t.Fatalf("cell 2 = %+v, want the blank scrubbed by the opaque frame", dst.CellAt(2, 0))
	}
	if got := dst.CellAt(3, 0).Text; got != "X" {
		t.Fatalf("cell 3 = %q, want X", got)
	}
	if got := dst.CellAt(4, 0).Text; got != "Y" {
		t.Fatalf("cell 4 = %q, want Y", got)
	}
	if got := dst.CellAt(5, 0).Text; got != "f" {
		t.Fatalf("cell 5 = %q, want program cell outside the frame", got)
	}
	if got := dst.CellAt(4, 0).Style; got != TokenAccent {
		t.Fatalf("cell style = %q, want accent", got)
	}
}

func TestFrameBlitFrameRebuildsWideClusters(t *testing.T) {
	src := NewFrame(3, 1)
	src.BlitLine(Line{X: 0, Y: 0, Text: "你x", Style: TokenDefault})
	dst := NewFrame(5, 1)
	dst.BlitFrame(src, 1, 0)

	if cell := dst.CellAt(1, 0); cell.Text != "你" || cell.Width != 2 {
		t.Fatalf("wide head = %+v", cell)
	}
	if !dst.CellAt(2, 0).Continuation {
		t.Fatalf("continuation missing: %+v", dst.CellAt(2, 0))
	}
	if got := dst.CellAt(3, 0).Text; got != "x" {
		t.Fatalf("cell 3 = %q, want x", got)
	}

	edge := NewFrame(4, 1)
	head := NewFrame(1, 1)
	head.BlitLine(Line{X: 0, Y: 0, Text: "你", Style: TokenDefault})
	edge.BlitFrame(head, 3, 0)
	if !edge.CellAt(3, 0).Blank() {
		t.Fatalf("wide grapheme split at the right edge: %+v", edge.CellAt(3, 0))
	}
}

func TestFrameBlitFrameLeftClipClearsWideHalf(t *testing.T) {
	dst := NewFrame(4, 1)
	dst.BlitLine(Line{X: 0, Y: 0, Text: "abcd", Style: TokenDefault})
	src := NewFrame(3, 1)
	src.BlitLine(Line{X: 0, Y: 0, Text: "你x", Style: TokenDefault})
	dst.BlitFrame(src, -1, 0)

	if !dst.CellAt(0, 0).Blank() {
		t.Fatalf("left-clipped wide half must blank the cell: %+v", dst.CellAt(0, 0))
	}
	if got := dst.CellAt(1, 0).Text; got != "x" {
		t.Fatalf("cell 1 = %q, want x", got)
	}
	if got := dst.CellAt(2, 0).Text; got != "c" {
		t.Fatalf("cell 2 = %q, want untouched c", got)
	}
}

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

func TestFrameResetClearsForReuse(t *testing.T) {
	f := NewFrame(4, 1)
	f.BlitLine(Line{X: 0, Y: 0, Text: "text"})
	f.SetCursor(2, 0)
	storage := &f.cells[0]
	f.Reset(4, 1)
	if &f.cells[0] != storage {
		t.Fatal("same-size reset should reuse cell storage")
	}
	if got := f.CellAt(0, 0); !got.Blank() {
		t.Fatalf("reset cell = %+v, want blank", got)
	}
	if _, _, visible := f.Cursor(); visible {
		t.Fatal("reset should hide the cursor")
	}
}

func TestFrameSetCursorClipsToViewport(t *testing.T) {
	f := NewFrame(4, 3)
	f.SetCursor(99, -5)
	if x, y, visible := f.Cursor(); !visible || x != 3 || y != 0 {
		t.Fatalf("clipped cursor = (%d,%d,%v), want (3,0,true)", x, y, visible)
	}
	empty := NewFrame(0, 0)
	empty.SetCursor(0, 0)
	if _, _, visible := empty.Cursor(); visible {
		t.Fatal("empty framebuffer must not expose a cursor")
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

func TestFrameDiffCursorShape(t *testing.T) {
	prev := NewFrame(2, 1)
	prev.SetCursor(0, 0)
	curr := NewFrame(2, 1)
	curr.SetCursor(0, 0)
	curr.SetCursorShape("bar")
	got := string(curr.Bytes(prev))
	want := CursorShapeBar + CursorPosition(0, 0) + ShowCursor
	if got != want {
		t.Fatalf("cursor shape diff = %q, want %q", got, want)
	}
}

func TestFrameDiffUsesScrollRegionForShift(t *testing.T) {
	prev := NewFrame(4, 4)
	prev.BlitLine(Line{Y: 0, Text: "one"})
	prev.BlitLine(Line{Y: 1, Text: "two"})
	prev.BlitLine(Line{Y: 2, Text: "three"})
	prev.BlitLine(Line{Y: 3, Text: "four"})
	curr := NewFrame(4, 4)
	curr.BlitLine(Line{Y: 0, Text: "two"})
	curr.BlitLine(Line{Y: 1, Text: "three"})
	curr.BlitLine(Line{Y: 2, Text: "four"})
	curr.BlitLine(Line{Y: 3, Text: "five"})
	got := string(curr.Bytes(prev))
	for _, want := range []string{ScrollRegion(0, 3), CursorPosition(0, 3), ScrollUp, ResetScrollRegion, "five"} {
		if !strings.Contains(got, want) {
			t.Fatalf("scroll diff %q missing %q", got, want)
		}
	}
	if strings.Contains(got, "one") || strings.Contains(got, "two") || strings.Contains(got, "three") {
		t.Fatalf("scroll diff rewrote old rows: %q", got)
	}
}

func TestFrameDiffUsesScrollRegionForReverseShift(t *testing.T) {
	prev := NewFrame(3, 4)
	prev.BlitLine(Line{Y: 0, Text: "one"})
	prev.BlitLine(Line{Y: 1, Text: "two"})
	prev.BlitLine(Line{Y: 2, Text: "three"})
	prev.BlitLine(Line{Y: 3, Text: "four"})
	curr := NewFrame(3, 4)
	curr.BlitLine(Line{Y: 0, Text: "zero"})
	curr.BlitLine(Line{Y: 1, Text: "one"})
	curr.BlitLine(Line{Y: 2, Text: "two"})
	curr.BlitLine(Line{Y: 3, Text: "three"})
	got := string(curr.Bytes(prev))
	for _, want := range []string{ScrollRegion(0, 3), CursorPosition(0, 0), ScrollDown, ResetScrollRegion, "zer"} {
		if !strings.Contains(got, want) {
			t.Fatalf("reverse scroll diff %q missing %q", got, want)
		}
	}
}

func TestFrameDiffUsesParameterizedScrollRegionForPageShift(t *testing.T) {
	prev := NewFrame(4, 6)
	for y, text := range []string{"zero", "one", "two", "three", "four", "five"} {
		prev.BlitLine(Line{Y: y, Text: text})
	}
	curr := NewFrame(4, 6)
	for y, text := range []string{"two", "three", "four", "five", "six", "seven"} {
		curr.BlitLine(Line{Y: y, Text: text})
	}
	got := string(curr.BytesWithSynchronizedScrollRegions(prev, []ScrollRect{{X: 0, Y: 0, Width: 4, Height: 6}}))
	for _, want := range []string{ScrollRegion(0, 5), ScrollUpN(2), ResetScrollRegion, "seve"} {
		if !strings.Contains(got, want) {
			t.Fatalf("page scroll diff %q missing %q", got, want)
		}
	}
}

func TestFrameDiffSynchronizesPhysicalScroll(t *testing.T) {
	prev := NewFrame(4, 4)
	prev.BlitLine(Line{Y: 0, Text: "one"})
	prev.BlitLine(Line{Y: 1, Text: "two"})
	prev.BlitLine(Line{Y: 2, Text: "three"})
	prev.BlitLine(Line{Y: 3, Text: "four"})
	curr := NewFrame(4, 4)
	curr.BlitLine(Line{Y: 0, Text: "two"})
	curr.BlitLine(Line{Y: 1, Text: "three"})
	curr.BlitLine(Line{Y: 2, Text: "four"})
	curr.BlitLine(Line{Y: 3, Text: "five"})
	got := string(curr.BytesWithSynchronizedScroll(prev))
	begin := strings.Index(got, BeginSynchronizedOutput)
	end := strings.Index(got, EndSynchronizedOutput)
	if begin < 0 || end <= begin {
		t.Fatalf("synchronized scroll = %q", got)
	}
	if plain := string(curr.Bytes(prev)); strings.Contains(plain, BeginSynchronizedOutput) {
		t.Fatalf("plain scroll unexpectedly synchronized: %q", plain)
	}
}

func TestFrameDiffScrollsOnlyPanelContentRect(t *testing.T) {
	prev := NewFrame(12, 5)
	for y, text := range []string{"abcd", "efgh", "ijkl"} {
		prev.BlitLine(Line{X: 2, Y: y + 1, Text: text})
	}
	prev.BlitLine(Line{X: 0, Y: 0, Text: "header"})
	prev.BlitLine(Line{X: 8, Y: 1, Text: "side"})

	curr := NewFrame(12, 5)
	for y, text := range []string{"efgh", "ijkl", "mnop"} {
		curr.BlitLine(Line{X: 2, Y: y + 1, Text: text})
	}
	curr.BlitLine(Line{X: 0, Y: 0, Text: "header"})
	curr.BlitLine(Line{X: 8, Y: 1, Text: "side"})

	got := string(curr.BytesWithSynchronizedScrollRegions(prev, []ScrollRect{{X: 2, Y: 1, Width: 4, Height: 3}}))
	if !strings.Contains(got, BeginSynchronizedOutput) || !strings.Contains(got, EndSynchronizedOutput) {
		t.Fatalf("panel scroll = %q, want synchronized update", got)
	}
	if strings.Contains(got, ScrollRegion(1, 3)) {
		t.Fatalf("partial panel must not set a full-width scroll region: %q", got)
	}
	if strings.Contains(got, "header") || strings.Contains(got, "side") {
		t.Fatalf("panel scroll rewrote unchanged sibling content: %q", got)
	}
	for _, want := range []string{"efgh", "ijkl", "mnop"} {
		if !strings.Contains(got, want) {
			t.Fatalf("panel scroll = %q, missing %q", got, want)
		}
	}
}

func TestFrameDiffDoesNotTreatRepeatedRowsAsPanelScroll(t *testing.T) {
	// A full-screen child often leaves several identical muted rows around
	// its content. When one row changes at the bottom, matching the repeated
	// overlap must not make the host emit a real CSI scroll: that would move
	// the child viewport and then repaint it, producing a visible jump.
	prev := NewFrame(4, 4)
	for y := 0; y < 4; y++ {
		prev.BlitLine(Line{Y: y, Text: "    ", Style: TokenMuted})
	}
	curr := NewFrame(4, 4)
	for y := 0; y < 3; y++ {
		curr.BlitLine(Line{Y: y, Text: "    ", Style: TokenMuted})
	}
	curr.BlitLine(Line{Y: 3, Text: "new!", Style: TokenMuted})

	got := string(curr.BytesWithSynchronizedScrollRegions(prev, []ScrollRect{{X: 0, Y: 0, Width: 4, Height: 4}}))
	if strings.Contains(got, ScrollRegion(0, 3)) || strings.Contains(got, ScrollUp) || strings.Contains(got, ScrollDown) {
		t.Fatalf("repeated rows were mistaken for a panel scroll: %q", got)
	}
	if !strings.Contains(got, "new!") {
		t.Fatalf("ordinary row diff omitted changed row: %q", got)
	}
}

func TestFrameDiffScrollsPanelByPageWithoutRepaintingSiblings(t *testing.T) {
	prev := NewFrame(12, 7)
	for y, text := range []string{"zero", "one", "two", "three", "four"} {
		prev.BlitLine(Line{X: 2, Y: y + 1, Text: text})
	}
	prev.BlitLine(Line{X: 0, Y: 0, Text: "header"})
	prev.BlitLine(Line{X: 8, Y: 2, Text: "side"})

	curr := NewFrame(12, 7)
	for y, text := range []string{"two", "three", "four", "five", "six"} {
		curr.BlitLine(Line{X: 2, Y: y + 1, Text: text})
	}
	curr.BlitLine(Line{X: 0, Y: 0, Text: "header"})
	curr.BlitLine(Line{X: 8, Y: 2, Text: "side"})

	got := string(curr.BytesWithSynchronizedScrollRegions(prev, []ScrollRect{{X: 2, Y: 1, Width: 5, Height: 5}}))
	if strings.Contains(got, ScrollRegion(1, 5)) {
		t.Fatalf("partial page scroll must not set a full-width scroll region: %q", got)
	}
	if strings.Contains(got, "header") || strings.Contains(got, "side") {
		t.Fatalf("page scroll rewrote unchanged sibling content: %q", got)
	}
	for _, want := range []string{"two", "three", "four", "five", "six"} {
		if !strings.Contains(got, want) {
			t.Fatalf("page scroll = %q, missing %q", got, want)
		}
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

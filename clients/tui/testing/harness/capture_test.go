package harness

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/render"
	"github.com/anytty/anytty/clients/tui/render/ansi"
)

func styledRow(text string, token render.Token) []ansi.Cell {
	row := make([]ansi.Cell, 0, len(text))
	for _, r := range text {
		row = append(row, ansi.Cell{Text: string(r), Width: 1, Style: token})
	}
	return row
}

func TestRenderRawEmitsSeparateAttributeSequences(t *testing.T) {
	focus := render.ANSIToken("1;38;2;240;171;252")
	row := append(styledRow("▎term-1", focus), ansi.Cell{Text: " ", Width: 1}, ansi.Cell{Text: "x", Width: 1})
	lines := RenderRaw(ansi.Screen{Lines: [][]ansi.Cell{row}})
	if len(lines) != 1 {
		t.Fatalf("lines = %d, want 1", len(lines))
	}
	want := "\x1b[1m\x1b[38;2;240;171;252m▎term-1"
	if !strings.Contains(lines[0], want) {
		t.Fatalf("raw row %q does not contain %q", lines[0], want)
	}
}

func TestRenderRawColorAndPlainText(t *testing.T) {
	colors := render.ANSIToken("38;2;125;211;252")
	row := []ansi.Cell{{Text: "a", Width: 1, Style: colors}, {Text: "b", Width: 1}}
	lines := RenderRaw(ansi.Screen{Lines: [][]ansi.Cell{row}})
	if got := lines[0]; got != "\x1b[38;2;125;211;252ma\x1b[0mb" {
		t.Fatalf("raw row = %q", got)
	}
}

func TestRenderTextTrimsTrailingBlanks(t *testing.T) {
	row := []ansi.Cell{{Text: "h", Width: 1}, {Text: "i", Width: 1}, {}, {}}
	lines := RenderText(ansi.Screen{Lines: [][]ansi.Cell{row}})
	if len(lines) != 1 || lines[0] != "hi" {
		t.Fatalf("text lines = %#v, want [hi]", lines)
	}
}

func TestDecodeOSC52(t *testing.T) {
	if got, ok := decodeOSC52("c;aGVsbG8="); !ok || got != "hello" {
		t.Fatalf("decodeOSC52 = %q, %v", got, ok)
	}
	if _, ok := decodeOSC52("garbage"); ok {
		t.Fatal("decodeOSC52 accepted a payload without a semicolon")
	}
}

// TestScanOSC52SplitAcrossChunks feeds the sequence one byte at a time: the
// scanner must keep the partial sequence and decode it once complete.
func TestScanOSC52SplitAcrossChunks(t *testing.T) {
	session := &Session{}
	sequence := []byte("\x1b]52;c;aGVsbG8=\x07")
	for _, b := range sequence {
		session.scanOSC52Locked([]byte{b})
	}
	if got, ok := session.Clipboard(); !ok || got != "hello" {
		t.Fatalf("clipboard = %q, %v, want hello", got, ok)
	}
}

// TestScanOSC52KeepsLatest ensures a later copy replaces an earlier one, like
// tmux's single paste buffer.
func TestScanOSC52KeepsLatest(t *testing.T) {
	session := &Session{}
	session.scanOSC52Locked([]byte("\x1b]52;c;b25l\x07\x1b]52;c;dHdv\x07"))
	if got, _ := session.Clipboard(); got != "two" {
		t.Fatalf("clipboard = %q, want two", got)
	}
}

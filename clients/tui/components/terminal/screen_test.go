package terminal

import (
	"testing"

	"github.com/anytty/anytty/clients/tui/render"
)

func TestScreenFromTextGraphemes(t *testing.T) {
	screen := ScreenFromText([]string{"👨‍👩‍👧‍👦x", "e\u0301"}, render.TokenAccent)
	if screen.Rows() != 2 {
		t.Fatalf("Rows = %d, want 2", screen.Rows())
	}
	row := screen.Line(0)
	if len(row) != 2 {
		t.Fatalf("row 0 cells = %+v, want 2 cells", row)
	}
	if row[0].Text != "👨‍👩‍👧‍👦" || row[0].Width != 2 || row[0].Style != render.TokenAccent {
		t.Fatalf("cell 0 = %+v, want family cluster width 2 accent", row[0])
	}
	if row[1].Text != "x" || row[1].Width != 1 {
		t.Fatalf("cell 1 = %+v, want x", row[1])
	}
	combining := screen.Line(1)
	if len(combining) != 1 || combining[0].Text != "e\u0301" || combining[0].Width != 1 {
		t.Fatalf("combining row = %+v, want one width-1 cluster", combining)
	}
}

func TestScreenFromTextSanitizesControls(t *testing.T) {
	screen := ScreenFromText([]string{"a\tb"}, "")
	row := screen.Line(0)
	if len(row) != 3 {
		t.Fatalf("row = %+v, want tab replaced by a space", row)
	}
	if row[1].Text != " " {
		t.Fatalf("cell 1 = %+v, want space", row[1])
	}
}

func TestScreenTextLinesRoundTrip(t *testing.T) {
	lines := []string{"hi", "", "你好😀"}
	screen := ScreenFromText(lines, render.TokenDefault)
	got := screen.TextLines()
	if len(got) != len(lines) {
		t.Fatalf("TextLines = %q, want %q", got, lines)
	}
	for i := range lines {
		if got[i] != lines[i] {
			t.Fatalf("TextLines[%d] = %q, want %q", i, got[i], lines[i])
		}
	}
}

func TestScreenLineOutOfRange(t *testing.T) {
	screen := ScreenFromText([]string{"x"}, render.TokenDefault)
	if got := screen.Line(-1); got != nil {
		t.Fatalf("Line(-1) = %+v, want nil", got)
	}
	if got := screen.Line(1); got != nil {
		t.Fatalf("Line(1) = %+v, want nil", got)
	}
	if got := (Screen{}).TextLines(); got != nil {
		t.Fatalf("empty Screen.TextLines = %+v, want nil", got)
	}
}

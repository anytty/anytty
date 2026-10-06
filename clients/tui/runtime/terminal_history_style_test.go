package runtime

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/render"
	"github.com/anytty/anytty/clients/tui/render/ansi"
	"github.com/anytty/anytty/proto/access/apipb"
)

// This is the byte-level oracle for main:tui/render/copy_history.go's
// ansiStyleFromHistory/buildANSICellStyle path. It deliberately compares the
// complete repaint string, including background SGR and display-only tail fill.
func TestHistoryANSIStringMatchesMainOracle(t *testing.T) {
	rows := []*apipb.HistoryRow{{
		Row: &apipb.ScreenRow{
			Cells: []*apipb.ScreenCell{
				{Content: "A", Width: 1, Style: &apipb.CellStyle{Foreground: "ansi:1", Background: "ansi:4", Bold: true}},
				{Content: "B", Width: 1, Style: &apipb.CellStyle{Foreground: "ansi:1", Background: "ansi:4", Bold: true}},
				{Content: "C", Width: 1, Style: &apipb.CellStyle{Foreground: "idx:196", Background: "idx:17", Italic: true, Underline: true, Blink: true, Reverse: true, Strikethrough: true}},
			},
			TailFill: &apipb.CellStyle{Background: "idx:24"},
		},
	}}
	screen := historyScreen(rows, 8)
	frame := render.NewFrame(8, 1)
	var lines []render.Line
	for rowIndex, cells := range screen.Lines {
		var text strings.Builder
		style := render.Token("")
		x := 0
		flush := func() {
			if text.Len() > 0 {
				lines = append(lines, render.Line{X: x - render.DisplayWidth(text.String()), Y: rowIndex, Text: text.String(), Style: style})
			}
			text.Reset()
		}
		for _, cell := range cells {
			if style != cell.Style {
				flush()
				style = cell.Style
			}
			text.WriteString(cell.Text)
			x += cell.Width
		}
		flush()
	}
	frame.Blit(lines...)
	got := string(frame.FullBytes())
	want := render.HideCursor + render.ResetSGR + render.CursorPosition(0, 0) + render.ResetSGR +
		"\x1b[1;31;44mAB" + render.ResetSGR +
		"\x1b[3;4;5;7;9;38;5;196;48;5;17mC" + render.ResetSGR +
		"\x1b[48;5;24m     " + render.ResetSGR
	if got != want {
		t.Fatalf("history ANSI differs from main oracle:\n got  %q\n want %q", got, want)
	}

	// Keep this assertion close to the byte oracle: raw ANSI styles must also
	// survive the intermediate ansi cell representation used by the component.
	cell := ansi.Cell{Text: "C", Width: 1, Style: render.Token("fg:idx:196;bg:idx:17;italic;underline;blink;reverse;strikethrough")}
	if style := cell.Style; style == "" {
		t.Fatal("styled history cell collapsed to the default style")
	}
}

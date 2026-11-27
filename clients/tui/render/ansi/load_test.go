package ansi

import (
	"testing"

	"github.com/anytty/anytty/clients/tui/render"
	"github.com/anytty/anytty/internal/screenmodel"
	"github.com/anytty/anytty/proto/access/apipb"
)

func modelScreen(cols, rows int, lines ...[]screenmodel.Cell) *screenmodel.Screen {
	screen := &screenmodel.Screen{
		Cols:          cols,
		Rows:          rows,
		CursorVisible: true,
		CursorShape:   "block",
		Modes:         &apipb.TerminalModes{},
	}
	for y := 0; y < rows; y++ {
		row := make([]screenmodel.Cell, cols)
		if y < len(lines) {
			copy(row, lines[y])
		}
		screen.Lines = append(screen.Lines, row)
	}
	return screen
}

func textRow(text string) []screenmodel.Cell {
	cells := make([]screenmodel.Cell, 0, len(text))
	for _, r := range text {
		cells = append(cells, screenmodel.Cell{Text: string(r), Width: 1})
	}
	return cells
}

func TestLoadScreenFullAndPartial(t *testing.T) {
	parser := New(4, 2)
	screen := modelScreen(4, 2, textRow("ab"), textRow("cd"))
	if !parser.LoadScreen(screen, nil) {
		t.Fatal("full load rejected")
	}
	got := parser.Screen()
	if RowText(got.Row(0)) != "ab" || RowText(got.Row(1)) != "cd" {
		t.Fatalf("grid = %q / %q", RowText(got.Row(0)), RowText(got.Row(1)))
	}

	// 只替换第 1 行：第 0 行必须保持原样。
	screen.Lines[1] = textRow("XY")
	if !parser.LoadScreen(screen, []int{1}) {
		t.Fatal("partial load rejected")
	}
	got = parser.Screen()
	if RowText(got.Row(0)) != "ab" || RowText(got.Row(1)) != "XY" {
		t.Fatalf("partial grid = %q / %q", RowText(got.Row(0)), RowText(got.Row(1)))
	}
}

func TestLoadScreenStylesAndBlanks(t *testing.T) {
	parser := New(3, 1)
	screen := modelScreen(3, 1, []screenmodel.Cell{
		{Text: "x", Width: 1, Style: &apipb.CellStyle{Bold: true, Foreground: "ansi:2"}},
		{Text: "", Style: &apipb.CellStyle{Background: "idx:4"}},
		{},
	})
	if !parser.LoadScreen(screen, nil) {
		t.Fatal("load rejected")
	}
	row := parser.Screen().Row(0)
	if row[0].Style != render.ANSIToken("1;32") {
		t.Fatalf("style = %q, want %q", row[0].Style, render.ANSIToken("1;32"))
	}
	if row[1].Text != " " || row[1].Style != render.ANSIToken("48;5;4") {
		t.Fatalf("styled blank = %#v", row[1])
	}
	if !row[2].Blank() {
		t.Fatalf("unstyled blank = %#v, want zero cell", row[2])
	}
}

func TestLoadScreenAlternateScreenSwitch(t *testing.T) {
	parser := New(3, 1)
	if !parser.LoadScreen(modelScreen(3, 1, textRow("main")), nil) {
		t.Fatal("main load rejected")
	}
	alt := modelScreen(3, 1, textRow("alt"))
	alt.Modes = &apipb.TerminalModes{AlternateScreen: true}
	if !parser.LoadScreen(alt, []int{0}) {
		t.Fatal("alt load rejected")
	}
	if !parser.Modes().AltScreen {
		t.Fatal("alt mode not applied")
	}
	// 回到主屏内容还在。
	main := modelScreen(3, 1, textRow("new"))
	main.Modes = &apipb.TerminalModes{AlternateScreen: false}
	if !parser.LoadScreen(main, nil) {
		t.Fatal("main restore rejected")
	}
	if parser.Modes().AltScreen {
		t.Fatal("alt mode not cleared")
	}
}

// TestLoadScreenClipsToHostGrid 覆盖“daemon 屏比面板小/大”的投影：解析器
// 网格尺寸保持不变（光标和滚动以面板为准），只把内容裁进交集区域。
func TestLoadScreenClipsToHostGrid(t *testing.T) {
	parser := New(3, 2)
	if !parser.LoadScreen(modelScreen(5, 2, textRow("hello"), textRow("world")), nil) {
		t.Fatal("clipped load rejected")
	}
	screen := parser.Screen()
	if len(screen.Lines) != 2 || len(screen.Lines[0]) != 3 {
		t.Fatalf("grid = %dx%d, want 3x2", len(screen.Lines[0]), len(screen.Lines))
	}
	if got := RowText(screen.Row(0)); got != "hel" {
		t.Fatalf("row 0 = %q, want first 3 columns", got)
	}

	// daemon 屏更小：交集以外保持空白，网格尺寸不变。
	small := New(6, 3)
	if !small.LoadScreen(modelScreen(2, 1, textRow("ab")), nil) {
		t.Fatal("small load rejected")
	}
	got := small.Screen()
	if len(got.Lines) != 3 || len(got.Lines[0]) != 6 {
		t.Fatalf("grid shrank to %dx%d", len(got.Lines[0]), len(got.Lines))
	}
	if RowText(got.Row(0)) != "ab" || RowText(got.Row(1)) != "" {
		t.Fatalf("rows = %q / %q", RowText(got.Row(0)), RowText(got.Row(1)))
	}
}

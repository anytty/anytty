package screenmodel

import (
	"testing"

	"github.com/anytty/anytty/proto/access/apipb"
)

func rowReplace(index int, cells ...*apipb.ScreenCell) *apipb.ScreenRowReplace {
	return &apipb.ScreenRowReplace{RowIndex: int32(index), Row: &apipb.ScreenRow{Cells: cells}}
}

func textCell(content string) *apipb.ScreenCell {
	return &apipb.ScreenCell{Content: content, Width: int32(len(content))}
}

func screenDelta(full bool, cols, rows int, replacements []*apipb.ScreenRowReplace, copies []*apipb.ScreenRowCopy) *apipb.NativeScreenResult {
	return &apipb.NativeScreenResult{
		Size:            &apipb.TerminalSize{Cols: uint32(cols), Rows: uint32(rows)},
		FullReplace:     full,
		RowReplacements: replacements,
		RowCopies:       copies,
		Cursor:          &apipb.TerminalCursor{Row: 0, Col: 0, Visible: true},
		Modes:           &apipb.TerminalModes{},
	}
}

func TestApplyFullReplaceBuildsDenseScreen(t *testing.T) {
	delta := screenDelta(true, 4, 2, []*apipb.ScreenRowReplace{
		rowReplace(0, textCell("ab")),
		rowReplace(1, textCell("cde")),
	}, nil)
	screen, changed := Apply(nil, delta)
	if changed != nil {
		t.Fatalf("first frame changed = %v, want nil (full repaint)", changed)
	}
	if screen.Rows != 2 || screen.Cols != 4 {
		t.Fatalf("size = %dx%d, want 4x2", screen.Cols, screen.Rows)
	}
	if got := lineText(screen.Line(0)); got != "ab  " {
		t.Fatalf("row 0 = %q, want %q", got, "ab  ")
	}
	if got := lineText(screen.Line(1)); got != "cde " {
		t.Fatalf("row 1 = %q, want %q", got, "cde ")
	}
}

func TestApplyRowCopyReadsFromPreviousFrame(t *testing.T) {
	first, _ := Apply(nil, screenDelta(true, 3, 3, []*apipb.ScreenRowReplace{
		rowReplace(0, textCell("aaa")),
		rowReplace(1, textCell("bbb")),
		rowReplace(2, textCell("ccc")),
	}, nil))
	next, changed := Apply(first, screenDelta(false, 3, 3, []*apipb.ScreenRowReplace{
		rowReplace(2, textCell("zzz")),
	}, []*apipb.ScreenRowCopy{{SourceRow: 1, DestinationRow: 0, Count: 1}}))
	if got := lineText(next.Line(0)); got != "bbb" {
		t.Fatalf("copied row = %q, want %q", got, "bbb")
	}
	if got := lineText(next.Line(2)); got != "zzz" {
		t.Fatalf("replaced row = %q, want %q", got, "zzz")
	}
	if len(changed) != 2 || changed[0] != 0 || changed[1] != 2 {
		t.Fatalf("changed = %v, want [0 2]", changed)
	}
	// 未变化的行应复用同一个切片，避免整屏拷贝。
	if &next.Lines[1][0] != &first.Lines[1][0] {
		t.Fatal("unchanged row was copied instead of reused")
	}
}

func TestApplyAlternateScreenSwitchForcesFull(t *testing.T) {
	first, _ := Apply(nil, screenDelta(true, 2, 1, []*apipb.ScreenRowReplace{
		rowReplace(0, textCell("ok")),
	}, nil))
	delta := screenDelta(false, 2, 1, []*apipb.ScreenRowReplace{
		rowReplace(0, textCell("al")),
	}, nil)
	delta.Modes = &apipb.TerminalModes{AlternateScreen: true}
	next, changed := Apply(first, delta)
	if changed != nil {
		t.Fatalf("alt-screen switch changed = %v, want nil", changed)
	}
	if !next.AlternateScreen() {
		t.Fatal("modes not applied")
	}
}

func TestApplyWideCellsAndTailFill(t *testing.T) {
	wide := &apipb.ScreenCell{Content: "你", Width: 2}
	delta := screenDelta(true, 4, 1, []*apipb.ScreenRowReplace{{
		RowIndex: 0,
		Row: &apipb.ScreenRow{
			Cells:    []*apipb.ScreenCell{wide, textCell("x")},
			TailFill: &apipb.CellStyle{Background: "ansi:4"},
		},
	}}, nil)
	screen, _ := Apply(nil, delta)
	row := screen.Line(0)
	if len(row) != 4 {
		t.Fatalf("row len = %d, want 4", len(row))
	}
	if row[0].Text != "你" || row[0].Width != 2 || !row[1].Continuation {
		t.Fatalf("wide cell not expanded: %#v", row[:2])
	}
	if row[2].Text != "x" {
		t.Fatalf("cell 2 = %#v", row[2])
	}
	if row[3].Text != " " || row[3].Style.GetBackground() != "ansi:4" {
		t.Fatalf("tail fill = %#v, want styled blank", row[3])
	}
}

func TestApplyEmptyContentFillsSpaces(t *testing.T) {
	delta := screenDelta(true, 3, 1, []*apipb.ScreenRowReplace{
		rowReplace(0, &apipb.ScreenCell{Width: 2, Style: &apipb.CellStyle{Bold: true}}),
	}, nil)
	screen, _ := Apply(nil, delta)
	row := screen.Line(0)
	if row[0].Text != " " || row[1].Text != " " || !row[0].Style.GetBold() {
		t.Fatalf("blank expansion = %#v", row[:2])
	}
}

func lineText(line []Cell) string {
	out := make([]rune, 0, len(line))
	for _, cell := range line {
		if cell.Continuation {
			continue
		}
		if cell.Text == "" {
			out = append(out, ' ')
			continue
		}
		out = append(out, []rune(cell.Text)...)
	}
	return string(out)
}

// TestApplySplitsRunCompressedCells 覆盖服务端的 run-compressed 行：一个
// ScreenCell 可能是同一样式的一整段文本，模型必须按 grapheme 拆开，否则光标
// 列（CursorOffset）会被整段宽度带偏。
func TestApplySplitsRunCompressedCells(t *testing.T) {
	delta := screenDelta(true, 6, 1, []*apipb.ScreenRowReplace{
		rowReplace(0,
			&apipb.ScreenCell{Content: "ab", Width: 2},
			&apipb.ScreenCell{Content: "你", Width: 2},
			&apipb.ScreenCell{Content: " ", Width: 1},
		),
	}, nil)
	screen, _ := Apply(nil, delta)
	row := screen.Line(0)
	if len(row) != 6 {
		t.Fatalf("row len = %d, want 6", len(row))
	}
	if row[0].Text != "a" || row[1].Text != "b" {
		t.Fatalf("run not split: %#v", row[:2])
	}
	if row[2].Text != "你" || row[2].Width != 2 || !row[3].Continuation {
		t.Fatalf("wide cluster = %#v", row[2:4])
	}
	if row[4].Text != " " {
		t.Fatalf("cell 4 = %#v", row[4])
	}
}

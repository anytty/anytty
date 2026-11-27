package binding

import (
	"testing"

	"github.com/anytty/anytty/internal/screenmodel"
	"github.com/anytty/anytty/proto/access/apipb"
	"google.golang.org/protobuf/proto"
)

func screenTextCells(text string) []*apipb.ScreenCell {
	cells := make([]*apipb.ScreenCell, 0, len(text))
	for _, r := range text {
		cells = append(cells, &apipb.ScreenCell{Content: string(r), Width: 1})
	}
	return cells
}

func flattenScreenForTest(t *testing.T, engine *Engine, delta *apipb.NativeScreenResult) *apipb.NativeScreenResult {
	t.Helper()
	raw, err := proto.Marshal(delta)
	if err != nil {
		t.Fatalf("marshal delta: %v", err)
	}
	payload, err := engine.flattenScreenFrame(1, screenmodel.EncodeFrame(raw))
	if err != nil {
		t.Fatalf("flatten frame: %v", err)
	}
	out := &apipb.NativeScreenResult{}
	if err := proto.Unmarshal(payload, out); err != nil {
		t.Fatalf("unmarshal flattened frame: %v", err)
	}
	return out
}

func rowTextOf(row *apipb.ScreenRow) string {
	var out []rune
	for _, cell := range row.GetCells() {
		out = append(out, []rune(cell.GetContent())...)
	}
	return string(out)
}

// TestFlattenScreenFrameAppliesDeltas 覆盖引擎侧的“一份模型”：
// delta 先在 screenmodel 里应用，再拍平成只有完整行替换的帧交给 UI，
// row_copies 不会出现在发往 Dart 的帧里。
func TestFlattenScreenFrameAppliesDeltas(t *testing.T) {
	engine := &Engine{streams: map[uint64]*streamRecord{1: {}}}
	seed := &apipb.NativeScreenResult{
		Size:        &apipb.TerminalSize{Cols: 4, Rows: 3},
		FullReplace: true,
		LiveRevision: 1,
		RowReplacements: []*apipb.ScreenRowReplace{
			{RowIndex: 0, Row: &apipb.ScreenRow{Cells: screenTextCells("aa")}},
			{RowIndex: 1, Row: &apipb.ScreenRow{Cells: screenTextCells("bb")}},
			{RowIndex: 2, Row: &apipb.ScreenRow{Cells: screenTextCells("cc")}},
		},
		Cursor: &apipb.TerminalCursor{Row: 2, Col: 2, Visible: true},
		Modes:  &apipb.TerminalModes{},
	}
	full := flattenScreenForTest(t, engine, seed)
	if !full.GetFullReplace() || len(full.GetRowReplacements()) != 3 {
		t.Fatalf("seed frame = %#v", full)
	}

	delta := &apipb.NativeScreenResult{
		Size:         &apipb.TerminalSize{Cols: 4, Rows: 3},
		LiveRevision: 2,
		BaseRevision: 1,
		RowCopies:    []*apipb.ScreenRowCopy{{SourceRow: 0, DestinationRow: 1, Count: 1}},
		RowReplacements: []*apipb.ScreenRowReplace{
			{RowIndex: 0, Row: &apipb.ScreenRow{Cells: screenTextCells("zz")}},
		},
		Cursor: &apipb.TerminalCursor{Row: 1, Col: 1, Visible: true},
		Modes:  &apipb.TerminalModes{},
	}
	flattened := flattenScreenForTest(t, engine, delta)
	if flattened.GetFullReplace() {
		t.Fatal("delta flattened into a full replace")
	}
	if len(flattened.GetRowCopies()) != 0 {
		t.Fatalf("flattened frame still carries row copies: %#v", flattened.GetRowCopies())
	}
	rows := map[int]*apipb.ScreenRowReplace{}
	for _, row := range flattened.GetRowReplacements() {
		rows[int(row.GetRowIndex())] = row
	}
	if got := rowTextOf(rows[0].GetRow()); got != "zz" {
		t.Fatalf("row 0 = %q, want zz", got)
	}
	// 行复制必须已经展开成目标行的完整内容。
	if got := rowTextOf(rows[1].GetRow()); got != "aa" {
		t.Fatalf("row 1 = %q, want aa (copied from row 0)", got)
	}
	if _, ok := rows[2]; ok {
		t.Fatal("unchanged row 2 was sent again")
	}
}

// TestFlattenScreenFrameKeepsTailFill 验证尾部同色空白折叠回 tail_fill。
func TestFlattenScreenFrameKeepsTailFill(t *testing.T) {
	engine := &Engine{streams: map[uint64]*streamRecord{1: {}}}
	delta := &apipb.NativeScreenResult{
		Size:        &apipb.TerminalSize{Cols: 4, Rows: 1},
		FullReplace: true,
		LiveRevision: 1,
		RowReplacements: []*apipb.ScreenRowReplace{{
			RowIndex: 0,
			Row: &apipb.ScreenRow{
				Cells:    []*apipb.ScreenCell{{Content: "x", Width: 1}},
				TailFill: &apipb.CellStyle{Background: "idx:4"},
			},
		}},
		Cursor: &apipb.TerminalCursor{Visible: true},
		Modes:  &apipb.TerminalModes{},
	}
	out := flattenScreenForTest(t, engine, delta)
	row := out.GetRowReplacements()[0].GetRow()
	if row.GetTailFill().GetBackground() != "idx:4" {
		t.Fatalf("tail fill lost: %#v", row.GetTailFill())
	}
	if len(row.GetCells()) != 1 {
		t.Fatalf("cells = %d, want only the non-blank head cell", len(row.GetCells()))
	}
}

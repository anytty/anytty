package endpoint

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/proto/access/apipb"
)

func seedScreen(rows ...string) *apipb.NativeScreenResult {
	replacements := make([]*apipb.ScreenRowReplace, 0, len(rows))
	for index, line := range rows {
		replacements = append(replacements, &apipb.ScreenRowReplace{
			RowIndex: int32(index),
			Row:      &apipb.ScreenRow{Cells: []*apipb.ScreenCell{{Content: line, Width: int32(len(line))}}},
		})
	}
	return &apipb.NativeScreenResult{
		Size:            &apipb.TerminalSize{Cols: 10, Rows: uint32(len(rows))},
		LiveRevision:    1,
		FullReplace:     true,
		RowReplacements: replacements,
	}
}

func deltaScreen(revision uint64, base uint64, rows []*apipb.ScreenRowReplace, copies []*apipb.ScreenRowCopy) *apipb.NativeScreenResult {
	return &apipb.NativeScreenResult{
		Size:            &apipb.TerminalSize{Cols: 10, Rows: 3},
		LiveRevision:    revision,
		BaseRevision:    base,
		RowReplacements: rows,
		RowCopies:       copies,
	}
}

func TestScreenDeltaRendererEmitsOnlyChangedRows(t *testing.T) {
	renderer := newScreenDeltaRenderer(seedScreen("aa", "bb", "cc"))
	if len(renderer.rowANSI) != 3 || !strings.Contains(renderer.rowANSI[0], "aa") {
		t.Fatalf("seed cache = %#v", renderer.rowANSI)
	}

	out := string(renderer.render(deltaScreen(2, 1, []*apipb.ScreenRowReplace{{
		RowIndex: 1,
		Row:      &apipb.ScreenRow{Cells: []*apipb.ScreenCell{{Content: "XY", Width: 2}}},
	}}, nil)))
	if !strings.Contains(out, "\x1b[2;1H") || !strings.Contains(out, "XY") {
		t.Fatalf("delta output = %q, want row 1 repaint", out)
	}
	if strings.Contains(out, "\x1b[1;1H") || strings.Contains(out, "\x1b[3;1H") {
		t.Fatalf("delta repainted unchanged rows: %q", out)
	}
	if !strings.Contains(renderer.rowANSI[1], "XY") {
		t.Fatalf("cache row 1 = %q, want XY", renderer.rowANSI[1])
	}
}

func TestScreenDeltaRendererReplaysRowCopiesLocally(t *testing.T) {
	renderer := newScreenDeltaRenderer(seedScreen("aa", "bb", "cc"))
	out := string(renderer.render(deltaScreen(2, 1, nil, []*apipb.ScreenRowCopy{{
		SourceRow: 0, DestinationRow: 2, Count: 1,
	}})))
	if !strings.Contains(out, "\x1b[3;1H") || !strings.Contains(out, "aa") {
		t.Fatalf("copy output = %q, want row 2 painted from row 0", out)
	}
	if !strings.Contains(renderer.rowANSI[2], "aa") {
		t.Fatalf("cache row 2 = %q, want aa", renderer.rowANSI[2])
	}
}

func TestScreenDeltaRendererFullReplaceOnSizeChange(t *testing.T) {
	renderer := newScreenDeltaRenderer(seedScreen("aa", "bb"))
	wider := seedScreen("dd", "ee")
	wider.Size = &apipb.TerminalSize{Cols: 20, Rows: 2}
	wider.FullReplace = false
	wider.LiveRevision = 5
	out := string(renderer.render(wider))
	if !strings.Contains(out, "\x1b[2J") {
		t.Fatalf("size change should repaint the full screen, got %q", out)
	}
	if renderer.cols != 20 {
		t.Fatalf("renderer cols = %d, want 20", renderer.cols)
	}
}

func TestScreenDeltaRendererCursorOnlyFrame(t *testing.T) {
	renderer := newScreenDeltaRenderer(seedScreen("aa", "bb"))
	screen := deltaScreen(2, 1, nil, nil)
	screen.Cursor = &apipb.TerminalCursor{Row: 1, Col: 2, Visible: true}
	out := string(renderer.render(screen))
	if !strings.Contains(out, "\x1b[2;3H") {
		t.Fatalf("cursor frame = %q, want cursor position", out)
	}
	// 同一光标位置重复出现时不再重发。
	again := string(renderer.render(deltaScreen(3, 2, nil, nil)))
	if again == "" || strings.Contains(again, "\x1b[2;3H") {
		t.Fatalf("repeated cursor frame = %q, want no duplicate cursor move", again)
	}
}

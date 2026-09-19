package live

import (
	"fmt"
	"strings"
	"testing"

	vterm "github.com/anytty/anytty/vterm/vterm"
)

func TestSurfaceSizeValid(t *testing.T) {
	if !(SurfaceSize{Cols: 80, Rows: 24}).Valid() {
		t.Fatal("expected positive size to be valid")
	}
	if (SurfaceSize{Cols: 0, Rows: 24}).Valid() {
		t.Fatal("expected zero cols to be invalid")
	}
}

func TestSurfaceTrackKeepsScreenSizeCursorAndStyle(t *testing.T) {
	surface := NewSurfaceTrack(SurfaceSize{Cols: 10, Rows: 2})
	surface.Write("one\rT\x1b[31mR\x1b[0m")
	snapshot := surface.Snapshot()

	if snapshot.Size != (SurfaceSize{Cols: 10, Rows: 2}) || len(snapshot.Screen.Cells) != 2 {
		t.Fatalf("expected size-bound screen snapshot, got %#v", snapshot)
	}
	if got := vtermRowText(snapshot.Screen.Cells[0]); got[:3] != "TRe" {
		t.Fatalf("expected CR overwrite to stay in live grid, got %q", got)
	}
	if snapshot.Screen.Cells[0][1].Style.FG != "ansi:1" {
		t.Fatalf("expected ANSI style in live cell matrix, got %#v", snapshot.Screen.Cells[0][1])
	}
	if !snapshot.Cursor.Visible || snapshot.Cursor.Row != 0 || snapshot.Cursor.Col != 2 {
		t.Fatalf("expected live cursor from vterm, got %#v", snapshot.Cursor)
	}

	surface.Resize(SurfaceSize{Cols: 5, Rows: 1})
	snapshot = surface.Snapshot()
	if snapshot.Size != (SurfaceSize{Cols: 5, Rows: 1}) || len(snapshot.Screen.Cells) != 1 {
		t.Fatalf("unexpected resized snapshot %#v", snapshot)
	}
}

func TestSurfaceTrackRestartPreservingScreenMapsCursorToAppendRow(t *testing.T) {
	surface := NewSurfaceTrack(SurfaceSize{Cols: 20, Rows: 4})
	surface.Write("before\n")

	surface.ResetForRestartPreservingScreen()
	snapshot := surface.Snapshot()
	if !snapshot.Cursor.Visible || snapshot.Cursor.Row != 1 || snapshot.Cursor.Col != 0 {
		t.Fatalf("restart should expose cursor at preserved-tail append row, got %#v", snapshot.Cursor)
	}
	if got := strings.Join(surface.Rows(), "\n"); !strings.Contains(got, "before") {
		t.Fatalf("restart should preserve visible live tail, got %q", got)
	}

	surface.Write("$ ")
	snapshot = surface.Snapshot()
	if !snapshot.Cursor.Visible || snapshot.Cursor.Row != 1 || snapshot.Cursor.Col != 2 {
		t.Fatalf("new process output should advance the real surface cursor, got %#v", snapshot.Cursor)
	}
	if got := strings.Join(surface.Rows(), "\n"); !strings.Contains(got, "before") || !strings.Contains(got, "$") {
		t.Fatalf("restart should keep old tail and new prompt in live surface, got %q", got)
	}
}

func TestSurfaceTrackLargeWriteKeepsLatestScreen(t *testing.T) {
	surface := NewSurfaceTrack(SurfaceSize{Cols: 80, Rows: 4})
	var out strings.Builder
	for i := 0; out.Len() <= 64*1024; i++ {
		fmt.Fprintf(&out, "%06d stress payload payload payload payload payload\n", i)
	}
	out.WriteString("999999 stress latest-tail")

	surface.Write(out.String())
	rows := surface.Rows()

	if !strings.Contains(strings.Join(rows, "\n"), "999999 stress latest-tail") {
		t.Fatalf("expected large live write to keep latest tail, got %#v", rows)
	}
}

func TestSurfaceTrackWriteResultOnlyCarriesLiveRawSegments(t *testing.T) {
	surface := NewSurfaceTrack(SurfaceSize{Cols: 20, Rows: 3})
	result := surface.WriteWithResult("one\r\ntwo\r\nlatest")
	if got := strings.Join(surface.Rows(), "\n"); !strings.Contains(got, "latest") {
		t.Fatalf("live screen write should update latest screen, got %q", got)
	}
	if len(result.Segments) != 1 || result.Segments[0].Raw == "" || len(result.Segments[0].AltScreenExitFrame) != 0 {
		t.Fatalf("live write result should carry only raw live segment data, got %#v", result)
	}
}

func TestSurfaceTrackWriteResultComposesRowCopiesAndReplacements(t *testing.T) {
	surface := NewSurfaceTrack(SurfaceSize{Cols: 12, Rows: 3})
	surface.Write("\x1b[1;1Hone\x1b[2;1Htwo\x1b[3;1Hthree")
	result := surface.WriteWithResult("\r\nfour")

	if result.FullReplace {
		t.Fatalf("ordinary scroll should remain incremental: %#v", result)
	}
	if len(result.RowCopies) != 1 || result.RowCopies[0] != (SurfaceRowCopy{SourceRow: 1, DestinationRow: 0, Count: 2}) {
		t.Fatalf("expected exact scroll row copy: %#v", result.RowCopies)
	}
	if len(result.ChangedRows) != 1 || result.ChangedRows[0] != 2 {
		t.Fatalf("only the rewritten bottom row should be replaced: %#v", result.ChangedRows)
	}
}

func BenchmarkSurfaceTrackFastSGRStressWrite(b *testing.B) {
	output := benchmarkSurfaceFastSGROutput(2048)
	b.ReportAllocs()
	b.SetBytes(int64(len(output)))

	for i := 0; i < b.N; i++ {
		surface := NewSurfaceTrack(SurfaceSize{Cols: 120, Rows: 36})
		surface.Write(output)
	}
}

func benchmarkSurfaceFastSGROutput(lines int) string {
	var builder strings.Builder
	for i := 0; i < lines; i++ {
		fmt.Fprintf(&builder, "\x1b[3%dm%06d live payload payload payload payload payload payload payload\x1b[0m\n", i%8, i)
	}
	builder.WriteString("999999 stress latest-tail")
	return builder.String()
}

func TestSurfaceTrackRestoresPrimaryOnAltScreenExitByDefault(t *testing.T) {
	surface := NewSurfaceTrack(SurfaceSize{Cols: 20, Rows: 3})
	surface.Write("primary")
	result := surface.WriteWithResult("\x1b[?1049h\x1b[2Jalt-final\x1b[?1049l")

	snapshot := surface.Snapshot()
	if snapshot.Modes.AlternateScreen || snapshot.Screen.IsAlternateScreen {
		t.Fatalf("expected alt exit to restore primary live screen, got modes=%#v screen=%#v", snapshot.Modes, snapshot.Screen)
	}
	if got := strings.Join(surface.Rows(), "\n"); strings.Contains(got, "alt-final") || !strings.Contains(got, "primary") {
		t.Fatalf("expected default alt exit to drop final frame and restore primary, got %q", got)
	}
	if len(result.Segments) != 1 || len(result.Segments[0].AltScreenExitFrame) == 0 {
		t.Fatalf("expected default alt exit to expose semantic current frame without replaying it to live, got %#v", result)
	}
}

func TestSurfaceTrackPreservesStyledAltScreenFrameOnExit(t *testing.T) {
	surface := NewSurfaceTrackWithOptions(SurfaceSize{Cols: 24, Rows: 4}, SurfaceTrackOptions{
		PreserveAltScreenFrameOnExit: true,
	})
	surface.Write("primary")
	result := surface.WriteWithResult("\x1b[?1049h\x1b[2J\x1b[31;44mALT\x1b[0m\x1b[2;1H\x1b[42mBAR   \x1b[0m\x1b[?1049l")

	snapshot := surface.Snapshot()
	if snapshot.Modes.AlternateScreen || snapshot.Screen.IsAlternateScreen {
		t.Fatalf("expected preserved styled frame to become primary live screen, got modes=%#v screen=%#v", snapshot.Modes, snapshot.Screen)
	}
	altCell, ok := findCell(snapshot.Screen.Cells, "A")
	if !ok {
		t.Fatalf("expected preserved alt frame text, got %#v", surface.Rows())
	}
	if altCell.Style.FG != "ansi:1" || altCell.Style.BG != "ansi:4" {
		t.Fatalf("expected styled alt cell to keep red-on-blue, got %#v", altCell)
	}
	if !hasStyledBlankInRowContaining(snapshot.Screen.Cells, "BAR", "ansi:2") {
		t.Fatalf("expected styled trailing blanks to survive alt exit replay, got %#v", snapshot.Screen.Cells)
	}
	if len(result.Segments) != 1 || len(result.Segments[0].AltScreenExitFrame) == 0 {
		t.Fatalf("expected one captured alt exit frame, got %#v", result)
	}
	capturedAltCell, ok := findCell(result.Segments[0].AltScreenExitFrame, "A")
	if !ok || capturedAltCell.Style.FG != "ansi:1" || capturedAltCell.Style.BG != "ansi:4" {
		t.Fatalf("expected captured frame to keep styled cells, got cell=%#v ok=%v", capturedAltCell, ok)
	}
}

func TestSurfaceTrackPreservedAltFrameIsPresentationOnly(t *testing.T) {
	surface := NewSurfaceTrackWithOptions(SurfaceSize{Cols: 24, Rows: 4}, SurfaceTrackOptions{
		CaptureLineHistory:           true,
		PreserveAltScreenFrameOnExit: true,
	})
	surface.Write("canonical-primary")
	surface.Write("\x1b[?1049h\x1b[2Jpresentation-alt\x1b[?1049l")

	if got := strings.Join(surface.Rows(), "\n"); !strings.Contains(got, "presentation-alt") {
		t.Fatalf("live presentation lost preserved alt frame: %q", got)
	}
	canonical := surface.LineHistoryScreenSnapshot()
	var canonicalText strings.Builder
	for _, row := range canonical.Rows {
		for _, cell := range row.Cells {
			canonicalText.WriteString(cell.Content)
		}
		canonicalText.WriteByte('\n')
	}
	if got := canonicalText.String(); strings.Contains(got, "presentation-alt") || !strings.Contains(got, "canonical-primary") {
		t.Fatalf("presentation overlay polluted canonical history snapshot: %q", got)
	}

	surface.Write("next")
	if got := strings.Join(surface.Rows(), "\n"); strings.Contains(got, "presentation-alt") || !strings.Contains(got, "next") {
		t.Fatalf("real PTY output did not replace presentation overlay: %q", got)
	}
}

func TestSurfaceTrackPreservesAltScreenFrameStyledBlankLayoutRows(t *testing.T) {
	surface := NewSurfaceTrackWithOptions(SurfaceSize{Cols: 18, Rows: 5}, SurfaceTrackOptions{
		PreserveAltScreenFrameOnExit: true,
	})
	surface.Write("primary")
	surface.Write("\x1b[?1049h\x1b[2J\x1b[45m      \x1b[0m\x1b[2;1Hcontent\x1b[?1049l")

	snapshot := surface.Snapshot()
	if !hasStyledBlankInRowBeforeMarker(snapshot.Screen.Cells, "content", "ansi:5") {
		t.Fatalf("expected styled blank layout row before preserved alt content, got %#v", snapshot.Screen.Cells)
	}
	if got := strings.Join(surface.Rows(), "\n"); !strings.Contains(got, "primary") {
		t.Fatalf("expected trimmed default blank rows not to evict primary tail, got %q", got)
	}
}

func TestSurfaceTrackAltScreenFramePreserveCanBeEnabledByEnv(t *testing.T) {
	t.Setenv(preserveAltScreenOnExitEnv, "1")
	surface := NewSurfaceTrack(SurfaceSize{Cols: 20, Rows: 3})
	surface.Write("primary")
	result := surface.WriteWithResult("\x1b[?1049h\x1b[2Jalt-final\x1b[?1049l")

	if got := strings.Join(surface.Rows(), "\n"); !strings.Contains(got, "alt-final") || !strings.Contains(got, "primary") {
		t.Fatalf("expected enabled preserve policy to append alt final frame after restored primary, got %q", got)
	}
	if surface.Snapshot().Modes.AlternateScreen {
		t.Fatal("expected enabled preserve policy to still leave alternate screen mode")
	}
	if len(result.Segments) != 1 || len(result.Segments[0].AltScreenExitFrame) == 0 {
		t.Fatalf("expected captured frame after enabled preserve policy, got %#v", result)
	}
}

func TestSurfaceTrackWriteResultKeepsRawOrderAroundAltExitFrame(t *testing.T) {
	surface := NewSurfaceTrackWithOptions(SurfaceSize{Cols: 20, Rows: 3}, SurfaceTrackOptions{
		PreserveAltScreenFrameOnExit: true,
	})
	result := surface.WriteWithResult("before\x1b[?1049h\x1b[2Jalt\x1b[?1049lafter")

	if len(result.Segments) != 2 {
		t.Fatalf("expected raw-before+frame and raw-after segments, got %#v", result.Segments)
	}
	if !strings.Contains(result.Segments[0].Raw, "before") || len(result.Segments[0].AltScreenExitFrame) == 0 {
		t.Fatalf("expected first segment to carry raw before exit and captured frame, got %#v", result.Segments[0])
	}
	if result.Segments[1].Raw != "after" || len(result.Segments[1].AltScreenExitFrame) != 0 {
		t.Fatalf("expected second segment to carry raw after exit, got %#v", result.Segments[1])
	}
}

func TestSurfaceTrackPreservesAltScreenFrameWhenExitSequenceIsSplit(t *testing.T) {
	surface := NewSurfaceTrackWithOptions(SurfaceSize{Cols: 20, Rows: 3}, SurfaceTrackOptions{
		PreserveAltScreenFrameOnExit: true,
	})
	surface.Write("\x1b[?1049h\x1b[2Jsplit-final")
	surface.Write("\x1b[?104")
	surface.Write("9l")

	if got := strings.Join(surface.Rows(), "\n"); !strings.Contains(got, "split-final") {
		t.Fatalf("expected split alt exit sequence to preserve final frame, got %q", got)
	}
	if surface.Snapshot().Modes.AlternateScreen {
		t.Fatal("expected split alt exit to leave alternate screen mode")
	}
}

func findCell(rows [][]vterm.Cell, content string) (vterm.Cell, bool) {
	for _, row := range rows {
		for _, cell := range row {
			if cell.Content == content {
				return cell, true
			}
		}
	}
	return vterm.Cell{}, false
}

func hasStyledBlankInRowContaining(rows [][]vterm.Cell, marker string, bg string) bool {
	for _, row := range rows {
		if !strings.Contains(vtermRowText(row), marker) {
			continue
		}
		for _, cell := range row {
			if cell.Content == " " && cell.Style.BG == bg {
				return true
			}
		}
	}
	return false
}

func hasStyledBlankInRowBeforeMarker(rows [][]vterm.Cell, marker string, bg string) bool {
	for i, row := range rows {
		if !strings.Contains(vtermRowText(row), marker) || i == 0 {
			continue
		}
		for _, cell := range rows[i-1] {
			if cell.Content == " " && cell.Style.BG == bg {
				return true
			}
		}
	}
	return false
}

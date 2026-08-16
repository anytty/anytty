package linehist

import (
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/core/history"
	vterm "github.com/anytty/anytty/vterm/vterm"
)

// Assembler 是纯拼装器：滚出物理行 + wrap 标志 -> 宽度无关 logical line。
// 这里只用合成滚出行验证拼装语义；真实 vterm 事务在 engine_vterm_test.go。

func evictedRowForTest(text string, wrapped bool) vterm.TerminalSemanticScrollOut {
	row := vterm.TerminalSemanticScrollOut{Wrapped: wrapped, WrappedSet: true}
	if text != "" {
		row.Runs = []vterm.TerminalSemanticCellRun{{Text: text}}
	}
	return row
}

func lineTextsForTest(lines []Line) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, LineText(line))
	}
	return out
}

func TestAssemblerEmitsHardLinesPerRow(t *testing.T) {
	asm := NewAssembler()
	var lines []Line
	for _, text := range []string{"one", "two", "three"} {
		lines = append(lines, asm.AppendEvictedRow(evictedRowForTest(text, false))...)
	}
	got := lineTextsForTest(lines)
	want := []string{"one", "two", "three"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("lines = %v, want %v", got, want)
	}
	for i, line := range lines {
		if !line.HardEnd {
			t.Fatalf("line %d must be hard-ended: %#v", i, line)
		}
	}
}

func TestAssemblerJoinsSoftWrappedRows(t *testing.T) {
	asm := NewAssembler()
	if lines := asm.AppendEvictedRow(evictedRowForTest("abcdef", true)); len(lines) != 0 {
		t.Fatalf("wrapped row must stay open, got %v", lineTextsForTest(lines))
	}
	lines := asm.AppendEvictedRow(evictedRowForTest("ghijkl", false))
	if len(lines) != 1 || LineText(lines[0]) != "abcdefghijkl" || !lines[0].HardEnd {
		t.Fatalf("joined line = %v, want single hard line abcdefghijkl", lineTextsForTest(lines))
	}
}

func TestAssemblerUsesLastPhysicalRowTimestamp(t *testing.T) {
	asm := NewAssembler()
	first := time.Date(2026, 8, 1, 9, 30, 0, 0, time.UTC)
	last := first.Add(2 * time.Second)
	wrapped := evictedRowForTest("abcdef", true)
	wrapped.Timestamp = first
	if lines := asm.AppendEvictedRow(wrapped); len(lines) != 0 {
		t.Fatalf("wrapped row must stay open, got %#v", lines)
	}
	hardEnd := evictedRowForTest("ghijkl", false)
	hardEnd.Timestamp = last
	lines := asm.AppendEvictedRow(hardEnd)
	if len(lines) != 1 || !lines[0].UpdatedAt.Equal(last) {
		t.Fatalf("logical line timestamp = %v, want %v", lines[0].UpdatedAt, last)
	}
}

func TestAssemblerPreservesBlankLines(t *testing.T) {
	asm := NewAssembler()
	var lines []Line
	lines = append(lines, asm.AppendEvictedRow(evictedRowForTest("para1", false))...)
	lines = append(lines, asm.AppendEvictedRow(evictedRowForTest("", false))...)
	lines = append(lines, asm.AppendEvictedRow(evictedRowForTest("para2", false))...)
	got := lineTextsForTest(lines)
	want := []string{"para1", "", "para2"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("lines = %v, want %v (blank line preserved)", got, want)
	}
	if len(lines[1].Runs) != 0 || !lines[1].HardEnd {
		t.Fatalf("blank line must be empty-runs hard line: %#v", lines[1])
	}
}

func TestAssemblerTrimsTrailingPaddingKeepsStyledBlank(t *testing.T) {
	asm := NewAssembler()
	row := vterm.TerminalSemanticScrollOut{
		Runs:       []vterm.TerminalSemanticCellRun{{Text: "ok"}, {Text: "    "}},
		WrappedSet: true,
	}
	lines := asm.AppendEvictedRow(row)
	if len(lines) != 1 || LineText(lines[0]) != "ok" {
		t.Fatalf("trailing unstyled padding must be trimmed, got %v", lineTextsForTest(lines))
	}

	styled := vterm.TerminalSemanticScrollOut{
		Runs: []vterm.TerminalSemanticCellRun{
			{Text: "bar"},
			{Text: "  ", Style: vterm.CellStyle{BG: "1"}},
		},
		WrappedSet: true,
	}
	lines = asm.AppendEvictedRow(styled)
	if len(lines) != 1 || LineText(lines[0]) != "bar  " {
		t.Fatalf("styled trailing blank is content, got %v", lineTextsForTest(lines))
	}
	last := lines[0].Runs[len(lines[0].Runs)-1]
	if last.Style.BG != "1" {
		t.Fatalf("styled trailing run must keep style: %#v", last)
	}
}

func TestAssemblerOpenTailUntilHardEnd(t *testing.T) {
	asm := NewAssembler()
	if lines := asm.AppendEvictedRow(evictedRowForTest("partial", true)); len(lines) != 0 {
		t.Fatalf("wrapped row must not emit lines, got %v", lineTextsForTest(lines))
	}
	open := asm.Open()
	if len(open) != 1 || open[0].Text != "partial" {
		t.Fatalf("open tail = %#v, want partial", open)
	}
	lines := asm.AppendEvictedRow(evictedRowForTest("done", false))
	if len(lines) != 1 || LineText(lines[0]) != "partialdone" {
		t.Fatalf("completed line = %v, want partialdone", lineTextsForTest(lines))
	}
	if len(asm.Open()) != 0 {
		t.Fatalf("open tail must reset after hard end: %#v", asm.Open())
	}
}

func TestAssemblerSealOpenClosesTail(t *testing.T) {
	asm := NewAssembler()
	asm.AppendEvictedRow(evictedRowForTest("tail", true))
	line, ok := asm.SealOpen()
	if !ok || LineText(line) != "tail" || !line.HardEnd {
		t.Fatalf("sealed open tail = %#v ok=%v, want hard line tail", line, ok)
	}
	if _, ok := asm.SealOpen(); ok {
		t.Fatalf("second seal must report nothing open")
	}
}

func TestAssemblerChunksPathologicalLongLine(t *testing.T) {
	asm := NewAssembler()
	asm.maxOpen = 10
	var lines []Line
	for i := 0; i < 4; i++ {
		lines = append(lines, asm.AppendEvictedRow(evictedRowForTest("abcdef", true))...)
	}
	lines = append(lines, asm.AppendEvictedRow(evictedRowForTest("end", false))...)
	if len(lines) < 2 {
		t.Fatalf("pathological long line must chunk-flush, got %v", lineTextsForTest(lines))
	}
	for i, line := range lines[:len(lines)-1] {
		if line.HardEnd {
			t.Fatalf("chunk %d must be HardEnd=false: %#v", i, line)
		}
	}
	last := lines[len(lines)-1]
	if !last.HardEnd {
		t.Fatalf("final record must be hard-ended: %#v", last)
	}
	joined := strings.Join(lineTextsForTest(lines), "")
	if joined != "abcdefabcdefabcdefabcdefend" {
		t.Fatalf("chunks must concatenate back to original text, got %q", joined)
	}
}

func TestAssemblerMergesCellsPayload(t *testing.T) {
	asm := NewAssembler()
	bold := vterm.CellStyle{Bold: true}
	row := vterm.TerminalSemanticScrollOut{
		Cells: []vterm.Cell{
			{Content: "a", Width: 1},
			{Content: "b", Width: 1},
			{Content: "你", Width: 2},
			{Content: "", Width: 0}, // 宽字符 continuation 占位
			{Content: "", Width: 1}, // 无文本但有列宽 -> 空格
			{Content: "c", Width: 1, Style: bold},
			{Content: "d", Width: 1, Style: bold},
		},
		WrappedSet: true,
	}
	lines := asm.AppendEvictedRow(row)
	if len(lines) != 1 || LineText(lines[0]) != "ab你 cd" {
		t.Fatalf("cells payload line = %q, want %q", LineText(lines[0]), "ab你 cd")
	}
	runs := lines[0].Runs
	if len(runs) != 2 {
		t.Fatalf("same-style cells must merge into runs, got %#v", runs)
	}
	if runs[0].Text != "ab你 " || runs[0].Style != (history.CellStyle{}) {
		t.Fatalf("first run = %#v, want unstyled %q", runs[0], "ab你 ")
	}
	if runs[1].Text != "cd" || !runs[1].Style.Bold {
		t.Fatalf("second run = %#v, want bold cd", runs[1])
	}
}

func TestAssemblerLinkStaysOnRunNotStyle(t *testing.T) {
	asm := NewAssembler()
	row := vterm.TerminalSemanticScrollOut{
		Runs: []vterm.TerminalSemanticCellRun{
			{Text: "link", Style: vterm.CellStyle{LinkURL: "https://example.com", LinkParams: "id=1"}},
		},
		WrappedSet: true,
	}
	lines := asm.AppendEvictedRow(row)
	if len(lines) != 1 || len(lines[0].Runs) != 1 {
		t.Fatalf("expected single-run line, got %#v", lines)
	}
	run := lines[0].Runs[0]
	if run.LinkURL != "https://example.com" || run.LinkParams != "id=1" {
		t.Fatalf("link must be carried on run fields: %#v", run)
	}
	if run.Style != (history.CellStyle{}) {
		t.Fatalf("history style must not carry link: %#v", run.Style)
	}
}

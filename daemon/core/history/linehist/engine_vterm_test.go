package linehist

import (
	"fmt"
	"strings"
	"testing"

	vterm "github.com/anytty/anytty/vterm/vterm"
)

// 合成事务 harness：用真实 vterm SemanticSource（scrollbackSize=0，与生产
// SemanticTap 一致）产出 EvictedRows，喂给 Engine，验证滚出内容按 logical
// line 语义落盘。这是 R432 的准入：engine 只消费 EvictedRows，不读屏幕。

func newEngineForTest(t *testing.T, dir string) *Engine {
	t.Helper()
	file := openTestLineStorage(t, dir, "term-e2e")
	return NewEngine(file)
}

func applyWriteForTest(t *testing.T, source *vterm.SemanticSource, engine *Engine, raw string) {
	t.Helper()
	tx, err := source.ApplyPTYWrite([]byte(raw))
	if err != nil {
		t.Fatalf("apply pty write: %v", err)
	}
	if err := engine.ApplyEvictedRows(tx.EvictedRows); err != nil {
		t.Fatalf("apply evicted rows: %v", err)
	}
}

func storedTextsForTest(t *testing.T, engine *Engine) []string {
	t.Helper()
	lines, err := engine.Lines(0, engine.LineCount())
	if err != nil {
		t.Fatalf("read stored lines: %v", err)
	}
	return lineTextsForTest(lines)
}

func TestEngineStoresEvictedLinesFromRealVTerm(t *testing.T) {
	engine := newEngineForTest(t, t.TempDir())
	defer engine.Close()
	source := vterm.NewSemanticSource(12, 2, 0, nil)
	for i := 1; i <= 6; i++ {
		applyWriteForTest(t, source, engine, fmt.Sprintf("line%02d\r\n", i))
	}
	got := storedTextsForTest(t, engine)
	want := []string{"line01", "line02", "line03", "line04", "line05"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("stored lines = %v, want %v", got, want)
	}
}

func TestEngineRejoinsSoftWrapFromRealVTerm(t *testing.T) {
	engine := newEngineForTest(t, t.TempDir())
	defer engine.Close()
	// 6 列屏：12 字符命令软换行成两物理行，滚出后必须拼回一条 logical line。
	source := vterm.NewSemanticSource(6, 2, 0, nil)
	applyWriteForTest(t, source, engine, "abcdefghijkl\r\nx\r\ny")
	got := storedTextsForTest(t, engine)
	want := []string{"abcdefghijkl"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("stored lines = %v, want %v (soft wrap rejoined)", got, want)
	}
}

func TestEnginePreservesBlankLinesFromRealVTerm(t *testing.T) {
	engine := newEngineForTest(t, t.TempDir())
	defer engine.Close()
	source := vterm.NewSemanticSource(12, 2, 0, nil)
	applyWriteForTest(t, source, engine, "para1\r\n\r\npara2\r\nx\r\ny")
	got := storedTextsForTest(t, engine)
	want := []string{"para1", "", "para2"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("stored lines = %v, want %v (blank preserved)", got, want)
	}
}

func TestEngineIgnoresAltScreenScrollFromRealVTerm(t *testing.T) {
	engine := newEngineForTest(t, t.TempDir())
	defer engine.Close()
	source := vterm.NewSemanticSource(12, 2, 0, nil)
	applyWriteForTest(t, source, engine, "shell1\r\nshell2\r\nshell3\r\nx")
	raw := "\x1b[?1049h"
	for i := 1; i <= 6; i++ {
		raw += fmt.Sprintf("alt%02d\r\n", i)
	}
	raw += "\x1b[?1049l"
	applyWriteForTest(t, source, engine, raw)
	got := storedTextsForTest(t, engine)
	for _, text := range got {
		if strings.HasPrefix(text, "alt") {
			t.Fatalf("alt screen scroll must not enter logical history: %v", got)
		}
	}
	want := []string{"shell1", "shell2"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("stored lines = %v, want %v", got, want)
	}
}

func TestEngineRejoinsWideCharWrapFromRealVTerm(t *testing.T) {
	engine := newEngineForTest(t, t.TempDir())
	defer engine.Close()
	// 4 列屏：宽字符两两一行软换行，滚出后拼回完整 CJK logical line。
	source := vterm.NewSemanticSource(4, 2, 0, nil)
	applyWriteForTest(t, source, engine, "你好世界\r\nx\r\ny")
	got := storedTextsForTest(t, engine)
	want := []string{"你好世界"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("stored lines = %v, want %v (wide-char wrap rejoined)", got, want)
	}
}

func TestEngineOpenTailExposedThenSealedOnClose(t *testing.T) {
	dir := t.TempDir()
	engine := newEngineForTest(t, dir)
	// 6 列屏写 9 字符：首物理行 "abcdef" 带 wrap 标志滚出，续行还在屏上。
	source := vterm.NewSemanticSource(6, 2, 0, nil)
	applyWriteForTest(t, source, engine, "abcdefghi\r\nx")
	if count := engine.LineCount(); count != 0 {
		t.Fatalf("open logical line must not be persisted yet, count=%d %v", count, storedTextsForTest(t, engine))
	}
	open := engine.OpenTail()
	if len(open) == 0 || open[0].Text != "abcdef" {
		t.Fatalf("open tail = %#v, want abcdef", open)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("close engine: %v", err)
	}
	// Close 把未闭合尾部按硬结束落盘，重启后不丢已滚出内容。
	reopened := newEngineForTest(t, dir)
	defer reopened.Close()
	got := storedTextsForTest(t, reopened)
	want := []string{"abcdef"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("lines after close/reopen = %v, want %v", got, want)
	}
}

func TestEngineLifecycleSealPersistsCurrentPrimaryScreen(t *testing.T) {
	engine := newEngineForTest(t, t.TempDir())
	defer engine.Close()
	source := vterm.NewSemanticSource(6, 2, 0, nil)
	applyWriteForTest(t, source, engine, "abcdefghi\r\nx")
	before := storedTextsForTest(t, engine)
	if len(before) != 0 {
		t.Fatalf("current screen rows must stay hot before lifecycle seal, got %v", before)
	}
	snap := ScreenSnapshotFromVTerm(source.VTerm())
	if err := engine.SealPrimaryScreenRows(snap.Rows); err != nil {
		t.Fatalf("seal current screen: %v", err)
	}
	got := storedTextsForTest(t, engine)
	want := []string{"abcdefghi", "x"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("lifecycle seal must persist open tail plus current screen, got %v want %v", got, want)
	}
}

func TestScreenSnapshotKeepsPrimaryTimestampsWhileAltIsActive(t *testing.T) {
	engine := newEngineForTest(t, t.TempDir())
	defer engine.Close()
	source := vterm.NewSemanticSource(12, 3, 0, nil)
	applyWriteForTest(t, source, engine, "one\r\ntwo")
	before := ScreenSnapshotFromVTerm(source.VTerm())
	applyWriteForTest(t, source, engine, "\x1b[?1049h\x1b[2J\x1b[HALT")
	after := ScreenSnapshotFromVTerm(source.VTerm())
	if !after.InAlt || len(after.PrimaryRows) < 2 {
		t.Fatalf("missing saved primary rows in alt snapshot: %#v", after)
	}
	if after.PrimaryRows[0].UpdatedAt.IsZero() || after.PrimaryRows[1].UpdatedAt.IsZero() {
		t.Fatalf("saved primary timestamps are zero: %#v", after.PrimaryRows)
	}
	if !after.PrimaryRows[0].UpdatedAt.Equal(before.Rows[0].UpdatedAt) || !after.PrimaryRows[1].UpdatedAt.Equal(before.Rows[1].UpdatedAt) {
		t.Fatalf("saved primary timestamps changed: before=%#v after=%#v", before.Rows, after.PrimaryRows)
	}
}

func TestEnginePersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	engine := newEngineForTest(t, dir)
	source := vterm.NewSemanticSource(12, 2, 0, nil)
	for i := 1; i <= 5; i++ {
		applyWriteForTest(t, source, engine, fmt.Sprintf("cold%02d\r\n", i))
	}
	before := storedTextsForTest(t, engine)
	if err := engine.Close(); err != nil {
		t.Fatalf("close engine: %v", err)
	}
	reopened := newEngineForTest(t, dir)
	defer reopened.Close()
	after := storedTextsForTest(t, reopened)
	if strings.Join(before, "|") != strings.Join(after, "|") {
		t.Fatalf("reopen changed history: before=%v after=%v", before, after)
	}
	// 继续写入必须接在冷历史之后。
	source2 := vterm.NewSemanticSource(12, 2, 0, nil)
	applyWriteForTest(t, source2, reopened, "new01\r\nnew02\r\nnew03\r\nx")
	got := storedTextsForTest(t, reopened)
	if len(got) <= len(after) || got[len(after)] != "new01" {
		t.Fatalf("new session lines must append after cold history: %v", got)
	}
}

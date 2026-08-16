package linehist

import (
	"fmt"
	"strings"
	"testing"

	"github.com/anytty/anytty/core/history"
	vterm "github.com/anytty/anytty/vterm/vterm"
)

// R434 准入（store 级）：resize / ED3-ClearScrollback / alt-screen 边界。
// 口径：clear 只前移可见分段（文件 append-only、frozen token 不受影响）；
// resize 前后投影不重不漏；alt 期间 primary 时间线尾部保持可见且 mutable。

func newStoreHarnessInDir(t *testing.T, dir string, terminalID string, cols int, rows int) *storeHarness {
	t.Helper()
	file := openTestLineStorage(t, dir, terminalID)
	harness := &storeHarness{
		t:      t,
		source: vterm.NewSemanticSource(cols, rows, 0, nil),
		store:  NewStore(terminalID, NewEngine(file)),
	}
	harness.store.Bind(func() ScreenSnapshot {
		return ScreenSnapshotFromVTerm(harness.source.VTerm())
	}, &harness.gate)
	return harness
}

func (harness *storeHarness) resize(cols int, rows int) {
	harness.t.Helper()
	harness.gate.Lock()
	tx, err := harness.source.Resize(vterm.TerminalSemanticSize{Cols: cols, Rows: rows})
	if err == nil {
		err = harness.store.ApplyTransaction(tx)
	}
	harness.gate.Unlock()
	if err != nil {
		harness.t.Fatalf("harness resize: %v", err)
	}
}

// collectAllTextsForTest 走 latest + older 分页闭环收集全量投影文本。
func collectAllTextsForTest(t *testing.T, store *Store, cols int, limit int) []string {
	t.Helper()
	latest, err := store.LatestWindow(history.HistoryWindowRequest{Cols: cols, Limit: limit})
	if err != nil {
		t.Fatalf("latest window: %v", err)
	}
	rows := append([]history.HistoryRow(nil), latest.Rows...)
	cursor := latest.Boundary.Cursor
	for cursor.Valid {
		older, err := store.OlderWindow(history.HistoryWindowRequest{Cols: cols, Limit: limit, Cursor: cursor})
		if err != nil {
			t.Fatalf("older window: %v", err)
		}
		if len(older.Rows) == 0 {
			break
		}
		rows = append(append([]history.HistoryRow(nil), older.Rows...), rows...)
		cursor = older.Boundary.Cursor
	}
	texts := make([]string, 0, len(rows))
	for _, row := range rows {
		texts = append(texts, rowText(row.Cells))
	}
	return texts
}

func TestStoreClearScrollbackKeepsColdHistoryAndLineIDs(t *testing.T) {
	harness := newStoreHarness(t, 12, 2)
	for i := 1; i <= 4; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	harness.write("\x1b[3J")
	window, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 12, Limit: 100})
	if err != nil {
		t.Fatalf("latest window: %v", err)
	}
	joined := strings.Join(windowTextsForTest(window), "|")
	for i := 1; i <= 4; i++ {
		want := fmt.Sprintf("line%02d", i)
		if strings.Count(joined, want) != 1 {
			t.Fatalf("ED3 soft boundary must keep %q exactly once, got %q", want, joined)
		}
	}
	if len(window.Rows) == 0 || window.Rows[0].LineID != 1 {
		t.Fatalf("ED3 must not restart visible LineID space, got %#v", window.Rows)
	}
	// clear 之后新的滚出照常进入同一份可翻页历史。
	for i := 5; i <= 7; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	afterJoined := strings.Join(collectAllTextsForTest(t, harness.store, 12, 100), "|")
	for i := 1; i <= 7; i++ {
		want := fmt.Sprintf("line%02d", i)
		if strings.Count(afterJoined, want) != 1 {
			t.Fatalf("history across ED3 must contain %q exactly once, got %q", want, afterJoined)
		}
	}
}

func TestStoreClearThenEd2ComboKeepsEverything(t *testing.T) {
	harness := newStoreHarness(t, 12, 2)
	for i := 1; i <= 4; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	// `clear` 的完整形态：ED2 把屏幕内容挤走 + ED3 清 scrollback，一次写入。
	harness.write("\x1b[2J\x1b[3J\x1b[H")
	joined := strings.Join(collectAllTextsForTest(t, harness.store, 12, 100), "|")
	for i := 1; i <= 4; i++ {
		want := fmt.Sprintf("line%02d", i)
		if strings.Count(joined, want) != 1 {
			t.Fatalf("ED2+ED3 clear must keep %q exactly once, got %q", want, joined)
		}
	}
}

func TestStoreClearScrollbackBoundaryKeepsHistoryAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	harness := newStoreHarnessInDir(t, dir, "term-reopen", 12, 2)
	for i := 1; i <= 4; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	harness.write("\x1b[3J")
	for i := 5; i <= 6; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	if err := harness.store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}
	reopened := newStoreHarnessInDir(t, dir, "term-reopen", 12, 2)
	t.Cleanup(func() { _ = reopened.store.Close() })
	texts := collectAllTextsForTest(t, reopened.store, 12, 100)
	joined := strings.Join(texts, "|")
	// reopen 后只能恢复已 seal 的冷段；仍在关闭前 live 屏上的尾行不强行
	// 从 snapshot 反推，但 clear 前已落盘的 history 必须继续可见。
	for _, want := range []string{"line01", "line02", "line03", "line04", "line05"} {
		if strings.Count(joined, want) != 1 {
			t.Fatalf("history across clear boundary must survive reopen with %q exactly once, got %v", want, texts)
		}
	}
}

func TestStoreFrozenTokenAndLiveViewServePreClearContent(t *testing.T) {
	harness := newStoreHarness(t, 12, 2)
	for i := 1; i <= 4; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	snapshot, err := harness.store.Freeze(history.FreezeHistoryRequest{Cols: 12, Limit: 100})
	if err != nil {
		t.Fatalf("freeze: %v", err)
	}
	harness.write("\x1b[3J")
	frozen, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 12, Limit: 100, Token: snapshot.Token})
	if err != nil {
		t.Fatalf("frozen window after clear: %v", err)
	}
	frozenJoined := strings.Join(windowTextsForTest(frozen), "|")
	for _, want := range []string{"line01", "line02", "line03", "line04"} {
		if !strings.Contains(frozenJoined, want) {
			t.Fatalf("frozen token must keep pre-clear content, missing %q in %q", want, frozenJoined)
		}
	}
	copied, err := harness.store.Copy(history.HistoryCopyRequest{
		Token: snapshot.Token,
		Cols:  12,
		Range: &history.HistoryCopyRange{Start: history.HistoryCopyPosition{LineID: 1}, End: history.HistoryCopyPosition{LineID: 2, Col: 6}},
	})
	if err != nil {
		t.Fatalf("copy from frozen token after clear: %v", err)
	}
	if copied != "line01\nline02" {
		t.Fatalf("frozen copy after clear = %q, want line01/line02", copied)
	}
	live, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 12, Limit: 100})
	if err != nil {
		t.Fatalf("live window after clear: %v", err)
	}
	liveJoined := strings.Join(windowTextsForTest(live), "|")
	for _, want := range []string{"line01", "line02", "line03", "line04"} {
		if !strings.Contains(liveJoined, want) {
			t.Fatalf("live view must keep pre-clear content %q, got %v", want, windowTextsForTest(live))
		}
	}
}

func TestStoreRISKeepsHistoryVisible(t *testing.T) {
	harness := newStoreHarness(t, 12, 2)
	for i := 1; i <= 4; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	// RIS 重置终端但不清 scrollback（xterm 默认行为）。
	harness.write("\x1bc")
	texts := collectAllTextsForTest(t, harness.store, 12, 100)
	joined := strings.Join(texts, "|")
	for _, want := range []string{"line01", "line02", "line03"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("RIS must not clear history, missing %q in %v", want, texts)
		}
	}
}

func TestStoreResizeNarrowerKeepsProjectionExact(t *testing.T) {
	harness := newStoreHarness(t, 12, 3)
	const totalLines = 10
	for i := 1; i <= totalLines; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	harness.resize(6, 2)
	texts := collectAllTextsForTest(t, harness.store, 12, 4)
	joined := strings.Join(texts, "|")
	for i := 1; i <= totalLines; i++ {
		want := fmt.Sprintf("line%02d", i)
		if strings.Count(joined, want) != 1 {
			t.Fatalf("resize narrower lost/duplicated %q, got %v", want, texts)
		}
	}
	for i := 1; i < totalLines; i++ {
		if strings.Index(joined, fmt.Sprintf("line%02d", i)) > strings.Index(joined, fmt.Sprintf("line%02d", i+1)) {
			t.Fatalf("resize narrower broke ordering: %v", texts)
		}
	}
}

func TestStoreResizeTallerKeepsProjectionExact(t *testing.T) {
	harness := newStoreHarness(t, 12, 2)
	const totalLines = 8
	for i := 1; i <= totalLines; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	harness.resize(12, 6)
	texts := collectAllTextsForTest(t, harness.store, 12, 4)
	joined := strings.Join(texts, "|")
	for i := 1; i <= totalLines; i++ {
		want := fmt.Sprintf("line%02d", i)
		if strings.Count(joined, want) != 1 {
			t.Fatalf("resize taller lost/duplicated %q, got %v", want, texts)
		}
	}
	// 变高后继续写入不产生重复。
	for i := totalLines + 1; i <= totalLines+4; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	after := collectAllTextsForTest(t, harness.store, 12, 4)
	afterJoined := strings.Join(after, "|")
	for i := 1; i <= totalLines+4; i++ {
		want := fmt.Sprintf("line%02d", i)
		if strings.Count(afterJoined, want) != 1 {
			t.Fatalf("post-resize writes lost/duplicated %q, got %v", want, after)
		}
	}
}

func TestStoreAltKeepsPrimaryTailVisibleAndMutable(t *testing.T) {
	harness := newStoreHarness(t, 12, 3)
	for i := 1; i <= 5; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	harness.write("\x1b[?1049h\x1b[2J\x1b[HALT-FRAME")
	window, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 12, Limit: 100})
	if err != nil {
		t.Fatalf("latest window: %v", err)
	}
	byText := make(map[string]history.HistoryRow)
	for _, row := range window.Rows {
		byText[rowText(row.Cells)] = row
	}
	for _, want := range []string{"line04", "line05"} {
		row, ok := byText[want]
		if !ok {
			t.Fatalf("primary tail %q must stay visible during alt, got %v", want, windowTextsForTest(window))
		}
		if row.Committed || row.Segment != history.HistorySegmentCurrentPrimaryFrame {
			t.Fatalf("primary tail %q must stay mutable primary-frame during alt, got %#v", want, row)
		}
	}
	altRow, ok := byText["ALT-FRAME"]
	if !ok || !altRow.FixedGrid || altRow.Segment != history.HistorySegmentCurrentAltFrame {
		t.Fatalf("alt frame row missing or malformed, got %#v (ok=%v)", altRow, ok)
	}
	if altRow.LineID <= byText["line05"].LineID {
		t.Fatalf("alt rows must come after primary tail: alt LineID %d vs %d", altRow.LineID, byText["line05"].LineID)
	}
	// 退出 alt 并继续写入：primary 尾部不重复、alt 内容不留在历史。
	harness.write("\x1b[?1049l")
	harness.write("post-alt\r\n")
	texts := collectAllTextsForTest(t, harness.store, 12, 100)
	joined := strings.Join(texts, "|")
	if strings.Contains(joined, "ALT-FRAME") {
		t.Fatalf("alt transient content must not survive alt exit, got %v", texts)
	}
	for _, want := range []string{"line04", "line05", "post-alt"} {
		if strings.Count(joined, want) != 1 {
			t.Fatalf("primary timeline must contain %q exactly once after alt exit, got %v", want, texts)
		}
	}
}

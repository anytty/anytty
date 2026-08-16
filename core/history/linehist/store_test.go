package linehist

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/anytty/anytty/core/history"
	vterm "github.com/anytty/anytty/vterm/vterm"
)

// R433 准入 harness：真实 vterm SemanticSource + Store，写入走
// gate 临界区（模拟 Terminal 的 tapOpMu），查询走 HistoryStore 接口。
// 断言口径：冷段（文件）+ 热段（emulator 当前屏）投影不重不漏。

type storeHarness struct {
	t      *testing.T
	source *vterm.SemanticSource
	store  *Store
	gate   sync.Mutex
}

func openTestLineStorage(t testing.TB, dir string, terminalID string) *CompressedLineFile {
	t.Helper()
	file, err := OpenCompressedLineFile(dir, terminalID, CompressedLineFileOptions{Compression: compressionNone})
	if err != nil {
		t.Fatalf("open compressed line file: %v", err)
	}
	return file
}

func newStoreHarness(t *testing.T, cols int, rows int) *storeHarness {
	t.Helper()
	file := openTestLineStorage(t, t.TempDir(), "term-store")
	harness := &storeHarness{
		t:      t,
		source: vterm.NewSemanticSource(cols, rows, 0, nil),
		store:  NewStore("term-store", NewEngine(file)),
	}
	harness.store.Bind(func() ScreenSnapshot {
		return ScreenSnapshotFromVTerm(harness.source.VTerm())
	}, &harness.gate)
	t.Cleanup(func() { _ = harness.store.Close() })
	return harness
}

func (harness *storeHarness) write(raw string) {
	harness.t.Helper()
	harness.gate.Lock()
	tx, err := harness.source.ApplyPTYWrite([]byte(raw))
	if err == nil {
		err = harness.store.ApplyTransaction(tx)
	}
	harness.gate.Unlock()
	if err != nil {
		harness.t.Fatalf("harness write: %v", err)
	}
}

func windowTextsForTest(window history.HistoryWindow) []string {
	texts := make([]string, 0, len(window.Rows))
	for _, row := range window.Rows {
		texts = append(texts, rowText(row.Cells))
	}
	return texts
}

func TestStoreLatestWindowJoinsColdAndHotWithoutDupOrLoss(t *testing.T) {
	harness := newStoreHarness(t, 12, 3)
	for i := 1; i <= 6; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	window, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 12, Limit: 100})
	if err != nil {
		t.Fatalf("latest window: %v", err)
	}
	got := windowTextsForTest(window)
	want := []string{"line01", "line02", "line03", "line04", "line05", "line06"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("latest rows = %v, want %v", got, want)
	}
	for i, row := range window.Rows {
		if int(row.LineID) != i+1 {
			t.Fatalf("row %d LineID = %d, want %d", i, row.LineID, i+1)
		}
	}
	// 冷段行 committed，热段（仍在屏上的行）mutable。
	if !window.Rows[0].Committed || window.Rows[0].Segment != history.HistorySegmentCommitted {
		t.Fatalf("cold row must be committed, got %#v", window.Rows[0])
	}
	last := window.Rows[len(window.Rows)-1]
	if last.Committed || last.Segment != history.HistorySegmentCurrentPrimaryFrame {
		t.Fatalf("hot row must be mutable current-primary, got %#v", last)
	}
	if window.LogicalTotal != 6 {
		t.Fatalf("LogicalTotal = %d, want 6", window.LogicalTotal)
	}
}

func TestStoreOlderPagingRoundTripCoversFullProjection(t *testing.T) {
	harness := newStoreHarness(t, 12, 3)
	const totalLines = 30
	for i := 1; i <= totalLines; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	latest, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 12, Limit: 5})
	if err != nil {
		t.Fatalf("latest window: %v", err)
	}
	rows := append([]history.HistoryRow(nil), latest.Rows...)
	cursor := latest.Boundary.Cursor
	pageCount := 1
	for cursor.Valid {
		older, err := harness.store.OlderWindow(history.HistoryWindowRequest{Cols: 12, Limit: 5, Cursor: cursor})
		if err != nil {
			t.Fatalf("older window: %v", err)
		}
		if older.Op != history.HistoryWindowPrepend {
			t.Fatalf("older op = %v, want prepend", older.Op)
		}
		if len(older.Rows) == 0 {
			break
		}
		pageCount++
		rows = append(append([]history.HistoryRow(nil), older.Rows...), rows...)
		cursor = older.Boundary.Cursor
	}
	if pageCount < 3 {
		t.Fatalf("expected multi-page older traversal, pages=%d", pageCount)
	}
	texts := make([]string, 0, len(rows))
	for _, row := range rows {
		texts = append(texts, rowText(row.Cells))
	}
	want := make([]string, 0, totalLines)
	for i := 1; i <= totalLines; i++ {
		want = append(want, fmt.Sprintf("line%02d", i))
	}
	if strings.Join(texts, "|") != strings.Join(want, "|") {
		t.Fatalf("older paging round trip = %v, want %v", texts, want)
	}
}

func TestStoreReadsEachColdWindowInOneStorageCall(t *testing.T) {
	storage := &durabilityLineStorage{lines: make([]Line, 100)}
	store := NewStore("bulk-window", NewEngine(storage))
	t.Cleanup(func() { _ = store.Close() })

	latest, err := store.LatestWindow(history.HistoryWindowRequest{Cols: 80, Limit: 100})
	if err != nil {
		t.Fatalf("latest window: %v", err)
	}
	if len(latest.Rows) != 100 || storage.linesCalls.Load() != 1 {
		t.Fatalf("latest rows=%d storage reads=%d, want 100 rows in one read", len(latest.Rows), storage.linesCalls.Load())
	}
	storage.linesCalls.Store(0)
	oldest, err := store.OldestWindow(history.HistoryWindowRequest{Cols: 80, Limit: 100})
	if err != nil {
		t.Fatalf("oldest window: %v", err)
	}
	if len(oldest.Rows) != 100 || storage.linesCalls.Load() != 1 {
		t.Fatalf("oldest rows=%d storage reads=%d, want 100 rows in one read", len(oldest.Rows), storage.linesCalls.Load())
	}
}

func TestStoreReturnsLogicalColdLinesIndependentOfRequestCols(t *testing.T) {
	harness := newStoreHarness(t, 6, 2)
	// 12 字符命令在 6 列屏软换行；滚出后落盘为一条 logical line。
	harness.write("abcdefghijkl\r\nx\r\ny\r\nz\r\n")
	wide, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 20, Limit: 100})
	if err != nil {
		t.Fatalf("wide window: %v", err)
	}
	wideTexts := windowTextsForTest(wide)
	if wideTexts[0] != "abcdefghijkl" {
		t.Fatalf("wide cols must rejoin soft wrap into one row, got %v", wideTexts)
	}
	if wide.Rows[0].Wrapped {
		t.Fatalf("hard-ended line at wide cols must not stay wrapped, got %#v", wide.Rows[0])
	}
	narrow, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 4, Limit: 100})
	if err != nil {
		t.Fatalf("narrow window: %v", err)
	}
	narrowTexts := windowTextsForTest(narrow)
	if narrowTexts[0] != "abcdefghijkl" {
		t.Fatalf("core history window must stay logical-line based; got %v", narrowTexts)
	}
	if narrow.Rows[0].LineID != 1 || narrow.Rows[0].RowInLine != 0 || narrow.Rows[0].Wrapped {
		t.Fatalf("narrow request must not create global visual rows, got %#v", narrow.Rows[0])
	}
}

func TestStoreJoinsOpenTailWithOnScreenContinuation(t *testing.T) {
	harness := newStoreHarness(t, 6, 2)
	// 18 字符不带换行：首物理行滚出（wrapped，进 open tail），后两行仍在屏。
	harness.write("abcdefghijklmnopqr")
	window, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 18, Limit: 100})
	if err != nil {
		t.Fatalf("latest window: %v", err)
	}
	texts := windowTextsForTest(window)
	if len(texts) != 1 || texts[0] != "abcdefghijklmnopqr" {
		t.Fatalf("open tail must rejoin with on-screen continuation, got %v", texts)
	}
	row := window.Rows[0]
	if row.Committed || row.Segment != history.HistorySegmentCurrentPrimaryFrame {
		t.Fatalf("joined open-tail line is still mutable hot content, got %#v", row)
	}
	wantAnchor := history.HistoryViewportAnchor{
		TopLineID: 1, TopCellOffset: 6, ScreenCols: 6, ScreenRows: 2, Valid: true,
	}
	if window.ViewportAnchor != wantAnchor {
		t.Fatalf("viewport anchor = %#v, want %#v", window.ViewportAnchor, wantAnchor)
	}
}

func TestStoreViewportAnchorPreservesLeadingBlankScreenRows(t *testing.T) {
	harness := newStoreHarness(t, 12, 5)
	harness.write("\x1b[3;1Hmiddle")

	window, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 12, Limit: 100})
	if err != nil {
		t.Fatalf("latest window: %v", err)
	}
	if got := windowTextsForTest(window); strings.Join(got, "|") != "||middle" {
		t.Fatalf("leading screen rows = %#v, want two blanks then middle", got)
	}
	wantAnchor := history.HistoryViewportAnchor{
		TopLineID: 1, ScreenCols: 12, ScreenRows: 5, Valid: true,
	}
	if window.ViewportAnchor != wantAnchor {
		t.Fatalf("viewport anchor = %#v, want %#v", window.ViewportAnchor, wantAnchor)
	}
}

func TestStoreAltScreenProjectsFixedGridWithColdIntact(t *testing.T) {
	harness := newStoreHarness(t, 12, 2)
	for i := 1; i <= 4; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	harness.write("\x1b[?1049h\x1b[2J\x1b[HALT-FRAME")
	window, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 12, Limit: 100})
	if err != nil {
		t.Fatalf("latest window: %v", err)
	}
	texts := windowTextsForTest(window)
	joined := strings.Join(texts, "|")
	for _, want := range []string{"line01", "line02", "line03"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("cold committed lines must stay in projection during alt, got %v", texts)
		}
	}
	var altRow *history.HistoryRow
	for i := range window.Rows {
		if window.Rows[i].Segment == history.HistorySegmentCurrentAltFrame {
			altRow = &window.Rows[i]
			break
		}
	}
	if altRow == nil {
		t.Fatalf("alt frame rows missing from projection, got %v", texts)
	}
	if !altRow.FixedGrid || altRow.Kind != history.LineKindAltScreenFrame || !altRow.ScreenRowSet {
		t.Fatalf("alt frame row must be fixed-grid alt kind, got %#v", *altRow)
	}
	if altRow.Committed {
		t.Fatalf("alt frame row must not be committed, got %#v", *altRow)
	}
	if !strings.Contains(joined, "ALT-FRAME") {
		t.Fatalf("alt frame content missing, got %v", texts)
	}
	if anchor := window.ViewportAnchor; !anchor.Valid || anchor.AtEnd || anchor.TopLineID != altRow.LineID || anchor.ScreenRows != 2 {
		t.Fatalf("alt viewport anchor must point at the first alt row, got %#v", anchor)
	}
}

func TestStoreFreezeIsStableAcrossLaterWritesAndCopyUsesToken(t *testing.T) {
	harness := newStoreHarness(t, 12, 3)
	for i := 1; i <= 6; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	snapshot, err := harness.store.Freeze(history.FreezeHistoryRequest{Cols: 12, Limit: 100})
	if err != nil {
		t.Fatalf("freeze: %v", err)
	}
	if snapshot.Token == "" {
		t.Fatalf("freeze must return token, got %#v", snapshot)
	}
	frozenBefore, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 12, Limit: 100, Token: snapshot.Token})
	if err != nil {
		t.Fatalf("frozen latest window: %v", err)
	}
	for i := 7; i <= 9; i++ {
		harness.write(fmt.Sprintf("line%02d\r\n", i))
	}
	frozenAfter, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 12, Limit: 100, Token: snapshot.Token})
	if err != nil {
		t.Fatalf("frozen latest window after writes: %v", err)
	}
	beforeTexts := strings.Join(windowTextsForTest(frozenBefore), "|")
	afterTexts := strings.Join(windowTextsForTest(frozenAfter), "|")
	if beforeTexts != afterTexts {
		t.Fatalf("frozen view changed after later writes: before=%q after=%q", beforeTexts, afterTexts)
	}
	if frozenBefore.ViewportAnchor != frozenAfter.ViewportAnchor || snapshot.ViewportAnchor != frozenBefore.ViewportAnchor {
		t.Fatalf("frozen viewport anchor changed: snapshot=%#v before=%#v after=%#v", snapshot.ViewportAnchor, frozenBefore.ViewportAnchor, frozenAfter.ViewportAnchor)
	}
	if strings.Contains(afterTexts, "line07") {
		t.Fatalf("frozen view must not include post-freeze writes, got %q", afterTexts)
	}
	live, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 12, Limit: 100})
	if err != nil {
		t.Fatalf("live latest window: %v", err)
	}
	if !strings.Contains(strings.Join(windowTextsForTest(live), "|"), "line09") {
		t.Fatalf("live view must include post-freeze writes, got %v", windowTextsForTest(live))
	}
	copied, err := harness.store.Copy(history.HistoryCopyRequest{
		Token: snapshot.Token,
		Cols:  12,
		Range: &history.HistoryCopyRange{Start: history.HistoryCopyPosition{LineID: 2}, End: history.HistoryCopyPosition{LineID: 4, Col: 6}},
	})
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
	if copied != "line02\nline03\nline04" {
		t.Fatalf("copy range = %q, want line02..line04", copied)
	}
	if err := harness.store.Release(snapshot.Token); err != nil {
		t.Fatalf("release: %v", err)
	}
	if _, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 12, Limit: 100, Token: snapshot.Token}); err == nil {
		t.Fatalf("released token must be rejected")
	}
}

func TestStoreEmptyAndInvalidCursorWindows(t *testing.T) {
	harness := newStoreHarness(t, 12, 3)
	latest, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 12, Limit: 100})
	if err != nil {
		t.Fatalf("empty latest window: %v", err)
	}
	if len(latest.Rows) != 0 || latest.LogicalTotal != 0 {
		t.Fatalf("empty store latest = %#v, want empty", latest)
	}
	if anchor := latest.ViewportAnchor; !anchor.Valid || !anchor.AtEnd || anchor.ScreenCols != 12 || anchor.ScreenRows != 3 {
		t.Fatalf("empty screen must anchor at the end without creating rows, got %#v", anchor)
	}
	older, err := harness.store.OlderWindow(history.HistoryWindowRequest{Cols: 12, Limit: 100})
	if err != nil {
		t.Fatalf("older without cursor: %v", err)
	}
	if len(older.Rows) != 0 || older.Op != history.HistoryWindowPrepend {
		t.Fatalf("older without valid cursor must be empty prepend, got %#v", older)
	}
	newer, err := harness.store.NewerWindow(history.HistoryWindowRequest{Cols: 12, Limit: 100})
	if err != nil {
		t.Fatalf("newer without cursor: %v", err)
	}
	if len(newer.Rows) != 0 || newer.Op != history.HistoryWindowAppend {
		t.Fatalf("newer without valid cursor must be empty append, got %#v", newer)
	}
}

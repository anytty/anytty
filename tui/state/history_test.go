package state

import (
	"errors"
	"reflect"
	"testing"
)

func TestHistoryStoreAcceptsLatestReplace(t *testing.T) {
	store, err := (HistoryStore{}).BeginLatest(HistoryPendingRequest{
		ID:         1,
		TerminalID: "term-1",
		Cols:       80,
	})
	if err != nil {
		t.Fatalf("begin latest: %v", err)
	}

	window := historyWindow(HistoryWindowReplace, "term-1", "tok-1", 80, 7, []HistoryRow{
		{Text: "one", LineID: 10},
		{Text: "two", LineID: 11},
	})
	store, inserted, err := store.ApplyWindow(1, window)
	if err != nil {
		t.Fatalf("apply latest: %v", err)
	}

	if inserted != 2 {
		t.Fatalf("expected 2 inserted rows, got %d", inserted)
	}
	if store.Pending != nil {
		t.Fatal("expected pending request cleared")
	}
	if store.Token != "tok-1" || store.Generation != 7 || store.Cols != 80 {
		t.Fatalf("unexpected store header %#v", store)
	}
	if len(store.SourceLines) != 2 || store.SourceLines[0].Text != "one" || store.SourceLines[1].Text != "two" {
		t.Fatalf("expected frozen source lines, got %#v", store.SourceLines)
	}
	if got := rowTexts(store.Rows); !reflect.DeepEqual(got, []string{"one", "two"}) {
		t.Fatalf("unexpected rows %v", got)
	}

	store.Rows[0].Text = "mutated"
	if window.Rows[0].Text != "one" {
		t.Fatal("latest replace must canonicalize rows from source instead of reusing protocol rows")
	}
}

func TestHistoryStoreDetachesStyledRowCells(t *testing.T) {
	store, err := (HistoryStore{}).BeginLatest(HistoryPendingRequest{
		ID:         1,
		TerminalID: "term-1",
		Cols:       80,
	})
	if err != nil {
		t.Fatalf("begin latest: %v", err)
	}
	window := historyWindow(HistoryWindowReplace, "term-1", "tok-1", 80, 7, []HistoryRow{{
		Text:   "ERR 好",
		LineID: 10,
		Cells: []HistoryCell{
			{Text: "ERR", Width: 3, Style: HistoryCellStyle{FG: "ansi:1", Bold: true}},
			{Text: " ", Width: 1},
			{Text: "好", Width: 2, Style: HistoryCellStyle{FG: "#ffcc00", Underline: true}, LinkURL: "file://build.log", LinkParams: "line=7"},
		},
	}})

	store, _, err = store.ApplyWindow(1, window)
	if err != nil {
		t.Fatalf("apply latest: %v", err)
	}
	store.Rows[0].Cells[0].Text = "mutated"
	store.Rows[0].Cells[2].Style.FG = "ansi:2"
	if window.Rows[0].Cells[0].Text != "ERR" || window.Rows[0].Cells[2].Style.FG != "#ffcc00" {
		t.Fatalf("latest canonical reflow must detach protocol row cells, window=%#v store=%#v", window.Rows[0].Cells, store.Rows[0].Cells)
	}
}

func TestHistoryStoreTakesOwnedSourceLinesAndCanonicalizesRowsWhenColsMatch(t *testing.T) {
	store, err := (HistoryStore{}).BeginLatest(HistoryPendingRequest{
		ID:         1,
		TerminalID: "term-1",
		Cols:       80,
	})
	if err != nil {
		t.Fatalf("begin latest: %v", err)
	}
	source := []HistoryLogicalLine{{
		Text:   "owned",
		LineID: 10,
		Cells:  []HistoryCell{{Text: "owned", Width: 5}},
	}}
	rows, spans := ReflowHistoryLogicalLines(source, 80)
	window := HistoryWindow{
		TerminalID:  "term-1",
		Token:       "tok-1",
		Op:          HistoryWindowReplace,
		Cols:        80,
		SourceLines: source,
		Rows:        rows,
		Lines:       spans,
		Generation:  7,
		Boundary:    HistoryBoundary{FirstLineID: 10, LastLineID: 10},
	}

	store, _, err = store.ApplyWindow(1, window)
	if err != nil {
		t.Fatalf("apply latest: %v", err)
	}
	if len(store.SourceLines) != 1 || len(store.SourceLines[0].Cells) == 0 || &store.SourceLines[0].Cells[0] != &source[0].Cells[0] {
		t.Fatalf("store should take owned source lines without deep clone, store=%#v source=%#v", store.SourceLines, source)
	}
	if len(store.Rows) != 1 || len(store.Rows[0].Cells) == 0 || &store.Rows[0].Cells[0] == &rows[0].Cells[0] {
		t.Fatalf("store should reflow canonical rows even when cols match, store=%#v rows=%#v", store.Rows, rows)
	}
}

func TestHistoryStoreRowsOnlyWindowStillDetachesCells(t *testing.T) {
	store, err := (HistoryStore{}).BeginLatest(HistoryPendingRequest{
		ID:         1,
		TerminalID: "term-1",
		Cols:       80,
	})
	if err != nil {
		t.Fatalf("begin latest: %v", err)
	}
	rows := []HistoryRow{{
		Text:   "ERR",
		LineID: 10,
		Cells:  []HistoryCell{{Text: "ERR", Width: 3, Style: HistoryCellStyle{FG: "ansi:1"}}},
	}}
	window := HistoryWindow{
		TerminalID: "term-1",
		Token:      "tok-1",
		Op:         HistoryWindowReplace,
		Cols:       80,
		Rows:       rows,
		Lines:      []HistoryLineSpan{{LineID: 10, StartRow: 0, EndRow: 0}},
		Generation: 7,
		Boundary:   HistoryBoundary{FirstLineID: 10, LastLineID: 10},
	}

	store, _, err = store.ApplyWindow(1, window)
	if err != nil {
		t.Fatalf("apply latest: %v", err)
	}
	store.SourceLines[0].Cells[0].Text = "mutated"
	store.Rows[0].Cells[0].Text = "row-mutated"
	if rows[0].Cells[0].Text != "ERR" {
		t.Fatalf("rows-only window must still detach cells, rows=%#v storeSource=%#v storeRows=%#v", rows, store.SourceLines, store.Rows)
	}
}

func TestHistoryStoreLatestReplaceReflowsFrozenSourceAtPendingCols(t *testing.T) {
	store, err := (HistoryStore{}).BeginLatest(HistoryPendingRequest{
		ID:         1,
		TerminalID: "term-1",
		Cols:       6,
	})
	if err != nil {
		t.Fatalf("begin latest: %v", err)
	}

	window := HistoryWindow{
		TerminalID: "term-1",
		Token:      "tok-1",
		Op:         HistoryWindowReplace,
		Cols:       3,
		SourceLines: []HistoryLogicalLine{{
			Text:   "abcdef",
			LineID: 10,
			Cells: []HistoryCell{
				{Text: "abc", Width: 3},
				{Text: "def", Width: 3},
			},
		}},
		Rows: []HistoryRow{
			{Text: "abc", LineID: 10, RowInLine: 0},
			{Text: "def", LineID: 10, RowInLine: 1},
		},
		Lines: []HistoryLineSpan{{LineID: 10, StartRow: 0, EndRow: 1}},
	}

	store, inserted, err := store.ApplyWindow(1, window)
	if err != nil {
		t.Fatalf("apply latest: %v", err)
	}
	if inserted != 1 {
		t.Fatalf("expected inserted rows to reflect local canonical rows, got %d", inserted)
	}
	if store.Cols != 6 {
		t.Fatalf("expected history store to keep local pending cols, got %d", store.Cols)
	}
	if got := rowTexts(store.Rows); !reflect.DeepEqual(got, []string{"abcdef"}) {
		t.Fatalf("expected frozen source to reflow at local pending cols, got %v", got)
	}
}

func TestR452HistoryStoreLatestReplaceCanonicalizesSourceRowsOnFirstEntry(t *testing.T) {
	store, err := (HistoryStore{}).BeginLatest(HistoryPendingRequest{
		ID:         1,
		TerminalID: "term-1",
		Cols:       4,
	})
	if err != nil {
		t.Fatalf("begin latest: %v", err)
	}

	window := HistoryWindow{
		TerminalID: "term-1",
		Token:      "tok-r452",
		Op:         HistoryWindowReplace,
		Cols:       4,
		SourceLines: []HistoryLogicalLine{{
			Text:   "abcdefgh",
			LineID: 10,
			Cells: []HistoryCell{
				{Text: "abcd", Width: 4},
				{Text: "efgh", Width: 4},
			},
		}},
		// 模拟 protocol 已投影 rows 和 frozen logical source 暂时不一致的首次 latest：
		// reducer 必须以 SourceLines 为本地显示真值，不能等 older 之后才重排。
		Rows: []HistoryRow{
			{Text: "abcdefgh", LineID: 10, RowInLine: 0},
		},
		Lines:      []HistoryLineSpan{{LineID: 10, StartRow: 0, EndRow: 0}},
		Generation: 7,
		Boundary:   HistoryBoundary{FirstLineID: 10, LastLineID: 10},
	}

	store, inserted, err := store.ApplyWindow(1, window)
	if err != nil {
		t.Fatalf("apply latest: %v", err)
	}
	if inserted != len(store.Rows) {
		t.Fatalf("latest inserted count should reflect canonical rows, inserted=%d rows=%d", inserted, len(store.Rows))
	}
	if got := rowTexts(store.Rows); !reflect.DeepEqual(got, []string{"abcd", "efgh"}) {
		t.Fatalf("first latest must reflow canonical logical source rows, got %v", got)
	}
	if got := spanRows(store.Lines); !reflect.DeepEqual(got, []spanRow{{id: 10, start: 0, end: 1}}) {
		t.Fatalf("first latest must rebuild spans from canonical rows, got %v", got)
	}
}

func TestHistoryStorePrependsOlderAndRebasesExistingSpans(t *testing.T) {
	store := HistoryStore{
		TerminalID: "term-1",
		Token:      "tok-1",
		Cols:       80,
		Rows: []HistoryRow{
			{Text: "new", LineID: 20},
		},
		Lines:      []HistoryLineSpan{{LineID: 20, StartRow: 0, EndRow: 0}},
		Generation: 7,
		Boundary:   HistoryBoundary{FirstLineID: 20, LastLineID: 20},
	}
	cursor := HistoryCursor{Valid: true, BeforeLineID: 20}
	store, err := store.BeginOlder(HistoryPendingRequest{
		ID:         2,
		TerminalID: "term-1",
		Cols:       80,
		Token:      "tok-1",
		Generation: 7,
		Cursor:     cursor,
		Boundary:   store.Boundary,
	})
	if err != nil {
		t.Fatalf("begin older: %v", err)
	}

	window := historyWindow(HistoryWindowPrepend, "term-1", "tok-1", 80, 7, []HistoryRow{
		{Text: "old", LineID: 10},
	})
	window.Cursor = cursor
	window.Boundary = HistoryBoundary{FirstLineID: 10, LastLineID: 20}
	window.Lines = []HistoryLineSpan{{LineID: 10, StartRow: 0, EndRow: 0}}
	store, inserted, err := store.ApplyWindow(2, window)
	if err != nil {
		t.Fatalf("apply older: %v", err)
	}

	if inserted != 1 {
		t.Fatalf("expected 1 inserted row, got %d", inserted)
	}
	if got := rowTexts(store.Rows); !reflect.DeepEqual(got, []string{"old", "new"}) {
		t.Fatalf("unexpected rows %v", got)
	}
	if got := spanRows(store.Lines); !reflect.DeepEqual(got, []spanRow{{id: 10, start: 0, end: 0}, {id: 20, start: 1, end: 1}}) {
		t.Fatalf("unexpected rebased spans %v", got)
	}
}

func TestHistoryStoreAcceptsOlderPrependFromCurrentFrameWhenTailBoundaryMatches(t *testing.T) {
	store := HistoryStore{
		TerminalID: "term-1",
		Token:      "tok-1",
		Cols:       80,
		SourceLines: []HistoryLogicalLine{
			{Text: "visible tail", LineID: 20},
			{Text: "codex current", LineID: 300001},
		},
		Rows: []HistoryRow{
			{Text: "visible tail", LineID: 20, Segment: HistoryCursorSegmentCommitted},
			{Text: "codex current", LineID: 300001, Kind: HistoryRowKindScreenFrame, Segment: HistoryCursorSegmentCurrentPrimaryFrame},
		},
		Lines:      []HistoryLineSpan{{LineID: 20, StartRow: 0, EndRow: 0}, {LineID: 300001, StartRow: 1, EndRow: 1}},
		Cursor:     HistoryCursor{Valid: true, BeforeLineID: 300001, Segment: HistoryCursorSegmentCurrentPrimaryFrame},
		Generation: 7,
		Boundary:   HistoryBoundary{FirstLineID: 20, LastLineID: 20},
	}
	store, err := store.BeginOlder(HistoryPendingRequest{
		ID:         2,
		TerminalID: "term-1",
		Cols:       80,
		Token:      "tok-1",
		Generation: 7,
		Cursor:     store.Cursor,
		Boundary:   store.Boundary,
	})
	if err != nil {
		t.Fatalf("begin older: %v", err)
	}

	window := historyWindow(HistoryWindowPrepend, "term-1", "tok-1", 80, 7, []HistoryRow{
		{Text: "older one", LineID: 10, Segment: HistoryCursorSegmentCommitted},
	})
	window.Cursor = HistoryCursor{Valid: true, BeforeLineID: 10, Segment: HistoryCursorSegmentCommitted}
	window.Boundary = HistoryBoundary{FirstLineID: 10, LastLineID: 20}
	window.Lines = []HistoryLineSpan{{LineID: 10, StartRow: 0, EndRow: 0}}
	store, inserted, err := store.ApplyWindow(2, window)
	if err != nil {
		t.Fatalf("apply older: %v", err)
	}

	if inserted != 1 {
		t.Fatalf("expected one prepended row, got %d", inserted)
	}
	if got := rowTexts(store.Rows); !reflect.DeepEqual(got, []string{"older one", "visible tail", "codex current"}) {
		t.Fatalf("current-frame older prepend should keep visible history and frame, got %v", got)
	}
	if store.Boundary.FirstLineID != 10 || store.Boundary.LastLineID != 20 {
		t.Fatalf("older prepend should extend first boundary but keep frozen tail, got %#v", store.Boundary)
	}
}

func TestHistoryStoreAppendsNewerAndKeepsFrozenBoundary(t *testing.T) {
	store := HistoryStore{
		TerminalID: "term-1",
		Token:      "tok-1",
		Cols:       80,
		Rows: []HistoryRow{
			{Text: "old", LineID: 10},
		},
		Lines:      []HistoryLineSpan{{LineID: 10, StartRow: 0, EndRow: 0}},
		Generation: 7,
		Boundary:   HistoryBoundary{FirstLineID: 10, LastLineID: 20},
	}
	store, err := store.BeginNewer(HistoryPendingRequest{
		ID:         3,
		TerminalID: "term-1",
		Cols:       80,
		Token:      "tok-1",
		Generation: 7,
		Cursor:     HistoryCursor{Valid: true, BeforeLineID: 10},
		Boundary:   store.Boundary,
	})
	if err != nil {
		t.Fatalf("begin newer: %v", err)
	}

	window := historyWindow(HistoryWindowAppend, "term-1", "tok-1", 80, 7, []HistoryRow{
		{Text: "new", LineID: 11},
		{Text: "newer", LineID: 12},
	})
	window.Boundary = HistoryBoundary{FirstLineID: 11, LastLineID: 20}
	store, inserted, err := store.ApplyWindow(3, window)
	if err != nil {
		t.Fatalf("apply newer: %v", err)
	}

	if inserted != 2 {
		t.Fatalf("expected 2 appended rows, got %d", inserted)
	}
	if got := rowTexts(store.Rows); !reflect.DeepEqual(got, []string{"old", "new", "newer"}) {
		t.Fatalf("unexpected rows %v", got)
	}
	if store.Boundary.FirstLineID != 10 || store.Boundary.LastLineID != 20 {
		t.Fatalf("append must keep frozen boundary, got %#v", store.Boundary)
	}
}

func TestHistoryStorePrependMergesBoundaryOverlapForSameLogicalLine(t *testing.T) {
	store := HistoryStore{
		TerminalID: "term-1",
		Token:      "tok-1",
		Cols:       6,
		SourceLines: []HistoryLogicalLine{{
			Text:          "ghi",
			Cells:         []HistoryCell{{Text: "ghi", Width: 3}},
			LineID:        10,
			ClippedBefore: true,
		}},
		Rows: []HistoryRow{
			{Text: "ghi", LineID: 10, RowInLine: 0, ClippedStart: true},
		},
		Lines:      []HistoryLineSpan{{LineID: 10, StartRow: 0, EndRow: 0, ClippedBefore: true}},
		Generation: 7,
		Boundary:   HistoryBoundary{FirstLineID: 10, LastLineID: 10},
	}
	cursor := HistoryCursor{Valid: true, BeforeLineID: 10, BeforeRowInLine: 0}
	store, err := store.BeginOlder(HistoryPendingRequest{
		ID:         2,
		TerminalID: "term-1",
		Cols:       6,
		Token:      "tok-1",
		Generation: 7,
		Cursor:     cursor,
		Boundary:   store.Boundary,
	})
	if err != nil {
		t.Fatalf("begin older: %v", err)
	}

	window := HistoryWindow{
		TerminalID: "term-1",
		Token:      "tok-1",
		Op:         HistoryWindowPrepend,
		Cols:       6,
		SourceLines: []HistoryLogicalLine{{
			Text:         "def",
			Cells:        []HistoryCell{{Text: "def", Width: 3}},
			LineID:       10,
			ClippedAfter: true,
		}},
		Rows: []HistoryRow{
			{Text: "def", LineID: 10, RowInLine: 0, ClippedEnd: true},
		},
		Lines:      []HistoryLineSpan{{LineID: 10, StartRow: 0, EndRow: 0, ClippedAfter: true}},
		Cursor:     HistoryCursor{},
		Generation: 7,
		Boundary:   HistoryBoundary{FirstLineID: 10, LastLineID: 10},
		HasMore:    false,
	}
	beforeRows := len(store.Rows)
	store, inserted, err := store.ApplyWindow(2, window)
	if err != nil {
		t.Fatalf("apply older: %v", err)
	}
	if inserted != len(store.Rows)-beforeRows {
		t.Fatalf("expected inserted row count to reflect real local reflow delta, got inserted=%d before=%d after=%d", inserted, beforeRows, len(store.Rows))
	}
	if len(store.SourceLines) != 1 || store.SourceLines[0].Text != "defghi" || store.SourceLines[0].ClippedBefore || store.SourceLines[0].ClippedAfter {
		t.Fatalf("boundary-overlap prepend should merge clipped source into one logical line, got %#v", store.SourceLines)
	}
	if got := rowTexts(store.Rows); !reflect.DeepEqual(got, []string{"defghi"}) {
		t.Fatalf("reflowed rows should stay stable after merged source, got %v", got)
	}
	if got := spanRows(store.Lines); !reflect.DeepEqual(got, []spanRow{{id: 10, start: 0, end: 0}}) {
		t.Fatalf("merged logical line should produce one span, got %v", got)
	}
}

func TestMergePrependedHistoryLogicalLinesPreservesOuterClippedEdges(t *testing.T) {
	merged := mergePrependedHistoryLogicalLines(
		[]HistoryLogicalLine{{Text: "middle", LineID: 10, ClippedBefore: true, ClippedAfter: true}},
		[]HistoryLogicalLine{{Text: "suffix", LineID: 10, ClippedBefore: true, ClippedAfter: true}},
	)
	if len(merged) != 1 || merged[0].Text != "middlesuffix" || !merged[0].ClippedBefore || !merged[0].ClippedAfter {
		t.Fatalf("boundary merge must preserve unresolved outer clipped edges, got %#v", merged)
	}
}

func TestHistoryStoreEnsureSourceLinesPreservesClippedFlagsFromSpans(t *testing.T) {
	store := HistoryStore{
		Rows: []HistoryRow{
			{Text: "abc", LineID: 10, RowInLine: 0, ClippedStart: true},
			{Text: "def", LineID: 10, RowInLine: 1, ClippedEnd: true},
		},
		Lines: []HistoryLineSpan{{
			LineID:        10,
			StartRow:      0,
			EndRow:        1,
			ClippedBefore: true,
			ClippedAfter:  true,
		}},
	}

	store = store.EnsureSourceLines()
	if len(store.SourceLines) != 1 {
		t.Fatalf("expected one merged source line, got %#v", store.SourceLines)
	}
	if got := store.SourceLines[0]; got.Text != "abcdef" || !got.ClippedBefore || !got.ClippedAfter {
		t.Fatalf("rows fallback must preserve clipped flags from authoritative spans, got %#v", got)
	}
}

func TestHistoryStoreRecordsOlderExhaustedMarker(t *testing.T) {
	cursor := HistoryCursor{Valid: true, BeforeLineID: 10}
	boundary := HistoryBoundary{FirstLineID: 10, LastLineID: 20}
	store := HistoryStore{
		Token:    "tok-1",
		Cols:     100,
		Cursor:   cursor,
		Boundary: boundary,
	}
	store, err := store.BeginOlder(HistoryPendingRequest{
		ID:         3,
		TerminalID: "term-1",
		Cols:       100,
		Token:      "tok-1",
		Generation: 9,
		Cursor:     cursor,
		Boundary:   boundary,
	})
	if err != nil {
		t.Fatalf("begin older: %v", err)
	}

	window := historyWindow(HistoryWindowPrepend, "term-1", "tok-1", 100, 9, nil)
	window.Cursor = cursor
	window.HasMore = false
	store, inserted, err := store.ApplyWindow(3, window)
	if err != nil {
		t.Fatalf("apply exhausted: %v", err)
	}

	if inserted != 0 {
		t.Fatalf("expected no inserted rows, got %d", inserted)
	}
	if !store.Exhausted.Valid || store.Exhausted.RequestID != 3 || store.Exhausted.Cursor != cursor || store.Exhausted.Cols != 100 {
		t.Fatalf("unexpected exhausted marker %#v", store.Exhausted)
	}
	if got := store.OlderRequestState(); got != OlderRequestExhausted {
		t.Fatalf("expected exhausted older state, got %s", got)
	}

	store.Cols = 80
	if got := store.OlderRequestState(); got != OlderRequestExhausted {
		t.Fatalf("local reflow cols must not clear frozen exhausted marker, got %s", got)
	}

	store.Token = "tok-next"
	if got := store.OlderRequestState(); got != OlderRequestReady {
		t.Fatalf("exhausted marker mismatch should fall back to current cursor readiness, got %s", got)
	}
}

func TestHistoryStoreOlderRequestStateUsesAuthoritativeBoundary(t *testing.T) {
	cursor := HistoryCursor{Valid: true, BeforeLineID: 20}
	boundary := HistoryBoundary{FirstLineID: 20, LastLineID: 30}
	store := HistoryStore{
		Token:      "tok-1",
		Cols:       80,
		Cursor:     cursor,
		Boundary:   boundary,
		HasMore:    true,
		Generation: 7,
	}
	if got := store.OlderRequestState(); got != OlderRequestReady {
		t.Fatalf("has-more window should be ready for older request, got %s", got)
	}

	store.Pending = &HistoryPendingRequest{ID: 2, Kind: HistoryRequestOlder, Token: "tok-1", Cursor: cursor, Boundary: boundary}
	if got := store.OlderRequestState(); got != OlderRequestPending {
		t.Fatalf("pending older request should win, got %s", got)
	}

	store.Pending = &HistoryPendingRequest{ID: 3, Kind: HistoryRequestLatest, Token: "tok-1"}
	if got := store.OlderRequestState(); got != OlderRequestPending {
		t.Fatalf("any pending history request should block older request, got %s", got)
	}

	store.Pending = nil
	store.HasMore = false
	if got := store.OlderRequestState(); got != OlderRequestReady {
		t.Fatalf("valid cursor should remain enough to request older even when HasMore is false, got %s", got)
	}

	store.Cursor = HistoryCursor{}
	if got := store.OlderRequestState(); got != OlderRequestMissing {
		t.Fatalf("missing authoritative cursor should not request older, got %s", got)
	}
}

func TestHistoryStoreRejectsStaleResponses(t *testing.T) {
	store, err := (HistoryStore{}).BeginOlder(HistoryPendingRequest{
		ID:         4,
		TerminalID: "term-1",
		Cols:       80,
		Token:      "tok-1",
		Generation: 7,
		Cursor:     HistoryCursor{Valid: true, BeforeLineID: 10},
	})
	if err != nil {
		t.Fatalf("begin older: %v", err)
	}

	window := historyWindow(HistoryWindowPrepend, "term-1", "tok-stale", 80, 7, []HistoryRow{{Text: "old", LineID: 1}})
	window.Cursor = HistoryCursor{Valid: true, BeforeLineID: 10}
	if _, _, err := store.ApplyWindow(4, window); !errors.Is(err, ErrStaleHistoryResponse) {
		t.Fatalf("expected stale token rejection, got %v", err)
	}

	window = historyWindow(HistoryWindowPrepend, "term-1", "tok-1", 100, 7, []HistoryRow{{Text: "old", LineID: 1}})
	window.Cursor = HistoryCursor{Valid: true, BeforeLineID: 10}
	if _, _, err := store.ApplyWindow(4, window); !errors.Is(err, ErrHistoryWindowMismatch) {
		t.Fatalf("expected cols mismatch rejection, got %v", err)
	}

	if _, _, err := store.ApplyWindow(99, window); !errors.Is(err, ErrStaleHistoryResponse) {
		t.Fatalf("expected request id rejection, got %v", err)
	}
}

func TestHistoryStoreRejectsSameTerminalWindowFromDifferentEndpoint(t *testing.T) {
	store, err := (HistoryStore{}).BeginLatest(HistoryPendingRequest{
		ID:         41,
		EndpointID: "west",
		TerminalID: "term-1",
		Cols:       80,
	})
	if err != nil {
		t.Fatalf("begin latest: %v", err)
	}

	window := historyWindow(HistoryWindowReplace, "term-1", "tok-1", 80, 1, []HistoryRow{{Text: "wrong endpoint", LineID: 1}})
	window.EndpointID = DefaultEndpointID
	if _, _, err := store.ApplyWindow(41, window); !errors.Is(err, ErrHistoryWindowMismatch) {
		t.Fatalf("expected endpoint mismatch rejection, got %v", err)
	}
}

func TestHistoryStoreRejectsDifferentViewResponse(t *testing.T) {
	store, err := (HistoryStore{}).BeginLatest(HistoryPendingRequest{
		ID:         5,
		PaneID:     "pane-1",
		ViewID:     "pane:pane-1",
		TerminalID: "term-1",
		Cols:       80,
	})
	if err != nil {
		t.Fatalf("begin latest: %v", err)
	}

	window := historyWindow(HistoryWindowReplace, "term-1", "tok-1", 80, 7, []HistoryRow{{Text: "other-view", LineID: 20}})
	window.PaneID = "pane-2"
	window.ViewID = "pane:pane-2"
	if _, _, err := store.ApplyWindow(5, window); !errors.Is(err, ErrStaleHistoryResponse) {
		t.Fatalf("expected different view response to be stale, got %v", err)
	}
}

func TestHistoryStoreAcceptsOldestReplaceForBoundFrozenToken(t *testing.T) {
	store := HistoryStore{
		TerminalID: "term-1",
		Token:      "tok-1",
		Cols:       80,
		Generation: 7,
		Boundary:   HistoryBoundary{FirstLineID: 90, LastLineID: 100},
	}
	store, err := store.BeginOldest(HistoryPendingRequest{
		ID:         6,
		TerminalID: "term-1",
		Cols:       80,
		Token:      "tok-1",
		Generation: 7,
		Boundary:   store.Boundary,
	})
	if err != nil {
		t.Fatalf("begin oldest: %v", err)
	}

	window := historyWindow(HistoryWindowReplace, "term-1", "tok-1", 80, 7, []HistoryRow{{Text: "oldest", LineID: 1}})
	store, inserted, err := store.ApplyWindow(6, window)
	if err != nil {
		t.Fatalf("apply oldest: %v", err)
	}
	if inserted != 1 || store.Pending != nil || store.Rows[0].Text != "oldest" || store.Boundary.FirstLineID != 1 {
		t.Fatalf("unexpected oldest replace state inserted=%d store=%#v", inserted, store)
	}
}

func TestHistoryStoreRejectsOldestFromDifferentFrozenToken(t *testing.T) {
	store, err := (HistoryStore{}).BeginOldest(HistoryPendingRequest{
		ID:         7,
		TerminalID: "term-1",
		Cols:       80,
		Token:      "tok-1",
		Generation: 7,
		Boundary:   HistoryBoundary{FirstLineID: 90, LastLineID: 100},
	})
	if err != nil {
		t.Fatalf("begin oldest: %v", err)
	}

	window := historyWindow(HistoryWindowReplace, "term-1", "tok-stale", 80, 7, []HistoryRow{{Text: "oldest", LineID: 1}})
	if _, _, err := store.ApplyWindow(7, window); !errors.Is(err, ErrStaleHistoryResponse) {
		t.Fatalf("expected stale oldest token rejection, got %v", err)
	}
}

func TestCopyModeBindsLatestAndAdjustsOlderViewport(t *testing.T) {
	copyMode := CopyModeStore{}.BindLatest("pane-1", "pane:pane-1", "term-1", 1, 80, 20)
	if copyMode.Active || !copyMode.Entering || !copyMode.Empty || copyMode.PaneID != "pane-1" || copyMode.ViewID != "pane:pane-1" || copyMode.TerminalID != "term-1" || copyMode.BoundCols != 80 || copyMode.ViewRows != 20 {
		t.Fatalf("unexpected bound copy mode %#v", copyMode)
	}

	latest := historyWindow(HistoryWindowReplace, "term-1", "tok-1", 80, 7, []HistoryRow{{Text: "new", LineID: 20}})
	copyMode = copyMode.AcceptLatest(latest, latest.Cols, len(latest.Rows))
	if !copyMode.Active || copyMode.Entering || copyMode.BoundToken != "tok-1" || copyMode.Empty {
		t.Fatalf("unexpected latest binding %#v", copyMode)
	}

	older := historyWindow(HistoryWindowPrepend, "term-1", "tok-1", 80, 7, []HistoryRow{{Text: "old", LineID: 10}, {Text: "older", LineID: 11}})
	before := HistoryStore{
		Cols:        80,
		SourceLines: []HistoryLogicalLine{{Text: "new", LineID: 20}},
		Rows:        []HistoryRow{{Text: "new", LineID: 20, RowInLine: 0}},
		Lines:       []HistoryLineSpan{{LineID: 20, StartRow: 0, EndRow: 0}},
	}
	after := HistoryStore{
		Cols: 100,
		SourceLines: []HistoryLogicalLine{
			{Text: "old", LineID: 10},
			{Text: "older", LineID: 11},
			{Text: "new", LineID: 20},
		},
	}
	after.Rows, after.Lines = ReflowHistoryLogicalLines(after.SourceLines, after.Cols)
	copyMode = copyMode.AcceptOlder(2, before, after, older, 100)
	if copyMode.ViewportTop != 2 || copyMode.BoundCols != 100 {
		t.Fatalf("older accept should keep local bound cols while adjusting viewport, got %#v", copyMode)
	}
}

func TestCopyModeAcceptLatestStartsAtNewestTail(t *testing.T) {
	rows := []HistoryRow{
		{Text: "one", LineID: 1},
		{Text: "two", LineID: 2},
		{Text: "three", LineID: 3},
		{Text: "four", LineID: 4},
		{Text: "five", LineID: 5},
	}
	latest := historyWindow(HistoryWindowReplace, "term-1", "tok-1", 80, 7, rows)
	copyMode := CopyModeStore{ViewRows: 3}.AcceptLatest(latest, latest.Cols, len(rows))

	if copyMode.Cursor != (CopyPosition{Row: 4}) || copyMode.ViewportTop != 2 {
		t.Fatalf("latest should start at newest visible tail, got %#v", copyMode)
	}
}

func TestCopyModeAcceptLatestAppliesPendingScrollAsCursorMove(t *testing.T) {
	rows := make([]HistoryRow, 0, 30)
	for i := 0; i < 30; i++ {
		rows = append(rows, HistoryRow{Text: "row", LineID: uint64(i + 1)})
	}
	latest := historyWindow(HistoryWindowReplace, "term-1", "tok-1", 80, 7, rows)
	copyMode := CopyModeStore{
		ViewRows:            5,
		EnteringScrollDelta: -3,
	}.AcceptLatest(latest, latest.Cols, len(rows))

	if copyMode.ViewportTop != 25 || copyMode.Cursor.Row != 26 {
		t.Fatalf("pending PageUp while latest is flying should move cursor before viewport, got %#v", copyMode)
	}
}

func TestCopyModeAcceptLatestKeepsCursorAtNewestLiveTail(t *testing.T) {
	rows := []HistoryRow{
		{Text: "shell prompt 1", LineID: 1},
		{Text: "shell prompt 2", LineID: 2},
		{Text: "Update available", LineID: 10, LiveTail: true},
		{Text: "OpenAI Codex", LineID: 11, LiveTail: true},
		{Text: "Tip: Visit the Codex community forum", LineID: 12, LiveTail: true},
		{Text: "> Explain this codebase", LineID: 13, LiveTail: true},
	}
	latest := historyWindow(HistoryWindowReplace, "term-1", "tok-1", 80, 7, rows)
	copyMode := CopyModeStore{ViewRows: 3}.AcceptLatest(latest, latest.Cols, len(rows))

	if copyMode.Cursor != (CopyPosition{Row: 5}) || copyMode.ViewportTop != 3 {
		t.Fatalf("latest should start from newest live-tail row, got %#v", copyMode)
	}
}

func TestCopyModeAcceptLatestKeepsPseudoTUILeadInContext(t *testing.T) {
	rows := []HistoryRow{
		{Text: "shell prompt", LineID: 1},
		{Text: "codex --yolo", LineID: 2},
		{Text: "", LineID: 3},
		{Text: "prelude", LineID: 4},
		{Text: "Update available! 0.141.0 -> 0.142.0", LineID: 5},
		{Text: "Run brew upgrade --cask codex to update.", LineID: 6},
		{Text: "See full release notes:", LineID: 7},
		{Text: "https://github.com/openai/codex/releases/latest", LineID: 8},
		{Text: "OpenAI Codex", LineID: 9, LiveTail: true},
		{Text: "model: gpt-5.5 xhigh", LineID: 10, LiveTail: true},
		{Text: "directory: ~/Documents/workdir/anytty", LineID: 11, LiveTail: true},
		{Text: "permissions: YOLO mode", LineID: 12, LiveTail: true},
		{Text: "Tip: Use /compact when the conversation gets long.", LineID: 13, LiveTail: true},
		{Text: "> Improve documentation in @filename", LineID: 14, LiveTail: true},
	}
	latest := historyWindow(HistoryWindowReplace, "term-1", "tok-1", 80, 7, rows)
	copyMode := CopyModeStore{ViewRows: 8}.AcceptLatest(latest, latest.Cols, len(rows))

	if copyMode.Cursor != (CopyPosition{Row: 13}) || copyMode.ViewportTop != 6 {
		t.Fatalf("latest should keep pseudo-TUI newest tail visible, got %#v", copyMode)
	}
}

func TestR413CopyModeAcceptLatestUsesAuthoritativeTailWithoutLiveSurface(t *testing.T) {
	rows := []HistoryRow{
		{Text: "old shell prompt 1", LineID: 1},
		{Text: "old shell prompt 2", LineID: 2},
		{Text: "old shell prompt 3", LineID: 3},
		{Text: "old shell prompt 4", LineID: 4},
		{Text: "old shell prompt 5", LineID: 5},
		{Text: "authoritative current 1", LineID: 6, LiveTail: true},
		{Text: "authoritative current 2", LineID: 7, LiveTail: true},
		{Text: "authoritative current 3", LineID: 8, LiveTail: true},
		{Text: "authoritative current 4", LineID: 9, LiveTail: true},
		{Text: "authoritative current 5", LineID: 10, LiveTail: true},
	}
	latest := historyWindow(HistoryWindowReplace, "term-1", "tok-1", 80, 7, rows)
	copyMode := CopyModeStore{ViewRows: 5}.AcceptLatest(latest, latest.Cols, len(rows))

	if copyMode.ViewportTop != 5 || copyMode.Cursor != (CopyPosition{Row: 9}) {
		t.Fatalf("latest should use authoritative newest tail only, got %#v", copyMode)
	}
}

func TestCopyModeAcceptLatestMapsLogicalCellAnchorAfterReflow(t *testing.T) {
	rows := []HistoryRow{
		{Text: "old shell prompt", LineID: 1, Segment: HistoryCursorSegmentCommitted},
		{Text: "old shell marker", LineID: 2, Segment: HistoryCursorSegmentCommitted},
		{Text: "abcdef", LineID: 20, RowInLine: 0, Segment: HistoryCursorSegmentCurrentPrimaryFrame},
		{Text: "ghijkl", LineID: 20, RowInLine: 1, Segment: HistoryCursorSegmentCurrentPrimaryFrame},
		{Text: "mnopqr", LineID: 20, RowInLine: 2, Segment: HistoryCursorSegmentCurrentPrimaryFrame},
	}
	latest := historyWindow(HistoryWindowReplace, "term-1", "tok-1", 80, 7, rows)
	latest.ViewportAnchor = HistoryViewportAnchor{TopLineID: 20, TopCellOffset: 6, ScreenCols: 6, ScreenRows: 2, Valid: true}
	copyMode := CopyModeStore{ViewRows: 2}.AcceptLatest(latest, latest.Cols, len(rows))

	if copyMode.ViewportTop != 3 || copyMode.Cursor != (CopyPosition{Row: 4}) {
		t.Fatalf("exact row-boundary offset must anchor the next reflow row, got %#v", copyMode)
	}
	if rows[copyMode.ViewportTop].Text != "ghijkl" {
		t.Fatalf("anchored top = %#v, want second visual row of logical line", rows[copyMode.ViewportTop])
	}
}

func TestCopyModeAcceptLatestKeepsBlankRowsBelowMiddleContent(t *testing.T) {
	rows := []HistoryRow{
		{Text: "older", LineID: 1},
		{Text: "", LineID: 20, Segment: HistoryCursorSegmentCurrentPrimaryFrame},
		{Text: "", LineID: 21, Segment: HistoryCursorSegmentCurrentPrimaryFrame},
		{Text: "middle", LineID: 22, Segment: HistoryCursorSegmentCurrentPrimaryFrame},
	}
	latest := historyWindow(HistoryWindowReplace, "term-1", "tok-1", 12, 7, rows)
	latest.ViewportAnchor = HistoryViewportAnchor{TopLineID: 20, ScreenCols: 12, ScreenRows: 5, Valid: true}

	copyMode := CopyModeStore{ViewRows: 5}.AcceptLatest(latest, latest.Cols, len(rows))
	if copyMode.ViewportTop != 1 || copyMode.ViewportTailRows != 2 {
		t.Fatalf("middle-screen anchor must preserve two trailing blank rows, got %#v", copyMode)
	}
	if rows[copyMode.ViewportTop+2].Text != "middle" {
		t.Fatalf("middle content moved inside the viewport: %#v", rows[copyMode.ViewportTop:])
	}
}

func TestCopyModeAcceptOldestStartsAtOldestHead(t *testing.T) {
	rows := []HistoryRow{
		{Text: "one", LineID: 1},
		{Text: "two", LineID: 2},
		{Text: "three", LineID: 3},
		{Text: "four", LineID: 4},
		{Text: "five", LineID: 5},
	}
	oldest := historyWindow(HistoryWindowReplace, "term-1", "tok-1", 80, 7, rows)
	copyMode := CopyModeStore{ViewRows: 3}.AcceptOldest(oldest, oldest.Cols, len(rows))

	if copyMode.Cursor != (CopyPosition{}) || copyMode.ViewportTop != 0 {
		t.Fatalf("oldest should start at oldest visible head, got %#v", copyMode)
	}
}

func TestCopyModeResizeInvalidatesBindingAndSelection(t *testing.T) {
	mark := CopyPosition{Row: 1, Col: 2}
	copyMode := CopyModeStore{
		Active:      true,
		PaneID:      "pane-1",
		ViewID:      "pane:pane-1",
		TerminalID:  "term-1",
		BoundToken:  "tok-1",
		BoundCols:   80,
		ViewRows:    20,
		ViewportTop: 4,
		Cursor:      CopyPosition{Row: 2, Col: 3},
		Mark:        &mark,
		Selection:   &CopySelection{Anchor: mark, Focus: CopyPosition{Row: 2, Col: 3}},
	}

	copyMode = copyMode.Resize(100, 30)
	if copyMode.BoundToken != "tok-1" || copyMode.BoundCols != 100 || copyMode.ViewRows != 30 || copyMode.ViewportTop != 4 {
		t.Fatalf("unexpected resized copy mode %#v", copyMode)
	}
	if copyMode.TerminalID != "term-1" || copyMode.ViewID != "pane:pane-1" || copyMode.PaneID != "pane-1" {
		t.Fatalf("resize should preserve frozen binding identity, got %#v", copyMode)
	}
	if copyMode.Mark == nil || copyMode.Selection == nil {
		t.Fatalf("state resize should not clear selection by itself, got mark=%#v selection=%#v", copyMode.Mark, copyMode.Selection)
	}
}

func TestCopyModeAcceptOlderShiftsCursorMarkAndSelectionWithPrependedRows(t *testing.T) {
	mark := CopyPosition{Row: 1, Col: 2}
	copyMode := CopyModeStore{
		Active:      true,
		Cursor:      CopyPosition{Row: 3, Col: 4},
		Mark:        &mark,
		Selection:   &CopySelection{Anchor: mark, Focus: CopyPosition{Row: 3, Col: 4}},
		ViewportTop: 2,
	}

	before := HistoryStore{
		Cols: 80,
		SourceLines: []HistoryLogicalLine{
			{Text: "line-1", LineID: 10},
			{Text: "line-2", LineID: 11},
			{Text: "line-3", LineID: 12},
			{Text: "line-4", LineID: 13},
		},
	}
	before.Rows, before.Lines = ReflowHistoryLogicalLines(before.SourceLines, before.Cols)
	after := HistoryStore{
		Cols: 90,
		SourceLines: []HistoryLogicalLine{
			{Text: "older-1", LineID: 8},
			{Text: "older-2", LineID: 9},
			{Text: "line-1", LineID: 10},
			{Text: "line-2", LineID: 11},
			{Text: "line-3", LineID: 12},
			{Text: "line-4", LineID: 13},
		},
	}
	after.Rows, after.Lines = ReflowHistoryLogicalLines(after.SourceLines, after.Cols)
	copyMode = copyMode.AcceptOlder(5, before, after, HistoryWindow{Token: "tok-older", Cols: 80}, 90)

	if copyMode.ViewportTop != 4 {
		t.Fatalf("expected viewport top to keep pointing at original content after prepend, got %d", copyMode.ViewportTop)
	}
	if copyMode.Cursor != (CopyPosition{Row: 5, Col: 4}) {
		t.Fatalf("expected cursor to keep pointing at original content after prepend, got %#v", copyMode.Cursor)
	}
	if copyMode.Mark == nil || *copyMode.Mark != (CopyPosition{Row: 3, Col: 2}) {
		t.Fatalf("expected mark to keep pointing at original content after prepend, got %#v", copyMode.Mark)
	}
	if copyMode.Selection == nil {
		t.Fatal("expected selection to be preserved")
	}
	if copyMode.Selection.Anchor != (CopyPosition{Row: 3, Col: 2}) || copyMode.Selection.Focus != (CopyPosition{Row: 5, Col: 4}) {
		t.Fatalf("expected selection to keep pointing at original content after prepend, got %#v", copyMode.Selection)
	}
	if copyMode.BoundToken != "tok-older" || copyMode.BoundCols != 90 {
		t.Fatalf("expected authoritative binding to update with older accept, got %#v", copyMode)
	}
}

func TestCopyModeAcceptOlderFastShiftsSimplePrependWithoutRebind(t *testing.T) {
	copyMode := CopyModeStore{
		Active:      true,
		Cursor:      CopyPosition{Row: 1, Col: 2},
		ViewportTop: 1,
	}
	before := HistoryStore{
		Cols: 6,
		SourceLines: []HistoryLogicalLine{
			{Text: "abcdef", LineID: 10},
			{Text: "ghijkl", LineID: 11},
		},
	}
	before.Rows, before.Lines = ReflowHistoryLogicalLines(before.SourceLines, before.Cols)
	after := HistoryStore{
		Cols: 6,
		SourceLines: []HistoryLogicalLine{
			{Text: "older", LineID: 9},
			{Text: "abcdef", LineID: 10},
			{Text: "ghijkl", LineID: 11},
		},
	}
	after.Rows, after.Lines = ReflowHistoryLogicalLines(after.SourceLines, after.Cols)

	copyMode = copyMode.AcceptOlder(1, before, after, HistoryWindow{Token: "tok-older", Cols: 6}, 6)

	if copyMode.ViewportTop != 2 || copyMode.Cursor != (CopyPosition{Row: 2, Col: 2}) {
		t.Fatalf("simple prepend should shift row positions without content rebind, got %#v", copyMode)
	}
}

func TestHistoryStoreTrimRowsReleasesWindowButKeepsFrozenTailBoundary(t *testing.T) {
	source := make([]HistoryLogicalLine, 0, 12)
	for i := 0; i < 12; i++ {
		source = append(source, HistoryLogicalLine{
			Text:   "line",
			LineID: uint64(i + 1),
		})
	}
	store := HistoryStore{
		Cols:        80,
		SourceLines: source,
		Boundary:    HistoryBoundary{FirstLineID: 1, LastLineID: 12},
	}
	store.Rows, store.Lines = ReflowHistoryLogicalLines(store.SourceLines, store.Cols)

	trimmed, result := store.TrimRows(2, 6)

	if result.DroppedRowsBefore != 2 || result.DroppedRowsAfter != 5 {
		t.Fatalf("unexpected trim accounting: %#v", result)
	}
	if len(trimmed.Rows) != 5 || len(trimmed.SourceLines) != 5 {
		t.Fatalf("trim should keep only requested local window, rows=%d sources=%d", len(trimmed.Rows), len(trimmed.SourceLines))
	}
	if trimmed.Rows[0].LineID != 3 || trimmed.Rows[len(trimmed.Rows)-1].LineID != 7 {
		t.Fatalf("trim kept wrong rows: %#v", trimmed.Rows)
	}
	if trimmed.Boundary.FirstLineID != 3 || trimmed.Boundary.LastLineID != 12 {
		t.Fatalf("trim must update local first boundary but keep frozen tail guard, got %#v", trimmed.Boundary)
	}
}

func TestHistoryStoreTrimRowsKeepsAuthoritativeCursorSegment(t *testing.T) {
	source := []HistoryLogicalLine{
		{Text: "one", LineID: 1, Segment: HistoryCursorSegmentCommitted},
		{Text: "two", LineID: 2, Segment: HistoryCursorSegmentArchivedPrimaryFrame},
		{Text: "three", LineID: 3, Segment: HistoryCursorSegmentArchivedPrimaryFrame},
	}
	store := HistoryStore{
		Cols:        80,
		SourceLines: source,
		Cursor:      HistoryCursor{Valid: true, BeforeLineID: 1, Segment: HistoryCursorSegmentArchivedPrimaryFrame},
		Boundary:    HistoryBoundary{FirstLineID: 1, LastLineID: 3},
	}
	store.Rows, store.Lines = ReflowHistoryLogicalLines(store.SourceLines, store.Cols)

	trimmed, _ := store.TrimRows(1, 2)

	if trimmed.Cursor != (HistoryCursor{Valid: true, BeforeLineID: 2, Segment: HistoryCursorSegmentArchivedPrimaryFrame}) {
		t.Fatalf("trim must keep core cursor segment while moving local boundary, got %#v", trimmed.Cursor)
	}
}

func TestHistoryStoreTrimRowsUsesLogicalLineCursor(t *testing.T) {
	source := make([]HistoryLogicalLine, 0, 8)
	for i := 0; i < 8; i++ {
		source = append(source, HistoryLogicalLine{
			Text:    "line",
			LineID:  uint64(240 + i),
			Segment: HistoryCursorSegmentCommitted,
		})
	}
	store := HistoryStore{
		Cols:        80,
		SourceLines: source,
		Cursor:      HistoryCursor{Valid: true, BeforeLineID: 240, BeforeRowInLine: 0, Segment: HistoryCursorSegmentCommitted},
		Boundary:    HistoryBoundary{FirstLineID: 240, LastLineID: 247},
	}
	store.Rows, store.Lines = ReflowHistoryLogicalLines(store.SourceLines, store.Cols)

	trimmed, result := store.TrimRows(2, 7)

	if result.DroppedRowsBefore != 2 {
		t.Fatalf("unexpected trim result %#v", result)
	}
	if trimmed.Cursor.BeforeLineID != 242 || trimmed.Cursor.BeforeRowInLine != 0 {
		t.Fatalf("trim must advance the logical-line cursor, got %#v", trimmed.Cursor)
	}
}

func TestR333HistoryStoreTrimRowsAdvancesCursorBySourceLinesNotVisualRows(t *testing.T) {
	store := HistoryStore{
		Cols: 4,
		SourceLines: []HistoryLogicalLine{
			{Text: "aaaa", LineID: 10, Segment: HistoryCursorSegmentCommitted},
			{Text: "bbbbbbbb", LineID: 11, Segment: HistoryCursorSegmentCommitted},
			{Text: "cccc", LineID: 12, Segment: HistoryCursorSegmentCommitted},
		},
		Cursor:   HistoryCursor{Valid: true, BeforeLineID: 10, Segment: HistoryCursorSegmentCommitted},
		Boundary: HistoryBoundary{FirstLineID: 10, LastLineID: 12},
	}
	store.Rows, store.Lines = ReflowHistoryLogicalLines(store.SourceLines, store.Cols)

	trimmed, result := store.TrimRows(2, len(store.Rows)-1)

	if result.DroppedRowsBefore != 1 || result.DroppedLinesBefore != 1 {
		t.Fatalf("test setup should drop one source line with one visual row, got %#v", result)
	}
	if trimmed.Cursor.BeforeLineID != 11 {
		t.Fatalf("cursor should advance by dropped source lines, got %#v", trimmed.Cursor)
	}

	trimmed, result = trimmed.TrimRows(2, len(trimmed.Rows)-1)
	if result.DroppedRowsBefore != 2 || result.DroppedLinesBefore != 1 {
		t.Fatalf("test setup should drop two visual rows from one source line, got %#v", result)
	}
	if trimmed.Cursor.BeforeLineID != 12 {
		t.Fatalf("cursor must advance by one source line, not two visual rows, got %#v", trimmed.Cursor)
	}
}

func TestR333HistoryStoreTrimRowsUsesSourceIdentityNotLineID(t *testing.T) {
	store := HistoryStore{
		Cols: 80,
		SourceLines: []HistoryLogicalLine{
			{Text: "old", LineID: 42, Kind: HistoryRowKindScreenFrame, Segment: HistoryCursorSegmentCurrentPrimaryFrame, SessionID: 3, FrameID: 10, FixedGrid: true, ScreenCols: 80},
			{Text: "new", LineID: 42, Kind: HistoryRowKindScreenFrame, Segment: HistoryCursorSegmentCurrentPrimaryFrame, SessionID: 3, FrameID: 11, FixedGrid: true, ScreenCols: 80},
		},
		Cursor:   HistoryCursor{Valid: true, BeforeLineID: 42, Segment: HistoryCursorSegmentCurrentPrimaryFrame},
		Boundary: HistoryBoundary{FirstLineID: 42, LastLineID: 42},
	}
	store.Rows, store.Lines = ReflowHistoryLogicalLines(store.SourceLines, store.Cols)

	trimmed, result := store.TrimRows(1, 1)

	if result.DroppedLinesBefore != 1 || len(trimmed.SourceLines) != 1 {
		t.Fatalf("trim should keep only the second source identity, result=%#v store=%#v", result, trimmed)
	}
	if trimmed.SourceLines[0].FrameID != 11 {
		t.Fatalf("trim must key by frame identity and logical cursor, got cursor=%#v source=%#v", trimmed.Cursor, trimmed.SourceLines)
	}
}

func TestHistoryStoreTrimRowsDetachesDroppedBackingArrays(t *testing.T) {
	source := make([]HistoryLogicalLine, 0, 64)
	for i := 0; i < 64; i++ {
		source = append(source, HistoryLogicalLine{
			Text:   "payload",
			LineID: uint64(i + 1),
			Cells:  []HistoryCell{{Text: "payload", Width: 7}},
		})
	}
	store := HistoryStore{
		Cols:        80,
		SourceLines: source,
		Boundary:    HistoryBoundary{FirstLineID: 1, LastLineID: 64},
	}
	store.Rows, store.Lines = ReflowHistoryLogicalLines(store.SourceLines, store.Cols)

	trimmed, result := store.TrimRows(24, 31)

	if result.DroppedRowsBefore != 24 || result.DroppedRowsAfter != 32 {
		t.Fatalf("unexpected trim accounting: %#v", result)
	}
	if len(trimmed.SourceLines) != 8 || cap(trimmed.SourceLines) != len(trimmed.SourceLines) {
		t.Fatalf("trimmed source lines must not retain dropped backing array, len=%d cap=%d", len(trimmed.SourceLines), cap(trimmed.SourceLines))
	}
	if len(trimmed.Rows) != 8 || cap(trimmed.Rows) != len(trimmed.Rows) {
		t.Fatalf("trimmed rows must not retain dropped backing array, len=%d cap=%d", len(trimmed.Rows), cap(trimmed.Rows))
	}
	if len(trimmed.Lines) != 8 || cap(trimmed.Lines) != len(trimmed.Lines) {
		t.Fatalf("trimmed spans must not retain dropped backing array, len=%d cap=%d", len(trimmed.Lines), cap(trimmed.Lines))
	}
	store.SourceLines[24].Cells[0].Text = "mutated-old"
	if trimmed.SourceLines[0].Cells[0].Text != "payload" {
		t.Fatalf("trimmed source cells must be detached from dropped window, got %#v", trimmed.SourceLines[0].Cells)
	}
}

func TestR333HistoryStoreAcceptsOlderPageWithOwnTailBoundary(t *testing.T) {
	store := HistoryStore{
		TerminalID: "term-1",
		Token:      "tok-1",
		Cols:       80,
		SourceLines: []HistoryLogicalLine{
			{Text: "new-1", LineID: 20, Segment: HistoryCursorSegmentCommitted},
			{Text: "new-2", LineID: 21, Segment: HistoryCursorSegmentCommitted},
		},
		Cursor:     HistoryCursor{Valid: true, BeforeLineID: 20, Segment: HistoryCursorSegmentCommitted},
		Generation: 7,
		Boundary:   HistoryBoundary{FirstLineID: 20, LastLineID: 21},
		HasMore:    true,
	}
	store.Rows, store.Lines = ReflowHistoryLogicalLines(store.SourceLines, store.Cols)
	pending := HistoryPendingRequest{
		ID:         1,
		Kind:       HistoryRequestOlder,
		TerminalID: "term-1",
		Cols:       80,
		Token:      "tok-1",
		Generation: 7,
		Cursor:     store.Cursor,
		Boundary:   store.Boundary,
	}
	var err error
	store, err = store.BeginOlder(pending)
	if err != nil {
		t.Fatalf("begin older: %v", err)
	}

	window := HistoryWindow{
		TerminalID: "term-1",
		Token:      "tok-1",
		Op:         HistoryWindowPrepend,
		Cols:       80,
		SourceLines: []HistoryLogicalLine{
			{Text: "old-1", LineID: 18, Segment: HistoryCursorSegmentCommitted},
			{Text: "old-2", LineID: 19, Segment: HistoryCursorSegmentCommitted},
		},
		Cursor:     HistoryCursor{Valid: true, BeforeLineID: 18, Segment: HistoryCursorSegmentCommitted},
		HasMore:    true,
		Generation: 7,
		Boundary:   HistoryBoundary{FirstLineID: 18, LastLineID: 19},
	}
	window.Rows, window.Lines = ReflowHistoryLogicalLines(window.SourceLines, window.Cols)

	next, inserted, err := store.ApplyWindow(1, window)
	if err != nil {
		t.Fatalf("older prepend with older tail boundary should be accepted: %v", err)
	}
	if inserted != 2 || len(next.SourceLines) != 4 || next.SourceLines[0].LineID != 18 || next.Boundary.LastLineID != 21 {
		t.Fatalf("unexpected prepended state inserted=%d store=%#v", inserted, next)
	}
}

func TestR333HistorySourceLinesDoNotMergeAcrossSegmentIdentity(t *testing.T) {
	older := []HistoryLogicalLine{{
		Text:         "frame",
		LineID:       42,
		Kind:         HistoryRowKindScreenFrame,
		Segment:      HistoryCursorSegmentCurrentPrimaryFrame,
		ClippedAfter: true,
	}}
	existing := []HistoryLogicalLine{{
		Text:          "prompt",
		LineID:        42,
		Segment:       HistoryCursorSegmentCommitted,
		ClippedBefore: true,
	}}

	merged := mergePrependedHistoryLogicalLines(older, existing)
	if len(merged) != 2 || merged[0].Text != "frame" || merged[1].Text != "prompt" {
		t.Fatalf("same line id from different segment/kind must not merge, got %#v", merged)
	}
}

func TestR333HistorySourceLinesDoNotMergeAcrossFrameIdentity(t *testing.T) {
	older := []HistoryLogicalLine{{
		Text:         "old-frame",
		LineID:       42,
		Kind:         HistoryRowKindScreenFrame,
		Segment:      HistoryCursorSegmentCurrentPrimaryFrame,
		SessionID:    3,
		FrameID:      10,
		FixedGrid:    true,
		ScreenCols:   80,
		ClippedAfter: true,
	}}
	existing := []HistoryLogicalLine{{
		Text:          "new-frame",
		LineID:        42,
		Kind:          HistoryRowKindScreenFrame,
		Segment:       HistoryCursorSegmentCurrentPrimaryFrame,
		SessionID:     3,
		FrameID:       11,
		FixedGrid:     true,
		ScreenCols:    80,
		ClippedBefore: true,
	}}

	merged := mergePrependedHistoryLogicalLines(older, existing)
	if len(merged) != 2 || merged[0].FrameID != 10 || merged[1].FrameID != 11 {
		t.Fatalf("same line id from different frame ids must not merge, got %#v", merged)
	}
}

func TestCopyModeApplyHistoryTrimRebasesCursorSelectionAndMatches(t *testing.T) {
	mark := CopyPosition{Row: 4, Col: 1}
	copyMode := CopyModeStore{
		Active:      true,
		ViewRows:    4,
		ViewportTop: 5,
		Cursor:      CopyPosition{Row: 6, Col: 2},
		Mark:        &mark,
		Selection: &CopySelection{
			Anchor:        mark,
			Focus:         CopyPosition{Row: 6, Col: 2},
			LogicalAnchor: CopyLogicalPosition{Valid: true, LineID: 40, Col: 1},
			LogicalFocus:  CopyLogicalPosition{Valid: true, LineID: 42, Col: 2},
		},
		Matches:     []CopyMatch{{StartRow: 3, StartCol: 0, EndRow: 3, EndCol: 4}, {StartRow: 6, StartCol: 1, EndRow: 6, EndCol: 3}},
		ActiveMatch: 1,
	}

	copyMode = copyMode.ApplyHistoryTrim(HistoryTrimResult{DroppedRowsBefore: 3, DroppedRowsAfter: 8}, 7)

	if copyMode.ViewportTop != 2 || copyMode.Cursor != (CopyPosition{Row: 3, Col: 2}) {
		t.Fatalf("trim should rebase viewport and cursor, got %#v", copyMode)
	}
	if copyMode.Mark == nil || *copyMode.Mark != (CopyPosition{Row: 1, Col: 1}) {
		t.Fatalf("trim should rebase mark, got %#v", copyMode.Mark)
	}
	if copyMode.Selection == nil || copyMode.Selection.Anchor != (CopyPosition{Row: 1, Col: 1}) || copyMode.Selection.Focus != (CopyPosition{Row: 3, Col: 2}) {
		t.Fatalf("trim should rebase selection, got %#v", copyMode.Selection)
	}
	if copyMode.Selection.LogicalAnchor != (CopyLogicalPosition{Valid: true, LineID: 40, Col: 1}) || copyMode.Selection.LogicalFocus != (CopyLogicalPosition{Valid: true, LineID: 42, Col: 2}) {
		t.Fatalf("trim must preserve logical selection range for backend copy, got %#v", copyMode.Selection)
	}
	if len(copyMode.Matches) != 2 || copyMode.Matches[0].StartRow != 0 || copyMode.Matches[1].StartRow != 3 {
		t.Fatalf("trim should rebase matches inside retained window, got %#v", copyMode.Matches)
	}
}

func TestCopyModeRestoresFrozenViewportTailAfterNewerPagination(t *testing.T) {
	history := HistoryStore{
		Rows: []HistoryRow{
			{Text: "older", LineID: 40},
			{Text: "top", LineID: 42},
			{Text: "middle", LineID: 43},
			{Text: "last", LineID: 44},
		},
		Boundary:       HistoryBoundary{FirstLineID: 40, LastLineID: 44},
		ViewportAnchor: HistoryViewportAnchor{Valid: true, TopLineID: 42},
	}
	copyMode := CopyModeStore{Active: true, ViewRows: 5, ViewportTailRows: 0}

	copyMode = copyMode.RestoreViewportTail(history)

	if copyMode.ViewportTailRows != 2 {
		t.Fatalf("restored viewport tail rows=%d, want 2", copyMode.ViewportTailRows)
	}
	copyMode.Cursor = CopyPosition{Row: 3}
	copyMode.ViewportTop = 0
	copyMode, consumed := copyMode.ScrollNewer(5, len(history.Rows))
	if consumed != 1 || copyMode.Cursor.Row != 3 || copyMode.ViewportTop != 1 {
		t.Fatalf("newer scroll should consume the virtual tail, consumed=%d store=%#v", consumed, copyMode)
	}
	if !copyMode.AtFrozenBottom(history) {
		t.Fatalf("cursor and viewport should reach the frozen bottom, store=%#v", copyMode)
	}
}

func TestCopyModeLogicalSelectionRefreshKeepsTrimmedAnchor(t *testing.T) {
	history := HistoryStore{
		Rows: []HistoryRow{
			{Text: "current", LineID: 30},
		},
	}
	copyMode := CopyModeStore{
		Active: true,
		Selection: &CopySelection{
			Anchor:        CopyPosition{Row: 0, Col: 0},
			Focus:         CopyPosition{Row: 0, Col: 3},
			LogicalAnchor: CopyLogicalPosition{Valid: true, LineID: 10, Col: 0},
		},
	}

	copyMode = copyMode.EnsureLogicalSelection(history)
	if copyMode.Selection.LogicalAnchor.LineID != 10 || copyMode.Selection.LogicalFocus.LineID != 30 || copyMode.Selection.LogicalFocus.Col != 3 {
		t.Fatalf("ensure should preserve existing anchor and fill missing focus, got %#v", copyMode.Selection)
	}
	copyMode.Selection.Focus = CopyPosition{Row: 0, Col: 5}
	copyMode = copyMode.RefreshLogicalSelectionFocus(history)
	if copyMode.Selection.LogicalAnchor.LineID != 10 || copyMode.Selection.LogicalFocus.LineID != 30 || copyMode.Selection.LogicalFocus.Col != 5 {
		t.Fatalf("focus refresh should update only focus endpoint, got %#v", copyMode.Selection)
	}
	mark := CopyPosition{Row: 0, Col: 0}
	copyMode.Mark = &mark
	copyMode = copyMode.MoveCursor(CopyPosition{Row: 0, Col: 6}).RefreshLogicalSelectionFocus(history)
	if copyMode.Selection.LogicalAnchor.LineID != 10 || copyMode.Selection.LogicalFocus.LineID != 30 || copyMode.Selection.LogicalFocus.Col != 6 {
		t.Fatalf("move cursor should preserve trimmed anchor and update focus, got %#v", copyMode.Selection)
	}
}

func TestCopyModeApplyDeferredOlderScrollConsumesPendingRows(t *testing.T) {
	copyMode := CopyModeStore{Active: true, ViewRows: 5, ViewportTop: 8, Cursor: CopyPosition{Row: 8, Col: 2}}
	copyMode = copyMode.ApplyDeferredOlderScroll(1, 30)

	if copyMode.ViewportTop != 7 || copyMode.Cursor != (CopyPosition{Row: 7, Col: 2}) {
		t.Fatalf("expected deferred scroll to consume one cursor row, got %#v", copyMode)
	}

	copyMode = (CopyModeStore{Active: true, ViewRows: 20}).ApplyDeferredOlderScroll(8, 30)
	if copyMode.ViewportTop != 0 || copyMode.Cursor != (CopyPosition{}) {
		t.Fatalf("deferred scroll should clamp cursor at history top, got %#v", copyMode)
	}
}

func TestCopyModeScrollCursorMovesCursorBeforeViewport(t *testing.T) {
	copyMode := CopyModeStore{
		Active:      true,
		ViewRows:    5,
		ViewportTop: 10,
		Cursor:      CopyPosition{Row: 14, Col: 3},
	}

	copyMode = copyMode.ScrollCursor(-1, 30)

	if copyMode.Cursor != (CopyPosition{Row: 13, Col: 3}) || copyMode.ViewportTop != 10 {
		t.Fatalf("cursor scroll should move cursor inside viewport before moving viewport, got %#v", copyMode)
	}

	copyMode = CopyModeStore{
		Active:      true,
		ViewRows:    5,
		ViewportTop: 10,
		Cursor:      CopyPosition{Row: 10, Col: 3},
	}.ScrollCursor(-1, 30)
	if copyMode.Cursor != (CopyPosition{Row: 9, Col: 3}) || copyMode.ViewportTop != 9 {
		t.Fatalf("cursor scroll should move viewport only after cursor crosses top edge, got %#v", copyMode)
	}

	copyMode = CopyModeStore{
		Active:      true,
		ViewRows:    5,
		ViewportTop: 10,
		Cursor:      CopyPosition{Row: 10, Col: 3},
	}.ScrollCursor(1, 30)
	if copyMode.Cursor != (CopyPosition{Row: 11, Col: 3}) || copyMode.ViewportTop != 10 {
		t.Fatalf("down scroll should move cursor inside viewport before moving viewport, got %#v", copyMode)
	}

	copyMode = CopyModeStore{
		Active:      true,
		ViewRows:    5,
		ViewportTop: 10,
		Cursor:      CopyPosition{Row: 14, Col: 3},
	}.ScrollCursor(1, 30)
	if copyMode.Cursor != (CopyPosition{Row: 15, Col: 3}) || copyMode.ViewportTop != 11 {
		t.Fatalf("down scroll should move viewport only after cursor crosses bottom edge, got %#v", copyMode)
	}
}

func TestCopyModeScrollViewportPagesWithoutStallingOnCursor(t *testing.T) {
	copyMode := CopyModeStore{
		Active:      true,
		ViewRows:    5,
		ViewportTop: 25,
		Cursor:      CopyPosition{Row: 29, Col: 3},
	}

	copyMode = copyMode.ScrollViewport(-3, 40)

	if copyMode.ViewportTop != 22 || copyMode.Cursor.Row != 26 {
		t.Fatalf("page-style scroll should move viewport first and keep cursor visible, got %#v", copyMode)
	}

	copyMode = CopyModeStore{
		Active:      true,
		ViewRows:    5,
		ViewportTop: 1,
		Cursor:      CopyPosition{Row: 1, Col: 3},
	}.ScrollViewport(-3, 40)
	if copyMode.ViewportTop != 0 || copyMode.Cursor.Row != 0 {
		t.Fatalf("viewport scroll should clamp at history top without losing visible cursor, got %#v", copyMode)
	}

	copyMode = CopyModeStore{
		Active:      true,
		ViewRows:    5,
		ViewportTop: 25,
		Cursor:      CopyPosition{Row: 25, Col: 3},
	}.ScrollViewport(3, 40)
	if copyMode.ViewportTop != 28 || copyMode.Cursor.Row != 28 {
		t.Fatalf("down page-style scroll should move viewport and keep cursor visible, got %#v", copyMode)
	}
}

func TestCopyModeRebindToReflowedHistoryKeepsCursorAndSelectionOnOriginalContent(t *testing.T) {
	before := HistoryStore{
		Cols: 3,
		SourceLines: []HistoryLogicalLine{{
			Text:   "abcdef",
			LineID: 10,
		}},
	}
	before.Rows, before.Lines = ReflowHistoryLogicalLines(before.SourceLines, before.Cols)
	after := HistoryStore{
		Cols:        6,
		SourceLines: cloneHistoryLogicalLines(before.SourceLines),
	}
	after.Rows, after.Lines = ReflowHistoryLogicalLines(after.SourceLines, after.Cols)
	mark := CopyPosition{Row: 0, Col: 1}
	copyMode := CopyModeStore{
		Active:           true,
		ViewportTop:      1,
		Cursor:           CopyPosition{Row: 1, Col: 2},
		Mark:             &mark,
		Selection:        &CopySelection{Anchor: mark, Focus: CopyPosition{Row: 1, Col: 2}},
		Query:            "ef",
		Matches:          []CopyMatch{{StartRow: 1, StartCol: 1, EndRow: 1, EndCol: 3}},
		ActiveMatch:      0,
		SearchMatchStart: CopyLogicalPosition{Valid: true, LineID: 10, Col: 4},
		SearchMatchEnd:   CopyLogicalPosition{Valid: true, LineID: 10, Col: 6},
	}

	copyMode = copyMode.RebindToReflowedHistory(before, after)

	if copyMode.ViewportTop != 0 {
		t.Fatalf("expected viewport top to rebind to containing row, got %#v", copyMode)
	}
	if copyMode.Cursor != (CopyPosition{Row: 0, Col: 5}) {
		t.Fatalf("expected cursor to keep pointing at original content after local reflow, got %#v", copyMode.Cursor)
	}
	if copyMode.Mark == nil || *copyMode.Mark != (CopyPosition{Row: 0, Col: 1}) {
		t.Fatalf("expected mark to keep pointing at original content after local reflow, got %#v", copyMode.Mark)
	}
	if copyMode.Selection == nil {
		t.Fatal("expected selection to be preserved after local reflow")
	}
	if copyMode.Selection.Anchor != (CopyPosition{Row: 0, Col: 1}) || copyMode.Selection.Focus != (CopyPosition{Row: 0, Col: 5}) {
		t.Fatalf("expected selection to rebind to original content after local reflow, got %#v", copyMode.Selection)
	}
	if len(copyMode.Matches) != 1 || copyMode.Matches[0] != (CopyMatch{StartRow: 0, StartCol: 4, EndRow: 0, EndCol: 6}) {
		t.Fatalf("expected query matches to reflow with local rows, got %#v", copyMode.Matches)
	}
	if copyMode.ActiveMatch != 0 {
		t.Fatalf("expected active match index to stay on rebound match, got %#v", copyMode)
	}
}

func TestCopyModeRebindAfterOlderBoundaryOverlapKeepsCursorAndSelectionOnOriginalContent(t *testing.T) {
	before := HistoryStore{
		Cols: 4,
		SourceLines: []HistoryLogicalLine{{
			Text:          "cdef",
			LineID:        10,
			ClippedBefore: true,
		}},
	}
	before.Rows, before.Lines = ReflowHistoryLogicalLines(before.SourceLines, before.Cols)
	after := HistoryStore{
		Cols: 4,
		SourceLines: []HistoryLogicalLine{{
			Text:          "abcdef",
			LineID:        10,
			ClippedBefore: true,
		}},
	}
	after.Rows, after.Lines = ReflowHistoryLogicalLines(after.SourceLines, after.Cols)
	mark := CopyPosition{Row: 0, Col: 1}
	copyMode := CopyModeStore{
		Active:      true,
		ViewportTop: 0,
		Cursor:      CopyPosition{Row: 0, Col: 3},
		Mark:        &mark,
		Selection:   &CopySelection{Anchor: mark, Focus: CopyPosition{Row: 0, Col: 3}},
	}

	copyMode = copyMode.RebindToReflowedHistory(before, after)

	if copyMode.Cursor != (CopyPosition{Row: 1, Col: 1}) {
		t.Fatalf("expected cursor to keep pointing at original suffix after boundary overlap, got %#v", copyMode.Cursor)
	}
	if copyMode.Mark == nil || *copyMode.Mark != (CopyPosition{Row: 0, Col: 3}) {
		t.Fatalf("expected mark to keep pointing at original suffix after boundary overlap, got %#v", copyMode.Mark)
	}
	if copyMode.Selection == nil {
		t.Fatal("expected selection after boundary overlap rebind")
	}
	if copyMode.Selection.Anchor != (CopyPosition{Row: 0, Col: 3}) || copyMode.Selection.Focus != (CopyPosition{Row: 1, Col: 1}) {
		t.Fatalf("expected selection to keep original suffix after boundary overlap, got %#v", copyMode.Selection)
	}
}

func TestHistoryStoreReflowsFrozenLogicalLinesAtNewCols(t *testing.T) {
	lines := []HistoryLogicalLine{{
		Text:   "abcdef",
		LineID: 10,
		Cells: []HistoryCell{
			{Text: "abc", Width: 3},
			{Text: "def", Width: 3},
		},
	}}
	rows, spans := ReflowHistoryLogicalLines(lines, 3)
	if got := rowTexts(rows); !reflect.DeepEqual(got, []string{"abc", "def"}) {
		t.Fatalf("expected local reflow rows, got %v", got)
	}
	if got := spanRows(spans); !reflect.DeepEqual(got, []spanRow{{id: 10, start: 0, end: 1}}) {
		t.Fatalf("expected reflowed span, got %v", got)
	}
}

func TestHistoryStoreReflowPreservesAuthoritativeCellPadding(t *testing.T) {
	lines := []HistoryLogicalLine{{
		LineID: 10,
		Cells: []HistoryCell{
			{Text: "AGENTS.md", Width: 12},
			{Text: "go.work", Width: 9},
			{Text: "README.md", Width: 9},
		},
	}}
	rows, _ := ReflowHistoryLogicalLines(lines, 40)
	if got := rows[0].Text; got != "AGENTS.md   go.work  README.md" {
		t.Fatalf("reflow text should materialize authoritative cell padding, got %q", got)
	}
}

func TestHistoryStoreDoesNotReflowScreenFrameRows(t *testing.T) {
	lines := []HistoryLogicalLine{{
		LineID: 10,
		Kind:   HistoryRowKindScreenFrame,
		Cells: []HistoryCell{
			{Text: "model:", Width: 6},
			{Text: "             ", Width: 13},
			{Text: "gpt-5.5 xhigh", Width: 13},
		},
	}}
	rows, spans := ReflowHistoryLogicalLines(lines, 12)

	if len(rows) != 1 || rows[0].Kind != HistoryRowKindScreenFrame || rows[0].Text != "model:             gpt-5.5 xhigh" {
		t.Fatalf("screen frame row must remain one fixed-grid row with padding, rows=%#v", rows)
	}
	if got := HistoryRowDisplayWidth(rows[0]); got != 32 {
		t.Fatalf("screen frame row should keep authoritative width, got %d row=%#v", got, rows[0])
	}
	if len(spans) != 1 || spans[0].Kind != HistoryRowKindScreenFrame || spans[0].StartRow != 0 || spans[0].EndRow != 0 {
		t.Fatalf("screen frame span should stay single-row, got %#v", spans)
	}
}

func TestHistoryStoreDoesNotReflowAltScreenFrameRows(t *testing.T) {
	lines := []HistoryLogicalLine{{
		LineID: 10,
		Kind:   HistoryRowKindAltScreenFrame,
		Cells: []HistoryCell{
			{Text: "Resume", Width: 6},
			{Text: "        ", Width: 8},
			{Text: "previous session", Width: 16},
		},
	}}
	rows, spans := ReflowHistoryLogicalLines(lines, 10)

	if len(rows) != 1 || rows[0].Kind != HistoryRowKindAltScreenFrame || rows[0].Text != "Resume        previous session" {
		t.Fatalf("alt-screen frame row must remain one fixed-grid row with padding, rows=%#v", rows)
	}
	if got := HistoryRowDisplayWidth(rows[0]); got != 30 {
		t.Fatalf("alt-screen frame row should keep authoritative width, got %d row=%#v", got, rows[0])
	}
	if len(spans) != 1 || spans[0].Kind != HistoryRowKindAltScreenFrame || spans[0].StartRow != 0 || spans[0].EndRow != 0 {
		t.Fatalf("alt-screen frame span should stay single-row, got %#v", spans)
	}
}

func TestHistoryStoreDoesNotReflowArchivedScreenFrameRows(t *testing.T) {
	line := HistoryLogicalLine{
		LineID: 7,
		Kind:   HistoryRowKindArchivedScreenFrame,
		Cells: []HistoryCell{
			{Text: "model:", Width: 6},
			{Text: "             ", Width: 13},
			{Text: "gpt-5.5 xhigh", Width: 13},
		},
	}

	rows, spans := ReflowHistoryLogicalLines([]HistoryLogicalLine{line}, 12)
	if len(rows) != 1 || rows[0].Kind != HistoryRowKindArchivedScreenFrame || rows[0].Text != "model:             gpt-5.5 xhigh" {
		t.Fatalf("archived screen frame row must remain fixed-grid with padding, rows=%#v", rows)
	}
	if len(spans) != 1 || spans[0].Kind != HistoryRowKindArchivedScreenFrame || spans[0].StartRow != 0 || spans[0].EndRow != 0 {
		t.Fatalf("archived screen frame span should stay single-row, got %#v", spans)
	}
}

func TestHistoryStorePreservesBlankScreenFrameRows(t *testing.T) {
	lines := []HistoryLogicalLine{
		{LineID: 10, Kind: HistoryRowKindScreenFrame, Cells: []HistoryCell{{Text: "Update available!", Width: 17}}},
		{LineID: 11, Kind: HistoryRowKindScreenFrame},
		{LineID: 12, Kind: HistoryRowKindScreenFrame, Cells: []HistoryCell{{Text: "OpenAI Codex", Width: 12}}},
	}
	rows, spans := ReflowHistoryLogicalLines(lines, 12)

	if got := rowTexts(rows); !reflect.DeepEqual(got, []string{"Update available!", "", "OpenAI Codex"}) {
		t.Fatalf("blank screen frame row should survive reflow, got %q rows=%#v", got, rows)
	}
	for index, row := range rows {
		if row.Kind != HistoryRowKindScreenFrame {
			t.Fatalf("row %d should stay screen-frame, got %#v", index, rows)
		}
	}
	if len(spans) != 3 || spans[1].Kind != HistoryRowKindScreenFrame || spans[1].StartRow != 1 || spans[1].EndRow != 1 {
		t.Fatalf("blank screen-frame span should stay single-row, got %#v", spans)
	}
}

func TestHistoryStoreReflowSplitsLongCellsAndKeepsLsSpacing(t *testing.T) {
	lines := []HistoryLogicalLine{{
		LineID: 10,
		Cells: []HistoryCell{
			{Text: "AGENTS.md   go.work.sum   anytty-tui", Width: 35},
		},
	}}
	rows, spans := ReflowHistoryLogicalLines(lines, 12)
	if got := rowTexts(rows); !reflect.DeepEqual(got, []string{"AGENTS.md   ", "go.work.sum ", "  anytty-tui"}) {
		t.Fatalf("reflow should split long cells by display cols without losing ls spacing, got %q", got)
	}
	if got := spanRows(spans); !reflect.DeepEqual(got, []spanRow{{id: 10, start: 0, end: 2}}) {
		t.Fatalf("expected one logical line span across wrapped ls rows, got %v", got)
	}
}

func TestHistoryStoreReflowsFrozenLogicalLineTextWithoutCellsAtNewCols(t *testing.T) {
	lines := []HistoryLogicalLine{{
		Text:   "abcdef",
		LineID: 10,
	}}
	rows, spans := ReflowHistoryLogicalLines(lines, 3)
	if got := rowTexts(rows); !reflect.DeepEqual(got, []string{"abc", "def"}) {
		t.Fatalf("text-only frozen source should still reflow by display cols, got %v", got)
	}
	if got := spanRows(spans); !reflect.DeepEqual(got, []spanRow{{id: 10, start: 0, end: 1}}) {
		t.Fatalf("expected text-only frozen source span, got %v", got)
	}
}

func TestHistoryStoreReflowTextOnlyASCIIFastPathKeepsWrap(t *testing.T) {
	lines := []HistoryLogicalLine{{
		Text:   "0123456789",
		LineID: 10,
	}}
	rows, spans := ReflowHistoryLogicalLines(lines, 4)
	if got := rowTexts(rows); !reflect.DeepEqual(got, []string{"0123", "4567", "89"}) {
		t.Fatalf("ascii fast path should wrap by local cols, got %v", got)
	}
	if got := spanRows(spans); !reflect.DeepEqual(got, []spanRow{{id: 10, start: 0, end: 2}}) {
		t.Fatalf("expected ascii source span, got %v", got)
	}
}

func TestHistoryInvalidateWindowClearsAuthoritativeRowsAndPending(t *testing.T) {
	store := HistoryStore{
		TerminalID: "term-1",
		Token:      "tok-1",
		Cols:       80,
		Rows:       []HistoryRow{{Text: "old", LineID: 10}},
		Lines:      []HistoryLineSpan{{LineID: 10, StartRow: 0, EndRow: 0}},
		Cursor:     HistoryCursor{Valid: true, BeforeLineID: 10},
		Generation: 7,
		Boundary:   HistoryBoundary{FirstLineID: 10, LastLineID: 10},
		HasMore:    true,
		Exhausted:  ExhaustedMarker{Valid: true, Token: "tok-1", Cols: 80},
		Pending:    &HistoryPendingRequest{ID: 9, TerminalID: "term-1", Cols: 80},
	}

	store = store.InvalidateWindow()
	if store.Token != "" || store.Cols != 0 || len(store.Rows) != 0 || len(store.Lines) != 0 || store.Pending != nil {
		t.Fatalf("history invalidate must clear window and pending request, got %#v", store)
	}
	if store.TerminalID != "term-1" {
		t.Fatalf("history invalidate should keep terminal binding, got %#v", store)
	}
}

func TestCopyModeSelectionFollowsMarkAndCursor(t *testing.T) {
	copyMode := CopyModeStore{}.SetMark(CopyPosition{Row: 1, Col: 0})
	copyMode = copyMode.MoveCursor(CopyPosition{Row: 2, Col: 4})

	if copyMode.Selection == nil {
		t.Fatal("expected selection")
	}
	if copyMode.Selection.Anchor != (CopyPosition{Row: 1, Col: 0}) || copyMode.Selection.Focus != (CopyPosition{Row: 2, Col: 4}) {
		t.Fatalf("unexpected selection %#v", copyMode.Selection)
	}
}

func TestCopyModeClearSelectionKeepsHistoryModeActive(t *testing.T) {
	copyMode := CopyModeStore{Active: true, Phase: CopyModeFrozenHistory, BoundToken: "tok-1"}.
		SetMark(CopyPosition{Row: 1, Col: 2})
	copyMode.CopyExitAfterSuccess = true

	copyMode = copyMode.ClearSelection()

	if !copyMode.HistoryRenderable() || !copyMode.Active || copyMode.BoundToken != "tok-1" {
		t.Fatalf("clearing a selection must keep frozen history active, got %#v", copyMode)
	}
	if copyMode.Mark != nil || copyMode.Selection != nil || copyMode.CopyExitAfterSuccess {
		t.Fatalf("clearing a selection must reset only selection state, got %#v", copyMode)
	}
}

func TestCopyModeQueryAndScrollClamp(t *testing.T) {
	history := HistoryStore{Rows: []HistoryRow{{Text: "alpha beta", LineID: 1}, {Text: "beta gamma", LineID: 2}, {Text: "delta", LineID: 3}}}
	copyMode := (CopyModeStore{ViewRows: 3}).SetQuery("beta", []CopyMatch{{StartRow: 0, StartCol: 6, EndRow: 0, EndCol: 10}})
	if copyMode.Cursor.Row != 0 || copyMode.Cursor.Col != 6 {
		t.Fatalf("expected cursor on first match, got %#v", copyMode)
	}
	copyMode = copyMode.Scroll(20, len(history.Rows))
	if copyMode.ViewportTop != 0 {
		t.Fatalf("scroll should clamp to max top for fully visible rows, got %#v", copyMode)
	}
}

func TestHistoryRowGraphemeDisplayColumnsUsesAuthoritativeCellWidth(t *testing.T) {
	row := HistoryRow{Text: "abx", Cells: []HistoryCell{
		{Text: "ab", Width: 4},
		{Text: "x", Width: 1},
	}}
	if got := HistoryRowGraphemeDisplayColumns(row); !reflect.DeepEqual(got, []int{0, 2, 4, 5}) {
		t.Fatalf("unexpected authoritative grapheme columns %#v", got)
	}
}

func TestHistoryRowSliceDisplayUsesDisplayColumns(t *testing.T) {
	row := HistoryRow{Text: "a好bc", Cells: []HistoryCell{
		{Text: "a", Width: 1},
		{Text: "好", Width: 2},
		{Text: "bc", Width: 2},
	}}
	if got := HistoryRowDisplayWidth(row); got != 5 {
		t.Fatalf("unexpected display width %d", got)
	}
	if got := HistoryRowSliceDisplay(row, 1, 4); got != "好b" {
		t.Fatalf("display slice should include full wide cell and following ASCII, got %q", got)
	}
}

func TestHistoryRowSliceDisplayUsesAuthoritativeCellWidth(t *testing.T) {
	row := HistoryRow{Text: "ab", Cells: []HistoryCell{
		{Text: "ab", Width: 4},
	}}
	if got := HistoryRowDisplayWidth(row); got != 4 {
		t.Fatalf("display width should use authoritative cell width, got %d", got)
	}
	if got := HistoryRowSliceDisplay(row, 0, 4); got != "ab  " {
		t.Fatalf("full-width slice should preserve authoritative padding, got %q", got)
	}
}

func TestHistoryRowSliceDisplayMaterializesAuthoritativePadding(t *testing.T) {
	row := HistoryRow{Text: "ab  cd", Cells: []HistoryCell{
		{Text: "ab", Width: 4},
		{Text: "cd", Width: 2},
	}}
	if got := HistoryRowSliceDisplay(row, 0, 6); got != "ab  cd" {
		t.Fatalf("full-width slice should materialize padded cells, got %q", got)
	}
	if got := HistoryRowSliceDisplay(row, 2, 5); got != "  c" {
		t.Fatalf("padding slice should expose blank display cells, got %q", got)
	}
}

func TestReflowHistoryLogicalLinesPreservesEmptyStyledCells(t *testing.T) {
	style := HistoryCellStyle{BG: "ansi:4"}
	rows, _ := ReflowHistoryLogicalLines([]HistoryLogicalLine{{
		LineID: 42,
		Cells: []HistoryCell{
			{Text: "X", Width: 1},
			{Text: "", Width: 3, Style: style},
			{Text: "Y", Width: 1},
		},
	}}, 10)

	if len(rows) != 1 || rows[0].Text != "X   Y" || HistoryRowDisplayWidth(rows[0]) != 5 {
		t.Fatalf("empty styled cell should materialize as blank display columns, got %#v", rows)
	}
	if len(rows[0].Cells) != 3 || rows[0].Cells[1].Text != "   " || rows[0].Cells[1].Width != 3 || rows[0].Cells[1].Style != style {
		t.Fatalf("empty styled cell should keep its visual footprint and style, got %#v", rows[0].Cells)
	}

	rows, _ = ReflowHistoryLogicalLines([]HistoryLogicalLine{{
		LineID: 43,
		Cells: []HistoryCell{
			{Text: "X", Width: 1},
			{Text: "", Width: 3, Style: style},
			{Text: "Y", Width: 1},
		},
	}}, 2)
	if got := rowTexts(rows); len(got) != 3 || got[0] != "X " || got[1] != "  " || got[2] != "Y" {
		t.Fatalf("empty styled footprint should wrap by display columns, got rows=%#v texts=%#v", rows, got)
	}
}

func TestReflowHistoryLogicalLinesTailFillDoesNotAddRowsOnResize(t *testing.T) {
	style := HistoryCellStyle{BG: "idx:24"}
	rows, _ := ReflowHistoryLogicalLines([]HistoryLogicalLine{{
		LineID:   42,
		Cells:    []HistoryCell{{Text: "abcdefghij", Width: 10, Style: style}},
		TailFill: &style,
	}}, 8)

	if got := rowTexts(rows); len(got) != 2 || got[0] != "abcdefgh" || got[1] != "ij" {
		t.Fatalf("tail fill must not materialize logical blank rows, got rows=%#v texts=%#v", rows, got)
	}
	if rows[0].TailFill != nil {
		t.Fatalf("tail fill should not attach to non-final reflow row, got %#v", rows[0])
	}
	if rows[1].TailFill == nil || rows[1].TailFill.BG != "idx:24" {
		t.Fatalf("final row should keep tail fill metadata, got %#v", rows[1])
	}
}

func TestCopyModeAppendSearchResultsKeepsLargeCatalogOffTheVisibleProjection(t *testing.T) {
	const total = 100_000
	results := make([]CopyLogicalMatch, total)
	for index := range results {
		lineID := uint64(index + 1)
		results[index] = CopyLogicalMatch{
			Start: CopyLogicalPosition{Valid: true, LineID: lineID},
			End:   CopyLogicalPosition{Valid: true, LineID: lineID, Col: 1},
		}
	}
	history := HistoryStore{Rows: make([]HistoryRow, 24)}
	for index := range history.Rows {
		history.Rows[index] = HistoryRow{Text: "a", LineID: uint64(50_000 + index)}
	}
	store := CopyModeStore{
		Active:           true,
		ViewRows:         24,
		SearchMatchStart: CopyLogicalPosition{Valid: true, LineID: 50_010},
		SearchMatchEnd:   CopyLogicalPosition{Valid: true, LineID: 50_010, Col: 1},
	}

	next, err := store.AppendSearchResults(history, results)
	if err != nil {
		t.Fatal(err)
	}
	if len(next.SearchResults) != total {
		t.Fatalf("catalog count = %d, want %d", len(next.SearchResults), total)
	}
	if len(next.Matches) != 24 {
		t.Fatalf("visible projection should contain only 24 viewport matches, got %d", len(next.Matches))
	}
	if current, gotTotal := next.SearchResultPosition(); current != 50_010 || gotTotal != total {
		t.Fatalf("search result position = %d/%d, want 50010/%d", current, gotTotal, total)
	}
}

func TestCopyModeSearchQueryEditingUsesCursorPosition(t *testing.T) {
	store := CopyModeStore{Query: "FINDME", SearchCursor: 3}
	store = store.DeleteSearchForward().InsertSearchText("X")
	if store.Query != "FINXME" || store.SearchCursor != 4 {
		t.Fatalf("replace middle query character = %#v", store)
	}
	store = store.MoveSearchCursor(-2).DeleteSearchBackward()
	if store.Query != "FNXME" || store.SearchCursor != 1 {
		t.Fatalf("backspace at cursor = %#v", store)
	}
	store = store.SetSearchCursor(0).DeleteSearchBackward()
	if store.Query != "FNXME" || store.SearchCursor != 0 {
		t.Fatalf("backspace at start changed query = %#v", store)
	}
	store = store.SetSearchCursor(999).DeleteSearchForward()
	if store.Query != "FNXME" || store.SearchCursor != len([]rune(store.Query)) {
		t.Fatalf("delete at end changed query = %#v", store)
	}
}

func historyWindow(
	op HistoryWindowOp,
	terminalID string,
	token string,
	cols int,
	generation uint64,
	rows []HistoryRow,
) HistoryWindow {
	firstLine := uint64(0)
	lastLine := uint64(0)
	if len(rows) > 0 {
		firstLine = rows[0].LineID
		lastLine = rows[len(rows)-1].LineID
	}
	window := HistoryWindow{
		TerminalID:  terminalID,
		Token:       token,
		Op:          op,
		Cols:        cols,
		SourceLines: historyLogicalLinesFromRows(rows),
		Rows:        rows,
		Generation:  generation,
		Boundary:    HistoryBoundary{FirstLineID: firstLine, LastLineID: lastLine},
	}
	if len(rows) > 0 {
		window.Lines = []HistoryLineSpan{{LineID: firstLine, StartRow: 0, EndRow: len(rows) - 1}}
	}
	return window
}

func historyLogicalLinesFromRows(rows []HistoryRow) []HistoryLogicalLine {
	if len(rows) == 0 {
		return nil
	}
	lines := make([]HistoryLogicalLine, len(rows))
	for i, row := range rows {
		lines[i] = HistoryLogicalLine{
			Text:   row.Text,
			Cells:  cloneHistoryCells(row.Cells),
			LineID: row.LineID,
		}
	}
	return lines
}

type spanRow struct {
	id    uint64
	start int
	end   int
}

func rowTexts(rows []HistoryRow) []string {
	texts := make([]string, len(rows))
	for i, row := range rows {
		texts[i] = row.Text
	}
	return texts
}

func spanRows(spans []HistoryLineSpan) []spanRow {
	out := make([]spanRow, len(spans))
	for i, span := range spans {
		out[i] = spanRow{id: span.LineID, start: span.StartRow, end: span.EndRow}
	}
	return out
}

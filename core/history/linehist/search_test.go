package linehist

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/anytty/anytty/core/history"
	vterm "github.com/anytty/anytty/vterm/vterm"
)

func searchScreenRow(text string) ScreenRow {
	if text == "" {
		return ScreenRow{}
	}
	return ScreenRow{Cells: []vterm.Cell{{Content: text, Width: len(text)}}}
}

func TestStoreSearchSupportsChineseGlobRegexAndContext(t *testing.T) {
	harness := newStoreHarness(t, 40, 3)
	harness.write("before\r\n构建-失败-42\r\nafter\r\n")
	frozen, err := harness.store.Freeze(history.FreezeHistoryRequest{Cols: 40, Limit: 3})
	if err != nil {
		t.Fatal(err)
	}

	glob, err := harness.store.Search(context.Background(), history.HistorySearchRequest{
		TerminalID: "term-store", Token: frozen.Token, Cols: 40, Limit: 2,
		Query: "构建-*-[0-9][0-9]", Mode: history.HistorySearchModeGlob,
		Direction: history.HistorySearchForward, Start: history.HistoryCopyPosition{LineID: 1}, ContextBefore: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !glob.Found || glob.Match.Start.LineID != 2 || len(glob.Window.Rows) == 0 || glob.Window.Rows[0].LineID != 1 {
		t.Fatalf("glob search with context = %#v", glob)
	}

	regex, err := harness.store.Search(context.Background(), history.HistorySearchRequest{
		TerminalID: "term-store", Token: frozen.Token, Cols: 40, Limit: 3,
		Query: "失败-[0-9]+", Mode: history.HistorySearchModeRegex,
		Direction: history.HistorySearchForward, Start: history.HistoryCopyPosition{LineID: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !regex.Found || regex.Match.Start.LineID != 2 || regex.Match.Start.Col != 5 || regex.Match.End.Col != 12 {
		t.Fatalf("regex display-cell match = %#v", regex.Match)
	}

	_, err = harness.store.Search(context.Background(), history.HistorySearchRequest{
		TerminalID: "term-store", Token: frozen.Token, Cols: 40, Limit: 3,
		Query: "[", Mode: history.HistorySearchModeRegex, Direction: history.HistorySearchForward,
	})
	if !errors.Is(err, history.ErrHistoryInvalidSearchPattern) {
		t.Fatalf("invalid regex error = %v", err)
	}
}

func TestStoreSearchCoversPersistedPendingOpenTailAndCurrentScreen(t *testing.T) {
	file := openTestLineStorage(t, t.TempDir(), "search-frontier")
	engine := NewEngine(file)
	store := NewStore("search-frontier", engine)
	var gate sync.Mutex
	store.Bind(func() ScreenSnapshot {
		return ScreenSnapshot{
			Cols:         80,
			ViewportRows: 2,
			Rows: []ScreenRow{
				searchScreenRow("screen continuation"),
				searchScreenRow("current-screen-needle"),
			},
		}
	}, &gate)
	t.Cleanup(func() { _ = store.Close() })

	if err := engine.ApplyEvictedRows([]vterm.TerminalSemanticScrollOut{evictedRowForTest("persisted-needle", false)}); err != nil {
		t.Fatal(err)
	}
	if err := engine.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := engine.ApplyEvictedRows([]vterm.TerminalSemanticScrollOut{
		evictedRowForTest("pending-block-needle", false),
		evictedRowForTest("open-tail-needle ", true),
	}); err != nil {
		t.Fatal(err)
	}
	if file.persistedLines != 1 || len(file.pending) != 1 || len(engine.OpenTail()) == 0 {
		t.Fatalf("test setup did not create all storage layers: persisted=%d pending=%d open=%d", file.persistedLines, len(file.pending), len(engine.OpenTail()))
	}

	frozen, err := store.Freeze(history.FreezeHistoryRequest{Cols: 80, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		query      string
		wantLineID history.LogicalLineID
		wantWrap   bool
	}{
		{query: "current-screen-needle", wantLineID: 4},
		{query: "open-tail-needle", wantLineID: 3},
		{query: "pending-block-needle", wantLineID: 2, wantWrap: true},
		{query: "persisted-needle", wantLineID: 1, wantWrap: true},
	} {
		t.Run(test.query, func(t *testing.T) {
			result, err := store.Search(context.Background(), history.HistorySearchRequest{
				TerminalID: "search-frontier",
				Token:      frozen.Token,
				Cols:       80,
				Limit:      10,
				Query:      test.query,
				Direction:  history.HistorySearchForward,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !result.Found || result.Match.Start.LineID != test.wantLineID || result.Wrapped != test.wantWrap {
				t.Fatalf("search %q = %#v, want line=%d wrapped=%v", test.query, result, test.wantLineID, test.wantWrap)
			}
		})
	}

	explicitOldest, err := store.Search(context.Background(), history.HistorySearchRequest{
		TerminalID: "search-frontier",
		Token:      frozen.Token,
		Cols:       80,
		Limit:      10,
		Query:      "persisted-needle",
		Direction:  history.HistorySearchForward,
		Start:      history.HistoryCopyPosition{LineID: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !explicitOldest.Found || explicitOldest.Match.Start.LineID != 1 || explicitOldest.Wrapped {
		t.Fatalf("explicit cursor must keep logical search order: %#v", explicitOldest)
	}
}

func TestStoreSearchIncludesOnlyCurrentAlternateScreenFrame(t *testing.T) {
	file := openTestLineStorage(t, t.TempDir(), "search-alt-frame")
	store := NewStore("search-alt-frame", NewEngine(file))
	var gate sync.Mutex
	snapshot := ScreenSnapshot{
		Cols:         80,
		ViewportRows: 2,
		InAlt:        true,
		PrimaryRows:  []ScreenRow{searchScreenRow("saved-primary")},
		Rows:         []ScreenRow{searchScreenRow("current-alt-needle")},
	}
	store.Bind(func() ScreenSnapshot { return snapshot }, &gate)
	t.Cleanup(func() { _ = store.Close() })

	current, err := store.Freeze(history.FreezeHistoryRequest{Cols: 80, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Search(context.Background(), history.HistorySearchRequest{
		TerminalID: "search-alt-frame", Token: current.Token, Cols: 80, Limit: 10,
		Query: "current-alt-needle", Direction: history.HistorySearchForward,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Found {
		t.Fatal("current alternate-screen frame must be searchable")
	}

	gate.Lock()
	snapshot = ScreenSnapshot{
		Cols:         80,
		ViewportRows: 2,
		Rows:         []ScreenRow{searchScreenRow("saved-primary")},
	}
	gate.Unlock()
	next, err := store.Freeze(history.FreezeHistoryRequest{Cols: 80, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	result, err = store.Search(context.Background(), history.HistorySearchRequest{
		TerminalID: "search-alt-frame", Token: next.Token, Cols: 80, Limit: 10,
		Query: "current-alt-needle", Direction: history.HistorySearchForward,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Found {
		t.Fatalf("an exited alternate-screen frame must not become history: %#v", result)
	}

	result, err = store.Search(context.Background(), history.HistorySearchRequest{
		TerminalID: "search-alt-frame", Token: current.Token, Cols: 80, Limit: 10,
		Query: "current-alt-needle", Direction: history.HistorySearchForward,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Found {
		t.Fatal("the original frozen token must retain its current alternate-screen frame")
	}
}

func TestStoreSearchFrozenHistoryAcrossColdAndHotRows(t *testing.T) {
	harness := newStoreHarness(t, 24, 3)
	for index := 1; index <= 20; index++ {
		text := fmt.Sprintf("line-%02d", index)
		if index == 4 || index == 18 {
			text += " needle"
		}
		harness.write(text + "\r\n")
	}
	frozen, err := harness.store.Freeze(history.FreezeHistoryRequest{Cols: 24, Limit: 6})
	if err != nil {
		t.Fatal(err)
	}
	result, err := harness.store.Search(context.Background(), history.HistorySearchRequest{
		TerminalID: "term-store", Token: frozen.Token, Cols: 24, Limit: 6,
		Query: "needle", Direction: history.HistorySearchForward,
		Start: history.HistoryCopyPosition{LineID: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Found || result.Match.Start.LineID != 18 || result.Wrapped {
		t.Fatalf("forward search = %#v", result)
	}
	if result.Window.Token != frozen.Token || result.Window.LogicalTotal != 20 {
		t.Fatalf("search window lost frozen metadata: %#v", result.Window)
	}

	wrapped, err := harness.store.Search(context.Background(), history.HistorySearchRequest{
		TerminalID: "term-store", Token: frozen.Token, Cols: 24, Limit: 6,
		Query: "needle", Direction: history.HistorySearchForward,
		Start: history.HistoryCopyPosition{LineID: 18, Col: result.Match.Start.Col + 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !wrapped.Found || wrapped.Match.Start.LineID != 4 || !wrapped.Wrapped {
		t.Fatalf("wrapped search = %#v", wrapped)
	}
}

func TestStoreSearchReportsDisplayCellColumns(t *testing.T) {
	harness := newStoreHarness(t, 24, 3)
	harness.write("a你b needle\r\n")
	frozen, err := harness.store.Freeze(history.FreezeHistoryRequest{Cols: 24, Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	result, err := harness.store.Search(context.Background(), history.HistorySearchRequest{
		TerminalID: "term-store", Token: frozen.Token, Cols: 24, Limit: 3,
		Query: "needle", Direction: history.HistorySearchForward,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Found || result.Match.Start.Col != 5 || result.Match.End.Col != 11 {
		t.Fatalf("display-cell match = %#v", result.Match)
	}
}

func TestStoreSearchBackwardFromLineStartSkipsLaterMatchOnCurrentLine(t *testing.T) {
	harness := newStoreHarness(t, 24, 3)
	harness.write("previous needle\r\ncurrent needle\r\n")
	frozen, err := harness.store.Freeze(history.FreezeHistoryRequest{Cols: 24, Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	result, err := harness.store.Search(context.Background(), history.HistorySearchRequest{
		TerminalID: "term-store", Token: frozen.Token, Cols: 24, Limit: 3,
		Query: "needle", Direction: history.HistorySearchBackward,
		Start: history.HistoryCopyPosition{LineID: 2, Col: 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Found || result.Match.Start.LineID != 1 || result.Wrapped {
		t.Fatalf("backward search from line start = %#v", result)
	}
}

func TestStoreSearchWindowAlwaysContainsMatchUnderByteBudget(t *testing.T) {
	harness := newStoreHarness(t, 80, 3)
	filler := strings.Repeat("x", 25_000)
	for index := 0; index < 4; index++ {
		harness.write(fmt.Sprintf("%d-%s\r\n", index, filler))
	}
	harness.write("needle\r\n")
	frozen, err := harness.store.Freeze(history.FreezeHistoryRequest{Cols: 80, Limit: 6})
	if err != nil {
		t.Fatal(err)
	}
	result, err := harness.store.Search(context.Background(), history.HistorySearchRequest{
		TerminalID: "term-store", Token: frozen.Token, Cols: 80, Limit: 6,
		Query: "needle", Direction: history.HistorySearchForward,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Found {
		t.Fatal("expected search match")
	}
	foundLine := false
	for _, row := range result.Window.Rows {
		if row.LineID == result.Match.Start.LineID {
			foundLine = true
			break
		}
	}
	if !foundLine {
		t.Fatalf("search window omitted match line: match=%#v rows=%d", result.Match, len(result.Window.Rows))
	}
}

func TestStoreSearchHonorsCancelledContext(t *testing.T) {
	harness := newStoreHarness(t, 24, 3)
	for index := 0; index < 600; index++ {
		harness.write(fmt.Sprintf("line-%04d\r\n", index))
	}
	frozen, err := harness.store.Freeze(history.FreezeHistoryRequest{Cols: 24, Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = harness.store.Search(ctx, history.HistorySearchRequest{
		TerminalID: "term-store", Token: frozen.Token, Cols: 24, Limit: 3,
		Query: "missing", Direction: history.HistorySearchForward,
	})
	if err != context.Canceled {
		t.Fatalf("search error = %v, want context canceled", err)
	}
}

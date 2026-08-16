package linehist

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/anytty/anytty/core/history"
)

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

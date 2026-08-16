package linehist

import (
	"errors"
	"strings"
	"testing"

	"github.com/anytty/anytty/core/history"
)

func TestHistoryWindowLimitAndByteBudgetCannotBeBypassed(t *testing.T) {
	store := newHistoryBudgetStore(t, "window-budget")
	lines := []string{
		strings.Repeat("a", 25<<10),
		strings.Repeat("b", 25<<10),
		strings.Repeat("c", 25<<10),
		strings.Repeat("d", 25<<10),
	}
	if err := store.AppendLifecycleLines(lines); err != nil {
		t.Fatal(err)
	}
	for _, limit := range []int{0, history.MaxHistoryWindowLines + 1} {
		if _, err := store.LatestWindow(history.HistoryWindowRequest{Limit: limit}); !errors.Is(err, history.ErrHistoryWindowLimit) {
			t.Fatalf("limit %d error = %v, want window limit", limit, err)
		}
	}

	latest, err := store.LatestWindow(history.HistoryWindowRequest{Limit: 4})
	if err != nil {
		t.Fatal(err)
	}
	if got := historyLineIDs(latest.Rows); !equalLineIDs(got, []history.LogicalLineID{3, 4}) {
		t.Fatalf("byte-truncated latest IDs = %v, want [3 4]", got)
	}
	if !latest.HasMore || !latest.Boundary.Cursor.Valid || latest.Boundary.Cursor.LineID != 3 {
		t.Fatalf("byte-truncated latest cursor = %#v", latest.Boundary)
	}
	older, err := store.OlderWindow(history.HistoryWindowRequest{Limit: 4, Cursor: latest.Boundary.Cursor})
	if err != nil {
		t.Fatal(err)
	}
	if got := historyLineIDs(older.Rows); !equalLineIDs(got, []history.LogicalLineID{1, 2}) {
		t.Fatalf("older IDs after byte truncation = %v, want [1 2]", got)
	}
	if older.HasMore || older.Boundary.Cursor.Valid {
		t.Fatalf("older page should reach the head: %#v", older.Boundary)
	}
}

func TestHistoryWindowStopsBeforeProjectingAnOversizedStoredLine(t *testing.T) {
	storage := &durabilityLineStorage{lines: make([]Line, history.MaxHistoryWindowLines)}
	storage.lines[len(storage.lines)-1] = Line{Runs: []Run{{Text: strings.Repeat("x", DefaultMaxOpenLineBytes)}}, HardEnd: true}
	store := NewStore("oversized-window", NewEngine(storage))

	_, err := store.LatestWindow(history.HistoryWindowRequest{Limit: history.MaxHistoryWindowLines})
	if !errors.Is(err, history.ErrHistoryWindowTooLarge) {
		t.Fatalf("latest error=%v, want oversized window", err)
	}
	if got := storage.linesSeen.Load(); got != 1 {
		t.Fatalf("visited lines=%d, want to stop at the first oversized line", got)
	}
}

func TestHistoryCopyByteBudgetIsAtomic(t *testing.T) {
	for _, test := range []struct {
		name      string
		extraByte bool
		wantError bool
	}{
		{name: "exact limit"},
		{name: "one byte over", extraByte: true, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := newHistoryBudgetStore(t, "copy-"+strings.ReplaceAll(test.name, " ", "-"))
			lines := make([]string, 1024)
			for index := range lines {
				lines[index] = strings.Repeat("x", 1023)
			}
			lines[len(lines)-1] += "x"
			if test.extraByte {
				lines[len(lines)-1] += "x"
			}
			if err := store.AppendLifecycleLines(lines); err != nil {
				t.Fatal(err)
			}
			snapshot, err := store.Freeze(history.FreezeHistoryRequest{Limit: 1})
			if err != nil {
				t.Fatal(err)
			}
			text, err := store.Copy(history.HistoryCopyRequest{Token: snapshot.Token})
			if test.wantError {
				if !errors.Is(err, history.ErrHistoryCopyTooLarge) || text != "" {
					t.Fatalf("overflow copy text bytes=%d err=%v", len(text), err)
				}
				return
			}
			if err != nil || len(text) != history.MaxHistoryCopyBytes {
				t.Fatalf("exact copy text bytes=%d err=%v", len(text), err)
			}
		})
	}
}

func TestHistoryCopyLineIterationBudgetRejectsEmptyLineDoS(t *testing.T) {
	store := newHistoryBudgetStore(t, "copy-lines")
	if err := store.AppendLifecycleLines(make([]string, history.MaxHistoryCopyLines+1)); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Freeze(history.FreezeHistoryRequest{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	text, err := store.Copy(history.HistoryCopyRequest{Token: snapshot.Token})
	if !errors.Is(err, history.ErrHistoryCopyTooLarge) || text != "" {
		t.Fatalf("line-budget copy text=%q err=%v", text, err)
	}
}

func TestRowTextRangeBoundedUsesDisplayCellsAndAuthoritativePadding(t *testing.T) {
	cells := []history.Cell{
		{Text: "a", Width: 1},
		{Text: "界", Width: 2},
		{Text: "e\u0301", Width: 1},
		{Text: "x", Width: 3},
		{Text: "", Width: 2},
	}
	for _, test := range []struct {
		name       string
		start, end int
		want       string
	}{
		{name: "ASCII", start: 0, end: 1, want: "a"},
		{name: "CJK", start: 1, end: 3, want: "界"},
		{name: "combining", start: 3, end: 4, want: "e\u0301"},
		{name: "cell padding", start: 4, end: 7, want: "x  "},
		{name: "padding only", start: 5, end: 7, want: "  "},
		{name: "empty authoritative padding", start: 7, end: 9, want: "  "},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := rowTextRangeBounded(cells, test.start, test.end, len(test.want))
			if !ok || got != test.want {
				t.Fatalf("range [%d,%d) = %q ok=%v, want %q", test.start, test.end, got, ok, test.want)
			}
		})
	}
	if got, ok := rowTextRangeBounded(cells, 0, 9, len("a界e\u0301x    ")-1); ok || got != "" {
		t.Fatalf("bounded overflow = %q ok=%v, want empty false", got, ok)
	}
}

func TestHistoryCopyRangeIsInclusiveExclusiveAndEndColumnZeroKeepsNewline(t *testing.T) {
	store := newHistoryBudgetStore(t, "copy-range")
	if err := store.AppendLifecycleLines([]string{"alpha", "beta", "gamma"}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Freeze(history.FreezeHistoryRequest{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	text, err := store.Copy(history.HistoryCopyRequest{
		Token: snapshot.Token,
		Range: &history.HistoryCopyRange{
			Start: history.HistoryCopyPosition{LineID: 1, Col: 2},
			End:   history.HistoryCopyPosition{LineID: 3, Col: 0},
		},
	})
	if err != nil || text != "pha\nbeta\n" {
		t.Fatalf("end_col=0 copy = %q err=%v", text, err)
	}
	for _, selection := range []history.HistoryCopyRange{
		{Start: history.HistoryCopyPosition{}, End: history.HistoryCopyPosition{LineID: 1}},
		{Start: history.HistoryCopyPosition{LineID: 1}, End: history.HistoryCopyPosition{}},
		{Start: history.HistoryCopyPosition{LineID: 1}, End: history.HistoryCopyPosition{LineID: 4}},
	} {
		if text, err := store.Copy(history.HistoryCopyRequest{Token: snapshot.Token, Range: &selection}); !errors.Is(err, history.ErrHistoryInvalidMutation) || text != "" {
			t.Fatalf("invalid range %#v text=%q err=%v", selection, text, err)
		}
	}
}

func TestHistoryCopyRangeRejectsLineIDsAgainstEmptySnapshot(t *testing.T) {
	store := newHistoryBudgetStore(t, "copy-empty-range")
	snapshot, err := store.Freeze(history.FreezeHistoryRequest{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	selection := history.HistoryCopyRange{
		Start: history.HistoryCopyPosition{LineID: 1},
		End:   history.HistoryCopyPosition{LineID: 1},
	}
	if text, err := store.Copy(history.HistoryCopyRequest{Token: snapshot.Token, Range: &selection}); !errors.Is(err, history.ErrHistoryInvalidMutation) || text != "" {
		t.Fatalf("empty snapshot explicit range text=%q err=%v", text, err)
	}
	if text, err := store.Copy(history.HistoryCopyRequest{Token: snapshot.Token}); err != nil || text != "" {
		t.Fatalf("empty snapshot full copy text=%q err=%v", text, err)
	}
}

func newHistoryBudgetStore(t *testing.T, terminalID string) *Store {
	t.Helper()
	store := NewStore(terminalID, NewEngine(openTestLineStorage(t, t.TempDir(), terminalID)))
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func historyLineIDs(rows []history.HistoryRow) []history.LogicalLineID {
	ids := make([]history.LogicalLineID, len(rows))
	for index := range rows {
		ids[index] = rows[index].LineID
	}
	return ids
}

func equalLineIDs(left, right []history.LogicalLineID) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

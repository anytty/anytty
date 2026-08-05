package state

import (
	"fmt"
	"testing"
)

var historyPerfStore HistoryStore
var historyPerfCopyMode CopyModeStore

func BenchmarkHistoryStoreApplyOlderPrependLoaded512(b *testing.B) {
	benchmarkHistoryStoreApplyOlderPrepend(b, 512, 128)
}

func BenchmarkHistoryStoreApplyOlderPrependLoaded8192(b *testing.B) {
	benchmarkHistoryStoreApplyOlderPrepend(b, 8192, 128)
}

func benchmarkHistoryStoreApplyOlderPrepend(b *testing.B, loadedLines int, olderLines int) {
	b.Helper()
	base := historyPerfStoreWithLines(100_000, loadedLines, 180)
	copyMode := CopyModeStore{
		Active:      true,
		ViewportTop: 0,
		ViewRows:    56,
		Cursor:      CopyPosition{Row: len(base.Rows) / 2, Col: 8},
		BoundToken:  base.Token,
		BoundCols:   base.Cols,
	}
	windows := make([]HistoryWindow, b.N)
	for i := 0; i < b.N; i++ {
		start := 100_000 - uint64(olderLines*(i+1))
		windows[i] = historyPerfOlderWindow(start, olderLines, base.Cols, base.Token, base.Generation, base.Boundary.LastLineID)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store := base
		store.Pending = &HistoryPendingRequest{
			ID:         RequestID(i + 1),
			Kind:       HistoryRequestOlder,
			TerminalID: store.TerminalID,
			Cols:       store.Cols,
			Token:      store.Token,
			Generation: store.Generation,
			Cursor:     store.Cursor,
			Boundary:   store.Boundary,
		}
		next, inserted, err := store.ApplyWindow(RequestID(i+1), windows[i])
		if err != nil {
			b.Fatalf("apply older: %v", err)
		}
		historyPerfCopyMode = copyMode.AcceptOlder(inserted, store, next, windows[i], next.Cols)
		historyPerfStore = next
	}
}

func historyPerfStoreWithLines(firstLineID uint64, count int, cols int) HistoryStore {
	source := historyPerfLogicalLines(firstLineID, count)
	rows, spans := ReflowHistoryLogicalLines(source, cols)
	return HistoryStore{
		ViewID:      "pane:pane-1",
		PaneID:      "pane-1",
		TerminalID:  "term-1",
		Token:       "tok-perf",
		Cols:        cols,
		SourceLines: source,
		Rows:        rows,
		Lines:       spans,
		Cursor:      HistoryCursor{Valid: true, BeforeLineID: firstLineID},
		Generation:  7,
		Boundary:    HistoryBoundary{FirstLineID: firstLineID, LastLineID: firstLineID + uint64(count) - 1},
		HasMore:     true,
	}
}

func historyPerfOlderWindow(firstLineID uint64, count int, cols int, token string, generation uint64, tailLineID uint64) HistoryWindow {
	source := historyPerfLogicalLines(firstLineID, count)
	rows, spans := ReflowHistoryLogicalLines(source, cols)
	return HistoryWindow{
		ViewID:      "pane:pane-1",
		PaneID:      "pane-1",
		TerminalID:  "term-1",
		Token:       token,
		Op:          HistoryWindowPrepend,
		Cols:        cols,
		SourceLines: source,
		Rows:        rows,
		Lines:       spans,
		Cursor:      HistoryCursor{Valid: true, BeforeLineID: firstLineID},
		Generation:  generation,
		Boundary:    HistoryBoundary{FirstLineID: firstLineID, LastLineID: tailLineID},
		HasMore:     true,
	}
}

func historyPerfLogicalLines(firstLineID uint64, count int) []HistoryLogicalLine {
	lines := make([]HistoryLogicalLine, count)
	for i := range lines {
		lineID := firstLineID + uint64(i)
		lines[i] = HistoryLogicalLine{
			Text:   fmt.Sprintf("history perf line %06d segment segment segment segment segment", lineID),
			LineID: lineID,
		}
	}
	return lines
}

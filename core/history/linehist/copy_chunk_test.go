package linehist

import (
	"context"
	"strings"
	"testing"

	"github.com/anytty/anytty/core/history"
)

func TestStoreCopyChunkCoversOneHundredThousandLines(t *testing.T) {
	file := openTestLineStorage(t, t.TempDir(), "copy-100k")
	lines := make([]Line, history.MaxHistoryCopyLines)
	for index := range lines {
		lines[index] = Line{Runs: []Run{{Text: "x"}}, HardEnd: true}
	}
	if err := file.AppendLines(lines); err != nil {
		t.Fatal(err)
	}
	store := NewStore("copy-100k", NewEngine(file))
	t.Cleanup(func() { _ = store.Close() })
	frozen, err := store.Freeze(history.FreezeHistoryRequest{Cols: 80, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	start := history.HistoryCopyPosition{LineID: 1}
	end := history.HistoryCopyPosition{LineID: history.MaxHistoryCopyLines, Col: 1}
	var copied strings.Builder
	chunks := 0
	for {
		result, err := store.CopyChunk(context.Background(), history.HistoryCopyChunkRequest{
			HistoryCopyRequest: history.HistoryCopyRequest{
				TerminalID: "copy-100k", Token: frozen.Token, Cols: 80,
				Range: &history.HistoryCopyRange{Start: start, End: end},
			},
			MaxLines: history.MaxHistoryCopyChunkLines,
			MaxBytes: history.MaxHistoryCopyChunkBytes,
		})
		if err != nil {
			t.Fatal(err)
		}
		if chunks > 0 {
			copied.WriteByte('\n')
		}
		copied.WriteString(result.Text)
		chunks++
		if result.Done {
			break
		}
		start = result.Next
	}
	if chunks < 2 {
		t.Fatalf("copy must use multiple chunks, got %d", chunks)
	}
	wantBytes := history.MaxHistoryCopyLines*2 - 1
	if copied.Len() != wantBytes || strings.Count(copied.String(), "\n") != history.MaxHistoryCopyLines-1 {
		t.Fatalf("copied bytes=%d newlines=%d, want bytes=%d newlines=%d", copied.Len(), strings.Count(copied.String(), "\n"), wantBytes, history.MaxHistoryCopyLines-1)
	}
}

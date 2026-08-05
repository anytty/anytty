package core

import (
	"context"
	"strings"
	"testing"
)

func TestR336ED2SmallPrimaryFrameRepaintKeepsScrollableHistory(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r329-ed2-primary",
		Command: []string{"shell"},
		Size:    Size{Cols: 18, Rows: 3},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}

	if err := server.IngestOutput(context.Background(), "term-r329-ed2-primary", "\x1b[?2026hframe-a\r\nframe-b\x1b[?2026l"); err != nil {
		t.Fatalf("seed primary frame: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-r329-ed2-primary", "\x1b[H\x1b[2Jnext-a\r\nnext-b"); err != nil {
		t.Fatalf("ingest ED2 redraw: %v", err)
	}

	rows, _ := r326CollectAllHistoryRows(t, server, "term-r329-ed2-primary", 18, 2)
	committed := strings.Join(committedHistoryRowTexts(rows), "|")
	if committed != "frame-a|frame-b" {
		t.Fatalf("ED2 clear-time primary frame must be preserved once in scrollable history, committed=%q rows=%#v", committed, rows)
	}
	joined := strings.Join(historyRowTexts(rows), "|")
	for _, want := range []string{"next-a", "next-b"} {
		if strings.Count(joined, want) != 1 {
			t.Fatalf("ED2 redraw should project %q exactly once, got %q rows=%#v", want, joined, rows)
		}
	}
}

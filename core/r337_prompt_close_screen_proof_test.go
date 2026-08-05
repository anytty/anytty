package core

import (
	"context"
	"strings"
	"testing"
)

func TestR337PromptAfterPrimaryFrameUsesCurrentScreenProof(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r337-prompt-proof",
		Command: []string{"shell"},
		Size:    Size{Cols: 40, Rows: 6},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}

	shutdown := "\x1b[?2026h\x1b[H" + strings.Join([]string{
		"codex header",
		"Shutting down...",
	}, "\r\n") + "\x1b[?2026l"
	if err := server.IngestOutput(context.Background(), "term-r337-prompt-proof", shutdown); err != nil {
		t.Fatalf("ingest shutdown frame: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-r337-prompt-proof", "\x1b[2;1Hshell prompt"); err != nil {
		t.Fatalf("ingest shell prompt: %v", err)
	}

	rows, _ := r326CollectAllHistoryRows(t, server, "term-r337-prompt-proof", 40, 3)
	texts := historyRowTexts(rows)
	joined := strings.Join(texts, "\n")
	if !strings.Contains(joined, "codex header") || !strings.Contains(joined, "shell prompt") {
		t.Fatalf("history should include visible final frame row and following prompt:\n%s\nrows=%#v", joined, rows)
	}
	if strings.Contains(joined, "Shutting down") {
		t.Fatalf("transient row overwritten by prompt must not be sealed from stale current frame:\n%s\nrows=%#v", joined, rows)
	}
	headerIndex := historyTextIndex(texts, "codex header")
	promptIndex := historyTextIndex(texts, "shell prompt")
	if headerIndex < 0 || promptIndex < 0 || promptIndex < headerIndex {
		t.Fatalf("prompt should project after final visible frame row, header=%d prompt=%d:\n%s\nrows=%#v", headerIndex, promptIndex, joined, rows)
	}
}

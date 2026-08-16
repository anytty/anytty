package core

import (
	"context"
	"strings"
	"testing"

	"github.com/anytty/anytty/core/history"
)

func TestR331OrdinaryPromptAfterScreenRedrawDoesNotInterleaveBeforeCurrentFrame(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r331-prompt-order",
		Command: []string{"shell"},
		Size:    Size{Cols: 40, Rows: 6},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}

	redraw := "\x1b[?2026h\x1b[H\x1b[2J" + strings.Join([]string{
		"S03 096/100 | seq=096",
		"S03 097/100 | seq=097",
		"S03 098/100 | seq=098",
		"S03 099/100 | seq=099",
		"S03 100/100 | seq=100 | 中文编号100中文",
		"=== REDRAW_END S03 lines=100 clear=ed2 sync=1 ===",
	}, "\r\n") + "\r\n\x1b[?2026l"
	if err := server.IngestOutput(context.Background(), "term-r331-prompt-order", redraw); err != nil {
		t.Fatalf("ingest redraw: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-r331-prompt-order", "PROMPT_AFTER"); err != nil {
		t.Fatalf("ingest prompt: %v", err)
	}

	rows, _ := r326CollectAllHistoryRows(t, server, "term-r331-prompt-order", 40, 3)
	texts := historyRowTexts(rows)
	joined := strings.Join(texts, "\n")
	promptIndex := historyTextIndex(texts, "PROMPT_AFTER")
	redrawEndIndex := historyTextIndex(texts, "REDRAW_END S03")
	if promptIndex < 0 || redrawEndIndex < 0 {
		t.Fatalf("history should include both redraw frame and following prompt:\n%s\nrows=%#v", joined, rows)
	}
	if promptIndex < redrawEndIndex {
		t.Fatalf("ordinary prompt must not be projected before the closed S03 frame, prompt=%d redraw_end=%d:\n%s\nrows=%#v", promptIndex, redrawEndIndex, joined, rows)
	}
	if rows[promptIndex].Kind != history.LineKindOrdinary {
		t.Fatalf("following prompt must remain ordinary history projection, rows=%#v", rows)
	}
}

func historyTextIndex(rows []string, needle string) int {
	for index, row := range rows {
		if strings.Contains(row, needle) {
			return index
		}
	}
	return -1
}

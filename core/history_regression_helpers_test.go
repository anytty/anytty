package core

import (
	"strings"

	"github.com/anytty/anytty/core/history"
)

func historyTextCount(rows []history.HistoryRow, needle string) int {
	count := 0
	texts := rawHistoryRowTexts(rows)
	for index, text := range texts {
		if isLifecycleHistoryRowTextAt(texts, index) {
			continue
		}
		if strings.Contains(text, needle) {
			count++
		}
	}
	return count
}

func committedHistoryRowTexts(rows []history.HistoryRow) []string {
	var out []string
	texts := rawHistoryRowTexts(rows)
	for index, row := range rows {
		text := texts[index]
		if row.Segment == history.HistorySegmentCommitted && !isLifecycleHistoryRowTextAt(texts, index) {
			out = append(out, text)
		}
	}
	return out
}

func rawHistoryRowTexts(rows []history.HistoryRow) []string {
	texts := make([]string, 0, len(rows))
	for _, row := range rows {
		texts = append(texts, historyCellsText(row.Cells))
	}
	return texts
}

func historyLineIDForText(rows []history.HistoryRow, want string) (history.LogicalLineID, bool) {
	for _, row := range rows {
		if historyCellsText(row.Cells) == want {
			return row.LineID, true
		}
	}
	return 0, false
}

func isLifecycleHistoryText(text string) bool {
	return strings.HasPrefix(text, "terminal started: ") ||
		strings.HasPrefix(text, "started at: ") ||
		strings.HasPrefix(text, "terminal exited: ") ||
		strings.HasPrefix(text, "exited at: ") ||
		strings.HasPrefix(text, "command: ")
}

func isLifecycleHistoryRowTextAt(texts []string, index int) bool {
	if index < 0 || index >= len(texts) {
		return false
	}
	text := texts[index]
	if isLifecycleHistoryText(text) {
		return true
	}
	return text == "" && index+1 < len(texts) && strings.HasPrefix(texts[index+1], "terminal exited: ")
}

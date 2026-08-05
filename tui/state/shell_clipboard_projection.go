package state

import "strings"

type ClipboardHistoryItem struct {
	ID                  string
	Title               string
	Preview             string
	Text                string
	Selected            bool
	TitleMatchIndexes   []int
	PreviewMatchIndexes []int
}

func ClipboardHistoryItems(root Root) []ClipboardHistoryItem {
	shell := root.Shell.ReadonlyDefaults()
	query := strings.TrimSpace(shell.Overlay.Query)
	items := make([]ClipboardHistoryItem, 0, len(root.Clipboard.Entries))
	for _, entry := range root.Clipboard.Entries {
		item := ClipboardHistoryItem{
			ID:      entry.ID,
			Title:   entry.Title,
			Preview: entry.Preview,
			Text:    entry.Text,
		}
		item, ok := matchClipboardHistoryItem(item, query)
		if !ok {
			continue
		}
		items = append(items, item)
	}
	if len(items) > 0 {
		selected := shell.Overlay.SelectedIndex
		if selected < 0 {
			selected = 0
		}
		if selected >= len(items) {
			selected = len(items) - 1
		}
		items[selected].Selected = true
	}
	return items
}

func matchClipboardHistoryItem(item ClipboardHistoryItem, query string) (ClipboardHistoryItem, bool) {
	if query == "" {
		return item, true
	}
	if indexes := TerminalPickerQueryMatchIndexes(item.Title, query); indexes != nil {
		item.TitleMatchIndexes = indexes
		return item, true
	}
	if indexes := TerminalPickerQueryMatchIndexes(item.Preview, query); indexes != nil {
		item.PreviewMatchIndexes = indexes
		return item, true
	}
	if indexes := TerminalPickerQueryMatchIndexes(item.Text, query); indexes != nil {
		if previewIndexes := TerminalPickerQueryMatchIndexes(item.Preview, query); previewIndexes != nil {
			item.PreviewMatchIndexes = previewIndexes
		}
		return item, true
	}
	return item, false
}

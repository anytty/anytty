package state

import "strings"

func (store ShellStore) EnterPromptSuggestion() ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayPrompt || !store.Overlay.Open {
		return store
	}
	items := store.Overlay.Prompt.ActiveSuggestionItems()
	if len(items) == 0 {
		return store
	}
	index := clampPromptInt(store.Overlay.Prompt.SuggestionSelected, 0, len(items)-1)
	if active := store.Overlay.Prompt.ActivePromptField(); active != nil {
		active.Value = items[index]
		active.Cursor = len([]rune(active.Value))
	}
	store.Overlay.Prompt.SuggestionFocused = true
	store.Overlay.Prompt.SuggestionSelected = 0
	store.Overlay.Prompt.SuggestionOffset = 0
	return store
}

func (store ShellStore) LeavePromptSuggestionPath() ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayPrompt || !store.Overlay.Open {
		return store
	}
	if active := store.Overlay.Prompt.ActivePromptField(); active != nil {
		active.Value = parentPromptPath(active.Value)
		active.Cursor = len([]rune(active.Value))
	}
	store.Overlay.Prompt.SuggestionFocused = true
	store.Overlay.Prompt.SuggestionSelected = 0
	store.Overlay.Prompt.SuggestionOffset = 0
	return store
}

func parentPromptPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	separator := "/"
	if strings.Contains(value, "\\") && !strings.Contains(value, "/") {
		separator = "\\"
	}
	trimmed := strings.TrimRight(value, "/\\")
	if trimmed == "" {
		return value
	}
	index := strings.LastIndex(trimmed, separator)
	if index < 0 {
		return ""
	}
	return trimmed[:index+1]
}

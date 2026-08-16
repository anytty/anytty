package state

import "strings"

const promptSuggestionVisibleRows = 6

func (store ShellStore) SetPromptValue(value string) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayPrompt || !store.Overlay.Open {
		return store
	}
	if active := store.Overlay.Prompt.ActivePromptField(); active != nil {
		active.Value = value
		active.Cursor = len([]rune(value))
		return store
	}
	store.Overlay.Prompt.Value = value
	return store
}

func (store ShellStore) InsertPromptText(value string) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayPrompt || !store.Overlay.Open || value == "" {
		return store
	}
	if active := store.Overlay.Prompt.ActivePromptField(); active != nil {
		runes := []rune(active.Value)
		cursor := clampPromptInt(active.Cursor, 0, len(runes))
		insert := []rune(value)
		next := make([]rune, 0, len(runes)+len(insert))
		next = append(next, runes[:cursor]...)
		next = append(next, insert...)
		next = append(next, runes[cursor:]...)
		active.Value = string(next)
		active.Cursor = cursor + len(insert)
		return store
	}
	store.Overlay.Prompt.Value += value
	return store
}

func (store ShellStore) DeletePromptBackward() ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayPrompt || !store.Overlay.Open {
		return store
	}
	if active := store.Overlay.Prompt.ActivePromptField(); active != nil {
		runes := []rune(active.Value)
		cursor := clampPromptInt(active.Cursor, 0, len(runes))
		if cursor == 0 {
			return store
		}
		next := make([]rune, 0, len(runes)-1)
		next = append(next, runes[:cursor-1]...)
		next = append(next, runes[cursor:]...)
		active.Value = string(next)
		active.Cursor = cursor - 1
		return store
	}
	store.Overlay.Prompt.Value = trimLastPromptRune(store.Overlay.Prompt.Value)
	return store
}

func (store ShellStore) DeletePromptForward() ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayPrompt || !store.Overlay.Open {
		return store
	}
	if active := store.Overlay.Prompt.ActivePromptField(); active != nil {
		runes := []rune(active.Value)
		cursor := clampPromptInt(active.Cursor, 0, len(runes))
		if cursor >= len(runes) {
			return store
		}
		next := make([]rune, 0, len(runes)-1)
		next = append(next, runes[:cursor]...)
		next = append(next, runes[cursor+1:]...)
		active.Value = string(next)
		active.Cursor = cursor
		return store
	}
	store.Overlay.Prompt.Value = trimLastPromptRune(store.Overlay.Prompt.Value)
	return store
}

func (store ShellStore) MovePromptCursor(delta int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayPrompt || !store.Overlay.Open || delta == 0 {
		return store
	}
	if active := store.Overlay.Prompt.ActivePromptField(); active != nil {
		active.Cursor = clampPromptInt(active.Cursor+delta, 0, len([]rune(active.Value)))
	}
	return store
}

func (store ShellStore) SetPromptCursor(cursor int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayPrompt || !store.Overlay.Open {
		return store
	}
	if active := store.Overlay.Prompt.ActivePromptField(); active != nil {
		active.Cursor = clampPromptInt(cursor, 0, len([]rune(active.Value)))
	}
	return store
}

func (store ShellStore) MovePromptField(delta int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayPrompt || !store.Overlay.Open || len(store.Overlay.Prompt.Fields) == 0 || delta == 0 {
		return store
	}
	next := store.Overlay.Prompt.ActiveField + delta
	if next < 0 {
		next = 0
	}
	if next >= len(store.Overlay.Prompt.Fields) {
		next = len(store.Overlay.Prompt.Fields) - 1
	}
	store.Overlay.Prompt.ActiveField = next
	store.Overlay.Prompt.SuggestionFocused = false
	store.Overlay.Prompt.SuggestionSelected = 0
	store.Overlay.Prompt.SuggestionOffset = 0
	return store
}

func (store ShellStore) ClearPromptSuggestions() ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayPrompt || !store.Overlay.Open {
		return store
	}
	for index := range store.Overlay.Prompt.Fields {
		store.Overlay.Prompt.Fields[index].SuggestionTitle = ""
		store.Overlay.Prompt.Fields[index].SuggestionItems = nil
		store.Overlay.Prompt.Fields[index].SuggestionEmpty = ""
	}
	store.Overlay.Prompt.SuggestionFocused = false
	store.Overlay.Prompt.SuggestionSelected = 0
	store.Overlay.Prompt.SuggestionOffset = 0
	return store
}

func (store ShellStore) SetActivePromptSuggestions(title string, items []string, empty string) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayPrompt || !store.Overlay.Open {
		return store
	}
	for index := range store.Overlay.Prompt.Fields {
		store.Overlay.Prompt.Fields[index].SuggestionTitle = ""
		store.Overlay.Prompt.Fields[index].SuggestionItems = nil
		store.Overlay.Prompt.Fields[index].SuggestionEmpty = ""
	}
	active := store.Overlay.Prompt.ActivePromptField()
	if active == nil {
		store.Overlay.Prompt.SuggestionFocused = false
		store.Overlay.Prompt.SuggestionSelected = 0
		store.Overlay.Prompt.SuggestionOffset = 0
		return store
	}
	active.SuggestionTitle = title
	active.SuggestionItems = append([]string(nil), items...)
	active.SuggestionEmpty = empty
	if len(items) == 0 {
		store.Overlay.Prompt.SuggestionFocused = false
		store.Overlay.Prompt.SuggestionSelected = 0
		store.Overlay.Prompt.SuggestionOffset = 0
		return store
	}
	store.Overlay.Prompt.SuggestionSelected = clampPromptInt(store.Overlay.Prompt.SuggestionSelected, 0, len(items)-1)
	store.Overlay.Prompt.SuggestionOffset = clampPromptInt(store.Overlay.Prompt.SuggestionOffset, 0, len(items)-1)
	return store
}

func (store ShellStore) SetPromptSuggestionFocused(focused bool) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayPrompt || !store.Overlay.Open {
		return store
	}
	if focused && len(store.Overlay.Prompt.ActiveSuggestionItems()) == 0 {
		focused = false
	}
	store.Overlay.Prompt.SuggestionFocused = focused
	if !focused {
		store.Overlay.Prompt.SuggestionSelected = 0
		store.Overlay.Prompt.SuggestionOffset = 0
	}
	return store
}

func (store ShellStore) MovePromptSuggestionSelection(delta int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayPrompt || !store.Overlay.Open || delta == 0 {
		return store
	}
	items := store.Overlay.Prompt.ActiveSuggestionItems()
	if len(items) == 0 {
		return store
	}
	next := store.Overlay.Prompt.SuggestionSelected + delta
	for next < 0 {
		next += len(items)
	}
	next %= len(items)
	store.Overlay.Prompt.SuggestionSelected = next
	store.Overlay.Prompt.SuggestionOffset = promptSuggestionOffsetForSelection(store.Overlay.Prompt.SuggestionOffset, next, len(items))
	return store
}

func (store ShellStore) AcceptPromptSuggestion() ShellStore {
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
	store.Overlay.Prompt.SuggestionFocused = false
	store.Overlay.Prompt.SuggestionSelected = 0
	store.Overlay.Prompt.SuggestionOffset = 0
	return store
}

func (store ShellStore) SubmitPrompt() ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayPrompt || !store.Overlay.Open {
		return store
	}
	prompt := store.Overlay.Prompt
	value := strings.TrimSpace(prompt.Value)
	if field := prompt.ActivePromptField(); field != nil {
		value = strings.TrimSpace(field.Value)
	}
	if prompt.Destructive && value != prompt.ConfirmText {
		prompt.LastResult = "confirm required: " + prompt.ConfirmText
		store.Overlay.Prompt = prompt
		return store
	}
	prompt.Submitted = true
	prompt.LastResult = value
	store.Overlay.Prompt = prompt
	return store
}

func (store ShellStore) CancelPrompt() ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayPrompt || !store.Overlay.Open {
		return store
	}
	store.Overlay.Prompt.Canceled = true
	return store
}

func (prompt *PromptState) ActivePromptField() *PromptFieldState {
	if prompt == nil || len(prompt.Fields) == 0 {
		return nil
	}
	if prompt.ActiveField < 0 {
		prompt.ActiveField = 0
	}
	if prompt.ActiveField >= len(prompt.Fields) {
		prompt.ActiveField = len(prompt.Fields) - 1
	}
	return &prompt.Fields[prompt.ActiveField]
}

func (prompt PromptState) FieldValue(key string) string {
	for _, field := range prompt.Fields {
		if field.Key == key {
			return strings.TrimSpace(field.Value)
		}
	}
	return ""
}

func (prompt PromptState) FieldRawValue(key string) string {
	for _, field := range prompt.Fields {
		if field.Key == key {
			return field.Value
		}
	}
	return ""
}

func (prompt PromptState) ActiveSuggestionItems() []string {
	if field := prompt.ActivePromptField(); field != nil {
		return field.SuggestionItems
	}
	return nil
}

func trimLastPromptRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
}

func clampPromptInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// 保持 workdir 候选选中项始终落在可见窗口内。
func promptSuggestionOffsetForSelection(offset int, selected int, count int) int {
	if count <= promptSuggestionVisibleRows {
		return 0
	}
	maxOffset := count - promptSuggestionVisibleRows
	offset = clampPromptInt(offset, 0, maxOffset)
	if selected < offset {
		return selected
	}
	if selected >= offset+promptSuggestionVisibleRows {
		return selected - promptSuggestionVisibleRows + 1
	}
	return offset
}

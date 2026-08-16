package state

const (
	DefaultClipboardHistoryNameWidth = 24
	MinClipboardHistoryNameWidth     = 14
	MaxClipboardHistoryNameWidth     = 72
)

func (store ShellStore) SetClipboardHistoryQuery(query string) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayClipboardHistory || !store.Overlay.Open {
		return store
	}
	store.Overlay.Query = query
	store.Overlay.SelectedIndex = 0
	return store
}

func (store ShellStore) MoveClipboardHistorySelection(delta int, itemCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayClipboardHistory || !store.Overlay.Open || itemCount <= 0 || delta == 0 {
		return store
	}
	next := store.Overlay.SelectedIndex + delta
	next %= itemCount
	if next < 0 {
		next += itemCount
	}
	store.Overlay.SelectedIndex = next
	return store
}

func (store ShellStore) SetClipboardHistorySelectedIndex(index int, itemCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayClipboardHistory || !store.Overlay.Open || itemCount <= 0 {
		return store
	}
	if index < 0 {
		index = 0
	}
	if index >= itemCount {
		index = itemCount - 1
	}
	store.Overlay.SelectedIndex = index
	return store
}

func (store ShellStore) SetClipboardHistoryNameWidth(width int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayClipboardHistory || !store.Overlay.Open {
		return store
	}
	store.Overlay.ClipboardNameWidth = normalizeClipboardHistoryNameWidth(width)
	return store
}

func (store ShellStore) MoveClipboardHistoryNameWidth(delta int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayClipboardHistory || !store.Overlay.Open || delta == 0 {
		return store
	}
	store.Overlay.ClipboardNameWidth = normalizeClipboardHistoryNameWidth(ClipboardHistoryNameWidth(store.Overlay) + delta)
	return store
}

func ClipboardHistoryNameWidth(overlay OverlayState) int {
	return normalizeClipboardHistoryNameWidth(overlay.ClipboardNameWidth)
}

func normalizeClipboardHistoryNameWidth(width int) int {
	if width <= 0 {
		width = DefaultClipboardHistoryNameWidth
	}
	if width < MinClipboardHistoryNameWidth {
		return MinClipboardHistoryNameWidth
	}
	if width > MaxClipboardHistoryNameWidth {
		return MaxClipboardHistoryNameWidth
	}
	return width
}

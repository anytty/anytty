package state

type ExitedPaneCTAState struct {
	SelectedIndex int
}

func (store ShellStore) MoveExitedPaneCTASelection(delta int, count int) ShellStore {
	store = store.EnsureDefaults()
	if count <= 0 {
		store.ExitedPaneCTA.SelectedIndex = 0
		return store
	}
	selected := store.ExitedPaneCTA.SelectedIndex + delta
	for selected < 0 {
		selected += count
	}
	store.ExitedPaneCTA.SelectedIndex = selected % count
	return store
}

func (store ShellStore) SetExitedPaneCTASelection(index int, count int) ShellStore {
	store = store.EnsureDefaults()
	if count <= 0 || index < 0 {
		store.ExitedPaneCTA.SelectedIndex = 0
		return store
	}
	if index >= count {
		index = count - 1
	}
	store.ExitedPaneCTA.SelectedIndex = index
	return store
}

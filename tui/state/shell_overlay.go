package state

func (store ShellStore) ExitInteractionMode() ShellStore {
	store = store.EnsureDefaults()
	if store.InteractionMode != InteractionModeNormal {
		store.InteractionModeSeq++
	}
	store.InteractionMode = InteractionModeNormal
	return store
}

func (store ShellStore) AddToast(spec ToastSpec) ShellStore {
	store = store.EnsureDefaults()
	if spec.Severity == "" {
		spec.Severity = ToastInfo
	}
	dismissAfterTicks := spec.DismissAfterTicks
	if dismissAfterTicks == 0 {
		dismissAfterTicks = defaultToastDismissAfterTicks(spec)
	}
	if index := store.findMatchingToast(spec); index >= 0 {
		// 同内容 toast 只刷新生命周期并移到当前 toast，避免拖动等连续操作刷屏。
		toasts := cloneToasts(store.Toasts)
		toast := ToastState{
			ID:                toasts[index].ID,
			Severity:          spec.Severity,
			Title:             spec.Title,
			Body:              spec.Body,
			Pending:           spec.Pending,
			DismissAfterTicks: dismissAfterTicks,
		}
		toasts = append(toasts[:index], toasts[index+1:]...)
		toasts = append(toasts, toast)
		store.Toasts = toasts
		return store
	}
	if spec.ID == "" {
		store.nextToastSeq++
		spec.ID = formatToastID(store.nextToastSeq)
	}
	toast := ToastState{
		ID:                spec.ID,
		Severity:          spec.Severity,
		Title:             spec.Title,
		Body:              spec.Body,
		Pending:           spec.Pending,
		DismissAfterTicks: dismissAfterTicks,
	}
	store.Toasts = append(cloneToasts(store.Toasts), toast)
	return store
}

func (store ShellStore) findMatchingToast(spec ToastSpec) int {
	if spec.ID != "" {
		for index, toast := range store.Toasts {
			if toast.ID == spec.ID {
				return index
			}
		}
	}
	for index, toast := range store.Toasts {
		if toast.Severity == spec.Severity &&
			toast.Title == spec.Title &&
			toast.Body == spec.Body &&
			toast.Pending == spec.Pending {
			return index
		}
	}
	return -1
}

func (store ShellStore) TickToasts(ticks uint64) ShellStore {
	if ticks == 0 || len(store.Toasts) == 0 {
		return store
	}
	kept := make([]ToastState, 0, len(store.Toasts))
	for _, toast := range store.Toasts {
		toast.AgeTicks += ticks
		if toast.DismissAfterTicks > 0 && toast.AgeTicks >= toast.DismissAfterTicks {
			continue
		}
		kept = append(kept, toast)
	}
	store.Toasts = cloneToasts(kept)
	return store
}

func (store ShellStore) CloseCurrentToast() ShellStore {
	if len(store.Toasts) == 0 {
		return store
	}
	store.Toasts = cloneToasts(store.Toasts[:len(store.Toasts)-1])
	return store
}

func (store ShellStore) ClearToasts() ShellStore {
	store.Toasts = nil
	return store
}

// defaultToastDismissAfterTicks 给新增 toast 明确生命周期，避免真实 runtime 中遗留静态消息。
func defaultToastDismissAfterTicks(spec ToastSpec) uint64 {
	if spec.Pending {
		return pendingToastDismissTicks
	}
	switch spec.Severity {
	case ToastWarning, ToastError:
		return attentionToastDismissTicks
	default:
		return defaultToastDismissTicks
	}
}

func (store ShellStore) OpenTerminalPicker() ShellStore {
	store = store.EnsureDefaults()
	targetID := store.ActivePaneID
	if activeFloatingID := store.ActiveFloatingID(); activeFloatingID != "" {
		targetID = activeFloatingID
	}
	store.Overlay = OverlayState{
		Kind:                 OverlayTerminalPicker,
		Open:                 true,
		TargetID:             targetID,
		SelectedIndex:        0,
		TerminalPickerStatus: TerminalPickerStatusAll,
		TerminalPickerView:   TerminalPickerViewList,
	}
	return store
}

func (store ShellStore) OpenTerminalPool() ShellStore {
	store = store.EnsureDefaults()
	targetID := store.ActivePaneID
	if activeFloatingID := store.ActiveFloatingID(); activeFloatingID != "" {
		targetID = activeFloatingID
	}
	store.Overlay = OverlayState{
		Kind:          OverlayTerminalPool,
		Open:          true,
		TargetID:      targetID,
		SelectedIndex: 0,
	}
	return store
}

func (store ShellStore) OpenWorkbenchTree() ShellStore {
	store = store.EnsureDefaults()
	store.Overlay = OverlayState{
		Kind:          OverlayWorkbenchTree,
		Open:          true,
		TargetID:      store.ActivePaneID,
		SelectedIndex: 0,
	}
	return store
}

func (store ShellStore) OpenClipboardHistory() ShellStore {
	store = store.EnsureDefaults()
	targetID := store.ActivePaneID
	if activeFloatingID := store.ActiveFloatingID(); activeFloatingID != "" {
		targetID = activeFloatingID
	}
	store.Overlay = OverlayState{
		Kind:               OverlayClipboardHistory,
		Open:               true,
		TargetID:           targetID,
		SelectedIndex:      0,
		ClipboardNameWidth: DefaultClipboardHistoryNameWidth,
	}
	return store
}

func (store ShellStore) OpenFloatingOverview() ShellStore {
	store = store.EnsureDefaults()
	store.Overlay = OverlayState{
		Kind:          OverlayFloatingOverview,
		Open:          true,
		TargetID:      store.ActiveFloatingID(),
		SelectedIndex: 0,
	}
	return store
}

func (store ShellStore) OpenPrompt(prompt PromptState) ShellStore {
	store = store.EnsureDefaults()
	if prompt.Title == "" {
		prompt.Title = "Command Prompt"
	}
	if prompt.Placeholder == "" {
		prompt.Placeholder = "type command"
	}
	if prompt.Destructive && prompt.ConfirmText == "" {
		prompt.ConfirmText = "confirm"
	}
	if len(prompt.Fields) > 0 {
		if prompt.ActiveField < 0 {
			prompt.ActiveField = 0
		}
		if prompt.ActiveField >= len(prompt.Fields) {
			prompt.ActiveField = len(prompt.Fields) - 1
		}
		for index := range prompt.Fields {
			valueWidth := len([]rune(prompt.Fields[index].Value))
			if prompt.Fields[index].Cursor < 0 || prompt.Fields[index].Cursor > valueWidth || (prompt.Fields[index].Cursor == 0 && valueWidth > 0) {
				prompt.Fields[index].Cursor = valueWidth
			}
		}
	}
	store.Overlay = OverlayState{
		Kind:   OverlayPrompt,
		Open:   true,
		Prompt: prompt,
	}
	return store
}

func (store ShellStore) OpenHelp(section string) ShellStore {
	store = store.EnsureDefaults()
	store.Overlay = OverlayState{
		Kind:        OverlayHelp,
		Open:        true,
		HelpSection: section,
	}
	return store
}

func (store ShellStore) MoveHelpSelection(delta int, itemCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayHelp || !store.Overlay.Open || itemCount <= 0 {
		return store
	}
	next := store.Overlay.SelectedIndex + delta
	if next < 0 {
		next = 0
	}
	if next >= itemCount {
		next = itemCount - 1
	}
	store.Overlay.SelectedIndex = next
	return store
}

func (store ShellStore) SetHelpSelection(index int, itemCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayHelp || !store.Overlay.Open || itemCount <= 0 {
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

func (store ShellStore) SetTerminalPickerQuery(query string) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open {
		return store
	}
	store.Overlay.Query = query
	store.Overlay.SelectedIndex = 0
	return store
}

func (store ShellStore) SetTerminalPickerTagQuery(query string) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open || store.Overlay.TerminalPickerView != TerminalPickerViewTags {
		return store
	}
	store.Overlay.TerminalPickerTagQuery = query
	store.Overlay.TerminalPickerTagIndex = 0
	return store
}

func (store ShellStore) SetTerminalPickerEndpointQuery(query string) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open || store.Overlay.TerminalPickerView != TerminalPickerViewEndpoints {
		return store
	}
	store.Overlay.TerminalPickerEndpointQuery = query
	store.Overlay.TerminalPickerEndpointIndex = 0
	return store
}

func (store ShellStore) SetTerminalPickerEndpoint(endpointID EndpointID) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open {
		return store
	}
	store.Overlay.TerminalPickerEndpointID = NormalizeEndpointID(endpointID)
	store.Overlay.TerminalPickerTagFilters = nil
	store.Overlay.TerminalPickerTagIndex = 0
	store.Overlay.SelectedIndex = 0
	return store
}

func (store ShellStore) SetTerminalPickerStatus(status TerminalPickerStatusFilter) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open {
		return store
	}
	store.Overlay.TerminalPickerStatus = normalizeTerminalPickerStatus(status)
	store.Overlay.SelectedIndex = 0
	return store
}

func (store ShellStore) OpenTerminalPickerTags() ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open {
		return store
	}
	store.Overlay.TerminalPickerView = TerminalPickerViewTags
	store.Overlay.TerminalPickerTagQuery = ""
	store.Overlay.TerminalPickerTagIndex = 0
	return store
}

func (store ShellStore) OpenTerminalPickerEndpoints() ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open {
		return store
	}
	store.Overlay.TerminalPickerView = TerminalPickerViewEndpoints
	store.Overlay.TerminalPickerEndpointQuery = ""
	store.Overlay.TerminalPickerEndpointIndex = 0
	return store
}

func (store ShellStore) CloseTerminalPickerEndpoints() ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open {
		return store
	}
	store.Overlay.TerminalPickerView = TerminalPickerViewList
	store.Overlay.TerminalPickerEndpointQuery = ""
	return store
}

func (store ShellStore) SetTerminalPickerEndpointIndex(index int, endpointCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open || endpointCount <= 0 {
		return store
	}
	if index < 0 {
		index = 0
	}
	if index >= endpointCount {
		index = endpointCount - 1
	}
	store.Overlay.TerminalPickerEndpointIndex = index
	return store
}

func (store ShellStore) MoveTerminalPickerEndpointIndex(delta int, endpointCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open || endpointCount <= 0 || delta == 0 {
		return store
	}
	next := (store.Overlay.TerminalPickerEndpointIndex + delta) % endpointCount
	if next < 0 {
		next += endpointCount
	}
	store.Overlay.TerminalPickerEndpointIndex = next
	return store
}

func (store ShellStore) CloseTerminalPickerTags() ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open {
		return store
	}
	store.Overlay.TerminalPickerView = TerminalPickerViewList
	store.Overlay.TerminalPickerTagQuery = ""
	return store
}

func (store ShellStore) SetTerminalPickerTagIndex(index int, tagCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open || tagCount <= 0 {
		return store
	}
	if index < 0 {
		index = 0
	}
	if index >= tagCount {
		index = tagCount - 1
	}
	store.Overlay.TerminalPickerTagIndex = index
	return store
}

func (store ShellStore) MoveTerminalPickerTagIndex(delta int, tagCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open || tagCount <= 0 || delta == 0 {
		return store
	}
	next := (store.Overlay.TerminalPickerTagIndex + delta) % tagCount
	if next < 0 {
		next += tagCount
	}
	store.Overlay.TerminalPickerTagIndex = next
	return store
}

func (store ShellStore) ToggleTerminalPickerTag(label string) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open || label == "" {
		return store
	}
	filters := append([]string(nil), store.Overlay.TerminalPickerTagFilters...)
	for index, current := range filters {
		if current == label {
			store.Overlay.TerminalPickerTagFilters = append(filters[:index], filters[index+1:]...)
			store.Overlay.SelectedIndex = 0
			return store
		}
	}
	store.Overlay.TerminalPickerTagFilters = append(filters, label)
	store.Overlay.SelectedIndex = 0
	return store
}

func normalizeTerminalPickerStatus(status TerminalPickerStatusFilter) TerminalPickerStatusFilter {
	switch status {
	case TerminalPickerStatusRunning, TerminalPickerStatusExited:
		return status
	default:
		return TerminalPickerStatusAll
	}
}

func (store ShellStore) SetTerminalPoolQuery(query string) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPool || !store.Overlay.Open {
		return store
	}
	store.Overlay.Query = query
	store.Overlay.SelectedIndex = 0
	return store
}

func (store ShellStore) SetWorkbenchTreeQuery(query string) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayWorkbenchTree || !store.Overlay.Open {
		return store
	}
	store.Overlay.Query = query
	store.Overlay.SelectedIndex = 0
	return store
}

func (store ShellStore) MoveTerminalPickerSelection(delta int, itemCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open || itemCount <= 0 || delta == 0 {
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

func (store ShellStore) MoveTerminalPoolSelection(delta int, itemCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPool || !store.Overlay.Open || itemCount <= 0 || delta == 0 {
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

func (store ShellStore) MoveWorkbenchTreeSelection(delta int, itemCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayWorkbenchTree || !store.Overlay.Open || itemCount <= 0 || delta == 0 {
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

func (store ShellStore) MoveFloatingOverviewSelection(delta int, itemCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayFloatingOverview || !store.Overlay.Open || itemCount <= 0 || delta == 0 {
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

func (store ShellStore) SetTerminalPickerSelectedIndex(index int, itemCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPicker || !store.Overlay.Open || itemCount <= 0 {
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

func (store ShellStore) SetTerminalPoolSelectedIndex(index int, itemCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayTerminalPool || !store.Overlay.Open || itemCount <= 0 {
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

func (store ShellStore) SetWorkbenchTreeSelectedIndex(index int, itemCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayWorkbenchTree || !store.Overlay.Open || itemCount <= 0 {
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

func (store ShellStore) SetFloatingOverviewSelectedIndex(index int, itemCount int) ShellStore {
	store = store.EnsureDefaults()
	if store.Overlay.Kind != OverlayFloatingOverview || !store.Overlay.Open || itemCount <= 0 {
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

func (store ShellStore) CloseOverlay() ShellStore {
	store.Overlay = OverlayState{}
	return store.EnsureDefaults()
}

func formatToastID(seq uint64) string {
	if seq == 0 {
		return "toast-0"
	}
	digits := make([]byte, 0, 20)
	for seq > 0 {
		digits = append(digits, byte('0'+seq%10))
		seq /= 10
	}
	for left, right := 0, len(digits)-1; left < right; left, right = left+1, right-1 {
		digits[left], digits[right] = digits[right], digits[left]
	}
	return "toast-" + string(digits)
}

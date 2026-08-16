package state

import "strings"

func (store ShellStore) ApplyFloatingCommand(command FloatingCommand) (ShellStore, FloatingCommandResult) {
	store = store.EnsureDefaults()
	if command.Source == "" {
		command.Source = PaneCommandSourceKeyboard
	}
	switch command.Action {
	case FloatingCommandCreate:
		return store.createFloating(command)
	case FloatingCommandFocusRaise:
		return store.focusRaiseFloating(command.TargetID, command.Action)
	case FloatingCommandDeactivate:
		return store.deactivateFloating(command.Action)
	case FloatingCommandClose:
		return store.closeFloating(command.TargetID)
	case FloatingCommandCenter:
		return store.centerFloating(command)
	case FloatingCommandToggleCollapse:
		return store.toggleCollapseFloating(command.TargetID)
	case FloatingCommandSummon:
		return store.summonFloating(command)
	case FloatingCommandMove:
		return store.moveFloating(command)
	case FloatingCommandPosition:
		return store.positionFloating(command)
	case FloatingCommandResize:
		return store.resizeFloating(command)
	case FloatingCommandToggleAll, FloatingCommandShowAll, FloatingCommandCollapseAll, FloatingCommandFit, FloatingCommandToggleAutoFit, FloatingCommandRefreshAutoFit:
		return store.applyFloatingGroupCommand(command)
	default:
		return store, floatingCommandInvalid(command.Action, "unknown action")
	}
}

func (store ShellStore) SummonFloatingByIndex(index int) (ShellStore, FloatingCommandResult) {
	return store.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandSummon, Index: index})
}

func (store ShellStore) BindFloatingTerminal(id string, terminalID string) ShellStore {
	store = store.EnsureDefaults()
	if terminalID == "" {
		return store
	}
	index := store.floatingIndexOrActive(id)
	if index < 0 {
		return store
	}
	floatings := cloneFloatings(store.activeFloatings())
	if floatings[index].Pane.ID == "" {
		floatings[index].Pane = PaneState{ID: floatings[index].ID + "-pane", Title: floatings[index].Title, Kind: PaneTerminalLive}
	}
	floatings[index].Pane.TerminalID = terminalID
	floatings[index].Pane.Kind = PaneTerminalLive
	if floatings[index].Pane.Title == "" {
		floatings[index].Pane.Title = terminalID
	}
	floatings[index].Active = true
	activeID := floatings[index].ID
	floatings[index].Z = store.nextFloatingZ() + 1
	return store.withActiveTabFloatings(floatings, activeID).EnsureDefaults()
}

func (store ShellStore) DetachFloatingTerminal(id string) ShellStore {
	store = store.EnsureDefaults()
	index := store.floatingIndexOrActive(id)
	if index < 0 {
		return store
	}
	floatings := cloneFloatings(store.activeFloatings())
	floatings[index].Pane.TerminalID = ""
	floatings[index].Pane.Kind = PaneEmpty
	return store.withActiveTabFloatings(floatings, store.activeFloatingID()).EnsureDefaults()
}

func (store ShellStore) createFloating(command FloatingCommand) (ShellStore, FloatingCommandResult) {
	pane := command.Pane
	if pane.ID == "" {
		store.nextFloatingSeq++
		pane.ID = formatFloatingID(store.nextFloatingSeq) + "-pane"
	}
	if pane.Title == "" {
		pane.Title = "floating"
	}
	if pane.Kind == "" {
		pane.Kind = PaneEmpty
	}
	id := command.TargetID
	if id == "" {
		id = strings.TrimSuffix(pane.ID, "-pane")
		if id == "" || id == pane.ID {
			store.nextFloatingSeq++
			id = formatFloatingID(store.nextFloatingSeq)
		}
	}
	if store.floatingIndex(id) >= 0 {
		return store, floatingCommandInvalid(command.Action, "floating already exists")
	}
	rect := command.Rect
	autoPlace := rect.X == 0 && rect.Y == 0
	placementBoundsW, placementBoundsH := floatingPlacementBounds(command.BoundsW, command.BoundsH)
	rect = defaultFloatingRect(rect, placementBoundsW, placementBoundsH)
	if autoPlace {
		rect = cascadeFloatingRect(rect, store.activeFloatings(), placementBoundsW, placementBoundsH)
	}
	rect = clampFloatingRect(rect, command.BoundsW, command.BoundsH)
	floating := FloatingPaneState{
		ID:      id,
		Title:   floatingTitle(command.Title, pane),
		Pane:    pane,
		Rect:    rect,
		Z:       store.nextFloatingZ() + 1,
		Active:  true,
		FitMode: FloatingFitManual,
	}
	floatings := append(cloneFloatings(store.activeFloatings()), floating)
	store = store.withActiveTabFloatings(floatings, id).EnsureDefaults()
	return store, FloatingCommandResult{Status: FloatingCommandOK, Action: command.Action, ID: id}
}

func (store ShellStore) focusRaiseFloating(id string, action FloatingCommandAction) (ShellStore, FloatingCommandResult) {
	index := store.floatingIndexOrActive(id)
	if index < 0 {
		return store, floatingCommandInvalid(action, "floating not found")
	}
	floatings := cloneFloatings(store.activeFloatings())
	id = floatings[index].ID
	if floatings[index].Collapsed {
		return store, floatingCommandInvalid(action, "floating hidden")
	}
	floatings[index].Z = store.nextFloatingZ() + 1
	store = store.withActiveTabFloatings(floatings, id).EnsureDefaults()
	return store, FloatingCommandResult{Status: FloatingCommandOK, Action: action, ID: id}
}

func (store ShellStore) deactivateFloating(action FloatingCommandAction) (ShellStore, FloatingCommandResult) {
	if store.activeFloatingID() == "" {
		return store, FloatingCommandResult{Status: FloatingCommandOK, Action: action}
	}
	store = store.withActiveTabFloatings(store.activeFloatings(), "").EnsureDefaults()
	return store, FloatingCommandResult{Status: FloatingCommandOK, Action: action}
}

func (store ShellStore) closeFloating(id string) (ShellStore, FloatingCommandResult) {
	index := store.floatingIndexOrActive(id)
	if index < 0 {
		return store, floatingCommandInvalid(FloatingCommandClose, "floating not found")
	}
	floatings := store.activeFloatings()
	id = floatings[index].ID
	next := make([]FloatingPaneState, 0, len(floatings)-1)
	for i, floating := range floatings {
		if i != index {
			next = append(next, floating)
		}
	}
	store = store.withActiveTabFloatings(next, topExpandedFloatingID(next)).EnsureDefaults()
	return store, FloatingCommandResult{Status: FloatingCommandOK, Action: FloatingCommandClose, ID: id}
}

func (store ShellStore) centerFloating(command FloatingCommand) (ShellStore, FloatingCommandResult) {
	index := store.floatingIndexOrActive(command.TargetID)
	if index < 0 {
		return store, floatingCommandInvalid(command.Action, "floating not found")
	}
	floatings := cloneFloatings(store.activeFloatings())
	rect := floatings[index].Rect
	floatings[index].Rect = centerFloatingRect(rect, command.BoundsW, command.BoundsH)
	id := floatings[index].ID
	return store.withActiveTabFloatings(floatings, store.activeFloatingID()).focusRaiseFloating(id, command.Action)
}

func (store ShellStore) toggleCollapseFloating(id string) (ShellStore, FloatingCommandResult) {
	index := store.floatingIndexForToggleCollapse(id)
	if index < 0 {
		return store, floatingCommandInvalid(FloatingCommandToggleCollapse, "floating not found")
	}
	floatings := cloneFloatings(store.activeFloatings())
	floatings[index].Collapsed = !floatings[index].Collapsed
	targetID := floatings[index].ID
	if floatings[index].Collapsed {
		// hide 后不能继续作为 active 输入目标，否则窗口不可见但键盘仍可能路由进去。
		store = store.withActiveTabFloatings(floatings, "").EnsureDefaults()
		return store, FloatingCommandResult{Status: FloatingCommandOK, Action: FloatingCommandToggleCollapse, ID: targetID}
	}
	return store.withActiveTabFloatings(floatings, store.activeFloatingID()).focusRaiseFloating(targetID, FloatingCommandToggleCollapse)
}

func (store ShellStore) summonFloating(command FloatingCommand) (ShellStore, FloatingCommandResult) {
	index := command.Index
	if command.TargetID != "" {
		index = store.floatingIndex(command.TargetID)
	}
	floatings := cloneFloatings(store.activeFloatings())
	if index < 0 || index >= len(floatings) {
		return store, floatingCommandInvalid(command.Action, "floating not found")
	}
	floatings[index].Collapsed = false
	id := floatings[index].ID
	return store.withActiveTabFloatings(floatings, store.activeFloatingID()).focusRaiseFloating(id, command.Action)
}

func (store ShellStore) moveFloating(command FloatingCommand) (ShellStore, FloatingCommandResult) {
	index := store.floatingIndexOrActive(command.TargetID)
	if index < 0 {
		return store, floatingCommandInvalid(command.Action, "floating not found")
	}
	floatings := cloneFloatings(store.activeFloatings())
	rect := floatings[index].Rect
	rect.X += command.DeltaX
	rect.Y += command.DeltaY
	floatings[index].Rect = clampFloatingRect(rect, command.BoundsW, command.BoundsH)
	floatings[index].FitMode = FloatingFitManual
	floatings[index].AutoFit = FloatingAutoFitState{}
	id := floatings[index].ID
	return store.withActiveTabFloatings(floatings, store.activeFloatingID()).focusRaiseFloating(id, command.Action)
}

func (store ShellStore) resizeFloating(command FloatingCommand) (ShellStore, FloatingCommandResult) {
	index := store.floatingIndexOrActive(command.TargetID)
	if index < 0 {
		return store, floatingCommandInvalid(command.Action, "floating not found")
	}
	floatings := cloneFloatings(store.activeFloatings())
	rect := floatings[index].Rect
	rect.W += command.DeltaW
	rect.H += command.DeltaH
	floatings[index].Rect = clampFloatingRect(rect, command.BoundsW, command.BoundsH)
	floatings[index].FitMode = FloatingFitManual
	floatings[index].AutoFit = FloatingAutoFitState{}
	id := floatings[index].ID
	return store.withActiveTabFloatings(floatings, store.activeFloatingID()).focusRaiseFloating(id, command.Action)
}

func topFloatingID(floatings []FloatingPaneState) string {
	if len(floatings) == 0 {
		return ""
	}
	top := floatings[0]
	for _, floating := range floatings {
		if floating.Z >= top.Z {
			top = floating
		}
	}
	return top.ID
}

func topExpandedFloatingID(floatings []FloatingPaneState) string {
	top := FloatingPaneState{}
	for _, floating := range floatings {
		if floating.Collapsed || floating.ID == "" {
			continue
		}
		if top.ID == "" || floating.Z >= top.Z {
			top = floating
		}
	}
	return top.ID
}

func floatingTitle(title string, pane PaneState) string {
	if title != "" {
		return title
	}
	if pane.Title != "" {
		return pane.Title
	}
	if pane.ID != "" {
		return pane.ID
	}
	return "floating"
}

func centerFloatingRect(rect FloatingRect, boundsW int, boundsH int) FloatingRect {
	if boundsW > 0 {
		rect.X = maxIntState(0, (boundsW-rect.W)/2)
	}
	if boundsH > 0 {
		rect.Y = maxIntState(0, (boundsH-rect.H)/2)
	}
	return clampFloatingRect(rect, boundsW, boundsH)
}

func clampFloatingRect(rect FloatingRect, boundsW int, boundsH int) FloatingRect {
	const minW = 16
	const minH = 4
	rect.W = maxIntState(minW, rect.W)
	rect.H = maxIntState(minH, rect.H)
	if boundsW > 0 {
		rect.W = minIntState(rect.W, maxIntState(minW, boundsW))
		rect.X = clampIntState(rect.X, 0, maxIntState(0, boundsW-rect.W))
	} else if rect.X < 0 {
		rect.X = 0
	}
	if boundsH > 0 {
		rect.H = minIntState(rect.H, maxIntState(minH, boundsH))
		rect.Y = clampIntState(rect.Y, 0, maxIntState(0, boundsH-rect.H))
	} else if rect.Y < 0 {
		rect.Y = 0
	}
	return rect
}

func clampIntState(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func minIntState(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxIntState(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func formatFloatingID(seq uint64) string {
	if seq == 0 {
		return "floating-0"
	}
	return "floating-" + formatToastID(seq)[len("toast-"):]
}

func floatingCommandInvalid(action FloatingCommandAction, reason string) FloatingCommandResult {
	return FloatingCommandResult{Status: FloatingCommandInvalid, Action: action, Reason: reason}
}

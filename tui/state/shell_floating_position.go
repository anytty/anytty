package state

func (store ShellStore) positionFloating(command FloatingCommand) (ShellStore, FloatingCommandResult) {
	index := store.floatingIndexOrActive(command.TargetID)
	if index < 0 {
		return store, floatingCommandInvalid(command.Action, "floating not found")
	}
	floatings := cloneFloatings(store.activeFloatings())
	rect := floatings[index].Rect
	switch command.PositionX {
	case TerminalViewAlignStart:
		rect.X = 0
	case TerminalViewAlignCenter:
		rect.X = (command.BoundsW - rect.W) / 2
	case TerminalViewAlignEnd:
		rect.X = command.BoundsW - rect.W
	}
	switch command.PositionY {
	case TerminalViewAlignStart:
		rect.Y = 0
	case TerminalViewAlignCenter:
		rect.Y = (command.BoundsH - rect.H) / 2
	case TerminalViewAlignEnd:
		rect.Y = command.BoundsH - rect.H
	}
	floatings[index].Rect = clampFloatingRect(rect, command.BoundsW, command.BoundsH)
	floatings[index].FitMode = FloatingFitManual
	floatings[index].AutoFit = FloatingAutoFitState{}
	id := floatings[index].ID
	return store.withActiveTabFloatings(floatings, store.activeFloatingID()).focusRaiseFloating(id, command.Action)
}

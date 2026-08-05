package app

import actiondomain "github.com/anytty/anytty/tui/action"

func shortcutTestMessage(id actiondomain.ID, paneID string, floating bool, row int) ShellShortcutActionMsg {
	return ShellShortcutActionMsg{
		Invocation: actiondomain.Invocation{ID: id, SourceActionID: id.String()},
		Surface:    &ShortcutSurfaceContext{ExplicitTarget: true, PaneID: paneID, Floating: floating, Row: row, HasRow: true},
	}
}

func shortcutActiveTargetTestMessage(id actiondomain.ID) ShellShortcutActionMsg {
	return ShellShortcutActionMsg{
		Invocation: actiondomain.Invocation{ID: id, SourceActionID: id.String()},
		Surface:    &ShortcutSurfaceContext{Row: -1},
	}
}

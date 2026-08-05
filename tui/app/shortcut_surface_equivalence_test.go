package app

import (
	"reflect"
	"testing"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
)

func TestKeyboardAndClickCanonicalActionUseSameReducerPath(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	invocation := actiondomain.Invocation{ID: "system.toggle_header", SourceActionID: "system.toggle_header"}
	intent, ok := shortcutIntentForInvocation(invocation, input.InputEvent{})
	if !ok {
		t.Fatal("system.toggle_header must have a canonical keyboard handler")
	}
	keyboardRoot, keyboardEffects := reduceShortcutIntentWithContext(root, intent, -1)
	clickRoot, clickEffects := NewShellReducer()(root, ShellShortcutActionMsg{Invocation: invocation, Surface: &ShortcutSurfaceContext{Row: -1}})
	if !reflect.DeepEqual(keyboardRoot, clickRoot) {
		t.Fatalf("keyboard and click must reach the same reducer state:\nkeyboard=%#v\nclick=%#v", keyboardRoot, clickRoot)
	}
	if len(keyboardEffects) != len(clickEffects) {
		t.Fatalf("keyboard and click must expose the same effect boundary: keyboard=%#v click=%#v", keyboardEffects, clickEffects)
	}
}

func TestSurfaceActionWithMissingRequiredTargetDoesNotFallbackToActivePane(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	for _, id := range []actiondomain.ID{"panel.close", "tab.close", actiondomain.ActionFloatingRaise} {
		invocation := actiondomain.Invocation{ID: id, SourceActionID: id.String()}
		surfaceRoot, surfaceEffects := NewShellReducer()(root, ShellShortcutActionMsg{
			Invocation: invocation,
			Surface:    &ShortcutSurfaceContext{ExplicitTarget: true, Row: -1},
		})
		if !reflect.DeepEqual(surfaceRoot, root) || len(surfaceEffects) != 0 {
			t.Fatalf("targetless surface %s must fail closed instead of using active target: root=%#v effects=%#v", id, surfaceRoot, surfaceEffects)
		}
	}

	invocation := actiondomain.Invocation{ID: "panel.close", SourceActionID: "panel.close"}
	keyboardRoot, keyboardEffects := NewShellReducer()(root, ShellShortcutActionMsg{Invocation: invocation})
	if reflect.DeepEqual(keyboardRoot, root) && len(keyboardEffects) == 0 {
		t.Fatal("keyboard close must retain active-target semantics when no surface context exists")
	}
}

func TestEmptyTabAttachSurfaceActionAllowsExplicitNoPaneContext(t *testing.T) {
	shell := state.DefaultShell()
	shell.Workspace.ActiveTabID = "tab-empty"
	shell.Workspace.Tabs = append(shell.Workspace.Tabs, state.TabState{ID: "tab-empty", Title: "empty"})
	shell.ActivePaneID = ""

	next, effects := NewShellReducer()(state.Root{Shell: shell}, ShellShortcutActionMsg{
		Invocation: actiondomain.Invocation{ID: actiondomain.ActionEmptyAttach, SourceActionID: actiondomain.ActionEmptyAttach.String()},
		Surface:    &ShortcutSurfaceContext{ExplicitTarget: true, Row: -1},
	})
	if !next.Shell.Overlay.Open || next.Shell.Overlay.Kind != state.OverlayTerminalPicker {
		t.Fatalf("empty-tab attach must open picker without inventing a pane target: %#v", next.Shell.Overlay)
	}
	if len(effects) == 0 {
		t.Fatal("empty-tab attach must request terminal picker data")
	}
}

func TestExplicitRowSurfaceActionsDistinguishMissingRowFromRowZero(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell().OpenTerminalPool(),
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{
			EndpointID: "local", TerminalID: "term-1", Title: "shell", State: "running",
		}}},
	}

	assertRowContract := func(t *testing.T, fixture state.Root, id actiondomain.ID, upperBound int, validRow int) {
		t.Helper()
		invocation := actiondomain.Invocation{ID: id, SourceActionID: id.String()}
		missingRoot, missingEffects := NewShellReducer()(fixture, ShellShortcutActionMsg{
			Invocation: invocation,
			Surface:    &ShortcutSurfaceContext{ExplicitTarget: true},
		})
		if !reflect.DeepEqual(missingRoot, fixture) || len(missingEffects) != 0 {
			t.Fatalf("%s without HasRow must fail closed: root=%#v effects=%#v", id, missingRoot, missingEffects)
		}
		for _, invalidRow := range []int{-1, upperBound} {
			invalidRoot, invalidEffects := NewShellReducer()(fixture, ShellShortcutActionMsg{
				Invocation: invocation,
				Surface:    &ShortcutSurfaceContext{ExplicitTarget: true, HasRow: true, Row: invalidRow},
			})
			if !reflect.DeepEqual(invalidRoot, fixture) || len(invalidEffects) != 0 {
				t.Fatalf("%s with invalid row %d must fail closed: root=%#v effects=%#v", id, invalidRow, invalidRoot, invalidEffects)
			}
		}

		validRoot, validEffects := NewShellReducer()(fixture, ShellShortcutActionMsg{
			Invocation: invocation,
			Surface:    &ShortcutSurfaceContext{ExplicitTarget: true, HasRow: true, Row: validRow},
		})
		if reflect.DeepEqual(validRoot, fixture) && len(validEffects) == 0 {
			t.Fatalf("%s with explicit valid row %d must remain executable", id, validRow)
		}
	}

	t.Run("specialized pool selection", func(t *testing.T) {
		assertRowContract(t, root, actiondomain.ActionTerminalPoolSelect, len(state.TerminalPoolPageItems(root)), 0)
	})
	t.Run("generic picker attach", func(t *testing.T) {
		pickerRoot := root
		pickerRoot.Shell = state.DefaultShell().OpenTerminalPicker()
		items := state.TerminalPickerItems(pickerRoot)
		terminalRow := -1
		for index, item := range items {
			if !item.CreateNew && item.TerminalID != "" {
				terminalRow = index
				break
			}
		}
		if terminalRow < 0 {
			t.Fatalf("picker fixture must contain terminal row: %#v", items)
		}
		assertRowContract(t, pickerRoot, "terminal_picker.attach", len(items), terminalRow)
	})
	t.Run("picker row kind", func(t *testing.T) {
		pickerRoot := root
		pickerRoot.Shell = state.DefaultShell().OpenTerminalPicker()
		items := state.TerminalPickerItems(pickerRoot)
		terminalRow, createRow := -1, -1
		for index, item := range items {
			if item.CreateNew {
				createRow = index
			} else if item.TerminalID != "" {
				terminalRow = index
			}
		}
		if terminalRow < 0 || createRow < 0 {
			t.Fatalf("picker fixture must contain terminal and create rows: %#v", items)
		}
		for _, tc := range []struct {
			id  actiondomain.ID
			row int
		}{
			{id: actiondomain.ActionTerminalPickerNew, row: terminalRow},
			{id: "terminal_picker.attach", row: createRow},
		} {
			invocation := actiondomain.Invocation{ID: tc.id, SourceActionID: tc.id.String()}
			next, effects := NewShellReducer()(pickerRoot, ShellShortcutActionMsg{
				Invocation: invocation,
				Surface:    &ShortcutSurfaceContext{ExplicitTarget: true, HasRow: true, Row: tc.row},
			})
			if !reflect.DeepEqual(next, pickerRoot) || len(effects) != 0 {
				t.Fatalf("%s must reject in-range wrong picker row kind at %d: root=%#v effects=%#v", tc.id, tc.row, next, effects)
			}
		}
	})
}

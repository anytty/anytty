package app

import (
	"context"
	"reflect"
	"testing"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
)

func focusedFloatingShortcutRoot(t *testing.T) state.Root {
	t.Helper()
	shell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-tiled-2"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}).
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-tiled")
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "cloud", Kind: state.PaneTerminalLive, TerminalID: "term-floating"},
		Rect:     state.FloatingRect{X: 8, Y: 4, W: 42, H: 10},
		Source:   state.PaneCommandSourceTest,
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating fixture: %#v", result)
	}
	root := state.Root{Shell: shell, Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 30}}
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewEndpointPaneTerminalView("local", state.DefaultPaneID, "term-tiled", 7, 50, 24, state.TerminalResizeRoleOwner, "surface-tiled", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindFloating(state.NewEndpointFloatingTerminalView("cloud", "floating-1", "floating-pane-1", "term-floating", 8, 40, 8, state.TerminalResizeRoleFollower, "surface-floating", state.TerminalFloatingViewID("floating-1"), false))
	return root
}

func TestPaneKeyboardCloseTargetsFocusedFloating(t *testing.T) {
	root := focusedFloatingShortcutRoot(t)
	root.Shell = root.Shell.SetInteractionMode(state.InteractionModePane)
	beforePanes := append([]state.PaneState(nil), root.Shell.ReadonlyDefaults().Workspace.Tabs[0].Panes...)

	next, _ := NewUIInputReducer()(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "w"}})

	if len(next.Shell.ActiveFloatings()) != 0 {
		t.Fatalf("pane close key must close the focused floating, shell=%#v", next.Shell)
	}
	if got := next.Shell.ReadonlyDefaults().Workspace.Tabs[0].Panes; !reflect.DeepEqual(got, beforePanes) {
		t.Fatalf("pane close key must not mutate tiled panes behind the floating:\ngot=%#v\nwant=%#v", got, beforePanes)
	}
	if _, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID); !ok {
		t.Fatal("closing a floating must preserve the tiled terminal binding")
	}
}

func TestPaneKeyboardLifecycleTargetsFocusedFloating(t *testing.T) {
	for _, tc := range []struct {
		name   string
		char   string
		assert func(*testing.T, Msg)
	}{
		{
			name: "reconnect",
			char: "r",
			assert: func(t *testing.T, msg Msg) {
				request, ok := msg.(TerminalPoolReconnectRequestMsg)
				if !ok || request.EndpointID != "cloud" || request.TerminalID != "term-floating" || request.TargetFloatingID != "floating-1" || request.TargetPaneID != "floating-pane-1" {
					t.Fatalf("reconnect must target the focused floating, got %#v", msg)
				}
			},
		},
		{
			name: "restart",
			char: "R",
			assert: func(t *testing.T, msg Msg) {
				request, ok := msg.(TerminalPoolRestartRequestMsg)
				if !ok || request.EndpointID != "cloud" || request.TerminalID != "term-floating" {
					t.Fatalf("restart must target the focused floating, got %#v", msg)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := focusedFloatingShortcutRoot(t)
			root.Shell = root.Shell.SetInteractionMode(state.InteractionModePane)
			_, effects := NewUIInputReducer()(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: tc.char}})
			tc.assert(t, runFirstNonStickyTimeoutEffect(t, effects))
		})
	}
}

func TestPaneKeyboardDetachKeepsFocusedFloatingAndClearsItsBinding(t *testing.T) {
	root := focusedFloatingShortcutRoot(t)
	root.Shell = root.Shell.SetInteractionMode(state.InteractionModePane)

	next, _ := NewUIInputReducer()(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "d"}})

	floating, ok := next.Shell.FloatingByID("floating-1")
	if !ok || floating.Pane.Kind != state.PaneEmpty || floating.Pane.TerminalID != "" {
		t.Fatalf("detach must retain an empty focused floating, got %#v ok=%v", floating, ok)
	}
	if _, ok := next.TerminalViews.FloatingBinding("floating-1"); ok {
		t.Fatal("detach must remove only the focused floating terminal binding")
	}
	if binding, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID); !ok || binding.TerminalID != "term-tiled" {
		t.Fatalf("detach must preserve the tiled binding behind the floating, got %#v ok=%v", binding, ok)
	}
}

func TestTiledSplitShortcutDoesNotPassThroughFocusedFloating(t *testing.T) {
	root := focusedFloatingShortcutRoot(t)
	root.Shell = root.Shell.SetInteractionMode(state.InteractionModePane)
	before := root.Shell.ReadonlyDefaults().Workspace.Tabs[0]

	next, _ := NewUIInputReducer()(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "%"}})

	after := next.Shell.ReadonlyDefaults().Workspace.Tabs[0]
	if !reflect.DeepEqual(after.Panes, before.Panes) || !reflect.DeepEqual(after.RootSplit, before.RootSplit) {
		t.Fatalf("split must not mutate the tiled layout behind a focused floating:\nafter=%#v\nbefore=%#v", after, before)
	}
	if next.Shell.ActiveFloatingID() != "floating-1" {
		t.Fatalf("split must keep floating focus, got %q", next.Shell.ActiveFloatingID())
	}
}

func TestResizeKeyboardPositionsFocusedFloating(t *testing.T) {
	for _, tc := range []struct {
		name  string
		event input.InputEvent
		wantX int
		wantY int
	}{
		{name: "left", event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyLeft}, wantX: 6, wantY: 4},
		{name: "right", event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyRight}, wantX: 10, wantY: 4},
		{name: "up", event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyUp}, wantX: 8, wantY: 3},
		{name: "down", event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyDown}, wantX: 8, wantY: 5},
		{name: "large left", event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "H"}, wantX: 2, wantY: 4},
		{name: "pan right", event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "D"}, wantX: 10, wantY: 4},
		{name: "align left", event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "0"}, wantX: 0, wantY: 4},
		{name: "align right", event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "$"}, wantX: 58, wantY: 4},
		{name: "align top", event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "^"}, wantX: 8, wantY: 0},
		{name: "align bottom", event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "B"}, wantX: 8, wantY: 20},
		{name: "center", event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "m"}, wantX: 29, wantY: 10},
		{name: "center x", event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "|"}, wantX: 29, wantY: 4},
		{name: "center y", event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "_"}, wantX: 8, wantY: 10},
		{name: "reset position", event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "r"}, wantX: 29, wantY: 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := focusedFloatingShortcutRoot(t)
			root.Shell = root.Shell.SetInteractionMode(state.InteractionModeResize)

			next, _ := NewUIInputReducer()(root, InputMsg{Event: tc.event})

			floating, ok := next.Shell.FloatingByID("floating-1")
			if !ok || floating.Rect.X != tc.wantX || floating.Rect.Y != tc.wantY {
				t.Fatalf("resize action must position focused floating at %d,%d, got %#v ok=%v", tc.wantX, tc.wantY, floating.Rect, ok)
			}
			if next.Shell.ActiveFloatingID() != "floating-1" {
				t.Fatalf("resize action must keep floating focus, got %q", next.Shell.ActiveFloatingID())
			}
		})
	}
}

func TestResizeLayoutToggleDoesNotMutateFocusedFloatingTerminalLayout(t *testing.T) {
	root := focusedFloatingShortcutRoot(t)
	root.Shell = root.Shell.SetInteractionMode(state.InteractionModeResize)
	before, _ := root.TerminalViews.FloatingBinding("floating-1")

	next, _ := NewUIInputReducer()(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: " "}})

	after, _ := next.TerminalViews.FloatingBinding("floating-1")
	if !reflect.DeepEqual(after.Layout, before.Layout) {
		t.Fatalf("floating position mode must not toggle terminal content layout, before=%#v after=%#v", before.Layout, after.Layout)
	}
	toasts := next.Shell.ReadonlyDefaults().Toasts
	if len(toasts) == 0 || toasts[len(toasts)-1].Body != "not available for active floating" {
		t.Fatalf("unsupported floating position action must explain itself, toasts=%#v", toasts)
	}
}

func TestActiveTargetFooterCloseUsesFocusedFloating(t *testing.T) {
	root := focusedFloatingShortcutRoot(t)
	next, _ := NewShellReducer()(root, shortcutActiveTargetTestMessage("panel.close"))
	if len(next.Shell.ActiveFloatings()) != 0 || len(next.Shell.ReadonlyDefaults().Workspace.Tabs[0].Panes) != 2 {
		t.Fatalf("active-target footer close must close only the focused floating, shell=%#v", next.Shell)
	}
}

func TestFloatingKillAndCloseCarriesConditionalFloatingTarget(t *testing.T) {
	root := focusedFloatingShortcutRoot(t)
	_, effects := NewShellReducer()(root, shortcutActiveTargetTestMessage("panel.kill_and_close"))
	request, ok := runFirstNonStickyTimeoutEffect(t, effects).(TerminalPoolKillRequestMsg)
	if !ok || request.EndpointID != "cloud" || request.TerminalID != "term-floating" || request.FloatingID != "floating-1" || !request.CloseOnSuccess || request.PaneID != "" {
		t.Fatalf("kill-and-close must carry the focused floating identity, got %#v", request)
	}

	resultRoot, resultEffects := reduceTerminalPoolKillResult(root, TerminalPoolKillResultMsg{
		EndpointID: "cloud", TerminalID: "term-floating", FloatingID: "floating-1", CloseOnSuccess: true,
	})
	if len(resultEffects) != 2 {
		t.Fatalf("successful floating kill-and-close must schedule refresh and conditional close, effects=%#v", resultEffects)
	}
	closeMsg, ok := resultEffects[1].(FuncEffect).Run(context.Background()).(ShellCloseFloatingIfTerminalRefMsg)
	if !ok || closeMsg.FloatingID != "floating-1" {
		t.Fatalf("kill result must schedule conditional floating close, got %#v", closeMsg)
	}
	closed, _ := NewShellReducer()(resultRoot, closeMsg)
	if len(closed.Shell.ActiveFloatings()) != 0 {
		t.Fatalf("conditional close must remove the still-matching floating, shell=%#v", closed.Shell)
	}
}

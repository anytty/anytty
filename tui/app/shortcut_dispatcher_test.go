package app

import (
	"github.com/anytty/anytty/tui/testkit"
	"strconv"
	"testing"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/shortcut"
	"github.com/anytty/anytty/tui/state"
)

func TestShortcutDispatcherPreservesParameterizedAndDirectionalSemantics(t *testing.T) {
	cases := []struct {
		source  string
		kind    input.IntentKind
		command string
		action  input.ShellAction
		reason  string
	}{
		{source: "panel.focus_prev", kind: input.IntentPaneCommand, command: "pane focus-prev"},
		{source: "resize.pan_left", kind: input.IntentWorkbenchCommand, command: "terminal layout pan-left"},
		{source: "tab.jump.3", kind: input.IntentWorkbenchCommand, command: "tab jump 3"},
		{source: "floating.summon.3", kind: input.IntentShellAction, action: input.ShellActionFloatingSummon, reason: "3"},
	}
	for _, tc := range cases {
		invocation, _, err := actiondomain.ParseInvocation(tc.source)
		if err != nil {
			t.Fatal(err)
		}
		intent, ok := shortcutIntentForInvocation(invocation, input.InputEvent{})
		if !ok || intent.Kind != tc.kind || intent.Command != tc.command || intent.Action != tc.action || intent.Reason != tc.reason || intent.Invocation.ID != invocation.ID {
			t.Fatalf("dispatch %q: %#v ok=%v", tc.source, intent, ok)
		}
	}
}

func TestActionHandlerRegistryCoversEveryShortcutAction(t *testing.T) {
	for id := range shortcut.Policies() {
		spec, ok := actiondomain.SpecByID(id)
		if !ok {
			t.Fatalf("shortcut action %q has no canonical action spec", id)
		}
		invocation := actiondomain.Invocation{ID: id}
		if spec.Param != nil {
			invocation.Params = map[string]int{spec.Param.Name: spec.Param.Min}
		}
		intent, ok := shortcutIntentForInvocation(invocation, input.InputEvent{})
		if !ok || intent.Kind == input.IntentNone || intent.Kind == input.IntentShortcutAction {
			t.Fatalf("shortcut action %q has no concrete app handler: %#v", id, intent)
		}
	}
}

func TestShellShortcutActionMessageUsesSameDispatcherAsKeyboard(t *testing.T) {
	invocation, _, err := actiondomain.ParseInvocation("menu.tab")
	if err != nil {
		t.Fatal(err)
	}
	next, _ := NewShellReducer()(state.Root{Shell: state.DefaultShell()}, ShellShortcutActionMsg{Invocation: invocation})
	if next.Shell.InteractionMode != state.InteractionModeTab {
		t.Fatalf("shortcut click should enter tab mode through dispatcher, got %#v", next.Shell)
	}
}

func TestCopyShortcutInvocationIsOwnedByCopyReducer(t *testing.T) {
	invocation, _, err := actiondomain.ParseInvocation("copy.enter")
	if err != nil {
		t.Fatal(err)
	}
	core := &testkit.FakeCoreClient{}
	reducer := ComposeReducers(NewShellReducer(), NewCopyModeReducer(CopyModeDeps{Core: core, Rows: 20}))
	root := state.Root{
		Shell:    state.DefaultShell(),
		Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 24},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
	}

	next, effects := reducer(root, ShellShortcutActionMsg{Invocation: invocation})
	if !next.CopyMode.Entering || next.History.Pending == nil {
		t.Fatalf("copy invocation should be consumed by copy reducer exactly once, got %#v", next.CopyMode)
	}
	requestEffects := 0
	for _, effect := range effects {
		if fn, ok := effect.(FuncEffect); ok && fn.Token == copyModeHistoryRequestToken(state.TerminalPaneViewID(state.DefaultPaneID)) {
			requestEffects++
		}
	}
	if requestEffects != 1 {
		t.Fatalf("copy invocation should emit one authoritative history request effect, got %#v", effects)
	}
}

func TestOverlayShortcutMessagePreservesRowContext(t *testing.T) {
	invocation, _, err := actiondomain.ParseInvocation("workbench_tree.open")
	if err != nil {
		t.Fatal(err)
	}
	root := state.Root{Shell: state.DefaultShell().SplitActivePane(
		state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneEmpty},
		state.SplitDirectionVertical,
	).OpenWorkbenchTree()}
	items := state.WorkbenchTreeItems(root)
	row := -1
	for index, item := range items {
		if item.PaneID == "pane-2" {
			row = index
			break
		}
	}
	if row < 0 {
		t.Fatalf("missing pane-2 workbench item: %#v", items)
	}

	next, _ := NewShellReducer()(root, ShellShortcutActionMsg{Invocation: invocation, Surface: &ShortcutSurfaceContext{ExplicitTarget: true, Row: row, HasRow: true}})
	if next.Shell.ActivePaneID != "pane-2" || next.Shell.Overlay.Open {
		t.Fatalf("overlay invocation lost row target context: %#v", next.Shell)
	}
}

func TestFloatingSummonShortcutUsesInvocationIndexWithoutRowFallback(t *testing.T) {
	invocation, _, err := actiondomain.ParseInvocation("floating.summon.3")
	if err != nil {
		t.Fatal(err)
	}
	root := state.Root{Shell: state.DefaultShell()}
	for index := 1; index <= 3; index++ {
		var result state.FloatingCommandResult
		root.Shell, result = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
			Action:   state.FloatingCommandCreate,
			TargetID: "floating-" + strconv.Itoa(index),
			Pane:     state.PaneState{ID: "floating-pane-" + strconv.Itoa(index), Kind: state.PaneEmpty},
		})
		if result.Status != state.FloatingCommandOK {
			t.Fatalf("create floating %d: %#v", index, result)
		}
	}

	next, _ := NewShellReducer()(root, ShellShortcutActionMsg{Invocation: invocation, Surface: &ShortcutSurfaceContext{ExplicitTarget: true, Row: 0, HasRow: true}})
	if next.Shell.ActiveFloatingID() != "floating-3" {
		t.Fatalf("parameterized summon must use invocation index, got %q", next.Shell.ActiveFloatingID())
	}
}

func TestShortcutDispatcherKeepsKillActionsDistinct(t *testing.T) {
	for source, wantCommand := range map[string]string{
		"panel.kill":           "pane kill confirm=accepted",
		"panel.kill_and_close": "pane close-kill confirm=accepted",
	} {
		invocation, _, err := actiondomain.ParseInvocation(source)
		if err != nil {
			t.Fatal(err)
		}
		intent, ok := shortcutIntentForInvocation(invocation, input.InputEvent{})
		if !ok || intent.Kind != input.IntentPaneCommand || string(intent.Invocation.ID) != source || intent.Command != wantCommand {
			t.Fatalf("dispatch %q lost action identity: %#v", source, intent)
		}
	}
}

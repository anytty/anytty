package app

import (
	"testing"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

func TestParsePaneMiniCommandCoversStructuralActions(t *testing.T) {
	command, err := ParsePaneMiniCommand("pane split-right pane=main new-pane=right")
	if err != nil {
		t.Fatalf("parse split: %v", err)
	}
	if command.Action != state.PaneCommandSplit || command.SplitDirection != state.SplitDirectionVertical || command.Target.PaneID != "main" || command.NewPane.ID != "right" {
		t.Fatalf("unexpected split command %#v", command)
	}

	command, err = ParsePaneMiniCommand("pane resize right delta=4 pane=right")
	if err != nil {
		t.Fatalf("parse resize: %v", err)
	}
	if command.Action != state.PaneCommandResize || command.ResizeDirection != state.PaneResizeRight || command.Delta != 4 || command.Source != state.PaneCommandSourceCLIMini {
		t.Fatalf("unexpected resize command %#v", command)
	}

	command, err = ParsePaneMiniCommand("pane set-size pane=right cols=24")
	if err != nil {
		t.Fatalf("parse set-size: %v", err)
	}
	if command.Action != state.PaneCommandSetSize || command.SizeMode != state.PaneSizeCells || command.Cols != 24 {
		t.Fatalf("unexpected set-size command %#v", command)
	}

	command, err = ParsePaneMiniCommand("pane close-kill pane=right confirm=accepted")
	if err != nil {
		t.Fatalf("parse close-kill: %v", err)
	}
	if command.Action != state.PaneCommandCloseAndKill || command.Confirm != state.PaneConfirmAccepted {
		t.Fatalf("unexpected close-kill command %#v", command)
	}

	command, err = ParsePaneMiniCommand("pane kill pane=right confirm=accepted")
	if err != nil {
		t.Fatalf("parse kill: %v", err)
	}
	if command.Action != state.PaneCommandKill || command.Confirm != state.PaneConfirmAccepted {
		t.Fatalf("unexpected kill command %#v", command)
	}
}

func TestPaneCommandAdaptersFromHitRegionAndIntent(t *testing.T) {
	command, ok := PaneCommandFromHitRegion(render.HitRegion{Kind: render.HitRegionPaneAction, PaneID: "pane-1", ActionID: "pane.close"})
	if ok {
		t.Fatalf("pane.close must route through workbench command, got command=%#v", command)
	}

	for _, actionID := range []string{"pane.split-down", "pane.split-right", render.ActionPaneZoom.String()} {
		if command, ok = PaneCommandFromHitRegion(render.HitRegion{Kind: render.HitRegionPaneAction, PaneID: "pane-1", ActionID: actionID}); ok {
			t.Fatalf("pane action %q must route through canonical dispatcher, got %#v", actionID, command)
		}
	}

	command, ok = PaneCommandFromHitRegion(render.HitRegion{Kind: render.HitRegionPaneResize, PaneID: "pane-1", SplitPath: "root/1"})
	if !ok || command.Action != state.PaneCommandResize || command.ResizeDirection != state.PaneResizeRight || command.Delta != 1 || command.ResizeSplitPath != "root/1" {
		t.Fatalf("unexpected resize hit command command=%#v ok=%v", command, ok)
	}

	command, ok = PaneCommandFromIntent(input.Intent{Kind: input.IntentPaneCommand, Command: "pane zoom pane-1"})
	if !ok || command.Action != state.PaneCommandZoom || command.Target.PaneID != "pane-1" || command.Source != state.PaneCommandSourceKeyboard {
		t.Fatalf("unexpected intent command command=%#v ok=%v", command, ok)
	}
}

func TestPaneAndResizeModeCommandHelpers(t *testing.T) {
	command, ok := PaneModeCommand(state.PaneCommandToggleZoom, "pane-1")
	if !ok || command.Action != state.PaneCommandToggleZoom || command.Target.PaneID != "pane-1" || command.Source != state.PaneCommandSourceKeyboard {
		t.Fatalf("unexpected pane mode command command=%#v ok=%v", command, ok)
	}

	command, ok = ResizeModeCommand("pane-1", state.PaneResizeLeft, 3)
	if !ok || command.Action != state.PaneCommandResize || command.ResizeDirection != state.PaneResizeLeft || command.Delta != 3 {
		t.Fatalf("unexpected resize mode command command=%#v ok=%v", command, ok)
	}
}

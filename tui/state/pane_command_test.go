package state

import "testing"

func TestPaneCommandDefaultsTargetToActivePane(t *testing.T) {
	shell := DefaultShell()
	command := PaneCommand{Action: PaneCommandFocus}.WithDefaults(shell)

	if command.Target.WorkspaceID != DefaultWorkspaceID || command.Target.TabID != DefaultTabID || command.Target.PaneID != DefaultPaneID {
		t.Fatalf("expected active target defaults, got %#v", command.Target)
	}
}

func TestPaneCommandValidateSplitRequiresDirectionAndNewPane(t *testing.T) {
	shell := DefaultShell()
	missingDirection := PaneCommand{Action: PaneCommandSplit, NewPane: PaneState{ID: "pane-2"}}
	if result := missingDirection.Validate(shell); result.Status != PaneCommandInvalid || result.Reason != "invalid split direction" {
		t.Fatalf("expected invalid split direction, got %#v", result)
	}

	valid := PaneCommand{Action: PaneCommandSplit, SplitDirection: SplitDirectionVertical, NewPane: PaneState{ID: "pane-2"}}
	if result := valid.Validate(shell); result.Status != PaneCommandOK {
		t.Fatalf("expected valid split command, got %#v", result)
	}
}

func TestPaneCommandValidateRejectsUnknownTarget(t *testing.T) {
	shell := DefaultShell()
	command := PaneCommand{Action: PaneCommandFocus, Target: PaneCommandTarget{PaneID: "missing"}}

	if result := command.Validate(shell); result.Status != PaneCommandInvalid || result.Reason != "target pane not found" {
		t.Fatalf("expected missing target, got %#v", result)
	}
}

func TestPaneCommandValidateCloseAndKillRequiresAcceptedConfirmation(t *testing.T) {
	shell := DefaultShell().SplitActivePane(PaneState{ID: "pane-2"}, SplitDirectionVertical)
	command := PaneCommand{Action: PaneCommandCloseAndKill}

	if result := command.Validate(shell); result.Status != PaneCommandNeedsConfirmation {
		t.Fatalf("expected confirmation requirement, got %#v", result)
	}
	command.Confirm = PaneConfirmAccepted
	if result := command.Validate(shell); result.Status != PaneCommandOK {
		t.Fatalf("expected accepted destructive command, got %#v", result)
	}
}

func TestPaneCommandKillDoesNotRequireClosablePane(t *testing.T) {
	shell := DefaultShell()
	for _, action := range []PaneCommandAction{PaneCommandKill, PaneCommandCloseAndKill} {
		command := PaneCommand{Action: action, Target: PaneCommandTarget{PaneID: DefaultPaneID}, Confirm: PaneConfirmAccepted}
		next, result := shell.ApplyPaneCommand(command)
		if result.Status != PaneCommandOK || !next.HasPane(PaneCommandTarget{PaneID: DefaultPaneID}) {
			t.Fatalf("%s should validate on last pane without closing it, shell=%#v result=%#v", action, next, result)
		}
	}
}

func TestPaneCommandValidateResizeAndSetSizeParameters(t *testing.T) {
	shell := DefaultShell()
	resize := PaneCommand{Action: PaneCommandResize, ResizeDirection: PaneResizeRight, Delta: 4}
	if result := resize.Validate(shell); result.Status != PaneCommandOK {
		t.Fatalf("expected valid resize, got %#v", result)
	}

	ratio := PaneCommand{Action: PaneCommandSetSize, SizeMode: PaneSizeRatio, Ratio: 0.5}
	if result := ratio.Validate(shell); result.Status != PaneCommandOK {
		t.Fatalf("expected valid ratio set size, got %#v", result)
	}
	cells := PaneCommand{Action: PaneCommandSetSize, SizeMode: PaneSizeCells, Cols: 80}
	if result := cells.Validate(shell); result.Status != PaneCommandOK {
		t.Fatalf("expected valid cell set size, got %#v", result)
	}
	invalid := PaneCommand{Action: PaneCommandSetSize, SizeMode: PaneSizeRatio, Ratio: 1.5}
	if result := invalid.Validate(shell); result.Status != PaneCommandInvalid || result.Reason != "invalid size ratio" {
		t.Fatalf("expected invalid ratio, got %#v", result)
	}
}

func TestApplyPaneCommandReusesExistingPresentationAndSplitState(t *testing.T) {
	shell, result := DefaultShell().ApplyPaneCommand(PaneCommand{
		Action:         PaneCommandSplit,
		SplitDirection: SplitDirectionHorizontal,
		NewPane:        PaneState{ID: "pane-2", Title: "logs", Kind: PaneTerminalLive},
	})
	if result.Status != PaneCommandOK {
		t.Fatalf("expected split ok, got %#v", result)
	}
	if shell.ActivePaneID != "pane-2" || len(shell.Workspace.Tabs[0].Panes) != 2 {
		t.Fatalf("expected split pane state, got %#v", shell)
	}

	shell, result = shell.ApplyPaneCommand(PaneCommand{Action: PaneCommandSetPresentation, Presentation: PanelPresentationSplitLine})
	if result.Status != PaneCommandOK || shell.PanelPresentation != PanelPresentationSplitLine {
		t.Fatalf("expected split-line presentation, shell=%#v result=%#v", shell, result)
	}
}

func TestApplyPaneCommandSplitsExplicitTargetPane(t *testing.T) {
	shell := DefaultShell().
		SplitActivePane(PaneState{ID: "pane-2"}, SplitDirectionVertical).
		FocusPane(PaneCommandTarget{PaneID: DefaultPaneID})

	next, result := shell.ApplyPaneCommand(PaneCommand{
		Action:         PaneCommandSplit,
		Target:         PaneCommandTarget{PaneID: "pane-2"},
		SplitDirection: SplitDirectionHorizontal,
		NewPane:        PaneState{ID: "pane-3"},
	})

	if result.Status != PaneCommandOK {
		t.Fatalf("expected split ok, got %#v", result)
	}
	rightChild := next.Workspace.Tabs[0].RootSplit.Children[1]
	if rightChild.Direction != SplitDirectionHorizontal || len(rightChild.Children) != 2 {
		t.Fatalf("expected target pane to be split in place, got %#v", next.Workspace.Tabs[0].RootSplit)
	}
	if rightChild.Children[0].PaneID != "pane-2" || rightChild.Children[1].PaneID != "pane-3" {
		t.Fatalf("expected split to preserve target pane order, got %#v", rightChild.Children)
	}
	if next.ActivePaneID != "pane-3" {
		t.Fatalf("expected new pane active, got %#v", next.ActivePaneID)
	}
}

func TestApplyPaneCommandAppliesResizeGeometryHints(t *testing.T) {
	shell := DefaultShell().SplitActivePane(PaneState{ID: "pane-2"}, SplitDirectionVertical)
	next, result := shell.ApplyPaneCommand(PaneCommand{Action: PaneCommandResize, ResizeDirection: PaneResizeRight, Delta: 2})

	if result.Status != PaneCommandOK {
		t.Fatalf("expected contract-valid resize, got %#v", result)
	}
	if next.Workspace.Tabs[0].RootSplit.BiasCells != 2 {
		t.Fatalf("expected resize bias on split root, got %#v", next.Workspace.Tabs[0].RootSplit)
	}
}

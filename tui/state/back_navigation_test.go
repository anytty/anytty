package state

import "testing"

func TestCurrentBackNavigationLayerUsesFixedPriority(t *testing.T) {
	copyMode := CopyModeStore{Active: true}
	cases := []struct {
		name string
		root Root
		want BackNavigationLayer
	}{
		{name: "none", root: Root{Shell: DefaultShell()}, want: BackNavigationNone},
		{name: "interaction", root: Root{Shell: DefaultShell().SetInteractionMode(InteractionModePane)}, want: BackNavigationInteraction},
		{name: "copy before interaction", root: Root{Shell: DefaultShell().SetInteractionMode(InteractionModePane), CopyMode: copyMode}, want: BackNavigationCopy},
		{name: "overlay before copy", root: Root{Shell: DefaultShell().OpenHelp("most-used"), CopyMode: copyMode}, want: BackNavigationOverlay},
		{name: "picker tags before picker overlay", root: Root{Shell: DefaultShell().OpenTerminalPicker().OpenTerminalPickerTags()}, want: BackNavigationTerminalTags},
		{name: "suggestion before prompt", root: Root{Shell: promptShellWithFocusedSuggestion()}, want: BackNavigationPromptSuggestion},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.root.CurrentBackNavigationLayer(); got != tc.want {
				t.Fatalf("back layer mismatch: got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestActiveViewCopyOwnershipSurvivesOverlayPriority(t *testing.T) {
	root := Root{Shell: DefaultShell().OpenHelp("most-used"), CopyMode: CopyModeStore{Active: true}}
	if !root.ActiveViewOwnsCopyInput() {
		t.Fatal("overlay must not erase underlying copy ownership")
	}
	if got := root.CurrentBackNavigationLayer(); got != BackNavigationOverlay {
		t.Fatalf("overlay must still own the first back step, got %q", got)
	}
}

func promptShellWithFocusedSuggestion() ShellStore {
	shell := DefaultShell().OpenPrompt(PromptState{Fields: []PromptFieldState{{Key: "name", SuggestionItems: []string{"shell"}}}})
	return shell.SetPromptSuggestionFocused(true)
}

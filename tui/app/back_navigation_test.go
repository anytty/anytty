package app

import (
	"context"
	"github.com/anytty/anytty/tui/testkit"
	"testing"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
)

func TestBackNavigationEscExitsOneLayerAtATime(t *testing.T) {
	shell := state.DefaultShell().OpenPrompt(state.PromptState{
		Fields: []state.PromptFieldState{{Key: "name", SuggestionItems: []string{"shell"}}},
	}).SetPromptSuggestionFocused(true)
	runtime := NewInteractiveRuntime(state.Root{Shell: shell}, NewFakeTerminalHost(1), NewSyncEffectRunner(), LiveDeps{}, CopyModeDeps{})

	postBackNavigationEsc(t, runtime)
	if overlay := runtime.State().Shell.EnsureDefaults().Overlay; !overlay.Open || overlay.Prompt.SuggestionFocused {
		t.Fatalf("first esc must only leave prompt suggestions, overlay=%#v", overlay)
	}
	postBackNavigationEsc(t, runtime)
	if runtime.State().Shell.EnsureDefaults().Overlay.Open {
		t.Fatalf("second esc must close prompt, overlay=%#v", runtime.State().Shell.Overlay)
	}
}

func TestBackNavigationEscLeavesPickerTagsBeforeClosingPicker(t *testing.T) {
	runtime := NewInteractiveRuntime(
		state.Root{Shell: state.DefaultShell().OpenTerminalPicker().OpenTerminalPickerTags()},
		NewFakeTerminalHost(2), NewSyncEffectRunner(), LiveDeps{}, CopyModeDeps{},
	)

	postBackNavigationEsc(t, runtime)
	overlay := runtime.State().Shell.ReadonlyDefaults().Overlay
	if !overlay.Open || overlay.TerminalPickerView != state.TerminalPickerViewList {
		t.Fatalf("first esc must return from tags to the picker list, overlay=%#v", overlay)
	}
	postBackNavigationEsc(t, runtime)
	if runtime.State().Shell.ReadonlyDefaults().Overlay.Open {
		t.Fatalf("second esc must close the terminal picker, overlay=%#v", runtime.State().Shell.Overlay)
	}
}

func TestBackNavigationPriorityPreservesUnderlyingCopyAndStickyState(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell().SetInteractionMode(state.InteractionModePane).OpenHelp("most-used"),
		CopyMode: state.CopyModeStore{
			Active: true, PaneID: state.DefaultPaneID, ViewID: state.TerminalPaneViewID(state.DefaultPaneID), TerminalID: "term-1", BoundToken: "copy-token",
		},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-1", 4, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
	}
	runtime := NewInteractiveRuntime(root, NewFakeTerminalHost(4), NewSyncEffectRunner(), LiveDeps{}, CopyModeDeps{})

	postBackNavigationEsc(t, runtime)
	if runtime.State().Shell.EnsureDefaults().Overlay.Open || !runtime.State().CopyMode.Active || runtime.State().Shell.InteractionMode != state.InteractionModePane {
		t.Fatalf("first esc must close only overlay: root=%#v", runtime.State())
	}
	postBackNavigationEsc(t, runtime)
	if runtime.State().CopyMode.InputActive() || runtime.State().Shell.InteractionMode != state.InteractionModePane {
		t.Fatalf("second esc must exit only copy mode: root=%#v", runtime.State())
	}
	postBackNavigationEsc(t, runtime)
	if runtime.State().Shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("third esc must exit sticky interaction: root=%#v", runtime.State())
	}
}

func TestBackNavigationLeavesPlainLiveEscForTerminal(t *testing.T) {
	host := NewFakeTerminalHost(1)
	terminal := &testkit.FakeTerminalService{}
	runtime := NewLiveRuntime(
		liveBackNavigationRoot(),
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}); err != nil {
		t.Fatalf("send esc: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain esc: %v", err)
	}
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "\x1b" {
		t.Fatalf("plain live esc must reach terminal, inputs=%#v", terminal.Inputs)
	}
}

func TestBackNavigationModifiedEscDoesNotCloseOverlay(t *testing.T) {
	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyEsc, Alt: true},
		{Kind: input.EventKindKey, Key: input.KeyEsc, Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyEsc, Shift: true},
		{Kind: input.EventKindKey, Key: input.KeyUnknown, RawSeq: "\x1b[27;1:3u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
	} {
		runtime := NewInteractiveRuntime(state.Root{Shell: state.DefaultShell().OpenHelp("most-used")}, NewFakeTerminalHost(1), NewSyncEffectRunner(), LiveDeps{}, CopyModeDeps{})
		if err := runtime.Post(InputMsg{Event: event}); err != nil {
			t.Fatalf("post modified esc %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain modified esc %#v: %v", event, err)
		}
		if !runtime.State().Shell.EnsureDefaults().Overlay.Open {
			t.Fatalf("modified/release esc must not close overlay: event=%#v", event)
		}
	}
}

func TestBackNavigationLiveAltEscWritesLegacyPrefix(t *testing.T) {
	host := NewFakeTerminalHost(1)
	terminal := &testkit.FakeTerminalService{}
	runtime := NewLiveRuntime(liveBackNavigationRoot(), host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal})
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc, Alt: true, KeyboardProtocol: input.KeyboardProtocolKittyCSIU}); err != nil {
		t.Fatalf("send alt esc: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain alt esc: %v", err)
	}
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "\x1b\x1b" {
		t.Fatalf("alt esc must retain legacy ESC prefix, inputs=%#v", terminal.Inputs)
	}
}

func liveBackNavigationRoot() state.Root {
	return state.Root{
		Shell:         state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1"),
		Session:       state.TerminalSessionStore{TerminalID: "term-1", Channel: 4, InputChannels: map[string]uint16{"term-1": 4}, Attached: true},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 4, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)),
	}
}

func postBackNavigationEsc(t *testing.T, runtime *AppRuntime) {
	t.Helper()
	if err := runtime.Post(InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}}); err != nil {
		t.Fatalf("post esc: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain esc: %v", err)
	}
}

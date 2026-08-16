package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/anytty/anytty/tui/testkit"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/shared/terminalmeta"
	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

func TestUIInputReducerOpensTerminalPickerFromCtrlF(t *testing.T) {
	reducer := NewUIInputReducer()
	root, effects := reducer(state.Root{Shell: state.DefaultShell()}, InputMsg{Event: input.InputEvent{
		Kind: input.EventKindKey,
		Key:  input.KeyChar,
		Char: "\x06",
		Ctrl: true,
	}})

	if !root.Shell.Overlay.Open || root.Shell.Overlay.Kind != state.OverlayTerminalPicker {
		t.Fatalf("expected terminal picker overlay, got %#v", root.Shell.Overlay)
	}
	if len(effects) != 2 {
		t.Fatalf("expected handled and pool list effects, got %#v", effects)
	}
	if _, ok := effects[0].(handledEffect); !ok {
		t.Fatalf("expected handled effect, got %#v", effects)
	}
	if effect, ok := effects[1].(FuncEffect); !ok || effect.Run == nil {
		t.Fatalf("expected terminal pool list effect, got %#v", effects[1])
	} else if msg, ok := effect.Run(context.Background()).(TerminalPoolListRequestMsg); !ok || !msg.Refresh {
		t.Fatalf("terminal picker shortcut must use silent pool refresh, got %#v", msg)
	}
	if hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("terminal picker is an overlay and must not arm sticky timeout, got %#v", effects)
	}
}

func TestUIInputReducerUsesConfiguredShortcutsAsRouteTruth(t *testing.T) {
	reducer := NewUIInputReducer()
	shell := state.DefaultShell()
	var result state.WorkbenchCommandResult
	shell, result = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: "logs"})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("create tab fixture: %#v", result)
	}
	secondTabID := shell.Workspace.ActiveTabID
	shell, result = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabSwitch, TargetID: state.DefaultTabID})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("switch tab fixture: %#v", result)
	}
	shell, result = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabSwitch, TargetID: secondTabID})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("switch second tab fixture: %#v", result)
	}
	root := state.Root{
		Shell: shell,
		Config: state.TUIConfigStore{
			Shortcuts: state.TUIShortcutConfig{Scenes: map[string]state.TUIShortcutSceneConfig{
				"global": {Bindings: map[string]state.TUIShortcutBindingConfig{
					"ctrl-1": {Action: "tab.jump.1"},
				}},
			}},
		},
	}

	next, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "p", Ctrl: true}})
	if next.Shell.InteractionMode == state.InteractionModePane || len(effects) != 0 {
		t.Fatalf("removed ctrl-p must not enter pane mode, shell=%#v effects=%#v", next.Shell, effects)
	}

	next, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "1", Ctrl: true, KeyboardProtocol: input.KeyboardProtocolKittyCSIU}})
	if len(effects) != 2 {
		t.Fatalf("custom ctrl-1 should emit handled workbench command effect, effects=%#v", effects)
	}
	fn, ok := effects[1].(FuncEffect)
	if !ok || fn.Run == nil {
		t.Fatalf("custom ctrl-1 should emit workbench command effect, got %#v", effects[1])
	}
	msg, ok := fn.Run(context.Background()).(ShellWorkbenchCommandMsg)
	if !ok || msg.Command.Action != state.WorkbenchCommandTabSwitch || msg.Command.TargetID != state.DefaultTabID {
		t.Fatalf("custom ctrl-1 should request first tab jump, got %#v", msg)
	}
	next, _ = NewShellReducer()(next, msg)
	if next.Shell.Workspace.ActiveTabID != state.DefaultTabID {
		t.Fatalf("custom ctrl-1 should jump to first tab, active=%q", next.Shell.Workspace.ActiveTabID)
	}
}

func TestBackNavigationClosesOverlayWithoutConfiguredShortcut(t *testing.T) {
	reducer := ComposeReducers(NewBackNavigationReducer(CopyModeDeps{}), NewUIInputReducer())
	root := state.Root{
		Shell: state.DefaultShell().OpenHelp("most-used"),
		Config: state.TUIConfigStore{
			Shortcuts: state.TUIShortcutConfig{
				Configured: true,
				Scenes: map[string]state.TUIShortcutSceneConfig{
					"help": {Bindings: map[string]state.TUIShortcutBindingConfig{
						"enter": {Action: "help.close"},
					}},
				},
			},
		},
	}

	footer := render.NewRenderVMBuilder().Build(root).Shell.Footer
	if !footerHasActionKey(footer.ActionTokens, "Esc") {
		t.Fatalf("global back key must be displayed independently from shortcut config, got %#v", footer.ActionTokens)
	}

	next, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}})
	if next.Shell.Overlay.Open {
		t.Fatalf("global back key must close overlay without scene binding, shell=%#v effects=%#v", next.Shell, effects)
	}
	if len(effects) != 0 {
		t.Fatalf("global back key should be consumed once, effects=%#v", effects)
	}

	next, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	if next.Shell.Overlay.Open || len(effects) != 0 {
		t.Fatalf("configured help enter shortcut should still close overlay, shell=%#v effects=%#v", next.Shell, effects)
	}
}

func TestHelpShortcutNavigationUsesConfiguredCatalogActions(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{
		Shell:    state.DefaultShell().OpenHelp("most-used"),
		Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 24},
	}
	itemCount := len(input.ShortcutEntriesForHelp(root.Config.Shortcuts, root.HostCapabilities.KeyboardDisambiguation))
	apply := func(key input.Key) {
		t.Helper()
		next, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: key}})
		if len(effects) == 0 {
			t.Fatalf("help navigation %s must be handled", key)
		}
		root = next
	}

	apply(input.KeyDown)
	if got := root.Shell.EnsureDefaults().Overlay.SelectedIndex; got != 1 {
		t.Fatalf("help down selected=%d want=1", got)
	}
	apply(input.KeyPageDn)
	if got := root.Shell.EnsureDefaults().Overlay.SelectedIndex; got != 9 {
		t.Fatalf("help page-down selected=%d want=9", got)
	}
	apply(input.KeyHome)
	if got := root.Shell.EnsureDefaults().Overlay.SelectedIndex; got != 0 {
		t.Fatalf("help home selected=%d want=0", got)
	}
	apply(input.KeyEnd)
	if got := root.Shell.EnsureDefaults().Overlay.SelectedIndex; got != itemCount-1 {
		t.Fatalf("help end selected=%d want=%d", got, itemCount-1)
	}
	apply(input.KeyUp)
	if got := root.Shell.EnsureDefaults().Overlay.SelectedIndex; got != itemCount-2 {
		t.Fatalf("help up selected=%d want=%d", got, itemCount-2)
	}
}

func TestUIInputReducerTerminalPoolEmptyShortcutSceneRemovesDefaultActions(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{
		Shell: state.DefaultShell().OpenTerminalPool(),
		TerminalPool: state.TerminalPoolStore{
			Status: state.TerminalPoolReady,
			Items:  []state.TerminalPoolItem{{TerminalID: "term-1", Title: "shell", State: "running"}},
		},
		Config: state.TUIConfigStore{
			Shortcuts: state.TUIShortcutConfig{
				Configured: true,
				Scenes: map[string]state.TUIShortcutSceneConfig{
					"terminal_pool": {Bindings: map[string]state.TUIShortcutBindingConfig{}},
				},
			},
		},
	}

	footer := render.NewRenderVMBuilder().Build(root).Shell.Footer
	if len(footer.ActionTokens) != 1 || !footerHasActionKey(footer.ActionTokens, "Esc") {
		t.Fatalf("empty terminal_pool shortcut scene must only display global back, got %#v", footer.ActionTokens)
	}

	next, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	if !next.Shell.Overlay.Open || next.Shell.Overlay.Kind != state.OverlayTerminalPool {
		t.Fatalf("removed terminal_pool enter shortcut must not attach or close overlay, shell=%#v effects=%#v", next.Shell, effects)
	}
	if effectEmitsTerminalPoolAttach(effects) {
		t.Fatalf("removed terminal_pool enter shortcut must not emit attach effect, effects=%#v", effects)
	}
}

func footerHasActionKey(actions []render.FooterActionVM, key string) bool {
	for _, action := range actions {
		if action.Key == key {
			return true
		}
	}
	return false
}

func effectEmitsTerminalPoolAttach(effects []Effect) bool {
	for _, effect := range effects {
		fn, ok := effect.(FuncEffect)
		if !ok || fn.Run == nil {
			continue
		}
		if _, ok := fn.Run(context.Background()).(TerminalPoolAttachRequestMsg); ok {
			return true
		}
	}
	return false
}

func TestUIInputReducerStickyInteractionModeTimeoutAndRearm(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{Shell: state.DefaultShell()}

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x10", Ctrl: true}})
	if root.Shell.InteractionMode != state.InteractionModePane || !hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("ctrl-p should enter pane mode and arm timeout, shell=%#v effects=%#v", root.Shell, effects)
	}
	paneSeq := root.Shell.InteractionModeSeq

	root, _ = reducer(root, ShellInteractionModeTimeoutMsg{Mode: state.InteractionModePane, Seq: paneSeq})
	if root.Shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("matching timeout should exit sticky mode, shell=%#v", root.Shell)
	}

	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x12", Ctrl: true}})
	if root.Shell.InteractionMode != state.InteractionModeResize || !hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("ctrl-r should enter resize mode and arm timeout, shell=%#v effects=%#v", root.Shell, effects)
	}
	oldSeq := root.Shell.InteractionModeSeq
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyRight}})
	if root.Shell.InteractionMode != state.InteractionModeResize || root.Shell.InteractionModeSeq <= oldSeq || !hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("resize action should rearm sticky timeout, shell=%#v effects=%#v oldSeq=%d", root.Shell, effects, oldSeq)
	}

	root, _ = reducer(root, ShellInteractionModeTimeoutMsg{Mode: state.InteractionModeResize, Seq: oldSeq})
	if root.Shell.InteractionMode != state.InteractionModeResize {
		t.Fatalf("stale timeout must not exit renewed mode, shell=%#v", root.Shell)
	}
	root, _ = reducer(root, ShellInteractionModeTimeoutMsg{Mode: state.InteractionModeResize, Seq: root.Shell.InteractionModeSeq})
	if root.Shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("latest timeout should exit renewed mode, shell=%#v", root.Shell)
	}
}

func TestUIInputReducerPrefixModeCommandExitPolicy(t *testing.T) {
	reducer := NewUIInputReducer()
	shell := state.DefaultShell()
	shell, _ = shell.ApplyPaneCommand(state.PaneCommand{
		Action:         state.PaneCommandSplit,
		Target:         state.PaneCommandTarget{PaneID: state.DefaultPaneID},
		SplitDirection: state.SplitDirectionVertical,
		NewPane:        state.PaneState{ID: "pane-2", Title: "pane-2", Kind: state.PaneEmpty},
	})
	root := state.Root{Shell: shell.SetInteractionMode(state.InteractionModePane)}

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "z"}})
	if root.Shell.InteractionMode != state.InteractionModeNormal || hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("ordinary pane command should exit prefix mode, shell=%#v effects=%#v", root.Shell, effects)
	}

	root.Shell = root.Shell.SetInteractionMode(state.InteractionModePane)
	oldSeq := root.Shell.InteractionModeSeq
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "n"}})
	if root.Shell.InteractionMode != state.InteractionModePane || root.Shell.InteractionModeSeq <= oldSeq || !hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("pane focus command should keep prefix mode for repeated navigation, shell=%#v effects=%#v oldSeq=%d", root.Shell, effects, oldSeq)
	}

	root.Shell = root.Shell.SetInteractionMode(state.InteractionModeTab)
	oldSeq = root.Shell.InteractionModeSeq
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "n"}})
	if root.Shell.InteractionMode != state.InteractionModeTab || root.Shell.InteractionModeSeq <= oldSeq || !hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("tab next should keep prefix mode for repeated navigation, shell=%#v effects=%#v oldSeq=%d", root.Shell, effects, oldSeq)
	}
}

func TestShortcutPassthroughLockSendsRootShortcutToTerminal(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	core := &testkit.FakeCoreClient{}
	reducer := ComposeReducers(
		NewUIInputReducer(),
		NewCopyModeReducer(CopyModeDeps{Core: core, Rows: 20}),
		NewTerminalInputRouterReducer(LiveDeps{Terminal: terminal}),
	)
	root := shortcutPassthroughInputRoot()

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x07", Ctrl: true}})
	if root.Shell.InteractionMode != state.InteractionModeGlobal || len(effects) == 0 {
		t.Fatalf("ctrl-g should enter global mode before lock toggle, shell=%#v effects=%#v", root.Shell, effects)
	}
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "l"}})
	if !root.Shell.ShortcutPassthroughLocked || root.Shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("global l should enable shortcut passthrough lock and exit mode, shell=%#v", root.Shell)
	}

	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x14", Ctrl: true}})
	if root.Shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("locked ctrl-t must not enter tab mode, shell=%#v", root.Shell)
	}
	runTerminalInputEffects(t, effects)
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "\x14" {
		t.Fatalf("locked ctrl-t should be sent to terminal, inputs=%#v effects=%#v", terminal.Inputs, effects)
	}

	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true}})
	if root.CopyMode.InputActive() || root.History.Pending != nil || len(core.LatestRequests) != 0 {
		t.Fatalf("locked ctrl-v must not enter copy mode, copy=%#v history=%#v latest=%#v", root.CopyMode, root.History, core.LatestRequests)
	}
	runTerminalInputEffects(t, effects)
	if len(terminal.Inputs) != 2 || string(terminal.Inputs[1].Bytes) != "\x16" {
		t.Fatalf("locked ctrl-v should be sent to terminal, inputs=%#v effects=%#v", terminal.Inputs, effects)
	}

	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x07", Ctrl: true}})
	if root.Shell.InteractionMode != state.InteractionModeGlobal {
		t.Fatalf("global entry must stay available as unlock control plane, shell=%#v effects=%#v", root.Shell, effects)
	}
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x12", Ctrl: true}})
	if root.Shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("locked ctrl-r inside global mode should exit mode before terminal send, shell=%#v", root.Shell)
	}
	runTerminalInputEffects(t, effects)
	if len(terminal.Inputs) != 3 || string(terminal.Inputs[2].Bytes) != "\x12" {
		t.Fatalf("locked ctrl-r inside global mode should be sent to terminal, inputs=%#v effects=%#v", terminal.Inputs, effects)
	}
}

func TestShortcutPassthroughLockSendsRootShortcutWhileCopyModeActive(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	reducer := ComposeReducers(
		NewUIInputReducer(),
		NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20}),
		NewTerminalInputRouterReducer(LiveDeps{Terminal: terminal}),
	)
	root := shortcutPassthroughInputRoot()
	root.Shell = root.Shell.ToggleShortcutPassthroughLock()
	root.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundCols:  80,
	}

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true}})
	if !root.CopyMode.Active || root.History.Pending != nil {
		t.Fatalf("locked ctrl-v should not mutate active copy mode, copy=%#v history=%#v", root.CopyMode, root.History)
	}
	runTerminalInputEffects(t, effects)
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "\x16" {
		t.Fatalf("locked ctrl-v should passthrough while copy is active, inputs=%#v effects=%#v", terminal.Inputs, effects)
	}
}

func TestDoubleTapStickyPrefixSendsSecondPrefixToTerminal(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	reducer := ComposeReducers(NewUIInputReducer(), NewTerminalInputRouterReducer(LiveDeps{Terminal: terminal}))
	root := shortcutPassthroughInputRoot()
	ctrlW := input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x17", Ctrl: true}

	root, effects := reducer(root, InputMsg{Event: ctrlW})
	runTerminalInputEffects(t, effects)
	if root.Shell.InteractionMode != state.InteractionModeWorkspace || len(terminal.Inputs) != 0 || !hasShortcutPassthroughTimeoutEffect(effects) {
		t.Fatalf("first ctrl-w should enter workspace mode only, shell=%#v inputs=%#v", root.Shell, terminal.Inputs)
	}

	root, effects = reducer(root, InputMsg{Event: ctrlW})
	if root.Shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("second ctrl-w should exit workspace mode before terminal send, shell=%#v", root.Shell)
	}
	runTerminalInputEffects(t, effects)
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "\x17" {
		t.Fatalf("second ctrl-w should be sent to terminal, inputs=%#v effects=%#v", terminal.Inputs, effects)
	}
}

func TestDoubleTapStickyPrefixAfterWindowTimeoutDoesNotPassthrough(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	reducer := ComposeReducers(NewUIInputReducer(), NewTerminalInputRouterReducer(LiveDeps{Terminal: terminal}))
	root := shortcutPassthroughInputRoot()
	ctrlW := input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x17", Ctrl: true}

	root, _ = reducer(root, InputMsg{Event: ctrlW})
	kind := shortcutPassthroughKindForMode(state.InteractionModeWorkspace)
	seq, ok := root.Shell.ShortcutPassthroughWindow(kind)
	if !ok {
		t.Fatalf("first ctrl-w should arm passthrough window, shell=%#v", root.Shell)
	}
	root, _ = reducer(root, ShellShortcutPassthroughTimeoutMsg{Kind: kind, Seq: seq})
	root, effects := reducer(root, InputMsg{Event: ctrlW})
	runTerminalInputEffects(t, effects)
	if root.Shell.InteractionMode != state.InteractionModeWorkspace || len(terminal.Inputs) != 0 {
		t.Fatalf("expired double-tap window must not send ctrl-w to terminal, shell=%#v inputs=%#v effects=%#v", root.Shell, terminal.Inputs, effects)
	}
}

func TestDoubleTapCopyModeEntrySendsSecondCtrlVToTerminal(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	core := &testkit.FakeCoreClient{}
	reducer := ComposeReducers(
		NewUIInputReducer(),
		NewCopyModeReducer(CopyModeDeps{Core: core, Rows: 20}),
		NewTerminalInputRouterReducer(LiveDeps{Terminal: terminal}),
	)
	root := shortcutPassthroughInputRoot()
	ctrlV := input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true}

	root, _ = reducer(root, InputMsg{Event: ctrlV})
	if !root.CopyMode.Entering || root.History.Pending == nil || !root.Shell.ShortcutPassthroughWindowMatches(shortcutPassthroughKindCopy) {
		t.Fatalf("first ctrl-v should enter pending copy mode, copy=%#v history=%#v", root.CopyMode, root.History)
	}

	root, effects := reducer(root, InputMsg{Event: ctrlV})
	if root.CopyMode.InputActive() || root.History.Pending != nil {
		t.Fatalf("second ctrl-v should exit pending copy before passthrough, copy=%#v history=%#v", root.CopyMode, root.History)
	}
	runTerminalInputEffects(t, effects)
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "\x16" {
		t.Fatalf("second ctrl-v should be sent to terminal, inputs=%#v effects=%#v", terminal.Inputs, effects)
	}
}

func TestDoubleTapCopyModeEntryAfterWindowTimeoutDoesNotPassthrough(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	core := &testkit.FakeCoreClient{}
	reducer := ComposeReducers(
		NewUIInputReducer(),
		NewCopyModeReducer(CopyModeDeps{Core: core, Rows: 20}),
		NewTerminalInputRouterReducer(LiveDeps{Terminal: terminal}),
	)
	root := shortcutPassthroughInputRoot()
	ctrlV := input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true}

	root, _ = reducer(root, InputMsg{Event: ctrlV})
	seq, ok := root.Shell.ShortcutPassthroughWindow(shortcutPassthroughKindCopy)
	if !ok {
		t.Fatalf("first ctrl-v should arm passthrough window, shell=%#v", root.Shell)
	}
	root, _ = reducer(root, ShellShortcutPassthroughTimeoutMsg{Kind: shortcutPassthroughKindCopy, Seq: seq})
	root, effects := reducer(root, InputMsg{Event: ctrlV})
	runTerminalInputEffects(t, effects)
	if !root.CopyMode.InputActive() || root.History.Pending == nil || len(terminal.Inputs) != 0 {
		t.Fatalf("expired copy double-tap window must stay in copy mode and not send terminal input, copy=%#v history=%#v inputs=%#v effects=%#v", root.CopyMode, root.History, terminal.Inputs, effects)
	}
}

func TestDoubleTapStickyPrefixKeepsTerminalInputOrder(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(8)
	runtime := NewAppRuntime(
		shortcutPassthroughInputRoot(),
		ComposeReducers(NewUIInputReducer(), NewTerminalInputRouterReducer(LiveDeps{Terminal: terminal})),
		nil,
		host,
		NewSyncEffectRunner(),
	)
	ctrlW := input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x17", Ctrl: true}
	if err := host.SendInput(ctrlW); err != nil {
		t.Fatalf("send first ctrl-w: %v", err)
	}
	if err := host.SendInput(ctrlW); err != nil {
		t.Fatalf("send second ctrl-w: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x", RawSeq: "x"}); err != nil {
		t.Fatalf("send x: %v", err)
	}

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(terminal.Inputs) != 2 || string(terminal.Inputs[0].Bytes) != "\x17" || string(terminal.Inputs[1].Bytes) != "x" {
		t.Fatalf("double-tap prefix must keep PTY input order, inputs=%#v", terminal.Inputs)
	}
}

func TestUIInputReducerStickyTimeoutDoesNotCloseOverlayOrCopyMode(t *testing.T) {
	reducer := NewUIInputReducer()
	shell := state.DefaultShell().SetInteractionMode(state.InteractionModeGlobal).OpenTerminalPicker()
	root := state.Root{
		Shell:    shell,
		CopyMode: state.CopyModeStore{Active: true, TerminalID: "term-1"},
	}
	seq := root.Shell.InteractionModeSeq

	root, _ = reducer(root, ShellInteractionModeTimeoutMsg{Mode: state.InteractionModeGlobal, Seq: seq})
	if root.Shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("timeout should clear only sticky interaction mode, shell=%#v", root.Shell)
	}
	if !root.Shell.Overlay.Open || root.Shell.Overlay.Kind != state.OverlayTerminalPicker {
		t.Fatalf("timeout must not close overlay, overlay=%#v", root.Shell.Overlay)
	}
	if !root.CopyMode.Active || root.CopyMode.TerminalID != "term-1" {
		t.Fatalf("timeout must not mutate copy mode, copy=%#v", root.CopyMode)
	}
}

func shortcutPassthroughInputRoot() state.Root {
	root := state.Root{
		Shell:    state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1"),
		Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 24},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(
		state.DefaultPaneID,
		"term-1",
		7,
		80,
		24,
		state.TerminalResizeRoleOwner,
		"surface-1",
		state.TerminalPaneViewID(state.DefaultPaneID),
		true,
	))
	return root
}

func runTerminalInputEffects(t *testing.T, effects []Effect) {
	t.Helper()
	for _, effect := range effects {
		fn, ok := effect.(FuncEffect)
		if !ok || fn.Run == nil || !fn.ForceSyncInTests || !strings.HasPrefix(fn.SerialKey, "terminal.input:") {
			continue
		}
		_ = fn.Run(context.Background())
	}
}

func hasStickyInteractionModeTimeoutEffect(effects []Effect) bool {
	for _, effect := range effects {
		if fn, ok := effect.(FuncEffect); ok && fn.Token == stickyInteractionModeTimeoutToken {
			return true
		}
	}
	return false
}

func hasShortcutPassthroughTimeoutEffect(effects []Effect) bool {
	for _, effect := range effects {
		if fn, ok := effect.(FuncEffect); ok && fn.Token == shellShortcutPassthroughTimeoutToken {
			return true
		}
	}
	return false
}

func runFirstNonStickyTimeoutEffect(t *testing.T, effects []Effect) Msg {
	t.Helper()
	for _, effect := range effects {
		fn, ok := effect.(FuncEffect)
		if !ok || fn.Run == nil || fn.Token == stickyInteractionModeTimeoutToken || fn.Token == shellShortcutPassthroughTimeoutToken {
			continue
		}
		return fn.Run(context.Background())
	}
	t.Fatalf("expected non-timeout function effect, got %#v", effects)
	return nil
}

func TestUIInputReducerPaneModeTuiv2OwnerPickerAndRestart(t *testing.T) {
	reducer := NewUIInputReducer()
	shell := state.DefaultShell()
	shell.ActivePaneID = state.DefaultPaneID
	root := state.Root{Shell: shell}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 3, 80, 24, state.TerminalResizeRoleFollower, "surface-1", "", false))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-1", 4, 80, 24, state.TerminalResizeRoleOwner, "surface-1", "", true))

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x10", Ctrl: true}})
	if root.Shell.InteractionMode != state.InteractionModePane || !hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("ctrl-p should enter pane mode, mode=%#v effects=%#v", root.Shell.InteractionMode, effects)
	}
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "a"}})
	if pending, ok := root.TerminalViews.PaneBinding(state.DefaultPaneID); !ok || !pending.OwnerAcquirePending || pending.ResizeRole != state.TerminalResizeRoleFollower {
		t.Fatalf("pane owner shortcut should wait for daemon confirmation, pending=%#v ok=%v", pending, ok)
	}
	ownerMsg, ok := runFirstNonStickyTimeoutEffect(t, effects).(LiveResizeMsg)
	if !ok || ownerMsg.TerminalID != "term-1" || ownerMsg.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) || !ownerMsg.TakeOwnership {
		t.Fatalf("pane owner shortcut should request authoritative resize control, got %#v", ownerMsg)
	}
	if root.Shell.InteractionMode != state.InteractionModeNormal || hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("pane owner shortcut should exit prefix mode, shell=%#v effects=%#v", root.Shell, effects)
	}

	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x10", Ctrl: true}})
	if root.Shell.InteractionMode != state.InteractionModePane || !hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("ctrl-p should re-enter pane mode, mode=%#v effects=%#v", root.Shell.InteractionMode, effects)
	}
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "r"}})
	if root.Shell.Overlay.Open || root.Shell.InteractionMode != state.InteractionModeNormal || hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("pane reconnect should target the current terminal directly, overlay=%#v effects=%#v", root.Shell.Overlay, effects)
	}
	reconnect, ok := runFirstNonStickyTimeoutEffect(t, effects).(TerminalPoolReconnectRequestMsg)
	if !ok || reconnect.TerminalID != "term-1" || reconnect.TargetPaneID != state.DefaultPaneID || !reconnect.LocalError {
		t.Fatalf("pane reconnect should emit endpoint-aware reconnect request, got %#v", effects)
	}

	root.Shell = root.Shell.SetInteractionMode(state.InteractionModePane)
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "R"}})
	if root.Shell.InteractionMode != state.InteractionModeNormal || hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("pane restart should emit handled and restart effects, got %#v", effects)
	}
	restartMsg, ok := runFirstNonStickyTimeoutEffect(t, effects).(TerminalPoolRestartRequestMsg)
	if !ok || restartMsg.TerminalID != "term-1" {
		t.Fatalf("pane restart should restart the active terminal directly, got %#v", restartMsg)
	}
	if root.Shell.InteractionMode != state.InteractionModeNormal || hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("pane restart shortcut should exit prefix mode, shell=%#v effects=%#v", root.Shell, effects)
	}
}

func TestUIInputReducerFloatingModeTuiv2PickerAndOwner(t *testing.T) {
	reducer := NewUIInputReducer()
	shell := state.DefaultShell()
	shell, _ = shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandCreate, TargetID: "float-1", Title: "float", Pane: state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-1"}})
	root := state.Root{Shell: shell}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 3, 80, 24, state.TerminalResizeRoleOwner, "surface-1", "", true))
	root.TerminalViews = root.TerminalViews.BindFloating(state.NewFloatingTerminalView("float-1", "", "term-1", 5, 80, 24, state.TerminalResizeRoleFollower, "surface-1", "", false))

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x0f", Ctrl: true}})
	if root.Shell.InteractionMode != state.InteractionModeFloating || !hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("ctrl-o should enter floating mode, mode=%#v effects=%#v", root.Shell.InteractionMode, effects)
	}
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "f"}})
	if root.Shell.InteractionMode != state.InteractionModeNormal || hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("floating picker shortcut should emit handled and picker effects, got %#v", effects)
	}
	if !root.Shell.Overlay.Open || root.Shell.Overlay.Kind != state.OverlayTerminalPicker {
		t.Fatalf("floating picker shortcut should open picker through canonical handler, got %#v", root.Shell.Overlay)
	}
	if msg, ok := runFirstNonStickyTimeoutEffect(t, effects).(TerminalPoolListRequestMsg); !ok || !msg.Refresh {
		t.Fatalf("floating picker shortcut should refresh picker inventory, got %#v", msg)
	}

	root.Shell = root.Shell.CloseOverlay().SetInteractionMode(state.InteractionModeFloating)
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "a"}})
	if pending, ok := root.TerminalViews.FloatingBinding("float-1"); !ok || !pending.OwnerAcquirePending || pending.ResizeRole != state.TerminalResizeRoleFollower {
		t.Fatalf("floating owner shortcut should wait for daemon confirmation, pending=%#v ok=%v", pending, ok)
	}
	msg, ok := runFirstNonStickyTimeoutEffect(t, effects).(LiveResizeMsg)
	if !ok || msg.TerminalID != "term-1" || msg.ViewID != state.TerminalFloatingViewID("float-1") || !msg.TakeOwnership {
		t.Fatalf("floating owner shortcut should request authoritative resize control, got %#v", msg)
	}
	if root.Shell.InteractionMode != state.InteractionModeNormal || hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("floating owner shortcut should exit prefix mode, shell=%#v effects=%#v", root.Shell, effects)
	}
}

func TestUIInputReducerEmptyPaneCTAKeyboardSelectionAndEnter(t *testing.T) {
	reducer := NewUIInputReducer()
	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0] = state.PaneState{ID: state.DefaultPaneID, Title: "slot", Kind: state.PaneEmpty, Active: true}
	root := state.Root{Shell: shell}

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyDown}})
	if root.Shell.EmptyPaneCTA.SelectedIndex != 1 {
		t.Fatalf("down should select create CTA, got %#v effects=%#v", root.Shell.EmptyPaneCTA, effects)
	}
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyDown}})
	if root.Shell.EmptyPaneCTA.SelectedIndex != 2 {
		t.Fatalf("second down should select manager CTA, got %#v effects=%#v", root.Shell.EmptyPaneCTA, effects)
	}
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	if root.Shell.EmptyPaneCTA.SelectedIndex != 2 || len(effects) != 2 {
		t.Fatalf("enter should keep selection and emit handled action effects, root=%#v effects=%#v", root.Shell.EmptyPaneCTA, effects)
	}
	effect, ok := effects[1].(FuncEffect)
	if !ok || effect.Run == nil {
		t.Fatalf("enter should emit content action effect, got %#v", effects)
	}
	msg, ok := effect.Run(context.Background()).(ShellShortcutActionMsg)
	if !ok || msg.Invocation.ID != actiondomain.ActionEmptyManager || msg.Surface == nil || msg.Surface.PaneID != state.DefaultPaneID {
		t.Fatalf("enter should execute selected manager CTA, got %#v", msg)
	}
}

func TestUIInputReducerFloatingEmptyPaneCTAKeyboardTargetsFloatingPanel(t *testing.T) {
	reducer := NewUIInputReducer()
	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0] = state.PaneState{ID: state.DefaultPaneID, Title: "tiled", Kind: state.PaneEmpty, Active: true}
	shell, _ = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float", Kind: state.PaneEmpty},
	})
	root := state.Root{Shell: shell}

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyDown}})
	if root.Shell.EmptyPaneCTA.SelectedIndex != 1 || len(effects) != 1 {
		t.Fatalf("down should select floating create CTA, shell=%#v effects=%#v", root.Shell, effects)
	}
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	if len(effects) != 2 {
		t.Fatalf("enter should emit handled floating action effects, effects=%#v", effects)
	}
	effect, ok := effects[1].(FuncEffect)
	if !ok || effect.Run == nil {
		t.Fatalf("enter should emit content action effect, got %#v", effects)
	}
	msg, ok := effect.Run(context.Background()).(ShellShortcutActionMsg)
	if !ok || msg.Invocation.ID != actiondomain.ActionEmptyCreate || msg.Surface == nil || msg.Surface.PaneID != "floating-pane-1" || !msg.Surface.Floating {
		t.Fatalf("enter should execute selected floating CTA, got %#v", msg)
	}
}

func TestUIInputReducerDisconnectedPaneCTAKeyboardSelectionAndEnter(t *testing.T) {
	reducer := NewUIInputReducer()
	ref := state.NewTerminalRef("west", "remote")
	root := state.Root{
		Shell:   state.DefaultShell(),
		Surface: state.TerminalSurfaceStore{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, State: state.TerminalLiveError, Err: "remote-daemon: daemon socket closed"},
	}
	root.Shell.Workspace.Tabs[0].Panes[0] = state.PaneState{ID: state.DefaultPaneID, Title: "unconnected", Kind: state.PaneEmpty, Active: true}
	binding := state.NewEndpointPaneTerminalView(ref.EndpointID, state.DefaultPaneID, ref.TerminalID, 0, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)
	binding.LastError = "remote-daemon: daemon socket closed"
	root.TerminalViews = root.TerminalViews.BindPane(binding)

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	if root.Shell.ExitedPaneCTA.SelectedIndex != 0 || len(effects) != 2 {
		t.Fatalf("enter should execute default disconnected reconnect action, shell=%#v effects=%#v", root.Shell.ExitedPaneCTA, effects)
	}
	msg, ok := effects[1].(FuncEffect).Run(context.Background()).(ShellShortcutActionMsg)
	if !ok || msg.Invocation.ID != actiondomain.ActionDisconnectedReconnect || msg.Surface == nil || msg.Surface.PaneID != state.DefaultPaneID {
		t.Fatalf("enter should execute disconnected reconnect CTA, got %#v", msg)
	}

	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyDown}})
	if root.Shell.ExitedPaneCTA.SelectedIndex != 1 || len(effects) != 1 {
		t.Fatalf("down should select disconnected disconnect CTA, shell=%#v effects=%#v", root.Shell.ExitedPaneCTA, effects)
	}
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	msg, ok = effects[1].(FuncEffect).Run(context.Background()).(ShellShortcutActionMsg)
	if !ok || msg.Invocation.ID != actiondomain.ActionDisconnectedDisconnect || msg.Surface == nil || msg.Surface.PaneID != state.DefaultPaneID {
		t.Fatalf("enter should execute disconnected disconnect CTA, got %#v", msg)
	}
}

func TestUIInputReducerExitedPaneCTAKeyboardSelectionAndEnter(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{
		Shell:   state.DefaultShell(),
		Surface: state.TerminalSurfaceStore{TerminalID: "term-exited", State: state.TerminalLiveExited, ExitCode: 23},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-exited", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
	}

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyDown}})
	if root.Shell.ExitedPaneCTA.SelectedIndex != 1 || len(effects) != 1 {
		t.Fatalf("down should select picker CTA, shell=%#v effects=%#v", root.Shell.ExitedPaneCTA, effects)
	}
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	if root.Shell.ExitedPaneCTA.SelectedIndex != 1 || len(effects) != 2 {
		t.Fatalf("enter should keep exited selection and emit action effects, root=%#v effects=%#v", root.Shell.ExitedPaneCTA, effects)
	}
	effect, ok := effects[1].(FuncEffect)
	if !ok || effect.Run == nil {
		t.Fatalf("enter should emit content action effect, got %#v", effects)
	}
	msg, ok := effect.Run(context.Background()).(ShellShortcutActionMsg)
	if !ok || msg.Invocation.ID != actiondomain.ActionExitedReconnect || msg.Surface == nil || msg.Surface.PaneID != state.DefaultPaneID {
		t.Fatalf("enter should execute selected exited picker CTA, got %#v", msg)
	}
}

func TestUIInputReducerExitedPaneCTAKeyboardRestartDefault(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{
		Shell:   state.DefaultShell(),
		Surface: state.TerminalSurfaceStore{TerminalID: "term-exited", State: state.TerminalLiveExited, ExitCode: 23},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-exited", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
	}

	_, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	if len(effects) != 2 {
		t.Fatalf("enter should emit handled restart action effects, got %#v", effects)
	}
	msg, ok := effects[1].(FuncEffect).Run(context.Background()).(ShellShortcutActionMsg)
	if !ok || msg.Invocation.ID != actiondomain.ActionExitedRestart || msg.Surface == nil || msg.Surface.PaneID != state.DefaultPaneID {
		t.Fatalf("enter should execute default restart CTA, got %#v", msg)
	}
}

func TestUIInputReducerExitedPaneDoesNotInventRShortcut(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{
		Shell:   state.DefaultShell(),
		Surface: state.TerminalSurfaceStore{TerminalID: "term-exited", State: state.TerminalLiveExited, ExitCode: 23},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-exited", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
	}

	for _, char := range []string{"R", "r"} {
		_, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: char}})
		if len(effects) != 0 {
			t.Fatalf("%q must not bypass the configurable shortcut catalog, got %#v", char, effects)
		}
	}
}

func TestUIInputReducerExitedPaneSessionLifecycleRestarts(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{
		Shell: state.DefaultShell(),
		Session: state.TerminalSessionStore{}.
			AttachWithResizeOwner("term-session-exited", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID)).
			MarkExitedWithMetadata("term-session-exited", 130, "exited", time.Date(2026, 7, 5, 1, 1, 34, 0, time.UTC), []string{"/bin/zsh"}),
		Surface: state.TerminalSurfaceStore{TerminalID: "term-session-exited", State: state.TerminalLiveAttached, Lines: []string{"terminal exited: term-session-exited code:130 exited"}},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-session-exited", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
	}

	for _, event := range []input.InputEvent{{Kind: input.EventKindKey, Key: input.KeyEnter}} {
		_, effects := reducer(root, InputMsg{Event: event})
		if len(effects) != 2 {
			t.Fatalf("session exited CTA input should emit restart action effects event=%#v effects=%#v", event, effects)
		}
		msg, ok := effects[1].(FuncEffect).Run(context.Background()).(ShellShortcutActionMsg)
		if !ok || msg.Invocation.ID != actiondomain.ActionExitedRestart || msg.Surface == nil || msg.Surface.PaneID != state.DefaultPaneID {
			t.Fatalf("session exited CTA input should execute restart action event=%#v msg=%#v", event, msg)
		}
	}
}

func TestUIInputReducerExitedCacheRestartQueriesCoreLifecycle(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{
		Shell:   state.DefaultShell(),
		Surface: state.TerminalSurfaceStore{TerminalID: "term-live", State: state.TerminalLiveExited, ExitCode: 23},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-live", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
	}

	_, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	if len(effects) != 2 {
		t.Fatalf("exited CTA enter should be handled and schedule core lifecycle query, got %#v", effects)
	}
	msg, ok := effects[1].(FuncEffect).Run(context.Background()).(ShellShortcutActionMsg)
	if !ok || msg.Invocation.ID != actiondomain.ActionExitedRestart || msg.Surface == nil || msg.Surface.PaneID != state.DefaultPaneID {
		t.Fatalf("enter should still route through exited restart action, got %#v", msg)
	}
	next, effects := NewShellReducer()(root, msg)
	if len(effects) != 2 {
		t.Fatalf("restart action should schedule a restart-if-exited core query, root=%#v effects=%#v", next, effects)
	}
	query, ok := effects[1].(FuncEffect).Run(context.Background()).(TerminalPoolRestartIfExitedRequestMsg)
	if !ok || query.TerminalID != "term-live" {
		t.Fatalf("restart action must query core lifecycle before restart, got %#v", query)
	}
}

func TestUIInputReducerTerminalPickerDeleteKeysTrimQuery(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{Shell: state.DefaultShell().OpenTerminalPicker().SetTerminalPickerQuery("日志")}

	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyBackspace}})
	if got := root.Shell.EnsureDefaults().Overlay.Query; got != "日" {
		t.Fatalf("backspace should trim picker query, got %q", got)
	}
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyDelete}})
	if got := root.Shell.EnsureDefaults().Overlay.Query; got != "" {
		t.Fatalf("delete should trim picker query, got %q", got)
	}
	root = state.Root{Shell: root.Shell.SetTerminalPickerQuery("x")}
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x7f"}})
	if got := root.Shell.EnsureDefaults().Overlay.Query; got != "" {
		t.Fatalf("DEL char should trim picker query, got %q", got)
	}
}

func TestUIInputReducerTerminalPoolSearchKeysEditQuery(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{Shell: state.DefaultShell().OpenTerminalPool().SetTerminalPoolQuery("日志")}

	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"}})
	if got := root.Shell.EnsureDefaults().Overlay.Query; got != "日志x" {
		t.Fatalf("plain char should append to terminal pool query, got %q", got)
	}
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyBackspace}})
	if got := root.Shell.EnsureDefaults().Overlay.Query; got != "日志" {
		t.Fatalf("backspace should trim terminal pool query, got %q", got)
	}
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyDelete}})
	if got := root.Shell.EnsureDefaults().Overlay.Query; got != "日" {
		t.Fatalf("delete should trim terminal pool query, got %q", got)
	}
	root = state.Root{Shell: root.Shell.SetTerminalPoolQuery("x")}
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x7f"}})
	if got := root.Shell.EnsureDefaults().Overlay.Query; got != "" {
		t.Fatalf("DEL char should trim terminal pool query, got %q", got)
	}
}

func TestUIInputReducerWorkbenchTreeDeleteKeysTrimQuery(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{Shell: state.DefaultShell().OpenWorkbenchTree().SetWorkbenchTreeQuery("日志")}

	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyBackspace}})
	if got := root.Shell.EnsureDefaults().Overlay.Query; got != "日" {
		t.Fatalf("backspace should trim workbench query, got %q", got)
	}
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyDelete}})
	if got := root.Shell.EnsureDefaults().Overlay.Query; got != "" {
		t.Fatalf("delete should trim workbench query, got %q", got)
	}
	root = state.Root{Shell: root.Shell.SetWorkbenchTreeQuery("x")}
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x7f"}})
	if got := root.Shell.EnsureDefaults().Overlay.Query; got != "" {
		t.Fatalf("DEL char should trim workbench query, got %q", got)
	}
}

func TestUIInputReducerOpensClipboardHistoryFromCopyModeH(t *testing.T) {
	reducer := ComposeReducers(NewClipboardActionReducer(ClipboardActionDeps{}), NewShellReducer(), NewUIInputReducer(), NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}, Clipboard: &testkit.FakeClipboardService{}, Rows: 20}))
	root := state.Root{
		CopyMode: state.CopyModeStore{Active: true},
		Clipboard: state.ClipboardStore{
			Entries: []state.ClipboardEntry{{ID: "clip:1", Title: "alpha", Text: "alpha", Preview: "alpha"}},
		},
	}

	next, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "H"}})
	if !next.Shell.Overlay.Open || next.Shell.Overlay.Kind != state.OverlayClipboardHistory {
		t.Fatalf("expected clipboard history overlay, got %#v", next.Shell.Overlay)
	}
	if len(effects) != 1 {
		t.Fatalf("expected storage load effect, got %#v", effects)
	}
	if _, ok := effects[0].(FuncEffect); !ok {
		t.Fatalf("expected storage load effect, got %#v", effects)
	}
}

func TestUIInputReducerOpensClipboardHistoryWhileCopyModeEntering(t *testing.T) {
	reducer := ComposeReducers(NewClipboardActionReducer(ClipboardActionDeps{}), NewShellReducer(), NewUIInputReducer(), NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}, Clipboard: &testkit.FakeClipboardService{}, Rows: 20}))
	root := state.Root{CopyMode: state.CopyModeStore{Entering: true, RequestID: 1}}

	next, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "H"}})
	if !next.Shell.Overlay.Open || next.Shell.Overlay.Kind != state.OverlayClipboardHistory || !next.CopyMode.Entering {
		t.Fatalf("expected clipboard history overlay without canceling entering copy mode, got %#v", next)
	}
	if len(effects) != 1 {
		t.Fatalf("expected storage load effect, got %#v", effects)
	}
	if _, ok := effects[0].(FuncEffect); !ok {
		t.Fatalf("expected storage load effect, got %#v", effects)
	}
}

func TestOverlayKeyboardCommandsUseCanonicalClipboardHandlers(t *testing.T) {
	inputReducer := NewUIInputReducer()
	root := state.Root{
		Shell: state.DefaultShell().OpenClipboardHistory(),
		Clipboard: state.ClipboardStore{
			Entries: []state.ClipboardEntry{{ID: "clip:1", Title: "alpha", Text: "alpha", Preview: "alpha"}},
		},
	}

	next, effects := inputReducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x05", Ctrl: true}})
	if next.Shell.Overlay.Kind != state.OverlayPrompt || next.Shell.Overlay.Prompt.Purpose != "clipboard.edit" || len(effects) != 1 {
		t.Fatalf("clipboard edit shortcut should enter its app handler directly, root=%#v effects=%#v", next, effects)
	}

	next, effects = inputReducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x0e", Ctrl: true}})
	if next.Shell.Overlay.Kind != state.OverlayPrompt || next.Shell.Overlay.Prompt.Purpose != "clipboard.new" || len(effects) != 1 {
		t.Fatalf("clipboard new shortcut should enter its app handler directly, root=%#v effects=%#v", next, effects)
	}

	next, effects = inputReducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x18", Ctrl: true}})
	if len(next.Clipboard.Entries) != 0 || len(effects) != 2 {
		t.Fatalf("clipboard delete shortcut should execute canonical reducer and persist, root=%#v effects=%#v", next, effects)
	}
}

func TestOverlayMouseWheelMovesCurrentSelection(t *testing.T) {
	inputReducer := NewUIInputReducer()
	tests := []struct {
		name        string
		root        state.Root
		wantKind    state.OverlayKind
		wantPreview bool
	}{
		{
			name: "terminal picker",
			root: state.Root{
				Shell:        state.DefaultShell().OpenTerminalPicker(),
				TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{TerminalID: "term-a", Title: "a"}, {TerminalID: "term-b", Title: "b"}}},
			},
			wantKind: state.OverlayTerminalPicker,
		},
		{
			name: "terminal pool",
			root: state.Root{
				Shell:        state.DefaultShell().OpenTerminalPool(),
				TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{TerminalID: "term-a", Title: "a"}, {TerminalID: "term-b", Title: "b"}}},
			},
			wantKind:    state.OverlayTerminalPool,
			wantPreview: true,
		},
		{
			name: "workbench tree",
			root: state.Root{
				Shell: state.DefaultShell().
					SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneEmpty}, state.SplitDirectionVertical).
					OpenWorkbenchTree(),
			},
			wantKind: state.OverlayWorkbenchTree,
		},
		{
			name: "clipboard history",
			root: state.Root{
				Shell: state.DefaultShell().OpenClipboardHistory(),
				Clipboard: state.ClipboardStore{
					Entries: []state.ClipboardEntry{
						{ID: "clip:1", Title: "one", Text: "one", Preview: "one"},
						{ID: "clip:2", Title: "two", Text: "two", Preview: "two"},
					},
				},
			},
			wantKind: state.OverlayClipboardHistory,
		},
		{
			name: "floating overview",
			root: state.Root{
				Shell: shellWithFloatingOverviewForMouseWheelTest(),
			},
			wantKind: state.OverlayFloatingOverview,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			next, effects := inputReducer(tc.root, ShellOverlayMouseSelectMsg{Delta: 1})
			if next.Shell.Overlay.Kind != tc.wantKind || next.Shell.Overlay.SelectedIndex != 1 {
				t.Fatalf("wheel down should move overlay selection, overlay=%#v", next.Shell.Overlay)
			}
			wantEffects := 1
			if tc.wantPreview {
				wantEffects = 2
			}
			if len(effects) != wantEffects {
				t.Fatalf("overlay wheel should be handled locally, effects=%#v", effects)
			}
			if _, ok := effects[0].(handledEffect); !ok {
				t.Fatalf("overlay wheel should emit handled effect, got %#v", effects[0])
			}
			if tc.wantPreview && !terminalPoolPreviewRefreshScheduled(t, effects) {
				t.Fatalf("terminal pool wheel should refresh preview, effects=%#v", effects)
			}
			next, effects = inputReducer(next, ShellOverlayMouseSelectMsg{Delta: -1})
			if next.Shell.Overlay.SelectedIndex != 0 {
				t.Fatalf("wheel up should move overlay selection back, overlay=%#v effects=%#v", next.Shell.Overlay, effects)
			}
		})
	}
}

func shellWithFloatingOverviewForMouseWheelTest() state.ShellStore {
	shell := state.DefaultShell()
	shell, _ = shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandCreate, TargetID: "floating-1", Title: "one", Pane: state.PaneState{ID: "floating-pane-1", Title: "one", Kind: state.PaneEmpty}})
	shell, _ = shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandCreate, TargetID: "floating-2", Title: "two", Pane: state.PaneState{ID: "floating-pane-2", Title: "two", Kind: state.PaneEmpty}})
	return shell.OpenFloatingOverview()
}

func TestInteractiveRuntimeCtrlFDoesNotSendTerminalInput(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(8)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	terminal.AttachResult = port.TerminalAttachResult{Channel: 9, Cols: 80, Rows: 24}
	terminal.ListResult = port.TerminalListResult{Items: []port.TerminalPoolItem{
		{TerminalID: "term-1", Title: "term-1", State: "running", Cols: 80, Rows: 24},
	}}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x06", Ctrl: true}); err != nil {
		t.Fatalf("send ctrl-f: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain ctrl-f: %v", err)
	}

	if !runtime.State().Shell.Overlay.Open || runtime.State().Shell.Overlay.Kind != state.OverlayTerminalPicker {
		t.Fatalf("expected terminal picker overlay, got %#v", runtime.State().Shell.Overlay)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("ctrl-f must not be sent to terminal, got %#v", terminal.Inputs)
	}
	last := lastFrame(t, host.Frames())
	if !frameContains(last, "terminal picker") ||
		!frameContains(last, "search:") ||
		!frameContains(last, "▸ + new terminal") ||
		!frameContains(last, "● term-1") ||
		!frameContains(last, "attached") ||
		!frameContains(last, "80x24") ||
		!frameContains(last, "+ new terminal") ||
		!frameContains(last, "Create terminal") ||
		frameContains(last, "@pane-main") ||
		frameContains(last, "Select terminal source state target") ||
		frameContains(last, "DETAIL") {
		t.Fatalf("expected terminal picker product content in frame, got %#v", last.Lines)
	}
}

func TestInteractiveRuntimeOverlayMouseWheelMovesSelectionWithoutTerminalLeak(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-main", Channel: 4, Cols: 80, Rows: 24},
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{
			{TerminalID: "term-a", Title: "alpha", State: "running"},
			{TerminalID: "term-b", Title: "beta", State: "running"},
		}},
	}
	host := NewFakeTerminalHost(32)
	host.SetSize(96, 28)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-main", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := runtime.Post(ShellOpenTerminalPoolMsg{}); err != nil {
		t.Fatalf("post terminal pool: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain terminal pool: %v", err)
	}
	overlay := frameHitRegion(t, lastFrame(t, host.Frames()), render.HitRegionOverlay, "")
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseWheelDown, Row: overlay.Rect.Y + 1, Col: overlay.Rect.X + 1}); err != nil {
		t.Fatalf("send overlay wheel: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain overlay wheel: %v", err)
	}
	if runtime.State().Shell.Overlay.SelectedIndex != 1 {
		t.Fatalf("overlay wheel should move selection, overlay=%#v", runtime.State().Shell.Overlay)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("overlay wheel must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeTerminalPickerKeyboardFlow(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-main", Channel: 4, Cols: 80, Rows: 24},
		ListResult:   port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-2", Title: "日志🚀", State: "running", Cols: 100, Rows: 30}}},
	}
	initialShell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "日志🚀", Kind: state.PaneTerminalLive, TerminalID: "term-2"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{Shell: initialShell},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-main", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	terminal.AttachResult = port.TerminalAttachResult{Channel: 9, Cols: 80, Rows: 24}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x06", Ctrl: true}); err != nil {
		t.Fatalf("send ctrl-f: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "日"}); err != nil {
		t.Fatalf("send query char: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "志"}); err != nil {
		t.Fatalf("send query char: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain query: %v", err)
	}
	if runtime.State().Shell.EnsureDefaults().Overlay.Query != "日志" {
		t.Fatalf("expected picker query retained in reducer state, got %#v", runtime.State().Shell.Overlay)
	}
	queryFrame := lastFrame(t, host.Frames())
	if !frameContains(queryFrame, "search: 日志") || !frameContains(queryFrame, "▸ ● 日志🚀") || !frameContains(queryFrame, "running") || !frameContains(queryFrame, "100x30") || frameContains(queryFrame, "@pane-2") || frameContains(queryFrame, "DETAIL 日志🚀") {
		t.Fatalf("expected filtered picker frame, got %#v", queryFrame.Lines)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("picker query must not leak to terminal input, got %#v", terminal.Inputs)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}); err != nil {
		t.Fatalf("send enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain enter: %v", err)
	}
	if runtime.State().Shell.EnsureDefaults().ActivePaneID != state.DefaultPaneID || runtime.State().Shell.Overlay.Open || runtime.State().Session.TerminalID != "term-2" {
		t.Fatalf("enter should attach selected terminal into current pane and close overlay, state=%#v shell=%#v", runtime.State().Session, runtime.State().Shell)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("picker navigation must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeTerminalPickerShowsExitedTerminalImmediatelyAfterAttach(t *testing.T) {
	exitedAt := time.Date(2026, 6, 17, 12, 45, 0, 0, time.UTC)
	exitCode := 23
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-main", Channel: 4, Cols: 80, Rows: 24},
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{
			TerminalID: "term-dead",
			Title:      "dead shell",
			State:      string(state.TerminalLiveExited),
			ExitCode:   &exitCode,
			ExitedAt:   exitedAt,
			Command:    []string{"bash", "-lc", "exit 23"},
			Cols:       80,
			Rows:       24,
		}}},
		SurfaceResult: port.TerminalSurfaceResult{
			Ready: true,
			Snapshot: state.LiveSurfaceSnapshot{
				TerminalID: "term-dead",
				Cols:       80,
				Rows:       24,
				State:      state.TerminalLiveExited,
				ExitCode:   23,
				ExitReason: "exited",
				ExitedAt:   exitedAt,
				Command:    []string{"bash", "-lc", "exit 23"},
			},
		},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{Shell: state.DefaultShell()},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-main", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	terminal.AttachResult = port.TerminalAttachResult{Channel: 9, Cols: 80, Rows: 24}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x06", Ctrl: true}); err != nil {
		t.Fatalf("send ctrl-f: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "d"}); err != nil {
		t.Fatalf("send query d: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "e"}); err != nil {
		t.Fatalf("send query e: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain picker query: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyDown}); err != nil {
		t.Fatalf("send down: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}); err != nil {
		t.Fatalf("send enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain exited attach: %v", err)
	}

	if runtime.State().Shell.Overlay.Open {
		t.Fatalf("picker should close after attach, overlay=%#v", runtime.State().Shell.Overlay)
	}
	surface := runtime.State().Surface.SurfaceForTerminal("term-dead")
	if surface.State != state.TerminalLiveExited || surface.ExitCode != 23 {
		t.Fatalf("attached exited terminal should be visible immediately, surface=%#v", surface)
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "terminal exited: term-dead code:23 exited") ||
		!frameContains(frame, "exited at: 2026-06-17T12:45:00Z") ||
		!frameContains(frame, "command: bash -lc exit 23") ||
		!frameContains(frame, "restart") ||
		!frameContains(frame, "reconnect") {
		t.Fatalf("exited terminal selected from picker should render lifecycle CTA without extra input, frame=%#v", frame.Lines)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("picker attach to exited terminal must not require/leak input, got %#v", terminal.Inputs)
	}
}

func TestUIInputReducerGlobalQuitShortcutEmitsQuitMsg(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModeGlobal)}

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "q"}})
	if root.Shell.InteractionMode != state.InteractionModeNormal || hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("global q should be handled and emit quit effect, got %#v", effects)
	}
	if _, ok := effects[0].(handledEffect); !ok {
		t.Fatalf("global q should mark input handled, got %#v", effects[0])
	}
	msg := runFirstNonStickyTimeoutEffect(t, effects)
	if _, ok := msg.(QuitMsg); !ok {
		t.Fatalf("global q should emit QuitMsg, got %#v", msg)
	}
}

func TestInteractiveRuntimeTerminalPickerEnterDefaultsToCreateTerminal(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-main", Channel: 4, Cols: 80, Rows: 24},
		CreateResult: port.TerminalCreateResult{TerminalID: "term-created", State: "running"},
	}
	initialShell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-2"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{Shell: initialShell},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-main", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x06", Ctrl: true}); err != nil {
		t.Fatalf("send ctrl-f: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}); err != nil {
		t.Fatalf("send enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain enter: %v", err)
	}
	if len(terminal.Creates) != 0 || len(terminal.Attaches) != 1 {
		t.Fatalf("picker default enter should only open create form, creates=%#v attaches=%#v", terminal.Creates, terminal.Attaches)
	}
	if !runtime.State().Shell.Overlay.Open || runtime.State().Shell.Overlay.Kind != state.OverlayPrompt || runtime.State().Shell.Overlay.Prompt.Purpose != "terminal.create" {
		t.Fatalf("picker default enter should open create terminal prompt, overlay=%#v", runtime.State().Shell.Overlay)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("picker create navigation must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeCreateTerminalFormSubmitsTerminalCreate(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		CreateResult: port.TerminalCreateResult{TerminalID: "term-created", State: "running"},
	}
	host := NewFakeTerminalHost(64)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{Shell: state.DefaultShell().OpenPrompt(createTerminalPrompt(state.DefaultPaneID))},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	sendChars := func(value string) {
		t.Helper()
		for _, r := range value {
			if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: string(r)}); err != nil {
				t.Fatalf("send char %q: %v", r, err)
			}
		}
	}
	sendChars("my-term")
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyTab}); err != nil {
		t.Fatalf("send tab: %v", err)
	}
	sendChars("bash -lc 'echo hi'")
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}); err != nil {
		t.Fatalf("send enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain create form: %v", err)
	}
	if len(terminal.Creates) != 1 {
		t.Fatalf("create form should call terminal create once, got %#v", terminal.Creates)
	}
	if len(terminal.Attaches) != 1 {
		t.Fatalf("successful create should attach new terminal once, got %#v", terminal.Attaches)
	}
	create := terminal.Creates[0]
	if create.EndpointID != state.DefaultEndpointID || create.Title != "my-term" || len(create.Command) != 3 || create.Command[2] != "echo hi" || len(create.Tags) != 0 {
		t.Fatalf("unexpected create request %#v", create)
	}
	attach := terminal.Attaches[0]
	if attach.TerminalID != "term-created" || attach.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) || attach.ResizePolicy != "owner" {
		t.Fatalf("create should attach created terminal to current pane, got %#v", attach)
	}
	if runtime.State().Session.TerminalID != "term-created" || runtime.State().Shell.EnsureDefaults().Workspace.Tabs[0].Panes[0].TerminalID != "term-created" {
		t.Fatalf("create should switch current pane to created terminal, state=%#v shell=%#v", runtime.State().Session, runtime.State().Shell)
	}
	if runtime.State().Shell.Overlay.Open {
		t.Fatalf("successful create should close overlay, got %#v", runtime.State().Shell.Overlay)
	}
}

func TestInteractiveRuntimeCreateTerminalFormRequiresName(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		CreateResult: port.TerminalCreateResult{TerminalID: "term-created", State: "running"},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{Shell: state.DefaultShell().OpenPrompt(createTerminalPrompt(state.DefaultPaneID))},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}); err != nil {
		t.Fatalf("send enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain create form: %v", err)
	}
	if len(terminal.Creates) != 0 {
		t.Fatalf("missing name must not create terminal, got %#v", terminal.Creates)
	}
	prompt := runtime.State().Shell.EnsureDefaults().Overlay.Prompt
	if !runtime.State().Shell.Overlay.Open || prompt.LastResult != "name is required" {
		t.Fatalf("missing name should keep form open with validation text, overlay=%#v", runtime.State().Shell.Overlay)
	}
}

func TestUIInputReducerCreateTerminalFormEditsAndCancels(t *testing.T) {
	reducer := ComposeReducers(NewBackNavigationReducer(CopyModeDeps{}), NewUIInputReducer())
	root := state.Root{Shell: state.DefaultShell().OpenPrompt(createTerminalPrompt(state.DefaultPaneID))}
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "a"}})
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "b"}})
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyLeft}})
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"}})
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyBackspace}})
	prompt := root.Shell.EnsureDefaults().Overlay.Prompt
	if prompt.FieldRawValue("name") != "ab" || prompt.ActiveField != 0 || prompt.Fields[0].Cursor != 1 {
		t.Fatalf("name field should edit at cursor and delete in-place, prompt=%#v", prompt)
	}
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyTab}})
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "s"}})
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "h"}})
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyHome}})
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "z"}})
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyShiftTab}})
	prompt = root.Shell.EnsureDefaults().Overlay.Prompt
	if prompt.ActiveField != 0 || prompt.FieldRawValue("command") != "zsh" || prompt.Fields[1].Cursor != 1 {
		t.Fatalf("tab should move between editable form fields without losing cursor values, prompt=%#v", prompt)
	}
	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}})
	if root.Shell.EnsureDefaults().Overlay.Open || len(effects) != 0 {
		t.Fatalf("esc should close create form and consume input, root=%#v effects=%#v", root, effects)
	}
}

func TestUIInputReducerPromptConsumesPasteAsText(t *testing.T) {
	reducer := ComposeReducers(NewBackNavigationReducer(CopyModeDeps{}), NewUIInputReducer())
	root := state.Root{Shell: state.DefaultShell().OpenPrompt(createTerminalPrompt(state.DefaultPaneID))}
	next, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindPaste, Paste: "name-\x07-tail"}})
	prompt := next.Shell.EnsureDefaults().Overlay.Prompt
	if prompt.FieldRawValue("name") != "name-\x07-tail" || !next.Shell.EnsureDefaults().Overlay.Open {
		t.Fatalf("prompt must consume paste body as one text edit, prompt=%#v", prompt)
	}
	if len(effects) == 0 {
		t.Fatal("prompt paste must be marked handled and refresh completion state")
	}
}

func TestUIInputReducerCreateTerminalWorkdirSuggestions(t *testing.T) {
	pathService := &testkit.FakeTerminalService{PathResult: promptPathResult("demo/", "delta/", "dev/")}
	runtime := newPromptPathRuntime(pathService, "d")

	postPromptKey(t, runtime, input.KeyTab)
	prompt := runtime.State().Shell.EnsureDefaults().Overlay.Prompt
	if !prompt.SuggestionFocused || len(prompt.ActiveSuggestionItems()) != 3 || prompt.ActiveField != 2 {
		t.Fatalf("tab on workdir should enter suggestion focus, prompt=%#v", prompt)
	}
	if len(pathService.PathRequests) != 1 || pathService.PathRequests[0].EndpointID != state.DefaultEndpointID || pathService.PathRequests[0].Prefix != "d" {
		t.Fatalf("local create workdir completion should use path service, requests=%#v", pathService.PathRequests)
	}
	postPromptKey(t, runtime, input.KeyTab)
	postPromptKey(t, runtime, input.KeyTab)
	postPromptKey(t, runtime, input.KeyEnter)
	prompt = runtime.State().Shell.EnsureDefaults().Overlay.Prompt
	want := "dev/"
	if prompt.Submitted || prompt.SuggestionFocused || prompt.FieldRawValue("workdir") != want || prompt.Fields[2].Cursor != len([]rune(want)) {
		t.Fatalf("enter should accept selected workdir suggestion without submitting, want=%q prompt=%#v", want, prompt)
	}

	pathService = &testkit.FakeTerminalService{PathResult: promptPathResult("demo/", "delta/", "dev/")}
	runtime = newPromptPathRuntime(pathService, "d")
	postPromptKey(t, runtime, input.KeyTab)
	postPromptKey(t, runtime, input.KeyShiftTab)
	prompt = runtime.State().Shell.EnsureDefaults().Overlay.Prompt
	if !prompt.SuggestionFocused || prompt.SuggestionSelected != 2 {
		t.Fatalf("shift-tab in suggestion focus should wrap to previous candidate, prompt=%#v", prompt)
	}

	pathService = &testkit.FakeTerminalService{PathResult: promptPathResult("demo/", "delta/", "dev/")}
	runtime = newPromptPathRuntime(pathService, "d")
	postPromptKey(t, runtime, input.KeyTab)
	postPromptKey(t, runtime, input.KeyTab)
	postPromptKey(t, runtime, input.KeyTab)
	pathService.PathResult = promptPathResult("dev/src/")
	postPromptKey(t, runtime, input.KeyRight)
	prompt = runtime.State().Shell.EnsureDefaults().Overlay.Prompt
	if !prompt.SuggestionFocused || prompt.FieldRawValue("workdir") != want || len(prompt.ActiveSuggestionItems()) != 1 || prompt.ActiveSuggestionItems()[0] != "dev/src/" {
		t.Fatalf("right should enter selected directory and keep suggestion focus, want=%q prompt=%#v", want, prompt)
	}
	pathService.PathResult = promptPathResult("demo/", "delta/", "dev/")
	postPromptKey(t, runtime, input.KeyLeft)
	prompt = runtime.State().Shell.EnsureDefaults().Overlay.Prompt
	if prompt.SuggestionFocused || prompt.FieldRawValue("workdir") != "" || len(prompt.ActiveSuggestionItems()) != 0 {
		t.Fatalf("left should move to empty parent path and clear suggestions, prompt=%#v", prompt)
	}

	pathService = &testkit.FakeTerminalService{PathResult: promptPathResult("demo/", "delta/", "dev/")}
	runtime = newPromptPathRuntime(pathService, "d")
	postPromptKey(t, runtime, input.KeyTab)
	postPromptKey(t, runtime, input.KeyEsc)
	prompt = runtime.State().Shell.EnsureDefaults().Overlay.Prompt
	if !runtime.State().Shell.EnsureDefaults().Overlay.Open || prompt.SuggestionFocused {
		t.Fatalf("esc in suggestion focus should only exit suggestions, overlay=%#v", runtime.State().Shell.Overlay)
	}

	postPromptKey(t, runtime, input.KeyEsc)
	if runtime.State().Shell.EnsureDefaults().Overlay.Open {
		t.Fatalf("second esc should close prompt after leaving suggestions, overlay=%#v", runtime.State().Shell.Overlay)
	}
}

func newPromptPathRuntime(pathService port.PathService, prefix string) *AppRuntime {
	root := state.Root{Shell: state.DefaultShell().OpenPrompt(state.PromptState{
		Title:            "Create Terminal",
		Purpose:          "terminal.create",
		TargetEndpointID: state.DefaultEndpointID,
		ActiveField:      2,
		Fields: []state.PromptFieldState{
			{Key: "name", Label: "name", Value: "shell", Required: true},
			{Key: "command", Label: "command", Value: "/bin/sh"},
			{Key: "workdir", Label: "workdir", Value: prefix, Cursor: len([]rune(prefix))},
		},
	})}
	return NewInteractiveRuntime(root, NewFakeTerminalHost(1), NewSyncEffectRunner(), LiveDeps{Path: pathService}, CopyModeDeps{})
}

func promptPathResult(paths ...string) port.PathListDirectoriesResult {
	result := port.PathListDirectoriesResult{BasePath: "/daemon/cwd", Entries: make([]port.PathDirectoryEntry, 0, len(paths))}
	for _, path := range paths {
		result.Entries = append(result.Entries, port.PathDirectoryEntry{Name: strings.TrimSuffix(path, "/"), Path: path})
	}
	return result
}

func postPromptKey(t *testing.T, runtime *AppRuntime, key input.Key) {
	t.Helper()
	if err := runtime.Post(InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: key}}); err != nil {
		t.Fatalf("post key %q: %v", key, err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain key %q: %v", key, err)
	}
}

func TestCreateTerminalPromptSubmitUsesEditedFields(t *testing.T) {
	dir := t.TempDir()
	workdir := filepath.Join(dir, "work")
	if err := os.Mkdir(workdir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	reducer := NewUIInputReducer()
	root := state.Root{Shell: state.DefaultShell().OpenPrompt(createTerminalPrompt(state.DefaultPaneID))}
	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "d"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "e"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "v"},
		{Kind: input.EventKindKey, Key: input.KeyTab},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "b"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "a"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "s"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "h"},
		{Kind: input.EventKindKey, Key: input.KeyTab},
		{Kind: input.EventKindKey, Key: input.KeyDown},
		{Kind: input.EventKindKey, Key: input.KeyHome},
		{Kind: input.EventKindKey, Key: input.KeyEnd},
	} {
		root, _ = reducer(root, InputMsg{Event: event})
	}
	root.Shell = root.Shell.SetPromptValue(workdir)

	next, effects := reducePromptSubmit(root)
	if next.Shell.Overlay.Open || len(effects) != 1 {
		t.Fatalf("submit should close prompt and emit create effect, root=%#v effects=%#v", next, effects)
	}
	effect, ok := effects[0].(FuncEffect)
	if !ok || effect.Run == nil {
		t.Fatalf("expected create FuncEffect, got %#v", effects)
	}
	msg := effect.Run(context.Background())
	request, ok := msg.(TerminalPoolCreateRequestMsg)
	if !ok {
		t.Fatalf("expected terminal create request, got %#v", msg)
	}
	if request.EndpointID != state.DefaultEndpointID || request.Title != "dev" || strings.Join(request.Command, " ") != "bash" || request.CWD != workdir {
		t.Fatalf("edited prompt fields should drive create request, got %#v", request)
	}

	root = state.Root{Shell: state.DefaultShell().OpenPrompt(createTerminalPrompt(state.DefaultPaneID))}
	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "c"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"},
		{Kind: input.EventKindKey, Key: input.KeyTab},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "c"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "o"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "d"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "e"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"},
	} {
		root, _ = reducer(root, InputMsg{Event: event})
	}
	next, effects = reducePromptSubmit(root)
	if next.Shell.Overlay.Open || len(effects) != 1 {
		t.Fatalf("submit custom command should close prompt and emit effect, root=%#v effects=%#v", next, effects)
	}
	effect = effects[0].(FuncEffect)
	request = effect.Run(context.Background()).(TerminalPoolCreateRequestMsg)
	if request.Title != "cx" || strings.Join(request.Command, " ") != "codex" {
		t.Fatalf("custom command should override default shell fallback, got %#v", request)
	}
}

func TestCreateTerminalPromptServerFieldUsesEndpointDropdown(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{
		Endpoints: ((state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "This Mac", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true}).
			Upsert(state.EndpointItem{ID: "west", Label: "US West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true})).
			ApplyDefaults(state.DefaultEndpointID, []string{"/bin/zsh"}, "/Users/me", "").
			ApplyDefaults("west", []string{"/bin/bash", "-l"}, "/srv/west", ""),
	}
	prompt := createTerminalPromptForTargetEndpoint(root, terminalPoolTarget{PaneID: state.DefaultPaneID}, state.DefaultEndpointID)
	root.Shell = state.DefaultShell().OpenPrompt(prompt).SetPromptValue("remote-shell").MovePromptField(2)

	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyTab}})
	prompt = root.Shell.EnsureDefaults().Overlay.Prompt
	if !prompt.SuggestionFocused || prompt.ActiveField != 2 || len(prompt.ActiveSuggestionItems()) != 2 || prompt.ActiveSuggestionItems()[0] != "This Mac (local)" || prompt.ActiveSuggestionItems()[1] != "US West (west)" {
		t.Fatalf("server field should show endpoint dropdown, prompt=%#v", prompt)
	}
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyTab}})
	root, _ = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	prompt = root.Shell.EnsureDefaults().Overlay.Prompt
	if prompt.SuggestionFocused || prompt.FieldRawValue("server") != "US West (west)" || prompt.FieldRawValue("workdir") != "/srv/west" ||
		strings.Join(prompt.Command, " ") != "/bin/bash -l" || promptFieldPlaceholder(prompt, "command") != "/bin/bash -l" {
		t.Fatalf("enter should accept selected endpoint suggestion, prompt=%#v", prompt)
	}

	next, effects := reducePromptSubmit(root)
	if next.Shell.Overlay.Open || len(effects) != 1 {
		t.Fatalf("submit should close prompt and emit create effect, root=%#v effects=%#v", next, effects)
	}
	effect := effects[0].(FuncEffect)
	request := effect.Run(context.Background()).(TerminalPoolCreateRequestMsg)
	if request.EndpointID != "west" || request.Title != "remote-shell" || request.CWD != "/srv/west" || strings.Join(request.Command, " ") != "/bin/bash -l" {
		t.Fatalf("dropdown-selected server should route create request, got %#v", request)
	}
}

func TestCreateTerminalPromptRemembersLastEndpointCommandAndWorkdir(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell(),
		Endpoints: ((state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "This Mac", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true}).
			Upsert(state.EndpointItem{ID: "west", Label: "US West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true})).
			ApplyDefaults("west", []string{"/bin/bash"}, "/srv/west", ""),
	}
	prompt := createTerminalPromptForTargetEndpoint(root, terminalPoolTarget{PaneID: state.DefaultPaneID}, "west")
	for index := range prompt.Fields {
		switch prompt.Fields[index].Key {
		case "name":
			prompt.Fields[index].Value = "build"
		case "command":
			prompt.Fields[index].Value = "npm run dev"
		case "workdir":
			prompt.Fields[index].Value = "/srv/project"
		}
	}
	root.Shell = root.Shell.OpenPrompt(prompt)

	next, effects := reducePromptSubmit(root)
	if next.Shell.Overlay.Open || len(effects) != 1 {
		t.Fatalf("submit should close prompt and emit create effect, root=%#v effects=%#v", next, effects)
	}
	if draft := next.Shell.TerminalCreateDraft; draft.EndpointID != "west" || draft.Command != "npm run dev" || draft.Workdir != "/srv/project" {
		t.Fatalf("submit should remember create draft, shell=%#v", next.Shell)
	}
	again := createTerminalPromptForTargetEndpoint(next, terminalPoolTarget{PaneID: state.DefaultPaneID}, "")
	if again.TargetEndpointID != "west" || again.FieldRawValue("server") != "US West (west)" ||
		again.FieldRawValue("command") != "npm run dev" || again.FieldRawValue("workdir") != "/srv/project" ||
		again.FieldRawValue("name") != "" {
		t.Fatalf("next create prompt should reuse endpoint/command/workdir but not unique name, prompt=%#v", again)
	}
}

func TestCreateTerminalPromptWorkdirDefaultFollowsEndpoint(t *testing.T) {
	root := state.Root{
		Endpoints: ((state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "This Mac", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true}).
			Upsert(state.EndpointItem{ID: "west", Label: "US West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true})).
			ApplyDefaults(state.DefaultEndpointID, []string{"/bin/zsh"}, "/Users/me", "").
			ApplyDefaults("west", []string{"/bin/bash"}, "/srv/west", ""),
	}

	localPrompt := createTerminalPromptForTargetEndpoint(root, terminalPoolTarget{PaneID: state.DefaultPaneID}, state.DefaultEndpointID)
	if localPrompt.Workdir != "/Users/me" || localPrompt.FieldRawValue("workdir") != "/Users/me" {
		t.Fatalf("local create prompt should default to local core cwd, prompt=%#v", localPrompt)
	}
	remotePrompt := createTerminalPromptForTargetEndpoint(root, terminalPoolTarget{PaneID: state.DefaultPaneID}, "west")
	if remotePrompt.Workdir != "/srv/west" || remotePrompt.FieldRawValue("workdir") != "/srv/west" {
		t.Fatalf("remote create prompt should default to remote core cwd, prompt=%#v", remotePrompt)
	}
}

func TestCreateTerminalPromptRemoteDefaultCommandDoesNotUseLocalShell(t *testing.T) {
	root := state.Root{
		Endpoints: ((state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "This Mac", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true}).
			Upsert(state.EndpointItem{ID: "west", Label: "US West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true})).
			ApplyDefaults(state.DefaultEndpointID, []string{"/bin/zsh"}, "/Users/me", "").
			ApplyDefaults("west", []string{"/bin/bash", "-l"}, "/srv/west", ""),
	}

	localPrompt := createTerminalPromptForTargetEndpoint(root, terminalPoolTarget{PaneID: state.DefaultPaneID}, state.DefaultEndpointID)
	if strings.Join(localPrompt.Command, " ") != "/bin/zsh" || promptFieldPlaceholder(localPrompt, "command") != "/bin/zsh" {
		t.Fatalf("local create prompt should use local core shell default, prompt=%#v", localPrompt)
	}
	remotePrompt := createTerminalPromptForTargetEndpoint(root, terminalPoolTarget{PaneID: state.DefaultPaneID}, "west")
	if strings.Join(remotePrompt.Command, " ") != "/bin/bash -l" || promptFieldPlaceholder(remotePrompt, "command") != "/bin/bash -l" {
		t.Fatalf("remote create prompt must use remote core shell default, prompt=%#v", remotePrompt)
	}

	remotePrompt.Fields[0].Value = "remote-shell"
	root.Shell = state.DefaultShell().OpenPrompt(remotePrompt)
	_, effects := reducePromptSubmit(root)
	request := effects[0].(FuncEffect).Run(context.Background()).(TerminalPoolCreateRequestMsg)
	if request.EndpointID != "west" || strings.Join(request.Command, " ") != "/bin/bash -l" {
		t.Fatalf("remote create submit should use remote core default command, got %#v", request)
	}
}

func promptFieldPlaceholder(prompt state.PromptState, key string) string {
	for _, field := range prompt.Fields {
		if field.Key == key {
			return field.Placeholder
		}
	}
	return ""
}

func TestCreateTerminalPromptDoesNotSendAutoLocalWorkdirToRemote(t *testing.T) {
	root := state.Root{
		Endpoints: ((state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "This Mac", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true}).
			Upsert(state.EndpointItem{ID: "west", Label: "US West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true})).
			ApplyDefaults(state.DefaultEndpointID, []string{"/bin/zsh"}, "/Users/me/project", "").
			ApplyDefaults("west", []string{"/bin/bash"}, "/srv/west", ""),
	}
	prompt := createTerminalPromptForTargetEndpoint(root, terminalPoolTarget{PaneID: state.DefaultPaneID}, state.DefaultEndpointID)
	for index := range prompt.Fields {
		switch prompt.Fields[index].Key {
		case "name":
			prompt.Fields[index].Value = "remote-shell"
		case "server":
			prompt.Fields[index].Value = "west"
		}
	}
	root.Shell = state.DefaultShell().OpenPrompt(prompt)

	next, effects := reducePromptSubmit(root)
	if next.Shell.Overlay.Open || len(effects) != 1 {
		t.Fatalf("submit should close prompt and emit create effect, root=%#v effects=%#v", next, effects)
	}
	request := effects[0].(FuncEffect).Run(context.Background()).(TerminalPoolCreateRequestMsg)
	if request.EndpointID != "west" || request.CWD != "/srv/west" {
		t.Fatalf("remote create must replace stale local cwd with remote default, got %#v", request)
	}

	prompt = createTerminalPromptForTargetEndpoint(root, terminalPoolTarget{PaneID: state.DefaultPaneID}, "west")
	for index := range prompt.Fields {
		switch prompt.Fields[index].Key {
		case "name":
			prompt.Fields[index].Value = "remote-shell"
		case "workdir":
			prompt.Fields[index].Value = "/root"
		}
	}
	root.Shell = state.DefaultShell().OpenPrompt(prompt)
	_, effects = reducePromptSubmit(root)
	request = effects[0].(FuncEffect).Run(context.Background()).(TerminalPoolCreateRequestMsg)
	if request.EndpointID != "west" || request.CWD != "/root" {
		t.Fatalf("explicit remote cwd should be preserved, got %#v", request)
	}
}

func TestCreateTerminalPromptSubmitRoutesSelectedEndpoint(t *testing.T) {
	root := state.Root{
		Endpoints: ((state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "This Mac", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true}).
			Upsert(state.EndpointItem{ID: "west", Label: "US West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true})).
			ApplyDefaults("west", []string{"/bin/bash"}, "/srv/west", ""),
	}
	prompt := createTerminalPromptForTargetEndpoint(root, terminalPoolTarget{PaneID: state.DefaultPaneID}, "west")
	root.Shell = state.DefaultShell().OpenPrompt(prompt).SetPromptValue("remote-shell")

	next, effects := reducePromptSubmit(root)
	if next.Shell.Overlay.Open || len(effects) != 1 {
		t.Fatalf("submit should close prompt and emit create effect, root=%#v effects=%#v", next, effects)
	}
	effect, ok := effects[0].(FuncEffect)
	if !ok || effect.Run == nil {
		t.Fatalf("expected create FuncEffect, got %#v", effects)
	}
	request, ok := effect.Run(context.Background()).(TerminalPoolCreateRequestMsg)
	if !ok {
		t.Fatalf("expected terminal create request, got %#v", effects)
	}
	if request.EndpointID != "west" || request.TargetPaneID != state.DefaultPaneID || request.Title != "remote-shell" {
		t.Fatalf("create prompt should route to selected endpoint, got %#v", request)
	}
}

func TestInteractiveRuntimeTerminalPickerUsesTerminalPoolService(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-pool", Channel: 9, Cols: 80, Rows: 24},
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{
			TerminalID: "term-pool",
			Title:      "远程🚀",
			State:      "running",
		}}},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x06", Ctrl: true}); err != nil {
		t.Fatalf("send ctrl-f: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain picker list: %v", err)
	}
	if len(terminal.Lists) != 1 || runtime.State().TerminalPool.Status != state.TerminalPoolReady {
		t.Fatalf("expected picker open to load terminal pool, lists=%#v pool=%#v", terminal.Lists, runtime.State().TerminalPool)
	}
	for _, frame := range host.Frames() {
		if frameContains(frame, "loading terminals") {
			t.Fatalf("picker open must not render transient loading frame, got %#v", frame.Lines)
		}
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "远程🚀") || !frameContains(frame, "running") || frameContains(frame, "@pool") || frameContains(frame, "term-pool") {
		t.Fatalf("expected pool row in picker frame, got %#v", frame.Lines)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyDown}); err != nil {
		t.Fatalf("send down: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}); err != nil {
		t.Fatalf("send enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pool attach: %v", err)
	}
	if len(terminal.Attaches) != 1 || terminal.Attaches[0].TerminalID != "term-pool" {
		t.Fatalf("expected pool attach through service, got %#v", terminal.Attaches)
	}
	if !runtime.State().Session.Attached || runtime.State().Session.TerminalID != "term-pool" || runtime.State().Shell.Overlay.Open {
		t.Fatalf("expected attached pool terminal and closed overlay, got session=%#v shell=%#v", runtime.State().Session, runtime.State().Shell)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("picker pool navigation must not leak terminal input, got %#v", terminal.Inputs)
	}
}

func TestTerminalPoolReducerHandlesListErrorCreateAndStaleResult(t *testing.T) {
	terminal := &testkit.FakeTerminalService{ListErr: errors.New("list failed")}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root, effects := reducer(state.Root{Shell: state.DefaultShell()}, TerminalPoolListRequestMsg{})
	if root.TerminalPool.Status != state.TerminalPoolLoading || len(effects) != 1 {
		t.Fatalf("expected loading pool and list effect, got root=%#v effects=%#v", root, effects)
	}
	root, _ = reducer(root, TerminalPoolListResultMsg{Seq: root.TerminalPool.RequestSeq, Err: errors.New("list failed")})
	if root.TerminalPool.Status != state.TerminalPoolError || root.TerminalPool.LastError != "list failed" || len(root.Shell.Toasts) == 0 {
		t.Fatalf("expected list error state and toast, got %#v", root)
	}
	staleSeq := root.TerminalPool.RequestSeq
	root.TerminalPool = root.TerminalPool.RequestList()
	root, _ = reducer(root, TerminalPoolListResultMsg{Seq: staleSeq, Result: port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "stale"}}}})
	if len(root.TerminalPool.Items) != 0 {
		t.Fatalf("stale result must not update pool, got %#v", root.TerminalPool)
	}

	terminal = &testkit.FakeTerminalService{CreateResult: port.TerminalCreateResult{TerminalID: "term-created", State: "running"}}
	reducer = newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root.Endpoints = (state.EndpointStore{}).
		Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "This Mac", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true}).
		ApplyDefaults(state.DefaultEndpointID, []string{"/bin/sh"}, "/Users/me", "")
	root, effects = reducer(root, TerminalPoolCreateRequestMsg{})
	if len(effects) != 1 {
		t.Fatalf("expected create effect, got %#v", effects)
	}
	createEffect, ok := effects[0].(FuncEffect)
	if !ok {
		t.Fatalf("expected create FuncEffect, got %#v", effects[0])
	}
	createMsg, ok := createEffect.Run(context.Background()).(TerminalPoolCreateResultMsg)
	if !ok {
		t.Fatalf("expected create result message, got %#v", createMsg)
	}
	if len(terminal.Creates) != 1 || len(terminal.Creates[0].Command) == 0 {
		t.Fatalf("terminal pool create must send a default shell command, creates=%#v", terminal.Creates)
	}
	root, effects = reducer(root, TerminalPoolCreateResultMsg{Result: port.TerminalCreateResult{TerminalID: "term-created", State: "running"}})
	if root.TerminalPool.LastCreatedID != "term-created" || len(root.Shell.Toasts) == 0 || root.Shell.Toasts[len(root.Shell.Toasts)-1].Body != "term-created" || len(effects) != 2 {
		t.Fatalf("expected create feedback plus refresh and attach effects, got root=%#v effects=%#v", root, effects)
	}
	attachEffect, ok := effects[1].(FuncEffect)
	if !ok {
		t.Fatalf("expected create attach FuncEffect, got %#v", effects[1])
	}
	attachMsg, ok := attachEffect.Run(context.Background()).(TerminalPoolAttachRequestMsg)
	if !ok || attachMsg.TerminalID != "term-created" {
		t.Fatalf("expected create to request attach for created terminal, got %#v", attachMsg)
	}
}

func TestTerminalSizeLockToggleUsesDaemonControlAndProjectsViews(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{
		Shell:        state.DefaultShell(),
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{TerminalID: "term-1", Title: "main", Tags: map[string]string{"role": "shell"}}}},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-1", 8, 80, 24, state.TerminalResizeRoleFollower, "surface", "view-2", false))
	ownerBinding, _ := root.TerminalViews.PaneBinding(state.DefaultPaneID)
	ownerBinding.OwnerSurfaceID = ownerBinding.SurfaceID
	ownerBinding.OwnerViewID = ownerBinding.ViewID
	root.TerminalViews.Views[ownerBinding.ViewID] = ownerBinding

	next, effects := reducer(root, TerminalSizeLockToggleRequestMsg{})
	if len(effects) != 1 {
		t.Fatalf("size lock should issue one daemon control request, got %#v", effects)
	}
	result, ok := effects[0].(FuncEffect).Run(context.Background()).(TerminalSizeLockToggleResultMsg)
	if !ok || result.TerminalID != "term-1" || !result.Locked {
		t.Fatalf("expected size lock result, got %#v", result)
	}
	if len(terminal.ResizeLocks) != 1 || !terminal.ResizeLocks[0].Locked || terminal.ResizeLocks[0].Channel != 7 || len(terminal.TagEdits) != 0 {
		t.Fatalf("size lock must use the attached owner control, locks=%#v tag_edits=%#v", terminal.ResizeLocks, terminal.TagEdits)
	}
	next, _ = reducer(next, result)
	owner, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	follower, _ := next.TerminalViews.PaneBinding("pane-2")
	if !owner.SizeLocked || owner.CanResize || owner.ControlReason != "size_locked" || owner.ResizeRole != state.TerminalResizeRoleOwner {
		t.Fatalf("owner view should project locked owner without resize rights, got %#v", owner)
	}
	if !follower.SizeLocked || follower.CanResize || follower.ControlReason != "size_locked" {
		t.Fatalf("follower view should project terminal lock, got %#v", follower)
	}
	if next.TerminalPool.Items[0].Tags["role"] != "shell" || terminalmeta.SizeLocked(next.TerminalPool.Items[0].Tags) {
		t.Fatalf("permission state must not be persisted as a metadata tag, pool=%#v", next.TerminalPool)
	}

	next, effects = reducer(next, TerminalSizeLockToggleRequestMsg{})
	result = effects[0].(FuncEffect).Run(context.Background()).(TerminalSizeLockToggleResultMsg)
	if result.Locked {
		t.Fatalf("unlock should report unlocked result, got %#v", result)
	}
	if len(terminal.ResizeLocks) != 2 || terminal.ResizeLocks[1].Locked {
		t.Fatalf("unlock should use daemon control, locks=%#v", terminal.ResizeLocks)
	}
	next, _ = reducer(next, result)
	owner, _ = next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if owner.SizeLocked || !owner.CanResize || owner.ControlReason != "owner" {
		t.Fatalf("unlock should restore owner resize rights, got %#v", owner)
	}
}

func TestTerminalSizeLockToggleDoesNotDependOnTerminalPoolTags(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-1", Title: "main", Tags: map[string]string{"role": "shell"}}}},
	}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{Shell: state.DefaultShell()}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	ownerBinding, _ := root.TerminalViews.PaneBinding(state.DefaultPaneID)
	ownerBinding.OwnerSurfaceID = ownerBinding.SurfaceID
	ownerBinding.OwnerViewID = ownerBinding.ViewID
	root.TerminalViews.Views[ownerBinding.ViewID] = ownerBinding

	next, effects := reducer(root, TerminalSizeLockToggleRequestMsg{})
	if len(terminal.ResizeLocks) != 0 || len(effects) != 1 || len(next.Shell.Toasts) != 0 {
		t.Fatalf("size lock should defer one daemon request without warning, locks=%#v effects=%#v root=%#v", terminal.ResizeLocks, effects, next)
	}
	result, ok := effects[0].(FuncEffect).Run(context.Background()).(TerminalSizeLockToggleResultMsg)
	if !ok || result.TerminalID != "term-1" || !result.Locked {
		t.Fatalf("daemon size lock should not require pool metadata, got %#v", result)
	}
	if len(terminal.Lists) != 0 || len(terminal.TagEdits) != 0 || len(terminal.ResizeLocks) != 1 || !terminal.ResizeLocks[0].Locked {
		t.Fatalf("size lock must call daemon control directly, lists=%#v edits=%#v locks=%#v", terminal.Lists, terminal.TagEdits, terminal.ResizeLocks)
	}
	next, _ = reducer(next, result)
	binding, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !binding.SizeLocked || binding.CanResize {
		t.Fatalf("listed tag result should project terminal size lock, got %#v", binding)
	}
}

func TestTerminalSizeLockToggleRequiresOwnerIdentity(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{
		Shell:        state.DefaultShell(),
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{TerminalID: "term-1", Title: "main", Tags: map[string]string{"role": "shell"}}}},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleFollower, "surface", "view-1", false))

	next, effects := reducer(root, TerminalSizeLockToggleRequestMsg{})
	if len(terminal.ResizeLocks) != 0 || len(effects) != 0 {
		t.Fatalf("follower must not change daemon size lock, locks=%#v effects=%#v", terminal.ResizeLocks, effects)
	}
	if len(next.Shell.Toasts) == 0 || next.Shell.Toasts[len(next.Shell.Toasts)-1].Body != "no active terminal" {
		t.Fatalf("follower size lock should warn, root=%#v", next)
	}
}

func TestTerminalPoolReducerHandlesRestartAndReconnectResults(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 3, Cols: 80, Rows: 24},
	}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{Shell: state.DefaultShell()}
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1")
	root.Session = root.Session.AttachWithResizeOwner("term-1", 9, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 9, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{TerminalID: "term-1", Cols: 80, Rows: 24, Lines: []string{"old live tail"}})
	root.Surface = root.Surface.MarkExitedWithMetadata("term-1", 23, "exited", time.Date(2026, 6, 17, 12, 30, 0, 0, time.UTC), []string{"bash", "-lc", "exit 23"})
	code := 23
	root.TerminalPool = state.TerminalPoolStore{Items: []state.TerminalPoolItem{{TerminalID: "term-1", State: string(state.TerminalLiveExited), ExitCode: &code, ExitedAt: time.Date(2026, 6, 17, 12, 30, 0, 0, time.UTC), Command: []string{"bash", "-lc", "exit 23"}}}}

	root, effects := reducer(root, TerminalPoolRestartRequestMsg{TerminalID: "term-1"})
	if len(effects) != 1 {
		t.Fatalf("expected restart effect, got %#v", effects)
	}
	restartEffect, ok := effects[0].(FuncEffect)
	if !ok {
		t.Fatalf("expected restart FuncEffect, got %#v", effects[0])
	}
	restartMsg, ok := restartEffect.Run(context.Background()).(TerminalPoolRestartResultMsg)
	if !ok || restartMsg.TerminalID != "term-1" || len(terminal.Restarts) != 1 {
		t.Fatalf("expected restart service result, msg=%#v restarts=%#v", restartMsg, terminal.Restarts)
	}
	root, effects = reducer(root, restartMsg)
	if len(root.Shell.Toasts) == 0 || root.Shell.Toasts[len(root.Shell.Toasts)-1].Title != "picker.restart" || len(effects) != 2 {
		t.Fatalf("expected restart result feedback and refresh effect, terminal=%#v root=%#v effects=%#v", terminal, root, effects)
	}
	if root.Surface.State != state.TerminalLiveAttached || root.Surface.ExitCode != 0 || !root.Surface.ExitedAt.IsZero() || len(root.Surface.Command) != 0 {
		t.Fatalf("restart should clear exited surface metadata, got %#v", root.Surface)
	}
	if root.TerminalPool.Items[0].State != string(state.TerminalLiveExited) || root.TerminalPool.Items[0].ExitCode == nil {
		t.Fatalf("restart ack must not forge pool running state before next core list, got %#v", root.TerminalPool.Items[0])
	}
	if len(root.Surface.Lines) != 1 || root.Surface.Lines[0] != "old live tail" || !root.Surface.Ready {
		t.Fatalf("restart should preserve live tail while reattaching, got %#v", root.Surface)
	}
	frame := render.NewRenderer(render.DefaultTheme()).Render(render.NewRenderVMBuilder().Build(root))
	if frame.Cursor.Visible {
		t.Fatalf("restart preserved tail must not synthesize a cursor at the old line end, cursor=%#v rect=%#v", frame.Cursor, frame.CursorRect)
	}
	if binding, ok := root.TerminalViews.PaneBinding(state.DefaultPaneID); !ok || binding.Channel != 0 || binding.Attached {
		t.Fatalf("restart should preserve binding intent but clear stale input channel, got %#v ok=%v", binding, ok)
	}
	if root.Session.Channel != 0 || root.Session.Attached {
		t.Fatalf("restart should clear session input channel before reattach, got %#v", root.Session)
	}
	if _, ok := effects[0].(FuncEffect).Run(context.Background()).(TerminalPoolListRequestMsg); !ok {
		t.Fatalf("first restart result effect should refresh pool, got %#v", effects[0])
	}
	if attachMsg, ok := effects[1].(FuncEffect).Run(context.Background()).(LiveAttachMsg); !ok || attachMsg.Config.TerminalID != "term-1" || attachMsg.Config.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) {
		t.Fatalf("restart should reattach current terminal view, got %#v ok=%v", attachMsg, ok)
	}

	root, effects = reducer(root, TerminalPoolReconnectRequestMsg{TerminalID: "term-1"})
	if len(effects) != 1 {
		t.Fatalf("expected reconnect effect, got %#v", effects)
	}
	if binding, ok := root.TerminalViews.PaneBinding(state.DefaultPaneID); !ok || !binding.AttachPending || binding.Attached || binding.Channel != 0 {
		t.Fatalf("reconnect request should immediately project connecting state, binding=%#v ok=%v", binding, ok)
	}
	if endpoint, ok := root.Endpoints.Endpoint(state.DefaultEndpointID); !ok || endpoint.DisplayStatus() != state.EndpointStatusConnecting {
		t.Fatalf("reconnect request should mark only owning endpoint connecting, endpoint=%#v ok=%v", endpoint, ok)
	}
	reconnectEffect, ok := effects[0].(FuncEffect)
	if !ok {
		t.Fatalf("expected reconnect FuncEffect, got %#v", effects[0])
	}
	reconnectMsg, ok := reconnectEffect.Run(context.Background()).(TerminalPoolReconnectResultMsg)
	if !ok || reconnectMsg.TerminalID != "term-1" || len(terminal.Reconnects) != 1 {
		t.Fatalf("expected reconnect service result, msg=%#v reconnects=%#v", reconnectMsg, terminal.Reconnects)
	}
	root, _ = reducer(root, reconnectMsg)
	if !root.Session.Attached || root.Session.TerminalID != "term-1" || root.TerminalPool.LastAttachedID != "term-1" {
		t.Fatalf("expected reconnect result to attach session, got session=%#v pool=%#v", root.Session, root.TerminalPool)
	}
	if endpoint, ok := root.Endpoints.Endpoint(state.DefaultEndpointID); !ok || endpoint.DisplayStatus() != state.EndpointStatusConnected {
		t.Fatalf("reconnect success should close connecting state, endpoint=%#v ok=%v", endpoint, ok)
	}
}

func TestTerminalPoolRestartResultPreventsStaleExitedPoolFromPoisoningReattach(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
		ListResult:   port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-1", State: "running", Command: []string{"/bin/zsh"}, Cols: 80, Rows: 24}}},
	}
	host := NewFakeTerminalHost(16)
	runtime := NewInteractiveRuntime(state.Root{}, host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal}, CopyModeDeps{})
	root := runtime.State()
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1")
	root.Session = root.Session.AttachWithResizeOwner("term-1", 9, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 9, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 2, Cols: 80, Rows: 24, Lines: []string{"terminal exited: term-1 code:0 exited"}})
	root.Surface = root.Surface.MarkExitedWithMetadata("term-1", 0, "exited", time.Date(2026, 6, 17, 12, 45, 0, 0, time.UTC), []string{"/bin/zsh"})
	code := 0
	root.TerminalPool = state.TerminalPoolStore{Items: []state.TerminalPoolItem{{TerminalID: "term-1", State: string(state.TerminalLiveExited), ExitCode: &code, ExitedAt: time.Date(2026, 6, 17, 12, 45, 0, 0, time.UTC), Command: []string{"/bin/zsh"}}}}
	runtime.state = root

	if err := runtime.Post(TerminalPoolRestartResultMsg{TerminalID: "term-1"}); err != nil {
		t.Fatalf("post restart result: %v", err)
	}
	if err := postPreparedTerminalPoolAttachResult(runtime, TerminalPoolAttachResultMsg{
		TerminalID:   "term-1",
		TargetPaneID: state.DefaultPaneID,
		ResizePolicy: state.TerminalResizeRoleOwner,
		Result:       port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24, ResizePolicy: state.TerminalResizeRoleOwner, SurfaceID: "surface", ViewID: state.TerminalPaneViewID(state.DefaultPaneID), CanResize: true},
	}); err != nil {
		t.Fatalf("post reattach result: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   3,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"terminal exited: term-1 code:0 exited", "% "},
		Cursor:     state.LiveCursor{Visible: true, Row: 1, Col: 2, Shape: "bar"},
	}}); err != nil {
		t.Fatalf("post live surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain restart reattach: %v", err)
	}

	final := runtime.State()
	if final.Surface.State != state.TerminalLiveAttached || final.Surface.ExitReason != "" || final.Surface.Cursor != (state.LiveCursor{Visible: true, Row: 1, Col: 2, Shape: "bar"}) {
		t.Fatalf("restart reattach should accept running live surface and cursor, surface=%#v", final.Surface)
	}
	if final.TerminalPool.Items[0].State != "running" || final.TerminalPool.Items[0].ExitCode != nil {
		t.Fatalf("restart should keep pool item running until refreshed list arrives, pool=%#v", final.TerminalPool)
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "% ") || frameContains(frame, "restart") || !frame.Cursor.Visible || frame.Cursor.Shape != render.CursorShapeBar {
		t.Fatalf("frame should show restarted live prompt with visible cursor, lines=%#v cursor=%#v", frame.Lines, frame.Cursor)
	}
}

func TestRestartIfExitedSkipsRunningCoreTerminal(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{
			TerminalID: "term-1",
			Title:      "main",
			State:      "running",
			Command:    []string{"/bin/zsh"},
			Cols:       80,
			Rows:       24,
		}}},
	}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{Shell: state.DefaultShell()}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))
	root.Session = root.Session.AttachWithResizeOwner("term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID))
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{TerminalID: "term-1", State: state.TerminalLiveAttached, Cols: 80, Rows: 24})

	root, effects := reducer(root, TerminalPoolRestartIfExitedRequestMsg{TerminalID: "term-1"})
	if len(effects) != 1 {
		t.Fatalf("restart-if-exited request should list core lifecycle, effects=%#v", effects)
	}
	resultMsg, ok := effects[0].(FuncEffect).Run(context.Background()).(TerminalPoolRestartIfExitedResultMsg)
	if !ok {
		t.Fatalf("expected restart-if-exited result msg, got %#v", effects[0])
	}
	root, effects = reducer(root, resultMsg)
	if len(effects) != 0 || len(terminal.Restarts) != 0 {
		t.Fatalf("running core terminal must not be restarted, effects=%#v restarts=%#v", effects, terminal.Restarts)
	}
	if root.Surface.State != state.TerminalLiveAttached {
		t.Fatalf("running core lifecycle should keep surface attached, surface=%#v", root.Surface)
	}
}

func TestTerminalPoolRunningListDoesNotCacheLifecycleBeforeSurface(t *testing.T) {
	host := NewFakeTerminalHost(16)
	runtime := NewInteractiveRuntime(state.Root{}, host, NewSyncEffectRunner(), LiveDeps{Terminal: &testkit.FakeTerminalService{}}, CopyModeDeps{})
	root := runtime.State()
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1")
	root.Session = root.Session.AttachWithResizeOwner("term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID))
	root.Session = root.Session.MarkExitedWithMetadata("term-1", 0, "exited", time.Date(2026, 6, 17, 12, 50, 0, 0, time.UTC), []string{"/bin/zsh"})
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 2, Cols: 80, Rows: 24, Lines: []string{"terminal exited: term-1 code:0 exited"}})
	root.Surface = root.Surface.MarkExitedWithMetadata("term-1", 0, "exited", time.Date(2026, 6, 17, 12, 50, 0, 0, time.UTC), []string{"/bin/zsh"})
	root.TerminalPool = root.TerminalPool.RequestList()
	seq := root.TerminalPool.RequestSeq
	runtime.state = root

	if err := runtime.Post(TerminalPoolListResultMsg{Seq: seq, Result: port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-1", State: "running", Command: []string{"/bin/zsh"}, Cols: 80, Rows: 24}}}}); err != nil {
		t.Fatalf("post running list: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain running list: %v", err)
	}
	afterList := runtime.State()
	if afterList.Surface.State != state.TerminalLiveExited || afterList.Session.State != state.TerminalLiveExited {
		t.Fatalf("running list must not mutate live surface/session lifecycle, surface=%#v session=%#v", afterList.Surface, afterList.Session)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   3,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"terminal exited: term-1 code:0 exited", "% "},
		Cursor:     state.LiveCursor{Visible: true, Row: 1, Col: 2, Shape: "bar"},
		State:      state.TerminalLiveAttached,
	}, LifecycleKnown: true}); err != nil {
		t.Fatalf("post live surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain running surface: %v", err)
	}

	final := runtime.State()
	if final.Surface.State != state.TerminalLiveAttached || final.Surface.ExitReason != "" || final.Session.State == state.TerminalLiveExited {
		t.Fatalf("running live surface should clear exited lifecycle after list-only cache, surface=%#v session=%#v", final.Surface, final.Session)
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "% ") || frameContains(frame, "restart") || !frame.Cursor.Visible {
		t.Fatalf("running live surface should unblock fresh live prompt and cursor, lines=%#v cursor=%#v", frame.Lines, frame.Cursor)
	}
}

func TestTerminalPoolRestartReattachesEachViewWithoutReusingExitedChannels(t *testing.T) {
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: &testkit.FakeTerminalService{}})
	root := state.Root{Shell: state.DefaultShell()}
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1")
	var result state.FloatingCommandResult
	root.Shell, result = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane", Title: "floating", Kind: state.PaneEmpty},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating failed: %#v", result)
	}
	root.Shell = root.Shell.BindFloatingTerminal("floating-1", "term-1")
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))
	root.TerminalViews = root.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "floating-pane", "term-1", 8, 40, 12, state.TerminalResizeRoleFollower, "surface", state.TerminalFloatingViewID("floating-1"), false))
	root.Session = root.Session.AttachWithResizeOwner("term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID))
	root.Session = root.Session.MarkExitedWithMetadata("term-1", 23, "exited", time.Date(2026, 6, 17, 12, 35, 0, 0, time.UTC), []string{"bash", "-lc", "exit 23"})

	next, effects := reducer(root, TerminalPoolRestartResultMsg{TerminalID: "term-1"})
	if len(effects) != 3 {
		t.Fatalf("restart should refresh pool and reattach both views, got %#v", effects)
	}
	pane, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	floating, _ := next.TerminalViews.FloatingBinding("floating-1")
	if pane.Channel != 0 || pane.Attached || floating.Channel != 0 || floating.Attached {
		t.Fatalf("restart must clear stale channels for all views, pane=%#v floating=%#v", pane, floating)
	}
	if next.Session.State != state.TerminalLivePending || next.Session.Channel != 0 || next.Session.Attached || !next.Session.ExitedAt.IsZero() || len(next.Session.Command) != 0 {
		t.Fatalf("restart should clear session exit metadata while waiting for attach, got %#v", next.Session)
	}
	seen := map[string]bool{}
	for _, effect := range effects[1:] {
		attachMsg, ok := effect.(FuncEffect).Run(context.Background()).(LiveAttachMsg)
		if !ok {
			t.Fatalf("restart view effect should produce LiveAttachMsg, got %#v", effect)
		}
		seen[attachMsg.Config.ViewID] = true
	}
	if !seen[state.TerminalPaneViewID(state.DefaultPaneID)] || !seen[state.TerminalFloatingViewID("floating-1")] {
		t.Fatalf("restart should reattach original pane and floating views, seen=%#v", seen)
	}
}

func TestInteractiveRuntimeTerminalPoolPageFlow(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-logs", Channel: 9, Cols: 100, Rows: 30},
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{
			TerminalID: "term-shell",
			Title:      "shell",
			State:      "running",
			Cols:       80,
			Rows:       24,
		}, {
			TerminalID: "term-logs",
			Title:      "日志🚀",
			State:      "running",
			CWD:        "/tmp/logs",
			Cols:       100,
			Rows:       30,
			Tags:       map[string]string{"role": "logs"},
		}}},
	}
	host := NewFakeTerminalHost(64)
	host.SetSize(96, 28)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x07", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "p"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "日"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "志"},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send pool input %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain pool input %#v: %v", event, err)
		}
	}
	if len(terminal.Lists) != 1 || runtime.State().Shell.Overlay.Kind != state.OverlayTerminalPool || runtime.State().Shell.Overlay.Query != "日志" {
		t.Fatalf("expected pool page loaded and queried, lists=%#v shell=%#v", terminal.Lists, runtime.State().Shell)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("pool query must not leak terminal input, got %#v", terminal.Inputs)
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "Terminal Manager") || !frameContains(frame, "▸ ● 日志🚀") || !frameContains(frame, "TERMINAL") || frameContains(frame, "HIST") || frameContains(frame, "CONN") {
		t.Fatalf("expected terminal pool page frame, got %#v", frame.Lines)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}); err != nil {
		t.Fatalf("send pool enter attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pool attach: %v", err)
	}
	if len(terminal.Attaches) != 1 || terminal.Attaches[0].TerminalID != "term-logs" || !runtime.State().Session.Attached {
		t.Fatalf("expected pool attach service result, attaches=%#v session=%#v", terminal.Attaches, runtime.State().Session)
	}

	if err := runtime.Post(ShellOpenTerminalPoolMsg{}); err != nil {
		t.Fatalf("post pool open: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pool reopen: %v", err)
	}
	frame = lastFrame(t, host.Frames())
	selectRegion := frameActionHitRegion(t, frame, "pool.select", "")
	if err := host.SendInput(mouseEventAt(selectRegion.Rect)); err != nil {
		t.Fatalf("send pool select click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pool select: %v", err)
	}
	if runtime.State().Shell.Overlay.SelectedIndex != 0 {
		t.Fatalf("expected row click to select first row, got %#v", runtime.State().Shell.Overlay)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x05", Ctrl: true}); err != nil {
		t.Fatalf("send pool edit shortcut: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pool edit: %v", err)
	}
	if runtime.State().Shell.Overlay.Kind != state.OverlayPrompt || runtime.State().Shell.Overlay.Prompt.Purpose != "terminal.rename" {
		t.Fatalf("expected pool edit to open rename prompt, shell=%#v", runtime.State().Shell)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}); err != nil {
		t.Fatalf("send pool rename submit: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pool rename submit: %v", err)
	}
	if len(terminal.Edits) != 1 || terminal.Edits[0].TerminalID != "term-shell" {
		t.Fatalf("expected edit metadata service result, edits=%#v", terminal.Edits)
	}
	if terminal.Edits[0].Title != "shell" {
		t.Fatalf("expected rename prompt to submit selected title, edits=%#v", terminal.Edits)
	}
	if runtime.State().Shell.Overlay.Kind != state.OverlayTerminalPool {
		t.Fatalf("terminal rename submit should return to terminal manager, shell=%#v", runtime.State().Shell)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x0b", Ctrl: true}); err != nil {
		t.Fatalf("send pool kill shortcut: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pool kill: %v", err)
	}
	if len(terminal.Kills) != 1 || terminal.Kills[0].TerminalID != "term-shell" || runtime.State().TerminalPool.LastKilledID != "term-shell" {
		t.Fatalf("expected kill service result without local lifecycle spoofing, kills=%#v pool=%#v", terminal.Kills, runtime.State().TerminalPool)
	}
}

func TestInteractiveRuntimeWorkbenchTreeOverlayFlow(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-main", Channel: 4, Cols: 80, Rows: 24},
	}
	initialShell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "日志🚀", Kind: state.PaneTerminalLive, TerminalID: "term-2"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	host := NewFakeTerminalHost(32)
	host.SetSize(96, 28)
	runtime := NewInteractiveRuntime(
		state.Root{Shell: initialShell},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-main", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}

	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x07", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "w"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "日"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "志"},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send tree input %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain tree input %#v: %v", event, err)
		}
	}
	if runtime.State().Shell.Overlay.Kind != state.OverlayWorkbenchTree || runtime.State().Shell.Overlay.Query != "日志" {
		t.Fatalf("expected workbench tree queried, shell=%#v", runtime.State().Shell)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("tree query must not leak terminal input, got %#v", terminal.Inputs)
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "Workbench Navigator") || !frameContains(frame, "WORKBENCH") || !frameContains(frame, "term-2") || frameContains(frame, "DETAIL") || frameContains(frame, "VIEWS") {
		t.Fatalf("expected workbench navigator frame, got %#v", frame.Lines)
	}
	if frameContains(frame, "Open  New  Zoom  Detach  Close") {
		t.Fatalf("workbench navigator content should not render footer actions inline, got %#v", frame.Lines)
	}
	if frameContains(frame, "● open") || frameContains(frame, " esc ") {
		t.Fatalf("workbench navigator title should not render generic open/esc chrome, got %#v", frame.Lines)
	}
	if frame.Cursor.Shape != render.CursorShapeBar {
		t.Fatalf("tree overlay should own search cursor, got %#v", frame.Cursor)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}); err != nil {
		t.Fatalf("send tree enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tree enter: %v", err)
	}
	if runtime.State().Shell.ActivePaneID != "pane-2" || runtime.State().Shell.Overlay.Open {
		t.Fatalf("tree enter should focus pane and close overlay, got %#v", runtime.State().Shell)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("tree enter must not leak terminal input, got %#v", terminal.Inputs)
	}

	if err := runtime.Post(ShellOpenWorkbenchTreeMsg{}); err != nil {
		t.Fatalf("post tree open: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tree open: %v", err)
	}
	frame = lastFrame(t, host.Frames())
	detailRegion := frameWorkbenchDetailOpenHitRegion(t, frame)
	if err := host.SendInput(mouseEventAt(detailRegion.Rect)); err != nil {
		t.Fatalf("send tree detail open click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tree detail open click: %v", err)
	}
	if runtime.State().Shell.ActivePaneID != "pane-2" || runtime.State().Shell.Overlay.Open {
		t.Fatalf("tree detail click should focus selected node and close overlay, got %#v", runtime.State().Shell)
	}

	if err := runtime.Post(ShellOpenWorkbenchTreeMsg{}); err != nil {
		t.Fatalf("post tree reopen after detail click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tree reopen after detail click: %v", err)
	}
	frame = lastFrame(t, host.Frames())
	openRegion := frameActionHitRegion(t, frame, "workbench.open", "")
	if openRegion.Row < 0 {
		t.Fatalf("expected first workbench open hit region to be a tree row, got %#v", openRegion)
	}
	if err := host.SendInput(mouseEventAt(openRegion.Rect)); err != nil {
		t.Fatalf("send tree row open click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tree row open click: %v", err)
	}
	if runtime.State().Shell.Overlay.Open {
		t.Fatalf("tree row click should open node and close overlay, got %#v", runtime.State().Shell.Overlay)
	}
	if len(runtime.State().Shell.Toasts) == 0 || runtime.State().Shell.Toasts[len(runtime.State().Shell.Toasts)-1].Title != "workbench.open" {
		t.Fatalf("expected workbench row open feedback toast, got %#v", runtime.State().Shell.Toasts)
	}
}

func TestInteractiveRuntimePromptAndHelpOverlayFlow(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-main", Channel: 4, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(48)
	host.SetSize(90, 26)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-main", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x07", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: ":"},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send prompt input %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain prompt input %#v: %v", event, err)
		}
	}
	for _, char := range "system.clear_toasts" {
		if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: string(char)}); err != nil {
			t.Fatalf("send prompt command char %q: %v", char, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain prompt command char %q: %v", char, err)
		}
	}
	if runtime.State().Shell.Overlay.Kind != state.OverlayPrompt || runtime.State().Shell.Overlay.Prompt.Value != "system.clear_toasts" {
		t.Fatalf("expected prompt input captured, shell=%#v", runtime.State().Shell)
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "Command Prompt") || !frameContains(frame, "NAME system.clear_toasts") || frame.Cursor.Shape != render.CursorShapeBar {
		t.Fatalf("expected prompt frame and cursor, got %#v", frame)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("prompt input must not leak to terminal, got %#v", terminal.Inputs)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x7f"}); err != nil {
		t.Fatalf("send prompt backspace: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "s"}); err != nil {
		t.Fatalf("restore prompt command suffix: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}); err != nil {
		t.Fatalf("send prompt enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain prompt submit: %v", err)
	}
	if runtime.State().Shell.Overlay.Open || len(runtime.State().Shell.Toasts) != 0 {
		t.Fatalf("expected prompt submit to execute clear-toasts action, shell=%#v", runtime.State().Shell)
	}

	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x07", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "?"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send help input %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain help input %#v: %v", event, err)
		}
	}
	if runtime.State().Shell.Overlay.Kind != state.OverlayHelp {
		t.Fatalf("expected help overlay, shell=%#v", runtime.State().Shell)
	}
	frame = lastFrame(t, host.Frames())
	if !frameContains(frame, "Help") || !frameContains(frame, "shortcuts 1/") || !frameContains(frame, "Most used") || !frameContains(frame, "[Ctrl+P]") || !frameContains(frame, "[Ctrl+R]") || frameContains(frame, "[Ctrl] •") {
		t.Fatalf("expected help content, got %#v", frame.Lines)
	}
	closeRegion := frameActionHitRegion(t, frame, "help.close", "")
	if err := host.SendInput(mouseEventAt(closeRegion.Rect)); err != nil {
		t.Fatalf("send help close click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain help close click: %v", err)
	}
	if runtime.State().Shell.Overlay.Open {
		t.Fatalf("help close action should close overlay, got %#v", runtime.State().Shell.Overlay)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("help input must not leak to terminal, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeCtrlVEntersCopyWithoutTerminalInput(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			1,
			nil,
		)}},
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: core, Rows: 20},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true}); err != nil {
		t.Fatalf("send ctrl-v: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain ctrl-v: %v", err)
	}

	if !runtime.State().CopyMode.Active {
		t.Fatalf("expected copy mode active, got %#v", runtime.State().CopyMode)
	}
	if len(core.LatestRequests) != 1 {
		t.Fatalf("expected authoritative latest request, got %#v", core.LatestRequests)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("ctrl-v must not be sent to terminal, got %#v", terminal.Inputs)
	}
	last := lastFrame(t, host.Frames())
	if !frameContains(last, "copy history empty") {
		t.Fatalf("expected copy empty content in frame, got %#v", last.Lines)
	}
}

func TestInteractiveRuntimeCtrlVPreemptsQueuedLiveRefresh(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
		SurfaceResult: port.TerminalSurfaceResult{
			Ready: true,
			Snapshot: state.LiveSurfaceSnapshot{
				TerminalID: "term-1",
				Revision:   2,
				Cols:       80,
				Rows:       24,
				Lines:      []string{"stress output still moving"},
			},
		},
	}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			1,
			[]state.HistoryRow{{Text: "latest before live backlog", LineID: 1}},
		)}},
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: core, Rows: 20},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}

	surfacesBeforePressure := len(terminal.Surfaces)
	runtime.maxMessagesPerBatch = 1
	if err := runtime.Post(LiveEventMsg{Event: port.TerminalLiveEvent{TerminalID: "term-1", Refresh: true}}); err != nil {
		t.Fatalf("post live refresh: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true}); err != nil {
		t.Fatalf("send ctrl-v: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain first pressure batch: %v", err)
	}

	if !runtime.State().CopyMode.Entering || runtime.State().CopyMode.Active {
		t.Fatalf("ctrl-v should enter copy before queued live refresh is processed, got %#v", runtime.State().CopyMode)
	}
	if len(core.LatestRequests) != 1 {
		t.Fatalf("ctrl-v should start authoritative latest request immediately, got %#v", core.LatestRequests)
	}
	if len(terminal.Surfaces) != surfacesBeforePressure {
		t.Fatalf("queued live refresh should not run before ctrl-v in first pressure batch, got before=%d after=%#v", surfacesBeforePressure, terminal.Surfaces)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("ctrl-v must not leak to terminal while live refresh is queued, got %#v", terminal.Inputs)
	}

	runtime.maxMessagesPerBatch = 0
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain remaining pressure queue: %v", err)
	}
	if !runtime.State().CopyMode.Active {
		t.Fatalf("expected copy mode active after latest result, got %#v", runtime.State().CopyMode)
	}
}

func TestInteractiveRuntimeCtrlVRendersLatestStressTail(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 120, Rows: 30},
	}
	rows := make([]state.HistoryRow, 0, 100)
	for i := 1; i <= 100; i++ {
		text := stressHistoryLineForTUI(i)
		rows = append(rows, state.HistoryRow{
			Text:   text,
			LineID: uint64(i),
			Cells:  []state.HistoryCell{{Text: text, Width: len(text)}},
		})
	}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-stress",
			118,
			1,
			rows,
		)}},
	}
	host := NewFakeTerminalHost(64)
	host.SetSize(120, 30)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: core, Rows: 20},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 120, Rows: 30}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true}); err != nil {
		t.Fatalf("send ctrl-v: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain ctrl-v: %v", err)
	}

	if !runtime.State().CopyMode.Active {
		t.Fatalf("expected copy mode active, got %#v", runtime.State().CopyMode)
	}
	last := lastFrame(t, host.Frames())
	if !frameContains(last, "000100") || frameContains(last, "ALT_SCREEN_MARK") {
		t.Fatalf("copy latest should render newest primary stress tail only, got %#v", last.Lines)
	}
}

func stressHistoryLineForTUI(n int) string {
	return fmt.Sprintf("%06d [DEBUG ] stream pending path=/var/tmp/alpha/beta/gamma wrap======================== tail-marker", n)
}

func TestInteractiveRuntimeShellSemanticActionsReachRenderPath(t *testing.T) {
	host := NewFakeTerminalHost(8)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{}},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)

	for _, msg := range []Msg{
		ShellSetPanelPresentationMsg{Presentation: state.PanelPresentationSplitLine},
		ShellSetHeaderVisibleMsg{Visible: false},
		ShellSetFooterVisibleMsg{Visible: false},
		ShellAddToastMsg{Toast: state.ToastSpec{ID: "toast-1", Severity: state.ToastWarning, Title: "warn"}},
		ShellCloseCurrentToastMsg{},
		ShellAddToastMsg{Toast: state.ToastSpec{ID: "toast-2", Severity: state.ToastInfo, Title: "notice"}},
		ShellClearToastsMsg{},
	} {
		if err := runtime.Post(msg); err != nil {
			t.Fatalf("post %T: %v", msg, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if runtime.State().Shell.PanelPresentation != state.PanelPresentationSplitLine {
		t.Fatalf("expected split line presentation, got %#v", runtime.State().Shell)
	}
	if runtime.State().Shell.HeaderVisible || runtime.State().Shell.FooterVisible {
		t.Fatalf("expected hidden header/footer, got %#v", runtime.State().Shell)
	}
	if len(runtime.State().Shell.Toasts) != 0 {
		t.Fatalf("expected cleared toasts, got %#v", runtime.State().Shell.Toasts)
	}
	last := lastFrame(t, host.Frames())
	if frameContains(last, " main ") || frameContains(last, " live ") {
		t.Fatalf("hidden header/footer should not render shell bars, got %#v", last.Lines)
	}
}

func TestInteractiveRuntimePaneAndResizeModeShortcutsUsePaneCommandPath(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action:         state.PaneCommandSplit,
		Target:         state.PaneCommandTarget{PaneID: state.DefaultPaneID},
		SplitDirection: state.SplitDirectionVertical,
		NewPane:        state.PaneState{ID: "pane-2", Title: "pane", Kind: state.PaneEmpty},
		Source:         state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post semantic pane split: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain semantic pane split: %v", err)
	}
	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x10", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "n"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x12", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyRight},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send input %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain input %#v: %v", event, err)
		}
	}

	if runtime.State().Shell.InteractionMode != state.InteractionModeResize {
		t.Fatalf("expected resize mode, got %#v", runtime.State().Shell.InteractionMode)
	}
	tab := runtime.State().Shell.Workspace.Tabs[0]
	if len(tab.Panes) != 2 || tab.RootSplit.Direction != state.SplitDirectionVertical {
		t.Fatalf("expected semantic split through pane command path, got %#v", tab)
	}
	if tab.RootSplit.BiasCells == 0 {
		t.Fatalf("expected resize key to update split bias, got %#v", tab.RootSplit)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("pane/resize shortcuts must not leak to terminal input, got %#v", terminal.Inputs)
	}
	last := lastFrame(t, host.Frames())
	if !frameContains(last, "RESIZE") {
		t.Fatalf("footer should show resize mode, got %#v", last.Lines)
	}
}

func TestInteractiveRuntimeActivePaneVisualFeedbackFollowsKeyboardAndMouse(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(32)
	host.SetSize(96, 28)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}

	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action:         state.PaneCommandSplit,
		Target:         state.PaneCommandTarget{PaneID: state.DefaultPaneID},
		SplitDirection: state.SplitDirectionVertical,
		NewPane:        state.PaneState{ID: "pane-2", Title: "pane", Kind: state.PaneEmpty},
		Source:         state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post semantic pane split: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain semantic pane split: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x10", Ctrl: true}); err != nil {
		t.Fatalf("send pane mode after semantic split: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane mode after semantic split: %v", err)
	}
	keyboardFrame := lastFrame(t, host.Frames())
	if runtime.State().Shell.EnsureDefaults().ActivePaneID != "pane-2" {
		t.Fatalf("semantic split should activate pane-2, got %#v", runtime.State().Shell)
	}
	assertPaneVisualState(t, keyboardFrame, "pane", render.StyleAccent)
	assertPaneVisualState(t, keyboardFrame, "shell", render.StyleMuted)
	if !frameContains(keyboardFrame, "PANE") || !frameContains(keyboardFrame, "CLOSE") || !frameContains(keyboardFrame, "OWNER") {
		t.Fatalf("footer should reflect the canonical pane action catalog within available width, got %#v", keyboardFrame.Lines)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "n"}); err != nil {
		t.Fatalf("send keyboard focus: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain keyboard focus: %v", err)
	}
	focusFrame := lastFrame(t, host.Frames())
	if runtime.State().Shell.EnsureDefaults().ActivePaneID != state.DefaultPaneID {
		t.Fatalf("keyboard focus-next should activate default pane, got %#v", runtime.State().Shell)
	}
	assertPaneVisualState(t, focusFrame, "shell", render.StyleAccent)
	assertPaneVisualState(t, focusFrame, "pane", render.StyleMuted)
	if frameContains(focusFrame, "pane.focus-next") || !frameContains(focusFrame, "PANE") {
		t.Fatalf("keyboard focus should update footer without low-value toast, got %#v", focusFrame.Lines)
	}

	paneContent := frameHitRegion(t, focusFrame, render.HitRegionPaneContent, "pane-2")
	if err := host.SendInput(mouseEventAt(paneContent.Rect)); err != nil {
		t.Fatalf("send mouse focus: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain mouse focus: %v", err)
	}
	mouseFrame := lastFrame(t, host.Frames())
	if runtime.State().Shell.EnsureDefaults().ActivePaneID != "pane-2" {
		t.Fatalf("mouse focus should activate pane-2, got %#v", runtime.State().Shell)
	}
	assertPaneVisualState(t, mouseFrame, "pane", render.StyleAccent)
	assertPaneVisualState(t, mouseFrame, "shell", render.StyleMuted)
	if frameContains(mouseFrame, "pane.focus") || !frameContains(mouseFrame, "PANE") {
		t.Fatalf("mouse focus should update footer without low-value toast, got %#v", mouseFrame.Lines)
	}

	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x12", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyRight},
		{Kind: input.EventKindKey, Key: input.KeyEsc},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x10", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "p"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x10", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "z"},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send post-focus event %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain post-focus event %#v: %v", event, err)
		}
	}
	zoomFrame := lastFrame(t, host.Frames())
	if runtime.State().Shell.ZoomedPaneID != "pane-2" {
		t.Fatalf("zoom should keep active pane zoomed, got %#v", runtime.State().Shell)
	}
	if runtime.State().Shell.EnsureDefaults().InteractionMode != state.InteractionModeNormal || len(runtime.State().Shell.Toasts) == 0 {
		t.Fatalf("resize/presentation/zoom should keep visible active feedback, got %#v", zoomFrame.Lines)
	}
	assertPaneVisualState(t, zoomFrame, "pane", render.StyleAccent)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x10", Ctrl: true}); err != nil {
		t.Fatalf("send pane mode before unzoom: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane mode before unzoom: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "z"}); err != nil {
		t.Fatalf("send unzoom before close: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain unzoom before close: %v", err)
	}
	unzoomFrame := lastFrame(t, host.Frames())
	if runtime.State().Shell.ZoomedPaneID != "" {
		t.Fatalf("unzoom before close should restore split layout, got %#v", runtime.State().Shell)
	}
	assertPaneVisualState(t, unzoomFrame, "pane", render.StyleAccent)

	if err := runtime.Post(ShellClearToastsMsg{}); err != nil {
		t.Fatalf("post clear toasts before close: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain clear toasts before close: %v", err)
	}
	clearFrame := lastFrame(t, host.Frames())
	action := frameHitRegionByAction(t, clearFrame, render.HitRegionPaneAction, "pane.close", "pane-2")
	if err := host.SendInput(mouseEventAt(action.Rect)); err != nil {
		t.Fatalf("send close click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain close click: %v", err)
	}
	closeFrame := lastFrame(t, host.Frames())
	if runtime.State().Shell.HasPane(state.PaneCommandTarget{PaneID: "pane-2"}) {
		t.Fatalf("mouse close should remove active pane, got %#v", runtime.State().Shell)
	}
	if runtime.State().Shell.EnsureDefaults().ActivePaneID != state.DefaultPaneID {
		t.Fatalf("close should choose stable next active pane, got %#v", runtime.State().Shell)
	}
	if !frameContains(closeFrame, "PANE") || len(runtime.State().Shell.Toasts) == 0 {
		t.Fatalf("close should update active pane visuals/footer/toast, got %#v", closeFrame.Lines)
	}
	assertPaneVisualState(t, closeFrame, "shell", render.StyleAccent)
}

func TestInteractiveRuntimeUIFrameworkProductizationFlow(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 68, Rows: 18},
	}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{
			{Window: historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-1", 33, 7, []state.HistoryRow{{Text: "copy-old", LineID: 20}})},
			{Window: historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-2", 32, 8, []state.HistoryRow{{Text: "copy-sized", LineID: 30}})},
		},
	}
	host := NewFakeTerminalHost(64)
	host.SetSize(70, 22)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: core, Rows: 20},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 70, Rows: 22, ResizePolicy: state.TerminalResizeRoleOwner, SurfaceID: "test-surface", ViewID: state.TerminalPaneViewID(state.DefaultPaneID)}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}

	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x10", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "c"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x10", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "p"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x10", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "b"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x12", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyLeft},
		{Kind: input.EventKindKey, Key: input.KeyEsc},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x07", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "h"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x07", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "f"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x07", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "T"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x07", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "c"},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send product flow input %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain product flow input %#v: %v", event, err)
		}
	}

	shell := runtime.State().Shell.EnsureDefaults()
	if shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("global esc should return to normal mode, got %#v", shell.InteractionMode)
	}
	if shell.HeaderVisible || shell.FooterVisible {
		t.Fatalf("global mode should hide header/footer, got %#v", shell)
	}
	if len(shell.Toasts) != 0 {
		t.Fatalf("global close/clear toast actions should clear toasts, got %#v", shell.Toasts)
	}
	if shell.PanelPresentation != state.PanelPresentationSplitLine {
		t.Fatalf("pane mode presentation switch should use split-line, got %#v", shell.PanelPresentation)
	}
	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action:         state.PaneCommandSplit,
		Target:         state.PaneCommandTarget{PaneID: state.DefaultPaneID},
		SplitDirection: state.SplitDirectionVertical,
		NewPane:        state.PaneState{ID: "pane-2", Title: "pane", Kind: state.PaneEmpty},
		Source:         state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post product flow semantic split: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain product flow semantic split: %v", err)
	}
	shell = runtime.State().Shell.EnsureDefaults()
	runtime.state.Shell = runtime.state.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: "pane-2"}, "term-1")
	if binding, ok := runtime.state.TerminalViews.PaneBinding(state.DefaultPaneID); ok {
		owner := state.NewPaneTerminalView(
			"pane-2", binding.TerminalID, binding.Channel, binding.DesiredCols, binding.DesiredRows, state.TerminalResizeRoleOwner, binding.SurfaceID, state.TerminalPaneViewID("pane-2"), true,
		)
		owner.OwnerSurfaceID = owner.SurfaceID
		owner.OwnerViewID = owner.ViewID
		runtime.state.TerminalViews = runtime.state.TerminalViews.BindPane(owner).TransferPaneResizeOwner("pane-2")
	}
	tab := shell.Workspace.Tabs[0]
	if len(tab.Panes) != 2 || tab.RootSplit.Direction != state.SplitDirectionVertical {
		t.Fatalf("semantic split should update pane tree geometry, got %#v", tab)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("framework shortcuts must not leak to terminal input, got %#v", terminal.Inputs)
	}
	if len(terminal.Resizes) == 0 {
		t.Fatalf("split/header/footer/resize changes should drive content rect terminal resize")
	}
	hiddenFrame := lastFrame(t, host.Frames())
	if len(hiddenFrame.Lines) != 22 {
		t.Fatalf("hidden chrome frame should still fill viewport rows, got %d", len(hiddenFrame.Lines))
	}
	for i, line := range hiddenFrame.Lines {
		if render.DisplayWidth(line) != 70 {
			t.Fatalf("hidden chrome frame row %d width must fill viewport, got %d line=%q", i, render.DisplayWidth(line), line)
		}
	}
	if frameContains(hiddenFrame, " ws:") || frameContains(hiddenFrame, " mode:") {
		t.Fatalf("hidden header/footer must reclaim shell bar rows, got %#v", hiddenFrame.Lines)
	}

	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action: state.PaneCommandFocus,
		Target: state.PaneCommandTarget{PaneID: state.DefaultPaneID},
		Source: state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("focus bound pane before missed mouse: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain bound pane focus: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseLeft, Row: 999, Col: 999}); err != nil {
		t.Fatalf("send missed mouse: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "q"}); err != nil {
		t.Fatalf("send terminal input after missed mouse: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain missed mouse and terminal input: %v", err)
	}
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "q" {
		t.Fatalf("missed mouse must not steal following terminal input, got %#v", terminal.Inputs)
	}
	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action: state.PaneCommandFocus,
		Target: state.PaneCommandTarget{PaneID: "pane-2"},
		Source: state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("restore split pane focus before copy mode entry: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain split pane focus: %v", err)
	}
	runtime.state.CopyMode = state.CopyModeStore{}
	runtime.state.History = state.HistoryStore{}
	runtime.state.Shell.InteractionMode = state.InteractionModeNormal
	runtime.state.Shell = runtime.state.Shell.CloseOverlay()
	runtime.state.Shell, _ = runtime.state.Shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandDeactivate, Source: state.PaneCommandSourceTest})

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send copy mode entry: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain copy mode entry: %v", err)
	}
	if len(core.LatestRequests) != 1 || core.LatestRequests[0].Cols <= 0 || core.LatestRequests[0].PaneID != "pane-2" {
		t.Fatalf("copy mode should bind to hidden split content cols, got %#v", core.LatestRequests)
	}
	if runtime.State().CopyMode.BoundToken != "tok-1" || runtime.State().CopyMode.BoundCols != core.LatestRequests[0].Cols {
		t.Fatalf("copy mode should accept first authoritative window, got %#v", runtime.State().CopyMode)
	}

	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action:   state.PaneCommandSetSize,
		Target:   state.PaneCommandTarget{PaneID: "pane-2"},
		SizeMode: state.PaneSizeCells,
		Cols:     34,
	}}); err != nil {
		t.Fatalf("post pane size rebind: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane size rebind: %v", err)
	}
	if len(core.LatestRequests) != 1 || core.LatestRequests[0].Cols != 33 {
		t.Fatalf("pane size local reflow must not request a second latest window, got %#v", core.LatestRequests)
	}
	if runtime.State().CopyMode.BoundToken != "tok-1" || runtime.State().History.Token != "tok-1" {
		t.Fatalf("pane size local reflow should keep current frozen token, got copy=%#v history=%#v", runtime.State().CopyMode, runtime.State().History)
	}
	if runtime.State().CopyMode.BoundCols != 32 || runtime.State().History.Cols != 32 {
		t.Fatalf("pane size local reflow should update local history cols binding, got copy=%#v history=%#v", runtime.State().CopyMode, runtime.State().History)
	}
	if err := runtime.Post(ShellClearToastsMsg{}); err != nil {
		t.Fatalf("post clear toasts after copy rebind: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain clear toasts after copy rebind: %v", err)
	}
	copyFrame := lastFrame(t, host.Frames())
	if !frameContains(copyFrame, "copy-old") || frameContains(copyFrame, "copy-sized") {
		t.Fatalf("pane size local reflow should keep rendering current frozen history, got %#v", copyFrame.Lines)
	}
}

func TestInteractiveRuntimeTUIProductShellAcceptanceFlow(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{Channel: 4, Cols: 100, Rows: 30},
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{
			TerminalID: "term-shell",
			Title:      "shell",
			State:      "running",
			Cols:       100,
			Rows:       30,
		}, {
			TerminalID: "term-logs",
			Title:      "日志🚀",
			State:      "running",
			CWD:        "/tmp/logs",
			Cols:       120,
			Rows:       40,
			Tags:       map[string]string{"role": "logs"},
		}}},
		CreateResult: port.TerminalCreateResult{TerminalID: "term-created", State: "running"},
	}
	host := NewFakeTerminalHost(160)
	host.SetSize(110, 32)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 100, Rows: 30, ResizePolicy: state.TerminalResizeRoleOwner, SurfaceID: "test-surface", ViewID: state.TerminalPaneViewID(state.DefaultPaneID)}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}

	send := func(event input.InputEvent) {
		t.Helper()
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send acceptance input %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain acceptance input %#v: %v", event, err)
		}
	}
	sendKey := func(key input.Key) {
		t.Helper()
		send(input.InputEvent{Kind: input.EventKindKey, Key: key})
	}
	sendChar := func(char string) {
		t.Helper()
		send(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: char})
	}
	sendCtrl := func(char string) {
		t.Helper()
		send(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: char, Ctrl: true})
	}

	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action:         state.PaneCommandSplit,
		Target:         state.PaneCommandTarget{PaneID: state.DefaultPaneID},
		SplitDirection: state.SplitDirectionVertical,
		NewPane:        state.PaneState{ID: "pane-2", Title: "pane", Kind: state.PaneEmpty},
		Source:         state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post product shell semantic split: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain product shell semantic split: %v", err)
	}
	sendCtrl("\x10")
	sendChar("n")
	sendCtrl("\x12")
	sendKey(input.KeyRight)
	sendKey(input.KeyEsc)
	sendCtrl("\x10")
	sendChar("p")
	sendCtrl("\x10")
	sendChar("b")

	shell := runtime.State().Shell.EnsureDefaults()
	if shell.PanelPresentation != state.PanelPresentationSplitLine {
		t.Fatalf("expected split-line presentation in product shell flow, got %#v", shell.PanelPresentation)
	}
	if tab := shell.Workspace.Tabs[0]; len(tab.Panes) != 2 || tab.RootSplit.Direction != state.SplitDirectionVertical {
		t.Fatalf("expected split/focus/balance to keep a vertical pane tree, got %#v", tab)
	}
	if len(terminal.Resizes) == 0 {
		t.Fatalf("resize mode should drive content rect terminal resize")
	}
	paneFrame := lastFrame(t, host.Frames())
	if shell.InteractionMode != state.InteractionModeNormal || !frameContains(paneFrame, "ws:main") {
		t.Fatalf("expected pane mode footer and active feedback, got %#v", paneFrame.Lines)
	}

	sendCtrl("\x0f")
	sendChar("n")
	sendKey(input.KeyRight)
	sendKey(input.KeyDown)
	sendChar("L")
	sendChar("J")
	sendCtrl("\x07")
	sendChar("c")
	sendKey(input.KeyEsc)
	floatingFrame := lastFrame(t, host.Frames())
	if len(runtime.State().Shell.ActiveFloatings()) != 1 ||
		!frameContains(floatingFrame, "No terminal connected") ||
		!frameContains(floatingFrame, "Attach existing terminal") ||
		!frameContains(floatingFrame, "["+render.DefaultPaneChromeGlyphs().Zoom+"]─["+render.DefaultPaneChromeGlyphs().Close+"]") ||
		frameContains(floatingFrame, render.DefaultPaneChromeGlyphs().Running+" float") {
		t.Fatalf("expected floating pane product shell content, shell=%#v frame=%#v", runtime.State().Shell, floatingFrame.Lines)
	}
	floatingClose := frameActionHitRegion(t, floatingFrame, "floating.close", "floating-pane-1")
	if !floatingClose.Floating {
		t.Fatalf("floating close hit region should carry floating flag, got %#v", floatingClose)
	}
	send(mouseEventAt(floatingClose.Rect))
	if len(runtime.State().Shell.ActiveFloatings()) != 0 {
		t.Fatalf("floating close action should remove floating pane, got %#v", runtime.State().Shell.ActiveFloatings())
	}

	sendCtrl("\x07")
	sendChar("p")
	sendChar("日")
	sendChar("志")
	poolFrame := lastFrame(t, host.Frames())
	if runtime.State().Shell.Overlay.Kind != state.OverlayTerminalPool || !frameContains(poolFrame, "Terminal Manager") || !frameContains(poolFrame, "日志🚀") {
		t.Fatalf("expected Terminal Manager page in product shell flow, shell=%#v frame=%#v", runtime.State().Shell, poolFrame.Lines)
	}
	sendKey(input.KeyEnter)
	if len(terminal.Attaches) < 2 || runtime.State().Session.TerminalID != "term-logs" {
		t.Fatalf("Terminal Manager attach should use terminal service, attaches=%#v session=%#v", terminal.Attaches, runtime.State().Session)
	}

	sendCtrl("\x07")
	sendChar("w")
	workbenchFrame := lastFrame(t, host.Frames())
	if runtime.State().Shell.Overlay.Kind != state.OverlayWorkbenchTree || !frameContains(workbenchFrame, "Workbench Navigator") {
		t.Fatalf("expected Workbench Navigator overlay, shell=%#v frame=%#v", runtime.State().Shell, workbenchFrame.Lines)
	}
	sendKey(input.KeyEnter)
	if runtime.State().Shell.Overlay.Open {
		t.Fatalf("Workbench Tree enter should close overlay, got %#v", runtime.State().Shell.Overlay)
	}

	sendCtrl("\x07")
	sendChar(":")
	for _, char := range "system.clear_toasts" {
		sendChar(string(char))
	}
	sendKey(input.KeyEnter)
	if runtime.State().Shell.Overlay.Open {
		t.Fatalf("Prompt submit should execute canonical action, shell=%#v", runtime.State().Shell)
	}
	sendCtrl("\x07")
	sendChar("?")
	helpFrame := lastFrame(t, host.Frames())
	if runtime.State().Shell.Overlay.Kind != state.OverlayHelp || !frameContains(helpFrame, "Most used") {
		t.Fatalf("expected Help overlay, shell=%#v frame=%#v", runtime.State().Shell, helpFrame.Lines)
	}
	if frameContains(helpFrame, "clear toasts") || frameContains(helpFrame, "close toast") {
		t.Fatalf("help overlay should not promote toast controls, got %#v", helpFrame.Lines)
	}
	sendKey(input.KeyEnter)

	sendCtrl("\x14")
	sendChar("c")
	sendKey(input.KeyEsc)
	sendCtrl("\x14")
	sendChar("r")
	sendChar("构")
	sendKey(input.KeyEnter)
	sendCtrl("\x17")
	sendChar("c")
	sendCtrl("\x17")
	sendChar("r")
	sendChar("云")
	sendKey(input.KeyEnter)
	sendKey(input.KeyEsc)
	shell = runtime.State().Shell.EnsureDefaults()
	if shell.Workspace.Name != "workspace 2云" || len(shell.Workspaces) != 2 {
		t.Fatalf("workspace mode should create and rename workspace, got %#v", shell)
	}
	mainWorkspace, ok := findWorkspaceForTest(shell.Workspaces, state.DefaultWorkspaceID)
	if !ok || len(mainWorkspace.Tabs) != 2 || mainWorkspace.Tabs[1].Title != "tab 2构" {
		t.Fatalf("tab mode should create and rename tab in original workspace, ok=%v workspace=%#v", ok, mainWorkspace)
	}
	tabWorkspaceFrame := lastFrame(t, host.Frames())
	if !frameContains(tabWorkspaceFrame, "workspace 2云") || !frameContains(tabWorkspaceFrame, "1 main "+render.DefaultPaneChromeGlyphs().Close) || !frameContains(tabWorkspaceFrame, render.HeaderTabCreateText) || !frameContains(tabWorkspaceFrame, "ws:workspace") {
		t.Fatalf("expected live footer/header after tab/workspace flow, got %#v", tabWorkspaceFrame.Lines)
	}

	sendCtrl("\x07")
	sendChar("h")
	sendCtrl("\x07")
	sendChar("f")
	sendCtrl("\x07")
	sendChar("T")
	sendCtrl("\x07")
	sendChar("c")
	shell = runtime.State().Shell.EnsureDefaults()
	if shell.HeaderVisible || shell.FooterVisible || len(shell.Toasts) != 0 || shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("global mode should hide chrome, clear toasts and return normal, got %#v", shell)
	}
	finalFrame := lastFrame(t, host.Frames())
	if len(finalFrame.Lines) != 32 {
		t.Fatalf("final frame must fill viewport rows, got %d", len(finalFrame.Lines))
	}
	for row, line := range finalFrame.Lines {
		if width := render.DisplayWidth(line); width != 110 {
			t.Fatalf("final frame row %d width=%d want=110 line=%q", row, width, line)
		}
	}
	if frameContains(finalFrame, " ws:") || frameContains(finalFrame, " mode:") {
		t.Fatalf("hidden header/footer must reclaim shell rows, got %#v", finalFrame.Lines)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("product shell UI operations must not leak to terminal input, got %#v", terminal.Inputs)
	}
	if len(terminal.Resizes) == 0 {
		t.Fatalf("product shell layout operations should drive content rect resize")
	}
}

func assertPaneVisualState(t *testing.T, frame render.Frame, text string, style render.StyleToken) {
	t.Helper()
	for _, line := range frame.StyledLines {
		var span strings.Builder
		flush := func() bool {
			if strings.Contains(span.String(), text) {
				return true
			}
			span.Reset()
			return false
		}
		for _, cell := range line.Cells {
			if cell.Style == style {
				span.WriteString(cell.Text)
				continue
			}
			if flush() {
				return
			}
		}
		if flush() {
			return
		}
	}
	t.Fatalf("expected styled pane text %q with style %s, got %#v", text, style, frame.StyledLines)
}

func assertPaneANSIState(t *testing.T, frame render.Frame, text string, style render.ANSICellStyle) {
	t.Helper()
	for _, line := range frame.StyledLines {
		var span strings.Builder
		flush := func() bool {
			if strings.Contains(span.String(), text) {
				return true
			}
			span.Reset()
			return false
		}
		for _, cell := range line.Cells {
			if cell.Style == "" && cell.ANSIStyle == style {
				span.WriteString(cell.Text)
				continue
			}
			if flush() {
				return
			}
		}
		if flush() {
			return
		}
	}
	t.Fatalf("expected ANSI pane text %q with style %#v, got %#v", text, style, frame.StyledLines)
}

func TestInteractiveRuntimeGlobalModeTogglesChromeAndEscExitsMode(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x07", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "h"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x07", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "f"},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send input %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain input %#v: %v", event, err)
		}
	}

	if runtime.State().Shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("expected normal mode after esc, got %#v", runtime.State().Shell.InteractionMode)
	}
	if runtime.State().Shell.HeaderVisible || runtime.State().Shell.FooterVisible {
		t.Fatalf("expected global mode toggles to hide header/footer, got %#v", runtime.State().Shell)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("global shortcuts must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeFloatingPaneProductFlow(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(32)
	host.SetSize(90, 28)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}

	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x0f", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "n"},
		{Kind: input.EventKindKey, Key: input.KeyRight},
		{Kind: input.EventKindKey, Key: input.KeyDown},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "L"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "J"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "z"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "z"},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send floating input %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain floating input %#v: %v", event, err)
		}
	}
	shell := runtime.State().Shell.EnsureDefaults()
	floatings := shell.ActiveFloatings()
	if len(floatings) != 1 || !floatings[0].Active || floatings[0].Collapsed {
		t.Fatalf("expected active restored floating, got %#v", floatings)
	}
	floatingRect := floatings[0].Rect
	frameAfterFloating := lastFrame(t, host.Frames())
	if frameAfterFloating.CursorRect.X < floatingRect.X+1 || frameAfterFloating.CursorRect.X >= floatingRect.X+floatingRect.W-1 ||
		frameAfterFloating.CursorRect.Y < floatingRect.Y+1 || frameAfterFloating.CursorRect.Y >= floatingRect.Y+floatingRect.H-1 {
		t.Fatalf("floating input should anchor cursor inside floating content for IME, floating=%#v cursor=%#v frame=%#v", floatingRect, frameAfterFloating.CursorRect, frameAfterFloating.Cursor)
	}
	vmAfterFloating := render.NewRenderVMBuilder().Build(runtime.State())
	if len(vmAfterFloating.Shell.Layout.Panels) == 0 || vmAfterFloating.Shell.Layout.Panels[0].Active {
		t.Fatalf("active floating should dim tiled pane visual active state, panels=%#v floating=%#v", vmAfterFloating.Shell.Layout.Panels, vmAfterFloating.Shell.Layout.Floating)
	}
	if shell.ActiveFloatings()[0].Rect.W <= 44 || shell.ActiveFloatings()[0].Rect.H <= 12 {
		t.Fatalf("expected keyboard resize to grow floating rect, got %#v", shell.ActiveFloatings()[0].Rect)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("floating shortcuts must not leak terminal input, got %#v", terminal.Inputs)
	}
	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x07", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "c"},
		{Kind: input.EventKindKey, Key: input.KeyEsc},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send clear toast input %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain clear toast input %#v: %v", event, err)
		}
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "No terminal connected") || !frameContains(frame, "Choose a terminal or create one") || !frameContains(frame, "Attach existing terminal") {
		t.Fatalf("expected rendered floating pane, got %#v", frame.Lines)
	}

	raiseRegion := frameActionHitRegion(t, frame, "floating.raise", "floating-pane-1")
	if !raiseRegion.Floating {
		t.Fatalf("floating raise hit region should carry floating flag, got %#v", raiseRegion)
	}
	if err := host.SendInput(mouseEventAt(raiseRegion.Rect)); err != nil {
		t.Fatalf("send floating raise click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating raise: %v", err)
	}
	moveRegion := frameActionHitRegion(t, lastFrame(t, host.Frames()), "floating.move-drag", "floating-pane-1")
	beforeMove := runtime.State().Shell.ActiveFloatings()[0].Rect
	moveStart := mouseEventAt(moveRegion.Rect)
	moveDrag := moveStart
	moveDrag.Mouse = input.MouseLeftDrag
	moveDrag.Col += 3
	moveDrag.Row += 2
	moveRelease := moveDrag
	moveRelease.Mouse = input.MouseLeftUp
	for _, event := range []input.InputEvent{moveStart, moveDrag, moveRelease} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send floating move event %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain floating move event %#v: %v", event, err)
		}
	}
	afterMove := runtime.State().Shell.ActiveFloatings()[0].Rect
	if afterMove.X != beforeMove.X+3 || afterMove.Y != beforeMove.Y+2 {
		t.Fatalf("mouse move should move floating rect, before=%#v after=%#v", beforeMove, afterMove)
	}
	resizeRegion := frameActionHitRegion(t, lastFrame(t, host.Frames()), "floating.resize-drag", "floating-pane-1")
	before := runtime.State().Shell.ActiveFloatings()[0].Rect
	resizeStart := mouseEventAt(resizeRegion.Rect)
	resizeDrag := resizeStart
	resizeDrag.Mouse = input.MouseLeftDrag
	resizeDrag.Col += 4
	resizeDrag.Row += 2
	resizeRelease := resizeDrag
	resizeRelease.Mouse = input.MouseLeftUp
	for _, event := range []input.InputEvent{resizeStart, resizeDrag, resizeRelease} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send floating resize event %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain floating resize event %#v: %v", event, err)
		}
	}
	after := runtime.State().Shell.ActiveFloatings()[0].Rect
	if after.W <= before.W || after.H <= before.H {
		t.Fatalf("mouse resize should grow floating rect, before=%#v after=%#v", before, after)
	}
	if err := runtime.Post(ShellClearToastsMsg{}); err != nil {
		t.Fatalf("post clear toasts before floating close: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain clear toasts before floating close: %v", err)
	}
	closeRegion := frameActionHitRegion(t, lastFrame(t, host.Frames()), "floating.close", "floating-pane-1")
	if err := host.SendInput(mouseEventAt(closeRegion.Rect)); err != nil {
		t.Fatalf("send floating close click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating close: %v", err)
	}
	if len(runtime.State().Shell.ActiveFloatings()) != 0 {
		t.Fatalf("mouse close should remove floating pane, got %#v", runtime.State().Shell.ActiveFloatings())
	}
	vmAfterClose := render.NewRenderVMBuilder().Build(runtime.State())
	if len(vmAfterClose.Shell.Layout.Panels) == 0 || !vmAfterClose.Shell.Layout.Panels[0].Active {
		t.Fatalf("tiled pane visual active state should restore after floating closes, panels=%#v", vmAfterClose.Shell.Layout.Panels)
	}
}

func TestInteractiveRuntimeTabAndWorkspaceProductFlow(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(40)
	host.SetSize(90, 26)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}

	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x14", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "c"},
		{Kind: input.EventKindKey, Key: input.KeyEsc},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x14", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "r"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "构"},
		{Kind: input.EventKindKey, Key: input.KeyEnter},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x14", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "h"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x17", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "c"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x17", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "r"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "云"},
		{Kind: input.EventKindKey, Key: input.KeyEnter},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send input %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain input %#v: %v", event, err)
		}
	}
	workspaceModeFrame := lastFrame(t, host.Frames())
	if !frameContains(workspaceModeFrame, "workspace 2云") || frameContains(workspaceModeFrame, "WORKSPACE") {
		t.Fatalf("expected frame to expose active workspace after prefix exit, got %#v", workspaceModeFrame.Lines)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}); err != nil {
		t.Fatalf("send workspace esc: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain workspace esc: %v", err)
	}

	shell := runtime.State().Shell.EnsureDefaults()
	if shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("expected normal mode after esc, got %#v", shell.InteractionMode)
	}
	if len(shell.Workspaces) != 2 || shell.Workspace.Name != "workspace 2云" {
		t.Fatalf("expected created and renamed workspace active, got %#v", shell)
	}
	if len(shell.Workspace.Tabs) != 1 {
		t.Fatalf("new workspace should start with one tab, got %#v", shell.Workspace.Tabs)
	}
	mainWorkspace, ok := findWorkspaceForTest(shell.Workspaces, state.DefaultWorkspaceID)
	if !ok || len(mainWorkspace.Tabs) != 2 || mainWorkspace.ActiveTabID != state.DefaultTabID {
		t.Fatalf("expected original workspace to retain two tabs and previous active tab, got ok=%v workspace=%#v all=%#v", ok, mainWorkspace, shell.Workspaces)
	}
	if mainWorkspace.Tabs[1].Title != "tab 2构" {
		t.Fatalf("expected renamed tab in original workspace, got %#v", mainWorkspace.Tabs)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("tab/workspace shortcuts must not leak to terminal input, got %#v", terminal.Inputs)
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "workspace 2云") || !frameContains(frame, "1 main "+render.DefaultPaneChromeGlyphs().Close) || !frameContains(frame, render.HeaderTabCreateText) || !frameContains(frame, "ws:workspace") {
		t.Fatalf("expected frame to return to live mode and keep active workspace, got %#v", frame.Lines)
	}
}

func TestShellReducerHandlesFloatingContentActions(t *testing.T) {
	shell := state.DefaultShell()
	shell, _ = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "floating", Kind: state.PaneEmpty},
		Rect:     state.FloatingRect{X: 2, Y: 2, W: 30, H: 8},
		BoundsW:  80,
		BoundsH:  24,
	})
	reducer := NewShellReducer()
	root, _ := reducer(state.Root{
		Shell:    shell,
		Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 24},
	}, shortcutTestMessage("floating.resize", "floating-1", false, 0))
	if got := root.Shell.ActiveFloatings()[0].Rect; got.W != 32 || got.H != 9 {
		t.Fatalf("floating resize action should update rect, got %#v", got)
	}
	root, _ = reducer(root, shortcutTestMessage("floating.center", "floating-1", false, 0))
	if got := root.Shell.ActiveFloatings()[0].Rect; got.X != 24 || got.Y != 7 {
		t.Fatalf("floating center action should center rect, got %#v", got)
	}
	root, _ = reducer(root, shortcutTestMessage("floating.collapse", "floating-1", false, 0))
	if !root.Shell.ActiveFloatings()[0].Collapsed {
		t.Fatalf("floating collapse action should toggle collapsed state, got %#v", root.Shell.ActiveFloatings()[0])
	}
	root, _ = reducer(root, shortcutTestMessage("floating_overview.show_all", "", false, 0))
	if root.Shell.ActiveFloatings()[0].Collapsed {
		t.Fatalf("floating show-all action should expand collapsed panes, got %#v", root.Shell.ActiveFloatings()[0])
	}
	root, _ = reducer(root, shortcutTestMessage("floating_overview.collapse_all", "", false, 0))
	if !root.Shell.ActiveFloatings()[0].Collapsed {
		t.Fatalf("floating collapse-all action should collapse all panes, got %#v", root.Shell.ActiveFloatings()[0])
	}
	root, _ = reducer(root, shortcutTestMessage("floating.close", "floating-1", false, 0))
	if len(root.Shell.ActiveFloatings()) != 0 {
		t.Fatalf("floating close action should remove floating, got %#v", root.Shell.ActiveFloatings())
	}
}

func TestPaneRestartShortcutPreservesEndpoint(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModePane)}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView("west", state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-west", state.TerminalPaneViewID(state.DefaultPaneID), true))

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "R"}})
	if root.Shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("pane restart shortcut should exit prefix mode, shell=%#v", root.Shell)
	}
	restartMsg, ok := runFirstNonStickyTimeoutEffect(t, effects).(TerminalPoolRestartRequestMsg)
	if !ok || restartMsg.EndpointID != "west" || restartMsg.TerminalID != "term-1" {
		t.Fatalf("pane restart should preserve endpoint ref, got %#v", restartMsg)
	}
}

func TestPaneReconnectShortcutPreservesEndpointAndLocalFailureOwner(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModePane)}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView("west", state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-west", state.TerminalPaneViewID(state.DefaultPaneID), true))

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "r"}})
	request, ok := runFirstNonStickyTimeoutEffect(t, effects).(TerminalPoolReconnectRequestMsg)
	if !ok || request.EndpointID != "west" || request.TerminalID != "term-1" || request.TargetPaneID != state.DefaultPaneID || !request.LocalError {
		t.Fatalf("pane reconnect must preserve owning TerminalRef and local error boundary, got %#v", request)
	}
}

func TestFloatingGroupKeyboardIntentsMapToCommands(t *testing.T) {
	root := state.Root{
		Shell:    state.DefaultShell().SetInteractionMode(state.InteractionModeFloating),
		Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 24},
	}
	reducer := NewUIInputReducer()
	for _, tc := range []struct {
		char   string
		action state.FloatingCommandAction
	}{
		{char: "v", action: state.FloatingCommandToggleAll},
		{char: "=", action: state.FloatingCommandFit},
		{char: "s", action: state.FloatingCommandToggleAutoFit},
	} {
		next, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: tc.char}})
		if next.Generation != root.Generation {
			t.Fatalf("keyboard intent %q should defer to effect message, got root=%#v", tc.char, next)
		}
		found := false
		for _, effect := range effects {
			fn, ok := effect.(FuncEffect)
			if !ok {
				continue
			}
			msg, ok := fn.Run(context.Background()).(ShellFloatingCommandMsg)
			if ok && msg.Command.Action == tc.action {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("keyboard intent %q should emit floating command %q, effects=%#v", tc.char, tc.action, effects)
		}
	}
}

func TestShellReducerHandlesResizeOwnerContentAction(t *testing.T) {
	root := state.Root{}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView("west", "pane-1", "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView("west", "pane-2", "term-1", 8, 40, 12, state.TerminalResizeRoleFollower, "surface", "view-2", false))

	next, effects := NewShellReducer()(root, shortcutTestMessage("panel.take_owner", "pane-2", false, 0))
	if len(effects) != 1 {
		t.Fatalf("owner transfer should request authoritative resize control, got %#v", effects)
	}
	first, _ := next.TerminalViews.PaneBinding("pane-1")
	second, _ := next.TerminalViews.PaneBinding("pane-2")
	if first.ResizeRole != state.TerminalResizeRoleOwner || !first.CanResize {
		t.Fatalf("previous owner must remain authoritative until control confirmation, got %#v", first)
	}
	if second.ResizeRole != state.TerminalResizeRoleFollower || second.CanResize || !second.OwnerAcquirePending {
		t.Fatalf("clicked pane should remain pending follower before service result, got %#v", second)
	}
	msg, ok := effects[0].(FuncEffect).Run(context.Background()).(LiveResizeMsg)
	if !ok || msg.EndpointID != "west" || msg.TerminalID != "term-1" || msg.ViewID != "view-2" || msg.Cols != 40 || msg.Rows != 12 || !msg.TakeOwnership {
		t.Fatalf("owner request should confirm clicked view with resize control, got %#v", msg)
	}
}

func TestShellReducerFloatingTakeOwnerContentActionRequiresConfirm(t *testing.T) {
	shell := state.DefaultShell()
	shell, _ = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-1"},
		Title:    "float",
		Rect:     state.FloatingRect{X: 2, Y: 2, W: 42, H: 10},
	})
	root := state.Root{Shell: shell}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	root.TerminalViews = root.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "floating-pane-1", "term-1", 8, 42, 10, state.TerminalResizeRoleFollower, "surface", state.TerminalFloatingViewID("floating-1"), false))

	reducer := NewShellReducer()
	next, effects := reducer(root, shortcutTestMessage("panel.take_owner", "floating-pane-1", true, 0))
	if len(effects) != 1 {
		t.Fatalf("first floating owner click should only arm confirmation timeout, got %#v", effects)
	}
	if fn, ok := effects[0].(FuncEffect); !ok || fn.Token != ownerConfirmClearToken {
		t.Fatalf("first floating owner click should arm owner confirm clear effect, got %#v", effects)
	}
	floating, _ := next.TerminalViews.FloatingBinding("floating-1")
	if floating.ResizeRole != state.TerminalResizeRoleFollower || floating.CanResize {
		t.Fatalf("first floating owner click must not mutate owner truth, got %#v", floating)
	}
	if got := next.Shell.EnsureDefaults().OwnerConfirm.ViewID; got != state.TerminalFloatingViewID("floating-1") {
		t.Fatalf("first floating owner click should arm floating view confirmation, got %q", got)
	}
	vm := render.NewRenderVMBuilder().Build(next)
	if len(vm.Shell.Layout.Floating) != 1 || vm.Shell.Layout.Floating[0].Chrome.Terminal.Owner.Text != "◆ owner?" {
		t.Fatalf("floating owner confirmation should project owner? chrome, got %#v", vm.Shell.Layout.Floating)
	}

	next, effects = reducer(next, shortcutTestMessage("panel.take_owner", "floating-pane-1", true, 0))
	msg, ok := liveResizeMsgFromEffects(effects)
	if !ok || msg.TerminalID != "term-1" || msg.ViewID != state.TerminalFloatingViewID("floating-1") || msg.Cols != 42 || msg.Rows != 10 {
		t.Fatalf("second floating owner click should request authoritative resize owner, msg=%#v effects=%#v", msg, effects)
	}
	floating, _ = next.TerminalViews.FloatingBinding("floating-1")
	if floating.ResizeRole != state.TerminalResizeRoleFollower || floating.CanResize || !floating.OwnerAcquirePending || floating.RequestSeq != msg.Seq || !msg.TakeOwnership {
		t.Fatalf("second floating owner click should wait for authoritative owner, got %#v msg=%#v", floating, msg)
	}
	if got := next.Shell.EnsureDefaults().OwnerConfirm.ViewID; got != "" {
		t.Fatalf("second floating owner click should clear confirmation state, got %q", got)
	}
}

func TestResizeModeTerminalSizeLockKeyAndFooterEmitTerminalLockRequest(t *testing.T) {
	root := state.Root{}
	root.Shell = state.DefaultShell().SetInteractionMode(state.InteractionModeResize)
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, state.TerminalResizeRoleFollower, "surface", "view-2", false))

	inputReducer := NewUIInputReducer()
	next, effects := inputReducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "s"}})
	if next.Shell.InteractionMode != state.InteractionModeNormal || hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("size lock key should emit request effect, got %#v", effects)
	}
	msg, ok := runFirstNonStickyTimeoutEffect(t, effects).(TerminalSizeLockToggleRequestMsg)
	if !ok || msg != (TerminalSizeLockToggleRequestMsg{}) {
		t.Fatalf("resize s should request terminal size lock toggle, got %#v", msg)
	}
	binding, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if binding.Layout.SizeLocked {
		t.Fatalf("resize s must not mutate view-local layout lock, got %#v", binding.Layout)
	}

	shellReducer := NewShellReducer()
	_, effects = shellReducer(next, shortcutActiveTargetTestMessage("panel.size_lock"))
	if hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("footer lock should emit request effect, got %#v", effects)
	}
	if _, ok := runFirstNonStickyTimeoutEffect(t, effects).(TerminalSizeLockToggleRequestMsg); !ok {
		t.Fatalf("footer lock should request terminal size lock toggle, got %#v", effects)
	}
}

func TestResizeModeTerminalLayoutKeysAndActionsShareViewLocalState(t *testing.T) {
	root := state.Root{}
	root.Shell = state.DefaultShell().SetInteractionMode(state.InteractionModeResize)
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, state.TerminalResizeRoleFollower, "surface", "view-2", false))

	inputReducer := NewUIInputReducer()
	next, _ := inputReducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyRight, Shift: true}})
	binding, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	sibling, _ := next.TerminalViews.PaneBinding("pane-2")
	if sibling.Layout.PanX != 0 || sibling.Layout.PanY != 0 {
		t.Fatalf("sibling view on same terminal must not inherit layout pan, got %#v", sibling.Layout)
	}
	binding, _ = next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if binding.Layout.PanX != 2 {
		t.Fatalf("shift-right should pan active terminal view, got %#v", binding.Layout)
	}

	shellReducer := NewShellReducer()
	next, _ = shellReducer(next, shortcutActiveTargetTestMessage("resize.center"))
	binding, _ = next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if binding.Layout.Mode != state.TerminalViewLayoutCenter || binding.Layout.AlignX != state.TerminalViewAlignCenter || binding.Layout.AlignY != state.TerminalViewAlignCenter {
		t.Fatalf("footer center should use same layout command path, got %#v", binding.Layout)
	}
	vm := render.NewRenderVMBuilder().Build(next)
	if len(vm.Shell.Layout.Panels) == 0 || vm.Shell.Layout.Panels[0].Chrome.Terminal.LayoutMode != state.TerminalViewLayoutCenter {
		t.Fatalf("render should project terminal view layout, got %#v", vm.Shell.Layout.Panels)
	}
	if vm.Shell.Layout.Panels[0].Content.Layout.Mode != state.TerminalViewLayoutCenter || vm.Shell.Layout.Panels[0].Content.Layout.AlignX != state.TerminalViewAlignCenter {
		t.Fatalf("render content should consume terminal view layout, got %#v", vm.Shell.Layout.Panels[0].Content.Layout)
	}
}

func TestResizeModeLayoutKeysCanOverrideCenterWithoutReset(t *testing.T) {
	root := state.Root{}
	root.Shell = state.DefaultShell().SetInteractionMode(state.InteractionModeResize)
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))

	inputReducer := NewUIInputReducer()
	next, _ := inputReducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "m"}})
	binding, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if binding.Layout.Mode != state.TerminalViewLayoutCenter || binding.Layout.AlignX != state.TerminalViewAlignCenter || binding.Layout.AlignY != state.TerminalViewAlignCenter {
		t.Fatalf("m should center both axes, got %#v", binding.Layout)
	}

	next.Shell = next.Shell.SetInteractionMode(state.InteractionModeResize)
	next, _ = inputReducer(next, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "0"}})
	binding, _ = next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if binding.Layout.Mode != state.TerminalViewLayoutAuto || binding.Layout.AlignX != state.TerminalViewAlignStart || binding.Layout.AlignY != state.TerminalViewAlignCenter {
		t.Fatalf("align-left should override centered mode without reset, got %#v", binding.Layout)
	}

	next.Shell = next.Shell.SetInteractionMode(state.InteractionModeResize)
	next, _ = inputReducer(next, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "|"}})
	binding, _ = next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if binding.Layout.Mode != state.TerminalViewLayoutAuto || binding.Layout.AlignX != state.TerminalViewAlignCenter || binding.Layout.AlignY != state.TerminalViewAlignStart {
		t.Fatalf("center-x should become horizontal-only center without reset, got %#v", binding.Layout)
	}
}

func TestPaneModeLockUsesTerminalSizeLockPath(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModePane)}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))

	next, effects := NewUIInputReducer()(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "s"}})
	if next.Shell.InteractionMode != state.InteractionModeNormal || hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("pane s should emit terminal size lock request, got %#v", effects)
	}
	if _, ok := runFirstNonStickyTimeoutEffect(t, effects).(TerminalSizeLockToggleRequestMsg); !ok {
		t.Fatalf("pane s should request terminal size lock toggle, got %#v", effects)
	}
	binding, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if binding.Layout.SizeLocked {
		t.Fatalf("pane s must not toggle view-local layout lock, got %#v", binding.Layout)
	}

	next, effects = NewUIInputReducer()(state.Root{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModePane)}, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "s"}})
	if next.Shell.InteractionMode != state.InteractionModeNormal || hasStickyInteractionModeTimeoutEffect(effects) {
		t.Fatalf("pane s without active terminal still routes to size lock reducer, got %#v toasts=%#v", effects, next.Shell.Toasts)
	}
}

func TestOverlayKeyboardCommandsUseCanonicalAppHandlers(t *testing.T) {
	inputReducer := NewUIInputReducer()
	root := state.Root{Shell: state.DefaultShell().OpenTerminalPicker()}

	next, effects := inputReducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyTab}})
	if len(effects) != 1 || len(next.Shell.Toasts) == 0 || next.Shell.Toasts[len(next.Shell.Toasts)-1].Title != "picker.split" {
		t.Fatalf("picker split shortcut should execute canonical handler, root=%#v effects=%#v", next, effects)
	}

	next, effects = inputReducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x05", Ctrl: true}})
	if len(effects) != 1 || len(next.Shell.Toasts) == 0 || next.Shell.Toasts[len(next.Shell.Toasts)-1].Title != "terminal_picker.edit" {
		t.Fatalf("picker edit shortcut should execute canonical handler, root=%#v effects=%#v", next, effects)
	}

	root.Shell = state.DefaultShell().OpenTerminalPool()
	next, effects = inputReducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x0f", Ctrl: true}})
	if len(effects) != 1 || len(next.Shell.Toasts) == 0 || next.Shell.Toasts[len(next.Shell.Toasts)-1].Title != "terminal_pool.attach_float" {
		t.Fatalf("pool attach-float shortcut should execute canonical handler, root=%#v effects=%#v", next, effects)
	}

	root.Shell = state.DefaultShell().OpenWorkbenchTree()
	next, effects = inputReducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x04", Ctrl: true}})
	if len(effects) == 0 || len(next.Shell.Toasts) == 0 {
		t.Fatalf("workbench detach shortcut should execute canonical handler, root=%#v effects=%#v", next, effects)
	}
}

func TestTerminalPickerEditUsesUnfilteredEndpointMetadataAndPreservesTags(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell().OpenTerminalPicker().SetTerminalPickerQuery("mn")}
	root.TerminalPool, _ = root.TerminalPool.ApplyList(0, []state.TerminalPoolItem{{
		EndpointID: "west", TerminalID: "term-main", Title: "main", State: "running", Tags: map[string]string{"role": "build"},
	}}, "")
	items := state.TerminalPickerItems(root)
	for index, item := range items {
		if item.EndpointID == "west" && item.TerminalID == "term-main" {
			root.Shell = root.Shell.SetTerminalPickerSelectedIndex(index, len(items))
			break
		}
	}
	next, effects := NewUIInputReducer()(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x05", Ctrl: true}})
	prompt := next.Shell.EnsureDefaults().Overlay.Prompt
	if next.Shell.EnsureDefaults().Overlay.Kind != state.OverlayPrompt || prompt.Purpose != "terminal.rename" ||
		prompt.TargetEndpointID != "west" || prompt.TargetID != "term-main" || prompt.Value != "main" || prompt.Tags["role"] != "build" {
		t.Fatalf("picker edit must open metadata-preserving endpoint prompt, prompt=%#v", prompt)
	}
	if len(effects) != 1 {
		t.Fatalf("picker edit must only be handled locally until submit, effects=%#v", effects)
	}

	next.Shell = next.Shell.SetPromptValue("build logs")
	next, effects = NewShellReducer()(next, ShellPromptSubmitMsg{})
	if len(effects) != 1 {
		t.Fatalf("rename submit must emit one service request message, effects=%#v", effects)
	}
	request, ok := effects[0].(FuncEffect).Run(context.Background()).(TerminalPoolEditRequestMsg)
	if !ok || request.EndpointID != "west" || request.TerminalID != "term-main" || request.Title != "build logs" || request.Tags["role"] != "build" {
		t.Fatalf("rename request must preserve TerminalRef and tags, got %#v", request)
	}
}

func TestWorkbenchTreeOverlayKeyboardTogglesCollapse(t *testing.T) {
	inputReducer := NewUIInputReducer()
	shell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-2"}, state.SplitDirectionVertical).
		OpenWorkbenchTree()
	root := state.Root{Shell: shell}
	items := state.WorkbenchTreeItems(root)
	root.Shell = root.Shell.SetWorkbenchTreeSelectedIndex(1, len(items))

	next, effects := inputReducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyLeft}})
	if len(effects) == 0 || len(state.WorkbenchTreeItems(next)) != 2 || !state.WorkbenchTreeItems(next)[1].Collapsed {
		t.Fatalf("left arrow should collapse selected tab, items=%#v effects=%#v", state.WorkbenchTreeItems(next), effects)
	}
	next, effects = inputReducer(next, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyRight}})
	if len(effects) == 0 || len(state.WorkbenchTreeItems(next)) != 4 || state.WorkbenchTreeItems(next)[1].Collapsed {
		t.Fatalf("right arrow should expand selected tab, items=%#v effects=%#v", state.WorkbenchTreeItems(next), effects)
	}
	next, effects = inputReducer(next, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	if len(effects) == 0 || len(state.WorkbenchTreeItems(next)) != 2 || !next.Shell.Overlay.Open {
		t.Fatalf("enter on expandable row should toggle without opening target, items=%#v shell=%#v effects=%#v", state.WorkbenchTreeItems(next), next.Shell, effects)
	}
}

func TestOverlayContentActionsUseSelectedItemsAndReducers(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-logs", Channel: 12, Cols: 100, Rows: 30},
	}
	reducer := ComposeReducers(NewShellReducer(), newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal}))
	root := state.Root{Shell: state.DefaultShell().OpenTerminalPool(), Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 30}}
	root.TerminalPool, _ = root.TerminalPool.ApplyList(0, []state.TerminalPoolItem{{TerminalID: "term-logs", Title: "logs", State: "running", Cols: 100, Rows: 30}}, "")

	next, effects := reducer(root, shortcutTestMessage("terminal_pool.attach_float", "", false, -1))
	if len(next.Shell.ActiveFloatings()) != 1 || next.Shell.ActiveFloatingID() != "floating-1" {
		t.Fatalf("pool ctrl-o should create floating before attach, shell=%#v", next.Shell)
	}
	if got := next.Shell.ActiveFloatings()[0].Rect; got != (state.FloatingRect{X: 10, Y: 4, W: 80, H: 22}) {
		t.Fatalf("pool ctrl-o floating should be centered with stable default size, got %#v", got)
	}
	var requestMsg TerminalPoolAttachRequestMsg
	foundAttach := false
	for _, effect := range effects {
		fn, ok := effect.(FuncEffect)
		if !ok {
			continue
		}
		msg, ok := fn.Run(context.Background()).(TerminalPoolAttachRequestMsg)
		if ok {
			requestMsg = msg
			foundAttach = true
			break
		}
	}
	if !foundAttach || requestMsg.TerminalID != "term-logs" || requestMsg.TargetFloatingID != "floating-1" {
		t.Fatalf("pool ctrl-o should request selected terminal attach, msg=%#v", requestMsg)
	}
	_, effects = reducer(next, requestMsg)
	if len(effects) != 1 {
		t.Fatalf("pool attach request should emit service effect, got %#v", effects)
	}
	msg, ok := effects[0].(FuncEffect).Run(context.Background()).(TerminalPoolAttachResultMsg)
	if !ok || msg.TerminalID != "term-logs" || msg.TargetFloatingID != "floating-1" || len(terminal.Attaches) != 1 {
		t.Fatalf("pool ctrl-o should attach selected terminal to floating, msg=%#v attaches=%#v", msg, terminal.Attaches)
	}

	root = state.Root{Shell: state.DefaultShell()}
	root.Shell, _ = root.Shell.ApplyPaneCommand(state.PaneCommand{Action: state.PaneCommandSplit, Target: state.PaneCommandTarget{PaneID: state.DefaultPaneID}, SplitDirection: state.SplitDirectionVertical, NewPane: state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-logs"}})
	root.Shell = root.Shell.OpenWorkbenchTree()
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-logs", 12, 100, 30, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID("pane-2"), false))
	items := state.WorkbenchTreeItems(root)
	selectedPane := false
	for index, item := range items {
		if item.Kind == state.WorkbenchTreeKindPane && item.PaneID == "pane-2" {
			root.Shell = root.Shell.OpenWorkbenchTree().SetWorkbenchTreeSelectedIndex(index, len(items))
			selectedPane = true
			break
		}
	}
	if !selectedPane {
		t.Fatalf("expected workbench tree to contain pane-2, items=%#v", items)
	}
	next, _ = NewShellReducer()(root, shortcutTestMessage("workbench_tree.detach", "", false, -1))
	if pane, ok := next.Shell.Pane(state.PaneCommandTarget{PaneID: "pane-2"}); !ok || pane.Kind != state.PaneEmpty || pane.TerminalID != "" {
		t.Fatalf("workbench ctrl-d should detach selected pane, pane=%#v ok=%v", pane, ok)
	}
	if _, ok := next.TerminalViews.PaneBinding("pane-2"); ok {
		t.Fatal("workbench ctrl-d should detach selected pane terminal view binding")
	}
}

func TestOverlayDeleteContentActionsDispatchTerminalRemove(t *testing.T) {
	reducer := NewShellReducer()
	root := state.Root{Shell: state.DefaultShell().OpenTerminalPool()}
	root.TerminalPool, _ = root.TerminalPool.ApplyList(0, []state.TerminalPoolItem{{TerminalID: "term-logs", Title: "logs"}}, "")

	next, effects := reducer(root, shortcutTestMessage("terminal_pool.delete", "", false, -1))
	if len(effects) != 2 || len(next.Shell.Toasts) != 0 {
		t.Fatalf("pool ctrl-x should dispatch terminal remove without local toast, effects=%#v toasts=%#v", effects, next.Shell.Toasts)
	}
	msg, ok := effects[1].(FuncEffect).Run(context.Background()).(TerminalPoolRemoveRequestMsg)
	if !ok || msg.TerminalID != "term-logs" {
		t.Fatalf("pool ctrl-x should request terminal inventory remove, got %#v", msg)
	}
}

func TestOverlayRestartContentActionsDispatchTerminalRestart(t *testing.T) {
	reducer := NewShellReducer()
	root := state.Root{Shell: state.DefaultShell().OpenTerminalPool()}
	root.TerminalPool, _ = root.TerminalPool.ApplyList(0, []state.TerminalPoolItem{{TerminalID: "term-logs", Title: "logs", State: "exited"}}, "")

	next, effects := reducer(root, shortcutTestMessage("terminal_pool.restart", "", false, -1))
	if len(effects) != 2 || len(next.Shell.Toasts) != 0 {
		t.Fatalf("pool ctrl-r should dispatch terminal restart without local toast, effects=%#v toasts=%#v", effects, next.Shell.Toasts)
	}
	msg, ok := effects[1].(FuncEffect).Run(context.Background()).(TerminalPoolRestartRequestMsg)
	if !ok || msg.TerminalID != "term-logs" {
		t.Fatalf("pool ctrl-r should request terminal restart, got %#v", msg)
	}
}

func TestTerminalPoolEditResultUpdatesVisibleManagerTitle(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell().OpenTerminalPool()}
	root.TerminalPool, _ = root.TerminalPool.ApplyList(0, []state.TerminalPoolItem{{
		TerminalID: "term-logs",
		Title:      "logs",
		Tags:       map[string]string{"role": "old"},
	}}, "")

	next, effects := reduceTerminalPoolEditResult(root, TerminalPoolEditResultMsg{
		TerminalID: "term-logs",
		Title:      "renamed",
		Tags:       map[string]string{"role": "new"},
	})
	if len(effects) == 0 || next.Shell.Overlay.Kind != state.OverlayTerminalPool {
		t.Fatalf("edit result should keep terminal manager visible and refresh later, shell=%#v effects=%#v", next.Shell, effects)
	}
	items := state.TerminalPoolPageItems(next)
	if len(items) != 1 || items[0].Title != "renamed" || items[0].Tags["role"] != "new" {
		t.Fatalf("edit result should update visible terminal metadata immediately, items=%#v", items)
	}
}

func TestInteractiveRuntimeTabJumpUsesWorkbenchCommand(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(24)
	host.SetSize(90, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x14", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "c"},
		{Kind: input.EventKindKey, Key: input.KeyEsc},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x14", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "c"},
		{Kind: input.EventKindKey, Key: input.KeyEsc},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x14", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "1"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x14", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "3"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x14", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "9"},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send tab jump input %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain tab jump input %#v: %v", event, err)
		}
	}
	shell := runtime.State().Shell.EnsureDefaults()
	if shell.Workspace.ActiveTabID != "tab-3" {
		t.Fatalf("tab jump should keep valid tab active after out-of-range jump, got %#v", shell.Workspace)
	}
	if len(shell.Toasts) == 0 || shell.Toasts[len(shell.Toasts)-1].Body != "tab not found" {
		t.Fatalf("out-of-range tab jump should show workbench invalid feedback, got %#v", shell.Toasts)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("tab jump shortcuts must not leak terminal input, got %#v", terminal.Inputs)
	}
}

func frameWorkbenchDetailOpenHitRegion(t *testing.T, frame render.Frame) render.HitRegion {
	t.Helper()
	for _, region := range frame.HitRegions {
		if region.Kind == render.HitRegionContentAction &&
			region.ActionID == render.ActionWorkbenchOpen.String() &&
			region.Row < 0 &&
			region.Rect.H > 1 {
			return region
		}
	}
	t.Fatalf("missing right-side workbench detail open hit region in %#v", frame.HitRegions)
	return render.HitRegion{}
}

func findWorkspaceForTest(workspaces []state.WorkspaceState, id string) (state.WorkspaceState, bool) {
	for _, workspace := range workspaces {
		if workspace.ID == id {
			return workspace, true
		}
	}
	return state.WorkspaceState{}, false
}

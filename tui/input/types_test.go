package input

import (
	"testing"

	"github.com/anytty/anytty/tui/state"
)

func assertShortcutAction(t *testing.T, intent Intent, actionID string) {
	t.Helper()
	if intent.Kind != IntentShortcutAction || string(intent.Invocation.ID) != actionID {
		t.Fatalf("expected shortcut action %q, got %#v", actionID, intent)
	}
}

func TestInputEventKind(t *testing.T) {
	event := InputEvent{Kind: EventKindKey}
	if event.Kind != EventKindKey {
		t.Fatalf("unexpected input event kind %q", event.Kind)
	}
}

func TestRoutePageUpEntersOrRequestsCopyMode(t *testing.T) {
	event := InputEvent{Kind: EventKindKey, Key: KeyPageUp}
	assertShortcutAction(t, Route(event, false), "menu.copy")
	assertShortcutAction(t, Route(event, true), "copy.request_older")
	down := InputEvent{Kind: EventKindKey, Key: KeyPageDn}
	assertShortcutAction(t, Route(down, true), "copy.request_newer")
}

func TestRouteWheelUpEntersOrRequestsCopyMode(t *testing.T) {
	event := InputEvent{Kind: EventKindMouse, Mouse: MouseWheelUp}
	if intent := Route(event, false); intent.Kind != IntentEnterCopyMode {
		t.Fatalf("expected enter copy mode, got %#v", intent)
	}
	if intent := Route(event, true); intent.Kind != IntentRequestOlder {
		t.Fatalf("expected older request, got %#v", intent)
	}
	down := InputEvent{Kind: EventKindMouse, Mouse: MouseWheelDown}
	if intent := Route(down, true); intent.Kind != IntentRequestNewer {
		t.Fatalf("expected newer request, got %#v", intent)
	}
}

func TestRouteTerminalInputAndCopyModeSelection(t *testing.T) {
	key := Route(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "x"}, false)
	if key.Kind != IntentTerminalInput || string(key.Bytes) != "x" {
		t.Fatalf("unexpected terminal input intent %#v", key)
	}
	enter := Route(InputEvent{Kind: EventKindKey, Key: KeyEnter}, false)
	if enter.Kind != IntentTerminalInput || string(enter.Bytes) != "\r" {
		t.Fatalf("unexpected enter intent %#v", enter)
	}
	esc := Route(InputEvent{Kind: EventKindKey, Key: KeyEsc}, false)
	if esc.Kind != IntentTerminalInput || string(esc.Bytes) != "\x1b" {
		t.Fatalf("unexpected esc intent %#v", esc)
	}

	mouse := Route(InputEvent{Kind: EventKindMouse, Mouse: MouseLeft, Row: 2, Col: 3}, true)
	if mouse.Kind != IntentMouseSelect || mouse.Event.Row != 2 || mouse.Event.Col != 3 {
		t.Fatalf("unexpected mouse selection intent %#v", mouse)
	}
}

func TestRouteHostThemeEventDoesNotBecomeTerminalInput(t *testing.T) {
	intent := Route(InputEvent{
		Kind:  EventKindHostTheme,
		Theme: HostThemeEvent{DefaultFG: "#aabbcc"},
	}, false)
	if intent.Kind != IntentNone {
		t.Fatalf("host theme event must not become terminal input, got %#v", intent)
	}
}

func TestRouteCtrlFAndCtrlVToUIIntents(t *testing.T) {
	ctrlF := Route(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\x06", Ctrl: true}, false)
	assertShortcutAction(t, ctrlF, "menu.terminal_picker")
	ctrlV := Route(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\x16", Ctrl: true}, false)
	assertShortcutAction(t, ctrlV, "menu.copy")
	namedCtrlF := Route(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "f", Ctrl: true}, false)
	assertShortcutAction(t, namedCtrlF, "menu.terminal_picker")
}

func TestRouteCopyModePasteShortcuts(t *testing.T) {
	history := Route(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "H"}, true)
	assertShortcutAction(t, history, "menu.clipboard_history")
	lastCopy := Route(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "p"}, true)
	assertShortcutAction(t, lastCopy, "clipboard.paste_latest")
	systemClipboard := Route(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "P"}, true)
	assertShortcutAction(t, systemClipboard, "clipboard.paste_system")
}

func TestRouteInteractionModePrefixesAndModeKeys(t *testing.T) {
	ctrlP := Route(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\x10", Ctrl: true}, false)
	assertShortcutAction(t, ctrlP, "menu.panel")
	ctrlT := Route(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\x14", Ctrl: true}, false)
	assertShortcutAction(t, ctrlT, "menu.tab")
	ctrlW := Route(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\x17", Ctrl: true}, false)
	assertShortcutAction(t, ctrlW, "menu.workspace")
	assertShortcutAction(t, RouteWithMode(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "n"}, false, InteractionModePane), "panel.focus_next")
	assertShortcutAction(t, RouteWithMode(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "c"}, false, InteractionModeTab), "tab.create")
	assertShortcutAction(t, RouteWithMode(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "n"}, false, InteractionModeTab), "tab.next")
	assertShortcutAction(t, RouteWithMode(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "r"}, false, InteractionModeTab), "tab.rename")
	tabJump := RouteWithMode(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "3"}, false, InteractionModeTab)
	assertShortcutAction(t, tabJump, "tab.jump")
	if index, ok := tabJump.Invocation.Param("index"); !ok || index != 3 {
		t.Fatalf("expected tab jump index 3, got %#v", tabJump)
	}
	assertShortcutAction(t, RouteWithMode(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "l"}, false, InteractionModeWorkspace), "workspace.next")
	assertShortcutAction(t, RouteWithMode(InputEvent{Kind: EventKindKey, Key: KeyRight}, false, InteractionModeResize), "resize.right")
	assertShortcutAction(t, RouteWithMode(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "f"}, false, InteractionModeGlobal), "system.toggle_footer")
	assertShortcutAction(t, RouteWithMode(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "t"}, false, InteractionModeGlobal), "menu.terminal_pool")
	assertShortcutAction(t, RouteWithMode(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: ":"}, false, InteractionModeGlobal), "menu.prompt")
	assertShortcutAction(t, RouteWithMode(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "?"}, false, InteractionModeGlobal), "menu.help")
	esc := RouteWithMode(InputEvent{Kind: EventKindKey, Key: KeyEsc}, false, InteractionModePane)
	if esc.Kind != IntentNone {
		t.Fatalf("global back key must not be owned by shortcut routing, got %#v", esc)
	}
}

func TestShortcutPassthroughHelpers(t *testing.T) {
	ctrlT := InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\x14", Ctrl: true}
	intent, ok := LockableRootShortcutIntent(ctrlT)
	if !ok || intent.Invocation.ID != "menu.tab" {
		t.Fatalf("ctrl-t should be a lockable tab shortcut, intent=%#v ok=%v", intent, ok)
	}
	ctrlG := InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\x07", Ctrl: true}
	if intent, ok := LockableRootShortcutIntent(ctrlG); ok {
		t.Fatalf("global entry must stay unlock control plane, got %#v", intent)
	}
	ctrlW := InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\x17", Ctrl: true}
	intent, ok = StickyModeEntryShortcutIntent(ctrlW, InteractionModeWorkspace)
	if !ok || intent.Invocation.ID != "menu.workspace" {
		t.Fatalf("ctrl-w should match workspace sticky entry, intent=%#v ok=%v", intent, ok)
	}
	if intent, ok = StickyModeEntryShortcutIntent(ctrlW, InteractionModeTab); ok {
		t.Fatalf("ctrl-w must not match tab sticky entry, got %#v", intent)
	}
	ctrlV := InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\x16", Ctrl: true}
	intent, ok = CopyModeEntryShortcutIntent(ctrlV)
	if !ok || intent.Invocation.ID != "menu.copy" {
		t.Fatalf("ctrl-v should match copy mode entry, intent=%#v ok=%v", intent, ok)
	}
}

func TestForceTerminalPassthroughBypassesRootShortcutBindings(t *testing.T) {
	ctrlW := InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\x17", Ctrl: true}
	intent := RouteWithOptions(ctrlW, RouteOptions{ForceTerminalPassthrough: true})
	if intent.Kind != IntentTerminalInput || string(intent.Bytes) != "\x17" {
		t.Fatalf("forced ctrl-w should become terminal bytes, got %#v", intent)
	}
	namedCtrlW := InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "w", Ctrl: true}
	intent = RouteWithOptions(namedCtrlW, RouteOptions{ForceTerminalPassthrough: true})
	if intent.Kind != IntentTerminalInput || string(intent.Bytes) != "\x17" {
		t.Fatalf("forced named ctrl-w should become control byte, got %#v", intent)
	}
}

func TestRouteUsesCustomShortcutsAsOnlyTruth(t *testing.T) {
	shortcuts := state.TUIShortcutConfig{
		Scenes: map[string]state.TUIShortcutSceneConfig{
			"global": {
				Bindings: map[string]state.TUIShortcutBindingConfig{
					"ctrl-1": {Action: "tab.jump.1"},
				},
			},
			"panel": {
				Bindings: map[string]state.TUIShortcutBindingConfig{
					"q": {Action: "panel.close"},
					"w": {Action: "panel.close"},
				},
			},
		},
	}

	jump := RouteWithOptions(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "1", Ctrl: true, KeyboardProtocol: KeyboardProtocolKittyCSIU}, RouteOptions{Shortcuts: shortcuts})
	if jump.Kind != IntentShortcutAction || jump.Invocation.ID != "tab.jump" {
		t.Fatalf("custom ctrl-1 should jump tab, got %#v", jump)
	}
	ctrlP := RouteWithOptions(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "p", Ctrl: true}, RouteOptions{Shortcuts: shortcuts})
	if ctrlP.Kind == IntentShortcutAction {
		t.Fatalf("removed ctrl-p must not fall back to default panel menu, got %#v", ctrlP)
	}
	for _, key := range []string{"q", "w"} {
		intent := RouteWithOptions(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: key}, RouteOptions{Mode: InteractionModePane, Shortcuts: shortcuts})
		if intent.Kind != IntentShortcutAction || intent.Invocation.ID != "panel.close" {
			t.Fatalf("custom panel key %q should close pane, got %#v", key, intent)
		}
	}
	removed := RouteWithOptions(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "x"}, RouteOptions{Mode: InteractionModePane, Shortcuts: shortcuts})
	if removed.Kind != IntentNone {
		t.Fatalf("removed panel x must not use default close binding, got %#v", removed)
	}
}

func TestKittyCSIUCtrlDigitRoutesToConfiguredTabJump(t *testing.T) {
	shortcuts := state.TUIShortcutConfig{Scenes: map[string]state.TUIShortcutSceneConfig{
		"global": {Bindings: map[string]state.TUIShortcutBindingConfig{
			"ctrl-1": {Action: "tab.jump.1"},
		}},
	}}
	event := InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "1", Ctrl: true, RawSeq: "\x1b[49;5u", KeyboardProtocol: KeyboardProtocolKittyCSIU}
	intent := RouteWithOptions(event, RouteOptions{Shortcuts: shortcuts})
	if intent.Kind != IntentShortcutAction || intent.Invocation.ID != "tab.jump" {
		t.Fatalf("CSI-u ctrl-1 should route to tab jump, got %#v", intent)
	}
	if index, ok := intent.Invocation.Param("index"); !ok || index != 1 {
		t.Fatalf("CSI-u ctrl-1 lost tab index: %#v", intent.Invocation)
	}
}

func TestEnhancedOnlyBindingRejectsAmbiguousTraditionalTTYEvent(t *testing.T) {
	shortcuts := state.TUIShortcutConfig{Scenes: map[string]state.TUIShortcutSceneConfig{
		"global": {Bindings: map[string]state.TUIShortcutBindingConfig{
			"ctrl-1":      {Action: "tab.jump.1"},
			"shift-enter": {Action: "menu.help"},
		}},
	}}
	for _, event := range []InputEvent{
		{Kind: EventKindKey, Key: KeyChar, Char: "1", Ctrl: true},
		{Kind: EventKindKey, Key: KeyEnter, Shift: true},
	} {
		if intent := RouteWithOptions(event, RouteOptions{Shortcuts: shortcuts}); intent.Kind == IntentShortcutAction {
			t.Fatalf("ambiguous traditional event must not hit enhanced-only binding: event=%#v intent=%#v", event, intent)
		}
		event.KeyboardProtocol = KeyboardProtocolKittyCSIU
		if intent := RouteWithOptions(event, RouteOptions{Shortcuts: shortcuts}); intent.Kind != IntentShortcutAction {
			t.Fatalf("normalized CSI-u event must hit enhanced-only binding: event=%#v intent=%#v", event, intent)
		}
	}
}

func TestShortcutEnhancedRequirementUsesTraditionalCanonicalEvent(t *testing.T) {
	for _, token := range []string{"ctrl-i", "ctrl-j", "ctrl-m", "ctrl-[", "ctrl-?", "ctrl-alt-i"} {
		if !ShortcutKeyRequiresEnhancedKeyboard(token) {
			t.Fatalf("%q is normalized to a different traditional named key and must require CSI-u", token)
		}
	}
	for _, token := range []string{"ctrl-a", "ctrl-@", "ctrl-alt-a", "shift-tab", "ctrl-left"} {
		if ShortcutKeyRequiresEnhancedKeyboard(token) {
			t.Fatalf("%q has an equivalent traditional TTY representation", token)
		}
	}
}

func TestShortcutShowPolicyDoesNotChangeKeyboardRouting(t *testing.T) {
	hide := false
	shortcuts := state.TUIShortcutConfig{Configured: true, Scenes: map[string]state.TUIShortcutSceneConfig{
		"floating": {Bindings: map[string]state.TUIShortcutBindingConfig{
			"n": {Action: "floating.new", Show: &hide},
		}},
	}}
	intent := RouteWithOptions(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "n"}, RouteOptions{Mode: InteractionModeFloating, Shortcuts: shortcuts})
	assertShortcutAction(t, intent, "floating.new")
	entry, ok := ShortcutEntryForEvent(shortcuts, "floating", InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "n"})
	if !ok || entry.Show == nil || *entry.Show {
		t.Fatalf("hidden footer binding must remain in keyboard catalog: %#v ok=%v", entry, ok)
	}
}

func TestCtrlNULAliasesMatchRoutedAndOverlayBindings(t *testing.T) {
	nulEvent := InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\x00", Ctrl: true, RawSeq: "\x00"}
	for _, key := range []string{"ctrl-space", "ctrl-@"} {
		shortcuts := state.TUIShortcutConfig{Configured: true, Scenes: map[string]state.TUIShortcutSceneConfig{
			"global": {Bindings: map[string]state.TUIShortcutBindingConfig{key: {Action: "menu.panel"}}},
			"help":   {Bindings: map[string]state.TUIShortcutBindingConfig{key: {Action: "help.close"}}},
		}}
		intent := RouteWithOptions(nulEvent, RouteOptions{Shortcuts: shortcuts})
		if intent.Kind != IntentShortcutAction || intent.Invocation.ID != "menu.panel" {
			t.Fatalf("routed %q should match NUL input, got %#v", key, intent)
		}
		entry, ok := ShortcutEntryForEvent(shortcuts, "help", nulEvent)
		if !ok || entry.ActionID != "help.close" {
			t.Fatalf("overlay %q should match NUL input, entry=%#v ok=%v", key, entry, ok)
		}
	}
}

func TestRouteUsesExplicitEmptyShortcutsAsOnlyTruth(t *testing.T) {
	shortcuts := state.TUIShortcutConfig{Configured: true, Scenes: map[string]state.TUIShortcutSceneConfig{
		"global": {Bindings: map[string]state.TUIShortcutBindingConfig{}},
	}}

	intent := RouteWithOptions(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "p", Ctrl: true}, RouteOptions{Shortcuts: shortcuts})
	if intent.Kind == IntentShortcutAction {
		t.Fatalf("explicit empty shortcuts must not fall back to default ctrl-p, got %#v", intent)
	}
	for name, options := range map[string]RouteOptions{
		"sticky": {Mode: InteractionModePane, Shortcuts: shortcuts},
		"copy":   {CopyModeActive: true, Shortcuts: shortcuts},
	} {
		intent := RouteWithOptions(InputEvent{Kind: EventKindKey, Key: KeyEsc}, options)
		if intent.Kind != IntentNone {
			t.Fatalf("explicit empty shortcuts must not hardcode %s esc, got %#v", name, intent)
		}
	}
}

func TestBindingCatalogIsUniqueAndContainsDocumentedAliases(t *testing.T) {
	seen := map[string]string{}
	for _, binding := range BindingCatalog() {
		key := string(binding.Mode) + "|" + string(binding.Key) + "|" + binding.Char + "|" + boolKey(binding.Ctrl) + boolKey(binding.Alt) + boolKey(binding.Shift)
		if previous, ok := seen[key]; ok {
			t.Fatalf("duplicate binding %s and %s for %s", previous, binding.ID, key)
		}
		seen[key] = binding.ID
	}
	cases := []struct {
		name   string
		event  InputEvent
		mode   InteractionMode
		action string
	}{
		{name: "pane w close", mode: InteractionModePane, event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "w"}, action: "panel.close"},
		{name: "pane d detach", mode: InteractionModePane, event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "d"}, action: "panel.detach"},
		{name: "pane percent split right", mode: InteractionModePane, event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "%"}, action: "panel.split_right"},
		{name: "pane quote split down", mode: InteractionModePane, event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\""}, action: "panel.split_down"},
		{name: "resize shift left pan", mode: InteractionModeResize, event: InputEvent{Kind: EventKindKey, Key: KeyLeft, Shift: true}, action: "resize.pan_left"},
		{name: "floating 3 summon", mode: InteractionModeFloating, event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "3"}, action: "floating.summon"},
		{name: "tab c create", mode: InteractionModeTab, event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "c"}, action: "tab.create"},
		{name: "workspace f tree", mode: InteractionModeWorkspace, event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "f"}, action: "menu.workbench_tree"},
	}
	for _, tc := range cases {
		assertShortcutAction(t, RouteWithMode(tc.event, false, tc.mode), tc.action)
	}
}

func TestPaneModeUsesTuiv2KeyboardSplitAliases(t *testing.T) {
	cases := []struct {
		name  string
		event InputEvent
		want  string
	}{
		{name: "percent", event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "%"}, want: "panel.split_right"},
		{name: "ctrl-d", event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "d", Ctrl: true}, want: "panel.split_right"},
		{name: "quote", event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\""}, want: "panel.split_down"},
		{name: "ctrl-e", event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "e", Ctrl: true}, want: "panel.split_down"},
	}
	for _, tc := range cases {
		assertShortcutAction(t, RouteWithMode(tc.event, false, InteractionModePane), tc.want)
	}
	for _, char := range []string{"v"} {
		intent := RouteWithMode(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: char}, false, InteractionModePane)
		if intent.Kind != IntentNone {
			t.Fatalf("legacy split key %q should stay unbound, got %#v", char, intent)
		}
	}
}

func TestNormalModePassesUnboundRawKeysAndCommonCtrlToTerminal(t *testing.T) {
	cases := []struct {
		name  string
		event InputEvent
		want  string
	}{
		{name: "ctrl-c", event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\x03", Ctrl: true, RawSeq: "\x03"}, want: "\x03"},
		{name: "arrow", event: InputEvent{Kind: EventKindKey, Key: KeyLeft, RawSeq: "\x1b[D"}, want: "\x1b[D"},
		{name: "home", event: InputEvent{Kind: EventKindKey, Key: KeyHome, RawSeq: "\x1b[H"}, want: "\x1b[H"},
		{name: "delete", event: InputEvent{Kind: EventKindKey, Key: KeyDelete, RawSeq: "\x1b[3~"}, want: "\x1b[3~"},
		{name: "shift-tab", event: InputEvent{Kind: EventKindKey, Key: KeyShiftTab, Shift: true, RawSeq: "\x1b[Z"}, want: "\x1b[Z"},
		{name: "alt-x", event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "x", Alt: true, RawSeq: "\x1bx"}, want: "\x1bx"},
	}
	for _, tc := range cases {
		intent := Route(tc.event, false)
		if intent.Kind != IntentTerminalInput || string(intent.Bytes) != tc.want {
			t.Fatalf("%s: unexpected terminal passthrough %#v", tc.name, intent)
		}
	}
}

func TestEnhancedKeyboardInputDoesNotLeakHostProtocolToPTY(t *testing.T) {
	cases := []struct {
		name  string
		event InputEvent
		want  string
	}{
		{name: "ctrl letter", event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "c", Ctrl: true, RawSeq: "\x1b[99;5u", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: "\x03"},
		{name: "plain unicode", event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "好", RawSeq: "\x1b[22909u", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: "好"},
		{name: "alt char", event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "x", Alt: true, RawSeq: "\x1b[120;3u", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: "\x1bx"},
		{name: "alt enter", event: InputEvent{Kind: EventKindKey, Key: KeyEnter, Alt: true, RawSeq: "\x1b[13;3u", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: "\x1b\r"},
		{name: "alt esc", event: InputEvent{Kind: EventKindKey, Key: KeyEsc, Alt: true, RawSeq: "\x1b[27;3u", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: "\x1b\x1b"},
		{name: "alt tab", event: InputEvent{Kind: EventKindKey, Key: KeyTab, Alt: true, RawSeq: "\x1b[9;3u", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: "\x1b\t"},
		{name: "alt backspace", event: InputEvent{Kind: EventKindKey, Key: KeyBackspace, Alt: true, RawSeq: "\x1b[127;3u", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: "\x1b\x7f"},
		{name: "shift tab", event: InputEvent{Kind: EventKindKey, Key: KeyShiftTab, Shift: true, RawSeq: "\x1b[9;2u", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: "\x1b[Z"},
		{name: "ctrl left", event: InputEvent{Kind: EventKindKey, Key: KeyLeft, Ctrl: true, RawSeq: "synthetic-host-key", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: "\x1b[1;5D"},
		{name: "f3", event: InputEvent{Kind: EventKindKey, Key: KeyF3, Ctrl: true, RawSeq: "synthetic-host-key", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: "\x1b[13;5~"},
		{name: "f5", event: InputEvent{Kind: EventKindKey, Key: KeyF5, RawSeq: "synthetic-host-key", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: "\x1b[15~"},
		{name: "unrepresentable ctrl enter", event: InputEvent{Kind: EventKindKey, Key: KeyEnter, Ctrl: true, RawSeq: "\x1b[13;5u", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: ""},
		{name: "unrepresentable shift enter", event: InputEvent{Kind: EventKindKey, Key: KeyEnter, Shift: true, RawSeq: "\x1b[13;2u", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: ""},
		{name: "unrepresentable ctrl tab", event: InputEvent{Kind: EventKindKey, Key: KeyTab, Ctrl: true, RawSeq: "\x1b[9;5u", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: ""},
		{name: "unrepresentable ctrl digit", event: InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "1", Ctrl: true, RawSeq: "\x1b[49;5u", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: ""},
		{name: "unsupported functional key", event: InputEvent{Kind: EventKindKey, Key: KeyUnknown, RawSeq: "\x1b[57376u", KeyboardProtocol: KeyboardProtocolKittyCSIU}, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			intent := RouteWithOptions(tc.event, RouteOptions{})
			if tc.want == "" {
				if intent.Kind != IntentNone {
					t.Fatalf("unrepresentable enhanced key must not leak to PTY: %#v", intent)
				}
				return
			}
			if intent.Kind != IntentTerminalInput || string(intent.Bytes) != tc.want {
				t.Fatalf("enhanced key semantic passthrough mismatch: got=%#v want=%q", intent, tc.want)
			}
		})
	}
}

func TestUIModesSwallowUnboundKeysAndMouseRequiresTracking(t *testing.T) {
	key := RouteWithMode(InputEvent{Kind: EventKindKey, Key: KeyDelete, RawSeq: "\x1b[3~"}, false, InteractionModePane)
	if key.Kind != IntentNone {
		t.Fatalf("ui mode should swallow unbound delete, got %#v", key)
	}
	mouse := InputEvent{Kind: EventKindMouse, Mouse: MouseRight, RawSeq: "\x1b[<2;10;4M"}
	if intent := RouteWithOptions(mouse, RouteOptions{}); intent.Kind != IntentNone {
		t.Fatalf("mouse without tracking should not passthrough, got %#v", intent)
	}
	intent := RouteWithOptions(mouse, RouteOptions{TerminalMousePassthrough: true})
	if intent.Kind != IntentTerminalInput || !intent.RawMouse || string(intent.Bytes) != mouse.RawSeq {
		t.Fatalf("mouse tracking should raw passthrough, got %#v", intent)
	}
	wheel := InputEvent{Kind: EventKindMouse, Mouse: MouseWheelUp, RawSeq: "\x1b[<64;10;5M"}
	intent = RouteWithOptions(wheel, RouteOptions{TerminalMousePassthrough: true})
	if intent.Kind != IntentTerminalInput || !intent.RawMouse || string(intent.Bytes) != wheel.RawSeq {
		t.Fatalf("tracked wheel should passthrough before entering copy mode, got %#v", intent)
	}
	intent = RouteWithOptions(wheel, RouteOptions{CopyModeActive: true, TerminalMousePassthrough: true})
	if intent.Kind != IntentRequestOlder {
		t.Fatalf("active copy mode should keep wheel for history, got %#v", intent)
	}
	wheelDown := InputEvent{Kind: EventKindMouse, Mouse: MouseWheelDown, RawSeq: "\x1b[<65;10;5M"}
	intent = RouteWithOptions(wheelDown, RouteOptions{CopyModeActive: true, TerminalMousePassthrough: true})
	if intent.Kind != IntentRequestNewer {
		t.Fatalf("active copy mode should keep wheel down for history, got %#v", intent)
	}
}

func boolKey(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

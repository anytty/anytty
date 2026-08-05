package input

import (
	"testing"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/shortcut"
	"github.com/anytty/anytty/tui/state"
)

func TestDefaultShortcutCatalogIsAllowedByDomainRegistry(t *testing.T) {
	for _, entry := range ShortcutEntriesForConfig(state.TUIShortcutConfig{}) {
		invocation, _, err := actiondomain.ParseInvocation(entry.ActionID)
		if err != nil {
			t.Fatalf("default shortcut %s.%s action=%q: %v", entry.Scene, entry.Key, entry.ActionID, err)
		}
		if !shortcut.AllowsScene(invocation.ID, entry.Scene) {
			t.Fatalf("default shortcut %s.%s invocation=%#v is not allowed by shortcut policy", entry.Scene, entry.Key, invocation)
		}
	}
}

func TestShortcutCatalogReplacementMatrix(t *testing.T) {
	defaults := ShortcutEntriesForConfig(state.TUIShortcutConfig{})
	if len(defaults) == 0 {
		t.Fatal("default shortcut catalog must not be empty")
	}

	actionOnly := state.TUIShortcutConfig{
		Configured: true,
		Actions: map[string]state.TUIShortcutActionConfig{
			"menu.panel": {Label: "custom panel"},
		},
	}
	actionEntries := ShortcutEntriesForConfig(actionOnly)
	if len(actionEntries) != len(defaults) {
		t.Fatalf("action-only config must retain default bindings: got=%d want=%d", len(actionEntries), len(defaults))
	}
	if got := RouteWithOptions(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\x10", Ctrl: true}, RouteOptions{Shortcuts: actionOnly}); got.Kind != IntentShortcutAction || got.Invocation.ID != "menu.panel" {
		t.Fatalf("action-only config must retain default ctrl-p binding, got %#v", got)
	}

	sceneOnly := state.TUIShortcutConfig{Configured: true, Scenes: map[string]state.TUIShortcutSceneConfig{
		"global": {Bindings: map[string]state.TUIShortcutBindingConfig{"q": {Action: "menu.panel"}}},
	}}
	entries := ShortcutEntriesForConfig(sceneOnly)
	if len(entries) != 1 || entries[0].Scene != "global" || entries[0].Key != "q" {
		t.Fatalf("declared scene catalog must replace all defaults, got %#v", entries)
	}
	if got := RouteWithOptions(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "\x10", Ctrl: true}, RouteOptions{Shortcuts: sceneOnly}); got.Kind == IntentShortcutAction {
		t.Fatalf("declared scene catalog must remove default ctrl-p, got %#v", got)
	}

	emptyScene := state.TUIShortcutConfig{Configured: true, Scenes: map[string]state.TUIShortcutSceneConfig{
		"global": {Bindings: map[string]state.TUIShortcutBindingConfig{}},
	}}
	if entries := ShortcutEntriesForConfig(emptyScene); len(entries) != 0 {
		t.Fatalf("explicit empty scene must produce an empty user catalog, got %#v", entries)
	}

	mixed := state.TUIShortcutConfig{
		Configured: true,
		Actions: map[string]state.TUIShortcutActionConfig{
			"menu.panel": {Label: "custom panel"},
		},
		Scenes: map[string]state.TUIShortcutSceneConfig{
			"global": {Bindings: map[string]state.TUIShortcutBindingConfig{"q": {Action: "menu.panel"}}},
		},
	}
	entries = ShortcutEntriesForConfig(mixed)
	if len(entries) != 1 || entries[0].Key != "q" {
		t.Fatalf("actions+scenes must keep user scene as the complete binding truth, got %#v", entries)
	}
}

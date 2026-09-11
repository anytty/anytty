package config

import (
	"fmt"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
	"strings"
)

func validPluginActionRef(ref string) bool {
	plugin, action, ok := strings.Cut(ref, "/")
	if !ok || plugin == "" || action == "" || strings.Contains(action, "/") {
		return false
	}
	for _, part := range []string{plugin, action} {
		for _, r := range part {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
				continue
			}
			return false
		}
	}
	return true
}
func validatePluginShortcuts(bindings map[string]string) error {
	for ref, key := range bindings {
		if !validPluginActionRef(ref) {
			return fmt.Errorf("tui.plugin_shortcuts.%s must use plugin-id/action-id", ref)
		}
		if key == "" {
			continue
		}
		if input.ShortcutKeyIsGlobalEscape(key) {
			return fmt.Errorf("tui.plugin_shortcuts.%s uses reserved Esc", ref)
		}
		if _, ok := input.ShortcutBindingSignature("plugin", key); !ok {
			return fmt.Errorf("tui.plugin_shortcuts.%s has invalid key %q", ref, key)
		}
	}
	return nil
}
func setPluginShortcutScalar(cfg *state.TUIConfigStore, ref, key string) (bool, error) {
	if err := validatePluginShortcuts(map[string]string{ref: key}); err != nil {
		return true, err
	}
	if cfg.PluginShortcuts == nil {
		cfg.PluginShortcuts = make(map[string]string)
	}
	cfg.PluginShortcuts[ref] = key
	return true, nil
}

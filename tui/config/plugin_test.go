package config

import "testing"

func TestPluginShortcutsParseWithoutReplacingCoreBindings(t *testing.T) {
	cfg, err := Parse([]byte("tui:\n  plugin_shortcuts:\n    org.anytty.agents/agents.attention: ctrl-alt-a\n    org.anytty.agents/agents.open: \"\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PluginShortcuts["org.anytty.agents/agents.attention"] != "ctrl-alt-a" {
		t.Fatal("override missing")
	}
	if key, ok := cfg.PluginShortcuts["org.anytty.agents/agents.open"]; !ok || key != "" {
		t.Fatal("explicit disable missing")
	}
	if cfg.Shortcuts.Configured {
		t.Fatal("plugin override removed core shortcuts")
	}
	for _, key := range []string{"esc", "ctrl-no-such-key"} {
		if _, err := Parse([]byte("tui:\n  plugin_shortcuts:\n    org.anytty.agents/open: " + key + "\n")); err == nil {
			t.Fatalf("invalid/reserved key %s accepted", key)
		}
	}
}

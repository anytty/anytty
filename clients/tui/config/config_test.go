package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	cfg := Default()
	if cfg.Theme != DefaultTheme {
		t.Fatalf("theme = %q", cfg.Theme)
	}
	if cfg.Icons.PresetName() != DefaultIconPreset {
		t.Fatalf("icons preset = %q", cfg.Icons.PresetName())
	}
	if !cfg.SidebarEnabled() || cfg.GapValue() != 1 {
		t.Fatalf("sidebar=%v gap=%d", cfg.SidebarEnabled(), cfg.GapValue())
	}
	if !cfg.ClockEnabled() || cfg.ClockLayout() != "15:04" {
		t.Fatalf("clock enabled=%v layout=%q", cfg.ClockEnabled(), cfg.ClockLayout())
	}
	if cfg.ForwardWindow() != 300*time.Millisecond {
		t.Fatalf("forward = %v", cfg.ForwardWindow())
	}
	if !cfg.AutoAttachFirst() {
		t.Fatal("auto_attach_first must default to true")
	}
}

func TestParsePartialKeepsDefaultsAndIgnoresUnknown(t *testing.T) {
	cfg, err := Parse([]byte(`{
		"sidebar": false,
		"gap": 0,
		"clock": false,
		"future_field": {"nested": [1,2,3]},
		"startup": {"cwd": "/tmp/work"}
	}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.SidebarEnabled() {
		t.Fatal("sidebar: false must disable the sidebar")
	}
	if cfg.GapValue() != 0 {
		t.Fatalf("gap = %d", cfg.GapValue())
	}
	if cfg.ClockEnabled() {
		t.Fatal("clock: false must disable the clock")
	}
	if cfg.Theme != DefaultTheme || cfg.ForwardWindow() != 300*time.Millisecond {
		t.Fatalf("untouched fields changed: theme=%q forward=%v", cfg.Theme, cfg.ForwardWindow())
	}
	if cfg.Startup.Cwd != "/tmp/work" {
		t.Fatalf("cwd = %q", cfg.Startup.Cwd)
	}
}

func TestClockObjectFormAndThemes(t *testing.T) {
	cfg, err := Parse([]byte(`{"theme": "Light", "clock": {"enabled": true, "format": "12h"}}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Theme != "Light" {
		t.Fatalf("theme = %q", cfg.Theme)
	}
	if cfg.ClockLayout() != "3:04 PM" {
		t.Fatalf("12h layout = %q", cfg.ClockLayout())
	}
}

func TestParseErrorsFallBackToDefaults(t *testing.T) {
	cases := map[string]string{
		"broken json":       `{"theme": `,
		"unknown theme":     `{"theme": "solarized"}`,
		"gap out of range":  `{"gap": 2}`,
		"unknown action":    `{"keybindings": {"pane.zoom": "ctrl-z"}}`,
		"reserved key":      `{"keybindings": {"pane.mode": "esc"}}`,
		"bad key":           `{"keybindings": {"pane.mode": "ctrl-!"}}`,
		"bad forward":       `{"double_click_forward_ms": 0}`,
		"bad clock format":  `{"clock": {"format": "epoch"}}`,
		"null clock object": `{"clock": null}`,
		"unknown icons":     `{"icons": "emoji"}`,
		"unknown icon name": `{"icons": {"map": {"nope": "x"}}}`,
		"empty endpoint":    `{"endpoints": [{"name": "", "argv": ["sh"]}]}`,
		"endpoint colon":    `{"endpoints": [{"name": "a:b", "argv": ["sh"]}]}`,
		"endpoint no argv":  `{"endpoints": [{"name": "remote"}]}`,
		"endpoint kind":     `{"endpoints": [{"name": "remote", "kind": "daemon", "argv": ["sh"]}]}`,
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			cfg, err := Parse([]byte(data))
			if name == "null clock object" {
				if err != nil {
					t.Fatalf("Parse: %v", err)
				}
			} else if err == nil {
				t.Fatalf("Parse(%s) = %+v, want error", data, cfg)
			}
			if cfg.Theme != DefaultTheme || cfg.GapValue() != 1 {
				t.Fatalf("fallback config = %+v", cfg)
			}
		})
	}
}

func TestLoadMissingAndBrokenFiles(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nope.json")
	cfg, err := Load(missing)
	if err != nil || cfg.Theme != DefaultTheme {
		t.Fatalf("missing file: cfg=%+v err=%v", cfg, err)
	}

	broken := filepath.Join(dir, "tui2.json")
	if err := os.WriteFile(broken, []byte(`{"gap": 9}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load(broken)
	if err == nil {
		t.Fatal("broken file must report an error")
	}
	if !strings.Contains(err.Error(), broken) {
		t.Fatalf("error should name the file: %v", err)
	}
	if cfg.GapValue() != 1 {
		t.Fatalf("broken file must fall back to defaults, gap=%d", cfg.GapValue())
	}
}

func TestPathHonorsEnv(t *testing.T) {
	t.Setenv(EnvVar, "/tmp/custom-tui2.json")
	if got := Path(); got != "/tmp/custom-tui2.json" {
		t.Fatalf("Path = %q", got)
	}
	t.Setenv(EnvVar, "")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	if got := Path(); got != filepath.Join("/tmp/xdg", "anytty", "tui2.json") {
		t.Fatalf("Path = %q", got)
	}
}

func TestExampleParses(t *testing.T) {
	cfg, err := Parse([]byte(Example()))
	if err != nil {
		t.Fatalf("example config does not parse: %v", err)
	}
	if cfg.GapValue() != 1 || !cfg.SidebarEnabled() {
		t.Fatalf("example = %+v", cfg)
	}
	if got := cfg.KeybindingOverrides()["picker.open"]; got != "ctrl-f" {
		t.Fatalf("example keybinding = %q", got)
	}
}

func TestIconsPresetsAndOverrides(t *testing.T) {
	recommended, ok := IconPreset("recommended")
	if !ok || recommended["tab_new"] != "\U000F0415" || recommended["tab_marker"] != "\u2387" {
		t.Fatalf("recommended preset = %+v", recommended)
	}
	unicode, ok := IconPreset("unicode")
	if !ok || unicode["tab_new"] != "+" || unicode["slot_close"] != "✕" {
		t.Fatalf("unicode preset = %+v", unicode)
	}
	ascii, ok := IconPreset("ascii")
	if !ok || ascii["slot_close"] != "x" {
		t.Fatalf("ascii preset = %+v", ascii)
	}
	for _, name := range IconPresetNames {
		table, _ := IconPreset(name)
		if len(table) != len(recommended) {
			t.Fatalf("preset %s defines %d icons, want %d", name, len(table), len(recommended))
		}
	}

	// String form selects the preset; the object form overrides single names.
	cfg, err := Parse([]byte(`{"icons": "ascii"}`))
	if err != nil || cfg.Icons.PresetName() != "ascii" {
		t.Fatalf("icons string form: cfg=%+v err=%v", cfg.Icons, err)
	}
	cfg, err = Parse([]byte(`{"icons": {"preset": "unicode", "map": {"tab_new": "N"}}}`))
	if err != nil {
		t.Fatalf("icons object form: %v", err)
	}
	resolved := ResolveIcons(cfg.Icons)
	if resolved["tab_new"] != "N" || resolved["slot_close"] != "✕" {
		t.Fatalf("resolved icons = %+v", resolved)
	}
	if got := cfg.Icons.PresetName(); got != "unicode" {
		t.Fatalf("preset = %q", got)
	}
}

func TestEndpointsParseAndDisplay(t *testing.T) {
	cfg, err := Parse([]byte(`{
		"endpoints": [
			{"name": "remote", "kind": "command", "label": "dev box",
			 "argv": ["sh", "-lc", "echo REMOTE-READY; exec cat"],
			 "cwd": "/tmp", "env": {"ANYTTY_REMOTE": "1"}},
			{"name": "local-2", "argv": ["bash"]}
		]
	}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(cfg.Endpoints) != 2 {
		t.Fatalf("endpoints = %+v", cfg.Endpoints)
	}
	first := cfg.Endpoints[0]
	if first.DisplayName() != "dev box" || first.KindName() != "command" || len(first.Argv) != 3 {
		t.Fatalf("endpoint = %+v", first)
	}
	if cfg.Endpoints[1].DisplayName() != "local-2" || cfg.Endpoints[1].KindName() != "command" {
		t.Fatalf("endpoint = %+v", cfg.Endpoints[1])
	}
}

func TestThemeByNameRecommended(t *testing.T) {
	for _, name := range []string{"", "recommended", "coralline-candy", "Dark", "light"} {
		if _, ok := ThemeByName(name); !ok {
			t.Fatalf("ThemeByName(%q) must be known", name)
		}
	}
	if got, _ := ThemeByName("coralline-candy"); got != "recommended" {
		t.Fatalf("alias = %q", got)
	}
	if _, ok := ThemeByName("solarized"); ok {
		t.Fatal("solarized must stay unknown")
	}
}

func TestValidateKey(t *testing.T) {
	for _, key := range []string{"x", ":", "?", "ctrl-g", "up", "down"} {
		if err := ValidateKey(key); err != nil {
			t.Fatalf("ValidateKey(%q) = %v", key, err)
		}
	}
	for _, key := range []string{"", "ctrl-q", "esc", "enter", "tab", "page-up", "ctrl-!", "ab"} {
		if err := ValidateKey(key); err == nil {
			t.Fatalf("ValidateKey(%q) must fail", key)
		}
	}
}

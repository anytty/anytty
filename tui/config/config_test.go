package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/anytty/anytty/tui/state"
)

func TestParseExampleConfigMatchesDefaults(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "docs", "tui-v3.example.yaml"))
	if err != nil {
		t.Fatalf("read example config: %v", err)
	}
	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("parse example config: %v", err)
	}
	want := Default()
	if want.Daemon.OutputBuffer.Overflow != "block" || want.Daemon.OutputBuffer.CapacityBytes != 32<<20 || want.Daemon.OutputBuffer.ResidentBudgetBytes != 512<<20 {
		t.Fatalf("production output buffer defaults changed: %#v", want.Daemon.OutputBuffer)
	}
	if cfg.Theme.Primary != "" || cfg.Theme.Secondary != "" {
		t.Fatalf("example config should keep primary/secondary host-aware by default, got %#v", cfg.Theme)
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Fatalf("example config should parse to defaults\n got=%#v\nwant=%#v", cfg, want)
	}
}

func TestDefaultVisibleChromeUsesPortableText(t *testing.T) {
	cfg := Default()
	if cfg.Chrome.TabCreateIcon != "+" || cfg.Footer.Templates.KeylockOn != "LOCK" {
		t.Fatalf("unexpected default create/keylock text: chrome=%#v footer=%#v", cfg.Chrome, cfg.Footer.Templates)
	}
	if !strings.Contains(cfg.Chrome.WorkspaceTemplate, " WS {{workspace | truncate 18}} ") ||
		!strings.Contains(cfg.Chrome.TabTemplate, "{{marker}}") ||
		!strings.Contains(cfg.Chrome.TabTemplate, "{{close_icon}}") {
		t.Fatalf("default header templates must keep visible workspace/tab actions, chrome=%#v", cfg.Chrome)
	}
}

func TestParseConfigOverridesScalars(t *testing.T) {
	cfg, err := Parse([]byte(`
version: 1
daemon:
  history:
    max_size_mb: 96
    max_age_days: 14
    compression: s2
    compression_level: balanced
tui:
  profile: work
  theme:
    palette: builtin
    primary: "#d65cff"
    secondary: "#66e3ff"
    border:
      active: "#ff00aa"
  chrome:
    header: false
    panel_presentation: card
    tab_create_icon: "+"
    tab_create_template: "[fg:#040a0d;bg:#8ffcff] {{create_icon}} [/]"
    workspace_template: "[style:header-workspace] {{workspace | truncate 8}} [/style]"
    tab_template: "{{tab_id}} {{if active}}▎{{else}}|{{end}} {{title | truncate 8}} [action:tab.close]{{close_icon}}[/action]"
    pane_title_template: "{{terminal}}@{{endpoint}}"
    pane_glyphs:
      action_left: ""
      action_right: " "
      action_separator: ""
      action_group_left: "[fg:#8ffcff]"
      action_group_right: "[reset]"
      owner_left: " "
      owner_right: " "
      owner: ""
      owner_pending: "owner?"
      take_owner: "follow"
      zoom: "󰁌"
      unzoom: "󰁄"
      split_vertical: "□"
      split_horizontal: "▭"
      close: "⤫"
      size_lock: "🔒"
      size_unlock: "○"
      running: "●"
      waiting: "○"
      exited: "×"
      overflow_left: "‹"
      overflow_right: "›"
      overflow_top: "˄"
      overflow_bottom: "˅"
      overflow_style: "#c7c7c7"
      extent_placeholder: "·"
      extent_placeholder_style: "#a8a8a8"
  footer:
    templates:
      mode_badge: "{{mode_icon}} {{mode_label}}"
      key: ""
      action: "{{key}} {{icon}} {{label}}"
      separator: " · "
      workspace_summary: "󰙅 {{workspace}}"
      floating_summary: "󰹙 {{count}}"
      floating_collapsed_summary: "󰃄 {{count}}"
      terminals_summary: " {{count}}"
      keylock_on: "󰌾 KEYLOCK"
    modes:
      live:
        icon: "󰆍"
        label: "TERM"
        style: footer-accent
  interaction:
    sticky_prefix_timeout_ms: 5000
    shortcut_passthrough_interval_ms: 750
    clipboard_history:
      max_items: 500
      preview_width_ratio: 0.72
  shortcuts:
    actions:
      panel.close:
        label: close
      panel.kill_and_close:
        label: kill+close
    global:
      ctrl-p: menu.panel
      ctrl-1: tab.jump.1
    panel:
      x: panel.close
      k:
        action: panel.kill_and_close
        label: kill terminal + close
    tab:
      "1": tab.jump.1
`))
	if err != nil {
		t.Fatalf("parse override config: %v", err)
	}
	if cfg.Profile != "work" || cfg.Theme.Palette != "builtin" || cfg.Theme.Primary != "#d65cff" || cfg.Theme.Secondary != "#66e3ff" {
		t.Fatalf("theme/profile overrides not applied: %#v", cfg)
	}
	if cfg.Daemon.History.MaxSizeMB != 96 || cfg.Daemon.History.MaxAgeDays != 14 || cfg.Daemon.History.Compression != "s2" || cfg.Daemon.History.CompressionLevel != "balanced" {
		t.Fatalf("daemon history overrides not applied: %#v", cfg.Daemon.History)
	}
	if cfg.Theme.Border.Active != "#ff00aa" || cfg.Chrome.Header || cfg.Chrome.PanelPresentation != "card" || cfg.Chrome.TabCreateIcon != "+" || !strings.Contains(cfg.Chrome.TabCreateTemplate, "{{create_icon}}") || !strings.Contains(cfg.Chrome.WorkspaceTemplate, "{{workspace | truncate 8}}") || !strings.Contains(cfg.Chrome.TabTemplate, "{{tab_id}}") || cfg.Chrome.PaneTitleTemplate != "{{terminal}}@{{endpoint}}" {
		t.Fatalf("chrome/border overrides not applied: %#v", cfg)
	}
	if cfg.Chrome.PaneGlyphs.Zoom != "󰁌" ||
		cfg.Chrome.PaneGlyphs.Unzoom != "󰁄" ||
		!cfg.Chrome.PaneGlyphs.ActionLeftSet ||
		cfg.Chrome.PaneGlyphs.ActionLeft != "" ||
		!cfg.Chrome.PaneGlyphs.ActionRightSet ||
		cfg.Chrome.PaneGlyphs.ActionRight != " " ||
		!cfg.Chrome.PaneGlyphs.ActionSeparatorSet ||
		cfg.Chrome.PaneGlyphs.ActionSeparator != "" ||
		cfg.Chrome.PaneGlyphs.ActionGroupLeft != "[fg:#8ffcff]" ||
		cfg.Chrome.PaneGlyphs.ActionGroupRight != "[reset]" ||
		cfg.Chrome.PaneGlyphs.OwnerLeft != " " ||
		cfg.Chrome.PaneGlyphs.OwnerRight != " " ||
		!cfg.Chrome.PaneGlyphs.OwnerSet ||
		cfg.Chrome.PaneGlyphs.Owner != "" ||
		!cfg.Chrome.PaneGlyphs.OwnerPendingSet ||
		cfg.Chrome.PaneGlyphs.OwnerPending != "owner?" ||
		!cfg.Chrome.PaneGlyphs.TakeOwnerSet ||
		cfg.Chrome.PaneGlyphs.TakeOwner != "follow" ||
		cfg.Chrome.PaneGlyphs.SplitVertical != "□" ||
		cfg.Chrome.PaneGlyphs.SplitHorizontal != "▭" ||
		cfg.Chrome.PaneGlyphs.Close != "⤫" ||
		cfg.Chrome.PaneGlyphs.SizeLock != "🔒" ||
		cfg.Chrome.PaneGlyphs.Running != "●" ||
		!cfg.Chrome.PaneGlyphs.OverflowLeftSet ||
		cfg.Chrome.PaneGlyphs.OverflowLeft != "‹" ||
		!cfg.Chrome.PaneGlyphs.OverflowRightSet ||
		cfg.Chrome.PaneGlyphs.OverflowRight != "›" ||
		!cfg.Chrome.PaneGlyphs.OverflowTopSet ||
		cfg.Chrome.PaneGlyphs.OverflowTop != "˄" ||
		!cfg.Chrome.PaneGlyphs.OverflowBottomSet ||
		cfg.Chrome.PaneGlyphs.OverflowBottom != "˅" ||
		!cfg.Chrome.PaneGlyphs.OverflowStyleSet ||
		cfg.Chrome.PaneGlyphs.OverflowStyle != "#c7c7c7" ||
		!cfg.Chrome.PaneGlyphs.ExtentPlaceholderSet ||
		cfg.Chrome.PaneGlyphs.ExtentPlaceholder != "·" ||
		!cfg.Chrome.PaneGlyphs.ExtentPlaceholderStyleSet ||
		cfg.Chrome.PaneGlyphs.ExtentPlaceholderStyle != "#a8a8a8" {
		t.Fatalf("pane chrome glyph overrides not applied: %#v", cfg.Chrome.PaneGlyphs)
	}
	if cfg.Footer.Templates.Separator != " · " ||
		cfg.Footer.Templates.Key != "" ||
		cfg.Footer.Templates.WorkspaceSummary != "󰙅 {{workspace}}" ||
		cfg.Footer.Templates.FloatingSummary != "󰹙 {{count}}" ||
		cfg.Footer.Templates.TerminalsSummary != " {{count}}" ||
		cfg.Footer.Modes["live"].Icon != "󰆍" ||
		cfg.Footer.Modes["live"].Label != "TERM" ||
		cfg.Footer.Modes["live"].Style != "footer-accent" {
		t.Fatalf("footer overrides not applied: %#v", cfg.Footer)
	}
	if cfg.Interaction.StickyPrefixTimeoutMS != 5000 ||
		cfg.Interaction.ShortcutPassthroughIntervalMS != 750 ||
		cfg.Interaction.ClipboardHistory.MaxItems != 500 ||
		cfg.Interaction.ClipboardHistory.PreviewWidthRatio != 0.72 {
		t.Fatalf("interaction overrides not applied: %#v", cfg.Interaction)
	}
	if cfg.Shortcuts.Actions["panel.close"].Label != "close" ||
		cfg.Shortcuts.Actions["panel.kill_and_close"].Label != "kill+close" ||
		cfg.Shortcuts.Scenes["global"].Bindings["ctrl-p"].Action != "menu.panel" ||
		cfg.Shortcuts.Scenes["global"].Bindings["ctrl-1"].Action != "tab.jump.1" ||
		cfg.Shortcuts.Scenes["panel"].Bindings["x"].Action != "panel.close" ||
		cfg.Shortcuts.Scenes["panel"].Bindings["k"].Action != "panel.kill_and_close" ||
		cfg.Shortcuts.Scenes["panel"].Bindings["k"].Label != "kill terminal + close" ||
		cfg.Shortcuts.Scenes["tab"].Bindings["1"].Action != "tab.jump.1" {
		t.Fatalf("shortcuts overrides not applied: %#v", cfg.Shortcuts)
	}
}

func TestParseRejectsUnknownFieldAndBadValues(t *testing.T) {
	_, err := Parse([]byte("tui:\n  theme:\n    primarry: \"#ffffff\"\n"))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}

	_, err = Parse([]byte("version: 2\n"))
	if err == nil || !strings.Contains(err.Error(), "version must be 1") {
		t.Fatalf("expected version validation error, got %v", err)
	}

	_, err = Parse([]byte("tui:\n  theme:\n    primary: red\n"))
	if err == nil || !strings.Contains(err.Error(), "must be empty or #RRGGBB") {
		t.Fatalf("expected color validation error, got %v", err)
	}

	_, err = Parse([]byte("tui:\n  interaction:\n    shortcut_passthrough_interval_ms: 0\n"))
	if err == nil || !strings.Contains(err.Error(), "shortcut_passthrough_interval_ms must be > 0") {
		t.Fatalf("expected passthrough interval validation error, got %v", err)
	}

	_, err = Parse([]byte("tui:\n  footer:\n    actions:\n      pane:\n        style: nope\n"))
	if err == nil || !strings.Contains(err.Error(), "unknown section") {
		t.Fatalf("expected removed footer actions section error, got %v", err)
	}

	_, err = Parse([]byte("tui:\n  footer:\n    modes:\n      live:\n        actions: \"pane,$bad\"\n"))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected removed footer mode actions field error, got %v", err)
	}

	_, err = Parse([]byte("tui:\n  keymap:\n    root:\n      terminal_picker: ctrl-f\n"))
	if err == nil || !strings.Contains(err.Error(), "unknown section") {
		t.Fatalf("expected removed keymap section error, got %v", err)
	}

	_, err = Parse([]byte("tui:\n  shortcuts:\n    pnale:\n      x: panel.close\n"))
	if err == nil || !strings.Contains(err.Error(), "unknown section") {
		t.Fatalf("expected unknown shortcut scene error, got %v", err)
	}

	_, err = Parse([]byte("daemon:\n  history:\n    max_size_mb: -1\n"))
	if err == nil || !strings.Contains(err.Error(), "max_size_mb") {
		t.Fatalf("expected negative history size error, got %v", err)
	}
	_, err = Parse([]byte("daemon:\n  history:\n    max_age_days: -1\n"))
	if err == nil || !strings.Contains(err.Error(), "max_age_days") {
		t.Fatalf("expected negative history age error, got %v", err)
	}
	_, err = Parse([]byte("daemon:\n  history:\n    compression: gzip\n"))
	if err == nil || !strings.Contains(err.Error(), "compression") {
		t.Fatalf("expected history compression error, got %v", err)
	}
	_, err = Parse([]byte("daemon:\n  history:\n    compression_level: maximum\n"))
	if err == nil || !strings.Contains(err.Error(), "compression_level") {
		t.Fatalf("expected history compression level error, got %v", err)
	}
}

func TestLoadUsesMissingDefaultPathButFailsExplicitMissingPath(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	cfg, err := Load("", nil)
	if err != nil {
		t.Fatalf("missing default config should not fail: %v", err)
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Fatalf("missing default config should return defaults, got %#v", cfg)
	}

	_, err = Load(filepath.Join(configHome, "missing.yaml"), nil)
	if err == nil {
		t.Fatalf("explicit missing config path should fail")
	}
}

func TestLoadAppliesEnvOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tui-v3.yaml")
	if err := os.WriteFile(path, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	env := map[string]string{
		"ANYTTY_OUTPUT_BUFFER_CAPACITY_BYTES":         "1048576",
		"ANYTTY_OUTPUT_BUFFER_OVERFLOW":               "block",
		"ANYTTY_OUTPUT_RESIDENT_BUDGET_BYTES":         "268435456",
		"ANYTTY_HISTORY_MAX_SIZE_MB":                  "128",
		"ANYTTY_HISTORY_MAX_AGE_DAYS":                 "30",
		"ANYTTY_HISTORY_COMPRESSION":                  "none",
		"ANYTTY_HISTORY_COMPRESSION_LEVEL":            "best",
		"ANYTTY_TUI_THEME_PRIMARY":                    "#010203",
		"ANYTTY_TUI_THEME_PALETTE":                    "builtin",
		"ANYTTY_TUI_CHROME_HEADER":                    "false",
		"ANYTTY_TUI_CHROME_TAB_CREATE_TEMPLATE":       "{{create_icon}}",
		"ANYTTY_TUI_CHROME_PANE_TITLE_TEMPLATE":       "{{terminal}}@{{endpoint}}",
		"ANYTTY_TUI_SHORTCUT_PASSTHROUGH_INTERVAL_MS": "650",
		"ANYTTY_TUI_CLIPBOARD_HISTORY_PREVIEW_RATIO":  "0.75",
	}
	cfg, err := Load(path, func(key string) string { return env[key] })
	if err != nil {
		t.Fatalf("load with env overrides: %v", err)
	}
	if cfg.Theme.Primary != "#010203" ||
		cfg.Daemon.OutputBuffer.CapacityBytes != 1<<20 ||
		cfg.Daemon.OutputBuffer.Overflow != "block" ||
		cfg.Daemon.OutputBuffer.ResidentBudgetBytes != 256<<20 ||
		cfg.Daemon.History.MaxSizeMB != 128 ||
		cfg.Daemon.History.MaxAgeDays != 30 ||
		cfg.Daemon.History.Compression != "none" ||
		cfg.Daemon.History.CompressionLevel != "best" ||
		cfg.Theme.Palette != "builtin" ||
		cfg.Chrome.Header ||
		cfg.Chrome.TabCreateTemplate != "{{create_icon}}" ||
		cfg.Chrome.PaneTitleTemplate != "{{terminal}}@{{endpoint}}" ||
		cfg.Interaction.ShortcutPassthroughIntervalMS != 650 ||
		cfg.Interaction.ClipboardHistory.PreviewWidthRatio != 0.75 {
		t.Fatalf("env overrides not applied: %#v", cfg)
	}
}

func TestOutputBufferYAMLAndValidation(t *testing.T) {
	cfg, err := Parse([]byte("version: 1\ndaemon:\n  output_buffer:\n    capacity_bytes: 65536\n    overflow: block\n    resident_budget_bytes: 2147483648\n"))
	if err != nil {
		t.Fatalf("parse output buffer config: %v", err)
	}
	if cfg.Daemon.OutputBuffer.CapacityBytes != 64<<10 || cfg.Daemon.OutputBuffer.Overflow != "block" || cfg.Daemon.OutputBuffer.ResidentBudgetBytes != 2<<30 {
		t.Fatalf("output buffer config not applied: %#v", cfg.Daemon.OutputBuffer)
	}
	for name, mutate := range map[string]func(*state.TUIConfigStore){
		"capacity below minimum": func(cfg *state.TUIConfigStore) { cfg.Daemon.OutputBuffer.CapacityBytes = 65535 },
		"capacity above maximum": func(cfg *state.TUIConfigStore) { cfg.Daemon.OutputBuffer.CapacityBytes = 256<<20 + 1 },
		"invalid overflow":       func(cfg *state.TUIConfigStore) { cfg.Daemon.OutputBuffer.Overflow = "bounded" },
		"budget above maximum":   func(cfg *state.TUIConfigStore) { cfg.Daemon.OutputBuffer.ResidentBudgetBytes = 2<<30 + 1 },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := Default()
			mutate(&invalid)
			if err := Validate(invalid); err == nil {
				t.Fatal("invalid output buffer config was accepted")
			}
		})
	}
}

func TestValidateRejectsMultiLineTabCreateTemplate(t *testing.T) {
	cfg := Default()
	cfg.Chrome.TabCreateTemplate = "{{create_icon}}\n+"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "tab_create_template") {
		t.Fatalf("expected multiline tab create template validation error, got %v", err)
	}
}

func TestValidateRejectsMultiLinePaneTitleTemplate(t *testing.T) {
	cfg := Default()
	cfg.Chrome.PaneTitleTemplate = "{{terminal}}\n{{endpoint}}"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "pane_title_template") {
		t.Fatalf("expected pane_title_template validation error, got %v", err)
	}
}

func TestValidateRejectsMultiLineWorkspaceTemplate(t *testing.T) {
	cfg := Default()
	cfg.Chrome.WorkspaceTemplate = "{{workspace}}\n{{title}}"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "workspace_template") {
		t.Fatalf("expected workspace_template validation error, got %v", err)
	}
}

func TestParseRejectsDuplicateShortcutKeyWithinScene(t *testing.T) {
	_, err := Parse([]byte(`
tui:
  shortcuts:
    panel:
      x: panel.close
      x: panel.detach
`))
	if err == nil || !strings.Contains(err.Error(), "duplicate shortcut key") {
		t.Fatalf("expected duplicate shortcut key error, got %v", err)
	}
}

func TestParseExpandsShortcutKeyRangesAndBindingDisplayPolicy(t *testing.T) {
	cfg, err := Parse([]byte(`
tui:
  shortcuts:
    floating:
      "[1...3]":
        action: floating.summon.{key}
        label: summon
        show: true
    global:
      "ctrl+[1...2]": tab.jump.{key}
`))
	if err != nil {
		t.Fatalf("parse shortcut ranges: %v", err)
	}
	for index := 1; index <= 3; index++ {
		key := strconv.Itoa(index)
		binding, ok := cfg.Shortcuts.Scenes["floating"].Bindings[key]
		if !ok || binding.Action != "floating.summon."+key || binding.Label != "summon" || binding.Show == nil || !*binding.Show {
			t.Fatalf("floating range did not expand binding %s: %#v", key, binding)
		}
	}
	for index := 1; index <= 2; index++ {
		key := "ctrl-" + strconv.Itoa(index)
		binding, ok := cfg.Shortcuts.Scenes["global"].Bindings[key]
		if !ok || binding.Action != "tab.jump."+strconv.Itoa(index) {
			t.Fatalf("modified range did not expand binding %s: %#v", key, binding)
		}
	}
}

func TestParseKeepsLiteralBracketShortcutKeys(t *testing.T) {
	cfg, err := Parse([]byte(`
tui:
  shortcuts:
    tab:
      "[": tab.previous
      "]": tab.next
    global:
      "ctrl-[": menu.tab
`))
	if err != nil {
		t.Fatalf("parse literal bracket shortcut keys: %v", err)
	}
	if got := cfg.Shortcuts.Scenes["tab"].Bindings["["]; got.Action != "tab.previous" {
		t.Fatalf("literal [ shortcut was not preserved: %#v", got)
	}
	if got := cfg.Shortcuts.Scenes["tab"].Bindings["]"]; got.Action != "tab.next" {
		t.Fatalf("literal ] shortcut was not preserved: %#v", got)
	}
	if got := cfg.Shortcuts.Scenes["global"].Bindings["ctrl-["]; got.Action != "menu.tab" {
		t.Fatalf("modified literal bracket shortcut was not preserved: %#v", got)
	}
}

func TestParseRejectsInvalidOrOverlappingShortcutKeyRanges(t *testing.T) {
	cases := []struct {
		name    string
		config  string
		wantErr string
	}{
		{name: "descending", config: `tui:\n  shortcuts:\n    floating:\n      "[9...1]": floating.summon.{key}\n`, wantErr: "invalid shortcut key range"},
		{name: "missing brackets", config: `tui:\n  shortcuts:\n    floating:\n      "1...9": floating.summon.{key}\n`, wantErr: "invalid shortcut key range"},
		{name: "missing range end", config: `tui:\n  shortcuts:\n    floating:\n      "[1...]": floating.summon.{key}\n`, wantErr: "invalid shortcut key range"},
		{name: "missing range start", config: `tui:\n  shortcuts:\n    floating:\n      "[...9]": floating.summon.{key}\n`, wantErr: "invalid shortcut key range"},
		{name: "non-digit range", config: `tui:\n  shortcuts:\n    floating:\n      "[a...9]": floating.summon.{key}\n`, wantErr: "invalid shortcut key range"},
		{name: "missing closing bracket", config: `tui:\n  shortcuts:\n    floating:\n      "[1...9": floating.summon.{key}\n`, wantErr: "invalid shortcut key range"},
		{name: "nested brackets", config: `tui:\n  shortcuts:\n    floating:\n      "[[1...9]]": floating.summon.{key}\n`, wantErr: "invalid shortcut key range"},
		{name: "multiple ranges", config: `tui:\n  shortcuts:\n    floating:\n      "[1...2...3]": floating.summon.{key}\n`, wantErr: "invalid shortcut key range"},
		{name: "modified missing closing bracket", config: `tui:\n  shortcuts:\n    global:\n      "ctrl+[1...9": tab.jump.{key}\n`, wantErr: "invalid shortcut key range"},
		{name: "placeholder without range", config: `tui:\n  shortcuts:\n    floating:\n      "1": floating.summon.{key}\n`, wantErr: "requires a shortcut key range"},
		{name: "overlap", config: `tui:\n  shortcuts:\n    floating:\n      "[1...3]": floating.summon.{key}\n      "3": floating.summon.3\n`, wantErr: "duplicate shortcut key"},
		{name: "canonical overlap", config: `tui:\n  shortcuts:\n    global:\n      "ctrl+[1...2]": tab.jump.{key}\n      "Ctrl-1": tab.jump.1\n`, wantErr: "runtime shortcut key"},
		{name: "expanded parameter out of range", config: `tui:\n  shortcuts:\n    tab:\n      "[0...1]": tab.jump.{key}\n`, wantErr: "invalid index"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(strings.ReplaceAll(tc.config, `\n`, "\n")))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestParseShortcutCatalogDeclarationMatrix(t *testing.T) {
	cfg, err := Parse([]byte("tui:\n  shortcuts: {}\n"))
	if err != nil {
		t.Fatalf("parse inline empty shortcuts map: %v", err)
	}
	if cfg.Shortcuts.Configured || len(cfg.Shortcuts.Scenes) != 0 {
		t.Fatalf("inline empty shortcuts map must keep defaults, got %#v", cfg.Shortcuts)
	}

	if _, err := Parse([]byte("tui:\n  shortcuts:\n")); err == nil || !strings.Contains(err.Error(), "must be a map") {
		t.Fatalf("null shortcuts must be rejected, got %v", err)
	}
	if _, err := Parse([]byte("tui:\n  shortcuts:\n    global:\n    panel: {}\n")); err == nil || !strings.Contains(err.Error(), "global must be a map") {
		t.Fatalf("null shortcut scene must be rejected before its sibling, got %v", err)
	}
	if _, err := Parse([]byte("tui:\n  shortcuts:\n    actions:\n")); err == nil || !strings.Contains(err.Error(), "actions must be a map") {
		t.Fatalf("null shortcut actions must be rejected, got %v", err)
	}

	cfg, err = Parse([]byte("tui:\n  shortcuts:\n    global: {}\n"))
	if err != nil {
		t.Fatalf("parse explicit empty shortcut scene: %v", err)
	}
	global, ok := cfg.Shortcuts.Scenes["global"]
	if !cfg.Shortcuts.Configured || !ok || len(global.Bindings) != 0 {
		t.Fatalf("explicit empty scene must be preserved, got %#v", cfg.Shortcuts)
	}

	cfg, err = Parse([]byte("tui:\n  shortcuts:\n    actions:\n      menu.panel:\n        label: custom panel\n        style: footer-key-pane\n"))
	if err != nil {
		t.Fatalf("parse action-only shortcuts: %v", err)
	}
	if !cfg.Shortcuts.Configured || len(cfg.Shortcuts.Scenes) != 0 || cfg.Shortcuts.Actions["menu.panel"].Label != "custom panel" || cfg.Shortcuts.Actions["menu.panel"].Style != "footer-key-pane" {
		t.Fatalf("action-only shortcuts must not declare a scene catalog, got %#v", cfg.Shortcuts)
	}

	cfg, err = Parse([]byte("tui:\n  shortcuts:\n    actions:\n      menu.pane:\n        label: alias panel\n"))
	if err != nil {
		t.Fatalf("parse alias action label: %v", err)
	}
	if len(cfg.Shortcuts.Actions) != 1 || cfg.Shortcuts.Actions["menu.panel"].Label != "alias panel" {
		t.Fatalf("alias action label must be stored by canonical id, got %#v", cfg.Shortcuts.Actions)
	}

	if _, err := Parse([]byte("tui:\n  shortcuts:\n    actions:\n      menu.panel:\n        label: panel\n      menu.pane:\n        label: pane\n")); err == nil || !strings.Contains(err.Error(), "duplicate shortcut action label") {
		t.Fatalf("canonical and alias action labels must not overwrite each other, got %v", err)
	}

	if _, err := Parse([]byte("tui:\n  shortcuts:\n    actions:\n      menu.panel:\n        style: neon\n")); err == nil || !strings.Contains(err.Error(), "unknown style token") {
		t.Fatalf("unknown shortcut action styles must be rejected, got %v", err)
	}
}

func TestValidateRejectsInvalidShortcutConfig(t *testing.T) {
	if _, err := Parse([]byte("tui:\n  shortcuts:\n    actions:\n      empty.attach:\n        label: attach\n")); err == nil {
		t.Fatal("surface-only action label must be rejected during parsing")
	}

	cfg := Default()
	cfg.Shortcuts.Actions = map[string]state.TUIShortcutActionConfig{
		"menu.panel": {Label: "panel"},
		"menu.pane":  {Label: "pane"},
	}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "duplicates canonical shortcut action") {
		t.Fatalf("expected canonical action label duplicate error, got %v", err)
	}

	cfg = Default()
	cfg.Shortcuts.Actions = map[string]state.TUIShortcutActionConfig{
		"empty.attach": {Label: "attach"},
	}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "unknown shortcut action") {
		t.Fatalf("surface-only action must not be accepted as a shortcut label, got %v", err)
	}

	cfg = Default()
	cfg.Shortcuts.Actions = map[string]state.TUIShortcutActionConfig{
		"panel$close": {Label: "close"},
	}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "invalid action id") {
		t.Fatalf("expected invalid action id error, got %v", err)
	}

	cfg = Default()
	cfg.Shortcuts.Actions = map[string]state.TUIShortcutActionConfig{
		"panel.close": {Label: "close\nnow"},
	}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "single-line") {
		t.Fatalf("expected multiline shortcut label error, got %v", err)
	}

	cfg = Default()
	cfg.Shortcuts.Actions = map[string]state.TUIShortcutActionConfig{
		"panel.clsoe": {Label: "close"},
	}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "unknown shortcut action") {
		t.Fatalf("expected unknown shortcut action label error, got %v", err)
	}

	cfg = Default()
	cfg.Shortcuts.Scenes = map[string]state.TUIShortcutSceneConfig{
		"pnale": {
			Bindings: map[string]state.TUIShortcutBindingConfig{
				"x": {Action: "panel.close"},
			},
		},
	}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "invalid scene") {
		t.Fatalf("expected invalid shortcut scene error, got %v", err)
	}

	cfg = Default()
	cfg.Shortcuts.Scenes = map[string]state.TUIShortcutSceneConfig{
		"global": {
			Bindings: map[string]state.TUIShortcutBindingConfig{
				"ctrl-z": {Action: "menu.unknown"},
			},
		},
	}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "unknown shortcut scene") {
		t.Fatalf("expected unknown menu target error, got %v", err)
	}

	cfg = Default()
	cfg.Shortcuts.Scenes = map[string]state.TUIShortcutSceneConfig{
		"global": {
			Bindings: map[string]state.TUIShortcutBindingConfig{
				"ctrl-z": {Action: "panel.clsoe"},
			},
		},
	}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "unknown shortcut action") {
		t.Fatalf("expected unknown shortcut action error, got %v", err)
	}

	cfg = Default()
	cfg.Shortcuts.Scenes = map[string]state.TUIShortcutSceneConfig{
		"global": {
			Bindings: map[string]state.TUIShortcutBindingConfig{
				"ctrl-f": {Action: "menu.terminal_picker"},
			},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected terminal picker menu action to validate, got %v", err)
	}

	cfg = Default()
	cfg.Shortcuts.Scenes = map[string]state.TUIShortcutSceneConfig{
		"panel": {
			Bindings: map[string]state.TUIShortcutBindingConfig{
				"x": {Action: "panel.close"},
			},
		},
		"pane": {
			Bindings: map[string]state.TUIShortcutBindingConfig{
				"x": {Action: "panel.detach"},
			},
		},
	}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime shortcut key") {
		t.Fatalf("expected panel/pane runtime key conflict error, got %v", err)
	}
}

func TestShortcutValidationUsesDomainScenesParametersAndCanonicalKeys(t *testing.T) {
	cases := []struct {
		name    string
		scenes  map[string]state.TUIShortcutSceneConfig
		wantErr string
	}{
		{name: "unknown prompt suggestion scene", scenes: map[string]state.TUIShortcutSceneConfig{
			"prompt_suggestion": {Bindings: map[string]state.TUIShortcutBindingConfig{"x": {Action: "help.close"}}},
		}, wantErr: "invalid scene"},
		{name: "wrong routed action scene", scenes: map[string]state.TUIShortcutSceneConfig{
			"panel": {Bindings: map[string]state.TUIShortcutBindingConfig{"x": {Action: "workspace.delete"}}},
		}, wantErr: "not allowed in scene"},
		{name: "parameter missing", scenes: map[string]state.TUIShortcutSceneConfig{
			"tab": {Bindings: map[string]state.TUIShortcutBindingConfig{"1": {Action: "tab.jump"}}},
		}, wantErr: "requires index"},
		{name: "parameter out of range", scenes: map[string]state.TUIShortcutSceneConfig{
			"tab": {Bindings: map[string]state.TUIShortcutBindingConfig{"0": {Action: "tab.jump.0"}}},
		}, wantErr: "invalid index"},
		{name: "canonical key alias conflict", scenes: map[string]state.TUIShortcutSceneConfig{
			"help": {Bindings: map[string]state.TUIShortcutBindingConfig{
				"enter": {Action: "help.close"}, "return": {Action: "help.close"},
			}},
		}, wantErr: "runtime shortcut key"},
		{name: "reserved global back key", scenes: map[string]state.TUIShortcutSceneConfig{
			"help": {Bindings: map[string]state.TUIShortcutBindingConfig{"escape": {Action: "help.close"}}},
		}, wantErr: "reserved global back key"},
		{name: "ctrl character case conflict", scenes: map[string]state.TUIShortcutSceneConfig{
			"global": {Bindings: map[string]state.TUIShortcutBindingConfig{
				"ctrl-a": {Action: "menu.panel"}, "ctrl-A": {Action: "menu.tab"},
			}},
		}, wantErr: "runtime shortcut key"},
		{name: "ctrl nul alias conflict", scenes: map[string]state.TUIShortcutSceneConfig{
			"global": {Bindings: map[string]state.TUIShortcutBindingConfig{
				"ctrl-space": {Action: "menu.panel"}, "ctrl-@": {Action: "menu.tab"},
			}},
		}, wantErr: "runtime shortcut key"},
		{name: "overlay ctrl nul alias conflict", scenes: map[string]state.TUIShortcutSceneConfig{
			"help": {Bindings: map[string]state.TUIShortcutBindingConfig{
				"ctrl-space": {Action: "help.close"}, "ctrl-@": {Action: "help.close"},
			}},
		}, wantErr: "runtime shortcut key"},
		{name: "modifier order conflict", scenes: map[string]state.TUIShortcutSceneConfig{
			"global": {Bindings: map[string]state.TUIShortcutBindingConfig{
				"ctrl-alt-x": {Action: "menu.panel"}, "alt-ctrl-x": {Action: "menu.tab"},
			}},
		}, wantErr: "runtime shortcut key"},
		{name: "repeated modifier conflict", scenes: map[string]state.TUIShortcutSceneConfig{
			"global": {Bindings: map[string]state.TUIShortcutBindingConfig{
				"ctrl-x": {Action: "menu.panel"}, "ctrl-ctrl-x": {Action: "menu.tab"},
			}},
		}, wantErr: "runtime shortcut key"},
		{name: "unrepresentable overlay key", scenes: map[string]state.TUIShortcutSceneConfig{
			"help": {Bindings: map[string]state.TUIShortcutBindingConfig{"ctrl-not-a-real-key": {Action: "help.close"}}},
		}, wantErr: "invalid key"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Default()
			cfg.Shortcuts.Configured = true
			cfg.Shortcuts.Scenes = tc.scenes
			if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}

	cfg := Default()
	cfg.Shortcuts.Configured = true
	cfg.Shortcuts.Scenes = map[string]state.TUIShortcutSceneConfig{
		"global": {Bindings: map[string]state.TUIShortcutBindingConfig{
			"ctrl-1":     {Action: "tab.jump.1"},
			"ctrl-enter": {Action: "menu.tab"},
			"ctrl-i":     {Action: "menu.panel"},
			".":          {Action: "menu.pane"},
		}},
		"help": {Bindings: map[string]state.TUIShortcutBindingConfig{"return": {Action: "help.close"}}},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("valid aliases and parameterized actions should pass: %v", err)
	}

	parsed, err := Parse([]byte("tui:\n  shortcuts:\n    global:\n      \".\": menu.pane\n"))
	if err != nil {
		t.Fatalf("quoted dot shortcut should load through the real parser: %v", err)
	}
	if got := parsed.Shortcuts.Scenes["global"].Bindings["."].Action; got != "menu.pane" {
		t.Fatalf("quoted dot shortcut action mismatch: %q", got)
	}

	parsed, err = Parse([]byte("tui:\n  shortcuts:\n    global:\n      \".\":\n        action: menu.pane\n        label: dot menu\n"))
	if err != nil {
		t.Fatalf("quoted dot long-form shortcut should load through the real parser: %v", err)
	}
	dotBinding := parsed.Shortcuts.Scenes["global"].Bindings["."]
	if dotBinding.Action != "menu.pane" || dotBinding.Label != "dot menu" {
		t.Fatalf("quoted dot long-form shortcut mismatch: %#v", dotBinding)
	}
}

func TestDefaultPathUsesXDGConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	if got, want := DefaultPath(), filepath.Join(configHome, "anytty", DefaultFileName); got != want {
		t.Fatalf("default path mismatch got=%q want=%q", got, want)
	}
}

var _ state.TUIConfigStore

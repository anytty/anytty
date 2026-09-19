package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/config"
)

func TestShellOptionsParse(t *testing.T) {
	opts, err := parseOptions([]string{"--config", "/tmp/x.json", "--print-default-config", "--version"})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opts.configPath != "/tmp/x.json" || !opts.printDefault || !opts.version {
		t.Fatalf("opts = %+v", opts)
	}
	if _, err := parseOptions([]string{"--nope"}); err == nil {
		t.Fatal("unknown flag must fail")
	}
}

func TestLoadConfigFallsBackWithWarning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tui2.json")
	if err := os.WriteFile(path, []byte(`{"gap": 3}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, warning := loadConfig(path)
	if !strings.Contains(warning, "config error") || !strings.Contains(warning, "gap") {
		t.Fatalf("warning = %q, want one-line gap error", warning)
	}
	if cfg.GapValue() != 1 || !cfg.SidebarEnabled() {
		t.Fatalf("fallback config = %+v", cfg)
	}
}

func TestApplyConfigKeybindingCollisionKeepsDefaults(t *testing.T) {
	m := newModel()
	cfg := config.Default()
	cfg.Keybindings = map[string]string{"pane.mode": "x"} // x is slot.close
	if err := m.applyConfig(cfg); err == nil {
		t.Fatal("colliding bindings must be rejected")
	}
	if got := m.binds.key(actionPaneMode); got != "ctrl-p" {
		t.Fatalf("pane.mode after rejected config = %q, want ctrl-p", got)
	}
}

func TestApplyConfigAppliesValues(t *testing.T) {
	m := newModel()
	sidebar := false
	gap := 0
	forward := 500
	cfg := config.Default()
	cfg.Sidebar = &sidebar
	cfg.Gap = &gap
	cfg.Clock.Enabled = false
	cfg.DoubleClickForwardMS = &forward
	cfg.Startup.Cwd = "/tmp/work"
	cfg.Keybindings = map[string]string{"tab.new": "ctrl-y"}
	if err := m.applyConfig(cfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if m.sidebar || m.gap != 0 || m.clockEnabled || m.startupCwd != "/tmp/work" {
		t.Fatalf("model config = sidebar=%v gap=%d clock=%v cwd=%q", m.sidebar, m.gap, m.clockEnabled, m.startupCwd)
	}
	if m.forwardWindow != 500*time.Millisecond {
		t.Fatalf("forwardWindow = %v", m.forwardWindow)
	}
	if got := m.binds.key(actionTabNew); got != "ctrl-y" {
		t.Fatalf("tab.new = %q", got)
	}
}

func TestParseOptionsAttach(t *testing.T) {
	opts, err := parseOptions([]string{"-attach", "local:term-1", "-config", "/tmp/tui2.json"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.attach != "local:term-1" || opts.configPath != "/tmp/tui2.json" {
		t.Fatalf("options = %+v", opts)
	}
}

// Package config parses the tui2-shell JSON configuration. Only the layout
// program reads it: the host has no configuration file, holds no palette and
// only implements the protocol. Parsing never panics: Load and Parse return
// the built-in defaults plus an error, and the shell surfaces the error as a
// one-line notice before running with the defaults.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EnvVar overrides the default config path when --config is not given.
const EnvVar = "ANYTTY_TUI2_CONFIG"

// Built-in defaults (v1). The default theme is the legacy "coralline-candy"
// recommended profile (see tui2/docs/RECOMMENDED_CONFIG.zh-CN.md); "dark" and
// "light" stay available for existing configurations.
const (
	DefaultTheme     = "recommended"
	DefaultGap       = 1
	DefaultForwardMS = 300
	DefaultClock24h  = "24h"
	DefaultClock12h  = "12h"
)

// ActionWhitelist is the fixed set of configurable keybinding actions. The
// names are stable API; unknown actions in "keybindings" are a config error.
var ActionWhitelist = []string{
	"pane.mode",
	"pane.split_h",
	"pane.split_v",
	"picker.open",
	"tab.new",
	"slot.close",
	"scroll.copy",
	"sidebar.toggle",
	"terminal.restart",
	"prompt.open",
	"help.open",
}

// reservedKeys can never be bound: the host and the overlay state machine
// own them unconditionally.
var reservedKeys = map[string]bool{
	"ctrl-q": true, "esc": true, "enter": true, "tab": true,
	"page-up": true, "page-down": true, "backspace": true,
}

// Clock accepts either a boolean ("clock": false) or the object form
// ({"enabled": true, "format": "12h"}).
type Clock struct {
	Enabled bool
	Format  string
}

// UnmarshalJSON implements the bool-or-object form.
func (c *Clock) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil
	}
	if trimmed[0] == 't' || trimmed[0] == 'f' {
		var enabled bool
		if err := json.Unmarshal(trimmed, &enabled); err != nil {
			return err
		}
		c.Enabled = enabled
		if c.Format == "" {
			c.Format = DefaultClock24h
		}
		return nil
	}
	type objectClock struct {
		Enabled *bool  `json:"enabled"`
		Format  string `json:"format"`
	}
	var raw objectClock
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return fmt.Errorf("clock: %w", err)
	}
	if raw.Enabled != nil {
		c.Enabled = *raw.Enabled
	}
	if raw.Format != "" {
		c.Format = raw.Format
	}
	return nil
}

// MarshalJSON always emits the explicit object form.
func (c Clock) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Enabled bool   `json:"enabled"`
		Format  string `json:"format"`
	}{c.Enabled, c.Format})
}

// Startup configures the cold-start behavior of the layout program.
type Startup struct {
	// AutoAttachFirst adopts the first known terminal on cold start; false
	// opens the picker instead.
	AutoAttachFirst *bool `json:"auto_attach_first"`
	// Cwd is passed to terminal.create; empty means the host default.
	Cwd string `json:"cwd"`
}

// Config is the v1 config file. Pointer fields track presence so a partial
// file keeps every other built-in default.
type Config struct {
	Theme                string            `json:"theme"`
	Icons                Icons             `json:"icons"`
	Sidebar              *bool             `json:"sidebar"`
	Gap                  *int              `json:"gap"`
	Clock                Clock             `json:"clock"`
	DoubleClickForwardMS *int              `json:"double_click_forward_ms"`
	Keybindings          map[string]string `json:"keybindings"`
	Startup              Startup           `json:"startup"`
	Endpoints            []Endpoint        `json:"endpoints"`
}

// Default returns the built-in configuration.
func Default() Config {
	sidebar := true
	gap := DefaultGap
	forward := DefaultForwardMS
	autoAttach := true
	return Config{
		Theme:                DefaultTheme,
		Icons:                Icons{Preset: DefaultIconPreset},
		Sidebar:              &sidebar,
		Gap:                  &gap,
		Clock:                Clock{Enabled: true, Format: DefaultClock24h},
		DoubleClickForwardMS: &forward,
		Keybindings:          map[string]string{},
		Startup:              Startup{AutoAttachFirst: &autoAttach},
	}
}

// Path is the default config path: $ANYTTY_TUI2_CONFIG when set, then
// $XDG_CONFIG_HOME/anytty/tui2.json, then ~/.config/anytty/tui2.json.
func Path() string {
	if path := os.Getenv(EnvVar); path != "" {
		return path
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "anytty", "tui2.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "anytty", "tui2.json")
}

// Load reads the config at path (empty means Path()). A missing file yields
// the defaults without error; a broken file yields the defaults plus the
// parse error.
func Load(path string) (Config, error) {
	if path == "" {
		path = Path()
	}
	if path == "" {
		return Default(), nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), fmt.Errorf("read %s: %w", path, err)
	}
	cfg, err := Parse(data)
	if err != nil {
		return Default(), fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// Parse decodes and validates one config document. Unknown fields are
// ignored; a decode or validation failure returns the defaults plus error.
func Parse(data []byte) (Config, error) {
	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), fmt.Errorf("invalid JSON: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Default(), err
	}
	return cfg, nil
}

// Validate rejects values outside the documented v1 space.
func (c Config) Validate() error {
	if _, ok := ThemeByName(c.Theme); !ok {
		return fmt.Errorf("theme %q: want dark or light", c.Theme)
	}
	if c.Gap != nil && *c.Gap != 0 && *c.Gap != 1 {
		return fmt.Errorf("gap %d: want 0 or 1", *c.Gap)
	}
	if c.DoubleClickForwardMS != nil && (*c.DoubleClickForwardMS <= 0 || *c.DoubleClickForwardMS > 5000) {
		return fmt.Errorf("double_click_forward_ms %d: want 1..5000", *c.DoubleClickForwardMS)
	}
	if format := c.Clock.Format; format != "" && format != DefaultClock24h && format != DefaultClock12h {
		return fmt.Errorf("clock.format %q: want 24h or 12h", format)
	}
	for action, key := range c.Keybindings {
		if !KnownAction(action) {
			return fmt.Errorf("keybindings: unknown action %q", action)
		}
		if err := ValidateKey(key); err != nil {
			return fmt.Errorf("keybindings.%s: %w", action, err)
		}
	}
	if _, ok := IconPreset(c.Icons.PresetName()); !ok {
		return fmt.Errorf("icons preset %q: want one of %s", c.Icons.PresetName(), strings.Join(IconPresetNames, ", "))
	}
	for name, glyph := range c.Icons.Map {
		if !KnownIcon(name) {
			return fmt.Errorf("icons.map: unknown icon %q", name)
		}
		if strings.ContainsAny(glyph, "\r\n") {
			return fmt.Errorf("icons.map.%s must be a single-line glyph", name)
		}
	}
	return validateEndpoints(c.Endpoints)
}

// KnownAction reports whether action is in the whitelist.
func KnownAction(action string) bool {
	for _, known := range ActionWhitelist {
		if action == known {
			return true
		}
	}
	return false
}

// ValidateKey accepts one normalized key: a single printable rune, a
// ctrl-<letter> chord, or one of up/down/left/right.
func ValidateKey(key string) error {
	if key == "" {
		return errors.New("empty key")
	}
	if reservedKeys[key] {
		return fmt.Errorf("key %q is reserved", key)
	}
	switch key {
	case "up", "down", "left", "right":
		return nil
	}
	if strings.HasPrefix(key, "ctrl-") {
		rest := strings.TrimPrefix(key, "ctrl-")
		if len(rest) == 1 && rest[0] >= 'a' && rest[0] <= 'z' {
			return nil
		}
		return fmt.Errorf("key %q: want ctrl-a..ctrl-z", key)
	}
	runes := []rune(key)
	if len(runes) == 1 && runes[0] > ' ' && runes[0] != 0x7f {
		return nil
	}
	return fmt.Errorf("key %q: want a single printable character", key)
}

// ThemeByName validates a theme name without importing the render palette.
// "recommended" is the legacy coralline-candy profile; "coralline-candy" is
// accepted as its alias.
func ThemeByName(name string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", DefaultTheme, "coralline-candy":
		return DefaultTheme, true
	case "dark":
		return "dark", true
	case "light":
		return "light", true
	default:
		return "", false
	}
}

// SidebarEnabled resolves the sidebar default.
func (c Config) SidebarEnabled() bool {
	return c.Sidebar == nil || *c.Sidebar
}

// GapValue resolves the gutter width (0 or 1).
func (c Config) GapValue() int {
	if c.Gap == nil {
		return DefaultGap
	}
	return *c.Gap
}

// ClockEnabled resolves the clock default.
func (c Config) ClockEnabled() bool {
	return c.Clock.Enabled
}

// ClockLayout resolves the clock format to a Go time layout.
func (c Config) ClockLayout() string {
	if c.Clock.Format == DefaultClock12h {
		return "3:04 PM"
	}
	return "15:04"
}

// ForwardWindow returns the picker double-press forward window.
func (c Config) ForwardWindow() time.Duration {
	ms := DefaultForwardMS
	if c.DoubleClickForwardMS != nil {
		ms = *c.DoubleClickForwardMS
	}
	return time.Duration(ms) * time.Millisecond
}

// AutoAttachFirst resolves the cold-start adoption default.
func (c Config) AutoAttachFirst() bool {
	return c.Startup.AutoAttachFirst == nil || *c.Startup.AutoAttachFirst
}

// KeybindingOverrides returns a copy of the action -> key overrides.
func (c Config) KeybindingOverrides() map[string]string {
	if len(c.Keybindings) == 0 {
		return nil
	}
	out := make(map[string]string, len(c.Keybindings))
	for action, key := range c.Keybindings {
		out[action] = key
	}
	return out
}

// Example is the deterministic JSON printed by --print-default-config.
func Example() string {
	return `{
  "theme": "recommended",
  "icons": { "preset": "recommended" },
  "sidebar": true,
  "gap": 1,
  "clock": { "enabled": true, "format": "24h" },
  "double_click_forward_ms": 300,
  "keybindings": { "picker.open": "ctrl-f" },
  "startup": { "auto_attach_first": true, "cwd": "" },
  "endpoints": [
    { "name": "remote", "kind": "command", "argv": ["ssh", "host", "anytty", "attach"] }
  ]
}
`
}

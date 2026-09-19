package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// DefaultIconPreset is the glyph set the shell uses when the config does not
// say: the Nerd Font icons of the legacy "coralline-candy" recommended
// profile (see tui2/docs/RECOMMENDED_CONFIG.zh-CN.md).
const DefaultIconPreset = "recommended"

// IconPresetNames lists the built-in glyph presets accepted by tui2.json.
// "unicode" is the pre-recommended compatibility set and "ascii" a
// conservative fallback for terminals without a Nerd Font; both stay
// available so an existing tui2.json keeps its exact look.
var IconPresetNames = []string{DefaultIconPreset, "unicode", "ascii"}

// iconPresets is the single source of truth for the icon name space: every
// preset defines the same names, and a config "icons.map" may override any of
// them. Names are stable API (documented in RECOMMENDED_CONFIG).
var iconPresets = map[string]map[string]string{
	"recommended": {
		"workspace":         "\U000F0645", // nf-md-folder; legacy workspace_template
		"tab_marker":        "\u2387",     // ⎇ legacy active/inactive tab marker
		"tab_new":           "\U000F0415", // legacy tab_create_icon
		"tab_close":         "\U000F0156", // legacy pane glyph close
		"slot_marker":       "\u258E",     // ▎ focus marker (legacy pane title)
		"slot_restart":      "\U000F0450", // legacy reconnect/restart
		"slot_split_h":      "\uEB56",     // legacy split_vertical (side-by-side)
		"slot_split_v":      "\uEB57",     // legacy split_horizontal (stacked)
		"slot_close":        "\U000F0156", // legacy pane glyph close
		"mode_normal":       "\U000F030C", // legacy live mode badge
		"mode_pane":         "\uEBEB",     // legacy pane mode badge
		"mode_scroll":       "\U000F018F", // legacy copy mode badge
		"mode_picker":       "\U000F0C7C", // legacy terminal picker badge
		"mode_prompt":       "\U0000F4B5", // legacy command prompt badge
		"mode_help":         "\U000F02D6", // legacy help badge
		"summary_workspace": "\U000F0645", // legacy workspace_summary
		"summary_tab":       "\U000F04E9", // legacy tabs_summary
		"summary_slot":      "\uEBEB",     // legacy panes_summary
		"summary_floating":  "\U000F0E59", // legacy floating_summary
		"summary_terminals": "\U0000F489", // legacy terminals_summary
		"endpoint":          "\U000F0337", // legacy connections icon
		// Footer key-group glyphs (RECOMMENDED_CONFIG "footer 规格表"): the
		// legacy shortcut labels carry these icons next to the key.
		"focus":     "\U000F0734", // legacy panel.focus H/L label icon
		"page_up":   "\U000F005D", // legacy copy PGUP label icon
		"page_down": "\U000F0045", // legacy copy PGDN label icon
		"attach":    "\U000F02FA", // legacy terminal_picker enter label icon
		"run":       "\U000F0627", // legacy prompt enter label icon
	},
	"unicode": {
		"workspace":         "",
		"tab_marker":        "",
		"tab_new":           "+",
		"tab_close":         "×",
		"slot_marker":       "\u258E",
		"slot_restart":      "⟳",
		"slot_split_h":      "⇔",
		"slot_split_v":      "⇕",
		"slot_close":        "✕",
		"mode_normal":       "",
		"mode_pane":         "",
		"mode_scroll":       "",
		"mode_picker":       "",
		"mode_prompt":       "",
		"mode_help":         "",
		"summary_workspace": "",
		"summary_tab":       "",
		"summary_slot":      "",
		"summary_floating":  "",
		"summary_terminals": "",
		"endpoint":          "",
		"focus":             "",
		"page_up":           "",
		"page_down":         "",
		"attach":            "",
		"run":               "",
	},
	"ascii": {
		"workspace":         "ws",
		"tab_marker":        "",
		"tab_new":           "+",
		"tab_close":         "x",
		"slot_marker":       ">",
		"slot_restart":      "R",
		"slot_split_h":      "|",
		"slot_split_v":      "-",
		"slot_close":        "x",
		"mode_normal":       "",
		"mode_pane":         "",
		"mode_scroll":       "",
		"mode_picker":       "",
		"mode_prompt":       "",
		"mode_help":         "",
		"summary_workspace": "",
		"summary_tab":       "",
		"summary_slot":      "",
		"summary_floating":  "",
		"summary_terminals": "",
		"endpoint":          "",
		"focus":             "",
		"page_up":           "",
		"page_down":         "",
		"attach":            "",
		"run":               "",
	},
}

// Icons selects the glyph preset and per-name overrides. The JSON form is
// either the preset name ("icons": "recommended") or the object form
// ({"preset": "...", "map": {"tab_new": "+"}}); the object form always wins
// over the preset, and a zero Icons means DefaultIconPreset.
type Icons struct {
	Preset string
	Map    map[string]string
}

// UnmarshalJSON implements the string-or-object form.
func (i *Icons) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil
	}
	if trimmed[0] == '"' {
		var preset string
		if err := json.Unmarshal(trimmed, &preset); err != nil {
			return fmt.Errorf("icons: %w", err)
		}
		i.Preset = preset
		return nil
	}
	type objectIcons struct {
		Preset string            `json:"preset"`
		Map    map[string]string `json:"map"`
	}
	var raw objectIcons
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return fmt.Errorf("icons: %w", err)
	}
	i.Preset = raw.Preset
	i.Map = raw.Map
	return nil
}

// MarshalJSON always emits the explicit object form.
func (i Icons) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Preset string            `json:"preset"`
		Map    map[string]string `json:"map,omitempty"`
	}{i.PresetName(), i.Map})
}

// PresetName resolves the effective preset name (empty means the default).
func (i Icons) PresetName() string {
	if strings.TrimSpace(i.Preset) == "" {
		return DefaultIconPreset
	}
	return strings.TrimSpace(i.Preset)
}

// IconPreset returns a copy of one preset's glyph table.
func IconPreset(name string) (map[string]string, bool) {
	table, ok := iconPresets[name]
	if !ok {
		return nil, false
	}
	out := make(map[string]string, len(table))
	for key, value := range table {
		out[key] = value
	}
	return out, true
}

// KnownIcon reports whether name is part of the icon name space.
func KnownIcon(name string) bool {
	_, ok := iconPresets[DefaultIconPreset][name]
	return ok
}

// ResolveIcons merges the selected preset with the per-name overrides. An
// unknown preset falls back to the default; callers should Validate first so
// a typo is surfaced as a config error instead of silently changing the look.
func ResolveIcons(icons Icons) map[string]string {
	out, ok := IconPreset(icons.PresetName())
	if !ok {
		out, _ = IconPreset(DefaultIconPreset)
	}
	for name, glyph := range icons.Map {
		if KnownIcon(name) {
			out[name] = glyph
		}
	}
	return out
}

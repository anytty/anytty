package main

// Icon names resolved from tui2.json "icons" (preset or per-name map). The
// name space lives in tui2/config/icons.go; the recommended preset carries
// the legacy Nerd Font glyphs (tui2/docs/RECOMMENDED_CONFIG.zh-CN.md).
const (
	iconWorkspace        = "workspace"
	iconTabMarker        = "tab_marker"
	iconTabNew           = "tab_new"
	iconTabClose         = "tab_close"
	iconSlotMarker       = "slot_marker"
	iconSlotRestart      = "slot_restart"
	iconSlotSplitH       = "slot_split_h"
	iconSlotSplitV       = "slot_split_v"
	iconSlotClose        = "slot_close"
	iconModeNormal       = "mode_normal"
	iconModePane         = "mode_pane"
	iconModeScroll       = "mode_scroll"
	iconModePicker       = "mode_picker"
	iconModePrompt       = "mode_prompt"
	iconModeHelp         = "mode_help"
	iconSummaryWorkspace = "summary_workspace"
	iconSummaryTab       = "summary_tab"
	iconSummarySlot      = "summary_slot"
	iconSummaryFloating  = "summary_floating"
	iconSummaryTerminals = "summary_terminals"
	iconEndpoint         = "endpoint"
	iconFocus            = "focus"
	iconPageUp           = "page_up"
	iconPageDown         = "page_down"
	iconAttach           = "attach"
	iconRun              = "run"
)

// icon returns the resolved glyph of one icon name ("" when unset, e.g. the
// ascii/unicode presets leave mode icons empty).
func (m *model) icon(name string) string { return m.icons[name] }

// iconLead is the "glyph " display prefix when the icon is set, else "".
func (m *model) iconLead(name string) string {
	if glyph := m.icons[name]; glyph != "" {
		return glyph + " "
	}
	return ""
}

// iconJoin combines an icon with a label ("<glyph> <label>", or just the
// label when the preset has no glyph for the name).
func (m *model) iconJoin(name, label string) string {
	return m.iconLead(name) + label
}

// modeIcon returns the glyph of one input mode.
func (m *model) modeIcon(mode mode) string {
	switch mode {
	case modeNormal:
		return m.icon(iconModeNormal)
	case modePane:
		return m.icon(iconModePane)
	case modeScroll:
		return m.icon(iconModeScroll)
	case modePicker:
		return m.icon(iconModePicker)
	case modePrompt:
		return m.icon(iconModePrompt)
	case modeHelp:
		return m.icon(iconModeHelp)
	}
	return ""
}

// buttonIcons resolves the fixed title-bar button group against the active
// icon set, preserving slotButtons order.
func (m *model) buttonIcons() []slotButton {
	names := map[string]string{
		slotButtonRestart: iconSlotRestart,
		slotButtonSplitH:  iconSlotSplitH,
		slotButtonSplitV:  iconSlotSplitV,
		slotButtonClose:   iconSlotClose,
	}
	out := make([]slotButton, 0, len(slotButtons))
	for _, action := range slotButtons {
		out = append(out, slotButton{action: action, icon: m.icon(names[action])})
	}
	return out
}

package main

import (
	"strconv"
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
)

// footerScene is one v2 footer scene. The scenes are the states the shell can
// actually enter (SCENARIOS): NORMAL, PANE, picker, prompt, help, scrollback
// and an exited focused terminal. There is deliberately no resize/system/
// floating/zoom/tab scene: v2 has no such modes, so their old recommended
// key groups must never reach the footer (RECOMMENDED_CONFIG "footer 规格表").
type footerScene string

const (
	footerSceneNormal footerScene = "NORMAL"
	footerScenePane   footerScene = "PANE"
	footerScenePicker footerScene = "PICKER"
	footerScenePrompt footerScene = "PROMPT"
	footerSceneHelp   footerScene = "HELP"
	footerSceneScroll footerScene = "SCROLL"
	footerSceneExited footerScene = "EXITED"
)

// footerItem is one key group entry. key is the resolved binding ("" when the
// label already embeds the old recommended key), icon a name in the icon
// preset ("" for no glyph) and label the old recommended text. Display is
// label-first: an explicit label wins, otherwise the normalized key name is
// shown (keyDisplay).
type footerItem struct {
	key   string
	icon  string
	label string
	style string
}

// text renders one footer entry ("KEY icon label" with empty parts dropped).
func (m *model) text(item footerItem) string {
	parts := make([]string, 0, 3)
	if key := strings.TrimSpace(item.key); key != "" {
		parts = append(parts, keyDisplay(key))
	}
	if glyph := m.icon(item.icon); glyph != "" {
		parts = append(parts, glyph)
	}
	if label := strings.TrimSpace(item.label); label != "" {
		parts = append(parts, label)
	}
	return strings.Join(parts, " ")
}

// keyDisplay is the normalized spelling of one key for the footer: the old
// recommended display names (SPACE, PGUP/PGDN, arrows) and upper-case letters,
// never a raw protocol/action name like "space" or "split-h".
func keyDisplay(key string) string {
	switch key {
	case "space":
		return "SPACE"
	case "page-up":
		return "PGUP"
	case "page-down":
		return "PGDN"
	case "up":
		return "↑"
	case "down":
		return "↓"
	case "left":
		return "←"
	case "right":
		return "→"
	case "enter":
		return "ENTER"
	case "esc":
		return "ESC"
	case "tab":
		return "TAB"
	case "backspace":
		return "BACKSPACE"
	}
	if strings.HasPrefix(key, "ctrl-") {
		return "CTRL-" + strings.ToUpper(strings.TrimPrefix(key, "ctrl-"))
	}
	if len(key) == 1 && key[0] >= 'a' && key[0] <= 'z' {
		return strings.ToUpper(key)
	}
	return key
}

// ctrlKey is the recommended-profile key spelling for the NORMAL/EXITED
// groups: the shared Ctrl modifier moves into the mode badge, so "ctrl-p"
// displays as "P" and any other binding keeps its normalized name.
func (m *model) ctrlKey(action string) string {
	key := m.binds.key(action)
	if rest, ok := strings.CutPrefix(key, "ctrl-"); ok && len(rest) == 1 {
		return strings.ToUpper(rest)
	}
	return keyDisplay(key)
}

// footerSceneFor returns the scene the footer must describe. The order is the
// v2 state machine order: overlays first, then scrollback, then the exited
// focused terminal, then PANE, else NORMAL.
func (m *model) footerSceneFor() footerScene {
	switch m.mode {
	case modePicker:
		return footerScenePicker
	case modePrompt:
		return footerScenePrompt
	case modeHelp:
		return footerSceneHelp
	case modeScroll:
		return footerSceneScroll
	case modePane:
		if m.focusedExited() {
			return footerSceneExited
		}
		return footerScenePane
	}
	if m.focusedExited() {
		return footerSceneExited
	}
	return footerSceneNormal
}

// focusedExited reports whether the focused slot holds an exited terminal.
func (m *model) focusedExited() bool {
	s := m.focusSlot()
	if s == nil || s.sourceID == "" {
		return false
	}
	src := m.sourceByID(s.sourceID)
	return src != nil && src.GetExited()
}

// badge returns the scene's mode badge (old recommended mode icon + label;
// prompt/help have no recommended mode entry and keep the legacy uppercase
// fallback).
func (m *model) badge(scene footerScene) string {
	switch scene {
	case footerSceneNormal, footerSceneExited:
		return m.iconJoin(iconModeNormal, "CTRL")
	case footerScenePane:
		return m.iconJoin(iconModePane, "PANE")
	case footerScenePicker:
		return m.iconJoin(iconModePicker, "PICK")
	case footerSceneScroll:
		return m.iconJoin(iconModeScroll, "COPY")
	case footerScenePrompt:
		return "PROMPT"
	case footerSceneHelp:
		return "HELP"
	}
	return ""
}

// footerSpec is the single recommended-profile source of truth for the left
// key group and the right summaries (mirrored by legacy.py). Every entry is
// the old yaml label/icon/order clipped to the actions v2 implements in that
// scene; unimplemented old scene actions (resize/system/floating/zoom/tab
// next-prev/kill/detach/…) are not mapped and never displayed.
func (m *model) footerSpec() (badge string, items []footerItem, summary bool) {
	switch m.footerSceneFor() {
	case footerScenePane:
		return m.badge(footerScenePane), []footerItem{
			{key: m.binds.key(actionSlotClose), icon: iconSlotClose, label: "CLOSE", style: "danger"},
			{key: m.binds.key(actionPaneSplitH), icon: iconSlotSplitH, label: "VSPLIT", style: "footer-key-resize"},
			{key: m.binds.key(actionPaneSplitV), icon: iconSlotSplitV, label: "HSPLIT", style: "footer-key-resize"},
			{key: "tab", icon: iconFocus, label: "FOCUS", style: "footer-key-tab"},
			{key: "esc", label: "BACK", style: "muted"},
		}, false
	case footerScenePicker:
		return m.badge(footerScenePicker), []footerItem{
			{label: "↑/↓ SELECT", style: "footer-key-picker"},
			{key: "enter", icon: iconAttach, label: "ATTACH", style: "footer-key-picker"},
			{key: "esc", label: "BACK", style: "muted"},
		}, false
	case footerScenePrompt:
		return m.badge(footerScenePrompt), []footerItem{
			{key: "enter", icon: iconRun, label: "RUN", style: "footer-key-global"},
		}, false
	case footerSceneHelp:
		return m.badge(footerSceneHelp), nil, false
	case footerSceneScroll:
		return m.badge(footerSceneScroll), []footerItem{
			{key: "page-up", icon: iconPageUp, label: "OLDER", style: "footer-key-copy"},
			{key: "page-down", icon: iconPageDown, label: "NEWER", style: "footer-key-copy"},
			{key: m.binds.key(actionScrollCopy), icon: iconModeScroll, label: "COPY", style: "footer-key-copy"},
			{key: "esc", label: "LIVE", style: "muted"},
		}, false
	case footerSceneExited:
		return m.badge(footerSceneExited), []footerItem{
			{key: m.ctrlKey(actionTerminalRestart), icon: iconSlotRestart, label: "RESTART", style: "footer-key-resize"},
			{key: m.ctrlKey(actionPickerOpen), icon: iconModePicker, label: "PICK", style: "footer-key-picker"},
		}, true
	default:
		return m.badge(footerSceneNormal), []footerItem{
			{key: m.ctrlKey(actionPaneMode), icon: iconModePane, label: "PANE", style: "footer-key-pane"},
			{key: m.ctrlKey(actionTabNew), icon: iconSummaryTab, label: "TAB", style: "footer-key-tab"},
			{key: m.ctrlKey(actionSidebarToggle), icon: iconWorkspace, label: "WORKSPACE", style: "footer-key-workspace"},
			{key: m.ctrlKey(actionPickerOpen), icon: iconModePicker, label: "PICK", style: "footer-key-picker"},
		}, true
	}
}

// footerLeftText is the exact left segment text (" badge " then one
// " ·  <item>" per entry, old recommended spacing).
func (m *model) footerLeftText() string {
	badge, items, _ := m.footerSpec()
	text := ""
	if badge != "" {
		text = " " + badge + " "
	}
	for _, item := range items {
		if text != "" {
			text += " · "
		}
		text += " " + m.text(item)
	}
	return text
}

// footerKeys is the ordered display text of the current scene's key group
// (badge excluded), kept for callers that only need the hint list.
func (m *model) footerKeys() []string {
	_, items, _ := m.footerSpec()
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, m.text(item))
	}
	return out
}

// footerRight is the right segment: the optional transient status notice
// followed by the old recommended summaries (workspace, floating count,
// terminal count) in the legacy template shape.
func (m *model) footerRight() string {
	_, _, summary := m.footerSpec()
	parts := make([]string, 0, 4)
	if m.status != "" {
		parts = append(parts, m.status)
	}
	if summary {
		parts = append(parts,
			m.iconJoin(iconSummaryWorkspace, m.workspaceName()),
			m.iconJoin(iconSummaryFloating, "0"),
			m.iconJoin(iconSummaryTerminals, strconv.Itoa(len(m.sources))),
		)
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ") + " "
}

// footerLine is the complete footer row (left, padding, right) as one string,
// so the golden tests and the view render the same bytes.
func (m *model) footerLine() string {
	left := m.footerLeftText()
	right := m.footerRight()
	pad := m.cols - sdk.DisplayWidth(left) - sdk.DisplayWidth(right)
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + right
}

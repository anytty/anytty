// Command v3shell is a Go replica of the legacy v3 TUI (the surface
// framework under tui/ on main) with the default recommended configuration
// (profile coralline-candy, tui/docs/tui-v3.recommended.yaml).
//
// The reference is the validated Python replica
// (clients/tui/examples/python-shell/v3ui.py + tui2sdk.widgets.chrome); this
// program keeps the same program-side state machine and pixel geometry on the
// Go SDK (clients/tui/sdk + sdk/app) and passes the same v3 golden suite.
//
// This file is the presentation policy: every style string, glyph, scene
// badge and footer action table, copied from the recommended yaml.
package main

import "strings"

// ---------------------------------------------------------------- palette
//
// coralline-candy tokens (tui-v3.recommended.yaml) as explicit style strings.
const (
	colorFG           = "#f8f4ff"
	colorPrimary      = "#f0abfc"
	colorSecondary    = "#3b2f63"
	colorBG           = "#070611"
	colorStatusBG     = "#17132a"
	colorOverlayBG    = "#261b44"
	colorMuted        = "#9ca3c9"
	colorSuccess      = "#86efac"
	colorWarning      = "#fde68a"
	colorDanger       = "#fb7185"
	colorInfo         = "#7dd3fc"
	colorTabInactive  = "#261b44"
	colorTabInactiveF = "#c4b5fd"
	colorCreateBG     = "#fde68a"
	colorCreateFG     = "#1b1230"
	colorWSFG         = "#1b1230"
	// footer-key-* mixed tokens (ansiForStyleToken + mixHostColor, same
	// rounding as the old renderer).
	colorFooterFloat = "#bcbdfc"
	colorFooterCopy  = "#83e5c8"
	colorFooterPick  = "#fc9a87"
	colorFooterGlob  = "#f5c0d4"
)

// style builds an explicit style string ("fg:...;bg:...;bold").
func style(fg, bg string, attrs ...string) string {
	out := ""
	add := func(part string) {
		if part == "" {
			return
		}
		if out != "" {
			out += ";"
		}
		out += part
	}
	if fg != "" {
		add("fg:" + fg)
	}
	if bg != "" {
		add("bg:" + bg)
	}
	for _, attr := range attrs {
		add(attr)
	}
	return out
}

var (
	stWSL        = style(colorPrimary, colorBG)
	stWSBody     = style(colorWSFG, colorPrimary, "bold")
	stWSR        = style(colorPrimary, colorSecondary)
	stTabL       = style(colorSecondary, colorPrimary)
	stTabBody    = style(colorWSFG, colorPrimary, "bold")
	stTabR       = style(colorPrimary, colorSecondary)
	stTabSpace   = style(colorFG, colorSecondary)
	stTabClose   = style(colorDanger, colorSecondary)
	stITabL      = style(colorSecondary, colorTabInactive)
	stITabBody   = style(colorTabInactiveF, colorTabInactive, "bold")
	stITabR      = style(colorTabInactive, colorSecondary)
	stITabSpace  = style(colorTabInactiveF, colorSecondary)
	stITabClose  = style(colorMuted, colorSecondary)
	stCreateL    = style(colorSecondary, colorCreateBG)
	stCreateBody = style(colorCreateFG, colorCreateBG, "bold")
	stCreateR    = style(colorCreateBG, colorBG)
	stHeaderFill = style(colorFG, colorBG)

	stAccent        = style(colorPrimary, "")
	stAccentGroup   = style(colorWSFG, colorPrimary, "bold")
	stPanelBorder   = style(colorSecondary, "")
	stInactiveGroup = style(colorTabInactiveF, colorSecondary, "bold")
	stMuted         = style(colorMuted, "")
	stSuccess       = style(colorSuccess, "")
	stWarning       = style(colorWarning, "")
	stDanger        = style(colorDanger, "")
	stDangerStrong  = style(colorDanger, "", "bold")
	stFooter        = style(colorMuted, "")
	stFooterAccent  = style(colorPrimary, "", "bold")

	stFooterKeyPane      = style(colorPrimary, "", "bold")
	stFooterKeyResize    = style(colorWarning, "", "bold")
	stFooterKeyTab       = style(colorInfo, "", "bold")
	stFooterKeyWorkspace = style(colorSuccess, "", "bold")
	stFooterKeyFloat     = style(colorFooterFloat, "", "bold")
	stFooterKeyCopy      = style(colorFooterCopy, "", "bold")
	stFooterKeyPicker    = style(colorFooterPick, "", "bold")
	stFooterKeyGlobal    = style(colorFooterGlob, "", "bold")
	stFooterInfo         = style(colorInfo, "")
	stFooterSuccess      = style(colorSuccess, "")
	stFooterDanger       = style(colorDanger, "")
	stFooterDangerStrong = style(colorDanger, "", "bold")
	stFooterFill         = style(colorFG, "")

	stOverlay = style(colorFG, colorOverlayBG)
	stContent = style(colorFG, "")
	stToast   = style(colorWarning, "")
	// stPickerMatch is main's `picker-match`: the warning foreground in bold,
	// used to mark query matches in picker rows (no background).
	stPickerMatch = style(colorWarning, "", "bold")
)

// ---------------------------------------------------------------- glyphs

const (
	wsIcon       = "\U000f0645" // nf-md-folder
	tabIcon      = "\u2387"     // ⎇
	tabClose     = "\U000f0156" // nf-md-close
	tabCreate    = "\U000f0415" // nf-md-plus
	edgeL        = "\ue0b6"
	edgeR        = "\ue0b0"
	edgeRoundR   = "\ue0b4"
	glyphLocked  = "\u25a0" // ■
	glyphUnlock  = "\u25a1" // □
	glyphZoom    = "\U000f004c"
	glyphSplitV  = "\ueb56"
	glyphSplitH  = "\ueb57"
	glyphClose   = "\U000f0156"
	glyphRunning = "\u25cf" // ●
	glyphCenter  = "\u25ce" // ◎
	glyphCollaps = "\u25be" // ▾
	glyphFloat   = "\U000f0e59"
	glyphTerm    = "\uf489"
	glyphTabs    = "\U000f04e9"
	glyphPanes   = "\uebeb"
	glyphGutter  = "\u2503" // ┃

	collapseHint = "Click to collapse"
)

// footerAction is one footer key group entry: the rendered label, its hit
// node (kept for parity with the reference; the old footer keys are hints),
// the canonical action id and the display key used by the color heuristic.
type footerAction struct {
	label string
	node  string
	id    string
	key   string
}

// sceneSpec is one footer scene: the mode badge (icon + label) and its action
// group, exactly the v3ui.py SCENES table.
type sceneSpec struct {
	icon    string
	label   string
	actions []footerAction
}

var scenes = map[string]sceneSpec{
	"live": {modeIconLive, "CTRL", []footerAction{
		{"P " + glyphPanes + " PANE", "f:ctrl-p", "menu.panel", "^P"},
		{"R \U000f0656 SIZE", "f:ctrl-r", "menu.resize", "^R"},
		{"O " + glyphFloat + " FLOAT", "f:ctrl-o", "menu.floating", "^O"},
		{"T " + glyphTabs + " TAB", "f:ctrl-t", "menu.tab", "^T"},
		{"W " + wsIcon + " WORKSPACE", "f:ctrl-w", "menu.workspace", "^W"},
		{"F " + modeIconPicker + " PICK", "f:ctrl-f", "terminal_picker.open", "^F"},
		{"\u21e7C \U000f0489 SELECT", "f:ctrl-shift-c", "copy.enter", "^C"},
		{"\u21e7H \U000f014c CLIPBOARD", "f:ctrl-shift-h", "menu.clipboard_history", "^H"},
		{"\u21e7V \U000f018f PASTE", "f:ctrl-shift-v", "clipboard.paste_system", "^V"},
		{"G \U000f0493 SYSTEM", "f:ctrl-g", "menu.system", "^G"},
	}},
	"pane": {modeIconPane, "PANE", []footerAction{
		{"X " + glyphClose + " CLOSE", "fs:pane:close", "panel.close", "x"},
		{"CTRL+D " + glyphSplitV + " VSPLIT", "fs:pane:split-h", "panel.split_right", "^D"},
		{"CTRL+E " + glyphSplitH + " HSPLIT", "fs:pane:split-v", "panel.split_down", "^E"},
		{"H/L \U000f0734 FOCUS", "fs:pane:focus", "panel.focus_prev", "h"},
		{"Q \U000f0688 KILL+CLOSE", "fs:pane:kill-close", "panel.kill_and_close", "q"},
	}},
	"resize": {modeIconResize, "SIZE", []footerAction{
		{"H \u2190 LEFT", "fs:resize:left", "resize.left", "h"},
		{"L \u2192 RIGHT", "fs:resize:right", "resize.right", "l"},
		{"K \u2191 UP", "fs:resize:up", "resize.up", "k"},
		{"J \u2193 DOWN", "fs:resize:down", "resize.down", "j"},
		{"S \U000f033e LOCK", "fs:resize:lock", "panel.size_lock", "s"},
		{"SPACE \U000f0636 LAYOUT", "fs:resize:layout", "resize.layout_toggle", "space"},
		{"R \U000f0410 RESET", "fs:resize:reset", "resize.layout_reset", "r"},
		{"= \U000f0555 BALANCE", "fs:resize:balance", "panel.balance", "="},
	}},
	"tab": {modeIconTab, "TAB", []footerAction{
		{"N \U000f04ad NEXT", "fs:tab:next", "tab.next", "n"},
		{"P \U000f04ae PREV", "fs:tab:prev", "tab.previous", "p"},
	}},
	"workspace": {wsIcon, "WORKSPACE", []footerAction{
		{"N \U000f04ad NEXT", "fs:ws:next", "workspace.next", "n"},
		{"P \U000f04ae PREV", "fs:ws:prev", "workspace.previous", "p"},
		{"T " + wsIcon + " TREE", "fs:ws:tree", "system.open_workbench_tree", "t"},
	}},
	"system": {modeIconSystem, "SYSTEM", []footerAction{
		{"P " + glyphTerm + " TERMINALS", "fs:sys:terminals", "system.open_terminal_pool", "p"},
		{"E \U000f0337 CONNECTIONS", "fs:sys:connections", "system.open_connections", "e"},
		{"W " + wsIcon + " TREE", "fs:sys:tree", "system.open_workbench_tree", "w"},
		{"O \uF4B5 COMMAND", "fs:sys:prompt", "system.open_prompt", "o"},
	}},
	"floating": {glyphFloat, "FLOAT", []footerAction{
		{"N \U000f0415 NEW", "fs:float:new", "floating.new", "n"},
		{"O " + glyphFloat + " OVERVIEW", "fs:float:overview", "floating.overview", "o"},
		{"F " + modeIconPicker + " PICK", "fs:float:pick", "system.open_terminal_picker", "f"},
		{"X " + glyphClose + " CLOSE", "fs:float:close", "floating.close", "x"},
		{"Z \U000f0615 HIDE", "fs:float:collapse", "floating.collapse", "z"},
	}},
	"terminal-picker": {modeIconPicker, "PICK", []footerAction{
		{"\u2190/\u2192 ENDPOINT", "", "terminal_picker.endpoint_previous", "\u2190"},
		{"\u2191/\u2193 SELECT", "", "terminal_picker.select_previous", "\u2191"},
		{"ENTER \U000f02fa ATTACH", "", "terminal_picker.attach", "enter"},
		{"ESC BACK", "", "shortcut.exit", "Esc"},
	}},
	"terminal-picker-tags": {modeIconPicker, "TAGS", []footerAction{
		{"\u2191/\u2193 SELECT", "", "terminal_picker.select_previous", "\u2191"},
		{"SPACE \U000f0636 TOGGLE", "", "terminal_picker.tag_toggle", "space"},
		{"CTRL+T DONE", "", "terminal_picker.tags", "^T"},
		{"ESC BACK", "", "shortcut.exit", "Esc"},
	}},
	"prompt": {modeIconPrompt, "", []footerAction{
		{"ENTER \U000f0627 RUN", "", "prompt.submit", "enter"},
	}},
	"help": {"", "", nil},
	"copy": {modeIconCopy, "COPY", []footerAction{
		{"PGUP \U000f005d OLDER", "fs:copy:older", "copy.request_older", "PgUp"},
		{"PGDN \U000f0045 NEWER", "fs:copy:newer", "copy.request_newer", "PgDn"},
		{"Y \U000f018f COPY", "fs:copy:copy", "copy.copy_selection", "y"},
		{"G \U000f005e OLDEST", "fs:copy:oldest", "copy.oldest", "g"},
	}},
}

const (
	modeIconLive   = "\U000f030c"
	modeIconCopy   = "\U000f018f"
	modeIconPane   = "\uebeb"
	modeIconResize = "\U000f0656"
	modeIconPicker = "\U000f0c7c"
	modeIconPrompt = "\U000f0627"
	modeIconHelp   = "\U000f02d6"
	modeIconTab    = "\U000f04e9"
	modeIconSystem = "\U000f0493"
)

// shortcutActionStyles is the yaml shortcuts.actions style override table.
var shortcutActionStyles = map[string]string{
	"menu.panel":             "footer-key-pane",
	"menu.resize":            "footer-key-resize",
	"menu.system":            "footer-key-global",
	"menu.floating":          "footer-key-float",
	"menu.tab":               "footer-key-tab",
	"menu.workspace":         "footer-key-workspace",
	"menu.copy":              "footer-key-copy",
	"menu.terminal_picker":   "footer-key-picker",
	"menu.terminal_pool":     "footer-key-picker",
	"menu.connections":       "info",
	"menu.workbench_tree":    "footer-key-workspace",
	"menu.prompt":            "footer-key-global",
	"menu.help":              "footer-key-tab",
	"system.toggle_header":   "info",
	"system.toggle_footer":   "success",
	"system.quit":            "danger-strong",
	"panel.close":            "danger",
	"panel.kill":             "danger-strong",
	"panel.kill_and_close":   "danger-strong",
	"clipboard.paste_latest": "success",
	"clipboard.paste_system": "info",
	"menu.clipboard_history": "footer-key-copy",
}

var footerStyleTokens = map[string]string{
	"footer-accent":        stFooterAccent,
	"footer-key-pane":      stFooterKeyPane,
	"footer-key-resize":    stFooterKeyResize,
	"footer-key-tab":       stFooterKeyTab,
	"footer-key-workspace": stFooterKeyWorkspace,
	"footer-key-float":     stFooterKeyFloat,
	"footer-key-copy":      stFooterKeyCopy,
	"footer-key-picker":    stFooterKeyPicker,
	"footer-key-global":    stFooterKeyGlobal,
	"info":                 stFooterInfo,
	"success":              stFooterSuccess,
	"warning":              stWarning,
	"danger":               stFooterDanger,
	"danger-strong":        stFooterDangerStrong,
}

// modeStyles is the yaml footer.modes per-scene badge token; unlisted scenes
// fall back to footer-accent, exactly like the old renderer.
var modeStyles = map[string]string{
	"live":                      "footer-accent",
	"copy":                      "footer-key-copy",
	"pane":                      "footer-key-pane",
	"resize":                    "footer-key-resize",
	"terminal-picker":           "footer-key-picker",
	"terminal-picker-endpoints": "footer-key-picker",
	"terminal-picker-tags":      "footer-key-picker",
	"workbench-tree":            "footer-key-workspace",
}

// footerFallbackStyle is the old footerActionKeyStyle() heuristic on the raw
// key + label text (v3ui.py footer_fallback_style).
func footerFallbackStyle(key, label string) string {
	upper := strings.ToUpper(key + " " + label)
	has := func(needle string) bool { return strings.Contains(upper, needle) }
	switch {
	case has("X") || has("KILL"):
		return stFooterKeyPicker
	case has("W") || has("WORKSPACE"):
		return stFooterKeyWorkspace
	case has("F") || has("PICK"):
		return stFooterKeyPicker
	case has("O") || has("FLOAT"):
		return stFooterKeyFloat
	case has("V") || has("COPY"):
		return stFooterKeyCopy
	case has("G") || has("GLOBAL"):
		return stFooterKeyGlobal
	case has("R") || has("RESIZE") || has("SIZE"):
		return stFooterKeyResize
	case has("P") || has("PANE"):
		return stFooterKeyPane
	case has("T") || has("TAB") || has("TREE"):
		return stFooterKeyTab
	}
	return stFooterAccent
}

// footerActionStyle is the old resolution chain: yaml override -> scene
// default (StatusAccent/StatusWarning) -> footerActionDisplayStyle -> the
// heuristic above.
func footerActionStyle(a footerAction) string {
	if token, ok := shortcutActionStyles[a.id]; ok {
		if value, ok := footerStyleTokens[token]; ok {
			return value
		}
	}
	return footerFallbackStyle(a.key, a.label)
}

// promptCommands is the prompt scene's command catalog.
var promptCommands = []string{
	"split row", "split col", "close pane", "kill pane", "new tab",
	"close tab", "help", "quit",
}

// helpLines is the help overlay body (v3ui.py HELP_LINES).
var helpLines = []string{
	"全局 (global)",
	"  Ctrl-P PANE   Ctrl-R RESIZE   Ctrl-O FLOAT",
	"  Ctrl-T TAB    Ctrl-W WORKSPACE Ctrl-F PICK",
	"  Ctrl-G SYSTEM \u21e7C copy \u21e7H clipboard \u21e7V paste",
	"PANE",
	"  x close  % / Ctrl-D vsplit  \" / Ctrl-E hsplit",
	"  h/l focus  t restart  k kill  q kill+close  z collapse",
	"TAB / WORKSPACE",
	"  c create  n/p next/prev  1-9 jump  x close",
	"SYSTEM",
	"  q quit  o command  ? help",
	"esc back \u00b7 Ctrl-Q quit",
}

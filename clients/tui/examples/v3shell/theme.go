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
	// colorExtentPlaceholder is the legacy `extent_placeholder_style` token
	// (#3b2f63): the dim dot that fills a frozen copy window which is shorter
	// than its pane. It is intentionally its own name even though the
	// recommended yaml reuses the secondary surface color.
	colorExtentPlaceholder = "#3b2f63"
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
	// stPickerMatch is main's `picker-match`: the warning foreground in bold,
	// used to mark query matches in picker rows (no background).
	stPickerMatch = style(colorWarning, "", "bold")
	// stHistoryBorder is the legacy `history-border` token: the warning
	// foreground in bold. The old paneChromeStyle returned it whenever a panel's
	// content kind was copy-history (checked before Active), so a frozen
	// scrollback panel's frame stays in the copy warning color regardless of
	// whether the panel owns focus. Title/action glyphs keep their own accent.
	stHistoryBorder = style(colorWarning, "", "bold")
	// stOverflowStyle is the recommended `overflow_style` token (#9ca3c9): the
	// muted color of the ◂ ▸ ▴ ▾ content-clipping markers the legacy renderer
	// drew on a pane's border (render/content_overflow_marker.go).
	stOverflowStyle = style(colorMuted, "")
	// stExtentPlaceholder is the recommended `extent_placeholder_style` token
	// (#3b2f63): the dim dot that masks the pane behind a frozen copy window
	// shorter than the content area (render/content_viewport.go, the extent
	// dots the live surface uses when its size is smaller than the pane).
	stExtentPlaceholder = style(colorExtentPlaceholder, "")
)

const (
	// terminalPropContentOffset / terminalPropContentSize / terminalPropPlaceholder
	// are the program-declared terminal content-framing props (PROTOCOL §5): the
	// extent origin, its footprint size and the outside-footprint fill style. The
	// host passes them through to the terminal component untouched; they must stay
	// in sync with the component's PropContentOffset/PropContentSize/PropPlaceholder.
	terminalPropContentOffset = "content.offset"
	terminalPropContentSize   = "content.size"
	terminalPropPlaceholder   = "chrome.placeholder"
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
	// glyphLayoutAdjusted prefixes the pane title when the view-local content
	// layout is non-default (legacy paneChromeTerminalTitlePrefix / "◇ ").
	glyphLayoutAdjusted = "\u25c7" // ◇

	// Content-clipping markers (recommended yaml pane_glyphs.overflow_*): drawn
	// on the pane border when the frozen copy window is clipped.
	glyphOverflowLeft   = "\u25c2" // ◂
	glyphOverflowRight  = "\u25b8" // ▸
	glyphOverflowTop    = "\u25b4" // ▴
	glyphOverflowBottom = "\u25be" // ▾
	// extentPlaceholder marks the pane area a short frozen copy window does not
	// cover (recommended yaml pane_glyphs.extent_placeholder).
	extentPlaceholder = "\u00b7" // ·

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
		// H/L/K/J are grouped into one token (the old TUI compacts the resize
		// resize/focus groups) so the content-layout keys below stay visible at
		// 120 columns instead of being truncated away.
		{"H/L/K/J SIZE", "fs:resize:left", "resize.left", "h"},
		{"S \U000f033e LOCK", "fs:resize:lock", "panel.size_lock", "s"},
		{"SPACE \U000f0636 LAYOUT", "fs:resize:layout", "resize.layout_toggle", "space"},
		{"R \U000f0410 RESET", "fs:resize:reset", "resize.layout_reset", "r"},
		{"= \U000f0555 BALANCE", "fs:resize:balance", "panel.balance", "="},
		// Content-layout keys (legacy resize.align_*/center*/pan_*). The old
		// TUI grouped them into single ALIGN/CENTER/PAN footer tokens so the
		// full set is discoverable without opening the `?` help overlay.
		{"0/$/^/B ALIGN", "fs:resize:align", "resize.align_left", "0"},
		{"m/|/_ CENTER", "fs:resize:center", "resize.center", "m"},
		{"A/S/W/D PAN", "fs:resize:pan", "resize.pan_left", "A"},
	}},
	"tab": {modeIconTab, "TAB", []footerAction{
		{"N \U000f04ad NEXT", "fs:tab:next", "tab.next", "n"},
		{"P \U000f04ae PREV", "fs:tab:prev", "tab.previous", "p"},
	}},
	"workspace": {wsIcon, "WORKSPACE", []footerAction{
		{"N \U000f04ad NEXT", "fs:ws:next", "workspace.next", "n"},
		{"P \U000f04ae PREV", "fs:ws:prev", "workspace.previous", "p"},
		// T opens the read-only workbench tree overlay (legacy
		// system.open_workbench_tree), so the TREE label is honest; the node
		// and action id keep that identity for click routing.
		{"T " + wsIcon + " TREE", "fs:ws:tree", "system.open_workbench_tree", "t"},
	}},
	"system": {modeIconSystem, "SYSTEM", []footerAction{
		{"P " + glyphTerm + " TERMINALS", "fs:sys:terminals", "system.open_terminal_pool", "p"},
		{"E \U000f0337 CONNECTIONS", "fs:sys:connections", "system.open_connections", "e"},
		// W opens the same read-only workbench tree overlay.
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
	"connections": {modeIconSystem, "SYSTEM", []footerAction{
		{"\u2191/\u2193 SELECT", "", "connections.select_previous", "\u2191"},
		{"T \U000f0337 TEST", "", "connections.test", "t"},
		{"R \U000f0450 RECONNECT", "", "connections.reconnect", "r"},
		{"ESC BACK", "", "shortcut.exit", "Esc"},
	}},
	"copy": {modeIconCopy, "COPY", []footerAction{
		{"PGUP \U000f005d OLDER", "fs:copy:older", "copy.request_older", "PgUp"},
		{"PGDN \U000f0045 NEWER", "fs:copy:newer", "copy.request_newer", "PgDn"},
		{"Y \U000f018f COPY", "fs:copy:copy", "copy.copy_selection", "y"},
		{"G \U000f005e OLDEST", "fs:copy:oldest", "copy.oldest", "g"},
	}},
	// clipboard is the Ctrl-Shift-H history overlay. It is a dedicated scene
	// (not "copy"): the handler only supports esc/up/down/enter, so the footer
	// must not advertise the copy scene's PGUP/PGDN/Y/G keys.
	"clipboard": {modeIconCopy, "CLIPBOARD", []footerAction{
		{"\u2191/\u2193 SELECT", "", "clipboard_history.select_previous", "\u2191"},
		{"ENTER \U000f018f PASTE", "", "clipboard_history.paste", "enter"},
		{"ESC BACK", "", "shortcut.exit", "Esc"},
	}},
	// workbench-tree is the read-only workbench navigator (legacy
	// system.open_workbench_tree): workspaces with their tabs, Enter jumps.
	"workbench-tree": {wsIcon, "TREE", []footerAction{
		{"\u2191/\u2193 SELECT", "", "workbench_tree.select_previous", "\u2191"},
		{"ENTER \U000f0734 JUMP", "", "workbench_tree.open", "enter"},
		{"ESC BACK", "", "shortcut.exit", "Esc"},
	}},
	// log is the bounded message log overlay: the on-demand replacement for the
	// top-right toast card the legacy suppressed. ↑/↓ scroll/select, esc closes.
	// It reuses the system badge icon (honest: the overlay is reached from
	// SYSTEM with `g` and from the `:` palette with `logs`).
	"log": {modeIconSystem, "LOG", []footerAction{
		{"\u2191/\u2193 SCROLL", "", "log.select_previous", "\u2191"},
		{"ESC BACK", "", "shortcut.exit", "Esc"},
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
	"clipboard":                 "footer-key-copy",
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
	"close tab", "help", "logs", "quit",
}

// helpLines is the help overlay body (v3ui.py HELP_LINES, extended with the
// RESIZE and COPY scenes the replica implements). The footer hint bar stays a
// curated subset pinned to the Python reference golden; the `?` overlay is the
// canonical full shortcut list, so every key that has no footer slot is listed
// here, matching the legacy buildHelpContent rule that enumerates all
// configured bindings per scene.
var helpLines = []string{
	"全局 (global)",
	"  Ctrl-P PANE   Ctrl-R RESIZE   Ctrl-O FLOAT",
	"  Ctrl-T TAB    Ctrl-W WORKSPACE Ctrl-F PICK",
	"  Ctrl-G SYSTEM \u21e7C copy \u21e7H clipboard \u21e7V paste",
	"PANE",
	"  x close  % / Ctrl-D vsplit  \" / Ctrl-E hsplit",
	"  h/l focus  t restart  k kill  q kill+close  z collapse",
	"RESIZE",
	"  h/l/k/j \u8c03\u6574 \u00b12   H/L/K/J \u00b16   s \u9501\u5c3a\u5bf8   space \u5207\u5206\u65b9\u5411",
	"  0/$ \u5de6\u53f3\u5bf9\u9f50   ^/B \u4e0a\u4e0b\u5bf9\u9f50   m \u5c45\u4e2d   |/_ \u5355\u8f74\u5c45\u4e2d",
	"  A/S/W/D \u6216 shift+\u65b9\u5411 \u5e73\u79fb   M \u5185\u5bb9\u6a21\u5f0f   r \u91cd\u7f6e   = \u5e73\u8861",
	"  \u6ce8\uff1aCtrl+\u65b9\u5411/Alt+\u5b57\u6bcd \u88ab\u8fd0\u884c\u65f6\u5f52\u4e00\uff0cpan \u4ec5\u7ed1 A/S/W/D",
	"  \u5185\u5bb9\u533a\u5168\u5e45\uff0cPTY \u5c3a\u5bf8\u4e0d\u53d8\uff1b\u975e\u9ed8\u8ba4\u5e03\u5c40\u65f6\u6807\u9898\u524d\u51fa\u73b0 \u25c7",
	"TAB / WORKSPACE",
	"  c create  n/p next/prev  1-9 jump  x close",
	"COPY",
	"  j/k \u79fb\u52a8  space \u6807\u8bb0  y \u590d\u5236  enter \u590d\u5236\u5e76\u9000\u51fa",
	"  / \u641c\u7d22  n/N \u4e0a\u4e0b\u5339\u914d  PgUp/PgDn \u7ffb\u9875  g \u6700\u8001  G \u56de\u5230live",
	"  \u62d6\u62fd\u9009\u5230\u4e0a\u4e0b\u8fb9\u7f18\u81ea\u52a8\u7ffb\u9875\u5e76\u5ef6\u4f38\u9009\u533a",
	"SYSTEM",
	"  q quit  o command  g logs  ? help",
	"LOGS / \u6d88\u606f\u65e5\u5fd7",
	"  g \u6216 :logs \u6253\u5f00 \u00b7 \u65e7\u7248\u53f3\u4e0a\u89d2 toast \u6d88\u606f\u5728\u6b64\u805a\u96c6",
	"  \u2191/\u2193/PgUp/PgDn \u6eda\u52a8\u9009\u62e9 \u00b7 \u6700\u65b0\u5728\u4e0b \u00b7 \u4e0a\u9650 200 \u884c",
	"esc back \u00b7 Ctrl-Q quit",
}

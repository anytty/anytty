package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/widgets"
)

// sidebarWidth is the fixed sidebar width in cells.
const sidebarWidth = 24

// view materializes the current model as a box tree plus its claim. Every
// appearance decision lives here: the kernel only solves geometry and the
// host only translates the explicit styles from the program-side theme into
// SGR (it holds no palette of its own).
func (m *model) view() (*sdk.Builder, sdk.Keys) {
	root := sdk.Col(m.header(), m.body(), m.footer())
	if overlay := m.overlay(); overlay != nil {
		root.Child(overlay)
	}
	return root, m.keys()
}

// keys derives the claim from the bindings, so a rebinding keeps routing
// consistent without touching the host.
func (m *model) keys() sdk.Keys {
	if m.mode != modeNormal {
		return sdk.Keys{All: true}
	}
	return sdk.Keys{Claim: []string{
		m.binds.key(actionPaneMode),
		m.binds.key(actionPickerOpen),
		m.binds.key(actionTabNew),
		m.binds.key(actionSidebarToggle),
		m.binds.key(actionTerminalRestart),
		"page-up",
		"page-down",
	}}
}

// header is the tab strip: workspace chip, one clickable box per tab
// (active tab highlighted with tab_active) and the "+" new-tab box. The
// recommended icon preset prefixes the workspace chip, every tab title and
// the new-tab box with the legacy Nerd Font glyphs.
func (m *model) header() *sdk.Builder {
	items := make([]widgets.TabItem, 0, len(m.tabs))
	for i, t := range m.tabs {
		items = append(items, widgets.TabItem{
			ID:     "tab:" + strconv.Itoa(i),
			Title:  m.iconJoin(iconTabMarker, strconv.Itoa(i+1)+":"+t.name),
			Active: i == m.active,
		})
	}
	plus := " + "
	if m.icon(iconTabNew) != "" {
		plus = " " + m.icon(iconTabNew) + " "
	}
	return widgets.TabBar{
		Left:          &widgets.Segment{Text: " " + m.iconJoin(iconWorkspace, m.workspaceName()) + " ", Style: m.style("chrome"), ID: "workspace"},
		Items:         items,
		Plus:          true,
		PlusText:      plus,
		ActiveStyle:   m.style("tab_active"),
		InactiveStyle: m.style("tab_inactive"),
		ChromeStyle:   m.style("chrome"),
	}.Build().ID("header").Height(1)
}

func (m *model) workspaceName() string {
	if m.viewID == "" {
		return "local"
	}
	parts := strings.Split(m.viewID, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return m.viewID
}

func (m *model) body() *sdk.Builder {
	width, height := m.bodySize()
	body := sdk.Row().ID("body").Height(height)
	if m.sidebar {
		body.Child(m.sidebarBox(height))
	}
	body.Child(m.panesBox(width, height))
	return body
}

// sidebarBox is the optional navigation column: workspace list, tab list and
// the focused slot summary (CUSTOMIZE §3). The panel chrome is drawn by the
// program-side Frame widget (the kernel has no border).
func (m *model) sidebarBox(height int) *sdk.Builder {
	rows := make([]widgets.FrameRow, 0, height)
	line := func(text, style string) {
		rows = append(rows, widgets.FrameRow{Text: text, Style: m.style(style)})
	}
	line(m.iconJoin(iconWorkspace, "workspaces"), "muted")
	line("> "+m.workspaceName(), "accent")
	line("", "")
	line(m.iconJoin(iconSummaryTab, "tabs"), "muted")
	for i, t := range m.tabs {
		marker := "  "
		style := "fg"
		if i == m.active {
			marker = "> "
			style = "selection"
		}
		rows = append(rows, widgets.FrameRow{
			Text:  marker + strconv.Itoa(i+1) + ":" + t.name,
			Style: m.style(style),
			ID:    sidebarTabNode(i),
			Input: []string{"mouse"},
		})
	}
	line("", "")
	line(m.iconJoin(iconSummarySlot, "slot"), "muted")
	t := m.activeTab()
	slot := m.focusSlot()
	label := "empty"
	style := "muted"
	if slot.sourceID != "" {
		label = shortSourceID(slot.sourceID)
		style = "ok"
		if src := m.sourceByID(slot.sourceID); src != nil && src.GetExited() {
			style = "danger"
			label += " [exited]"
		}
	}
	line(fmt.Sprintf("%d/%d %s", t.focus+1, len(t.slots), label), style)
	return widgets.Frame{
		ID:     "sidebar",
		Title:  "nav",
		Width:  sidebarWidth,
		Height: height,
		Style:  m.style("border"),
		Rows:   rows,
	}.Build()
}

// panesBox lays the slots of the active tab out with a 1-cell gutter when
// the configuration keeps gap=1.
func (m *model) panesBox(width, height int) *sdk.Builder {
	t := m.activeTab()
	panes := sdk.Box().ID("panes").Flow(t.flow).Width(width).Height(height)
	ratios := make([]int, len(t.slots))
	for i, s := range t.slots {
		ratios[i] = s.ratio
	}
	gap := m.gap
	if t.flow == "row" {
		sizes := widgets.Distribute(width-(len(t.slots)-1)*gap, ratios)
		for i, s := range t.slots {
			panes.Child(m.slotBox(s, i, sizes[i], height))
			if i < len(t.slots)-1 && gap > 0 {
				panes.Child(widgets.Divider{
					ID:       dividerID(i),
					Vertical: true,
					Length:   height,
					Style:    m.style("muted"),
					Input:    []string{"mouse"},
				}.Build())
			}
		}
		return panes
	}
	sizes := widgets.Distribute(height-(len(t.slots)-1)*gap, ratios)
	for i, s := range t.slots {
		panes.Child(m.slotBox(s, i, width, sizes[i]))
		if i < len(t.slots)-1 && gap > 0 {
			panes.Child(widgets.Divider{
				ID:     dividerID(i),
				Length: width,
				Style:  m.style("muted"),
				Input:  []string{"mouse"},
			}.Build())
		}
	}
	return panes
}

func dividerID(index int) string { return "divider:" + strconv.Itoa(index) }

// sidebarTabNode is the stable hit-test id of one sidebar tab row; the mouse
// handler maps a press back to selectTab(index).
func sidebarTabNode(index int) string { return "side:tab:" + strconv.Itoa(index) }

func (m *model) slotBox(s *slot, index, width, height int) *sdk.Builder {
	if s.sourceID != "" {
		box := sdk.Terminal(s.sourceID).ID(s.id).Width(width).Height(height)
		box.Input("key", "paste", "wheel")
		box.Props(m.terminalProps())
		box.Focused(m.terminalFocused(s, index))
		box.Child(m.slotTitleBar(s, width))
		return box
	}
	card := widgets.Card{
		ID:     s.id,
		Title:  "空槽",
		Lines:  []string{"Ctrl-F 选择终端", "Ctrl-P 面板命令"},
		Width:  width,
		Height: height,
		Style:  m.style("border"),
		Center: true,
	}.Build().Input("mouse")
	card.Child(m.slotTitleBar(s, width))
	return card
}

// slotTitleBar is the program-drawn 1-row title bar of one slot: the focus
// marker plus title (and badges) on the left, the action buttons on the
// right. It is a `pos` subtree at (0, 0), so it composites above the terminal
// component's own border title while the slot geometry (and therefore the PTY
// size) stays untouched. The kernel only hit-tests; every button is a box the
// program resolves itself.
func (m *model) slotTitleBar(s *slot, width int) *sdk.Builder {
	row := sdk.Row().ID("title:" + s.id).Width(width).Height(1)
	buttons := m.buttonIcons()
	group := 2*len(buttons) - 1
	if width < group+1 {
		// Too narrow for the group: degrade to a plain labeled row.
		if text := sdk.Truncate(m.slotLabel(s), width); text != "" {
			row.Child(sdk.Text(text).Style(m.slotLabelStyle(s)).Height(1))
		}
		return row.Pos(0, 0)
	}
	budget := width - group
	remaining := budget
	for _, run := range m.slotLabelRuns(s) {
		if remaining <= 0 {
			break
		}
		text := sdk.Truncate(run.text, remaining)
		if text == "" {
			continue
		}
		row.Child(sdk.Text(text).Style(run.style).Height(1))
		remaining -= sdk.DisplayWidth(text)
	}
	if spacer := width - (budget - remaining) - group; spacer > 0 {
		row.Child(sdk.Text(strings.Repeat(" ", spacer)).Height(1))
	}
	for i, button := range buttons {
		if i > 0 {
			row.Child(sdk.Text(" "))
		}
		node := slotButtonNode(s.id, button.action)
		row.Child(sdk.Text(button.icon).
			ID(node).
			Style(m.slotButtonStyle(s, button.action, node)).
			Width(1).
			Height(1).
			Input("mouse"))
	}
	return row.Pos(0, 0)
}

// slotLabelRun is one styled piece of the title bar label.
type slotLabelRun struct {
	text  string
	style string
}

// slotLabelRuns builds the left label: "▎title" plus the exit badge, the
// scrollback badge and the pending indicator, each with its own style. The
// title colors follow the existing theme (border_focus focused, border_dead
// exited, muted otherwise); badges keep the warning color of the component.
func (m *model) slotLabelRuns(s *slot) []slotLabelRun {
	marker := m.icon(iconSlotMarker)
	if marker == "" {
		marker = "▎"
	}
	runs := []slotLabelRun{{text: marker + m.slotTitle(s), style: m.slotLabelStyle(s)}}
	if src := m.sourceByID(s.sourceID); src != nil && src.GetExited() {
		runs = append(runs, slotLabelRun{
			text:  " [exited " + strconv.Itoa(int(src.GetExitCode())) + "]",
			style: m.style("warning"),
		})
	}
	if s.scrollOffset > 0 {
		runs = append(runs, slotLabelRun{
			text:  " [↑" + strconv.Itoa(s.scrollOffset) + "]",
			style: m.style("warning"),
		})
	}
	if s.pending != "" {
		runs = append(runs, slotLabelRun{text: " …", style: m.style("muted")})
	}
	return runs
}

// slotLabel is the whole left label as one plain string (measurement and the
// narrow-slot fallback).
func (m *model) slotLabel(s *slot) string {
	var b strings.Builder
	for _, run := range m.slotLabelRuns(s) {
		b.WriteString(run.text)
	}
	return b.String()
}

// slotTitle is the display title of a slot: the source title (the host's
// terminal id), or the empty-slot placeholder.
func (m *model) slotTitle(s *slot) string {
	if s.sourceID == "" {
		return "空槽"
	}
	if src := m.sourceByID(s.sourceID); src != nil && src.GetTitle() != "" {
		return src.GetTitle()
	}
	return shortSourceID(s.sourceID)
}

// slotLabelStyle is the title color: exited wins over focused (same
// precedence as the terminal component border).
func (m *model) slotLabelStyle(s *slot) string {
	if src := m.sourceByID(s.sourceID); src != nil && src.GetExited() {
		return m.style("border_dead")
	}
	if m.mode == modeNormal && s == m.focusSlot() && m.sourceAttached(s.sourceID) {
		return m.style("border_focus")
	}
	return m.style("muted")
}

// slotButtonStyle is the theme style of one title-bar button: the held button
// wins (button_pressed), then the exited-restart hint (warning: the terminal
// can be restarted), then the buttons of the focused terminal (button_hover;
// the program has no hover channel, so slot focus is the emphasis).
func (m *model) slotButtonStyle(s *slot, action, node string) string {
	if m.pressed == node {
		return m.style("button_pressed")
	}
	if action == slotButtonRestart {
		if src := m.sourceByID(s.sourceID); src != nil && src.GetExited() {
			return m.style("warning")
		}
	}
	if m.mode == modeNormal && s == m.focusSlot() && m.sourceAttached(s.sourceID) {
		return m.style("button_hover")
	}
	return m.style("button")
}

func (m *model) terminalFocused(s *slot, index int) bool {
	return m.mode == modeNormal && index == m.activeTab().focus && m.sourceAttached(s.sourceID)
}

// footer shows the current scene's key group on the left and the recommended
// summaries on the right (footer.go is the single spec table). The row is the
// old recommended shape: " badge " then " ·  <item>" per entry, right-aligned
// summaries, and the old padding rule (at least one fill cell).
func (m *model) footer() *sdk.Builder {
	badge, items, _ := m.footerSpec()
	left := sdk.Row().Height(1)
	if badge != "" {
		left.Child(sdk.Text(" " + badge + " ").Style(m.style("footer-accent")))
	}
	for _, item := range items {
		left.Child(sdk.Text(" · ").Style(m.style("muted")))
		style := item.style
		if style == "" {
			style = "muted"
		}
		left.Child(sdk.Text(" " + m.text(item)).Style(m.style(style)))
	}
	left.ID("footer.keys")

	right := m.footerRight()
	pad := m.cols - sdk.DisplayWidth(m.footerLeftText()) - sdk.DisplayWidth(right)
	if pad < 1 {
		pad = 1
	}

	row := sdk.Row().ID("footer").Height(1)
	row.Child(left)
	if pad > 0 {
		row.Child(sdk.Text(strings.Repeat(" ", pad)))
	}
	if right != "" {
		row.Child(sdk.Text(right).ID("footer.status").Style(m.style("status")))
	}
	return row
}

// ---------------------------------------------------------------- overlays

// overlayRow is one line of a centered overlay.
type overlayRow struct {
	text  string
	style string
	id    string
}

func (m *model) overlay() *sdk.Builder {
	switch m.mode {
	case modePicker:
		return m.pickerOverlay()
	case modePrompt:
		return m.promptOverlay()
	case modeHelp:
		return m.helpOverlay()
	}
	return nil
}

func (m *model) pickerOverlay() *sdk.Builder {
	items := m.pickerItems()
	grouped := m.pickerGrouped()
	visible := minInt(len(items), maxInt(1, m.rows-8))
	start := 0
	if m.pickerIdx >= visible {
		start = m.pickerIdx - visible + 1
	}
	rows := []overlayRow{
		{},
		{" select a terminal", m.style("muted"), ""},
		{},
	}
	lastGroup := ""
	for i := start; i < start+visible && i < len(items); i++ {
		if grouped {
			group := pickerGroupName(items[i])
			if group != lastGroup {
				rows = append(rows, overlayRow{text: "  " + group, style: m.style("muted")})
				lastGroup = group
			}
		}
		marker := "   "
		style := m.style("fg")
		if i == m.pickerIdx {
			marker = " > "
			style = m.style("selection")
		}
		rows = append(rows, overlayRow{
			text:  marker + pickerLine(items[i], 30),
			style: style,
			id:    "pick:" + strconv.Itoa(i),
		})
	}
	rows = append(rows,
		overlayRow{},
		overlayRow{text: " enter bind · esc close", style: m.style("muted")},
	)
	return m.overlayBox("terminals", m.iconJoin(iconModePicker, "terminals"), 58, rows)
}

// pickerGrouped reports whether the picker renders endpoint group headers.
// It turns on when a daemon endpoint is configured or the host publishes an
// endpoint placeholder source, so the v1 command layout keeps its flat
// pixel-exact rows.
func (m *model) pickerGrouped() bool {
	for i := range m.endpoints {
		if m.endpoints[i].isDaemon() {
			return true
		}
	}
	if len(m.endpointSources) > 0 {
		return true
	}
	return false
}

// pickerGroupName is the group header of one picker row.
func pickerGroupName(item pickerItem) string {
	switch {
	case item.source != nil:
		if endpoint := item.source.GetEndpoint(); endpoint != "" {
			return endpoint
		}
		return "local"
	case item.endpoint != nil:
		return item.endpoint.name
	default:
		return "local"
	}
}

// pickerLine pads the label and appends the short endpoint/state info
// column. The 21-cell info budget fits the longest implemented status
// ("endpoint · daemon tcp") inside the 56-cell picker body.
func pickerLine(item pickerItem, labelWidth int) string {
	label := sdk.Truncate(item.label, labelWidth)
	pad := labelWidth - sdk.DisplayWidth(label)
	info := sdk.Truncate(item.info, 21)
	return label + strings.Repeat(" ", maxInt(1, pad+2)) + info
}

func (m *model) promptOverlay() *sdk.Builder {
	matches := m.commandMatches()
	rows := []overlayRow{
		{},
		{text: " > " + m.promptText, style: m.style("fg"), id: "prompt:input"},
		{},
	}
	for i, cmd := range matches {
		marker := "   "
		style := m.style("fg")
		if i == m.promptIdx {
			marker = " > "
			style = m.style("selection")
		}
		rows = append(rows, overlayRow{text: marker + cmd, style: style, id: "prompt:" + strconv.Itoa(i)})
	}
	if len(matches) == 0 {
		rows = append(rows, overlayRow{text: "   (no matching command)", style: m.style("muted")})
	}
	rows = append(rows,
		overlayRow{},
		overlayRow{text: " enter run · esc close", style: m.style("muted")},
	)
	return m.overlayBox("command", m.iconJoin(iconModePrompt, "command"), 56, rows)
}

func (m *model) helpOverlay() *sdk.Builder {
	rows := []overlayRow{
		{},
		{text: " NORMAL", style: m.style("accent")},
		{text: "   Ctrl-P  pane mode          Ctrl-F  terminal picker"},
		{text: "   Ctrl-T  new tab            Ctrl-W  sidebar"},
		{text: "   Ctrl-E  restart terminal   PgUp/PgDn  scrollback"},
		{},
		{text: " PANE (prefix mode)", style: m.style("accent")},
		{text: "   %  split side by side      \"  split stacked"},
		{text: "   x  close slot (unbind)     Tab  next slot"},
		{text: "   1..9  switch tab           :  command  ?  help"},
		{},
		{text: " SLOT TITLE BAR (mouse)", style: m.style("accent")},
		{text: "   restart   VSPLIT   HSPLIT   close"},
		{},
		{text: " SCROLL", style: m.style("accent")},
		{text: "   PgUp/PgDn scroll   y copy   esc live"},
		{},
		{text: " Double-click Ctrl-F forwards the 2nd key to the terminal."},
		{},
		{text: " esc close", style: m.style("muted")},
	}
	return m.overlayBox("help", m.iconJoin(iconModeHelp, "help"), 64, rows)
}

// overlayBox centers a program-side framed overlay: the chrome is drawn by
// the Frame widget (explicit styles), and each row is its own styled box so
// one line can carry the selection background.
func (m *model) overlayBox(id, title string, width int, rows []overlayRow) *sdk.Builder {
	width = minInt(width, maxInt(8, m.cols-2))
	if maxRows := maxInt(1, m.rows-4); len(rows) > maxRows {
		rows = rows[:maxRows]
	}
	height := len(rows) + 2
	x := maxInt(0, (m.cols-width)/2)
	y := maxInt(0, (m.rows-height)/2)
	frameRows := make([]widgets.FrameRow, 0, len(rows))
	for _, row := range rows {
		frameRow := widgets.FrameRow{Text: row.text, Style: row.style}
		if row.id != "" {
			frameRow.ID = row.id
			frameRow.Input = []string{"mouse"}
		}
		frameRows = append(frameRows, frameRow)
	}
	return widgets.Frame{
		ID:     "overlay:" + id,
		Title:  title,
		Width:  width,
		Height: height,
		Style:  m.style("accent"),
		Rows:   frameRows,
	}.Build().Pos(x, y)
}

var commandNames = []string{"help", "split", "splitv", "close", "tab", "restart", "kill", "copy", "live", "quit"}

func (m *model) commandMatches() []string {
	prefix := strings.ToLower(strings.TrimSpace(m.promptText))
	matches := make([]string, 0, len(commandNames))
	for _, name := range commandNames {
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, name)
		}
	}
	return matches
}

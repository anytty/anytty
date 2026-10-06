package main

import (
	"fmt"
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/widgets"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// modeKeys is the action legend shown in the mode bar (SPEC §3/§4).
var modeKeys = map[string][]string{
	modePrefix: {
		"v split", "- split", "c tab", "w navigate", "z zoom", "? help", "q detach",
	},
	modeNavigate: {
		"↑/↓ workspace", "h/j/k/l pane", "enter activate", "esc cancel",
	},
	modeResize: {
		"h/j/k/l adjust 5%", "enter/esc done",
	},
	modeScroll: {
		"j/k line", "page-up/down page", "y copy", "q/esc live",
	},
}

// agentRow is one flattened Agents-list entry: a pane of any workspace.
type agentRow struct {
	ws    *workspace
	tab   *tab
	pane  *pane
	index int
}

// agentGroup is the panes bound to one machine (source endpoint).
type agentGroup struct {
	machine string
	rows    []agentRow
}

// unboundMachine is the Agents group for panes without a live source: empty
// panes and panes whose bound source vanished from the snapshot (SPEC §3).
const unboundMachine = "(unbound)"

// bodyHeight is the space under the top bar. The tab bar (or mode bar) is the
// only permanent chrome, so the body gets every remaining row; the toast is an
// overlay and reserves nothing.
func (m *model) bodyHeight() int {
	height := m.rows - 1
	if height < 1 {
		height = 1
	}
	return height
}

func (m *model) sidebarVisible() bool { return !m.sidebarCollapsed }

func (m *model) paneAreaWidth() int {
	width := m.cols
	if m.sidebarVisible() {
		width -= sidebarWidth
	}
	if width < 1 {
		width = 1
	}
	return width
}

// paneRects solves the current tab's geometry with widgets.SplitLayout.
func (m *model) paneRects(t *tab) ([]widgets.Rect, []widgets.Rect) {
	if t == nil || len(t.Panes) == 0 {
		return nil, nil
	}
	axis := t.Axis
	weights := t.Weights
	if len(weights) != len(t.Panes) {
		weights = equalWeights(len(t.Panes))
	}
	return widgets.SplitLayout{Orient: axis, Weights: weights, Gap: 1}.Rects(m.paneAreaWidth(), m.bodyHeight())
}

// sidebarSpacesHeight sizes the Spaces panel to its content (one row per
// workspace) and sidebarAgentsHeight gives the Agents panel the remaining
// rows.
func (m *model) sidebarSpacesHeight() int {
	height := len(m.workspaces) + 2 // header + rows
	if max := m.bodyHeight() / 2; height > max {
		height = max
	}
	if height < 3 {
		height = 3
	}
	if height > m.bodyHeight() {
		height = m.bodyHeight()
	}
	return height
}

func (m *model) sidebarAgentsHeight() int {
	height := m.bodyHeight() - m.sidebarSpacesHeight()
	if height < 3 {
		height = 3
	}
	return height
}

// agentRows flattens every pane of every workspace, in model order.
func (m *model) agentRows() []agentRow {
	var rows []agentRow
	for _, ws := range m.workspaces {
		for _, t := range ws.Tabs {
			for index, p := range t.Panes {
				rows = append(rows, agentRow{ws: ws, tab: t, pane: p, index: index})
			}
		}
	}
	return rows
}

// agentMachine is the Agents grouping key: the bound source's endpoint. Empty
// panes and gone sources have no machine and group under (unbound).
func (m *model) agentMachine(p *pane) string {
	src := m.source(p)
	if src == nil {
		return unboundMachine
	}
	endpoint, _ := sourceEndpointID(src)
	if endpoint == "" {
		return unboundMachine
	}
	return endpoint
}

// agentGroups buckets the flattened rows by machine in first-appearance
// order, with the (unbound) group last.
func (m *model) agentGroups() []agentGroup {
	var groups []agentGroup
	index := make(map[string]int)
	var unbound []agentRow
	for _, row := range m.agentRows() {
		machine := m.agentMachine(row.pane)
		if machine == unboundMachine {
			unbound = append(unbound, row)
			continue
		}
		i, ok := index[machine]
		if !ok {
			i = len(groups)
			index[machine] = i
			groups = append(groups, agentGroup{machine: machine})
		}
		groups[i].rows = append(groups[i].rows, row)
	}
	if len(unbound) > 0 {
		groups = append(groups, agentGroup{machine: unboundMachine, rows: unbound})
	}
	return groups
}

// agentListLen counts the rendered Agents rows (panes plus machine headers).
func (m *model) agentListLen() int {
	groups := m.agentGroups()
	count := 0
	for _, group := range groups {
		count += len(group.rows)
	}
	if len(groups) > 1 {
		count += len(groups)
	}
	return count
}

// clampOffset folds a scroll offset into [0, total-visible].
func clampOffset(offset, visible, total int) int {
	if visible <= 0 || total <= visible {
		return 0
	}
	if offset < 0 {
		return 0
	}
	if max := total - visible; offset > max {
		return max
	}
	return offset
}

// followOffset scrolls the minimum amount that keeps selected inside the
// window; selected < 0 only clamps the offset.
func followOffset(offset, selected, visible, total int) int {
	if selected >= 0 && visible > 0 {
		if selected < offset {
			offset = selected
		}
		if selected >= offset+visible {
			offset = selected - visible + 1
		}
	}
	return clampOffset(offset, visible, total)
}

// --- view ---

// View builds the whole tree: top bar (tab strip or mode bar) · body
// (sidebar + pane area), plus an optional transient toast and overlays. There
// is no permanent status or footer row (SPEC §3).
func (m *model) View() *pb.Box {
	m.dirty = false

	body := sdk.Row().ID(bodyID).Height(m.bodyHeight())
	if m.sidebarVisible() {
		body.Child(sdk.Raw(m.sidebarBox()))
	}
	_, t, _ := m.focusedPane()
	body.Child(m.panesArea(t))

	column := sdk.Col(m.topBar(), body)
	if m.toast != "" {
		column.Child(m.toastBox())
	}
	if overlay := m.overlay(); overlay != nil {
		return sdk.Stack(column, overlay).Build()
	}
	return column.Build()
}

func (m *model) topBar() *sdk.Builder {
	if m.mode != modeTerminal {
		return m.modeBar()
	}
	return m.tabBar()
}

// tabBar is the permanent top row: tabs on the left, a compact connection
// indicator on the right. Mode bars replace it while a mode is active. The
// indicator is reserved first: trailing tabs are dropped when the strip does
// not fit, so the connection state stays visible.
func (m *model) tabBar() *sdk.Builder {
	ws := m.currentWS()
	left := " herdr "
	if ws != nil {
		left = " " + widgetSafe(ws.Name) + " "
	}
	type tabLabel struct {
		text  string
		id    string
		style string
	}
	var tabs []tabLabel
	if ws != nil {
		for i, t := range ws.Tabs {
			title := fmt.Sprintf("%d:%s", i+1, widgetSafe(t.Name))
			label := tabLabel{text: " " + title + " ", id: tabPrefix + tabRef(ws, t), style: widgets.StyleTabInactive}
			if t.ID == ws.ActiveTab {
				label.text = "[" + title + "]"
				label.style = widgets.StyleTabActive
			}
			tabs = append(tabs, label)
		}
	}
	connection := m.connectionSegments()
	connWidth := 0
	for _, segment := range connection {
		connWidth += sdk.DisplayWidth(segment.Text)
	}
	used := sdk.DisplayWidth(left) + 3 // workspace label + the "+" tab
	if m.zoom {
		used += 6
	}
	for _, tab := range tabs {
		used += sdk.DisplayWidth(tab.text)
	}
	for len(tabs) > 1 && used+connWidth > m.cols {
		used -= sdk.DisplayWidth(tabs[len(tabs)-1].text)
		tabs = tabs[:len(tabs)-1]
	}

	row := sdk.Row().ID(tabbarID).Height(1)
	row.Child(sdk.Text(left).Style(widgets.StyleChrome))
	for _, tab := range tabs {
		row.Child(sdk.Text(tab.text).ID(tab.id).Style(tab.style).Input("mouse"))
	}
	row.Child(sdk.Text(" + ").ID(newTabID).Style(widgets.StyleChrome).Input("mouse"))
	if m.zoom {
		row.Child(sdk.Text(" ZOOM ").Style(widgets.StyleWarning))
	}
	if pad := m.cols - used - connWidth; pad > 0 {
		row.Child(sdk.Text(strings.Repeat(" ", pad)))
	}
	for _, segment := range connection {
		row.Child(sdk.Text(segment.Text).Style(segment.Style))
	}
	return row
}

// connectionSegments is the compact connection indicator: ● endpoint (or
// ● connected when no bound pane carries one), or ○ waiting before HELLO.
func (m *model) connectionSegments() []widgets.Segment {
	if m.hello == nil {
		return []widgets.Segment{{Text: " ○ waiting ", Style: widgets.StyleDanger}}
	}
	label := "connected"
	if _, _, p := m.focusedPane(); p != nil {
		if src := m.source(p); src != nil {
			if endpoint, _ := sourceEndpointID(src); endpoint != "" {
				label = endpoint
			}
		}
	}
	return []widgets.Segment{
		{Text: " ● ", Style: widgets.StyleSuccess},
		{Text: label + " ", Style: widgets.StyleMuted},
	}
}

func (m *model) modeBar() *sdk.Builder {
	hint := widgets.KeyHint{
		ID:        modebarID,
		Mode:      strings.ToUpper(m.mode),
		Keys:      modeKeys[m.mode],
		ModeStyle: widgets.StyleDanger,
		KeyStyle:  widgets.StyleMuted,
	}
	return sdk.Row(hint.Build()).Height(1)
}

// panesArea renders the visible panes of the current tab, separated by
// draggable divider boxes (implicit capture, PROTOCOL §6.7).
func (m *model) panesArea(t *tab) *sdk.Builder {
	width, height := m.paneAreaWidth(), m.bodyHeight()
	if t == nil || len(t.Panes) == 0 {
		return sdk.Box().ID("herdr.panes.empty").Width(width).Height(height)
	}
	visible := t.Panes
	axis := t.Axis
	weights := t.Weights
	if m.zoom && len(t.Panes) > 1 {
		if i := focusIndex(t); i >= 0 {
			visible = []*pane{t.Panes[i]}
		}
		axis, weights = "row", []int{100}
	}
	if len(weights) != len(visible) {
		weights = equalWeights(len(visible))
	}
	rects, dividers := widgets.SplitLayout{Orient: axis, Weights: weights, Gap: 1}.Rects(width, height)

	var container *sdk.Builder
	if axis == "col" {
		container = sdk.Col()
	} else {
		container = sdk.Row()
	}
	container.Width(width).Height(height)

	for i, p := range visible {
		index := paneIndexOf(t, p)
		rect := widgets.Rect{W: width, H: height}
		if i < len(rects) {
			rect = rects[i]
		}
		container.Child(m.paneBox(t, index, p, rect))
		if i < len(visible)-1 && i < len(dividers) {
			container.Child(m.dividerBox(t, i, dividers[i], axis))
		}
	}
	return container
}

func (m *model) dividerBox(t *tab, index int, rect widgets.Rect, axis string) *sdk.Builder {
	length := rect.W
	vertical := false
	if axis == "col" {
		length = rect.W
	} else {
		length = rect.H
		vertical = true
	}
	return widgets.Divider{
		ID:       fmt.Sprintf("%s%s:%d", dividerPrefix, tabRef(m.currentWS(), t), index),
		Vertical: vertical,
		Length:   length,
		Style:    widgets.StyleBorder,
		Input:    []string{"mouse"},
	}.Build()
}

// paneBox renders one pane: a live/exited terminal (with a program-drawn
// border when the tab is split) or the empty/gone/offline state.
func (m *model) paneBox(t *tab, index int, p *pane, rect widgets.Rect) *sdk.Builder {
	ref := m.refOf(m.currentWS(), t, p)
	state := m.paneState(p)

	if state == stateRunning || state == stateExited {
		chrome := len(t.Panes) > 1 && rect.H >= 4 && rect.W >= 6
		focused := m.terminalFocused(t, p)
		if !chrome {
			return sdk.Terminal(p.Source).ID(termPrefix+ref).
				Width(rect.W).Height(rect.H).
				Focused(focused).Input("key", "paste", "mouse").
				Props(map[string]string{"chrome.inset": "0"})
		}
		// The program draws the split chrome: a titled top rule and a bottom
		// rule; the focused pane uses StyleBorderFocus (SPEC §3). The terminal
		// stays borderless (chrome.inset=0) so the PTY owns the full content
		// rect.
		title := fmt.Sprintf("pane %d · %s", index+1, m.paneTitle(index, p))
		borderStyle := widgets.StyleBorder
		if focused {
			borderStyle = widgets.StyleBorderFocus
		}
		return sdk.Col(
			sdk.Text(paneTopLine(title, rect.W)).Style(borderStyle).Height(1).Width(rect.W),
			sdk.Terminal(p.Source).ID(termPrefix+ref).
				Width(rect.W).Height(rect.H-2).
				Focused(focused).Input("key", "paste", "mouse").
				Props(map[string]string{"chrome.inset": "0"}),
			sdk.Text(paneBottomLine(rect.W)).Style(borderStyle).Height(1).Width(rect.W),
		).Width(rect.W).Height(rect.H)
	}

	lines := []string{"empty pane", "enter creates a terminal"}
	title := fmt.Sprintf("pane %d", index+1)
	switch state {
	case stateGone:
		lines = []string{"source gone", p.Source, "enter creates a replacement"}
	case stateOffline:
		health := m.source(p).GetHealth()
		lines = []string{"endpoint offline", "health: " + health, "not interactive"}
	}
	return widgets.Card{
		ID:     paneBoxPrefix + ref,
		Title:  title,
		Lines:  lines,
		Width:  rect.W,
		Height: rect.H,
		Center: true,
	}.Build().Input("mouse")
}

// terminalFocused is true only for the focused, live pane in a mode where
// herdr claims just the escape chords. Program modes (and everything else)
// must clear `focused` (PROTOCOL §6.5).
func (m *model) terminalFocused(t *tab, p *pane) bool {
	if m.programMode() {
		return false
	}
	_, ct, cp := m.focusedPane()
	return ct == t && cp == p && m.paneState(p) == stateRunning
}

func paneIndexOf(t *tab, p *pane) int {
	for i, candidate := range t.Panes {
		if candidate == p {
			return i
		}
	}
	return 0
}

// paneTopLine draws the split top rule with the pane title embedded:
// "┌─ pane 1 · tui2-a ─────┐" (exactly width cells).
func paneTopLine(title string, width int) string {
	if width < 2 {
		return sdk.Truncate(title, maxInt(0, width))
	}
	inner := width - 2
	label := " " + title + " "
	if sdk.DisplayWidth(label) > inner-1 {
		label = sdk.Truncate(label, maxInt(0, inner-1))
	}
	fill := inner - 1 - sdk.DisplayWidth(label)
	if fill < 0 {
		fill = 0
	}
	return "┌─" + label + strings.Repeat("─", fill) + "┐"
}

func paneBottomLine(width int) string {
	inner := width - 2
	if inner < 0 {
		inner = 0
	}
	return "└" + strings.Repeat("─", inner) + "┘"
}

// --- sidebar ---

// sidebarKey captures every input the sidebar rows and chrome render.
func (m *model) sidebarKey() string {
	var b strings.Builder
	ws := m.currentWS()
	wsID, tabID, paneID := "", "", ""
	if ws != nil {
		wsID, tabID = ws.ID, ws.ActiveTab
	}
	if _, t, p := m.focusedPane(); t != nil && p != nil {
		paneID = p.ID
	}
	fmt.Fprintf(&b, "%d/%d/%d/%d/%d/%d/%d/%s/%s/%s/%s/%s/%t:",
		m.cols, m.rows, m.currentWSIndex(), m.navWS, m.spacesOffset, m.agentsOffset,
		m.bodyHeight(), m.viewID, m.mode, wsID, tabID, paneID, m.sidebarCollapsed)
	for _, w := range m.workspaces {
		fmt.Fprintf(&b, "w:%s:%s:%s:%s\x1e", w.ID, w.Name, w.ActiveTab, m.rollup(w))
		for _, t := range w.Tabs {
			fmt.Fprintf(&b, "t:%s:%s:%s:%s\x1e", t.ID, t.Name, t.Axis, t.Focus)
			for _, p := range t.Panes {
				fmt.Fprintf(&b, "p:%s:%s:%s:%s:%s:%s\x1e", p.ID, p.Name, p.Source, m.paneState(p), strings.Join(m.badges(p), ","), m.agentMachine(p))
			}
		}
	}
	return b.String()
}

func (m *model) sidebarBox() *pb.Box {
	return m.memo.Box(m.sidebarKey(), m.buildSidebar)
}

// buildSidebar stacks the two panels: Agents (all workspaces' panes grouped by
// machine) over Spaces (one row per workspace).
func (m *model) buildSidebar() *pb.Box {
	col := sdk.Col(sdk.Raw(m.agentsPanel()), sdk.Raw(m.spacesPanel())).
		Width(sidebarWidth).Height(m.bodyHeight())
	return col.Build()
}

// agentsPanel is the upstream Agent panel: every workspace's panes grouped by
// machine, two lines per pane, machine headers only when more than one
// machine is present.
func (m *model) agentsPanel() *pb.Box {
	groups := m.agentGroups()
	multi := len(groups) > 1
	height := m.sidebarAgentsHeight()
	visible := height - 1
	if visible < 0 {
		visible = 0
	}

	count := 0
	for _, group := range groups {
		count += len(group.rows)
	}
	if multi {
		count += len(groups)
	}
	rows := make([]*sdk.Builder, 0, count)
	selected := -1
	_, currentTab, currentPane := m.focusedPane()
	for _, group := range groups {
		if multi {
			rows = append(rows, machineHeaderRow(group.machine))
		}
		for _, row := range group.rows {
			focused := row.tab == currentTab && row.pane == currentPane
			if focused {
				selected = len(rows)
			}
			rows = append(rows, m.agentRowBox(row, focused))
		}
	}

	offset := followOffset(m.agentsOffset, selected, visible, len(rows))
	col := sdk.Col().ID("herdr.sidebar.agents").Width(sidebarWidth).Height(height)
	col.Child(panelHeader(" Agents"))
	end := minInt(offset+visible, len(rows))
	for _, row := range rows[offset:end] {
		col.Child(row)
	}
	if len(rows) == 0 {
		col.Child(panelEmpty(" no agents"))
	}
	return col.Build()
}

// spacesPanel is one row per workspace: rollup icon + name, tab names muted.
func (m *model) spacesPanel() *pb.Box {
	height := m.sidebarSpacesHeight()
	visible := height - 1
	if visible < 0 {
		visible = 0
	}
	selected := -1
	if m.mode == modeNavigate {
		selected = clampIndex(m.navWS, len(m.workspaces))
	} else if len(m.workspaces) > 0 {
		selected = m.currentWSIndex()
	}
	offset := followOffset(m.spacesOffset, selected, visible, len(m.workspaces))
	col := sdk.Col().ID("herdr.sidebar.spaces").Width(sidebarWidth).Height(height)
	col.Child(panelHeader(" Spaces"))
	end := minInt(offset+visible, len(m.workspaces))
	for i := offset; i < end; i++ {
		col.Child(m.spaceRowBox(m.workspaces[i], i))
	}
	if len(m.workspaces) == 0 {
		col.Child(panelEmpty(" none"))
	}
	return col.Build()
}

// agentRowBox renders one pane as the upstream two-line agent row: the state
// icon, workspace and tab on line 1, the pane name (plus attachment badges)
// on line 2. Active-workspace rows use the selection style; the focused pane
// carries the marker.
func (m *model) agentRowBox(row agentRow, focused bool) *sdk.Builder {
	active := row.ws == m.currentWS()
	state := m.paneState(row.pane)
	style := severityStyle(state)
	if active {
		style = widgets.StyleSelection
	}
	marker := "  "
	if focused {
		marker = "▸ "
	}
	line1 := marker + rollupDot(state) + " " + widgetSafe(row.ws.Name) + " · " + widgetSafe(row.tab.Name)
	line2 := "    " + m.paneTitle(row.index, row.pane)
	if badges := m.badges(row.pane); len(badges) > 0 {
		line2 += " " + strings.Join(badges, " ")
	}
	line2Style := widgets.StyleMuted
	if active {
		line2Style = widgets.StyleSelection
	}
	id := agentRowPrefix + paneRef(row.ws.ID, row.tab.ID, row.pane.ID)
	return sdk.Col(
		sdk.Text(padRow(line1, sidebarWidth)).ID(id).Style(style).Width(sidebarWidth).Height(1).Input("mouse"),
		sdk.Text(padRow(line2, sidebarWidth)).ID(id).Style(line2Style).Width(sidebarWidth).Height(1).Input("mouse"),
	).Width(sidebarWidth).Height(2)
}

// spaceRowBox renders one workspace: rollup icon + name, tab names appended
// muted. The active (or navigate-selected) workspace is highlighted.
func (m *model) spaceRowBox(ws *workspace, index int) *sdk.Builder {
	state := m.rollup(ws)
	selected := m.mode == modeNavigate && index == m.navWS
	active := ws == m.currentWS()
	marker := "  "
	if selected {
		marker = "▸ "
	}
	nameStyle := widgets.StyleDefault
	if selected || active {
		nameStyle = widgets.StyleSelection
	}
	spans := []widgets.Span{
		widgets.Styled(marker+rollupDot(state)+" ", severityStyle(state)),
		widgets.Styled(widgetSafe(ws.Name), nameStyle),
	}
	if names := workspaceTabNames(ws); names != "" {
		spans = append(spans, widgets.Styled("  "+names, widgets.StyleMuted))
	}
	row := widgets.RichText{Spans: spans, Width: sidebarWidth}.Build().Height(1)
	row.ID(spaceRowPrefix + ws.ID).Input("mouse")
	if selected || active {
		row.Style(widgets.StyleSelection)
	}
	return row
}

func workspaceTabNames(ws *workspace) string {
	names := make([]string, 0, len(ws.Tabs))
	for _, t := range ws.Tabs {
		names = append(names, widgetSafe(t.Name))
	}
	return strings.Join(names, " ")
}

func panelHeader(text string) *sdk.Builder {
	return sdk.Text(padRow(text, sidebarWidth)).Style(widgets.StyleStrongForeground).Width(sidebarWidth).Height(1)
}

func panelEmpty(text string) *sdk.Builder {
	return sdk.Text(padRow(text, sidebarWidth)).Style(widgets.StyleMuted).Width(sidebarWidth).Height(1)
}

func machineHeaderRow(machine string) *sdk.Builder {
	return sdk.Text(padRow(" "+machine, sidebarWidth)).Style(widgets.StyleStrongForeground).Width(sidebarWidth).Height(1)
}

// padRow truncates a row to width display cells and pads it with spaces so a
// style fills the whole sidebar row.
func padRow(text string, width int) string {
	text = sdk.Truncate(widgetSafe(text), width)
	if pad := width - sdk.DisplayWidth(text); pad > 0 {
		text += strings.Repeat(" ", pad)
	}
	return text
}

// rollupDot is the sidebar status dot for a workspace rollup (SPEC §2/§3).
func rollupDot(state paneState) string {
	switch state {
	case stateRunning:
		return "●"
	case stateExited:
		return "◐"
	case stateGone, stateOffline:
		return "○"
	default:
		return "·"
	}
}

func severityStyle(state paneState) string {
	switch state {
	case stateOffline, stateGone:
		return widgets.StyleDanger
	case stateExited:
		return widgets.StyleWarning
	case stateRunning:
		return widgets.StyleSuccess
	default:
		return widgets.StyleMuted
	}
}

// --- toast ---

// toastBox is a transient overlay pinned to the bottom-right corner: it exists
// only while a message does (no reserved row), and its width clips long host
// notices to the viewport.
func (m *model) toastBox() *sdk.Builder {
	width := sdk.DisplayWidth(widgetSafe(m.toast)) + 2
	if width > m.cols {
		width = m.cols
	}
	if width < 1 {
		width = 1
	}
	toast := widgets.Toast{ID: toastID, Text: m.toastText(width), Style: m.toastStyle()}
	return toast.Build().Width(width).Height(1).Pos(m.cols-width, m.rows-1)
}

// toastText pads the message with one space each side, clipping it to the
// overlay width so it cannot spill past the screen edge.
func (m *model) toastText(width int) string {
	inner := width - 2
	if inner < 1 {
		return sdk.Truncate(widgetSafe(m.toast), width)
	}
	return " " + sdk.Truncate(widgetSafe(m.toast), inner) + " "
}

func (m *model) toastStyle() string {
	if m.toastErr {
		return widgets.StyleDanger
	}
	return widgets.StyleOverlay
}

// --- overlays ---

func (m *model) overlay() *sdk.Builder {
	switch {
	case m.rename != nil:
		return m.renameModal()
	case m.help:
		return m.helpModal()
	case m.menu.Visible:
		return m.menu.Build()
	}
	return nil
}

func (m *model) renameModal() *sdk.Builder {
	label := string(m.rename.kind)
	value := m.rename.input.DisplayText()
	if value == "" {
		value = m.rename.input.Placeholder
	}
	rows := []widgets.FrameRow{
		{Text: widgetSafe(" " + label + ":"), Style: widgets.StyleMuted},
		{Text: widgetSafe(" ▏" + value), Style: widgets.StyleDefault},
	}
	width := m.renameInputWidth() + 4
	if width > m.cols-2 {
		width = m.cols - 2
	}
	if width < 12 {
		width = 12
	}
	return widgets.Modal{
		ID:           "herdr.rename",
		Title:        "herdr · rename " + label,
		Width:        width,
		Height:       4,
		Rows:         rows,
		Style:        widgets.StyleBorderFocus,
		Backdrop:     true,
		Center:       true,
		ParentWidth:  m.cols,
		ParentHeight: m.rows,
	}.Build()
}

func (m *model) helpModal() *sdk.Builder {
	inner := m.helpModalInnerWidth()
	lines := m.helpWrappedLines(inner)
	if maxLines := m.rows - 4; maxLines > 0 && len(lines) > maxLines {
		// Never clip silently: keep the first lines and mark the cut.
		lines = lines[:maxLines]
		lines[maxLines-1] = "… resize taller for the rest"
	}
	rows := make([]widgets.FrameRow, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, widgets.FrameRow{Text: line})
	}
	width := inner + 2
	height := len(rows) + 2
	if height > m.rows-2 {
		height = m.rows - 2
	}
	if height < 3 {
		height = 3
	}
	return widgets.Modal{
		ID:           helpID,
		Title:        "herdr · keys",
		Width:        width,
		Height:       height,
		Rows:         rows,
		Style:        widgets.StyleBorderFocus,
		Backdrop:     true,
		Center:       true,
		ParentWidth:  m.cols,
		ParentHeight: m.rows,
	}.Build()
}

// helpModalInnerWidth sizes the help modal to its longest row, clamped to the
// viewport (and a 100-cell readability cap); long rows wrap, they never clip.
func (m *model) helpModalInnerWidth() int {
	width := 24
	for _, row := range helpRows {
		if need := sdk.DisplayWidth(widgetSafe(row)) + 2; need > width {
			width = need
		}
	}
	if max := m.cols - 4; width > max {
		width = max
	}
	if width > 100 {
		width = 100
	}
	if width < 24 {
		width = 24
	}
	inner := width - 2
	if inner < 1 {
		inner = 1
	}
	return inner
}

var helpRows = []string{
	"ctrl+b then:",
	"  v/- split · h/j/k/l focus · H/J/K/L swap · tab cycle",
	"  x close pane · z zoom · r resize",
	"  c new tab · X close tab · p/n prev/next · 1..9 switch",
	"  N new workspace · D close · W/T/P rename",
	"  w navigate · b sidebar · [ scroll · ? help · q detach",
	"  q asks the host to quit; pool terminals keep running",
	"terminal: ctrl+alt+h/j/k/l focus · +c tab · +d split · +z zoom",
	"mouse: click sidebar/tab/pane · right-click menu · drag divider",
	"navigate: up/down workspace · h/j/k/l pane · enter · esc",
	"resize: h/j/k/l 5% steps · enter/esc done",
	"scroll: j/k · page-up/down · y copy · q/esc live",
	"empty pane: enter creates a local terminal",
}

// widgetSafe strips control runes so a user-provided name cannot break the
// pre-split row layout.
// helpWrappedLines wraps every help row to inner cells (CJK-safe), keeping the
// source order. Concatenating a row's wrapped lines reproduces the source text.
func (m *model) helpWrappedLines(inner int) []string {
	if inner < 1 {
		inner = 1
	}
	var out []string
	for _, row := range helpRows {
		out = append(out, widgets.WrapText(widgetSafe(row), inner)...)
	}
	return out
}

func widgetSafe(text string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, text)
}

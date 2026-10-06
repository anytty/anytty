package main

import (
	"strconv"
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk/app"
	"github.com/anytty/anytty/clients/tui/sdk/widgets"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// Node id namespaces. Every clickable box carries one of these ids; the host
// hit-tests the view and echoes the topmost id back in mouse events.
const (
	spaceRowPrefix = "herdr.space:"   // herdr.space:w1 (Spaces sidebar row)
	agentRowPrefix = "herdr.agent:"   // herdr.agent:w1:t1:p1 (Agents sidebar row)
	tabPrefix      = "herdr.tab:"     // herdr.tab:w1:t1
	newTabID       = "herdr.tab.new"  // the "+" tab
	paneBoxPrefix  = "herdr.panebox:" // herdr.panebox:w1:t1:p1 (empty/gone/offline)
	termPrefix     = "herdr.term:"    // herdr.term:w1:t1:p1 (live terminal)
	dividerPrefix  = "herdr.divider:" // herdr.divider:w1:t1:0
	menuPrefix     = "herdr.menu:"    // herdr.menu:split-right
	renameInputID  = "herdr.rename.input"
	helpID         = "herdr.help"
	bodyID         = "herdr.body"
	tabbarID       = "herdr.tabbar"
	modebarID      = "herdr.modebar"
	toastID        = "herdr.toast"
)

// menu kinds and actions.
const (
	menuPane      = "pane"
	menuTab       = "tab"
	menuWorkspace = "workspace"

	actSplitRight   = "split-right"
	actSplitDown    = "split-down"
	actNewTab       = "new-tab"
	actNewWorkspace = "new-workspace"
	actRename       = "rename"
	actClosePane    = "close-pane"
	actCloseTab     = "close-tab"
	actCloseWS      = "close-workspace"
)

// --- key routing ---

func (m *model) onKey(key string) app.Cmd {
	if key == "" {
		return app.None
	}
	if m.rename != nil {
		return m.onRenameKey(key)
	}
	if m.help {
		switch key {
		case "esc", "?", "q":
			m.help = false
			m.changed()
		}
		return app.None
	}
	if m.menu.Visible {
		return m.onMenuKey(key)
	}
	switch m.mode {
	case modePrefix:
		return m.onPrefixKey(key)
	case modeNavigate:
		return m.onNavigateKey(key)
	case modeResize:
		return m.onResizeKey(key)
	case modeScroll:
		return m.onScrollKey(key)
	default:
		return m.onTerminalKey(key)
	}
}

// onTerminalKey handles the direct chords (SPEC §5) plus enter on an
// empty/gone pane. Everything else never reaches the model while a live
// terminal is focused (the claim sends it to the PTY).
func (m *model) onTerminalKey(key string) app.Cmd {
	switch key {
	case "ctrl-b":
		m.mode = modePrefix
		m.changed()
		return app.None
	case "ctrl-alt-h":
		m.focusDir("h")
	case "ctrl-alt-j":
		m.focusDir("j")
	case "ctrl-alt-k":
		m.focusDir("k")
	case "ctrl-alt-l":
		m.focusDir("l")
	case "ctrl-alt-c":
		return m.newTab()
	case "ctrl-alt-d":
		return m.split("row")
	case "ctrl-alt-z":
		return m.toggleZoom()
	case "enter":
		return m.enterFocusedPane()
	}
	return app.None
}

// enterFocusedPane creates a terminal for an empty or gone pane (SPEC §2/§7).
func (m *model) enterFocusedPane() app.Cmd {
	_, _, p := m.focusedPane()
	if p == nil {
		return app.None
	}
	if state := m.paneState(p); state != stateEmpty && state != stateGone {
		return app.None
	}
	ws, t, p := m.focusedPane()
	return m.createCmd(m.refOf(ws, t, p))
}

// onPrefixKey implements the SPEC §5 prefix table. Every action consumes the
// prefix; unknown keys are ignored and leave prefix mode.
func (m *model) onPrefixKey(key string) app.Cmd {
	if key == "esc" || key == "ctrl-b" {
		m.mode = modeTerminal
		m.changed()
		return app.None
	}
	// Shifted letters may arrive as "H" or as "shift-h".
	if base, ok := shiftedLetter(key); ok {
		var cmd app.Cmd
		switch base {
		case "H":
			m.swapPane("h")
		case "J":
			m.swapPane("j")
		case "K":
			m.swapPane("k")
		case "L":
			m.swapPane("l")
		case "N":
			cmd = m.newWorkspace()
		case "D":
			cmd = m.closeCurrentWorkspace()
		case "W":
			cmd = m.openRename(renameWorkspace, m.currentWSID())
		case "T":
			cmd = m.openRename(renameTab, m.currentTabRef())
		case "P":
			cmd = m.openRename(renamePane, m.currentPaneRef())
		case "X":
			m.closeTab(m.currentTabRef())
		}
		m.mode = modeTerminal
		m.changed()
		return cmd
	}
	var cmd app.Cmd
	switch key {
	case "v":
		cmd = m.split("row")
	case "-":
		cmd = m.split("col")
	case "h", "j", "k", "l":
		m.focusDir(key)
	case "tab":
		m.cyclePane(1)
	case "shift-tab":
		m.cyclePane(-1)
	case "x":
		m.closeFocusedPane()
	case "z":
		cmd = m.toggleZoom()
	case "r":
		m.mode = modeResize
		m.changed()
		return app.None
	case "c":
		cmd = m.newTab()
	case "p":
		m.switchTab(-1)
	case "n":
		m.switchTab(1)
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		m.switchTabIndex(int(key[0] - '1'))
	case "w":
		m.mode = modeNavigate
		m.navWS = m.currentWSIndex()
		m.changed()
		return app.None
	case "b":
		m.sidebarCollapsed = !m.sidebarCollapsed
		m.structural()
	case "[":
		cmd = m.enterScroll()
	case "?":
		m.help = true
		m.mode = modeTerminal
		m.changed()
		return app.None
	case "q":
		cmd = m.quitCmd()
	default:
		// Unknown prefix key: ignore and leave prefix mode.
	}
	if m.mode == modePrefix {
		m.mode = modeTerminal
		m.changed()
	}
	return cmd
}

// onNavigateKey: up/down move the workspace selection, h/j/k/l move pane
// focus, enter activates the selected workspace (SPEC §4).
func (m *model) onNavigateKey(key string) app.Cmd {
	ws := m.currentWS()
	switch key {
	case "up":
		m.navWS = clampIndex(m.navWS-1, len(m.workspaces))
		m.touch()
	case "down":
		m.navWS = clampIndex(m.navWS+1, len(m.workspaces))
		m.touch()
	case "h", "j", "k", "l":
		if m.navWS != m.currentWSIndex() && ws != nil {
			m.navWS = m.currentWSIndex()
		}
		m.focusDir(key)
	case "enter":
		if m.navWS >= 0 && m.navWS < len(m.workspaces) {
			m.activeWorkspace = m.workspaces[m.navWS].ID
			m.zoom = false
			m.structural()
		}
		m.mode = modeTerminal
		m.changed()
	case "esc":
		m.mode = modeTerminal
		m.changed()
	case "ctrl-b":
		m.mode = modePrefix
		m.changed()
	}
	return app.None
}

// onResizeKey: h/j/k/l move the divider adjacent to the focused pane in 5%
// steps; enter/esc leave (SPEC §4).
func (m *model) onResizeKey(key string) app.Cmd {
	switch key {
	case "h", "j", "k", "l":
		m.resizeMove(key)
	case "enter", "esc":
		m.mode = modeTerminal
		m.changed()
	case "ctrl-b":
		m.mode = modePrefix
		m.changed()
	}
	return app.None
}

// onScrollKey: j/k and page keys drive terminal.scroll, y copies the visible
// area, q/esc return to live (scrollEnd) and leave the mode (SPEC §4).
func (m *model) onScrollKey(key string) app.Cmd {
	switch key {
	case "k":
		return m.scrollBy(1)
	case "j":
		return m.scrollBy(-1)
	case "page-up":
		return m.scrollBy(m.scrollPage())
	case "page-down":
		return m.scrollBy(-m.scrollPage())
	case "y":
		return m.copyCmd()
	case "q", "esc":
		m.mode = modeTerminal
		m.changed()
		return m.scrollEndCmd()
	case "ctrl-b":
		m.mode = modePrefix
		m.changed()
		return m.scrollEndCmd()
	}
	return app.None
}

func (m *model) onMenuKey(key string) app.Cmd {
	switch key {
	case "esc":
		m.menu.Close()
		m.changed()
	case "up", "k":
		m.menu.Move(-1)
		m.touch()
	case "down", "j":
		m.menu.Move(1)
		m.touch()
	case "enter":
		value := m.menu.Value()
		m.menu.Close()
		m.changed()
		return m.applyMenuAction(value)
	}
	return app.None
}

func (m *model) onRenameKey(key string) app.Cmd {
	switch key {
	case "esc":
		m.rename = nil
		m.changed()
		return app.None
	case "enter":
		m.commitRename()
		return app.None
	}
	if m.rename.input.HandleKey(&pb.KeyEvent{Key: key}) {
		m.touch()
	}
	return app.None
}

func (m *model) onPaste(p *pb.PasteEvent) app.Cmd {
	if m.rename == nil || p == nil {
		return app.None
	}
	if m.rename.input.InsertString(p.GetText()) {
		m.touch()
	}
	return app.None
}

// --- mode entry/exit helpers ---

func (m *model) enterScroll() app.Cmd {
	_, _, p := m.focusedPane()
	if p == nil || m.source(p) == nil {
		m.setToast("no terminal to scroll", true)
		m.mode = modeTerminal
		m.changed()
		return app.None
	}
	if m.paneState(p) == stateOffline {
		m.setToast("endpoint offline; scroll unavailable", true)
		m.mode = modeTerminal
		m.changed()
		return app.None
	}
	m.mode = modeScroll
	m.scrollLines = 0
	m.changed()
	return app.None
}

func (m *model) scrollBy(delta int) app.Cmd {
	if delta == 0 {
		return app.None
	}
	_, _, p := m.focusedPane()
	if p == nil || m.source(p) == nil {
		return app.None
	}
	next := m.scrollLines + delta
	if next < 0 {
		next = 0
	}
	if delta < 0 && m.scrollLines == 0 {
		return app.None // already live
	}
	m.scrollLines = next
	m.touch()
	if next == 0 {
		m.mode = modeTerminal
		m.changed()
		return chain(m.scrollCmd(delta), m.scrollEndCmd())
	}
	return m.scrollCmd(delta)
}

func (m *model) scrollPage() int {
	page := m.bodyHeight() - 2
	if page < 1 {
		page = 1
	}
	return page
}

func (m *model) toggleZoom() app.Cmd {
	_, t, p := m.focusedPane()
	if t == nil || p == nil {
		return app.None
	}
	if len(t.Panes) < 2 {
		m.setToast("nothing to zoom", false)
		return app.None
	}
	m.zoom = !m.zoom
	m.touch()
	return app.None
}

// --- focus / layout movement ---

func (m *model) focusDir(dir string) {
	_, t, _ := m.focusedPane()
	i := focusIndex(t)
	if t == nil || i < 0 {
		return
	}
	delta := axisDelta(t.Axis, dir)
	j := i + delta
	if delta == 0 || j < 0 || j >= len(t.Panes) {
		return
	}
	t.Focus = t.Panes[j].ID
	m.zoom = false
	m.structural()
}

func (m *model) swapPane(dir string) {
	_, t, _ := m.focusedPane()
	i := focusIndex(t)
	if t == nil || i < 0 {
		return
	}
	j := i + axisDelta(t.Axis, dir)
	if j < 0 || j >= len(t.Panes) {
		return
	}
	t.Panes[i], t.Panes[j] = t.Panes[j], t.Panes[i]
	m.structural()
}

func (m *model) cyclePane(delta int) {
	_, t, _ := m.focusedPane()
	if t == nil || len(t.Panes) == 0 {
		return
	}
	i := focusIndex(t)
	j := (i + delta + len(t.Panes)) % len(t.Panes)
	t.Focus = t.Panes[j].ID
	m.zoom = false
	m.structural()
}

// axisDelta maps a vi-direction to a pane-list delta (row: h/l; col: k/j).
func axisDelta(axis, dir string) int {
	if axis == "col" {
		switch dir {
		case "k":
			return -1
		case "j":
			return 1
		}
		return 0
	}
	switch dir {
	case "h":
		return -1
	case "l":
		return 1
	}
	return 0
}

// resizeMove moves the split divider adjacent to the focused pane in the
// pressed direction, 5% per press (SPEC §4): h/k move a divider toward the
// start of the pane list, l/j toward the end. At the outer edge the nearest
// divider moves the opposite way so both directions always work.
func (m *model) resizeMove(dir string) {
	_, t, _ := m.focusedPane()
	i := focusIndex(t)
	if t == nil || i < 0 || len(t.Panes) < 2 {
		return
	}
	back := dir == "h" || dir == "k"
	var ok bool
	if back {
		if i > 0 {
			ok = adjustWeight(t.Weights, i, i-1, minPaneWeight)
		} else {
			ok = adjustWeight(t.Weights, i, i+1, -minPaneWeight)
		}
	} else {
		if i < len(t.Panes)-1 {
			ok = adjustWeight(t.Weights, i, i+1, minPaneWeight)
		} else {
			ok = adjustWeight(t.Weights, i, i-1, -minPaneWeight)
		}
	}
	if ok {
		m.structural()
	}
}

// --- tab / workspace switching ---

func (m *model) switchTab(delta int) {
	ws := m.currentWS()
	if ws == nil || len(ws.Tabs) == 0 {
		return
	}
	i := 0
	for index, t := range ws.Tabs {
		if t.ID == ws.ActiveTab {
			i = index
			break
		}
	}
	j := (i + delta + len(ws.Tabs)) % len(ws.Tabs)
	ws.ActiveTab = ws.Tabs[j].ID
	m.zoom = false
	m.structural()
}

func (m *model) switchTabIndex(index int) {
	ws := m.currentWS()
	if ws == nil || index < 0 || index >= len(ws.Tabs) {
		return
	}
	ws.ActiveTab = ws.Tabs[index].ID
	m.zoom = false
	m.structural()
}

func (m *model) activateWorkspace(id string) {
	if m.workspaceByID(id) == nil || m.activeWorkspace == id {
		if m.workspaceByID(id) != nil {
			m.mode = modeTerminal
			m.changed()
		}
		return
	}
	m.activeWorkspace = id
	m.navWS = m.currentWSIndex()
	m.zoom = false
	m.mode = modeTerminal
	m.structural()
	m.keysChanged = true
}

func (m *model) activateTabRef(ref string) {
	parts := strings.Split(ref, ":")
	if len(parts) != 2 {
		return
	}
	ws := m.workspaceByID(parts[0])
	if ws == nil || tabByID(ws, parts[1]) == nil {
		return
	}
	changed := m.activeWorkspace != ws.ID || ws.ActiveTab != parts[1] || m.mode != modeTerminal
	m.activeWorkspace = ws.ID
	ws.ActiveTab = parts[1]
	m.navWS = m.currentWSIndex()
	m.zoom = false
	m.mode = modeTerminal
	if changed {
		m.structural()
		m.keysChanged = true
	} else {
		m.touch()
	}
}

func (m *model) focusPaneRef(ref string) {
	ws, t, p := m.findPane(ref)
	if ws == nil || t == nil || p == nil {
		return
	}
	changed := m.activeWorkspace != ws.ID || ws.ActiveTab != t.ID || t.Focus != p.ID || m.mode != modeTerminal
	m.activeWorkspace = ws.ID
	ws.ActiveTab = t.ID
	t.Focus = p.ID
	m.navWS = m.currentWSIndex()
	m.zoom = false
	m.mode = modeTerminal
	if changed {
		m.structural()
		m.keysChanged = true
	} else {
		m.touch()
	}
}

func (m *model) currentWSIndex() int {
	for i, ws := range m.workspaces {
		if ws.ID == m.activeWorkspace {
			return i
		}
	}
	return 0
}

func (m *model) currentWSID() string {
	if ws := m.currentWS(); ws != nil {
		return ws.ID
	}
	return ""
}

func (m *model) currentTabRef() string {
	ws := m.currentWS()
	return tabRef(ws, m.currentTab())
}

func (m *model) currentPaneRef() string {
	ws, t, p := m.focusedPane()
	return m.refOf(ws, t, p)
}

// --- structural operations (SPEC §2) ---

// newWorkspace appends a workspace with one empty tab/pane and activates it.
func (m *model) newWorkspace() app.Cmd {
	ids := make([]string, 0, len(m.workspaces))
	for _, ws := range m.workspaces {
		ids = append(ids, ws.ID)
	}
	id := nextNumericID("w", ids)
	name := defaultWorkspaceName
	if len(m.workspaces) > 0 {
		name = "ws" + strings.TrimPrefix(id, "w")
	}
	ws := &workspace{ID: id, Name: name, ActiveTab: "t1"}
	ws.Tabs = []*tab{{
		ID:      "t1",
		Name:    defaultTabName,
		Axis:    "row",
		Weights: []int{100},
		Focus:   "p1",
		Panes:   []*pane{{ID: "p1"}},
	}}
	m.workspaces = append(m.workspaces, ws)
	m.activeWorkspace = ws.ID
	m.navWS = len(m.workspaces) - 1
	m.zoom = false
	m.structural()
	m.setToast("new workspace "+name, false)
	return app.None
}

// newTab appends an empty tab to the current workspace and activates it.
func (m *model) newTab() app.Cmd {
	ws := m.currentWS()
	if ws == nil {
		return app.None
	}
	ids := make([]string, 0, len(ws.Tabs))
	for _, t := range ws.Tabs {
		ids = append(ids, t.ID)
	}
	id := nextNumericID("t", ids)
	t := &tab{
		ID:      id,
		Name:    addedTabName,
		Axis:    "row",
		Weights: []int{100},
		Focus:   "p1",
		Panes:   []*pane{{ID: "p1"}},
	}
	ws.Tabs = append(ws.Tabs, t)
	ws.ActiveTab = id
	m.zoom = false
	m.structural()
	m.setToast("new tab "+t.Name, false)
	return app.None
}

// split appends an empty pane to the current tab along axis and focuses it.
// A tab carries a single axis; the axis is set when the tab has one pane and
// kept afterwards.
func (m *model) split(axis string) app.Cmd {
	_, t, _ := m.focusedPane()
	if t == nil {
		return app.None
	}
	if len(t.Panes) >= maxPanes {
		m.setToast("pane limit reached", true)
		return app.None
	}
	if len(t.Panes) == 1 || t.Axis == "" {
		t.Axis = axis
	}
	ids := make([]string, 0, len(t.Panes))
	for _, p := range t.Panes {
		ids = append(ids, p.ID)
	}
	p := &pane{ID: nextNumericID("p", ids)}
	t.Panes = append(t.Panes, p)
	t.Weights = equalWeights(len(t.Panes))
	t.Focus = p.ID
	m.zoom = false
	m.structural()
	return app.None
}

func (m *model) closeFocusedPane() {
	if ref := m.currentPaneRef(); ref != "" {
		m.closePane(ref)
	}
}

func (m *model) closePane(ref string) {
	ws, t, p := m.findPane(ref)
	if ws == nil || t == nil || p == nil {
		return
	}
	index := -1
	for i, candidate := range t.Panes {
		if candidate == p {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	t.Panes = append(t.Panes[:index], t.Panes[index+1:]...)
	if index < len(t.Weights) {
		t.Weights = append(t.Weights[:index], t.Weights[index+1:]...)
	}
	t.Weights = normalizeTo100(t.Weights)
	if t.Focus == p.ID {
		switch {
		case len(t.Panes) == 0:
			t.Focus = ""
		case index >= len(t.Panes):
			t.Focus = t.Panes[len(t.Panes)-1].ID
		default:
			t.Focus = t.Panes[index].ID
		}
	}
	if len(t.Panes) == 0 {
		m.closeTab(tabRef(ws, t))
		return
	}
	m.zoom = false
	m.structural()
	m.setToast("closed pane "+p.ID, false)
}

func (m *model) closeTab(ref string) {
	parts := strings.Split(ref, ":")
	if len(parts) != 2 {
		return
	}
	ws := m.workspaceByID(parts[0])
	if ws == nil {
		return
	}
	index := -1
	for i, t := range ws.Tabs {
		if t.ID == parts[1] {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	ws.Tabs = append(ws.Tabs[:index], ws.Tabs[index+1:]...)
	if len(ws.Tabs) == 0 {
		m.closeWorkspace(ws.ID)
		return
	}
	if ws.ActiveTab == parts[1] {
		if index >= len(ws.Tabs) {
			index = len(ws.Tabs) - 1
		}
		ws.ActiveTab = ws.Tabs[index].ID
	}
	m.zoom = false
	m.structural()
	m.setToast("closed tab", false)
}

func (m *model) closeCurrentWorkspace() app.Cmd {
	if ws := m.currentWS(); ws != nil {
		m.closeWorkspace(ws.ID)
	}
	return app.None
}

func (m *model) closeWorkspace(id string) {
	index := -1
	for i, ws := range m.workspaces {
		if ws.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	m.workspaces = append(m.workspaces[:index], m.workspaces[index+1:]...)
	if len(m.workspaces) == 0 {
		m.seedLayout()
		m.zoom = false
		m.structural()
		m.setToast("new workspace "+defaultWorkspaceName, false)
		return
	}
	if m.activeWorkspace == id {
		if index >= len(m.workspaces) {
			index = len(m.workspaces) - 1
		}
		m.activeWorkspace = m.workspaces[index].ID
	}
	m.navWS = m.currentWSIndex()
	m.zoom = false
	m.structural()
	m.setToast("closed workspace", false)
}

// --- menus / rename ---

func menuItems(kind string) []widgets.MenuItem {
	item := func(action, label, hotkey string) widgets.MenuItem {
		return widgets.MenuItem{ID: menuPrefix + action, Label: label, Hotkey: hotkey}
	}
	switch kind {
	case menuTab:
		return []widgets.MenuItem{
			item(actNewTab, "new tab", "c"),
			item(actRename, "rename tab", "r"),
			item(actCloseTab, "close tab", "x"),
			{Separator: true},
			item(actNewWorkspace, "new workspace", "n"),
		}
	case menuWorkspace:
		return []widgets.MenuItem{
			item(actNewWorkspace, "new workspace", "n"),
			item(actRename, "rename workspace", "r"),
			item(actCloseWS, "close workspace", "x"),
			{Separator: true},
			item(actNewTab, "new tab", "c"),
		}
	default:
		return []widgets.MenuItem{
			item(actSplitRight, "split right", "v"),
			item(actSplitDown, "split down", "-"),
			item(actRename, "rename pane", "r"),
			item(actClosePane, "close pane", "x"),
			{Separator: true},
			item(actNewTab, "new tab", "c"),
			item(actNewWorkspace, "new workspace", "n"),
		}
	}
}

func (m *model) openMenu(kind, ref string, ev *pb.MouseEvent) app.Cmd {
	m.menuKind = kind
	m.menuRef = ref
	m.menu.Items = menuItems(kind)
	m.menu.Width = 24
	m.menu.ParentWidth = m.cols
	m.menu.ParentHeight = m.rows
	m.menu.Margin = 1
	m.menu.Open(int(ev.GetX()), int(ev.GetY()))
	m.menu.Select(0)
	m.changed()
	return app.None
}

func (m *model) applyMenuAction(action string) app.Cmd {
	switch action {
	case actSplitRight:
		return m.split("row")
	case actSplitDown:
		return m.split("col")
	case actNewTab:
		return m.newTab()
	case actNewWorkspace:
		return m.newWorkspace()
	case actRename:
		switch m.menuKind {
		case menuWorkspace:
			return m.openRename(renameWorkspace, m.menuRef)
		case menuTab:
			return m.openRename(renameTab, m.menuRef)
		default:
			return m.openRename(renamePane, m.menuRef)
		}
	case actClosePane:
		m.closePane(m.menuRef)
	case actCloseTab:
		m.closeTab(m.menuRef)
	case actCloseWS:
		m.closeWorkspace(m.menuRef)
	}
	return app.None
}

func (m *model) openRename(kind renameKind, ref string) app.Cmd {
	if ref == "" {
		return app.None
	}
	input := widgets.TextInput{
		ID:          renameInputID,
		Width:       m.renameInputWidth(),
		Placeholder: "name",
		Style:       widgets.StyleDefault,
	}
	input.SetValue(m.nameOf(kind, ref))
	m.rename = &renameState{kind: kind, ref: ref, input: input}
	m.mode = modeTerminal
	m.changed()
	return app.None
}

func (m *model) renameInputWidth() int {
	width := m.cols - 10
	if width > 44 {
		width = 44
	}
	if width < 12 {
		width = 12
	}
	return width
}

func (m *model) nameOf(kind renameKind, ref string) string {
	switch kind {
	case renameWorkspace:
		if ws := m.workspaceByID(ref); ws != nil {
			return ws.Name
		}
	case renameTab:
		parts := strings.Split(ref, ":")
		if len(parts) == 2 {
			if ws := m.workspaceByID(parts[0]); ws != nil {
				if t := tabByID(ws, parts[1]); t != nil {
					return t.Name
				}
			}
		}
	case renamePane:
		if _, _, p := m.findPane(ref); p != nil {
			return p.Name
		}
	}
	return ""
}

func (m *model) commitRename() {
	if m.rename == nil {
		return
	}
	value := strings.TrimSpace(m.rename.input.Text())
	switch m.rename.kind {
	case renameWorkspace:
		if ws := m.workspaceByID(m.rename.ref); ws != nil {
			ws.Name = value
		}
	case renameTab:
		parts := strings.Split(m.rename.ref, ":")
		if len(parts) == 2 {
			if ws := m.workspaceByID(parts[0]); ws != nil {
				if t := tabByID(ws, parts[1]); t != nil {
					t.Name = value
				}
			}
		}
	case renamePane:
		if _, _, p := m.findPane(m.rename.ref); p != nil {
			p.Name = value
		}
	}
	m.rename = nil
	m.structural()
	m.changed()
}

// --- mouse / wheel ---

func (m *model) onMouse(ev *pb.MouseEvent) app.Cmd {
	if ev == nil {
		return app.None
	}
	switch ev.GetAction() {
	case "press":
		return m.onMousePress(ev)
	case "drag":
		return m.onMouseDrag(ev)
	case "release":
		return m.onMouseRelease(ev)
	}
	return app.None
}

func (m *model) onMousePress(ev *pb.MouseEvent) app.Cmd {
	node := ev.GetNode()
	if m.menu.Visible {
		if action, ok := strings.CutPrefix(node, menuPrefix); ok {
			m.menu.Close()
			m.changed()
			return m.applyMenuAction(action)
		}
		m.menu.Close()
		m.changed()
		return app.None
	}
	if m.rename != nil || m.help {
		return app.None
	}
	right := ev.GetButton() == "right"
	switch {
	case strings.HasPrefix(node, spaceRowPrefix):
		id := strings.TrimPrefix(node, spaceRowPrefix)
		if right {
			return m.openMenu(menuWorkspace, id, ev)
		}
		m.activateWorkspace(id)
	case node == newTabID:
		if !right {
			return m.newTab()
		}
	case strings.HasPrefix(node, tabPrefix):
		ref := strings.TrimPrefix(node, tabPrefix)
		if right {
			return m.openMenu(menuTab, ref, ev)
		}
		m.activateTabRef(ref)
	case strings.HasPrefix(node, agentRowPrefix):
		ref := strings.TrimPrefix(node, agentRowPrefix)
		if right {
			return m.openMenu(menuPane, ref, ev)
		}
		m.focusPaneRef(ref)
	case strings.HasPrefix(node, termPrefix):
		ref := strings.TrimPrefix(node, termPrefix)
		if right {
			return m.openMenu(menuPane, ref, ev)
		}
		m.focusPaneRef(ref)
	case strings.HasPrefix(node, paneBoxPrefix):
		ref := strings.TrimPrefix(node, paneBoxPrefix)
		if right {
			return m.openMenu(menuPane, ref, ev)
		}
		m.focusPaneRef(ref)
	case strings.HasPrefix(node, dividerPrefix):
		if !right {
			m.beginDividerDrag(node, ev)
		}
	}
	return app.None
}

func (m *model) beginDividerDrag(node string, ev *pb.MouseEvent) {
	rest := strings.TrimPrefix(node, dividerPrefix)
	parts := strings.Split(rest, ":")
	if len(parts) != 3 {
		return
	}
	ws := m.workspaceByID(parts[0])
	if ws == nil {
		return
	}
	t := tabByID(ws, parts[1])
	index, err := strconv.Atoi(parts[2])
	if err != nil || t == nil || index < 0 || index+1 >= len(t.Panes) {
		return
	}
	rects, _ := m.paneRects(t)
	extent := 0
	if index < len(rects) && index+1 < len(rects) {
		if t.Axis == "col" {
			extent = rects[index].H + rects[index+1].H + 1
		} else {
			extent = rects[index].W + rects[index+1].W + 1
		}
	}
	m.drag = &dividerDrag{
		ref:     tabRef(ws, t),
		index:   index,
		axis:    t.Axis,
		startX:  int(ev.GetX()),
		startY:  int(ev.GetY()),
		weights: append([]int(nil), t.Weights...),
		extent:  extent,
	}
}

func (m *model) onMouseDrag(ev *pb.MouseEvent) app.Cmd {
	if m.drag == nil {
		return app.None
	}
	d := m.drag
	parts := strings.Split(d.ref, ":")
	if len(parts) != 2 {
		return app.None
	}
	ws := m.workspaceByID(parts[0])
	if ws == nil {
		return app.None
	}
	t := tabByID(ws, parts[1])
	if t == nil || d.index < 0 || d.index+1 >= len(t.Panes) || d.extent <= 0 {
		return app.None
	}
	delta := 0
	if d.axis == "col" {
		delta = (int(ev.GetY()) - d.startY) * 100 / d.extent
	} else {
		delta = (int(ev.GetX()) - d.startX) * 100 / d.extent
	}
	if delta == 0 {
		return app.None
	}
	weights := append([]int(nil), d.weights...)
	adjustWeight(weights, d.index, d.index+1, delta)
	if sameInts(weights, d.weights) {
		return app.None
	}
	t.Weights = weights
	d.weights = weights
	d.startX = int(ev.GetX())
	d.startY = int(ev.GetY())
	d.moved = true
	m.structural()
	return app.None
}

func (m *model) onMouseRelease(ev *pb.MouseEvent) app.Cmd {
	if m.drag != nil {
		moved := m.drag.moved
		m.drag = nil
		if moved {
			m.structural()
		} else {
			m.touch()
		}
	}
	return app.None
}

// wheelTarget decides whether a wheel event belongs to the sidebar and which
// of its two panels should scroll.
func (m *model) wheelTarget(ev *pb.WheelEvent) (string, bool) {
	if m.sidebarCollapsed {
		return "", false
	}
	node := ev.GetNode()
	switch {
	case node == "herdr.sidebar.spaces" || strings.HasPrefix(node, spaceRowPrefix):
		return "spaces", true
	case node == "herdr.sidebar.agents" || strings.HasPrefix(node, agentRowPrefix):
		return "agents", true
	}
	if x := int(ev.GetX()); x < 1 || x > sidebarWidth {
		return "", false
	}
	if int(ev.GetY())-1 < m.sidebarAgentsHeight() {
		return "agents", true
	}
	return "spaces", true
}

func (m *model) onWheel(ev *pb.WheelEvent) app.Cmd {
	if ev == nil || ev.GetDelta() == 0 {
		return app.None
	}
	target, ok := m.wheelTarget(ev)
	if !ok {
		return app.None
	}
	delta := int(ev.GetDelta())
	if target == "spaces" {
		visible := m.sidebarSpacesHeight() - 1
		m.spacesOffset = widgets.ApplyWheel(m.spacesOffset, len(m.workspaces), visible, delta)
	} else {
		visible := m.sidebarAgentsHeight() - 1
		m.agentsOffset = widgets.ApplyWheel(m.agentsOffset, m.agentListLen(), visible, delta)
	}
	m.touch()
	return app.None
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// shiftedLetter folds "H" and "shift-h" to the uppercase letter.
func shiftedLetter(key string) (string, bool) {
	if len(key) == 1 && key[0] >= 'A' && key[0] <= 'Z' {
		return key, true
	}
	if rest, ok := strings.CutPrefix(key, "shift-"); ok && len(rest) == 1 && rest[0] >= 'a' && rest[0] <= 'z' {
		return strings.ToUpper(rest), true
	}
	return "", false
}

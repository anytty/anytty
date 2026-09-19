package main

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/config"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func findBox(root *pb.Box, id string) *pb.Box {
	if root == nil {
		return nil
	}
	if root.GetId() == id {
		return root
	}
	for _, child := range root.GetChildren() {
		if found := findBox(child, id); found != nil {
			return found
		}
	}
	return nil
}

func boxChildren(root *pb.Box) []*pb.Box { return root.GetChildren() }

func contentText(box *pb.Box) string {
	if box == nil || box.GetContent() == nil {
		return ""
	}
	if len(box.GetContent().GetLines()) > 0 {
		return joinLines(box.GetContent().GetLines())
	}
	return box.GetContent().GetText()
}

func joinLines(lines []string) string {
	text := ""
	for i, line := range lines {
		if i > 0 {
			text += "\n"
		}
		text += line
	}
	return text
}

func TestViewTerminalStructure(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	root, keys := m.view()
	box := root.Build()

	if box.GetFlow() != "col" || len(boxChildren(box)) != 3 {
		t.Fatalf("root = flow %q children %d, want col/3", box.GetFlow(), len(boxChildren(box)))
	}
	for _, id := range []string{"header", "body", "footer"} {
		if findBox(box, id) == nil {
			t.Fatalf("missing %s box", id)
		}
	}
	if keys.All || len(keys.Claim) == 0 || keys.Claim[0] != "ctrl-p" {
		t.Fatalf("keys = %+v", keys)
	}

	slot := findBox(box, m.focusSlot().id)
	if slot == nil || slot.GetContent().GetSelf() != "terminal:local:main" {
		t.Fatalf("slot box = %+v", slot)
	}
	if !slot.GetFocused() {
		t.Fatal("bound focused slot must carry focused=true in NORMAL")
	}
	input := slot.GetInput()
	if len(input) != 3 || input[0] != "key" || input[1] != "paste" || input[2] != "wheel" {
		t.Fatalf("terminal input = %v", input)
	}
	wantWidth := m.cols - sidebarWidth
	if int(slot.GetSize().GetWidth()) != wantWidth || int(slot.GetSize().GetHeight()) != m.rows-2 {
		t.Fatalf("terminal size = %+v, want %dx%d", slot.GetSize(), wantWidth, m.rows-2)
	}
}

func TestViewTabBarAndHeaderStyles(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	m.key(keyPress("ctrl-t"))
	m.key(keyPress("esc"))
	m.key(keyPress("ctrl-p"))
	m.key(keyPress("1"))

	box := buildView(m)
	header := findBox(box, "header")
	if header.GetFlow() != "row" || header.GetSize().GetHeight() != 1 {
		t.Fatalf("header = flow %q height %d", header.GetFlow(), header.GetSize().GetHeight())
	}
	workspace := findBox(box, "workspace")
	if workspace == nil || workspace.GetStyle() != m.style("chrome") || contentText(workspace) != " 󰙅 local " {
		t.Fatalf("workspace chip = %+v", workspace)
	}
	active := findBox(box, "tab:0")
	if active.GetStyle() != m.style("tab_active") || contentText(active) != "[⎇ 1:1]" {
		t.Fatalf("active tab = style %q text %q", active.GetStyle(), contentText(active))
	}
	inactive := findBox(box, "tab:1")
	if inactive.GetStyle() != m.style("tab_inactive") || contentText(inactive) != " ⎇ 2:2 " {
		t.Fatalf("inactive tab = style %q text %q", inactive.GetStyle(), contentText(inactive))
	}
	plus := findBox(box, "tab:new")
	if plus == nil || contentText(plus) != " 󰐕 " {
		t.Fatalf("new-tab box = %+v", plus)
	}
	m.mouse("press", "left", "tab:new", 60, 0)
	if len(m.tabs) != 3 || m.active != 2 {
		t.Fatalf("plus click: tabs=%d active=%d, want 3/2", len(m.tabs), m.active)
	}
}

func TestViewPaneModeClearsFocusAndClaimsAll(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	m.key(keyPress("ctrl-p"))

	root, keys := m.view()
	box := root.Build()
	slot := findBox(box, m.focusSlot().id)
	if slot.GetFocused() {
		t.Fatal("PANE mode must clear focused")
	}
	if !keys.All {
		t.Fatalf("PANE keys = %+v, want all=true", keys)
	}
}

func TestViewSplitAndEmptyPlaceholder(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	m.key(keyPress("ctrl-p"))
	m.key(keyPress("%"))
	root, _ := m.view()
	box := root.Build()

	panes := findBox(box, "panes")
	if panes.GetFlow() != "row" || len(boxChildren(panes)) != 3 {
		t.Fatalf("panes flow=%q children=%d, want row/3 (slot, gutter, slot)", panes.GetFlow(), len(boxChildren(panes)))
	}
	divider := findBox(box, "divider:0")
	if divider == nil || divider.GetSize().GetWidth() != 1 {
		t.Fatalf("divider = %+v", divider)
	}
	if len(divider.GetInput()) != 1 || divider.GetInput()[0] != "mouse" {
		t.Fatalf("divider input = %v", divider.GetInput())
	}
	if divider.GetStyle() != m.style("muted") || !contains(contentText(divider), "│") {
		t.Fatalf("gutter = style %q text %q, want self-drawn muted bars", divider.GetStyle(), contentText(divider))
	}
	empty := findBox(box, m.activeTab().slots[1].id)
	if empty == nil || len(boxChildren(empty)) == 0 {
		t.Fatalf("empty placeholder = %+v", empty)
	}
	text := treeText(empty)
	if !contains(text, "空槽") || !contains(text, "Ctrl-F 选择终端") || !contains(text, "Ctrl-P 面板命令") {
		t.Fatalf("empty placeholder chrome/hints = %q", text)
	}
	if got := boxChildren(empty)[0].GetStyle(); got != m.style("border") {
		t.Fatalf("empty placeholder border style = %q, want program theme border", got)
	}
	widths := 0
	for _, child := range boxChildren(panes) {
		widths += int(child.GetSize().GetWidth())
	}
	if widths != m.cols-sidebarWidth {
		t.Fatalf("pane widths sum = %d, want %d", widths, m.cols-sidebarWidth)
	}
}

func TestViewGapZeroRemovesGutter(t *testing.T) {
	m := newModel()
	testHello(m)
	m.mustGapZero()
	m.key(keyPress("ctrl-p"))
	m.key(keyPress("%"))
	box := buildView(m)
	panes := findBox(box, "panes")
	if len(boxChildren(panes)) != 2 {
		t.Fatalf("gap=0 panes children = %d, want 2", len(boxChildren(panes)))
	}
	for _, child := range boxChildren(panes) {
		if int(child.GetSize().GetWidth()) < 1 {
			t.Fatalf("gap=0 slot width = %d", child.GetSize().GetWidth())
		}
	}
}

func (m *model) mustGapZero() {
	off := 0
	cfg := config.Default()
	cfg.Gap = &off
	if err := m.applyConfig(cfg); err != nil {
		panic(err)
	}
}

func TestViewOverlayIsPosSubtree(t *testing.T) {
	m := newModel()
	testHello(m)
	m.sourcesEvent(nil)
	if m.mode != modePicker {
		t.Fatalf("mode = %v", m.mode)
	}
	root, _ := m.view()
	box := root.Build()
	overlay := findBox(box, "overlay:terminals")
	if overlay == nil || overlay.GetPos() == nil {
		t.Fatalf("picker overlay = %+v", overlay)
	}
	if overlay.GetPos().GetX() <= 0 || overlay.GetPos().GetY() <= 0 {
		t.Fatalf("overlay position = %+v, want centered", overlay.GetPos())
	}
	text := treeText(overlay)
	if !contains(text, "terminals") || !contains(text, "New terminal") || !contains(text, "select a terminal") {
		t.Fatalf("picker chrome/content = %q", text)
	}
	if got := boxChildren(overlay)[0].GetStyle(); got != m.style("accent") {
		t.Fatalf("overlay border style = %q, want program theme accent", got)
	}
}

func TestViewSidebarToggle(t *testing.T) {
	m := newModel()
	testHello(m)
	box := buildView(m)
	sidebar := findBox(box, "sidebar")
	if sidebar == nil || sidebar.GetSize().GetWidth() != sidebarWidth {
		t.Fatalf("sidebar = %+v, want visible by default", sidebar)
	}
	panes := findBox(box, "panes")
	if int(panes.GetSize().GetWidth()) != m.cols-sidebarWidth {
		t.Fatalf("panes width = %d, want %d", panes.GetSize().GetWidth(), m.cols-sidebarWidth)
	}
	m.key(keyPress("ctrl-w"))
	if findBox(buildView(m), "sidebar") != nil {
		t.Fatal("ctrl-w must hide the sidebar")
	}
	panes = findBox(buildView(m), "panes")
	if int(panes.GetSize().GetWidth()) != m.cols {
		t.Fatalf("panes width after toggle = %d, want %d", panes.GetSize().GetWidth(), m.cols)
	}
}

func TestViewFooterStatus(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")

	box := buildView(m)
	footer := findBox(box, "footer")
	if footer.GetFlow() != "row" || footer.GetSize().GetHeight() != 1 {
		t.Fatalf("footer = flow %q height %d", footer.GetFlow(), footer.GetSize().GetHeight())
	}
	if findBox(box, "footer.keys") == nil || findBox(box, "footer.status") == nil {
		t.Fatal("footer must carry key hints and status segments")
	}
	status := m.footerRight()
	for _, want := range []string{"local", "\U000F0645", "\U000F0E59", "1"} {
		if !contains(status, want) {
			t.Fatalf("footer right %q missing %q", status, want)
		}
	}
	if contains(status, "NORMAL") || contains(status, "tab 1/1") || contains(status, "slot 1/1") {
		t.Fatalf("footer right %q still carries v2-only chrome", status)
	}

	m.status = "bound term-2"
	if got := m.footerRight(); !contains(got, "bound term-2") {
		t.Fatalf("notice status = %q", got)
	}
	m.status = ""
}

func TestViewModeFooterHints(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	if !contains(joinKeys(m.footerKeys()), "P \uEBEB PANE") {
		t.Fatalf("NORMAL hints = %v", m.footerKeys())
	}
	m.key(keyPress("ctrl-p"))
	if !contains(joinKeys(m.footerKeys()), "X \U000F0156 CLOSE") ||
		!contains(joinKeys(m.footerKeys()), "% \uEB56 VSPLIT") {
		t.Fatalf("PANE hints = %v", m.footerKeys())
	}
	if contains(joinKeys(m.footerKeys()), "split-h") || contains(joinKeys(m.footerKeys()), "spacebar") {
		t.Fatalf("PANE hints leaked raw key names: %v", m.footerKeys())
	}
	m.mode = modeScroll
	if !contains(joinKeys(m.footerKeys()), "PGUP \U000F005D OLDER") {
		t.Fatalf("SCROLL hints = %v", m.footerKeys())
	}
}

func joinKeys(keys []string) string {
	out := ""
	for i, key := range keys {
		if i > 0 {
			out += " · "
		}
		out += key
	}
	return out
}

func TestViewKeybindingOverride(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	cfg := config.Default()
	cfg.Keybindings = map[string]string{"picker.open": "ctrl-g"}
	if err := m.applyConfig(cfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	root, keys := m.view()
	if !contains(joinKeys(keys.Claim), "ctrl-g") || contains(joinKeys(keys.Claim), "ctrl-f") {
		t.Fatalf("claim = %v, want ctrl-g instead of ctrl-f", keys.Claim)
	}
	_ = root

	m.key(keyPress("ctrl-f"))
	if m.mode != modeNormal {
		t.Fatalf("ctrl-f must be inert after rebinding, mode=%v", m.mode)
	}
	if !contains(joinKeys(m.footerKeys()), "G \U000F0C7C PICK") {
		t.Fatalf("footer hints must follow the binding: %v", m.footerKeys())
	}
	m.key(keyPress("ctrl-g"))
	if m.mode != modePicker {
		t.Fatalf("ctrl-g must open the picker, mode=%v", m.mode)
	}
}

func TestViewAutoAttachFirstDisabledShowsPicker(t *testing.T) {
	m := newModel()
	testHello(m)
	off := false
	cfg := config.Default()
	cfg.Startup.AutoAttachFirst = &off
	if err := m.applyConfig(cfg); err != nil {
		t.Fatal(err)
	}
	reqs := m.sourcesEvent([]*pb.Source{terminalSource("terminal:local:main", "main", false)})
	if len(reqs) != 0 || m.mode != modePicker {
		t.Fatalf("auto_attach_first=false: reqs=%v mode=%v, want picker", reqs, m.mode)
	}
	if m.focusSlot().sourceID != "" {
		t.Fatal("no slot may be bound without auto attach")
	}
}

// TestViewSendsExplicitProgramStyles pins the M4 model: the program sends
// explicit styles on the wire, never bare token names, and the palette comes
// from the program-side theme selected by tui2.json.
func TestViewSendsExplicitProgramStyles(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")

	dark := buildView(m)
	styles := collectStyles(dark)
	if len(styles) == 0 {
		t.Fatal("view carries no styles at all")
	}
	for _, style := range styles {
		if !strings.Contains(style, ":") {
			t.Fatalf("style %q is not explicit (no fg:/bg: segment)", style)
		}
	}
	if got := findBox(dark, "workspace").GetStyle(); !strings.Contains(got, "#070611") {
		t.Fatalf("default chrome style = %q, want the recommended palette", got)
	}

	light := config.Default()
	light.Theme = "light"
	if err := m.applyConfig(light); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if got := findBox(buildView(m), "workspace").GetStyle(); !strings.Contains(got, "#e7e3ef") {
		t.Fatalf("light chrome style = %q, want the light palette", got)
	}

	darkCfg := config.Default()
	darkCfg.Theme = "dark"
	if err := m.applyConfig(darkCfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if got := findBox(buildView(m), "workspace").GetStyle(); !strings.Contains(got, "#161823") {
		t.Fatalf("dark chrome style = %q, want the dark palette", got)
	}
}

func collectStyles(root *pb.Box) []string {
	var out []string
	if root.GetStyle() != "" {
		out = append(out, root.GetStyle())
	}
	for _, child := range root.GetChildren() {
		out = append(out, collectStyles(child)...)
	}
	return out
}

// treeText concatenates the content of a subtree; overlays style one box
// per row, so the text lives in the children.
func treeText(root *pb.Box) string {
	text := contentText(root)
	for _, child := range root.GetChildren() {
		text += "\n" + treeText(child)
	}
	return text
}

func buildView(m *model) *pb.Box {
	root, _ := m.view()
	return root.Build()
}

func contains(text, want string) bool {
	for i := 0; i+len(want) <= len(text); i++ {
		if text[i:i+len(want)] == want {
			return true
		}
	}
	return false
}

// TestViewSlotTitleBarButtons pins the M17 title bar contract: every slot
// carries a program-drawn 1-row title bar (marker + title on the left) with
// one mouse-input box per action button, styled from the program theme.
func TestViewSlotTitleBarButtons(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	box := buildView(m)

	assertSlotButtons := func(s *slot, labelStyle, buttonStyle string) {
		t.Helper()
		bar := findBox(box, "title:"+s.id)
		if bar == nil {
			t.Fatalf("slot %s has no title bar", s.id)
		}
		if bar.GetSize().GetHeight() != 1 || bar.GetSize().GetWidth() <= 0 {
			t.Fatalf("title bar geometry = %+v", bar.GetSize())
		}
		if got := boxChildren(bar)[0].GetStyle(); got != labelStyle {
			t.Fatalf("slot %s label style = %q, want %q (focus %d, slots %d)", s.id, got, labelStyle, m.activeTab().focus, len(m.activeTab().slots))
		}
		for _, btn := range m.buttonIcons() {
			node := findBox(box, slotButtonNode(s.id, btn.action))
			if node == nil {
				t.Fatalf("slot %s missing %s button", s.id, btn.action)
			}
			if contentText(node) != btn.icon {
				t.Fatalf("button %s text = %q, want %q", btn.action, contentText(node), btn.icon)
			}
			if node.GetSize().GetWidth() != 1 || node.GetSize().GetHeight() != 1 {
				t.Fatalf("button %s size = %+v, want 1x1", btn.action, node.GetSize())
			}
			if len(node.GetInput()) != 1 || node.GetInput()[0] != "mouse" {
				t.Fatalf("button %s input = %v, want [mouse]", btn.action, node.GetInput())
			}
			if got := node.GetStyle(); got != buttonStyle {
				t.Fatalf("button %s style = %q, want %q", btn.action, got, buttonStyle)
			}
		}
	}

	first := m.focusSlot()
	assertSlotButtons(first, m.style("border_focus"), m.style("button_hover"))
	if text := treeText(findBox(box, "title:"+first.id)); !contains(text, "▎t") {
		t.Fatalf("title bar text = %q, want the focus marker plus title", text)
	}

	// Two slots: each keeps its own four buttons; clicking the bound slot's
	// title marker focuses it and paints the label with border_focus.
	m.key(keyPress("ctrl-p"))
	m.key(keyPress("%"))
	m.key(keyPress("esc"))
	bound := m.activeTab().slots[0]
	m.mouse("press", "left", "title:"+bound.id, 1, 1)
	if m.focusSlot() != bound {
		t.Fatalf("title-bar click must focus the slot: focus = %q", m.focusSlot().id)
	}
	box = buildView(m)
	for _, s := range m.activeTab().slots {
		labelStyle, buttonStyle := m.style("muted"), m.style("button")
		if s == bound {
			labelStyle, buttonStyle = m.style("border_focus"), m.style("button_hover")
		}
		assertSlotButtons(s, labelStyle, buttonStyle)
	}
	if ids := []string{
		slotButtonNode(m.activeTab().slots[0].id, slotButtonClose),
		slotButtonNode(m.activeTab().slots[1].id, slotButtonClose),
	}; ids[0] == ids[1] {
		t.Fatalf("button ids must be slot-scoped: %v", ids)
	}
}

// TestViewSlotTitleBarExitedRestartWarning: an exited terminal turns the
// restart button warning-colored (the action is available) and the label
// border_dead, with the exit badge owned by the program title row.
func TestViewSlotTitleBarExitedRestartWarning(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	m.sourcesEvent([]*pb.Source{terminalSource("terminal:local:main", "main", true)})
	s := m.focusSlot()
	box := buildView(m)

	restart := findBox(box, slotButtonNode(s.id, slotButtonRestart))
	if got := restart.GetStyle(); got != m.style("warning") {
		t.Fatalf("exited restart style = %q, want warning", got)
	}
	if got := findBox(box, slotButtonNode(s.id, slotButtonClose)).GetStyle(); got != m.style("button") {
		t.Fatalf("exited close style = %q, want theme button", got)
	}
	bar := findBox(box, "title:"+s.id)
	if got := boxChildren(bar)[0].GetStyle(); got != m.style("border_dead") {
		t.Fatalf("exited label style = %q, want border_dead", got)
	}
	if text := treeText(bar); !contains(text, "[exited 0]") {
		t.Fatalf("exited title text = %q, want the [exited N] badge", text)
	}

	// The scrollback badge also lives in the program title row.
	s.scrollOffset = 3
	if text := treeText(findBox(buildView(m), "title:"+s.id)); !contains(text, "[↑3]") {
		t.Fatalf("scrolled title text = %q, want the [↑N] badge", text)
	}
}

// TestViewSlotButtonPressedAndPending: a press highlights the button with the
// pressed theme slot and flags the slot pending; the release clears the mark
// and the response clears the pending indicator.
func TestViewSlotButtonPressedAndPending(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	s := m.focusSlot()
	node := slotButtonNode(s.id, slotButtonRestart)

	reqs := m.mouse("press", "left", node, 3, 0)
	if len(reqs) != 1 || reqs[0].Method != "terminal.restart" {
		t.Fatalf("restart requests = %+v", reqs)
	}
	if m.pressed != node {
		t.Fatalf("pressed = %q, want %q", m.pressed, node)
	}
	box := buildView(m)
	if got := findBox(box, node).GetStyle(); got != m.style("button_pressed") {
		t.Fatalf("pressed button style = %q, want theme button_pressed", got)
	}
	if text := treeText(findBox(box, "title:"+s.id)); !contains(text, "…") {
		t.Fatalf("pending title text = %q, want the pending indicator", text)
	}

	m.mouse("release", "left", node, 3, 0)
	if m.pressed != "" {
		t.Fatalf("pressed after release = %q, want empty", m.pressed)
	}
	if got := findBox(buildView(m), node).GetStyle(); got != m.style("button_hover") {
		t.Fatalf("released button style = %q, want theme button_hover", got)
	}

	reqs[0].After(&pb.Response{Ok: true})
	if s.pending != "" {
		t.Fatalf("pending after response = %q, want empty", s.pending)
	}
	if text := treeText(findBox(buildView(m), "title:"+s.id)); contains(text, "…") {
		t.Fatalf("pending indicator must disappear after the response: %q", text)
	}
}

// TestViewSlotTitleBarThemeSwitch: the button styles come from the program
// theme, so tui2.json theme=light repaints them without a frame concept.
func TestViewSlotTitleBarThemeSwitch(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	s := m.focusSlot()
	dark := findBox(buildView(m), slotButtonNode(s.id, slotButtonRestart)).GetStyle()

	light := config.Default()
	light.Theme = "light"
	if err := m.applyConfig(light); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	got := findBox(buildView(m), slotButtonNode(s.id, slotButtonRestart)).GetStyle()
	if got == dark || got != m.style("button_hover") {
		t.Fatalf("light button style = %q (dark %q), want the light theme button_hover", got, dark)
	}
}

// TestViewTerminalSlotCarriesThemeChromeProps pins the M4 contract: the shell
// pushes the program-side theme as explicit chrome props on every bound
// terminal slot, and a tui2.json theme switch changes them.
func TestViewTerminalSlotCarriesThemeChromeProps(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")

	slot := findBox(buildView(m), m.focusSlot().id)
	if slot == nil || slot.GetContent().GetSelf() == "" {
		t.Fatalf("bound slot = %+v", slot)
	}
	props := slot.GetContent().GetProps()
	for _, key := range []string{
		"chrome.border",
		"chrome.title",
		"chrome.border_focus",
		"chrome.border_dead",
		"chrome.badge",
	} {
		got, ok := props[key]
		if !ok || got == "" {
			t.Fatalf("slot props = %v, missing non-empty %s", props, key)
		}
		if !strings.Contains(got, ":") {
			t.Fatalf("prop %s = %q, want an explicit style string", key, got)
		}
		if want := m.style(keyStyleSlot(key)); got != want {
			t.Fatalf("prop %s = %q, want theme slot %q", key, got, want)
		}
	}

	light := config.Default()
	light.Theme = "light"
	if err := m.applyConfig(light); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	after := findBox(buildView(m), m.focusSlot().id).GetContent().GetProps()
	if after["chrome.border"] == props["chrome.border"] {
		t.Fatalf("theme switch kept border prop %q", after["chrome.border"])
	}
	if !strings.Contains(after["chrome.border"], "#b3adc0") {
		t.Fatalf("light border prop = %q, want the light palette", after["chrome.border"])
	}
	if !strings.Contains(after["chrome.title"], "#6f6a7c") {
		t.Fatalf("light title prop = %q, want the light muted color", after["chrome.title"])
	}
}

// keyStyleSlot maps a prop key back to the theme slot the shell resolves it
// from (the contract itself lives in components/terminal).
func keyStyleSlot(key string) string {
	switch key {
	case "chrome.border":
		return "border"
	case "chrome.title":
		return "muted"
	case "chrome.border_focus":
		return "border_focus"
	case "chrome.border_dead":
		return "border_dead"
	case "chrome.badge":
		return "warning"
	default:
		return ""
	}
}

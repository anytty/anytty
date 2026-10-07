package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/anytty/anytty/clients/tui/sdk"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// layoutEntry is one solved layout node: a pane leaf card or a split (with
// the 1-cell drag hit region on the shared card border).
type layoutEntry struct {
	pane     *pane
	node     *split
	r        rect
	boundary rect
}

func (m *model) bodyRect() rect {
	y := 0
	if m.headerVisible {
		y = 1
	}
	h := m.rows - y
	if m.footerVisible {
		h--
	}
	return rect{0, y, m.cols, maxInt(0, h)}
}

// paneRects solves the active tab: every leaf is a full card and sibling
// cards are adjacent (their own borders touch, exactly the old layout: no
// extra separator column/row). Each Split still exposes a 1-cell drag hit
// region on the shared border. A zoomed pane fills the whole body (old v3
// panel.toggle_zoom).
func (m *model) paneRects(t *tab) (rect, []layoutEntry) {
	body := m.bodyRect()
	var entries []layoutEntry
	if t == nil {
		return body, entries
	}
	if m.zoomPane != "" {
		for _, p := range t.panes {
			if p.id == m.zoomPane {
				return body, []layoutEntry{{pane: p, r: body}}
			}
		}
	}
	if t.root != nil {
		m.layoutNode(t.root, body, &entries)
	}
	return body, entries
}

func (m *model) layoutNode(node treeNode, r rect, entries *[]layoutEntry) {
	switch n := node.(type) {
	case *leaf:
		n.rect = r
		*entries = append(*entries, layoutEntry{pane: n.pane, r: r})
	case *split:
		n.rect = r
		if n.orient == "row" {
			first := n.splitFirstExtent(r.w)
			m.layoutNode(n.a, rect{r.x, r.y, first, r.h}, entries)
			*entries = append(*entries, layoutEntry{node: n, r: r, boundary: rect{r.x + first, r.y, 1, r.h}})
			m.layoutNode(n.b, rect{r.x + first, r.y, r.w - first, r.h}, entries)
			return
		}
		first := n.splitFirstExtent(r.h)
		m.layoutNode(n.a, rect{r.x, r.y, r.w, first}, entries)
		*entries = append(*entries, layoutEntry{node: n, r: r, boundary: rect{r.x, r.y + first, r.w, 1}})
		m.layoutNode(n.b, rect{r.x, r.y + first, r.w, r.h - first}, entries)
	}
}

func (m *model) splitEntries(t *tab) []layoutEntry {
	_, entries := m.paneRects(t)
	var out []layoutEntry
	for _, entry := range entries {
		if entry.node != nil {
			out = append(out, entry)
		}
	}
	return out
}

// addRun appends one positioned text box (the reference add_run). width <= 0
// derives the width from the text; a zero-width run is dropped.
func addRun(out *[]*sdk.Builder, x, y int, text, style, node string, mouse bool, width int) {
	if width <= 0 {
		width = sdk.DisplayWidth(text)
	}
	if width <= 0 {
		return
	}
	box := sdk.Text(text).Pos(x, y).Width(width).Height(1)
	if style != "" {
		box.Style(style)
	}
	if node != "" {
		box.ID(node)
	}
	if mouse {
		box.Input("mouse")
	}
	*out = append(*out, box)
}

func dimStyle(style string) string {
	if style == "" || strings.Contains(";"+style+";", ";dim;") {
		return style
	}
	return style + ";dim"
}

// ------------------------------------------------------------------ view

// View materializes the whole screen: header, cards, floating windows,
// overlays, toast and footer, in the reference declaration order.
func (m *model) View() *pb.Box {
	out := []*sdk.Builder{}
	if m.headerVisible {
		m.headerNodes(&out)
	}
	m.bodyNodes(&out)
	for _, f := range m.floatings {
		m.floatingNodes(&out, f)
	}
	if m.overlay != "" {
		m.overlayNodes(&out)
		m.promptSuggestionNodes(&out)
	}
	m.toastNodes(&out)
	if m.footerVisible {
		// The active panel's copy search replaces the global footer row, like
		// the legacy SearchBarVM.
		if st := m.copySearchFooterState(); st != nil {
			m.copySearchFooterNodes(&out, st)
		} else {
			m.footerNodes(&out)
		}
	}
	return sdk.Stack(out...).ID("root").Build()
}

// copySearchFooterState returns the active copy session whose search bar should
// replace the footer, or nil. It mirrors the legacy SearchBarVisible rule: the
// bar shows while editing, scanning, or whenever a query/error is present.
func (m *model) copySearchFooterState() *copyState {
	if m.overlay != "" || m.mode != modeLive {
		return nil
	}
	st := m.copyFor(m.focusContentPane())
	if st == nil {
		return nil
	}
	if !st.searching && strings.TrimSpace(st.query) == "" && st.searchErr == "" {
		return nil
	}
	return st
}

// copySearchFooterNodes paints the search bar on the footer row: left prefix +
// query, right status + hint, and a reverse-video caret on the query while
// editing (the legacy bar cursor).
func (m *model) copySearchFooterNodes(out *[]*sdk.Builder, st *copyState) {
	y := m.rows - 1
	left, right := m.copySearchFooterRuns(st)
	left = trimRuns(left, m.cols)
	leftWidth := runsWidth(left)
	right = trimRuns(right, m.cols-leftWidth)
	rightWidth := runsWidth(right)
	pad := maxInt(0, m.cols-leftWidth-rightWidth)
	x := 0
	prefixEnd := -1
	valueX := -1
	for _, run := range left {
		width := sdk.DisplayWidth(run.text)
		if width <= 0 {
			continue
		}
		if run.style == stAccent {
			prefixEnd = x + width
		}
		if run.style == stContent {
			valueX = x
		}
		addRun(out, x, y, run.text, run.style, run.node, run.node != "", width)
		x += width
	}
	if pad > 0 {
		addRun(out, x, y, "", stFooterFill, "", false, pad)
	}
	x += pad
	for _, run := range right {
		width := sdk.DisplayWidth(run.text)
		if width <= 0 {
			continue
		}
		addRun(out, x, y, run.text, run.style, run.node, run.node != "", width)
		x += width
	}
	if st.searching {
		col := minInt(maxInt(0, m.cols-1), m.copySearchCaretCol(st, prefixEnd, valueX))
		addRun(out, col, y, " ", copySearchCaretStyle, "", false, 1)
	}
}

// copySearchCaretCol is the footer column of the edit caret: the prefix end
// when the query is empty, otherwise prefix + the display width of the query
// runes before the cursor.
func (m *model) copySearchCaretCol(st *copyState, prefixEnd, valueX int) int {
	if valueX < 0 || prefixEnd < 0 {
		return maxInt(0, prefixEnd)
	}
	runes := []rune(st.query)
	col := minInt(maxInt(0, st.searchCol), len(runes))
	return valueX + sdk.DisplayWidth(string(runes[:col]))
}

func (m *model) bodyNodes(out *[]*sdk.Builder) {
	t := m.activeTab()
	_, entries := m.paneRects(t)
	focused := m.focusPane()
	paneFocus := m.mode == modeLive && m.activeFloat == "" && !m.copyActive()
	for _, entry := range entries {
		if entry.node != nil {
			m.boundaryNodes(out, entry)
			continue
		}
		p := entry.pane
		// Keep the active panel fully bright and dim every sibling while a
		// panel is focused, matching the old TI overlay across pane/resize and
		// floating scenes. Modal overlays dim themselves separately.
		dimmed := m.overlay == "" && focused != nil && p != focused
		m.cardNodes(out, p, entry.r, p == focused, paneFocus && p == focused, dimmed)
	}
}

func (m *model) headerNodes(out *[]*sdk.Builder) {
	x := 0
	for _, run := range m.headerRuns() {
		width := sdk.DisplayWidth(run.text)
		if x >= m.cols {
			break
		}
		text := run.text
		if x+width > m.cols {
			text = sdk.Truncate(text, m.cols-x)
			width = sdk.DisplayWidth(text)
		}
		addRun(out, x, 0, text, run.style, run.node, run.mouse, width)
		x += width
	}
	if x < m.cols {
		addRun(out, x, 0, "", stHeaderFill, "", false, m.cols-x)
	}
}

// cardNodes draws one leaf card: full border + pane title/action chrome +
// content. The focused card keeps its accent frame in every mode;
// contentFocus only controls the embedded terminal box.
func (m *model) cardNodes(out *[]*sdk.Builder, p *pane, r rect, active, contentFocus, dimmed bool) {
	runs := m.paneRunsRect(p, active, r.w, r.h)
	focusNode := ""
	if m.paneSource(p) == nil {
		focusNode = "pane:" + p.id + ":focus"
	}
	rx := r.x
	for _, run := range runs {
		width := sdk.DisplayWidth(run.text)
		if rx >= m.cols {
			break
		}
		text := run.text
		if rx+width > m.cols {
			text = sdk.Truncate(text, m.cols-rx)
			width = sdk.DisplayWidth(text)
		}
		style := run.style
		if dimmed {
			style = dimStyle(style)
		}
		node, mouse := run.node, run.mouse
		if node == "" && focusNode != "" {
			node, mouse = focusNode, true
		}
		addRun(out, rx, r.y, text, style, node, mouse, width)
		rx += width
	}
	if r.h <= 0 || r.w <= 0 {
		return
	}
	frame := stPanelBorder
	if active {
		frame = stAccent
	}
	// The renderer decides per pane: an open copy session forces the yellow
	// history-border frame (legacy paneChromeStyle), matching paneRunsRect.
	if m.copyFor(p) != nil {
		frame = stHistoryBorder
	}
	if dimmed {
		frame = dimStyle(frame)
	}
	borderNode := ""
	if focusNode != "" {
		borderNode = focusNode
	}
	for row := 1; row < r.h-1; row++ {
		addRun(out, r.x, r.y+row, "\u2502", frame, borderNode, borderNode != "", 1)
		addRun(out, r.x+r.w-1, r.y+row, "\u2502", frame, borderNode, borderNode != "", 1)
	}
	addRun(out, r.x, r.y+r.h-1, "\u2514"+strings.Repeat("\u2500", maxInt(0, r.w-2))+"\u2518", frame, borderNode, borderNode != "", 0)
	m.subPaneNodes(out, p, rect{r.x + 1, r.y + 1, maxInt(0, r.w-2), maxInt(0, r.h-2)}, contentFocus, dimmed)

	// Clipping markers on the card border, drawn last so they win over the
	// rules above: the legacy renderer overlays them on the border without
	// touching the content layer (render/content_overflow_marker.go). The top
	// marker is drawn on the top border row (paneRunsRect); the left marker sits
	// on the top content row's left edge, the right marker on the bottom content
	// row's right edge, and the bottom marker just before the bottom-right
	// corner (legacy contentOverflowVerticalMarkerY).
	//
	// Two sources of horizontal/vertical clipping share this chrome:
	//   - a frozen copy/scrollback pane whose window is scrolled or wider than
	//     the content area (copyOverflow), and
	//   - a live extent pane whose pane content is smaller than (or panned away
	//     from) the source's authoritative extent (the legacy extent model: the
	//     owner resizes the PTY to fit and only clips when the user pans/aligns,
	//     while a follower shows the owner's size and marks every clipped edge).
	if r.w > 2 && r.h > 2 {
		leftOverflow, rightOverflow, _, bottomOverflow := m.paneOverflow(p, r.w-2, r.h-2)
		markerStyle := stOverflowStyle
		if dimmed {
			markerStyle = dimStyle(markerStyle)
		}
		if leftOverflow {
			addRun(out, r.x, r.y+1, glyphOverflowLeft, markerStyle, borderNode, borderNode != "", 1)
		}
		if rightOverflow {
			addRun(out, r.x+r.w-1, r.y+r.h-2, glyphOverflowRight, markerStyle, borderNode, borderNode != "", 1)
		}
		if bottomOverflow {
			addRun(out, r.x+r.w-2, r.y+r.h-1, glyphOverflowBottom, markerStyle, borderNode, borderNode != "", 1)
		}
	}
}

// paneOverflow reports the directions a pane's content is clipped. It merges
// the frozen copy-window overflow (copyOverflow) with the live extent overflow:
// a source-bound pane is clipped whenever the source's authoritative extent,
// positioned by the pane's view-local content layout, extends outside the pane
// content area. This is the legacy contentViewportOverflow rule: Left when
// extent.X < 0, Right when X+Cols > width, Top when Y < 0, Bottom when
// Y+Rows > height. The terminal box itself is always full-bleed; the extent is a
// drawn content footprint (content.offset/content.size), so the existing
// copy/non-host goldens are unchanged.
func (m *model) paneOverflow(p *pane, contentWidth, contentHeight int) (left, right, top, bottom bool) {
	if st := m.copyFor(p); st != nil {
		left, right, top, bottom = m.copyOverflow(st, contentWidth)
	}
	if !m.paneRendersExtent(p) {
		return left, right, top, bottom
	}
	content := rect{0, 0, maxInt(0, contentWidth), maxInt(0, contentHeight)}
	x, y, cols, rows := m.paneContentExtent(p, content)
	if x < 0 {
		left = true
	}
	if x+cols > content.w {
		right = true
	}
	if y < 0 {
		top = true
	}
	if y+rows > content.h {
		bottom = true
	}
	return left, right, top, bottom
}

// paneRendersExtent reports whether p is drawn by the host as a live terminal
// extent box (rather than a program-drawn line buffer): the host paints the live
// path, the pane is source-bound, and it is not showing a frozen copy window.
// Both the owner and its followers send their extent as content.offset/size.
func (m *model) paneRendersExtent(p *pane) bool {
	if p == nil || p.sourceID == "" || m.demo || !m.host {
		return false
	}
	// A copy/scrollback session owns the pane's interaction viewport; the
	// extent framing (and its placeholder mask) must not fight it, so a pane
	// showing a frozen copy window is never treated as an extent pane.
	if m.copyFor(p) != nil {
		return false
	}
	src := m.paneSource(p)
	return src != nil && src.GetKind() == "terminal"
}

// boundaryNodes is the split's drag hit region: the shared card border between
// the two children (the right card's left border column, or the bottom card's
// top border row). It draws exactly the cells the b-side cards already draw,
// with their own focus style, so the pixels are unchanged while the host can
// hit-test and capture the drag. The old renderer uses the same 1-cell region.
func (m *model) boundaryNodes(out *[]*sdk.Builder, entry layoutEntry) {
	sp := entry.node
	t := m.activeTab()
	if sp == nil || t == nil {
		return
	}
	node := "divider:" + t.id + ":" + strconv.Itoa(sp.seq)
	focused := m.focusPane()
	styleFor := func(p *pane) string {
		// The shared divider carries the b-side pane's own frame style, so a
		// copy/scrollback pane keeps the yellow history border here too while a
		// sibling border stays muted or accent, matching mergeBoxCellStyle's
		// history-over-accent priority in the legacy renderer.
		if m.copyFor(p) != nil {
			return stHistoryBorder
		}
		if p == focused {
			return stAccent
		}
		return stPanelBorder
	}
	if sp.orient == "row" {
		for _, lf := range leafNodes(sp.b) {
			if lf.rect.x != entry.boundary.x {
				continue
			}
			style := styleFor(lf.pane)
			for row := lf.rect.y; row < lf.rect.y+lf.rect.h; row++ {
				addRun(out, lf.rect.x, row, "\u2502", style, node, true, 1)
			}
		}
		return
	}
	for _, lf := range leafNodes(sp.b) {
		if lf.rect.y != entry.boundary.y {
			continue
		}
		addRun(out, lf.rect.x, lf.rect.y, strings.Repeat("\u2500", maxInt(0, lf.rect.w)),
			styleFor(lf.pane), node, true, maxInt(0, lf.rect.w))
	}
}

func (m *model) subPaneNodes(out *[]*sdk.Builder, p *pane, r rect, active, dimmed bool) {
	src := m.paneSource(p)
	if src != nil && !m.demo && m.host {
		props := map[string]string{"chrome.inset": "0"}
		if dimmed {
			props["chrome.dim"] = "1"
		}
		// The program designates the resize owner explicitly: exactly the
		// sourceOwnerPane declares chrome.owner=1 so the host resizes the single
		// PTY to that pane's content rect. A follower never declares it, so
		// focusing a follower cannot steal the size (manual ownership, the legacy
		// panel.take_owner model).
		if m.paneOwnsSource(p) {
			props["chrome.owner"] = "1"
		}
		st := m.copyFor(p)
		if st != nil {
			// The copy scene lives in the program: the host paints the
			// cursor/selection/matches and resolves terminal.copy{sel}
			// against the window the view shows.
			st.cols = r.w
			st.viewRows = r.h
			m.clampCopyCursorToViewport(st)
			for key, value := range m.copyProps(st) {
				props[key] = value
			}
		}
		// One terminal source has one size, owned by a single pane. The owner
		// drives the PTY resize via chrome.owner=1, so its terminal box is
		// always declared at the FULL pane content rect r: the PTY stays
		// full-bleed for every pane, and the view-local content layout
		// (Ctrl-R align/center/pan) is expressed purely as a content.offset /
		// content.size prop that shifts the drawn screen inside that box. The
		// component masks the cells outside the extent footprint with the dim
		// `·` placeholder and the caller draws the border clipping markers.
		//
		// A frozen copy/scrollback session owns the pane viewport and must stay
		// unshifted (its own window already positions the rows): the content
		// layout framing is declared only on the live path.
		if st == nil {
			props[terminalPropContentOffset] = m.contentOffsetProp(p, r)
			props[terminalPropContentSize] = m.contentSizeProp(p, r)
			props[terminalPropPlaceholder] = stExtentPlaceholder
		}
		term := sdk.Terminal(p.sourceID).ID(p.id).Pos(r.x, r.y).Width(r.w).Height(r.h).
			Props(props).
			Input("key", "paste", "wheel").
			Focused(active)
		if st != nil {
			// The program cursor wins over the PTY cursor (the compositor
			// prefers the kernel frame cursor). The edit caret lives in the
			// footer search bar, so the panel keeps the selection cursor.
			term.Cursor(st.cursorRow, st.cursorCol, "block")
		}
		*out = append(*out, term)
		return
	}
	if src == nil {
		// A pane with no source normally shows the empty-panel CTA, but a frozen
		// copy window replaces that content entirely. (Live terminals always
		// have a source, so this path is the program-drawn copy window used by
		// the offline rasterizer and tests.)
		if st := m.copyFor(p); st != nil {
			m.copyWindowNodes(out, p, r, st, dimmed)
			return
		}
		// Only the focused empty panel carries the CTA selection highlight;
		// background panels render the default (first) entry, like main.
		m.emptyPaneNodes(out, p, r, p == m.focusContentPane(), dimmed)
		return
	}
	lines := p.lines
	if len(lines) == 0 {
		lines = []string{""}
	}
	for index := 0; index < r.h; index++ {
		text := ""
		if index < len(lines) {
			text = lines[index]
		}
		style := stContent
		if dimmed {
			style = dimStyle(style)
		}
		addRun(out, r.x, r.y+index, sdk.Truncate(text, r.w), style,
			"pane:"+p.id+":focus", true, r.w)
	}
}

// copyWindowNodes paints a frozen copy/scrollback window in program-drawn
// panes (the offline rasterizer and tests; live terminals are painted by the
// host component). It mirrors render.RenderContentViewport: the window scrolls
// by the session offset, each row is clipped to the pane width, and any rows
// the window does not cover are filled with the dim extent placeholder dots so
// the pane behind never shows through (recommended `extent_placeholder`).
func (m *model) copyWindowNodes(out *[]*sdk.Builder, p *pane, r rect, st *copyState, dimmed bool) {
	if r.w <= 0 || r.h <= 0 || st == nil {
		return
	}
	st.cols = r.w
	st.viewRows = r.h
	// st.rows is the window the host already positioned at st.offset, so it is
	// painted from the top of the content area; the session's offset only
	// drives the overflow markers (older/newer content outside the window).
	windowStyle := stContent
	if dimmed {
		windowStyle = dimStyle(windowStyle)
	}
	placeholderStyle := stExtentPlaceholder
	if dimmed {
		placeholderStyle = dimStyle(placeholderStyle)
	}
	for row := 0; row < r.h; row++ {
		text := ""
		style := windowStyle
		if row < len(st.rows) {
			text = sdk.Truncate(st.rows[row], r.w)
		} else {
			// The window does not cover this row: fill it with the dim extent
			// dots so the pane behind never shows through.
			text = strings.Repeat(extentPlaceholder, r.w)
			style = placeholderStyle
		}
		addRun(out, r.x, r.y+row, text, style, "pane:"+p.id+":focus", true, r.w)
	}
}

func (m *model) emptyPaneNodes(out *[]*sdk.Builder, p *pane, r rect, active, dimmed bool) {
	if r.w <= 0 || r.h <= 0 {
		return
	}
	// The empty content area has no terminal component to provide a mouse hit
	// target. Keep the whole area clickable so selecting a blank panel still
	// moves focus and updates its border highlight; the smaller CTA boxes below
	// win hit testing when the user clicks an action label.
	*out = append(*out, sdk.Box().ID("pane:"+p.id+":focus").Pos(r.x, r.y).
		Width(r.w).Height(r.h).Input("mouse"))
	lines := []struct{ text, style, action string }{
		{"○ No terminal connected", stOverlay, ""},
		{"Choose a terminal or create one.", stMuted, ""},
		{"", stMuted, ""},
		{"Choose how to start", stMuted, ""},
	}
	// main renders the CTA list with the highlighted entry bracketed and the
	// rest in square brackets; the same selection is used by keyboard and mouse.
	actions := []struct{ label, style, action string }{
		{"Attach existing terminal", stAccent, "empty-attach"},
		{"Create new terminal", stSuccess, "empty-create"},
		{"Open terminal manager", stOverlay, "empty-manager"},
		{"Close pane", stDanger, "close"},
	}
	selected := 0
	if active {
		selected = clampInt(m.emptyPaneSel, 0, len(actions)-1)
	}
	for index, action := range actions {
		text := "[ " + action.label + " ]"
		if index == selected {
			text = "► " + action.label + " ◄"
		}
		lines = append(lines, struct{ text, style, action string }{text, action.style, action.action})
	}
	start := maxInt(0, (r.h-len(lines))/2)
	for i, line := range lines {
		if start+i >= r.h {
			break
		}
		width := sdk.DisplayWidth(line.text)
		if width == 0 {
			continue
		}
		x := r.x + maxInt(0, (r.w-width)/2)
		style := line.style
		if dimmed {
			style = dimStyle(style)
		}
		addRun(out, x, r.y+start+i, line.text, style, "pane:"+p.id+":"+line.action, line.action != "", width)
	}
}

// --------------------------------------------------------------- floating

func (m *model) floatingRuns(f *floating, active bool, width int) []paneRun {
	frame := stPanelBorder
	if active {
		frame = stAccent
	}
	type glyphAction struct {
		name  string
		glyph string
	}
	actions := []glyphAction{
		{"center", glyphCenter}, {"collapse", glyphCollaps},
		{"zoom", glyphZoom}, {"close", glyphClose},
	}
	actionWidth := actionGroupWidth(len(actions))
	if actionWidth > width-6 {
		actions = actions[len(actions)-2:]
		actionWidth = actionGroupWidth(len(actions))
	}
	if actionWidth > width-5 {
		actions = nil
		actionWidth = 0
	}
	innerRight := width - 1
	actionX := innerRight
	rightLimit := innerRight
	if len(actions) > 0 {
		actionX = innerRight - actionWidth - 1
		rightLimit = actionX - 1
	}
	lock := " " + glyphUnlock + " "
	if f.pane.locked {
		lock = " " + glyphLocked + " "
	}
	paneTitle := m.paneTitle(f.pane)
	title := sdk.Truncate(paneTitle, maxInt(0, rightLimit-2-sdk.DisplayWidth(lock)))
	titleText := ""
	if title != "" {
		titleText = " " + title + " "
	}
	runs := []paneRun{
		{"\u250c", frame, "", false},
		{"\u2500", frame, "", false},
		{lock, frame, "float:" + f.id + ":lock", true},
		{titleText, frame, "float:" + f.id + ":title", true},
	}
	dragNode := "float:" + f.id + ":title"
	x := 2 + sdk.DisplayWidth(lock) + sdk.DisplayWidth(titleText)
	if x < rightLimit {
		runs = append(runs, paneRun{strings.Repeat("\u2500", rightLimit-x), frame, dragNode, true})
		x = rightLimit
	}
	for index, action := range actions {
		if x < actionX {
			runs = append(runs, paneRun{strings.Repeat("\u2500", actionX-x), frame, "", false})
			x = actionX
		}
		if index == 0 {
			runs = append(runs, paneRun{edgeL, frame, "", false})
			x++
		}
		itemStyle := stInactiveGroup
		if active {
			itemStyle = stAccentGroup
		}
		runs = append(runs, paneRun{" " + action.glyph + " ", itemStyle, "float:" + f.id + ":" + action.name, true})
		x += 3
	}
	if len(actions) > 0 {
		runs = append(runs, paneRun{edgeRoundR, frame, "", false})
		x++
	}
	tail := width - x
	if tail > 1 {
		runs = append(runs, paneRun{strings.Repeat("\u2500", tail-1), frame, "", false})
	}
	runs = append(runs, paneRun{"\u2510", frame, "", false})
	return runs
}

func (m *model) floatingNodes(out *[]*sdk.Builder, f *floating) {
	active := m.activeFloat == f.id
	dimmed := m.activeFloat != "" && !active
	focused := active && m.mode == modeLive && !f.collapsed
	frame := stPanelBorder
	if active {
		frame = stAccent
	}
	if dimmed {
		frame = dimStyle(frame)
	}
	rx := f.x
	for _, run := range m.floatingRuns(f, active, f.w) {
		width := sdk.DisplayWidth(run.text)
		if rx >= m.cols {
			break
		}
		text := run.text
		if rx+width > m.cols {
			text = sdk.Truncate(text, m.cols-rx)
			width = sdk.DisplayWidth(text)
		}
		style := run.style
		if dimmed {
			style = dimStyle(style)
		}
		addRun(out, rx, f.y, text, style, run.node, run.mouse, width)
		rx += width
	}
	if f.collapsed {
		return
	}
	for row := 1; row < f.h-1; row++ {
		addRun(out, f.x, f.y+row, "\u2502", frame, "", false, 1)
		addRun(out, f.x+f.w-1, f.y+row, "\u2502", frame, "", false, 1)
	}
	addRun(out, f.x, f.y+f.h-1, "\u2514"+strings.Repeat("\u2500", maxInt(0, f.w-2))+"\u2518", frame, "", false, 0)
	innerW, innerH := maxInt(0, f.w-2), maxInt(0, f.h-2)
	src := m.paneSource(f.pane)
	if src != nil && !m.demo && m.host {
		props := map[string]string{"chrome.inset": "0"}
		if dimmed {
			props["chrome.dim"] = "1"
		}
		box := sdk.Terminal(f.pane.sourceID).ID(f.pane.id).Pos(f.x+1, f.y+1).Width(innerW).Height(innerH).
			Props(props).
			Input("key", "paste", "wheel").
			Focused(focused)
		*out = append(*out, box)
		return
	}
	lines := f.pane.lines
	if len(lines) == 0 {
		lines = []string{""}
	}
	for index := 0; index < innerH; index++ {
		text := ""
		if index < len(lines) {
			text = lines[index]
		}
		addRun(out, f.x+1, f.y+1+index, sdk.Truncate(text, innerW), stOverlay, f.pane.id, true, innerW)
	}
}

// --------------------------------------------------------------- overlays

// overlayRun is one styled piece of an overlay row; overlayRow is a list of
// runs so picker rows can carry their per-column colors (glyph/state/view).
type overlayRun struct {
	text  string
	style string
	node  string
}

// run is the common two-field overlay run (no per-run hit node).
func run(text, style string) overlayRun { return overlayRun{text: text, style: style} }

type overlayRow struct {
	runs       []overlayRun
	selectable bool
	cursor     bool
	cursorCol  int
	node       string
	tabs       bool // the picker endpoint tab line
}

func textRow(text, style string) overlayRow {
	return overlayRow{runs: []overlayRun{{text: text, style: style}}}
}

func (m *model) centerRect(width, height int) (int, int, int, int) {
	width = minInt(width, m.cols)
	height = minInt(height, m.rows)
	x := maxInt(0, (m.cols-width)/2)
	y := maxInt(0, (m.rows-height)/2)
	return width, height, x, y
}

// ------------------------------------------------------ picker rendering

const (
	pickerPrefixWidth   = 4
	pickerStateColWidth = 8
	pickerViewColWidth  = 4
	pickerSizeColWidth  = 7
	pickerLastColWidth  = 5
	pickerColGapWidth   = 2
	// pickerMaxContentRows bounds the picker body, matching main's overlay
	// height cap of 24 (minus the two border rows).
	pickerMaxContentRows = 22
)

// pickerWindow keeps the selection visible inside a bounded row count,
// centering it like main's terminalPickerRowWindow.
func pickerWindow(selected, itemCount, visible int) (int, int) {
	if itemCount <= 0 || visible <= 0 {
		return 0, 0
	}
	visible = minInt(visible, itemCount)
	start := clampInt(selected-visible/2, 0, itemCount-visible)
	return start, start + visible
}

func pickerTabLabel(tab pickerTab) string {
	return " " + sdk.Truncate(tab.label, 22) + " " + strconv.Itoa(tab.count)
}

func pickerTabWidth(tab pickerTab) int {
	glyph, _ := endpointGlyph(tab.status)
	return 2 + sdk.DisplayWidth(glyph) + sdk.DisplayWidth(pickerTabLabel(tab))
}

// pickerTabRuns is the endpoint tab line: `▸ ● local 2` per machine with the
// status glyph from the endpoint health (recommended yaml endpoint_status).
func (m *model) pickerTabRuns(innerW int) []overlayRun {
	runs, _ := m.pickerTabRunsWithUnderline(innerW)
	return runs
}

// pickerTabUnderline mirrors main's `tabStyle: underline`: a ━ rule under the
// active endpoint tab, starting at the label (not the marker) for the label's
// width.
func (m *model) pickerTabUnderline(innerW int) string {
	_, underline := m.pickerTabRunsWithUnderline(innerW)
	return underline
}

func (m *model) pickerTabRunsWithUnderline(innerW int) ([]overlayRun, string) {
	tabs := m.pickerTabs()
	if len(tabs) == 0 {
		return nil, ""
	}
	selected := clampInt(m.pickerTab, 0, len(tabs)-1)
	start, end := selected, selected+1
	used := pickerTabWidth(tabs[selected])
	for {
		progressed := false
		if end < len(tabs) && used+3+pickerTabWidth(tabs[end])+2 <= innerW {
			used += 3 + pickerTabWidth(tabs[end])
			end++
			progressed = true
		}
		if start > 0 && used+3+pickerTabWidth(tabs[start-1])+2 <= innerW {
			used += 3 + pickerTabWidth(tabs[start-1])
			start--
			progressed = true
		}
		if !progressed {
			break
		}
	}
	var runs []overlayRun
	underline := ""
	x := 0
	if start > 0 {
		runs = append(runs, run("\u2039 ", stMuted))
		x += 2
	}
	for index := start; index < end; index++ {
		if index > start {
			runs = append(runs, run("   ", stMuted))
			x += 3
		}
		marker := "  "
		bodyStyle := stMuted
		if index == selected {
			marker = "\u25b8 "
			bodyStyle = stAccent
		}
		glyph, glyphStyle := endpointGlyph(tabs[index].status)
		tabNode := "picker:tab:" + strconv.Itoa(index)
		if index == selected {
			labelWidth := sdk.DisplayWidth(marker) + sdk.DisplayWidth(glyph) + sdk.DisplayWidth(pickerTabLabel(tabs[index]))
			underline = strings.Repeat(" ", x+2) + strings.Repeat("\u2501", maxInt(0, labelWidth-2))
		}
		runs = append(runs,
			overlayRun{marker, bodyStyle, tabNode},
			overlayRun{glyph, glyphStyle, tabNode},
			overlayRun{pickerTabLabel(tabs[index]), bodyStyle, tabNode},
		)
		x += pickerTabWidth(tabs[index])
	}
	if end < len(tabs) {
		runs = append(runs, run(" \u203a", stMuted))
	}
	return runs, underline
}

func pickerStateLabel(src *pb.Source) (string, string) {
	if src.GetExited() {
		return "exited", stWarning
	}
	if src.GetAttached() {
		return "attached", stSuccess
	}
	return "running", stSuccess
}

// pickerSourceRuns renders one terminal row in the old column layout:
// marker + state glyph + title · state · xN · size · activity.
func (m *model) pickerSourceRuns(src *pb.Source, innerW int, selected bool) []overlayRun {
	marker := "  "
	markerStyle := stMuted
	titleStyle := stContent
	if selected {
		marker = "\u25b8 "
		markerStyle = stAccent
		titleStyle = stAccent
	}
	title := strings.TrimSpace(src.GetTitle())
	if title == "" {
		title = strings.TrimSpace(src.GetTerminalId())
	}
	if title == "" {
		title = shortSourceID(src.GetId())
	}
	// main appends the public tag labels to the title.
	if labels := publicTagLabels(src); len(labels) > 0 {
		title += " · " + strings.Join(labels, " · ")
	}
	stateText, stateStyle := pickerStateLabel(src)
	viewCount := 0
	if src.GetAttached() {
		viewCount = 1
	}
	viewText := "x" + strconv.Itoa(viewCount)
	viewStyle := stMuted
	if viewCount > 0 {
		viewStyle = stFooterInfo
	}
	size := "-"
	if src.GetCols() > 0 && src.GetRows() > 0 {
		size = strconv.Itoa(int(src.GetCols())) + "x" + strconv.Itoa(int(src.GetRows()))
	}
	activity := "-"
	if src.GetLastOutputMs() > 0 {
		activity = pickerActivityLabel(time.UnixMilli(src.GetLastOutputMs()))
	}
	fixed := pickerPrefixWidth + pickerColGapWidth + pickerStateColWidth +
		pickerColGapWidth + pickerViewColWidth +
		pickerColGapWidth + pickerSizeColWidth +
		pickerColGapWidth + pickerLastColWidth
	titleWidth := innerW - fixed
	if titleWidth < 4 {
		return []overlayRun{
			run(marker, markerStyle),
			run(glyphRunning, stateStyle),
			run(" "+sdk.Truncate(title, maxInt(1, innerW-4)), titleStyle),
		}
	}
	runs := []overlayRun{
		run(marker, markerStyle),
		run(glyphRunning, stateStyle),
		run(" ", titleStyle),
	}
	runs = append(runs, pickerHighlightedRuns(title, titleWidth, m.pickerQuery, titleStyle)...)
	runs = append(runs, run(strings.Repeat(" ", pickerColGapWidth), stContent))
	runs = append(runs, pickerHighlightedRuns(stateText, pickerStateColWidth, m.pickerQuery, stateStyle)...)
	runs = append(runs, run(strings.Repeat(" ", pickerColGapWidth), stContent))
	runs = append(runs, pickerHighlightedRuns(viewText, pickerViewColWidth, m.pickerQuery, viewStyle)...)
	runs = append(runs, run(strings.Repeat(" ", pickerColGapWidth), stContent))
	runs = append(runs, pickerHighlightedRuns(size, pickerSizeColWidth, m.pickerQuery, titleStyle)...)
	runs = append(runs, run(strings.Repeat(" ", pickerColGapWidth), stContent))
	return append(runs, pickerHighlightedRuns(activity, pickerLastColWidth, m.pickerQuery, stMuted)...)
}

// pickerHighlightedRuns splits one picker cell value into runs, emphasizing
// the characters that matched the query (including pinyin matches on CJK
// characters). Width padding keeps the column layout stable.
func pickerHighlightedRuns(value string, width int, query, baseStyle string) []overlayRun {
	display := sdk.Truncate(value, width)
	indexes := matchSearchValue(value, query)
	if len(indexes) == 0 {
		return []overlayRun{{text: padRightStr(display, width), style: baseStyle}}
	}
	match := map[int]bool{}
	for _, index := range indexes {
		match[index] = true
	}
	var runs []overlayRun
	var current strings.Builder
	currentMatch := false
	flush := func() {
		if current.Len() == 0 {
			return
		}
		style := baseStyle
		if currentMatch {
			style = stPickerMatch
		}
		runs = append(runs, overlayRun{text: current.String(), style: style})
		current.Reset()
	}
	for index, r := range []rune(display) {
		matched := match[index]
		if matched != currentMatch {
			flush()
			currentMatch = matched
		}
		current.WriteRune(r)
	}
	flush()
	if pad := width - sdk.DisplayWidth(display); pad > 0 {
		runs = append(runs, overlayRun{text: strings.Repeat(" ", pad), style: baseStyle})
	}
	return runs
}

// pickerActivityLabel mirrors main's TerminalOutputActivityLabel: now / Ns /
// Nm / Nh since the last non-empty output.
func pickerActivityLabel(at time.Time) string {
	quiet := time.Since(at)
	if quiet < 0 {
		quiet = 0
	}
	switch {
	case quiet < 5*time.Second:
		return "now"
	case quiet < time.Minute:
		return strconv.Itoa(int(quiet.Seconds())) + "s"
	case quiet < time.Hour:
		return strconv.Itoa(int(quiet.Minutes())) + "m"
	default:
		return strconv.Itoa(int(quiet.Hours())) + "h"
	}
}

func (m *model) pickerToolbarRuns(innerW int) []overlayRun {
	endpoint := m.pickerTabName()
	running, exited := 0, 0
	for _, src := range m.terminals() {
		if endpointOf(src) != endpoint {
			continue
		}
		if src.GetExited() {
			exited++
		} else {
			running++
		}
	}
	all := running + exited
	// The status group is `● Current N   Other M   All K`, matching main's
	// terminalPickerToolbarLine: the selected filter carries the filled marker
	// and accent style, the rest stay muted.
	statusRun := func(label string, count int, value int) overlayRun {
		marker := "  "
		style := stMuted
		if m.pickerFilter == value {
			marker = glyphRunning + " "
			style = stAccent
		}
		return overlayRun{text: marker + label + " " + strconv.Itoa(count), style: style, node: "picker-status:" + strconv.Itoa(value)}
	}
	tagLabel := "Tags"
	if len(m.pickerTags) > 0 {
		tagLabel = "Tags " + m.pickerTags[0]
		if len(m.pickerTags) > 1 {
			tagLabel += " +" + strconv.Itoa(len(m.pickerTags)-1)
		}
	}
	// Search stays on the left; the status group and Tags stay on the right,
	// matching main's terminalPickerToolbarLine layout (search │ status │ tags).
	search := []overlayRun{{text: "⌕ /" + m.pickerQuery, style: stAccent}}
	right := []overlayRun{
		{text: " │ ", style: stMuted},
		statusRun("Running", running, pickerFilterRunning),
		{text: "  ", style: stMuted},
		statusRun("Exited", exited, pickerFilterExited),
		{text: "  ", style: stMuted},
		statusRun("All", all, pickerFilterAll),
		{text: "   ", style: stMuted},
		{text: tagLabel, style: stFooterInfo, node: "picker-tags"},
	}
	rightWidth := 0
	for _, run := range right {
		rightWidth += sdk.DisplayWidth(run.text)
	}
	searchWidth := 0
	for _, run := range search {
		searchWidth += sdk.DisplayWidth(run.text)
	}
	if searchWidth+rightWidth > innerW {
		return []overlayRun{{text: sdk.Truncate("⌕ /"+m.pickerQuery+"  Shift←/→ status", innerW), style: stAccent}}
	}
	gap := innerW - searchWidth - rightWidth
	return append(append(append([]overlayRun{}, search...), overlayRun{text: strings.Repeat(" ", gap), style: stMuted}), right...)
}

func (m *model) pickerEndpointLabel(endpoint string) string {
	for _, tab := range m.pickerTabs() {
		if tab.name == endpoint {
			return tab.label
		}
	}
	return endpoint
}

// pickerTagRows is the Ctrl-T tag checkbox sub-view (main
// terminal_picker_tags): marker + ✓/○ + label + count, with its own search.
func (m *model) pickerTagRows(innerW int) []overlayRow {
	rows := []overlayRow{{runs: []overlayRun{{text: "⌕ /" + m.pickerTagQuery, style: stAccent}}}}
	options := m.pickerTagOptions()
	if len(options) == 0 {
		return append(rows, overlayRow{runs: []overlayRun{{text: "No tags", style: stMuted}}})
	}
	for index, option := range options {
		marker := "  "
		markerStyle := stMuted
		if index == m.pickerTagSel {
			marker = "\u25b8 "
			markerStyle = stAccent
		}
		check := "\u25cb"
		checkStyle := stMuted
		if option.checked {
			check = "\u2713"
			checkStyle = stSuccess
		}
		label := padRightStr(option.label, maxInt(1, innerW-8))
		rows = append(rows, overlayRow{
			runs: []overlayRun{
				run(marker, markerStyle),
				run(check, checkStyle),
				run(" ", stContent),
				run(label, stContent),
				run(strconv.Itoa(option.count), stMuted),
			},
			selectable: true,
			node:       "picker-tag:" + strconv.Itoa(index),
		})
	}
	return rows
}

func padRightStr(text string, cells int) string {
	width := sdk.DisplayWidth(text)
	if width >= cells {
		return sdk.Truncate(text, cells)
	}
	return text + strings.Repeat(" ", cells-width)
}

func (m *model) overlayRows() []overlayRow {
	switch m.overlay {
	case overlayPicker:
		innerW := minInt(80, maxInt(8, m.cols-2)) - 2
		if m.pickerTagsOpen {
			return m.pickerTagRows(innerW)
		}
		rows := []overlayRow{
			{runs: m.pickerTabRuns(innerW), tabs: true},
		}
		if underline := m.pickerTabUnderline(innerW); underline != "" {
			rows = append(rows, overlayRow{runs: []overlayRun{{text: underline, style: stAccent}}})
		}
		rows = append(rows, overlayRow{runs: m.pickerToolbarRuns(innerW)})
		fixed := len(rows)
		pickerRows := m.pickerRows()
		start, end := pickerWindow(m.picker, len(pickerRows), maxInt(1, pickerMaxContentRows-fixed))
		for index := start; index < end; index++ {
			row := pickerRows[index]
			selected := index == m.picker
			if row.source != nil {
				rows = append(rows, overlayRow{
					runs:       m.pickerSourceRuns(row.source, innerW, selected),
					selectable: true,
					node:       "picker:" + strconv.Itoa(index),
				})
				continue
			}
			marker := "  "
			markerStyle := stMuted
			if selected {
				marker = "\u25b8 "
				markerStyle = stAccent
			}
			rows = append(rows, overlayRow{
				runs: []overlayRun{
					run(marker, markerStyle),
					run("+", stFooterInfo),
					run(" New terminal", stContent),
				},
				selectable: true,
				node:       "picker:" + strconv.Itoa(index),
			})
		}
		return rows
	case overlayPrompt:
		if m.promptKind == "terminal.create" {
			labels := []string{"name*", "command", "server*", "workdir", "tags"}
			rows := []overlayRow{{runs: []overlayRun{{text: "Create Terminal", style: stAccent}}}}
			for i, label := range labels {
				value := ""
				if i < len(m.promptFields) {
					value = m.promptFields[i]
				}
				displayValue := value
				if displayValue == "" {
					displayValue = m.createPromptPlaceholder(i)
				}
				style := stContent
				if i == m.promptField {
					style = stAccent
				}
				valueRunes := []rune(value)
				cursor := 0
				if i < len(m.promptCursors) {
					cursor = clampInt(m.promptCursors[i], 0, len(valueRunes))
				}
				rows = append(rows, overlayRow{
					runs:      []overlayRun{{text: label + ": " + displayValue, style: style, node: "prompt-field:" + strconv.Itoa(i)}},
					cursor:    i == m.promptField,
					cursorCol: sdk.DisplayWidth(label+": ") + sdk.DisplayWidth(string(valueRunes[:cursor])),
				})
			}
			if m.promptError != "" {
				rows = append(rows, overlayRow{runs: []overlayRun{{text: "error: " + m.promptError, style: stWarning}}})
			}
			rows = append(rows, overlayRow{runs: []overlayRun{{text: "Enter create · Tab field · Esc cancel", style: stMuted}}})
			return rows
		}
		matches := m.promptMatches()
		promptRunes := []rune(m.prompt)
		cursor := clampInt(m.promptCursor, 0, len(promptRunes))
		rows := []overlayRow{{runs: []overlayRun{run(": "+m.prompt, stOverlay)}, cursor: true, cursorCol: sdk.DisplayWidth(": " + string(promptRunes[:cursor]))}}
		for _, command := range matches {
			rows = append(rows, overlayRow{runs: []overlayRun{{text: command, style: stOverlay}}, selectable: true})
		}
		return rows
	case overlayHelp:
		var rows []overlayRow
		for _, line := range helpLines {
			rows = append(rows, textRow(line, stOverlay))
		}
		return rows
	case overlayClipboard:
		var rows []overlayRow
		for i, item := range m.clipboard {
			rows = append(rows, overlayRow{
				runs:       []overlayRun{{text: "[" + strconv.Itoa(i) + "] " + item, style: stOverlay}},
				selectable: true,
			})
		}
		return rows
	case overlayConnections:
		// Legacy system.open_connections table: one row per registered
		// endpoint showing its label/name, kind and health.
		var rows []overlayRow
		if len(m.connections) == 0 {
			return []overlayRow{textRow("no registered endpoints", stMuted)}
		}
		for i, conn := range m.connections {
			marker := "  "
			markerStyle := stMuted
			if i == m.connSel {
				marker = "\u25b8 "
				markerStyle = stAccent
			}
			label := conn.label
			if label == "" {
				label = conn.name
			}
			healthStyle := stMuted
			switch conn.health {
			case "ok":
				healthStyle = stSuccess
			case "unknown", "":
				healthStyle = stMuted
			default:
				healthStyle = stWarning
			}
			rows = append(rows, overlayRow{
				runs: []overlayRun{
					run(marker, markerStyle),
					run(glyphRunning, healthStyle),
					run(" "+label, stOverlay),
					run("  ("+conn.kind+" \u00b7 "+conn.health+")", stMuted),
				},
				selectable: true,
				node:       "connection:" + strconv.Itoa(i),
			})
		}
		return rows
	}
	return nil
}

func (m *model) createPromptPlaceholder(field int) string {
	switch field {
	case 1:
		command := createCommandDisplay(m.promptDefaults.command)
		if command == "" {
			return ""
		}
		return "[" + command + "]"
	case 2:
		return "[endpoint id or label]"
	case 3:
		return "[endpoint cwd]"
	case 4:
		if m.promptPublicTagList {
			return "[production, backend]"
		}
		return "[kind=task, agent=codex]"
	default:
		return ""
	}
}

func promptSuggestionItemName(item string) string {
	trimmed := strings.TrimRight(item, "/\\")
	if trimmed == "" {
		return item
	}
	if slash := strings.LastIndexAny(trimmed, "/\\"); slash >= 0 {
		trimmed = trimmed[slash+1:]
	}
	return trimmed + "/"
}

func (m *model) promptSuggestionNodes(out *[]*sdk.Builder) {
	if m.overlay != overlayPrompt || m.promptKind != "terminal.create" || len(m.promptSuggestions) == 0 && m.promptSuggestionEmpty == "" {
		return
	}
	if m.promptSuggestionField < 0 || m.promptSuggestionField >= len(m.promptFields) {
		return
	}
	labels := []string{"name*", "command", "server*", "workdir", "tags"}
	field := m.promptSuggestionField
	if field >= len(labels) {
		return
	}
	// Keep the popup outside the form rows. This matches main: the modal keeps
	// its fixed field layout while the completion popup floats beside the field.
	rows := m.overlayRows()
	width, minHeight := 64, 10
	height := minInt(m.rows, maxInt(minHeight, len(rows)+4))
	width, _, x, y := m.centerRect(width, height)
	anchorX := x + 1 + sdk.DisplayWidth(labels[field]+": ")
	anchorY := y + 3 + field

	lines := make([]string, 0, promptSuggestionVisibleRows+2)
	if title := strings.TrimSpace(m.promptSuggestionTitle); title != "" {
		lines = append(lines, "  "+title)
	}
	if len(m.promptSuggestions) == 0 {
		if m.promptSuggestionEmpty != "" {
			lines = append(lines, "  "+m.promptSuggestionEmpty)
		}
	} else {
		selected := clampInt(m.promptSuggestionSel, 0, len(m.promptSuggestions)-1)
		offset := promptSuggestionOffsetForSelection(m.promptSuggestionOffset, selected, len(m.promptSuggestions))
		end := minInt(len(m.promptSuggestions), offset+promptSuggestionVisibleRows)
		for index := offset; index < end; index++ {
			marker := "  "
			if m.promptSuggestionFocused && index == selected {
				marker = "▸ "
			}
			lines = append(lines, marker+"  "+promptSuggestionItemName(m.promptSuggestions[index]))
		}
		if offset > 0 || end < len(m.promptSuggestions) {
			lines = append(lines, fmt.Sprintf("  %d-%d/%d", offset+1, end, len(m.promptSuggestions)))
		}
	}
	if len(lines) == 0 {
		return
	}
	popupW := 0
	for _, line := range lines {
		popupW = maxInt(popupW, sdk.DisplayWidth(line))
	}
	popupW = minInt(maxInt(1, popupW), maxInt(1, m.cols))
	if anchorX+popupW > m.cols {
		anchorX = maxInt(0, m.cols-popupW)
	}
	popupY := anchorY
	bottom := m.rows
	if m.footerVisible {
		bottom--
	}
	if popupY+len(lines) > bottom {
		popupY = maxInt(0, anchorY-len(lines)-1)
	}
	for index, line := range lines {
		if popupY+index < 0 || popupY+index >= m.rows {
			continue
		}
		addRun(out, anchorX, popupY+index, strings.Repeat(" ", popupW), stContent, "", false, popupW)
		style := stMuted
		if index >= 0 && strings.HasPrefix(line, "▸ ") {
			style = stAccent
		}
		node := ""
		if m.promptSuggestionField >= 0 && len(m.promptSuggestions) > 0 {
			itemStart := 0
			if strings.TrimSpace(m.promptSuggestionTitle) != "" {
				itemStart = 1
			}
			if index >= itemStart && index < itemStart+minInt(promptSuggestionVisibleRows, len(m.promptSuggestions)) {
				selectedIndex := m.promptSuggestionOffset + index - itemStart
				if selectedIndex >= 0 && selectedIndex < len(m.promptSuggestions) {
					node = "prompt-suggestion:" + strconv.Itoa(selectedIndex)
				}
			}
		}
		addRun(out, anchorX, popupY+index, sdk.Truncate(line, popupW), style, node, node != "", popupW)
	}
}

func (m *model) overlayNodes(out *[]*sdk.Builder) {
	var title string
	var width, minHeight int
	switch m.overlay {
	case overlayPicker:
		title, width, minHeight = "Terminal Picker", 80, 10
	case overlayPrompt:
		title, width, minHeight = "Command", 56, 6
		if m.promptKind == "terminal.create" {
			title, width, minHeight = "Create Terminal", 64, 10
		}
	case overlayHelp:
		title, width, minHeight = "Help", 62, 8
	case overlayClipboard:
		title, width, minHeight = "Clipboard", 60, 8
	case overlayConnections:
		title, width, minHeight = "Connections", 64, 8
	default:
		return
	}
	rows := m.overlayRows()
	height := minInt(m.rows, maxInt(minHeight, len(rows)+4))
	if m.overlay == overlayPicker {
		// main caps the terminal picker overlay at 24 rows.
		height = minInt(height, 24)
	}
	width, height, x, y := m.centerRect(width, height)
	frame := stAccent
	if m.overlay == overlayPicker {
		frame = stPanelBorder
	}
	innerW := width - 2
	// main fills the whole overlay rectangle with a background-less style
	// before drawing its chrome: the page behind is erased, yet the overlay
	// keeps the terminal's default background instead of painting its own.
	for row := 0; row < height; row++ {
		addRun(out, x, y+row, strings.Repeat(" ", width), stContent, "", false, width)
	}
	titleText := " " + title + " "
	top := "\u250c\u2500" + titleText + strings.Repeat("\u2500", maxInt(0, width-3-sdk.DisplayWidth(titleText))) + "\u2510"
	addRun(out, x, y, top, frame, "", false, 0)
	for row := 1; row < height-1; row++ {
		addRun(out, x, y+row, "\u2502", frame, "", false, 1)
		addRun(out, x+width-1, y+row, "\u2502", frame, "", false, 1)
	}
	addRun(out, x, y+height-1, "\u2514"+strings.Repeat("\u2500", maxInt(0, width-2))+"\u2518", frame, "", false, 0)
	selectable := 0
	for index, row := range rows {
		if index >= height-2 {
			break
		}
		selected := false
		if row.selectable {
			switch m.overlay {
			case overlayPicker:
				selected = selectable == m.picker
			case overlayPrompt:
				selected = selectable == m.promptSel
			case overlayClipboard:
				selected = selectable == m.clipSel
			case overlayConnections:
				selected = selectable == m.connSel
			}
			selectable++
		}
		node := row.node
		if node == "" && row.tabs {
			node = ""
		}
		if node == "" && row.selectable {
			switch m.overlay {
			case overlayPrompt:
				node = "prompt:" + strconv.Itoa(selectable-1)
			}
		}
		runs := row.runs
		if selected {
			runs = overlaySelectedRuns(runs)
		}
		rx := x + 1
		runStart := 0
		for _, run := range runs {
			width := sdk.DisplayWidth(run.text)
			if width <= 0 || rx >= x+1+innerW {
				continue
			}
			text := run.text
			if rx+width > x+1+innerW {
				text = sdk.Truncate(text, x+1+innerW-rx)
				width = sdk.DisplayWidth(text)
			}
			boxWidth := width
			runNode := run.node
			if runNode == "" {
				runNode = node
			}
			cursorInRun := row.cursor && row.cursorCol >= runStart && row.cursorCol <= runStart+width
			cursorLocal := 0
			if cursorInRun {
				col := row.cursorCol
				if col < runStart {
					col = runStart
				}
				if col > runStart+width {
					col = runStart + width
				}
				cursorLocal = col - runStart
				// The kernel clips a cursor at the last cell of its box. Keep one
				// trailing cell available so an end-of-input caret sits after the
				// final rune instead of on top of it.
				if cursorLocal == width && rx+boxWidth < x+1+innerW {
					boxWidth++
				}
				m.cursor = &cursorPos{x: rx + cursorLocal, y: y + 1 + index}
			}
			box := sdk.Text(text).Pos(rx, y+1+index).Width(boxWidth).Height(1).Style(run.style)
			if cursorInRun {
				box.Cursor(0, cursorLocal, "bar")
			}
			if runNode != "" {
				box.ID(runNode).Input("mouse")
			}
			*out = append(*out, box)
			rx += boxWidth
			runStart += width
		}
	}
}

// overlaySelectedRuns recolors the prompt rows (which carry no per-column
// colors) with the accent selection style; picker rows already render their
// own selected marker.
func overlaySelectedRuns(runs []overlayRun) []overlayRun {
	out := make([]overlayRun, len(runs))
	for i, run := range runs {
		if run.style == stOverlay {
			run.style = stAccent
		}
		out[i] = run
	}
	return out
}

func (m *model) toastNodes(out *[]*sdk.Builder) {
	if m.toast == "" {
		return
	}
	text := " " + m.toast + "  \u00b7  Ctrl-Q quit "
	width := minInt(m.cols, sdk.DisplayWidth(text))
	x := maxInt(1, m.cols-width-1)
	// Toasts are global notices; keep them in the unused top-right header
	// space instead of covering the lower-right panel content.
	y := 0
	addRun(out, x, y, sdk.Truncate(text, width), stToast, "toast", true, width)
}

func (m *model) footerNodes(out *[]*sdk.Builder) {
	y := m.rows - 1
	runs, right := m.footerRuns()
	runs = trimRuns(runs, m.cols)
	leftWidth := 0
	for _, run := range runs {
		leftWidth += sdk.DisplayWidth(run.text)
	}
	right = trimRuns(right, m.cols-leftWidth)
	rightWidth := 0
	for _, run := range right {
		rightWidth += sdk.DisplayWidth(run.text)
	}
	pad := maxInt(0, m.cols-leftWidth-rightWidth)
	x := 0
	for _, run := range runs {
		width := sdk.DisplayWidth(run.text)
		if width <= 0 {
			continue
		}
		addRun(out, x, y, run.text, run.style, run.node, run.node != "", width)
		x += width
	}
	if pad > 0 {
		addRun(out, x, y, "", stFooterFill, "", false, pad)
	}
	x += pad
	for _, run := range right {
		width := sdk.DisplayWidth(run.text)
		if width <= 0 {
			continue
		}
		addRun(out, x, y, run.text, run.style, "", false, width)
		x += width
	}
}

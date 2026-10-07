package main

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/app"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// Search modes cycle with tab, exactly the old text -> glob -> regex order.
const (
	copySearchText = iota
	copySearchGlob
	copySearchRegex
)

const copySearchDebounce = 120 * time.Millisecond

// Copy-drag edge auto-scroll timing, ports of the legacy
// historyMouseScrollInterval and reduceCopyModeScroll*Rows(…, 2): while the
// pointer is held at a content edge the marked selection advances two rows per
// 100ms tick.
const (
	copyMouseScrollInterval = 100 * time.Millisecond
	copyMouseScrollRows     = 2
)

// Copy scene (the old tui/app copymode state machine, per pane).
//
// The host owns the terminal text; the program owns the interaction: a copy
// session per pane holds the scroll offset, the selection anchor/focus in
// viewport cells, the search query and its matches. The host paints the
// session state through program props (copy.cursor/copy.sel/copy.match) and
// resolves the final selection through terminal.copy{sel} (linear cell
// indices over the window the view shows). Scrolling back to the bottom with
// no mark closes the session, exactly the old AtFrozenBottom auto-exit.

const (
	// Recommended coralline-candy copy styles (the old theme mapped the
	// selection to ANSI palette 8/3, search matches to warning).
	copySelStyle       = "fg:ansi:8;bg:ansi:3"
	copyMatchStyle     = "fg:#fde68a;dim;underline"
	copyMatchCurStyle  = "fg:#070611;bg:#fde68a;bold"
	copyCursorStyle    = "reverse"
	copySearchBarStyle = "fg:#070611;bg:#fde68a"
	// copySearchCaretStyle is the reverse-video blank cell that shows the edit
	// caret in the footer search bar (legacy CursorShapeBar).
	copySearchCaretStyle = "reverse"
)

// copyMatch is one search match on one viewport row (inclusive display cols).
type copyMatch struct {
	row      int
	startCol int
	endCol   int
}

// copyState is one pane's copy session.
type copyState struct {
	offset      int // scroll offset the host reported (0 = live)
	cursorRow   int // viewport row, 0 = top of the content area
	cursorCol   int // display column
	markRow     int
	markCol     int
	marked      bool
	searching   bool
	searchMode  int
	searchCol   int // rune cursor inside the query
	query       string
	forward     bool
	searchSeq   uint64      // scan generation: stale window responses are dropped
	searchDirty bool        // query changed; a debounced scan is pending
	searchErr   string      // last search error, shown in the search bar
	matches     []copyMatch // visible matches (viewport rows)
	matchIdx    int
	// searchCurrent / searchTotal are the scan status shown in the footer bar
	// (the legacy "N/M" / "no match" badge) after a terminal.search response.
	searchCurrent int
	searchTotal   int
	searchPending bool
	currentMatch  []copyMatch // authoritative match, including soft-wrapped rows
	rows          []string    // the window the host last showed
	cols          int         // content width (PTY cols) used for linear indices
	viewRows      int         // content height the layout last showed
	// scrollSeq identifies the latest in-flight terminal.scroll request for
	// this session. A persistent history provider is asynchronous; a late
	// response from an older direction must never move the viewport back and
	// make repeated bottom scrolling oscillate between two offsets.
	scrollSeq uint64
	// pendingRows accumulates vertical cursor moves requested before the
	// first window arrived (wheel enters copy while the window is in flight).
	pendingRows int
	placed      bool // the cursor was placed on the first loaded window
}

// copyFor returns the pane's session, if any.
func (m *model) copyFor(p *pane) *copyState {
	if p == nil {
		return nil
	}
	return m.copyPanes[p.id]
}

// copyOverflow reports which directions a frozen copy window is clipped, so
// view.go can draw the legacy overflow markers on the pane border. It ports the
// truth half of render.contentViewportOverflow for the copy-history content
// kind (render/copy_history.go + content_viewport.go):
//
//   - top: the window is scrolled away from the oldest row (older content
//     exists above the window); offset is the distance from the live bottom,
//     so any non-zero offset means older rows remain above.
//   - bottom: the window is not at the live bottom (offset > 0); newer content
//     exists below the frozen window.
//   - right: a loaded window row is wider than the pane content area, i.e. the
//     row is horizontally clipped. (left stays false: the offset window is the
//     newest page ending at the live bottom, so it can only clip at the newer
//     right edge, and the exact per-row start column is not derivable here.)
//
// Live terminal panes have no session here, so this never invents scrollback
// state for them.
func (m *model) copyOverflow(st *copyState, contentWidth int) (left, right, top, bottom bool) {
	if st == nil {
		return false, false, false, false
	}
	top = st.offset > 0
	bottom = st.offset > 0
	if contentWidth > 0 {
		for _, row := range st.rows {
			if sdk.DisplayWidth(row) > contentWidth {
				right = true
				break
			}
		}
	}
	return left, right, top, bottom
}

// copyActive reports whether the focused pane has an open copy session (the
// active view owns copy input, exactly the old ActiveViewOwnsCopyInput).
func (m *model) copyActive() bool {
	return m.copyFor(m.focusContentPane()) != nil
}

// enterCopy opens a copy session at the live bottom and loads its window.
func (m *model) enterCopy() app.Cmd {
	p := m.focusContentPane()
	if p == nil || m.copyFor(p) != nil {
		return nil
	}
	src := m.paneSource(p)
	if src == nil || src.GetTerminalId() == "" {
		m.toast = "nothing to copy"
		return nil
	}
	st := &copyState{offset: 0}
	m.copyPanes[p.id] = st
	m.prepareCopyViewport(p, st)
	m.toast = "copy: h/j/k/l move · space mark · y copy · / search"
	// Fetch the visible window so cursor clamping and selection start from
	// the real text.
	return m.fetchCopyWindow(p, st)
}

// endCopy closes one pane's copy session and returns that terminal to live.
func (m *model) endCopy(p *pane) app.Cmd {
	if p == nil {
		return nil
	}
	st := m.copyFor(p)
	delete(m.copyPanes, p.id)
	if st == nil {
		return nil
	}
	if m.copyDragPane == p.id {
		m.clearCopyDragScroll()
	}
	src := m.paneSource(p)
	if src == nil || src.GetTerminalId() == "" {
		return nil
	}
	// view is the pane id: the terminal box node id is the pane id (view.go
	// sets sdk.Terminal(...).ID(p.id)), so each pane gets its own frozen viewport.
	return m.emit("terminal.scrollEnd", &pb.MethodParams{
		Endpoint: endpointOf(src), Id: src.GetTerminalId(), View: p.id,
	}, opMsg{op: "scrollEnd"})
}

// handleCopyKey is the copy scene input (the old copy shortcut scene). Note
// there is no esc binding here: the legacy copy scene exits with G (newest) or
// by pressing the copy entry chord again, never esc.
func (m *model) handleCopyKey(key, char string) app.Cmd {
	p := m.focusContentPane()
	st := m.copyFor(p)
	if p == nil || st == nil {
		return nil
	}
	if st.searching {
		return m.handleCopySearchKey(p, st, key, char)
	}
	switch key {
	case "ctrl-p":
		return m.endCopy(p)
	case "ctrl-shift-c":
		// Re-entering copy (the old copy.enter while active) refreshes to the
		// newest position and clears the selection/search.
		return m.resetCopyToLatest(p, st)
	case "G":
		return m.endCopy(p)
	case "enter":
		if st.marked {
			return m.copySelection(p, st, true)
		}
		m.toast = "nothing to copy: select text before pressing Enter"
	case "y":
		if st.marked {
			return m.copySelection(p, st, false)
		}
		m.toast = "nothing to copy: select text before copying"
	case "/":
		// The old copy.search_start clears the query only when it is empty and
		// otherwise re-opens editing with the cursor at the end.
		if strings.TrimSpace(st.query) == "" {
			st.query = ""
			st.searchErr = ""
			st.matches = nil
		}
		st.searchCol = len([]rune(st.query))
		st.searching = true
		st.searchDirty = false
	case "n":
		if strings.TrimSpace(st.query) != "" {
			return m.runCopySearch(p, st, true, true)
		}
	case "N":
		if strings.TrimSpace(st.query) != "" {
			return m.runCopySearch(p, st, false, true)
		}
	case "tab":
		// The old copy.search_mode requires the search bar to be visible.
		if st.searching || strings.TrimSpace(st.query) != "" || st.searchErr != "" {
			st.searchMode = (st.searchMode + 1) % 3
			st.matches = nil
			st.searchDirty = strings.TrimSpace(st.query) != ""
			m.toast = "search mode: " + copySearchModeName(st.searchMode)
			if st.searchDirty {
				return app.Tick(copySearchDebounce)
			}
		}
	case "page-up":
		return m.moveCopyCursorRows(p, st, -m.copyPage(st))
	case "page-down":
		return m.moveCopyCursorRows(p, st, m.copyPage(st))
	case "u":
		return m.moveCopyCursorRows(p, st, -maxInt(1, m.copyPage(st)/2))
	case "d":
		return m.moveCopyCursorRows(p, st, maxInt(1, m.copyPage(st)/2))
	case "g":
		return m.moveCopyCursorRows(p, st, -1<<20)
	case "k", "up":
		return m.moveCopyCursorRows(p, st, -1)
	case "j", "down":
		return m.moveCopyCursorRows(p, st, 1)
	case "h", "left":
		m.moveCopyCursor(st, 0, -1)
	case "l", "right":
		m.moveCopyCursor(st, 0, 1)
	case "home":
		st.cursorCol = 0
	case "end":
		st.cursorCol = maxInt(0, m.copyRowWidth(st, st.cursorRow)-1)
	case "space":
		if st.marked {
			st.marked = false
			m.clearCopyDragScroll()
		} else {
			st.marked = true
			st.markRow, st.markCol = st.cursorRow, st.cursorCol
		}
	case "H":
		// Legacy copy-scene H opens clipboard history.
		return m.openClipboardHistory()
	case "P":
		// Legacy copy-scene P pastes the system clipboard.
		return m.pasteSystem()
	case "p":
		// Legacy copy-scene p pastes the newest clipboard history entry.
		return m.pasteLatestClipboard()
	}
	return nil
}

// copyPage is the page step: the old copyModePageRows() uses the visible
// rows minus two (one row of overlap on each side), or 8 when the viewport is
// too small to page.
func (m *model) copyPage(st *copyState) int {
	viewRows := st.viewRows
	if viewRows <= 0 || (len(st.rows) > 0 && viewRows > len(st.rows)) {
		viewRows = len(st.rows)
	}
	if viewRows > 2 {
		return viewRows - 2
	}
	return 8
}

func (m *model) copyWindowRows(st *copyState) int {
	if st != nil && st.viewRows > 0 {
		return st.viewRows
	}
	if st == nil {
		return 0
	}
	return maxInt(0, len(st.rows))
}

// copyRowWidth is the display width of one viewport row (0 when unknown).
func (m *model) copyRowWidth(st *copyState, row int) int {
	if row < 0 || row >= len(st.rows) {
		return 0
	}
	return sdk.DisplayWidth(st.rows[row])
}

// copyContentRect returns the terminal content area in zero-based screen
// cells. Mouse events use one-based screen coordinates, so the caller can
// subtract this origin after converting the event to zero-based coordinates.
func (m *model) copyContentRect(p *pane) (rect, bool) {
	if p == nil {
		return rect{}, false
	}
	// Floating terminals are rendered above the normal pane cards. Search
	// from the topmost floating window so a pane's visible content wins.
	for i := len(m.floatings) - 1; i >= 0; i-- {
		f := m.floatings[i]
		if f.pane == p && !f.collapsed {
			return rect{f.x + 1, f.y + 1, maxInt(0, f.w-2), maxInt(0, f.h-2)}, true
		}
	}
	_, entries := m.paneRects(m.activeTab())
	for _, entry := range entries {
		if entry.node == nil && entry.pane == p {
			return rect{entry.r.x + 1, entry.r.y + 1, maxInt(0, entry.r.w-2), maxInt(0, entry.r.h-2)}, true
		}
	}
	return rect{}, false
}

func (m *model) prepareCopyViewport(p *pane, st *copyState) {
	if st == nil {
		return
	}
	if r, ok := m.copyContentRect(p); ok {
		st.viewRows = r.h
	}
}

// placeCopyCursorAtMouse maps a one-based terminal mouse coordinate to a
// copy cursor relative to the pane's content area.
func (m *model) placeCopyCursorAtMouse(p *pane, st *copyState, x, y int) bool {
	if st == nil || len(st.rows) == 0 {
		return false
	}
	r, ok := m.copyContentRect(p)
	if !ok || r.w <= 0 || r.h <= 0 {
		return false
	}
	col := x - 1 - r.x
	row := y - 1 - r.y
	if col < 0 || row < 0 || col >= r.w || row >= r.h {
		return false
	}
	row = clampInt(row, 0, len(st.rows)-1)
	maxCol := r.w - 1
	if st.cols > 0 {
		maxCol = minInt(maxCol, st.cols-1)
	}
	st.cursorRow = row
	st.cursorCol = clampInt(col, 0, maxInt(0, maxCol))
	return true
}

func (m *model) moveCopyCursor(st *copyState, dRow, dCol int) {
	row := clampInt(st.cursorRow+dRow, 0, maxInt(0, len(st.rows)-1))
	width := m.copyRowWidth(st, row)
	maxCol := maxInt(0, width-1)
	if width == 0 {
		maxCol = maxInt(0, st.cols-1)
	}
	st.cursorRow = row
	st.cursorCol = clampInt(st.cursorCol+dCol, 0, maxCol)
}

// moveCopyCursorRows moves the copy cursor by delta rows (positive = newer),
// scrolling the view only when the cursor hits an edge (the old ScrollCursor
// model: j/k move the cursor, the viewport follows). A mark rides with the
// content so a selection never drifts when the view scrolls.
func (m *model) moveCopyCursorRows(p *pane, st *copyState, delta int) app.Cmd {
	if st == nil || delta == 0 {
		return nil
	}
	if len(st.rows) == 0 {
		// The window is not loaded yet: remember the move and apply it once
		// the first window arrives.
		st.pendingRows += delta
		return nil
	}
	visibleRows := len(st.rows)
	if st.viewRows > 0 {
		visibleRows = minInt(visibleRows, st.viewRows)
	}
	want := st.cursorRow + delta
	if want >= 0 && want < visibleRows {
		st.cursorRow = want
		m.clampCopyCursor(st)
		// AtFrozenBottom: moving newer onto the newest row while the view is
		// already live (and nothing is selected) leaves copy mode, exactly
		// the old reduceCopyModeScrollNewer exit rule.
		if delta > 0 && !st.marked && st.offset == 0 && st.cursorRow >= len(st.rows)-1 {
			return m.endCopy(p)
		}
		return nil
	}
	overflow := want
	if want >= visibleRows {
		overflow = want - (visibleRows - 1)
	}
	// The cursor parks on the edge and the view scrolls by the leftover.
	if delta < 0 {
		st.cursorRow = 0
	} else {
		st.cursorRow = visibleRows - 1
	}
	scrollDelta := -overflow // positive = older
	return m.scrollCopyView(p, st, scrollDelta)
}

// scrollCopyView scrolls the copy view without moving the cursor and shifts
// the mark so the selection keeps pointing at the same text.
func (m *model) scrollCopyView(p *pane, st *copyState, delta int) app.Cmd {
	if st == nil || delta == 0 {
		return nil
	}
	src := m.paneSource(p)
	if src == nil || src.GetTerminalId() == "" {
		delete(m.copyPanes, p.id)
		return nil
	}
	if delta < 0 && st.offset+delta <= 0 {
		if st.marked {
			// Keep the session while a selection is open; clamp the step at
			// live instead of overshooting (legacy CopyModeStore.ScrollNewer
			// clamps the viewport at the frozen bottom).
			delta = -st.offset
			if delta == 0 {
				return nil
			}
		} else {
			return m.endCopy(p)
		}
	}
	st.scrollSeq++
	return m.emit("terminal.scroll", &pb.MethodParams{
		Endpoint: endpointOf(src), Id: src.GetTerminalId(), Delta: int32(delta),
		Rows: int32(m.copyWindowRows(st)), View: p.id,
	}, opMsg{op: "scroll", ref: p.id, delta: delta, seq: st.scrollSeq})
}

// copyScroll moves the session view by delta rows (positive = older) and
// closes it when the bottom is reached going newer (old AtFrozenBottom).
func (m *model) copyScroll(p *pane, st *copyState, delta int) app.Cmd {
	if st == nil || delta == 0 {
		return nil
	}
	src := m.paneSource(p)
	if src == nil || src.GetTerminalId() == "" {
		delete(m.copyPanes, p.id)
		return nil
	}
	if delta < 0 && st.offset+delta <= 0 {
		return m.endCopy(p)
	}
	st.scrollSeq++
	return m.emit("terminal.scroll", &pb.MethodParams{
		Endpoint: endpointOf(src), Id: src.GetTerminalId(), Delta: int32(delta),
		Rows: int32(m.copyWindowRows(st)), View: p.id,
	}, opMsg{op: "scroll", ref: p.id, delta: delta, seq: st.scrollSeq})
}

// copyDragEdgeDirection ports updateHistoryMouseScrollEdge: the edge band is
// the first and last content row inside the pane's content rect columns. The
// pointer only auto-scrolls while it is held in that band; leaving it (or the
// middle) clears the direction. It returns -1 (older), +1 (newer) or 0.
func (m *model) copyDragEdgeDirection(p *pane, x, y int) int {
	r, ok := m.copyContentRect(p)
	if !ok || r.w <= 0 || r.h <= 0 {
		return 0
	}
	col, row := x-1-r.x, y-1-r.y
	if col < 0 || col >= r.w {
		return 0
	}
	if row <= 0 {
		return -1
	}
	if row >= r.h-1 {
		return 1
	}
	return 0
}

// copyDragScrollAvailable is the legacy historyMouseScrollAvailable gate: the
// pane still has an open copy session with a MARK, and the active drag is the
// copy selection on that pane (the frozen token check is implicit here because
// the session only exists while its window is bound).
func (m *model) copyDragScrollAvailable(p *pane) bool {
	if p == nil || m.overlay != "" || m.dragging != "copy:"+p.id {
		return false
	}
	st := m.copyFor(p)
	return st != nil && st.marked
}

// copyDragAtLiveBottom reports whether the frozen viewport has reached the live
// bottom (the legacy CopyModeStore.AtFrozenBottom): the newer edge disarms there
// so a held pointer does not keep emitting clamped terminal.scroll requests.
func (m *model) copyDragAtLiveBottom(st *copyState) bool {
	return st != nil && st.offset == 0 && st.cursorRow >= len(st.rows)-1
}

// setCopyDragEdge updates the drag auto-scroll direction from one pointer
// position and arms the 100ms tick when the direction turns on. A direction
// change re-arms; leaving the band clears it (legacy
// updateHistoryMouseScrollEdge). The tick is only armed while a MARK is set.
func (m *model) setCopyDragEdge(p *pane, x, y int) app.Cmd {
	if p == nil || !m.copyDragScrollAvailable(p) {
		m.clearCopyDragScroll()
		return nil
	}
	dir := m.copyDragEdgeDirection(p, x, y)
	if dir == m.copyDragDir && m.copyDragPane == p.id {
		return nil
	}
	m.copyDragDir = dir
	m.copyDragPane = p.id
	if dir == 0 {
		m.copyDragTick = false
		return nil
	}
	if dir > 0 && m.copyDragAtLiveBottom(m.copyFor(p)) {
		// Already at the live bottom: the newer edge cannot advance.
		m.copyDragTick = false
		return nil
	}
	if m.copyDragTick {
		// A step is already in flight for the old direction; it will observe
		// the new direction when it fires.
		return nil
	}
	m.copyDragTick = true
	return app.Tick(copyMouseScrollInterval)
}

// clearCopyDragScroll stops the edge auto-scroll: release, direction loss, an
// ended copy session or a cleared mark all take this path (the legacy code
// clears mouseDrag.NextHistoryScroll / drops the gesture timer).
func (m *model) clearCopyDragScroll() {
	m.copyDragDir = 0
	m.copyDragPane = ""
	m.copyDragTick = false
}

// applyCopyEdgeAutoScroll is the CopyModeMouseAutoScrollMsg case: it scrolls
// the marked selection two rows older/newer per tick, keeps the cursor parked
// on the edge (moveCopyCursorRows parks it) and re-arms the next tick while the
// direction stays non-zero. It mirrors reduceCopyModeScroll*Rows(…, 2) and the
// "DeferredScrollRows = 0" reset that keeps a late page from replaying.
func (m *model) applyCopyEdgeAutoScroll() app.Cmd {
	// Only a step that our own 100ms Tick armed may run: a search-debounce
	// Tick must never move the selection, and two outstanding ticks must not
	// accumulate missed steps (legacy enqueueDueHistoryMouseScroll's fixed
	// interval re-arm).
	if !m.copyDragTick {
		return nil
	}
	m.copyDragTick = false
	dir := m.copyDragDir
	if dir == 0 || m.overlay != "" {
		return nil
	}
	p := m.paneByID(m.copyDragPane)
	if p == nil || !m.copyDragScrollAvailable(p) {
		m.clearCopyDragScroll()
		return nil
	}
	st := m.copyFor(p)
	var cmd app.Cmd
	switch {
	case dir < 0:
		cmd = m.moveCopyCursorRows(p, st, -copyMouseScrollRows)
	case dir > 0:
		cmd = m.moveCopyCursorRows(p, st, copyMouseScrollRows)
	}
	if dir > 0 && m.copyDragAtLiveBottom(st) {
		// The step reached the live bottom; stop the timer (legacy
		// historyMouseScrollAvailable).
		m.clearCopyDragScroll()
		return cmd
	}
	m.copyDragTick = true
	return chain(cmd, app.Tick(copyMouseScrollInterval))
}

// fetchCopyWindow asks the host for the window at the session offset; the
// response refreshes the cached rows (and, for a search, the matches).
func (m *model) fetchCopyWindow(p *pane, st *copyState) app.Cmd {
	src := m.paneSource(p)
	if src == nil || src.GetTerminalId() == "" {
		return nil
	}
	st.searchSeq++
	return m.emit("history.window", &pb.MethodParams{
		Endpoint: endpointOf(src), Id: src.GetTerminalId(),
		Rows: int32(m.copyWindowRows(st)), View: p.id,
	}, opMsg{op: "window", ref: p.id, seq: st.searchSeq})
}

// applyCopyWindow refreshes the session after a window response and, for a
// search, moves to the next match.
func (m *model) applyCopyWindow(p *pane, st *copyState, rows []string, offset int, seq uint64) app.Cmd {
	if seq != 0 && seq != st.searchSeq {
		return nil
	}
	st.offset = offset
	st.rows = rows
	if !st.placed && len(rows) > 0 {
		// Old copy mode enters at the latest position: bottom-left.
		st.placed = true
		st.cursorRow = len(rows) - 1
		st.cursorCol = 0
	}
	m.refreshCopyMatches(st)
	m.clampCopyCursor(st)
	m.clampCopyCursorToViewport(st)
	if st.pendingRows != 0 {
		pending := st.pendingRows
		st.pendingRows = 0
		return m.moveCopyCursorRows(p, st, pending)
	}
	return nil
}

// applyCopyScroll refreshes the session after a scroll response and exits
// when the view reached the live bottom going newer.
func (m *model) applyCopyScroll(p *pane, st *copyState, rows []string, offset, delta int) app.Cmd {
	if st.marked {
		st.markRow += offset - st.offset
	}
	previousOffset := st.offset
	st.offset = offset
	if len(rows) > 0 {
		st.rows = rows
	}
	m.refreshCopyMatches(st)
	m.clampCopyCursor(st)
	m.clampCopyCursorToViewport(st)
	if delta < 0 && offset == 0 && !st.marked {
		return m.endCopy(p)
	}
	if delta > 0 && offset == previousOffset && offset > 0 && m.copyDragDir < 0 {
		// The older edge could not load more history (the host clamped the same
		// offset). Stop the timer instead of re-emitting the same scroll, the
		// legacy historyMouseScrollAvailable OlderRequestState()==Ready check.
		m.clearCopyDragScroll()
	}
	return nil
}

func (m *model) clampCopyCursor(st *copyState) {
	if len(st.rows) == 0 {
		st.cursorRow = 0
		st.cursorCol = 0
		return
	}
	st.cursorRow = clampInt(st.cursorRow, 0, len(st.rows)-1)
	width := m.copyRowWidth(st, st.cursorRow)
	if width > 0 {
		st.cursorCol = clampInt(st.cursorCol, 0, width-1)
	}
}

// clampCopyCursorToViewport keeps the program cursor inside the currently
// rendered terminal box. A history window can be taller than a split pane,
// and a resize can shrink a pane after the window was loaded.
func (m *model) clampCopyCursorToViewport(st *copyState) {
	if st == nil || len(st.rows) == 0 {
		return
	}
	maxRow := len(st.rows) - 1
	if st.viewRows > 0 {
		maxRow = minInt(maxRow, st.viewRows-1)
	}
	st.cursorRow = clampInt(st.cursorRow, 0, maxRow)
	maxCol := st.cols - 1
	if maxCol < 0 {
		maxCol = m.copyRowWidth(st, st.cursorRow) - 1
	}
	st.cursorCol = clampInt(st.cursorCol, 0, maxInt(0, maxCol))
}

// selection returns the reading-order selection endpoints, ok=false when no
// mark is set.
func (st *copyState) selection() (startRow, startCol, endRow, endCol int, ok bool) {
	if !st.marked {
		return 0, 0, 0, 0, false
	}
	startRow, startCol = st.markRow, st.markCol
	endRow, endCol = st.cursorRow, st.cursorCol
	if startRow > endRow || (startRow == endRow && startCol > endCol) {
		startRow, endRow = endRow, startRow
		startCol, endCol = endCol, startCol
	}
	return startRow, startCol, endRow, endCol, true
}

// selectionSpans is the highlighted selection as per-row inclusive ranges.
// Like the old renderer, the start and middle rows are highlighted to the
// pane edge (the blank tail keeps the selection background), while the last
// row stops at the cursor column.
func (m *model) selectionSpans(st *copyState) []copyMatch {
	startRow, startCol, endRow, endCol, ok := st.selection()
	if !ok {
		return nil
	}
	var spans []copyMatch
	for row := startRow; row <= endRow; row++ {
		to := st.cols - 1
		if to < 0 {
			to = m.copyRowWidth(st, row) - 1
		}
		from := 0
		if row == startRow {
			from = startCol
		}
		if row == endRow {
			to = endCol
		}
		if to < from {
			continue
		}
		spans = append(spans, copyMatch{row: row, startCol: from, endCol: to})
	}
	return spans
}

// copySelection sends the selection to the host (terminal.copy{sel}) as
// linear cell indices over the visible window. exit=true closes the session
// on success (Enter), exit=false keeps it (y).
func (m *model) copySelection(p *pane, st *copyState, exit bool) app.Cmd {
	src := m.paneSource(p)
	if src == nil || src.GetTerminalId() == "" {
		return nil
	}
	startRow, startCol, endRow, endCol, ok := st.selection()
	if !ok {
		m.toast = "nothing to copy"
		return nil
	}
	cols := st.cols
	if cols <= 0 {
		cols = maxInt(m.copyRowWidth(st, startRow), 1)
	}
	start := startRow*cols + startCol
	end := endRow*cols + endCol
	return m.emit("terminal.copy", &pb.MethodParams{
		Endpoint: endpointOf(src), Id: src.GetTerminalId(), View: p.id,
		Sel: &pb.Selection{Mode: "char", Start: int32(start), End: int32(end)},
	}, opMsg{op: "copy", ref: p.id, exit: exit})
}

// findCopyMatches scans plain rows for the literal query (case sensitive,
// the old text search mode) and returns matches with display columns.
func findCopyMatches(rows []string, query string) []copyMatch {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	var out []copyMatch
	for row, line := range rows {
		out = append(out, findRowMatches(row, line, query)...)
	}
	return out
}

// findRowMatches finds the literal query in one row and returns display-cell
// ranges.
func findRowMatches(row int, line, query string) []copyMatch {
	if query == "" {
		return nil
	}
	type runeCell struct {
		text  string
		col   int
		width int
	}
	runes := []rune(line)
	cells := make([]runeCell, 0, len(runes))
	col := 0
	for _, r := range runes {
		width := sdk.RuneWidth(r)
		if width == 0 {
			continue
		}
		cells = append(cells, runeCell{text: string(r), col: col, width: width})
		col += width
	}
	var out []copyMatch
	queryRunes := []rune(query)
	for i := 0; i+len(queryRunes) <= len(cells); i++ {
		match := true
		for j, r := range queryRunes {
			if cells[i+j].text != string(r) {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		last := cells[i+len(queryRunes)-1]
		out = append(out, copyMatch{row: row, startCol: cells[i].col, endCol: last.col + last.width - 1})
		i += len(queryRunes) - 1
	}
	return out
}

// refreshCopyMatches recomputes the highlightable matches inside the visible
// window and refreshes the footer status counts.
func (m *model) refreshCopyMatches(st *copyState) {
	st.currentMatch = nil
	if strings.TrimSpace(st.query) == "" || len(st.rows) == 0 {
		st.matches = nil
		st.matchIdx = 0
		st.searchTotal = 0
		st.searchCurrent = 0
		return
	}
	matches, ok := findModeMatches(st.rows, st.searchMode, st.query)
	if !ok {
		// An invalid glob/regex is a search error, not "no match".
		st.matches = nil
		st.searchTotal = 0
		st.searchCurrent = 0
		st.searchErr = "invalid pattern"
		return
	}
	st.matches = matches
	if st.matchIdx >= len(st.matches) {
		st.matchIdx = maxInt(0, len(st.matches)-1)
	}
	st.searchTotal = len(st.matches)
	if len(st.matches) == 0 {
		st.searchCurrent = 0
	} else {
		st.searchCurrent = st.matchIdx + 1
	}
}

// runCopySearch delegates navigation to the terminal. Typing only refreshes
// highlights in the loaded window; n/N and enter move the frozen viewport.
// Like the old beginCopyModeSearch, a repeat search advances past the current
// match (forward: one column after its end, backward: its start) instead of
// restarting from the raw cursor.
func (m *model) runCopySearch(p *pane, st *copyState, forward, move bool) app.Cmd {
	if strings.TrimSpace(st.query) == "" {
		return nil
	}
	st.forward = forward
	if move {
		src := m.paneSource(p)
		if src == nil {
			return nil
		}
		st.searchSeq++
		cols := maxInt(1, st.cols)
		start := st.cursorRow*cols + st.cursorCol
		if len(st.currentMatch) > 0 {
			if forward {
				last := st.currentMatch[len(st.currentMatch)-1]
				start = last.row*cols + last.endCol + 1
			} else {
				first := st.currentMatch[0]
				start = first.row*cols + first.startCol
			}
		}
		return m.emit("terminal.search", &pb.MethodParams{
			Endpoint: endpointOf(src), Id: src.GetTerminalId(), Query: st.query,
			SearchMode: copySearchModeName(st.searchMode), Backward: !forward,
			Sel: &pb.Selection{Start: int32(start)}, View: p.id,
		}, opMsg{op: "search", ref: p.id, seq: st.searchSeq})
	}
	m.refreshCopyMatches(st)
	return nil
}

// applyTerminalSearch accepts the terminal-owned search result. The terminal
// has already moved its viewport; the shell only moves its cursor and paints
// the returned visible matches.
func (m *model) applyTerminalSearch(st *copyState, result opMsg) {
	if result.seq != st.searchSeq {
		return
	}
	if !result.ok {
		st.searchErr = result.err
		m.toast = "search: " + result.err
		return
	}
	if !result.found {
		st.searchErr = "no match"
		m.toast = "search: no match for " + st.query
		m.refreshCopyMatches(st)
		return
	}
	st.searchErr = ""
	if st.marked {
		st.markRow += result.offset - st.offset
	}
	st.rows, st.offset = result.rows, result.offset
	cols := maxInt(1, st.cols)
	st.cursorRow, st.cursorCol = result.matchStart/cols, result.matchStart%cols
	m.clampCopyCursorToViewport(st)
	m.refreshCopyMatches(st)
	for row := result.matchStart / cols; row <= (result.matchEnd-1)/cols && row < len(st.rows); row++ {
		from, to := 0, minInt(cols-1, maxInt(0, m.copyRowWidth(st, row)-1))
		if row == result.matchStart/cols {
			from = result.matchStart % cols
		}
		if row == (result.matchEnd-1)/cols {
			to = (result.matchEnd - 1) % cols
		}
		if to >= from {
			st.currentMatch = append(st.currentMatch, copyMatch{row: row, startCol: from, endCol: to})
		}
	}
	for i, match := range st.matches {
		if match.row == st.cursorRow && match.startCol == st.cursorCol {
			st.matchIdx = i
			break
		}
	}
	// The scan status badge: position of the current match within the visible
	// matches (the terminal.search response is per-match, so the total is the
	// loaded-window count, matching what the user sees highlighted).
	st.searchCurrent = st.matchIdx + 1
	st.searchTotal = len(st.matches)
	if result.wrapped {
		m.toast = "search wrapped"
	}
}

// copyProps renders the session state as terminal component props.
func (m *model) copyProps(st *copyState) map[string]string {
	if st == nil {
		return nil
	}
	props := map[string]string{
		"copy.cursor":          strconv.Itoa(st.cursorRow) + "," + strconv.Itoa(st.cursorCol),
		"copy.style.cursor":    copyCursorStyle,
		"copy.style.sel":       copySelStyle,
		"copy.style.match":     copyMatchStyle,
		"copy.style.match_cur": copyMatchCurStyle,
	}
	if spans := m.selectionSpans(st); len(spans) > 0 {
		props["copy.sel"] = formatCopySpans(spans)
	}
	if len(st.matches) > 0 {
		normal := make([]copyMatch, 0, len(st.matches))
		var current *copyMatch
		for i, match := range st.matches {
			if i == st.matchIdx {
				m := match
				current = &m
				continue
			}
			normal = append(normal, match)
		}
		if len(normal) > 0 {
			props["copy.match"] = formatCopySpans(normal)
		}
		if current != nil {
			props["copy.match_current"] = formatCopySpans([]copyMatch{*current})
		}
	}
	if len(st.currentMatch) > 0 {
		props["copy.match_current"] = formatCopySpans(st.currentMatch)
	}
	return props
}

func formatCopySpans(spans []copyMatch) string {
	var b strings.Builder
	for i, span := range spans {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(strconv.Itoa(span.row))
		b.WriteByte(',')
		b.WriteString(strconv.Itoa(span.startCol))
		b.WriteByte(',')
		b.WriteString(strconv.Itoa(span.endCol))
	}
	return b.String()
}

// copySearchLabel is the legacy mode label shown in the search bar prefix.
func copySearchLabel(mode int) string {
	switch mode {
	case copySearchGlob:
		return "GLOB"
	case copySearchRegex:
		return "REGEX"
	default:
		return "TEXT"
	}
}

// copySearchFooterRuns renders the legacy copy search bar that replaces the
// global footer: a left-aligned "⌕ [MODE] query" prefix/value with the text
// cursor placed on the value while editing, and a right-aligned status/hint
// (scanning count, "no match", errors, and the key hints on wide terminals).
// The query is windowed around the edit cursor so a long query still shows the
// caret (legacy searchBarPresentation).
func (m *model) copySearchFooterRuns(st *copyState) (left, right []footerRun) {
	if st == nil {
		return nil, nil
	}
	prefix := "\u2315 [" + copySearchLabel(st.searchMode) + "] "
	hint := "  /edit Tab mode N\u2191 n\u2193 Esc\u00d7"
	if st.searching {
		hint = "  Tab mode Enter\u2713 Esc\u00d7"
	}
	status := m.copySearchStatus(st)
	// Right side: status (styled) then the wide hint (legacy WideMinWidth 58).
	if status != "" {
		style := stMuted
		if st.searchErr != "" {
			style = stWarning
		}
		right = append(right, footerRun{status, style, "", 1})
	}
	if m.cols >= 58 {
		right = append(right, footerRun{hint, stMuted, "", 1})
	}

	// Left side: prefix + a query window that keeps the caret visible.
	avail := maxInt(1, m.cols-sdk.DisplayWidth(prefix)-runsWidth(right))
	value := st.query
	cursor := minInt(maxInt(0, st.searchCol), len([]rune(value)))
	valueStart := 0
	if sdk.DisplayWidth(value) > avail {
		valueCursor := sdk.DisplayWidth(string([]rune(value)[:cursor]))
		valueStart = minInt(maxInt(0, valueCursor-avail/2), sdk.DisplayWidth(value)-avail)
		value = sliceCells(value, valueStart, valueStart+avail)
	}
	left = append(left, footerRun{prefix, stAccent, "", 1})
	if value != "" {
		left = append(left, footerRun{value, stContent, "", 1})
	}
	return left, right
}

// copySearchStatus is the legacy search status string (the right-aligned badge).
func (m *model) copySearchStatus(st *copyState) string {
	switch {
	case st.searchErr != "":
		return st.searchErr
	case strings.TrimSpace(st.query) == "":
		return ""
	case st.searchTotal == 0:
		return "no match"
	case st.searchCurrent > 0:
		return strconv.Itoa(st.searchCurrent) + "/" + strconv.Itoa(st.searchTotal)
	default:
		return "\u2013/" + strconv.Itoa(st.searchTotal)
	}
}

func runsWidth(runs []footerRun) int {
	width := 0
	for _, run := range runs {
		width += sdk.DisplayWidth(run.text)
	}
	return width
}

// sliceCells returns the substring of s covering display columns [from, to).
func sliceCells(s string, from, to int) string {
	if to <= from {
		return ""
	}
	var b strings.Builder
	col := 0
	for _, r := range s {
		w := sdk.RuneWidth(r)
		if w <= 0 {
			if col >= from && col < to {
				b.WriteRune(r)
			}
			continue
		}
		if col+w > from && col < to {
			b.WriteRune(r)
		}
		col += w
		if col >= to {
			break
		}
	}
	return b.String()
}

func copySearchModeName(mode int) string {
	switch mode {
	case copySearchGlob:
		return "glob"
	case copySearchRegex:
		return "regex"
	default:
		return "text"
	}
}

// handleCopySearchKey edits the search query (the old search editing state):
// typing inserts at the query cursor, left/right/home/end move it, backspace
// and delete remove around it, enter runs the scan and esc leaves editing.
func (m *model) handleCopySearchKey(p *pane, st *copyState, key, char string) app.Cmd {
	runes := []rune(st.query)
	clampCol := func(col int) int {
		if col < 0 {
			return 0
		}
		if col > len(runes) {
			return len(runes)
		}
		return col
	}
	mutate := func(next []rune, col int) app.Cmd {
		st.query = string(next)
		st.searchErr = ""
		if col < 0 {
			col = 0
		}
		if col > len(next) {
			col = len(next)
		}
		st.searchCol = col
		st.searchDirty = strings.TrimSpace(st.query) != ""
		if !st.searchDirty {
			st.matches = nil
			return nil
		}
		return app.Tick(copySearchDebounce)
	}
	switch key {
	case "esc":
		st.searching = false
	case "enter":
		// The old copy.accept stops editing and runs the search from the current
		// match (or the cursor when there is none) when a query is present.
		st.searching = false
		st.searchDirty = false
		if strings.TrimSpace(st.query) != "" {
			st.forward = true
			return m.runCopySearch(p, st, true, true)
		}
	case "tab":
		// The old copy.search_mode cycles the mode and keeps editing.
		st.searchMode = (st.searchMode + 1) % 3
		m.toast = "search mode: " + copySearchModeName(st.searchMode)
		st.matches = nil
		st.searchDirty = strings.TrimSpace(st.query) != ""
		if st.searchDirty {
			return app.Tick(copySearchDebounce)
		}
	case "backspace":
		col := st.searchCol
		if col > 0 {
			next := append(append([]rune(nil), runes[:col-1]...), runes[col:]...)
			return mutate(next, col-1)
		}
	case "delete":
		col := st.searchCol
		if col < len(runes) {
			next := append(append([]rune(nil), runes[:col]...), runes[col+1:]...)
			return mutate(next, col)
		}
	case "left":
		st.searchCol = clampCol(st.searchCol - 1)
	case "right":
		st.searchCol = clampCol(st.searchCol + 1)
	case "home":
		st.searchCol = 0
	case "end":
		st.searchCol = len(runes)
	default:
		text := char
		if text == "" && len([]rune(key)) == 1 {
			text = key
		}
		if text != "" {
			col := st.searchCol
			next := append(append([]rune(nil), runes[:col]...), append([]rune(text), runes[col:]...)...)
			return mutate(next, col+len([]rune(text)))
		}
	}
	return nil
}

// applySearchScan runs the pending debounced scan.
func (m *model) applySearchScan() app.Cmd {
	p := m.focusContentPane()
	st := m.copyFor(p)
	if p == nil || st == nil || !st.searchDirty || !st.searching {
		return nil
	}
	st.searchDirty = false
	if strings.TrimSpace(st.query) == "" {
		return nil
	}
	return m.runCopySearch(p, st, st.forward, false)
}

// resetCopyToLatest re-enters copy at the newest position (the old
// copy.enter on an already active view): the frozen window is released, the
// cursor returns to the bottom and the selection/search are cleared.
func (m *model) resetCopyToLatest(p *pane, st *copyState) app.Cmd {
	if p == nil || st == nil {
		return nil
	}
	st.marked = false
	st.matches = nil
	st.matchIdx = 0
	st.query = ""
	st.searching = false
	st.searchDirty = false
	st.searchErr = ""
	st.pendingRows = 0
	st.searchSeq++
	st.offset = 0
	st.placed = false
	src := m.paneSource(p)
	if src != nil && src.GetTerminalId() != "" {
		return m.emit("terminal.scrollEnd", &pb.MethodParams{
			Endpoint: endpointOf(src), Id: src.GetTerminalId(), View: p.id,
		}, opMsg{op: "resetCopy", ref: p.id, seq: st.searchSeq})
	}
	return nil
}

// markCopyAtCursor anchors the selection at the copy cursor (the old mouse
// select intent and the space key).
func (m *model) markCopyAtCursor(st *copyState) {
	if st == nil {
		return
	}
	st.marked = true
	st.markRow, st.markCol = st.cursorRow, st.cursorCol
}

// copySearchRegexp compiles the query for the active mode. ok=false means the
// pattern is invalid (the caller keeps the previous matches).
func copySearchRegexp(mode int, query string) (*regexp.Regexp, bool) {
	switch mode {
	case copySearchGlob:
		var b strings.Builder
		b.WriteString("^(?:")
		runes := []rune(query)
		for i := 0; i < len(runes); i++ {
			switch runes[i] {
			case '*':
				b.WriteString(".*")
			case '?':
				b.WriteString(".")
			case '[':
				end := i + 1
				for end < len(runes) && runes[end] != ']' {
					end++
				}
				if end >= len(runes) {
					return nil, false
				}
				b.WriteByte('[')
				start := i + 1
				if start < end && runes[start] == '!' {
					b.WriteByte('^')
					start++
				}
				b.WriteString(string(runes[start:end]))
				b.WriteByte(']')
				i = end
			case '\\':
				if i+1 >= len(runes) {
					return nil, false
				}
				i++
				b.WriteString(regexp.QuoteMeta(string(runes[i])))
			default:
				b.WriteString(regexp.QuoteMeta(string(runes[i])))
			}
		}
		b.WriteString(")$")
		re, err := regexp.Compile(b.String())
		return re, err == nil
	case copySearchRegex:
		re, err := regexp.Compile(query)
		return re, err == nil
	default:
		return nil, true
	}
}

// findModeMatches scans rows for the active search mode and returns matches
// in display-cell columns. Invalid patterns return ok=false.
func findModeMatches(rows []string, mode int, query string) ([]copyMatch, bool) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, true
	}
	if mode == copySearchText {
		return findCopyMatches(rows, query), true
	}
	re, ok := copySearchRegexp(mode, query)
	if !ok {
		return nil, false
	}
	var out []copyMatch
	for row, line := range rows {
		byteRanges := re.FindAllStringIndex(line, -1)
		for _, loc := range byteRanges {
			startCol := sdk.DisplayWidth(line[:loc[0]])
			width := sdk.DisplayWidth(line[loc[0]:loc[1]])
			if width <= 0 {
				continue
			}
			out = append(out, copyMatch{row: row, startCol: startCol, endCol: startCol + width - 1})
		}
	}
	return out, true
}

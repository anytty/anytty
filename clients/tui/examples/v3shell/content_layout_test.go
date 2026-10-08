package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk"
)

// TestContentLayoutExtentFormula pins the ported legacy geometry
// (render.applyContentLayoutToExtent + alignedContentOrigin): mode=fit fills
// the content rect, mode=center aligns both axes center, otherwise the per-axis
// align decides, and the pan is applied last (X -= PanX, Y -= PanY).
func TestContentLayoutExtentFormula(t *testing.T) {
	content := rect{0, 0, 18, 8}
	auto := func(alignX, alignY string) contentLayout {
		return contentLayout{mode: layoutModeAuto, alignX: alignX, alignY: alignY}
	}
	cases := []struct {
		name                             string
		layout                           contentLayout
		baseCols, baseRows               int
		wantX, wantY, wantCols, wantRows int
	}{
		{"default start", contentLayout{}, 10, 4, 0, 0, 10, 4},
		{"align end x", auto(alignEnd, alignStart), 10, 4, 8, 0, 10, 4},
		{"align bottom y", auto(alignStart, alignEnd), 10, 4, 0, 4, 10, 4},
		{"center both", contentLayout{mode: layoutModeCenter}, 10, 4, 4, 2, 10, 4},
		{"center-x", contentLayout{mode: layoutModeAuto, alignX: alignCenter, alignY: alignStart}, 10, 4, 4, 0, 10, 4},
		{"center-y", contentLayout{mode: layoutModeAuto, alignX: alignStart, alignY: alignCenter}, 10, 4, 0, 2, 10, 4},
		{"fit fills", contentLayout{mode: layoutModeFit}, 10, 4, 0, 0, 18, 8},
		// pan is X -= PanX, Y -= PanY: pan-left (DeltaX -2) moves the content
		// RIGHT, pan-up (DeltaY -1) moves it DOWN (legacy semantics).
		{"pan-left", contentLayout{mode: layoutModeAuto, panX: -2}, 10, 4, 2, 0, 10, 4},
		{"pan-right", contentLayout{mode: layoutModeAuto, panX: 2}, 10, 4, -2, 0, 10, 4},
		{"pan-up", contentLayout{mode: layoutModeAuto, panY: -1}, 10, 4, 0, 1, 10, 4},
		{"pan-down", contentLayout{mode: layoutModeAuto, panY: 1}, 10, 4, 0, -1, 10, 4},
		// align + pan compose: center-x then pan-left.
		{"center-x pan-left", contentLayout{mode: layoutModeAuto, alignX: alignCenter, alignY: alignStart, panX: -2}, 10, 4, 6, 0, 10, 4},
	}
	for _, tc := range cases {
		x, y, cols, rows := contentLayoutExtent(tc.layout, tc.baseCols, tc.baseRows, content)
		if x != tc.wantX || y != tc.wantY || cols != tc.wantCols || rows != tc.wantRows {
			t.Fatalf("%s: extent = (%d,%d %dx%d), want (%d,%d %dx%d)",
				tc.name, x, y, cols, rows, tc.wantX, tc.wantY, tc.wantCols, tc.wantRows)
		}
	}
}

// TestAlignedContentOriginRounding pins the half-up slack rounding, including
// the negative (clipping) side.
func TestAlignedContentOriginRounding(t *testing.T) {
	if got := alignedContentOrigin(alignCenter, 18, 10); got != 4 {
		t.Fatalf("center 18/10 = %d, want 4", got)
	}
	if got := alignedContentOrigin(alignCenter, 18, 30); got != -6 {
		t.Fatalf("center 18/30 = %d, want -6", got)
	}
	if got := alignedContentOrigin(alignEnd, 18, 10); got != 8 {
		t.Fatalf("end 18/10 = %d, want 8", got)
	}
	if got := alignedContentOrigin(alignStart, 18, 10); got != 0 {
		t.Fatalf("start 18/10 = %d, want 0", got)
	}
}

// TestResizeAlignKeysMoveFollowerExtent drives the resize-scene align keys on a
// follower pane (source extent 10x4 in an 18x8 content rect) and asserts the
// applied extent origin matches the formula.
func TestResizeAlignKeysMoveFollowerExtent(t *testing.T) {
	// 40x12 viewport, row split: left card {0,1,20,10}, content {1,2,18,8}.
	// Right owns the source, so the left pane is the follower.
	m, left, _, _ := twoPaneSourceModel(t, 40, 12, 10, 4, true)
	m.mode = modeResize
	m.focusPaneObject(left)
	content := rect{1, 2, 18, 8}

	extent := func() (int, int) {
		x, y, _, _ := m.paneContentExtent(left, content)
		return x, y
	}

	runCmd(t, m, key(m, "0")) // align-left => X start
	if x, y := extent(); x != 0 || y != 0 {
		t.Fatalf("align-left extent origin = (%d,%d), want (0,0)", x, y)
	}
	runCmd(t, m, key(m, "$")) // align-right => X end
	if x, _ := extent(); x != 8 {
		t.Fatalf("align-right X = %d, want 8", x)
	}
	runCmd(t, m, key(m, "^")) // align-top => Y start
	if _, y := extent(); y != 0 {
		t.Fatalf("align-top Y = %d, want 0", y)
	}
	runCmd(t, m, key(m, "B")) // align-bottom => Y end
	if _, y := extent(); y != 4 {
		t.Fatalf("align-bottom Y = %d, want 4", y)
	}
	runCmd(t, m, key(m, "m")) // center both axes
	if x, y := extent(); x != 4 || y != 2 {
		t.Fatalf("center extent origin = (%d,%d), want (4,2)", x, y)
	}
	runCmd(t, m, key(m, "|")) // center-x only
	if x, y := extent(); x != 4 || y != 0 {
		t.Fatalf("center-x extent origin = (%d,%d), want (4,0)", x, y)
	}
	runCmd(t, m, key(m, "_")) // center-y only
	if x, y := extent(); x != 0 || y != 2 {
		t.Fatalf("center-y extent origin = (%d,%d), want (0,2)", x, y)
	}
}

// TestResizePanAndFitKeys pins the exact pan deltas and the fit/reset behavior
// on a follower pane.
func TestResizePanAndFitKeys(t *testing.T) {
	m, left, _, _ := twoPaneSourceModel(t, 40, 12, 10, 4, true)
	m.mode = modeResize
	m.focusPaneObject(left)
	content := rect{1, 2, 18, 8}

	extent := func() (x, y, cols, rows int) {
		return m.paneContentExtent(left, content)
	}

	// pan-left (A / shift-left) => PanX -2 => origin X +2.
	runCmd(t, m, key(m, "A"))
	if x, _, _, _ := extent(); x != 2 {
		t.Fatalf("pan-left X = %d, want 2", x)
	}
	// pan-right (D) adds +2, back to 0, then +2 again => X -2.
	runCmd(t, m, key(m, "D"))
	if x, _, _, _ := extent(); x != 0 {
		t.Fatalf("pan-right back X = %d, want 0", x)
	}
	runCmd(t, m, key(m, "D"))
	if x, _, _, _ := extent(); x != -2 {
		t.Fatalf("pan-right again X = %d, want -2", x)
	}
	// pan-up (W) => PanY -1 => origin Y +1.
	runCmd(t, m, key(m, "W"))
	if _, y, _, _ := extent(); y != 1 {
		t.Fatalf("pan-up Y = %d, want 1", y)
	}
	// pan-down (S) adds +1, back to 0, then +1 again => Y -1.
	runCmd(t, m, key(m, "S"))
	if _, y, _, _ := extent(); y != 0 {
		t.Fatalf("pan-down back Y = %d, want 0", y)
	}
	runCmd(t, m, key(m, "S"))
	if _, y, _, _ := extent(); y != -1 {
		t.Fatalf("pan-down again Y = %d, want -1", y)
	}
	// The shift+arrow aliases share the same commands.
	runCmd(t, m, key(m, "shift-left"))
	if x, _, _, _ := extent(); x != 0 {
		t.Fatalf("shift-left X = %d, want 0", x)
	}
	// fit fills the content rect (the pan still applies, so reset first).
	runCmd(t, m, key(m, "r")) // reset layout + split
	runCmd(t, m, key(m, "M")) // replica content layout_toggle: auto -> fit
	if m.focusPane() != left {
		t.Fatal("focus moved")
	}
	x, y, cols, rows := extent()
	if x != 0 || y != 0 || cols != 18 || rows != 8 {
		t.Fatalf("fit extent = (%d,%d %dx%d), want (0,0 18x8)", x, y, cols, rows)
	}
	// r resets both the split balance and the content layout back to the
	// source extent.
	runCmd(t, m, key(m, "r"))
	if !left.layout.isDefault() {
		t.Fatalf("r must reset the content layout, got %+v", left.layout)
	}
	x, y, cols, rows = extent()
	if x != 0 || y != 0 || cols != 10 || rows != 4 {
		t.Fatalf("reset extent = (%d,%d %dx%d), want (0,0 10x4)", x, y, cols, rows)
	}
}

// TestContentLayoutModeToggleCycle pins the legacy auto -> fit -> center -> auto
// cycle, bound to `M` in this replica (see the `space` deviation).
func TestContentLayoutModeToggleCycle(t *testing.T) {
	m, left, _, _ := twoPaneSourceModel(t, 40, 12, 10, 4, true)
	m.mode = modeResize
	m.focusPaneObject(left)
	steps := []string{layoutModeFit, layoutModeCenter, layoutModeAuto, layoutModeFit}
	for i, want := range steps {
		runCmd(t, m, key(m, "M"))
		if left.layout.mode != want {
			t.Fatalf("toggle step %d mode = %q, want %q", i, left.layout.mode, want)
		}
	}
}

// TestResizeLayoutToastBody pins the legacy terminalViewLayoutToast body:
// "<lock> <mode> pan:X,Y align:X/Y".
func TestResizeLayoutToastBody(t *testing.T) {
	m, left, _, _ := twoPaneSourceModel(t, 40, 12, 10, 4, true)
	m.mode = modeResize
	m.focusPaneObject(left)
	runCmd(t, m, key(m, "m"))
	if want := "unlocked center pan:0,0 align:center/center"; m.toast != want {
		t.Fatalf("center toast = %q, want %q", m.toast, want)
	}
	runCmd(t, m, key(m, "0")) // align-left forces mode auto (alignY stays center)
	if want := "unlocked auto pan:0,0 align:start/center"; m.toast != want {
		t.Fatalf("align toast = %q, want %q", m.toast, want)
	}
	runCmd(t, m, key(m, "^")) // align-top: alignY start
	if want := "unlocked auto pan:0,0 align:start/start"; m.toast != want {
		t.Fatalf("align-top toast = %q, want %q", m.toast, want)
	}
	runCmd(t, m, key(m, "A")) // pan-left: PanX -2
	if want := "unlocked auto pan:-2,0 align:start/start"; m.toast != want {
		t.Fatalf("pan toast = %q, want %q", m.toast, want)
	}
	left.locked = true
	runCmd(t, m, key(m, "$"))
	if want := "locked auto pan:-2,0 align:end/start"; m.toast != want {
		t.Fatalf("locked toast = %q, want %q", m.toast, want)
	}
}

// TestOwnerContentLayoutPanShifts pins the corrected owner behavior: the owner
// drives the single PTY via chrome.owner, but its drawn extent CAN shift (pan).
// The terminal box stays full-bleed (so the PTY size is untouched) and the shift
// is expressed as a non-zero content.offset prop; align/center alone (equal-size
// owner) are still no-ops.
func TestOwnerContentLayoutPanShifts(t *testing.T) {
	// ownerSecond=false: the first-bound left pane owns the source.
	m, left, _, _ := twoPaneSourceModel(t, 40, 12, 10, 4, false)
	m.mode = modeResize
	m.focusPaneObject(left)
	content := rect{1, 2, 18, 8}

	// align-right and center are no-ops when the owner extent fills the pane.
	for _, k := range []string{"$", "m"} {
		runCmd(t, m, key(m, k))
		x, y, cols, rows := m.paneContentExtent(left, content)
		if x != 0 || y != 0 || cols != 18 || rows != 8 {
			t.Fatalf("owner %q extent = (%d,%d %dx%d), want no-op (0,0 18x8)", k, x, y, cols, rows)
		}
	}

	// pan-left (PanX -2) shifts the drawn screen right by 2 and reveals dots on
	// the vacated left side; the box stays full-bleed and chrome.owner stays 1.
	runCmd(t, m, key(m, "A"))
	if x, y, cols, rows := m.paneContentExtent(left, content); x != 2 || y != 0 || cols != 18 || rows != 8 {
		t.Fatalf("owner pan-left extent = (%d,%d %dx%d), want (2,0 18x8)", x, y, cols, rows)
	}
	box := terminalBoxOf(t, m, left.id)
	if got := box.GetContent().GetProps()["content.offset"]; got != "2,0" {
		t.Fatalf("owner content.offset = %q, want 2,0", got)
	}
	if got := box.GetContent().GetProps()["content.size"]; got != "18,8" {
		t.Fatalf("owner content.size = %q, want 18,8", got)
	}
	// The box is still the full pane content rect: the PTY size is unaffected.
	if box.GetSize().GetWidth() != 18 || box.GetSize().GetHeight() != 8 {
		t.Fatalf("owner box = %dx%d, want full-bleed 18x8", box.GetSize().GetWidth(), box.GetSize().GetHeight())
	}
	if got := box.GetContent().GetProps()["chrome.owner"]; got != "1" {
		t.Fatalf("owner chrome.owner = %q, want 1", got)
	}
	// The panned owner clips on the right: the border shows the marker.
	lines, _ := m.rasterize(m.View())
	if got := cellAt(lines[9], 19); got != glyphOverflowRight {
		t.Fatalf("owner pan right marker = %q, want %q\n%s", got, glyphOverflowRight, lines[9])
	}
}

// TestLargerExtentPanShiftsOffset pins the larger-than-pane case: a follower
// whose source extent (30x20) exceeds the content rect (18x8) still shifts its
// drawn window on pan instead of only clipping from the top-left.
func TestLargerExtentPanShiftsOffset(t *testing.T) {
	m, left, right, _ := twoPaneSourceModel(t, 40, 12, 30, 20, false)
	_ = left
	m.mode = modeResize
	m.focusPaneObject(right) // right is the follower
	content := rect{21, 2, 18, 8}

	// Default start layout: origin 0,0, footprint 30x20 (clipped right/bottom).
	if x, y, cols, rows := m.paneContentExtent(right, content); x != 0 || y != 0 || cols != 30 || rows != 20 {
		t.Fatalf("start extent = (%d,%d %dx%d), want (0,0 30x20)", x, y, cols, rows)
	}
	box := terminalBoxOf(t, m, right.id)
	if got := box.GetContent().GetProps()["content.offset"]; got != "0,0" {
		t.Fatalf("start content.offset = %q, want 0,0", got)
	}
	if got := box.GetContent().GetProps()["content.size"]; got != "30,20" {
		t.Fatalf("start content.size = %q, want 30,20", got)
	}
	if box.GetSize().GetWidth() != 18 || box.GetSize().GetHeight() != 8 {
		t.Fatalf("follower box = %dx%d, want full-bleed 18x8", box.GetSize().GetWidth(), box.GetSize().GetHeight())
	}

	// pan-left shifts the 30-wide window right by 2 (the right/bottom clip
	// markers stay; the left edge is not yet clipped).
	runCmd(t, m, key(m, "A"))
	box = terminalBoxOf(t, m, right.id)
	if got := box.GetContent().GetProps()["content.offset"]; got != "2,0" {
		t.Fatalf("panned larger extent content.offset = %q, want 2,0", got)
	}
	lines, _ := m.rasterize(m.View())
	if got := cellAt(lines[9], 39); got != glyphOverflowRight {
		t.Fatalf("larger extent right marker = %q, want %q", got, glyphOverflowRight)
	}
	// Pan the other way (two D presses: offset -2,0) clips the left edge. The
	// left marker sits on the top content row's left border (20,2).
	runCmd(t, m, key(m, "D"))
	runCmd(t, m, key(m, "D"))
	box = terminalBoxOf(t, m, right.id)
	if got := box.GetContent().GetProps()["content.offset"]; got != "-2,0" {
		t.Fatalf("right-panned larger extent content.offset = %q, want -2,0", got)
	}
	lines, _ = m.rasterize(m.View())
	if got := cellAt(lines[2], 20); got != glyphOverflowLeft {
		t.Fatalf("larger extent left marker = %q, want %q\n%s", got, glyphOverflowLeft, lines[2])
	}
}

// TestContentLayoutOverflowMarkers pins the four-side clipping markers for a
// panned/centered extent that overflows the pane content area. The follower's
// source extent (30x20) is larger than the 18x8 content rect.
func TestContentLayoutOverflowMarkers(t *testing.T) {
	m, left, right, _ := twoPaneSourceModel(t, 40, 12, 30, 20, false)
	_ = left
	m.mode = modeResize
	m.focusPaneObject(right) // right is the follower (left owns the source)

	// Default start layout clips only the right and bottom edges.
	lines, _ := m.rasterize(m.View())
	if got := cellAt(lines[9], 39); got != glyphOverflowRight {
		t.Fatalf("start right marker = %q, want %q", got, glyphOverflowRight)
	}
	if got := cellAt(lines[10], 38); got != glyphOverflowBottom {
		t.Fatalf("start bottom marker = %q, want %q", got, glyphOverflowBottom)
	}
	if got := cellAt(lines[1], 21); got == glyphOverflowTop {
		t.Fatalf("start must not show a top marker")
	}

	// end/end clips the left and top edges.
	right.layout = contentLayout{mode: layoutModeAuto, alignX: alignEnd, alignY: alignEnd}.normalized()
	lines, _ = m.rasterize(m.View())
	if got := cellAt(lines[2], 20); got != glyphOverflowLeft {
		t.Fatalf("end left marker = %q, want %q\n%s", got, glyphOverflowLeft, lines[2])
	}
	if got := cellAt(lines[1], 21); got != glyphOverflowTop {
		t.Fatalf("end top marker = %q, want %q", got, glyphOverflowTop)
	}
	if got := cellAt(lines[9], 39); got == glyphOverflowRight {
		t.Fatalf("end layout must not show a right marker")
	}

	// center clips all four edges.
	right.layout = contentLayout{mode: layoutModeCenter}.normalized()
	lines, _ = m.rasterize(m.View())
	if got := cellAt(lines[2], 20); got != glyphOverflowLeft {
		t.Fatalf("center left marker = %q, want %q", got, glyphOverflowLeft)
	}
	if got := cellAt(lines[1], 21); got != glyphOverflowTop {
		t.Fatalf("center top marker = %q, want %q", got, glyphOverflowTop)
	}
	if got := cellAt(lines[9], 39); got != glyphOverflowRight {
		t.Fatalf("center right marker = %q, want %q", got, glyphOverflowRight)
	}
	if got := cellAt(lines[10], 38); got != glyphOverflowBottom {
		t.Fatalf("center bottom marker = %q, want %q", got, glyphOverflowBottom)
	}
}

// TestContentOffsetAllFourSides pins the declared offset on all four sides of a
// centered follower extent. The host component renders the dots; the program
// only declares the framing props.
func TestContentOffsetAllFourSides(t *testing.T) {
	m, left, _, _ := twoPaneSourceModel(t, 40, 12, 10, 4, true)
	left.layout = contentLayout{mode: layoutModeCenter}.normalized()
	box := terminalBoxOf(t, m, left.id)
	// Left follower content rect {1,2,18,8}; centered 10x4 footprint => offset
	// (4,2).
	if got := box.GetContent().GetProps()["content.offset"]; got != "4,2" {
		t.Fatalf("content.offset = %q, want 4,2", got)
	}
	if got := box.GetContent().GetProps()["content.size"]; got != "10,4" {
		t.Fatalf("content.size = %q, want 10,4", got)
	}
	if got := box.GetContent().GetProps()["chrome.placeholder"]; got != stExtentPlaceholder {
		t.Fatalf("chrome.placeholder = %q, want %q", got, stExtentPlaceholder)
	}
}

// TestWorkbenchPersistsContentLayout pins the versioned-JSON round trip and
// backward compatibility: a default pane writes no layout field, a custom pane
// restores exactly.
func TestWorkbenchPersistsContentLayout(t *testing.T) {
	m := newModel(nil, false)
	m.host = true
	tab := m.activeTab()
	tab.panes[0].layout = contentLayout{mode: layoutModeCenter, alignX: alignCenter, alignY: alignCenter, panX: 3, panY: -1}

	raw, err := json.Marshal(m.workbenchDoc())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"layout":{"mode":"center","align_x":"center","align_y":"center","pan_x":3,"pan_y":-1}`) {
		t.Fatalf("saved doc missing layout: %s", raw)
	}

	// A default pane must not serialize a layout field (backward compatible).
	m2 := newModel(nil, false)
	raw2, err := json.Marshal(m2.workbenchDoc())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw2), `"layout"`) {
		t.Fatalf("default pane must not persist a layout: %s", raw2)
	}

	// Restore the custom layout.
	var doc workbenchDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	fresh := newModel(nil, false)
	if !fresh.applyWorkbenchDoc(doc) {
		t.Fatal("applyWorkbenchDoc returned false")
	}
	got := fresh.activeTab().panes[0].layout
	if got.mode != layoutModeCenter || got.panX != 3 || got.panY != -1 || got.alignX != alignCenter || got.alignY != alignCenter {
		t.Fatalf("restored layout = %+v, want center/center/center pan:3,-1", got)
	}

	// A legacy document with no layout field restores to the default.
	legacy := `{"version":1,"active_workspace":0,"header_visible":true,"footer_visible":true,` +
		`"workspaces":[{"name":"main","active_tab":0,"tabs":[{"id":"tab-1","title":"t","focus":0,` +
		`"split_seq":0,"panes":[{"id":"pane-1","title":"term"}],"root":{"pane":"pane-1"}}]}]}`
	var legacyDoc workbenchDoc
	if err := json.Unmarshal([]byte(legacy), &legacyDoc); err != nil {
		t.Fatal(err)
	}
	legacyFresh := newModel(nil, false)
	if !legacyFresh.applyWorkbenchDoc(legacyDoc) {
		t.Fatal("legacy apply returned false")
	}
	if got := legacyFresh.activeTab().panes[0].layout; !got.isDefault() {
		t.Fatalf("legacy pane layout = %+v, want default", got)
	}
}

// TestHelpOverlayListsResizeAndCopyKeys pins the discoverability rule: every
// shortcut that has no footer hint slot must appear in the `?` help overlay,
// which is the canonical full list (the legacy buildHelpContent enumerated all
// configured bindings per scene). The footer hint bar stays pinned to the
// Python reference golden, so the resize/copy detail lives here.
func TestHelpOverlayListsResizeAndCopyKeys(t *testing.T) {
	joined := strings.Join(helpLines, "\n")
	for _, want := range []string{
		"RESIZE", "COPY",
		"0/$", "^/B", "A/S/W/D", "shift", "M", "space",
		"j/k", "PgUp/PgDn",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("help overlay must mention %q:\n%s", want, joined)
		}
	}
	// Every help line must fit the 62-wide overlay (60 content cells) so the
	// last key column is never truncated away.
	for _, line := range helpLines {
		if w := sdk.DisplayWidth(line); w > 60 {
			t.Fatalf("help line %q is %d cells wide (>60)", line, w)
		}
	}
}

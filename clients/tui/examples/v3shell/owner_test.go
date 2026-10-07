package main

import (
	"testing"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// twoPaneSourceModel builds a live host model with two panes in the active tab
// bound to the SAME terminal source (the daemon terminal has one extent). It
// returns the model and the two panes; focusSecond selects which pane owns the
// size (ownership is the focused pane of the source).
func twoPaneSourceModel(t *testing.T, cols, rows int, srcCols, srcRows int32, focusSecond bool) (*model, *pane, *pane, *pb.Source) {
	t.Helper()
	m := newModel(nil, false)
	m.host = true
	m.viewID = "view-a"
	m.cols, m.rows = cols, rows
	tab := m.activeTab()
	left := tab.panes[0]
	right := m.splitLeafFor("row", left)
	if right == nil {
		t.Fatal("split failed")
	}
	m.mode = modeLive
	src := &pb.Source{
		Id: "terminal:local:term-1", Kind: "terminal", Title: "term-1",
		Endpoint: "local", TerminalId: "term-1", Attached: true,
		ResizeOwner: "view-a", Cols: srcCols, Rows: srcRows,
	}
	m.sources = []*pb.Source{src}
	left.sourceID = src.GetId()
	right.sourceID = src.GetId()
	if focusSecond {
		m.focusPaneObject(right)
	} else {
		m.focusPaneObject(left)
	}
	return m, left, right, src
}

// TestSourceOwnerFollowUsesSingleOwnerPane pins requirement 1/2: with two panes
// on one source, only the focused pane is the source's owner (green "owner");
// the other pane is a muted "follow" with the take-owner action. Moving focus
// swaps the roles, because ownership is per source, not per view.
func TestSourceOwnerFollowUsesSingleOwnerPane(t *testing.T) {
	m, left, right, src := twoPaneSourceModel(t, 80, 24, 40, 10, true)
	if got := m.sourceOwnerPane(src.GetId()); got != right {
		t.Fatalf("focused owner pane = %v, want right", got)
	}

	text, style, node := m.paneOwner(right)
	if text != "owner" || style != stSuccess || node != "" {
		t.Fatalf("focused pane = %q/%q node=%q, want owner/%q", text, style, node, stSuccess)
	}
	text, style, node = m.paneOwner(left)
	if text != "follow" || style != stMuted {
		t.Fatalf("unfocused pane = %q/%q, want follow/%q", text, style, stMuted)
	}
	if node != "pane:"+left.id+":take-owner" {
		t.Fatalf("follower take-owner node = %q", node)
	}

	// Moving focus swaps ownership to the other pane on the same source.
	m.focusPaneObject(left)
	if got := m.sourceOwnerPane(src.GetId()); got != left {
		t.Fatalf("owner pane after focus move = %v, want left", got)
	}
	if text, style, _ := m.paneOwner(left); text != "owner" || style != stSuccess {
		t.Fatalf("new focused owner = %q/%q, want owner/%q", text, style, stSuccess)
	}
	if text, style, _ := m.paneOwner(right); text != "follow" || style != stMuted {
		t.Fatalf("old owner must become follow = %q/%q", text, style)
	}
}

// TestPaneSizeMismatchOnlyFlagsOwner pins requirement 5: a follower's pane rect
// never matches the terminal extent by design, so it must not project size?;
// only the owner pane reports a mismatch.
func TestPaneSizeMismatchOnlyFlagsOwner(t *testing.T) {
	m, left, right, _ := twoPaneSourceModel(t, 80, 24, 40, 10, true)
	// Right owns: its content rect (58x22 for an 80x24 viewport) != 40x10.
	if !m.paneSizeMismatch(right, 60, 24) {
		t.Fatal("owner pane must expose the terminal/panel mismatch")
	}
	if m.paneSizeMismatch(left, 60, 24) {
		t.Fatal("follower pane must never expose size?")
	}
}

// TestLiveFollowerPaintsExtentPlaceholder pins requirement 3: a live follower
// larger than the source extent draws the terminal box at the owner's size and
// masks the leftover pane content with the dim `·` extent placeholder, while
// the owner pane (full-bleed) never paints a placeholder.
func TestLiveFollowerPaintsExtentPlaceholder(t *testing.T) {
	// 40x12 viewport, row split: each card is 20x10, content 18x8 at x+1,y+2.
	// The source extent is 10x4, so the left follower masks the rest.
	m, _, _, _ := twoPaneSourceModel(t, 40, 12, 10, 4, true)
	lines, styles := m.rasterize(m.View())

	// Left follower content origin (1,2): columns >= 10 on the extent rows and
	// every column below the extent are placeholder dots.
	if got := cellAt(lines[2], 11); got != extentPlaceholder {
		t.Fatalf("right-of-extent cell = %q, want %q\n%s", got, extentPlaceholder, lines[2])
	}
	if got := styles[2][11]; got != dimStyle(stExtentPlaceholder) {
		t.Fatalf("placeholder style = %q, want %q", got, dimStyle(stExtentPlaceholder))
	}
	if got := cellAt(lines[6], 1); got != extentPlaceholder {
		t.Fatalf("below-extent cell = %q, want %q", got, extentPlaceholder)
	}
	// The extent box itself (row 2, col 1) is left to the host terminal, not a
	// dot.
	if got := cellAt(lines[2], 1); got == extentPlaceholder {
		t.Fatalf("extent box cell must not be masked: %q", got)
	}

	// The owner (right card, content origin (21,2)) is full-bleed: no dots.
	if got := cellAt(lines[2], 22); got == extentPlaceholder {
		t.Fatalf("owner pane must not paint a placeholder: %q", got)
	}
	if got := cellAt(lines[6], 21); got == extentPlaceholder {
		t.Fatalf("owner pane must not paint a placeholder: %q", got)
	}
}

// TestLiveFollowerClippedByExtentDrawsOverflowMarkers pins requirement 4: when a
// follower pane is smaller than the terminal extent, the card border gets the
// existing right/bottom overflow markers; the owner pane at its exact extent
// shows none.
func TestLiveFollowerClippedByExtentDrawsOverflowMarkers(t *testing.T) {
	// Extent 30x20 exceeds the 18x8 content area: the right follower is clipped
	// on both edges. Focus the left pane so the right pane is the follower.
	m, left, right, _ := twoPaneSourceModel(t, 40, 12, 30, 20, false)
	lines, styles := m.rasterize(m.View())

	// Follower card rect {20,1,20,10}: right marker on the last content row's
	// right border (39,9), bottom marker before the corner (38,10).
	if got := cellAt(lines[9], 39); got != glyphOverflowRight {
		t.Fatalf("follower right marker = %q, want %q\n%s", got, glyphOverflowRight, lines[9])
	}
	if got := styles[9][39]; got != dimStyle(stOverflowStyle) {
		t.Fatalf("follower right marker style = %q, want %q", got, dimStyle(stOverflowStyle))
	}
	if got := cellAt(lines[10], 38); got != glyphOverflowBottom {
		t.Fatalf("follower bottom marker = %q, want %q", got, glyphOverflowBottom)
	}
	if got := styles[10][38]; got != dimStyle(stOverflowStyle) {
		t.Fatalf("follower bottom marker style = %q, want %q", got, dimStyle(stOverflowStyle))
	}

	// The owner (left card) keeps its plain border: its extent fills the PTY,
	// so it never draws a clipping marker.
	if got := cellAt(lines[9], 19); got != "\u2502" {
		t.Fatalf("owner right border = %q, want plain \u2502", got)
	}
	if got := cellAt(lines[10], 18); got != "\u2500" {
		t.Fatalf("owner bottom border = %q, want plain \u2500", got)
	}
	// A top marker never appears for a live follower (the extent box is
	// top-aligned).
	if got := cellAt(lines[1], 21); got != "\u2500" {
		t.Fatalf("follower top border = %q, want plain \u2500", got)
	}

	_ = left
	_ = right
}

// TestLiveFollowerBoxTracksSourceExtent pins the emitted terminal box: a
// follower renders at the source extent (clipped to the pane) while the owner
// stays full-bleed.
func TestLiveFollowerBoxTracksSourceExtent(t *testing.T) {
	m, left, right, _ := twoPaneSourceModel(t, 40, 12, 10, 4, true)

	boxes := map[string]*pb.Box{}
	var walk func(box *pb.Box)
	walk = func(box *pb.Box) {
		if box.GetId() != "" {
			if _, ok := boxes[box.GetId()]; !ok || box.GetContent().GetSelf() != "" {
				boxes[box.GetId()] = box
			}
		}
		if box.GetContent().GetSelf() != "" {
			boxes[box.GetId()] = box
		}
		for _, child := range box.GetChildren() {
			walk(child)
		}
	}
	walk(m.View())

	ownerBox := boxes[right.id]
	if ownerBox == nil || ownerBox.GetContent().GetSelf() == "" {
		t.Fatalf("owner terminal box missing: %v", boxes)
	}
	if ownerBox.GetSize().GetWidth() != 18 || ownerBox.GetSize().GetHeight() != 8 {
		t.Fatalf("owner box = %dx%d, want full-bleed 18x8", ownerBox.GetSize().GetWidth(), ownerBox.GetSize().GetHeight())
	}
	followerBox := boxes[left.id]
	if followerBox == nil || followerBox.GetContent().GetSelf() == "" {
		t.Fatalf("follower terminal box missing: %v", boxes)
	}
	if followerBox.GetSize().GetWidth() != 10 || followerBox.GetSize().GetHeight() != 4 {
		t.Fatalf("follower box = %dx%d, want extent 10x4", followerBox.GetSize().GetWidth(), followerBox.GetSize().GetHeight())
	}
	if followerBox.GetPos().GetX() != 1 || followerBox.GetPos().GetY() != 2 {
		t.Fatalf("follower box origin = %v, want content origin (1,2)", followerBox.GetPos())
	}
}

// TestDemoSinglePaneSourcesStayOwner verifies the demo chrome is unchanged:
// the focused pane of its single-pane source with ResizeOwner view:demo is the
// green owner, and the ResizeOwner-less opencode source follows.
func TestDemoSinglePaneSourcesStayOwner(t *testing.T) {
	m := demoModel(120, 32)
	left := m.activeTab().panes[0]
	right := m.activeTab().panes[1]
	if text, style, _ := m.paneOwner(left); text != "owner" || style != stSuccess {
		t.Fatalf("demo left pane = %q/%q, want owner/%q", text, style, stSuccess)
	}
	if text, style, node := m.paneOwner(right); text != "follow" || style != stMuted || node == "" {
		t.Fatalf("demo right pane = %q/%q node=%q, want follow/%q", text, style, node, stMuted)
	}
}

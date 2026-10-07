package main

import (
	"testing"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// twoPaneSourceModel builds a live host model with two panes in the active tab
// bound to the SAME terminal source (the daemon terminal has one extent). It
// returns the model and the two panes. Binding the first pane records it as the
// source's owner; ownerSecond simulates a later manual take-owner on the right
// pane. Focus never decides ownership, so it is set independently.
func twoPaneSourceModel(t *testing.T, cols, rows int, srcCols, srcRows int32, ownerSecond bool) (*model, *pane, *pane, *pb.Source) {
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
	m.bindPane(left.id, src.GetId())
	m.bindPane(right.id, src.GetId())
	if ownerSecond {
		// A prior take-owner designated the right pane, independent of focus.
		m.ownerPaneBySource[src.GetId()] = right.id
		m.focusPaneObject(right)
	} else {
		m.focusPaneObject(left)
	}
	return m, left, right, src
}

// ownerPropOf returns the chrome.owner prop of the live terminal box whose id
// is id (the program asserts ownership on exactly that box).
func ownerPropOf(t *testing.T, m *model, id string) string {
	t.Helper()
	return terminalBoxOf(t, m, id).GetContent().GetProps()["chrome.owner"]
}

// terminalBoxOf walks the emitted view for the terminal component box with the
// given id (content.self set), returning it. It is the program-contract probe:
// the offline rasterizer does not run the terminal component, so tests assert
// the declared content.offset/content.size props here.
func terminalBoxOf(t *testing.T, m *model, id string) *pb.Box {
	t.Helper()
	var found *pb.Box
	var walk func(*pb.Box)
	walk = func(b *pb.Box) {
		if b == nil || found != nil {
			return
		}
		if b.GetId() == id && b.GetContent().GetSelf() != "" {
			found = b
			return
		}
		for _, child := range b.GetChildren() {
			walk(child)
		}
	}
	walk(m.View())
	if found == nil {
		t.Fatalf("terminal box %q not found", id)
	}
	return found
}

// TestTakeOwnerIsManual pins the manual ownership rule (the legacy
// panel.take_owner model): with two panes on one source the first bind owns it,
// focusing the follower does NOT transfer ownership, and only an explicit
// take-owner (the PANE `a` command / the badge click) designates the follower.
// The host is told which pane owns the single PTY via chrome.owner=1 on the
// owner box only.
func TestTakeOwnerIsManual(t *testing.T) {
	m, left, right, src := twoPaneSourceModel(t, 80, 24, 40, 10, false)
	if got := m.sourceOwnerPane(src.GetId()); got != left {
		t.Fatalf("first-bound owner = %v, want left", got)
	}

	// Focusing the follower must not move ownership.
	m.focusPaneObject(right)
	if got := m.sourceOwnerPane(src.GetId()); got != left {
		t.Fatalf("focus must not transfer ownership: owner = %v, want left", got)
	}
	if text, style, _ := m.paneOwner(left); text != "owner" || style != stSuccess {
		t.Fatalf("owner pane = %q/%q, want owner/%q", text, style, stSuccess)
	}
	if text, style, node := m.paneOwner(right); text != "follow" || style != stMuted || node != "pane:"+right.id+":take-owner" {
		t.Fatalf("follower pane = %q/%q node=%q, want follow/%q", text, style, node, stMuted)
	}

	// Clicking the follower badge dispatches take-owner and designates it.
	runCmd(t, m, press(m, "pane:"+right.id+":take-owner"))
	if right.pending != "owner" {
		t.Fatalf("take-owner pending = %q, want owner", right.pending)
	}
	if got := m.sourceOwnerPane(src.GetId()); got != right {
		t.Fatalf("owner after take-owner = %v, want right", got)
	}
	right.pending = "" // the host accepted the attach

	if text, style, _ := m.paneOwner(right); text != "owner" || style != stSuccess {
		t.Fatalf("new owner = %q/%q, want owner/%q", text, style, stSuccess)
	}
	if text, style, _ := m.paneOwner(left); text != "follow" || style != stMuted {
		t.Fatalf("old owner must become follow = %q/%q", text, style)
	}

	// Only the owner's box declares chrome.owner=1 to the host.
	if got := ownerPropOf(t, m, right.id); got != "1" {
		t.Fatalf("owner chrome.owner = %q, want 1", got)
	}
	if got := ownerPropOf(t, m, left.id); got != "" {
		t.Fatalf("follower chrome.owner = %q, want unset", got)
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

// TestFollowerDeclaresContentFraming pins the host-path contract: the terminal
// box is always full-bleed (so the PTY size is never shrunk), and the extent is
// declared as content.offset/content.size for the component to draw. The owner
// of an equal-size extent declares the full content framing.
func TestFollowerDeclaresContentFraming(t *testing.T) {
	// 40x12 viewport, row split: each card is 20x10, content 18x8 at x+1,y+2.
	// The source extent is 10x4, so the left pane follows and frames 10x4.
	m, _, _, _ := twoPaneSourceModel(t, 40, 12, 10, 4, true)

	follower := terminalBoxOf(t, m, m.activeTab().panes[0].id)
	if follower.GetSize().GetWidth() != 18 || follower.GetSize().GetHeight() != 8 {
		t.Fatalf("follower box = %dx%d, want full-bleed 18x8", follower.GetSize().GetWidth(), follower.GetSize().GetHeight())
	}
	if got := follower.GetContent().GetProps()["content.offset"]; got != "0,0" {
		t.Fatalf("follower content.offset = %q, want 0,0", got)
	}
	if got := follower.GetContent().GetProps()["content.size"]; got != "10,4" {
		t.Fatalf("follower content.size = %q, want 10,4", got)
	}

	// The owner (right card) has an equal-size extent: full content framing.
	owner := terminalBoxOf(t, m, m.activeTab().panes[1].id)
	if owner.GetSize().GetWidth() != 18 || owner.GetSize().GetHeight() != 8 {
		t.Fatalf("owner box = %dx%d, want full-bleed 18x8", owner.GetSize().GetWidth(), owner.GetSize().GetHeight())
	}
	if got := owner.GetContent().GetProps()["content.size"]; got != "18,8" {
		t.Fatalf("owner content.size = %q, want 18,8", got)
	}
}

// TestLiveFollowerClippedByExtentDrawsOverflowMarkers pins the border clipping
// markers: when a follower's extent is larger than its content area the card
// border gets the right/bottom overflow markers; the owner at its exact extent
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
	// The default start layout is top-aligned: no top marker.
	if got := cellAt(lines[1], 21); got != "\u2500" {
		t.Fatalf("follower top border = %q, want plain \u2500", got)
	}

	_ = left
	_ = right
}

// TestTerminalBoxesStayFullBleed pins the emitted terminal box geometry: both
// the follower and the owner declare the FULL pane content rect, so the owner's
// PTY resize path is measured from the full content rect; the extent difference
// lives only in the content.offset/content.size props.
func TestTerminalBoxesStayFullBleed(t *testing.T) {
	m, left, right, _ := twoPaneSourceModel(t, 40, 12, 10, 4, true)

	ownerBox := terminalBoxOf(t, m, right.id)
	if ownerBox.GetSize().GetWidth() != 18 || ownerBox.GetSize().GetHeight() != 8 {
		t.Fatalf("owner box = %dx%d, want full-bleed 18x8", ownerBox.GetSize().GetWidth(), ownerBox.GetSize().GetHeight())
	}
	followerBox := terminalBoxOf(t, m, left.id)
	if followerBox.GetSize().GetWidth() != 18 || followerBox.GetSize().GetHeight() != 8 {
		t.Fatalf("follower box = %dx%d, want full-bleed 18x8", followerBox.GetSize().GetWidth(), followerBox.GetSize().GetHeight())
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

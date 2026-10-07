package main

import (
	"strings"
	"testing"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// TestPaneOwnerUsesSuccessPendingWarningFollowMuted pins the legacy
// terminalChromeVMFromBinding owner projection: the projected owner (this view
// owns resize) is the green success slot, an acquire-pending view is warning,
// and a follower is muted with a take-owner action.
func TestPaneOwnerUsesSuccessPendingWarningFollowMuted(t *testing.T) {
	m := newModel(nil, false)
	m.viewID = "view-a"
	p := m.focusPane()
	p.sourceID = "terminal:devA:term"
	m.sources = []*pb.Source{{
		Id: p.sourceID, Kind: "terminal", Title: "term", Endpoint: "devA", TerminalId: "term",
		ResizeOwner: "view-a",
	}}

	text, style, node := m.paneOwner(p)
	if text != "owner" || style != stSuccess || node != "" {
		t.Fatalf("projected owner = %q/%q node=%q, want owner/%q", text, style, node, stSuccess)
	}

	// Acquire pending stays the warning slot, even while projected owner.
	p.pending = "owner"
	text, style, _ = m.paneOwner(p)
	if text != "owner?" || style != stWarning {
		t.Fatalf("pending owner = %q/%q, want owner?/%q", text, style, stWarning)
	}
	p.pending = ""

	// A follower keeps the muted follow text and offers take-owner.
	m.sources[0].ResizeOwner = "view-other"
	text, style, node = m.paneOwner(p)
	if text != "follow" || style != stMuted {
		t.Fatalf("follower = %q/%q, want follow/%q", text, style, stMuted)
	}
	if node != "pane:"+p.id+":take-owner" {
		t.Fatalf("follower take-owner node = %q", node)
	}
}

// TestCopyOverflowMarkersOnPaneBorder pins feature B: a frozen copy window
// clipped in a direction draws the legacy arrow on that pane border, styled
// with the `overflow_style` token; offset 0 (the live bottom) has no markers.
func TestCopyOverflowMarkersOnPaneBorder(t *testing.T) {
	m := newModel(nil, false)
	m.cols, m.rows = 120, 32
	p := m.focusPane()
	// offset 3: older rows above the window (top marker) and newer rows below
	// it (bottom marker, not at the live bottom).
	m.copyPanes[p.id] = &copyState{offset: 3, rows: []string{"a", "b", "c", "d"}}
	lines, styles := m.rasterize(m.View())

	// Header row 0; card top border row 1 (the marker is on the second cell);
	// card bottom border row 30, marker on the cell before the corner.
	if got := cellAt(lines[1], 1); got != glyphOverflowTop {
		t.Fatalf("top overflow marker = %q, want %q\n%s", got, glyphOverflowTop, lines[1])
	}
	if got := styles[1][1]; got != stOverflowStyle {
		t.Fatalf("top overflow marker style = %q, want %q", got, stOverflowStyle)
	}
	if got := cellAt(lines[30], 118); got != glyphOverflowBottom {
		t.Fatalf("bottom overflow marker = %q, want %q", got, glyphOverflowBottom)
	}
	if got := styles[30][118]; got != stOverflowStyle {
		t.Fatalf("bottom overflow marker style = %q, want %q", got, stOverflowStyle)
	}
	if got := cellAt(lines[30], 119); got != "\u2518" {
		t.Fatalf("bottom-right corner clobbered by marker: %q", got)
	}

	// At the live bottom (offset 0) nothing is clipped in either direction.
	m.copyPanes[p.id] = &copyState{offset: 0, rows: []string{"a", "b"}}
	lines, _ = m.rasterize(m.View())
	if got := cellAt(lines[1], 1); got == glyphOverflowTop {
		t.Fatalf("top marker must hide at the live bottom")
	}
	if got := cellAt(lines[30], 118); got == glyphOverflowBottom {
		t.Fatalf("bottom marker must hide at the live bottom")
	}
}

// TestCopyRightOverflowMarkerOnPaneBorder pins the horizontal clip case: a
// frozen window row wider than the pane content area draws the right arrow on
// the last content row's right border cell.
func TestCopyRightOverflowMarkerOnPaneBorder(t *testing.T) {
	m := newModel(nil, false)
	m.cols, m.rows = 40, 12
	p := m.focusPane()
	m.copyPanes[p.id] = &copyState{rows: []string{strings.Repeat("x", 60)}}
	lines, styles := m.rasterize(m.View())
	// body y=1, h=10 => card bottom row 10; right border cell x = 40-1 = 39,
	// marker on the last content row (row 9), one above the bottom border.
	if got := cellAt(lines[9], 39); got != glyphOverflowRight {
		t.Fatalf("right overflow marker = %q, want %q", got, glyphOverflowRight)
	}
	if got := styles[9][39]; got != stOverflowStyle {
		t.Fatalf("right overflow marker style = %q, want %q", got, stOverflowStyle)
	}
}

// TestCopyShortWindowPaintsExtentPlaceholder pins feature C: rows the frozen
// window does not cover are filled with the dim extent dots so the pane behind
// never shows through.
func TestCopyShortWindowPaintsExtentPlaceholder(t *testing.T) {
	m := newModel(nil, false)
	m.cols, m.rows = 120, 32
	p := m.focusPane()
	m.copyPanes[p.id] = &copyState{offset: 0, rows: []string{"alpha", "beta"}}
	lines, styles := m.rasterize(m.View())

	// Content rect is y=2..29: the two window rows sit on top and every row
	// below is a placeholder dot.
	if got := cellAt(lines[2], 1); got != "a" {
		t.Fatalf("first window row = %q, want %q", got, "a")
	}
	if got := cellAt(lines[3], 1); got != "b" {
		t.Fatalf("second window row = %q, want %q", got, "b")
	}
	if got := cellAt(lines[4], 1); got != extentPlaceholder {
		t.Fatalf("placeholder cell = %q, want %q", got, extentPlaceholder)
	}
	if got := styles[4][1]; got != stExtentPlaceholder {
		t.Fatalf("placeholder style = %q, want %q", got, stExtentPlaceholder)
	}
	if got := cellAt(lines[29], 40); got != extentPlaceholder {
		t.Fatalf("last content row placeholder = %q, want %q", got, extentPlaceholder)
	}
}

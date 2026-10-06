package main

import (
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk/app"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func TestNewPanelStartsUnconnectedWithActions(t *testing.T) {
	m, fake := boundModel(t)
	fake.calls = nil
	old := m.focusPane()
	runCmd(t, m, m.splitPane("row"))
	p := m.focusPane()
	if p == old || p.sourceID != "" {
		t.Fatalf("new panel = %+v, want an unconnected panel", p)
	}
	if got := len(fake.calls); got != 0 {
		t.Fatalf("split must not create a terminal implicitly: %v", fake.calls)
	}
	lines := screenLines(m)
	joined := ""
	for _, line := range lines {
		joined += line + "\n"
	}
	for _, want := range []string{"No terminal connected", "Attach existing terminal", "Create new terminal", "Close pane"} {
		if !contains(joined, want) {
			t.Fatalf("empty panel missing %q:\n%s", want, joined)
		}
	}
}

func TestEmptyPanelCreateOpensPickerAtCreateRow(t *testing.T) {
	m, _ := boundModel(t)
	runCmd(t, m, m.splitPane("row"))
	p := m.focusPane()
	runCmd(t, m, press(m, "pane:"+p.id+":empty-create"))
	if m.overlay != overlayPicker {
		t.Fatalf("overlay=%q", m.overlay)
	}
	rows := m.pickerRows()
	if len(rows) == 0 || !rows[m.picker].create {
		t.Fatalf("picker=%d rows=%+v", m.picker, rows)
	}
}

func TestEmptyPanelAttachOpensFilteredPicker(t *testing.T) {
	m, _ := boundModel(t)
	runCmd(t, m, m.splitPane("row"))
	p := m.focusPane()
	runCmd(t, m, press(m, "pane:"+p.id+":empty-attach"))
	if m.overlay != overlayPicker || m.picker != 0 {
		t.Fatalf("picker state=%q/%d", m.overlay, m.picker)
	}
	runCmd(t, m, m.onKey("t", "t"))
	if m.pickerQuery != "t" || len(m.pickerRows()) != 2 || !m.pickerRows()[0].create || m.pickerRows()[1].source == nil {
		t.Fatalf("picker query=%q rows=%+v", m.pickerQuery, m.pickerRows())
	}
}

// The focused empty panel owns Up/Down/Enter for its CTA list (main's
// EmptyPaneCTA keyboard selection), not just mouse clicks.
func TestEmptyPanelKeyboardSelection(t *testing.T) {
	m, _ := boundModel(t)
	runCmd(t, m, m.splitPane("row"))
	p := m.focusPane()
	if p.sourceID != "" {
		t.Fatalf("split must leave an empty panel: %+v", p)
	}
	if m.emptyPaneSel != 0 {
		t.Fatalf("default CTA selection = %d, want 0 (Attach)", m.emptyPaneSel)
	}
	runCmd(t, m, key(m, "down"))
	if m.emptyPaneSel != 1 {
		t.Fatalf("down selection = %d, want 1 (Create)", m.emptyPaneSel)
	}
	runCmd(t, m, key(m, "down"))
	runCmd(t, m, key(m, "up"))
	if m.emptyPaneSel != 1 {
		t.Fatalf("up selection = %d, want 1", m.emptyPaneSel)
	}
	// Enter on Create opens the picker at the create row.
	runCmd(t, m, key(m, "enter"))
	if m.overlay != overlayPicker {
		t.Fatalf("enter must open the picker, overlay=%q", m.overlay)
	}
	rows := m.pickerRows()
	if len(rows) == 0 || !rows[m.picker].create {
		t.Fatalf("enter on Create must select the create row: picker=%d rows=%+v", m.picker, rows)
	}
}

// Up/Down on an empty panel must not fall through to global focus changes.
func TestEmptyPanelSelectionWrapsAndDoesNotLeak(t *testing.T) {
	m, _ := boundModel(t)
	runCmd(t, m, m.splitPane("row"))
	runCmd(t, m, key(m, "up"))
	if m.emptyPaneSel != len(emptyPaneActions)-1 {
		t.Fatalf("up must wrap to the last CTA, got %d", m.emptyPaneSel)
	}
	focus := m.focusPane()
	runCmd(t, m, key(m, "down"))
	if m.focusPane() != focus {
		t.Fatal("empty-panel CTA navigation must not move pane focus")
	}
}

func TestEmptyPanelKeepsStructuralSplitActions(t *testing.T) {
	m := newModel(nil, false)
	p := m.focusPane()
	ids := map[string]bool{}
	for _, run := range m.paneRuns(p, true, 80) {
		if run.node != "" {
			ids[run.node] = true
		}
	}
	for _, action := range []string{"split-v", "split-h", "close"} {
		if !ids["pane:"+p.id+":"+action] {
			t.Fatalf("empty panel missing structural action %q: %v", action, ids)
		}
	}
}

func TestEmptyPanelMouseFocusHighlightsBorder(t *testing.T) {
	m := newModel(nil, false)
	first := m.focusPane()
	runCmd(t, m, m.splitPane("row"))
	second := m.focusPane()
	if first == second {
		t.Fatal("split must create a second panel")
	}

	// The empty content area itself is a mouse target, not only the CTA text.
	if !viewHasMouseNode(m.View(), "pane:"+first.id+":focus") || !viewHasMouseNode(m.View(), "pane:"+second.id+":focus") {
		t.Fatal("empty panels must expose full-area focus hit boxes")
	}

	runCmd(t, m, press(m, "pane:"+first.id+":focus"))
	if m.focusPane() != first {
		t.Fatal("clicking an empty panel must move pane focus")
	}
	if got := m.paneRuns(first, true, 80)[0].style; got != stAccent {
		t.Fatalf("focused empty panel border style = %q, want %q", got, stAccent)
	}
	if got := m.paneRuns(second, false, 80)[0].style; got == stAccent {
		t.Fatal("unfocused empty panel must not keep the accent border")
	}

	// Chrome actions on a different empty panel also select it before acting.
	runCmd(t, m, press(m, "pane:"+second.id+":lock"))
	if m.focusPane() != second {
		t.Fatal("clicking an empty panel chrome action must move pane focus")
	}
	if !second.locked {
		t.Fatal("empty panel lock action must still run after focusing")
	}
}

func TestToastRendersInTopRight(t *testing.T) {
	m := newModel(nil, false)
	m.toast = "bound MIX-TERM"
	toast := viewBoxByID(m.View(), "toast")
	if toast == nil {
		t.Fatal("toast node is missing")
	}
	if toast.GetPos().GetY() != 0 {
		t.Fatalf("toast position = %+v, want top row", toast.GetPos())
	}
}

func viewHasMouseNode(root *pb.Box, id string) bool {
	if root == nil {
		return false
	}
	if root.GetId() == id {
		for _, input := range root.GetInput() {
			if input == "mouse" {
				return true
			}
		}
	}
	for _, child := range root.GetChildren() {
		if viewHasMouseNode(child, id) {
			return true
		}
	}
	return false
}

func viewBoxByID(root *pb.Box, id string) *pb.Box {
	if root == nil {
		return nil
	}
	if root.GetId() == id {
		return root
	}
	for _, child := range root.GetChildren() {
		if found := viewBoxByID(child, id); found != nil {
			return found
		}
	}
	return nil
}

func TestOwnerFollowAndSizeMismatchProjection(t *testing.T) {
	m := newModel(nil, false)
	m.viewID = "view-a"
	p := m.focusPane()
	p.sourceID = "terminal:devA:term"
	m.sources = []*pb.Source{{
		Id: p.sourceID, Kind: "terminal", Title: "term", Endpoint: "devA", TerminalId: "term",
		ResizeOwner: "view-other", OwnerEpoch: 7, Cols: 80, Rows: 24,
	}}
	owner, _, action := m.paneOwner(p)
	if owner != "follow" || action == "" {
		t.Fatalf("follower projection = %q action=%q", owner, action)
	}
	m.sources[0].ResizeOwner = "view-a"
	owner, _, action = m.paneOwner(p)
	if owner != "owner" || action != "" {
		t.Fatalf("owner projection = %q action=%q", owner, action)
	}
	if !m.paneSizeMismatch(p, 102, 32) {
		t.Fatal("different terminal/panel sizes must expose a mismatch")
	}
}

func contains(text, want string) bool { return len(text) >= len(want) && indexOf(text, want) >= 0 }
func indexOf(text, want string) int {
	for i := 0; i+len(want) <= len(text); i++ {
		if text[i:i+len(want)] == want {
			return i
		}
	}
	return -1
}

// Endpoint health is represented by the picker's endpoint tabs, so a route
// failure must not leave a persistent toast covering the footer. Protocol and
// crash notices must still surface.
func TestEndpointHealthNoticeIsNotAToast(t *testing.T) {
	m := newModel(nil, false)
	m.Update(app.NoticeMsg{Level: "warning", Message: `endpoint device-V55ROrkr7rSJuXEYm8V0Lg offline: route "cloud" (managed-webrtc) failed; check this route's configuration or choose another configured route: Cloud connection failed (RPC NotFound): cached`})
	if m.toast != "" {
		t.Fatalf("endpoint route failure must not become a toast: %q", m.toast)
	}
	m.Update(app.NoticeMsg{Level: "info", Message: "endpoint local connected"})
	if m.toast != "" {
		t.Fatalf("endpoint connect notice must not become a toast: %q", m.toast)
	}
	m.Update(app.NoticeMsg{Level: "warning", Message: "protocol error: bad frame"})
	if m.toast == "" {
		t.Fatal("protocol warnings must still surface")
	}
}

// main defaults the picker to Running and cycles Running -> Exited -> All on
// Shift+Left/Right (terminal_picker.status_previous/next).
func TestPickerStatusDefaultsToRunningAndCycles(t *testing.T) {
	m, _ := boundModel(t)
	m.sources = append(m.sources,
		&pb.Source{Id: "terminal:local:run", Kind: "terminal", Title: "run", Endpoint: "local", TerminalId: "run"},
		&pb.Source{Id: "terminal:local:dead", Kind: "terminal", Title: "dead", Endpoint: "local", TerminalId: "dead", Exited: true},
	)
	runCmd(t, m, key(m, "ctrl-f"))
	if m.pickerFilter != pickerFilterRunning {
		t.Fatalf("default filter = %d, want Running", m.pickerFilter)
	}
	runCmd(t, m, m.onKey("shift-right", ""))
	if m.pickerFilter != pickerFilterExited {
		t.Fatalf("shift-right filter = %d, want Exited", m.pickerFilter)
	}
	rows := m.pickerRows()
	if len(rows) != 1 || rows[0].source.GetTerminalId() != "dead" {
		t.Fatalf("Exited filter rows = %+v", rows)
	}
	runCmd(t, m, m.onKey("shift-right", ""))
	if m.pickerFilter != pickerFilterAll || len(m.pickerRows()) != 4 {
		t.Fatalf("All filter = %d rows=%+v", m.pickerFilter, m.pickerRows())
	}
	runCmd(t, m, m.onKey("shift-left", ""))
	if m.pickerFilter != pickerFilterExited {
		t.Fatalf("shift-left filter = %d, want Exited", m.pickerFilter)
	}
}

// Ctrl-T opens a tag checkbox list; Space toggles the highlighted tag and the
// list view filters terminals by every selected tag.
func TestPickerTagsCheckboxFilters(t *testing.T) {
	m, _ := boundModel(t)
	m.sources = append(m.sources,
		&pb.Source{Id: "terminal:local:a", Kind: "terminal", Title: "a", Endpoint: "local", TerminalId: "a", Tags: map[string]string{"tag1": "backend"}},
		&pb.Source{Id: "terminal:local:b", Kind: "terminal", Title: "b", Endpoint: "local", TerminalId: "b", Tags: map[string]string{"tag1": "frontend"}},
	)
	runCmd(t, m, key(m, "ctrl-f"))
	m.pickerFilter = pickerFilterAll
	runCmd(t, m, m.onKey("ctrl-t", ""))
	if !m.pickerTagsOpen {
		t.Fatal("ctrl-t must open the tag list")
	}
	options := m.pickerTagOptions()
	if len(options) != 2 || options[0].label != "backend" || options[1].label != "frontend" {
		t.Fatalf("tag options = %+v", options)
	}
	runCmd(t, m, m.onKey("space", ""))
	if len(m.pickerTags) != 1 || m.pickerTags[0] != "backend" {
		t.Fatalf("selected tags = %v", m.pickerTags)
	}
	runCmd(t, m, m.onKey("ctrl-t", ""))
	rows := m.pickerRows()
	if len(rows) != 2 || rows[1].source.GetTerminalId() != "a" {
		t.Fatalf("tag-filtered rows = %+v", rows)
	}
}

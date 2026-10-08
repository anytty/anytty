package main

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk/app"
	apipb "github.com/anytty/anytty/proto/access/apipb"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
	gproto "google.golang.org/protobuf/proto"
)

// fakeEmitter records RESULT calls and answers them synchronously.
type fakeEmitter struct {
	calls  []emitCall
	answer func(method string, params *pb.MethodParams) *pb.Response
}

type emitCall struct {
	method string
	params *pb.MethodParams
}

func (f *fakeEmitter) Emit(method string, params *pb.MethodParams, onResponse func(*pb.Response)) (uint64, error) {
	f.calls = append(f.calls, emitCall{method: method, params: params})
	if onResponse != nil && f.answer != nil {
		if resp := f.answer(method, params); resp != nil {
			onResponse(resp)
		}
	}
	return uint64(len(f.calls)), nil
}

func (f *fakeEmitter) last() emitCall {
	if len(f.calls) == 0 {
		return emitCall{}
	}
	return f.calls[len(f.calls)-1]
}

// runCmd executes one command tree and feeds every resulting message back
// into the model (the app.Run loop, synchronous for tests).
func runCmd(t *testing.T, m *model, cmd app.Cmd) {
	t.Helper()
	for _, msg := range app.RunCmd(cmd) {
		if next := m.Update(msg); next != nil {
			runCmd(t, m, next)
		}
	}
}

func key(m *model, name string) app.Cmd { return m.onKey(name, "") }

func press(m *model, node string) app.Cmd {
	return m.onMouse(&pb.MouseEvent{Action: "press", Node: node})
}

func okResponse(endpoint, id string) *pb.Response {
	return &pb.Response{Ok: true, Data: &pb.MethodData{Endpoint: endpoint, Id: id}}
}

func failResponse(errText string) *pb.Response {
	return &pb.Response{Ok: false, Error: errText}
}

func boundModel(t *testing.T) (*model, *fakeEmitter) {
	t.Helper()
	m := newModel(nil, false)
	fake := &fakeEmitter{answer: func(method string, params *pb.MethodParams) *pb.Response {
		if method == "access.call" {
			payload, err := gproto.Marshal(&apipb.ResultEnvelope{
				Result: &apipb.ResultEnvelope_TerminalDefaults{TerminalDefaults: &apipb.TerminalDefaultsResult{
					Defaults: &apipb.TerminalDefaults{DefaultCommand: []string{"/bin/sh"}, DefaultCwd: "/"},
				}},
			})
			if err != nil {
				t.Fatalf("marshal defaults: %v", err)
			}
			return &pb.Response{Ok: true, Data: &pb.MethodData{AccessResult: payload}}
		}
		return okResponse("local", "term-1")
	}}
	m.client = fake
	m.host = true
	p := m.focusPane()
	m.sources = append(m.sources, &pb.Source{
		Id: "terminal:local:term-1", Kind: "terminal", Title: "term-1",
		Endpoint: "local", TerminalId: "term-1", Attached: true,
	})
	runCmd(t, m, m.bindPending(p.id, "", &pb.MethodParams{Endpoint: "local"}))
	if p.sourceID != "terminal:local:term-1" {
		t.Fatalf("pane not bound: %q", p.sourceID)
	}
	return m, fake
}

func TestClaimIsModal(t *testing.T) {
	m := newModel(nil, false)
	if m.claim().All {
		t.Fatalf("live claim must list chords, not All")
	}
	m.mode = modePane
	if !m.claim().All {
		t.Fatalf("pane mode must claim all keys")
	}
	m.mode = modeLive
	m.overlay = overlayPicker
	if !m.claim().All {
		t.Fatalf("overlay must claim all keys")
	}
	m.overlay = ""
	m.copyPanes[m.focusPane().id] = &copyState{}
	if !m.claim().All {
		t.Fatalf("copy mode must claim all keys")
	}
}

func TestShortcutLockLeavesOnlyUnlockChordClaimed(t *testing.T) {
	m := newModel(nil, false)
	m.mode = modeSystem
	runCmd(t, m, key(m, "l"))
	if !m.shortcutLocked || m.mode != modeLive {
		t.Fatalf("shortcut lock state = %v mode=%q", m.shortcutLocked, m.mode)
	}
	claim := m.claim()
	if claim.All || len(claim.Claim) != 1 || claim.Claim[0] != "ctrl-g" {
		t.Fatalf("locked claim = %+v", claim)
	}
	m.mode = modeSystem
	runCmd(t, m, key(m, "l"))
	if m.shortcutLocked {
		t.Fatal("shortcut lock did not toggle off")
	}
}

func TestClipboardPasteUsesHostMethod(t *testing.T) {
	m, fake := boundModel(t)
	fake.calls = nil
	runCmd(t, m, key(m, "ctrl-shift-v"))
	call := fake.last()
	if call.method != "clipboard.paste" {
		t.Fatalf("paste method = %q, want clipboard.paste", call.method)
	}
	if call.params.GetEndpoint() != "local" || call.params.GetId() != "term-1" {
		t.Fatalf("paste target = %+v", call.params)
	}
}

func TestTerminalRenameUsesHostMethod(t *testing.T) {
	m, fake := boundModel(t)
	src := m.sourceByID(m.focusPane().sourceID)
	m.openRename("terminal.rename", src.GetId(), src.GetTitle())
	m.prompt = "renamed"
	m.promptCursor = len([]rune(m.prompt))
	fake.calls = nil
	runCmd(t, m, key(m, "enter"))
	call := fake.last()
	if call.method != "terminal.rename" || call.params.GetTitle() != "renamed" {
		t.Fatalf("rename call = %+v", call)
	}
}

func TestDetachedPaneCanReconnectFromItsSavedSource(t *testing.T) {
	m, fake := boundModel(t)
	p := m.focusPane()
	m.mode = modePane
	fake.calls = nil
	runCmd(t, m, key(m, "d"))
	if p.sourceID != "" || p.detachedSourceID != "terminal:local:term-1" {
		t.Fatalf("detach source state = source=%q detached=%q", p.sourceID, p.detachedSourceID)
	}
	fake.calls = nil
	runCmd(t, m, key(m, "r"))
	call := fake.last()
	if call.method != "terminal.reconnect" || call.params.GetEndpoint() != "local" || call.params.GetId() != "term-1" {
		t.Fatalf("reconnect call = %+v", call)
	}
	if p.sourceID != "terminal:local:term-1" || p.detachedSourceID != "" {
		t.Fatalf("reconnect source state = source=%q detached=%q", p.sourceID, p.detachedSourceID)
	}
}

func TestSplitTreeOnlySplitsFocusedLeaf(t *testing.T) {
	m := demoModel(120, 32)
	tab := m.activeTab()
	m.closePane(tab, tab.panes[1])
	runCmd(t, m, m.splitPane("row"))
	if len(tab.panes) != 2 {
		t.Fatalf("panes=%d, want 2", len(tab.panes))
	}
	left := tab.panes[0]
	// Focus is the new right leaf; split it column-wise.
	runCmd(t, m, m.splitPane("col"))
	if len(tab.panes) != 3 {
		t.Fatalf("panes=%d, want 3", len(tab.panes))
	}
	_, entries := m.paneRects(tab)
	var rects []rect
	for _, entry := range entries {
		if entry.pane != nil {
			rects = append(rects, entry.r)
		}
	}
	if len(rects) != 3 {
		t.Fatalf("leaf rects=%v", rects)
	}
	if rects[0].h != 30 {
		t.Fatalf("left leaf must stay full height: %+v", rects[0])
	}
	if rects[1].h+rects[2].h != 30 {
		t.Fatalf("right column must stack inside the left height: %+v", rects)
	}
	// Closing the focused leaf promotes its sibling.
	focused := m.focusPane()
	m.closePane(tab, focused)
	if len(tab.panes) != 2 {
		t.Fatalf("after close panes=%d, want 2", len(tab.panes))
	}
	if tab.panes[0] != left {
		t.Fatalf("left leaf must survive")
	}
}

func TestDividerDragWritesOneRatio(t *testing.T) {
	m := demoModel(120, 32)
	tab := m.activeTab()
	m.closePane(tab, tab.panes[1])
	runCmd(t, m, m.splitPane("row"))
	splits := m.splitEntries(tab)
	if len(splits) != 1 {
		t.Fatalf("splits=%d", len(splits))
	}
	sp := splits[0].node
	before := sp.ratio
	boundary := splits[0].boundary
	m.onMouse(&pb.MouseEvent{Action: "press", Node: "divider:" + tab.id + ":" + itoa(sp.seq)})
	m.onMouse(&pb.MouseEvent{Action: "drag", X: int32(boundary.x + 11), Y: int32(boundary.y + 1)})
	if sp.ratio == before {
		t.Fatalf("drag did not change the split ratio")
	}
	m.onMouse(&pb.MouseEvent{Action: "release"})
	if m.dragging != "" {
		t.Fatalf("release must end the capture")
	}
}

func TestZoomFillsBodyAndKeepsTerminal(t *testing.T) {
	m := demoModel(120, 32)
	tab := m.activeTab()
	m.closePane(tab, tab.panes[1])
	runCmd(t, m, m.splitPane("row"))
	left := tab.panes[0]
	runCmd(t, m, m.splitPane("col"))
	if m.zoomPane != "" {
		t.Fatalf("unexpected zoom")
	}
	// Focus the left leaf and zoom it.
	m.focusPaneObject(left)
	runCmd(t, m, key(m, "z"))
	if m.zoomPane != left.id {
		t.Fatalf("zoom=%q, want %q", m.zoomPane, left.id)
	}
	_, entries := m.paneRects(tab)
	if len(entries) != 1 || entries[0].pane != left {
		t.Fatalf("zoomed layout must show only the zoomed leaf: %+v", entries)
	}
	body := m.bodyRect()
	if entries[0].r != body {
		t.Fatalf("zoomed leaf rect %+v, want body %+v", entries[0].r, body)
	}
	// The zoom action glyph switches to unzoom.
	runs := m.paneRuns(left, true, 60)
	found := false
	for _, run := range runs {
		if run.node == "pane:"+left.id+":zoom" {
			found = run.text == " \u2199 "
		}
	}
	if !found {
		t.Fatalf("zoomed pane must show the unzoom glyph")
	}
	runCmd(t, m, key(m, "z"))
	if m.zoomPane != "" {
		t.Fatalf("second z must unzoom")
	}
}

func TestFloatingLifecycle(t *testing.T) {
	m := demoModel(120, 32)
	runCmd(t, m, m.newFloating())
	if len(m.floatings) != 1 {
		t.Fatalf("floatings=%d", len(m.floatings))
	}
	f := m.floatings[0]
	if m.activeFloat != f.id {
		t.Fatalf("new floating must become active")
	}
	m.mode = modeFloating
	// Move by keyboard and drag.
	x0 := f.x
	runCmd(t, m, key(m, "l"))
	if f.x != x0+2 {
		t.Fatalf("float move right: %d -> %d", x0, f.x)
	}
	m.onMouse(&pb.MouseEvent{Action: "press", Node: "float:" + f.id + ":title", X: int32(f.x + 2), Y: int32(f.y + 1)})
	m.onMouse(&pb.MouseEvent{Action: "drag", X: int32(f.x + 5), Y: int32(f.y + 3)})
	if f.x != x0+5 {
		t.Fatalf("drag must move the window: %d", f.x)
	}
	m.onMouse(&pb.MouseEvent{Action: "release"})
	// Collapse keeps only the title row.
	runCmd(t, m, key(m, "z"))
	if !f.collapsed {
		t.Fatalf("z must collapse")
	}
	lines := screenLines(m)
	if cellAt(lines[f.y+1], f.x) == "\u2502" {
		t.Fatalf("collapsed floating must not draw a body")
	}
	runCmd(t, m, key(m, "z"))
	if f.collapsed {
		t.Fatalf("second z must expand")
	}
	runCmd(t, m, key(m, "x"))
	if len(m.floatings) != 0 || m.mode != modeLive {
		t.Fatalf("x must close the floating and return to live")
	}
}

func TestResizeModeChangesNearestSplit(t *testing.T) {
	m := demoModel(120, 32)
	tab := m.activeTab()
	m.closePane(tab, tab.panes[1])
	runCmd(t, m, m.splitPane("row"))
	sp := m.splitEntries(tab)[0].node
	before := sp.splitFirstExtent(sp.rect.w)
	m.mode = modeResize
	// The focused pane is the second child; h moves the divider left, so the
	// first child's extent grows by the bias (2).
	runCmd(t, m, key(m, "h"))
	after := sp.splitFirstExtent(sp.rect.w)
	if after != before+2 {
		t.Fatalf("h bias = first extent %d -> %d, want %d", before, after, before+2)
	}
	runCmd(t, m, key(m, "r"))
	if sp.bias != 0 || sp.ratio != 0.5 {
		t.Fatalf("r must reset the split: ratio=%v bias=%d", sp.ratio, sp.bias)
	}
	runCmd(t, m, key(m, "space"))
	if sp.orient != "col" {
		t.Fatalf("space must toggle the layout orientation: %q", sp.orient)
	}
}

func TestWorkspacesKeepTheirOwnTabs(t *testing.T) {
	m := demoModel(120, 32)
	m.mode = modeWorkspace
	runCmd(t, m, key(m, "c"))
	if len(m.spaces) != 2 || m.ws().name != "ws-2" {
		t.Fatalf("workspace create failed: %+v", m.spaces)
	}
	if len(m.ws().tabs) != 1 {
		t.Fatalf("new workspace must own one tab")
	}
	m.space = 0
	if len(m.ws().tabs) != 2 {
		t.Fatalf("first workspace must keep its two tabs")
	}
	m.space = 1
	runCmd(t, m, key(m, "x"))
	if len(m.spaces) != 1 {
		t.Fatalf("workspace delete failed")
	}
}

func TestPromptFiltersAndRunsCommands(t *testing.T) {
	m := demoModel(120, 32)
	m.mode = modeSystem
	runCmd(t, m, key(m, "o"))
	if m.overlay != overlayPrompt {
		t.Fatalf("system o must open the prompt")
	}
	for _, r := range "split" {
		runCmd(t, m, m.onKey(string(r), string(r)))
	}
	if got := m.promptMatches(); len(got) != 2 {
		t.Fatalf("matches=%v, want split row/col", got)
	}
	tab := m.activeTab()
	before := len(tab.panes)
	runCmd(t, m, m.onKey("enter", ""))
	if len(tab.panes) != before+1 {
		t.Fatalf("prompt command must split: %d -> %d", before, len(tab.panes))
	}
}

func TestPromptEditingUsesCursorAndPaste(t *testing.T) {
	m := demoModel(120, 32)
	m.openRename("tab", m.activeTab().id, "document")
	runCmd(t, m, key(m, "left"))
	runCmd(t, m, key(m, "left"))
	runCmd(t, m, key(m, "left"))
	runCmd(t, m, key(m, "left"))
	runCmd(t, m, m.onKey("X", "X"))
	if m.prompt != "docuXment" || m.promptCursor != 5 {
		t.Fatalf("insert at cursor = %q cursor=%d", m.prompt, m.promptCursor)
	}
	runCmd(t, m, m.Update(app.PasteMsg{Paste: &pb.PasteEvent{Text: "!"}}))
	if m.prompt != "docuX!ment" || m.promptCursor != 6 {
		t.Fatalf("paste at cursor = %q cursor=%d", m.prompt, m.promptCursor)
	}
	runCmd(t, m, m.onKey("delete", ""))
	if m.prompt != "docuX!ent" {
		t.Fatalf("delete at cursor = %q", m.prompt)
	}
}

func TestCommandPromptRowsHighlightSelectedCommand(t *testing.T) {
	m := demoModel(120, 32)
	m.openPrompt()
	m.prompt = "split"
	rows := m.overlayRows()
	if len(rows) != 3 || rows[0].selectable || !rows[1].selectable || !rows[2].selectable {
		t.Fatalf("command prompt rows must expose selectable matches: %+v", rows)
	}
	m.promptSel = 1
	if got := m.promptMatches()[m.promptSel]; got != "split col" {
		t.Fatalf("selected command = %q", got)
	}
}

func TestPickerUsesSubsequenceSearchEndToEnd(t *testing.T) {
	m := newModel(nil, false)
	m.sources = []*pb.Source{{
		Id: "terminal:local:document", Kind: "terminal", Title: "document",
		Endpoint: "local", TerminalId: "document",
	}}
	m.sourcesReady = true
	m.terminalCount = 1
	m.openPicker()
	m.pickerQuery = "DCT"
	rows := m.pickerRows()
	if len(rows) != 1 || rows[0].source == nil || rows[0].source.GetTitle() != "document" {
		t.Fatalf("DCT should match document: %+v", rows)
	}
}

func TestClipboardOverlayRowsExposeSelection(t *testing.T) {
	m := demoModel(120, 32)
	m.overlay = overlayClipboard
	m.clipboard = []string{"first", "second"}
	rows := m.overlayRows()
	if len(rows) != 2 || !rows[0].selectable || !rows[1].selectable {
		t.Fatalf("clipboard rows must be selectable: %+v", rows)
	}
	m.clipSel = 1
	if rows := m.overlayRows(); rows[1].runs[0].text != "[1] second" {
		t.Fatalf("clipboard row text changed unexpectedly: %+v", rows[1])
	}
}

func TestRenameTabAppliesPrompt(t *testing.T) {
	m := demoModel(120, 32)
	m.mode = modeTab
	runCmd(t, m, key(m, "r"))
	if m.overlay != overlayPrompt || m.promptKind != "rename" {
		t.Fatalf("r must open the rename prompt")
	}
	m.prompt = "renamed"
	runCmd(t, m, m.onKey("enter", ""))
	if m.activeTab().title != "renamed" {
		t.Fatalf("rename not applied: %q", m.activeTab().title)
	}
}

func TestHeaderFooterToggles(t *testing.T) {
	m := demoModel(120, 32)
	body := m.bodyRect()
	m.mode = modeSystem
	runCmd(t, m, key(m, "h"))
	if m.headerVisible {
		t.Fatalf("h must hide the header")
	}
	if got := m.bodyRect(); got.y != 0 || got.h != body.h+1 {
		t.Fatalf("body must grow when the header hides: %+v", got)
	}
	runCmd(t, m, key(m, "f"))
	if m.footerVisible {
		t.Fatalf("f must hide the footer")
	}
	if got := m.bodyRect(); got.h != body.h+2 {
		t.Fatalf("body must grow when the footer hides: %+v", got)
	}
}

func TestCopyModeScrollsCopiesAndExits(t *testing.T) {
	window := []string{"line-0", "line-1", "line-2", "line-3", "line-4",
		"line-5", "line-6", "line-7", "line-8", "line-9"}
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "history.window" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: window, Offset: 0}}
		}
		if method == "terminal.scroll" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: window, Offset: 1}}
		}
		return &pb.Response{Ok: true}
	}
	runCmd(t, m, key(m, "ctrl-shift-c"))
	if !m.copyActive() {
		t.Fatalf("copy mode must track the focused pane")
	}
	st := m.copyFor(m.focusPane())
	if st == nil || len(st.rows) != len(window) {
		t.Fatalf("copy session must load the window: %+v", st)
	}
	if st.cursorRow != len(window)-1 {
		t.Fatalf("copy must enter at the newest row: %d", st.cursorRow)
	}
	// Paging uses ViewRows-2: from the bottom row 9, page-up lands on row 1
	// without scrolling (the viewport follows the cursor only at the edges).
	runCmd(t, m, key(m, "page-up"))
	if st.cursorRow != 1 {
		t.Fatalf("page-up cursor = %d, want 1", st.cursorRow)
	}
	// Mark the whole row and copy it (linear indices: row 1 * width 6).
	runCmd(t, m, key(m, "home"))
	runCmd(t, m, key(m, "space"))
	runCmd(t, m, key(m, "end"))
	fake.calls = nil
	runCmd(t, m, key(m, "y"))
	if fake.last().method != "terminal.copy" {
		t.Fatalf("y must copy the selection: %+v", fake.calls)
	}
	sel := fake.last().params.GetSel()
	if sel.GetMode() != "char" || sel.GetStart() != 6 || sel.GetEnd() != 11 {
		t.Fatalf("selection = %+v, want char 6..11", sel)
	}
	if !m.copyActive() {
		t.Fatalf("y must keep copy mode open")
	}
	// Enter copies and leaves copy mode.
	fake.calls = nil
	runCmd(t, m, key(m, "enter"))
	if !hasCall(fake.calls, "terminal.copy") {
		t.Fatalf("enter must copy: %+v", fake.calls)
	}
	if m.copyActive() {
		t.Fatalf("enter must leave copy mode")
	}
	// Scrolling back to the bottom auto-exits (the old AtFrozenBottom rule).
	runCmd(t, m, key(m, "ctrl-shift-c"))
	runCmd(t, m, key(m, "page-up"))
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "terminal.scroll" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: window, Offset: 0}}
		}
		return &pb.Response{Ok: true}
	}
	runCmd(t, m, key(m, "page-down"))
	if m.copyActive() {
		t.Fatalf("reaching the bottom must close the copy session")
	}
}

func hasCall(calls []emitCall, method string) bool {
	for _, call := range calls {
		if call.method == method {
			return true
		}
	}
	return false
}

// TestLateScrollResponseCannotReverseCopyViewport pins the bottom anti-jitter
// rule for the persistent (asynchronous) history provider: when the user
// reverses the wheel before the previous terminal.scroll response arrives, the
// stale response must be dropped instead of moving the viewport back.
func TestLateScrollResponseCannotReverseCopyViewport(t *testing.T) {
	m, _ := boundModel(t)
	p := m.focusPane()
	st := &copyState{rows: make([]string, 64), viewRows: 20, offset: 4, cols: 80, placed: true}
	st.scrollSeq = 2 // the newer (downward) request is the current one
	m.copyPanes[p.id] = st

	// A late response from the older (upward) request (seq 1) must not apply.
	m.onOp(opMsg{op: "scroll", ref: p.id, seq: 1, delta: 1, offset: 6, rows: st.rows, ok: true})
	if st.offset != 4 {
		t.Fatalf("late upward response reversed the viewport: offset=%d, want 4", st.offset)
	}

	// The current response still applies normally.
	m.onOp(opMsg{op: "scroll", ref: p.id, seq: 2, delta: -1, offset: 2, rows: st.rows, ok: true})
	if st.offset != 2 {
		t.Fatalf("current response dropped: offset=%d, want 2", st.offset)
	}
}

func TestCopySearchSelectsAndHighlights(t *testing.T) {
	m, fake := boundModel(t)
	searches := 0
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "terminal.search" {
			searches++
			start, wrapped := int32(4), true
			if searches == 2 {
				start, wrapped = int32(m.copyFor(m.focusPane()).cols), false
			}
			return &pb.Response{Ok: true, Data: &pb.MethodData{Found: true, Wrapped: wrapped,
				Rows: []string{"one two", "two three", "four"}, MatchStart: start, MatchEnd: start + 3}}
		}
		if method == "history.window" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{
				Rows: []string{"one two", "two three", "four"}, Offset: 0,
			}}
		}
		return &pb.Response{Ok: true}
	}
	runCmd(t, m, key(m, "ctrl-shift-c"))
	st := m.copyFor(m.focusPane())
	if st == nil {
		t.Fatalf("no copy session")
	}
	st.cols = 80
	// Type a query and run it.
	runCmd(t, m, key(m, "/"))
	for _, r := range "two" {
		runCmd(t, m, m.onKey(string(r), string(r)))
	}
	runCmd(t, m, m.onKey("enter", ""))
	if st.query != "two" || st.searching {
		t.Fatalf("search state = %+v", st)
	}
	if len(st.matches) != 2 {
		t.Fatalf("visible matches = %+v, want 2", st.matches)
	}
	// The cursor enters at the bottom, so the forward search wraps to the
	// oldest match ("two" on row 0 at col 4).
	if st.cursorRow != 0 || st.cursorCol != 4 {
		t.Fatalf("search cursor = (%d,%d), want (0,4)", st.cursorRow, st.cursorCol)
	}
	props := m.copyProps(st)
	if props["copy.match_current"] == "" || props["copy.sel"] != "" {
		t.Fatalf("props = %+v", props)
	}
	// n moves to the next match below the cursor (row 1, col 0).
	runCmd(t, m, key(m, "n"))
	if st.cursorRow != 1 || st.cursorCol != 0 {
		t.Fatalf("next match cursor = (%d,%d), want (1,0)", st.cursorRow, st.cursorCol)
	}
	// The selection spans the marked stream.
	runCmd(t, m, key(m, "space"))
	runCmd(t, m, key(m, "l"))
	spans := m.selectionSpans(st)
	if len(spans) != 1 || spans[0].startCol != 0 || spans[0].endCol != 1 {
		t.Fatalf("spans = %+v", spans)
	}
}

func TestRestartAndKillFlows(t *testing.T) {
	m, fake := boundModel(t)
	p := m.focusPane()
	fake.calls = nil
	runCmd(t, m, m.restartPane(p))
	if fake.last().method != "terminal.restart" {
		t.Fatalf("restart emit: %+v", fake.calls)
	}
	if p.pending != "" {
		t.Fatalf("response must clear the restart pending flag")
	}
	fake.calls = nil
	runCmd(t, m, m.killPane(p))
	if fake.last().method != "terminal.kill" {
		t.Fatalf("kill emit: %+v", fake.calls)
	}
}

func TestPickerAttachAndCreate(t *testing.T) {
	m, fake := boundModel(t)
	m.sources = append(m.sources, &pb.Source{
		Id: "terminal:local:term-9", Kind: "terminal", Title: "term-9",
		Endpoint: "local", TerminalId: "term-9", Attached: false,
	})
	m.mode = modeLive
	runCmd(t, m, key(m, "ctrl-f"))
	if m.overlay != overlayPicker {
		t.Fatalf("ctrl-f must open the picker")
	}
	m.picker = 2 // rows: + New terminal, term-1, term-9 (sorted)
	fake.calls = nil
	runCmd(t, m, m.onKey("enter", ""))
	if fake.last().method != "terminal.attach" || fake.last().params.GetId() != "term-9" {
		t.Fatalf("enter must attach the selected terminal: %+v", fake.calls)
	}
	if fake.last().params.GetFit() != true {
		t.Fatalf("attach must request fit=true")
	}
	// The first picker row creates a new terminal on the selected endpoint.
	m.overlay = overlayPicker
	m.picker = 0
	fake.calls = nil
	runCmd(t, m, m.onKey("enter", ""))
	if m.overlay != overlayPrompt || m.promptKind != "terminal.create" {
		t.Fatalf("create row must open terminal.create prompt: overlay=%q kind=%q calls=%+v", m.overlay, m.promptKind, fake.calls)
	}
	m.promptFields[0] = "term-new"
	m.ensurePromptCursors()
	runCmd(t, m, m.onKey("enter", ""))
	if fake.last().method != "terminal.create" || fake.last().params.GetEndpoint() != "local" {
		t.Fatalf("submitted create endpoint=%q calls=%+v", fake.last().params.GetEndpoint(), fake.calls)
	}
}

// TestPickerPartitionsByEndpoint pins the old picker's machine partitions:
// one tab per endpoint, rows filtered by the selected tab, left/right switch.
func TestPickerPartitionsByEndpoint(t *testing.T) {
	m, fake := boundModel(t)
	m.sources = append(m.sources,
		&pb.Source{Id: "terminal:hs:term-hs", Kind: "terminal", Title: "term-hs",
			Endpoint: "hs", TerminalId: "term-hs"},
		&pb.Source{Id: "endpoint:hs", Kind: "endpoint", Title: "hs-box",
			Endpoint: "hs", Health: "offline"},
	)
	m.mode = modeLive
	runCmd(t, m, key(m, "ctrl-f"))
	tabs := m.pickerTabs()
	if len(tabs) != 2 || tabs[0].name != "local" || tabs[1].name != "hs" {
		t.Fatalf("tabs=%+v, want local first then hs", tabs)
	}
	if tabs[1].label != "hs-box" || tabs[1].count != 1 || tabs[1].status != "offline" {
		t.Fatalf("hs tab=%+v", tabs[1])
	}
	if rows := m.pickerRows(); len(rows) != 2 || rows[0].source != nil || rows[1].source.GetId() != "terminal:local:term-1" {
		t.Fatalf("local rows=%+v", rows)
	}
	runCmd(t, m, m.onKey("right", ""))
	if m.pickerTabName() != "hs" {
		t.Fatalf("right must select the hs tab: %q", m.pickerTabName())
	}
	rows := m.pickerRows()
	if len(rows) != 2 || rows[0].source != nil || rows[1].source.GetId() != "terminal:hs:term-hs" {
		t.Fatalf("hs rows=%+v", rows)
	}
	// Creating on the selected endpoint targets that machine.
	m.picker = 0
	fake.calls = nil
	runCmd(t, m, m.onKey("enter", ""))
	if m.overlay != overlayPrompt || m.promptKind != "terminal.create" || m.promptEndpoint != "hs" {
		t.Fatalf("create on hs: %+v", fake.calls)
	}
	m.promptFields[0] = "hs-new"
	m.ensurePromptCursors()
	runCmd(t, m, m.onKey("enter", ""))
	if fake.last().method != "terminal.create" || fake.last().params.GetEndpoint() != "hs" {
		t.Fatalf("submitted create on hs: %+v", fake.calls)
	}
}

func TestCreatePromptLoadsEndpointDefaultsAndUsesThem(t *testing.T) {
	m := newModel(nil, false)
	fake := &fakeEmitter{answer: func(method string, params *pb.MethodParams) *pb.Response {
		if method == "access.call" {
			payload, err := gproto.Marshal(&apipb.ResultEnvelope{
				Result: &apipb.ResultEnvelope_TerminalDefaults{TerminalDefaults: &apipb.TerminalDefaultsResult{
					Defaults: &apipb.TerminalDefaults{DefaultCommand: []string{"/bin/bash", "-l"}, DefaultCwd: "/srv/hs"},
				}},
			})
			if err != nil {
				t.Fatalf("marshal defaults: %v", err)
			}
			return &pb.Response{Ok: true, Data: &pb.MethodData{AccessResult: payload}}
		}
		return okResponse(params.GetEndpoint(), "term-created")
	}}
	m.client = fake
	m.host = true
	m.sources = []*pb.Source{{Id: "endpoint:hs", Kind: "endpoint", Title: "HS", Endpoint: "hs", Health: "ok"}}
	m.overlay = overlayPicker
	m.pickerTab = 0
	runCmd(t, m, m.attach(0, false))
	if m.promptDefaults.cwd != "/srv/hs" || createCommandDisplay(m.promptDefaults.command) != "/bin/bash -l" {
		t.Fatalf("endpoint defaults not applied: %+v", m.promptDefaults)
	}
	if m.promptFields[3] != "/srv/hs" || m.promptFields[1] != "" {
		t.Fatalf("default cwd/command values = %#v", m.promptFields)
	}
	if !hasCall(fake.calls, "access.call") {
		t.Fatalf("create form must request endpoint defaults: %+v", fake.calls)
	}
	m.promptFields[0] = "hs-shell"
	m.ensurePromptCursors()
	runCmd(t, m, m.onKey("enter", ""))
	last := fake.last()
	if last.method != "terminal.create" || last.params.GetEndpoint() != "hs" || last.params.GetCwd() != "/srv/hs" || len(last.params.GetArgv()) != 2 || last.params.GetArgv()[0] != "/bin/bash" {
		t.Fatalf("create did not use endpoint defaults: %+v", last)
	}
}

func TestCreatePromptServerSelectionUsesLabelAndFreshDefaults(t *testing.T) {
	m := newModel(nil, false)
	fake := &fakeEmitter{answer: func(method string, params *pb.MethodParams) *pb.Response {
		if method == "access.call" {
			command := []string{"/bin/bash", "-l"}
			cwd := "/srv/hs"
			if params.GetEndpoint() == "local" {
				command = []string{"/bin/zsh"}
				cwd = "/Users/test"
			}
			payload, err := gproto.Marshal(&apipb.ResultEnvelope{
				Result: &apipb.ResultEnvelope_TerminalDefaults{TerminalDefaults: &apipb.TerminalDefaultsResult{
					Defaults: &apipb.TerminalDefaults{DefaultCommand: command, DefaultCwd: cwd},
				}},
			})
			if err != nil {
				t.Fatalf("marshal defaults: %v", err)
			}
			return &pb.Response{Ok: true, Data: &pb.MethodData{AccessResult: payload}}
		}
		return okResponse(params.GetEndpoint(), "term-created")
	}}
	m.client, m.host = fake, true
	m.sources = []*pb.Source{
		{Id: "endpoint:local", Kind: "endpoint", Title: "This Mac", Endpoint: "local", Health: "ok"},
		{Id: "endpoint:hs", Kind: "endpoint", Title: "HS Box", Endpoint: "hs", Health: "ok"},
	}
	m.overlay, m.picker = overlayPicker, 0
	runCmd(t, m, m.attach(0, false))
	if m.promptFields[2] != "This Mac (local)" || m.promptFields[3] != "/Users/test" {
		t.Fatalf("initial endpoint display/defaults = %#v", m.promptFields)
	}
	m.promptField = 2
	m.promptFields[2] = "hs"
	m.ensurePromptCursors()
	runCmd(t, m, m.onKey("tab", ""))
	if !m.promptSuggestionFocused || len(m.promptSuggestions) != 2 || m.promptSuggestions[0] != "HS Box (hs)" || m.promptSuggestions[1] != "This Mac (local)" {
		t.Fatalf("server tab should focus endpoint suggestions: focused=%v suggestions=%v", m.promptSuggestionFocused, m.promptSuggestions)
	}
	runCmd(t, m, m.onKey("enter", ""))
	if m.promptSuggestionFocused || m.promptEndpoint != "hs" || m.promptFields[2] != "HS Box (hs)" || m.promptFields[3] != "/srv/hs" {
		t.Fatalf("accepting server suggestion should switch endpoint/defaults: endpoint=%q fields=%#v", m.promptEndpoint, m.promptFields)
	}
	m.promptFields[0] = "hs-shell"
	m.ensurePromptCursors()
	runCmd(t, m, m.onKey("enter", ""))
	last := fake.last()
	if last.method != "terminal.create" || last.params.GetEndpoint() != "hs" || last.params.GetCwd() != "/srv/hs" || len(last.params.GetArgv()) != 2 || last.params.GetArgv()[0] != "/bin/bash" {
		t.Fatalf("selected endpoint defaults should reach create request: %+v", last)
	}
}

func TestCreatePromptWorkdirUsesFloatingPathSuggestions(t *testing.T) {
	m := newModel(nil, false)
	fake := &fakeEmitter{answer: func(method string, params *pb.MethodParams) *pb.Response {
		if method != "access.call" {
			return okResponse(params.GetEndpoint(), "term-created")
		}
		var command apipb.CommandEnvelope
		if err := gproto.Unmarshal(params.GetAccessCommand(), &command); err != nil {
			t.Fatalf("decode access command: %v", err)
		}
		var result *apipb.ResultEnvelope
		switch command.GetPathListDirectories().GetPrefix() {
		case "d":
			result = &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_PathListDirectories{PathListDirectories: &apipb.PathListDirectoriesResult{
				BasePath: "/srv", Entries: []*apipb.PathDirectoryEntry{{Path: "/srv/demo/"}, {Path: "/srv/dev/"}, {Path: "/srv/delta/"}},
			}}}
		default:
			result = &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_PathListDirectories{PathListDirectories: &apipb.PathListDirectoriesResult{
				BasePath: "/srv/dev", Entries: []*apipb.PathDirectoryEntry{{Path: "/srv/dev/src/"}},
			}}}
		}
		payload, err := gproto.Marshal(result)
		if err != nil {
			t.Fatalf("marshal path result: %v", err)
		}
		return &pb.Response{Ok: true, Data: &pb.MethodData{AccessResult: payload}}
	}}
	m.client, m.host = fake, true
	m.overlay = overlayPrompt
	m.promptKind = "terminal.create"
	m.promptRef = "empty"
	m.promptEndpoint = "local"
	m.promptFields = []string{"shell", "", "local", "d", ""}
	m.promptCursors = []int{5, 0, 5, 1, 0}
	m.promptField = 3
	runCmd(t, m, m.refreshCreateSuggestions(true))
	if !m.promptSuggestionFocused || m.promptSuggestionField != 3 || len(m.promptSuggestions) != 3 || m.promptSuggestionTitle != "path: /srv" {
		t.Fatalf("workdir tab should focus path suggestions: focused=%v field=%d title=%q suggestions=%v", m.promptSuggestionFocused, m.promptSuggestionField, m.promptSuggestionTitle, m.promptSuggestions)
	}
	if strings := m.overlayRows(); len(strings) != 7 {
		t.Fatalf("suggestions must stay out of form rows, rows=%d", len(strings))
	}
	runCmd(t, m, m.onKey("right", ""))
	if m.promptFields[3] != "/srv/demo/" || !m.promptSuggestionFocused || len(m.promptSuggestions) != 1 {
		t.Fatalf("right should enter selected directory and keep focus: fields=%v focused=%v suggestions=%v", m.promptFields, m.promptSuggestionFocused, m.promptSuggestions)
	}
	runCmd(t, m, m.onKey("enter", ""))
	if m.promptFields[3] != "/srv/dev/src/" || m.promptSuggestionFocused {
		t.Fatalf("enter should accept path without submitting: fields=%v focused=%v", m.promptFields, m.promptSuggestionFocused)
	}
}

func TestCreatePromptWorkdirTabWaitsForAsyncSuggestions(t *testing.T) {
	m := newModel(nil, false)
	fake := &fakeEmitter{}
	m.client, m.host = fake, true
	m.overlay = overlayPrompt
	m.promptKind = "terminal.create"
	m.promptRef = "empty"
	m.promptEndpoint = "local"
	m.promptFields = []string{"shell", "", "local", "d", ""}
	m.promptCursors = []int{5, 0, 5, 1, 0}
	m.promptField = 3

	cmd := m.onKey("tab", "")
	if cmd == nil {
		t.Fatal("tab must return the pending path completion command")
	}
	if m.promptField != 3 || m.promptSuggestionFocused {
		t.Fatalf("tab must keep workdir active while async completion is pending: field=%d focused=%v", m.promptField, m.promptSuggestionFocused)
	}
}

func TestCreatePromptMouseSuggestionUsesActiveField(t *testing.T) {
	m := newModel(nil, false)
	fake := &fakeEmitter{answer: func(method string, params *pb.MethodParams) *pb.Response {
		var result *apipb.ResultEnvelope
		if method == "access.call" {
			result = &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_PathListDirectories{PathListDirectories: &apipb.PathListDirectoriesResult{
				BasePath: "/srv", Entries: []*apipb.PathDirectoryEntry{{Path: "/srv/demo/"}, {Path: "/srv/dev/"}},
			}}}
			payload, err := gproto.Marshal(result)
			if err != nil {
				t.Fatalf("marshal path result: %v", err)
			}
			return &pb.Response{Ok: true, Data: &pb.MethodData{AccessResult: payload}}
		}
		return okResponse(params.GetEndpoint(), "term-created")
	}}
	m.client, m.host = fake, true
	m.overlay = overlayPrompt
	m.promptKind = "terminal.create"
	m.promptRef = "empty"
	m.promptEndpoint = "local"
	m.promptFields = []string{"shell", "", "local", "d", ""}
	m.promptCursors = []int{5, 0, 5, 1, 0}
	m.promptField = 3
	runCmd(t, m, m.refreshCreateSuggestions(true))
	if !m.promptSuggestionFocused || m.promptSuggestionField != 3 {
		t.Fatalf("workdir suggestions should own focus before mouse selection: field=%d focused=%v", m.promptSuggestionField, m.promptSuggestionFocused)
	}
	runCmd(t, m, m.onMouse(&pb.MouseEvent{Action: "press", Node: "prompt-suggestion:1"}))
	if m.promptFields[3] != "/srv/dev/" || m.promptFields[2] != "local" {
		t.Fatalf("mouse selection must update workdir only: fields=%v", m.promptFields)
	}
}

func TestCreatePromptRemembersEndpointDraftWithoutName(t *testing.T) {
	m := newModel(nil, false)
	m.sources = []*pb.Source{{Id: "endpoint:hs", Kind: "endpoint", Title: "HS Box", Endpoint: "hs", Health: "ok"}}
	m.createDrafts["hs"] = createDraft{command: "npm run dev", cwd: "/srv/project"}
	m.overlay, m.pickerTab, m.picker = overlayPicker, 0, 0
	if cmd := m.attach(0, false); cmd != nil {
		t.Fatal("offline draft setup should not emit a defaults request")
	}
	if m.promptFields[0] != "" || m.promptFields[1] != "npm run dev" || m.promptFields[2] != "HS Box (hs)" || m.promptFields[3] != "/srv/project" {
		t.Fatalf("next create prompt should reuse endpoint draft but keep name empty: fields=%#v", m.promptFields)
	}
	if !m.promptPublicTagList || m.createPromptPlaceholder(4) != "[production, backend]" {
		t.Fatalf("picker create form should use public tag defaults: public=%v placeholder=%q", m.promptPublicTagList, m.createPromptPlaceholder(4))
	}
}

func TestCreatePromptMatchesMainFieldLayoutAndCursor(t *testing.T) {
	m := newModel(nil, false)
	p := m.focusPane()
	m.overlay = overlayPrompt
	m.promptKind = "terminal.create"
	m.promptRef = p.id
	m.promptEndpoint = "hs"
	m.promptFields = []string{"shell", "bash -lc 'echo hi'", "hs", "/tmp", "kind=task"}
	m.promptCursors = []int{5, 5, 2, 4, 9}
	m.promptField = 1

	box := viewBoxByID(m.View(), "prompt-field:1")
	if box == nil || box.GetCursor() == nil {
		t.Fatal("active create field must expose a cursor")
	}
	wantCol := 9 + 5 // "command: " + cursor before "-lc"
	if got := int(box.GetCursor().GetCol()); got != wantCol {
		t.Fatalf("command cursor col = %d, want %d", got, wantCol)
	}
	if got := int(box.GetCursor().GetRow()); got != 0 {
		t.Fatalf("field cursor row inside its box = %d, want 0", got)
	}
	if got := box.GetCursor().GetShape(); got != "bar" {
		t.Fatalf("create prompt cursor shape = %q, want bar", got)
	}
	// An end-of-input caret needs a trailing cell. Otherwise kernel layout
	// clips it back onto the final rune of the field box.
	m.promptCursors[1] = len([]rune(m.promptFields[1]))
	box = viewBoxByID(m.View(), "prompt-field:1")
	wantWidth := len([]rune("command: ")) + len([]rune(m.promptFields[1])) + 1
	if box.GetSize() == nil || int(box.GetSize().GetWidth()) != wantWidth {
		t.Fatalf("end cursor field width = %v, want %d", box.GetSize(), wantWidth)
	}
	m.promptCursors[1] = 5
	m.rasterize(m.View())
	if m.cursor == nil || m.cursor.y != 13 || m.cursor.x != 43 {
		t.Fatalf("absolute create cursor = %+v, want x=43 y=13", m.cursor)
	}
	m.cols, m.rows = 24, 12
	m.rasterize(m.View())
	if m.cursor == nil || m.cursor.x < 0 || m.cursor.x >= m.cols || m.cursor.y < 0 || m.cursor.y >= m.rows {
		t.Fatalf("small viewport escaped create cursor: %+v", m.cursor)
	}
	for i, want := range []string{"name*:", "command:", "server*:", "workdir:", "tags:"} {
		if viewBoxByID(m.View(), "prompt-field:"+itoa(i)) == nil {
			t.Fatalf("missing create prompt field %q", want)
		}
	}
}

func TestCreatePromptEditsAtCursorAndParsesCommand(t *testing.T) {
	m := newModel(nil, false)
	m.overlay = overlayPrompt
	m.promptKind = "terminal.create"
	m.promptRef = "empty"
	m.promptEndpoint = "local"
	m.promptFields = []string{"shell", "bash -lc 'echo hi'", "local", "", ""}
	m.promptCursors = []int{5, 5, 5, 0, 0}
	m.promptField = 1
	runCmd(t, m, m.onKey("left", ""))
	runCmd(t, m, m.onKey("x", "x"))
	if got := m.promptFields[1]; got != "bashx -lc 'echo hi'" {
		t.Fatalf("insert at cursor changed command to %q", got)
	}
	runCmd(t, m, m.onKey("delete", ""))
	if got := m.promptFields[1]; got != "bashx-lc 'echo hi'" {
		t.Fatalf("delete at cursor changed command to %q", got)
	}
	got, err := parsePromptCommand(`bash -lc 'echo hi there'`)
	if err != nil || len(got) != 3 || got[2] != "echo hi there" {
		t.Fatalf("quoted command parse = %#v", got)
	}
}

func TestCreatePromptRequiresName(t *testing.T) {
	m := newModel(nil, false)
	m.overlay = overlayPrompt
	m.promptKind = "terminal.create"
	m.promptRef = "empty"
	m.promptEndpoint = "local"
	m.promptFields = []string{"", "", "local", "", ""}
	m.promptCursors = make([]int, len(m.promptFields))
	if cmd := m.submitCreatePrompt(); cmd != nil {
		t.Fatal("invalid create form must not emit a command")
	}
	if m.overlay != overlayPrompt || m.promptError != "name is required" {
		t.Fatalf("missing name should keep form open with an error: overlay=%q error=%q", m.overlay, m.promptError)
	}
}

func TestCreatePromptWaitsForEndpointDefaults(t *testing.T) {
	m := newModel(nil, false)
	m.overlay = overlayPrompt
	m.promptKind = "terminal.create"
	m.promptRef = "empty"
	m.promptEndpoint = "local"
	m.promptFields = []string{"shell", "", "local", "", ""}
	m.promptCursors = make([]int, len(m.promptFields))
	if cmd := m.submitCreatePrompt(); cmd != nil {
		t.Fatal("create must not emit while endpoint defaults are unavailable")
	}
	if m.overlay != overlayPrompt || m.promptError != "endpoint defaults are not loaded" {
		t.Fatalf("missing defaults should keep form open: overlay=%q error=%q", m.overlay, m.promptError)
	}
}

func TestWheelScrollsLocalPaneAndTerminal(t *testing.T) {
	local := newModel(nil, false)
	pane := local.focusPane()
	pane.lines = []string{"a", "b", "c"}
	local.onWheel(&pb.WheelEvent{Node: pane.id, Delta: -3})
	if pane.scroll == 0 {
		t.Fatalf("local pane wheel must move the line window")
	}

	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "history.window" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{
				Rows: []string{"one", "two", "three"}, Offset: 0,
			}}
		}
		return &pb.Response{Ok: true}
	}
	p := m.focusPane()
	fake.calls = nil
	// The first wheel enters copy and loads the window; the movement is
	// applied once the window arrives (cursor at the bottom, then up).
	runCmd(t, m, m.onWheel(&pb.WheelEvent{Node: p.id, Delta: 1}))
	st := m.copyFor(p)
	if st == nil || len(st.rows) != 3 {
		t.Fatalf("wheel must open a copy session with the window: %+v", st)
	}
	if st.cursorRow != 1 {
		t.Fatalf("wheel moved the cursor to row %d, want 1", st.cursorRow)
	}
	// At the top edge the view scrolls instead (old ScrollCursor overflow).
	fake.calls = nil
	runCmd(t, m, m.onWheel(&pb.WheelEvent{Node: p.id, Delta: 5}))
	if !hasCall(fake.calls, "terminal.scroll") {
		t.Fatalf("wheel past the top edge must scroll the view: %+v", fake.calls)
	}
	// Wheel down while not in copy does nothing (old binding).
	m2, fake2 := boundModel(t)
	fake2.calls = nil
	runCmd(t, m2, m2.onWheel(&pb.WheelEvent{Node: m2.focusPane().id, Delta: -1}))
	if len(fake2.calls) != 0 {
		t.Fatalf("wheel down without a session must be a no-op: %+v", fake2.calls)
	}
}

// TestMarkThenMoveSelects pins the user flow: wheel enters copy, space marks
// at the bottom, k moves the cursor up through the text and the selection
// spans grow with it (the mark never drifts).
func TestMarkThenMoveSelects(t *testing.T) {
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "history.window" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{
				Rows: []string{"alpha", "beta", "gamma"}, Offset: 0,
			}}
		}
		return &pb.Response{Ok: true}
	}
	p := m.focusPane()
	runCmd(t, m, m.onWheel(&pb.WheelEvent{Node: p.id, Delta: 1}))
	st := m.copyFor(p)
	if st == nil || st.cursorRow != 1 {
		t.Fatalf("wheel must enter at the bottom and move one row older: %+v", st)
	}
	runCmd(t, m, key(m, "space"))
	if !st.marked || st.markRow != 1 {
		t.Fatalf("space must mark the cursor row: %+v", st)
	}
	runCmd(t, m, key(m, "k"))
	if st.cursorRow != 0 || st.markRow != 1 {
		t.Fatalf("k must move the cursor up while the mark stays: cursor=%d mark=%d", st.cursorRow, st.markRow)
	}
	spans := m.selectionSpans(st)
	if len(spans) != 2 {
		t.Fatalf("spans = %+v, want two rows", spans)
	}
	if spans[0].row != 0 || spans[0].startCol != 0 || spans[1].row != 1 || spans[1].endCol != 0 {
		t.Fatalf("spans = %+v", spans)
	}
	props := m.copyProps(st)
	if props["copy.sel"] == "" {
		t.Fatalf("selection props missing: %+v", props)
	}
}

func TestQuitEmitsSystemQuit(t *testing.T) {
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response { return &pb.Response{Ok: true} }
	fake.calls = nil
	msgs := app.RunCmd(m.quitCmd())
	if fake.last().method != "system.quit" {
		t.Fatalf("quit must emit system.quit: %+v", fake.calls)
	}
	if len(msgs) == 0 {
		t.Fatalf("an accepted quit must yield the loop-stop message")
	}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}

func TestPickerSplitAttach(t *testing.T) {
	m, fake := boundModel(t)
	m.sources = append(m.sources, &pb.Source{
		Id: "terminal:local:term-9", Kind: "terminal", Title: "term-9",
		Endpoint: "local", TerminalId: "term-9", Attached: false,
	})
	tab := m.activeTab()
	before := len(tab.panes)
	m.overlay = overlayPicker
	rows := m.pickerRows()
	m.picker = 0
	for i, row := range rows {
		if row.source != nil && row.source.GetId() == "terminal:local:term-9" {
			m.picker = i
		}
	}
	fake.calls = nil
	runCmd(t, m, m.onKey("tab", ""))
	if len(tab.panes) != before+1 {
		t.Fatalf("tab must split-attach: panes %d -> %d", before, len(tab.panes))
	}
	if fake.last().method != "terminal.attach" || fake.last().params.GetId() != "term-9" {
		t.Fatalf("split attach must bind the chosen terminal: %+v", fake.calls)
	}
	newPane := m.focusPane()
	if newPane.sourceID != "terminal:local:term-9" {
		t.Fatalf("new leaf not bound: %q", newPane.sourceID)
	}
	if m.overlay != "" {
		t.Fatalf("split attach must close the picker")
	}
}

func TestPromptAcceptsSpacesAndWideRunes(t *testing.T) {
	m := demoModel(120, 32)
	m.mode = modeSystem
	runCmd(t, m, key(m, "o"))
	runCmd(t, m, m.onKey("space", " "))
	runCmd(t, m, m.onKey("s", "s"))
	runCmd(t, m, m.onKey("p", "p"))
	if m.prompt != " sp" {
		t.Fatalf("prompt=%q, want %q", m.prompt, " sp")
	}
	m.prompt = "复制"
	runCmd(t, m, m.onKey("backspace", ""))
	if m.prompt != "复" {
		t.Fatalf("backspace must drop one rune: %q", m.prompt)
	}
}

func TestLayoutToggleOnlyFocusedSplit(t *testing.T) {
	m := demoModel(120, 32)
	tab := m.activeTab()
	m.closePane(tab, tab.panes[1])
	runCmd(t, m, m.splitPane("row"))
	outer := m.splitEntries(tab)[0].node
	runCmd(t, m, m.splitPane("col"))
	inner := m.splitEntries(tab)
	if len(inner) != 2 {
		t.Fatalf("splits=%d, want 2", len(inner))
	}
	m.mode = modeResize
	runCmd(t, m, key(m, "space"))
	if outer.orient != "row" {
		t.Fatalf("outer split must keep its axis: %q", outer.orient)
	}
	if inner[1].node.orient != "row" {
		t.Fatalf("focused inner split must toggle: %q", inner[1].node.orient)
	}
}

func TestAttachOwnerConflictRetriesAsFollower(t *testing.T) {
	m := newModel(nil, false)
	m.host = true
	fake := &fakeEmitter{}
	attachCalls := 0
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "terminal.attach" {
			attachCalls++
			if attachCalls == 1 {
				return failResponse("owner conflict")
			}
			if params.GetFit() {
				t.Fatalf("retry must use fit=false")
			}
		}
		return &pb.Response{Ok: true}
	}
	m.client = fake
	m.sources = append(m.sources, &pb.Source{
		Id: "terminal:local:term-1", Kind: "terminal", Title: "term-1",
		Endpoint: "local", TerminalId: "term-1", Attached: true, ResizeOwner: "view:other",
	})
	p := m.focusPane()
	fit := true
	runCmd(t, m, m.bindPending(p.id, "terminal:local:term-1", &pb.MethodParams{
		Endpoint: "local", Id: "term-1", Fit: &fit,
	}))
	if attachCalls != 2 {
		t.Fatalf("attach calls=%d, want 2 (conflict + follow retry)", attachCalls)
	}
	if p.sourceID != "terminal:local:term-1" || p.pending != "" {
		t.Fatalf("pane not bound after the follow retry: %q pending=%q", p.sourceID, p.pending)
	}
}

func TestPaneLockClickPreventsSplitResize(t *testing.T) {
	m := demoModel(120, 32)
	tab := m.activeTab()
	left := m.focusPane()
	m.closePane(tab, tab.panes[1])
	runCmd(t, m, m.splitPane("row"))
	m.focusPaneObject(left)
	entries := m.splitEntries(tab)
	if len(entries) != 1 {
		t.Fatalf("splits=%d, want 1", len(entries))
	}
	sp := entries[0].node
	ratio := sp.ratio
	if cmd := m.handleChromeClick("pane:" + left.id + ":lock"); cmd != nil {
		runCmd(t, m, cmd)
	}
	if !left.locked {
		t.Fatal("lock click must toggle the pane lock")
	}
	m.mode = modeResize
	m.resizeFocused(4, false)
	if sp.ratio != ratio {
		t.Fatalf("locked split ratio changed from %v to %v", ratio, sp.ratio)
	}
}

// TestCopySessionsArePerPane pins the old CopyModeByView model: copy sessions
// live per pane, only the focused pane's session owns input, clicking another
// pane transfers input without discarding the first session, and returning to
// it resumes the copy scene.
func TestCopySessionsArePerPane(t *testing.T) {
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response { return &pb.Response{Ok: true} }
	left := m.focusPane()
	right := m.splitLeafFor("row", nil)
	if right == nil {
		t.Fatalf("split failed")
	}
	m.focusPaneObject(left)
	runCmd(t, m, key(m, "ctrl-shift-c"))
	if !m.copyActive() || m.copyPanes[left.id] == nil {
		t.Fatalf("left pane must own a copy session: %+v", m.copyPanes)
	}
	fake.calls = nil
	// Click the right pane: it becomes active, so it owns input; the left
	// session is kept (scrollback position preserved), no scrollEnd is sent.
	runCmd(t, m, m.onMouse(&pb.MouseEvent{Action: "press", Node: right.id, X: 90, Y: 10}))
	if m.focusPane() != right {
		t.Fatalf("clicked pane must take focus")
	}
	if m.copyActive() {
		t.Fatalf("the active pane has no session, so copy must not own input")
	}
	if _, ok := m.copyPanes[left.id]; !ok {
		t.Fatalf("the left pane must keep its copy session")
	}
	m.mode = modeLive // the split left the pane scene open
	if m.claim().All {
		t.Fatalf("without a session on the active pane, input must go to its PTY")
	}
	for _, call := range fake.calls {
		if call.method == "terminal.scrollEnd" {
			t.Fatalf("focus change must not end the other pane's session: %+v", fake.calls)
		}
	}
	// Return to the left pane: its copy scene is active again.
	runCmd(t, m, m.onMouse(&pb.MouseEvent{Action: "press", Node: left.id, X: 20, Y: 10}))
	if m.focusPane() != left || !m.copyActive() {
		t.Fatalf("returning to the pane must resume its copy scene")
	}
}

// TestCopyFrameTurnsHistoryYellow pins the legacy paneChromeStyle rule: a pane
// with an open copy/scrollback session draws its frame in the yellow
// history-border color, focused or not, while the title and action glyphs keep
// the accent style.
func TestCopyFrameTurnsHistoryYellow(t *testing.T) {
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response { return &pb.Response{Ok: true} }
	m.cols, m.rows = 120, 32
	left := m.focusPane()
	right := m.splitLeafFor("row", nil)
	if right == nil {
		t.Fatal("split failed")
	}
	m.focusPaneObject(left)
	runCmd(t, m, key(m, "ctrl-shift-c"))

	// The focused copy pane's top border is the history color.
	runs := m.paneRuns(left, true, 80)
	if runs[0].style != stHistoryBorder {
		t.Fatalf("copy pane frame = %q, want %q", runs[0].style, stHistoryBorder)
	}
	// The title runs on the same row stay accent (only the border recolors).
	sawAccentTitle := false
	for _, run := range runs[2:] {
		if run.style == stAccent {
			sawAccentTitle = true
		}
	}
	if !sawAccentTitle {
		t.Fatalf("copy pane title must keep the accent style: %+v", runs)
	}

	// Focus the sibling: it has no session, so its frame is accent, and the
	// unfocused copy pane stays yellow.
	m.mode = modeLive
	m.focusPaneObject(right)
	if got := m.paneRuns(right, true, 80)[0].style; got != stAccent {
		t.Fatalf("non-copy focused frame = %q, want accent", got)
	}
	if got := m.paneRuns(left, false, 80)[0].style; got != stHistoryBorder {
		t.Fatalf("unfocused copy frame = %q, want %q", got, stHistoryBorder)
	}

	// Ending the copy session restores the normal frame color.
	m.focusPaneObject(left)
	runCmd(t, m, key(m, "G"))
	if m.copyFor(left) != nil {
		t.Fatal("G must close the left copy session")
	}
	if got := m.paneRuns(left, false, 80)[0].style; got == stHistoryBorder {
		t.Fatalf("closed copy frame stayed history yellow: %q", got)
	}
}

// TestCopySearchTypingDoesNotMoveCursor pins the scan-as-you-type state:
// typing highlights matches but leaves the cursor where it was; Enter jumps.
func TestCopySearchTypingDoesNotMoveCursor(t *testing.T) {
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "terminal.search" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{Found: true, Wrapped: true,
				Rows: []string{"one two", "two three", "four"}, MatchStart: 4, MatchEnd: 7}}
		}
		if method == "history.window" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{
				Rows: []string{"one two", "two three", "four"}, Offset: 0,
			}}
		}
		return &pb.Response{Ok: true}
	}
	runCmd(t, m, key(m, "ctrl-shift-c"))
	st := m.copyFor(m.focusPane())
	st.cols = 80
	runCmd(t, m, key(m, "k")) // move up one row from the entry position
	entryRow := st.cursorRow
	runCmd(t, m, key(m, "/"))
	for _, r := range "two" {
		runCmd(t, m, m.onKey(string(r), string(r)))
	}
	if st.query != "two" || st.searching != true {
		t.Fatalf("query=%q searching=%v", st.query, st.searching)
	}
	if len(st.matches) == 0 {
		t.Fatalf("scan-as-you-type must highlight matches")
	}
	if st.cursorRow != entryRow {
		t.Fatalf("typing moved the cursor: %d -> %d", entryRow, st.cursorRow)
	}
	runCmd(t, m, m.onKey("enter", ""))
	if st.searching || st.cursorRow == entryRow {
		t.Fatalf("enter must jump to the next match: searching=%v row=%d", st.searching, st.cursorRow)
	}
}

func TestCopySearchModes(t *testing.T) {
	rows := []string{"one two", "two three", "four"}
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "history.window" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: rows, Offset: 0}}
		}
		return &pb.Response{Ok: true}
	}
	runCmd(t, m, key(m, "ctrl-shift-c"))
	st := m.copyFor(m.focusPane())
	// text mode treats the pattern literally: no "t*e" match.
	st.query = "t*e"
	runCmd(t, m, m.runCopySearch(m.focusPane(), st, true, false))
	if len(st.matches) != 0 {
		t.Fatalf("text mode matched a glob pattern: %+v", st.matches)
	}
	// glob mode is anchored: "t*e" matches "two three".
	st.searchMode = copySearchGlob
	runCmd(t, m, m.runCopySearch(m.focusPane(), st, true, false))
	if len(st.matches) != 1 || st.matches[0].row != 1 {
		t.Fatalf("glob matches = %+v", st.matches)
	}
	// regex mode is unanchored: "t.o" matches "two" on two rows.
	st.searchMode = copySearchRegex
	st.query = "t.o"
	runCmd(t, m, m.runCopySearch(m.focusPane(), st, true, false))
	if len(st.matches) != 2 {
		t.Fatalf("regex matches = %+v", st.matches)
	}
}

func TestCopySearchInputEditing(t *testing.T) {
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "history.window" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: []string{"one"}, Offset: 0}}
		}
		return &pb.Response{Ok: true}
	}
	runCmd(t, m, key(m, "ctrl-shift-c"))
	st := m.copyFor(m.focusPane())
	runCmd(t, m, key(m, "/"))
	for _, r := range "two" {
		runCmd(t, m, m.onKey(string(r), string(r)))
	}
	runCmd(t, m, m.onKey("home", ""))
	runCmd(t, m, m.onKey("x", "x"))
	if st.query != "xtwo" || st.searchCol != 1 {
		t.Fatalf("insert at home = %q col=%d", st.query, st.searchCol)
	}
	runCmd(t, m, m.onKey("delete", ""))
	if st.query != "xwo" {
		t.Fatalf("delete = %q", st.query)
	}
	runCmd(t, m, m.onKey("backspace", ""))
	if st.query != "wo" {
		t.Fatalf("backspace = %q", st.query)
	}
}

// TestCopySearchInteractionMatchesLegacy pins the copy-search flow the legacy
// copymode used: esc does not exit the copy scene, the search bar persists
// after Enter, tab only cycles the mode while the bar is visible, and n/p
// advance past the current match.
func TestCopySearchInteractionMatchesLegacy(t *testing.T) {
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "history.window" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{
				Rows: []string{"one two", "two three", "four"}, Offset: 0,
			}}
		}
		if method == "terminal.search" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{Found: true,
				Rows: []string{"one two", "two three", "four"}, MatchStart: 4, MatchEnd: 7}}
		}
		return &pb.Response{Ok: true}
	}
	runCmd(t, m, key(m, "ctrl-shift-c"))
	st := m.copyFor(m.focusPane())
	st.cols = 80

	// esc must stay in the copy scene (the legacy scene has no esc binding).
	p := m.focusPane()
	m.mode = modeLive
	runCmd(t, m, key(m, "esc"))
	if m.copyFor(p) == nil {
		t.Fatal("esc must not exit the copy scene")
	}

	// tab while the search bar is hidden must not cycle the mode.
	mode := st.searchMode
	runCmd(t, m, key(m, "tab"))
	if st.searchMode != mode {
		t.Fatalf("tab with a hidden search bar cycled the mode: %d -> %d", mode, st.searchMode)
	}

	// / then a query, then Enter: editing stops but the bar persists.
	runCmd(t, m, key(m, "/"))
	for _, r := range "two" {
		runCmd(t, m, m.onKey(string(r), string(r)))
	}
	runCmd(t, m, m.onKey("enter", ""))
	if st.searching {
		t.Fatal("enter must stop search editing")
	}
	if st.query != "two" {
		t.Fatalf("query cleared by enter: %q", st.query)
	}
	if left, _ := m.copySearchFooterRuns(st); len(left) == 0 {
		t.Fatal("the search bar must stay visible after enter")
	}

	// tab now cycles the mode (bar visible).
	mode = st.searchMode
	runCmd(t, m, key(m, "tab"))
	if st.searchMode == mode {
		t.Fatal("tab with a visible search bar must cycle the mode")
	}

	// n advances past the current match instead of restarting at the cursor.
	fake.calls = nil
	runCmd(t, m, key(m, "n"))
	if fake.last().method != "terminal.search" {
		t.Fatalf("n must emit terminal.search: %+v", fake.calls)
	}
	last := st.currentMatch[len(st.currentMatch)-1]
	wantStart := int32(last.row*st.cols + last.endCol + 1)
	if got := fake.last().params.GetSel().GetStart(); got != wantStart {
		t.Fatalf("n start = %d, want %d (one past the current match)", got, wantStart)
	}
}

// TestCopySearchBarReplacesFooter pins the legacy placement: the copy search
// bar overwrites the global footer row (not the panel's last content row) and
// shows the mode prefix, the query and the caret.
func TestCopySearchBarReplacesFooter(t *testing.T) {
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "history.window" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{
				Rows: []string{"alpha", "beta"}, Offset: 0,
			}}
		}
		return &pb.Response{Ok: true}
	}
	runCmd(t, m, key(m, "ctrl-shift-c"))
	runCmd(t, m, key(m, "/"))
	for _, r := range "be" {
		runCmd(t, m, m.onKey(string(r), string(r)))
	}
	lines := screenLines(m)
	footer := lines[m.rows-1]
	if !strings.Contains(footer, "\u2315 [TEXT] be") {
		t.Fatalf("footer search bar = %q, want the mode prefix and query", footer)
	}
	// The panel content area must not carry the "/query" bar any more.
	body := strings.Join(lines[1:m.rows-1], "\n")
	if strings.Contains(body, "\u2315 [TEXT]") {
		t.Fatalf("search bar leaked into the panel body:\n%s", body)
	}
}

// TestLegacyKeyAliases aligns the remaining legacy scene bindings: panel
// x/w close, X kill, R restart; tab X kill; floating H/L/K/J resize; global
// Ctrl-V/PageUp copy entry; system T close toast.
// TestResizeCenterAndLarge pins the legacy resize alignment keys: m centers
// the focused split, H/L move it by a large (quarter-axis) step.
func TestResizeCenterAndLarge(t *testing.T) {
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response { return &pb.Response{Ok: true} }
	m.cols, m.rows = 120, 40
	m.splitLeafFor("row", nil)
	m.mode = modeResize
	sp := m.activeTab().root.(*split)
	sp.ratio = 0.5
	runCmd(t, m, key(m, "m"))
	if sp.ratio != 0.5 {
		t.Fatalf("resize.center m = %v, want 0.5", sp.ratio)
	}
	// A large step must move the axis at least as far as the small step.
	runCmd(t, m, key(m, "l"))
	small := sp.ratio
	sp.ratio = 0.5
	runCmd(t, m, key(m, "L"))
	if absFloat(sp.ratio-0.5) < absFloat(small-0.5) {
		t.Fatalf("resize.right_large step %.3f < small %.3f", sp.ratio, small)
	}
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func TestLegacyKeyAliases(t *testing.T) {
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: []string{"alpha"}}}
	}

	// Floating H/L/K/J resize the active floating.
	m.cols, m.rows = 120, 32
	runCmd(t, m, key(m, "ctrl-o"))
	if cmd := m.newFloating(); cmd != nil {
		runCmd(t, m, cmd)
	}
	f := m.activeFloating()
	if f == nil {
		t.Fatal("floating.new must create a floating")
	}
	m.overlay = "" // floating.new opens the picker; test the floating scene itself
	m.mode = modeFloating
	w0 := f.w
	runCmd(t, m, key(m, "L"))
	if f.w <= w0 {
		t.Fatalf("floating L must widen: %d -> %d", w0, f.w)
	}

	// system T clears the toast.
	m.mode = modeSystem
	m.toast = "hello"
	runCmd(t, m, key(m, "T"))
	if m.toast != "" {
		t.Fatalf("system T must clear the toast, got %q", m.toast)
	}

	// panel w closes like x.
	m.mode = modePane
	before := len(m.activeTab().panes)
	runCmd(t, m, key(m, "w"))
	if len(m.activeTab().panes) != before {
		t.Fatalf("panel w must close: %d -> %d", before, len(m.activeTab().panes))
	}
}

func TestCopyReenterGoesToLatest(t *testing.T) {
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "history.window" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{
				Rows: []string{"alpha", "beta", "gamma"}, Offset: 0,
			}}
		}
		return &pb.Response{Ok: true}
	}
	runCmd(t, m, key(m, "ctrl-shift-c"))
	st := m.copyFor(m.focusPane())
	runCmd(t, m, key(m, "space"))
	runCmd(t, m, key(m, "k"))
	st.offset = 4 // pretend the view is scrolled
	fake.calls = nil
	runCmd(t, m, key(m, "ctrl-shift-c"))
	if st.marked || st.query != "" || st.searching {
		t.Fatalf("re-enter must clear selection/search: %+v", st)
	}
	if st.cursorRow != len(st.rows)-1 || st.offset != 0 {
		t.Fatalf("re-enter must return to the newest row: row=%d offset=%d", st.cursorRow, st.offset)
	}
	if !hasCall(fake.calls, "terminal.scrollEnd") {
		t.Fatalf("re-enter must release the frozen window: %+v", fake.calls)
	}
}

func TestMouseClickMarksInCopy(t *testing.T) {
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "history.window" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: []string{"alpha", "beta"}, Offset: 0}}
		}
		return &pb.Response{Ok: true}
	}
	runCmd(t, m, key(m, "ctrl-shift-c"))
	st := m.copyFor(m.focusPane())
	runCmd(t, m, key(m, "k"))
	runCmd(t, m, m.onMouse(&pb.MouseEvent{Action: "press", Node: m.focusPane().id, X: 10, Y: 5}))
	if !st.marked || st.markRow != st.cursorRow {
		t.Fatalf("mouse click must anchor at the copy cursor: %+v", st)
	}
}

func TestMouseCopySelectionUsesContentCoordinates(t *testing.T) {
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "history.window" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: []string{"alpha", "beta", "gamma"}, Offset: 0}}
		}
		return &pb.Response{Ok: true}
	}
	p := m.focusPane()
	runCmd(t, m, key(m, "ctrl-shift-c"))
	st := m.copyFor(p)
	if st == nil {
		t.Fatal("copy session was not opened")
	}
	// A normal card has a one-cell border and its terminal content starts at
	// zero-based (1, 2); input coordinates are one-based screen cells.
	runCmd(t, m, m.onMouse(&pb.MouseEvent{Action: "press", Node: p.id, X: 5, Y: 3}))
	if st.cursorRow != 0 || st.cursorCol != 3 || st.markRow != 0 || st.markCol != 3 {
		t.Fatalf("mouse press mapped to row=%d col=%d mark=%d,%d", st.cursorRow, st.cursorCol, st.markRow, st.markCol)
	}
	runCmd(t, m, m.onMouse(&pb.MouseEvent{Action: "drag", Node: p.id, X: 6, Y: 4}))
	if st.cursorRow != 1 || st.cursorCol != 4 {
		t.Fatalf("mouse drag mapped to row=%d col=%d", st.cursorRow, st.cursorCol)
	}
	runCmd(t, m, m.onMouse(&pb.MouseEvent{Action: "release", Node: p.id}))
}

func TestCopyCursorStaysInsidePaneViewport(t *testing.T) {
	m, _ := boundModel(t)
	m.rows = 8
	p := m.focusPane()
	st := &copyState{rows: make([]string, 64), cursorRow: 63, cursorCol: 999}
	m.copyPanes[p.id] = st
	m.View()
	if st.viewRows != 4 {
		t.Fatalf("copy viewport rows = %d, want 4", st.viewRows)
	}
	if st.cursorRow != st.viewRows-1 || st.cursorCol != st.cols-1 {
		t.Fatalf("cursor escaped viewport: row=%d/%d col=%d/%d", st.cursorRow, st.viewRows, st.cursorCol, st.cols)
	}
}

func TestCopyHistoryUsesPaneViewportRows(t *testing.T) {
	m, fake := boundModel(t)
	m.rows = 8
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "history.window" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: make([]string, 64), Offset: 0}}
		}
		return &pb.Response{Ok: true}
	}
	// The initial live frame establishes the pane content height before copy
	// mode creates its state and requests history.
	m.View()
	p := m.focusPane()
	runCmd(t, m, key(m, "ctrl-shift-c"))
	st := m.copyFor(p)
	if st == nil || st.viewRows != 4 || st.cursorRow != 3 {
		t.Fatalf("history cursor was not placed in visible rows: %+v", st)
	}
	if fake.last().method != "history.window" || fake.last().params.GetRows() != 4 {
		t.Fatalf("history request rows = %d, want 4", fake.last().params.GetRows())
	}
	fake.calls = nil
	runCmd(t, m, key(m, "g"))
	if !hasCall(fake.calls, "terminal.scroll") {
		t.Fatalf("moving past the visible history window must scroll: %+v", fake.calls)
	}
}

func TestExitedTerminalCanReadHistory(t *testing.T) {
	m, fake := boundModel(t)
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		if method == "history.window" {
			return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: []string{"old", "last"}, Offset: 0}}
		}
		return &pb.Response{Ok: true}
	}
	m.sources[0].Exited = true
	p := m.focusPane()
	runCmd(t, m, key(m, "ctrl-shift-c"))
	if m.copyFor(p) == nil || !hasCall(fake.calls, "history.window") {
		t.Fatalf("exited terminal must still open history: copy=%+v calls=%+v", m.copyFor(p), fake.calls)
	}
	fake.calls = nil
	runCmd(t, m, m.onWheel(&pb.WheelEvent{Node: p.id, Delta: 5}))
	if !hasCall(fake.calls, "terminal.scroll") {
		t.Fatalf("exited terminal history must remain scrollable: %+v", fake.calls)
	}
}

// Search navigation is terminal-owned. The shell receives a deep viewport
// without downloading or walking the intervening pages.
func TestCopySearchDelegatesDeepHistory(t *testing.T) {
	m, fake := boundModel(t)
	deep := []string{"target here", "more"}
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		switch method {
		case "history.window":
			return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: []string{"latest"}}}
		case "terminal.search":
			if params.GetQuery() != "target" || params.GetSearchMode() != "text" || params.GetBackward() {
				t.Errorf("search intent = %v", params)
			}
			return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: deep, Offset: 6200, Found: true, MatchStart: 0, MatchEnd: 6}}
		}
		return &pb.Response{Ok: true}
	}
	runCmd(t, m, key(m, "ctrl-shift-c"))
	st := m.copyFor(m.focusPane())
	st.cols = 80
	st.query = "target"
	fake.calls = nil
	runCmd(t, m, m.runCopySearch(m.focusPane(), st, true, true))
	if len(fake.calls) != 1 || fake.calls[0].method != "terminal.search" {
		t.Fatalf("search must be a single terminal operation: %v", fake.calls)
	}
	if st.offset != 6200 || st.cursorRow != 0 || st.cursorCol != 0 || st.rows[0] != "target here" {
		t.Fatalf("deep search viewport = %+v", st)
	}
}

// TestConnectionsOverlayListsAndReconnects pins the SYSTEM connections overlay
// (legacy system.open_connections): e opens it, endpoint.list rows parse into
// the model, Down moves the selection and r reconnects the selected endpoint
// and re-lists so health refreshes.
func TestConnectionsOverlayListsAndReconnects(t *testing.T) {
	m, fake := boundModel(t)
	rows := []string{
		`{"name":"local","label":"Local","kind":"local","health":"ok"}`,
		`{"name":"dev","label":"Dev box","kind":"daemon","health":"unknown"}`,
	}
	fake.answer = func(method string, params *pb.MethodParams) *pb.Response {
		switch method {
		case "endpoint.list":
			return &pb.Response{Ok: true, Data: &pb.MethodData{Rows: rows}}
		case "endpoint.reconnect":
			if params.GetEndpoint() != "dev" {
				t.Errorf("reconnect endpoint = %q, want dev", params.GetEndpoint())
			}
			return okResponse(params.GetEndpoint(), "")
		}
		return okResponse("local", "term-1")
	}
	m.mode = modeSystem
	runCmd(t, m, key(m, "e"))
	if m.overlay != overlayConnections {
		t.Fatalf("e must open the connections overlay, got %q", m.overlay)
	}
	if len(m.connections) != 2 || m.connections[0].label != "Local" ||
		m.connections[1].name != "dev" || m.connections[1].kind != "daemon" {
		t.Fatalf("connection rows not parsed: %+v", m.connections)
	}
	if rows := m.overlayRows(); len(rows) != 2 || !rows[0].selectable {
		t.Fatalf("connections overlay must render selectable rows: %+v", rows)
	}
	runCmd(t, m, m.onKey("down", ""))
	if m.connSel != 1 {
		t.Fatalf("down must move the selection to the second row, got %d", m.connSel)
	}
	fake.calls = nil
	runCmd(t, m, m.onKey("r", ""))
	var reconnect *emitCall
	for i := range fake.calls {
		if fake.calls[i].method == "endpoint.reconnect" {
			reconnect = &fake.calls[i]
		}
	}
	if reconnect == nil || reconnect.params.GetEndpoint() != "dev" {
		t.Fatalf("r must reconnect the selected endpoint: %+v", fake.calls)
	}
	if !hasCall(fake.calls, "endpoint.list") {
		t.Fatalf("reconnect must re-list so health refreshes: %+v", fake.calls)
	}
	if m.toast != "reconnect dev: ok" {
		t.Fatalf("reconnect toast = %q", m.toast)
	}
}

// TestAttachCountPerPane pins the observer-count badge: the visible count is
// the daemon total with this client's single shared attachment replaced by the
// number of local panes bound to the source. With no daemon count (0) it is
// just the local pane count, so a lone pane shows x1 and two panes show x2;
// with daemon=2 and two local panes the total is 3 (another client's pane plus
// this program's two). A pane with no source shows 0.
func TestAttachCountPerPane(t *testing.T) {
	newSource := func(count int32) *pb.Source {
		return &pb.Source{
			Id: "terminal:local:term-1", Kind: "terminal", Title: "term-1",
			Endpoint: "local", TerminalId: "term-1", Attached: true,
			AttachmentCount: count,
		}
	}

	m := newModel(nil, false)
	m.host = true
	m.viewID = "view-a"
	m.cols, m.rows = 120, 32
	tab := m.activeTab()
	first := tab.panes[0]
	second := m.splitLeafFor("row", first)
	if second == nil {
		t.Fatal("split failed")
	}
	m.sources = []*pb.Source{newSource(0)}
	m.bindPane(first.id, "terminal:local:term-1")

	// One local pane, no daemon count -> x1.
	if got := m.attachCount(first); got != 1 {
		t.Fatalf("one pane, daemon 0 = %d, want 1", got)
	}
	// Two local panes on the same source -> x2 (no shared-seat subtraction).
	m.bindPane(second.id, "terminal:local:term-1")
	if got := m.attachCount(first); got != 2 {
		t.Fatalf("two panes, daemon 0 = %d, want 2", got)
	}
	if got := m.attachCount(second); got != 2 {
		t.Fatalf("second pane, daemon 0 = %d, want 2", got)
	}
	// Daemon reports 2 (another client plus this shared client) with two local
	// panes: 2 + 2 - 1 = 3.
	m.sources = []*pb.Source{newSource(2)}
	if got := m.attachCount(first); got != 3 {
		t.Fatalf("two panes, daemon 2 = %d, want 3", got)
	}
	// A pane with no source has no observers.
	empty := m.newPane("empty", nil)
	if got := m.attachCount(empty); got != 0 {
		t.Fatalf("pane without a source = %d, want 0", got)
	}

	// A floating window bound to the same source counts as another observer.
	floating := &floating{id: "float-test", pane: m.newPane("float", nil)}
	floating.pane.sourceID = "terminal:local:term-1"
	m.floatings = append(m.floatings, floating)
	if got := m.attachCount(first); got != 4 {
		t.Fatalf("two tab panes + one floating, daemon 2 = %d, want 4", got)
	}
}

// TestFooterNodeClickRoutesToKeyHandler pins HIGH 1: every atomic footer hit
// node must reach the same handler as its advertised key, not be a no-op. A
// representative subset covers each scene family (mode chord, pane, resize,
// tab, workspace, system, floating, copy).
func TestFooterNodeClickRoutesToKeyHandler(t *testing.T) {
	t.Run("mode chords", func(t *testing.T) {
		for _, tc := range []struct {
			node string
			mode string
		}{
			{"f:ctrl-p", modePane},
			{"f:ctrl-r", modeResize},
			{"f:ctrl-t", modeTab},
			{"f:ctrl-w", modeWorkspace},
			{"f:ctrl-g", modeSystem},
		} {
			m := demoModel(120, 32)
			runCmd(t, m, press(m, tc.node))
			if m.mode != tc.mode {
				t.Fatalf("%s: mode=%q want %q", tc.node, m.mode, tc.mode)
			}
		}
		m := demoModel(120, 32)
		runCmd(t, m, press(m, "f:ctrl-f"))
		if m.overlay != overlayPicker {
			t.Fatalf("f:ctrl-f must open the picker, overlay=%q", m.overlay)
		}
		m2 := demoModel(120, 32)
		runCmd(t, m2, press(m2, "f:ctrl-o"))
		if m2.mode != modeFloating || m2.activeFloat != "" {
			t.Fatalf("f:ctrl-o must enter floating (no windows): mode=%q active=%q", m2.mode, m2.activeFloat)
		}
	})

	t.Run("pane", func(t *testing.T) {
		m := demoModel(120, 32)
		m.mode = modePane
		tab := m.activeTab()
		before := len(tab.panes)
		runCmd(t, m, press(m, "fs:pane:split-h"))
		if len(tab.panes) != before+1 {
			t.Fatalf("fs:pane:split-h must split like %%: panes %d -> %d", before, len(tab.panes))
		}
		// Close the focused pane (the freshly split one) via the footer node.
		m.mode = modePane
		runCmd(t, m, press(m, "fs:pane:close"))
		if len(tab.panes) != before {
			t.Fatalf("fs:pane:close must close like x: panes %d -> %d", before, len(tab.panes))
		}
		if m.mode != modeLive {
			t.Fatalf("fs:pane:close must leave the scene: mode=%q", m.mode)
		}
	})

	t.Run("resize", func(t *testing.T) {
		m := demoModel(120, 32)
		tab := m.activeTab()
		m.closePane(tab, tab.panes[1])
		runCmd(t, m, m.splitPane("row"))
		sp := m.splitEntries(tab)[0].node
		before := sp.splitFirstExtent(sp.rect.w)
		m.mode = modeResize
		runCmd(t, m, press(m, "fs:resize:left"))
		if got := sp.splitFirstExtent(sp.rect.w); got != before+2 {
			t.Fatalf("fs:resize:left must resize like h: %d -> %d", before, got)
		}
		// Lock and layout are atomic toggles.
		runCmd(t, m, press(m, "fs:resize:lock"))
		if !m.focusPane().locked {
			t.Fatalf("fs:resize:lock must toggle p.locked")
		}
		orient := sp.orient
		runCmd(t, m, press(m, "fs:resize:layout"))
		if sp.orient == orient {
			t.Fatalf("fs:resize:layout must toggle the split orientation")
		}
	})

	t.Run("tab and workspace", func(t *testing.T) {
		m := demoModel(120, 32)
		m.mode = modeTab
		runCmd(t, m, press(m, "fs:tab:next"))
		if m.ws().active != 1 {
			t.Fatalf("fs:tab:next: active=%d want 1", m.ws().active)
		}
		runCmd(t, m, press(m, "fs:tab:prev"))
		if m.ws().active != 0 {
			t.Fatalf("fs:tab:prev: active=%d want 0", m.ws().active)
		}
		m.mode = modeWorkspace
		runCmd(t, m, press(m, "fs:ws:tree"))
		if m.overlay != overlayWorkbenchTree {
			t.Fatalf("fs:ws:tree must open the workbench tree, overlay=%q", m.overlay)
		}
	})

	t.Run("system and floating", func(t *testing.T) {
		m := demoModel(120, 32)
		m.mode = modeSystem
		runCmd(t, m, press(m, "fs:sys:prompt"))
		if m.overlay != overlayPrompt {
			t.Fatalf("fs:sys:prompt must open the prompt, overlay=%q", m.overlay)
		}
		m2 := demoModel(120, 32)
		m2.mode = modeFloating
		runCmd(t, m2, press(m2, "fs:float:new"))
		if len(m2.floatings) != 1 {
			t.Fatalf("fs:float:new must create a floating, got %d", len(m2.floatings))
		}
		f := m2.floatings[0]
		runCmd(t, m2, press(m2, "fs:float:collapse"))
		if !f.collapsed {
			t.Fatalf("fs:float:collapse must collapse the active floating")
		}
		runCmd(t, m2, press(m2, "fs:float:close"))
		if len(m2.floatings) != 0 {
			t.Fatalf("fs:float:close must close the active floating")
		}
	})
}

// TestFooterMergedGroupsAreHints pins the HIGH 1 decision: merged resize group
// tokens (ALIGN/CENTER/PAN) cannot be represented by one click, so they render
// as non-clickable hints while atomic tokens stay clickable.
func TestFooterMergedGroupsAreHints(t *testing.T) {
	for _, node := range []string{"fs:resize:align", "fs:resize:center", "fs:resize:pan"} {
		if !footerHintOnly(node) {
			t.Fatalf("%s must be a hint-only footer token", node)
		}
	}
	for _, node := range []string{"fs:resize:left", "fs:resize:lock", "f:ctrl-p", "fs:pane:close"} {
		if footerHintOnly(node) {
			t.Fatalf("%s must stay clickable", node)
		}
	}
}

// TestPickerRowClickSelectsThenActivates pins HIGH 2: the first click selects a
// picker row, a second click on the already-selected row activates it exactly
// like Enter.
func TestPickerRowClickSelectsThenActivates(t *testing.T) {
	m, fake := boundModel(t)
	m.sources = append(m.sources, &pb.Source{
		Id: "terminal:local:term-9", Kind: "terminal", Title: "term-9",
		Endpoint: "local", TerminalId: "term-9", Attached: false,
	})
	m.openPicker()
	// rows: + New terminal, term-1, term-9 (sorted).
	fake.calls = nil
	runCmd(t, m, press(m, "picker:1"))
	if m.picker != 1 {
		t.Fatalf("first click must select row 1, picker=%d", m.picker)
	}
	if len(fake.calls) != 0 {
		t.Fatalf("first click must not activate: calls=%+v", fake.calls)
	}
	runCmd(t, m, press(m, "picker:1"))
	if m.picker != 1 {
		t.Fatalf("second click kept selection, picker=%d", m.picker)
	}
	if fake.last().method != "terminal.attach" || fake.last().params.GetId() != "term-1" {
		t.Fatalf("second click must attach like Enter: %+v", fake.calls)
	}
	// The + New terminal row opens the create form on activation.
	m.overlay = overlayPicker
	m.picker = 1
	runCmd(t, m, press(m, "picker:0"))
	if m.picker != 0 {
		t.Fatalf("clicking the create row must select it, picker=%d", m.picker)
	}
	runCmd(t, m, press(m, "picker:0"))
	if m.overlay != overlayPrompt || m.promptKind != "terminal.create" {
		t.Fatalf("create-row activation must open terminal.create: %q/%q", m.overlay, m.promptKind)
	}
}

// TestFloatingOwnerKeyWithoutActiveFloating pins HIGH 3: dispatching `a` in the
// floating scene with no active floating must not panic (it used to dereference
// a nil f); it takes ownership of the focused pane instead.
func TestFloatingOwnerKeyWithoutActiveFloating(t *testing.T) {
	m := demoModel(120, 32)
	m.mode = modeFloating
	m.activeFloat = ""
	if len(m.floatings) != 0 {
		t.Fatalf("precondition: expected no floatings")
	}
	// Must not panic.
	runCmd(t, m, key(m, "a"))
	// The demo pane is bound to view:demo, so this is an owner request.
	if m.focusPane().pending != "owner" {
		t.Fatalf("a with no active floating must take ownership of the focused pane, pending=%q", m.focusPane().pending)
	}
}

// TestWorkbenchTreeOverlayNavigateAndJump pins HIGH 6: the TREE tokens open a
// real read-only navigator, ↑/↓ moves the selection and Enter jumps to the
// selected workspace+tab.
func TestWorkbenchTreeOverlayNavigateAndJump(t *testing.T) {
	m := demoModel(120, 32)
	m.mode = modeWorkspace
	runCmd(t, m, key(m, "t"))
	if m.overlay != overlayWorkbenchTree {
		t.Fatalf("workspace t must open the workbench tree, overlay=%q", m.overlay)
	}
	// Demo rows: main/auto-push (0), main/local (1); selection starts on the
	// active tab 0.
	if m.treeSel != 0 {
		t.Fatalf("initial tree selection=%d want 0", m.treeSel)
	}
	runCmd(t, m, m.onKey("down", ""))
	if m.treeSel != 1 {
		t.Fatalf("down must move the tree selection, sel=%d", m.treeSel)
	}
	runCmd(t, m, m.onKey("enter", ""))
	if m.ws().active != 1 {
		t.Fatalf("enter must jump to the selected tab, active=%d want 1", m.ws().active)
	}
	if m.overlay != "" {
		t.Fatalf("jump must close the overlay, overlay=%q", m.overlay)
	}
}

// TestClipboardSceneOwnFooter pins MED 5: the clipboard history overlay is its
// own scene, not the copy scene with its PGUP/PGDN/Y/G keys.
func TestClipboardSceneOwnFooter(t *testing.T) {
	m := demoModel(120, 32)
	m.overlay = overlayClipboard
	if got := m.scene(); got != "clipboard" {
		t.Fatalf("clipboard overlay scene=%q want clipboard", got)
	}
	spec, ok := scenes["clipboard"]
	if !ok {
		t.Fatalf("missing clipboard scene entry")
	}
	labels := ""
	for _, action := range spec.actions {
		labels += action.label + "|"
	}
	for _, want := range []string{"SELECT", "PASTE", "ESC BACK"} {
		if !strings.Contains(labels, want) {
			t.Fatalf("clipboard footer missing %q: %s", want, labels)
		}
	}
	if strings.Contains(labels, "OLDER") || strings.Contains(labels, "PGDN") {
		t.Fatalf("clipboard footer must not advertise copy keys: %s", labels)
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/app"
	"github.com/anytty/anytty/clients/tui/sdk/widgets"
	apipb "github.com/anytty/anytty/proto/access/apipb"
	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
	gproto "google.golang.org/protobuf/proto"
)

// --- in-memory transport (same pattern as clients/tui/sdk/app tests) ---

type pipe struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
}

func newPipe() *pipe {
	p := &pipe{}
	p.cond = sync.NewCond(&p.mu)
	return p
}

func (p *pipe) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0, io.ErrClosedPipe
	}
	p.buf = append(p.buf, b...)
	p.cond.Broadcast()
	return len(b), nil
}

func (p *pipe) Read(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(p.buf) == 0 && !p.closed {
		p.cond.Wait()
	}
	if len(p.buf) == 0 {
		return 0, io.EOF
	}
	n := copy(b, p.buf)
	p.buf = p.buf[n:]
	return n, nil
}

func (p *pipe) Close() error {
	p.mu.Lock()
	p.closed = true
	p.cond.Broadcast()
	p.mu.Unlock()
	return nil
}

// frameResult is one decoded program -> host frame.
type frameResult struct {
	typ     wire.Type
	payload []byte
	err     error
}

// readFrame consumes one frame from the host's single reader pump. Exactly one
// goroutine decodes per fakeHost (see newFakeHost), so helpers never race on
// the wire decoder.
func readFrame(t *testing.T, host *fakeHost, timeout time.Duration) (wire.Type, []byte) {
	t.Helper()
	select {
	case got, ok := <-host.frames:
		if !ok || got.err != nil {
			t.Fatalf("read frame: %v", got.err)
		}
		return got.typ, got.payload
	case <-time.After(timeout):
		t.Fatal("timeout reading frame")
		return 0, nil
	}
}

func decodePayload(t *testing.T, typ wire.Type, payload []byte) any {
	t.Helper()
	msg, err := wire.UnmarshalPayload(typ, payload)
	if err != nil {
		t.Fatalf("decode %v: %v", typ, err)
	}
	return msg
}

// fakeHost is a scripted host for one in-memory sdk.Client: it answers HELLO
// and lets a test read the RESULT frames the model emits.
type fakeHost struct {
	client  *sdk.Client
	encoder *wire.Encoder
	frames  chan frameResult
}

func newFakeHost(t *testing.T) *fakeHost {
	t.Helper()
	toProgram := newPipe()
	fromProgram := newPipe()
	t.Cleanup(func() {
		toProgram.Close()
		fromProgram.Close()
	})
	client := sdk.New(toProgram, fromProgram, sdk.Handlers{})
	go func() { _ = client.Loop() }()
	decoder := wire.NewDecoder(fromProgram, wire.RoleHost, 0)
	host := &fakeHost{
		client:  client,
		encoder: wire.NewEncoder(toProgram, wire.RoleHost, 0),
		frames:  make(chan frameResult, 64),
	}
	go func() {
		for {
			typ, payload, err := decoder.Decode()
			host.frames <- frameResult{typ: typ, payload: payload, err: err}
			if err != nil {
				close(host.frames)
				return
			}
		}
	}()
	if err := host.encoder.Encode(wire.TypeHello, &pb.Hello{Schema: 1, ViewId: "view-1", Epoch: 1, Cols: 100, Rows: 30}); err != nil {
		t.Fatalf("hello: %v", err)
	}
	waitFor(t, "hello processed", func() bool { return client.Hello() != nil })
	return host
}

func (h *fakeHost) respond(t *testing.T, requestID uint64, ok bool, errText string, data *pb.MethodData) {
	t.Helper()
	resp := &pb.Response{RequestId: requestID, Epoch: 1, Ok: ok, Error: errText, Data: data}
	if err := h.encoder.Encode(wire.TypeResponse, resp); err != nil {
		t.Fatalf("response: %v", err)
	}
}

// runEmit executes one command (the model's emit helper blocks until the
// RESPONSE) and returns the RESULT the model emitted plus the message the
// command resolved to.
func runEmit(t *testing.T, host *fakeHost, cmd app.Cmd, data *pb.MethodData) (*pb.Result, app.Msg) {
	t.Helper()
	if cmd == nil {
		t.Fatal("command is nil")
	}
	done := make(chan []app.Msg, 1)
	go func() { done <- app.RunCmd(cmd) }()
	typ, payload := readFrame(t, host, 3*time.Second)
	if typ != wire.TypeResult {
		t.Fatalf("frame = %v, want RESULT", typ)
	}
	result := decodePayload(t, typ, payload).(*pb.Result)
	host.respond(t, result.GetRequestId(), true, "", data)
	select {
	case msgs := <-done:
		return result, firstMsg(msgs)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for the command result")
		return nil, nil
	}
}

// firstMsg returns the first non-nil message of a RunCmd result (the command
// response; a trailing keysMsg from a batched claim refresh is ignored here).
func firstMsg(msgs []app.Msg) app.Msg {
	for _, msg := range msgs {
		if msg != nil {
			return msg
		}
	}
	return nil
}

// runEmitAll runs cmd through app.RunCmd and answers every RESULT it emits
// (batch sub-commands run sequentially), returning the methods in order.
func runEmitAll(t *testing.T, host *fakeHost, cmd app.Cmd) []string {
	t.Helper()
	done := make(chan struct{})
	go func() { app.RunCmd(cmd); close(done) }()
	var methods []string
	for {
		select {
		case <-done:
			return methods
		default:
		}
		var typ wire.Type
		var payload []byte
		select {
		case got, ok := <-host.frames:
			if !ok {
				t.Fatalf("frame pump closed: %v", got.err)
			}
			typ, payload = got.typ, got.payload
		case <-done:
			return methods
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for the command batch")
		}
		if typ != wire.TypeResult {
			continue
		}
		result := decodePayload(t, typ, payload).(*pb.Result)
		methods = append(methods, result.GetMethod())
		host.respond(t, result.GetRequestId(), true, "", nil)
	}
}

// runEmitExpect is runEmit with a failure RESPONSE.
func runEmitExpect(t *testing.T, host *fakeHost, cmd app.Cmd, data *pb.MethodData, ok bool, errText string) (*pb.Result, app.Msg) {
	t.Helper()
	if cmd == nil {
		t.Fatal("command is nil")
	}
	done := make(chan []app.Msg, 1)
	go func() { done <- app.RunCmd(cmd) }()
	typ, payload := readFrame(t, host, 3*time.Second)
	if typ != wire.TypeResult {
		t.Fatalf("frame = %v, want RESULT", typ)
	}
	result := decodePayload(t, typ, payload).(*pb.Result)
	host.respond(t, result.GetRequestId(), ok, errText, data)
	select {
	case msgs := <-done:
		return result, firstMsg(msgs)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for the command result")
		return nil, nil
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

// --- fixtures and helpers ---

func terminalSource(id, title, endpoint string) *pb.Source {
	src := &pb.Source{Id: id, Kind: "terminal", Title: title, Endpoint: endpoint, Health: "ok", Attached: true}
	if parts := strings.SplitN(id, ":", 3); len(parts) == 3 {
		src.TerminalId = parts[2]
	}
	return src
}

func helloMsg(viewID string) app.HelloMsg {
	return app.HelloMsg{Hello: &pb.Hello{Schema: 1, ViewId: viewID, Epoch: 1, Cols: 100, Rows: 30}}
}

func key(m *model, k string) app.Cmd {
	return m.Update(app.KeyMsg{Key: &pb.KeyEvent{Key: k}})
}

func mouse(button, action, node string, x, y int) app.MouseMsg {
	return app.MouseMsg{Mouse: &pb.MouseEvent{Action: action, Button: button, Node: node, X: int32(x), Y: int32(y)}}
}

func feed(m *model, items ...*pb.Source) app.Cmd {
	return m.Update(app.SourcesMsg{Items: items})
}

// seededModel returns a model with the default workspace/tab/pane and a HELLO.
func seededModel(t *testing.T) *model {
	t.Helper()
	m := newModel(nil, &app.Memo{})
	m.Update(helloMsg("view-1"))
	m.Init()
	if m.currentPaneRef() != "w1:t1:p1" {
		t.Fatalf("seed = %q, want w1:t1:p1", m.currentPaneRef())
	}
	return m
}

// bindSource feeds a snapshot and binds one source to the focused pane.
func bindSource(t *testing.T, m *model, src *pb.Source) {
	t.Helper()
	feed(m, src)
	_, _, p := m.focusedPane()
	if p == nil {
		t.Fatal("no focused pane")
	}
	p.Source = src.GetId()
	m.touch()
}

func boxTexts(box *pb.Box) []string {
	if box == nil {
		return nil
	}
	var out []string
	if text := box.GetContent().GetText(); text != "" {
		out = append(out, text)
	}
	out = append(out, box.GetContent().GetLines()...)
	for _, child := range box.GetChildren() {
		out = append(out, boxTexts(child)...)
	}
	return out
}

func allText(box *pb.Box) string { return strings.Join(boxTexts(box), "\n") }

func findBox(box *pb.Box, id string) *pb.Box {
	if box == nil {
		return nil
	}
	if box.GetId() == id {
		return box
	}
	for _, child := range box.GetChildren() {
		if found := findBox(child, id); found != nil {
			return found
		}
	}
	return nil
}

func prefix(m *model, k string) {
	key(m, "ctrl-b")
	key(m, k)
}

// --- workspace / tab / pane CRUD ---

func TestWorkspaceTabPaneCRUD(t *testing.T) {
	m := seededModel(t)
	if m.workspaceByID("w1") == nil || m.currentTab().ID != "t1" {
		t.Fatalf("seed layout = %+v", m.layoutDoc())
	}

	// N: new workspace with one tab/pane, activated.
	prefix(m, "N")
	if len(m.workspaces) != 2 || m.activeWorkspace != "w2" {
		t.Fatalf("workspaces/active = %d/%s, want 2/w2", len(m.workspaces), m.activeWorkspace)
	}
	if m.currentPaneRef() != "w2:t1:p1" {
		t.Fatalf("ref after N = %q", m.currentPaneRef())
	}

	// c: new tab in the active workspace.
	prefix(m, "c")
	if len(m.currentWS().Tabs) != 2 || m.currentWS().ActiveTab != "t2" {
		t.Fatalf("tabs/active = %d/%s, want 2/t2", len(m.currentWS().Tabs), m.currentWS().ActiveTab)
	}
	if m.currentPaneRef() != "w2:t2:p1" {
		t.Fatalf("ref after c = %q", m.currentPaneRef())
	}

	// v: split the tab into two panes with equal weights and focus the new one.
	prefix(m, "v")
	tab := m.currentTab()
	if len(tab.Panes) != 2 || tab.Axis != "row" {
		t.Fatalf("split = %d panes axis %q", len(tab.Panes), tab.Axis)
	}
	if !sameInts(tab.Weights, []int{50, 50}) {
		t.Fatalf("weights = %v, want [50 50]", tab.Weights)
	}
	if tab.Focus != "p2" {
		t.Fatalf("focus = %q, want p2", tab.Focus)
	}

	// x: close the focused pane.
	prefix(m, "x")
	if len(m.currentTab().Panes) != 1 || m.currentTab().Focus != "p1" {
		t.Fatalf("after x: panes/focus = %d/%s", len(m.currentTab().Panes), m.currentTab().Focus)
	}
	if !sameInts(m.currentTab().Weights, []int{100}) {
		t.Fatalf("weights after close = %v", m.currentTab().Weights)
	}

	// X: close the tab.
	prefix(m, "X")
	if len(m.currentWS().Tabs) != 1 || m.currentWS().ActiveTab != "t1" {
		t.Fatalf("after X: tabs/active = %d/%s", len(m.currentWS().Tabs), m.currentWS().ActiveTab)
	}

	// D: close the workspace, then D again on the last one re-seeds.
	prefix(m, "D")
	if len(m.workspaces) != 1 || m.activeWorkspace != "w1" {
		t.Fatalf("after D: %d workspaces active %s", len(m.workspaces), m.activeWorkspace)
	}
	prefix(m, "D")
	if len(m.workspaces) != 1 || m.currentPaneRef() != "w1:t1:p1" {
		t.Fatalf("after last D: workspaces=%d ref=%s", len(m.workspaces), m.currentPaneRef())
	}

	// IDs never collide after a re-seed.
	prefix(m, "N")
	if len(m.workspaces) != 2 || m.workspaces[0].ID == m.workspaces[1].ID {
		t.Fatalf("duplicate workspace id: %+v", m.workspaces)
	}
}

func TestClosePaneKeepsTabAndFocus(t *testing.T) {
	m := seededModel(t)
	prefix(m, "v")
	prefix(m, "v")
	tab := m.currentTab()
	if len(tab.Panes) != 3 || !sameInts(tab.Weights, []int{34, 33, 33}) {
		t.Fatalf("3-way split = %d %v", len(tab.Panes), tab.Weights)
	}
	// Focus p3 and close it: focus falls back to the previous pane.
	if tab.Focus != "p3" {
		t.Fatalf("focus = %s", tab.Focus)
	}
	prefix(m, "x")
	if len(tab.Panes) != 2 || tab.Focus != "p2" {
		t.Fatalf("after close: panes=%d focus=%s", len(tab.Panes), tab.Focus)
	}
	if sumInts(tab.Weights) != 100 {
		t.Fatalf("weights = %v, want sum 100", tab.Weights)
	}
}

// --- split / focus / swap ---

func TestFocusMoveSwapAndCycle(t *testing.T) {
	m := seededModel(t)
	prefix(m, "v")

	prefix(m, "h")
	if m.currentTab().Focus != "p1" {
		t.Fatalf("h focus = %s, want p1", m.currentTab().Focus)
	}
	prefix(m, "h") // clamped at the left edge
	if m.currentTab().Focus != "p1" {
		t.Fatalf("h at edge focus = %s", m.currentTab().Focus)
	}
	prefix(m, "l")
	if m.currentTab().Focus != "p2" {
		t.Fatalf("l focus = %s, want p2", m.currentTab().Focus)
	}
	prefix(m, "k") // row axis: vertical direction is not a pane move
	if m.currentTab().Focus != "p2" {
		t.Fatalf("k on a row split changed focus to %s", m.currentTab().Focus)
	}

	// H swaps with the previous pane; the focused pane keeps focus.
	prefix(m, "H")
	tab := m.currentTab()
	if tab.Panes[0].ID != "p2" || tab.Panes[1].ID != "p1" {
		t.Fatalf("after H: %s %s", tab.Panes[0].ID, tab.Panes[1].ID)
	}
	if tab.Focus != "p2" {
		t.Fatalf("focus after swap = %s, want p2", tab.Focus)
	}
	prefix(m, "L")
	if tab.Panes[0].ID != "p1" || tab.Panes[1].ID != "p2" {
		t.Fatalf("after L: %s %s", tab.Panes[0].ID, tab.Panes[1].ID)
	}

	prefix(m, "tab")
	if tab.Focus != "p1" {
		t.Fatalf("tab cycle focus = %s, want p1", tab.Focus)
	}
	prefix(m, "shift-tab")
	if tab.Focus != "p2" {
		t.Fatalf("shift-tab cycle focus = %s, want p2", tab.Focus)
	}

	// A column split uses k/j for pane movement.
	prefix(m, "c")
	prefix(m, "-")
	col := m.currentTab()
	if col.Axis != "col" || len(col.Panes) != 2 {
		t.Fatalf("col split = %q %d panes", col.Axis, len(col.Panes))
	}
	if col.Focus != "p2" {
		t.Fatalf("col focus = %s", col.Focus)
	}
	prefix(m, "k")
	if col.Focus != "p1" {
		t.Fatalf("col k focus = %s, want p1", col.Focus)
	}
	prefix(m, "j")
	prefix(m, "K")
	if col.Panes[0].ID != "p2" {
		t.Fatalf("col K swap = %s %s", col.Panes[0].ID, col.Panes[1].ID)
	}
}

// --- resize mode ---

func TestResizeModeClampsToFivePercent(t *testing.T) {
	m := seededModel(t)
	prefix(m, "v")
	prefix(m, "r")
	if m.mode != modeResize {
		t.Fatalf("mode = %q, want resize", m.mode)
	}
	tab := m.currentTab()
	// l pushes the divider right until the left pane hits 95%.
	for i := 0; i < 40; i++ {
		key(m, "l")
	}
	if tab.Weights[0] != maxPaneWeight || tab.Weights[1] != minPaneWeight {
		t.Fatalf("l clamp = %v, want [95 5]", tab.Weights)
	}
	for i := 0; i < 40; i++ {
		key(m, "h")
	}
	if tab.Weights[0] != minPaneWeight || tab.Weights[1] != maxPaneWeight {
		t.Fatalf("h clamp = %v, want [5 95]", tab.Weights)
	}
	if sumInts(tab.Weights) != 100 {
		t.Fatalf("weights do not sum to 100: %v", tab.Weights)
	}
	key(m, "enter")
	if m.mode != modeTerminal {
		t.Fatalf("mode after enter = %q", m.mode)
	}

	// Divider direction semantics on a fresh split: h moves the divider left
	// (the left pane shrinks) and l moves it right.
	m2 := seededModel(t)
	prefix(m2, "v")
	prefix(m2, "r")
	tab2 := m2.currentTab()
	key(m2, "h")
	if !sameInts(tab2.Weights, []int{45, 55}) {
		t.Fatalf("h weights = %v, want [45 55]", tab2.Weights)
	}
	key(m2, "l")
	if !sameInts(tab2.Weights, []int{50, 50}) {
		t.Fatalf("l weights = %v, want [50 50]", tab2.Weights)
	}
	key(m2, "l")
	if !sameInts(tab2.Weights, []int{55, 45}) {
		t.Fatalf("l weights = %v, want [55 45]", tab2.Weights)
	}
	key(m2, "esc")
	if m2.mode != modeTerminal {
		t.Fatalf("mode after esc = %q", m2.mode)
	}
}

// --- prefix state machine ---

func TestPrefixModeStateMachine(t *testing.T) {
	m := seededModel(t)
	if m.mode != modeTerminal {
		t.Fatalf("initial mode = %q", m.mode)
	}
	key(m, "ctrl-b")
	if m.mode != modePrefix {
		t.Fatalf("after ctrl+b mode = %q", m.mode)
	}
	if keys := m.keys(); !keys.All {
		t.Fatalf("prefix claim = %+v, want All", keys)
	}
	key(m, "Z") // unknown action: ignored, prefix consumed
	if m.mode != modeTerminal {
		t.Fatalf("unknown prefix key left mode %q", m.mode)
	}

	key(m, "ctrl-b")
	key(m, "esc")
	if m.mode != modeTerminal {
		t.Fatalf("esc left mode %q", m.mode)
	}

	// c is a new tab, not a terminal-created pane.
	prefix(m, "c")
	if len(m.currentWS().Tabs) != 2 {
		t.Fatalf("tabs after prefix c = %d", len(m.currentWS().Tabs))
	}

	// z on a single-pane tab toasts instead of zooming.
	prefix(m, "z")
	if m.zoom || !strings.Contains(m.toast, "nothing to zoom") {
		t.Fatalf("zoom on single pane = %v toast %q", m.zoom, m.toast)
	}

	// z zooms on a split tab and the tab bar advertises it.
	prefix(m, "v")
	prefix(m, "z")
	if !m.zoom {
		t.Fatal("zoom did not turn on")
	}
	if !strings.Contains(allText(m.View()), "ZOOM") {
		t.Fatalf("view is missing the ZOOM marker:\n%s", allText(m.View()))
	}
	prefix(m, "z")
	if m.zoom {
		t.Fatal("zoom did not turn off")
	}

	// b toggles the sidebar and is persisted.
	prefix(m, "b")
	if !m.sidebarCollapsed {
		t.Fatal("sidebar did not collapse")
	}
	if findBox(m.View(), "herdr.sidebar.spaces") != nil {
		t.Fatal("collapsed sidebar still renders")
	}
	prefix(m, "b")
	if m.sidebarCollapsed {
		t.Fatal("sidebar did not expand")
	}
}

// --- navigate mode ---

func TestNavigateModeSelectsAndActivatesWorkspace(t *testing.T) {
	m := seededModel(t)
	prefix(m, "N")
	prefix(m, "N")
	if m.activeWorkspace != "w3" || len(m.workspaces) != 3 {
		t.Fatalf("setup = %d workspaces active %s", len(m.workspaces), m.activeWorkspace)
	}
	prefix(m, "w")
	if m.mode != modeNavigate || m.navWS != 2 {
		t.Fatalf("navigate = %q navWS=%d", m.mode, m.navWS)
	}
	// The selected workspace row uses the selection style.
	row := findBox(m.View(), spaceRowPrefix+"w3")
	if row == nil || row.GetStyle() != widgets.StyleSelection {
		t.Fatalf("navigate selection row = %+v", row)
	}
	key(m, "up")
	key(m, "up")
	if m.navWS != 0 {
		t.Fatalf("navWS after up = %d", m.navWS)
	}
	key(m, "up") // clamped
	if m.navWS != 0 {
		t.Fatalf("navWS clamped = %d", m.navWS)
	}
	key(m, "enter")
	if m.mode != modeTerminal || m.activeWorkspace != "w1" {
		t.Fatalf("enter activate = %q/%s", m.mode, m.activeWorkspace)
	}
	prefix(m, "w")
	key(m, "down")
	key(m, "enter")
	if m.activeWorkspace != "w2" {
		t.Fatalf("down+enter = %s, want w2", m.activeWorkspace)
	}
	prefix(m, "w")
	key(m, "esc")
	if m.mode != modeTerminal {
		t.Fatalf("navigate esc = %q", m.mode)
	}
}

// --- scroll mode ---

func TestScrollModeStateMachine(t *testing.T) {
	host := newFakeHost(t)
	m := newModel(host.client, &app.Memo{})
	m.Update(helloMsg("view-1"))
	m.Init()
	m.layoutReady = true
	alpha := terminalSource("terminal:local:alpha", "alpha", "local")
	bindSource(t, m, alpha)

	prefix(m, "[")
	if m.mode != modeScroll {
		t.Fatalf("mode = %q, want scroll", m.mode)
	}
	if keys := m.keys(); !keys.All {
		t.Fatalf("scroll claim = %+v, want All", keys)
	}
	cmd := key(m, "k")
	if cmd == nil || m.scrollLines != 1 {
		t.Fatalf("k = cmd %v lines %d", cmd != nil, m.scrollLines)
	}
	result, msg := runEmit(t, host, cmd, nil)
	if result.GetMethod() != "terminal.scroll" {
		t.Fatalf("method = %q, want terminal.scroll", result.GetMethod())
	}
	params := result.GetParams()
	if params.GetEndpoint() != "local" || params.GetId() != "alpha" || params.GetDelta() != 1 {
		t.Fatalf("scroll params = %+v", params)
	}
	m.Update(msg)

	// page-up uses the body height as the page size.
	cmd = key(m, "page-up")
	if cmd == nil {
		t.Fatal("page-up emitted nothing")
	}
	result, msg = runEmit(t, host, cmd, nil)
	if result.GetMethod() != "terminal.scroll" || result.GetParams().GetDelta() != int32(m.scrollPage()) {
		t.Fatalf("page-up = %q %+v", result.GetMethod(), result.GetParams())
	}
	m.Update(msg)
	if m.scrollLines != 1+m.scrollPage() {
		t.Fatalf("scrollLines = %d", m.scrollLines)
	}

	// y copies the visible area and keeps scroll mode.
	cmd = key(m, "y")
	result, msg = runEmit(t, host, cmd, nil)
	if result.GetMethod() != "terminal.copy" {
		t.Fatalf("y method = %q, want terminal.copy", result.GetMethod())
	}
	m.Update(msg)
	if m.mode != modeScroll {
		t.Fatalf("copy left scroll mode: %q", m.mode)
	}

	// page-down back to one line above live, then j exits scroll mode.
	cmd = key(m, "page-down")
	if cmd == nil || m.scrollLines != 1 {
		t.Fatalf("page-down = cmd %v lines %d", cmd != nil, m.scrollLines)
	}
	result, msg = runEmit(t, host, cmd, nil)
	if result.GetMethod() != "terminal.scroll" || result.GetParams().GetDelta() != -int32(m.scrollPage()) {
		t.Fatalf("page-down = %q %+v", result.GetMethod(), result.GetParams())
	}
	m.Update(msg)
	// j back to offset 0 exits scroll mode and must emit terminal.scrollEnd
	// (the emit pair is a batch, so answer every RESULT it produces).
	cmd = key(m, "j")
	if cmd == nil {
		t.Fatal("j emitted nothing")
	}
	if m.mode != modeTerminal || m.scrollLines != 0 {
		t.Fatalf("j to live = %q/%d", m.mode, m.scrollLines)
	}
	methods := runEmitAll(t, host, cmd)
	foundEnd := false
	for _, method := range methods {
		if method == "terminal.scrollEnd" {
			foundEnd = true
		}
	}
	if !foundEnd {
		t.Fatalf("j to live methods = %v, want terminal.scrollEnd", methods)
	}
	// A j at live is ignored.
	if cmd := key(m, "j"); cmd != nil {
		t.Fatal("j at live emitted something")
	}

	// q emits scrollEnd and leaves.
	prefix(m, "[")
	cmd = key(m, "q")
	if cmd == nil || m.mode != modeTerminal {
		t.Fatalf("q = cmd %v mode %q", cmd != nil, m.mode)
	}
	result, msg = runEmit(t, host, m.scrollEndCmd(), nil)
	if result.GetMethod() != "terminal.scrollEnd" {
		t.Fatalf("scrollEnd method = %q", result.GetMethod())
	}
	m.Update(msg)

	// No terminal: entering scroll is refused with a toast.
	m2 := seededModel(t)
	prefix(m2, "[")
	if m2.mode != modeTerminal || !strings.Contains(m2.toast, "no terminal to scroll") {
		t.Fatalf("empty scroll = %q toast %q", m2.mode, m2.toast)
	}
}

// --- sources reconcile / rollup / badges / gone / offline ---

func TestSourcesReconcileRollupAndBadges(t *testing.T) {
	m := seededModel(t)
	prefix(m, "v")
	prefix(m, "v")
	prefix(m, "v")
	tab := m.currentTab()
	if len(tab.Panes) != 4 {
		t.Fatalf("panes = %d, want 4", len(tab.Panes))
	}
	you := terminalSource("terminal:local:you", "you", "local")
	you.ResizeOwner = "view-1"
	other := terminalSource("terminal:local:other", "other", "local")
	other.ResizeOwner, other.OwnerEpoch = "view-9", 4
	exited := terminalSource("terminal:local:exited", "exited", "local")
	exited.Exited, exited.ExitCode = true, 3
	sick := terminalSource("terminal:local:sick", "sick", "local")
	sick.Health = "degraded"

	tab.Panes[0].Source = you.GetId()
	tab.Panes[1].Source = other.GetId()
	tab.Panes[2].Source = exited.GetId()
	tab.Panes[3].Source = sick.GetId()
	feed(m, you, other, exited, sick)

	if got := m.paneState(tab.Panes[0]); got != stateRunning {
		t.Fatalf("you state = %v", got)
	}
	if got := m.paneState(tab.Panes[1]); got != stateRunning {
		t.Fatalf("other state = %v", got)
	}
	if got := m.paneState(tab.Panes[2]); got != stateExited {
		t.Fatalf("exited state = %v", got)
	}
	if got := m.paneState(tab.Panes[3]); got != stateOffline {
		t.Fatalf("sick state = %v", got)
	}
	if m.rollup(m.currentWS()) != stateOffline {
		t.Fatalf("rollup = %v, want offline (offline > exited > running)", m.rollup(m.currentWS()))
	}
	if !strings.Contains(m.stateText(tab.Panes[2]), "exited(3)") {
		t.Fatalf("stateText = %q", m.stateText(tab.Panes[2]))
	}

	text := allText(m.View())
	for _, want := range []string{"[you]", "[other]", "[exit 3]", "[degraded]", "you", "other", "sick"} {
		if !strings.Contains(text, want) {
			t.Fatalf("view is missing %q:\n%s", want, text)
		}
	}

	// Removing a snapshot row marks the pane gone.
	feed(m, you, other, exited)
	if got := m.paneState(tab.Panes[3]); got != stateGone {
		t.Fatalf("missing source state = %v, want gone", got)
	}
	if !strings.Contains(allText(m.View()), "source gone") {
		t.Fatalf("gone pane is not rendered:\n%s", allText(m.View()))
	}
	// gone ranks between offline and exited; exited+pane-missing rolls up to
	// the gone tier.
	if m.rollup(m.currentWS()) != stateGone {
		t.Fatalf("rollup with a missing pane = %v, want gone", m.rollup(m.currentWS()))
	}

	// Snapshot order does not matter: the key is built from the model, not
	// from the wire order.
	keyOnce := m.sidebarKey()
	feed(m, exited, other, you)
	if got := m.sidebarKey(); got != keyOnce {
		t.Fatalf("reconcile is order dependent:\n%s\n%s", keyOnce, got)
	}

	// exited + running rolls up to exited once the missing pane is unbound.
	tab.Panes[3].Source = ""
	feed(m, you, other, exited)
	if got := m.rollup(m.currentWS()); got != stateExited {
		t.Fatalf("rollup = %v, want exited", got)
	}
	// running only.
	tab.Panes[2].Source = ""
	feed(m, you, other)
	if got := m.rollup(m.currentWS()); got != stateRunning {
		t.Fatalf("rollup = %v, want running", got)
	}
	// A workspace of empty panes rolls up to empty.
	m2 := seededModel(t)
	if got := m2.rollup(m2.currentWS()); got != stateEmpty {
		t.Fatalf("empty rollup = %v", got)
	}
}

func TestSourcesReconcileAutoAttachesUnattachedPane(t *testing.T) {
	host := newFakeHost(t)
	m := newModel(host.client, &app.Memo{})
	m.Update(helloMsg("view-1"))
	m.Init()
	m.layoutReady = true
	_, _, p := m.focusedPane()
	p.Source = "terminal:prod:web"

	remote := terminalSource("terminal:prod:web", "web", "prod")
	remote.Attached = false
	remote.ResizeOwner = ""
	cmd := feed(m, remote)
	if cmd == nil {
		t.Fatal("an unattached bound source did not emit terminal.attach")
	}
	// A repeated snapshot does not re-attach while the first is in flight.
	if again := feed(m, remote); again != nil {
		t.Fatal("duplicate snapshot emitted a second attach")
	}
	result, msg := runEmit(t, host, cmd, nil)
	if result.GetMethod() != "terminal.attach" {
		t.Fatalf("method = %q, want terminal.attach", result.GetMethod())
	}
	if params := result.GetParams(); params.GetEndpoint() != "prod" || params.GetId() != "web" || !params.GetFit() {
		t.Fatalf("attach params = %+v, want prod/web/fit", params)
	}
	m.Update(msg)
}

// --- persistence ---

func TestLayoutJSONRoundTrip(t *testing.T) {
	m := seededModel(t)
	alpha := terminalSource("terminal:local:alpha", "alpha", "local")
	bindSource(t, m, alpha)

	prefix(m, "N") // w2
	prefix(m, "W") // rename workspace: clear the prefilled name first
	key(m, "ctrl-u")
	key(m, "ctrl-u")
	for _, r := range "logs" {
		key(m, string(r))
	}
	key(m, "enter")
	prefix(m, "c") // w2:t2
	prefix(m, "T")
	key(m, "ctrl-u")
	key(m, "ctrl-u")
	for _, r := range "build" {
		key(m, string(r))
	}
	key(m, "enter")
	prefix(m, "v") // split
	prefix(m, "r")
	key(m, "l")
	key(m, "l")
	key(m, "l")
	key(m, "enter")
	prefix(m, "b") // collapse the sidebar

	if m.workspaceByID("w2").Name != "logs" {
		t.Fatalf("workspace rename = %q", m.workspaceByID("w2").Name)
	}
	if t2 := tabByID(m.workspaceByID("w2"), "t2"); t2 == nil || t2.Name != "build" {
		t.Fatalf("tab rename = %+v", t2)
	}
	want := m.layoutDoc()
	if !sameInts(want.Workspaces[1].Tabs[1].Weights, []int{65, 35}) {
		t.Fatalf("resized weights = %v", want.Workspaces[1].Tabs[1].Weights)
	}
	if want.Workspaces[1].Tabs[1].Focus != "p2" {
		t.Fatalf("focus = %q", want.Workspaces[1].Tabs[1].Focus)
	}
	if !want.SidebarCollapsed {
		t.Fatal("sidebar collapse not captured")
	}

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back layoutDoc
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(want, back) {
		t.Fatalf("json round trip mismatch:\n%+v\n%+v", want, back)
	}

	fresh := newModel(nil, &app.Memo{})
	fresh.Update(helloMsg("view-1"))
	fresh.Init()
	if !fresh.applyLayoutDoc(back) {
		t.Fatal("applyLayoutDoc rejected a valid document")
	}
	if got := fresh.layoutDoc(); !reflect.DeepEqual(got, want) {
		t.Fatalf("apply mismatch:\n%+v\n%+v", got, want)
	}
	// Sources still reconcile by id, including the restored binding.
	feed(fresh, alpha)
	ws, tab, pane := fresh.findPane("w1:t1:p1")
	if ws == nil || tab == nil || pane == nil || pane.Source != alpha.GetId() {
		t.Fatalf("restored binding lost: %+v", pane)
	}
	if got := fresh.paneState(pane); got != stateRunning {
		t.Fatalf("restored pane state = %v", got)
	}
}

func TestLayoutRestoreNormalizesWeights(t *testing.T) {
	doc := layoutDoc{
		Version:         1,
		ActiveWorkspace: "w9",
		Workspaces: []layoutWorkspace{{
			ID: "w9", Name: "odd", ActiveTab: "t3",
			Tabs: []layoutTab{{
				ID: "t3", Name: "odd", Axis: "weird", Weights: []int{1, 999},
				Focus: "pmissing",
				Panes: []layoutPane{{ID: "pa"}, {ID: "pb"}},
			}},
		}},
	}
	m := newModel(nil, &app.Memo{})
	if !m.applyLayoutDoc(doc) {
		t.Fatal("apply rejected the document")
	}
	tab := m.currentTab()
	if tab.Axis != "row" {
		t.Fatalf("axis = %q, want row", tab.Axis)
	}
	if sumInts(tab.Weights) != 100 || tab.Weights[0] < minPaneWeight || tab.Weights[1] > maxPaneWeight {
		t.Fatalf("weights = %v", tab.Weights)
	}
	if tab.Focus != "pa" {
		t.Fatalf("focus repaired to %q, want pa", tab.Focus)
	}
	if m.activeWorkspace != "w9" {
		t.Fatalf("active = %q", m.activeWorkspace)
	}
}

func TestPersistenceSaveEmitAndCoalescing(t *testing.T) {
	host := newFakeHost(t)
	m := newModel(host.client, &app.Memo{})
	m.Update(helloMsg("view-1"))
	m.Init()
	m.layoutReady = true
	if m.saveLayoutCmd() != nil {
		t.Fatal("clean layout must not save")
	}

	alpha := terminalSource("terminal:local:alpha", "alpha", "local")
	bindSource(t, m, alpha)
	m.split("row")
	if !m.layoutDirty {
		t.Fatal("split did not mark the layout dirty")
	}
	cmd := m.saveLayoutCmd()
	if cmd == nil {
		t.Fatal("dirty layout did not produce a save command")
	}
	if m.saveLayoutCmd() != nil {
		t.Fatal("a second save started while one is in flight")
	}
	result, msg := runEmit(t, host, cmd, nil)
	if result.GetMethod() != "access.call" {
		t.Fatalf("method = %q, want access.call", result.GetMethod())
	}
	envelope := storageCommand(t, result.GetParams().GetAccessCommand())
	put := envelope.GetStoragePut()
	if put == nil {
		t.Fatalf("command is not a storage put: %+v", envelope)
	}
	key := put.GetKey()
	if key.GetAppId() != herdrAppID || key.GetKey() != layoutKey || key.GetScope() != apipb.StorageScope_STORAGE_SCOPE_PRIVATE {
		t.Fatalf("storage key = %+v", key)
	}
	var stored layoutDoc
	if err := json.Unmarshal(put.GetValue(), &stored); err != nil {
		t.Fatalf("stored layout is not json: %v", err)
	}
	if len(stored.Workspaces) != 1 || len(stored.Workspaces[0].Tabs) != 1 || len(stored.Workspaces[0].Tabs[0].Panes) != 2 {
		t.Fatalf("stored layout = %+v", stored)
	}
	if !sameInts(stored.Workspaces[0].Tabs[0].Weights, []int{50, 50}) {
		t.Fatalf("stored weights = %v", stored.Workspaces[0].Tabs[0].Weights)
	}
	if stored.Workspaces[0].Tabs[0].Panes[1].Source != "" {
		t.Fatalf("untouched pane must stay unbound: %+v", stored.Workspaces[0].Tabs[0].Panes[1])
	}
	m.Update(msg)
	if m.savePending {
		t.Fatal("savePending was not cleared by the response")
	}
}

func storageCommand(t *testing.T, payload []byte) *apipb.CommandEnvelope {
	t.Helper()
	var envelope apipb.CommandEnvelope
	if err := gproto.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("unmarshal access command: %v", err)
	}
	return &envelope
}

func TestStorageErrorsAndLegacyPinHint(t *testing.T) {
	m := seededModel(t)
	m.onStorage(storageMsg{op: "get", key: layoutKey, err: "access unavailable"})
	if !strings.Contains(m.toast, "layout restore failed") || !m.toastErr {
		t.Fatalf("layout get error toast = %q (err=%v)", m.toast, m.toastErr)
	}
	if !m.layoutReady {
		t.Fatal("a failed load must still open the save gate")
	}

	// Legacy pinned hint binds the first empty pane once the source appears.
	m.onStorage(storageMsg{op: "get", key: legacyPinKey, ok: true, value: "terminal:local:alpha"})
	if m.pendingPin != "terminal:local:alpha" {
		t.Fatalf("pendingPin = %q", m.pendingPin)
	}
	feed(m, terminalSource("terminal:local:alpha", "alpha", "local"))
	if m.currentPaneSource() != "terminal:local:alpha" {
		t.Fatalf("pane source = %q, want the pinned hint", m.currentPaneSource())
	}

	// Save failures surface the host reason and keep the layout dirty.
	m.layoutDirty = true
	m.savePending = true
	m.onStorage(storageMsg{op: "set", key: layoutKey, err: "no access"})
	if !strings.Contains(m.toast, "layout save failed: no access") {
		t.Fatalf("save error toast = %q", m.toast)
	}
	if !m.layoutDirty {
		t.Fatal("a failed save must keep the layout dirty for a retry")
	}
}

// --- generated commands ---

func TestCreateAndAttachMethods(t *testing.T) {
	host := newFakeHost(t)
	m := newModel(host.client, &app.Memo{})
	m.Update(helloMsg("view-1"))
	m.Init()
	m.layoutReady = true

	// enter on the empty pane creates a local terminal.
	cmd := key(m, "enter")
	if cmd == nil {
		t.Fatal("enter on an empty pane emitted nothing")
	}
	result, msg := runEmit(t, host, cmd, &pb.MethodData{Endpoint: "local", Id: "fresh"})
	if result.GetMethod() != "terminal.create" || result.GetParams().GetEndpoint() != "local" {
		t.Fatalf("create = %q %+v", result.GetMethod(), result.GetParams())
	}
	m.Update(msg)
	if got := m.currentPaneSource(); got != "terminal:local:fresh" {
		t.Fatalf("bound source = %q", got)
	}
	if !strings.Contains(m.toast, "created terminal:local:fresh") {
		t.Fatalf("toast = %q", m.toast)
	}
	// The host publishes the created terminal in the next snapshot.
	feed(m, terminalSource("terminal:local:fresh", "fresh", "local"))
	if got := m.paneState(m.currentTab().Panes[0]); got != stateRunning {
		t.Fatalf("created pane state = %v", got)
	}

	// enter on a running pane does nothing.
	if cmd := key(m, "enter"); cmd != nil {
		t.Fatalf("enter on a live pane emitted %v", cmd)
	}

	// Create failures toast the host reason.
	prefix(m, "v")
	result, msg = runEmitExpect(t, host, key(m, "enter"), nil, false, "no pool")
	if result.GetMethod() != "terminal.create" {
		t.Fatalf("method = %q", result.GetMethod())
	}
	m.Update(msg)
	if !strings.Contains(m.toast, "create failed: no pool") {
		t.Fatalf("create failure toast = %q", m.toast)
	}

	// Attach CAS conflict falls back to fit=false.
	sick := terminalSource("terminal:local:shared", "shared", "local")
	sick.Attached, sick.ResizeOwner, sick.OwnerEpoch = true, "view-9", 7
	_, _, p := m.focusedPane()
	p.Source = sick.GetId()
	m.sources[sick.GetId()] = sick
	cmd = m.attachParamsCmd("w1:t1:p2", sick, true)
	result, msg = runEmitExpect(t, host, cmd, nil, false, "owner conflict")
	if params := result.GetParams(); params.GetExpectedOwnerEpoch() != 7 {
		t.Fatalf("CAS epoch = %d", params.GetExpectedOwnerEpoch())
	}
	cmd = m.Update(msg)
	result, msg = runEmit(t, host, cmd, nil)
	if result.GetMethod() != "terminal.attach" || result.GetParams().GetFit() {
		t.Fatalf("follow attach = %q fit=%v", result.GetMethod(), result.GetParams().GetFit())
	}
	m.Update(msg)
	if m.pendingAttach[sick.GetId()] {
		t.Fatal("pendingAttach was not cleared")
	}
}

func TestQuitCmdUsesSystemQuit(t *testing.T) {
	host := newFakeHost(t)
	m := newModel(host.client, &app.Memo{})
	m.Update(helloMsg("view-1"))
	result, _ := runEmit(t, host, m.quitCmd(), nil)
	if result.GetMethod() != "system.quit" {
		t.Fatalf("method = %q, want system.quit", result.GetMethod())
	}
}

// --- key coverage ---

func TestKeyCoverage(t *testing.T) {
	newSplit := func() *model {
		m := seededModel(t)
		prefix(m, "v")
		prefix(m, "c")
		prefix(m, "v")
		return m
	}

	t.Run("v and minus set the axis", func(t *testing.T) {
		m := seededModel(t)
		prefix(m, "v")
		if m.currentTab().Axis != "row" || len(m.currentTab().Panes) != 2 {
			t.Fatalf("v = %q %d", m.currentTab().Axis, len(m.currentTab().Panes))
		}
		m2 := seededModel(t)
		prefix(m2, "-")
		if m2.currentTab().Axis != "col" || len(m2.currentTab().Panes) != 2 {
			t.Fatalf("- = %q %d", m2.currentTab().Axis, len(m2.currentTab().Panes))
		}
	})
	t.Run("h j k l move focus", func(t *testing.T) {
		m := newSplit()
		prefix(m, "h")
		if m.currentTab().Focus != "p1" {
			t.Fatalf("h = %s", m.currentTab().Focus)
		}
		prefix(m, "l")
		if m.currentTab().Focus != "p2" {
			t.Fatalf("l = %s", m.currentTab().Focus)
		}
	})
	t.Run("shift letters swap", func(t *testing.T) {
		m := newSplit()
		before := m.currentTab().Panes[0].ID
		prefix(m, "H")
		if m.currentTab().Panes[0].ID == before {
			t.Fatal("H did not swap panes")
		}
		prefix(m, "L")
		if m.currentTab().Panes[0].ID != before {
			t.Fatal("L did not swap back")
		}
	})
	t.Run("tab cycles panes", func(t *testing.T) {
		m := newSplit()
		focus := m.currentTab().Focus
		prefix(m, "tab")
		if m.currentTab().Focus == focus {
			t.Fatal("tab did not cycle focus")
		}
		prefix(m, "shift-tab")
		if m.currentTab().Focus != focus {
			t.Fatal("shift-tab did not cycle back")
		}
	})
	t.Run("x closes pane and X closes tab", func(t *testing.T) {
		m := newSplit()
		panes := len(m.currentTab().Panes)
		prefix(m, "x")
		if len(m.currentTab().Panes) != panes-1 {
			t.Fatalf("x = %d panes", len(m.currentTab().Panes))
		}
		tabs := len(m.currentWS().Tabs)
		prefix(m, "X")
		if len(m.currentWS().Tabs) != tabs-1 {
			t.Fatalf("X = %d tabs", len(m.currentWS().Tabs))
		}
	})
	t.Run("z zooms", func(t *testing.T) {
		m := newSplit()
		prefix(m, "z")
		if !m.zoom {
			t.Fatal("z did not zoom")
		}
	})
	t.Run("r enters resize", func(t *testing.T) {
		m := seededModel(t)
		prefix(m, "r")
		if m.mode != modeResize {
			t.Fatalf("r = %q", m.mode)
		}
	})
	t.Run("c and N create", func(t *testing.T) {
		m := seededModel(t)
		prefix(m, "c")
		prefix(m, "N")
		if len(m.currentWS().Tabs) != 1 || len(m.workspaces) != 2 {
			t.Fatalf("c/N = %d tabs %d workspaces", len(m.currentWS().Tabs), len(m.workspaces))
		}
	})
	t.Run("p and n switch tabs", func(t *testing.T) {
		m := seededModel(t)
		prefix(m, "c")
		prefix(m, "p")
		if m.currentWS().ActiveTab != "t1" {
			t.Fatalf("p = %s", m.currentWS().ActiveTab)
		}
		prefix(m, "n")
		if m.currentWS().ActiveTab != "t2" {
			t.Fatalf("n = %s", m.currentWS().ActiveTab)
		}
	})
	t.Run("digits switch tabs", func(t *testing.T) {
		m := seededModel(t)
		prefix(m, "c")
		prefix(m, "c")
		prefix(m, "1")
		if m.currentWS().ActiveTab != "t1" {
			t.Fatalf("1 = %s", m.currentWS().ActiveTab)
		}
		prefix(m, "3")
		if m.currentWS().ActiveTab != "t3" {
			t.Fatalf("3 = %s", m.currentWS().ActiveTab)
		}
		prefix(m, "9") // out of range: ignored
		if m.currentWS().ActiveTab != "t3" {
			t.Fatalf("9 changed tab to %s", m.currentWS().ActiveTab)
		}
	})
	t.Run("D closes a workspace", func(t *testing.T) {
		m := seededModel(t)
		prefix(m, "N")
		prefix(m, "D")
		if len(m.workspaces) != 1 {
			t.Fatalf("D = %d workspaces", len(m.workspaces))
		}
	})
	t.Run("W T P open rename modals", func(t *testing.T) {
		for _, tc := range []struct {
			key  string
			kind renameKind
		}{{"W", renameWorkspace}, {"T", renameTab}, {"P", renamePane}} {
			m := seededModel(t)
			prefix(m, tc.key)
			if m.rename == nil || m.rename.kind != tc.kind {
				t.Fatalf("%s rename = %+v", tc.key, m.rename)
			}
			key(m, "esc")
			if m.rename != nil {
				t.Fatalf("%s rename did not close on esc", tc.key)
			}
		}
	})
	t.Run("renaming via the TextInput", func(t *testing.T) {
		m := seededModel(t)
		prefix(m, "P")
		for _, r := range "webx" {
			key(m, string(r))
		}
		key(m, "backspace")
		key(m, "enter")
		if got := m.currentTab().Panes[0].Name; got != "web" {
			t.Fatalf("pane name = %q, want web", got)
		}
		if m.rename != nil {
			t.Fatal("rename modal stayed open after enter")
		}
	})
	t.Run("w and question and bracket", func(t *testing.T) {
		m := seededModel(t)
		prefix(m, "w")
		if m.mode != modeNavigate {
			t.Fatalf("w = %q", m.mode)
		}
		key(m, "esc")
		prefix(m, "?")
		if !m.help {
			t.Fatal("? did not open help")
		}
		key(m, "esc")
		alpha := terminalSource("terminal:local:alpha", "alpha", "local")
		bindSource(t, m, alpha)
		prefix(m, "[")
		if m.mode != modeScroll {
			t.Fatalf("[ = %q", m.mode)
		}
		key(m, "q")
	})
	t.Run("b toggles the sidebar", func(t *testing.T) {
		m := seededModel(t)
		prefix(m, "b")
		if !m.sidebarCollapsed {
			t.Fatal("b did not collapse")
		}
	})
	t.Run("q quits through the host", func(t *testing.T) {
		host := newFakeHost(t)
		m := newModel(host.client, &app.Memo{})
		m.Update(helloMsg("view-1"))
		key(m, "ctrl-b")
		if m.mode != modePrefix {
			t.Fatalf("mode = %q, want prefix", m.mode)
		}
		cmd := key(m, "q")
		if cmd == nil {
			t.Fatal("prefix q emitted nothing")
		}
		if m.mode != modeTerminal {
			t.Fatalf("mode after q = %q", m.mode)
		}
		// The quit command is chained with SetKeys, so assert the wire method
		// by running the model's own quit command.
		result, _ := runEmit(t, host, m.quitCmd(), nil)
		if result.GetMethod() != "system.quit" {
			t.Fatalf("method = %q", result.GetMethod())
		}
	})
	t.Run("direct chords", func(t *testing.T) {
		m := newSplit()
		focus := m.currentTab().Focus
		key(m, "ctrl-alt-h")
		if m.currentTab().Focus == focus {
			t.Fatal("ctrl-alt-h did not move focus")
		}
		tabs := len(m.currentWS().Tabs)
		key(m, "ctrl-alt-c")
		if len(m.currentWS().Tabs) != tabs+1 {
			t.Fatal("ctrl-alt-c did not create a tab")
		}
		panes := len(m.currentTab().Panes)
		key(m, "ctrl-alt-d")
		if len(m.currentTab().Panes) != panes+1 {
			t.Fatal("ctrl-alt-d did not split")
		}
		key(m, "ctrl-alt-z")
		if !m.zoom {
			t.Fatal("ctrl-alt-z did not zoom")
		}
	})
}

func TestKeyClaimMatrix(t *testing.T) {
	m := seededModel(t)
	if keys := m.keys(); !keys.All || len(keys.Claim) != 0 {
		t.Fatalf("empty pane claim = %+v, want All", keys)
	}

	alpha := terminalSource("terminal:local:alpha", "alpha", "local")
	bindSource(t, m, alpha)
	keys := m.keys()
	if keys.All {
		t.Fatal("live terminal must not claim All")
	}
	if !reflect.DeepEqual(keys.Claim, directChords) {
		t.Fatalf("claim = %v, want %v", keys.Claim, directChords)
	}

	// Program modes always claim everything.
	for _, enter := range []func(){
		func() { key(m, "ctrl-b") },
		func() { key(m, "ctrl-b") },
		func() { prefix(m, "w") },
		func() { prefix(m, "r") },
		func() { prefix(m, "[") },
		func() { prefix(m, "?") },
		func() { prefix(m, "P") },
	} {
		enter()
		if keys := m.keys(); !keys.All {
			t.Fatalf("program mode %q claim = %+v, want All", m.mode, keys)
		}
		key(m, "esc")
	}
	// menu
	m.Update(mouse("right", "press", agentRowPrefix+"w1:t1:p1", 5, 3))
	if keys := m.keys(); !keys.All {
		t.Fatalf("menu claim = %+v, want All", keys)
	}
	key(m, "esc")

	// Exited/offline/gone panes are not PTY targets.
	beta := terminalSource("terminal:local:beta", "beta", "local")
	beta.Exited, beta.ExitCode = true, 1
	m.currentTab().Panes[0].Source = beta.GetId()
	m.sources[beta.GetId()] = beta
	if keys := m.keys(); !keys.All {
		t.Fatalf("exited pane claim = %+v, want All", keys)
	}
}

// --- mouse ---

func TestMouseClicksAndContextMenu(t *testing.T) {
	m := seededModel(t)
	prefix(m, "N") // w2 active

	// Sidebar workspace row activates.
	m.Update(mouse("left", "press", spaceRowPrefix+"w1", 2, 5))
	if m.activeWorkspace != "w1" {
		t.Fatalf("ws click active = %s, want w1", m.activeWorkspace)
	}

	// Tab click switches.
	prefix(m, "c") // w1:t2 active
	prefix(m, "N") // w2
	prefix(m, "N") // w3 active
	prefix(m, "c") // w3:t2 active
	m.Update(mouse("left", "press", tabPrefix+"w3:t1", 30, 0))
	if m.currentWS().ActiveTab != "t1" {
		t.Fatalf("tab click = %s", m.currentWS().ActiveTab)
	}

	// Pane row click focuses the pane.
	prefix(m, "v")
	m.Update(mouse("left", "press", agentRowPrefix+"w3:t1:p1", 3, 20))
	if m.currentPaneRef() != "w3:t1:p1" {
		t.Fatalf("pane row click = %s", m.currentPaneRef())
	}

	// Clicking the pane box focuses it too.
	m.Update(mouse("left", "press", paneBoxPrefix+"w3:t1:p2", 60, 12))
	if m.currentPaneRef() != "w3:t1:p2" {
		t.Fatalf("pane box click = %s", m.currentPaneRef())
	}

	// Right-click opens the context menu and close-pane closes the target.
	m.Update(mouse("right", "press", agentRowPrefix+"w3:t1:p1", 3, 20))
	if !m.menu.Visible || m.menuKind != menuPane {
		t.Fatalf("menu = %v kind %q", m.menu.Visible, m.menuKind)
	}
	m.Update(mouse("left", "press", menuPrefix+actClosePane, 30, 10))
	if m.menu.Visible {
		t.Fatal("menu did not close after the item was chosen")
	}
	if len(m.currentTab().Panes) != 1 {
		t.Fatalf("close-pane left %d panes", len(m.currentTab().Panes))
	}

	// Right-clicking a tab offers rename/close tab.
	m.Update(mouse("right", "press", tabPrefix+"w3:t1", 30, 0))
	if !m.menu.Visible || m.menuKind != menuTab {
		t.Fatalf("tab menu = %v/%q", m.menu.Visible, m.menuKind)
	}
	m.Update(mouse("left", "press", menuPrefix+actRename, 30, 4))
	if m.rename == nil || m.rename.kind != renameTab {
		t.Fatalf("menu rename = %+v", m.rename)
	}
	key(m, "esc")

	// Right-click a workspace row.
	m.Update(mouse("right", "press", spaceRowPrefix+"w1", 3, 3))
	if !m.menu.Visible || m.menuKind != menuWorkspace {
		t.Fatalf("ws menu = %v/%q", m.menu.Visible, m.menuKind)
	}
	key(m, "esc")

	// A right-click on the + tab does not open a menu.
	m.Update(mouse("right", "press", newTabID, 40, 0))
	if m.menu.Visible {
		t.Fatal("right-click on + opened a menu")
	}
}

func TestDividerDragAdjustsWeights(t *testing.T) {
	m := seededModel(t)
	prefix(m, "v")
	node := dividerPrefix + "w1:t1:0"
	m.Update(mouse("left", "press", node, 90, 10))
	if m.drag == nil {
		t.Fatal("press on the divider did not start a drag")
	}
	m.Update(mouse("none", "drag", node, 80, 10))
	tab := m.currentTab()
	if sameInts(tab.Weights, []int{50, 50}) {
		t.Fatalf("drag did not change weights: %v", tab.Weights)
	}
	if sumInts(tab.Weights) != 100 || tab.Weights[0] >= tab.Weights[1] {
		t.Fatalf("drag left weights = %v, want the left pane shrunk", tab.Weights)
	}
	if !m.drag.moved {
		t.Fatal("drag.moved is false")
	}
	m.Update(mouse("none", "release", node, 80, 10))
	if m.drag != nil {
		t.Fatal("release did not end the drag")
	}
	after := append([]int(nil), tab.Weights...)

	// A huge drag clamps without breaking the sum.
	m.Update(mouse("left", "press", node, 10, 10))
	m.Update(mouse("none", "drag", node, 200, 10))
	if tab.Weights[0] < minPaneWeight || tab.Weights[1] > maxPaneWeight || sumInts(tab.Weights) != 100 {
		t.Fatalf("clamped weights = %v", tab.Weights)
	}
	m.Update(mouse("none", "release", node, 200, 10))
	if sameInts(tab.Weights, after) {
		t.Fatal("second drag had no effect")
	}
}

func TestWheelScrollsSidebarLists(t *testing.T) {
	m := seededModel(t)
	m.Update(app.ResizeMsg{Cols: 100, Rows: 12}) // shrink so the panels scroll
	for i := 0; i < 6; i++ {
		prefix(m, "N")
	}
	if len(m.workspaces) != 7 {
		t.Fatalf("workspaces = %d", len(m.workspaces))
	}
	m.Update(app.WheelMsg{Wheel: &pb.WheelEvent{Delta: 1, X: 3, Y: 5, Node: "herdr.sidebar.spaces"}})
	if m.spacesOffset != 1 {
		t.Fatalf("spacesOffset = %d, want 1", m.spacesOffset)
	}
	m.Update(app.WheelMsg{Wheel: &pb.WheelEvent{Delta: 100, X: 3, Y: 5, Node: "herdr.sidebar.spaces"}})
	if m.spacesOffset != len(m.workspaces)-(m.sidebarSpacesHeight()-1) {
		t.Fatalf("spacesOffset clamped = %d", m.spacesOffset)
	}
	// Wheel on the Agents panel scrolls that one: every workspace
	// contributes at least one pane, plus the two splits on the active tab.
	prefix(m, "v")
	prefix(m, "v")
	if m.agentListLen() <= m.sidebarAgentsHeight()-1 {
		t.Fatalf("agents list = %d rows, not scrollable at height %d", m.agentListLen(), m.sidebarAgentsHeight())
	}
	m.Update(app.WheelMsg{Wheel: &pb.WheelEvent{Delta: 1, X: 3, Y: 1, Node: "herdr.sidebar.agents"}})
	if m.agentsOffset != 1 {
		t.Fatalf("agentsOffset = %d, want 1", m.agentsOffset)
	}
	// A wheel outside the sidebar is ignored.
	m.Update(app.WheelMsg{Wheel: &pb.WheelEvent{Delta: 1, X: 70, Y: 5, Node: ""}})
	if m.spacesOffset != len(m.workspaces)-(m.sidebarSpacesHeight()-1) {
		t.Fatalf("outside wheel changed spacesOffset = %d", m.spacesOffset)
	}
	// Collapsed sidebar ignores wheel events entirely.
	prefix(m, "b")
	spaceBefore, agentBefore := m.spacesOffset, m.agentsOffset
	m.Update(app.WheelMsg{Wheel: &pb.WheelEvent{Delta: 1, X: 3, Y: 5, Node: "herdr.sidebar.spaces"}})
	if m.spacesOffset != spaceBefore || m.agentsOffset != agentBefore {
		t.Fatalf("collapsed wheel changed offsets %d/%d", m.spacesOffset, m.agentsOffset)
	}
}

// --- view ---

func TestViewStatesAndModeBars(t *testing.T) {
	m := seededModel(t)
	text := allText(m.View())
	for _, want := range []string{"empty pane", "enter creates a terminal", "1:main", "Agents", "Spaces"} {
		if !strings.Contains(text, want) {
			t.Fatalf("empty view is missing %q:\n%s", want, text)
		}
	}
	if findBox(m.View(), "herdr.sidebar.spaces") == nil || findBox(m.View(), "herdr.sidebar.agents") == nil {
		t.Fatal("sidebar panels are missing")
	}

	// A live pane renders a terminal box and no border (single pane).
	alpha := terminalSource("terminal:local:alpha", "alpha", "local")
	bindSource(t, m, alpha)
	root := m.View()
	term := findBox(root, termPrefix+"w1:t1:p1")
	if term == nil || term.GetContent().GetSelf() != alpha.GetId() {
		t.Fatalf("terminal box = %+v", term)
	}
	if !term.GetFocused() {
		t.Fatal("live terminal pane must be focused in terminal mode")
	}
	if props := term.GetContent().GetProps(); props["chrome.inset"] != "0" {
		t.Fatalf("single pane props = %v, want borderless chrome.inset=0", props)
	}

	// Split: program-drawn titles and a divider box.
	prefix(m, "v")
	root = m.View()
	if findBox(root, dividerPrefix+"w1:t1:0") == nil {
		t.Fatal("split divider box is missing")
	}
	if !strings.Contains(allText(root), "pane 1 · alpha") {
		t.Fatalf("split title is missing:\n%s", allText(root))
	}

	// Offline and gone states.
	sick := terminalSource("terminal:local:sick", "sick", "local")
	sick.Health = "down"
	_, _, p2 := m.focusedPane()
	p2.Source = sick.GetId()
	m.sources = map[string]*pb.Source{sick.GetId(): sick}
	if !strings.Contains(allText(m.View()), "endpoint offline") {
		t.Fatalf("offline pane is not rendered:\n%s", allText(m.View()))
	}
	delete(m.sources, sick.GetId())
	if !strings.Contains(allText(m.View()), "source gone") {
		t.Fatalf("gone pane is not rendered:\n%s", allText(m.View()))
	}

	// Mode bars replace the tab row.
	key(m, "ctrl-b")
	if text := allText(m.View()); !strings.Contains(text, "PREFIX") || !strings.Contains(text, "q detach") {
		t.Fatalf("prefix mode bar:\n%s", text)
	}
	if findBox(m.View(), tabbarID) != nil {
		t.Fatal("tab bar still rendered in prefix mode")
	}
	key(m, "esc")
	prefix(m, "r")
	if !strings.Contains(allText(m.View()), "RESIZE") {
		t.Fatalf("resize mode bar:\n%s", allText(m.View()))
	}
	key(m, "esc")
	prefix(m, "w")
	if !strings.Contains(allText(m.View()), "NAVIGATE") {
		t.Fatalf("navigate mode bar:\n%s", allText(m.View()))
	}
	key(m, "esc")

	// Help overlay.
	prefix(m, "?")
	if findBox(m.View(), helpID) == nil {
		t.Fatal("help overlay is missing")
	}
	key(m, "esc")

	// Zoom shows only the focused pane (p2, currently gone) and the marker.
	prefix(m, "z")
	root = m.View()
	if !strings.Contains(allText(root), "ZOOM") {
		t.Fatal("zoom marker is missing")
	}
	if findBox(root, paneBoxPrefix+"w1:t1:p2") == nil || findBox(root, termPrefix+"w1:t1:p1") != nil {
		t.Fatal("zoom did not restrict the pane area to the focused pane")
	}
}

func TestProgramModeClearsFocused(t *testing.T) {
	m := seededModel(t)
	alpha := terminalSource("terminal:local:alpha", "alpha", "local")
	bindSource(t, m, alpha)
	if term := findBox(m.View(), termPrefix+"w1:t1:p1"); term == nil || !term.GetFocused() {
		t.Fatal("terminal is not focused in terminal mode")
	}
	key(m, "ctrl-b")
	if term := findBox(m.View(), termPrefix+"w1:t1:p1"); term == nil || term.GetFocused() {
		t.Fatal("prefix mode did not clear focused (PROTOCOL §6.5)")
	}
	key(m, "esc")
	if term := findBox(m.View(), termPrefix+"w1:t1:p1"); term == nil || !term.GetFocused() {
		t.Fatal("terminal focus was not restored")
	}
}

// --- dirty / batching ---

// TestDirtySurvivesANoopMessageInTheSameBatch pins the batch-safe dirty flag:
// a state change followed by a no-op result in one batch must still commit.
func TestDirtySurvivesANoopMessageInTheSameBatch(t *testing.T) {
	m := seededModel(t)
	m.View() // clear the seed's dirty flag
	if m.Dirty() {
		t.Fatal("View did not clear dirty")
	}
	feed(m, terminalSource("terminal:local:alpha", "alpha", "local"))
	if !m.Dirty() {
		t.Fatal("a sources snapshot must mark the model dirty")
	}
	m.Update(storageMsg{op: "set", key: layoutKey, ok: true}) // no-op result
	if !m.Dirty() {
		t.Fatal("a no-op result in the same batch suppressed the commit")
	}
	m.View()
	if m.Dirty() {
		t.Fatal("View did not clear dirty")
	}
}

// --- memo ---

func TestSidebarMemoInvalidation(t *testing.T) {
	m := seededModel(t)
	alpha := terminalSource("terminal:local:alpha", "alpha", "local")
	bindSource(t, m, alpha)
	first := m.sidebarBox()

	// An ignored key changes nothing.
	key(m, "z")
	if second := m.sidebarBox(); second != first {
		t.Fatal("an ignored key rebuilt the sidebar")
	}
	// A toast changes the view but not the sidebar.
	m.Update(app.NoticeMsg{Level: "info", Message: "still here"})
	if third := m.sidebarBox(); third != first {
		t.Fatal("a toast rebuilt the sidebar")
	}
	// A status change invalidates it.
	sick := terminalSource("terminal:local:alpha", "alpha", "local")
	sick.Health = "degraded"
	feed(m, sick)
	if changed := m.sidebarBox(); changed == first {
		t.Fatal("a state change did not invalidate the sidebar")
	}
}

// --- chrome / sidebar (SPEC §3) ---

// TestFirstRunMissingLayoutIsSilent pins SPEC §7: a storage Get that answers
// "not found" (either as the rejected RESPONSE text or as an ApiError
// NOT_FOUND envelope) means "no saved layout yet" — the default seed stays and
// nothing is toasted.
func TestFirstRunMissingLayoutIsSilent(t *testing.T) {
	host := newFakeHost(t)
	m := newModel(host.client, &app.Memo{})
	m.Update(helloMsg("view-1"))

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init did not read the layout")
	}
	// The host rejects the access.call itself with the storage error text.
	_, msg := runEmitExpect(t, host, cmd, nil, false, "storage entry was not found")
	stored, ok := msg.(storageMsg)
	if !ok || !stored.empty || stored.err != "" {
		t.Fatalf("not-found get = %#v, want an empty success", msg)
	}
	cmd = m.Update(msg)
	if cmd == nil {
		t.Fatal("a missing layout did not fall back to the legacy pinned read")
	}
	_, msg = runEmitExpect(t, host, cmd, nil, false, "storage entry was not found")
	m.Update(msg)

	if m.toast != "" {
		t.Fatalf("first run toasted %q", m.toast)
	}
	if findBox(m.View(), toastID) != nil {
		t.Fatal("first run rendered a toast overlay")
	}
	if !m.layoutReady {
		t.Fatal("a missing layout must still open the save gate")
	}
	if m.currentPaneRef() != "w1:t1:p1" {
		t.Fatalf("seed layout = %q, want w1:t1:p1", m.currentPaneRef())
	}
	if m.pendingPin != "" {
		t.Fatalf("pendingPin = %q, want empty", m.pendingPin)
	}

	// The ApiError NOT_FOUND envelope means the same thing.
	payload, err := gproto.Marshal(&apipb.ResultEnvelope{
		Result: &apipb.ResultEnvelope_Error{Error: &apipb.ApiError{
			Code:    apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND,
			Message: "storage entry was not found",
		}},
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	envelope := decodeStorageGet(payload)
	decoded, ok := envelope.(storageMsg)
	if !ok || !decoded.empty || decoded.err != "" {
		t.Fatalf("NOT_FOUND envelope = %#v, want an empty success", envelope)
	}
}

// TestViewHasNoPersistentStatusOrFooter pins the row budget: the tree is the
// tab (or mode) bar plus the body only — no status row, no footer row and no
// reserved toast row.
func TestViewHasNoPersistentStatusOrFooter(t *testing.T) {
	m := seededModel(t)
	root := m.View()
	if findBox(root, "herdr.status") != nil {
		t.Fatal("a permanent status row is still in the tree")
	}
	if findBox(root, toastID) != nil {
		t.Fatal("an empty toast still reserved a row")
	}
	if got := m.bodyHeight(); got != m.rows-1 {
		t.Fatalf("bodyHeight = %d, want rows-1 = %d", got, m.rows-1)
	}
	if len(root.GetChildren()) != 2 {
		t.Fatalf("root children = %d, want tab bar + body", len(root.GetChildren()))
	}
	top, body := root.GetChildren()[0], root.GetChildren()[1]
	if top.GetSize().GetHeight() != 1 {
		t.Fatalf("top bar height = %d, want 1", top.GetSize().GetHeight())
	}
	if body.GetSize().GetHeight() != int32(m.rows-1) {
		t.Fatalf("body height = %d, want %d", body.GetSize().GetHeight(), m.rows-1)
	}
	if text := allText(root); strings.Contains(text, fmt.Sprintf("%dx%d", m.cols, m.rows)) {
		t.Fatalf("view still renders the removed size text:\n%s", text)
	}
}

// TestTabBarConnectionIndicator pins the compact right-side indicator: the
// endpoint of the focused pane, or ● connected when no bound source exists.
func TestTabBarConnectionIndicator(t *testing.T) {
	m := seededModel(t)
	// The indicator is split into styled runs; join the runs without newlines.
	tabText := func() string {
		return strings.ReplaceAll(allText(findBox(m.View(), tabbarID)), "\n", "")
	}
	if text := tabText(); !strings.Contains(text, "● connected") {
		t.Fatalf("tab bar is missing the connection indicator:\n%s", text)
	}
	alpha := terminalSource("terminal:prod:web", "web", "prod")
	bindSource(t, m, alpha)
	if text := tabText(); !strings.Contains(text, "● prod") {
		t.Fatalf("tab bar does not show the focused endpoint:\n%s", text)
	}
	// The indicator is reserved: with many tabs and a narrow viewport the
	// trailing tabs drop first and the connection state stays visible.
	for i := 0; i < 20; i++ {
		prefix(m, "c")
	}
	m.Update(app.ResizeMsg{Cols: 40, Rows: 30})
	if text := tabText(); !strings.Contains(text, "● connected") {
		t.Fatalf("the connection indicator was clipped by the tab strip:\n%s", text)
	}
}

// TestToastIsTransientBottomRightOverlay pins the transient toast: no box
// while empty, a bottom-right Pos overlay while set, and clipping to the
// viewport for long host notices.
func TestToastIsTransientBottomRightOverlay(t *testing.T) {
	m := seededModel(t)
	if findBox(m.View(), toastID) != nil {
		t.Fatal("an empty toast still rendered a box")
	}

	m.setToast("layout restore failed: storage entry was not found", true)
	root := m.View()
	box := findBox(root, toastID)
	if box == nil {
		t.Fatal("toast box is missing")
	}
	pos := box.GetPos()
	if pos == nil {
		t.Fatal("toast is not an overlay")
	}
	width := int(box.GetSize().GetWidth())
	if width != sdk.DisplayWidth(m.toast)+2 {
		t.Fatalf("toast width = %d, want text width + 2 padding", width)
	}
	if pos.GetX() != int32(m.cols-width) || pos.GetY() != int32(m.rows-1) {
		t.Fatalf("toast pos = (%d,%d), want bottom-right (%d,%d)", pos.GetX(), pos.GetY(), m.cols-width, m.rows-1)
	}
	if !strings.Contains(box.GetContent().GetText(), "layout restore failed") {
		t.Fatalf("toast text = %q", box.GetContent().GetText())
	}

	// A long notice clips to the viewport width.
	m.setToast(strings.Repeat("x", m.cols*2), true)
	box = findBox(m.View(), toastID)
	if box == nil {
		t.Fatal("long toast box is missing")
	}
	if got := box.GetSize().GetWidth(); got != int32(m.cols) {
		t.Fatalf("long toast width = %d, want %d", got, m.cols)
	}
	if box.GetPos().GetX() != 0 {
		t.Fatalf("long toast x = %d, want 0", box.GetPos().GetX())
	}
	if got := sdk.DisplayWidth(box.GetContent().GetText()); got != m.cols {
		t.Fatalf("long toast text width = %d, want %d", got, m.cols)
	}

	// A toast composes with an overlay without losing either.
	prefix(m, "?")
	root = m.View()
	if findBox(root, toastID) == nil || findBox(root, helpID) == nil {
		t.Fatal("toast and overlay did not compose")
	}
}

// TestSidebarAgentsGroupAndSpacesPanel pins the upstream sidebar shape: the
// panes of every workspace grouped by machine, two lines per pane, plus the
// Spaces panel.
func TestSidebarAgentsGroupAndSpacesPanel(t *testing.T) {
	m := seededModel(t)
	prefix(m, "v")
	prefix(m, "v")
	tab := m.currentTab()
	web := terminalSource("terminal:prod:web", "web", "prod")
	db := terminalSource("terminal:prod:db", "db", "prod")
	api := terminalSource("terminal:staging:api", "api", "staging")
	tab.Panes[0].Source = web.GetId()
	tab.Panes[1].Source = db.GetId()
	tab.Panes[2].Source = api.GetId()
	feed(m, web, db, api)

	root := m.View()
	agents := findBox(root, "herdr.sidebar.agents")
	if agents == nil {
		t.Fatal("Agents panel is missing")
	}
	if findBox(root, "herdr.sidebar.spaces") == nil {
		t.Fatal("Spaces panel is missing")
	}
	text := allText(agents)
	// Two machine headers, the two-line rows and the focused-pane marker.
	for _, want := range []string{"prod", "staging", "main · main", "web", "db", "api", "▸ ● main · main"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Agents panel is missing %q:\n%s", want, text)
		}
	}
	for _, ref := range []string{"w1:t1:p1", "w1:t1:p2", "w1:t1:p3"} {
		if findBox(root, agentRowPrefix+ref) == nil {
			t.Fatalf("agent row %s is missing:\n%s", ref, text)
		}
	}
	row := findBox(root, agentRowPrefix+"w1:t1:p3")
	if row == nil || row.GetStyle() != widgets.StyleSelection {
		t.Fatalf("active workspace agent row style = %q, want selection", row.GetStyle())
	}
	if findBox(root, spaceRowPrefix+"w1") == nil {
		t.Fatal("Spaces row is missing")
	}
	if !strings.Contains(allText(findBox(root, "herdr.sidebar.spaces")), "main") {
		t.Fatalf("Spaces panel is missing the workspace name:\n%s", allText(root))
	}
}

// TestSidebarAgentsSingleMachineOmitsHeader pins the grouping rule: with only
// one machine no machine header row is rendered.
func TestSidebarAgentsSingleMachineOmitsHeader(t *testing.T) {
	m := seededModel(t)
	prefix(m, "v")
	web := terminalSource("terminal:prod:web", "web", "prod")
	db := terminalSource("terminal:prod:db", "db", "prod")
	m.currentTab().Panes[0].Source = web.GetId()
	m.currentTab().Panes[1].Source = db.GetId()
	feed(m, web, db)

	agents := findBox(m.View(), "herdr.sidebar.agents")
	if agents == nil {
		t.Fatal("Agents panel is missing")
	}
	if strings.Contains(allText(agents), "prod") {
		t.Fatalf("a single machine must not render a header:\n%s", allText(agents))
	}
	// Panel header + two two-line row blocks.
	if got := len(agents.GetChildren()); got != 3 {
		t.Fatalf("Agents children = %d, want 3 (header + 2 two-line rows)", got)
	}
	for _, child := range agents.GetChildren()[1:] {
		if child.GetSize().GetHeight() != 2 || len(child.GetChildren()) != 2 {
			t.Fatalf("agent row block = %+v, want two lines", child)
		}
	}
}

// TestSidebarAgentsUnboundGroup pins the (unbound) bucket for empty and gone
// panes.
func TestSidebarAgentsUnboundGroup(t *testing.T) {
	m := seededModel(t)
	prefix(m, "v")
	web := terminalSource("terminal:prod:web", "web", "prod")
	m.currentTab().Panes[0].Source = web.GetId()
	feed(m, web) // p2 stays unbound

	text := allText(findBox(m.View(), "herdr.sidebar.agents"))
	for _, want := range []string{"prod", unboundMachine} {
		if !strings.Contains(text, want) {
			t.Fatalf("Agents panel is missing %q:\n%s", want, text)
		}
	}
	// A gone source falls back into the same bucket.
	m.currentTab().Panes[1].Source = "terminal:prod:dead"
	feed(m, web)
	text = allText(findBox(m.View(), "herdr.sidebar.agents"))
	if strings.Count(text, unboundMachine) != 1 {
		t.Fatalf("a gone pane did not join the (unbound) group:\n%s", text)
	}
}

// --- small helpers used by the tests above ---

func sumInts(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func (m *model) currentPaneSource() string {
	_, _, p := m.focusedPane()
	if p == nil {
		return ""
	}
	return p.Source
}

// TestHelpModalWrapsWithoutDroppingText pins the help overlay: every help row
// is wrapped to the modal inner width (no mid-word truncation), and the modal
// grows with the wrapped line count instead of clipping them.
func TestHelpModalWrapsWithoutDroppingText(t *testing.T) {
	for _, cols := range []int{40, 64, 100, 221} {
		m := newModel(nil, &app.Memo{})
		m.cols, m.rows = cols, 40
		m.help = true
		inner := m.helpModalInnerWidth()
		lines := m.helpWrappedLines(inner)
		if len(lines) == 0 {
			t.Fatalf("cols=%d: no help lines", cols)
		}
		for i, line := range lines {
			if got := sdk.DisplayWidth(line); got > inner {
				t.Fatalf("cols=%d: line %d width %d > inner %d (%q)", cols, i, got, inner, line)
			}
		}
		// No source row may lose text when the content fits the viewport; when
		// it does not, the modal must say so instead of clipping silently.
		if len(lines)+2 <= m.rows {
			joined := strings.Join(lines, "")
			compact := func(s string) string {
				return strings.Join(strings.Fields(s), "")
			}
			for _, row := range helpRows {
				if !strings.Contains(compact(joined), compact(row)) {
					t.Fatalf("cols=%d: help row lost text: %q", cols, row)
				}
			}
		}
	}
}

// TestHelpModalFitsViewport checks the modal height covers the wrapped rows
// whenever the viewport is tall enough.
func TestHelpModalFitsViewport(t *testing.T) {
	m := newModel(nil, &app.Memo{})
	m.cols, m.rows = 80, 40
	m.help = true
	lines := m.helpWrappedLines(m.helpModalInnerWidth())
	box := m.helpModal().Build()
	if box == nil {
		t.Fatal("no help modal")
	}
	if got := countTextBoxes(box); got < len(lines) {
		t.Fatalf("help text rows = %d, want >= %d wrapped lines", got, len(lines))
	}
}

// countTextBoxes counts text-bearing boxes in a built view tree.
func countTextBoxes(box *pb.Box) int {
	if box == nil {
		return 0
	}
	n := 0
	if box.GetContent().GetText() != "" || len(box.GetContent().GetLines()) > 0 {
		n++
	}
	for _, child := range box.GetChildren() {
		n += countTextBoxes(child)
	}
	return n
}

// TestHelpModalMarksOverflowInShortViewports pins the no-silent-clip rule: a
// viewport too short for the whole list gets a trailing marker instead of a
// cut-off row.
func TestHelpModalMarksOverflowInShortViewports(t *testing.T) {
	m := newModel(nil, &app.Memo{})
	m.cols, m.rows = 60, 12
	m.help = true
	box := m.helpModal().Build()
	if box == nil {
		t.Fatal("no help modal")
	}
	if !strings.Contains(allText(box), "resize taller") {
		t.Fatalf("short viewport help did not mark the overflow:\n%s", allText(box))
	}
}

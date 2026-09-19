package main

import (
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/config"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func testHello(m *model) {
	m.hello(&pb.Hello{Schema: 1, ViewId: "view:local:1", Epoch: 1, Cols: 100, Rows: 30})
}

func terminalSource(id, title string, exited bool) *pb.Source {
	return &pb.Source{
		Id:         id,
		Kind:       "terminal",
		Title:      title,
		Endpoint:   "local",
		TerminalId: strings.TrimPrefix(id, "terminal:local:"),
		Attached:   !exited,
		Exited:     exited,
		Health:     "ok",
	}
}

func keyPress(key string) keyEvent { return keyEvent{ID: "ev:" + key, Key: key} }

func mustBind(t *testing.T, m *model, sourceID string) {
	t.Helper()
	reqs := m.sourcesEvent([]*pb.Source{terminalSource(sourceID, "t", false)})
	if len(reqs) != 1 || reqs[0].Method != "terminal.attach" {
		t.Fatalf("auto-bind requests = %+v, want one attach", reqs)
	}
	if m.focusSlot().sourceID != sourceID {
		t.Fatalf("focus slot source = %q, want %q", m.focusSlot().sourceID, sourceID)
	}
}

func TestColdStartPickerWhenEmpty(t *testing.T) {
	m := newModel()
	testHello(m)
	reqs := m.sourcesEvent(nil)
	if len(reqs) != 0 {
		t.Fatalf("empty sources requests = %+v, want none", reqs)
	}
	if m.mode != modePicker {
		t.Fatalf("mode = %v, want PICKER", m.mode)
	}
	if len(m.tabs) != 1 || len(m.tabs[0].slots) != 1 {
		t.Fatalf("default layout = %d tabs / %d slots", len(m.tabs), len(m.tabs[0].slots))
	}
}

func TestSourcesAutoBind(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	if m.mode != modeNormal {
		t.Fatalf("mode after auto-bind = %v, want NORMAL", m.mode)
	}
	req := m.attachRequest("terminal:local:main")
	if req.Params.GetEndpoint() != "local" || req.Params.GetId() != "main" || !req.Params.GetFit() {
		t.Fatalf("attach params = %+v", req.Params)
	}
}

func TestModeTransitions(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")

	if reqs := m.key(keyPress("ctrl-p")); len(reqs) != 0 {
		t.Fatalf("ctrl-p requests = %+v", reqs)
	}
	if m.mode != modePane {
		t.Fatalf("mode = %v, want PANE", m.mode)
	}

	if reqs := m.key(keyPress("%")); len(reqs) != 0 {
		t.Fatalf("split requests = %+v", reqs)
	}
	tab := m.activeTab()
	if tab.flow != "row" || len(tab.slots) != 2 {
		t.Fatalf("after %%: flow=%q slots=%d", tab.flow, len(tab.slots))
	}
	if tab.focus != 1 || tab.slots[1].sourceID != "" {
		t.Fatalf("new slot focus=%d source=%q", tab.focus, tab.slots[1].sourceID)
	}

	m.key(keyPress("tab"))
	if tab.focus != 0 {
		t.Fatalf("tab focus = %d, want 0", tab.focus)
	}

	m.key(keyPress(":"))
	if m.mode != modePrompt {
		t.Fatalf("mode = %v, want PROMPT", m.mode)
	}
	m.key(keyPress("q"))
	if m.promptText != "q" {
		t.Fatalf("prompt text = %q", m.promptText)
	}
	m.key(keyPress("backspace"))
	if m.promptText != "" {
		t.Fatalf("prompt text after backspace = %q", m.promptText)
	}
	m.key(keyPress("esc"))
	if m.mode != modeNormal {
		t.Fatalf("mode after prompt esc = %v, want NORMAL", m.mode)
	}

	m.key(keyPress("ctrl-p"))
	m.key(keyPress("?"))
	if m.mode != modeHelp {
		t.Fatalf("mode = %v, want HELP", m.mode)
	}
	m.key(keyPress("esc"))
	if m.mode != modeNormal {
		t.Fatalf("mode after help esc = %v, want NORMAL", m.mode)
	}

	m.key(keyPress("ctrl-p"))
	m.key(keyPress("ctrl-t"))
	if len(m.tabs) != 2 || m.active != 1 {
		t.Fatalf("after ctrl-t: tabs=%d active=%d", len(m.tabs), m.active)
	}
	if m.mode != modePicker {
		t.Fatalf("mode after ctrl-t = %v, want PICKER", m.mode)
	}
	m.key(keyPress("esc"))

	m.key(keyPress("ctrl-p"))
	m.key(keyPress("1"))
	if m.active != 0 {
		t.Fatalf("active after digit = %d, want 0", m.active)
	}
	if m.mode != modePane {
		t.Fatalf("mode after digit = %v, want PANE", m.mode)
	}

	m.key(keyPress("x"))
	if len(m.activeTab().slots) != 1 {
		t.Fatalf("slots after x = %d, want 1", len(m.activeTab().slots))
	}
	m.key(keyPress("x"))
	if len(m.activeTab().slots) != 1 || m.activeTab().slots[0].sourceID != "" {
		t.Fatalf("last slot must stay and unbind, got %d slots source=%q", len(m.activeTab().slots), m.activeTab().slots[0].sourceID)
	}
}

// TestModeStateMachineTable is the table-driven mode transition contract:
// each row sets up a model, sends one key and asserts the resulting mode.
func TestModeStateMachineTable(t *testing.T) {
	bound := func(m *model) { mustBind(t, m, "terminal:local:main") }
	cases := []struct {
		name  string
		setup func(*model)
		key   keyEvent
		want  mode
		check func(*testing.T, *model)
	}{
		{"normal ctrl-p enters pane", bound, keyPress("ctrl-p"), modePane, nil},
		{"normal ctrl-f opens picker", bound, keyPress("ctrl-f"), modePicker, nil},
		{"normal ctrl-t adds tab and picks", bound, keyPress("ctrl-t"), modePicker, func(t *testing.T, m *model) {
			if len(m.tabs) != 2 {
				t.Fatalf("tabs = %d", len(m.tabs))
			}
		}},
		{"pane esc returns normal", func(m *model) {
			bound(m)
			m.key(keyPress("ctrl-p"))
		}, keyPress("esc"), modeNormal, nil},
		{"pane percent stays pane and splits", func(m *model) {
			bound(m)
			m.key(keyPress("ctrl-p"))
		}, keyPress("%"), modePane, func(t *testing.T, m *model) {
			if len(m.activeTab().slots) != 2 || m.activeTab().flow != "row" {
				t.Fatalf("split = %d slots flow %q", len(m.activeTab().slots), m.activeTab().flow)
			}
		}},
		{"pane quote splits stacked", func(m *model) {
			bound(m)
			m.key(keyPress("ctrl-p"))
		}, keyPress("\""), modePane, func(t *testing.T, m *model) {
			if len(m.activeTab().slots) != 2 || m.activeTab().flow != "col" {
				t.Fatalf("split = %d slots flow %q", len(m.activeTab().slots), m.activeTab().flow)
			}
		}},
		{"pane colon opens prompt", func(m *model) {
			bound(m)
			m.key(keyPress("ctrl-p"))
		}, keyPress(":"), modePrompt, nil},
		{"pane question opens help", func(m *model) {
			bound(m)
			m.key(keyPress("ctrl-p"))
		}, keyPress("?"), modeHelp, nil},
		{"pane x closes slot", func(m *model) {
			bound(m)
			m.key(keyPress("ctrl-p"))
			m.key(keyPress("%"))
		}, keyPress("x"), modePane, func(t *testing.T, m *model) {
			if len(m.activeTab().slots) != 1 {
				t.Fatalf("slots = %d, want 1", len(m.activeTab().slots))
			}
		}},
		{"picker esc closes", func(m *model) {
			bound(m)
			m.key(keyPress("ctrl-f"))
		}, keyPress("esc"), modeNormal, nil},
		{"prompt esc closes", func(m *model) {
			bound(m)
			m.key(keyPress("ctrl-p"))
			m.key(keyPress(":"))
		}, keyPress("esc"), modeNormal, nil},
		{"help esc closes", func(m *model) {
			bound(m)
			m.key(keyPress("ctrl-p"))
			m.key(keyPress("?"))
		}, keyPress("esc"), modeNormal, nil},
		{"scroll esc returns live", func(m *model) {
			bound(m)
			reqs := m.key(keyPress("page-up"))
			if len(reqs) != 1 {
				t.Fatalf("page-up = %+v", reqs)
			}
			reqs[0].After(&pb.Response{Ok: true})
		}, keyPress("esc"), modeNormal, nil},
		{"pane digit switches tab", func(m *model) {
			bound(m)
			m.key(keyPress("ctrl-p"))
			m.key(keyPress("ctrl-t"))
			m.key(keyPress("esc"))
			m.key(keyPress("ctrl-p"))
		}, keyPress("1"), modePane, func(t *testing.T, m *model) {
			if m.active != 0 {
				t.Fatalf("active = %d, want 0", m.active)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newModel()
			testHello(m)
			tc.setup(m)
			m.key(tc.key)
			if m.mode != tc.want {
				t.Fatalf("mode = %v, want %v", m.mode, tc.want)
			}
			if tc.check != nil {
				tc.check(t, m)
			}
		})
	}
}

// TestKeybindingOverridesDriveStateMachine is the config -> state machine
// contract: the overridden keys drive the transitions and the old defaults
// become inert.
func TestKeybindingOverridesDriveStateMachine(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	cfg := config.Default()
	cfg.Keybindings = map[string]string{
		"picker.open": "ctrl-g",
		"pane.mode":   "ctrl-o",
		"slot.close":  "q",
	}
	if err := m.applyConfig(cfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}

	if reqs := m.key(keyPress("ctrl-p")); len(reqs) != 0 || m.mode != modeNormal {
		t.Fatalf("old pane.mode key must be inert, mode=%v", m.mode)
	}
	m.key(keyPress("ctrl-o"))
	if m.mode != modePane {
		t.Fatalf("ctrl-o must enter PANE, mode=%v", m.mode)
	}
	m.key(keyPress("%"))
	m.key(keyPress("q"))
	if len(m.activeTab().slots) != 1 {
		t.Fatalf("q must close the focused slot, slots=%d", len(m.activeTab().slots))
	}
	m.key(keyPress("x"))
	if len(m.activeTab().slots) != 1 {
		t.Fatalf("old slot.close key must be inert, slots=%d", len(m.activeTab().slots))
	}
	m.key(keyPress("ctrl-g"))
	if m.mode != modePicker {
		t.Fatalf("ctrl-g must open the picker, mode=%v", m.mode)
	}
}

func TestPickerBindExistingSource(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	m.key(keyPress("ctrl-p"))
	m.key(keyPress("x"))
	if m.focusSlot().sourceID != "" {
		t.Fatal("slot should be unbound")
	}

	m.key(keyPress("ctrl-f"))
	if m.mode != modePicker {
		t.Fatalf("mode = %v, want PICKER", m.mode)
	}
	reqs := m.key(keyEvent{ID: "ev:enter", Key: "enter"})
	if len(reqs) != 1 || reqs[0].Method != "terminal.attach" {
		t.Fatalf("picker enter requests = %+v", reqs)
	}
	if reqs[0].Params.GetId() != "main" || !reqs[0].Params.GetFit() {
		t.Fatalf("attach params = %+v", reqs[0].Params)
	}
	if m.mode != modeNormal {
		t.Fatalf("mode after pick = %v, want NORMAL", m.mode)
	}
	reqs[0].After(&pb.Response{RequestId: 1, Epoch: 1, Ok: true})
	if m.focusSlot().sourceID != "terminal:local:main" {
		t.Fatalf("slot source = %q", m.focusSlot().sourceID)
	}
	if m.status != "bound main" {
		t.Fatalf("status = %q, want bound main", m.status)
	}
}

func TestKillCommandRequestsTerminalKill(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	m.key(keyPress("ctrl-p"))
	m.key(keyPress(":"))
	for _, r := range "kill" {
		m.key(keyPress(string(r)))
	}
	reqs := m.key(keyPress("enter"))
	if len(reqs) != 1 || reqs[0].Method != "terminal.kill" {
		t.Fatalf("kill requests = %+v", reqs)
	}
	if reqs[0].Params.GetEndpoint() != "local" || reqs[0].Params.GetId() != "main" {
		t.Fatalf("kill params = %+v", reqs[0].Params)
	}
}

// TestPickerMouseClickBinds: clicking a picker row is the mouse equivalent of
// selecting it and pressing enter: it must bind that terminal to the focused
// slot in one press (not just move the selection).
func TestPickerMouseClickBinds(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	m.key(keyPress("ctrl-p"))
	m.key(keyPress("x"))
	if m.focusSlot().sourceID != "" {
		t.Fatal("setup: slot must be unbound")
	}
	m.key(keyPress("ctrl-f"))
	if m.mode != modePicker {
		t.Fatalf("mode = %v, want PICKER", m.mode)
	}

	reqs := m.mouse("press", "left", "pick:0", 5, 18)
	if len(reqs) != 1 || reqs[0].Method != "terminal.attach" {
		t.Fatalf("picker click requests = %+v", reqs)
	}
	if reqs[0].Params.GetId() != "main" || !reqs[0].Params.GetFit() {
		t.Fatalf("attach params = %+v", reqs[0].Params)
	}
	if m.mode != modeNormal {
		t.Fatalf("mode after picker click = %v, want NORMAL", m.mode)
	}
	reqs[0].After(&pb.Response{RequestId: 1, Epoch: 1, Ok: true})
	if m.focusSlot().sourceID != "terminal:local:main" {
		t.Fatalf("slot source = %q, want the clicked terminal", m.focusSlot().sourceID)
	}
}

// TestSidebarMouseClickSwitchesTab: the sidebar tab list must be clickable
// ("side:tab:<index>" boxes declaring mouse input) and a press switches the
// active tab without touching the layout.
func TestSidebarMouseClickSwitchesTab(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	m.key(keyPress("ctrl-t"))
	m.key(keyPress("esc"))
	if m.active != 1 || len(m.tabs) != 2 {
		t.Fatalf("setup: active=%d tabs=%d, want 1/2", m.active, len(m.tabs))
	}

	row := findBox(buildView(m), "side:tab:0")
	if row == nil {
		t.Fatal("sidebar must expose a stable side:tab:<index> hit box")
	}
	if len(row.GetInput()) != 1 || row.GetInput()[0] != "mouse" {
		t.Fatalf("sidebar row input = %v, want [mouse]", row.GetInput())
	}
	m.mouse("press", "left", "side:tab:0", 5, 6)
	if m.active != 0 {
		t.Fatalf("active after sidebar click = %d, want 0", m.active)
	}

	// The active row moves to the clicked tab and the old one loses it.
	box := buildView(m)
	if got := findBox(box, "side:tab:0").GetStyle(); got != m.style("selection") {
		t.Fatalf("clicked sidebar row style = %q, want selection", got)
	}
	if got := findBox(box, "side:tab:1").GetStyle(); got == m.style("selection") {
		t.Fatalf("old sidebar row kept the selection style")
	}
}

func TestPickerCreateNewTerminal(t *testing.T) {
	m := newModel()
	testHello(m)
	m.sourcesEvent(nil)
	if m.mode != modePicker {
		t.Fatalf("mode = %v, want PICKER", m.mode)
	}
	reqs := m.key(keyEvent{ID: "ev:enter", Key: "enter"})
	if len(reqs) != 1 || reqs[0].Method != "terminal.create" {
		t.Fatalf("create requests = %+v", reqs)
	}
	if reqs[0].Params.GetEndpoint() != "local" {
		t.Fatalf("create params = %+v", reqs[0].Params)
	}
	reqs[0].After(&pb.Response{Ok: true, Data: &pb.MethodData{Endpoint: "local", Id: "term-7"}})
	if got := m.focusSlot().sourceID; got != "terminal:local:term-7" {
		t.Fatalf("slot source = %q", got)
	}
}

func TestCtrlFDoubleClickForwards(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")

	m.key(keyEvent{ID: "ev:first", Key: "ctrl-f"})
	if m.mode != modePicker {
		t.Fatalf("mode = %v, want PICKER", m.mode)
	}
	reqs := m.key(keyEvent{ID: "ev:second", Key: "ctrl-f"})
	if len(reqs) != 1 || reqs[0].Method != "input.forward" {
		t.Fatalf("double-click requests = %+v", reqs)
	}
	if reqs[0].Params.GetEventId() != "ev:second" || reqs[0].Params.GetSource() != "terminal:local:main" {
		t.Fatalf("forward params = %+v", reqs[0].Params)
	}
	if m.mode != modeNormal {
		t.Fatalf("mode after forward = %v, want NORMAL", m.mode)
	}

	// Outside the window the second press just closes the picker.
	m.key(keyEvent{ID: "ev:third", Key: "ctrl-f"})
	m.lastCtrlF = time.Now().Add(-time.Second)
	if reqs := m.key(keyEvent{ID: "ev:fourth", Key: "ctrl-f"}); len(reqs) != 0 {
		t.Fatalf("slow double-click requests = %+v, want none", reqs)
	}
	if m.mode != modeNormal {
		t.Fatalf("mode = %v, want NORMAL", m.mode)
	}
}

func TestWheelAndScrollLifecycle(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	slot := m.focusSlot()

	reqs := m.wheel(slot.id, 1)
	if len(reqs) != 1 || reqs[0].Method != "terminal.scroll" {
		t.Fatalf("wheel requests = %+v", reqs)
	}
	if reqs[0].Params.GetDelta() != 3 {
		t.Fatalf("scroll delta = %d, want 3", reqs[0].Params.GetDelta())
	}
	reqs[0].After(&pb.Response{Ok: true, Data: &pb.MethodData{Rows: []string{"x"}}})
	if slot.scrollOffset != 3 || m.mode != modeScroll {
		t.Fatalf("offset=%d mode=%v", slot.scrollOffset, m.mode)
	}

	reqs = m.key(keyPress("page-down"))
	if len(reqs) != 1 || reqs[0].Params.GetDelta() != -10 {
		t.Fatalf("page-down requests = %+v", reqs)
	}
	reqs[0].After(&pb.Response{Ok: true})
	if slot.scrollOffset != 0 || m.mode != modeNormal {
		t.Fatalf("offset=%d mode=%v, want 0/NORMAL", slot.scrollOffset, m.mode)
	}

	// Scrolling again then esc returns to live through scrollEnd.
	reqs = m.key(keyPress("page-up"))
	reqs[0].After(&pb.Response{Ok: true})
	reqs = m.key(keyPress("esc"))
	if len(reqs) != 1 || reqs[0].Method != "terminal.scrollEnd" {
		t.Fatalf("scroll esc requests = %+v", reqs)
	}
	if slot.scrollOffset != 0 || m.mode != modeNormal {
		t.Fatalf("after scrollEnd offset=%d mode=%v", slot.scrollOffset, m.mode)
	}
}

func TestDividerDragAdjustsRatios(t *testing.T) {
	m := newModel()
	testHello(m)
	m.key(keyPress("ctrl-p"))
	m.key(keyPress("%"))
	tab := m.activeTab()
	if tab.flow != "row" || len(tab.slots) != 2 {
		t.Fatalf("setup: flow=%q slots=%d", tab.flow, len(tab.slots))
	}
	before := [2]int{tab.slots[0].ratio, tab.slots[1].ratio}

	m.mouse("press", "left", "divider:0", 50, 5)
	m.mouse("drag", "left", "divider:0", 60, 5)
	if tab.slots[0].ratio <= before[0] || tab.slots[1].ratio >= before[1] {
		t.Fatalf("ratios after drag = %d/%d, want left>%d right<%d", tab.slots[0].ratio, tab.slots[1].ratio, before[0], before[1])
	}
	if tab.slots[0].ratio+tab.slots[1].ratio != before[0]+before[1] {
		t.Fatalf("ratios must conserve the total")
	}
	m.mouse("release", "left", "divider:0", 60, 5)
	if m.drag != nil {
		t.Fatal("drag must be cleared on release")
	}
	m.mouse("drag", "left", "divider:0", 30, 5)
	if tab.slots[0].ratio+tab.slots[1].ratio != before[0]+before[1] {
		t.Fatal("drag after release must be ignored")
	}
}

func TestMouseClickFocusAndTabSwitch(t *testing.T) {
	m := newModel()
	testHello(m)
	m.key(keyPress("ctrl-p"))
	m.key(keyPress("%"))
	tab := m.activeTab()
	first, second := tab.slots[0].id, tab.slots[1].id
	if tab.focus != 1 {
		t.Fatalf("focus = %d, want 1", tab.focus)
	}
	m.mouse("press", "left", first, 3, 3)
	if tab.focus != 0 {
		t.Fatalf("focus after click = %d, want 0", tab.focus)
	}
	m.mouse("press", "left", second, 60, 3)
	if tab.focus != 1 {
		t.Fatalf("focus after click = %d, want 1", tab.focus)
	}
	m.key(keyPress("ctrl-t"))
	m.key(keyPress("esc"))
	m.key(keyPress("ctrl-p"))
	m.mouse("press", "left", "tab:0", 5, 1)
	if m.active != 0 {
		t.Fatalf("active tab = %d, want 0", m.active)
	}
}

func TestRestartAndCopyRequests(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	m.sourcesEvent([]*pb.Source{terminalSource("terminal:local:main", "main", true)})

	reqs := m.key(keyPress("ctrl-e"))
	if len(reqs) != 1 || reqs[0].Method != "terminal.restart" {
		t.Fatalf("ctrl-e requests = %+v", reqs)
	}
	if reqs[0].Params.GetEndpoint() != "local" || reqs[0].Params.GetId() != "main" {
		t.Fatalf("restart params = %+v", reqs[0].Params)
	}

	m.key(keyPress("ctrl-p"))
	reqs = m.key(keyPress(":"))
	if len(reqs) != 0 {
		t.Fatalf("prompt open requests = %+v", reqs)
	}
	for _, r := range "copy" {
		m.key(keyPress(string(r)))
	}
	reqs = m.key(keyPress("enter"))
	if len(reqs) != 1 || reqs[0].Method != "terminal.copy" {
		t.Fatalf("copy requests = %+v", reqs)
	}
}

func TestQuitCommandRequestsHostConfirm(t *testing.T) {
	m := newModel()
	testHello(m)
	m.key(keyPress("ctrl-p"))
	m.key(keyPress(":"))
	for _, r := range "quit" {
		m.key(keyPress(string(r)))
	}
	reqs := m.key(keyPress("enter"))
	if len(reqs) != 1 || reqs[0].Method != "system.quit" {
		t.Fatalf("quit requests = %+v", reqs)
	}
	if reqs[0].Params.GetCleanupOwned() {
		t.Fatal("cleanup_owned must default to false (unbind-style lifecycle)")
	}
}

func TestSourceReconcileUnbindsMissing(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	m.sourcesEvent([]*pb.Source{terminalSource("terminal:local:other", "other", false)})
	if m.focusSlot().sourceID != "" {
		t.Fatalf("slot source = %q, want unbound after source vanished", m.focusSlot().sourceID)
	}
}

// TestSlotButtonDispatchTable is the hit-dispatch contract: each of the four
// title-bar buttons maps to the documented program action (the same ones the
// PANE keys `%`, `"`, `x` and Ctrl-E use).
func TestSlotButtonDispatchTable(t *testing.T) {
	cases := []struct {
		name   string
		button string
		check  func(*testing.T, *model, []request)
	}{
		{
			name:   "restart emits terminal.restart and tracks pending",
			button: slotButtonRestart,
			check: func(t *testing.T, m *model, reqs []request) {
				if len(reqs) != 1 || reqs[0].Method != "terminal.restart" {
					t.Fatalf("requests = %+v", reqs)
				}
				if reqs[0].Params.GetEndpoint() != "local" || reqs[0].Params.GetId() != "main" {
					t.Fatalf("restart params = %+v", reqs[0].Params)
				}
				if m.focusSlot().pending != slotButtonRestart {
					t.Fatalf("pending = %q, want restart", m.focusSlot().pending)
				}
				reqs[0].After(&pb.Response{Ok: true})
				if m.focusSlot().pending != "" {
					t.Fatalf("pending after response = %q", m.focusSlot().pending)
				}
			},
		},
		{
			name:   "split-h splits side by side like %",
			button: slotButtonSplitH,
			check: func(t *testing.T, m *model, reqs []request) {
				if len(reqs) != 0 {
					t.Fatalf("split requests = %+v", reqs)
				}
				tab := m.activeTab()
				if tab.flow != "row" || len(tab.slots) != 2 || tab.focus != 1 {
					t.Fatalf("split-h = flow %q slots %d focus %d", tab.flow, len(tab.slots), tab.focus)
				}
			},
		},
		{
			name:   "split-v splits stacked like quote",
			button: slotButtonSplitV,
			check: func(t *testing.T, m *model, reqs []request) {
				if len(reqs) != 0 {
					t.Fatalf("split requests = %+v", reqs)
				}
				tab := m.activeTab()
				if tab.flow != "col" || len(tab.slots) != 2 || tab.focus != 1 {
					t.Fatalf("split-v = flow %q slots %d focus %d", tab.flow, len(tab.slots), tab.focus)
				}
			},
		},
		{
			name:   "close unbinds the last slot like x",
			button: slotButtonClose,
			check: func(t *testing.T, m *model, reqs []request) {
				if len(reqs) != 0 {
					t.Fatalf("close requests = %+v", reqs)
				}
				tab := m.activeTab()
				if len(tab.slots) != 1 || tab.slots[0].sourceID != "" {
					t.Fatalf("close = %d slots source %q, want one unbound slot", len(tab.slots), tab.slots[0].sourceID)
				}
				if m.sourceByID("terminal:local:main") == nil {
					t.Fatal("close must keep the terminal alive for rebinding")
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newModel()
			testHello(m)
			mustBind(t, m, "terminal:local:main")
			node := slotButtonNode(m.focusSlot().id, tc.button)
			reqs := m.mouse("press", "left", node, 3, 0)
			if m.pressed != node {
				t.Fatalf("pressed = %q, want %q", m.pressed, node)
			}
			tc.check(t, m, reqs)
		})
	}
}

// TestSlotButtonTargetsItsOwnSlot: a button acts on the slot it belongs to,
// not on whichever slot happens to be focused.
func TestSlotButtonTargetsItsOwnSlot(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	m.key(keyPress("ctrl-p"))
	m.key(keyPress("%"))
	tab := m.activeTab()
	if tab.focus != 1 || tab.slots[0].sourceID != "terminal:local:main" {
		t.Fatalf("setup focus = %d slot0 = %q", tab.focus, tab.slots[0].sourceID)
	}
	node := slotButtonNode(tab.slots[0].id, slotButtonRestart)
	reqs := m.mouse("press", "left", node, 3, 0)
	if tab.focus != 0 {
		t.Fatalf("focus after button press = %d, want 0", tab.focus)
	}
	if len(reqs) != 1 || reqs[0].Method != "terminal.restart" || reqs[0].Params.GetId() != "main" {
		t.Fatalf("requests = %+v, want the slot-0 terminal.restart", reqs)
	}
	if tab.slots[0].pending != slotButtonRestart || tab.slots[1].pending != "" {
		t.Fatalf("pending = %q/%q, want restart on the clicked slot only", tab.slots[0].pending, tab.slots[1].pending)
	}
}

// TestSlotButtonRestartWithoutTerminal: restart on an empty slot is a no-op
// plus a footer toast, never a protocol call.
func TestSlotButtonRestartWithoutTerminal(t *testing.T) {
	m := newModel()
	testHello(m)
	s := m.focusSlot()
	node := slotButtonNode(s.id, slotButtonRestart)
	reqs := m.mouse("press", "left", node, 3, 0)
	if len(reqs) != 0 {
		t.Fatalf("empty-slot restart requests = %+v, want none", reqs)
	}
	if s.pending != "" {
		t.Fatalf("pending = %q, want empty", s.pending)
	}
	if !contains(m.footerRight(), "no terminal to restart") {
		t.Fatalf("status = %q, want the restart toast", m.footerRight())
	}
}

// TestAttachTargetBindsMatchingSource: -attach picks the named terminal
// instead of the first source.
func TestAttachTargetBindsMatchingSource(t *testing.T) {
	m := newModel()
	testHello(m)
	m.attachTarget = "term-b"
	reqs := m.sourcesEvent([]*pb.Source{
		terminalSource("terminal:local:term-a", "a", false),
		terminalSource("terminal:local:term-b", "b", false),
	})
	if len(reqs) != 1 || reqs[0].Method != "terminal.attach" || reqs[0].Params.GetId() != "term-b" {
		t.Fatalf("attach target requests = %+v", reqs)
	}
	if got := m.focusSlot().sourceID; got != "terminal:local:term-b" {
		t.Fatalf("focus slot source = %q, want terminal:local:term-b", got)
	}
	if m.attachTarget != "" {
		t.Fatalf("attach target was not consumed: %q", m.attachTarget)
	}
}

// TestAttachTargetFallsBackToPickerWhenMissing: an unknown -attach target
// keeps the normal cold-start shape.
func TestAttachTargetFallsBackToPickerWhenMissing(t *testing.T) {
	m := newModel()
	testHello(m)
	m.autoAttachFirst = false
	m.attachTarget = "nope"
	reqs := m.sourcesEvent([]*pb.Source{terminalSource("terminal:local:term-a", "a", false)})
	if len(reqs) != 0 {
		t.Fatalf("missing attach target requests = %+v, want none", reqs)
	}
	if m.mode != modePicker {
		t.Fatalf("mode = %v, want PICKER", m.mode)
	}
	if m.focusSlot().sourceID != "" {
		t.Fatalf("focus slot source = %q, want empty", m.focusSlot().sourceID)
	}
}

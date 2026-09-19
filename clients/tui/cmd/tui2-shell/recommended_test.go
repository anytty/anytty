package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/config"
	"github.com/anytty/anytty/clients/tui/runtime"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// applyTestConfig replaces the model config with a parsed custom document so
// the recommended theme/icons/endpoints tests never touch the file system.
func applyTestConfig(t *testing.T, m *model, data string) config.Config {
	t.Helper()
	cfg, err := config.Parse([]byte(data))
	if err != nil {
		t.Fatalf("parse test config: %v", err)
	}
	if err := m.applyConfig(cfg); err != nil {
		t.Fatalf("apply test config: %v", err)
	}
	return cfg
}

// TestRecommendedIconsInViewTree pins M1: the default config resolves the
// legacy recommended Nerd Font icons and every chrome region carries them
// (workspace chip, tab marker, new-tab box, title-bar buttons, footer mode
// badge, status summaries, picker rows).
func TestRecommendedIconsInViewTree(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	box := buildView(m)

	workspace := findBox(box, "workspace")
	if got := contentText(workspace); !strings.Contains(got, "\U000F0645") {
		t.Fatalf("workspace chip = %q, want the recommended folder icon", got)
	}
	if got := contentText(findBox(box, "tab:0")); !strings.Contains(got, "\u2387") {
		t.Fatalf("active tab = %q, want the ⎇ marker", got)
	}
	if got := contentText(findBox(box, "tab:new")); !strings.Contains(got, "\U000F0415") {
		t.Fatalf("new-tab box = %q, want the recommended plus icon", got)
	}

	bar := findBox(box, "title:"+m.focusSlot().id)
	for action, want := range map[string]string{
		slotButtonRestart: "\U000F0450",
		slotButtonSplitH:  "\uEB56",
		slotButtonSplitV:  "\uEB57",
		slotButtonClose:   "\U000F0156",
	} {
		node := findBox(bar, slotButtonNode(m.focusSlot().id, action))
		if node == nil || contentText(node) != want {
			t.Fatalf("button %s = %q, want %q", action, contentText(node), want)
		}
	}

	footer := treeText(findBox(box, "footer"))
	for _, want := range []string{"\U000F030C", "\U000F0C7C", "\U000F04E9"} {
		if !strings.Contains(footer, want) {
			t.Fatalf("footer = %q, want recommended icon %q", footer, want)
		}
	}
	if got := m.modeIcon(modeScroll); got != "\U000F018F" {
		t.Fatalf("scroll mode icon = %q", got)
	}

	// The picker lists endpoints with the recommended connection icon and
	// the create row with the plus icon.
	m.openPicker()
	picker := treeText(findBox(buildView(m), "overlay:terminals"))
	if !strings.Contains(picker, "\U000F0415") {
		t.Fatalf("picker create row = %q, want the plus icon", picker)
	}
}

// TestRecommendedFinalFrameCarriesIcons proves the icons survive the whole
// host pipeline: the committed view is solved by the real runtime and the
// composed ANSI frame (the bytes the host would write) contains the glyphs.
func TestRecommendedFinalFrameCarriesIcons(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	root, keys := m.view()
	view := &pb.View{
		Epoch: 1,
		Rev:   1,
		Keys:  &pb.Keys{Claim: append([]string(nil), keys.Claim...), All: keys.All},
		Root:  root.Build(),
	}
	session := runtime.NewSession(runtime.Options{ViewID: "view:local:1", Cols: m.cols, Rows: m.rows}, nil, nil)
	if err := session.HandleView(view); err != nil {
		t.Fatalf("HandleView: %v", err)
	}
	frameBytes := session.FrameBytes(nil, nil)
	for _, want := range []string{
		"\U000F0645",                     // workspace
		"\u2387",                         // tab marker
		"\U000F0415",                     // new tab
		"\U000F0450",                     // restart button
		"\uEB56", "\uEB57", "\U000F0156", // split/close buttons
		"\U000F030C", // NORMAL mode badge
		"\U000F04E9", // tab summary
	} {
		if !bytes.Contains(frameBytes, []byte(want)) {
			t.Fatalf("final frame does not contain icon %q", want)
		}
	}
}

// TestIconsPresetAndMapOverride pins the configuration surface: the legacy
// unicode preset restores the pre-recommended glyphs, and a map override
// changes a single name on top of any preset.
func TestIconsPresetAndMapOverride(t *testing.T) {
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")

	applyTestConfig(t, m, `{"icons": "unicode"}`)
	box := buildView(m)
	if got := contentText(findBox(box, "workspace")); got != " local " {
		t.Fatalf("unicode workspace = %q, want no icon", got)
	}
	if got := contentText(findBox(box, "tab:new")); got != " + " {
		t.Fatalf("unicode plus = %q", got)
	}
	if got := treeText(findBox(box, "title:"+m.focusSlot().id)); !strings.Contains(got, "▎") {
		t.Fatalf("unicode title bar = %q (slot %q)", got, m.focusSlot().id)
	}
	closeBtn := findBox(box, slotButtonNode(m.focusSlot().id, slotButtonClose))
	if contentText(closeBtn) != "✕" {
		t.Fatalf("unicode close button = %q", contentText(closeBtn))
	}

	applyTestConfig(t, m, `{"icons": {"preset": "recommended", "map": {"tab_new": "N", "slot_close": "C"}}}`)
	box = buildView(m)
	if got := contentText(findBox(box, "tab:new")); got != " N " {
		t.Fatalf("overridden plus = %q", got)
	}
	closeBtn = findBox(box, slotButtonNode(m.focusSlot().id, slotButtonClose))
	if contentText(closeBtn) != "C" {
		t.Fatalf("overridden close button = %q", contentText(closeBtn))
	}
}

// TestRecommendedSplitKeybindings pins M2: pane.split_h/pane.split_v are real
// whitelisted actions (defaults `%`/`"`), so the recommended profile's panel
// ctrl-d/ctrl-e splits (and the ctrl-r reconnect-style restart) are a
// documented keybindings override away and reach the same program actions.
func TestRecommendedSplitKeybindings(t *testing.T) {
	const recommendedKeys = `{"keybindings": {
		"pane.split_h": "ctrl-d",
		"pane.split_v": "ctrl-e",
		"terminal.restart": "ctrl-r"
	}}`
	if _, err := config.Parse([]byte(recommendedKeys)); err != nil {
		t.Fatalf("split/restart actions must be whitelisted: %v", err)
	}
	m := newModel()
	testHello(m)
	mustBind(t, m, "terminal:local:main")
	applyTestConfig(t, m, recommendedKeys)

	m.key(keyPress("ctrl-p"))
	m.key(keyPress("ctrl-d"))
	tab := m.activeTab()
	if len(tab.slots) != 2 || tab.flow != "row" {
		t.Fatalf("ctrl-d split: slots=%d flow=%q, want 2/row", len(tab.slots), tab.flow)
	}
	m.key(keyPress("ctrl-e"))
	if len(tab.slots) != 3 || tab.flow != "col" {
		t.Fatalf("ctrl-e split: slots=%d flow=%q, want 3/col", len(tab.slots), tab.flow)
	}
	// ctrl-r is the recommended restart key in this override; focus the
	// bound slot first (the two splits left an empty slot focused).
	tab.focus = 0
	reqs := m.key(keyPress("ctrl-r"))
	if len(reqs) != 1 || reqs[0].Method != "terminal.restart" {
		t.Fatalf("ctrl-r requests = %+v", reqs)
	}
	// The default `%`/`"`/Ctrl-E bindings are replaced, not duplicated.
	if got := m.binds.key(actionPaneSplitH); got != "ctrl-d" {
		t.Fatalf("pane.split_h = %q", got)
	}
	defaults := newModel()
	if got := defaults.binds.key(actionPaneSplitH); got != "%" {
		t.Fatalf("default pane.split_h = %q, want %%", got)
	}
	if got := defaults.binds.key(actionPaneSplitV); got != "\"" {
		t.Fatalf("default pane.split_v = %q, want quote", got)
	}
	if got := defaults.binds.key(actionTerminalRestart); got != "ctrl-e" {
		t.Fatalf("default terminal.restart = %q, want ctrl-e", got)
	}
}

// TestPickerListsConfiguredEndpoints pins M3: the picker lists each
// configured command endpoint under its display name with the recommended
// connection icon, before the create row.
func TestPickerListsConfiguredEndpoints(t *testing.T) {
	m := newModel()
	testHello(m)
	applyTestConfig(t, m, `{
		"endpoints": [
			{"name": "remote", "kind": "command", "label": "dev box",
			 "argv": ["sh", "-lc", "echo REMOTE-READY; exec cat"]}
		]
	}`)
	m.sourcesEvent(nil)

	items := m.pickerItems()
	if len(items) != 2 || items[0].endpoint == nil || items[1].endpoint != nil {
		t.Fatalf("picker items = %+v", items)
	}
	if items[0].label != "\U000F0337 dev box" || items[0].info != "endpoint · command sh" {
		t.Fatalf("endpoint row = %+v", items[0])
	}
	picker := treeText(findBox(buildView(m), "overlay:terminals"))
	if !strings.Contains(picker, "dev box") || !strings.Contains(picker, "\U000F0337") {
		t.Fatalf("picker overlay = %q, want the endpoint name and icon", picker)
	}
}

// TestEndpointPickCreatesCommandTerminal pins the M3 launch path: selecting
// an endpoint row emits terminal.create with the endpoint name and its
// command/cwd/env, and the response binds terminal:<endpoint>:<id>.
func TestEndpointPickCreatesCommandTerminal(t *testing.T) {
	m := newModel()
	testHello(m)
	applyTestConfig(t, m, `{
		"endpoints": [
			{"name": "remote", "kind": "command",
			 "argv": ["sh", "-lc", "echo REMOTE-READY; exec cat"],
			 "cwd": "/tmp", "env": {"ANYTTY_REMOTE": "1"}}
		]
	}`)
	m.openPicker()

	items := m.pickerItems()
	m.pickerIdx = 0
	reqs := m.pick(items)
	if len(reqs) != 1 || reqs[0].Method != "terminal.create" {
		t.Fatalf("pick requests = %+v", reqs)
	}
	params := reqs[0].Params
	if params.GetEndpoint() != "remote" || strings.Join(params.GetArgv(), " ") != "sh -lc echo REMOTE-READY; exec cat" {
		t.Fatalf("create params = %+v", params)
	}
	if params.GetCwd() != "/tmp" || params.GetEnv()["ANYTTY_REMOTE"] != "1" {
		t.Fatalf("create cwd/env = %q %v", params.GetCwd(), params.GetEnv())
	}
	if m.mode != modeNormal {
		t.Fatalf("mode after pick = %v", m.mode)
	}

	reqs[0].After(&pb.Response{Ok: true, Data: &pb.MethodData{Endpoint: "remote", Id: "term-1"}})
	if got := m.focusSlot().sourceID; got != "terminal:remote:term-1" {
		t.Fatalf("bound source = %q", got)
	}
	if !strings.Contains(m.status, "remote:term-1") {
		t.Fatalf("status = %q, want the endpoint-qualified id", m.status)
	}

	// The remote source reconciles and keeps the endpoint in its id.
	m.sourcesEvent([]*pb.Source{{
		Id: "terminal:remote:term-1", Kind: "terminal", Endpoint: "remote",
		TerminalId: "term-1", Title: "remote shell", Attached: true,
	}})
	if got := m.focusSlot().sourceID; got != "terminal:remote:term-1" {
		t.Fatalf("source after snapshot = %q", got)
	}
}

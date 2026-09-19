package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// mode is the program-side input mode. Business state stays in this package;
// the host only ever sees boxes, sources and a claim.
type mode string

const (
	modeNormal mode = "NORMAL"
	modePane   mode = "PANE"
	modeScroll mode = "SCROLL"
	modePicker mode = "PICKER"
	modePrompt mode = "PROMPT"
	modeHelp   mode = "HELP"
)

// defaultForwardWindow is the picker double-press window of SCENARIOS §8.1:
// the second press inside the window is forwarded into the focused terminal.
// Config overrides it with double_click_forward_ms.
const defaultForwardWindow = 300 * time.Millisecond

// minRatio keeps a divider drag from collapsing a slot to zero cells.
const minRatio = 10

// Configurable actions (config.ActionWhitelist). The state machine routes
// keys through action names so "keybindings" can rebind them.
const (
	actionPaneMode        = "pane.mode"
	actionPaneSplitH      = "pane.split_h"
	actionPaneSplitV      = "pane.split_v"
	actionPickerOpen      = "picker.open"
	actionTabNew          = "tab.new"
	actionSlotClose       = "slot.close"
	actionScrollCopy      = "scroll.copy"
	actionSidebarToggle   = "sidebar.toggle"
	actionTerminalRestart = "terminal.restart"
	actionPromptOpen      = "prompt.open"
	actionHelpOpen        = "help.open"
)

// Slot title-bar action buttons. A button is not a framework concept: the
// program draws one mouse-input box per icon and resolves the hit itself
// (CUSTOMIZE §5). The keyboard equivalents in PANE mode are the existing
// bindings (pane.split_h/pane.split_v default `%`/`"`, `x` close,
// `Ctrl-E` restart).
const (
	slotButtonRestart = "restart"
	slotButtonSplitH  = "split-h"
	slotButtonSplitV  = "split-v"
	slotButtonClose   = "close"
)

// slotButton is one button of the right-side title-bar group: the program
// action it dispatches and the icon name resolved through m.icons.
type slotButton struct {
	action string
	icon   string
}

// slotButtons is the fixed action order of every slot title bar.
var slotButtons = []string{slotButtonRestart, slotButtonSplitH, slotButtonSplitV, slotButtonClose}

// slotButtonNode is the hit-test id of one title-bar button. The slot id is
// embedded so the same four buttons can coexist in every slot.
func slotButtonNode(slotID, action string) string { return "btn:" + slotID + ":" + action }

// parseButtonNode splits "btn:<slotID>:<action>" back into its parts.
func parseButtonNode(node string) (slotID, action string, ok bool) {
	rest, found := strings.CutPrefix(node, "btn:")
	if !found {
		return "", "", false
	}
	slotID, action, found = strings.Cut(rest, ":")
	if !found || slotID == "" || action == "" {
		return "", "", false
	}
	return slotID, action, true
}

// defaultKeybindings is the built-in action -> key table. Keys follow the
// legacy default TUI (which the recommended profile shares); the recommended
// profile's panel ctrl-d/ctrl-e splits are available as overrides for
// pane.split_h/pane.split_v.
func defaultKeybindings() map[string]string {
	return map[string]string{
		actionPaneMode:        "ctrl-p",
		actionPaneSplitH:      "%",
		actionPaneSplitV:      "\"",
		actionPickerOpen:      "ctrl-f",
		actionTabNew:          "ctrl-t",
		actionSlotClose:       "x",
		actionScrollCopy:      "y",
		actionSidebarToggle:   "ctrl-w",
		actionTerminalRestart: "ctrl-e",
		actionPromptOpen:      ":",
		actionHelpOpen:        "?",
	}
}

// keybindings is the resolved action <-> key table. A rebinding that makes
// two actions share a key is rejected at load time, so the reverse map is
// unambiguous.
type keybindings struct {
	keys map[string]string // action -> key
	rev  map[string]string // key -> action
}

// newKeybindings merges overrides onto the defaults. An unknown action is
// ignored (config.Validate already reports it); two actions ending on the
// same key return an error so the caller can fall back to the defaults.
func newKeybindings(overrides map[string]string) (keybindings, error) {
	keys := defaultKeybindings()
	for action, key := range overrides {
		if _, known := keys[action]; known {
			keys[action] = key
		}
	}
	rev := make(map[string]string, len(keys))
	for action, key := range keys {
		if other, clash := rev[key]; clash {
			return keybindings{}, fmt.Errorf("keybindings: %s and %s both use %q", other, action, key)
		}
		rev[key] = action
	}
	return keybindings{keys: keys, rev: rev}, nil
}

// action returns the action bound to key, or "".
func (k keybindings) action(key string) string { return k.rev[key] }

// key returns the key bound to action, or "".
func (k keybindings) key(action string) string { return k.keys[action] }

// slot is one pane of a tab (SCENARIOS §10). sourceID is the only binding to
// a host content source; an empty sourceID means "empty slot". pending names
// the title-bar action whose method call is still in flight ("" = idle).
type slot struct {
	id           string
	ratio        int
	sourceID     string
	scrollOffset int
	pending      string
}

// tab is one flat split group. flow is "row" (panes side by side) or "col"
// (panes stacked); ratios drive the cell distribution.
type tab struct {
	id    string
	name  string
	flow  string
	slots []*slot
	focus int
}

// keyEvent is the normalized key the host delivered.
type keyEvent struct {
	ID   string
	Key  string
	Char string
}

// request is one method call the model wants the program to emit.
type request struct {
	Method string
	Params *pb.MethodParams
	After  func(*pb.Response)
}

// endpointSpec is one configured launch target (tui2.json "endpoints"): the
// command model starts a local PTY whose argv is the endpoint command; the
// daemon model connects the host to an existing anytty daemon
// (ENDPOINTS.zh-CN.md §2).
type endpointSpec struct {
	name              string
	label             string
	kind              string
	argv              []string
	cwd               string
	env               map[string]string
	socket            string
	address           string
	connectMode       string
	signaling         []string
	iceTCP            []string
	daemonDeviceID    string
	daemonFingerprint string
	credentialDir     string
	credentialRef     string
	cloudGateway      string
}

// isDaemon reports whether the endpoint uses the daemon connect strategy.
func (e endpointSpec) isDaemon() bool {
	return e.kind == "daemon"
}

// displayName is the picker label of the endpoint.
func (e endpointSpec) displayName() string {
	if e.label != "" {
		return e.label
	}
	return e.name
}

// pickerItem is one row of the terminal picker: a known source, a configured
// endpoint, or the "+ New terminal" entry (both nil). info is the short
// endpoint/state column of the row.
type pickerItem struct {
	label    string
	info     string
	source   *pb.Source
	endpoint *endpointSpec
}

// dragState is the program half of a divider drag (PROTOCOL §6.7 capture is
// the host half): the ratios are snapshotted on press so every drag frame
// adjusts from the same origin.
type dragState struct {
	divider int
	startX  int
	startY  int
	ratios  []int
	axis    string
	avail   int
}

// model is the whole TUI: workspace > tabs > slots, the input mode, the
// source snapshot and the overlay state (SCENARIOS §10).
type model struct {
	viewID string
	epoch  uint64
	schema uint32
	cols   int
	rows   int

	tabs   []*tab
	active int
	mode   mode

	// Appearance and behavior knobs, all set from config (config.go).
	// The theme is program-side: view.go sends explicit styles on the wire.
	theme           theme
	icons           map[string]string
	endpoints       []endpointSpec
	binds           keybindings
	gap             int
	sidebar         bool
	clockEnabled    bool
	clockLayout     string
	forwardWindow   time.Duration
	autoAttachFirst bool
	startupCwd      string
	// attachTarget is the -attach terminal request: <id> or <endpoint>:<id>.
	// It binds as soon as the matching source appears; otherwise the normal
	// cold-start shape (picker/auto-attach) stays in effect.
	attachTarget string

	status string

	sources         []*pb.Source
	endpointSources []*pb.Source
	sourcesReady    bool

	pickerIdx  int
	promptText string
	promptIdx  int

	seq int

	drag      *dragState
	lastCtrlF time.Time
	now       func() time.Time

	// pressed is the title-bar button node currently held down. It is set on
	// a mouse press (the committed frame highlights it) and cleared by the
	// release or the next redraw (program.commit).
	pressed string
}

// newModel builds the default layout (one workspace, one tab, one empty
// slot) plus the built-in config defaults, so tests and -print-default-config
// agree without reading any file.
func newModel() *model {
	m := &model{cols: 80, rows: 24}
	m.applyDefaults()
	m.tabs = []*tab{m.newTab("1")}
	m.active = 0
	m.mode = modeNormal
	return m
}

func (m *model) nextID(prefix string) string {
	m.seq++
	return prefix + "-" + strconv.Itoa(m.seq)
}

func (m *model) newTab(name string) *tab {
	if name == "" {
		name = strconv.Itoa(len(m.tabs) + 1)
	}
	t := &tab{
		id:   m.nextID("tab"),
		name: name,
		flow: "col",
		slots: []*slot{{
			id:    m.nextID("slot"),
			ratio: 100,
		}},
	}
	return t
}

// activeTab returns the current tab, creating one if the model lost it.
func (m *model) activeTab() *tab {
	if len(m.tabs) == 0 {
		m.tabs = append(m.tabs, m.newTab("1"))
		m.active = 0
	}
	if m.active < 0 || m.active >= len(m.tabs) {
		m.active = 0
	}
	return m.tabs[m.active]
}

// focusSlot returns the focused slot of the active tab (never nil for a
// well-formed tab).
func (m *model) focusSlot() *slot {
	t := m.activeTab()
	if len(t.slots) == 0 {
		t.slots = append(t.slots, &slot{id: m.nextID("slot"), ratio: 100})
		t.focus = 0
	}
	if t.focus < 0 || t.focus >= len(t.slots) {
		t.focus = 0
	}
	return t.slots[t.focus]
}

func (m *model) findSlot(id string) (*tab, int) {
	for _, t := range m.tabs {
		for i, s := range t.slots {
			if s.id == id {
				return t, i
			}
		}
	}
	return nil, -1
}

func (m *model) sourceByID(id string) *pb.Source {
	for _, src := range m.sources {
		if src.GetId() == id && src.GetKind() == "terminal" {
			return src
		}
	}
	return nil
}

func (m *model) sourceAttached(id string) bool {
	src := m.sourceByID(id)
	return src != nil && src.GetAttached() && !src.GetExited()
}

func (m *model) hasBinding() bool {
	for _, t := range m.tabs {
		for _, s := range t.slots {
			if s.sourceID != "" {
				return true
			}
		}
	}
	return false
}

// parseSourceID splits "terminal:<endpoint>:<id>".
func parseSourceID(sourceID string) (endpoint, id string, ok bool) {
	parts := strings.SplitN(sourceID, ":", 3)
	if len(parts) != 3 || parts[0] != "terminal" || parts[1] == "" || parts[2] == "" {
		return "", "", false
	}
	return parts[1], parts[2], true
}

func terminalSourceID(endpoint, id string) string {
	return "terminal:" + endpoint + ":" + id
}

// shortSourceID is the display form of a source id: the terminal id alone.
func shortSourceID(sourceID string) string {
	if _, id, ok := parseSourceID(sourceID); ok {
		return id
	}
	return sourceID
}

// hello records the handshake and the viewport the host gave us.
func (m *model) hello(h *pb.Hello) {
	m.viewID = h.GetViewId()
	m.epoch = h.GetEpoch()
	m.schema = h.GetSchema()
	m.cols = int(h.GetCols())
	m.rows = int(h.GetRows())
}

func (m *model) resize(cols, rows int) {
	if cols > 0 {
		m.cols = cols
	}
	if rows > 0 {
		m.rows = rows
	}
}

func (m *model) notice(level, message string) {
	m.status = message
}

// sources reconciles the authoritative source snapshot (PROTOCOL §9.2). The
// first snapshot decides the cold-start shape: picker when empty, auto-bind
// when there is a terminal to adopt.
func (m *model) sourcesEvent(items []*pb.Source) []request {
	m.sources = m.sources[:0]
	m.endpointSources = m.endpointSources[:0]
	for _, src := range items {
		switch src.GetKind() {
		case "terminal":
			m.sources = append(m.sources, src)
		case "endpoint":
			m.endpointSources = append(m.endpointSources, src)
		}
	}
	sort.SliceStable(m.sources, func(i, j int) bool { return m.sources[i].GetId() < m.sources[j].GetId() })
	sort.SliceStable(m.endpointSources, func(i, j int) bool { return m.endpointSources[i].GetId() < m.endpointSources[j].GetId() })

	if m.attachTarget != "" && !m.hasBinding() {
		if src := m.findAttachTarget(); src != nil {
			m.attachTarget = ""
			slot := m.focusSlot()
			m.bind(slot.id, src.GetId())
			return []request{m.attachRequest(src.GetId())}
		}
	}
	if !m.sourcesReady {
		m.sourcesReady = true
		if len(m.sources) == 0 {
			m.openPicker()
			return nil
		}
		if !m.hasBinding() {
			if !m.autoAttachFirst {
				m.openPicker()
				return nil
			}
			src := m.sources[0]
			slot := m.focusSlot()
			m.bind(slot.id, src.GetId())
			return []request{m.attachRequest(src.GetId())}
		}
	}

	present := map[string]bool{}
	for _, src := range m.sources {
		present[src.GetId()] = true
	}
	for _, t := range m.tabs {
		for _, s := range t.slots {
			if s.sourceID != "" && !present[s.sourceID] {
				s.sourceID = ""
				s.scrollOffset = 0
			}
		}
	}
	return nil
}

// findAttachTarget matches the -attach request against the current sources by
// terminal id, full source id or <endpoint>:<id>.
func (m *model) findAttachTarget() *pb.Source {
	target := strings.TrimSpace(m.attachTarget)
	if target == "" {
		return nil
	}
	for _, src := range m.sources {
		endpoint, id, ok := parseSourceID(src.GetId())
		if !ok {
			continue
		}
		if target == id || target == src.GetId() || target == endpoint+":"+id {
			return src
		}
	}
	return nil
}

func (m *model) attachRequest(sourceID string) request {
	endpointName, id, _ := parseSourceID(sourceID)
	fit := true
	params := &pb.MethodParams{Endpoint: endpointName, Id: id, Fit: &fit}
	if endpoint, ok := m.endpointByName(endpointName); ok && endpoint.isDaemon() {
		params.Kind = endpoint.kind
		params.Socket = endpoint.socket
		params.Address = endpoint.address
		params.ConnectMode = endpoint.connectMode
	}
	return request{Method: "terminal.attach", Params: params}
}

// bind associates a slot with a source and returns to the terminal.
func (m *model) bind(slotID, sourceID string) {
	t, i := m.findSlot(slotID)
	if t == nil {
		return
	}
	t.slots[i].sourceID = sourceID
	t.slots[i].scrollOffset = 0
	m.mode = modeNormal
}

// key routes one normalized key through the mode state machine. The picker
// key is special-cased first: the second press inside the double-click
// window is forwarded into the focused terminal instead of toggling the
// picker (SCENARIOS §8.1). The picker key itself is configurable.
func (m *model) key(ev keyEvent) []request {
	if m.status != "" {
		m.status = ""
	}
	if ev.Key == m.binds.key(actionPickerOpen) {
		previous := m.lastCtrlF
		m.lastCtrlF = m.now()
		if m.mode == modePicker && !previous.IsZero() && m.lastCtrlF.Sub(previous) <= m.forwardWindow {
			m.mode = modeNormal
			return m.forward(ev.ID)
		}
	}
	switch m.mode {
	case modePicker:
		return m.keyPicker(ev)
	case modePrompt:
		return m.keyPrompt(ev)
	case modeHelp:
		return m.keyHelp(ev)
	case modeScroll:
		return m.keyScroll(ev)
	case modePane:
		return m.keyPane(ev)
	default:
		return m.keyNormal(ev)
	}
}

func (m *model) keyNormal(ev keyEvent) []request {
	switch m.binds.action(ev.Key) {
	case actionPaneMode:
		m.mode = modePane
	case actionPickerOpen:
		m.openPicker()
	case actionTabNew:
		m.addTab(true)
	case actionSidebarToggle:
		m.sidebar = !m.sidebar
	case actionTerminalRestart:
		return m.restartFocused()
	}
	switch ev.Key {
	case "page-up":
		return m.scrollFocused(10)
	case "page-down":
		return m.scrollFocused(-10)
	}
	return nil
}

func (m *model) keyPane(ev keyEvent) []request {
	switch m.binds.action(ev.Key) {
	case actionPaneMode:
		m.mode = modeNormal
	case actionPickerOpen:
		m.openPicker()
	case actionTabNew:
		m.addTab(true)
	case actionSidebarToggle:
		m.sidebar = !m.sidebar
	case actionTerminalRestart:
		return m.restartFocused()
	case actionSlotClose:
		return m.closeSlot()
	case actionPaneSplitH:
		m.split("row")
	case actionPaneSplitV:
		m.split("col")
	case actionPromptOpen:
		m.mode = modePrompt
		m.promptText = ""
		m.promptIdx = 0
	case actionHelpOpen:
		m.mode = modeHelp
	}
	switch ev.Key {
	case "esc":
		m.mode = modeNormal
	case "tab":
		t := m.activeTab()
		t.focus = (t.focus + 1) % len(t.slots)
	case "page-up":
		return m.scrollFocused(10)
	case "page-down":
		return m.scrollFocused(-10)
	default:
		if len(ev.Key) == 1 && ev.Key[0] >= '1' && ev.Key[0] <= '9' {
			m.selectTab(int(ev.Key[0] - '1'))
		}
	}
	return nil
}

func (m *model) keyScroll(ev keyEvent) []request {
	switch m.binds.action(ev.Key) {
	case actionScrollCopy:
		return m.copyFocused()
	}
	switch ev.Key {
	case "esc":
		return m.scrollEnd()
	case "page-up", "up":
		return m.scrollFocused(10)
	case "page-down", "down":
		return m.scrollFocused(-10)
	}
	return nil
}

func (m *model) keyHelp(ev keyEvent) []request {
	if ev.Key == "esc" || ev.Key == "?" {
		m.mode = modeNormal
	}
	return nil
}

func (m *model) keyPrompt(ev keyEvent) []request {
	switch ev.Key {
	case "esc":
		m.mode = modeNormal
	case "backspace":
		if m.promptText != "" {
			_, size := lastRune(m.promptText)
			m.promptText = m.promptText[:len(m.promptText)-size]
		}
	case "enter":
		cmd := strings.TrimSpace(m.promptText)
		m.mode = modeNormal
		return m.runCommand(cmd)
	case "up", "down":
		matches := m.commandMatches()
		if len(matches) > 0 {
			if ev.Key == "up" {
				m.promptIdx = clampIndex(m.promptIdx-1, len(matches))
			} else {
				m.promptIdx = clampIndex(m.promptIdx+1, len(matches))
			}
			m.promptText = matches[m.promptIdx]
		}
	default:
		if text := printable(ev); text != "" {
			m.promptText += text
			m.promptIdx = 0
		}
	}
	return nil
}

func (m *model) keyPicker(ev keyEvent) []request {
	items := m.pickerItems()
	switch ev.Key {
	case "up":
		m.pickerIdx = clampIndex(m.pickerIdx-1, len(items))
	case "down":
		m.pickerIdx = clampIndex(m.pickerIdx+1, len(items))
	case "enter":
		return m.pick(items)
	case "esc", "ctrl-f":
		m.mode = modeNormal
	}
	return nil
}

func (m *model) openPicker() {
	m.mode = modePicker
	m.pickerIdx = 0
}

func (m *model) pickerItems() []pickerItem {
	items := make([]pickerItem, 0, len(m.sources)+len(m.endpoints)+1)
	for _, src := range m.sources {
		label := src.GetTitle()
		if label == "" {
			label = src.GetId()
		}
		endpoint := src.GetEndpoint()
		if endpoint == "" {
			endpoint = "local"
		}
		state := "live"
		switch {
		case src.GetExited():
			state = "exited"
		case src.GetHealth() == "offline":
			state = "offline"
		case src.GetHealth() == "connecting":
			state = "connecting"
		}
		items = append(items, pickerItem{
			label:  label,
			info:   endpoint + " · " + state,
			source: src,
		})
	}
	// Endpoint placeholder sources (paired backend with no terminal yet)
	// show the endpoint itself; selecting one creates a terminal there, and an
	// offline backend surfaces its readable error in the status.
	for _, src := range m.endpointSources {
		state := "live"
		switch src.GetHealth() {
		case "offline":
			state = "offline"
		case "connecting":
			state = "connecting"
		}
		items = append(items, pickerItem{
			label:  m.iconJoin(iconEndpoint, src.GetTitle()),
			info:   "endpoint · " + state,
			source: src,
		})
	}
	for i := range m.endpoints {
		endpoint := &m.endpoints[i]
		items = append(items, pickerItem{
			label:    m.iconJoin(iconEndpoint, endpoint.displayName()),
			info:     "endpoint · " + endpoint.kindInfo(),
			endpoint: endpoint,
		})
	}
	items = append(items, pickerItem{label: m.iconJoin(iconTabNew, "+ New terminal"), info: "create"})
	return items
}

// kindInfo is the short picker column of a configured endpoint.
func (e endpointSpec) kindInfo() string {
	if e.isDaemon() {
		if e.connectMode == "" {
			return "daemon"
		}
		return "daemon " + e.connectMode
	}
	if len(e.argv) == 0 {
		return "command"
	}
	return "command " + e.argv[0]
}

// endpointByName finds a configured endpoint.
func (m *model) endpointByName(name string) (endpointSpec, bool) {
	for i := range m.endpoints {
		if m.endpoints[i].name == name {
			return m.endpoints[i], true
		}
	}
	return endpointSpec{}, false
}

// endpointCreateParams builds terminal.create params for a configured
// endpoint. Daemon endpoints carry kind/socket/connect_mode; an empty argv
// asks the daemon for its default command.
func endpointCreateParams(endpoint endpointSpec) *pb.MethodParams {
	params := &pb.MethodParams{
		Endpoint: endpoint.name,
		Argv:     append([]string(nil), endpoint.argv...),
		Cwd:      endpoint.cwd,
		Env:      cloneEnv(endpoint.env),
	}
	if endpoint.isDaemon() {
		params.Kind = endpoint.kind
		params.Socket = endpoint.socket
		params.Address = endpoint.address
		params.ConnectMode = endpoint.connectMode
	}
	return params
}

// syncEndpoints emits one endpoint.sync per daemon endpoint so the host
// connects and publishes the daemon terminal inventory as sources.
func (m *model) syncEndpoints() []request {
	var out []request
	for i := range m.endpoints {
		endpoint := m.endpoints[i]
		if !endpoint.isDaemon() {
			continue
		}
		out = append(out, request{
			Method: "endpoint.sync",
			Params: &pb.MethodParams{
				Endpoint:    endpoint.name,
				Kind:        endpoint.kind,
				Socket:      endpoint.socket,
				Address:     endpoint.address,
				ConnectMode: endpoint.connectMode,
			},
		})
	}
	return out
}

func (m *model) pick(items []pickerItem) []request {
	if len(items) == 0 {
		return nil
	}
	item := items[clampIndex(m.pickerIdx, len(items))]
	slot := m.focusSlot()
	slotID := slot.id
	if item.endpoint != nil {
		endpoint := *item.endpoint
		m.mode = modeNormal
		m.status = "creating " + endpoint.name
		return []request{{
			Method: "terminal.create",
			Params: endpointCreateParams(endpoint),
			After: func(resp *pb.Response) {
				if !resp.GetOk() {
					m.status = "create failed: " + resp.GetError()
					return
				}
				sourceID := terminalSourceID(resp.GetData().GetEndpoint(), resp.GetData().GetId())
				m.bind(slotID, sourceID)
				m.status = "bound " + endpoint.name + ":" + resp.GetData().GetId()
			},
		}}
	}
	if item.source == nil {
		m.mode = modeNormal
		m.status = "creating terminal"
		endpoint := "local"
		return []request{{
			Method: "terminal.create",
			Params: &pb.MethodParams{Endpoint: endpoint, Cwd: m.startupCwd},
			After: func(resp *pb.Response) {
				if !resp.GetOk() {
					m.status = "create failed: " + resp.GetError()
					return
				}
				sourceID := terminalSourceID(resp.GetData().GetEndpoint(), resp.GetData().GetId())
				m.bind(slotID, sourceID)
				m.status = "bound " + resp.GetData().GetId()
			},
		}}
	}
	if item.source != nil && item.source.GetKind() == "endpoint" {
		m.mode = modeNormal
		endpoint := item.source.GetEndpoint()
		m.status = "creating " + endpoint
		return []request{{
			Method: "terminal.create",
			Params: &pb.MethodParams{Endpoint: endpoint},
			After: func(resp *pb.Response) {
				if !resp.GetOk() {
					m.status = "create failed: " + resp.GetError()
					return
				}
				sourceID := terminalSourceID(resp.GetData().GetEndpoint(), resp.GetData().GetId())
				m.bind(slotID, sourceID)
				m.status = "bound " + endpoint + ":" + resp.GetData().GetId()
			},
		}}
	}
	m.mode = modeNormal
	m.status = "binding " + shortSourceID(item.source.GetId())
	req := m.attachRequest(item.source.GetId())
	base := req.After
	req.After = func(resp *pb.Response) {
		if base != nil {
			base(resp)
		}
		if !resp.GetOk() {
			m.status = "attach failed: " + resp.GetError()
			return
		}
		m.bind(slotID, item.source.GetId())
		m.status = "bound " + shortSourceID(item.source.GetId())
	}
	return []request{req}
}

func (m *model) forward(eventID string) []request {
	slot := m.focusSlot()
	if slot.sourceID == "" {
		m.status = "no terminal to forward to"
		return nil
	}
	if eventID == "" {
		return nil
	}
	return []request{{
		Method: "input.forward",
		Params: &pb.MethodParams{EventId: eventID, Source: slot.sourceID},
		After: func(resp *pb.Response) {
			if !resp.GetOk() {
				m.status = "forward failed: " + resp.GetError()
			}
		},
	}}
}

func (m *model) restartFocused() []request {
	return m.restartSlot(m.focusSlot())
}

// restartSlot asks the host to rebuild the terminal of one slot under the
// same id. The slot shows a pending indicator until the response arrives; a
// slot without a terminal is a no-op plus a status toast.
func (m *model) restartSlot(s *slot) []request {
	endpoint, id, ok := parseSourceID(s.sourceID)
	if !ok {
		m.status = "no terminal to restart"
		return nil
	}
	slotID := s.id
	s.pending = slotButtonRestart
	m.status = "restarting " + id
	return []request{{
		Method: "terminal.restart",
		Params: &pb.MethodParams{Endpoint: endpoint, Id: id},
		After: func(resp *pb.Response) {
			if t, i := m.findSlot(slotID); t != nil {
				t.slots[i].pending = ""
			}
			if !resp.GetOk() {
				m.status = "restart failed: " + resp.GetError()
			}
		},
	}}
}

// killFocused asks the host to kill the focused terminal. The host upgrades
// terminal.kill to a core confirmation (SCENARIOS §5, MAP fig. 7); the slot
// keeps its binding and shows the exit badge afterwards.
func (m *model) killFocused() []request {
	slot := m.focusSlot()
	endpoint, id, ok := parseSourceID(slot.sourceID)
	if !ok {
		m.status = "no terminal to kill"
		return nil
	}
	m.status = "killing " + id
	return []request{{
		Method: "terminal.kill",
		Params: &pb.MethodParams{Endpoint: endpoint, Id: id},
		After: func(resp *pb.Response) {
			if !resp.GetOk() {
				m.status = "kill failed: " + resp.GetError()
				return
			}
			m.status = "killed " + id
		},
	}}
}

func (m *model) scrollFocused(delta int) []request {
	return m.scrollSlot(m.focusSlot(), delta)
}

func (m *model) scrollSlot(slot *slot, delta int) []request {
	if delta == 0 {
		return nil
	}
	endpoint, id, ok := parseSourceID(slot.sourceID)
	if !ok {
		return nil
	}
	slotID := slot.id
	req := request{
		Method: "terminal.scroll",
		Params: &pb.MethodParams{Endpoint: endpoint, Id: id, Delta: int32(delta)},
		After: func(resp *pb.Response) {
			if !resp.GetOk() {
				m.status = "scroll failed: " + resp.GetError()
				return
			}
			t, i := m.findSlot(slotID)
			if t == nil {
				return
			}
			t.slots[i].scrollOffset += delta
			if t.slots[i].scrollOffset < 0 {
				t.slots[i].scrollOffset = 0
			}
			m.mode = modeScroll
			if t.slots[i].scrollOffset == 0 {
				m.mode = modeNormal
			}
		},
	}
	return []request{req}
}

func (m *model) scrollEnd() []request {
	slot := m.focusSlot()
	slot.scrollOffset = 0
	m.mode = modeNormal
	endpoint, id, ok := parseSourceID(slot.sourceID)
	if !ok {
		return nil
	}
	return []request{{
		Method: "terminal.scrollEnd",
		Params: &pb.MethodParams{Endpoint: endpoint, Id: id},
	}}
}

func (m *model) copyFocused() []request {
	endpoint, id, ok := parseSourceID(m.focusSlot().sourceID)
	if !ok {
		m.status = "no terminal to copy"
		return nil
	}
	return []request{{
		Method: "terminal.copy",
		Params: &pb.MethodParams{Endpoint: endpoint, Id: id},
		After: func(resp *pb.Response) {
			if resp.GetOk() {
				m.status = "copied visible screen"
			} else {
				m.status = "copy failed: " + resp.GetError()
			}
		},
	}}
}

func (m *model) runCommand(cmd string) []request {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return nil
	}
	switch fields[0] {
	case "help", "h", "?":
		m.mode = modeHelp
	case "split", "split-h", "hsplit":
		if len(fields) > 1 && (fields[1] == "v" || fields[1] == "vertical") {
			m.split("col")
		} else {
			m.split("row")
		}
	case "splitv", "vsplit":
		m.split("col")
	case "close", "x":
		return m.closeSlot()
	case "tab", "new-tab":
		m.addTab(true)
	case "restart":
		return m.restartFocused()
	case "kill":
		return m.killFocused()
	case "copy", "y":
		return m.copyFocused()
	case "live", "scroll-end":
		return m.scrollEnd()
	case "quit", "q", "exit":
		cleanup := false
		return []request{{
			Method: "system.quit",
			Params: &pb.MethodParams{CleanupOwned: &cleanup},
		}}
	default:
		m.status = "unknown command: " + fields[0]
	}
	return nil
}

// split adds an empty slot next to the focused one and switches the tab to
// the requested direction (SCENARIOS §2).
func (m *model) split(dir string) []request {
	t := m.activeTab()
	t.flow = dir
	focus := t.focus
	ratio := t.slots[focus].ratio
	next := &slot{id: m.nextID("slot"), ratio: ratio}
	t.slots = append(t.slots, nil)
	copy(t.slots[focus+2:], t.slots[focus+1:])
	t.slots[focus+1] = next
	t.focus = focus + 1
	return nil
}

// closeSlot is the anytty-style close: the slot goes away (or unbinds when it
// is the last one), the terminal keeps running in the daemon.
func (m *model) closeSlot() []request {
	t := m.activeTab()
	return m.closeSlotAt(t, t.focus)
}

// closeSlotAt closes one slot of a tab (title-bar ✕ and `x` share this).
func (m *model) closeSlotAt(t *tab, index int) []request {
	if t == nil || index < 0 || index >= len(t.slots) {
		return nil
	}
	slot := t.slots[index]
	var reqs []request
	if slot.scrollOffset > 0 {
		if endpoint, id, ok := parseSourceID(slot.sourceID); ok {
			reqs = append(reqs, request{
				Method: "terminal.scrollEnd",
				Params: &pb.MethodParams{Endpoint: endpoint, Id: id},
			})
		}
	}
	if len(t.slots) > 1 {
		t.slots = append(t.slots[:index], t.slots[index+1:]...)
		if t.focus >= len(t.slots) {
			t.focus = len(t.slots) - 1
		}
	} else {
		slot.sourceID = ""
		slot.scrollOffset = 0
	}
	return reqs
}

func (m *model) selectTab(index int) {
	if index < 0 || index >= len(m.tabs) || index == m.active {
		return
	}
	m.active = index
	m.activeTab().focus = clampIndex(m.activeTab().focus, len(m.activeTab().slots))
}

func (m *model) addTab(pickOnEmpty bool) {
	m.tabs = append(m.tabs, m.newTab(strconv.Itoa(len(m.tabs)+1)))
	m.active = len(m.tabs) - 1
	if pickOnEmpty {
		m.openPicker()
	}
}

// wheel routes one wheel notch to the slot under the pointer, else to the
// focused slot (SCENARIOS §4). Positive delta scrolls back into history.
func (m *model) wheel(node string, delta int) []request {
	if delta == 0 || m.mode == modePicker || m.mode == modePrompt || m.mode == modeHelp {
		return nil
	}
	slot := m.focusSlot()
	if t, i := m.findSlot(node); t != nil && t == m.activeTab() {
		slot = t.slots[i]
	}
	if slot.sourceID == "" {
		return nil
	}
	return m.scrollSlot(slot, delta*3)
}

// mouse handles clicks on the tab bar, sidebar rows, slots and dividers plus
// the divider drag sequence (SCENARIOS §9). A picker row is a one-press bind:
// the click selects the row and runs the same pick as enter.
func (m *model) mouse(action, button, node string, x, y int) []request {
	if strings.HasPrefix(node, "pick:") {
		if action == "press" && m.mode == modePicker {
			m.pickerIdx = clampIndex(parseNodeIndex(node, "pick:"), len(m.pickerItems()))
			return m.pick(m.pickerItems())
		}
		return nil
	}
	switch m.mode {
	case modePicker, modePrompt, modeHelp:
		return nil
	}
	switch {
	case node == "tab:new":
		if action == "press" {
			m.addTab(true)
		}
	case strings.HasPrefix(node, "tab:"):
		if action == "press" {
			m.selectTab(parseNodeIndex(node, "tab:"))
		}
	case strings.HasPrefix(node, "side:tab:"):
		if action == "press" {
			m.selectTab(parseNodeIndex(node, "side:tab:"))
		}
	case strings.HasPrefix(node, "btn:"):
		return m.buttonMouse(action, node)
	case strings.HasPrefix(node, "title:"):
		if action == "press" {
			m.focusSlotByID(strings.TrimPrefix(node, "title:"))
		}
	case strings.HasPrefix(node, "divider:"):
		return m.dividerMouse(action, node, x, y)
	case strings.HasPrefix(node, "slot-"):
		if action == "press" {
			t, i := m.findSlot(node)
			if t != nil {
				m.active = indexOfTab(m.tabs, t)
				t.focus = i
				if t.slots[i].sourceID == "" && m.mode == modeNormal {
					m.mode = modePane
				}
			}
		}
	}
	return nil
}

// buttonMouse routes one mouse action on a slot title-bar button: the press
// marks the button pressed (highlighted until the next redraw), focuses its
// slot and runs the matching program action; the release clears the mark.
func (m *model) buttonMouse(action, node string) []request {
	slotID, button, ok := parseButtonNode(node)
	if !ok {
		return nil
	}
	t, i := m.findSlot(slotID)
	if t == nil {
		return nil
	}
	if action == "release" {
		if m.pressed == node {
			m.pressed = ""
		}
		return nil
	}
	if action != "press" {
		return nil
	}
	m.active = indexOfTab(m.tabs, t)
	t.focus = i
	m.pressed = node
	return m.slotAction(t.slots[i], button)
}

// slotAction runs one title-bar button against its slot. split-h/split-v are
// exactly the `%`/`"` program actions; close is the unbind-style close of `x`.
func (m *model) slotAction(s *slot, button string) []request {
	switch button {
	case slotButtonRestart:
		return m.restartSlot(s)
	case slotButtonSplitH:
		m.split("row")
	case slotButtonSplitV:
		m.split("col")
	case slotButtonClose:
		t, i := m.findSlot(s.id)
		if t == nil {
			return nil
		}
		return m.closeSlotAt(t, i)
	}
	return nil
}

// focusSlotByID focuses a slot by id (title-bar label click), switching to
// its tab first when needed.
func (m *model) focusSlotByID(id string) {
	t, i := m.findSlot(id)
	if t == nil {
		return
	}
	m.active = indexOfTab(m.tabs, t)
	t.focus = i
}

func (m *model) dividerMouse(action, node string, x, y int) []request {
	index := parseNodeIndex(node, "divider:")
	t := m.activeTab()
	if index < 0 || index >= len(t.slots)-1 {
		return nil
	}
	switch action {
	case "press":
		bodyWidth, bodyHeight := m.bodySize()
		ratios := make([]int, len(t.slots))
		for i, s := range t.slots {
			ratios[i] = s.ratio
		}
		avail := bodyWidth - (len(t.slots) - 1)
		if t.flow == "col" {
			avail = bodyHeight - (len(t.slots) - 1)
		}
		if avail < 1 {
			avail = 1
		}
		m.drag = &dragState{divider: index, startX: x, startY: y, ratios: ratios, axis: t.flow, avail: avail}
	case "drag":
		if m.drag == nil || m.drag.divider != index {
			return nil
		}
		total := 0
		for _, r := range m.drag.ratios {
			total += r
		}
		delta := x - m.drag.startX
		if m.drag.axis == "col" {
			delta = y - m.drag.startY
		}
		shift := delta * total / m.drag.avail
		left := m.drag.ratios[index] + shift
		right := m.drag.ratios[index+1] - shift
		if left < minRatio {
			right -= minRatio - left
			left = minRatio
		}
		if right < minRatio {
			left -= minRatio - right
			right = minRatio
		}
		t.slots[index].ratio = left
		t.slots[index+1].ratio = right
	case "release":
		m.drag = nil
	}
	return nil
}

func indexOfTab(tabs []*tab, want *tab) int {
	for i, t := range tabs {
		if t == want {
			return i
		}
	}
	return 0
}

func parseNodeIndex(node, prefix string) int {
	value := strings.TrimPrefix(node, prefix)
	n, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}
	return n
}

func clampIndex(index, length int) int {
	if length <= 0 {
		return 0
	}
	if index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

func lastRune(text string) (rune, int) {
	if text == "" {
		return 0, 0
	}
	return utf8.DecodeLastRuneInString(text)
}

// printable reports the text one key event contributes to the prompt.
func printable(ev keyEvent) string {
	if ev.Key == "" || strings.HasPrefix(ev.Key, "ctrl-") {
		return ""
	}
	if utf8.RuneCountInString(ev.Key) != 1 || ev.Key < " " || ev.Key == "\x7f" {
		return ""
	}
	if ev.Char != "" && utf8.RuneCountInString(ev.Char) == 1 {
		return ev.Char
	}
	return ev.Key
}

// bodySize is the cell budget of the pane area (header and footer excluded).
func (m *model) bodySize() (width, height int) {
	width = m.cols
	if m.sidebar {
		width -= sidebarWidth
	}
	height = m.rows - 2
	return maxInt(1, width), maxInt(1, height)
}

// distribute splits avail cells among ratios, preserving the exact sum.
// cloneEnv copies one endpoint env map so the request never aliases config
// state.
func cloneEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return nil
	}
	out := make(map[string]string, len(env))
	for key, value := range env {
		out[key] = value
	}
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

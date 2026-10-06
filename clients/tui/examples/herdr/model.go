package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/app"
	"github.com/anytty/anytty/clients/tui/sdk/widgets"
	apipb "github.com/anytty/anytty/proto/access/apipb"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// Layout constants. The sidebar width matches the SPEC (26 cells, prefix+b
// collapses it); the body is solved with widgets.SplitLayout so pane geometry
// and the divider hit boxes stay explicit.
const (
	sidebarWidth = 26

	// minPaneWeight / maxPaneWeight are the resize clamps from the SPEC: every
	// pane keeps 5%..95% of the tab's main axis. maxPanes follows from the
	// floor: 20 panes at 5% each.
	minPaneWeight = 5
	maxPaneWeight = 95
	maxPanes      = 100 / minPaneWeight

	// defaultWorkspaceName / defaultTabName name the auto-created first
	// workspace/tab. Added tabs are named "shell".
	defaultWorkspaceName = "main"
	defaultTabName       = "main"
	addedTabName         = "shell"

	// storage identity (SPEC §7): AppId "herdr", PRIVATE scope.
	herdrAppID   = "herdr"
	layoutKey    = "layout"
	legacyPinKey = "pinned"
)

// Modes (SPEC §4). terminal is the default; prefix waits for one action;
// navigate selects a workspace; resize adjusts weights; scroll drives
// terminal.scroll/scrollEnd/terminal.copy.
const (
	modeTerminal = "terminal"
	modePrefix   = "prefix"
	modeNavigate = "navigate"
	modeResize   = "resize"
	modeScroll   = "scroll"
)

// Direct terminal-mode chords (SPEC §5). While a live terminal pane is
// focused these are the ONLY keys herdr claims; everything else reaches the
// PTY (plus the host-reserved ctrl-q).
var directChords = []string{
	"ctrl-b",
	"ctrl-alt-h", "ctrl-alt-j", "ctrl-alt-k", "ctrl-alt-l",
	"ctrl-alt-c", "ctrl-alt-d", "ctrl-alt-z",
}

// pane is one terminal slot of a tab. Source is the bound terminal source id
// ("terminal:<endpoint>:<id>"); empty means the empty state. Name is the user
// rename (empty falls back to the live source title).
type pane struct {
	ID     string
	Name   string
	Source string
}

// tab owns a flat pane list plus one split axis, the pane weights (percent,
// summing to 100) and the focused pane id.
type tab struct {
	ID      string
	Name    string
	Axis    string // "row" (left/right) or "col" (top/bottom)
	Weights []int
	Focus   string
	Panes   []*pane
}

// workspace groups tabs and remembers the active one.
type workspace struct {
	ID        string
	Name      string
	ActiveTab string
	Tabs      []*tab
}

// paneState is the derived lifecycle of a pane (SPEC §2): every value comes
// from the sources snapshot, never from a guessed process state.
type paneState int

const (
	stateEmpty paneState = iota
	stateRunning
	stateExited
	stateGone
	stateOffline
)

func (s paneState) String() string {
	switch s {
	case stateRunning:
		return "running"
	case stateExited:
		return "exited"
	case stateGone:
		return "gone"
	case stateOffline:
		return "offline"
	default:
		return "empty"
	}
}

// severity ranks states for the workspace rollup. The SPEC fixes
// offline > exited > running; "gone" (a bound source missing from the
// snapshot) is treated as an unavailable/pane-dead tier next to offline.
func (s paneState) severity() int {
	switch s {
	case stateOffline:
		return 4
	case stateGone:
		return 3
	case stateExited:
		return 2
	case stateRunning:
		return 1
	default:
		return 0
	}
}

// renameKind targets the rename modal.
type renameKind string

const (
	renameWorkspace renameKind = "workspace"
	renameTab       renameKind = "tab"
	renamePane      renameKind = "pane"
)

type renameState struct {
	kind  renameKind
	ref   string
	input widgets.TextInput
}

// dividerDrag tracks an in-flight split-divider drag. The host implicitly
// captures the divider box (non-terminal + input:["mouse"], PROTOCOL §6.7),
// so press/drag/release all come back with the same node id.
type dividerDrag struct {
	ref     string // tab ref "w1:t1"
	index   int    // divider index (between pane index and index+1)
	axis    string
	startX  int
	startY  int
	weights []int
	extent  int // cells of the two adjacent panes plus the gap
	moved   bool
}

// opMsg is the result of one terminal.*/system.quit method call. The emit
// helper keeps the whole RESPONSE (ok/error plus create data), which is more
// than app.Emit's access-only decode exposes.
type opMsg struct {
	op       string
	ref      string // pane ref for create/attach/scroll targets
	source   string // terminal source id
	delta    int
	follow   bool
	ok       bool
	err      string
	endpoint string
	id       string
}

// storageMsg is the decoded access.call storage result. op is "get"/"set".
type storageMsg struct {
	op    string
	key   string
	ok    bool
	err   string
	empty bool
	value string
}

// model is herdr's state machine: workspaces -> tabs -> panes plus the mode
// machine. It performs no protocol I/O itself; side effects are returned as
// app.Cmd values built on the SDK client, so tests drive it over in-memory
// pipes without a host.
type model struct {
	client *sdk.Client
	memo   *app.Memo

	hello  *pb.Hello
	viewID string
	cols   int
	rows   int

	workspaces      []*workspace
	activeWorkspace string

	// sources is the last full snapshot, keyed by source id (kind=="terminal"
	// only). Reconcile is idempotent and order independent.
	sources map[string]*pb.Source

	sidebarCollapsed bool
	spacesOffset     int
	agentsOffset     int

	mode  string
	navWS int
	help  bool

	menu     widgets.ContextMenu
	menuKind string
	menuRef  string
	rename   *renameState

	zoom        bool
	drag        *dividerDrag
	scrollLines int

	toast    string
	toastErr bool

	// persistence bookkeeping (SPEC §7).
	layoutReady       bool
	layoutDirty       bool
	layoutUserTouched bool
	savePending       bool
	pendingPin        string
	pendingAttach     map[string]bool

	// per-Update markers; Update composes the returned Cmd from them.
	dirty         bool
	keysChanged   bool
	layoutChanged bool
	// sentKeys is the claim actually delivered with the last SetKeys. The
	// declared claim depends on derived state (focused pane live vs empty,
	// program modes), so refresh it whenever the computed value differs —
	// a sources reconcile can flip the focused pane to live without any key
	// or mode transition (SPEC §5).
	sentKeys sdk.Keys
}

func newModel(client *sdk.Client, memo *app.Memo) *model {
	return &model{
		client:          client,
		memo:            memo,
		cols:            80,
		rows:            24,
		mode:            modeTerminal,
		sources:         make(map[string]*pb.Source),
		pendingAttach:   make(map[string]bool),
		activeWorkspace: "",
	}
}

// defaultModel returns a model with the seed workspace/tab/pane, used by
// tests and by the startup path when no persisted layout exists.
func defaultModel(client *sdk.Client, memo *app.Memo) *model {
	m := newModel(client, memo)
	m.seedLayout()
	return m
}

// seedLayout installs the auto-created workspace: one tab, one empty pane.
func (m *model) seedLayout() {
	ws := &workspace{ID: "w1", Name: defaultWorkspaceName, ActiveTab: "t1"}
	ws.Tabs = []*tab{{
		ID:      "t1",
		Name:    defaultTabName,
		Axis:    "row",
		Weights: []int{100},
		Focus:   "p1",
		Panes:   []*pane{{ID: "p1"}},
	}}
	m.workspaces = []*workspace{ws}
	m.activeWorkspace = ws.ID
}

// Init runs on every HELLO epoch: restore the persisted layout through
// access.call. A missing pool/access is a toast, not a crash. On a host
// restart (new epoch) the in-memory layout is already authoritative and is
// NOT re-read, so an in-flight save cannot be clobbered by an older value.
func (m *model) Init() app.Cmd {
	m.dirty = true
	if len(m.workspaces) > 0 && m.layoutReady {
		return app.None
	}
	if len(m.workspaces) == 0 {
		m.seedLayout()
	}
	return m.storageGetCmd(layoutKey)
}

// Reset drops connection-scoped state on a new HELLO epoch. The layout and
// the pane bindings stay: the next sources snapshot reconciles them by id.
func (m *model) Reset(epoch uint64) {
	m.hello = nil
	m.viewID = ""
	m.help = false
	m.menu.Close()
	m.rename = nil
	m.zoom = false
	m.drag = nil
	m.mode = modeTerminal
	m.scrollLines = 0
	m.toast = ""
	m.pendingAttach = make(map[string]bool)
	m.dirty = true
}

// Dirty lets Program suppress a commit for a batch that changed nothing.
func (m *model) Dirty() bool { return m.dirty }

// Update handles one message. dirty is deliberately NOT reset here: Run
// coalesces a burst of messages into one batch and checks Dirty() once at the
// end, so resetting per message would let a later no-op message in the same
// batch suppress the commit for an earlier real change. View (called once per
// committed batch) clears it instead.
func (m *model) Update(msg app.Msg) app.Cmd {
	m.keysChanged = false
	m.layoutChanged = false

	var cmd app.Cmd
	switch v := msg.(type) {
	case app.HelloMsg:
		m.hello = v.Hello
		m.viewID = v.Hello.GetViewId()
		if cols, rows := int(v.Hello.GetCols()), int(v.Hello.GetRows()); cols > 0 && rows > 0 {
			m.cols, m.rows = cols, rows
		}
		m.touch()
	case app.SourcesMsg:
		cmd = m.reconcile(v.Items)
	case app.ResizeMsg:
		m.cols, m.rows = v.Cols, v.Rows
		m.touch()
	case app.KeyMsg:
		cmd = m.onKey(v.Key.GetKey())
	case app.PasteMsg:
		cmd = m.onPaste(v.Paste)
	case app.MouseMsg:
		cmd = m.onMouse(v.Mouse)
	case app.WheelMsg:
		cmd = m.onWheel(v.Wheel)
	case opMsg:
		cmd = m.onOp(v)
	case storageMsg:
		cmd = m.onStorage(v)
	case app.NoticeMsg:
		// The endpoint manager repeats the same offline notice on every
		// backoff; ignore an identical repeat so it cannot wipe a toast.
		text := v.Level + ": " + v.Message
		if text == m.toast {
			return app.None
		}
		m.setToast(text, v.Level == "error" || v.Level == "warn")
	case app.ViewRejectedMsg:
		m.setToast("view rejected: "+v.Reason, true)
	case app.ErrorMsg:
		m.setToast("transport: "+v.Err.Error(), true)
	}

	if m.layoutChanged {
		cmd = chain(cmd, m.saveLayoutCmd())
	}
	want := m.keys()
	if m.keysChanged || !keysEqual(want, m.sentKeys) {
		m.keysChanged = false
		m.sentKeys = want
		cmd = chain(cmd, app.SetKeys(want))
	}
	return cmd
}

// keysEqual compares two routing declarations element-wise.
func keysEqual(a, b sdk.Keys) bool {
	if a.All != b.All || len(a.Claim) != len(b.Claim) {
		return false
	}
	for i := range a.Claim {
		if a.Claim[i] != b.Claim[i] {
			return false
		}
	}
	return true
}

// chain flattens nil commands and returns the single command as-is, so a
// one-effect Update stays directly invocable (tests, and one less batch).
func chain(cmds ...app.Cmd) app.Cmd {
	var alive []app.Cmd
	for _, cmd := range cmds {
		if cmd != nil {
			alive = append(alive, cmd)
		}
	}
	switch len(alive) {
	case 0:
		return nil
	case 1:
		return alive[0]
	default:
		return app.Batch(alive...)
	}
}

// --- mutations markers ---

func (m *model) touch() {
	m.dirty = true
}

// changed marks a state change that must be re-committed with a new key
// claim (mode transitions).
func (m *model) changed() {
	m.dirty = true
	m.keysChanged = true
}

// structural marks a state change that must be persisted (SPEC §7). A change
// made before the initial layout load completes wins over the stored value.
func (m *model) structural() {
	m.dirty = true
	m.layoutChanged = true
	m.layoutDirty = true
	if !m.layoutReady {
		m.layoutUserTouched = true
	}
}

// --- lookups ---

func (m *model) currentWS() *workspace {
	if ws := m.workspaceByID(m.activeWorkspace); ws != nil {
		return ws
	}
	if len(m.workspaces) > 0 {
		return m.workspaces[0]
	}
	return nil
}

func (m *model) workspaceByID(id string) *workspace {
	for _, ws := range m.workspaces {
		if ws.ID == id {
			return ws
		}
	}
	return nil
}

func (m *model) currentTab() *tab {
	ws := m.currentWS()
	if ws == nil {
		return nil
	}
	if t := tabByID(ws, ws.ActiveTab); t != nil {
		return t
	}
	if len(ws.Tabs) > 0 {
		return ws.Tabs[0]
	}
	return nil
}

func tabByID(ws *workspace, id string) *tab {
	if ws == nil {
		return nil
	}
	for _, t := range ws.Tabs {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func paneByID(t *tab, id string) *pane {
	if t == nil {
		return nil
	}
	for _, p := range t.Panes {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// focusedPane returns the current workspace's active tab and its focused
// pane.
func (m *model) focusedPane() (*workspace, *tab, *pane) {
	ws := m.currentWS()
	t := m.currentTab()
	if ws == nil || t == nil {
		return ws, t, nil
	}
	p := paneByID(t, t.Focus)
	if p == nil && len(t.Panes) > 0 {
		p = t.Panes[0]
	}
	return ws, t, p
}

// findPane resolves a "w1:t1:p1" pane ref.
func (m *model) findPane(ref string) (*workspace, *tab, *pane) {
	parts := strings.Split(ref, ":")
	if len(parts) != 3 {
		return nil, nil, nil
	}
	ws := m.workspaceByID(parts[0])
	if ws == nil {
		return nil, nil, nil
	}
	t := tabByID(ws, parts[1])
	if t == nil {
		return nil, nil, nil
	}
	return ws, t, paneByID(t, parts[2])
}

func paneRef(ws, tab, pane string) string { return ws + ":" + tab + ":" + pane }

func (m *model) refOf(ws *workspace, t *tab, p *pane) string {
	if ws == nil || t == nil || p == nil {
		return ""
	}
	return paneRef(ws.ID, t.ID, p.ID)
}

func tabRef(ws *workspace, t *tab) string {
	if ws == nil || t == nil {
		return ""
	}
	return ws.ID + ":" + t.ID
}

// focusIndex returns the focused pane index in t (-1 when absent).
func focusIndex(t *tab) int {
	if t == nil {
		return -1
	}
	for i, p := range t.Panes {
		if p.ID == t.Focus {
			return i
		}
	}
	return -1
}

// --- sources / state (SPEC §2) ---

// source returns the last snapshot row for a pane's binding, or nil.
func (m *model) source(p *pane) *pb.Source {
	if p == nil || p.Source == "" {
		return nil
	}
	return m.sources[p.Source]
}

// paneState derives the lifecycle from the snapshot only.
func (m *model) paneState(p *pane) paneState {
	if p == nil || p.Source == "" {
		return stateEmpty
	}
	src := m.source(p)
	if src == nil {
		return stateGone
	}
	if health := src.GetHealth(); health != "" && health != "ok" {
		return stateOffline
	}
	if src.GetExited() {
		return stateExited
	}
	return stateRunning
}

// stateText is the human label used in titles/status.
func (m *model) stateText(p *pane) string {
	src := m.source(p)
	switch state := m.paneState(p); state {
	case stateExited:
		return fmt.Sprintf("exited(%d)", src.GetExitCode())
	default:
		return state.String()
	}
}

// badges renders the attachment markers from the snapshot (SPEC §2): you /
// other / attached plus exit/health annotations.
func (m *model) badges(p *pane) []string {
	src := m.source(p)
	if src == nil {
		if p != nil && p.Source != "" {
			return []string{"[gone]"}
		}
		return nil
	}
	var out []string
	// The plain [attached] badge is redundant next to exited/offline
	// annotations and does not fit the 26-cell sidebar; you/other always show.
	switch {
	case src.GetAttached() && m.viewID != "" && src.GetResizeOwner() == m.viewID:
		out = append(out, "[you]")
	case src.GetAttached() && src.GetResizeOwner() != "":
		out = append(out, "[other]")
	case src.GetAttached() && m.paneState(p) == stateRunning:
		out = append(out, "[attached]")
	}
	if src.GetExited() {
		out = append(out, fmt.Sprintf("[exit %d]", src.GetExitCode()))
	}
	if health := src.GetHealth(); health != "" && health != "ok" {
		out = append(out, "["+health+"]")
	}
	return out
}

// displaySourceName is the title fallback chain: source title, terminal id,
// source id.
func displaySourceName(src *pb.Source) string {
	if src == nil {
		return ""
	}
	if title := src.GetTitle(); title != "" {
		return title
	}
	if id := src.GetTerminalId(); id != "" {
		return id
	}
	return src.GetId()
}

// paneTitle is the visible pane label: user rename, else the live source
// title, else "pane N". Control runes are stripped so a host-supplied title
// cannot break the row layout.
func (m *model) paneTitle(index int, p *pane) string {
	if p != nil && p.Name != "" {
		return widgetSafe(p.Name)
	}
	if src := m.source(p); src != nil {
		if name := displaySourceName(src); name != "" {
			return widgetSafe(name)
		}
	}
	return fmt.Sprintf("pane %d", index+1)
}

// rollup returns a workspace's worst pane state and the label to show.
func (m *model) rollup(ws *workspace) paneState {
	worst := stateEmpty
	for _, t := range ws.Tabs {
		for _, p := range t.Panes {
			if s := m.paneState(p); s.severity() > worst.severity() {
				worst = s
			}
		}
	}
	return worst
}

// reconcile replaces the snapshot and re-binds pending state by source id.
// It is idempotent and never assumes event order.
func (m *model) reconcile(items []*pb.Source) app.Cmd {
	next := make(map[string]*pb.Source, len(items))
	for _, src := range items {
		if src.GetKind() == "terminal" {
			next[src.GetId()] = src
		}
	}
	m.sources = next
	m.touch()

	var cmds []app.Cmd
	// Legacy pinned hint: bind the pinned source to the first empty pane on
	// the first snapshot that carries it (SPEC §7).
	m.bindPendingPin()
	// Re-bind restored panes: a bound pane whose source exists but has no
	// local attachment yet gets one terminal.attach (pool terminals survive
	// detach; this is the re-attach half of SPEC §7).
	for _, ws := range m.workspaces {
		for _, t := range ws.Tabs {
			for _, p := range t.Panes {
				src := m.source(p)
				if src == nil || src.GetAttached() || m.pendingAttach[src.GetId()] {
					continue
				}
				m.pendingAttach[src.GetId()] = true
				cmds = append(cmds, m.attachCmd(ws, t, p, src))
			}
		}
	}
	return chain(cmds...)
}

// --- methods / results ---

// emit is an app.Emit sibling that keeps the whole RESPONSE: terminal
// methods report ok/error plus terminal.create's {endpoint,id} data, none of
// which travel in data.access_result (the only payload app.Emit decodes).
func (m *model) emit(method string, params *pb.MethodParams, base opMsg) app.Cmd {
	client := m.client
	if client == nil {
		base.err = "not connected"
		return func() app.Msg { return base }
	}
	return func() app.Msg {
		responses := make(chan *pb.Response, 1)
		if _, err := client.Emit(method, params, func(resp *pb.Response) { responses <- resp }); err != nil {
			base.err = err.Error()
			return base
		}
		resp := <-responses
		base.ok = resp.GetOk()
		base.err = resp.GetError()
		base.endpoint = resp.GetData().GetEndpoint()
		base.id = resp.GetData().GetId()
		return base
	}
}

// createCmd creates a terminal on the local endpoint for an empty/gone pane.
func (m *model) createCmd(ref string) app.Cmd {
	m.setToast("creating a terminal…", false)
	return m.emit("terminal.create", &pb.MethodParams{Endpoint: "local"}, opMsg{op: "create", ref: ref})
}

// attachCmd attaches an existing pool terminal to a pane (restore path). When
// another client owns the resize lease the call carries
// expected_owner_epoch as a CAS; onOwnerConflict in onOp retries as
// fit=false so the pane still mirrors the terminal.
func (m *model) attachCmd(ws *workspace, t *tab, p *pane, src *pb.Source) app.Cmd {
	return m.attachParamsCmd(m.refOf(ws, t, p), src, true)
}

func (m *model) attachParamsCmd(ref string, src *pb.Source, fit bool) app.Cmd {
	endpoint, id := sourceEndpointID(src)
	params := &pb.MethodParams{Endpoint: endpoint, Id: id, Fit: &fit}
	if fit && m.viewID != "" && src.GetAttached() && src.GetResizeOwner() != "" && src.GetResizeOwner() != m.viewID {
		if epoch := src.GetOwnerEpoch(); epoch > 0 {
			params.ExpectedOwnerEpoch = &epoch
		}
	}
	return m.emit("terminal.attach", params, opMsg{
		op: "attach", ref: ref, source: src.GetId(), follow: !fit,
	})
}

func isOwnerConflict(errText string) bool {
	return strings.Contains(strings.ToLower(errText), "owner")
}

// scrollCmd drives terminal.scroll (positive delta = older). The optimistic
// offset is tracked by scrollBy, not here.
func (m *model) scrollCmd(delta int) app.Cmd {
	_, _, p := m.focusedPane()
	src := m.source(p)
	if src == nil {
		m.setToast("nothing to scroll", true)
		return app.None
	}
	endpoint, id := sourceEndpointID(src)
	return m.emit("terminal.scroll", &pb.MethodParams{Endpoint: endpoint, Id: id, Delta: int32(delta)},
		opMsg{op: "scroll", ref: m.currentPaneRef(), delta: delta})
}

func (m *model) scrollEndCmd() app.Cmd {
	_, _, p := m.focusedPane()
	src := m.source(p)
	if src == nil {
		return app.None
	}
	endpoint, id := sourceEndpointID(src)
	return m.emit("terminal.scrollEnd", &pb.MethodParams{Endpoint: endpoint, Id: id}, opMsg{op: "scrollEnd"})
}

func (m *model) copyCmd() app.Cmd {
	_, _, p := m.focusedPane()
	src := m.source(p)
	if src == nil {
		m.setToast("nothing to copy", true)
		return app.None
	}
	endpoint, id := sourceEndpointID(src)
	m.setToast("copied visible screen", false)
	return m.emit("terminal.copy", &pb.MethodParams{Endpoint: endpoint, Id: id}, opMsg{op: "copy"})
}

func (m *model) quitCmd() app.Cmd {
	// Detach = exit the TUI: a layout program exiting on its own would be
	// restarted by the host, so emit system.quit and let the host confirm.
	return m.emit("system.quit", &pb.MethodParams{}, opMsg{op: "quit"})
}

func (m *model) onOp(v opMsg) app.Cmd {
	switch v.op {
	case "create":
		if !v.ok {
			m.setToast("create failed: "+v.err, true)
			return app.None
		}
		id := "terminal:" + v.endpoint + ":" + v.id
		if _, _, p := m.findPane(v.ref); p != nil {
			p.Source = id
			m.pendingAttach[id] = true
			m.structural()
		}
		m.setToast("created "+id, false)
	case "attach":
		delete(m.pendingAttach, v.source)
		if !v.ok {
			if !v.follow && isOwnerConflict(v.err) {
				if src := m.sources[v.source]; src != nil {
					m.setToast("resize owned by another client; following", false)
					return m.attachParamsCmd(v.ref, src, false)
				}
			}
			m.setToast("attach failed: "+v.err, true)
			return app.None
		}
		m.setToast("attached "+displaySourceName(m.sources[v.source]), false)
	case "scroll":
		if !v.ok {
			m.setToast("scroll failed: "+v.err, true)
		}
	case "copy":
		if !v.ok {
			m.setToast("copy failed: "+v.err, true)
		}
	case "scrollEnd":
		if !v.ok {
			m.setToast("scrollEnd failed: "+v.err, true)
		}
	case "quit":
		if !v.ok {
			m.setToast("quit refused: "+v.err, true)
		}
	}
	return app.None
}

// setToast shows one transient line. There is no timer: the next action or
// result replaces it, and the toast only occupies a bottom-right overlay
// while it is non-empty, so an empty toast reserves no row.
func (m *model) setToast(text string, isErr bool) {
	m.toast = text
	m.toastErr = isErr
	m.touch()
}

// --- key claim (SPEC §5) ---

// programMode reports whether herdr must own every key: an overlay is open or
// a non-terminal mode waits for input.
func (m *model) programMode() bool {
	return m.help || m.rename != nil || m.menu.Visible ||
		m.mode == modePrefix || m.mode == modeNavigate ||
		m.mode == modeResize || m.mode == modeScroll
}

// liveTerminal reports whether the focused pane can accept PTY input. Only
// then may herdr claim just the escape chords and leave the rest to the
// terminal (SPEC §5).
func (m *model) liveTerminal() bool {
	_, _, p := m.focusedPane()
	return m.paneState(p) == stateRunning
}

// keys is the routing declaration that travels with the next view. In
// terminal mode with a live terminal focused, only ctrl+b and the direct
// chords are claimed; otherwise herdr claims every key (and clears focused).
func (m *model) keys() sdk.Keys {
	if m.programMode() || !m.liveTerminal() {
		return sdk.Keys{All: true}
	}
	return sdk.Keys{Claim: append([]string(nil), directChords...)}
}

// --- helpers ---

func sourceEndpointID(src *pb.Source) (endpoint, id string) {
	endpoint, id = src.GetEndpoint(), src.GetTerminalId()
	if id == "" {
		if parts := strings.SplitN(src.GetId(), ":", 3); len(parts) == 3 {
			endpoint, id = parts[1], parts[2]
		}
	}
	if endpoint == "" {
		endpoint = "local"
	}
	if id == "" {
		id = src.GetId()
	}
	return endpoint, id
}

func clampIndex(index, count int) int {
	if index < 0 {
		return 0
	}
	if count <= 0 {
		return 0
	}
	if index >= count {
		return count - 1
	}
	return index
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

// nextNumericID returns the smallest unused "<prefix>N" for the given
// existing ids (IDs are assigned monotonically and never reused).
func nextNumericID(prefix string, existing []string) string {
	used := make(map[int]bool, len(existing))
	max := 0
	for _, id := range existing {
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(id, prefix))
		if err != nil {
			continue
		}
		used[n] = true
		if n > max {
			max = n
		}
	}
	for n := 1; ; n++ {
		if !used[n] {
			return prefix + strconv.Itoa(n)
		}
	}
}

// equalWeights distributes 100 percent over n panes (min 5 each).
func equalWeights(n int) []int {
	if n <= 0 {
		return nil
	}
	if n == 1 {
		return []int{100}
	}
	out := make([]int, n)
	base, rem := 100/n, 100%n
	for i := range out {
		out[i] = base
		if i < rem {
			out[i]++
		}
	}
	return out
}

// normalizeTo100 clamps and rescales pane weights so they sum to 100 and
// each stays within [5, 95].
func normalizeTo100(weights []int) []int {
	n := len(weights)
	if n == 0 {
		return nil
	}
	if n == 1 {
		return []int{100}
	}
	out := make([]int, n)
	total := 0
	for i, w := range weights {
		if w < minPaneWeight {
			w = minPaneWeight
		}
		if w > maxPaneWeight {
			w = maxPaneWeight
		}
		out[i] = w
		total += w
	}
	if total == 0 {
		return equalWeights(n)
	}
	sum := 0
	rem := make([]int, n)
	for i := range out {
		scaled := out[i] * 100
		out[i] = scaled / total
		rem[i] = scaled % total
		sum += out[i]
	}
	for sum < 100 {
		best := 0
		for i := 1; i < n; i++ {
			if rem[i] > rem[best] {
				best = i
			}
		}
		out[best]++
		rem[best] = -1
		sum++
	}
	for sum > 100 {
		best := -1
		for i := range out {
			if out[i] > minPaneWeight && (best == -1 || out[i] > out[best]) {
				best = i
			}
		}
		if best == -1 {
			break
		}
		out[best]--
		sum--
	}
	return out
}

// adjustWeight moves `delta` percent from weight j to weight i, clamping both
// at the 5%..95% bounds. It reports whether anything changed.
func adjustWeight(weights []int, i, j, delta int) bool {
	if i < 0 || j < 0 || i >= len(weights) || j >= len(weights) || i == j {
		return false
	}
	if delta > 0 {
		d := minInt(delta, minInt(maxPaneWeight-weights[i], weights[j]-minPaneWeight))
		if d <= 0 {
			return false
		}
		weights[i] += d
		weights[j] -= d
		return true
	}
	d := minInt(-delta, minInt(weights[i]-minPaneWeight, maxPaneWeight-weights[j]))
	if d <= 0 {
		return false
	}
	weights[i] -= d
	weights[j] += d
	return true
}

// storageKey builds the apipb storage key. Kept here so model and tests share
// the exact identity.
func storageKey(key string) *apipb.StorageKey {
	return &apipb.StorageKey{AppId: herdrAppID, Scope: apipb.StorageScope_STORAGE_SCOPE_PRIVATE, Key: key}
}

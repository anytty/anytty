package runtime

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/clients/tui/components/terminal"
	"github.com/anytty/anytty/clients/tui/history"
	"github.com/anytty/anytty/clients/tui/kernel"
	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/clients/tui/render/ansi"

	"github.com/anytty/anytty/proto/access/apipb"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// Errors reported by the PTY-backed terminal handler.
var (
	// ErrNoTerminal means the method referenced an unknown terminal id.
	ErrNoTerminal = errors.New("runtime: no such terminal")
	// ErrOwnerConflict is the minimal owner CAS failure (PROTOCOL §4/§5).
	ErrOwnerConflict = errors.New("owner conflict")
	// ErrStillRunning rejects terminal.remove on a live process (§9.3).
	ErrStillRunning = errors.New("still running; use terminal.kill")
)

// TerminalOptions configures a TerminalHandler.
type TerminalOptions struct {
	// Cols and Rows are the fallback PTY size for attach calls that do not
	// carry a solved content rect; 0 means the pty defaults (80×24).
	Cols, Rows int
	// Command is the fallback argv for attach calls without one; nil means
	// $SHELL, then /bin/sh.
	Command []string
	// Cwd and Env are host defaults for spawned processes.
	Cwd string
	Env []string
	// OwnerID is the view id that attach{fit:true} registers as resize owner.
	OwnerID string
	// OwnerLeaseTTL bounds how long a resize owner may be silent before a
	// second view can take over. Zero keeps explicit-CAS-only behavior.
	OwnerLeaseTTL time.Duration
	// Clipboard receives terminal.copy text; nil discards it.
	Clipboard func(text string) error
	// NewPTY overrides the PTY implementation (tests inject a fake).
	NewPTY func(pty.Config) pty.PTY
	// OnOutput is invoked after each parsed PTY output chunk with the source
	// id ("terminal:<endpoint>:<id>"). The host wires it to
	// Session.MarkOutput so a frame can be repainted without new input.
	OnOutput func(sourceID string)
}

// TerminalHandler is the minimal PTY-backed Handler for the terminal.*
// methods: terminal.attach starts a process and parses its output into a
// components/terminal.Screen, terminal.scroll/history.window serve the
// scrollback window, terminal.copy writes the visible text to the clipboard
// port and terminal.scrollEnd returns to live. All other methods fall back to
// the StubHandler.
type TerminalHandler struct {
	mu          sync.Mutex
	opts        TerminalOptions
	seq         uint64
	terms       map[string]*Terminal
	fallback    Handler
	listeners   map[uint64]func(string)
	listenerSeq uint64
}

// NewTerminalHandler builds a handler; a zero Cols/Rows or Command falls back
// to the PTY defaults and $SHELL.
func NewTerminalHandler(opts TerminalOptions) *TerminalHandler {
	if opts.NewPTY == nil {
		opts.NewPTY = pty.New
	}
	return &TerminalHandler{opts: opts, terms: map[string]*Terminal{}, fallback: &StubHandler{}, listeners: map[uint64]func(string){}}
}

// SubscribeOutput adds a repaint listener for PTY output and returns an
// idempotent unsubscribe function. Hosts use this when several views share
// one TerminalHandler: each view gets its own wake-up without replacing the
// handler's original OnOutput callback.
func (h *TerminalHandler) SubscribeOutput(fn func(string)) func() {
	if fn == nil {
		return func() {}
	}
	h.mu.Lock()
	h.listenerSeq++
	id := h.listenerSeq
	h.listeners[id] = fn
	h.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.listeners, id)
			h.mu.Unlock()
		})
	}
}

func (h *TerminalHandler) notifyOutput(sourceID string) {
	h.mu.Lock()
	primary := h.opts.OnOutput
	listeners := make([]func(string), 0, len(h.listeners))
	for _, fn := range h.listeners {
		listeners = append(listeners, fn)
	}
	h.mu.Unlock()
	if primary != nil {
		primary(sourceID)
	}
	for _, fn := range listeners {
		fn(sourceID)
	}
}

// Handle implements Handler.
func (h *TerminalHandler) Handle(req Request) (Outcome, bool) {
	return h.HandleContext(context.Background(), req)
}

// HandleContext permits the host to bound terminal-owned history operations.
func (h *TerminalHandler) HandleContext(ctx context.Context, req Request) (Outcome, bool) {
	params := req.Params
	switch req.Method.Name {
	case "terminal.attach":
		term, err := h.attach(params, req.OwnerID)
		if err != nil {
			return Outcome{Error: err.Error()}, false
		}
		if fitRequested(params) {
			ownerID := h.opts.OwnerID
			if req.OwnerID != "" {
				ownerID = req.OwnerID
			}
			if _, err := term.ClaimOwner(ownerID, params.GetExpectedOwnerEpoch()); err != nil {
				return Outcome{Error: err.Error()}, false
			}
		}
		return Outcome{OK: true}, false
	case "terminal.scroll":
		term := h.terminal(params.GetEndpoint(), params.GetId())
		if term == nil {
			return h.fallback.Handle(req)
		}
		rows, offset, err := term.HistoryScroll(ctx, int(params.GetDelta()), int(params.GetRows()))
		if err != nil {
			return Outcome{Error: err.Error()}, false
		}
		return Outcome{OK: true, Data: &pb.MethodData{Rows: rows, Offset: int32(offset)}}, false
	case "history.window", "terminal.history.window":
		term := h.terminal(params.GetEndpoint(), params.GetId())
		if term == nil {
			return h.fallback.Handle(req)
		}
		rows, offset, err := term.HistoryWindow(ctx, int(params.GetOffset()), int(params.GetRows()))
		if err != nil {
			return Outcome{Error: err.Error()}, false
		}
		return Outcome{OK: true, Data: &pb.MethodData{Rows: rows, Offset: int32(offset)}}, false
	case "terminal.search":
		term := h.terminal(params.GetEndpoint(), params.GetId())
		if term == nil {
			return Outcome{Error: "no such terminal"}, false
		}
		result, err := term.Search(ctx, params.GetQuery(), params.GetSearchMode(), params.GetBackward(), int(params.GetSel().GetStart()))
		if err != nil {
			return Outcome{Error: err.Error()}, false
		}
		return Outcome{OK: true, Data: result}, false
	case "terminal.scrollEnd":
		if term := h.terminal(params.GetEndpoint(), params.GetId()); term != nil {
			if err := term.HistoryRelease(ctx); err != nil {
				return Outcome{Error: err.Error()}, false
			}
		}
		return Outcome{OK: true}, false
	case "terminal.copy":
		term := h.terminal(params.GetEndpoint(), params.GetId())
		if term == nil {
			return h.fallback.Handle(req)
		}
		var spec *CopySpec
		if sel := params.GetSel(); sel != nil && sel.GetMode() != "" {
			cols := term.Cols()
			if cols < 1 {
				return Outcome{Error: "terminal has no columns"}, false
			}
			start, end := int(sel.GetStart()), int(sel.GetEnd())
			spec = &CopySpec{
				Mode:     sel.GetMode(),
				StartRow: start / cols, StartCol: start % cols,
				EndRow: end / cols, EndCol: end % cols,
			}
		}
		text, err := term.HistoryCopy(ctx, spec)
		if err != nil {
			return Outcome{Error: err.Error()}, false
		}
		if h.opts.Clipboard != nil {
			if err := h.opts.Clipboard(text); err != nil {
				return Outcome{Error: err.Error()}, false
			}
		}
		return Outcome{OK: true}, false
	case "terminal.kill":
		if term := h.terminal(params.GetEndpoint(), params.GetId()); term != nil {
			_ = term.Close()
		}
		return Outcome{OK: true}, false
	case "terminal.remove":
		return h.remove(params)
	default:
		return h.fallback.Handle(req)
	}
}

// Terminal returns an attached terminal by id.
func (h *TerminalHandler) Terminal(id string) (*Terminal, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, term := range h.terms {
		if term.ID() == id {
			return term, true
		}
	}
	return nil, false
}

// TerminalAt resolves a terminal by its protocol endpoint and id. Unlike
// Terminal, it cannot select the wrong terminal when two endpoints reuse an
// id.
func (h *TerminalHandler) TerminalAt(endpoint, id string) (*Terminal, bool) {
	term := h.terminal(endpoint, id)
	return term, term != nil
}

// TerminalBySource resolves a protocol source id ("terminal:<endpoint>:<id>")
// to its terminal, falling back to an id match for sources registered under
// another spelling.
func (h *TerminalHandler) TerminalBySource(sourceID string) (*Terminal, bool) {
	if endpoint, id, ok := parseSourceID(sourceID); ok {
		if term := h.terminal(endpoint, id); term != nil && term.Attached() {
			return term, true
		}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, term := range h.terms {
		if term.Attached() && (term.SourceID() == sourceID || term.ID() == sourceID) {
			return term, true
		}
	}
	return nil, false
}

// Detach releases one terminal's current view attachment without killing it.
func (h *TerminalHandler) Detach(endpoint, id string) error {
	term := h.terminal(endpoint, id)
	if term == nil {
		return ErrNoTerminal
	}
	return term.Detach()
}

// Reconnect rebinds a detached or temporarily disconnected terminal.
func (h *TerminalHandler) Reconnect(endpoint, id string) error {
	term := h.terminal(endpoint, id)
	if term == nil {
		return ErrNoTerminal
	}
	return term.Reattach()
}

// WriteInput writes host-encoded bytes to the PTY behind a source. It is the
// runtime's InputSink: routing and input.forward put bytes here after the
// keys package encoded them.
func (h *TerminalHandler) WriteInput(sourceID string, data []byte) error {
	term, ok := h.TerminalBySource(sourceID)
	if !ok {
		return ErrNoTerminal
	}
	_, err := term.Write(data)
	return err
}

// BracketPaste reports whether the terminal behind a source currently has
// DEC mode 2004 on (PROTOCOL §6.8).
func (h *TerminalHandler) BracketPaste(sourceID string) bool {
	term, ok := h.TerminalBySource(sourceID)
	if !ok {
		return false
	}
	return term.Modes().BracketPaste
}

// MouseSGR reports whether the child requested DEC 1006. It is optional on
// InputSink so embedders with a simple recorder keep the default SGR encoding.
func (h *TerminalHandler) MouseSGR(sourceID string) bool {
	term, ok := h.TerminalBySource(sourceID)
	if !ok {
		return true
	}
	return term.Modes().MouseSGR
}

// parseSourceID splits "terminal:<endpoint>:<id>".
func parseSourceID(sourceID string) (endpoint, id string, ok bool) {
	parts := strings.SplitN(sourceID, ":", 3)
	if len(parts) != 3 || parts[0] != "terminal" || parts[1] == "" || parts[2] == "" {
		return "", "", false
	}
	return parts[1], parts[2], true
}

// Resize changes one terminal's PTY and parser size by id.
func (h *TerminalHandler) Resize(id string, cols, rows int) error {
	term, ok := h.Terminal(id)
	if !ok {
		return ErrNoTerminal
	}
	return term.Resize(cols, rows)
}

// Close closes every attached terminal and clears the registry.
func (h *TerminalHandler) Close() {
	h.mu.Lock()
	terms := h.terms
	h.terms = map[string]*Terminal{}
	h.mu.Unlock()
	for _, term := range terms {
		_ = term.Close()
	}
}

func (h *TerminalHandler) remove(params *pb.MethodParams) (Outcome, bool) {
	key := terminalKey(params.GetEndpoint(), params.GetId())
	h.mu.Lock()
	term := h.terms[key]
	if term == nil {
		h.mu.Unlock()
		return Outcome{OK: true}, false
	}
	if !term.Exited() {
		h.mu.Unlock()
		return Outcome{Error: ErrStillRunning.Error()}, false
	}
	delete(h.terms, key)
	h.mu.Unlock()
	_ = term.Close()
	return Outcome{OK: true}, false
}

func (h *TerminalHandler) terminal(endpoint, id string) *Terminal {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.terms[terminalKey(endpoint, id)]
}

// attach starts the PTY for a source, or returns the existing terminal when
// the id is already attached. argv/cwd/env come from the params when present,
// otherwise from the handler defaults.
func (h *TerminalHandler) attach(params *pb.MethodParams, requestOwnerID string) (*Terminal, error) {
	h.mu.Lock()
	key := terminalKey(params.GetEndpoint(), params.GetId())
	if term, ok := h.terms[key]; ok {
		h.mu.Unlock()
		if err := term.Reattach(); err != nil {
			return nil, err
		}
		return term, nil
	}
	argv := append([]string(nil), params.GetArgv()...)
	if len(argv) == 0 {
		argv = append([]string(nil), h.opts.Command...)
	}
	if len(argv) == 0 {
		argv = []string{defaultShell()}
	}
	cwd := params.GetCwd()
	if cwd == "" {
		cwd = h.opts.Cwd
	}
	var env []string
	if len(h.opts.Env) > 0 || len(params.GetEnv()) > 0 {
		env = append(env, os.Environ()...)
		env = append(env, h.opts.Env...)
		for k, v := range params.GetEnv() {
			env = append(env, k+"="+v)
		}
	}
	cols, rows := h.opts.Cols, h.opts.Rows
	if cols <= 0 {
		cols = pty.DefaultCols
	}
	if rows <= 0 {
		rows = pty.DefaultRows
	}
	ownerID := h.opts.OwnerID
	if requestOwnerID != "" {
		ownerID = requestOwnerID
	}
	proc := h.opts.NewPTY(pty.Config{
		Argv: argv, Cwd: cwd, Env: env, Cols: cols, Rows: rows,
		Endpoint: params.GetEndpoint(), ID: params.GetId(),
		ViewID:             ownerID,
		Fit:                fitRequested(params),
		ExpectedOwnerEpoch: params.GetExpectedOwnerEpoch(),
	})
	if err := proc.Start(); err != nil {
		_ = proc.Close()
		h.mu.Unlock()
		return nil, err
	}
	h.seq++
	term := newTerminal(SourceID(params.GetEndpoint(), params.GetId()), params.GetId(), argv, proc, cols, rows, h.notifyOutput)
	term.ownerTTL = h.opts.OwnerLeaseTTL
	h.terms[key] = term
	h.mu.Unlock()
	term.start()
	return term, nil
}

func terminalKey(endpoint, id string) string { return endpoint + ":" + id }

// SourceID is the protocol source id of a terminal: "terminal:<endpoint>:<id>".
func SourceID(endpoint, id string) string { return "terminal:" + terminalKey(endpoint, id) }

// fitRequested mirrors the PROTOCOL §4 default: attach fits unless fit is
// explicitly false.
func fitRequested(params *pb.MethodParams) bool {
	return params.Fit == nil || *params.Fit
}

func defaultShell() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/sh"
}

// Terminal is one attached PTY plus its parsed screen state. All methods are
// safe for concurrent use; the runtime serializes view updates itself.
type Terminal struct {
	id       string
	sourceID string
	argv     []string
	proc     pty.PTY
	onOutput func(string)

	historyMu               sync.Mutex // serializes provider I/O, never held by rendering
	historySnapshot         *history.Snapshot
	historySnapshotEpoch    uint64
	historyQueueMu          sync.Mutex
	historyQueue            []historyWork
	historyRunning          bool
	historyScrollGeneration uint64
	historyContext          context.Context
	historyCancel           context.CancelFunc
	// historyScrollCancel is set only for the currently executing latest-wins
	// scroll operation. A direction change cancels it so stale scroll states
	// cannot be published after the user's newest gesture.
	historyScrollCancel context.CancelFunc
	closeOnce           sync.Once
	closeErr            error

	mu sync.Mutex
	// readMu serializes a PTY Read with the parser transaction. RemotePTY uses
	// this barrier when a reconnect snapshot replaces a stream after an old
	// chunk has already been copied out of its queue.
	readMu sync.Mutex
	parser *ansi.Parser
	// snapshotCache is invalidated only when the PTY parser changes. The host
	// may ask for a visible screen on every frame tick, so reusing this immutable
	// snapshot avoids cloning every terminal grid while it is idle.
	snapshotCache      ansi.Screen
	snapshotCacheValid bool
	screenCache        terminal.Screen
	screenCacheValid   bool
	// pinned/viewEnd implement the old frozen copy window: the first scroll
	// away from live pins the window's bottom content row, so terminal output
	// arriving afterwards does not shift what the user is reading/selecting.
	pinned   bool
	viewEnd  int
	owner    string
	epoch    uint64
	ownerAt  time.Time
	ownerTTL time.Duration
	exited   bool
	readErr  error
	attached bool
	// lastOutput is the time of the most recent non-empty PTY output; the
	// picker renders it as a coarse activity label.
	lastOutput time.Time
	// Published history viewport: read under mu without waiting for I/O.
	historyActive bool
	// historyRouting keeps wheel/mouse events in the host while the first
	// persistent history request is in flight. A remote request can take long
	// enough for the child to receive the next wheel while it still reports
	// DEC mouse tracking; routing that wheel to the PTY leaks the SGR/ESC bytes
	// which made direction reversals visibly oscillate.
	historyRoutingSeq uint64
	historyRouting    bool
	// historyEpoch invalidates provider pages and snapshots across a remote
	// attachment resync. A reconnect is a new authoritative live screen, so
	// an in-flight history request from the old attachment must not republish
	// its frozen rows over that screen.
	historyEpoch  uint64
	historyOffset int
	historyRows   []*apipb.HistoryRow
}

type historyWork struct {
	fn         func(context.Context)
	latest     bool
	generation uint64
}

func newTerminal(sourceID, id string, argv []string, proc pty.PTY, cols, rows int, onOutput func(string)) *Terminal {
	historyContext, historyCancel := context.WithCancel(context.Background())
	t := &Terminal{
		historyContext: historyContext,
		historyCancel:  historyCancel,
		id:             id,
		sourceID:       sourceID,
		argv:           append([]string(nil), argv...),
		proc:           proc,
		onOutput:       onOutput,
		parser:         ansi.New(cols, rows),
		attached:       true,
	}
	if resetter, ok := proc.(interface{ SetSnapshotReset(func()) }); ok {
		resetter.SetSnapshotReset(t.resetForSnapshot)
	}
	if barrier, ok := proc.(interface{ SetSnapshotBarrier(func()) }); ok {
		barrier.SetSnapshotBarrier(t.waitForSnapshotReset)
	}
	return t
}

// resetForSnapshot drops parser state from the previous remote stream
// generation. A reconnect snapshot is a full replacement; retaining an
// incomplete ESC/CSI sequence would join old bytes to the new snapshot and
// corrupt mode bits or screen cells before the first live frame arrives.
func (t *Terminal) resetForSnapshot() {
	t.mu.Lock()
	t.parser.Reset()
	t.invalidateSnapshotLocked()
	t.pinned, t.viewEnd = false, 0
	t.historyEpoch++
	t.historyActive, t.historyRows, t.historyOffset = false, nil, 0
	t.historyRoutingSeq++
	t.historyRouting = false
	historyEpoch := t.historyEpoch
	t.mu.Unlock()

	// Cancel interactive history work after publishing the new live-state
	// epoch. Do not hold this queue lock while touching t.mu: the latest-wins
	// publisher takes the locks in the opposite order.
	t.historyQueueMu.Lock()
	t.historyScrollGeneration++
	if t.historyScrollCancel != nil {
		t.historyScrollCancel()
	}
	t.historyQueueMu.Unlock()

	// historySnapshot is protected by historyMu because provider I/O uses it
	// outside the render lock. Release snapshots from the old epoch in the
	// background so reconnect never waits for a slow/offline provider.
	go t.closeStaleHistorySnapshot(historyEpoch)
}

// waitForSnapshotReset is called by a remote endpoint when it replaced the
// stream while the runtime had already copied an old chunk. Holding readMu
// until pump finishes parsing that chunk makes the reset happen between the
// old transaction and the next snapshot transaction.
func (t *Terminal) waitForSnapshotReset() {
	t.readMu.Lock()
	t.readMu.Unlock()
	t.resetForSnapshot()
}

func (t *Terminal) closeStaleHistorySnapshot(epoch uint64) {
	t.historyMu.Lock()
	if t.historySnapshot == nil || t.historySnapshotEpoch >= epoch {
		t.historyMu.Unlock()
		return
	}
	snapshot := t.historySnapshot
	t.historySnapshot = nil
	t.historySnapshotEpoch = 0
	t.historyMu.Unlock()
	releaseCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	_ = snapshot.Close(releaseCtx)
	cancel()
}

// ID returns the terminal id.
func (t *Terminal) ID() string { return t.id }

// SourceID returns the protocol source id ("terminal:<endpoint>:<id>").
func (t *Terminal) SourceID() string { return t.sourceID }

// Attached reports whether this terminal is currently bound to a view.
func (t *Terminal) Attached() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.attached
}

// Detach releases the view binding without killing the process. Remote PTYs
// additionally release their daemon attachment through the optional hook.
func (t *Terminal) Detach() error {
	t.mu.Lock()
	if !t.attached {
		t.mu.Unlock()
		return nil
	}
	t.attached = false
	t.mu.Unlock()
	if detachable, ok := t.proc.(interface{ Detach() error }); ok {
		return detachable.Detach()
	}
	return nil
}

// Reattach binds a previously detached terminal to the current view.
func (t *Terminal) Reattach() error {
	if reattachable, ok := t.proc.(interface{ Reattach() error }); ok {
		if err := reattachable.Reattach(); err != nil {
			return err
		}
	}
	t.mu.Lock()
	t.attached = true
	t.mu.Unlock()
	return nil
}

// Argv returns the process argv this terminal was started with.
func (t *Terminal) Argv() []string { return append([]string(nil), t.argv...) }

// Write forwards input bytes to the PTY.
func (t *Terminal) Write(p []byte) (int, error) { return t.proc.Write(p) }

// Exited reports whether the read loop saw the child close its side.
func (t *Terminal) Exited() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.exited
}

// ExitCode returns the child's exit status (-1 while still unknown).
func (t *Terminal) ExitCode() int { return t.proc.ExitCode() }

// ReadError returns the error that ended the read loop, if any.
func (t *Terminal) ReadError() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.readErr
}

// LastOutput returns the time of the most recent non-empty output.
func (t *Terminal) LastOutput() time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastOutput
}

// Size reports the current PTY window size.
func (t *Terminal) Size() (cols, rows int, err error) { return t.proc.Size() }

// Close terminates the process and releases the PTY (idempotent).
func (t *Terminal) Close() error {
	t.closeOnce.Do(func() {
		t.historyCancel()
		t.closeErr = t.proc.Close()
		// Cleanup must not be dropped when the request queue is saturated.
		// Cancellation releases in-flight I/O before this acquires historyMu.
		go func() { _ = t.HistoryRelease(context.Background()) }()
	})
	return t.closeErr
}

// Resize applies a new window size to the PTY and the parser.
func (t *Terminal) Resize(cols, rows int) error {
	if cols <= 0 || rows <= 0 {
		return pty.ErrInvalidSize
	}
	if err := t.proc.Resize(cols, rows); err != nil {
		return err
	}
	t.mu.Lock()
	t.parser.Resize(cols, rows)
	t.invalidateSnapshotLocked()
	t.mu.Unlock()
	return nil
}

// State is one consistent snapshot of a terminal: the component screen, the
// cursor and the parser mode bits.
type TerminalState struct {
	Screen        terminal.Screen
	CursorX       int
	CursorY       int
	CursorVisible bool
	CursorShape   string
	Modes         ansi.Modes
}

// State snapshots the parsed screen.
func (t *Terminal) State() TerminalState {
	t.mu.Lock()
	defer t.mu.Unlock()
	snap := t.snapshotLocked()
	return TerminalState{
		Screen:        t.liveScreenLocked(snap),
		CursorX:       snap.CursorX,
		CursorY:       snap.CursorY,
		CursorVisible: snap.CursorVisible,
		CursorShape:   snap.CursorShape,
		Modes:         t.parser.Modes(),
	}
}

// Screen returns just the component screen snapshot.
func (t *Terminal) Screen() terminal.Screen { return t.State().Screen }

// VisibleScreen returns the styled viewport, including frozen history. Plain
// text projections such as VisibleLines are for searching/copying, not display.
func (t *Terminal) VisibleScreen() terminal.Screen {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.historyActive {
		return historyScreen(t.historyRows, t.parser.Cols())
	}
	if t.pinned {
		return terminalScreen(ansi.Screen{Lines: t.windowCellsEndingLocked(t.windowEndLocked())})
	}
	return t.liveScreenLocked(t.snapshotLocked())
}

// Cols returns the terminal grid width (display cells per row).
func (t *Terminal) Cols() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.parser.Cols()
}

// Modes returns the parser mode bits (mouse tracking, bracket paste, …).
func (t *Terminal) Modes() ansi.Modes {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.parser.Modes()
}

// Cursor returns the grid cursor position and visibility.
func (t *Terminal) Cursor() (int, int, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.parser.Cursor()
}

// Offset returns the scrollback offset (0 = live): the distance between the
// window the view shows and the live bottom. A pinned (frozen) window reports
// the growing distance as new output arrives.
func (t *Terminal) Offset() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.offsetLocked()
}

func (t *Terminal) contentTotalLocked() int {
	return len(t.parser.Scrollback()) + len(t.parser.Screen().Lines)
}

func (t *Terminal) offsetLocked() int {
	if t.historyActive {
		return t.historyOffset
	}
	if !t.pinned {
		return 0
	}
	total := t.contentTotalLocked()
	if t.viewEnd >= total {
		return 0
	}
	return total - t.viewEnd
}

// windowEndLocked is the exclusive content row the visible window ends at.
func (t *Terminal) windowEndLocked() int {
	total := t.contentTotalLocked()
	if t.pinned && t.viewEnd < total {
		return t.viewEnd
	}
	return total
}

// Window returns up to rows lines ending `offset` lines before the view's
// bottom (the frozen copy window when scrolled, the live tail otherwise) plus
// the clamped offset actually used. offset=0 with rows<=0 is the visible
// window; a positive offset pages further back for the copy search, which is
// how the whole frozen history can be scanned. The frozen bottom itself never
// moves when output arrives, so a scan window is stable.
func (t *Terminal) Window(offset, rows int) ([]string, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if offset < 0 {
		offset = 0
	}
	if maxOffset := t.contentTotalLocked(); offset > maxOffset {
		offset = maxOffset
	}
	end := t.windowEndLocked() - offset
	return t.windowEndingLocked(end, rows), offset
}

// Scroll moves the view by delta lines (positive = older) and returns the
// resulting window plus the clamped offset the view now shows. The first
// scroll away from live pins the window (the old frozen copy/history window):
// later terminal output does not move it, the offset just grows.
func (t *Terminal) Scroll(delta, rows int) ([]string, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	total := t.contentTotalLocked()
	if !t.pinned {
		if delta <= 0 {
			return t.windowEndingLocked(total, rows), 0
		}
		t.pinned = true
		t.viewEnd = total - delta
	} else {
		t.viewEnd -= delta
	}
	minEnd := t.windowRowsLocked(rows)
	if minEnd > total {
		minEnd = total
	}
	if t.viewEnd < minEnd {
		t.viewEnd = minEnd
	}
	if t.viewEnd >= total {
		t.pinned = false
		t.viewEnd = 0
	}
	return t.windowEndingLocked(t.windowEndLocked(), rows), t.offsetLocked()
}

// maxOffsetLocked is the largest offset the history can show.
func (t *Terminal) maxOffsetLocked() int {
	history := len(t.parser.Scrollback())
	if history < 0 {
		return 0
	}
	return history
}

// windowRowsLocked resolves the requested row count: a non-positive value
// means "one visible screen".
func (t *Terminal) windowRowsLocked(rows int) int {
	if rows > 0 {
		return rows
	}
	snap := t.snapshotLocked()
	if len(snap.Lines) > 0 {
		return len(snap.Lines)
	}
	return t.parser.Rows()
}

// windowEndingLocked builds the window of windowRowsLocked(rows) rows ending
// at the exclusive content row `end`.
func (t *Terminal) windowEndingLocked(end, rows int) []string {
	history := t.parser.Scrollback()
	snap := t.parser.Screen()
	rows = t.windowRowsLocked(rows)
	total := len(history) + len(snap.Lines)
	if end > total {
		end = total
	}
	if end < 0 {
		end = 0
	}
	start := end - rows
	if start < 0 {
		start = 0
	}
	out := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		if i < len(history) {
			out = append(out, strings.TrimRight(ansi.RowText(history[i]), " "))
			continue
		}
		out = append(out, strings.TrimRight(snap.Text(i-len(history)), " "))
	}
	return out
}

// ScrollEnd returns the view to live and drops the frozen window.
func (t *Terminal) ScrollEnd() {
	t.mu.Lock()
	t.pinned = false
	t.viewEnd = 0
	t.mu.Unlock()
}

// VisibleLines returns the rows the view currently shows (live or scrolled),
// trailing spaces trimmed.
func (t *Terminal) VisibleLines() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.historyActive {
		return history.Text(t.historyRows)
	}
	if t.pinned {
		return t.windowEndingLocked(t.windowEndLocked(), 0)
	}
	snap := t.parser.Screen()
	out := make([]string, len(snap.Lines))
	for y := range snap.Lines {
		out[y] = strings.TrimRight(snap.Text(y), " ")
	}
	return out
}

// CopySpec is one terminal.copy selection expressed in the currently visible
// window: rows are 0-based from the top of the terminal content area (the
// scrolled window when the view is not live), columns are display cells, and
// both endpoints are inclusive. Mode is "char" (reading-order stream),
// "line" (whole rows) or "block" (rectangle).
type CopySpec struct {
	Mode     string
	StartRow int
	StartCol int
	EndRow   int
	EndCol   int
}

// CopyWindow resolves a selection against the window the view currently
// shows and returns the text (rows joined with "\n", trailing blanks
// trimmed). ok is false when the terminal has no content.
func (t *Terminal) CopyWindow(spec CopySpec) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	rows := t.windowCellsEndingLocked(t.windowEndLocked())
	if t.historyActive {
		rows = historyCells(t.historyRows)
	}
	if len(rows) == 0 {
		return "", false
	}
	mode := strings.ToLower(strings.TrimSpace(spec.Mode))
	if mode == "" {
		mode = "char"
	}
	clampRow := func(row int) int {
		if row < 0 {
			return 0
		}
		if row > len(rows)-1 {
			return len(rows) - 1
		}
		return row
	}
	r1, r2 := clampRow(spec.StartRow), clampRow(spec.EndRow)
	c1, c2 := spec.StartCol, spec.EndCol
	if c1 < 0 {
		c1 = 0
	}
	if c2 < 0 {
		c2 = 0
	}
	switch mode {
	case "line":
		if r1 > r2 {
			r1, r2 = r2, r1
		}
		out := make([]string, 0, r2-r1+1)
		for row := r1; row <= r2; row++ {
			out = append(out, strings.TrimRight(sliceCells(rows[row], 0, maxCells(rows[row])), " "))
		}
		return strings.Join(out, "\n"), true
	case "block":
		if r1 > r2 {
			r1, r2 = r2, r1
		}
		if c1 > c2 {
			c1, c2 = c2, c1
		}
		out := make([]string, 0, r2-r1+1)
		for row := r1; row <= r2; row++ {
			out = append(out, strings.TrimRight(sliceCells(rows[row], c1, c2+1), " "))
		}
		return strings.Join(out, "\n"), true
	default: // char: reading-order stream from start to end
		if r1 > r2 || (r1 == r2 && c1 > c2) {
			r1, r2 = r2, r1
			c1, c2 = c2, c1
		}
		out := make([]string, 0, r2-r1+1)
		for row := r1; row <= r2; row++ {
			start := 0
			if row == r1 {
				start = c1
			}
			end := maxCells(rows[row])
			if row == r2 {
				end = c2 + 1
			}
			out = append(out, strings.TrimRight(sliceCells(rows[row], start, end), " "))
		}
		return strings.Join(out, "\n"), true
	}
}

// windowCellsEndingLocked returns the visible window (one screen of rows
// ending at the exclusive content row `end`) as raw cells, preserving
// wide-rune clusters so a selection slice can resolve display columns. The
// end comes from the frozen view (windowEndLocked), so a copy selection
// resolves against exactly what the view shows.
func (t *Terminal) windowCellsEndingLocked(end int) [][]ansi.Cell {
	history := t.parser.Scrollback()
	snap := t.parser.Screen()
	height := len(snap.Lines)
	if height == 0 {
		height = t.parser.Rows()
	}
	total := len(history) + len(snap.Lines)
	if end > total {
		end = total
	}
	if end < 0 {
		end = 0
	}
	start := end - height
	if start < 0 {
		start = 0
	}
	out := make([][]ansi.Cell, 0, end-start)
	for i := start; i < end; i++ {
		if i < len(history) {
			out = append(out, history[i])
			continue
		}
		out = append(out, snap.Lines[i-len(history)])
	}
	return out
}

// maxCells is the display width of one cell row.
func maxCells(row []ansi.Cell) int {
	width := 0
	for _, cell := range row {
		w := cell.Width
		if w <= 0 {
			w = 1
		}
		width += w
	}
	return width
}

// sliceCells extracts the display-cell range [start, end) of one row. A wide
// cluster that overlaps the range is included whole (columns can never split
// a grapheme).
func sliceCells(row []ansi.Cell, start, end int) string {
	if start >= end {
		return ""
	}
	var b strings.Builder
	col := 0
	for _, cell := range row {
		w := cell.Width
		if w <= 0 {
			w = 1
		}
		cellStart, cellEnd := col, col+w
		col = cellEnd
		if cellStart >= end {
			break
		}
		if cellEnd <= start {
			continue
		}
		if cell.Continuation {
			continue
		}
		b.WriteString(cell.Text)
	}
	return b.String()
}

// CopyText is the default terminal.copy payload: the visible screen or
// scrollback window with trailing blank lines removed.
func (t *Terminal) CopyText() string {
	lines := t.VisibleLines()
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// ClaimOwner is the minimal resize-owner CAS (PROTOCOL §5): the first owner
// wins, the same owner renews, a different owner needs the current epoch.
func (t *Terminal) ClaimOwner(owner string, expectedEpoch uint64) (uint64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if owner == "" {
		return t.epoch, nil
	}
	now := time.Now()
	if t.owner != "" && t.ownerTTL > 0 && !t.ownerAt.IsZero() && now.Sub(t.ownerAt) >= t.ownerTTL {
		// A lease expiry is an implicit release. Preserve the epoch so a
		// takeover is still visible as a new epoch below.
		t.owner = ""
	}
	if t.owner != "" && t.owner != owner && expectedEpoch != t.epoch {
		return t.epoch, ErrOwnerConflict
	}
	if t.owner != owner {
		t.owner = owner
		t.epoch++
	}
	t.ownerAt = now
	return t.epoch, nil
}

// RenewOwner refreshes an active lease without changing its epoch. It is
// called while a view continues to render a focused terminal.
func (t *Terminal) RenewOwner(owner string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if owner == "" || t.owner != owner {
		return false
	}
	if t.ownerTTL > 0 && !t.ownerAt.IsZero() && time.Since(t.ownerAt) >= t.ownerTTL {
		t.owner = ""
		return false
	}
	t.ownerAt = time.Now()
	return true
}

// Owner returns the current resize owner and its epoch.
func (t *Terminal) Owner() (string, uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.owner != "" && t.ownerTTL > 0 && !t.ownerAt.IsZero() && time.Since(t.ownerAt) >= t.ownerTTL {
		t.owner = ""
	}
	return t.owner, t.epoch
}

// ResizeOwner returns the authoritative owner for remote PTYs and the local
// lease for local PTYs.
func (t *Terminal) ResizeOwner() (string, uint64) {
	if remote, ok := t.proc.(pty.ResizeOwner); ok {
		return remote.ResizeOwner()
	}
	return t.Owner()
}

// RefreshResizeOwner refreshes a remote daemon's owner projection. Local
// PTYs have no remote control plane and return false.
func (t *Terminal) RefreshResizeOwner() bool {
	if remote, ok := t.proc.(interface{ RefreshResizeOwner() bool }); ok {
		return remote.RefreshResizeOwner()
	}
	return false
}

// Placement renders component at rect and, when the placement is focused and
// showing live output, carries the PTY cursor mapped to the component frame.
// The content box is derived from the inset the component declares for this
// rect, so a resized component stays the single source of truth for chrome.
func (t *Terminal) Placement(component *terminal.Component, rect kernel.Rect, focused bool) Placement {
	inset := component.Inset(rect.Width, rect.Height)
	contentRect := rect.Inset(inset)
	placement := Placement{
		Rect:                      rect,
		ContentRect:               contentRect,
		Lines:                     component.Render(rect.Width, rect.Height),
		Props:                     component.Props().Chrome,
		Focused:                   focused,
		DisableScrollOptimization: component.Offset() == 0,
	}
	t.mu.Lock()
	placement.SynchronizedOutput = t.parser.Modes().SynchronizedOutput
	t.mu.Unlock()
	if !focused || rect.Empty() || component.Offset() > 0 {
		return placement
	}
	t.mu.Lock()
	snap := t.snapshotLocked()
	modes := t.parser.Modes()
	t.mu.Unlock()
	if !modes.CursorVisible {
		return placement
	}
	contentW, contentH := rect.Width-2*inset, rect.Height-2*inset
	if contentW <= 0 || contentH <= 0 {
		return placement
	}
	offset, ok := snap.CursorOffset(snap.CursorX, snap.CursorY)
	if !ok || offset >= contentW || snap.CursorY >= contentH {
		return placement
	}
	placement.CursorX = inset + offset
	placement.CursorY = inset + snap.CursorY
	placement.CursorVisible = true
	placement.CursorShape = snap.CursorShape
	return placement
}

func (t *Terminal) start() {
	go t.pump()
}

func (t *Terminal) pump() {
	buf := make([]byte, 32*1024)
	for {
		t.readMu.Lock()
		n, err := t.proc.Read(buf)
		if n > 0 {
			t.mu.Lock()
			t.parser.Write(buf[:n])
			t.invalidateSnapshotLocked()
			nowSynchronized := t.parser.Modes().SynchronizedOutput
			if len(bytes.TrimSpace(buf[:n])) > 0 {
				t.lastOutput = time.Now()
			}
			t.mu.Unlock()
			// openCode and similar full-screen apps wrap a redraw in DEC 2026.
			// Do not wake the host for intermediate chunks; the chunk that
			// carries ?2026l publishes the completed screen immediately.
			if t.onOutput != nil && !nowSynchronized {
				t.onOutput(t.sourceID)
			}
		}
		if done, ok := t.proc.(interface{ ReadDone() }); ok {
			done.ReadDone()
		}
		t.readMu.Unlock()
		if err != nil {
			t.mu.Lock()
			t.exited = true
			t.readErr = err
			onOutput := t.onOutput
			sourceID := t.sourceID
			t.mu.Unlock()
			// A child can exit before closing a DEC 2026 batch. Publish the
			// last parsed cells so the host does not remain on an old frame.
			if onOutput != nil {
				onOutput(sourceID)
			}
			return
		}
	}
}

func (t *Terminal) invalidateSnapshotLocked() {
	t.snapshotCacheValid = false
	t.screenCacheValid = false
}

func (t *Terminal) snapshotLocked() ansi.Screen {
	if !t.snapshotCacheValid {
		t.snapshotCache = t.parser.Screen()
		t.snapshotCacheValid = true
	}
	return t.snapshotCache
}

func (t *Terminal) liveScreenLocked(snap ansi.Screen) terminal.Screen {
	if !t.screenCacheValid {
		t.screenCache = terminalScreen(snap)
		t.screenCacheValid = true
	}
	return t.screenCache
}

// terminalScreen converts a parser snapshot into the terminal component's
// one-cell-per-cluster screen: continuation halves fold into their head and
// blank cells become spaces so column alignment survives.
func terminalScreen(snap ansi.Screen) terminal.Screen {
	lines := make([][]terminal.Cell, len(snap.Lines))
	for y, row := range snap.Lines {
		cells := make([]terminal.Cell, 0, len(row))
		for _, cell := range row {
			if cell.Continuation {
				continue
			}
			if cell.Text == "" {
				cells = append(cells, terminal.Cell{Text: " ", Width: 1, Style: cell.Style})
				continue
			}
			width := cell.Width
			if width <= 0 {
				width = 1
			}
			cells = append(cells, terminal.Cell{Text: cell.Text, Width: width, Style: cell.Style})
		}
		lines[y] = cells
	}
	return terminal.Screen{Lines: lines}
}

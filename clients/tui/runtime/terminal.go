package runtime

import (
	"errors"
	"os"
	"strings"
	"sync"

	"github.com/anytty/anytty/clients/tui/components/terminal"
	"github.com/anytty/anytty/clients/tui/kernel"
	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/clients/tui/render/ansi"

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
	mu       sync.Mutex
	opts     TerminalOptions
	seq      uint64
	terms    map[string]*Terminal
	fallback Handler
}

// NewTerminalHandler builds a handler; a zero Cols/Rows or Command falls back
// to the PTY defaults and $SHELL.
func NewTerminalHandler(opts TerminalOptions) *TerminalHandler {
	if opts.NewPTY == nil {
		opts.NewPTY = pty.New
	}
	return &TerminalHandler{opts: opts, terms: map[string]*Terminal{}, fallback: &StubHandler{}}
}

// Handle implements Handler.
func (h *TerminalHandler) Handle(req Request) (Outcome, bool) {
	params := req.Params
	switch req.Method.Name {
	case "terminal.attach":
		term, err := h.attach(params)
		if err != nil {
			return Outcome{Error: err.Error()}, false
		}
		if fitRequested(params) {
			if _, err := term.ClaimOwner(h.opts.OwnerID, params.GetExpectedOwnerEpoch()); err != nil {
				return Outcome{Error: err.Error()}, false
			}
		}
		return Outcome{OK: true}, false
	case "terminal.scroll":
		term := h.terminal(params.GetEndpoint(), params.GetId())
		if term == nil {
			return h.fallback.Handle(req)
		}
		rows := term.Scroll(int(params.GetDelta()), int(params.GetRows()))
		return Outcome{OK: true, Data: &pb.MethodData{Rows: rows}}, false
	case "history.window":
		term := h.terminal(params.GetEndpoint(), params.GetId())
		if term == nil {
			return h.fallback.Handle(req)
		}
		rows := term.Window(int(params.GetOffset()), int(params.GetRows()))
		return Outcome{OK: true, Data: &pb.MethodData{Rows: rows}}, false
	case "terminal.scrollEnd":
		if term := h.terminal(params.GetEndpoint(), params.GetId()); term != nil {
			term.ScrollEnd()
		}
		return Outcome{OK: true}, false
	case "terminal.copy":
		term := h.terminal(params.GetEndpoint(), params.GetId())
		if term == nil {
			return h.fallback.Handle(req)
		}
		if h.opts.Clipboard != nil {
			if err := h.opts.Clipboard(term.CopyText()); err != nil {
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

// TerminalBySource resolves a protocol source id ("terminal:<endpoint>:<id>")
// to its terminal, falling back to an id match for sources registered under
// another spelling.
func (h *TerminalHandler) TerminalBySource(sourceID string) (*Terminal, bool) {
	if endpoint, id, ok := parseSourceID(sourceID); ok {
		if term := h.terminal(endpoint, id); term != nil {
			return term, true
		}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, term := range h.terms {
		if term.SourceID() == sourceID || term.ID() == sourceID {
			return term, true
		}
	}
	return nil, false
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
func (h *TerminalHandler) attach(params *pb.MethodParams) (*Terminal, error) {
	h.mu.Lock()
	key := terminalKey(params.GetEndpoint(), params.GetId())
	if term, ok := h.terms[key]; ok {
		h.mu.Unlock()
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
	proc := h.opts.NewPTY(pty.Config{Argv: argv, Cwd: cwd, Env: env, Cols: cols, Rows: rows, Endpoint: params.GetEndpoint(), ID: params.GetId()})
	if err := proc.Start(); err != nil {
		_ = proc.Close()
		h.mu.Unlock()
		return nil, err
	}
	h.seq++
	term := newTerminal(SourceID(params.GetEndpoint(), params.GetId()), params.GetId(), argv, proc, cols, rows, h.opts.OnOutput)
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

	mu      sync.Mutex
	parser  *ansi.Parser
	offset  int
	owner   string
	epoch   uint64
	exited  bool
	readErr error
}

func newTerminal(sourceID, id string, argv []string, proc pty.PTY, cols, rows int, onOutput func(string)) *Terminal {
	return &Terminal{
		id:       id,
		sourceID: sourceID,
		argv:     append([]string(nil), argv...),
		proc:     proc,
		onOutput: onOutput,
		parser:   ansi.New(cols, rows),
	}
}

// ID returns the terminal id.
func (t *Terminal) ID() string { return t.id }

// SourceID returns the protocol source id ("terminal:<endpoint>:<id>").
func (t *Terminal) SourceID() string { return t.sourceID }

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

// Size reports the current PTY window size.
func (t *Terminal) Size() (cols, rows int, err error) { return t.proc.Size() }

// Close terminates the process and releases the PTY (idempotent).
func (t *Terminal) Close() error { return t.proc.Close() }

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
	Modes         ansi.Modes
}

// State snapshots the parsed screen.
func (t *Terminal) State() TerminalState {
	t.mu.Lock()
	defer t.mu.Unlock()
	snap := t.parser.Screen()
	return TerminalState{
		Screen:        terminalScreen(snap),
		CursorX:       snap.CursorX,
		CursorY:       snap.CursorY,
		CursorVisible: snap.CursorVisible,
		Modes:         t.parser.Modes(),
	}
}

// Screen returns just the component screen snapshot.
func (t *Terminal) Screen() terminal.Screen { return t.State().Screen }

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

// Offset returns the scrollback offset (0 = live).
func (t *Terminal) Offset() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.offset
}

// Window returns up to rows lines ending offset lines before the live bottom.
// A non-positive rows means "one visible screen".
func (t *Terminal) Window(offset, rows int) []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.windowLocked(offset, rows)
}

// Scroll moves the view by delta lines (positive = older) and returns the
// resulting window; the offset clamps to the available scrollback.
func (t *Terminal) Scroll(delta, rows int) []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.offset += delta
	if t.offset < 0 {
		t.offset = 0
	}
	if history := len(t.parser.Scrollback()); t.offset > history {
		t.offset = history
	}
	return t.windowLocked(t.offset, rows)
}

// ScrollEnd returns the view to live.
func (t *Terminal) ScrollEnd() {
	t.mu.Lock()
	t.offset = 0
	t.mu.Unlock()
}

// VisibleLines returns the rows the view currently shows (live or scrolled),
// trailing spaces trimmed.
func (t *Terminal) VisibleLines() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.offset > 0 {
		return t.windowLocked(t.offset, 0)
	}
	snap := t.parser.Screen()
	out := make([]string, len(snap.Lines))
	for y := range snap.Lines {
		out[y] = strings.TrimRight(snap.Text(y), " ")
	}
	return out
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
	if t.owner != "" && t.owner != owner && expectedEpoch != t.epoch {
		return t.epoch, ErrOwnerConflict
	}
	if t.owner != owner {
		t.owner = owner
		t.epoch++
	}
	return t.epoch, nil
}

// Owner returns the current resize owner and its epoch.
func (t *Terminal) Owner() (string, uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.owner, t.epoch
}

// Placement renders component at rect and, when the placement is focused and
// showing live output, carries the PTY cursor mapped to the component frame.
// The content box is derived from the inset the component declares for this
// rect, so a resized component stays the single source of truth for chrome.
func (t *Terminal) Placement(component *terminal.Component, rect kernel.Rect, focused bool) Placement {
	placement := Placement{
		Rect:    rect,
		Lines:   component.Render(rect.Width, rect.Height),
		Props:   component.Props().Chrome,
		Focused: focused,
	}
	if !focused || rect.Empty() || component.Offset() > 0 {
		return placement
	}
	t.mu.Lock()
	snap := t.parser.Screen()
	modes := t.parser.Modes()
	t.mu.Unlock()
	if !modes.CursorVisible {
		return placement
	}
	inset := component.Inset(rect.Width, rect.Height)
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
	return placement
}

func (t *Terminal) start() {
	go t.pump()
}

func (t *Terminal) pump() {
	buf := make([]byte, 32*1024)
	for {
		n, err := t.proc.Read(buf)
		if n > 0 {
			t.mu.Lock()
			t.parser.Write(buf[:n])
			t.mu.Unlock()
			if t.onOutput != nil {
				t.onOutput(t.sourceID)
			}
		}
		if err != nil {
			t.mu.Lock()
			t.exited = true
			t.readErr = err
			t.mu.Unlock()
			return
		}
	}
}

// windowLocked joins the scrollback and the live screen and returns the rows
// ending offset lines before the live bottom.
func (t *Terminal) windowLocked(offset, rows int) []string {
	history := t.parser.Scrollback()
	snap := t.parser.Screen()
	if rows <= 0 {
		rows = len(snap.Lines)
	}
	total := len(history) + len(snap.Lines)
	end := total - offset
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

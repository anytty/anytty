package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	"github.com/anytty/anytty/clients/tui/components/terminal"
	"github.com/anytty/anytty/clients/tui/endpoint"
	"github.com/anytty/anytty/clients/tui/kernel"
	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/clients/tui/render"
	"github.com/anytty/anytty/clients/tui/runtime"
	"github.com/anytty/anytty/clients/tui/runtime/keys"
	"github.com/anytty/anytty/proto/access/apipb"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// viewID is the connection identifier of the single local view
// (PROTOCOL §0.5: one client connection = one view).
const viewID = "view:local:1"

// escTimeout is how long a lone Esc waits for the rest of a sequence before
// being delivered as the Esc key.
const escTimeout = 50 * time.Millisecond

// restartDelay is the backoff before the layout program is started again
// after a crash or a clean exit (SCENARIOS §3).
const restartDelay = 250 * time.Millisecond

// immediateExitWindow classifies a program that dies right after start: the
// host prints an explicit diagnostic in that case instead of restarting
// silently, so a bad -shell command is visible.
const immediateExitWindow = time.Second

// Options configures a Host. In/Out and the process/PTY factories are
// injectable so tests can run without a real TTY or a real child process.
type Options struct {
	Shell       []string
	In          io.Reader
	Out         io.Writer
	NewProcess  func(argv []string) (process, error)
	NewPTY      func(pty.Config) pty.PTY
	Cols, Rows  int
	Tick        time.Duration
	RestartWait time.Duration
	// Endpoints is the daemon endpoint manager (ENDPOINTS.zh-CN.md §2). Nil
	// creates a private manager owned and closed by this host.
	Endpoints *endpoint.Manager
	// LoadSharedRegistry loads every CLI-paired endpoint from the shared
	// endpoints.yaml registries at startup (M2). main.go enables it; tests
	// keep it off so they stay deterministic.
	LoadSharedRegistry bool
	// RegistryPaths is the ordered read list of CLI-owned endpoints.yaml
	// registries: explicit TUI2_ENDPOINTS/-endpoints files first, then the
	// default client/endpoint.DefaultPath(). Read-only; explicit files win on
	// duplicate endpoint names. Empty resolves TUI2_ENDPOINTS + default.
	RegistryPaths []string
	// Routes is the shared route policy of the endpoint manager (nil = the
	// default: local-unix plus credential-gated ssh). main.go resolves the
	// -routes flag / TUI2_ROUTES; tests keep it off so they stay hermetic.
	Routes []clientendpoint.RouteKind
	// Logf receives host diagnostics (layout program crashes, input errors
	// and endpoint notices). Defaults to the standard logger, which main.go
	// has already redirected to the TUI log file, so these lines can never
	// reach the alt screen. Tests inject a collector.
	Logf func(format string, args ...any)
	// Dev is the -dev instrumentation (frame log, stderr capture, hot
	// reload). Nil keeps the production behavior; non-nil is additive.
	Dev *devMode
}

// trackedTerminal is one locally known terminal the host publishes as a
// content source.
type trackedTerminal struct {
	term *runtime.Terminal
}

// pendingConfirm is a destructive action waiting for the user: either a
// program RESULT that needs authorization or the host-initiated Ctrl-Q.
type pendingConfirm struct {
	req          runtime.Request
	hostQuit     bool
	programQuit  bool
	cleanupOwned bool
}

// Host owns the local single-machine session: its TTY, the layout program,
// the terminal handler and the frame loop.
type Host struct {
	opts    Options
	handler *runtime.TerminalHandler
	gate    *gateHandler
	tick    time.Duration

	mu          sync.Mutex
	session     *runtime.Session
	proc        process
	started     time.Time
	cols, rows  int
	epoch       uint64
	outMu       sync.Mutex
	tracked     map[string]*trackedTerminal
	sources     []*pb.Source
	sizes       map[string][2]int
	confirm     *pendingConfirm
	quitting    bool
	sourcesSent bool

	confirmCh chan struct{}

	components map[string]*terminal.Component

	endpoints     *endpoint.Manager
	registryPaths []string
	ownsEndpoints bool
	logf          func(format string, args ...any)
	noticeMu      sync.Mutex
	notices       [][2]string
}

// NewHost wires the session, the PTY handler and the confirmation gate.
func NewHost(opts Options) *Host {
	if opts.Cols <= 0 {
		opts.Cols = 80
	}
	if opts.Rows <= 0 {
		opts.Rows = 24
	}
	if opts.Tick <= 0 {
		opts.Tick = 16 * time.Millisecond
	}
	if opts.RestartWait <= 0 {
		opts.RestartWait = restartDelay
	}
	localPTY := opts.NewPTY
	if localPTY == nil {
		localPTY = pty.New
	}
	if opts.NewProcess == nil {
		if opts.Dev != nil {
			dev := opts.Dev
			opts.NewProcess = func(argv []string) (process, error) { return dev.startProcess(argv) }
		} else {
			opts.NewProcess = startExecProcess
		}
	}
	if opts.Logf == nil {
		opts.Logf = log.Printf
	}
	ownsEndpoints := false
	registryPaths := append([]string(nil), opts.RegistryPaths...)
	if len(registryPaths) == 0 {
		registryPaths = endpoint.RegistryPathsFromEnv()
	}
	if opts.Endpoints == nil {
		opts.Endpoints = endpoint.NewManager(endpoint.Options{RegistryPaths: registryPaths, Routes: opts.Routes})
		ownsEndpoints = true
	}
	h := &Host{
		opts:          opts,
		cols:          opts.Cols,
		rows:          opts.Rows,
		tick:          opts.Tick,
		tracked:       map[string]*trackedTerminal{},
		sizes:         map[string][2]int{},
		components:    map[string]*terminal.Component{},
		confirmCh:     make(chan struct{}, 1),
		endpoints:     opts.Endpoints,
		registryPaths: registryPaths,
		ownsEndpoints: ownsEndpoints,
		logf:          opts.Logf,
	}
	if h.opts.Dev != nil {
		h.opts.Dev.onNotice = h.queueNotice
	}
	opts.Endpoints.SetOnNotice(h.queueNotice)
	if ownsEndpoints && opts.LoadSharedRegistry {
		h.registerSharedEndpoints()
	}
	opts.NewPTY = func(cfg pty.Config) pty.PTY {
		if cfg.Endpoint != "" {
			if kind, ok := opts.Endpoints.Kind(cfg.Endpoint); ok && kind == endpoint.KindDaemon {
				return opts.Endpoints.NewRemotePTY(cfg)
			}
		}
		return localPTY(cfg)
	}
	h.handler = runtime.NewTerminalHandler(runtime.TerminalOptions{
		Cols:      opts.Cols,
		Rows:      opts.Rows,
		OwnerID:   viewID,
		NewPTY:    opts.NewPTY,
		Clipboard: h.writeClipboard,
		OnOutput: func(string) {
			h.withSession(func(s *runtime.Session) { s.MarkOutput() })
		},
	})
	h.gate = &gateHandler{host: h, inner: h.handler}
	return h
}

// Resize follows the TTY size (SIGWINCH).
func (h *Host) Resize(cols, rows int) {
	if cols <= 0 || rows <= 0 {
		return
	}
	h.mu.Lock()
	h.cols, h.rows = cols, rows
	h.mu.Unlock()
	h.withSession(func(s *runtime.Session) { s.Resize(cols, rows) })
}

func (h *Host) size() (int, int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.cols, h.rows
}

// Run owns the terminal until quit or a fatal error: alt screen, raw frame
// loop, program supervision and confirmation overlay.
func (h *Host) Run() error {
	if err := h.writeOut(render.EnterScreen()); err != nil {
		return err
	}
	defer func() {
		_ = h.writeOut(render.ExitScreen())
	}()

	if err := h.startProgram(); err != nil {
		return err
	}
	defer h.stopProgram()
	if h.ownsEndpoints {
		defer func() { _ = h.endpoints.Close() }()
	}

	parser := keys.NewParser()
	inputCh := make(chan []byte, 64)
	go h.pumpInput(inputCh)

	sessionDone := make(chan sessionEnd, 4)
	h.serve(sessionDone)

	ticker := time.NewTicker(h.tick)
	defer ticker.Stop()

	for {
		select {
		case chunk, ok := <-inputCh:
			if !ok {
				return nil
			}
			for _, ev := range parser.Feed(chunk) {
				h.handleInput(ev)
			}
		case end := <-sessionDone:
			if end.session != h.currentSession() {
				// A deliberate restart (hot reload) closes the old pipes;
				// its Serve error must not restart the new session again.
				break
			}
			if h.isQuitting() {
				return nil
			}
			h.restartProgram(end.err)
			h.serve(sessionDone)
		case <-h.confirmCh:
			h.openPendingConfirm()
		case path := <-h.reloadChannel():
			h.reloadProgram(path)
			h.serve(sessionDone)
		case <-ticker.C:
			for _, ev := range parser.Flush(escTimeout) {
				h.handleInput(ev)
			}
		}
		h.publishSources()
		h.flushNotices()
		h.flush()
		if h.isQuitting() {
			return nil
		}
	}
}

// sessionEnd pairs a finished Serve with the exact session that produced it,
// so the frame loop can ignore a stale end from a deliberately stopped
// program (hot reload) instead of restarting the new one.
type sessionEnd struct {
	session *runtime.Session
	err     error
}

func (h *Host) serve(done chan<- sessionEnd) {
	session := h.currentSession()
	go func() {
		if session == nil {
			done <- sessionEnd{}
			return
		}
		done <- sessionEnd{session: session, err: session.Serve()}
	}()
}

func (h *Host) withSession(fn func(*runtime.Session)) {
	if session := h.currentSession(); session != nil {
		fn(session)
	}
}

func (h *Host) currentSession() *runtime.Session {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.session
}

func (h *Host) startProgram() error {
	proc, err := h.opts.NewProcess(h.opts.Shell)
	if err != nil {
		return err
	}
	h.mu.Lock()
	h.epoch++
	epoch := h.epoch
	old := h.proc
	h.proc = proc
	h.started = time.Now()
	cols, rows := h.cols, h.rows
	h.mu.Unlock()
	if old != nil {
		old.Stop()
	}

	programOut, programIn := proc.Stdout(), proc.Stdin()
	if h.opts.Dev != nil {
		// In dev mode both pipes are tapped so every frame reaches the
		// protocol log before the runtime sees it (byte-for-byte passthrough).
		programOut = h.opts.Dev.tapRead(programOut)
		programIn = h.opts.Dev.tapWrite(programIn)
	}
	session := runtime.NewSession(runtime.Options{
		ViewID:     viewID,
		Epoch:      epoch,
		Cols:       cols,
		Rows:       rows,
		Components: []string{"terminal"},
		Handler:    h.gate,
		InputSink:  h.handler,
		MouseTracking: func(id string) bool {
			term, ok := h.handler.TerminalBySource(id)
			return ok && term.Modes().MouseTracking()
		},
	}, programOut, programIn)

	h.mu.Lock()
	h.session = session
	h.sourcesSent = false
	h.mu.Unlock()

	if err := session.SendHello(); err != nil {
		// The program can die between Start and the handshake write. Keep the
		// session so the frame loop observes the end of the stream and applies
		// the usual restart policy instead of leaving a half-dead host behind.
		return nil
	}
	h.publishSources()
	return nil
}

// uptime reports how long the current program has been running (zero before
// the first start).
func (h *Host) uptime() time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.started.IsZero() {
		return 0
	}
	return time.Since(h.started)
}

func (h *Host) stopProgram() {
	h.mu.Lock()
	proc := h.proc
	h.proc = nil
	h.mu.Unlock()
	if proc != nil {
		proc.Stop()
	}
}

// reloadChannel is the hot-reload signal of -watch/-reload-on-save; nil in
// production mode, which makes the select case block forever.
func (h *Host) reloadChannel() <-chan string {
	if h.opts.Dev == nil {
		return nil
	}
	return h.opts.Dev.reload
}

// reloadProgram restarts the layout program after a watched file changed.
// The old frame is kept until the new program commits its first view, and a
// one-line notice tells the developer which file triggered the reload.
func (h *Host) reloadProgram(path string) {
	h.logf("tui2: reload %s", path)
	h.queueNotice("info", "reloaded "+path)
	h.restartProgramMode(nil, true)
}

func (h *Host) restartProgram(cause error) { h.restartProgramMode(cause, false) }

func (h *Host) restartProgramMode(cause error, reload bool) {
	uptime := h.uptime()
	h.stopProgram()
	var stderrTail []string
	if h.opts.Dev != nil {
		// Stop waited for the child, so its stderr ring is complete.
		stderrTail = h.opts.Dev.stderrTail()
	}
	h.mu.Lock()
	h.confirm = nil
	h.mu.Unlock()
	var diagnostic string
	switch {
	case reload:
		// The "reloaded <file>" notice is already queued; no crash text.
	case cause != nil && !errors.Is(cause, io.EOF):
		diagnostic = fmt.Sprintf("layout program stopped: %v", cause)
		if h.opts.Dev != nil {
			diagnostic += fmt.Sprintf("; restarting in %s", h.opts.RestartWait)
		}
	case uptime < immediateExitWindow:
		diagnostic = fmt.Sprintf("layout program exited immediately (%s after start); restarting in %s",
			uptime.Round(time.Millisecond), h.opts.RestartWait)
	}
	if diagnostic != "" && h.opts.Dev != nil && len(stderrTail) > 0 {
		// "程序 stderr 末尾若干行以 notice 呈现": one compact notice line so
		// the footer status stays readable.
		diagnostic += "; stderr: " + strings.Join(stderrTail, " / ")
	}
	if diagnostic != "" {
		// Terminal output stays frame-only: the diagnostic goes to the log
		// file and to the layout program as a notice.
		h.logf("tui2: %s", diagnostic)
		h.queueNotice("warning", diagnostic)
	}
	time.Sleep(h.opts.RestartWait)
	if err := h.startProgram(); err != nil {
		diagnostic := fmt.Sprintf("layout program failed to start: %v; giving up", err)
		h.logf("tui2: %s", diagnostic)
		h.queueNotice("warning", diagnostic)
		h.quit()
	}
}

func (h *Host) pumpInput(ch chan<- []byte) {
	defer close(ch)
	buf := make([]byte, 4096)
	for {
		n, err := h.opts.In.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			ch <- chunk
		}
		if err != nil {
			return
		}
	}
}

// handleInput routes one normalized event: core overlay first, then the
// §6.5 table through the session, with implicit drag capture for non-terminal
// boxes (PROTOCOL §6.7).
func (h *Host) handleInput(ev keys.Event) {
	session := h.currentSession()
	if session == nil {
		return
	}
	if session.CoreOverlayOpen() {
		if ev.Kind == keys.KindKey {
			switch keys.Name(ev) {
			case "enter":
				h.confirmYes()
			case "esc", "ctrl-c":
				h.confirmNo()
			}
		}
		return
	}
	if ev.Kind == keys.KindMouse || ev.Kind == keys.KindWheel {
		h.preparePointer(session, &ev)
	}
	dst, err := h.route(ev)
	if err != nil {
		h.logf("tui2: input: %v", err)
		h.queueNotice("warning", fmt.Sprintf("input: %v", err))
	}
	if dst == runtime.DestinationHost && ev.Kind == keys.KindKey && keys.Name(ev) == runtime.KeyCtrlQ {
		h.askHostQuit()
	}
	if ev.Kind == keys.KindMouse && ev.Action == keys.ActionRelease {
		session.ReleaseCapture()
	}
}

func (h *Host) route(ev keys.Event) (runtime.Destination, error) {
	var dst runtime.Destination
	var err error
	h.withSession(func(s *runtime.Session) { dst, err = s.Input(ev) })
	return dst, err
}

// preparePointer fills the hit test results of a mouse/wheel event and starts
// the implicit capture on a non-terminal box that declares input:"mouse".
func (h *Host) preparePointer(session *runtime.Session, ev *keys.Event) {
	frame, ok := session.Frame()
	if !ok {
		return
	}
	if captured := session.CapturedNode(); captured != "" {
		ev.Node = captured
		return
	}
	hit := frame.Hit(ev.X-1, ev.Y-1)
	ev.Node = hit
	if ev.Kind == keys.KindWheel {
		return
	}
	ev.HitFocused = hit != "" && hit == focusedBoxID(session, hit)
	if ev.Action == keys.ActionPress && hit != "" && isCaptureBox(session, hit) {
		session.Capture(hit)
	}
}

func focusedBoxID(session *runtime.Session, hit string) string {
	view := session.View()
	if view == nil {
		return ""
	}
	var found string
	var walk func(*pb.Box)
	walk = func(b *pb.Box) {
		if b == nil || found != "" {
			return
		}
		if b.GetFocused() && b.GetContent().GetSelf() != "" {
			found = b.GetId()
			return
		}
		for _, child := range b.GetChildren() {
			walk(child)
		}
	}
	walk(view.GetRoot())
	return found
}

func isCaptureBox(session *runtime.Session, id string) bool {
	view := session.View()
	if view == nil {
		return false
	}
	var box *pb.Box
	var walk func(*pb.Box)
	walk = func(b *pb.Box) {
		if b == nil || box != nil {
			return
		}
		if b.GetId() == id {
			box = b
			return
		}
		for _, child := range b.GetChildren() {
			walk(child)
		}
	}
	walk(view.GetRoot())
	if box == nil || box.GetContent().GetSelf() != "" {
		return false
	}
	for _, in := range box.GetInput() {
		if in == "mouse" {
			return true
		}
	}
	return false
}

// placements renders every bound terminal source for the current view and
// keeps its PTY window size aligned with the solved content rect.
func (h *Host) placements() []runtime.Placement {
	session := h.currentSession()
	if session == nil {
		return nil
	}
	view := session.View()
	frame, ok := session.Frame()
	if !ok || view == nil {
		return nil
	}
	var out []runtime.Placement
	var walk func(*pb.Box)
	walk = func(b *pb.Box) {
		if b == nil {
			return
		}
		sourceID := b.GetContent().GetSelf()
		if sourceID != "" {
			if term, ok := h.handler.TerminalBySource(sourceID); ok {
				if rect, ok := frame.Rect(b.GetId()); ok {
					out = append(out, h.placement(session, sourceID, term, b, rect))
				}
			}
		}
		for _, child := range b.GetChildren() {
			walk(child)
		}
	}
	walk(view.GetRoot())
	return out
}

// placement pushes the declarative state of one terminal box into its
// component and renders it. The box's program-declared props (content.props)
// are passed through untouched: the component interprets them, the host never
// does.
func (h *Host) placement(session *runtime.Session, sourceID string, term *runtime.Terminal, box *pb.Box, rect kernel.Rect) runtime.Placement {
	component := h.components[sourceID]
	if component == nil {
		component = terminal.New(nil, nil)
		h.components[sourceID] = component
	}
	offset := term.Offset()
	source := h.sourceByID(sourceID)
	props := terminal.Props{
		Title:        sourceTitle(source, term),
		Focused:      box.GetFocused(),
		Scrolled:     offset > 0,
		ScrollOffset: offset,
		Exited:       term.Exited(),
		Chrome:       session.BoxProps(box.GetId()),
	}
	// 中文说明：card 风格 pane 自己画框，程序用 chrome.inset=0 关掉组件边框，
	// PTY 尺寸随之等于完整内容矩形（不再固定减 2）。
	if inset, ok := terminal.InsetFromProps(box.GetContent().GetProps()); ok {
		props.Inset = inset
		props.InsetSet = true
	}
	if source != nil {
		props.ExitCode = int(source.GetExitCode())
	}
	component.SetProps(props)
	if offset > 0 {
		component.SetScreen(terminal.ScreenFromText(term.VisibleLines(), render.TokenDefault))
	} else {
		component.SetScreen(term.Screen())
	}
	inset := component.Inset(rect.Width, rect.Height)
	h.resizePTY(term, rect, inset)
	return term.Placement(component, rect, box.GetFocused())
}

// resizePTY aligns the PTY winsize with the content rect the component
// declares for this placement: rect minus the component's own chrome inset
// (the host does not assume a fixed "width-2/height-2").
func (h *Host) resizePTY(term *runtime.Terminal, rect kernel.Rect, inset int) {
	cols, rows := rect.Width-2*inset, rect.Height-2*inset
	if cols <= 0 || rows <= 0 {
		return
	}
	sourceID := term.SourceID()
	if last, ok := h.sizes[sourceID]; ok && last == [2]int{cols, rows} {
		return
	}
	if err := term.Resize(cols, rows); err != nil {
		return
	}
	h.sizes[sourceID] = [2]int{cols, rows}
}

func (h *Host) flush() {
	session := h.currentSession()
	if session == nil {
		return
	}
	if data := session.FrameBytes(h.placements(), nil); len(data) > 0 {
		if err := h.writeOutBytes(data); err != nil {
			h.quit()
		}
	}
}

// writeOut serializes one control-sequence write with the frame loop.
func (h *Host) writeOut(text string) error {
	h.outMu.Lock()
	defer h.outMu.Unlock()
	_, err := io.WriteString(h.opts.Out, text)
	return err
}

func (h *Host) writeOutBytes(data []byte) error {
	h.outMu.Lock()
	defer h.outMu.Unlock()
	_, err := h.opts.Out.Write(data)
	return err
}

// writeClipboard puts terminal.copy text on the client clipboard as OSC 52
// (PROTOCOL §9.3); tmux and modern terminals turn that into a real clipboard
// entry. It runs on the session goroutine, so writes share the output lock
// with the frame loop.
func (h *Host) writeClipboard(text string) error {
	sequence := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07"
	return h.writeOut(sequence)
}

// publishSources keeps the program's sources snapshot in sync with the
// locally tracked terminals (exit state, owner epoch).
func (h *Host) publishSources() {
	session := h.currentSession()
	if session == nil {
		return
	}
	items, changed := h.snapshotSources()
	if !changed {
		return
	}
	session.SetSources(items)
}

func (h *Host) snapshotSources() ([]*pb.Source, bool) {
	h.mu.Lock()
	items := make([]*pb.Source, 0, len(h.tracked))
	tracked := make(map[string]bool, len(h.tracked))
	for sourceID, trackedTerm := range h.tracked {
		endpointName, id := splitSourceID(sourceID)
		owner, epoch := trackedTerm.term.Owner()
		exitCode := trackedTerm.term.ExitCode()
		if exitCode < 0 {
			exitCode = 0
		}
		health := "ok"
		if kind, ok := h.endpoints.Kind(endpointName); ok && kind == endpoint.KindDaemon {
			health = h.endpoints.Health(endpointName)
		}
		items = append(items, &pb.Source{
			Id:          sourceID,
			Kind:        "terminal",
			Title:       trackedTerm.term.ID(),
			Endpoint:    endpointName,
			TerminalId:  id,
			Attached:    true,
			Exited:      trackedTerm.term.Exited(),
			ExitCode:    int32(exitCode),
			Health:      health,
			ResizeOwner: owner,
			OwnerEpoch:  epoch,
		})
		tracked[sourceID] = true
	}
	h.mu.Unlock()
	// Daemon-listed terminals that have no local attachment yet are added
	// unattached so the picker can bind them position-transparently.
	for _, source := range h.endpoints.Sources() {
		if tracked[source.GetId()] {
			continue
		}
		items = append(items, source)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].GetId() < items[j].GetId() })
	changed := !h.sourcesSent || len(items) != len(h.sources)
	if !changed {
		for i, item := range items {
			old := h.sources[i]
			if old.GetId() != item.GetId() || old.GetExited() != item.GetExited() ||
				old.GetExitCode() != item.GetExitCode() || old.GetOwnerEpoch() != item.GetOwnerEpoch() ||
				old.GetHealth() != item.GetHealth() || old.GetAttached() != item.GetAttached() {
				changed = true
				break
			}
		}
	}
	if changed {
		h.sources = items
		h.sourcesSent = true
	}
	return items, changed
}

func (h *Host) trackSource(sourceID string, term *runtime.Terminal) {
	h.mu.Lock()
	h.tracked[sourceID] = &trackedTerminal{term: term}
	h.mu.Unlock()
}

func (h *Host) sourceByID(id string) *pb.Source {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, src := range h.sources {
		if src.GetId() == id {
			return src
		}
	}
	return nil
}

func splitSourceID(sourceID string) (endpoint, id string) {
	parts := strings.SplitN(sourceID, ":", 3)
	if len(parts) != 3 {
		return "local", sourceID
	}
	return parts[1], parts[2]
}

func sourceTitle(source *pb.Source, term *runtime.Terminal) string {
	if source != nil && source.GetTitle() != "" {
		return source.GetTitle()
	}
	return term.ID()
}

// askHostQuit opens the host confirmation for Ctrl-Q (the reserved key).
func (h *Host) askHostQuit() {
	h.mu.Lock()
	if h.confirm == nil {
		h.confirm = &pendingConfirm{hostQuit: true}
	}
	h.mu.Unlock()
	h.signalConfirm()
}

// requestConfirm queues a program RESULT that needs authorization. It runs on
// the session goroutine, so it must not touch the session itself.
func (h *Host) requestConfirm(req runtime.Request, programQuit, cleanupOwned bool) {
	h.mu.Lock()
	if h.confirm == nil {
		h.confirm = &pendingConfirm{req: req, programQuit: programQuit, cleanupOwned: cleanupOwned}
	}
	h.mu.Unlock()
	h.signalConfirm()
}

func (h *Host) signalConfirm() {
	select {
	case h.confirmCh <- struct{}{}:
	default:
	}
}

func (h *Host) takeConfirm() *pendingConfirm {
	h.mu.Lock()
	defer h.mu.Unlock()
	confirm := h.confirm
	h.confirm = nil
	return confirm
}

func (h *Host) openPendingConfirm() {
	h.mu.Lock()
	confirm := h.confirm
	cols, rows := h.cols, h.rows
	h.mu.Unlock()
	if confirm == nil {
		return
	}
	message := "Quit tui2?"
	if confirm.programQuit {
		message = "Layout program requests quit?"
	} else if confirm.req.Method.Name != "" {
		message = "Allow " + confirm.req.Method.Name + "?"
	}
	h.withSession(func(s *runtime.Session) { s.OpenCoreOverlay(confirmFrame(cols, rows, message)) })
}

func confirmFrame(cols, rows int, message string) kernel.Frame {
	width := minInt(56, maxInt(20, cols-4))
	height := 7
	x := maxInt(0, (cols-width)/2)
	y := maxInt(0, (rows-height)/2)
	root := &kernel.Node{
		ID: "core.root",
		Children: []kernel.Node{{
			ID:      "core.confirm",
			Pos:     &kernel.Pos{X: x, Y: y},
			Size:    kernel.Size{Width: width, Height: height},
			Style:   "warning",
			Content: &kernel.Content{Lines: confirmLines(width, message)},
		}},
	}
	return kernel.Layout(root, cols, rows)
}

// confirmLines self-draws the core overlay chrome (the kernel has no border
// concept): the host is the only renderer of this frame.
func confirmLines(width int, message string) []string {
	inner := maxInt(0, width-2)
	row := func(text string) string {
		body := render.Truncate(text, inner)
		return "│" + body + strings.Repeat(" ", inner-kernel.DisplayWidth(body)) + "│"
	}
	title := render.Truncate(" confirm ", inner)
	top := "┌" + title + strings.Repeat("─", maxInt(0, inner-kernel.DisplayWidth(title))) + "┐"
	bottom := "└" + strings.Repeat("─", inner) + "┘"
	return []string{
		top,
		row(""),
		row(" " + message),
		row(""),
		row(" enter allow · esc deny"),
		row(""),
		bottom,
	}
}

func (h *Host) confirmYes() {
	confirm := h.takeConfirm()
	h.withSession(func(s *runtime.Session) { s.CloseCoreOverlay() })
	if confirm == nil {
		return
	}
	switch {
	case confirm.hostQuit:
		h.quit()
	case confirm.programQuit:
		h.withSession(func(s *runtime.Session) { _ = s.Complete(confirm.req.RequestID, nil, "") })
		h.quit()
	default:
		outcome, pending := h.handleConfirmed(confirm.req)
		if pending {
			h.withSession(func(s *runtime.Session) {
				_ = s.Complete(confirm.req.RequestID, nil, "unsupported")
			})
			return
		}
		h.withSession(func(s *runtime.Session) {
			_ = s.Complete(confirm.req.RequestID, outcome.Data, outcome.Error)
		})
	}
}

func (h *Host) confirmNo() {
	confirm := h.takeConfirm()
	h.withSession(func(s *runtime.Session) { s.CloseCoreOverlay() })
	if confirm != nil && !confirm.hostQuit {
		h.withSession(func(s *runtime.Session) {
			_ = s.Complete(confirm.req.RequestID, nil, "denied by user")
		})
	}
}

func (h *Host) quit() {
	h.mu.Lock()
	h.quitting = true
	h.mu.Unlock()
}

func (h *Host) isQuitting() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.quitting
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

// queueNotice stores one endpoint notice until the frame loop forwards it to
// the layout program (the manager may be on any goroutine) and mirrors it to
// the log file, so an offline endpoint is diagnosable after the fact.
func (h *Host) queueNotice(level, message string) {
	h.logf("tui2 notice level=%s message=%s", level, message)
	h.noticeMu.Lock()
	if len(h.notices) < 32 {
		h.notices = append(h.notices, [2]string{level, message})
	}
	h.noticeMu.Unlock()
}

func (h *Host) flushNotices() {
	h.noticeMu.Lock()
	notices := h.notices
	h.notices = nil
	h.noticeMu.Unlock()
	for _, notice := range notices {
		h.withSession(func(s *runtime.Session) { _ = s.SendNotice(notice[0], notice[1]) })
	}
}

// registerSharedEndpoints loads the CLI-owned endpoints.yaml registries at
// startup (explicit TUI2_ENDPOINTS/-endpoints first, then the default path).
// It is read-only: pairing and management stay with the CLI, and a missing or
// corrupt registry is reported as a notice instead of crashing the host.
func (h *Host) registerSharedEndpoints() {
	configs, warnings := endpoint.LoadSharedEndpointConfigs(h.registryPaths...)
	for _, warning := range warnings {
		h.queueNotice("warning", warning)
	}
	for _, cfg := range configs {
		if err := h.endpoints.Register(cfg); err != nil {
			h.queueNotice("warning", fmt.Sprintf("endpoint %s: %v", cfg.Name, err))
		}
	}
}

// registerEndpoint learns a daemon endpoint from protocol params. Local
// command endpoints need no registration: they stay on the v1 PTY path.
// The CLI-owned shared endpoints.yaml registry is the canonical source: a
// same-name shared entry wins over the tui2.json compatibility fields, so
// direct/cloud/paired endpoints are not duplicated into a second config
// (tui2/docs/CLIENT_SHARING.zh-CN.md §4).
func (h *Host) registerEndpoint(params *pb.MethodParams) error {
	if params.GetKind() != endpoint.KindDaemon {
		return nil
	}
	cfg := endpoint.Config{
		Name:        params.GetEndpoint(),
		Kind:        params.GetKind(),
		Socket:      params.GetSocket(),
		Address:     params.GetAddress(),
		ConnectMode: params.GetConnectMode(),
	}
	if shared, ok, err := endpoint.SharedConfigForEndpointIn(h.registryPaths, cfg.Name); err != nil {
		return err
	} else if ok {
		cfg = shared
	}
	return h.endpoints.Register(cfg)
}

func (h *Host) isDaemonEndpoint(name string) bool {
	kind, ok := h.endpoints.Kind(name)
	return ok && kind == endpoint.KindDaemon
}

// handleConfirmed executes an authorized destructive request. Daemon terminal
// removal additionally deletes the terminal on the daemon and drops the local
// source tracking entry.
func (h *Host) handleConfirmed(req runtime.Request) (runtime.Outcome, bool) {
	outcome, pending := h.handler.Handle(req)
	if !outcome.OK || pending {
		return outcome, pending
	}
	if req.Method.Name == "terminal.remove" && h.isDaemonEndpoint(req.Params.GetEndpoint()) {
		if err := h.endpoints.Remove(context.Background(), req.Params.GetEndpoint(), req.Params.GetId()); err != nil {
			return runtime.Outcome{Error: err.Error()}, false
		}
		h.mu.Lock()
		delete(h.tracked, runtime.SourceID(req.Params.GetEndpoint(), req.Params.GetId()))
		h.mu.Unlock()
	}
	return outcome, pending
}

// gateHandler is the authorization point before the real TerminalHandler:
// terminal.create and terminal.restart get real implementations, destructive
// methods go through the core overlay, and everything else is delegated.
type gateHandler struct {
	host  *Host
	inner *runtime.TerminalHandler
	seq   int
}

func (g *gateHandler) Handle(req runtime.Request) (runtime.Outcome, bool) {
	switch req.Method.Name {
	case "endpoint.sync":
		if err := g.host.registerEndpoint(req.Params); err != nil {
			return runtime.Outcome{Error: err.Error()}, false
		}
		return runtime.Outcome{OK: true}, false
	case "terminal.create":
		if err := g.host.registerEndpoint(req.Params); err != nil {
			return runtime.Outcome{Error: err.Error()}, false
		}
		if g.host.isDaemonEndpoint(req.Params.GetEndpoint()) {
			return g.createDaemon(req)
		}
		return g.create(req)
	case "terminal.restart":
		if g.host.isDaemonEndpoint(req.Params.GetEndpoint()) {
			return g.restartDaemon(req)
		}
		return g.restart(req)
	case "terminal.attach":
		if err := g.host.registerEndpoint(req.Params); err != nil {
			return runtime.Outcome{Error: err.Error()}, false
		}
		outcome, pending := g.inner.Handle(req)
		if outcome.OK && !pending {
			g.track(req.Params.GetEndpoint(), req.Params.GetId())
		}
		return outcome, pending
	case "system.quit":
		cleanup := false
		if req.Params != nil {
			cleanup = req.Params.GetCleanupOwned()
		}
		g.host.requestConfirm(req, true, cleanup)
		return runtime.Outcome{}, true
	default:
		if req.Method.Confirm {
			g.host.requestConfirm(req, false, false)
			return runtime.Outcome{}, true
		}
		return g.inner.Handle(req)
	}
}

func (g *gateHandler) create(req runtime.Request) (runtime.Outcome, bool) {
	g.seq++
	endpoint := req.Params.GetEndpoint()
	if endpoint == "" {
		endpoint = "local"
	}
	id := "term-" + strconv.Itoa(g.seq)
	params := &pb.MethodParams{
		Endpoint: endpoint,
		Id:       id,
		Argv:     req.Params.GetArgv(),
		Cwd:      req.Params.GetCwd(),
		Env:      req.Params.GetEnv(),
		Title:    req.Params.GetTitle(),
	}
	outcome, pending := g.inner.Handle(runtime.Request{
		Epoch:     req.Epoch,
		RequestID: req.RequestID,
		Method:    runtime.Method{Name: "terminal.attach"},
		Params:    params,
	})
	if !outcome.OK || pending {
		return outcome, pending
	}
	g.track(endpoint, id)
	return runtime.Outcome{
		OK:   true,
		Data: &pb.MethodData{Endpoint: endpoint, Id: id},
	}, false
}

// createDaemon creates a terminal on the daemon endpoint and attaches it. The
// daemon assigns or accepts the terminal id, and that id becomes the protocol
// source id terminal:<endpoint>:<id>.
func (g *gateHandler) createDaemon(req runtime.Request) (runtime.Outcome, bool) {
	endpointName := req.Params.GetEndpoint()
	cols, rows := g.host.size()
	spec := &apipb.TerminalCreateSpec{
		TerminalId: req.Params.GetId(),
		Name:       req.Params.GetTitle(),
		Command:    append([]string(nil), req.Params.GetArgv()...),
		Cwd:        req.Params.GetCwd(),
		Env:        envSlice(req.Params.GetEnv()),
		Size:       &apipb.TerminalSize{Cols: uint32(cols), Rows: uint32(rows)},
	}
	info, err := g.host.endpoints.Create(context.Background(), endpointName, spec)
	if err != nil {
		return runtime.Outcome{Error: err.Error()}, false
	}
	id := info.GetRef().GetTerminalId()
	if id == "" {
		return runtime.Outcome{Error: "daemon returned an empty terminal id"}, false
	}
	command := append([]string(nil), info.GetCommand()...)
	if len(command) == 0 {
		command = append([]string(nil), req.Params.GetArgv()...)
	}
	params := &pb.MethodParams{
		Endpoint:    endpointName,
		Id:          id,
		Argv:        command,
		Cwd:         req.Params.GetCwd(),
		Env:         req.Params.GetEnv(),
		Title:       req.Params.GetTitle(),
		Kind:        req.Params.GetKind(),
		Socket:      req.Params.GetSocket(),
		Address:     req.Params.GetAddress(),
		ConnectMode: req.Params.GetConnectMode(),
	}
	outcome, pending := g.inner.Handle(runtime.Request{
		Epoch:     req.Epoch,
		RequestID: req.RequestID,
		Method:    runtime.Method{Name: "terminal.attach"},
		Params:    params,
	})
	if !outcome.OK || pending {
		return outcome, pending
	}
	g.track(endpointName, id)
	return runtime.Outcome{
		OK:   true,
		Data: &pb.MethodData{Endpoint: endpointName, Id: id},
	}, false
}

// restartDaemon kills and removes the daemon terminal, recreates it with the
// same id and command, and re-attaches under the same protocol source.
func (g *gateHandler) restartDaemon(req runtime.Request) (runtime.Outcome, bool) {
	endpointName := req.Params.GetEndpoint()
	id := req.Params.GetId()
	term, ok := g.inner.Terminal(id)
	if !ok {
		return runtime.Outcome{Error: "no such terminal"}, false
	}
	argv := term.Argv()
	_ = term.Close()
	deadline := time.Now().Add(2 * time.Second)
	for !term.Exited() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	g.inner.Handle(runtime.Request{
		Method: runtime.Method{Name: "terminal.remove"},
		Params: &pb.MethodParams{Endpoint: endpointName, Id: id},
	})
	if err := g.host.endpoints.Restart(context.Background(), endpointName, id); err != nil {
		return runtime.Outcome{Error: err.Error()}, false
	}
	outcome, pending := g.inner.Handle(runtime.Request{
		Epoch:     req.Epoch,
		RequestID: req.RequestID,
		Method:    runtime.Method{Name: "terminal.attach"},
		Params: &pb.MethodParams{
			Endpoint: endpointName, Id: id, Argv: argv,
			Kind: req.Params.GetKind(), Socket: req.Params.GetSocket(), Address: req.Params.GetAddress(), ConnectMode: req.Params.GetConnectMode(),
		},
	})
	if !outcome.OK || pending {
		return outcome, pending
	}
	g.track(endpointName, id)
	return runtime.Outcome{OK: true}, false
}

// envSlice converts an endpoint env map to a deterministic slice.
func envSlice(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+env[key])
	}
	return out
}

// restart closes and removes the old process, then attaches a fresh one under
// the same terminal id (SCENARIOS §3).
func (g *gateHandler) restart(req runtime.Request) (runtime.Outcome, bool) {
	endpoint := req.Params.GetEndpoint()
	id := req.Params.GetId()
	term, ok := g.inner.Terminal(id)
	if !ok {
		return runtime.Outcome{Error: "no such terminal"}, false
	}
	argv := term.Argv()
	_ = term.Close()
	deadline := time.Now().Add(2 * time.Second)
	for !term.Exited() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	g.inner.Handle(runtime.Request{
		Method: runtime.Method{Name: "terminal.remove"},
		Params: &pb.MethodParams{Endpoint: endpoint, Id: id},
	})
	outcome, pending := g.inner.Handle(runtime.Request{
		Epoch:     req.Epoch,
		RequestID: req.RequestID,
		Method:    runtime.Method{Name: "terminal.attach"},
		Params:    &pb.MethodParams{Endpoint: endpoint, Id: id, Argv: argv},
	})
	if !outcome.OK || pending {
		return outcome, pending
	}
	g.track(endpoint, id)
	return runtime.Outcome{OK: true}, false
}

func (g *gateHandler) track(endpoint, id string) {
	if id == "" {
		return
	}
	if endpoint == "" {
		endpoint = "local"
	}
	term, ok := g.inner.Terminal(id)
	if !ok {
		return
	}
	g.host.trackSource(runtime.SourceID(endpoint, id), term)
}

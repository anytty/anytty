package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

// viewID is the deterministic fallback used by embedded/test hosts. The
// command entry point supplies a process-unique ViewID so separate clients
// cannot be mistaken for the same resize owner.
const viewID = "view:local:1"

// wheelDebugEnabled is the TUI2_DEBUG_WHEEL escape hatch: when set, the host
// logs one line per routed wheel (destination, focus capabilities), per raw
// input chunk that carries an escape sequence, and per terminal history
// operation. It is off in production and costs a single env lookup per wheel.
var wheelDebugEnabled = sync.OnceValue(func() bool {
	value := strings.TrimSpace(os.Getenv("TUI2_DEBUG_WHEEL"))
	return value == "1" || strings.EqualFold(value, "true")
})

// chunkHasEsc reports whether a raw input chunk carries an ESC byte, i.e. it
// may hold a mouse report or another control sequence worth logging.
func chunkHasEsc(chunk []byte) bool {
	for _, b := range chunk {
		if b == 0x1b {
			return true
		}
	}
	return false
}

// escTimeout is how long a lone Esc waits for the rest of a sequence before
// being delivered as the Esc key.
const escTimeout = 50 * time.Millisecond

// frameCoalesceWindow bounds the time spent merging a burst of wakeups into
// one render. The first frame after an idle period is still immediate; only
// events that arrive during the previous frame's short budget are merged.
const frameCoalesceWindow = 8 * time.Millisecond

// interactionUrgencyWindow keeps PTY output caused by a recent user event on
// the immediate path. Background output still uses frame coalescing.
const interactionUrgencyWindow = 150 * time.Millisecond

// restartDelay is the backoff before the layout program is started again
// after a crash or a clean exit (SCENARIOS §3).
const restartDelay = 250 * time.Millisecond

// sourceRefreshInterval bounds how stale the daemon attachment_count shown in
// the pane badge can be. It is deliberately low-frequency: the count only
// changes when another client attaches/detaches, and each tick costs one
// terminal.list per connected daemon endpoint. The manager emits no change
// event when the list is unchanged, so an idle tick repaints nothing.
const sourceRefreshInterval = 3 * time.Second

// immediateExitWindow classifies a program that dies right after start: the
// host prints an explicit diagnostic in that case instead of restarting
// silently, so a bad -shell command is visible.
const immediateExitWindow = time.Second

// Options configures a Host. In/Out and the process/PTY factories are
// injectable so tests can run without a real TTY or a real child process.
type Options struct {
	Shell []string
	// ViewID identifies this layout connection for resize-owner leases. The
	// default keeps the historic single local view; callers embedding Host can
	// supply distinct ids and share a TerminalHandler for multiple views.
	ViewID     string
	In         io.Reader
	Out        io.Writer
	NewProcess func(argv []string) (process, error)
	NewPTY     func(pty.Config) pty.PTY
	Cols, Rows int
	// Tick is accepted for compatibility but unused: the frame loop is
	// event-driven, with a short bounded window for merging burst wakeups.
	Tick        time.Duration
	RestartWait time.Duration
	// Endpoints is the daemon endpoint manager (ENDPOINTS.zh-CN.md §2). Nil
	// creates a private manager owned and closed by this host.
	Endpoints *endpoint.Manager
	// Handler optionally supplies a shared terminal handler. This is useful for
	// multiple Host views over one terminal inventory; ownership is taken from
	// the Request.OwnerID metadata rather than a process-global owner string.
	Handler *runtime.TerminalHandler
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
	// Limits overrides the HELLO limits (message size, paste size, in-flight
	// requests). Zero keeps runtime.DefaultLimits.
	Limits runtime.Limits
	// DialTimeout/CallTimeout override the endpoint manager timeouts. Zero
	// keeps the endpoint package defaults.
	DialTimeout time.Duration
	CallTimeout time.Duration
	// Logf receives host diagnostics (layout program crashes, input errors
	// and endpoint notices). Defaults to the standard logger, which main.go
	// has already redirected to the TUI log file, so these lines can never
	// reach the alt screen. Tests inject a collector.
	Logf func(format string, args ...any)
	// Dev is the -dev instrumentation (frame log, stderr capture, hot
	// reload). Nil keeps the production behavior; non-nil is additive.
	Dev *devMode
	// ClipboardHistoryPath and ClipboardHistoryMaxItems configure the host
	// owned persistent clipboard store. Empty path selects the XDG state path.
	ClipboardHistoryPath     string
	ClipboardHistoryMaxItems int
	// ClipboardRead supplies the system clipboard for clipboard.read/paste.
	// Nil uses the platform clipboard command (pbpaste/wl-paste/xclip).
	ClipboardRead func() (string, error)
}

// trackedTerminal is one locally known terminal the host publishes as a
// content source.
type trackedTerminal struct {
	term      *runtime.Terminal
	ephemeral bool
	title     string
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
	// outputUnsub scopes PTY repaint notifications to this view when several
	// hosts share one TerminalHandler.
	outputUnsub func()

	mu         sync.Mutex
	session    *runtime.Session
	proc       process
	started    time.Time
	cols, rows int
	epoch      uint64
	outMu      sync.Mutex
	tracked    map[string]*trackedTerminal
	sources    []*pb.Source
	sizes      map[string][2]int
	// pendingSize coalesces per-terminal resize requests: while a resize is in
	// flight the newest requested size is kept and applied when it returns, so
	// a drag burst never queues one blocking round-trip per frame.
	pendingSize map[string][2]int
	resizing    map[string]bool
	confirm     *pendingConfirm
	quitting    bool
	sourcesSent bool
	ownerMu     sync.Mutex
	ownerBusy   bool
	// sourceMu/sourceBusy coalesce the observer-count refresh: at most one list
	// round-trip per endpoint is in flight, so a slow daemon cannot pile up
	// refreshes across ticks.
	sourceMu   sync.Mutex
	sourceBusy bool
	// terminalMouseDown keeps PTY drag/release events routed to the focused
	// terminal after the pointer leaves its panel.
	terminalMouseDown bool

	confirmCh chan struct{}
	// wake is the background repaint signal: PTY output, endpoint changes and
	// notices wake the frame loop without polling. Buffered(1) so a burst
	// coalesces into one wake; urgentWake is reserved for interactive commits.
	wake       chan struct{}
	urgentWake chan struct{}
	wakeOnce   sync.Once
	// lastInteraction is read by PTY output callbacks, which run outside the
	// frame loop. UnixNano keeps the callback lock-free and bounded.
	lastInteraction atomic.Int64

	components    map[string]*terminal.Component
	accessStreams map[uint64]*accessStreamBridge
	accessEvents  map[uint64]*accessEventBridge
	// accessPending reserves stream ids whose open/subscribe is in flight on
	// a background goroutine, so the session loop never blocks on endpoint I/O
	// and a second open cannot race the same id.
	accessPending map[uint64]bool
	clipboard     *clipboardHistory
	clipboardRead func() (string, error)

	endpoints     *endpoint.Manager
	endpointUnsub func()
	registryPaths []string
	ownsEndpoints bool
	// daemonSourceFn overrides the endpoint manager's source projection. It is
	// nil in production (daemonSources reads the manager); tests inject a
	// deterministic daemon inventory without a live connection.
	daemonSourceFn func() []*pb.Source
	logf           func(format string, args ...any)
	noticeMu       sync.Mutex
	notices        [][2]string
}

// NewHost wires the session, the PTY handler and the confirmation gate.
func NewHost(opts Options) *Host {
	if strings.TrimSpace(opts.ViewID) == "" {
		opts.ViewID = viewID
	}
	if opts.Cols <= 0 {
		opts.Cols = 80
	}
	if opts.Rows <= 0 {
		opts.Rows = 24
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
	clipboardPath := opts.ClipboardHistoryPath
	if strings.TrimSpace(clipboardPath) == "" {
		clipboardPath = defaultClipboardHistoryPath()
	}
	clipboardRead := opts.ClipboardRead
	if clipboardRead == nil {
		clipboardRead = readSystemClipboard
	}
	ownsEndpoints := false
	registryPaths := append([]string(nil), opts.RegistryPaths...)
	if len(registryPaths) == 0 {
		registryPaths = endpoint.RegistryPathsFromEnv()
	}
	if opts.Endpoints == nil {
		opts.Endpoints = endpoint.NewManager(endpoint.Options{
			RegistryPaths: registryPaths, Routes: opts.Routes,
			DialTimeout: opts.DialTimeout, CallTimeout: opts.CallTimeout,
		})
		ownsEndpoints = true
	}
	h := &Host{
		opts:          opts,
		cols:          opts.Cols,
		rows:          opts.Rows,
		tracked:       map[string]*trackedTerminal{},
		sizes:         map[string][2]int{},
		pendingSize:   map[string][2]int{},
		resizing:      map[string]bool{},
		components:    map[string]*terminal.Component{},
		accessStreams: map[uint64]*accessStreamBridge{},
		accessEvents:  map[uint64]*accessEventBridge{},
		confirmCh:     make(chan struct{}, 1),
		wake:          make(chan struct{}, 1),
		urgentWake:    make(chan struct{}, 1),
		endpoints:     opts.Endpoints,
		registryPaths: registryPaths,
		ownsEndpoints: ownsEndpoints,
		logf:          opts.Logf,
		clipboard:     newClipboardHistory(clipboardPath, opts.ClipboardHistoryMaxItems),
		clipboardRead: clipboardRead,
	}
	if h.opts.Dev != nil {
		h.opts.Dev.onNotice = h.queueNotice
	}
	// Endpoint health/inventory changes must repaint immediately (the picker
	// lists them), so drive the frame loop from the change event.
	h.endpointUnsub = opts.Endpoints.Subscribe(func() { h.signalWake() }, h.queueNotice)
	if ownsEndpoints && opts.LoadSharedRegistry {
		h.registerSharedEndpoints()
		h.registerLocalAccessEndpoint()
	}
	opts.NewPTY = func(cfg pty.Config) pty.PTY {
		if cfg.Endpoint != "" {
			if kind, ok := opts.Endpoints.Kind(cfg.Endpoint); ok && kind == endpoint.KindDaemon {
				return opts.Endpoints.NewRemotePTY(cfg)
			}
		}
		return localPTY(cfg)
	}
	h.handler = opts.Handler
	if h.handler == nil {
		h.handler = runtime.NewTerminalHandler(runtime.TerminalOptions{
			Cols:          opts.Cols,
			Rows:          opts.Rows,
			OwnerID:       opts.ViewID,
			OwnerLeaseTTL: 15 * time.Second,
			NewPTY:        opts.NewPTY,
			Clipboard:     h.writeClipboardAndRemember,
		})
	}
	// Subscribe even for a private handler. This makes the callback correct
	// when a caller supplies the same handler to multiple Host views.
	h.outputUnsub = h.handler.SubscribeOutput(func(string) {
		h.withSession(func(s *runtime.Session) { s.MarkOutput() })
		if h.interactionRecent(time.Now()) {
			h.signalWakeImmediate()
		} else {
			h.signalWake()
		}
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

// signalWake asks the frame loop to repaint now. It never blocks: a burst
// coalesces into one pending wake, so a program commit or PTY output burst
// costs at most one extra frame.
func (h *Host) signalWake() {
	select {
	case h.wake <- struct{}{}:
	default:
	}
}

// signalWakeImmediate bypasses the background frame budget for a committed
// interactive view or PTY output caused by a recent user event.
func (h *Host) signalWakeImmediate() {
	select {
	case h.urgentWake <- struct{}{}:
	default:
	}
}

func (h *Host) interactionRecent(now time.Time) bool {
	stamp := h.lastInteraction.Load()
	if stamp == 0 {
		return false
	}
	return now.Sub(time.Unix(0, stamp)) <= interactionUrgencyWindow
}

// Run owns the terminal until quit or a fatal error: alt screen, raw frame
// loop, program supervision and confirmation overlay.
func (h *Host) Run() error {
	if h.outputUnsub != nil {
		defer h.outputUnsub()
	}
	if h.endpointUnsub != nil {
		defer h.endpointUnsub()
	}
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

	// A buffered incomplete Esc/Alt sequence needs a one-shot deadline; arm a
	// timer only while bytes are pending and stop it otherwise, so the loop
	// stays event-driven (no polling ticker).
	escTimer := time.NewTimer(escTimeout)
	if !escTimer.Stop() {
		<-escTimer.C
	}
	armEsc := func() {
		if !escTimer.Stop() {
			select {
			case <-escTimer.C:
			default:
			}
		}
		if parser.Pending() > 0 {
			// A 1-2 byte lone Esc/Alt is resolved by the timeout; longer
			// incomplete CSI/OSC sequences wait for more input, so do not
			// re-arm and risk a poll loop.
			if parser.Pending() <= 2 {
				escTimer.Reset(escTimeout)
			}
		}
	}
	ownerTicker := time.NewTicker(5 * time.Second)
	defer ownerTicker.Stop()
	// The observer-count refresh runs on a low-frequency background ticker, so
	// it never blocks the frame loop. It is suppressed in a `go test` binary:
	// tests drive Run with a manager that has no live daemon, so a periodic
	// list would be pure noise (and could race a test's own list expectations).
	sourceTicker := time.NewTicker(sourceRefreshInterval)
	defer sourceTicker.Stop()
	if testBinary() {
		sourceTicker.Stop()
	}

	// Keep the newest session state and spend at most one short budget per
	// burst. This gives PTY output and VIEW commits a bounded cadence without
	// waiting for terminal synchronization packets or replaying every
	// intermediate scroll position.
	var frameTimer *time.Timer
	var frameC <-chan time.Time
	pendingFrame := false
	lastFlush := time.Time{}
	stopFrameTimer := func() {
		if frameTimer == nil {
			frameC = nil
			return
		}
		if !frameTimer.Stop() {
			select {
			case <-frameTimer.C:
			default:
			}
		}
		frameTimer = nil
		frameC = nil
	}
	defer stopFrameTimer()
	armFrameTimer := func(now time.Time) {
		if !pendingFrame || frameC != nil {
			return
		}
		delay := frameFlushDelay(lastFlush, now)
		if delay <= 0 {
			return
		}
		frameTimer = time.NewTimer(delay)
		frameC = frameTimer.C
	}
	flushPending := func(now time.Time) {
		if !pendingFrame {
			return
		}
		if h.anyTerminalSynchronizedOutput() {
			// Keep pendingFrame and the timer state intact. The terminal pump
			// emits a wake after ?2026l, which is the only safe commit point for
			// a full-screen child's redraw.
			return
		}
		stopFrameTimer()
		h.publishSources()
		h.flushNotices()
		if !h.flush() {
			return
		}
		pendingFrame = false
		lastFlush = now
	}

	for {
		flushNow := false
		renderEvent := true
		select {
		case chunk, ok := <-inputCh:
			if !ok {
				flushPending(time.Now())
				return nil
			}
			if wheelDebugEnabled() && chunkHasEsc(chunk) {
				h.logf("tui2 input raw=%q", chunk)
			}
			renderEvent = false
			for _, ev := range parser.Feed(chunk) {
				renderEvent = h.handleInput(ev) || renderEvent
			}
			armEsc()
		case <-escTimer.C:
			renderEvent = false
			for _, ev := range parser.Flush(escTimeout) {
				renderEvent = h.handleInput(ev) || renderEvent
			}
			// A longer incomplete sequence may still be buffered; keep the
			// deadline armed instead of stalling until the next input.
			armEsc()
		case <-ownerTicker.C:
			go h.renewOwners()
		case <-sourceTicker.C:
			go h.refreshSources()
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
		case <-h.urgentWake:
			flushNow = true
		case <-h.wake:
		case <-frameC:
			// The timer is a fixed deadline from the previous flush, so a
			// continuous burst cannot keep extending it indefinitely.
			frameTimer = nil
			frameC = nil
		}
		if !renderEvent {
			// A PTY passthrough event does not change the host layout. The child
			// will wake the frame loop after its output is parsed; rendering here
			// would sample a possible DEC 2026 clear/partial redraw.
			continue
		}
		pendingFrame = true
		now := time.Now()
		if flushNow || frameFlushDelay(lastFlush, now) == 0 {
			flushPending(now)
		} else {
			armFrameTimer(now)
		}
		if h.isQuitting() {
			return nil
		}
	}
}

// frameFlushDelay returns how long a pending frame should wait before it can
// be rendered. A zero lastFlush means the host was idle and should paint
// immediately; otherwise a burst shares the fixed frame budget.
func frameFlushDelay(lastFlush, now time.Time) time.Duration {
	if lastFlush.IsZero() {
		return 0
	}
	if elapsed := now.Sub(lastFlush); elapsed < frameCoalesceWindow {
		return frameCoalesceWindow - elapsed
	}
	return 0
}

// testBinary reports whether this process is a compiled `go test` binary. It
// detects the test flag registered by the testing package without importing it
// (importing testing in a main package would register test flags in production
// builds). Used to keep polling timers out of unit tests.
func testBinary() bool { return flag.Lookup("test.v") != nil || flag.Lookup("test.timeout") != nil }

// refreshSources re-lists the daemon inventory so attachment_count tracks
// other clients' attach/detach in real time. The manager's change callback
// (subscribed in NewHost) republishes the sources snapshot and wakes the frame
// loop, and an unchanged count emits nothing, so this stays off the hot path.
func (h *Host) refreshSources() {
	if h.endpoints == nil {
		return
	}
	h.sourceMu.Lock()
	if h.sourceBusy {
		h.sourceMu.Unlock()
		return
	}
	h.sourceBusy = true
	h.sourceMu.Unlock()
	defer func() {
		h.sourceMu.Lock()
		h.sourceBusy = false
		h.sourceMu.Unlock()
	}()
	// Refresh bounds each list call with the manager's own CallTimeout and
	// skips offline endpoints, so a plain context is enough here; it runs on a
	// background goroutine to keep the frame loop free.
	h.endpoints.Refresh(context.Background())
}

// renewOwners is the protocol's owner heartbeat. It is deliberately a
// low-frequency timer rather than part of the render loop: a quiet terminal
// still keeps its lease, while a dead host stops renewing and another view can
// take over after the TTL.
func (h *Host) renewOwners() {
	h.ownerMu.Lock()
	if h.ownerBusy {
		h.ownerMu.Unlock()
		return
	}
	h.ownerBusy = true
	h.ownerMu.Unlock()
	defer func() {
		h.ownerMu.Lock()
		h.ownerBusy = false
		h.ownerMu.Unlock()
	}()
	h.mu.Lock()
	terms := make([]*runtime.Terminal, 0, len(h.tracked))
	for _, tracked := range h.tracked {
		if tracked != nil && tracked.term != nil {
			terms = append(terms, tracked.term)
		}
	}
	h.mu.Unlock()
	for _, term := range terms {
		if !term.RefreshResizeOwner() {
			term.RenewOwner(h.opts.ViewID)
		}
	}
	h.signalWake()
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
		ViewID:      h.opts.ViewID,
		Epoch:       epoch,
		Cols:        cols,
		Rows:        rows,
		Components:  []string{"terminal"},
		Limits:      h.opts.Limits,
		Handler:     h.gate,
		OnStream:    h.handleAccessStreamFrame,
		OnView:      h.signalWakeImmediate,
		OutputQueue: 256,
		InputSink:   h.handler,
		MouseTracking: func(id string) bool {
			term, ok := h.handler.TerminalBySource(id)
			return ok && term.Modes().MouseTracking()
		},
		HistoryActive: func(id, view string) bool {
			term, ok := h.handler.TerminalBySource(id)
			return ok && term.HistoryRoutingActive(view)
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
	session := h.session
	h.proc = nil
	h.mu.Unlock()
	if proc != nil {
		// Close the child pipes first. The bounded session writer may be blocked
		// in a kernel pipe write when a layout program stops reading; stopping
		// the process releases that write so Session.Close can join the worker.
		proc.Stop()
	}
	if session != nil {
		session.Close()
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
	h.closeAllAccessStreams()
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
// boxes (PROTOCOL §6.7). It returns whether the host layout may need a frame;
// pure PTY passthrough is deliberately render-free.
func (h *Host) handleInput(ev keys.Event) bool {
	session := h.currentSession()
	if session == nil {
		return false
	}
	h.lastInteraction.Store(time.Now().UnixNano())
	if session.CoreOverlayOpen() {
		if ev.Kind == keys.KindKey {
			switch keys.Name(ev) {
			case "enter":
				h.confirmYes()
			case "esc", "ctrl-c":
				h.confirmNo()
			}
		}
		return true
	}
	if ev.Kind == keys.KindMouse || ev.Kind == keys.KindWheel {
		h.preparePointer(session, &ev)
	}
	dst, err := h.route(ev)
	if err != nil {
		h.logf("tui2: input: %v", err)
		h.queueNotice("warning", fmt.Sprintf("input: %v", err))
	}
	if ev.Kind == keys.KindWheel && wheelDebugEnabled() {
		focusID := ""
		isTerm, track, hist, inputs := false, false, false, ""
		if focus := session.Focus(); focus != nil {
			focusID = focus.ID
			isTerm = focus.IsTerminal
			track = focus.MouseTracking
			hist = focus.HistoryActive
			inputs = strings.Join(focus.Input, ",")
		}
		h.logf("tui2 wheel node=%q delta=%d hit=%v dst=%s focus=%q term=%v track=%v history=%v input=[%s]",
			ev.Node, ev.Delta, ev.HitFocused, dst, focusID, isTerm, track, hist, inputs)
	}
	if dst == runtime.DestinationHost && ev.Kind == keys.KindKey && keys.Name(ev) == runtime.KeyCtrlQ {
		h.askHostQuit()
	}
	if ev.Kind == keys.KindMouse && ev.Action == keys.ActionRelease {
		session.ReleaseCapture()
		h.terminalMouseDown = false
	}
	return err != nil || dst != runtime.DestinationPTY
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
	focus := session.Focus()
	focusedNode := ""
	if focus != nil {
		focusedNode = focus.NodeID
	}
	ev.HitFocused = hit != "" && hit == focusedNode
	if ev.Kind == keys.KindMouse && ev.Action == keys.ActionPress && ev.HitFocused && focus != nil && focus.IsTerminal {
		h.terminalMouseDown = true
	}
	if ev.Kind == keys.KindMouse && h.terminalMouseDown && focus != nil && focus.IsTerminal && (ev.Action == keys.ActionDrag || ev.Action == keys.ActionRelease) {
		// A terminal owns its mouse gesture once the press was accepted. Keep
		// sending the clamped content coordinate when the pointer leaves the
		// panel, then clear the gesture on release.
		ev.HitFocused = true
	}
	if focus != nil && focus.IsTerminal && (ev.Kind == keys.KindMouse || ev.Kind == keys.KindWheel) {
		if x, y, ok := h.terminalPointerPosition(session, frame, focus, ev.X, ev.Y); ok {
			ev.PTYX, ev.PTYY = x, y
		}
	}
	if ev.Kind != keys.KindWheel && ev.Action == keys.ActionPress && hit != "" && isCaptureBox(session, hit) {
		session.Capture(hit)
	}
}

// terminalPointerPosition translates outer 1-based TUI coordinates to the
// focused terminal's 1-based PTY content grid. The component's declared
// chrome inset is the source of truth; coordinates are clamped because a
// captured drag may leave the panel.
func (h *Host) terminalPointerPosition(session *runtime.Session, frame kernel.Frame, focus *runtime.Focus, x, y int) (int, int, bool) {
	if focus == nil || focus.NodeID == "" || !focus.IsTerminal {
		return 0, 0, false
	}
	rect, ok := frame.Rect(focus.NodeID)
	if !ok || rect.Empty() {
		return 0, 0, false
	}
	inset := terminal.DefaultInset
	if declared, ok := terminal.InsetFromProps(session.BoxProps(focus.NodeID)); ok {
		inset = declared
	}
	if rect.Width < 2*inset+1 || rect.Height < 2*inset+1 {
		inset = 0
	}
	width, height := rect.Width-2*inset, rect.Height-2*inset
	if width <= 0 || height <= 0 {
		return 0, 0, false
	}
	localX := x - (rect.X + inset)
	localY := y - (rect.Y + inset)
	if localX < 1 {
		localX = 1
	}
	if localY < 1 {
		localY = 1
	}
	if localX > width {
		localX = width
	}
	if localY > height {
		localY = height
	}
	return localX, localY, true
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

// ownerFromProps parses the program's chrome.owner declaration from a box's
// content props, the same way the host reads chrome.inset. A truthy value
// designates the box as its source's resize owner. The host never infers this
// from focus: ownership is a manual user action (the legacy panel.take_owner),
// so clicking a follower pane must not silently transfer the size.
func ownerFromProps(props map[string]string) bool {
	switch strings.ToLower(strings.TrimSpace(props["chrome.owner"])) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// terminalOwners maps each terminal source to the box that owns its single PTY
// size, following the legacy render.TerminalViewBinding owner/follower split:
// a terminal has exactly one authoritative size driven by one view (here: one
// pane's box). The program's chrome.owner=1 box wins when exactly one box
// declares it; otherwise the focused box is the fallback, and without focus the
// first box in stable declaration order owns it. Followers render the same
// shared component but must never resize the PTY, so focusing a follower never
// steals the size back from a declared owner.
func (h *Host) terminalOwners(view *pb.View) map[string]string {
	owners := map[string]string{}
	declared := map[string]string{}
	declaredCount := map[string]int{}
	focused := map[string]string{}
	var walk func(*pb.Box)
	walk = func(b *pb.Box) {
		if b == nil {
			return
		}
		if sourceID := b.GetContent().GetSelf(); sourceID != "" {
			if _, ok := owners[sourceID]; !ok {
				// First box in declaration order is the fallback owner.
				owners[sourceID] = b.GetId()
			}
			if ownerFromProps(b.GetContent().GetProps()) {
				declared[sourceID] = b.GetId()
				declaredCount[sourceID]++
			}
			if b.GetFocused() {
				focused[sourceID] = b.GetId()
			}
		}
		for _, child := range b.GetChildren() {
			walk(child)
		}
	}
	walk(view.GetRoot())
	for sourceID := range owners {
		switch {
		case declaredCount[sourceID] == 1:
			// The program designated exactly one owner; focus never overrides it.
			owners[sourceID] = declared[sourceID]
		case focused[sourceID] != "":
			owners[sourceID] = focused[sourceID]
		}
	}
	return owners
}

// placements renders every bound terminal source for the current view and
// keeps its PTY window size aligned with the solved content rect. Only the
// per-source owner box resizes the PTY; every box still gets a placement so
// the render output is unchanged. Every (source, view) pair seen this frame is
// recorded so components of closed panes can be pruned afterwards.
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
	owners := h.terminalOwners(view)
	var out []runtime.Placement
	live := map[string]bool{}
	var walk func(*pb.Box)
	walk = func(b *pb.Box) {
		if b == nil {
			return
		}
		sourceID := b.GetContent().GetSelf()
		if sourceID != "" {
			if term, ok := h.handler.TerminalBySource(sourceID); ok {
				if rect, ok := frame.Rect(b.GetId()); ok {
					live[componentKey(sourceID, b.GetId())] = true
					resize := owners[sourceID] == b.GetId()
					out = append(out, h.placementResize(session, sourceID, term, b, rect, resize))
				}
			}
		}
		for _, child := range b.GetChildren() {
			walk(child)
		}
	}
	walk(view.GetRoot())
	h.pruneComponents(live)
	return out
}

// pruneComponents drops cached components for panes that no longer appear in
// the current view, so a closed pane does not leak its terminal component.
func (h *Host) pruneComponents(live map[string]bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for key := range h.components {
		if !live[key] {
			delete(h.components, key)
		}
	}
}

// placement renders one terminal box as if it were the source's resize owner.
// It is the single-box convenience used outside placements; callers that walk
// several boxes for one source must use placementResize so only the owner
// reflows the PTY.
func (h *Host) placement(session *runtime.Session, sourceID string, term *runtime.Terminal, box *pb.Box, rect kernel.Rect) runtime.Placement {
	return h.placementResize(session, sourceID, term, box, rect, true)
}

// placementResize pushes the declarative state of one terminal box into its
// component and renders it. The box's program-declared props (content.props)
// are passed through untouched: the component interprets them, the host never
// does. resize is true only for the per-source owner box: follower panes still
// render the shared component at their own rect but must not reflow the single
// PTY, matching the legacy owner/follower resize semantics.
func (h *Host) placementResize(session *runtime.Session, sourceID string, term *runtime.Terminal, box *pb.Box, rect kernel.Rect, resize bool) runtime.Placement {
	// A component is cached per (source, view): the box id is the pane's view
	// key, and two panes sharing one terminal source must render independent
	// frozen viewports instead of fighting over one component/screen.
	view := box.GetId()
	key := componentKey(sourceID, view)
	component := h.components[key]
	if component == nil {
		component = terminal.New(nil, nil)
		h.components[key] = component
	}
	offset := term.Offset(view)
	source := h.sourceByID(sourceID)
	props := terminal.Props{
		Title:        sourceTitle(source, term),
		Focused:      box.GetFocused(),
		Scrolled:     offset > 0,
		ScrollOffset: offset,
		Exited:       term.Exited(),
		Chrome:       session.BoxProps(box.GetId()),
	}
	props.Dimmed = terminal.DimmedFromProps(props.Chrome)
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
	component.SetScreen(term.VisibleScreen(view))
	if box.GetFocused() {
		term.RenewOwner(h.opts.ViewID)
	}
	if resize {
		inset := component.Inset(rect.Width, rect.Height)
		h.resizePTY(term, rect, inset)
	}
	placement := term.Placement(component, rect, box.GetFocused())
	// The program may shift the terminal screen inside the content area
	// (content.offset): the component draws the screen at that offset, so the
	// PTY cursor must move with it. The PTY box itself stays full-bleed, so the
	// owner's winsize is unaffected.
	inset := component.Inset(rect.Width, rect.Height)
	shiftCursorForContentOffset(&placement, props.Chrome, inset, rect.Width, rect.Height)
	return placement
}

// shiftCursorForContentOffset moves a placement's PTY cursor by the program's
// content.offset and hides it when the shifted cell leaves the content area. A
// zero offset is a no-op, so the default render/cursor is untouched.
func shiftCursorForContentOffset(p *runtime.Placement, chrome map[string]string, inset, width, height int) {
	dx, dy, shifted := terminal.FramingFromProps(chrome)
	if !shifted || !p.CursorVisible {
		return
	}
	p.CursorX += dx
	p.CursorY += dy
	if p.CursorX < inset || p.CursorY < inset ||
		p.CursorX >= width-inset || p.CursorY >= height-inset {
		p.CursorVisible = false
	}
}

// componentKey names one cached terminal component. The view (box id) is part
// of the key so sibling panes on one source each keep their own frozen screen.
func componentKey(sourceID, view string) string { return sourceID + "\x00" + view }

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
	h.applyResize(term, sourceID, cols, rows)
}

// applyResize dispatches the PTY size change off the frame loop and coalesces
// bursts per terminal (latest-only): a drag generates one resize per frame, but
// only one blocking round-trip runs at a time and the last requested size wins.
// This mirrors main's queued resize coalescing and keeps drag interactive.
func (h *Host) applyResize(term *runtime.Terminal, sourceID string, cols, rows int) {
	size := [2]int{cols, rows}
	h.mu.Lock()
	if h.sizes == nil {
		h.sizes = map[string][2]int{}
	}
	if h.pendingSize == nil {
		h.pendingSize = map[string][2]int{}
	}
	if h.resizing == nil {
		h.resizing = map[string]bool{}
	}
	if h.resizing[sourceID] {
		h.pendingSize[sourceID] = size
		h.mu.Unlock()
		return
	}
	h.resizing[sourceID] = true
	h.sizes[sourceID] = size
	h.mu.Unlock()
	go func() {
		if err := term.Resize(cols, rows); err != nil {
			h.logf("tui2: resize %s to %dx%d: %v", sourceID, cols, rows, err)
		}
		for {
			h.mu.Lock()
			next, ok := h.pendingSize[sourceID]
			if !ok {
				delete(h.resizing, sourceID)
				h.mu.Unlock()
				return
			}
			delete(h.pendingSize, sourceID)
			h.sizes[sourceID] = next
			h.mu.Unlock()
			if err := term.Resize(next[0], next[1]); err != nil {
				h.logf("tui2: resize %s to %dx%d: %v", sourceID, next[0], next[1], err)
			}
		}
	}()
}

func (h *Host) flush() bool {
	session := h.currentSession()
	if session == nil {
		return false
	}
	placements := h.placements()
	for _, placement := range placements {
		if placement.SynchronizedOutput {
			// DEC 2026 is a transaction boundary owned by the child TUI. Keep
			// the last committed host frame until the parser sees ?2026l;
			// input and unrelated wakeups must not publish the clear/half-frame.
			return false
		}
	}
	if data := session.FrameBytes(placements, nil); len(data) > 0 {
		if err := h.writeOutBytes(data); err != nil {
			h.quit()
		}
	}
	return true
}

// anyTerminalSynchronizedOutput reports whether a visible PTY child has an
// open DEC 2026 redraw transaction. It walks the current view directly so the
// guard does not call placements (which also performs resize bookkeeping).
func (h *Host) anyTerminalSynchronizedOutput() bool {
	session := h.currentSession()
	if session == nil || h.handler == nil {
		return false
	}
	view := session.View()
	if view == nil {
		return false
	}
	var walk func(*pb.Box) bool
	walk = func(box *pb.Box) bool {
		if box == nil {
			return false
		}
		if sourceID := box.GetContent().GetSelf(); sourceID != "" {
			if term, ok := h.handler.TerminalBySource(sourceID); ok {
				// A child that exits while a malformed/incomplete batch is open
				// must still publish its last parsed screen; otherwise the sync
				// guard would hold the panel forever after the pump reports exit.
				if !term.Exited() && term.Modes().SynchronizedOutput {
					return true
				}
			}
		}
		for _, child := range box.GetChildren() {
			if walk(child) {
				return true
			}
		}
		return false
	}
	return walk(view.GetRoot())
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

func (h *Host) writeClipboardAndRemember(text string) error {
	if err := h.writeClipboard(text); err != nil {
		return err
	}
	if h.clipboard != nil {
		if _, err := h.clipboard.Add(text); err != nil {
			h.logf("tui2: persist clipboard history: %v", err)
		}
	}
	return nil
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

// daemonSources is the daemon inventory projection used by snapshotSources. A
// test-injected daemonSourceFn wins so snapshotSources stays exercisable
// without a live endpoint; otherwise it asks the endpoint manager.
func (h *Host) daemonSources() []*pb.Source {
	if h.daemonSourceFn != nil {
		return h.daemonSourceFn()
	}
	if h.endpoints == nil {
		return nil
	}
	return h.endpoints.Sources()
}

func (h *Host) snapshotSources() ([]*pb.Source, bool) {
	h.mu.Lock()
	items := make([]*pb.Source, 0, len(h.tracked))
	tracked := make(map[string]bool, len(h.tracked))
	previousTitles := make(map[string]string, len(h.sources))
	for _, source := range h.sources {
		if source != nil {
			previousTitles[source.GetId()] = source.GetTitle()
		}
	}
	// Daemon inventory carries tags/size for pool terminals; merge it into the
	// locally attached source so the picker can filter by tag.
	daemonTags := map[string]map[string]string{}
	daemonTitles := map[string]string{}
	endpointLabels := map[string]string{}
	daemonSize := map[string][2]int{}
	daemonLastOutput := map[string]int64{}
	// daemonAttachments is the daemon's authoritative observer count per
	// terminal. The program's pane badge adds its own local pane count and
	// subtracts this client's single shared attachment; republishing it here is
	// what makes other clients' attach/detach visible in real time.
	daemonAttachments := map[string]int32{}
	for _, source := range h.daemonSources() {
		if title := strings.TrimSpace(source.GetTitle()); title != "" {
			daemonTitles[source.GetId()] = title
		}
		if len(source.GetTags()) > 0 {
			daemonTags[source.GetId()] = source.GetTags()
		}
		if source.GetEndpointLabel() != "" {
			endpointLabels[source.GetEndpoint()] = source.GetEndpointLabel()
		}
		if source.GetCols() > 0 && source.GetRows() > 0 {
			daemonSize[source.GetId()] = [2]int{int(source.GetCols()), int(source.GetRows())}
		}
		if source.GetLastOutputMs() > 0 {
			daemonLastOutput[source.GetId()] = source.GetLastOutputMs()
		}
		if count := source.GetAttachmentCount(); count > 0 {
			daemonAttachments[source.GetId()] = count
		}
	}
	for sourceID, trackedTerm := range h.tracked {
		endpointName, id := splitSourceID(sourceID)
		owner, epoch := trackedTerm.term.ResizeOwner()
		exitCode := trackedTerm.term.ExitCode()
		if exitCode < 0 {
			exitCode = 0
		}
		cols, rows, _ := trackedTerm.term.Size()
		// Prefer the daemon inventory grid when the local attachment still
		// reports the pre-attach default (the daemon is the resize authority).
		if size, ok := daemonSize[sourceID]; ok {
			if cols <= 0 || rows <= 0 {
				cols, rows = size[0], size[1]
			}
		}
		lastOutput := trackedTerm.term.LastOutput()
		lastOutputMs := int64(0)
		if !lastOutput.IsZero() {
			lastOutputMs = lastOutput.UnixMilli()
		}
		if lastOutputMs == 0 {
			lastOutputMs = daemonLastOutput[sourceID]
		}
		health := "ok"
		if kind, ok := h.endpoints.Kind(endpointName); ok && kind == endpoint.KindDaemon {
			health = h.endpoints.Health(endpointName)
		}
		endpointLabel := h.endpoints.Label(endpointName)
		if endpointLabel == "" {
			endpointLabel = endpointLabels[endpointName]
		}
		items = append(items, &pb.Source{
			Id:              sourceID,
			Kind:            "terminal",
			Title:           trackedSourceTitle(sourceID, trackedTerm, daemonTitles, previousTitles),
			Endpoint:        endpointName,
			TerminalId:      id,
			Attached:        true,
			Exited:          trackedTerm.term.Exited(),
			ExitCode:        int32(exitCode),
			Health:          health,
			ResizeOwner:     owner,
			OwnerEpoch:      epoch,
			Cols:            int32(cols),
			Rows:            int32(rows),
			Tags:            daemonTags[sourceID],
			EndpointLabel:   endpointLabel,
			LastOutputMs:    lastOutputMs,
			AttachmentCount: daemonAttachments[sourceID],
		})
		tracked[sourceID] = true
	}
	h.mu.Unlock()
	// Daemon-listed terminals that have no local attachment yet are added
	// unattached so the picker can bind them position-transparently.
	for _, source := range h.daemonSources() {
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
				old.GetTitle() != item.GetTitle() ||
				old.GetExitCode() != item.GetExitCode() || old.GetResizeOwner() != item.GetResizeOwner() || old.GetOwnerEpoch() != item.GetOwnerEpoch() ||
				old.GetHealth() != item.GetHealth() || old.GetAttached() != item.GetAttached() ||
				old.GetCols() != item.GetCols() || old.GetRows() != item.GetRows() ||
				old.GetEndpointLabel() != item.GetEndpointLabel() || old.GetLastOutputMs() != item.GetLastOutputMs() ||
				old.GetAttachmentCount() != item.GetAttachmentCount() {
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

// attachedSourceTitle keeps the daemon's user-facing terminal name when a
// local attachment is projected into the host source list. The previous
// title covers a short inventory gap during reconnect; the runtime terminal
// id is only a last-resort fallback for sources with no name metadata.
func attachedSourceTitle(sourceID string, daemonTitles, previousTitles map[string]string, fallback string) string {
	if title := strings.TrimSpace(daemonTitles[sourceID]); title != "" {
		return title
	}
	if title := strings.TrimSpace(previousTitles[sourceID]); title != "" {
		return title
	}
	return fallback
}

func (h *Host) trackSource(sourceID string, term *runtime.Terminal) {
	h.trackSourceWithOptions(sourceID, term, false, "")
}

func (h *Host) trackSourceWithOptions(sourceID string, term *runtime.Terminal, ephemeral bool, title string) {
	h.mu.Lock()
	if strings.TrimSpace(title) == "" {
		if previous := h.tracked[sourceID]; previous != nil {
			title = previous.title
		}
	}
	h.tracked[sourceID] = &trackedTerminal{term: term, ephemeral: ephemeral, title: strings.TrimSpace(title)}
	h.mu.Unlock()
}

func trackedSourceFallback(tracked *trackedTerminal) string {
	if tracked != nil && strings.TrimSpace(tracked.title) != "" {
		return tracked.title
	}
	if tracked != nil && tracked.term != nil {
		return tracked.term.ID()
	}
	return "terminal"
}

func trackedSourceTitle(sourceID string, tracked *trackedTerminal, daemonTitles, previousTitles map[string]string) string {
	if tracked != nil && strings.TrimSpace(tracked.title) != "" {
		return tracked.title
	}
	return attachedSourceTitle(sourceID, daemonTitles, previousTitles, trackedSourceFallback(tracked))
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
	// Both the reserved Ctrl-Q shortcut and a program-issued system.quit ask
	// the same question, so the user sees one dialog for "quit" regardless of
	// who initiated it.
	message := "Quit tui2?"
	if !confirm.programQuit && confirm.req.Method.Name != "" {
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
		if confirm.cleanupOwned {
			h.cleanupOwnedTerminals()
		}
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
	// Wake the frame loop so a quit set from another goroutine is observed
	// without waiting for input.
	h.signalWake()
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
	// Flush the notice on the next frame rather than waiting for input.
	h.signalWake()
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
	if req.Method.Name == "clipboard.read" {
		text, err := h.clipboardRead()
		if err != nil {
			return runtime.Outcome{Error: err.Error()}, false
		}
		if h.clipboard != nil {
			_, _ = h.clipboard.Add(text)
		}
		return runtime.Outcome{OK: true, Data: &pb.MethodData{Text: text}}, false
	}
	if req.Method.Name == "clipboard.history.delete" {
		if err := h.clipboard.Delete(req.Params.GetClipboardId()); err != nil {
			return runtime.Outcome{Error: err.Error()}, false
		}
		return runtime.Outcome{OK: true}, false
	}
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

// cleanupOwnedTerminals implements system.quit{cleanup_owned:true}. Only
// terminals explicitly created as ephemeral by this layout program are
// touched; ordinary attached terminals keep the historical detach semantics.
// Cleanup is best-effort and bounded because it runs on the final quit path.
func (h *Host) cleanupOwnedTerminals() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h.mu.Lock()
	owned := make([]struct {
		source string
		term   *runtime.Terminal
	}, 0)
	for sourceID, tracked := range h.tracked {
		if tracked != nil && tracked.ephemeral && tracked.term != nil {
			owned = append(owned, struct {
				source string
				term   *runtime.Terminal
			}{source: sourceID, term: tracked.term})
		}
	}
	h.mu.Unlock()
	for _, item := range owned {
		endpointName, id := splitSourceID(item.source)
		if kind, ok := h.endpoints.Kind(endpointName); ok && kind == endpoint.KindDaemon {
			_ = h.endpoints.Kill(ctx, endpointName, id)
			_ = h.endpoints.Remove(ctx, endpointName, id)
		} else {
			_ = item.term.Close()
			for !item.term.Exited() && ctx.Err() == nil {
				time.Sleep(5 * time.Millisecond)
			}
			_, _ = h.handler.Handle(runtime.Request{Method: runtime.Method{Name: "terminal.remove"}, Params: &pb.MethodParams{Endpoint: endpointName, Id: id}})
		}
		h.mu.Lock()
		delete(h.tracked, item.source)
		h.mu.Unlock()
	}
}

// gateHandler is the authorization point before the real TerminalHandler:
// terminal.create and terminal.restart get real implementations, destructive
// methods go through the core overlay, and everything else is delegated.
type gateHandler struct {
	host  *Host
	inner *runtime.TerminalHandler
	seqMu sync.Mutex
	seq   int
}

// nextSeq allocates a local terminal id; it is safe because local create runs
// on the session loop while daemon work runs on background workers.
func (g *gateHandler) nextSeq() int {
	g.seqMu.Lock()
	defer g.seqMu.Unlock()
	g.seq++
	return g.seq
}

func (g *gateHandler) Handle(req runtime.Request) (runtime.Outcome, bool) {
	switch req.Method.Name {
	case "history.window", "terminal.history.window", "terminal.search", "terminal.scroll", "terminal.scrollEnd", "terminal.copy":
		return g.terminalHistory(req)
	case "terminal.rename":
		if g.host.isDaemonEndpoint(req.Params.GetEndpoint()) {
			return g.asyncDaemon(req, g.renameDaemon)
		}
		return g.renameLocal(req)
	case "terminal.detach":
		if g.host.isDaemonEndpoint(req.Params.GetEndpoint()) {
			return g.asyncDaemon(req, g.detachTerminal)
		}
		return g.detachTerminal(req)
	case "terminal.reconnect":
		return g.reconnectTerminal(req)
	case "clipboard.history.list":
		return runtime.Outcome{OK: true, Data: &pb.MethodData{Rows: g.host.clipboard.Rows()}}, false
	case "clipboard.paste":
		return g.pasteClipboard(req)
	case "endpoint.list":
		return g.listEndpoints()
	case "endpoint.test":
		if g.host.endpoints.Health(req.Params.GetEndpoint()) == endpoint.HealthUnknown {
			return runtime.Outcome{Error: "endpoint not registered"}, false
		}
		return runtime.Outcome{OK: true}, false
	case "endpoint.reconnect":
		return g.asyncDaemon(req, func(req runtime.Request) (runtime.Outcome, bool) {
			if err := g.host.endpoints.Reconnect(req.Params.GetEndpoint()); err != nil {
				return runtime.Outcome{Error: err.Error()}, false
			}
			return runtime.Outcome{OK: true}, false
		})
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
			return g.asyncDaemon(req, g.createDaemon)
		}
		return g.create(req)
	case "terminal.restart":
		if g.host.isDaemonEndpoint(req.Params.GetEndpoint()) {
			return g.asyncDaemon(req, g.restartDaemon)
		}
		return g.restart(req)
	case "access.call":
		return g.accessCall(req)
	case "access.stream.open":
		return g.openAccessStream(req)
	case "access.stream.subscribe":
		return g.openAccessSubscription(req)
	case "terminal.attach":
		if err := g.host.registerEndpoint(req.Params); err != nil {
			return runtime.Outcome{Error: err.Error()}, false
		}
		if g.host.isDaemonEndpoint(req.Params.GetEndpoint()) {
			return g.asyncDaemon(req, g.attachDaemon)
		}
		outcome, pending := g.inner.Handle(req)
		if outcome.OK && !pending {
			g.track(req.Params.GetEndpoint(), req.Params.GetId(), false, req.Params.GetTitle())
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

func (g *gateHandler) renameLocal(req runtime.Request) (runtime.Outcome, bool) {
	endpointName, id := req.Params.GetEndpoint(), req.Params.GetId()
	if _, ok := g.inner.TerminalBySource(runtime.SourceID(endpointName, id)); !ok {
		return runtime.Outcome{Error: runtime.ErrNoTerminal.Error()}, false
	}
	sourceID := runtime.SourceID(endpointName, id)
	g.host.mu.Lock()
	tracked := g.host.tracked[sourceID]
	if tracked == nil {
		g.host.mu.Unlock()
		return runtime.Outcome{Error: runtime.ErrNoTerminal.Error()}, false
	}
	tracked.title = strings.TrimSpace(req.Params.GetTitle())
	g.host.mu.Unlock()
	g.host.publishSources()
	return runtime.Outcome{OK: true}, false
}

func (g *gateHandler) renameDaemon(req runtime.Request) (runtime.Outcome, bool) {
	endpointName, id := req.Params.GetEndpoint(), req.Params.GetId()
	command := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalSetMetadata{
		TerminalSetMetadata: &apipb.TerminalSetMetadataCommand{
			Terminal: &apipb.TerminalRef{EndpointId: endpointName, TerminalId: id},
			Name:     strings.TrimSpace(req.Params.GetTitle()),
		},
	}}
	if _, err := g.host.endpoints.Execute(context.Background(), endpointName, command); err != nil {
		return runtime.Outcome{Error: err.Error()}, false
	}
	g.host.mu.Lock()
	if tracked := g.host.tracked[runtime.SourceID(endpointName, id)]; tracked != nil {
		tracked.title = strings.TrimSpace(req.Params.GetTitle())
	}
	g.host.mu.Unlock()
	g.host.publishSources()
	return runtime.Outcome{OK: true}, false
}

func (g *gateHandler) detachTerminal(req runtime.Request) (runtime.Outcome, bool) {
	endpointName, id := req.Params.GetEndpoint(), req.Params.GetId()
	if err := g.inner.Detach(endpointName, id); err != nil {
		return runtime.Outcome{Error: err.Error()}, false
	}
	g.host.mu.Lock()
	delete(g.host.tracked, runtime.SourceID(endpointName, id))
	g.host.mu.Unlock()
	g.host.publishSources()
	return runtime.Outcome{OK: true}, false
}

func (g *gateHandler) reconnectTerminal(req runtime.Request) (runtime.Outcome, bool) {
	endpointName, id := req.Params.GetEndpoint(), req.Params.GetId()
	if err := g.inner.Reconnect(endpointName, id); err != nil {
		return runtime.Outcome{Error: err.Error()}, false
	}
	g.track(endpointName, id, false, "")
	g.host.publishSources()
	return runtime.Outcome{OK: true}, false
}

func (g *gateHandler) pasteClipboard(req runtime.Request) (runtime.Outcome, bool) {
	text := ""
	if id := strings.TrimSpace(req.Params.GetClipboardId()); id != "" {
		entry, ok := g.host.clipboard.Get(id)
		if !ok {
			return runtime.Outcome{Error: "clipboard entry not found"}, false
		}
		text = entry.Text
	} else {
		var err error
		text, err = g.host.clipboardRead()
		if err != nil {
			return runtime.Outcome{Error: err.Error()}, false
		}
		if g.host.clipboard != nil {
			_, _ = g.host.clipboard.Add(text)
		}
	}
	sourceID := runtime.SourceID(req.Params.GetEndpoint(), req.Params.GetId())
	bracket := g.inner.BracketPaste(sourceID)
	limits := g.host.opts.Limits
	if limits.MaxPasteBytes == 0 {
		limits = runtime.DefaultLimits()
	}
	for _, chunk := range keys.ChunkPaste(text, int(limits.MaxPasteBytes), bracket, nil) {
		if err := g.inner.WriteInput(sourceID, chunk.Bytes); err != nil {
			return runtime.Outcome{Error: err.Error()}, false
		}
	}
	return runtime.Outcome{OK: true}, false
}

func (g *gateHandler) listEndpoints() (runtime.Outcome, bool) {
	rows := make([]string, 0)
	for _, cfg := range g.host.endpoints.Configs() {
		row, _ := json.Marshal(map[string]string{
			"name": cfg.Name, "label": g.host.endpoints.Label(cfg.Name),
			"kind": cfg.KindName(), "health": g.host.endpoints.Health(cfg.Name),
		})
		rows = append(rows, string(row))
	}
	return runtime.Outcome{OK: true, Data: &pb.MethodData{Rows: rows}}, false
}

// asyncDaemon runs a daemon create/restart/attach off the protocol reader: the
// dial, create and attach are network round trips to a remote endpoint, so
// running them inline would stall the frame loop and every other pane while a
// slow or offline endpoint times out (the same reason access.call is async).
// The request is registered in flight and answered by a background worker,
// fenced by the request's epoch.
func (g *gateHandler) asyncDaemon(req runtime.Request, work func(runtime.Request) (runtime.Outcome, bool)) (runtime.Outcome, bool) {
	session := g.host.currentSession()
	if session == nil {
		return runtime.Outcome{Error: "no active session"}, false
	}
	go func() {
		outcome, pending := work(req)
		if pending {
			// Daemon work is self-contained; a worker that reports pending has
			// nothing to complete, so treat it as accepted.
			outcome = runtime.Outcome{OK: true}
		}
		g.completeAsyncEpoch(session, req.Epoch, req.RequestID, outcome)
	}()
	return runtime.Outcome{}, true
}

// attachDaemon attaches an existing daemon terminal (network round trip).
func (g *gateHandler) attachDaemon(req runtime.Request) (runtime.Outcome, bool) {
	outcome, pending := g.inner.Handle(req)
	if outcome.OK && !pending {
		g.track(req.Params.GetEndpoint(), req.Params.GetId(), false, req.Params.GetTitle())
	}
	return outcome, pending
}

func (g *gateHandler) create(req runtime.Request) (runtime.Outcome, bool) {
	seq := g.nextSeq()
	endpoint := req.Params.GetEndpoint()
	if endpoint == "" {
		endpoint = "local"
	}
	id := "term-" + strconv.Itoa(seq)
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
		OwnerID:   req.OwnerID,
		Method:    runtime.Method{Name: "terminal.attach"},
		Params:    params,
	})
	if !outcome.OK || pending {
		return outcome, pending
	}
	g.track(endpoint, id, req.Params.GetEphemeral(), req.Params.GetTitle())
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
		Tags:       req.Params.GetTags(),
		Size:       &apipb.TerminalSize{Cols: uint32(cols), Rows: uint32(rows)},
	}
	info, err := g.host.endpoints.Create(context.Background(), endpointName, spec)
	if err != nil {
		return runtime.Outcome{Error: err.Error()}, false
	}
	id := info.GetRef().GetTerminalId()
	if id == "" {
		return runtime.Outcome{Error: "terminal pool returned an empty terminal id"}, false
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
		OwnerID:   req.OwnerID,
		Method:    runtime.Method{Name: "terminal.attach"},
		Params:    params,
	})
	if !outcome.OK || pending {
		return outcome, pending
	}
	g.track(endpointName, id, req.Params.GetEphemeral(), req.Params.GetTitle())
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
	term, ok := g.inner.TerminalAt(endpointName, id)
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
		OwnerID:   req.OwnerID,
		Method:    runtime.Method{Name: "terminal.attach"},
		Params: &pb.MethodParams{
			Endpoint: endpointName, Id: id, Argv: argv,
			Kind: req.Params.GetKind(), Socket: req.Params.GetSocket(), Address: req.Params.GetAddress(), ConnectMode: req.Params.GetConnectMode(),
		},
	})
	if !outcome.OK || pending {
		return outcome, pending
	}
	g.track(endpointName, id, false, req.Params.GetTitle())
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
	term, ok := g.inner.TerminalAt(endpoint, id)
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
		OwnerID:   req.OwnerID,
		Method:    runtime.Method{Name: "terminal.attach"},
		Params:    &pb.MethodParams{Endpoint: endpoint, Id: id, Argv: argv},
	})
	if !outcome.OK || pending {
		return outcome, pending
	}
	g.track(endpoint, id, false, req.Params.GetTitle())
	return runtime.Outcome{OK: true}, false
}

func (g *gateHandler) track(endpoint, id string, ephemeral bool, title string) {
	if id == "" {
		return
	}
	if endpoint == "" {
		endpoint = "local"
	}
	term, ok := g.inner.TerminalAt(endpoint, id)
	if !ok {
		return
	}
	g.host.trackSourceWithOptions(runtime.SourceID(endpoint, id), term, ephemeral, title)
}

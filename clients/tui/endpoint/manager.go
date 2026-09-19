package endpoint

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	pb "github.com/anytty/anytty/proto/ui/protobuf"

	"github.com/anytty/anytty/proto/access/apipb"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	gproto "google.golang.org/protobuf/proto"
)

// Default manager timings. They are short because local-unix dials are local.
const (
	DefaultDialTimeout = 3 * time.Second
	DefaultCallTimeout = 5 * time.Second
	DefaultBackoffMin  = 250 * time.Millisecond
	DefaultBackoffMax  = 5 * time.Second
)

// Options configures a Manager. Zero values fall back to the defaults.
type Options struct {
	DialTimeout time.Duration
	CallTimeout time.Duration
	BackoffMin  time.Duration
	BackoffMax  time.Duration
	// Dial overrides the session dialer (tests inject a fake); the default
	// dials every daemon endpoint through the shared client layer
	// (client/runtime + client/adapter/protocol).
	Dial SessionDialer
	// Routes is the shared route policy this manager may dial (nil = the
	// default policy: local-unix plus credential-gated ssh). Direct and
	// managed WebRTC only enter the plan when explicitly enabled; see
	// policy.go and tui2/docs/REMOTE.zh-CN.md §3.1.
	Routes []clientendpoint.RouteKind
	// RegistryPath overrides the shared endpoints.yaml path with a single
	// explicit registry (tests use this to isolate); empty uses RegistryPaths.
	RegistryPath string
	// RegistryPaths is the ordered read list of CLI-owned endpoints.yaml
	// registries (explicit TUI2_ENDPOINTS/-endpoints first, then the default
	// path). Explicit files win on duplicate endpoint names. Empty resolves
	// TUI2_ENDPOINTS + client/endpoint.DefaultPath().
	RegistryPaths []string
	// OnChange is invoked after every health or terminal-list change so the
	// host can republish its sources snapshot.
	OnChange func()
	// OnNotice is invoked on health transitions with a level ("info",
	// "warning") and a human-readable message. It must not block.
	OnNotice func(level, message string)
}

func (o Options) withDefaults() Options {
	if o.DialTimeout <= 0 {
		o.DialTimeout = DefaultDialTimeout
	}
	if o.CallTimeout <= 0 {
		o.CallTimeout = DefaultCallTimeout
	}
	if o.BackoffMin <= 0 {
		o.BackoffMin = DefaultBackoffMin
	}
	if o.BackoffMax < o.BackoffMin {
		o.BackoffMax = DefaultBackoffMax
	}
	if o.Dial == nil {
		registryPaths := append([]string(nil), o.RegistryPaths...)
		if len(registryPaths) == 0 && strings.TrimSpace(o.RegistryPath) != "" {
			registryPaths = []string{strings.TrimSpace(o.RegistryPath)}
		}
		routes := normalizeRouteKinds(o.Routes)
		o.Dial = func(ctx context.Context, cfg Config) (sessionConn, error) {
			return dialSharedSessionWithRoutes(ctx, cfg, registryPaths, routes)
		}
	}
	return o
}

// Status is one daemon terminal projection for the host sources snapshot.
type Status struct {
	Endpoint string
	ID       string
	Name     string
	Exited   bool
	ExitCode int
	Attached bool
	Health   string
	Cols     int
	Rows     int
}

// Manager owns every configured endpoint: one connection supervisor per
// endpoint with reconnect backoff, the daemon terminal inventory and the
// attached RemotePTYs.
type Manager struct {
	opts Options

	mu        sync.Mutex
	endpoints map[string]*endpointState
	closed    bool
	onChange  func()
	onNotice  func(level, message string)

	baseCtx context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewManager creates an empty manager. No endpoint connects until Register.
func NewManager(opts Options) *Manager {
	opts = opts.withDefaults()
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		opts:      opts,
		endpoints: map[string]*endpointState{},
		onChange:  opts.OnChange,
		onNotice:  opts.OnNotice,
		baseCtx:   ctx,
		cancel:    cancel,
	}
}

// SetOnChange installs the change callback (safe after construction).
func (m *Manager) SetOnChange(fn func()) {
	m.mu.Lock()
	m.onChange = fn
	m.mu.Unlock()
}

// SetOnNotice installs the notice callback (safe after construction).
func (m *Manager) SetOnNotice(fn func(level, message string)) {
	m.mu.Lock()
	m.onNotice = fn
	m.mu.Unlock()
}

func (m *Manager) notifyChange() {
	m.mu.Lock()
	fn := m.onChange
	m.mu.Unlock()
	if fn != nil {
		fn()
	}
}

func (m *Manager) notifyNotice(level, message string) {
	m.mu.Lock()
	fn := m.onNotice
	m.mu.Unlock()
	if fn != nil {
		fn(level, message)
	}
}

type endpointState struct {
	cfg         Config
	health      string
	lastErr     string
	client      sessionConn
	terminals   map[string]*apipb.TerminalInfo
	attachments map[string]*RemotePTY
	running     bool
	cancel      context.CancelFunc
}

// Register stores or updates one endpoint and starts its connection
// supervisor. Registering the same configuration twice is a no-op.
func (m *Manager) Register(cfg Config) error {
	if m == nil {
		return errors.New("endpoint: nil manager")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return ErrEndpointClosed
	}
	if existing, ok := m.endpoints[cfg.Name]; ok {
		if reflect.DeepEqual(existing.cfg, cfg) {
			m.mu.Unlock()
			return nil
		}
		m.mu.Unlock()
		return fmt.Errorf("endpoint %q is already registered with a different configuration", cfg.Name)
	}
	state := &endpointState{cfg: cfg, health: HealthConnecting, terminals: map[string]*apipb.TerminalInfo{}, attachments: map[string]*RemotePTY{}}
	m.endpoints[cfg.Name] = state
	ctx, cancel := context.WithCancel(m.baseCtx)
	state.cancel = cancel
	state.running = true
	m.wg.Add(1)
	m.mu.Unlock()
	go m.runEndpoint(ctx, state)
	return nil
}

// Config returns the registered configuration of an endpoint.
func (m *Manager) Config(name string) (Config, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.endpoints[name]
	if !ok {
		return Config{}, false
	}
	return state.cfg, true
}

// Kind reports the registered kind of an endpoint.
func (m *Manager) Kind(name string) (string, bool) {
	cfg, ok := m.Config(name)
	if !ok {
		return "", false
	}
	return cfg.KindName(), true
}

// Health returns the endpoint health ("unknown" for unregistered endpoints).
func (m *Manager) Health(name string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if state, ok := m.endpoints[name]; ok {
		return state.health
	}
	return HealthUnknown
}

// Snapshot returns every known daemon terminal across all endpoints. The host
// merges it with its locally attached terminals into the sources snapshot.
func (m *Manager) Snapshot() []Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Status, 0)
	for name, state := range m.endpoints {
		if state.cfg.KindName() != KindDaemon {
			continue
		}
		for id, info := range state.terminals {
			status := Status{
				Endpoint: name,
				ID:       id,
				Name:     info.GetName(),
				Exited:   info.GetState() == apipb.TerminalState_TERMINAL_STATE_EXITED,
				Health:   state.health,
				Cols:     int(info.GetSize().GetCols()),
				Rows:     int(info.GetSize().GetRows()),
			}
			if exitCode := info.GetExitCode(); info.ExitCode != nil {
				status.ExitCode = int(exitCode)
			}
			if p, ok := state.attachments[id]; ok && p != nil && !p.Closed() {
				status.Attached = true
			}
			out = append(out, status)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Endpoint != out[j].Endpoint {
			return out[i].Endpoint < out[j].Endpoint
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Close shuts every connection down and releases attached terminals. Remote
// PTYs stop reading; daemon terminals keep running (detach semantics).
func (m *Manager) Close() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	states := make([]*endpointState, 0, len(m.endpoints))
	for _, state := range m.endpoints {
		states = append(states, state)
	}
	m.mu.Unlock()
	m.cancel()
	for _, state := range states {
		if state.cancel != nil {
			state.cancel()
		}
	}
	m.wg.Wait()
	for _, state := range states {
		m.mu.Lock()
		ptyList := make([]*RemotePTY, 0, len(state.attachments))
		for _, p := range state.attachments {
			ptyList = append(ptyList, p)
		}
		m.mu.Unlock()
		for _, p := range ptyList {
			// Shutdown never kills: the daemon terminal must survive a TUI
			// exit, exactly like a local PTY survives an unbind.
			p.shutdown()
		}
	}
	return nil
}

// runEndpoint is the reconnect supervisor: dial, handshake, refresh the
// terminal list, rebind attachments, then wait for the connection to end and
// repeat with exponential backoff. Health changes emit one notice per
// transition.
func (m *Manager) runEndpoint(ctx context.Context, state *endpointState) {
	defer m.wg.Done()
	backoff := m.opts.BackoffMin
	first := true
	for {
		if ctx.Err() != nil {
			return
		}
		if !first {
			m.setHealth(state, HealthConnecting, "")
		}
		first = false
		dialCtx, cancel := context.WithTimeout(ctx, m.opts.DialTimeout)
		conn, err := m.opts.Dial(dialCtx, state.cfg)
		cancel()
		if err != nil {
			m.setHealth(state, HealthOffline, err.Error())
			if !sleepCtx(ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff, m.opts.BackoffMax)
			continue
		}
		backoff = m.opts.BackoffMin
		m.setClient(state, conn)
		m.refreshList(ctx, state)
		// Re-attach before publishing health=ok: "connected" must mean the
		// subscriptions are rebound, otherwise the first keystroke after the
		// notice could hit a dead attachment.
		m.rebindAll(ctx, state)
		m.setHealth(state, HealthOK, "")
		select {
		case <-ctx.Done():
			_ = conn.close()
			return
		case <-conn.Done():
		}
		m.clearClient(state, conn)
		m.setHealth(state, HealthOffline, conn.Err().Error())
		if !sleepCtx(ctx, backoff) {
			return
		}
		backoff = nextBackoff(backoff, m.opts.BackoffMax)
	}
}

func nextBackoff(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		next = max
	}
	return next
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// setHealth updates health and emits a notice on transitions.
func (m *Manager) setHealth(state *endpointState, health, detail string) {
	m.mu.Lock()
	if state.health == health {
		m.mu.Unlock()
		return
	}
	previous := state.health
	state.health = health
	if detail != "" {
		state.lastErr = detail
	}
	name := state.cfg.Name
	m.mu.Unlock()
	m.notifyChange()
	switch health {
	case HealthOffline:
		message := fmt.Sprintf("endpoint %s offline", name)
		if detail != "" {
			message += ": " + detail
		}
		m.notifyNotice("warning", message)
	case HealthOK:
		if previous == HealthOffline || previous == HealthConnecting {
			m.notifyNotice("info", fmt.Sprintf("endpoint %s connected", name))
		}
	}
}

func (m *Manager) setClient(state *endpointState, conn sessionConn) {
	m.mu.Lock()
	state.client = conn
	m.mu.Unlock()
}

func (m *Manager) clearClient(state *endpointState, conn sessionConn) {
	m.mu.Lock()
	if state.client == conn {
		state.client = nil
	}
	m.mu.Unlock()
}

// refreshList updates the cached daemon terminal inventory.
func (m *Manager) refreshList(ctx context.Context, state *endpointState) {
	conn := m.currentClient(state)
	if conn == nil {
		return
	}
	callCtx, cancel := context.WithTimeout(ctx, m.opts.CallTimeout)
	items, err := conn.list(callCtx)
	cancel()
	if err != nil {
		return
	}
	index := make(map[string]*apipb.TerminalInfo, len(items))
	for _, item := range items {
		if id := item.GetRef().GetTerminalId(); id != "" {
			index[id] = item
		}
	}
	m.mu.Lock()
	state.terminals = index
	m.mu.Unlock()
	m.notifyChange()
	if state.cfg.KindName() == KindDaemon {
		m.pruneDetached(state, index)
	}
}

// pruneDetached closes remote PTYs whose daemon terminal no longer exists.
func (m *Manager) pruneDetached(state *endpointState, index map[string]*apipb.TerminalInfo) {
	m.mu.Lock()
	var stale []*RemotePTY
	for id, p := range state.attachments {
		if _, ok := index[id]; !ok {
			stale = append(stale, p)
		}
	}
	m.mu.Unlock()
	for _, p := range stale {
		p.fail(fmt.Errorf("endpoint %q: terminal %s was removed in the terminal pool", state.cfg.Name, p.id))
	}
}

func (m *Manager) currentClient(state *endpointState) sessionConn {
	m.mu.Lock()
	defer m.mu.Unlock()
	return state.client
}

// rebindAll re-attaches every remote PTY after a reconnect.
func (m *Manager) rebindAll(ctx context.Context, state *endpointState) {
	m.mu.Lock()
	ptyList := make([]*RemotePTY, 0, len(state.attachments))
	for _, p := range state.attachments {
		ptyList = append(ptyList, p)
	}
	m.mu.Unlock()
	for _, p := range ptyList {
		m.rebindOne(ctx, state, p)
	}
}

// rebindOne opens a fresh attachment for one RemotePTY on the current client
// and hands it to the PTY pump.
func (m *Manager) rebindOne(ctx context.Context, state *endpointState, p *RemotePTY) {
	conn := m.currentClient(state)
	if conn == nil || p.Closed() {
		return
	}
	att, err := m.openAttachment(ctx, conn, state, p)
	if err != nil {
		if isNotFound(err) {
			p.fail(fmt.Errorf("endpoint %q: terminal %s no longer exists", state.cfg.Name, p.id))
			m.unregisterAttachment(state, p)
		}
		return
	}
	p.setPending(att)
}

// rebindNow is the RemotePTY-initiated resync (stream sync lost while the
// connection is still alive).
func (m *Manager) rebindNow(p *RemotePTY) {
	m.mu.Lock()
	state := m.endpoints[p.cfg.Name]
	m.mu.Unlock()
	if state == nil {
		return
	}
	ctx, cancel := context.WithTimeout(m.baseCtx, m.opts.CallTimeout)
	defer cancel()
	m.rebindOne(ctx, state, p)
}

// openAttachment attaches one terminal and seeds the authoritative screen
// snapshot before the live stream starts (ENDPOINTS §4: snapshot is truth,
// live deltas never replay history).
func (m *Manager) openAttachment(ctx context.Context, conn sessionConn, state *endpointState, p *RemotePTY) (*attachment, error) {
	cols, rows := p.window()
	att, err := conn.openAttachment(ctx, p.id, p.surface, p.view, cols, rows)
	if err != nil {
		return nil, err
	}
	if screen, err := conn.liveScreen(ctx, p.id, 0); err == nil {
		p.seedSnapshot(screen)
	}
	if err := conn.startStream(ctx, att, m.opts.CallTimeout); err != nil {
		_ = conn.detach(context.Background(), att)
		return nil, err
	}
	return att, nil
}

// attachRemote performs the initial attach for a RemotePTY (Start).
func (m *Manager) attachRemote(ctx context.Context, p *RemotePTY) (*attachment, error) {
	m.mu.Lock()
	state := m.endpoints[p.cfg.Name]
	deadline := time.Now().Add(m.opts.DialTimeout + m.opts.CallTimeout)
	m.mu.Unlock()
	if state == nil {
		return nil, fmt.Errorf("endpoint %q is not registered", p.cfg.Name)
	}
	var conn sessionConn
	for {
		if conn = m.currentClient(state); conn != nil {
			break
		}
		if time.Now().After(deadline) {
			return nil, m.endpointUnavailableError(p.cfg.Name, nil)
		}
		select {
		case <-ctx.Done():
			return nil, m.endpointUnavailableError(p.cfg.Name, ctx.Err())
		case <-time.After(25 * time.Millisecond):
		}
	}
	m.registerAttachment(state, p)
	att, err := m.openAttachment(ctx, conn, state, p)
	if err != nil {
		m.unregisterAttachment(state, p)
		return nil, err
	}
	return att, nil
}

func (m *Manager) registerAttachment(state *endpointState, p *RemotePTY) {
	m.mu.Lock()
	state.attachments[p.id] = p
	m.mu.Unlock()
}

func (m *Manager) unregisterAttachment(state *endpointState, p *RemotePTY) {
	m.mu.Lock()
	if state.attachments[p.id] == p {
		delete(state.attachments, p.id)
	}
	m.mu.Unlock()
}

// unregisterRemote removes one RemotePTY from its endpoint registry.
func (m *Manager) unregisterRemote(p *RemotePTY) {
	m.mu.Lock()
	state := m.endpoints[p.cfg.Name]
	m.mu.Unlock()
	if state != nil {
		m.unregisterAttachment(state, p)
	}
}

// detachRemote is the Close-path unregister.
func (m *Manager) detachRemote(p *RemotePTY) {
	m.unregisterRemote(p)
}

// Create makes a new terminal on a daemon endpoint and returns its info.
func (m *Manager) Create(ctx context.Context, name string, spec *apipb.TerminalCreateSpec) (*apipb.TerminalInfo, error) {
	m.mu.Lock()
	state := m.endpoints[name]
	m.mu.Unlock()
	if state == nil {
		return nil, fmt.Errorf("endpoint %q is not registered", name)
	}
	if state.cfg.KindName() != KindDaemon {
		return nil, fmt.Errorf("endpoint %q is not a daemon endpoint", name)
	}
	if err := state.cfg.UnsupportedModeError(); err != nil {
		return nil, err
	}
	waitCtx, waitCancel := context.WithTimeout(ctx, m.opts.DialTimeout+m.opts.CallTimeout)
	conn, err := m.waitClient(waitCtx, state)
	waitCancel()
	if err != nil {
		return nil, err
	}
	if len(spec.GetCommand()) == 0 {
		defaultsCtx, cancel := context.WithTimeout(ctx, m.opts.CallTimeout)
		defaults, defaultsErr := conn.defaults(defaultsCtx)
		cancel()
		if defaultsErr == nil {
			spec = cloneSpec(spec)
			spec.Command = append([]string(nil), defaults.GetDefaultCommand()...)
			if spec.GetCwd() == "" {
				spec.Cwd = defaults.GetDefaultCwd()
			}
		}
	}
	if len(spec.GetCommand()) == 0 {
		spec = cloneSpec(spec)
		spec.Command = []string{"/bin/sh"}
	}
	if spec.GetTerminalId() == "" {
		spec = cloneSpec(spec)
		spec.TerminalId = newTerminalID()
	}
	callCtx, cancel := context.WithTimeout(ctx, m.opts.CallTimeout)
	info, err := conn.create(callCtx, spec)
	cancel()
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	state.terminals[info.GetRef().GetTerminalId()] = info
	m.mu.Unlock()
	m.changed()
	return info, nil
}

// Kill terminates a daemon terminal (idempotent for exited terminals).
func (m *Manager) Kill(ctx context.Context, name, id string) error {
	conn, err := m.connectionFor(name)
	if err != nil {
		return err
	}
	callCtx, cancel := context.WithTimeout(ctx, m.opts.CallTimeout)
	defer cancel()
	if err := conn.kill(callCtx, id); err != nil {
		return err
	}
	m.markExited(name, id, -1)
	return nil
}

// markExited updates the cached inventory projection after a terminal dies.
func (m *Manager) markExited(name, id string, code int) {
	m.mu.Lock()
	if state := m.endpoints[name]; state != nil {
		if info := state.terminals[id]; info != nil {
			updated := gproto.Clone(info).(*apipb.TerminalInfo)
			updated.State = apipb.TerminalState_TERMINAL_STATE_EXITED
			if code >= 0 {
				exitCode := int32(code)
				updated.ExitCode = &exitCode
			}
			state.terminals[id] = updated
		}
	}
	m.mu.Unlock()
	m.notifyChange()
}

// Remove deletes an exited daemon terminal.
func (m *Manager) Remove(ctx context.Context, name, id string) error {
	conn, err := m.connectionFor(name)
	if err != nil {
		return err
	}
	callCtx, cancel := context.WithTimeout(ctx, m.opts.CallTimeout)
	defer cancel()
	if err := conn.remove(callCtx, id); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.endpoints[name].terminals, id)
	m.mu.Unlock()
	m.changed()
	return nil
}

// Restart asks the daemon to restart a terminal from its saved process
// specification (SCENARIOS §3 restart semantics). The caller re-attaches
// under the same terminal id afterwards.
func (m *Manager) Restart(ctx context.Context, name, id string) error {
	conn, err := m.connectionFor(name)
	if err != nil {
		return err
	}
	callCtx, cancel := context.WithTimeout(ctx, m.opts.CallTimeout)
	defer cancel()
	if err := conn.restart(callCtx, id); err != nil {
		return err
	}
	m.mu.Lock()
	if state := m.endpoints[name]; state != nil {
		if info := state.terminals[id]; info != nil {
			updated := gproto.Clone(info).(*apipb.TerminalInfo)
			updated.State = apipb.TerminalState_TERMINAL_STATE_RUNNING
			updated.ExitCode = nil
			state.terminals[id] = updated
		}
	}
	m.mu.Unlock()
	m.notifyChange()
	return nil
}

func (m *Manager) connectionFor(name string) (sessionConn, error) {
	m.mu.Lock()
	state := m.endpoints[name]
	m.mu.Unlock()
	if state == nil {
		return nil, fmt.Errorf("endpoint %q is not registered", name)
	}
	conn := m.currentClient(state)
	if conn == nil {
		return nil, fmt.Errorf("endpoint %q is %s", name, m.Health(name))
	}
	return conn, nil
}

// waitClient blocks until the endpoint has a live connection or ctx expires.
func (m *Manager) waitClient(ctx context.Context, state *endpointState) (sessionConn, error) {
	for {
		if conn := m.currentClient(state); conn != nil {
			return conn, nil
		}
		m.mu.Lock()
		closed := m.closed
		m.mu.Unlock()
		if closed {
			return nil, ErrEndpointClosed
		}
		select {
		case <-ctx.Done():
			return nil, m.endpointUnavailableError(state.cfg.Name, ctx.Err())
		case <-time.After(25 * time.Millisecond):
		}
	}
}

// endpointUnavailableError is the readable offline surface for create/attach:
// health plus the supervisor's last dial error, never a bare ctx deadline.
func (m *Manager) endpointUnavailableError(name string, cause error) error {
	m.mu.Lock()
	lastErr := ""
	if state := m.endpoints[name]; state != nil {
		lastErr = state.lastErr
	}
	m.mu.Unlock()
	message := fmt.Sprintf("endpoint %q is %s", name, m.Health(name))
	if lastErr != "" {
		message += ": " + lastErr
	}
	if cause == nil {
		return errors.New(message)
	}
	return fmt.Errorf("%s: %w", message, cause)
}

func (m *Manager) changed() {
	m.notifyChange()
}

// Source merges daemon statuses into protocol sources for the host. Attached
// terminals are normally also tracked locally; daemon-only entries are added
// with attached=false so the picker can list and bind them. Every daemon
// endpoint without a known terminal also yields one kind="endpoint"
// placeholder source carrying health, so a paired-but-empty or offline
// backend stays visible in the picker with a readable state instead of
// disappearing.
func (m *Manager) Sources() []*pb.Source {
	statuses := m.Snapshot()
	out := make([]*pb.Source, 0, len(statuses))
	for _, status := range statuses {
		title := status.Name
		if title == "" {
			title = status.ID
		}
		exitCode := status.ExitCode
		if exitCode < 0 {
			exitCode = 0
		}
		out = append(out, &pb.Source{
			Id:         "terminal:" + status.Endpoint + ":" + status.ID,
			Kind:       "terminal",
			Title:      title,
			Endpoint:   status.Endpoint,
			TerminalId: status.ID,
			Attached:   status.Attached,
			Exited:     status.Exited,
			ExitCode:   int32(exitCode),
			Health:     status.Health,
		})
	}
	m.mu.Lock()
	names := make([]string, 0, len(m.endpoints))
	for name, state := range m.endpoints {
		if state.cfg.KindName() == KindDaemon && len(state.terminals) == 0 {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		state := m.endpoints[name]
		title := state.cfg.Label
		if title == "" {
			title = name
		}
		out = append(out, &pb.Source{
			Id:       "endpoint:" + name,
			Kind:     "endpoint",
			Title:    title,
			Endpoint: name,
			Health:   state.health,
		})
	}
	m.mu.Unlock()
	return out
}

func cloneSpec(spec *apipb.TerminalCreateSpec) *apipb.TerminalCreateSpec {
	clone := gproto.Clone(spec).(*apipb.TerminalCreateSpec)
	clone.Command = append([]string(nil), spec.GetCommand()...)
	clone.Env = append([]string(nil), spec.GetEnv()...)
	return clone
}

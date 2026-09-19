package endpoint

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
)

// errTerminalClosed marks a clean daemon stream close (process exited).
var errTerminalClosed = errors.New("endpoint: terminal stream closed")

// RemotePTY is the pty.PTY adapter of one daemon terminal. It satisfies the
// same interface as a local PTY, so runtime.Terminal, the ANSI parser, the
// component pipeline, input routing, kill and restart are position
// transparent (ENDPOINTS.zh-CN.md §3).
type RemotePTY struct {
	mgr     *Manager
	cfg     Config
	id      string
	argv    []string
	surface string
	view    string

	mu       sync.Mutex
	cond     *sync.Cond
	started  bool
	closed   bool
	exited   bool
	exitCode int
	readErr  error
	buf      []byte
	pending  *attachment
	att      *attachment
	cols     int
	rows     int
	closing  bool
	notify   chan struct{}
	closeCh  chan struct{}
	doneCh   chan struct{}
}

// NewRemotePTY builds the unstarted daemon PTY for one attach call. The
// endpoint must already be registered.
func (m *Manager) NewRemotePTY(cfg pty.Config) *RemotePTY {
	endpointCfg, _ := m.Config(cfg.Endpoint)
	id := cfg.ID
	if id == "" {
		id = newTerminalID()
	}
	p := &RemotePTY{
		mgr:     m,
		cfg:     endpointCfg,
		id:      id,
		argv:    append([]string(nil), cfg.Argv...),
		surface: "tui2-host",
		view:    fmt.Sprintf("tui2:%s:%s", cfg.Endpoint, id),
		cols:    cfg.Cols,
		rows:    cfg.Rows,
		notify:  make(chan struct{}, 1),
		closeCh: make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
	p.cond = sync.NewCond(&p.mu)
	return p
}

// ID returns the daemon terminal id.
func (p *RemotePTY) ID() string { return p.id }

// Endpoint returns the endpoint name.
func (p *RemotePTY) Endpoint() string { return p.cfg.Name }

// Closed reports whether the local view has been closed.
func (p *RemotePTY) Closed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closed
}

// Start connects (or waits for the reconnect loop), attaches the terminal,
// seeds the authoritative screen snapshot and begins live output.
func (p *RemotePTY) Start() error {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return pty.ErrStarted
	}
	if p.closed {
		p.mu.Unlock()
		return pty.ErrClosed
	}
	if p.cfg.KindName() != KindDaemon {
		p.mu.Unlock()
		return fmt.Errorf("endpoint %q is not a daemon endpoint", p.cfg.Name)
	}
	if err := p.cfg.UnsupportedModeError(); err != nil {
		p.mu.Unlock()
		return err
	}
	p.started = true
	p.mu.Unlock()

	ctx, cancel := context.WithTimeout(p.mgr.baseCtx, p.mgr.opts.DialTimeout+p.mgr.opts.CallTimeout)
	defer cancel()
	att, err := p.mgr.attachRemote(ctx, p)
	if err != nil {
		p.mu.Lock()
		p.started = false
		p.mu.Unlock()
		return err
	}
	p.setPending(att)
	go p.run()
	return nil
}

// Read returns daemon PTY output. It blocks while the terminal is quiet and
// keeps blocking across reconnects; only exit or Close end the stream.
func (p *RemotePTY) Read(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(p.buf) == 0 && p.readErr == nil && !p.closed {
		p.cond.Wait()
	}
	if len(p.buf) > 0 {
		n := copy(b, p.buf)
		p.buf = p.buf[n:]
		return n, nil
	}
	if p.closed {
		return 0, pty.ErrClosed
	}
	return 0, p.readErr
}

// Write forwards bytes to the daemon terminal as input.
func (p *RemotePTY) Write(b []byte) (int, error) {
	p.mu.Lock()
	closed := p.closed
	att := p.att
	p.mu.Unlock()
	if closed {
		return 0, pty.ErrClosed
	}
	if att == nil {
		return 0, fmt.Errorf("endpoint %q: terminal %s is offline", p.cfg.Name, p.id)
	}
	ctx, cancel := context.WithTimeout(p.mgr.baseCtx, p.mgr.opts.CallTimeout)
	defer cancel()
	if err := att.client.input(ctx, att, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

// Resize applies the local viewport to the daemon terminal through the
// resize-owner CAS. A lost ownership race yields a readable error.
func (p *RemotePTY) Resize(cols, rows int) error {
	if cols <= 0 || rows <= 0 {
		return pty.ErrInvalidSize
	}
	p.mu.Lock()
	p.cols, p.rows = cols, rows
	closed := p.closed
	att := p.att
	p.mu.Unlock()
	if closed {
		return pty.ErrClosed
	}
	if att == nil {
		return fmt.Errorf("endpoint %q: terminal %s is offline", p.cfg.Name, p.id)
	}
	take := att.epoch == 0
	ctx, cancel := context.WithTimeout(p.mgr.baseCtx, p.mgr.opts.CallTimeout)
	defer cancel()
	result, err := att.client.resize(ctx, att, cols, rows, take, att.epoch)
	if err != nil {
		return fmt.Errorf("endpoint %q: resize %s: %w", p.cfg.Name, p.id, err)
	}
	if result.GetResized() {
		return nil
	}
	if control := result.GetResizeControl(); control != nil {
		if size := control.GetOwnership().GetSize(); size != nil && int(size.GetCols()) == cols && int(size.GetRows()) == rows {
			return nil
		}
		return fmt.Errorf("endpoint %q: resize %s denied: %s (owner view %q epoch %d)",
			p.cfg.Name, p.id, resizeReason(control), control.GetOwnership().GetOwnerViewId(), control.GetOwnership().GetEpoch())
	}
	return nil
}

func resizeReason(control *apipb.ResizeControl) string {
	switch control.GetReason() {
	case apipb.ResizeControlReason_RESIZE_CONTROL_REASON_SIZE_LOCKED:
		return "terminal size is locked by another attachment"
	case apipb.ResizeControlReason_RESIZE_CONTROL_REASON_OBSERVER:
		return "attachment is an observer"
	case apipb.ResizeControlReason_RESIZE_CONTROL_REASON_FOLLOWER:
		return "attachment is a resize follower"
	default:
		if !control.GetCanResize() {
			return "attachment does not own resize"
		}
		return "resize was not applied"
	}
}

// Size reports the last viewport sent to the daemon (or the attach size).
func (p *RemotePTY) Size() (int, int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cols <= 0 || p.rows <= 0 {
		return pty.DefaultCols, pty.DefaultRows, nil
	}
	return p.cols, p.rows, nil
}

// ExitCode returns the daemon-reported exit status, or -1 while running.
func (p *RemotePTY) ExitCode() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitCode
}

// Close terminates the daemon terminal if it is still running, then detaches
// the local view. It is idempotent. The daemon terminal is never removed here;
// terminal.remove is an explicit separate call.
func (p *RemotePTY) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	att := p.att
	p.mu.Unlock()
	close(p.closeCh)
	p.cond.Broadcast()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if att != nil && !p.Exited() {
		if err := att.client.kill(ctx, p.id); err != nil {
			p.mgr.notifyNotice("warning", fmt.Sprintf("endpoint %s: kill %s: %v", p.cfg.Name, p.id, err))
		}
		deadline := time.Now().Add(1500 * time.Millisecond)
		for !p.Exited() && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if att != nil {
		_ = att.client.detach(ctx, att)
	}
	select {
	case <-p.doneCh:
	case <-time.After(time.Second):
	}
	p.mgr.detachRemote(p)
	return nil
}

// Exited reports whether the daemon terminal stream ended.
func (p *RemotePTY) Exited() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exited
}

// shutdown ends the local view without killing the daemon terminal: it is the
// manager-close path (TUI exit), while Close is the terminal.kill path.
func (p *RemotePTY) shutdown() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	att := p.att
	p.mu.Unlock()
	close(p.closeCh)
	p.cond.Broadcast()
	if att != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_ = att.client.detach(ctx, att)
		cancel()
	}
	select {
	case <-p.doneCh:
	case <-time.After(200 * time.Millisecond):
	}
	p.mgr.unregisterRemote(p)
}

// fail ends the read stream with an error (terminal removed, manager close).
func (p *RemotePTY) fail(err error) {
	p.mu.Lock()
	if p.readErr == nil && !p.closed {
		p.readErr = err
	}
	att := p.att
	p.pending = nil
	p.cond.Broadcast()
	p.mu.Unlock()
	if att != nil {
		att.stream.close(err)
	}
}

// window returns the viewport the next attach should use.
func (p *RemotePTY) window() (int, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cols, rows := p.cols, p.rows
	if cols <= 0 {
		cols = pty.DefaultCols
	}
	if rows <= 0 {
		rows = pty.DefaultRows
	}
	return cols, rows
}

// seedSnapshot renders the daemon native screen snapshot and prepends it to
// the read stream: the snapshot is authoritative after attach or reconnect,
// and live deltas continue from there.
func (p *RemotePTY) seedSnapshot(screen *apipb.NativeScreenResult) {
	if screen == nil {
		return
	}
	rendered := renderScreenSnapshot(screen)
	if len(rendered) == 0 {
		return
	}
	p.mu.Lock()
	if !p.closed && p.readErr == nil {
		p.buf = append(rendered, p.buf...)
	}
	p.cond.Broadcast()
	p.mu.Unlock()
}

func (p *RemotePTY) setPending(att *attachment) {
	p.mu.Lock()
	p.pending = att
	// Input/resize follow the newest live attachment immediately: the stream
	// was already started by the manager, and keeping the stale attachment
	// here would route the first keystrokes after a reconnect into a dead
	// connection.
	p.att = att
	p.mu.Unlock()
	select {
	case p.notify <- struct{}{}:
	default:
	}
}

func (p *RemotePTY) takePending() *attachment {
	p.mu.Lock()
	defer p.mu.Unlock()
	att := p.pending
	p.pending = nil
	if att != nil {
		p.att = att
	}
	return att
}

// run pumps attachments until the terminal exits or the local view closes.
// A connection loss leaves Read blocked: the manager rebinds a fresh
// attachment after reconnect and the same stream continues.
func (p *RemotePTY) run() {
	defer close(p.doneCh)
	for {
		p.mu.Lock()
		att := p.pending
		p.pending = nil
		if att != nil {
			p.att = att
		}
		closed := p.closed
		p.mu.Unlock()
		if closed {
			return
		}
		if att == nil {
			select {
			case <-p.notify:
				continue
			case <-p.closeCh:
				return
			}
		}
		err := p.pump(att)
		if errors.Is(err, errTerminalClosed) || errors.Is(err, io.EOF) {
			p.finish()
			return
		}
		if p.Closed() {
			return
		}
		// Transient loss (disconnect or dropped frames): ask the manager to
		// re-attach now when the connection is still alive, otherwise wait
		// for the reconnect supervisor to rebind.
		if errors.Is(err, ErrStreamSyncLost) {
			go p.mgr.rebindNow(p)
		}
		select {
		case <-p.notify:
		case <-p.closeCh:
			return
		}
	}
}

func (p *RemotePTY) pump(att *attachment) error {
	for {
		frame, err := att.stream.recv(p.mgr.baseCtx)
		if err != nil {
			return err
		}
		switch frame.typ {
		case wire.TypePTYOutput:
			if len(frame.payload) > 0 {
				p.feed(frame.payload)
			}
		case wire.TypeSyncLost:
			return ErrStreamSyncLost
		case wire.TypeClosed:
			p.mu.Lock()
			p.exitCode = decodeClosed(frame.payload)
			p.mu.Unlock()
			return errTerminalClosed
		case wire.TypeStreamReady:
		default:
		}
	}
}

func (p *RemotePTY) feed(data []byte) {
	p.mu.Lock()
	if !p.closed && p.readErr == nil {
		p.buf = append(p.buf, data...)
	}
	p.cond.Broadcast()
	p.mu.Unlock()
}

// finish marks the terminal exited and wakes every reader with EOF.
func (p *RemotePTY) finish() {
	p.mu.Lock()
	p.exited = true
	code := p.exitCode
	if p.readErr == nil {
		p.readErr = io.EOF
	}
	p.cond.Broadcast()
	p.mu.Unlock()
	p.mgr.markExited(p.cfg.Name, p.id, code)
	p.mgr.unregisterRemote(p)
}

// Argv returns the command this remote terminal was created with.
func (p *RemotePTY) Argv() []string { return append([]string(nil), p.argv...) }

// newTerminalID returns a process-unique daemon terminal id.
func newTerminalID() string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("tui2-%d", time.Now().UnixNano())
	}
	return "tui2-" + hex.EncodeToString(raw)
}

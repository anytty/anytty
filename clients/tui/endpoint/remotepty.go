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
var (
	errTerminalClosed = errors.New("endpoint: terminal stream closed")
	// errAttachmentDetached is local control flow. A detached attachment
	// must wake the pump without being mistaken for a daemon terminal exit.
	errAttachmentDetached = errors.New("endpoint: attachment detached")
)

// RemotePTY is the pty.PTY adapter of one daemon terminal. It satisfies the
// same interface as a local PTY, so runtime.Terminal, the ANSI parser, the
// component pipeline, input routing, kill and restart are position
// transparent (ENDPOINTS.zh-CN.md §3).
type RemotePTY struct {
	mgr                *Manager
	cfg                Config
	id                 string
	argv               []string
	surface            string
	view               string
	fit                bool
	expectedOwnerEpoch uint64
	ownerView          string
	ownerEpoch         uint64

	mu       sync.Mutex
	resizeMu sync.Mutex
	cond     *sync.Cond
	started  bool
	closed   bool
	detached bool
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
	// snapshotReset is installed by runtime.Terminal. It clears any partial
	// ANSI sequence from the previous attachment before the authoritative
	// snapshot becomes readable.
	snapshotReset func()
	// snapshotResetPending is set when seedSnapshot replaces the stream. The
	// reset is deliberately performed by Read, immediately before returning
	// the first snapshot bytes. This ordering matters when a pump has already
	// read old bytes from the previous attachment but has not parsed them yet:
	// seedSnapshot cannot reset the parser underneath that in-flight read.
	snapshotResetPending bool
	snapshotResetDone    bool
	// readInFlight/readHasData let a reconnect distinguish a blocked Read
	// (which will receive the snapshot) from a Read that already copied old
	// bytes. The runtime barrier waits for the latter chunk to be parsed before
	// resetting the parser.
	readInFlight    bool
	readHasData     bool
	readSnapshot    bool
	snapshotBarrier func()
}

// NewRemotePTY builds the unstarted daemon PTY for one attach call. The
// endpoint must already be registered.
func (m *Manager) NewRemotePTY(cfg pty.Config) *RemotePTY {
	endpointCfg, _ := m.Config(cfg.Endpoint)
	id := cfg.ID
	if id == "" {
		id = newTerminalID()
	}
	view := cfg.ViewID
	if view == "" {
		// Hosts pass their HELLO view id explicitly. Keep standalone endpoint
		// users isolated per attachment when they do not.
		view = fmt.Sprintf("tui2:%s:%s", cfg.Endpoint, id)
	}
	p := &RemotePTY{
		mgr:                m,
		cfg:                endpointCfg,
		id:                 id,
		argv:               append([]string(nil), cfg.Argv...),
		surface:            "tui2-host",
		view:               view,
		fit:                cfg.Fit,
		expectedOwnerEpoch: cfg.ExpectedOwnerEpoch,
		cols:               cfg.Cols,
		rows:               cfg.Rows,
		notify:             make(chan struct{}, 1),
		closeCh:            make(chan struct{}),
		doneCh:             make(chan struct{}),
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

// Detached reports whether this local view has released its daemon
// attachment while keeping the daemon terminal alive.
func (p *RemotePTY) Detached() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.detached
}

// Detach releases only this view's attachment. The reconnect supervisor keeps
// the RemotePTY registered but will not rebind it until Reattach is called.
func (p *RemotePTY) Detach() error {
	p.mu.Lock()
	if p.closed || p.detached {
		p.mu.Unlock()
		return nil
	}
	p.detached = true
	att := p.att
	p.att = nil
	p.pending = nil
	p.mu.Unlock()
	if att == nil {
		return nil
	}
	if att.stop != nil {
		att.stop()
		att.stop = nil
	}
	if att.stream != nil {
		att.stream.close(errAttachmentDetached)
	}
	ctx, cancel := context.WithTimeout(context.Background(), p.mgr.opts.CallTimeout)
	defer cancel()
	return att.client.detach(ctx, att)
}

// Reattach requests an immediate bind of this view to the same daemon
// terminal. It is idempotent and does not create or restart the terminal.
func (p *RemotePTY) Reattach() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return pty.ErrClosed
	}
	wasDetached := p.detached
	p.detached = false
	started := p.started
	p.mu.Unlock()
	if !started || !wasDetached {
		return nil
	}
	go p.mgr.rebindNow(p)
	return nil
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
	p.readInFlight = true
	p.readHasData = false
	p.readSnapshot = false
	for len(p.buf) == 0 && p.readErr == nil && !p.closed {
		p.cond.Wait()
	}
	if len(p.buf) > 0 {
		n := copy(b, p.buf)
		p.buf = p.buf[n:]
		reset := p.snapshotReset
		snapshotRead := p.snapshotResetPending && !p.snapshotResetDone
		p.readHasData = true
		p.readSnapshot = snapshotRead
		if !snapshotRead {
			reset = nil
		} else {
			// Mark it before unlocking so a concurrent reader cannot run the
			// hook twice. Terminal.pump is the only parser writer, and invokes
			// this hook before it writes the returned bytes to the parser.
			p.snapshotResetDone = true
		}
		p.mu.Unlock()
		if reset != nil {
			reset()
		}
		return n, nil
	}
	if p.closed {
		p.readInFlight = false
		p.mu.Unlock()
		return 0, pty.ErrClosed
	}
	err := p.readErr
	p.readInFlight = false
	p.mu.Unlock()
	return 0, err
}

// ReadDone marks the end of the runtime's parser transaction for the last
// Read. RemotePTY uses it to let a reconnect reset the parser after an old
// chunk that was already copied, but before the new snapshot is consumed.
func (p *RemotePTY) ReadDone() {
	p.mu.Lock()
	p.readInFlight = false
	p.readHasData = false
	p.readSnapshot = false
	p.mu.Unlock()
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
	p.resizeMu.Lock()
	defer p.resizeMu.Unlock()
	take := att.epoch == 0
	ctx, cancel := context.WithTimeout(p.mgr.baseCtx, p.mgr.opts.CallTimeout)
	defer cancel()
	result, err := att.client.resize(ctx, att, cols, rows, take, att.epoch)
	if err != nil {
		return fmt.Errorf("endpoint %q: resize %s: %w", p.cfg.Name, p.id, err)
	}
	p.mu.Lock()
	p.ownerView, p.ownerEpoch = att.ownerView, att.epoch
	p.mu.Unlock()
	if result.GetResized() {
		return nil
	}
	if control := result.GetResizeControl(); control != nil {
		ownership := control.GetOwnership()
		if ownership == nil {
			return fmt.Errorf("endpoint %q: resize %s denied: %s", p.cfg.Name, p.id, resizeReason(control))
		}
		if size := ownership.GetSize(); size != nil {
			// Followers must expose the daemon's authoritative size. Keeping
			// the requested local size makes every frame look out of sync and
			// causes the host to keep trying the same denied resize.
			p.mu.Lock()
			p.cols, p.rows = int(size.GetCols()), int(size.GetRows())
			p.mu.Unlock()
			if int(size.GetCols()) == cols && int(size.GetRows()) == rows {
				return nil
			}
		}
		return fmt.Errorf("endpoint %q: resize %s denied: %s (owner view %q epoch %d)",
			p.cfg.Name, p.id, resizeReason(control), ownership.GetOwnerViewId(), ownership.GetEpoch())
	}
	return nil
}

// claimOwner performs the initial remote resize-owner CAS after attachment.
// The attach API has no epoch fence, so attachments start as followers and a
// fit request is applied explicitly here.
func (p *RemotePTY) claimOwner(ctx context.Context, att *attachment) error {
	p.resizeMu.Lock()
	defer p.resizeMu.Unlock()
	cols, rows := p.window()
	result, err := att.client.resize(ctx, att, cols, rows, true, p.expectedOwnerEpoch)
	if err != nil {
		return fmt.Errorf("endpoint %q: terminal %s: claim resize owner: %w", p.cfg.Name, p.id, err)
	}
	if result.GetResized() {
		return nil
	}
	control := result.GetResizeControl()
	if control != nil {
		ownership := control.GetOwnership()
		if ownership != nil {
			size := ownership.GetSize()
			if ownership.GetOwnerViewId() == p.view && size != nil && size.GetCols() == uint32(cols) && size.GetRows() == uint32(rows) {
				return nil
			}
			return fmt.Errorf("endpoint %q: terminal %s: resize owner held by view %q epoch %d", p.cfg.Name, p.id, ownership.GetOwnerViewId(), ownership.GetEpoch())
		}
	}
	return fmt.Errorf("endpoint %q: terminal %s: resize owner claim was rejected", p.cfg.Name, p.id)
}

// RefreshResizeOwner asks the daemon for the current ownership projection
// without taking ownership or changing the requested size. It keeps every
// client's owner badge accurate after another client takes the lease.
func (p *RemotePTY) RefreshResizeOwner() bool {
	p.resizeMu.Lock()
	defer p.resizeMu.Unlock()
	p.mu.Lock()
	att := p.att
	cols, rows := p.cols, p.rows
	closed := p.closed
	p.mu.Unlock()
	if closed || att == nil || cols <= 0 || rows <= 0 {
		return false
	}
	ctx, cancel := context.WithTimeout(p.mgr.baseCtx, p.mgr.opts.CallTimeout)
	defer cancel()
	result, err := att.client.resize(ctx, att, cols, rows, false, att.epoch)
	if err != nil {
		return false
	}
	p.mu.Lock()
	p.ownerView, p.ownerEpoch = att.ownerView, att.epoch
	p.mu.Unlock()
	return result != nil
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

// ResizeOwner returns the last authoritative daemon owner projection seen by
// this attachment. A missing attachment means there is no remote snapshot.
func (p *RemotePTY) ResizeOwner() (string, uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.att == nil {
		return "", 0
	}
	return p.ownerView, p.ownerEpoch
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
	killer := sessionConn(nil)
	if att != nil {
		killer = att.client
	} else {
		p.mgr.mu.Lock()
		state := p.mgr.endpoints[p.cfg.Name]
		p.mgr.mu.Unlock()
		if state != nil {
			killer = p.mgr.currentClient(state)
		}
	}
	if killer != nil && !p.Exited() {
		if err := killer.kill(ctx, p.id); err != nil {
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

// seedSnapshot renders the daemon native screen snapshot and replaces the
// unread stream with it: the snapshot is authoritative after attach or
// reconnect, and live deltas continue from that new generation.
func (p *RemotePTY) seedSnapshot(screen *apipb.NativeScreenResult) {
	if screen == nil {
		return
	}
	rendered := renderScreenSnapshot(screen)
	if len(rendered) == 0 {
		return
	}
	var reset, barrier func()
	p.mu.Lock()
	if !p.closed && p.readErr == nil {
		// A native snapshot is an authoritative resynchronization point. Any
		// bytes still buffered belong to the previous attachment generation
		// (the stream may have reported sync-lost while the reader had not
		// drained its queue yet). Replaying them after the fresh snapshot
		// applies stale screen/input output on top of the current screen and
		// produces the characteristic one-frame back-and-forth jump at a TUI
		// scroll boundary.
		p.buf = append(p.buf[:0], rendered...)
		p.snapshotResetPending = true
		p.snapshotResetDone = false
		if p.readInFlight {
			if p.readHasData && !p.readSnapshot {
				barrier = p.snapshotBarrier
			}
		} else {
			// No parser transaction is active, so reset immediately. A blocked
			// Read will reset itself immediately before returning the snapshot.
			reset = p.snapshotReset
		}
	}
	p.cond.Broadcast()
	p.mu.Unlock()
	if barrier != nil {
		barrier()
	} else if reset != nil {
		reset()
	}
}

// SetSnapshotReset installs the parser reset hook used on reconnect. It is a
// small optional interface so RemotePTY remains usable by endpoint callers
// that do not embed the tui runtime.
func (p *RemotePTY) SetSnapshotReset(reset func()) {
	p.mu.Lock()
	p.snapshotReset = reset
	p.mu.Unlock()
}

// SetSnapshotBarrier installs the runtime hook used when seedSnapshot races a
// Read that already copied bytes from the previous attachment. The hook must
// wait for that parser transaction to finish before resetting state.
func (p *RemotePTY) SetSnapshotBarrier(barrier func()) {
	p.mu.Lock()
	p.snapshotBarrier = barrier
	p.mu.Unlock()
}

func (p *RemotePTY) setPending(att *attachment) {
	p.mu.Lock()
	p.pending = att
	if att != nil {
		p.ownerView, p.ownerEpoch = att.ownerView, att.epoch
	}
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
		if errors.Is(err, errTerminalClosed) {
			p.finish()
			return
		}
		if errors.Is(err, errAttachmentDetached) {
			select {
			case <-p.notify:
			case <-p.closeCh:
				return
			}
			continue
		}
		if errors.Is(err, io.EOF) {
			// A raw test/session implementation may still report a clean EOF
			// for detach. Check the state before treating it as process exit.
			p.mu.Lock()
			detached := p.detached
			p.mu.Unlock()
			if detached {
				select {
				case <-p.notify:
				case <-p.closeCh:
					return
				}
				continue
			}
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

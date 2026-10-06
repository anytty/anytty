package app

import (
	"sync"

	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/core"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// sender is the transport subset the event loop uses. *sdk.Client implements
// it; tests substitute an in-memory fake.
type sender interface {
	Commit(root *pb.Box, keys sdk.Keys) error
	CommitDelta(base, next *pb.Box, keys sdk.Keys) (bool, error)
	HasBase() bool
	DropBase()
	Supports(feature string) bool
	Emit(method string, params *pb.MethodParams, onResponse func(*pb.Response)) (uint64, error)
	SendStream(frame *pb.StreamFrame) error
}

// Internal loop messages; they never reach Model.Update.
type (
	// batchMsg carries the sub-commands of Batch.
	batchMsg []Cmd
	// keysMsg applies a SetKeys Cmd in the loop.
	keysMsg struct{ keys sdk.Keys }
	// quitMsg asks the loop to stop with ErrQuit.
	quitMsg struct{}
	// loopDoneMsg reports that the transport read loop ended.
	loopDoneMsg struct{ err error }
)

// queued is one message. Command results carry the epoch they were started
// in, so results from before a HELLO epoch change are dropped instead of
// leaking into the new session; messages posted by the read loop are always
// current.
type queued struct {
	epoch   uint64
	fromCmd bool
	msg     Msg
}

// engine is the transport-free core of Program: it batches queued messages,
// runs Update per message, commits at most once per batch (coalescing) and
// runs commands asynchronously. Post is safe for concurrent use; everything
// else runs on the loop goroutine.
type engine struct {
	sender  sender
	model   Model
	cleaner Cleaner
	keymap  *Keymap
	focus   *Focus
	initial sdk.Keys
	// forceFull disables the automatic delta path so every commit is a full
	// VIEW (Program.ForceFullView).
	forceFull bool
	// memo scopes Model.View to one frame per committed batch so unchanged
	// subtrees keep their *pb.Box pointer (Program.Memo). Nil disables it.
	memo *Memo

	mu        sync.Mutex
	queue     []queued
	epoch     uint64
	started   bool
	haveHello bool
	keys      sdk.Keys
	stopped   bool
	finalErr  error
	wake      chan struct{}
}

func newEngine(sender sender, m Model, keymap *Keymap, focus *Focus, keys sdk.Keys, forceFull bool, memo *Memo) *engine {
	if focus == nil {
		focus = &Focus{}
	}
	e := &engine{
		sender:    sender,
		model:     m,
		keymap:    keymap,
		focus:     focus,
		initial:   keys,
		keys:      keys,
		forceFull: forceFull,
		memo:      memo,
		wake:      make(chan struct{}, 1),
	}
	if c, ok := m.(Cleaner); ok {
		e.cleaner = c
	}
	return e
}

// Post enqueues one message from the read loop. It never blocks and is safe
// for concurrent use.
func (e *engine) Post(msg Msg) {
	if msg == nil {
		return
	}
	e.mu.Lock()
	if e.stopped {
		e.mu.Unlock()
		return
	}
	e.queue = append(e.queue, queued{msg: msg})
	e.mu.Unlock()
	e.wakeUp()
}

// postCmd enqueues the result of a command, tagged with the epoch the
// command was started in.
func (e *engine) postCmd(msg Msg, epoch uint64) {
	if msg == nil {
		return
	}
	e.mu.Lock()
	if e.stopped {
		e.mu.Unlock()
		return
	}
	e.queue = append(e.queue, queued{epoch: epoch, fromCmd: true, msg: msg})
	e.mu.Unlock()
	e.wakeUp()
}

func (e *engine) wakeUp() {
	select {
	case e.wake <- struct{}{}:
	default:
	}
}

// run waits for queued messages and feeds them to step until the loop stops.
func (e *engine) run() error {
	for {
		e.mu.Lock()
		for len(e.queue) == 0 && !e.stopped {
			e.mu.Unlock()
			<-e.wake
			e.mu.Lock()
		}
		if e.stopped {
			err := e.finalErr
			e.mu.Unlock()
			return err
		}
		batch := e.queue
		e.queue = nil
		e.mu.Unlock()
		e.process(batch)
	}
}

// step processes everything currently queued as one batch. Commands are
// spawned asynchronously; step reports whether the loop is still running.
func (e *engine) step() bool {
	e.mu.Lock()
	if e.stopped {
		e.mu.Unlock()
		return false
	}
	batch := e.queue
	e.queue = nil
	e.mu.Unlock()
	e.process(batch)
	return !e.isStopped()
}

// pendingCmd is one command plus the epoch its Update/Init ran in.
type pendingCmd struct {
	cmd   Cmd
	epoch uint64
}

// process runs one batch: Update per message, one commit, then the commands.
func (e *engine) process(batch []queued) {
	var cmds []pendingCmd
	commit, force := false, false
	for _, item := range batch {
		if item.fromCmd && item.epoch != e.currentEpoch() {
			continue // command result from before the current HELLO epoch
		}
		switch m := item.msg.(type) {
		case batchMsg:
			for _, cmd := range m {
				if cmd != nil {
					cmds = append(cmds, pendingCmd{cmd: cmd, epoch: item.epoch})
				}
			}
			continue
		case HelloMsg:
			if init := e.onHello(m.Hello); init != nil {
				cmds = append(cmds, pendingCmd{cmd: init, epoch: e.currentEpoch()})
			}
			cmds = append(cmds, pendingCmd{cmd: e.model.Update(m), epoch: e.currentEpoch()})
			commit, force = true, true
			continue
		case keysMsg:
			e.setKeys(m.keys)
			commit, force = true, true
			continue
		case ViewRejectedMsg:
			// The host rejected the committed revision (base_mismatch or
			// path_invalid): our baseline is no longer trustworthy. Drop it so
			// the next batch sends a full VIEW; never retry inside this batch
			// (PROTOCOL §2.1). The core client already dropped its base before
			// dispatching the event; this keeps a fake/other sender honest too.
			e.sender.DropBase()
			if cmd := e.model.Update(item.msg); cmd != nil {
				cmds = append(cmds, pendingCmd{cmd: cmd, epoch: e.currentEpoch()})
			}
			continue
		case quitMsg:
			e.stop(ErrQuit)
			return
		case loopDoneMsg:
			e.stop(m.err)
			return
		}
		if km, ok := item.msg.(KeyMsg); ok && e.keymap != nil {
			if bound, matched := e.keymap.Bind(km.Key); matched {
				cmds = append(cmds, pendingCmd{cmd: e.model.Update(bound), epoch: e.currentEpoch()})
				commit = true
				continue
			}
		}
		if cm, ok := item.msg.(ComponentMsg); ok {
			if h, ok := e.focus.Current().(ComponentHandler); ok {
				cmds = append(cmds, pendingCmd{cmd: e.model.Update(h.Component(cm.Component)), epoch: e.currentEpoch()})
				commit = true
				continue
			}
		}
		cmds = append(cmds, pendingCmd{cmd: e.model.Update(item.msg), epoch: e.currentEpoch()})
		commit = true
	}
	if commit {
		if err := e.commit(force); err != nil {
			e.stop(err)
			return
		}
	}
	for _, pc := range cmds {
		e.spawn(pc.cmd, pc.epoch)
	}
}

// onHello applies a HELLO: on a new epoch it resets the focus stack and the
// key declaration, calls the optional Resetter and re-runs Init. It returns
// Init's command.
func (e *engine) onHello(h *pb.Hello) Cmd {
	epoch := h.GetEpoch()
	e.mu.Lock()
	fresh := !e.started || epoch != e.epoch
	e.started = true
	e.epoch = epoch
	e.haveHello = true
	e.keys = e.initial
	e.mu.Unlock()
	if !fresh {
		return nil
	}
	e.focus.Reset()
	if r, ok := e.model.(Resetter); ok {
		r.Reset(epoch)
	}
	return e.model.Init()
}

// commit sends exactly one frame per batch unless the model's optional Cleaner
// reports it clean. force is set for HELLO/SetKeys batches, which must reach the
// host regardless.
//
// When the host advertised features["view_delta"] and a baseline exists, the
// batch is committed as a VIEW_DELTA; the SDK falls back to a full VIEW on its
// own when the delta is not expressible or not smaller. ForceFullView disables
// the delta path. A full VIEW is always used on a HELLO batch (new epoch
// baseline, PROTOCOL §2.1).
func (e *engine) commit(force bool) error {
	if !e.hasHello() {
		return nil
	}
	if !force && e.cleaner != nil && !e.cleaner.Dirty() {
		return nil
	}
	e.mu.Lock()
	keys := e.keys
	e.mu.Unlock()
	// One memo frame per batch: BeginFrame before View, EndFrame after the
	// commit, so pruning always happens exactly once per commit (including the
	// full-VIEW paths) and unused keys cannot accumulate.
	if e.memo != nil {
		e.memo.BeginFrame()
	}
	view := e.model.View()
	var err error
	if !e.forceFull && e.sender.Supports(core.ViewDeltaFeature) && e.sender.HasBase() {
		// The SDK's own committed baseline is authoritative (CommitDelta(nil,
		// ...)); it diffs against what the host actually holds and falls back
		// to a full VIEW automatically.
		_, err = e.sender.CommitDelta(nil, view, keys)
	} else {
		err = e.sender.Commit(view, keys)
	}
	if e.memo != nil {
		e.memo.EndFrame()
	}
	return err
}

// setKeys applies a SetKeys command.
func (e *engine) setKeys(keys sdk.Keys) {
	e.mu.Lock()
	e.keys = sdk.Keys{Claim: append([]string(nil), keys.Claim...), All: keys.All}
	e.mu.Unlock()
}

// spawn runs one command on its own goroutine and posts its result tagged
// with the epoch the command was created in.
func (e *engine) spawn(cmd Cmd, epoch uint64) {
	if cmd == nil {
		return
	}
	e.mu.Lock()
	stopped := e.stopped
	e.mu.Unlock()
	if stopped {
		return
	}
	go func() { e.postCmd(cmd(), epoch) }()
}

func (e *engine) stop(err error) {
	e.mu.Lock()
	if !e.stopped {
		e.stopped = true
		e.finalErr = err
	}
	e.mu.Unlock()
}

func (e *engine) isStopped() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.stopped
}

func (e *engine) currentEpoch() uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.epoch
}

func (e *engine) hasHello() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.haveHello
}

package app

import (
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/core"
	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// testModel records everything the loop does. All fields are touched on the
// test goroutine (engine.step) or, in Run integration tests, on the Run
// goroutine; the integration test only observes it through channels.
type testModel struct {
	journal   *[]string
	inits     int
	updates   []Msg
	viewCalls int
	initCmd   Cmd
	onUpdate  func(Msg) Cmd
	view      *pb.Box
}

func (m *testModel) Init() Cmd {
	m.inits++
	if m.journal != nil {
		*m.journal = append(*m.journal, "init")
	}
	return m.initCmd
}

func (m *testModel) Update(msg Msg) Cmd {
	m.updates = append(m.updates, msg)
	if m.journal != nil {
		*m.journal = append(*m.journal, "update:"+kindOf(msg))
	}
	if m.onUpdate != nil {
		return m.onUpdate(msg)
	}
	return nil
}

func (m *testModel) View() *pb.Box {
	m.viewCalls++
	return m.view
}

type dirtyModel struct {
	*testModel
	dirty bool
}

func (m *dirtyModel) Dirty() bool { return m.dirty }

type resetModel struct {
	*testModel
	resets []uint64
}

func (m *resetModel) Reset(epoch uint64) { m.resets = append(m.resets, epoch) }

// fakeSender is an in-memory Sender.
type fakeSender struct {
	mu        sync.Mutex
	journal   *[]string
	commits   []*pb.Box
	keys      []sdk.Keys
	methods   []string
	streams   []*pb.StreamFrame
	callbacks []func(*pb.Response)
	emitErr   error
	commitErr error
	// deltaEnabled and base model the client-side delta negotiation so tests
	// exercise the automatic delta path.
	deltaEnabled bool
	base         *pb.Box
	deltas       int
}

func (f *fakeSender) Commit(root *pb.Box, keys sdk.Keys) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.commitErr != nil {
		return f.commitErr
	}
	f.commits = append(f.commits, root)
	f.keys = append(f.keys, keys)
	f.base = root
	f.deltas = 0
	if f.journal != nil {
		*f.journal = append(*f.journal, "commit")
	}
	return nil
}

func (f *fakeSender) CommitDelta(base, next *pb.Box, keys sdk.Keys) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.commitErr != nil {
		return false, f.commitErr
	}
	f.commits = append(f.commits, next)
	f.keys = append(f.keys, keys)
	// Mirror the client: the committed baseline is authoritative; the base
	// argument is a hint only.
	baseline := f.base
	if baseline == nil && base != nil {
		baseline = base
	}
	f.base = next
	if f.deltaEnabled && baseline != nil {
		f.deltas++
		if f.journal != nil {
			*f.journal = append(*f.journal, "delta")
		}
		return true, nil
	}
	f.deltas = 0
	if f.journal != nil {
		*f.journal = append(*f.journal, "commit")
	}
	return false, nil
}

func (f *fakeSender) HasBase() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.base != nil
}

func (f *fakeSender) DropBase() {
	f.mu.Lock()
	f.base = nil
	f.mu.Unlock()
}

func (f *fakeSender) Supports(feature string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return feature == core.ViewDeltaFeature && f.deltaEnabled
}

func (f *fakeSender) deltaCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.deltas
}

func (f *fakeSender) Emit(method string, params *pb.MethodParams, onResponse func(*pb.Response)) (uint64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.emitErr != nil {
		return 0, f.emitErr
	}
	f.methods = append(f.methods, method)
	f.callbacks = append(f.callbacks, onResponse)
	return uint64(len(f.methods)), nil
}

func (f *fakeSender) SendStream(frame *pb.StreamFrame) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.streams = append(f.streams, frame)
	return nil
}

func (f *fakeSender) commitCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.commits)
}

// committed returns a snapshot of the committed roots in order, so tests can
// diff one frame against the previous one.
func (f *fakeSender) committed() []*pb.Box {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*pb.Box(nil), f.commits...)
}

func (f *fakeSender) lastKeys() sdk.Keys {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.keys) == 0 {
		return sdk.Keys{}
	}
	return f.keys[len(f.keys)-1]
}

func kindOf(msg Msg) string {
	switch msg.(type) {
	case HelloMsg:
		return "hello"
	case KeyMsg:
		return "key"
	case ComponentMsg:
		return "component"
	case asyncMsg:
		return "async"
	case secondMsg:
		return "second"
	default:
		return "other"
	}
}

func helloFor(epoch uint64) HelloMsg {
	return HelloMsg{Hello: &pb.Hello{Schema: 1, ViewId: "v", Epoch: epoch, Cols: 80, Rows: 24}}
}

func keyFor(key string) KeyMsg {
	return KeyMsg{Key: &pb.KeyEvent{Id: "ev", Key: key}}
}

func newTestEngine(m Model, f *fakeSender, km *Keymap, focus *Focus) *engine {
	return newEngine(f, m, km, focus, sdk.Keys{All: true}, false, nil)
}

func newFullEngine(m Model, f *fakeSender, km *Keymap, focus *Focus) *engine {
	return newEngine(f, m, km, focus, sdk.Keys{All: true}, true, nil)
}

func newMemoEngine(m Model, f *fakeSender, memo *Memo) *engine {
	return newEngine(f, m, nil, nil, sdk.Keys{All: true}, false, memo)
}

// waitStep pumps the loop until cond holds (async commands post as they
// finish).
func waitStep(t *testing.T, e *engine, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		e.step()
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

type asyncMsg struct{ n int }
type secondMsg struct{}

func TestBatchCoalescing(t *testing.T) {
	journal := &[]string{}
	model := &testModel{journal: journal}
	sender := &fakeSender{journal: journal}
	e := newTestEngine(model, sender, nil, nil)

	e.Post(helloFor(1))
	e.Post(keyFor("a"))
	e.Post(keyFor("b"))
	e.Post(keyFor("c"))
	if !e.step() {
		t.Fatal("engine stopped unexpectedly")
	}

	if model.inits != 1 {
		t.Fatalf("inits = %d, want 1", model.inits)
	}
	if len(model.updates) != 4 {
		t.Fatalf("updates = %d, want 4 (hello + 3 keys)", len(model.updates))
	}
	if got := sender.commitCount(); got != 1 {
		t.Fatalf("commits = %d, want exactly 1 per batch", got)
	}
	if model.viewCalls != 1 {
		t.Fatalf("view calls = %d, want 1", model.viewCalls)
	}
	// The commit happens once, after every Update in the batch.
	joined := strings.Join(*journal, ",")
	if !strings.HasSuffix(joined, "update:key,commit") {
		t.Fatalf("journal = %v, want the batch committed once at the end", *journal)
	}

	// An empty batch is not a commit.
	e.step()
	if got := sender.commitCount(); got != 1 {
		t.Fatalf("commits after empty batch = %d, want 1", got)
	}
}

func TestInitOrderingAndDefaultKeys(t *testing.T) {
	journal := &[]string{}
	model := &testModel{journal: journal}
	sender := &fakeSender{journal: journal}
	e := newTestEngine(model, sender, nil, nil)

	e.Post(helloFor(3))
	e.Post(keyFor("x"))
	e.step()

	if len(*journal) < 3 || (*journal)[0] != "init" {
		t.Fatalf("journal = %v, want init first", *journal)
	}
	if (*journal)[1] != "update:hello" || (*journal)[2] != "update:key" {
		t.Fatalf("journal = %v, want update:hello before update:key", *journal)
	}
	if keys := sender.lastKeys(); !keys.All || len(keys.Claim) != 0 {
		t.Fatalf("committed keys = %+v, want All:true", keys)
	}
}

func TestCmdAsyncResultAndBatch(t *testing.T) {
	model := &testModel{}
	sender := &fakeSender{}
	e := newTestEngine(model, sender, nil, nil)
	e.Post(helloFor(1))
	e.step()

	seen := func(kind string) func() bool {
		return func() bool {
			for _, msg := range model.updates {
				if kindOf(msg) == kind {
					return true
				}
			}
			return false
		}
	}

	model.onUpdate = func(msg Msg) Cmd {
		if _, ok := msg.(KeyMsg); !ok {
			return nil
		}
		return Batch(
			func() Msg { return asyncMsg{n: 1} },
			func() Msg { return secondMsg{} },
		)
	}
	e.Post(keyFor("go"))
	e.step()

	waitStep(t, e, "async result delivered", seen("async"))
	waitStep(t, e, "second batch result delivered", seen("second"))
	if sender.commitCount() < 3 {
		t.Fatalf("commits = %d, want one per processed batch", sender.commitCount())
	}
}

func TestCmdErrorIsDelivered(t *testing.T) {
	model := &testModel{}
	sender := &fakeSender{emitErr: errors.New("boom")}
	e := newTestEngine(model, sender, nil, nil)
	e.Post(helloFor(1))
	e.step()

	model.onUpdate = func(msg Msg) Cmd {
		if _, ok := msg.(KeyMsg); ok {
			return func() Msg { return ErrorMsg{Err: sender.emitErr} }
		}
		return nil
	}
	e.Post(keyFor("e"))
	e.step()
	waitStep(t, e, "error msg delivered", func() bool {
		for _, msg := range model.updates {
			if em, ok := msg.(ErrorMsg); ok && em.Err != nil {
				return true
			}
		}
		return false
	})
}

func TestQuit(t *testing.T) {
	model := &testModel{}
	sender := &fakeSender{}
	e := newTestEngine(model, sender, nil, nil)
	e.Post(helloFor(1))
	e.step()

	model.onUpdate = func(msg Msg) Cmd {
		if _, ok := msg.(KeyMsg); ok {
			return Quit()
		}
		return nil
	}
	e.Post(keyFor("q"))
	waitStep(t, e, "quit", func() bool { return e.isStopped() })
	if !errors.Is(e.finalErr, ErrQuit) {
		t.Fatalf("final error = %v, want ErrQuit", e.finalErr)
	}
	// Posting after quit is a no-op.
	e.Post(keyFor("x"))
	if e.step() {
		t.Fatal("step after quit reported running")
	}
}

func TestSetKeysAppliesFromNextCommit(t *testing.T) {
	model := &testModel{}
	sender := &fakeSender{}
	e := newTestEngine(model, sender, nil, nil)
	e.Post(helloFor(1))
	e.step()
	if keys := sender.lastKeys(); !keys.All {
		t.Fatalf("initial keys = %+v, want All:true", keys)
	}

	model.onUpdate = func(msg Msg) Cmd {
		if _, ok := msg.(KeyMsg); ok {
			return SetKeys(sdk.Keys{Claim: []string{"ctrl-p"}})
		}
		return nil
	}
	e.Post(keyFor("s"))
	waitStep(t, e, "keys applied", func() bool {
		keys := sender.lastKeys()
		return !keys.All && len(keys.Claim) == 1 && keys.Claim[0] == "ctrl-p"
	})
}

func TestKeymapDispatch(t *testing.T) {
	model := &testModel{}
	sender := &fakeSender{}
	km := NewKeymap(
		Binding{Key: "ctrl-p", Msg: asyncMsg{n: 1}},
		Binding{Key: "p", Mods: []string{"alt"}, Msg: asyncMsg{n: 2}},
		Binding{Key: "g", Context: "picker", Msg: asyncMsg{n: 3}},
		Binding{Key: "g", Msg: asyncMsg{n: 4}},
	)
	e := newTestEngine(model, sender, km, nil)
	e.Post(helloFor(1))
	e.Post(keyFor("ctrl-p"))
	e.Post(keyFor("alt-p"))
	e.Post(keyFor("g"))
	e.Post(keyFor("unbound"))
	e.step()

	var got []int
	for _, msg := range model.updates {
		switch m := msg.(type) {
		case asyncMsg:
			got = append(got, m.n)
		case KeyMsg:
			got = append(got, 100)
		}
	}
	if len(got) != 4 || got[0] != 1 || got[1] != 2 || got[2] != 4 || got[3] != 100 {
		t.Fatalf("routed msgs = %v, want [1 2 4 100]", got)
	}

	// A contextual binding wins while its context is active.
	km.SetContext("picker")
	e.Post(keyFor("g"))
	e.step()
	last := model.updates[len(model.updates)-1].(asyncMsg)
	if last.n != 3 {
		t.Fatalf("contextual binding = %d, want 3", last.n)
	}
}

func TestFocusRouting(t *testing.T) {
	model := &testModel{}
	sender := &fakeSender{}
	focus := &Focus{}
	e := newTestEngine(model, sender, nil, focus)

	// HELLO resets the stack, so focus is pushed the way a model would: in
	// response to the handshake.
	e.Post(helloFor(1))
	e.step()
	entry := &focusEntry{ret: asyncMsg{n: 9}}
	focus.Push(entry)

	e.Post(ComponentMsg{Component: &pb.ComponentEvent{Source: "picker", Name: "pick", Value: "x"}})
	e.step()

	if len(entry.got) != 1 || entry.got[0].GetValue() != "x" {
		t.Fatalf("focus handler saw %d events, want the component event", len(entry.got))
	}
	last := model.updates[len(model.updates)-1]
	if m, ok := last.(asyncMsg); !ok || m.n != 9 {
		t.Fatalf("last update = %#v, want the routed asyncMsg", last)
	}

	focus.Pop()
	e.Post(ComponentMsg{Component: &pb.ComponentEvent{Value: "y"}})
	e.step()
	last = model.updates[len(model.updates)-1]
	if _, ok := last.(ComponentMsg); !ok {
		t.Fatalf("unfocused component event = %#v, want ComponentMsg", last)
	}
}

type focusEntry struct {
	got []*pb.ComponentEvent
	ret Msg
}

func (f *focusEntry) Component(ev *pb.ComponentEvent) Msg {
	f.got = append(f.got, ev)
	return f.ret
}

func TestEpochReInitAndStaleCmdDrop(t *testing.T) {
	model := &resetModel{testModel: &testModel{}}
	sender := &fakeSender{}
	focus := &Focus{}
	focus.Push(&focusEntry{})
	e := newTestEngine(model, sender, nil, focus)

	release := make(chan struct{})
	helloCount := 0
	model.onUpdate = func(msg Msg) Cmd {
		if _, ok := msg.(HelloMsg); ok {
			helloCount++
			if helloCount == 1 {
				return func() Msg { <-release; return asyncMsg{n: 1} }
			}
		}
		return nil
	}

	e.Post(helloFor(1))
	e.step()
	if model.inits != 1 || len(model.resets) != 1 || model.resets[0] != 1 {
		t.Fatalf("inits/resets = %d/%v, want 1/[1]", model.inits, model.resets)
	}
	if focus.Len() != 0 {
		t.Fatalf("focus depth = %d, want 0 after a new epoch", focus.Len())
	}

	e.Post(helloFor(2))
	e.step()
	if model.inits != 2 || len(model.resets) != 2 || model.resets[1] != 2 {
		t.Fatalf("inits/resets = %d/%v, want 2/[1 2]", model.inits, model.resets)
	}
	if sender.commitCount() != 2 {
		t.Fatalf("commits = %d, want a forced commit per epoch", sender.commitCount())
	}

	// The command started in epoch 1 completes after the epoch-2 HELLO; its
	// message must not reach Update.
	close(release)
	waitFor(t, "stale command queued", func() bool { return e.queuedLen() > 0 })
	before := len(model.updates)
	e.step()
	if len(model.updates) != before {
		t.Fatalf("stale command result leaked into epoch 2: %v", model.updates[before:])
	}
}

func (e *engine) queuedLen() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.queue)
}

func TestDirtySuppression(t *testing.T) {
	base := &testModel{}
	model := &dirtyModel{testModel: base}
	sender := &fakeSender{}
	e := newTestEngine(model, sender, nil, nil)

	e.Post(helloFor(1))
	e.step()
	if sender.commitCount() != 1 {
		t.Fatalf("hello commit = %d, want 1", sender.commitCount())
	}
	if base.viewCalls != 1 {
		t.Fatalf("view calls = %d, want 1", base.viewCalls)
	}

	model.dirty = false
	e.Post(keyFor("x"))
	e.step()
	if sender.commitCount() != 1 || base.viewCalls != 1 {
		t.Fatalf("clean batch committed: commits=%d views=%d", sender.commitCount(), base.viewCalls)
	}

	model.dirty = true
	e.Post(keyFor("y"))
	e.step()
	if sender.commitCount() != 2 || base.viewCalls != 2 {
		t.Fatalf("dirty batch not committed: commits=%d views=%d", sender.commitCount(), base.viewCalls)
	}
}

func TestCommitErrorStopsRun(t *testing.T) {
	model := &testModel{}
	sender := &fakeSender{commitErr: errors.New("write failed")}
	e := newTestEngine(model, sender, nil, nil)

	e.Post(helloFor(1))
	e.Post(helloFor(1))
	e.step()
	if !e.isStopped() {
		t.Fatal("engine kept running after a commit error")
	}
	if e.finalErr == nil || e.finalErr.Error() != "write failed" {
		t.Fatalf("final error = %v, want write failed", e.finalErr)
	}
}

func TestCmdHelpers(t *testing.T) {
	if None != nil {
		t.Fatal("None must be the nil Cmd")
	}
	if Batch(nil, nil) != nil {
		t.Fatal("empty Batch must be nil")
	}
	msg := Tick(0)()
	if _, ok := msg.(TickMsg); !ok {
		t.Fatalf("Tick msg = %T, want TickMsg", msg)
	}
	msg = SetKeys(sdk.Keys{Claim: []string{"x"}})()
	km, ok := msg.(keysMsg)
	if !ok || len(km.keys.Claim) != 1 || km.keys.Claim[0] != "x" {
		t.Fatalf("SetKeys msg = %#v", msg)
	}
	msg = Quit()()
	if _, ok := msg.(quitMsg); !ok {
		t.Fatalf("Quit msg = %T, want quitMsg", msg)
	}
	// Tick must not fire before its delay.
	start := time.Now()
	cmd := Tick(5 * time.Millisecond)
	if cmd == nil {
		t.Fatal("Tick returned nil")
	}
	if time.Since(start) > time.Second {
		t.Fatal("Tick must not block at creation")
	}
}

func TestRunNilInputs(t *testing.T) {
	if err := Run(nil, &testModel{}); err == nil {
		t.Fatal("Run(nil, model) = nil, want an error")
	}
	if err := (&Program{}).Run(); err == nil {
		t.Fatal("empty Program.Run() = nil, want an error")
	}
}

// --- Program.Run integration over an in-memory pipe ---

type pipe struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
}

func newPipe() *pipe {
	p := &pipe{}
	p.cond = sync.NewCond(&p.mu)
	return p
}

func (p *pipe) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0, io.ErrClosedPipe
	}
	p.buf = append(p.buf, b...)
	p.cond.Broadcast()
	return len(b), nil
}

func (p *pipe) Read(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(p.buf) == 0 && !p.closed {
		p.cond.Wait()
	}
	if len(p.buf) == 0 {
		return 0, io.EOF
	}
	n := copy(b, p.buf)
	p.buf = p.buf[n:]
	return n, nil
}

func (p *pipe) Close() error {
	p.mu.Lock()
	p.closed = true
	p.cond.Broadcast()
	p.mu.Unlock()
	return nil
}

type runModel struct {
	client *sdk.Client
	mu     sync.Mutex
	seen   []Msg
}

func (m *runModel) Init() Cmd { return nil }

func (m *runModel) Update(msg Msg) Cmd {
	m.mu.Lock()
	m.seen = append(m.seen, msg)
	m.mu.Unlock()
	if key, ok := msg.(KeyMsg); ok {
		switch key.Key.GetKey() {
		case "e":
			return Emit(m.client, "access.call", &pb.MethodParams{Endpoint: "local"},
				func(payload []byte) Msg { return decodedMsg(payload) })
		case "q":
			return Quit()
		}
	}
	return nil
}

func (m *runModel) View() *pb.Box { return sdk.Text("hello").Build() }

func (m *runModel) seenKinds() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.seen))
	for _, msg := range m.seen {
		switch msg := msg.(type) {
		case HelloMsg:
			out = append(out, "hello")
		case KeyMsg:
			out = append(out, "key:"+msg.Key.GetKey())
		case decodedMsg:
			out = append(out, "decoded:"+string(msg))
		default:
			out = append(out, "other")
		}
	}
	return out
}

type decodedMsg []byte

func TestProgramRunIntegration(t *testing.T) {
	hostToProg := newPipe()
	progToHost := newPipe()
	defer hostToProg.Close()
	defer progToHost.Close()

	client := sdk.New(hostToProg, progToHost, sdk.Handlers{})
	model := &runModel{client: client}
	errCh := make(chan error, 1)
	go func() { errCh <- Run(client, model) }()

	hostEnc := wire.NewEncoder(hostToProg, wire.RoleHost, 0)
	hostDec := wire.NewDecoder(progToHost, wire.RoleHost, 0)

	if err := hostEnc.Encode(wire.TypeHello, &pb.Hello{Schema: 1, ViewId: "v", Epoch: 1, Cols: 80, Rows: 24}); err != nil {
		t.Fatalf("hello: %v", err)
	}
	typ, payload := readFrame(t, hostDec)
	if typ != wire.TypeView {
		t.Fatalf("first frame = %v, want VIEW", typ)
	}
	view, err := wire.UnmarshalPayload(wire.TypeView, payload)
	if err != nil {
		t.Fatalf("view: %v", err)
	}
	if view.(*pb.View).GetRoot().GetContent().GetText() != "hello" {
		t.Fatalf("view root = %v", view)
	}

	if err := hostEnc.Encode(wire.TypeEvent, &pb.Event{Event: &pb.Event_Key{Key: &pb.KeyEvent{Id: "e1", Key: "e"}}}); err != nil {
		t.Fatalf("key e: %v", err)
	}
	// The handling batch commits first, then the Emit command runs.
	if typ, _ = readFrame(t, hostDec); typ != wire.TypeView {
		t.Fatalf("key batch frame = %v, want VIEW", typ)
	}
	typ, payload = readFrame(t, hostDec)
	if typ != wire.TypeResult {
		t.Fatalf("emit frame = %v, want RESULT", typ)
	}
	result, err := wire.UnmarshalPayload(wire.TypeResult, payload)
	if err != nil {
		t.Fatalf("result: %v", err)
	}
	reqID := result.(*pb.Result).GetRequestId()
	if err := hostEnc.Encode(wire.TypeResponse, &pb.Response{
		RequestId: reqID, Epoch: 1, Ok: true, Data: &pb.MethodData{AccessResult: []byte("payload")},
	}); err != nil {
		t.Fatalf("response: %v", err)
	}
	// The decoded result arrives and triggers the batch commit.
	typ, _ = readFrame(t, hostDec)
	if typ != wire.TypeView {
		t.Fatalf("post-response frame = %v, want VIEW", typ)
	}
	waitFor(t, "decoded payload", func() bool {
		for _, kind := range model.seenKinds() {
			if kind == "decoded:payload" {
				return true
			}
		}
		return false
	})

	if err := hostEnc.Encode(wire.TypeEvent, &pb.Event{Event: &pb.Event_Key{Key: &pb.KeyEvent{Id: "q", Key: "q"}}}); err != nil {
		t.Fatalf("key q: %v", err)
	}
	select {
	case err := <-errCh:
		if !errors.Is(err, ErrQuit) {
			t.Fatalf("Run error = %v, want ErrQuit", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for Run to quit")
	}
}

// runDeltaModel keeps a small changing tree so the loop produces a real delta
// through the production sdk.Client.
type runDeltaModel struct {
	mu  sync.Mutex
	val string
}

func (m *runDeltaModel) Init() Cmd { return nil }

func (m *runDeltaModel) Update(msg Msg) Cmd {
	if _, ok := msg.(KeyMsg); ok {
		m.mu.Lock()
		m.val += "x"
		m.mu.Unlock()
	}
	return nil
}

func (m *runDeltaModel) View() *pb.Box {
	m.mu.Lock()
	defer m.mu.Unlock()
	return sdk.Row(sdk.Text("head"), sdk.Text(m.val)).Build()
}

// TestProgramRunDeltaIntegration drives the real Client through Run with a
// host that advertises view_delta: the HELLO batch is a full VIEW and the next
// batch is a VIEW_DELTA against it.
func TestProgramRunDeltaIntegration(t *testing.T) {
	hostToProg := newPipe()
	progToHost := newPipe()
	defer hostToProg.Close()
	defer progToHost.Close()

	client := sdk.New(hostToProg, progToHost, sdk.Handlers{})
	model := &runDeltaModel{}
	errCh := make(chan error, 1)
	go func() { errCh <- Run(client, model) }()

	hostEnc := wire.NewEncoder(hostToProg, wire.RoleHost, 0)
	hostDec := wire.NewDecoder(progToHost, wire.RoleHost, 0)

	if err := hostEnc.Encode(wire.TypeHello, &pb.Hello{
		Schema: 1, ViewId: "v", Epoch: 1, Cols: 80, Rows: 24,
		Features: map[string]bool{"view_delta": true},
	}); err != nil {
		t.Fatalf("hello: %v", err)
	}
	if typ, _ := readFrame(t, hostDec); typ != wire.TypeView {
		t.Fatalf("first frame = %v, want VIEW (no baseline)", typ)
	}

	if err := hostEnc.Encode(wire.TypeEvent, &pb.Event{Event: &pb.Event_Key{Key: &pb.KeyEvent{Id: "k", Key: "a"}}}); err != nil {
		t.Fatalf("key: %v", err)
	}
	typ, payload := readFrame(t, hostDec)
	if typ != wire.TypeViewDelta {
		t.Fatalf("second frame = %v, want VIEW_DELTA", typ)
	}
	m, err := wire.UnmarshalPayload(wire.TypeViewDelta, payload)
	if err != nil {
		t.Fatalf("decode delta: %v", err)
	}
	delta := m.(*pb.ViewDelta)
	if delta.GetRev() != 2 || delta.GetRevBase() != 1 {
		t.Fatalf("delta rev/base = %d/%d, want 2/1", delta.GetRev(), delta.GetRevBase())
	}
	if len(delta.GetPatches()) != 1 || delta.GetPatches()[0].GetOp() != "set" {
		t.Fatalf("delta patches = %v, want one set", delta.GetPatches())
	}

	// Close the host side and let Run finish cleanly.
	_ = hostToProg.Close()
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, ErrQuit) {
			t.Fatalf("Run error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for Run to finish")
	}
}

func readFrame(t *testing.T, dec *wire.Decoder) (wire.Type, []byte) {
	t.Helper()
	type result struct {
		t       wire.Type
		payload []byte
		err     error
	}
	ch := make(chan result, 1)
	go func() {
		typ, payload, err := dec.Decode()
		ch <- result{typ, payload, err}
	}()
	select {
	case got := <-ch:
		if got.err != nil {
			t.Fatalf("read frame: %v", got.err)
		}
		return got.t, got.payload
	case <-time.After(3 * time.Second):
		t.Fatal("timeout reading frame")
		return 0, nil
	}
}

func TestDeltaPathAutomaticWhenSupported(t *testing.T) {
	model := &testModel{view: sdk.Text("v").Build()}
	sender := &fakeSender{deltaEnabled: true}
	e := newTestEngine(model, sender, nil, nil)

	// The HELLO batch has no baseline: it is a full commit.
	e.Post(helloFor(1))
	e.step()
	if sender.commitCount() != 1 {
		t.Fatalf("hello commits = %d, want 1", sender.commitCount())
	}
	if sender.deltaCount() != 0 {
		t.Fatalf("hello delta count = %d, want 0 (no baseline yet)", sender.deltaCount())
	}

	// The next batch reuses the baseline and goes out as a delta.
	e.Post(keyFor("x"))
	e.step()
	if sender.commitCount() != 2 {
		t.Fatalf("commits = %d, want one frame per batch", sender.commitCount())
	}
	if sender.deltaCount() != 1 {
		t.Fatalf("delta count = %d, want 1", sender.deltaCount())
	}
}

func TestForceFullViewDisablesDelta(t *testing.T) {
	model := &testModel{view: sdk.Text("v").Build()}
	sender := &fakeSender{deltaEnabled: true}
	e := newFullEngine(model, sender, nil, nil)

	e.Post(helloFor(1))
	e.step()
	e.Post(keyFor("x"))
	e.step()

	if sender.commitCount() != 2 {
		t.Fatalf("commits = %d, want 2", sender.commitCount())
	}
	if sender.deltaCount() != 0 {
		t.Fatalf("delta count = %d, want 0 with ForceFullView", sender.deltaCount())
	}
}

func TestDeltaNotUsedWithoutFeature(t *testing.T) {
	model := &testModel{}
	sender := &fakeSender{}
	e := newTestEngine(model, sender, nil, nil)

	e.Post(helloFor(1))
	e.step()
	e.Post(keyFor("x"))
	e.step()
	if sender.deltaCount() != 0 {
		t.Fatalf("delta count = %d, want 0 without the feature", sender.deltaCount())
	}
}

func TestViewRejectedDropsBaseAndDoesNotRetry(t *testing.T) {
	model := &testModel{view: sdk.Text("v").Build()}
	sender := &fakeSender{deltaEnabled: true}
	e := newTestEngine(model, sender, nil, nil)

	e.Post(helloFor(1))
	e.step()
	e.Post(keyFor("x"))
	e.step()
	if sender.deltaCount() != 1 {
		t.Fatalf("pre-rejection delta count = %d, want 1", sender.deltaCount())
	}
	if !sender.HasBase() {
		t.Fatal("baseline missing before the rejection")
	}

	// The rejection is delivered to Update and drops the baseline; the batch
	// must not retry in place, so it re-commits at most the single frame it
	// would otherwise have committed.
	before := sender.commitCount()
	e.Post(ViewRejectedMsg{Epoch: 1, Rev: 2, Reason: "base_mismatch"})
	e.step()

	if sender.HasBase() {
		t.Fatal("view_rejected did not drop the baseline")
	}
	if got := sender.commitCount(); got > before+1 {
		t.Fatalf("commits after rejection = %d, want at most %d (no in-batch retry)", got, before+1)
	}
	last := model.updates[len(model.updates)-1]
	if _, ok := last.(ViewRejectedMsg); !ok {
		t.Fatalf("last update = %#v, want ViewRejectedMsg", last)
	}

	// The next batch must be a full commit, not a delta.
	e.Post(keyFor("y"))
	e.step()
	if sender.deltaCount() != 0 {
		t.Fatalf("delta count = %d after rejection, want 0 (full VIEW)", sender.deltaCount())
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

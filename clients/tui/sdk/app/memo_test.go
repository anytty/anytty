package app

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/builder"
	"github.com/anytty/anytty/clients/tui/sdk/core"
	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func textBox(text string) *pb.Box { return sdk.Text(text).Build() }

// TestMemoZeroValue pins that a fresh Memo works without any setup: the first
// Box call builds and caches, the second returns the same pointer.
func TestMemoZeroValue(t *testing.T) {
	var m Memo
	built := 0
	build := func() *pb.Box { built++; return textBox("x") }
	first := m.Box("k", build)
	second := m.Box("k", build)
	if first != second {
		t.Fatal("same key returned different pointers")
	}
	if built != 1 {
		t.Fatalf("build calls = %d, want 1", built)
	}
}

// TestMemoFrameReuse pins that a key requested in consecutive frames keeps its
// pointer without rebuilding.
func TestMemoFrameReuse(t *testing.T) {
	var m Memo
	built := 0
	build := func() *pb.Box { built++; return textBox("x") }

	m.BeginFrame()
	first := m.Box("k", build)
	m.EndFrame()

	m.BeginFrame()
	second := m.Box("k", build)
	m.EndFrame()

	if first != second {
		t.Fatal("same key across frames returned different pointers")
	}
	if built != 1 {
		t.Fatalf("build calls = %d, want 1", built)
	}
}

// TestMemoChangedKey pins that a key that changed rebuilds and yields a new
// pointer, even in the same frame.
func TestMemoChangedKey(t *testing.T) {
	var m Memo
	build := func() *pb.Box { return textBox("x") }

	m.BeginFrame()
	before := m.Box("v1", build)
	after := m.Box("v2", build)
	m.EndFrame()

	if before == after {
		t.Fatal("different keys returned the same pointer")
	}
	if before.GetContent().GetText() != after.GetContent().GetText() {
		t.Fatal("test boxes must be deeply equal, only pointers differ")
	}
}

// TestMemoPruning pins the frame scope: a key unused in a frame is dropped,
// while a key used in every frame survives without a rebuild.
func TestMemoPruning(t *testing.T) {
	var m Memo
	built := 0
	build := func() *pb.Box { built++; return textBox("x") }

	// Frame 1: build both.
	m.BeginFrame()
	a := m.Box("a", build)
	b := m.Box("b", build)
	m.EndFrame()
	if built != 2 {
		t.Fatalf("after frame 1 builds = %d, want 2", built)
	}

	// Frame 2: request both again; both are reused, nothing is built.
	m.BeginFrame()
	if got := m.Box("a", build); got != a {
		t.Fatal("a was rebuilt in frame 2")
	}
	if got := m.Box("b", build); got != b {
		t.Fatal("b was rebuilt in frame 2")
	}
	m.EndFrame()
	if built != 2 {
		t.Fatalf("after reuse builds = %d, want 2", built)
	}

	// Frame 3: only b is requested, so a is pruned.
	m.BeginFrame()
	if got := m.Box("b", build); got != b {
		t.Fatal("b was rebuilt in frame 2")
	}
	m.EndFrame()

	// Frame 4: a must be rebuilt (pruned), b must still be reused.
	m.BeginFrame()
	if got := m.Box("a", build); got == a {
		t.Fatal("a was not pruned at EndFrame")
	}
	if got := m.Box("b", build); got != b {
		t.Fatal("b was pruned although it was requested in frame 3")
	}
	m.EndFrame()
	if built != 3 {
		t.Fatalf("after prune builds = %d, want 3", built)
	}
}

// TestMemoNonComparableKeyPanics documents that keys must be comparable: an
// unhashable key panics on the map lookup.
func TestMemoNonComparableKeyPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("unhashable key did not panic")
		}
	}()
	var m Memo
	m.Box([]string{"k"}, func() *pb.Box { return textBox("x") })
}

// memoTree builds a deterministic 8-way tree with exactly n boxes, mirroring
// the bench tree, so the integration tests exercise a realistic node count.
func memoTree(n int) *builder.Builder {
	if n <= 1 {
		return builder.Text("leaf")
	}
	root := builder.Col()
	rest := n - 1
	children := 8
	if rest < children {
		children = rest
	}
	base, extra := rest/children, rest%children
	for i := 0; i < children; i++ {
		count := base
		if i < extra {
			count++
		}
		root.Child(memoTree(count))
	}
	return root
}

// memoModel has two subtrees: a large one that never changes and is memoized
// under a constant key, and a small one keyed by the counter so every batch
// rebuilds only it. The root is built fresh per View.
type memoModel struct {
	memo *Memo
	n    int
	mu   sync.Mutex
}

func (m *memoModel) Init() Cmd { return nil }

func (m *memoModel) Update(msg Msg) Cmd {
	if _, ok := msg.(KeyMsg); ok {
		m.mu.Lock()
		m.n++
		m.mu.Unlock()
	}
	return nil
}

type memoCounterKey struct{ n int }

func (m *memoModel) View() *pb.Box {
	m.mu.Lock()
	n := m.n
	m.mu.Unlock()
	stable := m.memo.Box("stable", func() *pb.Box { return memoTree(9000).Build() })
	counter := m.memo.Box(memoCounterKey{n: n}, func() *pb.Box {
		return sdk.Row(sdk.Text("counter"), sdk.Text(fmt.Sprintf("n=%d", n))).Build()
	})
	// The root is rebuilt fresh every frame; only its children are memoized.
	return &pb.Box{Flow: "col", Children: []*pb.Box{stable, counter}}
}

// TestProgramMemoDeltaIsProportionalToChange drives the engine over the fake
// sender and asserts that each batch after the first diffs as one patch on the
// small subtree, while the large subtree keeps its pointer across frames.
func TestProgramMemoDeltaIsProportionalToChange(t *testing.T) {
	var memo Memo
	model := &memoModel{memo: &memo}
	sender := &fakeSender{deltaEnabled: true}
	e := newMemoEngine(model, sender, &memo)

	e.Post(helloFor(1))
	e.step()
	for i := 0; i < 3; i++ {
		e.Post(keyFor("x"))
		e.step()
	}

	frames := sender.committed()
	if len(frames) != 4 {
		t.Fatalf("committed frames = %d, want 4 (hello + 3 keys)", len(frames))
	}
	if sender.deltaCount() != 3 {
		t.Fatalf("delta count = %d, want 3", sender.deltaCount())
	}
	for i := 1; i < len(frames); i++ {
		base, next := frames[i-1], frames[i]
		if base.GetChildren()[0] != next.GetChildren()[0] {
			t.Fatalf("frame %d: stable subtree pointer changed across frames", i)
		}
		patches, ok := core.DiffView(base, next)
		if !ok {
			t.Fatalf("frame %d: diff not expressible", i)
		}
		if len(patches) != 1 || patches[0].GetOp() != core.OpSet {
			t.Fatalf("frame %d: patches = %v, want one set on the changed subtree", i, patches)
		}
	}
}

// TestProgramRunMemoDeltaBytes drives the real wire over in-memory pipes with
// Program.Memo: the HELLO batch is a full VIEW and each later batch is a
// VIEW_DELTA under 200 bytes carrying one patch, accepted and chained by the
// baseline revision.
func TestProgramRunMemoDeltaBytes(t *testing.T) {
	hostToProg := newPipe()
	progToHost := newPipe()
	defer hostToProg.Close()
	defer progToHost.Close()

	client := sdk.New(hostToProg, progToHost, sdk.Handlers{})
	var memo Memo
	model := &memoModel{memo: &memo}
	errCh := make(chan error, 1)
	go func() { errCh <- (&Program{Client: client, Model: model, Memo: &memo}).Run() }()

	hostEnc := wire.NewEncoder(hostToProg, wire.RoleHost, 0)
	hostDec := wire.NewDecoder(progToHost, wire.RoleHost, 0)

	if err := hostEnc.Encode(wire.TypeHello, &pb.Hello{
		Schema: 1, ViewId: "v", Epoch: 1, Cols: 80, Rows: 24,
		Features: map[string]bool{"view_delta": true},
	}); err != nil {
		t.Fatalf("hello: %v", err)
	}
	typ, payload := readFrame(t, hostDec)
	if typ != wire.TypeView {
		t.Fatalf("first frame = %v, want VIEW (no baseline)", typ)
	}
	base, err := wire.UnmarshalPayload(wire.TypeView, payload)
	if err != nil {
		t.Fatalf("decode view: %v", err)
	}
	if got := countBoxes(base.(*pb.View).GetRoot()); got < 9000 {
		t.Fatalf("full VIEW boxes = %d, want the ~10k tree", got)
	}

	for wantRev := uint64(2); wantRev <= 3; wantRev++ {
		if err := hostEnc.Encode(wire.TypeEvent, &pb.Event{Event: &pb.Event_Key{Key: &pb.KeyEvent{Id: "k", Key: "a"}}}); err != nil {
			t.Fatalf("key: %v", err)
		}
		typ, payload = readFrame(t, hostDec)
		if typ != wire.TypeViewDelta {
			t.Fatalf("frame for rev %d = %v, want VIEW_DELTA", wantRev, typ)
		}
		if len(payload) >= 200 {
			t.Fatalf("delta payload = %d bytes, want < 200 on a 10k-node tree", len(payload))
		}
		v, err := wire.UnmarshalPayload(wire.TypeViewDelta, payload)
		if err != nil {
			t.Fatalf("host rejected delta: %v", err)
		}
		delta := v.(*pb.ViewDelta)
		if delta.GetRev() != wantRev || delta.GetRevBase() != wantRev-1 {
			t.Fatalf("delta rev/base = %d/%d, want %d/%d", delta.GetRev(), delta.GetRevBase(), wantRev, wantRev-1)
		}
		if got := len(delta.GetPatches()); got != 1 {
			t.Fatalf("delta patches = %d, want 1 (only the changed subtree)", got)
		}
	}

	_ = hostToProg.Close()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for Run to finish")
	}
}

func countBoxes(box *pb.Box) int {
	if box == nil {
		return 0
	}
	n := 1
	for _, child := range box.GetChildren() {
		n += countBoxes(child)
	}
	return n
}

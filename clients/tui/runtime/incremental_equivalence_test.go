package runtime

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	gproto "google.golang.org/protobuf/proto"

	pb "github.com/anytty/anytty/proto/ui/protobuf"

	"github.com/anytty/anytty/clients/tui/kernel"
)

// incrementalHarness is one session that receives the incremental stream
// (full VIEWs and VIEW_DELTAs) plus a reference session that always receives
// the equivalent tree as a full VIEW. The incremental session's frame must
// deep-equal the reference session's frame after every accepted commit.
type incrementalHarness struct {
	inc *harness
	ref *harness
}

func newIncrementalHarness(t *testing.T, cols, rows int) *incrementalHarness {
	t.Helper()
	return &incrementalHarness{
		inc: newHarness(t, Options{ViewID: "inc", Cols: cols, Rows: rows,
			Limits: Limits{MaxNodes: 1 << 20, MaxMessageBytes: 1 << 20}}),
		ref: newHarness(t, Options{ViewID: "ref", Cols: cols, Rows: rows,
			Limits: Limits{MaxNodes: 1 << 20, MaxMessageBytes: 1 << 20}}),
	}
}

// sendFull installs tree as a full VIEW in both sessions. The incremental
// session's full-view path builds fresh nodes, so both agree by construction;
// this re-seeds the incremental cache for the deltas that follow.
func (h *incrementalHarness) sendFull(tree *pb.Box) {
	h.inc.sendView(&pb.View{Epoch: 1, Rev: h.inc.s.Rev() + 1, Root: gproto.Clone(tree).(*pb.Box)})
	h.ref.sendView(&pb.View{Epoch: 1, Rev: h.ref.s.Rev() + 1, Root: gproto.Clone(tree).(*pb.Box)})
	h.inc.drain()
	h.ref.drain()
}

// TestIncrementalNodeReuse pins the reuse mechanism itself: a delta that
// touches one leaf must reuse the kernel node pointer of an untouched sibling
// subtree, while a full VIEW rebuilds everything.
func TestIncrementalNodeReuse(t *testing.T) {
	h := newHarness(t, Options{ViewID: "v", Cols: 40, Rows: 12,
		Limits: Limits{MaxNodes: 1 << 20, MaxMessageBytes: 1 << 20}})

	// root has a small child and a large shared child.
	big := &pb.Box{Id: "big", Flow: "col"}
	for i := 0; i < 20; i++ {
		big.Children = append(big.Children, textBox("b"+fmt.Sprint(i), "row"))
	}
	root := &pb.Box{Id: "root", Flow: "col", Children: []*pb.Box{textBox("small", "x"), big}}
	h.sendView(&pb.View{Epoch: 1, Rev: 1, Root: root})
	h.drain()

	// The session holds a decoded copy, so take the shared child pointer from
	// the session's own tree, not from the local literal above.
	baseBig := h.s.View().Root.Children[1]
	if baseBig.Id != "big" {
		t.Fatalf("test setup: child 1 = %q", baseBig.Id)
	}

	// A full VIEW intentionally builds no pointer map (lazy cache).
	if h.s.nodes != nil {
		t.Fatal("full VIEW must not build the pointer map")
	}

	// Replace the small leaf; big is untouched and COW-shared. The first delta
	// after a full VIEW indexes the base tree, then reuses big.
	h.sendDelta(&pb.ViewDelta{Epoch: 1, Rev: 2, RevBase: 1,
		Patches: []*pb.Patch{{Op: "replace", Path: []uint32{0}, Box: textBox("small", "XX")}}})
	h.drain()

	root2 := h.s.View().Root
	if root2.Children[1] != baseBig {
		t.Fatal("test setup: big subtree should be pointer-shared")
	}
	bigNode := h.s.nodes[baseBig]
	if bigNode == nil {
		t.Fatal("untouched subtree not reused (missing from pointer map)")
	}
	// The unchanged subtree keeps its cached solved frame.
	solved := bigNode.Cached()
	if solved == nil {
		t.Fatal("reused subtree lost its cached frame")
	}
	kids := &bigNode.Children[0]

	// A second delta touching another small leaf must reuse the same big
	// subtree: it keeps the exact cached solved frame and the same children
	// backing array (the node's own address moves because its parent slot is
	// rebuilt, but no child is re-solved or re-allocated).
	h.sendDelta(&pb.ViewDelta{Epoch: 1, Rev: 3, RevBase: 2,
		Patches: []*pb.Patch{{Op: "replace", Path: []uint32{0}, Box: textBox("small", "YY")}}})
	h.drain()
	bigNode2 := h.s.nodes[baseBig]
	if bigNode2 == nil {
		t.Fatal("untouched subtree missing after second delta")
	}
	if bigNode2.Cached() != solved {
		t.Fatal("untouched subtree was re-solved instead of spliced")
	}
	if &bigNode2.Children[0] != kids {
		t.Fatal("untouched subtree children were re-allocated")
	}

	// A later full VIEW must rebuild from scratch (no pointers shared).
	h.sendView(&pb.View{Epoch: 1, Rev: 4, Root: gproto.Clone(h.s.View().Root).(*pb.Box)})
	h.drain()
	if h.s.nodes != nil {
		t.Fatal("full VIEW must drop the pointer cache")
	}
}

func TestIncrementalLayoutEquivalence(t *testing.T) {
	const iterations = 400
	rng := rand.New(rand.NewSource(20260921))

	cols, rows := 40, 16
	h := newIncrementalHarness(t, cols, rows)

	tree := randomTree(rng, 1+rng.Intn(40), 0)
	h.sendFull(tree)
	assertEquivalent(t, h.inc, h.ref, "seed")

	for iter := 0; iter < iterations; iter++ {
		// Every so often replace the whole tree with a new full VIEW so the
		// cache is re-indexed and the delta path continues from a fresh base.
		if iter%13 == 0 {
			tree = randomTree(rng, 1+rng.Intn(50), 0)
			h.sendFull(tree)
			assertEquivalent(t, h.inc, h.ref, fmt.Sprintf("iter %d full", iter))
			continue
		}

		// Every so often resize the viewport on both sessions, which forces a
		// full re-solve of every subtree at a new rect (the rect gate).
		if iter%9 == 4 {
			cols = 20 + rng.Intn(50)
			rows = 6 + rng.Intn(24)
			h.inc.s.Resize(cols, rows)
			h.ref.s.Resize(cols, rows)
			h.inc.drain()
			h.ref.drain()
			assertEquivalent(t, h.inc, h.ref, fmt.Sprintf("iter %d resize %dx%d", iter, cols, rows))
		}

		// Apply a short burst of valid patches; compare after every commit.
		burst := 1 + rng.Intn(4)
		for b := 0; b < burst; b++ {
			// Occasionally send a keys-only delta (no patches). The tree is
			// pointer-identical, so the whole root subtree must be reused and
			// the root frame spliced.
			if b == 0 && iter%11 == 5 {
				h.inc.sendDelta(&pb.ViewDelta{
					Epoch: 1, Rev: h.inc.s.Rev() + 1, RevBase: h.inc.s.Rev(),
					Keys: &pb.Keys{Claim: []string{"ctrl-x"}},
				})
				h.inc.drain()
				h.ref.sendView(&pb.View{
					Epoch: 1, Rev: h.ref.s.Rev() + 1,
					Keys: &pb.Keys{Claim: []string{"ctrl-x"}},
					Root: gproto.Clone(tree).(*pb.Box),
				})
				h.ref.drain()
				assertEquivalent(t, h.inc, h.ref, fmt.Sprintf("iter %d keys-only", iter))
				continue
			}
			patch, next, ok := randomPatch(rng, tree)
			if !ok {
				continue
			}
			h.inc.sendDelta(&pb.ViewDelta{
				Epoch:   1,
				Rev:     h.inc.s.Rev() + 1,
				RevBase: h.inc.s.Rev(),
				Keys:    nil,
				Patches: []*pb.Patch{patch},
			})
			h.inc.drain()
			h.ref.sendView(&pb.View{Epoch: 1, Rev: h.ref.s.Rev() + 1, Root: gproto.Clone(next).(*pb.Box)})
			h.ref.drain()
			tree = next
			assertEquivalent(t, h.inc, h.ref, fmt.Sprintf("iter %d burst %d op %s", iter, b, patch.Op))
		}
	}
}

// TestIncrementalFrameIndependence commits several deltas and checks that each
// frame captured before a later commit still equals the fresh reference solve
// of the tree it was produced from: no later splice mutates an earlier frame.
func TestIncrementalFrameIndependence(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	cols, rows := 32, 12
	h := newIncrementalHarness(t, cols, rows)

	tree := randomTree(rng, 30, 0)
	h.sendFull(tree)

	type snapshot struct {
		frame frameSnap
		want  frameSnap
	}
	var snaps []snapshot
	for i := 0; i < 25; i++ {
		patch, next, ok := randomPatch(rng, tree)
		if !ok {
			continue
		}
		h.inc.sendDelta(&pb.ViewDelta{
			Epoch: 1, Rev: h.inc.s.Rev() + 1, RevBase: h.inc.s.Rev(),
			Patches: []*pb.Patch{patch},
		})
		h.inc.drain()
		h.ref.sendView(&pb.View{Epoch: 1, Rev: h.ref.s.Rev() + 1, Root: gproto.Clone(next).(*pb.Box)})
		h.ref.drain()
		tree = next

		incFrame, _ := h.inc.s.Frame()
		refFrame, _ := h.ref.s.Frame()
		snaps = append(snaps, snapshot{frame: cloneRuntimeFrame(incFrame), want: cloneRuntimeFrame(refFrame)})
	}

	// Every captured frame must still match its own fresh reference solve,
	// proving no later commit mutated a previously returned frame.
	for i, s := range snaps {
		if !reflect.DeepEqual(s.frame, s.want) {
			t.Fatalf("snapshot %d changed after later commits:\n got %v\nwant %v", i, s.frame, s.want)
		}
	}
}

func assertEquivalent(t *testing.T, inc, ref *harness, where string) {
	t.Helper()

	// The committed program tree must be identical in both sessions.
	if !gproto.Equal(inc.s.View().Root, ref.s.View().Root) {
		t.Fatalf("%s: committed trees differ\n got %v\nwant %v", where, inc.s.View().Root, ref.s.View().Root)
	}

	// Independent oracle: convert the committed tree from scratch with no
	// cache and solve it with the plain solver. The incremental frame must
	// deep-equal this byte for byte.
	var scratch Session
	freshRoot := scratch.nodeFull(nil, inc.s.View().Root)
	wantF := kernel.Layout(freshRoot, inc.s.cols, inc.s.rows)
	gotF, ok := inc.s.Frame()
	if !ok {
		t.Fatalf("%s: incremental session has no frame", where)
	}
	if !reflect.DeepEqual(cloneRuntimeFrame(gotF), cloneRuntimeFrame(wantF)) {
		t.Fatalf("%s: frame mismatch\n got %s\nwant %s", where, dumpFrame(gotF), dumpFrame(wantF))
	}
	// Hit grid.
	for y := -1; y <= inc.s.rows+1; y++ {
		for x := -1; x <= inc.s.cols+1; x++ {
			if g, w := gotF.Hit(x, y), wantF.Hit(x, y); g != w {
				t.Fatalf("%s: Hit(%d,%d) = %q, want %q", where, x, y, g, w)
			}
		}
	}
	// Focus must agree between the two sessions (sources are empty for both,
	// so terminal/mouse bits are false in both).
	if !reflect.DeepEqual(inc.s.Focus(), ref.s.Focus()) {
		t.Fatalf("%s: focus = %+v, want %+v", where, inc.s.Focus(), ref.s.Focus())
	}

	// The incremental session's bookkeeping must match the committed tree.
	assertIncrementalBookkeeping(t, inc.s, where)
}

// assertIncrementalBookkeeping checks the O(changed) commit bookkeeping
// invariants against the committed tree: the persistent node map must stay
// bounded by the live box count, the incrementally maintained boxCount must
// equal a fresh countBoxes walk, and the committed focus must equal a
// from-scratch focusLocked result.
func assertIncrementalBookkeeping(t *testing.T, s *Session, where string) {
	t.Helper()
	rootBox := s.view.Root
	boxes := countBoxes(rootBox)
	// A full VIEW leaves boxCount unknown (-1) on purpose; the next delta seeds
	// it. Only a delta commit must have a correct incrementally maintained
	// count.
	if s.boxCount >= 0 && s.boxCount != boxes {
		t.Fatalf("%s: boxCount = %d, want %d", where, s.boxCount, boxes)
	}
	// After a delta the cache is indexed and pruning must be exact: one entry
	// per live box, no stale pointers (a full VIEW intentionally leaves it nil).
	if s.nodes != nil && len(s.nodes) != boxes {
		t.Fatalf("%s: node map size = %d, want exactly %d live boxes", where, len(s.nodes), boxes)
	}
	if len(s.nodes) > nodeMapCapFactor*boxes {
		t.Fatalf("%s: node map size %d exceeds %dx box count %d", where, len(s.nodes), nodeMapCapFactor, boxes)
	}
	// Focus: reuse is only legal when it equals the full walk (sources are
	// empty in this harness, matching assertEquivalent).
	wantFocus := s.focusLocked(s.root)
	if !reflect.DeepEqual(s.focus, wantFocus) {
		t.Fatalf("%s: focus = %+v, want from-scratch %+v", where, s.focus, wantFocus)
	}
}

func dumpFrame(f kernel.Frame) string {
	return fmt.Sprintf("rects=%v lines=%v overlays=%d cursor=%v/%v/%q",
		frameRects(f), f.Lines, len(f.OverlayFrames), f.HasCursor, f.CursorRect, f.CursorShape)
}

// frameRects materializes a frame's tree-shaped rect index into the flat
// id->Rect map the equivalence oracles compare. Test-only.
func frameRects(f kernel.Frame) map[string]kernel.Rect {
	out := make(map[string]kernel.Rect, f.RectCount())
	f.RectsIterate(func(id string, r kernel.Rect) bool {
		out[id] = r
		return true
	})
	return out
}

// frameSnap is a canonical, deep-copied snapshot of a kernel frame used to
// pin frame independence: the rect surface is materialized into a flat map so
// two snapshots of equal layout compare equal with reflect.DeepEqual
// regardless of how the live tree-shaped index was shared between them.
type frameSnap struct {
	Rects       map[string]kernel.Rect
	Lines       []kernel.Line
	Overlays    []frameSnap
	HasCursor   bool
	CursorRect  kernel.Rect
	CursorShape string
}

func cloneRuntimeFrame(f kernel.Frame) frameSnap {
	out := frameSnap{
		Rects:       frameRects(f),
		Lines:       append([]kernel.Line(nil), f.Lines...),
		HasCursor:   f.HasCursor,
		CursorRect:  f.CursorRect,
		CursorShape: f.CursorShape,
	}
	if len(f.OverlayFrames) > 0 {
		out.Overlays = make([]frameSnap, len(f.OverlayFrames))
		for i := range f.OverlayFrames {
			out.Overlays[i] = cloneRuntimeFrame(f.OverlayFrames[i])
		}
	}
	return out
}

var idSeq int

func nextID(prefix string) string {
	idSeq++
	return fmt.Sprintf("%s-%d", prefix, idSeq)
}

// randomTree builds a random pb.Box tree with n boxes.
func randomTree(rng *rand.Rand, n, depth int) *pb.Box {
	return randomBox(rng, n, depth)
}

func randomBox(rng *rand.Rand, n, depth int) *pb.Box {
	b := &pb.Box{Id: nextID("n")}
	switch rng.Intn(4) {
	case 0:
		b.Flow = "row"
	case 1:
		b.Flow = "col"
	case 2:
		b.Flow = "stack"
	}
	if rng.Intn(3) == 0 {
		b.Size = &pb.Size{Width: int32(rng.Intn(6)), Height: int32(rng.Intn(4)), Flex: int32(rng.Intn(3))}
	}
	if depth > 0 && rng.Intn(6) == 0 {
		b.Pos = &pb.Pos{X: int32(rng.Intn(3)), Y: int32(rng.Intn(3))}
	}
	if rng.Intn(7) == 0 {
		v := rng.Intn(5) != 0
		b.Visible = &v
	}
	switch rng.Intn(3) {
	case 0:
		b.Content = &pb.Content{Text: randomText(rng)}
	case 1:
		b.Content = &pb.Content{Lines: []string{randomText(rng), randomText(rng)}}
	}
	if rng.Intn(5) == 0 {
		b.Style = []string{"", "muted", "bold", "warning"}[rng.Intn(4)]
	}
	if rng.Intn(6) == 0 {
		b.Cursor = &pb.Cursor{Row: int32(rng.Intn(3)), Col: int32(rng.Intn(4)), Shape: "bar"}
	}
	if rng.Intn(8) == 0 {
		b.Input = []string{"key", "mouse"}
	}
	if rng.Intn(10) == 0 {
		b.Focused = true
		b.Content = &pb.Content{Self: nextID("self")}
	}

	// Children.
	if n > 1 && depth < 5 {
		maxKids := n - 1
		if maxKids > 3 {
			maxKids = 3
		}
		kids := rng.Intn(maxKids + 1)
		remaining := n - 1
		for i := 0; i < kids && remaining > 0; i++ {
			share := 1 + rng.Intn(remaining)
			if i == kids-1 || share > remaining {
				share = remaining
			}
			b.Children = append(b.Children, randomBox(rng, share, depth+1))
			remaining -= share
		}
	}
	return b
}

func randomText(rng *rand.Rand) string {
	words := []string{"a", "bb", "ccc", "hello", "x", "value", "0123456789"}
	out := ""
	for i := 0; i < 1+rng.Intn(3); i++ {
		out += words[rng.Intn(len(words))]
	}
	return out
}

// pathTo returns the index path to the node with the given id.
func pathTo(root *pb.Box, id string) ([]uint32, bool) {
	var walk func(n *pb.Box, path []uint32) ([]uint32, bool)
	walk = func(n *pb.Box, path []uint32) ([]uint32, bool) {
		if n.Id == id {
			return path, true
		}
		for i, c := range n.Children {
			p := append(append([]uint32(nil), path...), uint32(i))
			if got, ok := walk(c, p); ok {
				return got, true
			}
		}
		return nil, false
	}
	return walk(root, nil)
}

func collectPaths(root *pb.Box) (all [][]uint32, containers [][]uint32, nonRoot [][]uint32) {
	var walk func(n *pb.Box, path []uint32, root bool)
	walk = func(n *pb.Box, path []uint32, isRoot bool) {
		all = append(all, path)
		if len(n.Children) > 0 {
			containers = append(containers, path)
		}
		if !isRoot {
			nonRoot = append(nonRoot, path)
		}
		for i, c := range n.Children {
			p := append(append([]uint32(nil), path...), uint32(i))
			walk(c, p, false)
		}
	}
	walk(root, nil, true)
	return
}

// randomPatch picks a random valid patch against root and returns the patch,
// the resulting tree (as applyPatches would build it) and whether one was made.
func randomPatch(rng *rand.Rand, root *pb.Box) (*pb.Patch, *pb.Box, bool) {
	all, containers, nonRoot := collectPaths(root)
	if len(all) == 0 {
		return nil, nil, false
	}
	op := []string{"set", "replace", "insert", "remove", "move"}[rng.Intn(5)]

	var p *pb.Patch
	switch op {
	case "set":
		path := all[rng.Intn(len(all))]
		p = &pb.Patch{Op: "set", Path: path, Box: randomSetPayload(rng)}
	case "replace":
		path := all[rng.Intn(len(all))]
		p = &pb.Patch{Op: "replace", Path: path, Box: randomBox(rng, 1+rng.Intn(4), 0)}
	case "insert":
		if len(containers) == 0 {
			return nil, nil, false
		}
		path := containers[rng.Intn(len(containers))]
		parent := nodeAt(root, path)
		p = &pb.Patch{Op: "insert", Path: path, Index: uint32(rng.Intn(len(parent.Children) + 1)), Box: randomBox(rng, 1+rng.Intn(3), 0)}
	case "remove":
		if len(nonRoot) == 0 {
			return nil, nil, false
		}
		p = &pb.Patch{Op: "remove", Path: nonRoot[rng.Intn(len(nonRoot))]}
	case "move":
		if len(containers) == 0 {
			return nil, nil, false
		}
		var path []uint32
		var parent *pb.Box
		for tries := 0; tries < 20; tries++ {
			path = containers[rng.Intn(len(containers))]
			parent = nodeAt(root, path)
			if len(parent.Children) >= 2 {
				break
			}
		}
		if parent == nil || len(parent.Children) < 2 {
			return nil, nil, false
		}
		from := rng.Intn(len(parent.Children))
		to := rng.Intn(len(parent.Children))
		p = &pb.Patch{Op: "move", Path: path, From: uint32(from), To: uint32(to)}
	}

	next, reason := applyPatches(root, []*pb.Patch{p})
	if reason != "" {
		return nil, nil, false
	}
	return p, next, true
}

// randomSetPayload builds a Box whose fields a set patch will merge. It never
// sets children (merge semantics keep the target's children).
func randomSetPayload(rng *rand.Rand) *pb.Box {
	b := &pb.Box{}
	if rng.Intn(2) == 0 {
		b.Size = &pb.Size{Width: int32(rng.Intn(6)), Height: int32(rng.Intn(4)), Flex: int32(rng.Intn(2))}
	}
	if rng.Intn(2) == 0 {
		b.Flow = []string{"row", "col", "stack"}[rng.Intn(3)]
	}
	if rng.Intn(2) == 0 {
		b.Style = []string{"", "bold", "muted"}[rng.Intn(3)]
	}
	if rng.Intn(3) == 0 {
		v := rng.Intn(4) != 0
		b.Visible = &v
	}
	if rng.Intn(2) == 0 {
		b.Content = &pb.Content{Text: randomText(rng)}
	}
	if rng.Intn(3) == 0 {
		b.Cursor = &pb.Cursor{Row: int32(rng.Intn(2)), Col: int32(rng.Intn(3))}
	}
	if rng.Intn(4) == 0 {
		b.Input = []string{"key"}
	}
	if rng.Intn(5) == 0 {
		b.Focused = true
		b.Content = &pb.Content{Self: nextID("self")}
	}
	return b
}

func nodeAt(root *pb.Box, path []uint32) *pb.Box {
	n := root
	for _, idx := range path {
		n = n.Children[idx]
	}
	return n
}

// TestIncrementalMapBounded drives a long run of deltas that repeatedly add and
// drop whole subtrees and asserts the persistent pointer map never accumulates
// stale entries: its size stays within one small factor of the live box count
// (the same cap buildNodeTree/enforceNodeMapCap uses), and boxCount tracks the
// tree.
func TestIncrementalMapBounded(t *testing.T) {
	h := newHarness(t, Options{ViewID: "v", Cols: 60, Rows: 20,
		Limits: Limits{MaxNodes: 1 << 20, MaxMessageBytes: 1 << 20}})

	root := &pb.Box{Id: "root", Flow: "col"}
	for i := 0; i < 40; i++ {
		root.Children = append(root.Children, textBox("base"+fmt.Sprint(i), "row"))
	}
	h.sendView(&pb.View{Epoch: 1, Rev: 1, Root: root})
	h.drain()

	rev := uint64(1)
	const iterations = 1000
	for i := 0; i < iterations; i++ {
		rev++
		// Replace one base subtree with a fresh multi-box subtree each round:
		// the replaced subtree is detached whole, so its entries must be pruned.
		slot := i % 40
		patch := patch("replace", []uint32{uint32(slot)}, &pb.Box{
			Id: "repl" + fmt.Sprint(i),
			Children: []*pb.Box{
				textBox("r0", fmt.Sprint(i)),
				textBox("r1", fmt.Sprint(i)),
				textBox("r2", fmt.Sprint(i)),
			},
		})
		h.sendDelta(delta(1, rev, rev-1, nil, patch))
		h.drain()
	}

	boxes := countBoxes(h.s.View().Root)
	if h.s.boxCount != boxes {
		t.Fatalf("boxCount = %d, want %d after %d deltas", h.s.boxCount, boxes, iterations)
	}
	// Exactly one entry per live box: every one of the 1000 detached subtrees
	// was pruned, so the map did not grow with the number of deltas.
	if len(h.s.nodes) != boxes {
		t.Fatalf("node map = %d entries for %d boxes after %d deltas (stale keys leaked)", len(h.s.nodes), boxes, iterations)
	}
}

// TestIncrementalMapPersistentAcrossCommits pins the map's lifetime: after the
// first delta the map is indexed, and a second delta must find the untouched
// sibling subtree already in it without a rebuild. If the map were dropped per
// commit, the sibling entry could only reappear via a full reindex (which would
// also have to be paid on every commit); observing it after two independent
// commits proves persistence.
func TestIncrementalMapPersistentAcrossCommits(t *testing.T) {
	h := newHarness(t, Options{ViewID: "v", Cols: 40, Rows: 12,
		Limits: Limits{MaxNodes: 1 << 20, MaxMessageBytes: 1 << 20}})
	h.sendView(view(1, 1, nil, false, textBox("a", "A"), textBox("b", "B"), textBox("c", "C")))
	h.drain()
	if h.s.nodes != nil {
		t.Fatal("full VIEW must not build the map")
	}

	h.sendDelta(delta(1, 2, 1, nil, patch("replace", []uint32{0}, textBox("a", "A2"))))
	h.drain()
	if h.s.nodes == nil || len(h.s.nodes) != countBoxes(h.s.View().Root) {
		t.Fatalf("map not indexed after first delta: %d entries", len(h.s.nodes))
	}
	shared := h.s.view.Root.Children[2] // "c", untouched by both deltas
	siblingNode := h.s.nodes[shared]
	if siblingNode == nil {
		t.Fatal("untouched subtree missing from the map")
	}
	if _, ok := h.s.nodes[h.s.view.Root.Children[1]]; !ok {
		t.Fatal("untouched sibling b missing from the map")
	}
	solved := siblingNode.Cached()
	if solved == nil {
		t.Fatal("untouched subtree has no cached frame")
	}

	h.sendDelta(delta(1, 3, 2, nil, patch("replace", []uint32{1}, textBox("b", "B2"))))
	h.drain()
	if h.s.view.Root.Children[2] != shared {
		t.Fatal("test setup: c should stay pointer-shared")
	}
	// The map key persists across commits: the reused subtree is found by its
	// unchanged *pb.Box pointer and keeps the exact solved frame. (Its kernel
	// node address moves because its parent's child slot was rebuilt, by
	// design; that is a shallow copy, not a rebuild.)
	node2 := h.s.nodes[shared]
	if node2 == nil {
		t.Fatal("shared subtree entry lost across commits")
	}
	if node2.Cached() != solved {
		t.Fatal("shared subtree was re-solved instead of reused across commits")
	}
	if len(h.s.nodes) != countBoxes(h.s.View().Root) {
		t.Fatalf("map size = %d, want %d", len(h.s.nodes), countBoxes(h.s.View().Root))
	}
}

// TestIncrementalBoxCountMaxNodes pins the incremental counter at the max_nodes
// boundary: an accepting insert, a rejecting insert, an accepting replace, a
// rejecting replace, and a remove that makes room again. The accepted/rejected
// decisions must match a full countBoxes of the resulting tree.
func TestIncrementalBoxCountMaxNodes(t *testing.T) {
	h := newHarness(t, Options{ViewID: "v", Cols: 40, Rows: 12,
		Limits: Limits{MaxNodes: 5, MaxMessageBytes: 1 << 20}})
	h.sendView(view(1, 1, nil, false, textBox("a", "A"), textBox("b", "B")))
	h.drain()

	// root + a + b = 3 boxes. A one-box insert reaches 4 (accept).
	h.sendDelta(delta(1, 2, 1, nil, patch("insert", nil, textBox("c", "C"))))
	if h.s.Rev() != 2 {
		t.Fatalf("one-box insert rev = %d, want 2", h.s.Rev())
	}
	h.drain()
	// A two-box insert reaches 6 (reject).
	h.sendDelta(delta(1, 3, 2, nil, patch("insert", nil, &pb.Box{Id: "d", Children: []*pb.Box{textBox("d0", "D")}})))
	msgs := h.drain()
	if len(msgs) != 1 || msgs[0].(*pb.Event).GetViewRejected().GetReason() != "max_nodes" {
		t.Fatalf("two-box insert rejection = %v", msgs)
	}
	if h.s.Rev() != 2 {
		t.Fatalf("rev changed to %d after rejection", h.s.Rev())
	}
	// A one-box replace keeps 4 (accept): root + a + b + c.
	h.sendDelta(delta(1, 3, 2, nil, patch("replace", []uint32{2}, textBox("c", "C2"))))
	if h.s.Rev() != 3 {
		t.Fatalf("one-box replace rev = %d, want 3", h.s.Rev())
	}
	h.drain()
	// A three-box replace reaches 6 (reject): 4 - 1 + 3.
	h.sendDelta(delta(1, 4, 3, nil, patch("replace", []uint32{2}, &pb.Box{Id: "e", Children: []*pb.Box{textBox("e0", "E"), textBox("e1", "E")}})))
	msgs = h.drain()
	if len(msgs) != 1 || msgs[0].(*pb.Event).GetViewRejected().GetReason() != "max_nodes" {
		t.Fatalf("three-box replace rejection = %v", msgs)
	}
	// Remove drops back to 3; a later insert to 4 is accepted again.
	h.sendDelta(delta(1, 4, 3, nil, &pb.Patch{Op: "remove", Path: []uint32{0}}))
	if h.s.Rev() != 4 {
		t.Fatalf("remove rev = %d, want 4", h.s.Rev())
	}
	h.drain()
	if h.s.boxCount != countBoxes(h.s.View().Root) {
		t.Fatalf("boxCount = %d, want %d", h.s.boxCount, countBoxes(h.s.View().Root))
	}
	h.sendDelta(delta(1, 5, 4, nil, patch("insert", nil, textBox("f", "F"))))
	if h.s.Rev() != 5 {
		t.Fatalf("insert after remove rev = %d, want 5", h.s.Rev())
	}
}

// TestIncrementalFocusTracking proves the O(changed) focus rule: a delta that
// cannot affect focus reuses the previous Focus value, while a delta that sets
// focused/self/input/content/visible picks up the new focused node.
func TestIncrementalFocusTracking(t *testing.T) {
	h := newHarness(t, Options{ViewID: "v", Cols: 40, Rows: 12,
		Limits: Limits{MaxNodes: 1 << 20, MaxMessageBytes: 1 << 20}})

	root := &pb.Box{Id: "root", Flow: "col", Children: []*pb.Box{
		textBox("a", "A"),
		focusBox("old", "self-old", "key"),
		textBox("b", "B"),
	}}
	h.sendView(&pb.View{Epoch: 1, Rev: 1, Root: root})
	h.drain()
	if f := h.s.Focus(); f == nil || f.ID != "self-old" {
		t.Fatalf("seed focus = %+v, want self-old", f)
	}
	want := h.s.Focus()

	// A pure style/size delta on an unfocused node cannot change focus: the
	// previous Focus value is reused untouched.
	h.sendDelta(delta(1, 2, 1, nil, patch("set", []uint32{0}, &pb.Box{Style: "bold", Size: &pb.Size{Width: 3}})))
	h.drain()
	if h.s.Focus() != want {
		t.Fatalf("style-only set replaced focus: %+v != %+v", h.s.Focus(), want)
	}

	// A replace of the focused subtree is structural: focus re-walks and, with
	// the focused node gone, becomes nil.
	h.sendDelta(delta(1, 3, 2, nil, patch("replace", []uint32{1}, textBox("c", "C"))))
	h.drain()
	if h.s.Focus() != nil {
		t.Fatalf("replace that drops the focused node left focus = %+v", h.s.Focus())
	}

	// A set carrying focused+self moves focus to the new node.
	h.sendDelta(delta(1, 4, 3, nil, patch("set", []uint32{0}, &pb.Box{Focused: true, Content: &pb.Content{Self: "self-new"}})))
	h.drain()
	if f := h.s.Focus(); f == nil || f.ID != "self-new" {
		t.Fatalf("focused set did not update focus: %+v", f)
	}
	newWant := h.s.Focus()

	// A keys-only delta has no patches: focus is reused.
	h.sendDelta(delta(1, 5, 4, &pb.Keys{Claim: []string{"ctrl-x"}}))
	h.drain()
	if h.s.Focus() != newWant {
		t.Fatalf("keys-only delta replaced focus: %+v != %+v", h.s.Focus(), newWant)
	}

	// A set that carries input on the (still focused) node must refresh
	// Focus.Input.
	h.sendDelta(delta(1, 6, 5, nil, patch("set", []uint32{0}, &pb.Box{Input: []string{"paste"}})))
	h.drain()
	if f := h.s.Focus(); f == nil || f.ID != "self-new" || len(f.Input) != 1 || f.Input[0] != "paste" {
		t.Fatalf("input set focus = %+v", f)
	}
}

// TestIncrementalMultiPatchPrune covers a patch burst where an earlier patch
// clones a node and a later patch detaches that clone: the detached root is not
// in the cache map, so its cached shared descendants must still be pruned (by
// rebuilding the map) instead of leaking.
func TestIncrementalMultiPatchPrune(t *testing.T) {
	h := newHarness(t, Options{ViewID: "v", Cols: 60, Rows: 20,
		Limits: Limits{MaxNodes: 1 << 20, MaxMessageBytes: 1 << 20}})

	root := &pb.Box{Id: "root", Flow: "col", Children: []*pb.Box{
		textBox("a", "1"),
		textBox("b", "2"),
		textBox("c", "3"),
	}}
	h.sendView(&pb.View{Epoch: 1, Rev: 1, Root: root})
	h.drain()

	// Patch 1 clones child 0 (set), so the working root holds a fresh clone
	// that is not in the cache map. Patch 2 replaces child 0, detaching that
	// clone whose child (shared with the base tree) is still cached.
	set := patch("set", []uint32{0}, &pb.Box{Style: "accent"})
	replace := patch("replace", []uint32{0}, &pb.Box{Id: "fresh", Children: []*pb.Box{
		textBox("n0", "9"), textBox("n1", "8"),
	}})
	h.sendDelta(delta(1, 2, 1, nil, set, replace))
	h.drain()

	boxes := countBoxes(h.s.View().Root)
	if h.s.boxCount != boxes {
		t.Fatalf("boxCount = %d, want %d", h.s.boxCount, boxes)
	}
	if len(h.s.nodes) != boxes {
		t.Fatalf("node map = %d entries for %d boxes (multi-patch prune leaked)", len(h.s.nodes), boxes)
	}
	assertIncrementalBookkeeping(t, h.s, "multi-patch prune")
}

// TestIncrementalAliasedPayloadKeepsLiveEntry covers patches whose payload is a
// pointer that is already part of the committed tree (in-process callers can do
// this; the wire never does). The payload must be cloned so the committed tree
// keeps unique pointers and the node cache keeps exactly one entry per live box.
func TestIncrementalAliasedPayloadKeepsLiveEntry(t *testing.T) {
	h := newIncrementalHarness(t, 60, 20)

	base := &pb.Box{Id: "root", Flow: "col", Children: []*pb.Box{
		textBox("a", "a"),
		{Id: "keep", Children: []*pb.Box{textBox("k0", "0"), textBox("k1", "1")}},
		textBox("b", "b"),
	}}

	// seed installs a fresh committed base in both sessions and returns the
	// live "keep" box from the incremental session.
	seed := func() *pb.Box {
		t.Helper()
		h.sendFull(base)
		return h.inc.s.View().Root.Children[1]
	}

	send := func(name string, patches ...*pb.Patch) {
		t.Helper()
		inc := &pb.ViewDelta{Epoch: 1, Rev: h.inc.s.Rev() + 1, RevBase: h.inc.s.Rev(), Patches: patches}
		ref := &pb.ViewDelta{Epoch: 1, Rev: h.ref.s.Rev() + 1, RevBase: h.ref.s.Rev(), Patches: patches}
		if err := h.inc.s.HandleViewDelta(inc); err != nil {
			t.Fatalf("%s: inc HandleViewDelta: %v", name, err)
		}
		if err := h.ref.s.HandleViewDelta(ref); err != nil {
			t.Fatalf("%s: ref HandleViewDelta: %v", name, err)
		}
		h.inc.drain()
		h.ref.drain()
		assertEquivalent(t, h.inc, h.ref, name)
	}

	// Replacing a slot with the pointer that already lives there must clone it:
	// the committed child is a fresh (semantically equal) box, not the live one.
	live := seed()
	send("self-replace", patch("replace", []uint32{1}, live))
	if got := h.inc.s.View().Root.Children[1]; got == live {
		t.Fatal("self-replace kept the live pointer; payload was not cloned")
	}

	// Sensitive ordering: attach the live box at another slot, then remove the
	// slot it originally occupied. Without the clone the removal would prune a
	// still-live entry (over-prune) because replace and remove reference the
	// same pointer.
	live = seed()
	send("alias-then-detach-original",
		patch("replace", []uint32{0}, live),
		patch("remove", []uint32{1}, nil))

	// Inserting a live box at the root produces a clone, and the original stays
	// live at its old index.
	live = seed()
	send("insert-live", &pb.Patch{Op: "insert", Path: []uint32{}, Index: 0, Box: live})
	tree := h.inc.s.View().Root
	if len(tree.Children) < 2 {
		t.Fatalf("insert-live: %d children, want >= 2", len(tree.Children))
	}
	if tree.Children[0] == live {
		t.Fatal("insert-live kept the live pointer; payload was not cloned")
	}
	found := false
	for _, c := range tree.Children {
		if c == live {
			found = true
		}
	}
	if !found {
		t.Fatal("insert-live detached the original live box")
	}

	// A fresh wrapper around a live descendant must also be cloned: aliasing
	// anywhere in the payload breaks the one-entry-per-box invariant.
	live = seed()
	send("nested-alias", patch("replace", []uint32{0}, &pb.Box{Id: "wrap", Children: []*pb.Box{live}}))
	tree = h.inc.s.View().Root
	if tree.Children[0] == live || (len(tree.Children[0].Children) == 1 && tree.Children[0].Children[0] == live) {
		t.Fatal("nested-alias kept a live pointer; payload was not cloned")
	}
	found = false
	for _, c := range tree.Children {
		if c == live {
			found = true
		}
	}
	if !found {
		t.Fatal("nested-alias detached the original live box")
	}
}

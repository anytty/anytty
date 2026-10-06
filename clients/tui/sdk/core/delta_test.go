package core

import (
	"bytes"
	"fmt"
	"testing"

	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
	gproto "google.golang.org/protobuf/proto"
)

// applyPatches is a reference implementation of PROTOCOL §2.1 patch
// application. It is deliberately independent of the diff so a test can prove
// DiffView's output reconstructs the target exactly.
func applyPatches(t *testing.T, root *pb.Box, patches []*pb.Patch) *pb.Box {
	t.Helper()
	tree := gproto.Clone(root).(*pb.Box)
	for _, patch := range patches {
		switch patch.GetOp() {
		case OpSet:
			target := nodeAt(t, tree, patch.GetPath(), true)
			mergeSet(t, target, patch.GetBox())
		case OpReplace:
			if len(patch.GetPath()) == 0 {
				// Replace at the empty path replaces the root in place. Copy
				// field by field: assigning a protobuf message would copy its
				// internal lock.
				replaced := gproto.Clone(patch.GetBox()).(*pb.Box)
				copyBox(tree, replaced)
				break
			}
			parent, index := parentAndIndex(t, tree, patch.GetPath())
			parent.Children[index] = gproto.Clone(patch.GetBox()).(*pb.Box)
		case OpInsert:
			parent := nodeAt(t, tree, patch.GetPath(), false)
			index := int(patch.GetIndex())
			if index < 0 || index > len(parent.Children) {
				t.Fatalf("insert index %d out of range", index)
			}
			box := gproto.Clone(patch.GetBox()).(*pb.Box)
			parent.Children = append(parent.Children, nil)
			copy(parent.Children[index+1:], parent.Children[index:])
			parent.Children[index] = box
		case OpRemove:
			parent, index := parentAndIndex(t, tree, patch.GetPath())
			if parent == nil {
				t.Fatal("cannot remove the root")
			}
			if index < 0 || index >= len(parent.Children) {
				t.Fatalf("remove index %d out of range", index)
			}
			parent.Children = append(parent.Children[:index], parent.Children[index+1:]...)
		default:
			t.Fatalf("unknown op %q", patch.GetOp())
		}
	}
	return tree
}

// copyBox overwrites dst with src field by field, avoiding a protobuf message
// assignment (which would copy the internal message lock).
func copyBox(dst, src *pb.Box) {
	dst.Id = src.Id
	dst.Size = src.Size
	dst.Pos = src.Pos
	dst.Flow = src.Flow
	dst.Visible = src.Visible
	dst.Cursor = src.Cursor
	dst.Content = src.Content
	dst.Input = src.Input
	dst.Focused = src.Focused
	dst.Children = src.Children
	dst.Style = src.Style
}

// mergeSet applies a set patch: unset fields keep their old value and the
// patch box's children are ignored (PROTOCOL §2.1).
func mergeSet(t *testing.T, target, patch *pb.Box) {
	t.Helper()
	if patch.GetId() != "" {
		target.Id = patch.GetId()
	}
	if patch.GetFlow() != "" {
		target.Flow = patch.GetFlow()
	}
	if patch.GetStyle() != "" {
		target.Style = patch.GetStyle()
	}
	if patch.GetFocused() {
		target.Focused = true
	}
	if patch.Visible != nil {
		visible := *patch.Visible
		target.Visible = &visible
	}
	if len(patch.GetInput()) > 0 {
		if len(target.GetInput()) == 0 {
			target.Input = append([]string(nil), patch.GetInput()...)
		} else if !equalStrings(target.GetInput(), patch.GetInput()) {
			t.Fatalf("set input merge would concatenate: base=%v patch=%v", target.GetInput(), patch.GetInput())
		}
	}
	if patch.GetSize() != nil {
		if target.Size == nil {
			target.Size = &pb.Size{}
		}
		if v := patch.GetSize().GetWidth(); v != 0 {
			target.Size.Width = v
		}
		if v := patch.GetSize().GetHeight(); v != 0 {
			target.Size.Height = v
		}
		if v := patch.GetSize().GetFlex(); v != 0 {
			target.Size.Flex = v
		}
	}
	if patch.GetPos() != nil {
		if target.Pos == nil {
			target.Pos = &pb.Pos{}
		}
		if v := patch.GetPos().GetX(); v != 0 {
			target.Pos.X = v
		}
		if v := patch.GetPos().GetY(); v != 0 {
			target.Pos.Y = v
		}
	}
	if patch.GetCursor() != nil {
		if target.Cursor == nil {
			target.Cursor = &pb.Cursor{}
		}
		if v := patch.GetCursor().GetRow(); v != 0 {
			target.Cursor.Row = v
		}
		if v := patch.GetCursor().GetCol(); v != 0 {
			target.Cursor.Col = v
		}
		if v := patch.GetCursor().GetShape(); v != "" {
			target.Cursor.Shape = v
		}
	}
	if patch.GetContent() != nil {
		if target.Content == nil {
			target.Content = &pb.Content{}
		}
		if v := patch.GetContent().GetText(); v != "" {
			target.Content.Text = v
		}
		if v := patch.GetContent().GetSelf(); v != "" {
			target.Content.Self = v
		}
		if v := patch.GetContent().GetLines(); len(v) > 0 {
			if len(target.Content.GetLines()) > 0 && !equalStrings(target.Content.GetLines(), v) {
				t.Fatalf("set lines merge would concatenate: base=%v patch=%v", target.Content.GetLines(), v)
			}
			target.Content.Lines = append([]string(nil), v...)
		}
		for key, value := range patch.GetContent().GetProps() {
			if target.Content.Props == nil {
				target.Content.Props = map[string]string{}
			}
			target.Content.Props[key] = value
		}
	}
}

func nodeAt(t *testing.T, root *pb.Box, path []uint32, wantLeaf bool) *pb.Box {
	t.Helper()
	node := root
	for _, index := range path {
		if int(index) >= len(node.GetChildren()) {
			t.Fatalf("path %v out of range (%d children)", path, len(node.GetChildren()))
		}
		node = node.GetChildren()[index]
	}
	if wantLeaf && len(node.GetChildren()) > 0 {
		// A set on a container is legal; children are ignored. No assertion.
	}
	return node
}

func parentAndIndex(t *testing.T, root *pb.Box, path []uint32) (*pb.Box, int) {
	t.Helper()
	if len(path) == 0 {
		return nil, -1
	}
	parent := nodeAt(t, root, path[:len(path)-1], false)
	index := int(path[len(path)-1])
	if index >= len(parent.GetChildren()) {
		t.Fatalf("path %v out of range (%d children)", path, len(parent.GetChildren()))
	}
	return parent, index
}

// assertRoundTrip checks that applying DiffView(base, next) to base yields next.
func assertRoundTrip(t *testing.T, base, next *pb.Box) {
	t.Helper()
	patches, ok := DiffView(base, next)
	identical := gproto.Equal(base, next)
	if identical {
		if ok {
			t.Fatalf("identical trees produced %d patches", len(patches))
		}
		return
	}
	if !ok {
		t.Fatalf("DiffView returned ok=false for a non-identical change")
	}
	applied := applyPatches(t, base, patches)
	// A set that unhides a box carries explicit visible=true; the target may
	// hold the implicit default (nil). They are protocol-equivalent, so compare
	// after normalising presence.
	if !gproto.Equal(normalizeVisible(applied), normalizeVisible(next)) {
		t.Fatalf("patched tree != next\npatches=%v\ngot=%v\nwant=%v", describe(patches), applied, next)
	}
}

// normalizeVisible clones box and rewrites every explicit visible=true to the
// implicit default (nil), so a merge that materialises presence still compares
// equal. The input is not mutated.
func normalizeVisible(box *pb.Box) *pb.Box {
	clone := gproto.Clone(box).(*pb.Box)
	var walk func(*pb.Box)
	walk = func(node *pb.Box) {
		if node == nil {
			return
		}
		if node.Visible != nil && *node.Visible {
			node.Visible = nil
		}
		for _, child := range node.GetChildren() {
			walk(child)
		}
	}
	walk(clone)
	return clone
}

func describe(patches []*pb.Patch) string {
	var b bytes.Buffer
	for _, p := range patches {
		fmt.Fprintf(&b, "%s%v(index=%d,box=%v) ", p.GetOp(), p.GetPath(), p.GetIndex(), p.GetBox())
	}
	return b.String()
}

func box(id string, children ...*pb.Box) *pb.Box {
	return &pb.Box{Id: id, Children: children}
}

func text(id, value string) *pb.Box {
	return &pb.Box{Id: id, Content: &pb.Content{Text: value}}
}

func TestDiffViewIdentity(t *testing.T) {
	tree := text("a", "hello")
	if _, ok := DiffView(tree, gproto.Clone(tree).(*pb.Box)); ok {
		t.Fatal("clone of an identical tree should produce no patches")
	}
	if _, ok := DiffView(nil, tree); ok {
		t.Fatal("nil base must yield ok=false")
	}
	if _, ok := DiffView(tree, nil); ok {
		t.Fatal("nil next must yield ok=false")
	}
}

func TestDiffViewScalarAndContent(t *testing.T) {
	base := box("root", text("a", "one"), text("b", "two"))
	next := gproto.Clone(base).(*pb.Box)
	next.GetChildren()[0].Content.Text = "ONE"
	next.Style = "bold"
	assertRoundTrip(t, base, next)

	patches, ok := DiffView(base, next)
	if !ok {
		t.Fatal("expected patches")
	}
	if len(patches) != 2 {
		t.Fatalf("patches = %d, want 2 (root style set, leaf content set): %s", len(patches), describe(patches))
	}
}

func TestDiffViewClearForcesReplace(t *testing.T) {
	// proto3 set cannot clear a scalar, so a cleared style must become replace.
	base := box("root", &pb.Box{Id: "a", Style: "bold", Content: &pb.Content{Text: "x"}})
	next := box("root", &pb.Box{Id: "a", Content: &pb.Content{}})
	assertRoundTrip(t, base, next)

	patches, _ := DiffView(base, next)
	if len(patches) != 1 || patches[0].GetOp() != OpReplace || len(patches[0].GetPath()) != 1 {
		t.Fatalf("patches = %s, want one replace at [0]", describe(patches))
	}
}

func TestDiffViewInsertRemoveTail(t *testing.T) {
	base := box("root", text("a", "1"), text("b", "2"))
	grown := box("root", text("a", "1"), text("b", "2"), text("c", "3"))
	assertRoundTrip(t, base, grown)
	patches, _ := DiffView(base, grown)
	if len(patches) != 1 || patches[0].GetOp() != OpInsert || patches[0].GetIndex() != 2 {
		t.Fatalf("grow patches = %s, want one insert at 2", describe(patches))
	}

	shrunk := box("root", text("a", "1"))
	assertRoundTrip(t, base, shrunk)
	patches, _ = DiffView(base, shrunk)
	if len(patches) != 1 || patches[0].GetOp() != OpRemove || len(patches[0].GetPath()) != 1 {
		t.Fatalf("shrink patches = %s, want one remove of path [1]", describe(patches))
	}
}

func TestDiffViewInsertMiddle(t *testing.T) {
	// A middle insert keeps the common suffix stable and uses insert, not a
	// chain of replaces.
	base := box("root", text("a", "1"), text("c", "3"))
	next := box("root", text("a", "1"), text("b", "2"), text("c", "3"))
	assertRoundTrip(t, base, next)
	patches, _ := DiffView(base, next)
	if len(patches) != 1 || patches[0].GetOp() != OpInsert || patches[0].GetIndex() != 1 {
		t.Fatalf("middle insert patches = %s, want one insert at 1", describe(patches))
	}
}

func TestDiffViewSubtreeChange(t *testing.T) {
	base := box("root", box("p", text("x", "1"), text("y", "2")), text("z", "3"))
	next := gproto.Clone(base).(*pb.Box)
	next.GetChildren()[0].GetChildren()[1].Content.Text = "TWO"
	assertRoundTrip(t, base, next)
	patches, _ := DiffView(base, next)
	if len(patches) != 1 || patches[0].GetOp() != OpSet || len(patches[0].GetPath()) != 2 {
		t.Fatalf("subtree change patches = %s, want one set at [0 1]", describe(patches))
	}
}

func TestDiffViewShapeChangeRemovesAnchoredChild(t *testing.T) {
	// Dropping the middle child of p: the prefix (x) still lines up, so the
	// edit is a remove at the prefix boundary, not a whole-subtree replace.
	base := box("root", box("p", text("x", "1"), text("y", "2")), text("z", "3"))
	next := gproto.Clone(base).(*pb.Box)
	next.GetChildren()[0] = box("p", text("x", "1"))
	assertRoundTrip(t, base, next)
	patches, _ := DiffView(base, next)
	if len(patches) != 1 || patches[0].GetOp() != OpRemove || len(patches[0].GetPath()) != 2 {
		t.Fatalf("shape change patches = %s, want one remove at [0 1]", describe(patches))
	}
}

func TestDiffViewShapeChangeReplacesUnanchoredChild(t *testing.T) {
	// Nothing in the child lines up: a subtree replace.
	base := box("root", box("p", text("x", "1"), text("y", "2")), text("z", "3"))
	next := gproto.Clone(base).(*pb.Box)
	next.GetChildren()[0] = box("q", text("w", "9"))
	assertRoundTrip(t, base, next)
	patches, _ := DiffView(base, next)
	if len(patches) != 1 || patches[0].GetOp() != OpReplace || len(patches[0].GetPath()) != 1 {
		t.Fatalf("unanchored shape change patches = %s, want one replace at [0]", describe(patches))
	}
}

func TestDiffViewVisible(t *testing.T) {
	hidden := false
	base := box("root", text("a", "1"), text("b", "2"))
	next := gproto.Clone(base).(*pb.Box)
	next.GetChildren()[0].Visible = &hidden
	assertRoundTrip(t, base, next)

	patches, _ := DiffView(base, next)
	if len(patches) != 1 || patches[0].GetOp() != OpSet {
		t.Fatalf("hide patches = %s, want one set", describe(patches))
	}
	if set := patches[0].GetBox(); set.Visible == nil || set.GetVisible() {
		t.Fatalf("hide set visible = %v, want explicit false", set.Visible)
	}

	// Unhide (explicit false -> default true) must stay a set and carry the
	// explicit true, because a zero-omitting set would otherwise not clear the
	// false.
	unhide := gproto.Clone(next).(*pb.Box)
	unhide.GetChildren()[0].Visible = nil
	assertRoundTrip(t, next, unhide)
	patches, _ = DiffView(next, unhide)
	if len(patches) != 1 || patches[0].GetOp() != OpSet {
		t.Fatalf("unhide patches = %s, want one set", describe(patches))
	}
	if set := patches[0].GetBox(); set.Visible == nil || !set.GetVisible() {
		t.Fatalf("unhide set visible = %v, want explicit true", set.Visible)
	}
}

// TestDiffViewDeepTree round-trips a deep, mixed change on a large tree.
func TestDiffViewDeepTree(t *testing.T) {
	base := bigTree(2000)
	next := gproto.Clone(base).(*pb.Box)
	// Change one container a few levels down and hide another.
	descend(next, 0, 0).Style = "changed"
	descend(next, 4, 4).Visible = protoBool(false)
	assertRoundTrip(t, base, next)

	patches, _ := DiffView(base, next)
	if len(patches) != 2 {
		t.Fatalf("patches = %d, want 2 (one set, one set for visible): %s", len(patches), describe(patches))
	}
}

func TestDiffViewUnrepresentableBase(t *testing.T) {
	// A nil child has no addressable index: the diff reports ok=false so the
	// caller falls back to a full VIEW.
	base := &pb.Box{Id: "root", Children: []*pb.Box{nil}}
	next := &pb.Box{Id: "root", Children: []*pb.Box{text("a", "1")}}
	if _, ok := DiffView(base, next); ok {
		t.Fatal("a nil child that changed should force the full-VIEW fallback")
	}
	// A nil child that is unchanged (or removed) still produces an exact diff.
	if _, ok := DiffView(base, &pb.Box{Id: "root"}); !ok {
		t.Fatal("removing a nil child is expressible")
	}
}

func protoBool(v bool) *bool { return &v }

// bigTree builds a deterministic fan-out-5 tree with roughly n nodes.
func bigTree(n int) *pb.Box {
	if n <= 1 {
		return text("leaf", "v")
	}
	root := box("root")
	rest := n - 1
	children := 5
	if rest < children {
		children = rest
	}
	base, extra := rest/children, rest%children
	for i := 0; i < children; i++ {
		count := base
		if i < extra {
			count++
		}
		root.Children = append(root.Children, bigTree(count))
	}
	return root
}

// descend follows the first index of each level.
func descend(box *pb.Box, indices ...int) *pb.Box {
	for _, index := range indices {
		box = box.GetChildren()[index]
	}
	return box
}

// --- CommitDelta ---

// helloWith writes one HELLO (with the delta feature flag) into a reader and
// returns a client whose Loop has already consumed it plus the write buffer.
func helloWith(t *testing.T, viewDelta bool) (*Client, *bytes.Buffer) {
	t.Helper()
	features := map[string]bool{}
	if viewDelta {
		features[ViewDeltaFeature] = true
	}
	var in bytes.Buffer
	if err := wire.NewEncoder(&in, wire.RoleHost, 0).Encode(wire.TypeHello, &pb.Hello{
		Schema: 1, ViewId: "v", Epoch: 1, Cols: 80, Rows: 24, Features: features,
	}); err != nil {
		t.Fatalf("hello: %v", err)
	}
	out := &bytes.Buffer{}
	client := New(&in, out, Handlers{})
	if err := client.Loop(); err != nil {
		t.Fatalf("loop: %v", err)
	}
	return client, out
}

func decodeNext(t *testing.T, out *bytes.Buffer) (wire.Type, []byte) {
	t.Helper()
	buf := out.Bytes()
	typ, payload, err := wire.DecodeFrame(buf, wire.RoleHost, 0)
	if err != nil {
		t.Fatalf("decode frame: %v", err)
	}
	out.Next(4 + len(payload) + 1)
	return typ, payload
}

func TestSupportsFeature(t *testing.T) {
	client, _ := helloWith(t, true)
	if !client.Supports(ViewDeltaFeature) {
		t.Fatal("Supports(view_delta) = false, want true")
	}
	if client.Supports("component") {
		t.Fatal("Supports(component) = true, want false for an absent feature")
	}
}

func TestCommitDeltaUsesPatchesThenFallsBackOnWholeTreeChange(t *testing.T) {
	client, out := helloWith(t, true)
	base := bigTree(1000)
	if err := client.Commit(base, Keys{}); err != nil {
		t.Fatalf("commit: %v", err)
	}
	typ, _ := decodeNext(t, out)
	if typ != wire.TypeView {
		t.Fatalf("first frame = %v, want VIEW", typ)
	}
	if !client.HasBase() {
		t.Fatal("HasBase = false after a full commit")
	}

	// One container edit: a delta.
	next := gproto.Clone(base).(*pb.Box)
	descend(next, 0, 0).Style = "edit"
	sent, err := client.CommitDelta(base, next, Keys{})
	if err != nil {
		t.Fatalf("commit delta: %v", err)
	}
	if !sent {
		t.Fatal("one-leaf edit fell back to a full VIEW")
	}
	typ, payload := decodeNext(t, out)
	if typ != wire.TypeViewDelta {
		t.Fatalf("frame = %v, want VIEW_DELTA", typ)
	}
	delta := decodeDelta(t, payload)
	if delta.GetRev() != 2 || delta.GetRevBase() != 1 {
		t.Fatalf("delta rev/base = %d/%d, want 2/1", delta.GetRev(), delta.GetRevBase())
	}
	if got := applyPatches(t, base, delta.GetPatches()); !gproto.Equal(got, next) {
		t.Fatal("delta did not reconstruct next")
	}

	// Whole-tree change: clearing a root scalar makes the root unrepresentable
	// by a set, so the diff is a root replace whose box is the whole tree —
	// larger than the full VIEW, and the guard picks the full snapshot.
	huge := bigTree(1000)
	huge.Style = "whole-tree"
	if err := client.Commit(huge, Keys{}); err != nil {
		t.Fatalf("commit huge: %v", err)
	}
	decodeNext(t, out)
	hugeNext := gproto.Clone(huge).(*pb.Box)
	hugeNext.Style = ""
	sent, err = client.CommitDelta(huge, hugeNext, Keys{})
	if err != nil {
		t.Fatalf("commit delta huge: %v", err)
	}
	if sent {
		t.Fatal("whole-tree change stayed a delta, want full VIEW fallback")
	}
	typ, _ = decodeNext(t, out)
	if typ != wire.TypeView {
		t.Fatalf("frame = %v, want VIEW fallback", typ)
	}
	if client.Rev() != 4 {
		t.Fatalf("rev = %d, want 4 after commit+delta+commit+fallback", client.Rev())
	}
}

func TestCommitDeltaWithoutFeatureIsAlwaysFull(t *testing.T) {
	client, out := helloWith(t, false)
	base := bigTree(50)
	if err := client.Commit(base, Keys{}); err != nil {
		t.Fatalf("commit: %v", err)
	}
	decodeNext(t, out)
	next := gproto.Clone(base).(*pb.Box)
	next.GetChildren()[0].Style = "x"
	sent, err := client.CommitDelta(base, next, Keys{})
	if err != nil {
		t.Fatalf("commit delta: %v", err)
	}
	if sent {
		t.Fatal("delta sent without the view_delta feature")
	}
	if typ, _ := decodeNext(t, out); typ != wire.TypeView {
		t.Fatalf("frame = %v, want VIEW", typ)
	}
}

func TestCommitDeltaNoBaseIsFull(t *testing.T) {
	client, out := helloWith(t, true)
	base := bigTree(20)
	sent, err := client.CommitDelta(nil, base, Keys{})
	if err != nil {
		t.Fatalf("commit delta: %v", err)
	}
	if sent {
		t.Fatal("delta sent with no base")
	}
	if typ, _ := decodeNext(t, out); typ != wire.TypeView {
		t.Fatalf("frame = %v, want VIEW", typ)
	}
}

func TestDropBaseForcesFull(t *testing.T) {
	client, out := helloWith(t, true)
	base := bigTree(30)
	if err := client.Commit(base, Keys{}); err != nil {
		t.Fatalf("commit: %v", err)
	}
	decodeNext(t, out)
	client.DropBase()
	if client.HasBase() {
		t.Fatal("HasBase = true after DropBase")
	}
	next := gproto.Clone(base).(*pb.Box)
	next.GetChildren()[0].Style = "x"
	sent, err := client.CommitDelta(base, next, Keys{})
	if err != nil {
		t.Fatalf("commit delta: %v", err)
	}
	if sent {
		t.Fatal("delta sent after DropBase")
	}
}

// TestFullViewFrameLenMatchesMarshal pins the cost guard's size computation to
// the exact frame Commit writes, including shared subtrees whose protobuf size
// caches are populated by earlier calls (the memoized-subtree fast path) and
// zero/empty fields that proto3 omits.
func TestFullViewFrameLenMatchesMarshal(t *testing.T) {
	client, _ := helloWith(t, true)
	keyCases := []Keys{{}, {All: true}, {Claim: []string{"ctrl-p", "ctrl-q"}}}
	large := bigTree(1000)
	fat := &pb.Box{
		Id: "fat", Style: "bold", Focused: true, Input: []string{"key"},
		Size:    &pb.Size{Width: 3, Height: 4, Flex: 1},
		Pos:     &pb.Pos{X: 1, Y: 2},
		Cursor:  &pb.Cursor{Row: 1, Col: 2, Shape: "bar"},
		Content: &pb.Content{Text: "x", Self: "s", Lines: []string{"a", "b"}, Props: map[string]string{"k": "v"}},
	}
	trees := []*pb.Box{
		large,
		&pb.Box{},
		box("root", &pb.Box{}),
		fat,
	}
	// A tree whose children all point at the same subtree: a size cache entry
	// for one child is valid for all three.
	trees = append(trees, &pb.Box{Id: "shared-root", Children: []*pb.Box{large, large, large}})
	// The next generation shares every child but one, like a memoized frame.
	fresh := &pb.Box{Id: large.Id, Children: append([]*pb.Box(nil), large.GetChildren()...)}
	fresh.Children[0] = gproto.Clone(fresh.Children[0]).(*pb.Box)
	fresh.Children[0].Style = "changed"
	trees = append(trees, fresh)
	// Unknown fields survive a decode and are part of proto.Size.
	trees = append(trees, withUnknownField(t, box("unknown", text("a", "b"))))

	for i, tree := range trees {
		for _, keys := range keyCases {
			for _, rev := range []uint64{0, 1, 42} {
				epoch := rev + 7
				got := client.fullViewFrameLen(tree, keys, epoch, rev)
				frame, err := wire.Marshal(wire.TypeView, &pb.View{
					Epoch: epoch, Rev: rev,
					Keys: &pb.Keys{Claim: keys.Claim, All: keys.All}, Root: tree,
				}, 0)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				if got != len(frame) {
					t.Fatalf("tree %d keys %+v rev %d: fullViewFrameLen = %d, marshalled = %d", i, keys, rev, got, len(frame))
				}
				// The second call takes the size-cache path; it must return the
				// same exact length.
				if again := client.fullViewFrameLen(tree, keys, epoch, rev); again != got {
					t.Fatalf("tree %d keys %+v rev %d: second call = %d, want %d", i, keys, rev, again, got)
				}
			}
		}
	}
}

// withUnknownField returns a copy of box carrying one unknown (field 100)
// varint, the state a decoded message can hold and proto.Size must count.
func withUnknownField(t *testing.T, box *pb.Box) *pb.Box {
	t.Helper()
	data, err := gproto.Marshal(box)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// field 100, wire type 0: tag varint 0xA0 0x06, value 1.
	data = append(data, 0xA0, 0x06, 0x01)
	out := &pb.Box{}
	if err := gproto.Unmarshal(data, out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.ProtoReflect().GetUnknown()) == 0 {
		t.Fatal("test setup: box has no unknown fields")
	}
	return out
}

func TestViewRejectedDropsBase(t *testing.T) {
	client, out := helloWith(t, true)
	base := bigTree(30)
	if err := client.Commit(base, Keys{}); err != nil {
		t.Fatalf("commit: %v", err)
	}
	decodeNext(t, out)
	client.dispatchEvent(&pb.Event{Event: &pb.Event_ViewRejected{
		ViewRejected: &pb.ViewRejectedEvent{Epoch: 1, Rev: 1, Reason: "base_mismatch"},
	}})
	if client.HasBase() {
		t.Fatal("view_rejected did not drop the base")
	}
}

func TestNewHelloDropsBase(t *testing.T) {
	client, _ := helloWith(t, true)
	if err := client.Commit(bigTree(10), Keys{}); err != nil {
		t.Fatalf("commit: %v", err)
	}
	client.dispatchHello(&pb.Hello{Schema: 1, Epoch: 2, Features: map[string]bool{ViewDeltaFeature: true}})
	if client.HasBase() {
		t.Fatal("a new HELLO did not drop the base")
	}
}

func TestCommitDeltaNilNext(t *testing.T) {
	client, _ := helloWith(t, true)
	if _, err := client.CommitDelta(nil, nil, Keys{}); err != ErrNilRoot {
		t.Fatalf("err = %v, want ErrNilRoot", err)
	}
}

func decodeDelta(t *testing.T, payload []byte) *pb.ViewDelta {
	t.Helper()
	m, err := wire.UnmarshalPayload(wire.TypeViewDelta, payload)
	if err != nil {
		t.Fatalf("decode delta: %v", err)
	}
	return m.(*pb.ViewDelta)
}

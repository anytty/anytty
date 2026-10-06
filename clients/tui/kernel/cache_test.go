package kernel

import (
	"reflect"
	"testing"
)

// cloneTree deep-copies a Node tree without its solved cache, so a test can
// solve a structurally equal but pointer-distinct tree from scratch.
func cloneTree(n *Node) *Node {
	if n == nil {
		return nil
	}
	c := *n
	c.solved = nil
	if n.Children != nil {
		c.Children = make([]Node, len(n.Children))
		for i := range n.Children {
			c.Children[i] = *cloneTree(&n.Children[i])
		}
	}
	return &c
}

func TestLayoutCachedMatchesLayout(t *testing.T) {
	cases := []struct {
		name          string
		root          *Node
		width, height int
	}{
		{
			name: "nested flow with overlays and cursor",
			root: &Node{ID: "root", Flow: FlowCol, Children: []Node{
				{ID: "a", Content: &Content{Lines: []string{"hello"}}, Cursor: &Cursor{Row: 0, Col: 2, Shape: "bar"}},
				{ID: "wrap", Flow: FlowRow, Children: []Node{
					{ID: "b", Size: Size{Width: 4}},
					{ID: "c", Size: Size{Flex: 1}},
				}},
				{ID: "over", Pos: &Pos{X: 2, Y: 1}, Size: Size{Width: 5, Height: 2}, Content: &Content{Text: "popup"}},
				{ID: "invis", Visible: vp(false), Content: &Content{Text: "nope"}},
			}},
			width: 20, height: 8,
		},
		{
			name: "stack and intrinsic",
			root: &Node{ID: "root", Flow: FlowStack, Children: []Node{
				{ID: "a", Content: &Content{Lines: []string{"ab", "cd"}}},
				{ID: "b", Size: Size{Width: 3, Height: 2}, Content: &Content{Text: "x"}},
			}},
			width: 10, height: 5,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := Layout(cloneTree(tc.root), tc.width, tc.height)

			first := LayoutCached(tc.root, tc.width, tc.height)
			again := LayoutCached(tc.root, tc.width, tc.height)
			assertFrameEqual(t, first, want)
			assertFrameEqual(t, again, want)
		})
	}
}

// TestLayoutCachedRectGate proves the hard gate: a cached frame is only reused
// at the identical rect. A different viewport re-solves and yields exactly the
// frame Layout would.
func TestLayoutCachedRectGate(t *testing.T) {
	root := &Node{ID: "root", Children: []Node{
		{ID: "a", Content: &Content{Text: "hello"}},
		{ID: "b"},
	}}
	LayoutCached(root, 10, 4)
	if root.Cached() == nil {
		t.Fatal("root not cached")
	}
	if r := root.Cached().Rect; r != (Rect{0, 0, 10, 4}) {
		t.Fatalf("cached rect = %v", r)
	}

	got := LayoutCached(root, 30, 9)
	want := Layout(cloneTree(root), 30, 9)
	assertFrameEqual(t, got, want)
	if r := root.Cached().Rect; r != (Rect{0, 0, 30, 9}) {
		t.Fatalf("re-solved cached rect = %v, want 30x9", r)
	}
}

// TestLayoutCachedSubtreeRectGate moves a cached child to a new rect and checks
// it is re-solved rather than spliced at its old absolute coordinates.
func TestLayoutCachedSubtreeRectGate(t *testing.T) {
	root := &Node{ID: "root", Flow: FlowCol, Children: []Node{
		{ID: "head", Size: Size{Height: 1}},
		{ID: "body", Size: Size{Height: 1}, Content: &Content{Text: "body"}},
	}}
	LayoutCached(root, 10, 4)
	stale := root.Children[1].Cached()
	if stale == nil {
		t.Fatal("body not cached")
	}
	if stale.Rect.Y != 1 {
		t.Fatalf("stale body rect = %v, want y=1", stale.Rect)
	}

	// Reorder: body now lands at y=0. Its stale frame is at y=1 and must be
	// rejected (rect gate), then re-solved.
	reordered := &Node{ID: "root", Flow: FlowCol, Children: []Node{
		{ID: "body", Size: Size{Height: 1}, Content: &Content{Text: "body"}},
		{ID: "head", Size: Size{Height: 1}},
	}}
	reordered.Children[0].SetCached(stale)
	got := LayoutCached(reordered, 10, 4)
	want := Layout(cloneTree(reordered), 10, 4)
	assertFrameEqual(t, got, want)
	if r, _ := got.Rect("body"); r != (Rect{0, 0, 4, 1}) {
		t.Fatalf("body rect = %v, want {0 0 4 1} (re-solved)", r)
	}
}

// TestLayoutCachedAliasing checks that splicing links a child's rect index by
// pointer: the parent frame and the cached child frame stay independent frame
// values (the parent's index is its own object), the cached child keeps its
// rect, and a later solve of a structurally identical tree is unaffected.
func TestLayoutCachedAliasing(t *testing.T) {
	shared := &Node{ID: "shared", Content: &Content{Text: "abc"}}
	root1 := &Node{ID: "root", Flow: FlowCol, Children: []Node{
		{ID: "top", Size: Size{Height: 1}},
		*shared,
	}}
	LayoutCached(root1, 12, 4)
	if root1.Children[1].Cached() == nil {
		t.Fatal("shared child not cached")
	}

	// A new tree copies the solved child in; the splice path runs.
	root2 := &Node{ID: "root", Flow: FlowCol, Children: []Node{
		{ID: "top", Size: Size{Height: 1}, Content: &Content{Text: "TOP"}},
		*shared,
	}}
	f := LayoutCached(root2, 12, 4)
	if _, ok := f.Rect("shared"); !ok {
		t.Fatal("shared rect missing")
	}

	// The parent frame links the cached child's index (splice by reference),
	// while the cached child frame remains its own value with the same rect.
	childFrame := root2.Children[1].Cached().Frame
	if f.index == childFrame.index {
		t.Fatal("parent frame reused the cached child index instead of linking it")
	}
	linked := false
	for i := range f.index.items {
		if f.index.items[i].ref == childFrame.index {
			linked = true
		}
	}
	if !linked {
		t.Fatal("parent frame did not link the cached child index")
	}
	if got, ok := childFrame.Rect("shared"); !ok || got != (Rect{0, 1, 3, 1}) {
		t.Fatalf("cached child rect = %v/%v, want {0 1 3 1}", ok, got)
	}

	// A later solve of a structurally identical tree is unaffected.
	fresh := LayoutCached(&Node{ID: "root", Flow: FlowCol, Children: []Node{
		{ID: "top", Size: Size{Height: 1}, Content: &Content{Text: "TOP"}},
		*shared,
	}}, 12, 4)
	want := Layout(&Node{ID: "root", Flow: FlowCol, Children: []Node{
		{ID: "top", Size: Size{Height: 1}, Content: &Content{Text: "TOP"}},
		{ID: "shared", Content: &Content{Text: "abc"}},
	}}, 12, 4)
	assertFrameEqual(t, fresh, want)
}

// TestLayoutCachedNestedSubtreeReuse checks that a shared child subtree keeps
// its exact solved frame across an unrelated sibling change, and that the
// combined frame equals a from-scratch solve.
func TestLayoutCachedNestedSubtreeReuse(t *testing.T) {
	shared := &Node{ID: "shared", Flow: FlowRow, Children: []Node{
		{ID: "s1", Content: &Content{Text: "one"}},
		{ID: "s2", Size: Size{Width: 3}, Content: &Content{Text: "two"}},
	}}
	root1 := &Node{ID: "root", Flow: FlowCol, Children: []Node{
		{ID: "top", Size: Size{Height: 1}, Content: &Content{Text: "top"}},
		*shared,
	}}
	LayoutCached(root1, 20, 6)
	leftFrame := root1.Children[1].Cached()
	if leftFrame == nil {
		t.Fatal("shared subtree not cached")
	}

	// Rebuild the tree with a changed sibling but the same shared child carried
	// over as a shallow value copy (exactly what the host does when it reuses a
	// kernel node): the copy keeps the solved pointer, so the kernel splices it.
	root2 := &Node{ID: "root", Flow: FlowCol, Children: []Node{
		{ID: "top", Size: Size{Height: 1}, Content: &Content{Text: "TOP"}},
		root1.Children[1],
	}}
	got := LayoutCached(root2, 20, 6)
	want := Layout(cloneTree(root2), 20, 6)
	assertFrameEqual(t, got, want)

	// The carried-over subtree must keep its Solved by value (spliced, not
	// re-solved). A shallow Node copy shares the solved pointer.
	if r2 := root2.Children[1].Cached(); r2 != leftFrame {
		t.Fatal("shared subtree was re-solved instead of spliced")
	}
}

// TestLayoutCachedIndexSharedOnReuse proves the tree-shaped index is spliced
// by pointer, not re-materialized: after a one-leaf delta on a large tree the
// untouched row's exact rect index is linked into the new root index, and the
// root only materializes its own O(depth) changed nodes. The total RectCount is
// unchanged and the frame still equals a from-scratch solve.
func TestLayoutCachedIndexSharedOnReuse(t *testing.T) {
	const rows = 2000
	root, rowsNodes := abKernelTree(rows)
	first := LayoutCached(root, 120, 40)
	if got, want := first.RectCount(), rows+1; got != want {
		t.Fatalf("RectCount = %d, want %d", got, want)
	}

	// A far-away untouched row keeps its cached index pointer.
	untouched := rowsNodes[0]
	untouchedIdx := untouched.Cached().Frame.index
	if untouchedIdx == nil {
		t.Fatal("untouched row has no cached index")
	}

	// Change one leaf and invalidate exactly its path (root -> row -> leaf),
	// mirroring the host's copy-on-write rebuild.
	row := rowsNodes[rows/2]
	target := &row.Children[0]
	target.Content = &Content{Text: "changed"}
	root.SetCached(nil)
	row.SetCached(nil)
	target.SetCached(nil)

	second := LayoutCached(root, 120, 40)

	linked := false
	for i := range second.index.items {
		if second.index.items[i].ref == untouchedIdx {
			linked = true
			break
		}
	}
	if !linked {
		t.Fatal("untouched subtree index was not pointer-shared into the new frame")
	}
	if untouched.Cached().Frame.index != untouchedIdx {
		t.Fatal("untouched subtree index was rebuilt")
	}
	// The root re-solved only itself plus the changed path (root, row, leaf);
	// every untouched row is a linked ref, not an own entry.
	own := 0
	for i := range second.index.items {
		if second.index.items[i].ref == nil {
			own++
		}
	}
	if own > 8 {
		t.Fatalf("root materialized %d own entries for a one-leaf delta, want O(depth)", own)
	}
	if second.RectCount() != first.RectCount() {
		t.Fatalf("RectCount = %d, want %d (unchanged)", second.RectCount(), first.RectCount())
	}

	want := Layout(cloneTree(root), 120, 40)
	assertFrameEqual(t, second, want)
}

func cloneFrameForTest(f Frame) Frame {
	return flattenFrame(f)
}

func assertFrameEqual(t *testing.T, got, want Frame) {
	t.Helper()
	// Compare flattened copies so index sharing (refs vs own entries) does not
	// change equality: the same layout always flattens to the same entry order.
	got, want = flattenFrame(got), flattenFrame(want)
	if !reflect.DeepEqual(got.index.items, want.index.items) {
		t.Fatalf("Rects =\n%v\nwant\n%v", got.rectsForTest(), want.rectsForTest())
	}
	if !reflect.DeepEqual(got.Lines, want.Lines) {
		t.Fatalf("Lines =\n%v\nwant\n%v", got.Lines, want.Lines)
	}
	if got.HasCursor != want.HasCursor || got.CursorRect != want.CursorRect || got.CursorShape != want.CursorShape {
		t.Fatalf("cursor = %v/%v/%q, want %v/%v/%q",
			got.HasCursor, got.CursorRect, got.CursorShape,
			want.HasCursor, want.CursorRect, want.CursorShape)
	}
	if len(got.OverlayFrames) != len(want.OverlayFrames) {
		t.Fatalf("OverlayFrames = %d, want %d", len(got.OverlayFrames), len(want.OverlayFrames))
	}
	for i := range got.OverlayFrames {
		assertFrameEqual(t, got.OverlayFrames[i], want.OverlayFrames[i])
	}
	if len(got.hits) != len(want.hits) {
		t.Fatalf("hits = %d, want %d", len(got.hits), len(want.hits))
	}
	for i := range got.hits {
		if got.hits[i] != want.hits[i] {
			t.Fatalf("hits[%d] = %+v, want %+v", i, got.hits[i], want.hits[i])
		}
	}
	for y := -1; y <= 6; y++ {
		for x := -1; x <= 12; x++ {
			if g, w := got.Hit(x, y), want.Hit(x, y); g != w {
				t.Fatalf("Hit(%d,%d) = %q, want %q", x, y, g, w)
			}
		}
	}
}

// TestLayoutCachedDuplicateIDOrder matches plain Layout when ids repeat: both
// must resolve an id to the same (last-in-solve-order) rect, so LayoutCached
// stays an exact drop-in for Layout.
func TestLayoutCachedDuplicateIDOrder(t *testing.T) {
	// A regular child with id X and a Pos overlay also with id X: solve order
	// is the regular child then the Pos overlay, so the overlay wins.
	root := &Node{ID: "root", Children: []Node{
		{ID: "X", Size: Size{Width: 1, Height: 1}},
		{ID: "ov", Pos: &Pos{X: 2, Y: 1}, Size: Size{Width: 2, Height: 1},
			Children: []Node{{ID: "X", Size: Size{Width: 2, Height: 1}}}},
	}}
	plain := Layout(root, 10, 4)
	cached := LayoutCached(root, 10, 4)
	p, okP := plain.Rect("X")
	c, okC := cached.Rect("X")
	if !okP || !okC || p != c {
		t.Fatalf("duplicate id X: Layout=%v/%v LayoutCached=%v/%v", p, okP, c, okC)
	}
}

// TestLayoutCachedSharedRefOrder keeps the rect lookup identical to a flat map
// when a reused subtree is linked by reference next to own entries.
func TestLayoutCachedSharedRefOrder(t *testing.T) {
	root := &Node{ID: "root", Children: []Node{
		{ID: "a", Size: Size{Height: 1}},
		{ID: "b", Size: Size{Height: 1}},
	}}
	first := Layout(root, 10, 4)
	if _, ok := first.Rect("a"); !ok {
		t.Fatal("missing a")
	}
	second := LayoutCached(root, 10, 4)
	if got, want := second.RectCount(), first.RectCount(); got != want {
		t.Fatalf("RectCount = %d, want %d", got, want)
	}
	for _, id := range []string{"root", "a", "b"} {
		if r, ok := second.Rect(id); !ok {
			t.Fatalf("missing %s after cached solve", id)
		} else if r2, _ := first.Rect(id); r != r2 {
			t.Fatalf("%s = %v, want %v", id, r, r2)
		}
	}
}

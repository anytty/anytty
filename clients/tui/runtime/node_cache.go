package runtime

import (
	pb "github.com/anytty/anytty/proto/ui/protobuf"

	"github.com/anytty/anytty/clients/tui/kernel"
)

// Node reuse across commits.
//
// A VIEW_DELTA is applied with copy-on-write path copying (view_delta.go), so
// every subtree that is not on a patched path is the exact same *pb.Box
// pointer it was in the previous revision. That pointer identity is the cache
// key: the session keeps a map from *pb.Box to the *kernel.Node built from it,
// and a hit means the whole kernel subtree (and every `kernel.Solved` frame
// cached inside it) can be carried over instead of rebuilt.
//
// The kernel layout is then re-solved with kernel.LayoutCached, which splices
// a reused node's frame when the node is placed at the identical absolute
// rect, and solves only the nodes on the changed path.

// nodeMapCapFactor bounds the pointer cache: when an incremental commit would
// leave more than this multiple of the current box count in the map, the map
// is rebuilt from the live tree with indexNodes. Under correct incremental
// pruning the map holds exactly one entry per live box, so the fallback only
// fires if a future patch op forgets to report a detached pointer.
const nodeMapCapFactor = 4

// buildNodeTree converts a program box tree to the kernel tree used for
// layout, reusing kernel nodes for subtrees whose *pb.Box pointer is unchanged
// since the previous commit. It returns the new root and whether any subtree
// was reused: a commit with no reuse has no cached frames to splice, so the
// caller lays it out with the cheaper full solver.
//
// A full VIEW shares no pointers with the previous tree, so it is built
// without consulting or populating the cache (incremental=false); the cache is
// then lazily indexed on the next delta commit (see indexNodes). A delta tree
// reuses shared subtrees in place and inserts only the new boxes it creates
// into the persistent map, so both building and bookkeeping are O(changed)
// rather than O(tree). The map is pruned of the pointers a patch detached
// (stale) and bounded by nodeMapCapFactor times the live box count.
func (s *Session) buildNodeTree(root *pb.Box, incremental bool, stale staleSet) (node *kernel.Node, reused bool, inexact bool) {
	if root == nil {
		s.nodes = nil
		return nil, false, false
	}
	if !incremental {
		// A full VIEW has no shared pointer with the old tree. Drop the stale
		// map so the next delta rebuilds it from this tree.
		s.nodes = nil
		return s.nodeFull(nil, root), false, false
	}
	if s.nodes == nil {
		s.indexNodes()
	}
	node, reused = s.nodeFor(nil, root, false)
	// The map could not be pruned exactly (see pruneStale): the caller must
	// rebuild it from the COMMITTED tree, so the reindex happens after the new
	// view is installed. This is the rare multi-patch case, not the steady
	// state.
	inexact = !s.pruneStale(stale)
	return node, reused, inexact
}

// enforceNodeMapCap is the O(1) safety fallback for the pointer cache: when
// the map holds more than nodeMapCapFactor times the (incrementally known) live
// box count it is rebuilt from the committed tree. It runs after the commit so
// the maintained boxCount is already up to date, and is a no-op while the
// count is unknown (the map is then rebuilt lazily anyway).
func (s *Session) enforceNodeMapCap() {
	if s.nodes == nil || s.boxCount < 0 {
		return
	}
	if len(s.nodes) > nodeMapCapFactor*s.boxCount {
		s.indexNodes()
	}
}

// staleSet carries the old pointers a patch burst detached: spine holds nodes
// replaced by copy-on-write spine clones (only their own entry is stale; their
// shared children remain reachable), and subtrees holds whole detached
// subtrees whose entries can be dropped recursively.
type staleSet struct {
	spine    []*pb.Box
	subtrees []*pb.Box
}

// pruneStale drops the detached pointers from the persistent map. A spine node
// is dropped only for itself so untouched shared descendants keep their
// entries; a detached subtree is dropped recursively. It reports whether the
// map could be pruned exactly: a detached subtree whose root has no entry can
// still hide cached shared descendants (the root may be a clone created by an
// earlier patch in the same burst, so the walk cannot tell whether deeper
// pointers are live), and the caller then rebuilds the map from the committed
// tree.
func (s *Session) pruneStale(stale staleSet) (exact bool) {
	if s.nodes == nil {
		return true
	}
	exact = true
	for _, b := range stale.spine {
		delete(s.nodes, b)
	}
	for _, b := range stale.subtrees {
		if !dropSubtrees(b, s.nodes) {
			exact = false
		}
	}
	return exact
}

// dropSubtrees deletes every box of a detached subtree from the map. It stops
// descending at a box with no entry; that is exact only when the map mirrors
// the subtree, which is not guaranteed when the detached root was itself
// created earlier in the same patch burst (its descendants may still be
// cached from the base tree). It returns false in that case so the caller can
// rebuild the map instead of leaking stale pointers.
func dropSubtrees(b *pb.Box, nodes map[*pb.Box]*kernel.Node) bool {
	if b == nil {
		return true
	}
	if _, ok := nodes[b]; !ok {
		return false
	}
	delete(nodes, b)
	exact := true
	for _, c := range b.Children {
		if !dropSubtrees(c, nodes) {
			exact = false
		}
	}
	return exact
}

// indexNodes (re)builds the box -> kernel node map for the current tree by
// pairing the cached pb tree with its kernel tree. It is called once after a
// full VIEW (which intentionally does not build the map), after a Reset, and
// as the bounded-map safety fallback; it is O(tree), so it is paid only when
// the map has to be re-established.
func (s *Session) indexNodes() {
	if s.view == nil || s.root == nil {
		return
	}
	s.nodes = make(map[*pb.Box]*kernel.Node, countBoxes(s.view.Root))
	indexBoxes(s.view.Root, s.root, s.nodes)
}

// boxLive reports whether a box is part of the committed tree, i.e. present in
// the pointer cache. It is used to detect aliased patch payloads.
func (s *Session) boxLive(b *pb.Box) bool {
	if b == nil || s.nodes == nil {
		return false
	}
	_, ok := s.nodes[b]
	return ok
}

// indexBoxes pairs a pb tree with the mirroring kernel tree. The two trees are
// built 1:1, so the walk touches no node fields.
func indexBoxes(b *pb.Box, n *kernel.Node, out map[*pb.Box]*kernel.Node) {
	if b == nil || n == nil {
		return
	}
	out[b] = n
	for i, child := range b.Children {
		if child == nil || i >= len(n.Children) {
			continue
		}
		indexBoxes(child, &n.Children[i], out)
	}
}

// nodeFor builds the kernel node for b and returns it. dst is the slot the
// caller wants the node in (nil means "allocate"); the slot is always filled
// in place, so a fresh tree allocates one Node per box and never a temporary.
// reused reports whether this subtree (or any descendant) was served from the
// previous commit's cache.
//
// A cache hit reuses the previously built kernel node. For the root that is
// the very same *kernel.Node pointer; for a child it is a shallow value copy
// into the caller's child slot (kernel.Node.Children is []Node, so a reused
// child must live inside its parent's new slice). The shallow copy carries the
// node's cached Solved by value and shares its read-only children backing
// array, so no part of the subtree is rebuilt or re-solved. On a hit the walk
// stops: every descendant is already present in the persistent map under its
// original pointer, so there is nothing to re-register. Kernel nodes are
// read-only after they are solved, so sharing them is safe.
func (s *Session) nodeFor(dst *kernel.Node, b *pb.Box, reused bool) (*kernel.Node, bool) {
	if b == nil {
		return nil, reused
	}
	if cached, ok := s.nodes[b]; ok {
		if dst == nil {
			// Root reuse: keep the exact pointer and its cached frame.
			return cached, true
		}
		if dst != cached {
			*dst = *cached
		}
		// Descendants live in the shared children backing array, so their
		// addresses are unchanged; only this node's slot moved.
		s.nodes[b] = dst
		return dst, true
	}
	if dst == nil {
		dst = new(kernel.Node)
	}
	s.fillNode(dst, b)
	if len(b.Children) > 0 {
		dst.Children = make([]kernel.Node, len(b.Children))
		for i, child := range b.Children {
			if child != nil {
				_, r := s.nodeFor(&dst.Children[i], child, reused)
				reused = reused || r
			}
		}
	}
	s.nodes[b] = dst
	return dst, reused
}

// nodeFull builds the kernel tree for b with no cache. dst is reused when the
// caller already has a slot (never for the initial full build). It is the
// non-incremental path, so it never reads s.nodes and never allocates the map.
func (s *Session) nodeFull(dst *kernel.Node, b *pb.Box) *kernel.Node {
	if b == nil {
		return nil
	}
	if dst == nil {
		dst = new(kernel.Node)
	}
	s.fillNode(dst, b)
	if len(b.Children) > 0 {
		dst.Children = make([]kernel.Node, len(b.Children))
		for i, child := range b.Children {
			if child != nil {
				s.nodeFull(&dst.Children[i], child)
			}
		}
	}
	return dst
}

// fillNode copies the scalar and submessages of one box into a kernel node.
func (s *Session) fillNode(dst *kernel.Node, b *pb.Box) {
	dst.ID = b.GetId()
	dst.Flow = kernel.Flow(b.GetFlow())
	dst.Style = b.GetStyle()
	dst.Focused = b.GetFocused()
	if len(b.Input) > 0 {
		dst.Input = append([]string(nil), b.Input...)
	}
	if b.Size != nil {
		dst.Size = kernel.Size{
			Width:  int(b.Size.GetWidth()),
			Height: int(b.Size.GetHeight()),
			Flex:   int(b.Size.GetFlex()),
		}
	}
	if b.Pos != nil {
		dst.Pos = &kernel.Pos{X: int(b.Pos.GetX()), Y: int(b.Pos.GetY())}
	}
	if b.Visible != nil {
		v := b.GetVisible()
		dst.Visible = &v
	}
	if b.Content != nil {
		c := &kernel.Content{
			Text: b.Content.GetText(),
			Self: b.Content.GetSelf(),
		}
		if len(b.Content.Lines) > 0 {
			c.Lines = append([]string(nil), b.Content.Lines...)
		}
		c.Props = cloneProps(b.Content.GetProps())
		dst.Content = c
	}
	if b.Cursor != nil {
		c := &kernel.Cursor{
			Row:   int(b.Cursor.GetRow()),
			Col:   int(b.Cursor.GetCol()),
			Shape: b.Cursor.GetShape(),
		}
		if b.Cursor.Visible != nil {
			v := b.Cursor.GetVisible()
			c.Visible = &v
		}
		dst.Cursor = c
	}
}

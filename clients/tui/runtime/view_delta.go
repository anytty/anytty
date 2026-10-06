package runtime

import (
	"errors"

	gproto "google.golang.org/protobuf/proto"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// Patch rejection / view_rejected reason strings (PROTOCOL §2.1). The set of
// values is fixed by the wire spec: base_mismatch, path_invalid, max_nodes,
// oversize (oversize is produced by the frame decoder, not here).
const (
	reasonBaseMismatch = "base_mismatch"
	reasonPathInvalid  = "path_invalid"
	reasonMaxNodes     = "max_nodes"
)

// HandleViewDelta applies one VIEW_DELTA frame (PROTOCOL §2.1). It follows
// HandleView's locking/stale rules exactly: a nil delta is a programming
// error, a foreign epoch or non-monotonic rev is dropped silently, and any
// malformed patch answers view_rejected once per (epoch, rev). A delta may
// never be the first frame of an epoch: the host clears the cache on HELLO,
// so without a cached base the only legal way to start is a full VIEW.
//
// Patches are applied with copy-on-write path copying: only the nodes on the
// patched paths are rebuilt, untouched subtrees stay shared with the cached
// tree, and the cached tree itself is never mutated. That keeps a small edit
// on a large tree O(depth + patch size) instead of O(tree).
func (s *Session) HandleViewDelta(d *pb.ViewDelta) error {
	if d == nil {
		return errors.New("runtime: nil view delta")
	}
	s.mu.Lock()
	if d.Epoch != s.epoch {
		s.mu.Unlock()
		return nil
	}
	if d.Rev <= s.rev {
		s.mu.Unlock()
		return nil
	}
	if s.view == nil {
		err := s.rejectViewLocked(d.Rev, reasonBaseMismatch)
		s.mu.Unlock()
		return err
	}
	if d.RevBase != s.rev {
		err := s.rejectViewLocked(d.Rev, reasonBaseMismatch)
		s.mu.Unlock()
		return err
	}
	// Index the committed tree now when the last commit was a full VIEW, so the
	// payload liveness check below can see every live box.
	if s.nodes == nil {
		s.indexNodes()
	}
	root, effect, reason := applyPatchesEffect(s.view.Root, d.Patches, s.boxLive)
	if reason != "" {
		err := s.rejectViewLocked(d.Rev, reason)
		s.mu.Unlock()
		return err
	}
	// The new box count is maintained incrementally (O(1)) once known; only
	// the first growth delta after a full VIEW pays the O(tree) walk to seed
	// it. Validation order/reasons are unchanged: the count is only consulted
	// when a patch could grow the tree and max_nodes is set.
	max := s.limits.MaxNodes
	grow := patchesCanGrow(d.Patches)
	boxes := -1
	if s.boxCount >= 0 {
		boxes = s.boxCount + effect.boxDelta
	} else if max > 0 && grow {
		boxes = countBoxes(root)
	}
	if max > 0 && grow && boxes >= 0 && uint32(boxes) > max {
		err := s.rejectViewLocked(d.Rev, reasonMaxNodes)
		s.mu.Unlock()
		return err
	}
	view := &pb.View{Epoch: d.Epoch, Rev: d.Rev, Keys: d.Keys, Root: root}
	s.commitViewLocked(d.Rev, view, root, d.Keys, d.Keys != nil, true,
		staleSet{spine: effect.spineStale, subtrees: effect.subtreeStale},
		effect.focusDirty, boxes)
	onView := s.onView
	s.mu.Unlock()
	if onView != nil {
		onView()
	}
	return nil
}

// patchesCanGrow reports whether any op may add nodes. set/move keep the node
// count and remove only decreases it, so the O(tree) max_nodes walk can be
// skipped for those.
func patchesCanGrow(patches []*pb.Patch) bool {
	for _, p := range patches {
		switch p.GetOp() {
		case "insert", "replace":
			return true
		}
	}
	return false
}

// patchEffect accumulates the commit bookkeeping a patch burst implies, so the
// caller can keep the node cache and the derived session state O(changed)
// instead of re-walking the tree: the old pointers a patch detaches (for cache
// pruning), whether any patch could have changed focus (so focusLocked can be
// skipped), and the net box-count delta (so max_nodes needs no countBoxes
// walk). It is collected while the tree is already being rebuilt, so it adds no
// extra traversal.
type patchEffect struct {
	// spineStale are old nodes replaced by cloneSpine/cloneForEdit. Their own
	// map entry is stale, but their untouched children are still shared into
	// the new tree, so only the node itself may be dropped.
	spineStale []*pb.Box
	// subtreeStale are old subtrees a replace/remove detached whole. No node of
	// such a subtree is reachable from the new tree, so their entries can be
	// dropped recursively.
	subtreeStale []*pb.Box
	// focusDirty is set when any patch could have changed which node
	// focusLocked would pick, or its Input. Conservative: structural ops are
	// always dirty, and a set is dirty when its payload carries focused,
	// content, input or visible.
	focusDirty bool
	// boxDelta is the net change in box count; valid for every accepted op
	// because insert/replace/remove count the payload or the removed subtree.
	boxDelta int
}

// payload returns the subtree a replace/insert patch attaches. A payload that
// shares ANY pointer with the committed tree (in-process callers can pass a live
// box or a fresh wrapper holding one; the wire always decodes fresh) is cloned
// so the new tree keeps one unique pointer per box: the node cache is
// pointer-keyed, so aliasing the same pointer into two places would break its
// one-entry-per-box invariant and make pruning ambiguous. The scan is
// O(payload), the same order as the boxDelta count the caller already does.
func (e *patchEffect) payload(b *pb.Box, live func(*pb.Box) bool) *pb.Box {
	if b == nil || live == nil || !hasLiveBox(b, live) {
		return b
	}
	return gproto.Clone(b).(*pb.Box)
}

// hasLiveBox reports whether b or any descendant is part of the committed tree.
func hasLiveBox(b *pb.Box, live func(*pb.Box) bool) bool {
	if b == nil {
		return false
	}
	if live(b) {
		return true
	}
	for _, c := range b.Children {
		if hasLiveBox(c, live) {
			return true
		}
	}
	return false
}

// applyPatches applies every patch in order to root, returning the new tree.
// On failure it returns a view_rejected reason and a nil tree; the cached tree
// is never modified because each op rebuilds only the nodes it touches.
func applyPatches(root *pb.Box, patches []*pb.Patch) (*pb.Box, string) {
	next, _, reason := applyPatchesEffect(root, patches, nil)
	return next, reason
}

// applyPatchesEffect is applyPatches plus the bookkeeping the incremental
// commit path needs (see patchEffect). The effect is only meaningful when the
// returned reason is empty.
func applyPatchesEffect(root *pb.Box, patches []*pb.Patch, live func(*pb.Box) bool) (*pb.Box, *patchEffect, string) {
	effect := &patchEffect{}
	work := root
	for _, p := range patches {
		if p == nil {
			return nil, nil, reasonPathInvalid
		}
		next, reason := applyPatch(work, p, effect, live)
		if reason != "" {
			return nil, nil, reason
		}
		work = next
	}
	return work, effect, ""
}

// focusRelevant reports whether a set payload carries any field focusLocked
// reads (directly or through the node it selects). It is deliberately
// conservative: any content change counts, even a props-only one.
func focusRelevant(b *pb.Box) bool {
	if b == nil {
		return false
	}
	if b.GetFocused() || b.Content != nil || b.Visible != nil {
		return true
	}
	return len(b.GetInput()) > 0
}

func applyPatch(root *pb.Box, p *pb.Patch, e *patchEffect, live func(*pb.Box) bool) (*pb.Box, string) {
	switch p.GetOp() {
	case "set":
		if p.Box == nil {
			return nil, reasonPathInvalid
		}
		if focusRelevant(p.Box) {
			e.focusDirty = true
		}
		return updateAt(root, p.GetPath(), func(target *pb.Box) { mergeBox(target, p.Box) }, e)
	case "replace":
		if p.Box == nil {
			return nil, reasonPathInvalid
		}
		e.focusDirty = true
		box := e.payload(p.Box, live)
		e.boxDelta += countBoxes(box)
		return replaceAt(root, p.GetPath(), box, e)
	case "insert":
		if p.Box == nil {
			return nil, reasonPathInvalid
		}
		e.focusDirty = true
		box := e.payload(p.Box, live)
		e.boxDelta += countBoxes(box)
		return insertAt(root, p.GetPath(), int(p.GetIndex()), box, e)
	case "remove":
		e.focusDirty = true
		return removeAt(root, p.GetPath(), e)
	case "move":
		e.focusDirty = true
		return moveAt(root, p.GetPath(), int(p.GetFrom()), int(p.GetTo()), e)
	default:
		return nil, reasonPathInvalid
	}
}

// updateAt rebuilds the path to the target, then applies fn to a writable
// clone of the target (its children stay shared). Every node it clones is the
// old pointer of a new spine node, so its cache entry is stale (the node's
// children are still shared and stay reachable, so only the node is dropped).
func updateAt(node *pb.Box, path []uint32, fn func(*pb.Box), e *patchEffect) (*pb.Box, string) {
	if node == nil {
		return nil, reasonPathInvalid
	}
	if len(path) == 0 {
		c := cloneForEdit(node)
		fn(c)
		e.spineStale = append(e.spineStale, node)
		return c, ""
	}
	idx := int(path[0])
	if idx < 0 || idx >= len(node.Children) {
		return nil, reasonPathInvalid
	}
	child, reason := updateAt(node.Children[idx], path[1:], fn, e)
	if reason != "" {
		return nil, reason
	}
	c := cloneSpine(node)
	c.Children[idx] = child
	e.spineStale = append(e.spineStale, node)
	return c, ""
}

// replaceAt swaps the subtree at path; the replacement is used as-is (the
// patch payload is freshly decoded and never mutated in place). The replaced
// subtree is detached whole, so its cache entries can be pruned recursively.
func replaceAt(node *pb.Box, path []uint32, box *pb.Box, e *patchEffect) (*pb.Box, string) {
	if node == nil {
		return nil, reasonPathInvalid
	}
	if len(path) == 0 {
		e.subtreeStale = append(e.subtreeStale, node)
		e.boxDelta -= countBoxes(node)
		return box, ""
	}
	idx := int(path[0])
	if idx < 0 || idx >= len(node.Children) {
		return nil, reasonPathInvalid
	}
	child, reason := replaceAt(node.Children[idx], path[1:], box, e)
	if reason != "" {
		return nil, reason
	}
	c := cloneSpine(node)
	c.Children[idx] = child
	e.spineStale = append(e.spineStale, node)
	return c, ""
}

// insertAt inserts box into the children of the container at path; index may
// equal the current child count (append). Insert detaches nothing.
func insertAt(node *pb.Box, path []uint32, idx int, box *pb.Box, e *patchEffect) (*pb.Box, string) {
	if node == nil {
		return nil, reasonPathInvalid
	}
	if len(path) == 0 {
		if idx < 0 || idx > len(node.Children) {
			return nil, reasonPathInvalid
		}
		c := cloneSpine(node)
		c.Children = append(c.Children, nil)
		copy(c.Children[idx+1:], c.Children[idx:])
		c.Children[idx] = box
		e.spineStale = append(e.spineStale, node)
		return c, ""
	}
	seg := int(path[0])
	if seg < 0 || seg >= len(node.Children) {
		return nil, reasonPathInvalid
	}
	child, reason := insertAt(node.Children[seg], path[1:], idx, box, e)
	if reason != "" {
		return nil, reason
	}
	c := cloneSpine(node)
	c.Children[seg] = child
	e.spineStale = append(e.spineStale, node)
	return c, ""
}

// removeAt drops the node at path; the root is not removable. The dropped
// subtree is detached whole, so its cache entries can be pruned recursively.
func removeAt(node *pb.Box, path []uint32, e *patchEffect) (*pb.Box, string) {
	if node == nil || len(path) == 0 {
		return nil, reasonPathInvalid
	}
	idx := int(path[0])
	if idx < 0 || idx >= len(node.Children) {
		return nil, reasonPathInvalid
	}
	if len(path) == 1 {
		dropped := node.Children[idx]
		e.subtreeStale = append(e.subtreeStale, dropped)
		e.boxDelta -= countBoxes(dropped)
		c := cloneSpine(node)
		c.Children = append(c.Children[:idx], c.Children[idx+1:]...)
		e.spineStale = append(e.spineStale, node)
		return c, ""
	}
	child, reason := removeAt(node.Children[idx], path[1:], e)
	if reason != "" {
		return nil, reason
	}
	c := cloneSpine(node)
	c.Children[idx] = child
	e.spineStale = append(e.spineStale, node)
	return c, ""
}

// moveAt reorders children[from] to children[to] within the container at
// path; to is the post-move index. Move detaches no subtree.
func moveAt(node *pb.Box, path []uint32, from, to int, e *patchEffect) (*pb.Box, string) {
	if node == nil {
		return nil, reasonPathInvalid
	}
	if len(path) == 0 {
		n := len(node.Children)
		if from < 0 || from >= n || to < 0 || to >= n {
			return nil, reasonPathInvalid
		}
		if from == to {
			return node, ""
		}
		c := cloneSpine(node)
		child := c.Children[from]
		c.Children = append(c.Children[:from], c.Children[from+1:]...)
		c.Children = append(c.Children, nil)
		copy(c.Children[to+1:], c.Children[to:])
		c.Children[to] = child
		e.spineStale = append(e.spineStale, node)
		return c, ""
	}
	idx := int(path[0])
	if idx < 0 || idx >= len(node.Children) {
		return nil, reasonPathInvalid
	}
	child, reason := moveAt(node.Children[idx], path[1:], from, to, e)
	if reason != "" {
		return nil, reason
	}
	c := cloneSpine(node)
	c.Children[idx] = child
	e.spineStale = append(e.spineStale, node)
	return c, ""
}

// mergeBox applies a set patch to target using protobuf merge semantics:
// fields present in box overwrite target, absent fields keep the old value
// (including an explicitly present visible=false). Children always stay with
// target, because a set must never replace the child list. Repeated input is
// replaced (not appended) when the patch carries it. target is a writable
// clone, so mutating it in place is safe.
func mergeBox(target, box *pb.Box) {
	children := target.Children
	input := target.Input
	target.Children = nil
	target.Input = nil
	gproto.Merge(target, box)
	target.Children = children
	if len(box.GetInput()) > 0 {
		target.Input = append([]string(nil), box.GetInput()...)
	} else {
		target.Input = input
	}
}

// cloneSpine copies one node along a patched path. Non-children fields and
// submessages are shared with the source node, which is safe because path
// copying never mutates them; only the children slice is copied so an edit to
// one child slot cannot touch the source node.
func cloneSpine(b *pb.Box) *pb.Box {
	c := &pb.Box{
		Id:      b.Id,
		Size:    b.Size,
		Pos:     b.Pos,
		Flow:    b.Flow,
		Visible: b.Visible,
		Cursor:  b.Cursor,
		Content: b.Content,
		Input:   b.Input,
		Focused: b.Focused,
		Style:   b.Style,
	}
	if b.Children != nil {
		c.Children = append([]*pb.Box(nil), b.Children...)
	}
	return c
}

// cloneForEdit copies one node that a set patch will mutate. Submessages a
// protobuf merge may write through are deep-cloned so the shared source tree
// stays untouched; children are shared and preserved by mergeBox.
func cloneForEdit(b *pb.Box) *pb.Box {
	c := cloneSpine(b)
	if b.Visible != nil {
		v := *b.Visible
		c.Visible = &v
	}
	if b.Size != nil {
		c.Size = gproto.Clone(b.Size).(*pb.Size)
	}
	if b.Pos != nil {
		c.Pos = gproto.Clone(b.Pos).(*pb.Pos)
	}
	if b.Cursor != nil {
		c.Cursor = gproto.Clone(b.Cursor).(*pb.Cursor)
	}
	if b.Content != nil {
		c.Content = gproto.Clone(b.Content).(*pb.Content)
	}
	return c
}

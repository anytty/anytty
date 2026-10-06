package kernel

import (
	"strings"
)

// Incremental layout: LayoutCached is Layout plus subtree reuse. A host that
// re-solves a tree whose subtrees are structurally shared with the previously
// solved tree (pointer identity at the host's box layer) can offer each
// subtree the frame it was last solved at: when the subtree is placed at the
// exact same absolute rect, LayoutCached splices that frame instead of
// walking the subtree again.
//
// The contract is deliberately narrow and purely geometric:
//
//   - A Solved is only reusable at the identical Rect. Frames hold absolute
//     cell coordinates, so a subtree that moved or resized is re-solved.
//   - A node's Solved frame is immutable. LayoutCached only reads a spliced
//     frame; it never mutates its rect index, Lines, hits or OverlayFrames.
//   - Reuse changes no observable output: for any tree and viewport,
//     LayoutCached returns exactly the frame Layout returns.
//
// LayoutCached records a Solved on every node it solves, so a later
// LayoutCached call can reuse it. A spliced node keeps its existing Solved,
// because a splice happens only when its rect is unchanged. Plain Layout
// ignores any Solved and records none, so it stays the cheap path for a tree
// with no reuse (e.g. a full VIEW at the host).

// LayoutCached is Layout with subtree reuse. It returns exactly the frame
// Layout(root, width, height) would return; the only difference is that a node
// carrying a Solved at the same absolute rect contributes its cached frame
// instead of being solved again. After the call every solved node (including
// the root) carries a Solved for next time.
func LayoutCached(root *Node, width, height int) Frame {
	if root == nil || width <= 0 || height <= 0 || !root.IsVisible() {
		return Frame{}
	}
	rect := Rect{X: 0, Y: 0, Width: width, Height: height}
	if s := root.Cached(); s != nil && s.Rect == rect {
		return s.Frame
	}
	// Seed the allocation sizes for the new frame. A cached child frame's
	// rect/lines lengths are an O(1) exact-enough hint for its subtree, so
	// countHintsPruned stops at any cached node and the walk is O(changed path
	// + fanout), not O(tree). A slice capacity is only a hint, so a slight
	// underestimate (duplicate ids stay in the index; renderOwn clips lines to
	// the rect) just grows as needed.
	h := countHintsPruned(root)
	b := frameBuilder{
		cached: true,
		index:  &rectIndex{items: make([]rectNode, 0, h.ownIDs+h.refs)},
		lines:  make([]Line, 0, h.lines),
		hits:   make([]hit, 0, h.ids),
	}
	solve(root, rect, &b, 0)
	f := b.frame()
	root.SetCached(&Solved{Rect: rect, Frame: f})
	return f
}

// countHintsPruned is countHints with cached-subtree pruning: a node carrying a
// Solved contributes its frame's reachable lengths (so hits presize exactly)
// but no own ids (its rect index is linked, not rebuilt), and is not walked. It
// bounds the cached layout's allocation pass to the changed path.
func countHintsPruned(n *Node) layoutHints {
	if !n.IsVisible() {
		return layoutHints{}
	}
	if s := n.Cached(); s != nil {
		return layoutHints{ids: s.Frame.RectCount(), refs: 1, lines: len(s.Frame.Lines)}
	}
	h := layoutHints{}
	if n.ID != "" {
		h.ids++
		h.ownIDs++
	}
	if n.Content != nil {
		switch {
		case len(n.Content.Lines) > 0:
			h.lines += len(n.Content.Lines)
		case n.Content.Text != "":
			h.lines += strings.Count(n.Content.Text, "\n") + 1
		}
	}
	for i := range n.Children {
		ch := countHintsPruned(&n.Children[i])
		h.ids += ch.ids
		h.ownIDs += ch.ownIDs
		h.refs += ch.refs
		h.lines += ch.lines
	}
	return h
}

// addFrame splices a solved subtree frame into b. Its rect index is linked by
// pointer (O(1): one ref append plus the cached count), and the ordered
// Lines/hits/OverlayFrames are appended so z order and hit ties are preserved
// exactly as if the subtree had been walked. The appended Frame and Line
// values are shallow copies that stay backed by the cached frame's storage,
// which is read-only by contract, so sharing the index is safe.
func (b *frameBuilder) addFrame(f Frame) {
	if f.index != nil {
		b.index.items = append(b.index.items, rectNode{ref: f.index})
		b.index.count += f.index.count
	}
	if len(f.Lines) > 0 {
		b.lines = append(b.lines, f.Lines...)
	}
	if len(f.hits) > 0 {
		b.hits = append(b.hits, f.hits...)
	}
	if len(f.OverlayFrames) > 0 {
		b.overlays = append(b.overlays, f.OverlayFrames...)
	}
	if f.HasCursor {
		b.hasCursor = true
		b.cursorRect = f.CursorRect
		b.cursorShape = f.CursorShape
	}
}

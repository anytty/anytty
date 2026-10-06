package kernel

// Rect is an absolute half-open cell rectangle: columns [X, X+Width) and
// rows [Y, Y+Height).
type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

// Empty reports whether the rect covers no cell.
func (r Rect) Empty() bool {
	return r.Width <= 0 || r.Height <= 0
}

// Area returns the number of cells covered by the rect (0 when empty).
func (r Rect) Area() int {
	if r.Empty() {
		return 0
	}
	return r.Width * r.Height
}

// Contains reports whether the cell (x, y) is inside the rect.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

// Inset shrinks the rect by n cells on every side, clamping at zero size.
func (r Rect) Inset(n int) Rect {
	return Rect{
		X:      r.X + n,
		Y:      r.Y + n,
		Width:  max(0, r.Width-2*n),
		Height: max(0, r.Height-2*n),
	}
}

// Intersect returns the overlap of two rects (possibly empty).
func (r Rect) Intersect(o Rect) Rect {
	x := max(r.X, o.X)
	y := max(r.Y, o.Y)
	right := min(r.X+r.Width, o.X+o.Width)
	bottom := min(r.Y+r.Height, o.Y+o.Height)
	return Rect{X: x, Y: y, Width: max(0, right-x), Height: max(0, bottom-y)}
}

// Line is one rendered horizontal text run positioned in absolute cell
// coordinates. Style is opaque: an explicit style string or a host-internal
// token resolved by the host renderer; the kernel never emits ANSI escape
// bytes.
type Line struct {
	X     int
	Y     int
	Text  string
	Style string
}

// rectEntry is the solved absolute rect of one id-bearing node.
type rectNode struct {
	// id/rect describe one node's absolute rect; ref is set instead when this
	// item is a whole referenced index (a reused cached subtree or a Pos
	// overlay), so solve order is preserved exactly with one ordered list.
	id   string
	rect Rect
	ref  *rectIndex
}

type rectIndex struct {
	items []rectNode
	count int
}

// rect returns the last rect recorded for id, walking the index in solve
// order. Ids are unique in a well-formed view (PROTOCOL §2); when they are
// not, the last entry in solve order wins, matching the flat-map behavior this
// index replaced (a plain Layout and a LayoutCached produce the same order).
func (x *rectIndex) rect(id string) (Rect, bool) {
	var out Rect
	found := false
	if x == nil {
		return out, false
	}
	for i := range x.items {
		it := &x.items[i]
		if it.ref != nil {
			if r, ok := it.ref.rect(id); ok {
				out, found = r, true
			}
			continue
		}
		if it.id == id {
			out, found = it.rect, true
		}
	}
	return out, found
}

// iterate calls fn for every reachable entry until fn returns false, in solve
// order (own entries and referenced indexes interleaved as they were solved).
func (x *rectIndex) iterate(fn func(id string, r Rect) bool) bool {
	if x == nil {
		return true
	}
	for i := range x.items {
		it := &x.items[i]
		if it.ref != nil {
			if !it.ref.iterate(fn) {
				return false
			}
			continue
		}
		if !fn(it.id, it.rect) {
			return false
		}
	}
	return true
}

// Frame is the solved layout of one subtree.
//
// The rect surface (every node with a non-empty id, including nodes inside Pos
// subtrees at absolute coordinates) lives in a tree-shaped index: a reused
// subtree's index is linked by pointer, never copied. Use Rect, RectCount and
// RectsIterate to read it. Lines holds the regular-flow text runs of this frame
// only; Pos subtree output lives in OverlayFrames, in compositing order (later
// frames draw on top). Cursor fields describe the last cursor in z order for
// this frame.
type Frame struct {
	Lines         []Line
	OverlayFrames []Frame
	CursorRect    Rect
	HasCursor     bool
	CursorShape   string

	index *rectIndex
	hits  []hit
}

type hit struct {
	id   string
	rect Rect
}

// Rect looks up the absolute rect of a node by id. It walks the tree-shaped
// rect index; an index shared by a spliced subtree is followed, so the lookup
// sees reused rects too.
func (f Frame) Rect(id string) (Rect, bool) {
	if f.index == nil {
		return Rect{}, false
	}
	return f.index.rect(id)
}

// RectCount returns the number of id-bearing nodes reachable in this frame's
// rect surface (own nodes plus every spliced subtree). It is O(1).
func (f Frame) RectCount() int {
	if f.index == nil {
		return 0
	}
	return f.index.count
}

// RectsIterate calls fn for every id-bearing node in this frame's rect surface,
// including nodes inside spliced subtrees and Pos overlays. fn returns false to
// stop.
func (f Frame) RectsIterate(fn func(id string, r Rect) bool) {
	if f.index == nil {
		return
	}
	f.index.iterate(fn)
}

// Hit returns the id of the topmost, smallest-area visible node containing
// (x, y). Empty ids and zero-area nodes never win; equal areas go to the
// later declaration in z order. It returns "" when nothing is hit.
func (f Frame) Hit(x, y int) string {
	best := ""
	bestArea := 0
	found := false
	for _, h := range f.allHits() {
		if !h.rect.Contains(x, y) {
			continue
		}
		area := h.rect.Area()
		if area <= 0 {
			continue
		}
		if !found || area <= bestArea {
			best, bestArea, found = h.id, area, true
		}
	}
	return best
}

// HitTest is a convenience wrapper around Layout followed by Frame.Hit.
func HitTest(root *Node, width, height, x, y int) string {
	return Layout(root, width, height).Hit(x, y)
}

// allHits returns hit candidates in z order: regular flow first, then every
// overlay subtree in compositing order.
func (f Frame) allHits() []hit {
	out := make([]hit, 0, len(f.hits))
	out = append(out, f.hits...)
	for i := range f.OverlayFrames {
		out = append(out, f.OverlayFrames[i].allHits()...)
	}
	return out
}

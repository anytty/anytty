package kernel

import "strings"

// Flow selects how the children of a container are placed.
type Flow string

const (
	// FlowCol stacks children vertically (the default).
	FlowCol Flow = "col"
	// FlowRow places children left to right.
	FlowRow Flow = "row"
	// FlowStack places every child on top of the parent content rect.
	FlowStack Flow = "stack"
)

// Size is the declared box geometry. Width/Height <= 0 means "unset": the
// axis falls back to the intrinsic content size, then to stretching.
// Flex participates in the leftover distribution of the parent's main axis.
type Size struct {
	Width  int
	Height int
	Flex   int
}

// Pos takes a box out of the regular flow. Coordinates are relative to the
// parent rect (the parent content area is the full rect: the kernel has no
// border concept, chrome is drawn by the content or the component).
type Pos struct {
	X int
	Y int
}

// Content is the textual payload of a box. Lines wins over Text when both
// are set; Text is split on "\n". Self references a host content source and
// contributes no intrinsic size (the component owns its own geometry and
// chrome). Props are program-declared component properties/styles: the kernel
// only carries them, the host passes them through and the component
// interprets the keys it knows (unknown keys are ignored).
type Content struct {
	Text  string
	Lines []string
	Self  string
	Props map[string]string
}

// Cursor is a program cursor relative to the box rect. Visible defaults to
// true when the cursor is present.
type Cursor struct {
	Row     int
	Col     int
	Shape   string
	Visible *bool
}

// Node is one box of the view tree. The box rect is pure geometry: the
// kernel never draws borders or insets. Chrome (borders, titles, badges,
// colors) belongs to the content or the host component.
//
// A Node is immutable once solved: Layout/LayoutCached only read the exported
// fields, and the host treats reused subtrees as read-only. The unexported
// cache field below is the only mutable state (written by LayoutCached).
type Node struct {
	ID      string
	Size    Size
	Pos     *Pos
	Flow    Flow
	Visible *bool
	// Style is an opaque content style; the host renderer resolves it
	// (explicit style string or host-internal token). Empty stays unset.
	Style    string
	Content  *Content
	Cursor   *Cursor
	Input    []string
	Focused  bool
	Children []Node

	// solved is the cached layout of this subtree (see Solved). It is
	// written by LayoutCached after a successful solve and is immutable
	// afterwards. It is a pointer so an unsolved Node stays small (the host
	// builds millions of them) and only solved nodes carry a frame.
	solved *Solved
}

// Solved is a cached solved layout of one subtree: the absolute viewport
// rect the subtree was solved at plus the resulting frame.
//
// Frames store absolute coordinates, so a Solved is only reusable at the
// identical Rect; LayoutCached gates on Rect equality and solves normally
// when the rect moved or resized. The stored Frame is immutable: callers
// must never mutate its index, Lines, hits or OverlayFrames.
type Solved struct {
	Rect  Rect
	Frame Frame
}

// Cached returns the solved layout recorded by SetCached, or nil when the
// subtree has no cached frame. The returned value is read-only.
func (n *Node) Cached() *Solved {
	if n == nil {
		return nil
	}
	return n.solved
}

// SetCached records a solved layout for this subtree, replacing any previous
// value. The passed Solved becomes owned by the node; the caller must not
// mutate it afterwards. A nil value clears the cache.
func (n *Node) SetCached(s *Solved) {
	if n == nil {
		return
	}
	n.solved = s
}

// IsVisible reports the effective visibility (default true).
func (n *Node) IsVisible() bool {
	return n.Visible == nil || *n.Visible
}

// EffectiveFlow returns the normalized flow value (default FlowCol).
func (n *Node) EffectiveFlow() Flow {
	switch n.Flow {
	case FlowRow:
		return FlowRow
	case FlowStack:
		return FlowStack
	default:
		return FlowCol
	}
}

// CursorVisible reports whether the box carries a cursor to render.
func (n *Node) CursorVisible() bool {
	if n.Cursor == nil {
		return false
	}
	return n.Cursor.Visible == nil || *n.Cursor.Visible
}

// ContentLines returns the effective content lines, or nil when the box has
// no intrinsic text. Lines takes precedence over Text.
func (n *Node) ContentLines() []string {
	if n.Content == nil {
		return nil
	}
	if len(n.Content.Lines) > 0 {
		return n.Content.Lines
	}
	if n.Content.Text == "" {
		return nil
	}
	return strings.Split(n.Content.Text, "\n")
}

// forEachContentLine calls fn for every effective content line, mirroring
// ContentLines exactly but without allocating the split result. fn returns
// false to stop iterating.
func forEachContentLine(n *Node, fn func(i int, line string) bool) {
	if n.Content == nil {
		return
	}
	if len(n.Content.Lines) > 0 {
		for i, line := range n.Content.Lines {
			if !fn(i, line) {
				return
			}
		}
		return
	}
	text := n.Content.Text
	if text == "" {
		return
	}
	for i := 0; ; i++ {
		j := strings.IndexByte(text, '\n')
		if j < 0 {
			fn(i, text)
			return
		}
		if !fn(i, text[:j]) {
			return
		}
		text = text[j+1:]
	}
}

// IntrinsicSize returns the content-derived box size. (0, 0) means "no
// intrinsic size", so the axis stretches. Chrome is not part of the node:
// components declare their own inset and draw their own borders.
func (n *Node) IntrinsicSize() (int, int) {
	if n.Content == nil {
		return 0, 0
	}
	if len(n.Content.Lines) > 0 {
		width := 0
		for _, line := range n.Content.Lines {
			if w := DisplayWidth(line); w > width {
				width = w
			}
		}
		return width, len(n.Content.Lines)
	}
	text := n.Content.Text
	if text == "" {
		return 0, 0
	}
	width, lineWidth, lines := 0, 0, 1
	for _, r := range text {
		if r == '\n' {
			if lineWidth > width {
				width = lineWidth
			}
			lineWidth = 0
			lines++
			continue
		}
		lineWidth += RuneWidth(r)
	}
	if lineWidth > width {
		width = lineWidth
	}
	return width, lines
}

// AcceptsInput reports whether the box declares the given input kind
// ("key", "paste", "mouse", "wheel").
func (n *Node) AcceptsInput(kind string) bool {
	for _, in := range n.Input {
		if in == kind {
			return true
		}
	}
	return false
}

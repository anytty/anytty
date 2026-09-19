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

// IntrinsicSize returns the content-derived box size. (0, 0) means "no
// intrinsic size", so the axis stretches. Chrome is not part of the node:
// components declare their own inset and draw their own borders.
func (n *Node) IntrinsicSize() (int, int) {
	lines := n.ContentLines()
	if len(lines) == 0 {
		return 0, 0
	}
	width := 0
	for _, line := range lines {
		if w := DisplayWidth(line); w > width {
			width = w
		}
	}
	return width, len(lines)
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

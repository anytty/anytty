package runtime

import (
	"strings"

	"github.com/anytty/anytty/clients/tui/kernel"
	"github.com/anytty/anytty/clients/tui/render"
)

// Placement is one builtin component instance in a composite: the absolute
// rect the kernel solved for its box plus the styled lines the component
// rendered at its own origin (relative coordinates). Lines outside the rect
// are clipped; the rect is opaque, so blank cells cover whatever is beneath.
type Placement struct {
	Rect  kernel.Rect
	Lines []render.Line
	// Props are the program-declared component properties/styles of the box
	// that produced this placement (content.props), passed through unchanged
	// to the component factory. The compositor never interprets them.
	Props map[string]string
	// Focused marks the placement that owns the terminal cursor when neither
	// the core overlay nor the program draws one.
	Focused bool
	// CursorX and CursorY are the cursor cell relative to the placement's
	// rendered frame (chrome inset included); CursorVisible turns it on.
	CursorX       int
	CursorY       int
	CursorVisible bool
}

// Compositor merges the solved program frame (regular flow plus its `pos`
// overlay subtrees), the builtin component frames, a host notice and the core
// overlay into one framebuffer. The z order is fixed (PROTOCOL §6.4, §9.1):
//
//	program regular flow
//	component placements (given order)
//	program `pos` overlay subtrees (declaration order)
//	notice
//	core overlay (always last, never coverable)
//
// The Compositor is a pure renderer: it never mutates session state and is
// safe to reuse across frames of the same size. Named style tokens resolve
// through the render default palette for host-internal chrome; program
// styles are explicit and need no palette.
type Compositor struct {
	cols, rows int
}

// NewCompositor returns a compositor for a cols×rows viewport.
func NewCompositor(cols, rows int) *Compositor {
	return &Compositor{cols: max(0, cols), rows: max(0, rows)}
}

// SetSize updates the viewport used by the next Compose.
func (c *Compositor) SetSize(cols, rows int) {
	c.cols = max(0, cols)
	c.rows = max(0, rows)
}

// Size returns the current viewport.
func (c *Compositor) Size() (cols, rows int) { return c.cols, c.rows }

// Compose draws the ordered layers into a fresh framebuffer and resolves the
// hardware cursor policy: core overlay > program cursor > focused terminal
// cursor. While the core overlay is open it owns the cursor, so a core
// overlay without its own cursor hides the hardware cursor.
//
// A component placed inside a program `pos` subtree is drawn as part of that
// subtree (after its lines, before nested overlays) instead of in the global
// component layer: the subtree is blitted opaquely, so the global layer would
// otherwise be erased by the component's own overlay bounds.
func (c *Compositor) Compose(program kernel.Frame, placements []Placement, notice, core *kernel.Frame) *render.Frame {
	frame := render.NewFrame(c.cols, c.rows)
	blitKernel(frame, program.Lines)
	top := make([]Placement, 0, len(placements))
	for _, placement := range placements {
		if placementInOverlays(program.OverlayFrames, placement.Rect) {
			top = append(top, placement)
			continue
		}
		blitPlacement(frame, placement)
	}
	groups, _ := assignOverlayPlacements(program.OverlayFrames, top)
	for i := range program.OverlayFrames {
		blitOverlayFrame(frame, &program.OverlayFrames[i], groups[i])
	}
	if notice != nil {
		blitKernelFrame(frame, notice)
	}
	if core != nil {
		blitKernelFrame(frame, core)
	}
	if x, y, ok := resolveCursor(program, placements, core); ok {
		frame.SetCursor(x, y)
	}
	return frame
}

// placementInOverlays reports whether the rect lies inside any top-level
// program overlay subtree.
func placementInOverlays(overlays []kernel.Frame, rect kernel.Rect) bool {
	for i := range overlays {
		if rectWithin(overlayBounds(&overlays[i]), rect) {
			return true
		}
	}
	return false
}

// assignOverlayPlacements groups placements by the last (topmost, deepest)
// overlay frame that fully contains them. Sibling `pos` subtrees may overlap
// (a card-sized terminal box and a floating terminal inside it); a placement
// must belong to the last declared overlay that covers it, because every
// overlay blit is opaque and an earlier owner would be erased by the later
// overlay's fill. Placements outside every frame belong to the caller's
// level and are returned in rest.
func assignOverlayPlacements(overlays []kernel.Frame, placements []Placement) ([][]Placement, []Placement) {
	if len(overlays) == 0 {
		return nil, placements
	}
	bounds := make([]kernel.Rect, len(overlays))
	for i := range overlays {
		bounds[i] = overlayBounds(&overlays[i])
	}
	groups := make([][]Placement, len(overlays))
	var rest []Placement
	for _, placement := range placements {
		target := -1
		for i := range bounds {
			if rectWithin(bounds[i], placement.Rect) {
				target = i
			}
		}
		if target >= 0 {
			groups[target] = append(groups[target], placement)
			continue
		}
		rest = append(rest, placement)
	}
	return groups, rest
}

// rectWithin reports whether inner is fully contained in outer.
func rectWithin(outer kernel.Rect, inner kernel.Rect) bool {
	if inner.Empty() || outer.Empty() {
		return false
	}
	return inner.X >= outer.X && inner.Y >= outer.Y &&
		inner.X+inner.Width <= outer.X+outer.Width &&
		inner.Y+inner.Height <= outer.Y+outer.Height
}

// blitOverlayFrame draws one program `pos` subtree opaquely and interleaves
// the component placements that belong to its nodes: the frame's own lines,
// then its components, then nested overlays (which own their placements).
func blitOverlayFrame(frame *render.Frame, k *kernel.Frame, placements []Placement) {
	if k == nil {
		return
	}
	fillRect(frame, overlayBounds(k))
	blitKernel(frame, k.Lines)
	groups, remaining := assignOverlayPlacements(k.OverlayFrames, placements)
	for _, placement := range remaining {
		blitPlacement(frame, placement)
	}
	for i := range k.OverlayFrames {
		blitOverlayFrame(frame, &k.OverlayFrames[i], groups[i])
	}
}

// overlayBounds is the union of every rect of an overlay subtree; the root of
// a `pos` subtree is its outer box, so the union is exactly that box.
func overlayBounds(k *kernel.Frame) kernel.Rect {
	var bounds kernel.Rect
	first := true
	for _, rect := range k.Rects {
		if rect.Empty() {
			continue
		}
		if first {
			bounds, first = rect, false
			continue
		}
		right := max(bounds.X+bounds.Width, rect.X+rect.Width)
		bottom := max(bounds.Y+bounds.Height, rect.Y+rect.Height)
		bounds.X = min(bounds.X, rect.X)
		bounds.Y = min(bounds.Y, rect.Y)
		bounds.Width = right - bounds.X
		bounds.Height = bottom - bounds.Y
	}
	return bounds
}

// fillRect blanks every cell of a rect with the default style.
func fillRect(frame *render.Frame, rect kernel.Rect) {
	if rect.Empty() {
		return
	}
	text := strings.Repeat(" ", rect.Width)
	for y := rect.Y; y < rect.Y+rect.Height; y++ {
		frame.BlitLine(render.Line{X: rect.X, Y: y, Text: text, Style: render.TokenDefault})
	}
}

// blitKernelFrame draws one kernel frame and its nested overlays in order.
func blitKernelFrame(frame *render.Frame, k *kernel.Frame) {
	if k == nil {
		return
	}
	blitKernel(frame, k.Lines)
	for i := range k.OverlayFrames {
		blitKernelFrame(frame, &k.OverlayFrames[i])
	}
}

func blitKernel(frame *render.Frame, lines []kernel.Line) {
	for _, line := range lines {
		frame.BlitLine(render.Line{X: line.X, Y: line.Y, Text: line.Text, Style: render.Token(line.Style)})
	}
}

// blitPlacement clips component lines to their placement rect by rendering
// them into a rect-sized scratch frame. The scratch frame starts blank, so
// the blit also makes the component opaque over the layers beneath.
func blitPlacement(frame *render.Frame, placement Placement) {
	rect := placement.Rect
	if rect.Empty() {
		return
	}
	scratch := render.NewFrame(rect.Width, rect.Height)
	scratch.SetTheme(frame.Theme())
	scratch.Blit(placement.Lines...)
	frame.BlitFrame(scratch, rect.X, rect.Y)
}

// resolveCursor applies the cursor priority. The boolean is false when the
// hardware cursor stays hidden.
func resolveCursor(program kernel.Frame, placements []Placement, core *kernel.Frame) (int, int, bool) {
	if core != nil {
		if core.HasCursor {
			return core.CursorRect.X, core.CursorRect.Y, true
		}
		return 0, 0, false
	}
	if program.HasCursor {
		return program.CursorRect.X, program.CursorRect.Y, true
	}
	for i := range placements {
		placement := &placements[i]
		if placement.Focused && placement.CursorVisible {
			return placement.Rect.X + placement.CursorX, placement.Rect.Y + placement.CursorY, true
		}
	}
	return 0, 0, false
}

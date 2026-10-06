package widgets

import "github.com/anytty/anytty/clients/tui/sdk"

// ContextMenu is a menu anchored at a click point. It embeds Menu for its
// items/selection and adds the overlay placement: Open records the click and
// resolves a clamped top-left (AnchorX/AnchorY) that keeps the whole menu
// inside the parent rect, Close hides it and Build emits the positioned
// FloatingLayer. It stays a pure widget: the caller decides whether a click
// dismisses it.
type ContextMenu struct {
	Menu
	// X, Y is the requested (click) anchor.
	X, Y int
	// Visible toggles the overlay.
	Visible bool
	// AnchorX, AnchorY is the resolved clamped top-left.
	AnchorX, AnchorY int
	// ParentWidth/ParentHeight bound the placement; <= 0 disables clamping.
	ParentWidth  int
	ParentHeight int
	// Margin keeps that many cells between the menu and the parent edges.
	Margin int
}

// Open places the menu at the click point (x, y), clamped so it fits inside
// the parent rect, marks it visible and returns the resolved top-left.
func (c *ContextMenu) Open(x, y int) (int, int) {
	c.X, c.Y = x, y
	c.Visible = true
	c.AnchorX, c.AnchorY = c.place()
	c.Menu.X, c.Menu.Y = c.AnchorX, c.AnchorY
	return c.AnchorX, c.AnchorY
}

// Close hides the menu.
func (c *ContextMenu) Close() { c.Visible = false }

// MenuWidth returns the declared menu width (0 when unset).
func (c ContextMenu) MenuWidth() int { return c.Menu.Width }

// MenuHeight returns the framed overlay height for the current items: one row
// per item plus a two-cell frame, at least two.
func (c ContextMenu) MenuHeight() int {
	height := len(c.Items) + 2
	if height < 2 {
		height = 2
	}
	return height
}

// place clamps (X, Y) into the parent rect, keeping at least Margin cells of
// breathing room and never going negative.
func (c ContextMenu) place() (int, int) {
	margin := c.Margin
	if margin < 0 {
		margin = 0
	}
	x, y := c.X, c.Y
	width, height := c.Menu.Width, c.MenuHeight()
	if c.ParentWidth > 0 {
		if x+width > c.ParentWidth-margin {
			x = c.ParentWidth - margin - width
		}
	}
	if c.ParentHeight > 0 {
		if y+height > c.ParentHeight-margin {
			y = c.ParentHeight - margin - height
		}
	}
	if x < margin {
		x = margin
	}
	if y < margin {
		y = margin
	}
	return x, y
}

// Build returns the positioned overlay. A hidden menu renders an invisible
// box, so callers can drop it into the view unconditionally.
func (c ContextMenu) Build() *sdk.Builder {
	if !c.Visible {
		return sdk.Box().Visible(false)
	}
	c.AnchorX, c.AnchorY = c.place()
	c.Menu.X, c.Menu.Y = c.AnchorX, c.AnchorY
	return c.Menu.Build()
}

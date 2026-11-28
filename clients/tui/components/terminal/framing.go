package terminal

// Extent is the one description of where the virtual screen sits inside a
// component's content rectangle, shared by rendering, the cursor and pointer
// routing. Every consumer must use these helpers instead of repeating the
// subtraction; that is what kept drifting (0-based vs 1-based, missing offset,
// stale size).
//
// Coordinates are 0-based cells inside the content rect:
//
//	panel cell (px,py)  ->  screen cell (px-X, py-Y)
//
// X/Y may be negative when the screen is larger than the pane (bottom/right
// anchored followers), and Cols/Rows may exceed the pane; every helper clips.
type Extent struct {
	X, Y       int
	Cols, Rows int
	Sized      bool
}

// ExtentFromProps resolves content.offset/content.size against a pane size.
// A missing size means the screen covers the whole content rect (the legacy
// byte-identical path).
func ExtentFromProps(chrome map[string]string, paneCols, paneRows int) Extent {
	parsed := ContentOffsetFromProps(chrome)
	e := Extent{X: parsed.X, Y: parsed.Y, Cols: parsed.Cols, Rows: parsed.Rows, Sized: parsed.Sized}
	if !e.Sized || e.Cols <= 0 || e.Rows <= 0 {
		e.Cols, e.Rows = paneCols, paneRows
		e.Sized = false
	}
	return e
}

// PanelToScreen maps a 0-based panel cell to a 0-based screen cell and reports
// whether it lands on the screen (inside the extent).
func (e Extent) PanelToScreen(px, py int) (sx, sy int, inside bool) {
	sx, sy = px-e.X, py-e.Y
	return sx, sy, sx >= 0 && sy >= 0 && sx < e.Cols && sy < e.Rows
}

// ScreenToPanel maps a 0-based screen cell to a 0-based panel cell. inside
// reports whether the mapped cell still falls inside the pane rectangle.
func (e Extent) ScreenToPanel(sx, sy, paneCols, paneRows int) (px, py int, inside bool) {
	px, py = sx+e.X, sy+e.Y
	return px, py, px >= 0 && py >= 0 && px < paneCols && py < paneRows
}

// RowSpan returns the clipped panel column range [start,end) that the screen
// occupies on one panel row, and the screen column the range starts at. ok is
// false when the row is entirely outside the extent.
func (e Extent) RowSpan(panelCols, contentRow int) (start, end, screenStart int, ok bool) {
	sy := contentRow - e.Y
	if sy < 0 || sy >= e.Rows {
		return 0, 0, 0, false
	}
	start = max(0, e.X)
	end = min(panelCols, e.X+e.Cols)
	if start >= end {
		return 0, 0, 0, false
	}
	return start, end, start - e.X, true
}

package render

import "bytes"

// Line is one styled horizontal text run at an absolute cell position. Y is
// a row, X is the starting column. Text is plain UTF-8; ANSI bytes are never
// embedded (the host renderer owns all escape output).
type Line struct {
	X     int
	Y     int
	Text  string
	Style Token
}

// Cell is one framebuffer cell. Text holds exactly one grapheme cluster
// (a single rune for ASCII, a whole cluster for emoji); Width is still
// reported for wide cells. The second half of a wide cluster is a
// Continuation cell with empty text.
type Cell struct {
	Text         string
	Width        int
	Continuation bool
	Style        Token
}

// Blank reports whether the cell carries no content or style.
func (c Cell) Blank() bool {
	return c.Text == "" && !c.Continuation && c.Style == ""
}

// Frame is a fixed-size cell grid plus a cursor. Blit styled lines into it,
// then call Bytes(prev) to obtain the minimal ANSI update for the TTY. The
// zero value is not usable; use NewFrame.
type Frame struct {
	cols  int
	rows  int
	cells []Cell

	cursorX       int
	cursorY       int
	cursorVisible bool

	theme Theme
}

// NewFrame allocates an empty cols×rows framebuffer with the default theme.
func NewFrame(cols, rows int) *Frame {
	if cols < 0 {
		cols = 0
	}
	if rows < 0 {
		rows = 0
	}
	return &Frame{
		cols:  cols,
		rows:  rows,
		cells: make([]Cell, cols*rows),
		theme: DefaultTheme(),
	}
}

// Cols returns the grid width in cells.
func (f *Frame) Cols() int { return f.cols }

// Rows returns the grid height in rows.
func (f *Frame) Rows() int { return f.rows }

// SetTheme sets the palette used when resolving host-internal tokens.
// Program styles are explicit and ignore the palette entirely.
func (f *Frame) SetTheme(theme Theme) { f.theme = theme.WithFallback() }

// Theme returns the effective palette.
func (f *Frame) Theme() Theme { return f.theme.WithFallback() }

// Clear resets every cell and hides the cursor.
func (f *Frame) Clear() {
	for i := range f.cells {
		f.cells[i] = Cell{}
	}
	f.cursorVisible = false
}

// CellAt returns the cell at (x, y), or the zero Cell when out of bounds.
func (f *Frame) CellAt(x, y int) Cell {
	if f == nil || x < 0 || y < 0 || x >= f.cols || y >= f.rows {
		return Cell{}
	}
	return f.cells[y*f.cols+x]
}

// SetCursor places the visible cursor at the zero-based cell (x, y).
func (f *Frame) SetCursor(x, y int) {
	f.cursorX = x
	f.cursorY = y
	f.cursorVisible = true
}

// HideCursor hides the cursor.
func (f *Frame) HideCursor() { f.cursorVisible = false }

// Cursor returns the cursor position and visibility.
func (f *Frame) Cursor() (x, y int, visible bool) {
	return f.cursorX, f.cursorY, f.cursorVisible
}

// Blit draws lines in order; later lines overwrite earlier ones.
func (f *Frame) Blit(lines ...Line) {
	for _, line := range lines {
		f.BlitLine(line)
	}
}

// BlitLine draws one styled run at its absolute position. The run is clipped
// to the grid; a wide grapheme is dropped rather than split when only one of
// its two cells fits. Overwriting half of a wide grapheme clears the other
// half so the grid never contains orphaned continuation cells.
func (f *Frame) BlitLine(line Line) {
	if f == nil || line.Y < 0 || line.Y >= f.rows {
		return
	}
	x := line.X
	lastHead := -1
	for _, cluster := range Clusters(VisibleText(line.Text)) {
		if cluster.Width == 0 {
			// Zero-width clusters never advance; glue them to the previous
			// head cell so combining marks survive a blit.
			if lastHead >= 0 {
				f.cells[lastHead].Text += cluster.Text
			}
			continue
		}
		if x >= f.cols {
			break
		}
		if x < 0 {
			x += cluster.Width
			lastHead = -1
			continue
		}
		if x+cluster.Width > f.cols {
			// A wide grapheme cannot be split across the right edge.
			break
		}
		head := line.Y*f.cols + x
		f.clearAt(x, line.Y)
		if cluster.Width > 1 {
			f.clearAt(x+1, line.Y)
			f.cells[head] = Cell{Text: cluster.Text, Width: cluster.Width, Style: line.Style}
			f.cells[head+1] = Cell{Continuation: true, Style: line.Style}
		} else {
			f.cells[head] = Cell{Text: cluster.Text, Width: 1, Style: line.Style}
		}
		lastHead = head
		x += cluster.Width
	}
}

// BlitFrame copies src into f with its top-left corner at (x, y). Blank src
// cells overwrite (components are opaque), styled blanks keep their style,
// continuation halves are rebuilt at the destination and a wide grapheme that
// falls off any edge is dropped, never split. Out-of-bounds source columns
// are clipped.
func (f *Frame) BlitFrame(src *Frame, x, y int) {
	if f == nil || src == nil {
		return
	}
	for sy := 0; sy < src.rows; sy++ {
		dy := y + sy
		if dy < 0 || dy >= f.rows {
			continue
		}
		for sx := 0; sx < src.cols; sx++ {
			srcCell := src.cells[sy*src.cols+sx]
			if srcCell.Continuation {
				continue
			}
			dx := x + sx
			if dx >= f.cols {
				break
			}
			if dx < 0 {
				// A cell clipped on the left leaves nothing to draw; the
				// visible half of a wide head is still blanked so the layer
				// stays opaque.
				if srcCell.Width > 1 && dx+1 >= 0 && dx+1 < f.cols {
					f.clearAt(dx+1, dy)
				}
				continue
			}
			if srcCell.Text == "" && srcCell.Style == "" {
				f.clearAt(dx, dy)
				continue
			}
			f.clearAt(dx, dy)
			if srcCell.Width > 1 {
				if dx+1 >= f.cols {
					continue
				}
				f.clearAt(dx+1, dy)
				f.cells[dy*f.cols+dx] = Cell{Text: srcCell.Text, Width: srcCell.Width, Style: srcCell.Style}
				f.cells[dy*f.cols+dx+1] = Cell{Continuation: true, Style: srcCell.Style}
				continue
			}
			f.cells[dy*f.cols+dx] = Cell{Text: srcCell.Text, Width: 1, Style: srcCell.Style}
		}
	}
}

// clearAt empties one cell, cleaning up the partner cell of a wide grapheme:
// a continuation clears its head, a head clears its continuation.
func (f *Frame) clearAt(x, y int) {
	idx := y*f.cols + x
	if f.cells[idx].Continuation && x > 0 {
		f.cells[idx-1] = Cell{}
	}
	if f.cells[idx].Width > 1 && x+1 < f.cols {
		f.cells[idx+1] = Cell{}
	}
	f.cells[idx] = Cell{}
}

// Equal reports whether two frames have identical dimensions, cells and
// cursor state.
func (f *Frame) Equal(o *Frame) bool {
	if f == nil || o == nil {
		return f == o
	}
	if f.cols != o.cols || f.rows != o.rows {
		return false
	}
	if f.cursorX != o.cursorX || f.cursorY != o.cursorY || f.cursorVisible != o.cursorVisible {
		return false
	}
	for i := range f.cells {
		if f.cells[i] != o.cells[i] {
			return false
		}
	}
	return true
}

// Bytes returns the minimal ANSI update that turns prev into f. A nil prev
// (or a size change) yields a full repaint. When nothing changed — content,
// styles and cursor alike — it returns nil.
func (f *Frame) Bytes(prev *Frame) []byte {
	if f == nil {
		return nil
	}
	if prev == nil || prev.cols != f.cols || prev.rows != f.rows {
		return f.fullBytes()
	}
	var b []byte
	changed := false
	for y := 0; y < f.rows; y++ {
		if f.rowEqual(prev, y) {
			continue
		}
		if !changed {
			b = append(b, HideCursor...)
			changed = true
		}
		b = append(b, CursorPosition(0, y)...)
		b = append(b, f.rowBytes(y)...)
	}
	cursorMoved := f.cursorVisible != prev.cursorVisible ||
		(f.cursorVisible && (f.cursorX != prev.cursorX || f.cursorY != prev.cursorY))
	if !changed && !cursorMoved {
		return nil
	}
	if f.cursorVisible {
		b = append(b, CursorPosition(f.cursorX, f.cursorY)...)
		b = append(b, ShowCursor...)
	} else if prev.cursorVisible && !changed {
		b = append(b, HideCursor...)
	}
	return b
}

// FullBytes returns an unconditional full repaint.
func (f *Frame) FullBytes() []byte {
	if f == nil {
		return nil
	}
	return f.fullBytes()
}

func (f *Frame) fullBytes() []byte {
	var b []byte
	b = append(b, HideCursor...)
	b = append(b, ResetSGR...)
	for y := 0; y < f.rows; y++ {
		b = append(b, CursorPosition(0, y)...)
		b = append(b, f.rowBytes(y)...)
	}
	if f.cursorVisible {
		b = append(b, CursorPosition(f.cursorX, f.cursorY)...)
		b = append(b, ShowCursor...)
	}
	return b
}

func (f *Frame) rowEqual(o *Frame, y int) bool {
	start := y * f.cols
	for i := start; i < start+f.cols; i++ {
		if f.cells[i] != o.cells[i] {
			return false
		}
	}
	return true
}

// rowBytes renders one row as SGR runs. Every style change starts from a
// reset so the output is independent of what the TTY had before; blank cells
// are emitted as spaces because the cursor may sit anywhere.
func (f *Frame) rowBytes(y int) []byte {
	var b bytes.Buffer
	current := Token("\x00unset")
	start := y * f.cols
	for i := start; i < start+f.cols; i++ {
		cell := f.cells[i]
		if cell.Continuation {
			continue
		}
		style := cell.Style
		if style == "" {
			style = TokenDefault
		}
		if style != current {
			b.WriteString(ResetSGR)
			b.WriteString(style.SGR(f.theme))
			current = style
		}
		if cell.Text == "" {
			b.WriteByte(' ')
		} else {
			b.WriteString(cell.Text)
		}
	}
	b.WriteString(ResetSGR)
	return b.Bytes()
}

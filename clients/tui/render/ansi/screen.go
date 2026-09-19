package ansi

import (
	"strings"

	"github.com/anytty/anytty/clients/tui/render"
)

// Cell is one grid cell of the parsed screen. Head cells of a wide grapheme
// carry Width 2; the cell to their right is a Continuation with empty text.
// Text holds a whole grapheme cluster, so combining marks never split.
type Cell struct {
	Text         string
	Width        int
	Continuation bool
	Style        render.Token
}

// Blank reports whether the cell carries neither text nor style.
func (c Cell) Blank() bool {
	return c.Text == "" && !c.Continuation && c.Style == ""
}

// Screen is a point-in-time snapshot of the parser grid. Lines has exactly
// rows entries and every row has exactly cols cells, so CellAt uses the same
// coordinates as the cursor. The continuation half of a wide grapheme is a
// Width 0 cell with Continuation set; consumers that hold one cell per
// cluster (like the terminal component) skip it.
type Screen struct {
	Lines [][]Cell
	// CursorX and CursorY are zero-based grid coordinates.
	CursorX       int
	CursorY       int
	CursorVisible bool
}

// Row returns the cells of one row, or nil when the row does not exist.
func (s Screen) Row(y int) []Cell {
	if y < 0 || y >= len(s.Lines) {
		return nil
	}
	return s.Lines[y]
}

// RowText flattens one cell row to plain text: wide clusters stay whole,
// interior blank cells become spaces, trailing blank cells are dropped and
// continuation cells contribute nothing. A space the program actually wrote
// is content, not padding, so it survives trailing-blank trimming.
func RowText(row []Cell) string {
	if len(row) == 0 {
		return ""
	}
	var b strings.Builder
	blanks := 0
	for _, cell := range row {
		switch {
		case cell.Continuation:
		case cell.Text == "":
			blanks++
		default:
			for ; blanks > 0; blanks-- {
				b.WriteByte(' ')
			}
			b.WriteString(cell.Text)
		}
	}
	return b.String()
}

// Text returns the plain text of one row.
func (s Screen) Text(y int) string { return RowText(s.Row(y)) }

// TextLines flattens every row to plain text.
func (s Screen) TextLines() []string {
	if len(s.Lines) == 0 {
		return nil
	}
	out := make([]string, len(s.Lines))
	for y := range s.Lines {
		out[y] = s.Text(y)
	}
	return out
}

// CursorOffset maps grid column x to its display-column offset within the
// row's cluster sequence (continuation halves fold into their head, blanks
// count one column). It returns false when (x, y) is out of range.
func (s Screen) CursorOffset(x, y int) (int, bool) {
	row := s.Row(y)
	if len(row) == 0 || x < 0 || x >= len(row) {
		return 0, false
	}
	if row[x].Continuation && x > 0 {
		x--
	}
	offset := 0
	for col := 0; col < x; col++ {
		cell := row[col]
		if cell.Continuation {
			continue
		}
		width := cell.Width
		if width <= 0 {
			width = 1
		}
		offset += width
	}
	return offset, true
}

// CellAt returns the cell at (x, y), or the zero Cell when out of bounds.
// Continuation cells are reported as-is.
func (s Screen) CellAt(x, y int) Cell {
	if y < 0 || y >= len(s.Lines) {
		return Cell{}
	}
	row := s.Lines[y]
	if x < 0 || x >= len(row) {
		return Cell{}
	}
	return row[x]
}

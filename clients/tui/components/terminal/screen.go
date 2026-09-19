package terminal

import (
	"strings"

	"github.com/anytty/anytty/clients/tui/render"
)

// Cell is one screen cell of a terminal source: one grapheme cluster with
// its display width and style token. Wide clusters occupy one Cell with
// Width 2 (the framebuffer expands them into two grid cells when blitted).
type Cell struct {
	Text  string
	Width int
	Style render.Token
}

// Screen is an immutable snapshot of terminal rows as cells.
type Screen struct {
	Lines [][]Cell
}

// ScreenFromText builds a screen from plain text lines. Text is split into
// grapheme clusters so emoji and combining marks never split; control bytes
// are replaced by spaces.
func ScreenFromText(lines []string, style render.Token) Screen {
	if style == "" {
		style = render.TokenDefault
	}
	out := Screen{Lines: make([][]Cell, len(lines))}
	for i, line := range lines {
		clusters := render.Clusters(render.VisibleText(line))
		cells := make([]Cell, 0, len(clusters))
		for _, cluster := range clusters {
			if cluster.Width == 0 {
				if len(cells) > 0 {
					cells[len(cells)-1].Text += cluster.Text
				}
				continue
			}
			cells = append(cells, Cell{Text: cluster.Text, Width: cluster.Width, Style: style})
		}
		out.Lines[i] = cells
	}
	return out
}

// Rows returns the number of screen rows.
func (s Screen) Rows() int { return len(s.Lines) }

// Line returns the cells of one row, or nil when the row does not exist.
func (s Screen) Line(row int) []Cell {
	if row < 0 || row >= len(s.Lines) {
		return nil
	}
	return s.Lines[row]
}

// TextLines flattens every row back to plain text. Continuation cells are
// not represented (a wide cluster is one Cell), so TextLines is lossless.
func (s Screen) TextLines() []string {
	if len(s.Lines) == 0 {
		return nil
	}
	out := make([]string, len(s.Lines))
	for i, line := range s.Lines {
		var b strings.Builder
		for _, cell := range line {
			b.WriteString(cell.Text)
		}
		out[i] = b.String()
	}
	return out
}

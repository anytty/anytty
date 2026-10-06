package kernel

// renderOwn appends the box content lines and its cursor to the frame. The
// content area is the full box rect: the kernel draws no chrome and applies
// no inset (chrome belongs to the content or the host component). Lines
// carry the opaque node style and never contain ANSI escape bytes.
func renderOwn(b *frameBuilder, n *Node, rect Rect) {
	if rect.Width > 0 {
		forEachContentLine(n, func(i int, line string) bool {
			if i >= rect.Height {
				return false
			}
			text := Truncate(line, rect.Width)
			if text == "" {
				return true
			}
			b.lines = append(b.lines, Line{X: rect.X, Y: rect.Y + i, Text: text, Style: n.Style})
			return true
		})
	}
	if n.CursorVisible() && !rect.Empty() {
		row := n.Cursor.Row
		if row < 0 {
			row = 0
		} else if row >= rect.Height {
			row = rect.Height - 1
		}
		col := n.Cursor.Col
		if col < 0 {
			col = 0
		} else if col >= rect.Width {
			col = rect.Width - 1
		}
		b.hasCursor = true
		b.cursorRect = Rect{
			X:      rect.X + col,
			Y:      rect.Y + row,
			Width:  1,
			Height: 1,
		}
		b.cursorShape = n.Cursor.Shape
	}
}

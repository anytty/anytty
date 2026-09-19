package kernel

// renderOwn appends the box content lines and its cursor to the frame. The
// content area is the full box rect: the kernel draws no chrome and applies
// no inset (chrome belongs to the content or the host component). Lines
// carry the opaque node style and never contain ANSI escape bytes.
func renderOwn(f *Frame, n *Node, rect Rect) {
	if rect.Width > 0 {
		lines := n.ContentLines()
		for i, line := range lines {
			if i >= rect.Height {
				break
			}
			text := Truncate(line, rect.Width)
			if text == "" {
				continue
			}
			f.Lines = append(f.Lines, Line{X: rect.X, Y: rect.Y + i, Text: text, Style: n.Style})
		}
	}
	if n.CursorVisible() && !rect.Empty() {
		f.HasCursor = true
		f.CursorRect = Rect{
			X:      rect.X + n.Cursor.Col,
			Y:      rect.Y + n.Cursor.Row,
			Width:  1,
			Height: 1,
		}
		f.CursorShape = n.Cursor.Shape
	}
}

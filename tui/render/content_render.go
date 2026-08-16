package render

import "strings"

func renderContent(c *canvas, content ContentVM, rect Rect, owner string, layer LayerKind) ContentRenderResult {
	return renderContentWithFill(c, content, rect, owner, layer, "")
}

func renderContentWithFill(c *canvas, content ContentVM, rect Rect, owner string, layer LayerKind, fill StyleToken) ContentRenderResult {
	if rect.W <= 0 || rect.H <= 0 {
		return ContentRenderResult{}
	}
	result := RenderContentViewport(ContentRenderRequest{Rect: rect, Content: content})
	for i, line := range result.Lines {
		line = contentLineWithFill(line, fill)
		result.Lines[i] = line
		c.writeLine(rect.X, rect.Y+i, rect.W, line, owner, layer)
	}
	renderWorkbenchNavigatorSnapshotContent(c, content, rect, owner, layer)
	return result
}

func contentLineWithFill(line Line, fill StyleToken) Line {
	if fill == "" {
		return line
	}
	line = line.Clone()
	line.FillStyle = fill
	cells := make([]Cell, 0, len(line.Cells)+2)
	for _, cell := range line.Cells {
		cells = append(cells, contentCellsWithFill(cell, fill)...)
	}
	line.Cells = cells
	return line
}

func contentCellsWithFill(cell Cell, fill StyleToken) []Cell {
	if fill == "" ||
		cell.Style != "" ||
		!cell.ANSIStyle.IsZero() ||
		cell.TerminalContent ||
		cell.LinkURL != "" ||
		cell.LinkParams != "" ||
		cell.Text == "" {
		return []Cell{cell}
	}
	displayWidth := DisplayWidth(cell.Text)
	if displayWidth != cell.Width {
		return []Cell{cell}
	}
	left := leadingASCIISpaceWidth(cell.Text)
	right := trailingASCIISpaceWidth(cell.Text)
	if left == 0 && right == 0 {
		return []Cell{cell}
	}
	if left+right >= len(cell.Text) {
		cell.Style = fill
		return []Cell{cell}
	}
	cells := make([]Cell, 0, 3)
	if left > 0 {
		cells = append(cells, contentFillSpaceCell(left, fill))
	}
	middle := cell
	middle.Text = cell.Text[left : len(cell.Text)-right]
	middle.Width = DisplayWidth(middle.Text)
	cells = append(cells, middle)
	if right > 0 {
		cells = append(cells, contentFillSpaceCell(right, fill))
	}
	return cells
}

func contentFillSpaceCell(width int, fill StyleToken) Cell {
	return Cell{Text: strings.Repeat(" ", width), Width: width, Style: fill, Safe: true}
}

func leadingASCIISpaceWidth(text string) int {
	width := 0
	for width < len(text) && text[width] == ' ' {
		width++
	}
	return width
}

func trailingASCIISpaceWidth(text string) int {
	width := 0
	for index := len(text) - 1; index >= 0 && text[index] == ' '; index-- {
		width++
	}
	return width
}

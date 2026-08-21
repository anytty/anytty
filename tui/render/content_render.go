package render

import "strings"

func renderContent(c *canvas, content ContentVM, rect Rect, owner string, layer LayerKind) ContentRenderResult {
	return renderContentWithFill(c, content, rect, owner, layer, "")
}

func renderSearchBar(c *canvas, search SearchBarVM, rect Rect, owner string, layer LayerKind) {
	if !search.Visible || rect.W <= 0 || rect.H <= 0 {
		return
	}
	line, _ := searchBarPresentation(search, rect.W)
	c.writeLine(rect.X, rect.Y, rect.W, line, owner, layer)
}

func searchBarPresentation(search SearchBarVM, width int) (Line, Cursor) {
	if !search.Visible || width <= 0 {
		return Line{}, Cursor{}
	}
	prefix := search.Prefix.Clone()
	suffix := search.Status.Clone()
	if search.WideHint.Width() > 0 && width >= search.WideMinWidth {
		suffix.Cells = append(suffix.Cells, search.WideHint.Clone().Cells...)
	}
	available := maxInt(1, width-prefix.Width()-suffix.Width())
	value := search.Value
	valueWidth := DisplayWidth(value)
	valueCursor := valueWidth
	if search.CursorPlaced {
		runes := []rune(value)
		cursor := minInt(maxInt(0, search.ValueCursor), len(runes))
		valueCursor = DisplayWidth(string(runes[:cursor]))
	}
	valueStart := 0
	if valueWidth > available {
		if search.CursorPlaced {
			valueStart = minInt(maxInt(0, valueCursor-available/2), valueWidth-available)
		} else {
			valueStart = valueWidth - available
		}
		value = SliceCells(value, valueStart, valueStart+available)
	}
	line := Line{Cells: append([]Cell(nil), prefix.Cells...)}
	line.Cells = append(line.Cells, NewCell(value))
	if gap := width - line.Width() - suffix.Width(); gap > 0 {
		line.Cells = append(line.Cells, NewCell(strings.Repeat(" ", gap)))
	}
	line.Cells = append(line.Cells, suffix.Cells...)
	cursor := search.Cursor
	if cursor.Visible || cursor.Anchor {
		cursor.Row = 0
		valueCol := minInt(maxInt(0, available-1), maxInt(0, valueCursor-valueStart))
		cursor.Col = minInt(maxInt(0, width-1), prefix.Width()+valueCol)
		if suffix.Width() > 0 {
			cursor.Col = minInt(cursor.Col, maxInt(prefix.Width(), width-suffix.Width()-1))
		}
	}
	return line, cursor
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

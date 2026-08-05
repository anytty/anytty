package render

import "strings"

func (c *canvas) lines() []Line {
	lines := make([]Line, len(c.rows))
	for i, row := range c.rows {
		lines[i] = Line{Cells: canvasRowOutputCells(row)}
	}
	return lines
}

func (c *canvas) ansiLines(theme Theme) []string {
	lines := make([]string, len(c.rows))
	for rowIndex, row := range c.rows {
		lines[rowIndex] = ensureANSIReset(canvasRowANSIString(row, theme))
	}
	return lines
}

func canvasRowANSIString(row []canvasCell, theme Theme) string {
	if len(row) == 0 {
		return ""
	}
	return Line{Cells: canvasRowOutputCells(row)}.ANSIString(theme)
}

func canvasRowOutputCells(row []canvasCell) []Cell {
	cells := make([]Cell, 0, canvasOutputCellCapacity(row))
	for col := 0; col < len(row); col++ {
		cell := row[col]
		if cell.continuation {
			continue
		}
		cells = appendCanvasOutputCell(cells, canvasOutputCellFromMatrix(cell))
		if cell.width > 1 {
			col += minInt(cell.width-1, len(row)-col-1)
		}
	}
	return cells
}

func canvasOutputCellCapacity(row []canvasCell) int {
	count := 0
	for col := 0; col < len(row); col++ {
		cell := row[col]
		if cell.continuation {
			continue
		}
		count++
		if cell.width > 1 {
			col += minInt(cell.width-1, len(row)-col-1)
		}
	}
	if count <= 0 {
		return 1
	}
	return count
}

func canvasOutputCellFromMatrix(cell canvasCell) Cell {
	if cell.text == "" && cell.width == 0 {
		return Cell{Text: " ", Width: 1, Safe: true}
	}
	width := cell.width
	if width <= 0 {
		width = 1
	}
	return Cell{
		Text:            cell.text,
		Width:           width,
		Style:           cell.style,
		ANSIStyle:       cell.ansiStyle,
		LinkURL:         cell.linkURL,
		LinkParams:      cell.linkParams,
		TerminalContent: cell.terminal,
		Safe:            cell.safe,
	}
}

func appendCanvasOutputCell(cells []Cell, next Cell) []Cell {
	if len(cells) == 0 || !canMergeCanvasOutputCell(cells[len(cells)-1], next) {
		return append(cells, next)
	}
	last := &cells[len(cells)-1]
	last.Text += next.Text
	last.Width += maxInt(0, next.Width)
	last.Safe = last.Safe && next.Safe
	return cells
}

func canMergeCanvasOutputCell(left Cell, right Cell) bool {
	if left.TerminalContent ||
		right.TerminalContent ||
		left.Style != right.Style ||
		left.ANSIStyle != right.ANSIStyle ||
		left.LinkURL != right.LinkURL ||
		left.LinkParams != right.LinkParams {
		return false
	}
	if left.Style == "" && left.ANSIStyle.IsZero() && left.LinkURL == "" && left.LinkParams == "" {
		return isASCIIWidthCell(left) && isASCIIWidthCell(right)
	}
	// 中文说明：带样式 UI cell 只压缩 live extent 占位 glyph；pane chrome 边角和 emoji 列锚仍保持原 cell 边界。
	return isRepeatedCanvasOutputExtentPlaceholder(left.Text, right.Text) &&
		isCanvasOutputMergeSafeDisplayCell(left) &&
		isCanvasOutputMergeSafeDisplayCell(right)
}

func isCanvasOutputMergeSafeDisplayCell(cell Cell) bool {
	if cell.Width < 0 || strings.Contains(cell.Text, "\ufe0f") {
		return false
	}
	if cell.Text == "" {
		return cell.Width == 0
	}
	return cell.Width == DisplayWidth(cell.Text)
}

func isRepeatedCanvasOutputExtentPlaceholder(text string, unit string) bool {
	glyph := contentViewportOutsideExtentGlyph()
	if glyph == "" || glyph == " " || unit != glyph {
		return false
	}
	return isRepeatedCanvasOutputText(text, glyph)
}

func isRepeatedCanvasOutputText(text string, unit string) bool {
	if text == "" || unit == "" || len(text)%len(unit) != 0 {
		return false
	}
	for start := 0; start < len(text); start += len(unit) {
		if text[start:start+len(unit)] != unit {
			return false
		}
	}
	return true
}

func isASCIIWidthCell(cell Cell) bool {
	if cell.Width != len(cell.Text) {
		return false
	}
	for i := 0; i < len(cell.Text); i++ {
		if cell.Text[i] < 0x20 || cell.Text[i] > 0x7e {
			return false
		}
	}
	return true
}

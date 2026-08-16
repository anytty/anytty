package vt

import uv "github.com/charmbracelet/ultraviolet"

type reflowLine struct {
	cells   uv.Line
	wrapped bool
}

// Reflow resizes the normal screen by preserving soft-wrapped logical lines.
func (s *Screen) Reflow(width, height, cursorX, cursorY int) (int, int) {
	if s == nil {
		return cursorX, cursorY
	}
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	oldWidth := s.Width()
	oldHeight := s.Height()
	if oldWidth < 1 {
		s.Resize(width, height)
		return clampInt(cursorX, 0, width-1), clampInt(cursorY, 0, height-1)
	}
	if oldWidth == width {
		s.Resize(width, height)
		return clampInt(cursorX, 0, width-1), clampInt(cursorY, 0, height-1)
	}

	sb := s.scrollback
	historyCount := 0
	if sb != nil {
		historyCount = sb.Len()
	}
	allRows := make([]reflowLine, 0, historyCount+oldHeight)
	if sb != nil {
		sb.compact()
		for i, row := range sb.Lines() {
			wrapped := sb.LineWrapped(i)
			allRows = append(allRows, reflowLine{
				cells:   cloneUVLineWithTrailingSpaces(row),
				wrapped: wrapped,
			})
		}
	}
	for y := 0; y < oldHeight; y++ {
		row := uv.Line(nil)
		row = s.Line(y)
		wrapped := s.LineWrapped(y)
		allRows = append(allRows, reflowLine{
			cells:   cloneUVLineWithTrailingSpaces(row),
			wrapped: wrapped,
		})
	}

	cursorAbs := historyCount + clampInt(cursorY, 0, max(0, oldHeight-1))
	topOffset := logicalOffsetForPosition(allRows, historyCount, 0)
	cursorOffset := logicalOffsetForPosition(allRows, cursorAbs, clampInt(cursorX, 0, max(0, oldWidth-1)))
	reflowed := reflowLines(allRows, width)
	newTopAbs, _ := positionForLogicalOffset(reflowed, topOffset)
	newAbs, newX := positionForLogicalOffset(reflowed, cursorOffset)

	visibleStart := newTopAbs
	if newAbs < visibleStart {
		visibleStart = newAbs
	}
	if newAbs >= visibleStart+height {
		visibleStart = newAbs - height + 1
	}
	maxVisibleStart := len(reflowed) - height
	if maxVisibleStart < 0 {
		maxVisibleStart = 0
	}
	visibleStart = clampInt(visibleStart, 0, maxVisibleStart)
	if sb != nil {
		sb.lines = make([]uv.Line, visibleStart)
		sb.wrapped = make([]bool, visibleStart)
		for i := 0; i < visibleStart; i++ {
			sb.lines[i] = cloneUVLineWithTrailingSpaces(reflowed[i].cells)
			sb.wrapped[i] = reflowed[i].wrapped
		}
		sb.offset = 0
		sb.size = len(sb.lines)
		if sb.size > sb.maxLines {
			sb.dropOldest(sb.size - sb.maxLines)
			sb.compact()
		}
		sb.normalizeWrapped()
	}

	cursor := s.cur
	s.buf = uv.NewRenderBuffer(width, height)
	s.wrapped = make([]bool, height)
	s.used = make([]int, height)
	for y := 0; y < height; y++ {
		src := visibleStart + y
		if src >= len(reflowed) {
			continue
		}
		writeLineToBuffer(s.buf, y, reflowed[src].cells)
		s.wrapped[y] = reflowed[src].wrapped
		s.used[y] = min(lineCellWidth(reflowed[src].cells), width)
	}
	s.scroll = s.buf.Bounds()
	s.cur = cursor
	s.buf.Touched = nil
	s.recordDamage(ScreenDamage{Width: width, Height: height})

	newY := newAbs - visibleStart
	if newY < 0 {
		newX = 0
		newY = 0
	}
	if newY >= height {
		newY = height - 1
	}
	newX = clampInt(newX, 0, width-1)
	return newX, newY
}

func reflowLines(rows []reflowLine, width int) []reflowLine {
	if width < 1 {
		width = 1
	}
	out := make([]reflowLine, 0, len(rows))
	for i := 0; i < len(rows); i++ {
		cells := append(uv.Line(nil), rows[i].cells...)
		for i < len(rows)-1 && rows[i].wrapped {
			i++
			cells = append(cells, rows[i].cells...)
		}
		out = append(out, splitLogicalLine(cells, width)...)
	}
	if len(out) == 0 {
		return []reflowLine{{}}
	}
	return out
}

func splitLogicalLine(cells uv.Line, width int) []reflowLine {
	if width < 1 {
		width = 1
	}
	if width == 1 {
		return splitLogicalLineByColumns(cells, width)
	}
	if len(cells) == 0 {
		return []reflowLine{{}}
	}
	out := make([]reflowLine, 0, (lineCellWidth(cells)+width-1)/width)
	var current uv.Line
	currentWidth := 0
	flush := func(wrapped bool) {
		out = append(out, reflowLine{cells: cloneUVLineWithTrailingSpaces(current), wrapped: wrapped})
		current = nil
		currentWidth = 0
	}
	for _, cell := range cells {
		cellWidth := visibleCellWidth(cell)
		if cellWidth <= 0 {
			continue
		}
		if currentWidth > 0 && currentWidth+cellWidth > width {
			flush(true)
		}
		if cellWidth > width {
			if currentWidth > 0 {
				flush(true)
			}
			out = append(out, reflowLine{cells: uv.Line{cell}, wrapped: false})
			continue
		}
		current = append(current, cell)
		currentWidth += cellWidth
		if currentWidth == width {
			flush(true)
		}
	}
	if current != nil || len(out) == 0 {
		flush(false)
	} else {
		out[len(out)-1].wrapped = false
	}
	return out
}

func splitLogicalLineByColumns(cells uv.Line, width int) []reflowLine {
	if width < 1 {
		width = 1
	}
	if len(cells) == 0 {
		return []reflowLine{{}}
	}
	out := make([]reflowLine, 0, len(cells))
	for _, cell := range cells {
		if visibleCellWidth(cell) <= 0 {
			continue
		}
		out = append(out, reflowLine{cells: uv.Line{cell}, wrapped: true})
	}
	if len(out) == 0 {
		return []reflowLine{{}}
	}
	out[len(out)-1].wrapped = false
	return out
}

func logicalOffsetForPosition(rows []reflowLine, rowIndex, col int) int {
	offset := 0
	for i := 0; i < rowIndex && i < len(rows); i++ {
		offset += lineCellWidth(rows[i].cells)
		if !rows[i].wrapped {
			offset++
		}
	}
	return offset + min(col, lineCellWidth(rowAtReflow(rows, rowIndex).cells))
}

func positionForLogicalOffset(rows []reflowLine, offset int) (int, int) {
	if offset <= 0 {
		return 0, 0
	}
	remaining := offset
	for i, row := range rows {
		width := lineCellWidth(row.cells)
		if remaining <= width {
			return i, remaining
		}
		remaining -= width
		if !row.wrapped {
			if remaining == 0 {
				return i, min(width, max(0, width-1))
			}
			remaining--
		}
	}
	last := max(0, len(rows)-1)
	return last, min(lineCellWidth(rowAtReflow(rows, last).cells), max(0, lineCellWidth(rowAtReflow(rows, last).cells)-1))
}

func rowAtReflow(rows []reflowLine, index int) reflowLine {
	if index < 0 || index >= len(rows) {
		return reflowLine{}
	}
	return rows[index]
}

func writeLineToBuffer(buf *uv.RenderBuffer, y int, row uv.Line) {
	if buf == nil || y < 0 || y >= buf.Height() {
		return
	}
	x := 0
	for _, cell := range row {
		if x >= buf.Width() {
			return
		}
		cellWidth := visibleCellWidth(cell)
		if cellWidth <= 0 {
			continue
		}
		buf.Buffer.SetCell(x, y, &cell)
		x += cellWidth
	}
}

func cloneUVLineWithTrailingSpaces(line uv.Line) uv.Line {
	if len(line) == 0 {
		return nil
	}
	return append(uv.Line(nil), line...)
}

func lineCellWidth(line uv.Line) int {
	width := 0
	for _, cell := range line {
		width += visibleCellWidth(cell)
	}
	return width
}

func visibleCellWidth(cell uv.Cell) int {
	if cell.IsZero() {
		return 0
	}
	if cell.Width <= 0 {
		return 0
	}
	return cell.Width
}

func clampInt(value, low, high int) int {
	if high < low {
		return low
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

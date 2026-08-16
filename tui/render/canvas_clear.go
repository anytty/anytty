package render

import "strings"

func (c *canvas) clearCellColumnInFootprint(y int, start int, target int) {
	cell := c.rows[y][start]
	width := maxInt(1, cell.width)
	end := minInt(c.width, start+width)
	if target <= start && target+1 >= end {
		c.clearCellFootprint(y, start)
		return
	}
	leftWidth := target - start
	rightWidth := end - target - 1
	left := canvasSegmentFromCellPart(cell, 0, leftWidth)
	right := canvasSegmentFromCellPart(cell, width-rightWidth, rightWidth)
	for col := start; col < end; col++ {
		c.rows[y][col] = blankCanvasCell()
	}
	// 中文说明：局部覆盖一个合并的 ASCII run 时，只清目标列，保留未被覆盖的左右文本。
	if left.width > 0 {
		c.writeSegmentNoClear(start, y, left)
	}
	if right.width > 0 {
		c.writeSegmentNoClear(end-rightWidth, y, right)
	}
}

func canvasSegmentFromCellPart(cell canvasCell, offset int, width int) canvasSegment {
	if width <= 0 {
		return canvasSegment{}
	}
	text := cell.text
	if text != "" && cell.width == len(text) && offset >= 0 && offset+width <= len(text) {
		text = text[offset : offset+width]
	} else if text != "" && isRepeatedCanvasOutputExtentPlaceholder(text, contentViewportOutsideExtentGlyph()) && offset >= 0 && offset+width <= cell.width {
		// 中文说明：R467 后 live extent 占位点可作为非 ASCII 宽 segment 存在；局部覆盖时左右段必须保留 glyph，不能退化成空格。
		text = strings.Repeat(contentViewportOutsideExtentGlyph(), width)
	} else if text != "" && offset == 0 && width == cell.width {
		text = cell.text
	} else {
		text = strings.Repeat(" ", width)
	}
	return canvasSegment{
		text:       text,
		width:      width,
		style:      cell.style,
		ansiStyle:  cell.ansiStyle,
		linkURL:    cell.linkURL,
		linkParams: cell.linkParams,
		owner:      cell.owner,
		layer:      cell.layer,
		terminal:   cell.terminal,
		safe:       cell.safe,
	}
}

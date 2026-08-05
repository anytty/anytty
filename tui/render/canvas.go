package render

import (
	xansi "github.com/charmbracelet/x/ansi"
	"strings"
)

type canvas struct {
	width  int
	height int
	rows   [][]canvasCell
}

type canvasCell struct {
	text         string
	width        int
	style        StyleToken
	ansiStyle    ANSICellStyle
	linkURL      string
	linkParams   string
	owner        string
	layer        LayerKind
	continuation bool
	terminal     bool
	safe         bool
}

func newCanvas(width int, height int) *canvas {
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}
	rows := make([][]canvasCell, height)
	for i := range rows {
		rows[i] = make([]canvasCell, width)
	}
	return &canvas{width: width, height: height, rows: rows}
}

func (c *canvas) writeText(x int, y int, width int, text string) {
	c.writeTextStyled(x, y, width, text, "", "", LayerBase)
}

func (c *canvas) writeTextStyled(x int, y int, width int, text string, style StyleToken, owner string, layer LayerKind) {
	fitted := FitText(text, width)
	c.writeLine(x, y, width, Line{Cells: []Cell{{Text: fitted, Width: DisplayWidth(fitted), Style: style, Safe: true}}}, owner, layer)
}

func (c *canvas) overlayTextStyled(x int, y int, maxWidth int, text string, style StyleToken, owner string, layer LayerKind) {
	if y < 0 || y >= c.height || maxWidth <= 0 || x >= c.width {
		return
	}
	if x < 0 {
		text = SliceCells(text, -x, DisplayWidth(text))
		maxWidth += x
		x = 0
	}
	maxWidth = minInt(maxWidth, c.width-x)
	text = TruncateCells(text, maxWidth)
	if text == "" {
		return
	}
	cursor := x
	for _, segment := range cellSegments(text, style, owner, layer) {
		if cursor+segment.width > x+maxWidth {
			break
		}
		c.writeSegment(cursor, y, segment)
		cursor += segment.width
	}
}

func (c *canvas) writeLine(x int, y int, width int, line Line, owner string, layer LayerKind) {
	if y < 0 || y >= c.height || width <= 0 || x >= c.width {
		return
	}
	if x < 0 {
		width += x
		x = 0
	}
	width = minInt(width, c.width-x)
	if width <= 0 {
		return
	}
	c.clearCellRange(y, x, width)
	cursor := x
	remaining := width
	fillStyle := lineFillStyle(line)
	for _, cell := range line.Cells {
		if remaining <= 0 {
			break
		}
		text := safeCanvasCellText(cell)
		cellWidth := maxInt(0, cell.Width)
		if text == "" {
			if cellWidth > 0 {
				if cellWidth > remaining {
					break
				}
				// 中文说明：真实 terminal cell 可以只有 footprint 没有文本；它的背景仍必须写进 canvas。
				c.writeSegmentNoClear(cursor, y, canvasSegment{
					text:       strings.Repeat(" ", cellWidth),
					width:      cellWidth,
					style:      cell.Style,
					ansiStyle:  cell.ANSIStyle,
					linkURL:    cell.LinkURL,
					linkParams: cell.LinkParams,
					owner:      owner,
					layer:      layer,
					terminal:   cell.TerminalContent,
					safe:       cell.Safe,
				})
				cursor += cellWidth
				remaining -= cellWidth
			}
			continue
		}
		if cellWidth > 0 && cellWidth != DisplayWidth(text) {
			if cellWidth > remaining {
				break
			}
			// 中文说明：protocol/live cell 的 Width 是真实 terminal footprint；宽度与本地测量不一致时不能再拆分重算。
			c.writeSegmentNoClear(cursor, y, canvasSegment{
				text:       text,
				width:      cellWidth,
				style:      cell.Style,
				ansiStyle:  cell.ANSIStyle,
				linkURL:    cell.LinkURL,
				linkParams: cell.LinkParams,
				owner:      owner,
				layer:      layer,
				terminal:   cell.TerminalContent,
				safe:       cell.Safe,
			})
			cursor += cellWidth
			remaining -= cellWidth
			continue
		}
		if canWriteCanvasCellAsSingleSegment(text, cell, cellWidth) {
			if cellWidth > remaining {
				break
			}
			// 中文说明：live surface 已经把相同样式 ASCII run 合并；这里保留整段，
			// 避免再拆成逐列 terminal cell 后在 ANSI 输出里反复写模型列锚点。
			c.writeSegmentNoClear(cursor, y, canvasSegment{
				text:       text,
				width:      cellWidth,
				style:      cell.Style,
				ansiStyle:  cell.ANSIStyle,
				linkURL:    cell.LinkURL,
				linkParams: cell.LinkParams,
				owner:      owner,
				layer:      layer,
				terminal:   cell.TerminalContent,
				safe:       cell.Safe,
			})
			cursor += cellWidth
			remaining -= cellWidth
			continue
		}
		for len(text) > 0 && remaining > 0 {
			cluster, clusterWidth := xansi.FirstGraphemeCluster(text, xansi.GraphemeWidth)
			if cluster == "" {
				break
			}
			text = text[len(cluster):]
			if clusterWidth <= 0 {
				continue
			}
			if clusterWidth > remaining {
				break
			}
			c.writeSegmentNoClear(cursor, y, canvasSegment{
				text:       cluster,
				width:      clusterWidth,
				style:      cell.Style,
				ansiStyle:  cell.ANSIStyle,
				linkURL:    cell.LinkURL,
				linkParams: cell.LinkParams,
				owner:      owner,
				layer:      layer,
				terminal:   cell.TerminalContent,
				safe:       cell.Safe,
			})
			cursor += clusterWidth
			remaining -= clusterWidth
		}
	}
	for remaining > 0 {
		c.writeSegmentNoClear(cursor, y, canvasSegment{
			text:  " ",
			width: 1,
			style: fillStyle,
			owner: owner,
			layer: layer,
			safe:  true,
		})
		cursor++
		remaining--
	}
}

func safeCanvasCellText(cell Cell) string {
	if cell.Safe && !canvasCellTextNeedsSanitize(cell.Text) {
		return cell.Text
	}
	return xansi.Strip(SafeLine(cell.Text))
}

func canvasCellTextNeedsSanitize(text string) bool {
	for i := 0; i < len(text); i++ {
		b := text[i]
		if b < 0x20 || b == 0x7f {
			return true
		}
	}
	return false
}

func canWriteCanvasCellAsSingleSegment(text string, cell Cell, cellWidth int) bool {
	if text == "" || cellWidth <= 1 {
		return false
	}
	if canWriteCanvasExtentPlaceholderAsSingleSegment(text, cell, cellWidth) {
		return true
	}
	if cellWidth != len(text) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if text[i] < 0x20 || text[i] > 0x7e {
			return false
		}
	}
	return true
}

func (c *canvas) writeStyledBoxCell(x int, y int, glyph string, style StyleToken, owner string, layer LayerKind) {
	if DisplayWidth(glyph) != 1 {
		c.writeTextStyled(x, y, 1, glyph, style, owner, layer)
		return
	}
	c.writeSegment(x, y, canvasSegment{
		text:  glyph,
		width: 1,
		style: style,
		owner: owner,
		layer: layer,
		safe:  true,
	})
}

func (c *canvas) mergeStyledBoxCell(x int, y int, connections uint8, style StyleToken, owner string, layer LayerKind) {
	if x < 0 || y < 0 || x >= c.width || y >= c.height {
		return
	}
	existingCell := c.rows[y][x]
	existing := c.cellText(x, y)
	if existingConnections, ok := boxConnectionsForGlyph(existing); ok {
		connections = mergeBoxCellConnections(existingConnections, connections, existingCell.style, style)
		style = mergeBoxCellStyle(existingCell.style, style)
	}
	glyph, ok := boxGlyphForConnections(connections)
	if !ok {
		return
	}
	c.writeStyledBoxCell(x, y, glyph, style, owner, layer)
}

func (c *canvas) fillStyledRect(rect Rect, style StyleToken, owner string, layer LayerKind) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	for y := rect.Y; y < rect.Y+rect.H; y++ {
		c.writeTextStyled(rect.X, y, rect.W, "", style, owner, layer)
	}
}

func (c *canvas) fillRect(rect Rect, owner string, layer LayerKind) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	for y := rect.Y; y < rect.Y+rect.H; y++ {
		c.writeLine(rect.X, y, rect.W, Line{}, owner, layer)
	}
}

func (c *canvas) drawStyledBox(rect Rect, style boxStyle, token StyleToken, owner string, layer LayerKind) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	if rect.W == 1 {
		for y := 0; y < rect.H; y++ {
			c.writeStyledBoxCell(rect.X, rect.Y+y, style.Vertical, token, owner, layer)
		}
		return
	}
	if rect.H == 1 {
		c.writeTextStyled(rect.X, rect.Y, rect.W, style.TopLeft+strings.Repeat(style.Horizontal, maxInt(0, rect.W-2))+style.TopRight, token, owner, layer)
		return
	}
	c.writeTextStyled(rect.X, rect.Y, rect.W, style.TopLeft+strings.Repeat(style.Horizontal, maxInt(0, rect.W-2))+style.TopRight, token, owner, layer)
	c.writeTextStyled(rect.X, rect.Y+rect.H-1, rect.W, style.BottomLeft+strings.Repeat(style.Horizontal, maxInt(0, rect.W-2))+style.BottomRight, token, owner, layer)
	for y := rect.Y + 1; y < rect.Y+rect.H-1; y++ {
		c.writeStyledBoxCell(rect.X, y, style.Vertical, token, owner, layer)
		c.writeStyledBoxCell(rect.X+rect.W-1, y, style.Vertical, token, owner, layer)
	}
}

func (c *canvas) drawStyledPaneFrame(rect Rect, style StyleToken, owner string, layer LayerKind) {
	if rect.W < 2 || rect.H < 2 {
		return
	}
	// 参考 tuiv2 的 pane frame 语义：按连接位逐格绘制边框，再让标题槽位覆盖顶边内部。
	c.drawStyledPaneHBorder(rect.X, rect.X+rect.W-1, rect.Y, style, owner, layer, true)
	c.drawStyledPaneHBorder(rect.X, rect.X+rect.W-1, rect.Y+rect.H-1, style, owner, layer, false)
	c.drawStyledPaneVBorder(rect.X, rect.Y+1, rect.Y+rect.H-2, style, owner, layer)
	c.drawStyledPaneVBorder(rect.X+rect.W-1, rect.Y+1, rect.Y+rect.H-2, style, owner, layer)
}

func (c *canvas) drawStyledPaneHBorder(startX int, endX int, y int, style StyleToken, owner string, layer LayerKind, top bool) {
	if startX > endX {
		return
	}
	for x := startX; x <= endX; x++ {
		connections := uint8(boxConnLeft | boxConnRight)
		if x == startX {
			connections = boxConnRight
			if top {
				connections |= boxConnDown
			} else {
				connections |= boxConnUp
			}
		} else if x == endX {
			connections = boxConnLeft
			if top {
				connections |= boxConnDown
			} else {
				connections |= boxConnUp
			}
		}
		c.mergeStyledBoxCell(x, y, connections, style, owner, layer)
	}
}

func (c *canvas) drawStyledPaneVBorder(x int, startY int, endY int, style StyleToken, owner string, layer LayerKind) {
	if startY > endY {
		return
	}
	for y := startY; y <= endY; y++ {
		c.mergeStyledBoxCell(x, y, boxConnUp|boxConnDown, style, owner, layer)
	}
}

func (c *canvas) clearCellRange(y int, x int, width int) {
	for col := x; col < x+width && col < c.width; col++ {
		c.clearCell(y, col)
	}
}

func (c *canvas) clearCell(y int, x int) {
	if y < 0 || y >= c.height || x < 0 || x >= c.width {
		return
	}
	if c.rows[y][x].continuation {
		for col := x - 1; col >= 0; col-- {
			if !c.rows[y][col].continuation {
				c.clearCellColumnInFootprint(y, col, x)
				break
			}
		}
		return
	}
	c.clearCellColumnInFootprint(y, x, x)
}

func (c *canvas) clearCellFootprint(y int, x int) {
	cell := c.rows[y][x]
	width := maxInt(1, cell.width)
	for col := x; col < x+width && col < c.width; col++ {
		c.rows[y][col] = blankCanvasCell()
	}
}

func (c *canvas) writeSegment(x int, y int, segment canvasSegment) {
	if y < 0 || y >= c.height || x < 0 || x >= c.width || segment.width <= 0 {
		return
	}
	if x+segment.width > c.width {
		return
	}
	c.clearCellRange(y, x, segment.width)
	c.writeSegmentNoClear(x, y, segment)
}

func (c *canvas) writeSegmentNoClear(x int, y int, segment canvasSegment) {
	if y < 0 || y >= c.height || x < 0 || x >= c.width || segment.width <= 0 {
		return
	}
	if x+segment.width > c.width {
		return
	}
	c.rows[y][x] = canvasCell{
		text:       segment.text,
		width:      segment.width,
		style:      segment.style,
		ansiStyle:  segment.ansiStyle,
		linkURL:    segment.linkURL,
		linkParams: segment.linkParams,
		owner:      segment.owner,
		layer:      segment.layer,
		terminal:   segment.terminal,
		safe:       segment.safe,
	}
	for col := x + 1; col < x+segment.width; col++ {
		c.rows[y][col] = canvasCell{
			width:        0,
			style:        segment.style,
			ansiStyle:    segment.ansiStyle,
			linkURL:      segment.linkURL,
			linkParams:   segment.linkParams,
			owner:        segment.owner,
			layer:        segment.layer,
			continuation: true,
			terminal:     segment.terminal,
			safe:         segment.safe,
		}
	}
}

func (c *canvas) cellText(x int, y int) string {
	if y < 0 || y >= c.height || x < 0 || x >= c.width {
		return ""
	}
	if c.rows[y][x].continuation {
		for col := x - 1; col >= 0; col-- {
			if !c.rows[y][col].continuation {
				return c.rows[y][col].text
			}
		}
		return ""
	}
	return c.rows[y][x].text
}

func lineFillStyle(line Line) StyleToken {
	return line.FillStyle
}

func blankCanvasCell() canvasCell {
	return canvasCell{}
}

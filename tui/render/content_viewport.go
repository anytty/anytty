package render

import "strings"

// RenderContentViewport 是所有 pane/floating 内容投影的基础合同：
// terminal live 的真实 extent 外区域保留可见边界提示；遮挡只通过 overflow hint 交给 chrome。
func RenderContentViewport(request ContentRenderRequest) ContentRenderResult {
	rect := request.Rect
	if rect.W <= 0 || rect.H <= 0 {
		return ContentRenderResult{}
	}
	content := request.Content
	if content.Kind == ContentEmptyPane {
		return renderCenteredActionContentViewport(request)
	}
	if content.Kind == ContentExitedPane {
		return renderExitedContentViewport(request)
	}
	lines := content.Lines
	if len(lines) == 0 {
		lines = []Line{NewLine(content.Status)}
	}
	extent := normalizeContentExtent(content.Extent, rect)
	extent = applyContentLayoutToExtent(content.Layout, extent, rect)
	overflow := contentViewportOverflow(lines, extent, rect)
	markOutsideExtent := contentViewportMarksOutsideExtent(content)
	if contentViewportCanUseDirectRows(extent, rect) {
		return ContentRenderResult{
			Lines:      contentViewportDirectRows(lines, rect.W, rect.H),
			Cursor:     contentViewportCursor(content.Cursor, extent, rect),
			HitRegions: contentViewportHitRegions(content.HitRegions, extent, rect),
			Metadata:   RenderMetadata{Width: rect.W, Height: rect.H},
			Overflow:   overflow,
		}
	}
	rendered := make([]Line, 0, rect.H)
	for row := 0; row < rect.H; row++ {
		rendered = append(rendered, renderContentViewportRow(lines, extent, rect.W, row, markOutsideExtent))
	}
	return ContentRenderResult{
		Lines:      rendered,
		Cursor:     contentViewportCursor(content.Cursor, extent, rect),
		HitRegions: contentViewportHitRegions(content.HitRegions, extent, rect),
		Metadata:   RenderMetadata{Width: rect.W, Height: rect.H},
		Overflow:   overflow,
	}
}

func contentViewportHitRegions(regions []HitRegion, extent ContentExtent, rect Rect) []HitRegion {
	if len(regions) == 0 {
		return nil
	}
	out := make([]HitRegion, 0, len(regions))
	for _, region := range regions {
		region.Rect.X += extent.X
		region.Rect.Y += extent.Y
		left := maxInt(maxInt(0, extent.X), region.Rect.X)
		top := maxInt(maxInt(0, extent.Y), region.Rect.Y)
		right := minInt(minInt(rect.W, extent.X+extent.Cols), region.Rect.X+region.Rect.W)
		bottom := minInt(minInt(rect.H, extent.Y+extent.Rows), region.Rect.Y+region.Rect.H)
		if right <= left || bottom <= top {
			continue
		}
		region.Rect = Rect{X: left, Y: top, W: right - left, H: bottom - top}
		out = append(out, region)
	}
	return out
}

func contentViewportCursor(cursor Cursor, extent ContentExtent, rect Rect) Cursor {
	if !cursor.Visible && !cursor.Anchor {
		return cursor
	}
	cursor.Row += extent.Y
	cursor.Col += extent.X
	if cursor.Row < 0 || cursor.Row >= rect.H || cursor.Col < 0 || cursor.Col >= rect.W {
		cursor.Visible = false
	}
	return cursor
}

func projectContentCursor(content ContentVM, rect Rect) Cursor {
	extent := normalizeContentExtent(content.Extent, rect)
	extent = applyContentLayoutToExtent(content.Layout, extent, rect)
	return contentViewportCursor(content.Cursor, extent, rect)
}

func applyContentLayoutToExtent(layout ContentLayoutVM, extent ContentExtent, rect Rect) ContentExtent {
	if !extent.Known || !layout.Known {
		return extent
	}
	if layout.Mode == "fit" {
		extent.X = 0
		extent.Y = 0
		extent.Cols = maxInt(0, rect.W)
		extent.Rows = maxInt(0, rect.H)
	} else {
		alignX := layout.AlignX
		alignY := layout.AlignY
		if layout.Mode == "center" {
			alignX = "center"
			alignY = "center"
		}
		extent.X = alignedContentOrigin(alignX, rect.W, extent.Cols)
		extent.Y = alignedContentOrigin(alignY, rect.H, extent.Rows)
	}
	extent.X -= layout.PanX
	extent.Y -= layout.PanY
	return extent
}

func alignedContentOrigin(align string, viewport int, size int) int {
	switch align {
	case "center", "base":
		return centeredContentOrigin(viewport - size)
	case "end":
		return viewport - size
	default:
		return 0
	}
}

func centeredContentOrigin(delta int) int {
	if delta >= 0 {
		return (delta + 1) / 2
	}
	return -((-delta + 1) / 2)
}

func contentViewportMarksOutsideExtent(content ContentVM) bool {
	// Live 与 frozen history 共享同一 terminal extent；两种模式外侧都保留 pane/terminal 尺寸差异提示。
	return (content.Kind == ContentTerminalLive || content.Kind == ContentCopyHistory) &&
		content.Extent.Known && !content.Pending && content.Error == ""
}

func contentViewportCanUseDirectRows(extent ContentExtent, rect Rect) bool {
	// 中文说明：extent 正好覆盖内容区时没有外部区域需要标记，live surface 也可以走直投快路径。
	return extent.Known &&
		extent.X == 0 &&
		extent.Y == 0 &&
		extent.Cols == rect.W &&
		extent.Rows == rect.H
}

func contentViewportDirectRows(lines []Line, width int, height int) []Line {
	rendered := make([]Line, height)
	for row := 0; row < height; row++ {
		if row >= len(lines) {
			rendered[row] = Line{Cells: []Cell{contentViewportBlankRun(width)}}
			continue
		}
		rendered[row] = contentViewportFitLine(lines[row], width)
	}
	return rendered
}

func contentViewportFitLine(line Line, width int) Line {
	if width <= 0 {
		return Line{}
	}
	lineWidth := line.Width()
	if lineWidth == width {
		return line
	}
	if lineWidth > width {
		return contentViewportLineWindow(line, 0, width)
	}
	cells := make([]Cell, 0, len(line.Cells)+1)
	cells = append(cells, line.Cells...)
	cells = append(cells, contentViewportBlankRun(width-lineWidth))
	return Line{Cells: cells}
}

func renderCenteredActionContentViewport(request ContentRenderRequest) ContentRenderResult {
	rect := request.Rect
	content := request.Content
	lines := content.Lines
	if len(lines) == 0 {
		lines = []Line{NewLine(content.Status)}
	}
	rendered := make([]Line, rect.H)
	for row := 0; row < rect.H; row++ {
		rendered[row] = NewLine(strings.Repeat(" ", rect.W))
	}
	firstLine, startY := centeredActionContentWindow(content.Kind, len(lines), rect.H)
	translatedRegions := make([]HitRegion, 0, len(content.HitRegions))
	for index := firstLine; index < len(lines); index++ {
		line := lines[index]
		y := startY + index - firstLine
		if y < 0 || y >= rect.H {
			continue
		}
		centered := centerContentLine(line, rect.W)
		rendered[y] = centered
	}
	for _, region := range content.HitRegions {
		if region.Rect.Y < firstLine || region.Rect.Y >= len(lines) {
			continue
		}
		line := lines[region.Rect.Y]
		region.Rect.X = centeredLineTextX(line, rect.W)
		region.Rect.Y = startY + region.Rect.Y - firstLine
		region.Rect.W = minInt(line.Width(), rect.W)
		region.Rect.H = 1
		translatedRegions = append(translatedRegions, region)
	}
	return ContentRenderResult{Lines: rendered, Cursor: Cursor{}, HitRegions: translatedRegions, Metadata: RenderMetadata{Width: rect.W, Height: rect.H}}
}

func renderExitedContentViewport(request ContentRenderRequest) ContentRenderResult {
	rect := request.Rect
	content := request.Content
	lines := content.Lines
	if len(lines) == 0 {
		lines = []Line{NewLine(content.Status)}
	}
	rendered := make([]Line, rect.H)
	for row := 0; row < rect.H; row++ {
		rendered[row] = NewLine(strings.Repeat(" ", rect.W))
	}
	firstLine, startY := exitedContentWindow(len(lines), rect.H)
	actionRows := exitedContentActionRows(content.HitRegions)
	for index := firstLine; index < len(lines); index++ {
		y := startY + index - firstLine
		if y < 0 || y >= rect.H {
			continue
		}
		line := lines[index]
		if actionRows[index] {
			rendered[y] = centerContentLine(line, rect.W)
			continue
		}
		rendered[y] = contentViewportFitLine(line, rect.W)
	}
	return ContentRenderResult{
		Lines:      rendered,
		Cursor:     Cursor{},
		HitRegions: exitedContentHitRegions(content, rect.W, rect.H),
		Metadata:   RenderMetadata{Width: rect.W, Height: rect.H},
	}
}

func exitedContentHitRegions(content ContentVM, width int, height int) []HitRegion {
	lines := content.Lines
	if len(lines) == 0 {
		lines = []Line{NewLine(content.Status)}
	}
	firstLine, startY := exitedContentWindow(len(lines), height)
	regions := make([]HitRegion, 0, len(content.HitRegions))
	for _, region := range content.HitRegions {
		if region.Rect.Y < firstLine || region.Rect.Y >= len(lines) {
			continue
		}
		line := lines[region.Rect.Y]
		region.Rect.X = centeredLineTextX(line, width)
		region.Rect.Y = startY + region.Rect.Y - firstLine
		region.Rect.W = minInt(line.Width(), width)
		region.Rect.H = 1
		regions = append(regions, region)
	}
	return regions
}

func exitedContentActionRows(regions []HitRegion) map[int]bool {
	rows := make(map[int]bool, len(regions))
	for _, region := range regions {
		if region.Kind == HitRegionContentAction && region.ActionID != "" {
			rows[region.Rect.Y] = true
		}
	}
	return rows
}

func centeredActionContentKind(kind ContentKind) bool {
	return kind == ContentEmptyPane
}

func centeredActionContentWindow(kind ContentKind, lineCount int, height int) (int, int) {
	if height <= 0 || lineCount <= 0 {
		return 0, 0
	}
	if height >= lineCount+2 {
		return 0, 1
	}
	return 0, 0
}

func exitedContentWindow(lineCount int, height int) (int, int) {
	if height <= 0 || lineCount <= 0 {
		return 0, 0
	}
	if lineCount > height {
		// 中文说明：退出信息属于 live tail 后续内容；溢出时只裁旧尾部，不横向改写 terminal 文本。
		return lineCount - height, 0
	}
	if height >= lineCount+2 {
		return 0, 1
	}
	return 0, 0
}

func centerContentLine(line Line, width int) Line {
	if width <= 0 {
		return Line{}
	}
	lineWidth := minInt(line.Width(), width)
	left := maxInt(0, (width-lineWidth)/2)
	right := maxInt(0, width-left-lineWidth)
	cells := make([]Cell, 0, len(line.Cells)+2)
	if left > 0 {
		cells = append(cells, NewCell(strings.Repeat(" ", left)))
	}
	cells = append(cells, contentViewportLineWindow(line, 0, lineWidth).Cells...)
	if right > 0 {
		cells = append(cells, NewCell(strings.Repeat(" ", right)))
	}
	return Line{Cells: cells}
}

func centeredLineTextX(line Line, width int) int {
	if width <= 0 {
		return 0
	}
	return maxInt(0, (width-minInt(line.Width(), width))/2)
}

func normalizeContentExtent(extent ContentExtent, rect Rect) ContentExtent {
	if !extent.Known {
		return ContentExtent{Known: true, Cols: maxInt(0, rect.W), Rows: maxInt(0, rect.H)}
	}
	if extent.Cols < 0 {
		extent.Cols = 0
	}
	if extent.Rows < 0 {
		extent.Rows = 0
	}
	return extent
}

func contentViewportOverflow(lines []Line, extent ContentExtent, rect Rect) ContentOverflow {
	overflow := ContentOverflow{
		Left:   extent.X < 0,
		Right:  extent.X+extent.Cols > rect.W,
		Top:    extent.Y < 0,
		Bottom: extent.Y+extent.Rows > rect.H,
	}
	if len(lines) > extent.Rows {
		overflow.Bottom = true
	}
	for _, line := range lines {
		if line.Width() > extent.Cols {
			overflow.Right = true
			break
		}
	}
	return overflow
}

func renderContentViewportRow(lines []Line, extent ContentExtent, width int, row int, markOutsideExtent bool) Line {
	if width <= 0 {
		return Line{}
	}
	sourceRow := row - extent.Y
	if sourceRow < 0 || sourceRow >= extent.Rows {
		return contentViewportOutsideExtentLine(width, markOutsideExtent)
	}
	line := Line{}
	if sourceRow < len(lines) {
		line = lines[sourceRow]
	}
	insideStart := maxInt(0, extent.X)
	insideEnd := minInt(width, extent.X+extent.Cols)
	if insideStart >= insideEnd {
		return contentViewportOutsideExtentLine(width, markOutsideExtent)
	}
	if insideStart == 0 && insideEnd == width {
		start := -extent.X
		if start == 0 {
			return contentViewportFitLine(line, width)
		}
		return contentViewportLineWindow(line, start, width)
	}

	cells := make([]Cell, 0, minInt(width, len(line.Cells)+2))
	if insideStart > 0 {
		cells = append(cells, contentViewportOutsideExtentRun(insideStart, markOutsideExtent))
	}
	visible := contentViewportLineWindow(line, insideStart-extent.X, insideEnd-insideStart)
	cells = append(cells, visible.Cells...)
	if insideEnd < width {
		cells = append(cells, contentViewportOutsideExtentRun(width-insideEnd, markOutsideExtent))
	}
	return Line{Cells: cells}
}

func contentViewportLineWindow(line Line, start int, width int) Line {
	if width <= 0 {
		return Line{}
	}
	if start < 0 {
		start = 0
	}
	end := start + width
	sourceCol := 0
	outWidth := 0
	cells := make([]Cell, 0, minInt(width, len(line.Cells)+1))
	for _, cell := range line.Cells {
		if outWidth >= width {
			break
		}
		cellWidth := maxInt(0, cell.Width)
		if cellWidth == 0 {
			continue
		}
		cellStart := sourceCol
		cellEnd := sourceCol + cellWidth
		sourceCol = cellEnd
		if cellEnd <= start {
			continue
		}
		if cellStart >= end {
			break
		}
		visibleStart := maxInt(cellStart, start)
		visibleEnd := minInt(cellEnd, end)
		visibleWidth := maxInt(0, visibleEnd-visibleStart)
		if visibleWidth == 0 {
			continue
		}
		if cellStart >= start && cellEnd <= end && cellWidth <= width-outWidth {
			cells = append(cells, cell)
			outWidth += cellWidth
			continue
		}
		if cell.Text != "" {
			text := SliceCells(cell.Text, visibleStart-cellStart, visibleEnd-cellStart)
			if text != "" {
				visibleCell := cell
				visibleCell.Text = text
				visibleCell.Width = DisplayWidth(text)
				if visibleCell.Width > 0 && visibleCell.Width <= width-outWidth {
					cells = append(cells, visibleCell)
					outWidth += visibleCell.Width
					continue
				}
			}
		}
		blankWidth := minInt(width, outWidth+visibleWidth) - outWidth
		if blankWidth > 0 {
			cells = append(cells, contentViewportStyledBlankRun(blankWidth, cell))
			outWidth += blankWidth
		}
	}
	if outWidth < width {
		cells = append(cells, contentViewportBlankRun(width-outWidth))
	}
	return Line{Cells: cells}
}

func contentViewportBlankRun(width int) Cell {
	if width <= 0 {
		return Cell{}
	}
	return Cell{Text: strings.Repeat(" ", width), Width: width, Safe: true}
}

func contentViewportStyledBlankRun(width int, source Cell) Cell {
	cell := contentViewportBlankRun(width)
	// 中文说明：裁剪落在 terminal cell 的声明宽度尾部时，空白列仍属于该 cell 的视觉区域。
	cell.Style = source.Style
	cell.ANSIStyle = source.ANSIStyle
	cell.TerminalContent = source.TerminalContent
	return cell
}

func contentViewportBlankCell() Cell {
	return Cell{Text: " ", Width: 1, Safe: true}
}

func contentViewportOutsideExtentCell(markOutsideExtent bool) Cell {
	if markOutsideExtent {
		return Cell{Text: contentViewportOutsideExtentGlyph(), Width: 1, Style: paneChromeExtentPlaceholderStyle(), Safe: true}
	}
	return contentViewportBlankCell()
}

func contentViewportOutsideExtentRun(width int, markOutsideExtent bool) Cell {
	if width <= 1 {
		return contentViewportOutsideExtentCell(markOutsideExtent)
	}
	if !markOutsideExtent {
		return contentViewportBlankRun(width)
	}
	glyph := contentViewportOutsideExtentGlyph()
	if glyph == " " {
		return contentViewportBlankRun(width)
	}
	return Cell{Text: strings.Repeat(glyph, width), Width: width, Style: paneChromeExtentPlaceholderStyle(), Safe: true}
}

func contentViewportOutsideExtentGlyph() string {
	glyph := paneChromeExtentPlaceholderGlyph()
	if glyph == "" {
		return " "
	}
	if DisplayWidth(glyph) != 1 {
		return "·"
	}
	return glyph
}

func contentViewportOutsideExtentLine(width int, markOutsideExtent bool) Line {
	if width <= 0 {
		return Line{}
	}
	return Line{Cells: []Cell{contentViewportOutsideExtentRun(width, markOutsideExtent)}}
}

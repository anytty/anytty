package render

func renderWorkbenchNavigatorOverlay(c *canvas, overlay OverlayVM, rect Rect, contentRect Rect) Layer {
	primitive := OverlayChromePrimitive(overlay, rect, contentRect)
	owner := primitive.Owner
	layer := primitive.Layer
	c.fillStyledRect(rect, StyleForeground, owner, layer)
	c.drawStyledBox(rect, squareBoxStyle, StyleForeground, owner, layer)
	renderChromeCardTitle(c, rect, primitive.Title.Text, "", "", StyleAccent, owner, layer)
	contentResult := renderContent(c, overlay.Content, contentRect, owner+":content", layer)
	renderWorkbenchNavigatorTChrome(c, overlay.Content, rect, contentRect, owner, layer)
	return Layer{Kind: LayerOverlay, Rect: rect, Lines: contentResult.Lines, ContentOverflow: contentResult.Overflow}
}

func renderWorkbenchNavigatorTChrome(c *canvas, content ContentVM, rect Rect, contentRect Rect, owner string, layer LayerKind) {
	if rect.W < 4 || rect.H < 4 || contentRect.W <= 0 || contentRect.H <= 0 {
		return
	}
	dividerY := contentRect.Y + 1
	if dividerY >= rect.Y+rect.H-1 {
		return
	}
	// 中文说明：Workbench Navigator 的内部 T 分割线要和外框合并，避免普通内容行留下内缩空隙。
	for x := rect.X + 1; x < rect.X+rect.W-1; x++ {
		c.mergeStyledBoxCell(x, dividerY, boxConnLeft|boxConnRight, StyleForeground, owner, layer)
	}
	c.mergeStyledBoxCell(rect.X, dividerY, boxConnUp|boxConnDown|boxConnRight, StyleForeground, owner, layer)
	c.mergeStyledBoxCell(rect.X+rect.W-1, dividerY, boxConnUp|boxConnDown|boxConnLeft, StyleForeground, owner, layer)
	leftWidth := content.Meta.SplitPageLeftWidth
	if leftWidth <= 0 {
		leftWidth = content.Meta.WorkbenchTreeWidth
	}
	dividerX := contentRect.X + leftWidth
	if dividerX <= rect.X || dividerX >= rect.X+rect.W-1 {
		return
	}
	c.mergeStyledBoxCell(dividerX, dividerY, boxConnLeft|boxConnRight|boxConnDown, StyleForeground, owner, layer)
	for y := dividerY + 1; y < rect.Y+rect.H-1; y++ {
		c.mergeStyledBoxCell(dividerX, y, boxConnUp|boxConnDown, StyleForeground, owner, layer)
	}
	c.mergeStyledBoxCell(dividerX, rect.Y+rect.H-1, boxConnUp|boxConnLeft|boxConnRight, StyleForeground, owner, layer)
}

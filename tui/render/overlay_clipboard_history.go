package render

func renderClipboardHistoryOverlay(c *canvas, overlay OverlayVM, rect Rect, contentRect Rect) Layer {
	primitive := OverlayChromePrimitive(overlay, rect, contentRect)
	owner := primitive.Owner
	layer := primitive.Layer
	c.fillStyledRect(rect, StyleForeground, owner, layer)
	c.drawStyledBox(rect, squareBoxStyle, StyleForeground, owner, layer)
	renderChromeCardTitle(c, rect, primitive.Title.Text, "", "", StyleAccent, owner, layer)
	contentResult := renderContent(c, overlay.Content, contentRect, owner+":content", layer)
	renderClipboardHistoryTChrome(c, overlay.Content, rect, contentRect, owner, layer)
	return Layer{Kind: LayerOverlay, Rect: rect, Lines: contentResult.Lines, ContentOverflow: contentResult.Overflow}
}

func renderClipboardHistoryTChrome(c *canvas, content ContentVM, rect Rect, contentRect Rect, owner string, layer LayerKind) {
	if rect.W < 4 || rect.H < 4 || contentRect.W <= 0 || contentRect.H <= 0 {
		return
	}
	searchBottomY := contentRect.Y + 1
	if searchBottomY >= rect.Y+rect.H-1 {
		return
	}
	for x := rect.X + 1; x < rect.X+rect.W-1; x++ {
		c.mergeStyledBoxCell(x, searchBottomY, boxConnLeft|boxConnRight, StyleForeground, owner, layer)
	}
	c.mergeStyledBoxCell(rect.X, searchBottomY, boxConnUp|boxConnDown|boxConnRight, StyleForeground, owner, layer)
	c.mergeStyledBoxCell(rect.X+rect.W-1, searchBottomY, boxConnUp|boxConnDown|boxConnLeft, StyleForeground, owner, layer)
	dividerX := contentRect.X + clipboardHistoryContentNameWidth(content)
	if dividerX <= rect.X || dividerX >= rect.X+rect.W-1 {
		return
	}
	c.mergeStyledBoxCell(dividerX, searchBottomY, boxConnLeft|boxConnRight|boxConnDown, StyleForeground, owner, layer)
	for y := searchBottomY + 1; y < rect.Y+rect.H-1; y++ {
		c.mergeStyledBoxCell(dividerX, y, boxConnUp|boxConnDown, StyleForeground, owner, layer)
	}
	c.mergeStyledBoxCell(dividerX, rect.Y+rect.H-1, boxConnUp|boxConnLeft|boxConnRight, StyleForeground, owner, layer)
}

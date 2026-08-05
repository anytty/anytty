package render

func renderPanelContentOverflowMarkers(c *canvas, layout PanelLayoutPlan, overflow ContentOverflow) {
	style := paneChromeStyle(layout.Panel)
	owner := "panel:" + layout.Panel.ID + ":overflow"
	renderContentOverflowMarkers(c, layout.Rect, layout.ContentRect, overflow, style, owner, LayerChrome)
}

func renderFloatingContentOverflowMarkers(c *canvas, layout FloatingLayoutPlan, overflow ContentOverflow) {
	style := StyleMuted
	if layout.Floating.Active {
		style = StyleAccent
	}
	owner := "floating:" + layout.Floating.ID + ":overflow"
	renderContentOverflowMarkers(c, layout.Rect, layout.ContentRect, overflow, style, owner, LayerFloating)
}

func renderContentOverflowMarkers(c *canvas, chromeRect Rect, contentRect Rect, overflow ContentOverflow, style StyleToken, owner string, layer LayerKind) {
	if chromeRect.W <= 0 || chromeRect.H <= 0 || contentRect.W <= 0 || contentRect.H <= 0 {
		return
	}
	markerStyle := paneChromeOverflowStyle(style)
	if overflow.Top {
		// top marker 位于锁图标前方的顶边槽位，不能和 left marker 合并成角标。
		renderContentOverflowMarker(c, chromeRect.X+1, chromeRect.Y, maxInt(0, chromeRect.W-2), paneChromeOverflowTopGlyph(), markerStyle, owner, layer)
	}
	if overflow.Left {
		if y, ok := contentOverflowVerticalMarkerY(chromeRect, contentRect.Y); ok {
			renderContentOverflowMarker(c, chromeRect.X, y, 1, paneChromeOverflowLeftGlyph(), markerStyle, owner, layer)
		}
	}
	if overflow.Right {
		if y, ok := contentOverflowVerticalMarkerY(chromeRect, contentRect.Y+contentRect.H-1); ok {
			renderContentOverflowMarker(c, chromeRect.X+chromeRect.W-1, y, 1, paneChromeOverflowRightGlyph(), markerStyle, owner, layer)
		}
	}
	if overflow.Bottom {
		width := contentOverflowMarkerWidth(paneChromeOverflowBottomGlyph(), maxInt(0, chromeRect.W-2))
		if width > 0 {
			x := chromeRect.X + chromeRect.W - 1 - width
			renderContentOverflowMarker(c, x, chromeRect.Y+chromeRect.H-1, width, paneChromeOverflowBottomGlyph(), markerStyle, owner, layer)
		}
	}
}

func renderContentOverflowMarker(c *canvas, x int, y int, maxWidth int, marker string, style StyleToken, owner string, layer LayerKind) {
	width := contentOverflowMarkerWidth(marker, maxWidth)
	if width <= 0 {
		return
	}
	c.overlayTextStyled(x, y, width, marker, style, owner, layer)
}

func contentOverflowMarkerWidth(marker string, maxWidth int) int {
	if marker == "" || maxWidth <= 0 {
		return 0
	}
	return minInt(maxWidth, DisplayWidth(marker))
}

func contentOverflowVerticalMarkerY(chromeRect Rect, preferred int) (int, bool) {
	minY := chromeRect.Y + 1
	maxY := chromeRect.Y + chromeRect.H - 2
	if minY > maxY {
		return 0, false
	}
	return clampInt(preferred, minY, maxY), true
}

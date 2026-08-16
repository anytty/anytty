package render

func (c *canvas) drawStyledActiveSplitLeadingEdge(rect Rect, body Rect, style StyleToken, owner string, layer LayerKind) {
	if rect.Y > body.Y {
		// 中文说明：active pane 位于水平 split 下侧时，顶部应是自己的 L 角，不继承上侧 pane 的 T 连接。
		c.drawStyledActiveSplitHBorder(rect.X, splitPaneBorderEndX(rect, body), rect.Y, style, owner, layer, true)
	}
}

func (c *canvas) drawStyledActiveSplitTrailingEdge(rect Rect, body Rect, style StyleToken, owner string, layer LayerKind) {
	if rect.X+rect.W < body.X+body.W {
		// 中文说明：内部 split divider 是共享格子；active pane 在左侧时必须主动声明右边框归属。
		startY, endY := splitPaneVerticalRange(rect, body)
		c.drawStyledPaneVBorder(rect.X+rect.W, startY, endY, style, owner, layer)
	}
	if rect.Y+rect.H < body.Y+body.H {
		// 中文说明：水平 split 同理，active pane 在上侧时不能让下侧 inactive pane 独占分隔线。
		c.drawStyledActiveSplitHBorder(rect.X, splitPaneBorderEndX(rect, body), rect.Y+rect.H, style, owner, layer, false)
	}
}

func (c *canvas) drawStyledActiveSplitHBorder(startX int, endX int, y int, style StyleToken, owner string, layer LayerKind, top bool) {
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
		glyph, ok := boxGlyphForConnections(connections)
		if !ok {
			continue
		}
		c.writeStyledBoxCell(x, y, glyph, style, owner, layer)
	}
}

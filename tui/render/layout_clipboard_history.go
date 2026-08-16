package render

func measureClipboardHistoryOverlay(content ContentVM, viewport Rect) Rect {
	// 中文说明：clipboard history 是全局快速面板，尽量遮住 terminal body，但保留外层 footer 承载操作提示。
	width := viewport.W - clipboardHistoryHorizontalMargin(viewport.W)
	maxHeight := viewport.H - clipboardHistoryVerticalMargin(viewport.H)
	height := minInt(maxInt(len(content.Lines)+2, clipboardHistoryPreferredHeight(viewport.H)), maxHeight)
	width = maxInt(width, clipboardHistoryMinimumWidth(content))
	height = minInt(height, maxHeight)
	width = minInt(width, viewport.W)
	height = minInt(height, viewport.H)
	y := maxInt(0, (viewport.H-height)/2)
	if viewport.H >= 12 && y < 1 {
		y = 1
	}
	return Rect{
		X: maxInt(0, (viewport.W-width)/2),
		Y: y,
		W: width,
		H: height,
	}
}

func measureClipboardHistoryContentRect(rect Rect) Rect {
	return Rect{X: rect.X + 1, Y: rect.Y + 1, W: maxInt(0, rect.W-2), H: maxInt(0, rect.H-2)}
}

func clipboardHistoryHorizontalMargin(width int) int {
	switch {
	case width >= 160:
		return maxInt(24, width/5)
	case width >= 100:
		return 16
	case width >= 48:
		return 8
	case width >= 16:
		return 2
	default:
		return 0
	}
}

func clipboardHistoryVerticalMargin(height int) int {
	switch {
	case height >= 32:
		return 4
	case height >= 18:
		return 3
	case height >= 8:
		return 2
	default:
		return 0
	}
}

func clipboardHistoryPreferredHeight(height int) int {
	switch {
	case height >= 34:
		return height * 3 / 4
	case height >= 20:
		return height - 4
	case height >= 12:
		return height - 2
	default:
		return height
	}
}

func clipboardHistoryMinimumWidth(content ContentVM) int {
	return clipboardHistoryContentNameWidth(content) + 1 + 32 + 2
}

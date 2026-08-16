package render

import "strings"

func canWriteCanvasExtentPlaceholderAsSingleSegment(text string, cell Cell, cellWidth int) bool {
	if cell.TerminalContent ||
		!cell.ANSIStyle.IsZero() ||
		cell.LinkURL != "" ||
		cell.LinkParams != "" ||
		cellWidth != DisplayWidth(text) ||
		strings.Contains(text, "\ufe0f") {
		return false
	}
	glyph := contentViewportOutsideExtentGlyph()
	if glyph == "" || glyph == " " || DisplayWidth(glyph) != 1 {
		return false
	}
	// 中文说明：live extent 占位点是 pane chrome 投影，不是 terminal truth；整段落格可减少拖动浮窗时的矩阵写入量。
	return isRepeatedCanvasOutputText(text, glyph)
}

package render

import "strconv"

func renderWorkbenchNavigatorSnapshotContent(c *canvas, content ContentVM, rect Rect, owner string, layer LayerKind) {
	if content.Kind != ContentWorkbenchTree && content.Kind != ContentTerminalPool && content.Kind != ContentConnections {
		return
	}
	snapshots := content.Meta.WorkbenchSnapshots
	if len(snapshots) == 0 && content.Meta.WorkbenchSnapshotPanel != nil {
		snapshots = []WorkbenchSnapshotVM{{
			Panel:   *content.Meta.WorkbenchSnapshotPanel,
			Rect:    content.Meta.WorkbenchSnapshotRect,
			Content: content.Meta.WorkbenchSnapshotContent,
		}}
	}
	for index, snapshot := range snapshots {
		renderWorkbenchNavigatorSnapshot(c, snapshot, rect, owner, index, layer)
	}
	renderWorkbenchNavigatorSnapshotSplit(c, content.Meta, rect, owner, layer)
	for index, snapshot := range snapshots {
		renderWorkbenchNavigatorSnapshotOverflow(c, snapshot, rect, owner, index, layer)
	}
}

func renderWorkbenchNavigatorSnapshot(c *canvas, snapshot WorkbenchSnapshotVM, rect Rect, owner string, index int, layer LayerKind) {
	// 中文说明：Workbench 投影只携带 snapshot panel VM，真实嵌套绘制留在 renderer runtime 边界。
	snapshotRect := snapshot.Rect
	if snapshotRect.W <= 0 || snapshotRect.H <= 0 {
		return
	}
	snapshotRect.X += rect.X
	snapshotRect.Y += rect.Y
	if snapshotRect.X >= rect.X+rect.W || snapshotRect.Y >= rect.Y+rect.H {
		return
	}
	snapshotRect.W = minInt(snapshotRect.W, rect.X+rect.W-snapshotRect.X)
	snapshotRect.H = minInt(snapshotRect.H, rect.Y+rect.H-snapshotRect.Y)
	if snapshotRect.W <= 0 || snapshotRect.H <= 0 {
		return
	}
	contentRect := snapshot.Content
	contentRect.X += rect.X
	contentRect.Y += rect.Y
	contentRect.W = minInt(contentRect.W, snapshotRect.X+snapshotRect.W-contentRect.X)
	contentRect.H = minInt(contentRect.H, snapshotRect.Y+snapshotRect.H-contentRect.Y)
	if contentRect.W < 0 {
		contentRect.W = 0
	}
	if contentRect.H < 0 {
		contentRect.H = 0
	}
	panel := snapshot.Panel
	style := paneChromeStyle(panel)
	ownerID := owner + ":workbench-snapshot:" + panel.ID + ":" + strconv.Itoa(index)
	if snapshot.PreviewFrame {
		c.drawStyledPaneFrame(snapshotRect, StyleMuted, ownerID+":frame", layer)
		renderWorkbenchNavigatorSnapshotPreviewHeader(c, snapshot, snapshotRect, contentRect, ownerID+":header", layer)
		contentResult := ContentRenderResult{}
		if contentRect.W > 0 && contentRect.H > 0 {
			contentResult = renderContent(c, panel.Content, contentRect, ownerID, layer)
		}
		if snapshot.ShowTitle {
			renderWorkbenchNavigatorSnapshotTitle(c, snapshotRect, panel, style, ownerID+":title", layer)
		}
		renderContentOverflowMarkers(c, snapshotRect, contentRect, contentResult.Overflow, StyleMuted, ownerID+":overflow", layer)
		return
	}
	if snapshot.Borderless {
		if contentRect.W > 0 && contentRect.H > 0 {
			renderContent(c, panel.Content, contentRect, ownerID, layer)
		}
		if snapshot.ShowTitle {
			renderWorkbenchNavigatorBorderlessTitle(c, snapshotRect, panel, style, ownerID+":title", layer)
		}
		return
	}
	c.drawStyledPaneFrame(snapshotRect, style, ownerID+":chrome", layer)
	renderWorkbenchNavigatorSnapshotTitle(c, snapshotRect, panel, style, ownerID+":title", layer)
	if contentRect.W > 0 && contentRect.H > 0 {
		renderContent(c, panel.Content, contentRect, ownerID, layer)
	}
}

func renderWorkbenchNavigatorSnapshotPreviewHeader(c *canvas, snapshot WorkbenchSnapshotVM, snapshotRect Rect, contentRect Rect, owner string, layer LayerKind) {
	if len(snapshot.PreviewHeader) == 0 || snapshotRect.W < 2 || snapshotRect.H < 4 {
		return
	}
	innerWidth := maxInt(0, snapshotRect.W-2)
	headerRows := minInt(len(snapshot.PreviewHeader), maxInt(0, contentRect.Y-snapshotRect.Y-2))
	for index := 0; index < headerRows; index++ {
		c.writeLine(snapshotRect.X+1, snapshotRect.Y+1+index, innerWidth, snapshot.PreviewHeader[index], owner, layer)
	}
	dividerY := contentRect.Y - 1
	if headerRows == 0 || dividerY <= snapshotRect.Y || dividerY >= snapshotRect.Y+snapshotRect.H-1 {
		return
	}
	for x := snapshotRect.X; x < snapshotRect.X+snapshotRect.W; x++ {
		connections := uint8(boxConnLeft | boxConnRight)
		if x == snapshotRect.X {
			connections = boxConnRight
		} else if x == snapshotRect.X+snapshotRect.W-1 {
			connections = boxConnLeft
		}
		c.mergeStyledBoxCell(x, dividerY, connections, StyleMuted, owner+":divider", layer)
	}
}

func renderWorkbenchNavigatorBorderlessTitle(c *canvas, rect Rect, panel PanelVM, style StyleToken, owner string, layer LayerKind) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	prefix := "  "
	if panel.Active {
		prefix = "▌ "
	}
	title := TruncateCells(prefix+panelTitle(panel), rect.W)
	if DisplayWidth(title) <= 0 {
		return
	}
	c.overlayTextStyled(rect.X, rect.Y, DisplayWidth(title), title, style, owner, layer)
}

func renderWorkbenchNavigatorSnapshotSplit(c *canvas, meta ContentMetaVM, rect Rect, owner string, layer LayerKind) {
	canvasRect := meta.WorkbenchSnapshotCanvas
	if canvasRect.W <= 0 || canvasRect.H <= 0 || len(meta.WorkbenchSnapshotSplit.Children) < 2 {
		return
	}
	canvasRect.X += rect.X
	canvasRect.Y += rect.Y
	canvasRect = intersectRect(canvasRect, rect)
	if canvasRect.W <= 0 || canvasRect.H <= 0 {
		return
	}
	renderWorkbenchNavigatorSnapshotSplitNode(c, meta.WorkbenchSnapshotSplit, canvasRect, owner+":workbench-split", layer)
}

func renderWorkbenchNavigatorSnapshotSplitNode(c *canvas, split SplitVM, rect Rect, owner string, layer LayerKind) {
	if rect.W <= 0 || rect.H <= 0 || len(split.Children) < 2 {
		return
	}
	first := split.Children[0]
	second := split.Children[1]
	switch split.Direction {
	case SplitVertical:
		firstWidth := splitFirstExtent(split, rect.W)
		firstRect := Rect{X: rect.X, Y: rect.Y, W: firstWidth, H: rect.H}
		secondRect := Rect{X: rect.X + firstWidth, Y: rect.Y, W: rect.W - firstWidth, H: rect.H}
		for y := rect.Y; y < rect.Y+rect.H; y++ {
			connections := uint8(0)
			if y > rect.Y {
				connections |= boxConnUp
			}
			if y < rect.Y+rect.H-1 {
				connections |= boxConnDown
			}
			c.mergeStyledBoxCell(secondRect.X, y, connections, StyleMuted, owner, layer)
		}
		renderWorkbenchNavigatorSnapshotSplitNode(c, first, firstRect, owner, layer)
		renderWorkbenchNavigatorSnapshotSplitNode(c, second, secondRect, owner, layer)
	default:
		firstHeight := splitFirstExtent(split, rect.H)
		firstRect := Rect{X: rect.X, Y: rect.Y, W: rect.W, H: firstHeight}
		secondRect := Rect{X: rect.X, Y: rect.Y + firstHeight, W: rect.W, H: rect.H - firstHeight}
		for x := rect.X; x < rect.X+rect.W; x++ {
			connections := uint8(0)
			if x > rect.X {
				connections |= boxConnLeft
			}
			if x < rect.X+rect.W-1 {
				connections |= boxConnRight
			}
			c.mergeStyledBoxCell(x, secondRect.Y, connections, StyleMuted, owner, layer)
		}
		renderWorkbenchNavigatorSnapshotSplitNode(c, first, firstRect, owner, layer)
		renderWorkbenchNavigatorSnapshotSplitNode(c, second, secondRect, owner, layer)
	}
}

func renderWorkbenchNavigatorSnapshotOverflow(c *canvas, snapshot WorkbenchSnapshotVM, rect Rect, owner string, index int, layer LayerKind) {
	if !snapshot.PreviewFrame {
		return
	}
	snapshotRect := snapshot.Rect
	snapshotRect.X += rect.X
	snapshotRect.Y += rect.Y
	snapshotRect = intersectRect(snapshotRect, rect)
	if snapshotRect.W <= 0 || snapshotRect.H <= 0 {
		return
	}
	contentRect := snapshot.Content
	contentRect.X += rect.X
	contentRect.Y += rect.Y
	contentRect = intersectRect(contentRect, snapshotRect)
	if contentRect.W <= 0 || contentRect.H <= 0 {
		return
	}
	overflow := RenderContentViewport(ContentRenderRequest{Rect: contentRect, Content: snapshot.Panel.Content}).Overflow
	ownerID := owner + ":workbench-snapshot:" + snapshot.Panel.ID + ":" + strconv.Itoa(index) + ":overflow"
	renderContentOverflowMarkers(c, snapshotRect, contentRect, overflow, StyleMuted, ownerID, layer)
}

func renderWorkbenchNavigatorSnapshotTitle(c *canvas, rect Rect, panel PanelVM, style StyleToken, owner string, layer LayerKind) {
	if rect.W < 8 || rect.H <= 0 {
		return
	}
	title := " " + panelTitle(panel) + " "
	width := minInt(DisplayWidth(title), maxInt(0, rect.W-4))
	if width <= 0 {
		return
	}
	c.overlayTextStyled(rect.X+2, rect.Y, width, title, style, owner, layer)
}

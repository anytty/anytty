package render

func measureCursor(shell ShellVM, plan LayoutPlan) (Cursor, Rect) {
	if overlayOwnsCursor(shell.Overlay) {
		return cursorWithRectOrAnchor(shell.Overlay.Content.Cursor, plan.OverlayContentRect)
	}
	if search := shell.Footer.Search; search.Visible && (search.Cursor.Visible || search.Cursor.Anchor) {
		_, cursor := searchBarPresentation(search, plan.Footer.W)
		return cursorWithRectOrAnchor(cursor, plan.Footer)
	}
	for i := len(plan.Floatings) - 1; i >= 0; i-- {
		floating := plan.Floatings[i]
		if !floating.Floating.Active {
			continue
		}
		cursor := projectContentCursor(floating.Floating.Content, floating.ContentRect)
		if cursorWasClipped(floating.Floating.Content.Cursor, cursor) {
			return Cursor{}, Rect{}
		}
		measuredCursor, cursorRect := cursorWithRectOrAnchor(cursor, floating.ContentRect)
		return cursorCoveredByFloating(measuredCursor, cursorRect, plan.Floatings, i)
	}
	for _, panel := range plan.Panels {
		if !panel.Panel.Active {
			continue
		}
		if panel.Panel.Content.Kind == ContentEmptyPane {
			return Cursor{}, Rect{}
		}
		cursor := projectContentCursor(panel.Panel.Content, panel.ContentRect)
		if cursorWasClipped(panel.Panel.Content.Cursor, cursor) {
			return Cursor{}, Rect{}
		}
		if !cursor.Visible && !cursor.Anchor {
			cursor = shell.Cursor
		}
		measuredCursor, cursorRect := cursorWithRectOrAnchor(cursor, panel.ContentRect)
		return cursorCoveredByFloating(measuredCursor, cursorRect, plan.Floatings, -1)
	}
	if len(plan.Panels) == 0 && shell.Layout.BodyContent.Kind == ContentEmptyPane {
		// 空 tab / 空 workspace 是纯提示页，没有输入焦点，不能停靠宿主光标。
		return Cursor{}, Rect{}
	}
	measuredCursor, cursorRect := cursorWithRectOrAnchor(shell.Cursor, plan.Body)
	return cursorCoveredByFloating(measuredCursor, cursorRect, plan.Floatings, -1)
}

func cursorCoveredByFloating(cursor Cursor, rect Rect, floatings []FloatingLayoutPlan, ownerIndex int) (Cursor, Rect) {
	if !cursor.Visible || rect.W <= 0 || rect.H <= 0 {
		return cursor, rect
	}
	for index := len(floatings) - 1; index >= 0; index-- {
		if index == ownerIndex || (ownerIndex >= 0 && index <= ownerIndex) {
			continue
		}
		floating := floatings[index]
		if floating.Rect.W <= 0 || floating.Rect.H <= 0 {
			continue
		}
		if intersectRect(rect, floating.Rect) == rect {
			// 中文说明：宿主硬光标不能透过 floating；保留 anchor 位置但隐藏可见光标。
			return Cursor{Anchor: true, Row: cursor.Row, Col: cursor.Col, Shape: cursor.Shape}, rect
		}
	}
	return cursor, rect
}

func cursorWasClipped(source Cursor, projected Cursor) bool {
	// 中文说明：真实 terminal cursor 被 view layout 裁掉时应隐藏；
	// 不能走 IME anchor fallback，否则光标会停在内容区旧原点。
	return source.Visible && !projected.Visible && !projected.Anchor
}

func overlayOwnsCursor(overlay OverlayVM) bool {
	return overlay.Kind != OverlayNone && overlay.Content.Kind != "" && (overlay.Opaque || overlay.Content.Kind == ContentPrompt || overlay.Content.Kind == ContentTerminalPicker || overlay.Content.Kind == ContentTerminalPool || overlay.Content.Kind == ContentConnections || overlay.Content.Kind == ContentWorkbenchTree || overlay.Content.Kind == ContentClipboardHistory || overlay.Content.Kind == ContentHelp)
}

func cursorWithRect(cursor Cursor, origin Rect) (Cursor, Rect) {
	if (!cursor.Visible && !cursor.Anchor) || origin.W <= 0 || origin.H <= 0 {
		return Cursor{}, Rect{}
	}
	rect := Rect{X: origin.X + cursor.Col, Y: origin.Y + cursor.Row, W: 1, H: 1}
	if intersectRect(rect, origin) != rect {
		return Cursor{}, Rect{}
	}
	return cursor, rect
}

func cursorWithRectOrAnchor(cursor Cursor, origin Rect) (Cursor, Rect) {
	if measured, rect := cursorWithRect(cursor, origin); measured.Visible || measured.Anchor {
		return measured, rect
	}
	if origin.W <= 0 || origin.H <= 0 {
		return Cursor{}, Rect{}
	}
	// 中文输入法候选区跟随宿主真实光标；内容暂无 cursor 时只锚定位置，不显示系统光标。
	anchor := Cursor{Anchor: true, Row: 0, Col: 0, Shape: CursorShapeBar}
	return anchor, Rect{X: origin.X, Y: origin.Y, W: 1, H: 1}
}

package render

type LayoutPlan struct {
	Viewport           Rect
	Header             Rect
	Footer             Rect
	ShellFrame         Rect
	HeaderTopFrame     Rect
	HeaderDividerFrame Rect
	FooterFrame        Rect
	Body               Rect
	Panels             []PanelLayoutPlan
	Floatings          []FloatingLayoutPlan
	Overlay            Rect
	OverlayContentRect Rect
	OverlayPopup       OverlayPopupLayoutPlan
	Toasts             []Rect
	HitRegions         []HitRegion
	Cursor             Cursor
	CursorRect         Rect
}

type PanelLayoutPlan struct {
	Panel       PanelVM
	Rect        Rect
	Body        Rect
	ShellFrame  Rect
	ContentRect Rect
}

type FloatingLayoutPlan struct {
	Floating    FloatingVM
	Rect        Rect
	ContentRect Rect
}

func MeasureLayout(shell ShellVM, viewport Rect) LayoutPlan {
	viewport.W = normalizeViewportDimension(viewport.W, defaultWidth)
	viewport.H = normalizeViewportDimension(viewport.H, defaultHeight)
	viewport.X = 0
	viewport.Y = 0

	body := viewport
	plan := LayoutPlan{Viewport: viewport}
	if shell.Header.Visible && body.H > 0 {
		headerH := shellBandHeight(viewport.H)
		headerH = minInt(headerH, body.H)
		plan.Header = Rect{X: 0, Y: 0, W: viewport.W, H: headerH}
		body.Y += headerH
		body.H -= headerH
	}
	if shell.Footer.Visible && body.H > 0 {
		footerH := shellBandHeight(viewport.H)
		footerH = minInt(footerH, body.H)
		body.H -= footerH
		plan.Footer = Rect{X: 0, Y: body.Y + body.H, W: viewport.W, H: footerH}
	}
	// 中文说明：header/footer 可见时是顶层产品 chrome；所有 pane、floating 和 overlay 只能使用剩余 body。
	chromeSafeBody := body
	body = layoutBodyOverride(body, shell.Layout.Body, viewport)
	body = intersectRect(body, chromeSafeBody)
	plan.Body = body
	plan.ShellFrame = shellFrameRect(body, shell.Layout.ShellFrame, viewport)
	plan.HeaderTopFrame = footerFrameRect(plan.ShellFrame, shell.Layout.HeaderTopFrame, viewport)
	plan.HeaderDividerFrame = footerFrameRect(plan.ShellFrame, shell.Layout.HeaderDividerFrame, viewport)
	plan.FooterFrame = footerFrameRect(plan.ShellFrame, shell.Layout.FooterFrame, viewport)
	plan.Panels = measurePanels(shell.Layout, body, plan.ShellFrame)
	plan.Floatings = measureFloatings(shell.Layout.Floating, body)
	plan.Overlay = measureOverlayInRect(shell.Overlay, body)
	plan.OverlayContentRect = measureOverlayContentRect(shell.Overlay, plan.Overlay)
	plan.OverlayPopup = measureOverlayPopup(shell.Overlay.Popup, plan.OverlayContentRect, body)
	plan.Toasts = measureToasts(shell.Toasts, viewport)
	plan.Cursor, plan.CursorRect = measureCursor(shell, plan)
	plan.HitRegions = measureHitRegions(shell, plan)
	return plan
}

func footerFrameRect(shellFrame Rect, override Rect, viewport Rect) Rect {
	if override.W <= 0 {
		return shellFrame
	}
	x := override.X
	if x < 0 {
		x = 0
	}
	x = clampInt(x, 0, maxInt(0, viewport.W-1))
	width := minInt(override.W, maxInt(0, viewport.W-x))
	if width <= 0 {
		return shellFrame
	}
	return Rect{X: x, W: width}
}

func layoutBodyOverride(body Rect, override Rect, viewport Rect) Rect {
	if override.W <= 0 && override.H <= 0 && override.X == 0 && override.Y == 0 {
		return body
	}
	if override.X > 0 {
		body.X = clampInt(override.X, 0, maxInt(0, viewport.W-1))
	}
	if override.Y > 0 {
		body.Y = clampInt(override.Y, 0, maxInt(0, viewport.H-1))
	}
	if override.W > 0 {
		body.W = minInt(override.W, maxInt(0, viewport.W-body.X))
	}
	if override.H > 0 {
		body.H = minInt(override.H, maxInt(0, viewport.H-body.Y))
	}
	return body
}

func shellFrameRect(body Rect, override Rect, viewport Rect) Rect {
	if body.W <= 0 || body.H <= 0 {
		return Rect{}
	}
	left := body.X - 1
	if left < 0 {
		left = 0
	}
	right := body.X + body.W
	if right >= viewport.W {
		right = viewport.W - 1
	}
	if override.W > 0 {
		if override.X >= 0 {
			left = clampInt(override.X, 0, maxInt(0, viewport.W-1))
		}
		right = minInt(left+override.W-1, maxInt(0, viewport.W-1))
	}
	width := right - left + 1
	if width <= 0 {
		return Rect{}
	}
	return Rect{X: left, Y: body.Y, W: width, H: body.H}
}

func shellBandHeight(_ int) int {
	// header/footer 是产品栏，不再作为整屏线框的一部分占用第二行。
	return 1
}

func normalizeViewportDimension(value int, fallback int) int {
	if value <= 0 {
		value = fallback
	}
	return maxInt(value, 1)
}

func measurePanels(layout LayoutVM, body Rect, shellFrame Rect) []PanelLayoutPlan {
	if body.W <= 0 || body.H <= 0 || len(layout.Panels) == 0 {
		return nil
	}
	rects := splitPanelRects(layout, body)
	out := make([]PanelLayoutPlan, len(layout.Panels))
	for i, panel := range layout.Panels {
		rect := rects[panel.ID]
		if rect.W == 0 || rect.H == 0 {
			rect = body
		}
		contentRect := measurePanelContentRect(panel, rect, body)
		out[i] = PanelLayoutPlan{Panel: panel, Rect: rect, Body: body, ShellFrame: shellFrame, ContentRect: contentRect}
	}
	return out
}

func measurePanelContentRect(panel PanelVM, rect Rect, body Rect) Rect {
	if panel.Presentation == PanelPresentationCard {
		return Rect{X: rect.X + 1, Y: rect.Y + 1, W: maxInt(0, rect.W-2), H: maxInt(0, rect.H-2)}
	}
	if body.W <= 0 || body.H <= 0 {
		body = rect
	}
	content := Rect{X: rect.X + 1, Y: rect.Y + 1, W: maxInt(0, rect.W-1), H: maxInt(0, rect.H-1)}
	if rect.X+rect.W >= body.X+body.W {
		content.W = maxInt(0, content.W-1)
	}
	if rect.Y+rect.H >= body.Y+body.H {
		content.H = maxInt(0, content.H-1)
	}
	return content
}

func measureFloatings(floatings []FloatingVM, bounds Rect) []FloatingLayoutPlan {
	if len(floatings) == 0 {
		return nil
	}
	sorted := append([]FloatingVM(nil), floatings...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j-1].Z > sorted[j].Z; j-- {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
		}
	}
	out := make([]FloatingLayoutPlan, 0, len(sorted))
	for _, floating := range sorted {
		if floating.Collapsed {
			// hide 语义是整个 floating pane 不可见且不可命中，不只是隐藏内容区。
			continue
		}
		rect := constrainRectToBounds(floating.Rect, bounds)
		if rect.W <= 0 || rect.H <= 0 {
			continue
		}
		contentRect := Rect{X: rect.X + 1, Y: rect.Y + 1, W: maxInt(0, rect.W-2), H: maxInt(0, rect.H-2)}
		out = append(out, FloatingLayoutPlan{Floating: floating, Rect: rect, ContentRect: contentRect})
	}
	return out
}

func constrainRectToBounds(rect Rect, bounds Rect) Rect {
	if bounds.W <= 0 || bounds.H <= 0 || rect.W <= 0 || rect.H <= 0 {
		return Rect{}
	}
	rect.W = minInt(rect.W, bounds.W)
	rect.H = minInt(rect.H, bounds.H)
	if rect.X < bounds.X {
		rect.X = bounds.X
	}
	if rect.Y < bounds.Y {
		rect.Y = bounds.Y
	}
	if rect.X+rect.W > bounds.X+bounds.W {
		rect.X = bounds.X + bounds.W - rect.W
	}
	if rect.Y+rect.H > bounds.Y+bounds.H {
		rect.Y = bounds.Y + bounds.H - rect.H
	}
	return intersectRect(rect, bounds)
}

func measureOverlayInRect(overlay OverlayVM, bounds Rect) Rect {
	if bounds.W <= 0 || bounds.H <= 0 {
		return Rect{}
	}
	// 中文说明：overlay 先按 body 尺寸自我布局，再平移到 body 原点，不能回到整屏 viewport 遮住 header/footer。
	rect := measureOverlay(overlay, Rect{W: bounds.W, H: bounds.H})
	if rect.W <= 0 || rect.H <= 0 {
		return Rect{}
	}
	rect.X += bounds.X
	rect.Y += bounds.Y
	return intersectRect(rect, bounds)
}

func measureOverlay(overlay OverlayVM, viewport Rect) Rect {
	if overlay.Kind == OverlayNone || overlay.Content.Kind == "" {
		return Rect{}
	}
	switch overlay.Content.Kind {
	case ContentTerminalPicker, ContentPrompt:
		return measureCompactOverlay(overlay.Content, viewport)
	case ContentClipboardHistory:
		return measureClipboardHistoryOverlay(overlay.Content, viewport)
	case ContentWorkbenchTree, ContentTerminalPool, ContentConnections:
		return measureWorkbenchNavigatorOverlay(viewport)
	case ContentHelp:
		return measurePageOverlay(viewport)
	}
	return measureDefaultOverlay(viewport)
}

func measureOverlayContentRect(overlay OverlayVM, rect Rect) Rect {
	if rect.W <= 0 || rect.H <= 0 {
		return Rect{}
	}
	if overlay.Content.Kind == ContentTerminalPicker {
		padX, padY := compactOverlayPadding(rect)
		return Rect{X: rect.X + padX, Y: rect.Y + padY, W: maxInt(0, rect.W-padX*2), H: maxInt(0, rect.H-padY*2)}
	}
	if overlay.Content.Kind == ContentClipboardHistory {
		return measureClipboardHistoryContentRect(rect)
	}
	if overlay.Content.Kind == ContentWorkbenchTree || overlay.Content.Kind == ContentTerminalPool || overlay.Content.Kind == ContentConnections {
		return Rect{X: rect.X + 1, Y: rect.Y + 1, W: maxInt(0, rect.W-2), H: maxInt(0, rect.H-2)}
	}
	if rect.W >= 48 && rect.H >= 10 {
		return Rect{X: rect.X + 4, Y: rect.Y + 3, W: maxInt(0, rect.W-8), H: maxInt(0, rect.H-5)}
	}
	if rect.W >= 28 && rect.H >= 6 {
		return Rect{X: rect.X + 2, Y: rect.Y + 2, W: maxInt(0, rect.W-4), H: maxInt(0, rect.H-3)}
	}
	return Rect{X: rect.X + 1, Y: rect.Y + 1, W: maxInt(0, rect.W-2), H: maxInt(0, rect.H-2)}
}

func measureToasts(toasts []ToastVM, viewport Rect) []Rect {
	if len(toasts) == 0 {
		return nil
	}
	rects := make([]Rect, 0, len(toasts))
	y := 3
	bottomLimit := maxInt(0, viewport.H-1)
	for i := len(toasts) - 1; i >= 0 && y < bottomLimit; i-- {
		width := minInt(maxInt(42, viewport.W/3), 56)
		width = minInt(width, viewport.W-4)
		if viewport.W < 40 {
			width = viewport.W
		}
		height := minInt(5, bottomLimit-y)
		if height < 4 {
			break
		}
		rect := Rect{X: maxInt(0, viewport.W-width-2), Y: y, W: width, H: height}
		rects = append(rects, rect)
		y += rect.H + 1
	}
	return rects
}

func intersectRect(left Rect, right Rect) Rect {
	x1 := maxInt(left.X, right.X)
	y1 := maxInt(left.Y, right.Y)
	x2 := minInt(left.X+maxInt(0, left.W), right.X+maxInt(0, right.W))
	y2 := minInt(left.Y+maxInt(0, left.H), right.Y+maxInt(0, right.H))
	if x2 <= x1 || y2 <= y1 {
		return Rect{}
	}
	return Rect{X: x1, Y: y1, W: x2 - x1, H: y2 - y1}
}

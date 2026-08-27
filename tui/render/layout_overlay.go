package render

func measureDefaultOverlay(viewport Rect) Rect {
	width := minInt(maxInt(54, viewport.W*3/5), viewport.W-8)
	height := minInt(maxInt(10, viewport.H/3), viewport.H-4)
	if width < 16 || height < 4 {
		width = maxInt(8, viewport.W)
		height = maxInt(3, minInt(viewport.H, 4))
	}
	return Rect{
		X: maxInt(0, (viewport.W-width)/2),
		Y: maxInt(0, (viewport.H-height)/2),
		W: minInt(width, viewport.W),
		H: minInt(height, viewport.H),
	}
}

func measureCompactOverlay(content ContentVM, viewport Rect) Rect {
	lineWidth := DisplayWidth("terminal picker")
	for _, line := range content.Lines {
		lineWidth = maxInt(lineWidth, line.Width())
	}
	padX, padY := compactOverlayPadding(viewport)
	width := minInt(maxInt(lineWidth+padX*2+2, 64), 80)
	maxHeight := 12
	if content.Kind == ContentTerminalPicker {
		maxHeight = 24
	}
	height := minInt(maxInt(len(content.Lines)+padY*2+2, 6), maxHeight)
	if width > viewport.W-4 {
		width = maxInt(8, viewport.W-2)
	}
	if height > viewport.H-4 {
		height = maxInt(3, viewport.H-2)
	}
	width = minInt(width, viewport.W)
	height = minInt(height, viewport.H)
	return Rect{
		X: maxInt(0, (viewport.W-width)/2),
		Y: maxInt(0, (viewport.H-height)/2),
		W: width,
		H: height,
	}
}

func measureTerminalPickerOverlay(content ContentVM, viewport Rect, picker PickerChromeVM) Rect {
	lineWidth := DisplayWidth("terminal picker")
	for _, line := range content.Lines {
		lineWidth = maxInt(lineWidth, line.Width())
	}
	padX, padY := terminalPickerOverlayPadding(viewport, picker)
	frame := 2
	if picker.Presentation == "flat" {
		frame = 0
	}
	minWidth, maxWidth := 64, 80
	if picker.Width == "wide" {
		minWidth, maxWidth = 88, 112
	}
	width := minInt(maxInt(lineWidth+padX*2+frame, minWidth), maxWidth)
	height := minInt(maxInt(len(content.Lines)+padY*2+frame, 6), 24)
	margin := 4
	if picker.Presentation == "flat" {
		margin = 2
	}
	if width > viewport.W-margin {
		width = maxInt(8, viewport.W-2)
	}
	if height > viewport.H-margin {
		height = maxInt(3, viewport.H-2)
	}
	width = minInt(width, viewport.W)
	height = minInt(height, viewport.H)
	return Rect{X: maxInt(0, (viewport.W-width)/2), Y: maxInt(0, (viewport.H-height)/2), W: width, H: height}
}

func measurePageOverlay(viewport Rect) Rect {
	width := minInt(maxInt(76, viewport.W-12), 132)
	height := minInt(maxInt(18, viewport.H-8), viewport.H-6)
	if width < 40 {
		width = maxInt(8, viewport.W)
	}
	if height < 10 {
		height = maxInt(3, minInt(viewport.H, 12))
	}
	width = minInt(width, viewport.W)
	height = minInt(height, viewport.H)
	return Rect{
		X: maxInt(0, (viewport.W-width)/2),
		Y: maxInt(0, (viewport.H-height)/2),
		W: width,
		H: height,
	}
}

func measureWorkbenchNavigatorOverlay(viewport Rect) Rect {
	width := maxInt(1, viewport.W)
	height := maxInt(1, viewport.H)
	return Rect{
		X: 0,
		Y: 0,
		W: width,
		H: height,
	}
}

func compactOverlayPadding(rect Rect) (int, int) {
	padX := 4
	padY := 2
	if rect.W < 56 {
		padX = 2
	}
	if rect.W < 32 {
		padX = 1
	}
	if rect.W < 32 || rect.H < 6 {
		padY = 1
	}
	return padX, padY
}

func terminalPickerOverlayPadding(rect Rect, picker PickerChromeVM) (int, int) {
	padX, padY := 2, 1
	if picker.Density == "comfortable" {
		padX, padY = 4, 2
	}
	if rect.W < 56 {
		padX = minInt(padX, 2)
	}
	if rect.W < 32 {
		padX = 1
	}
	if rect.W < 32 || rect.H < 6 {
		padY = 1
	}
	return padX, padY
}

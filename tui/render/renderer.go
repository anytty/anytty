package render

type Renderer struct {
	Theme Theme
}

func NewRenderer(theme Theme) Renderer {
	if theme == (Theme{}) {
		theme = DefaultTheme()
	}
	return Renderer{Theme: theme}
}

func (renderer Renderer) RenderResult(vm RenderVM) RenderResult {
	if vm.Theme != (Theme{}) {
		renderer.Theme = vm.Theme
	}
	return renderer.renderFramework(vm)
}

func (renderer Renderer) Render(vm RenderVM) Frame {
	return renderer.RenderResult(vm).Frame()
}

func (renderer Renderer) RenderANSI(vm RenderVM) Frame {
	if vm.Theme != (Theme{}) {
		renderer.Theme = vm.Theme
	}
	frame := renderer.renderFrameworkCanvas(vm)
	defer releaseCanvas(frame.Canvas)
	return Frame{
		ANSILines:   frame.Canvas.ansiLines(frame.Theme),
		Cursor:      frame.Cursor,
		CursorRect:  frame.CursorRect,
		HitRegions:  cloneHitRegions(frame.HitRegions),
		LiveTargets: append([]LiveRenderTarget(nil), frame.LiveTargets...),
		Metadata:    RenderMetadata{Width: frame.Canvas.width, Height: frame.Canvas.height},
		Theme:       frame.Theme,
	}
}

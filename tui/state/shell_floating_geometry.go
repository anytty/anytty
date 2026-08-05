package state

func defaultFloatingRect(rect FloatingRect, boundsW int, boundsH int) FloatingRect {
	if rect.W <= 0 {
		rect.W = defaultFloatingWidth(boundsW)
	}
	if rect.H <= 0 {
		rect.H = defaultFloatingHeight(boundsH)
	}
	return rect
}

func defaultFloatingWidth(boundsW int) int {
	if boundsW <= 0 {
		return 64
	}
	minW := minIntState(64, maxIntState(16, boundsW-8))
	maxW := minIntState(112, maxIntState(16, boundsW-4))
	if maxW < minW {
		maxW = minW
	}
	return clampIntState(boundsW*4/5, minW, maxW)
}

func defaultFloatingHeight(boundsH int) int {
	if boundsH <= 0 {
		return 18
	}
	minH := minIntState(18, maxIntState(4, boundsH-4))
	maxH := minIntState(32, maxIntState(4, boundsH-2))
	if maxH < minH {
		maxH = minH
	}
	return clampIntState(boundsH*3/4, minH, maxH)
}

func floatingPlacementBounds(boundsW int, boundsH int) (int, int) {
	if boundsW <= 0 {
		boundsW = 80
	}
	if boundsH <= 0 {
		boundsH = 24
	}
	return boundsW, boundsH
}

// 中文说明：floating 创建时的默认几何由 ShellStore 统一决定；renderer 只负责把
// 已有 rect clamp 到可见 body，不能在展示层猜测“新窗口应该放哪”。
func cascadeFloatingRect(rect FloatingRect, existing []FloatingPaneState, boundsW int, boundsH int) FloatingRect {
	rect = centerFloatingRect(rect, boundsW, boundsH)
	step := 0
	for _, floating := range existing {
		if !floating.Collapsed {
			step++
		}
	}
	if step == 0 {
		return rect
	}
	const offsetX = 4
	const offsetY = 1
	maxX := maxIntState(0, boundsW-rect.W)
	maxY := maxIntState(0, boundsH-rect.H)
	maxStepsX := 0
	if maxX > rect.X {
		maxStepsX = (maxX - rect.X) / offsetX
	}
	maxStepsY := 0
	if maxY > rect.Y {
		maxStepsY = (maxY - rect.Y) / offsetY
	}
	maxSteps := maxIntState(maxStepsX, maxStepsY)
	if maxSteps > 0 {
		step = step % (maxSteps + 1)
	}
	rect.X += step * offsetX
	rect.Y += step * offsetY
	return clampFloatingRect(rect, boundsW, boundsH)
}

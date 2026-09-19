package kernel

// Layout solves root into a viewport of width x height and returns the root
// frame. The root box always fills the viewport; its own Size is ignored.
// Non-positive viewports and nil roots produce an empty frame.
func Layout(root *Node, width, height int) Frame {
	empty := Frame{Rects: map[string]Rect{}}
	if root == nil || width <= 0 || height <= 0 || !root.IsVisible() {
		return empty
	}
	return solve(root, Rect{X: 0, Y: 0, Width: width, Height: height})
}

// solve places one visible node at rect, then lays out its regular-flow
// children and collects Pos subtrees as overlay frames.
func solve(n *Node, rect Rect) Frame {
	rect = normalizeRect(rect)
	f := Frame{Rects: map[string]Rect{}}
	if n.ID != "" {
		f.Rects[n.ID] = rect
		if !rect.Empty() {
			f.hits = append(f.hits, hit{id: n.ID, rect: rect})
		}
	}
	renderOwn(&f, n, rect)

	placements := assignFlow(n, rect)
	for i := range n.Children {
		child := &n.Children[i]
		if !child.IsVisible() {
			continue
		}
		if child.Pos != nil {
			overlay := solve(child, posRect(child, rect))
			f.mergeRects(overlay.Rects)
			f.OverlayFrames = append(f.OverlayFrames, overlay)
			f.adoptCursor(overlay)
			continue
		}
		childRect, ok := placements[i]
		if !ok {
			continue
		}
		flow := solve(child, childRect)
		f.Lines = append(f.Lines, flow.Lines...)
		f.hits = append(f.hits, flow.hits...)
		f.mergeRects(flow.Rects)
		f.adoptCursor(flow)
		f.OverlayFrames = append(f.OverlayFrames, flow.OverlayFrames...)
	}
	return f
}

// assignFlow resolves the rect of every visible regular-flow child, keyed by
// child index in n.Children. The parent content area is the full parent rect
// (the kernel has no border inset).
func assignFlow(n *Node, rect Rect) map[int]Rect {
	flow := n.EffectiveFlow()
	type plan struct {
		index    int
		node     *Node
		fixed    int
		base     int
		flex     int
		flexible bool
		cross    int
	}
	plans := make([]plan, 0, len(n.Children))
	for i := range n.Children {
		child := &n.Children[i]
		if !child.IsVisible() || child.Pos != nil {
			continue
		}
		p := plan{index: i, node: child}
		if flow == FlowStack {
			plans = append(plans, p)
			continue
		}
		intrinsicW, intrinsicH := child.IntrinsicSize()
		row := flow == FlowRow
		mainSpec, crossSpec := child.Size.Height, child.Size.Width
		mainIntrinsic, crossIntrinsic := intrinsicH, intrinsicW
		if row {
			mainSpec, crossSpec = child.Size.Width, child.Size.Height
			mainIntrinsic, crossIntrinsic = intrinsicW, intrinsicH
		}
		switch {
		case crossSpec > 0:
			p.cross = crossSpec
		case crossIntrinsic > 0:
			p.cross = crossIntrinsic
		default:
			crossAxis := rect.Width
			if row {
				crossAxis = rect.Height
			}
			p.cross = crossAxis
		}
		switch {
		case mainSpec > 0:
			p.fixed = mainSpec
		case child.Size.Flex > 0:
			p.flex = child.Size.Flex
			p.base = mainIntrinsic
			p.flexible = true
		case mainIntrinsic > 0:
			p.fixed = mainIntrinsic
		default:
			p.flexible = true
		}
		plans = append(plans, p)
	}

	mainTotal := rect.Height
	if flow == FlowRow {
		mainTotal = rect.Width
	}
	if flow == FlowStack {
		out := make(map[int]Rect, len(plans))
		for _, p := range plans {
			width, height := rect.Width, rect.Height
			if p.node.Size.Width > 0 {
				width = p.node.Size.Width
			}
			if p.node.Size.Height > 0 {
				height = p.node.Size.Height
			}
			out[p.index] = Rect{X: rect.X, Y: rect.Y, Width: width, Height: height}
		}
		return out
	}

	used := 0
	flexSum := 0
	for _, p := range plans {
		if p.flexible {
			used += p.base
			flexSum += p.flex
		} else {
			used += p.fixed
		}
	}
	remaining := max(0, mainTotal-used)
	if flexSum > 0 {
		distributed := 0
		lastFlex := -1
		for i := range plans {
			if !plans[i].flexible || plans[i].flex <= 0 {
				continue
			}
			extra := remaining * plans[i].flex / flexSum
			plans[i].base += extra
			distributed += extra
			lastFlex = i
		}
		if lastFlex >= 0 {
			plans[lastFlex].base += remaining - distributed
		}
	} else {
		count := 0
		lastStretch := -1
		for i := range plans {
			if plans[i].flexible {
				count++
				lastStretch = i
			}
		}
		if count > 0 {
			each := remaining / count
			for i := range plans {
				if plans[i].flexible {
					plans[i].base += each
				}
			}
			if lastStretch >= 0 {
				plans[lastStretch].base += remaining - each*count
			}
		}
	}

	row := flow == FlowRow
	mainPos := rect.Y
	if row {
		mainPos = rect.X
	}
	out := make(map[int]Rect, len(plans))
	for _, p := range plans {
		size := p.fixed
		if p.flexible {
			size = p.base
		}
		size = max(0, size)
		cross := max(0, p.cross)
		var r Rect
		if row {
			r = Rect{X: mainPos, Y: rect.Y, Width: size, Height: cross}
		} else {
			r = Rect{X: rect.X, Y: mainPos, Width: cross, Height: size}
		}
		out[p.index] = r
		mainPos += size
	}
	return out
}

// posRect places a Pos subtree inside the parent rect (the parent content
// area is the full rect). Unset axes fall back to the intrinsic size, then
// to the parent content size.
func posRect(n *Node, rect Rect) Rect {
	width, height := n.IntrinsicSize()
	if n.Size.Width > 0 {
		width = n.Size.Width
	}
	if width <= 0 {
		width = rect.Width
	}
	if n.Size.Height > 0 {
		height = n.Size.Height
	}
	if height <= 0 {
		height = rect.Height
	}
	return Rect{X: rect.X + n.Pos.X, Y: rect.Y + n.Pos.Y, Width: width, Height: height}
}

func normalizeRect(r Rect) Rect {
	if r.Width < 0 {
		r.Width = 0
	}
	if r.Height < 0 {
		r.Height = 0
	}
	return r
}

func (f *Frame) mergeRects(rects map[string]Rect) {
	for id, r := range rects {
		f.Rects[id] = r
	}
}

func (f *Frame) adoptCursor(src Frame) {
	if !src.HasCursor {
		return
	}
	f.HasCursor = true
	f.CursorRect = src.CursorRect
	f.CursorShape = src.CursorShape
}

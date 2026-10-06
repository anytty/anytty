package kernel

import "strings"

// Layout solves root into a viewport of width x height and returns the root
// frame. The root box always fills the viewport; its own Size is ignored.
// Non-positive viewports and nil roots produce an empty frame.
func Layout(root *Node, width, height int) Frame {
	if root == nil || width <= 0 || height <= 0 || !root.IsVisible() {
		return Frame{}
	}
	hints := countHints(root)
	b := frameBuilder{
		index: &rectIndex{items: make([]rectNode, 0, hints.ids)},
		lines: make([]Line, 0, hints.lines),
		hits:  make([]hit, 0, hints.ids),
	}
	solve(root, Rect{X: 0, Y: 0, Width: width, Height: height}, &b, 0)
	return b.frame()
}

// frameBuilder accumulates the solved layout of one frame while it is walked.
// Regular-flow descendants all write into the same builder, so no per-node map
// or slice is allocated and nothing is merged upward; each Pos subtree gets
// its own builder because it becomes its own OverlayFrame.
//
// A cache-aware builder (cached) instead gives every child its own builder so
// the child's frame can be recorded on the node for the next call; a child
// whose Solved matches its rect is spliced instead of solved.
type frameBuilder struct {
	// cached selects the cache-aware solver (see LayoutCached).
	cached bool

	index       *rectIndex
	lines       []Line
	hits        []hit
	overlays    []Frame
	cursorRect  Rect
	hasCursor   bool
	cursorShape string

	// plans is a per-depth scratch stack of flow plans. A container at depth d
	// keeps its slice alive while its children recurse into depth d+1, and the
	// slot is reused by the next sibling at depth d, so assignFlow allocates
	// only on the first (deepest) encounter.
	plans [][]flowPlan
}

func (b *frameBuilder) frame() Frame {
	if b.index == nil {
		b.index = &rectIndex{}
	}
	return Frame{
		index:         b.index,
		Lines:         b.lines,
		OverlayFrames: b.overlays,
		CursorRect:    b.cursorRect,
		HasCursor:     b.hasCursor,
		CursorShape:   b.cursorShape,
		hits:          b.hits,
	}
}

func (b *frameBuilder) adoptCursor(src *frameBuilder) {
	if !src.hasCursor {
		return
	}
	b.hasCursor = true
	b.cursorRect = src.cursorRect
	b.cursorShape = src.cursorShape
}

// flowPlan is the resolved placement of one visible regular-flow child. It is
// reused across siblings at the same tree depth.
type flowPlan struct {
	node     *Node
	rect     Rect
	fixed    int
	base     int
	flex     int
	flexible bool
	cross    int
}

// solve places one visible node at rect, then lays out its regular-flow
// children and collects Pos subtrees as overlay frames. Regular-flow children
// share b; a Pos child builds its own frameBuilder and is merged into b.
//
// A cache-aware builder (b.cached) solves every child into its own builder so
// the child's frame can be recorded, and splices a child whose Solved matches
// the child's absolute rect instead of walking it. A plain builder shares one
// builder across the regular flow, which allocates less.
func solve(n *Node, rect Rect, b *frameBuilder, depth int) {
	rect = normalizeRect(rect)
	if n.ID != "" {
		b.index.items = append(b.index.items, rectNode{id: n.ID, rect: rect})
		b.index.count++
		if !rect.Empty() {
			b.hits = append(b.hits, hit{id: n.ID, rect: rect})
		}
	}
	renderOwn(b, n, rect)

	if len(n.Children) == 0 {
		return
	}
	if depth == len(b.plans) {
		b.plans = append(b.plans, nil)
	}
	if cap(b.plans[depth]) < len(n.Children) {
		b.plans[depth] = make([]flowPlan, 0, len(n.Children))
	}
	plans := assignFlow(n, rect, b.plans[depth][:0])
	b.plans[depth] = plans

	pi := 0
	for i := range n.Children {
		child := &n.Children[i]
		if !child.IsVisible() {
			continue
		}
		if child.Pos != nil {
			pr := normalizeRect(posRect(child, rect))
			f, cur := solveOverlay(child, pr, b.cached)
			// A Pos subtree becomes its own overlay frame: only its rects are
			// lifted into the parent, its lines and hits stay in the overlay.
			if f.index != nil {
				b.index.items = append(b.index.items, rectNode{ref: f.index})
				b.index.count += f.index.count
			}
			b.overlays = append(b.overlays, f)
			b.adoptCursor(&cur)
			continue
		}
		plan := plans[pi]
		pi++
		solveChild(child, plan.rect, b, depth+1)
	}
}

// solveOverlay produces the frame of one Pos child at its absolute rect. With
// a cache-aware builder it reuses the child's Solved frame when the rect is
// unchanged, otherwise it solves the subtree into its own builder and records
// the frame on the child. It returns the frame and a carrier holding the
// cursor the parent adopts.
func solveOverlay(n *Node, pr Rect, cached bool) (Frame, frameBuilder) {
	if cached {
		if s := n.Cached(); s != nil && s.Rect == pr {
			cur := frameBuilder{
				hasCursor:   s.Frame.HasCursor,
				cursorRect:  s.Frame.CursorRect,
				cursorShape: s.Frame.CursorShape,
			}
			return s.Frame, cur
		}
	}
	var ob frameBuilder
	ob.cached = cached
	ob.index = &rectIndex{}
	solve(n, pr, &ob, 0)
	f := ob.frame()
	if cached {
		n.SetCached(&Solved{Rect: pr, Frame: f})
	}
	return f, ob
}

// solveChild solves one regular-flow child of a builder. A cache-aware builder
// splices a child whose Solved matches its absolute rect; otherwise it solves
// the child into its own builder so its frame can be recorded. A plain builder
// recurses with the shared builder.
func solveChild(n *Node, rect Rect, b *frameBuilder, depth int) {
	if !b.cached {
		solve(n, rect, b, depth)
		return
	}
	pr := normalizeRect(rect)
	if s := n.Cached(); s != nil && s.Rect == pr {
		b.addFrame(s.Frame)
		return
	}
	own := frameBuilder{cached: true, index: &rectIndex{}}
	solve(n, pr, &own, 0)
	f := own.frame()
	n.SetCached(&Solved{Rect: pr, Frame: f})
	b.addFrame(f)
}

// assignFlow resolves the rect of every visible regular-flow child, appending
// one plan per child to plans (indexed by child index in n.Children). Pos and
// invisible children get no plan. The parent content area is the full parent
// rect (the kernel has no border inset).
func assignFlow(n *Node, rect Rect, plans []flowPlan) []flowPlan {
	flow := n.EffectiveFlow()
	for i := range n.Children {
		child := &n.Children[i]
		if !child.IsVisible() || child.Pos != nil {
			continue
		}
		p := flowPlan{node: child}
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
	if len(plans) == 0 {
		return plans
	}

	mainTotal := rect.Height
	if flow == FlowRow {
		mainTotal = rect.Width
	}
	if flow == FlowStack {
		for i := range plans {
			width, height := rect.Width, rect.Height
			if plans[i].node.Size.Width > 0 {
				width = plans[i].node.Size.Width
			}
			if plans[i].node.Size.Height > 0 {
				height = plans[i].node.Size.Height
			}
			plans[i].rect = Rect{X: rect.X, Y: rect.Y, Width: width, Height: height}
		}
		return plans
	}

	used := 0
	flexSum := 0
	for i := range plans {
		if plans[i].flexible {
			used += plans[i].base
			flexSum += plans[i].flex
		} else {
			used += plans[i].fixed
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
	for i := range plans {
		p := &plans[i]
		size := p.fixed
		if p.flexible {
			size = p.base
		}
		size = max(0, size)
		cross := max(0, p.cross)
		if row {
			p.rect = Rect{X: mainPos, Y: rect.Y, Width: size, Height: cross}
		} else {
			p.rect = Rect{X: rect.X, Y: mainPos, Width: cross, Height: size}
		}
		mainPos += size
	}
	return plans
}

// layoutHints bounds the allocations for one solved frame: ids counts nodes
// that can contribute a hit (every reachable id-bearing node, spliced subtrees
// included), ownIDs counts nodes this builder will solve itself (a cached
// subtree contributes none: it is linked by pointer), lines counts the content
// lines that can be rendered. Invisible subtrees contribute nothing.
type layoutHints struct {
	ids    int
	ownIDs int
	// refs is the number of spliced indexes a cached solve will append: one
	// per cached subtree stop (and per Pos overlay), so it presizes the owned
	// items slice together with ownIDs.
	refs  int
	lines int
}

func countHints(n *Node) layoutHints {
	if !n.IsVisible() {
		return layoutHints{}
	}
	h := layoutHints{}
	if n.ID != "" {
		h.ids++
		h.ownIDs++
	}
	if n.Content != nil {
		switch {
		case len(n.Content.Lines) > 0:
			h.lines += len(n.Content.Lines)
		case n.Content.Text != "":
			h.lines += strings.Count(n.Content.Text, "\n") + 1
		}
	}
	for i := range n.Children {
		ch := countHints(&n.Children[i])
		h.ids += ch.ids
		h.ownIDs += ch.ownIDs
		h.lines += ch.lines
	}
	return h
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

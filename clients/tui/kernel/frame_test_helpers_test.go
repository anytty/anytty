package kernel

// rectsForTest materializes the tree-shaped rect index into the flat
// id->Rect map the geometry tests were written against. Test-only: the
// production layout path never builds this map.
func (f Frame) rectsForTest() map[string]Rect {
	out := make(map[string]Rect, f.RectCount())
	f.RectsIterate(func(id string, r Rect) bool {
		out[id] = r
		return true
	})
	return out
}

// flattenFrame returns a deep copy of f whose rect index has no spliced refs:
// every reachable entry is written into the copy's own storage. Two frames of
// equal layout flatten to reflect.DeepEqual values regardless of how the
// original index was shared.
func flattenFrame(f Frame) Frame {
	idx := &rectIndex{}
	f.RectsIterate(func(id string, r Rect) bool {
		idx.items = append(idx.items, rectNode{id: id, rect: r})
		idx.count++
		return true
	})
	out := Frame{
		index:       idx,
		Lines:       append([]Line(nil), f.Lines...),
		CursorRect:  f.CursorRect,
		HasCursor:   f.HasCursor,
		CursorShape: f.CursorShape,
		hits:        append([]hit(nil), f.hits...),
	}
	if len(f.OverlayFrames) > 0 {
		out.OverlayFrames = make([]Frame, len(f.OverlayFrames))
		for i := range f.OverlayFrames {
			out.OverlayFrames[i] = flattenFrame(f.OverlayFrames[i])
		}
	}
	return out
}

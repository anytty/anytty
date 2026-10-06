package widgets

// Hover is a pure hover tracker over a set of HitRegions. Update resolves the
// region under (x, y) and returns its id (empty when none); StyleFor returns
// the style the hovered id should render with, falling back to the base style
// for every non-hovered box. A region id that equals Node means "the whole
// hover surface is one target"; CurrentStyle is then StyleFor(regions[id])'s
// result, i.e. Style while hovered.
type Hover struct {
	Node         string
	Regions      []HitRegion
	Style        string
	CurrentStyle string
}

// Update hit-tests (x, y) and returns the hovered region id, or "" when the
// pointer is over no region.
func (h *Hover) Update(x, y int) string {
	region, ok := Hit(h.Regions, x, y)
	if !ok {
		h.CurrentStyle = ""
		return ""
	}
	h.CurrentStyle = firstNonEmpty(region.ID, h.Node)
	return h.CurrentStyle
}

// StyleFor returns the hover accent when id is the hovered target and the base
// style otherwise. An empty id means nothing is hovered.
func (h Hover) StyleFor(id string) string {
	if id != "" && id == h.CurrentStyle {
		return h.Style
	}
	return ""
}

// Hovered reports the currently hovered id.
func (h Hover) Hovered() string { return h.CurrentStyle }

// Clear forgets the hovered target.
func (h *Hover) Clear() { h.CurrentStyle = "" }

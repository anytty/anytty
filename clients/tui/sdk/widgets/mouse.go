package widgets

import (
	"time"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// DefaultRowHeight is the cell height of one List/Table row region.
const DefaultRowHeight = 1

// DefaultClickThreshold is the maximum gap between two clicks of the same
// gesture (double/triple click).
const DefaultClickThreshold = 400 * time.Millisecond

// HitRegion is one rectangular hit-test target: the box id, its viewport rect
// and opaque payload (row index, row/col pair, ...). A W or H <= 0 means the
// region is unbounded on that axis, so a row whose width was never declared
// stays clickable.
type HitRegion struct {
	ID   string
	X, Y int
	W, H int
	Data any
}

// Contains reports whether (x, y) falls inside the region.
func (r HitRegion) Contains(x, y int) bool {
	if r.W > 0 && (x < r.X || x >= r.X+r.W) {
		return false
	}
	if r.H > 0 && (y < r.Y || y >= r.Y+r.H) {
		return false
	}
	return true
}

// Rect returns the region bounds.
func (r HitRegion) Rect() Rect { return Rect{X: r.X, Y: r.Y, W: r.W, H: r.H} }

// Hit returns the topmost region containing (x, y): later regions win, so a
// caller orders regions bottom-to-top (the declaration/z order).
func Hit(regions []HitRegion, x, y int) (HitRegion, bool) {
	for i := len(regions) - 1; i >= 0; i-- {
		if regions[i].Contains(x, y) {
			return regions[i], true
		}
	}
	return HitRegion{}, false
}

// HitID is Hit's convenience form: the id of the topmost hit, empty and false
// when nothing matches.
func HitID(regions []HitRegion, x, y int) (string, bool) {
	region, ok := Hit(regions, x, y)
	if !ok {
		return "", false
	}
	return region.ID, true
}

// ListRegions builds one region per visible List row. originX/originY are the
// widget's top-left in the same coordinate space as the event; width <= 0
// leaves the row width unbounded. Data is the absolute row index.
func ListRegions(list List, originX, originY, width int) []HitRegion {
	start, end := list.VisibleRange()
	if end <= start {
		return nil
	}
	top := originY
	if list.Header != "" {
		top++
	}
	regions := make([]HitRegion, 0, end-start)
	for row := start; row < end; row++ {
		id := ""
		if list.RowID != nil {
			id = list.RowID(row)
		}
		regions = append(regions, HitRegion{
			ID:   id,
			X:    originX,
			Y:    top + (row - start),
			W:    width,
			H:    DefaultRowHeight,
			Data: row,
		})
	}
	return regions
}

// VirtualListRegions is ListRegions for a VirtualList; the row's own ID wins
// and Data is the absolute row index.
func VirtualListRegions(list VirtualList, originX, originY, width int) []HitRegion {
	start, end := list.VisibleRange()
	if end <= start {
		return nil
	}
	top := originY
	if list.Header != "" {
		top++
	}
	regions := make([]HitRegion, 0, end-start)
	for row := start; row < end; row++ {
		regions = append(regions, HitRegion{
			ID:   list.Rows[row].ID,
			X:    originX,
			Y:    top + (row - start),
			W:    width,
			H:    DefaultRowHeight,
			Data: row,
		})
	}
	return regions
}

// TableRegions builds one region per visible Table cell. Data is a
// [2]int{row, col} pair. Cells are laid out by ColumnWidths with the separator
// gutter between them; row -1 is the header and -2 the footer, so callers can
// ignore them by filtering on Data.
func TableRegions(table Table, originX, originY int) []HitRegion {
	if len(table.Columns) == 0 {
		return nil
	}
	widths := table.ColumnWidths()
	sep := table.sepWidth()
	top := originY
	if !table.HideHeader {
		if table.Rule {
			top += 2
		} else {
			top++
		}
	}
	xs := make([]int, len(widths))
	x := originX
	for col, width := range widths {
		if col > 0 {
			x += sep
		}
		xs[col] = x
		x += width
	}
	regions := make([]HitRegion, 0, table.RowCount()*len(table.Columns))
	for row := 0; row < table.RowCount(); row++ {
		cellID := ""
		if table.RowID != nil {
			cellID = table.RowID(row)
		}
		for col, width := range widths {
			regions = append(regions, HitRegion{
				X:    xs[col],
				Y:    top + row,
				W:    width,
				H:    DefaultRowHeight,
				ID:   cellID,
				Data: [2]int{row, col},
			})
		}
	}
	return regions
}

// RowAt returns the absolute List row at viewport y, accounting for the header
// row and the current window. It reports whether y landed on a data row.
func (l List) RowAt(y int) (int, bool) {
	top := 0
	if l.Header != "" {
		top++
	}
	size := windowSize(l.Height, chromeRows(l.Header, l.Footer), len(l.Items))
	if size <= 0 || y < top || y >= top+size {
		return 0, false
	}
	start, _ := l.VisibleRange()
	row := start + (y - top)
	if row < 0 || row >= len(l.Items) {
		return 0, false
	}
	return row, true
}

// RowAt returns the absolute VirtualList row at viewport y.
func (v VirtualList) RowAt(y int) (int, bool) {
	top := 0
	if v.Header != "" {
		top++
	}
	size := windowSize(v.Height, chromeRows(v.Header, v.Footer), len(v.Rows))
	if size <= 0 || y < top || y >= top+size {
		return 0, false
	}
	start, _ := v.VisibleRange()
	row := start + (y - top)
	if row < 0 || row >= len(v.Rows) {
		return 0, false
	}
	return row, true
}

// RowAt returns the data row at viewport y, skipping the header and rule. It
// reports false for the header/rule/footer or out-of-range rows.
func (t Table) RowAt(y int) (int, bool) {
	if len(t.Columns) == 0 {
		return 0, false
	}
	top := 0
	if !t.HideHeader {
		top++
		if t.Rule {
			top++
		}
	}
	row := y - top
	if row < 0 || row >= t.RowCount() {
		return 0, false
	}
	return row, true
}

// ApplyWheel folds one wheel delta into an offset and clamps it to
// [0, total-visible]. Positive delta scrolls down/right. A fully visible
// (or empty) content set pins the offset to 0.
func ApplyWheel(offset, total, visible, delta int) int {
	offset += delta
	if visible <= 0 || visible >= total {
		return 0
	}
	return clampOffset(offset, visible, total)
}

// OnWheel applies one protocol wheel event to the list offset, leaving the
// selection untouched. A nil event is a no-op.
func (l *List) OnWheel(ev *pb.WheelEvent) {
	if ev == nil {
		return
	}
	l.Offset = ApplyWheel(l.Offset, len(l.Items), windowSize(l.Height, chromeRows(l.Header, l.Footer), len(l.Items)), int(ev.GetDelta()))
}

// OnWheel applies one protocol wheel event to the list offset.
func (v *VirtualList) OnWheel(ev *pb.WheelEvent) {
	if ev == nil {
		return
	}
	v.Offset = ApplyWheel(v.Offset, len(v.Rows), windowSize(v.Height, chromeRows(v.Header, v.Footer), len(v.Rows)), int(ev.GetDelta()))
}

// TableScroll is the Table counterpart of OnWheel: Table has no window offset
// field, so the caller owns the offset and folds wheel deltas through here.
// visible is the number of on-screen data rows.
func TableScroll(offset, total, visible, delta int) int {
	return ApplyWheel(offset, total, visible, delta)
}

// Drag is a pure mouse drag selection model: Begin on a press, Update on each
// motion, End on release. Coordinates are absolute viewport cells; all methods
// are pointer-based and emit nothing.
type Drag struct {
	Active      bool
	StartX      int
	StartY      int
	X           int
	Y           int
	Button      string
	ThresholdPx int
}

// Begin starts a drag at (x, y) with the given button.
func (d *Drag) Begin(x, y int, button string) {
	d.Active = true
	d.StartX, d.StartY = x, y
	d.X, d.Y = x, y
	d.Button = button
}

// Update moves the drag's current position. A drag that was never begun is
// left inactive.
func (d *Drag) Update(x, y int) {
	if !d.Active {
		return
	}
	d.X, d.Y = x, y
}

// End finishes the drag and returns the normalized selection rect plus whether
// a drag was active. The current position is not clamped to the start.
func (d *Drag) End() (Rect, bool) {
	if !d.Active {
		return Rect{}, false
	}
	d.Active = false
	x, y, w, h := d.Rect()
	return Rect{X: x, Y: y, W: w, H: h}, true
}

// Rect returns the drag selection normalized to a non-negative origin:
// x/y are the min corner, w/h the inclusive span (at least 1).
func (d Drag) Rect() (x, y, w, h int) {
	minX, maxX := d.StartX, d.X
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	minY, maxY := d.StartY, d.Y
	if minY > maxY {
		minY, maxY = maxY, minY
	}
	return minX, minY, maxX - minX + 1, maxY - minY + 1
}

// SelectRange normalizes two positions into an inclusive (lo, hi) pair.
func SelectRange(a, b int) (lo, hi int) {
	if a > b {
		return b, a
	}
	return a, b
}

// ListSelectRange maps two viewport y positions to the inclusive range of
// absolute List rows they span, clamped to the current window. It reports
// false when neither position lands on a row.
func ListSelectRange(list List, y0, y1 int) (int, int, bool) {
	lo, hi := SelectRange(y0, y1)
	first, ok := list.RowAt(lo)
	if !ok {
		first, ok = list.RowAt(hi)
		if !ok {
			return 0, 0, false
		}
	}
	last, ok := list.RowAt(hi)
	if !ok {
		last = first
	}
	if first > last {
		first, last = last, first
	}
	return first, last, true
}

// VirtualListSelectRange is ListSelectRange for a VirtualList.
func VirtualListSelectRange(list VirtualList, y0, y1 int) (int, int, bool) {
	lo, hi := SelectRange(y0, y1)
	first, ok := list.RowAt(lo)
	if !ok {
		first, ok = list.RowAt(hi)
		if !ok {
			return 0, 0, false
		}
	}
	last, ok := list.RowAt(hi)
	if !ok {
		last = first
	}
	if first > last {
		first, last = last, first
	}
	return first, last, true
}

// ClickTracker classifies clicks into single/double/triple clicks purely from
// timing and position. Callers keep one tracker per target.
type ClickTracker struct {
	LastAt    time.Time
	LastX     int
	LastY     int
	Threshold time.Duration
	Count     int
}

// Click records a click at (x, y) happening at at and returns the click count
// (1, 2 or 3). A gap longer than Threshold or a move to another cell restarts
// the count; a fourth click starts a new gesture.
func (c *ClickTracker) Click(x, y int, at time.Time) int {
	threshold := c.Threshold
	if threshold <= 0 {
		threshold = DefaultClickThreshold
	}
	sameCell := c.LastX == x && c.LastY == y
	close := !c.LastAt.IsZero() && at.Sub(c.LastAt) <= threshold
	if close && sameCell && c.Count >= 1 && c.Count < 3 {
		c.Count++
	} else {
		c.Count = 1
	}
	c.LastAt = at
	c.LastX, c.LastY = x, y
	return c.Count
}

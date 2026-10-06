package render

import (
	"bytes"

	"github.com/rivo/uniseg"
)

// Line is one styled horizontal text run at an absolute cell position. Y is
// a row, X is the starting column. Text is plain UTF-8; ANSI bytes are never
// embedded (the host renderer owns all escape output).
type Line struct {
	X     int
	Y     int
	Text  string
	Style Token
}

// Cell is one framebuffer cell. Text holds exactly one grapheme cluster
// (a single rune for ASCII, a whole cluster for emoji); Width is still
// reported for wide cells. The second half of a wide cluster is a
// Continuation cell with empty text.
type Cell struct {
	Text         string
	Width        int
	Continuation bool
	Style        Token
}

// Blank reports whether the cell carries no content or style.
func (c Cell) Blank() bool {
	return c.Text == "" && !c.Continuation && c.Style == ""
}

// Frame is a fixed-size cell grid plus a cursor. Blit styled lines into it,
// then call Bytes(prev) to obtain the minimal ANSI update for the TTY. The
// zero value is not usable; use NewFrame.
type Frame struct {
	cols  int
	rows  int
	cells []Cell

	cursorX       int
	cursorY       int
	cursorVisible bool
	cursorShape   string

	theme Theme
}

// ScrollRect identifies a panel content rectangle that may have moved
// vertically since the previous frame. The renderer uses it only when the rows
// inside the rectangle prove that they are a real scroll; ordinary frame diffs
// remain unchanged.
type ScrollRect struct {
	X, Y          int
	Width, Height int
}

// NewFrame allocates an empty cols×rows framebuffer with the default theme.
func NewFrame(cols, rows int) *Frame {
	if cols < 0 {
		cols = 0
	}
	if rows < 0 {
		rows = 0
	}
	return &Frame{
		cols:  cols,
		rows:  rows,
		cells: make([]Cell, cols*rows),
		theme: DefaultTheme(),
	}
}

// Reset clears the framebuffer for reuse. A size change replaces the cell
// storage; an unchanged size only clears the existing cells and cursor state.
// The theme is reset to the default, matching NewFrame.
func (f *Frame) Reset(cols, rows int) {
	if f == nil {
		return
	}
	if cols < 0 {
		cols = 0
	}
	if rows < 0 {
		rows = 0
	}
	if f.cols != cols || f.rows != rows {
		f.cols = cols
		f.rows = rows
		f.cells = make([]Cell, cols*rows)
	} else {
		f.Clear()
	}
	f.cursorX = 0
	f.cursorY = 0
	f.cursorVisible = false
	f.cursorShape = ""
	f.theme = DefaultTheme()
}

// Cols returns the grid width in cells.
func (f *Frame) Cols() int { return f.cols }

// Rows returns the grid height in rows.
func (f *Frame) Rows() int { return f.rows }

// SetTheme sets the palette used when resolving host-internal tokens.
// Program styles are explicit and ignore the palette entirely.
func (f *Frame) SetTheme(theme Theme) { f.theme = theme.WithFallback() }

// Theme returns the effective palette.
func (f *Frame) Theme() Theme { return f.theme.WithFallback() }

// Clear resets every cell and hides the cursor.
func (f *Frame) Clear() {
	for i := range f.cells {
		f.cells[i] = Cell{}
	}
	f.cursorVisible = false
	f.cursorShape = ""
}

// CellAt returns the cell at (x, y), or the zero Cell when out of bounds.
func (f *Frame) CellAt(x, y int) Cell {
	if f == nil || x < 0 || y < 0 || x >= f.cols || y >= f.rows {
		return Cell{}
	}
	return f.cells[y*f.cols+x]
}

// SetCursor places the visible cursor at the zero-based cell (x, y), clipped
// to the framebuffer. An empty framebuffer cannot carry a visible cursor.
func (f *Frame) SetCursor(x, y int) {
	if f == nil || f.cols <= 0 || f.rows <= 0 {
		if f != nil {
			f.cursorVisible = false
		}
		return
	}
	if x < 0 {
		x = 0
	} else if x >= f.cols {
		x = f.cols - 1
	}
	if y < 0 {
		y = 0
	} else if y >= f.rows {
		y = f.rows - 1
	}
	f.cursorX = x
	f.cursorY = y
	f.cursorVisible = true
}

// SetCursorShape selects the hardware cursor shape. An empty shape leaves the
// terminal's current/default shape unchanged; unknown values are ignored.
func (f *Frame) SetCursorShape(shape string) {
	if f == nil {
		return
	}
	switch shape {
	case "block", "underline", "bar":
		f.cursorShape = shape
	case "":
		f.cursorShape = ""
	}
}

// HideCursor hides the cursor.
func (f *Frame) HideCursor() { f.cursorVisible = false }

// Cursor returns the cursor position and visibility.
func (f *Frame) Cursor() (x, y int, visible bool) {
	return f.cursorX, f.cursorY, f.cursorVisible
}

// CursorShape returns the optional shape requested for the hardware cursor.
func (f *Frame) CursorShape() string {
	if f == nil {
		return ""
	}
	return f.cursorShape
}

// Blit draws lines in order; later lines overwrite earlier ones.
func (f *Frame) Blit(lines ...Line) {
	for _, line := range lines {
		f.BlitLine(line)
	}
}

// BlitLine draws one styled run at its absolute position. The run is clipped
// to the grid; a wide grapheme is dropped rather than split when only one of
// its two cells fits. Overwriting half of a wide grapheme clears the other
// half so the grid never contains orphaned continuation cells.
func (f *Frame) BlitLine(line Line) {
	if f == nil || line.Y < 0 || line.Y >= f.rows {
		return
	}
	text := VisibleText(line.Text)
	if simpleASCII(text) {
		f.blitASCII(line, text)
		return
	}
	x := line.X
	lastHead := -1
	g := uniseg.NewGraphemes(text)
	for g.Next() {
		clusterText := g.Str()
		clusterWidth := uniseg.StringWidth(clusterText)
		if clusterWidth == 0 {
			// Zero-width clusters never advance; glue them to the previous
			// head cell so combining marks survive a blit.
			if lastHead >= 0 {
				f.cells[lastHead].Text += clusterText
			}
			continue
		}
		if x >= f.cols {
			break
		}
		if x < 0 {
			x += clusterWidth
			lastHead = -1
			continue
		}
		if x+clusterWidth > f.cols {
			// A wide grapheme cannot be split across the right edge.
			break
		}
		head := line.Y*f.cols + x
		f.clearAt(x, line.Y)
		if clusterWidth > 1 {
			f.clearAt(x+1, line.Y)
			f.cells[head] = Cell{Text: clusterText, Width: clusterWidth, Style: line.Style}
			f.cells[head+1] = Cell{Continuation: true, Style: line.Style}
		} else {
			f.cells[head] = Cell{Text: clusterText, Width: 1, Style: line.Style}
		}
		lastHead = head
		x += clusterWidth
	}
}

func simpleASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] >= 0x7f {
			return false
		}
	}
	return true
}

func (f *Frame) blitASCII(line Line, text string) {
	x := line.X
	for i := 0; i < len(text); i++ {
		if x >= f.cols {
			break
		}
		if x < 0 {
			x++
			continue
		}
		f.clearAt(x, line.Y)
		f.cells[line.Y*f.cols+x] = Cell{Text: text[i : i+1], Width: 1, Style: line.Style}
		x++
	}
}

// BlitFrame copies src into f with its top-left corner at (x, y). Blank src
// cells overwrite (components are opaque), styled blanks keep their style,
// continuation halves are rebuilt at the destination and a wide grapheme that
// falls off any edge is dropped, never split. Out-of-bounds source columns
// are clipped.
func (f *Frame) BlitFrame(src *Frame, x, y int) {
	if f == nil || src == nil {
		return
	}
	for sy := 0; sy < src.rows; sy++ {
		dy := y + sy
		if dy < 0 || dy >= f.rows {
			continue
		}
		for sx := 0; sx < src.cols; sx++ {
			srcCell := src.cells[sy*src.cols+sx]
			if srcCell.Continuation {
				continue
			}
			dx := x + sx
			if dx >= f.cols {
				break
			}
			if dx < 0 {
				// A cell clipped on the left leaves nothing to draw; the
				// visible half of a wide head is still blanked so the layer
				// stays opaque.
				if srcCell.Width > 1 && dx+1 >= 0 && dx+1 < f.cols {
					f.clearAt(dx+1, dy)
				}
				continue
			}
			if srcCell.Text == "" && srcCell.Style == "" {
				f.clearAt(dx, dy)
				continue
			}
			f.clearAt(dx, dy)
			if srcCell.Width > 1 {
				if dx+1 >= f.cols {
					continue
				}
				f.clearAt(dx+1, dy)
				f.cells[dy*f.cols+dx] = Cell{Text: srcCell.Text, Width: srcCell.Width, Style: srcCell.Style}
				f.cells[dy*f.cols+dx+1] = Cell{Continuation: true, Style: srcCell.Style}
				continue
			}
			f.cells[dy*f.cols+dx] = Cell{Text: srcCell.Text, Width: 1, Style: srcCell.Style}
		}
	}
}

// clearAt empties one cell, cleaning up the partner cell of a wide grapheme:
// a continuation clears its head, a head clears its continuation.
func (f *Frame) clearAt(x, y int) {
	idx := y*f.cols + x
	if f.cells[idx].Continuation && x > 0 {
		f.cells[idx-1] = Cell{}
	}
	if f.cells[idx].Width > 1 && x+1 < f.cols {
		f.cells[idx+1] = Cell{}
	}
	f.cells[idx] = Cell{}
}

// Equal reports whether two frames have identical dimensions, cells and
// cursor state.
func (f *Frame) Equal(o *Frame) bool {
	if f == nil || o == nil {
		return f == o
	}
	if f.cols != o.cols || f.rows != o.rows {
		return false
	}
	if f.cursorX != o.cursorX || f.cursorY != o.cursorY || f.cursorVisible != o.cursorVisible || f.cursorShape != o.cursorShape {
		return false
	}
	for i := range f.cells {
		if f.cells[i] != o.cells[i] {
			return false
		}
	}
	return true
}

// Bytes returns the minimal ANSI update that turns prev into f. A nil prev
// (or a size change) yields a full repaint. When nothing changed — content,
// styles and cursor alike — it returns nil.
func (f *Frame) Bytes(prev *Frame) []byte {
	return f.bytes(prev, false, nil, true)
}

// BytesWithoutPhysicalScroll emits a regular cell diff and never changes the
// terminal's scroll region. This is used for live PTY applications whose own
// full-screen redraw can resemble a translated viewport; moving the host
// terminal in that case would apply the same update twice.
func (f *Frame) BytesWithoutPhysicalScroll(prev *Frame) []byte {
	return f.bytes(prev, false, nil, false)
}

// BytesWithoutPhysicalScrollSynchronized emits the live-terminal diff inside
// one DEC synchronized-output transaction. Live PTY applications redraw many
// rows at once; keeping the host's diff atomic prevents the terminal emulator
// from displaying each row while the redraw is still being written. It still
// never changes the terminal's scroll region.
func (f *Frame) BytesWithoutPhysicalScrollSynchronized(prev *Frame) []byte {
	return wrapSynchronizedOutput(f.BytesWithoutPhysicalScroll(prev))
}

// BytesWithSynchronizedScroll returns the same minimal update as Bytes, but
// wraps a physical scroll-region optimization in DEC synchronized output.
// This keeps a history viewport from briefly showing the intermediate state
// after CSI S/T and before its newly exposed row is painted.
func (f *Frame) BytesWithSynchronizedScroll(prev *Frame) []byte {
	return f.bytes(prev, true, nil, true)
}

// BytesWithSynchronizedScrollRegions emits a diff that understands scrolling
// inside individual panel content rectangles. A partial-width panel is
// rewritten only within its content rect; a full-width rect uses the terminal
// scroll region and repaints its exposed row. Both paths are emitted as one
// synchronized update so no intermediate viewport is visible.
func (f *Frame) BytesWithSynchronizedScrollRegions(prev *Frame, regions []ScrollRect) []byte {
	return f.bytes(prev, true, regions, true)
}

type scrollRegionPlan struct {
	rect     ScrollRect
	dir      int // -1: current rows are previous rows shifted up; +1: down
	distance int
	physical bool
}

func (f *Frame) bytes(prev *Frame, synchronizeScroll bool, regions []ScrollRect, allowPhysicalScroll bool) []byte {
	if f == nil {
		return nil
	}
	if prev == nil || prev.cols != f.cols || prev.rows != f.rows {
		return f.fullBytes()
	}
	plans := []scrollRegionPlan(nil)
	skipped := []bool(nil)
	if allowPhysicalScroll {
		plans, skipped = f.scrollPlans(prev, regions)
	}
	var b []byte
	changed := false
	synchronizedScroll := len(plans) > 0 && synchronizeScroll
	if synchronizedScroll {
		b = append(b, BeginSynchronizedOutput...)
	}
	firstDiff, lastDiff := -1, -1
	for y := 0; y < f.rows; y++ {
		if f.rowChangedOutside(prev, y, skipped) {
			if firstDiff < 0 {
				firstDiff = y
			}
			lastDiff = y
		}
	}
	if firstDiff >= 0 {
		if allowPhysicalScroll && len(plans) == 0 && lastDiff-firstDiff >= 2 {
			if start, end, ok := f.shiftedUpWindow(prev); ok {
				if synchronizeScroll {
					b = append(b, BeginSynchronizedOutput...)
					synchronizedScroll = true
				}
				b = append(b, HideCursor...)
				b = append(b, ScrollRegion(start, end)...)
				b = append(b, CursorPosition(0, end)...)
				b = append(b, ScrollUp...)
				b = append(b, ResetScrollRegion...)
				b = append(b, CursorPosition(0, end)...)
				b = append(b, f.rowBytes(end)...)
				changed = true
			} else if start, end, ok := f.shiftedDownWindow(prev); ok {
				if synchronizeScroll {
					b = append(b, BeginSynchronizedOutput...)
					synchronizedScroll = true
				}
				b = append(b, HideCursor...)
				b = append(b, ScrollRegion(start, end)...)
				b = append(b, CursorPosition(0, start)...)
				b = append(b, ScrollDown...)
				b = append(b, ResetScrollRegion...)
				b = append(b, CursorPosition(0, start)...)
				b = append(b, f.rowBytes(start)...)
				changed = true
			}
		}
		if !changed {
			b = append(b, HideCursor...)
			changed = true
			for y := firstDiff; y <= lastDiff; y++ {
				if !f.rowChangedOutside(prev, y, skipped) {
					continue
				}
				b = append(b, f.rowDiffBytes(prev, y, skipped)...)
			}
		}
	}
	for _, plan := range plans {
		if plan.physical {
			if !changed {
				b = append(b, HideCursor...)
				changed = true
			}
			start, end := plan.rect.Y, plan.rect.Y+plan.rect.Height-1
			if plan.dir < 0 {
				b = append(b, ScrollRegion(start, end)...)
				b = append(b, CursorPosition(0, end)...)
				b = append(b, ScrollUpN(plan.distance)...)
				b = append(b, ResetScrollRegion...)
				b = append(b, CursorPosition(0, end)...)
				b = append(b, f.rowBytes(end)...)
			} else {
				b = append(b, ScrollRegion(start, end)...)
				b = append(b, CursorPosition(0, start)...)
				b = append(b, ScrollDownN(plan.distance)...)
				b = append(b, ResetScrollRegion...)
				b = append(b, CursorPosition(0, start)...)
				b = append(b, f.rowBytes(start)...)
			}
			continue
		}
		if !changed {
			b = append(b, HideCursor...)
			changed = true
		}
		for y := plan.rect.Y; y < plan.rect.Y+plan.rect.Height; y++ {
			b = append(b, CursorPosition(plan.rect.X, y)...)
			b = append(b, f.rowBytesRange(y, plan.rect.X, plan.rect.X+plan.rect.Width)...)
		}
	}
	cursorMoved := f.cursorVisible != prev.cursorVisible ||
		(f.cursorVisible && (f.cursorX != prev.cursorX || f.cursorY != prev.cursorY))
	cursorShapeChanged := f.cursorVisible && f.cursorShape != prev.cursorShape
	if !changed && !cursorMoved && !cursorShapeChanged {
		return nil
	}
	if f.cursorVisible {
		b = append(b, cursorShapeBytes(f.cursorShape)...)
		b = append(b, CursorPosition(f.cursorX, f.cursorY)...)
		b = append(b, ShowCursor...)
	} else if prev.cursorVisible && !changed {
		b = append(b, HideCursor...)
	}
	if synchronizedScroll {
		b = append(b, EndSynchronizedOutput...)
	}
	return b
}

func wrapSynchronizedOutput(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	b := make([]byte, 0, len(BeginSynchronizedOutput)+len(data)+len(EndSynchronizedOutput))
	b = append(b, BeginSynchronizedOutput...)
	b = append(b, data...)
	b = append(b, EndSynchronizedOutput...)
	return b
}

func (f *Frame) scrollPlans(prev *Frame, regions []ScrollRect) ([]scrollRegionPlan, []bool) {
	if len(regions) == 0 {
		return nil, nil
	}
	skipped := make([]bool, f.cols*f.rows)
	plans := make([]scrollRegionPlan, 0, len(regions))
	for _, raw := range regions {
		r := raw
		if r.X < 0 {
			r.Width += r.X
			r.X = 0
		}
		if r.Y < 0 {
			r.Height += r.Y
			r.Y = 0
		}
		if r.X+r.Width > f.cols {
			r.Width = f.cols - r.X
		}
		if r.Y+r.Height > f.rows {
			r.Height = f.rows - r.Y
		}
		if r.Width <= 0 || r.Height < 2 {
			continue
		}
		dir, distance, ok := f.regionShift(prev, r)
		if !ok {
			continue
		}
		plans = append(plans, scrollRegionPlan{
			rect:     r,
			dir:      dir,
			distance: distance,
			physical: r.X == 0 && r.Width == f.cols,
		})
		for y := r.Y; y < r.Y+r.Height; y++ {
			for x := r.X; x < r.X+r.Width; x++ {
				skipped[y*f.cols+x] = true
			}
		}
	}
	return plans, skipped
}

// regionShift returns the direction and distance when the rows in r are a
// vertical translation of the previous frame. It matches the overlap and
// verifies that both newly exposed edges changed, allowing wheel-sized and
// page-sized history movement without mistaking repeated rows for a shift.
func (f *Frame) regionShift(prev *Frame, r ScrollRect) (int, int, bool) {
	same := true
	for y := r.Y; y < r.Y+r.Height && same; y++ {
		for x := r.X; x < r.X+r.Width; x++ {
			if f.cells[y*f.cols+x] != prev.cells[y*prev.cols+x] {
				same = false
				break
			}
		}
	}
	if same {
		return 0, 0, false
	}
	for _, dir := range []int{-1, 1} {
		for distance := 1; distance < r.Height; distance++ {
			// The overlap alone is not enough evidence of a scroll: repeated
			// rows (blank/grey rows are common in full-screen TUIs) can make a
			// one-row append look exactly like a translated viewport. Require
			// both exposed edges to change before emitting CSI S/T. If either
			// edge is unchanged, fall back to ordinary cell diffs; that is a
			// little more work but cannot move the terminal viewport twice.
			if f.cellsEqualRange(prev, r.Y, f, r.Y, r.X, r.X+r.Width) ||
				f.cellsEqualRange(prev, r.Y+r.Height-1, f, r.Y+r.Height-1, r.X, r.X+r.Width) {
				continue
			}
			matched := true
			start, end := 0, r.Height-distance-1
			if dir > 0 {
				start = distance
				end = r.Height - 1
			}
			for y := start; y <= end; y++ {
				cy := r.Y + y
				py := cy - dir*distance
				if !f.cellsEqualRange(prev, py, f, cy, r.X, r.X+r.Width) {
					matched = false
					break
				}
			}
			if matched {
				return dir, distance, true
			}
		}
	}
	return 0, 0, false
}

func (f *Frame) cellsEqualRange(a *Frame, ay int, b *Frame, by, x0, x1 int) bool {
	if ay < 0 || by < 0 || ay >= a.rows || by >= b.rows {
		return false
	}
	for x := x0; x < x1; x++ {
		if a.cells[ay*a.cols+x] != b.cells[by*b.cols+x] {
			return false
		}
	}
	return true
}

func (f *Frame) rowChangedOutside(prev *Frame, y int, skipped []bool) bool {
	if len(skipped) == 0 {
		return !f.rowEqual(prev, y)
	}
	start := y * f.cols
	for x := 0; x < f.cols; x++ {
		if skipped[start+x] {
			continue
		}
		if f.cells[start+x] != prev.cells[start+x] {
			return true
		}
	}
	return false
}

func (f *Frame) rowDiffBytes(prev *Frame, y int, skipped []bool) []byte {
	if len(skipped) == 0 {
		var out []byte
		out = append(out, CursorPosition(0, y)...)
		return append(out, f.rowBytes(y)...)
	}
	var out []byte
	start := y * f.cols
	for x := 0; x < f.cols; {
		for x < f.cols && skipped[start+x] {
			x++
		}
		seg := x
		for x < f.cols && !skipped[start+x] {
			x++
		}
		if seg == x {
			continue
		}
		changed := false
		for i := seg; i < x; i++ {
			if f.cells[start+i] != prev.cells[start+i] {
				changed = true
				break
			}
		}
		if changed {
			out = append(out, CursorPosition(seg, y)...)
			out = append(out, f.rowBytesRange(y, seg, x)...)
		}
	}
	return out
}

// shiftedUpWindow reports a one-row upward shift inside [start,end]. Rows
// outside the window must remain identical; the caller can therefore use the
// terminal's scroll region and repaint only the newly exposed row.
func (f *Frame) shiftedUpWindow(prev *Frame) (start, end int, ok bool) {
	start = 0
	for start < f.rows && f.rowEqual(prev, start) {
		start++
	}
	end = f.rows - 1
	for end >= start && f.rowEqual(prev, end) {
		end--
	}
	if end-start < 2 {
		return 0, 0, false
	}
	for y := start; y < end; y++ {
		if !f.rowEqualAtFrame(prev, y+1, f, y) {
			return 0, 0, false
		}
	}
	return start, end, true
}

// shiftedDownWindow is the inverse of shiftedUpWindow.
func (f *Frame) shiftedDownWindow(prev *Frame) (start, end int, ok bool) {
	start = 0
	for start < f.rows && f.rowEqual(prev, start) {
		start++
	}
	end = f.rows - 1
	for end >= start && f.rowEqual(prev, end) {
		end--
	}
	if end-start < 2 {
		return 0, 0, false
	}
	for y := start + 1; y <= end; y++ {
		if !f.rowEqualAtFrame(prev, y-1, f, y) {
			return 0, 0, false
		}
	}
	return start, end, true
}

func (f *Frame) rowEqualAtFrame(a *Frame, ay int, b *Frame, by int) bool {
	if a == nil || b == nil || ay < 0 || by < 0 || ay >= a.rows || by >= b.rows || a.cols != b.cols {
		return false
	}
	aStart, bStart := ay*a.cols, by*b.cols
	for x := 0; x < a.cols; x++ {
		if a.cells[aStart+x] != b.cells[bStart+x] {
			return false
		}
	}
	return true
}

// FullBytes returns an unconditional full repaint.
func (f *Frame) FullBytes() []byte {
	if f == nil {
		return nil
	}
	return f.fullBytes()
}

func (f *Frame) fullBytes() []byte {
	var b []byte
	b = append(b, HideCursor...)
	b = append(b, ResetSGR...)
	for y := 0; y < f.rows; y++ {
		b = append(b, CursorPosition(0, y)...)
		b = append(b, f.rowBytes(y)...)
	}
	if f.cursorVisible {
		b = append(b, cursorShapeBytes(f.cursorShape)...)
		b = append(b, CursorPosition(f.cursorX, f.cursorY)...)
		b = append(b, ShowCursor...)
	}
	return b
}

func cursorShapeBytes(shape string) []byte {
	switch shape {
	case "block":
		return []byte(CursorShapeBlock)
	case "underline":
		return []byte(CursorShapeUnderline)
	case "bar":
		return []byte(CursorShapeBar)
	default:
		return nil
	}
}

func (f *Frame) rowEqual(o *Frame, y int) bool {
	start := y * f.cols
	for i := start; i < start+f.cols; i++ {
		if f.cells[i] != o.cells[i] {
			return false
		}
	}
	return true
}

// rowBytes renders one row as SGR runs. Every style change starts from a
// reset so the output is independent of what the TTY had before; blank cells
// are emitted as spaces because the cursor may sit anywhere.
func (f *Frame) rowBytes(y int) []byte {
	return f.rowBytesRange(y, 0, f.cols)
}

func (f *Frame) rowBytesRange(y, x0, x1 int) []byte {
	var b bytes.Buffer
	current := Token("\x00unset")
	if x0 < 0 {
		x0 = 0
	}
	if x1 > f.cols {
		x1 = f.cols
	}
	if x1 < x0 {
		x1 = x0
	}
	start := y * f.cols
	for i := start + x0; i < start+x1; i++ {
		cell := f.cells[i]
		if cell.Continuation {
			continue
		}
		style := cell.Style
		if style == "" {
			style = TokenDefault
		}
		if style != current {
			b.WriteString(ResetSGR)
			b.WriteString(style.SGR(f.theme))
			current = style
		}
		if cell.Text == "" {
			b.WriteByte(' ')
		} else {
			b.WriteString(cell.Text)
		}
	}
	b.WriteString(ResetSGR)
	return b.Bytes()
}

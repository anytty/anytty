package vt

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/exp/ordered"
)

// Screen represents a virtual terminal screen.
type Screen struct {
	// cb is the callbacks struct to use.
	cb *Callbacks
	// damage records screen mutations for the current write batch.
	damage *screenDamageRecorder
	// The buffer of the screen.
	buf *uv.RenderBuffer
	// The cur of the screen.
	cur, saved Cursor
	// scroll is the scroll region.
	scroll uv.Rectangle
	// scrollback is the scrollback buffer for lines scrolled off the top.
	scrollback *Scrollback
	// wrapped marks rows that visually continue onto the next row due to
	// automatic terminal wrapping.
	wrapped []bool
	// used stores the logical cell count that has actually been written for
	// each row. This is separate from the fixed render-buffer width so trailing
	// spaces printed by a program survive later reflow.
	used []int
}

type pendingScrollbackDamage struct {
	y       int
	line    uv.Line
	wrapped bool
}

// NewScreen creates a new screen.
func NewScreen(w, h int) *Screen {
	s := Screen{
		buf:        uv.NewRenderBuffer(w, h),
		scrollback: NewScrollback(DefaultScrollbackSize),
		wrapped:    make([]bool, h),
		used:       make([]int, h),
	}
	s.scroll = s.buf.Bounds()
	return &s
}

// Reset resets the screen.
// It clears the screen, sets the cursor to the top left corner, reset the
// cursor styles, and resets the scroll region.
func (s *Screen) Reset() {
	s.buf.Clear()
	s.cur = Cursor{}
	s.saved = Cursor{}
	s.scroll = s.buf.Bounds()
	s.wrapped = make([]bool, s.buf.Height())
	s.used = make([]int, s.buf.Height())
	s.buf.Touched = nil
	s.recordDamage(ScreenDamage{Width: s.buf.Width(), Height: s.buf.Height()})
}

// Bounds returns the bounds of the screen.
func (s *Screen) Bounds() uv.Rectangle {
	return s.buf.Bounds()
}

// Touched returns touched lines in the screen buffer.
func (s *Screen) Touched() []*uv.LineData {
	return s.buf.Touched
}

// ClearTouched clears the touched state.
func (s *Screen) ClearTouched() {
	s.buf.Touched = nil
}

// CellAt returns the cell at the given x, y position.
func (s *Screen) CellAt(x int, y int) *uv.Cell {
	return s.buf.CellAt(x, y)
}

// Line returns the current row truncated to its logical used width. The
// returned slice aliases the render buffer and is invalidated by mutation.
func (s *Screen) Line(y int) uv.Line {
	if s == nil || s.buf == nil || y < 0 || y >= s.buf.Height() {
		return nil
	}
	line := s.buf.Line(y)
	if line == nil {
		return nil
	}
	used := s.LineUsed(y)
	if used > len(line) {
		used = len(line)
	}
	return line[:used]
}

// SetCell sets the cell at the given x, y position.
func (s *Screen) SetCell(x, y int, c *uv.Cell) {
	s.markUsed(y, x+cellColumnWidth(c))
	// On the common write path we only need row-level dirtiness. Bypass the
	// RenderBuffer equality check so high-volume redraws don't pay a
	// cell-by-cell compare before every mutation.
	if s.damage == nil {
		width := 1
		if c != nil && c.Width > 1 {
			width = c.Width
		}
		s.buf.TouchLine(x, y, width)
		s.buf.Buffer.SetCell(x, y, c)
		return
	}
	s.buf.SetCell(x, y, c)
	s.recordCellDamage(x, y, c)
}

func (s *Screen) SetASCIICells(x, y int, data []byte, style uv.Style, link uv.Link) {
	if s == nil || s.buf == nil || len(data) == 0 || y < 0 || y >= s.buf.Height() || x >= s.buf.Width() {
		return
	}
	if x < 0 {
		skip := min(len(data), -x)
		data = data[skip:]
		x = 0
	}
	if len(data) == 0 {
		return
	}
	if maxCells := s.buf.Width() - x; len(data) > maxCells {
		data = data[:maxCells]
	}
	s.markUsed(y, x+len(data))
	line := s.buf.Line(y)
	if line == nil {
		for i, b := range data {
			s.SetCell(x+i, y, &uv.Cell{
				Content: printableASCIIStrings[b],
				Width:   1,
				Style:   style,
				Link:    link,
			})
		}
		return
	}
	for i, b := range data {
		line[x+i] = uv.Cell{
			Content: printableASCIIStrings[b],
			Width:   1,
			Style:   style,
			Link:    link,
		}
	}
	s.buf.TouchLine(x, y, len(data))
	if s.damage != nil && s.damage.recordsScreenDiff() {
		cells := make([]uv.Cell, len(data))
		copy(cells, line[x:x+len(data)])
		s.damage.record(SpanDamage{X: x, Y: y, Cells: cells})
	}
}

// Height returns the height of the screen.
func (s *Screen) Height() int {
	return s.buf.Height()
}

// Resize resizes the screen.
func (s *Screen) Resize(width int, height int) {
	if s.buf == nil {
		s.buf = uv.NewRenderBuffer(width, height)
	} else {
		s.buf.Resize(width, height)
		s.buf.Touched = nil
	}
	s.wrapped = resizeBoolSlice(s.wrapped, height)
	s.used = resizeIntSlice(s.used, height)
	s.scroll = s.buf.Bounds()
	s.recordDamage(ScreenDamage{Width: width, Height: height})
}

// Width returns the width of the screen.
func (s *Screen) Width() int {
	return s.buf.Width()
}

// Clear clears the screen with blank cells.
func (s *Screen) Clear() {
	s.ClearArea(s.Bounds())
	clear(s.used)
}

// ClearWithScrollback saves all non-empty lines to scrollback before clearing.
// This is used for operations like ED 2 (erase screen) where content should
// be preserved in history. The blank cell is used to fill the cleared area;
// pass nil to clear to empty cells, or blankCell() to preserve the current
// pen background color.
func (s *Screen) ClearWithScrollback(blank *uv.Cell) []ScrollbackDamage {
	// 中文说明：damage 记录是 core-v2 history 的语义来源，不能因为本地
	// emulator scrollback 被禁用而丢掉 ED2 清屏时离开可见区的行。
	var capturedRows []ScrollbackDamage
	for y := 0; y < s.buf.Height(); y++ {
		line := s.buf.Line(y)
		if line == nil {
			continue
		}
		used := min(len(line), s.LineUsed(y))
		if used <= 0 {
			continue
		}
		captured := line[:used]
		if !s.isLineEmpty(captured) {
			capturedRows = append(capturedRows, s.scrollbackDamageForLine(y, captured, s.LineWrapped(y)))
		}
	}
	if s.scrollback != nil {
		// Save all lines that have content before clearing
		for y := 0; y < s.buf.Height(); y++ {
			line := s.buf.Line(y)
			if line == nil {
				continue
			}
			used := min(len(line), s.LineUsed(y))
			if used <= 0 {
				continue
			}
			captured := line[:used]
			if !s.isLineEmpty(captured) {
				s.scrollback.PushWrapped(captured, s.LineWrapped(y))
			}
		}
	}
	s.FillArea(blank, s.Bounds())
	clear(s.wrapped)
	clear(s.used)
	return capturedRows
}

// isLineEmpty returns true if the line contains only empty/space cells.
func (s *Screen) isLineEmpty(line uv.Line) bool {
	for _, cell := range line {
		if cell.Width != 0 && cell.Width != 1 {
			return false
		}
		if !cell.Style.IsZero() || !cell.Link.IsZero() {
			return false
		}
		if cell.Content != "" && cell.Content != " " {
			return false
		}
	}
	return true
}

// ClearArea clears the given area.
func (s *Screen) ClearArea(area uv.Rectangle) {
	s.buf.ClearArea(area)
	if area.Min.X <= 0 && area.Max.X >= s.buf.Width() {
		for y := max(0, area.Min.Y); y < min(area.Max.Y, len(s.wrapped)); y++ {
			s.wrapped[y] = false
			s.setLineUsed(y, 0)
		}
	}
	s.touchArea(area)
	s.recordFillDamage(nil, area)
}

// Fill fills the screen or part of it.
func (s *Screen) Fill(c *uv.Cell) {
	s.FillArea(c, s.Bounds())
}

// FillArea fills the given area with the given cell.
func (s *Screen) FillArea(c *uv.Cell, area uv.Rectangle) {
	s.buf.FillArea(c, area)
	if area.Min.X <= 0 && area.Max.X >= s.buf.Width() {
		for y := max(0, area.Min.Y); y < min(area.Max.Y, len(s.used)); y++ {
			if c == nil {
				s.setLineUsed(y, 0)
			} else {
				s.setLineUsed(y, s.buf.Width())
			}
		}
	} else if c != nil {
		for y := max(0, area.Min.Y); y < min(area.Max.Y, len(s.used)); y++ {
			s.markUsed(y, area.Max.X)
		}
	}
	s.touchArea(area)
	s.recordFillDamage(c, area)
}

// setHorizontalMargins sets the horizontal margins.
func (s *Screen) setHorizontalMargins(left, right int) {
	s.scroll.Min.X = left
	s.scroll.Max.X = right
}

// setVerticalMargins sets the vertical margins.
func (s *Screen) setVerticalMargins(top, bottom int) {
	s.scroll.Min.Y = top
	s.scroll.Max.Y = bottom
}

// setCursorX sets the cursor X position. If margins is true, the cursor is
// only set if it is within the scroll margins.
func (s *Screen) setCursorX(x int, margins bool) {
	s.setCursor(x, s.cur.Y, margins)
}

// setCursor sets the cursor position. If margins is true, the cursor is only
// set if it is within the scroll margins. This follows how [ansi.CUP] works.
func (s *Screen) setCursor(x, y int, margins bool) {
	old := s.cur.Position
	if !margins {
		y = ordered.Clamp(y, 0, s.buf.Height()-1)
		x = ordered.Clamp(x, 0, s.buf.Width()-1)
	} else {
		y = ordered.Clamp(s.scroll.Min.Y+y, s.scroll.Min.Y, s.scroll.Max.Y-1)
		x = ordered.Clamp(s.scroll.Min.X+x, s.scroll.Min.X, s.scroll.Max.X-1)
	}
	s.cur.X, s.cur.Y = x, y

	if s.cb.CursorPosition != nil && (old.X != x || old.Y != y) {
		s.cb.CursorPosition(old, uv.Pos(x, y))
	}
}

// moveCursor moves the cursor by the given x and y deltas. If the cursor
// position is inside the scroll region, it is bounded by the scroll region.
// Otherwise, it is bounded by the screen bounds.
// This follows how [ansi.CUU], [ansi.CUD], [ansi.CUF], [ansi.CUB], [ansi.CNL],
// [ansi.CPL].
func (s *Screen) moveCursor(dx, dy int) {
	scroll := s.scroll
	old := s.cur.Position
	if old.X < scroll.Min.X {
		scroll.Min.X = 0
	}
	if old.X >= scroll.Max.X {
		scroll.Max.X = s.buf.Width()
	}

	pt := uv.Pos(s.cur.X+dx, s.cur.Y+dy)

	var x, y int
	if old.In(scroll) {
		y = ordered.Clamp(pt.Y, scroll.Min.Y, scroll.Max.Y-1)
		x = ordered.Clamp(pt.X, scroll.Min.X, scroll.Max.X-1)
	} else {
		y = ordered.Clamp(pt.Y, 0, s.buf.Height()-1)
		x = ordered.Clamp(pt.X, 0, s.buf.Width()-1)
	}

	s.cur.X, s.cur.Y = x, y

	if s.cb.CursorPosition != nil && (old.X != x || old.Y != y) {
		s.cb.CursorPosition(old, uv.Pos(x, y))
	}
}

// Cursor returns the cursor.
func (s *Screen) Cursor() Cursor {
	return s.cur
}

// CursorPosition returns the cursor position.
func (s *Screen) CursorPosition() (x, y int) {
	return s.cur.X, s.cur.Y
}

// ScrollRegion returns the scroll region.
func (s *Screen) ScrollRegion() uv.Rectangle {
	return s.scroll
}

// SaveCursor saves the cursor.
func (s *Screen) SaveCursor() {
	s.saved = s.cur
}

// RestoreCursor restores the cursor.
func (s *Screen) RestoreCursor() {
	old := s.cur.Position
	s.cur = s.saved

	if s.cb.CursorPosition != nil && (old.X != s.cur.X || old.Y != s.cur.Y) {
		s.cb.CursorPosition(old, s.cur.Position)
	}
}

// setCursorHidden sets the cursor hidden.
func (s *Screen) setCursorHidden(hidden bool) {
	changed := s.cur.Hidden != hidden
	s.cur.Hidden = hidden
	if changed && s.cb.CursorVisibility != nil {
		s.cb.CursorVisibility(!hidden)
	}
}

// setCursorStyle sets the cursor style.
func (s *Screen) setCursorStyle(style CursorStyle, blink bool) {
	changed := s.cur.Style != style || s.cur.Steady != !blink
	s.cur.Style = style
	s.cur.Steady = !blink
	if changed && s.cb.CursorStyle != nil {
		s.cb.CursorStyle(style, !blink)
	}
}

// cursorPen returns the cursor pen.
func (s *Screen) cursorPen() uv.Style {
	return s.cur.Pen
}

// cursorLink returns the cursor link.
func (s *Screen) cursorLink() uv.Link {
	return s.cur.Link
}

// ShowCursor shows the cursor.
func (s *Screen) ShowCursor() {
	s.setCursorHidden(false)
}

// HideCursor hides the cursor.
func (s *Screen) HideCursor() {
	s.setCursorHidden(true)
}

// InsertCell inserts n blank characters at the cursor position pushing out
// cells to the right and out of the screen.
func (s *Screen) InsertCell(n int) {
	if n <= 0 {
		return
	}

	x, y := s.cur.X, s.cur.Y
	fill := s.blankCell()
	limit := min(s.scroll.Max.X, s.buf.Width())
	if x+n > limit {
		n = limit - x
	}
	if n <= 0 {
		return
	}
	s.buf.InsertCellArea(x, y, n, fill, s.scroll)
	s.markUsed(y, min(limit, s.LineUsed(y)+n))
	rectWidth := limit - x
	if rectWidth > 0 {
		s.recordDamage(ScrollDamage{
			Rectangle: uv.Rect(x, y, rectWidth, 1),
			Dx:        n,
		})
		s.recordFillDamage(fill, uv.Rect(x, y, n, 1))
	}
}

// DeleteCell deletes n cells at the cursor position moving cells to the left.
// This has no effect if the cursor is outside the scroll region.
func (s *Screen) DeleteCell(n int) {
	if n <= 0 {
		return
	}

	x, y := s.cur.X, s.cur.Y
	fill := s.blankCell()
	limit := min(s.scroll.Max.X, s.buf.Width())
	if x+n > limit {
		n = limit - x
	}
	if n <= 0 {
		return
	}
	s.buf.DeleteCellArea(x, y, n, fill, s.scroll)
	used := s.LineUsed(y)
	if x < used {
		s.setLineUsed(y, max(x, used-n))
	}
	rectWidth := limit - x
	if rectWidth > 0 {
		s.recordDamage(ScrollDamage{
			Rectangle: uv.Rect(x, y, rectWidth, 1),
			Dx:        -n,
		})
		s.recordFillDamage(fill, uv.Rect(limit-n, y, n, 1))
	}
}

// ScrollUp scrolls the content up n lines within the given region. Lines
// scrolled past the top margin are lost. This is equivalent to [ansi.SU] which
// moves the cursor to the top margin and performs a [ansi.DL] operation.
func (s *Screen) ScrollUp(n int) {
	x, y := s.CursorPosition()
	s.setCursor(s.cur.X, 0, true)
	s.DeleteLine(n)
	s.setCursor(x, y, false)
}

// ScrollDown scrolls the content down n lines within the given region. Lines
// scrolled past the bottom margin are lost. This is equivalent to [ansi.SD]
// which moves the cursor to top margin and performs a [ansi.IL] operation.
func (s *Screen) ScrollDown(n int) {
	x, y := s.CursorPosition()
	s.setCursor(s.cur.X, 0, true)
	s.InsertLine(n)
	s.setCursor(x, y, false)
}

// InsertLine inserts n blank lines at the cursor position Y coordinate.
// Only operates if cursor is within scroll region. Lines below cursor Y
// are moved down, with those past bottom margin being discarded.
// It returns true if the operation was successful.
func (s *Screen) InsertLine(n int) bool {
	if n <= 0 {
		return false
	}

	x, y := s.cur.X, s.cur.Y

	// Only operate if cursor Y is within scroll region
	if y < s.scroll.Min.Y || y >= s.scroll.Max.Y ||
		x < s.scroll.Min.X || x >= s.scroll.Max.X {
		return false
	}

	fill := s.blankCell()
	if y+n > s.scroll.Max.Y {
		n = s.scroll.Max.Y - y
	}
	if n <= 0 {
		return false
	}
	s.buf.InsertLineArea(y, n, fill, s.scroll)
	s.insertWrappedLines(y, n)
	s.insertUsedLines(y, n)
	rect := uv.Rect(s.scroll.Min.X, y, s.scroll.Dx(), s.scroll.Max.Y-y)
	s.recordDamage(ScrollDamage{Rectangle: rect, Dy: n})
	s.recordFillDamage(fill, uv.Rect(s.scroll.Min.X, y, s.scroll.Dx(), n))

	return true
}

// DeleteLine deletes n lines at the cursor position Y coordinate.
// Only operates if cursor is within scroll region. Lines below cursor Y
// are moved up, with blank lines inserted at the bottom of scroll region.
// If scrollback is enabled and cursor is at top of scroll region, lines
// are saved to the scrollback buffer before deletion.
// It returns true if the operation was successful.
func (s *Screen) DeleteLine(n int) bool {
	if n <= 0 {
		return false
	}

	scroll := s.scroll
	x, y := s.cur.X, s.cur.Y

	// Only operate if cursor Y is within scroll region
	if y < scroll.Min.Y || y >= scroll.Max.Y ||
		x < scroll.Min.X || x >= scroll.Max.X {
		return false
	}

	fill := s.blankCell()
	if n > scroll.Max.Y-y {
		n = scroll.Max.Y - y
	}
	if n <= 0 {
		return false
	}
	var pendingScrollback []pendingScrollbackDamage
	// Save lines to scrollback if we're at the top of the scroll region
	// and the scroll region uses the full width (typical terminal scroll).
	// This captures lines that would be lost during scroll up operations.
	if y == scroll.Min.Y && scroll.Min.X == 0 && scroll.Max.X == s.buf.Width() {
		// Save lines that will be deleted
		linesToSave := min(n, scroll.Max.Y-y)
		if s.damage != nil && (s.damage.scrollbackOnly || s.damage.lineHistoryOnly) {
			for i := range min(linesToSave, s.buf.Height()-y) {
				if line := s.buf.Line(y + i); line != nil {
					s.recordScrollbackLine(y+i, line[:min(len(line), s.LineUsed(y+i))], boolAt(s.wrapped, y+i))
				}
			}
		} else if s.damage != nil {
			pendingScrollback = make([]pendingScrollbackDamage, 0, min(linesToSave, s.buf.Height()-y))
			for i := range min(linesToSave, s.buf.Height()-y) {
				if line := s.buf.Line(y + i); line != nil {
					used := min(len(line), s.LineUsed(y+i))
					// 中文说明：空行 used=0 时 append(nil, ...) 会得到 nil slice，
					// recordScrollbackLine 会把 nil 当"无行"丢弃，导致滚出的空白行
					// 从 scrollback damage 序列消失。用 make 保证空行也是非 nil 捕获。
					captured := make(uv.Line, used)
					copy(captured, line[:used])
					pendingScrollback = append(pendingScrollback, pendingScrollbackDamage{
						y:       y + i,
						line:    captured,
						wrapped: boolAt(s.wrapped, y+i),
					})
				}
			}
		}
		if s.scrollback != nil {
			s.scrollback.PushN(s.buf, s.wrapped, s.used, y, linesToSave)
		}
	}
	if scroll.Min.X == 0 && scroll.Max.X == s.buf.Width() {
		deleteFullWidthLines(s.buf, y, n, scroll, fill)
	} else {
		s.buf.DeleteLineArea(y, n, fill, scroll)
	}
	s.deleteWrappedLines(y, n)
	s.deleteUsedLines(y, n)
	rect := uv.Rect(scroll.Min.X, y, scroll.Dx(), scroll.Max.Y-y)
	s.recordDamage(ScrollDamage{Rectangle: rect, Dy: -n})
	s.recordFillDamage(fill, uv.Rect(scroll.Min.X, scroll.Max.Y-n, scroll.Dx(), n))
	if len(pendingScrollback) > 0 {
		for _, row := range pendingScrollback {
			s.recordScrollbackLine(row.y, row.line, row.wrapped)
		}
	}

	return true
}

func deleteFullWidthLines(buf *uv.RenderBuffer, y, n int, scroll uv.Rectangle, fill *uv.Cell) {
	if buf == nil || n <= 0 || y < scroll.Min.Y || y >= scroll.Max.Y {
		return
	}
	if n > scroll.Max.Y-y {
		n = scroll.Max.Y - y
	}
	if n <= 0 {
		return
	}
	lines := buf.Lines
	if len(lines) == 0 {
		return
	}
	width := buf.Width()
	reuseStart := scroll.Max.Y - n
	var reuse []uv.Line
	if n == 1 {
		reuse = []uv.Line{lines[y]}
	} else {
		reuse = append([]uv.Line(nil), lines[y:y+n]...)
	}
	copy(lines[y:reuseStart], lines[y+n:scroll.Max.Y])
	for i := 0; i < n; i++ {
		line := reuse[i]
		if len(line) != width {
			line = uv.NewLine(width)
		} else {
			fillLine(line, fill)
		}
		lines[reuseStart+i] = line
	}
	for row := y; row < scroll.Max.Y; row++ {
		buf.TouchLine(scroll.Min.X, row, scroll.Dx())
	}
}

func fillLine(line uv.Line, cell *uv.Cell) {
	if cell == nil {
		for i := range line {
			line[i] = uv.EmptyCell
		}
		return
	}
	for i := range line {
		line[i] = *cell
	}
}

// blankCell returns the cursor blank cell with the background color set to the
// current pen background color. If the pen background color is nil, the return
// value is nil.
func (s *Screen) blankCell() *uv.Cell {
	if s.cur.Pen.Bg == nil {
		return nil
	}

	c := uv.EmptyCell
	c.Style.Bg = s.cur.Pen.Bg
	return &c
}

// touchArea marks all lines in the given area as touched.
func (s *Screen) touchArea(area uv.Rectangle) {
	for y := area.Min.Y; y < area.Max.Y; y++ {
		s.buf.TouchLine(area.Min.X, y, area.Max.X-area.Min.X)
	}
}

// Scrollback returns the screen's scrollback buffer.
func (s *Screen) Scrollback() *Scrollback {
	return s.scrollback
}

// SetScrollback sets the screen's scrollback buffer.
// Pass nil to disable scrollback.
func (s *Screen) SetScrollback(sb *Scrollback) {
	s.scrollback = sb
}

// SetScrollbackSize sets the maximum number of lines in the scrollback buffer.
func (s *Screen) SetScrollbackSize(maxLines int) {
	if s.scrollback == nil {
		s.scrollback = NewScrollback(maxLines)
	} else {
		s.scrollback.SetMaxLines(maxLines)
	}
}

// LineWrapped returns whether row y visually continues onto the next row.
func (s *Screen) LineWrapped(y int) bool {
	return boolAt(s.wrapped, y)
}

// SetLineWrapped updates whether row y visually continues onto the next row.
func (s *Screen) SetLineWrapped(y int, wrapped bool) {
	if y < 0 || y >= s.buf.Height() {
		return
	}
	s.ensureWrappedHeight()
	s.wrapped[y] = wrapped
}

// LineUsed returns the logical used column count for a row.
func (s *Screen) LineUsed(y int) int {
	if s == nil || s.buf == nil || y < 0 || y >= s.buf.Height() {
		return 0
	}
	s.ensureUsedHeight()
	if y >= len(s.used) {
		return 0
	}
	return clampLocalInt(s.used[y], 0, s.buf.Width())
}

// SetLineUsed sets the logical used column count for a row.
func (s *Screen) SetLineUsed(y int, used int) {
	s.setLineUsed(y, used)
}

// WrappedLines returns the current screen soft-wrap markers.
func (s *Screen) WrappedLines() []bool {
	s.ensureWrappedHeight()
	return s.wrapped
}

// SetWrappedLines replaces all screen soft-wrap markers.
func (s *Screen) SetWrappedLines(wrapped []bool) {
	s.wrapped = make([]bool, s.buf.Height())
	copy(s.wrapped, wrapped)
}

func (s *Screen) ensureWrappedHeight() {
	height := 0
	if s != nil && s.buf != nil {
		height = s.buf.Height()
	}
	s.wrapped = resizeBoolSlice(s.wrapped, height)
}

func (s *Screen) ensureUsedHeight() {
	height := 0
	if s != nil && s.buf != nil {
		height = s.buf.Height()
	}
	s.used = resizeIntSlice(s.used, height)
}

func (s *Screen) markUsed(y int, used int) {
	if s == nil || s.buf == nil || y < 0 || y >= s.buf.Height() {
		return
	}
	s.ensureUsedHeight()
	if used > s.used[y] {
		s.used[y] = clampLocalInt(used, 0, s.buf.Width())
	}
}

func (s *Screen) setLineUsed(y int, used int) {
	if s == nil || s.buf == nil || y < 0 || y >= s.buf.Height() {
		return
	}
	s.ensureUsedHeight()
	s.used[y] = clampLocalInt(used, 0, s.buf.Width())
}

func (s *Screen) insertWrappedLines(y, n int) {
	if n <= 0 || y < 0 || y >= len(s.wrapped) {
		return
	}
	if y+n > len(s.wrapped) {
		n = len(s.wrapped) - y
	}
	copy(s.wrapped[y+n:], s.wrapped[y:len(s.wrapped)-n])
	clear(s.wrapped[y : y+n])
}

func (s *Screen) deleteWrappedLines(y, n int) {
	if n <= 0 || y < 0 || y >= len(s.wrapped) {
		return
	}
	if y+n > len(s.wrapped) {
		n = len(s.wrapped) - y
	}
	copy(s.wrapped[y:], s.wrapped[y+n:])
	clear(s.wrapped[len(s.wrapped)-n:])
}

func (s *Screen) insertUsedLines(y, n int) {
	if n <= 0 || y < 0 || y >= len(s.used) {
		return
	}
	if y+n > len(s.used) {
		n = len(s.used) - y
	}
	copy(s.used[y+n:], s.used[y:len(s.used)-n])
	clear(s.used[y : y+n])
}

func (s *Screen) deleteUsedLines(y, n int) {
	if n <= 0 || y < 0 || y >= len(s.used) {
		return
	}
	if y+n > len(s.used) {
		n = len(s.used) - y
	}
	copy(s.used[y:], s.used[y+n:])
	clear(s.used[len(s.used)-n:])
}

func resizeBoolSlice(values []bool, size int) []bool {
	if size <= 0 {
		return nil
	}
	previous := len(values)
	if cap(values) < size {
		next := make([]bool, size)
		copy(next, values)
		return next
	}
	values = values[:size]
	if size > previous {
		clear(values[previous:])
	}
	return values
}

func resizeIntSlice(values []int, size int) []int {
	if size <= 0 {
		return nil
	}
	previous := len(values)
	if cap(values) < size {
		next := make([]int, size)
		copy(next, values)
		return next
	}
	values = values[:size]
	if size > previous {
		clear(values[previous:])
	}
	return values
}

func cellColumnWidth(c *uv.Cell) int {
	if c == nil {
		return 1
	}
	if c.Width > 0 {
		return c.Width
	}
	return 1
}

func clampLocalInt(value, low, high int) int {
	if high < low {
		return low
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func (s *Screen) recordCellDamage(x, y int, c *uv.Cell) {
	if s == nil || s.damage == nil || !s.damage.recordsScreenDiff() {
		return
	}
	if isClearDamageCell(c) {
		s.recordDamage(ClearDamage(uv.Rect(x, y, 1, 1)))
		return
	}
	s.damage.recordSpanCell(x, y, cloneDamageCell(c))
}

func (s *Screen) recordFillDamage(c *uv.Cell, area uv.Rectangle) {
	if s == nil || s.damage == nil || !s.damage.recordsScreenDiff() || area.Empty() {
		return
	}
	if isClearDamageCell(c) {
		s.recordDamage(ClearDamage(area))
		return
	}
	cell := cloneDamageCell(c)
	for y := area.Min.Y; y < area.Max.Y; y++ {
		s.damage.recordRepeatedSpan(area.Min.X, y, area.Dx(), cell)
	}
}

func (s *Screen) recordDamage(d Damage) {
	if s == nil || s.damage == nil {
		return
	}
	s.damage.record(d)
}

func (s *Screen) recordScrollbackLine(y int, line uv.Line, wrapped bool) {
	if s == nil || s.damage == nil || line == nil {
		return
	}
	s.damage.recordScrollbackLine(y, line, wrapped)
}

func (s *Screen) scrollbackDamageForLine(y int, line uv.Line, wrapped bool) ScrollbackDamage {
	if line == nil {
		return ScrollbackDamage{}
	}
	return compactScrollbackDamage(y, line, wrapped)
}

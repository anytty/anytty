package ansi

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/anytty/anytty/clients/tui/render"
)

// maxSequence caps how many bytes one unterminated escape sequence may hold
// before the parser gives up on it, so a malformed stream cannot grow memory.
const maxSequence = 8192

// maxScrollback caps the number of scrolled-off lines retained per parser.
const maxScrollback = 1024

// Modes is the mode-bit snapshot the host routing layer tracks (PROTOCOL
// §6.5, §6.8). The parser mirrors what the program asked for; the host
// decides what to do with it.
type Modes struct {
	// CursorVisible is DECTCEM (?25), on by default.
	CursorVisible bool
	// AltScreen is the alternate screen buffer (?1049/?1047/?47).
	AltScreen bool
	// MouseCell is mode 1000 (press/release reporting).
	MouseCell bool
	// MouseDrag is mode 1002 (button-motion reporting).
	MouseDrag bool
	// MouseAny is mode 1003 (any-motion reporting).
	MouseAny bool
	// MouseSGR is mode 1006 (SGR coordinate encoding).
	MouseSGR bool
	// BracketPaste is mode 2004 (paste is wrapped in ESC[200~..ESC[201~).
	BracketPaste bool
}

// MouseTracking reports whether any mouse reporting mode is enabled.
func (m Modes) MouseTracking() bool { return m.MouseCell || m.MouseDrag || m.MouseAny }

// Parser turns a raw PTY byte stream into a fixed-size screen grid. It is not
// safe for concurrent use; the runtime serializes feeds and snapshots.
type Parser struct {
	cols, rows int

	main []Cell
	alt  []Cell

	x, y        int
	pendingWrap bool

	savedX, savedY       int
	altSavedX, altSavedY int

	style sgrState
	modes Modes

	scrollback [][]Cell

	pending []byte
}

// New returns a parser for a cols×rows grid. The cursor starts at the top
// left and is visible.
func New(cols, rows int) *Parser {
	p := &Parser{}
	p.Resize(cols, rows)
	p.modes.CursorVisible = true
	return p
}

// Resize changes the grid size, preserving the top-left overlap. A wide
// grapheme cut off at the right edge is dropped, never left half-present.
func (p *Parser) Resize(cols, rows int) {
	if cols < 0 {
		cols = 0
	}
	if rows < 0 {
		rows = 0
	}
	main := resizeGrid(p.main, p.cols, p.rows, cols, rows)
	alt := resizeGrid(p.alt, p.cols, p.rows, cols, rows)
	p.cols, p.rows = cols, rows
	p.main, p.alt = main, alt
	p.x = clamp(p.x, 0, max(0, cols-1))
	p.y = clamp(p.y, 0, max(0, rows-1))
	p.pendingWrap = false
}

// Write feeds one chunk of PTY output. Chunks may split UTF-8 runes or escape
// sequences anywhere; the parser buffers the remainder until the next call.
func (p *Parser) Write(data []byte) {
	if len(data) == 0 {
		return
	}
	p.pending = append(p.pending, data...)
	p.consume()
	if len(p.pending) == 0 && cap(p.pending) > 4096 {
		p.pending = nil
	}
}

// Screen snapshots the active buffer, cursor included.
func (p *Parser) Screen() Screen {
	visible := p.modes.CursorVisible
	if p.cols == 0 || p.rows == 0 {
		return Screen{CursorVisible: visible}
	}
	grid := p.grid()
	lines := make([][]Cell, p.rows)
	for y := 0; y < p.rows; y++ {
		row := make([]Cell, p.cols)
		copy(row, grid[y*p.cols:(y+1)*p.cols])
		lines[y] = row
	}
	return Screen{
		Lines:         lines,
		CursorX:       p.x,
		CursorY:       p.y,
		CursorVisible: visible,
	}
}

// Modes returns the current mode bits.
func (p *Parser) Modes() Modes { return p.modes }

// Cursor returns the zero-based cursor position and its visibility.
func (p *Parser) Cursor() (x, y int, visible bool) { return p.x, p.y, p.modes.CursorVisible }

// Cols and Rows return the grid size.
func (p *Parser) Cols() int { return p.cols }
func (p *Parser) Rows() int { return p.rows }

// Scrollback returns a copy of the lines that scrolled off the top of the
// primary screen, oldest first. The alternate screen never contributes.
func (p *Parser) Scrollback() [][]Cell {
	out := make([][]Cell, len(p.scrollback))
	for i, row := range p.scrollback {
		out[i] = append([]Cell(nil), row...)
	}
	return out
}

// Reset returns the parser to its initial state at the current size.
func (p *Parser) Reset() {
	p.clear(p.main)
	p.clear(p.alt)
	p.x, p.y = 0, 0
	p.savedX, p.savedY = 0, 0
	p.altSavedX, p.altSavedY = 0, 0
	p.pendingWrap = false
	p.style.reset()
	p.modes = Modes{CursorVisible: true}
	p.scrollback = nil
	p.pending = nil
}

func (p *Parser) consume() {
	for len(p.pending) > 0 {
		b := p.pending[0]
		switch {
		case b == 0x1b:
			n, ok := p.consumeEscape(p.pending)
			if !ok {
				return
			}
			p.pending = p.pending[n:]
		case b == '\r':
			p.carriageReturn()
			p.pending = p.pending[1:]
		case b == '\n':
			p.lineFeed()
			p.pending = p.pending[1:]
		case b == '\b':
			p.backspace()
			p.pending = p.pending[1:]
		case b == '\t':
			p.tab()
			p.pending = p.pending[1:]
		case b < 0x20 || b == 0x7f:
			p.pending = p.pending[1:]
		default:
			r, size := utf8.DecodeRune(p.pending)
			if r == utf8.RuneError && size == 1 {
				if !utf8.FullRune(p.pending) {
					return
				}
				p.pending = p.pending[1:]
				continue
			}
			p.pending = p.pending[size:]
			p.putRune(r)
		}
	}
}

// consumeEscape consumes one escape sequence starting at data[0]. It returns
// false when the sequence is incomplete and the caller should wait.
func (p *Parser) consumeEscape(data []byte) (int, bool) {
	if len(data) < 2 {
		return 0, false
	}
	switch data[1] {
	case '[':
		return p.consumeCSI(data)
	case ']':
		return p.consumeControlString(data, 0x07)
	case 'P', '^', '_', 'X':
		return p.consumeControlString(data, 0)
	case '(', ')', '*', '+', '#':
		if len(data) < 3 {
			return 0, false
		}
		return 3, true
	case 0x1b:
		return 1, true
	case '7':
		p.saveCursor()
		return 2, true
	case '8':
		p.restoreCursor()
		return 2, true
	case 'D':
		p.lineFeed()
		return 2, true
	case 'E':
		p.carriageReturn()
		p.lineFeed()
		return 2, true
	case 'M':
		p.reverseIndex()
		return 2, true
	case 'c':
		p.Reset()
		return 2, true
	default:
		return 2, true
	}
}

// consumeCSI parses ESC [ (params) final. Private prefixes (?, >, <, =, !)
// are carried into the dispatcher; unknown finals are skipped whole.
func (p *Parser) consumeCSI(data []byte) (int, bool) {
	i := 2
	var prefix byte
	if i < len(data) {
		switch data[i] {
		case '?', '>', '<', '=', '!':
			prefix = data[i]
			i++
		}
	}
	start := i
	for i < len(data) {
		b := data[i]
		switch {
		case b >= 0x40 && b <= 0x7e:
			p.dispatchCSI(prefix, b, parseParams(data[start:i]))
			return i + 1, true
		case b >= 0x20 && b <= 0x3f:
			i++
		default:
			return i + 1, true
		}
		if i-start > maxSequence {
			return i, true
		}
	}
	return 0, false
}

// consumeControlString skips OSC/DCS/APC/PM/SOS payloads. bel >= 0 also
// accepts BEL as a terminator (OSC); every control string accepts ST.
func (p *Parser) consumeControlString(data []byte, bel byte) (int, bool) {
	for i := 2; i < len(data); i++ {
		switch data[i] {
		case 0x07:
			if bel != 0 {
				return i + 1, true
			}
		case 0x1b:
			if i+1 >= len(data) {
				return 0, false
			}
			if data[i+1] == '\\' {
				return i + 2, true
			}
		}
	}
	if len(data) > maxSequence {
		return len(data) - 1, true
	}
	return 0, false
}

func (p *Parser) dispatchCSI(prefix byte, final byte, params [][]int) {
	if prefix == '?' {
		switch final {
		case 'h':
			p.setPrivateModes(params, true)
		case 'l':
			p.setPrivateModes(params, false)
		}
		return
	}
	if prefix != 0 {
		return
	}
	switch final {
	case 'm':
		p.style.apply(params)
	case 'A':
		p.moveCursor(0, -paramAt(params, 0, 1), false)
	case 'B':
		p.moveCursor(0, paramAt(params, 0, 1), false)
	case 'C':
		p.moveCursor(paramAt(params, 0, 1), 0, false)
	case 'D':
		p.moveCursor(-paramAt(params, 0, 1), 0, false)
	case 'E':
		p.moveCursor(0, paramAt(params, 0, 1), true)
	case 'F':
		p.moveCursor(0, -paramAt(params, 0, 1), true)
	case 'G', '`':
		p.setColumn(paramAt(params, 0, 1) - 1)
	case 'H', 'f':
		p.setCursor(paramAt(params, 1, 1)-1, paramAt(params, 0, 1)-1)
	case 'd':
		p.setCursor(-1, paramAt(params, 0, 1)-1)
	case 'J':
		p.eraseDisplay(paramAt(params, 0, 0))
	case 'K':
		p.eraseLine(paramAt(params, 0, 0))
	case 'S':
		p.scrollUp(paramAt(params, 0, 1))
	case 'T':
		p.scrollDown(paramAt(params, 0, 1))
	case 's':
		p.saveCursor()
	case 'u':
		p.restoreCursor()
	case 'r':
		// Scroll regions are simplified to the full screen.
		p.pendingWrap = false
	}
}

func (p *Parser) setPrivateModes(params [][]int, on bool) {
	for _, group := range params {
		if len(group) == 0 || group[0] < 0 {
			continue
		}
		switch group[0] {
		case 25:
			p.modes.CursorVisible = on
		case 47, 1047, 1049:
			p.setAltScreen(on, group[0])
		case 1000:
			p.modes.MouseCell = on
		case 1002:
			p.modes.MouseDrag = on
		case 1003:
			p.modes.MouseAny = on
		case 1006:
			p.modes.MouseSGR = on
		case 2004:
			p.modes.BracketPaste = on
		}
	}
}

func (p *Parser) setAltScreen(on bool, code int) {
	if on == p.modes.AltScreen {
		return
	}
	if on {
		p.altSavedX, p.altSavedY = p.x, p.y
		p.modes.AltScreen = true
		if code != 47 {
			p.clear(p.alt)
		}
		p.x, p.y = 0, 0
		p.pendingWrap = false
		return
	}
	p.modes.AltScreen = false
	p.x, p.y = p.altSavedX, p.altSavedY
	p.pendingWrap = false
}

func (p *Parser) putRune(r rune) {
	if p.cols <= 0 || p.rows <= 0 {
		return
	}
	width := render.RuneWidth(r)
	if width == 0 {
		p.combine(string(r))
		return
	}
	if p.pendingWrap {
		p.pendingWrap = false
		p.x = 0
		p.lineFeed()
	}
	if width > 1 && p.x+width > p.cols {
		p.x = 0
		p.lineFeed()
	}
	if width > p.cols {
		return
	}
	p.eraseCell(p.x, p.y)
	grid := p.grid()
	style := p.style.token()
	index := p.y*p.cols + p.x
	if width > 1 {
		p.eraseCell(p.x+1, p.y)
		grid[index] = Cell{Text: string(r), Width: width, Style: style}
		grid[index+1] = Cell{Continuation: true, Style: style}
	} else {
		grid[index] = Cell{Text: string(r), Width: 1, Style: style}
	}
	p.x += width
	if p.x >= p.cols {
		p.x = p.cols - 1
		p.pendingWrap = true
	}
}

// combine glues a zero-width cluster (combining mark, variation selector) to
// the previous head cell; a mark with no base is dropped.
func (p *Parser) combine(text string) {
	if p.cols <= 0 || p.rows <= 0 {
		return
	}
	x := p.x - 1
	if p.pendingWrap {
		x = p.x
	}
	if x < 0 {
		return
	}
	grid := p.grid()
	index := p.y*p.cols + x
	if grid[index].Continuation && x > 0 {
		index--
	}
	if grid[index].Text == "" || grid[index].Width == 0 {
		return
	}
	grid[index].Text += text
}

func (p *Parser) carriageReturn() {
	p.x = 0
	p.pendingWrap = false
}

func (p *Parser) backspace() {
	if p.x > 0 {
		p.x--
	}
	p.pendingWrap = false
}

func (p *Parser) tab() {
	if p.cols <= 0 {
		return
	}
	next := (p.x/8 + 1) * 8
	if next >= p.cols {
		next = p.cols - 1
	}
	p.x = next
	p.pendingWrap = false
}

func (p *Parser) lineFeed() {
	p.pendingWrap = false
	if p.rows <= 0 {
		return
	}
	if p.y >= p.rows-1 {
		p.scrollUp(1)
		return
	}
	p.y++
}

func (p *Parser) reverseIndex() {
	p.pendingWrap = false
	if p.rows <= 0 {
		return
	}
	if p.y == 0 {
		p.scrollDown(1)
		return
	}
	p.y--
}

func (p *Parser) scrollUp(n int) {
	if p.cols <= 0 || p.rows <= 0 || n <= 0 {
		return
	}
	if n > p.rows {
		n = p.rows
	}
	grid := p.grid()
	if !p.modes.AltScreen {
		for i := 0; i < n; i++ {
			p.pushScrollback(grid[i*p.cols : (i+1)*p.cols])
		}
	}
	copy(grid, grid[n*p.cols:])
	for i := (p.rows - n) * p.cols; i < p.rows*p.cols; i++ {
		grid[i] = Cell{}
	}
}

func (p *Parser) scrollDown(n int) {
	if p.cols <= 0 || p.rows <= 0 || n <= 0 {
		return
	}
	if n > p.rows {
		n = p.rows
	}
	grid := p.grid()
	copy(grid[n*p.cols:], grid[:(p.rows-n)*p.cols])
	for i := 0; i < n*p.cols; i++ {
		grid[i] = Cell{}
	}
}

func (p *Parser) pushScrollback(row []Cell) {
	p.scrollback = append(p.scrollback, append([]Cell(nil), row...))
	if len(p.scrollback) > maxScrollback {
		p.scrollback = append(p.scrollback[:0], p.scrollback[len(p.scrollback)-maxScrollback:]...)
	}
}

func (p *Parser) moveCursor(dx, dy int, home bool) {
	if home {
		p.x = 0
	}
	p.x = clamp(p.x+dx, 0, max(0, p.cols-1))
	p.y = clamp(p.y+dy, 0, max(0, p.rows-1))
	p.pendingWrap = false
}

func (p *Parser) setColumn(x int) {
	if p.cols > 0 {
		p.x = clamp(x, 0, p.cols-1)
	}
	p.pendingWrap = false
}

func (p *Parser) setCursor(x, y int) {
	if x >= 0 && p.cols > 0 {
		p.x = clamp(x, 0, p.cols-1)
	}
	if y >= 0 && p.rows > 0 {
		p.y = clamp(y, 0, p.rows-1)
	}
	p.pendingWrap = false
}

func (p *Parser) saveCursor() {
	p.savedX, p.savedY = p.x, p.y
	p.pendingWrap = false
}

func (p *Parser) restoreCursor() {
	p.x = clamp(p.savedX, 0, max(0, p.cols-1))
	p.y = clamp(p.savedY, 0, max(0, p.rows-1))
	p.pendingWrap = false
}

func (p *Parser) eraseDisplay(mode int) {
	switch mode {
	case 0:
		p.eraseCells(p.y, p.x, p.cols-1)
		for y := p.y + 1; y < p.rows; y++ {
			p.eraseCells(y, 0, p.cols-1)
		}
	case 1:
		for y := 0; y < p.y; y++ {
			p.eraseCells(y, 0, p.cols-1)
		}
		p.eraseCells(p.y, 0, p.x)
	default:
		for y := 0; y < p.rows; y++ {
			p.eraseCells(y, 0, p.cols-1)
		}
	}
}

func (p *Parser) eraseLine(mode int) {
	switch mode {
	case 1:
		p.eraseCells(p.y, 0, p.x)
	case 2:
		p.eraseCells(p.y, 0, p.cols-1)
	default:
		p.eraseCells(p.y, p.x, p.cols-1)
	}
}

// eraseCells erases [x0, x1] on row y, including the partner half of any wide
// grapheme that the range only partially covers.
func (p *Parser) eraseCells(y, x0, x1 int) {
	if y < 0 || y >= p.rows {
		return
	}
	if x0 < 0 {
		x0 = 0
	}
	if x1 >= p.cols {
		x1 = p.cols - 1
	}
	for x := x0; x <= x1; x++ {
		p.eraseCell(x, y)
	}
}

// eraseCell blanks one cell and keeps wide graphemes whole: erasing a
// continuation erases its head, erasing a head erases its continuation.
func (p *Parser) eraseCell(x, y int) {
	if x < 0 || x >= p.cols || y < 0 || y >= p.rows {
		return
	}
	grid := p.grid()
	index := y*p.cols + x
	if grid[index].Continuation && x > 0 {
		grid[index-1] = Cell{}
	}
	if grid[index].Width > 1 && x+1 < p.cols {
		grid[index+1] = Cell{}
	}
	grid[index] = Cell{}
}

func (p *Parser) clear(grid []Cell) {
	for i := range grid {
		grid[i] = Cell{}
	}
}

func (p *Parser) grid() []Cell {
	if p.modes.AltScreen {
		return p.alt
	}
	return p.main
}

// resizeGrid allocates a cols×rows grid and copies the top-left overlap from
// old. A wide head cut off at the right edge is dropped with its partner.
func resizeGrid(old []Cell, oldCols, oldRows, cols, rows int) []Cell {
	grid := make([]Cell, cols*rows)
	if oldCols <= 0 || oldRows <= 0 {
		return grid
	}
	copyCols := min(cols, oldCols)
	copyRows := min(rows, oldRows)
	for y := 0; y < copyRows; y++ {
		copy(grid[y*cols:y*cols+copyCols], old[y*oldCols:y*oldCols+copyCols])
	}
	if cols < oldCols && cols > 0 {
		for y := 0; y < copyRows; y++ {
			index := y*cols + cols - 1
			switch {
			case grid[index].Continuation:
				grid[index] = Cell{}
				if cols > 1 {
					grid[index-1] = Cell{}
				}
			case grid[index].Width > 1:
				grid[index] = Cell{}
			}
		}
	}
	return grid
}

func parseParams(data []byte) [][]int {
	if len(data) == 0 {
		return nil
	}
	groups := strings.Split(string(data), ";")
	out := make([][]int, 0, len(groups))
	for _, group := range groups {
		parts := strings.Split(group, ":")
		nums := make([]int, 0, len(parts))
		for _, part := range parts {
			if part == "" {
				nums = append(nums, -1)
				continue
			}
			n, err := strconv.Atoi(part)
			if err != nil {
				nums = append(nums, -1)
				continue
			}
			nums = append(nums, n)
		}
		out = append(out, nums)
	}
	return out
}

func paramAt(params [][]int, i, fallback int) int {
	if i >= len(params) || len(params[i]) == 0 || params[i][0] < 0 {
		return fallback
	}
	return params[i][0]
}

func clamp(value, lo, hi int) int {
	if value < lo {
		return lo
	}
	if value > hi {
		return hi
	}
	return value
}

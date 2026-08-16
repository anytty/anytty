package vt

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
)

func (e *Emulator) tryFastSGRText(data []byte) bool {
	if len(data) == 0 || !e.canFastPrintASCII() || e.IsAltScreen() || e.isModeSet(ansi.ModeLineFeedNewLine) {
		return false
	}
	if !e.isModeSet(ansi.ModeAutoWrap) {
		return false
	}
	if e.scr == nil || e.scr.buf == nil {
		return false
	}
	if e.scr.damage != nil && !e.scr.damage.scrollbackOnly && !e.scr.damage.semanticOnly && !e.scr.damage.lineHistoryOnly {
		return false
	}
	width := e.scr.Width()
	height := e.scr.Height()
	if width <= 0 || height <= 0 {
		return false
	}
	scroll := e.scr.ScrollRegion()
	if scroll.Min.X != 0 || scroll.Min.Y != 0 || scroll.Max.X != width || scroll.Max.Y != height {
		return false
	}
	if !canApplyFastSGRTextBatch(data) {
		return false
	}
	x, cursorY := e.scr.CursorPosition()
	if x < 0 {
		x = 0
	}
	if x >= width {
		x = width - 1
	}
	if cursorY < 0 {
		cursorY = 0
	}
	if cursorY >= height {
		cursorY = height - 1
	}
	style := e.scr.cursorPen()
	link := e.scr.cursorLink()
	damage := e.scr.damage
	nextPen := style
	atPhantom := e.atPhantom
	lastChar := e.lastChar
	scrollY := 0
	scrollback := e.Scrollback()
	scrollbackHint := fastSGRScrollbackHint(data, cursorY, height)
	scrollbackRows := make([]fastSGRScrollbackRow, 0, scrollbackHint)
	scrollbackDamages := make([]ScrollbackDamage, 0, scrollbackHint)

	flushScrollbackRow := func(row uv.Line, wasWrapped bool) {
		if damage != nil && (damage.scrollbackOnly || damage.semanticOnly || damage.lineHistoryOnly) {
			damage := ScrollbackDamage{
				Y:       scrollY,
				Wrapped: wasWrapped,
			}
			if runs, ok := compactASCIIStyleRuns(row); ok && len(runs) > 0 {
				damage.Runs = runs
			} else if text, ok := compactASCIIPlainLine(row); ok {
				damage.ASCII = true
				damage.Text = text
			} else {
				damage.Cells = cloneFastSGRLine(row, len(row))
			}
			scrollbackDamages = append(scrollbackDamages, damage)
			if scrollback != nil {
				scrollbackRows = append(scrollbackRows, fastSGRScrollbackRow{line: cloneFastSGRLine(row, len(row)), wrapped: wasWrapped})
			}
			scrollY++
			return
		}
		if scrollback != nil {
			scrollbackRows = append(scrollbackRows, fastSGRScrollbackRow{line: cloneFastSGRLine(row, len(row)), wrapped: wasWrapped})
		}
		scrollY++
	}

	linefeed := func() {
		if cursorY == height-1 {
			scrolled := e.scr.buf.Line(0)
			scrolledWrapped := e.scr.LineWrapped(0)
			scrolledUsed := clampLocalInt(e.scr.LineUsed(0), 0, len(scrolled))
			flushScrollbackRow(scrolled[:scrolledUsed], scrolledWrapped)
			scrollFastSGRScreenRows(e.scr, style)
			return
		}
		cursorY++
	}
	recordFastControl := func(kind string, x int, y int) {
		if damage == nil || !damage.semanticOnly {
			return
		}
		damage.recordControl(kind, x, y, 0)
	}
	printRun := func(run []byte) {
		for len(run) > 0 {
			if atPhantom {
				oldX, oldY := x, cursorY
				e.scr.SetLineWrapped(cursorY, true)
				linefeed()
				x = 0
				atPhantom = false
				recordFastControl("soft-wrap", oldX, oldY)
			}
			available := width - x
			if available <= 0 {
				atPhantom = true
				continue
			}
			count := min(len(run), available)
			startX, startY := x, cursorY
			row := e.scr.buf.Line(cursorY)
			if row == nil || len(row) != width {
				for i, b := range run[:count] {
					cell := uv.Cell{
						Content: printableASCIIStrings[b],
						Width:   1,
						Style:   style,
						Link:    link,
					}
					e.scr.SetCell(x+i, cursorY, &cell)
				}
			} else {
				for i, b := range run[:count] {
					row[x+i] = uv.Cell{
						Content: printableASCIIStrings[b],
						Width:   1,
						Style:   style,
						Link:    link,
					}
				}
				e.scr.buf.TouchLine(x, cursorY, count)
			}
			if damage != nil && damage.semanticOnly {
				damage.recordASCIITextRun(startX, startY, run[:count], style, link)
			}
			lastChar = rune(run[count-1])
			e.scr.markUsed(cursorY, x+count)
			run = run[count:]
			if x+count >= width {
				x = width - 1
				atPhantom = true
			} else {
				x += count
			}
		}
	}

	var paramsStack [16]int
	paramsScratch := paramsStack[:0]
	for i := 0; i < len(data); {
		start := i
		for i < len(data) && isPrintableASCII(data[i]) {
			i++
		}
		if i > start {
			printRun(data[start:i])
			continue
		}
		switch data[i] {
		case '\r':
			oldX, oldY := x, cursorY
			x = 0
			atPhantom = false
			recordFastControl("cr", oldX, oldY)
			i++
		case '\n':
			oldX, oldY := x, cursorY
			linefeed()
			atPhantom = false
			recordFastControl("lf", oldX, oldY)
			i++
		case ansi.ESC:
			next, ok := fastSGRSequenceEnd(data, i)
			if !ok {
				return false
			}
			paramsScratch = paramsScratch[:0]
			params, ok := parseSGRParams(data[i+2:next], paramsScratch)
			if !ok || !canApplyFastSGRStyle(params) {
				return false
			}
			readFastSGRStyle(params, &nextPen)
			style = nextPen
			i = next + 1
		default:
			return false
		}
	}
	if scrollback != nil {
		for _, row := range scrollbackRows {
			scrollback.PushWrapped(row.line, row.wrapped)
		}
	}
	for _, damage := range scrollbackDamages {
		e.scr.damage.record(damage)
	}
	e.scr.cur.Pen = nextPen
	e.scr.setCursor(x, cursorY, false)
	e.atPhantom = atPhantom
	e.lastChar = lastChar
	e.lastState = parser.GroundState
	return true
}

func canApplyFastSGRTextBatch(data []byte) bool {
	var paramsStack [16]int
	paramsScratch := paramsStack[:0]
	for i := 0; i < len(data); {
		start := i
		for i < len(data) && isPrintableASCII(data[i]) {
			i++
		}
		if i > start {
			continue
		}
		switch data[i] {
		case '\r', '\n':
			i++
		case ansi.ESC:
			next, ok := fastSGRSequenceEnd(data, i)
			if !ok {
				return false
			}
			paramsScratch = paramsScratch[:0]
			params, ok := parseSGRParams(data[i+2:next], paramsScratch)
			if !ok || !canApplyFastSGRStyle(params) {
				return false
			}
			i = next + 1
		default:
			return false
		}
	}
	return true
}

type fastSGRScrollbackRow struct {
	line    uv.Line
	wrapped bool
}

func scrollFastSGRScreenRows(screen *Screen, style uv.Style) {
	if screen == nil || screen.buf == nil {
		return
	}
	height := screen.buf.Height()
	width := screen.buf.Width()
	if height <= 0 || width <= 0 {
		return
	}
	lines := screen.buf.Lines
	if len(lines) < height {
		deleteFullWidthLines(screen.buf, 0, 1, screen.ScrollRegion(), fastSGRBlankCell(style))
		screen.deleteWrappedLines(0, 1)
		screen.deleteUsedLines(0, 1)
		return
	}
	// 中文说明：fast SGR 压力日志只需要 live latest screen。这里复用离屏行作为新底行，
	// 避免每个 PTY 批次先复制整屏；离屏历史仍在调用方按 used width 显式复制。
	reused := lines[0]
	copy(lines[0:height-1], lines[1:height])
	if len(reused) != width {
		reused = uv.NewLine(width)
	}
	fillFastSGRLine(reused, style)
	lines[height-1] = reused
	screen.deleteWrappedLines(0, 1)
	screen.deleteUsedLines(0, 1)
	for y := 0; y < height; y++ {
		screen.buf.TouchLine(0, y, width)
	}
}

func cloneFastSGRLine(line uv.Line, width int) uv.Line {
	if width <= 0 {
		return nil
	}
	out := make(uv.Line, width)
	copy(out, line)
	return out
}

func fillFastSGRLine(line uv.Line, style uv.Style) {
	cell := uv.EmptyCell
	cell.Style.Bg = style.Bg
	for i := range line {
		line[i] = cell
	}
}

func fastSGRBlankCell(style uv.Style) *uv.Cell {
	if style.Bg == nil {
		return nil
	}
	cell := uv.EmptyCell
	cell.Style.Bg = style.Bg
	return &cell
}

func fastSGRSequenceEnd(data []byte, start int) (int, bool) {
	if start+2 >= len(data) || data[start] != ansi.ESC || data[start+1] != '[' {
		return 0, false
	}
	for i := start + 2; i < len(data); i++ {
		b := data[i]
		if b == 'm' {
			return i, true
		}
		if (b >= '0' && b <= '9') || b == ';' {
			continue
		}
		return 0, false
	}
	return 0, false
}

func fastSGRScrollbackHint(data []byte, cursorY int, height int) int {
	if height <= 0 || cursorY < 0 {
		return 0
	}
	lines := cursorY
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if lines < height {
		return 0
	}
	return lines - height + 1
}

func parseSGRParams(data []byte, scratch []int) ([]int, bool) {
	if len(data) == 0 {
		return scratch, true
	}
	start := 0
	for start <= len(data) {
		end := start
		for end < len(data) && data[end] != ';' {
			end++
		}
		if end == start {
			scratch = append(scratch, 0)
		} else {
			value := 0
			for _, b := range data[start:end] {
				if b < '0' || b > '9' {
					return nil, false
				}
				value = value*10 + int(b-'0')
			}
			if value < 0 {
				return nil, false
			}
			scratch = append(scratch, value)
		}
		if end == len(data) {
			break
		}
		start = end + 1
	}
	return scratch, true
}

func canApplyFastSGRStyle(params []int) bool {
	for i := 0; i < len(params); i++ {
		switch p := params[i]; p {
		case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			22, 23, 24, 25, 27, 28, 29,
			30, 31, 32, 33, 34, 35, 36, 37, 39,
			40, 41, 42, 43, 44, 45, 46, 47, 49,
			90, 91, 92, 93, 94, 95, 96, 97,
			100, 101, 102, 103, 104, 105, 106, 107:
		case 38, 48:
			if i+2 >= len(params) || params[i+1] != 5 || params[i+2] < 0 || params[i+2] > 255 {
				return false
			}
			i += 2
		default:
			return false
		}
	}
	return true
}

func readFastSGRStyle(params []int, pen *uv.Style) {
	if len(params) == 0 {
		*pen = uv.Style{}
		return
	}
	for i := 0; i < len(params); i++ {
		switch p := params[i]; p {
		case 0:
			*pen = uv.Style{}
		case 1:
			pen.Attrs |= uv.AttrBold
		case 2:
			pen.Attrs |= uv.AttrFaint
		case 3:
			pen.Attrs |= uv.AttrItalic
		case 4:
			pen.Underline = uv.UnderlineSingle
		case 5:
			pen.Attrs |= uv.AttrBlink
		case 6:
			pen.Attrs |= uv.AttrRapidBlink
		case 7:
			pen.Attrs |= uv.AttrReverse
		case 8:
			pen.Attrs |= uv.AttrConceal
		case 9:
			pen.Attrs |= uv.AttrStrikethrough
		case 22:
			pen.Attrs &^= (uv.AttrBold | uv.AttrFaint)
		case 23:
			pen.Attrs &^= uv.AttrItalic
		case 24:
			pen.Underline = uv.UnderlineNone
		case 25:
			pen.Attrs &^= (uv.AttrBlink | uv.AttrRapidBlink)
		case 27:
			pen.Attrs &^= uv.AttrReverse
		case 28:
			pen.Attrs &^= uv.AttrConceal
		case 29:
			pen.Attrs &^= uv.AttrStrikethrough
		case 30, 31, 32, 33, 34, 35, 36, 37:
			pen.Fg = ansi.Black + ansi.BasicColor(p-30)
		case 38:
			if i+2 < len(params) && params[i+1] == 5 {
				pen.Fg = ansi.IndexedColor(params[i+2])
				i += 2
			}
		case 39:
			pen.Fg = nil
		case 40, 41, 42, 43, 44, 45, 46, 47:
			pen.Bg = ansi.Black + ansi.BasicColor(p-40)
		case 48:
			if i+2 < len(params) && params[i+1] == 5 {
				pen.Bg = ansi.IndexedColor(params[i+2])
				i += 2
			}
		case 49:
			pen.Bg = nil
		case 90, 91, 92, 93, 94, 95, 96, 97:
			pen.Fg = ansi.BrightBlack + ansi.BasicColor(p-90)
		case 100, 101, 102, 103, 104, 105, 106, 107:
			pen.Bg = ansi.BrightBlack + ansi.BasicColor(p-100)
		default:
			return
		}
	}
}

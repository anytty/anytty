package vt

import (
	"unicode/utf8"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

var printableASCIIStrings = func() [ansi.DEL]string {
	var out [ansi.DEL]string
	for i := byte(0); i < ansi.DEL; i++ {
		out[i] = string([]byte{i})
	}
	return out
}()

// handlePrint handles printable characters.
func (e *Emulator) handlePrint(r rune) {
	if r >= ansi.SP && r < ansi.DEL {
		if len(e.grapheme) > 0 {
			// If we have a grapheme buffer, flush it before handling the ASCII character.
			e.flushGrapheme()
		}
		e.handleASCIIPrint(byte(r))
	} else {
		e.grapheme = append(e.grapheme, r)
	}
}

func (e *Emulator) handleASCIIPrint(b byte) {
	awm := e.isModeSet(ansi.ModeAutoWrap)
	x, y := e.scr.CursorPosition()
	if e.atPhantom && awm {
		e.scr.SetLineWrapped(y, true)
		// moves cursor down similar to [Terminal.linefeed] except it doesn't
		// respects [ansi.LNM] mode.
		// This will reset the phantom state i.e. pending wrap state.
		e.softWrapIndex()
		_, y = e.scr.CursorPosition()
		x = 0
	}

	content := printableASCIIStrings[b]
	var charset CharSet
	if e.gsingle > 1 && e.gsingle < 4 {
		charset = e.charsets[e.gsingle]
		e.gsingle = 0
	} else {
		charset = e.charsets[e.gl]
	}
	if charset != nil {
		if mapped, ok := charset[b]; ok {
			content = mapped
		}
	}

	cell := uv.Cell{
		Content: content,
		Width:   1,
		Style:   e.scr.cursorPen(),
		Link:    e.scr.cursorLink(),
	}
	e.lastChar = rune(b)
	e.scr.SetCell(x, y, &cell)
	e.scr.damage.recordTextSpan(x, y, []uv.Cell{cell})

	e.atPhantom = awm && x >= e.scr.Width()-1
	if !e.atPhantom {
		x++
	}
	e.scr.setCursor(x, y, false)
}

func (e *Emulator) handleASCIIPrintRun(data []byte) {
	if len(data) == 0 {
		return
	}
	awm := e.isModeSet(ansi.ModeAutoWrap)
	style := e.scr.cursorPen()
	link := e.scr.cursorLink()
	for len(data) > 0 {
		x, y := e.scr.CursorPosition()
		if e.atPhantom && awm {
			e.scr.SetLineWrapped(y, true)
			e.softWrapIndex()
			_, y = e.scr.CursorPosition()
			x = 0
		}

		width := e.scr.Width()
		if width <= 0 {
			return
		}
		available := width - x
		if available <= 0 {
			e.atPhantom = awm
			if !awm {
				e.scr.setCursor(max(0, width-1), y, false)
			}
			continue
		}
		count := min(len(data), available)
		e.scr.SetASCIICells(x, y, data[:count], style, link)
		if e.scr.damage != nil && !e.scr.damage.scrollbackOnly {
			if e.scr.damage.semanticOnly {
				e.scr.damage.recordASCIITextRun(x, y, data[:count], style, link)
			} else if !e.scr.damage.lineHistoryOnly {
				cells := make([]uv.Cell, count)
				for i, b := range data[:count] {
					cells[i] = uv.Cell{
						Content: printableASCIIStrings[b],
						Width:   1,
						Style:   style,
						Link:    link,
					}
				}
				e.scr.damage.recordTextSpan(x, y, cells)
			}
		}
		e.lastChar = rune(data[count-1])
		data = data[count:]
		e.atPhantom = awm && x+count >= width
		if e.atPhantom {
			x = width - 1
		} else {
			x += count
		}
		e.scr.setCursor(x, y, false)
	}
}

// flushGrapheme flushes the current grapheme buffer, if any, and handles the
// grapheme as a single unit.
func (e *Emulator) flushGrapheme() {
	if len(e.grapheme) == 0 {
		return
	}

	// XXX: We always use [ansi.GraphemeWidth] here to report accurate widths
	// and it's up to the caller to decide how to handle Unicode vs non-Unicode
	// modes.
	method := ansi.GraphemeWidth
	graphemes := string(e.grapheme)
	for len(graphemes) > 0 {
		cluster, width := ansi.FirstGraphemeCluster(graphemes, method)
		e.handleGrapheme(cluster, width)
		graphemes = graphemes[len(cluster):]
	}
	e.grapheme = e.grapheme[:0] // Reset the grapheme buffer.
}

// handleGrapheme handles UTF-8 graphemes.
func (e *Emulator) handleGrapheme(content string, width int) {
	awm := e.isModeSet(ansi.ModeAutoWrap)
	cell := uv.Cell{
		Content: content,
		Width:   width,
		Style:   e.scr.cursorPen(),
		Link:    e.scr.cursorLink(),
	}

	x, y := e.scr.CursorPosition()
	if e.atPhantom && awm {
		e.scr.SetLineWrapped(y, true)
		// moves cursor down similar to [Terminal.linefeed] except it doesn't
		// respects [ansi.LNM] mode.
		// This will reset the phantom state i.e. pending wrap state.
		e.softWrapIndex()
		_, y = e.scr.CursorPosition()
		x = 0
	}

	// Handle character set mappings
	if len(content) == 1 { //nolint:nestif
		var charset CharSet
		c := content[0]
		if e.gsingle > 1 && e.gsingle < 4 {
			charset = e.charsets[e.gsingle]
			e.gsingle = 0
		} else if c < 128 {
			charset = e.charsets[e.gl]
		} else {
			charset = e.charsets[e.gr]
		}

		if charset != nil {
			if r, ok := charset[c]; ok {
				cell.Content = r
				cell.Width = 1
			}
		}
	}

	if cell.Width == 1 && len(content) == 1 {
		e.lastChar, _ = utf8.DecodeRuneInString(content)
	}

	// 中文说明：宽字符（width>1）行尾剩余列不足时不能直接落在行尾——
	// SetCell 会拒写或擦掉相邻宽字符，字符被静默丢弃。DECAWM 开启时应
	// 整字软换行到下一行（xterm 行为）；关闭时按能容纳的位置左移。
	screenWidth := e.scr.Width()
	if cell.Width > 1 && x+cell.Width > screenWidth {
		if awm {
			e.scr.SetLineWrapped(y, true)
			e.softWrapIndex()
			_, y = e.scr.CursorPosition()
			x = 0
		} else if screenWidth >= cell.Width {
			x = screenWidth - cell.Width
		}
	}

	e.scr.SetCell(x, y, &cell)
	e.scr.damage.recordTextSpan(x, y, []uv.Cell{cell})

	// Handle phantom state at the end of the line
	// 中文说明：pending-wrap 判定必须按 cell 结束列（x+Width）算；
	// 宽字符恰好填满行尾时起始列到不了 width-1，旧判定会让 cursor 越界
	// clamp 回行尾，下一个字符覆盖行尾并擦掉刚写入的宽字符。
	e.atPhantom = awm && x+cell.Width >= screenWidth
	if e.atPhantom {
		x = max(screenWidth-1, 0)
	} else {
		x += cell.Width
	}

	// NOTE: We don't reset the phantom state here, we handle it up above.
	e.scr.setCursor(x, y, false)
}

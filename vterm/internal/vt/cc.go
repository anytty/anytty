package vt

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// handleControl handles a control character.
func (e *Emulator) handleControl(r byte) {
	e.flushGrapheme() // Flush any pending grapheme before handling control codes.
	if !e.handleCc(r) {
		e.logf("unhandled sequence: ControlCode %q", r)
	}
}

// linefeed is the same as [index], except that it respects [ansi.LNM] mode.
func (e *Emulator) linefeed() {
	x, y := e.scr.CursorPosition()
	e.indexMove()
	e.scr.damage.recordControl("lf", x, y, 0)
	if e.isModeSet(ansi.ModeLineFeedNewLine) {
		e.carriageReturn()
	}
}

func (e *Emulator) nextLine() {
	// 中文说明：NEL 是 CR+LF 语义；按 ordered control 暴露给 history，
	// 避免 history parser 另存一份 next-line 状态。
	e.carriageReturn()
	e.linefeed()
}

// index moves the cursor down one line, scrolling up if necessary. This
// always resets the phantom state i.e. pending wrap state.
func (e *Emulator) index() {
	e.indexWithControl("ind")
}

func (e *Emulator) softWrapIndex() {
	e.indexWithControl("soft-wrap")
}

func (e *Emulator) indexWithControl(kind string) {
	x, y := e.scr.CursorPosition()
	e.indexMove()
	e.scr.damage.recordControl(kind, x, y, 0)
}

func (e *Emulator) indexMove() {
	x, y := e.scr.CursorPosition()
	scroll := e.scr.ScrollRegion()
	// XXX: Handle scrollback whenever we add it.
	if y == scroll.Max.Y-1 && x >= scroll.Min.X && x < scroll.Max.X {
		e.scr.ScrollUp(1)
	} else if y < scroll.Max.Y-1 || !uv.Pos(x, y).In(scroll) {
		e.scr.moveCursor(0, 1)
	}
	e.atPhantom = false
}

// horizontalTabSet sets a horizontal tab stop at the current cursor position.
func (e *Emulator) horizontalTabSet() {
	x, y := e.scr.CursorPosition()
	e.tabstops.Set(x)
	// 中文说明：HTS 修改的是 vterm tab stop 状态；history 只消费后续
	// HT 的 resolved column，这里只把状态变更保留在 ordered semantic stream。
	e.scr.damage.recordControl("hts", x, y, 0)
}

// reverseIndex moves the cursor up one line, or scrolling down. This does not
// reset the phantom state i.e. pending wrap state.
func (e *Emulator) reverseIndex() {
	x, y := e.scr.CursorPosition()
	scroll := e.scr.ScrollRegion()
	if y == scroll.Min.Y && x >= scroll.Min.X && x < scroll.Max.X {
		e.scr.ScrollDown(1)
	} else {
		e.scr.moveCursor(0, -1)
	}
	e.scr.damage.recordControl("ri", x, y, 0)
}

// backspace moves the cursor back one cell, if possible.
func (e *Emulator) backspace() {
	// This acts like [ansi.CUB]
	e.moveCursor(-1, 0)
	x, y := e.scr.CursorPosition()
	e.scr.damage.recordControl("bs", x, y, 1)
}

package vt

import (
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
)

// handleEsc handles an escape sequence.
func (e *Emulator) handleEsc(cmd ansi.Cmd) {
	e.flushGrapheme() // Flush any pending grapheme before handling ESC sequences.
	if !e.handlers.handleEsc(int(cmd)) {
		var str string
		if inter := cmd.Intermediate(); inter != 0 {
			str += string(inter) + " "
		}
		if final := cmd.Final(); final != 0 {
			str += string(final)
		}
		e.logf("unhandled sequence: ESC %q", str)
	}
}

// fullReset performs a full terminal reset as in [ansi.RIS].
func (e *Emulator) fullReset() {
	if e.scr != nil && e.scr.damage != nil {
		// 中文说明：RIS 是终端级 reset 语义；screen reset damage 只服务 live，
		// history 必须消费这个 ordered control 才能丢弃 reset 前 mutable frontier。
		x, y := e.scr.CursorPosition()
		e.scr.damage.recordControl("ris", x, y, 0)
	}
	e.scrs[0].Reset()
	e.scrs[1].Reset()
	e.resetTabStops()

	// XXX: Do we reset all modes here? Investigate.
	e.resetModes()

	e.gl, e.gr = 0, 1
	e.gsingle = 0
	e.charsets = [4]CharSet{}
	e.atPhantom = false
	e.grapheme = e.grapheme[:0]
	e.lastChar = 0
	e.lastState = parser.GroundState
}

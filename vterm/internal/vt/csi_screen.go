package vt

import (
	uv "github.com/charmbracelet/ultraviolet"
)

// eraseCharacter erases n characters starting from the cursor position. It
// does not move the cursor. This is equivalent to [ansi.ECH].
func (e *Emulator) eraseCharacter(n int) {
	e.eraseCharacterWithBlank(n, e.scr.blankCell())
}

func (e *Emulator) eraseCharacterWithBlank(n int, blank *uv.Cell) {
	if n <= 0 {
		n = 1
	}
	x, y := e.scr.CursorPosition()
	rect := uv.Rect(x, y, n, 1)
	e.scr.FillArea(blank, rect)
	e.atPhantom = false
	e.scr.damage.recordControlWithCell("ech", x, y, n, blank)
	// ECH does not move the cursor.
}

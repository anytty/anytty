package render

import (
	"strconv"
	"strings"
)

// Full-screen mode sequences. Entering/leaving a screen is always emitted as
// one atomic string so the host never leaves the TTY half-switched.
const (
	// EnterAltScreen switches to the alternate screen buffer.
	EnterAltScreen = "\x1b[?1049h"
	// ExitAltScreen returns to the primary screen buffer.
	ExitAltScreen = "\x1b[?1049l"
	// HideCursor hides the hardware cursor.
	HideCursor = "\x1b[?25l"
	// ShowCursor shows the hardware cursor.
	ShowCursor = "\x1b[?25h"
	// EnableMouseCell reports cell-motion mouse events (mode 1000).
	EnableMouseCell = "\x1b[?1000h"
	// DisableMouseCell disables mode 1000.
	DisableMouseCell = "\x1b[?1000l"
	// EnableMouseDrag reports drag events (mode 1002).
	EnableMouseDrag = "\x1b[?1002h"
	// DisableMouseDrag disables mode 1002.
	DisableMouseDrag = "\x1b[?1002l"
	// EnableMouseSGR enables SGR mouse coordinates (mode 1006).
	EnableMouseSGR = "\x1b[?1006h"
	// DisableMouseSGR disables mode 1006.
	DisableMouseSGR = "\x1b[?1006l"
	// EnableBracketPaste enables bracketed paste (mode 2004).
	EnableBracketPaste = "\x1b[?2004h"
	// DisableBracketPaste disables mode 2004.
	DisableBracketPaste = "\x1b[?2004l"
	// ResetSGR resets every SGR attribute.
	ResetSGR = "\x1b[0m"
)

// EnterScreen returns the sequence that enters the alternate screen and
// enables the host modes a v2 session needs: hidden cursor, bracketed paste
// and mouse reporting (1000/1002/1006).
func EnterScreen() string {
	return EnterAltScreen + HideCursor + EnableBracketPaste + EnableMouseCell + EnableMouseDrag + EnableMouseSGR
}

// ExitScreen returns the exact inverse of EnterScreen: mouse modes first,
// then paste, cursor and finally the alternate screen so no mode leaks into
// the shell.
func ExitScreen() string {
	return DisableMouseSGR + DisableMouseDrag + DisableMouseCell + DisableBracketPaste + ShowCursor + ExitAltScreen
}

// EnableMouse returns the mouse reporting sequence for an explicit set of
// mode numbers (1000 cell, 1002 drag, 1006 SGR); unknown numbers are ignored.
func EnableMouse(modes ...int) string {
	return mouseModes(true, modes)
}

// DisableMouse is the inverse of EnableMouse.
func DisableMouse(modes ...int) string {
	return mouseModes(false, modes)
}

func mouseModes(enable bool, modes []int) string {
	var b strings.Builder
	for _, mode := range modes {
		var suffix string
		switch mode {
		case 1000, 1002, 1006:
		default:
			continue
		}
		if enable {
			suffix = "h"
		} else {
			suffix = "l"
		}
		b.WriteString("\x1b[?")
		b.WriteString(strconv.Itoa(mode))
		b.WriteString(suffix)
	}
	return b.String()
}

// CursorPosition returns the CSI sequence that moves the cursor to the
// zero-based cell (x, y); coordinates are clamped to >= 0.
func CursorPosition(x, y int) string {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return "\x1b[" + strconv.Itoa(y+1) + ";" + strconv.Itoa(x+1) + "H"
}

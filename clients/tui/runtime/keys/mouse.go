package keys

import "strconv"

// SGR 1006 button codes (PROTOCOL §6.7): the low two bits are the button,
// drag adds 32, motion without a button is 35 and the wheel is 64/65.
const (
	sgrDragOffset  = 32
	sgrWheelUp     = 64
	sgrWheelDown   = 65
	sgrShiftFlag   = 4
	sgrAltFlag     = 8
	sgrCtrlFlag    = 16
	sgrReleaseBase = 3
)

// EncodeMouse encodes one mouse action as an SGR 1006 sequence. X and Y are
// 1-based terminal coordinates; non-positive values clamp to 1. Only call
// this when routing decided the event goes to a terminal with mouse tracking
// on (PROTOCOL §6.5 priority 5).
func EncodeMouse(ev Event) ([]byte, bool) {
	if ev.Kind != KindMouse {
		return nil, false
	}
	button, final, ok := sgrMouseButton(ev.Action, ev.Button)
	if !ok {
		return nil, false
	}
	button += mouseModifier(ev.Mods)
	return sgrSequence(button, ev.X, ev.Y, final), true
}

// EncodeWheel encodes one wheel notch as an SGR 1006 sequence: positive
// Delta is wheel-up (64), negative is wheel-down (65).
func EncodeWheel(ev Event) ([]byte, bool) {
	if ev.Kind != KindWheel || ev.Delta == 0 {
		return nil, false
	}
	button := sgrWheelUp
	if ev.Delta < 0 {
		button = sgrWheelDown
	}
	button += mouseModifier(ev.Mods)
	return sgrSequence(button, ev.X, ev.Y, "M"), true
}

func sgrMouseButton(action, button string) (code int, final string, ok bool) {
	base, ok := sgrBaseButton(button)
	if !ok {
		return 0, "", false
	}
	switch action {
	case ActionPress:
		return base, "M", true
	case ActionDrag:
		// "none" + drag is motion without a button: 3+32 = 35.
		return base + sgrDragOffset, "M", true
	case ActionRelease:
		if button == ButtonNone {
			return sgrReleaseBase, "m", true
		}
		return base, "m", true
	default:
		return 0, "", false
	}
}

func sgrBaseButton(button string) (int, bool) {
	switch button {
	case ButtonLeft:
		return 0, true
	case ButtonMiddle:
		return 1, true
	case ButtonRight:
		return 2, true
	case ButtonNone:
		return 3, true
	default:
		return 0, false
	}
}

func mouseModifier(mods Mods) int {
	mod := 0
	if mods.Shift {
		mod += sgrShiftFlag
	}
	if mods.Alt {
		mod += sgrAltFlag
	}
	if mods.Ctrl {
		mod += sgrCtrlFlag
	}
	return mod
}

func sgrSequence(button, x, y int, final string) []byte {
	if x < 1 {
		x = 1
	}
	if y < 1 {
		y = 1
	}
	return []byte("\x1b[<" + strconv.Itoa(button) + ";" + strconv.Itoa(x) + ";" + strconv.Itoa(y) + final)
}

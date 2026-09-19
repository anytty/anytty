package keys

import "testing"

func TestEncodeMouseGolden(t *testing.T) {
	tests := []struct {
		name string
		ev   Event
		want string
	}{
		{"left press", Event{Kind: KindMouse, Action: ActionPress, Button: ButtonLeft, X: 4, Y: 7}, "\x1b[<0;4;7M"},
		{"left release", Event{Kind: KindMouse, Action: ActionRelease, Button: ButtonLeft, X: 4, Y: 7}, "\x1b[<0;4;7m"},
		{"middle press", Event{Kind: KindMouse, Action: ActionPress, Button: ButtonMiddle, X: 1, Y: 2}, "\x1b[<1;1;2M"},
		{"right drag", Event{Kind: KindMouse, Action: ActionDrag, Button: ButtonRight, X: 9, Y: 3}, "\x1b[<34;9;3M"},
		{"left drag", Event{Kind: KindMouse, Action: ActionDrag, Button: ButtonLeft, X: 2, Y: 2}, "\x1b[<32;2;2M"},
		{"motion without button", Event{Kind: KindMouse, Action: ActionDrag, Button: ButtonNone, X: 2, Y: 2}, "\x1b[<35;2;2M"},
		{"generic release", Event{Kind: KindMouse, Action: ActionRelease, Button: ButtonNone, X: 2, Y: 2}, "\x1b[<3;2;2m"},
		{"shift+ctrl left press", Event{Kind: KindMouse, Action: ActionPress, Button: ButtonLeft, X: 5, Y: 6, Mods: Mods{Shift: true, Ctrl: true}}, "\x1b[<20;5;6M"},
		{"alt middle press", Event{Kind: KindMouse, Action: ActionPress, Button: ButtonMiddle, X: 5, Y: 6, Mods: Mods{Alt: true}}, "\x1b[<9;5;6M"},
		{"clamp coords", Event{Kind: KindMouse, Action: ActionPress, Button: ButtonLeft, X: 0, Y: -3}, "\x1b[<0;1;1M"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := EncodeMouse(tc.ev)
			if !ok {
				t.Fatalf("EncodeMouse(%+v) not encodable", tc.ev)
			}
			if string(got) != tc.want {
				t.Fatalf("EncodeMouse(%+v) = %q, want %q", tc.ev, got, tc.want)
			}
		})
	}
}

func TestEncodeWheelGolden(t *testing.T) {
	up, ok := EncodeWheel(Event{Kind: KindWheel, Delta: 1, X: 3, Y: 5})
	if !ok || string(up) != "\x1b[<64;3;5M" {
		t.Fatalf("wheel up = %q ok=%v, want ESC[<64;3;5M", up, ok)
	}
	down, ok := EncodeWheel(Event{Kind: KindWheel, Delta: -2, X: 3, Y: 5})
	if !ok || string(down) != "\x1b[<65;3;5M" {
		t.Fatalf("wheel down = %q ok=%v, want ESC[<65;3;5M", down, ok)
	}
	ctrlUp, ok := EncodeWheel(Event{Kind: KindWheel, Delta: 1, X: 1, Y: 1, Mods: Mods{Ctrl: true}})
	if !ok || string(ctrlUp) != "\x1b[<80;1;1M" {
		t.Fatalf("ctrl wheel = %q ok=%v, want ESC[<80;1;1M", ctrlUp, ok)
	}
}

func TestEncodeMouseRejects(t *testing.T) {
	for _, ev := range []Event{
		{Kind: KindWheel, Delta: 0},
		{Kind: KindMouse, Action: ActionPress, Button: "side"},
		{Kind: KindMouse, Action: "hover", Button: ButtonLeft},
		{Kind: KindPaste, Text: "a"},
	} {
		if _, ok := Encode(ev); ok {
			t.Fatalf("Encode(%+v) should fail", ev)
		}
	}
}

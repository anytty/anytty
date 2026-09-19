package keys

import "testing"

func TestName(t *testing.T) {
	tests := []struct {
		name string
		ev   Event
		want string
	}{
		{"ctrl name", Event{Kind: KindKey, Key: "ctrl-p"}, "ctrl-p"},
		{"ctrl from char", Event{Kind: KindKey, Char: "Z", Mods: Mods{Ctrl: true}}, "ctrl-z"},
		{"ctrl from lower char flag", Event{Kind: KindKey, Key: "a", Mods: Mods{Ctrl: true}}, "ctrl-a"},
		{"printable keeps case", Event{Kind: KindKey, Key: "A"}, "A"},
		{"utf8 char", Event{Kind: KindKey, Key: "中"}, "中"},
		{"alias pgup", Event{Kind: KindKey, Key: "pgup"}, "page-up"},
		{"alias escape", Event{Kind: KindKey, Key: "escape"}, "esc"},
		{"named key", Event{Kind: KindKey, Key: "shift-tab"}, "shift-tab"},
		{"paste has no name", Event{Kind: KindPaste, Text: "x"}, ""},
		{"ctrl alt keeps ctrl name", Event{Kind: KindKey, Key: "c", Mods: Mods{Ctrl: true, Alt: true}}, "ctrl-c"},
		{"ctrl shift keeps both", Event{Kind: KindKey, Key: "v", Mods: Mods{Ctrl: true, Shift: true}}, "ctrl-shift-v"},
		{"ctrl shift from char", Event{Kind: KindKey, Char: "C", Mods: Mods{Ctrl: true, Shift: true}}, "ctrl-shift-c"},
		{"ctrl alt digit", Event{Kind: KindKey, Key: "3", Mods: Mods{Ctrl: true, Alt: true}}, "ctrl-alt-3"},
		{"ctrl alt shift letter keeps base", Event{Kind: KindKey, Key: "x", Mods: Mods{Ctrl: true, Alt: true, Shift: true}}, "ctrl-x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Name(tc.ev); got != tc.want {
				t.Fatalf("Name(%+v) = %q, want %q", tc.ev, got, tc.want)
			}
		})
	}
}

func TestEncodeKeyGolden(t *testing.T) {
	tests := []struct {
		name string
		ev   Event
		want string
	}{
		{"printable", Event{Kind: KindKey, Key: "a"}, "a"},
		{"printable from char", Event{Kind: KindKey, Char: "?"}, "?"},
		{"utf8", Event{Kind: KindKey, Key: "中"}, "中"},
		{"shifted letter", Event{Kind: KindKey, Char: "A"}, "A"},
		{"ctrl-a name", Event{Kind: KindKey, Key: "ctrl-a"}, "\x01"},
		{"ctrl-z from flags", Event{Kind: KindKey, Char: "z", Mods: Mods{Ctrl: true}}, "\x1a"},
		{"ctrl-space", Event{Kind: KindKey, Key: "ctrl-space"}, "\x00"},
		{"enter", Event{Kind: KindKey, Key: "enter"}, "\r"},
		{"tab", Event{Kind: KindKey, Key: "tab"}, "\t"},
		{"shift-tab", Event{Kind: KindKey, Key: "shift-tab"}, "\x1b[Z"},
		{"backspace", Event{Kind: KindKey, Key: "backspace"}, "\x7f"},
		{"esc", Event{Kind: KindKey, Key: "esc"}, "\x1b"},
		{"up", Event{Kind: KindKey, Key: "up"}, "\x1b[A"},
		{"down", Event{Kind: KindKey, Key: "down"}, "\x1b[B"},
		{"right", Event{Kind: KindKey, Key: "right"}, "\x1b[C"},
		{"left", Event{Kind: KindKey, Key: "left"}, "\x1b[D"},
		{"ctrl-up", Event{Kind: KindKey, Key: "up", Mods: Mods{Ctrl: true}}, "\x1b[1;5A"},
		{"shift+alt+right", Event{Kind: KindKey, Key: "right", Mods: Mods{Shift: true, Alt: true}}, "\x1b[1;4C"},
		{"alt+left", Event{Kind: KindKey, Key: "left", Mods: Mods{Alt: true}}, "\x1b[1;3D"},
		{"home", Event{Kind: KindKey, Key: "home"}, "\x1b[H"},
		{"end", Event{Kind: KindKey, Key: "end"}, "\x1b[F"},
		{"insert", Event{Kind: KindKey, Key: "insert"}, "\x1b[2~"},
		{"delete", Event{Kind: KindKey, Key: "delete"}, "\x1b[3~"},
		{"page-up", Event{Kind: KindKey, Key: "page-up"}, "\x1b[5~"},
		{"page-down", Event{Kind: KindKey, Key: "page-down"}, "\x1b[6~"},
		{"ctrl+page-down", Event{Kind: KindKey, Key: "page-down", Mods: Mods{Ctrl: true}}, "\x1b[6;5~"},
		{"f1", Event{Kind: KindKey, Key: "f1"}, "\x1bOP"},
		{"f4", Event{Kind: KindKey, Key: "f4"}, "\x1bOS"},
		{"f5", Event{Kind: KindKey, Key: "f5"}, "\x1b[15~"},
		{"f12", Event{Kind: KindKey, Key: "f12"}, "\x1b[24~"},
		{"ctrl+f1", Event{Kind: KindKey, Key: "f1", Mods: Mods{Ctrl: true}}, "\x1b[1;5P"},
		{"alt+x", Event{Kind: KindKey, Key: "x", Mods: Mods{Alt: true}}, "\x1bx"},
		{"alt+enter", Event{Kind: KindKey, Key: "enter", Mods: Mods{Alt: true}}, "\x1b\r"},
		{"alt+backspace", Event{Kind: KindKey, Key: "backspace", Mods: Mods{Alt: true}}, "\x1b\x7f"},
		{"alt+ctrl+a", Event{Kind: KindKey, Key: "ctrl-a", Mods: Mods{Alt: true}}, "\x1b\x01"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := EncodeKey(tc.ev)
			if !ok {
				t.Fatalf("EncodeKey(%+v) not encodable", tc.ev)
			}
			if string(got) != tc.want {
				t.Fatalf("EncodeKey(%+v) = %q, want %q", tc.ev, got, tc.want)
			}
		})
	}
}

func TestEncodeKeyRejects(t *testing.T) {
	for _, ev := range []Event{
		{Kind: KindPaste, Text: "x"},
		{Kind: KindKey},
		{Kind: KindKey, Key: "ctrl-中"},
	} {
		if got, ok := EncodeKey(ev); ok {
			t.Fatalf("EncodeKey(%+v) = %q, want not encodable", ev, got)
		}
	}
}

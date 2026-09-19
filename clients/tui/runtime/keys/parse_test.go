package keys

import (
	"testing"
	"time"
)

func TestParserFeedBasics(t *testing.T) {
	p := NewParser()
	events := p.Feed([]byte("hi\r\t\x7f\x11"))
	if len(events) != 6 {
		t.Fatalf("events = %d, want 6: %+v", len(events), events)
	}
	if Name(events[0]) != "h" || Name(events[1]) != "i" {
		t.Fatalf("printable = %q,%q", Name(events[0]), Name(events[1]))
	}
	if Name(events[2]) != "enter" || Name(events[3]) != "tab" || Name(events[4]) != "backspace" || Name(events[5]) != "ctrl-q" {
		t.Fatalf("named = %q,%q,%q,%q", Name(events[2]), Name(events[3]), Name(events[4]), Name(events[5]))
	}
	ctrl := p.Feed([]byte{0x01, 0x1b})
	if len(ctrl) != 1 || Name(ctrl[0]) != "ctrl-a" {
		t.Fatalf("ctrl events = %+v", ctrl)
	}
	if p.Pending() != 1 {
		t.Fatalf("pending = %d, want the lone ESC buffered", p.Pending())
	}
}

func TestParserSplitsUTF8AcrossReads(t *testing.T) {
	p := NewParser()
	if events := p.Feed([]byte{0xe4, 0xb8}); len(events) != 0 {
		t.Fatalf("half rune produced %+v", events)
	}
	events := p.Feed([]byte{0xad})
	if len(events) != 1 || Name(events[0]) != "中" {
		t.Fatalf("events = %+v", events)
	}
}

func TestParserSpecialKeys(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"\x1b[A", "up"},
		{"\x1b[B", "down"},
		{"\x1b[1;5C", "right"},
		{"\x1b[H", "home"},
		{"\x1b[3~", "delete"},
		{"\x1b[5~", "page-up"},
		{"\x1b[6~", "page-down"},
		{"\x1b[Z", "shift-tab"},
		{"\x1bOP", "f1"},
		{"\x1b[15~", "f5"},
		// Kitty/CSI-u enhanced keys carry the modifiers the v3 recommended
		// profile binds (ctrl-shift-c/v/h, ctrl-alt-1..5).
		{"\x1b[118;6u", "ctrl-shift-v"},
		{"\x1b[99;6u", "ctrl-shift-c"},
		{"\x1b[49;7u", "ctrl-alt-1"},
	}
	for _, tc := range cases {
		p := NewParser()
		events := p.Feed([]byte(tc.raw))
		if len(events) != 1 {
			t.Fatalf("%q produced %+v", tc.raw, events)
		}
		if got := Name(events[0]); got != tc.want {
			t.Fatalf("%q = %q, want %q", tc.raw, got, tc.want)
		}
	}
	p := NewParser()
	events := p.Feed([]byte("\x1b[1;5C"))
	if events[0].Mods.Ctrl != true {
		t.Fatalf("ctrl-right mods = %+v", events[0].Mods)
	}
}

func TestParserMouseAndWheel(t *testing.T) {
	p := NewParser()
	events := p.Feed([]byte("\x1b[<0;10;5M\x1b[<32;12;5M\x1b[<0;12;5m\x1b[<64;3;4M\x1b[<65;3;4M"))
	if len(events) != 5 {
		t.Fatalf("events = %+v", events)
	}
	press, drag, release, wheelUp, wheelDown := events[0], events[1], events[2], events[3], events[4]
	if press.Kind != KindMouse || press.Action != ActionPress || press.Button != ButtonLeft || press.X != 10 || press.Y != 5 {
		t.Fatalf("press = %+v", press)
	}
	if drag.Action != ActionDrag || drag.X != 12 {
		t.Fatalf("drag = %+v", drag)
	}
	if release.Action != ActionRelease || release.Button != ButtonLeft {
		t.Fatalf("release = %+v", release)
	}
	if wheelUp.Kind != KindWheel || wheelUp.Delta != 1 || wheelUp.X != 3 || wheelUp.Y != 4 {
		t.Fatalf("wheel up = %+v", wheelUp)
	}
	if wheelDown.Kind != KindWheel || wheelDown.Delta != -1 {
		t.Fatalf("wheel down = %+v", wheelDown)
	}
}

func TestParserBracketedPaste(t *testing.T) {
	p := NewParser()
	first := p.Feed([]byte("\x1b[200~echo one\necho two"))
	if len(first) != 0 {
		t.Fatalf("incomplete paste produced %+v", first)
	}
	events := p.Feed([]byte("\x1b[201~"))
	if len(events) != 1 || events[0].Kind != KindPaste || events[0].Text != "echo one\necho two" {
		t.Fatalf("paste = %+v", events)
	}
}

func TestParserAltAndOSC(t *testing.T) {
	p := NewParser()
	events := p.Feed([]byte("\x1bx\x1b]11;rgb:00/00/00\x07"))
	if len(events) != 1 {
		t.Fatalf("events = %+v", events)
	}
	if events[0].Char != "x" || !events[0].Mods.Alt {
		t.Fatalf("alt-x = %+v", events[0])
	}
	if p.Pending() != 0 {
		t.Fatalf("OSC must be consumed, pending=%d", p.Pending())
	}
}

func TestParserFlushEscape(t *testing.T) {
	p := NewParser()
	p.Feed([]byte("\x1b"))
	if events := p.Flush(time.Hour); len(events) != 0 {
		t.Fatalf("early flush = %+v", events)
	}
	events := p.Flush(0)
	if len(events) != 1 || Name(events[0]) != "esc" {
		t.Fatalf("flush = %+v", events)
	}
	if p.Pending() != 0 {
		t.Fatalf("pending after flush = %d", p.Pending())
	}

	p = NewParser()
	events = p.Feed([]byte("\x1bA"))
	if len(events) != 1 || events[0].Char != "A" || !events[0].Mods.Alt {
		t.Fatalf("alt-a = %+v", events)
	}
	if events = p.Flush(0); len(events) != 0 {
		t.Fatalf("flush after alt = %+v", events)
	}
}

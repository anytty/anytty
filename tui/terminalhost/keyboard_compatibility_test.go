package terminalhost

import (
	"fmt"
	"testing"

	"github.com/anytty/anytty/tui/input"
)

func TestControlAliasesAcrossKeyboardProtocols(t *testing.T) {
	aliases := map[rune]byte{'/': 31, '2': 0, '3': 27, '4': 28, '5': 29, '6': 30, '7': 31, '8': 127, ' ': 0, '@': 0, '[': 27, '\\': 28, ']': 29, '^': 30, '_': 31, '?': 127}
	for char, want := range aliases {
		for _, alt := range []bool{false, true} {
			modifier := 5
			output := []byte{want}
			if alt {
				modifier = 7
				output = append([]byte{27}, output...)
			}
			for _, seq := range []string{
				fmt.Sprintf("\x1b[%d;%du", char, modifier),
				fmt.Sprintf("\x1b[%d;%d:2u", char, modifier),
				fmt.Sprintf("\x1b[%d::%d;%du", char, char, modifier),
				fmt.Sprintf("\x1b[27;%d;%d~", modifier, char),
			} {
				assertTerminalBytes(t, seq, string(output))
			}
		}
	}
}

func TestModifiedFunctionKeysReachTerminal(t *testing.T) {
	for modifier := 1; modifier <= 8; modifier++ {
		for _, final := range "PQS" {
			seq := fmt.Sprintf("\x1b[1;%d%c", modifier, final)
			assertTerminalBytes(t, seq, seq)
		}
		seq := fmt.Sprintf("\x1b[13;%d~", modifier)
		assertTerminalBytes(t, seq, seq)
	}
	// CSI R is indistinguishable from a cursor-position response. The supported
	// F3 encodings are SS3 R and CSI 13~, not arbitrary cursor reports.
	assertTerminalBytes(t, "\x1bOR", "\x1bOR")
	for _, seq := range []string{"\x1b[1;5R", "\x1b[24;80R", "\x1b[?1;5R", "\x1b[>1;5P", "\x1b[1;5;9Q"} {
		assertTerminalBytes(t, seq, "")
	}
}

func TestExtendedKeyboardTextKeepsShortcutIdentity(t *testing.T) {
	for _, tc := range []struct{ seq, want string }{
		{"\x1b[97;2u", "A"},
		{"\x1b[47:63;2u", "?"},
		{"\x1b[47:63;6u", "\x7f"},
		{"\x1b[97;1;22909:20320u", "好你"},
		{"\x1b[97;;22909u", "好"},
		{"\x1b[47;5;47u", "\x1f"},
		{"\x1b[47::47;5u", "\x1f"},
		{"\x1b[47;5:3u", ""},
		{"\x1b[47:63;5u", ""}, // shifted field without Shift is malformed
		{"\x1b[47::55296;5u", ""},
		{"\x1b[97;1;27u", ""},
		{"\x1b[97;1;57376u", ""},
		{"\x1b[27;9;47~", ""}, // unsupported Super must not leak raw bytes
		{"\x1b[27;5;55296~", ""},
		{"\x1b[27;6;47:63~", ""},
		{"\x1b[27;5:2;47~", ""},
	} {
		assertTerminalBytes(t, tc.seq, tc.want)
	}
	event := NewInputParser().Feed([]byte("\x1b[110:78;2u"))[0]
	if event.Char != "n" || event.ShiftedChar != "N" {
		t.Fatalf("shortcut identity changed: %+v", event)
	}
}

func assertTerminalBytes(t *testing.T, seq, want string) {
	t.Helper()
	// Every byte boundary must produce the same result; reads need not align
	// with a terminal escape sequence.
	for split := 0; split <= len(seq); split++ {
		parser := NewInputParser()
		events := parser.Feed([]byte(seq[:split]))
		events = append(events, parser.Feed([]byte(seq[split:]))...)
		events = append(events, parser.FlushEscape()...)
		if len(events) != 1 {
			t.Fatalf("%q split %d: %d events", seq, split, len(events))
		}
		intent := input.RouteWithOptions(events[0], input.RouteOptions{ForceTerminalPassthrough: true})
		if string(intent.Bytes) != want {
			t.Fatalf("%q split %d: output %q want %q event=%+v", seq, split, intent.Bytes, want, events[0])
		}
	}
}

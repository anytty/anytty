package render

import "testing"

func TestCursorPosition(t *testing.T) {
	tests := []struct {
		x, y int
		want string
	}{
		{0, 0, "\x1b[1;1H"},
		{4, 2, "\x1b[3;5H"},
		{-3, -1, "\x1b[1;1H"},
	}
	for _, tt := range tests {
		if got := CursorPosition(tt.x, tt.y); got != tt.want {
			t.Fatalf("CursorPosition(%d,%d) = %q, want %q", tt.x, tt.y, got, tt.want)
		}
	}
}

func TestScreenSequences(t *testing.T) {
	wantEnter := "\x1b[?1049h\x1b[?25l\x1b[?2004h\x1b[?1000h\x1b[?1002h\x1b[?1006h"
	if got := EnterScreen(); got != wantEnter {
		t.Fatalf("EnterScreen = %q, want %q", got, wantEnter)
	}
	wantExit := "\x1b[?1006l\x1b[?1002l\x1b[?1000l\x1b[?2004l\x1b[?25h\x1b[?1049l"
	if got := ExitScreen(); got != wantExit {
		t.Fatalf("ExitScreen = %q, want %q", got, wantExit)
	}
}

func TestEnableDisableMouse(t *testing.T) {
	if got := EnableMouse(1000, 1002, 1006); got != "\x1b[?1000h\x1b[?1002h\x1b[?1006h" {
		t.Fatalf("EnableMouse = %q", got)
	}
	if got := DisableMouse(1006, 1002, 1000); got != "\x1b[?1006l\x1b[?1002l\x1b[?1000l" {
		t.Fatalf("DisableMouse = %q", got)
	}
	if got := EnableMouse(9999); got != "" {
		t.Fatalf("EnableMouse(unknown) = %q, want empty", got)
	}
}

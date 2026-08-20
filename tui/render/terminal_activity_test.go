package render

import (
	"testing"
	"time"
)

func TestTerminalOutputActivityLabel(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		at   time.Time
		want string
	}{
		{name: "never output", at: time.Time{}, want: ""},
		{name: "just now", at: now.Add(-2 * time.Second), want: "now"},
		{name: "seconds", at: now.Add(-42 * time.Second), want: "42s"},
		{name: "minutes", at: now.Add(-5 * time.Minute), want: "5m"},
		{name: "hours", at: now.Add(-3 * time.Hour), want: "3h"},
		{name: "future clamps to now", at: now.Add(time.Minute), want: "now"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := TerminalOutputActivityLabel(test.at, now); got != test.want {
				t.Fatalf("TerminalOutputActivityLabel(%v, now) = %q, want %q", test.at, got, test.want)
			}
		})
	}
}

func TestTerminalOutputActivityStyle(t *testing.T) {
	if TerminalOutputActivityStyle("now") != StyleInfo {
		t.Fatalf("expected fresh activity to use info style, got %q", TerminalOutputActivityStyle("now"))
	}
	if TerminalOutputActivityStyle("5m") != StyleMuted {
		t.Fatalf("expected quiet activity to use muted style, got %q", TerminalOutputActivityStyle("5m"))
	}
	if TerminalOutputActivityStyle("") != StyleMuted {
		t.Fatalf("expected empty activity to use muted style, got %q", TerminalOutputActivityStyle(""))
	}
}

package vt

import (
	"fmt"
	"strings"
	"testing"
)

func TestOSCTitleUTF8DoesNotTerminateAtContinuationByte(t *testing.T) {
	for _, title := range []string{
		"✓ APP历史模式下拉跳转原因 | anytty", // 9C in the second byte
		"工作 | anytty", // 9C in the third byte
		"Ü title",     // two-byte UTF-8
		"🔜 title",     // four-byte UTF-8
	} {
		for _, terminator := range []string{"\x07", "\x1b\\", "\x9c"} {
			for _, introducer := range []string{"\x1b]0;", "\x1b]2;", "\x9d2;"} {
				raw := "before" + introducer + title + terminator + "after"
				for split := 0; split <= len(raw); split++ {
					t.Run(fmt.Sprintf("%s/%x/%x/split%d", title, introducer, terminator, split), func(t *testing.T) {
						e := NewEmulator(80, 3)
						defer e.Close()
						var titles []string
						e.SetCallbacks(Callbacks{Title: func(s string) { titles = append(titles, s) }})
						for _, part := range []string{raw[:split], raw[split:]} {
							if _, err := e.Write([]byte(part)); err != nil {
								t.Fatal(err)
							}
						}
						if len(titles) != 1 || titles[0] != title {
							t.Fatalf("titles = %q, want [%q]", titles, title)
						}
						if got := strings.TrimSpace(e.String()); got != "beforeafter" {
							t.Fatalf("screen = %q, want only beforeafter", got)
						}
					})
				}
			}
		}
	}
}

func TestOSCUnicodeWorkingDirectoryAndHyperlink(t *testing.T) {
	e := NewEmulator(80, 3)
	defer e.Close()
	const url = "file://host/工作"
	var cwd string
	e.SetCallbacks(Callbacks{WorkingDirectory: func(s string) { cwd = s }})
	raw := "\x1b]7;" + url + "\x07" + "\x1b]8;;" + url + "\x1b\\linked\x1b]8;;\x07 tail"
	for i := range len(raw) {
		if _, err := e.Write([]byte(raw[i : i+1])); err != nil {
			t.Fatal(err)
		}
	}
	if cwd != url {
		t.Fatalf("cwd = %q, want %q", cwd, url)
	}
	if got := strings.TrimSpace(e.String()); got != "linked tail" {
		t.Fatalf("screen = %q", got)
	}
	if got := e.CellAt(0, 0).Link.URL; got != url {
		t.Fatalf("hyperlink = %q, want %q", got, url)
	}
	if got := e.CellAt(7, 0).Link.URL; got != "" {
		t.Fatalf("hyperlink not reset: %q", got)
	}
}

func TestOSCTitleOverflowStillConsumesUTF8Payload(t *testing.T) {
	e := NewEmulator(80, 3)
	defer e.Close()
	raw := "before\x1b]0;" + strings.Repeat("x", emulatorParserDataBufferSize) + "✓ title overflow\x07after"
	if _, err := e.Write([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(e.String()); got != "beforeafter" {
		t.Fatalf("screen = %q", got)
	}
}

func TestOSCTerminationAndCancellationAfterUTF8(t *testing.T) {
	for _, tc := range []struct {
		name       string
		raw        string
		wantTitles string
	}{
		{"cancel", "\x1b]0;✓ canceled\x18", ""},
		{"substitute", "\x1b]0;✓ canceled\x1a", ""},
		{"new escape", "\x1b]0;✓ first\x1b[0m", "✓ first"},
		{"bare ST after text", "\x1b]0;✓ first\x9c", "✓ first"},
		{"invalid UTF8 lead", "\x1b]0;\xff\x9c", "\xff"},
		{"invalid UTF8 prefix", "\x1b]0;\xe0\x9c", "\xe0"},
		{"ASCII breaks UTF8 prefix", "\x1b]0;\xe2x\x9c", "\xe2x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewEmulator(80, 3)
			defer e.Close()
			var titles []string
			e.SetCallbacks(Callbacks{Title: func(s string) { titles = append(titles, s) }})
			raw := "before" + tc.raw + "\x1b]0;工作\x07after"
			for i := range len(raw) {
				if _, err := e.Write([]byte(raw[i : i+1])); err != nil {
					t.Fatal(err)
				}
			}
			want := "工作"
			if tc.wantTitles != "" {
				want = tc.wantTitles + "|" + want
			}
			if got := strings.Join(titles, "|"); got != want {
				t.Fatalf("titles = %q, want %q", got, want)
			}
			if got := strings.TrimSpace(e.String()); got != "beforeafter" {
				t.Fatalf("screen = %q", got)
			}
		})
	}
}

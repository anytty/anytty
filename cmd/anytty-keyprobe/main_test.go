package main

import (
	"reflect"
	"testing"

	"github.com/anytty/anytty/tui/config"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/terminalhost"
)

func TestCtrlBackslashReachesTerminal(t *testing.T) {
	for _, seq := range []string{"\x1c", "\x1b[92;5u", "\x1b[92;5:2u"} {
		row := inspect(sample{Sequence: seq}, config.Default().Shortcuts)
		if row.Status != "forwarded" || len(row.Events) != 1 || row.Events[0].OutputHex != "1c" {
			t.Fatalf("%q: %+v", seq, row)
		}
	}
}

func TestMatrixChunkBoundaries(t *testing.T) {
	for _, s := range samples() {
		parser := terminalhost.NewInputParser()
		want := parser.Feed([]byte(s.Sequence))
		want = append(want, parser.FlushEscape()...)
		for split := 1; split < len(s.Sequence); split++ {
			parser := terminalhost.NewInputParser()
			got := parser.Feed([]byte(s.Sequence[:split]))
			got = append(got, parser.Feed([]byte(s.Sequence[split:]))...)
			got = append(got, parser.FlushEscape()...)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%s %s split %d: got %#v want %#v", s.Family, s.Key, split, got, want)
			}
		}
	}
}

func TestMatrixSeparatesShortcutFromMissingEncoding(t *testing.T) {
	defaults := config.Default().Shortcuts
	shortcut := inspect(sample{Sequence: "\x1b[112;5u"}, defaults)
	if shortcut.Status != "shortcut" || shortcut.Events[0].ForcedOutputHex != "10" {
		t.Fatalf("ctrl-p: %+v", shortcut)
	}
	missing := inspect(sample{Sequence: "\x1b[13;5u"}, defaults)
	if missing.Status != "no-terminal-encoding" {
		t.Fatalf("ctrl-enter: %+v", missing)
	}
	function := inspect(sample{Sequence: "\x1b[1;5P"}, defaults)
	if function.Status != "forwarded" || function.Events[0].Kind != input.EventKindKey {
		t.Fatalf("ctrl-f1: %+v", function)
	}
}

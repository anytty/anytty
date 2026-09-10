package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
	"github.com/anytty/anytty/tui/terminalhost"
)

type sample struct {
	Family   string `json:"family"`
	Key      string `json:"key"`
	Sequence string `json:"-"`
}
type matrixRow struct {
	sample
	RawHex string        `json:"raw_hex"`
	Status string        `json:"status"`
	Events []eventReport `json:"events"`
}

func samples() []sample {
	var out []sample
	add := func(family, key, seq string) { out = append(out, sample{family, key, seq}) }
	// Every ASCII byte and its traditional Alt-prefixed encoding, including C0.
	for code := 0; code <= 127; code++ {
		add("legacy", fmt.Sprintf("byte-%02x", code), string(rune(code)))
		add("legacy-alt", fmt.Sprintf("alt-byte-%02x", code), "\x1b"+string(rune(code)))
	}
	// All printable ASCII characters, named controls and functional keys with
	// every combination of Shift/Alt/Ctrl. These are protocol inputs, not OS key presses.
	codes := []int{9, 13, 27, 127}
	for code := 32; code <= 126; code++ {
		codes = append(codes, code)
	}
	// Only functional keys whose specified wire format is CSI-u: locks,
	// Print Screen, Pause, Menu, and F13-F24. F1-F12/navigation use CSI/SS3 below.
	for code := 57358; code <= 57363; code++ {
		codes = append(codes, code)
	}
	for code := 57376; code <= 57387; code++ {
		codes = append(codes, code)
	}
	for _, code := range codes {
		for mask := 0; mask < 8; mask++ {
			name := fmt.Sprintf("U+%04X/shift=%t/alt=%t/ctrl=%t", code, mask&1 != 0, mask&2 != 0, mask&4 != 0)
			add("csi-u", name, fmt.Sprintf("\x1b[%d;%du", code, mask+1))
		}
	}
	for mask := 0; mask < 8; mask++ {
		for _, final := range "ABCDFHPQRSZ" {
			add("csi-function", fmt.Sprintf("%c/modifier=%d", final, mask+1), fmt.Sprintf("\x1b[1;%d%c", mask+1, final))
		}
		for _, code := range []int{1, 2, 3, 4, 5, 6, 7, 8, 11, 12, 13, 14, 15, 17, 18, 19, 20, 21, 23, 24} {
			add("csi-tilde", fmt.Sprintf("%d/modifier=%d", code, mask+1), fmt.Sprintf("\x1b[%d;%d~", code, mask+1))
		}
	}
	for _, final := range "ABCDFHPQRS" {
		add("ss3", string(final), "\x1bO"+string(final))
	}
	for _, code := range codes[:99] {
		for mask := 0; mask < 8; mask++ {
			add("modify-other-keys", fmt.Sprintf("U+%04X/modifier=%d", code, mask+1), fmt.Sprintf("\x1b[27;%d;%d~", mask+1, code))
		}
	}
	for _, seq := range []string{"\x1b[92;5u", "\x1b[92;5:2u", "\x1b[92;5:3u", "\x1b[92::92;5u", "\x1b[92;5;92u"} {
		add("protocol-variants", "ctrl-backslash", seq)
	}
	return out
}

func inspect(s sample, shortcuts state.TUIShortcutConfig) matrixRow {
	parser := terminalhost.NewInputParser()
	events := parser.Feed([]byte(s.Sequence))
	events = append(events, parser.FlushEscape()...)
	row := matrixRow{sample: s, RawHex: hex.EncodeToString([]byte(s.Sequence)), Status: "no-event"}
	for _, event := range events {
		row.Events = append(row.Events, describe(event, shortcuts))
	}
	if len(row.Events) != 1 {
		if len(row.Events) > 1 {
			row.Status = "multiple-events"
		}
		return row
	}
	event := row.Events[0]
	switch {
	case s.Family == "protocol-variants" && strings.HasSuffix(s.Sequence, ";5:3u"):
		row.Status = "release-ignored"
	case event.Kind != input.EventKindKey || event.Key == input.KeyUnknown:
		row.Status = "parser-unsupported"
	case event.Route == input.IntentShortcutAction:
		row.Status = "shortcut"
	case event.OutputHex != "":
		row.Status = "forwarded"
	default:
		row.Status = "no-terminal-encoding"
	}
	return row
}

func writeMatrix(w io.Writer, shortcuts state.TUIShortcutConfig) error {
	encoder := json.NewEncoder(w)
	counts := map[string]int{}
	for _, s := range samples() {
		row := inspect(s, shortcuts)
		counts[row.Status]++
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}
	summary, _ := json.Marshal(counts)
	fmt.Fprintf(os.Stderr, "Synthetic protocol matrix: %d cases %s\n", len(samples()), summary)
	return nil
}

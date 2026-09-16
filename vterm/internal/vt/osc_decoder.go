package vt

import (
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
)

// oscDecoder owns only OSC payload framing. x/ansi v0.11.8 treats every 0x9c
// in an OSC as ST, including the continuation bytes in characters like ✓ and 作.
// Keep the normal parser for the introducer and all other control sequences.
type oscDecoder struct {
	data    []byte
	prefix  [utf8.UTFMax]byte
	prefixN int
}

func (e *Emulator) advanceParser(b byte) {
	if e.parser.State() != parser.OscStringState {
		e.parser.Advance(b)
		return
	}
	if e.osc.continuation(b) {
		e.osc.put(b)
		return
	}
	switch b {
	case ansi.BEL, ansi.ST, ansi.ESC:
		cmd := parser.MissingCommand
		for _, digit := range e.osc.data {
			if digit < '0' || digit > '9' {
				break
			}
			if cmd == parser.MissingCommand {
				cmd = 0
			}
			cmd = cmd*10 + int(digit-'0')
		}
		e.handleOsc(cmd, e.osc.data)
		e.osc.reset()
		e.parser.Reset()
		if b == ansi.ESC {
			// ESC terminates OSC and begins the next escape sequence (usually ST).
			e.parser.Advance(b)
		}
	case ansi.CAN, ansi.SUB:
		e.osc.reset()
		e.parser.Reset()
	default:
		if b >= 0x20 {
			if b >= 0xc2 && b <= 0xf4 {
				e.osc.prefix[0], e.osc.prefixN = b, 1
			}
			e.osc.put(b)
		}
	}
}

func (osc *oscDecoder) continuation(b byte) bool {
	if osc.prefixN == 0 {
		return false
	}
	if b&0xc0 != 0x80 {
		osc.prefixN = 0
		return false
	}
	osc.prefix[osc.prefixN] = b
	osc.prefixN++
	prefix := osc.prefix[:osc.prefixN]
	if !utf8.FullRune(prefix) {
		return true
	}
	_, size := utf8.DecodeRune(prefix)
	osc.prefixN = 0
	return size > 1
}

func (osc *oscDecoder) put(b byte) {
	// Match the existing parser's limit, but keep consuming and tracking UTF-8
	// after overflow so a long title can never spill into the visible grid.
	if len(osc.data) < emulatorParserDataBufferSize {
		osc.data = append(osc.data, b)
	}
}

func (osc *oscDecoder) reset() {
	osc.data = osc.data[:0]
	osc.prefixN = 0
}

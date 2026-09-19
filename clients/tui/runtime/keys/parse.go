package keys

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Parser turns raw TTY input bytes into normalized events (the inverse of
// Encode). It keeps incomplete UTF-8, CSI, OSC and bracketed-paste sequences
// across reads; a lone Esc needs a caller-driven timeout, see Flush.
type Parser struct {
	pending []byte
	since   time.Time
}

// NewParser returns an empty parser. One instance must live for the whole
// session so pending bytes are never lost between reads.
func NewParser() *Parser { return &Parser{} }

// Pending reports the number of buffered bytes of an incomplete sequence.
func (p *Parser) Pending() int { return len(p.pending) }

// Feed appends a raw chunk and returns every complete event. A trailing
// incomplete sequence stays buffered.
func (p *Parser) Feed(chunk []byte) []Event {
	if len(chunk) == 0 {
		return nil
	}
	p.pending = append(p.pending, chunk...)
	p.since = time.Now()
	var events []Event
	for len(p.pending) > 0 {
		event, consumed, complete := parseOne(p.pending)
		if !complete {
			break
		}
		p.pending = p.pending[consumed:]
		if event.Kind == 0 && event.Key == "" && event.Char == "" && event.Text == "" {
			continue
		}
		events = append(events, event)
	}
	if len(p.pending) == 0 {
		p.pending = nil
	}
	return events
}

// Flush emits a buffered Esc (or Alt+char) once timeout passed without more
// bytes. Longer incomplete sequences keep waiting.
func (p *Parser) Flush(timeout time.Duration) []Event {
	if len(p.pending) == 0 || p.pending[0] != 0x1b || time.Since(p.since) < timeout {
		return nil
	}
	switch len(p.pending) {
	case 1:
		p.pending = nil
		return []Event{{Kind: KindKey, Key: "esc"}}
	case 2:
		r, size := utf8.DecodeRune(p.pending[1:])
		if r == utf8.RuneError && size <= 1 {
			return nil
		}
		p.pending = nil
		return []Event{charEvent(string(r), Mods{Alt: true})}
	default:
		return nil
	}
}

// parseOne decodes the first event in buffer. complete=false means more bytes
// are needed; otherwise consumed bytes are removed.
func parseOne(buffer []byte) (Event, int, bool) {
	first := buffer[0]
	switch {
	case first == '\r' || first == '\n':
		return keyEvent("enter"), 1, true
	case first == '\t':
		return keyEvent("tab"), 1, true
	case first == 0x7f:
		return keyEvent("backspace"), 1, true
	case first == 0x1b:
		return parseEscape(buffer)
	case first < 0x20:
		return keyEvent(controlName(first)), 1, true
	}
	r, size := utf8.DecodeRune(buffer)
	if r == utf8.RuneError && !utf8.FullRune(buffer) {
		return Event{}, 0, false
	}
	if r == utf8.RuneError {
		return Event{}, 1, true
	}
	return charEvent(string(r), Mods{}), size, true
}

func keyEvent(name string) Event { return Event{Kind: KindKey, Key: name} }

func charEvent(char string, mods Mods) Event {
	return Event{Kind: KindKey, Key: char, Char: char, Mods: mods}
}

// controlName maps a C0 byte to its normalized key name.
func controlName(b byte) string {
	switch b {
	case 0x00:
		return "ctrl-space"
	case 0x1c:
		return "ctrl-\\"
	case 0x1d:
		return "ctrl-]"
	case 0x1e:
		return "ctrl-^"
	case 0x1f:
		return "ctrl-_"
	}
	if b >= 0x01 && b <= 0x1a {
		return "ctrl-" + string(rune('a'+b-1))
	}
	return "ctrl-space"
}

func parseEscape(buffer []byte) (Event, int, bool) {
	if len(buffer) == 1 {
		return Event{}, 0, false
	}
	switch buffer[1] {
	case '[':
		return parseCSI(buffer)
	case 'O':
		return parseSS3(buffer)
	case ']':
		return parseOSC(buffer)
	}
	r, size := utf8.DecodeRune(buffer[1:])
	if r == utf8.RuneError && !utf8.FullRune(buffer[1:]) {
		return Event{}, 0, false
	}
	if r == utf8.RuneError {
		return Event{}, 2, true
	}
	if r < 0x20 {
		return keyEvent(controlName(byte(r))), 1 + size, true
	}
	return charEvent(string(r), Mods{Alt: true}), 1 + size, true
}

const bracketedPasteStart = "\x1b[200~"

func parseCSI(buffer []byte) (Event, int, bool) {
	if len(buffer) < 3 {
		return Event{}, 0, false
	}
	if strings.HasPrefix(string(buffer), bracketedPasteStart) {
		end := strings.Index(string(buffer[len(bracketedPasteStart):]), "\x1b[201~")
		if end < 0 {
			return Event{}, 0, false
		}
		start := len(bracketedPasteStart)
		text := string(buffer[start : start+end])
		return Event{Kind: KindPaste, Text: text}, start + end + len("\x1b[201~"), true
	}
	if buffer[2] == '<' {
		return parseSGRMouse(buffer)
	}
	for i := 2; i < len(buffer); i++ {
		if buffer[i] >= 0x40 && buffer[i] <= 0x7e {
			return parseCSIKey(buffer[:i+1]), i + 1, true
		}
	}
	return Event{}, 0, false
}

func parseSS3(buffer []byte) (Event, int, bool) {
	if len(buffer) < 3 {
		return Event{}, 0, false
	}
	name := ""
	switch buffer[2] {
	case 'A':
		name = "up"
	case 'B':
		name = "down"
	case 'C':
		name = "right"
	case 'D':
		name = "left"
	case 'F':
		name = "end"
	case 'H':
		name = "home"
	case 'P':
		name = "f1"
	case 'Q':
		name = "f2"
	case 'R':
		name = "f3"
	case 'S':
		name = "f4"
	default:
		return Event{}, 3, true
	}
	return keyEvent(name), 3, true
}

// parseOSC consumes a terminal control response (host capability / theme
// replies); such sequences are never user input and must not reach a PTY.
func parseOSC(buffer []byte) (Event, int, bool) {
	for i := 2; i < len(buffer); i++ {
		if buffer[i] == 0x07 {
			return Event{}, i + 1, true
		}
		if buffer[i] == '\\' && i > 2 && buffer[i-1] == 0x1b {
			return Event{}, i + 1, true
		}
	}
	return Event{}, 0, false
}

func parseCSIKey(seq []byte) Event {
	if len(seq) < 3 {
		return Event{}
	}
	body := string(seq[2 : len(seq)-1])
	params := strings.Split(body, ";")
	final := seq[len(seq)-1]
	mods := modsFromCSI(params)
	switch final {
	case 'A', 'B', 'C', 'D':
		return Event{Kind: KindKey, Key: map[string]string{"A": "up", "B": "down", "C": "right", "D": "left"}[string(final)], Mods: mods}
	case 'H', 'F':
		return Event{Kind: KindKey, Key: map[string]string{"H": "home", "F": "end"}[string(final)], Mods: mods}
	case 'Z':
		return Event{Kind: KindKey, Key: "shift-tab", Mods: mods}
	case 'P', 'Q', 'R', 'S':
		return Event{Kind: KindKey, Key: "f" + string(rune('1'+final-'P')), Mods: mods}
	case '~':
		code, _ := strconv.Atoi(params[0])
		return Event{Kind: KindKey, Key: tildeName(code), Mods: mods}
	case 'u':
		if len(params) > 0 {
			if code, err := strconv.Atoi(params[0]); err == nil && code >= 0x20 && code <= utf8.MaxRune {
				return charEvent(string(rune(code)), mods)
			}
		}
	}
	return Event{}
}

func modsFromCSI(params []string) Mods {
	if len(params) < 2 {
		return Mods{}
	}
	value, err := strconv.Atoi(params[1])
	if err != nil || value <= 1 {
		return Mods{}
	}
	mask := value - 1
	return Mods{Shift: mask&1 != 0, Alt: mask&2 != 0, Ctrl: mask&4 != 0}
}

func tildeName(code int) string {
	switch code {
	case 1, 7:
		return "home"
	case 2:
		return "insert"
	case 3:
		return "delete"
	case 4, 8:
		return "end"
	case 5:
		return "page-up"
	case 6:
		return "page-down"
	case 11:
		return "f1"
	case 12:
		return "f2"
	case 13:
		return "f3"
	case 14:
		return "f4"
	case 15:
		return "f5"
	case 17:
		return "f6"
	case 18:
		return "f7"
	case 19:
		return "f8"
	case 20:
		return "f9"
	case 21:
		return "f10"
	case 23:
		return "f11"
	case 24:
		return "f12"
	}
	return ""
}

func parseSGRMouse(buffer []byte) (Event, int, bool) {
	end := -1
	for i := 3; i < len(buffer); i++ {
		if buffer[i] == 'M' || buffer[i] == 'm' {
			end = i
			break
		}
	}
	if end < 0 {
		return Event{}, 0, false
	}
	consumed := end + 1
	parts := strings.Split(string(buffer[3:end]), ";")
	if len(parts) != 3 {
		return Event{}, consumed, true
	}
	code, errCode := strconv.Atoi(parts[0])
	x, errX := strconv.Atoi(parts[1])
	y, errY := strconv.Atoi(parts[2])
	if errCode != nil || errX != nil || errY != nil || x <= 0 || y <= 0 {
		return Event{}, consumed, true
	}
	if code&64 != 0 {
		delta := 1
		if code&1 != 0 {
			delta = -1
		}
		return Event{Kind: KindWheel, Delta: delta, X: x, Y: y, Mods: mouseMods(code)}, consumed, true
	}
	button := mouseButtonName(code & 3)
	final := buffer[end]
	switch {
	case final == 'm':
		return Event{Kind: KindMouse, Action: ActionRelease, Button: button, X: x, Y: y, Mods: mouseMods(code)}, consumed, true
	case code&32 != 0:
		if button == "" {
			return Event{}, consumed, true
		}
		return Event{Kind: KindMouse, Action: ActionDrag, Button: button, X: x, Y: y, Mods: mouseMods(code)}, consumed, true
	default:
		if button == "" {
			return Event{}, consumed, true
		}
		return Event{Kind: KindMouse, Action: ActionPress, Button: button, X: x, Y: y, Mods: mouseMods(code)}, consumed, true
	}
}

func mouseButtonName(code int) string {
	switch code {
	case 0:
		return ButtonLeft
	case 1:
		return ButtonMiddle
	case 2:
		return ButtonRight
	}
	return ""
}

func mouseMods(code int) Mods {
	return Mods{Shift: code&4 != 0, Alt: code&8 != 0, Ctrl: code&16 != 0}
}

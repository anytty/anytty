package keys

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// Kind is the normalized input class of one host input event.
type Kind uint8

const (
	// KindKey is a key press.
	KindKey Kind = iota
	// KindPaste is one complete paste text (the host chunks it on delivery).
	KindPaste
	// KindMouse is a mouse action (press/drag/release).
	KindMouse
	// KindWheel is a wheel notch.
	KindWheel
)

func (k Kind) String() string {
	switch k {
	case KindKey:
		return "key"
	case KindPaste:
		return "paste"
	case KindMouse:
		return "mouse"
	case KindWheel:
		return "wheel"
	default:
		return "unknown"
	}
}

// Mods are the keyboard modifiers of one event.
type Mods struct {
	Shift bool
	Alt   bool
	Ctrl  bool
}

// Event is one normalized input event. Key events carry the normalized key
// name ("ctrl-p", "page-up", "enter") and, when the key produces text, the
// printable Char (possibly multi-byte UTF-8). Paste events carry the original
// text; mouse and wheel events carry 1-based terminal coordinates. Node and
// HitFocused are routing inputs filled by the hit test.
type Event struct {
	Kind Kind

	// Key is the normalized key name; alias spellings ("pgup", "escape") are
	// accepted by Name and EncodeKey.
	Key  string
	Char string
	Mods Mods

	// Text is the original paste text; CR/LF normalization happens on
	// delivery, not here.
	Text string

	// Mouse/wheel fields.
	Action string
	Button string
	X, Y   int
	Delta  int

	// Routing hints.
	Node       string
	HitFocused bool
}

// Mouse actions and buttons used by EncodeMouse (PROTOCOL §6.7).
const (
	ActionPress   = "press"
	ActionDrag    = "drag"
	ActionRelease = "release"

	ButtonLeft   = "left"
	ButtonMiddle = "middle"
	ButtonRight  = "right"
	ButtonNone   = "none"
)

// namedKeys is the set of key names the encoder understands beyond a single
// printable character.
var namedKeys = map[string]bool{
	"ctrl-a": true, "ctrl-b": true, "ctrl-c": true, "ctrl-d": true,
	"ctrl-e": true, "ctrl-f": true, "ctrl-g": true, "ctrl-h": true,
	"ctrl-i": true, "ctrl-j": true, "ctrl-k": true, "ctrl-l": true,
	"ctrl-m": true, "ctrl-n": true, "ctrl-o": true, "ctrl-p": true,
	"ctrl-q": true, "ctrl-r": true, "ctrl-s": true, "ctrl-t": true,
	"ctrl-u": true, "ctrl-v": true, "ctrl-w": true, "ctrl-x": true,
	"ctrl-y": true, "ctrl-z": true,
	"ctrl-space": true, "ctrl-@": true, "ctrl-[": true, "ctrl-\\": true,
	"ctrl-]": true, "ctrl-^": true, "ctrl-_": true, "ctrl-/": true,
	"ctrl-?": true,
	"enter":  true, "tab": true, "shift-tab": true, "backspace": true,
	"esc": true, "up": true, "down": true, "left": true, "right": true,
	"home": true, "end": true, "insert": true, "delete": true,
	"page-up": true, "page-down": true,
	"f1": true, "f2": true, "f3": true, "f4": true, "f5": true, "f6": true,
	"f7": true, "f8": true, "f9": true, "f10": true, "f11": true, "f12": true,
}

// keyAliases folds vendor spellings into the protocol names.
var keyAliases = map[string]string{
	"return":    "enter",
	"escape":    "esc",
	"pgup":      "page-up",
	"pageup":    "page-up",
	"page_up":   "page-up",
	"pgdn":      "page-down",
	"pagedown":  "page-down",
	"page_down": "page-down",
	"del":       "delete",
	"ins":       "insert",
	"bs":        "backspace",
}

// Name returns the normalized protocol key name for routing and key events:
// "ctrl-p", "page-up", "enter", a printable character like "?" or "中", or ""
// when the event is not a key.
func Name(ev Event) string {
	if ev.Kind != KindKey {
		return ""
	}
	key := strings.TrimSpace(ev.Key)
	if key != "" {
		if alias, ok := keyAliases[strings.ToLower(key)]; ok {
			key = alias
		} else if namedKeys[strings.ToLower(key)] {
			key = strings.ToLower(key)
		}
	}
	if key == "" {
		key = ev.Char
	}
	if key == "" {
		return ""
	}
	if _, ok := ctrlByte(key); ok {
		return key
	}
	if strings.HasPrefix(key, "ctrl-") {
		return key
	}
	// Enhanced keyboards (CSI-u) can deliver shifted/alted letters; the v3
	// recommended profile binds ctrl-shift-c/v/h (copy/paste/clipboard) and
	// ctrl-alt-1..5 (tab jump), so keep those modifiers in the name instead
	// of folding them into the base ctrl key.
	if ev.Mods.Ctrl && ev.Mods.Shift && !ev.Mods.Alt {
		if r, size := utf8.DecodeRuneInString(key); size > 0 && size == len(key) && r < utf8.RuneSelf {
			if letter, ok := ctrlLetter(r); ok {
				return "ctrl-shift-" + string(letter)
			}
		}
	}
	if ev.Mods.Ctrl && ev.Mods.Alt && !ev.Mods.Shift {
		if r, size := utf8.DecodeRuneInString(key); size > 0 && size == len(key) && r >= '0' && r <= '9' {
			return "ctrl-alt-" + string(r)
		}
	}
	if ev.Mods.Ctrl {
		if r, size := utf8.DecodeRuneInString(key); size > 0 && size == len(key) && r < utf8.RuneSelf {
			if letter, ok := ctrlLetter(r); ok {
				return "ctrl-" + string(letter)
			}
		}
	}
	return key
}

// Encode encodes one key or pointer event as PTY bytes. Paste is not handled
// here: one paste event expands to several independently encoded chunks, see
// ChunkPaste.
func Encode(ev Event) ([]byte, bool) {
	switch ev.Kind {
	case KindKey:
		return EncodeKey(ev)
	case KindMouse:
		return EncodeMouse(ev)
	case KindWheel:
		return EncodeWheel(ev)
	default:
		return nil, false
	}
}

// EncodeKey encodes one key press as xterm-compatible bytes. The result is
// ok=false when the event names no encodable key.
func EncodeKey(ev Event) ([]byte, bool) {
	if ev.Kind != KindKey {
		return nil, false
	}
	name := Name(ev)
	if name == "" {
		return nil, false
	}
	if b, ok := ctrlByte(name); ok {
		return altPrefix([]byte{b}, ev.Mods), true
	}
	switch name {
	case "enter":
		return altSimple([]byte{'\r'}, ev.Mods), true
	case "tab":
		return altSimple([]byte{'\t'}, ev.Mods), true
	case "backspace":
		return altSimple([]byte{0x7f}, ev.Mods), true
	case "esc":
		return altSimple([]byte{0x1b}, ev.Mods), true
	case "shift-tab":
		return []byte(csi("Z", modifier(ev.Mods))), true
	case "up":
		return []byte(csi("A", modifier(ev.Mods))), true
	case "down":
		return []byte(csi("B", modifier(ev.Mods))), true
	case "right":
		return []byte(csi("C", modifier(ev.Mods))), true
	case "left":
		return []byte(csi("D", modifier(ev.Mods))), true
	case "home":
		return []byte(csi("H", modifier(ev.Mods))), true
	case "end":
		return []byte(csi("F", modifier(ev.Mods))), true
	case "insert":
		return []byte(csiTilde(2, modifier(ev.Mods))), true
	case "delete":
		return []byte(csiTilde(3, modifier(ev.Mods))), true
	case "page-up":
		return []byte(csiTilde(5, modifier(ev.Mods))), true
	case "page-down":
		return []byte(csiTilde(6, modifier(ev.Mods))), true
	case "f1":
		return []byte(functionKey(1, modifier(ev.Mods))), true
	case "f2":
		return []byte(functionKey(2, modifier(ev.Mods))), true
	case "f3":
		return []byte(functionKey(3, modifier(ev.Mods))), true
	case "f4":
		return []byte(functionKey(4, modifier(ev.Mods))), true
	case "f5":
		return []byte(csiTilde(15, modifier(ev.Mods))), true
	case "f6":
		return []byte(csiTilde(17, modifier(ev.Mods))), true
	case "f7":
		return []byte(csiTilde(18, modifier(ev.Mods))), true
	case "f8":
		return []byte(csiTilde(19, modifier(ev.Mods))), true
	case "f9":
		return []byte(csiTilde(20, modifier(ev.Mods))), true
	case "f10":
		return []byte(csiTilde(21, modifier(ev.Mods))), true
	case "f11":
		return []byte(csiTilde(23, modifier(ev.Mods))), true
	case "f12":
		return []byte(csiTilde(24, modifier(ev.Mods))), true
	}

	char := ev.Char
	if char == "" {
		if utf8.RuneCountInString(name) != 1 {
			return nil, false
		}
		char = name
	}
	return altPrefix([]byte(char), ev.Mods), true
}

// ctrlByte maps a "ctrl-X" name to its control byte. ASCII case is folded;
// non-ASCII and unknown suffixes report false so the caller can fall back to
// the printable form.
func ctrlByte(name string) (byte, bool) {
	suffix, ok := strings.CutPrefix(name, "ctrl-")
	if !ok {
		return 0, false
	}
	switch suffix {
	case "space", "@":
		return 0x00, true
	case "[":
		return 0x1b, true
	case "\\":
		return 0x1c, true
	case "]":
		return 0x1d, true
	case "^":
		return 0x1e, true
	case "_", "/":
		return 0x1f, true
	case "?":
		return 0x7f, true
	}
	if len(suffix) == 1 {
		if letter, ok := ctrlLetter(rune(suffix[0])); ok {
			return byte(letter-'a') + 1, true
		}
	}
	return 0, false
}

func ctrlLetter(r rune) (rune, bool) {
	if r >= 'A' && r <= 'Z' {
		r += 'a' - 'A'
	}
	if r < 'a' || r > 'z' {
		return 0, false
	}
	return r, true
}

// modifier folds Shift/Alt/Ctrl into the xterm CSI modifier (1 = none).
func modifier(mods Mods) int {
	mod := 1
	if mods.Shift {
		mod += 1
	}
	if mods.Alt {
		mod += 2
	}
	if mods.Ctrl {
		mod += 4
	}
	return mod
}

// altSimple prepends ESC for Alt+key forms that xterm encodes as a meta
// prefix (printable characters, Enter, Tab, Backspace, Esc). Other named
// keys carry Alt in their CSI modifier instead.
func altSimple(base []byte, mods Mods) []byte {
	if mods.Alt && !mods.Shift && !mods.Ctrl {
		return append([]byte{0x1b}, base...)
	}
	return base
}

// altPrefix prepends ESC to a self-contained payload (control byte or
// printable character) when Alt is held.
func altPrefix(base []byte, mods Mods) []byte {
	if mods.Alt {
		return append([]byte{0x1b}, base...)
	}
	return base
}

func csi(final string, mod int) string {
	if mod <= 1 {
		return "\x1b[" + final
	}
	return "\x1b[1;" + strconv.Itoa(mod) + final
}

func csiTilde(code, mod int) string {
	if mod <= 1 {
		return "\x1b[" + strconv.Itoa(code) + "~"
	}
	return "\x1b[" + strconv.Itoa(code) + ";" + strconv.Itoa(mod) + "~"
}

// functionKey encodes F1-F4: SS3 without modifiers, CSI 1;mod with them.
func functionKey(n, mod int) string {
	final := string(rune('P' + n - 1))
	if mod <= 1 {
		return "\x1bO" + final
	}
	return "\x1b[1;" + strconv.Itoa(mod) + final
}

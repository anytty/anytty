package terminalhost

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/anytty/anytty/tui/input"
)

// InputParser 是 raw TTY 输入分帧与规范化的 owner。
// 它跨 read chunk 保存未完成 UTF-8/CSI/OSC，并只产出完整 InputEvent；业务 scene、快捷键和 PTY 目标不属于本类型。
type InputParser struct {
	pending []byte
}

var bracketedPasteEnd = []byte("\x1b[201~")

// NewInputParser 建立一个会话级 parser；同一个 TerminalHost 生命周期必须复用实例，不能逐 read 重建并丢失 pending bytes。
func NewInputParser() *InputParser {
	return &InputParser{}
}

// Feed 接收任意 raw chunk 并返回当前已完整分帧的事件。
// 尾部不完整序列留在 parser 内；单独 Esc 也先等待 Host 的歧义窗口，再由 FlushEscape 提交。
func (parser *InputParser) Feed(chunk []byte) []input.InputEvent {
	if len(chunk) == 0 {
		return nil
	}
	parser.pending = append(parser.pending, chunk...)
	var events []input.InputEvent
	for len(parser.pending) > 0 {
		event, consumed, complete := parseOneInput(parser.pending)
		if !complete {
			break
		}
		events = append(events, event)
		parser.pending = parser.pending[consumed:]
	}
	return events
}

// FlushEscape 在宿主输入等待窗口到期后提交一个独立 Esc。
// parser 只提交单字节 Esc，或恰好两个字节且同时可解释为传统 Alt+char 的歧义前缀；
// 已包含参数/正文的不完整 CSI/OSC 仍等待后续 raw bytes，避免把宿主控制序列拆成普通字符。
func (parser *InputParser) FlushEscape() []input.InputEvent {
	if len(parser.pending) == 0 || parser.pending[0] != '\x1b' {
		return nil
	}
	if len(parser.pending) == 1 {
		parser.pending = parser.pending[:0]
		return []input.InputEvent{{Kind: input.EventKindKey, Key: input.KeyEsc, RawSeq: "\x1b"}}
	}
	if len(parser.pending) == 2 {
		if event, consumed, ok := parseAltChar(parser.pending); ok && consumed == 2 {
			parser.pending = parser.pending[:0]
			return []input.InputEvent{event}
		}
	}
	return nil
}

func (parser *InputParser) hasPendingEscape() bool {
	return len(parser.pending) >= 1 && len(parser.pending) <= 2 && parser.pending[0] == '\x1b'
}

func parseOneInput(buffer []byte) (input.InputEvent, int, bool) {
	if len(buffer) == 0 {
		return input.InputEvent{}, 0, false
	}
	first := buffer[0]
	switch first {
	case '\x1b':
		return parseEscape(buffer)
	case '\r', '\n':
		return input.InputEvent{
			Kind:   input.EventKindKey,
			Key:    input.KeyEnter,
			RawSeq: string(buffer[:1]),
		}, 1, true
	case '\t':
		return input.InputEvent{
			Kind:   input.EventKindKey,
			Key:    input.KeyTab,
			RawSeq: string(buffer[:1]),
		}, 1, true
	case 0x7f:
		return input.InputEvent{
			Kind:   input.EventKindKey,
			Key:    input.KeyBackspace,
			RawSeq: string(buffer[:1]),
		}, 1, true
	}
	if first < 0x20 {
		return input.InputEvent{
			Kind:   input.EventKindKey,
			Key:    input.KeyChar,
			Char:   string([]byte{first}),
			Ctrl:   first < 0x20,
			RawSeq: string(buffer[:1]),
		}, 1, true
	}
	r, size := utf8.DecodeRune(buffer)
	if r == utf8.RuneError && size == 1 && !utf8.FullRune(buffer) {
		return input.InputEvent{}, 0, false
	}
	if r == utf8.RuneError && size == 1 {
		return unknownKey(buffer[:1]), 1, true
	}
	return input.InputEvent{
		Kind:   input.EventKindKey,
		Key:    input.KeyChar,
		Char:   string(r),
		RawSeq: string(buffer[:size]),
	}, size, true
}

func parseEscape(buffer []byte) (input.InputEvent, int, bool) {
	if len(buffer) == 1 {
		return input.InputEvent{}, 0, false
	}
	if buffer[1] == ']' {
		return parseOSC(buffer)
	}
	if buffer[1] == 'O' {
		return parseSS3(buffer)
	}
	if buffer[1] != '[' {
		event, consumed, ok := parseAltChar(buffer)
		if ok {
			return event, consumed, true
		}
		return input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc, RawSeq: "\x1b"}, 1, true
	}
	if len(buffer) == 2 {
		return input.InputEvent{}, 0, false
	}
	if bytes.HasPrefix(buffer, []byte("\x1b[200~")) {
		return parseBracketedPaste(buffer)
	}
	if buffer[2] == '<' {
		return parseSGRMouse(buffer)
	}
	for i := 2; i < len(buffer); i++ {
		if buffer[i] >= 0x40 && buffer[i] <= 0x7e {
			if event, ok := csiKeyEvent(buffer[:i+1]); ok {
				return event, i + 1, true
			}
			event := hostControl(buffer[:i+1])
			if buffer[i] == 'u' {
				event.KeyboardProtocol = input.KeyboardProtocolKittyCSIU
			}
			return event, i + 1, true
		}
	}
	return input.InputEvent{}, 0, false
}

func parseBracketedPaste(buffer []byte) (input.InputEvent, int, bool) {
	const startLength = len("\x1b[200~")
	endOffset := bytes.Index(buffer[startLength:], bracketedPasteEnd)
	if endOffset < 0 {
		return input.InputEvent{}, 0, false
	}
	contentEnd := startLength + endOffset
	consumed := contentEnd + len(bracketedPasteEnd)
	return input.InputEvent{Kind: input.EventKindPaste, Paste: string(buffer[startLength:contentEnd])}, consumed, true
}

func parseOSC(buffer []byte) (input.InputEvent, int, bool) {
	if len(buffer) < 3 {
		return input.InputEvent{}, 0, false
	}
	end := -1
	st := false
	for i := 2; i < len(buffer); i++ {
		if buffer[i] == '\a' {
			end = i
			break
		}
		if buffer[i] == '\\' && i > 2 && buffer[i-1] == '\x1b' {
			end = i
			st = true
			break
		}
	}
	if end < 0 {
		return input.InputEvent{}, 0, false
	}
	consumed := end + 1
	bodyEnd := end
	if st {
		bodyEnd = end - 1
	}
	seq := buffer[:consumed]
	body := string(buffer[2:bodyEnd])
	if event, ok := oscThemeEvent(body, string(seq)); ok {
		return event, consumed, true
	}
	// OSC 是 host terminal 的控制响应，不是用户键盘输入；未知响应也必须结构化吞掉，不能进入 PTY。
	return input.InputEvent{Kind: input.EventKindHostControl, RawSeq: string(seq)}, consumed, true
}

func oscThemeEvent(body string, raw string) (input.InputEvent, bool) {
	parts := strings.Split(body, ";")
	if len(parts) < 2 {
		return input.InputEvent{}, false
	}
	switch parts[0] {
	case "10":
		colorValue, ok := parseOSCColor(parts[1])
		if !ok {
			return input.InputEvent{}, false
		}
		return input.InputEvent{Kind: input.EventKindHostTheme, Theme: input.HostThemeEvent{DefaultFG: colorValue}, RawSeq: raw}, true
	case "11":
		colorValue, ok := parseOSCColor(parts[1])
		if !ok {
			return input.InputEvent{}, false
		}
		return input.InputEvent{Kind: input.EventKindHostTheme, Theme: input.HostThemeEvent{DefaultBG: colorValue}, RawSeq: raw}, true
	case "4":
		if len(parts) < 3 {
			return input.InputEvent{}, false
		}
		index, err := strconv.Atoi(parts[1])
		if err != nil || index < 0 || index > 255 {
			return input.InputEvent{}, false
		}
		colorValue, ok := parseOSCColor(parts[2])
		if !ok {
			return input.InputEvent{}, false
		}
		return input.InputEvent{Kind: input.EventKindHostTheme, Theme: input.HostThemeEvent{PaletteIndex: index, PaletteColor: colorValue}, RawSeq: raw}, true
	default:
		return input.InputEvent{}, false
	}
}

func parseOSCColor(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "?" {
		return "", false
	}
	if strings.HasPrefix(value, "#") && len(value) == 7 {
		return strings.ToLower(value), true
	}
	if strings.HasPrefix(value, "rgb:") {
		parts := strings.Split(strings.TrimPrefix(value, "rgb:"), "/")
		if len(parts) != 3 {
			return "", false
		}
		r, okR := parseOSCColorComponent(parts[0])
		g, okG := parseOSCColorComponent(parts[1])
		b, okB := parseOSCColorComponent(parts[2])
		if !okR || !okG || !okB {
			return "", false
		}
		return fmt.Sprintf("#%02x%02x%02x", r, g, b), true
	}
	return "", false
}

func parseOSCColorComponent(value string) (int, bool) {
	if len(value) == 0 {
		return 0, false
	}
	parsed, err := strconv.ParseUint(value, 16, 16)
	if err != nil {
		return 0, false
	}
	max := uint64(1)<<(len(value)*4) - 1
	if max == 0 {
		return 0, false
	}
	return int((parsed*255 + max/2) / max), true
}

func parseSS3(buffer []byte) (input.InputEvent, int, bool) {
	if len(buffer) == 2 {
		return input.InputEvent{}, 0, false
	}
	seq := buffer[:3]
	event := input.InputEvent{Kind: input.EventKindKey, RawSeq: string(seq)}
	switch buffer[2] {
	case 'A':
		event.Key = input.KeyUp
	case 'B':
		event.Key = input.KeyDown
	case 'C':
		event.Key = input.KeyRight
	case 'D':
		event.Key = input.KeyLeft
	case 'F':
		event.Key = input.KeyEnd
	case 'H':
		event.Key = input.KeyHome
	case 'P':
		event.Key = input.KeyF1
	case 'Q':
		event.Key = input.KeyF2
	case 'R':
		event.Key = input.KeyF3
	case 'S':
		event.Key = input.KeyF4
	default:
		return hostControl(seq), 3, true
	}
	return event, 3, true
}

func csiKeyEvent(seq []byte) (input.InputEvent, bool) {
	if len(seq) < 3 || seq[0] != '\x1b' || seq[1] != '[' {
		return input.InputEvent{}, false
	}
	body := string(seq[2 : len(seq)-1])
	final := seq[len(seq)-1]
	event := input.InputEvent{Kind: input.EventKindKey, RawSeq: string(seq)}
	switch final {
	case 'u':
		if strings.HasPrefix(body, "?") {
			return csiKeyboardCapabilityEvent(strings.TrimPrefix(body, "?"), string(seq))
		}
		return csiUnicodeKeyEvent(body, string(seq))
	case 'A', 'B', 'C', 'D', 'F', 'H', 'Z':
		parts := splitCSIParams(body)
		switch final {
		case 'A':
			event.Key = input.KeyUp
		case 'B':
			event.Key = input.KeyDown
		case 'C':
			event.Key = input.KeyRight
		case 'D':
			event.Key = input.KeyLeft
		case 'F':
			event.Key = input.KeyEnd
		case 'H':
			event.Key = input.KeyHome
		case 'Z':
			event.Key = input.KeyShiftTab
			event.Shift = true
		}
		modifier, valid := keyModifierParam(parts, 1)
		if !valid {
			return input.InputEvent{}, false
		}
		if !applyKeyModifier(&event, modifier) {
			return hostControl(seq), true
		}
		return event, true
	case '~':
		parts := splitCSIParams(body)
		code, err := strconv.Atoi(parts[0])
		if err != nil {
			return input.InputEvent{}, false
		}
		key, ok := tildeKey(code)
		if !ok {
			return input.InputEvent{}, false
		}
		event.Key = key
		modifier, valid := keyModifierParam(parts, 1)
		if !valid {
			return input.InputEvent{}, false
		}
		if !applyKeyModifier(&event, modifier) {
			return hostControl(seq), true
		}
		return event, true
	default:
		return input.InputEvent{}, false
	}
}

func csiKeyboardCapabilityEvent(body string, raw string) (input.InputEvent, bool) {
	flags, err := strconv.Atoi(body)
	if err != nil || flags < 0 {
		return input.InputEvent{}, false
	}
	return input.InputEvent{
		Kind: input.EventKindHostCapability,
		Capability: input.HostCapabilityEvent{
			KeyboardDisambiguation: flags&1 != 0,
		},
		RawSeq: raw,
	}, true
}

// csiUnicodeKeyEvent 解码 Kitty keyboard protocol 的 CSI-u 键事件。
// TerminalHost 只负责把宿主协议还原为 InputEvent；快捷键命中仍由 input catalog 决定。
func csiUnicodeKeyEvent(body string, raw string) (input.InputEvent, bool) {
	parts := splitCSIParams(body)
	if len(parts) == 0 || len(parts) > 2 {
		return input.InputEvent{}, false
	}
	codeText := parts[0]
	if strings.Contains(codeText, ":") {
		return input.InputEvent{}, false
	}
	codepoint, err := strconv.Atoi(codeText)
	if err != nil || codepoint < 0 || codepoint > utf8.MaxRune || codepoint >= 0xd800 && codepoint <= 0xdfff {
		return input.InputEvent{}, false
	}
	modifier := 1
	eventType := 1
	if len(parts) == 2 {
		modifierParts := strings.Split(parts[1], ":")
		if len(modifierParts) > 2 {
			return input.InputEvent{}, false
		}
		modifier, err = strconv.Atoi(modifierParts[0])
		if err != nil || modifier < 1 || modifier > 256 {
			return input.InputEvent{}, false
		}
		if len(modifierParts) == 2 {
			eventType, err = strconv.Atoi(modifierParts[1])
			if err != nil || eventType < 1 || eventType > 3 {
				return input.InputEvent{}, false
			}
		}
	}
	event := input.InputEvent{
		Kind:             input.EventKindKey,
		Key:              input.KeyChar,
		Char:             string(rune(codepoint)),
		RawSeq:           raw,
		KeyboardProtocol: input.KeyboardProtocolKittyCSIU,
	}
	// 当前 input domain 不建模 PUA functional key；不得把这些宿主协议 codepoint 当作文本。
	if codepoint >= 57344 && codepoint <= 63743 {
		event.Key = input.KeyUnknown
		event.Char = ""
	} else {
		applyCSIUControlKey(&event, codepoint)
	}
	if !applyKeyModifier(&event, modifier) {
		event.Key = input.KeyUnknown
		event.Char = ""
	}
	if event.Key == input.KeyTab && event.Shift {
		event.Key = input.KeyShiftTab
	}
	if eventType == 3 {
		event.Key = input.KeyUnknown
		event.Char = ""
	}
	return event, true
}

// applyCSIUControlKey 把增强协议中的控制 codepoint 还原为 TUI 通用命名键。
// overlay、prompt 和普通 route 只消费标准 Key，不应感知 Kitty CSI-u 的编码差异。
func applyCSIUControlKey(event *input.InputEvent, codepoint int) {
	switch codepoint {
	case 9:
		event.Key = input.KeyTab
		event.Char = ""
	case 13:
		event.Key = input.KeyEnter
		event.Char = ""
	case 27:
		event.Key = input.KeyEsc
		event.Char = ""
	case 127:
		event.Key = input.KeyBackspace
		event.Char = ""
	}
}

func splitCSIParams(body string) []string {
	if body == "" {
		return []string{""}
	}
	return strings.Split(body, ";")
}

func keyModifierParam(parts []string, index int) (int, bool) {
	if index >= len(parts) {
		return 1, true
	}
	value, err := strconv.Atoi(parts[index])
	if err != nil || value < 1 || value > 256 {
		return 0, false
	}
	return value, true
}

func applyKeyModifier(event *input.InputEvent, encoded int) bool {
	if encoded <= 1 {
		return true
	}
	mask := encoded - 1
	event.Shift = mask&1 != 0
	event.Alt = mask&2 != 0
	event.Ctrl = mask&4 != 0
	const superHyperMeta = 8 | 16 | 32
	return mask&superHyperMeta == 0
}

func tildeKey(code int) (input.Key, bool) {
	switch code {
	case 1, 7:
		return input.KeyHome, true
	case 2:
		return input.KeyInsert, true
	case 3:
		return input.KeyDelete, true
	case 4, 8:
		return input.KeyEnd, true
	case 5:
		return input.KeyPageUp, true
	case 6:
		return input.KeyPageDn, true
	case 11:
		return input.KeyF1, true
	case 12:
		return input.KeyF2, true
	case 13:
		return input.KeyF3, true
	case 14:
		return input.KeyF4, true
	case 15:
		return input.KeyF5, true
	case 17:
		return input.KeyF6, true
	case 18:
		return input.KeyF7, true
	case 19:
		return input.KeyF8, true
	case 20:
		return input.KeyF9, true
	case 21:
		return input.KeyF10, true
	case 23:
		return input.KeyF11, true
	case 24:
		return input.KeyF12, true
	default:
		return "", false
	}
}

func parseAltChar(buffer []byte) (input.InputEvent, int, bool) {
	r, size := utf8.DecodeRune(buffer[1:])
	if r == utf8.RuneError && size == 1 && !utf8.FullRune(buffer[1:]) {
		return input.InputEvent{}, 0, false
	}
	if r == utf8.RuneError && size == 1 {
		return input.InputEvent{}, 0, false
	}
	event := input.InputEvent{
		Kind:   input.EventKindKey,
		Key:    input.KeyChar,
		Char:   string(r),
		Alt:    true,
		RawSeq: string(buffer[:1+size]),
	}
	applyCSIUControlKey(&event, int(r))
	if event.Key == input.KeyChar && r < 0x20 {
		event.Ctrl = true
	}
	return event, 1 + size, true
}

func parseSGRMouse(buffer []byte) (input.InputEvent, int, bool) {
	end := -1
	for i := 3; i < len(buffer); i++ {
		if buffer[i] == 'M' || buffer[i] == 'm' {
			end = i
			break
		}
	}
	if end < 0 {
		return input.InputEvent{}, 0, false
	}
	seq := buffer[:end+1]
	parts := strings.Split(string(buffer[3:end]), ";")
	if len(parts) != 3 {
		return hostControl(seq), len(seq), true
	}
	code, errCode := strconv.Atoi(parts[0])
	col, errCol := strconv.Atoi(parts[1])
	row, errRow := strconv.Atoi(parts[2])
	if errCode != nil || errCol != nil || errRow != nil || code < 0 || col <= 0 || row <= 0 {
		return hostControl(seq), len(seq), true
	}
	button, ok := mouseButton(code, buffer[end])
	if !ok {
		return hostControl(seq), len(seq), true
	}
	return input.InputEvent{
		Kind:   input.EventKindMouse,
		Mouse:  button,
		Row:    row,
		Col:    col,
		Shift:  code&4 != 0,
		Alt:    code&8 != 0,
		Ctrl:   code&16 != 0,
		RawSeq: string(seq),
	}, len(seq), true
}

func mouseButton(code int, final byte) (input.MouseButton, bool) {
	const supportedBits = 3 | 4 | 8 | 16 | 32 | 64
	if code & ^supportedBits != 0 {
		return "", false
	}
	if code&64 != 0 {
		if final != 'M' || code&32 != 0 || code&3 > 1 {
			return "", false
		}
		if code&1 != 0 {
			return input.MouseWheelDown, true
		}
		return input.MouseWheelUp, true
	}
	button := code & 3
	if final == 'm' {
		switch button {
		case 0:
			return input.MouseLeftUp, true
		case 1:
			return input.MouseMiddleUp, true
		case 2:
			return input.MouseRightUp, true
		default:
			return input.MouseRelease, true
		}
	}
	if final != 'M' {
		return "", false
	}
	if code&32 != 0 {
		switch button {
		case 0:
			return input.MouseLeftDrag, true
		case 1:
			return input.MouseMiddleDrag, true
		case 2:
			return input.MouseRightDrag, true
		default:
			return input.MouseMove, true
		}
	}
	if button == 3 {
		return "", false
	}
	switch button {
	case 0:
		return input.MouseLeft, true
	case 1:
		return input.MouseMiddle, true
	case 2:
		return input.MouseRight, true
	default:
		return input.MouseMove, true
	}
}

func unknownKey(seq []byte) input.InputEvent {
	return input.InputEvent{
		Kind:   input.EventKindKey,
		Key:    input.KeyUnknown,
		RawSeq: string(seq),
	}
}

func hostControl(seq []byte) input.InputEvent {
	return input.InputEvent{Kind: input.EventKindHostControl, RawSeq: string(seq)}
}

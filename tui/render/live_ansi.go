package render

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// terminalLiveLineFromANSI 把 live surface 的基础 SGR 行投影为 styled cells。
// 它只服务实时内容展示，不参与 copy/history truth。
func terminalLiveLineFromANSI(value string) Line {
	if value == "" {
		return NewLine("")
	}
	parser := liveANSIParser{}
	parser.parse(value)
	return Line{Cells: parser.cells}
}

type liveANSIParser struct {
	style StyleToken
	buf   strings.Builder
	cells []Cell
}

func (parser *liveANSIParser) parse(value string) {
	for len(value) > 0 {
		if strings.HasPrefix(value, "\x1b[") {
			if consumed, ok := parser.consumeCSI(value); ok {
				value = value[consumed:]
				continue
			}
		}
		if value[0] == '\x1b' {
			value = value[1:]
			continue
		}
		r, size := utf8.DecodeRuneInString(value)
		if r == utf8.RuneError && size == 1 {
			value = value[size:]
			continue
		}
		if r < 0x20 && r != '\t' {
			value = value[size:]
			continue
		}
		parser.buf.WriteString(value[:size])
		value = value[size:]
	}
	parser.flush()
}

func (parser *liveANSIParser) consumeCSI(value string) (int, bool) {
	final := -1
	for i := 2; i < len(value); i++ {
		ch := value[i]
		if ch >= 0x40 && ch <= 0x7e {
			final = i
			break
		}
	}
	if final == -1 {
		return 0, false
	}
	parser.flush()
	if value[final] == 'm' {
		parser.applySGR(value[2:final])
	}
	return final + 1, true
}

func (parser *liveANSIParser) applySGR(params string) {
	if params == "" {
		parser.style = ""
		return
	}
	for _, raw := range strings.Split(params, ";") {
		if raw == "" {
			raw = "0"
		}
		code, err := strconv.Atoi(raw)
		if err != nil {
			continue
		}
		switch code {
		case 0, 22, 23, 24, 27, 39:
			parser.style = ""
		case 2, 90:
			parser.style = StyleMuted
		case 31, 91:
			parser.style = StyleDanger
		case 32, 92:
			parser.style = StyleSuccess
		case 33, 93:
			parser.style = StyleWarning
		case 34, 36, 94, 96:
			parser.style = StyleInfo
		case 35, 95:
			parser.style = StyleAccent
		}
	}
}

func (parser *liveANSIParser) flush() {
	if parser.buf.Len() == 0 {
		return
	}
	text := SafeLine(parser.buf.String())
	parser.buf.Reset()
	width := DisplayWidth(text)
	if width <= 0 {
		return
	}
	parser.cells = append(parser.cells, Cell{
		Text:            text,
		Width:           width,
		Style:           parser.style,
		TerminalContent: true,
		Safe:            true,
	})
}

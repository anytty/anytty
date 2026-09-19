package harness

import (
	"bytes"
	"encoding/base64"
	"strings"

	"github.com/anytty/anytty/clients/tui/render"
	"github.com/anytty/anytty/clients/tui/render/ansi"
)

// RenderText flattens a screen to plain text rows. Trailing blanks are
// trimmed, matching the default `tmux capture-pane -p` output.
func RenderText(screen ansi.Screen) []string {
	lines := make([]string, 0, len(screen.Lines))
	for _, row := range screen.Lines {
		lines = append(lines, strings.TrimRight(ansi.RowText(row), " "))
	}
	return lines
}

// RenderRaw flattens a screen to rows carrying SGR escape runs. Each
// attribute change is emitted as its own CSI sequence (like tmux -e) so
// callers can grep for a bold-then-color cell run without depending on how
// the program originally combined the parameters.
func RenderRaw(screen ansi.Screen) []string {
	lines := make([]string, 0, len(screen.Lines))
	for _, row := range screen.Lines {
		lines = append(lines, renderRawRow(row))
	}
	return lines
}

func renderRawRow(row []ansi.Cell) string {
	end := len(row)
	for end > 0 {
		cell := row[end-1]
		if cell.Continuation || (cell.Text == "" && cell.Style == "") {
			end--
			continue
		}
		break
	}
	var b strings.Builder
	current := ""
	for x := 0; x < end; x++ {
		cell := row[x]
		if cell.Continuation {
			continue
		}
		params := tokenParams(cell.Style)
		if params != current {
			if current != "" {
				b.WriteString("\x1b[0m")
			}
			b.WriteString(paramsSequences(params))
			current = params
		}
		if cell.Text == "" {
			b.WriteByte(' ')
		} else {
			b.WriteString(cell.Text)
		}
	}
	if current != "" {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

// tokenParams returns the raw SGR parameter run carried by a parsed cell.
// Parsed screens only carry ANSI tokens; anything else has no style.
func tokenParams(token render.Token) string {
	if params, ok := token.RawSGR(); ok {
		return params
	}
	return ""
}

// paramsSequences translates one "a;b;c" SGR parameter run into separate CSI
// sequences: extended colors stay whole, every other parameter gets its own
// sequence. tmux capture-pane -e emits the same shape.
func paramsSequences(params string) string {
	if params == "" {
		return ""
	}
	parts := strings.Split(params, ";")
	var b strings.Builder
	for i := 0; i < len(parts); i++ {
		switch {
		case parts[i] == "38" || parts[i] == "48":
			if i+1 < len(parts) && parts[i+1] == "2" && i+4 < len(parts) {
				b.WriteString("\x1b[" + strings.Join(parts[i:i+5], ";") + "m")
				i += 4
				continue
			}
			if i+1 < len(parts) && parts[i+1] == "5" && i+2 < len(parts) {
				b.WriteString("\x1b[" + strings.Join(parts[i:i+3], ";") + "m")
				i += 2
				continue
			}
		}
		b.WriteString("\x1b[" + parts[i] + "m")
	}
	return b.String()
}

// scanOSC52Locked consumes data for OSC 52 clipboard writes. The payload
// ("52;<selection>;<base64>") is decoded and kept as the current clipboard.
// Callers hold s.mu.
func (s *Session) scanOSC52Locked(data []byte) {
	s.rawTail = append(s.rawTail, data...)
	const prefix = "\x1b]52;"
	for {
		start := bytes.Index(s.rawTail, []byte(prefix))
		if start < 0 {
			s.trimRawTail()
			return
		}
		end, term := oscTerminator(s.rawTail, start+len(prefix))
		if end < 0 {
			s.rawTail = s.rawTail[start:]
			s.trimRawTail()
			return
		}
		payload := string(s.rawTail[start+len(prefix) : end])
		if decoded, ok := decodeOSC52(payload); ok {
			s.clip = decoded
			s.hasClip = true
		}
		s.rawTail = s.rawTail[end+term:]
	}
}

// trimRawTail bounds the OSC scan window so a malformed stream cannot grow it
// without limit. A partial OSC 52 prefix at the tail is preserved.
func (s *Session) trimRawTail() {
	if len(s.rawTail) > 1<<20 {
		s.rawTail = s.rawTail[len(s.rawTail)-4096:]
	}
}

// oscTerminator finds the first BEL (returning 1) or ST (returning 2) at or
// after from. A -1 offset means the sequence is still incomplete.
func oscTerminator(data []byte, from int) (int, int) {
	for i := from; i < len(data); i++ {
		switch data[i] {
		case 0x07:
			return i, 1
		case 0x1b:
			if i+1 >= len(data) {
				return -1, 0
			}
			if data[i+1] == '\\' {
				return i, 2
			}
		}
	}
	return -1, 0
}

// decodeOSC52 extracts the base64 text from an OSC 52 payload. The selection
// field is ignored; the text is the part after the first semicolon.
func decodeOSC52(payload string) (string, bool) {
	sep := strings.IndexByte(payload, ';')
	if sep < 0 {
		return "", false
	}
	encoded := strings.TrimSpace(payload[sep+1:])
	if encoded == "" {
		return "", false
	}
	if decoded, err := base64.StdEncoding.DecodeString(encoded); err == nil {
		return string(decoded), true
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(encoded); err == nil {
		return string(decoded), true
	}
	return "", false
}

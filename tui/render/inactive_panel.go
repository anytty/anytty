package render

import (
	"strconv"
	"strings"
)

// dimRect marks only the painted content; terminal snapshots and copy text keep
// their original values. Apply before higher floating/overlay layers are drawn.
func (c *canvas) dimRect(rect Rect) {
	for y := maxInt(0, rect.Y); y < minInt(c.height, rect.Y+rect.H); y++ {
		for x := maxInt(0, rect.X); x < minInt(c.width, rect.X+rect.W); x++ {
			cell := &c.rows[y][x]
			cell.dimmed = true
			if cell.text == "" && cell.width == 0 && !cell.continuation {
				cell.text = " "
				cell.width = 1
			}
		}
	}
}

// dimANSISequence transforms only SGR produced by our style serializer, never
// arbitrary PTY bytes. Resolve default and indexed colors before blending so
// reverse video, colored backgrounds and host palettes dim consistently.
func dimANSISequence(sequence string, theme Theme) string {
	theme = theme.WithFallback()
	colorParams := func(value string, foreground bool) string {
		var out strings.Builder
		appendANSIColorParams(&out, mixHostColor(value, theme.HostBG, theme.InactivePanelDimAmount), foreground)
		return out.String()
	}
	var out strings.Builder
	out.WriteString("\x1b[" + colorParams(theme.HostFG, true) + ";" + colorParams(theme.HostBG, false) + "m")
	for len(sequence) > 0 {
		start := strings.Index(sequence, "\x1b[")
		if start < 0 {
			out.WriteString(sequence)
			break
		}
		out.WriteString(sequence[:start])
		sequence = sequence[start+2:]
		end := strings.IndexByte(sequence, 'm')
		if end < 0 {
			out.WriteString("\x1b[" + sequence)
			break
		}
		params := strings.Split(sequence[:end], ";")
		converted := make([]string, 0, len(params))
		for i := 0; i < len(params); i++ {
			code, _ := strconv.Atoi(params[i])
			fg := true
			index := -1
			switch {
			case code >= 30 && code <= 37:
				index = code - 30
			case code >= 90 && code <= 97:
				index = code - 90 + 8
			case code >= 40 && code <= 47:
				index = code - 40
				fg = false
			case code >= 100 && code <= 107:
				index = code - 100 + 8
				fg = false
			}
			if index >= 0 {
				converted = append(converted, colorParams(inactivePaletteColor(index, theme), fg))
			} else if (code == 38 || code == 48) && i+2 < len(params) {
				fg = code == 38
				if params[i+1] == "2" && i+4 < len(params) {
					r, _ := strconv.Atoi(params[i+2])
					g, _ := strconv.Atoi(params[i+3])
					b, _ := strconv.Atoi(params[i+4])
					converted = append(converted, colorParams(formatHexColor(r, g, b), fg))
					i += 4
				} else if params[i+1] == "5" {
					index, _ := strconv.Atoi(params[i+2])
					converted = append(converted, colorParams(inactivePaletteColor(index, theme), fg))
					i += 2
				} else {
					converted = append(converted, params[i])
				}
			} else {
				converted = append(converted, params[i])
			}
		}
		out.WriteString("\x1b[" + strings.Join(converted, ";") + "m")
		sequence = sequence[end+1:]
	}
	return out.String()
}

func inactivePaletteColor(index int, theme Theme) string {
	if index < 0 || index > 255 {
		return theme.HostFG
	}
	if index < 16 {
		if color := theme.TerminalPalette[index]; color != "" {
			return color
		}
		// Standard xterm palette when the host has not answered palette queries.
		palette := [...]string{"#000000", "#800000", "#008000", "#808000", "#000080", "#800080", "#008080", "#c0c0c0", "#808080", "#ff0000", "#00ff00", "#ffff00", "#0000ff", "#ff00ff", "#00ffff", "#ffffff"}
		return palette[index]
	}
	if index >= 232 {
		level := 8 + (index-232)*10
		return formatHexColor(level, level, level)
	}
	index -= 16
	levels := [...]int{0, 95, 135, 175, 215, 255}
	return formatHexColor(levels[index/36], levels[(index/6)%6], levels[index%6])
}

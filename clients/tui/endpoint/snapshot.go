package endpoint

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/anytty/anytty/proto/access/apipb"
)

// renderScreenSnapshot turns a daemon native screen snapshot
// (LiveScreenNext with observed_revision=0, i.e. full_replace) into the ANSI
// byte stream the local terminal parser consumes. This is how the daemon
// screen becomes the component picture after attach or reconnect without
// replaying history (ENDPOINTS.zh-CN.md §4).
func renderScreenSnapshot(screen *apipb.NativeScreenResult) []byte {
	if screen == nil {
		return nil
	}
	var b strings.Builder
	b.WriteString("\x1b[0m")
	b.WriteString("\x1b[2J")
	writeSnapshotModes(&b, screen.GetModes())
	rows := computeRows(screen)
	for _, row := range rows {
		if row == nil {
			continue
		}
		b.WriteString(fmt.Sprintf("\x1b[%d;1H\x1b[0m", row.GetRowIndex()+1))
		for _, cell := range row.GetRow().GetCells() {
			writeCell(&b, cell)
		}
	}
	if cursor := screen.GetCursor(); cursor != nil {
		b.WriteString(fmt.Sprintf("\x1b[%d;%dH", cursor.GetRow()+1, cursor.GetCol()+1))
		if cursor.GetVisible() {
			b.WriteString("\x1b[?25h")
		} else {
			b.WriteString("\x1b[?25l")
		}
	}
	return []byte(b.String())
}

// computeRows flattens row replacements into a dense screen: full_replace
// snapshots carry every row, deltas only the changed ones (which are then
// painted over the seeded screen).
func computeRows(screen *apipb.NativeScreenResult) []*apipb.ScreenRowReplace {
	replacements := screen.GetRowReplacements()
	if !screen.GetFullReplace() {
		return replacements
	}
	maxRow := 0
	for _, replacement := range replacements {
		if int(replacement.GetRowIndex()) > maxRow {
			maxRow = int(replacement.GetRowIndex())
		}
	}
	dense := make([]*apipb.ScreenRowReplace, maxRow+1)
	for _, replacement := range replacements {
		dense[int(replacement.GetRowIndex())] = replacement
	}
	return dense
}

func writeSnapshotModes(b *strings.Builder, modes *apipb.TerminalModes) {
	if modes == nil {
		return
	}
	switch {
	case modes.GetAlternateScreen():
		b.WriteString("\x1b[?1049h")
	default:
		b.WriteString("\x1b[?1049l")
	}
	if modes.GetAlternateScroll() {
		b.WriteString("\x1b[?1007h")
	} else {
		b.WriteString("\x1b[?1007l")
	}
	setMode := func(enabled bool, mode int) {
		if enabled {
			b.WriteString(fmt.Sprintf("\x1b[?%dh", mode))
		} else {
			b.WriteString(fmt.Sprintf("\x1b[?%dl", mode))
		}
	}
	setMode(modes.GetMouseX10(), 9)
	setMode(modes.GetMouseNormal(), 1000)
	setMode(modes.GetMouseButtonEvent(), 1002)
	setMode(modes.GetMouseAnyEvent(), 1003)
	setMode(modes.GetMouseSgr(), 1006)
	setMode(modes.GetBracketedPaste(), 2004)
	setMode(modes.GetApplicationCursor(), 1)
	if modes.GetAutoWrap() {
		b.WriteString("\x1b[?7h")
	} else {
		b.WriteString("\x1b[?7l")
	}
}

func writeCell(b *strings.Builder, cell *apipb.ScreenCell) {
	if cell == nil {
		return
	}
	width := int(cell.GetWidth())
	if width <= 0 {
		width = 1
	}
	if cell.GetContent() == "" {
		b.WriteString(strings.Repeat(" ", width))
		return
	}
	b.WriteString(sgrForStyle(cell.GetStyle()))
	b.WriteString(cell.GetContent())
}

// sgrForStyle mirrors the daemon vterm CellStyle encoding ("ansi:N",
// "idx:N" or "#rrggbb").
func sgrForStyle(style *apipb.CellStyle) string {
	if style == nil {
		return "\x1b[0m"
	}
	params := []string{"0"}
	if style.GetBold() {
		params = append(params, "1")
	}
	if style.GetItalic() {
		params = append(params, "3")
	}
	if style.GetUnderline() {
		params = append(params, "4")
	}
	if style.GetBlink() {
		params = append(params, "5")
	}
	if style.GetReverse() {
		params = append(params, "7")
	}
	if style.GetStrikethrough() {
		params = append(params, "9")
	}
	if code := sgrColorParams(style.GetForeground(), true); code != "" {
		params = append(params, code)
	}
	if code := sgrColorParams(style.GetBackground(), false); code != "" {
		params = append(params, code)
	}
	return "\x1b[" + strings.Join(params, ";") + "m"
}

func sgrColorParams(value string, foreground bool) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "ansi:") {
		if index, err := strconv.Atoi(strings.TrimPrefix(value, "ansi:")); err == nil && index >= 0 && index <= 15 {
			if index < 8 {
				if foreground {
					return strconv.Itoa(30 + index)
				}
				return strconv.Itoa(40 + index)
			}
			if foreground {
				return strconv.Itoa(90 + index - 8)
			}
			return strconv.Itoa(100 + index - 8)
		}
	}
	if strings.HasPrefix(value, "idx:") {
		if index, err := strconv.Atoi(strings.TrimPrefix(value, "idx:")); err == nil && index >= 0 && index <= 255 {
			if foreground {
				return fmt.Sprintf("38;5;%d", index)
			}
			return fmt.Sprintf("48;5;%d", index)
		}
	}
	if r, g, bl, ok := parseHexColor(value); ok {
		if foreground {
			return fmt.Sprintf("38;2;%d;%d;%d", r, g, bl)
		}
		return fmt.Sprintf("48;2;%d;%d;%d", r, g, bl)
	}
	return ""
}

func parseHexColor(value string) (uint8, uint8, uint8, bool) {
	if !strings.HasPrefix(value, "#") {
		return 0, 0, 0, false
	}
	hex := strings.TrimPrefix(value, "#")
	switch len(hex) {
	case 3:
		r, err1 := strconv.ParseUint(hex[0:1]+hex[0:1], 16, 8)
		g, err2 := strconv.ParseUint(hex[1:2]+hex[1:2], 16, 8)
		b, err3 := strconv.ParseUint(hex[2:3]+hex[2:3], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return 0, 0, 0, false
		}
		return uint8(r), uint8(g), uint8(b), true
	case 6:
		r, err1 := strconv.ParseUint(hex[0:2], 16, 8)
		g, err2 := strconv.ParseUint(hex[2:4], 16, 8)
		b, err3 := strconv.ParseUint(hex[4:6], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return 0, 0, 0, false
		}
		return uint8(r), uint8(g), uint8(b), true
	default:
		return 0, 0, 0, false
	}
}

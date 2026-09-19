package render

import (
	"strconv"
	"strings"
)

// Token is the opaque style value carried by a rendered line. There are two
// accepted spellings:
//
//   - explicit styles (what v2 layout programs send): the theme-free format
//     "fg:#RRGGBB;bg:#RRGGBB;bold;dim;italic;underline;reverse", translated
//     directly to SGR by the host without consulting any palette;
//   - host-internal tokens ("accent", "muted", "border", ...): resolved
//     against a Theme. Builtin components and the core overlay use these;
//     a program never needs them and should send explicit styles instead.
//
// Unknown spellings safely degrade to no SGR (never to an error or panic).
type Token string

const (
	// TokenDefault is the host foreground on the terminal background.
	TokenDefault Token = "default"
	// TokenBackground is the chrome background color.
	TokenBackground Token = "bg"
	// TokenFG is the chrome foreground color.
	TokenFG Token = "fg"
	// TokenForeground is the v1 alias of the chrome foreground color.
	TokenForeground Token = "foreground"
	// TokenStrongForeground is the chrome foreground, bold.
	TokenStrongForeground Token = "strong-foreground"
	// TokenMuted is the muted foreground, dimmed.
	TokenMuted Token = "muted"
	// TokenAccent is the accent color, bold.
	TokenAccent Token = "accent"
	// TokenAccentDim is the accent color, not bold (secondary emphasis).
	TokenAccentDim Token = "accent_dim"
	// TokenSuccess is the success color (v1 alias of ok).
	TokenSuccess Token = "success"
	// TokenOK is the success color.
	TokenOK Token = "ok"
	// TokenWarning is the warning color.
	TokenWarning Token = "warning"
	// TokenDanger is the danger color.
	TokenDanger Token = "danger"
	// TokenInfo is the info color.
	TokenInfo Token = "info"
	// TokenChrome is the chrome band (foreground on chrome background).
	TokenChrome Token = "chrome"
	// TokenChromeFocus is the focused chrome band.
	TokenChromeFocus Token = "chrome_focus"
	// TokenHeader is the v1 alias of the chrome band.
	TokenHeader Token = "header"
	// TokenTabActive is the active tab block (foreground on tab background).
	TokenTabActive Token = "tab_active"
	// TokenTabInactive is the inactive tab label.
	TokenTabInactive Token = "tab_inactive"
	// TokenFooter is the footer text (v1 alias of muted).
	TokenFooter Token = "footer"
	// TokenFooterAccent is the emphasized footer text.
	TokenFooterAccent Token = "footer-accent"
	// TokenStatus is the status line (foreground on status background).
	TokenStatus Token = "status"
	// TokenOverlay is overlay body text on the overlay background.
	TokenOverlay Token = "overlay"
	// TokenBorder is the default box border.
	TokenBorder Token = "border"
	// TokenBorderFocus is the focused box border.
	TokenBorderFocus Token = "border_focus"
	// TokenBorderDead is the border of an exited/dead box.
	TokenBorderDead Token = "border_dead"
	// TokenActiveBorder is the v1 alias of the focused border.
	TokenActiveBorder Token = "active-border"
	// TokenInactiveBorder is the v1 alias of the muted border.
	TokenInactiveBorder Token = "inactive-border"
	// TokenSelection is the selected list item (foreground on selection
	// background).
	TokenSelection Token = "selection"
)

// Theme holds the colors tokens resolve to. Every field is a "#rrggbb"
// string; the semantic fields drive the M1 token set and the v1 aliases
// resolve through them, so a theme is one coherent palette.
type Theme struct {
	// Name is the palette selector ("dark" or "light"), informational.
	Name string

	Background string // bg
	Foreground string // fg / default
	Muted      string
	Accent     string
	AccentDim  string
	Warning    string
	Danger     string
	OK         string
	Info       string

	ChromeFG      string
	ChromeBG      string
	ChromeFocusFG string
	ChromeFocusBG string

	TabActiveFG   string
	TabActiveBG   string
	TabInactiveFG string

	StatusFG  string
	StatusBG  string
	OverlayBG string

	Border      string
	BorderFocus string
	BorderDead  string

	SelectionFG string
	SelectionBG string

	// v1 compatibility aliases kept in the struct so old call sites keep
	// compiling; WithFallback maps them onto the semantic fields.
	Success        string
	ActiveBorder   string
	InactiveBorder string
}

// ThemeDark returns the default modern dark palette. It avoids pure black
// and pure white so the chrome stays readable on OLED and low-contrast
// displays alike.
func ThemeDark() Theme {
	return Theme{
		Name: "dark",

		Background: "#0f1117",
		Foreground: "#e6e2ec",
		Muted:      "#9a94a8",
		Accent:     "#a78bfa",
		AccentDim:  "#6b5fa8",
		Warning:    "#f0c05a",
		Danger:     "#ef6f6f",
		OK:         "#5fd08a",
		Info:       "#7fb2f0",

		ChromeFG:      "#d9d4e4",
		ChromeBG:      "#161823",
		ChromeFocusFG: "#f2effa",
		ChromeFocusBG: "#23263a",

		TabActiveFG:   "#14121c",
		TabActiveBG:   "#a78bfa",
		TabInactiveFG: "#9a94a8",

		StatusFG:  "#d5d0e0",
		StatusBG:  "#12141c",
		OverlayBG: "#191b26",

		Border:      "#464a5a",
		BorderFocus: "#a78bfa",
		BorderDead:  "#a35d5d",

		SelectionFG: "#f4f1fa",
		SelectionBG: "#3a3357",

		Success:        "#5fd08a",
		ActiveBorder:   "#a78bfa",
		InactiveBorder: "#9a94a8",
	}
}

// ThemeLight returns the light counterpart: dark text on a soft background,
// with the same semantic slots so a layout program never changes tokens.
func ThemeLight() Theme {
	return Theme{
		Name: "light",

		Background: "#f5f3f8",
		Foreground: "#2c2936",
		Muted:      "#6f6a7c",
		Accent:     "#6d3fe0",
		AccentDim:  "#9a7fe0",
		Warning:    "#9a6a00",
		Danger:     "#c0392b",
		OK:         "#2f7d4f",
		Info:       "#2860b8",

		ChromeFG:      "#39354a",
		ChromeBG:      "#e7e3ef",
		ChromeFocusFG: "#241f33",
		ChromeFocusBG: "#d9d3e8",

		TabActiveFG:   "#fbfaff",
		TabActiveBG:   "#6d3fe0",
		TabInactiveFG: "#6f6a7c",

		StatusFG:  "#3a3648",
		StatusBG:  "#e2deeb",
		OverlayBG: "#efeaf7",

		Border:      "#b3adc0",
		BorderFocus: "#6d3fe0",
		BorderDead:  "#a05a5a",

		SelectionFG: "#241f33",
		SelectionBG: "#cfc3f2",

		Success:        "#2f7d4f",
		ActiveBorder:   "#6d3fe0",
		InactiveBorder: "#6f6a7c",
	}
}

// DefaultTheme returns the default palette (dark). It is the host-internal
// palette for builtin chrome; layout programs send explicit styles instead
// and never depend on it.
func DefaultTheme() Theme { return ThemeDark() }

// ThemeByName selects a palette by its config name. An empty name means the
// default; ok is false for an unknown name.
func ThemeByName(name string) (Theme, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "dark":
		return ThemeDark(), true
	case "light":
		return ThemeLight(), true
	default:
		return Theme{}, false
	}
}

// WithFallback fills every zero field with the dark palette, so a theme that
// only overrides one or two colors stays coherent.
func (t Theme) WithFallback() Theme {
	d := ThemeDark()
	fill := func(value *string, fallback string) {
		if *value == "" {
			*value = fallback
		}
	}
	fill(&t.Name, d.Name)
	fill(&t.Background, d.Background)
	fill(&t.Foreground, d.Foreground)
	fill(&t.Muted, d.Muted)
	fill(&t.Accent, d.Accent)
	fill(&t.AccentDim, d.AccentDim)
	fill(&t.Warning, d.Warning)
	fill(&t.Danger, d.Danger)
	fill(&t.OK, d.OK)
	fill(&t.Info, d.Info)
	fill(&t.ChromeFG, d.ChromeFG)
	fill(&t.ChromeBG, d.ChromeBG)
	fill(&t.ChromeFocusFG, d.ChromeFocusFG)
	fill(&t.ChromeFocusBG, d.ChromeFocusBG)
	fill(&t.TabActiveFG, d.TabActiveFG)
	fill(&t.TabActiveBG, d.TabActiveBG)
	fill(&t.TabInactiveFG, d.TabInactiveFG)
	fill(&t.StatusFG, d.StatusFG)
	fill(&t.StatusBG, d.StatusBG)
	fill(&t.OverlayBG, d.OverlayBG)
	fill(&t.Border, d.Border)
	fill(&t.BorderFocus, d.BorderFocus)
	fill(&t.BorderDead, d.BorderDead)
	fill(&t.SelectionFG, d.SelectionFG)
	fill(&t.SelectionBG, d.SelectionBG)
	fill(&t.Success, d.Success)
	fill(&t.ActiveBorder, d.ActiveBorder)
	fill(&t.InactiveBorder, d.InactiveBorder)
	return t
}

// ansiTokenPrefix marks a Token that carries raw SGR parameters produced by
// the PTY parser (e.g. "ansi:38;5;196"). Raw tokens are emitted verbatim
// between a reset and the next style change so program-chosen colors survive
// the framebuffer; builtin chrome keeps using named theme tokens.
const ansiTokenPrefix = "ansi:"

// ANSIToken builds a Token from raw SGR parameters (without the CSI wrapper
// and final "m"). Empty parameters yield the default token.
func ANSIToken(params string) Token {
	if params == "" {
		return TokenDefault
	}
	return Token(ansiTokenPrefix + params)
}

// RawSGR returns the raw SGR parameters when t is an ANSI token.
func (t Token) RawSGR() (string, bool) {
	value := string(t)
	if strings.HasPrefix(value, ansiTokenPrefix) {
		return value[len(ansiTokenPrefix):], true
	}
	return "", false
}

// Style is an explicit, theme-free style: the v2 wire format programs send
// ("fg:#RRGGBB;bg:#RRGGBB;bold;dim;reverse"). The host translates it to SGR
// verbatim; no palette, theme or configuration is involved.
type Style struct {
	FG        string
	BG        string
	Bold      bool
	Dim       bool
	Italic    bool
	Underline bool
	Reverse   bool
}

// styleAttributes maps the accepted attribute spellings to their Style flag.
var styleAttributes = map[string]func(*Style){
	"bold":      func(s *Style) { s.Bold = true },
	"dim":       func(s *Style) { s.Dim = true },
	"italic":    func(s *Style) { s.Italic = true },
	"underline": func(s *Style) { s.Underline = true },
	"reverse":   func(s *Style) { s.Reverse = true },
}

// ParseStyle parses an explicit style string. ok is false when the value is
// not explicit-style syntax (a named token or empty), in which case the
// caller may fall back to token resolution. An explicit value with unknown
// or malformed segments parses to an empty Style: unknown pieces are
// skipped, never fatal.
func ParseStyle(value string) (Style, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return Style{}, false
	}
	explicit := strings.ContainsAny(trimmed, ":;")
	for _, segment := range strings.Split(trimmed, ";") {
		if _, ok := styleAttributes[strings.TrimSpace(segment)]; ok {
			explicit = true
			break
		}
	}
	if !explicit {
		return Style{}, false
	}
	var style Style
	for _, segment := range strings.Split(trimmed, ";") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		if attr, ok := styleAttributes[segment]; ok {
			attr(&style)
			continue
		}
		switch {
		case strings.HasPrefix(segment, "fg:"):
			if color, ok := normalizeHexColor(strings.TrimPrefix(segment, "fg:")); ok {
				style.FG = color
			}
		case strings.HasPrefix(segment, "bg:"):
			if color, ok := normalizeHexColor(strings.TrimPrefix(segment, "bg:")); ok {
				style.BG = color
			}
		}
	}
	return style, true
}

// String serializes the style in the canonical wire format. An empty style
// serializes to "".
func (s Style) String() string {
	var parts []string
	if s.FG != "" {
		parts = append(parts, "fg:"+s.FG)
	}
	if s.BG != "" {
		parts = append(parts, "bg:"+s.BG)
	}
	for _, attr := range []struct {
		name string
		set  bool
	}{
		{"bold", s.Bold},
		{"dim", s.Dim},
		{"italic", s.Italic},
		{"underline", s.Underline},
		{"reverse", s.Reverse},
	} {
		if attr.set {
			parts = append(parts, attr.name)
		}
	}
	return strings.Join(parts, ";")
}

// SGR translates the style to one SGR sequence (without a trailing reset).
// An empty style yields the empty sequence.
func (s Style) SGR() string {
	var params []string
	if s.Bold {
		params = append(params, "1")
	}
	if s.Dim {
		params = append(params, "2")
	}
	if s.Italic {
		params = append(params, "3")
	}
	if s.Underline {
		params = append(params, "4")
	}
	if s.Reverse {
		params = append(params, "7")
	}
	if r, g, b, ok := parseHexColor(s.FG); ok {
		params = append(params, "38", "2", r, g, b)
	}
	if r, g, b, ok := parseHexColor(s.BG); ok {
		params = append(params, "48", "2", r, g, b)
	}
	if len(params) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(params, ";") + "m"
}

// normalizeHexColor rewrites a color to canonical lowercase "#rrggbb".
func normalizeHexColor(value string) (string, bool) {
	r, g, b, ok := parseHexColor(value)
	if !ok {
		return "", false
	}
	toHex := func(v string) string {
		n, _ := strconv.Atoi(v)
		if n < 16 {
			return "0" + strconv.FormatInt(int64(n), 16)
		}
		return strconv.FormatInt(int64(n), 16)
	}
	return "#" + toHex(r) + toHex(g) + toHex(b), true
}

// SGR resolves the token to an SGR sequence (without a trailing reset).
// Explicit styles translate theme-free; named tokens resolve against the
// theme. Unknown tokens and unparseable styles resolve to the empty
// sequence (safe degrade).
func (t Token) SGR(theme Theme) string {
	if params, ok := t.RawSGR(); ok {
		return "\x1b[" + params + "m"
	}
	if style, ok := ParseStyle(string(t)); ok {
		return style.SGR()
	}
	theme = theme.WithFallback()
	switch t {
	case "", TokenDefault:
		return sgrForeground(theme.Foreground, false)
	case TokenBackground:
		return sgrBackground(theme.Background)
	case TokenFG:
		return sgrForeground(theme.Foreground, false)
	case TokenForeground:
		return sgrForeground(theme.ChromeFG, false)
	case TokenStrongForeground:
		return sgrForeground(theme.ChromeFG, true)
	case TokenMuted:
		return sgrForeground(theme.Muted, false) + "\x1b[2m"
	case TokenAccent:
		return sgrForeground(theme.Accent, true)
	case TokenAccentDim:
		return sgrForeground(theme.AccentDim, false)
	case TokenSuccess, TokenOK:
		return sgrForeground(theme.OK, false)
	case TokenWarning:
		return sgrForeground(theme.Warning, false)
	case TokenDanger:
		return sgrForeground(theme.Danger, false)
	case TokenInfo:
		return sgrForeground(theme.Info, false)
	case TokenChrome, TokenHeader:
		return sgrForegroundBackground(theme.ChromeFG, theme.ChromeBG, true)
	case TokenChromeFocus:
		return sgrForegroundBackground(theme.ChromeFocusFG, theme.ChromeFocusBG, true)
	case TokenTabActive:
		return sgrForegroundBackground(theme.TabActiveFG, theme.TabActiveBG, true)
	case TokenTabInactive:
		return sgrForeground(theme.TabInactiveFG, false) + "\x1b[2m"
	case TokenFooter:
		return sgrForeground(theme.Muted, false)
	case TokenFooterAccent:
		return sgrForeground(theme.Accent, true)
	case TokenStatus:
		return sgrForegroundBackground(theme.StatusFG, theme.StatusBG, false)
	case TokenOverlay:
		return sgrForegroundBackground(theme.ChromeFG, theme.OverlayBG, false)
	case TokenBorder:
		return sgrForeground(theme.Border, false)
	case TokenBorderFocus, TokenActiveBorder:
		return sgrForeground(theme.BorderFocus, true)
	case TokenBorderDead:
		return sgrForeground(theme.BorderDead, false)
	case TokenInactiveBorder:
		return sgrForeground(theme.InactiveBorder, false)
	case TokenSelection:
		return sgrForegroundBackground(theme.SelectionFG, theme.SelectionBG, false)
	default:
		return ""
	}
}

func sgrForeground(hex string, bold bool) string {
	r, g, b, ok := parseHexColor(hex)
	if !ok {
		if bold {
			return "\x1b[1m"
		}
		return ""
	}
	prefix := "\x1b["
	if bold {
		prefix += "1;"
	}
	return prefix + "38;2;" + r + ";" + g + ";" + b + "m"
}

func sgrBackground(hex string) string {
	r, g, b, ok := parseHexColor(hex)
	if !ok {
		return ""
	}
	return "\x1b[48;2;" + r + ";" + g + ";" + b + "m"
}

func sgrForegroundBackground(fg string, bg string, bold bool) string {
	seq := sgrForeground(fg, bold)
	r, g, b, ok := parseHexColor(bg)
	if !ok {
		return seq
	}
	return seq + "\x1b[48;2;" + r + ";" + g + ";" + b + "m"
}

func parseHexColor(value string) (string, string, string, bool) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(value) != 6 {
		return "", "", "", false
	}
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if !isHexDigit(ch) {
			return "", "", "", false
		}
	}
	return strconv.Itoa(hexByte(value[0:2])), strconv.Itoa(hexByte(value[2:4])), strconv.Itoa(hexByte(value[4:6])), true
}

func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func hexByte(pair string) int {
	value := 0
	for i := 0; i < len(pair); i++ {
		value *= 16
		ch := pair[i]
		switch {
		case ch >= '0' && ch <= '9':
			value += int(ch - '0')
		case ch >= 'a' && ch <= 'f':
			value += int(ch-'a') + 10
		case ch >= 'A' && ch <= 'F':
			value += int(ch-'A') + 10
		}
	}
	return value
}

// VisibleText strips characters that cannot live in a single cell: newlines
// and other C0 controls become spaces so a text run never breaks a row.
func VisibleText(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r == '\n' || r == '\r' || r == '\t' || (r < 0x20) || (r >= 0x7F && r < 0xA0) {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

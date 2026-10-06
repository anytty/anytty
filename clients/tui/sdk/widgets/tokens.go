package widgets

import "strings"

// This file names the host style tokens (render/style.go) as exported string
// constants and adds tiny helpers that compose raw explicit-style strings
// ("fg:#RRGGBB;bg:#RRGGBB;bold") idempotently. A widget can pass either a
// token or a raw string to sdk.Builder.Style: the host resolves tokens against
// its palette and translates raw strings verbatim.
//
// The names mirror the render.Token values exactly; widgets must not import
// the host render package (the protocol is the only contract), so the values
// are repeated here as plain strings.
const (
	// StyleDefault is the host foreground on the terminal background.
	StyleDefault = "default"
	// StyleBackground is the chrome background color.
	StyleBackground = "bg"
	// StyleFG is the chrome foreground color.
	StyleFG = "fg"
	// StyleForeground is the v1 alias of the chrome foreground color.
	StyleForeground = "foreground"
	// StyleStrongForeground is the chrome foreground, bold.
	StyleStrongForeground = "strong-foreground"
	// StyleMuted is the muted foreground, dimmed.
	StyleMuted = "muted"
	// StyleAccent is the accent color, bold.
	StyleAccent = "accent"
	// StyleAccentDim is the accent color, not bold (secondary emphasis).
	StyleAccentDim = "accent_dim"
	// StyleSuccess is the success color (v1 alias of ok).
	StyleSuccess = "success"
	// StyleOK is the success color.
	StyleOK = "ok"
	// StyleWarning is the warning color.
	StyleWarning = "warning"
	// StyleDanger is the danger color.
	StyleDanger = "danger"
	// StyleInfo is the info color.
	StyleInfo = "info"
	// StyleChrome is the chrome band (foreground on chrome background).
	StyleChrome = "chrome"
	// StyleChromeFocus is the focused chrome band.
	StyleChromeFocus = "chrome_focus"
	// StyleHeader is the v1 alias of the chrome band.
	StyleHeader = "header"
	// StyleTabActive is the active tab block (foreground on tab background).
	StyleTabActive = "tab_active"
	// StyleTabInactive is the inactive tab label.
	StyleTabInactive = "tab_inactive"
	// StyleFooter is the footer text (v1 alias of muted).
	StyleFooter = "footer"
	// StyleFooterAccent is the emphasized footer text.
	StyleFooterAccent = "footer-accent"
	// StyleStatus is the status line (foreground on status background).
	StyleStatus = "status"
	// StyleOverlay is overlay body text on the overlay background.
	StyleOverlay = "overlay"
	// StyleBorder is the default box border.
	StyleBorder = "border"
	// StyleBorderFocus is the focused box border.
	StyleBorderFocus = "border_focus"
	// StyleBorderDead is the border of an exited/dead box.
	StyleBorderDead = "border_dead"
	// StyleActiveBorder is the v1 alias of the focused border.
	StyleActiveBorder = "active-border"
	// StyleInactiveBorder is the v1 alias of the muted border.
	StyleInactiveBorder = "inactive-border"
	// StyleSelection is the selected list item (foreground on selection
	// background).
	StyleSelection = "selection"
)

// styleHasSegment reports whether style already carries segment as a
// semicolon-separated piece.
func styleHasSegment(style, segment string) bool {
	if style == "" {
		return false
	}
	for _, part := range strings.Split(style, ";") {
		if strings.TrimSpace(part) == segment {
			return true
		}
	}
	return false
}

// WithBold appends the bold attribute unless the style already carries it.
// WithBold("") is "bold"; repeated calls are idempotent.
func WithBold(style string) string {
	if styleHasSegment(style, "bold") {
		return style
	}
	if style == "" {
		return "bold"
	}
	return style + ";bold"
}

// WithUnderline appends the underline attribute unless the style already
// carries it. WithUnderline("") is "underline"; repeated calls are idempotent.
func WithUnderline(style string) string {
	if styleHasSegment(style, "underline") {
		return style
	}
	if style == "" {
		return "underline"
	}
	return style + ";underline"
}

// WithReverse appends the reverse attribute unless the style already carries
// it. WithReverse("") is "reverse"; repeated calls are idempotent.
func WithReverse(style string) string {
	if styleHasSegment(style, "reverse") {
		return style
	}
	if style == "" {
		return "reverse"
	}
	return style + ";reverse"
}

// WithFg sets the foreground color of a raw style string. An existing "fg:"
// segment is replaced in place, otherwise the color is prepended; an empty hex
// leaves the style untouched. WithFg("", "#aabbcc") is "fg:#aabbcc" and
// repeated calls with the same hex are idempotent.
func WithFg(style, hex string) string {
	return setStyleColor(style, "fg", hex)
}

// WithBg sets the background color of a raw style string, replacing an
// existing "bg:" segment or appending it. An empty hex leaves the style
// untouched; repeated calls with the same hex are idempotent.
func WithBg(style, hex string) string {
	return setStyleColor(style, "bg", hex)
}

// setStyleColor replaces or inserts one "key:hex" color segment.
func setStyleColor(style, key, hex string) string {
	if hex == "" {
		return style
	}
	segment := key + ":" + hex
	var parts []string
	replaced := false
	for _, part := range strings.Split(style, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, key+":") {
			if replaced {
				continue
			}
			parts = append(parts, segment)
			replaced = true
			continue
		}
		parts = append(parts, part)
	}
	if !replaced {
		parts = append([]string{segment}, parts...)
	}
	return strings.Join(parts, ";")
}

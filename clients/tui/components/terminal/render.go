package terminal

import (
	"strings"

	"github.com/anytty/anytty/clients/tui/render"
)

// Render draws the component at its own origin: lines use coordinates
// relative to (0, 0) with X in [0, width) and Y in [0, height). The runtime
// translates them to the placement rect and blits them into the framebuffer.
//
// The chrome follows the v1 terminal component: a titled border, a focus
// marker, an exit badge and a scrollback badge:
//
//	┌─▎main [↑3] ─────┐
//	│ …content…       │
//	└─────────────────┘
//
// The component declares its own inset (Inset(width, height)): the runtime
// derives the PTY winsize and the cursor offset from it. Chrome colors come
// from the program-declared props (content.props: chrome.border / border_focus
// / border_dead for the frame, chrome.title / chrome.badge for the label) and
// fall back to the built-in semantic tokens when a key is absent. Content
// rows keep their own style tokens.
func (c *Component) Render(width, height int) []render.Line {
	if width <= 0 || height <= 0 {
		return nil
	}
	props := c.props.withDefaults()
	inset := c.Inset(width, height)
	bordered := inset == DefaultInset
	contentW, contentH := width-2*inset, height-2*inset
	if contentH < 0 {
		contentH = 0
	}
	if contentW < 0 {
		contentW = 0
	}
	// Remember the visible content height: it is the window size requested
	// from the history port on scroll.
	c.visible = contentH

	border := borderToken(props)
	lines := make([]render.Line, 0, height)
	if bordered {
		lines = append(lines, topBorderLines(width, props, border)...)
	}
	screen := c.visibleScreen()
	for row := 0; row < contentH; row++ {
		y := row + inset
		x := inset
		if bordered {
			lines = append(lines, render.Line{X: 0, Y: y, Text: "│", Style: border})
		}
		if row < len(screen.Lines) {
			lines = append(lines, rowLines(screen.Lines[row], x, y, contentW)...)
		}
		if bordered {
			lines = append(lines, render.Line{X: width - 1, Y: y, Text: "│", Style: border})
		}
	}
	if bordered {
		lines = append(lines, render.Line{
			X:     0,
			Y:     height - 1,
			Text:  "└" + strings.Repeat("─", width-2) + "┘",
			Style: border,
		})
	}
	return lines
}

// borderToken follows the v1 precedence (exited wins over focused, focused
// over inactive). A program-declared chrome style (content.props) wins over
// the built-in semantic token; the token remains the default so a program
// that sends no props keeps the golden v1 look.
func borderToken(props Props) render.Token {
	switch {
	case props.Exited:
		return chromeStyle(props, PropBorderDead, render.TokenBorderDead)
	case props.Focused:
		return chromeStyle(props, PropBorderFocus, render.TokenBorderFocus)
	default:
		return chromeStyle(props, PropBorder, render.TokenBorder)
	}
}

// chromeStyle resolves one program-declared chrome prop: the explicit style
// string when present, the built-in default token otherwise. Unknown keys are
// irrelevant here; an unparsable value degrades to no SGR in the renderer.
func chromeStyle(props Props, key string, fallback render.Token) render.Token {
	if value := strings.TrimSpace(props.Chrome[key]); value != "" {
		return render.Token(value)
	}
	return fallback
}

// titleSegments splits the decorated title into the base title and the
// appended badges ([exited N], [↑M]), so each part can carry its own style.
func titleSegments(props Props) (title, badges string) {
	title = strings.TrimSpace(props.Title)
	if title == "" {
		title = "terminal"
	}
	if props.Exited {
		// Legacy parity: a zero exit code prints the bare badge, the code is
		// only shown when it carries information ([exited] vs [exited 3]).
		if props.ExitCode != 0 {
			badges += " [exited " + itoa(props.ExitCode) + "]"
		} else {
			badges += " [exited]"
		}
	}
	if props.Scrolled {
		badges += " [↑" + itoa(props.ScrollOffset) + "]"
	}
	return title, badges
}

// TitleText is the decorated title shown in the border: the base title plus
// the exit badge and the scrollback badge, in that order.
func TitleText(props Props) string {
	title, badges := titleSegments(props)
	return title + badges
}

// titleRun is one styled piece of the border label.
type titleRun struct {
	text  string
	style render.Token
}

// topBorderLines builds the top border as styled runs: "┌", the label pieces
// (base title and badges may use different chrome styles) and the rule.
func topBorderLines(width int, props Props, border render.Token) []render.Line {
	runs := titleRuns(width, props, border)
	lines := []render.Line{{X: 0, Y: 0, Text: "┌", Style: border}}
	labelWidth := 0
	for _, run := range runs {
		lines = append(lines, render.Line{X: 1 + labelWidth, Y: 0, Text: run.text, Style: run.style})
		labelWidth += render.DisplayWidth(run.text)
	}
	dashes := width - 2 - labelWidth
	if dashes < 0 {
		dashes = 0
	}
	lines = append(lines, render.Line{
		X:     1 + labelWidth,
		Y:     0,
		Text:  strings.Repeat("─", dashes) + "┐",
		Style: border,
	})
	return lines
}

// titleRuns mirrors the v1 label algorithm byte for byte: " " (plus a focus
// marker) + decorated title + " ", truncated to always leave at least one
// dash on the top rule. The base title uses the chrome.title prop (default:
// the border style) and the badges use the chrome.badge prop (default: the
// border style); runs with the same style merge, so without props the output
// is the single v1 run. A box narrower than four cells keeps an empty label.
func titleRuns(width int, props Props, border render.Token) []titleRun {
	if width < 4 {
		return nil
	}
	title, badges := titleSegments(props)
	marker := ""
	if props.Focused && !props.Exited {
		marker = "▎"
	}
	prefix := " " + marker
	available := width - 2 - render.DisplayWidth(prefix) - 1
	if available <= 0 {
		return nil
	}
	truncated := render.Truncate(title+badges, available)
	titleStyle := chromeStyle(props, PropTitle, border)
	badgeStyle := chromeStyle(props, PropBadge, border)

	runs := make([]titleRun, 0, 3)
	if prefix != "" {
		runs = append(runs, titleRun{text: prefix, style: titleStyle})
	}
	switch {
	case truncated == "":
	case strings.HasPrefix(title, truncated):
		// Truncation stopped inside (or exactly at) the base title.
		runs = append(runs, titleRun{text: truncated, style: titleStyle})
	case strings.HasPrefix(truncated, title):
		runs = append(runs, titleRun{text: title, style: titleStyle})
		if rest := truncated[len(title):]; rest != "" {
			runs = append(runs, titleRun{text: rest, style: badgeStyle})
		}
	default:
		// Unreachable for a cluster-prefix truncation; keep the title style.
		runs = append(runs, titleRun{text: truncated, style: titleStyle})
	}
	runs = append(runs, titleRun{text: " ", style: titleStyle})
	return mergeTitleRuns(runs)
}

// mergeTitleRuns concatenates adjacent runs that share a style, so a label
// without per-part styles stays one render run (the v1 shape).
func mergeTitleRuns(runs []titleRun) []titleRun {
	out := runs[:0]
	for _, run := range runs {
		if run.text == "" {
			continue
		}
		if n := len(out); n > 0 && out[n-1].style == run.style {
			out[n-1].text += run.text
			continue
		}
		out = append(out, run)
	}
	return out
}

// rowLines converts one screen row into styled runs, clipped to maxWidth
// cells. A wide cluster that does not fully fit is dropped, never split.
func rowLines(cells []Cell, x, y, maxWidth int) []render.Line {
	if maxWidth <= 0 || len(cells) == 0 {
		return nil
	}
	var lines []render.Line
	remaining := maxWidth
	currentX := x
	var run strings.Builder
	runStyle := render.Token("")
	runWidth := 0
	flush := func() {
		if run.Len() == 0 {
			return
		}
		lines = append(lines, render.Line{X: currentX, Y: y, Text: run.String(), Style: runStyle})
		currentX += runWidth
		run.Reset()
		runWidth = 0
	}
	for _, cell := range cells {
		width := cell.Width
		if width <= 0 {
			width = render.DisplayWidth(cell.Text)
		}
		if width <= 0 {
			continue
		}
		if width > remaining {
			break
		}
		style := cell.Style
		if style == "" {
			style = render.TokenDefault
		}
		if runStyle != "" && style != runStyle {
			flush()
		}
		runStyle = style
		run.WriteString(cell.Text)
		runWidth += width
		remaining -= width
	}
	flush()
	return lines
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var digits [20]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		i--
		digits[i] = '-'
	}
	return string(digits[i:])
}

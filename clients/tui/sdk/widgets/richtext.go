package widgets

import (
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
)

// Span is one styled run of rich text. Style is an opaque style string: a
// host-internal token name (see tokens.go) or an explicit raw style such as
// "fg:#RRGGBB;bg:#RRGGBB;bold;underline".
type Span struct {
	Text  string
	Style string
}

// RichText is a single row of styled runs. Width > 0 declares the row width
// (the row is padded so it occupies Width cells). Wrap is advisory metadata
// for callers that reflow: Build renders the spans as one row.
type RichText struct {
	Spans []Span
	Width int
	Wrap  bool
}

// Styled returns a Span of text rendered with style.
func Styled(text, style string) Span { return Span{Text: text, Style: style} }

// Line is the variadic constructor of a RichText from spans.
func Line(spans ...Span) RichText { return RichText{Spans: spans} }

// Text returns the plain text of the spans with no styling, for measurement.
func (r RichText) Text() string { return spansText(r.Spans) }

// Build renders the spans as a one-row box tree: one text box per run, with
// adjacent runs that share a style merged so the host receives the fewest
// possible boxes. Runs with empty text are dropped. A declared Width pads the
// row with a trailing unstyled box; an empty RichText still yields a row.
func (r RichText) Build() *sdk.Builder {
	row := sdk.Row().Height(1)
	spans := mergeSpans(r.Spans)
	used := 0
	for _, span := range spans {
		box := sdk.Text(span.Text)
		if span.Style != "" {
			box.Style(span.Style)
		}
		row.Child(box)
		used += sdk.DisplayWidth(span.Text)
	}
	if pad := r.Width - used; pad > 0 {
		row.Child(sdk.Text(strings.Repeat(" ", pad)))
	}
	return row
}

// Styled returns the RichText re-styled as a single span.
func (r RichText) Styled(style string) RichText {
	return RichText{Spans: []Span{Styled(r.Text(), style)}, Width: r.Width, Wrap: r.Wrap}
}

// WrapSpans wraps the RichText to width and marks every resulting row as
// wrapped.
func (r RichText) WrapSpans(width int) []RichText {
	wrapped := WrapSpans(r.Spans, width)
	for i := range wrapped {
		wrapped[i].Wrap = true
	}
	return wrapped
}

// mergeSpans coalesces adjacent runs with the same style and drops empty runs.
func mergeSpans(spans []Span) []Span {
	out := make([]Span, 0, len(spans))
	for _, span := range spans {
		if span.Text == "" {
			continue
		}
		if n := len(out); n > 0 && out[n-1].Style == span.Style {
			out[n-1].Text += span.Text
			continue
		}
		out = append(out, span)
	}
	return out
}

// spansText concatenates the text of spans.
func spansText(spans []Span) string {
	var b strings.Builder
	for _, span := range spans {
		b.WriteString(span.Text)
	}
	return b.String()
}

// WrapText hard-wraps s into lines of at most width display cells. It is
// CJK/wide-char aware: a wide rune is never split across lines. It breaks on
// spaces when possible, drops the break space, and hard-breaks words longer
// than the width. Embedded newlines force a line break. A width <= 0 yields
// nil.
func WrapText(s string, width int) []string {
	if width <= 0 {
		return nil
	}
	runes := []rune(s)
	indexLines := wrapIndexes(runes, width)
	out := make([]string, 0, len(indexLines))
	for _, line := range indexLines {
		var b strings.Builder
		for _, i := range line {
			b.WriteRune(runes[i])
		}
		out = append(out, b.String())
	}
	return out
}

// WrapSpans wraps styled runs into lines of at most width display cells. Runs
// are split at wrap boundaries with their style preserved; a wide rune is
// never split. Line breaks are decided on the plain text, so styling never
// changes the layout. A width <= 0 yields nil.
func WrapSpans(spans []Span, width int) []RichText {
	if width <= 0 {
		return nil
	}
	var runes []rune
	var styles []string
	for _, span := range spans {
		for _, r := range span.Text {
			runes = append(runes, r)
			styles = append(styles, span.Style)
		}
	}
	indexLines := wrapIndexes(runes, width)
	out := make([]RichText, 0, len(indexLines))
	for _, line := range indexLines {
		out = append(out, RichText{Spans: groupRunes(runes, styles, line), Width: width})
	}
	return out
}

// groupRunes builds spans from the rune indexes of one wrapped line, merging
// adjacent runes that share a style.
func groupRunes(runes []rune, styles []string, indexes []int) []Span {
	var out []Span
	for _, i := range indexes {
		if n := len(out); n > 0 && out[n-1].Style == styles[i] {
			out[n-1].Text += string(runes[i])
			continue
		}
		out = append(out, Span{Text: string(runes[i]), Style: styles[i]})
	}
	return out
}

// wrapIndexes is the shared wrap core. Every returned line is a list of runes
// indexes into the source, so callers can recover either the plain text or the
// per-rune styles. Lines never exceed width cells; a wide rune that cannot fit
// on a non-empty line moves to the next one (a wide rune is never split, so a
// width < 2 line may hold a single wide rune).
func wrapIndexes(runes []rune, width int) [][]int {
	var lines [][]int
	var cur []int
	curWidth := 0
	flush := func(force bool) {
		for len(cur) > 0 && runes[cur[len(cur)-1]] == ' ' {
			cur = cur[:len(cur)-1]
		}
		if len(cur) > 0 || force {
			lines = append(lines, cur)
		}
		cur = nil
		curWidth = 0
	}
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if ch == '\n' {
			flush(true)
			continue
		}
		if ch == ' ' && curWidth == 0 {
			continue
		}
		rw := sdk.RuneWidth(ch)
		if curWidth+rw > width {
			if ch == ' ' {
				flush(false)
				continue
			}
			if last := lastSpaceIndex(cur, runes); last >= 0 {
				tail := append([]int(nil), cur[last+1:]...)
				cur = cur[:last]
				flush(false)
				tailWidth := 0
				for _, idx := range tail {
					tailWidth += sdk.RuneWidth(runes[idx])
				}
				if tailWidth+rw <= width {
					cur = tail
					curWidth = tailWidth
					cur = append(cur, i)
					curWidth += rw
					continue
				}
				cur = append(cur, i)
				curWidth = rw
				flush(false)
				continue
			}
			flush(false)
		}
		cur = append(cur, i)
		curWidth += rw
	}
	flush(len(lines) == 0)
	return lines
}

// lastSpaceIndex returns the index (within line) of the last space rune.
func lastSpaceIndex(line []int, runes []rune) int {
	for i := len(line) - 1; i >= 0; i-- {
		if runes[line[i]] == ' ' {
			return i
		}
	}
	return -1
}

// ParseEmphasis parses a tiny emphasis grammar into spans: "**bold**" becomes
// bold, "_em_" becomes italic, plain text keeps the base style. Unclosed,
// nested or empty markers stay literal.
func ParseEmphasis(s string) []Span { return ParseEmphasisStyled(s, "") }

// ParseEmphasisStyled is ParseEmphasis with a base style composed onto every
// span; emphasis attributes are appended to it.
func ParseEmphasisStyled(s, base string) []Span {
	var out []Span
	var plain strings.Builder
	flushPlain := func() {
		if plain.Len() > 0 {
			out = append(out, Span{Text: plain.String(), Style: base})
			plain.Reset()
		}
	}
	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], "**") {
			if end := strings.Index(s[i+2:], "**"); end > 0 {
				flushPlain()
				out = append(out, Span{Text: s[i+2 : i+2+end], Style: WithBold(base)})
				i += 2 + end + 2
				continue
			}
		}
		if s[i] == '_' {
			if end := strings.IndexByte(s[i+1:], '_'); end > 0 {
				flushPlain()
				out = append(out, Span{Text: s[i+1 : i+1+end], Style: WithItalic(base)})
				i += 1 + end + 1
				continue
			}
		}
		plain.WriteByte(s[i])
		i++
	}
	flushPlain()
	return out
}

// WithItalic appends the italic attribute unless the style already carries it.
func WithItalic(style string) string {
	if styleHasSegment(style, "italic") {
		return style
	}
	if style == "" {
		return "italic"
	}
	return style + ";italic"
}

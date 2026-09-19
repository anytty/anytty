package render

import (
	"strings"

	"github.com/rivo/uniseg"
)

// Cluster is one extended grapheme cluster together with its display width.
// A cluster is the smallest unit a framebuffer cell can hold: emoji ZWJ
// sequences, flags, skin-tone modified emoji and combining marks each stay
// glued to their base rune.
type Cluster struct {
	Text  string
	Width int
}

// Clusters segments s into grapheme clusters. It never returns nil entries;
// an empty string yields an empty slice.
func Clusters(s string) []Cluster {
	if s == "" {
		return nil
	}
	g := uniseg.NewGraphemes(s)
	out := make([]Cluster, 0, len(s))
	for g.Next() {
		text := g.Str()
		out = append(out, Cluster{Text: text, Width: uniseg.StringWidth(text)})
	}
	return out
}

// RuneWidth returns the display width of a single rune: 0 for controls,
// combining marks and zero-width runes, 2 for wide (CJK/emoji) runes, 1
// otherwise. Use DisplayWidth for anything longer than one rune.
func RuneWidth(r rune) int {
	return uniseg.StringWidth(string(r))
}

// DisplayWidth returns the display width of s in terminal cells. Width is
// measured per grapheme cluster, so "👨‍👩‍👧‍👦" is 2, "🇨🇳" is 2, "👍🏽" is 2 and
// both "é" (precomposed) and "e\u0301" are 1.
//
// Control characters are width 0 and therefore never advance the cursor;
// callers must strip or replace them before writing to a cell grid.
func DisplayWidth(s string) int {
	return uniseg.StringWidth(s)
}

// Truncate clips s to at most maxCells display cells without splitting a
// grapheme cluster in half. A zero or negative budget yields "".
func Truncate(s string, maxCells int) string {
	if maxCells <= 0 {
		return ""
	}
	var b strings.Builder
	width := 0
	for _, c := range Clusters(s) {
		if width+c.Width > maxCells {
			break
		}
		b.WriteString(c.Text)
		width += c.Width
	}
	return b.String()
}

// PadRight truncates s to width cells and pads it with spaces to exactly
// width cells.
func PadRight(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = Truncate(s, width)
	if pad := width - DisplayWidth(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

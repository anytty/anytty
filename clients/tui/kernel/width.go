package kernel

import "strings"

// wideRanges is a minimal east-asian-width table: runes in these ranges are
// rendered with width 2 (CJK, fullwidth forms, common emoji, CJK extension
// planes). Everything else is width 1 unless it is a zero-width rune.
var wideRanges = [...][2]rune{
	{0x1100, 0x115F},   // Hangul Jamo
	{0x2329, 0x232A},   // angle brackets
	{0x2E80, 0x303E},   // CJK radicals, Kangxi, CJK symbols
	{0x3041, 0x33FF},   // Kana, CJK compatibility
	{0x3400, 0x4DBF},   // CJK extension A
	{0x4E00, 0x9FFF},   // CJK unified ideographs
	{0xA000, 0xA4CF},   // Yi
	{0xA960, 0xA97F},   // Hangul Jamo extended A
	{0xAC00, 0xD7A3},   // Hangul syllables
	{0xF900, 0xFAFF},   // CJK compatibility ideographs
	{0xFE10, 0xFE19},   // vertical forms
	{0xFE30, 0xFE6F},   // CJK compatibility forms
	{0xFF00, 0xFF60},   // fullwidth forms
	{0xFFE0, 0xFFE6},   // fullwidth signs
	{0x16FE0, 0x16FE4}, // Tangut components
	{0x17000, 0x18AFF}, // Tangut
	{0x1B000, 0x1B2FF}, // Kana supplement
	{0x1F000, 0x1F02F}, // Mahjong tiles
	{0x1F0CF, 0x1F0CF}, // joker
	{0x1F18E, 0x1F18E}, // enclosed AB
	{0x1F191, 0x1F19A}, // enclosed symbols
	{0x1F1E6, 0x1F1FF}, // regional indicators
	{0x1F200, 0x1F2FF}, // enclosed ideographic supplement
	{0x1F300, 0x1F64F}, // misc symbols, pictographs, emoticons
	{0x1F680, 0x1F6FF}, // transport and map
	{0x1F7E0, 0x1F7EB}, // colored circles and squares
	{0x1F900, 0x1F9FF}, // supplemental symbols and pictographs
	{0x1FA70, 0x1FAFF}, // symbols and pictographs extended A
	{0x20000, 0x3FFFD}, // CJK extensions B..F
}

// RuneWidth returns the display width of r: 0 for controls, combining marks
// and zero-width runes, 2 for wide (CJK/emoji) runes, 1 otherwise.
func RuneWidth(r rune) int {
	switch {
	case r == 0:
		return 0
	case r < 0x20 || (r >= 0x7F && r < 0xA0):
		return 0
	case r >= 0x0300 && r <= 0x036F:
		return 0
	case r >= 0x0483 && r <= 0x0489:
		return 0
	case r >= 0x0591 && r <= 0x05BD:
		return 0
	case r >= 0x1AB0 && r <= 0x1AFF:
		return 0
	case r >= 0x1DC0 && r <= 0x1DFF:
		return 0
	case r >= 0x200B && r <= 0x200F:
		return 0
	case r >= 0x2028 && r <= 0x202E:
		return 0
	case r >= 0x2060 && r <= 0x2064:
		return 0
	case r >= 0x20D0 && r <= 0x20FF:
		return 0
	case r >= 0xFE00 && r <= 0xFE0F:
		return 0
	case r >= 0xFE20 && r <= 0xFE2F:
		return 0
	case r == 0xFEFF:
		return 0
	}
	for _, rng := range wideRanges {
		if r >= rng[0] && r <= rng[1] {
			return 2
		}
	}
	return 1
}

// DisplayWidth returns the display width of s, treating CJK and emoji runes
// as width 2 and combining/zero-width runes as width 0.
func DisplayWidth(s string) int {
	width := 0
	for _, r := range s {
		width += RuneWidth(r)
	}
	return width
}

// Truncate clips s to at most maxWidth display cells without splitting a
// wide rune in half. A zero or negative maxWidth yields "".
func Truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	var b strings.Builder
	width := 0
	for _, r := range s {
		rw := RuneWidth(r)
		if width+rw > maxWidth && rw > 0 {
			break
		}
		b.WriteRune(r)
		width += rw
	}
	return b.String()
}

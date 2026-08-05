package render

import (
	"strings"

	xansi "github.com/charmbracelet/x/ansi"
)

func DisplayWidth(value string) int {
	return xansi.StringWidth(SafeLine(value))
}

func TruncateCells(value string, width int) string {
	if width <= 0 {
		return ""
	}
	return xansi.Truncate(SafeLine(value), width, "")
}

func SliceCells(value string, left int, right int) string {
	if right <= left {
		return ""
	}
	return xansi.Cut(SafeLine(value), left, right)
}

func PadRightCells(value string, width int) string {
	if width <= 0 {
		return ""
	}
	value = TruncateCells(value, width)
	pad := width - DisplayWidth(value)
	if pad <= 0 {
		return value
	}
	return value + strings.Repeat(" ", pad)
}

func FitText(value string, width int) string {
	return PadRightCells(value, width)
}

func LineFromText(value string, width int) Line {
	if width > 0 {
		value = FitText(value, width)
	} else {
		value = SafeLine(value)
	}
	return NewLine(value)
}

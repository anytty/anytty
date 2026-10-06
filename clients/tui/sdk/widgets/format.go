package widgets

import (
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// This file holds the numeric/text formatters the chart widgets (and callers)
// use to turn raw values into fixed, deterministic cell strings. Every helper
// is pure, allocation-light and free of ANSI escapes and CJK width logic:
// values are rendered as plain ASCII cells.
//
// Non-finite inputs are not errors. FormatFloat, FormatPercent and ScaleValue
// keep Go's canonical spellings ("NaN", "+Inf", "-Inf") so a bad sample is
// visible instead of silently zeroed.

// Scale suffix ramp for ScaleValue (base 1000).
var scaleSuffixes = []string{"", "K", "M", "G", "T", "P", "E"}

// byteUnits is the FormatBytes ramp (base 1024).
var byteUnits = []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}

// FormatBytes renders n as a base-1024 size with a unit suffix: "0 B",
// "512 B", "1 KB", "1.5 MB", "8 EB". Integral values keep no decimal;
// fractional values keep one. The sign is preserved for negative n.
func FormatBytes(n int64) string {
	if n == 0 {
		return "0 B"
	}
	negative := n < 0
	magnitude := uint64(n)
	if negative {
		magnitude = ^magnitude + 1
	}
	value := float64(magnitude)
	unit := 0
	for value >= 1024 && unit < len(byteUnits)-1 {
		value /= 1024
		unit++
	}
	text := formatTrimmed(value, 1) + " " + byteUnits[unit]
	if negative {
		return "-" + text
	}
	return text
}

// FormatCount renders n with "," thousands separators: 0, 42, "1,234,567".
// Negative values keep their sign ("-1,000").
func FormatCount(n int64) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	magnitude := uint64(n)
	if negative {
		magnitude = ^magnitude + 1
	}
	digits := strconv.FormatUint(magnitude, 10)
	var b strings.Builder
	if negative {
		b.WriteByte('-')
	}
	for i := 0; i < len(digits); i++ {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteByte(digits[i])
	}
	return b.String()
}

// FormatDuration renders d in the largest unit that stays readable:
//
//	0            -> "0s"
//	500ns        -> "500ns"
//	500µs        -> "500µs"
//	250ms        -> "250ms"
//	1500ms       -> "1.5s"
//	90s          -> "1m30s"
//	3600s        -> "1h"
//	3661s        -> "1h1m1s"
//
// Sub-second values keep one trimmed decimal (ms/µs/ns); second values keep
// one trimmed decimal; zero sub-components above the leading unit are dropped
// ("1h", "1m"), so the output is stable across runs. Negative durations keep
// their sign.
func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	negative := d < 0
	nanos := uint64(d)
	if negative {
		nanos = ^nanos + 1
	}
	text := formatDurationNanos(nanos)
	if negative {
		return "-" + text
	}
	return text
}

const (
	nanosPerMicro = uint64(1000)
	nanosPerMilli = uint64(1000) * nanosPerMicro
	nanosPerSec   = uint64(1000) * nanosPerMilli
	nanosPerMin   = uint64(60) * nanosPerSec
	nanosPerHour  = uint64(60) * nanosPerMin
)

func formatDurationNanos(nanos uint64) string {
	switch {
	case nanos < nanosPerMicro:
		return strconv.FormatUint(nanos, 10) + "ns"
	case nanos < nanosPerMilli:
		return strconv.FormatUint((nanos+nanosPerMicro/2)/nanosPerMicro, 10) + "µs"
	case nanos < nanosPerSec:
		return formatTrimmed(float64(nanos)/float64(nanosPerMilli), 1) + "ms"
	case nanos < nanosPerMin:
		return formatSeconds(nanos)
	case nanos < nanosPerHour:
		minutes := nanos / nanosPerMin
		remainder := nanos % nanosPerMin
		if remainder == 0 {
			return strconv.FormatUint(minutes, 10) + "m"
		}
		return strconv.FormatUint(minutes, 10) + "m" + formatSeconds(remainder)
	default:
		hours := nanos / nanosPerHour
		remainder := nanos % nanosPerHour
		if remainder == 0 {
			return strconv.FormatUint(hours, 10) + "h"
		}
		text := strconv.FormatUint(hours, 10) + "h"
		if remainder < nanosPerMin {
			return text + formatSeconds(remainder)
		}
		minutes := remainder / nanosPerMin
		seconds := remainder % nanosPerMin
		if seconds == 0 {
			return text + strconv.FormatUint(minutes, 10) + "m"
		}
		return text + strconv.FormatUint(minutes, 10) + "m" + formatSeconds(seconds)
	}
}

// formatSeconds renders a positive sub-minute duration (0 < nanos < 1m).
func formatSeconds(nanos uint64) string {
	if nanos < nanosPerSec {
		return formatTrimmed(float64(nanos)/float64(nanosPerMilli), 1) + "ms"
	}
	return formatTrimmed(float64(nanos)/float64(nanosPerSec), 1) + "s"
}

// FormatPercent renders a fraction as a percentage with the requested number
// of decimals: FormatPercent(0.125, 1) == "12.5%". f is the fraction, not the
// percentage. A negative digits falls back to 0. Non-finite fractions keep
// Go's spelling ("NaN%", "+Inf%").
func FormatPercent(f float64, digits int) string {
	if digits < 0 {
		digits = 0
	}
	return strconv.FormatFloat(f*100, 'f', digits, 64) + "%"
}

// FormatFloat renders f with digits decimals and right-aligns it in width
// display cells (PadLeft). width <= 0 leaves the text unpadded. digits < 0
// falls back to 0.
func FormatFloat(f float64, digits int, width int) string {
	if digits < 0 {
		digits = 0
	}
	return PadLeft(strconv.FormatFloat(f, 'f', digits, 64), width)
}

// ScaleValue divides v down to a base-1000 magnitude and returns the scaled
// value plus its suffix: 999 -> (999, ""), 1500 -> (1.5, "K"),
// 2_000_000 -> (2, "M"). Values below 1000 are returned unchanged. NaN/Inf
// are returned as-is with an empty suffix.
func ScaleValue(v float64) (float64, string) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return v, ""
	}
	scaled := v
	index := 0
	abs := math.Abs(v)
	for abs >= 1000 && index < len(scaleSuffixes)-1 {
		abs /= 1000
		scaled /= 1000
		index++
	}
	return scaled, scaleSuffixes[index]
}

// PadLeft right-aligns s in width cells, padding with ASCII spaces. s is never
// truncated; a width <= its rune count is returned unchanged.
func PadLeft(s string, width int) string {
	if pad := width - utf8.RuneCountInString(s); pad > 0 {
		return strings.Repeat(" ", pad) + s
	}
	return s
}

// PadRight left-aligns s in width cells, padding with ASCII spaces. s is never
// truncated; a width <= its rune count is returned unchanged.
func PadRight(s string, width int) string {
	if pad := width - utf8.RuneCountInString(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

// formatTrimmed formats value with digits decimals and removes a trailing
// fractional zero run ("1.0" -> "1", "1.50" -> "1.5"). Integers stay
// integral; non-finite values pass through untouched.
func formatTrimmed(value float64, digits int) string {
	text := strconv.FormatFloat(value, 'f', digits, 64)
	if strings.Contains(text, ".") {
		text = strings.TrimRight(text, "0")
		text = strings.TrimRight(text, ".")
	}
	return text
}

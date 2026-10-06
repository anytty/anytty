"""Numeric/text formatters (port of the Go ``sdk/widgets`` format.go).

Every helper is pure and deterministic: raw values become fixed cell strings.
Non-finite inputs are not errors: ``format_float``, ``format_percent`` and
``scale_value`` keep Go's canonical spellings ("NaN", "+Inf", "-Inf") so a bad
sample is visible instead of silently zeroed.
"""

import math

# Scale suffix ramp for scale_value (base 1000).
SCALE_SUFFIXES = ["", "K", "M", "G", "T", "P", "E"]

# BYTE_UNITS is the format_bytes ramp (base 1024).
BYTE_UNITS = ["B", "KB", "MB", "GB", "TB", "PB", "EB"]

_NANOS_PER_MICRO = 1000
_NANOS_PER_MILLI = 1000 * _NANOS_PER_MICRO
_NANOS_PER_SEC = 1000 * _NANOS_PER_MILLI
_NANOS_PER_MIN = 60 * _NANOS_PER_SEC
_NANOS_PER_HOUR = 60 * _NANOS_PER_MIN


def _go_format_float(value, digits):
    """Format like Go's strconv.FormatFloat(value, 'f', digits, 64)."""
    if math.isnan(value):
        return "NaN"
    if math.isinf(value):
        return "+Inf" if value > 0 else "-Inf"
    return "%.*f" % (digits, value)


def _format_trimmed(value, digits):
    """Format value with digits decimals, removing a trailing zero run."""
    text = _go_format_float(value, digits)
    if "." in text:
        text = text.rstrip("0")
        text = text.rstrip(".")
    return text


def format_bytes(n):
    """Render n as a base-1024 size with a unit suffix: "0 B", "512 B",
    "1 KB", "1.5 MB", "8 EB". Integral values keep no decimal; fractional
    values keep one. The sign is preserved for negative n."""
    n = int(n)
    if n == 0:
        return "0 B"
    negative = n < 0
    magnitude = -n if negative else n
    value = float(magnitude)
    unit = 0
    while value >= 1024 and unit < len(BYTE_UNITS) - 1:
        value /= 1024
        unit += 1
    text = _format_trimmed(value, 1) + " " + BYTE_UNITS[unit]
    if negative:
        return "-" + text
    return text


def format_count(n):
    """Render n with "," thousands separators: 0, 42, "1,234,567"."""
    n = int(n)
    if n == 0:
        return "0"
    negative = n < 0
    magnitude = -n if negative else n
    digits = str(magnitude)
    out = []
    if negative:
        out.append("-")
    for i, ch in enumerate(digits):
        if i > 0 and (len(digits) - i) % 3 == 0:
            out.append(",")
        out.append(ch)
    return "".join(out)


def format_duration(d):
    """Render d nanoseconds in the largest readable unit:

        0       -> "0s"
        500ns   -> "500ns"
        500us   -> "500\u00b5s"
        250ms   -> "250ms"
        1500ms  -> "1.5s"
        90s     -> "1m30s"
        3600s   -> "1h"
        3661s   -> "1h1m1s"

    Negative durations keep their sign."""
    d = int(d)
    if d == 0:
        return "0s"
    negative = d < 0
    nanos = -d if negative else d
    text = _format_duration_nanos(nanos)
    if negative:
        return "-" + text
    return text


def _format_duration_nanos(nanos):
    if nanos < _NANOS_PER_MICRO:
        return str(nanos) + "ns"
    if nanos < _NANOS_PER_MILLI:
        return str((nanos + _NANOS_PER_MICRO // 2) // _NANOS_PER_MICRO) + "\u00b5s"
    if nanos < _NANOS_PER_SEC:
        return _format_trimmed(float(nanos) / float(_NANOS_PER_MILLI), 1) + "ms"
    if nanos < _NANOS_PER_MIN:
        return _format_seconds(nanos)
    if nanos < _NANOS_PER_HOUR:
        minutes = nanos // _NANOS_PER_MIN
        remainder = nanos % _NANOS_PER_MIN
        if remainder == 0:
            return str(minutes) + "m"
        return str(minutes) + "m" + _format_seconds(remainder)
    hours = nanos // _NANOS_PER_HOUR
    remainder = nanos % _NANOS_PER_HOUR
    if remainder == 0:
        return str(hours) + "h"
    text = str(hours) + "h"
    if remainder < _NANOS_PER_MIN:
        return text + _format_seconds(remainder)
    minutes = remainder // _NANOS_PER_MIN
    seconds = remainder % _NANOS_PER_MIN
    if seconds == 0:
        return text + str(minutes) + "m"
    return text + str(minutes) + "m" + _format_seconds(seconds)


def _format_seconds(nanos):
    """Render a positive sub-minute duration (0 < nanos < 1m)."""
    if nanos < _NANOS_PER_SEC:
        return _format_trimmed(float(nanos) / float(_NANOS_PER_MILLI), 1) + "ms"
    return _format_trimmed(float(nanos) / float(_NANOS_PER_SEC), 1) + "s"


def format_percent(f, digits):
    """Render a fraction as a percentage with the requested decimals:
    format_percent(0.125, 1) == "12.5%". A negative digits falls back to 0.
    Non-finite fractions keep Go's spelling ("NaN%", "+Inf%")."""
    if digits < 0:
        digits = 0
    return _go_format_float(f * 100, digits) + "%"


def format_float(f, digits, width):
    """Render f with digits decimals and right-align it in width display
    cells (pad_left). width <= 0 leaves the text unpadded; digits < 0 falls
    back to 0."""
    if digits < 0:
        digits = 0
    return pad_left(_go_format_float(f, digits), width)


def scale_value(v):
    """Divide v down to a base-1000 magnitude and return (scaled, suffix):
    999 -> (999, ""), 1500 -> (1.5, "K"), 2_000_000 -> (2, "M"). Values below
    1000 are returned unchanged. NaN/Inf are returned as-is with no suffix."""
    if math.isnan(v) or math.isinf(v):
        return v, ""
    scaled = v
    index = 0
    magnitude = math.fabs(v)
    while magnitude >= 1000 and index < len(SCALE_SUFFIXES) - 1:
        magnitude /= 1000
        scaled /= 1000
        index += 1
    return scaled, SCALE_SUFFIXES[index]


def pad_left(s, width):
    """Right-align s in width cells, padding with ASCII spaces. s is never
    truncated; a width <= its rune count is returned unchanged."""
    pad = width - len(s)
    if pad > 0:
        return " " * pad + s
    return s


def pad_right(s, width):
    """Left-align s in width cells, padding with ASCII spaces. s is never
    truncated; a width <= its rune count is returned unchanged."""
    pad = width - len(s)
    if pad > 0:
        return s + " " * pad
    return s

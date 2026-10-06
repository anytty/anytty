'use strict';

// Numeric/text formatters: pure, deterministic cell strings with no ANSI and
// no CJK width logic. Non-finite inputs keep Go's canonical spellings (NaN,
// +Inf, -Inf) so a bad sample is visible instead of silently zeroed.
// Port of tui2/sdk/widgets/format.go.

const SCALE_SUFFIXES = ['', 'K', 'M', 'G', 'T', 'P', 'E'];
const BYTE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB'];

const NANOS_PER_MICRO = 1000;
const NANOS_PER_MILLI = 1000 * NANOS_PER_MICRO;
const NANOS_PER_SEC = 1000 * NANOS_PER_MILLI;
const NANOS_PER_MIN = 60 * NANOS_PER_SEC;
const NANOS_PER_HOUR = 60 * NANOS_PER_MIN;

// formatGoFloat mirrors strconv.FormatFloat(f, 'f', digits, 64), including the
// non-finite spellings JavaScript's toFixed does not produce.
function formatGoFloat(f, digits) {
  if (Number.isNaN(f)) return 'NaN';
  if (f === Infinity) return '+Inf';
  if (f === -Infinity) return '-Inf';
  return f.toFixed(digits);
}

// formatTrimmed formats value with digits decimals and removes a trailing
// fractional zero run ("1.0" -> "1", "1.50" -> "1.5").
function formatTrimmed(value, digits) {
  let out = formatGoFloat(value, digits);
  if (out.indexOf('.') >= 0) {
    out = out.replace(/0+$/, '');
    out = out.replace(/\.$/, '');
  }
  return out;
}

// formatBytes renders n as a base-1024 size with a unit suffix.
function formatBytes(n) {
  if (n === 0) return '0 B';
  const negative = n < 0;
  let value = Math.abs(n);
  let unit = 0;
  while (value >= 1024 && unit < BYTE_UNITS.length - 1) {
    value /= 1024;
    unit++;
  }
  const out = formatTrimmed(value, 1) + ' ' + BYTE_UNITS[unit];
  return negative ? '-' + out : out;
}

// formatCount renders n with "," thousands separators.
function formatCount(n) {
  if (n === 0) return '0';
  const negative = n < 0;
  const digits = String(Math.abs(n));
  let out = negative ? '-' : '';
  for (let i = 0; i < digits.length; i++) {
    if (i > 0 && (digits.length - i) % 3 === 0) out += ',';
    out += digits[i];
  }
  return out;
}

// formatSeconds renders a positive sub-minute duration (0 < nanos < 1m).
function formatSeconds(nanos) {
  if (nanos < NANOS_PER_SEC) {
    return formatTrimmed(nanos / NANOS_PER_MILLI, 1) + 'ms';
  }
  return formatTrimmed(nanos / NANOS_PER_SEC, 1) + 's';
}

function formatDurationNanos(nanos) {
  if (nanos < NANOS_PER_MICRO) return String(nanos) + 'ns';
  if (nanos < NANOS_PER_MILLI) {
    return String(Math.floor((nanos + NANOS_PER_MICRO / 2) / NANOS_PER_MICRO)) + 'µs';
  }
  if (nanos < NANOS_PER_SEC) {
    return formatTrimmed(nanos / NANOS_PER_MILLI, 1) + 'ms';
  }
  if (nanos < NANOS_PER_MIN) return formatSeconds(nanos);
  if (nanos < NANOS_PER_HOUR) {
    const minutes = Math.floor(nanos / NANOS_PER_MIN);
    const remainder = nanos % NANOS_PER_MIN;
    if (remainder === 0) return String(minutes) + 'm';
    return String(minutes) + 'm' + formatSeconds(remainder);
  }
  const hours = Math.floor(nanos / NANOS_PER_HOUR);
  const remainder = nanos % NANOS_PER_HOUR;
  if (remainder === 0) return String(hours) + 'h';
  const text = String(hours) + 'h';
  if (remainder < NANOS_PER_MIN) return text + formatSeconds(remainder);
  const minutes = Math.floor(remainder / NANOS_PER_MIN);
  const seconds = remainder % NANOS_PER_MIN;
  if (seconds === 0) return text + String(minutes) + 'm';
  return text + String(minutes) + 'm' + formatSeconds(seconds);
}

// formatDuration renders d (nanoseconds) in the largest readable unit.
function formatDuration(d) {
  if (d === 0) return '0s';
  const negative = d < 0;
  const out = formatDurationNanos(Math.abs(d));
  return negative ? '-' + out : out;
}

// formatPercent renders a fraction as a percentage: formatPercent(0.125, 1)
// is "12.5%". A negative digits falls back to 0.
function formatPercent(f, digits) {
  if (digits < 0) digits = 0;
  return formatGoFloat(f * 100, digits) + '%';
}

// formatFloat renders f with digits decimals, right-aligned in width cells.
function formatFloat(f, digits, width) {
  if (digits < 0) digits = 0;
  return padLeft(formatGoFloat(f, digits), width);
}

// scaleValue divides v down to a base-1000 magnitude; 1500 -> (1.5, "K").
// NaN/Inf are returned as-is with an empty suffix.
function scaleValue(v) {
  if (Number.isNaN(v) || !Number.isFinite(v)) return [v, ''];
  let scaled = v;
  let index = 0;
  let abs = Math.abs(v);
  while (abs >= 1000 && index < SCALE_SUFFIXES.length - 1) {
    abs /= 1000;
    scaled /= 1000;
    index++;
  }
  return [scaled, SCALE_SUFFIXES[index]];
}

// padLeft right-aligns s in width cells; s is never truncated.
function padLeft(s, width) {
  const pad = width - Array.from(s).length;
  return pad > 0 ? ' '.repeat(pad) + s : s;
}

// padRight left-aligns s in width cells; s is never truncated.
function padRight(s, width) {
  const pad = width - Array.from(s).length;
  return pad > 0 ? s + ' '.repeat(pad) : s;
}

module.exports = {
  formatBytes,
  formatCount,
  formatDuration,
  formatPercent,
  formatFloat,
  scaleValue,
  padLeft,
  padRight,
};

"""Pure text-chart widgets (port of chart.go): Sparkline, BarChart, Heatmap,
Meter (and its Gauge alias) and Legend.

Like every widget here they hold no state, schedule no timer and emit no
protocol method: ``build`` returns a plain SDK box tree the caller commits.
All geometry is cell-based and every glyph is display width 1.
"""

import math

from .. import builder
from .format import format_float
from .list import first_non_empty

# Shared eighth-block ramp, low to high. Index 7 is the full block, used for
# whole cells; indices 0..6 are the partial cells.
_SPARK_GLYPHS = ["\u2581", "\u2582", "\u2583", "\u2584", "\u2585", "\u2586",
                 "\u2587", "\u2588"]

# Horizontal bar length when BarChart.width is unset.
_DEFAULT_BAR_WIDTH = 20

# ProgressBar glyphs reused by Meter (the visual half owns the canonical
# progress widget; these mirror its defaults).
_PROGRESS_FULL = "\u2588"
_PROGRESS_EMPTY = "\u2591"
_PROGRESS_WIDTH = 20

# Default low-to-high heat ramp: blank, middle dot, then light/medium/dark/
# full blocks.
DEFAULT_HEAT_SHADES = " \u00b7\u2591\u2592\u2593\u2588"

# Marker glyph used when LegendItem.marker is empty.
DEFAULT_LEGEND_MARKER = "\u25a0"


def _clamp_int(v, lo, hi):
    if v < lo:
        return lo
    if v > hi:
        return hi
    return v


def _clamp_float(v, lo, hi):
    if v < lo:
        return lo
    if v > hi:
        return hi
    return v


def _pad_to(text, width):
    text = builder.truncate(text, width)
    pad = width - builder.display_width(text)
    if pad > 0:
        text += " " * pad
    return text


class Sparkline:
    """Renders a series as one row of eighth-block glyphs. ``min``/``max``
    are optional bounds; when ``None`` they are derived from ``values``.
    Empty input renders an empty line; an all-equal (or single-value) series
    renders the mid-ramp glyph."""

    def __init__(self, values=None, width=0, style="", min=None, max=None):
        self.values = list(values or [])
        self.width = width
        self.style = style
        self.min = min
        self.max = max

    def bounds(self):
        """The effective low/high values, honouring explicit ``min``/``max``
        and deriving the rest from ``values``. Non-finite samples are
        ignored; an empty or all-NaN series yields (0, 0)."""
        lo, hi = math.inf, -math.inf
        for v in self.values:
            if math.isnan(v) or math.isinf(v):
                continue
            if v < lo:
                lo = v
            if v > hi:
                hi = v
        if math.isinf(lo) and lo > 0:
            lo, hi = 0, 0
        if self.min is not None:
            lo = self.min
        if self.max is not None:
            hi = self.max
        return lo, hi

    def level(self, v):
        """The eighth-block index [0, 7] for v under the effective bounds.
        ``min == max`` (including all-equal and one-value input) maps every
        sample to the mid-ramp glyph; NaN and out-of-range samples clamp."""
        lo, hi = self.bounds()
        if hi <= lo:
            return (len(_SPARK_GLYPHS) - 1) // 2
        if math.isnan(v):
            return 0
        t = _clamp_float((v - lo) / (hi - lo), 0, 1)
        level = int(t * float(len(_SPARK_GLYPHS) - 1) + 0.5)
        return _clamp_int(level, 0, len(_SPARK_GLYPHS) - 1)

    def samples(self):
        """The values to render: all of them, or ``width`` buckets of averaged
        samples when width is smaller than the series."""
        if self.width <= 0 or len(self.values) <= self.width:
            return self.values
        out = []
        n = len(self.values)
        for i in range(self.width):
            start = i * n // self.width
            end = (i + 1) * n // self.width
            if end <= start:
                end = start + 1
            total = 0.0
            for v in self.values[start:end]:
                total += v
            out.append(total / float(end - start))
        return out

    def line(self):
        """The sparkline as plain text, or "" for empty input."""
        samples = self.samples()
        if not samples:
            return ""
        return "".join(_SPARK_GLYPHS[self.level(v)] for v in samples)

    def build(self):
        """The sparkline as a one-row text box."""
        box = builder.text(self.line()).height(1)
        if self.style:
            box.style(self.style)
        return box


class BarChart:
    """Renders values as vertical sub-cell columns (eighth blocks) or as
    horizontal full-block runs. ``max`` <= 0 auto-scales to the largest value.
    Vertical bars occupy ``height`` rows (default 1) and share ``width``
    cells; when width cannot fit every bar, trailing bars are dropped.
    Horizontal bars use each value's row, optionally prefixed by its
    ``labels`` entry; ``height`` caps the number of rows. ``selected`` >= 0
    styles that bar with ``selected_style``."""

    def __init__(self, values=None, labels=None, width=0, height=0, max=0.0,
                 style="", label_style="", selected_style="", selected=0,
                 horizontal=False):
        self.values = list(values or [])
        self.labels = list(labels or [])
        self.width = width
        self.height = height
        self.max = max
        self.style = style
        self.label_style = label_style
        self.selected_style = selected_style
        self.selected = selected
        self.horizontal = horizontal

    def max_value(self):
        """``max`` when positive, else the largest finite value (at least 1
        when the series is empty or non-positive)."""
        if self.max > 0:
            return self.max
        maximum = 0.0
        for v in self.values:
            if math.isnan(v) or math.isinf(v):
                continue
            if v > maximum:
                maximum = v
        if maximum <= 0:
            return 1
        return maximum

    def bar_height(self):
        if self.height > 0:
            return self.height
        return 1

    def vertical_values(self):
        """The values a vertical chart can fit in ``width`` cells. Bars are
        one cell wide with a single gap, so a width of w fits (w+1)/2
        bars."""
        values = self.values
        if self.width <= 0:
            return values
        max_bars = (self.width + 1) // 2
        if max_bars > len(values):
            max_bars = len(values)
        if max_bars < 0:
            max_bars = 0
        return values[:max_bars]

    def horizontal_values(self):
        """The values a horizontal chart renders, capped by height when
        set."""
        values = self.values
        if self.height > 0 and len(values) > self.height:
            return values[:self.height]
        return values

    def column_width(self, count):
        """The cell width of each vertical bar: width split evenly, always at
        least 1."""
        if count <= 0:
            return 1
        if self.width <= 0:
            return 1
        width = (self.width - (count - 1)) // count
        if width < 1:
            return 1
        return width

    def label_width(self):
        """The widest label in display cells, or 0 without labels."""
        width = 0
        for label in self.labels:
            w = builder.display_width(label)
            if w > width:
                width = w
        return width

    def horizontal_bar_width(self, label_width):
        """The run length for a horizontal bar, leaving room for the labels
        and the gap."""
        if self.width <= 0:
            return _DEFAULT_BAR_WIDTH
        width = self.width
        if label_width > 0:
            width -= label_width + 1
        if width < 1:
            return 1
        return width

    def bar_style(self, index):
        if index == self.selected and self.selected_style:
            return self.selected_style
        return self.style

    def bar_units(self, v, maximum, height):
        if height <= 0 or maximum <= 0 or math.isnan(v) or v <= 0:
            return 0
        t = _clamp_float(v / maximum, 0, 1)
        units = int(t * float(height * 8) + 0.5)
        return _clamp_int(units, 0, height * 8)

    def vertical_cells(self):
        """The bar grid as [row][bar] cells, top row first."""
        values = self.vertical_values()
        count = len(values)
        if count == 0:
            return None
        height = self.bar_height()
        width = self.column_width(count)
        maximum = self.max_value()
        units = [self.bar_units(v, maximum, height) for v in values]
        cells = []
        for row in range(height):
            k = height - 1 - row
            cells.append([_bar_cell(units[i], k, width) for i in range(count)])
        return cells

    def vertical_lines(self):
        """The vertical chart as plain text rows, including the label row when
        labels is set. Returns ``None`` for empty input."""
        cells = self.vertical_cells()
        if cells is None:
            return None
        width = self.column_width(len(cells[0]))
        lines = [" ".join(row) for row in cells]
        if self.labels:
            count = len(cells[0])
            labels = []
            for i in range(count):
                if i < len(self.labels):
                    labels.append(_pad_to(self.labels[i], width))
                else:
                    labels.append(" " * width)
            lines.append(" ".join(labels))
        return lines

    def horizontal_lines(self):
        """The horizontal chart as plain text rows."""
        values = self.horizontal_values()
        if not values:
            return None
        label_width = self.label_width()
        bar_width = self.horizontal_bar_width(label_width)
        maximum = self.max_value()
        lines = []
        for i, v in enumerate(values):
            prefix = ""
            if label_width > 0:
                label = ""
                if i < len(self.labels):
                    label = self.labels[i]
                prefix = _pad_to(label, label_width) + " "
            lines.append(prefix + "\u2588" * self.run_length(v, maximum, bar_width))
        return lines

    def run_length(self, v, maximum, width):
        if math.isnan(v) or v <= 0 or maximum <= 0:
            return 0
        length = int(_clamp_float(v / maximum, 0, 1) * float(width) + 0.5)
        return _clamp_int(length, 0, width)

    def build(self):
        """The chart as a column of styled rows."""
        if self.horizontal:
            return self._build_horizontal()
        return self._build_vertical()

    def _build_vertical(self):
        col = builder.box("col")
        values = self.vertical_values()
        if not values:
            return col
        height = self.bar_height()
        width = self.column_width(len(values))
        maximum = self.max_value()
        units = [self.bar_units(v, maximum, height) for v in values]
        for row in range(height):
            k = height - 1 - row
            line = builder.row().height(1)
            for i in range(len(values)):
                if i > 0:
                    line.child(builder.text(" "))
                line.child(builder.text(_bar_cell(units[i], k, width))
                           .style(self.bar_style(i)))
            col.child(line)
        if self.labels:
            line = builder.row().height(1)
            for i in range(len(values)):
                if i > 0:
                    line.child(builder.text(" "))
                label = ""
                if i < len(self.labels):
                    label = self.labels[i]
                line.child(builder.text(_pad_to(label, width))
                           .style(self.label_style))
            col.child(line)
        return col

    def _build_horizontal(self):
        col = builder.box("col")
        values = self.horizontal_values()
        if not values:
            return col
        label_width = self.label_width()
        bar_width = self.horizontal_bar_width(label_width)
        maximum = self.max_value()
        for i, v in enumerate(values):
            line = builder.row().height(1)
            if label_width > 0:
                label = ""
                if i < len(self.labels):
                    label = self.labels[i]
                line.child(builder.text(_pad_to(label, label_width) + " ")
                           .style(self.label_style))
            length = self.run_length(v, maximum, bar_width)
            if length > 0:
                line.child(builder.text("\u2588" * length)
                           .style(self.bar_style(i)))
            col.child(line)
        return col


def _bar_cell(units, k, width):
    """One vertical bar cell for the k-th row counted from the bottom: a full
    run, a single partial eighth glyph, or blanks."""
    if width < 1:
        width = 1
    if units >= (k + 1) * 8:
        return "\u2588" * width
    if units > k * 8:
        level = units - k * 8
        return _SPARK_GLYPHS[level - 1] + " " * (width - 1)
    return " " * width


class Heatmap:
    """Renders a matrix as shaded cells. Values may be ragged: every row is
    padded to ``cols()`` and missing cells render as the lowest shade.
    ``min``/``max`` are the value range; when ``min >= max`` the range is
    derived from the data. ``shades`` overrides the default ramp (low to
    high); ``row_labels``/``col_labels`` are optional."""

    def __init__(self, values=None, row_labels=None, col_labels=None, min=0.0,
                 max=0.0, style="", label_style="", shades=None):
        self.values = [list(row) for row in (values or [])]
        self.row_labels = list(row_labels or [])
        self.col_labels = list(col_labels or [])
        self.min = min
        self.max = max
        self.style = style
        self.label_style = label_style
        self.shades = list(shades) if shades is not None else None

    def shade_list(self):
        """The active ramp, defaulting to ``DEFAULT_HEAT_SHADES``."""
        if self.shades:
            return self.shades
        return list(DEFAULT_HEAT_SHADES)

    def rows(self):
        """The number of value rows."""
        return len(self.values)

    def cols(self):
        """The matrix width: the widest row, raised to len(col_labels)."""
        cols = len(self.col_labels)
        for row in self.values:
            if len(row) > cols:
                cols = len(row)
        return cols

    def bounds(self):
        """The effective value range. An explicit ``min < max`` wins;
        otherwise the range is derived from the finite samples. All-equal
        (and empty) data collapses to a zero-width range."""
        if self.min < self.max:
            return self.min, self.max
        lo, hi = math.inf, -math.inf
        for row in self.values:
            for v in row:
                if math.isnan(v) or math.isinf(v):
                    continue
                if v < lo:
                    lo = v
                if v > hi:
                    hi = v
        if math.isinf(lo) and lo > 0:
            return 0, 0
        return lo, hi

    def level(self, v):
        """The shade index [0, len(shades)-1] for v. A zero-width range maps
        every sample to the mid-ramp; NaN clamps to the lowest shade."""
        shades = self.shade_list()
        count = len(shades)
        if count <= 0:
            return 0
        lo, hi = self.bounds()
        if hi <= lo:
            return count // 2
        if math.isnan(v):
            return 0
        t = _clamp_float((v - lo) / (hi - lo), 0, 1)
        return _clamp_int(int(t * float(count - 1) + 0.5), 0, count - 1)

    def cell(self, row, col):
        """The plain shade text of one matrix cell. Out-of-range columns and
        missing (short) rows return the lowest shade, so the grid is always
        rectangular."""
        shades = self.shade_list()
        if not shades:
            return ""
        if row < 0 or row >= len(self.values) or col < 0 or col >= len(self.values[row]):
            return shades[0]
        return shades[self.level(self.values[row][col])]

    def grid(self):
        """The heatmap as plain text rows: an optional column-label header,
        then one row per value row with its optional row label."""
        rows, cols = self.rows(), self.cols()
        if rows == 0 or cols == 0:
            return None
        shades = self.shade_list()
        if not shades:
            return None
        row_label_width = self.row_label_width()
        col_width = self.col_width()
        gutter = ""
        if row_label_width > 0:
            gutter = " "
        blank = " " * (row_label_width + len(gutter))
        lines = []
        if self.col_labels:
            parts = [blank]
            for c in range(cols):
                label = ""
                if c < len(self.col_labels):
                    label = self.col_labels[c]
                parts.append(_pad_to(label, col_width))
            lines.append("".join(parts))
        for r in range(rows):
            parts = []
            if row_label_width > 0:
                label = ""
                if r < len(self.row_labels):
                    label = self.row_labels[r]
                parts.append(_pad_to(label, row_label_width))
                parts.append(gutter)
            for c in range(cols):
                parts.append(_pad_to(self.cell(r, c), col_width))
            lines.append("".join(parts))
        return lines

    def row_label_width(self):
        width = 0
        for label in self.row_labels:
            w = builder.display_width(label)
            if w > width:
                width = w
        return width

    def col_width(self):
        width = 1
        for label in self.col_labels:
            w = builder.display_width(label)
            if w > width:
                width = w
        return width

    def build(self):
        """The heatmap as a column of styled one-row boxes."""
        col = builder.box("col")
        lines = self.grid()
        if not lines:
            return col
        col.child(builder.text(lines[0])
                  .style(first_non_empty(self.label_style, self.style)).height(1))
        for i in range(1, len(lines)):
            col.child(builder.text(lines[i]).style(self.style).height(1))
        return col


class Meter:
    """A labelled bar for a float value, reusing the ProgressBar glyphs and
    semantics without depending on its integer fields."""

    def __init__(self, id="", value=0.0, max=0.0, width=0, label="",
                 show_value=False, style="", track_style="", label_style="",
                 value_style=""):
        self.id = id
        self.value = value
        self.max = max
        self.width = width
        self.label = label
        self.show_value = show_value
        self.style = style
        self.track_style = track_style
        self.label_style = label_style
        self.value_style = value_style

    def fraction(self):
        """``value/max`` clamped to [0, 1]; a non-positive max (or NaN value)
        is 0."""
        if self.max <= 0 or math.isnan(self.value):
            return 0
        return _clamp_float(self.value / self.max, 0, 1)

    def bar_width(self):
        if self.width > 0:
            return self.width
        return _PROGRESS_WIDTH

    def bar_text(self):
        """The full bar string (filled + empty glyphs)."""
        width = self.bar_width()
        filled = _clamp_int(int(self.fraction() * float(width) + 0.5), 0, width)
        return _PROGRESS_FULL * filled + _PROGRESS_EMPTY * (width - filled)

    def value_text(self):
        """The optional value readout ("3/10"), or ""."""
        if not self.show_value:
            return ""
        text = format_float(self.value, 0, 0)
        if self.max > 0:
            text += "/" + format_float(self.max, 0, 0)
        return text

    def build(self):
        """The meter as one row: label, fill, track and value."""
        row = builder.row().height(1)
        if self.id:
            row.id(self.id)
        if self.label:
            row.child(builder.text(self.label).style(self.label_style).height(1))
        width = self.bar_width()
        filled = _clamp_int(int(self.fraction() * float(width) + 0.5), 0, width)
        if filled > 0:
            row.child(builder.text(_PROGRESS_FULL * filled)
                      .style(self.style).height(1))
        empty = width - filled
        if empty > 0:
            row.child(builder.text(_PROGRESS_EMPTY * empty)
                      .style(self.track_style).height(1))
        text = self.value_text()
        if text != "":
            row.child(builder.text(text).style(self.value_style).height(1))
        return row


# Gauge is the alias of Meter.
Gauge = Meter


class LegendItem:
    """One legend entry: a marker coloured with ``color``, followed by its
    ``label`` styled with the Legend style."""

    __slots__ = ("label", "color", "marker")

    def __init__(self, label="", color="", marker=""):
        self.label = label
        self.color = color
        self.marker = marker


class Legend:
    """A one-row key of coloured markers and labels."""

    def __init__(self, items=None, style="", separator=""):
        self.items = list(items or [])
        self.style = style
        self.separator = separator

    def text(self):
        """The legend as plain text for measurement."""
        separator = first_non_empty(self.separator, " ")
        parts = []
        for i, item in enumerate(self.items):
            if i > 0:
                parts.append(separator)
            parts.append(first_non_empty(item.marker, DEFAULT_LEGEND_MARKER))
            if item.label:
                parts.append(" " + item.label)
        return "".join(parts)

    def build(self):
        """The legend as a one-row box tree."""
        row = builder.row().height(1)
        separator = first_non_empty(self.separator, " ")
        for i, item in enumerate(self.items):
            if i > 0:
                row.child(builder.text(separator).style(self.style))
            marker = builder.text(first_non_empty(item.marker, DEFAULT_LEGEND_MARKER))
            if item.color:
                marker.style(item.color)
            row.child(marker)
            if item.label:
                row.child(builder.text(" " + item.label).style(self.style))
        return row

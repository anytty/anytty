"""Layout/chrome geometry and overlay widgets.

:class:`SplitLayout` lays panes along one axis with a fixed gap and hands back
one divider rect per gap; :class:`TitleBar` and :class:`Footer` are one-row
bar builders; :class:`Picker`, :class:`FloatingLayer` and :class:`Toast` are
the program-side overlay primitives. Everything is pure presentation: the
widgets never emit protocol methods.
"""

from .. import builder
from .basics import (
    Button,
    DEFAULT_SEP_STYLE,
    DEFAULT_SEPARATOR,
    Frame,
    FrameRow,
    Segment,
    _append_group,
    _fit_segments,
    _group_width,
    _segment_texts,
    _segments_text,
    _segments_width,
)


class Rect:
    """One solved rectangle in viewport cells."""

    def __init__(self, x=0, y=0, w=0, h=0):
        self.x = x
        self.y = y
        self.w = w
        self.h = h

    def _key(self):
        return (self.x, self.y, self.w, self.h)

    def __eq__(self, other):
        if not isinstance(other, Rect):
            return NotImplemented
        return self._key() == other._key()

    def __repr__(self):
        return "Rect(x=%d, y=%d, w=%d, h=%d)" % self._key()


def _trunc_div(numerator, denominator):
    if numerator < 0:
        return -((-numerator) // denominator)
    return numerator // denominator


def distribute(avail, ratios):
    """Split avail cells over integer ratios.

    Proportional, at least one cell each, remainder on the last pane. It is
    the geometry primitive behind :class:`SplitLayout`.
    """
    if not ratios:
        return []
    out = [0] * len(ratios)
    if avail < len(ratios):
        avail = len(ratios)
    total = sum(ratio for ratio in ratios if ratio > 0)
    if total <= 0:
        total = len(ratios)
        out = [1] * len(out)
    else:
        for index, ratio in enumerate(ratios):
            value = _trunc_div(avail * ratio, total)
            out[index] = value if value >= 1 else 1
    used = sum(out)
    out[-1] += avail - used
    if out[-1] < 1:
        out[-1] = 1
    return out


class SplitLayout:
    """Panes along one axis with a fixed gap.

    Each visible gap is a draggable Divider cell the program hit-tests like
    any other box.
    """

    def __init__(self, orient="", weights=None, gap=0):
        self.orient = orient
        self.weights = list(weights or [])
        self.gap = gap

    def axis(self):
        """Return the resolved orientation ("row" unless "col" is explicit)."""
        if self.orient == "col":
            return "col"
        return "row"

    def gap_width(self):
        """Return the separator cells between two panes."""
        if self.gap > 0:
            return self.gap
        return 1

    def rects(self, width, height):
        """Solve the layout inside a width x height box.

        Returns ``(panes, dividers)``: one rect per pane and one divider rect
        per gap. Pane count comes from ``weights`` (or one pane).
        """
        weights = self.weights or [1]
        count = len(weights)
        gap = self.gap_width()
        panes = []
        dividers = []
        if self.axis() == "col":
            avail = height - (count - 1) * gap
            sizes = distribute(avail, weights)
            y = 0
            for index, size in enumerate(sizes):
                panes.append(Rect(0, y, width, size))
                y += size
                if index < count - 1:
                    dividers.append(Rect(0, y, width, gap))
                    y += gap
            return panes, dividers
        avail = width - (count - 1) * gap
        sizes = distribute(avail, weights)
        x = 0
        for index, size in enumerate(sizes):
            panes.append(Rect(x, 0, size, height))
            x += size
            if index < count - 1:
                dividers.append(Rect(x, 0, gap, height))
                x += gap
        return panes, dividers


class TitleBar:
    """A one-row title strip: left segments plus right-aligned buttons.

    With ``width > 0`` the buttons are pushed to the right edge and the left
    group is truncated.
    """

    def __init__(self, id="", left=None, buttons=None, width=0, fill=""):
        self.id = id
        self.left = list(left or [])
        self.buttons = list(buttons or [])
        self.width = width
        self.fill = fill

    def line(self):
        """Render the bar as plain text for measurement."""
        left = _segments_text(self.left)
        right = " ".join(button.line() for button in self.buttons)
        if self.width > 0:
            pad = self.width - builder.display_width(left) - builder.display_width(right)
            if pad > 0:
                return left + " " * pad + right
        return left + right

    def build(self):
        """Return the bar as a one-row box tree."""
        row = builder.row().height(1)
        if self.id:
            row.id(self.id)
        left = list(self.left)
        right = " ".join(button.line() for button in self.buttons)
        if self.width > 0 and builder.display_width(right) < self.width:
            left, _ = _fit_segments(
                left, self.width - builder.display_width(right),
                builder.display_width(DEFAULT_SEPARATOR))
        _append_group(row, left, self.fill, DEFAULT_SEPARATOR)
        if self.width > 0:
            pad = self.width - _segments_width(left) - builder.display_width(right)
            if pad > 0:
                row.child(builder.text(" " * pad))
        for index, button in enumerate(self.buttons):
            if index > 0:
                row.child(builder.text(" "))
            row.child(button.build())
        return row


class Footer:
    """The bottom bar: optional scene badge plus key groups.

    ``width > 0`` right-aligns the summary and truncates the left groups.
    """

    def __init__(self, id="", badge=None, has_badge=False, groups=None,
                 right=None, width=0, separator="", sep_style=""):
        self.id = id
        self.badge = badge if badge is not None else Segment()
        self.has_badge = has_badge
        self.groups = list(groups or [])
        self.right = list(right or [])
        self.width = width
        self.separator = separator
        self.sep_style = sep_style

    def separator_text(self):
        if self.separator:
            return " " + self.separator + " "
        return " " + DEFAULT_SEPARATOR + " "

    def _sep_style(self):
        return self.sep_style or DEFAULT_SEP_STYLE

    def left_segments(self):
        left = []
        if self.has_badge:
            left.append(self.badge)
        left.extend(self.groups)
        return left

    def line(self):
        """Render the whole bar as plain text at ``width`` cells."""
        sep = self.separator_text()
        left, right = self.left_segments(), list(self.right)
        if self.width > 0:
            sep_width = builder.display_width(sep)
            right, _ = _fit_segments(right, self.width, sep_width)
            budget = self.width - _group_width(right, sep_width)
            if left and right:
                budget -= sep_width
            left, _ = _fit_segments(left, budget, sep_width)
        left_text = sep.join(_segment_texts(left))
        right_text = sep.join(_segment_texts(right))
        if self.width <= 0:
            if left_text and right_text:
                return left_text + sep + right_text
            return left_text + right_text
        pad = self.width - builder.display_width(left_text) - builder.display_width(right_text)
        if pad < 0:
            pad = 0
        return left_text + " " * pad + right_text

    def build(self):
        """Return the footer as a one-row box tree."""
        row = builder.row().height(1)
        if self.id:
            row.id(self.id)
        sep = self.separator_text()
        sep_width = builder.display_width(sep)
        left = self.left_segments()
        right = list(self.right)
        if self.width > 0:
            right, _ = _fit_segments(right, self.width, sep_width)
            left_budget = self.width - _group_width(right, sep_width)
            if left and right:
                left_budget -= sep_width
            left, _ = _fit_segments(left, left_budget, sep_width)
        used = _group_width(left, sep_width) + _group_width(right, sep_width)
        if left and right:
            used += sep_width
        _append_group(row, left, self._sep_style(), sep)
        if left and right:
            row.child(builder.text(sep).style(self._sep_style()))
        if self.width > used:
            row.child(builder.text(" " * (self.width - used)))
        _append_group(row, right, self._sep_style(), sep)
        return row


class PickerRow:
    """One selectable row of a Picker."""

    def __init__(self, text="", id="", style="", selected=False,
                 selectable=False):
        self.text = text
        self.id = id
        self.style = style
        self.selected = selected
        self.selectable = selectable


class Picker:
    """A framed selectable list (the program-side picker overlay).

    Row ids are hit-test targets; the program moves the selection itself.
    """

    def __init__(self, id="", title="", width=0, height=0, rows=None,
                 style="", marker="", selected_style=""):
        self.id = id
        self.title = title
        self.width = width
        self.height = height
        self.rows = list(rows or [])
        self.style = style
        self.marker = marker
        self.selected_style = selected_style

    def build(self):
        """Return the picker as a framed box tree."""
        marker = self.marker or "▸ "
        rows = []
        for row in self.rows:
            text = "  " + row.text
            style = row.style
            if row.selected:
                text = marker + row.text
                if self.selected_style:
                    style = self.selected_style
            frame_row = FrameRow(text=text, style=style)
            if row.id:
                frame_row.id = row.id
                frame_row.input = ["mouse"]
            rows.append(frame_row)
        width = self.width if self.width > 0 else 40
        height = self.height if self.height > 0 else len(rows) + 2
        return Frame(id=self.id, title=self.title, width=width, height=height,
                     style=self.style, rows=rows).build()


class FloatingLayer:
    """A floating window: a Frame placed with ``pos`` above the regular flow.

    Collapsed keeps only the title row.
    """

    def __init__(self, id="", title="", x=0, y=0, width=0, height=0, style="",
                 rows=None, collapsed=False):
        self.id = id
        self.title = title
        self.x = x
        self.y = y
        self.width = width
        self.height = height
        self.style = style
        self.rows = list(rows or [])
        self.collapsed = collapsed

    def build(self):
        """Return the floating window as a positioned box tree."""
        height = 1 if self.collapsed else self.height
        frame = Frame(id=self.id, title=self.title, width=self.width,
                      height=height, style=self.style, rows=self.rows)
        return frame.build().pos(self.x, self.y)


class Toast:
    """A transient one-line notice (program-side), dropped to hide."""

    def __init__(self, id="", text="", style=""):
        self.id = id
        self.text = text
        self.style = style

    def build(self):
        """Return the toast as a text box."""
        box = builder.text(self.text)
        if self.id:
            box.id(self.id)
        if self.style:
            box.style(self.style)
        return box

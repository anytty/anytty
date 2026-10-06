"""Windowed one-line-per-item lists (port of list.go).

``List`` and ``VirtualList`` render only the rows inside ``visible_range()``,
so a 100k-item list costs O(height). ``height`` counts the whole widget:
``header`` and ``footer`` each consume one row, the rest is window; a height
<= 0 falls back to every row. The caller owns ``offset``/``selected`` and
drives them through the helpers; with ``follow`` the selected row is kept in
the window without mutating the declared offset.
"""

from .. import builder
from .mouse import apply_wheel

# The selected-row gutter of List and VirtualList.
DEFAULT_LIST_MARKER = "\u25b8 "


class ListRow:
    """One row of a VirtualList: text, an optional hit-test id, an optional
    explicit style and a disabled flag (rendered with ``disabled_style``)."""

    __slots__ = ("text", "id", "style", "disabled")

    def __init__(self, text="", id="", style="", disabled=False):
        self.text = text
        self.id = id
        self.style = style
        self.disabled = disabled


def first_non_empty(*values):
    """The first non-empty value, or ""."""
    for value in values:
        if value:
            return value
    return ""


def chrome_rows(header, footer):
    """Count the fixed rows a header and a footer occupy."""
    rows = 0
    if header:
        rows += 1
    if footer:
        rows += 1
    return rows


def window_size(height, chrome, count):
    """Visible window rows for a declared height. A height <= 0 means "all
    rows minus chrome"; it never returns a negative size."""
    size = height
    if size <= 0:
        size = count
    size -= chrome
    if size < 0:
        size = 0
    if size > count:
        size = count
    return size


def clamp_index(i, count):
    """Fold i into [0, count-1]; an empty set selects 0."""
    if i < 0 or count <= 0:
        return 0
    if i >= count:
        return count - 1
    return i


def clamp_offset(offset, size, count):
    """Fold offset into [0, count-size]."""
    if size <= 0:
        return 0
    if offset < 0:
        return 0
    maximum = count - size
    if offset > maximum:
        return maximum
    return offset


def follow_offset(offset, selected, size):
    """Scroll offset the minimum amount that puts selected inside a size-row
    window."""
    if size <= 0:
        return 0
    if selected < offset:
        offset = selected
    if selected >= offset + size:
        offset = selected - size + 1
    if offset < 0:
        return 0
    return offset


def move_window(offset, selected, delta, size, count, follow):
    """Apply a selection delta and follow it when requested."""
    selected = clamp_index(selected + delta, count)
    if follow:
        offset = follow_offset(offset, selected, size)
    return clamp_offset(offset, size, count), selected


def _list_box(list_id, width, height):
    col = builder.box("col")
    if list_id:
        col.id(list_id)
    if width > 0:
        col.width(width)
    if height > 0:
        col.height(height)
    return col


def _pad_to(text, width):
    text = builder.truncate(text, width)
    pad = width - builder.display_width(text)
    if pad > 0:
        text += " " * pad
    return text


def list_row_box(text, style, row_id, width):
    """One list row box: padded text, optional style and mouse hit id."""
    if width > 0:
        text = _pad_to(text, width)
    box = builder.text(text).height(1)
    if width > 0:
        box.width(width)
    if style:
        box.style(style)
    if row_id:
        box.id(row_id).input("mouse")
    return box


class List:
    """A windowed one-line-per-item list."""

    def __init__(self, id="", items=None, width=0, height=0, offset=0,
                 selected=0, follow=False, header="", footer="", empty="",
                 marker="", style="", selected_style="", header_style="",
                 footer_style="", empty_style="", row_id=None, row_style=None):
        self.id = id
        self.items = list(items or [])
        self.width = width
        self.height = height
        self.offset = offset
        self.selected = selected
        self.follow = follow
        self.header = header
        self.footer = footer
        self.empty = empty
        self.marker = marker
        self.style = style
        self.selected_style = selected_style
        self.header_style = header_style
        self.footer_style = footer_style
        self.empty_style = empty_style
        self.row_id = row_id
        self.row_style = row_style

    def visible_range(self):
        """The half-open row window [start, end) ``build()`` renders."""
        size = window_size(self.height,
                           chrome_rows(self.header, self.footer),
                           len(self.items))
        if size <= 0:
            return 0, 0
        offset = self.offset
        if self.follow:
            offset = follow_offset(offset, clamp_index(self.selected, len(self.items)), size)
        offset = clamp_offset(offset, size, len(self.items))
        return offset, offset + size

    def ensure_visible(self):
        """Scroll the window so ``selected`` is visible."""
        size = window_size(self.height,
                           chrome_rows(self.header, self.footer),
                           len(self.items))
        if size <= 0 or not self.items:
            return
        self.selected = clamp_index(self.selected, len(self.items))
        self.offset = follow_offset(self.offset, self.selected, size)
        self.offset = clamp_offset(self.offset, size, len(self.items))

    def move(self, delta):
        """Shift the selection by delta and follow it when ``follow`` is set."""
        self.offset, self.selected = move_window(
            self.offset, self.selected, delta,
            window_size(self.height, chrome_rows(self.header, self.footer),
                        len(self.items)),
            len(self.items), self.follow)

    def page_up(self):
        """Move the selection one window up."""
        self.move(-window_size(self.height,
                               chrome_rows(self.header, self.footer),
                               len(self.items)))

    def page_down(self):
        """Move the selection one window down."""
        self.move(window_size(self.height,
                              chrome_rows(self.header, self.footer),
                              len(self.items)))

    def top(self):
        """Select the first row and scroll to the start."""
        self.selected = 0
        self.offset = 0

    def bottom(self):
        """Select the last row and scroll to the end."""
        if not self.items:
            self.selected, self.offset = 0, 0
            return
        self.selected = len(self.items) - 1
        size = window_size(self.height,
                           chrome_rows(self.header, self.footer),
                           len(self.items))
        if size > 0:
            self.offset = clamp_offset(len(self.items) - size, size,
                                       len(self.items))

    def row_at(self, y):
        """The absolute List row at viewport y, accounting for the header row
        and the current window. Returns ``(row, True)`` or ``(0, False)``."""
        top = 0
        if self.header:
            top += 1
        size = window_size(self.height,
                           chrome_rows(self.header, self.footer),
                           len(self.items))
        if size <= 0 or y < top or y >= top + size:
            return 0, False
        start, _ = self.visible_range()
        row = start + (y - top)
        if row < 0 or row >= len(self.items):
            return 0, False
        return row, True

    def on_wheel(self, ev):
        """Apply one wheel event (``{"delta": n}``) to the list offset,
        leaving the selection untouched. ``None`` is a no-op."""
        if ev is None:
            return
        delta = int(ev.get("delta") or 0)
        self.offset = apply_wheel(
            self.offset, len(self.items),
            window_size(self.height, chrome_rows(self.header, self.footer),
                        len(self.items)),
            delta)

    def build(self):
        """The list as a column of text rows."""
        col = _list_box(self.id, self.width, self.height)
        if self.header:
            col.child(list_row_box(self.header, self.header_style, "", self.width))
        start, end = self.visible_range()
        if end > start:
            for i in range(start, end):
                col.child(self._row(i))
        elif not self.items and self.empty:
            col.child(list_row_box(self.empty, self.empty_style, "", self.width))
        if self.footer:
            col.child(list_row_box(self.footer, self.footer_style, "", self.width))
        return col

    def _row(self, index):
        text = "  " + self.items[index]
        style = self.style
        if index == self.selected:
            text = first_non_empty(self.marker, DEFAULT_LIST_MARKER) + self.items[index]
            if self.selected_style:
                style = self.selected_style
        if self.row_style is not None:
            row_style = self.row_style(index, index == self.selected)
            if row_style:
                style = row_style
        row_id = ""
        if self.row_id is not None:
            row_id = self.row_id(index)
        return list_row_box(text, style, row_id, self.width)


class VirtualList:
    """List with rich :class:`ListRow` data. The same windowing rules apply."""

    def __init__(self, id="", rows=None, width=0, height=0, offset=0,
                 selected=0, follow=False, header="", footer="", empty="",
                 marker="", style="", selected_style="", disabled_style="",
                 header_style="", footer_style="", empty_style=""):
        self.id = id
        self.rows = list(rows or [])
        self.width = width
        self.height = height
        self.offset = offset
        self.selected = selected
        self.follow = follow
        self.header = header
        self.footer = footer
        self.empty = empty
        self.marker = marker
        self.style = style
        self.selected_style = selected_style
        self.disabled_style = disabled_style
        self.header_style = header_style
        self.footer_style = footer_style
        self.empty_style = empty_style

    def visible_range(self):
        """The half-open row window [start, end) ``build()`` renders."""
        size = window_size(self.height,
                           chrome_rows(self.header, self.footer),
                           len(self.rows))
        if size <= 0:
            return 0, 0
        offset = self.offset
        if self.follow:
            offset = follow_offset(offset, clamp_index(self.selected, len(self.rows)), size)
        offset = clamp_offset(offset, size, len(self.rows))
        return offset, offset + size

    def ensure_visible(self):
        """Scroll the window so ``selected`` is visible."""
        size = window_size(self.height,
                           chrome_rows(self.header, self.footer),
                           len(self.rows))
        if size <= 0 or not self.rows:
            return
        self.selected = clamp_index(self.selected, len(self.rows))
        self.offset = follow_offset(self.offset, self.selected, size)
        self.offset = clamp_offset(self.offset, size, len(self.rows))

    def move(self, delta):
        """Shift the selection by delta and follow it when ``follow`` is set."""
        self.offset, self.selected = move_window(
            self.offset, self.selected, delta,
            window_size(self.height, chrome_rows(self.header, self.footer),
                        len(self.rows)),
            len(self.rows), self.follow)

    def page_up(self):
        """Move the selection one window up."""
        self.move(-window_size(self.height,
                               chrome_rows(self.header, self.footer),
                               len(self.rows)))

    def page_down(self):
        """Move the selection one window down."""
        self.move(window_size(self.height,
                              chrome_rows(self.header, self.footer),
                              len(self.rows)))

    def top(self):
        """Select the first row and scroll to the start."""
        self.selected = 0
        self.offset = 0

    def bottom(self):
        """Select the last row and scroll to the end."""
        if not self.rows:
            self.selected, self.offset = 0, 0
            return
        self.selected = len(self.rows) - 1
        size = window_size(self.height,
                           chrome_rows(self.header, self.footer),
                           len(self.rows))
        if size > 0:
            self.offset = clamp_offset(len(self.rows) - size, size,
                                       len(self.rows))

    def row_at(self, y):
        """The absolute VirtualList row at viewport y."""
        top = 0
        if self.header:
            top += 1
        size = window_size(self.height,
                           chrome_rows(self.header, self.footer),
                           len(self.rows))
        if size <= 0 or y < top or y >= top + size:
            return 0, False
        start, _ = self.visible_range()
        row = start + (y - top)
        if row < 0 or row >= len(self.rows):
            return 0, False
        return row, True

    def on_wheel(self, ev):
        """Apply one wheel event (``{"delta": n}``) to the list offset."""
        if ev is None:
            return
        delta = int(ev.get("delta") or 0)
        self.offset = apply_wheel(
            self.offset, len(self.rows),
            window_size(self.height, chrome_rows(self.header, self.footer),
                        len(self.rows)),
            delta)

    def build(self):
        """The list as a column of text rows."""
        col = _list_box(self.id, self.width, self.height)
        if self.header:
            col.child(list_row_box(self.header, self.header_style, "", self.width))
        start, end = self.visible_range()
        if end > start:
            for i in range(start, end):
                col.child(self._row(i))
        elif not self.rows and self.empty:
            col.child(list_row_box(self.empty, self.empty_style, "", self.width))
        if self.footer:
            col.child(list_row_box(self.footer, self.footer_style, "", self.width))
        return col

    def _row(self, index):
        item = self.rows[index]
        text = "  " + item.text
        style = first_non_empty(item.style, self.style)
        if item.disabled:
            style = first_non_empty(self.disabled_style, style)
        if index == self.selected:
            text = first_non_empty(self.marker, DEFAULT_LIST_MARKER) + item.text
            style = first_non_empty(self.selected_style, style)
        return list_row_box(text, style, item.id, self.width)

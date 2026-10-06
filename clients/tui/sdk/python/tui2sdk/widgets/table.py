"""Aligned, truncated table rendering (port of table.go)."""

from .. import builder
from .list import first_non_empty

# Cell alignment values of Column.align.
ALIGN_LEFT = "left"
ALIGN_RIGHT = "right"
ALIGN_CENTER = "center"

# The one-cell gutter between Table columns.
DEFAULT_TABLE_SEPARATOR = " "


class Column:
    """One Table column spec. ``width`` > 0 is a fixed cell count; otherwise
    the width is the widest of ``title`` and the formatted cells, raised to
    ``min_width``. ``flex`` distributes leftover cells on a declared
    ``Table.width``."""

    __slots__ = ("title", "width", "min_width", "flex", "align", "format")

    def __init__(self, title="", width=0, min_width=0, flex=0, align="",
                 format=None):
        self.title = title
        self.width = width
        self.min_width = min_width
        self.flex = flex
        self.align = align
        self.format = format


def _sprint(value):
    """fmt.Sprint for the common record value types."""
    if value is None:
        return "<nil>"
    if value is True:
        return "true"
    if value is False:
        return "false"
    return str(value)


class Table:
    """Header + rows of aligned, truncated text cells. ``rows`` are lists of
    strings; ``records`` are formatted by the column ``format`` (``str`` when
    absent). ``records`` wins when both are set. Cell text is truncated with
    ``builder.truncate``, so CJK/emoji never split in half."""

    def __init__(self, id="", columns=None, rows=None, records=None, width=0,
                 hide_header=False, rule=False, zebra=False, zebra_style="",
                 selected=0, style="", header_style="", selected_style="",
                 footer_style="", separator_style="", footer=None,
                 separator="", row_id=None):
        self.id = id
        self.columns = list(columns or [])
        self.rows = list(rows or [])
        self.records = list(records or [])
        self.width = width
        self.hide_header = hide_header
        self.rule = rule
        self.zebra = zebra
        self.zebra_style = zebra_style
        self.selected = selected
        self.style = style
        self.header_style = header_style
        self.selected_style = selected_style
        self.footer_style = footer_style
        self.separator_style = separator_style
        self.footer = list(footer or [])
        self.separator = separator
        self.row_id = row_id

    def row_count(self):
        """Number of data rows (``records`` wins over ``rows``)."""
        if self.records:
            return len(self.records)
        return len(self.rows)

    def cell(self, row, col):
        """The formatted text of one cell; out-of-range cells are empty."""
        if self.records:
            if row < 0 or row >= len(self.records) or col < 0 or col >= len(self.records[row]):
                return ""
            if col < len(self.columns) and self.columns[col].format is not None:
                return self.columns[col].format(self.records[row][col])
            return _sprint(self.records[row][col])
        if row < 0 or row >= len(self.rows) or col < 0 or col >= len(self.rows[row]):
            return ""
        return self.rows[row][col]

    def column_widths(self):
        """Solve the cell widths: natural size plus ``min_width``, then flex
        distribution over a declared width, then a right-to-left shrink to
        fit."""
        widths = [0] * len(self.columns)
        if not self.columns:
            return widths
        for i, col in enumerate(self.columns):
            if col.width > 0:
                widths[i] = col.width
                continue
            width = builder.display_width(col.title)
            for row in range(self.row_count()):
                cell = builder.display_width(self.cell(row, i))
                if cell > width:
                    width = cell
            floor = self._min_width(i)
            if width < floor:
                width = floor
            if width < 1:
                width = 1
            widths[i] = width
        if self.width <= 0:
            return widths
        avail = self.width - self._sep_width() * (len(self.columns) - 1)
        if avail <= 0:
            return widths
        used = sum(widths)
        if used < avail:
            flex = 0
            for col in self.columns:
                if col.width <= 0 and col.flex > 0:
                    flex += col.flex
            if flex == 0:
                return widths
            extra, added, last = avail - used, 0, -1
            for i, col in enumerate(self.columns):
                if col.width > 0 or col.flex <= 0:
                    continue
                share = extra * col.flex // flex
                widths[i] += share
                added += share
                last = i
            if last >= 0:
                widths[last] += extra - added
        elif used > avail:
            for i in range(len(widths) - 1, -1, -1):
                if used <= avail:
                    break
                if self.columns[i].width > 0:
                    continue
                reduce = used - avail
                floor = self._min_width(i)
                if widths[i] - reduce < floor:
                    reduce = widths[i] - floor
                if reduce > 0:
                    widths[i] -= reduce
                    used -= reduce
        return widths

    def build(self):
        """The table as a column of one-row boxes."""
        col = builder.box("col")
        if self.id:
            col.id(self.id)
        if self.width > 0:
            col.width(self.width)
        if not self.columns:
            return col
        widths = self.column_widths()
        if not self.hide_header:
            col.child(self._row_box(self._header_cells(), widths,
                                    self.header_style, "", -1))
            if self.rule:
                col.child(builder.text("\u2500" * self._total_width(widths))
                          .style(self.header_style).height(1))
        for row in range(self.row_count()):
            style = self.style
            if self.zebra and row % 2 == 1:
                style = first_non_empty(self.zebra_style, style)
            if row == self.selected:
                style = first_non_empty(self.selected_style, style)
            row_id = ""
            if self.row_id is not None:
                row_id = self.row_id(row)
            col.child(self._row_box(self._cells(row), widths, style, row_id, row))
        if self.footer:
            col.child(self._row_box(list(self.footer), widths,
                                    self.footer_style, "", -1))
        return col

    def row_at(self, y):
        """The data row at viewport y, skipping the header and rule. Returns
        ``(0, False)`` for the header/rule/footer or out-of-range rows."""
        if not self.columns:
            return 0, False
        top = 0
        if not self.hide_header:
            top += 1
            if self.rule:
                top += 1
        row = y - top
        if row < 0 or row >= self.row_count():
            return 0, False
        return row, True

    def _header_cells(self):
        return [col.title for col in self.columns]

    def _cells(self, row):
        return [self.cell(row, i) for i in range(len(self.columns))]

    def _row_box(self, cells, widths, style, row_id, row):
        box = builder.row().height(1)
        if row_id:
            box.id(row_id).input("mouse")
        for i, width in enumerate(widths):
            if i > 0:
                box.child(builder.text(self._separator_text())
                          .style(self.separator_style))
            text = ""
            if i < len(cells):
                text = cells[i]
            box.child(builder.text(pad_cell(text, width, self._align(i)))
                      .style(style).width(width).height(1))
        return box

    def _align(self, col):
        if 0 <= col < len(self.columns):
            return self.columns[col].align
        return ""

    def _min_width(self, col):
        if 0 <= col < len(self.columns) and self.columns[col].min_width > 0:
            return self.columns[col].min_width
        return 1

    def _separator_text(self):
        if not self.separator:
            return DEFAULT_TABLE_SEPARATOR
        return self.separator

    def _sep_width(self):
        return builder.display_width(self._separator_text())

    def _total_width(self, widths):
        if not widths:
            return 0
        total = self._sep_width() * (len(widths) - 1)
        for width in widths:
            total += width
        return total


def pad_cell(text, width, align):
    """Truncate text to width display cells and pad it according to the
    column alignment."""
    text = builder.truncate(text, width)
    pad = width - builder.display_width(text)
    if pad <= 0:
        return text
    if align == ALIGN_RIGHT:
        return " " * pad + text
    if align == ALIGN_CENTER:
        left = pad // 2
        return " " * left + text + " " * (pad - left)
    return text + " " * pad

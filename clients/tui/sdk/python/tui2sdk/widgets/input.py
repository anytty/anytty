"""Single- and multi-line editors (port of input.go).

The caller owns the state: feed keys through ``handle_key`` (or call the
editing methods) and commit the view itself; the widget never sends frames.
``cursor`` is a rune index into ``value``; ``offset`` is the first visible
rune (horizontal scroll) / the first visible line-column for TextArea.
"""

import unicodedata

from .. import builder

# Style of the rendered cursor cell when the widget declares no cursor_style.
DEFAULT_CURSOR_STYLE = "reverse"

# Protocol cursor shape declared by focused inputs.
DEFAULT_CURSOR_SHAPE = "bar"


def _coerce_runes(value):
    if value is None:
        return []
    if isinstance(value, str):
        return list(value)
    return list(value)


def _filter_plain_runes(runes):
    return _filter_runes(runes, False)


def _filter_runes(runes, newlines):
    out = []
    for r in runes:
        if r == "\n" and not newlines:
            continue
        if r == "\r":
            continue
        out.append(r)
    return out


def _clamp_cursor(cursor, length):
    if cursor < 0:
        return 0
    if cursor > length:
        return length
    return cursor


def _insert_rune_at(value, cursor, r):
    cursor = _clamp_cursor(cursor, len(value))
    value.insert(cursor, r)
    return cursor + 1


def _insert_runes_at(value, cursor, runes):
    cursor = _clamp_cursor(cursor, len(value))
    value[cursor:cursor] = runes
    return cursor + len(runes)


def _is_word_rune(r):
    if r == "_":
        return True
    category = unicodedata.category(r)
    return category.startswith("L") or category == "Nd"


def _word_left_index(value, cursor):
    cursor = _clamp_cursor(cursor, len(value))
    i = cursor
    while i > 0 and not _is_word_rune(value[i - 1]):
        i -= 1
    while i > 0 and _is_word_rune(value[i - 1]):
        i -= 1
    return i


def _word_right_index(value, cursor):
    cursor = _clamp_cursor(cursor, len(value))
    i = cursor
    while i < len(value) and _is_word_rune(value[i]):
        i += 1
    while i < len(value) and not _is_word_rune(value[i]):
        i += 1
    return i


def _cell_width(runes):
    return sum(builder.cell_width(r) for r in runes)


def _cursor_cell_width(runes, cursor):
    if 0 <= cursor < len(runes):
        width = builder.cell_width(runes[cursor])
        if width > 0:
            return width
    return 1


def _truncate_runes_to_cells(runes, width):
    if width <= 0:
        return []
    cells = 0
    for i, r in enumerate(runes):
        rw = builder.cell_width(r)
        if rw > 0 and cells + rw > width:
            return runes[:i]
        cells += rw
    return list(runes)


def _pad_to(text, width):
    text = builder.truncate(text, width)
    pad = width - builder.display_width(text)
    if pad > 0:
        text += " " * pad
    return text


def _cursor_shape(shape):
    if shape:
        return shape
    return DEFAULT_CURSOR_SHAPE


def printable_rune(ev):
    """The single printable rune of a key event (``{"key", "char"}``), or
    ``None``. Control runes are rejected."""
    if ev is None:
        return None
    text = ev.get("char") or ev.get("key") or ""
    if len(text) != 1 or unicodedata.category(text) == "Cc":
        return None
    return text


class TextInput:
    """A single-line editor state plus a pure ``build``. ``value`` is a list
    of runes; ``cursor`` is a rune index into it."""

    def __init__(self, id="", value=None, cursor=0, placeholder="", mask="",
                 max_len=0, width=0, offset=0, focused=False, style="",
                 placeholder_style="", cursor_style="", cursor_shape="",
                 input=None):
        self.id = id
        self.value = _coerce_runes(value)
        self.cursor = cursor
        self.placeholder = placeholder
        self.mask = mask
        self.max_len = max_len
        self.width = width
        self.offset = offset
        self.focused = focused
        self.style = style
        self.placeholder_style = placeholder_style
        self.cursor_style = cursor_style
        self.cursor_shape = cursor_shape
        self.input = list(input) if input is not None else None

    def text(self):
        """The raw value (``mask`` only affects rendering)."""
        return "".join(self.value)

    def display_text(self):
        """The rendered text: masked value, or the placeholder when the value
        is empty."""
        if not self.value and self.placeholder:
            return self.placeholder
        return "".join(self._display_runes())

    def set_value(self, s):
        """Replace the value and place the cursor at the end."""
        self.value = _filter_plain_runes(list(s))
        if self.max_len > 0 and len(self.value) > self.max_len:
            self.value = self.value[:self.max_len]
        self.cursor = len(self.value)

    def insert_rune(self, r):
        """Insert r at the cursor. Newlines are rejected (single line)."""
        if r == "\n" or r == "\r":
            return False
        if self.max_len > 0 and len(self.value) >= self.max_len:
            return False
        self.cursor = _insert_rune_at(self.value, self.cursor, r)
        return True

    def insert_string(self, s):
        """Insert s at the cursor, clipped to max_len; newlines are stripped.
        Reports whether anything was inserted."""
        runes = _filter_plain_runes(list(s))
        if not runes:
            return False
        if self.max_len > 0:
            room = self.max_len - len(self.value)
            if room <= 0:
                return False
            if len(runes) > room:
                runes = runes[:room]
        self.cursor = _insert_runes_at(self.value, self.cursor, runes)
        return True

    def backspace(self):
        """Delete the rune before the cursor."""
        self.cursor = _clamp_cursor(self.cursor, len(self.value))
        if self.cursor == 0:
            return False
        del self.value[self.cursor - 1]
        self.cursor -= 1
        return True

    def delete(self):
        """Delete the rune at the cursor."""
        self.cursor = _clamp_cursor(self.cursor, len(self.value))
        if self.cursor >= len(self.value):
            return False
        del self.value[self.cursor]
        return True

    def delete_word_left(self):
        """Delete from the cursor back to the previous word start."""
        self.cursor = _clamp_cursor(self.cursor, len(self.value))
        start = _word_left_index(self.value, self.cursor)
        if start == self.cursor:
            return False
        del self.value[start:self.cursor]
        self.cursor = start
        return True

    def delete_to_end(self):
        """Delete from the cursor to the end of the value."""
        self.cursor = _clamp_cursor(self.cursor, len(self.value))
        if self.cursor >= len(self.value):
            return False
        del self.value[self.cursor:]
        return True

    def left(self):
        """Move the cursor one rune left."""
        self.cursor = _clamp_cursor(self.cursor, len(self.value))
        if self.cursor == 0:
            return False
        self.cursor -= 1
        return True

    def right(self):
        """Move the cursor one rune right."""
        self.cursor = _clamp_cursor(self.cursor, len(self.value))
        if self.cursor >= len(self.value):
            return False
        self.cursor += 1
        return True

    def word_left(self):
        """Move the cursor to the previous word start."""
        self.cursor = _clamp_cursor(self.cursor, len(self.value))
        nxt = _word_left_index(self.value, self.cursor)
        if nxt == self.cursor:
            return False
        self.cursor = nxt
        return True

    def word_right(self):
        """Move the cursor past the next word."""
        self.cursor = _clamp_cursor(self.cursor, len(self.value))
        nxt = _word_right_index(self.value, self.cursor)
        if nxt == self.cursor:
            return False
        self.cursor = nxt
        return True

    def home(self):
        """Move the cursor to the start."""
        moved = self.cursor != 0
        self.cursor = 0
        return moved

    def end(self):
        """Move the cursor past the last rune."""
        moved = self.cursor != len(self.value)
        self.cursor = len(self.value)
        return moved

    def handle_key(self, ev):
        """Apply one key event and report whether the input consumed it.
        Enter/Tab/Esc/Up/Down are left for the caller (commit)."""
        if ev is None:
            return False
        key = ev.get("key")
        if key == "left":
            self.left()
            return True
        if key == "right":
            self.right()
            return True
        if key in ("home", "ctrl-a"):
            self.home()
            return True
        if key in ("end", "ctrl-e"):
            self.end()
            return True
        if key == "backspace":
            self.backspace()
            return True
        if key == "delete":
            self.delete()
            return True
        if key in ("ctrl-left", "alt-left", "ctrl-b"):
            self.word_left()
            return True
        if key in ("ctrl-right", "alt-right", "ctrl-f"):
            self.word_right()
            return True
        if key in ("ctrl-w", "alt-backspace"):
            self.delete_word_left()
            return True
        if key == "ctrl-k":
            self.delete_to_end()
            return True
        if key == "ctrl-u":
            self.delete_to_end()
            self.home()
            return True
        r = printable_rune(ev)
        if r is not None:
            self.insert_rune(r)
            return True
        return False

    def build(self):
        """The input as a one-row box tree. A focused input carries the
        program cursor; the cursor cell is rendered with ``cursor_style``
        (default reverse) so the row reads as one text run."""
        box = _input_row_box(self.id, self.width)
        box.focused(self.focused)
        box.input(*self._inputs())
        if not self.value and self.placeholder:
            text = self.placeholder
            if self.width > 0:
                text = builder.truncate(text, self.width)
            box.child(builder.text(text).style(self.placeholder_style).height(1))
            self._pad(box, builder.display_width(text))
            if self.focused:
                box.cursor(0, 0, _cursor_shape(self.cursor_shape))
            return box
        runes = self._display_runes()
        cursor = _clamp_cursor(self.cursor, len(runes))
        start = self._visible_offset(runes)
        visible = runes[start:]
        if not self.focused:
            text = "".join(visible)
            if self.width > 0:
                text = _pad_to(text, self.width)
            if text:
                box.child(builder.text(text).style(self.style).height(1))
            return box
        rel = _clamp_cursor(cursor - start, len(visible))
        pre = "".join(visible[:rel])
        post = visible[rel:]
        cursor_text = " "
        if post:
            cursor_text = post[0]
            post = post[1:]
        post_text = "".join(post)
        if self.width > 0:
            post_text = builder.truncate(
                post_text,
                self.width - builder.display_width(pre) - builder.display_width(cursor_text))
        if pre:
            box.child(builder.text(pre).style(self.style).height(1))
        box.child(builder.text(cursor_text)
                  .style(self.cursor_style or DEFAULT_CURSOR_STYLE).height(1))
        if post_text:
            box.child(builder.text(post_text).style(self.style).height(1))
        self._pad(box, builder.display_width(pre)
                  + builder.display_width(cursor_text)
                  + builder.display_width(post_text))
        box.cursor(0, builder.display_width(pre), _cursor_shape(self.cursor_shape))
        return box

    def _display_runes(self):
        if not self.mask:
            return self.value
        return [self.mask] * len(self.value)

    def _visible_offset(self, runes):
        """Scroll the rune window so the cursor cell fits ``width`` cells."""
        if self.width <= 0:
            return 0
        offset = _clamp_cursor(self.offset, len(runes))
        cursor = _clamp_cursor(self.cursor, len(runes))
        if cursor < offset:
            offset = cursor
        while offset < cursor and (
                _cell_width(runes[offset:cursor])
                + _cursor_cell_width(runes, cursor) > self.width):
            offset += 1
        return offset

    def _pad(self, box, used):
        if self.width <= 0 or used >= self.width:
            return
        box.child(builder.text(" " * (self.width - used)))

    def _inputs(self):
        if self.input:
            return self.input
        return ["key", "paste"]


class TextArea:
    """The multi-line counterpart of TextInput: ``cursor`` is a rune index
    into ``value`` and ``row_offset``/``col_offset`` are the first visible
    line/column. ``build`` is pure and renders the visible rows only."""

    def __init__(self, id="", value=None, cursor=0, placeholder="", mask="",
                 max_len=0, width=0, height=0, row_offset=0, col_offset=0,
                 focused=False, style="", placeholder_style="",
                 cursor_style="", cursor_shape="", input=None):
        self.id = id
        self.value = _coerce_runes(value)
        self.cursor = cursor
        self.placeholder = placeholder
        self.mask = mask
        self.max_len = max_len
        self.width = width
        self.height = height
        self.row_offset = row_offset
        self.col_offset = col_offset
        self.focused = focused
        self.style = style
        self.placeholder_style = placeholder_style
        self.cursor_style = cursor_style
        self.cursor_shape = cursor_shape
        self.input = list(input) if input is not None else None

    def text(self):
        """The raw value."""
        return "".join(self.value)

    def lines(self):
        """The raw value split into lines."""
        return self.text().split("\n")

    def line(self, row):
        """One raw line, empty when out of range."""
        lines = self.lines()
        if row < 0 or row >= len(lines):
            return ""
        return lines[row]

    def row_col(self):
        """The cursor line and its rune column."""
        cursor = _clamp_cursor(self.cursor, len(self.value))
        row, start = 0, 0
        for i in range(cursor):
            if self.value[i] == "\n":
                row += 1
                start = i + 1
        return row, cursor - start

    def set_value(self, s):
        """Replace the value and place the cursor at the end."""
        self.value = _filter_runes(list(s.replace("\r\n", "\n")), True)
        if self.max_len > 0 and len(self.value) > self.max_len:
            self.value = self.value[:self.max_len]
        self.cursor = len(self.value)

    def insert_rune(self, r):
        """Insert r at the cursor; "\\n" starts a new line."""
        if r == "\r":
            return False
        if self.max_len > 0 and len(self.value) >= self.max_len:
            return False
        self.cursor = _insert_rune_at(self.value, self.cursor, r)
        return True

    def insert_string(self, s):
        """Insert s at the cursor, clipped to max_len. CRLF is normalized."""
        runes = _filter_runes(list(s.replace("\r\n", "\n")), True)
        if not runes:
            return False
        if self.max_len > 0:
            room = self.max_len - len(self.value)
            if room <= 0:
                return False
            if len(runes) > room:
                runes = runes[:room]
        self.cursor = _insert_runes_at(self.value, self.cursor, runes)
        return True

    def backspace(self):
        """Delete the rune before the cursor (joining lines at a line start)."""
        self.cursor = _clamp_cursor(self.cursor, len(self.value))
        if self.cursor == 0:
            return False
        del self.value[self.cursor - 1]
        self.cursor -= 1
        return True

    def delete(self):
        """Delete the rune at the cursor."""
        self.cursor = _clamp_cursor(self.cursor, len(self.value))
        if self.cursor >= len(self.value):
            return False
        del self.value[self.cursor]
        return True

    def delete_word_left(self):
        """Delete from the cursor back to the previous word start."""
        self.cursor = _clamp_cursor(self.cursor, len(self.value))
        start = _word_left_index(self.value, self.cursor)
        if start == self.cursor:
            return False
        del self.value[start:self.cursor]
        self.cursor = start
        return True

    def left(self):
        """Move the cursor one rune left."""
        self.cursor = _clamp_cursor(self.cursor, len(self.value))
        if self.cursor == 0:
            return False
        self.cursor -= 1
        return True

    def right(self):
        """Move the cursor one rune right."""
        self.cursor = _clamp_cursor(self.cursor, len(self.value))
        if self.cursor >= len(self.value):
            return False
        self.cursor += 1
        return True

    def word_left(self):
        """Move the cursor to the previous word start."""
        self.cursor = _clamp_cursor(self.cursor, len(self.value))
        nxt = _word_left_index(self.value, self.cursor)
        if nxt == self.cursor:
            return False
        self.cursor = nxt
        return True

    def word_right(self):
        """Move the cursor past the next word."""
        self.cursor = _clamp_cursor(self.cursor, len(self.value))
        nxt = _word_right_index(self.value, self.cursor)
        if nxt == self.cursor:
            return False
        self.cursor = nxt
        return True

    def up(self):
        """Move the cursor one line up, keeping the column when possible."""
        row, col = self.row_col()
        if row == 0:
            return False
        bounds = self._line_bounds()
        start, end = bounds[row - 1]
        self.cursor = _clamp_cursor(start + col, end)
        return True

    def down(self):
        """Move the cursor one line down, keeping the column when possible."""
        row, col = self.row_col()
        bounds = self._line_bounds()
        if row >= len(bounds) - 1:
            return False
        start, end = bounds[row + 1]
        self.cursor = _clamp_cursor(start + col, end)
        return True

    def home(self):
        """Move the cursor to the start of its line."""
        row, _ = self.row_col()
        start = self._line_bounds()[row][0]
        moved = self.cursor != start
        self.cursor = start
        return moved

    def end(self):
        """Move the cursor to the end of its line."""
        row, _ = self.row_col()
        end = self._line_bounds()[row][1]
        moved = self.cursor != end
        self.cursor = end
        return moved

    def page_up(self):
        """Move the cursor up one visible page."""
        moved = False
        for _ in range(self._page_height()):
            if not self.up():
                break
            moved = True
        return moved

    def page_down(self):
        """Move the cursor down one visible page."""
        moved = False
        for _ in range(self._page_height()):
            if not self.down():
                break
            moved = True
        return moved

    def handle_key(self, ev):
        """Apply one key event and report whether the textarea consumed it.
        Enter inserts a newline; Esc/Tab are left for the caller."""
        if ev is None:
            return False
        key = ev.get("key")
        if key == "left":
            self.left()
            return True
        if key == "right":
            self.right()
            return True
        if key == "up":
            self.up()
            return True
        if key == "down":
            self.down()
            return True
        if key in ("home", "ctrl-a"):
            self.home()
            return True
        if key in ("end", "ctrl-e"):
            self.end()
            return True
        if key == "page-up":
            self.page_up()
            return True
        if key == "page-down":
            self.page_down()
            return True
        if key == "backspace":
            self.backspace()
            return True
        if key == "delete":
            self.delete()
            return True
        if key in ("ctrl-left", "alt-left", "ctrl-b"):
            self.word_left()
            return True
        if key in ("ctrl-right", "alt-right", "ctrl-f"):
            self.word_right()
            return True
        if key in ("ctrl-w", "alt-backspace"):
            self.delete_word_left()
            return True
        if key in ("enter", "ctrl-j", "ctrl-m"):
            self.insert_rune("\n")
            return True
        r = printable_rune(ev)
        if r is not None:
            self.insert_rune(r)
            return True
        return False

    def build(self):
        """The textarea as a column of visible line rows."""
        col = builder.box("col")
        if self.id:
            col.id(self.id)
        col.focused(self.focused)
        col.input(*self._inputs())
        if self.width > 0:
            col.width(self.width)
        if self.height > 0:
            col.height(self.height)
        if not self.value and self.placeholder:
            text = self.placeholder
            if self.width > 0:
                text = _pad_to(text, self.width)
            col.child(builder.text(text).style(self.placeholder_style).height(1))
            if self.focused:
                col.cursor(0, 0, _cursor_shape(self.cursor_shape))
            return col
        lines = "".join(self._display_runes()).split("\n")
        if not lines:
            lines = [""]
        cursor_row, cursor_col = self.row_col()
        offset = self._row_offset(len(lines))
        col_offset, cursor_display_col = self._col_window(lines, cursor_row, cursor_col)
        height = self._visible_height(len(lines) - offset)
        for i in range(height):
            row = offset + i
            col.child(self._line_box(list(lines[row]), row == cursor_row,
                                     cursor_col, col_offset))
        if self.focused:
            row = cursor_row - offset
            if row < 0:
                row = 0
            col.cursor(row, cursor_display_col, _cursor_shape(self.cursor_shape))
        return col

    def _display_runes(self):
        if not self.mask:
            return self.value
        return [r if r == "\n" else self.mask for r in self.value]

    def _line_bounds(self):
        """[start, end) rune bounds per line, end excluding "\\n"."""
        bounds = [[0, 0]]
        for i, r in enumerate(self.value):
            if r == "\n":
                bounds[-1][1] = i
                bounds.append([i + 1, i + 1])
        bounds[-1][1] = len(self.value)
        return bounds

    def _row_offset(self, lines):
        height = self._visible_height(lines)
        offset = self.row_offset
        cursor_row, _ = self.row_col()
        if height <= 0:
            return 0
        if cursor_row < offset:
            offset = cursor_row
        if cursor_row >= offset + height:
            offset = cursor_row - height + 1
        maximum = lines - height
        if offset > maximum:
            offset = maximum
        if offset < 0:
            offset = 0
        return offset

    def _visible_height(self, lines):
        height = self.height
        if height <= 0:
            height = lines
        if height < 0:
            return 0
        if height > lines:
            height = lines
        return height

    def _page_height(self):
        if self.height > 1:
            return self.height
        return 1

    def _col_window(self, lines, cursor_row, cursor_col):
        """The horizontal offset and the cursor display column."""
        line = _line_runes(lines, cursor_row)
        if self.width <= 0:
            return 0, _cell_width(line[:_clamp_cursor(cursor_col, len(line))])
        offset = self.col_offset
        if offset < 0:
            offset = 0
        if cursor_col < offset:
            offset = cursor_col
        if offset > len(line):
            offset = len(line)
        while offset < cursor_col and (
                _cell_width(line[offset:cursor_col])
                + _cursor_cell_width(line, cursor_col) > self.width):
            offset += 1
        if offset > cursor_col:
            offset = cursor_col
        col = _cell_width(line[offset:_clamp_cursor(cursor_col, len(line))])
        return offset, col

    def _line_box(self, line, cursor_line, cursor_col, offset):
        row = builder.row().height(1)
        if self.width > 0:
            row.width(self.width)
        start = _clamp_cursor(offset, len(line))
        visible = line[start:]
        if self.width > 0:
            visible = _truncate_runes_to_cells(visible, self.width)
        if not cursor_line or not self.focused:
            text = "".join(visible)
            if self.width > 0:
                text = _pad_to(text, self.width)
            if text:
                row.child(builder.text(text).style(self.style).height(1))
            return row
        rel = _clamp_cursor(cursor_col - start, len(visible))
        pre = "".join(visible[:rel])
        post = visible[rel:]
        cursor_text = " "
        if post:
            cursor_text = post[0]
            post = post[1:]
        post_text = "".join(post)
        if self.width > 0:
            post_text = builder.truncate(
                post_text,
                self.width - builder.display_width(pre) - builder.display_width(cursor_text))
        if pre:
            row.child(builder.text(pre).style(self.style).height(1))
        row.child(builder.text(cursor_text)
                  .style(self.cursor_style or DEFAULT_CURSOR_STYLE).height(1))
        if post_text:
            row.child(builder.text(post_text).style(self.style).height(1))
        used = (builder.display_width(pre) + builder.display_width(cursor_text)
                + builder.display_width(post_text))
        if self.width > used:
            row.child(builder.text(" " * (self.width - used)))
        return row

    def _inputs(self):
        if self.input:
            return self.input
        return ["key", "paste"]


def _line_runes(lines, row):
    if row < 0 or row >= len(lines):
        return []
    return list(lines[row])


def _input_row_box(input_id, width):
    box = builder.row().height(1)
    if input_id:
        box.id(input_id)
    if width > 0:
        box.width(width)
    return box

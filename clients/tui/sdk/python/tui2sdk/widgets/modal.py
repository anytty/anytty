"""Program-side dialogs and menus (port of modal.go).

Nothing is emitted by the widgets; the caller decides what a click/commit
means and keeps/drops the modal from the next view. ``Modal`` and ``Menu``
place a framed floating layer, whose frame layout is reproduced privately
here (the visual half of the toolkit owns the canonical Frame widget).
"""

from .. import builder
from .input import printable_rune
from .list import DEFAULT_LIST_MARKER, first_non_empty

# Dims everything behind a Modal.
DEFAULT_BACKDROP_STYLE = "dim"

# Border style of the private framed layer.
_DEFAULT_BORDER_STYLE = "border"


class FrameRow:
    """One content row of a framed layer: the text, its style, an optional
    hit-test id and the input kinds that id accepts."""

    __slots__ = ("text", "style", "id", "input")

    def __init__(self, text="", style="", id="", input=None):
        self.text = text
        self.style = style
        self.id = id
        self.input = list(input) if input is not None else None


def _pad_to(text, width):
    text = builder.truncate(text, width)
    pad = width - builder.display_width(text)
    if pad > 0:
        text += " " * pad
    return text


def _max_int(a, b):
    return a if a > b else b


def _rule_text(width, title):
    """A top rule of exactly width display cells with an optional centered
    title, always leaving at least one horizontal glyph."""
    inner = width - 2
    if inner <= 0:
        return "\u250c\u2510"
    if not title:
        return "\u250c" + "\u2500" * inner + "\u2510"
    label = " " + title + " "
    budget = inner - 1
    if budget >= 0:
        label = builder.truncate(label, budget)
    else:
        label = ""
    fill = inner - 1 - builder.display_width(label)
    if fill < 0:
        fill = 0
    return "\u250c\u2500" + label + "\u2500" * fill + "\u2510"


def _bottom_rule_text(width):
    if width < 2:
        return "\u2514"
    return "\u2514" + "\u2500" * (width - 2) + "\u2518"


def _frame_text_width(rows):
    width = 0
    for row in rows:
        w = builder.display_width(row.text)
        if w > width:
            width = w
    return width


def _frame_row_box(row, width, fallback_style):
    style = row.style or fallback_style
    box = builder.text(_pad_to(row.text, width)).style(style).width(width).height(1)
    if row.id:
        box.id(row.id)
    if row.input:
        box.input(*row.input)
    return box


def _frame_build(frame_id, title, width, height, style, rows, fill_style=""):
    """Private port of the Frame widget: a program-side bordered panel."""
    if width <= 0:
        width = _frame_text_width(rows) + 2
    if height <= 0:
        height = len(rows) + 2
    if not style:
        style = _DEFAULT_BORDER_STYLE
    col = builder.box("col").id(frame_id).width(width).height(height)
    if width < 3 or height < 2:
        for row in rows:
            col.child(_frame_row_box(row, _max_int(1, width), ""))
        return col
    inner = width - 2
    col.child(builder.text(_rule_text(width, title)).style(style)
              .width(width).height(1))
    for i in range(height - 2):
        row = FrameRow(style=fill_style)
        if i < len(rows):
            row = rows[i]
        line = builder.row().height(1)
        line.child(builder.text("\u2502").style(style).width(1))
        line.child(_frame_row_box(row, inner, ""))
        line.child(builder.text("\u2502").style(style).width(1))
        col.child(line)
    col.child(builder.text(_bottom_rule_text(width)).style(style)
              .width(width).height(1))
    return col


def _floating_layer(layer_id, title, x, y, width, height, style, rows,
                    collapsed=False):
    """Private port of FloatingLayer: a Frame placed with pos."""
    if collapsed:
        height = 1
    return _frame_build(layer_id, title, width, height, style, rows).pos(x, y)


class Modal:
    """A program-side dialog: a framed layer placed by a floating layer,
    optionally centered in a known parent and optionally sitting on a dimming
    backdrop."""

    def __init__(self, id="", title="", x=0, y=0, width=0, height=0,
                 rows=None, style="", backdrop=False, backdrop_style="",
                 backdrop_id="", center=False, parent_width=0,
                 parent_height=0):
        self.id = id
        self.title = title
        self.x = x
        self.y = y
        self.width = width
        self.height = height
        self.rows = list(rows or [])
        self.style = style
        self.backdrop = backdrop
        self.backdrop_style = backdrop_style
        self.backdrop_id = backdrop_id
        self.center = center
        self.parent_width = parent_width
        self.parent_height = parent_height

    def position(self):
        """The floating layer origin: ``(x, y)``, or the centered origin when
        ``center`` is set and the parent size is known. Never negative."""
        x, y = self.x, self.y
        if self.center and self.parent_width > 0 and self.parent_height > 0:
            x = (self.parent_width - self.width) // 2
            y = (self.parent_height - self.height) // 2
        if x < 0:
            x = 0
        if y < 0:
            y = 0
        return x, y

    def build(self):
        """A stack: the optional backdrop stretches to fill the parent, the
        framed layer is positioned on top."""
        x, y = self.position()
        stack = builder.stack()
        if self.parent_width > 0:
            stack.width(self.parent_width)
        if self.parent_height > 0:
            stack.height(self.parent_height)
        if self.backdrop:
            backdrop = builder.box().style(
                first_non_empty(self.backdrop_style, DEFAULT_BACKDROP_STYLE))
            backdrop_id = self.backdrop_id
            if not backdrop_id and self.id:
                backdrop_id = self.id + ":backdrop"
            if backdrop_id:
                backdrop.id(backdrop_id)
            if self.parent_width > 0:
                backdrop.width(self.parent_width)
            if self.parent_height > 0:
                backdrop.height(self.parent_height)
            stack.child(backdrop)
        stack.child(_floating_layer(self.id, self.title, x, y, self.width,
                                    self.height, self.style, self.rows))
        return stack


class MenuItem:
    """One Menu row. A separator item draws a rule, ignores hotkey and is
    never selectable; a disabled item is rendered with ``disabled_style`` and
    skipped by move/hotkey. ``id`` is the value carried to the caller
    (``label`` is the fallback)."""

    __slots__ = ("id", "label", "hotkey", "disabled", "separator", "style")

    def __init__(self, id="", label="", hotkey="", disabled=False,
                 separator=False, style=""):
        self.id = id
        self.label = label
        self.hotkey = hotkey
        self.disabled = disabled
        self.separator = separator
        self.style = style

    def value(self):
        """The item identity emitted by ``Menu.value``."""
        if self.id:
            return self.id
        return self.label


def _abs_int(value):
    return -value if value < 0 else value


class Menu:
    """A keyboard-driven overlay menu: items with optional hotkeys, separators
    and disabled entries. ``move``/``hotkey`` update ``selected``; the caller
    reads ``value()`` and dispatches it in the same shape as Button/Picker
    values (source = component id, value = item id)."""

    _FIELDS = ("id", "title", "items", "selected", "width", "x", "y", "style",
               "selected_style", "disabled_style", "separator_style",
               "frame_style", "marker")

    def __init__(self, id="", title="", items=None, selected=0, width=0, x=0,
                 y=0, style="", selected_style="", disabled_style="",
                 separator_style="", frame_style="", marker=""):
        self.id = id
        self.title = title
        self.items = list(items or [])
        self.selected = selected
        self.width = width
        self.x = x
        self.y = y
        self.style = style
        self.selected_style = selected_style
        self.disabled_style = disabled_style
        self.separator_style = separator_style
        self.frame_style = frame_style
        self.marker = marker

    def move(self, delta):
        """Shift ``selected`` by delta selectable items, skipping separators
        and disabled items; it stops at the boundaries."""
        if delta == 0 or not self.items:
            return
        self.selected = _clamp_index(self.selected, len(self.items))
        step = 1
        if delta < 0:
            step = -1
        for _ in range(_abs_int(delta)):
            nxt = self.selected + step
            while 0 <= nxt < len(self.items) and not self._selectable(nxt):
                nxt += step
            if nxt < 0 or nxt >= len(self.items):
                break
            self.selected = nxt

    def select(self, index):
        """Move the selection to index when it is selectable."""
        if not self._selectable(index):
            return False
        self.selected = index
        return True

    def hotkey(self, ev):
        """Match one key event against the item hotkeys (case-insensitive,
        disabled/separator items never match), select the item and return its
        value. A non-matching event returns ``("", False)``."""
        r = printable_rune(ev)
        if r is None:
            return "", False
        for i, item in enumerate(self.items):
            if item.separator or item.disabled or not item.hotkey:
                continue
            hot = item.hotkey
            if len(hot) == 1 and hot.lower() == r.lower():
                self.selected = i
                return item.value(), True
        return "", False

    def value(self):
        """The selected item value, empty when nothing selectable is
        selected."""
        item, ok = self.selected_item()
        if not ok:
            return ""
        return item.value()

    def selected_item(self):
        """``(item, True)`` when the selected item is selectable."""
        if self.selected < 0 or self.selected >= len(self.items):
            return None, False
        item = self.items[self.selected]
        if item.separator or item.disabled:
            return item, False
        return item, True

    def build(self):
        """The menu as a positioned floating layer."""
        marker = first_non_empty(self.marker, DEFAULT_LIST_MARKER)
        rows = []
        for i, item in enumerate(self.items):
            if item.separator:
                rows.append(FrameRow("\u2500" * 3,
                                     first_non_empty(self.separator_style,
                                                     self.disabled_style)))
                continue
            text = "  " + item.label
            style = first_non_empty(item.style, self.style)
            if i == self.selected and self._selectable(i):
                text = marker + item.label
                style = first_non_empty(self.selected_style, style)
            if item.disabled:
                style = first_non_empty(self.disabled_style, style)
            row = FrameRow(text, style)
            if item.id:
                row.id = item.id
                row.input = ["mouse"]
            rows.append(row)
        x, y = self._layer_pos()
        return _floating_layer(self.id, self.title, x, y, self.width, 0,
                               self.frame_style, rows)

    def _layer_pos(self):
        return self.x, self.y

    def _selectable(self, index):
        if index < 0 or index >= len(self.items):
            return False
        item = self.items[index]
        return not item.separator and not item.disabled


def _clamp_index(i, count):
    if i < 0 or count <= 0:
        return 0
    if i >= count:
        return count - 1
    return i

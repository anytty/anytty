"""Dropdown select state machine (port of select.go).

The caller owns the value and toggles ``open``; the widget never emits frames.
``value`` is the ``Option.value`` of the selection ("" means none, so
``placeholder`` is shown).
"""

from .. import builder
from .list import DEFAULT_LIST_MARKER, first_non_empty, list_row_box

# The closed-state caret drawn after the value.
DEFAULT_SELECT_INDICATOR = " \u25be"


class Option:
    """One Select choice. Disabled options render with ``disabled_style`` and
    are skipped by ``move`` and ``typeahead``."""

    __slots__ = ("value", "label", "disabled", "style")

    def __init__(self, value="", label="", disabled=False, style=""):
        self.value = value
        self.label = label
        self.disabled = disabled
        self.style = style

    def display(self):
        """The option label, falling back to its value."""
        if self.label:
            return self.label
        return self.value


def _abs_int(value):
    return -value if value < 0 else value


def _to_lower_ascii_char(ch):
    if "A" <= ch <= "Z":
        return chr(ord(ch) + 32)
    return ch


def _to_lower_ascii(s):
    return "".join(_to_lower_ascii_char(ch) for ch in s)


def _has_prefix_fold(text, lower_prefix):
    """Whether text starts with lower_prefix, folding ASCII case without
    allocating a lowercased copy of text."""
    if len(text) < len(lower_prefix):
        return False
    for i, ch in enumerate(lower_prefix):
        if _to_lower_ascii_char(text[i]) != ch:
            return False
    return True


def _pad_spaces(n):
    if n <= 0:
        return ""
    return " " * n


class Select:
    """A dropdown state machine plus a pure ``build``."""

    def __init__(self, id="", label="", options=None, value="", open=False,
                 width=0, placeholder="", style="", label_style="",
                 placeholder_style="", selected_style="", disabled_style="",
                 marker="", indicator="", dropdown_style=""):
        self.id = id
        self.label = label
        self.options = list(options or [])
        self.value = value
        self.open = open
        self.width = width
        self.placeholder = placeholder
        self.style = style
        self.label_style = label_style
        self.placeholder_style = placeholder_style
        self.selected_style = selected_style
        self.disabled_style = disabled_style
        self.marker = marker
        self.indicator = indicator
        self.dropdown_style = dropdown_style

    def index(self):
        """The position of the selected option, or -1 when nothing is
        selected."""
        for i, option in enumerate(self.options):
            if option.value == self.value:
                return i
        return -1

    def selected(self):
        """``(option, True)`` for the selected option, ``(None, False)`` when
        nothing is selected or the selection is disabled (Go ``Selected``)."""
        i = self.index()
        if i < 0 or self.options[i].disabled:
            return None, False
        return self.options[i], True

    def select_index(self, index):
        """Select the option at index when it is in range and enabled."""
        if index < 0 or index >= len(self.options) or self.options[index].disabled:
            return False
        self.value = self.options[index].value
        return True

    def move(self, delta):
        """Shift the selection by delta enabled options, skipping disabled
        ones, and stop at the boundaries. With nothing selected it starts from
        the first (or last, for a negative delta) enabled option."""
        if delta == 0 or not self.options:
            return
        current = self.index()
        step = 1
        if delta < 0:
            step = -1
        if current < 0:
            current = -1
            if step < 0:
                current = len(self.options)
        for _ in range(_abs_int(delta)):
            nxt = current + step
            while 0 <= nxt < len(self.options) and self.options[nxt].disabled:
                nxt += step
            if nxt < 0 or nxt >= len(self.options):
                break
            current = nxt
        if 0 <= current < len(self.options):
            self.value = self.options[current].value

    def typeahead(self, prefix):
        """Select the next enabled option whose label or value starts with
        prefix (ASCII-case-insensitive), searching after the current selection
        and wrapping around. An empty prefix matches nothing."""
        if prefix == "" or not self.options:
            return False
        needle = _to_lower_ascii(prefix)
        count = len(self.options)
        start = self.index()
        for i in range(1, count + 1):
            index = start + i
            if start < 0:
                index = i - 1
            index %= count
            if index < 0:
                index += count
            option = self.options[index]
            if option.disabled:
                continue
            if (_has_prefix_fold(option.display(), needle)
                    or _has_prefix_fold(option.value, needle)):
                self.value = option.value
                return True
        return False

    def build(self):
        """The dropdown when open, otherwise the closed value row."""
        if self.open:
            return self.build_dropdown()
        col = self._container()
        if self.label:
            col.child(builder.text(self.label).style(self.label_style).height(1))
        row = builder.row().height(1)
        if self.width > 0:
            row.width(self.width)
        option, ok = self.selected()
        text = self.placeholder
        style = self.placeholder_style
        if ok:
            text = option.display()
            style = first_non_empty(option.style, self.selected_style, self.style)
        if text == "":
            text = " "
        indicator = first_non_empty(self.indicator, DEFAULT_SELECT_INDICATOR)
        total = self.width
        if total > 0:
            text = builder.truncate(
                text, max(0, total - builder.display_width(indicator)))
        row.child(builder.text(text).style(style).height(1))
        row.child(builder.text(indicator).style(self.placeholder_style).height(1))
        if total > 0:
            pad = total - builder.display_width(text) - builder.display_width(indicator)
            if pad > 0:
                row.child(builder.text(_pad_spaces(pad)))
        col.child(row)
        return col

    def build_dropdown(self):
        """The open list of options as a column of one-row boxes, ready to be
        placed in a floating layer or modal. Disabled options use
        ``disabled_style`` and the selected option carries the marker."""
        col = self._container()
        if self.dropdown_style:
            col.style(self.dropdown_style)
        selected = self.index()
        marker = first_non_empty(self.marker, DEFAULT_LIST_MARKER)
        for i, option in enumerate(self.options):
            text = "  " + option.display()
            style = first_non_empty(option.style, self.style)
            if option.disabled:
                style = first_non_empty(self.disabled_style, style)
            if i == selected and not option.disabled:
                text = marker + option.display()
                style = first_non_empty(self.selected_style, style)
            col.child(list_row_box(text, style, "", self.width))
        return col

    def _container(self):
        col = builder.box("col")
        if self.id:
            col.id(self.id)
        if self.width > 0:
            col.width(self.width)
        return col

"""Widget design tokens and themed widget constructors.

Every :class:`Theme` field is an opaque style string (a host-internal token
name or an explicit "fg:#RRGGBB;bg:#RRGGBB;bold" style), so widgets stay
palette-agnostic and the host remains the single ANSI resolver.
:func:`dark_theme` maps onto host tokens; :func:`light_theme` pins explicit
light colors for programs that want a light look independent of the host.
"""

import copy
import importlib

from .progress import Badge, ProgressBar, Spinner, Tags


class Theme:
    """The widget design-token set."""

    def __init__(self, name="", text="", title="", accent="", muted="",
                 success="", warning="", danger="", border="",
                 border_focus="", selection="", selection_text="", input="",
                 placeholder="", cursor="", status_bar="", tab_active="",
                 tab_inactive="", header="", footer="", toast="", overlay="",
                 backdrop="", marker="", separator="", zebra=""):
        self.name = name
        self.text = text
        self.title = title
        self.accent = accent
        self.muted = muted
        self.success = success
        self.warning = warning
        self.danger = danger
        self.border = border
        self.border_focus = border_focus
        self.selection = selection
        self.selection_text = selection_text
        self.input = input
        self.placeholder = placeholder
        self.cursor = cursor
        self.status_bar = status_bar
        self.tab_active = tab_active
        self.tab_inactive = tab_inactive
        self.header = header
        self.footer = footer
        self.toast = toast
        self.overlay = overlay
        self.backdrop = backdrop
        self.marker = marker
        self.separator = separator
        self.zebra = zebra

    def __eq__(self, other):
        if not isinstance(other, Theme):
            return NotImplemented
        return self.__dict__ == other.__dict__

    def __repr__(self):
        return "Theme(name=%r)" % (self.name,)


def dark_theme():
    """The token-based default; follows the host palette."""
    return Theme(
        name="dark",
        text="",
        title="chrome",
        accent="accent",
        muted="muted",
        success="success",
        warning="warning",
        danger="danger",
        border="border",
        border_focus="border_focus",
        selection="selection",
        selection_text="selection",
        input="",
        placeholder="muted",
        cursor="reverse",
        status_bar="status",
        tab_active="tab_active",
        tab_inactive="tab_inactive",
        header="status",
        footer="muted",
        toast="overlay",
        overlay="overlay",
        backdrop="dim",
        marker="accent",
        separator="muted",
        zebra="muted",
    )


def light_theme():
    """An explicit light palette independent of the host token table."""
    return Theme(
        name="light",
        text="fg:#24242a",
        title="fg:#f7f6fa;bg:#6d5ae6;bold",
        accent="fg:#6d3bd4;bold",
        muted="fg:#77737f",
        success="fg:#1d8a4a;bold",
        warning="fg:#a96b00;bold",
        danger="fg:#c02b3a;bold",
        border="fg:#b9b4c4",
        border_focus="fg:#6d3bd4",
        selection="fg:#24242a;bg:#d9d1f5",
        selection_text="fg:#24242a;bg:#d9d1f5",
        input="fg:#24242a",
        placeholder="fg:#9a96a4",
        cursor="reverse",
        status_bar="fg:#3a3743;bg:#e4e1ec",
        tab_active="fg:#ffffff;bg:#6d5ae6;bold",
        tab_inactive="fg:#77737f",
        header="fg:#3a3743;bg:#eceaf2;bold",
        footer="fg:#77737f",
        toast="fg:#24242a;bg:#fff2c2",
        overlay="fg:#24242a;bg:#f7f6fa",
        backdrop="dim",
        marker="fg:#6d3bd4;bold",
        separator="fg:#b9b4c4",
        zebra="fg:#77737f",
    )


DEFAULT_THEME_VALUE = dark_theme()


def default_theme():
    """Return a copy of the current package default theme."""
    return copy.copy(DEFAULT_THEME_VALUE)


def theme_by_name(name):
    """Resolve "dark"/"light" (empty means dark); returns ``(theme, found)``."""
    if name in ("", "dark"):
        return dark_theme(), True
    if name == "light":
        return light_theme(), True
    return Theme(), False


_FOREIGN_SOURCES = {
    "List": (".list",),
    "VirtualList": (".list",),
    "Table": (".table",),
    "TextInput": (".input", ".form"),
    "TextArea": (".input", ".form"),
    "Modal": (".modal",),
    "Menu": (".modal", ".menu"),
}


def _prefill(class_name, **fields):
    """Instantiate a sibling-module widget and set its themed fields."""
    for module_name in _FOREIGN_SOURCES[class_name]:
        try:
            module = importlib.import_module(module_name, __package__)
        except ImportError:
            continue
        widget = getattr(module, class_name)()
        for key, value in fields.items():
            setattr(widget, key, value)
        return widget
    raise ImportError(
        "theme.%s needs %s, owned by the content/form modules"
        % (class_name, "/".join(_FOREIGN_SOURCES[class_name])))


def themed_list(theme):
    """A List pre-filled with theme tokens."""
    return _prefill("List", style=theme.text, selected_style=theme.selection,
                    header_style=theme.header, footer_style=theme.footer,
                    empty_style=theme.muted, marker=theme.marker)


def themed_virtual_list(theme):
    """A VirtualList pre-filled with theme tokens."""
    return _prefill("VirtualList", style=theme.text,
                    selected_style=theme.selection,
                    disabled_style=theme.muted, header_style=theme.header,
                    footer_style=theme.footer, empty_style=theme.muted,
                    marker=theme.marker)


def themed_table(theme):
    """A Table pre-filled with theme tokens."""
    return _prefill("Table", style=theme.text, header_style=theme.header,
                    selected_style=theme.selection, zebra_style=theme.zebra,
                    footer_style=theme.footer,
                    separator_style=theme.separator)


def themed_text_input(theme):
    """A TextInput pre-filled with theme tokens."""
    return _prefill("TextInput", style=theme.input,
                    placeholder_style=theme.placeholder,
                    cursor_style=theme.cursor)


def themed_text_area(theme):
    """A TextArea pre-filled with theme tokens."""
    return _prefill("TextArea", style=theme.input,
                    placeholder_style=theme.placeholder,
                    cursor_style=theme.cursor)


def themed_modal(theme):
    """A Modal pre-filled with theme tokens."""
    return _prefill("Modal", style=theme.border_focus,
                    backdrop_style=theme.backdrop)


def themed_menu(theme):
    """A Menu pre-filled with theme tokens."""
    return _prefill("Menu", style=theme.text,
                    selected_style=theme.selection,
                    disabled_style=theme.muted,
                    separator_style=theme.separator,
                    frame_style=theme.border, marker=theme.marker)


def themed_progress_bar(theme):
    """A ProgressBar pre-filled with theme tokens."""
    return ProgressBar(style=theme.accent, track_style=theme.muted,
                       label_style=theme.text, percent_style=theme.muted)


def themed_spinner(theme):
    """A Spinner pre-filled with theme tokens."""
    return Spinner(style=theme.accent, label_style=theme.text)


def themed_badge(theme):
    """A Badge pre-filled with theme tokens."""
    return Badge(style=theme.accent)


def themed_tags(theme):
    """A Tags pre-filled with theme tokens."""
    return Tags(style=theme.text, sep_style=theme.separator)

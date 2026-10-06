"""Host style token names and raw-style string helpers.

This module names the host style tokens (render/style.go) as exported string
constants and adds tiny helpers that compose raw explicit-style strings
("fg:#RRGGBB;bg:#RRGGBB;bold") idempotently. A widget can pass either a token
or a raw string to :meth:`tui2sdk.builder.Node.style`: the host resolves tokens
against its palette and translates raw strings verbatim.

The names mirror the ``render.Token`` values exactly; widgets must not import
the host render package (the protocol is the only contract), so the values are
repeated here as plain strings.
"""

STYLE_DEFAULT = "default"
STYLE_BACKGROUND = "bg"
STYLE_FG = "fg"
STYLE_FOREGROUND = "foreground"
STYLE_STRONG_FOREGROUND = "strong-foreground"
STYLE_MUTED = "muted"
STYLE_ACCENT = "accent"
STYLE_ACCENT_DIM = "accent_dim"
STYLE_SUCCESS = "success"
STYLE_OK = "ok"
STYLE_WARNING = "warning"
STYLE_DANGER = "danger"
STYLE_INFO = "info"
STYLE_CHROME = "chrome"
STYLE_CHROME_FOCUS = "chrome_focus"
STYLE_HEADER = "header"
STYLE_TAB_ACTIVE = "tab_active"
STYLE_TAB_INACTIVE = "tab_inactive"
STYLE_FOOTER = "footer"
STYLE_FOOTER_ACCENT = "footer-accent"
STYLE_STATUS = "status"
STYLE_OVERLAY = "overlay"
STYLE_BORDER = "border"
STYLE_BORDER_FOCUS = "border_focus"
STYLE_BORDER_DEAD = "border_dead"
STYLE_ACTIVE_BORDER = "active-border"
STYLE_INACTIVE_BORDER = "inactive-border"
STYLE_SELECTION = "selection"


def _style_has_segment(style, segment):
    """Report whether ``style`` already carries ``segment`` as a piece."""
    if not style:
        return False
    return any(part.strip() == segment for part in style.split(";"))


def with_bold(style):
    """Append the bold attribute unless the style already carries it.

    ``with_bold("")`` is ``"bold"``; repeated calls are idempotent.
    """
    if _style_has_segment(style, "bold"):
        return style
    if not style:
        return "bold"
    return style + ";bold"


def with_underline(style):
    """Append underline unless present; ``with_underline("")`` is "underline"."""
    if _style_has_segment(style, "underline"):
        return style
    if not style:
        return "underline"
    return style + ";underline"


def with_reverse(style):
    """Append reverse unless present; ``with_reverse("")`` is "reverse"."""
    if _style_has_segment(style, "reverse"):
        return style
    if not style:
        return "reverse"
    return style + ";reverse"


def _set_style_color(style, key, hex_color):
    """Replace or insert one ``key:hex`` color segment."""
    if not hex_color:
        return style
    segment = key + ":" + hex_color
    parts = []
    replaced = False
    for raw in style.split(";"):
        part = raw.strip()
        if not part:
            continue
        if part.startswith(key + ":"):
            if replaced:
                continue
            parts.append(segment)
            replaced = True
            continue
        parts.append(part)
    if not replaced:
        parts.insert(0, segment)
    return ";".join(parts)


def with_fg(style, hex_color):
    """Set the foreground color of a raw style string.

    An existing ``fg:`` segment is replaced in place, otherwise the color is
    prepended; an empty hex leaves the style untouched. ``with_fg("",
    "#aabbcc")`` is ``"fg:#aabbcc"`` and repeated calls with the same hex are
    idempotent.
    """
    return _set_style_color(style, "fg", hex_color)


def with_bg(style, hex_color):
    """Set the background color of a raw style string.

    Replaces an existing ``bg:`` segment or prepends it. An empty hex leaves
    the style untouched; repeated calls with the same hex are idempotent.
    """
    return _set_style_color(style, "bg", hex_color)

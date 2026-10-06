"""Box-drawing glyph sets and the program-side bordered box.

The border glyphs and the title are drawn program-side as styled text rows.
A :class:`BorderBox` declares width x height; a zero geometry falls back to
the child's measured content or a sensible minimum. Boxes too small for chrome
degrade to the bare child.
"""

from .. import builder
from .tokens import STYLE_BORDER, STYLE_STRONG_FOREGROUND


class BorderSet:
    """A box-drawing glyph set.

    Every field is one glyph except ``top``, ``bottom``, ``left`` and
    ``right``, which are the repeated edge pieces.
    """

    def __init__(self, top_left="", top="", top_right="", left="", right="",
                 bottom_left="", bottom="", bottom_right=""):
        self.top_left = top_left
        self.top = top
        self.top_right = top_right
        self.left = left
        self.right = right
        self.bottom_left = bottom_left
        self.bottom = bottom
        self.bottom_right = bottom_right

    def _key(self):
        return (self.top_left, self.top, self.top_right, self.left,
                self.right, self.bottom_left, self.bottom, self.bottom_right)

    def __eq__(self, other):
        if not isinstance(other, BorderSet):
            return NotImplemented
        return self._key() == other._key()

    def __repr__(self):
        return "BorderSet(%r)" % (self._key(),)


def border_normal():
    """The default single-line box set (┌─┐│└┘)."""
    return BorderSet(
        top_left="┌", top="─", top_right="┐",
        left="│", right="│",
        bottom_left="└", bottom="─", bottom_right="┘",
    )


def border_rounded():
    """The rounded-corner set (╭─╮│╰╯)."""
    return BorderSet(
        top_left="╭", top="─", top_right="╮",
        left="│", right="│",
        bottom_left="╰", bottom="─", bottom_right="╯",
    )


def border_thick():
    """The heavy set (┏━┓┃┗┛)."""
    return BorderSet(
        top_left="┏", top="━", top_right="┓",
        left="┃", right="┃",
        bottom_left="┗", bottom="━", bottom_right="┛",
    )


def border_double():
    """The double-line set (╔═╗║╚╝)."""
    return BorderSet(
        top_left="╔", top="═", top_right="╗",
        left="║", right="║",
        bottom_left="╚", bottom="═", bottom_right="╝",
    )


def border_set_or_default(border_set):
    """Return the given set when non-zero, else :func:`border_normal`."""
    if border_set is None or border_set == BorderSet():
        return border_normal()
    return border_set


class BorderBox:
    """A bordered box around an optional child builder."""

    def __init__(self, id="", title="", width=0, height=0, border=None,
                 style="", title_style="", child=None):
        self.id = id
        self.title = title
        self.width = width
        self.height = height
        self.border = border
        self.style = style
        self.title_style = title_style
        self.child = child

    def _border_style(self):
        if self.style:
            return self.style
        return STYLE_BORDER

    def build(self):
        """Return the bordered box as a box tree (:class:`~tui2sdk.builder.Node`)."""
        glyphs = border_set_or_default(self.border)
        style = self.style or STYLE_BORDER
        title_style = self.title_style or STYLE_STRONG_FOREGROUND

        width, height = self.width, self.height
        if width <= 0:
            width = _box_child_width(self.child) + 2
            if width < 2:
                width = 2
        if height <= 0:
            height = 2
            if self.child is not None:
                height += 1

        col = builder.box("col")
        if self.id:
            col.id(self.id)
        col.width(width).height(height)

        if width < 2 or height < 2:
            if self.child is not None:
                col.child(self.child)
            return col

        inner = width - 2
        col.child(self._top_edge(glyphs, title_style, inner))
        body = height - 2
        for index in range(body):
            line = builder.row().height(1)
            line.child(builder.text(glyphs.left).style(style).width(1))
            if self.child is not None and index == 0:
                line.child(self.child)
            else:
                line.child(builder.text(" " * inner).width(inner).flex(1))
            line.child(builder.text(glyphs.right).style(style).width(1))
            col.child(line)
        col.child(builder.text(_bottom_edge(glyphs, inner)).style(style)
                  .width(width).height(1))
        return col

    def _top_edge(self, glyphs, title_style, inner):
        line = builder.row().height(1)
        line.child(builder.text(glyphs.top_left).style(self._border_style()).width(1))
        if not self.title:
            line.child(builder.text(glyphs.top * inner)
                       .style(self._border_style()).width(inner).flex(1))
            line.child(builder.text(glyphs.top_right).style(self._border_style()).width(1))
            return line
        fill = inner - 1
        if fill < 0:
            fill = 0
        label = builder.truncate(" " + self.title + " ", fill + 1)
        label_width = builder.display_width(label)
        rest = inner - label_width
        if rest < 0:
            rest = 0
        line.child(builder.text(label).style(title_style))
        if rest > 0:
            line.child(builder.text(glyphs.top * rest)
                       .style(self._border_style()).width(rest).flex(1))
        line.child(builder.text(glyphs.top_right).style(self._border_style()).width(1))
        return line


def _bottom_edge(glyphs, inner):
    return glyphs.bottom_left + glyphs.bottom * inner + glyphs.bottom_right


def _box_child_width(child):
    """Measure a child's declared width, falling back to its content width."""
    if child is None:
        return 0
    node = child.build()
    size = node.get("size")
    if size and size[0] > 0:
        return size[0]
    content = node.get("content")
    if content is not None:
        width = 0
        if content.get("text"):
            width = builder.display_width(content["text"])
        for line in content.get("lines") or []:
            line_width = builder.display_width(line)
            if line_width > width:
                width = line_width
        return width
    return 0

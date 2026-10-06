"""Proportional scroll indicator (port of scrollbar.go).

Usable vertically (``height`` > 0) or horizontally (``width`` > 0). ``total``
is the item count, ``visible`` the window size in items and ``offset`` the
first visible item. The thumb size is ``visible*length/total`` with a minimum
of one cell; the thumb position maps ``offset`` onto ``[0, length-size]``.
When the whole content fits (``total <= visible``) the thumb fills the track.
"""

from .. import builder
from .list import first_non_empty

DEFAULT_SCROLLBAR_TRACK_STYLE = "muted"
DEFAULT_SCROLLBAR_THUMB_STYLE = "accent"
DEFAULT_SCROLLBAR_UP = "\u2191"
DEFAULT_SCROLLBAR_DOWN = "\u2193"
SCROLLBAR_TRACK_VERTICAL = "\u2502"
SCROLLBAR_THUMB_VERTICAL = "\u2503"
SCROLLBAR_TRACK_HORIZONTAL = "\u2500"
SCROLLBAR_THUMB_HORIZONTAL = "\u2501"


class Scrollbar:
    """A pure proportional scroll indicator."""

    def __init__(self, id="", total=0, visible=0, offset=0, height=0,
                 width=0, vertical=False, track_style="", thumb_style="",
                 up="", down=""):
        self.id = id
        self.total = total
        self.visible = visible
        self.offset = offset
        self.height = height
        self.width = width
        self.vertical = vertical
        self.track_style = track_style
        self.thumb_style = thumb_style
        self.up = up
        self.down = down

    def is_vertical(self):
        """The resolved orientation: an explicit ``vertical`` wins, otherwise
        a height means vertical and a width means horizontal."""
        if self.vertical:
            return True
        if self.width > 0 and self.height <= 0:
            return False
        return True

    def track_length(self):
        """The number of cells of the track (height when vertical, width when
        horizontal)."""
        if self.is_vertical():
            if self.height > 0:
                return self.height
            return 0
        if self.width > 0:
            return self.width
        return 0

    def _thumb_max_offset(self):
        maximum = self.total - self.visible
        if maximum < 0:
            maximum = 0
        return maximum

    def _clamp_offset(self):
        offset = self.offset
        if offset < 0:
            offset = 0
        maximum = self._thumb_max_offset()
        if offset > maximum:
            offset = maximum
        return offset

    def thumb(self):
        """The thumb start cell and size along the track. The size is at least
        one cell and never exceeds the track; a zero-length track yields
        ``(0, 0)``."""
        length = self.track_length()
        if length <= 0:
            return 0, 0
        if self.total <= 0 or self.total <= self.visible:
            return 0, length
        size = self.visible * length // self.total
        if size < 1:
            size = 1
        if size > length:
            size = length
        start = 0
        maximum = self._thumb_max_offset()
        if maximum > 0:
            start = self._clamp_offset() * (length - size) // maximum
        if start < 0:
            start = 0
        max_start = length - size
        if start > max_start:
            start = max_start
        return start, size

    def offset_at(self, pos):
        """Map a click position (cells from the start of the track, 0-based)
        to an offset, centering the thumb on the click. Degenerate scrollbars
        return 0."""
        length = self.track_length()
        if length <= 0:
            return 0
        maximum = self._thumb_max_offset()
        if maximum <= 0:
            return 0
        _, size = self.thumb()
        denom = length - size
        if denom <= 0:
            return 0
        offset = (pos - size // 2) * maximum // denom
        if offset < 0:
            return 0
        if offset > maximum:
            return maximum
        return offset

    def build(self):
        """The track with the proportional thumb. ``up``/``down`` are optional
        one-cell end caps drawn outside the measured track."""
        length = self.track_length()
        start, size = self.thumb()
        track_style = first_non_empty(self.track_style, DEFAULT_SCROLLBAR_TRACK_STYLE)
        thumb_style = first_non_empty(self.thumb_style, DEFAULT_SCROLLBAR_THUMB_STYLE)
        track_glyph, thumb_glyph = self._glyphs()

        def cell(i):
            if start <= i < start + size:
                return thumb_glyph, thumb_style
            return track_glyph, track_style

        if self.is_vertical():
            col = builder.box("col").width(1)
            if self.id:
                col.id(self.id)
            col.input("mouse")
            col.height(length + self._cap_count())
            if self.up:
                col.child(builder.text(self.up).style(track_style).width(1).height(1))
            for i in range(length):
                glyph, style = cell(i)
                col.child(builder.text(glyph).style(style).width(1).height(1))
            if self.down:
                col.child(builder.text(self.down).style(track_style).width(1).height(1))
            return col
        row = builder.box("row").height(1)
        if self.id:
            row.id(self.id)
        row.input("mouse")
        row.width(length + self._cap_count())
        if self.up:
            row.child(builder.text(self.up).style(track_style).width(1).height(1))
        for i in range(length):
            glyph, style = cell(i)
            row.child(builder.text(glyph).style(style).width(1).height(1))
        if self.down:
            row.child(builder.text(self.down).style(track_style).width(1).height(1))
        return row

    def _cap_count(self):
        count = 0
        if self.up:
            count += 1
        if self.down:
            count += 1
        return count

    def _glyphs(self):
        if self.is_vertical():
            return SCROLLBAR_TRACK_VERTICAL, SCROLLBAR_THUMB_VERTICAL
        return SCROLLBAR_TRACK_HORIZONTAL, SCROLLBAR_THUMB_HORIZONTAL

"""Progress, spinner and label widgets.

None of these hold a timer: the caller advances ``value``/``frame`` from its
own tick and commits the view.
"""

from .. import builder

DEFAULT_PROGRESS_FULL = "█"
DEFAULT_PROGRESS_EMPTY = "░"
DEFAULT_PROGRESS_WIDTH = 20
DEFAULT_TABLE_SEPARATOR = " "

SPINNER_FRAMES = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"]


class ProgressBar:
    """A fixed-width bar from value/max plus an optional label and percentage."""

    def __init__(self, id="", value=0, max=0, width=0, label="",
                 hide_percent=False, full="", empty="", style="",
                 track_style="", label_style="", percent_style=""):
        self.id = id
        self.value = value
        self.max = max
        self.width = width
        self.label = label
        self.hide_percent = hide_percent
        self.full = full
        self.empty = empty
        self.style = style
        self.track_style = track_style
        self.label_style = label_style
        self.percent_style = percent_style

    def fraction(self):
        """Return value/max clamped to [0, 1]; a non-positive max is 0."""
        if self.max <= 0:
            return 0
        value = self.value
        if value < 0:
            value = 0
        if value > self.max:
            value = self.max
        return value / self.max

    def percent(self):
        """Return the rounded percentage in [0, 100]."""
        return int(self.fraction() * 100 + 0.5)

    def bar_text(self):
        """Return the full bar string (filled + empty glyphs)."""
        width = self._bar_width()
        filled = int(self.fraction() * width + 0.5)
        if filled > width:
            filled = width
        return self._full_glyph() * filled + self._empty_glyph() * (width - filled)

    def _bar_width(self):
        if self.width > 0:
            return self.width
        return DEFAULT_PROGRESS_WIDTH

    def _full_glyph(self):
        return self.full or DEFAULT_PROGRESS_FULL

    def _empty_glyph(self):
        return self.empty or DEFAULT_PROGRESS_EMPTY

    def build(self):
        """Return the bar as a one-row box tree: label, fill, track, percent."""
        row = builder.row().height(1)
        if self.id:
            row.id(self.id)
        if self.label:
            row.child(builder.text(self.label).style(self.label_style).height(1))
        width = self._bar_width()
        filled = int(self.fraction() * width + 0.5)
        if filled > width:
            filled = width
        if filled > 0:
            row.child(builder.text(self._full_glyph() * filled)
                      .style(self.style).height(1))
        empty = width - filled
        if empty > 0:
            row.child(builder.text(self._empty_glyph() * empty)
                      .style(self.track_style).height(1))
        if not self.hide_percent:
            row.child(builder.text(str(self.percent()) + "%")
                      .style(self.percent_style).height(1))
        return row


class Spinner:
    """One frame of an animation; the caller advances ``frame`` itself."""

    def __init__(self, id="", frame=0, frames=None, style="", label="",
                 label_style=""):
        self.id = id
        self.frame = frame
        self.frames = list(frames) if frames else []
        self.style = style
        self.label = label
        self.label_style = label_style

    def frame_text(self):
        """Return the current frame; ``frame`` wraps for negative values."""
        frames = self.frames or SPINNER_FRAMES
        if not frames:
            return ""
        return frames[self.frame % len(frames)]

    def build(self):
        """Return the spinner and its optional label as one row."""
        row = builder.row().height(1)
        if self.id:
            row.id(self.id)
        row.child(builder.text(self.frame_text()).style(self.style).height(1))
        if self.label:
            row.child(builder.text(self.label).style(self.label_style).height(1))
        return row


class Badge:
    """A small styled label (status chip, count, key marker)."""

    def __init__(self, id="", text="", style="", input=None):
        self.id = id
        self.text = text
        self.style = style
        self.input = list(input or [])

    def build(self):
        """Return the badge as one text box."""
        box = builder.text(self.text).height(1)
        if self.id:
            box.id(self.id)
        if self.style:
            box.style(self.style)
        if self.input:
            box.input(*self.input)
        return box


class Tag:
    """One item of :class:`Tags`."""

    def __init__(self, text="", id="", style="", input=None):
        self.text = text
        self.id = id
        self.style = style
        self.input = list(input or [])


class Tags:
    """A row of tagged labels separated by ``separator``."""

    def __init__(self, id="", items=None, separator="", sep_style="",
                 style=""):
        self.id = id
        self.items = list(items or [])
        self.separator = separator
        self.sep_style = sep_style
        self.style = style

    def build(self):
        """Return the tags as a one-row box tree."""
        row = builder.row().height(1)
        if self.id:
            row.id(self.id)
        separator = self.separator or DEFAULT_TABLE_SEPARATOR
        for index, tag in enumerate(self.items):
            if index > 0:
                row.child(builder.text(separator).style(self.sep_style))
            box = builder.text(tag.text).height(1)
            style = tag.style or self.style
            if style:
                box.style(style)
            if tag.id:
                box.id(tag.id)
            if tag.input:
                box.input(*tag.input)
            row.child(box)
        return row

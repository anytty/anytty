"""Pure builder widgets: tab bar, status bar, frame, divider, card, button.

Widgets hold no state and never emit protocol methods; clicks and keys are
handled by the caller after hit-testing the box id, exactly like every other
box. Every widget takes its colors from style token names, so the host palette
stays the single source of ANSI resolution.
"""

from .. import builder

DEFAULT_ACTIVE_STYLE = "tab_active"
DEFAULT_INACTIVE_STYLE = "tab_inactive"
DEFAULT_CHROME_STYLE = "chrome"
DEFAULT_STATUS_STYLE = "status"
DEFAULT_SEP_STYLE = "muted"
DEFAULT_KEY_STYLE = "muted"
DEFAULT_BORDER_STYLE = "border"
DEFAULT_BUTTON_STYLE = "accent"
DEFAULT_SEPARATOR = "│"


class Segment:
    """One styled text run of a bar."""

    def __init__(self, text="", style="", id="", input=None):
        self.text = text
        self.style = style
        self.id = id
        self.input = list(input or [])

    def clone(self):
        return Segment(self.text, self.style, self.id, self.input)


class TabItem:
    """One tab: the hit-test id, the visible title, whether it is active."""

    def __init__(self, id="", title="", active=False):
        self.id = id
        self.title = title
        self.active = active


class TabBar:
    """The top tab strip: optional left segment, one box per tab, optional +."""

    def __init__(self, left=None, items=None, plus=False, plus_id="",
                 plus_text="", active_style="", inactive_style="",
                 chrome_style=""):
        self.left = left
        self.items = list(items or [])
        self.plus = plus
        self.plus_id = plus_id
        self.plus_text = plus_text
        self.active_style = active_style
        self.inactive_style = inactive_style
        self.chrome_style = chrome_style

    def _active_style(self):
        return self.active_style or DEFAULT_ACTIVE_STYLE

    def _inactive_style(self):
        return self.inactive_style or DEFAULT_INACTIVE_STYLE

    def _chrome_style(self):
        return self.chrome_style or DEFAULT_CHROME_STYLE

    def build(self):
        """Return the tab strip as a one-row box tree."""
        row = builder.row()
        if self.left is not None:
            row.child(_segment_box(self.left, self._chrome_style()))
        for item in self.items:
            label = " " + item.title + " "
            style = self._inactive_style()
            if item.active:
                label = "[" + item.title + "]"
                style = self._active_style()
            row.child(builder.text(label).id(item.id).style(style).input("mouse"))
        if self.plus:
            node_id = self.plus_id or "tab:new"
            label = self.plus_text or " + "
            row.child(builder.text(label).id(node_id)
                      .style(self._chrome_style()).input("mouse"))
        return row


class StatusBar:
    """A one-row status line from styled segments.

    With ``width > 0`` the right group is pushed to the right edge and the
    left group is trimmed (last segment first, then text) until the row fits.
    """

    def __init__(self, left=None, right=None, width=0, separator="",
                 sep_style="", id=""):
        self.left = list(left or [])
        self.right = list(right or [])
        self.width = width
        self.separator = separator
        self.sep_style = sep_style
        self.id = id

    def text(self):
        """Render the segments as one plain string for measurement."""
        return self.separator_text().join(self._segment_texts())

    def _segment_texts(self):
        return [seg.text for seg in self.left] + [seg.text for seg in self.right]

    def separator_text(self):
        if self.separator:
            return " " + self.separator + " "
        return " " + DEFAULT_SEPARATOR + " "

    def _sep_style(self):
        return self.sep_style or DEFAULT_SEP_STYLE

    def build(self):
        """Return the status bar as a row of text boxes."""
        row = builder.row().height(1)
        if self.id:
            row.id(self.id)
        left, right = self.left, self.right
        sep_text = self.separator_text()
        sep_width = builder.display_width(sep_text)
        if self.width > 0:
            right, _ = _fit_segments(right, self.width, sep_width)
            right_width = _group_width(right, sep_width)
            left_budget = 0
            if left:
                left_budget = self.width - right_width
                if right:
                    left_budget -= sep_width
            left, _ = _fit_segments(left, left_budget, sep_width)
        used = _group_width(left, sep_width) + _group_width(right, sep_width)
        if left and right:
            used += sep_width
        _append_group(row, left, self._sep_style(), sep_text)
        if left and right:
            row.child(builder.text(sep_text).style(self._sep_style()))
        if self.width > 0:
            pad = self.width - used
            if pad > 0:
                row.child(builder.text(" " * pad))
        _append_group(row, right, self._sep_style(), sep_text)
        return row


def _segment_texts(segs):
    """The per-segment texts (Go ``segmentTexts``), for callers that join."""
    return [seg.text for seg in segs]


def _segments_text(segs):
    """The concatenated texts (Go ``segmentsText``), for measurement."""
    return "".join(seg.text for seg in segs)


def _segments_width(segs):
    return sum(builder.display_width(seg.text) for seg in segs)


def _append_group(row, segs, sep_style, sep_text):
    for index, seg in enumerate(segs):
        if index > 0:
            row.child(builder.text(sep_text).style(sep_style))
        row.child(_segment_box(seg, ""))


def _segment_box(seg, fallback_style):
    box = builder.text(seg.text)
    if seg.id:
        box.id(seg.id)
    if seg.style:
        box.style(seg.style)
    elif fallback_style:
        box.style(fallback_style)
    if seg.input:
        box.input(*seg.input)
    return box


def _group_width(segs, sep_width):
    width = 0
    for index, seg in enumerate(segs):
        if index > 0:
            width += sep_width
        width += builder.display_width(seg.text)
    return width


def _fit_segments(segs, budget, sep_width):
    """Trim segs to budget cells: trailing segments first, then last text."""
    if budget <= 0:
        return [], 0
    out = [seg.clone() for seg in segs]
    while out:
        width = _group_width(out, sep_width)
        if width <= budget:
            return out, width
        if len(out) == 1:
            out[0].text = builder.truncate(out[0].text, budget)
            return out, builder.display_width(out[0].text)
        out.pop()
    return [], 0


class FrameRow:
    """One content row of a Frame."""

    def __init__(self, text="", style="", id="", input=None):
        self.text = text
        self.style = style
        self.id = id
        self.input = list(input or [])


class Frame:
    """A program-side bordered panel drawn as styled text rows.

    Width/height <= 0 fall back to the row content size; boxes too small for
    chrome degrade to plain rows.
    """

    def __init__(self, id="", title="", width=0, height=0, style="",
                 rows=None, fill_style=""):
        self.id = id
        self.title = title
        self.width = width
        self.height = height
        self.style = style
        self.rows = list(rows or [])
        self.fill_style = fill_style

    def build(self):
        """Return the framed panel as a box tree."""
        width, height = self.width, self.height
        if width <= 0:
            width = _frame_text_width(self.rows) + 2
        if height <= 0:
            height = len(self.rows) + 2
        style = self.style or DEFAULT_BORDER_STYLE
        col = builder.box("col").id(self.id).width(width).height(height)
        if width < 3 or height < 2:
            for row in self.rows:
                col.child(_frame_row_box(row, _max_int(1, width), ""))
            return col
        inner = width - 2
        col.child(builder.text(_rule_text(width, self.title)).style(style)
                  .width(width).height(1))
        for index in range(height - 2):
            row = FrameRow(style=self.fill_style)
            if index < len(self.rows):
                row = self.rows[index]
            line = builder.row().height(1)
            line.child(builder.text("│").style(style).width(1))
            line.child(_frame_row_box(row, inner, ""))
            line.child(builder.text("│").style(style).width(1))
            col.child(line)
        col.child(builder.text(_bottom_rule_text(width)).style(style)
                  .width(width).height(1))
        return col


def _frame_row_box(row, width, fallback_style):
    style = row.style or fallback_style
    box = builder.text(_pad_to(row.text, width)).style(style).width(width).height(1)
    if row.id:
        box.id(row.id)
    if row.input:
        box.input(*row.input)
    return box


def _frame_text_width(rows):
    width = 0
    for row in rows:
        row_width = builder.display_width(row.text)
        if row_width > width:
            width = row_width
    return width


def _rule_text(width, title):
    """Build a top rule of exactly width cells with an optional title."""
    inner = width - 2
    if inner <= 0:
        return "┌┐"
    if not title:
        return "┌" + "─" * inner + "┐"
    label = " " + title + " "
    budget = inner - 1
    if budget >= 0:
        label = builder.truncate(label, budget)
    else:
        label = ""
    fill = inner - 1 - builder.display_width(label)
    if fill < 0:
        fill = 0
    return "┌─" + label + "─" * fill + "┐"


def _bottom_rule_text(width):
    if width < 2:
        return "└"
    return "└" + "─" * (width - 2) + "┘"


def _pad_to(text, width):
    text = builder.truncate(text, width)
    pad = width - builder.display_width(text)
    if pad > 0:
        text += " " * pad
    return text


def _max_int(a, b):
    return a if a > b else b


class Divider:
    """A one-cell rule used as a pane gutter: vertical or horizontal run."""

    def __init__(self, id="", vertical=False, length=1, style="", input=None):
        self.id = id
        self.vertical = vertical
        self.length = length
        self.style = style
        self.input = list(input or [])

    def build(self):
        """Return the divider box."""
        glyph, width, height = "─", self.length, 1
        if self.vertical:
            glyph, width, height = "│", 1, self.length
        if width <= 0:
            width = 1
        if height <= 0:
            height = 1
        style = self.style or DEFAULT_SEP_STYLE
        box = (builder.text(glyph * _max_int(1, self.length)).id(self.id)
               .width(width).height(height).style(style))
        if self.input:
            box.input(*self.input)
        return box


class Card:
    """A framed text box, optionally centering its lines in the geometry."""

    def __init__(self, id="", title="", lines=None, width=0, height=0,
                 style="", line_style="", center=False):
        self.id = id
        self.title = title
        self.lines = list(lines or [])
        self.width = width
        self.height = height
        self.style = style
        self.line_style = line_style
        self.center = center

    def build(self):
        """Return the card box."""
        lines = list(self.lines)
        if self.center:
            lines = _center_lines(lines, self.width, self.height)
        rows = [FrameRow(text=line, style=self.line_style) for line in lines]
        frame = Frame(
            id=self.id,
            title=self.title,
            width=self.width,
            height=self.height,
            style=self.style,
            rows=rows,
        )
        return frame.build()


def _center_lines(lines, width, height):
    if width > 2:
        inner = width - 2
        for index, line in enumerate(lines):
            pad = (inner - builder.display_width(line)) // 2
            if pad > 0:
                lines[index] = " " * pad + line
    if height > 2:
        inner = height - 2
        pad = (inner - len(lines)) // 2
        if pad > 0:
            lines = [""] * pad + lines
    return lines


class Button:
    """A clickable text box (a box plus program-side hit handling)."""

    def __init__(self, id="", text="", hot="", style="", input=None):
        self.id = id
        self.text = text
        self.hot = hot
        self.style = style
        self.input = list(input or [])

    def line(self):
        """Render the button label as plain text for measurement."""
        return self.text

    def build(self):
        """Return the button box (a row when a hotkey run is split out)."""
        inputs = self.input or ["mouse"]
        style = self.style or DEFAULT_BUTTON_STYLE
        if not self.hot:
            return (builder.text(self.text).id(self.id).style(style)
                    .input(*inputs))
        index = self.text.find(self.hot)
        if index < 0:
            return (builder.text(self.text).id(self.id).style(style)
                    .input(*inputs))
        row = builder.row().id(self.id).input(*inputs)
        pre = self.text[:index]
        post = self.text[index + len(self.hot):]
        if pre:
            row.child(builder.text(pre).style(style))
        row.child(builder.text(self.hot).style(DEFAULT_BUTTON_STYLE))
        if post:
            row.child(builder.text(post).style(style))
        return row


class KeyHint:
    """The left footer group: the current mode plus its bindings."""

    def __init__(self, mode="", keys=None, mode_style="", key_style="",
                 sep_style="", id=""):
        self.mode = mode
        self.keys = list(keys or [])
        self.mode_style = mode_style
        self.key_style = key_style
        self.sep_style = sep_style
        self.id = id

    def text(self):
        """Render the hint as one plain string for measurement."""
        keys = " · ".join(self.keys)
        if not self.mode:
            return keys
        if not keys:
            return self.mode
        return self.mode + " " + DEFAULT_SEPARATOR + " " + keys

    def build(self):
        """Return the hint as a row of styled text boxes."""
        row = builder.row().height(1)
        if self.id:
            row.id(self.id)
        mode_style = self.mode_style or DEFAULT_STATUS_STYLE
        key_style = self.key_style or DEFAULT_KEY_STYLE
        sep_style = self.sep_style or DEFAULT_SEP_STYLE
        if self.mode:
            row.child(builder.text(self.mode).style(mode_style))
        if self.keys:
            if self.mode:
                row.child(builder.text(" " + DEFAULT_SEPARATOR + " ")
                          .style(sep_style))
            row.child(builder.text(" · ".join(self.keys)).style(key_style))
        return row

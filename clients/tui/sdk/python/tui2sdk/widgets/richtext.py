"""Styled rich text, wrapping and emphasis parsing (port of richtext.go).

A :class:`RichText` is a single row of styled :class:`Span` runs. ``build()``
returns a composable ``builder.Node`` (one text box per merged run, with a
declared width padded by a trailing unstyled box). Wrapping is CJK/wide-char
aware: a wide rune is never split across lines.
"""

from .. import builder


class Span:
    """One styled run of rich text. ``style`` is an opaque style string: a
    host token name or a raw style such as
    "fg:#RRGGBB;bg:#RRGGBB;bold;underline"."""

    __slots__ = ("text", "style")

    def __init__(self, text="", style=""):
        self.text = text
        self.style = style

    def __eq__(self, other):
        if not isinstance(other, Span):
            return NotImplemented
        return self.text == other.text and self.style == other.style

    def __repr__(self):
        return "Span(text=%r, style=%r)" % (self.text, self.style)


class RichText:
    """A single row of styled runs. ``width`` > 0 declares the row width (the
    row is padded so it occupies ``width`` cells). ``wrap`` is advisory
    metadata for callers that reflow: ``build()`` renders the spans as one
    row."""

    __slots__ = ("spans", "width", "wrap")

    def __init__(self, spans=None, width=0, wrap=False):
        self.spans = list(spans or [])
        self.width = width
        self.wrap = wrap

    def text(self):
        """The plain text of the spans with no styling, for measurement."""
        return "".join(span.text for span in self.spans)

    def build(self):
        """Render the spans as a one-row box tree: one text box per run,
        adjacent runs sharing a style merged, empty runs dropped. A declared
        width pads the row with a trailing unstyled box; an empty RichText
        still yields a row."""
        row = builder.row().height(1)
        used = 0
        for span in _merge_spans(self.spans):
            box = builder.text(span.text)
            if span.style:
                box.style(span.style)
            row.child(box)
            used += builder.display_width(span.text)
        pad = self.width - used
        if pad > 0:
            row.child(builder.text(" " * pad))
        return row

    def styled(self, style):
        """The RichText re-styled as a single span."""
        return RichText([Span(self.text(), style)], width=self.width, wrap=self.wrap)

    def wrap_spans(self, width):
        """Wrap the RichText to width and mark every resulting row wrapped."""
        wrapped = wrap_spans(self.spans, width)
        if wrapped is None:
            return None
        for row in wrapped:
            row.wrap = True
        return wrapped


def styled(text, style):
    """A Span of text rendered with style."""
    return Span(text, style)


def line(*spans):
    """Variadic constructor of a RichText from spans."""
    return RichText(list(spans))


def _merge_spans(spans):
    """Coalesce adjacent runs with the same style and drop empty runs."""
    out = []
    for span in spans:
        if span.text == "":
            continue
        if out and out[-1].style == span.style:
            out[-1].text += span.text
            continue
        out.append(Span(span.text, span.style))
    return out


def wrap_text(s, width):
    """Hard-wrap s into lines of at most width display cells. Breaks on spaces
    when possible, drops the break space, hard-breaks long words. Embedded
    newlines force a line break. A width <= 0 yields None."""
    if width <= 0:
        return None
    runes = list(s)
    index_lines = _wrap_indexes(runes, width)
    return ["".join(runes[i] for i in line_indexes)
            for line_indexes in index_lines]


def wrap_spans(spans, width):
    """Wrap styled runs into lines of at most width display cells. Runs are
    split at wrap boundaries with their style preserved; a wide rune is never
    split. A width <= 0 yields None."""
    if width <= 0:
        return None
    runes = []
    styles = []
    for span in spans:
        for ch in span.text:
            runes.append(ch)
            styles.append(span.style)
    out = []
    for line_indexes in _wrap_indexes(runes, width):
        out.append(RichText(_group_runes(runes, styles, line_indexes), width=width))
    return out


def _group_runes(runes, styles, indexes):
    """Build spans from the rune indexes of one wrapped line, merging
    adjacent runes that share a style."""
    out = []
    for i in indexes:
        if out and out[-1].style == styles[i]:
            out[-1].text += runes[i]
            continue
        out.append(Span(runes[i], styles[i]))
    return out


def _wrap_indexes(runes, width):
    """Shared wrap core: each line is a list of rune indexes into the source.
    Lines never exceed width cells; a wide rune that cannot fit on a non-empty
    line moves to the next one."""
    lines = []
    cur = []
    cur_width = 0

    def flush(force):
        nonlocal cur, cur_width
        while cur and runes[cur[-1]] == " ":
            cur.pop()
        if cur or force:
            lines.append(cur)
        cur = []
        cur_width = 0

    i = 0
    while i < len(runes):
        ch = runes[i]
        if ch == "\n":
            flush(True)
            i += 1
            continue
        if ch == " " and cur_width == 0:
            i += 1
            continue
        rw = builder.cell_width(ch)
        if cur_width + rw > width:
            if ch == " ":
                flush(False)
                i += 1
                continue
            last = _last_space_index(cur, runes)
            if last >= 0:
                tail = list(cur[last + 1:])
                cur = cur[:last]
                flush(False)
                tail_width = sum(builder.cell_width(runes[idx]) for idx in tail)
                if tail_width + rw <= width:
                    cur = tail
                    cur_width = tail_width
                    cur.append(i)
                    cur_width += rw
                    i += 1
                    continue
                cur = [i]
                cur_width = rw
                flush(False)
                i += 1
                continue
            flush(False)
        cur.append(i)
        cur_width += rw
        i += 1
    flush(not lines)
    return lines


def _last_space_index(line_indexes, runes):
    for i in range(len(line_indexes) - 1, -1, -1):
        if runes[line_indexes[i]] == " ":
            return i
    return -1


def parse_emphasis(s):
    """Parse a tiny emphasis grammar into spans: "**bold**" becomes bold,
    "_em_" becomes italic, plain text keeps the base style. Unclosed, nested
    or empty markers stay literal."""
    return parse_emphasis_styled(s, "")


def parse_emphasis_styled(s, base):
    """parse_emphasis with a base style composed onto every span; emphasis
    attributes are appended to it."""
    out = []
    plain = []

    def flush_plain():
        if plain:
            out.append(Span("".join(plain), base))
            plain.clear()

    i = 0
    n = len(s)
    while i < n:
        if s.startswith("**", i):
            end = s.find("**", i + 2)
            if end > i + 2:
                flush_plain()
                out.append(Span(s[i + 2:end], _with_bold(base)))
                i = end + 2
                continue
        if s[i] == "_":
            end = s.find("_", i + 1)
            if end > i + 1:
                flush_plain()
                out.append(Span(s[i + 1:end], with_italic(base)))
                i = end + 1
                continue
        plain.append(s[i])
        i += 1
    flush_plain()
    return out


def _style_has_segment(style, segment):
    """Whether style already carries segment as a semicolon-separated piece."""
    if not style:
        return False
    return any(part.strip() == segment for part in style.split(";"))


def _with_bold(style):
    """Append the bold attribute unless the style already carries it."""
    if _style_has_segment(style, "bold"):
        return style
    if style == "":
        return "bold"
    return style + ";bold"


def with_italic(style):
    """Append the italic attribute unless the style already carries it.
    with_italic("") is "italic"; repeated calls are idempotent."""
    if _style_has_segment(style, "italic"):
        return style
    if style == "":
        return "italic"
    return style + ";italic"

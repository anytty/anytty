"""Pure mouse helpers: hit regions, wheel folding, drag and click tracking
(port of mouse.go).

Everything here is a pure function over coordinates and plain values: no
timers, no threads, no protocol emission. ``ClickTracker`` consumes caller
supplied ``datetime`` values, the same way Go consumes ``time.Time``.
"""

from datetime import timedelta

# Cell height of one List/Table row region.
DEFAULT_ROW_HEIGHT = 1

# Maximum gap between two clicks of the same gesture (double/triple click).
DEFAULT_CLICK_THRESHOLD = timedelta(milliseconds=400)


class Rect:
    """A rectangle in viewport cells."""

    __slots__ = ("x", "y", "w", "h")

    def __init__(self, x=0, y=0, w=0, h=0):
        self.x = x
        self.y = y
        self.w = w
        self.h = h

    def __eq__(self, other):
        if not isinstance(other, Rect):
            return NotImplemented
        return (self.x, self.y, self.w, self.h) == (other.x, other.y, other.w, other.h)

    def __repr__(self):
        return "Rect(x=%d, y=%d, w=%d, h=%d)" % (self.x, self.y, self.w, self.h)


class HitRegion:
    """One rectangular hit-test target: the box id, its viewport rect and an
    opaque payload (row index, row/col pair, ...). A W or H <= 0 means the
    region is unbounded on that axis, so a row whose width was never declared
    stays clickable."""

    __slots__ = ("id", "x", "y", "w", "h", "data")

    def __init__(self, id="", x=0, y=0, w=0, h=0, data=None):
        self.id = id
        self.x = x
        self.y = y
        self.w = w
        self.h = h
        self.data = data

    def contains(self, x, y):
        """Whether (x, y) falls inside the region."""
        if self.w > 0 and (x < self.x or x >= self.x + self.w):
            return False
        if self.h > 0 and (y < self.y or y >= self.y + self.h):
            return False
        return True

    def rect(self):
        """The region bounds."""
        return Rect(self.x, self.y, self.w, self.h)

    def __repr__(self):
        return ("HitRegion(id=%r, x=%d, y=%d, w=%d, h=%d, data=%r)"
                % (self.id, self.x, self.y, self.w, self.h, self.data))


def hit(regions, x, y):
    """The topmost region containing (x, y): later regions win, so a caller
    orders regions bottom-to-top (the declaration/z order). Returns
    ``(region, True)`` or ``(None, False)``."""
    for region in reversed(regions):
        if region.contains(x, y):
            return region, True
    return None, False


def hit_id(regions, x, y):
    """hit()'s convenience form: ``(id, True)`` for the topmost hit,
    ``("", False)`` when nothing matches."""
    region, ok = hit(regions, x, y)
    if not ok:
        return "", False
    return region.id, True


def list_regions(list_, origin_x, origin_y, width):
    """One region per visible List row. ``origin_x``/``origin_y`` are the
    widget's top-left in the same coordinate space as the event; width <= 0
    leaves the row width unbounded. ``data`` is the absolute row index."""
    start, end = list_.visible_range()
    if end <= start:
        return None
    top = origin_y
    if list_.header:
        top += 1
    regions = []
    for row in range(start, end):
        row_id = ""
        if list_.row_id is not None:
            row_id = list_.row_id(row)
        regions.append(HitRegion(row_id, origin_x, top + (row - start), width,
                                 DEFAULT_ROW_HEIGHT, row))
    return regions


def virtual_list_regions(list_, origin_x, origin_y, width):
    """list_regions for a VirtualList; the row's own id wins and ``data`` is
    the absolute row index."""
    start, end = list_.visible_range()
    if end <= start:
        return None
    top = origin_y
    if list_.header:
        top += 1
    regions = []
    for row in range(start, end):
        regions.append(HitRegion(list_.rows[row].id, origin_x,
                                 top + (row - start), width,
                                 DEFAULT_ROW_HEIGHT, row))
    return regions


def table_regions(table, origin_x, origin_y):
    """One region per visible Table cell. ``data`` is a ``(row, col)`` pair.
    Cells are laid out by ``column_widths`` with the separator gutter between
    them; row -1 is the header and -2 the footer, so callers can ignore them
    by filtering on ``data``."""
    if not table.columns:
        return None
    widths = table.column_widths()
    sep = table._sep_width()
    top = origin_y
    if not table.hide_header:
        if table.rule:
            top += 2
        else:
            top += 1
    xs = []
    x = origin_x
    for col, width in enumerate(widths):
        if col > 0:
            x += sep
        xs.append(x)
        x += width
    regions = []
    for row in range(table.row_count()):
        cell_id = ""
        if table.row_id is not None:
            cell_id = table.row_id(row)
        for col, width in enumerate(widths):
            regions.append(HitRegion(cell_id, xs[col], top + row, width,
                                     DEFAULT_ROW_HEIGHT, (row, col)))
    return regions


def apply_wheel(offset, total, visible, delta):
    """Fold one wheel delta into an offset and clamp it to
    [0, total-visible]. Positive delta scrolls down/right. A fully visible
    (or empty) content set pins the offset to 0."""
    offset += delta
    if visible <= 0 or visible >= total:
        return 0
    return _clamp_offset(offset, visible, total)


def table_scroll(offset, total, visible, delta):
    """Table counterpart of the list wheel fold: Table has no window offset
    field, so the caller owns the offset and folds wheel deltas through here.
    ``visible`` is the number of on-screen data rows."""
    return apply_wheel(offset, total, visible, delta)


def _clamp_offset(offset, size, count):
    if size <= 0:
        return 0
    if offset < 0:
        return 0
    maximum = count - size
    if offset > maximum:
        return maximum
    return offset


class Drag:
    """A pure mouse drag selection model: begin on a press, update on each
    motion, end on release. Coordinates are absolute viewport cells; all
    methods are pointer-based and emit nothing."""

    __slots__ = ("active", "start_x", "start_y", "x", "y", "button",
                 "threshold_px")

    def __init__(self, active=False, start_x=0, start_y=0, x=0, y=0,
                 button="", threshold_px=0):
        self.active = active
        self.start_x = start_x
        self.start_y = start_y
        self.x = x
        self.y = y
        self.button = button
        self.threshold_px = threshold_px

    def begin(self, x, y, button):
        """Start a drag at (x, y) with the given button."""
        self.active = True
        self.start_x = x
        self.start_y = y
        self.x = x
        self.y = y
        self.button = button

    def update(self, x, y):
        """Move the drag's current position. A drag that was never begun is
        left inactive."""
        if not self.active:
            return
        self.x = x
        self.y = y

    def end(self):
        """Finish the drag and return ``(rect, True)`` plus whether a drag was
        active. The current position is not clamped to the start."""
        if not self.active:
            return None, False
        self.active = False
        return self._rect(), True

    def _rect(self):
        min_x, max_x = self.start_x, self.x
        if min_x > max_x:
            min_x, max_x = max_x, min_x
        min_y, max_y = self.start_y, self.y
        if min_y > max_y:
            min_y, max_y = max_y, min_y
        return Rect(min_x, min_y, max_x - min_x + 1, max_y - min_y + 1)

    def rect(self):
        """The drag selection normalized to a non-negative origin: x/y are the
        min corner, w/h the inclusive span (at least 1)."""
        return self._rect()


def select_range(a, b):
    """Normalize two positions into an inclusive (lo, hi) pair."""
    if a > b:
        return b, a
    return a, b


def list_select_range(list_, y0, y1):
    """Map two viewport y positions to the inclusive range of absolute List
    rows they span, clamped to the current window. Returns
    ``(first, last, True)`` or ``(0, 0, False)`` when neither position lands
    on a row."""
    lo, hi = select_range(y0, y1)
    first, ok = list_.row_at(lo)
    if not ok:
        first, ok = list_.row_at(hi)
        if not ok:
            return 0, 0, False
    last, ok = list_.row_at(hi)
    if not ok:
        last = first
    if first > last:
        first, last = last, first
    return first, last, True


def virtual_list_select_range(list_, y0, y1):
    """list_select_range for a VirtualList."""
    lo, hi = select_range(y0, y1)
    first, ok = list_.row_at(lo)
    if not ok:
        first, ok = list_.row_at(hi)
        if not ok:
            return 0, 0, False
    last, ok = list_.row_at(hi)
    if not ok:
        last = first
    if first > last:
        first, last = last, first
    return first, last, True


class ClickTracker:
    """Classify clicks into single/double/triple clicks purely from timing and
    position. Callers keep one tracker per target."""

    __slots__ = ("last_at", "last_x", "last_y", "threshold", "count")

    def __init__(self, last_at=None, last_x=0, last_y=0, threshold=None,
                 count=0):
        self.last_at = last_at
        self.last_x = last_x
        self.last_y = last_y
        self.threshold = threshold
        self.count = count

    def click(self, x, y, at):
        """Record a click at (x, y) happening at ``at`` and return the click
        count (1, 2 or 3). A gap longer than ``threshold`` or a move to
        another cell restarts the count; a fourth click starts a new
        gesture."""
        threshold = self.threshold
        if threshold is None or threshold <= timedelta(0):
            threshold = DEFAULT_CLICK_THRESHOLD
        same_cell = self.last_x == x and self.last_y == y
        close = (self.last_at is not None
                 and (at - self.last_at) <= threshold)
        if close and same_cell and 1 <= self.count < 3:
            self.count += 1
        else:
            self.count = 1
        self.last_at = at
        self.last_x = x
        self.last_y = y
        return self.count

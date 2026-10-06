"""Calendar dates, a Monday-first month grid and a date field (port of
datepicker.go).

``Date`` is a plain year/month/day triple, independent of the time package's
location and clock. Month is 1..12 and Day is 1..31; the zero Date is month 0
and is treated as "unset" by the calendar helpers. Calendar arithmetic is done
with the civil-days algorithms, never with the wall clock.
"""

import re

from .. import builder
from .form import Field
from .list import first_non_empty
from .table import pad_cell

# The wire/display layout accepted by parse_date.
DATE_LAYOUT = "%04d-%02d-%02d"

# Style of the cursor cell.
DEFAULT_CALENDAR_SELECTED_STYLE = "reverse"

# Error stored for a malformed or out-of-range date.
DATE_FIELD_MESSAGE = "invalid date"

# Monday-first to match Calendar.grid.
CALENDAR_WEEKDAYS = ["Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"]

_MONTH_NAMES = ["", "January", "February", "March", "April", "May", "June",
                "July", "August", "September", "October", "November",
                "December"]

_DATE_RE = re.compile(r"^[0-9]{4}-[0-9]{2}-[0-9]{2}$")


def is_leap_year(year):
    """Whether year is a Gregorian leap year."""
    return year % 4 == 0 and (year % 100 != 0 or year % 400 == 0)


def days_in_month(year, month):
    """The number of days in a month, or 0 when the month is out of range."""
    if month < 1 or month > 12:
        return 0
    if month in (4, 6, 9, 11):
        return 30
    if month == 2:
        if is_leap_year(year):
            return 29
        return 28
    return 31


def _days_from_civil(year, month, day):
    y = year - 1 if month <= 2 else year
    era = y // 400
    yoe = y - era * 400
    doy = (153 * (month + (-3 if month > 2 else 9)) + 2) // 5 + day - 1
    doe = yoe * 365 + yoe // 4 - yoe // 100 + doy
    return era * 146097 + doe - 719468


def _civil_from_days(z):
    z += 719468
    era = z // 146097
    doe = z - era * 146097
    yoe = (doe - doe // 1460 + doe // 36524 - doe // 146096) // 365
    y = yoe + era * 400
    doy = doe - (365 * yoe + yoe // 4 - yoe // 100)
    mp = (5 * doy + 2) // 153
    d = doy - (153 * mp + 2) // 5 + 1
    m = mp + (3 if mp < 10 else -9)
    return (y + (1 if m <= 2 else 0), m, d)


def _add_months(year, month, n):
    if month < 1 or month > 12:
        return year, month
    index = year * 12 + (month - 1) + n
    target_year = index // 12
    target_month = index % 12
    return target_year, target_month + 1


def _sign(n):
    if n < 0:
        return -1
    if n > 0:
        return 1
    return 0


class Date:
    """A calendar date: a plain year/month/day triple."""

    __slots__ = ("year", "month", "day")

    def __init__(self, year=0, month=0, day=0):
        self.year = year
        self.month = month
        self.day = day

    def __eq__(self, other):
        if not isinstance(other, Date):
            return NotImplemented
        return (self.year, self.month, self.day) == (other.year, other.month, other.day)

    def __hash__(self):
        return hash((self.year, self.month, self.day))

    def __repr__(self):
        return "Date(%d, %d, %d)" % (self.year, self.month, self.day)

    def __str__(self):
        if not self._valid():
            return ""
        return DATE_LAYOUT % (self.year, self.month, self.day)

    def _valid(self):
        return 1 <= self.month <= 12 and 1 <= self.day <= 31

    def is_leap(self):
        """Whether the date's year is a Gregorian leap year."""
        return is_leap_year(self.year)

    def days_in_month(self):
        """The number of days in the date's month, or 0 when the month is out
        of range."""
        return days_in_month(self.year, self.month)

    def weekday(self):
        """The day of week, where Sunday is 0 (Go's time.Weekday order)."""
        return (_days_from_civil(self.year, self.month, self.day) + 4) % 7

    def weekday_name(self):
        """The English weekday name, e.g. "Monday"."""
        return ("Sunday", "Monday", "Tuesday", "Wednesday", "Thursday",
                "Friday", "Saturday")[self.weekday()]

    def compare(self, other):
        """-1, 0 or 1 as the date sorts before, equal to, or after other."""
        if self.year != other.year:
            return _sign(self.year - other.year)
        if self.month != other.month:
            return _sign(self.month - other.month)
        if self.day != other.day:
            return _sign(self.day - other.day)
        return 0

    def equal(self, other):
        """Whether two dates fall on the same day."""
        return self.compare(other) == 0

    def before(self, other):
        """Whether the date sorts strictly before other."""
        return self.compare(other) < 0

    def after(self, other):
        """Whether the date sorts strictly after other."""
        return self.compare(other) > 0

    def add_days(self, n):
        """The date n days after this one (n may be negative), normalizing
        through the civil calendar so month and year boundaries carry."""
        if not self._valid():
            return self
        year, month, day = _civil_from_days(
            _days_from_civil(self.year, self.month, self.day) + n)
        return Date(year, month, day)

    def add_months(self, n):
        """The date n months after this one (n may be negative), clamping the
        day to the target month's length (2024-01-31 + 1 month =
        2024-02-29)."""
        if not self._valid():
            return self
        year, month = _add_months(self.year, self.month, n)
        day = self.day
        max_day = days_in_month(year, month)
        if day > max_day:
            day = max_day
        return Date(year, month, day)


def parse_date(s):
    """Parse "YYYY-MM-DD" and reject any value time would silently normalize
    (2024-02-30, 2023-13-01, trailing characters). Raises ValueError."""
    if not isinstance(s, str) or _DATE_RE.search(s) is None:
        raise ValueError("invalid date %r" % (s,))
    year = int(s[0:4])
    month = int(s[5:7])
    day = int(s[8:10])
    if month < 1 or month > 12 or day < 1 or day > days_in_month(year, month):
        raise ValueError("invalid date %r" % (s,))
    return Date(year, month, day)


class Calendar:
    """A month grid state plus a pure ``build``. ``month`` is the first day of
    the displayed month (only its year/month matter); ``selected`` is the
    cursor cell. ``today`` is caller-supplied; the widget never reads the
    wall clock. ``marked`` maps a date to opaque caller metadata; only its
    presence is used."""

    def __init__(self, id="", month=None, selected=None, width=0, marked=None,
                 style="", selected_style="", today_style="", header_style="",
                 weekday_style="", marked_style="", today=None):
        self.id = id
        self.month = month if month is not None else Date()
        self.selected = selected if selected is not None else Date()
        self.width = width
        self.marked = dict(marked or {})
        self.style = style
        self.selected_style = selected_style
        self.today_style = today_style
        self.header_style = header_style
        self.weekday_style = weekday_style
        self.marked_style = marked_style
        self.today = today if today is not None else Date()

    def month_label(self):
        """The header text ("January 2024") for the displayed month."""
        if self.month.month < 1 or self.month.month > 12:
            return ""
        return "%s %d" % (_MONTH_NAMES[self.month.month], self.month.year)

    def grid(self):
        """The displayed month as 6 rows of 7 Dates, Monday first. Days
        outside the month are still valid Date values (they carry over from
        the neighbouring months), which keeps every cell addressable."""
        first = Date(self.month.year, self.month.month, 1)
        if first.month < 1 or first.month > 12:
            return None
        offset = (first.weekday() + 6) % 7
        start = first.add_days(-offset)
        grid = []
        for row in range(6):
            grid.append([start.add_days(row * 7 + col) for col in range(7)])
        return grid

    def move_cursor(self, days):
        """Move the cursor by days, following it into the adjacent month when
        it leaves the grid. Reports whether the cursor moved."""
        if days == 0:
            return False
        cursor = self._normalized_selected()
        nxt = cursor.add_days(days)
        self.selected = nxt
        if nxt.month != self.month.month or nxt.year != self.month.year:
            self.month = Date(nxt.year, nxt.month, 1)
        return True

    def _normalized_selected(self):
        if self.selected.month >= 1 and self.selected.month <= 12 and self.selected.day >= 1:
            return self.selected
        if self.month.month >= 1 and self.month.month <= 12:
            return Date(self.month.year, self.month.month, 1)
        return self.month

    def build(self):
        """The calendar as a column: header, weekday row, then 6 grid rows.
        Every cell is padded to a fixed display width so CJK content cannot
        shift the grid."""
        col = builder.box("col")
        if self.id:
            col.id(self.id)
        if self.width > 0:
            col.width(self.width)
        if self.style:
            col.style(self.style)
        grid = self.grid()
        width = self._cell_width(grid)
        label = self.month_label()
        if label:
            col.child(builder.text(label).style(self.header_style).height(1))
        col.child(_calendar_weekday_row(self.weekday_style, width))
        if grid is None:
            return col
        for row in grid:
            line = builder.row().height(1)
            if self.width > 0:
                line.width(self.width)
            for date in row:
                line.child(builder.text(_calendar_cell_text(date, width))
                           .style(self._cell_style(date)).width(width).height(1))
            col.child(line)
        return col

    def _cell_width(self, grid):
        width = 3
        for row in grid or []:
            for date in row:
                width = max(width, builder.display_width(_calendar_cell_text(date, 0)))
        return width

    def _cell_style(self, date):
        if date == self._normalized_selected():
            return first_non_empty(self.selected_style,
                                   DEFAULT_CALENDAR_SELECTED_STYLE)
        mark = self.marked.get(date)
        if mark is not None and mark != "":
            return first_non_empty(self.marked_style, self.style)
        if self.today.month >= 1 and date == self.today:
            return first_non_empty(self.today_style, self.style)
        return self.style


def _calendar_weekday_row(style, cell_width):
    row = builder.row().height(1)
    for name in CALENDAR_WEEKDAYS:
        row.child(builder.text(pad_cell(name, cell_width, "center"))
                  .style(style).width(cell_width).height(1))
    return row


def _calendar_cell_text(date, width):
    return pad_cell("%2d" % date.day, width, "center")


def day_validator(min_date, max_date, msg):
    """A validator accepting "YYYY-MM-DD" dates inside the inclusive
    [min_date, max_date] range. The zero Date for a bound leaves that side
    open. Empty input passes; a malformed date fails with msg."""
    def validate(value):
        if value == "":
            return ""
        try:
            date = parse_date(value)
        except ValueError:
            return msg
        if min_date._valid() and date.before(min_date):
            return msg
        if max_date._valid() and date.after(max_date):
            return msg
        return ""
    return validate


class DateField(Field):
    """Wraps a TextInput with date validation and a ``date`` helper. The
    caller keeps the input in the form and reads back a parsed Date."""

    def __init__(self, id="", label="", input=None, min_date=None,
                 max_date=None, **field_fields):
        super().__init__(id=id, label=label, input=input, **field_fields)
        self.min = min_date if min_date is not None else Date()
        self.max = max_date if max_date is not None else Date()

    def validate_date(self, value):
        """The validate hook wired by ``new_date_field``."""
        return day_validator(self.min, self.max, DATE_FIELD_MESSAGE)(value)

    def date(self):
        """Parse the current input text, returning ``(date, True)`` or
        ``(None, False)`` when it is empty or malformed."""
        if self.input is None:
            return None, False
        try:
            return parse_date(self.input.text()), True
        except ValueError:
            return None, False


def new_date_field(id, label, input):
    """Build a DateField with the standard date validator attached."""
    field = DateField(id=id, label=label, input=input)
    field.validate = field.validate_date
    return field

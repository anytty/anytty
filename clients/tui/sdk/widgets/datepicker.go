package widgets

import (
	"fmt"
	"time"

	"github.com/anytty/anytty/clients/tui/sdk"
)

// Date is a calendar date: a plain year/month/day triple, independent of the
// time package's location and clock. Month is 1..12 and Day is 1..31; the
// zero Date is month 0 and is treated as "unset" by the calendar helpers.
// time is used only for calendar arithmetic, never for the current time.
type Date struct {
	Year  int
	Month int
	Day   int
}

// DateLayout is the wire/display layout accepted by ParseDate.
const DateLayout = "2006-01-02"

// IsLeap reports whether year is a Gregorian leap year.
func (d Date) IsLeap() bool { return isLeapYear(d.Year) }

// DaysInMonth returns the number of days in the date's month, or 0 when the
// month is out of range or the year is invalid.
func (d Date) DaysInMonth() int { return daysInMonth(d.Year, d.Month) }

// Weekday returns the day of week, where Sunday is 0.
func (d Date) Weekday() time.Weekday { return d.civil().Weekday() }

// Compare returns -1, 0 or 1 as d sorts before, equal to, or after other.
func (d Date) Compare(other Date) int {
	switch {
	case d.Year != other.Year:
		return sign(d.Year - other.Year)
	case d.Month != other.Month:
		return sign(d.Month - other.Month)
	case d.Day != other.Day:
		return sign(d.Day - other.Day)
	}
	return 0
}

// Equal reports whether two dates fall on the same day.
func (d Date) Equal(other Date) bool { return d.Compare(other) == 0 }

// Before reports whether d sorts strictly before other.
func (d Date) Before(other Date) bool { return d.Compare(other) < 0 }

// After reports whether d sorts strictly after other.
func (d Date) After(other Date) bool { return d.Compare(other) > 0 }

// AddDays returns the date n days after d (n may be negative). It normalizes
// through time, so month and year boundaries carry correctly.
func (d Date) AddDays(n int) Date {
	if !d.valid() {
		return d
	}
	return fromTime(d.civil().AddDate(0, 0, n))
}

// AddMonths returns the date n months after d (n may be negative), clamping
// the day to the target month's length (2024-01-31 + 1 month = 2024-02-29).
func (d Date) AddMonths(n int) Date {
	if !d.valid() {
		return d
	}
	year, month := addMonths(d.Year, d.Month, n)
	day := d.Day
	if maxDay := daysInMonth(year, month); day > maxDay {
		day = maxDay
	}
	return Date{Year: year, Month: month, Day: day}
}

// String renders the date as YYYY-MM-DD; the zero Date renders as "".
func (d Date) String() string {
	if !d.valid() {
		return ""
	}
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

// ParseDate parses "YYYY-MM-DD" and rejects any value time would silently
// normalize (2024-02-30, 2023-13-01, trailing characters).
func ParseDate(s string) (Date, error) {
	parsed, err := time.Parse(DateLayout, s)
	if err != nil {
		return Date{}, err
	}
	date := fromTime(parsed)
	if date.String() != s {
		return Date{}, fmt.Errorf("invalid date %q", s)
	}
	return date, nil
}

func (d Date) valid() bool {
	return d.Month >= 1 && d.Month <= 12 && d.Day >= 1 && d.Day <= 31
}

func (d Date) civil() time.Time {
	return time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
}

func fromTime(t time.Time) Date {
	return Date{Year: t.Year(), Month: int(t.Month()), Day: t.Day()}
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func daysInMonth(year, month int) int {
	if month < 1 || month > 12 {
		return 0
	}
	switch month {
	case 4, 6, 9, 11:
		return 30
	case 2:
		if isLeapYear(year) {
			return 29
		}
		return 28
	default:
		return 31
	}
}

func addMonths(year, month, n int) (int, int) {
	if month < 1 || month > 12 {
		return year, month
	}
	index := year*12 + (month - 1) + n
	targetYear := index / 12
	targetMonth := index % 12
	if targetMonth < 0 {
		targetMonth += 12
		targetYear--
	}
	return targetYear, targetMonth + 1
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

// DefaultCalendarSelectedStyle is the style of the cursor cell.
const DefaultCalendarSelectedStyle = "reverse"

// Calendar is a month grid state plus a pure Build. Month is the first day of
// the displayed month (only its Year/Month matter); Selected is the cursor
// cell. The caller owns both and drives them through the helpers; the widget
// never emits frames and never reads the wall clock — Today is caller-supplied.
// Marked maps a date to opaque caller metadata; only its presence is used, so
// marking a day never changes the fixed cell width.
type Calendar struct {
	ID            string
	Month         Date
	Selected      Date
	Width         int
	Marked        map[Date]string
	Style         string
	SelectedStyle string
	TodayStyle    string
	HeaderStyle   string
	WeekdayStyle  string
	MarkedStyle   string
	Today         Date
}

// MonthLabel returns the header text ("January 2024") for the displayed month.
func (c Calendar) MonthLabel() string {
	if c.Month.Month < 1 || c.Month.Month > 12 {
		return ""
	}
	return fmt.Sprintf("%s %d", time.Month(c.Month.Month).String(), c.Month.Year)
}

// Grid returns the displayed month as 6 rows of 7 Dates, Monday first. Days
// outside the month are still valid Date values (they carry over from the
// neighbouring months), which keeps every cell addressable.
func (c Calendar) Grid() [][]Date {
	first := Date{Year: c.Month.Year, Month: c.Month.Month, Day: 1}
	if first.Month < 1 || first.Month > 12 {
		return nil
	}
	offset := (int(first.Weekday()) + 6) % 7
	start := first.AddDays(-offset)
	grid := make([][]Date, 6)
	for row := range grid {
		grid[row] = make([]Date, 7)
		for col := range grid[row] {
			grid[row][col] = start.AddDays(row*7 + col)
		}
	}
	return grid
}

// MoveCursor moves the cursor by days, following it into the adjacent month
// when it leaves the grid. It reports whether the cursor moved.
func (c *Calendar) MoveCursor(days int) bool {
	if days == 0 {
		return false
	}
	cursor := c.normalizedSelected()
	next := cursor.AddDays(days)
	c.Selected = next
	if next.Month != c.Month.Month || next.Year != c.Month.Year {
		c.Month = Date{Year: next.Year, Month: next.Month, Day: 1}
	}
	return true
}

func (c Calendar) normalizedSelected() Date {
	if c.Selected.Month >= 1 && c.Selected.Month <= 12 && c.Selected.Day >= 1 {
		return c.Selected
	}
	if c.Month.Month >= 1 && c.Month.Month <= 12 {
		return Date{Year: c.Month.Year, Month: c.Month.Month, Day: 1}
	}
	return c.Month
}

// Build returns the calendar as a column: header, weekday row, then 6 grid
// rows. Every cell is padded to a fixed display width so CJK content cannot
// shift the grid.
func (c Calendar) Build() *sdk.Builder {
	col := sdk.Box().Flow("col")
	if c.ID != "" {
		col.ID(c.ID)
	}
	if c.Width > 0 {
		col.Width(c.Width)
	}
	if c.Style != "" {
		col.Style(c.Style)
	}
	grid := c.Grid()
	width := c.cellWidth(grid)
	if label := c.MonthLabel(); label != "" {
		col.Child(sdk.Text(label).Style(c.HeaderStyle).Height(1))
	}
	col.Child(calendarWeekdayRow(c.WeekdayStyle, width))
	if grid == nil {
		return col
	}
	for _, row := range grid {
		line := sdk.Row().Height(1)
		if c.Width > 0 {
			line.Width(c.Width)
		}
		for _, date := range row {
			line.Child(sdk.Text(calendarCellText(date, width)).Style(c.cellStyle(date)).Width(width).Height(1))
		}
		col.Child(line)
	}
	return col
}

func calendarWeekdayRow(style string, cellWidth int) *sdk.Builder {
	row := sdk.Row().Height(1)
	for _, name := range calendarWeekdays {
		row.Child(sdk.Text(padCell(name, cellWidth, AlignCenter)).Style(style).Width(cellWidth).Height(1))
	}
	return row
}

// calendarWeekdays is Monday-first to match Grid.
var calendarWeekdays = []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}

// cellWidth returns a fixed cell width that fits a two-digit day plus the
// space the cursor cell needs, so every row has identical display width.
func (c Calendar) cellWidth(grid [][]Date) int {
	width := 3
	for _, row := range grid {
		for _, date := range row {
			width = maxInt(width, sdk.DisplayWidth(calendarCellText(date, 0)))
		}
	}
	return width
}

func calendarCellText(date Date, width int) string {
	return padCell(fmt.Sprintf("%2d", date.Day), width, AlignCenter)
}

func (c Calendar) cellStyle(date Date) string {
	if date == c.normalizedSelected() {
		return firstNonEmpty(c.SelectedStyle, DefaultCalendarSelectedStyle)
	}
	if mark, ok := c.Marked[date]; ok && mark != "" {
		return firstNonEmpty(c.MarkedStyle, c.Style)
	}
	if c.Today.Month >= 1 && date == c.Today {
		return firstNonEmpty(c.TodayStyle, c.Style)
	}
	return c.Style
}

// DayValidator returns a Validator accepting "YYYY-MM-DD" dates inside the
// inclusive [min, max] range. Passing the zero Date for a bound leaves that
// side open. Empty input passes so the validator composes with Required, and
// a malformed date fails with msg.
func DayValidator(min, max Date, msg string) Validator {
	return func(value string) string {
		if value == "" {
			return ""
		}
		date, err := ParseDate(value)
		if err != nil {
			return msg
		}
		if min.valid() && date.Before(min) {
			return msg
		}
		if max.valid() && date.After(max) {
			return msg
		}
		return ""
	}
}

// DateField wraps a TextInput with date validation and a Display helper. It
// is the text-entry counterpart of Calendar: the caller keeps the input in the
// form and reads back a parsed Date.
type DateField struct {
	Field
	Min Date
	Max Date
}

// DateFieldMessage is the error stored for a malformed or out-of-range date.
const DateFieldMessage = "invalid date"

// NewDateField builds a DateField with the standard date validator attached.
func NewDateField(id, label string, input *TextInput) DateField {
	field := DateField{Field: Field{ID: id, Label: label, Input: input}}
	field.Validate = field.validate
	return field
}

// validate is the Validate hook wired by NewDateField.
func (d DateField) validate(value string) string {
	return DayValidator(d.Min, d.Max, DateFieldMessage)(value)
}

// Date parses the current input text, returning ok=false when it is empty or
// malformed.
func (d DateField) Date() (Date, bool) {
	if d.Input == nil {
		return Date{}, false
	}
	date, err := ParseDate(d.Input.Text())
	if err != nil {
		return Date{}, false
	}
	return date, true
}

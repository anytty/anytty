package widgets

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func TestDateDaysInMonthAndIsLeap(t *testing.T) {
	cases := []struct {
		year, month, days int
		leap              bool
	}{
		{2023, 1, 31, false},
		{2023, 2, 28, false},
		{2024, 2, 29, true},
		{1900, 2, 28, false},
		{2000, 2, 29, true},
		{2023, 4, 30, false},
		{2023, 12, 31, false},
	}
	for _, tc := range cases {
		date := Date{Year: tc.year, Month: tc.month, Day: 1}
		if got := date.DaysInMonth(); got != tc.days {
			t.Fatalf("%s DaysInMonth = %d, want %d", date, got, tc.days)
		}
		if got := date.IsLeap(); got != tc.leap {
			t.Fatalf("%d IsLeap = %v", tc.year, got)
		}
	}
	if got := (Date{Year: 2024, Month: 13, Day: 1}).DaysInMonth(); got != 0 {
		t.Fatalf("month 13 = %d", got)
	}
}

func TestDateAddDaysAcrossBoundaries(t *testing.T) {
	cases := []struct {
		start Date
		days  int
		want  Date
	}{
		{Date{2024, 2, 28}, 1, Date{2024, 2, 29}},
		{Date{2024, 2, 29}, 1, Date{2024, 3, 1}},
		{Date{2023, 2, 28}, 1, Date{2023, 3, 1}},
		{Date{2023, 12, 31}, 1, Date{2024, 1, 1}},
		{Date{2024, 1, 1}, -1, Date{2023, 12, 31}},
		{Date{2024, 3, 1}, -1, Date{2024, 2, 29}},
		{Date{2024, 1, 1}, 60, Date{2024, 3, 1}},
		{Date{2024, 1, 1}, 0, Date{2024, 1, 1}},
	}
	for _, tc := range cases {
		if got := tc.start.AddDays(tc.days); got != tc.want {
			t.Fatalf("%s + %d days = %s, want %s", tc.start, tc.days, got, tc.want)
		}
	}
	zero := Date{}
	if zero.AddDays(1) != zero || zero.AddMonths(1) != zero {
		t.Fatal("zero date arithmetic must be a no-op")
	}
}

func TestDateAddMonthsClamps(t *testing.T) {
	cases := []struct {
		start  Date
		months int
		want   Date
	}{
		{Date{2024, 1, 31}, 1, Date{2024, 2, 29}},
		{Date{2023, 1, 31}, 1, Date{2023, 2, 28}},
		{Date{2024, 1, 15}, 1, Date{2024, 2, 15}},
		{Date{2024, 12, 15}, 1, Date{2025, 1, 15}},
		{Date{2024, 1, 15}, -1, Date{2023, 12, 15}},
		{Date{2024, 1, 15}, 12, Date{2025, 1, 15}},
		{Date{2024, 1, 15}, -13, Date{2022, 12, 15}},
	}
	for _, tc := range cases {
		if got := tc.start.AddMonths(tc.months); got != tc.want {
			t.Fatalf("%s + %d months = %s, want %s", tc.start, tc.months, got, tc.want)
		}
	}
}

func TestDateCompareWeekdayString(t *testing.T) {
	a, b := Date{2024, 1, 1}, Date{2024, 1, 2}
	if a.Compare(b) != -1 || b.Compare(a) != 1 || a.Compare(a) != 0 {
		t.Fatal("Compare ordering is wrong")
	}
	if !a.Before(b) || a.After(b) || !a.Equal(Date{2024, 1, 1}) {
		t.Fatal("Before/After/Equal are wrong")
	}
	if got := (Date{2024, 1, 1}).Weekday().String(); got != "Monday" {
		t.Fatalf("2024-01-01 weekday = %s", got)
	}
	if got := (Date{2024, 2, 29}).Weekday().String(); got != "Thursday" {
		t.Fatalf("2024-02-29 weekday = %s", got)
	}
	if got := (Date{2024, 1, 2}).String(); got != "2024-01-02" {
		t.Fatalf("String = %q", got)
	}
	if got := (Date{}).String(); got != "" {
		t.Fatalf("zero String = %q", got)
	}
}

func TestParseDate(t *testing.T) {
	date, err := ParseDate("2024-02-29")
	if err != nil || date != (Date{2024, 2, 29}) {
		t.Fatalf("ParseDate = %+v %v", date, err)
	}
	for _, bad := range []string{"", "2024-2-9", "2024-02-30", "2023-02-29", "2023-13-01", "2024-00-10", "2024-01-32", "not-a-date", "2024-01-01T00:00:00Z"} {
		if _, err := ParseDate(bad); err == nil {
			t.Fatalf("ParseDate(%q) must fail", bad)
		}
	}
}

func TestCalendarGridAlignment(t *testing.T) {
	cases := []struct {
		month     Date
		firstCell Date
		lastCell  Date
		firstRow  Date
	}{
		// 2024-01-01 is a Monday: the grid starts exactly on the 1st and the
		// 6-row window ends on Feb 11.
		{Date{2024, 1, 1}, Date{2024, 1, 1}, Date{2024, 2, 11}, Date{2024, 1, 1}},
		// 2024-02-01 is a Thursday: offset 3 -> Mon Jan 29.
		{Date{2024, 2, 1}, Date{2024, 1, 29}, Date{2024, 3, 10}, Date{2024, 1, 29}},
		// 2023-12-01 is a Friday: offset 4 -> Mon Nov 27.
		{Date{2023, 12, 1}, Date{2023, 11, 27}, Date{2024, 1, 7}, Date{2023, 11, 27}},
		// 2024-09-01 is a Sunday: offset 6 -> Mon Aug 26.
		{Date{2024, 9, 1}, Date{2024, 8, 26}, Date{2024, 10, 6}, Date{2024, 8, 26}},
	}
	for _, tc := range cases {
		cal := Calendar{Month: tc.month}
		grid := cal.Grid()
		if len(grid) != 6 {
			t.Fatalf("%s rows = %d", tc.month, len(grid))
		}
		for _, row := range grid {
			if len(row) != 7 {
				t.Fatalf("%s cols = %d", tc.month, len(row))
			}
		}
		if grid[0][0] != tc.firstCell {
			t.Fatalf("%s first cell = %s, want %s", tc.month, grid[0][0], tc.firstCell)
		}
		if grid[5][6] != tc.lastCell {
			t.Fatalf("%s last cell = %s, want %s", tc.month, grid[5][6], tc.lastCell)
		}
		for _, row := range grid {
			if row[0].Weekday().String() != "Monday" {
				t.Fatalf("%s row starts on %s", tc.month, row[0].Weekday())
			}
			for col := 1; col < 7; col++ {
				if row[col] != row[col-1].AddDays(1) {
					t.Fatalf("%s cell %d is not contiguous", tc.month, col)
				}
			}
		}
	}
	if (&Calendar{}).Grid() != nil {
		t.Fatal("unset month grid must be nil")
	}
}

func TestCalendarMoveCursor(t *testing.T) {
	cal := Calendar{Month: Date{2024, 1, 1}, Selected: Date{2024, 1, 31}}
	if !cal.MoveCursor(1) || cal.Selected != (Date{2024, 2, 1}) {
		t.Fatalf("MoveCursor over month end = %s month %s", cal.Selected, cal.Month)
	}
	if cal.Month.Month != 2 || cal.Month.Day != 1 {
		t.Fatalf("month should follow cursor: %+v", cal.Month)
	}
	if !cal.MoveCursor(-1) || cal.Selected != (Date{2024, 1, 31}) {
		t.Fatalf("MoveCursor back = %s", cal.Selected)
	}
	if cal.Month.Month != 1 {
		t.Fatalf("month should follow back: %+v", cal.Month)
	}
	if cal.MoveCursor(0) {
		t.Fatal("zero delta must not report movement")
	}
	cursorless := Calendar{Month: Date{2024, 3, 1}}
	cursorless.MoveCursor(7)
	if cursorless.Selected != (Date{2024, 3, 8}) {
		t.Fatalf("MoveCursor from unset cursor = %s", cursorless.Selected)
	}
}

func TestCalendarBuild(t *testing.T) {
	cal := Calendar{
		ID:            "cal",
		Month:         Date{2024, 2, 1},
		Selected:      Date{2024, 2, 14},
		Marked:        map[Date]string{{2024, 2, 20}: "★"},
		Today:         Date{2024, 2, 1},
		SelectedStyle: "reverse",
		MarkedStyle:   "accent",
		TodayStyle:    "bold",
	}
	root := cal.Build().Build()
	if root.GetId() != "cal" {
		t.Fatalf("id = %q", root.GetId())
	}
	if got := boxText(root.GetChildren()[0]); got != "February 2024" {
		t.Fatalf("header = %q", got)
	}
	weekdays := rowText(root.GetChildren()[1])
	if got := sdk.DisplayWidth(weekdays); got != 21 {
		t.Fatalf("weekday row width = %d (%q)", got, weekdays)
	}
	trimmed := strings.TrimSpace(weekdays)
	if !strings.HasPrefix(trimmed, "Mo") || !strings.HasSuffix(trimmed, "Su") {
		t.Fatalf("weekday row = %q", weekdays)
	}
	grid := root.GetChildren()[2:]
	if len(grid) != 6 {
		t.Fatalf("grid rows = %d", len(grid))
	}
	rowWidth := -1
	for _, row := range grid {
		width := sdk.DisplayWidth(rowText(row))
		if rowWidth == -1 {
			rowWidth = width
		}
		if width != rowWidth {
			t.Fatalf("ragged grid: %d != %d", width, rowWidth)
		}
	}
	if rowWidth != 21 {
		t.Fatalf("grid width = %d, want 21", rowWidth)
	}
	if got := findMarkedStyle(root); got != "accent" {
		t.Fatalf("marked style = %q", got)
	}
	if got := findStyledCell(root, "reverse"); got == "" {
		t.Fatal("selected cell style not found")
	}

	unset := Calendar{Month: Date{2024, 2, 1}}
	if got := len(unset.Build().Build().GetChildren()); got != 8 {
		t.Fatalf("unset calendar children = %d, want header+weekday+6 rows", got)
	}
}

func findMarkedStyle(root *pb.Box) string {
	for _, row := range root.GetChildren() {
		for _, cell := range row.GetChildren() {
			if cell.GetStyle() == "accent" {
				return cell.GetStyle()
			}
		}
	}
	return ""
}

func findStyledCell(root *pb.Box, style string) string {
	for _, row := range root.GetChildren() {
		for _, cell := range row.GetChildren() {
			if cell.GetStyle() == style {
				return boxText(cell)
			}
		}
	}
	return ""
}

func TestDayValidator(t *testing.T) {
	v := DayValidator(Date{2024, 1, 1}, Date{2024, 12, 31}, "bad range")
	if got := v(""); got != "" {
		t.Fatalf("empty = %q", got)
	}
	if got := v("2024-06-15"); got != "" {
		t.Fatalf("in range = %q", got)
	}
	if got := v("2023-12-31"); got != "bad range" {
		t.Fatalf("before min = %q", got)
	}
	if got := v("2025-01-01"); got != "bad range" {
		t.Fatalf("after max = %q", got)
	}
	if got := v("2024-02-30"); got != "bad range" {
		t.Fatalf("malformed = %q", got)
	}
	openMin := DayValidator(Date{}, Date{2024, 12, 31}, "bad range")
	if got := openMin("1900-01-01"); got != "" {
		t.Fatalf("open min = %q", got)
	}
	openMax := DayValidator(Date{2024, 1, 1}, Date{}, "bad range")
	if got := openMax("2999-12-31"); got != "" {
		t.Fatalf("open max = %q", got)
	}
}

func TestDateField(t *testing.T) {
	input := &TextInput{ID: "due", Placeholder: "YYYY-MM-DD"}
	field := NewDateField("due", "Due", input)
	if field.ID != "due" || field.Label != "Due" || field.Validate == nil {
		t.Fatalf("NewDateField = %+v", field.Field)
	}
	if got := field.Validate("2024-02-29"); got != "" {
		t.Fatalf("valid date = %q", got)
	}
	if got := field.Validate("2024-02-30"); got != DateFieldMessage {
		t.Fatalf("invalid date = %q", got)
	}
	if _, ok := field.Date(); ok {
		t.Fatal("empty input must not parse")
	}
	input.SetValue("2024-07-04")
	date, ok := field.Date()
	if !ok || date != (Date{2024, 7, 4}) {
		t.Fatalf("Date = %+v %v", date, ok)
	}

	required := NewDateField("due", "Due", input)
	required.Required = true
	form := &Form{Fields: []Field{required.Field}}
	input.SetValue("")
	if form.Validate() {
		t.Fatal("empty required date field should be invalid")
	}
	if form.Fields[0].Error != FormRequiredMessage {
		t.Fatalf("field error = %q", form.Fields[0].Error)
	}
	input.SetValue("2024-02-30")
	if form.Validate() {
		t.Fatal("malformed date should be invalid")
	}
	if form.Fields[0].Error != DateFieldMessage {
		t.Fatalf("field error = %q", form.Fields[0].Error)
	}
}

func TestCalendarCJKMarkedWidthStable(t *testing.T) {
	cal := Calendar{Month: Date{2024, 2, 1}, Marked: map[Date]string{{2024, 2, 1}: "春节", {2024, 2, 2}: "★"}}
	root := cal.Build().Build()
	widths := make([]int, 0, 6)
	for _, row := range root.GetChildren()[2:] {
		widths = append(widths, sdk.DisplayWidth(rowText(row)))
	}
	for _, width := range widths {
		if width != widths[0] {
			t.Fatalf("CJK marked calendar rows ragged: %v", widths)
		}
	}
}

package widgets

import (
	"fmt"
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
)

// Cell alignment values of Column.Align.
const (
	AlignLeft   = "left"
	AlignRight  = "right"
	AlignCenter = "center"
)

// DefaultTableSeparator is the one-cell gutter between Table columns.
const DefaultTableSeparator = " "

// Column is one Table column spec. Width > 0 is a fixed cell count; otherwise
// the width is the widest of Title and the formatted cells, raised to
// MinWidth. Flex distributes leftover cells on a declared Table.Width.
type Column struct {
	Title    string
	Width    int
	MinWidth int
	Flex     int
	Align    string
	Format   func(any) string
}

// Table renders header + rows of aligned, truncated text cells. Rows are
// []string; Records are []any formatted by the column Format (fmt.Sprint when
// absent). Records wins when both are set. Cell text is truncated with
// sdk.Truncate, so CJK/emoji never split in half.
type Table struct {
	ID             string
	Columns        []Column
	Rows           [][]string
	Records        [][]any
	Width          int
	HideHeader     bool
	Rule           bool
	Zebra          bool
	ZebraStyle     string
	Selected       int
	Style          string
	HeaderStyle    string
	SelectedStyle  string
	FooterStyle    string
	SeparatorStyle string
	Footer         []string
	Separator      string
	RowID          func(index int) string
}

// RowCount returns the number of data rows (Records wins over Rows).
func (t Table) RowCount() int {
	if len(t.Records) > 0 {
		return len(t.Records)
	}
	return len(t.Rows)
}

// Cell returns the formatted text of one cell; out-of-range cells are empty.
func (t Table) Cell(row, col int) string {
	if len(t.Records) > 0 {
		if row < 0 || row >= len(t.Records) || col < 0 || col >= len(t.Records[row]) {
			return ""
		}
		if col < len(t.Columns) && t.Columns[col].Format != nil {
			return t.Columns[col].Format(t.Records[row][col])
		}
		return fmt.Sprint(t.Records[row][col])
	}
	if row < 0 || row >= len(t.Rows) || col < 0 || col >= len(t.Rows[row]) {
		return ""
	}
	return t.Rows[row][col]
}

// ColumnWidths solves the cell widths: natural size plus MinWidth, then Flex
// distribution over a declared Width, then a right-to-left shrink to fit.
func (t Table) ColumnWidths() []int {
	widths := make([]int, len(t.Columns))
	if len(t.Columns) == 0 {
		return widths
	}
	for i, col := range t.Columns {
		if col.Width > 0 {
			widths[i] = col.Width
			continue
		}
		width := sdk.DisplayWidth(col.Title)
		for row := 0; row < t.RowCount(); row++ {
			if cell := sdk.DisplayWidth(t.Cell(row, i)); cell > width {
				width = cell
			}
		}
		if min := t.minWidth(i); width < min {
			width = min
		}
		if width < 1 {
			width = 1
		}
		widths[i] = width
	}
	if t.Width <= 0 {
		return widths
	}
	avail := t.Width - t.sepWidth()*(len(t.Columns)-1)
	if avail <= 0 {
		return widths
	}
	used := 0
	for _, width := range widths {
		used += width
	}
	switch {
	case used < avail:
		flex := 0
		for _, col := range t.Columns {
			if col.Width <= 0 && col.Flex > 0 {
				flex += col.Flex
			}
		}
		if flex == 0 {
			return widths
		}
		extra, added, last := avail-used, 0, -1
		for i, col := range t.Columns {
			if col.Width > 0 || col.Flex <= 0 {
				continue
			}
			share := extra * col.Flex / flex
			widths[i] += share
			added += share
			last = i
		}
		if last >= 0 {
			widths[last] += extra - added
		}
	case used > avail:
		for i := len(widths) - 1; i >= 0 && used > avail; i-- {
			if t.Columns[i].Width > 0 {
				continue
			}
			reduce := used - avail
			if floor := t.minWidth(i); widths[i]-reduce < floor {
				reduce = widths[i] - floor
			}
			if reduce > 0 {
				widths[i] -= reduce
				used -= reduce
			}
		}
	}
	return widths
}

// Build returns the table as a column of one-row Rows.
func (t Table) Build() *sdk.Builder {
	col := sdk.Box().Flow("col")
	if t.ID != "" {
		col.ID(t.ID)
	}
	if t.Width > 0 {
		col.Width(t.Width)
	}
	if len(t.Columns) == 0 {
		return col
	}
	widths := t.ColumnWidths()
	if !t.HideHeader {
		col.Child(t.rowBox(t.headerCells(), widths, t.HeaderStyle, "", -1))
		if t.Rule {
			col.Child(sdk.Text(strings.Repeat("─", t.totalWidth(widths))).Style(t.HeaderStyle).Height(1))
		}
	}
	for row := 0; row < t.RowCount(); row++ {
		style := t.Style
		if t.Zebra && row%2 == 1 {
			style = firstNonEmpty(t.ZebraStyle, style)
		}
		if row == t.Selected {
			style = firstNonEmpty(t.SelectedStyle, style)
		}
		id := ""
		if t.RowID != nil {
			id = t.RowID(row)
		}
		col.Child(t.rowBox(t.cells(row), widths, style, id, row))
	}
	if len(t.Footer) > 0 {
		col.Child(t.rowBox(append([]string(nil), t.Footer...), widths, t.FooterStyle, "", -1))
	}
	return col
}

func (t Table) headerCells() []string {
	cells := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		cells[i] = col.Title
	}
	return cells
}

func (t Table) cells(row int) []string {
	cells := make([]string, len(t.Columns))
	for i := range t.Columns {
		cells[i] = t.Cell(row, i)
	}
	return cells
}

func (t Table) rowBox(cells []string, widths []int, style, id string, row int) *sdk.Builder {
	box := sdk.Row().Height(1)
	if id != "" {
		box.ID(id).Input("mouse")
	}
	for i, width := range widths {
		if i > 0 {
			box.Child(sdk.Text(t.separatorText()).Style(t.SeparatorStyle))
		}
		text := ""
		if i < len(cells) {
			text = cells[i]
		}
		box.Child(sdk.Text(padCell(text, width, t.align(i))).Style(style).Width(width).Height(1))
	}
	return box
}

func (t Table) align(col int) string {
	if col >= 0 && col < len(t.Columns) {
		return t.Columns[col].Align
	}
	return ""
}

func (t Table) minWidth(col int) int {
	if col >= 0 && col < len(t.Columns) && t.Columns[col].MinWidth > 0 {
		return t.Columns[col].MinWidth
	}
	return 1
}

func (t Table) separatorText() string {
	if t.Separator == "" {
		return DefaultTableSeparator
	}
	return t.Separator
}

func (t Table) sepWidth() int {
	return sdk.DisplayWidth(t.separatorText())
}

func (t Table) totalWidth(widths []int) int {
	if len(widths) == 0 {
		return 0
	}
	total := t.sepWidth() * (len(widths) - 1)
	for _, width := range widths {
		total += width
	}
	return total
}

// padCell truncates text to width display cells and pads it according to the
// column alignment.
func padCell(text string, width int, align string) string {
	text = sdk.Truncate(text, width)
	pad := width - sdk.DisplayWidth(text)
	if pad <= 0 {
		return text
	}
	switch align {
	case AlignRight:
		return strings.Repeat(" ", pad) + text
	case AlignCenter:
		left := pad / 2
		return strings.Repeat(" ", left) + text + strings.Repeat(" ", pad-left)
	default:
		return text + strings.Repeat(" ", pad)
	}
}

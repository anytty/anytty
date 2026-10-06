package widgets

import (
	"fmt"
	"testing"
)

func TestTableColumnWidths(t *testing.T) {
	table := Table{
		Columns: []Column{{Title: "Name"}, {Title: "N", MinWidth: 4}},
		Rows:    [][]string{{"alpha", "1"}, {"b", "12345"}},
	}
	widths := table.ColumnWidths()
	if len(widths) != 2 || widths[0] != 5 || widths[1] != 5 {
		t.Fatalf("natural widths = %v, want [5 5]", widths)
	}

	table.Columns[0].Width = 3
	widths = table.ColumnWidths()
	if widths[0] != 3 || widths[1] != 5 {
		t.Fatalf("fixed width = %v, want [3 5]", widths)
	}
}

func TestTableFlexAndShrink(t *testing.T) {
	flex := Table{
		Width:   20,
		Columns: []Column{{Title: "A"}, {Title: "B", Flex: 1}},
		Rows:    [][]string{{"x", "y"}},
	}
	widths := flex.ColumnWidths()
	if total := widths[0] + widths[1] + flex.sepWidth(); total != 20 {
		t.Fatalf("flex widths = %v, total = %d, want 20", widths, total)
	}
	if widths[1] <= widths[0] {
		t.Fatalf("flex column must absorb the leftover: %v", widths)
	}

	shrink := Table{
		Width:   6,
		Columns: []Column{{Title: "A"}, {Title: "B"}},
		Rows:    [][]string{{"aaaa", "bbbb"}},
	}
	widths = shrink.ColumnWidths()
	if total := widths[0] + widths[1] + shrink.sepWidth(); total != 6 {
		t.Fatalf("shrunk widths = %v, total = %d, want 6", widths, total)
	}
	if widths[0] != 4 || widths[1] != 1 {
		t.Fatalf("shrink must reduce the right column first: %v", widths)
	}
}

func TestTableBuildHeaderSelectionZebraFooter(t *testing.T) {
	table := Table{
		ID: "tbl",
		Columns: []Column{
			{Title: "Name", Width: 4},
			{Title: "V", Width: 3, Align: AlignRight},
		},
		Rows:          [][]string{{"ab", "7"}, {"c", "12"}, {"dd", "3"}},
		Selected:      2,
		Zebra:         true,
		ZebraStyle:    "zebra",
		SelectedStyle: "sel",
		HeaderStyle:   "hdr",
		FooterStyle:   "ftr",
		Rule:          true,
		Footer:        []string{"F", "9"},
		RowID:         func(index int) string { return fmt.Sprintf("row:%d", index) },
	}
	children := table.Build().Build().GetChildren()
	if len(children) != 6 {
		t.Fatalf("table rows = %d, want header + rule + 3 rows + footer", len(children))
	}
	header := children[0].GetChildren()
	if len(header) != 3 || boxText(header[0]) != "Name" || boxText(header[2]) != "  V" {
		t.Fatalf("header cells = %+v", header)
	}
	if header[0].GetStyle() != "hdr" {
		t.Fatalf("header style = %q", header[0].GetStyle())
	}
	if children[1].GetStyle() != "hdr" {
		t.Fatalf("rule style = %q", children[1].GetStyle())
	}
	if children[2].GetChildren()[0].GetStyle() != "" {
		t.Fatalf("even row must stay unstyled: %+v", children[2])
	}
	if children[3].GetChildren()[0].GetStyle() != "zebra" {
		t.Fatalf("zebra row style = %q", children[3].GetChildren()[0].GetStyle())
	}
	if children[4].GetId() != "row:2" || children[4].GetChildren()[0].GetStyle() != "sel" {
		t.Fatalf("selected row = %+v", children[4])
	}
	if len(children[4].GetInput()) != 1 || children[4].GetInput()[0] != "mouse" {
		t.Fatalf("selected row input = %v", children[4].GetInput())
	}
	if footer := children[5].GetChildren(); boxText(footer[0]) != "F   " || footer[0].GetStyle() != "ftr" {
		t.Fatalf("footer cells = %+v", footer)
	}
}

func TestTableCellTruncationAndAlignment(t *testing.T) {
	table := Table{
		HideHeader: true,
		Columns: []Column{
			{Title: "L", Width: 4},
			{Title: "R", Width: 4, Align: AlignRight},
			{Title: "C", Width: 4, Align: AlignCenter},
		},
		Rows: [][]string{{"中文你好", "ab", "ab"}},
	}
	row := table.Build().Build().GetChildren()[0].GetChildren()
	if got := boxText(row[0]); got != "中文" {
		t.Fatalf("CJK cell = %q", got)
	}
	if got := boxText(row[2]); got != "  ab" {
		t.Fatalf("right cell = %q", got)
	}
	if got := boxText(row[4]); got != " ab " {
		t.Fatalf("center cell = %q", got)
	}
}

func TestTableRecordsFormat(t *testing.T) {
	table := Table{
		Columns: []Column{
			{Title: "N", Format: func(value any) string { return fmt.Sprintf("<%v>", value) }},
			{Title: "D"},
		},
		Records: [][]any{{42, "x"}, {"七", 7}},
		Rows:    [][]string{{"ignored", "ignored"}},
	}
	if table.RowCount() != 2 {
		t.Fatalf("RowCount = %d, want records to win", table.RowCount())
	}
	if got := table.Cell(0, 0); got != "<42>" {
		t.Fatalf("formatted cell = %q", got)
	}
	if got := table.Cell(1, 0); got != "<七>" {
		t.Fatalf("formatted CJK cell = %q", got)
	}
	if got := table.Cell(0, 1); got != "x" {
		t.Fatalf("fallback cell = %q", got)
	}
	if got := table.Cell(9, 9); got != "" {
		t.Fatalf("out-of-range cell = %q", got)
	}
}

func TestTableEmptyAndShortRows(t *testing.T) {
	var empty Table
	if children := empty.Build().Build().GetChildren(); len(children) != 0 {
		t.Fatalf("empty table children = %d", len(children))
	}
	short := Table{
		Columns:    []Column{{Title: "A", Width: 2}, {Title: "B", Width: 2}},
		Rows:       [][]string{{"1"}},
		HideHeader: true,
		Width:      5,
	}
	children := short.Build().Build().GetChildren()
	if len(children) != 1 {
		t.Fatalf("short table rows = %d", len(children))
	}
	cells := children[0].GetChildren()
	if got := boxText(cells[2]); got != "  " {
		t.Fatalf("missing cell = %q, want two padded spaces", got)
	}
}

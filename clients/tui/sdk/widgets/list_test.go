package widgets

import (
	"fmt"
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk"
)

func TestListRendersOnlyVisibleRows(t *testing.T) {
	items := make([]string, 100000)
	for i := range items {
		items[i] = fmt.Sprintf("item-%d", i)
	}
	list := List{Items: items, Height: 10, Width: 20, Selected: 50000, Follow: true}
	start, end := list.VisibleRange()
	if end-start != 10 {
		t.Fatalf("VisibleRange = (%d, %d), want a 10-row window", start, end)
	}
	root := list.Build().Build()
	if len(root.GetChildren()) != 10 {
		t.Fatalf("rendered rows = %d, want 10", len(root.GetChildren()))
	}
	if first := boxText(root.GetChildren()[0]); !strings.Contains(first, "item-49991") {
		t.Fatalf("first visible row = %q, want item-49991", first)
	}
	last := boxText(root.GetChildren()[9])
	if !strings.Contains(last, "item-50000") || !strings.HasPrefix(last, DefaultListMarker) {
		t.Fatalf("last visible row = %q, want the selected item-50000 with a marker", last)
	}
}

func TestListHelpers(t *testing.T) {
	list := List{Items: []string{"a", "b", "c", "d", "e"}, Height: 2}
	list.Move(1)
	if list.Selected != 1 {
		t.Fatalf("Move(1) selected = %d", list.Selected)
	}
	list.Move(99)
	if list.Selected != 4 {
		t.Fatalf("Move(99) selected = %d, want the last row", list.Selected)
	}
	list.PageUp()
	if list.Selected != 2 {
		t.Fatalf("PageUp selected = %d, want 2", list.Selected)
	}
	list.PageDown()
	if list.Selected != 4 {
		t.Fatalf("PageDown selected = %d, want 4", list.Selected)
	}
	list.Top()
	if list.Selected != 0 || list.Offset != 0 {
		t.Fatalf("Top = (%d, %d), want (0, 0)", list.Selected, list.Offset)
	}
	list.Bottom()
	if list.Selected != 4 || list.Offset != 3 {
		t.Fatalf("Bottom = (%d, %d), want (4, 3)", list.Selected, list.Offset)
	}
	list.Offset = 100
	list.Selected = 100
	list.EnsureVisible()
	if list.Selected != 4 || list.Offset != 3 {
		t.Fatalf("EnsureVisible = (%d, %d), want (4, 3)", list.Selected, list.Offset)
	}
}

func TestListEmptyAndHeightOne(t *testing.T) {
	var empty List
	if start, end := empty.VisibleRange(); start != 0 || end != 0 {
		t.Fatalf("empty VisibleRange = (%d, %d)", start, end)
	}
	empty.Move(1)
	empty.PageUp()
	empty.PageDown()
	empty.Top()
	empty.Bottom()
	if empty.Selected != 0 || len(empty.Build().Build().GetChildren()) != 0 {
		t.Fatalf("empty list mutated: %+v", empty)
	}
	placeholder := List{Empty: "no items", Height: 3, Width: 10}.Build().Build()
	if len(placeholder.GetChildren()) != 1 || boxText(placeholder.GetChildren()[0]) != "no items  " {
		t.Fatalf("empty placeholder = %+v", placeholder)
	}

	one := List{Items: []string{"only"}, Height: 1, Width: 8}.Build().Build()
	if len(one.GetChildren()) != 1 || sdk.DisplayWidth(boxText(one.GetChildren()[0])) != 8 {
		t.Fatalf("height-one list = %+v", one)
	}
}

func TestListHeaderFooterWindow(t *testing.T) {
	list := List{Items: []string{"a", "b", "c", "d"}, Height: 4, Header: "HDR", Footer: "FTR", HeaderStyle: "h", FooterStyle: "f"}
	start, end := list.VisibleRange()
	if start != 0 || end != 2 {
		t.Fatalf("window with chrome = (%d, %d), want (0, 2)", start, end)
	}
	children := list.Build().Build().GetChildren()
	if len(children) != 4 {
		t.Fatalf("children = %d, want header + 2 rows + footer", len(children))
	}
	if boxText(children[0]) != "HDR" || children[0].GetStyle() != "h" {
		t.Fatalf("header = %+v", children[0])
	}
	if boxText(children[3]) != "FTR" || children[3].GetStyle() != "f" {
		t.Fatalf("footer = %+v", children[3])
	}
}

func TestListFollowFalseKeepsOffset(t *testing.T) {
	list := List{Items: []string{"a", "b", "c", "d", "e"}, Height: 2, Selected: 4}
	if _, end := list.VisibleRange(); end != 2 {
		t.Fatalf("Follow=false end = %d, want 2", end)
	}
	list.Follow = true
	if start, _ := list.VisibleRange(); start != 3 {
		t.Fatalf("Follow=true start = %d, want 3", start)
	}
}

func TestListRowIDStyleAndCJKTruncation(t *testing.T) {
	list := List{
		Items:         []string{"中文中文", "short"},
		Height:        2,
		Width:         6,
		Selected:      1,
		Style:         "base",
		SelectedStyle: "sel",
		RowID:         func(index int) string { return fmt.Sprintf("row:%d", index) },
	}
	children := list.Build().Build().GetChildren()
	cell := boxText(children[0])
	if sdk.DisplayWidth(cell) != 6 || !strings.Contains(cell, "中文") {
		t.Fatalf("CJK row = %q (width %d)", cell, sdk.DisplayWidth(cell))
	}
	if children[0].GetId() != "row:0" || children[0].GetInput()[0] != "mouse" {
		t.Fatalf("row 0 id/input = %q %v", children[0].GetId(), children[0].GetInput())
	}
	if !strings.HasPrefix(boxText(children[1]), DefaultListMarker) || children[1].GetStyle() != "sel" {
		t.Fatalf("selected row = %+v", children[1])
	}
}

func TestVirtualListRowsAndDisabled(t *testing.T) {
	list := VirtualList{
		ID: "vl",
		Rows: []ListRow{
			{Text: "one", ID: "vl:0", Style: "s0"},
			{Text: "two", ID: "vl:1", Disabled: true},
			{Text: "three", ID: "vl:2"},
		},
		Height: 2, Width: 8, Offset: 1, Selected: 2, Follow: true,
		DisabledStyle: "off", SelectedStyle: "sel",
	}
	start, end := list.VisibleRange()
	if start != 1 || end != 3 {
		t.Fatalf("VirtualList window = (%d, %d), want (1, 3)", start, end)
	}
	children := list.Build().Build().GetChildren()
	if len(children) != 2 {
		t.Fatalf("VirtualList rows = %d, want 2", len(children))
	}
	if children[0].GetStyle() != "off" || children[0].GetId() != "vl:1" {
		t.Fatalf("disabled row = %+v", children[0])
	}
	if children[1].GetStyle() != "sel" || !strings.HasPrefix(boxText(children[1]), DefaultListMarker) {
		t.Fatalf("selected row = %+v", children[1])
	}

	var empty VirtualList
	empty.Move(3)
	empty.Bottom()
	if empty.Selected != 0 || empty.Build().Build().GetId() != "" {
		t.Fatalf("empty VirtualList mutated: %+v", empty)
	}
}

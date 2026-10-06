package widgets

import (
	"testing"
	"time"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func TestHitPrecedenceTopmostWins(t *testing.T) {
	regions := []HitRegion{
		{ID: "below", X: 0, Y: 0, W: 10, H: 10},
		{ID: "above", X: 2, Y: 2, W: 3, H: 3},
	}
	region, ok := Hit(regions, 3, 3)
	if !ok || region.ID != "above" {
		t.Fatalf("overlap hit = %+v, want the last region", region)
	}
	if region, ok := Hit(regions, 0, 0); !ok || region.ID != "below" {
		t.Fatalf("non-overlap hit = %+v, want below", region)
	}
	if _, ok := Hit(regions, 20, 20); ok {
		t.Fatal("outside click must miss")
	}
	if id, ok := HitID(regions, 3, 3); !ok || id != "above" {
		t.Fatalf("HitID = %q/%v", id, ok)
	}
}

func TestHitRegionUnboundedAxis(t *testing.T) {
	region := HitRegion{ID: "row", X: 0, Y: 4, W: 0, H: 1}
	if !region.Contains(999, 4) {
		t.Fatal("W=0 must be unbounded horizontally")
	}
	if region.Contains(0, 5) {
		t.Fatal("H=1 must bound vertically")
	}
}

func TestListRegionsAndRowAt(t *testing.T) {
	list := List{
		Items:  []string{"a", "b", "c", "d", "e"},
		Height: 3,
		Header: "H",
		Offset: 1,
		RowID:  func(i int) string { return "row:" + string(rune('a'+i)) },
	}
	regions := ListRegions(list, 0, 0, 20)
	if len(regions) != 2 {
		t.Fatalf("regions = %d, want the 2-row window (height 3 - header)", len(regions))
	}
	if regions[0].Y != 1 || regions[0].Data != 1 || regions[0].ID != "row:b" {
		t.Fatalf("first region = %+v", regions[0])
	}
	if regions[1].Y != 2 || regions[1].Data != 2 {
		t.Fatalf("second region = %+v", regions[1])
	}
	if row, ok := list.RowAt(2); !ok || row != 2 {
		t.Fatalf("RowAt(2) = %d/%v, want row 2", row, ok)
	}
	if _, ok := list.RowAt(0); ok {
		t.Fatal("RowAt(0) must miss the header")
	}
	if _, ok := list.RowAt(9); ok {
		t.Fatal("RowAt past the window must miss")
	}
}

func TestVirtualListRegions(t *testing.T) {
	list := VirtualList{
		Rows:   []ListRow{{Text: "a", ID: "a"}, {Text: "b", ID: "b"}, {Text: "c", ID: "c"}},
		Height: 2,
	}
	regions := VirtualListRegions(list, 3, 5, 8)
	if len(regions) != 2 || regions[0].X != 3 || regions[0].Y != 5 {
		t.Fatalf("virtual regions = %+v", regions)
	}
	if regions[1].ID != "b" || regions[1].Data != 1 {
		t.Fatalf("second virtual region = %+v", regions[1])
	}
}

func TestTableRegionsAndRowAt(t *testing.T) {
	table := Table{
		Columns: []Column{{Title: "A", Width: 2}, {Title: "B", Width: 3}},
		Rows:    [][]string{{"x", "y"}, {"z", "w"}},
		RowID:   func(i int) string { return "r" },
	}
	regions := TableRegions(table, 0, 0)
	if len(regions) != 4 {
		t.Fatalf("cell regions = %d, want 4", len(regions))
	}
	if data, ok := regions[0].Data.([2]int); !ok || data != [2]int{0, 0} {
		t.Fatalf("first cell data = %#v", regions[0].Data)
	}
	if data, ok := regions[1].Data.([2]int); !ok || data != [2]int{0, 1} {
		t.Fatalf("second cell data = %#v", regions[1].Data)
	}
	if regions[1].X != 3 {
		t.Fatalf("second column x = %d, want 2 + 1 separator", regions[1].X)
	}
	if regions[2].Y != 2 {
		t.Fatalf("second row y = %d, want under the header", regions[2].Y)
	}
	if row, ok := table.RowAt(1); !ok || row != 0 {
		t.Fatalf("RowAt(1) = %d/%v, want the first data row", row, ok)
	}
	if _, ok := table.RowAt(0); ok {
		t.Fatal("RowAt(0) must miss the header")
	}
	if _, ok := table.RowAt(9); ok {
		t.Fatal("RowAt past the rows must miss")
	}
}

func TestApplyWheelClamps(t *testing.T) {
	if got := ApplyWheel(0, 100, 10, 3); got != 3 {
		t.Fatalf("ApplyWheel down = %d", got)
	}
	if got := ApplyWheel(5, 100, 10, -100); got != 0 {
		t.Fatalf("ApplyWheel up clamp = %d", got)
	}
	if got := ApplyWheel(80, 100, 10, 999); got != 90 {
		t.Fatalf("ApplyWheel bottom clamp = %d, want 90", got)
	}
	if got := ApplyWheel(4, 5, 10, 3); got != 0 {
		t.Fatalf("ApplyWheel fully visible = %d, want 0", got)
	}
}

func TestOnWheelMethods(t *testing.T) {
	list := List{Items: make([]string, 50), Height: 5}
	list.OnWheel(&pb.WheelEvent{Delta: 4})
	if list.Offset != 4 {
		t.Fatalf("list offset = %d", list.Offset)
	}
	list.OnWheel(nil)
	if list.Offset != 4 {
		t.Fatalf("nil wheel must be a no-op, offset = %d", list.Offset)
	}
	virtual := VirtualList{Rows: make([]ListRow, 50), Height: 5}
	virtual.OnWheel(&pb.WheelEvent{Delta: 3})
	if virtual.Offset != 3 {
		t.Fatalf("virtual offset = %d", virtual.Offset)
	}
	if got := TableScroll(0, 100, 10, 7); got != 7 {
		t.Fatalf("TableScroll = %d", got)
	}
}

func TestDragRectNormalization(t *testing.T) {
	cases := []struct {
		name         string
		sx, sy, x, y int
		want         Rect
	}{
		{"down-right", 2, 3, 5, 8, Rect{X: 2, Y: 3, W: 4, H: 6}},

		{"up-left", 5, 8, 2, 3, Rect{X: 2, Y: 3, W: 4, H: 6}},
		{"up-right", 5, 3, 2, 8, Rect{X: 2, Y: 3, W: 4, H: 6}},
		{"down-left", 2, 8, 5, 3, Rect{X: 2, Y: 3, W: 4, H: 6}},
	}
	for _, tc := range cases {
		drag := Drag{}
		drag.Begin(tc.sx, tc.sy, "left")
		drag.Update(tc.x, tc.y)
		rect, ok := drag.End()
		if !ok || rect != tc.want {
			t.Fatalf("%s drag = %+v/%v, want %+v", tc.name, rect, ok, tc.want)
		}
		if drag.Active {
			t.Fatalf("%s drag must end inactive", tc.name)
		}
	}
	if _, ok := (&Drag{}).End(); ok {
		t.Fatal("ending a drag that never began must report false")
	}
}

func TestSelectRangeAndListRange(t *testing.T) {
	if lo, hi := SelectRange(9, 2); lo != 2 || hi != 9 {
		t.Fatalf("SelectRange = (%d, %d)", lo, hi)
	}
	list := List{Items: []string{"a", "b", "c", "d", "e"}, Height: 3}
	first, last, ok := ListSelectRange(list, 2, 0)
	if !ok || first != 0 || last != 2 {
		t.Fatalf("ListSelectRange = (%d, %d, %v)", first, last, ok)
	}
	first, last, ok = ListSelectRange(list, 5, 7)
	if ok || first != 0 || last != 0 {
		t.Fatalf("off-window range = (%d, %d, %v)", first, last, ok)
	}
	virtual := VirtualList{Rows: make([]ListRow, 9), Height: 4}
	first, last, ok = VirtualListSelectRange(virtual, 0, 3)
	if !ok || first != 0 || last != 3 {
		t.Fatalf("VirtualListSelectRange = (%d, %d, %v)", first, last, ok)
	}
}

func TestClickTracker(t *testing.T) {
	base := time.Unix(0, 0)
	tracker := ClickTracker{Threshold: 400 * time.Millisecond}
	if count := tracker.Click(1, 1, base); count != 1 {
		t.Fatalf("first click = %d", count)
	}
	if count := tracker.Click(1, 1, base.Add(100*time.Millisecond)); count != 2 {
		t.Fatalf("double click = %d", count)
	}
	if count := tracker.Click(1, 1, base.Add(150*time.Millisecond)); count != 3 {
		t.Fatalf("triple click = %d", count)
	}
	if count := tracker.Click(1, 1, base.Add(160*time.Millisecond)); count != 1 {
		t.Fatalf("fourth click = %d, want a fresh gesture", count)
	}
	if count := tracker.Click(1, 1, base.Add(time.Second)); count != 1 {
		t.Fatalf("slow click = %d, want single", count)
	}
	if count := tracker.Click(1, 1, base.Add(time.Second+100*time.Millisecond)); count != 2 {
		t.Fatalf("new double = %d", count)
	}
	if count := tracker.Click(5, 5, base.Add(time.Second+150*time.Millisecond)); count != 1 {
		t.Fatalf("moved click = %d, want single", count)
	}
	zero := ClickTracker{}
	if count := zero.Click(0, 0, base); count != 1 {
		t.Fatalf("default threshold first click = %d", count)
	}
	if count := zero.Click(0, 0, base.Add(10*time.Millisecond)); count != 2 {
		t.Fatalf("default threshold double = %d", count)
	}
}

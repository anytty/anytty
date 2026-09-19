package widgets_test

import (
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/widgets"
)

func TestDistribute(t *testing.T) {
	cases := []struct {
		avail  int
		ratios []int
		want   []int
	}{
		{10, []int{1, 1}, []int{5, 5}},
		{10, []int{1, 2}, []int{3, 7}},
		{5, []int{1, 1, 1}, []int{1, 1, 3}},
		{2, []int{1, 1, 1}, []int{1, 1, 1}},
		{0, []int{1, 1}, []int{1, 1}},
		{7, nil, []int{}},
		{7, []int{1}, []int{7}},
	}
	for _, tc := range cases {
		got := widgets.Distribute(tc.avail, tc.ratios)
		if len(got) != len(tc.want) {
			t.Fatalf("Distribute(%d, %v) = %v, want %v", tc.avail, tc.ratios, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("Distribute(%d, %v) = %v, want %v", tc.avail, tc.ratios, got, tc.want)
			}
		}
	}
}

func TestSplitLayoutRects(t *testing.T) {
	layout := widgets.SplitLayout{Orient: "row", Weights: []int{1, 2}, Gap: 1}
	panes, dividers := layout.Rects(10, 3)
	if len(panes) != 2 || len(dividers) != 1 {
		t.Fatalf("panes=%d dividers=%d", len(panes), len(dividers))
	}
	if panes[0] != (widgets.Rect{X: 0, Y: 0, W: 3, H: 3}) {
		t.Errorf("pane[0] = %+v", panes[0])
	}
	if dividers[0] != (widgets.Rect{X: 3, Y: 0, W: 1, H: 3}) {
		t.Errorf("divider = %+v", dividers[0])
	}
	if panes[1].X != 4 || panes[1].W != 6 {
		t.Errorf("pane[1] = %+v", panes[1])
	}

	stacked := widgets.SplitLayout{Orient: "col", Weights: []int{1, 1}, Gap: 1}
	panes, dividers = stacked.Rects(4, 5)
	if panes[0].H != 2 || dividers[0].Y != 2 || panes[1].Y != 3 {
		t.Errorf("stacked panes=%+v dividers=%+v", panes, dividers)
	}
}

func TestChromeWidgetsBuild(t *testing.T) {
	title := widgets.TitleBar{
		ID: "title", Width: 20,
		Left:    []widgets.Segment{{Text: " pane "}},
		Buttons: []widgets.Button{{ID: "btn:close", Text: "x", Input: []string{"mouse"}}},
	}
	if box := title.Build().Build(); box.GetId() != "title" || len(box.GetChildren()) == 0 {
		t.Fatalf("TitleBar.Build() = %+v", box)
	}
	if line := title.Line(); len(line) != 20 {
		t.Fatalf("TitleBar.Line() = %q (%d cells)", line, len(line))
	}

	footer := widgets.Footer{
		ID: "footer", Width: 30, HasBadge: true,
		Badge:  widgets.Segment{Text: " CTRL "},
		Groups: []widgets.Segment{{Text: " P PANE"}, {Text: " T TAB"}},
		Right:  []widgets.Segment{{Text: " ws:main "}},
	}
	if line := footer.Line(); sdk.DisplayWidth(line) != 30 {
		t.Fatalf("Footer.Line() = %q (%d cells)", line, sdk.DisplayWidth(line))
	}
	if box := footer.Build().Build(); box.GetId() != "footer" {
		t.Fatalf("Footer.Build() = %+v", box)
	}

	picker := widgets.Picker{ID: "picker", Title: "Terminals", Width: 24, Rows: []widgets.PickerRow{
		{Text: "term-1", ID: "picker:0", Selectable: true},
		{Text: "term-2", ID: "picker:1", Selected: true, Selectable: true},
	}}
	box := picker.Build().Build()
	if box.GetId() != "picker" || len(box.GetChildren()) < 3 {
		t.Fatalf("Picker.Build() = %+v", box)
	}

	float := widgets.FloatingLayer{ID: "float-1", Title: "float", X: 3, Y: 2, Width: 20, Height: 6,
		Rows: []widgets.FrameRow{{Text: "body"}}}
	floatBox := float.Build().Build()
	if floatBox.GetPos().GetX() != 3 || floatBox.GetPos().GetY() != 2 {
		t.Fatalf("FloatingLayer pos = %+v", floatBox.GetPos())
	}
	collapsed := widgets.FloatingLayer{ID: "float-1", Title: "float", Width: 20, Height: 6, Collapsed: true}
	if size := collapsed.Build().Build().GetSize(); size.GetHeight() != 1 {
		t.Fatalf("collapsed FloatingLayer height = %d", size.GetHeight())
	}

	toast := widgets.Toast{ID: "toast", Text: "saved", Style: "fg:#fff"}
	if box := toast.Build().Build(); box.GetId() != "toast" || box.GetStyle() != "fg:#fff" {
		t.Fatalf("Toast.Build() = %+v", box)
	}
}

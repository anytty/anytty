package kernel

import (
	"reflect"
	"testing"
)

func TestLayoutViewportSafety(t *testing.T) {
	root := &Node{ID: "root", Children: []Node{{ID: "a"}}}
	tests := []struct {
		name          string
		root          *Node
		width, height int
	}{
		{"nil root", nil, 10, 10},
		{"zero width", root, 0, 10},
		{"zero height", root, 10, 0},
		{"zero both", root, 0, 0},
		{"negative both", root, -5, -7},
		{"invisible root", &Node{ID: "root", Visible: vp(false)}, 10, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Layout(tt.root, tt.width, tt.height)
			if len(f.Rects) != 0 {
				t.Fatalf("Rects = %v, want empty", f.Rects)
			}
			if len(f.Lines) != 0 {
				t.Fatalf("Lines = %v, want empty", f.Lines)
			}
			if f.HasCursor {
				t.Fatalf("HasCursor = true, want false")
			}
			if got := f.Hit(0, 0); got != "" {
				t.Fatalf("Hit(0,0) = %q, want empty", got)
			}
		})
	}
}

func TestLayoutFlow(t *testing.T) {
	tests := []struct {
		name          string
		root          *Node
		width, height int
		want          map[string]Rect
	}{
		{
			name:   "single empty child fills",
			root:   &Node{ID: "root", Children: []Node{{ID: "a"}}},
			width:  10,
			height: 6,
			want: map[string]Rect{
				"root": {0, 0, 10, 6},
				"a":    {0, 0, 10, 6},
			},
		},
		{
			name:   "col equal split",
			root:   &Node{ID: "root", Children: []Node{{ID: "a"}, {ID: "b"}}},
			width:  10,
			height: 6,
			want: map[string]Rect{
				"root": {0, 0, 10, 6},
				"a":    {0, 0, 10, 3},
				"b":    {0, 3, 10, 3},
			},
		},
		{
			name:   "col equal split remainder to last",
			root:   &Node{ID: "root", Children: []Node{{ID: "a"}, {ID: "b"}, {ID: "c"}}},
			width:  10,
			height: 7,
			want: map[string]Rect{
				"root": {0, 0, 10, 7},
				"a":    {0, 0, 10, 2},
				"b":    {0, 2, 10, 2},
				"c":    {0, 4, 10, 3},
			},
		},
		{
			name:   "row equal split remainder to last",
			root:   &Node{ID: "root", Flow: FlowRow, Children: []Node{{ID: "a"}, {ID: "b"}, {ID: "c"}}},
			width:  11,
			height: 4,
			want: map[string]Rect{
				"root": {0, 0, 11, 4},
				"a":    {0, 0, 3, 4},
				"b":    {3, 0, 3, 4},
				"c":    {6, 0, 5, 4},
			},
		},
		{
			name: "intrinsic main and cross then stretch",
			root: &Node{ID: "root", Children: []Node{
				{ID: "a", Content: &Content{Lines: []string{"hello"}}},
				{ID: "b"},
			}},
			width:  10,
			height: 6,
			want: map[string]Rect{
				"root": {0, 0, 10, 6},
				"a":    {0, 0, 5, 1},
				"b":    {0, 1, 10, 5},
			},
		},
		{
			name: "intrinsic size is content only",
			root: &Node{ID: "root", Children: []Node{
				{ID: "a", Content: &Content{Lines: []string{"hi"}}},
			}},
			width:  10,
			height: 4,
			want: map[string]Rect{
				"root": {0, 0, 10, 4},
				"a":    {0, 0, 2, 1},
			},
		},
		{
			name: "self content stretches",
			root: &Node{ID: "root", Children: []Node{
				{ID: "a", Content: &Content{Self: "terminal:x"}},
			}},
			width:  10,
			height: 4,
			want: map[string]Rect{
				"root": {0, 0, 10, 4},
				"a":    {0, 0, 10, 4},
			},
		},
		{
			name: "row cross intrinsic",
			root: &Node{ID: "root", Flow: FlowRow, Children: []Node{
				{ID: "a", Content: &Content{Lines: []string{"ab", "cd"}}},
				{ID: "b"},
			}},
			width:  10,
			height: 4,
			want: map[string]Rect{
				"root": {0, 0, 10, 4},
				"a":    {0, 0, 2, 2},
				"b":    {2, 0, 8, 4},
			},
		},
		{
			name: "fixed size wins over intrinsic",
			root: &Node{ID: "root", Children: []Node{
				{ID: "a", Size: Size{Width: 7, Height: 2}, Content: &Content{Lines: []string{"hello"}}},
				{ID: "b"},
			}},
			width:  10,
			height: 6,
			want: map[string]Rect{
				"root": {0, 0, 10, 6},
				"a":    {0, 0, 7, 2},
				"b":    {0, 2, 10, 4},
			},
		},
		{
			name: "negative size treated as unset",
			root: &Node{ID: "root", Children: []Node{
				{ID: "a", Size: Size{Width: -3, Height: -2}},
			}},
			width:  6,
			height: 4,
			want: map[string]Rect{
				"root": {0, 0, 6, 4},
				"a":    {0, 0, 6, 4},
			},
		},
		{
			name: "multiline text intrinsic height",
			root: &Node{ID: "root", Children: []Node{
				{ID: "a", Content: &Content{Text: "a\nb\nc"}},
				{ID: "b"},
			}},
			width:  10,
			height: 5,
			want: map[string]Rect{
				"root": {0, 0, 10, 5},
				"a":    {0, 0, 1, 3},
				"b":    {0, 3, 10, 2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Layout(tt.root, tt.width, tt.height)
			if !reflect.DeepEqual(f.Rects, tt.want) {
				t.Fatalf("Rects =\n%v\nwant\n%v", f.Rects, tt.want)
			}
		})
	}
}

func TestLayoutFlex(t *testing.T) {
	tests := []struct {
		name          string
		root          *Node
		width, height int
		want          map[string]Rect
	}{
		{
			name: "flex equal",
			root: &Node{ID: "root", Flow: FlowRow, Children: []Node{
				{ID: "a", Size: Size{Flex: 1}},
				{ID: "b", Size: Size{Flex: 1}},
			}},
			width:  10,
			height: 2,
			want: map[string]Rect{
				"root": {0, 0, 10, 2},
				"a":    {0, 0, 5, 2},
				"b":    {5, 0, 5, 2},
			},
		},
		{
			name: "flex remainder to last",
			root: &Node{ID: "root", Flow: FlowRow, Children: []Node{
				{ID: "a", Size: Size{Flex: 1}},
				{ID: "b", Size: Size{Flex: 1}},
			}},
			width:  11,
			height: 2,
			want: map[string]Rect{
				"root": {0, 0, 11, 2},
				"a":    {0, 0, 5, 2},
				"b":    {5, 0, 6, 2},
			},
		},
		{
			name: "flex ratio",
			root: &Node{ID: "root", Flow: FlowRow, Children: []Node{
				{ID: "a", Size: Size{Flex: 1}},
				{ID: "b", Size: Size{Flex: 2}},
			}},
			width:  10,
			height: 2,
			want: map[string]Rect{
				"root": {0, 0, 10, 2},
				"a":    {0, 0, 3, 2},
				"b":    {3, 0, 7, 2},
			},
		},
		{
			name: "flex grows from intrinsic base",
			root: &Node{ID: "root", Flow: FlowRow, Children: []Node{
				{ID: "a", Size: Size{Flex: 1}, Content: &Content{Lines: []string{"ab"}}},
				{ID: "b", Size: Size{Flex: 2}},
			}},
			width:  12,
			height: 3,
			want: map[string]Rect{
				"root": {0, 0, 12, 3},
				"a":    {0, 0, 5, 1},
				"b":    {5, 0, 7, 3},
			},
		},
		{
			name: "fixed plus stretch",
			root: &Node{ID: "root", Flow: FlowRow, Children: []Node{
				{ID: "a", Size: Size{Width: 3}},
				{ID: "b"},
			}},
			width:  10,
			height: 2,
			want: map[string]Rect{
				"root": {0, 0, 10, 2},
				"a":    {0, 0, 3, 2},
				"b":    {3, 0, 7, 2},
			},
		},
		{
			name: "fixed plus two stretch equal",
			root: &Node{ID: "root", Flow: FlowRow, Children: []Node{
				{ID: "a", Size: Size{Width: 4}},
				{ID: "b"},
				{ID: "c"},
			}},
			width:  10,
			height: 2,
			want: map[string]Rect{
				"root": {0, 0, 10, 2},
				"a":    {0, 0, 4, 2},
				"b":    {4, 0, 3, 2},
				"c":    {7, 0, 3, 2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Layout(tt.root, tt.width, tt.height)
			if !reflect.DeepEqual(f.Rects, tt.want) {
				t.Fatalf("Rects =\n%v\nwant\n%v", f.Rects, tt.want)
			}
		})
	}
}

func TestLayoutStack(t *testing.T) {
	root := &Node{ID: "root", Flow: FlowStack, Children: []Node{
		{ID: "a"},
		{ID: "b"},
		{ID: "c", Size: Size{Width: 3, Height: 2}},
	}}
	f := Layout(root, 8, 4)
	want := map[string]Rect{
		"root": {0, 0, 8, 4},
		"a":    {0, 0, 8, 4},
		"b":    {0, 0, 8, 4},
		"c":    {0, 0, 3, 2},
	}
	if !reflect.DeepEqual(f.Rects, want) {
		t.Fatalf("Rects =\n%v\nwant\n%v", f.Rects, want)
	}
}

func TestLayoutPosition(t *testing.T) {
	root := &Node{ID: "root", Children: []Node{
		{ID: "a"},
		{
			ID:      "p",
			Pos:     &Pos{X: 2, Y: 1},
			Size:    Size{Width: 3, Height: 2},
			Content: &Content{Lines: []string{"pp"}},
		},
	}}
	f := Layout(root, 10, 4)
	if got, want := f.Rects["a"], (Rect{0, 0, 10, 4}); got != want {
		t.Fatalf("a rect = %v, want %v (pos must not consume flow space)", got, want)
	}
	if got, want := f.Rects["p"], (Rect{2, 1, 3, 2}); got != want {
		t.Fatalf("p rect = %v, want %v", got, want)
	}
	if len(f.OverlayFrames) != 1 {
		t.Fatalf("OverlayFrames = %d, want 1", len(f.OverlayFrames))
	}
	overlay := f.OverlayFrames[0]
	if got, want := overlay.Rects["p"], (Rect{2, 1, 3, 2}); got != want {
		t.Fatalf("overlay p rect = %v, want %v", got, want)
	}
	for _, line := range f.Lines {
		if line.Text == "pp" {
			t.Fatalf("overlay content leaked into base frame lines: %v", f.Lines)
		}
	}
	if len(overlay.Lines) != 1 || overlay.Lines[0].X != 2 || overlay.Lines[0].Y != 1 {
		t.Fatalf("overlay lines = %v, want one line at (2,1)", overlay.Lines)
	}
	if got, ok := f.Rect("p"); !ok || got != (Rect{2, 1, 3, 2}) {
		t.Fatalf("Rect(p) = %v,%v", got, ok)
	}
}

func TestLayoutPositionRelativeToParentRect(t *testing.T) {
	root := &Node{ID: "root", Children: []Node{
		{ID: "p", Pos: &Pos{X: 1, Y: 1}, Size: Size{Width: 2, Height: 2}},
	}}
	f := Layout(root, 10, 4)
	if got, want := f.Rects["p"], (Rect{1, 1, 2, 2}); got != want {
		t.Fatalf("p rect = %v, want %v (parent rect origin, no border inset)", got, want)
	}
}

func TestLayoutOverlayZOrder(t *testing.T) {
	root := &Node{ID: "root", Children: []Node{
		{ID: "a"},
		{ID: "p1", Pos: &Pos{}, Size: Size{Width: 6, Height: 3}},
		{ID: "p2", Pos: &Pos{}, Size: Size{Width: 6, Height: 3}},
	}}
	f := Layout(root, 6, 3)
	if len(f.OverlayFrames) != 2 {
		t.Fatalf("OverlayFrames = %d, want 2", len(f.OverlayFrames))
	}
	if _, ok := f.OverlayFrames[0].Rects["p1"]; !ok {
		t.Fatalf("first overlay frame missing p1: %v", f.OverlayFrames[0].Rects)
	}
	if _, ok := f.OverlayFrames[1].Rects["p2"]; !ok {
		t.Fatalf("second overlay frame missing p2: %v", f.OverlayFrames[1].Rects)
	}
	if got := f.Hit(1, 1); got != "p2" {
		t.Fatalf("Hit(1,1) = %q, want p2 (later overlay on top)", got)
	}
}

func TestLayoutNestedOverlayLiftedToRoot(t *testing.T) {
	root := &Node{ID: "root", Children: []Node{
		{ID: "a", Children: []Node{
			{
				ID:      "p",
				Pos:     &Pos{X: 1, Y: 1},
				Size:    Size{Width: 2, Height: 2},
				Content: &Content{Lines: []string{"x"}},
			},
		}},
	}}
	f := Layout(root, 10, 6)
	if len(f.OverlayFrames) != 1 {
		t.Fatalf("OverlayFrames = %d, want 1 (nested pos lifted)", len(f.OverlayFrames))
	}
	if got, want := f.Rects["p"], (Rect{1, 1, 2, 2}); got != want {
		t.Fatalf("root Rects[p] = %v, want %v", got, want)
	}
	for _, line := range f.Lines {
		if line.Text == "x" {
			t.Fatalf("overlay content leaked into root lines: %v", f.Lines)
		}
	}
	if len(f.OverlayFrames[0].Lines) != 1 {
		t.Fatalf("overlay lines = %v, want 1 line", f.OverlayFrames[0].Lines)
	}
	if got := f.Hit(1, 1); got != "p" {
		t.Fatalf("Hit(1,1) = %q, want p", got)
	}
}

func TestLayoutVisibleFalse(t *testing.T) {
	root := &Node{ID: "root", Children: []Node{
		{ID: "a", Content: &Content{Lines: []string{"aa"}}},
		{ID: "b", Visible: vp(false), Content: &Content{Lines: []string{"bb"}}},
		{ID: "c"},
	}}
	f := Layout(root, 10, 6)
	if _, ok := f.Rect("b"); ok {
		t.Fatalf("invisible node b must not have a rect: %v", f.Rects)
	}
	if got, want := f.Rects["a"], (Rect{0, 0, 2, 1}); got != want {
		t.Fatalf("a rect = %v, want %v", got, want)
	}
	if got, want := f.Rects["c"], (Rect{0, 1, 10, 5}); got != want {
		t.Fatalf("c rect = %v, want %v (siblings must close the gap)", got, want)
	}
	for _, line := range f.Lines {
		if line.Text == "bb" {
			t.Fatalf("invisible node rendered: %v", f.Lines)
		}
	}
}

func TestLayoutInvisiblePosSkipped(t *testing.T) {
	root := &Node{ID: "root", Children: []Node{
		{ID: "p", Pos: &Pos{}, Size: Size{Width: 2, Height: 2}, Visible: vp(false)},
	}}
	f := Layout(root, 4, 2)
	if len(f.OverlayFrames) != 0 {
		t.Fatalf("OverlayFrames = %d, want 0", len(f.OverlayFrames))
	}
	if _, ok := f.Rect("p"); ok {
		t.Fatalf("invisible pos node must not have a rect: %v", f.Rects)
	}
}

func TestLayoutZeroAndNegativeSafety(t *testing.T) {
	root := &Node{
		ID:      "root",
		Content: &Content{Lines: []string{"abcdef", "gh"}},
		Cursor:  &Cursor{Row: -2, Col: -1},
		Children: []Node{
			{ID: "a", Size: Size{Width: -1, Height: -5}, Content: &Content{Text: "x"}},
			{ID: "p", Pos: &Pos{X: -3, Y: -4}, Size: Size{Width: -1, Height: -1}, Content: &Content{Lines: []string{"y"}}},
		},
	}
	for _, size := range [][2]int{{3, 2}, {1, 1}, {0, 0}, {-4, -4}} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Layout(%v) panicked: %v", size, r)
				}
			}()
			f := Layout(root, size[0], size[1])
			for id, rect := range f.Rects {
				if rect.Width < 0 || rect.Height < 0 {
					t.Fatalf("negative rect for %q: %v", id, rect)
				}
			}
			for _, overlay := range f.OverlayFrames {
				for id, rect := range overlay.Rects {
					if rect.Width < 0 || rect.Height < 0 {
						t.Fatalf("negative overlay rect for %q: %v", id, rect)
					}
				}
			}
			_ = f.Hit(0, 0)
			_ = f.Hit(-10, 99)
		}()
	}
}

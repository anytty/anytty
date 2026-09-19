package kernel

import "testing"

func TestCursorContentOffset(t *testing.T) {
	root := &Node{
		ID:     "root",
		Style:  "muted",
		Cursor: &Cursor{Row: 1, Col: 2, Shape: "bar"},
	}
	f := Layout(root, 10, 5)
	if !f.HasCursor {
		t.Fatalf("HasCursor = false, want true")
	}
	if got, want := f.CursorRect, (Rect{2, 1, 1, 1}); got != want {
		t.Fatalf("CursorRect = %v, want %v (box rect origin, no inset)", got, want)
	}
	if f.CursorShape != "bar" {
		t.Fatalf("CursorShape = %q, want bar", f.CursorShape)
	}
}

func TestCursorAtRectOrigin(t *testing.T) {
	root := &Node{ID: "root", Children: []Node{
		{ID: "top", Size: Size{Height: 2}},
		{ID: "box", Size: Size{Height: 2}, Cursor: &Cursor{Row: 1, Col: 1}},
	}}
	f := Layout(root, 8, 6)
	if got, want := f.CursorRect, (Rect{1, 3, 1, 1}); got != want {
		t.Fatalf("CursorRect = %v, want %v", got, want)
	}
	if !f.HasCursor {
		t.Fatalf("HasCursor = false, want true")
	}
}

func TestCursorVisibility(t *testing.T) {
	tests := []struct {
		name   string
		cursor *Cursor
		want   bool
	}{
		{"absent", nil, false},
		{"default visible", &Cursor{Row: 1, Col: 1}, true},
		{"explicit false", &Cursor{Row: 1, Col: 1, Visible: vp(false)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Layout(&Node{ID: "root", Cursor: tt.cursor}, 4, 3)
			if f.HasCursor != tt.want {
				t.Fatalf("HasCursor = %v, want %v", f.HasCursor, tt.want)
			}
			if !tt.want && f.CursorRect != (Rect{}) {
				t.Fatalf("CursorRect = %v, want zero", f.CursorRect)
			}
		})
	}
}

func TestCursorLastDeclarationWins(t *testing.T) {
	root := &Node{
		ID:     "root",
		Cursor: &Cursor{Row: 0, Col: 0},
		Children: []Node{
			{ID: "a", Cursor: &Cursor{Row: 1, Col: 1}},
			{ID: "p", Pos: &Pos{X: 3, Y: 2}, Size: Size{Width: 2, Height: 2}, Cursor: &Cursor{Row: 1, Col: 1}},
		},
	}
	f := Layout(root, 10, 5)
	if got, want := f.CursorRect, (Rect{4, 3, 1, 1}); got != want {
		t.Fatalf("CursorRect = %v, want %v (last declared cursor wins)", got, want)
	}
}

func TestCursorHiddenDoesNotOverride(t *testing.T) {
	root := &Node{ID: "root", Children: []Node{
		{ID: "a", Cursor: &Cursor{Row: 0, Col: 1}},
		{ID: "b", Cursor: &Cursor{Row: 2, Col: 2, Visible: vp(false)}},
	}}
	f := Layout(root, 8, 4)
	if got, want := f.CursorRect, (Rect{1, 0, 1, 1}); got != want {
		t.Fatalf("CursorRect = %v, want %v (hidden cursor must not win)", got, want)
	}
}

func TestCursorZeroSizeBox(t *testing.T) {
	f := Layout(&Node{ID: "root", Cursor: &Cursor{Row: 1, Col: 1}}, 0, 0)
	if f.HasCursor {
		t.Fatalf("HasCursor = true, want false for empty viewport")
	}
}

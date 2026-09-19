package kernel

import "testing"

func TestHitSmallestArea(t *testing.T) {
	root := &Node{ID: "root", Children: []Node{{ID: "a"}}}
	f := Layout(root, 10, 5)
	tests := []struct {
		name string
		x, y int
		want string
	}{
		{"top left belongs to child", 0, 0, "a"},
		{"content area belongs to child", 4, 2, "a"},
		{"bottom right corner", 9, 4, "a"},
		{"outside viewport", 10, 0, ""},
		{"negative coords", -1, 2, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.Hit(tt.x, tt.y); got != tt.want {
				t.Fatalf("Hit(%d,%d) = %q, want %q", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestHitTieLaterDeclarationWins(t *testing.T) {
	root := &Node{ID: "root", Children: []Node{{ID: "a"}}}
	f := Layout(root, 4, 2)
	if got := f.Hit(1, 1); got != "a" {
		t.Fatalf("Hit(1,1) = %q, want a (same area, later declaration)", got)
	}
}

func TestHitSkipEmptyID(t *testing.T) {
	root := &Node{Children: []Node{{ID: "c"}}}
	f := Layout(root, 4, 2)
	if got := f.Hit(0, 0); got != "c" {
		t.Fatalf("Hit(0,0) = %q, want c (empty parent id skipped)", got)
	}
	noIDs := &Node{Content: &Content{Lines: []string{"x"}}}
	if got := HitTest(noIDs, 4, 2, 0, 0); got != "" {
		t.Fatalf("HitTest = %q, want empty", got)
	}
}

func TestHitSkipInvisible(t *testing.T) {
	root := &Node{ID: "root", Children: []Node{
		{ID: "a"},
		{ID: "b", Visible: vp(false)},
	}}
	f := Layout(root, 6, 4)
	if got := f.Hit(2, 3); got != "a" {
		t.Fatalf("Hit(2,3) = %q, want a (invisible sibling skipped)", got)
	}
}

func TestHitOverlayAboveFlow(t *testing.T) {
	root := &Node{ID: "root", Children: []Node{
		{ID: "a"},
		{ID: "p", Pos: &Pos{}, Size: Size{Width: 6, Height: 3}},
	}}
	f := Layout(root, 6, 3)
	if got := f.Hit(1, 1); got != "p" {
		t.Fatalf("Hit(1,1) = %q, want p (pos subtree above flow)", got)
	}
}

func TestHitZeroAreaViewport(t *testing.T) {
	root := &Node{ID: "root"}
	if got := HitTest(root, 0, 5, 0, 0); got != "" {
		t.Fatalf("HitTest with zero viewport = %q, want empty", got)
	}
}

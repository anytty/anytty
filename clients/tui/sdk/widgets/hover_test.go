package widgets

import "testing"

func TestHoverEnterLeaveAndStyle(t *testing.T) {
	hover := Hover{
		Style: "hover",
		Regions: []HitRegion{
			{ID: "a", X: 0, Y: 0, W: 4, H: 1},
			{ID: "b", X: 4, Y: 0, W: 4, H: 1},
		},
	}
	if id := hover.Update(1, 0); id != "a" {
		t.Fatalf("hover id = %q, want a", id)
	}
	if got := hover.StyleFor("a"); got != "hover" {
		t.Fatalf("StyleFor(hovered) = %q", got)
	}
	if got := hover.StyleFor("b"); got != "" {
		t.Fatalf("StyleFor(idle) = %q, want the base style", got)
	}
	if id := hover.Update(5, 0); id != "b" {
		t.Fatalf("moved hover id = %q, want b", id)
	}
	if got := hover.StyleFor("a"); got != "" {
		t.Fatalf("previous target style = %q, want base", got)
	}
	if id := hover.Update(20, 0); id != "" {
		t.Fatalf("leaving hover id = %q, want empty", id)
	}
	if hover.Hovered() != "" || hover.StyleFor("b") != "" {
		t.Fatalf("left hover leaked: %+v", hover)
	}
	hover.Clear()
	if hover.CurrentStyle != "" {
		t.Fatalf("Clear left %q", hover.CurrentStyle)
	}
}

func TestHoverNodeFallback(t *testing.T) {
	hover := Hover{
		Node:    "surface",
		Style:   "hover",
		Regions: []HitRegion{{X: 0, Y: 0, W: 10, H: 10}},
	}
	if id := hover.Update(2, 2); id != "surface" {
		t.Fatalf("node fallback = %q", id)
	}
	if hover.StyleFor("surface") != "hover" {
		t.Fatalf("node style = %q", hover.StyleFor("surface"))
	}
}

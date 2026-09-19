package kernel

import (
	"strings"
	"testing"
)

// TestRenderContentUsesFullRect pins the M1 semantics: the content area is
// the full box rect, with no border inset of any kind.
func TestRenderContentUsesFullRect(t *testing.T) {
	root := &Node{ID: "root", Content: &Content{Lines: []string{"ab", "cd"}}}
	f := Layout(root, 5, 3)
	want := []Line{
		{X: 0, Y: 0, Text: "ab"},
		{X: 0, Y: 1, Text: "cd"},
	}
	if len(f.Lines) != len(want) {
		t.Fatalf("lines = %v, want %v", f.Lines, want)
	}
	for i, line := range f.Lines {
		if line != want[i] {
			t.Fatalf("line %d = %+v, want %+v", i, line, want[i])
		}
	}
}

func TestRenderContentClippedToRect(t *testing.T) {
	root := &Node{ID: "root", Content: &Content{Lines: []string{"你好世界", "toolong"}}}
	f := Layout(root, 6, 4)
	want := map[int]string{0: "你好世", 1: "toolon"}
	for y, text := range want {
		found := false
		for _, line := range f.Lines {
			if line.Y != y || line.X != 0 {
				continue
			}
			found = true
			if line.Text != text {
				t.Fatalf("line %d = %q, want %q", y, line.Text, text)
			}
			if DisplayWidth(line.Text) > 6 {
				t.Fatalf("line %d wider than rect: %q", y, line.Text)
			}
		}
		if !found {
			t.Fatalf("missing rendered line at Y=%d in %v", y, f.Lines)
		}
	}
}

func TestRenderContentHeightClipped(t *testing.T) {
	root := &Node{ID: "root", Content: &Content{Lines: []string{"a", "b", "c"}}}
	f := Layout(root, 5, 2)
	rendered := 0
	for _, line := range f.Lines {
		if line.X == 0 && line.Text != "" {
			rendered++
		}
	}
	if rendered != 2 {
		t.Fatalf("rendered content lines = %d, want 2 (%v)", rendered, f.Lines)
	}
}

func TestRenderDegenerateRectKeepsTextClipping(t *testing.T) {
	t.Run("one column", func(t *testing.T) {
		f := Layout(&Node{ID: "root", Content: &Content{Lines: []string{"ab", "cd", "ef"}}}, 1, 3)
		if len(f.Lines) != 3 {
			t.Fatalf("lines = %v, want 3 single-cell runs", f.Lines)
		}
		for _, line := range f.Lines {
			if line.Text != "a" && line.Text != "c" && line.Text != "e" {
				t.Fatalf("line = %q, want one cell of content", line.Text)
			}
		}
	})
	t.Run("one row", func(t *testing.T) {
		f := Layout(&Node{ID: "root", Content: &Content{Lines: []string{"abc", "def"}}}, 5, 1)
		if len(f.Lines) != 1 || f.Lines[0].Text != "abc" {
			t.Fatalf("lines = %v, want only the first content row", f.Lines)
		}
	})
}

func TestRenderNoANSIBytes(t *testing.T) {
	root := &Node{
		ID:      "root",
		Style:   "fg:#ffffff;bold",
		Content: &Content{Lines: []string{"你好", "world"}},
		Cursor:  &Cursor{Row: 0, Col: 1, Shape: "block"},
		Children: []Node{
			{ID: "p", Pos: &Pos{X: 1, Y: 1}, Content: &Content{Lines: []string{"x"}}},
		},
	}
	f := Layout(root, 12, 6)
	var walk func(Frame)
	walk = func(frame Frame) {
		for _, line := range frame.Lines {
			if strings.ContainsRune(line.Text, '\x1b') || strings.ContainsRune(line.Style, '\x1b') {
				t.Fatalf("ANSI escape found in line %+v", line)
			}
		}
		for _, overlay := range frame.OverlayFrames {
			walk(overlay)
		}
	}
	walk(f)
}

func TestRenderStyleIsOpaque(t *testing.T) {
	root := &Node{ID: "root", Style: "status", Content: &Content{Lines: []string{"x"}}}
	f := Layout(root, 8, 3)
	if len(f.Lines) == 0 {
		t.Fatal("no lines rendered")
	}
	for _, line := range f.Lines {
		if line.Style != "status" {
			t.Fatalf("line style = %q, want the opaque status style", line.Style)
		}
	}
}

func TestRenderExplicitStylePassesThrough(t *testing.T) {
	root := &Node{ID: "root", Style: "fg:#a78bfa;bg:#161823;bold", Content: &Content{Lines: []string{"x"}}}
	f := Layout(root, 8, 3)
	if len(f.Lines) != 1 || f.Lines[0].Style != "fg:#a78bfa;bg:#161823;bold" {
		t.Fatalf("lines = %+v, want the explicit style untouched", f.Lines)
	}
}

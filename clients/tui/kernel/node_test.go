package kernel

import (
	"reflect"
	"testing"
)

func vp(v bool) *bool { return &v }

func TestNodeDefaults(t *testing.T) {
	var n Node
	if n.ID != "" {
		t.Fatalf("id default = %q, want empty", n.ID)
	}
	if !n.IsVisible() {
		t.Fatalf("visible default = false, want true")
	}
	if got := n.EffectiveFlow(); got != FlowCol {
		t.Fatalf("flow default = %q, want %q", got, FlowCol)
	}
	if n.Pos != nil || n.Content != nil || n.Cursor != nil {
		t.Fatalf("nil defaults expected, got pos=%v content=%v cursor=%v", n.Pos, n.Content, n.Cursor)
	}
	if n.Input != nil {
		t.Fatalf("input default = %v, want nil", n.Input)
	}
	if n.Focused {
		t.Fatalf("focused default = true, want false")
	}
	if n.CursorVisible() {
		t.Fatalf("CursorVisible() = true, want false")
	}
	if lines := n.ContentLines(); lines != nil {
		t.Fatalf("ContentLines() = %v, want nil", lines)
	}
	if w, h := n.IntrinsicSize(); w != 0 || h != 0 {
		t.Fatalf("IntrinsicSize() = %d,%d, want 0,0", w, h)
	}
}

func TestNodeEffectiveFlow(t *testing.T) {
	tests := []struct {
		in   Flow
		want Flow
	}{
		{"", FlowCol},
		{FlowCol, FlowCol},
		{FlowRow, FlowRow},
		{FlowStack, FlowStack},
		{Flow("bogus"), FlowCol},
	}
	for _, tt := range tests {
		n := Node{Flow: tt.in}
		if got := n.EffectiveFlow(); got != tt.want {
			t.Errorf("EffectiveFlow(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNodeVisibility(t *testing.T) {
	tests := []struct {
		name string
		flag *bool
		want bool
	}{
		{"default nil", nil, true},
		{"explicit true", vp(true), true},
		{"explicit false", vp(false), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := Node{Visible: tt.flag}
			if got := n.IsVisible(); got != tt.want {
				t.Fatalf("IsVisible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeCursorVisible(t *testing.T) {
	tests := []struct {
		name   string
		cursor *Cursor
		want   bool
	}{
		{"no cursor", nil, false},
		{"default visible", &Cursor{Row: 1}, true},
		{"explicit true", &Cursor{Visible: vp(true)}, true},
		{"explicit false", &Cursor{Visible: vp(false)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := Node{Cursor: tt.cursor}
			if got := n.CursorVisible(); got != tt.want {
				t.Fatalf("CursorVisible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeIntrinsicSize(t *testing.T) {
	tests := []struct {
		name  string
		node  Node
		wantW int
		wantH int
	}{
		{"empty node", Node{}, 0, 0},
		{"self only", Node{Content: &Content{Self: "terminal:x"}}, 0, 0},
		{"text single line", Node{Content: &Content{Text: "hello"}}, 5, 1},
		{"text multiline", Node{Content: &Content{Text: "ab\ncde"}}, 3, 2},
		{"text trailing newline", Node{Content: &Content{Text: "a\n"}}, 1, 2},
		{"lines cjk", Node{Content: &Content{Lines: []string{"你好", "x"}}}, 4, 2},
		{"empty text", Node{Content: &Content{Text: ""}}, 0, 0},
		{"empty line", Node{Content: &Content{Lines: []string{""}}}, 0, 1},
		{"cjk width", Node{Content: &Content{Lines: []string{"你好"}}}, 4, 1},
		{"widest line wins", Node{Content: &Content{Lines: []string{"abc", "de"}}}, 3, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h := tt.node.IntrinsicSize()
			if w != tt.wantW || h != tt.wantH {
				t.Fatalf("IntrinsicSize() = %d,%d, want %d,%d", w, h, tt.wantW, tt.wantH)
			}
		})
	}
}

func TestNodeContentLinesPrecedence(t *testing.T) {
	n := Node{Content: &Content{Text: "ignored", Lines: []string{"a", "b"}}}
	if got, want := n.ContentLines(), []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ContentLines() = %v, want %v", got, want)
	}
}

func TestNodeAcceptsInput(t *testing.T) {
	n := Node{Input: []string{"key", "mouse"}}
	for _, kind := range []string{"key", "mouse"} {
		if !n.AcceptsInput(kind) {
			t.Fatalf("AcceptsInput(%q) = false, want true", kind)
		}
	}
	for _, kind := range []string{"paste", "wheel", ""} {
		if n.AcceptsInput(kind) {
			t.Fatalf("AcceptsInput(%q) = true, want false", kind)
		}
	}
}

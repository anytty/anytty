package widgets

import (
	"strings"
	"testing"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func TestModalPositionAndBackdrop(t *testing.T) {
	modal := Modal{
		ID: "confirm", Title: "Confirm", Width: 20, Height: 6,
		Center: true, ParentWidth: 80, ParentHeight: 24,
		Backdrop: true, BackdropStyle: "dim",
		Rows: []FrameRow{{Text: "ok", ID: "confirm:ok", Input: []string{"mouse"}}},
	}
	if x, y := modal.Position(); x != 30 || y != 9 {
		t.Fatalf("centered Position = (%d, %d), want (30, 9)", x, y)
	}
	root := modal.Build().Build()
	backdrop := findBox(root, "confirm:backdrop")
	if backdrop == nil || backdrop.GetStyle() != "dim" {
		t.Fatalf("backdrop = %+v", backdrop)
	}
	if backdrop.GetSize().GetWidth() != 80 || backdrop.GetSize().GetHeight() != 24 {
		t.Fatalf("backdrop size = %+v", backdrop.GetSize())
	}
	layer := findBox(root, "confirm")
	if layer == nil || layer.GetPos().GetX() != 30 || layer.GetPos().GetY() != 9 {
		t.Fatalf("layer = %+v", layer)
	}
	if row := findBox(root, "confirm:ok"); row == nil {
		t.Fatal("modal row id lost")
	}
}

func TestModalPositionClampsAndOffsets(t *testing.T) {
	modal := Modal{Width: 60, Height: 40, Center: true, ParentWidth: 40, ParentHeight: 20}
	if x, y := modal.Position(); x != 0 || y != 0 {
		t.Fatalf("oversized centered modal = (%d, %d), want (0, 0)", x, y)
	}
	offset := Modal{X: 3, Y: 2}
	if x, y := offset.Position(); x != 3 || y != 2 {
		t.Fatalf("offset modal = (%d, %d)", x, y)
	}
	if root := offset.Build().Build(); findBox(root, "confirm:backdrop") != nil {
		t.Fatal("backdrop must be opt-in")
	}
}

func TestMenuMoveSkipsAndSelects(t *testing.T) {
	menu := Menu{Items: []MenuItem{
		{ID: "open", Label: "Open", Hotkey: "o"},
		{Separator: true},
		{ID: "del", Label: "Delete", Hotkey: "d", Disabled: true},
		{ID: "quit", Label: "Quit", Hotkey: "q"},
	}, Selected: 0}
	menu.Move(1)
	if menu.Selected != 3 {
		t.Fatalf("Move down selected = %d, want 3 (skip separator/disabled)", menu.Selected)
	}
	menu.Move(1)
	if menu.Selected != 3 {
		t.Fatalf("Move past the end selected = %d, want 3", menu.Selected)
	}
	menu.Move(-1)
	if menu.Selected != 0 {
		t.Fatalf("Move up selected = %d, want 0", menu.Selected)
	}
	menu.Move(-1)
	if menu.Selected != 0 {
		t.Fatalf("Move before the start selected = %d", menu.Selected)
	}
	if menu.Select(1) || menu.Select(2) {
		t.Fatal("separators and disabled items must not be selectable")
	}
	if !menu.Select(3) || menu.Value() != "quit" {
		t.Fatalf("Select/Value = %d %q", menu.Selected, menu.Value())
	}
}

func TestMenuHotkeyAndBuild(t *testing.T) {
	menu := Menu{
		ID: "menu", Title: "Actions", Width: 16, X: 1, Y: 1,
		SelectedStyle: "sel", DisabledStyle: "off",
		Items: []MenuItem{
			{ID: "open", Label: "Open", Hotkey: "o"},
			{ID: "del", Label: "Delete", Hotkey: "d", Disabled: true},
			{ID: "quit", Label: "Quit", Hotkey: "q"},
		},
	}
	if value, ok := menu.Hotkey(&pb.KeyEvent{Key: "d", Char: "d"}); ok || value != "" {
		t.Fatalf("disabled hotkey matched: %q %v", value, ok)
	}
	value, ok := menu.Hotkey(&pb.KeyEvent{Key: "Q", Char: "Q"})
	if !ok || value != "quit" || menu.Selected != 2 {
		t.Fatalf("hotkey Q = %q %v selected %d", value, ok, menu.Selected)
	}
	if _, ok := menu.Hotkey(&pb.KeyEvent{Key: "x", Char: "x"}); ok {
		t.Fatal("unknown hotkey must not match")
	}

	root := menu.Build().Build()
	if root.GetPos().GetX() != 1 || root.GetPos().GetY() != 1 {
		t.Fatalf("menu pos = %+v", root.GetPos())
	}
	open := findBox(root, "open")
	if open == nil || !strings.Contains(boxText(open), "Open") {
		t.Fatalf("open row = %+v", open)
	}
	if len(open.GetInput()) != 1 || open.GetInput()[0] != "mouse" {
		t.Fatalf("open row input = %v", open.GetInput())
	}
	selected := findBox(root, "quit")
	if selected == nil || !strings.HasPrefix(boxText(selected), DefaultListMarker) || selected.GetStyle() != "sel" {
		t.Fatalf("selected row = %+v", selected)
	}

	var empty Menu
	empty.Move(1)
	if empty.Value() != "" || empty.Build().Build() == nil {
		t.Fatalf("empty menu = %+v", empty)
	}
}

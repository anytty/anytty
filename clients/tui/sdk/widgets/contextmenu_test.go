package widgets

import (
	"testing"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func contextTestMenu() Menu {
	return Menu{
		ID:    "ctx",
		Width: 10,
		Items: []MenuItem{{ID: "copy", Label: "Copy", Hotkey: "c"}, {ID: "paste", Label: "Paste", Hotkey: "p"}},
	}
}

func TestContextMenuOpenClampsInsideParent(t *testing.T) {
	menu := ContextMenu{Menu: contextTestMenu(), ParentWidth: 20, ParentHeight: 10}
	x, y := menu.Open(18, 8)
	if !menu.Visible {
		t.Fatal("Open must mark the menu visible")
	}
	if x != 10 || y != 6 {
		t.Fatalf("clamped anchor = (%d, %d), want (10, 6)", x, y)
	}
	if menu.AnchorX != x || menu.AnchorY != y || menu.Menu.X != x || menu.Menu.Y != y {
		t.Fatalf("anchor bookkeeping = %+v", menu)
	}
	box := menu.Build().Build()
	if pos := box.GetPos(); pos.GetX() != 10 || pos.GetY() != 6 {
		t.Fatalf("built pos = %+v", pos)
	}
}

func TestContextMenuOpenWithoutParent(t *testing.T) {
	menu := ContextMenu{Menu: contextTestMenu()}
	x, y := menu.Open(100, 50)
	if x != 100 || y != 50 {
		t.Fatalf("unclamped anchor = (%d, %d)", x, y)
	}
}

func TestContextMenuMarginAndNegative(t *testing.T) {
	menu := ContextMenu{Menu: contextTestMenu(), ParentWidth: 12, ParentHeight: 6, Margin: 1}
	x, y := menu.Open(11, 5)
	if x != 1 || y != 1 {
		t.Fatalf("margin clamp = (%d, %d), want (1, 1)", x, y)
	}
	menu.Margin = 0
	if x, _ := menu.Open(-4, -4); x != 0 {
		t.Fatalf("negative anchor x = %d, want 0", x)
	}
}

func TestContextMenuCloseAndHeight(t *testing.T) {
	menu := ContextMenu{Menu: contextTestMenu()}
	menu.Open(3, 4)
	if menu.MenuHeight() != 4 {
		t.Fatalf("MenuHeight = %d, want items+frame", menu.MenuHeight())
	}
	menu.Close()
	if menu.Visible {
		t.Fatal("Close must hide the menu")
	}
	box := menu.Build().Build()
	if box.GetVisible() {
		t.Fatalf("hidden build = %+v, want Visible=false", box.GetVisible())
	}
	if len(box.GetChildren()) != 0 {
		t.Fatalf("hidden build children = %d", len(box.GetChildren()))
	}
}

func TestContextMenuKeyboardDrivesEmbeddedMenu(t *testing.T) {
	menu := ContextMenu{Menu: contextTestMenu()}
	menu.Open(0, 0)
	menu.Move(1)
	if menu.Selected != 1 {
		t.Fatalf("embedded Move selected = %d", menu.Selected)
	}
	if value, ok := menu.Hotkey(&pb.KeyEvent{Key: "c"}); !ok || value != "copy" {
		t.Fatalf("embedded Hotkey = %q/%v", value, ok)
	}
	if menu.Value() != "copy" {
		t.Fatalf("embedded Value = %q", menu.Value())
	}
}

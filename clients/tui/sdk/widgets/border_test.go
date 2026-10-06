package widgets

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk"
)

func TestBorderPresets(t *testing.T) {
	normal := BorderNormal()
	if normal.TopLeft != "┌" || normal.TopRight != "┐" || normal.BottomLeft != "└" || normal.BottomRight != "┘" {
		t.Fatalf("normal = %+v", normal)
	}
	if normal.Top != "─" || normal.Bottom != "─" || normal.Left != "│" || normal.Right != "│" {
		t.Fatalf("normal edges = %+v", normal)
	}
	rounded := BorderRounded()
	if rounded.TopLeft != "╭" || rounded.TopRight != "╮" || rounded.BottomLeft != "╰" || rounded.BottomRight != "╯" {
		t.Fatalf("rounded = %+v", rounded)
	}
	thick := BorderThick()
	if thick.TopLeft != "┏" || thick.Top != "━" || thick.Left != "┃" {
		t.Fatalf("thick = %+v", thick)
	}
	double := BorderDouble()
	if double.TopLeft != "╔" || double.Top != "═" || double.Left != "║" {
		t.Fatalf("double = %+v", double)
	}
	if BorderSetOrDefault(BorderRounded()).TopLeft != "╭" {
		t.Fatal("explicit set must be kept")
	}
	if BorderSetOrDefault(BorderSet{}).TopLeft != "┌" {
		t.Fatal("zero set must default to normal")
	}
}

func TestBorderBoxDrawsFrameAndTitle(t *testing.T) {
	root := BorderBox{
		ID:     "panel",
		Title:  "hi",
		Width:  10,
		Height: 4,
	}.Build().Build()

	if root.GetId() != "panel" {
		t.Fatalf("id = %q", root.GetId())
	}
	if root.GetSize().GetWidth() != 10 || root.GetSize().GetHeight() != 4 {
		t.Fatalf("size = %+v", root.GetSize())
	}
	children := root.GetChildren()
	if len(children) != 4 {
		t.Fatalf("rows = %d, want 4", len(children))
	}
	top := rowText(children[0])
	if !strings.HasPrefix(top, "┌") || !strings.HasSuffix(top, "┐") || !strings.Contains(top, " hi ") {
		t.Fatalf("top edge = %q", top)
	}
	if sdk.DisplayWidth(top) != 10 {
		t.Fatalf("top edge width = %d, want 10 (%q)", sdk.DisplayWidth(top), top)
	}
	if got := rowText(children[3]); got != "└"+strings.Repeat("─", 8)+"┘" {
		t.Fatalf("bottom edge = %q", got)
	}
	body := children[1].GetChildren()
	if len(body) != 3 || boxText(body[0]) != "│" || boxText(body[2]) != "│" {
		t.Fatalf("side bars = %+v", body)
	}
}

func TestBorderBoxTitleTooLongIsTruncated(t *testing.T) {
	root := BorderBox{Title: "a very long title indeed", Width: 8, Height: 3}.Build().Build()
	top := rowText(root.GetChildren()[0])
	if sdk.DisplayWidth(top) != 8 {
		t.Fatalf("top width = %d (%q)", sdk.DisplayWidth(top), top)
	}
	if !strings.Contains(top, " a ver") {
		t.Fatalf("title not truncated as expected: %q", top)
	}
	if !strings.HasSuffix(top, "┐") {
		t.Fatalf("top edge must close: %q", top)
	}
}

func TestBorderBoxTitleCJKSafe(t *testing.T) {
	root := BorderBox{Title: "终端列表", Width: 9, Height: 3}.Build().Build()
	top := rowText(root.GetChildren()[0])
	if sdk.DisplayWidth(top) != 9 {
		t.Fatalf("top width = %d (%q), wide title split?", sdk.DisplayWidth(top), top)
	}
	if !strings.HasSuffix(top, "┐") {
		t.Fatalf("top edge must close: %q", top)
	}
}

func TestBorderBoxZeroSizeDegrades(t *testing.T) {
	child := sdk.Text("body")
	root := BorderBox{Child: child}.Build().Build()
	// No explicit size: the box falls back to the child's measured width.
	if root.GetSize().GetWidth() != 6 {
		t.Fatalf("fallback width = %d, want 6", root.GetSize().GetWidth())
	}
	if root.GetSize().GetHeight() != 3 {
		t.Fatalf("fallback height = %d, want 3", root.GetSize().GetHeight())
	}

	tiny := BorderBox{Child: child, Width: 1, Height: 1}.Build().Build()
	if len(tiny.GetChildren()) != 1 || rowText(tiny) != "body" {
		t.Fatalf("too-small box must degrade to the child: %+v", tiny)
	}

	empty := BorderBox{Width: 0, Height: 0}.Build().Build()
	if empty.GetSize().GetWidth() != 2 || empty.GetSize().GetHeight() != 2 {
		t.Fatalf("empty box size = %+v", empty.GetSize())
	}
}

func TestBorderBoxStyleDefaultsAndChild(t *testing.T) {
	child := sdk.Text("x").Height(1).Flex(1)
	root := BorderBox{Child: child, Title: "T", Width: 6, Height: 3, Style: "fg:#ff0000", TitleStyle: "accent"}.Build().Build()
	top := root.GetChildren()[0].GetChildren()
	if top[0].GetStyle() != "fg:#ff0000" {
		t.Fatalf("border style = %q", top[0].GetStyle())
	}
	if len(top) < 2 || top[1].GetStyle() != "accent" {
		t.Fatalf("title style = %+v", top)
	}
	// Default border style token when unset.
	def := BorderBox{Width: 4, Height: 2}.Build().Build()
	if def.GetChildren()[0].GetChildren()[0].GetStyle() != StyleBorder {
		t.Fatalf("default border style = %q", def.GetChildren()[0].GetChildren()[0].GetStyle())
	}
}

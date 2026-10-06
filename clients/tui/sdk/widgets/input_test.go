package widgets

import (
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func TestTextInputEditing(t *testing.T) {
	in := TextInput{}
	if !in.InsertString("hello") || in.Text() != "hello" || in.Cursor != 5 {
		t.Fatalf("InsertString = %q cursor %d", in.Text(), in.Cursor)
	}
	if !in.Left() || in.Cursor != 4 {
		t.Fatalf("Left cursor = %d", in.Cursor)
	}
	if !in.InsertRune('X') || in.Text() != "hellXo" || in.Cursor != 5 {
		t.Fatalf("InsertRune = %q cursor %d", in.Text(), in.Cursor)
	}
	if !in.Backspace() || in.Text() != "hello" || in.Cursor != 4 {
		t.Fatalf("Backspace = %q cursor %d", in.Text(), in.Cursor)
	}
	if !in.Delete() || in.Text() != "hell" || in.Cursor != 4 {
		t.Fatalf("Delete = %q cursor %d", in.Text(), in.Cursor)
	}
	if !in.Home() || in.Cursor != 0 {
		t.Fatalf("Home cursor = %d", in.Cursor)
	}
	if in.Home() {
		t.Fatal("Home at the start must not report movement")
	}
	if in.Left() {
		t.Fatal("Left at the start must not move")
	}
	if !in.Right() || in.Cursor != 1 {
		t.Fatalf("Right cursor = %d", in.Cursor)
	}
	if !in.End() || in.Cursor != 4 {
		t.Fatalf("End cursor = %d", in.Cursor)
	}

	var empty TextInput
	if empty.Backspace() || empty.Delete() || empty.Left() || empty.Right() || empty.DeleteToEnd() {
		t.Fatal("empty input editing ops must be no-ops")
	}
}

func TestTextInputWordsAndLimits(t *testing.T) {
	in := TextInput{Value: []rune("foo bar  baz")}
	in.End()
	if !in.WordLeft() || in.Cursor != 9 {
		t.Fatalf("WordLeft cursor = %d, want 9", in.Cursor)
	}
	if !in.WordLeft() || in.Cursor != 4 {
		t.Fatalf("WordLeft cursor = %d, want 4", in.Cursor)
	}
	if !in.WordRight() || in.Cursor != 9 {
		t.Fatalf("WordRight cursor = %d, want 9", in.Cursor)
	}
	in.Home()
	in.End()
	if !in.DeleteWordLeft() || in.Text() != "foo bar  " || in.Cursor != 9 {
		t.Fatalf("DeleteWordLeft = %q cursor %d", in.Text(), in.Cursor)
	}

	limited := TextInput{MaxLen: 3}
	if !limited.InsertString("abcdef") || limited.Text() != "abc" {
		t.Fatalf("MaxLen insert = %q", limited.Text())
	}
	if limited.InsertRune('x') || limited.InsertString("de") {
		t.Fatalf("MaxLen must reject overflow: %q", limited.Text())
	}
	limited.SetValue("toolong")
	if limited.Text() != "too" || limited.Cursor != 3 {
		t.Fatalf("SetValue MaxLen = %q cursor %d", limited.Text(), limited.Cursor)
	}
}

func TestTextInputBuildMaskPlaceholderAndCursor(t *testing.T) {
	masked := TextInput{Value: []rune("secret"), Mask: '•', Width: 10, Focused: true, Cursor: 6}
	root := masked.Build().Build()
	if got := rowText(root); got != "••••••    " {
		t.Fatalf("masked row = %q", got)
	}
	if root.GetCursor() == nil || root.GetCursor().GetCol() != 6 || root.GetCursor().GetRow() != 0 {
		t.Fatalf("masked cursor = %+v", root.GetCursor())
	}
	if !root.GetFocused() {
		t.Fatal("focused input must declare focused")
	}

	unfocused := TextInput{Value: []rune("secret"), Mask: '•', Width: 4}
	if got := rowText(unfocused.Build().Build()); got != "••••" {
		t.Fatalf("unfocused row = %q", got)
	}
	if root := unfocused.Build().Build(); root.GetCursor() != nil {
		t.Fatalf("unfocused input must not declare a cursor: %+v", root.GetCursor())
	}

	placeholder := TextInput{Placeholder: "name", Width: 6, Focused: true}
	pRoot := placeholder.Build().Build()
	if got := rowText(pRoot); got != "name  " {
		t.Fatalf("placeholder row = %q", got)
	}
	if pRoot.GetCursor() == nil || pRoot.GetCursor().GetCol() != 0 {
		t.Fatalf("placeholder cursor = %+v", pRoot.GetCursor())
	}
}

func TestTextInputHorizontalScroll(t *testing.T) {
	scrolled := TextInput{Value: []rune("abcdefghij"), Cursor: 10, Width: 5, Focused: true}
	root := scrolled.Build().Build()
	if got := rowText(root); got != "ghij " {
		t.Fatalf("scrolled row = %q", got)
	}
	if root.GetCursor().GetCol() != 4 {
		t.Fatalf("scrolled cursor col = %d, want 4", root.GetCursor().GetCol())
	}

	cjk := TextInput{Value: []rune("中文中文中文"), Cursor: 6, Width: 5, Focused: true}
	cjkRoot := cjk.Build().Build()
	if got := rowText(cjkRoot); sdk.DisplayWidth(got) != 5 || got != "中文 " {
		t.Fatalf("CJK scrolled row = %q (width %d)", got, sdk.DisplayWidth(got))
	}
	if cjkRoot.GetCursor().GetCol() != 4 {
		t.Fatalf("CJK cursor col = %d, want 4", cjkRoot.GetCursor().GetCol())
	}
}

func TestTextInputHandleKey(t *testing.T) {
	in := TextInput{}
	if !in.HandleKey(&pb.KeyEvent{Key: "a", Char: "a"}) || in.Text() != "a" {
		t.Fatalf("printable key = %q", in.Text())
	}
	if !in.HandleKey(&pb.KeyEvent{Key: "left"}) || in.Cursor != 0 {
		t.Fatalf("left key cursor = %d", in.Cursor)
	}
	if !in.HandleKey(&pb.KeyEvent{Key: "中", Char: "中"}) || in.Text() != "中a" {
		t.Fatalf("CJK key = %q", in.Text())
	}
	if in.HandleKey(&pb.KeyEvent{Key: "enter"}) || in.HandleKey(&pb.KeyEvent{Key: "tab"}) || in.HandleKey(nil) {
		t.Fatal("enter/tab/nil must not be consumed by a single-line input")
	}
	if !in.HandleKey(&pb.KeyEvent{Key: "ctrl-a"}) || in.Cursor != 0 {
		t.Fatalf("ctrl-a cursor = %d", in.Cursor)
	}
	if !in.HandleKey(&pb.KeyEvent{Key: "delete"}) || in.Text() != "a" {
		t.Fatalf("delete key = %q", in.Text())
	}
}

func TestTextAreaEditingAndCursor(t *testing.T) {
	area := TextArea{}
	if !area.InsertString("ab\r\ncd") || area.Text() != "ab\ncd" {
		t.Fatalf("InsertString = %q", area.Text())
	}
	if row, col := area.RowCol(); row != 1 || col != 2 {
		t.Fatalf("RowCol = (%d, %d), want (1, 2)", row, col)
	}
	if !area.Up() {
		t.Fatal("Up must move between lines")
	}
	if row, col := area.RowCol(); row != 0 || col != 2 {
		t.Fatalf("after Up = (%d, %d), want (0, 2)", row, col)
	}
	if area.Up() {
		t.Fatal("Up at the first line must not move")
	}
	if !area.Down() || area.Down() {
		t.Fatal("Down must move exactly one line")
	}
	if !area.Home() || area.Cursor != 3 {
		t.Fatalf("Home cursor = %d, want line start 3", area.Cursor)
	}
	if !area.End() || area.Cursor != 5 {
		t.Fatalf("End cursor = %d, want line end 5", area.Cursor)
	}
	if !area.InsertRune('\n') || area.Text() != "ab\ncd\n" {
		t.Fatalf("newline insert = %q", area.Text())
	}
	if row, col := area.RowCol(); row != 2 || col != 0 {
		t.Fatalf("after newline RowCol = (%d, %d)", row, col)
	}
	if !area.Backspace() || area.Text() != "ab\ncd" || area.Cursor != 5 {
		t.Fatalf("Backspace after newline = %q cursor %d", area.Text(), area.Cursor)
	}
	if got := area.Line(0); got != "ab" {
		t.Fatalf("Line(0) = %q", got)
	}
	if got := area.Line(7); got != "" {
		t.Fatalf("out-of-range Line = %q", got)
	}
}

func TestTextAreaScrollAndCursorProtocol(t *testing.T) {
	area := TextArea{Value: []rune("l0\nl1\nl2\nl3\nl4"), Height: 2, Cursor: 14, Focused: true}
	root := area.Build().Build()
	children := root.GetChildren()
	if len(children) != 2 {
		t.Fatalf("scrolled textarea rows = %d, want 2", len(children))
	}
	if rowText(children[0]) != "l3" || rowText(children[1]) != "l4 " {
		t.Fatalf("scrolled rows = %q / %q", rowText(children[0]), rowText(children[1]))
	}
	if root.GetCursor().GetRow() != 1 || root.GetCursor().GetCol() != 2 {
		t.Fatalf("textarea cursor = %+v", root.GetCursor())
	}

	cjk := TextArea{Value: []rune("中文中文\n短"), Width: 5, Focused: true, Cursor: 4}
	cjkRoot := cjk.Build().Build()
	if got := rowText(cjkRoot.GetChildren()[0]); got != "中文 " {
		t.Fatalf("CJK textarea row = %q (width %d)", got, sdk.DisplayWidth(got))
	}
	if cjkRoot.GetCursor().GetCol() != 4 {
		t.Fatalf("CJK textarea cursor col = %d", cjkRoot.GetCursor().GetCol())
	}
}

func TestTextAreaMaskAndPageKeys(t *testing.T) {
	masked := TextArea{Value: []rune("ab\ncd"), Mask: '*', Width: 4, Height: 2, Focused: true, Cursor: 2}
	root := masked.Build().Build()
	if got := rowText(root.GetChildren()[0]); got != "**  " {
		t.Fatalf("masked cursor row = %q", got)
	}
	if got := rowText(root.GetChildren()[1]); got != "**  " {
		t.Fatalf("masked row = %q", got)
	}

	area := TextArea{Value: []rune("a\nb\nc\nd\ne\nf"), Height: 2}
	area.Cursor = 0
	if !area.PageDown() || area.Cursor < 2 {
		t.Fatalf("PageDown cursor = %d", area.Cursor)
	}
	if !area.PageUp() || area.Cursor != 0 {
		t.Fatalf("PageUp cursor = %d", area.Cursor)
	}
	if !area.HandleKey(&pb.KeyEvent{Key: "enter"}) || area.Cursor != 1 {
		t.Fatalf("enter insert cursor = %d", area.Cursor)
	}
	if area.HandleKey(&pb.KeyEvent{Key: "esc"}) {
		t.Fatal("esc must be left for the caller")
	}
}

func TestTextPlaceholderEmptyBuild(t *testing.T) {
	area := TextArea{Placeholder: "notes", Width: 7, Focused: true}
	root := area.Build().Build()
	if len(root.GetChildren()) != 1 || rowText(root.GetChildren()[0]) != "notes  " {
		t.Fatalf("textarea placeholder = %+v", root)
	}
	if root.GetCursor() == nil {
		t.Fatal("focused placeholder must carry the cursor")
	}
}

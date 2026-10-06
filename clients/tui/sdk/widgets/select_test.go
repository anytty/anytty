package widgets

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk"
)

func testSelect() *Select {
	return &Select{
		ID:    "color",
		Label: "Color",
		Width: 20,
		Options: []Option{
			{Value: "red", Label: "Red"},
			{Value: "green", Label: "Green", Disabled: true},
			{Value: "grey", Label: "Grey"},
			{Value: "blue", Label: "Blue"},
		},
	}
}

func TestSelectSelected(t *testing.T) {
	sel := testSelect()
	if _, ok := sel.Selected(); ok {
		t.Fatal("empty select must have no selection")
	}
	if sel.Index() != -1 {
		t.Fatalf("Index = %d", sel.Index())
	}
	if !sel.SelectIndex(3) || sel.Value != "blue" {
		t.Fatalf("SelectIndex = %q", sel.Value)
	}
	if option, ok := sel.Selected(); !ok || option.Value != "blue" {
		t.Fatalf("Selected = %+v %v", option, ok)
	}
	if sel.SelectIndex(1) {
		t.Fatal("SelectIndex on a disabled option must fail")
	}
	if sel.SelectIndex(len(sel.Options)) || sel.SelectIndex(-1) {
		t.Fatal("SelectIndex out of range must fail")
	}
	sel.Value = "green"
	if _, ok := sel.Selected(); ok {
		t.Fatal("a disabled selected option reports ok=false")
	}
}

func TestSelectMoveSkipsDisabled(t *testing.T) {
	sel := testSelect()
	sel.SelectIndex(0)
	sel.Move(1)
	if sel.Value != "grey" {
		t.Fatalf("Move(1) skipped wrong, value = %q", sel.Value)
	}
	sel.Move(1)
	if sel.Value != "blue" {
		t.Fatalf("Move(1) = %q", sel.Value)
	}
	sel.Move(1)
	if sel.Value != "blue" {
		t.Fatalf("Move past the end must stop, value = %q", sel.Value)
	}
	sel.Move(-1)
	if sel.Value != "grey" {
		t.Fatalf("Move(-1) = %q", sel.Value)
	}
	sel.Move(-3)
	if sel.Value != "red" {
		t.Fatalf("Move(-3) = %q", sel.Value)
	}

	unselected := testSelect()
	unselected.Move(1)
	if unselected.Value != "red" {
		t.Fatalf("Move from nothing = %q", unselected.Value)
	}
	back := testSelect()
	back.Move(-1)
	if back.Value != "blue" {
		t.Fatalf("Move(-1) from nothing = %q", back.Value)
	}
	testSelect().Move(1)
	empty := &Select{}
	empty.Move(1)
	if empty.Value != "" {
		t.Fatalf("empty select moved to %q", empty.Value)
	}
	empty.Move(0)
}

func TestSelectTypeahead(t *testing.T) {
	sel := testSelect()
	sel.SelectIndex(0)
	if !sel.Typeahead("gr") {
		t.Fatal("typeahead gr should match Grey, skipping disabled Green")
	}
	if sel.Value != "grey" {
		t.Fatalf("Typeahead(gr) = %q", sel.Value)
	}
	if !sel.Typeahead("b") || sel.Value != "blue" {
		t.Fatalf("Typeahead(b) = %q", sel.Value)
	}
	if !sel.Typeahead("Re") || sel.Value != "red" {
		t.Fatalf("Typeahead must wrap and ignore case, value = %q", sel.Value)
	}
	if sel.Typeahead("zzz") {
		t.Fatal("Typeahead with no match must fail")
	}
	if sel.Typeahead("") {
		t.Fatal("empty prefix must match nothing")
	}
	if (&Select{}).Typeahead("a") {
		t.Fatal("empty option set must fail")
	}

	values := &Select{Options: []Option{{Value: "alpha"}, {Value: "beta", Label: "B"}}}
	if !values.Typeahead("be") || values.Value != "beta" {
		t.Fatalf("value-prefix typeahead = %q", values.Value)
	}
}

func TestSelectBuildClosed(t *testing.T) {
	sel := testSelect()
	sel.Value = "blue"
	root := sel.Build().Build()
	if root.GetId() != "color" {
		t.Fatalf("id = %q", root.GetId())
	}
	if got := boxText(root.GetChildren()[0]); got != "Color" {
		t.Fatalf("label = %q", got)
	}
	row := root.GetChildren()[1]
	text := rowText(row)
	if got := sdk.DisplayWidth(text); got != 20 {
		t.Fatalf("closed width = %d, want 20 (%q)", got, text)
	}
	if text[:4] != "Blue" {
		t.Fatalf("closed value = %q", text)
	}
	trimmed := strings.TrimRight(text, " ")
	if !strings.HasSuffix(trimmed, "▾") {
		t.Fatalf("closed indicator missing: %q", text)
	}

	empty := &Select{Placeholder: "choose", Indicator: "!"}
	emptyText := rowText(empty.Build().Build())
	if emptyText != "choose!" {
		t.Fatalf("placeholder = %q", emptyText)
	}
}

func TestSelectBuildDropdown(t *testing.T) {
	sel := testSelect()
	sel.Open = true
	sel.Value = "grey"
	root := sel.Build().Build()
	if len(root.GetChildren()) != len(sel.Options) {
		t.Fatalf("dropdown rows = %d", len(root.GetChildren()))
	}
	selected := rowText(root.GetChildren()[2])
	if selected[:len(DefaultListMarker)] != DefaultListMarker {
		t.Fatalf("selected marker missing: %q", selected)
	}
	disabled := root.GetChildren()[1]
	if boxText(disabled)[:2] != "  " {
		t.Fatalf("disabled row must not be marked: %q", boxText(disabled))
	}

	sel.DisabledStyle = "muted"
	dropdown := sel.BuildDropdown().Build()
	if dropdown.GetChildren()[1].GetStyle() != "muted" {
		t.Fatalf("disabled style = %q", dropdown.GetChildren()[1].GetStyle())
	}
}

func TestSelectUnicodeLabels(t *testing.T) {
	sel := &Select{
		Options: []Option{
			{Value: "cn", Label: "中文"},
			{Value: "jp", Label: "日本語"},
		},
		Placeholder: "选择",
	}
	if !sel.Typeahead("中") || sel.Value != "cn" {
		t.Fatalf("CJK typeahead = %q", sel.Value)
	}
	sel.Value = "jp"
	text := rowText(sel.Build().Build())
	if got := sdk.DisplayWidth(text); got != sdk.DisplayWidth("日本語 ▾") {
		t.Fatalf("unicode closed width = %d (%q)", got, text)
	}
}

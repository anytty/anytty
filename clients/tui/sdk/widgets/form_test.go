package widgets

import (
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func newTestForm() *Form {
	return &Form{
		ID:    "login",
		Width: 30,
		Fields: []Field{
			{ID: "user", Label: "User", Input: &TextInput{ID: "user"}, Required: true},
			{ID: "skip", Label: "Skip", Input: &TextInput{ID: "skip"}, Disabled: true},
			{ID: "pass", Label: "Pass", Input: &TextInput{ID: "pass"}, Validate: MinLen(4, "too short")},
		},
	}
}

func TestFormNavigationSkipsDisabledAndReadonly(t *testing.T) {
	form := newTestForm()
	if form.FocusID() != "user" {
		t.Fatalf("initial focus = %q", form.FocusID())
	}
	form.Next()
	if form.FocusID() != "pass" {
		t.Fatalf("Next should skip disabled, focus = %q", form.FocusID())
	}
	form.Next()
	if form.FocusID() != "user" {
		t.Fatalf("Next should wrap, focus = %q", form.FocusID())
	}
	form.Prev()
	if form.FocusID() != "pass" {
		t.Fatalf("Prev should skip disabled and wrap, focus = %q", form.FocusID())
	}

	form.Fields[2].Readonly = true
	if form.FocusID() != "pass" {
		t.Fatalf("Readonly field should still render, focus = %q", form.FocusID())
	}
	form.Prev()
	if form.FocusID() != "user" {
		t.Fatalf("Prev must skip readonly, focus = %q", form.FocusID())
	}
}

func TestFormNavigationAllDisabledIsStable(t *testing.T) {
	form := &Form{Fields: []Field{{ID: "a", Disabled: true}, {ID: "b", Readonly: true}}}
	form.Next()
	if form.FocusID() != "a" {
		t.Fatalf("focus = %q", form.FocusID())
	}
	empty := &Form{}
	empty.Next()
	empty.Prev()
	if empty.FocusID() != "" {
		t.Fatalf("empty form focus = %q", empty.FocusID())
	}
}

func TestFormValuesAndInput(t *testing.T) {
	form := newTestForm()
	form.SetValues(map[string]string{"user": "ada", "pass": "hunter2"})
	values := form.Values()
	if values["user"] != "ada" || values["pass"] != "hunter2" {
		t.Fatalf("Values = %v", values)
	}
	if form.Input("user") == nil || form.Input("user").Text() != "ada" {
		t.Fatalf("Input(user) = %+v", form.Input("user"))
	}
	if form.Input("missing") != nil {
		t.Fatal("Input(missing) must be nil")
	}
	if form.Input("user").InsertRune('!'); form.Values()["user"] != "ada!" {
		t.Fatalf("editing through Input pointer failed: %v", form.Values())
	}
}

func TestFormValidateAggregation(t *testing.T) {
	form := &Form{
		Fields: []Field{
			{ID: "user", Input: &TextInput{}, Required: true},
			{ID: "pass", Input: &TextInput{}, Validate: MinLen(4, "too short")},
		},
	}
	if form.Validate() {
		t.Fatal("empty form should be invalid")
	}
	if form.Fields[0].Error != FormRequiredMessage || form.Fields[1].Error != "" {
		t.Fatalf("errors = %q / %q", form.Fields[0].Error, form.Fields[1].Error)
	}
	form.SetValues(map[string]string{"user": "ada", "pass": "x"})
	if form.Validate() {
		t.Fatal("short password should be invalid")
	}
	if form.Fields[0].Error != "" || form.Fields[1].Error != "too short" {
		t.Fatalf("errors = %q / %q", form.Fields[0].Error, form.Fields[1].Error)
	}
	form.SetValues(map[string]string{"pass": "longenough"})
	if !form.Validate() {
		t.Fatal("valid form reported invalid")
	}
	if form.Fields[0].Error != "" || form.Fields[1].Error != "" {
		t.Fatalf("errors not cleared: %q / %q", form.Fields[0].Error, form.Fields[1].Error)
	}

	requiredOverrides := &Form{Fields: []Field{{
		Input:    &TextInput{},
		Required: true,
		Validate: func(string) string { return "custom" },
	}}}
	requiredOverrides.Validate()
	if requiredOverrides.Fields[0].Error != FormRequiredMessage {
		t.Fatalf("Required should win over Validate on empty, got %q", requiredOverrides.Fields[0].Error)
	}
}

func TestFormHandleKeyRouting(t *testing.T) {
	form := newTestForm()
	if form.HandleKey("unknown", &pb.KeyEvent{Key: "a", Char: "a"}) {
		t.Fatal("key for a non-focused id must be ignored")
	}
	if form.Input("user").Text() != "" {
		t.Fatal("non-focused id must not edit")
	}

	if !form.HandleKey("user", &pb.KeyEvent{Key: "a", Char: "a"}) {
		t.Fatal("focused id must accept a printable key")
	}
	if form.Input("user").Text() != "a" {
		t.Fatalf("user = %q", form.Input("user").Text())
	}
	if !form.HandleKey("user", &pb.KeyEvent{Key: "tab"}) {
		t.Fatal("tab must be consumed")
	}
	if form.FocusID() != "pass" {
		t.Fatalf("tab focus = %q", form.FocusID())
	}
	if !form.HandleKey("", &pb.KeyEvent{Key: "shift-tab"}) {
		t.Fatal("shift-tab must be consumed")
	}
	if form.FocusID() != "user" {
		t.Fatalf("shift-tab focus = %q", form.FocusID())
	}
	if !form.HandleKey("user", &pb.KeyEvent{Key: "enter"}) {
		t.Fatal("enter must navigate")
	}
	if form.FocusID() != "pass" {
		t.Fatalf("enter focus = %q", form.FocusID())
	}
	if form.HandleKey("user", nil) {
		t.Fatal("nil event must be a no-op")
	}
}

func TestFormValidateOnChange(t *testing.T) {
	form := &Form{
		ValidateOnChange: true,
		Fields: []Field{
			{ID: "email", Input: &TextInput{ID: "email"}, Required: true, Validate: Email("bad email")},
		},
	}
	form.Fields[0].Error = FormRequiredMessage
	form.HandleKey("email", &pb.KeyEvent{Key: "a", Char: "a"})
	if form.Fields[0].Error != "bad email" {
		t.Fatalf("error = %q, want bad email", form.Fields[0].Error)
	}

	form.Input("email").SetValue("a@b.co")
	form.Fields[0].Error = "stale"
	form.HandleKey("email", &pb.KeyEvent{Key: "x", Char: "x"})
	if form.Fields[0].Error != "" {
		t.Fatalf("error should clear on valid edit, got %q", form.Fields[0].Error)
	}

	form.Input("email").SetValue("ab")
	form.Fields[0].Error = "stale"
	form.HandleKey("email", &pb.KeyEvent{Key: "backspace"})
	if form.Fields[0].Error != "bad email" {
		t.Fatalf("error after invalid edit = %q", form.Fields[0].Error)
	}
}

func TestFormAndFieldBuild(t *testing.T) {
	form := newTestForm()
	form.Fields[0].Error = "user is required"
	form.Fields[0].Help = "your login"
	root := form.Build().Build()

	if root.GetId() != "login" || len(root.GetChildren()) != len(form.Fields) {
		t.Fatalf("root = %+v", root)
	}
	field := root.GetChildren()[0]
	if got := boxText(field.GetChildren()[0]); got != "User *" {
		t.Fatalf("label = %q", got)
	}
	errorLine := field.GetChildren()[2]
	if boxText(errorLine) != "user is required" {
		t.Fatalf("error line = %q", boxText(errorLine))
	}

	form.Fields[0].Error = ""
	helpRoot := form.Build().Build()
	helpLine := helpRoot.GetChildren()[0].GetChildren()[2]
	if boxText(helpLine) != "your login" {
		t.Fatalf("help line = %q", boxText(helpLine))
	}

	var bare Field
	built := bare.Build().Build()
	if len(built.GetChildren()) != 0 {
		t.Fatalf("empty field children = %d", len(built.GetChildren()))
	}
}

func TestFormErrorStylePropagation(t *testing.T) {
	form := Form{
		ErrorStyle: "danger",
		Fields: []Field{{
			ID:    "a",
			Error: "boom",
			Input: &TextInput{},
		}},
	}
	root := form.Build().Build()
	field := root.GetChildren()[0]
	if got := field.GetChildren()[1].GetStyle(); got != "danger" {
		t.Fatalf("error style = %q", got)
	}
}

func TestFormWidthPropagatesToInput(t *testing.T) {
	form := Form{
		ID:    "f",
		Width: 20,
		Fields: []Field{{
			ID:    "a",
			Input: &TextInput{ID: "a", Width: 20},
		}},
	}
	root := form.Build().Build()
	input := findBox(root, "a")
	if input == nil || input.GetSize().GetWidth() != 20 {
		t.Fatalf("input box = %+v", input)
	}
	if sdk.DisplayWidth(rowText(findBox(root, "a"))) != 20 {
		t.Fatalf("input row width = %d", sdk.DisplayWidth(rowText(findBox(root, "a"))))
	}
}

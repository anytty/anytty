package widgets

import (
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// FormRequiredMessage is the fallback error stored for a Required field that
// is empty and has no Validate of its own.
const FormRequiredMessage = "required"

// Field is one Form row: an optional label, an owned TextInput, and a help or
// error line rendered underneath. Disabled and Readonly fields are skipped by
// Form navigation; Readonly still shows its value and can be validated.
type Field struct {
	ID         string
	Label      string
	Help       string
	Error      string
	Input      *TextInput
	Required   bool
	Validate   func(string) string
	Style      string
	LabelStyle string
	ErrorStyle string
	HelpStyle  string
	Focused    bool
	Disabled   bool
	Readonly   bool
}

// Selectable reports whether Form navigation may land on the field.
func (f Field) Selectable() bool { return !f.Disabled && !f.Readonly }

// Text returns the field value, or "" when it owns no input.
func (f Field) Text() string {
	if f.Input == nil {
		return ""
	}
	return f.Input.Text()
}

// Build returns the field as a column: label, input, then the error line when
// set, otherwise the help line.
func (f Field) Build() *sdk.Builder { return formFieldBox(f, 0) }

func formFieldBox(f Field, width int) *sdk.Builder {
	col := sdk.Box().Flow("col")
	if width > 0 {
		col.Width(width)
	}
	if f.Style != "" {
		col.Style(f.Style)
	}
	if f.Label != "" {
		label := f.Label
		if f.Required {
			label += " *"
		}
		col.Child(sdk.Text(label).Style(f.LabelStyle).Height(1))
	}
	if f.Input != nil {
		in := *f.Input
		in.Focused = f.Focused
		if in.Width <= 0 && width > 0 {
			in.Width = width
		}
		col.Child(in.Build())
	}
	switch {
	case f.Error != "":
		col.Child(sdk.Text(f.Error).Style(f.ErrorStyle).Height(1))
	case f.Help != "":
		col.Child(sdk.Text(f.Help).Style(f.HelpStyle).Height(1))
	}
	return col
}

// Form is a column of Fields with keyboard focus, navigation and aggregate
// validation. The caller owns the values and drives keys through HandleKey;
// the widget never emits frames and holds no timers.
type Form struct {
	ID               string
	Fields           []Field
	Focus            int
	Width            int
	Style            string
	ErrorStyle       string
	ValidateOnChange bool
}

// Next moves focus to the next selectable field, wrapping around. A form with
// no selectable field leaves Focus unchanged.
func (f *Form) Next() { f.moveFocus(1) }

// Prev moves focus to the previous selectable field, wrapping around.
func (f *Form) Prev() { f.moveFocus(-1) }

func (f *Form) moveFocus(step int) {
	count := len(f.Fields)
	if count == 0 {
		f.Focus = 0
		return
	}
	f.Focus = clampIndex(f.Focus, count)
	for i := 1; i <= count; i++ {
		next := (f.Focus + step*i) % count
		if next < 0 {
			next += count
		}
		if f.Fields[next].Selectable() {
			f.Focus = next
			return
		}
	}
}

// FocusID returns the id of the focused field, or "" when the form has none.
func (f Form) FocusID() string {
	if f.Focus < 0 || f.Focus >= len(f.Fields) {
		return ""
	}
	return f.Fields[f.Focus].ID
}

// SetValues copies values keyed by field id into the matching inputs.
func (f *Form) SetValues(values map[string]string) {
	for i := range f.Fields {
		if f.Fields[i].Input == nil {
			continue
		}
		if f.Fields[i].ID == "" {
			continue
		}
		if value, ok := values[f.Fields[i].ID]; ok {
			f.Fields[i].Input.SetValue(value)
		}
	}
}

// Values returns the current text of every identified field.
func (f Form) Values() map[string]string {
	out := make(map[string]string, len(f.Fields))
	for i := range f.Fields {
		if f.Fields[i].ID == "" || f.Fields[i].Input == nil {
			continue
		}
		out[f.Fields[i].ID] = f.Fields[i].Input.Text()
	}
	return out
}

// Input returns the input owned by the field with the given id so the caller
// can edit it in place, or nil when no field matches.
func (f *Form) Input(id string) *TextInput {
	for i := range f.Fields {
		if f.Fields[i].ID == id {
			return f.Fields[i].Input
		}
	}
	return nil
}

// Validate runs every field's Required check and Validate, stores each result
// in Field.Error and reports whether the whole form is valid.
func (f *Form) Validate() bool {
	valid := true
	for i := range f.Fields {
		msg := f.fieldError(i)
		f.Fields[i].Error = msg
		if msg != "" {
			valid = false
		}
	}
	return valid
}

func (f *Form) fieldError(index int) string {
	if index < 0 || index >= len(f.Fields) {
		return ""
	}
	field := &f.Fields[index]
	value := field.Text()
	if field.Required && strings.TrimSpace(value) == "" {
		return FormRequiredMessage
	}
	if field.Validate != nil {
		return field.Validate(value)
	}
	return ""
}

// HandleKey routes one key to the focused field. A non-empty id that is not
// the focused field is ignored. Enter/Tab advance focus and Shift-Tab retreats;
// every other key is delegated to the focused TextInput. With ValidateOnChange
// the focused field's error is refreshed after an edit.
func (f *Form) HandleKey(id string, ev *pb.KeyEvent) bool {
	if ev == nil {
		return false
	}
	if id != "" && id != f.FocusID() {
		return false
	}
	switch ev.GetKey() {
	case "enter", "tab":
		f.Next()
		return true
	case "shift-tab":
		f.Prev()
		return true
	}
	if f.Focus < 0 || f.Focus >= len(f.Fields) || f.Fields[f.Focus].Input == nil {
		return false
	}
	if !f.Fields[f.Focus].Input.HandleKey(ev) {
		return false
	}
	if f.ValidateOnChange {
		f.Fields[f.Focus].Error = f.fieldError(f.Focus)
	}
	return true
}

// Build returns the form as a column of fields, marking the focused one.
func (f Form) Build() *sdk.Builder {
	col := sdk.Box().Flow("col")
	if f.ID != "" {
		col.ID(f.ID)
	}
	if f.Width > 0 {
		col.Width(f.Width)
	}
	if f.Style != "" {
		col.Style(f.Style)
	}
	for i := range f.Fields {
		field := f.Fields[i]
		field.Focused = i == f.Focus
		if field.ErrorStyle == "" {
			field.ErrorStyle = f.ErrorStyle
		}
		col.Child(formFieldBox(field, f.Width))
	}
	return col
}

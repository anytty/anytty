package widgets

import "github.com/anytty/anytty/clients/tui/sdk"

// DefaultSelectIndicator is the closed-state caret drawn after the value.
const DefaultSelectIndicator = " ▾"

// Option is one Select choice. Disabled options render with DisabledStyle and
// are skipped by Move and Typeahead.
type Option struct {
	Value    string
	Label    string
	Disabled bool
	Style    string
}

// Display returns the option label, falling back to its value.
func (o Option) Display() string {
	if o.Label != "" {
		return o.Label
	}
	return o.Value
}

// Select is a dropdown state machine plus a pure Build. The caller owns the
// value and toggles Open; the widget never emits frames. Value is the
// Option.Value of the selection ("" means none, so Placeholder is shown).
type Select struct {
	ID               string
	Label            string
	Options          []Option
	Value            string
	Open             bool
	Width            int
	Placeholder      string
	Style            string
	LabelStyle       string
	PlaceholderStyle string
	SelectedStyle    string
	DisabledStyle    string
	Marker           string
	Indicator        string
	DropdownStyle    string
}

// Index returns the position of the selected option, or -1 when nothing is
// selected.
func (s Select) Index() int {
	for i, option := range s.Options {
		if option.Value == s.Value {
			return i
		}
	}
	return -1
}

// Selected returns the selected option. A disabled selected option still
// reports ok=false, matching Form's navigation rules.
func (s Select) Selected() (Option, bool) {
	i := s.Index()
	if i < 0 || s.Options[i].Disabled {
		return Option{}, false
	}
	return s.Options[i], true
}

// SelectIndex selects the option at index when it is in range and enabled.
func (s *Select) SelectIndex(index int) bool {
	if index < 0 || index >= len(s.Options) || s.Options[index].Disabled {
		return false
	}
	s.Value = s.Options[index].Value
	return true
}

// Move shifts the selection by delta enabled options, skipping disabled ones,
// and stops at the boundaries. With nothing selected it starts from the first
// (or last, for a negative delta) enabled option.
func (s *Select) Move(delta int) {
	if delta == 0 || len(s.Options) == 0 {
		return
	}
	current := s.Index()
	step := 1
	if delta < 0 {
		step = -1
	}
	if current < 0 {
		current = -1
		if step < 0 {
			current = len(s.Options)
		}
	}
	for n := 0; n < absInt(delta); n++ {
		next := current + step
		for next >= 0 && next < len(s.Options) && s.Options[next].Disabled {
			next += step
		}
		if next < 0 || next >= len(s.Options) {
			break
		}
		current = next
	}
	if current >= 0 && current < len(s.Options) {
		s.Value = s.Options[current].Value
	}
}

// Typeahead selects the next enabled option whose Label or Value starts with
// prefix (case-insensitive), searching after the current selection and
// wrapping around. An empty prefix matches nothing.
func (s *Select) Typeahead(prefix string) bool {
	if prefix == "" || len(s.Options) == 0 {
		return false
	}
	needle := toLowerASCII(prefix)
	count := len(s.Options)
	start := s.Index()
	for i := 1; i <= count; i++ {
		index := start + i
		if start < 0 {
			index = i - 1
		}
		index %= count
		if index < 0 {
			index += count
		}
		option := s.Options[index]
		if option.Disabled {
			continue
		}
		if hasPrefixFold(option.Display(), needle) || hasPrefixFold(option.Value, needle) {
			s.Value = option.Value
			return true
		}
	}
	return false
}

// Build returns the dropdown when Open, otherwise the closed value row.
func (s Select) Build() *sdk.Builder {
	if s.Open {
		return s.BuildDropdown()
	}
	col := s.container()
	if s.Label != "" {
		col.Child(sdk.Text(s.Label).Style(s.LabelStyle).Height(1))
	}
	row := sdk.Row().Height(1)
	if s.Width > 0 {
		row.Width(s.Width)
	}
	option, ok := s.Selected()
	text := s.Placeholder
	style := s.PlaceholderStyle
	if ok {
		text = option.Display()
		style = firstNonEmpty(option.Style, s.SelectedStyle, s.Style)
	}
	if text == "" {
		text = " "
	}
	indicator := firstNonEmpty(s.Indicator, DefaultSelectIndicator)
	total := s.Width
	if total > 0 {
		text = sdk.Truncate(text, maxInt(0, total-sdk.DisplayWidth(indicator)))
	}
	row.Child(sdk.Text(text).Style(style).Height(1))
	row.Child(sdk.Text(indicator).Style(s.PlaceholderStyle).Height(1))
	if total > 0 {
		if pad := total - sdk.DisplayWidth(text) - sdk.DisplayWidth(indicator); pad > 0 {
			row.Child(sdk.Text(padSpaces(pad)))
		}
	}
	col.Child(row)
	return col
}

// BuildDropdown returns the open list of options as a column of one-row
// boxes, ready to be placed in a FloatingLayer or Modal. Disabled options use
// DisabledStyle and the selected option carries the marker.
func (s Select) BuildDropdown() *sdk.Builder {
	col := s.container()
	if s.DropdownStyle != "" {
		col.Style(s.DropdownStyle)
	}
	selected := s.Index()
	marker := firstNonEmpty(s.Marker, DefaultListMarker)
	for i, option := range s.Options {
		text := "  " + option.Display()
		style := firstNonEmpty(option.Style, s.Style)
		if option.Disabled {
			style = firstNonEmpty(s.DisabledStyle, style)
		}
		if i == selected && !option.Disabled {
			text = marker + option.Display()
			style = firstNonEmpty(s.SelectedStyle, style)
		}
		col.Child(listRowBox(text, style, "", s.Width))
	}
	return col
}

func (s Select) container() *sdk.Builder {
	col := sdk.Box().Flow("col")
	if s.ID != "" {
		col.ID(s.ID)
	}
	if s.Width > 0 {
		col.Width(s.Width)
	}
	return col
}

// hasPrefixFold reports whether text starts with lowerPrefix, folding ASCII
// case without allocating a lowercased copy of text.
func hasPrefixFold(text, lowerPrefix string) bool {
	if len(text) < len(lowerPrefix) {
		return false
	}
	for i := 0; i < len(lowerPrefix); i++ {
		if toLowerASCIIByte(text[i]) != lowerPrefix[i] {
			return false
		}
	}
	return true
}

func toLowerASCII(s string) string {
	out := []byte(s)
	for i, b := range out {
		out[i] = toLowerASCIIByte(b)
	}
	return string(out)
}

func toLowerASCIIByte(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

func padSpaces(n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]byte, n)
	for i := range out {
		out[i] = ' '
	}
	return string(out)
}

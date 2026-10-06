package widgets

// Theme is the widget design-token set. Every field is an opaque style string
// (a host-internal token name or an explicit "fg:#RRGGBB;bg:#RRGGBB;bold"
// style), so widgets stay palette-agnostic and the host remains the single
// ANSI resolver. DarkTheme maps onto host tokens; LightTheme pins explicit
// light colors for programs that want a light look independent of the host.
type Theme struct {
	// Name identifies the palette ("dark" or "light"); informational.
	Name string

	Text          string
	Title         string
	Accent        string
	Muted         string
	Success       string
	Warning       string
	Danger        string
	Border        string
	BorderFocus   string
	Selection     string
	SelectionText string
	Input         string
	Placeholder   string
	Cursor        string
	StatusBar     string
	TabActive     string
	TabInactive   string
	Header        string
	Footer        string
	Toast         string
	Overlay       string
	Backdrop      string
	Marker        string
	Separator     string
	Zebra         string
}

// DarkTheme returns the token-based default. Tokens resolve against the host
// palette, so this variant follows the host theme.
func DarkTheme() Theme {
	return Theme{
		Name:          "dark",
		Text:          "",
		Title:         "chrome",
		Accent:        "accent",
		Muted:         "muted",
		Success:       "success",
		Warning:       "warning",
		Danger:        "danger",
		Border:        "border",
		BorderFocus:   "border_focus",
		Selection:     "selection",
		SelectionText: "selection",
		Input:         "",
		Placeholder:   "muted",
		Cursor:        "reverse",
		StatusBar:     "status",
		TabActive:     "tab_active",
		TabInactive:   "tab_inactive",
		Header:        "status",
		Footer:        "muted",
		Toast:         "overlay",
		Overlay:       "overlay",
		Backdrop:      "dim",
		Marker:        "accent",
		Separator:     "muted",
		Zebra:         "muted",
	}
}

// LightTheme returns an explicit light palette for programs that must not
// depend on the host token table.
func LightTheme() Theme {
	return Theme{
		Name:          "light",
		Text:          "fg:#24242a",
		Title:         "fg:#f7f6fa;bg:#6d5ae6;bold",
		Accent:        "fg:#6d3bd4;bold",
		Muted:         "fg:#77737f",
		Success:       "fg:#1d8a4a;bold",
		Warning:       "fg:#a96b00;bold",
		Danger:        "fg:#c02b3a;bold",
		Border:        "fg:#b9b4c4",
		BorderFocus:   "fg:#6d3bd4",
		Selection:     "fg:#24242a;bg:#d9d1f5",
		SelectionText: "fg:#24242a;bg:#d9d1f5",
		Input:         "fg:#24242a",
		Placeholder:   "fg:#9a96a4",
		Cursor:        "reverse",
		StatusBar:     "fg:#3a3743;bg:#e4e1ec",
		TabActive:     "fg:#ffffff;bg:#6d5ae6;bold",
		TabInactive:   "fg:#77737f",
		Header:        "fg:#3a3743;bg:#eceaf2;bold",
		Footer:        "fg:#77737f",
		Toast:         "fg:#24242a;bg:#fff2c2",
		Overlay:       "fg:#24242a;bg:#f7f6fa",
		Backdrop:      "dim",
		Marker:        "fg:#6d3bd4;bold",
		Separator:     "fg:#b9b4c4",
		Zebra:         "fg:#77737f",
	}
}

// DefaultThemeValue is the mutable package-level default returned by
// DefaultTheme; a program may replace it to restyle every themed widget.
var DefaultThemeValue = DarkTheme()

// DefaultTheme returns the current package default theme.
func DefaultTheme() Theme { return DefaultThemeValue }

// ThemeByName resolves "dark"/"light" (empty means dark).
func ThemeByName(name string) (Theme, bool) {
	switch name {
	case "", "dark":
		return DarkTheme(), true
	case "light":
		return LightTheme(), true
	default:
		return Theme{}, false
	}
}

// ThemedList returns a List pre-filled with theme tokens.
func ThemedList(theme Theme) List {
	return List{
		Style:         theme.Text,
		SelectedStyle: theme.Selection,
		HeaderStyle:   theme.Header,
		FooterStyle:   theme.Footer,
		EmptyStyle:    theme.Muted,
		Marker:        theme.Marker,
	}
}

// ThemedVirtualList returns a VirtualList pre-filled with theme tokens.
func ThemedVirtualList(theme Theme) VirtualList {
	return VirtualList{
		Style:         theme.Text,
		SelectedStyle: theme.Selection,
		DisabledStyle: theme.Muted,
		HeaderStyle:   theme.Header,
		FooterStyle:   theme.Footer,
		EmptyStyle:    theme.Muted,
		Marker:        theme.Marker,
	}
}

// ThemedTable returns a Table pre-filled with theme tokens.
func ThemedTable(theme Theme) Table {
	return Table{
		Style:          theme.Text,
		HeaderStyle:    theme.Header,
		SelectedStyle:  theme.Selection,
		ZebraStyle:     theme.Zebra,
		FooterStyle:    theme.Footer,
		SeparatorStyle: theme.Separator,
	}
}

// ThemedTextInput returns a TextInput pre-filled with theme tokens.
func ThemedTextInput(theme Theme) TextInput {
	return TextInput{
		Style:            theme.Input,
		PlaceholderStyle: theme.Placeholder,
		CursorStyle:      theme.Cursor,
	}
}

// ThemedTextArea returns a TextArea pre-filled with theme tokens.
func ThemedTextArea(theme Theme) TextArea {
	return TextArea{
		Style:            theme.Input,
		PlaceholderStyle: theme.Placeholder,
		CursorStyle:      theme.Cursor,
	}
}

// ThemedModal returns a Modal pre-filled with theme tokens.
func ThemedModal(theme Theme) Modal {
	return Modal{Style: theme.BorderFocus, BackdropStyle: theme.Backdrop}
}

// ThemedMenu returns a Menu pre-filled with theme tokens.
func ThemedMenu(theme Theme) Menu {
	return Menu{
		Style:          theme.Text,
		SelectedStyle:  theme.Selection,
		DisabledStyle:  theme.Muted,
		SeparatorStyle: theme.Separator,
		FrameStyle:     theme.Border,
		Marker:         theme.Marker,
	}
}

// ThemedProgressBar returns a ProgressBar pre-filled with theme tokens.
func ThemedProgressBar(theme Theme) ProgressBar {
	return ProgressBar{
		Style:        theme.Accent,
		TrackStyle:   theme.Muted,
		LabelStyle:   theme.Text,
		PercentStyle: theme.Muted,
	}
}

// ThemedSpinner returns a Spinner pre-filled with theme tokens.
func ThemedSpinner(theme Theme) Spinner {
	return Spinner{Style: theme.Accent, LabelStyle: theme.Text}
}

// ThemedBadge returns a Badge pre-filled with theme tokens.
func ThemedBadge(theme Theme) Badge { return Badge{Style: theme.Accent} }

// ThemedTags returns a Tags pre-filled with theme tokens.
func ThemedTags(theme Theme) Tags { return Tags{Style: theme.Text, SepStyle: theme.Separator} }

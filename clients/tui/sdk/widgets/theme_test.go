package widgets

import "testing"

func TestThemeVariants(t *testing.T) {
	if DefaultTheme() != DarkTheme() {
		t.Fatal("DefaultTheme must return the dark palette")
	}
	if DarkTheme() == LightTheme() {
		t.Fatal("dark and light themes must differ")
	}
	if _, ok := ThemeByName("light"); !ok {
		t.Fatal("ThemeByName(light) not found")
	}
	if _, ok := ThemeByName("neon"); ok {
		t.Fatal("ThemeByName must reject unknown names")
	}

	dark := DarkTheme()
	for name, value := range map[string]string{
		"Title":         dark.Title,
		"Accent":        dark.Accent,
		"Muted":         dark.Muted,
		"Danger":        dark.Danger,
		"Success":       dark.Success,
		"Warning":       dark.Warning,
		"Border":        dark.Border,
		"BorderFocus":   dark.BorderFocus,
		"Selection":     dark.Selection,
		"SelectionText": dark.SelectionText,
		"Placeholder":   dark.Placeholder,
		"Cursor":        dark.Cursor,
		"StatusBar":     dark.StatusBar,
		"TabActive":     dark.TabActive,
		"TabInactive":   dark.TabInactive,
		"Header":        dark.Header,
		"Footer":        dark.Footer,
		"Toast":         dark.Toast,
		"Overlay":       dark.Overlay,
		"Backdrop":      dark.Backdrop,
		"Marker":        dark.Marker,
		"Separator":     dark.Separator,
		"Zebra":         dark.Zebra,
	} {
		if value == "" {
			t.Errorf("DarkTheme.%s is empty", name)
		}
	}
	light := LightTheme()
	for name, value := range map[string]string{
		"Text":      light.Text,
		"Accent":    light.Accent,
		"Selection": light.Selection,
		"Overlay":   light.Overlay,
	} {
		if value == "" {
			t.Errorf("LightTheme.%s is empty", name)
		}
	}
}

func TestThemedConstructors(t *testing.T) {
	theme := LightTheme()
	list := ThemedList(theme)
	if list.SelectedStyle != theme.Selection || list.HeaderStyle != theme.Header || list.Marker != theme.Marker {
		t.Fatalf("ThemedList = %+v", list)
	}
	if vl := ThemedVirtualList(theme); vl.DisabledStyle != theme.Muted || vl.SelectedStyle != theme.Selection {
		t.Fatalf("ThemedVirtualList = %+v", vl)
	}
	table := ThemedTable(theme)
	if table.HeaderStyle != theme.Header || table.ZebraStyle != theme.Zebra || table.SeparatorStyle != theme.Separator {
		t.Fatalf("ThemedTable = %+v", table)
	}
	if input := ThemedTextInput(theme); input.PlaceholderStyle != theme.Placeholder || input.CursorStyle != theme.Cursor {
		t.Fatalf("ThemedTextInput = %+v", input)
	}
	if area := ThemedTextArea(theme); area.Style != theme.Input || area.CursorStyle != theme.Cursor {
		t.Fatalf("ThemedTextArea = %+v", area)
	}
	if modal := ThemedModal(theme); modal.Style != theme.BorderFocus || modal.BackdropStyle != theme.Backdrop {
		t.Fatalf("ThemedModal = %+v", modal)
	}
	if menu := ThemedMenu(theme); menu.SelectedStyle != theme.Selection || menu.FrameStyle != theme.Border || menu.Marker != theme.Marker {
		t.Fatalf("ThemedMenu = %+v", menu)
	}
	if bar := ThemedProgressBar(theme); bar.Style != theme.Accent || bar.TrackStyle != theme.Muted {
		t.Fatalf("ThemedProgressBar = %+v", bar)
	}
	if spinner := ThemedSpinner(theme); spinner.Style != theme.Accent {
		t.Fatalf("ThemedSpinner = %+v", spinner)
	}
	if badge := ThemedBadge(theme); badge.Style != theme.Accent {
		t.Fatalf("ThemedBadge = %+v", badge)
	}
	if tags := ThemedTags(theme); tags.Style != theme.Text || tags.SepStyle != theme.Separator {
		t.Fatalf("ThemedTags = %+v", tags)
	}
}

func TestThemeIsOverridable(t *testing.T) {
	previous := DefaultThemeValue
	defer func() { DefaultThemeValue = previous }()
	DefaultThemeValue = LightTheme()
	if DefaultTheme().Name != "light" {
		t.Fatalf("DefaultTheme = %q, want light", DefaultTheme().Name)
	}
}

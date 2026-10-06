package widgets

import (
	"strings"
	"unicode"

	"github.com/anytty/anytty/clients/tui/sdk"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// DefaultBackdropStyle dims everything behind a Modal.
const DefaultBackdropStyle = "dim"

// Modal is a program-side dialog: a Frame placed by FloatingLayer, optionally
// centered in a known parent and optionally sitting on a dimming backdrop.
// Nothing is emitted by the widget; the caller decides what a click/commit
// means and keeps/drops the modal from the next view.
type Modal struct {
	ID            string
	Title         string
	X, Y          int
	Width, Height int
	Rows          []FrameRow
	Style         string
	Backdrop      bool
	BackdropStyle string
	BackdropID    string
	Center        bool
	ParentWidth   int
	ParentHeight  int
}

// Position returns the floating layer origin: (X, Y), or the centered origin
// when Center is set and the parent size is known. It never returns negatives.
func (m Modal) Position() (int, int) {
	x, y := m.X, m.Y
	if m.Center && m.ParentWidth > 0 && m.ParentHeight > 0 {
		x = (m.ParentWidth - m.Width) / 2
		y = (m.ParentHeight - m.Height) / 2
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

// Build returns a Stack: the optional backdrop stretches to fill the parent,
// the framed layer is positioned on top.
func (m Modal) Build() *sdk.Builder {
	x, y := m.Position()
	stack := sdk.Stack()
	if m.ParentWidth > 0 {
		stack.Width(m.ParentWidth)
	}
	if m.ParentHeight > 0 {
		stack.Height(m.ParentHeight)
	}
	if m.Backdrop {
		backdrop := sdk.Box().Style(firstNonEmpty(m.BackdropStyle, DefaultBackdropStyle))
		if m.BackdropID == "" && m.ID != "" {
			m.BackdropID = m.ID + ":backdrop"
		}
		if m.BackdropID != "" {
			backdrop.ID(m.BackdropID)
		}
		if m.ParentWidth > 0 {
			backdrop.Width(m.ParentWidth)
		}
		if m.ParentHeight > 0 {
			backdrop.Height(m.ParentHeight)
		}
		stack.Child(backdrop)
	}
	layer := FloatingLayer{ID: m.ID, Title: m.Title, X: x, Y: y, Width: m.Width, Height: m.Height, Style: m.Style, Rows: m.Rows}
	stack.Child(layer.Build())
	return stack
}

// MenuItem is one Menu row. A Separator item draws a rule, ignores Hotkey and
// is never selectable; a Disabled item is rendered with DisabledStyle and
// skipped by Move/Hotkey. ID is the value carried to the caller (Label is the
// fallback) — the caller dispatches it just like Picker rows.
type MenuItem struct {
	ID        string
	Label     string
	Hotkey    string
	Disabled  bool
	Separator bool
	Style     string
}

// Value returns the item identity emitted by Menu.Value.
func (i MenuItem) Value() string {
	if i.ID != "" {
		return i.ID
	}
	return i.Label
}

// Menu is a keyboard-driven overlay menu: items with optional hotkeys,
// separators and disabled entries. Move/Hotkey update Selected; the caller
// reads Value and dispatches it in the same shape as Button/Picker values
// (source = component id, value = item id).
type Menu struct {
	ID             string
	Title          string
	Items          []MenuItem
	Selected       int
	Width          int
	X, Y           int
	Style          string
	SelectedStyle  string
	DisabledStyle  string
	SeparatorStyle string
	FrameStyle     string
	Marker         string
}

// Move shifts Selected by delta selectable items, skipping separators and
// disabled items; it stops at the boundaries.
func (m *Menu) Move(delta int) {
	if delta == 0 || len(m.Items) == 0 {
		return
	}
	m.Selected = clampIndex(m.Selected, len(m.Items))
	step := 1
	if delta < 0 {
		step = -1
	}
	for n := 0; n < absInt(delta); n++ {
		next := m.Selected + step
		for next >= 0 && next < len(m.Items) && !m.selectable(next) {
			next += step
		}
		if next < 0 || next >= len(m.Items) {
			break
		}
		m.Selected = next
	}
}

// Select moves the selection to index when it is selectable.
func (m *Menu) Select(index int) bool {
	if !m.selectable(index) {
		return false
	}
	m.Selected = index
	return true
}

// Hotkey matches one key event against the item hotkeys (case-insensitive,
// disabled/separator items never match), selects the item and returns its
// value. A non-matching event returns ("", false).
func (m *Menu) Hotkey(ev *pb.KeyEvent) (string, bool) {
	r, ok := printableRune(ev)
	if !ok {
		return "", false
	}
	for i, item := range m.Items {
		if item.Separator || item.Disabled || item.Hotkey == "" {
			continue
		}
		hot := []rune(item.Hotkey)
		if len(hot) == 1 && unicode.ToLower(hot[0]) == unicode.ToLower(r) {
			m.Selected = i
			return item.Value(), true
		}
	}
	return "", false
}

// Value returns the selected item value, empty when nothing selectable is
// selected.
func (m Menu) Value() string {
	item, ok := m.SelectedItem()
	if !ok {
		return ""
	}
	return item.Value()
}

// SelectedItem returns the selected item when it is selectable.
func (m Menu) SelectedItem() (MenuItem, bool) {
	if m.Selected < 0 || m.Selected >= len(m.Items) {
		return MenuItem{}, false
	}
	item := m.Items[m.Selected]
	if item.Separator || item.Disabled {
		return item, false
	}
	return item, true
}

// Build returns the menu as a positioned FloatingLayer.
func (m Menu) Build() *sdk.Builder {
	marker := firstNonEmpty(m.Marker, DefaultListMarker)
	rows := make([]FrameRow, 0, len(m.Items))
	for i, item := range m.Items {
		if item.Separator {
			rows = append(rows, FrameRow{Text: strings.Repeat("─", 3), Style: firstNonEmpty(m.SeparatorStyle, m.DisabledStyle)})
			continue
		}
		text := "  " + item.Label
		style := firstNonEmpty(item.Style, m.Style)
		if i == m.Selected && m.selectable(i) {
			text = marker + item.Label
			style = firstNonEmpty(m.SelectedStyle, style)
		}
		if item.Disabled {
			style = firstNonEmpty(m.DisabledStyle, style)
		}
		row := FrameRow{Text: text, Style: style}
		if item.ID != "" {
			row.ID = item.ID
			row.Input = []string{"mouse"}
		}
		rows = append(rows, row)
	}
	layer := FloatingLayer{ID: m.ID, Title: m.Title, X: m.X, Y: m.Y, Width: m.Width, Style: m.FrameStyle, Rows: rows}
	return layer.Build()
}

func (m Menu) selectable(index int) bool {
	if index < 0 || index >= len(m.Items) {
		return false
	}
	item := m.Items[index]
	return !item.Separator && !item.Disabled
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

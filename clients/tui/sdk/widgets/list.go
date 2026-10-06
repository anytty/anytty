package widgets

import "github.com/anytty/anytty/clients/tui/sdk"

// DefaultListMarker is the selected-row gutter of List and VirtualList.
const DefaultListMarker = "▸ "

// ListRow is one row of a VirtualList: text, an optional hit-test id, an
// optional explicit style and a disabled flag (rendered with DisabledStyle).
type ListRow struct {
	Text     string
	ID       string
	Style    string
	Disabled bool
}

// List is a windowed one-line-per-item list. Build renders only the rows
// inside VisibleRange, so a 100k-item list costs O(Height). Height counts the
// whole widget: Header and Footer each consume one row, the rest is window;
// Height <= 0 falls back to every row. The caller owns Offset/Selected and
// drives them through the helpers; with Follow set the selected row is kept in
// the window without mutating the declared Offset.
type List struct {
	ID            string
	Items         []string
	Width         int
	Height        int
	Offset        int
	Selected      int
	Follow        bool
	Header        string
	Footer        string
	Empty         string
	Marker        string
	Style         string
	SelectedStyle string
	HeaderStyle   string
	FooterStyle   string
	EmptyStyle    string
	RowID         func(index int) string
	RowStyle      func(index int, selected bool) string
}

// VirtualList is List with rich ListRow data. The same windowing rules apply.
type VirtualList struct {
	ID            string
	Rows          []ListRow
	Width         int
	Height        int
	Offset        int
	Selected      int
	Follow        bool
	Header        string
	Footer        string
	Empty         string
	Marker        string
	Style         string
	SelectedStyle string
	DisabledStyle string
	HeaderStyle   string
	FooterStyle   string
	EmptyStyle    string
}

// chromeRows counts the fixed rows a header and a footer occupy.
func chromeRows(header, footer string) int {
	rows := 0
	if header != "" {
		rows++
	}
	if footer != "" {
		rows++
	}
	return rows
}

// windowSize returns the visible window rows for a declared height. A height
// <= 0 means "all rows minus chrome"; it never returns a negative size.
func windowSize(height, chrome, count int) int {
	size := height
	if size <= 0 {
		size = count
	}
	size -= chrome
	if size < 0 {
		size = 0
	}
	if size > count {
		size = count
	}
	return size
}

// clampIndex folds i into [0, count-1]; an empty set selects 0.
func clampIndex(i, count int) int {
	if i < 0 || count <= 0 {
		return 0
	}
	if i >= count {
		return count - 1
	}
	return i
}

// clampOffset folds offset into [0, count-size].
func clampOffset(offset, size, count int) int {
	if size <= 0 {
		return 0
	}
	if offset < 0 {
		return 0
	}
	if max := count - size; offset > max {
		return max
	}
	return offset
}

// followOffset scrolls offset the minimum amount that puts selected inside a
// size-row window.
func followOffset(offset, selected, size int) int {
	if size <= 0 {
		return 0
	}
	if selected < offset {
		offset = selected
	}
	if selected >= offset+size {
		offset = selected - size + 1
	}
	if offset < 0 {
		return 0
	}
	return offset
}

// moveWindow applies a selection delta and follows it when requested.
func moveWindow(offset, selected, delta, size, count int, follow bool) (int, int) {
	selected = clampIndex(selected+delta, count)
	if follow {
		offset = followOffset(offset, selected, size)
	}
	return clampOffset(offset, size, count), selected
}

// VisibleRange returns the half-open row window [start, end) Build renders.
func (l List) VisibleRange() (int, int) {
	size := windowSize(l.Height, chromeRows(l.Header, l.Footer), len(l.Items))
	if size <= 0 {
		return 0, 0
	}
	offset := l.Offset
	if l.Follow {
		offset = followOffset(offset, clampIndex(l.Selected, len(l.Items)), size)
	}
	offset = clampOffset(offset, size, len(l.Items))
	return offset, offset + size
}

// EnsureVisible scrolls the window so Selected is visible.
func (l *List) EnsureVisible() {
	size := windowSize(l.Height, chromeRows(l.Header, l.Footer), len(l.Items))
	if size <= 0 || len(l.Items) == 0 {
		return
	}
	l.Selected = clampIndex(l.Selected, len(l.Items))
	l.Offset = followOffset(l.Offset, l.Selected, size)
	l.Offset = clampOffset(l.Offset, size, len(l.Items))
}

// Move shifts the selection by delta and follows it when Follow is set.
func (l *List) Move(delta int) {
	l.Offset, l.Selected = moveWindow(l.Offset, l.Selected, delta, windowSize(l.Height, chromeRows(l.Header, l.Footer), len(l.Items)), len(l.Items), l.Follow)
}

// PageUp moves the selection one window up.
func (l *List) PageUp() { l.Move(-windowSize(l.Height, chromeRows(l.Header, l.Footer), len(l.Items))) }

// PageDown moves the selection one window down.
func (l *List) PageDown() { l.Move(windowSize(l.Height, chromeRows(l.Header, l.Footer), len(l.Items))) }

// Top selects the first row and scrolls to the start.
func (l *List) Top() {
	l.Selected = 0
	l.Offset = 0
}

// Bottom selects the last row and scrolls to the end.
func (l *List) Bottom() {
	if len(l.Items) == 0 {
		l.Selected, l.Offset = 0, 0
		return
	}
	l.Selected = len(l.Items) - 1
	if size := windowSize(l.Height, chromeRows(l.Header, l.Footer), len(l.Items)); size > 0 {
		l.Offset = clampOffset(len(l.Items)-size, size, len(l.Items))
	}
}

// Build returns the list as a column of text rows.
func (l List) Build() *sdk.Builder {
	col := listBox(l.ID, l.Width, l.Height)
	if l.Header != "" {
		col.Child(listRowBox(l.Header, l.HeaderStyle, "", l.Width))
	}
	start, end := l.VisibleRange()
	switch {
	case end > start:
		for i := start; i < end; i++ {
			col.Child(l.row(i))
		}
	case len(l.Items) == 0 && l.Empty != "":
		col.Child(listRowBox(l.Empty, l.EmptyStyle, "", l.Width))
	}
	if l.Footer != "" {
		col.Child(listRowBox(l.Footer, l.FooterStyle, "", l.Width))
	}
	return col
}

func (l List) row(index int) *sdk.Builder {
	text := "  " + l.Items[index]
	style := l.Style
	if index == l.Selected {
		text = firstNonEmpty(l.Marker, DefaultListMarker) + l.Items[index]
		if l.SelectedStyle != "" {
			style = l.SelectedStyle
		}
	}
	if l.RowStyle != nil {
		if rowStyle := l.RowStyle(index, index == l.Selected); rowStyle != "" {
			style = rowStyle
		}
	}
	id := ""
	if l.RowID != nil {
		id = l.RowID(index)
	}
	return listRowBox(text, style, id, l.Width)
}

// VisibleRange returns the half-open row window [start, end) Build renders.
func (v VirtualList) VisibleRange() (int, int) {
	size := windowSize(v.Height, chromeRows(v.Header, v.Footer), len(v.Rows))
	if size <= 0 {
		return 0, 0
	}
	offset := v.Offset
	if v.Follow {
		offset = followOffset(offset, clampIndex(v.Selected, len(v.Rows)), size)
	}
	offset = clampOffset(offset, size, len(v.Rows))
	return offset, offset + size
}

// EnsureVisible scrolls the window so Selected is visible.
func (v *VirtualList) EnsureVisible() {
	size := windowSize(v.Height, chromeRows(v.Header, v.Footer), len(v.Rows))
	if size <= 0 || len(v.Rows) == 0 {
		return
	}
	v.Selected = clampIndex(v.Selected, len(v.Rows))
	v.Offset = followOffset(v.Offset, v.Selected, size)
	v.Offset = clampOffset(v.Offset, size, len(v.Rows))
}

// Move shifts the selection by delta and follows it when Follow is set.
func (v *VirtualList) Move(delta int) {
	v.Offset, v.Selected = moveWindow(v.Offset, v.Selected, delta, windowSize(v.Height, chromeRows(v.Header, v.Footer), len(v.Rows)), len(v.Rows), v.Follow)
}

// PageUp moves the selection one window up.
func (v *VirtualList) PageUp() {
	v.Move(-windowSize(v.Height, chromeRows(v.Header, v.Footer), len(v.Rows)))
}

// PageDown moves the selection one window down.
func (v *VirtualList) PageDown() {
	v.Move(windowSize(v.Height, chromeRows(v.Header, v.Footer), len(v.Rows)))
}

// Top selects the first row and scrolls to the start.
func (v *VirtualList) Top() {
	v.Selected = 0
	v.Offset = 0
}

// Bottom selects the last row and scrolls to the end.
func (v *VirtualList) Bottom() {
	if len(v.Rows) == 0 {
		v.Selected, v.Offset = 0, 0
		return
	}
	v.Selected = len(v.Rows) - 1
	if size := windowSize(v.Height, chromeRows(v.Header, v.Footer), len(v.Rows)); size > 0 {
		v.Offset = clampOffset(len(v.Rows)-size, size, len(v.Rows))
	}
}

// Build returns the list as a column of text rows.
func (v VirtualList) Build() *sdk.Builder {
	col := listBox(v.ID, v.Width, v.Height)
	if v.Header != "" {
		col.Child(listRowBox(v.Header, v.HeaderStyle, "", v.Width))
	}
	start, end := v.VisibleRange()
	switch {
	case end > start:
		for i := start; i < end; i++ {
			col.Child(v.row(i))
		}
	case len(v.Rows) == 0 && v.Empty != "":
		col.Child(listRowBox(v.Empty, v.EmptyStyle, "", v.Width))
	}
	if v.Footer != "" {
		col.Child(listRowBox(v.Footer, v.FooterStyle, "", v.Width))
	}
	return col
}

func (v VirtualList) row(index int) *sdk.Builder {
	item := v.Rows[index]
	text := "  " + item.Text
	style := firstNonEmpty(item.Style, v.Style)
	if item.Disabled {
		style = firstNonEmpty(v.DisabledStyle, style)
	}
	if index == v.Selected {
		text = firstNonEmpty(v.Marker, DefaultListMarker) + item.Text
		style = firstNonEmpty(v.SelectedStyle, style)
	}
	return listRowBox(text, style, item.ID, v.Width)
}

func listBox(id string, width, height int) *sdk.Builder {
	col := sdk.Box().Flow("col")
	if id != "" {
		col.ID(id)
	}
	if width > 0 {
		col.Width(width)
	}
	if height > 0 {
		col.Height(height)
	}
	return col
}

func listRowBox(text, style, id string, width int) *sdk.Builder {
	if width > 0 {
		text = padTo(text, width)
	}
	box := sdk.Text(text).Height(1)
	if width > 0 {
		box.Width(width)
	}
	if style != "" {
		box.Style(style)
	}
	if id != "" {
		box.ID(id).Input("mouse")
	}
	return box
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

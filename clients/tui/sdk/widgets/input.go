package widgets

import (
	"strings"
	"unicode"

	"github.com/anytty/anytty/clients/tui/sdk"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// DefaultCursorStyle is the style of the rendered cursor cell when the widget
// declares no CursorStyle.
const DefaultCursorStyle = "reverse"

// DefaultCursorShape is the protocol cursor shape declared by focused inputs.
const DefaultCursorShape = "bar"

// TextInput is a single-line editor state plus a pure Build. The caller owns
// the state: feed keys through HandleKey (or call the editing methods) and
// Commit the view itself; the widget never sends frames. Cursor is a rune
// index into Value; Offset is the first visible rune (horizontal scroll).
type TextInput struct {
	ID               string
	Value            []rune
	Cursor           int
	Placeholder      string
	Mask             rune
	MaxLen           int
	Width            int
	Offset           int
	Focused          bool
	Style            string
	PlaceholderStyle string
	CursorStyle      string
	CursorShape      string
	Input            []string
}

// Text returns the raw value (Mask only affects rendering).
func (t TextInput) Text() string { return string(t.Value) }

// DisplayText returns the rendered text: masked value, or the placeholder
// when the value is empty.
func (t TextInput) DisplayText() string {
	if len(t.Value) == 0 && t.Placeholder != "" {
		return t.Placeholder
	}
	return string(t.displayRunes())
}

// SetValue replaces the value and places the cursor at the end.
func (t *TextInput) SetValue(s string) {
	t.Value = filterPlainRunes([]rune(s))
	if t.MaxLen > 0 && len(t.Value) > t.MaxLen {
		t.Value = t.Value[:t.MaxLen]
	}
	t.Cursor = len(t.Value)
}

// InsertRune inserts r at the cursor. Newlines are rejected (single line).
func (t *TextInput) InsertRune(r rune) bool {
	if r == '\n' || r == '\r' {
		return false
	}
	if t.MaxLen > 0 && len(t.Value) >= t.MaxLen {
		return false
	}
	insertRuneAt(&t.Value, &t.Cursor, r)
	return true
}

// InsertString inserts s at the cursor, clipped to MaxLen; newlines are
// stripped. It reports whether anything was inserted.
func (t *TextInput) InsertString(s string) bool {
	runes := filterPlainRunes([]rune(s))
	if len(runes) == 0 {
		return false
	}
	if t.MaxLen > 0 {
		room := t.MaxLen - len(t.Value)
		if room <= 0 {
			return false
		}
		if len(runes) > room {
			runes = runes[:room]
		}
	}
	insertRunesAt(&t.Value, &t.Cursor, runes)
	return true
}

// Backspace deletes the rune before the cursor.
func (t *TextInput) Backspace() bool {
	t.Cursor = clampCursor(t.Cursor, len(t.Value))
	if t.Cursor == 0 {
		return false
	}
	t.Value = append(t.Value[:t.Cursor-1], t.Value[t.Cursor:]...)
	t.Cursor--
	return true
}

// Delete deletes the rune at the cursor.
func (t *TextInput) Delete() bool {
	t.Cursor = clampCursor(t.Cursor, len(t.Value))
	if t.Cursor >= len(t.Value) {
		return false
	}
	t.Value = append(t.Value[:t.Cursor], t.Value[t.Cursor+1:]...)
	return true
}

// DeleteWordLeft deletes from the cursor back to the previous word start.
func (t *TextInput) DeleteWordLeft() bool {
	t.Cursor = clampCursor(t.Cursor, len(t.Value))
	start := wordLeftIndex(t.Value, t.Cursor)
	if start == t.Cursor {
		return false
	}
	t.Value = append(t.Value[:start], t.Value[t.Cursor:]...)
	t.Cursor = start
	return true
}

// DeleteToEnd deletes from the cursor to the end of the value.
func (t *TextInput) DeleteToEnd() bool {
	t.Cursor = clampCursor(t.Cursor, len(t.Value))
	if t.Cursor >= len(t.Value) {
		return false
	}
	t.Value = t.Value[:t.Cursor]
	return true
}

// Left moves the cursor one rune left.
func (t *TextInput) Left() bool {
	t.Cursor = clampCursor(t.Cursor, len(t.Value))
	if t.Cursor == 0 {
		return false
	}
	t.Cursor--
	return true
}

// Right moves the cursor one rune right.
func (t *TextInput) Right() bool {
	t.Cursor = clampCursor(t.Cursor, len(t.Value))
	if t.Cursor >= len(t.Value) {
		return false
	}
	t.Cursor++
	return true
}

// WordLeft moves the cursor to the previous word start.
func (t *TextInput) WordLeft() bool {
	t.Cursor = clampCursor(t.Cursor, len(t.Value))
	next := wordLeftIndex(t.Value, t.Cursor)
	if next == t.Cursor {
		return false
	}
	t.Cursor = next
	return true
}

// WordRight moves the cursor past the next word.
func (t *TextInput) WordRight() bool {
	t.Cursor = clampCursor(t.Cursor, len(t.Value))
	next := wordRightIndex(t.Value, t.Cursor)
	if next == t.Cursor {
		return false
	}
	t.Cursor = next
	return true
}

// Home moves the cursor to the start.
func (t *TextInput) Home() bool {
	moved := t.Cursor != 0
	t.Cursor = 0
	return moved
}

// End moves the cursor past the last rune.
func (t *TextInput) End() bool {
	moved := t.Cursor != len(t.Value)
	t.Cursor = len(t.Value)
	return moved
}

// HandleKey applies one protocol key event and reports whether the input
// consumed it. Enter/Tab/Esc/Up/Down are left for the caller (Commit).
func (t *TextInput) HandleKey(ev *pb.KeyEvent) bool {
	if ev == nil {
		return false
	}
	switch ev.GetKey() {
	case "left":
		t.Left()
		return true
	case "right":
		t.Right()
		return true
	case "home", "ctrl-a":
		t.Home()
		return true
	case "end", "ctrl-e":
		t.End()
		return true
	case "backspace":
		t.Backspace()
		return true
	case "delete":
		t.Delete()
		return true
	case "ctrl-left", "alt-left", "ctrl-b":
		t.WordLeft()
		return true
	case "ctrl-right", "alt-right", "ctrl-f":
		t.WordRight()
		return true
	case "ctrl-w", "alt-backspace":
		t.DeleteWordLeft()
		return true
	case "ctrl-k":
		t.DeleteToEnd()
		return true
	case "ctrl-u":
		t.DeleteToEnd()
		t.Home()
		return true
	}
	if r, ok := printableRune(ev); ok {
		t.InsertRune(r)
		return true
	}
	return false
}

// Build returns the input as a one-row box tree. A focused input carries the
// program cursor; the cursor cell is rendered with CursorStyle (default
// reverse) so the row reads as one text run.
func (t TextInput) Build() *sdk.Builder {
	box := inputRowBox(t.ID, t.Width)
	box.Focused(t.Focused).Input(t.inputs()...)
	if len(t.Value) == 0 && t.Placeholder != "" {
		text := t.Placeholder
		if t.Width > 0 {
			text = sdk.Truncate(text, t.Width)
		}
		box.Child(sdk.Text(text).Style(t.PlaceholderStyle).Height(1))
		t.pad(box, sdk.DisplayWidth(text))
		if t.Focused {
			box.Cursor(0, 0, cursorShape(t.CursorShape))
		}
		return box
	}
	runes := t.displayRunes()
	cursor := clampCursor(t.Cursor, len(runes))
	start := t.visibleOffset(runes)
	visible := runes[start:]
	if !t.Focused {
		text := string(visible)
		if t.Width > 0 {
			text = padTo(text, t.Width)
		}
		if text != "" {
			box.Child(sdk.Text(text).Style(t.Style).Height(1))
		}
		return box
	}
	rel := clampCursor(cursor-start, len(visible))
	pre := string(visible[:rel])
	post := visible[rel:]
	cursorText := " "
	if len(post) > 0 {
		cursorText = string(post[0])
		post = post[1:]
	}
	postText := string(post)
	if t.Width > 0 {
		postText = sdk.Truncate(postText, t.Width-sdk.DisplayWidth(pre)-sdk.DisplayWidth(cursorText))
	}
	if pre != "" {
		box.Child(sdk.Text(pre).Style(t.Style).Height(1))
	}
	box.Child(sdk.Text(cursorText).Style(firstNonEmpty(t.CursorStyle, DefaultCursorStyle)).Height(1))
	if postText != "" {
		box.Child(sdk.Text(postText).Style(t.Style).Height(1))
	}
	t.pad(box, sdk.DisplayWidth(pre)+sdk.DisplayWidth(cursorText)+sdk.DisplayWidth(postText))
	box.Cursor(0, sdk.DisplayWidth(pre), cursorShape(t.CursorShape))
	return box
}

func (t TextInput) displayRunes() []rune {
	if t.Mask == 0 {
		return t.Value
	}
	out := make([]rune, len(t.Value))
	for i := range out {
		out[i] = t.Mask
	}
	return out
}

// visibleOffset scrolls the rune window so the cursor cell fits Width cells.
func (t TextInput) visibleOffset(runes []rune) int {
	if t.Width <= 0 {
		return 0
	}
	offset := clampCursor(t.Offset, len(runes))
	cursor := clampCursor(t.Cursor, len(runes))
	if cursor < offset {
		offset = cursor
	}
	for offset < cursor && cellWidth(runes[offset:cursor])+cursorCellWidth(runes, cursor) > t.Width {
		offset++
	}
	return offset
}

func (t TextInput) pad(box *sdk.Builder, used int) {
	if t.Width <= 0 || used >= t.Width {
		return
	}
	box.Child(sdk.Text(strings.Repeat(" ", t.Width-used)))
}

func (t TextInput) inputs() []string {
	if len(t.Input) > 0 {
		return t.Input
	}
	return []string{"key", "paste"}
}

// TextArea is the multi-line counterpart of TextInput: Cursor is a rune index
// into Value and RowOffset/ColOffset are the first visible line/column. Build
// is pure and renders the visible rows only.
type TextArea struct {
	ID               string
	Value            []rune
	Cursor           int
	Placeholder      string
	Mask             rune
	MaxLen           int
	Width            int
	Height           int
	RowOffset        int
	ColOffset        int
	Focused          bool
	Style            string
	PlaceholderStyle string
	CursorStyle      string
	CursorShape      string
	Input            []string
}

// Text returns the raw value.
func (a TextArea) Text() string { return string(a.Value) }

// Lines splits the raw value into lines.
func (a TextArea) Lines() []string { return strings.Split(string(a.Value), "\n") }

// Line returns one raw line, empty when out of range.
func (a TextArea) Line(row int) string {
	lines := a.Lines()
	if row < 0 || row >= len(lines) {
		return ""
	}
	return lines[row]
}

// RowCol returns the cursor line and its rune column.
func (a TextArea) RowCol() (int, int) {
	cursor := clampCursor(a.Cursor, len(a.Value))
	row, start := 0, 0
	for i := 0; i < cursor; i++ {
		if a.Value[i] == '\n' {
			row++
			start = i + 1
		}
	}
	return row, cursor - start
}

// SetValue replaces the value and places the cursor at the end.
func (a *TextArea) SetValue(s string) {
	a.Value = filterRunes([]rune(strings.ReplaceAll(s, "\r\n", "\n")), true)
	if a.MaxLen > 0 && len(a.Value) > a.MaxLen {
		a.Value = a.Value[:a.MaxLen]
	}
	a.Cursor = len(a.Value)
}

// InsertRune inserts r at the cursor; "\n" starts a new line.
func (a *TextArea) InsertRune(r rune) bool {
	if r == '\r' {
		return false
	}
	if a.MaxLen > 0 && len(a.Value) >= a.MaxLen {
		return false
	}
	insertRuneAt(&a.Value, &a.Cursor, r)
	return true
}

// InsertString inserts s at the cursor, clipped to MaxLen. CRLF is normalized.
func (a *TextArea) InsertString(s string) bool {
	runes := filterRunes([]rune(strings.ReplaceAll(s, "\r\n", "\n")), true)
	if len(runes) == 0 {
		return false
	}
	if a.MaxLen > 0 {
		room := a.MaxLen - len(a.Value)
		if room <= 0 {
			return false
		}
		if len(runes) > room {
			runes = runes[:room]
		}
	}
	insertRunesAt(&a.Value, &a.Cursor, runes)
	return true
}

// Backspace deletes the rune before the cursor (joining lines at a line start).
func (a *TextArea) Backspace() bool {
	a.Cursor = clampCursor(a.Cursor, len(a.Value))
	if a.Cursor == 0 {
		return false
	}
	a.Value = append(a.Value[:a.Cursor-1], a.Value[a.Cursor:]...)
	a.Cursor--
	return true
}

// Delete deletes the rune at the cursor.
func (a *TextArea) Delete() bool {
	a.Cursor = clampCursor(a.Cursor, len(a.Value))
	if a.Cursor >= len(a.Value) {
		return false
	}
	a.Value = append(a.Value[:a.Cursor], a.Value[a.Cursor+1:]...)
	return true
}

// DeleteWordLeft deletes from the cursor back to the previous word start.
func (a *TextArea) DeleteWordLeft() bool {
	a.Cursor = clampCursor(a.Cursor, len(a.Value))
	start := wordLeftIndex(a.Value, a.Cursor)
	if start == a.Cursor {
		return false
	}
	a.Value = append(a.Value[:start], a.Value[a.Cursor:]...)
	a.Cursor = start
	return true
}

// Left moves the cursor one rune left.
func (a *TextArea) Left() bool {
	a.Cursor = clampCursor(a.Cursor, len(a.Value))
	if a.Cursor == 0 {
		return false
	}
	a.Cursor--
	return true
}

// Right moves the cursor one rune right.
func (a *TextArea) Right() bool {
	a.Cursor = clampCursor(a.Cursor, len(a.Value))
	if a.Cursor >= len(a.Value) {
		return false
	}
	a.Cursor++
	return true
}

// WordLeft moves the cursor to the previous word start.
func (a *TextArea) WordLeft() bool {
	a.Cursor = clampCursor(a.Cursor, len(a.Value))
	next := wordLeftIndex(a.Value, a.Cursor)
	if next == a.Cursor {
		return false
	}
	a.Cursor = next
	return true
}

// WordRight moves the cursor past the next word.
func (a *TextArea) WordRight() bool {
	a.Cursor = clampCursor(a.Cursor, len(a.Value))
	next := wordRightIndex(a.Value, a.Cursor)
	if next == a.Cursor {
		return false
	}
	a.Cursor = next
	return true
}

// Up moves the cursor one line up, keeping the column when possible.
func (a *TextArea) Up() bool {
	row, col := a.RowCol()
	if row == 0 {
		return false
	}
	lines := a.lineBounds()
	start, end := lines[row-1][0], lines[row-1][1]
	a.Cursor = clampCursor(start+col, end)
	return true
}

// Down moves the cursor one line down, keeping the column when possible.
func (a *TextArea) Down() bool {
	row, col := a.RowCol()
	lines := a.lineBounds()
	if row >= len(lines)-1 {
		return false
	}
	start, end := lines[row+1][0], lines[row+1][1]
	a.Cursor = clampCursor(start+col, end)
	return true
}

// Home moves the cursor to the start of its line.
func (a *TextArea) Home() bool {
	row, _ := a.RowCol()
	start := a.lineBounds()[row][0]
	moved := a.Cursor != start
	a.Cursor = start
	return moved
}

// End moves the cursor to the end of its line.
func (a *TextArea) End() bool {
	row, _ := a.RowCol()
	end := a.lineBounds()[row][1]
	moved := a.Cursor != end
	a.Cursor = end
	return moved
}

// PageUp moves the cursor up one visible page.
func (a *TextArea) PageUp() bool {
	moved := false
	for i := 0; i < a.pageHeight(); i++ {
		if !a.Up() {
			break
		}
		moved = true
	}
	return moved
}

// PageDown moves the cursor down one visible page.
func (a *TextArea) PageDown() bool {
	moved := false
	for i := 0; i < a.pageHeight(); i++ {
		if !a.Down() {
			break
		}
		moved = true
	}
	return moved
}

// HandleKey applies one protocol key event and reports whether the textarea
// consumed it. Enter inserts a newline; Esc/Tab are left for the caller.
func (a *TextArea) HandleKey(ev *pb.KeyEvent) bool {
	if ev == nil {
		return false
	}
	switch ev.GetKey() {
	case "left":
		a.Left()
		return true
	case "right":
		a.Right()
		return true
	case "up":
		a.Up()
		return true
	case "down":
		a.Down()
		return true
	case "home", "ctrl-a":
		a.Home()
		return true
	case "end", "ctrl-e":
		a.End()
		return true
	case "page-up":
		a.PageUp()
		return true
	case "page-down":
		a.PageDown()
		return true
	case "backspace":
		a.Backspace()
		return true
	case "delete":
		a.Delete()
		return true
	case "ctrl-left", "alt-left", "ctrl-b":
		a.WordLeft()
		return true
	case "ctrl-right", "alt-right", "ctrl-f":
		a.WordRight()
		return true
	case "ctrl-w", "alt-backspace":
		a.DeleteWordLeft()
		return true
	case "enter", "ctrl-j", "ctrl-m":
		a.InsertRune('\n')
		return true
	}
	if r, ok := printableRune(ev); ok {
		a.InsertRune(r)
		return true
	}
	return false
}

// Build returns the textarea as a column of visible line rows.
func (a TextArea) Build() *sdk.Builder {
	col := sdk.Box().Flow("col")
	if a.ID != "" {
		col.ID(a.ID)
	}
	col.Focused(a.Focused).Input(a.inputs()...)
	if a.Width > 0 {
		col.Width(a.Width)
	}
	if a.Height > 0 {
		col.Height(a.Height)
	}
	if len(a.Value) == 0 && a.Placeholder != "" {
		text := a.Placeholder
		if a.Width > 0 {
			text = padTo(text, a.Width)
		}
		col.Child(sdk.Text(text).Style(a.PlaceholderStyle).Height(1))
		if a.Focused {
			col.Cursor(0, 0, cursorShape(a.CursorShape))
		}
		return col
	}
	lines := strings.Split(string(a.displayRunes()), "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	cursorRow, cursorCol := a.RowCol()
	offset := a.rowOffset(len(lines))
	colOffset, cursorDisplayCol := a.colWindow(lines, cursorRow, cursorCol)
	height := a.visibleHeight(len(lines) - offset)
	for i := 0; i < height; i++ {
		row := offset + i
		col.Child(a.lineBox([]rune(lines[row]), row == cursorRow, cursorCol, colOffset))
	}
	if a.Focused {
		row := cursorRow - offset
		if row < 0 {
			row = 0
		}
		col.Cursor(row, cursorDisplayCol, cursorShape(a.CursorShape))
	}
	return col
}

func (a TextArea) displayRunes() []rune {
	if a.Mask == 0 {
		return a.Value
	}
	out := make([]rune, len(a.Value))
	for i, r := range a.Value {
		if r == '\n' {
			out[i] = r
			continue
		}
		out[i] = a.Mask
	}
	return out
}

// lineBounds returns [start, end) rune bounds per line, end excluding "\n".
func (a TextArea) lineBounds() [][2]int {
	bounds := [][2]int{{0, 0}}
	for i, r := range a.Value {
		if r == '\n' {
			bounds[len(bounds)-1][1] = i
			bounds = append(bounds, [2]int{i + 1, i + 1})
		}
	}
	bounds[len(bounds)-1][1] = len(a.Value)
	return bounds
}

func (a TextArea) rowOffset(lines int) int {
	height := a.visibleHeight(lines)
	offset := a.RowOffset
	cursorRow, _ := a.RowCol()
	if height <= 0 {
		return 0
	}
	if cursorRow < offset {
		offset = cursorRow
	}
	if cursorRow >= offset+height {
		offset = cursorRow - height + 1
	}
	if max := lines - height; offset > max {
		offset = max
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}

func (a TextArea) visibleHeight(lines int) int {
	height := a.Height
	if height <= 0 {
		height = lines
	}
	if height < 0 {
		return 0
	}
	if height > lines {
		height = lines
	}
	return height
}

func (a TextArea) pageHeight() int {
	if a.Height > 1 {
		return a.Height
	}
	return 1
}

// colWindow returns the horizontal offset and the cursor display column.
func (a TextArea) colWindow(lines []string, cursorRow, cursorCol int) (int, int) {
	line := lineRunes(lines, cursorRow)
	if a.Width <= 0 {
		return 0, cellWidth(line[:clampCursor(cursorCol, len(line))])
	}
	offset := a.ColOffset
	if offset < 0 {
		offset = 0
	}
	if cursorCol < offset {
		offset = cursorCol
	}
	if offset > len(line) {
		offset = len(line)
	}
	for offset < cursorCol && cellWidth(line[offset:cursorCol])+cursorCellWidth(line, cursorCol) > a.Width {
		offset++
	}
	if offset > cursorCol {
		offset = cursorCol
	}
	col := cellWidth(line[offset:clampCursor(cursorCol, len(line))])
	return offset, col
}

func lineRunes(lines []string, row int) []rune {
	if row < 0 || row >= len(lines) {
		return nil
	}
	return []rune(lines[row])
}

func (a TextArea) lineBox(line []rune, cursorLine bool, cursorCol, offset int) *sdk.Builder {
	row := sdk.Row().Height(1)
	if a.Width > 0 {
		row.Width(a.Width)
	}
	start := clampCursor(offset, len(line))
	visible := line[start:]
	if a.Width > 0 {
		visible = truncateRunesToCells(visible, a.Width)
	}
	if !cursorLine || !a.Focused {
		text := string(visible)
		if a.Width > 0 {
			text = padTo(text, a.Width)
		}
		if text != "" {
			row.Child(sdk.Text(text).Style(a.Style).Height(1))
		}
		return row
	}
	rel := clampCursor(cursorCol-start, len(visible))
	pre := string(visible[:rel])
	post := visible[rel:]
	cursorText := " "
	if len(post) > 0 {
		cursorText = string(post[0])
		post = post[1:]
	}
	postText := string(post)
	if a.Width > 0 {
		postText = sdk.Truncate(postText, a.Width-sdk.DisplayWidth(pre)-sdk.DisplayWidth(cursorText))
	}
	if pre != "" {
		row.Child(sdk.Text(pre).Style(a.Style).Height(1))
	}
	row.Child(sdk.Text(cursorText).Style(firstNonEmpty(a.CursorStyle, DefaultCursorStyle)).Height(1))
	if postText != "" {
		row.Child(sdk.Text(postText).Style(a.Style).Height(1))
	}
	if used := sdk.DisplayWidth(pre) + sdk.DisplayWidth(cursorText) + sdk.DisplayWidth(postText); a.Width > used {
		row.Child(sdk.Text(strings.Repeat(" ", a.Width-used)))
	}
	return row
}

func (a TextArea) inputs() []string {
	if len(a.Input) > 0 {
		return a.Input
	}
	return []string{"key", "paste"}
}

func inputRowBox(id string, width int) *sdk.Builder {
	box := sdk.Row().Height(1)
	if id != "" {
		box.ID(id)
	}
	if width > 0 {
		box.Width(width)
	}
	return box
}

func cursorShape(shape string) string {
	if shape != "" {
		return shape
	}
	return DefaultCursorShape
}

func insertRuneAt(value *[]rune, cursor *int, r rune) {
	*cursor = clampCursor(*cursor, len(*value))
	*value = append(*value, 0)
	copy((*value)[*cursor+1:], (*value)[*cursor:])
	(*value)[*cursor] = r
	*cursor++
}

func insertRunesAt(value *[]rune, cursor *int, runes []rune) {
	*cursor = clampCursor(*cursor, len(*value))
	tail := append([]rune(nil), (*value)[*cursor:]...)
	*value = append((*value)[:*cursor], runes...)
	*value = append(*value, tail...)
	*cursor += len(runes)
}

func filterPlainRunes(runes []rune) []rune {
	return filterRunes(runes, false)
}

func filterRunes(runes []rune, newlines bool) []rune {
	out := make([]rune, 0, len(runes))
	for _, r := range runes {
		if r == '\n' && !newlines {
			continue
		}
		if r == '\r' {
			continue
		}
		out = append(out, r)
	}
	return out
}

func clampCursor(cursor, length int) int {
	if cursor < 0 {
		return 0
	}
	if cursor > length {
		return length
	}
	return cursor
}

func printableRune(ev *pb.KeyEvent) (rune, bool) {
	text := ev.GetChar()
	if text == "" {
		text = ev.GetKey()
	}
	runes := []rune(text)
	if len(runes) != 1 || unicode.IsControl(runes[0]) {
		return 0, false
	}
	return runes[0], true
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func wordLeftIndex(value []rune, cursor int) int {
	cursor = clampCursor(cursor, len(value))
	i := cursor
	for i > 0 && !isWordRune(value[i-1]) {
		i--
	}
	for i > 0 && isWordRune(value[i-1]) {
		i--
	}
	return i
}

func wordRightIndex(value []rune, cursor int) int {
	cursor = clampCursor(cursor, len(value))
	i := cursor
	for i < len(value) && isWordRune(value[i]) {
		i++
	}
	for i < len(value) && !isWordRune(value[i]) {
		i++
	}
	return i
}

func cellWidth(runes []rune) int {
	width := 0
	for _, r := range runes {
		width += sdk.RuneWidth(r)
	}
	return width
}

func cursorCellWidth(runes []rune, cursor int) int {
	if cursor >= 0 && cursor < len(runes) {
		if width := sdk.RuneWidth(runes[cursor]); width > 0 {
			return width
		}
	}
	return 1
}

func truncateRunesToCells(runes []rune, width int) []rune {
	if width <= 0 {
		return nil
	}
	cells := 0
	for i, r := range runes {
		rw := sdk.RuneWidth(r)
		if rw > 0 && cells+rw > width {
			return runes[:i]
		}
		cells += rw
	}
	return runes
}

// Package widgets is the layout-program widget toolkit: pure builders that
// turn a small config struct into an SDK box tree. Widgets hold no state and
// never emit protocol methods; clicks and keys are handled by the caller
// after hit-testing the box id, exactly like every other box ("a button is
// a box plus program-side hit handling" — CUSTOMIZE §5).
//
// Every widget takes its colors from style token names, so the host palette
// (render.ThemeDark/ThemeLight) stays the single source of ANSI resolution.
package widgets

import (
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
)

// Default style tokens used by the widgets; callers may override each one.
const (
	DefaultActiveStyle   = "tab_active"
	DefaultInactiveStyle = "tab_inactive"
	DefaultChromeStyle   = "chrome"
	DefaultStatusStyle   = "status"
	DefaultSepStyle      = "muted"
	DefaultKeyStyle      = "muted"
	DefaultBorderStyle   = "border"
	DefaultButtonStyle   = "accent"
	DefaultSeparator     = "│"
)

// Segment is one styled text run of a bar.
type Segment struct {
	Text  string
	Style string
	ID    string
	Input []string
}

// TabItem is one tab: the id the caller hit-tests, the visible title and
// whether it is the active one.
type TabItem struct {
	ID     string
	Title  string
	Active bool
}

// TabBar builds the top tab strip: an optional left segment (workspace), one
// clickable text box per tab, and an optional "+" new-tab box.
type TabBar struct {
	Left          *Segment
	Items         []TabItem
	Plus          bool
	PlusID        string
	PlusText      string
	ActiveStyle   string
	InactiveStyle string
	ChromeStyle   string
}

// Build returns the tab strip as a one-row box tree. The caller adds
// ID("header")/Height(1) as needed.
func (t TabBar) Build() *sdk.Builder {
	row := sdk.Row()
	if t.Left != nil {
		row.Child(segmentBox(*t.Left, t.chromeStyle()))
	}
	for _, item := range t.Items {
		label := " " + item.Title + " "
		style := t.inactiveStyle()
		if item.Active {
			label = "[" + item.Title + "]"
			style = t.activeStyle()
		}
		row.Child(sdk.Text(label).ID(item.ID).Style(style).Input("mouse"))
	}
	if t.Plus {
		id := t.PlusID
		if id == "" {
			id = "tab:new"
		}
		text := t.PlusText
		if text == "" {
			text = " + "
		}
		row.Child(sdk.Text(text).ID(id).Style(t.chromeStyle()).Input("mouse"))
	}
	return row
}

func (t TabBar) activeStyle() string {
	if t.ActiveStyle != "" {
		return t.ActiveStyle
	}
	return DefaultActiveStyle
}

func (t TabBar) inactiveStyle() string {
	if t.InactiveStyle != "" {
		return t.InactiveStyle
	}
	return DefaultInactiveStyle
}

func (t TabBar) chromeStyle() string {
	if t.ChromeStyle != "" {
		return t.ChromeStyle
	}
	return DefaultChromeStyle
}

// StatusBar builds a one-row status line from styled segments. With
// Width > 0 the right group is pushed to the right edge and the left group
// is trimmed (last segment first, then text) until the row fits.
type StatusBar struct {
	Left      []Segment
	Right     []Segment
	Width     int
	Separator string
	SepStyle  string
	ID        string
}

// Text renders the segments as one plain string, so callers can measure the
// bar before building it.
func (s StatusBar) Text() string {
	return strings.Join(s.segmentTexts(), s.separatorText())
}

func (s StatusBar) segmentTexts() []string {
	out := make([]string, 0, len(s.Left)+len(s.Right))
	for _, seg := range s.Left {
		out = append(out, seg.Text)
	}
	for _, seg := range s.Right {
		out = append(out, seg.Text)
	}
	return out
}

func (s StatusBar) separatorText() string {
	if s.Separator != "" {
		return " " + s.Separator + " "
	}
	return " " + DefaultSeparator + " "
}

func (s StatusBar) sepStyle() string {
	if s.SepStyle != "" {
		return s.SepStyle
	}
	return DefaultSepStyle
}

// Build returns the status bar as a row of text boxes.
func (s StatusBar) Build() *sdk.Builder {
	row := sdk.Row().Height(1)
	if s.ID != "" {
		row.ID(s.ID)
	}
	left, right := s.Left, s.Right
	sepW := sdk.DisplayWidth(s.separatorText())
	if s.Width > 0 {
		right, _ = fitSegments(right, s.Width, sepW)
		rightW := groupWidth(right, sepW)
		leftBudget := 0
		if len(left) > 0 {
			leftBudget = s.Width - rightW
			if len(right) > 0 {
				leftBudget -= sepW
			}
		}
		left, _ = fitSegments(left, leftBudget, sepW)
	}
	used := groupWidth(left, sepW) + groupWidth(right, sepW)
	if len(left) > 0 && len(right) > 0 {
		used += sepW
	}
	appendGroup(row, left, s.sepStyle(), s.separatorText())
	if len(left) > 0 && len(right) > 0 {
		row.Child(sdk.Text(s.separatorText()).Style(s.sepStyle()))
	}
	if s.Width > 0 {
		if pad := s.Width - used; pad > 0 {
			row.Child(sdk.Text(strings.Repeat(" ", pad)))
		}
	}
	appendGroup(row, right, s.sepStyle(), s.separatorText())
	return row
}

func appendGroup(row *sdk.Builder, segs []Segment, sepStyle, sepText string) {
	for i, seg := range segs {
		if i > 0 {
			row.Child(sdk.Text(sepText).Style(sepStyle))
		}
		row.Child(segmentBox(seg, ""))
	}
}

func segmentBox(seg Segment, fallbackStyle string) *sdk.Builder {
	box := sdk.Text(seg.Text)
	if seg.ID != "" {
		box.ID(seg.ID)
	}
	if seg.Style != "" {
		box.Style(seg.Style)
	} else if fallbackStyle != "" {
		box.Style(fallbackStyle)
	}
	if len(seg.Input) > 0 {
		box.Input(seg.Input...)
	}
	return box
}

func groupWidth(segs []Segment, sepW int) int {
	width := 0
	for i, seg := range segs {
		if i > 0 {
			width += sepW
		}
		width += sdk.DisplayWidth(seg.Text)
	}
	return width
}

// fitSegments trims segs to budget cells: trailing segments are dropped
// first, then the last remaining text is truncated.
func fitSegments(segs []Segment, budget, sepW int) ([]Segment, int) {
	if budget <= 0 {
		return nil, 0
	}
	out := append([]Segment(nil), segs...)
	for len(out) > 0 {
		width := groupWidth(out, sepW)
		if width <= budget {
			return out, width
		}
		if len(out) == 1 {
			text := sdk.Truncate(out[0].Text, budget)
			out[0].Text = text
			return out, sdk.DisplayWidth(text)
		}
		out = out[:len(out)-1]
	}
	return nil, 0
}

// FrameRow is one content row of a Frame: the text, its style (explicit
// style string or a host-internal token), an optional hit-test id and the
// input kinds that id accepts.
type FrameRow struct {
	Text  string
	Style string
	ID    string
	Input []string
}

// Frame builds a program-side bordered panel. The kernel has no border
// concept, so the widget draws the chrome itself as styled text rows: a top
// rule with the title, side bars on every row and a bottom rule. It is the
// shared primitive behind Card and the layout program's overlays/sidebar.
// Width/height <= 0 fall back to the row content size; boxes too small for
// chrome degrade to plain rows.
type Frame struct {
	ID        string
	Title     string
	Width     int
	Height    int
	Style     string
	Rows      []FrameRow
	FillStyle string
}

// Build returns the framed panel as a box tree.
func (f Frame) Build() *sdk.Builder {
	width, height := f.Width, f.Height
	if width <= 0 {
		width = frameTextWidth(f.Rows) + 2
	}
	if height <= 0 {
		height = len(f.Rows) + 2
	}
	style := f.Style
	if style == "" {
		style = DefaultBorderStyle
	}
	col := sdk.Box().Flow("col").ID(f.ID).Width(width).Height(height)
	if width < 3 || height < 2 {
		for _, row := range f.Rows {
			col.Child(frameRowBox(row, maxInt(1, width), ""))
		}
		return col
	}
	inner := width - 2
	col.Child(sdk.Text(ruleText(width, f.Title)).Style(style).Width(width).Height(1))
	for i := 0; i < height-2; i++ {
		row := FrameRow{Style: f.FillStyle}
		if i < len(f.Rows) {
			row = f.Rows[i]
		}
		line := sdk.Row().Height(1)
		line.Child(sdk.Text("│").Style(style).Width(1))
		line.Child(frameRowBox(row, inner, ""))
		line.Child(sdk.Text("│").Style(style).Width(1))
		col.Child(line)
	}
	col.Child(sdk.Text(bottomRuleText(width)).Style(style).Width(width).Height(1))
	return col
}

func frameRowBox(row FrameRow, width int, fallbackStyle string) *sdk.Builder {
	style := row.Style
	if style == "" {
		style = fallbackStyle
	}
	box := sdk.Text(padTo(row.Text, width)).Style(style).Width(width).Height(1)
	if row.ID != "" {
		box.ID(row.ID)
	}
	if len(row.Input) > 0 {
		box.Input(row.Input...)
	}
	return box
}

func frameTextWidth(rows []FrameRow) int {
	width := 0
	for _, row := range rows {
		if w := sdk.DisplayWidth(row.Text); w > width {
			width = w
		}
	}
	return width
}

// ruleText builds a top rule of exactly width display cells with an optional
// centered title, always leaving at least one horizontal glyph.
func ruleText(width int, title string) string {
	inner := width - 2
	if inner <= 0 {
		return "┌┐"
	}
	if title == "" {
		return "┌" + strings.Repeat("─", inner) + "┐"
	}
	label := " " + title + " "
	if budget := inner - 1; budget >= 0 {
		label = sdk.Truncate(label, budget)
	} else {
		label = ""
	}
	fill := inner - 1 - sdk.DisplayWidth(label)
	if fill < 0 {
		fill = 0
	}
	return "┌─" + label + strings.Repeat("─", fill) + "┐"
}

func bottomRuleText(width int) string {
	if width < 2 {
		return "└"
	}
	return "└" + strings.Repeat("─", width-2) + "┘"
}

func padTo(text string, width int) string {
	text = sdk.Truncate(text, width)
	if pad := width - sdk.DisplayWidth(text); pad > 0 {
		text += strings.Repeat(" ", pad)
	}
	return text
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Divider builds a one-cell rule used as a pane gutter: a vertical run of
// "│" cells (length rows) or a horizontal run of "─" cells (length cols).
// Clicks are resolved by the caller through ID/Input like any other box.
type Divider struct {
	ID       string
	Vertical bool
	Length   int
	Style    string
	Input    []string
}

// Build returns the divider box.
func (d Divider) Build() *sdk.Builder {
	glyph, width, height := "─", d.Length, 1
	if d.Vertical {
		glyph, width, height = "│", 1, d.Length
	}
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}
	style := d.Style
	if style == "" {
		style = DefaultSepStyle
	}
	box := sdk.Text(strings.Repeat(glyph, maxInt(1, d.Length))).
		ID(d.ID).Width(width).Height(height).Style(style)
	if len(d.Input) > 0 {
		box.Input(d.Input...)
	}
	return box
}

// Card builds a framed text box, optionally centering its lines in the
// declared geometry. It is the empty-slot / dialog building block; the
// border is drawn program-side by Frame, never by the kernel.
type Card struct {
	ID        string
	Title     string
	Lines     []string
	Width     int
	Height    int
	Style     string
	LineStyle string
	Center    bool
}

// Build returns the card box.
func (c Card) Build() *sdk.Builder {
	lines := append([]string(nil), c.Lines...)
	if c.Center {
		lines = centerLines(lines, c.Width, c.Height)
	}
	rows := make([]FrameRow, len(lines))
	for i, line := range lines {
		rows[i] = FrameRow{Text: line, Style: c.LineStyle}
	}
	frame := Frame{
		ID:     c.ID,
		Title:  c.Title,
		Width:  c.Width,
		Height: c.Height,
		Style:  c.Style,
		Rows:   rows,
	}
	return frame.Build()
}

func centerLines(lines []string, width, height int) []string {
	if width > 2 {
		inner := width - 2
		for i, line := range lines {
			pad := (inner - sdk.DisplayWidth(line)) / 2
			if pad > 0 {
				lines[i] = strings.Repeat(" ", pad) + line
			}
		}
	}
	if height > 2 {
		inner := height - 2
		if pad := (inner - len(lines)) / 2; pad > 0 {
			blank := make([]string, pad)
			lines = append(blank, lines...)
		}
	}
	return lines
}

// Button builds a clickable text box. "Clickable" is a protocol input flag
// plus the caller's hit handling; there is no framework button concept.
// A non-empty Hot character is rendered as its own accent-styled run so the
// caller can advertise and match the hotkey.
type Button struct {
	ID    string
	Text  string
	Hot   string
	Style string
	Input []string
}

// Line renders the button label as plain text for measurement.
func (b Button) Line() string { return b.Text }

// Build returns the button box (a row when a hotkey run is split out).
func (b Button) Build() *sdk.Builder {
	inputs := b.Input
	if len(inputs) == 0 {
		inputs = []string{"mouse"}
	}
	style := b.Style
	if style == "" {
		style = DefaultButtonStyle
	}
	if b.Hot == "" {
		return sdk.Text(b.Text).ID(b.ID).Style(style).Input(inputs...)
	}
	idx := strings.Index(b.Text, b.Hot)
	if idx < 0 {
		return sdk.Text(b.Text).ID(b.ID).Style(style).Input(inputs...)
	}
	row := sdk.Row().ID(b.ID).Input(inputs...)
	pre := b.Text[:idx]
	hot := b.Text[idx : idx+len(b.Hot)]
	post := b.Text[idx+len(b.Hot):]
	if pre != "" {
		row.Child(sdk.Text(pre).Style(style))
	}
	row.Child(sdk.Text(hot).Style(DefaultButtonStyle))
	if post != "" {
		row.Child(sdk.Text(post).Style(style))
	}
	return row
}

// KeyHint builds the left footer group: the current mode plus the bindings
// that work in it.
type KeyHint struct {
	Mode      string
	Keys      []string
	ModeStyle string
	KeyStyle  string
	SepStyle  string
	ID        string
}

// Text renders the hint as one plain string for measurement.
func (k KeyHint) Text() string {
	keys := strings.Join(k.Keys, " · ")
	if k.Mode == "" {
		return keys
	}
	if keys == "" {
		return k.Mode
	}
	return k.Mode + " " + DefaultSeparator + " " + keys
}

// Build returns the hint as a row of styled text boxes.
func (k KeyHint) Build() *sdk.Builder {
	row := sdk.Row().Height(1)
	if k.ID != "" {
		row.ID(k.ID)
	}
	modeStyle := k.ModeStyle
	if modeStyle == "" {
		modeStyle = DefaultStatusStyle
	}
	keyStyle := k.KeyStyle
	if keyStyle == "" {
		keyStyle = DefaultKeyStyle
	}
	sepStyle := k.SepStyle
	if sepStyle == "" {
		sepStyle = DefaultSepStyle
	}
	if k.Mode != "" {
		row.Child(sdk.Text(k.Mode).Style(modeStyle))
	}
	if len(k.Keys) > 0 {
		if k.Mode != "" {
			row.Child(sdk.Text(" " + DefaultSeparator + " ").Style(sepStyle))
		}
		row.Child(sdk.Text(strings.Join(k.Keys, " · ")).Style(keyStyle))
	}
	return row
}

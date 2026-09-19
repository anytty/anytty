package widgets

import (
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
)

// Rect is one solved rectangle in viewport cells.
type Rect struct {
	X, Y, W, H int
}

// Distribute splits avail cells over integer ratios (the old shell layout
// formula): proportional, at least one cell each, remainder on the last
// pane. It is the geometry primitive behind SplitLayout.
func Distribute(avail int, ratios []int) []int {
	out := make([]int, len(ratios))
	if len(ratios) == 0 {
		return out
	}
	if avail < len(ratios) {
		avail = len(ratios)
	}
	total := 0
	for _, ratio := range ratios {
		if ratio > 0 {
			total += ratio
		}
	}
	if total <= 0 {
		total = len(ratios)
		for i := range out {
			out[i] = 1
		}
	} else {
		for i, ratio := range ratios {
			out[i] = avail * ratio / total
			if out[i] < 1 {
				out[i] = 1
			}
		}
	}
	used := 0
	for _, value := range out {
		used += value
	}
	out[len(out)-1] += avail - used
	if out[len(out)-1] < 1 {
		out[len(out)-1] = 1
	}
	return out
}

// SplitLayout lays panes along one axis with a fixed gap; each visible gap is
// a draggable Divider cell the program hit-tests like any other box.
//
//	layout := widgets.SplitLayout{Orient: "row", Weights: []int{1, 2}, Gap: 1}
//	panes, dividers := layout.Rects(width, height)
type SplitLayout struct {
	// Orient is "row" (left-to-right) or "col" (top-to-bottom).
	Orient string
	// Weights are the relative pane sizes; empty means one pane.
	Weights []int
	// Gap is the separator cells between panes (default 1).
	Gap int
}

// Axis returns the resolved orientation ("row" unless "col" is explicit).
func (s SplitLayout) Axis() string {
	if s.Orient == "col" {
		return "col"
	}
	return "row"
}

// GapWidth returns the separator cells between two panes.
func (s SplitLayout) GapWidth() int {
	if s.Gap > 0 {
		return s.Gap
	}
	return 1
}

// Rects solves the layout inside a width x height box: one rect per pane and
// one divider rect per gap. Pane count comes from Weights (or one pane).
func (s SplitLayout) Rects(width, height int) ([]Rect, []Rect) {
	weights := s.Weights
	if len(weights) == 0 {
		weights = []int{1}
	}
	count := len(weights)
	gap := s.GapWidth()
	if s.Axis() == "col" {
		avail := height - (count-1)*gap
		sizes := Distribute(avail, weights)
		panes := make([]Rect, 0, count)
		dividers := make([]Rect, 0, count-1)
		y := 0
		for i, size := range sizes {
			panes = append(panes, Rect{X: 0, Y: y, W: width, H: size})
			y += size
			if i < count-1 {
				dividers = append(dividers, Rect{X: 0, Y: y, W: width, H: gap})
				y += gap
			}
		}
		return panes, dividers
	}
	avail := width - (count-1)*gap
	sizes := Distribute(avail, weights)
	panes := make([]Rect, 0, count)
	dividers := make([]Rect, 0, count-1)
	x := 0
	for i, size := range sizes {
		panes = append(panes, Rect{X: x, Y: 0, W: size, H: height})
		x += size
		if i < count-1 {
			dividers = append(dividers, Rect{X: x, Y: 0, W: gap, H: height})
			x += gap
		}
	}
	return panes, dividers
}

// TitleBar builds a one-row title strip: left segments (title, status) and
// right-aligned buttons. With Width > 0 the buttons are pushed to the right
// edge and the left group is truncated.
type TitleBar struct {
	ID      string
	Left    []Segment
	Buttons []Button
	Width   int
	Fill    string
}

// Line renders the bar as plain text for measurement.
func (t TitleBar) Line() string {
	left := segmentsText(t.Left)
	buttons := make([]string, 0, len(t.Buttons))
	for _, button := range t.Buttons {
		buttons = append(buttons, button.Line())
	}
	right := strings.Join(buttons, " ")
	if t.Width > 0 {
		pad := t.Width - sdk.DisplayWidth(left) - sdk.DisplayWidth(right)
		if pad > 0 {
			return left + strings.Repeat(" ", pad) + right
		}
	}
	return left + right
}

// Build returns the bar as a one-row box tree.
func (t TitleBar) Build() *sdk.Builder {
	row := sdk.Row().Height(1)
	if t.ID != "" {
		row.ID(t.ID)
	}
	left := t.Left
	buttons := make([]string, 0, len(t.Buttons))
	for _, button := range t.Buttons {
		buttons = append(buttons, button.Line())
	}
	right := strings.Join(buttons, " ")
	if t.Width > 0 && sdk.DisplayWidth(right) < t.Width {
		left, _ = fitSegments(left, t.Width-sdk.DisplayWidth(right), sdk.DisplayWidth(DefaultSeparator))
	}
	appendGroup(row, left, t.Fill, DefaultSeparator)
	if t.Width > 0 {
		if pad := t.Width - segmentsWidth(left) - sdk.DisplayWidth(right); pad > 0 {
			row.Child(sdk.Text(strings.Repeat(" ", pad)))
		}
	}
	for i, button := range t.Buttons {
		if i > 0 {
			row.Child(sdk.Text(" "))
		}
		row.Child(button.Build())
	}
	return row
}

func segmentsText(segments []Segment) string {
	texts := make([]string, 0, len(segments))
	for _, segment := range segments {
		texts = append(texts, segment.Text)
	}
	return strings.Join(texts, "")
}

func segmentsWidth(segments []Segment) int {
	width := 0
	for _, segment := range segments {
		width += sdk.DisplayWidth(segment.Text)
	}
	return width
}

// Footer builds the bottom bar: an optional scene badge plus key groups on
// the left and right-aligned summary segments. Width > 0 right-aligns the
// summary and truncates the left groups.
type Footer struct {
	ID        string
	Badge     Segment
	HasBadge  bool
	Groups    []Segment
	Right     []Segment
	Width     int
	Separator string
	SepStyle  string
}

func (f Footer) separatorText() string {
	if f.Separator != "" {
		return " " + f.Separator + " "
	}
	return " " + DefaultSeparator + " "
}

func (f Footer) sepStyle() string {
	if f.SepStyle != "" {
		return f.SepStyle
	}
	return DefaultSepStyle
}

func (f Footer) leftSegments() []Segment {
	left := make([]Segment, 0, len(f.Groups)+1)
	if f.HasBadge {
		left = append(left, f.Badge)
	}
	left = append(left, f.Groups...)
	return left
}

// Line renders the whole bar as plain text at Width cells.
func (f Footer) Line() string {
	sep := f.separatorText()
	left, right := f.leftSegments(), f.Right
	if f.Width > 0 {
		sepW := sdk.DisplayWidth(sep)
		right, _ = fitSegments(right, f.Width, sepW)
		budget := f.Width - groupWidth(right, sepW)
		if len(left) > 0 && len(right) > 0 {
			budget -= sepW
		}
		left, _ = fitSegments(left, budget, sepW)
	}
	leftText := strings.Join(segmentTexts(left), sep)
	rightText := strings.Join(segmentTexts(right), sep)
	if f.Width <= 0 {
		if leftText != "" && rightText != "" {
			return leftText + sep + rightText
		}
		return leftText + rightText
	}
	pad := f.Width - sdk.DisplayWidth(leftText) - sdk.DisplayWidth(rightText)
	if pad < 0 {
		pad = 0
	}
	return leftText + strings.Repeat(" ", pad) + rightText
}

// Build returns the footer as a one-row box tree.
func (f Footer) Build() *sdk.Builder {
	row := sdk.Row().Height(1)
	if f.ID != "" {
		row.ID(f.ID)
	}
	sep := f.separatorText()
	left := f.leftSegments()
	right := f.Right
	if f.Width > 0 {
		sepW := sdk.DisplayWidth(sep)
		right, _ = fitSegments(right, f.Width, sepW)
		leftBudget := f.Width - groupWidth(right, sepW)
		if len(left) > 0 && len(right) > 0 {
			leftBudget -= sepW
		}
		left, _ = fitSegments(left, leftBudget, sepW)
	}
	used := groupWidth(left, sdk.DisplayWidth(sep)) + groupWidth(right, sdk.DisplayWidth(sep))
	if len(left) > 0 && len(right) > 0 {
		used += sdk.DisplayWidth(sep)
	}
	appendGroup(row, left, f.sepStyle(), sep)
	if len(left) > 0 && len(right) > 0 {
		row.Child(sdk.Text(sep).Style(f.sepStyle()))
	}
	if f.Width > used {
		row.Child(sdk.Text(strings.Repeat(" ", f.Width-used)))
	}
	appendGroup(row, right, f.sepStyle(), sep)
	return row
}

func segmentTexts(segments []Segment) []string {
	texts := make([]string, 0, len(segments))
	for _, segment := range segments {
		texts = append(texts, segment.Text)
	}
	return texts
}

// PickerRow is one selectable row of a Picker.
type PickerRow struct {
	Text       string
	ID         string
	Style      string
	Selected   bool
	Selectable bool
}

// Picker builds a framed selectable list (the program-side picker overlay).
// Row ids are hit-test targets; the program moves the selection itself.
type Picker struct {
	ID            string
	Title         string
	Width         int
	Height        int
	Rows          []PickerRow
	Style         string
	Marker        string
	SelectedStyle string
}

// Build returns the picker as a framed box tree.
func (p Picker) Build() *sdk.Builder {
	marker := p.Marker
	if marker == "" {
		marker = "▸ "
	}
	rows := make([]FrameRow, 0, len(p.Rows))
	for _, row := range p.Rows {
		text := "  " + row.Text
		style := row.Style
		if row.Selected {
			text = marker + row.Text
			if p.SelectedStyle != "" {
				style = p.SelectedStyle
			}
		}
		frameRow := FrameRow{Text: text, Style: style}
		if row.ID != "" {
			frameRow.ID = row.ID
			frameRow.Input = []string{"mouse"}
		}
		rows = append(rows, frameRow)
	}
	width := p.Width
	if width <= 0 {
		width = 40
	}
	height := p.Height
	if height <= 0 {
		height = len(rows) + 2
	}
	return Frame{ID: p.ID, Title: p.Title, Width: width, Height: height, Style: p.Style, Rows: rows}.Build()
}

// FloatingLayer is a program-side floating window: a Frame placed with Pos so
// it composites above the regular flow. Collapsed keeps only the title row.
type FloatingLayer struct {
	ID        string
	Title     string
	X, Y      int
	Width     int
	Height    int
	Style     string
	Rows      []FrameRow
	Collapsed bool
}

// Build returns the floating window as a positioned box tree.
func (f FloatingLayer) Build() *sdk.Builder {
	height := f.Height
	if f.Collapsed {
		height = 1
	}
	frame := Frame{ID: f.ID, Title: f.Title, Width: f.Width, Height: height, Style: f.Style, Rows: f.Rows}
	return frame.Build().Pos(f.X, f.Y)
}

// Toast is a transient one-line notice (program-side): a styled text box the
// program hides by dropping it from the next view.
type Toast struct {
	ID    string
	Text  string
	Style string
}

// Build returns the toast as a text box.
func (t Toast) Build() *sdk.Builder {
	box := sdk.Text(t.Text)
	if t.ID != "" {
		box.ID(t.ID)
	}
	if t.Style != "" {
		box.Style(t.Style)
	}
	return box
}

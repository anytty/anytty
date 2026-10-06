package widgets

import (
	"math"
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
)

// This file adds the pure text-chart widgets: Sparkline, BarChart, Heatmap,
// Meter (and its Gauge alias) and Legend. Like every widget here they hold no
// state, schedule no timer and emit no protocol method: Build returns a plain
// SDK box tree the caller commits. All geometry is cell-based and every glyph
// is display width 1, so the host layout and the widget agree without a width
// table.

// chartSparkGlyphs is the shared eighth-block ramp, low to high. Index 7 is
// the full block, used for whole cells; indices 0..6 are the partial cells.
var chartSparkGlyphs = []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

// chartDefaultBarWidth is the horizontal bar length when BarChart.Width is
// unset.
const chartDefaultBarWidth = 20

// chartClampInt folds v into [lo, hi].
func chartClampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// chartClampFloat folds v into [lo, hi].
func chartClampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ---------------------------------------------------------------------------
// Sparkline

// Sparkline renders a series as one row of eighth-block glyphs. Min/Max are
// optional bounds; when nil they are derived from Values. Empty input renders
// an empty line; an all-equal (or single-value) series renders the mid-ramp
// glyph, since there is no range to spread.
type Sparkline struct {
	Values []float64
	Width  int
	Style  string
	Min    *float64
	Max    *float64
}

// Bounds returns the effective low/high values, honouring explicit Min/Max and
// deriving the rest from Values. Non-finite samples are ignored; an empty or
// all-NaN series yields (0, 0).
func (s Sparkline) Bounds() (float64, float64) {
	min, max := math.Inf(1), math.Inf(-1)
	for _, v := range s.Values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if math.IsInf(min, 1) {
		min, max = 0, 0
	}
	if s.Min != nil {
		min = *s.Min
	}
	if s.Max != nil {
		max = *s.Max
	}
	return min, max
}

// Level returns the eighth-block index [0, 7] for v under the effective
// bounds. min == max (including all-equal and one-value input) maps every
// sample to the mid-ramp glyph; NaN and out-of-range samples clamp.
func (s Sparkline) Level(v float64) int {
	min, max := s.Bounds()
	if max <= min {
		return (len(chartSparkGlyphs) - 1) / 2
	}
	if math.IsNaN(v) {
		return 0
	}
	t := chartClampFloat((v-min)/(max-min), 0, 1)
	level := int(t*float64(len(chartSparkGlyphs)-1) + 0.5)
	return chartClampInt(level, 0, len(chartSparkGlyphs)-1)
}

// samples returns the values to render: all of them, or Width buckets of
// averaged samples when Width is smaller than the series.
func (s Sparkline) samples() []float64 {
	if s.Width <= 0 || len(s.Values) <= s.Width {
		return s.Values
	}
	out := make([]float64, s.Width)
	n := len(s.Values)
	for i := 0; i < s.Width; i++ {
		start := i * n / s.Width
		end := (i + 1) * n / s.Width
		if end <= start {
			end = start + 1
		}
		sum := 0.0
		for _, v := range s.Values[start:end] {
			sum += v
		}
		out[i] = sum / float64(end-start)
	}
	return out
}

// Line returns the sparkline as plain text, or "" for empty input.
func (s Sparkline) Line() string {
	samples := s.samples()
	if len(samples) == 0 {
		return ""
	}
	var b strings.Builder
	for _, v := range samples {
		b.WriteString(chartSparkGlyphs[s.Level(v)])
	}
	return b.String()
}

// Build returns the sparkline as a one-row text box.
func (s Sparkline) Build() *sdk.Builder {
	box := sdk.Text(s.Line()).Height(1)
	if s.Style != "" {
		box.Style(s.Style)
	}
	return box
}

// ---------------------------------------------------------------------------
// BarChart

// BarChart renders values as vertical sub-cell columns (eighth blocks) or as
// horizontal full-block runs. Max <= 0 auto-scales to the largest value.
// Vertical bars occupy Height rows (default 1) and share Width cells; when
// Width cannot fit every bar, trailing bars are dropped. Horizontal bars use
// each value's row, optionally prefixed by its Labels entry; Height caps the
// number of rows. Selected >= 0 styles that bar with SelectedStyle.
type BarChart struct {
	Values        []float64
	Labels        []string
	Width         int
	Height        int
	Max           float64
	Style         string
	LabelStyle    string
	SelectedStyle string
	Selected      int
	Horizontal    bool
}

// MaxValue returns Max when positive, else the largest finite value (at least
// 1 when the series is empty or non-positive).
func (b BarChart) MaxValue() float64 {
	if b.Max > 0 {
		return b.Max
	}
	max := 0.0
	for _, v := range b.Values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		if v > max {
			max = v
		}
	}
	if max <= 0 {
		return 1
	}
	return max
}

func (b BarChart) barHeight() int {
	if b.Height > 0 {
		return b.Height
	}
	return 1
}

// verticalValues returns the values a vertical chart can fit in Width cells.
// Bars are one cell wide with a single gap, so a width of w fits (w+1)/2
// bars.
func (b BarChart) verticalValues() []float64 {
	values := b.Values
	if b.Width <= 0 {
		return values
	}
	maxBars := (b.Width + 1) / 2
	if maxBars > len(values) {
		maxBars = len(values)
	}
	if maxBars < 0 {
		maxBars = 0
	}
	return values[:maxBars]
}

// horizontalValues returns the values a horizontal chart renders, capped by
// Height when set.
func (b BarChart) horizontalValues() []float64 {
	values := b.Values
	if b.Height > 0 && len(values) > b.Height {
		return values[:b.Height]
	}
	return values
}

// columnWidth returns the cell width of each vertical bar: Width split evenly,
// always at least 1.
func (b BarChart) columnWidth(count int) int {
	if count <= 0 {
		return 1
	}
	if b.Width <= 0 {
		return 1
	}
	width := (b.Width - (count - 1)) / count
	if width < 1 {
		return 1
	}
	return width
}

// labelWidth returns the widest label in display cells, or 0 without labels.
func (b BarChart) labelWidth() int {
	width := 0
	for _, label := range b.Labels {
		if w := sdk.DisplayWidth(label); w > width {
			width = w
		}
	}
	return width
}

// horizontalBarWidth returns the run length for a horizontal bar, leaving room
// for the labels and the gap.
func (b BarChart) horizontalBarWidth(labelWidth int) int {
	if b.Width <= 0 {
		return chartDefaultBarWidth
	}
	width := b.Width
	if labelWidth > 0 {
		width -= labelWidth + 1
	}
	if width < 1 {
		return 1
	}
	return width
}

func (b BarChart) barStyle(index int) string {
	if index == b.Selected && b.SelectedStyle != "" {
		return b.SelectedStyle
	}
	return b.Style
}

func (b BarChart) barUnits(v float64, max float64, height int) int {
	if height <= 0 || max <= 0 || math.IsNaN(v) || v <= 0 {
		return 0
	}
	t := chartClampFloat(v/max, 0, 1)
	units := int(t*float64(height*8) + 0.5)
	return chartClampInt(units, 0, height*8)
}

// barCell renders one vertical bar cell for the k-th row counted from the
// bottom: a full run, a single partial eighth glyph, or blanks.
func barCell(units, k, width int) string {
	if width < 1 {
		width = 1
	}
	if units >= (k+1)*8 {
		return strings.Repeat("█", width)
	}
	if units > k*8 {
		level := units - k*8
		return chartSparkGlyphs[level-1] + strings.Repeat(" ", width-1)
	}
	return strings.Repeat(" ", width)
}

// verticalCells returns the bar grid as [row][bar] cells, top row first.
func (b BarChart) verticalCells() [][]string {
	values := b.verticalValues()
	count := len(values)
	if count == 0 {
		return nil
	}
	height := b.barHeight()
	width := b.columnWidth(count)
	max := b.MaxValue()
	units := make([]int, count)
	for i, v := range values {
		units[i] = b.barUnits(v, max, height)
	}
	cells := make([][]string, height)
	for row := 0; row < height; row++ {
		k := height - 1 - row
		cells[row] = make([]string, count)
		for i := 0; i < count; i++ {
			cells[row][i] = barCell(units[i], k, width)
		}
	}
	return cells
}

// VerticalLines returns the vertical chart as plain text rows, including the
// label row when Labels is set. It returns nil for empty input.
func (b BarChart) VerticalLines() []string {
	cells := b.verticalCells()
	if cells == nil {
		return nil
	}
	width := b.columnWidth(len(cells[0]))
	lines := make([]string, 0, len(cells)+1)
	for _, row := range cells {
		lines = append(lines, strings.Join(row, " "))
	}
	if len(b.Labels) > 0 {
		count := len(cells[0])
		labels := make([]string, count)
		for i := 0; i < count && i < len(b.Labels); i++ {
			labels[i] = padTo(b.Labels[i], width)
		}
		for i := len(b.Labels); i < count; i++ {
			labels[i] = strings.Repeat(" ", width)
		}
		lines = append(lines, strings.Join(labels, " "))
	}
	return lines
}

// HorizontalLines returns the horizontal chart as plain text rows.
func (b BarChart) HorizontalLines() []string {
	values := b.horizontalValues()
	if len(values) == 0 {
		return nil
	}
	labelWidth := b.labelWidth()
	barWidth := b.horizontalBarWidth(labelWidth)
	max := b.MaxValue()
	lines := make([]string, len(values))
	for i, v := range values {
		prefix := ""
		if labelWidth > 0 {
			label := ""
			if i < len(b.Labels) {
				label = b.Labels[i]
			}
			prefix = padTo(label, labelWidth) + " "
		}
		lines[i] = prefix + strings.Repeat("█", b.runLength(v, max, barWidth))
	}
	return lines
}

func (b BarChart) runLength(v, max float64, width int) int {
	if math.IsNaN(v) || v <= 0 || max <= 0 {
		return 0
	}
	length := int(chartClampFloat(v/max, 0, 1)*float64(width) + 0.5)
	return chartClampInt(length, 0, width)
}

// Build returns the chart as a column of styled rows.
func (b BarChart) Build() *sdk.Builder {
	if b.Horizontal {
		return b.buildHorizontal()
	}
	return b.buildVertical()
}

func (b BarChart) buildVertical() *sdk.Builder {
	col := sdk.Box().Flow("col")
	values := b.verticalValues()
	if len(values) == 0 {
		return col
	}
	height := b.barHeight()
	width := b.columnWidth(len(values))
	max := b.MaxValue()
	units := make([]int, len(values))
	for i, v := range values {
		units[i] = b.barUnits(v, max, height)
	}
	for row := 0; row < height; row++ {
		k := height - 1 - row
		line := sdk.Row().Height(1)
		for i := 0; i < len(values); i++ {
			if i > 0 {
				line.Child(sdk.Text(" "))
			}
			line.Child(sdk.Text(barCell(units[i], k, width)).Style(b.barStyle(i)))
		}
		col.Child(line)
	}
	if len(b.Labels) > 0 {
		line := sdk.Row().Height(1)
		for i := 0; i < len(values); i++ {
			if i > 0 {
				line.Child(sdk.Text(" "))
			}
			label := ""
			if i < len(b.Labels) {
				label = b.Labels[i]
			}
			line.Child(sdk.Text(padTo(label, width)).Style(b.LabelStyle))
		}
		col.Child(line)
	}
	return col
}

func (b BarChart) buildHorizontal() *sdk.Builder {
	col := sdk.Box().Flow("col")
	values := b.horizontalValues()
	if len(values) == 0 {
		return col
	}
	labelWidth := b.labelWidth()
	barWidth := b.horizontalBarWidth(labelWidth)
	max := b.MaxValue()
	for i, v := range values {
		line := sdk.Row().Height(1)
		if labelWidth > 0 {
			label := ""
			if i < len(b.Labels) {
				label = b.Labels[i]
			}
			line.Child(sdk.Text(padTo(label, labelWidth) + " ").Style(b.LabelStyle))
		}
		if length := b.runLength(v, max, barWidth); length > 0 {
			line.Child(sdk.Text(strings.Repeat("█", length)).Style(b.barStyle(i)))
		}
		col.Child(line)
	}
	return col
}

// ---------------------------------------------------------------------------
// Heatmap

// DefaultHeatShades is the default low-to-high shade ramp: blank, middle dot,
// then the light/medium/dark/full blocks.
const DefaultHeatShades = " ·░▒▓█"

// Heatmap renders a matrix as shaded cells. Values may be ragged: every row is
// padded to Cols() and missing cells render as the lowest shade. Min/Max are
// the value range; when Min >= Max the range is derived from the data. Shades
// overrides the default ramp (low to high); RowLabels/ColLabels are optional
// and add a labelled gutter and header.
type Heatmap struct {
	Values     [][]float64
	RowLabels  []string
	ColLabels  []string
	Min        float64
	Max        float64
	Style      string
	LabelStyle string
	Shades     []string
}

// shadeList returns the active ramp, defaulting to DefaultHeatShades.
func (h Heatmap) shadeList() []string {
	if len(h.Shades) > 0 {
		return h.Shades
	}
	shades := make([]string, 0, 8)
	for _, r := range DefaultHeatShades {
		shades = append(shades, string(r))
	}
	return shades
}

// Rows returns the number of value rows.
func (h Heatmap) Rows() int { return len(h.Values) }

// Cols returns the matrix width: the widest row, raised to len(ColLabels).
func (h Heatmap) Cols() int {
	cols := len(h.ColLabels)
	for _, row := range h.Values {
		if len(row) > cols {
			cols = len(row)
		}
	}
	return cols
}

// Bounds returns the effective value range. An explicit Min < Max wins;
// otherwise the range is derived from the finite samples. All-equal data (and
// empty data) collapses to a zero-width range, which Level maps to the
// mid-ramp.
func (h Heatmap) Bounds() (float64, float64) {
	if h.Min < h.Max {
		return h.Min, h.Max
	}
	min, max := math.Inf(1), math.Inf(-1)
	for _, row := range h.Values {
		for _, v := range row {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
	}
	if math.IsInf(min, 1) {
		return 0, 0
	}
	return min, max
}

// Level returns the shade index [0, len(Shades)-1] for v. A zero-width range
// maps every sample to the mid-ramp; NaN clamps to the lowest shade.
func (h Heatmap) Level(v float64) int {
	shades := h.shadeList()
	count := len(shades)
	if count <= 0 {
		return 0
	}
	min, max := h.Bounds()
	if max <= min {
		return count / 2
	}
	if math.IsNaN(v) {
		return 0
	}
	t := chartClampFloat((v-min)/(max-min), 0, 1)
	return chartClampInt(int(t*float64(count-1)+0.5), 0, count-1)
}

// Cell returns the plain shade text of one matrix cell. Out-of-range columns
// and missing (short) rows return the lowest shade, so the grid is always
// rectangular.
func (h Heatmap) Cell(row, col int) string {
	shades := h.shadeList()
	if len(shades) == 0 {
		return ""
	}
	if row < 0 || row >= len(h.Values) || col < 0 || col >= len(h.Values[row]) {
		return shades[0]
	}
	return shades[h.Level(h.Values[row][col])]
}

// Grid returns the heatmap as plain text rows: an optional column-label header,
// then one row per value row with its optional row label.
func (h Heatmap) Grid() []string {
	rows, cols := h.Rows(), h.Cols()
	if rows == 0 || cols == 0 {
		return nil
	}
	shades := h.shadeList()
	if len(shades) == 0 {
		return nil
	}
	rowLabelWidth := h.rowLabelWidth()
	colWidth := h.colWidth()
	gutter := ""
	if rowLabelWidth > 0 {
		gutter = " "
	}
	blank := strings.Repeat(" ", rowLabelWidth+len(gutter))
	lines := make([]string, 0, rows+1)
	if len(h.ColLabels) > 0 {
		var b strings.Builder
		b.WriteString(blank)
		for c := 0; c < cols; c++ {
			label := ""
			if c < len(h.ColLabels) {
				label = h.ColLabels[c]
			}
			b.WriteString(padTo(label, colWidth))
		}
		lines = append(lines, b.String())
	}
	for r := 0; r < rows; r++ {
		var b strings.Builder
		if rowLabelWidth > 0 {
			label := ""
			if r < len(h.RowLabels) {
				label = h.RowLabels[r]
			}
			b.WriteString(padTo(label, rowLabelWidth))
			b.WriteString(gutter)
		}
		for c := 0; c < cols; c++ {
			b.WriteString(padTo(h.Cell(r, c), colWidth))
		}
		lines = append(lines, b.String())
	}
	return lines
}

func (h Heatmap) rowLabelWidth() int {
	width := 0
	for _, label := range h.RowLabels {
		if w := sdk.DisplayWidth(label); w > width {
			width = w
		}
	}
	return width
}

func (h Heatmap) colWidth() int {
	width := 1
	for _, label := range h.ColLabels {
		if w := sdk.DisplayWidth(label); w > width {
			width = w
		}
	}
	return width
}

// Build returns the heatmap as a column of styled one-row boxes.
func (h Heatmap) Build() *sdk.Builder {
	col := sdk.Box().Flow("col")
	lines := h.Grid()
	if len(lines) == 0 {
		return col
	}
	col.Child(sdk.Text(lines[0]).Style(firstNonEmpty(h.LabelStyle, h.Style)).Height(1))
	for i := 1; i < len(lines); i++ {
		col.Child(sdk.Text(lines[i]).Style(h.Style).Height(1))
	}
	return col
}

// ---------------------------------------------------------------------------
// Meter / Gauge

// Meter is a labelled bar for a float value, reusing the ProgressBar glyphs
// and semantics without depending on its integer fields. Gauge is an alias.
type Meter struct {
	ID         string
	Value      float64
	Max        float64
	Width      int
	Label      string
	ShowValue  bool
	Style      string
	TrackStyle string
	LabelStyle string
	ValueStyle string
}

// Gauge is the alias of Meter.
type Gauge = Meter

// Fraction returns Value/Max clamped to [0, 1]; a non-positive Max (or NaN
// Value) is 0.
func (m Meter) Fraction() float64 {
	if m.Max <= 0 || math.IsNaN(m.Value) {
		return 0
	}
	return chartClampFloat(m.Value/m.Max, 0, 1)
}

func (m Meter) barWidth() int {
	if m.Width > 0 {
		return m.Width
	}
	return DefaultProgressWidth
}

// BarText returns the full bar string (filled + empty glyphs).
func (m Meter) BarText() string {
	width := m.barWidth()
	filled := chartClampInt(int(m.Fraction()*float64(width)+0.5), 0, width)
	return strings.Repeat(DefaultProgressFull, filled) + strings.Repeat(DefaultProgressEmpty, width-filled)
}

// ValueText returns the optional value readout ("3/10"), or "".
func (m Meter) ValueText() string {
	if !m.ShowValue {
		return ""
	}
	text := FormatFloat(m.Value, 0, 0)
	if m.Max > 0 {
		text += "/" + FormatFloat(m.Max, 0, 0)
	}
	return text
}

// Build returns the meter as one row: label, fill, track and value.
func (m Meter) Build() *sdk.Builder {
	row := sdk.Row().Height(1)
	if m.ID != "" {
		row.ID(m.ID)
	}
	if m.Label != "" {
		row.Child(sdk.Text(m.Label).Style(m.LabelStyle).Height(1))
	}
	width := m.barWidth()
	filled := chartClampInt(int(m.Fraction()*float64(width)+0.5), 0, width)
	if filled > 0 {
		row.Child(sdk.Text(strings.Repeat(DefaultProgressFull, filled)).Style(m.Style).Height(1))
	}
	if empty := width - filled; empty > 0 {
		row.Child(sdk.Text(strings.Repeat(DefaultProgressEmpty, empty)).Style(m.TrackStyle).Height(1))
	}
	if text := m.ValueText(); text != "" {
		row.Child(sdk.Text(text).Style(m.ValueStyle).Height(1))
	}
	return row
}

// ---------------------------------------------------------------------------
// Legend

// LegendItem is one legend entry: a marker coloured with Color, followed by
// its Label styled with the Legend style.
type LegendItem struct {
	Label  string
	Color  string
	Marker string
}

// DefaultLegendMarker is the marker glyph used when LegendItem.Marker is
// empty.
const DefaultLegendMarker = "■"

// Legend renders a one-row key of coloured markers and labels.
type Legend struct {
	Items     []LegendItem
	Style     string
	Separator string
}

// Text returns the legend as plain text for measurement.
func (l Legend) Text() string {
	separator := firstNonEmpty(l.Separator, " ")
	var b strings.Builder
	for i, item := range l.Items {
		if i > 0 {
			b.WriteString(separator)
		}
		b.WriteString(firstNonEmpty(item.Marker, DefaultLegendMarker))
		if item.Label != "" {
			b.WriteString(" " + item.Label)
		}
	}
	return b.String()
}

// Build returns the legend as a one-row box tree.
func (l Legend) Build() *sdk.Builder {
	row := sdk.Row().Height(1)
	separator := firstNonEmpty(l.Separator, " ")
	for i, item := range l.Items {
		if i > 0 {
			row.Child(sdk.Text(separator).Style(l.Style))
		}
		marker := sdk.Text(firstNonEmpty(item.Marker, DefaultLegendMarker))
		if item.Color != "" {
			marker.Style(item.Color)
		}
		row.Child(marker)
		if item.Label != "" {
			row.Child(sdk.Text(" " + item.Label).Style(l.Style))
		}
	}
	return row
}

package widgets

import (
	"strconv"
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
)

// Progress bar glyphs and geometry defaults.
const (
	DefaultProgressFull  = "█"
	DefaultProgressEmpty = "░"
	DefaultProgressWidth = 20
)

// spinnerFrames is the default braille spinner cycle; Spinner.Frames
// overrides it per widget.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ProgressBar renders a fixed-width bar from Value/Max plus an optional label
// and percentage. It holds no timer; the caller advances Value and commits.
type ProgressBar struct {
	ID           string
	Value        int
	Max          int
	Width        int
	Label        string
	HidePercent  bool
	Full         string
	Empty        string
	Style        string
	TrackStyle   string
	LabelStyle   string
	PercentStyle string
}

// Fraction returns Value/Max clamped to [0, 1]; a non-positive Max is 0.
func (p ProgressBar) Fraction() float64 {
	if p.Max <= 0 {
		return 0
	}
	value := p.Value
	if value < 0 {
		value = 0
	}
	if value > p.Max {
		value = p.Max
	}
	return float64(value) / float64(p.Max)
}

// Percent returns the rounded percentage in [0, 100].
func (p ProgressBar) Percent() int {
	return int(p.Fraction()*100 + 0.5)
}

// BarText returns the full bar string (filled + empty glyphs).
func (p ProgressBar) BarText() string {
	width := p.barWidth()
	filled := int(p.Fraction()*float64(width) + 0.5)
	if filled > width {
		filled = width
	}
	return strings.Repeat(p.fullGlyph(), filled) + strings.Repeat(p.emptyGlyph(), width-filled)
}

// Build returns the bar as a one-row box tree: label, filled run, empty run
// and percentage.
func (p ProgressBar) Build() *sdk.Builder {
	row := sdk.Row().Height(1)
	if p.ID != "" {
		row.ID(p.ID)
	}
	if p.Label != "" {
		row.Child(sdk.Text(p.Label).Style(p.LabelStyle).Height(1))
	}
	width := p.barWidth()
	filled := int(p.Fraction()*float64(width) + 0.5)
	if filled > width {
		filled = width
	}
	if filled > 0 {
		row.Child(sdk.Text(strings.Repeat(p.fullGlyph(), filled)).Style(p.Style).Height(1))
	}
	if empty := width - filled; empty > 0 {
		row.Child(sdk.Text(strings.Repeat(p.emptyGlyph(), empty)).Style(p.TrackStyle).Height(1))
	}
	if !p.HidePercent {
		row.Child(sdk.Text(strconv.Itoa(p.Percent()) + "%").Style(p.PercentStyle).Height(1))
	}
	return row
}

func (p ProgressBar) barWidth() int {
	if p.Width > 0 {
		return p.Width
	}
	return DefaultProgressWidth
}

func (p ProgressBar) fullGlyph() string {
	if p.Full != "" {
		return p.Full
	}
	return DefaultProgressFull
}

func (p ProgressBar) emptyGlyph() string {
	if p.Empty != "" {
		return p.Empty
	}
	return DefaultProgressEmpty
}

// Spinner renders one frame of an animation. The widget has no timer: the
// caller advances Frame (e.g. from its own tick) and commits the view.
type Spinner struct {
	ID         string
	Frame      int
	Frames     []string
	Style      string
	Label      string
	LabelStyle string
}

// FrameText returns the current frame; Frame is wrapped for negative values.
func (s Spinner) FrameText() string {
	frames := s.Frames
	if len(frames) == 0 {
		frames = spinnerFrames
	}
	if len(frames) == 0 {
		return ""
	}
	index := ((s.Frame % len(frames)) + len(frames)) % len(frames)
	return frames[index]
}

// Build returns the spinner and its optional label as one row.
func (s Spinner) Build() *sdk.Builder {
	row := sdk.Row().Height(1)
	if s.ID != "" {
		row.ID(s.ID)
	}
	row.Child(sdk.Text(s.FrameText()).Style(s.Style).Height(1))
	if s.Label != "" {
		row.Child(sdk.Text(s.Label).Style(s.LabelStyle).Height(1))
	}
	return row
}

// Badge is a small styled label (status chip, count, key marker).
type Badge struct {
	ID    string
	Text  string
	Style string
	Input []string
}

// Build returns the badge as one text box.
func (b Badge) Build() *sdk.Builder {
	box := sdk.Text(b.Text).Height(1)
	if b.ID != "" {
		box.ID(b.ID)
	}
	if b.Style != "" {
		box.Style(b.Style)
	}
	if len(b.Input) > 0 {
		box.Input(b.Input...)
	}
	return box
}

// Tag is one item of Tags.
type Tag struct {
	Text  string
	ID    string
	Style string
	Input []string
}

// Tags renders a row of tagged labels separated by Separator.
type Tags struct {
	ID        string
	Items     []Tag
	Separator string
	SepStyle  string
	Style     string
}

// Build returns the tags as a one-row box tree.
func (t Tags) Build() *sdk.Builder {
	row := sdk.Row().Height(1)
	if t.ID != "" {
		row.ID(t.ID)
	}
	separator := firstNonEmpty(t.Separator, DefaultTableSeparator)
	for i, tag := range t.Items {
		if i > 0 {
			row.Child(sdk.Text(separator).Style(t.SepStyle))
		}
		box := sdk.Text(tag.Text).Height(1)
		if style := firstNonEmpty(tag.Style, t.Style); style != "" {
			box.Style(style)
		}
		if tag.ID != "" {
			box.ID(tag.ID)
		}
		if len(tag.Input) > 0 {
			box.Input(tag.Input...)
		}
		row.Child(box)
	}
	return row
}

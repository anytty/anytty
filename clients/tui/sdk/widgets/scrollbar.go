package widgets

import (
	"github.com/anytty/anytty/clients/tui/sdk"
)

// Default style tokens and glyphs of Scrollbar.
const (
	DefaultScrollbarTrackStyle = "muted"
	DefaultScrollbarThumbStyle = "accent"
	DefaultScrollbarUp         = "↑"
	DefaultScrollbarDown       = "↓"
	ScrollbarTrackVertical     = "│"
	ScrollbarThumbVertical     = "┃"
	ScrollbarTrackHorizontal   = "─"
	ScrollbarThumbHorizontal   = "━"
)

// Scrollbar is a pure proportional scroll indicator usable vertically
// (Height > 0) or horizontally (Width > 0). Total is the item count, Visible
// the window size in items and Offset the first visible item. The thumb size
// is Visible*Length/Total with a minimum of one cell; the thumb position maps
// Offset onto [0, Length-size]. When the whole content fits (Total <= Visible)
// the thumb fills the track. Build draws the track and never emits protocol
// methods; the caller hit-tests ID and uses OffsetAt to turn a click into an
// offset.
type Scrollbar struct {
	ID         string
	Total      int
	Visible    int
	Offset     int
	Height     int
	Width      int
	Vertical   bool
	TrackStyle string
	ThumbStyle string
	Up         string
	Down       string
}

// IsVertical reports the resolved orientation: an explicit Vertical wins,
// otherwise a Height means vertical and a Width means horizontal.
func (s Scrollbar) IsVertical() bool {
	if s.Vertical {
		return true
	}
	if s.Width > 0 && s.Height <= 0 {
		return false
	}
	return true
}

// TrackLength returns the number of cells of the track (Height when vertical,
// Width when horizontal).
func (s Scrollbar) TrackLength() int {
	if s.IsVertical() {
		if s.Height > 0 {
			return s.Height
		}
		return 0
	}
	if s.Width > 0 {
		return s.Width
	}
	return 0
}

// thumbMaxOffset returns the largest Offset that still shows content.
func (s Scrollbar) thumbMaxOffset() int {
	max := s.Total - s.Visible
	if max < 0 {
		max = 0
	}
	return max
}

// clampOffset folds Offset into [0, Total-Visible] (never negative).
func (s Scrollbar) clampOffset() int {
	offset := s.Offset
	if offset < 0 {
		offset = 0
	}
	if max := s.thumbMaxOffset(); offset > max {
		offset = max
	}
	return offset
}

// Thumb returns the thumb start cell and size along the track. The size is at
// least one cell and never exceeds the track; a track of zero length yields
// (0, 0).
func (s Scrollbar) Thumb() (start, size int) {
	length := s.TrackLength()
	if length <= 0 {
		return 0, 0
	}
	if s.Total <= 0 || s.Total <= s.Visible {
		return 0, length
	}
	size = s.Visible * length / s.Total
	if size < 1 {
		size = 1
	}
	if size > length {
		size = length
	}
	if max := s.thumbMaxOffset(); max > 0 {
		start = s.clampOffset() * (length - size) / max
	}
	if start < 0 {
		start = 0
	}
	if maxStart := length - size; start > maxStart {
		start = maxStart
	}
	return start, size
}

// OffsetAt maps a click position (cells from the start of the track, 0-based)
// to an offset, centering the thumb on the click. Degenerate scrollbars
// return 0.
func (s Scrollbar) OffsetAt(pos int) int {
	length := s.TrackLength()
	if length <= 0 {
		return 0
	}
	max := s.thumbMaxOffset()
	if max <= 0 {
		return 0
	}
	_, size := s.Thumb()
	denom := length - size
	if denom <= 0 {
		return 0
	}
	offset := (pos - size/2) * max / denom
	if offset < 0 {
		return 0
	}
	if offset > max {
		return max
	}
	return offset
}

// Build returns the track with the proportional thumb. Up/Down are optional
// one-cell end caps drawn outside the measured track.
func (s Scrollbar) Build() *sdk.Builder {
	length := s.TrackLength()
	start, size := s.Thumb()
	trackStyle := firstNonEmpty(s.TrackStyle, DefaultScrollbarTrackStyle)
	thumbStyle := firstNonEmpty(s.ThumbStyle, DefaultScrollbarThumbStyle)
	trackGlyph, thumbGlyph := s.glyphs()
	cell := func(i int) (string, string) {
		if i >= start && i < start+size {
			return thumbGlyph, thumbStyle
		}
		return trackGlyph, trackStyle
	}
	if s.IsVertical() {
		col := sdk.Box().Flow("col").Width(1)
		if s.ID != "" {
			col.ID(s.ID)
		}
		col.Input("mouse")
		col.Height(length + s.capCount())
		if s.Up != "" {
			col.Child(sdk.Text(s.Up).Style(trackStyle).Width(1).Height(1))
		}
		for i := 0; i < length; i++ {
			glyph, style := cell(i)
			col.Child(sdk.Text(glyph).Style(style).Width(1).Height(1))
		}
		if s.Down != "" {
			col.Child(sdk.Text(s.Down).Style(trackStyle).Width(1).Height(1))
		}
		return col
	}
	row := sdk.Box().Flow("row").Height(1)
	if s.ID != "" {
		row.ID(s.ID)
	}
	row.Input("mouse")
	row.Width(length + s.capCount())
	if s.Up != "" {
		row.Child(sdk.Text(s.Up).Style(trackStyle).Width(1).Height(1))
	}
	for i := 0; i < length; i++ {
		glyph, style := cell(i)
		row.Child(sdk.Text(glyph).Style(style).Width(1).Height(1))
	}
	if s.Down != "" {
		row.Child(sdk.Text(s.Down).Style(trackStyle).Width(1).Height(1))
	}
	return row
}

func (s Scrollbar) capCount() int {
	count := 0
	if s.Up != "" {
		count++
	}
	if s.Down != "" {
		count++
	}
	return count
}

func (s Scrollbar) glyphs() (track, thumb string) {
	if s.IsVertical() {
		return ScrollbarTrackVertical, ScrollbarThumbVertical
	}
	return ScrollbarTrackHorizontal, ScrollbarThumbHorizontal
}

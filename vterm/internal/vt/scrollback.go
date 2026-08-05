package vt

import uv "github.com/charmbracelet/ultraviolet"

// DefaultScrollbackSize is the default maximum number of lines in the scrollback buffer.
const DefaultScrollbackSize = 2000

// Scrollback represents a scrollback buffer that stores lines scrolled off the screen.
type Scrollback struct {
	lines    []uv.Line
	wrapped  []bool
	maxLines int
	offset   int
	size     int
}

// NewScrollback creates a new scrollback buffer with the given maximum number of lines.
func NewScrollback(maxLines int) *Scrollback {
	if maxLines <= 0 {
		maxLines = DefaultScrollbackSize
	}
	return &Scrollback{
		lines:    make([]uv.Line, 0, min(maxLines, 1000)), // Pre-allocate reasonable capacity
		maxLines: maxLines,
	}
}

// Push adds a line to the scrollback buffer.
// If the buffer is full, the oldest line is removed.
func (s *Scrollback) Push(line uv.Line) {
	s.PushWrapped(line, false)
}

// PushWrapped adds a line to the scrollback buffer with its soft-wrap marker.
// The marker means this row visually continues onto the next row.
func (s *Scrollback) PushWrapped(line uv.Line, wrapped bool) {
	if s == nil || s.maxLines <= 0 {
		return
	}

	cloned := cloneScrollbackLine(line, wrapped)

	s.normalizeStorage()
	if s.size >= s.maxLines {
		trim := s.maxLines / 10
		if trim < 1 {
			trim = 1
		}
		if trim > s.size {
			trim = s.size
		}
		s.dropOldest(trim)
	}
	if cap(s.lines) < s.maxLines && s.offset == 0 && s.size == len(s.lines) {
		s.lines = append(s.lines, cloned)
		s.wrapped = append(s.wrapped, wrapped)
		s.size++
		return
	}
	if len(s.lines) < s.maxLines {
		s.lines = append(s.lines, nil)
		s.wrapped = append(s.wrapped, false)
	}
	index := s.physicalIndex(s.size)
	s.lines[index] = cloned
	s.wrapped[index] = wrapped
	s.size++
}

func cloneScrollbackLine(line uv.Line, wrapped bool) uv.Line {
	return cloneUVLineWithTrailingSpaces(line)
}

// PushN adds n lines from the buffer starting at line y to the scrollback.
func (s *Scrollback) PushN(buf *uv.RenderBuffer, wrapped []bool, used []int, y, n int) {
	if s == nil || buf == nil || n <= 0 {
		return
	}

	for i := range min(n, buf.Height()-y) {
		if line := buf.Line(y + i); line != nil {
			lineUsed := clampLocalInt(intAt(used, y+i), 0, len(line))
			s.PushWrapped(line[:lineUsed], boolAt(wrapped, y+i))
		}
	}
}

// Len returns the number of lines in the scrollback buffer.
func (s *Scrollback) Len() int {
	if s == nil {
		return 0
	}
	s.normalizeStorage()
	return s.size
}

// MaxLines returns the maximum number of lines the scrollback buffer can hold.
func (s *Scrollback) MaxLines() int {
	if s == nil {
		return 0
	}
	return s.maxLines
}

// SetMaxLines sets the maximum number of lines in the scrollback buffer.
// If the current number of lines exceeds the new maximum, oldest lines are removed.
func (s *Scrollback) SetMaxLines(maxLines int) {
	if s == nil || maxLines <= 0 {
		return
	}

	s.maxLines = maxLines
	s.normalizeStorage()
	if s.size > maxLines {
		s.dropOldest(s.size - maxLines)
	}
	s.compact()
}

// TrimOldest removes up to n oldest lines from the scrollback buffer.
func (s *Scrollback) TrimOldest(n int) {
	s.dropOldest(n)
}

// Line returns the line at the given index.
// Index 0 is the oldest line, Len()-1 is the most recent.
// Returns nil if index is out of bounds.
func (s *Scrollback) Line(index int) uv.Line {
	if s == nil || index < 0 {
		return nil
	}
	s.normalizeStorage()
	if index >= s.size {
		return nil
	}
	return s.lines[s.physicalIndex(index)]
}

// Lines returns all lines in the scrollback buffer.
// Index 0 is the oldest line.
func (s *Scrollback) Lines() []uv.Line {
	if s == nil {
		return nil
	}
	s.normalizeStorage()
	return s.logicalLines()
}

// LineWrapped returns whether the row visually continues onto the next row.
func (s *Scrollback) LineWrapped(index int) bool {
	if s == nil || index < 0 {
		return false
	}
	s.normalizeStorage()
	if index >= s.size {
		return false
	}
	return s.wrapped[s.physicalIndex(index)]
}

// Wrapped returns the soft-wrap markers for all scrollback rows.
func (s *Scrollback) Wrapped() []bool {
	if s == nil {
		return nil
	}
	s.normalizeStorage()
	s.normalizeWrapped()
	return s.logicalWrapped()
}

// SetLineWrapped updates the soft-wrap marker for a scrollback row.
func (s *Scrollback) SetLineWrapped(index int, wrapped bool) {
	if s == nil || index < 0 {
		return
	}
	s.normalizeStorage()
	if index >= s.size {
		return
	}
	s.normalizeWrapped()
	s.wrapped[s.physicalIndex(index)] = wrapped
}

// SetWrapped replaces all soft-wrap markers.
func (s *Scrollback) SetWrapped(wrapped []bool) {
	if s == nil {
		return
	}
	s.normalizeStorage()
	next := make([]bool, max(s.size, len(s.lines)))
	for i := 0; i < s.size && i < len(wrapped); i++ {
		next[s.physicalIndex(i)] = wrapped[i]
	}
	s.wrapped = next
}

// Clear removes all lines from the scrollback buffer.
func (s *Scrollback) Clear() {
	if s == nil {
		return
	}
	clear(s.lines)
	clear(s.wrapped)
	s.lines = s.lines[:0]
	s.wrapped = s.wrapped[:0]
	s.offset = 0
	s.size = 0
}

// CellAt returns the cell at the given position in the scrollback buffer.
// x is the column, y is the line index (0 = oldest).
// Returns nil if position is out of bounds.
func (s *Scrollback) CellAt(x, y int) *uv.Cell {
	line := s.Line(y)
	if line == nil || x < 0 || x >= len(line) {
		return nil
	}
	return &line[x]
}

func (s *Scrollback) normalizeWrapped() {
	if s == nil {
		return
	}
	s.normalizeStorage()
	switch {
	case len(s.wrapped) == len(s.lines):
	case len(s.wrapped) > len(s.lines):
		s.wrapped = s.wrapped[:len(s.lines)]
	default:
		s.wrapped = append(s.wrapped, make([]bool, len(s.lines)-len(s.wrapped))...)
	}
}

func (s *Scrollback) normalizeStorage() {
	if s == nil {
		return
	}
	if s.offset < 0 || s.offset >= max(1, len(s.lines)) {
		s.offset = 0
	}
	if s.size < 0 {
		s.size = 0
	}
	if s.size > len(s.lines) {
		s.size = len(s.lines)
	}
	if len(s.wrapped) < len(s.lines) {
		s.wrapped = append(s.wrapped, make([]bool, len(s.lines)-len(s.wrapped))...)
	} else if len(s.wrapped) > len(s.lines) {
		s.wrapped = s.wrapped[:len(s.lines)]
	}
}

func (s *Scrollback) physicalIndex(logical int) int {
	if len(s.lines) == 0 {
		return 0
	}
	return (s.offset + logical) % len(s.lines)
}

func (s *Scrollback) dropOldest(n int) {
	if s == nil || n <= 0 {
		return
	}
	s.normalizeStorage()
	if n > s.size {
		n = s.size
	}
	for i := 0; i < n; i++ {
		index := s.physicalIndex(i)
		s.lines[index] = nil
		if index < len(s.wrapped) {
			s.wrapped[index] = false
		}
	}
	if len(s.lines) > 0 {
		s.offset = (s.offset + n) % len(s.lines)
	}
	s.size -= n
	if s.size == 0 {
		s.offset = 0
	}
}

func (s *Scrollback) logicalLines() []uv.Line {
	if s == nil || s.size <= 0 {
		return nil
	}
	if s.offset == 0 && s.size == len(s.lines) {
		return s.lines[:s.size]
	}
	out := make([]uv.Line, s.size)
	for i := 0; i < s.size; i++ {
		out[i] = s.lines[s.physicalIndex(i)]
	}
	return out
}

func (s *Scrollback) logicalWrapped() []bool {
	if s == nil || s.size <= 0 {
		return nil
	}
	if s.offset == 0 && s.size == len(s.wrapped) {
		return s.wrapped[:s.size]
	}
	out := make([]bool, s.size)
	for i := 0; i < s.size; i++ {
		index := s.physicalIndex(i)
		if index < len(s.wrapped) {
			out[i] = s.wrapped[index]
		}
	}
	return out
}

func (s *Scrollback) compact() {
	if s == nil {
		return
	}
	lines := s.logicalLines()
	wrapped := s.logicalWrapped()
	s.lines = append([]uv.Line(nil), lines...)
	s.wrapped = append([]bool(nil), wrapped...)
	s.offset = 0
	s.size = len(s.lines)
}

func boolAt(values []bool, index int) bool {
	return index >= 0 && index < len(values) && values[index]
}

func intAt(values []int, index int) int {
	if index < 0 || index >= len(values) {
		return 0
	}
	return values[index]
}

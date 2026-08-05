package vt

import (
	"image/color"
	"io"
	"sync"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// SafeEmulator is a wrapper around an Emulator that adds concurrency safety.
type SafeEmulator struct {
	*Emulator
	mu sync.RWMutex
}

var _ Terminal = (*SafeEmulator)(nil)

// NewSafeEmulator creates a new SafeEmulator instance.
func NewSafeEmulator(w, h int) *SafeEmulator {
	return &SafeEmulator{
		Emulator: NewEmulator(w, h),
	}
}

// Write writes data to the emulator in a concurrency-safe manner.
func (se *SafeEmulator) Write(data []byte) (int, error) {
	se.mu.Lock()
	defer se.mu.Unlock()
	return se.Emulator.Write(data)
}

// WriteWithDamage writes data to the emulator and returns the damages observed
// while processing that batch in a concurrency-safe manner.
func (se *SafeEmulator) WriteWithDamage(data []byte) (int, error, []Damage) {
	se.mu.Lock()
	defer se.mu.Unlock()
	return se.Emulator.WriteWithDamage(data)
}

// WriteWithScrollbackDamage writes data and records only rows that enter
// scrollback in a concurrency-safe manner.
func (se *SafeEmulator) WriteWithScrollbackDamage(data []byte) (int, error, []Damage) {
	se.mu.Lock()
	defer se.mu.Unlock()
	return se.Emulator.WriteWithScrollbackDamage(data)
}

// WriteWithSemanticDamage 以并发安全方式返回 history-facing terminal semantic
// damages；调用方不能把它当完整 screen diff 使用。
func (se *SafeEmulator) WriteWithSemanticDamage(data []byte) (int, error, []Damage) {
	se.mu.Lock()
	defer se.mu.Unlock()
	return se.Emulator.WriteWithSemanticDamage(data)
}

// WriteForLineHistoryDamage 以并发安全方式返回 linehist 生产 ingest 需要的
// eviction/boundary damages，不携带普通 ordered text payload。
func (se *SafeEmulator) WriteForLineHistoryDamage(data []byte) (int, error, []Damage) {
	se.mu.Lock()
	defer se.mu.Unlock()
	return se.Emulator.WriteForLineHistoryDamage(data)
}

// Read reads data from the emulator in a concurrency-safe manner.
func (se *SafeEmulator) Read(p []byte) (int, error) {
	return se.Emulator.Read(p)
}

// Resize resizes the emulator in a concurrency-safe manner.
func (se *SafeEmulator) Resize(w, h int) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.Resize(w, h)
}

// ResizeAndTailScreen resizes the emulator and keeps the tail of the supplied
// reflowed screen rows visible while holding the emulator lock.
func (se *SafeEmulator) ResizeAndTailScreen(w, h int, rows []uv.Line, wrapped []bool, used []int) {
	se.mu.Lock()
	defer se.mu.Unlock()
	e := se.Emulator
	pos := e.CursorPosition()
	x, y := pos.X, pos.Y
	if e.IsAltScreen() {
		e.Resize(w, h)
		return
	}
	scrollback := e.scr.Scrollback()
	visibleStart := len(rows) - h
	if visibleStart < 0 {
		visibleStart = 0
	}
	if scrollback != nil {
		for i := 0; i < visibleStart; i++ {
			scrollback.PushWrapped(rows[i], i < len(wrapped) && wrapped[i])
		}
	}
	e.scrs[0].Resize(w, h)
	e.scrs[1].Resize(w, h)
	e.tabstops = uv.DefaultTabStops(w)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			e.SetCell(x, y, nil)
		}
		src := visibleStart + y
		if src < len(rows) {
			for x, cell := range rows[src] {
				if x >= w {
					break
				}
				e.SetCell(x, y, &cell)
			}
		}
		e.SetScreenLineWrapped(y, src < len(wrapped) && wrapped[src])
		if src < len(used) {
			e.SetScreenLineUsed(y, used[src])
		} else if src < len(rows) {
			e.SetScreenLineUsed(y, len(rows[src]))
		} else {
			e.SetScreenLineUsed(y, 0)
		}
	}
	e.setCursor(x, y)
	if e.isModeSet(ansi.ModeInBandResize) {
		_, _ = io.WriteString(e.pw, ansi.InBandResize(e.Height(), e.Width(), 0, 0))
	}
}

// Render renders the emulator's current state in a concurrency-safe manner.
func (se *SafeEmulator) Render() string {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.Render()
}

// SetCell sets a cell in the emulator in a concurrency-safe manner.
func (se *SafeEmulator) SetCell(x, y int, cell *uv.Cell) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.SetCell(x, y, cell)
}

// CellAt retrieves a cell from the emulator in a concurrency-safe manner.
func (se *SafeEmulator) CellAt(x, y int) *uv.Cell {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.CellAt(x, y)
}

// Line returns a visible row truncated to its logical used width in a
// concurrency-safe manner.
func (se *SafeEmulator) Line(y int) uv.Line {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.Line(y)
}

// SendKey sends a key event to the emulator in a concurrency-safe manner.
func (se *SafeEmulator) SendKey(key uv.KeyEvent) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.SendKey(key)
}

// SendMouse sends a mouse event to the emulator in a concurrency-safe manner.
func (se *SafeEmulator) SendMouse(mouse uv.MouseEvent) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.SendMouse(mouse)
}

// SendText sends text input to the emulator in a concurrency-safe manner.
func (se *SafeEmulator) SendText(text string) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.SendText(text)
}

// Paste pastes text into the emulator in a concurrency-safe manner.
func (se *SafeEmulator) Paste(text string) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.Paste(text)
}

// SetForegroundColor sets the foreground color in a concurrency-safe manner.
func (se *SafeEmulator) SetForegroundColor(color color.Color) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.SetForegroundColor(color)
}

// SetBackgroundColor sets the background color in a concurrency-safe manner.
func (se *SafeEmulator) SetBackgroundColor(color color.Color) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.SetBackgroundColor(color)
}

// SetCursorColor sets the cursor color in a concurrency-safe manner.
func (se *SafeEmulator) SetCursorColor(color color.Color) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.SetCursorColor(color)
}

// SetIndexedColor sets an indexed color in a concurrency-safe manner.
func (se *SafeEmulator) SetIndexedColor(index int, color color.Color) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.SetIndexedColor(index, color)
}

// IndexedColor retrieves an indexed color in a concurrency-safe manner.
func (se *SafeEmulator) IndexedColor(index int) color.Color {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.IndexedColor(index)
}

// Touched returns the touched lines in a concurrency-safe manner.
func (se *SafeEmulator) Touched() []*uv.LineData {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.Touched()
}

// Height returns the height of the emulator in a concurrency-safe manner.
func (se *SafeEmulator) Height() int {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.Height()
}

// Width returns the width of the emulator in a concurrency-safe manner.
func (se *SafeEmulator) Width() int {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.Width()
}

// ForegroundColor returns the foreground color in a concurrency-safe manner.
func (se *SafeEmulator) ForegroundColor() color.Color {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.ForegroundColor()
}

// BackgroundColor returns the background color in a concurrency-safe manner.
func (se *SafeEmulator) BackgroundColor() color.Color {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.BackgroundColor()
}

// CursorColor returns the cursor color in a concurrency-safe manner.
func (se *SafeEmulator) CursorColor() color.Color {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.CursorColor()
}

// CursorPosition returns the cursor position in a concurrency-safe manner.
func (se *SafeEmulator) CursorPosition() uv.Position {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.CursorPosition()
}

// Draw draws the emulator's content onto a given surface in a concurrency-safe manner.
func (se *SafeEmulator) Draw(s uv.Screen, a uv.Rectangle) {
	se.mu.RLock()
	defer se.mu.RUnlock()
	se.Emulator.Draw(s, a)
}

// Scrollback returns the scrollback buffer in a concurrency-safe manner.
func (se *SafeEmulator) Scrollback() *Scrollback {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.Scrollback()
}

// ScrollbackLen returns the number of lines in the scrollback buffer in a concurrency-safe manner.
func (se *SafeEmulator) ScrollbackLen() int {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.ScrollbackLen()
}

// ScrollbackCellAt returns a cell from the scrollback buffer in a concurrency-safe manner.
func (se *SafeEmulator) ScrollbackCellAt(x, y int) *uv.Cell {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.ScrollbackCellAt(x, y)
}

// ScrollbackLine returns a copy of a scrollback row at its stored logical width
// in a concurrency-safe manner.
func (se *SafeEmulator) ScrollbackLine(y int) uv.Line {
	se.mu.RLock()
	defer se.mu.RUnlock()
	line := se.Emulator.ScrollbackLine(y)
	if len(line) == 0 {
		return nil
	}
	return append(uv.Line(nil), line...)
}

// ScrollbackLineWrapped returns whether a scrollback row visually continues
// onto the next row in a concurrency-safe manner.
func (se *SafeEmulator) ScrollbackLineWrapped(y int) bool {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.ScrollbackLineWrapped(y)
}

// SetScrollbackLineWrapped updates whether a scrollback row visually continues
// onto the next row in a concurrency-safe manner.
func (se *SafeEmulator) SetScrollbackLineWrapped(y int, wrapped bool) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.SetScrollbackLineWrapped(y, wrapped)
}

// ScreenLineWrapped returns whether a visible row visually continues onto the
// next row in a concurrency-safe manner.
func (se *SafeEmulator) ScreenLineWrapped(y int) bool {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.ScreenLineWrapped(y)
}

// SetScreenLineWrapped updates whether a visible row visually continues onto
// the next row in a concurrency-safe manner.
func (se *SafeEmulator) SetScreenLineWrapped(y int, wrapped bool) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.SetScreenLineWrapped(y, wrapped)
}

// ScreenLineUsed returns the logical used column count for a visible row in a
// concurrency-safe manner.
func (se *SafeEmulator) ScreenLineUsed(y int) int {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.ScreenLineUsed(y)
}

// SetScreenLineUsed updates the logical used column count for a visible row in
// a concurrency-safe manner.
func (se *SafeEmulator) SetScreenLineUsed(y int, used int) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.SetScreenLineUsed(y, used)
}

// SetScrollbackSize sets the scrollback buffer size in a concurrency-safe manner.
func (se *SafeEmulator) SetScrollbackSize(maxLines int) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.SetScrollbackSize(maxLines)
}

// DisableScrollback disables storage of main-screen scrollback rows while
// keeping scrollback damage recording available.
func (se *SafeEmulator) DisableScrollback() {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.DisableScrollback()
}

// ClearScrollback clears the scrollback buffer in a concurrency-safe manner.
func (se *SafeEmulator) ClearScrollback() {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.Emulator.ClearScrollback()
}

// IsAltScreen returns whether in alternate screen mode in a concurrency-safe manner.
func (se *SafeEmulator) IsAltScreen() bool {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Emulator.IsAltScreen()
}

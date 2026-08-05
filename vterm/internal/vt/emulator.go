package vt

import (
	"image/color"
	"io"
	"sync/atomic"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/screen"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
)

const emulatorParserDataBufferSize = 64 * 1024

// Logger represents a logger interface.
type Logger interface {
	Printf(format string, v ...any)
}

// Emulator represents a virtual terminal emulator.
type Emulator struct {
	handlers

	// The terminal's indexed 256 colors.
	colors [256]color.Color

	// Both main and alt screens and a pointer to the currently active screen.
	scrs [2]Screen
	scr  *Screen

	// Character sets
	charsets [4]CharSet

	// log is the logger to use.
	logger Logger

	// terminal default colors.
	defaultFg, defaultBg, defaultCur color.Color
	fgColor, bgColor, curColor       color.Color

	// Terminal modes.
	modes ansi.Modes

	// The last written character.
	lastChar rune // either ansi.Rune or ansi.Grapheme
	// A slice of runes to compose a grapheme.
	grapheme []rune

	// The ANSI parser to use.
	parser *ansi.Parser
	// The last parser state.
	lastState parser.State

	cb Callbacks

	// The terminal's icon name and title.
	iconName, title string
	// The current reported working directory. This is not validated.
	cwd string

	// tabstop is the list of tab stops.
	tabstops *uv.TabStops

	// I/O pipes.
	pr *io.PipeReader
	pw *io.PipeWriter

	// The GL and GR character set identifiers.
	gl, gr  int
	gsingle int // temporarily select GL or GR

	// closed 是 response pipe 生命周期真值；Read drain 与 restart Close 会并发访问。
	closed atomic.Bool

	// atPhantom indicates if the cursor is out of bounds.
	// When true, and a character is written, the cursor is moved to the next line.
	atPhantom bool
}

var _ Terminal = (*Emulator)(nil)

// NewEmulator creates a new virtual terminal emulator.
func NewEmulator(w, h int) *Emulator {
	t := new(Emulator)
	t.scrs[0] = *NewScreen(w, h)
	t.scrs[1] = *NewScreen(w, h)
	t.scr = &t.scrs[0]
	t.scrs[0].cb = &t.cb
	t.scrs[1].cb = &t.cb
	t.parser = ansi.NewParser()
	t.parser.SetParamsSize(parser.MaxParamsSize)
	// 中文说明：OSC/DCS 只需要有限 data buffer；避免每个 live surface 固定预分配数 MB。
	t.parser.SetDataSize(emulatorParserDataBufferSize)
	t.parser.SetHandler(ansi.Handler{
		Print:     t.handlePrint,
		Execute:   t.handleControl,
		HandleCsi: t.handleCsi,
		HandleEsc: t.handleEsc,
		HandleDcs: t.handleDcs,
		HandleOsc: t.handleOsc,
		HandleApc: t.handleApc,
		HandlePm:  t.handlePm,
		HandleSos: t.handleSos,
	})
	t.pr, t.pw = io.Pipe()
	t.resetModes()
	t.tabstops = uv.DefaultTabStops(w)
	t.registerDefaultHandlers()

	// Default colors
	t.defaultFg = color.White
	t.defaultBg = color.Black
	t.defaultCur = color.White

	return t
}

// SetLogger sets the terminal's logger.
func (e *Emulator) SetLogger(l Logger) {
	e.logger = l
}

// SetCallbacks sets the terminal's callbacks.
func (e *Emulator) SetCallbacks(cb Callbacks) {
	e.cb = cb
	e.scrs[0].cb = &e.cb
	e.scrs[1].cb = &e.cb
}

// Touched returns the touched lines in the current screen buffer.
func (e *Emulator) Touched() []*uv.LineData {
	return e.scr.Touched()
}

// String returns a string representation of the underlying screen buffer.
func (e *Emulator) String() string {
	s := e.scr.buf.String()
	return uv.TrimSpace(s)
}

// Render renders a snapshot of the terminal screen as a string with styles and
// links encoded as ANSI escape codes.
func (e *Emulator) Render() string {
	return e.scr.buf.Render()
}

var _ uv.Screen = (*Emulator)(nil)

// Bounds returns the bounds of the terminal.
func (e *Emulator) Bounds() uv.Rectangle {
	return e.scr.Bounds()
}

// CellAt returns the current focused screen cell at the given x, y position.
// It returns nil if the cell is out of bounds.
func (e *Emulator) CellAt(x, y int) *uv.Cell {
	return e.scr.CellAt(x, y)
}

// Line returns the current focused screen row truncated to its logical used
// width. The returned slice aliases emulator storage.
func (e *Emulator) Line(y int) uv.Line {
	return e.scr.Line(y)
}

// ScreenLineUsed returns the logical used column count for a visible row.
func (e *Emulator) ScreenLineUsed(y int) int {
	if e.scr == nil {
		return 0
	}
	return e.scr.LineUsed(y)
}

// SetCell sets the current focused screen cell at the given x, y position.
func (e *Emulator) SetCell(x, y int, c *uv.Cell) {
	e.scr.SetCell(x, y, c)
}

// WidthMethod returns the width method used by the terminal.
func (e *Emulator) WidthMethod() uv.WidthMethod {
	if e.isModeSet(ansi.ModeUnicodeCore) {
		return ansi.GraphemeWidth
	}
	return ansi.WcWidth
}

// Draw implements the [uv.Drawable] interface.
func (e *Emulator) Draw(scr uv.Screen, area uv.Rectangle) {
	bg := uv.EmptyCell
	bg.Style.Bg = e.BackgroundColor()
	screen.FillArea(scr, &bg, area)
	for y := range e.Touched() {
		if y < 0 || y >= e.Height() {
			continue
		}
		for x := 0; x < e.Width(); {
			w := 1
			cell := e.CellAt(x, y)
			if cell != nil {
				cell = cell.Clone()
				if cell.Width > 1 {
					w = cell.Width
				}
				if cell.Style.Bg == nil && e.bgColor != nil {
					cell.Style.Bg = e.bgColor
				}
				if cell.Style.Fg == nil && e.fgColor != nil {
					cell.Style.Fg = e.fgColor
				}
				scr.SetCell(x+area.Min.X, y+area.Min.Y, cell)
			}
			x += w
		}
	}
}

// Height returns the height of the terminal.
func (e *Emulator) Height() int {
	return e.scr.Height()
}

// Width returns the width of the terminal.
func (e *Emulator) Width() int {
	return e.scr.Width()
}

// CursorPosition returns the terminal's cursor position.
func (e *Emulator) CursorPosition() uv.Position {
	x, y := e.scr.CursorPosition()
	return uv.Pos(x, y)
}

// Resize resizes the terminal.
func (e *Emulator) Resize(width int, height int) {
	x, y := e.scr.CursorPosition()
	if e.atPhantom {
		if x < width-1 {
			e.atPhantom = false
			x++
		}
	}

	if y < 0 {
		y = 0
	}
	if y >= height {
		y = height - 1
	}
	if x < 0 {
		x = 0
	}
	if x >= width {
		x = width - 1
	}

	if !e.IsAltScreen() {
		x, y = e.scrs[0].Reflow(width, height, x, y)
	} else {
		_, _ = e.scrs[0].Reflow(width, height, x, y)
	}
	e.scrs[1].Resize(width, height)
	e.tabstops = uv.DefaultTabStops(width)

	e.setCursor(x, y)

	if e.isModeSet(ansi.ModeInBandResize) {
		_, _ = io.WriteString(e.pw, ansi.InBandResize(e.Height(), e.Width(), 0, 0))
	}
}

// Read reads data from the terminal input buffer.
func (e *Emulator) Read(p []byte) (n int, err error) {
	if e.closed.Load() {
		return 0, io.EOF
	}

	return e.pr.Read(p) //nolint:wrapcheck
}

// Close closes the terminal.
func (e *Emulator) Close() error {
	if !e.closed.CompareAndSwap(false, true) {
		return nil
	}
	return e.pw.CloseWithError(io.EOF) //nolint:wrapcheck
}

// Write writes data to the terminal output buffer.
func (e *Emulator) Write(p []byte) (n int, err error) {
	if e.closed.Load() {
		return 0, io.ErrClosedPipe
	}

	if e.tryFastSGRText(p) {
		return len(p), nil
	}

	for i := 0; i < len(p); {
		if e.canFastPrintASCII() && isPrintableASCII(p[i]) {
			start := i
			i++
			for i < len(p) && isPrintableASCII(p[i]) {
				i++
			}
			e.handleASCIIPrintRun(p[start:i])
			e.lastState = parser.GroundState
			continue
		}
		e.parser.Advance(p[i])
		state := e.parser.State()
		// flush grapheme if we transitioned to a non-utf8 state or we have
		// written the whole byte slice.
		if len(e.grapheme) > 0 {
			if (e.lastState == parser.GroundState && state != parser.Utf8State) || i == len(p)-1 {
				e.flushGrapheme()
			}
		}
		e.lastState = state
		i++
	}
	return len(p), nil
}

func (e *Emulator) canFastPrintASCII() bool {
	return e != nil &&
		e.parser.State() == parser.GroundState &&
		len(e.grapheme) == 0 &&
		e.gsingle == 0 &&
		e.charsets[e.gl] == nil
}

func isPrintableASCII(b byte) bool {
	return b >= ansi.SP && b < ansi.DEL
}

// WriteWithDamage writes data to the terminal output buffer and returns the
// structured damages observed while processing that batch.
func (e *Emulator) WriteWithDamage(p []byte) (n int, err error, damages []Damage) {
	if e.closed.Load() {
		return 0, io.ErrClosedPipe, nil
	}
	recorder := &screenDamageRecorder{}
	e.scrs[0].damage = recorder
	e.scrs[1].damage = recorder
	defer func() {
		e.scrs[0].damage = nil
		e.scrs[1].damage = nil
	}()
	n, err = e.Write(p)
	return n, err, recorder.snapshot()
}

// WriteWithScrollbackDamage writes data to the terminal output buffer and only
// records rows that leave the visible screen and enter scrollback.
func (e *Emulator) WriteWithScrollbackDamage(p []byte) (n int, err error, damages []Damage) {
	if e.closed.Load() {
		return 0, io.ErrClosedPipe, nil
	}
	recorder := &screenDamageRecorder{scrollbackOnly: true}
	e.scrs[0].damage = recorder
	e.scrs[1].damage = recorder
	defer func() {
		e.scrs[0].damage = nil
		e.scrs[1].damage = nil
	}()
	n, err = e.Write(p)
	return n, err, recorder.snapshot()
}

// WriteWithSemanticDamage 写入 PTY 输出，并只记录 history 需要的终端语义：
// ordered text/control/mode、滚动几何，以及真实离开可见屏的 scrollback rows。
// 调用边界：它仍更新同一个 emulator live screen，但不记录 screen diff cell payload。
func (e *Emulator) WriteWithSemanticDamage(p []byte) (n int, err error, damages []Damage) {
	if e.closed.Load() {
		return 0, io.ErrClosedPipe, nil
	}
	recorder := &screenDamageRecorder{semanticOnly: true}
	e.scrs[0].damage = recorder
	e.scrs[1].damage = recorder
	defer func() {
		e.scrs[0].damage = nil
		e.scrs[1].damage = nil
	}()
	n, err = e.Write(p)
	return n, err, recorder.snapshot()
}

// WriteForLineHistoryDamage 写入 PTY 输出，并只记录 linehist 需要的 eviction
// 与边界 proof。它不记录普通 text/control payload，避免 history ingest 热路径
// 在 100K/1M stdout 下生成第二份 ordered op backlog。
func (e *Emulator) WriteForLineHistoryDamage(p []byte) (n int, err error, damages []Damage) {
	if e.closed.Load() {
		return 0, io.ErrClosedPipe, nil
	}
	recorder := &screenDamageRecorder{lineHistoryOnly: true}
	e.scrs[0].damage = recorder
	e.scrs[1].damage = recorder
	defer func() {
		e.scrs[0].damage = nil
		e.scrs[1].damage = nil
	}()
	n, err = e.Write(p)
	return n, err, recorder.snapshot()
}

// WriteString writes a string to the terminal output buffer.
func (e *Emulator) WriteString(s string) (n int, err error) {
	return e.Write([]byte(s))
}

// InputPipe returns the terminal's input pipe.
// This can be used to send input to the terminal.
func (e *Emulator) InputPipe() io.Writer {
	return e.pw
}

// Paste pastes text into the terminal.
// If bracketed paste mode is enabled, the text is bracketed with the
// appropriate escape sequences.
func (e *Emulator) Paste(text string) {
	if e.isModeSet(ansi.ModeBracketedPaste) {
		_, _ = io.WriteString(e.pw, ansi.BracketedPasteStart)
		defer io.WriteString(e.pw, ansi.BracketedPasteEnd) //nolint:errcheck
	}

	_, _ = io.WriteString(e.pw, text)
}

// SendText sends arbitrary text to the terminal.
func (e *Emulator) SendText(text string) {
	_, _ = io.WriteString(e.pw, text)
}

// SendKeys sends multiple keys to the terminal.
func (e *Emulator) SendKeys(keys ...uv.KeyEvent) {
	for _, k := range keys {
		e.SendKey(k)
	}
}

// ForegroundColor returns the terminal's foreground color. This returns nil if
// the foreground color is not set which means the outer terminal color is
// used.
func (e *Emulator) ForegroundColor() color.Color {
	if e.fgColor == nil {
		return e.defaultFg
	}
	return e.fgColor
}

// SetForegroundColor sets the terminal's foreground color.
func (e *Emulator) SetForegroundColor(c color.Color) {
	if c == nil {
		c = e.defaultFg
	}
	e.fgColor = c
	if e.cb.ForegroundColor != nil {
		e.cb.ForegroundColor(c)
	}
}

// SetDefaultForegroundColor sets the terminal's default foreground color.
func (e *Emulator) SetDefaultForegroundColor(c color.Color) {
	if c == nil {
		c = color.White
	}
	e.defaultFg = c
}

// BackgroundColor returns the terminal's background color. This returns nil if
// the background color is not set which means the outer terminal color is
// used.
func (e *Emulator) BackgroundColor() color.Color {
	if e.bgColor == nil {
		return e.defaultBg
	}
	return e.bgColor
}

// SetBackgroundColor sets the terminal's background color.
func (e *Emulator) SetBackgroundColor(c color.Color) {
	if c == nil {
		c = e.defaultBg
	}
	e.bgColor = c
	if e.cb.BackgroundColor != nil {
		e.cb.BackgroundColor(c)
	}
}

// SetDefaultBackgroundColor sets the terminal's default background color.
func (e *Emulator) SetDefaultBackgroundColor(c color.Color) {
	if c == nil {
		c = color.Black
	}
	e.defaultBg = c
}

// CursorColor returns the terminal's cursor color. This returns nil if the
// cursor color is not set which means the outer terminal color is used.
func (e *Emulator) CursorColor() color.Color {
	if e.curColor == nil {
		return e.defaultCur
	}
	return e.curColor
}

// SetCursorColor sets the terminal's cursor color.
func (e *Emulator) SetCursorColor(c color.Color) {
	if c == nil {
		c = e.defaultCur
	}
	e.curColor = c
	if e.cb.CursorColor != nil {
		e.cb.CursorColor(c)
	}
}

// SetDefaultCursorColor sets the terminal's default cursor color.
func (e *Emulator) SetDefaultCursorColor(c color.Color) {
	if c == nil {
		c = color.White
	}
	e.defaultCur = c
}

// IndexedColor returns a terminal's indexed color. An indexed color is a color
// between 0 and 255.
func (e *Emulator) IndexedColor(i int) color.Color {
	if i < 0 || i > 255 {
		return nil
	}

	c := e.colors[i]
	if c == nil {
		// Return the default color.
		return ansi.IndexedColor(i) //nolint:gosec
	}

	return c
}

// SetIndexedColor sets a terminal's indexed color.
// The index must be between 0 and 255.
func (e *Emulator) SetIndexedColor(i int, c color.Color) {
	if i < 0 || i > 255 {
		return
	}

	e.colors[i] = c
}

// resetTabStops resets the terminal tab stops to the default set.
func (e *Emulator) resetTabStops() {
	e.tabstops = uv.DefaultTabStops(e.Width())
}

func (e *Emulator) logf(format string, v ...any) {
	if e.logger != nil {
		e.logger.Printf(format, v...)
	}
}

// Scrollback returns the scrollback buffer for the main screen.
// Returns nil if the terminal is in alternate screen mode, as the alternate
// screen typically doesn't use scrollback.
func (e *Emulator) Scrollback() *Scrollback {
	// Return main screen's scrollback only
	return e.scrs[0].Scrollback()
}

// ScrollbackLen returns the number of lines in the scrollback buffer.
func (e *Emulator) ScrollbackLen() int {
	sb := e.Scrollback()
	if sb == nil {
		return 0
	}
	return sb.Len()
}

// ScrollbackCellAt returns the cell at the given position in the scrollback buffer.
// x is the column, y is the line index (0 = oldest line in scrollback).
// Returns nil if position is out of bounds.
func (e *Emulator) ScrollbackCellAt(x, y int) *uv.Cell {
	sb := e.Scrollback()
	if sb == nil {
		return nil
	}
	return sb.CellAt(x, y)
}

// ScrollbackLine returns a scrollback row at its stored logical width.
// The returned slice aliases emulator storage.
func (e *Emulator) ScrollbackLine(y int) uv.Line {
	sb := e.Scrollback()
	if sb == nil {
		return nil
	}
	return sb.Line(y)
}

// ScrollbackLineWrapped returns whether the scrollback row visually continues
// onto the next row.
func (e *Emulator) ScrollbackLineWrapped(y int) bool {
	sb := e.Scrollback()
	if sb == nil {
		return false
	}
	return sb.LineWrapped(y)
}

// SetScrollbackLineWrapped updates whether a scrollback row visually continues
// onto the next row.
func (e *Emulator) SetScrollbackLineWrapped(y int, wrapped bool) {
	sb := e.Scrollback()
	if sb == nil {
		return
	}
	sb.SetLineWrapped(y, wrapped)
}

// SetScreenLineUsed updates the logical used column count for a visible row.
func (e *Emulator) SetScreenLineUsed(y int, used int) {
	if e.scr == nil {
		return
	}
	e.scr.SetLineUsed(y, used)
}

// ScreenLineWrapped returns whether the visible row visually continues onto
// the next row.
func (e *Emulator) ScreenLineWrapped(y int) bool {
	if e.scr == nil {
		return false
	}
	return e.scr.LineWrapped(y)
}

// SetScreenLineWrapped updates whether a visible row visually continues onto
// the next row.
func (e *Emulator) SetScreenLineWrapped(y int, wrapped bool) {
	if e.scr == nil {
		return
	}
	e.scr.SetLineWrapped(y, wrapped)
}

// PrimaryLine returns the primary (normal) screen row truncated to its
// logical used width, regardless of which screen is currently active. The
// returned slice aliases emulator storage.
func (e *Emulator) PrimaryLine(y int) uv.Line {
	return e.scrs[0].Line(y)
}

// PrimaryLineWrapped returns whether the primary screen row visually
// continues onto the next row, regardless of which screen is active.
func (e *Emulator) PrimaryLineWrapped(y int) bool {
	return e.scrs[0].LineWrapped(y)
}

// PrimaryHeight returns the primary screen height, regardless of which
// screen is active.
func (e *Emulator) PrimaryHeight() int {
	return e.scrs[0].Height()
}

// SetScrollbackSize sets the maximum number of lines in the scrollback buffer.
func (e *Emulator) SetScrollbackSize(maxLines int) {
	e.scrs[0].SetScrollbackSize(maxLines)
}

// DisableScrollback disables storage of main-screen scrollback rows. Damage
// recording can still observe rows as they leave the visible screen.
func (e *Emulator) DisableScrollback() {
	e.scrs[0].SetScrollback(nil)
}

// ClearScrollback clears the scrollback buffer.
func (e *Emulator) ClearScrollback() {
	sb := e.Scrollback()
	if sb != nil {
		sb.Clear()
	}
}

// IsAltScreen returns whether the terminal is in alternate screen mode.
func (e *Emulator) IsAltScreen() bool {
	return e.scr == &e.scrs[1]
}

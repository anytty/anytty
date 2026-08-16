package vt

import uv "github.com/charmbracelet/ultraviolet"

// Damage represents a damaged area.
type Damage interface {
	// Bounds returns the bounds of the damaged area.
	Bounds() uv.Rectangle
}

// CellDamage represents a damaged cell.
type CellDamage struct {
	X, Y  int
	Width int
}

// Bounds returns the bounds of the damaged area.
func (d CellDamage) Bounds() uv.Rectangle {
	return uv.Rect(d.X, d.Y, d.Width, 1)
}

// SpanDamage represents a sequence of cells written on a single row.
type SpanDamage struct {
	X, Y  int
	Cells []uv.Cell
}

// Bounds returns the bounds of the damaged area.
func (d SpanDamage) Bounds() uv.Rectangle {
	width := 0
	for _, cell := range d.Cells {
		if cell.Width > 0 {
			width += cell.Width
			continue
		}
		width++
	}
	if width <= 0 {
		width = 1
	}
	return uv.Rect(d.X, d.Y, width, 1)
}

// TextDamage 表示真实 print path 写入的文本语义。它和 SpanDamage 分开：
// SpanDamage 是 screen diff，可能来自填充、clear 或重排；TextDamage 才能给
// history projector 当成 PTY 文本写入语义。
type TextDamage struct {
	X, Y  int
	Cells []uv.Cell
	Runs  []ScrollbackRun
	ASCII bool
	Text  string
}

// Bounds returns the bounds of the text semantic area.
func (d TextDamage) Bounds() uv.Rectangle {
	if d.Text != "" {
		return uv.Rect(d.X, d.Y, d.X+len(d.Text), d.Y+1)
	}
	if len(d.Runs) > 0 {
		width := 0
		for _, run := range d.Runs {
			width += len(run.Text)
		}
		if width <= 0 {
			width = 1
		}
		return uv.Rect(d.X, d.Y, d.X+width, d.Y+1)
	}
	return SpanDamage{X: d.X, Y: d.Y, Cells: d.Cells}.Bounds()
}

// RectDamage represents a damaged rectangle.
type RectDamage uv.Rectangle

// Bounds returns the bounds of the damaged area.
func (d RectDamage) Bounds() uv.Rectangle {
	return uv.Rectangle(d)
}

// X returns the x-coordinate of the damaged area.
func (d RectDamage) X() int {
	return uv.Rectangle(d).Min.X
}

// Y returns the y-coordinate of the damaged area.
func (d RectDamage) Y() int {
	return uv.Rectangle(d).Min.Y
}

// Width returns the width of the damaged area.
func (d RectDamage) Width() int {
	return uv.Rectangle(d).Dx()
}

// Height returns the height of the damaged area.
func (d RectDamage) Height() int {
	return uv.Rectangle(d).Dy()
}

// ClearDamage represents a cleared rectangle.
type ClearDamage uv.Rectangle

// Bounds returns the bounds of the damaged area.
func (d ClearDamage) Bounds() uv.Rectangle {
	return uv.Rectangle(d)
}

// X returns the x-coordinate of the damaged area.
func (d ClearDamage) X() int {
	return uv.Rectangle(d).Min.X
}

// Y returns the y-coordinate of the damaged area.
func (d ClearDamage) Y() int {
	return uv.Rectangle(d).Min.Y
}

// Width returns the width of the damaged area.
func (d ClearDamage) Width() int {
	return uv.Rectangle(d).Dx()
}

// Height returns the height of the damaged area.
func (d ClearDamage) Height() int {
	return uv.Rectangle(d).Dy()
}

// ScreenDamage represents a damaged screen.
type ScreenDamage struct {
	Width, Height int
}

// Bounds returns the bounds of the damaged area.
func (d ScreenDamage) Bounds() uv.Rectangle {
	return uv.Rect(0, 0, d.Width, d.Height)
}

// MoveDamage represents a moved area.
// The area is moved from the source to the destination.
type MoveDamage struct {
	Src, Dst uv.Rectangle
}

// Bounds returns the union of the source and destination rectangles.
func (d MoveDamage) Bounds() uv.Rectangle {
	return d.Src.Union(d.Dst)
}

// ScrollDamage represents a scrolled area.
// The area is scrolled by the given deltas.
type ScrollDamage struct {
	uv.Rectangle
	Dx, Dy int
}

// Bounds returns the damaged rectangle.
func (d ScrollDamage) Bounds() uv.Rectangle {
	return d.Rectangle
}

// ControlDamage 记录同一 parser transaction 内的控制语义；这些控制不一定
// 产生 cell damage，但 history projector 需要按顺序消费。
type ControlDamage struct {
	Kind      string
	X, Y      int
	Mode      int
	Bottom    int
	Cell      uv.Cell
	HasCell   bool
	ScrollOut []ScrollbackDamage
}

// Bounds 返回控制语义发生时的 cursor 位置。
func (d ControlDamage) Bounds() uv.Rectangle {
	return uv.Rect(d.X, d.Y, 1, 1)
}

// ModeDamage 记录同一 parser transaction 内的 mode 变化。
type ModeDamage struct {
	Mode    int
	Private bool
	Enabled bool
}

// Bounds 返回 origin 占位；mode 变化是语义控制，不是 cell mutation。
func (d ModeDamage) Bounds() uv.Rectangle {
	return uv.Rect(0, 0, 1, 1)
}

// ScrollbackDamage represents a row captured before it leaves the visible
// screen and enters scrollback.
type ScrollbackDamage struct {
	Y       int
	Cells   []uv.Cell
	Runs    []ScrollbackRun
	Wrapped bool
	ASCII   bool
	Text    string
}

type ScrollbackRun struct {
	Style uv.Style
	Text  string
}

// Bounds returns the original row bounds.
func (d ScrollbackDamage) Bounds() uv.Rectangle {
	return uv.Rect(0, d.Y, len(d.Cells), 1)
}

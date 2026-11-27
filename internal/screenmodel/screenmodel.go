// Package screenmodel 是 TUI 与手机端共用的唯一 live-screen 模型。
//
// 服务端的 native screen delta（行替换 + 行复制 + 光标 + 模式）只在这里被
// 应用一次；两端各自只做渲染投影：
//   - TUI：投影到 ANSI 解析器的网格（clients/tui/render/ansi 的 LoadRows）；
//   - 手机：binding 把应用后的行交给 Flutter 绘制。
//
// 这样“怎么从 delta 得到新屏幕”只有一份实现，不再出现客户端各写一套的情况。
package screenmodel

import (
	"github.com/anytty/anytty/proto/access/apipb"
)

// Cell 是一个网格单元。宽字形的右侧是 Continuation=true 的占位单元。
// Style 保留服务端的结构化样式，渲染端各自投影（TUI 转 SGR token，手机直接
// 交给 Flutter），不需要把样式再编码成转义序列。
type Cell struct {
	Text         string
	Width        int
	Continuation bool
	Style        *apipb.CellStyle
}

// Screen 是一次应用之后的完整屏幕状态。
type Screen struct {
	Lines     [][]Cell
	Cols      int
	Rows      int
	CursorX   int
	CursorY   int
	CursorVisible bool
	// CursorShape 是 "block" / "underline" / "bar"。
	CursorShape string
	Modes       *apipb.TerminalModes
}

// Line 返回一行，越界返回 nil。
func (s *Screen) Line(y int) []Cell {
	if s == nil || y < 0 || y >= len(s.Lines) {
		return nil
	}
	return s.Lines[y]
}

// AlternateScreen 报告当前是否在备用屏。
func (s *Screen) AlternateScreen() bool {
	return s != nil && s.Modes.GetAlternateScreen()
}

// Apply 把一帧 delta 应用到 prev 上，返回新屏幕和发生变化的行号。
//
// changed 为 nil 表示整屏有效（首帧、尺寸变化、alt-screen 切换或
// full_replace），调用方应整屏重绘。prev 为 nil 时按首帧处理。
func Apply(prev *Screen, delta *apipb.NativeScreenResult) (*Screen, []int) {
	if delta == nil {
		return prev, nil
	}
	cols := int(delta.GetSize().GetCols())
	rows := int(delta.GetSize().GetRows())
	if cols <= 0 || rows <= 0 {
		if prev == nil {
			return nil, nil
		}
		cols, rows = prev.Cols, prev.Rows
	}
	full := prev == nil || prev.Cols != cols || prev.Rows != rows || delta.GetFullReplace()
	if !full && prev.AlternateScreen() != delta.GetModes().GetAlternateScreen() {
		// alt-screen 切换后两份缓冲区内容不同，delta 没有意义。
		full = true
	}

	next := &Screen{
		Cols:          cols,
		Rows:          rows,
		CursorX:       0,
		CursorY:       0,
		CursorVisible: true,
		CursorShape:   "block",
	}
	if full {
		next.Lines = blankLines(cols, rows)
	} else {
		next.Lines = append([][]Cell(nil), prev.Lines...)
		next.CursorX, next.CursorY = prev.CursorX, prev.CursorY
		next.CursorVisible, next.CursorShape = prev.CursorVisible, prev.CursorShape
	}

	var changed []int
	if full {
		// 整屏帧按行号落位；缺失的行保持空白。
		for _, replacement := range delta.GetRowReplacements() {
			index := int(replacement.GetRowIndex())
			if index < 0 || index >= rows {
				continue
			}
			next.Lines[index] = buildRow(cols, replacement)
		}
	} else {
		// 行复制从 prev 取源行：复制链不会级联到同一帧里已被覆盖的行。
		for _, copyRow := range delta.GetRowCopies() {
			count := int(copyRow.GetCount())
			source := int(copyRow.GetSourceRow())
			destination := int(copyRow.GetDestinationRow())
			if count <= 0 || source < 0 || destination < 0 ||
				source+count > rows || destination+count > rows {
				continue
			}
			for offset := 0; offset < count; offset++ {
				next.Lines[destination+offset] = prev.Lines[source+offset]
				changed = append(changed, destination+offset)
			}
		}
		for _, replacement := range delta.GetRowReplacements() {
			index := int(replacement.GetRowIndex())
			if index < 0 || index >= rows {
				continue
			}
			next.Lines[index] = buildRow(cols, replacement)
			changed = append(changed, index)
		}
	}

	if cursor := delta.GetCursor(); cursor != nil {
		next.CursorX = clamp(int(cursor.GetCol()), 0, max(cols-1, 0))
		next.CursorY = clamp(int(cursor.GetRow()), 0, max(rows-1, 0))
		next.CursorVisible = cursor.GetVisible()
		next.CursorShape = cursorShape(cursor.GetShape())
	}
	if modes := delta.GetModes(); modes != nil {
		next.Modes = modes
	} else if prev != nil {
		next.Modes = prev.Modes
	}
	return next, changed
}

// buildRow 把一帧行替换展开成恰好 cols 个单元。
//
// 空内容单元按宽度展开成空格（与服务端 ANSI 快照的写法一致）；tail_fill
// 补齐剩余列，没有 tail_fill 的尾部按默认样式留白。
func buildRow(cols int, replacement *apipb.ScreenRowReplace) []Cell {
	row := make([]Cell, 0, cols)
	used := 0
	if replacement != nil {
		for _, cell := range replacement.GetRow().GetCells() {
			if used >= cols {
				break
			}
			width := int(cell.GetWidth())
			if width <= 0 {
				width = 1
			}
			if width > cols-used {
				width = cols - used
			}
			style := cell.GetStyle()
			if cell.GetContent() == "" {
				for index := 0; index < width; index++ {
					row = append(row, Cell{Text: " ", Width: 1, Style: style})
				}
				used += width
				continue
			}
			row = append(row, Cell{Text: cell.GetContent(), Width: width, Style: style})
			used += width
			for index := 1; index < width; index++ {
				row = append(row, Cell{Continuation: true, Style: style})
			}
		}
		if tail := replacement.GetRow().GetTailFill(); tail != nil {
			for used < cols {
				row = append(row, Cell{Text: " ", Width: 1, Style: tail})
				used++
			}
		}
	}
	// 没有 tail_fill 的尾部是真的空白单元（没有内容也没有样式），
	// 与 ANSI 快照里清屏后的单元一致。
	for len(row) < cols {
		row = append(row, Cell{})
	}
	return row
}

func blankLines(cols, rows int) [][]Cell {
	lines := make([][]Cell, rows)
	for y := 0; y < rows; y++ {
		lines[y] = make([]Cell, cols)
	}
	return lines
}

func cursorShape(shape apipb.CursorShape) string {
	switch shape {
	case apipb.CursorShape_CURSOR_SHAPE_BAR:
		return "bar"
	case apipb.CursorShape_CURSOR_SHAPE_UNDERLINE:
		return "underline"
	default:
		return "block"
	}
}

func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

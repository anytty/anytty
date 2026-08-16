package terminalhost

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/anytty/anytty/tui/render"
	xansi "github.com/charmbracelet/x/ansi"
)

const (
	cursorHome              = "\x1b[H"
	clearScreen             = "\x1b[2J"
	clearLine               = "\x1b[2K"
	scrollUpOne             = "\x1b[S"
	scrollDownOne           = "\x1b[T"
	resetScrollRegion       = "\x1b[r"
	cursorShapeBlock        = "\x1b[2 q"
	cursorShapeBar          = "\x1b[6 q"
	synchronizedOutputBegin = "\x1b[?2026h"
	synchronizedOutputEnd   = "\x1b[?2026l"
)

// FrameSink 把 Renderer 生成的 frame 直接写到宿主输出。
type FrameSink struct {
	mu                    sync.Mutex
	writer                io.Writer
	lastLines             []string
	lastWidth             int
	lastHeight            int
	lastCursor            string
	lastVisibleCursor     render.Cursor
	lastVisibleCursorRect render.Rect
	hasLastVisibleCursor  bool
	hasLastFrame          bool
}

func NewFrameSink(writer io.Writer) *FrameSink {
	return &FrameSink{writer: writer}
}

func (sink *FrameSink) NeedsCompleteFrame() bool {
	return false
}

func (sink *FrameSink) WriteFrame(frame render.Frame) error {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if frame.Patch != nil {
		return sink.writePatchFrame(frame)
	}
	lines := frame.ANSILines
	if len(lines) == 0 {
		lines = frame.Lines
	}
	width := frame.Metadata.Width
	height := frame.Metadata.Height
	if height == 0 {
		height = len(lines)
	}
	fullRepaint := frame.Metadata.ForceFullRepaint || !sink.hasLastFrame || width != sink.lastWidth || height != sink.lastHeight || len(lines) != len(sink.lastLines)
	cursorSequence := sink.frameCursorSequence(frame)
	dirtyRows := fullRepaint
	if !dirtyRows {
		for i, line := range lines {
			if sink.lastLines[i] != line {
				dirtyRows = true
				break
			}
		}
	}
	if !dirtyRows && sink.lastCursor == cursorSequence {
		return nil
	}

	var builder strings.Builder
	builder.Grow(sink.frameWriteCapacity(lines, width, fullRepaint, cursorSequence))
	builder.WriteString(synchronizedOutputBegin)
	builder.WriteString(hideCursor)
	if !fullRepaint && sink.writeScrollShift(&builder, lines) {
		// 一行滚动时让终端移动已有行，只补新露出的一行；避免滚轮每次重写整块 copy history。
	} else if !fullRepaint {
		sink.writePresentedRows(&builder, lines, width)
	} else {
		builder.WriteString(cursorHome)
		builder.WriteString(clearScreen)
		for i, line := range lines {
			// 满宽 frame 写到最后一列后继续输出换行，会在部分终端触发额外自动换行；
			// 每行绝对定位可以保持 pane 竖向边框连续。
			builder.WriteString(cursorPosition(i+1, 1))
			builder.WriteString(clearLine)
			builder.WriteString(line)
			builder.WriteString(render.ANSIReset)
		}
	}
	builder.WriteString(render.ANSIReset)
	builder.WriteString(cursorSequence)
	builder.WriteString(synchronizedOutputEnd)
	_, err := io.WriteString(sink.writer, builder.String())
	if err != nil {
		return err
	}
	sink.lastLines = append(sink.lastLines[:0], lines...)
	sink.lastWidth = width
	sink.lastHeight = height
	sink.lastCursor = cursorSequence
	sink.rememberVisibleCursor(frame)
	sink.hasLastFrame = true
	return nil
}

func (sink *FrameSink) writeScrollShift(builder *strings.Builder, lines []string) bool {
	if len(lines) != len(sink.lastLines) || len(lines) < 3 {
		return false
	}
	if start, end, ok := shiftedUpWindow(sink.lastLines, lines); ok {
		builder.WriteString(scrollRegion(start+1, end+1))
		builder.WriteString(cursorPosition(end+1, 1))
		builder.WriteString(scrollUpOne)
		builder.WriteString(resetScrollRegion)
		builder.WriteString(cursorPosition(end+1, 1))
		builder.WriteString(clearLine)
		builder.WriteString(lines[end])
		builder.WriteString(render.ANSIReset)
		return true
	}
	if start, end, ok := shiftedDownWindow(sink.lastLines, lines); ok {
		builder.WriteString(scrollRegion(start+1, end+1))
		builder.WriteString(cursorPosition(start+1, 1))
		builder.WriteString(scrollDownOne)
		builder.WriteString(resetScrollRegion)
		builder.WriteString(cursorPosition(start+1, 1))
		builder.WriteString(clearLine)
		builder.WriteString(lines[start])
		builder.WriteString(render.ANSIReset)
		return true
	}
	return false
}

func (sink *FrameSink) writePresentedRows(builder *strings.Builder, lines []string, width int) {
	for row := 0; row < len(lines); row++ {
		if row < len(sink.lastLines) && sink.lastLines[row] == lines[row] {
			continue
		}
		builder.WriteString(cursorPosition(row+1, 1))
		previous := ""
		if row < len(sink.lastLines) {
			previous = sink.lastLines[row]
		}
		if frameSinkLineRequiresWholeRowClear(lines[row], previous) {
			// 中文说明：renderer 的 ANSI 行可能含 CSI G/X 这类绝对列定位；
			// 这种行不是线性文本，不能靠 StringWidth 估算尾部补空格。
			builder.WriteString(render.ANSIReset)
			builder.WriteString(clearLine)
			builder.WriteString(lines[row])
			builder.WriteString(render.ANSIReset)
			continue
		}
		// 中文说明：同尺寸帧不清整行，直接覆盖变化行；新内容变短时补空格盖掉旧尾巴，
		// 避免 clear-line 造成整行闪烁。
		writePresentedRowLine(builder, lines[row], rowWidth(width, lines[row], sink.lastLines[row]))
		builder.WriteString(render.ANSIReset)
	}
}

func (sink *FrameSink) frameWriteCapacity(lines []string, width int, fullRepaint bool, cursorSequence string) int {
	capacity := len(synchronizedOutputBegin) + len(hideCursor) + len(render.ANSIReset) + len(cursorSequence) + len(synchronizedOutputEnd)
	if fullRepaint {
		capacity += len(cursorHome) + len(clearScreen)
		for row, line := range lines {
			capacity += cursorPositionCapacity(row+1, 1) + len(clearLine) + len(line) + len(render.ANSIReset)
		}
		return capacity
	}
	for row, line := range lines {
		if row < len(sink.lastLines) && sink.lastLines[row] == line {
			continue
		}
		previous := ""
		if row < len(sink.lastLines) {
			previous = sink.lastLines[row]
		}
		if frameSinkLineRequiresWholeRowClear(line, previous) {
			capacity += cursorPositionCapacity(row+1, 1) + len(render.ANSIReset) + len(clearLine) + len(line) + len(render.ANSIReset)
			continue
		}
		targetWidth := rowWidth(width, line, previous)
		padding := targetWidth - frameSinkLineWidth(line)
		if padding < 0 {
			padding = 0
		}
		capacity += cursorPositionCapacity(row+1, 1) + len(line) + len(render.ANSIReset) + padding + len(render.ANSIReset)
	}
	return capacity
}

func frameSinkLineRequiresWholeRowClear(next string, previous string) bool {
	return frameSinkLineHasCursorAddressing(next) || frameSinkLineHasCursorAddressing(previous)
}

func frameSinkLineHasCursorAddressing(line string) bool {
	for i := 0; i < len(line); i++ {
		if line[i] != '\x1b' || i+1 >= len(line) || line[i+1] != '[' {
			continue
		}
		j := i + 2
		for ; j < len(line); j++ {
			b := line[j]
			if b < 0x40 || b > 0x7e {
				continue
			}
			switch b {
			case 'G', 'H', 'f', 'C', 'D', 'X':
				return true
			}
			i = j
			break
		}
	}
	return false
}

func writePresentedRowLine(builder *strings.Builder, line string, width int) {
	displayWidth := frameSinkLineWidth(line)
	if width <= displayWidth {
		builder.WriteString(line)
		return
	}
	builder.WriteString(line)
	builder.WriteString(render.ANSIReset)
	builder.WriteString(strings.Repeat(" ", width-displayWidth))
}

func rowWidth(frameWidth int, next string, previous string) int {
	width := frameWidth
	if nextWidth := frameSinkLineWidth(next); width < nextWidth {
		width = nextWidth
	}
	if previousWidth := frameSinkLineWidth(previous); width < previousWidth {
		width = previousWidth
	}
	return width
}

func frameSinkLineWidth(line string) int {
	for i := 0; i < len(line); i++ {
		b := line[i]
		if b == '\x1b' || b < 0x20 || b >= 0x7f {
			return xansi.StringWidth(line)
		}
	}
	return len(line)
}

func (sink *FrameSink) writePatchFrame(frame render.Frame) error {
	patch := *frame.Patch
	if patch.CursorOnly {
		return sink.writeCursorOnlyPatchFrame(frame)
	}
	if patch.Rewrite {
		if patch.Rect.H <= 0 || patch.Rect.W <= 0 || patch.LineWidth <= 0 {
			return nil
		}
		return sink.writeRewritePatchFrame(frame, patch)
	}
	if patch.Rect.H <= 1 || patch.Rect.W <= 0 || patch.LineWidth <= 0 {
		return nil
	}
	lines := patch.LinesANSI
	if len(lines) == 0 && patch.LineANSI != "" {
		lines = []string{patch.LineANSI}
	}
	if len(lines) == 0 || len(lines) >= patch.Rect.H {
		return nil
	}
	var builder strings.Builder
	builder.WriteString(synchronizedOutputBegin)
	builder.WriteString(hideCursor)
	builder.WriteString(scrollRegion(patch.Rect.Y+1, patch.Rect.Y+patch.Rect.H))
	scrolls := len(lines)
	switch patch.Dir {
	case render.FramePatchScrollUp:
		builder.WriteString(cursorPosition(patch.Rect.Y+patch.Rect.H, 1))
		for i := 0; i < scrolls; i++ {
			builder.WriteString(scrollUpOne)
		}
	case render.FramePatchScrollDown:
		builder.WriteString(cursorPosition(patch.Rect.Y+1, 1))
		for i := 0; i < scrolls; i++ {
			builder.WriteString(scrollDownOne)
		}
	default:
		return nil
	}
	builder.WriteString(resetScrollRegion)
	for i, line := range lines {
		lineY := patch.LineY
		if patch.Dir == render.FramePatchScrollUp {
			lineY += i
		} else {
			lineY -= len(lines) - 1 - i
		}
		builder.WriteString(cursorPosition(lineY+1, patch.LineX+1))
		builder.WriteString(render.ANSIReset)
		builder.WriteString(eraseChars(patch.LineWidth))
		builder.WriteString(line)
		builder.WriteString(render.ANSIReset)
	}
	builder.WriteString(sink.frameCursorSequence(frame))
	builder.WriteString(synchronizedOutputEnd)
	_, err := io.WriteString(sink.writer, builder.String())
	if err != nil {
		return err
	}
	// 增量 patch 不重建完整 lastLines；下一次完整帧自然会全量校准。
	sink.hasLastFrame = false
	sink.lastCursor = sink.frameCursorSequence(frame)
	sink.rememberVisibleCursor(frame)
	return nil
}

func (sink *FrameSink) writeCursorOnlyPatchFrame(frame render.Frame) error {
	cursorSequence := sink.frameCursorSequence(frame)
	if sink.lastCursor == cursorSequence {
		return nil
	}
	var builder strings.Builder
	builder.WriteString(synchronizedOutputBegin)
	builder.WriteString(cursorSequence)
	builder.WriteString(synchronizedOutputEnd)
	_, err := io.WriteString(sink.writer, builder.String())
	if err != nil {
		return err
	}
	sink.lastCursor = cursorSequence
	sink.rememberVisibleCursor(frame)
	return nil
}

func (sink *FrameSink) writeRewritePatchFrame(frame render.Frame, patch render.FramePatch) error {
	lines := patch.LinesANSI
	if len(lines) == 0 && patch.LineANSI != "" {
		lines = []string{patch.LineANSI}
	}
	if len(lines) == 0 {
		return nil
	}
	if len(lines) > patch.Rect.H {
		lines = lines[:patch.Rect.H]
	}
	var builder strings.Builder
	builder.WriteString(synchronizedOutputBegin)
	builder.WriteString(hideCursor)
	for i, line := range lines {
		lineY := patch.LineY + i
		if lineY < patch.Rect.Y || lineY >= patch.Rect.Y+patch.Rect.H {
			continue
		}
		builder.WriteString(cursorPosition(lineY+1, patch.LineX+1))
		builder.WriteString(render.ANSIReset)
		builder.WriteString(eraseChars(patch.LineWidth))
		builder.WriteString(line)
		builder.WriteString(render.ANSIReset)
	}
	builder.WriteString(sink.frameCursorSequence(frame))
	builder.WriteString(synchronizedOutputEnd)
	_, err := io.WriteString(sink.writer, builder.String())
	if err != nil {
		return err
	}
	// 矩形重写只校准了内容区；完整帧缓存失效，避免后续 diff 误判屏幕状态。
	sink.hasLastFrame = false
	sink.lastCursor = sink.frameCursorSequence(frame)
	sink.rememberVisibleCursor(frame)
	return nil
}

func (sink *FrameSink) frameCursorSequence(frame render.Frame) string {
	if !frame.Cursor.Visible && frame.Cursor.Anchor && sink.hasLastVisibleCursor {
		frame.Cursor = render.Cursor{Anchor: true, Shape: sink.lastVisibleCursor.Shape}
		frame.CursorRect = sink.lastVisibleCursorRect
	}
	return frameCursorSequence(frame)
}

func (sink *FrameSink) rememberVisibleCursor(frame render.Frame) {
	if frame.Cursor.Visible && frame.CursorRect.W > 0 && frame.CursorRect.H > 0 {
		sink.lastVisibleCursor = frame.Cursor
		sink.lastVisibleCursorRect = frame.CursorRect
		sink.hasLastVisibleCursor = true
		return
	}
	if !frame.Cursor.Anchor {
		sink.hasLastVisibleCursor = false
	}
}

func shiftedUpWindow(previous []string, next []string) (int, int, bool) {
	start := 0
	for start < len(previous) && previous[start] == next[start] {
		start++
	}
	end := len(previous) - 1
	for end >= start && previous[end] == next[end] {
		end--
	}
	if end-start < 2 {
		return 0, 0, false
	}
	for i := start; i < end; i++ {
		if previous[i+1] != next[i] {
			return 0, 0, false
		}
	}
	return start, end, true
}

func shiftedDownWindow(previous []string, next []string) (int, int, bool) {
	start := 0
	for start < len(previous) && previous[start] == next[start] {
		start++
	}
	end := len(previous) - 1
	for end >= start && previous[end] == next[end] {
		end--
	}
	if end-start < 2 {
		return 0, 0, false
	}
	for i := start + 1; i <= end; i++ {
		if previous[i-1] != next[i] {
			return 0, 0, false
		}
	}
	return start, end, true
}

func frameCursorSequence(frame render.Frame) string {
	if (!frame.Cursor.Visible && !frame.Cursor.Anchor) || frame.CursorRect.W <= 0 || frame.CursorRect.H <= 0 {
		return hideCursor
	}
	// 中文说明：参考 tuiv2 的最终光标复投经验。v3 已经有全局 CursorRect，
	// 因此只在真实 virtual cursor 可见时显示宿主光标；anchor-only 仍隐藏但停靠。
	sequence := cursorShapeSequence(frame.Cursor.Shape) + cursorPosition(frame.CursorRect.Y+1, frame.CursorRect.X+1)
	if frame.Cursor.Visible {
		return sequence + showCursor
	}
	return hideCursor + sequence
}

func cursorShapeSequence(shape render.CursorShape) string {
	switch shape {
	case render.CursorShapeBar:
		return cursorShapeBar
	default:
		return cursorShapeBlock
	}
}

func cursorPosition(row int, col int) string {
	if row < 1 {
		row = 1
	}
	if col < 1 {
		col = 1
	}
	return fmt.Sprintf("\x1b[%d;%dH", row, col)
}

func cursorPositionCapacity(row int, col int) int {
	if row < 1 {
		row = 1
	}
	if col < 1 {
		col = 1
	}
	return 4 + decimalDigits(row) + decimalDigits(col)
}

func scrollRegion(top int, bottom int) string {
	if top < 1 {
		top = 1
	}
	if bottom < top {
		bottom = top
	}
	return fmt.Sprintf("\x1b[%d;%dr", top, bottom)
}

func eraseChars(count int) string {
	if count <= 0 {
		return ""
	}
	return fmt.Sprintf("\x1b[%dX", count)
}

func decimalDigits(value int) int {
	if value < 0 {
		value = -value
	}
	digits := 1
	for value >= 10 {
		value /= 10
		digits++
	}
	return digits
}

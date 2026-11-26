package endpoint

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/anytty/anytty/proto/access/apipb"
)

// screenDeltaRenderer 把 LiveScreenNext 的行级 delta 渲染成 ANSI 字节，喂给
// 本地终端解析器——与 renderScreenSnapshot 用的是同一个通道。
//
// 它缓存每一行已经画过的 ANSI，因此 row_copies（滚动）可以在本地重放，不需要
// 从网络重发内容。这样 TUI 与手机端走同一条「服务端 vterm + 行级差异」路径，
// 每次只传真正变化的行。
type screenDeltaRenderer struct {
	cols, rows int
	rowANSI    []string
	altScreen  bool
	cursorPos  string
	cursorVis  *bool
}

func newScreenDeltaRenderer(seed *apipb.NativeScreenResult) *screenDeltaRenderer {
	renderer := &screenDeltaRenderer{}
	renderer.reset(seed)
	return renderer
}

// reset 用一张 full_replace 快照重建行缓存。
func (renderer *screenDeltaRenderer) reset(screen *apipb.NativeScreenResult) {
	renderer.cols = int(screen.GetSize().GetCols())
	renderer.rows = int(screen.GetSize().GetRows())
	if renderer.cols < 0 {
		renderer.cols = 0
	}
	if renderer.rows < 0 {
		renderer.rows = 0
	}
	renderer.rowANSI = make([]string, renderer.rows)
	for _, replacement := range computeRows(screen) {
		if replacement == nil {
			continue
		}
		index := int(replacement.GetRowIndex())
		if index < 0 || index >= renderer.rows {
			continue
		}
		renderer.rowANSI[index] = renderRowANSI(replacement, renderer.cols)
	}
	renderer.altScreen = screen.GetAlternateScreen()
	renderer.cursorPos = ""
	renderer.cursorVis = nil
}

// render 返回本次 delta 需要喂给终端解析器的 ANSI。首帧/尺寸变化/alt-screen
// 切换/基线对不上时退回整屏重绘。
func (renderer *screenDeltaRenderer) render(screen *apipb.NativeScreenResult) []byte {
	if screen == nil {
		return nil
	}
	cols, rows := int(screen.GetSize().GetCols()), int(screen.GetSize().GetRows())
	if screen.GetFullReplace() || cols != renderer.cols || rows != renderer.rows || screen.GetAlternateScreen() != renderer.altScreen {
		renderer.reset(screen)
		return renderScreenSnapshot(screen)
	}

	var b strings.Builder
	next := append([]string(nil), renderer.rowANSI...)
	// 同步输出：本地解析器不会看到半帧。
	b.WriteString("\x1b[?2026h")

	for _, copyRow := range screen.GetRowCopies() {
		count := int(copyRow.GetCount())
		source := int(copyRow.GetSourceRow())
		destination := int(copyRow.GetDestinationRow())
		if count <= 0 || source < 0 || destination < 0 || source+count > renderer.rows || destination+count > renderer.rows {
			continue
		}
		for offset := 0; offset < count; offset++ {
			sourceRow, destinationRow := source+offset, destination+offset
			line := renderer.rowANSI[sourceRow]
			next[destinationRow] = line
			b.WriteString(fmt.Sprintf("\x1b[%d;1H\x1b[0m", destinationRow+1))
			b.WriteString(line)
			b.WriteString("\x1b[0m\x1b[K")
		}
	}
	for _, replacement := range screen.GetRowReplacements() {
		index := int(replacement.GetRowIndex())
		if index < 0 || index >= renderer.rows {
			continue
		}
		line := renderRowANSI(replacement, renderer.cols)
		next[index] = line
		b.WriteString(fmt.Sprintf("\x1b[%d;1H\x1b[0m", index+1))
		b.WriteString(line)
		b.WriteString("\x1b[0m\x1b[K")
	}
	renderer.rowANSI = next

	if cursor := screen.GetCursor(); cursor != nil {
		position := fmt.Sprintf("\x1b[%d;%dH", cursor.GetRow()+1, cursor.GetCol()+1)
		visible := cursor.GetVisible()
		if position != renderer.cursorPos || renderer.cursorVis == nil || *renderer.cursorVis != visible {
			b.WriteString(position)
			if visible {
				b.WriteString("\x1b[?25h")
			} else {
				b.WriteString("\x1b[?25l")
			}
			renderer.cursorPos = position
			renderer.cursorVis = &visible
		}
	}
	b.WriteString("\x1b[?2026l")
	return []byte(b.String())
}

// renderRowANSI 与 renderScreenSnapshot 的行渲染保持一致：cells + tail fill。
func renderRowANSI(row *apipb.ScreenRowReplace, cols int) string {
	var b strings.Builder
	used := 0
	for _, cell := range row.GetRow().GetCells() {
		writeCell(&b, cell)
		width := int(cell.GetWidth())
		if width <= 0 {
			width = 1
		}
		used += width
	}
	if tail := row.GetRow().GetTailFill(); tail != nil && used < cols {
		b.WriteString(sgrForStyle(tail))
		b.WriteString(strings.Repeat(" ", cols-used))
	}
	return b.String()
}

// screenDeltaStarter 是 sessionConn 的可选扩展：能直接消费行级 delta 的会话
// 实现它。没有实现（或环境变量没开）时继续走原始 PTY 字节流。
type screenDeltaStarter interface {
	StartScreenDeltas(ctx context.Context, att *attachment, seed *apipb.NativeScreenResult) error
}

// tuiScreenTraceEnabled gates per-second screen-delta byte accounting.
func tuiScreenTraceEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ANYTTY_TUI_TRACE"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// screenDeltasEnabled 默认开启：TUI 走「服务端 vterm + 行级 delta」，只传变化
// 的行；设置 ANYTTY_TUI_SCREEN_DELTAS=0 可退回原始 PTY 字节流。
func screenDeltasEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ANYTTY_TUI_SCREEN_DELTAS"))) {
	case "0", "false", "no", "off":
		return false
	}
	return true
}

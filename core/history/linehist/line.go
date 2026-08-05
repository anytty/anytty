// Package linehist 是重做后的 logical-line 无限历史 domain（LL 系列）。
//
// 单一真值：vterm emulator 是唯一屏幕真值。本包不维护第二份屏幕模型、
// 不做 classifier/frame reducer/文本去重对账；它只消费 vterm 事务里
// ungated 的 EvictedRows（真正离开 primary 可见区的物理行，seal-on-eviction
// 后不可变），按软换行标志拼成宽度无关的 logical line，append 落盘，
// copy/history 查询按 logical-line window 返回 source rows；visual reflow
// 由 TUI 本地完成。
//
// 消息链路：PTY bytes -> SemanticTap -> TerminalSemanticTransaction.EvictedRows
// -> Assembler 拼 logical line -> CompressedLineFile 紧凑块文件。
// 失败条件：从 current frame 反推历史、把文本相等当正确性依据、
// 在本包内出现第二份屏幕状态机。
package linehist

import (
	"strings"
	"time"

	"github.com/anytty/anytty/core/history"
	vterm "github.com/anytty/anytty/vterm/vterm"
)

// Run 是 logical line 内一段样式一致的文本。Style 不含 link；link 单独放在
// Run 字段上，与 history.Cell 的拆分约定一致。
type Run struct {
	Text       string
	Style      history.CellStyle
	LinkURL    string
	LinkParams string
}

// Line 是存储与分页的最小记录单位。HardEnd=true 表示硬换行结束的完整
// logical line；HardEnd=false 表示病态超长行的 chunk 截断，投影时与后继
// 记录连续拼接。Line 宽度无关：不存 fixed grid / ScreenCols；已落盘
// 历史的 resize 重排只影响 TUI 本地 visual reflow，不改写 cold truth。
type Line struct {
	Runs      []Run
	HardEnd   bool
	UpdatedAt time.Time
}

// LineText 返回 logical line 的 plain text 派生值。它只用于测试与诊断，
// 不能反过来成为渲染或去重 truth。
func LineText(line Line) string {
	var out strings.Builder
	for _, run := range line.Runs {
		out.WriteString(run.Text)
	}
	return out.String()
}

// runsFromScrollOut 把一条滚出物理行的 payload 规整成 Run 序列。
// Runs 是 vterm 保留的 styled text 快路径；Cells 是逐 cell payload，
// 宽字符 continuation（width<=0 且无文本）必须跳过，纯占位 blank
// （无文本但 width>0）展开成空格保持列宽语义。
func runsFromScrollOut(row vterm.TerminalSemanticScrollOut) []Run {
	if len(row.Runs) > 0 {
		out := make([]Run, 0, len(row.Runs))
		for _, run := range row.Runs {
			if run.Text == "" {
				continue
			}
			out = appendMergedRun(out, Run{
				Text:       run.Text,
				Style:      historyStyleFromVTerm(run.Style),
				LinkURL:    run.Style.LinkURL,
				LinkParams: run.Style.LinkParams,
			})
		}
		return out
	}
	var out []Run
	for _, cell := range row.Cells {
		text := cell.Content
		if text == "" {
			if cell.Width <= 0 {
				// 宽字符 continuation 占位，不产出内容。
				continue
			}
			text = strings.Repeat(" ", cell.Width)
		}
		linkURL := cell.LinkURL
		linkParams := cell.LinkParams
		if linkURL == "" && linkParams == "" {
			linkURL = cell.Style.LinkURL
			linkParams = cell.Style.LinkParams
		}
		out = appendMergedRun(out, Run{
			Text:       text,
			Style:      historyStyleFromVTerm(cell.Style),
			LinkURL:    linkURL,
			LinkParams: linkParams,
		})
	}
	return out
}

func appendMergedRun(runs []Run, next Run) []Run {
	if next.Text == "" {
		return runs
	}
	if len(runs) > 0 {
		last := &runs[len(runs)-1]
		if last.Style == next.Style && last.LinkURL == next.LinkURL && last.LinkParams == next.LinkParams {
			last.Text += next.Text
			return runs
		}
	}
	return append(runs, next)
}

// historyStyleFromVTerm 按字段复制 style，link 不进 style（放 Run 字段），
// 与 history.Cell/CellRun 的既有拆分保持一致。
func historyStyleFromVTerm(style vterm.CellStyle) history.CellStyle {
	return history.CellStyle{
		FG:            style.FG,
		BG:            style.BG,
		Bold:          style.Bold,
		Italic:        style.Italic,
		Underline:     style.Underline,
		Blink:         style.Blink,
		Reverse:       style.Reverse,
		Strikethrough: style.Strikethrough,
	}
}

// trimTrailingPaddingRuns 去掉硬结束行尾部的无样式空格 padding
// （terminal 行尾填充不属于 logical text）。带样式或带 link 的尾部空白
// 是内容（如反色光标块、背景色条），必须保留。
func trimTrailingPaddingRuns(runs []Run) []Run {
	for len(runs) > 0 {
		last := &runs[len(runs)-1]
		if last.Style != (history.CellStyle{}) || last.LinkURL != "" || last.LinkParams != "" {
			break
		}
		trimmed := strings.TrimRight(last.Text, " ")
		if trimmed != "" {
			last.Text = trimmed
			break
		}
		runs = runs[:len(runs)-1]
	}
	if len(runs) == 0 {
		return nil
	}
	return runs
}

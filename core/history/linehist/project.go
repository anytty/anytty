package linehist

import (
	"strings"
	"time"

	"github.com/anytty/anytty/core/history"
	vterm "github.com/anytty/anytty/vterm/vterm"
	xansi "github.com/charmbracelet/x/ansi"
)

// 本文件是 linehist 的查询投影层：把宽度无关的 logical line 展开成
// history.HistoryRow source row，并提供 window/cursor/boundary 的领域帮助函数。
// visual reflow 属于 TUI 本地窗口，core 不再构建全局 visual row 坐标。
// 旧 history 包的同名 helper 是未导出实现；R433 重做不修改旧包，这里按
// 相同对外语义（TUI/protocol 兼容）独立实现。

// cellsFromRuns 把 styled runs 展开成逐 grapheme 的 history cells。
// 宽度由 grapheme cluster 派生（与 vterm 写屏时的 GraphemeWidth 一致），
// 投影换行必须按 cell 宽度进行，不能按字节数。
func cellsFromRuns(runs []Run) []history.Cell {
	var out []history.Cell
	for _, run := range runs {
		text := run.Text
		for text != "" {
			cluster, width := xansi.FirstGraphemeCluster(text, xansi.GraphemeWidth)
			if cluster == "" {
				break
			}
			out = append(out, history.Cell{
				Text:       cluster,
				Width:      width,
				Style:      run.Style,
				LinkURL:    run.LinkURL,
				LinkParams: run.LinkParams,
			})
			text = text[len(cluster):]
		}
	}
	return out
}

// cellsFromVTermCells 把 emulator 当前屏的一行 cells 规整成 history cells。
// 宽字符 continuation 占位（无文本且 width<=0）跳过；纯 blank 占位展开成
// 空格，保持列宽语义，与 runsFromScrollOut 的落盘规整一致。
func cellsFromVTermCells(cells []vterm.Cell) []history.Cell {
	var out []history.Cell
	for _, cell := range cells {
		text := cell.Content
		width := cell.Width
		if text == "" {
			if width <= 0 {
				continue
			}
			text = strings.Repeat(" ", width)
		}
		if width <= 0 {
			width = 0
		}
		linkURL := cell.LinkURL
		linkParams := cell.LinkParams
		if linkURL == "" && linkParams == "" {
			linkURL = cell.Style.LinkURL
			linkParams = cell.Style.LinkParams
		}
		out = append(out, history.Cell{
			Text:       text,
			Width:      width,
			Style:      historyStyleFromVTerm(cell.Style),
			LinkURL:    linkURL,
			LinkParams: linkParams,
		})
	}
	return out
}

func normalizedLimit(limit int) (int, error) {
	if limit < 1 || limit > history.MaxHistoryWindowLines {
		return 0, history.ErrHistoryWindowLimit
	}
	return limit, nil
}

// historyRowBudgetBytes conservatively tracks the coalesced protobuf shape
// without importing transport or generated API types into the history store.
func historyRowBudgetBytes(row history.HistoryRow) int {
	size := 96
	for index, cell := range row.Cells {
		if index > 0 && historyCellsShareBudgetRun(row.Cells[index-1], cell) {
			size += len(cell.Text)
			continue
		}
		size += 24 + len(cell.Text) + len(cell.Style.FG) + len(cell.Style.BG) + len(cell.LinkURL) + len(cell.LinkParams)
	}
	return size
}

func historyLineBudgetBytes(line Line) int {
	size := 96
	var previous history.Cell
	havePrevious := false
	for _, run := range line.Runs {
		text := run.Text
		for text != "" {
			cluster, width := xansi.FirstGraphemeCluster(text, xansi.GraphemeWidth)
			if cluster == "" {
				break
			}
			cell := history.Cell{Width: width, Style: run.Style, LinkURL: run.LinkURL, LinkParams: run.LinkParams}
			if !havePrevious || !historyCellsShareBudgetRun(previous, cell) {
				size += 24 + len(run.Style.FG) + len(run.Style.BG) + len(run.LinkURL) + len(run.LinkParams)
			}
			size += len(cluster)
			previous = cell
			havePrevious = true
			if size > history.MaxHistoryWindowBytes {
				return size
			}
			text = text[len(cluster):]
		}
	}
	return size
}

func historyCellsShareBudgetRun(left history.Cell, right history.Cell) bool {
	return left.Width == right.Width && left.Style == right.Style && left.LinkURL == right.LinkURL && left.LinkParams == right.LinkParams
}

func cloneRows(rows []history.HistoryRow) []history.HistoryRow {
	if len(rows) == 0 {
		return nil
	}
	out := make([]history.HistoryRow, len(rows))
	for i, row := range rows {
		out[i] = cloneRow(row)
	}
	return out
}

func cloneRow(row history.HistoryRow) history.HistoryRow {
	result := row
	result.Cells = append([]history.Cell(nil), row.Cells...)
	return result
}

func boundaryForRows(rows []history.HistoryRow, generation history.Generation, token history.HistoryToken) history.HistoryBoundary {
	if len(rows) == 0 {
		return history.HistoryBoundary{Cursor: history.HistoryCursor{Generation: generation, Token: token}}
	}
	return history.HistoryBoundary{
		FirstLineID: rows[0].LineID,
		LastLineID:  rows[len(rows)-1].LineID,
		Cursor: history.HistoryCursor{
			Segment:    rows[0].Segment,
			SessionID:  rows[0].SessionID,
			FrameID:    rows[0].FrameID,
			LineID:     rows[0].LineID,
			Generation: generation,
			Token:      token,
			Valid:      true,
		},
	}
}

func cursorBeforeLine(row history.HistoryRow, generation history.Generation, token history.HistoryToken, valid bool) history.HistoryCursor {
	return history.HistoryCursor{
		Segment:    row.Segment,
		SessionID:  row.SessionID,
		FrameID:    row.FrameID,
		LineID:     row.LineID,
		RowInLine:  row.RowInLine,
		Generation: generation,
		Token:      token,
		Valid:      valid,
	}
}

func spansForRows(rows []history.HistoryRow) []history.HistoryLineSpan {
	spans := make([]history.HistoryLineSpan, 0, len(rows))
	for i, row := range rows {
		spans = append(spans, history.HistoryLineSpan{
			StartRow:       i,
			EndRow:         i,
			Kind:           row.Kind,
			Segment:        row.Segment,
			LogicalLineID:  row.LineID,
			SessionID:      row.SessionID,
			FrameID:        row.FrameID,
			TimestampStart: row.Timestamp,
			TimestampEnd:   row.Timestamp,
			ScreenRow:      row.ScreenRow,
			ScreenRowSet:   row.ScreenRowSet,
		})
	}
	return spans
}

func rowText(cells []history.Cell) string {
	var out strings.Builder
	for _, cell := range cells {
		out.WriteString(cell.Text)
	}
	return out.String()
}

func buildWindow(req history.HistoryWindowRequest, rows []history.HistoryRow, total int, op history.HistoryWindowOp, boundary history.HistoryBoundary, generation history.Generation, hasMore bool) history.HistoryWindow {
	if total < len(rows) {
		total = len(rows)
	}
	return history.HistoryWindow{
		TerminalID:   req.TerminalID,
		Token:        req.Token,
		Op:           op,
		Cols:         req.Cols,
		Rows:         cloneRows(rows),
		Lines:        spansForRows(rows),
		Generation:   generation,
		Boundary:     boundary,
		HasMore:      hasMore,
		LogicalTotal: total,
		Timestamp:    time.Now().UTC(),
	}
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

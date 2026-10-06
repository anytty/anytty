package history

import (
	"strings"

	"github.com/anytty/anytty/proto/access/apipb"
	xansi "github.com/charmbracelet/x/ansi"
)

// The provider sends width-independent source rows and coalesced cell runs.
// Reflow is terminal-owned. Split at grapheme boundaries and keep wide glyphs
// intact; fixed-grid frames retain their physical rows.
func project(rows []*apipb.HistoryRow, cols int) []*apipb.HistoryRow {
	var out []*apipb.HistoryRow
	for _, source := range rows {
		var cells []*apipb.ScreenCell
		width, index := 0, source.GetRowInLine()
		flush := func(wrapped bool) {
			out = append(out, &apipb.HistoryRow{
				Row:           &apipb.ScreenRow{Cells: cells, Wrapped: wrapped, TailFill: source.GetRow().GetTailFill()},
				LogicalLineId: source.GetLogicalLineId(), RowInLine: index, Segment: source.GetSegment(),
				SessionId: source.GetSessionId(), FrameId: source.GetFrameId(), FixedGrid: source.GetFixedGrid(),
				ScreenCols: source.GetScreenCols(), ScreenRows: source.GetScreenRows(), ScreenRowSet: source.GetScreenRowSet(),
				Ownership: source.GetOwnership(), TimestampUnixNano: source.GetTimestampUnixNano(), RowKind: source.GetRowKind(), Wrapped: wrapped,
			})
			cells, width = nil, 0
			index++
		}
		for _, run := range source.GetRow().GetCells() {
			text := run.GetContent()
			if text == "" && run.GetWidth() > 0 {
				text = strings.Repeat(" ", int(run.GetWidth()))
			}
			for text != "" {
				cluster, cellWidth := xansi.FirstGraphemeCluster(text, xansi.GraphemeWidth)
				if cluster == "" {
					break
				}
				if !source.GetFixedGrid() && width > 0 && width+cellWidth > cols {
					flush(true)
				}
				cells = append(cells, &apipb.ScreenCell{Content: cluster, Width: int32(cellWidth), Style: run.GetStyle(), LinkUrl: run.GetLinkUrl(), LinkParams: run.GetLinkParams()})
				width += cellWidth
				text = text[len(cluster):]
			}
		}
		flush(source.GetWrapped())
	}
	return out
}

package linehist

import (
	"context"
	"errors"
	"strings"

	"github.com/anytty/anytty/core/history"
	xansi "github.com/charmbracelet/x/ansi"
)

const historySearchReadBatchLines = 256

func (store *Store) Search(ctx context.Context, req history.HistorySearchRequest) (history.HistorySearchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if req.Token == "" || req.Query == "" {
		return history.HistorySearchResult{}, history.ErrHistoryInvalidMutation
	}
	if req.Direction == "" {
		req.Direction = history.HistorySearchForward
	}
	if req.Direction != history.HistorySearchForward && req.Direction != history.HistorySearchBackward {
		return history.HistorySearchResult{}, history.ErrHistoryInvalidMutation
	}
	pattern, err := history.CompileHistorySearchPattern(req.Mode, req.Query)
	if err != nil {
		return history.HistorySearchResult{}, err
	}
	limit, err := normalizedLimit(req.Limit)
	if err != nil {
		return history.HistorySearchResult{}, err
	}
	if req.ContextBefore < 0 || req.ContextBefore >= limit {
		return history.HistorySearchResult{}, history.ErrHistoryInvalidMutation
	}
	windowReq, view, err := store.viewForRequest(history.HistoryWindowRequest{
		TerminalID: req.TerminalID,
		Token:      req.Token,
		Cols:       req.Cols,
		Limit:      limit,
	})
	if err != nil {
		return history.HistorySearchResult{}, err
	}
	total := viewLogicalTotal(view)
	if total == 0 {
		return history.HistorySearchResult{}, nil
	}
	startLine := 0
	startCol := maxInt(0, req.Start.Col)
	if req.Start.LineID != 0 {
		startLine = clampInt(int(req.Start.LineID)-1, 0, total-1)
	} else if req.Direction == history.HistorySearchBackward {
		startLine = total - 1
		startCol = int(^uint(0) >> 1)
	} else if len(view.hot) > 0 {
		// A first search has no client cursor. Start at the frozen in-memory
		// frontier so pending/open-tail/current-screen content does not wait
		// behind a complete cold-file scan. Forward search still wraps through
		// the complete frozen snapshot when the frontier has no match.
		startLine = view.coldCount
	}

	match, found, wrapped, err := store.searchFrozenView(ctx, view, pattern, req.Direction, startLine, startCol)
	if err != nil {
		if errors.Is(err, errRetentionChanged) {
			return history.HistorySearchResult{}, history.ErrHistoryStaleWindow
		}
		return history.HistorySearchResult{}, err
	}
	if !found {
		return history.HistorySearchResult{}, nil
	}
	matchOffset := int(match.Start.LineID) - 1
	pageStart := maxInt(0, matchOffset-req.ContextBefore)
	pageEnd := minInt(total, pageStart+limit)
	rows, _, err := store.windowRowsForward(view, pageStart, pageEnd)
	if err != nil {
		return history.HistorySearchResult{}, err
	}
	boundary := boundaryForRows(rows, view.generation, req.Token)
	if len(rows) > 0 {
		boundary.Cursor = cursorBeforeLine(rows[0], view.generation, req.Token, pageStart > 0)
		boundary.LastLineID = history.LogicalLineID(total)
	}
	window := buildWindow(windowReq, rows, total, history.HistoryWindowReplace, boundary, view.generation, pageStart > 0)
	window.ViewportAnchor = view.anchor
	return history.HistorySearchResult{Found: true, Match: match, Window: window, Wrapped: wrapped}, nil
}

func (store *Store) searchFrozenView(ctx context.Context, view liveView, pattern *history.HistorySearchPattern, direction history.HistorySearchDirection, startLine int, startCol int) (history.HistoryCopyRange, bool, bool, error) {
	total := viewLogicalTotal(view)
	if direction == history.HistorySearchForward {
		if match, ok, err := store.searchLineRange(ctx, view, pattern, direction, startLine, total, startLine, startCol, -1); err != nil || ok {
			return match, ok, false, err
		}
		if startCol == 0 {
			match, ok, err := store.searchLineRange(ctx, view, pattern, direction, 0, startLine, -1, 0, -1)
			return match, ok, ok, err
		}
		match, ok, err := store.searchLineRange(ctx, view, pattern, direction, 0, startLine+1, startLine, 0, startCol-1)
		return match, ok, ok, err
	}
	if startCol == 0 {
		if match, ok, err := store.searchLineRange(ctx, view, pattern, direction, 0, startLine, -1, 0, -1); err != nil || ok {
			return match, ok, false, err
		}
		match, ok, err := store.searchLineRange(ctx, view, pattern, direction, startLine, total, startLine, startCol, -1)
		return match, ok, ok, err
	}
	if match, ok, err := store.searchLineRange(ctx, view, pattern, direction, 0, startLine+1, startLine, 0, startCol-1); err != nil || ok {
		return match, ok, false, err
	}
	match, ok, err := store.searchLineRange(ctx, view, pattern, direction, startLine, total, startLine, startCol, -1)
	return match, ok, ok, err
}

func (store *Store) searchLineRange(ctx context.Context, view liveView, pattern *history.HistorySearchPattern, direction history.HistorySearchDirection, start int, end int, constrainedLine int, minCol int, maxCol int) (history.HistoryCopyRange, bool, error) {
	reverse := direction == history.HistorySearchBackward
	var result history.HistoryCopyRange
	found := false
	visitText := func(logicalIndex int, text string) bool {
		lineMin, lineMax := 0, -1
		if logicalIndex == constrainedLine {
			lineMin, lineMax = minCol, maxCol
		}
		startMatch, endMatch, ok := findHistoryPatternMatch(text, pattern, reverse, lineMin, lineMax)
		if !ok {
			return true
		}
		lineID := history.LogicalLineID(logicalIndex + 1)
		result = history.HistoryCopyRange{
			Start: history.HistoryCopyPosition{LineID: lineID, Col: startMatch},
			End:   history.HistoryCopyPosition{LineID: lineID, Col: endMatch},
		}
		found = true
		return false
	}
	visitCells := func(logicalIndex int, cells []history.Cell) bool {
		return visitText(logicalIndex, historySearchText(cells))
	}
	if reverse {
		hotStart := maxInt(start, view.coldCount)
		hotEnd := minInt(end, view.coldCount+len(view.hot))
		for index := hotEnd - 1; index >= hotStart && !found; index-- {
			if err := ctx.Err(); err != nil {
				return result, false, err
			}
			visitCells(index, view.hot[index-view.coldCount].Cells)
		}
		if !found && start < view.coldCount {
			coldEnd := minInt(end, view.coldCount)
			err := store.engine.VisitSearchLinesAtRetention(ctx, view.retention, view.coldBase+start, view.coldBase+coldEnd, true, pattern.RequiredLiterals(), func(index int, line Line) bool {
				return visitText(index-view.coldBase, LineText(line))
			})
			if err != nil {
				return result, false, err
			}
		}
		return result, found, nil
	}
	if start < view.coldCount {
		coldEnd := minInt(end, view.coldCount)
		err := store.engine.VisitSearchLinesAtRetention(ctx, view.retention, view.coldBase+start, view.coldBase+coldEnd, false, pattern.RequiredLiterals(), func(index int, line Line) bool {
			return visitText(index-view.coldBase, LineText(line))
		})
		if err != nil {
			return result, false, err
		}
	}
	if !found {
		hotStart := maxInt(start, view.coldCount)
		hotEnd := minInt(end, view.coldCount+len(view.hot))
		for index := hotStart; index < hotEnd && !found; index++ {
			if err := ctx.Err(); err != nil {
				return result, false, err
			}
			visitCells(index, view.hot[index-view.coldCount].Cells)
		}
	}
	return result, found, nil
}

func findHistoryLineMatch(cells []history.Cell, query string, reverse bool, minCol int, maxCol int) (int, int, bool) {
	return findHistoryTextMatch(historySearchText(cells), query, reverse, minCol, maxCol)
}

func findHistoryTextMatch(text string, query string, reverse bool, minCol int, maxCol int) (int, int, bool) {
	pattern, err := history.CompileHistorySearchPattern(history.HistorySearchModeText, query)
	if err != nil {
		return 0, 0, false
	}
	return findHistoryPatternMatch(text, pattern, reverse, minCol, maxCol)
}

func findHistoryPatternMatch(text string, pattern *history.HistorySearchPattern, reverse bool, minCol int, maxCol int) (int, int, bool) {
	if text == "" || pattern == nil {
		return 0, 0, false
	}
	matchStart, matchEnd := 0, 0
	found := false
	for _, match := range pattern.FindAllStringIndex(text) {
		startCol := xansi.StringWidth(text[:match.Start])
		if startCol >= minCol && (maxCol < 0 || startCol <= maxCol) {
			matchStart = startCol
			matchEnd = xansi.StringWidth(text[:match.End])
			found = true
			if !reverse {
				break
			}
		}
	}
	return matchStart, matchEnd, found
}

func historySearchText(cells []history.Cell) string {
	var text strings.Builder
	for _, cell := range cells {
		text.WriteString(cell.Text)
		padding := maxInt(0, cell.Width-xansi.StringWidth(cell.Text))
		if padding > 0 {
			text.WriteString(strings.Repeat(" ", padding))
		}
	}
	return text.String()
}

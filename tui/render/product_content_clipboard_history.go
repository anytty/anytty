package render

import (
	"fmt"
	"strings"

	"github.com/anytty/anytty/tui/state"
)

const clipboardHistoryPreviewWidth = 200
const clipboardHistoryBodyRows = 10
const clipboardHistoryMaxBodyRows = 20

// Clipboard History overlay 只消费 reducer-owned clipboard 历史快照。
func buildClipboardHistoryContent(root state.Root, shell state.ShellStore) ContentVM {
	shell = shell.ReadonlyDefaults()
	query := strings.TrimSpace(shell.Overlay.Query)
	rows := state.ClipboardHistoryItems(root)
	nameWidth := clipboardHistoryNameWidth(shell)
	lines := []Line{clipboardHistorySearchLine(query, nameWidth), clipboardHistoryDividerSpaceLine(nameWidth)}
	rowOffset := len(lines)
	bodyRowLimit := clipboardHistoryBodyRowsForViewport(chromeSafeViewportForShell(root.Viewport, shell))
	selectedIndex := clipboardHistorySelectedIndex(rows)
	listStart := clipboardHistoryListStart(selectedIndex, len(rows), bodyRowLimit)
	selectedItem, selectedOK := clipboardHistorySelectedItem(rows)
	// 中文说明：右侧预览展示当前选中项正文，左侧列表只负责选择入口。
	previewSegments := clipboardHistoryPreviewSegments(selectedItem, selectedOK, query, clipboardHistoryPreviewWidth, bodyRowLimit)
	bodyRows := maxInt(bodyRowLimit, len(previewSegments))
	if len(rows) == 0 {
		previewSegments = []clipboardHistoryPreviewSegment{{Text: "No clipboard entries"}}
		bodyRows = bodyRowLimit
	}
	for row := 0; row < bodyRows; row++ {
		entryIndex := listStart + row
		var entry state.ClipboardHistoryItem
		hasEntry := entryIndex >= 0 && entryIndex < len(rows)
		if hasEntry {
			entry = rows[entryIndex]
		}
		preview := clipboardHistoryPreviewSegment{}
		if row < len(previewSegments) {
			preview = previewSegments[row]
		}
		lines = append(lines, clipboardHistoryBodyLine(entry, hasEntry, preview, nameWidth))
	}
	regions := clipboardHistoryHitRegions(rows, rowOffset, listStart, bodyRows, nameWidth)
	regions = append(regions, clipboardHistoryDividerHitRegion(rowOffset, bodyRows, nameWidth))
	return ContentVM{
		Kind:       ContentClipboardHistory,
		Lines:      lines,
		Meta:       ContentMetaVM{ClipboardNameWidth: nameWidth},
		Status:     clipboardHistoryStatus(len(rows), query),
		Cursor:     Cursor{Visible: true, Row: 0, Col: clipboardHistorySearchCursorCol(query), Shape: CursorShapeBar},
		HitRegions: regions,
		Empty:      len(rows) == 0,
	}
}

type clipboardHistoryPreviewSegment struct {
	Text       string
	MatchIndex []int
}

func clipboardHistoryBodyLine(row state.ClipboardHistoryItem, hasRow bool, preview clipboardHistoryPreviewSegment, nameWidth int) Line {
	cells := clipboardHistoryNameCells(row, hasRow, nameWidth)
	cells = append(cells, styledCell("│", StyleForeground))
	cells = append(cells, clipboardHistoryColumnCells(preview.Text, preview.MatchIndex, StyleForeground, clipboardHistoryPreviewWidth)...)
	return Line{Cells: cells}
}

func clipboardHistoryNameCells(row state.ClipboardHistoryItem, hasRow bool, nameWidth int) []Cell {
	if !hasRow {
		return []Cell{styledCell(strings.Repeat(" ", nameWidth), StylePicker)}
	}
	prefix := "  "
	titleStyle := StylePicker
	if row.Selected {
		prefix = "› "
		titleStyle = StylePickerAccent
	}
	title := strings.TrimSpace(row.Title)
	if title == "" {
		title = clipboardHistoryTitleFromText(row.Text)
	}
	cells := []Cell{styledCell(prefix, titleStyle)}
	cells = append(cells, clipboardHistoryColumnCells(title, row.TitleMatchIndexes, titleStyle, nameWidth-DisplayWidth(prefix))...)
	return cells
}

func clipboardHistorySearchLine(query string, nameWidth int) Line {
	value := query
	style := StylePickerAccent
	if value == "" {
		value = "Search:"
		style = StylePickerMuted
	} else {
		value = "Search: " + value
	}
	return clipboardHistoryPlainLine(value, style, nameWidth)
}

func clipboardHistoryDividerSpaceLine(nameWidth int) Line {
	return clipboardHistoryPlainLine("", StylePicker, nameWidth)
}

func clipboardHistoryColumnCells(value string, matchIndexes []int, baseStyle StyleToken, width int) []Cell {
	if width <= 0 {
		return nil
	}
	value = TruncateCells(value, width)
	cells := clipboardHistoryHighlightedCells(value, matchIndexes, baseStyle)
	if pad := width - DisplayWidth(value); pad > 0 {
		cells = append(cells, styledCell(strings.Repeat(" ", pad), baseStyle))
	}
	return cells
}

func clipboardHistoryHighlightedCells(value string, matchIndexes []int, baseStyle StyleToken) []Cell {
	if value == "" {
		return nil
	}
	if len(matchIndexes) == 0 {
		return []Cell{styledCell(value, baseStyle)}
	}
	matchSet := make(map[int]struct{}, len(matchIndexes))
	for _, index := range matchIndexes {
		matchSet[index] = struct{}{}
	}
	runes := []rune(value)
	cells := make([]Cell, 0, len(runes))
	for index, r := range runes {
		style := baseStyle
		if _, ok := matchSet[index]; ok {
			style = StylePickerMatch
		}
		cells = append(cells, styledCell(string(r), style))
	}
	return cells
}

func clipboardHistoryPlainLine(value string, style StyleToken, nameWidth int) Line {
	width := clipboardHistoryRowWidth(nameWidth)
	value = TruncateCells(value, width)
	cells := []Cell{styledCell(value, style)}
	if pad := width - DisplayWidth(value); pad > 0 {
		cells = append(cells, styledCell(strings.Repeat(" ", pad), StylePicker))
	}
	return Line{Cells: cells}
}

func clipboardHistoryTitleFromText(text string) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if text == "" {
		return "clipboard"
	}
	return text
}

func clipboardHistorySelectedIndex(rows []state.ClipboardHistoryItem) int {
	for index, row := range rows {
		if row.Selected {
			return index
		}
	}
	if len(rows) > 0 {
		return 0
	}
	return -1
}

func clipboardHistorySelectedItem(rows []state.ClipboardHistoryItem) (state.ClipboardHistoryItem, bool) {
	index := clipboardHistorySelectedIndex(rows)
	if index < 0 || index >= len(rows) {
		return state.ClipboardHistoryItem{}, false
	}
	return rows[index], true
}

func clipboardHistoryBodyRowsForViewport(viewport state.ViewportStore) int {
	if !viewport.Valid || viewport.Rows <= 0 {
		return clipboardHistoryBodyRows
	}
	bodyRows := viewport.Rows - clipboardHistoryVerticalMargin(viewport.Rows) - 4
	return clampInt(bodyRows, 4, clipboardHistoryMaxBodyRows)
}

func clipboardHistoryListStart(selected int, total int, visibleRows int) int {
	if selected < 0 || total <= visibleRows {
		return 0
	}
	start := selected - visibleRows/2
	maxStart := maxInt(0, total-visibleRows)
	return clampInt(start, 0, maxStart)
}

func clipboardHistoryPreviewSegments(item state.ClipboardHistoryItem, ok bool, query string, width int, limit int) []clipboardHistoryPreviewSegment {
	if !ok || width <= 0 || limit <= 0 {
		return nil
	}
	text := clipboardHistoryPreviewText(item)
	if text == "" {
		return nil
	}
	var matches []int
	if strings.TrimSpace(query) != "" {
		matches = state.TerminalPickerQueryMatchIndexes(text, query)
	}
	return clipboardHistoryPreviewTextSegments(text, matches, width, limit)
}

func clipboardHistoryPreviewText(item state.ClipboardHistoryItem) string {
	text := item.Text
	if strings.TrimSpace(text) == "" {
		text = item.Preview
	}
	if strings.TrimSpace(text) == "" {
		text = item.Title
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

func clipboardHistoryPreviewTextSegments(text string, matchIndexes []int, width int, limit int) []clipboardHistoryPreviewSegment {
	segments := make([]clipboardHistoryPreviewSegment, 0, limit)
	runeStart := 0
	for _, rawLine := range strings.Split(text, "\n") {
		line := SafeLine(rawLine)
		if line == "" {
			segments = append(segments, clipboardHistoryPreviewSegment{})
			runeStart++
			if len(segments) >= limit {
				return segments
			}
			continue
		}
		for DisplayWidth(line) > width {
			chunk := SliceCells(line, 0, width)
			segments = append(segments, clipboardHistoryPreviewSegment{
				Text:       chunk,
				MatchIndex: clipboardHistoryShiftMatchIndexes(matchIndexes, runeStart, len([]rune(chunk))),
			})
			runeStart += len([]rune(chunk))
			line = SliceCells(line, width, DisplayWidth(line))
			if len(segments) >= limit {
				return segments
			}
		}
		segments = append(segments, clipboardHistoryPreviewSegment{
			Text:       line,
			MatchIndex: clipboardHistoryShiftMatchIndexes(matchIndexes, runeStart, len([]rune(line))),
		})
		runeStart += len([]rune(line)) + 1
		if len(segments) >= limit {
			return segments
		}
	}
	return segments
}

func clipboardHistoryShiftMatchIndexes(matchIndexes []int, start int, length int) []int {
	if len(matchIndexes) == 0 || length <= 0 {
		return nil
	}
	out := make([]int, 0, len(matchIndexes))
	for _, index := range matchIndexes {
		if index >= start && index < start+length {
			out = append(out, index-start)
		}
	}
	return out
}

func clipboardHistoryNameWidth(shell state.ShellStore) int {
	return state.ClipboardHistoryNameWidth(shell.ReadonlyDefaults().Overlay)
}

func clipboardHistoryContentNameWidth(content ContentVM) int {
	return state.ClipboardHistoryNameWidth(state.OverlayState{ClipboardNameWidth: content.Meta.ClipboardNameWidth})
}

func clipboardHistoryRowWidth(nameWidth int) int {
	return nameWidth + 1 + clipboardHistoryPreviewWidth
}

func clipboardHistorySearchCursorCol(query string) int {
	if query == "" {
		return DisplayWidth("Search:")
	}
	return DisplayWidth("Search: ") + DisplayWidth(query)
}

func clipboardHistoryHitRegions(rows []state.ClipboardHistoryItem, rowOffset int, listStart int, visibleRows int, nameWidth int) []HitRegion {
	if visibleRows <= 0 || listStart >= len(rows) {
		return nil
	}
	end := minInt(len(rows), listStart+visibleRows)
	regions := make([]HitRegion, 0, maxInt(0, end-listStart))
	for index := listStart; index < end; index++ {
		regions = append(regions, HitRegion{
			Kind:       HitRegionContentAction,
			Rect:       Rect{Y: rowOffset + index - listStart, W: nameWidth, H: 1},
			Row:        index,
			HasRow:     true,
			ActionID:   ActionClipboardHistorySelect.String(),
			Invocation: invocationForProjection(ActionClipboardHistorySelect),
			TargetMode: HitTargetExplicit,
		})
	}
	return regions
}

func clipboardHistoryDividerHitRegion(rowOffset int, visibleRows int, nameWidth int) HitRegion {
	return HitRegion{
		Kind:       HitRegionContentAction,
		Rect:       Rect{X: nameWidth, Y: rowOffset - 1, W: 1, H: visibleRows + 1},
		ActionID:   ActionClipboardHistoryDividerDrag.String(),
		Invocation: invocationForProjection(ActionClipboardHistoryDividerDrag),
		TargetMode: HitTargetExplicit,
	}
}

func clipboardHistoryStatus(count int, query string) string {
	if count == 0 && strings.TrimSpace(query) != "" {
		return "clipboard history: 0 filtered"
	}
	return fmt.Sprintf("clipboard history: %d", count)
}

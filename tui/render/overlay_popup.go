package render

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/anytty/anytty/tui/state"
)

const promptSuggestionVisibleRows = 6

type OverlayPopupLayoutPlan struct {
	Popup OverlayPopupVM
	Rect  Rect
}

func buildPromptSuggestionPopupVM(prompt state.PromptState) OverlayPopupVM {
	active := prompt.ActivePromptField()
	if active == nil {
		return OverlayPopupVM{}
	}
	lines := promptSuggestionLines(*active, prompt.SuggestionFocused, prompt.SuggestionSelected, prompt.SuggestionOffset)
	if len(lines) == 0 {
		return OverlayPopupVM{}
	}
	return OverlayPopupVM{
		Kind:      OverlayPopupPromptSuggestion,
		AnchorRow: 1 + prompt.ActiveField + 1,
		AnchorCol: promptFormFieldValueCol(*active),
		Lines:     lines,
	}
}

func promptSuggestionLines(field state.PromptFieldState, focused bool, selected int, offset int) []Line {
	if len(field.SuggestionItems) == 0 && strings.TrimSpace(field.SuggestionTitle) == "" && strings.TrimSpace(field.SuggestionEmpty) == "" {
		return nil
	}
	lines := []Line{}
	if strings.TrimSpace(field.SuggestionTitle) != "" {
		lines = append(lines, Line{Cells: []Cell{
			styledCell("  "+field.SuggestionTitle, StylePromptSuggestion),
		}})
	}
	if len(field.SuggestionItems) == 0 {
		if strings.TrimSpace(field.SuggestionEmpty) != "" {
			lines = append(lines, Line{Cells: []Cell{
				styledCell("  "+field.SuggestionEmpty, StylePromptSuggestion),
			}})
		}
		return lines
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= len(field.SuggestionItems) {
		selected = len(field.SuggestionItems) - 1
	}
	offset = promptSuggestionVisibleOffset(offset, selected, len(field.SuggestionItems))
	end := offset + promptSuggestionVisibleRows
	if end > len(field.SuggestionItems) {
		end = len(field.SuggestionItems)
	}
	for index := offset; index < end; index++ {
		item := field.SuggestionItems[index]
		marker := "  "
		markerStyle := StylePromptSuggestion
		itemStyle := StylePromptSuggestion
		if focused && index == selected {
			marker = "▸ "
			markerStyle = StylePromptSuggestionHit
			itemStyle = StylePromptSuggestionHit
		}
		lines = append(lines, Line{Cells: []Cell{
			styledCell(marker, markerStyle),
			styledCell("  ", itemStyle),
			styledCell(promptSuggestionItemName(item), itemStyle),
		}})
	}
	if offset > 0 || end < len(field.SuggestionItems) {
		lines = append(lines, Line{Cells: []Cell{
			styledCell(fmt.Sprintf("  %d-%d/%d", offset+1, end, len(field.SuggestionItems)), StylePromptSuggestion),
		}})
	}
	return lines
}

func promptSuggestionItemName(item string) string {
	trimmed := strings.TrimRight(item, "/\\")
	if trimmed == "" {
		return item
	}
	name := filepath.Base(trimmed)
	if name == "." || name == string(filepath.Separator) {
		return item
	}
	return name + "/"
}

// 候选框只渲染可见窗口，避免弹出层随目录数量无限增高。
func promptSuggestionVisibleOffset(offset int, selected int, count int) int {
	if count <= promptSuggestionVisibleRows {
		return 0
	}
	maxOffset := count - promptSuggestionVisibleRows
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	if selected < offset {
		return selected
	}
	if selected >= offset+promptSuggestionVisibleRows {
		return selected - promptSuggestionVisibleRows + 1
	}
	return offset
}

func measureOverlayPopup(popup OverlayPopupVM, contentRect Rect, viewport Rect) OverlayPopupLayoutPlan {
	if popup.Kind == OverlayPopupNone || len(popup.Lines) == 0 || contentRect.W <= 0 || contentRect.H <= 0 {
		return OverlayPopupLayoutPlan{}
	}
	width := overlayPopupWidth(popup)
	height := len(popup.Lines)
	if width <= 0 || height <= 0 {
		return OverlayPopupLayoutPlan{}
	}
	width = minInt(width, viewport.W)
	height = minInt(height, viewport.H)
	if width <= 0 || height <= 0 {
		return OverlayPopupLayoutPlan{}
	}
	x := contentRect.X + popup.AnchorCol
	y := contentRect.Y + popup.AnchorRow
	right := viewport.X + viewport.W
	bottom := viewport.Y + viewport.H
	if x+width > right {
		x = maxInt(viewport.X, right-width)
	}
	if y+height > bottom {
		// 候选框优先贴在字段下方；空间不足时翻到字段上方，避免被 viewport 裁掉。
		y = contentRect.Y + popup.AnchorRow - height - 1
	}
	if y < viewport.Y {
		y = viewport.Y
	}
	return OverlayPopupLayoutPlan{
		Popup: popup,
		Rect:  Rect{X: maxInt(viewport.X, x), Y: y, W: width, H: height},
	}
}

func overlayPopupWidth(popup OverlayPopupVM) int {
	width := 0
	for _, line := range popup.Lines {
		width = maxInt(width, line.Width())
	}
	return width
}

func renderOverlayPopup(c *canvas, plan OverlayPopupLayoutPlan) Layer {
	if plan.Rect.W <= 0 || plan.Rect.H <= 0 || len(plan.Popup.Lines) == 0 {
		return Layer{}
	}
	lines := make([]Line, 0, minInt(plan.Rect.H, len(plan.Popup.Lines)))
	for index := 0; index < plan.Rect.H && index < len(plan.Popup.Lines); index++ {
		line := popupLineWithFill(plan.Popup.Lines[index], plan.Rect.W)
		c.writeLine(plan.Rect.X, plan.Rect.Y+index, plan.Rect.W, line, string(plan.Popup.Kind), LayerPopup)
		lines = append(lines, line)
	}
	return Layer{Kind: LayerPopup, Rect: plan.Rect, Lines: lines}
}

func popupLineWithFill(line Line, width int) Line {
	if width <= 0 {
		return Line{}
	}
	line = line.Clone()
	if fill := width - line.Width(); fill > 0 {
		line.Cells = append(line.Cells, Cell{Text: strings.Repeat(" ", fill), Width: fill, Style: StylePromptSuggestion, Safe: true})
	}
	return line
}

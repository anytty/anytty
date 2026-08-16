package render

import "strings"

func paneChromeActionText(width int) string {
	items := paneChromeActionItems(width)
	return paneChromeActionTextFromItems(items)
}

func paneChromeActionTextFromItems(items []paneChromeActionItem) string {
	if len(items) == 0 {
		return ""
	}
	return paneChromeActionRenderedFromItemsForState(items, "", false).Text
}

func paneChromeActionItems(width int) []paneChromeActionItem {
	if width < 8 {
		return nil
	}
	items := paneChromeActionItemsFromSpecs(ActionPaneClose)
	full := paneChromeActionItemsFromSpecs(ActionPaneZoom, ActionPaneSplitRight, ActionPaneSplitDown, ActionPaneClose)
	if paneChromeActionItemsWidth(full) <= maxInt(0, width-6) {
		return full
	}
	return items
}

func visiblePaneChromeActionItems(panel PanelVM, width int) []paneChromeActionItem {
	actions := paneChromeActionItemsFromVM(panel.Chrome.Actions)
	if len(actions) == 0 {
		actions = paneChromeActionItems(width)
	}
	return fitPaneChromeActionItems(actions, width)
}

func paneChromeActionItemsFromVM(actions []ChromeActionVM) []paneChromeActionItem {
	pending := make([]ChromeActionVM, 0, len(actions))
	for _, action := range actions {
		text := strings.TrimSpace(action.Text)
		if text == "" || action.ActionID == "" {
			continue
		}
		action.Text = text
		pending = append(pending, action)
	}
	out := make([]paneChromeActionItem, 0, len(pending))
	for index, action := range pending {
		label := action.ActionID
		if action.Label != "" {
			label = action.Label
		} else if spec, ok := ProjectionByID(ProjectionID(action.ActionID)); ok {
			label = projectionActionLabel(spec)
		}
		out = append(out, paneChromeActionItemFromGlyph(action.Text, action.ActionID, label, action.Style, action.IsZoomMode, index, len(pending)))
	}
	return out
}

func fitPaneChromeActionItems(actions []paneChromeActionItem, width int) []paneChromeActionItem {
	if width < 8 || len(actions) == 0 {
		return nil
	}
	if paneChromeActionItemsWidth(actions) <= maxInt(0, width-6) {
		return actions
	}
	for i := len(actions) - 1; i >= 0; i-- {
		if actions[i].ActionID == ActionPaneClose.String() {
			if paneChromeActionItemsWidth([]paneChromeActionItem{actions[i]}) <= maxInt(0, width-5) {
				return []paneChromeActionItem{actions[i]}
			}
			return nil
		}
	}
	return nil
}

func paneChromeActionItemsWidth(items []paneChromeActionItem) int {
	return paneChromeActionItemsWidthForState(items, false)
}

func paneChromeActionItemsWidthForState(items []paneChromeActionItem, active bool) int {
	if len(items) == 0 {
		return 0
	}
	return paneChromeSegmentsWidth(paneChromeActionRenderedFromItemsForState(items, "", active).Segments)
}

type paneChromeActionItem struct {
	Text     string
	Markup   string
	ActionID string
	Style    StyleToken
	ZoomMode bool
}

func paneChromeActionItemFromGlyph(glyph string, actionID string, label string, style StyleToken, zoomMode bool, index int, count int) paneChromeActionItem {
	glyph = strings.TrimSpace(glyph)
	if glyph == "" {
		glyph = "?"
	}
	ctx := paneChromeTemplateContext{
		Glyph:    glyph,
		Text:     glyph,
		ActionID: actionID,
		Label:    label,
		Index:    index,
		Count:    count,
		First:    index == 0,
		Last:     count > 0 && index == count-1,
		ZoomMode: zoomMode,
	}
	// 中文说明：action 左右部是纯展示模板；ActionID 仍来自 spec/VM，
	// 不能让模板绕过 pane reducer 的命令链路。
	format := paneChromeActionLeft() + "{{glyph}}" + paneChromeActionRight()
	markup := paneChromeExecuteTemplateString(format, ctx)
	segments := paneChromeTemplateSegments(markup, style)
	return paneChromeActionItem{
		Text:     paneChromeSegmentsText(segments),
		Markup:   markup,
		ActionID: actionID,
		Style:    style,
		ZoomMode: zoomMode,
	}
}

func paneChromeActionRenderedFromItems(items []paneChromeActionItem, style StyleToken) paneChromeRenderedText {
	return paneChromeActionRenderedFromItemsForState(items, style, style == StyleAccent)
}

func paneChromeActionRenderedFromItemsForState(items []paneChromeActionItem, style StyleToken, active bool) paneChromeRenderedText {
	if len(items) == 0 {
		return paneChromeRenderedText{}
	}
	markup := paneChromeActionMarkupFromItems(items, active)
	segments := paneChromeTemplateSegments(markup, style)
	return paneChromeRenderedText{Text: paneChromeSegmentsText(segments), Segments: segments}
}

func paneChromeActionMarkupFromItems(items []paneChromeActionItem, active bool) string {
	if len(items) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString(paneChromeActionGroupPart(paneChromeActionGroupLeft(), items, 0, active))
	for index, item := range items {
		if index > 0 {
			out.WriteString(paneChromeActionGroupPart(paneChromeActionSeparator(), items, index, active))
		}
		out.WriteString(item.Markup)
	}
	out.WriteString(paneChromeActionGroupPart(paneChromeActionGroupRight(), items, len(items)-1, active))
	return out.String()
}

func paneChromeActionGroupPart(format string, items []paneChromeActionItem, index int, active bool) string {
	if format == "" {
		return ""
	}
	count := len(items)
	ctx := paneChromeTemplateContext{
		Index:  index,
		Count:  count,
		First:  index == 0,
		Last:   count > 0 && index == count-1,
		Active: active,
	}
	if index >= 0 && index < count {
		ctx.Glyph = items[index].Text
		ctx.Text = items[index].Text
		ctx.ActionID = items[index].ActionID
		ctx.ZoomMode = items[index].ZoomMode
	}
	return paneChromeExecuteTemplateString(format, ctx)
}

func paneChromeActionItemsFromSpecs(ids ...ProjectionID) []paneChromeActionItem {
	type projectionSpecItem struct {
		spec ProjectionSpec
	}
	specs := make([]projectionSpecItem, 0, len(ids))
	for _, id := range ids {
		spec, ok := ProjectionByID(id)
		if !ok || spec.ChromeGlyph == "" {
			continue
		}
		specs = append(specs, projectionSpecItem{spec: spec})
	}
	out := make([]paneChromeActionItem, 0, len(specs))
	for index, item := range specs {
		out = append(out, paneChromeActionItemFromGlyph(item.spec.ChromeGlyph, item.spec.ID.String(), projectionActionLabel(item.spec), "", false, index, len(specs)))
	}
	return out
}

func paneChromeBracketToken(glyph string) string {
	return paneChromeActionItemFromGlyph(glyph, "", "", "", false, 0, 1).Text
}

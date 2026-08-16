package render

import (
	"strconv"
	"strings"

	actiondomain "github.com/anytty/anytty/tui/action"
)

const HeaderTabCreateText = "  + "

func renderShellFrame(c *canvas, plan LayoutPlan) {
	// shell 默认不绘制整屏外框；边界由 pane、floating 和 overlay 各自表达。
	_ = c
	_ = plan
}

func renderHeader(c *canvas, header HeaderVM, rect Rect, topFrame Rect, dividerFrame Rect) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	_ = topFrame
	_ = dividerFrame
	left := headerLeftSegments(header)
	right := []barSegment{}
	if header.Notice != "" {
		right = append(right, barText(" ! "+header.Notice+" ", StyleStatusWarning, 0))
	}
	c.writeLine(rect.X, rect.Y, rect.W, composeHeaderBarLine(left, right, rect.W), "shell:header", LayerChrome)
}

func headerLeftSegments(header HeaderVM) []barSegment {
	workspace := header.Workspace
	if workspace == "" {
		workspace = header.Title
	}
	if workspace == "" {
		workspace = "anytty"
	}
	left := headerWorkspaceSegments(header, workspace)
	left = append(left, headerTabSegmentsForHeader(header, header.Tab)...)
	left = append(left, headerCreateSegments(header)...)
	if header.Notice != "" {
		if active := compactHeaderMeta("pane", header.ActivePane); active != "" {
			left = append(left, headerSep(), barText(" "+active+" ", StyleStatusMuted, 4))
		}
	}
	return left
}

func headerCreateSegments(header HeaderVM) []barSegment {
	if segments := headerCreateTemplateSegments(header.TabCreateTemplate, header.TabCreateIcon); len(segments) > 0 {
		return segments
	}
	return []barSegment{barText(headerTabCreateText(header.TabCreateIcon), StyleHeaderCreate, 3).withAction(ActionTabCreate.String())}
}

func headerWorkspaceSegments(header HeaderVM, workspace string) []barSegment {
	if segments := headerWorkspaceTemplateSegments(header.WorkspaceTemplate, workspace); len(segments) > 0 {
		return segments
	}
	return []barSegment{
		barText(" WS "+workspace, StyleHeaderWorkspace, 1).withAction("menu.workbench_tree"),
	}
}

func renderFooter(c *canvas, footer FooterVM, rect Rect, frame Rect) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	_ = frame
	left := footerLeftSegments(footer, rect.W)
	right := footerMetadataSegments(footer, footerHintIsCritical(footer))
	if remaining := rect.W - barSegmentsWidth(left); remaining > 0 {
		right = trimBarSegments(right, remaining)
	} else {
		right = nil
	}
	c.writeLine(rect.X, rect.Y, rect.W, composeFooterBarLine(left, right, rect.W), "shell:footer", LayerChrome)
}

func footerLeftSegments(footer FooterVM, width int) []barSegment {
	mode := footer.Mode
	if mode == "" {
		mode = "live"
	}
	floatingPositionMode := mode == "resize" && strings.HasPrefix(footer.ActiveTarget, "float:")
	available := footerActionAvailableWidth(footer, width)
	left := []barSegment{}
	if modeBadge := footerModeBadgeSegments(footer, mode); len(modeBadge) > 0 && (!floatingPositionMode || width >= 72) {
		left = append(left, modeBadge...)
	}
	if len(footer.ActionTokens) > 0 {
		actions := footer.ActionTokens
		if !floatingPositionMode {
			actions = footerActionTokensVisibleForWidth(actions, mode, width)
		} else if width < 72 {
			actions = footerActionKeysOnly(actions)
		}
		left = appendFooterActionSegments(left, actions, available, footerKeyTemplateForFooter(footer), footer.ActionTemplate, footer.ActionSeparator)
	}
	return left
}

func footerActionKeysOnly(actions []FooterActionVM) []FooterActionVM {
	out := append([]FooterActionVM(nil), actions...)
	for index := range out {
		out[index].Icon = ""
		out[index].Label = ""
	}
	return out
}

func footerActionTokensVisibleForWidth(actions []FooterActionVM, mode string, width int) []FooterActionVM {
	return footerActionTokensVisibleByModeAndWidth(actions, mode, width)
}

func footerActionAvailableWidth(footer FooterVM, width int) int {
	if width <= 0 {
		return width
	}
	reserved := footerSummaryReservedWidth(footer, width)
	if reserved <= 0 {
		return width
	}
	return maxInt(0, width-reserved)
}

func footerSummaryReservedWidth(footer FooterVM, width int) int {
	segments := footerMetadataSegments(footer, footerHintIsCritical(footer))
	if len(segments) == 0 {
		return 0
	}
	if footerHintIsCritical(footer) {
		return barSegmentsWidth(segments)
	}
	if footer.Mode != "" && footer.Mode != "live" && footer.Mode != "normal" {
		return 0
	}
	if width >= 120 {
		return barSegmentsWidth(segments)
	}
	return footerSummaryTokenWidth(segments, "ws:") + footerFloatingSummaryWidth(segments) + 1
}

func footerSummaryTokenWidth(segments []barSegment, prefix string) int {
	width := 0
	for _, segment := range segments {
		if strings.HasPrefix(strings.TrimSpace(segment.text), prefix) {
			width += DisplayWidth(segment.text)
		}
	}
	return width
}

func footerFloatingSummaryWidth(segments []barSegment) int {
	width := 0
	for _, segment := range segments {
		token := strings.TrimSpace(segment.text)
		if strings.HasPrefix(token, "float:") || strings.HasPrefix(token, "collapsed:") {
			width += DisplayWidth(segment.text)
		}
	}
	return width
}

func footerHintIsCritical(footer FooterVM) bool {
	return strings.HasPrefix(footer.Hint, "error:") || strings.HasPrefix(footer.Hint, "exited:")
}

func footerMetadataSegments(footer FooterVM, hintIsCritical bool) []barSegment {
	right := []barSegment{}
	if hintIsCritical {
		if target := compactActiveTarget(footer.ActiveTarget); target != "" {
			right = append(right, barText(" "+target+" ", StyleFooterAccent, 2))
		}
	}
	right = append(right, footerSummarySegmentsForFooter(footer)...)
	if hintIsCritical && footer.Hint != "" {
		right = append(right, barText(" "+footer.Hint+" ", StyleWarning, 0))
	}
	return right
}

func compactFooterSummary(value string) string {
	return compactGlobalSummary(value)
}

func footerSummarySegmentsForFooter(footer FooterVM) []barSegment {
	value := compactFooterSummary(footer.GlobalSummary)
	tokens := metadataTokens(value)
	if len(tokens) == 0 {
		return nil
	}
	segments := make([]barSegment, 0, len(tokens)+1)
	for _, token := range tokens {
		style := StyleFooterMuted
		priority := 2
		actionID := ""
		invocation := actiondomain.Invocation{}
		displayToken := token
		if strings.HasPrefix(token, "float:") {
			style = StyleFooterAccent
			priority = 1
			displayToken = footerSummaryTemplateText(footer.FloatingSummaryTemplate, defaultFooterFloatingSummaryTemplate, "count", strings.TrimSpace(strings.TrimPrefix(token, "float:")))
			if footer.FloatingSummaryOpen {
				actionID = "menu.floating_overview"
				invocation = actiondomain.Invocation{ID: "menu.floating_overview", SourceActionID: "menu.floating_overview"}
			}
		} else if strings.HasPrefix(token, "collapsed:") {
			style = StyleFooterAccent
			priority = 1
			displayToken = footerSummaryTemplateText(footer.FloatingCollapsedSummaryTemplate, defaultFooterFloatingCollapsedSummaryTemplate, "count", strings.TrimSpace(strings.TrimPrefix(token, "collapsed:")))
			if footer.FloatingSummaryOpen {
				actionID = "menu.floating_overview"
				invocation = actiondomain.Invocation{ID: "menu.floating_overview", SourceActionID: "menu.floating_overview"}
			}
		} else if strings.HasPrefix(token, "terminals:") {
			priority = 4
			displayToken = footerSummaryTemplateText(footer.TerminalsSummaryTemplate, defaultFooterTerminalsSummaryTemplate, "count", strings.TrimSpace(strings.TrimPrefix(token, "terminals:")))
		} else if strings.HasPrefix(token, "ws:") {
			displayToken = footerSummaryTemplateText(footer.WorkspaceSummaryTemplate, defaultFooterWorkspaceSummaryTemplate, "workspace", strings.TrimSpace(strings.TrimPrefix(token, "ws:")))
		} else if strings.HasPrefix(token, "tabs:") {
			displayToken = footerSummaryTemplateText(footer.TabsSummaryTemplate, defaultFooterTabsSummaryTemplate, "count", strings.TrimSpace(strings.TrimPrefix(token, "tabs:")))
		} else if strings.HasPrefix(token, "panes:") {
			displayToken = footerSummaryTemplateText(footer.PanesSummaryTemplate, defaultFooterPanesSummaryTemplate, "count", strings.TrimSpace(strings.TrimPrefix(token, "panes:")))
		} else if token == "keylock:on" {
			style = StyleStatusWarning
			if replacement := footerTokenTemplateText(footer.KeylockOnTemplate, map[string]string{"keylock": "on"}); replacement != "" {
				displayToken = replacement
			}
		}
		if displayToken == "" {
			continue
		}
		segments = append(segments, barText(" "+displayToken, style, priority).withAction(actionID).withInvocation(invocation))
	}
	segments = append(segments, barText(" ", StyleFooterMuted, 4))
	return segments
}

func headerTabSegmentsForHeader(header HeaderVM, fallback string) []barSegment {
	if len(header.Tabs) == 0 {
		if strings.TrimSpace(fallback) == "" {
			return nil
		}
		return headerTabSegments(fallback)
	}
	segments := make([]barSegment, 0, len(header.Tabs)*3)
	for index, tab := range header.Tabs {
		tabIndex := tab.Index
		if tabIndex <= 0 {
			tabIndex = index + 1
		}
		label := strings.TrimSpace(tab.Title)
		if label == "" {
			label = tab.ID
		}
		if label == "" {
			label = "tab"
		}
		closeAction := tab.CloseActionID
		if closeAction == "" {
			closeAction = ActionTabClose.String()
		}
		closeTarget := tab.CloseTargetID
		if closeTarget == "" {
			closeTarget = tab.ID
		}
		segments = append(segments, headerTabSegmentParts(tabIndex, label, tab.Active, tab.ID, closeAction, closeTarget, header.TabTemplate)...)
	}
	return segments
}

func headerTabSegments(tab string) []barSegment {
	fields := strings.Fields(tab)
	if len(fields) == 0 {
		fields = []string{"[main]"}
	}
	segments := make([]barSegment, 0, len(fields)*3)
	for index, field := range fields {
		active := strings.HasPrefix(field, "[") && strings.HasSuffix(field, "]")
		if len(fields) == 1 {
			active = true
		}
		label := strings.Trim(field, "[]")
		if label == "" {
			label = field
		}
		if active {
			segments = append(segments, headerTabSegmentParts(index+1, label, true, "", ActionTabClose.String(), "", "")...)
			continue
		}
		segments = append(segments, headerTabSegmentParts(index+1, label, false, "", ActionTabClose.String(), "", "")...)
	}
	return segments
}

func headerTabSegmentParts(index int, label string, active bool, tabID string, closeAction string, closeTarget string, template string) []barSegment {
	tabAction := ActionTabSwitch.String()
	if segments := headerTabTemplateSegments(template, headerTabTemplateContext{
		Index:        index,
		Title:        label,
		TabID:        tabID,
		Active:       active,
		SwitchAction: tabAction,
		CloseAction:  closeAction,
		CloseTarget:  closeTarget,
		CloseIcon:    paneChromeCloseGlyph(),
	}); len(segments) > 0 {
		return segments
	}
	marker := " "
	markerStyle := StyleHeaderSpacer
	indexStyle := StyleHeaderInactiveIndex
	titleStyle := StyleHeaderInactiveTitle
	closeStyle := StyleHeaderInactiveClose
	if active {
		marker = "▎"
		markerStyle = StyleHeaderActiveMarker
		indexStyle = StyleHeaderActiveIndex
		titleStyle = StyleHeaderActiveTitle
		closeStyle = StyleHeaderActiveClose
	}
	return []barSegment{
		barText(" ", StyleHeaderSpacer, 1),
		barText(marker, markerStyle, 1).withAction(tabAction).withTarget(tabID),
		barText(" "+intLabel(index), indexStyle, 1).withAction(tabAction).withTarget(tabID),
		barText(" "+label+" ", titleStyle, 1).withAction(tabAction).withTarget(tabID),
		barText(paneChromeCloseGlyph(), closeStyle, 2).withAction(closeAction).withTarget(closeTarget),
	}
}

func headerTabCreateText(icon string) string {
	icon = strings.TrimSpace(icon)
	if icon == "" {
		return HeaderTabCreateText
	}
	return "  " + icon + " "
}

func intLabel(value int) string {
	if value < 0 {
		value = 0
	}
	return strconv.Itoa(value)
}

func appendFooterActionSegments(segments []barSegment, actions []FooterActionVM, width int, keyTemplate string, actionTemplate string, separatorText string) []barSegment {
	limit := 58
	if width < 60 {
		limit = 34
	} else if width < 100 {
		limit = 44
	}
	if width >= 120 {
		limit = width
	} else if width > 0 {
		limit = width
	}
	separator := footerActionSep(width, separatorText)
	selected := selectFooterActionTokens(actions, limit, DisplayWidth(separator.text), keyTemplate, actionTemplate)
	for _, action := range selected {
		key := strings.TrimSpace(action.Key)
		keyText := footerActionKeyText(key, keyTemplate)
		decor := footerActionDecorText(action, actionTemplate)
		if keyText == "" && decor == "" {
			continue
		}
		if len(segments) > 0 {
			segments = append(segments, separator)
		}
		style := action.Style
		if style == "" {
			style = footerActionKeyStyle(key, decor)
		}
		style = footerActionDisplayStyle(key, decor, style)
		actionID := action.ActionID
		invocation := action.Invocation
		if action.Click != ClickClickable || invocation.ID == "" {
			actionID = ""
			invocation = actiondomain.Invocation{}
		}
		withAction := func(segment barSegment) barSegment {
			if actionID == "" {
				return segment
			}
			return segment.withAction(actionID).withInvocation(invocation)
		}
		if keyText == "" {
			// key 模板为空时只展示 icon/label，仍保留 action hit region。
		} else {
			segments = appendFooterKeySegmentsWithInvocation(segments, keyText, style, actionID, invocation)
		}
		showLabel := decor != ""
		if showLabel {
			decorStyle := StyleFooterMuted
			if keyText == "" {
				decorStyle = style
			}
			segments = append(segments, withAction(barText(" "+decor, decorStyle, 1)))
		}
	}
	return segments
}

func appendFooterKeySegmentsWithInvocation(segments []barSegment, key string, style StyleToken, actionID string, invocation actiondomain.Invocation) []barSegment {
	start := len(segments)
	segments = appendFooterKeySegments(segments, key, style, actionID)
	for index := start; index < len(segments); index++ {
		if segments[index].actionID != "" {
			segments[index].invocation = invocation
		}
	}
	return segments
}

func appendFooterKeySegments(segments []barSegment, key string, style StyleToken, actionID string) []barSegment {
	for tokenIndex, token := range formatFooterKeySegments(key) {
		if tokenIndex > 0 {
			segments = append(segments, barText(" • ", StyleFooterMuted, 1).withAction(actionID))
		}
		segments = appendFooterBracketTokenSegments(segments, token, style, actionID)
	}
	return segments
}

func appendFooterBracketTokenSegments(segments []barSegment, token string, style StyleToken, actionID string) []barSegment {
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, "[") || !strings.HasSuffix(token, "]") || DisplayWidth(token) < 2 {
		return append(segments, barText(token, style, 1).withAction(actionID))
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(token, "["), "]")
	segments = append(segments,
		barText("[", StyleFooterChrome, 1).withAction(actionID),
		barText(inner, style, 1).withAction(actionID),
		barText("]", StyleFooterChrome, 1).withAction(actionID),
	)
	return segments
}

func footerActionDisplayStyle(key string, label string, style StyleToken) StyleToken {
	switch style {
	case StyleStatusAccent:
		return footerActionKeyStyle(key, label)
	case StyleStatusWarning:
		return footerActionKeyStyle(key, label)
	case StyleStatusMuted, StyleStatus:
		return StyleFooterMuted
	default:
		return style
	}
}

func selectFooterActionTokens(actions []FooterActionVM, limit int, separatorWidth int, keyTemplate string, actionTemplate string) []FooterActionVM {
	selected := make([]FooterActionVM, 0, len(actions))
	used := 0
	truncated := false
	for _, action := range actions {
		tokenWidth := footerActionTokenDisplayWidthForSelection(action, keyTemplate, actionTemplate)
		if tokenWidth <= 0 {
			continue
		}
		if len(selected) > 0 {
			tokenWidth += separatorWidth
		}
		if len(selected) > 0 && used+tokenWidth > limit {
			truncated = true
			break
		}
		if len(selected) == 0 && tokenWidth > limit {
			truncated = true
			break
		}
		selected = append(selected, action)
		used += tokenWidth
	}
	if !truncated || len(selected) == len(actions) {
		return selected
	}
	tail := footerTailActionToken(actions)
	if tail.Key == "" || containsFooterActionToken(selected, tail) {
		return selected
	}
	tailWidth := footerTailActionWidth(selected, tail, separatorWidth, keyTemplate, actionTemplate)
	for len(selected) > 0 && used+tailWidth > limit {
		selected = selected[:len(selected)-1]
		used = footerSelectedActionWidth(selected, separatorWidth, keyTemplate, actionTemplate)
		tailWidth = footerTailActionWidth(selected, tail, separatorWidth, keyTemplate, actionTemplate)
	}
	if tailWidth <= limit && used+tailWidth <= limit {
		selected = append(selected, tail)
	}
	return selected
}

func footerSelectedActionWidth(actions []FooterActionVM, separatorWidth int, keyTemplate string, actionTemplate string) int {
	width := 0
	for _, action := range actions {
		tokenWidth := footerActionTokenDisplayWidthForSelection(action, keyTemplate, actionTemplate)
		if tokenWidth <= 0 {
			continue
		}
		if width > 0 {
			width += separatorWidth
		}
		width += tokenWidth
	}
	return width
}

func footerTailActionWidth(selected []FooterActionVM, tail FooterActionVM, separatorWidth int, keyTemplate string, actionTemplate string) int {
	width := footerActionTokenDisplayWidthForSelection(tail, keyTemplate, actionTemplate)
	if width > 0 && len(selected) > 0 {
		width += separatorWidth
	}
	return width
}

func footerActionTokenDisplayWidthForSelection(action FooterActionVM, keyTemplate string, actionTemplate string) int {
	keyText := footerActionKeyText(action.Key, keyTemplate)
	decor := footerActionDecorText(action, actionTemplate)
	if keyText == "" && decor == "" {
		return 0
	}
	width := 0
	if keyText != "" {
		width += DisplayWidth(formatFooterKeyToken(keyText))
	}
	if decor != "" {
		width += 1 + DisplayWidth(decor)
	}
	return width
}

func footerTailActionToken(actions []FooterActionVM) FooterActionVM {
	for _, action := range actions {
		if action.Invocation.ID == "terminal_pool.restart" {
			// Terminal Pool 中 Ctrl+R 是高频恢复入口；footer 宽度不足时必须优先展示，
			// 避免键盘入口可触发但底栏只留下 remove 这类尾部危险动作。
			return action
		}
	}
	for i := len(actions) - 1; i >= 0; i-- {
		action := actions[i]
		if strings.TrimSpace(action.Key) != "" && !strings.HasPrefix(strings.TrimSpace(action.Key), "esc") {
			return action
		}
	}
	if len(actions) == 0 {
		return FooterActionVM{}
	}
	return actions[len(actions)-1]
}

func containsFooterActionToken(values []FooterActionVM, target FooterActionVM) bool {
	for _, value := range values {
		if value.Key == target.Key && value.Label == target.Label && value.ActionID == target.ActionID {
			return true
		}
	}
	return false
}

func formatFooterKeyToken(key string) string {
	return strings.Join(formatFooterKeySegments(key), " • ")
}

func formatFooterKeySegments(key string) []string {
	key = strings.TrimSpace(key)
	if strings.HasPrefix(key, "^") && len(key) > 1 {
		return []string{"[Ctrl+" + strings.ToUpper(strings.TrimPrefix(key, "^")) + "]"}
	}
	return []string{"[" + key + "]"}
}

func footerActionKeyStyle(key string, label string) StyleToken {
	upper := strings.ToUpper(key + " " + label)
	switch {
	case strings.Contains(upper, "X") || strings.Contains(upper, "CLOSE") || strings.Contains(upper, "KILL"):
		return StyleFooterKeyPicker
	case strings.Contains(upper, "W") || strings.Contains(upper, "WORKSPACE"):
		return StyleFooterKeyWorkspace
	case strings.Contains(upper, "F") || strings.Contains(upper, "PICK"):
		return StyleFooterKeyPicker
	case strings.Contains(upper, "O") || strings.Contains(upper, "FLOAT"):
		return StyleFooterKeyFloat
	case strings.Contains(upper, "V") || strings.Contains(upper, "COPY"):
		return StyleFooterKeyCopy
	case strings.Contains(upper, "G") || strings.Contains(upper, "GLOBAL"):
		return StyleFooterKeyGlobal
	case strings.Contains(upper, "R") || strings.Contains(upper, "RESIZE") || strings.Contains(upper, "SIZE"):
		return StyleFooterKeyResize
	case strings.Contains(upper, "P") || strings.Contains(upper, "PANE"):
		return StyleFooterKeyPane
	case strings.Contains(upper, "T") || strings.Contains(upper, "TAB") || strings.Contains(upper, "TREE"):
		return StyleFooterKeyTab
	default:
		return StyleFooterAccent
	}
}

func compactActiveTarget(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.TrimPrefix(value, "pane:")
	value = strings.ReplaceAll(value, " live", "")
	value = strings.ReplaceAll(value, " copy", "")
	return "● " + value
}

func compactHeaderMeta(label string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return label + ":" + value
}

func compactGlobalSummary(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return value
}

func metadataTokens(value string) []string {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return nil
	}
	tokens := make([]string, 0, len(fields))
	current := ""
	for _, field := range fields {
		if metadataTokenStartsField(field) {
			if current != "" {
				tokens = append(tokens, current)
			}
			current = field
			continue
		}
		if current == "" {
			current = field
			continue
		}
		current += " " + field
	}
	if current != "" {
		tokens = append(tokens, current)
	}
	return tokens
}

func metadataTokenStartsField(field string) bool {
	return strings.HasPrefix(field, "ws:") ||
		strings.HasPrefix(field, "float:") ||
		strings.HasPrefix(field, "terminals:") ||
		strings.HasPrefix(field, "keylock:") ||
		strings.HasPrefix(field, "tabs:") ||
		strings.HasPrefix(field, "panes:")
}

type barSegment struct {
	text       string
	style      StyleToken
	ansi       ANSICellStyle
	priority   int
	actionID   string
	invocation actiondomain.Invocation
	targetID   string
	joint      bool
}

func barText(text string, style StyleToken, priority int) barSegment {
	return barSegment{text: text, style: style, priority: priority}
}

func (segment barSegment) withAction(actionID string) barSegment {
	segment.actionID = actionID
	return segment
}

func (segment barSegment) withInvocation(invocation actiondomain.Invocation) barSegment {
	segment.invocation = invocation
	return segment
}

func (segment barSegment) withTarget(targetID string) barSegment {
	segment.targetID = targetID
	return segment
}

func headerSep() barSegment {
	return barText(" ", StyleHeaderSpacer, 1)
}

func footerSep() barSegment {
	return barText(" • ", StyleFooterMuted, 1)
}

func footerActionSep(width int, separatorText string) barSegment {
	if width < 56 {
		return barText(" ", StyleFooterMuted, 1)
	}
	if separatorText = strings.TrimSpace(separatorText); separatorText != "" {
		return barText(" "+separatorText+" ", StyleFooterMuted, 1)
	}
	return footerSep()
}

func composeHeaderBarLine(left []barSegment, right []barSegment, width int) Line {
	return composeBarLineWithFill(left, right, width, StyleHeaderSpacer)
}

func composeFooterBarLine(left []barSegment, right []barSegment, width int) Line {
	return composeBarLineWithFill(left, right, width, "")
}

func composeBarLineWithFill(left []barSegment, right []barSegment, width int, fillStyle StyleToken) Line {
	if width <= 0 {
		return Line{}
	}
	left = trimBarSegments(left, width)
	right = trimBarSegments(right, width-barSegmentsWidth(left))
	total := barSegmentsWidth(left) + barSegmentsWidth(right)
	if total > width {
		right = trimBarSegments(right, width-barSegmentsWidth(left))
		total = barSegmentsWidth(left) + barSegmentsWidth(right)
	}
	spacer := width - total
	cells := make([]Cell, 0, len(left)+len(right)+1)
	cells = append(cells, cellsFromBarSegments(left)...)
	if spacer > 0 {
		cells = append(cells, Cell{Text: strings.Repeat(" ", spacer), Width: spacer, Style: fillStyle, Safe: true})
	}
	cells = append(cells, cellsFromBarSegments(right)...)
	return Line{Cells: cells}
}

func trimBarSegments(segments []barSegment, width int) []barSegment {
	if width <= 0 {
		return nil
	}
	out := append([]barSegment(nil), segments...)
	for barSegmentsWidth(out) > width && len(out) > 0 {
		drop := lowestPriorityBarSegment(out)
		out = append(out[:drop], out[drop+1:]...)
	}
	if barSegmentsWidth(out) <= width {
		return cleanBarSegments(out)
	}
	return nil
}

func lowestPriorityBarSegment(segments []barSegment) int {
	index := len(segments) - 1
	for i, segment := range segments {
		if segment.priority > segments[index].priority {
			index = i
		}
	}
	return index
}

func barSegmentsWidth(segments []barSegment) int {
	width := 0
	for _, segment := range segments {
		width += DisplayWidth(segment.text)
	}
	return width
}

func cellsFromBarSegments(segments []barSegment) []Cell {
	cells := make([]Cell, 0, len(segments))
	for _, segment := range segments {
		if segment.text == "" {
			continue
		}
		style := segment.style
		if style == "" {
			style = StyleStatus
		}
		cells = append(cells, Cell{Text: segment.text, Width: DisplayWidth(segment.text), Style: style, ANSIStyle: segment.ansi, Safe: true})
	}
	return cells
}

func cleanBarSegments(segments []barSegment) []barSegment {
	if len(segments) == 0 {
		return nil
	}
	out := make([]barSegment, 0, len(segments))
	for _, segment := range segments {
		isSep := segment.text == "│" || segment.joint
		if isSep && len(out) == 0 {
			continue
		}
		if isSep && len(out) > 0 && (out[len(out)-1].text == "│" || out[len(out)-1].joint) {
			continue
		}
		out = append(out, segment)
	}
	for len(out) > 0 && (out[len(out)-1].text == "│" || out[len(out)-1].joint) {
		out = out[:len(out)-1]
	}
	return out
}

package render

import (
	"strconv"
	"strings"
)

func paneChromeTerminalLabelSlots(panel PanelVM, borderStyle StyleToken, width int) []paneChromeTopSlot {
	terminal := panel.Chrome.Terminal
	// 中文说明：右侧 terminal 状态槽固定宽度，标题只吃剩余宽度，避免长标题挤动 action 命中区。
	right := paneChromeTerminalRightSlots(terminal, borderStyle)
	for len(right) > 0 && paneChromeSlotsWidth(right)+paneChromeTerminalMinimumTitleWidth(terminal) > width {
		right = right[1:]
	}
	rightWidth := paneChromeSlotsWidth(right)
	left := paneChromeTerminalLeftSlots(terminal, panel, maxInt(0, width-rightWidth), borderStyle)
	slots := make([]paneChromeTopSlot, 0, len(left)+len(right))
	slots = append(slots, left...)
	return append(slots, right...)
}

func paneChromeTerminalLeftSlots(terminal TerminalChromeVM, panel PanelVM, width int, borderStyle StyleToken) []paneChromeTopSlot {
	if width <= 0 {
		return nil
	}
	prefix := paneChromeTerminalSizeLockSlot(terminal, borderStyle)
	for len(prefix) > 0 && paneChromeSlotsWidth(prefix)+paneChromeTerminalMinimumTitleWidth(terminal) > width {
		prefix = prefix[:len(prefix)-1]
	}
	titleWidth := maxInt(0, width-paneChromeSlotsWidth(prefix))
	title := paneChromeTerminalLeftTitle(terminal, panel, titleWidth, borderStyle)
	slots := make([]paneChromeTopSlot, 0, len(prefix)+1)
	slots = append(slots, prefix...)
	if strings.TrimSpace(title.text) != "" {
		slots = append(slots, title)
	}
	return slots
}

func paneChromeTerminalLeftTitle(terminal TerminalChromeVM, panel PanelVM, width int, borderStyle StyleToken) paneChromeTopSlot {
	if width <= 0 {
		return paneChromeTopSlot{}
	}
	title := strings.TrimSpace(terminal.Title.Text)
	if title == "" {
		title = paneChromeTitleSource(panel)
	}
	if title == "" {
		return paneChromeTopSlot{}
	}
	prefix := paneChromeTerminalTitlePrefix(terminal)
	prefixWidth := DisplayWidth(prefix)
	text := prefix + title
	if width > 2 {
		titleWidth := maxInt(0, width-2-prefixWidth)
		text = prefix + TruncateCells(title, titleWidth)
		text = " " + text + " "
	} else {
		text = TruncateCells(text, width)
	}
	style := terminal.Title.Style
	if style == "" {
		style = paneChromeTitleStyle(panel, borderStyle)
	}
	return paneChromeTopSlot{text: text, layoutWidth: width, style: style, priority: 0}
}

func paneChromeTerminalSizeLockSlot(terminal TerminalChromeVM, borderStyle StyleToken) []paneChromeTopSlot {
	lockGlyph := paneChromeSizeUnlockGlyph()
	if terminal.Locked {
		lockGlyph = paneChromeSizeLockGlyph()
	}
	actionID := ""
	if terminal.CanLockSize {
		actionID = ActionResizeLayoutLock.String()
	}
	item := paneChromeActionItemFromGlyph(lockGlyph, actionID, "layout lock", borderStyle, false, 0, 1)
	return []paneChromeTopSlot{{text: item.Text, segments: paneChromeTemplateSegments(item.Markup, borderStyle), style: borderStyle, priority: 1, actionID: actionID}}
}

func paneChromeTerminalTitlePrefix(terminal TerminalChromeVM) string {
	if terminalChromeLayoutAdjusted(terminal) {
		return "◇ "
	}
	return ""
}

func paneChromeTerminalRightSlots(terminal TerminalChromeVM, borderStyle StyleToken) []paneChromeTopSlot {
	stateText := strings.TrimSpace(terminal.State.Text)
	if stateText == "" {
		stateText = paneChromeRunningGlyph()
	}
	stateStyle := paneChromeSlotStyle(terminal.State, borderStyle)
	ownerStyle := paneChromeSlotStyle(terminal.Owner, borderStyle)
	ownerText := strings.TrimSpace(terminal.Owner.Text)
	if ownerText == "" && terminal.TakeOwner {
		ownerText = paneChromeTakeOwnerText()
	}
	count := terminal.AttachCount
	if count < 1 {
		count = 1
	}
	slots := []paneChromeTopSlot{
		{text: paneChromeFixedSlot(stateText, 3), style: stateStyle, priority: 2},
		{text: paneChromeFixedSlot("x"+strconv.Itoa(count), 4), style: borderStyle, priority: 3},
	}
	if owner := paneChromeOwnerSlot(ownerText, ownerStyle, terminalOwnerActionID(terminal)); owner.text != "" || len(owner.segments) > 0 {
		slots = append(slots, owner)
	}
	return slots
}

func terminalChromeLayoutAdjusted(terminal TerminalChromeVM) bool {
	return (terminal.LayoutMode != "" && terminal.LayoutMode != "auto") ||
		terminal.PanX != 0 ||
		terminal.PanY != 0 ||
		(terminal.AlignX != "" && terminal.AlignX != "start") ||
		(terminal.AlignY != "" && terminal.AlignY != "start")
}

func paneChromeFixedSlot(text string, width int) string {
	text = strings.TrimSpace(text)
	if width <= 0 || text == "" {
		return ""
	}
	text = TruncateCells(text, width)
	pad := width - DisplayWidth(text)
	if pad <= 0 {
		return text
	}
	left := pad / 2
	right := pad - left
	return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
}

func paneChromeOwnerSlot(text string, style StyleToken, actionID string) paneChromeTopSlot {
	text = strings.TrimSpace(text)
	if text == "" {
		return paneChromeTopSlot{}
	}
	text = paneChromeFixedSlot(text, maxInt(8, DisplayWidth(text)))
	ctx := paneChromeTemplateContext{Text: text}
	format := paneChromeOwnerLeft() + "{{text}}" + paneChromeOwnerRight()
	markup := paneChromeExecuteTemplateString(format, ctx)
	segments := paneChromeTemplateSegments(markup, style)
	return paneChromeTopSlot{
		text:     paneChromeSegmentsText(segments),
		segments: segments,
		style:    style,
		priority: 4,
		actionID: actionID,
	}
}

func terminalOwnerActionID(terminal TerminalChromeVM) string {
	if terminal.TakeOwner {
		return ActionTerminalTakeResizeOwner.String()
	}
	return ""
}

func paneChromeTerminalMinimumTitleWidth(terminal TerminalChromeVM) int {
	if strings.TrimSpace(terminal.Title.Text) == "" {
		return 0
	}
	return 5
}

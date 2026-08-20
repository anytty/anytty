package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/anytty/anytty/tui/state"
)

const defaultWorkbenchNavigatorTreeWidth = 36
const workbenchNavigatorTreeWidthBoost = 10

type workbenchNavigatorLayout struct {
	ContentWidth   int
	BodyRows       int
	TreeWidth      int
	RightWidth     int
	ActionRow      int
	SnapshotX      int
	SnapshotY      int
	SnapshotWidth  int
	SnapshotHeight int
}

// Workbench Navigator 展示 reducer-owned workspace/tab/pane 树；右侧 snapshot 只消费当前 TUI 已持有的 live 投影。
func buildWorkbenchTreeContent(root state.Root, shell state.ShellStore) ContentVM {
	shell = shell.ReadonlyDefaults()
	query := strings.TrimSpace(shell.Overlay.Query)
	rows := state.WorkbenchTreeItems(root)
	layout := workbenchNavigatorLayoutForViewport(chromeSafeViewportForShell(root.Viewport, shell))
	lines := workbenchNavigatorLines(root, rows, query, layout)
	rowOffset := 3
	visibleRows := rows
	if len(visibleRows) > layout.BodyRows {
		visibleRows = visibleRows[:layout.BodyRows]
	}
	regions := workbenchTreeHitRegions(visibleRows, rowOffset, layout.TreeWidth)
	regions = append(regions, workbenchNavigatorDetailHitRegions(root, rows, layout)...)
	return ContentVM{
		Kind:       ContentWorkbenchTree,
		Lines:      lines,
		Meta:       workbenchNavigatorMeta(root, rows, layout),
		Status:     workbenchTreeStatus(len(rows), query),
		Cursor:     Cursor{Visible: true, Row: 0, Col: searchCursorCol(query), Shape: CursorShapeBar},
		HitRegions: regions,
		Empty:      len(rows) == 0,
	}
}

func workbenchTreeRowLine(row state.WorkbenchTreeItem) Line {
	prefixStyle := StyleMuted
	if row.Selected {
		prefixStyle = StyleForeground
	}
	title := workbenchTreeTitle(row)
	prefix := workbenchTreeDepthPrefix(row.Depth)
	cells := []Cell{
		styledCell(prefix, prefixStyle),
		workbenchTreeDisclosureCell(row),
		tokenCell(workbenchTreeKindGlyph(row), workbenchTreeKindStyle(row)),
		NewCell(" "),
		workbenchTreeTitleCell(title, row),
		NewCell(" "),
	}
	// 中文说明：左侧树只表达 workbench 结构和当前选择；terminal id、runtime
	// 状态与资源采样属于右侧 detail，避免 pane title 后面堆出第二套调试信息。
	cells = append(cells, workbenchTreeInlineMetaCells(row)...)
	return Line{Cells: cells}
}

func workbenchTreeDepthPrefix(depth int) string {
	if depth <= 0 {
		return ""
	}
	// 中文说明：Workbench Navigator 以 workspace 为根层，tab 缩进一层，
	// pane/floating 再缩进一层；展开/折叠只由 disclosure glyph 表达。
	return strings.Repeat("  ", depth)
}

func workbenchTreeDisclosureCell(row state.WorkbenchTreeItem) Cell {
	if !row.Expandable {
		return styledCell("  ", StyleMuted)
	}
	glyph := "▾ "
	if row.Collapsed {
		glyph = "▸ "
	}
	style := StyleForeground
	if row.Selected {
		style = StyleAccent
	}
	return styledCell(glyph, style)
}

func workbenchNavigatorLines(root state.Root, rows []state.WorkbenchTreeItem, query string, layout workbenchNavigatorLayout) []Line {
	lines := []Line{
		workbenchNavigatorFullLine(searchRowLine(query, "main"), layout),
		workbenchNavigatorDividerLine(layout),
		workbenchNavigatorBodyLine(workbenchNavigatorHeaderLine("WORKBENCH"), Line{}, layout),
	}
	for row := 0; row < layout.BodyRows; row++ {
		left := Line{}
		if row < len(rows) {
			left = workbenchTreeRowLine(rows[row])
		}
		lines = append(lines, workbenchNavigatorBodyLine(left, Line{}, layout))
	}
	return lines
}

func workbenchNavigatorFullLine(line Line, layout workbenchNavigatorLayout) Line {
	return fitContentLine(line, layout.ContentWidth, StyleForeground)
}

func workbenchNavigatorDividerLine(layout workbenchNavigatorLayout) Line {
	return NewLine(strings.Repeat(" ", layout.ContentWidth))
}

func workbenchNavigatorBodyLine(left Line, right Line, layout workbenchNavigatorLayout) Line {
	cells := fitContentLine(left, layout.TreeWidth, StyleForeground).Cells
	cells = append(cells, NewCell(" "))
	cells = append(cells, fitContentLine(right, layout.RightWidth, StyleForeground).Cells...)
	return Line{Cells: cells}
}

func workbenchNavigatorHeaderLine(label string) Line {
	return Line{Cells: []Cell{styledCell(label, StyleStrongForeground)}}
}

func selectedWorkbenchTreeItem(rows []state.WorkbenchTreeItem) (state.WorkbenchTreeItem, bool) {
	if len(rows) == 0 {
		return state.WorkbenchTreeItem{}, false
	}
	selected := rows[0]
	for _, row := range rows {
		if row.Selected {
			return row, true
		}
	}
	return selected, true
}

func workbenchNavigatorRightHeader(selected state.WorkbenchTreeItem, ok bool) string {
	if !ok {
		return "DETAIL"
	}
	return "DETAIL"
}

func workbenchNavigatorRightLines(root state.Root, selected state.WorkbenchTreeItem, layout workbenchNavigatorLayout) []Line {
	if selected.Kind == "" {
		return []Line{NewLine("No workbench node selected")}
	}
	switch selected.Kind {
	case state.WorkbenchTreeKindPane:
		return workbenchNavigatorPaneLines(root, selected, layout)
	case state.WorkbenchTreeKindTab:
		lines := []Line{
			workbenchNavigatorDetailTitleLine(selected),
			workbenchNavigatorBadgeLine(workbenchNavigatorBadges(root, selected)),
			workbenchNavigatorDetailLine("workspace", selected.WorkspaceName),
			workbenchNavigatorDetailLine("summary", workbenchReadableSummary(selected)),
			NewLine(""),
		}
		lines = append(lines, workbenchNavigatorPaneSnapshotLines(layout)...)
		return lines
	case state.WorkbenchTreeKindWorkspace:
		return []Line{
			workbenchNavigatorDetailTitleLine(selected),
			workbenchNavigatorBadgeLine(workbenchNavigatorBadges(root, selected)),
			workbenchNavigatorDetailLine("summary", workbenchReadableSummary(selected)),
		}
	case state.WorkbenchTreeKindFloating:
		lines := []Line{
			workbenchNavigatorDetailTitleLine(selected),
			workbenchNavigatorBadgeLine(workbenchNavigatorBadges(root, selected)),
			workbenchNavigatorDetailLine("workspace", selected.WorkspaceName),
			workbenchNavigatorDetailLine("tab", selected.TabTitle),
		}
		if selected.FloatingID == "" {
			lines = append(lines, Line{Cells: []Cell{styledCell(workbenchTreePreview(selected), StyleForeground)}})
			return lines
		}
		lines = append(lines, workbenchNavigatorResourceLines(root, selected)...)
		lines = append(lines, NewLine(""))
		lines = append(lines, workbenchNavigatorPaneSnapshotLines(layout)...)
		lines = append(lines, terminalManagerDetailLine("path", workbenchTreePath(selected)))
		return lines
	default:
		return []Line{{Cells: []Cell{styledCell(workbenchTreePreview(selected), StyleForeground)}}}
	}
}

func workbenchNavigatorPaneLines(root state.Root, selected state.WorkbenchTreeItem, layout workbenchNavigatorLayout) []Line {
	lines := []Line{
		workbenchNavigatorDetailTitleLine(selected),
		workbenchNavigatorBadgeLine(workbenchNavigatorBadges(root, selected)),
		workbenchNavigatorDetailLine("workspace", selected.WorkspaceName),
		workbenchNavigatorDetailLine("tab", selected.TabTitle),
	}
	lines = append(lines, workbenchNavigatorResourceLines(root, selected)...)
	lines = append(lines, NewLine(""))
	lines = append(lines, workbenchNavigatorPaneSnapshotLines(layout)...)
	lines = append(lines, terminalManagerDetailLine("path", workbenchTreePath(selected)))
	return lines
}

func workbenchNavigatorDetailTitleLine(row state.WorkbenchTreeItem) Line {
	title := workbenchTreeTitle(row)
	if title == "" {
		title = workbenchTreeKindLabel(row)
	}
	return Line{Cells: []Cell{
		tokenCell(workbenchTreeKindGlyph(row), workbenchTreeKindStyle(row)),
		NewCell(" "),
		styledCell(title, StyleStrongForeground),
	}}
}

func workbenchNavigatorBadgeLine(badges []string) Line {
	if len(badges) == 0 {
		return NewLine("")
	}
	cells := []Cell{}
	for index, badge := range badges {
		badge = strings.TrimSpace(badge)
		if badge == "" {
			continue
		}
		if len(cells) > 0 || index > 0 {
			cells = append(cells, NewCell("  "))
		}
		cells = append(cells, styledCell(badge, workbenchNavigatorBadgeStyle(badge)))
	}
	if len(cells) == 0 {
		return NewLine("")
	}
	return Line{Cells: cells}
}

func workbenchNavigatorBadgeStyle(badge string) StyleToken {
	switch strings.ToLower(strings.TrimSpace(badge)) {
	case "active", "running", "bound", "live":
		return StyleSuccess
	case "empty", "collapsed", "exited":
		return StyleWarning
	case "error", "offline", "transport-closed", "transport-dial", "auth", "host-key", "remote-daemon", "protocol", "config", "unavailable":
		return StyleDanger
	default:
		return StyleMuted
	}
}

func workbenchNavigatorBadges(root state.Root, row state.WorkbenchTreeItem) []string {
	badges := []string{workbenchTreeKindLabel(row)}
	if row.Active {
		badges = append(badges, "active")
	}
	switch row.Kind {
	case state.WorkbenchTreeKindPane:
		badges = append(badges, workbenchPaneStateLabel(root, row))
		badges = append(badges, workbenchEndpointBadges(row)...)
		if role := workbenchPaneRoleLabel(root, row); role != "" {
			badges = append(badges, role)
		}
	case state.WorkbenchTreeKindFloating:
		if row.PaneKind == state.PaneEmpty {
			badges = append(badges, "empty")
		} else if row.PaneKind != "" {
			badges = append(badges, "live")
		}
		if strings.Contains(strings.ToLower(row.Summary), "collapsed") {
			badges = append(badges, "collapsed")
		}
		badges = append(badges, workbenchEndpointBadges(row)...)
	}
	return compactStringTokens(badges)
}

func workbenchEndpointBadges(row state.WorkbenchTreeItem) []string {
	if row.EndpointID == "" || row.TerminalID == "" {
		return nil
	}
	badges := []string{}
	switch row.EndpointStatus {
	case state.EndpointStatusOffline, state.EndpointStatusDisabled, state.EndpointStatusReconnectRequired, state.EndpointStatusUnregistered:
		badges = append(badges, string(row.EndpointStatus))
	}
	if row.EndpointErrorKind != state.EndpointErrorUnknown {
		badges = append(badges, string(row.EndpointErrorKind))
	}
	return badges
}

func workbenchNavigatorDetailLine(label string, value string) Line {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "-"
	}
	return Line{Cells: []Cell{
		styledCell(strings.ToUpper(label)+" ", StyleAccent),
		styledCell(value, StyleForeground),
	}}
}

func workbenchTreeInlineMetaCells(row state.WorkbenchTreeItem) []Cell {
	meta := workbenchTreeInlineMeta(row)
	if meta == "" {
		return nil
	}
	style := StyleMuted
	if row.Active {
		style = StyleAccent
	}
	return []Cell{styledCell(meta, style)}
}

func workbenchTreeInlineMeta(row state.WorkbenchTreeItem) string {
	switch row.Kind {
	case state.WorkbenchTreeKindWorkspace, state.WorkbenchTreeKindTab:
		return workbenchReadableSummary(row)
	case state.WorkbenchTreeKindPane, state.WorkbenchTreeKindFloating:
		if row.EndpointStatus == state.EndpointStatusOffline {
			if row.EndpointErrorKind != state.EndpointErrorUnknown {
				return string(row.EndpointErrorKind)
			}
			return string(row.EndpointStatus)
		}
		meta := ""
		if row.Active {
			meta = "active"
		}
		if activity := TerminalOutputActivityLabel(row.LastOutputAt, time.Now()); activity != "" {
			if meta != "" {
				meta += " "
			}
			meta += activity
		}
		return meta
	}
	return ""
}

func workbenchReadableSummary(row state.WorkbenchTreeItem) string {
	tokens := strings.Fields(row.Summary)
	parts := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if part := workbenchReadableSummaryToken(token); part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(compactStringTokens(parts), " · ")
}

func workbenchReadableSummaryToken(token string) string {
	key, value, ok := strings.Cut(strings.TrimSpace(token), ":")
	if !ok {
		return ""
	}
	switch strings.ToLower(key) {
	case "tabs":
		return pluralCount(value, "tab", "tabs")
	case "panes":
		return pluralCount(value, "pane", "panes")
	case "floating", "float":
		return pluralCount(value, "floating", "floating")
	default:
		return ""
	}
}

func pluralCount(value string, singular string, plural string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if value == "1" {
		return value + " " + singular
	}
	return value + " " + plural
}

func viewCountLabel(count int) string {
	if count == 1 {
		return "1 view"
	}
	return fmt.Sprintf("%d views", count)
}

func compactStringTokens(tokens []string) []string {
	out := make([]string, 0, len(tokens))
	seen := map[string]struct{}{}
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		key := strings.ToLower(token)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, token)
	}
	return out
}

func workbenchNavigatorResourceLines(root state.Root, selected state.WorkbenchTreeItem) []Line {
	row, ok := workbenchTerminalPoolItem(root, selected)
	activity := TerminalOutputActivityLabel(selected.LastOutputAt, time.Now())
	if !ok {
		lines := []Line{
			workbenchNavigatorConnectionLine(root, selected, state.TerminalPoolPageItem{}),
			workbenchNavigatorEndpointLine(selected, state.TerminalPoolPageItem{}),
		}
		if activity != "" {
			lines = append(lines, workbenchNavigatorDetailLine("last output", activity))
		}
		return lines
	}
	lines := []Line{}
	if !row.Resources.SampledAt.IsZero() {
		lines = append(lines, terminalManagerResourceGaugeLines(row)...)
	}
	lines = append(lines, workbenchNavigatorConnectionLine(root, selected, row))
	lines = append(lines, workbenchNavigatorEndpointLine(selected, row))
	if activity != "" {
		lines = append(lines, workbenchNavigatorDetailLine("last output", activity))
	}
	return lines
}

func workbenchNavigatorConnectionLine(root state.Root, selected state.WorkbenchTreeItem, row state.TerminalPoolPageItem) Line {
	ref := workbenchTreeTerminalRef(root, selected)
	count := 0
	if !ref.Empty() {
		count = len(root.TerminalViews.BindingsForTerminalRef(ref))
	}
	if row.AttachmentCount > count {
		count = row.AttachmentCount
	}
	return workbenchNavigatorDetailLine("views", viewCountLabel(maxInt(0, count)))
}

func workbenchNavigatorEndpointLine(selected state.WorkbenchTreeItem, row state.TerminalPoolPageItem) Line {
	label := selected.EndpointLabel
	if label == "" {
		label = row.EndpointLabel
	}
	status := selected.EndpointStatus
	if status == "" {
		status = row.EndpointStatus
	}
	kind := selected.EndpointErrorKind
	if kind == state.EndpointErrorUnknown {
		kind = row.EndpointErrorKind
	}
	errText := selected.EndpointLastError
	if errText == "" {
		errText = row.EndpointLastError
	}
	parts := compactStringTokens([]string{label, string(status), endpointErrorLabel(kind, errText)})
	return workbenchNavigatorDetailLine("endpoint", strings.Join(parts, " "))
}

func workbenchTerminalPoolItem(root state.Root, selected state.WorkbenchTreeItem) (state.TerminalPoolPageItem, bool) {
	ref := workbenchTreeTerminalRef(root, selected)
	if ref.Empty() {
		return state.TerminalPoolPageItem{}, false
	}
	for _, item := range state.TerminalPoolPageItems(root) {
		if state.NewTerminalRef(item.EndpointID, item.TerminalID).Equal(ref) {
			return item, true
		}
	}
	return state.TerminalPoolPageItem{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID}, false
}

func workbenchTreeTerminalRef(root state.Root, selected state.WorkbenchTreeItem) state.TerminalRef {
	switch selected.Kind {
	case state.WorkbenchTreeKindFloating:
		if selected.FloatingID != "" {
			if binding, ok := root.TerminalViews.FloatingBinding(selected.FloatingID); ok {
				return binding.TerminalRef()
			}
		}
	case state.WorkbenchTreeKindPane:
		if selected.PaneID != "" {
			if binding, ok := root.TerminalViews.PaneBinding(selected.PaneID); ok {
				return binding.TerminalRef()
			}
		}
	}
	if selected.TerminalID == "" {
		return state.TerminalRef{}
	}
	return state.LocalTerminalRef(selected.TerminalID)
}

func workbenchNavigatorPaneSnapshotLines(layout workbenchNavigatorLayout) []Line {
	lines := make([]Line, 0, layout.SnapshotHeight)
	for row := 0; row < layout.SnapshotHeight; row++ {
		lines = append(lines, NewLine(strings.Repeat(" ", layout.SnapshotWidth)))
	}
	return lines
}

func workbenchNavigatorLayoutForViewport(viewport state.ViewportStore) workbenchNavigatorLayout {
	cols := 100
	rows := 30
	if viewport.Valid && viewport.Cols > 0 {
		cols = viewport.Cols
	}
	if viewport.Valid && viewport.Rows > 0 {
		rows = viewport.Rows
	}
	overlay := measureWorkbenchNavigatorOverlay(Rect{W: cols, H: rows})
	content := measureOverlayContentRect(OverlayVM{Content: ContentVM{Kind: ContentWorkbenchTree}}, overlay)
	contentWidth := maxInt(40, content.W)
	contentHeight := maxInt(10, content.H)
	treeWidth := clampInt(contentWidth*34/100, defaultWorkbenchNavigatorTreeWidth, 64)
	if contentWidth < 96 {
		treeWidth = clampInt(contentWidth*40/100, 28, 40)
	}
	rightWidth := maxInt(20, contentWidth-treeWidth-1)
	if rightWidth < 36 && contentWidth > 44 {
		treeWidth = maxInt(24, contentWidth-37)
		rightWidth = maxInt(20, contentWidth-treeWidth-1)
	}
	maxBoostedTreeWidth := contentWidth - 1 - 36
	if maxBoostedTreeWidth < treeWidth {
		maxBoostedTreeWidth = contentWidth - 1 - 20
	}
	if maxBoostedTreeWidth > treeWidth {
		treeWidth = minInt(treeWidth+workbenchNavigatorTreeWidthBoost, maxBoostedTreeWidth)
		rightWidth = maxInt(20, contentWidth-treeWidth-1)
	}
	bodyRows := maxInt(8, contentHeight-3)
	actionRow := -1
	// 中文说明：snapshot 坐标以 overlay content rect 为原点，和最终 runtime 叠加坐标保持一致。
	snapshotWidth := maxInt(0, rightWidth-2)
	snapshotHeight := clampInt(bodyRows-6, 3, maxInt(3, bodyRows-4))
	return workbenchNavigatorLayout{
		ContentWidth:   contentWidth,
		BodyRows:       bodyRows,
		TreeWidth:      treeWidth,
		RightWidth:     rightWidth,
		ActionRow:      actionRow,
		SnapshotX:      treeWidth + 2,
		SnapshotY:      9,
		SnapshotWidth:  snapshotWidth,
		SnapshotHeight: snapshotHeight,
	}
}

func workbenchNavigatorMeta(root state.Root, rows []state.WorkbenchTreeItem, layout workbenchNavigatorLayout) ContentMetaVM {
	meta := ContentMetaVM{
		SplitPageLeftWidth: layout.TreeWidth,
		WorkbenchTreeWidth: layout.TreeWidth,
		WorkbenchBodyRows:  layout.BodyRows,
		WorkbenchActionRow: layout.ActionRow,
	}
	selected, ok := selectedWorkbenchTreeItem(rows)
	if !ok {
		return meta
	}
	meta.WorkbenchSnapshots, meta.WorkbenchSnapshotCanvas, meta.WorkbenchSnapshotSplit = workbenchNavigatorSnapshotVMs(root, selected, layout)
	if len(meta.WorkbenchSnapshots) > 0 {
		first := meta.WorkbenchSnapshots[0]
		meta.WorkbenchSnapshotPanel = &first.Panel
		meta.WorkbenchSnapshotRect = first.Rect
		meta.WorkbenchSnapshotContent = first.Content
	}
	return meta
}

type workbenchNavigatorPreviewPane struct {
	Pane         state.PaneState
	Floating     state.FloatingPaneState
	Active       bool
	FloatingMode bool
}

func workbenchNavigatorSnapshotVMs(root state.Root, selected state.WorkbenchTreeItem, layout workbenchNavigatorLayout) ([]WorkbenchSnapshotVM, Rect, SplitVM) {
	panes := workbenchNavigatorPreviewPanes(root, selected)
	if len(panes) == 0 {
		return nil, Rect{}, SplitVM{}
	}
	canvas := Rect{X: layout.TreeWidth + 1, Y: 2, W: layout.RightWidth, H: layout.BodyRows + 1}
	if canvas.W <= 0 || canvas.H <= 0 {
		return nil, Rect{}, SplitVM{}
	}
	split := workbenchNavigatorPreviewSplit(root, selected, panes)
	rects := make(map[string]Rect, len(panes))
	if len(panes) == 1 || len(split.Children) == 0 {
		rects[panes[0].Pane.ID] = canvas
		if split.PaneID == "" {
			split.PaneID = panes[0].Pane.ID
		}
	} else {
		assignSplitRects(split, canvas, rects)
	}
	showTitles := len(panes) > 1
	out := make([]WorkbenchSnapshotVM, 0, len(panes))
	for _, preview := range panes {
		rect := rects[preview.Pane.ID]
		if rect.W <= 0 || rect.H <= 0 {
			continue
		}
		rect = workbenchNavigatorFramedSplitRect(rect, canvas)
		if rect.W <= 0 || rect.H <= 0 {
			continue
		}
		panel := workbenchNavigatorSnapshotPanelForPreview(root, preview)
		content := Rect{X: rect.X + 1, Y: rect.Y + 1, W: maxInt(0, rect.W-2), H: maxInt(0, rect.H-2)}
		showTitle := showTitles && rect.W >= 8 && rect.H > 1
		out = append(out, WorkbenchSnapshotVM{
			Panel:        panel,
			Rect:         rect,
			Content:      content,
			PreviewFrame: true,
			ShowTitle:    showTitle,
		})
	}
	return out, canvas, split
}

func workbenchNavigatorFramedSplitRect(rect Rect, canvas Rect) Rect {
	if rect.X+rect.W < canvas.X+canvas.W {
		rect.W++
	}
	if rect.Y+rect.H < canvas.Y+canvas.H {
		rect.H++
	}
	return rect
}

func workbenchNavigatorPreviewSplit(root state.Root, selected state.WorkbenchTreeItem, panes []workbenchNavigatorPreviewPane) SplitVM {
	if selected.Kind == state.WorkbenchTreeKindPane || selected.Kind == state.WorkbenchTreeKindFloating {
		if len(panes) == 0 {
			return SplitVM{}
		}
		return SplitVM{PaneID: panes[0].Pane.ID}
	}
	tab, ok := workbenchNavigatorSelectedPreviewTab(root.Shell.ReadonlyDefaults(), selected)
	if !ok {
		return SplitVM{}
	}
	return buildSplitVM(tab.RootSplit)
}

func workbenchNavigatorPreviewPanes(root state.Root, selected state.WorkbenchTreeItem) []workbenchNavigatorPreviewPane {
	shell := root.Shell.ReadonlyDefaults()
	switch selected.Kind {
	case state.WorkbenchTreeKindPane:
		pane, ok := workbenchNavigatorPane(shell, selected.WorkspaceID, selected.TabID, selected.PaneID)
		if !ok {
			return nil
		}
		return []workbenchNavigatorPreviewPane{{Pane: pane, Active: selected.Active}}
	case state.WorkbenchTreeKindWorkspace, state.WorkbenchTreeKindTab:
		tab, ok := workbenchNavigatorSelectedPreviewTab(shell, selected)
		if !ok {
			return nil
		}
		out := make([]workbenchNavigatorPreviewPane, 0, len(tab.Panes))
		for _, pane := range tab.Panes {
			out = append(out, workbenchNavigatorPreviewPane{Pane: pane, Active: pane.ID == tab.ActivePaneID})
		}
		return out
	case state.WorkbenchTreeKindFloating:
		if selected.FloatingID == "" {
			return nil
		}
		floating, ok := workbenchNavigatorFloating(shell, selected.WorkspaceID, selected.TabID, selected.FloatingID)
		if !ok {
			return nil
		}
		return []workbenchNavigatorPreviewPane{{Pane: floating.Pane, Floating: floating, Active: floating.Active, FloatingMode: true}}
	default:
		return nil
	}
}

func workbenchNavigatorSelectedPreviewTab(shell state.ShellStore, selected state.WorkbenchTreeItem) (state.TabState, bool) {
	workspace, ok := workbenchNavigatorWorkspace(shell, selected.WorkspaceID)
	if !ok {
		return state.TabState{}, false
	}
	tabID := selected.TabID
	if selected.Kind == state.WorkbenchTreeKindWorkspace {
		tabID = workspace.ActiveTabID
	}
	if tab, ok := workbenchNavigatorTab(workspace, tabID); ok {
		return tab, true
	}
	if len(workspace.Tabs) > 0 {
		return workspace.Tabs[0], true
	}
	return state.TabState{}, false
}

func workbenchNavigatorSnapshotRects(layout workbenchNavigatorLayout, count int) []Rect {
	if count <= 0 || layout.SnapshotWidth <= 0 || layout.SnapshotHeight <= 0 {
		return nil
	}
	if count == 1 {
		return []Rect{{X: layout.SnapshotX, Y: layout.SnapshotY, W: layout.SnapshotWidth, H: layout.SnapshotHeight}}
	}
	gap := 1
	usable := layout.SnapshotHeight - gap*(count-1)
	if usable < count*3 {
		gap = 0
		usable = layout.SnapshotHeight
	}
	base := maxInt(1, usable/count)
	remainder := usable % count
	rects := make([]Rect, 0, count)
	y := layout.SnapshotY
	for index := 0; index < count; index++ {
		height := base
		if index < remainder {
			height++
		}
		remaining := layout.SnapshotY + layout.SnapshotHeight - y
		if height > remaining {
			height = remaining
		}
		if height <= 0 {
			break
		}
		rects = append(rects, Rect{X: layout.SnapshotX, Y: y, W: layout.SnapshotWidth, H: height})
		y += height + gap
	}
	return rects
}

func workbenchNavigatorSnapshotPanelForPreview(root state.Root, preview workbenchNavigatorPreviewPane) PanelVM {
	content := workbenchNavigatorPreviewContent(root, preview)
	pane := preview.Pane
	title := workbenchNavigatorPreviewTitle(root, preview)
	chrome := buildPanelChromeVM(root, pane, preview.Active, content)
	if preview.FloatingMode {
		chrome = PanelChromeVM{
			Title:    ChromeSlotVM{Text: title, Style: workbenchPreviewChromeStyle(preview.Active)},
			State:    paneChromeStateSlot(preview.Active, content),
			Terminal: terminalChromeVMForFloating(root, preview.Floating, content, workbenchPreviewChromeStyle(preview.Active)),
			Actions:  defaultPaneChromeActionVMs(workbenchPreviewChromeStyle(preview.Active)),
		}
	}
	return PanelVM{
		ID:           pane.ID,
		Title:        title,
		Presentation: PanelPresentationSplitLine,
		Active:       preview.Active,
		Content:      content,
		Chrome:       chrome,
	}
}

func workbenchNavigatorPreviewTitle(root state.Root, preview workbenchNavigatorPreviewPane) string {
	if terminalID := workbenchNavigatorPreviewTerminalID(root, preview); terminalID != "" {
		return workbenchNavigatorTerminalTitle(root, terminalID)
	}
	if preview.FloatingMode && strings.TrimSpace(preview.Floating.Title) != "" {
		return preview.Floating.Title
	}
	return activePaneTitle(preview.Pane)
}

func workbenchNavigatorPreviewTerminalID(root state.Root, preview workbenchNavigatorPreviewPane) string {
	if preview.FloatingMode {
		if binding, ok := root.TerminalViews.FloatingBinding(preview.Floating.ID); ok {
			return binding.TerminalID
		}
		return strings.TrimSpace(preview.Pane.TerminalID)
	}
	if binding, ok := root.TerminalViews.PaneBinding(preview.Pane.ID); ok {
		return binding.TerminalID
	}
	return strings.TrimSpace(preview.Pane.TerminalID)
}

func workbenchNavigatorTerminalTitle(root state.Root, terminalID string) string {
	for _, item := range root.TerminalPool.Items {
		if item.TerminalID != terminalID {
			continue
		}
		if title := strings.TrimSpace(item.Title); title != "" {
			return title
		}
		break
	}
	return terminalID
}

func workbenchNavigatorPreviewContent(root state.Root, preview workbenchNavigatorPreviewPane) ContentVM {
	pane := preview.Pane
	switch pane.Kind {
	case state.PaneEmpty:
		return buildEmptyPaneContent(pane)
	case state.PaneTerminalLive:
		var surface state.TerminalSurfaceStore
		var session state.TerminalSessionStore
		if preview.FloatingMode {
			surface = surfaceForFloating(root, preview.Floating.ID)
			session = sessionForFloating(root, preview.Floating.ID)
		} else {
			surface, session = terminalContentStoresForPane(root, pane)
		}
		content := buildLiveContentVM(surface, session)
		if preview.FloatingMode {
			return contentWithFloatingLayout(root, preview.Floating, content)
		}
		return contentWithPaneLayout(root, pane, content)
	default:
		return placeholderContentForPane(pane)
	}
}

func workbenchPreviewChromeStyle(active bool) StyleToken {
	if active {
		return StyleAccent
	}
	return StyleMuted
}

func workbenchNavigatorWorkspace(shell state.ShellStore, workspaceID string) (state.WorkspaceState, bool) {
	shell = shell.ReadonlyDefaults()
	if workspaceID == "" || workspaceID == shell.Workspace.ID {
		return shell.Workspace, true
	}
	for _, workspace := range shell.Workspaces {
		if workspace.ID == workspaceID {
			return workspace, true
		}
	}
	return state.WorkspaceState{}, false
}

func workbenchNavigatorTab(workspace state.WorkspaceState, tabID string) (state.TabState, bool) {
	for _, tab := range workspace.Tabs {
		if tab.ID == tabID {
			return tab, true
		}
	}
	return state.TabState{}, false
}

func workbenchNavigatorPane(shell state.ShellStore, workspaceID string, tabID string, paneID string) (state.PaneState, bool) {
	workspace, ok := workbenchNavigatorWorkspace(shell, workspaceID)
	if !ok {
		return state.PaneState{}, false
	}
	for _, tab := range workspace.Tabs {
		if tabID != "" && tab.ID != tabID {
			continue
		}
		for _, pane := range tab.Panes {
			if pane.ID == paneID {
				return pane, true
			}
		}
	}
	return state.PaneState{}, false
}

func workbenchNavigatorFloating(shell state.ShellStore, workspaceID string, tabID string, floatingID string) (state.FloatingPaneState, bool) {
	workspace, ok := workbenchNavigatorWorkspace(shell, workspaceID)
	if !ok {
		return state.FloatingPaneState{}, false
	}
	for _, tab := range workspace.Tabs {
		if tabID != "" && tab.ID != tabID {
			continue
		}
		for _, floating := range tab.Floatings {
			if floating.ID == floatingID {
				return floating, true
			}
		}
	}
	return state.FloatingPaneState{}, false
}

func workbenchTreeHitRegions(rows []state.WorkbenchTreeItem, rowOffset int, treeWidth int) []HitRegion {
	regions := make([]HitRegion, 0, len(rows)+1)
	for index := range rows {
		regions = append(regions, HitRegion{
			Kind:   HitRegionContentAction,
			Rect:   Rect{Y: rowOffset + index, W: treeWidth, H: 1},
			Row:    index,
			HasRow: true,
			// 中文说明：Workbench Navigator 的鼠标行点击只带 row，由 reducer 用 WorkbenchTreeItem 决定 workspace/tab/pane/floating 真实目标。
			ActionID:   ActionWorkbenchOpen.String(),
			Invocation: invocationForProjection(ActionWorkbenchOpen),
			TargetMode: HitTargetExplicit,
		})
	}
	return regions
}

func workbenchNavigatorDetailHitRegions(root state.Root, rows []state.WorkbenchTreeItem, layout workbenchNavigatorLayout) []HitRegion {
	_, ok := selectedWorkbenchTreeItem(rows)
	if !ok || layout.RightWidth <= 0 {
		return nil
	}
	rightX := layout.TreeWidth + 1
	// 中文说明：右侧 live canvas 只声明打开当前 selected node；不在预览层引入第二套 pane 选择状态。
	return []HitRegion{{
		Kind:       HitRegionContentAction,
		Rect:       Rect{X: rightX, Y: 2, W: layout.RightWidth, H: layout.BodyRows + 1},
		Row:        -1,
		HasRow:     true,
		ActionID:   ActionWorkbenchOpen.String(),
		Invocation: invocationForProjection(ActionWorkbenchOpen),
		TargetMode: HitTargetExplicit,
	}}
}

func workbenchTreeKindLabel(row state.WorkbenchTreeItem) string {
	switch row.Kind {
	case state.WorkbenchTreeKindWorkspace:
		return "workspace"
	case state.WorkbenchTreeKindTab:
		return "tab"
	case state.WorkbenchTreeKindPane:
		return "pane"
	case state.WorkbenchTreeKindFloating:
		return "floating"
	default:
		return row.Kind
	}
}

func workbenchTreeKindGlyph(row state.WorkbenchTreeItem) string {
	switch row.Kind {
	case state.WorkbenchTreeKindWorkspace:
		return "W"
	case state.WorkbenchTreeKindTab:
		return "T"
	case state.WorkbenchTreeKindPane:
		return "P"
	case state.WorkbenchTreeKindFloating:
		return "F"
	default:
		return workbenchTreeKindLabel(row)
	}
}

func workbenchTreeKindStyle(row state.WorkbenchTreeItem) StyleToken {
	if row.Selected {
		return StyleAccent
	}
	switch row.Kind {
	case state.WorkbenchTreeKindWorkspace:
		return StyleStatusAccent
	case state.WorkbenchTreeKindTab:
		if row.Active {
			return StyleAccent
		}
		return StyleStrongForeground
	case state.WorkbenchTreeKindPane:
		switch row.PaneKind {
		case state.PaneEmpty:
			return StyleWarning
		case state.PaneTerminalLive:
			return StyleSuccess
		default:
			return StyleForeground
		}
	case state.WorkbenchTreeKindFloating:
		switch row.PaneKind {
		case state.PaneEmpty:
			return StyleWarning
		case state.PaneTerminalLive:
			return StyleSuccess
		default:
			return StyleForeground
		}
	default:
		return StyleForeground
	}
}

func workbenchTreeTitleStyle(row state.WorkbenchTreeItem) StyleToken {
	if row.Selected {
		return StyleAccent
	}
	if row.Active {
		return StyleStrongForeground
	}
	if row.Kind == state.WorkbenchTreeKindFloating {
		return StyleForeground
	}
	return StyleForeground
}

func workbenchTreeTitleCell(title string, row state.WorkbenchTreeItem) Cell {
	cell := styledCell(title, workbenchTreeTitleStyle(row))
	if row.Selected {
		cell.ANSIStyle.Underline = true
	}
	return cell
}

func workbenchTreeTitle(row state.WorkbenchTreeItem) string {
	switch row.Kind {
	case state.WorkbenchTreeKindWorkspace:
		if row.WorkspaceName != "" {
			return row.WorkspaceName
		}
		return row.WorkspaceID
	case state.WorkbenchTreeKindTab:
		if row.TabTitle != "" {
			return row.TabTitle
		}
		return row.TabID
	case state.WorkbenchTreeKindPane:
		if row.DisplayTitle != "" {
			return row.DisplayTitle
		}
		if row.PaneTitle != "" {
			return row.PaneTitle
		}
		return row.PaneID
	case state.WorkbenchTreeKindFloating:
		if row.DisplayTitle != "" {
			return row.DisplayTitle
		}
		if row.FloatingTitle != "" {
			return row.FloatingTitle
		}
		if row.PaneTitle != "" {
			return row.PaneTitle
		}
		if row.FloatingID != "" {
			return row.FloatingID
		}
		return "floating panes"
	default:
		return "node"
	}
}

func workbenchTreePath(row state.WorkbenchTreeItem) string {
	parts := []string{}
	if row.WorkspaceName != "" {
		parts = append(parts, "ws:"+row.WorkspaceName)
	}
	if row.TabTitle != "" {
		parts = append(parts, "tab:"+row.TabTitle)
	}
	if row.FloatingTitle != "" {
		parts = append(parts, "floating:"+row.FloatingTitle)
	}
	if row.PaneTitle != "" {
		parts = append(parts, "pane:"+row.PaneTitle)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " / ")
}

func workbenchTreePreview(row state.WorkbenchTreeItem) string {
	if row.Kind == state.WorkbenchTreeKindFloating {
		if row.FloatingID != "" {
			return row.Summary
		}
		return "no floating panes"
	}
	if row.Summary != "" {
		return row.Summary
	}
	return "ready"
}

func workbenchPaneStateLabel(root state.Root, selected state.WorkbenchTreeItem) string {
	binding, hasBinding := root.TerminalViews.PaneBinding(selected.PaneID)
	surface := state.TerminalSurfaceStore{}
	session := state.TerminalSessionStore{}
	if hasBinding && binding.TerminalID != "" {
		surface = surfaceForBinding(root, binding)
		session = sessionForBinding(root, binding)
	}
	switch {
	case session.LastError != "" || surface.Err != "":
		return "error"
	case session.State == state.TerminalLiveExited || surface.State == state.TerminalLiveExited:
		return "exited"
	case hasBinding && binding.Attached:
		return "running"
	case hasBinding && binding.TerminalID != "":
		return "bound"
	case selected.TerminalID != "":
		return "bound"
	case selected.PaneKind == state.PaneEmpty:
		return "empty"
	default:
		return string(selected.PaneKind)
	}
}

func workbenchPaneRoleLabel(root state.Root, selected state.WorkbenchTreeItem) string {
	binding, ok := root.TerminalViews.PaneBinding(selected.PaneID)
	if !ok || binding.ResizeRole == "" {
		return ""
	}
	return binding.ResizeRole
}

func workbenchTreeStatus(count int, query string) string {
	if query == "" {
		return fmt.Sprintf("workbench tree: %d items", count)
	}
	return fmt.Sprintf("workbench tree: %d items query:%s", count, query)
}

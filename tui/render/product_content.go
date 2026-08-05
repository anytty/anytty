package render

import (
	"fmt"
	"strings"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
)

const contentActionWidth = 12

const floatingOverviewTitleWidth = 24
const floatingOverviewStateWidth = 12
const floatingOverviewSizeWidth = 8
const floatingOverviewIDWidth = 18
const defaultTerminalManagerListWidth = 40
const terminalManagerListWidthBoost = 8

type terminalManagerLayout struct {
	ContentWidth   int
	BodyRows       int
	ListWidth      int
	DetailWidth    int
	SnapshotX      int
	SnapshotY      int
	SnapshotWidth  int
	SnapshotHeight int
}

// empty pane 内容只描述当前 pane 可执行的产品动作，不创建 terminal。
func buildEmptyPaneContent(pane state.PaneState) ContentVM {
	lines, regions, cursor := emptyPaneContentLayout(pane.ID, 0)
	return ContentVM{
		Kind:       ContentEmptyPane,
		Lines:      lines,
		Status:     "not connected: choose or create terminal",
		Cursor:     cursor,
		Empty:      true,
		HitRegions: regions,
	}
}

func buildEmptyPaneContentWithSelection(pane state.PaneState, selectedIndex int) ContentVM {
	lines, regions, cursor := emptyPaneContentLayout(pane.ID, selectedIndex)
	return ContentVM{
		Kind:       ContentEmptyPane,
		Lines:      lines,
		Status:     "not connected: choose or create terminal",
		Cursor:     cursor,
		Empty:      true,
		HitRegions: regions,
	}
}

func emptyPaneContentLayout(paneID string, selectedIndex int) ([]Line, []HitRegion, Cursor) {
	actions := emptyPaneActions()
	if selectedIndex < 0 || selectedIndex >= len(actions) {
		selectedIndex = 0
	}
	lines := []Line{
		centeredStyledLine("○ No terminal connected", StyleForeground),
		centeredStyledLine("Choose a terminal or create one.", StyleMuted),
		NewLine(""),
		centeredStyledLine("Choose how to start", StyleMuted),
	}
	regions := make([]HitRegion, 0, len(actions))
	for index, action := range actions {
		selected := index == selectedIndex
		text := emptyPaneActionLabel(action.Label, selected)
		style := action.Style
		if selected && style == StyleForeground {
			style = StyleStrongForeground
		}
		line := centeredStyledLine(text, style)
		lines = append(lines, line)
		regions = append(regions, HitRegion{Kind: HitRegionContentAction, Rect: Rect{Y: len(lines) - 1, W: DisplayWidth(text), H: 1}, PaneID: paneID, ActionID: action.ID.String(), Invocation: invocationForProjection(action.ID), TargetMode: HitTargetExplicit})
	}
	return lines, regions, Cursor{}
}

type emptyPaneProjectionSpec struct {
	ID    ProjectionID
	Label string
	Style StyleToken
}

func emptyPaneActions() []emptyPaneProjectionSpec {
	labels := map[actiondomain.ID]string{
		actiondomain.ActionEmptyAttach:  "Attach existing terminal",
		actiondomain.ActionEmptyCreate:  "Create new terminal",
		actiondomain.ActionEmptyManager: "Open terminal manager",
		actiondomain.ActionEmptyClose:   "Close pane",
	}
	styles := map[actiondomain.ID]StyleToken{
		actiondomain.ActionEmptyAttach: StyleAccent, actiondomain.ActionEmptyCreate: StyleSuccess,
		actiondomain.ActionEmptyManager: StyleForeground, actiondomain.ActionEmptyClose: StyleDangerStrong,
	}
	out := make([]emptyPaneProjectionSpec, 0, len(labels))
	for _, id := range actiondomain.EmptyPaneCTAActions() {
		out = append(out, emptyPaneProjectionSpec{ID: ProjectionID(id), Label: labels[id], Style: styles[id]})
	}
	return out
}

func emptyPaneActionLabel(label string, selected bool) string {
	label = strings.TrimSpace(label)
	if selected {
		return "► " + label + " ◄"
	}
	return "[ " + label + " ]"
}

type liveExitedProjectionSpec struct {
	ID    ProjectionID
	Label string
	Style StyleToken
}

type liveDisconnectedProjectionSpec struct {
	ID    ProjectionID
	Label string
	Style StyleToken
}

const (
	terminalPickerTitleColumnWidth    = 22
	terminalPickerEndpointColumnWidth = 14
	terminalPickerStateColumnWidth    = 8
	terminalPickerSizeColumnWidth     = 7
	terminalPickerActionColumnWidth   = 6
	terminalPickerColumnGapWidth      = 2
	terminalPickerPrefixWidth         = 4
	terminalPickerHitRegionWidth      = terminalPickerPrefixWidth +
		terminalPickerTitleColumnWidth +
		terminalPickerColumnGapWidth +
		terminalPickerEndpointColumnWidth +
		terminalPickerColumnGapWidth +
		terminalPickerStateColumnWidth +
		terminalPickerColumnGapWidth +
		terminalPickerSizeColumnWidth +
		terminalPickerColumnGapWidth +
		terminalPickerActionColumnWidth
)

func liveExitedActions() []liveExitedProjectionSpec {
	styles := map[actiondomain.ID]StyleToken{
		actiondomain.ActionExitedRestart: StyleWarning, actiondomain.ActionExitedReconnect: StyleMuted,
	}
	out := make([]liveExitedProjectionSpec, 0, len(styles))
	for _, id := range actiondomain.ExitedPaneCTAActions() {
		spec, _ := ProjectionByID(ProjectionID(id))
		out = append(out, liveExitedProjectionSpec{ID: ProjectionID(id), Label: projectionActionLabel(spec), Style: styles[id]})
	}
	return out
}

func liveDisconnectedActions() []liveDisconnectedProjectionSpec {
	labels := map[actiondomain.ID]string{
		actiondomain.ActionDisconnectedReconnect:  "Reconnect this pane",
		actiondomain.ActionDisconnectedDisconnect: "Disconnect pane",
	}
	styles := map[actiondomain.ID]StyleToken{
		actiondomain.ActionDisconnectedReconnect:  StyleAccent,
		actiondomain.ActionDisconnectedDisconnect: StyleDangerStrong,
	}
	out := make([]liveDisconnectedProjectionSpec, 0, len(labels))
	for _, id := range actiondomain.DisconnectedPaneCTAActions() {
		out = append(out, liveDisconnectedProjectionSpec{ID: ProjectionID(id), Label: labels[id], Style: styles[id]})
	}
	return out
}

func centeredStyledLine(text string, style StyleToken) Line {
	return Line{Cells: []Cell{styledCell(text, style)}}
}

// empty tab 是 workspace/tab truth，不伪造 pane；用户动作再创建或连接真实 pane。
func buildEmptyTabContent(tab state.TabState) ContentVM {
	title := strings.TrimSpace(tab.Title)
	if title == "" {
		title = tab.ID
	}
	if title == "" {
		title = "tab"
	}
	lines := []Line{
		{Cells: []Cell{styledCell("No panel in tab ", StyleMuted), styledCell(title, StyleAccent)}},
		NewLine(""),
		contentActionLine("attach", "Choose terminal"),
		contentActionLine("create", "New terminal"),
		contentActionLine("manager", "Terminal Manager"),
	}
	return ContentVM{
		Kind:       ContentEmptyPane,
		Lines:      lines,
		Status:     "empty tab: Choose terminal / New terminal / Terminal Manager",
		Empty:      true,
		HitRegions: contentActionRegions([]ProjectionID{ActionEmptyAttach, ActionEmptyCreate, ActionEmptyManager}, "", 2),
	}
}

// empty workspace 表示用户已关闭全部 tab；这里不伪造 tab/pane，只提示全局入口。
func buildEmptyWorkspaceContent(workspace state.WorkspaceState) ContentVM {
	lines := []Line{
		centeredStyledLine("No tabs in this workspace", StyleForeground),
		contentActionLine("picker", "Choose terminal"),
		contentActionLine("tab", "New tab"),
	}
	return ContentVM{
		Kind:   ContentEmptyPane,
		Lines:  lines,
		Status: "empty workspace: Choose terminal / New tab",
		Empty:  true,
		HitRegions: []HitRegion{
			{Kind: HitRegionContentAction, Rect: Rect{Y: 1, W: lines[1].Width(), H: 1}, ActionID: "menu.terminal_picker", Invocation: actiondomain.Invocation{ID: "menu.terminal_picker", SourceActionID: "menu.terminal_picker"}, TargetMode: HitTargetActive},
			{Kind: HitRegionContentAction, Rect: Rect{Y: 2, W: lines[2].Width(), H: 1}, ActionID: "tab.create", Invocation: actiondomain.Invocation{ID: "tab.create", SourceActionID: "tab.create"}, TargetMode: HitTargetActive},
		},
	}
}

// Terminal Picker 只消费 reducer-owned root；服务端 terminal list 必须先回投 TerminalPoolStore。
func buildTerminalPickerContent(root state.Root, shell state.ShellStore) ContentVM {
	shell = shell.ReadonlyDefaults()
	query := strings.TrimSpace(shell.Overlay.Query)
	lines := []Line{terminalPickerSearchLine(query)}
	if poolLine, ok := terminalPoolStateLine(root.TerminalPool); ok {
		lines = append(lines, poolLine)
	}
	lines = append(lines, terminalPickerManagedEndpointLines(root)...)
	rowOffset := len(lines)
	rows := state.TerminalPickerItems(root)
	for _, row := range rows {
		lines = append(lines, terminalPickerLine(row, query))
	}
	return ContentVM{
		Kind:       ContentTerminalPicker,
		Lines:      lines,
		Status:     terminalPickerStatus(terminalPickerSelectableCount(rows), query),
		Cursor:     Cursor{Visible: true, Row: 0, Col: terminalPickerSearchCursorCol(query), Shape: CursorShapeBar},
		HitRegions: terminalPickerHitRegions(rows, rowOffset),
	}
}

func terminalPickerManagedEndpointLines(root state.Root) []Line {
	groups := state.TerminalPickerGroups(root)
	lines := make([]Line, 0, len(groups))
	for _, group := range groups {
		if group.Transport != state.EndpointTransportHubP2P {
			continue
		}
		lines = append(lines, terminalPickerEndpointHeaderLine(group))
	}
	return lines
}

func chromeSafeViewportForShell(viewport state.ViewportStore, shell state.ShellStore) state.ViewportStore {
	if !viewport.Valid || viewport.Rows <= 0 {
		return viewport
	}
	if shell.HeaderVisible && viewport.Rows > 0 {
		viewport.Rows--
	}
	if shell.FooterVisible && viewport.Rows > 0 {
		viewport.Rows--
	}
	return viewport
}

// Floating Overview 只投影 reducer-owned floating 列表；打开/召回通过 ActionID 回到 app reducer。
func buildFloatingOverviewContent(root state.Root, _ state.ShellStore) ContentVM {
	rows := state.FloatingOverviewItems(root)
	lines := []Line{floatingOverviewHeaderLine()}
	rowOffset := len(lines)
	for index, row := range rows {
		lines = append(lines, floatingOverviewRowLine(index, row))
	}
	if len(rows) == 0 {
		lines = append(lines, Line{Cells: []Cell{styledCell("No floating terminals", StyleMuted)}})
	}
	regions := floatingOverviewHitRegions(rows, rowOffset)
	return ContentVM{
		Kind:       ContentFloatingOverview,
		Lines:      lines,
		Status:     floatingOverviewStatus(len(rows)),
		Cursor:     Cursor{Visible: false},
		HitRegions: regions,
		Empty:      len(rows) == 0,
	}
}

// Prompt 是 reducer-owned 表单 overlay；提交只回投 shell message，不直接执行业务 IO。
func buildPromptContent(shell state.ShellStore) ContentVM {
	shell = shell.ReadonlyDefaults()
	prompt := shell.Overlay.Prompt
	title := prompt.Title
	if title == "" {
		title = "Command Prompt"
	}
	if len(prompt.Fields) > 0 {
		return buildPromptFormContent(prompt, title)
	}
	if prompt.Purpose == "terminal.rename" {
		return buildPromptSingleInputContent(prompt)
	}
	placeholder := prompt.Placeholder
	if placeholder == "" {
		placeholder = "command"
	}
	value := prompt.Value
	displayValue := value
	if displayValue == "" {
		displayValue = "[" + placeholder + "]"
	}
	lines := []Line{
		pageTitleLine(title, ""),
		formFieldLine("Name", displayValue, value != ""),
	}
	if prompt.Destructive {
		lines = append(lines, Line{Cells: []Cell{styledCell(" ! confirm ", StyleWarning), NewCell("type " + prompt.ConfirmText + " before submit")}})
	}
	return ContentVM{
		Kind:   ContentPrompt,
		Lines:  lines,
		Status: "prompt",
		Cursor: Cursor{Visible: true, Row: 1, Col: DisplayWidth("Name ") + DisplayWidth(value), Shape: CursorShapeBar},
	}
}

func buildPromptSingleInputContent(prompt state.PromptState) ContentVM {
	placeholder := prompt.Placeholder
	if placeholder == "" {
		placeholder = "value"
	}
	value := prompt.Value
	displayValue := value
	valueSet := value != ""
	if displayValue == "" {
		displayValue = "[" + placeholder + "]"
	}
	valueStyle := StyleForeground
	if !valueSet {
		valueStyle = StyleStrongForeground
	}
	return ContentVM{
		Kind:   ContentPrompt,
		Lines:  []Line{{Cells: []Cell{styledCell(displayValue, valueStyle)}}},
		Status: "prompt",
		Cursor: Cursor{Visible: true, Row: 0, Col: DisplayWidth(value), Shape: CursorShapeBar},
	}
}

func buildPromptFormContent(prompt state.PromptState, title string) ContentVM {
	lines := []Line{pageTitleLine(title, "")}
	activeField := prompt.ActiveField
	if activeField < 0 {
		activeField = 0
	}
	if activeField >= len(prompt.Fields) {
		activeField = len(prompt.Fields) - 1
	}
	cursorRow := 1 + activeField
	cursorCol := 0
	for index, field := range prompt.Fields {
		active := index == activeField
		lines = append(lines, promptFormFieldLine(field, active))
		if active {
			cursorCol = promptFormFieldValueCol(field) + promptFieldCursorDisplayWidth(field)
		}
	}
	return ContentVM{
		Kind:   ContentPrompt,
		Lines:  lines,
		Status: "prompt",
		Cursor: Cursor{Visible: true, Row: cursorRow, Col: cursorCol, Shape: CursorShapeBar},
	}
}

func promptFormFieldLine(field state.PromptFieldState, active bool) Line {
	label := field.Label
	if label == "" {
		label = field.Key
	}
	if field.Required {
		label += "*"
	}
	value := field.Value
	valueSet := value != ""
	if value == "" && field.Placeholder != "" {
		value = "[" + field.Placeholder + "]"
	}
	labelStyle := StyleStrongForeground
	if active {
		labelStyle = StyleAccent
	}
	valueStyle := StyleForeground
	if !valueSet {
		valueStyle = StyleStrongForeground
	}
	return Line{Cells: []Cell{
		styledCell(label+": ", labelStyle),
		styledCell(value, valueStyle),
	}}
}

func promptFormFieldValueCol(field state.PromptFieldState) int {
	label := field.Label
	if label == "" {
		label = field.Key
	}
	if field.Required {
		label += "*"
	}
	return DisplayWidth(label + ": ")
}

func promptFieldCursorDisplayWidth(field state.PromptFieldState) int {
	runes := []rune(field.Value)
	cursor := field.Cursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	return DisplayWidth(string(runes[:cursor]))
}

func buildHelpContent(root state.Root) ContentVM {
	entries := input.ShortcutEntriesForHelp(root.Config.Shortcuts, root.HostCapabilities.KeyboardDisambiguation)
	contentWidth, contentHeight := helpContentDimensions(root)
	shell := root.Shell.ReadonlyDefaults()
	selected := shell.Overlay.SelectedIndex
	if selected < 0 {
		selected = 0
	}
	if selected >= len(entries) && len(entries) > 0 {
		selected = len(entries) - 1
	}
	showClose := shortcutSceneHasAction(root.Config.Shortcuts, "help", "help.close")
	renderClose := showClose && contentHeight > 1
	visibleRows := maxInt(0, contentHeight-1)
	if renderClose {
		visibleRows = maxInt(0, visibleRows-1)
	}
	start, end := helpShortcutWindow(selected, len(entries), visibleRows)
	subtitle := fmt.Sprintf("shortcuts %d/%d", selected+1, len(entries))
	if len(entries) == 0 {
		subtitle = "no configured shortcuts"
	}
	lines := []Line{pageTitleLine("Help", subtitle)}
	for index := start; index < end; index++ {
		lines = append(lines, helpShortcutLine(entries[index], index == selected, contentWidth, root.Config))
	}
	if len(entries) == 0 && len(lines) < contentHeight {
		lines = append(lines, NewLine("No shortcuts are available in the active catalog."))
	}
	status := fmt.Sprintf("help: shortcuts %d-%d of %d", start+1, end, len(entries))
	if len(entries) == 0 {
		status = "help: no configured shortcuts"
	}
	content := ContentVM{
		Kind:   ContentHelp,
		Lines:  lines,
		Status: status,
		Cursor: Cursor{Visible: false},
	}
	if renderClose {
		closeKey := shortcutSceneActionKey(root.Config.Shortcuts, "help", "help.close")
		content.Lines = append(content.Lines, contentActionLine(closeKey, "Close Help"))
		content.HitRegions = []HitRegion{{
			Kind:       HitRegionContentAction,
			Rect:       Rect{Y: len(content.Lines) - 1, W: content.Lines[len(content.Lines)-1].Width(), H: 1},
			ActionID:   ActionHelpClose.String(),
			Invocation: invocationForProjection(ActionHelpClose),
			TargetMode: HitTargetExplicit,
		}}
	}
	return content
}

func shortcutSceneActionKey(shortcuts state.TUIShortcutConfig, scene string, actionID string) string {
	for _, entry := range input.ShortcutEntriesForScene(shortcuts, scene) {
		if entry.ActionID == actionID {
			if entry.KeyLabel != "" {
				return entry.KeyLabel
			}
			return input.ShortcutKeyDisplay(entry.Key)
		}
	}
	return ""
}

func helpContentDimensions(root state.Root) (int, int) {
	viewport := viewportRect(root.Viewport)
	viewport.W = normalizeViewportDimension(viewport.W, defaultWidth)
	viewport.H = normalizeViewportDimension(viewport.H, defaultHeight)
	shell := root.Shell.ReadonlyDefaults()
	body := viewport
	if shell.HeaderVisible && body.H > 0 {
		body.H--
	}
	if shell.FooterVisible && body.H > 0 {
		body.H--
	}
	overlay := measurePageOverlay(body)
	content := measureOverlayContentRect(OverlayVM{Content: ContentVM{Kind: ContentHelp}}, overlay)
	return maxInt(1, content.W), maxInt(1, content.H)
}

func helpShortcutWindow(selected int, itemCount int, visibleRows int) (int, int) {
	if itemCount <= 0 || visibleRows <= 0 {
		return 0, 0
	}
	visibleRows = minInt(visibleRows, itemCount)
	start := selected - visibleRows/2
	if start < 0 {
		start = 0
	}
	if start+visibleRows > itemCount {
		start = itemCount - visibleRows
	}
	return start, start + visibleRows
}

func helpShortcutLine(entry input.ShortcutEntry, selected bool, width int, cfg state.TUIConfigStore) Line {
	action, ok := shortcutActionFromShortcutEntry(entry, cfg, true)
	if !ok {
		return Line{}
	}
	marker := "  "
	markerStyle := StyleMuted
	if selected {
		marker = "> "
		markerStyle = StyleAccent
	}
	sceneWidth := 20
	keyWidth := 14
	if width < 56 {
		sceneWidth = 14
		keyWidth = 12
	}
	if width < 38 {
		sceneWidth = 9
		keyWidth = 10
	}
	labelWidth := maxInt(1, width-DisplayWidth(marker)-sceneWidth-keyWidth)
	return Line{Cells: []Cell{
		styledCell(marker, markerStyle),
		styledCell(PadRightCells(helpSceneLabel(entry.Scene), sceneWidth), StyleMuted),
		styledCell(PadRightCells(formatFooterKeyToken(action.Key), keyWidth), action.Style),
		styledCell(TruncateCells(action.Label, labelWidth), StyleForeground),
	}}
}

func helpSceneLabel(scene string) string {
	labels := map[string]string{
		"global": "Most used", "panel": "Pane", "resize": "Resize", "system": "Shell",
		"floating": "Floating", "tab": "Tab", "workspace": "Workspace", "copy": "Display / Copy",
		"terminal_picker": "Terminal Picker", "terminal_pool": "Terminal Manager", "connections": "Connections",
		"workbench_tree": "Workbench Tree", "clipboard_history": "Clipboard History",
		"floating_overview": "Floating Overview", "prompt": "Prompt", "help": "Help",
	}
	if label := labels[shortcutCatalogScene(scene)]; label != "" {
		return label
	}
	return scene
}

func shortcutSceneHasAction(shortcuts state.TUIShortcutConfig, scene string, actionID string) bool {
	for _, entry := range input.ShortcutEntriesForScene(shortcuts, scene) {
		if entry.ActionID == actionID {
			return true
		}
	}
	return false
}

func terminalPickerLine(row state.TerminalPickerItem, query string) Line {
	marker := "  "
	textStyle := StylePicker
	markerStyle := StylePicker
	if row.Selected {
		marker = "▸ "
		markerStyle = StylePickerAccent
	}
	if row.CreateNew {
		endpointLabel := terminalPickerEndpointLabel(row)
		cells := []Cell{
			styledCell(marker, markerStyle),
			styledCell("+", StylePickerInfo),
			pickerSpace(" "),
		}
		cells = append(cells, terminalPickerColumnCells(row.Title, query, textStyle, terminalPickerTitleColumnWidth)...)
		cells = append(cells, pickerSpace("  "))
		cells = append(cells, terminalPickerColumnCells(endpointLabel, query, StylePickerInfo, terminalPickerEndpointColumnWidth)...)
		cells = append(cells, pickerSpace("  "))
		cells = append(cells, terminalPickerColumnCells("-", query, StylePickerMuted, terminalPickerStateColumnWidth)...)
		cells = append(cells, pickerSpace("  "))
		cells = append(cells, highlightPickerText("Create terminal", query, textStyle)...)
		return Line{Cells: cells}
	}
	stateText := terminalPickerStateLabel(row)
	sizeText := terminalPickerSizeLabel(row)
	if sizeText == "" {
		sizeText = "-"
	}
	endpointLabel := terminalPickerEndpointLabel(row)
	cells := []Cell{
		styledCell(marker, markerStyle),
		styledCell("●", terminalPoolStateStyle(stateText)),
		pickerSpace(" "),
	}
	cells = append(cells, terminalPickerColumnCells(row.Title, query, textStyle, terminalPickerTitleColumnWidth)...)
	cells = append(cells, pickerSpace("  "))
	cells = append(cells, terminalPickerColumnCells(endpointLabel, query, StylePickerInfo, terminalPickerEndpointColumnWidth)...)
	cells = append(cells, pickerSpace("  "))
	cells = append(cells, terminalPickerColumnCells(stateText, query, terminalPoolStateStyle(stateText), terminalPickerStateColumnWidth)...)
	cells = append(cells, pickerSpace("  "))
	cells = append(cells, terminalPickerColumnCells(sizeText, query, textStyle, terminalPickerSizeColumnWidth)...)
	cells = append(cells, pickerSpace("  "))
	cells = append(cells, terminalPickerColumnCells("Attach", query, StylePickerMuted, terminalPickerActionColumnWidth)...)
	return Line{Cells: cells}
}

func terminalPickerEndpointLabel(row state.TerminalPickerItem) string {
	if row.EndpointLabel != "" {
		return row.EndpointLabel
	}
	if row.EndpointID != "" {
		return string(row.EndpointID)
	}
	return "-"
}

func terminalPickerEndpointHeaderLine(group state.EndpointPickerGroup) Line {
	status := string(group.Status)
	if group.Status == state.EndpointStatusConnecting && group.ConnectionPhase != "" {
		status = string(group.ConnectionPhase)
	}
	if status == "" {
		status = "unknown"
	}
	countLabel := fmt.Sprintf("%d terminals", group.TerminalCount)
	if group.TerminalCount == 1 {
		countLabel = "1 terminal"
	}
	cells := []Cell{
		styledCell("  ", StyleMuted),
		styledCell(group.Label, StyleStrongForeground),
		NewCell(" "),
		tokenCell(status, endpointStatusStyle(group.Status)),
		NewCell(" "),
		tokenCell(string(group.ConnectMode), StyleMuted),
		NewCell(" "),
		tokenCell(string(group.Transport), StyleMuted),
		NewCell(" "),
		styledCell(countLabel, StyleMuted),
	}
	if group.ObservedPath != "" {
		cells = append(cells, NewCell(" "), tokenCell(group.ObservedPath, StyleAccent))
	}
	if group.RouteSelectionReason != "" {
		cells = append(cells, NewCell(" "), styledCell("("+group.RouteSelectionReason+")", StyleMuted))
	}
	if group.LastError != "" {
		cells = append(cells, NewCell(" "), styledCell(endpointErrorLabel(group.ErrorKind, group.LastError), StyleWarning))
	}
	return Line{Cells: cells}
}

func terminalPickerColumnCells(value string, query string, baseStyle StyleToken, width int) []Cell {
	if width <= 0 {
		return nil
	}
	// 中文说明：picker 行的列边界属于 render view-model 展示契约；长名称只能在本列内截断，
	// 不能把 endpoint、状态或尺寸列挤偏，否则同名 terminal 的机器归属会失去可比性。
	value = TruncateCells(value, width)
	cells := highlightPickerText(value, query, baseStyle)
	pad := width - DisplayWidth(value)
	if pad > 0 {
		cells = append(cells, pickerSpace(strings.Repeat(" ", pad)))
	}
	return cells
}

func terminalPickerStateLabel(row state.TerminalPickerItem) string {
	if strings.TrimSpace(row.PoolState) != "" {
		return row.PoolState
	}
	switch row.Kind {
	case state.PaneTerminalLive:
		return "live"
	case state.PaneEmpty:
		return "empty"
	default:
		return string(row.Kind)
	}
}

func terminalPickerSizeLabel(row state.TerminalPickerItem) string {
	if row.Cols <= 0 || row.Rows <= 0 {
		return ""
	}
	return fmt.Sprintf("%dx%d", row.Cols, row.Rows)
}

func terminalPickerSearchLine(query string) Line {
	if query == "" {
		return Line{Cells: []Cell{
			styledCell("search:", StylePickerAccent),
		}}
	}
	return Line{Cells: []Cell{
		styledCell("search: ", StylePickerAccent),
		styledCell(query, StylePickerAccent),
	}}
}

func highlightPickerText(value string, query string, baseStyle StyleToken) []Cell {
	if value == "" {
		return nil
	}
	matchIndexes := state.TerminalPickerQueryMatchIndexes(value, query)
	if matchIndexes == nil || len(matchIndexes) == 0 {
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

func pickerSpace(value string) Cell {
	return styledCell(value, StylePicker)
}

func terminalPickerSearchCursorCol(query string) int {
	if query == "" {
		return DisplayWidth("search:")
	}
	return DisplayWidth("search: ") + DisplayWidth(query)
}

func terminalManagerFullLine(line Line, layout terminalManagerLayout) Line {
	return fitContentLine(line, layout.ContentWidth, StyleForeground)
}

func terminalManagerDividerLine(layout terminalManagerLayout) Line {
	return NewLine(strings.Repeat(" ", layout.ContentWidth))
}

func terminalManagerBodyLine(left Line, right Line, layout terminalManagerLayout) Line {
	cells := fitContentLine(left, layout.ListWidth, StyleForeground).Cells
	cells = append(cells, NewCell(" "))
	cells = append(cells, fitContentLine(right, layout.DetailWidth, StyleForeground).Cells...)
	return Line{Cells: cells}
}

func terminalManagerHeaderLine(label string) Line {
	return Line{Cells: []Cell{styledCell(label, StyleStrongForeground)}}
}

func floatingOverviewRowLine(index int, row state.FloatingOverviewItem) Line {
	_ = index
	textStyle := StylePicker
	markerStyle := StylePicker
	if row.Selected {
		markerStyle = StylePickerAccent
	}
	title := row.Title
	if title == "" {
		title = row.FloatingID
	}
	stateText := floatingOverviewStateLabel(row)
	sizeText := floatingOverviewSizeText(row)
	marker := "  "
	if row.Selected {
		marker = "▸ "
	}
	cells := []Cell{
		styledCell(marker, markerStyle),
		styledCell("●", terminalPoolStateStyle(stateText)),
		pickerSpace(" "),
	}
	cells = append(cells, terminalPickerColumnCells(title, "", textStyle, floatingOverviewTitleWidth)...)
	cells = append(cells, pickerSpace("  "))
	cells = append(cells, terminalPickerColumnCells(stateText, "", terminalPoolStateStyle(stateText), floatingOverviewStateWidth)...)
	cells = append(cells, pickerSpace("  "))
	cells = append(cells, terminalPickerColumnCells(sizeText, "", textStyle, floatingOverviewSizeWidth)...)
	cells = append(cells, pickerSpace("  "))
	id := row.TerminalID
	if id == "" {
		id = row.FloatingID
	}
	cells = append(cells, terminalPickerColumnCells(id, "", StylePickerMuted, floatingOverviewIDWidth)...)
	return Line{Cells: cells}
}

func floatingOverviewHeaderLine() Line {
	cells := []Cell{
		styledCell("  ", StylePickerMuted),
		styledCell(" ", StylePickerMuted),
		pickerSpace(" "),
	}
	cells = append(cells, terminalPickerColumnCells("terminal", "", StylePickerMuted, floatingOverviewTitleWidth)...)
	cells = append(cells, pickerSpace("  "))
	cells = append(cells, terminalPickerColumnCells("state", "", StylePickerMuted, floatingOverviewStateWidth)...)
	cells = append(cells, pickerSpace("  "))
	cells = append(cells, terminalPickerColumnCells("size", "", StylePickerMuted, floatingOverviewSizeWidth)...)
	cells = append(cells, pickerSpace("  "))
	cells = append(cells, terminalPickerColumnCells("floating", "", StylePickerMuted, floatingOverviewIDWidth)...)
	return Line{Cells: cells}
}

func floatingOverviewStateLabel(row state.FloatingOverviewItem) string {
	stateText := strings.TrimSpace(row.State)
	if stateText == "" || stateText == string(state.PaneTerminalLive) {
		stateText = "live"
	}
	if row.Collapsed {
		stateText = "collapsed"
	}
	return stateText
}

func floatingOverviewSizeText(row state.FloatingOverviewItem) string {
	return floatingOverviewSizeLabel(row)
}

func floatingOverviewHitRegions(rows []state.FloatingOverviewItem, rowOffset int) []HitRegion {
	regions := make([]HitRegion, 0, len(rows)+1)
	for index, row := range rows {
		regions = append(regions, HitRegion{
			Kind:       HitRegionContentAction,
			Rect:       Rect{Y: rowOffset + index, W: 72, H: 1},
			Row:        index,
			HasRow:     true,
			PaneID:     row.FloatingID,
			ActionID:   ActionFloatingSummon.String(),
			Invocation: actiondomain.Invocation{ID: "floating_overview.open", SourceActionID: "floating_overview.open"},
			TargetMode: HitTargetExplicit,
		})
	}
	return regions
}

func terminalPickerHitRegions(rows []state.TerminalPickerItem, rowOffset int) []HitRegion {
	regions := make([]HitRegion, 0, len(rows)+1)
	for index, row := range rows {
		action := ActionPickerAttach
		if row.CreateNew {
			action = ActionPickerNew
		}
		regions = append(regions, HitRegion{
			Kind:       HitRegionContentAction,
			Rect:       Rect{Y: rowOffset + index, W: 72, H: 1},
			PaneID:     row.PaneID,
			Row:        index,
			HasRow:     true,
			ActionID:   action.String(),
			Invocation: invocationForProjection(action),
			TargetMode: HitTargetExplicit,
		})
	}
	return regions
}

func fitContentLine(line Line, width int, fill StyleToken) Line {
	if width <= 0 {
		return Line{}
	}
	if line.Width() > width {
		return contentViewportFitLine(line, width)
	}
	cells := append([]Cell(nil), line.Cells...)
	if pad := width - line.Width(); pad > 0 {
		cells = append(cells, styledCell(strings.Repeat(" ", pad), fill))
	}
	return Line{Cells: cells}
}

func padRightCells(value string, width int) string {
	if width <= 0 {
		return ""
	}
	value = TruncateCells(value, width)
	if pad := width - DisplayWidth(value); pad > 0 {
		value += strings.Repeat(" ", pad)
	}
	return value
}

func terminalManagerLayoutForViewport(viewport state.ViewportStore) terminalManagerLayout {
	cols := 100
	rows := 30
	if viewport.Valid && viewport.Cols > 0 {
		cols = viewport.Cols
	}
	if viewport.Valid && viewport.Rows > 0 {
		rows = viewport.Rows
	}
	overlay := measureWorkbenchNavigatorOverlay(Rect{W: cols, H: rows})
	content := measureOverlayContentRect(OverlayVM{Content: ContentVM{Kind: ContentTerminalPool}}, overlay)
	contentWidth := maxInt(44, content.W)
	contentHeight := maxInt(10, content.H)
	listWidth := clampInt(contentWidth*38/100, defaultTerminalManagerListWidth, 68)
	if contentWidth < 96 {
		listWidth = clampInt(contentWidth*42/100, 28, 44)
	}
	detailWidth := maxInt(22, contentWidth-listWidth-1)
	if detailWidth < 36 && contentWidth > 46 {
		listWidth = maxInt(26, contentWidth-37)
		detailWidth = maxInt(22, contentWidth-listWidth-1)
	}
	maxBoostedListWidth := contentWidth - 1 - 36
	if maxBoostedListWidth < listWidth {
		maxBoostedListWidth = contentWidth - 1 - 22
	}
	if maxBoostedListWidth > listWidth {
		listWidth = minInt(listWidth+terminalManagerListWidthBoost, maxBoostedListWidth)
		detailWidth = maxInt(22, contentWidth-listWidth-1)
	}
	bodyRows := maxInt(8, contentHeight-3)
	snapshotWidth := maxInt(0, detailWidth-2)
	snapshotHeight := clampInt(bodyRows-6, 3, maxInt(3, bodyRows-4))
	return terminalManagerLayout{
		ContentWidth:   contentWidth,
		BodyRows:       bodyRows,
		ListWidth:      listWidth,
		DetailWidth:    detailWidth,
		SnapshotX:      listWidth + 2,
		SnapshotY:      9,
		SnapshotWidth:  snapshotWidth,
		SnapshotHeight: snapshotHeight,
	}
}

func floatingOverviewStatus(count int) string {
	return fmt.Sprintf("floating windows: %d items", count)
}

func floatingOverviewSizeLabel(row state.FloatingOverviewItem) string {
	cols, rows := row.Cols, row.Rows
	if cols <= 0 || rows <= 0 {
		cols = maxInt(0, row.Rect.W-2)
		rows = maxInt(0, row.Rect.H-2)
	}
	if cols <= 0 || rows <= 0 {
		return "-"
	}
	return fmt.Sprintf("%dx%d", cols, rows)
}

func endpointErrorLabel(kind state.EndpointErrorKind, message string) string {
	kind = state.NormalizeEndpointErrorKind(kind)
	message = strings.TrimSpace(message)
	if kind == state.EndpointErrorUnknown {
		return message
	}
	if message == "" {
		return string(kind)
	}
	return string(kind) + ": " + message
}

func terminalPoolStateLine(pool state.TerminalPoolStore) (Line, bool) {
	switch pool.Status {
	case state.TerminalPoolLoading:
		return Line{Cells: []Cell{styledCell("pool ", StyleMuted), NewCell("loading terminals")}}, true
	case state.TerminalPoolError:
		return Line{Cells: []Cell{styledCell("pool error ", StyleWarning), NewCell(pool.LastError)}}, true
	}
	return Line{}, false
}

func terminalPickerSelectableCount(rows []state.TerminalPickerItem) int {
	count := 0
	for _, row := range rows {
		if !row.CreateNew {
			count++
		}
	}
	return count
}

func terminalPickerStatus(count int, query string) string {
	if query == "" {
		return fmt.Sprintf("terminal picker: %d items", count)
	}
	return fmt.Sprintf("terminal picker: %d items query:%s", count, query)
}

func searchLabel(query string) string {
	if query == "" {
		return "[type to filter]"
	}
	return query
}

func contentActionLine(action string, label string) Line {
	return Line{Cells: []Cell{
		styledCell(" ["+action+"] ", StyleAccent),
		NewCell(" " + label),
	}}
}

func pageTitleLine(title string, subtitle string) Line {
	return Line{Cells: []Cell{
		styledCell("◆ "+title, StyleAccent),
		NewCell(" "),
		styledCell(subtitle, StyleMuted),
	}}
}

func searchRowLine(query string, placeholder string) Line {
	value := searchLabel(query)
	if query == "" && placeholder != "" {
		value = "[" + placeholder + "]"
	}
	style := StyleAccent
	if query == "" {
		style = StyleForeground
	}
	return Line{Cells: []Cell{
		styledCell("⌕ search ", StyleAccent),
		styledCell(value, style),
	}}
}

func searchCursorCol(query string) int {
	return DisplayWidth("⌕ search ") + DisplayWidth(query)
}

func formFieldLine(label string, value string, filled bool) Line {
	style := StyleMuted
	if filled {
		style = StyleAccent
	}
	return Line{Cells: []Cell{
		styledCell(strings.ToUpper(label)+" ", StyleMuted),
		styledCell(value, style),
	}}
}

func tokenCell(text string, style StyleToken) Cell {
	return styledCell(" "+text+" ", style)
}

func terminalPoolStateStyle(stateText string) StyleToken {
	switch strings.ToLower(stateText) {
	case "ready", "running", "attached", "live":
		return StyleSuccess
	case "failed", "error", "exited":
		return StyleWarning
	default:
		return StyleMuted
	}
}

func endpointStatusStyle(status state.EndpointStatusKind) StyleToken {
	switch status {
	case state.EndpointStatusConnected:
		return StyleSuccess
	case state.EndpointStatusOffline, state.EndpointStatusDisabled, state.EndpointStatusReconnectRequired, state.EndpointStatusUnregistered:
		return StyleWarning
	case state.EndpointStatusConnecting:
		return StyleAccent
	default:
		return StyleMuted
	}
}

func contentActionRegions(actions []ProjectionID, paneID string, rowOffset int) []HitRegion {
	regions := make([]HitRegion, len(actions))
	for index, action := range actions {
		regions[index] = HitRegion{
			Kind:       HitRegionContentAction,
			Rect:       Rect{Y: index + rowOffset, W: contentActionWidth, H: 1},
			PaneID:     paneID,
			ActionID:   action.String(),
			Invocation: invocationForProjection(action),
			TargetMode: HitTargetExplicit,
		}
	}
	return regions
}

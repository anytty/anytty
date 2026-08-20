package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/anytty/anytty/tui/state"
)

// Terminal Manager Page 是独立管理页面；renderer 只消费 reducer-owned page/list state。
func buildTerminalPoolContent(root state.Root, shell state.ShellStore) ContentVM {
	shell = shell.ReadonlyDefaults()
	query := strings.TrimSpace(shell.Overlay.Query)
	rows := state.TerminalPoolPageItems(root)
	layout := terminalManagerLayoutForViewport(chromeSafeViewportForShell(root.Viewport, shell))
	var lines []Line
	var regions []HitRegion
	if root.Endpoints.HasItems() {
		displayRows := terminalManagerDisplayRows(root, rows)
		listStart := terminalManagerDisplayListStart(displayRows, layout.BodyRows)
		lines = terminalManagerGroupedLines(root, rows, query, layout, displayRows, listStart)
		visibleRows := terminalManagerVisibleDisplayRows(displayRows, listStart, layout.BodyRows)
		regions = terminalManagerGroupedHitRegions(visibleRows, 3, layout)
	} else {
		listStart := terminalManagerListStart(rows, layout.BodyRows)
		lines = terminalManagerLines(root, rows, query, layout, listStart)
		visibleRows := terminalManagerVisibleRows(rows, listStart, layout.BodyRows)
		regions = terminalManagerHitRegions(visibleRows, 3, listStart, layout)
	}
	regions = append(regions, terminalManagerDetailHitRegions(root, rows, layout)...)
	return ContentVM{
		Kind:       ContentTerminalPool,
		Lines:      lines,
		Meta:       terminalManagerMeta(root, rows, layout),
		Status:     terminalPoolPageStatus(root.TerminalPool, len(rows), query),
		Cursor:     Cursor{Visible: true, Row: 0, Col: searchCursorCol(query), Shape: CursorShapeBar},
		HitRegions: regions,
		Pending:    root.TerminalPool.Status == state.TerminalPoolLoading,
		Empty:      root.TerminalPool.Status == state.TerminalPoolReady && len(rows) == 0,
		Error:      root.TerminalPool.LastError,
	}
}

func terminalManagerLines(root state.Root, rows []state.TerminalPoolPageItem, query string, layout terminalManagerLayout, listStart int) []Line {
	statusLine, hasStatus := terminalPoolPageStateLine(root.TerminalPool, len(rows))
	lines := []Line{
		terminalManagerFullLine(searchRowLine(query, "shell"), layout),
		terminalManagerDividerLine(layout),
		terminalManagerBodyLine(terminalManagerListHeaderLine(layout.ListWidth), Line{}, layout),
	}
	for row := 0; row < layout.BodyRows; row++ {
		left := Line{}
		itemIndex := listStart + row
		if itemIndex >= 0 && itemIndex < len(rows) {
			left = terminalManagerRowLine(rows[itemIndex], layout.ListWidth)
		} else if row == 0 && hasStatus {
			left = statusLine
		}
		lines = append(lines, terminalManagerBodyLine(left, Line{}, layout))
	}
	return lines
}

type terminalManagerDisplayRowKind string

const (
	terminalManagerDisplayEndpoint terminalManagerDisplayRowKind = "endpoint"
	terminalManagerDisplayEmpty    terminalManagerDisplayRowKind = "empty"
	terminalManagerDisplayTerminal terminalManagerDisplayRowKind = "terminal"
)

type terminalManagerDisplayRow struct {
	Kind      terminalManagerDisplayRowKind
	Group     state.TerminalPoolPageGroup
	Item      state.TerminalPoolPageItem
	ItemIndex int
}

func terminalManagerDisplayRows(root state.Root, rows []state.TerminalPoolPageItem) []terminalManagerDisplayRow {
	groups := state.TerminalPoolPageGroups(root)
	displayRows := make([]terminalManagerDisplayRow, 0, len(rows)+len(groups))
	for _, group := range groups {
		displayRows = append(displayRows, terminalManagerDisplayRow{Kind: terminalManagerDisplayEndpoint, Group: group, ItemIndex: -1})
		if len(group.VisibleTerminalRows) == 0 {
			displayRows = append(displayRows, terminalManagerDisplayRow{Kind: terminalManagerDisplayEmpty, Group: group, ItemIndex: -1})
			continue
		}
		for _, row := range group.VisibleTerminalRows {
			displayRows = append(displayRows, terminalManagerDisplayRow{Kind: terminalManagerDisplayTerminal, Group: group, Item: row, ItemIndex: terminalPoolPageItemIndex(rows, row)})
		}
	}
	return displayRows
}

func terminalManagerGroupedLines(root state.Root, rows []state.TerminalPoolPageItem, query string, layout terminalManagerLayout, displayRows []terminalManagerDisplayRow, listStart int) []Line {
	statusLine, hasStatus := terminalPoolPageStateLine(root.TerminalPool, len(rows))
	lines := []Line{
		terminalManagerFullLine(searchRowLine(query, "shell"), layout),
		terminalManagerDividerLine(layout),
		terminalManagerBodyLine(terminalManagerListHeaderLine(layout.ListWidth), Line{}, layout),
	}
	for row := 0; row < layout.BodyRows; row++ {
		left := Line{}
		displayIndex := listStart + row
		if displayIndex >= 0 && displayIndex < len(displayRows) {
			left = terminalManagerDisplayRowLine(displayRows[displayIndex], layout.ListWidth)
		} else if row == 0 && hasStatus {
			left = statusLine
		}
		lines = append(lines, terminalManagerBodyLine(left, Line{}, layout))
	}
	return lines
}

func terminalManagerDisplayRowLine(row terminalManagerDisplayRow, width int) Line {
	switch row.Kind {
	case terminalManagerDisplayEndpoint:
		return terminalManagerEndpointHeaderLine(row.Group)
	case terminalManagerDisplayEmpty:
		return terminalManagerEndpointEmptyLine(row.Group)
	default:
		return terminalManagerRowLine(row.Item, width)
	}
}

func terminalManagerEndpointHeaderLine(group state.TerminalPoolPageGroup) Line {
	return Line{Cells: []Cell{
		styledCell("▾ ", StyleAccent),
		styledCell(group.Label, StyleForeground),
	}}
}

func terminalManagerEndpointEmptyLine(_ state.TerminalPoolPageGroup) Line {
	return Line{Cells: []Cell{
		styledCell("  ", StyleMuted),
		styledCell("no terminals", StyleMuted),
	}}
}

func terminalManagerVisibleDisplayRows(rows []terminalManagerDisplayRow, start int, limit int) []terminalManagerDisplayRow {
	if limit <= 0 || start >= len(rows) {
		return nil
	}
	if start < 0 {
		start = 0
	}
	end := minInt(len(rows), start+limit)
	return rows[start:end]
}

func terminalManagerDisplayListStart(rows []terminalManagerDisplayRow, visibleRows int) int {
	if visibleRows <= 0 || len(rows) <= visibleRows {
		return 0
	}
	selected := terminalManagerSelectedDisplayIndex(rows)
	if selected < 0 {
		selected = 0
	}
	start := selected - visibleRows/2
	return clampInt(start, 0, maxInt(0, len(rows)-visibleRows))
}

func terminalManagerMeta(root state.Root, rows []state.TerminalPoolPageItem, layout terminalManagerLayout) ContentMetaVM {
	meta := ContentMetaVM{
		SplitPageLeftWidth: layout.ListWidth,
	}
	selected, ok := selectedTerminalPoolPageItem(rows)
	if !ok {
		return meta
	}
	snapshot, ok := terminalManagerSnapshotVM(root, selected, layout)
	if !ok {
		return meta
	}
	meta.WorkbenchSnapshots = []WorkbenchSnapshotVM{snapshot}
	meta.WorkbenchSnapshotPanel = &snapshot.Panel
	meta.WorkbenchSnapshotRect = snapshot.Rect
	meta.WorkbenchSnapshotContent = snapshot.Content
	return meta
}

func terminalManagerSnapshotVM(root state.Root, selected state.TerminalPoolPageItem, layout terminalManagerLayout) (WorkbenchSnapshotVM, bool) {
	if layout.DetailWidth <= 0 || layout.BodyRows < 0 {
		return WorkbenchSnapshotVM{}, false
	}
	ref := state.NewTerminalRef(selected.EndpointID, selected.TerminalID)
	surface := root.Surface.SurfaceForTerminalRef(ref)
	session := state.TerminalSessionStore{
		EndpointID: ref.EndpointID,
		TerminalID: ref.TerminalID,
		Attached:   selected.Attached,
		Cols:       selected.Cols,
		Rows:       selected.Rows,
		State:      state.TerminalLiveState(selected.State),
	}
	content := buildLiveContentVM(surface, session)
	title := strings.TrimSpace(selected.Title)
	if title == "" {
		title = selected.TerminalID
	}
	panel := PanelVM{
		ID:           "terminal-manager:" + ref.Key(),
		Title:        title,
		Presentation: PanelPresentationSplitLine,
		Active:       true,
		Content:      content,
	}
	rect := Rect{X: layout.ListWidth + 1, Y: 2, W: layout.DetailWidth, H: layout.BodyRows + 1}
	innerWidth := maxInt(0, rect.W-2)
	var previewHeader []Line
	if rect.H >= terminalManagerTrendMinHeight {
		previewHeader = terminalManagerResourceTrendLines(selected, innerWidth)
	}
	headerRows := len(previewHeader)
	contentY := rect.Y + 1
	if headerRows > 0 {
		contentY += headerRows + 1
	}
	contentRect := Rect{X: rect.X + 1, Y: contentY, W: innerWidth, H: maxInt(0, rect.Y+rect.H-1-contentY)}
	return WorkbenchSnapshotVM{Panel: panel, Rect: rect, Content: contentRect, PreviewHeader: previewHeader, PreviewFrame: true}, true
}

const (
	terminalManagerTrendGraphRows  = 2
	terminalManagerTrendHeaderRows = terminalManagerTrendGraphRows * 2
	terminalManagerTrendValueWidth = 7
	terminalManagerTrendMinWidth   = 24
	terminalManagerTrendMinHeight  = terminalManagerTrendHeaderRows + 5
)

func terminalManagerResourceTrendLines(row state.TerminalPoolPageItem, width int) []Line {
	if width < terminalManagerTrendMinWidth {
		return nil
	}
	samples := append([]state.TerminalResourceUsage(nil), row.ResourceHistory...)
	if !row.Resources.SampledAt.IsZero() && (len(samples) == 0 || row.Resources.SampledAt.After(samples[len(samples)-1].SampledAt)) {
		samples = append(samples, row.Resources)
	}
	if len(samples) == 0 {
		return nil
	}
	current := row.Resources
	if current.SampledAt.IsZero() {
		current = samples[len(samples)-1]
	}
	chartWidth := width - len("CPU") - 2 - terminalManagerTrendValueWidth
	if chartWidth < 6 {
		return nil
	}
	if len(samples) > chartWidth {
		samples = samples[len(samples)-chartWidth:]
	}
	cpuValues := make([]uint64, len(samples))
	memoryValues := make([]uint64, len(samples))
	for index, sample := range samples {
		cpuValues[index] = uint64(maxInt(0, sample.CPUPercentX100))
		memoryValues[index] = sample.MemoryBytes
	}
	cpuScale := uint64(10000)
	for _, value := range cpuValues {
		if value > cpuScale {
			cpuScale = value
		}
	}
	lines := terminalManagerTrendLines(
		"CPU",
		terminalPoolCPUPercentShortLabel(current.CPUPercentX100),
		terminalManagerSparklineRows(cpuValues, chartWidth, 0, cpuScale, terminalManagerTrendGraphRows),
		StyleWarning,
	)
	return append(lines, terminalManagerTrendLines(
		"MEM",
		terminalPoolMemoryShortLabel(current.MemoryBytes),
		terminalManagerSparklineRows(memoryValues, chartWidth, terminalManagerMinMetric(memoryValues), terminalManagerMaxMetric(memoryValues), terminalManagerTrendGraphRows),
		StyleAccent,
	)...)
}

func terminalManagerTrendLines(label string, value string, sparklines []string, graphStyle StyleToken) []Line {
	lines := make([]Line, 0, len(sparklines))
	prefixWidth := len(label) + 2 + terminalManagerTrendValueWidth
	for index, sparkline := range sparklines {
		cells := []Cell{NewCell(strings.Repeat(" ", prefixWidth))}
		if index == 0 {
			cells = []Cell{
				styledCell(label, StyleMuted),
				NewCell(" "),
				styledCell(terminalManagerPadLeft(value, terminalManagerTrendValueWidth), StyleStrongForeground),
				NewCell(" "),
			}
		}
		cells = append(cells, styledCell(sparkline, graphStyle))
		lines = append(lines, Line{Cells: cells})
	}
	return lines
}

func terminalManagerSparklineRows(values []uint64, width int, minimum uint64, maximum uint64, height int) []string {
	if width <= 0 || height <= 0 {
		return nil
	}
	if len(values) > width {
		values = values[len(values)-width:]
	}
	levels := []rune(" ▁▂▃▄▅▆▇█")
	units := make([]int, len(values))
	for index, value := range values {
		filled := height * 4
		if maximum > minimum {
			filled = 0
			if value > minimum {
				ratio := float64(value-minimum) / float64(maximum-minimum)
				filled = clampInt(int(ratio*float64(height*8)+0.5), 0, height*8)
			}
			if value > 0 && filled == 0 {
				filled = 1
			}
		}
		units[index] = filled
	}
	rows := make([]string, height)
	for row := 0; row < height; row++ {
		out := make([]rune, 0, width)
		for index := len(values); index < width; index++ {
			out = append(out, ' ')
		}
		base := (height - row - 1) * 8
		for _, filled := range units {
			cellUnits := clampInt(filled-base, 0, 8)
			out = append(out, levels[cellUnits])
		}
		rows[row] = string(out)
	}
	return rows
}

func terminalManagerMinMetric(values []uint64) uint64 {
	if len(values) == 0 {
		return 0
	}
	minimum := values[0]
	for _, value := range values[1:] {
		if value < minimum {
			minimum = value
		}
	}
	return minimum
}

func terminalManagerMaxMetric(values []uint64) uint64 {
	var maximum uint64
	for _, value := range values {
		if value > maximum {
			maximum = value
		}
	}
	return maximum
}

func terminalManagerSelectedDisplayIndex(rows []terminalManagerDisplayRow) int {
	for index, row := range rows {
		if row.Kind == terminalManagerDisplayTerminal && row.Item.Selected {
			return index
		}
	}
	return -1
}

func terminalManagerVisibleRows(rows []state.TerminalPoolPageItem, start int, limit int) []state.TerminalPoolPageItem {
	if limit <= 0 || start >= len(rows) {
		return nil
	}
	if start < 0 {
		start = 0
	}
	end := minInt(len(rows), start+limit)
	return rows[start:end]
}

func terminalManagerListStart(rows []state.TerminalPoolPageItem, visibleRows int) int {
	if visibleRows <= 0 || len(rows) <= visibleRows {
		return 0
	}
	selected := terminalManagerSelectedIndex(rows)
	start := selected - visibleRows/2
	return clampInt(start, 0, maxInt(0, len(rows)-visibleRows))
}

func terminalManagerSelectedIndex(rows []state.TerminalPoolPageItem) int {
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

const (
	terminalManagerCPUColumnWidth    = 6
	terminalManagerMemoryColumnWidth = 6
	terminalManagerViewsColumnWidth  = 3
	terminalManagerLastColumnWidth   = 5
	terminalManagerColumnGapWidth    = 1
	terminalManagerTrailingWidth     = 1
)

func terminalManagerListHeaderLine(width int) Line {
	titleWidth := terminalManagerTitleColumnWidth(width)
	return Line{Cells: []Cell{
		styledCell(padRightCells("TERMINAL", titleWidth), StyleStrongForeground),
		NewCell(" "),
		styledCell(terminalManagerPadLeft("CPU", terminalManagerCPUColumnWidth), StyleMuted),
		NewCell(" "),
		styledCell(terminalManagerPadLeft("MEM", terminalManagerMemoryColumnWidth), StyleMuted),
		NewCell(" "),
		styledCell(terminalManagerPadLeft("V", terminalManagerViewsColumnWidth), StyleMuted),
		NewCell(" "),
		styledCell(terminalManagerPadLeft("LAST", terminalManagerLastColumnWidth), StyleMuted),
		NewCell(" "),
	}}
}

func terminalManagerRowLine(row state.TerminalPoolPageItem, width int) Line {
	marker := "  "
	markerStyle := StyleMuted
	titleStyle := StyleForeground
	if row.Selected {
		marker = "▸ "
		markerStyle = StyleAccent
		titleStyle = StyleAccent
	}
	stateText := terminalPoolStateLabel(row)
	title := Line{Cells: []Cell{
		styledCell(marker, markerStyle),
		styledCell("●", terminalPoolStateStyle(stateText)),
		NewCell(" "),
		styledCell(row.Title, titleStyle),
	}}
	cpu := "--"
	memory := "--"
	if !row.Resources.SampledAt.IsZero() {
		cpu = terminalPoolCPUPercentShortLabel(row.Resources.CPUPercentX100)
		memory = terminalPoolMemoryShortLabel(row.Resources.MemoryBytes)
	}
	activityLabel := TerminalOutputActivityLabel(row.LastOutputAt, time.Now())
	if activityLabel == "" {
		activityLabel = "-"
	}
	cells := fitContentLine(title, terminalManagerTitleColumnWidth(width), StyleForeground).Cells
	cells = append(cells,
		NewCell(" "),
		styledCell(terminalManagerPadLeft(cpu, terminalManagerCPUColumnWidth), StyleForeground),
		NewCell(" "),
		styledCell(terminalManagerPadLeft(memory, terminalManagerMemoryColumnWidth), StyleAccent),
		NewCell(" "),
		styledCell(terminalManagerPadLeft(terminalPoolAttachmentValue(row), terminalManagerViewsColumnWidth), StyleAccent),
		NewCell(" "),
		styledCell(terminalManagerPadLeft(activityLabel, terminalManagerLastColumnWidth), TerminalOutputActivityStyle(activityLabel)),
		NewCell(" "),
	)
	return Line{Cells: cells}
}

func terminalManagerTitleColumnWidth(width int) int {
	fixed := terminalManagerCPUColumnWidth + terminalManagerMemoryColumnWidth + terminalManagerViewsColumnWidth + terminalManagerLastColumnWidth +
		4*terminalManagerColumnGapWidth + terminalManagerTrailingWidth
	return maxInt(1, width-fixed)
}

func terminalManagerPadLeft(value string, width int) string {
	if width <= 0 {
		return ""
	}
	value = TruncateCells(value, width)
	if pad := width - DisplayWidth(value); pad > 0 {
		value = strings.Repeat(" ", pad) + value
	}
	return value
}

func selectedTerminalPoolPageItem(rows []state.TerminalPoolPageItem) (state.TerminalPoolPageItem, bool) {
	if len(rows) == 0 {
		return state.TerminalPoolPageItem{}, false
	}
	selected := rows[0]
	for _, row := range rows {
		if row.Selected {
			return row, true
		}
	}
	return selected, true
}

func terminalManagerDetailLines(root state.Root, selected state.TerminalPoolPageItem, ok bool, visibleRows int) []Line {
	if !ok {
		return terminalManagerEmptyDetailLines(root.TerminalPool)
	}
	lines := []Line{}
	lines = append(lines, terminalManagerResourceGaugeLines(selected)...)
	lines = append(lines,
		terminalManagerHistoryLine(),
		terminalManagerHistoryGaugeLine(),
		terminalManagerConnectionsLine(root, selected),
		terminalManagerSnapshotSpacerLine(),
	)
	lines = append(lines, terminalManagerSnapshotPlaceholderLines(terminalManagerSnapshotPlaceholderHeight(visibleRows, len(lines)))...)
	if strings.TrimSpace(selected.CWD) != "" {
		lines = append(lines, terminalManagerDetailLine("cwd", selected.CWD))
	}
	if visibleRows > 0 && len(lines) >= visibleRows {
		return lines[:visibleRows]
	}
	if command := terminalPoolCommandLabel(selected.Command); command != "-" {
		lines = append(lines, terminalManagerDetailLine("cmd", command))
	}
	if exit := terminalPoolExitLabel(selected); exit != "" {
		lines = append(lines, terminalManagerDetailLine("exit", exit))
	}
	if visibleRows > 0 && len(lines) > visibleRows {
		return lines[:visibleRows]
	}
	return lines
}

func terminalManagerEmptyDetailLines(pool state.TerminalPoolStore) []Line {
	switch pool.Status {
	case state.TerminalPoolLoading:
		return []Line{{Cells: []Cell{styledCell("Loading terminal inventory", StyleForeground)}}}
	case state.TerminalPoolError:
		return []Line{
			{Cells: []Cell{styledCell("Terminal inventory error", StyleWarning)}},
			{Cells: []Cell{styledCell(pool.LastError, StyleForeground)}},
		}
	default:
		return []Line{{Cells: []Cell{styledCell("No terminal selected", StyleForeground)}}}
	}
}

func terminalManagerRowMetricsCells(row state.TerminalPoolPageItem) []Cell {
	usage := row.Resources
	cells := []Cell{NewCell(" ")}
	if usage.SampledAt.IsZero() {
		cells = append(cells, styledCell("--", StyleForeground), NewCell(" "), styledCell("--", StyleForeground))
	} else {
		cells = append(cells,
			styledCell(terminalPoolCPUPercentShortLabel(usage.CPUPercentX100), StyleForeground),
			NewCell(" "),
			styledCell(terminalPoolMemoryShortLabel(usage.MemoryBytes), StyleForeground),
		)
	}
	cells = append(cells, NewCell(" "), styledCell(terminalPoolAttachmentValue(row)+"v", StyleAccent))
	return cells
}

func terminalManagerResourceGaugeLines(row state.TerminalPoolPageItem) []Line {
	usage := row.Resources
	if usage.SampledAt.IsZero() {
		return []Line{
			metricGaugeLine("CPU", "--", 0, false),
			metricGaugeLine("MEM", "--", 0, false),
		}
	}
	cpuRatio := float64(maxInt(0, usage.CPUPercentX100)) / 10000
	memRatio := terminalMemoryGaugeRatio(usage.MemoryBytes)
	return []Line{
		metricGaugeLine("CPU", terminalPoolCPUPercentShortLabel(usage.CPUPercentX100), cpuRatio, true),
		metricGaugeLine("MEM", terminalPoolMemoryShortLabel(usage.MemoryBytes), memRatio, true),
	}
}

func metricGaugeLine(label string, value string, ratio float64, known bool) Line {
	barStyle := StyleAccent
	if !known {
		barStyle = StyleForeground
		ratio = 0
	}
	return Line{Cells: []Cell{
		styledCell(label+" ", StyleAccent),
		styledCell(padRightCells(value, 6), StyleForeground),
		NewCell(" "),
		styledCell(gaugeBar(ratio, 8), barStyle),
	}}
}

func gaugeBar(ratio float64, width int) string {
	if width <= 0 {
		return ""
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio*float64(width) + 0.5)
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("▰", filled) + strings.Repeat("▱", width-filled)
}

func terminalMemoryGaugeRatio(bytes uint64) float64 {
	if bytes == 0 {
		return 0
	}
	// 中文说明：core 当前只提供内存用量，没有该进程的内存上限；这里用分档条表达量级，不能解读为容量占比。
	steps := []uint64{64 << 20, 128 << 20, 256 << 20, 512 << 20, 1 << 30, 2 << 30, 4 << 30, 8 << 30}
	filled := 0
	for _, step := range steps {
		if bytes >= step {
			filled++
		}
	}
	if filled == 0 {
		filled = 1
	}
	return float64(filled) / float64(len(steps))
}

func terminalManagerHistoryLine() Line {
	return Line{Cells: []Cell{
		styledCell("HIST ", StyleAccent),
		styledCell("metrics unavailable", StyleForeground),
	}}
}

func terminalManagerHistoryGaugeLine() Line {
	return Line{Cells: []Cell{
		styledCell("     ", StyleForeground),
		styledCell("oldest kept → newest live ", StyleForeground),
		styledCell(gaugeBar(0, 8), StyleForeground),
	}}
}

func terminalManagerConnectionsLine(root state.Root, row state.TerminalPoolPageItem) Line {
	ref := state.NewTerminalRef(row.EndpointID, row.TerminalID)
	localViews := len(root.TerminalViews.BindingsForTerminalRef(ref))
	count := row.AttachmentCount
	if localViews > count {
		count = localViews
	}
	owner := "owner unknown"
	if binding, ok := terminalManagerOwnerBinding(root, ref); ok {
		if binding.SizeLocked {
			owner = "owner locked"
		} else {
			owner = "owner"
		}
	}
	return Line{Cells: []Cell{
		styledCell("CONN ", StyleAccent),
		styledCell(fmt.Sprintf("%d views", maxInt(0, count)), StyleForeground),
		NewCell(" · "),
		styledCell(owner, StyleForeground),
	}}
}

func terminalManagerOwnerBinding(root state.Root, ref state.TerminalRef) (state.TerminalViewBinding, bool) {
	for _, binding := range root.TerminalViews.BindingsForTerminalRef(ref) {
		if binding.ResizeRole == state.TerminalResizeRoleOwner {
			return binding, true
		}
	}
	return state.TerminalViewBinding{}, false
}

func terminalManagerSnapshotSpacerLine() Line {
	return NewLine("")
}

func terminalManagerSnapshotPlaceholderHeight(visibleRows int, usedRows int) int {
	if visibleRows <= 0 {
		return 4
	}
	remaining := visibleRows - usedRows
	return clampInt(remaining, 3, visibleRows)
}

func terminalManagerSnapshotPlaceholderLines(height int) []Line {
	if height <= 0 {
		return nil
	}
	lines := make([]Line, 0, height)
	for index := 0; index < height; index++ {
		lines = append(lines, NewLine(""))
	}
	return lines
}

func terminalManagerDetailLine(label string, value string) Line {
	if strings.TrimSpace(value) == "" {
		value = "-"
	}
	return Line{Cells: []Cell{
		styledCell(strings.ToUpper(label)+" ", StyleAccent),
		NewCell(value),
	}}
}

func terminalManagerPreviewStatus(surface state.TerminalSurfaceStore) string {
	if surface.TerminalID == "" || (!surface.Ready && len(surface.Lines) == 0 && len(surface.Screen) == 0) {
		return "not loaded"
	}
	if surface.Revision > 0 {
		return fmt.Sprintf("rev:%d", surface.Revision)
	}
	return "latest"
}

func terminalManagerPreviewContentLines(surface state.TerminalSurfaceStore) []Line {
	lines := terminalLiveLineVMs(surface)
	if len(lines) == 0 {
		return nil
	}
	lines = terminalManagerTrimPreviewBlankRows(lines)
	if !terminalManagerPreviewHasContent(lines) {
		return nil
	}
	return lines
}

func terminalManagerTrimPreviewBlankRows(lines []Line) []Line {
	end := len(lines)
	for end > 0 && strings.TrimSpace(lines[end-1].PlainString()) == "" {
		end--
	}
	return lines[:end]
}

func terminalManagerPreviewHasContent(lines []Line) bool {
	for _, line := range lines {
		if strings.TrimSpace(line.PlainString()) != "" {
			return true
		}
	}
	return false
}

func terminalManagerHitRegions(rows []state.TerminalPoolPageItem, rowOffset int, listStart int, layout terminalManagerLayout) []HitRegion {
	regions := make([]HitRegion, 0, len(rows))
	for index := range rows {
		regions = append(regions, HitRegion{
			Kind:       HitRegionContentAction,
			Rect:       Rect{Y: rowOffset + index, W: layout.ListWidth, H: 1},
			Row:        listStart + index,
			HasRow:     true,
			ActionID:   ActionPoolSelect.String(),
			Invocation: invocationForProjection(ActionPoolSelect),
			TargetMode: HitTargetExplicit,
		})
	}
	return regions
}

func terminalManagerGroupedHitRegions(rows []terminalManagerDisplayRow, rowOffset int, layout terminalManagerLayout) []HitRegion {
	regions := make([]HitRegion, 0, len(rows))
	for index, row := range rows {
		if row.Kind != terminalManagerDisplayTerminal || row.ItemIndex < 0 {
			continue
		}
		regions = append(regions, HitRegion{
			Kind:       HitRegionContentAction,
			Rect:       Rect{Y: rowOffset + index, W: layout.ListWidth, H: 1},
			Row:        row.ItemIndex,
			HasRow:     true,
			ActionID:   ActionPoolSelect.String(),
			Invocation: invocationForProjection(ActionPoolSelect),
			TargetMode: HitTargetExplicit,
		})
	}
	return regions
}

func terminalManagerDetailHitRegions(root state.Root, rows []state.TerminalPoolPageItem, layout terminalManagerLayout) []HitRegion {
	_ = root
	_ = rows
	_ = layout
	// 中文说明：Terminal Manager 主内容区只负责选择和观察；attach/restart/delete 等管理动作统一交给全局 footer。
	return nil
}

func terminalPoolPageItemIndex(rows []state.TerminalPoolPageItem, target state.TerminalPoolPageItem) int {
	for index, row := range rows {
		if state.NewTerminalRef(row.EndpointID, row.TerminalID).Equal(state.NewTerminalRef(target.EndpointID, target.TerminalID)) {
			return index
		}
	}
	return -1
}

func terminalPoolPageStateLine(pool state.TerminalPoolStore, rowCount int) (Line, bool) {
	switch pool.Status {
	case state.TerminalPoolLoading:
		return Line{Cells: []Cell{styledCell("list ", StyleMuted), NewCell("loading terminals")}}, true
	case state.TerminalPoolError:
		return Line{Cells: []Cell{styledCell("list error ", StyleWarning), NewCell(pool.LastError)}}, true
	case state.TerminalPoolReady:
		if rowCount == 0 {
			return Line{Cells: []Cell{styledCell("list ", StyleMuted), NewCell("empty")}}, true
		}
	}
	return Line{}, false
}

func terminalPoolPageStatus(pool state.TerminalPoolStore, count int, query string) string {
	prefix := "terminal manager"
	switch pool.Status {
	case state.TerminalPoolLoading:
		prefix += ": loading"
	case state.TerminalPoolError:
		prefix += ": error"
	default:
		prefix += fmt.Sprintf(": %d terminals", count)
	}
	if query != "" {
		prefix += " query:" + query
	}
	return prefix
}

func terminalPoolSizeLabel(cols int, rows int) string {
	if cols <= 0 || rows <= 0 {
		return "size:-"
	}
	return fmt.Sprintf("%dx%d", cols, rows)
}

func terminalPoolStateLabel(row state.TerminalPoolPageItem) string {
	stateText := strings.TrimSpace(row.State)
	if stateText == "" {
		return "unknown"
	}
	return stateText
}

func terminalPoolAttachmentValue(row state.TerminalPoolPageItem) string {
	count := row.AttachmentCount
	if count <= 0 && row.Attached {
		count = 1
	}
	return fmt.Sprintf("%d", maxInt(0, count))
}

func terminalPoolDetailStatus(row state.TerminalPoolPageItem) string {
	views := terminalPoolAttachmentValue(row)
	viewLabel := "views"
	if views == "1" {
		viewLabel = "view"
	}
	return strings.Join([]string{
		terminalPoolStateLabel(row),
		views + " " + viewLabel,
		terminalPoolSizeLabel(row.Cols, row.Rows),
	}, " · ")
}

func terminalPoolCPUPercentShortLabel(percentX100 int) string {
	if percentX100 < 0 {
		percentX100 = 0
	}
	if percentX100 < 100 {
		return fmt.Sprintf("%.1f%%", float64(percentX100)/100)
	}
	return fmt.Sprintf("%.0f%%", float64(percentX100)/100)
}

func terminalPoolMemoryShortLabel(bytes uint64) string {
	switch {
	case bytes >= 1024*1024*1024:
		return fmt.Sprintf("%.1fG", float64(bytes)/(1024*1024*1024))
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.0fM", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.0fK", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

func terminalPoolCommandLabel(command []string) string {
	if len(command) == 0 {
		return "-"
	}
	return strings.Join(command, " ")
}

func terminalPoolExitLabel(row state.TerminalPoolPageItem) string {
	if row.ExitCode == nil && row.ExitedAt.IsZero() {
		return ""
	}
	parts := make([]string, 0, 2)
	if row.ExitCode != nil {
		parts = append(parts, fmt.Sprintf("code:%d", *row.ExitCode))
	}
	if !row.ExitedAt.IsZero() {
		parts = append(parts, row.ExitedAt.Format("2006-01-02 15:04"))
	}
	return strings.Join(parts, " ")
}

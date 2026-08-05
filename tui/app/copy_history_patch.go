package app

import (
	"strings"

	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

type copyHistoryPatchCache struct {
	Valid       bool
	ViewID      string
	EndpointID  state.EndpointID
	TerminalID  string
	Token       string
	Cols        int
	RowsLen     int
	HistoryGen  uint64
	Boundary    state.HistoryBoundary
	HasMore     bool
	Pending     bool
	ViewportTop int
	ViewRows    int
	Cursor      state.CopyPosition
	ContentRect render.Rect
	Metadata    render.RenderMetadata
	Theme       render.Theme
	TopAnchor   copyHistoryPatchRowAnchor
}

type copyHistoryPatchRowAnchor struct {
	Valid        bool
	LineID       uint64
	RowInLine    int
	Text         string
	ClippedStart bool
	ClippedEnd   bool
}

func (runtime *AppRuntime) tryRenderCopyHistoryPatch() bool {
	if !runtime.copyHistoryPatch.Valid || !runtime.canUseIncompleteFrameSink() {
		return false
	}
	patchRoot, current, ok := buildCopyHistoryPatchCacheFromPrevious(runtime.state, runtime.copyHistoryPatch)
	if !ok || !copyHistoryPatchStable(runtime.copyHistoryPatch, current) {
		return false
	}
	delta, ok := copyHistoryPatchVisualDelta(runtime.copyHistoryPatch, current, patchRoot.History)
	if !ok {
		return false
	}
	if delta == 0 {
		return runtime.tryRenderCopyHistoryCursorPatch(patchRoot, current)
	}
	visibleRows := current.ContentRect.H
	if visibleRows <= 1 || visibleRows > len(patchRoot.History.Rows) {
		return false
	}
	if delta <= -visibleRows || delta >= visibleRows {
		return false
	}
	if current.ViewportTop < 0 || current.ViewportTop+visibleRows > len(patchRoot.History.Rows) {
		return false
	}
	if copyHistoryPatchCoveredByFloating(runtime.state, current.ContentRect) {
		return false
	}
	if !copyHistoryPatchContentSafeForIncremental(patchRoot.History, patchRoot.CopyMode, current.ContentRect.W) {
		return false
	}
	scrollRows := delta
	if scrollRows < 0 {
		scrollRows = -scrollRows
	}
	patch := render.FramePatch{
		Rect:      current.ContentRect,
		LineX:     current.ContentRect.X,
		LineWidth: current.ContentRect.W,
	}
	if !copyHistoryPatchCanScrollRegion(current) {
		// 中文说明：终端 scroll region 只能限制上下边界，不能限制左右列。
		// pane 内容区不是全宽时滚动整行会把边框卷走，所以退化为只重写内容矩形。
		patch.Rewrite = true
		patch.LineY = current.ContentRect.Y
		patch.LinesANSI = copyHistoryPatchANSILinesAt(patchRoot.History, patchRoot.CopyMode, current.ViewportTop, visibleRows, current.ContentRect.W, current.ContentRect.X, current.Theme)
	} else if delta > 0 {
		patch.Dir = render.FramePatchScrollUp
		patch.LineY = current.ContentRect.Y + visibleRows - scrollRows
		copyHistoryPatchSetANSILines(&patch, patchRoot.History, patchRoot.CopyMode, current.ViewportTop+visibleRows-scrollRows, scrollRows, current.ContentRect.W, current.Theme)
	} else {
		patch.Dir = render.FramePatchScrollDown
		patch.LineY = current.ContentRect.Y + scrollRows - 1
		copyHistoryPatchSetANSILines(&patch, patchRoot.History, patchRoot.CopyMode, current.ViewportTop, scrollRows, current.ContentRect.W, current.Theme)
	}
	frame := render.Frame{
		Patch:      &patch,
		Cursor:     copyHistoryPatchCursor(patchRoot.History, patchRoot.CopyMode),
		CursorRect: copyHistoryPatchCursorRect(patchRoot.History, patchRoot.CopyMode, current.ContentRect),
		Metadata:   current.Metadata,
		Theme:      current.Theme,
	}
	hitRegions := copyHistoryPatchHitRegions(runtime.lastHitRegions, patchRoot.History, patchRoot.CopyMode, current.ContentRect)
	done := runtime.writeFrame(frame)
	runtime.firstFrameWritten = true
	runtime.observeRuntimePatchFrame(frame)
	runtime.trackFrameCompletion(done, func() {
		runtime.lastHitRegions = hitRegions
		runtime.copyHistoryPatch = current
	})
	return true
}

func (runtime *AppRuntime) tryRenderCopyHistoryCursorPatch(root state.Root, current copyHistoryPatchCache) bool {
	if current.Cursor == runtime.copyHistoryPatch.Cursor {
		return false
	}
	if copyHistoryPatchCoveredByFloating(runtime.state, current.ContentRect) {
		return false
	}
	frame := render.Frame{
		Patch:      &render.FramePatch{CursorOnly: true},
		Cursor:     copyHistoryPatchCursor(root.History, root.CopyMode),
		CursorRect: copyHistoryPatchCursorRect(root.History, root.CopyMode, current.ContentRect),
		Metadata:   current.Metadata,
		Theme:      current.Theme,
	}
	done := runtime.writeFrame(frame)
	runtime.firstFrameWritten = true
	runtime.observeRuntimePatchFrame(frame)
	runtime.trackFrameCompletion(done, func() {
		runtime.copyHistoryPatch = current
	})
	return true
}

func copyHistoryPatchSetANSILines(patch *render.FramePatch, history state.HistoryStore, copyMode state.CopyModeStore, startRow int, count int, width int, theme render.Theme) {
	if count == 1 {
		patch.LineANSI = render.CopyHistoryContentANSILineAt(history, copyMode, startRow, width, patch.LineX, theme)
		return
	}
	patch.LinesANSI = copyHistoryPatchANSILinesAt(history, copyMode, startRow, count, width, patch.LineX, theme)
}

func copyHistoryPatchANSILinesAt(history state.HistoryStore, copyMode state.CopyModeStore, startRow int, count int, width int, lineX int, theme render.Theme) []string {
	if count <= 0 {
		return nil
	}
	lines := make([]string, count)
	for i := 0; i < count; i++ {
		lines[i] = render.CopyHistoryContentANSILineAt(history, copyMode, startRow+i, width, lineX, theme)
	}
	return lines
}

func copyHistoryPatchContentSafeForIncremental(history state.HistoryStore, copyMode state.CopyModeStore, width int) bool {
	if width <= 0 || len(history.Rows) == 0 {
		return false
	}
	top := clampCopyHistoryPatchInt(copyMode.ViewportTop, 0, len(history.Rows)-1)
	visibleRows := copyHistoryPatchVisibleRows(history, copyMode)
	if visibleRows <= 0 || top+visibleRows > len(history.Rows) {
		return false
	}
	for rowIndex := top; rowIndex < top+visibleRows; rowIndex++ {
		row := history.Rows[rowIndex]
		// 中文说明：partial patch 直接写真实 TTY。任何一行如果可能越过内容宽度，
		// 终端自动换行会污染后续行，所以必须回退完整帧重画。
		if copyHistoryPatchPrefixWidth(row)+state.HistoryRowDisplayWidth(row) > width {
			return false
		}
	}
	return true
}

func (runtime *AppRuntime) canUseIncompleteFrameSink() bool {
	if runtime.host == nil {
		return false
	}
	sink := runtime.host.FrameSink()
	if sink == nil {
		return false
	}
	preference, ok := sink.(render.FrameSinkPreference)
	return ok && !preference.NeedsCompleteFrame()
}

func copyHistoryPatchCacheForFrame(runtime *AppRuntime, root state.Root, frame render.Frame) (copyHistoryPatchCache, bool) {
	if runtime == nil || !runtime.canUseIncompleteFrameSink() || frame.Metadata.Width <= 0 || frame.Metadata.Height <= 0 {
		return copyHistoryPatchCache{}, false
	}
	cache, ok := buildCopyHistoryPatchCache(root, frame.Theme)
	if !ok {
		return copyHistoryPatchCache{}, false
	}
	cache.Metadata = frame.Metadata
	return cache, true
}

func buildCopyHistoryPatchCache(root state.Root, theme render.Theme) (copyHistoryPatchCache, bool) {
	root, _ = rootWithActiveCopyHistorySession(root)
	if !copyHistoryPatchStateEligible(root) {
		return copyHistoryPatchCache{}, false
	}
	rect, ok := copyModeContentRect(root)
	if !ok || rect.W <= 0 || rect.H <= 1 {
		return copyHistoryPatchCache{}, false
	}
	if root.CopyMode.ViewRows != rect.H || root.CopyMode.BoundCols != rect.W || root.History.Cols != rect.W {
		return copyHistoryPatchCache{}, false
	}
	if binding, ok := copyModeTerminalBinding(root); ok && !copyHistoryPatchUsesDirectLayout(binding.Layout) {
		return copyHistoryPatchCache{}, false
	}
	if root.CopyMode.ViewportTop < 0 || root.CopyMode.ViewportTop+rect.H > len(root.History.Rows) {
		return copyHistoryPatchCache{}, false
	}
	theme = theme.WithFallback()
	return copyHistoryPatchCache{
		Valid:       true,
		ViewID:      root.CopyMode.ViewID,
		EndpointID:  state.NormalizeEndpointID(root.CopyMode.EndpointID),
		TerminalID:  root.CopyMode.TerminalID,
		Token:       root.CopyMode.BoundToken,
		Cols:        root.CopyMode.BoundCols,
		RowsLen:     len(root.History.Rows),
		HistoryGen:  root.History.Generation,
		Boundary:    root.History.Boundary,
		HasMore:     root.History.HasMore,
		Pending:     root.History.Pending != nil,
		ViewportTop: root.CopyMode.ViewportTop,
		ViewRows:    root.CopyMode.ViewRows,
		Cursor:      root.CopyMode.Cursor,
		ContentRect: rect,
		Metadata:    render.RenderMetadata{Width: root.Viewport.Cols, Height: root.Viewport.Rows},
		Theme:       theme,
		TopAnchor:   copyHistoryPatchRowAnchorAt(root.History, root.CopyMode.ViewportTop),
	}, true
}

func copyHistoryPatchUsesDirectLayout(layout state.TerminalViewLayout) bool {
	layout = layout.Normalize()
	return layout.Mode == state.TerminalViewLayoutAuto && layout.PanX == 0 && layout.PanY == 0 &&
		(layout.AlignX == "" || layout.AlignX == state.TerminalViewAlignStart) &&
		(layout.AlignY == "" || layout.AlignY == state.TerminalViewAlignStart)
}

func buildCopyHistoryPatchCacheFromPrevious(root state.Root, previous copyHistoryPatchCache) (state.Root, copyHistoryPatchCache, bool) {
	if previous.ViewID != "" {
		root = rootWithCopyHistorySessionForView(root, previous.ViewID)
	} else {
		root, _ = rootWithActiveCopyHistorySession(root)
	}
	if !copyHistoryPatchStateEligible(root) {
		return state.Root{}, copyHistoryPatchCache{}, false
	}
	if root.Viewport.Cols != previous.Metadata.Width || root.Viewport.Rows != previous.Metadata.Height {
		return state.Root{}, copyHistoryPatchCache{}, false
	}
	if root.CopyMode.ViewRows != previous.ViewRows || root.CopyMode.BoundCols != previous.Cols || root.History.Cols != previous.Cols {
		return state.Root{}, copyHistoryPatchCache{}, false
	}
	if root.CopyMode.ViewportTop < 0 || root.CopyMode.ViewportTop+previous.ContentRect.H > len(root.History.Rows) {
		return state.Root{}, copyHistoryPatchCache{}, false
	}
	current := previous
	current.ViewID = root.CopyMode.ViewID
	current.EndpointID = state.NormalizeEndpointID(root.CopyMode.EndpointID)
	current.TerminalID = root.CopyMode.TerminalID
	current.Token = root.CopyMode.BoundToken
	current.Cols = root.CopyMode.BoundCols
	current.RowsLen = len(root.History.Rows)
	current.HistoryGen = root.History.Generation
	current.Boundary = root.History.Boundary
	current.HasMore = root.History.HasMore
	current.Pending = root.History.Pending != nil
	current.ViewportTop = root.CopyMode.ViewportTop
	current.ViewRows = root.CopyMode.ViewRows
	current.Cursor = root.CopyMode.Cursor
	current.TopAnchor = copyHistoryPatchRowAnchorAt(root.History, root.CopyMode.ViewportTop)
	current.Valid = true
	return root, current, true
}

func copyHistoryPatchStateEligible(root state.Root) bool {
	copyMode := root.CopyMode
	history := root.History
	return copyMode.Active &&
		copyMode.TerminalID != "" &&
		state.NormalizeEndpointID(copyMode.EndpointID) == state.NormalizeEndpointID(history.EndpointID) &&
		copyMode.TerminalID == history.TerminalID &&
		copyMode.BoundToken != "" &&
		copyMode.BoundToken == history.Token &&
		copyMode.BoundCols > 0 &&
		copyMode.BoundCols == history.Cols &&
		copyMode.Query == "" &&
		copyMode.Mark == nil &&
		copyMode.Selection == nil &&
		len(copyMode.Matches) == 0 &&
		len(history.Rows) > 0
}

func copyHistoryPatchStable(previous copyHistoryPatchCache, current copyHistoryPatchCache) bool {
	return previous.ViewID == current.ViewID &&
		previous.EndpointID == current.EndpointID &&
		previous.TerminalID == current.TerminalID &&
		previous.Token == current.Token &&
		previous.Cols == current.Cols &&
		previous.HistoryGen == current.HistoryGen &&
		previous.HasMore == current.HasMore &&
		previous.ViewRows == current.ViewRows &&
		previous.ContentRect == current.ContentRect &&
		previous.Metadata == current.Metadata &&
		previous.Theme == current.Theme
}

func copyHistoryPatchVisualDelta(previous copyHistoryPatchCache, current copyHistoryPatchCache, history state.HistoryStore) (int, bool) {
	delta := current.ViewportTop - previous.ViewportTop
	if current.RowsLen == previous.RowsLen && current.Boundary == previous.Boundary && current.Pending == previous.Pending {
		return delta, true
	}
	// 中文说明：older prepend 后 RowsLen/Boundary 会变，但旧内容只是整体下移；
	// 这里用上一帧 top anchor 在当前 rows 中的位置抵消 inserted rows，避免误判成整屏跳变。
	if previous.TopAnchor.Valid {
		searchEnd := current.ViewportTop + maxCopyHistoryPatchInt(0, current.RowsLen-previous.RowsLen)
		// 中文说明：older prepend 后如果同时回收了已经滚过的新尾部，RowsLen 可能净减少；
		// 但上一帧 top anchor 仍应落在当前可见窗口附近，不能因此退回全量帧。
		if visibleEnd := current.ViewportTop + current.ViewRows; searchEnd < visibleEnd {
			searchEnd = visibleEnd
		}
		if shiftedTop, ok := copyHistoryPatchFindAnchorNear(history, previous.TopAnchor, current.ViewportTop, searchEnd); ok {
			return current.ViewportTop - shiftedTop, true
		}
	}
	return 0, false
}

func copyHistoryPatchRowAnchorAt(history state.HistoryStore, rowIndex int) copyHistoryPatchRowAnchor {
	if rowIndex < 0 || rowIndex >= len(history.Rows) {
		return copyHistoryPatchRowAnchor{}
	}
	row := history.Rows[rowIndex]
	return copyHistoryPatchRowAnchor{
		Valid:        true,
		LineID:       row.LineID,
		RowInLine:    row.RowInLine,
		Text:         row.Text,
		ClippedStart: row.ClippedStart,
		ClippedEnd:   row.ClippedEnd,
	}
}

func copyHistoryPatchFindAnchorNear(history state.HistoryStore, anchor copyHistoryPatchRowAnchor, start int, end int) (int, bool) {
	if !anchor.Valid || len(history.Rows) == 0 {
		return 0, false
	}
	start = clampCopyHistoryPatchInt(start, 0, len(history.Rows)-1)
	end = clampCopyHistoryPatchInt(end, start, len(history.Rows)-1)
	for rowIndex := start; rowIndex <= end; rowIndex++ {
		if copyHistoryPatchRowAnchorAt(history, rowIndex) == anchor {
			return rowIndex, true
		}
	}
	return 0, false
}

func copyHistoryPatchCursor(history state.HistoryStore, copyMode state.CopyModeStore) render.Cursor {
	if len(history.Rows) == 0 {
		return render.Cursor{}
	}
	row := clampColumn(copyMode.Cursor.Row, 0, len(history.Rows)-1)
	visibleTop := clampColumn(copyMode.ViewportTop, 0, len(history.Rows)-1)
	visibleRow := row - visibleTop
	if visibleRow < 0 {
		visibleRow = 0
	}
	visibleRows := copyHistoryPatchVisibleRows(history, copyMode)
	if visibleRows > 0 && visibleRow >= visibleRows {
		visibleRow = visibleRows - 1
	}
	col := clampColumn(copyMode.Cursor.Col, 0, state.HistoryRowDisplayWidth(history.Rows[row]))
	return render.Cursor{
		Visible: true,
		Row:     visibleRow,
		Col:     copyHistoryPatchPrefixWidth(history.Rows[row]) + col,
		Shape:   render.CursorShapeBlock,
	}
}

func copyHistoryPatchVisibleRows(history state.HistoryStore, copyMode state.CopyModeStore) int {
	if len(history.Rows) == 0 {
		return 0
	}
	top := clampColumn(copyMode.ViewportTop, 0, len(history.Rows)-1)
	height := copyMode.ViewRows
	if height <= 0 {
		height = 8
	}
	if height <= 0 || top+height > len(history.Rows) {
		height = len(history.Rows) - top
	}
	if height < 0 {
		return 0
	}
	return height
}

func copyHistoryPatchCursorRect(history state.HistoryStore, copyMode state.CopyModeStore, rect render.Rect) render.Rect {
	cursor := copyHistoryPatchCursor(history, copyMode)
	if !cursor.Visible {
		return render.Rect{}
	}
	x := rect.X + cursor.Col
	y := rect.Y + cursor.Row
	if x < rect.X || x >= rect.X+rect.W || y < rect.Y || y >= rect.Y+rect.H {
		return render.Rect{}
	}
	return render.Rect{X: x, Y: y, W: 1, H: 1}
}

func copyHistoryPatchHitRegions(previous []render.HitRegion, history state.HistoryStore, copyMode state.CopyModeStore, rect render.Rect) []render.HitRegion {
	replacement := makeCopyHistoryPatchHitRegions(history, copyMode, rect)
	insertAt := -1
	write := 0
	for read, region := range previous {
		if region.Kind == render.HitRegionHistoryRow && !regionBelongsToDifferentCopyView(region, copyMode) {
			if insertAt < 0 {
				insertAt = write
			}
			continue
		}
		if write != read {
			previous[write] = previous[read]
		}
		write++
	}
	out := previous[:write]
	if len(replacement) == 0 {
		return out
	}
	if insertAt < 0 {
		insertAt = len(out)
	}
	tail := append([]render.HitRegion(nil), out[insertAt:]...)
	out = append(out[:insertAt], replacement...)
	return append(out, tail...)
}

func makeCopyHistoryPatchHitRegions(history state.HistoryStore, copyMode state.CopyModeStore, rect render.Rect) []render.HitRegion {
	limit := minCopyHistoryPatchInt(rect.H, len(history.Rows)-copyMode.ViewportTop)
	if limit <= 0 {
		return nil
	}
	out := make([]render.HitRegion, 0, limit)
	floating := strings.HasPrefix(copyMode.ViewID, "floating:")
	for i := 0; i < limit; i++ {
		rowIndex := copyMode.ViewportTop + i
		row := history.Rows[rowIndex]
		width := state.HistoryRowDisplayWidth(row)
		if width == 0 {
			width = 1
		}
		prefix := copyHistoryPatchPrefixWidth(row)
		regionWidth := minCopyHistoryPatchInt(width, rect.W-prefix)
		if regionWidth <= 0 {
			regionWidth = 1
		}
		out = append(out, render.HitRegion{
			Kind:     render.HitRegionHistoryRow,
			Rect:     render.Rect{X: rect.X + prefix, Y: rect.Y + i, W: regionWidth, H: 1},
			LineID:   row.LineID,
			Row:      rowIndex,
			PaneID:   copyMode.PaneID,
			Floating: floating,
		})
	}
	return out
}

func regionBelongsToDifferentCopyView(region render.HitRegion, copyMode state.CopyModeStore) bool {
	if copyMode.ViewID == "" || region.PaneID == "" {
		return false
	}
	if copyMode.PaneID != "" {
		return region.PaneID != copyMode.PaneID
	}
	return true
}

func copyHistoryPatchCanScrollRegion(cache copyHistoryPatchCache) bool {
	return cache.ContentRect.X == 0 && cache.ContentRect.W == cache.Metadata.Width
}

func copyHistoryPatchCoveredByFloating(root state.Root, rect render.Rect) bool {
	if rect.W <= 0 || rect.H <= 0 {
		return false
	}
	viewport := render.Rect{W: root.Viewport.Cols, H: root.Viewport.Rows}
	for _, floating := range root.Shell.ActiveFloatings() {
		if floating.Collapsed {
			continue
		}
		floatingRect := render.Rect{X: floating.Rect.X, Y: floating.Rect.Y, W: floating.Rect.W, H: floating.Rect.H}
		if viewport.W > 0 && viewport.H > 0 {
			floatingRect = intersectCopyHistoryPatchRect(floatingRect, viewport)
		}
		if copyHistoryPatchRectsOverlap(rect, floatingRect) {
			return true
		}
	}
	return false
}

func copyHistoryPatchRectsOverlap(left render.Rect, right render.Rect) bool {
	return left.W > 0 && left.H > 0 &&
		right.W > 0 && right.H > 0 &&
		left.X < right.X+right.W &&
		right.X < left.X+left.W &&
		left.Y < right.Y+right.H &&
		right.Y < left.Y+left.H
}

func intersectCopyHistoryPatchRect(left render.Rect, right render.Rect) render.Rect {
	x1 := maxCopyHistoryPatchInt(left.X, right.X)
	y1 := maxCopyHistoryPatchInt(left.Y, right.Y)
	x2 := minCopyHistoryPatchInt(left.X+maxCopyHistoryPatchInt(0, left.W), right.X+maxCopyHistoryPatchInt(0, right.W))
	y2 := minCopyHistoryPatchInt(left.Y+maxCopyHistoryPatchInt(0, left.H), right.Y+maxCopyHistoryPatchInt(0, right.H))
	if x2 <= x1 || y2 <= y1 {
		return render.Rect{}
	}
	return render.Rect{X: x1, Y: y1, W: x2 - x1, H: y2 - y1}
}

func copyHistoryPatchPrefixWidth(row state.HistoryRow) int {
	if row.ClippedStart {
		return 2
	}
	return 0
}

func minCopyHistoryPatchInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxCopyHistoryPatchInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func clampCopyHistoryPatchInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

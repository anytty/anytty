package state

func sameHistoryLogicalLineSource(left HistoryLogicalLine, right HistoryLogicalLine) bool {
	return left.LineID != 0 &&
		left.LineID == right.LineID &&
		left.Kind == right.Kind &&
		left.Segment == right.Segment &&
		left.SessionID == right.SessionID &&
		left.FrameID == right.FrameID &&
		left.FixedGrid == right.FixedGrid &&
		(!left.FixedGrid || left.ScreenCols == right.ScreenCols)
}

func (store HistoryStore) BeginLatest(req HistoryPendingRequest) (HistoryStore, error) {
	if store.Pending != nil {
		return store, ErrHistoryRequestPending
	}
	req.Kind = HistoryRequestLatest
	store.Pending = &req
	store.Exhausted = ExhaustedMarker{}
	return store, nil
}

// RebindPendingLatest updates the local reflow width while the frozen logical
// source is in flight. The server response does not need to be requested again.
func (store HistoryStore) RebindPendingLatest(cols int) HistoryStore {
	if store.Pending == nil || store.Pending.Kind != HistoryRequestLatest {
		return store
	}
	pending := *store.Pending
	pending.Cols = cols
	store.Pending = &pending
	return store
}

func (store HistoryStore) BeginOlder(req HistoryPendingRequest) (HistoryStore, error) {
	if store.Pending != nil {
		return store, ErrHistoryRequestPending
	}
	req.Kind = HistoryRequestOlder
	store.Pending = &req
	return store, nil
}

func (store HistoryStore) BeginNewer(req HistoryPendingRequest) (HistoryStore, error) {
	if store.Pending != nil {
		return store, ErrHistoryRequestPending
	}
	req.Kind = HistoryRequestNewer
	store.Pending = &req
	return store, nil
}

func (store HistoryStore) BeginOldest(req HistoryPendingRequest) (HistoryStore, error) {
	if store.Pending != nil {
		return store, ErrHistoryRequestPending
	}
	req.Kind = HistoryRequestOldest
	store.Pending = &req
	return store, nil
}

func (store HistoryStore) ApplyWindow(requestID RequestID, window HistoryWindow) (HistoryStore, int, error) {
	if store.Pending == nil || store.Pending.ID != requestID {
		return store, 0, ErrStaleHistoryResponse
	}
	pending := *store.Pending
	if err := validateWindowAgainstPending(pending, window); err != nil {
		return store, 0, err
	}
	store.Pending = nil
	switch window.Op {
	case HistoryWindowReplace:
		store = store.replace(window, pending.Cols)
		return store, len(store.Rows), nil
	case HistoryWindowPrepend:
		beforeRows := len(store.Rows)
		if len(window.Rows) == 0 && !window.HasMore {
			store.Exhausted = ExhaustedMarker{
				Valid:     true,
				RequestID: requestID,
				Token:     pending.Token,
				Cols:      pending.Cols,
				Cursor:    pending.Cursor,
				Boundary:  pending.Boundary,
			}
			return store, 0, nil
		}
		store = store.prepend(window)
		inserted := len(store.Rows) - beforeRows
		if inserted < 0 {
			inserted = 0
		}
		return store, inserted, nil
	case HistoryWindowAppend:
		beforeRows := len(store.Rows)
		store = store.append(window)
		inserted := len(store.Rows) - beforeRows
		if inserted < 0 {
			inserted = 0
		}
		return store, inserted, nil
	default:
		return store, 0, ErrHistoryWindowMismatch
	}
}

func (store HistoryStore) InvalidateWindow() HistoryStore {
	store.Token = ""
	store.ViewID = ""
	store.PaneID = ""
	store.EndpointID = ""
	store.Cols = 0
	store.SourceLines = nil
	store.Rows = nil
	store.Lines = nil
	store.Cursor = HistoryCursor{}
	store.Generation = 0
	store.Boundary = HistoryBoundary{}
	store.HasMore = false
	store.Exhausted = ExhaustedMarker{}
	store.Pending = nil
	return store
}

func (store HistoryStore) OlderRequestState() OlderRequestState {
	if store.Pending != nil {
		return OlderRequestPending
	}
	if store.Exhausted.Valid &&
		store.Exhausted.Token == store.Token &&
		store.Exhausted.Cursor == store.Cursor &&
		store.Exhausted.Boundary == store.Boundary {
		return OlderRequestExhausted
	}
	if store.Token == "" || !store.Cursor.Valid {
		return OlderRequestMissing
	}
	return OlderRequestReady
}

func (store HistoryStore) NewerRequestState() NewerRequestState {
	if store.Pending != nil {
		return NewerRequestPending
	}
	if store.Token == "" || len(store.Rows) == 0 || store.Boundary.LastLineID == 0 {
		return NewerRequestMissing
	}
	tail := store.Rows[len(store.Rows)-1]
	if tail.LineID == 0 || tail.LineID == store.Boundary.LastLineID {
		return NewerRequestMissing
	}
	return NewerRequestReady
}

func validateWindowAgainstPending(pending HistoryPendingRequest, window HistoryWindow) error {
	if NormalizeEndpointID(pending.EndpointID) != NormalizeEndpointID(window.EndpointID) {
		return ErrHistoryWindowMismatch
	}
	if pending.TerminalID != "" && pending.TerminalID != window.TerminalID {
		return ErrHistoryWindowMismatch
	}
	if pending.ViewID != "" && pending.ViewID != window.ViewID {
		return ErrStaleHistoryResponse
	}
	if pending.PaneID != "" && window.PaneID != "" && pending.PaneID != window.PaneID {
		return ErrStaleHistoryResponse
	}
	switch pending.Kind {
	case HistoryRequestLatest:
		if window.Op != HistoryWindowReplace {
			return ErrHistoryWindowMismatch
		}
		// frozen snapshot latest 接纳的是 authoritative logical-line source，
		// response 的 window.Cols 只是 core 当前投影使用的 source cols；TUI 会
		// 基于 SourceLines 按本地 pane 宽度重新 reflow，因此 latest 不要求
		// response cols 与本地请求 cols 完全一致。
	case HistoryRequestOldest:
		if window.Op != HistoryWindowReplace {
			return ErrHistoryWindowMismatch
		}
		if pending.Token == "" || pending.Token != window.Token {
			return ErrStaleHistoryResponse
		}
		if pending.Generation != 0 && pending.Generation != window.Generation {
			return ErrStaleHistoryResponse
		}
	case HistoryRequestOlder:
		if window.Op != HistoryWindowPrepend {
			return ErrHistoryWindowMismatch
		}
		if pending.Cols != 0 && pending.Cols != window.Cols {
			return ErrHistoryWindowMismatch
		}
		if pending.Token != "" && pending.Token != window.Token {
			return ErrStaleHistoryResponse
		}
		if pending.Generation != 0 && pending.Generation != window.Generation {
			return ErrStaleHistoryResponse
		}
		// 中文说明：older prepend 返回的是 existing head 之前的页面，
		// 它的 LastLineID 应该是 older page 自己的尾部，不应等于当前窗口尾部。
		// stale guard 只依赖 token/generation/request id 和 core cursor。
	case HistoryRequestNewer:
		if window.Op != HistoryWindowAppend {
			return ErrHistoryWindowMismatch
		}
		if pending.Cols != 0 && pending.Cols != window.Cols {
			return ErrHistoryWindowMismatch
		}
		if pending.Token != "" && pending.Token != window.Token {
			return ErrStaleHistoryResponse
		}
		if pending.Generation != 0 && pending.Generation != window.Generation {
			return ErrStaleHistoryResponse
		}
		if len(window.SourceLines) != 0 && pending.Boundary.LastLineID != 0 && pending.Boundary.LastLineID != window.Boundary.LastLineID {
			return ErrStaleHistoryResponse
		}
	default:
		return ErrHistoryWindowMismatch
	}
	return nil
}

func (store HistoryStore) replace(window HistoryWindow, cols int) HistoryStore {
	store.ViewID = window.ViewID
	store.PaneID = window.PaneID
	store.EndpointID = NormalizeEndpointID(window.EndpointID)
	store.TerminalID = window.TerminalID
	store.Token = window.Token
	if cols <= 0 {
		cols = window.Cols
	}
	store.Cols = cols
	store.SourceLines = historyWindowSourceLinesOwned(window)
	// 中文说明：latest replace 必须和 older/newer 使用同一条 frozen source
	// canonical reflow 链路。protocol rows 只是 core 当时宽度下的投影；首次
	// 进入 copy/history 时如果直接复用它们，后续 older 触发本地 reflow 后会看到
	// 行序/换行突然变化。
	store.Rows, store.Lines = ReflowHistoryLogicalLines(store.SourceLines, cols)
	store.Cursor = window.Cursor
	store.Generation = window.Generation
	store.Boundary = window.Boundary
	store.ViewportAnchor = window.ViewportAnchor
	store.HasMore = window.HasMore
	store.LoadedLines = len(store.SourceLines)
	store.TotalLines = window.TotalLines
	store.Exhausted = ExhaustedMarker{}
	return store
}

func (store HistoryStore) ReplaceSearchWindow(window HistoryWindow) (HistoryStore, error) {
	if window.Op != HistoryWindowReplace {
		return store, ErrHistoryWindowMismatch
	}
	if store.Token == "" || store.Token != window.Token {
		return store, ErrStaleHistoryResponse
	}
	if store.Generation != 0 && store.Generation != window.Generation {
		return store, ErrStaleHistoryResponse
	}
	if store.TerminalID != "" && store.TerminalID != window.TerminalID {
		return store, ErrHistoryWindowMismatch
	}
	return store.replace(window, store.Cols), nil
}

func (store HistoryStore) prepend(window HistoryWindow) HistoryStore {
	existing := store.SourceLines
	if len(existing) == 0 && len(store.Rows) > 0 {
		existing = historyRowsToLogicalLines(store.Rows, store.Lines)
	}
	older := historyWindowSourceLinesOwned(window)
	if fast, rows, spans := fastPrependedHistoryRows(older, existing, store.Rows, store.Lines, store.Cols, window); fast {
		store.SourceLines = prependHistoryLogicalLines(older, existing)
		store.Rows = rows
		store.Lines = spans
	} else {
		store.SourceLines = mergePrependedHistoryLogicalLines(older, existing)
		store.Rows, store.Lines = ReflowHistoryLogicalLines(store.SourceLines, store.Cols)
	}
	store.Token = window.Token
	store.Cursor = window.Cursor
	store.Generation = window.Generation
	store.Boundary.FirstLineID = window.Boundary.FirstLineID
	store.HasMore = window.HasMore
	store.LoadedLines = len(store.SourceLines)
	store.TotalLines = window.TotalLines
	store.Exhausted = ExhaustedMarker{}
	return store
}

func (store HistoryStore) append(window HistoryWindow) HistoryStore {
	existing := store.SourceLines
	if len(existing) == 0 && len(store.Rows) > 0 {
		existing = historyRowsToLogicalLines(store.Rows, store.Lines)
	}
	newer := historyWindowSourceLinesOwned(window)
	if fast, rows, spans := fastAppendedHistoryRows(existing, newer, store.Rows, store.Lines, store.Cols, window); fast {
		store.SourceLines = appendHistoryLogicalLines(existing, newer)
		store.Rows = rows
		store.Lines = spans
	} else {
		store.SourceLines = mergeAppendedHistoryLogicalLines(existing, newer)
		store.Rows, store.Lines = ReflowHistoryLogicalLines(store.SourceLines, store.Cols)
	}
	store.Token = window.Token
	store.Generation = window.Generation
	store.Boundary.LastLineID = window.Boundary.LastLineID
	store.HasMore = window.HasMore
	store.LoadedLines = len(store.SourceLines)
	store.TotalLines = window.TotalLines
	store.Exhausted = ExhaustedMarker{}
	return store
}

func fastPrependedHistoryRows(older []HistoryLogicalLine, existing []HistoryLogicalLine, existingRows []HistoryRow, existingSpans []HistoryLineSpan, cols int, window HistoryWindow) (bool, []HistoryRow, []HistoryLineSpan) {
	if historyPrependNeedsBoundaryMerge(older, existing) {
		return false, nil, nil
	}
	olderRows, olderSpans := windowRowsForCols(window, older, cols)
	if len(existing) > 0 && len(existingRows) == 0 {
		return false, nil, nil
	}
	// 中文说明：existing tail 来自当前 frozen history，reducer 后续只读它；
	// prepend older 时只复制 slice header，避免每次加载上一页都深拷贝全部已加载历史。
	rows := make([]HistoryRow, 0, len(olderRows)+len(existingRows))
	rows = append(rows, olderRows...)
	rows = append(rows, existingRows...)
	spans := make([]HistoryLineSpan, 0, len(olderSpans)+len(existingSpans))
	spans = append(spans, olderSpans...)
	if len(existingSpans) == 0 && len(existingRows) > 0 {
		existingSpans = historyLineSpansForSearch(HistoryStore{Rows: existingRows})
	}
	spans = appendRebasedHistoryLineSpans(spans, existingSpans, len(olderRows))
	return true, rows, spans
}

func fastAppendedHistoryRows(existing []HistoryLogicalLine, newer []HistoryLogicalLine, existingRows []HistoryRow, existingSpans []HistoryLineSpan, cols int, window HistoryWindow) (bool, []HistoryRow, []HistoryLineSpan) {
	if historyAppendNeedsBoundaryMerge(existing, newer) {
		return false, nil, nil
	}
	newerRows, newerSpans := windowRowsForCols(window, newer, cols)
	if len(existing) > 0 && len(existingRows) == 0 {
		return false, nil, nil
	}
	rows := make([]HistoryRow, 0, len(existingRows)+len(newerRows))
	rows = append(rows, existingRows...)
	rows = append(rows, newerRows...)
	if len(existingSpans) == 0 && len(existingRows) > 0 {
		existingSpans = historyLineSpansForSearch(HistoryStore{Rows: existingRows})
	}
	spans := make([]HistoryLineSpan, 0, len(existingSpans)+len(newerSpans))
	spans = append(spans, existingSpans...)
	spans = appendRebasedHistoryLineSpans(spans, newerSpans, len(existingRows))
	return true, rows, spans
}

func historyPrependNeedsBoundaryMerge(older []HistoryLogicalLine, existing []HistoryLogicalLine) bool {
	if len(older) == 0 || len(existing) == 0 {
		return false
	}
	lastOlder := older[len(older)-1]
	firstExisting := existing[0]
	return sameHistoryLogicalLineSource(lastOlder, firstExisting) &&
		lastOlder.ClippedAfter &&
		firstExisting.ClippedBefore
}

func historyAppendNeedsBoundaryMerge(existing []HistoryLogicalLine, newer []HistoryLogicalLine) bool {
	if len(existing) == 0 || len(newer) == 0 {
		return false
	}
	lastExisting := existing[len(existing)-1]
	firstNewer := newer[0]
	return sameHistoryLogicalLineSource(lastExisting, firstNewer) &&
		lastExisting.ClippedAfter &&
		firstNewer.ClippedBefore
}

func prependHistoryLogicalLines(older []HistoryLogicalLine, existing []HistoryLogicalLine) []HistoryLogicalLine {
	if len(older) == 0 {
		return existing
	}
	if len(existing) == 0 {
		return older
	}
	out := make([]HistoryLogicalLine, 0, len(older)+len(existing))
	out = append(out, older...)
	return append(out, existing...)
}

func appendHistoryLogicalLines(existing []HistoryLogicalLine, newer []HistoryLogicalLine) []HistoryLogicalLine {
	if len(newer) == 0 {
		return existing
	}
	if len(existing) == 0 {
		return newer
	}
	out := make([]HistoryLogicalLine, 0, len(existing)+len(newer))
	out = append(out, existing...)
	return append(out, newer...)
}

func mergePrependedHistoryLogicalLines(older []HistoryLogicalLine, existing []HistoryLogicalLine) []HistoryLogicalLine {
	if len(older) == 0 {
		return cloneHistoryLogicalLines(existing)
	}
	if len(existing) == 0 {
		return cloneHistoryLogicalLines(older)
	}
	merged := cloneHistoryLogicalLines(older)
	rest := cloneHistoryLogicalLines(existing)
	lastOlder := &merged[len(merged)-1]
	firstExisting := rest[0]
	if sameHistoryLogicalLineSource(*lastOlder, firstExisting) &&
		lastOlder.ClippedAfter &&
		firstExisting.ClippedBefore {
		lastOlder.Text += firstExisting.Text
		lastOlder.Cells = append(lastOlder.Cells, cloneHistoryCells(firstExisting.Cells)...)
		if firstExisting.TailFill != nil {
			lastOlder.TailFill = cloneHistoryCellStyle(firstExisting.TailFill)
		}
		lastOlder.LiveTail = lastOlder.LiveTail || firstExisting.LiveTail
		if firstExisting.UpdatedAt.After(lastOlder.UpdatedAt) {
			lastOlder.UpdatedAt = firstExisting.UpdatedAt
		}
		// 中文说明：boundary overlap 代表 older partial 的尾部和 existing partial 的头部
		// 正好拼上了同一 logical line 的中缝；合并后只保留真正外侧还没补齐的 clipped 边。
		lastOlder.ClippedAfter = firstExisting.ClippedAfter
		rest = rest[1:]
	}
	return append(merged, rest...)
}

func mergeAppendedHistoryLogicalLines(existing []HistoryLogicalLine, newer []HistoryLogicalLine) []HistoryLogicalLine {
	if len(existing) == 0 {
		return cloneHistoryLogicalLines(newer)
	}
	if len(newer) == 0 {
		return cloneHistoryLogicalLines(existing)
	}
	merged := cloneHistoryLogicalLines(existing)
	rest := cloneHistoryLogicalLines(newer)
	lastExisting := &merged[len(merged)-1]
	firstNewer := rest[0]
	if sameHistoryLogicalLineSource(*lastExisting, firstNewer) &&
		lastExisting.ClippedAfter &&
		firstNewer.ClippedBefore {
		lastExisting.Text += firstNewer.Text
		lastExisting.Cells = append(lastExisting.Cells, cloneHistoryCells(firstNewer.Cells)...)
		if firstNewer.TailFill != nil {
			lastExisting.TailFill = cloneHistoryCellStyle(firstNewer.TailFill)
		}
		lastExisting.LiveTail = lastExisting.LiveTail || firstNewer.LiveTail
		lastExisting.ClippedAfter = firstNewer.ClippedAfter
		rest = rest[1:]
	}
	return append(merged, rest...)
}

func windowRowsForCols(window HistoryWindow, lines []HistoryLogicalLine, cols int) ([]HistoryRow, []HistoryLineSpan) {
	return ReflowHistoryLogicalLines(lines, cols)
}

func historyWindowSourceLinesOwned(window HistoryWindow) []HistoryLogicalLine {
	if len(window.SourceLines) > 0 {
		return window.SourceLines
	}
	if len(window.Rows) == 0 {
		return nil
	}
	return historyRowsToLogicalLines(window.Rows, window.Lines)
}

func appendRebasedHistoryLineSpans(out []HistoryLineSpan, spans []HistoryLineSpan, delta int) []HistoryLineSpan {
	for _, span := range spans {
		span.StartRow += delta
		span.EndRow += delta
		out = append(out, span)
	}
	return out
}

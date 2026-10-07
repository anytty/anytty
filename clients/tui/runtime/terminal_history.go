package runtime

import (
	"context"
	"errors"
	"time"

	"github.com/anytty/anytty/clients/tui/components/terminal"
	"github.com/anytty/anytty/clients/tui/history"
	"github.com/anytty/anytty/clients/tui/render"
	"github.com/anytty/anytty/clients/tui/render/ansi"
	"github.com/anytty/anytty/proto/access/apipb"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

type historyScrollGenerationKey struct{}

// EnqueueHistory preserves request order for this terminal without occupying
// the protocol reader or another terminal's queue. Saturation is explicit.
func (t *Terminal) EnqueueHistory(work func(context.Context)) bool {
	return t.enqueueHistory(historyWork{fn: work})
}

// EnqueueLatestHistory is for interactive scrolling. A newer scroll cancels
// the current scroll and replaces queued scrolls, so reversing direction
// never waits behind stale requests from the previous gesture.
func (t *Terminal) EnqueueLatestHistory(work func(context.Context)) bool {
	return t.enqueueHistory(historyWork{fn: work, latest: true})
}

func (t *Terminal) enqueueHistory(work historyWork) bool {
	t.historyQueueMu.Lock()
	defer t.historyQueueMu.Unlock()
	if len(t.historyQueue) >= 64 || t.historyContext.Err() != nil {
		return false
	}
	if work.latest {
		t.historyScrollGeneration++
		work.generation = t.historyScrollGeneration
		if t.historyScrollCancel != nil {
			t.historyScrollCancel()
		}
		// Keep non-scroll work intact, but move the newest scroll ahead of
		// queued history operations so input remains responsive.
		kept := make([]historyWork, 0, len(t.historyQueue)+1)
		kept = append(kept, work)
		for _, queued := range t.historyQueue {
			if !queued.latest {
				kept = append(kept, queued)
			}
		}
		t.historyQueue = kept
	} else {
		t.historyQueue = append(t.historyQueue, work)
	}
	if !t.historyRunning {
		t.historyRunning = true
		go t.runHistoryQueue()
	}
	return true
}

func (t *Terminal) runHistoryQueue() {
	for {
		t.historyQueueMu.Lock()
		if len(t.historyQueue) == 0 {
			t.historyRunning = false
			t.historyQueueMu.Unlock()
			return
		}
		work := t.historyQueue[0]
		t.historyQueue[0] = historyWork{}
		t.historyQueue = t.historyQueue[1:]
		var cancel context.CancelFunc
		ctx := t.historyContext
		if work.latest {
			ctx, cancel = context.WithCancel(ctx)
			ctx = context.WithValue(ctx, historyScrollGenerationKey{}, work.generation)
			t.historyScrollCancel = cancel
		}
		t.historyQueueMu.Unlock()

		work.fn(ctx)
		if cancel != nil {
			cancel()
			t.historyQueueMu.Lock()
			// A replacement cannot start until this worker returns, so the
			// active cancel function is still the one being cleared here.
			t.historyScrollCancel = nil
			t.historyQueueMu.Unlock()
		}
	}
}

// historyLocked opens a provider snapshot once per copy session. The PTY is
// the terminal's capability source; the shell never resolves endpoints or
// holds tokens. A local command PTY has no persistent backend.
func (t *Terminal) historyLocked(ctx context.Context) (*history.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	source, ok := t.proc.(history.Source)
	if !ok {
		return nil, nil
	}
	cols := t.Cols()
	t.mu.Lock()
	epoch := t.historyEpoch
	t.mu.Unlock()
	if t.historySnapshot != nil && (t.historySnapshot.Cols != cols || t.historySnapshotEpoch != epoch) {
		if err := t.historySnapshot.Close(ctx); err != nil {
			return nil, err
		}
		t.historySnapshot = nil
		t.historySnapshotEpoch = 0
	}
	if t.historySnapshot == nil {
		backend, ref, err := source.HistoryBackend(ctx)
		if err != nil {
			return nil, err
		}
		snapshot, err := history.Open(ctx, backend, ref, cols)
		if err != nil {
			return nil, err
		}
		t.historySnapshot = snapshot
		t.historySnapshotEpoch = epoch
	}
	return t.historySnapshot, nil
}

func (t *Terminal) historyEpochNow() uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.historyEpoch
}

func (t *Terminal) historyHeight(rows int) int {
	if rows > 0 {
		return rows
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return max(1, t.parser.Rows())
}

// HistoryWindow reads relative to one pane's frozen viewport. Provider
// cursors page beyond the parser cache; local PTYs retain the existing path.
// view is the pane key (terminal box id); the provider snapshot stays shared.
func (t *Terminal) HistoryWindow(ctx context.Context, view string, offset, rows int) ([]string, int, error) {
	t.historyMu.Lock()
	defer t.historyMu.Unlock()
	epoch := t.historyEpochNow()
	snapshot, err := t.historyLocked(ctx)
	if err != nil {
		return nil, 0, err
	}
	if snapshot == nil {
		lines, applied := t.Window(view, offset, rows)
		return lines, t.Offset(view) + applied, nil
	}
	t.mu.Lock()
	base := t.viewLocked(view).historyOffset
	t.mu.Unlock()
	height := t.historyHeight(rows)
	page, applied, err := snapshot.Window(ctx, base+max(0, offset), height)
	if err != nil {
		return nil, 0, err
	}
	if rows <= 0 && offset == 0 {
		t.publishHistoryIfEpoch(epoch, view, page, applied)
	}
	return history.Text(page), applied, nil
}

// HistoryScroll changes one pane's viewport using persistent history. Only the
// published viewport is protected by mu; network I/O never is. The provider
// snapshot is shared per terminal but each view keeps its own offset, so
// scrolling one pane does not move a sibling bound to the same source.
func (t *Terminal) HistoryScroll(ctx context.Context, view string, delta, rows int) ([]string, int, error) {
	t.historyMu.Lock()
	defer t.historyMu.Unlock()
	if err := t.validateHistoryScroll(ctx); err != nil {
		return nil, 0, err
	}
	// A downward wheel event while live must stay on the live PTY. Opening a
	// provider snapshot for a no-op at offset zero would replace the live frame
	// with a frozen "latest" page and makes repeated bottom scrolling oscillate
	// between two representations of the same terminal.
	t.mu.Lock()
	v := t.viewLocked(view)
	historyActive, historyOffset, pinned := v.historyActive, v.historyOffset, v.pinned
	epoch := t.historyEpoch
	t.mu.Unlock()
	_, persistent := t.proc.(history.Source)
	if persistent && !historyActive && delta <= 0 {
		// A persistent source normally uses historyActive for its frozen
		// viewport. Keep the parser-backed pinned path coherent as well: a
		// reconnect or a capability transition can leave a local frozen window
		// behind while no provider snapshot is published. Returning its old
		// rows as offset zero would make the shell believe it reached live while
		// the renderer still showed stale content.
		if pinned {
			lines, offset := t.Scroll(view, delta, rows)
			return lines, offset, nil
		}
		lines, _ := t.Window(view, 0, rows)
		return lines, 0, nil
	}
	if persistent && historyActive && delta < 0 && historyOffset+delta <= 0 {
		// Match the old copy-mode boundary: returning to the frozen bottom
		// releases the snapshot and immediately reveals the live PTY screen.
		if err := t.detachHistoryViewportForScroll(ctx, view); err != nil {
			return nil, 0, err
		}
		lines, _ := t.Window(view, 0, rows)
		return lines, 0, nil
	}
	snapshot, err := t.historyLocked(ctx)
	if err != nil {
		return nil, 0, err
	}
	if snapshot == nil {
		lines, offset := t.Scroll(view, delta, rows)
		return lines, offset, nil
	}
	t.mu.Lock()
	offset := max(0, t.viewLocked(view).historyOffset+delta)
	t.mu.Unlock()
	page, applied, err := snapshot.Window(ctx, offset, t.historyHeight(rows))
	if err != nil {
		return nil, 0, err
	}
	// A provider may complete a request just as a newer wheel event cancels
	// it. Do not publish that stale page after cancellation; the latest-wins
	// queue will run the newer direction against the previous viewport.
	if err := t.validateHistoryScroll(ctx); err != nil {
		return nil, 0, err
	}
	if delta > 0 && applied <= 0 {
		// There is no older row to enter. Keep the terminal live instead of
		// marking an unchanged provider page as an active history viewport.
		if err := t.detachHistoryViewportForScroll(ctx, view); err != nil {
			return nil, 0, err
		}
		lines, _ := t.Window(view, 0, rows)
		return lines, 0, nil
	}
	if err := t.publishHistoryForScroll(ctx, view, page, applied, epoch); err != nil {
		return nil, 0, err
	}
	return history.Text(page), applied, nil
}

func (t *Terminal) validateHistoryScroll(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	generation, latest := ctx.Value(historyScrollGenerationKey{}).(uint64)
	if !latest {
		return nil
	}
	t.historyQueueMu.Lock()
	current := t.historyScrollGeneration
	t.historyQueueMu.Unlock()
	if generation != current {
		return context.Canceled
	}
	return ctx.Err()
}

func (t *Terminal) publishHistoryForScroll(ctx context.Context, view string, page []*apipb.HistoryRow, offset int, epoch uint64) error {
	generation, latest := ctx.Value(historyScrollGenerationKey{}).(uint64)
	if !latest {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !t.publishHistoryIfEpoch(epoch, view, page, offset) {
			return context.Canceled
		}
		return nil
	}
	t.historyQueueMu.Lock()
	defer t.historyQueueMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if generation != t.historyScrollGeneration {
		return context.Canceled
	}
	if !t.publishHistoryIfEpoch(epoch, view, page, offset) {
		return context.Canceled
	}
	return nil
}

func (t *Terminal) detachHistoryViewportForScroll(ctx context.Context, view string) error {
	generation, latest := ctx.Value(historyScrollGenerationKey{}).(uint64)
	if !latest {
		if err := ctx.Err(); err != nil {
			return err
		}
		t.detachHistoryViewportLocked(view)
		return nil
	}
	t.historyQueueMu.Lock()
	defer t.historyQueueMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if generation != t.historyScrollGeneration {
		return context.Canceled
	}
	t.detachHistoryViewportLocked(view)
	return nil
}

// detachHistoryViewportLocked clears one view's published frozen view while
// historyMu is held. The shared provider token is released only when no other
// view still holds a frozen viewport, so clearing one pane cannot drop the
// snapshot a sibling bound to the same source is still reading. The release is
// asynchronous so the bottom wheel stays local and immediate.
func (t *Terminal) detachHistoryViewportLocked(view string) {
	t.mu.Lock()
	v := t.viewLocked(view)
	v.historyActive, v.historyRows, v.historyOffset = false, nil, 0
	v.pinned, v.viewEnd = false, 0
	var snapshot *history.Snapshot
	if !t.anyViewActiveLocked() {
		snapshot = t.historySnapshot
		t.historySnapshot = nil
		t.historySnapshotEpoch = 0
	}
	t.mu.Unlock()
	if snapshot == nil {
		return
	}
	go func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = snapshot.Close(releaseCtx)
		cancel()
	}()
}

func (t *Terminal) publishHistory(view string, page []*apipb.HistoryRow, offset int) {
	t.mu.Lock()
	v := t.viewLocked(view)
	v.historyActive, v.historyRows, v.historyOffset = true, page, offset
	t.mu.Unlock()
}

func (t *Terminal) publishHistoryIfEpoch(epoch uint64, view string, page []*apipb.HistoryRow, offset int) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.historyEpoch != epoch {
		return false
	}
	v := t.viewLocked(view)
	v.historyActive, v.historyRows, v.historyOffset = true, page, offset
	return true
}

// HistoryActive distinguishes frozen-at-offset-zero from a live viewport for
// one pane.
func (t *Terminal) HistoryActive(view string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	v := t.viewLocked(view)
	return v.historyActive || v.pinned
}

// BeginHistoryScroll marks an asynchronous persistent scroll as host-owned
// before any provider I/O starts. The token prevents an older canceled
// request from clearing the routing bit belonging to a newer wheel event.
// Routing is per-view so one pane's pending scroll never captures a sibling.
func (t *Terminal) BeginHistoryScroll(view string, delta int) uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	v := t.viewLocked(view)
	v.historyRoutingSeq++
	token := v.historyRoutingSeq
	if delta > 0 || v.historyActive || v.pinned || v.historyRouting {
		v.historyRouting = true
	}
	return token
}

// EndHistoryScroll clears the pending host-owned routing state when the
// matching request completes. A newer request keeps the bit set.
func (t *Terminal) EndHistoryScroll(view string, token uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	v := t.viewLocked(view)
	if token == v.historyRoutingSeq {
		v.historyRouting = false
	}
}

// HistoryRoutingActive includes a pending remote history request. It is used
// only by input routing; HistoryActive remains the published frozen viewport
// state for rendering and copy semantics.
func (t *Terminal) HistoryRoutingActive(view string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	v := t.viewLocked(view)
	return v.historyActive || v.pinned || v.historyRouting
}

// HasPersistentHistory reports whether scrolling needs the endpoint history
// provider. Local PTYs use the parser's in-memory scrollback and can take the
// synchronous hot path in the TUI host.
func (t *Terminal) HasPersistentHistory() bool {
	_, ok := t.proc.(history.Source)
	return ok
}

// HistoryRelease ends copy mode for one pane and releases the shared token on
// its original connection. The provider snapshot is released only once no
// other view still holds a frozen viewport, so closing one pane cannot break a
// sibling bound to the same source. Closing an offline connection must not
// delay terminal cleanup.
func (t *Terminal) HistoryRelease(ctx context.Context, view string) error {
	t.mu.Lock()
	v := t.viewLocked(view)
	v.historyRoutingSeq++
	v.historyRouting = false
	t.mu.Unlock()
	t.historyQueueMu.Lock()
	t.historyScrollGeneration++
	if t.historyScrollCancel != nil {
		t.historyScrollCancel()
	}
	t.historyQueueMu.Unlock()
	t.historyMu.Lock()
	defer t.historyMu.Unlock()
	return t.detachHistoryViewportSync(ctx, view)
}

// historyReleaseAll clears every viewport and always releases the shared
// snapshot. It is the terminal-destroy path (Close), where no sibling view can
// remain active.
func (t *Terminal) historyReleaseAll(ctx context.Context) error {
	t.mu.Lock()
	for _, v := range t.views {
		v.historyRouting = false
		v.historyActive, v.historyRows, v.historyOffset = false, nil, 0
		v.pinned, v.viewEnd = false, 0
	}
	t.mu.Unlock()
	t.historyMu.Lock()
	defer t.historyMu.Unlock()
	snapshot := t.historySnapshot
	t.historySnapshot = nil
	t.historySnapshotEpoch = 0
	if snapshot == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return snapshot.Close(ctx)
}

// detachHistoryViewportSync clears one view and synchronously closes the
// shared snapshot when no other view remains active. Callers must hold
// historyMu.
func (t *Terminal) detachHistoryViewportSync(ctx context.Context, view string) error {
	t.mu.Lock()
	v := t.viewLocked(view)
	v.historyActive, v.historyRows, v.historyOffset = false, nil, 0
	v.pinned, v.viewEnd = false, 0
	var snapshot *history.Snapshot
	if !t.anyViewActiveLocked() {
		snapshot = t.historySnapshot
		t.historySnapshot = nil
		t.historySnapshotEpoch = 0
	}
	t.mu.Unlock()
	if snapshot == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return snapshot.Close(ctx)
}

// HistorySearch belongs to the terminal, and uses the same frozen token as
// paging/copy. Provider search understands logical lines, glob and regex.
func (t *Terminal) HistorySearch(ctx context.Context, query string, mode apipb.HistorySearchMode, backward bool, start *apipb.HistoryTextPosition) (*apipb.HistorySearchResult, error) {
	t.historyMu.Lock()
	defer t.historyMu.Unlock()
	snapshot, err := t.historyLocked(ctx)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, errors.New("terminal has no persistent history provider")
	}
	return snapshot.Search(ctx, query, mode, backward, start)
}

// Search searches one pane's frozen history and moves its viewport to the
// match. The shell supplies intent and a viewport cursor, not provider tokens
// or a scan loop. start is a row-major cell index; match end is exclusive.
// view is the pane key, so a search moves only the pane that issued it.
func (t *Terminal) Search(ctx context.Context, view, query, mode string, backward bool, start int) (*pb.MethodData, error) {
	pattern, err := history.Pattern(mode, query)
	if err != nil {
		return nil, err
	}
	t.historyMu.Lock()
	defer t.historyMu.Unlock()
	epoch := t.historyEpochNow()
	snapshot, err := t.historyLocked(ctx)
	if err != nil {
		return nil, err
	}
	cols, height := t.Cols(), t.historyHeight(0)
	if cols <= 0 {
		return nil, errors.New("terminal has no columns")
	}
	if snapshot != nil {
		t.mu.Lock()
		visible := t.viewLocked(view).historyRows
		t.mu.Unlock()
		var position *apipb.HistoryTextPosition
		if row := start / cols; row >= 0 && row < len(visible) {
			col := int32(snapshot.ColumnBase(visible[row]) + start%cols)
			if !backward {
				col++
			}
			position = &apipb.HistoryTextPosition{LineId: visible[row].GetLogicalLineId(), Col: col}
		}
		searchMode := apipb.HistorySearchMode_HISTORY_SEARCH_MODE_TEXT
		if mode == "glob" {
			searchMode = apipb.HistorySearchMode_HISTORY_SEARCH_MODE_GLOB
		}
		if mode == "regex" {
			searchMode = apipb.HistorySearchMode_HISTORY_SEARCH_MODE_REGEX
		}
		result, err := snapshot.Search(ctx, query, searchMode, backward, position)
		if err != nil {
			return nil, err
		}
		if !result.GetFound() {
			return &pb.MethodData{}, nil
		}
		rows, offset, begin, end, err := snapshot.Locate(ctx, result.GetMatch(), height)
		if err != nil {
			return nil, err
		}
		if !t.publishHistoryIfEpoch(epoch, view, rows, offset) {
			return &pb.MethodData{}, context.Canceled
		}
		return &pb.MethodData{Found: true, Wrapped: result.GetWrapped(), Rows: history.Text(rows), Offset: int32(offset), MatchStart: int32(begin), MatchEnd: int32(end)}, nil
	}
	// Command PTYs use their locally retained history. Search remains owned by
	// Terminal even in this fallback; shell programs use the same method.
	t.mu.Lock()
	defer t.mu.Unlock()
	v := t.viewLocked(view)
	total := t.contentTotalLocked()
	text := t.windowEndingLocked(total, total)
	current := max(0, t.windowEndLocked(view)-height)*cols + max(0, start)
	type match struct{ begin, end int }
	var matches []match
	for row, line := range text {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, pair := range pattern.FindAllStringIndex(line, -1) {
			matches = append(matches, match{row*cols + render.DisplayWidth(line[:pair[0]]), row*cols + render.DisplayWidth(line[:pair[1]])})
		}
	}
	if len(matches) == 0 {
		return &pb.MethodData{}, nil
	}
	selected, wrapped := -1, false
	if backward {
		for i := len(matches) - 1; i >= 0; i-- {
			if matches[i].begin < current {
				selected = i
				break
			}
		}
		if selected < 0 {
			selected, wrapped = len(matches)-1, true
		}
	} else {
		for i, match := range matches {
			if match.begin > current {
				selected = i
				break
			}
		}
		if selected < 0 {
			selected, wrapped = 0, true
		}
	}
	chosen := matches[selected]
	top := min(max(0, chosen.begin/cols-height/2), max(0, total-height))
	v.pinned, v.viewEnd = true, min(total, top+height)
	return &pb.MethodData{Found: true, Wrapped: wrapped, Rows: t.windowEndingLocked(v.viewEnd, height),
		Offset: int32(t.offsetLocked(view)), MatchStart: int32(chosen.begin - top*cols), MatchEnd: int32(chosen.end - top*cols)}, nil
}

// HistoryCopy resolves viewport coordinates to provider logical-line ranges,
// so soft-wrapped rows copy as one logical line. Block selection remains a
// visual rectangle and is resolved against the pane's published viewport.
func (t *Terminal) HistoryCopy(ctx context.Context, view string, spec *CopySpec) (string, error) {
	t.historyMu.Lock()
	defer t.historyMu.Unlock()
	if t.historySnapshot == nil || spec == nil || spec.Mode == "block" {
		if spec == nil {
			return t.CopyText(view), nil
		}
		text, _ := t.CopyWindow(view, *spec)
		return text, nil
	}
	t.mu.Lock()
	rows := t.viewLocked(view).historyRows
	t.mu.Unlock()
	if len(rows) == 0 {
		return "", errors.New("terminal history: no visible rows")
	}
	a, b := min(max(0, spec.StartRow), len(rows)-1), min(max(0, spec.EndRow), len(rows)-1)
	x, y := max(0, spec.StartCol), max(0, spec.EndCol)
	if a > b || (a == b && x > y) {
		a, b, x, y = b, a, y, x
	}
	cols := t.historySnapshot.Cols
	if spec.Mode == "line" {
		x, y = 0, cols-1
	}
	return t.historySnapshot.Copy(ctx, &apipb.HistoryRange{
		StartLineId: rows[a].GetLogicalLineId(), StartCol: int32(t.historySnapshot.ColumnBase(rows[a]) + x),
		EndLineId: rows[b].GetLogicalLineId(), EndCol: int32(t.historySnapshot.ColumnBase(rows[b]) + y + 1),
	})
}

func historyCells(rows []*apipb.HistoryRow) [][]ansi.Cell {
	out := make([][]ansi.Cell, len(rows))
	for i, row := range rows {
		for _, cell := range row.GetRow().GetCells() {
			width := max(1, int(cell.GetWidth()))
			out[i] = append(out[i], ansi.Cell{Text: cell.GetContent(), Width: width, Style: historyStyle(cell.GetStyle())})
		}
	}
	return out
}

func historyStyle(style *apipb.CellStyle) render.Token {
	return render.Token((render.Style{FG: style.GetForeground(), BG: style.GetBackground(),
		Bold: style.GetBold(), Italic: style.GetItalic(), Underline: style.GetUnderline(),
		Blink: style.GetBlink(), Reverse: style.GetReverse(), Strikethrough: style.GetStrikethrough()}).String())
}

func historyScreen(rows []*apipb.HistoryRow, cols int) terminal.Screen {
	screen := terminalScreen(ansi.Screen{Lines: historyCells(rows)})
	for i, row := range rows {
		style := historyStyle(row.GetRow().GetTailFill())
		if style == "" {
			continue
		}
		width := 0
		for _, cell := range screen.Lines[i] {
			width += cell.Width
		}
		// Tail fill is display-only: never append it to historyRows or copy text.
		for ; width < cols; width++ {
			screen.Lines[i] = append(screen.Lines[i], terminal.Cell{Text: " ", Width: 1, Style: style})
		}
	}
	return screen
}

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

// HistoryWindow reads relative to this terminal's frozen viewport. Provider
// cursors page beyond the parser cache; local PTYs retain the existing path.
func (t *Terminal) HistoryWindow(ctx context.Context, offset, rows int) ([]string, int, error) {
	t.historyMu.Lock()
	defer t.historyMu.Unlock()
	epoch := t.historyEpochNow()
	snapshot, err := t.historyLocked(ctx)
	if err != nil {
		return nil, 0, err
	}
	if snapshot == nil {
		lines, applied := t.Window(offset, rows)
		return lines, t.Offset() + applied, nil
	}
	t.mu.Lock()
	base := t.historyOffset
	t.mu.Unlock()
	height := t.historyHeight(rows)
	page, applied, err := snapshot.Window(ctx, base+max(0, offset), height)
	if err != nil {
		return nil, 0, err
	}
	if rows <= 0 && offset == 0 {
		t.publishHistoryIfEpoch(epoch, page, applied)
	}
	return history.Text(page), applied, nil
}

// HistoryScroll changes the terminal's viewport using persistent history.
// Only the published viewport is protected by mu; network I/O never is.
func (t *Terminal) HistoryScroll(ctx context.Context, delta, rows int) ([]string, int, error) {
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
	historyActive, historyOffset, pinned := t.historyActive, t.historyOffset, t.pinned
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
			lines, offset := t.Scroll(delta, rows)
			return lines, offset, nil
		}
		lines, _ := t.Window(0, rows)
		return lines, 0, nil
	}
	if persistent && historyActive && delta < 0 && historyOffset+delta <= 0 {
		// Match the old copy-mode boundary: returning to the frozen bottom
		// releases the snapshot and immediately reveals the live PTY screen.
		if err := t.detachHistoryViewportForScroll(ctx); err != nil {
			return nil, 0, err
		}
		lines, _ := t.Window(0, rows)
		return lines, 0, nil
	}
	snapshot, err := t.historyLocked(ctx)
	if err != nil {
		return nil, 0, err
	}
	if snapshot == nil {
		lines, offset := t.Scroll(delta, rows)
		return lines, offset, nil
	}
	t.mu.Lock()
	offset := max(0, t.historyOffset+delta)
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
		if err := t.detachHistoryViewportForScroll(ctx); err != nil {
			return nil, 0, err
		}
		lines, _ := t.Window(0, rows)
		return lines, 0, nil
	}
	if err := t.publishHistoryForScroll(ctx, page, applied, epoch); err != nil {
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

func (t *Terminal) publishHistoryForScroll(ctx context.Context, page []*apipb.HistoryRow, offset int, epoch uint64) error {
	generation, latest := ctx.Value(historyScrollGenerationKey{}).(uint64)
	if !latest {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !t.publishHistoryIfEpoch(epoch, page, offset) {
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
	if !t.publishHistoryIfEpoch(epoch, page, offset) {
		return context.Canceled
	}
	return nil
}

func (t *Terminal) detachHistoryViewportForScroll(ctx context.Context) error {
	generation, latest := ctx.Value(historyScrollGenerationKey{}).(uint64)
	if !latest {
		if err := ctx.Err(); err != nil {
			return err
		}
		t.detachHistoryViewportLocked()
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
	t.detachHistoryViewportLocked()
	return nil
}

// detachHistoryViewportLocked clears the published frozen view while
// historyMu is held. The provider token is released asynchronously so the
// bottom wheel remains a local, immediate operation even when the endpoint is
// slow or offline.
func (t *Terminal) detachHistoryViewportLocked() {
	t.mu.Lock()
	snapshot := t.historySnapshot
	t.historySnapshot = nil
	t.historySnapshotEpoch = 0
	t.historyActive, t.historyRows, t.historyOffset = false, nil, 0
	t.pinned, t.viewEnd = false, 0
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

func (t *Terminal) publishHistory(page []*apipb.HistoryRow, offset int) {
	t.mu.Lock()
	t.historyActive, t.historyRows, t.historyOffset = true, page, offset
	t.mu.Unlock()
}

func (t *Terminal) publishHistoryIfEpoch(epoch uint64, page []*apipb.HistoryRow, offset int) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.historyEpoch != epoch {
		return false
	}
	t.historyActive, t.historyRows, t.historyOffset = true, page, offset
	return true
}

// HistoryActive distinguishes frozen-at-offset-zero from a live viewport.
func (t *Terminal) HistoryActive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.historyActive || t.pinned
}

// BeginHistoryScroll marks an asynchronous persistent scroll as host-owned
// before any provider I/O starts. The token prevents an older canceled
// request from clearing the routing bit belonging to a newer wheel event.
func (t *Terminal) BeginHistoryScroll(delta int) uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.historyRoutingSeq++
	token := t.historyRoutingSeq
	if delta > 0 || t.historyActive || t.pinned || t.historyRouting {
		t.historyRouting = true
	}
	return token
}

// EndHistoryScroll clears the pending host-owned routing state when the
// matching request completes. A newer request keeps the bit set.
func (t *Terminal) EndHistoryScroll(token uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if token == t.historyRoutingSeq {
		t.historyRouting = false
	}
}

// HistoryRoutingActive includes a pending remote history request. It is used
// only by input routing; HistoryActive remains the published frozen viewport
// state for rendering and copy semantics.
func (t *Terminal) HistoryRoutingActive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.historyActive || t.pinned || t.historyRouting
}

// HasPersistentHistory reports whether scrolling needs the endpoint history
// provider. Local PTYs use the parser's in-memory scrollback and can take the
// synchronous hot path in the TUI host.
func (t *Terminal) HasPersistentHistory() bool {
	_, ok := t.proc.(history.Source)
	return ok
}

// HistoryRelease ends copy mode and releases the token on its original
// connection. Closing an offline connection must not delay terminal cleanup.
func (t *Terminal) HistoryRelease(ctx context.Context) error {
	t.mu.Lock()
	t.historyRoutingSeq++
	t.historyRouting = false
	t.mu.Unlock()
	t.historyQueueMu.Lock()
	t.historyScrollGeneration++
	if t.historyScrollCancel != nil {
		t.historyScrollCancel()
	}
	t.historyQueueMu.Unlock()
	t.historyMu.Lock()
	defer t.historyMu.Unlock()
	t.mu.Lock()
	snapshot := t.historySnapshot
	t.historySnapshot = nil
	t.historySnapshotEpoch = 0
	t.historyActive, t.historyRows, t.historyOffset = false, nil, 0
	t.pinned, t.viewEnd = false, 0
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

// Search searches the terminal's frozen history and moves its viewport to the
// match. The shell supplies intent and a viewport cursor, not provider tokens
// or a scan loop. start is a row-major cell index; match end is exclusive.
func (t *Terminal) Search(ctx context.Context, query, mode string, backward bool, start int) (*pb.MethodData, error) {
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
		visible := t.historyRows
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
		if !t.publishHistoryIfEpoch(epoch, rows, offset) {
			return &pb.MethodData{}, context.Canceled
		}
		return &pb.MethodData{Found: true, Wrapped: result.GetWrapped(), Rows: history.Text(rows), Offset: int32(offset), MatchStart: int32(begin), MatchEnd: int32(end)}, nil
	}
	// Command PTYs use their locally retained history. Search remains owned by
	// Terminal even in this fallback; shell programs use the same method.
	t.mu.Lock()
	defer t.mu.Unlock()
	total := t.contentTotalLocked()
	text := t.windowEndingLocked(total, total)
	current := max(0, t.windowEndLocked()-height)*cols + max(0, start)
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
	t.pinned, t.viewEnd = true, min(total, top+height)
	return &pb.MethodData{Found: true, Wrapped: wrapped, Rows: t.windowEndingLocked(t.viewEnd, height),
		Offset: int32(t.offsetLocked()), MatchStart: int32(chosen.begin - top*cols), MatchEnd: int32(chosen.end - top*cols)}, nil
}

// HistoryCopy resolves viewport coordinates to provider logical-line ranges,
// so soft-wrapped rows copy as one logical line. Block selection remains a
// visual rectangle and is resolved against the published viewport.
func (t *Terminal) HistoryCopy(ctx context.Context, spec *CopySpec) (string, error) {
	t.historyMu.Lock()
	defer t.historyMu.Unlock()
	if t.historySnapshot == nil || spec == nil || spec.Mode == "block" {
		if spec == nil {
			return t.CopyText(), nil
		}
		text, _ := t.CopyWindow(*spec)
		return text, nil
	}
	t.mu.Lock()
	rows := t.historyRows
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

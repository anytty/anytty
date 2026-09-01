package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/core/history/linehist"
	"github.com/anytty/anytty/core/live"
	"github.com/anytty/anytty/shared/perftrace"
	vterm "github.com/anytty/anytty/vterm/vterm"
)

type Terminal struct {
	mu                   sync.Mutex
	info                 TerminalInfo
	options              TerminalCreateOptions
	process              TerminalProcess
	resourceMu           sync.Mutex
	resourceHistory      []TerminalResourceUsage
	resourceHistoryStart int
	resourceSampling     TerminalResourceSamplingConfig
	resourceCancel       context.CancelFunc
	live                 *live.SurfaceTrack
	// 中文说明：liveOpMu 串行化 live SurfaceTrack 的 PTY 输出与 resize/restart 操作。
	// live 是唯一 response owner；resize 期间产生的新输出必须等 live 调整到新尺寸后再写入。
	liveOpMu       sync.Mutex
	liveRevision   LiveRevision
	liveGeneration uint64
	tap            *SemanticTap
	// tapOpMu serializes typed history delta application. Production no longer
	// owns a second history tap; the name remains for harness compatibility.
	tapOpMu sync.Mutex
	// historyMu 串行化显式 history API 与 terminal close/process-exit 的 store
	// 所有权；锁序固定为 historyMu -> tapOpMu。
	historyMu    sync.Mutex
	historyStore history.HistoryStore
	// 中文说明：lineHistory 是 R436 后唯一的 history 引擎（logical-line 文件历史）。
	// 它在 tapOpMu 临界区内直接消费 tap 事务的 EvictedRows，查询用同一把 gate
	// 采集当前屏热段；vterm emulator 是唯一屏幕真值，没有第二份屏幕模型。
	lineHistory        *linehist.Store
	historyEnabled     bool
	outputConfig       TerminalOutputBufferConfig
	outputBudget       *terminalOutputResidentBudget
	logger             *slog.Logger
	historyStatus      HistoryBacklogStatus
	queueMu            sync.Mutex
	outputBuffer       *terminalOutputBuffer
	historyDeltaQueue  *terminalHistoryDeltaQueue
	liveOutputError    error
	historyOutputError error
	historyStickyError error
	rawPTYMu           sync.Mutex
	rawPTYProcess      TerminalProcess
	rawPTY             *rawPTYBroadcaster
	events             *eventBroker
	update             func(TerminalInfo)
}

func newTerminal(info TerminalInfo, options TerminalCreateOptions, process TerminalProcess, events *eventBroker, update func(TerminalInfo), historyStore history.HistoryStore, historyEnabled bool, outputConfig TerminalOutputBufferConfig, resourceSampling TerminalResourceSamplingConfig, outputBudget *terminalOutputResidentBudget, logger *slog.Logger) *Terminal {
	terminal := &Terminal{
		info:             info.Clone(),
		options:          cloneTerminalCreateOptions(options),
		process:          process,
		resourceSampling: resourceSampling.normalized(),
		events:           events,
		update:           update,
		historyEnabled:   historyEnabled,
		outputConfig:     outputConfig.normalized(),
		outputBudget:     outputBudget,
		logger:           logger,
		rawPTYProcess:    process,
		rawPTY:           newRawPTYBroadcaster(),
	}
	terminal.live = live.NewSurfaceTrackWithOptions(live.SurfaceSize{Cols: int(info.Size.Cols), Rows: int(info.Size.Rows)}, live.SurfaceTrackOptions{
		CaptureLineHistory: historyEnabled,
		OnResponse:         terminal.handleLiveSurfaceResponse,
	})
	if historyEnabled {
		terminal.historyStatus = HistoryBacklogStatus{
			TerminalID:          info.ID,
			HistoryEnabled:      true,
			OutputBufferPolicy:  terminal.outputConfig.Overflow,
			BufferCapacityBytes: terminal.outputConfig.CapacityBytes,
		}
		if lineStore, ok := historyStore.(*linehist.Store); ok {
			terminal.lineHistory = lineStore
			// The composite gate freezes live, drains its typed delta queue, then
			// snapshots the same emulator used to produce those deltas.
			lineStore.Bind(func() linehist.ScreenSnapshot {
				return terminal.live.LineHistoryScreenSnapshot()
			}, &terminalHistoryViewGate{terminal: terminal})
		}
		terminal.historyStore = historyStore
		terminal.historyDeltaQueue = newTerminalHistoryDeltaQueue(terminal.outputConfig)
		go terminal.runHistoryDeltaQueue(terminal.historyDeltaQueue)
	}
	terminal.appendStartMarker(info.CreatedAt)
	terminal.watchProcess(process)
	resourceContext, resourceCancel := context.WithCancel(context.Background())
	terminal.resourceCancel = resourceCancel
	terminal.sampleResourceUsage()
	go terminal.runResourceSampler(resourceContext)
	return terminal
}

func (terminal *Terminal) Info() TerminalInfo {
	terminal.mu.Lock()
	defer terminal.mu.Unlock()
	return terminal.info.Clone()
}

// ResourceUsage 返回当前 terminal 进程的资源诊断采样。domain owner 是
// TerminalProcess/OS 进程；Terminal 只负责在 running 状态下转发采样结果，
// 采样失败不能改变 lifecycle，也不能被用作 running/exited 的判断依据。
func (terminal *Terminal) ResourceUsage() (TerminalResourceUsage, bool) {
	terminal.mu.Lock()
	process := terminal.process
	state := terminal.info.State
	terminal.mu.Unlock()
	if process == nil || state != TerminalStateRunning {
		return TerminalResourceUsage{}, false
	}
	sampler, ok := process.(terminalProcessResourceSampler)
	if !ok {
		return TerminalResourceUsage{}, false
	}
	return sampler.ResourceUsage()
}

// ResourceSnapshot 返回 Terminal 对象持续维护的资源采样快照。历史按时间升序排列，
// 最多保留配置指定数量的样本；调用方拿到的是副本，不能修改 Terminal 内部状态。
func (terminal *Terminal) ResourceSnapshot() (TerminalResourceUsage, []TerminalResourceUsage) {
	terminal.resourceMu.Lock()
	defer terminal.resourceMu.Unlock()
	if len(terminal.resourceHistory) == 0 {
		return TerminalResourceUsage{}, nil
	}
	history := make([]TerminalResourceUsage, len(terminal.resourceHistory))
	start := terminal.resourceHistoryStart
	if start == 0 {
		copy(history, terminal.resourceHistory)
	} else {
		n := copy(history, terminal.resourceHistory[start:])
		copy(history[n:], terminal.resourceHistory[:start])
	}
	return history[len(history)-1], history
}

func (terminal *Terminal) LastOutputAt() time.Time {
	terminal.mu.Lock()
	lastOutputAt := terminal.info.LastOutputAt
	terminal.mu.Unlock()
	return lastOutputAt
}

func (terminal *Terminal) runResourceSampler(ctx context.Context) {
	ticker := time.NewTicker(terminal.resourceSampling.normalized().Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			terminal.sampleResourceUsage()
		}
	}
}

func (terminal *Terminal) sampleResourceUsage() {
	usage, ok := terminal.ResourceUsage()
	if !ok || usage.SampledAt.IsZero() {
		return
	}
	terminal.recordResourceUsage(usage)
}

func (terminal *Terminal) recordResourceUsage(usage TerminalResourceUsage) {
	terminal.resourceMu.Lock()
	defer terminal.resourceMu.Unlock()
	history := terminal.resourceHistory
	if len(history) > 0 {
		latestIndex := len(history) - 1
		if terminal.resourceHistoryStart > 0 {
			latestIndex = (terminal.resourceHistoryStart + len(history) - 1) % len(history)
		}
		latest := history[latestIndex]
		if latest.PID > 0 && usage.PID > 0 && latest.PID != usage.PID {
			history = nil
			terminal.resourceHistoryStart = 0
		} else if !usage.SampledAt.After(latest.SampledAt) {
			return
		}
	}
	limit := terminal.resourceSampling.normalized().MaxSamples
	if len(history) >= limit {
		history[terminal.resourceHistoryStart] = usage
		terminal.resourceHistoryStart = (terminal.resourceHistoryStart + 1) % limit
	} else {
		history = append(history, usage)
	}
	terminal.resourceHistory = history
}

func (terminal *Terminal) resetResourceHistory() {
	terminal.resourceMu.Lock()
	terminal.resourceHistory = nil
	terminal.resourceHistoryStart = 0
	terminal.resourceMu.Unlock()
}

func (terminal *Terminal) SetMetadata(name string, tags map[string]string) TerminalInfo {
	terminal.mu.Lock()
	terminal.info.Name = name
	terminal.info.Tags = cloneStringMap(tags)
	info := terminal.info.Clone()
	terminal.mu.Unlock()
	terminal.syncInfo(info)
	terminal.publish(EventTerminalMetadataChanged, info)
	return info
}

func (terminal *Terminal) Input(data []byte) error {
	terminal.mu.Lock()
	process := terminal.process
	if terminal.info.State == TerminalStateExited || terminal.info.State == TerminalStateRemoved {
		terminal.mu.Unlock()
		return ErrTerminalExited
	}
	terminal.mu.Unlock()
	return process.Input(data)
}

func (terminal *Terminal) handleLiveSurfaceResponse(data []byte) {
	if len(data) == 0 {
		return
	}
	terminal.mu.Lock()
	process := terminal.process
	state := terminal.info.State
	terminal.mu.Unlock()
	if process == nil || state == TerminalStateExited || state == TerminalStateRemoved {
		return
	}
	// 中文说明：OSC/DSR/DA 等终端查询的响应必须回写到当前 PTY，
	// 否则 Codex 这类 TUI 会误判颜色/能力并降级渲染。
	_ = process.Input(data)
}

// IngestOutput 是测试和诊断入口，用一段 PTY 输出同步推进 terminal。
// 它先走 live SurfaceTrack latest 快路径，再把同一 PTY bytes 交给 history semantic
// consumer；history truth 仍只能来自 HistoryStore，不能从 live rows 反推。
func (terminal *Terminal) IngestOutput(output string) error {
	terminal.mu.Lock()
	if terminal.info.State == TerminalStateExited || terminal.info.State == TerminalStateRemoved {
		terminal.mu.Unlock()
		return ErrTerminalExited
	}
	info := terminal.info.Clone()
	process := terminal.process
	terminal.mu.Unlock()
	terminal.publishRawPTYOutput(process, []byte(output))

	revision, synchronized, err := terminal.applyLiveOutput(output)
	if err != nil {
		return err
	}

	if !synchronized {
		terminal.publishLiveInvalidated(info.ID, uint64(revision))
	}
	return nil
}

// Resize 调整 terminal PTY 和 core native live screen 的尺寸。
// resize 和 PTY bytes 按同一顺序进入 live SurfaceTrack 与 history semantic consumer；
// history 只消费 resize transaction 作为 non-history boundary，不能由 resized live snapshot 生成 sealed history。
func (terminal *Terminal) Resize(size Size) error {
	if !size.Valid() {
		return ErrInvalidServerSize
	}
	terminal.mu.Lock()
	process := terminal.process
	if terminal.info.State == TerminalStateExited || terminal.info.State == TerminalStateRemoved {
		terminal.mu.Unlock()
		return ErrTerminalExited
	}
	terminal.mu.Unlock()
	if err := terminal.flushLiveOutput(context.Background()); err != nil {
		return err
	}
	if err := terminal.flushHistoryOutput(context.Background()); err != nil {
		return err
	}

	terminal.liveOpMu.Lock()
	terminal.tapOpMu.Lock()
	defer terminal.tapOpMu.Unlock()
	defer terminal.liveOpMu.Unlock()

	terminal.mu.Lock()
	if terminal.process != process || terminal.info.State == TerminalStateExited || terminal.info.State == TerminalStateRemoved {
		terminal.mu.Unlock()
		return ErrTerminalExited
	}
	terminal.mu.Unlock()
	if err := process.Resize(size); err != nil {
		return err
	}
	terminal.mu.Lock()
	if terminal.process != process || terminal.info.State == TerminalStateExited || terminal.info.State == TerminalStateRemoved {
		terminal.mu.Unlock()
		return ErrTerminalExited
	}
	oldSize := terminal.info.Size
	terminal.info.Size = size
	info := terminal.info.Clone()
	terminal.mu.Unlock()

	tx, hasHistoryDelta := terminal.live.ResizeWithHistory(live.SurfaceSize{Cols: int(size.Cols), Rows: int(size.Rows)})
	terminal.liveRevision++
	terminal.liveGeneration++
	revision := terminal.liveRevision
	if hasHistoryDelta {
		if queue := terminal.currentHistoryDeltaQueue(); queue != nil {
			queue.Enqueue(tx, 1)
		}
	}

	terminal.syncInfo(info)
	terminal.publishResize(info, oldSize, size)
	terminal.publishLiveInvalidated(info.ID, uint64(revision))
	return nil
}

func (terminal *Terminal) Kill() error {
	terminal.mu.Lock()
	process := terminal.process
	terminal.mu.Unlock()
	return process.Kill()
}

func (terminal *Terminal) Close() error {
	return terminal.closeWithReason()
}

func (terminal *Terminal) closeWithReason() error {
	if terminal.resourceCancel != nil {
		terminal.resourceCancel()
	}
	terminal.historyMu.Lock()
	terminal.mu.Lock()
	process := terminal.process
	shouldCloseHistory := terminal.info.State == TerminalStateRunning
	terminal.mu.Unlock()

	var result error
	if shouldCloseHistory && terminal.historyEnabled {
		// Flush captures the shared-buffer sequence at this Close boundary. Bytes
		// produced later, including before process.Close returns, are future output
		// and may be canceled below; the final lineHistory.Close performs the only sync.
		result = errors.Join(result, terminal.flushHistoryOutput(context.Background()))
	}

	terminal.mu.Lock()
	terminal.info.State = TerminalStateRemoved
	info := terminal.info.Clone()
	terminal.mu.Unlock()
	terminal.abortOutputBuffer(process)
	if shouldCloseHistory && terminal.historyEnabled {
		// 中文说明：remove/shutdown 不一定会经过 process-exit watcher；running terminal
		// 的最后 open line 必须交给 linehist 强制闭合。
		result = errors.Join(result, terminal.sealHistoryLifecycleTail(false))
	}
	terminal.closeHistoryDeltaQueue()
	if terminal.lineHistory != nil {
		// 中文说明：terminal remove/shutdown 后不再有查询与续写；linehist 把
		// 未闭合尾部按硬结束落盘并关闭文件，重启进程不丢已滚出内容。
		result = errors.Join(result, terminal.lineHistory.Close())
	}
	terminal.historyMu.Unlock()
	terminal.syncInfo(info)
	if process == nil {
		terminal.finishRawPTYProcess(nil, -1)
		return result
	}
	result = errors.Join(result, process.Close())
	terminal.finishRawPTYProcess(process, -1)
	return result
}

func (terminal *Terminal) Restart(ctx context.Context, factory ProcessFactory) error {
	terminal.mu.Lock()
	info := terminal.info.Clone()
	options := terminal.options
	previousProcess := terminal.process
	terminal.mu.Unlock()
	// 中文说明：restart 生成的是 core 持有的新 terminal process，不能绑定到本次
	// protocol request/session 的 ctx；否则 TUI 退出关闭 socket 会把刚重启的 PTY 杀掉。
	process, err := factory.Spawn(context.Background(), processSpecFromTerminal(info, options))
	if err != nil {
		return err
	}
	// Spawn 成功前旧 generation 必须保持完整可用；只有新 process 已建立后，
	// 才关闭旧 output handoff 并切换所有权。
	terminal.stopAndDrainOutputBuffer(previousProcess)
	terminal.historyMu.Lock()
	if terminal.historyEnabled {
		if err := terminal.flushHistoryOutput(context.Background()); err == nil {
			if err := terminal.sealHistoryLifecycleTail(true); err != nil {
				terminal.recordHistoryUnavailable(err)
			}
		}
	}
	terminal.historyMu.Unlock()
	terminal.mu.Lock()
	old := terminal.process
	oldInfo := terminal.info.Clone()
	terminal.process = process
	terminal.info.State = TerminalStateRunning
	terminal.info.ExitCode = nil
	terminal.info.ExitedAt = time.Time{}
	terminal.info.LastOutputAt = time.Time{}
	info = terminal.info.Clone()
	terminal.mu.Unlock()
	terminal.resetResourceHistory()
	terminal.replaceRawPTYProcess(process, -1)
	terminal.queueMu.Lock()
	terminal.outputBuffer = nil
	wasLiveUnavailable := terminal.liveOutputError != nil
	terminal.liveOutputError = nil
	terminal.historyOutputError = terminal.historyStickyError
	historyAvailable := terminal.historyStickyError == nil
	terminal.queueMu.Unlock()
	terminal.recoverHistoryDeltaQueue()
	terminal.liveOpMu.Lock()
	if wasLiveUnavailable {
		terminal.live = live.NewSurfaceTrackWithOptions(live.SurfaceSize{Cols: int(info.Size.Cols), Rows: int(info.Size.Rows)}, live.SurfaceTrackOptions{
			CaptureLineHistory: terminal.historyEnabled && historyAvailable,
			OnResponse:         terminal.handleLiveSurfaceResponse,
		})
	} else {
		terminal.live.ResetForRestartPreservingScreen()
	}
	terminal.liveRevision++
	terminal.liveGeneration++
	revision := terminal.liveRevision
	terminal.liveOpMu.Unlock()
	if startRevision, ok := terminal.appendStartMarker(time.Now().UTC()); ok {
		revision = startRevision
	}
	terminal.syncInfo(info)
	terminal.watchProcess(process)
	terminal.sampleResourceUsage()
	terminal.publishLifecycle(EventTerminalChanged, info)
	terminal.publishLiveInvalidated(info.ID, uint64(revision))
	if err := old.Close(); err != nil {
		// The new generation is already committed and observable. Cleanup failure
		// must not turn Restart into a retryable whole-operation failure.
		terminal.logger.Warn("close previous terminal process after restart failed",
			"terminal_id", info.ID,
			"state_before", string(oldInfo.State),
			"error_kind", "process_close_failed",
		)
	}
	return nil
}

func (terminal *Terminal) Wait() <-chan ProcessExit {
	terminal.mu.Lock()
	defer terminal.mu.Unlock()
	return terminal.process.Wait()
}

func (terminal *Terminal) LiveRows() []string {
	terminal.liveOpMu.Lock()
	defer terminal.liveOpMu.Unlock()
	return terminal.live.Rows()
}

func (terminal *Terminal) LiveSnapshot() live.SurfaceSnapshot {
	terminal.liveOpMu.Lock()
	defer terminal.liveOpMu.Unlock()
	return terminal.live.Snapshot()
}

func (terminal *Terminal) VisitLiveTrimmedScreenRows(visit func(rowIndex int, cellCount int, cellAt func(int) vterm.Cell)) vterm.TrimmedScreenRowsInfo {
	terminal.liveOpMu.Lock()
	defer terminal.liveOpMu.Unlock()
	return terminal.live.VisitTrimmedScreenRows(visit)
}

// VisitLiveTrimmedScreenRowsWithRevision 在同一 live 锁内读取当前 native screen
// 与 live projection revision。revision 只表示当前屏投影版本，不是 history truth
// generation；protocol/TUI 用它拒绝旧 snapshot，不能把它解释成 logical history 版本。
func (terminal *Terminal) VisitLiveTrimmedScreenRowsWithRevision(visit func(rowIndex int, cellCount int, cellAt func(int) vterm.Cell)) (vterm.TrimmedScreenRowsInfo, uint64) {
	terminal.liveOpMu.Lock()
	defer terminal.liveOpMu.Unlock()
	info := terminal.live.VisitTrimmedScreenRows(visit)
	return info, uint64(terminal.liveRevision)
}

// NativeScreenSnapshot 返回 core 当前 latest native screen。
// 调用方只能把它用于实时显示 projection；history/window/copy truth 必须继续走 HistoryWindow/Copy。
func (terminal *Terminal) NativeScreenSnapshot(terminalID string) NativeScreenSnapshot {
	snapshot, _ := terminal.nativeScreenSnapshotSinceBaseline(terminalID, 0, nil)
	return snapshot
}

func (terminal *Terminal) nativeScreenSnapshotSinceBaseline(terminalID string, observed LiveRevision, base *nativeScreenBaseline) (NativeScreenSnapshot, *nativeScreenBaseline) {
	terminal.liveOpMu.Lock()
	defer terminal.liveOpMu.Unlock()
	if terminal.live == nil {
		snapshot := NativeScreenSnapshot{TerminalID: terminalID, BaseRevision: observed, Revision: terminal.liveRevision, FullReplace: true, Timestamp: time.Now().UTC()}
		return snapshot, &nativeScreenBaseline{terminal: terminal, revision: terminal.liveRevision, generation: terminal.liveGeneration}
	}
	current := terminal.liveRevision
	size := terminal.live.Size()
	nativeSize := NativeScreenSize{Cols: size.Cols, Rows: size.Rows}
	rowHashes := terminal.live.VisualRowHashes()
	altScreen := terminal.live.IsAlternateScreen()
	currentBase := &nativeScreenBaseline{
		terminal: terminal, revision: current, generation: terminal.liveGeneration, size: nativeSize,
		rowHashes: rowHashes, altScreen: altScreen,
	}
	fullReplace := observed == 0 || observed > current || base == nil ||
		base.terminal != terminal || base.revision != observed || base.generation != terminal.liveGeneration ||
		base.size != nativeSize || base.altScreen != altScreen
	var rowCopies []NativeScreenRowCopy
	var replacedRows []int
	if !fullReplace {
		var ok bool
		rowCopies, replacedRows, ok = nativeScreenDeltaRows(base, rowHashes)
		fullReplace = !ok
	}
	if fullReplace {
		replacedRows = make([]int, size.Rows)
		for row := range replacedRows {
			replacedRows[row] = row
		}
		rowCopies = nil
	}
	replaced := make([]bool, size.Rows)
	for _, row := range replacedRows {
		if row >= 0 && row < len(replaced) {
			replaced[row] = true
		}
	}
	rows := make([]NativeScreenRow, 0, size.Rows)
	info := terminal.live.VisitTrimmedScreenRows(func(rowIndex int, cellCount int, cellAt func(int) vterm.Cell) {
		if rowIndex < 0 || rowIndex >= len(replaced) || !replaced[rowIndex] {
			return
		}
		row := NativeScreenRow{Index: rowIndex}
		if cellCount > 0 {
			row.Cells = make([]vterm.Cell, cellCount)
			for index := 0; index < cellCount; index++ {
				row.Cells[index] = cellAt(index)
			}
		}
		rows = append(rows, row)
	})
	return NativeScreenSnapshot{
		TerminalID:   terminalID,
		BaseRevision: observed,
		Revision:     current,
		FullReplace:  fullReplace,
		Size:         NativeScreenSize{Cols: info.Cols, Rows: info.Rows},
		RowCopies:    rowCopies,
		Rows:         rows,
		Cursor:       info.Cursor,
		Modes:        info.Modes,
		AltScreen:    info.IsAlternateScreen,
		Timestamp:    time.Now().UTC(),
	}, currentBase
}

// LiveRevision 返回当前 terminal native screen 的 latest-only revision。
// 它只服务 live invalidation one-shot arm 的“是否已经有新屏幕”判断；
// history/window/copy 不能把它当成 logical history generation。
func (terminal *Terminal) LiveRevision() LiveRevision {
	terminal.liveOpMu.Lock()
	defer terminal.liveOpMu.Unlock()
	return terminal.liveRevision
}

// HistoryBacklogStatus 返回 history consumer 在共享 terminal output buffer 中的诊断。
// R436 后 linehist 在 tap 队列 worker 的 tapOpMu 临界区内同步落盘，
// 没有独立的 compact journal worker；诊断只反映 history 是否启用。
func (terminal *Terminal) HistoryBacklogStatus() HistoryBacklogStatus {
	terminal.mu.Lock()
	terminalID := terminal.info.ID
	terminal.mu.Unlock()
	terminal.queueMu.Lock()
	status := terminal.historyStatus
	queue := terminal.historyDeltaQueue
	if terminal.outputBuffer != nil {
		bufferStatus := terminal.outputBuffer.Status(terminalOutputConsumerLive)
		status.AggregateResidentBytes = bufferStatus.AggregateResidentBytes
		status.AggregateBudgetBytes = bufferStatus.AggregateBudgetBytes
		status.OutputBufferWaitNanos = bufferStatus.WaitNanos
		status.Closed = bufferStatus.Closed
	}
	if queue != nil {
		queueStatus := queue.Status()
		status.OutputBufferPolicy = queueStatus.Policy
		status.BufferCapacityBytes = queueStatus.CapacityBytes
		status.ResidentBytes = queueStatus.ResidentBytes
		status.DroppedBytes += queueStatus.DroppedBytes
		status.GapCount += queueStatus.GapCount
		if queueStatus.Unavailable {
			status.Unavailable = true
			status.UnavailableReason = queueStatus.UnavailableReason
		}
	}
	if terminal.historyOutputError != nil {
		status.Unavailable = true
		status.UnavailableReason = terminal.historyOutputError.Error()
	}
	terminal.queueMu.Unlock()
	status.TerminalID = terminalID
	status.HistoryEnabled = terminal.historyEnabled
	if terminal.historyEnabled {
		terminal.historyMu.Lock()
		status.HistoryEnabled = terminal.historyStore != nil
		terminal.historyMu.Unlock()
	}
	return status
}

func (terminal *Terminal) waitForHistory(ctx context.Context) error {
	terminal.historyMu.Lock()
	defer terminal.historyMu.Unlock()
	return terminal.flushHistoryOutput(ctx)
}

// FlushHistory waits for history payload already in the shared output buffer,
// then establishes an explicit durability fence. Read-only pagination uses
// waitForHistory so it does not turn every history request into an fsync.
func (terminal *Terminal) FlushHistory(ctx context.Context) error {
	terminal.historyMu.Lock()
	defer terminal.historyMu.Unlock()
	if err := terminal.flushHistoryOutput(ctx); err != nil {
		return err
	}
	if terminal.lineHistory == nil {
		return nil
	}
	return terminal.lineHistory.Sync()
}

func (terminal *Terminal) pruneHistoryRetention() error {
	if terminal == nil || terminal.lineHistory == nil {
		return nil
	}
	return terminal.lineHistory.PruneRetention()
}

func (terminal *Terminal) publish(typ EventType, info TerminalInfo) {
	terminal.publishEvent(typ, info, false)
}

func (terminal *Terminal) publishLifecycle(typ EventType, info TerminalInfo) {
	terminal.publishEvent(typ, info, true)
}

func (terminal *Terminal) publishEvent(typ EventType, info TerminalInfo, lifecycleKnown bool) {
	if terminal.events == nil {
		return
	}
	terminalCopy := info.Clone()
	terminal.events.publish(Event{
		Type:           typ,
		TerminalID:     info.ID,
		Terminal:       &terminalCopy,
		LifecycleKnown: lifecycleKnown,
	})
}

func (terminal *Terminal) publishLiveInvalidated(terminalID string, revision uint64) {
	if terminal.events == nil {
		return
	}
	terminal.events.publish(Event{
		Type:       EventTerminalLiveInvalidated,
		TerminalID: terminalID,
		Live: &LiveScreenInvalidated{
			TerminalID: terminalID,
			Revision:   LiveRevision(revision),
		},
	})
}

func (terminal *Terminal) publishResize(info TerminalInfo, oldSize Size, newSize Size) {
	if terminal.events == nil {
		return
	}
	terminalCopy := info.Clone()
	terminal.events.publish(Event{
		Type:       EventTerminalResized,
		TerminalID: info.ID,
		Terminal:   &terminalCopy,
		OldSize:    oldSize,
		NewSize:    newSize,
	})
}

func (terminal *Terminal) watchProcess(process TerminalProcess) {
	outputDone := terminal.watchOutput(process)
	terminal.watchExit(process, outputDone)
}

func (terminal *Terminal) watchOutput(process TerminalProcess) <-chan struct{} {
	done := make(chan struct{})
	output := process.Output()
	if output == nil {
		close(done)
		return done
	}
	// Raw PTY bytes have one consumer: the canonical live/history emulator.
	// History IO consumes typed deltas from historyDeltaQueue instead.
	buffer := newTerminalOutputBuffer(terminal.outputConfig, terminal.outputBudget, false)
	terminal.setOutputBuffer(process, buffer)
	go func() {
		buffer.run(terminalOutputConsumerLive, func(output []byte) error {
			return terminal.ingestProcessLiveOutput(process, string(output))
		}, func(gap terminalOutputGap) error {
			return terminal.markLiveParserStale(process, gap)
		}, func(failure terminalOutputConsumerFailure) {
			terminal.handleOutputConsumerFailure(process, terminalOutputConsumerLive, failure)
		})
	}()
	go func() {
		defer close(done)
		defer func() {
			buffer.Seal()
			buffer.Wait()
			buffer.Close()
			terminal.clearOutputBuffer(buffer)
			process.CancelOutput()
		}()
		for {
			var chunk []byte
			var ok bool
			select {
			case <-buffer.Closed():
				return
			case chunk, ok = <-output:
				if !ok {
					return
				}
			}
			if len(chunk) == 0 {
				continue
			}
			terminal.publishRawPTYOutput(process, chunk)
			if !buffer.Write(chunk) {
				return
			}
		}
	}()
	return done
}

func (terminal *Terminal) setOutputBuffer(process TerminalProcess, buffer *terminalOutputBuffer) {
	terminal.mu.Lock()
	current := terminal.process == process
	terminal.mu.Unlock()
	if !current {
		return
	}
	terminal.queueMu.Lock()
	terminal.outputBuffer = buffer
	terminal.historyStatus.Closed = false
	terminal.queueMu.Unlock()
}

func (terminal *Terminal) clearOutputBuffer(buffer *terminalOutputBuffer) {
	terminal.mu.Lock()
	terminalID := terminal.info.ID
	terminal.mu.Unlock()
	terminal.queueMu.Lock()
	if terminal.outputBuffer == buffer {
		terminal.historyStatus = terminal.historyBacklogStatusFromBufferLocked(buffer)
		if err := buffer.ConsumerError(terminalOutputConsumerLive); err != nil {
			terminal.liveOutputError = terminalOutputErrorForTerminal(err, terminalID)
		}
		if terminal.historyStickyError != nil {
			terminal.historyOutputError = terminal.historyStickyError
		}
		terminal.outputBuffer = nil
	}
	terminal.queueMu.Unlock()
}

func (terminal *Terminal) flushLiveOutput(ctx context.Context) error {
	terminal.queueMu.Lock()
	buffer := terminal.outputBuffer
	err := terminal.liveOutputError
	terminal.queueMu.Unlock()
	if err != nil {
		return err
	}
	if buffer == nil {
		return nil
	}
	return buffer.Flush(ctx, terminalOutputConsumerLive)
}

func (terminal *Terminal) flushHistoryOutput(ctx context.Context) error {
	terminal.queueMu.Lock()
	err := terminal.historyOutputError
	queue := terminal.historyDeltaQueue
	terminal.queueMu.Unlock()
	if err != nil {
		return err
	}
	// First advance the only raw consumer so every byte before this fence has
	// produced its typed history delta, then drain those deltas.
	if err := terminal.flushLiveOutput(ctx); err != nil {
		return err
	}
	finish := perftrace.Measure("core.terminal.output_buffer.history_flush")
	if queue != nil {
		err = queue.Flush(ctx)
	}
	finish(0)
	if err != nil {
		// Queue completion wakes flush waiters before the worker's failure
		// callback records its diagnostic. Preserve the typed public contract in
		// that small window while retaining the underlying storage error.
		terminal.queueMu.Lock()
		outputErr := terminal.historyOutputError
		terminal.queueMu.Unlock()
		if outputErr != nil {
			return outputErr
		}
		terminal.mu.Lock()
		terminalID := terminal.info.ID
		terminal.mu.Unlock()
		return &TerminalOutputError{
			TerminalID: terminalID,
			Consumer:   terminalOutputConsumerHistory.String(),
			Cause:      err,
		}
	}
	return err
}

func (terminal *Terminal) currentHistoryDeltaQueue() *terminalHistoryDeltaQueue {
	if terminal == nil {
		return nil
	}
	terminal.queueMu.Lock()
	defer terminal.queueMu.Unlock()
	return terminal.historyDeltaQueue
}

type terminalHistoryDeltaFailure struct {
	applyErr error
	gapErr   error
}

func (failure *terminalHistoryDeltaFailure) Error() string {
	return errors.Join(failure.applyErr, failure.gapErr).Error()
}

func (failure *terminalHistoryDeltaFailure) Unwrap() []error {
	return []error{failure.applyErr, failure.gapErr}
}

func (failure *terminalHistoryDeltaFailure) HistoryDeltaRecovered() bool {
	return failure != nil && failure.applyErr != nil && failure.gapErr == nil
}

func (terminal *Terminal) runHistoryDeltaQueue(queue *terminalHistoryDeltaQueue) {
	if terminal == nil || queue == nil {
		return
	}
	queue.Run(func(tx vterm.TerminalSemanticTransaction) error {
		terminal.tapOpMu.Lock()
		defer terminal.tapOpMu.Unlock()
		if terminal.lineHistory == nil {
			return nil
		}
		if err := terminal.lineHistory.ApplyTransaction(tx); err != nil {
			gapErr := terminal.lineHistory.ApplyGapBoundary()
			return &terminalHistoryDeltaFailure{applyErr: err, gapErr: gapErr}
		}
		return nil
	}, func() error {
		terminal.tapOpMu.Lock()
		defer terminal.tapOpMu.Unlock()
		if terminal.lineHistory == nil {
			return nil
		}
		return terminal.lineHistory.ApplyGapBoundary()
	}, func(err error, droppedBytes uint64, epoch uint64) {
		terminal.recordHistoryDeltaRecovery(err, droppedBytes, epoch)
	}, func(err error) {
		terminal.recordHistoryDeltaFailure(err)
	})
}

func (terminal *Terminal) recoverHistoryDeltaQueue() {
	if terminal == nil || !terminal.historyEnabled {
		return
	}
	terminal.queueMu.Lock()
	old := terminal.historyDeltaQueue
	sticky := terminal.historyStickyError
	terminal.queueMu.Unlock()
	if sticky != nil {
		return
	}
	if old != nil {
		status := old.Status()
		if !status.Unavailable && !status.Closed {
			return
		}
	}
	next := newTerminalHistoryDeltaQueue(terminal.outputConfig)
	terminal.queueMu.Lock()
	if terminal.historyDeltaQueue != old || terminal.historyStickyError != nil {
		terminal.queueMu.Unlock()
		next.Close()
		return
	}
	if old != nil {
		status := old.Status()
		terminal.historyStatus.DroppedBytes += status.DroppedBytes
		terminal.historyStatus.GapCount += status.GapCount
	}
	terminal.historyDeltaQueue = next
	terminal.historyOutputError = nil
	terminal.queueMu.Unlock()
	if old != nil {
		old.Close()
		old.Wait()
	}
	go terminal.runHistoryDeltaQueue(next)
}

func (terminal *Terminal) closeHistoryDeltaQueue() {
	queue := terminal.currentHistoryDeltaQueue()
	if queue == nil {
		return
	}
	queue.Seal()
	queue.Wait()
	queue.Close()
	terminal.queueMu.Lock()
	terminal.historyStatus.Closed = true
	terminal.queueMu.Unlock()
}

func (terminal *Terminal) recordHistoryDeltaFailure(err error) {
	if err == nil {
		return
	}
	terminal.mu.Lock()
	id := terminal.info.ID
	terminal.mu.Unlock()
	wrapped := &TerminalOutputError{
		TerminalID: id,
		Consumer:   terminalOutputConsumerHistory.String(),
		Cause:      err,
	}
	terminal.queueMu.Lock()
	terminal.historyOutputError = wrapped
	var failure *terminalHistoryDeltaFailure
	if errors.As(err, &failure) && failure.gapErr != nil {
		terminal.historyStickyError = wrapped
	}
	terminal.queueMu.Unlock()
	fields := []any{"terminal_id", id, "error", wrapped, "cause", err}
	if failure != nil {
		fields = append(fields, "apply_error", failure.applyErr, "gap_error", failure.gapErr)
	}
	terminal.logger.Error("terminal history delta consumer unavailable", fields...)
}

func (terminal *Terminal) recordHistoryDeltaRecovery(err error, droppedBytes uint64, epoch uint64) {
	if err == nil {
		return
	}
	terminal.mu.Lock()
	id := terminal.info.ID
	terminal.mu.Unlock()
	fields := []any{
		"terminal_id", id,
		"epoch", epoch,
		"dropped_bytes", droppedBytes,
		"cause", err,
	}
	var failure *terminalHistoryDeltaFailure
	if errors.As(err, &failure) {
		fields = append(fields, "apply_error", failure.applyErr)
	}
	terminal.logger.Warn("terminal history delta recovered after persisted gap", fields...)
}

func (terminal *Terminal) abortOutputBuffer(process TerminalProcess) {
	terminal.queueMu.Lock()
	buffer := terminal.outputBuffer
	terminal.queueMu.Unlock()
	terminal.mu.Lock()
	current := terminal.process == process
	terminal.mu.Unlock()
	if !current {
		return
	}
	if buffer == nil {
		process.CancelOutput()
		return
	}
	terminalID := terminal.Info().ID
	process.CancelOutput()
	buffer.Close()
	buffer.Wait()
	liveStatus := buffer.Status(terminalOutputConsumerLive)
	liveErr := buffer.ConsumerError(terminalOutputConsumerLive)
	if liveErr == nil && liveStatus.PendingGapBytes > 0 {
		liveErr = &TerminalOutputError{
			TerminalID: terminalID, Consumer: terminalOutputConsumerLive.String(),
			Epoch: liveStatus.Epoch + 1, DroppedBytes: liveStatus.PendingGapBytes, Cause: ErrTerminalOutputSyncLost,
		}
	}
	if liveErr != nil {
		terminal.queueMu.Lock()
		terminal.liveOutputError = liveErr
		terminal.queueMu.Unlock()
	}
	terminal.clearOutputBuffer(buffer)
}

// stopAndDrainOutputBuffer closes a successful restart boundary without
// discarding bytes already accepted by the old process generation.
func (terminal *Terminal) stopAndDrainOutputBuffer(process TerminalProcess) {
	terminal.queueMu.Lock()
	buffer := terminal.outputBuffer
	terminal.queueMu.Unlock()
	terminal.mu.Lock()
	current := terminal.process == process
	terminal.mu.Unlock()
	if !current {
		return
	}
	process.CancelOutput()
	if buffer == nil {
		return
	}
	buffer.Seal()
	buffer.Wait()
	buffer.Close()
	terminal.clearOutputBuffer(buffer)
}

func (terminal *Terminal) handleOutputConsumerFailure(process TerminalProcess, consumer terminalOutputConsumer, failure terminalOutputConsumerFailure) {
	if failure.Err == nil {
		return
	}
	terminal.mu.Lock()
	current := terminal.process == process
	id := terminal.info.ID
	terminal.mu.Unlock()
	if !current {
		return
	}
	err := terminalOutputErrorForTerminal(failure.Err, id)
	if consumer == terminalOutputConsumerHistory {
		var outputErr *TerminalOutputError
		if errors.As(err, &outputErr) && outputErr.DroppedBytes > 0 {
			if failure.DuringGap {
				terminal.setStickyHistoryOutputError(process, outputErr.Epoch, outputErr.DroppedBytes, outputErr.Cause)
				err = terminal.historyStickyOutputError()
			} else {
				gap := terminalOutputGap{Consumer: consumer, Epoch: outputErr.Epoch, DroppedBytes: outputErr.DroppedBytes}
				if gapErr := terminal.beginHistoryParserEpoch(process, gap); gapErr != nil {
					terminal.setStickyHistoryOutputError(process, outputErr.Epoch, outputErr.DroppedBytes, errors.Join(outputErr.Cause, fmt.Errorf("persist history output gap: %w", gapErr)))
					err = terminal.historyStickyOutputError()
				}
			}
		}
	}
	terminal.queueMu.Lock()
	if consumer == terminalOutputConsumerLive {
		terminal.liveOutputError = err
	} else if terminal.historyStickyError == nil {
		terminal.historyOutputError = err
	} else {
		terminal.historyOutputError = terminal.historyStickyError
	}
	terminal.queueMu.Unlock()
	terminal.logger.Error("terminal output consumer unavailable", "terminal_id", id, "consumer", consumer.String(), "error", err)
}

func terminalOutputErrorForTerminal(err error, terminalID string) error {
	var outputErr *TerminalOutputError
	if !errors.As(err, &outputErr) {
		return err
	}
	copy := *outputErr
	copy.TerminalID = terminalID
	return &copy
}

func (terminal *Terminal) setStickyHistoryOutputError(process TerminalProcess, epoch uint64, droppedBytes uint64, cause error) {
	terminal.mu.Lock()
	current := terminal.process == process
	terminalID := terminal.info.ID
	terminal.mu.Unlock()
	if !current {
		return
	}
	sticky := &TerminalOutputError{
		TerminalID: terminalID, Consumer: terminalOutputConsumerHistory.String(),
		Epoch: epoch, DroppedBytes: droppedBytes, Cause: cause,
	}
	terminal.queueMu.Lock()
	if terminal.historyStickyError == nil {
		terminal.historyStickyError = sticky
	}
	terminal.historyOutputError = terminal.historyStickyError
	terminal.queueMu.Unlock()
}

func (terminal *Terminal) historyStickyOutputError() error {
	terminal.queueMu.Lock()
	defer terminal.queueMu.Unlock()
	return terminal.historyStickyError
}

func (terminal *Terminal) historyOutputIsSticky() bool {
	terminal.queueMu.Lock()
	defer terminal.queueMu.Unlock()
	return terminal.historyStickyError != nil
}

func (terminal *Terminal) historyBacklogStatusFromBufferLocked(buffer *terminalOutputBuffer) HistoryBacklogStatus {
	status := terminal.historyStatus
	bufferStatus := buffer.Status(terminalOutputConsumerLive)
	status.AggregateResidentBytes = bufferStatus.AggregateResidentBytes
	status.AggregateBudgetBytes = bufferStatus.AggregateBudgetBytes
	status.OutputBufferWaitNanos = bufferStatus.WaitNanos
	status.Closed = bufferStatus.Closed
	return status
}

func (terminal *Terminal) watchExit(process TerminalProcess, outputDone <-chan struct{}) {
	go func() {
		exit, ok := <-process.Wait()
		if !ok {
			return
		}
		if outputDone != nil {
			<-outputDone
		}
		terminal.markExited(process, exit)
	}()
}

func (terminal *Terminal) ingestProcessLiveOutput(process TerminalProcess, output string) error {
	terminal.mu.Lock()
	if terminal.process != process {
		terminal.mu.Unlock()
		return nil
	}
	if terminal.info.State == TerminalStateExited || terminal.info.State == TerminalStateRemoved {
		terminal.mu.Unlock()
		return ErrTerminalExited
	}
	info := terminal.info.Clone()
	terminal.mu.Unlock()

	finishLiveWrite := perftrace.Measure("core.live.write_screen")
	revision, synchronized, err := terminal.applyLiveOutput(output)
	finishLiveWrite(len(output))
	if err != nil {
		return err
	}

	terminal.mu.Lock()
	stillCurrent := terminal.process == process && terminal.info.State != TerminalStateExited && terminal.info.State != TerminalStateRemoved
	terminal.mu.Unlock()
	if !stillCurrent {
		return nil
	}
	// 中文说明：live latest screen 已经由 SurfaceTrack 快路径更新；wake 必须先发出，
	// history semantic write / journal queue / store 写入不能反压用户可见的实时屏幕。
	perftrace.Count("core.terminal.changed", len(output))
	perftrace.Count("core.live.invalidation_publish", len(output))
	if !synchronized {
		terminal.publishLiveInvalidated(info.ID, uint64(revision))
	}
	return nil
}

func (terminal *Terminal) ingestProcessHistoryTapOutput(process TerminalProcess, output string) error {
	terminal.mu.Lock()
	if terminal.process != process {
		terminal.mu.Unlock()
		return nil
	}
	if terminal.info.State == TerminalStateExited || terminal.info.State == TerminalStateRemoved {
		terminal.mu.Unlock()
		return ErrTerminalExited
	}
	terminal.mu.Unlock()
	// Production history no longer replays raw PTY bytes. Keep this harness
	// entry as a generation guard while tests migrate to typed deltas.
	return nil
}

func (terminal *Terminal) markLiveParserStale(process TerminalProcess, gap terminalOutputGap) error {
	terminal.mu.Lock()
	current := terminal.process == process
	terminalID := terminal.info.ID
	terminal.mu.Unlock()
	if !current {
		return ErrTerminalExited
	}
	err := &TerminalOutputError{
		TerminalID: terminalID, Consumer: terminalOutputConsumerLive.String(),
		Epoch: gap.Epoch, DroppedBytes: gap.DroppedBytes, Cause: ErrTerminalOutputSyncLost,
	}
	terminal.queueMu.Lock()
	terminal.liveOutputError = err
	terminal.queueMu.Unlock()
	return err
}

func (terminal *Terminal) beginHistoryParserEpoch(process TerminalProcess, _ terminalOutputGap) error {
	terminal.mu.Lock()
	current := terminal.process == process
	terminal.mu.Unlock()
	if !current {
		return ErrTerminalExited
	}
	if terminal.lineHistory == nil {
		return fmt.Errorf("%w: history storage cannot persist output gap", ErrTerminalOutputUnavailable)
	}
	// Legacy raw-buffer callers can still persist a diagnostic gap, but there is
	// no independent history parser epoch to reset in the single-emulator path.
	return terminal.lineHistory.AppendGapBoundary()
}

func (terminal *Terminal) applyLiveOutput(output string) (LiveRevision, bool, error) {
	terminal.liveOpMu.Lock()
	defer terminal.liveOpMu.Unlock()
	if terminal.live == nil {
		return terminal.liveRevision, false, nil
	}
	result := terminal.live.WriteWithResult(output)
	terminal.liveRevision++
	synchronized := terminal.live.Snapshot().Modes.SynchronizedOutput
	queue := terminal.currentHistoryDeltaQueue()
	remainingBytes := len(output)
	for index, tx := range result.HistoryTransactions {
		if queue != nil {
			remainingTransactions := len(result.HistoryTransactions) - index
			sourceBytes := remainingBytes / remainingTransactions
			if sourceBytes < 1 {
				sourceBytes = 1
			}
			queue.Enqueue(tx, sourceBytes)
			remainingBytes -= sourceBytes
			if remainingBytes < 0 {
				remainingBytes = 0
			}
		}
	}
	return terminal.liveRevision, synchronized, nil
}

func (terminal *Terminal) ingestHistorySemanticOutput(output string) error {
	if !terminal.historyEnabled {
		return nil
	}
	terminal.tapOpMu.Lock()
	if terminal.tap == nil {
		terminal.tapOpMu.Unlock()
		return nil
	}
	finishHistoryWrite := perftrace.Measure("core.history.semantic_write")
	result, err := terminal.tap.ApplyPTYWrite([]byte(output))
	finishHistoryWrite(len(output))
	if err == nil && terminal.lineHistory != nil {
		// 中文说明：EvictedRows 必须在同一 tapOpMu 临界区内落盘——查询用同一把
		// gate 采集冷段计数与当前屏，任一滚出行要么已在文件、要么还在屏上，
		// 不存在两边都看不到的窗口。
		err = terminal.lineHistory.ApplyTransaction(result.tx)
	}
	terminal.tapOpMu.Unlock()
	if err != nil {
		return err
	}
	// 中文说明：linehist 是 R436 后唯一 history 真值路径；落盘已在上面的
	// tapOpMu 临界区内完成，没有 journal/classifier fanout。
	return nil
}

func (terminal *Terminal) markExited(process TerminalProcess, exit ProcessExit) {
	terminal.historyMu.Lock()
	terminal.mu.Lock()
	if terminal.process != process || terminal.info.State == TerminalStateRemoved {
		terminal.mu.Unlock()
		terminal.historyMu.Unlock()
		return
	}
	terminal.info.State = TerminalStateExited
	code := exit.Code
	terminal.info.ExitCode = &code
	// 退出时间以 core-v2 完成输出收口并标记生命周期的 UTC 时刻为准。
	terminal.info.ExitedAt = time.Now().UTC()
	info := terminal.info.Clone()
	terminal.mu.Unlock()
	terminal.finishRawPTYProcess(process, code)
	if terminal.historyEnabled {
		if err := terminal.sealHistoryLifecycleTail(true); err != nil {
			terminal.recordHistoryUnavailable(err)
		}
	}
	terminal.appendExitMarker(info)
	terminal.historyMu.Unlock()
	terminal.syncInfo(info)
	terminal.publish(EventTerminalExited, info)
}

func (terminal *Terminal) subscribeRawPTY(ctx context.Context) *rawPTYSubscription {
	if terminal == nil {
		subscription := newRawPTYSubscription()
		subscription.close(nil, 0, nil)
		return subscription
	}
	terminal.rawPTYMu.Lock()
	broadcaster := terminal.rawPTY
	terminal.rawPTYMu.Unlock()
	if broadcaster == nil {
		subscription := newRawPTYSubscription()
		subscription.close(nil, 0, nil)
		return subscription
	}
	return broadcaster.subscribe(ctx)
}

func (terminal *Terminal) publishRawPTYOutput(process TerminalProcess, raw []byte) {
	if terminal == nil || len(raw) == 0 {
		return
	}
	terminal.rawPTYMu.Lock()
	if terminal.rawPTYProcess != process {
		terminal.rawPTYMu.Unlock()
		return
	}
	broadcaster := terminal.rawPTY
	terminal.rawPTYMu.Unlock()
	terminal.mu.Lock()
	if terminal.process == process {
		terminal.info.LastOutputAt = time.Now().UTC()
	}
	terminal.mu.Unlock()
	broadcaster.publish(raw)
}

func (terminal *Terminal) replaceRawPTYProcess(process TerminalProcess, previousExitCode int) {
	if terminal == nil {
		return
	}
	terminal.rawPTYMu.Lock()
	previous := terminal.rawPTY
	terminal.rawPTYProcess = process
	terminal.rawPTY = newRawPTYBroadcaster()
	terminal.rawPTYMu.Unlock()
	if previous != nil {
		previous.close(&previousExitCode)
	}
}

func (terminal *Terminal) finishRawPTYProcess(process TerminalProcess, exitCode int) {
	if terminal == nil {
		return
	}
	terminal.rawPTYMu.Lock()
	if process != nil && terminal.rawPTYProcess != process {
		terminal.rawPTYMu.Unlock()
		return
	}
	broadcaster := terminal.rawPTY
	terminal.rawPTYMu.Unlock()
	if broadcaster != nil {
		broadcaster.close(&exitCode)
	}
}

func (terminal *Terminal) appendExitMarker(info TerminalInfo) {
	lines := terminalExitMarkerLines(info)
	if len(lines) == 0 {
		return
	}
	terminal.appendLifecycleHistoryMarker(lines, true)
	revision, ok := terminal.appendLifecycleLiveMarker(lines, true)
	if !ok {
		return
	}
	terminal.publishLiveInvalidated(info.ID, uint64(revision))
}

func (terminal *Terminal) appendStartMarker(startedAt time.Time) (LiveRevision, bool) {
	terminal.mu.Lock()
	info := terminal.info.Clone()
	terminal.mu.Unlock()
	lines := terminalStartMarkerLines(info, startedAt)
	terminal.appendLifecycleHistoryMarker(lines, false)
	return terminal.appendLifecycleLiveMarker(lines, false)
}

func (terminal *Terminal) appendLifecycleLiveMarker(lines []string, leadingBlankLine bool) (LiveRevision, bool) {
	if len(lines) == 0 {
		return 0, false
	}
	text := strings.Join(lines, "\r\n") + "\r\n"
	if leadingBlankLine {
		text = "\r\n" + text
	}
	// The marker was appended to linehist explicitly. Render it through the one
	// canonical emulator, but retain persisted ownership so it never enters the
	// typed history-delta stream or the hot-screen history projection.
	terminal.liveOpMu.Lock()
	if terminal.live == nil {
		terminal.liveOpMu.Unlock()
		return 0, false
	}
	terminal.live.WriteHistoryPersisted(text)
	terminal.liveRevision++
	revision := terminal.liveRevision
	terminal.liveOpMu.Unlock()
	return revision, true
}

func (terminal *Terminal) appendLifecycleHistoryMarker(lines []string, leadingBlankLine bool) {
	if terminal.lineHistory == nil || len(lines) == 0 || terminal.historyOutputIsSticky() {
		return
	}
	if leadingBlankLine {
		lines = append([]string{""}, lines...)
	}
	// 中文说明：lifecycle marker 是 core terminal owner 明确写入的历史事件，
	// 不经过 PTY/raw parser，也不从 live screen 反推程序正文。
	if err := terminal.lineHistory.AppendLifecycleLines(lines); err != nil {
		terminal.mu.Lock()
		id := terminal.info.ID
		terminal.mu.Unlock()
		terminal.logger.Warn("append terminal lifecycle marker to history failed", "terminal_id", id, "error", err)
	}
}

func terminalStartMarkerLines(info TerminalInfo, startedAt time.Time) []string {
	if info.ID == "" {
		return nil
	}
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	lines := []string{"terminal started: " + info.ID}
	lines = append(lines, "started at: "+startedAt.UTC().Format(time.RFC3339))
	if len(info.Command) > 0 {
		lines = append(lines, "command: "+strings.Join(info.Command, " "))
	}
	return lines
}

func terminalExitMarkerLines(info TerminalInfo) []string {
	if info.ID == "" {
		return nil
	}
	head := "terminal exited: " + info.ID
	if info.ExitCode != nil {
		head += fmt.Sprintf(" code:%d", *info.ExitCode)
	}
	head += " exited"
	lines := []string{head}
	if !info.ExitedAt.IsZero() {
		lines = append(lines, "exited at: "+info.ExitedAt.UTC().Format(time.RFC3339))
	}
	if len(info.Command) > 0 {
		lines = append(lines, "command: "+strings.Join(info.Command, " "))
	}
	return lines
}

func (terminal *Terminal) HistoryWindow(req history.HistoryWindowRequest) (history.HistoryWindow, error) {
	mode := historyWindowPerfMode(req.Mode)
	finish := perftrace.Measure("core.terminal.history_window." + mode + ".total")
	terminal.historyMu.Lock()
	defer terminal.historyMu.Unlock()
	if !terminal.historyEnabled {
		finish(0)
		return history.HistoryWindow{}, ErrHistoryDisabled
	}
	if terminal.historyStore == nil {
		finish(0)
		return history.HistoryWindow{}, ErrHistoryNotRebuilt
	}
	var window history.HistoryWindow
	var err error
	switch req.Mode {
	case "", history.HistoryWindowModeLatest:
		window, err = terminal.historyStore.LatestWindow(req)
	case history.HistoryWindowModeOlder:
		window, err = terminal.historyStore.OlderWindow(req)
	case history.HistoryWindowModeOldest:
		window, err = terminal.historyStore.OldestWindow(req)
	case history.HistoryWindowModeNewer:
		window, err = terminal.historyStore.NewerWindow(req)
	default:
		finish(0)
		return history.HistoryWindow{}, history.ErrHistoryUnsupportedWindowMode
	}
	finish(len(window.Rows))
	return window, err
}

func (terminal *Terminal) HistoryCopy(req history.HistoryCopyRequest) (string, error) {
	terminal.historyMu.Lock()
	defer terminal.historyMu.Unlock()
	if !terminal.historyEnabled {
		return "", ErrHistoryDisabled
	}
	if terminal.historyStore == nil {
		return "", ErrHistoryNotRebuilt
	}
	return terminal.historyStore.Copy(req)
}

func (terminal *Terminal) HistoryCopyChunk(ctx context.Context, req history.HistoryCopyChunkRequest) (history.HistoryCopyChunkResult, error) {
	terminal.historyMu.Lock()
	defer terminal.historyMu.Unlock()
	if !terminal.historyEnabled {
		return history.HistoryCopyChunkResult{}, ErrHistoryDisabled
	}
	if terminal.historyStore == nil {
		return history.HistoryCopyChunkResult{}, ErrHistoryNotRebuilt
	}
	return terminal.historyStore.CopyChunk(ctx, req)
}

func (terminal *Terminal) HistorySearch(ctx context.Context, req history.HistorySearchRequest) (history.HistorySearchResult, error) {
	terminal.historyMu.Lock()
	defer terminal.historyMu.Unlock()
	if !terminal.historyEnabled {
		return history.HistorySearchResult{}, ErrHistoryDisabled
	}
	if terminal.historyStore == nil {
		return history.HistorySearchResult{}, ErrHistoryNotRebuilt
	}
	return terminal.historyStore.Search(ctx, req)
}

func (terminal *Terminal) HistoryFreeze(req history.FreezeHistoryRequest) (history.FrozenHistorySnapshot, error) {
	finish := perftrace.Measure("core.terminal.history_freeze.total")
	terminal.historyMu.Lock()
	defer terminal.historyMu.Unlock()
	if !terminal.historyEnabled {
		finish(0)
		return history.FrozenHistorySnapshot{}, ErrHistoryDisabled
	}
	if terminal.historyStore == nil {
		finish(0)
		return history.FrozenHistorySnapshot{}, ErrHistoryNotRebuilt
	}
	snapshot, err := terminal.historyStore.Freeze(req)
	finish(int(snapshot.CommittedUpperBound))
	return snapshot, err
}

// HistoryRelease 释放 terminal history store 中的 core-owned token。
func (terminal *Terminal) HistoryRelease(token history.HistoryToken) error {
	terminal.historyMu.Lock()
	defer terminal.historyMu.Unlock()
	if !terminal.historyEnabled {
		return ErrHistoryDisabled
	}
	if terminal.historyStore == nil {
		return ErrHistoryNotRebuilt
	}
	return terminal.historyStore.Release(token)
}

func (terminal *Terminal) sealHistoryLifecycleTail(markPersisted bool) error {
	if terminal.lineHistory != nil && !terminal.historyOutputIsSticky() {
		// 中文说明：process exit/remove/restart 会重置旧 process 的 history tap；
		// 尚未滚出屏幕的最后一屏必须在同一 gate 下封存，否则 live 保留屏幕但
		// copy/history 只能看到冷段尾部，出现旧进程最后几行缺失。
		if err := terminal.lineHistory.SealLifecycleTail(); err != nil {
			return err
		}
		if markPersisted {
			terminal.liveOpMu.Lock()
			if terminal.live != nil {
				terminal.live.MarkCurrentScreenHistoryPersisted()
			}
			terminal.liveOpMu.Unlock()
		}
	}
	return nil
}

func (terminal *Terminal) recordHistoryUnavailable(err error) {
	if err == nil {
		return
	}
	terminal.mu.Lock()
	id := terminal.info.ID
	terminal.mu.Unlock()
	wrapped := &TerminalOutputError{
		TerminalID: id,
		Consumer:   terminalOutputConsumerHistory.String(),
		Cause:      err,
	}
	terminal.queueMu.Lock()
	terminal.historyOutputError = errors.Join(terminal.historyOutputError, wrapped)
	terminal.queueMu.Unlock()
	terminal.logger.Error("terminal history unavailable", "terminal_id", id, "error", wrapped)
}

func (terminal *Terminal) syncInfo(info TerminalInfo) {
	if terminal.update != nil {
		terminal.update(info)
	}
}

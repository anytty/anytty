package terminalhost

import (
	"log/slog"
	"os"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/shared/perftrace"
	"github.com/anytty/anytty/tui/render"
)

// LatestFrameSink 把真实 TTY 写帧变成 latest-only 背压边界。
// 前台只更新最新待写帧；后台写不过来时，中间帧会被新帧覆盖。
type LatestFrameSink struct {
	mu      sync.Mutex
	cond    *sync.Cond
	sink    render.FrameSink
	pending *latestFrameSinkItem
	patches []latestFrameSinkItem
	closed  bool
	wg      sync.WaitGroup

	logger        *slog.Logger
	diagEnabled   bool
	diagInterval  time.Duration
	lastDiag      time.Time
	highWaterMark int
}

type latestFrameSinkItem struct {
	frame render.Frame
	done  chan render.FrameWriteCompletion
}

func NewLatestFrameSink(sink render.FrameSink) *LatestFrameSink {
	writer := &LatestFrameSink{sink: sink}
	writer.cond = sync.NewCond(&writer.mu)
	writer.wg.Add(1)
	go writer.loop()
	return writer
}

func (sink *LatestFrameSink) SetLogger(logger *slog.Logger) {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	sink.logger = logger
	sink.diagEnabled = logger != nil && latestFrameSinkDiagnosticsEnabled()
	sink.diagInterval = latestFrameSinkDiagnosticsInterval()
}

func (sink *LatestFrameSink) NeedsCompleteFrame() bool {
	preference, ok := sink.sink.(render.FrameSinkPreference)
	if !ok {
		return true
	}
	return preference.NeedsCompleteFrame()
}

func (sink *LatestFrameSink) WriteFrame(frame render.Frame) error {
	_, err := sink.WriteFrameWithCompletion(frame)
	return err
}

func (sink *LatestFrameSink) WriteFrameWithCompletion(frame render.Frame) (<-chan render.FrameWriteCompletion, error) {
	cloned := frame.Clone()
	done := make(chan render.FrameWriteCompletion, 1)
	sink.mu.Lock()
	if sink.closed {
		sink.mu.Unlock()
		completeLatestFrameSinkItem(done, false)
		return done, nil
	}
	item := latestFrameSinkItem{frame: cloned, done: done}
	if cloned.Patch != nil {
		// 中文说明：增量滚动 patch 依赖顺序，不能像完整 live frame 一样丢中间帧。
		sink.patches = append(sink.patches, item)
	} else {
		// 中文说明：完整帧是绝对状态；它会覆盖之前还没写出的增量 patch。
		if sink.pending != nil {
			perftrace.Count("tui.frame_sink_overwrite", latestFrameApproxBytes(sink.pending.frame))
			completeLatestFrameSinkItem(sink.pending.done, false)
		}
		for _, patch := range sink.patches {
			completeLatestFrameSinkItem(patch.done, false)
		}
		sink.pending = &item
		sink.patches = nil
	}
	sink.observeQueueLocked(cloned.Patch != nil)
	sink.cond.Signal()
	sink.mu.Unlock()
	return done, nil
}

func (sink *LatestFrameSink) Close() {
	sink.mu.Lock()
	sink.closed = true
	if sink.pending != nil {
		completeLatestFrameSinkItem(sink.pending.done, false)
		sink.pending = nil
	}
	for _, patch := range sink.patches {
		completeLatestFrameSinkItem(patch.done, false)
	}
	sink.patches = nil
	sink.cond.Broadcast()
	sink.mu.Unlock()
	sink.wg.Wait()
}

func (sink *LatestFrameSink) loop() {
	defer sink.wg.Done()
	for {
		item, ok := sink.next()
		if !ok {
			return
		}
		var err error
		if sink.sink != nil {
			finishWrite := perftrace.Measure("tui.frame_sink_write")
			err = sink.sink.WriteFrame(item.frame)
			finishWrite(latestFrameApproxBytes(item.frame))
		}
		if err == nil {
			perftrace.Count("tui.frame_sink_written", latestFrameApproxBytes(item.frame))
		}
		completeLatestFrameSinkItemWithError(item.done, err == nil, err)
	}
}

func completeLatestFrameSinkItem(done chan render.FrameWriteCompletion, written bool) {
	completeLatestFrameSinkItemWithError(done, written, nil)
}

func completeLatestFrameSinkItemWithError(done chan render.FrameWriteCompletion, written bool, err error) {
	if done == nil {
		return
	}
	done <- render.FrameWriteCompletion{Written: written, Err: err}
	close(done)
}

func (sink *LatestFrameSink) next() (latestFrameSinkItem, bool) {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	for sink.pending == nil && len(sink.patches) == 0 && !sink.closed {
		sink.cond.Wait()
	}
	if sink.pending == nil && len(sink.patches) == 0 && sink.closed {
		return latestFrameSinkItem{}, false
	}
	if sink.pending != nil {
		item := *sink.pending
		sink.pending = nil
		return item, true
	}
	item := sink.patches[0]
	copy(sink.patches, sink.patches[1:])
	last := len(sink.patches) - 1
	sink.patches[last] = latestFrameSinkItem{}
	if last == 0 {
		sink.patches = nil
		return item, true
	}
	sink.patches = sink.patches[:last]
	return item, true
}

func (sink *LatestFrameSink) observeQueueLocked(patch bool) {
	if !sink.diagEnabled || sink.logger == nil {
		return
	}
	queueLen := len(sink.patches)
	force := false
	if queueLen > sink.highWaterMark {
		sink.highWaterMark = queueLen
		force = queueLen == 1 || queueLen%128 == 0
	}
	now := time.Now()
	if !force && now.Sub(sink.lastDiag) < sink.diagInterval {
		return
	}
	sink.lastDiag = now
	var mem goruntime.MemStats
	goruntime.ReadMemStats(&mem)
	sink.logger.Debug("tui-v3 latest-frame-sink diagnostic",
		"patch", patch,
		"pending_complete", sink.pending != nil,
		"patch_queue_len", queueLen,
		"patch_queue_high_water", sink.highWaterMark,
		"queued_patch_bytes", latestFrameSinkPatchBytes(sink.patches),
		"heap_alloc", mem.HeapAlloc,
		"heap_objects", mem.HeapObjects,
		"goroutines", goruntime.NumGoroutine(),
	)
}

func latestFrameSinkDiagnosticsEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ANYTTY_TUI_DIAG"))) {
	case "1", "true", "on", "yes", "debug":
		return true
	default:
		return false
	}
}

func latestFrameSinkDiagnosticsInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("ANYTTY_TUI_DIAG_INTERVAL_MS"))
	if raw == "" {
		return time.Second
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return time.Second
	}
	return time.Duration(value) * time.Millisecond
}

func latestFrameSinkPatchBytes(items []latestFrameSinkItem) int {
	total := 0
	for _, item := range items {
		frame := item.frame
		if frame.Patch == nil {
			continue
		}
		total += len(frame.Patch.LineANSI)
		for _, line := range frame.Patch.LinesANSI {
			total += len(line)
		}
	}
	return total
}

func latestFrameApproxBytes(frame render.Frame) int {
	if frame.Patch != nil {
		return latestFrameSinkPatchBytes([]latestFrameSinkItem{{frame: frame}})
	}
	total := 0
	for _, line := range frame.Lines {
		total += len(line)
	}
	for _, line := range frame.ANSILines {
		total += len(line)
	}
	return total
}

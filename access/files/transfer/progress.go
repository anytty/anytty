package transfer

import "time"

// State 是结构化进度事件的状态字段。
type State string

const (
	// StateActive 表示传输中。
	StateActive State = "active"
	// StateCompleted 表示已完成。
	StateCompleted State = "completed"
)

// Progress 是合并后的结构化进度投影：已传字节、总量、速率与剩余时间。
// 它是通知，不是 durable truth；恢复真值永远是 transfer record 的 Offset。
type Progress struct {
	Transferred      int64
	Total            int64
	RateBytesPerSec  int64
	RemainingSeconds int64
	State            State
}

// Coalescer 按时间窗合并进度更新，避免每个 chunk 都推一帧。
// 它不做 IO；调用方把 Observe 返回 true 的结果编码成可选进度事件。
type Coalescer struct {
	interval  time.Duration
	lastEmit  time.Time
	lastBytes int64
	lastRate  int64
	started   time.Time
}

// NewCoalescer 创建进度合并器；interval<=0 时默认 250ms。
func NewCoalescer(interval time.Duration) *Coalescer {
	if interval <= 0 {
		interval = 250 * time.Millisecond
	}
	return &Coalescer{interval: interval}
}

// Observe 记录一次进度；返回是否应当发送事件。
// 第一次调用总是发送；总量未知或未完成时按时间窗合并；达到总量时总是发送。
func (coalescer *Coalescer) Observe(transferred int64, total int64, now time.Time) (Progress, bool) {
	if coalescer == nil {
		return Progress{Transferred: transferred, Total: total, State: StateActive}, true
	}
	if now.IsZero() {
		now = time.Now()
	}
	if coalescer.started.IsZero() {
		coalescer.started = now
	}
	complete := total > 0 && transferred >= total
	first := coalescer.lastEmit.IsZero()
	if !first && !complete && now.Sub(coalescer.lastEmit) < coalescer.interval {
		return Progress{}, false
	}
	progress := coalescer.snapshotLocked(transferred, total, now, complete)
	coalescer.lastEmit = now
	coalescer.lastBytes = transferred
	coalescer.lastRate = progress.RateBytesPerSec
	return progress, true
}

// Finish 生成终态进度（总是发送）。
func (coalescer *Coalescer) Finish(transferred int64, total int64, now time.Time) Progress {
	if now.IsZero() {
		now = time.Now()
	}
	if coalescer == nil {
		return Progress{Transferred: transferred, Total: total, State: StateCompleted}
	}
	progress := coalescer.snapshotLocked(transferred, total, now, true)
	coalescer.lastEmit = now
	coalescer.lastBytes = transferred
	coalescer.lastRate = progress.RateBytesPerSec
	return progress
}

func (coalescer *Coalescer) snapshotLocked(transferred int64, total int64, now time.Time, complete bool) Progress {
	progress := Progress{Transferred: transferred, Total: total, State: StateActive}
	if complete {
		progress.State = StateCompleted
	}
	deltaBytes := transferred - coalescer.lastBytes
	deltaTime := now.Sub(coalescer.lastEmit)
	if !coalescer.lastEmit.IsZero() && deltaBytes > 0 && deltaTime > 0 {
		progress.RateBytesPerSec = int64(float64(deltaBytes) / deltaTime.Seconds())
	} else {
		progress.RateBytesPerSec = coalescer.lastRate
	}
	if progress.RateBytesPerSec <= 0 && !coalescer.started.IsZero() {
		elapsed := now.Sub(coalescer.started)
		if elapsed > 0 && transferred > 0 {
			progress.RateBytesPerSec = int64(float64(transferred) / elapsed.Seconds())
		}
	}
	if total > 0 && progress.RateBytesPerSec > 0 && transferred < total {
		remaining := total - transferred
		progress.RemainingSeconds = int64(float64(remaining)/float64(progress.RateBytesPerSec) + 0.999)
	}
	if complete {
		progress.RemainingSeconds = 0
	}
	return progress
}

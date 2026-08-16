package perftrace

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	envPath         = "ANYTTY_PERF_TRACE"
	envIntervalMs   = "ANYTTY_PERF_TRACE_INTERVAL_MS"
	envBucketMs     = "ANYTTY_PERF_TRACE_BUCKET_MS"
	defaultBucketMs = 100
	maxBuckets      = 6000
)

type Recorder struct {
	startedAt  time.Time
	bucketSize time.Duration

	mu      sync.Mutex
	events  map[string]*eventStats
	buckets map[int64]map[string]*eventStats
}

type eventStats struct {
	count uint64
	bytes uint64
	total time.Duration
	max   time.Duration
}

type Snapshot struct {
	StartedAt   time.Time        `json:"started_at"`
	CollectedAt time.Time        `json:"collected_at"`
	ElapsedMs   float64          `json:"elapsed_ms"`
	BucketMs    float64          `json:"bucket_ms,omitempty"`
	Events      []EventSnapshot  `json:"events"`
	Buckets     []BucketSnapshot `json:"buckets,omitempty"`
}

// TraceRecord 是 ANYTTY_PERF_TRACE=.jsonl 统一日志中的单条进程快照。
// 它携带进程角色和 PID，允许 daemon 与 TUI 继承同一路径并追加写入，
// 诊断工具再按 process/pid/sequence 还原各自的耗时链路。
type TraceRecord struct {
	Format    string    `json:"format"`
	Process   string    `json:"process"`
	PID       int       `json:"pid"`
	Sequence  uint64    `json:"sequence"`
	Reason    string    `json:"reason,omitempty"`
	WrittenAt time.Time `json:"written_at"`
	Snapshot  Snapshot  `json:"snapshot"`
}

type EventSnapshot struct {
	Name      string  `json:"name"`
	Count     uint64  `json:"count"`
	Bytes     uint64  `json:"bytes"`
	TotalMs   float64 `json:"total_ms"`
	AverageMs float64 `json:"average_ms"`
	MaxMs     float64 `json:"max_ms"`
}

// BucketSnapshot 是 perftrace 的短时间桶视图，用来定位高压输出下的瞬时卡顿。
// 它只在显式开启 ANYTTY_PERF_TRACE 时产生，不参与业务状态或历史 truth。
type BucketSnapshot struct {
	Index   int64           `json:"index"`
	StartMs float64         `json:"start_ms"`
	EndMs   float64         `json:"end_ms"`
	Events  []EventSnapshot `json:"events"`
}

var active atomic.Pointer[Recorder]

const traceRecordFormat = "anytty.perftrace.snapshot.v1"

func NewRecorder() *Recorder {
	return &Recorder{
		startedAt:  time.Now().UTC(),
		bucketSize: time.Duration(defaultBucketMs) * time.Millisecond,
		events:     make(map[string]*eventStats),
		buckets:    make(map[int64]map[string]*eventStats),
	}
}

func Enable() *Recorder {
	recorder := NewRecorder()
	active.Store(recorder)
	return recorder
}

// EnableFromEnv 按 ANYTTY_PERF_TRACE 打开进程内聚合性能采样。
// 返回的 stop 必须在进程退出前调用；未设置环境变量时保持默认空操作。
// 路径以 .jsonl 结尾时追加写多进程统一日志；其他路径保持旧的 JSON 快照覆盖写。
func EnableFromEnv(ctx context.Context) (func(), string, bool) {
	return EnableFromEnvWithProcess(ctx, defaultProcessName())
}

// EnableFromEnvWithProcess 按 ANYTTY_PERF_TRACE 打开带进程角色的性能采样。
// process 是统一日志里的消息链路归属，例如 core-v2-daemon 或 tui-v3；
// 当 daemon/TUI 共享同一个 .jsonl 路径时，写入边界由追加记录表达，不互相覆盖。
func EnableFromEnvWithProcess(ctx context.Context, process string) (func(), string, bool) {
	path := strings.TrimSpace(os.Getenv(envPath))
	if path == "" {
		return func() {}, "", false
	}
	process = strings.TrimSpace(process)
	if process == "" {
		process = defaultProcessName()
	}
	recorder := NewRecorder()
	recorder.bucketSize = bucketIntervalFromEnv()
	active.Store(recorder)
	done := make(chan struct{})
	stopOnce := sync.Once{}
	writer := newEnvWriter(path, process)
	write := func(reason string) {
		_ = writer.Write(recorder.Snapshot(), reason)
	}
	interval := intervalFromEnv()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case <-ticker.C:
				write("periodic")
			}
		}
	}()
	stop := func() {
		stopOnce.Do(func() {
			close(done)
			write("final")
			Disable()
		})
	}
	return stop, path, true
}

func Disable() {
	active.Store(nil)
}

func Current() *Recorder {
	return active.Load()
}

func Reset() {
	recorder := Current()
	if recorder == nil {
		return
	}
	recorder.Reset()
}

func Measure(name string) func(bytes int) {
	recorder := Current()
	if recorder == nil || name == "" {
		return func(int) {}
	}
	start := time.Now()
	return func(bytes int) {
		recorder.observe(name, time.Since(start), bytes)
	}
}

func Count(name string, bytes int) {
	recorder := Current()
	if recorder == nil || name == "" {
		return
	}
	recorder.observe(name, 0, bytes)
}

func SnapshotCurrent() Snapshot {
	recorder := Current()
	if recorder == nil {
		return Snapshot{}
	}
	return recorder.Snapshot()
}

func WriteJSON(path string, snapshot Snapshot) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (r *Recorder) WriteJSON(path string) error {
	if r == nil {
		return nil
	}
	return WriteJSON(path, r.Snapshot())
}

type envWriter struct {
	path    string
	process string
	jsonl   bool
	seq     atomic.Uint64
}

func newEnvWriter(path, process string) *envWriter {
	return &envWriter{path: path, process: process, jsonl: perfTraceJSONLinesPath(path)}
}

func (w *envWriter) Write(snapshot Snapshot, reason string) error {
	if w == nil || w.path == "" {
		return nil
	}
	if !w.jsonl {
		return WriteJSON(w.path, snapshot)
	}
	return appendTraceRecord(w.path, TraceRecord{
		Format:    traceRecordFormat,
		Process:   w.process,
		PID:       os.Getpid(),
		Sequence:  w.seq.Add(1),
		Reason:    reason,
		WrittenAt: time.Now().UTC(),
		Snapshot:  snapshot,
	})
}

func appendTraceRecord(path string, record TraceRecord) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

func perfTraceJSONLinesPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".jsonl" || ext == ".ndjson"
}

func defaultProcessName() string {
	exe, err := os.Executable()
	if err == nil {
		name := strings.TrimSpace(filepath.Base(exe))
		if name != "" {
			return name
		}
	}
	return "anytty"
}

func intervalFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv(envIntervalMs))
	if raw == "" {
		return time.Second
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return time.Second
	}
	return time.Duration(value) * time.Millisecond
}

func bucketIntervalFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv(envBucketMs))
	if raw == "" {
		return time.Duration(defaultBucketMs) * time.Millisecond
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return time.Duration(defaultBucketMs) * time.Millisecond
	}
	return time.Duration(value) * time.Millisecond
}

func (r *Recorder) Reset() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.startedAt = time.Now().UTC()
	r.events = make(map[string]*eventStats)
	r.buckets = make(map[int64]map[string]*eventStats)
	r.mu.Unlock()
}

func (r *Recorder) Snapshot() Snapshot {
	if r == nil {
		return Snapshot{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	events := snapshotEvents(r.events)
	buckets := r.snapshotBucketsLocked()

	return Snapshot{
		StartedAt:   r.startedAt,
		CollectedAt: time.Now().UTC(),
		ElapsedMs:   durationMillis(time.Since(r.startedAt)),
		BucketMs:    durationMillis(r.bucketSize),
		Events:      events,
		Buckets:     buckets,
	}
}

func (s Snapshot) Event(name string) (EventSnapshot, bool) {
	for _, event := range s.Events {
		if event.Name == name {
			return event, true
		}
	}
	return EventSnapshot{}, false
}

func (r *Recorder) observe(name string, duration time.Duration, bytes int) {
	if r == nil || name == "" {
		return
	}
	now := time.Now()
	r.mu.Lock()
	stats := r.events[name]
	if stats == nil {
		stats = &eventStats{}
		r.events[name] = stats
	}
	observeStats(stats, duration, bytes)
	if r.bucketSize > 0 {
		index := int64(now.Sub(r.startedAt) / r.bucketSize)
		if index < 0 {
			index = 0
		}
		bucket := r.buckets[index]
		if bucket == nil {
			bucket = make(map[string]*eventStats)
			r.buckets[index] = bucket
			r.trimBucketsLocked(index)
		}
		bucketStats := bucket[name]
		if bucketStats == nil {
			bucketStats = &eventStats{}
			bucket[name] = bucketStats
		}
		observeStats(bucketStats, duration, bytes)
	}
	r.mu.Unlock()
}

func (r *Recorder) trimBucketsLocked(latest int64) {
	if maxBuckets <= 0 {
		return
	}
	min := latest - int64(maxBuckets) + 1
	for index := range r.buckets {
		if index < min {
			delete(r.buckets, index)
		}
	}
}

func (r *Recorder) snapshotBucketsLocked() []BucketSnapshot {
	if r == nil || r.bucketSize <= 0 || len(r.buckets) == 0 {
		return nil
	}
	indexes := make([]int64, 0, len(r.buckets))
	for index := range r.buckets {
		indexes = append(indexes, index)
	}
	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i] < indexes[j]
	})
	out := make([]BucketSnapshot, 0, len(indexes))
	sizeMs := durationMillis(r.bucketSize)
	for _, index := range indexes {
		out = append(out, BucketSnapshot{
			Index:   index,
			StartMs: float64(index) * sizeMs,
			EndMs:   float64(index+1) * sizeMs,
			Events:  snapshotEvents(r.buckets[index]),
		})
	}
	return out
}

func snapshotEvents(values map[string]*eventStats) []EventSnapshot {
	events := make([]EventSnapshot, 0, len(values))
	for name, stats := range values {
		event := EventSnapshot{
			Name:    name,
			Count:   stats.count,
			Bytes:   stats.bytes,
			TotalMs: durationMillis(stats.total),
			MaxMs:   durationMillis(stats.max),
		}
		if stats.count > 0 {
			event.AverageMs = durationMillis(stats.total) / float64(stats.count)
		}
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].Name < events[j].Name
	})
	return events
}

func observeStats(stats *eventStats, duration time.Duration, bytes int) {
	stats.count++
	if bytes > 0 {
		stats.bytes += uint64(bytes)
	}
	stats.total += duration
	if duration > stats.max {
		stats.max = duration
	}
}

func durationMillis(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

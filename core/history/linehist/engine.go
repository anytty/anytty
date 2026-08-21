package linehist

import (
	"context"
	"errors"
	"sync"
	"time"

	vterm "github.com/anytty/anytty/vterm/vterm"
)

var errRetentionChanged = errors.New("history retention changed")

// Engine 组合拼装器与文件 backend，是 logical-line 历史的落盘入口。
// 它只消费 vterm 事务的 EvictedRows（seal-on-eviction 信号），
// 不读 raw PTY、不读 live snapshot、不持有第二份屏幕状态。
type Engine struct {
	mu   sync.Mutex
	asm  *Assembler
	file LineStorage
}

// LineStorage 是 linehist engine 所需的最小持久化契约。
type LineStorage interface {
	AppendLines([]Line) error
	AppendBoundary() error
	LineCount() int
	Base() int
	Lines(start int, end int) ([]Line, error)
	Sync() error
	Close() error
}

type lineVisitor interface {
	VisitLines(start int, end int, reverse bool, visit func(index int, line Line) bool) error
}

type searchLineBatchReader interface {
	readSearchBatch(start int, end int, cursor int, reverse bool, filters []searchQueryGrams) (firstLine int, lines []Line, nextCursor int, err error)
}

type retentionPruner interface {
	PruneRetention() error
}

type retentionTracker interface {
	RetentionEpoch() uint64
}

type gapStorage interface {
	AppendGap() error
	GapOffsets() []int
}

// NewEngine 用已打开的 LineStorage 创建引擎。文件里已有的记录即恢复出的
// 冷历史；未闭合尾部不跨进程恢复（重启时旧行的续写上下文已不存在，
// Close 会把它按硬结束落盘）。
func NewEngine(file LineStorage) *Engine {
	return &Engine{asm: NewAssembler(), file: file}
}

// ApplyEvictedRows 按事务顺序消费滚出的物理行，把拼装完成的 logical line
// 批量落盘。滚出行的完整性（空行保留、alt 归因、ring 关闭）由 vterm
// EvictedRows 保证，这里不再做任何过滤或对账。
func (e *Engine) ApplyEvictedRows(rows []vterm.TerminalSemanticScrollOut) error {
	if e == nil || len(rows) == 0 {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	var batch []Line
	for _, row := range rows {
		batch = append(batch, e.asm.AppendEvictedRow(row)...)
	}
	return e.file.AppendLines(batch)
}

// LineCount 返回已落盘记录数（含 chunk 记录），绝对域。
func (e *Engine) LineCount() int {
	if e == nil {
		return 0
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.file.LineCount()
}

// VisibleLineRange 返回当前可见冷段记录区间。R439 后 ED3/ClearScrollback
// 只是不删除历史的软页边界，所以 live history/copy 继续覆盖全部已落盘
// logical lines；base 保留为绝对域起点，当前始终为 0。
func (e *Engine) VisibleLineRange() (int, int) {
	if e == nil {
		return 0, 0
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	base := e.file.Base()
	return base, e.file.LineCount() - base
}

// RetentionEpoch 在持久化后端淘汰最旧历史时递增。Store 用它废止仍引用
// 已淘汰 cold range 的 frozen token。
func (e *Engine) RetentionEpoch() uint64 {
	if e == nil {
		return 0
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return storageRetentionEpoch(e.file)
}

// PruneRetention 立即执行后端配置的物理淘汰策略。
func (e *Engine) PruneRetention() error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if pruner, ok := e.file.(retentionPruner); ok {
		return pruner.PruneRetention()
	}
	return nil
}

// ApplyClearScrollbackBoundary 处理 ED3/ClearScrollback 软页边界。
// authoritative history 不因终端 clear-scrollback 被删除或隐藏；这里仅把
// assembler 的未闭合尾部封成一条 logical line，避免 clear 前后页面被继续
// 拼接，然后写入 append-only boundary 记录供 generation/cursor 失效使用。
func (e *Engine) ApplyClearScrollbackBoundary() error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if line, ok := e.asm.SealOpen(); ok {
		if err := e.file.AppendLines([]Line{line}); err != nil {
			return err
		}
	}
	return e.file.AppendBoundary()
}

func (e *Engine) AppendGapBoundary() error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if line, ok := e.asm.SealOpen(); ok {
		if err := e.file.AppendLines([]Line{line}); err != nil {
			return err
		}
	}
	storage, ok := e.file.(gapStorage)
	if !ok {
		return errors.New("history storage cannot persist output gaps")
	}
	return storage.AppendGap()
}

func (e *Engine) GapOffsets() []int {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	storage, ok := e.file.(gapStorage)
	if !ok {
		return nil
	}
	return storage.GapOffsets()
}

// Lines 按绝对序号分页读取已落盘记录。
func (e *Engine) Lines(start int, end int) ([]Line, error) {
	if e == nil {
		return nil, nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.file.Lines(start, end)
}

// LinesAtRetention 把 epoch 校验与分页读放在同一临界区内，避免 range 捕获后
// 恰好发生前缀淘汰时，把旧坐标错误映射到新文件的另一批行。
func (e *Engine) LinesAtRetention(epoch uint64, start int, end int) ([]Line, error) {
	if e == nil {
		return nil, nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if storageRetentionEpoch(e.file) != epoch {
		return nil, errRetentionChanged
	}
	return e.file.Lines(start, end)
}

// VisitLinesAtRetention streams a retained range without materializing the
// complete page. The storage keeps block I/O batched while the caller can stop
// before projecting rows that exceed its response budget.
func (e *Engine) VisitLinesAtRetention(epoch uint64, start int, end int, reverse bool, visit func(index int, line Line) bool) error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if storageRetentionEpoch(e.file) != epoch {
		return errRetentionChanged
	}
	if storage, ok := e.file.(lineVisitor); ok {
		return storage.VisitLines(start, end, reverse, visit)
	}
	if reverse {
		for index := end - 1; index >= start; index-- {
			lines, err := e.file.Lines(index, index+1)
			if err != nil {
				return err
			}
			if len(lines) > 0 && !visit(index, lines[0]) {
				break
			}
		}
		return nil
	}
	for index := start; index < end; index++ {
		lines, err := e.file.Lines(index, index+1)
		if err != nil {
			return err
		}
		if len(lines) > 0 && !visit(index, lines[0]) {
			break
		}
	}
	return nil
}

// VisitLinesAtRetentionBatched releases the engine lock between storage
// batches. Long-running search therefore does not hold up PTY ingestion for
// the duration of the complete scan.
func (e *Engine) VisitLinesAtRetentionBatched(ctx context.Context, epoch uint64, start int, end int, reverse bool, batchSize int, visit func(index int, line Line) bool) error {
	if e == nil || end <= start {
		return nil
	}
	if batchSize <= 0 {
		batchSize = 256
	}
	if reverse {
		for batchEnd := end; batchEnd > start; {
			if err := ctx.Err(); err != nil {
				return err
			}
			batchStart := maxInt(start, batchEnd-batchSize)
			lines, err := e.LinesAtRetention(epoch, batchStart, batchEnd)
			if err != nil {
				return err
			}
			for index := len(lines) - 1; index >= 0; index-- {
				if !visit(batchStart+index, lines[index]) {
					return nil
				}
			}
			batchEnd = batchStart
		}
		return nil
	}
	for batchStart := start; batchStart < end; {
		if err := ctx.Err(); err != nil {
			return err
		}
		batchEnd := minInt(end, batchStart+batchSize)
		lines, err := e.LinesAtRetention(epoch, batchStart, batchEnd)
		if err != nil {
			return err
		}
		for index, line := range lines {
			if !visit(batchStart+index, line) {
				return nil
			}
		}
		batchStart = batchEnd
	}
	return nil
}

// VisitSearchLinesAtRetention visits one physical storage block per lock hold.
// Compressed storage can reject a block through its trigram filter without
// decompressing it; legacy/test storage keeps the generic 256-line fallback.
func (e *Engine) VisitSearchLinesAtRetention(ctx context.Context, epoch uint64, start int, end int, reverse bool, requiredLiterals []string, visit func(index int, line Line) bool) error {
	if e == nil || end <= start {
		return nil
	}
	e.mu.Lock()
	reader, ok := e.file.(searchLineBatchReader)
	e.mu.Unlock()
	if !ok {
		return e.VisitLinesAtRetentionBatched(ctx, epoch, start, end, reverse, 256, visit)
	}
	filters := make([]searchQueryGrams, 0, len(requiredLiterals))
	for _, literal := range requiredLiterals {
		filters = append(filters, querySearchGrams(literal))
	}
	cursor := start
	if reverse {
		cursor = end
	}
	for (reverse && cursor > start) || (!reverse && cursor < end) {
		if err := ctx.Err(); err != nil {
			return err
		}
		e.mu.Lock()
		if storageRetentionEpoch(e.file) != epoch {
			e.mu.Unlock()
			return errRetentionChanged
		}
		firstLine, lines, nextCursor, err := reader.readSearchBatch(start, end, cursor, reverse, filters)
		e.mu.Unlock()
		if err != nil {
			return err
		}
		if reverse {
			for index := len(lines) - 1; index >= 0; index-- {
				if !visit(firstLine+index, lines[index]) {
					return nil
				}
			}
			if nextCursor >= cursor {
				return errors.New("history reverse search batch did not advance")
			}
		} else {
			for index, line := range lines {
				if !visit(firstLine+index, line) {
					return nil
				}
			}
			if nextCursor <= cursor {
				return errors.New("history forward search batch did not advance")
			}
		}
		cursor = nextCursor
	}
	return nil
}

func storageRetentionEpoch(file LineStorage) uint64 {
	if tracker, ok := file.(retentionTracker); ok {
		return tracker.RetentionEpoch()
	}
	return 0
}

// OpenTail 返回未闭合尾部（已滚出但还没等到硬换行的续行内容）。
// 投影层把它拼在冷段之后、emulator 当前屏之前，作为热段一部分。
func (e *Engine) OpenTail() []Run {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.asm.Open()
}

func (e *Engine) OpenTailUpdatedAt() time.Time {
	if e == nil {
		return time.Time{}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.asm.OpenUpdatedAt()
}

// SealOpenTail 把未闭合尾部强制闭合落盘，用于进程退出等边界。
func (e *Engine) SealOpenTail() error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	line, ok := e.asm.SealOpen()
	if !ok {
		return nil
	}
	return e.file.AppendLines([]Line{line})
}

// SealPrimaryScreenRows 在 terminal lifecycle 边界把仍停留在 primary 当前屏的
// 热段行封存为 cold logical lines。消息链路仍是同一个 Assembler：已滚出的
// open tail 与当前屏 Wrapped 行按同一规则拼接；alt 当前屏不应传入本方法。
func (e *Engine) SealPrimaryScreenRows(rows []ScreenRow) error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	var batch []Line
	for _, row := range rows {
		batch = append(batch, e.asm.AppendEvictedRow(vterm.TerminalSemanticScrollOut{
			Cells:      cloneScreenRowCells(row.Cells),
			Timestamp:  row.UpdatedAt,
			Wrapped:    row.Wrapped,
			WrappedSet: true,
		})...)
	}
	if line, ok := e.asm.SealOpen(); ok {
		batch = append(batch, line)
	}
	if len(batch) == 0 {
		return nil
	}
	return e.file.AppendLines(batch)
}

// AppendLifecycleLines 追加 core terminal lifecycle marker。它不是 PTY 输出，
// 而是 core-v2 lifecycle owner 明确写入的历史事件；调用方必须只用于
// start/exit/restart 这类用户可见 terminal 边界，不能拿来补程序正文。
func (e *Engine) AppendLifecycleLines(texts []string) error {
	if e == nil || len(texts) == 0 {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	batch := make([]Line, 0, len(texts))
	for _, text := range texts {
		batch = append(batch, Line{
			Runs:      []Run{{Text: text}},
			HardEnd:   true,
			UpdatedAt: time.Now().UTC(),
		})
	}
	return e.file.AppendLines(batch)
}

// Sync flushes pending logical-line blocks and establishes the storage
// durability fence without sealing the assembler's open tail.
func (e *Engine) Sync() error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.file.Sync()
}

// Close 把未闭合尾部按硬结束落盘后关闭文件：重启后旧行的续写上下文
// 已不存在，把已滚出的内容留在内存里只会丢数据。
func (e *Engine) Close() error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	var result error
	if line, ok := e.asm.SealOpen(); ok {
		result = errors.Join(result, e.file.AppendLines([]Line{line}))
	}
	return errors.Join(result, e.file.Close())
}

func cloneScreenRowCells(cells []vterm.Cell) []vterm.Cell {
	if len(cells) == 0 {
		return nil
	}
	out := make([]vterm.Cell, len(cells))
	copy(out, cells)
	return out
}

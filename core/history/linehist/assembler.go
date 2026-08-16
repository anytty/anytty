package linehist

import (
	"time"

	vterm "github.com/anytty/anytty/vterm/vterm"
)

// DefaultMaxOpenLineBytes 是未闭合 logical line 的 chunk-flush 阈值。
// 它只是病态输入（永不硬换行的超长行）的安全阀：达到阈值就落一条
// HardEnd=false 的 chunk，投影时与后继记录连续拼接。正常输出永远达不到。
const DefaultMaxOpenLineBytes = 256 * 1024

// Assembler 把滚出的物理行按软换行标志拼成 logical line。
// 它是纯拼装器：不持有屏幕状态，不判断内容归属——滚出行的完整性由
// vterm EvictedRows 保证（含空行、alt 归因、ring 关闭场景）。
type Assembler struct {
	open      []Run
	openBytes int
	maxOpen   int
	updatedAt time.Time
}

// NewAssembler 创建 logical line 拼装器。
func NewAssembler() *Assembler {
	return &Assembler{maxOpen: DefaultMaxOpenLineBytes}
}

// AppendEvictedRow 消费一条滚出的物理行，返回本次拼装完成、可落盘的记录。
// Wrapped=true 的行是软换行续行：并入未闭合行；Wrapped=false 表示硬换行
// 结束，未闭合内容连同本行一起闭合为一条 logical line。空行（无 payload
// 且硬结束）产出空 Runs 的 Line，保持段落间距。
func (a *Assembler) AppendEvictedRow(row vterm.TerminalSemanticScrollOut) []Line {
	if a == nil {
		return nil
	}
	if a.maxOpen <= 0 {
		a.maxOpen = DefaultMaxOpenLineBytes
	}
	for _, run := range runsFromScrollOut(row) {
		a.open = appendMergedRun(a.open, run)
		a.openBytes += len(run.Text)
	}
	if row.Timestamp.After(a.updatedAt) {
		a.updatedAt = row.Timestamp
	}
	wrapped := row.WrappedSet && row.Wrapped
	if wrapped {
		if a.openBytes < a.maxOpen {
			return nil
		}
		chunk := Line{Runs: a.open, HardEnd: false, UpdatedAt: a.updatedAt}
		a.open = nil
		a.openBytes = 0
		a.updatedAt = time.Time{}
		return []Line{chunk}
	}
	line := Line{Runs: trimTrailingPaddingRuns(a.open), HardEnd: true, UpdatedAt: a.updatedAt}
	a.open = nil
	a.openBytes = 0
	a.updatedAt = time.Time{}
	return []Line{line}
}

// Open 返回未闭合尾部（已滚出但还没等到硬换行的续行内容）的副本。
// 投影层把它作为热段一部分参与展示；它不是第二份真值，只是拼装中间态。
func (a *Assembler) Open() []Run {
	if a == nil || len(a.open) == 0 {
		return nil
	}
	out := make([]Run, len(a.open))
	copy(out, a.open)
	return out
}

func (a *Assembler) OpenUpdatedAt() time.Time {
	if a == nil {
		return time.Time{}
	}
	return a.updatedAt
}

// SealOpen 把未闭合尾部强制闭合为一条硬结束 line（用于进程退出等边界：
// 续写上下文已不存在，再等续行没有意义）。没有未闭合内容时返回 false。
func (a *Assembler) SealOpen() (Line, bool) {
	if a == nil || len(a.open) == 0 {
		return Line{}, false
	}
	line := Line{Runs: trimTrailingPaddingRuns(a.open), HardEnd: true, UpdatedAt: a.updatedAt}
	a.open = nil
	a.openBytes = 0
	a.updatedAt = time.Time{}
	return line, true
}

// DiscardOpen 丢弃未闭合尾部。它只适用于显式 reset/放弃 frontier 的
// 语义；ED3/ClearScrollback 不是删除 authoritative history，不能走这里。
func (a *Assembler) DiscardOpen() {
	if a == nil {
		return
	}
	a.open = nil
	a.openBytes = 0
	a.updatedAt = time.Time{}
}

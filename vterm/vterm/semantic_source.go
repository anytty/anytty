package vterm

import (
	"sort"
	"time"
)

// TerminalSemanticSource 是 vterm 对 core-v2 暴露的终端语义事务源。
// domain owner：vterm 解码 PTY bytes；history 只能消费 transaction，不能回放 raw。
type TerminalSemanticSource interface {
	ApplyPTYWrite(raw []byte) (TerminalSemanticTransaction, error)
	Resize(size TerminalSemanticSize) (TerminalSemanticTransaction, error)
}

type TerminalSemanticSize struct {
	Cols int
	Rows int
}

type TerminalSemanticOp = DamageOp
type TerminalSemanticCell = Cell
type TerminalSemanticStyle = CellStyle

// TerminalSemanticCellRun 是 vterm 在 scroll-out proof 中保留的 styled text run。
// domain owner：vterm；core 只能消费它展开历史 payload，不能把 raw PTY 当 fallback。
type TerminalSemanticCellRun = CellRun

// TerminalSemanticScrollOut 是同一个 transaction 内 primary 内容离开可见区域的证明。
// Cells 是 cell 级 payload；Runs 是 vterm 在 raw damage 中保留的 styled text payload。
// core 只能把这些 payload 复制进 logical-line history，不能从 current frame 反推历史。
type TerminalSemanticScrollOut struct {
	Cells     []TerminalSemanticCell
	Runs      []TerminalSemanticCellRun
	Timestamp time.Time
	// Row/RowSet 表示该 proof 来源于清屏/滚动前的 primary viewport row。
	// core-v2 只能用它过滤 current primary frame ownership，不能据此反推新内容。
	Row        int
	RowSet     bool
	Wrapped    bool
	WrappedSet bool
}

type TerminalSemanticFrame struct {
	Rows [][]TerminalSemanticCell
	Cols int
}

// TerminalSemanticTransaction 是一次 PTY write/resize 的 ordered semantic 边界。
// Raw 只用于上层 shadow state，不是 history fallback 输入。
type TerminalSemanticTransaction struct {
	Seq  uint64
	Raw  string
	Size TerminalSemanticSize

	Ops []TerminalSemanticOp

	PrimaryScrollOut []TerminalSemanticScrollOut
	// EvictedRows 是本次 transaction 中真正离开 primary 可见区、不可能再被程序寻址
	// 修改的物理行，按离屏顺序排列。它直接来自 vterm eviction 捕获
	// （damage.ScrollbackAppend），不经过 PrimaryScrollOut 的 sync/alt/clear gate，
	// 因此普通 stdout 换行滚出也会出现在这里。alt 屏滚动被 vterm 导流到
	// AlternateAppend，不进入 EvictedRows。
	//
	// 这是 logical-line 无限历史引擎的唯一 seal 信号：滚出即落盘、落盘即不可变。
	// 每行携带 cells/runs 与 Wrapped 软换行标志，供上层把物理行拼成宽度无关的
	// logical line。core 只能顺序消费它 append 历史，不能从 current frame 反推。
	EvictedRows []TerminalSemanticScrollOut

	PrimaryFrame *TerminalSemanticFrame
	// PrimaryFrameTouchedRows 表示本次 transaction 确认触达的 primary frame 行。
	// truth source 是 vterm direct damage/ordered ops；core 只能按这些行接管
	// current frame ownership，不能把整张 PrimaryFrame 快照当作新历史。
	PrimaryFrameTouchedRows []int
	AltFrame                *TerminalSemanticFrame
	AltExitFrame            *TerminalSemanticFrame

	AltEntered        bool
	AltExited         bool
	SynchronizedBegin bool
	// SynchronizedActive 表示 transaction 结束后 mode 2026 仍处于开启状态。
	// core classifier 用它识别 begin/end 分片之间的 payload，不回看 raw bytes。
	SynchronizedActive  bool
	SynchronizedEnd     bool
	RequiresFullReplace bool
	FullReplaceReason   string
	// ClearScrollback 表示 PTY 明确发出了清除 terminal scrollback/history 的语义
	// 边界，例如 ED3。它不是 vterm 内部 scrollback 容量淘汰，core 只能在该字段为
	// true 时清 authoritative scrollback/history projection。
	ClearScrollback bool

	SourceDamage WriteDamage
}

type SemanticSource struct {
	vt           *VTerm
	seq          uint64
	evictionOnly bool
}

func NewSemanticSource(cols int, rows int, scrollbackSize int, onResponse ResponseHandler) *SemanticSource {
	return &SemanticSource{vt: New(cols, rows, scrollbackSize, onResponse)}
}

// NewLineHistorySemanticSource 创建 linehist 生产路径使用的语义源。
// 它仍由同一个 vterm emulator 解释 PTY bytes，但 transaction 只携带
// EvictedRows/ClearScrollback 等 linehist 真值边界，避免普通大输出把 ordered
// text ops payload 重复放进 history backlog。
func NewLineHistorySemanticSource(cols int, rows int, scrollbackSize int, onResponse ResponseHandler) *SemanticSource {
	return &SemanticSource{vt: New(cols, rows, scrollbackSize, onResponse), evictionOnly: true}
}

func NewSemanticSourceFromVTerm(vt *VTerm) *SemanticSource {
	if vt == nil {
		vt = New(80, 24, 0, nil)
	}
	return &SemanticSource{vt: vt}
}

// NewLineHistorySemanticSourceFromVTerm 以现有 vterm state 继续 linehist 生产
// ingest。用于 terminal restart 后保留当前屏幕，但丢弃旧进程 parser state。
func NewLineHistorySemanticSourceFromVTerm(vt *VTerm) *SemanticSource {
	if vt == nil {
		vt = New(80, 24, 0, nil)
	}
	return &SemanticSource{vt: vt, evictionOnly: true}
}

func (source *SemanticSource) VTerm() *VTerm {
	if source == nil {
		return nil
	}
	return source.vt
}

func (source *SemanticSource) ApplyPTYWrite(raw []byte) (TerminalSemanticTransaction, error) {
	if source == nil {
		return TerminalSemanticTransaction{}, nil
	}
	source.ensureVTerm()
	var damage WriteDamage
	var err error
	if source.evictionOnly {
		_, err, damage = source.vt.WriteForLineHistory(raw)
	} else {
		_, err, damage = source.vt.WriteWithSemanticDamage(raw)
	}
	source.seq++
	return source.transactionFromDamage(source.seq, string(raw), damage), err
}

// ApplyPTYWriteWithLiveDamage advances the emulator exactly once and returns
// both the line-history transaction and the latest-screen damage produced by
// that write. It is the single-emulator production path: history consumes the
// immutable transaction while live projection consumes the row damage.
func (source *SemanticSource) ApplyPTYWriteWithLiveDamage(raw []byte) (TerminalSemanticTransaction, WriteDamage, error) {
	if source == nil {
		return TerminalSemanticTransaction{}, WriteDamage{}, nil
	}
	source.ensureVTerm()
	_, err, damage := source.vt.WriteForLineHistory(raw)
	source.seq++
	return source.transactionFromDamage(source.seq, string(raw), damage), damage, err
}

func (source *SemanticSource) Resize(size TerminalSemanticSize) (TerminalSemanticTransaction, error) {
	if source == nil || size.Cols <= 0 || size.Rows <= 0 {
		return TerminalSemanticTransaction{}, nil
	}
	source.ensureVTerm()
	damage := source.vt.ResizeWithDamage(size.Cols, size.Rows)
	source.seq++
	tx := source.transactionFromDamage(source.seq, "", damage)
	tx.Size = size
	return tx, nil
}

func (source *SemanticSource) ensureVTerm() {
	if source.vt == nil {
		source.vt = New(80, 24, 0, nil)
	}
}

func (source *SemanticSource) transactionFromDamage(seq uint64, raw string, damage WriteDamage) TerminalSemanticTransaction {
	size := TerminalSemanticSize{Cols: damage.SizeCols, Rows: damage.SizeRows}
	if size.Cols == 0 || size.Rows == 0 {
		size = source.currentSize()
	}
	ops := semanticOpsForTransactionDamage(damage)
	tx := TerminalSemanticTransaction{
		Seq:                     seq,
		Raw:                     raw,
		Size:                    size,
		Ops:                     ops,
		PrimaryFrameTouchedRows: terminalSemanticPrimaryFrameTouchedRows(damage, ops, size),
		RequiresFullReplace:     damage.RequiresFullReplace,
		FullReplaceReason:       damage.FullReplaceReason,
		ClearScrollback:         terminalSemanticDamageHasClearScrollback(damage),
	}
	// 中文说明：EvictedRows 无条件携带本次 write/resize 真正离屏的行，
	// 不复用 PrimaryScrollOut 的 attach gate。优先取保留空行的 EvictedAppend，
	// 无则回退 ScrollbackAppend（ring-diff/resize 路径本身保留空行）。
	// EvictedRows 是 logical-line 历史引擎的 seal 信号。
	evictedSource := damage.EvictedAppend
	if len(evictedSource) == 0 {
		evictedSource = damage.ScrollbackAppend
	}
	for _, scrollOut := range evictedSource {
		tx.EvictedRows = append(tx.EvictedRows, terminalSemanticScrollOutFromDamageOp(scrollOut))
	}
	// 中文说明：WriteDamage 是本次 write 的临时语义 payload，transaction 在
	// SemanticSource 边界接管它的所有权；tap 之后的 fan-out deep-copy 仍由
	// SemanticTapResult.Transaction 负责，避免 live hot path 多做一轮 op/cell clone。
	if len(damage.SemanticOps) > 0 {
		damage.SemanticOps = nil
	} else {
		damage.Ops = nil
	}
	// 中文说明：SourceDamage 只保留诊断摘要。payload truth 已在 Ops、必要的
	// PrimaryScrollOut side proof 和 frame side proof 中，不能把完整 damage
	// payload 再放进 tap 后 backlog，避免高压普通输出重复拷贝第二份语义数据。
	tx.SourceDamage = terminalSemanticSourceDamageSummary(damage)
	for _, op := range tx.Ops {
		for _, scrollOut := range op.ScrollOut {
			tx.PrimaryScrollOut = append(tx.PrimaryScrollOut, terminalSemanticScrollOutFromAppend(scrollOut))
		}
	}
	tx.AltEntered = terminalSemanticDamageHasAltMode(damage, true)
	tx.AltExited = terminalSemanticDamageHasAltMode(damage, false)
	tx.SynchronizedBegin = terminalSemanticDamageHasSyncMode(damage, true)
	tx.SynchronizedEnd = terminalSemanticDamageHasSyncMode(damage, false)
	if source.vt != nil {
		tx.SynchronizedActive = source.vt.Modes().SynchronizedOutput
	}
	if terminalSemanticShouldAttachTransactionScrollOut(damage, tx) {
		for _, scrollOut := range damage.ScrollbackAppend {
			proof := terminalSemanticScrollOutFromDamageOp(scrollOut)
			if terminalSemanticScrollOutAlreadyIncluded(tx.PrimaryScrollOut, proof) {
				continue
			}
			tx.PrimaryScrollOut = append(tx.PrimaryScrollOut, proof)
		}
	}
	if source.vt != nil && terminalSemanticShouldAttachFrame(damage, tx) {
		screen := source.vt.UsedScreenContent()
		frame := &TerminalSemanticFrame{Rows: cloneCellRows(screen.Cells), Cols: size.Cols}
		if screen.IsAlternateScreen {
			tx.AltFrame = frame
		} else {
			tx.PrimaryFrame = frame
		}
	}
	return tx
}

func terminalSemanticScrollOutFromAppend(scrollOut ScrollbackRowAppend) TerminalSemanticScrollOut {
	return TerminalSemanticScrollOut{
		Cells:      cloneCellSlice(scrollOut.Cells),
		Runs:       cloneCellRuns(scrollOut.Runs),
		Timestamp:  scrollOut.Timestamp,
		Row:        scrollOut.Row,
		RowSet:     scrollOut.RowSet,
		Wrapped:    scrollOut.Wrapped,
		WrappedSet: scrollOut.WrappedSet,
	}
}

func terminalSemanticScrollOutFromDamageOp(scrollOut DamageOp) TerminalSemanticScrollOut {
	return TerminalSemanticScrollOut{
		Cells:      cloneCellSlice(scrollOut.Cells),
		Runs:       cloneCellRuns(scrollOut.Runs),
		Timestamp:  scrollOut.Timestamp,
		Row:        scrollOut.Row,
		RowSet:     scrollOut.RowSet,
		Wrapped:    scrollOut.Wrapped,
		WrappedSet: scrollOut.WrappedSet,
	}
}

func terminalSemanticScrollOutAlreadyIncluded(existing []TerminalSemanticScrollOut, proof TerminalSemanticScrollOut) bool {
	for _, current := range existing {
		if terminalSemanticScrollOutEqual(current, proof) {
			return true
		}
	}
	return false
}

func terminalSemanticScrollOutEqual(left TerminalSemanticScrollOut, right TerminalSemanticScrollOut) bool {
	if left.Row != right.Row || left.RowSet != right.RowSet || left.Wrapped != right.Wrapped || left.WrappedSet != right.WrappedSet || !left.Timestamp.Equal(right.Timestamp) {
		return false
	}
	if len(left.Cells) != len(right.Cells) || len(left.Runs) != len(right.Runs) {
		return false
	}
	for i := range left.Cells {
		if left.Cells[i] != right.Cells[i] {
			return false
		}
	}
	for i := range left.Runs {
		if left.Runs[i] != right.Runs[i] {
			return false
		}
	}
	return true
}

func terminalSemanticShouldAttachTransactionScrollOut(damage WriteDamage, tx TerminalSemanticTransaction) bool {
	if len(damage.ScrollbackAppend) == 0 {
		return false
	}
	if tx.AltEntered || tx.AltExited || tx.SynchronizedBegin || tx.SynchronizedActive || tx.SynchronizedEnd || tx.RequiresFullReplace {
		return true
	}
	for _, op := range tx.Ops {
		if op.Code != ScreenOpControl {
			continue
		}
		switch op.Control {
		case "ed", "ris":
			return true
		}
	}
	return false
}

func terminalSemanticShouldAttachFrame(damage WriteDamage, tx TerminalSemanticTransaction) bool {
	if tx.AltEntered || tx.AltExited || tx.SynchronizedBegin || tx.SynchronizedActive || tx.SynchronizedEnd {
		return true
	}
	if tx.RequiresFullReplace {
		return true
	}
	// 中文说明：frame attach 的 owner 边界只能看本次 transaction 暴露给
	// history 的 ordered semantic ops。WriteDamage.Ops 可能包含 live diff
	// 层的屏幕损伤摘要，普通单调 stdout 不能因为 diff 触达行而升级成 frame。
	ops := tx.Ops
	if len(ops) == 0 {
		ops = semanticOpsForTransactionDamage(damage)
	}
	for _, op := range ops {
		if op.Code == ScreenOpModes || op.Code == ScreenOpResize {
			return true
		}
		switch op.Code {
		case ScreenOpClearToEOL, ScreenOpClearRect, ScreenOpScrollRect, ScreenOpCopyRect:
			return true
		}
		if op.Code == ScreenOpControl {
			switch op.Control {
			case "ed", "ris", "cup", "vpa", "hpa", "cha",
				"cub", "cuf", "cuu", "cud",
				"el", "ech", "dch", "ich", "il", "dl", "su", "sd":
				return true
			}
		}
	}
	return false
}

func terminalSemanticPrimaryFrameTouchedRows(damage WriteDamage, ops []TerminalSemanticOp, size TerminalSemanticSize) []int {
	if damage.RequiresFullReplace {
		return cloneIntSlice(damage.DirectDamageTouchedRows)
	}
	// 中文说明：非 full-replace 的 current-frame ownership proof 必须优先来自
	// ordered semantic ops。live direct damage rows 只能作为没有 ordered proof
	// 时的诊断兜底，不能把普通 screen diff 扩大成 history frame ownership。
	if rows := terminalSemanticTouchedRowsFromOps(ops, size); len(rows) > 0 {
		return rows
	}
	return cloneIntSlice(damage.DirectDamageTouchedRows)
}

func terminalSemanticTouchedRowsFromOps(ops []TerminalSemanticOp, size TerminalSemanticSize) []int {
	if len(ops) == 0 {
		return nil
	}
	touched := make(map[int]struct{}, len(ops))
	mark := func(row int) {
		if row < 0 {
			return
		}
		if size.Rows > 0 && row >= size.Rows {
			return
		}
		touched[row] = struct{}{}
	}
	markRange := func(start int, end int) {
		if end < start {
			start, end = end, start
		}
		if size.Rows > 0 && end > size.Rows {
			end = size.Rows
		}
		for row := start; row < end; row++ {
			mark(row)
		}
	}
	for _, op := range ops {
		switch op.Code {
		case ScreenOpWriteSpan, ScreenOpClearToEOL:
			mark(op.Row)
		case ScreenOpClearRect:
			markRange(op.Rect.Y, op.Rect.Y+op.Rect.Height)
		case ScreenOpScrollRect:
			markRange(op.Rect.Y, op.Rect.Y+op.Rect.Height)
		case ScreenOpCopyRect:
			markRange(op.DstY, op.DstY+op.Src.Height)
		case ScreenOpControl:
			switch op.Control {
			case "ed":
				switch op.Mode {
				case 1:
					markRange(0, op.Row+1)
				case 2, 3:
					markRange(0, size.Rows)
				default:
					markRange(op.Row, size.Rows)
				}
			case "el", "ech", "dch", "ich", "ri":
				mark(op.Row)
			case "il", "dl", "su", "sd":
				if op.Bottom > op.Row {
					markRange(op.Row, op.Bottom)
				} else {
					mark(op.Row)
				}
			case "ris":
				markRange(0, size.Rows)
			}
		}
	}
	if len(touched) == 0 {
		return nil
	}
	rows := make([]int, 0, len(touched))
	for row := range touched {
		rows = append(rows, row)
	}
	sort.Ints(rows)
	return rows
}

func terminalSemanticSourceDamageSummary(damage WriteDamage) WriteDamage {
	return WriteDamage{
		LiveTailAppendRows:      damage.LiveTailAppendRows,
		ResizeLiveTailRows:      damage.ResizeLiveTailRows,
		ScrollbackTrim:          damage.ScrollbackTrim,
		ScreenScroll:            damage.ScreenScroll,
		RequiresFullReplace:     damage.RequiresFullReplace,
		FullReplaceReason:       damage.FullReplaceReason,
		DirectDamageItems:       damage.DirectDamageItems,
		DirectDamageRows:        damage.DirectDamageRows,
		DirectDamageCells:       damage.DirectDamageCells,
		DirectDamageTouchedRows: cloneIntSlice(damage.DirectDamageTouchedRows),
		Cursor:                  damage.Cursor,
		Modes:                   damage.Modes,
		SizeCols:                damage.SizeCols,
		SizeRows:                damage.SizeRows,
		DiffCPUNanos:            damage.DiffCPUNanos,
	}
}

func (source *SemanticSource) currentSize() TerminalSemanticSize {
	if source == nil || source.vt == nil {
		return TerminalSemanticSize{}
	}
	source.vt.mu.RLock()
	defer source.vt.mu.RUnlock()
	if source.vt.emu == nil {
		return TerminalSemanticSize{}
	}
	return TerminalSemanticSize{Cols: source.vt.emu.Width(), Rows: source.vt.emu.Height()}
}

func semanticOpsForTransactionDamage(damage WriteDamage) []DamageOp {
	if len(damage.SemanticOps) > 0 {
		return damage.SemanticOps
	}
	return damage.Ops
}

func terminalSemanticDamageHasAltMode(damage WriteDamage, enabled bool) bool {
	for _, op := range semanticOpsForTransactionDamage(damage) {
		if op.Code == ScreenOpModes && op.Private && (op.Mode == 47 || op.Mode == 1047 || op.Mode == 1049) && op.Enabled == enabled {
			return true
		}
	}
	return false
}

func terminalSemanticDamageHasSyncMode(damage WriteDamage, enabled bool) bool {
	for _, op := range semanticOpsForTransactionDamage(damage) {
		if op.Code == ScreenOpModes && op.Private && op.Mode == 2026 && op.Enabled == enabled {
			return true
		}
	}
	return false
}

func terminalSemanticDamageHasClearScrollback(damage WriteDamage) bool {
	for _, op := range semanticOpsForTransactionDamage(damage) {
		if op.Code == ScreenOpControl && op.Control == "ed" && op.Mode == 3 {
			return true
		}
	}
	return false
}

func cloneCellRows(rows [][]Cell) [][]Cell {
	if len(rows) == 0 {
		return nil
	}
	out := make([][]Cell, len(rows))
	for i, row := range rows {
		out[i] = cloneCellSlice(row)
	}
	return out
}

func cloneCellRuns(runs []CellRun) []CellRun {
	if len(runs) == 0 {
		return nil
	}
	out := make([]CellRun, len(runs))
	copy(out, runs)
	return out
}

func cloneScrollbackRowAppends(rows []ScrollbackRowAppend) []ScrollbackRowAppend {
	if len(rows) == 0 {
		return nil
	}
	out := make([]ScrollbackRowAppend, len(rows))
	for i, row := range rows {
		out[i] = row
		out[i].Cells = cloneCellSlice(row.Cells)
		out[i].Runs = cloneCellRuns(row.Runs)
	}
	return out
}

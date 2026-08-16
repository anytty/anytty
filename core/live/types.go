package live

import (
	"os"
	"strings"

	vterm "github.com/anytty/anytty/vterm/vterm"
)

const preserveAltScreenOnExitEnv = "ANYTTY_PRESERVE_ALT_SCREEN_ON_EXIT"

// SurfaceSize describes the current host projection size.
type SurfaceSize struct {
	Cols int
	Rows int
}

// Valid reports whether the size can be used for a live surface projection.
func (s SurfaceSize) Valid() bool {
	return s.Cols > 0 && s.Rows > 0
}

// SurfaceTrack 是 core-v2 native live screen 的 owner。
// truth source 是同一个 terminal 的 PTY 输出经过 VTerm 解释后的当前屏；R358 后它
// 不再拥有 authoritative history transaction，也不能被 history/window/copy 当作
// logical-line truth 来源。
type SurfaceTrack struct {
	size                         SurfaceSize
	vt                           *vterm.VTerm
	onResponse                   vterm.ResponseHandler
	pending                      string
	preserveAltScreenFrameOnExit bool
}

// SurfaceTrackOptions 定义 live surface 的本地渲染策略和 PTY response 边界。
// OnResponse 只用于把 OSC/DA/DSR 等终端查询响应回写 PTY；不能借这个回调写 history。
type SurfaceTrackOptions struct {
	PreserveAltScreenFrameOnExit bool
	OnResponse                   vterm.ResponseHandler
}

// SurfaceWriteResult 描述一次 live 写入在 live 层需要保留的轻量分段。
// 它不是 history semantic evidence；当前只允许承载 raw live segment 和 alt-exit
// frame capture，禁止重新加入 transaction/damage/frame rows。
type SurfaceWriteResult struct {
	Segments    []SurfaceWriteSegment
	RowCopies   []SurfaceRowCopy
	ChangedRows []int
	FullReplace bool
	rowSources  []int
}

// SurfaceRowCopy maps exact final rows back to the screen before WriteWithResult.
// Consumers must read all sources before applying any destination writes.
type SurfaceRowCopy struct {
	SourceRow      int
	DestinationRow int
	Count          int
}

// SurfaceWriteSegment 是 live surface 的本地分段结果。
// Raw 只用于保持 alt-exit capture 周围的顺序；AltScreenExitFrame 是 live/copy 进入态
// 可用的最后 alt frame，不进入 primary authoritative history truth。
type SurfaceWriteSegment struct {
	Raw                string
	AltScreenExitFrame [][]vterm.Cell
}

// SurfaceSnapshot 是真实 live terminal 的 size-bound cell matrix，不是 history truth。
type SurfaceSnapshot struct {
	Size   SurfaceSize
	Screen vterm.ScreenData
	Cursor vterm.CursorState
	Modes  vterm.TerminalModes
}

// DefaultSurfaceTrackOptions 返回 live surface 的默认策略。
// 默认不把 alt-screen final frame replay 到 primary live screen；调用方可通过 env 或
// options 显式开启，用于保留某些 transient alt 画面的可见尾屏。
func DefaultSurfaceTrackOptions() SurfaceTrackOptions {
	return SurfaceTrackOptions{
		PreserveAltScreenFrameOnExit: boolEnvDefault(preserveAltScreenOnExitEnv, false),
	}
}

// NewSurfaceTrack 创建只维护 native live screen 的 SurfaceTrack。
// 调用边界是 core terminal owner；history owner 不应从返回对象读取 snapshot 反推历史。
func NewSurfaceTrack(size SurfaceSize) *SurfaceTrack {
	return NewSurfaceTrackWithOptions(size, DefaultSurfaceTrackOptions())
}

// NewSurfaceTrackWithOptions 创建带 response/preserve 策略的 live screen track。
// 无效 size 会回退到 80x24，保证 core 在 PTY 初始尺寸缺失时仍有可写 native screen。
func NewSurfaceTrackWithOptions(size SurfaceSize, options SurfaceTrackOptions) *SurfaceTrack {
	if !size.Valid() {
		size = SurfaceSize{Cols: 80, Rows: 24}
	}
	return &SurfaceTrack{
		size:                         size,
		vt:                           vterm.New(size.Cols, size.Rows, 0, options.OnResponse),
		onResponse:                   options.OnResponse,
		preserveAltScreenFrameOnExit: options.PreserveAltScreenFrameOnExit,
	}
}

// Size 返回 live surface 当前尺寸。它只描述 native screen 投影尺寸，不是 history
// window 的 wrap 宽度或 logical line 生成宽度。
func (surface *SurfaceTrack) Size() SurfaceSize {
	return surface.size
}

// Resize 调整 native live screen 的 VTerm 尺寸。
// R358 后该方法不返回 history transaction；Terminal 会通过独立 history semantic
// source 发送 resize boundary，避免 live surface 重新承载 history proof。
func (surface *SurfaceTrack) Resize(size SurfaceSize) {
	if !size.Valid() {
		return
	}
	surface.size = size
	surface.ensureVTerm()
	surface.vt.ResizeWithDamage(size.Cols, size.Rows)
}

// ResetForRestartPreservingScreen 在进程重启时保留当前可见 tail 并重建 VTerm。
// terminal identity 未变，因此 live 可见尾屏保留；外部进程已换新，因此旧程序的
// alt-screen、mouse、pending escape 和 mode 状态必须丢弃。
func (surface *SurfaceTrack) ResetForRestartPreservingScreen() {
	surface.ensureVTerm()
	snapshot := surface.Snapshot()
	rows := cloneVTermCellRows(snapshot.Screen.Cells)
	size := surface.size
	rows, cursor := restartPreservedScreenRows(rows, size.Rows)
	// 中文说明：重启的是外部进程，不是 terminal identity。保留可见 tail，
	// 但用全新 VTerm 丢弃旧程序的 mouse/bracketed paste/alt-screen/pending escape 状态。
	_ = surface.vt.Close()
	surface.vt = vterm.New(size.Cols, size.Rows, 0, surface.onResponse)
	surface.pending = ""
	if len(rows) == 0 {
		return
	}
	surface.vt.LoadSizedSnapshotWithExtendedMetadata(
		size.Cols,
		size.Rows,
		nil,
		nil,
		nil,
		nil,
		vterm.ScreenData{Cells: rows},
		nil,
		nil,
		nil,
		cursor,
		vterm.TerminalModes{AutoWrap: true},
	)
}

// Write 把 PTY text 写入 native live screen。
// 它是实时显示热路径，只维护 latest screen；需要 history 的调用方必须走 core
// Terminal 的 history semantic worker，而不是从这里取 transaction。
func (surface *SurfaceTrack) Write(text string) {
	if text != "" && surface.pending == "" && !strings.Contains(text, "\x1b[?") {
		surface.ensureVTerm()
		_, _, _ = surface.vt.WriteForLatestFrame([]byte(text))
		return
	}
	_ = surface.WriteWithResult(text)
}

// WriteWithResult 写入 native live screen 并返回 live 层轻量分段。
// 失败条件是输入为空且没有 pending CSI 时返回空结果；不构造 history semantic
// transaction、damage 或 screen-frame clone，避免 live 热路径变成历史证据链。
func (surface *SurfaceTrack) WriteWithResult(text string) SurfaceWriteResult {
	var result SurfaceWriteResult
	var raw strings.Builder
	if text == "" && surface.pending == "" {
		return result
	}
	surface.ensureVTerm()
	text = surface.pending + text
	surface.pending = ""
	for text != "" {
		idx := strings.Index(text, "\x1b[?")
		if idx < 0 {
			mergeSurfaceWriteDamage(&result, surface.writeRaw(text))
			raw.WriteString(text)
			appendSurfaceWriteRawSegment(&result, &raw)
			return result
		}
		if idx > 0 {
			mergeSurfaceWriteDamage(&result, surface.writeRaw(text[:idx]))
			raw.WriteString(text[:idx])
			text = text[idx:]
			continue
		}
		consumed, action, _, complete := consumePrivateModeCSI(text)
		if !complete {
			surface.pending = text
			appendSurfaceWriteRawSegment(&result, &raw)
			return result
		}
		if consumed <= 0 {
			mergeSurfaceWriteDamage(&result, surface.writeRaw(text[:1]))
			raw.WriteString(text[:1])
			text = text[1:]
			continue
		}
		if action == privateModeAltExit && surface.vt.IsAltScreen() {
			altFrame := surface.altScreenFrameCells()
			mergeSurfaceWriteDamage(&result, surface.writeRaw(text[:consumed]))
			raw.WriteString(text[:consumed])
			if surface.preserveAltScreenFrameOnExit {
				surface.appendAltScreenFrameCells(altFrame)
			}
			if len(altFrame) > 0 {
				result.Segments = append(result.Segments, SurfaceWriteSegment{
					Raw:                raw.String(),
					AltScreenExitFrame: cloneVTermCellRows(altFrame),
				})
				raw.Reset()
			}
			text = text[consumed:]
			continue
		}
		mergeSurfaceWriteDamage(&result, surface.writeRaw(text[:consumed]))
		raw.WriteString(text[:consumed])
		text = text[consumed:]
	}
	if raw.Len() > 0 {
		appendSurfaceWriteRawSegment(&result, &raw)
	}
	return result
}

func appendSurfaceWriteRawSegment(result *SurfaceWriteResult, raw *strings.Builder) {
	if result == nil || raw == nil || raw.Len() == 0 {
		return
	}
	result.Segments = append(result.Segments, SurfaceWriteSegment{Raw: raw.String()})
	raw.Reset()
}

func (surface *SurfaceTrack) writeRaw(text string) vterm.WriteDamage {
	if text == "" {
		return vterm.WriteDamage{}
	}
	// 这里只保留 touched-row 索引，不构造 history transaction、cell damage 或帧队列。
	_, _, damage := surface.vt.WriteForLatestFrame([]byte(text))
	return damage
}

func mergeSurfaceWriteDamage(result *SurfaceWriteResult, damage vterm.WriteDamage) {
	if result == nil || result.FullReplace {
		return
	}
	if !damage.IncrementalRowsReliable {
		result.FullReplace = true
		result.RowCopies = nil
		result.ChangedRows = nil
		result.rowSources = nil
		return
	}
	if len(result.rowSources) == 0 {
		result.rowSources = make([]int, damage.SizeRows)
		for row := range result.rowSources {
			result.rowSources[row] = row
		}
	}
	if len(result.rowSources) != damage.SizeRows {
		result.FullReplace = true
		result.RowCopies = nil
		result.ChangedRows = nil
		result.rowSources = nil
		return
	}
	previousSources := append([]int(nil), result.rowSources...)
	copyDestinations := make([]bool, len(previousSources))
	for _, rowCopy := range damage.RowCopies {
		for offset := 0; offset < rowCopy.Count; offset++ {
			sourceRow := rowCopy.SourceRow + offset
			destinationRow := rowCopy.DestinationRow + offset
			if sourceRow < 0 || sourceRow >= len(previousSources) || destinationRow < 0 || destinationRow >= len(previousSources) {
				result.FullReplace = true
				result.RowCopies = nil
				result.ChangedRows = nil
				result.rowSources = nil
				return
			}
			result.rowSources[destinationRow] = previousSources[sourceRow]
			copyDestinations[destinationRow] = true
		}
	}
	for _, row := range damage.DirectDamageTouchedRows {
		if row < 0 || row >= len(result.rowSources) || copyDestinations[row] {
			continue
		}
		result.rowSources[row] = -1
	}
	result.RowCopies = result.RowCopies[:0]
	result.ChangedRows = result.ChangedRows[:0]
	for destinationRow, sourceRow := range result.rowSources {
		if sourceRow < 0 {
			result.ChangedRows = append(result.ChangedRows, destinationRow)
			continue
		}
		if sourceRow == destinationRow {
			continue
		}
		if len(result.RowCopies) > 0 {
			last := &result.RowCopies[len(result.RowCopies)-1]
			if last.SourceRow+last.Count == sourceRow && last.DestinationRow+last.Count == destinationRow {
				last.Count++
				continue
			}
		}
		result.RowCopies = append(result.RowCopies, SurfaceRowCopy{SourceRow: sourceRow, DestinationRow: destinationRow, Count: 1})
	}
}

func (surface *SurfaceTrack) altScreenFrameCells() [][]vterm.Cell {
	snapshot := surface.Snapshot()
	rows := make([][]vterm.Cell, 0, len(snapshot.Screen.Cells))
	for _, row := range snapshot.Screen.Cells {
		cloned := make([]vterm.Cell, len(row))
		copy(cloned, row)
		rows = append(rows, trimTrailingDefaultBlankCells(cloned))
	}
	for len(rows) > 0 && !rowHasVisibleFootprint(rows[0]) {
		rows = rows[1:]
	}
	for len(rows) > 0 && !rowHasVisibleFootprint(rows[len(rows)-1]) {
		rows = rows[:len(rows)-1]
	}
	return rows
}

func (surface *SurfaceTrack) appendAltScreenFrameCells(rows [][]vterm.Cell) {
	if len(rows) == 0 {
		return
	}
	var builder strings.Builder
	builder.WriteString("\r\n")
	// 中文说明：alt-screen 退出时只把最后一帧追加到 live surface，
	// 这里用 cell replay 保留 SGR、带背景空白和列布局，但不回写 history parser。
	builder.Write(vterm.EncodeHistoryRowsReplay(rows))
	surface.writeRaw(builder.String())
}

// Rows 返回当前 native screen 的纯文本行，主要用于测试和兼容诊断。
// 它不是 authoritative history window，不能用于 copy/history/search truth。
func (surface *SurfaceTrack) Rows() []string {
	snapshot := surface.Snapshot()
	if len(snapshot.Screen.Cells) == 0 {
		return nil
	}
	out := make([]string, len(snapshot.Screen.Cells))
	for rowIndex, row := range snapshot.Screen.Cells {
		out[rowIndex] = strings.TrimRight(vtermRowText(row), " ")
	}
	return trimTrailingEmptyRows(out)
}

// Snapshot 返回当前 native live screen 的 size-bound cell matrix。
// 调用方只能用于实时显示或进入态上下文；history truth 必须继续走 core history store。
func (surface *SurfaceTrack) Snapshot() SurfaceSnapshot {
	surface.ensureVTerm()
	return SurfaceSnapshot{
		Size: surface.size,
		// 中文说明：live snapshot 是协议/渲染高频路径，保留行数和 styled footprint，
		// 但不克隆每行尾部的纯默认空白，避免压力输出反复搬运整屏空白。
		Screen: surface.vt.TrimmedScreenContent(),
		Cursor: surface.vt.CursorState(),
		Modes:  surface.vt.Modes(),
	}
}

// VisitTrimmedScreenRows 以零拷贝访问当前 trimmed native screen rows。
// visit 回调只在调用期间有效，调用方不能保存 cellAt 闭包或把这些 rows 当历史来源。
func (surface *SurfaceTrack) VisitTrimmedScreenRows(visit func(rowIndex int, cellCount int, cellAt func(int) vterm.Cell)) vterm.TrimmedScreenRowsInfo {
	surface.ensureVTerm()
	return surface.vt.VisitTrimmedScreenRows(visit)
}

// VisualRowHashes returns the compact current-screen baseline used by a
// protocol session to compare the next client-confirmed frame.
func (surface *SurfaceTrack) VisualRowHashes() []uint64 {
	surface.ensureVTerm()
	return surface.vt.ScreenVisualHashes()
}

func (surface *SurfaceTrack) IsAlternateScreen() bool {
	surface.ensureVTerm()
	return surface.vt.IsAltScreen()
}

func (surface *SurfaceTrack) ensureVTerm() {
	if surface.vt != nil {
		return
	}
	if !surface.size.Valid() {
		surface.size = SurfaceSize{Cols: 80, Rows: 24}
	}
	surface.vt = vterm.New(surface.size.Cols, surface.size.Rows, 0, surface.onResponse)
}

type privateModeAltAction int

const (
	privateModeNoAlt privateModeAltAction = iota
	privateModeAltEnter
	privateModeAltExit
)

func consumePrivateModeCSI(input string) (int, privateModeAltAction, string, bool) {
	if !strings.HasPrefix(input, "\x1b[?") {
		return 0, privateModeNoAlt, "", true
	}
	end := -1
	for i := 3; i < len(input); i++ {
		b := input[i]
		if b >= 0x40 && b <= 0x7e {
			end = i
			break
		}
	}
	if end < 0 {
		return 0, privateModeNoAlt, "", false
	}
	final := input[end]
	sequence := input[:end+1]
	if final != 'h' && final != 'l' {
		return end + 1, privateModeNoAlt, sequence, true
	}
	params := strings.FieldsFunc(input[3:end], func(r rune) bool {
		return r == ';' || r == ':'
	})
	hasAlt := false
	kept := make([]string, 0, len(params))
	for _, param := range params {
		if param == "" {
			continue
		}
		if isAltScreenPrivateMode(param) {
			hasAlt = true
			continue
		}
		kept = append(kept, param)
	}
	if !hasAlt {
		return end + 1, privateModeNoAlt, sequence, true
	}
	if final == 'h' {
		return end + 1, privateModeAltEnter, sequence, true
	}
	filtered := ""
	if len(kept) > 0 {
		filtered = "\x1b[?" + strings.Join(kept, ";") + string(final)
	}
	return end + 1, privateModeAltExit, filtered, true
}

func isAltScreenPrivateMode(param string) bool {
	switch param {
	case "47", "1047", "1049":
		return true
	default:
		return false
	}
}

func vtermRowText(row []vterm.Cell) string {
	var out strings.Builder
	for _, cell := range row {
		out.WriteString(cell.Content)
	}
	return out.String()
}

func trimTrailingEmptyRows(rows []string) []string {
	last := len(rows) - 1
	for last >= 0 && rows[last] == "" {
		last--
	}
	if last < 0 {
		return []string{""}
	}
	out := make([]string, last+1)
	copy(out, rows[:last+1])
	return out
}

func rowHasVisibleFootprint(row []vterm.Cell) bool {
	for _, cell := range row {
		if cellHasVisibleFootprint(cell) {
			return true
		}
	}
	return false
}

func cellHasVisibleFootprint(cell vterm.Cell) bool {
	if cell.Content != "" && strings.Trim(cell.Content, " ") != "" {
		return true
	}
	if cell.Style != (vterm.CellStyle{}) || cell.LinkURL != "" || cell.LinkParams != "" {
		return true
	}
	return cell.Width > 1
}

func trimTrailingDefaultBlankCells(row []vterm.Cell) []vterm.Cell {
	last := len(row) - 1
	for last >= 0 && !cellHasVisibleFootprint(row[last]) {
		last--
	}
	return row[:last+1]
}

func cloneVTermCellRows(rows [][]vterm.Cell) [][]vterm.Cell {
	if len(rows) == 0 {
		return nil
	}
	cloned := make([][]vterm.Cell, len(rows))
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		cloned[i] = make([]vterm.Cell, len(row))
		copy(cloned[i], row)
	}
	return cloned
}

func restartPreservedScreenRows(rows [][]vterm.Cell, maxRows int) ([][]vterm.Cell, vterm.CursorState) {
	last := -1
	for i, row := range rows {
		if rowHasVisibleFootprint(row) {
			last = i
		}
	}
	if last < 0 {
		return nil, vterm.CursorState{Visible: false}
	}
	rows = rows[:last+1]
	if maxRows > 0 && len(rows) >= maxRows {
		rows = rows[len(rows)-maxRows+1:]
	}
	// 中文说明：保留旧 tail 后，新进程从下一空行继续写；这里的 cursor 是真实
	// surface 坐标种子，不能隐藏，否则新 shell 不显式 show cursor 时会一直不可见。
	return rows, vterm.CursorState{Row: len(rows), Col: 0, Visible: true}
}

func boolEnvDefault(name string, fallback bool) bool {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

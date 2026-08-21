package vterm

import (
	"bytes"
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/anytty/anytty/shared/gridtrace"
	charmvt "github.com/anytty/anytty/vterm/internal/vt"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/text/unicode/norm"
)

var (
	basicColorStrings   = buildIndexedColorStrings("ansi:", 16)
	indexedColorStrings = buildIndexedColorStrings("idx:", 256)
)

var safeEmulatorWrite = func(emu *charmvt.SafeEmulator, data []byte) (int, error) {
	return emu.Write(data)
}

var safeEmulatorWriteWithDamage = func(emu *charmvt.SafeEmulator, data []byte) (int, error, []charmvt.Damage, bool) {
	type damageWriter interface {
		WriteWithDamage([]byte) (int, error, []charmvt.Damage)
	}
	if writer, ok := any(emu).(damageWriter); ok {
		n, err, damages := writer.WriteWithDamage(data)
		return n, err, damages, true
	}
	n, err := emu.Write(data)
	return n, err, nil, false
}

var safeEmulatorWriteWithSemanticDamage = func(emu *charmvt.SafeEmulator, data []byte) (int, error, []charmvt.Damage, bool) {
	type semanticDamageWriter interface {
		WriteWithSemanticDamage([]byte) (int, error, []charmvt.Damage)
	}
	if writer, ok := any(emu).(semanticDamageWriter); ok {
		n, err, damages := writer.WriteWithSemanticDamage(data)
		return n, err, damages, true
	}
	n, err := emu.Write(data)
	return n, err, nil, false
}

var safeEmulatorWriteForLineHistoryDamage = func(emu *charmvt.SafeEmulator, data []byte) (int, error, []charmvt.Damage, bool) {
	type lineHistoryDamageWriter interface {
		WriteForLineHistoryDamage([]byte) (int, error, []charmvt.Damage)
	}
	if writer, ok := any(emu).(lineHistoryDamageWriter); ok {
		n, err, damages := writer.WriteForLineHistoryDamage(data)
		return n, err, damages, true
	}
	n, err := emu.Write(data)
	return n, err, nil, false
}

type Cell struct {
	Content    string
	Width      int
	Style      CellStyle
	LinkURL    string
	LinkParams string
}

type CellStyle struct {
	FG            string
	BG            string
	Bold          bool
	Italic        bool
	Underline     bool
	Blink         bool
	Reverse       bool
	Strikethrough bool
	LinkURL       string
	LinkParams    string
}

type CursorShape string

const (
	CursorBlock     CursorShape = "block"
	CursorUnderline CursorShape = "underline"
	CursorBar       CursorShape = "bar"
)

type CursorState struct {
	Row     int
	Col     int
	Visible bool
	Shape   CursorShape
	Blink   bool
}

// TerminalModes 是 vterm 持有的终端模式状态。
// domain owner：vterm 解码 PTY private/public mode；上层只能读取该状态做语义判定，
// 不能用 raw bytes 或程序名重建 mode truth。
type TerminalModes struct {
	AlternateScreen   bool
	AlternateScroll   bool
	MouseTracking     bool
	MouseX10          bool
	MouseNormal       bool
	MouseButtonEvent  bool
	MouseAnyEvent     bool
	MouseSGR          bool
	BracketedPaste    bool
	ApplicationCursor bool
	AutoWrap          bool
	// SynchronizedOutput 对应 DEC private mode 2026；它让 core 能识别
	// begin/end 分片之间的 payload 仍属于同一个 primary screen session。
	SynchronizedOutput bool
}

type ScreenData struct {
	Cells             [][]Cell
	IsAlternateScreen bool
}

type TrimmedScreenRowsInfo struct {
	Cols              int
	Rows              int
	IsAlternateScreen bool
	Cursor            CursorState
	Modes             TerminalModes
}

type SurfaceSnapshot struct {
	Cols                 int
	Rows                 int
	Scrollback           [][]Cell
	ScrollbackTimestamps []time.Time
	ScrollbackRowKinds   []string
	ScrollbackWrapped    []bool
	ScrollbackOwnership  []string
	Screen               ScreenData
	ScreenTimestamps     []time.Time
	ScreenRowKinds       []string
	ScreenWrapped        []bool
	ScreenOwnership      []string
	Cursor               CursorState
	Modes                TerminalModes
}

// ResponseHandler is called when the emulator produces a response (e.g. DSR
// cursor position report). The data should be written to the PTY's stdin so
// the child process receives it.
type ResponseHandler func(data []byte)

// TitleHandler is called when the terminal title changes (OSC 2).
type TitleHandler func(title string)

// WorkingDirectoryHandler is called when the terminal reports its current
// working directory (OSC 7).
type WorkingDirectoryHandler func(path string)

type VTerm struct {
	emu *charmvt.SafeEmulator

	mu        sync.RWMutex
	cursor    CursorState
	modes     TerminalModes
	mouseMode mouseModeState
	resp      ResponseHandler
	onTitle   TitleHandler
	onCWD     WorkingDirectoryHandler
	sbSize    int
	defaultFG string
	defaultBG string
	palette   map[int]string

	scrollbackTimestamps     []time.Time
	screenTimestamps         []time.Time
	primarySavedTimestamps   []time.Time
	primarySavedOwnership    []string
	scrollbackRowKinds       []string
	screenRowKinds           []string
	scrollbackOwnership      []string
	screenOwnership          []string
	screenRowCache           [][]Cell
	scrollbackRowCache       [][]Cell
	screenFingerprintCache   []rowFingerprint
	screenTimestampsScratch  []time.Time
	screenRowKindsScratch    []string
	screenFingerprintScratch []rowFingerprint

	done chan struct{} // closed when drain goroutine exits
}

type mouseModeState struct {
	x10         bool
	normal      bool
	highlight   bool
	buttonEvent bool
	anyEvent    bool
	sgr         bool
}

type rowFingerprint struct {
	hash  uint64
	blank bool
}

type resizeReflowLine struct {
	cells       []Cell
	tailFill    *CellStyle
	wrapped     bool
	sourceRow   int
	timestamp   time.Time
	rowKind     string
	fingerprint rowFingerprint
}

type resizeReflowCell struct {
	cell      Cell
	sourceRow int
	timestamp time.Time
	rowKind   string
}

type rowCacheReconcilePlan struct {
	afterScreen               []rowFingerprint
	preservedFromBefore       int
	requiredScrollbackAppends int
	beforeScrollbackLen       int
	screenScrollShift         int
	scrollbackSourceRows      []int
}

type DamageRect struct {
	X      int
	Y      int
	Width  int
	Height int
}

// RowCopy describes final screen rows whose cell contents are exactly equal to
// rows from the screen before the write. All copies from one WriteDamage read
// from that same pre-write screen.
type RowCopy struct {
	SourceRow      int
	DestinationRow int
	Count          int
}

type ScreenOpCode uint8

const (
	ScreenOpWriteSpan ScreenOpCode = iota
	ScreenOpScrollRect
	ScreenOpCopyRect
	ScreenOpClearRect
	ScreenOpClearToEOL
	ScreenOpCursor
	ScreenOpModes
	ScreenOpResize
	ScreenOpTitle
	ScreenOpControl
)

type Size struct {
	Cols uint16
	Rows uint16
}

type DamageOp struct {
	Code    ScreenOpCode
	Control string
	Rect    DamageRect
	Src     DamageRect
	DstX    int
	DstY    int
	Dx      int
	Dy      int
	Row     int
	Col     int
	RowSet  bool
	Mode    int
	Bottom  int
	Private bool
	Enabled bool
	Cells   []Cell
	Runs    []CellRun
	// ScrollOut 携带该 op 语义位置上离开 primary 可见区的行，例如 ED2 清屏。
	// 它属于 vterm ordered semantic proof，core 不能从最终 frame 反推这些行。
	ScrollOut  []ScrollbackRowAppend
	Size       Size
	TailFill   *CellStyle
	Timestamp  time.Time
	RowKind    string
	Ownership  string
	Wrapped    bool
	WrappedSet bool
}

type ScrollbackRowAppend struct {
	Cells     []Cell
	Runs      []CellRun
	Timestamp time.Time
	RowKind   string
	// Row/RowSet 表示该 proof 来源于清屏或滚动前的 primary viewport 行。
	// core-v2 history 用它按 current frame ownership 过滤 ED2 clear-time proof，
	// 避免把已经 sealed 的 shell 行重复写入 authoritative history。
	Row        int
	RowSet     bool
	Wrapped    bool
	WrappedSet bool
	Ownership  string
}

// HistoryPersistedOwnership marks visible rows that core has already appended
// to authoritative history. They remain renderable, but must not be emitted as
// new history when they later leave the shared emulator's screen.
const HistoryPersistedOwnership = "history-persisted"

type ScreenUpdate struct {
	FullReplace      bool
	ResetScrollback  bool
	Size             Size
	ScreenScroll     int
	Title            string
	Screen           ScreenData
	ScreenTimestamps []time.Time
	ScreenRowKinds   []string
	ScreenWrapped    []bool
	Ops              []DamageOp
	ScrollbackTrim   int
	ScrollbackAppend []ScrollbackRowAppend
	Cursor           CursorState
	Modes            TerminalModes
}

func normalizeScreenUpdate(update ScreenUpdate) ScreenUpdate {
	normalized := update
	if normalized.ScrollbackTrim < 0 {
		normalized.ScrollbackTrim = 0
	}
	if normalized.FullReplace {
		normalized.ScreenTimestamps = normalizeTimeSlice(normalized.ScreenTimestamps, len(normalized.Screen.Cells))
		normalized.ScreenRowKinds = normalizeStringSlice(normalized.ScreenRowKinds, len(normalized.Screen.Cells))
		normalized.ScreenWrapped = normalizeBoolSlice(normalized.ScreenWrapped, len(normalized.Screen.Cells))
	} else {
		normalized.Screen.IsAlternateScreen = normalized.Modes.AlternateScreen
	}
	normalized.Ops = normalizeScreenOps(normalized.Ops)
	normalized.ScrollbackAppend = normalizeScrollbackAppendWrapped(normalized.ScrollbackAppend)
	return normalized
}

func normalizeScreenOps(ops []DamageOp) []DamageOp {
	if len(ops) == 0 {
		return nil
	}
	normalized := make([]DamageOp, 0, len(ops))
	for _, op := range ops {
		if !isValidScreenOpCode(op.Code) {
			continue
		}
		normalized = append(normalized, normalizeScreenOp(op))
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeScreenOp(op DamageOp) DamageOp {
	op.Wrapped = wrappedSet(op.WrappedSet, op.Wrapped)
	switch op.Code {
	case ScreenOpWriteSpan:
		op.Rect = DamageRect{}
		op.Src = DamageRect{}
		op.DstX = 0
		op.DstY = 0
		op.Dx = 0
		op.Dy = 0
	case ScreenOpScrollRect:
		op.Row = 0
		op.Col = 0
		op.Cells = nil
		op.Runs = nil
		op.ScrollOut = nil
		op.Wrapped = false
		op.WrappedSet = false
		op.Size = Size{}
		op.Src = DamageRect{}
		op.DstX = 0
		op.DstY = 0
		op.Rect = normalizeDamageRect(op.Rect)
	case ScreenOpCopyRect:
		op.Row = 0
		op.Col = 0
		op.Cells = nil
		op.Runs = nil
		op.ScrollOut = nil
		op.Wrapped = false
		op.WrappedSet = false
		op.Size = Size{}
		op.Dx = 0
		op.Dy = 0
		op.Src = normalizeDamageRect(op.Src)
	case ScreenOpClearRect:
		op.Row = 0
		op.Col = 0
		op.Cells = nil
		op.Runs = nil
		op.ScrollOut = nil
		op.Size = Size{}
		op.Src = DamageRect{}
		op.DstX = 0
		op.DstY = 0
		op.Dx = 0
		op.Dy = 0
		op.Rect = normalizeDamageRect(op.Rect)
	case ScreenOpClearToEOL:
		op.Rect = DamageRect{}
		op.Src = DamageRect{}
		op.DstX = 0
		op.DstY = 0
		op.Dx = 0
		op.Dy = 0
		op.Cells = nil
		op.Runs = nil
		op.ScrollOut = nil
	case ScreenOpResize:
		op.Rect = DamageRect{}
		op.Src = DamageRect{}
		op.DstX = 0
		op.DstY = 0
		op.Dx = 0
		op.Dy = 0
		op.Row = 0
		op.Col = 0
		op.Cells = nil
		op.Runs = nil
		op.ScrollOut = nil
		op.Wrapped = false
		op.WrappedSet = false
	case ScreenOpCursor, ScreenOpTitle:
		return DamageOp{Code: op.Code}
	case ScreenOpModes:
		return DamageOp{Code: op.Code, Mode: op.Mode, Private: op.Private, Enabled: op.Enabled}
	case ScreenOpControl:
		return DamageOp{Code: op.Code, Control: op.Control, Row: op.Row, Col: op.Col, RowSet: op.RowSet, Mode: op.Mode, Bottom: op.Bottom, Cells: op.Cells, ScrollOut: cloneScrollbackRowAppends(op.ScrollOut), TailFill: cloneCellStylePointer(op.TailFill)}
	}
	return op
}

func normalizeDamageRect(rect DamageRect) DamageRect {
	if rect.Width < 0 {
		rect.Width = 0
	}
	if rect.Height < 0 {
		rect.Height = 0
	}
	return rect
}

func normalizeScrollbackAppendWrapped(rows []ScrollbackRowAppend) []ScrollbackRowAppend {
	if len(rows) == 0 {
		return nil
	}
	out := make([]ScrollbackRowAppend, len(rows))
	for i, row := range rows {
		row.Wrapped = wrappedSet(row.WrappedSet, row.Wrapped)
		out[i] = row
	}
	return out
}

func wrappedSet(set bool, value bool) bool {
	return set && value
}

func isValidScreenOpCode(code ScreenOpCode) bool {
	switch code {
	case ScreenOpWriteSpan,
		ScreenOpScrollRect,
		ScreenOpCopyRect,
		ScreenOpClearRect,
		ScreenOpClearToEOL,
		ScreenOpCursor,
		ScreenOpModes,
		ScreenOpResize,
		ScreenOpTitle,
		ScreenOpControl:
		return true
	default:
		return false
	}
}

type CellRun struct {
	Style CellStyle
	Text  string
}

type WriteDamage struct {
	Ops              []DamageOp
	SemanticOps      []DamageOp
	ScrollbackAppend []DamageOp
	// EvictedAppend 与 ScrollbackAppend 来自同一份 eviction 捕获，但不做空行过滤：
	// 旧 proof 消费路径依赖 ScrollbackAppend 丢弃空行的行为，而 logical-line
	// 无限历史必须保留空行的原始顺序。为空时消费方回退读 ScrollbackAppend
	// （ring-diff/resize 路径本身保留空行）。
	EvictedAppend       []DamageOp
	LiveTailAppendRows  int
	ResizeLiveTailRows  int
	AlternateAppend     []DamageOp
	ScrollbackTrim      int
	ScreenScroll        int
	RequiresFullReplace bool
	FullReplaceReason   string
	DirectDamageItems   int
	DirectDamageRows    int
	DirectDamageCells   int
	// DirectDamageTouchedRows 是本次 direct damage 在 primary/alt screen 上触达的
	// row index。domain owner 是 vterm damage 解码；core 只能把它当作
	// current-frame ownership proof，不能用 final screen snapshot 反推历史。
	DirectDamageTouchedRows []int
	// IncrementalRowsReliable 表示 DirectDamageTouchedRows 完整覆盖本次当前屏变化。
	// latest-screen consumer 可据此只投影这些行；尺寸或 alternate screen 边界会回退整屏。
	IncrementalRowsReliable bool
	// RowCopies 只包含经过 before/after 行指纹校验的整行来源。它描述 cell matrix
	// 的 canonical 变化，不代表 renderer 必须执行滚屏动画。
	RowCopies    []RowCopy
	Cursor       CursorState
	Modes        TerminalModes
	SizeCols     int
	SizeRows     int
	DiffCPUNanos int64
}

type TraceHooks struct {
	Measure func(name string) func(bytes int)
	Count   func(name string, bytes int)
}

var traceHooks TraceHooks

func SetTraceHooks(hooks TraceHooks) {
	traceHooks = hooks
}

func traceMeasure(name string) func(bytes int) {
	if traceHooks.Measure == nil {
		return func(int) {}
	}
	return traceHooks.Measure(name)
}

func traceCount(name string, bytes int) {
	if traceHooks.Count != nil {
		traceHooks.Count(name, bytes)
	}
}

const modeAlternateScroll ansi.DECMode = 1007

const (
	rowFingerprintOffset64 = 14695981039346656037
	rowFingerprintPrime64  = 1099511628211

	broadDirectDamageRowRatioNumerator       = 3
	broadDirectDamageRowRatioDenominator     = 4
	broadDirectDamageCellRatioNumerator      = 3
	broadDirectDamageCellRatioDenominator    = 5
	broadDirectDamageMinRows                 = 8
	broadDirectDamageMinCells                = 512
	repeatedDirectDamageItemRatioNumerator   = 3
	repeatedDirectDamageItemRatioDenominator = 2
	repeatedDirectDamageMinItems             = 512
	repeatedDirectDamageMinCells             = 4096
)

func New(cols, rows int, scrollbackSize int, onResponse ResponseHandler) *VTerm {
	v := &VTerm{
		cursor: CursorState{
			Visible: true,
			Shape:   CursorBlock,
		},
		modes:  TerminalModes{AutoWrap: true},
		resp:   onResponse,
		sbSize: scrollbackSize,
		done:   make(chan struct{}),
	}
	v.resetEmulator(cols, rows)
	return v
}

func (v *VTerm) DisableEmulatorScrollback() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.disableEmulatorScrollbackLocked()
}

func (v *VTerm) resetEmulator(cols, rows int) {
	emu := charmvt.NewSafeEmulator(cols, rows)
	if v.sbSize <= 0 {
		emu.DisableScrollback()
	} else {
		emu.SetScrollbackSize(v.sbSize)
	}
	v.applyDefaultColorsToEmulator(emu)
	v.emu = emu
	v.scrollbackTimestamps = nil
	v.screenTimestamps = make([]time.Time, maxInt(rows, 1))
	v.scrollbackRowKinds = nil
	v.screenRowKinds = make([]string, maxInt(rows, 1))
	v.scrollbackOwnership = nil
	v.screenOwnership = make([]string, maxInt(rows, 1))
	v.invalidateRowCachesLocked()
	v.invalidateFingerprintCacheLocked()
	emu.SetCallbacks(charmvt.Callbacks{
		AltScreen: func(on bool) {
			// Called from within Write(), which already holds v.mu.Lock()
			v.modes.AlternateScreen = on
		},
		CursorVisibility: func(visible bool) {
			v.cursor.Visible = visible
		},
		CursorStyle: func(style charmvt.CursorStyle, blink bool) {
			switch style {
			case charmvt.CursorUnderline:
				v.cursor.Shape = CursorUnderline
			case charmvt.CursorBar:
				v.cursor.Shape = CursorBar
			default:
				v.cursor.Shape = CursorBlock
			}
			v.cursor.Blink = blink
		},
		EnableMode: func(mode ansi.Mode) {
			v.setMode(mode, true)
		},
		DisableMode: func(mode ansi.Mode) {
			v.setMode(mode, false)
		},
		Title: func(title string) {
			if v.onTitle != nil {
				v.onTitle(title)
			}
		},
		WorkingDirectory: func(path string) {
			if v.onCWD != nil {
				v.onCWD(path)
			}
		},
	})

	// Drain the emulator's response pipe. Without this, programs that send
	// DSR (Device Status Report, e.g. vi/vim) will deadlock because the
	// emulator writes the response to an io.Pipe and nobody reads it,
	// blocking the Write() call that holds the lock.
	done := make(chan struct{})
	v.done = done
	go func(emu *charmvt.SafeEmulator) {
		defer close(done)
		v.drainResponses(emu, v.resp)
	}(emu)
}

// drainResponses reads from the emulator's response pipe and forwards data
// to the handler. Exits when the emulator is closed (Read returns error).
func (v *VTerm) drainResponses(emu *charmvt.SafeEmulator, handler ResponseHandler) {
	buf := make([]byte, 256)
	for {
		n, err := emu.Read(buf)
		if n > 0 && handler != nil {
			data := make([]byte, n)
			copy(data, buf[:n])
			handler(data)
		}
		if err != nil {
			return
		}
	}
}

func (v *VTerm) Write(data []byte) (n int, err error) {
	n, err, _ = v.write(data, false)
	return n, err
}

func (v *VTerm) WriteMirror(data []byte) (n int, err error) {
	finish := traceMeasure("vterm.write")
	defer func() {
		finish(len(data))
	}()
	v.mu.Lock()
	defer v.mu.Unlock()
	defer func() {
		if r := recover(); r != nil {
			n = 0
			err = fmt.Errorf("vterm write panic: %v", r)
		}
	}()
	beforeHeight := 0
	beforeAltScreen := false
	beforeScrollbackLen := 0
	if v.emu != nil {
		beforeHeight = v.emu.Height()
		beforeAltScreen = v.emu.IsAltScreen()
		beforeScrollbackLen = v.emu.ScrollbackLen()
	}
	normalized := normalizeRenderableUTF8(data)
	v.clearTouchedRowsLocked()
	emulatorFinish := traceMeasure("vterm.write.emulator")
	n, err = safeEmulatorWrite(v.emu, normalized)
	emulatorFinish(len(normalized))
	pos := v.emu.CursorPosition()
	v.cursor.Row = pos.Y
	v.cursor.Col = pos.X
	v.modes.AlternateScreen = v.emu.IsAltScreen()
	reconcileFinish := traceMeasure("vterm.write.reconcile")
	afterHeight := 0
	afterAltScreen := false
	afterScrollbackLen := 0
	if v.emu != nil {
		afterHeight = v.emu.Height()
		afterAltScreen = v.emu.IsAltScreen()
		afterScrollbackLen = v.emu.ScrollbackLen()
	}
	now := time.Now().UTC()
	dirtyRows, dirtyReliable := v.consumeTouchedRowsLocked()
	switch {
	case !dirtyReliable || beforeHeight != afterHeight || beforeAltScreen != afterAltScreen:
		v.screenTimestamps = normalizeTimeSlice(v.screenTimestamps, afterHeight)
		v.screenRowKinds = normalizeStringSlice(v.screenRowKinds, afterHeight)
		v.screenOwnership = normalizeStringSlice(v.screenOwnership, afterHeight)
		for row := 0; row < afterHeight; row++ {
			v.screenTimestamps[row] = now
			v.screenRowKinds[row] = ""
			v.screenOwnership[row] = ""
		}
	default:
		v.screenTimestamps = normalizeTimeSlice(v.screenTimestamps, afterHeight)
		v.screenRowKinds = normalizeStringSlice(v.screenRowKinds, afterHeight)
		v.screenOwnership = normalizeStringSlice(v.screenOwnership, afterHeight)
		for _, row := range dirtyRows {
			if row < 0 || row >= afterHeight {
				continue
			}
			v.screenTimestamps[row] = now
			v.screenRowKinds[row] = ""
			v.screenOwnership[row] = ""
		}
	}
	switch {
	case afterScrollbackLen <= 0:
		v.scrollbackTimestamps = nil
		v.scrollbackRowKinds = nil
		v.scrollbackOwnership = nil
	case afterScrollbackLen > beforeScrollbackLen:
		added := afterScrollbackLen - beforeScrollbackLen
		v.scrollbackTimestamps = append(v.scrollbackTimestamps, make([]time.Time, added)...)
		v.scrollbackRowKinds = append(v.scrollbackRowKinds, make([]string, added)...)
		v.scrollbackOwnership = append(v.scrollbackOwnership, make([]string, added)...)
		for i := afterScrollbackLen - added; i < afterScrollbackLen; i++ {
			v.scrollbackTimestamps[i] = now
			v.scrollbackRowKinds[i] = ""
			v.scrollbackOwnership[i] = ""
		}
	default:
		v.alignScrollbackMetadataLocked()
	}
	v.invalidateRowCachesLocked()
	v.invalidateFingerprintCacheLocked()
	reconcileFinish(0)
	return n, err
}

func (v *VTerm) WriteWithDamage(data []byte) (n int, err error, damage WriteDamage) {
	return v.write(data, true)
}

// WriteWithSemanticDamage 只收集 core-v2 history 需要的 ordered semantic
// evidence。domain owner 仍是 vterm；它更新同一个 emulator live screen，但不把
// screen diff cell payload 当作 history truth 交给上游。
func (v *VTerm) WriteWithSemanticDamage(data []byte) (n int, err error, damage WriteDamage) {
	return v.write(data, true, writeDamageSemanticOnly)
}

// WriteForLineHistory 只返回 linehist 当前需要的 eviction/boundary 语义。
// domain owner 仍是 vterm emulator；调用方只能消费 EvictedAppend 和 ED3/alt/sync
// boundary，不能把该结果当作完整 ordered terminal op stream。
func (v *VTerm) WriteForLineHistory(data []byte) (n int, err error, damage WriteDamage) {
	return v.write(data, true, writeDamageEvictionOnly)
}

// WriteForLatestFrame 只维护当前 live screen；调用方不消费增量 damage，
// 因此这里不能为压力输出构造 scrollback damage 临时对象。
func (v *VTerm) WriteForLatestFrame(data []byte) (n int, err error, damage WriteDamage) {
	return v.writeLatest(data)
}

type writeDamageMode int

const (
	writeDamageFull writeDamageMode = iota
	writeDamageSemanticOnly
	writeDamageEvictionOnly
)

func (v *VTerm) write(data []byte, collectDamage bool, mode ...writeDamageMode) (n int, err error, damage WriteDamage) {
	damageMode := writeDamageFull
	if len(mode) > 0 {
		damageMode = mode[0]
	}
	finish := traceMeasure("vterm.write")
	defer func() {
		finish(len(data))
	}()
	v.mu.Lock()
	defer v.mu.Unlock()
	v.ensureScreenFingerprintCacheLocked()
	beforeWidth := 0
	beforeHeight := 0
	beforeAltScreen := false
	if v.emu != nil {
		beforeWidth = v.emu.Width()
		beforeHeight = v.emu.Height()
		beforeAltScreen = v.emu.IsAltScreen()
	}
	snapshotFinish := traceMeasure("vterm.write.before_snapshot")
	beforeScreen := reuseRowFingerprintSlice(v.screenFingerprintScratch, v.screenFingerprintCache)
	v.screenFingerprintScratch = beforeScreen
	beforeScrollbackLen := v.scrollbackRowCountLocked()
	beforeScreenTimestamps := v.screenTimestamps
	beforeScreenRowKinds := v.screenRowKinds
	beforeScreenOwnership := v.screenOwnership
	beforePersistedRows := v.persistedScreenRowsLocked(beforeScreenOwnership)
	snapshotFinish(0)
	defer func() {
		if r := recover(); r != nil {
			n = 0
			err = fmt.Errorf("vterm write panic: %v", r)
			damage = WriteDamage{}
		}
	}()
	normalizeFinish := traceMeasure("vterm.write.normalize")
	normalized := normalizeRenderableUTF8(data)
	normalizeFinish(len(normalized))
	clearTouchedFinish := traceMeasure("vterm.write.clear_touched")
	v.clearTouchedRowsLocked()
	clearTouchedFinish(0)
	emulatorFinish := traceMeasure("vterm.write.emulator")
	var (
		directDamages   []charmvt.Damage
		hasDirectDamage bool
	)
	if collectDamage {
		if damageMode == writeDamageEvictionOnly {
			n, err, directDamages, hasDirectDamage = safeEmulatorWriteForLineHistoryDamage(v.emu, normalized)
		} else if damageMode == writeDamageSemanticOnly {
			n, err, directDamages, hasDirectDamage = safeEmulatorWriteWithSemanticDamage(v.emu, normalized)
		} else {
			n, err, directDamages, hasDirectDamage = safeEmulatorWriteWithDamage(v.emu, normalized)
		}
	} else {
		n, err = safeEmulatorWrite(v.emu, normalized)
	}
	emulatorFinish(len(normalized))
	pos := v.emu.CursorPosition()
	v.cursor.Row = pos.Y
	v.cursor.Col = pos.X
	v.modes.AlternateScreen = v.emu.IsAltScreen()
	diffStart := time.Now()
	reconcileFinish := traceMeasure("vterm.write.reconcile")
	afterWidth := 0
	afterHeight := 0
	afterAltScreen := false
	if v.emu != nil {
		afterWidth = v.emu.Width()
		afterHeight = v.emu.Height()
		afterAltScreen = v.emu.IsAltScreen()
	}
	v.capturePrimaryMetadataOnAltEnter(beforeAltScreen, afterAltScreen, beforeScreenTimestamps, beforeScreenOwnership)
	dirtyRows, dirtyReliable := v.consumeTouchedRowsLocked()
	now := time.Now().UTC()
	var afterScreen []rowFingerprint
	fingerprintFinish := traceMeasure("vterm.write.reconcile.fingerprint")
	switch {
	case !dirtyReliable,
		beforeWidth != afterWidth,
		beforeHeight != afterHeight,
		beforeAltScreen != afterAltScreen:
		afterScreen = v.screenRowFingerprintsLocked()
		v.screenFingerprintCache = afterScreen
	default:
		v.ensureScreenFingerprintCacheLocked()
		for _, row := range dirtyRows {
			if row < 0 || row >= len(v.screenFingerprintCache) {
				continue
			}
			v.screenFingerprintCache[row] = v.screenRowFingerprintLocked(row)
		}
		afterScreen = v.screenFingerprintCache
	}
	fingerprintFinish(0)
	metadataFinish := traceMeasure("vterm.write.reconcile.metadata")
	cachePlan := v.reconcileRowMetadataLocked(beforeScreen, beforeScreenTimestamps, beforeScreenRowKinds, beforeScreenOwnership, beforeScrollbackLen, afterScreen, now)
	v.restorePrimaryMetadataOnAltExit(beforeAltScreen, afterAltScreen, afterHeight)
	metadataFinish(0)
	rowCacheFinish := traceMeasure("vterm.write.reconcile.row_cache")
	v.reconcileRowCachesLocked(beforeScreen, cachePlan)
	rowCacheFinish(0)
	if collectDamage {
		if damageMode == writeDamageEvictionOnly {
			damage = v.writeDamageHeaderLocked(cachePlan)
			if hasDirectDamage {
				evictedOps, altEvictedOps := v.evictedAppendOpsFromCharmVTDamages(directDamages, beforeAltScreen, beforeScreenTimestamps, beforeScreenRowKinds, beforeScreenOwnership, beforePersistedRows)
				damage.SemanticOps = v.semanticBoundaryOpsFromCharmVTDamagesLocked(directDamages)
				damage.EvictedAppend = evictedOps
				if len(altEvictedOps) > 0 {
					damage.AlternateAppend = altEvictedOps
				}
			}
			// The line-history path is also the production live write path when a
			// terminal owns a single emulator. Preserve the same incremental row
			// contract as WriteForLatestFrame without collecting ordinary text ops.
			damage.RequiresFullReplace = true
			damage.IncrementalRowsReliable = dirtyReliable &&
				beforeWidth == afterWidth &&
				beforeHeight == afterHeight &&
				beforeAltScreen == afterAltScreen
			if damage.IncrementalRowsReliable {
				damage.DirectDamageTouchedRows = cloneIntSlice(dirtyRows)
				damage.RowCopies = rowCopiesFromReconcilePlan(beforeScreen, cachePlan)
			}
			damage.DiffCPUNanos = time.Since(diffStart).Nanoseconds()
			traceCount("vterm.write.changed_rows", damageChangedRowCount(damage))
			traceCount("vterm.write.changed_cells", damageChangedCellCount(damage))
			reconcileFinish(0)
			return n, err, damage
		}
		if hasDirectDamage &&
			beforeWidth == afterWidth &&
			beforeHeight == afterHeight &&
			beforeAltScreen == afterAltScreen {
			evictedOps, altEvictedOps := v.evictedAppendOpsFromCharmVTDamages(directDamages, beforeAltScreen, beforeScreenTimestamps, beforeScreenRowKinds, beforeScreenOwnership, beforePersistedRows)
			historyOps := filterNonEmptyEvictedAppendOps(evictedOps)
			alternateOps := filterNonEmptyEvictedAppendOps(altEvictedOps)
			directStats := directDamageStats(directDamages, afterWidth, afterHeight)
			if reason, broad := directStats.fullReplaceReason(); broad && (damageMode != writeDamageSemanticOnly || len(historyOps) == 0) {
				damage = v.writeDamageRequiresFullReplaceLocked(cachePlan, reason)
				damage.SemanticOps = v.semanticControlOpsFromCharmVTDamagesLocked(directDamages, beforeScreenTimestamps, beforeScreenRowKinds)
				if len(historyOps) > 0 {
					applySemanticHistoryOpsToDamage(&damage, cachePlan, historyOps, v.scrollbackRowCountLocked())
				}
				traceCount("vterm.write.direct_damage_full_replace", 1)
			} else if directOps, ok := v.damageOpsFromCharmVTDamages(directDamages, afterWidth, afterHeight, v.screenTimestamps, v.screenRowKinds); ok {
				damage = v.writeDamageFromDirectOpsLocked(directOps, v.semanticControlOpsFromCharmVTDamagesLocked(directDamages, beforeScreenTimestamps, beforeScreenRowKinds), cachePlan, damageMode)
				if len(historyOps) > 0 {
					applySemanticHistoryOpsToDamage(&damage, cachePlan, historyOps, v.scrollbackRowCountLocked())
				}
			} else {
				damage = v.writeDamageRequiresFullReplaceLocked(cachePlan, "direct_damage_unsupported")
				damage.SemanticOps = v.semanticControlOpsFromCharmVTDamagesLocked(directDamages, beforeScreenTimestamps, beforeScreenRowKinds)
				if len(historyOps) > 0 {
					applySemanticHistoryOpsToDamage(&damage, cachePlan, historyOps, v.scrollbackRowCountLocked())
				}
			}
			if len(alternateOps) > 0 {
				damage.AlternateAppend = alternateOps
			}
			damage.EvictedAppend = evictedOps
			damage.DirectDamageItems = directStats.Items
			damage.DirectDamageRows = directStats.Rows
			damage.DirectDamageCells = directStats.Cells
			damage.DirectDamageTouchedRows = cloneIntSlice(directStats.TouchedRows)
		} else {
			damage = v.writeDamageRequiresFullReplaceLocked(cachePlan, "screen_shape_changed")
			if hasDirectDamage {
				damage.SemanticOps = v.semanticControlOpsFromCharmVTDamagesLocked(directDamages, beforeScreenTimestamps, beforeScreenRowKinds)
				// 中文说明：shape-changed 批次（resize / alt 切换跨 write 边界）此前
				// 完全不带 eviction payload；tap vterm 关闭 ring 时 ScrollbackAppend
				// 兜底也为空，进 alt 前滚出的主屏行会永久丢失。这里同样按有序
				// damage 流归属 primary eviction，只填 EvictedAppend，不改旧
				// AlternateAppend/ScrollbackAppend 消费契约。
				evictedOps, _ := v.evictedAppendOpsFromCharmVTDamages(directDamages, beforeAltScreen, beforeScreenTimestamps, beforeScreenRowKinds, beforeScreenOwnership, beforePersistedRows)
				damage.EvictedAppend = evictedOps
			}
		}
		damage.DiffCPUNanos = time.Since(diffStart).Nanoseconds()
		traceCount("vterm.write.changed_rows", damageChangedRowCount(damage))
		traceCount("vterm.write.changed_cells", damageChangedCellCount(damage))
		traceCount("vterm.write.diff_cpu_ns", int(damage.DiffCPUNanos))
	}
	reconcileFinish(0)
	return n, err, damage
}

func (v *VTerm) writeLatest(data []byte) (n int, err error, damage WriteDamage) {
	finish := traceMeasure("vterm.write_latest")
	defer func() {
		finish(len(data))
	}()
	v.mu.Lock()
	defer v.mu.Unlock()
	v.ensureScreenFingerprintCacheLocked()
	beforeWidth := 0
	beforeHeight := 0
	beforeAltScreen := false
	if v.emu != nil {
		beforeWidth = v.emu.Width()
		beforeHeight = v.emu.Height()
		beforeAltScreen = v.emu.IsAltScreen()
	}
	snapshotFinish := traceMeasure("vterm.write_latest.before_snapshot")
	beforeScreen := reuseRowFingerprintSlice(v.screenFingerprintScratch, v.screenFingerprintCache)
	v.screenFingerprintScratch = beforeScreen
	beforeScrollbackLen := v.scrollbackRowCountLocked()
	beforeScreenTimestamps := v.screenTimestamps
	beforeScreenRowKinds := v.screenRowKinds
	beforeScreenOwnership := v.screenOwnership
	snapshotFinish(0)
	defer func() {
		if r := recover(); r != nil {
			n = 0
			err = fmt.Errorf("vterm write panic: %v", r)
			damage = WriteDamage{}
		}
	}()
	normalizeFinish := traceMeasure("vterm.write_latest.normalize")
	normalized := normalizeRenderableUTF8(data)
	normalizeFinish(len(normalized))
	clearTouchedFinish := traceMeasure("vterm.write_latest.clear_touched")
	v.clearTouchedRowsLocked()
	clearTouchedFinish(0)
	emulatorFinish := traceMeasure("vterm.write_latest.emulator")
	n, err = safeEmulatorWrite(v.emu, normalized)
	emulatorFinish(len(normalized))
	pos := v.emu.CursorPosition()
	v.cursor.Row = pos.Y
	v.cursor.Col = pos.X
	v.modes.AlternateScreen = v.emu.IsAltScreen()
	diffStart := time.Now()
	reconcileFinish := traceMeasure("vterm.write_latest.reconcile")
	afterWidth := 0
	afterHeight := 0
	afterAltScreen := false
	if v.emu != nil {
		afterWidth = v.emu.Width()
		afterHeight = v.emu.Height()
		afterAltScreen = v.emu.IsAltScreen()
	}
	v.capturePrimaryMetadataOnAltEnter(beforeAltScreen, afterAltScreen, beforeScreenTimestamps, beforeScreenOwnership)
	dirtyRows, dirtyReliable := v.consumeTouchedRowsLocked()
	now := time.Now().UTC()
	var afterScreen []rowFingerprint
	fingerprintFinish := traceMeasure("vterm.write_latest.reconcile.fingerprint")
	switch {
	case !dirtyReliable,
		beforeWidth != afterWidth,
		beforeHeight != afterHeight,
		beforeAltScreen != afterAltScreen:
		afterScreen = v.screenRowFingerprintsLocked()
		v.screenFingerprintCache = afterScreen
	default:
		v.ensureScreenFingerprintCacheLocked()
		for _, row := range dirtyRows {
			if row < 0 || row >= len(v.screenFingerprintCache) {
				continue
			}
			v.screenFingerprintCache[row] = v.screenRowFingerprintLocked(row)
		}
		afterScreen = v.screenFingerprintCache
	}
	fingerprintFinish(0)
	metadataFinish := traceMeasure("vterm.write_latest.reconcile.metadata")
	cachePlan := v.reconcileRowMetadataLocked(beforeScreen, beforeScreenTimestamps, beforeScreenRowKinds, beforeScreenOwnership, beforeScrollbackLen, afterScreen, now)
	v.restorePrimaryMetadataOnAltExit(beforeAltScreen, afterAltScreen, afterHeight)
	metadataFinish(0)
	rowCacheFinish := traceMeasure("vterm.write_latest.reconcile.row_cache")
	v.reconcileRowCachesLocked(beforeScreen, cachePlan)
	rowCacheFinish(0)
	damage = v.writeDamageHeaderLocked(cachePlan)
	damage.RequiresFullReplace = true
	damage.IncrementalRowsReliable = dirtyReliable &&
		beforeWidth == afterWidth &&
		beforeHeight == afterHeight &&
		beforeAltScreen == afterAltScreen
	if damage.IncrementalRowsReliable {
		damage.DirectDamageTouchedRows = cloneIntSlice(dirtyRows)
		damage.RowCopies = rowCopiesFromReconcilePlan(beforeScreen, cachePlan)
	}
	damage.DiffCPUNanos = time.Since(diffStart).Nanoseconds()
	traceCount("vterm.write_latest.changed_rows", damageChangedRowCount(damage))
	traceCount("vterm.write_latest.changed_cells", damageChangedCellCount(damage))
	traceCount("vterm.write_latest.diff_cpu_ns", int(damage.DiffCPUNanos))
	reconcileFinish(0)
	return n, err, damage
}

func (v *VTerm) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.emu == nil {
		return nil
	}
	err := v.emu.Close()
	<-v.done
	v.emu = nil
	v.invalidateRowCachesLocked()
	v.invalidateFingerprintCacheLocked()
	return err
}

func (v *VTerm) LoadSnapshot(screen ScreenData, cursor CursorState, modes TerminalModes) {
	v.LoadSnapshotWithScrollback(nil, screen, cursor, modes)
}

func (v *VTerm) LoadSnapshotWithScrollback(scrollback [][]Cell, screen ScreenData, cursor CursorState, modes TerminalModes) {
	v.LoadSnapshotWithTimestamps(scrollback, nil, screen, nil, cursor, modes)
}

func (v *VTerm) LoadSnapshotWithTimestamps(scrollback [][]Cell, scrollbackTimestamps []time.Time, screen ScreenData, screenTimestamps []time.Time, cursor CursorState, modes TerminalModes) {
	v.LoadSnapshotWithMetadata(scrollback, scrollbackTimestamps, nil, screen, screenTimestamps, nil, cursor, modes)
}

func (v *VTerm) LoadSnapshotWithMetadata(scrollback [][]Cell, scrollbackTimestamps []time.Time, scrollbackRowKinds []string, screen ScreenData, screenTimestamps []time.Time, screenRowKinds []string, cursor CursorState, modes TerminalModes) {
	v.LoadSnapshotWithExtendedMetadata(scrollback, scrollbackTimestamps, scrollbackRowKinds, nil, screen, screenTimestamps, screenRowKinds, nil, cursor, modes)
}

func (v *VTerm) LoadSnapshotWithExtendedMetadata(scrollback [][]Cell, scrollbackTimestamps []time.Time, scrollbackRowKinds []string, scrollbackWrapped []bool, screen ScreenData, screenTimestamps []time.Time, screenRowKinds []string, screenWrapped []bool, cursor CursorState, modes TerminalModes) {
	v.LoadSizedSnapshotWithExtendedMetadata(0, 0, scrollback, scrollbackTimestamps, scrollbackRowKinds, scrollbackWrapped, screen, screenTimestamps, screenRowKinds, screenWrapped, cursor, modes)
}

func (v *VTerm) LoadSizedSnapshotWithExtendedMetadata(cols, rows int, scrollback [][]Cell, scrollbackTimestamps []time.Time, scrollbackRowKinds []string, scrollbackWrapped []bool, screen ScreenData, screenTimestamps []time.Time, screenRowKinds []string, screenWrapped []bool, cursor CursorState, modes TerminalModes) {
	v.LoadSizedSnapshotWithOwnership(cols, rows, scrollback, scrollbackTimestamps, scrollbackRowKinds, scrollbackWrapped, nil, screen, screenTimestamps, screenRowKinds, screenWrapped, nil, cursor, modes)
}

func (v *VTerm) LoadSnapshotWithOwnership(scrollback [][]Cell, scrollbackTimestamps []time.Time, scrollbackRowKinds []string, scrollbackWrapped []bool, scrollbackOwnership []string, screen ScreenData, screenTimestamps []time.Time, screenRowKinds []string, screenWrapped []bool, screenOwnership []string, cursor CursorState, modes TerminalModes) {
	v.LoadSizedSnapshotWithOwnership(0, 0, scrollback, scrollbackTimestamps, scrollbackRowKinds, scrollbackWrapped, scrollbackOwnership, screen, screenTimestamps, screenRowKinds, screenWrapped, screenOwnership, cursor, modes)
}

func (v *VTerm) LoadSizedSnapshotWithOwnership(cols, rows int, scrollback [][]Cell, scrollbackTimestamps []time.Time, scrollbackRowKinds []string, scrollbackWrapped []bool, scrollbackOwnership []string, screen ScreenData, screenTimestamps []time.Time, screenRowKinds []string, screenWrapped []bool, screenOwnership []string, cursor CursorState, modes TerminalModes) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.emu != nil {
		_ = v.emu.Close()
		<-v.done
	}

	height := maxInt(rows, len(screen.Cells))
	width := maxInt(cols, 1)
	for _, row := range screen.Cells {
		if len(row) > width {
			width = len(row)
		}
	}
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	if cursor.Col+1 > width {
		width = cursor.Col + 1
	}
	if cursor.Row+1 > height {
		height = cursor.Row + 1
	}

	v.cursor = cursor
	v.modes = modes
	v.resetEmulator(width, height)
	v.scrollbackTimestamps = normalizeTimeSlice(scrollbackTimestamps, len(scrollback))
	v.scrollbackRowKinds = normalizeStringSlice(scrollbackRowKinds, len(scrollback))
	v.scrollbackOwnership = normalizeStringSlice(scrollbackOwnership, len(scrollback))
	v.screenTimestamps = normalizeTimeSlice(screenTimestamps, height)
	v.screenRowKinds = normalizeStringSlice(screenRowKinds, height)
	v.screenOwnership = normalizeStringSlice(screenOwnership, height)
	v.loadMouseModesLocked(modes)
	if len(scrollback) > 0 {
		sb := v.emu.Emulator.Scrollback()
		for i, row := range scrollback {
			sb.PushWrapped(uvLine(row), boolAt(scrollbackWrapped, i))
		}
	}
	v.alignScrollbackMetadataLocked()
	if modes.AlternateScreen {
		_, _ = v.emu.Write([]byte("\x1b[?1049h"))
	}
	if len(screen.Cells) > 0 {
		for y, row := range screen.Cells {
			if y >= height {
				break
			}
			for x, cell := range row {
				if x >= width || isWideContinuationCell(cell) {
					continue
				}
				v.emu.Emulator.SetCell(x, y, uvCell(cell))
			}
			v.emu.Emulator.SetScreenLineUsed(y, snapshotRowUsedWidth(row, width))
		}
	}
	for y, wrapped := range normalizeBoolSlice(screenWrapped, height) {
		v.emu.SetScreenLineWrapped(y, wrapped)
	}
	if cursor.Visible {
		_, _ = v.emu.Write([]byte("\x1b[?25h"))
	} else {
		_, _ = v.emu.Write([]byte("\x1b[?25l"))
	}
	if modes.ApplicationCursor {
		_, _ = v.emu.Write([]byte("\x1b[?1h"))
	} else {
		_, _ = v.emu.Write([]byte("\x1b[?1l"))
	}
	if modes.BracketedPaste {
		_, _ = v.emu.Write([]byte("\x1b[?2004h"))
	} else {
		_, _ = v.emu.Write([]byte("\x1b[?2004l"))
	}
	if !modes.AutoWrap {
		_, _ = v.emu.Write([]byte("\x1b[?7l"))
	}
	if cursor.Row >= 0 && cursor.Col >= 0 {
		_, _ = v.emu.Write([]byte(fmt.Sprintf("\x1b[%d;%dH", cursor.Row+1, cursor.Col+1)))
	}
	v.invalidateRowCachesLocked()
	v.invalidateFingerprintCacheLocked()
}

func (v *VTerm) ApplyScreenUpdate(update ScreenUpdate) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.applyScreenUpdateLocked(normalizeScreenUpdate(update))
}

func (v *VTerm) applyScreenUpdateLocked(update ScreenUpdate) bool {
	if !v.canApplyScreenUpdateLocked(update) {
		return false
	}
	targetCols, targetRows := v.screenUpdateTargetSizeLocked(update)
	if targetCols != v.emu.Width() || targetRows != v.emu.Height() {
		v.resizeLocked(targetCols, targetRows)
	}
	if !v.applyScreenUpdateScrollbackLocked(update) {
		return false
	}
	if len(update.Ops) > 0 {
		return v.applyScreenUpdateOpsLocked(update, targetCols, targetRows)
	}
	v.applyScreenScrollLocked(update.ScreenScroll)

	var b strings.Builder
	modes := update.Modes
	cursor := update.Cursor
	writeTerminalModesANSI(&b, modes)
	writeCursorShapeANSI(&b, cursor)
	if cursor.Row >= 0 && cursor.Col >= 0 {
		fmt.Fprintf(&b, "\x1b[%d;%dH", cursor.Row+1, cursor.Col+1)
	}
	if cursor.Visible {
		b.WriteString("\x1b[?25h")
	} else {
		b.WriteString("\x1b[?25l")
	}

	if b.Len() > 0 {
		if _, err := safeEmulatorWrite(v.emu, []byte(b.String())); err != nil {
			return false
		}
	}

	height := v.emu.Height()
	v.screenTimestamps = normalizeTimeSlice(v.screenTimestamps, height)
	v.screenRowKinds = normalizeStringSlice(v.screenRowKinds, height)
	v.screenOwnership = normalizeStringSlice(v.screenOwnership, height)
	v.cursor = cursor
	v.modes = modes
	v.loadMouseModesLocked(modes)
	v.invalidateRowCachesLocked()
	v.invalidateFingerprintCacheLocked()
	return true
}

func (v *VTerm) applyScreenScrollLocked(delta int) {
	if v == nil || v.emu == nil || delta == 0 {
		return
	}
	height := v.emu.Height()
	width := v.emu.Width()
	if height <= 0 || width <= 0 {
		return
	}
	if delta >= height || delta <= -height {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				v.emu.Emulator.SetCell(x, y, uvBlankCell())
			}
			v.emu.Emulator.SetScreenLineWrapped(y, false)
			v.emu.Emulator.SetScreenLineUsed(y, 0)
		}
		zeroTime := make([]time.Time, height)
		zeroKinds := make([]string, height)
		zeroOwnership := make([]string, height)
		v.screenTimestamps = zeroTime
		v.screenRowKinds = zeroKinds
		v.screenOwnership = zeroOwnership
		return
	}
	screen := make([][]Cell, height)
	wrapped := make([]bool, height)
	for y := 0; y < height; y++ {
		screen[y] = cloneCellSlice(v.screenRowViewLocked(y))
		wrapped[y] = v.screenRowWrappedAtLocked(y)
	}
	nextTimes := normalizeTimeSlice(v.screenTimestamps, height)
	nextKinds := normalizeStringSlice(v.screenRowKinds, height)
	nextOwnership := normalizeStringSlice(v.screenOwnership, height)
	if delta > 0 {
		for y := 0; y < height-delta; y++ {
			screen[y] = screen[y+delta]
			wrapped[y] = wrapped[y+delta]
			nextTimes[y] = nextTimes[y+delta]
			nextKinds[y] = nextKinds[y+delta]
			nextOwnership[y] = nextOwnership[y+delta]
		}
		for y := height - delta; y < height; y++ {
			screen[y] = nil
			wrapped[y] = false
			nextTimes[y] = time.Time{}
			nextKinds[y] = ""
			nextOwnership[y] = ""
		}
	} else {
		shift := -delta
		for y := height - 1; y >= shift; y-- {
			screen[y] = screen[y-shift]
			wrapped[y] = wrapped[y-shift]
			nextTimes[y] = nextTimes[y-shift]
			nextKinds[y] = nextKinds[y-shift]
			nextOwnership[y] = nextOwnership[y-shift]
		}
		for y := 0; y < shift; y++ {
			screen[y] = nil
			wrapped[y] = false
			nextTimes[y] = time.Time{}
			nextKinds[y] = ""
			nextOwnership[y] = ""
		}
	}
	for y := 0; y < height; y++ {
		row := screen[y]
		for x := 0; x < width; x++ {
			if x < len(row) {
				if isWideContinuationCell(row[x]) {
					continue
				}
				v.emu.Emulator.SetCell(x, y, uvCell(row[x]))
				continue
			}
			v.emu.Emulator.SetCell(x, y, uvBlankCell())
		}
		v.emu.Emulator.SetScreenLineWrapped(y, wrapped[y])
		v.emu.Emulator.SetScreenLineUsed(y, len(row))
	}
	v.screenTimestamps = nextTimes
	v.screenRowKinds = nextKinds
	v.screenOwnership = nextOwnership
}

func (v *VTerm) applyScreenUpdateOpsLocked(update ScreenUpdate, targetCols, targetRows int) bool {
	if v == nil || v.emu == nil {
		return false
	}
	screen := cloneDenseCellRows(v.screenRowsLocked(), targetRows, targetCols)
	nextWrapped := v.screenWrappedLocked(targetRows)
	nextTimes := normalizeTimeSlice(v.screenTimestamps, targetRows)
	nextKinds := normalizeStringSlice(v.screenRowKinds, targetRows)
	nextOwnership := normalizeStringSlice(v.screenOwnership, targetRows)
	changedRows := make(map[int]struct{}, targetRows)
	changedRowUsed := make(map[int]int)
	markRowRange := func(start, end int) {
		for row := start; row < end; row++ {
			if row < 0 || row >= targetRows {
				continue
			}
			changedRows[row] = struct{}{}
		}
	}
	extendChangedRowUsed := func(row, used int) {
		if row < 0 || row >= targetRows || used <= 0 {
			return
		}
		if used > targetCols {
			used = targetCols
		}
		if used > changedRowUsed[row] {
			changedRowUsed[row] = used
		}
	}
	for _, op := range update.Ops {
		switch op.Code {
		case ScreenOpWriteSpan:
			if op.Row < 0 || op.Row >= targetRows {
				continue
			}
			localCells := op.Cells
			for col, cell := range localCells {
				dstCol := op.Col + col
				if dstCol < 0 || dstCol >= targetCols {
					continue
				}
				screen[op.Row][dstCol] = cell
			}
			nextTimes[op.Row] = op.Timestamp
			nextKinds[op.Row] = op.RowKind
			nextOwnership[op.Row] = ""
			if op.WrappedSet {
				nextWrapped[op.Row] = op.Wrapped
			}
			extendChangedRowUsed(op.Row, op.Col+len(localCells))
			markRowRange(op.Row, op.Row+1)
		case ScreenOpClearToEOL:
			if op.Row < 0 || op.Row >= targetRows {
				continue
			}
			for col := maxInt(op.Col, 0); col < targetCols; col++ {
				screen[op.Row][col] = Cell{Content: " ", Width: 1}
			}
			nextTimes[op.Row] = op.Timestamp
			nextKinds[op.Row] = op.RowKind
			nextOwnership[op.Row] = ""
			if op.WrappedSet {
				nextWrapped[op.Row] = op.Wrapped
			}
			markRowRange(op.Row, op.Row+1)
		case ScreenOpClearRect:
			damageOp := DamageOp{
				Code:      ScreenOpClearRect,
				Rect:      DamageRect{X: op.Rect.X, Y: op.Rect.Y, Width: op.Rect.Width, Height: op.Rect.Height},
				Timestamp: op.Timestamp,
				RowKind:   op.RowKind,
			}
			applyDamageClearRect(screen, damageOp)
			for row := op.Rect.Y; row < op.Rect.Y+op.Rect.Height && row < targetRows; row++ {
				if row < 0 {
					continue
				}
				nextTimes[row] = op.Timestamp
				nextKinds[row] = op.RowKind
				nextOwnership[row] = ""
				if op.WrappedSet {
					nextWrapped[row] = op.Wrapped
				} else if op.Rect.X == 0 && op.Rect.Width >= targetCols {
					nextWrapped[row] = false
				}
			}
			markRowRange(op.Rect.Y, op.Rect.Y+op.Rect.Height)
		case ScreenOpScrollRect:
			damageOp := DamageOp{
				Code: ScreenOpScrollRect,
				Rect: DamageRect{X: op.Rect.X, Y: op.Rect.Y, Width: op.Rect.Width, Height: op.Rect.Height},
				Dx:   op.Dx,
				Dy:   op.Dy,
			}
			beforeTimes := append([]time.Time(nil), nextTimes...)
			beforeKinds := append([]string(nil), nextKinds...)
			beforeWrapped := append([]bool(nil), nextWrapped...)
			beforeOwnership := append([]string(nil), nextOwnership...)
			applyDamageScrollRect(screen, damageOp)
			if op.Dx == 0 && op.Rect.X == 0 && op.Rect.Width >= targetCols {
				for row := op.Rect.Y; row < op.Rect.Y+op.Rect.Height && row < targetRows; row++ {
					if row < 0 {
						continue
					}
					srcRow := row - op.Dy
					if srcRow >= op.Rect.Y && srcRow < op.Rect.Y+op.Rect.Height && srcRow >= 0 && srcRow < len(beforeTimes) {
						nextTimes[row] = beforeTimes[srcRow]
						nextKinds[row] = beforeKinds[srcRow]
						nextWrapped[row] = boolAt(beforeWrapped, srcRow)
						nextOwnership[row] = stringAt(beforeOwnership, srcRow)
						continue
					}
					nextTimes[row] = time.Time{}
					nextKinds[row] = ""
					nextWrapped[row] = false
					nextOwnership[row] = ""
				}
			}
			markRowRange(op.Rect.Y, op.Rect.Y+op.Rect.Height)
		case ScreenOpCopyRect:
			damageOp := DamageOp{
				Code: ScreenOpCopyRect,
				Src:  DamageRect{X: op.Src.X, Y: op.Src.Y, Width: op.Src.Width, Height: op.Src.Height},
				DstX: op.DstX,
				DstY: op.DstY,
			}
			beforeTimes := append([]time.Time(nil), nextTimes...)
			beforeKinds := append([]string(nil), nextKinds...)
			beforeWrapped := append([]bool(nil), nextWrapped...)
			beforeOwnership := append([]string(nil), nextOwnership...)
			applyDamageCopyRect(screen, damageOp)
			if op.Src.X == 0 && op.DstX == 0 && op.Src.Width >= targetCols {
				for row := 0; row < op.Src.Height; row++ {
					srcRow := op.Src.Y + row
					dstRow := op.DstY + row
					if srcRow < 0 || srcRow >= len(beforeTimes) || dstRow < 0 || dstRow >= targetRows {
						continue
					}
					nextTimes[dstRow] = beforeTimes[srcRow]
					nextKinds[dstRow] = beforeKinds[srcRow]
					nextWrapped[dstRow] = boolAt(beforeWrapped, srcRow)
					nextOwnership[dstRow] = stringAt(beforeOwnership, srcRow)
				}
			}
			markRowRange(op.DstY, op.DstY+op.Src.Height)
		case ScreenOpResize:
			if op.Size.Cols > 0 {
				targetCols = int(op.Size.Cols)
			}
			if op.Size.Rows > 0 {
				targetRows = int(op.Size.Rows)
			}
		}
	}
	for row := range changedRows {
		if row < 0 || row >= targetRows {
			continue
		}
		used := 0
		for col := 0; col < targetCols; col++ {
			if col < len(screen[row]) && !cellIsBlank(screen[row][col]) {
				used = col + 1
			}
			if isWideContinuationCell(screen[row][col]) {
				continue
			}
			v.emu.Emulator.SetCell(col, row, uvCell(screen[row][col]))
		}
		used = maxInt(used, changedRowUsed[row])
		v.emu.Emulator.SetScreenLineWrapped(row, boolAt(nextWrapped, row))
		v.emu.Emulator.SetScreenLineUsed(row, used)
	}

	modes := update.Modes
	cursor := update.Cursor
	var b strings.Builder
	writeTerminalModesANSI(&b, modes)
	writeCursorShapeANSI(&b, cursor)
	if cursor.Row >= 0 && cursor.Col >= 0 {
		fmt.Fprintf(&b, "\x1b[%d;%dH", cursor.Row+1, cursor.Col+1)
	}
	if cursor.Visible {
		b.WriteString("\x1b[?25h")
	} else {
		b.WriteString("\x1b[?25l")
	}
	if b.Len() > 0 {
		if _, err := safeEmulatorWrite(v.emu, []byte(b.String())); err != nil {
			return false
		}
	}
	v.screenTimestamps = nextTimes
	v.screenRowKinds = nextKinds
	v.screenOwnership = nextOwnership
	v.cursor = cursor
	v.modes = modes
	v.loadMouseModesLocked(modes)
	v.invalidateRowCachesLocked()
	v.invalidateFingerprintCacheLocked()
	return true
}

func (v *VTerm) canApplyScreenUpdateLocked(update ScreenUpdate) bool {
	if v == nil || v.emu == nil || update.FullReplace {
		return false
	}
	if update.ResetScrollback {
		return false
	}
	targetCols, targetRows := v.screenUpdateTargetSizeLocked(update)
	if targetCols <= 0 || targetRows <= 0 {
		return false
	}
	if v.emu.IsAltScreen() != update.Modes.AlternateScreen {
		return false
	}
	for _, op := range update.Ops {
		switch op.Code {
		case ScreenOpWriteSpan:
			if op.Row < 0 || op.Row >= targetRows || op.Col < 0 || op.Col > targetCols {
				return false
			}
		case ScreenOpClearToEOL:
			if op.Row < 0 || op.Row >= targetRows || op.Col < 0 || op.Col > targetCols {
				return false
			}
		case ScreenOpScrollRect, ScreenOpClearRect:
			if op.Rect.Y < 0 || op.Rect.X < 0 || op.Rect.Width < 0 || op.Rect.Height < 0 {
				return false
			}
		case ScreenOpCopyRect:
			if op.Src.Y < 0 || op.Src.X < 0 || op.Src.Width < 0 || op.Src.Height < 0 || op.DstX < 0 || op.DstY < 0 {
				return false
			}
		}
	}
	return true
}

func (v *VTerm) applyScreenUpdateScrollbackLocked(update ScreenUpdate) bool {
	if v == nil || v.emu == nil {
		return false
	}
	if update.ScrollbackTrim <= 0 && len(update.ScrollbackAppend) == 0 {
		return true
	}

	sb := v.emu.Emulator.Scrollback()
	trim := update.ScrollbackTrim
	if trim < 0 {
		trim = 0
	}
	currentLen := sb.Len()
	if trim > currentLen {
		trim = currentLen
	}

	sb.TrimOldest(trim)
	for _, row := range update.ScrollbackAppend {
		sb.PushWrapped(uvLine(row.Cells), row.WrappedSet && row.Wrapped)
	}

	nextTimestamps := append([]time.Time(nil), tailTimeSlice(v.scrollbackTimestamps, trim)...)
	nextKinds := append([]string(nil), tailStringSlice(v.scrollbackRowKinds, trim)...)
	nextOwnership := append([]string(nil), tailStringSlice(v.scrollbackOwnership, trim)...)
	for _, row := range update.ScrollbackAppend {
		nextTimestamps = append(nextTimestamps, row.Timestamp)
		nextKinds = append(nextKinds, row.RowKind)
		nextOwnership = append(nextOwnership, row.Ownership)
	}

	scrollbackLen := sb.Len()
	if len(nextTimestamps) > scrollbackLen {
		nextTimestamps = nextTimestamps[len(nextTimestamps)-scrollbackLen:]
	}
	if len(nextKinds) > scrollbackLen {
		nextKinds = nextKinds[len(nextKinds)-scrollbackLen:]
	}
	if len(nextOwnership) > scrollbackLen {
		nextOwnership = nextOwnership[len(nextOwnership)-scrollbackLen:]
	}
	v.scrollbackTimestamps = normalizeTimeSlice(nextTimestamps, scrollbackLen)
	v.scrollbackRowKinds = normalizeStringSlice(nextKinds, scrollbackLen)
	v.scrollbackOwnership = normalizeStringSlice(nextOwnership, scrollbackLen)
	v.alignScrollbackMetadataLocked()
	return true
}

func (v *VTerm) screenUpdateTargetSizeLocked(update ScreenUpdate) (cols, rows int) {
	if v == nil || v.emu == nil {
		return 0, 0
	}
	cols, rows = v.emu.Width(), v.emu.Height()
	if update.Size.Cols > 0 {
		cols = int(update.Size.Cols)
	}
	if update.Size.Rows > 0 {
		rows = int(update.Size.Rows)
	}
	return cols, rows
}

func (v *VTerm) Resize(cols, rows int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.resizeLocked(cols, rows)
}

func (v *VTerm) ResizeWithDamage(cols, rows int) WriteDamage {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.resizeWithDamageLocked(cols, rows)
}

func (v *VTerm) resizeLocked(cols, rows int) {
	_ = v.resizeWithDamageLocked(cols, rows)
}

func (v *VTerm) resizeWithDamageLocked(cols, rows int) WriteDamage {
	if v == nil || v.emu == nil {
		return WriteDamage{}
	}
	v.ensureScreenFingerprintCacheLocked()
	beforeScreen := reuseRowFingerprintSlice(v.screenFingerprintScratch, v.screenFingerprintCache)
	v.screenFingerprintScratch = beforeScreen
	beforeScrollbackLen := v.scrollbackRowCountLocked()
	beforeScreenTimestamps := v.screenTimestamps
	beforeScreenRowKinds := v.screenRowKinds
	beforeScreenOwnership := v.screenOwnership
	beforeScreenRows := v.screenRowsForResizeLocked(len(beforeScreen))
	beforeScreenTailFills := v.screenTailFillsForResizeLocked(len(beforeScreen))
	beforeScreenWrapped := v.screenWrappedLocked(len(beforeScreen))
	beforeCursorRow := v.cursor.Row
	var beforeResizeRows []resizeReflowLine
	var tailResizeRows []resizeReflowLine
	var resizePlan resizeHistoryPlan
	beforeCols, beforeRows := v.emu.Width(), v.emu.Height()
	if !v.emu.IsAltScreen() {
		beforeResizeRows = resizeReflowRowsForResize(beforeScreenRows, beforeScreenTailFills, beforeScreenWrapped, beforeScreenTimestamps, beforeScreenRowKinds, cols, rows)
		tailResizeRows = trimTrailingBlankResizeReflowLines(beforeResizeRows)
	}
	if shouldUseTailFillResizeWriteback(tailResizeRows) || shouldTailVisibleRowsForResize(beforeCols, beforeRows, cols, rows, tailResizeRows) {
		resizePlan = planTailResizeHistoryAppend(tailResizeRows, rows)
		v.emu.ResizeAndTailScreen(cols, rows, resizeReflowLinesUVRows(tailResizeRows, cols), resizeReflowLinesWrapped(tailResizeRows), resizeReflowLinesUsed(tailResizeRows))
	} else {
		v.emu.Resize(cols, rows)
	}
	pos := v.emu.CursorPosition()
	v.cursor.Row = pos.Y
	v.cursor.Col = pos.X
	afterScreen := v.screenRowFingerprintsLocked()
	v.screenFingerprintCache = afterScreen
	plan := v.reconcileRowMetadataLocked(beforeScreen, beforeScreenTimestamps, beforeScreenRowKinds, beforeScreenOwnership, beforeScrollbackLen, afterScreen, time.Now().UTC())
	v.invalidateRowCachesLocked()
	damage := v.writeDamageRequiresFullReplaceLocked(plan, "resize")
	afterScreenRows := v.screenRowsForResizeLocked(len(afterScreen))
	afterScreenTailFills := v.screenTailFillsForResizeLocked(len(afterScreen))
	afterScreenWrapped := v.screenWrappedLocked(len(afterScreen))
	damage.ScrollbackAppend = resizePlan.scrollbackAppend()
	if shouldFallbackResizeHistoryAppend(beforeCols, beforeRows, cols, rows, beforeResizeRows, resizePlan) {
		damage.ScrollbackAppend = resizeScrollbackAppendFromReflowedRows(beforeResizeRows, afterScreenRows, afterScreenTailFills, v.screenTimestamps, v.screenRowKinds, afterScreenWrapped)
	}
	damage.LiveTailAppendRows = trailingWrappedDamageRows(damage.ScrollbackAppend)
	damage.ResizeLiveTailRows = resizeLiveTailRowsFromPlan(resizePlan, beforeCursorRow)
	damage.ScrollbackTrim = 0
	damage.SizeCols = cols
	damage.SizeRows = rows
	if gridtrace.Enabled() {
		gridtrace.Log(
			"vterm.resize.summary",
			"old_cols", beforeCols,
			"old_rows", beforeRows,
			"new_cols", cols,
			"new_rows", rows,
			"before_scrollback_len", beforeScrollbackLen,
			"after_scrollback_len", v.scrollbackRowCountLocked(),
			"damage_scrollback_append_rows", len(damage.ScrollbackAppend),
			"tail_plan", !resizePlan.isZero(),
			"before_resize_rows", len(beforeResizeRows),
		)
	}
	return damage
}

func (v *VTerm) CellAt(x, y int) Cell {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.convertCell(v.emu.CellAt(x, y))
}

func (v *VTerm) ScreenContent() ScreenData {
	v.mu.RLock()
	defer v.mu.RUnlock()
	height := v.emu.Height()
	rows := make([][]Cell, height)
	for y := 0; y < height; y++ {
		rows[y] = cloneCellSlice(v.screenRowViewLocked(y))
	}
	return ScreenData{
		Cells:             rows,
		IsAlternateScreen: v.emu.IsAltScreen(),
	}
}

func (v *VTerm) UsedScreenContent() ScreenData {
	v.mu.RLock()
	defer v.mu.RUnlock()
	height := v.emu.Height()
	rows := make([][]Cell, height)
	for y := 0; y < height; y++ {
		rows[y] = cloneCellSlice(v.screenRowUsedViewLocked(y))
	}
	return ScreenData{
		Cells:             rows,
		IsAlternateScreen: v.emu.IsAltScreen(),
	}
}

// TrimmedScreenContent preserves live row positions but drops pure default
// blank suffixes. 这条路径只服务高频 live snapshot/protocol，不改变
// ScreenContent/UsedScreenContent 这些需要完整索引语义的公开快照。
func (v *VTerm) TrimmedScreenContent() ScreenData {
	v.mu.RLock()
	defer v.mu.RUnlock()
	height := v.emu.Height()
	rows := make([][]Cell, height)
	for y := 0; y < height; y++ {
		rows[y] = v.trimmedScreenRowCellsLocked(y)
	}
	return ScreenData{
		Cells:             rows,
		IsAlternateScreen: v.emu.IsAltScreen(),
	}
}

func (v *VTerm) VisitTrimmedScreenRows(visit func(rowIndex int, cellCount int, cellAt func(int) Cell)) TrimmedScreenRowsInfo {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.emu == nil {
		return TrimmedScreenRowsInfo{}
	}
	width := v.emu.Width()
	height := v.emu.Height()
	info := TrimmedScreenRowsInfo{
		Cols:              width,
		Rows:              height,
		IsAlternateScreen: v.emu.IsAltScreen(),
		Cursor:            v.cursor,
		Modes:             v.modes,
	}
	if visit == nil {
		return info
	}
	for y := 0; y < height; y++ {
		end := width
		for end > 0 {
			if !cellIsDefaultBlank(v.convertCell(v.emu.CellAt(end-1, y))) {
				break
			}
			end--
		}
		rowIndex := y
		rowEnd := end
		// 中文说明：callback 只在读锁内同步使用 cellAt，不能保存闭包或把
		// 它传到锁外；这样 live snapshot 可以直出 compact rows 而不克隆整行。
		visit(rowIndex, rowEnd, func(index int) Cell {
			if index < 0 || index >= rowEnd {
				return Cell{}
			}
			return v.convertCell(v.emu.CellAt(index, rowIndex))
		})
	}
	return info
}

func (v *VTerm) ScreenRowCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.emu == nil {
		return 0
	}
	return v.emu.Height()
}

func (v *VTerm) ScreenRow(y int) []Cell {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return cloneCellSlice(v.screenRowViewLocked(y))
}

// ScreenRowView returns a read-only view of the current screen row.
// The returned slice is invalidated by the next write, resize, or snapshot load.
func (v *VTerm) ScreenRowView(y int) []Cell {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.screenRowViewLocked(y)
}

func (v *VTerm) UsedScreenRow(y int) []Cell {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return cloneCellSlice(v.screenRowUsedViewLocked(y))
}

func (v *VTerm) Size() (int, int) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.emu.Width(), v.emu.Height()
}

func (v *VTerm) ScrollbackContent() [][]Cell {
	v.mu.RLock()
	defer v.mu.RUnlock()
	rows := v.scrollbackRowsLocked()
	out := make([][]Cell, len(rows))
	for i, row := range rows {
		out[i] = cloneCellSlice(row)
	}
	return out
}

func (v *VTerm) SurfaceSnapshot() SurfaceSnapshot {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.emu == nil {
		return SurfaceSnapshot{}
	}
	cols := v.emu.Width()
	rows := v.emu.Height()
	scrollbackCount := v.scrollbackRowCountLocked()
	scrollback := make([][]Cell, scrollbackCount)
	for y := 0; y < scrollbackCount; y++ {
		scrollback[y] = cloneCellSlice(v.scrollbackRowViewLocked(y))
	}
	screen := make([][]Cell, rows)
	for y := 0; y < rows; y++ {
		screen[y] = cloneCellSlice(v.screenRowViewLocked(y))
	}
	scrollbackWrapped := make([]bool, scrollbackCount)
	for y := 0; y < scrollbackCount; y++ {
		scrollbackWrapped[y] = v.scrollbackRowWrappedAtLocked(y)
	}
	screenWrapped := make([]bool, rows)
	for y := 0; y < rows; y++ {
		screenWrapped[y] = v.screenRowWrappedAtLocked(y)
	}
	return SurfaceSnapshot{
		Cols:                 cols,
		Rows:                 rows,
		Scrollback:           scrollback,
		ScrollbackTimestamps: normalizeTimeSlice(v.scrollbackTimestamps, scrollbackCount),
		ScrollbackRowKinds:   normalizeStringSlice(v.scrollbackRowKinds, scrollbackCount),
		ScrollbackWrapped:    scrollbackWrapped,
		ScrollbackOwnership:  normalizeStringSlice(v.scrollbackOwnership, scrollbackCount),
		Screen: ScreenData{
			Cells:             screen,
			IsAlternateScreen: v.emu.IsAltScreen(),
		},
		ScreenTimestamps: normalizeTimeSlice(v.screenTimestamps, rows),
		ScreenRowKinds:   normalizeStringSlice(v.screenRowKinds, rows),
		ScreenWrapped:    screenWrapped,
		ScreenOwnership:  normalizeStringSlice(v.screenOwnership, rows),
		Cursor:           v.cursor,
		Modes:            v.modes,
	}
}

func (v *VTerm) ScrollbackRowCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.emu == nil {
		return 0
	}
	return v.emu.ScrollbackLen()
}

func (v *VTerm) ScrollbackRow(y int) []Cell {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return cloneCellSlice(v.scrollbackRowViewLocked(y))
}

// ScrollbackRowView returns a read-only view of the current scrollback row.
// The returned slice is invalidated by the next write, resize, or snapshot load.
func (v *VTerm) ScrollbackRowView(y int) []Cell {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.scrollbackRowViewLocked(y)
}

func (v *VTerm) scrollbackRowsLocked() [][]Cell {
	count := v.scrollbackRowCountLocked()
	rows := make([][]Cell, 0, count)
	for y := 0; y < count; y++ {
		rows = append(rows, v.scrollbackRowViewLocked(y))
	}
	return rows
}

func (v *VTerm) ScreenTimestamps() []time.Time {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return cloneTimeSlice(v.screenTimestamps)
}

func (v *VTerm) ScrollbackTimestamps() []time.Time {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return cloneTimeSlice(v.scrollbackTimestamps)
}

func (v *VTerm) ScreenRowTimestampAt(y int) time.Time {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return timeAt(v.screenTimestamps, y)
}

func (v *VTerm) ScrollbackRowTimestampAt(y int) time.Time {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return timeAt(v.scrollbackTimestamps, y)
}

func (v *VTerm) ScreenRowKinds() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return cloneStringSlice(v.screenRowKinds)
}

func (v *VTerm) ScrollbackRowKinds() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return cloneStringSlice(v.scrollbackRowKinds)
}

func (v *VTerm) ScreenOwnership() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return cloneStringSlice(v.screenOwnership)
}

func (v *VTerm) ScrollbackOwnership() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return cloneStringSlice(v.scrollbackOwnership)
}

func (v *VTerm) ScreenRowKindAt(y int) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return stringAt(v.screenRowKinds, y)
}

func (v *VTerm) ScrollbackRowKindAt(y int) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return stringAt(v.scrollbackRowKinds, y)
}

func (v *VTerm) ScreenRowOwnershipAt(y int) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return stringAt(v.screenOwnership, y)
}

func (v *VTerm) ScrollbackRowOwnershipAt(y int) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return stringAt(v.scrollbackOwnership, y)
}

func (v *VTerm) capturePrimaryMetadataOnAltEnter(beforeAlt bool, afterAlt bool, timestamps []time.Time, ownership []string) {
	if !beforeAlt && afterAlt {
		v.primarySavedTimestamps = cloneTimeSlice(timestamps)
		v.primarySavedOwnership = cloneStringSlice(ownership)
	}
}

func (v *VTerm) restorePrimaryMetadataOnAltExit(beforeAlt bool, afterAlt bool, height int) {
	if !beforeAlt || afterAlt {
		return
	}
	v.screenTimestamps = normalizeTimeSlice(v.primarySavedTimestamps, height)
	v.screenOwnership = normalizeStringSlice(v.primarySavedOwnership, height)
	v.primarySavedTimestamps = nil
	v.primarySavedOwnership = nil
}

// MarkCurrentScreenHistoryPersisted records that core has committed the current
// visible timeline through an explicit lifecycle operation. The cells stay in
// the emulator for presentation and restart context, while future semantic
// deltas retain enough ownership to avoid committing them twice.
func (v *VTerm) MarkCurrentScreenHistoryPersisted() {
	if v == nil {
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	height := len(v.screenOwnership)
	if v.emu != nil {
		height = v.emu.Height()
	}
	v.screenOwnership = normalizeStringSlice(v.screenOwnership, height)
	for row := range v.screenOwnership {
		fingerprint := v.screenRowFingerprintLocked(row)
		if !rowFingerprintIsBlank(fingerprint) || row < v.cursor.Row || row == v.cursor.Row && v.cursor.Col > 0 {
			v.screenOwnership[row] = HistoryPersistedOwnership
		}
	}
	if v.emu != nil && v.emu.IsAltScreen() {
		height := v.emu.PrimaryHeight()
		v.primarySavedOwnership = normalizeStringSlice(v.primarySavedOwnership, height)
		for row := range v.primarySavedOwnership {
			v.primarySavedOwnership[row] = HistoryPersistedOwnership
		}
	}
}

// PrimarySavedScreenRows 返回 primary 屏当前行（used 宽度裁剪）、软换行标志与时间，
// 与是否处于 alt screen 无关。linehist 无限历史在 alt 期间用它投影"被 alt
// 覆盖但仍未滚出"的主屏时间线尾部；它是 live 读投影，不是第二份 history truth。
func (v *VTerm) PrimarySavedScreenRows() ([][]Cell, []bool, []time.Time) {
	rows, wrapped, timestamps, _ := v.PrimarySavedScreenRowsWithOwnership()
	return rows, wrapped, timestamps
}

// PrimarySavedScreenRowsWithOwnership extends PrimarySavedScreenRows with the
// lifecycle ownership used by the single-emulator history projection.
func (v *VTerm) PrimarySavedScreenRowsWithOwnership() ([][]Cell, []bool, []time.Time, []string) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.emu == nil {
		return nil, nil, nil, nil
	}
	height := v.emu.PrimaryHeight()
	if height <= 0 {
		return nil, nil, nil, nil
	}
	rows := make([][]Cell, height)
	wrapped := make([]bool, height)
	timestamps := v.screenTimestamps
	if v.emu.IsAltScreen() {
		timestamps = v.primarySavedTimestamps
	}
	for y := 0; y < height; y++ {
		line := v.emu.PrimaryLine(y)
		if len(line) > 0 {
			cells := make([]Cell, len(line))
			for i := range line {
				cells[i] = v.convertCell(&line[i])
			}
			rows[y] = cells
		}
		wrapped[y] = v.emu.PrimaryLineWrapped(y)
	}
	ownership := v.screenOwnership
	if v.emu.IsAltScreen() {
		ownership = v.primarySavedOwnership
	}
	return rows, wrapped, normalizeTimeSlice(timestamps, height), normalizeStringSlice(ownership, height)
}

func (v *VTerm) ScreenWrapped() []bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	height := v.screenRowCountLocked()
	if height <= 0 {
		return nil
	}
	out := make([]bool, height)
	for row := 0; row < height; row++ {
		out[row] = v.screenRowWrappedAtLocked(row)
	}
	return out
}

func (v *VTerm) ScrollbackWrapped() []bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	count := v.scrollbackRowCountLocked()
	if count <= 0 {
		return nil
	}
	out := make([]bool, count)
	for row := 0; row < count; row++ {
		out[row] = v.scrollbackRowWrappedAtLocked(row)
	}
	return out
}

func (v *VTerm) ScreenRowWrappedAt(y int) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.screenRowWrappedAtLocked(y)
}

func (v *VTerm) ScrollbackRowWrappedAt(y int) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.scrollbackRowWrappedAtLocked(y)
}

func (v *VTerm) RowVisualHash(rowIndex int) uint64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.emu == nil || rowIndex < 0 {
		return 0
	}
	scrollbackRows := v.emu.ScrollbackLen()
	if rowIndex < scrollbackRows {
		return rowFingerprintVisualHash(v.scrollbackRowFingerprintLocked(rowIndex))
	}
	rowIndex -= scrollbackRows
	if rowIndex < 0 || rowIndex >= v.emu.Height() {
		return 0
	}
	if len(v.screenFingerprintCache) == v.emu.Height() {
		return rowFingerprintVisualHash(v.screenFingerprintCache[rowIndex])
	}
	return rowFingerprintVisualHash(v.screenRowFingerprintLocked(rowIndex))
}

// ScreenVisualHashes returns one compact visual fingerprint per current screen
// row. The returned slice is detached from VTerm's incremental cache.
func (v *VTerm) ScreenVisualHashes() []uint64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.emu == nil {
		return nil
	}
	height := v.emu.Height()
	rows := make([]uint64, height)
	if len(v.screenFingerprintCache) == height {
		for row, fingerprint := range v.screenFingerprintCache {
			rows[row] = rowFingerprintVisualHash(fingerprint)
		}
		return rows
	}
	for row := 0; row < height; row++ {
		rows[row] = rowFingerprintVisualHash(v.screenRowFingerprintLocked(row))
	}
	return rows
}

// VisualRowHash returns the same visual fingerprint shape used by
// ScreenVisualHashes for a detached presentation row.
func VisualRowHash(cells []Cell, width int) uint64 {
	if width < 0 {
		width = 0
	}
	fingerprint := rowFingerprint{hash: rowFingerprintOffset64, blank: true}
	hashUint64(&fingerprint.hash, uint64(width))
	hashBool(&fingerprint.hash, false)
	for col := 0; col < width; col++ {
		cell := Cell{}
		if col < len(cells) {
			cell = cells[col]
		}
		if !hashVTermCellFingerprint(&fingerprint.hash, cell) {
			fingerprint.blank = false
		}
	}
	return rowFingerprintVisualHash(fingerprint)
}

func (v *VTerm) CursorState() CursorState {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.cursor
}

func (v *VTerm) Modes() TerminalModes {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.modes
}

func (v *VTerm) IsAltScreen() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.emu.IsAltScreen()
}

func (v *VTerm) EncodeReplay(scrollbackLimit int) []byte {
	v.mu.RLock()
	defer v.mu.RUnlock()

	scrollback := v.scrollbackRowsLocked()
	if scrollbackLimit > 0 && len(scrollback) > scrollbackLimit {
		scrollback = scrollback[len(scrollback)-scrollbackLimit:]
	}
	return encodeTerminalReplay(scrollback, v.usedScreenRowsLocked(), v.cursor, v.modes)
}

func (v *VTerm) EncodeHistoryReplay(beforeOffset int, limit int) ([]byte, int, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	scrollback := v.scrollbackRowsLocked()
	total := len(scrollback)
	if total == 0 {
		return nil, 0, false
	}
	if limit <= 0 {
		limit = 100
	}
	if beforeOffset < 0 {
		beforeOffset = 0
	}
	if beforeOffset > total {
		beforeOffset = total
	}
	end := total - beforeOffset
	if end < 0 {
		end = 0
	}
	start := end - limit
	if start < 0 {
		start = 0
	}
	rows := scrollback[start:end]
	if len(rows) == 0 {
		return nil, 0, false
	}
	var b strings.Builder
	writeSequentialRows(&b, rows)
	return []byte(b.String()), len(rows), start > 0
}

func EncodeHistoryRowsReplay(rows [][]Cell) []byte {
	return EncodeHistoryRowsReplayWithWrapped(rows, nil)
}

func EncodeHistoryRowsReplayWithWrapped(rows [][]Cell, wrapped []bool) []byte {
	if len(rows) == 0 {
		return nil
	}
	var b strings.Builder
	writeSequentialRowsWithWrapped(&b, rows, wrapped)
	return []byte(b.String())
}

func (v *VTerm) setMode(mode ansi.Mode, enabled bool) {
	switch mode {
	case ansi.ModeCursorKeys:
		v.modes.ApplicationCursor = enabled
	case modeAlternateScroll:
		v.modes.AlternateScroll = enabled
	case ansi.ModeMouseX10:
		v.mouseMode.x10 = enabled
		v.updateMouseTrackingLocked()
	case ansi.ModeMouseNormal:
		v.mouseMode.normal = enabled
		v.updateMouseTrackingLocked()
	case ansi.ModeMouseHighlight:
		v.mouseMode.highlight = enabled
		v.updateMouseTrackingLocked()
	case ansi.ModeMouseButtonEvent:
		v.mouseMode.buttonEvent = enabled
		v.updateMouseTrackingLocked()
	case ansi.ModeMouseAnyEvent:
		v.mouseMode.anyEvent = enabled
		v.updateMouseTrackingLocked()
	case ansi.ModeMouseExtSgr:
		v.mouseMode.sgr = enabled
		v.updateMouseTrackingLocked()
	case ansi.ModeNumericKeypad:
		// x/vt uses "numeric keypad" mode for keypad application mode.
		// Keep this for future input translation support if needed.
	case ansi.ModeBracketedPaste:
		v.modes.BracketedPaste = enabled
	case ansi.ModeAutoWrap:
		v.modes.AutoWrap = enabled
	case ansi.ModeSynchronizedOutput:
		v.modes.SynchronizedOutput = enabled
	}
}

func (v *VTerm) updateMouseTrackingLocked() {
	v.syncMouseModesLocked()
}

func (v *VTerm) syncMouseModesLocked() {
	v.modes.MouseX10 = v.mouseMode.x10
	v.modes.MouseNormal = v.mouseMode.normal
	v.modes.MouseButtonEvent = v.mouseMode.buttonEvent
	v.modes.MouseAnyEvent = v.mouseMode.anyEvent
	v.modes.MouseSGR = v.mouseMode.sgr
	v.modes.MouseTracking = v.mouseMode.x10 ||
		v.mouseMode.normal ||
		v.mouseMode.highlight ||
		v.mouseMode.buttonEvent ||
		v.mouseMode.anyEvent
}

func (v *VTerm) loadMouseModesLocked(modes TerminalModes) {
	v.mouseMode = mouseModeState{
		x10:         modes.MouseX10,
		normal:      modes.MouseNormal,
		buttonEvent: modes.MouseButtonEvent,
		anyEvent:    modes.MouseAnyEvent,
		sgr:         modes.MouseSGR,
	}
	if !v.mouseMode.x10 && !v.mouseMode.normal && !v.mouseMode.buttonEvent && !v.mouseMode.anyEvent && modes.MouseTracking {
		// Older snapshots only persisted the aggregate tracking bit. Preserve the
		// previous compatibility behavior by treating that as button-event mode
		// until explicit protocol fields are available.
		v.mouseMode.buttonEvent = true
	}
	v.syncMouseModesLocked()
}

func (v *VTerm) RenderLines() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	rendered := v.emu.Render()
	if rendered == "" {
		return nil
	}
	return strings.Split(rendered, "\n")
}

func (v *VTerm) SendKey(key uv.KeyEvent) {
	v.emu.SendKey(key)
}

func (v *VTerm) SendText(text string) {
	v.emu.SendText(text)
}

func uvCell(cell Cell) *uv.Cell {
	if cell.Content == "" && cell.Width == 0 {
		// 中文说明：这是宽字符的续位占位符，不是普通空格。恢复快照时必须原样保留，
		// 否则后续增量写入会把已经正确的中日韩宽字符行重新串成“字符 + 一堆空格”。
		return &uv.Cell{}
	}
	c := &uv.Cell{
		Content: cell.Content,
		Width:   cell.Width,
		Link:    uv.Link{URL: cell.LinkURL, Params: cell.LinkParams},
	}
	if c.Content == "" {
		c.Content = " "
	}
	if c.Width <= 0 {
		c.Width = 1
	}
	if cell.Style.FG != "" {
		c.Style.Fg = decodeTerminalColor(cell.Style.FG)
	}
	if cell.Style.BG != "" {
		c.Style.Bg = decodeTerminalColor(cell.Style.BG)
	}
	if cell.Style.Bold {
		c.Style.Attrs |= uv.AttrBold
	}
	if cell.Style.Italic {
		c.Style.Attrs |= uv.AttrItalic
	}
	if cell.Style.Blink {
		c.Style.Attrs |= uv.AttrBlink
	}
	if cell.Style.Reverse {
		c.Style.Attrs |= uv.AttrReverse
	}
	if cell.Style.Strikethrough {
		c.Style.Attrs |= uv.AttrStrikethrough
	}
	if cell.Style.Underline {
		c.Style.Underline = uv.UnderlineSingle
	}
	return c
}

func uvLine(row []Cell) uv.Line {
	line := make(uv.Line, 0, len(row))
	for _, cell := range row {
		line = append(line, *uvCell(cell))
	}
	return line
}

func cellIsBlank(cell Cell) bool {
	return strings.TrimSpace(cell.Content) == "" &&
		cell.Style == (CellStyle{}) &&
		cell.LinkURL == "" &&
		cell.LinkParams == "" &&
		cell.Width <= 1
}

func cellIsDefaultBlank(cell Cell) bool {
	return (cell.Content == "" || cell.Content == " ") &&
		cell.Style == (CellStyle{}) &&
		cell.LinkURL == "" &&
		cell.LinkParams == "" &&
		cell.Width <= 1
}

func snapshotRowUsedWidth(row []Cell, width int) int {
	limit := minInt(len(row), width)
	for i := 0; i < limit; i++ {
		if !cellIsBlank(row[i]) {
			return limit
		}
	}
	return 0
}

func (v *VTerm) disableEmulatorScrollbackLocked() {
	v.sbSize = 0
	if v.emu != nil {
		v.emu.DisableScrollback()
	}
	v.scrollbackTimestamps = nil
	v.scrollbackRowKinds = nil
	v.scrollbackOwnership = nil
	v.scrollbackRowCache = nil
}

func uvBlankCell() *uv.Cell {
	return &uv.Cell{Content: " ", Width: 1}
}

func encodeScreenSnapshot(rows [][]Cell) []byte {
	var b strings.Builder
	for y, row := range rows {
		for x, cell := range row {
			if cell.Content == "" && cell.Width == 0 {
				// 中文说明：续位列本身不再重复写字符，由前一个宽字符占满即可。
				continue
			}
			content := cell.Content
			if content == "" {
				content = " "
			}
			b.WriteString(fmt.Sprintf("\x1b[%d;%dH", y+1, x+1))
			b.WriteString(cellANSI(cell))
			b.WriteString(content)
		}
	}
	if b.Len() == 0 {
		return nil
	}
	b.WriteString(resetCellStyleANSI())
	return []byte(b.String())
}

func encodeTerminalReplay(scrollback, screen [][]Cell, cursor CursorState, modes TerminalModes) []byte {
	var b strings.Builder

	if !modes.AlternateScreen && len(scrollback) > 0 {
		writeSequentialRows(&b, scrollback)
		b.WriteString("\r\n")
		visibleRows := len(screen)
		if visibleRows < 1 {
			visibleRows = 1
		}
		for i := 0; i < visibleRows-1; i++ {
			b.WriteByte('\n')
		}
		b.WriteString(resetCellStyleANSI())
	}

	if modes.AlternateScreen {
		b.WriteString("\x1b[?1049h")
	}
	b.WriteString("\x1b[H\x1b[2J\x1b[H")
	b.Write(encodeScreenSnapshot(screen))
	writeTerminalModesANSI(&b, modes)
	writeCursorShapeANSI(&b, cursor)
	if cursor.Row >= 0 && cursor.Col >= 0 {
		fmt.Fprintf(&b, "\x1b[%d;%dH", cursor.Row+1, cursor.Col+1)
	}
	if cursor.Visible {
		b.WriteString("\x1b[?25h")
	} else {
		b.WriteString("\x1b[?25l")
	}

	return []byte(b.String())
}

func writeSequentialRows(b *strings.Builder, rows [][]Cell) {
	writeSequentialRowsWithWrapped(b, rows, nil)
}

func writeSequentialRowsWithWrapped(b *strings.Builder, rows [][]Cell, wrapped []bool) {
	if b == nil || len(rows) == 0 {
		return
	}
	style := CellStyle{}
	for i, row := range rows {
		style = writeSequentialRow(b, row, style)
		if i < len(rows)-1 && !boolAt(wrapped, i) {
			b.WriteString("\r\n")
		}
	}
	if style != (CellStyle{}) {
		b.WriteString(resetCellStyleANSI())
	}
}

func writeSequentialRow(b *strings.Builder, row []Cell, currentStyle CellStyle) CellStyle {
	if b == nil {
		return currentStyle
	}
	for i := 0; i < len(row); i++ {
		cell := row[i]
		if cell.Content == "" && cell.Width == 0 {
			continue
		}
		content := cell.Content
		if content == "" {
			content = " "
		}
		style := cell.Style
		style.LinkURL = cell.LinkURL
		style.LinkParams = cell.LinkParams
		if style != currentStyle {
			b.WriteString(cellStyleANSI(style))
			currentStyle = style
		}
		b.WriteString(content)
	}
	return currentStyle
}

func writeTerminalModesANSI(b *strings.Builder, modes TerminalModes) {
	if b == nil {
		return
	}
	writePrivateModeANSI(b, 1, modes.ApplicationCursor)
	writePrivateModeANSI(b, 7, modes.AutoWrap)
	writePrivateModeANSI(b, 1007, modes.AlternateScroll)
	writePrivateModeANSI(b, 2004, modes.BracketedPaste)
	writePrivateModeANSI(b, 2026, modes.SynchronizedOutput)

	mouseX10 := modes.MouseX10
	mouseNormal := modes.MouseNormal
	mouseButton := modes.MouseButtonEvent
	mouseAny := modes.MouseAnyEvent
	if modes.MouseTracking && !mouseX10 && !mouseNormal && !mouseButton && !mouseAny {
		mouseNormal = true
	}
	writePrivateModeANSI(b, 9, mouseX10)
	writePrivateModeANSI(b, 1000, mouseNormal)
	writePrivateModeANSI(b, 1002, mouseButton)
	writePrivateModeANSI(b, 1003, mouseAny)
	writePrivateModeANSI(b, 1005, false)
	writePrivateModeANSI(b, 1006, modes.MouseSGR)
}

func writeCursorShapeANSI(b *strings.Builder, cursor CursorState) {
	if b == nil {
		return
	}
	code := 0
	switch cursor.Shape {
	case CursorUnderline:
		if cursor.Blink {
			code = 3
		} else {
			code = 4
		}
	case CursorBar:
		if cursor.Blink {
			code = 5
		} else {
			code = 6
		}
	case CursorBlock:
		if cursor.Blink {
			code = 1
		} else {
			code = 2
		}
	}
	if code > 0 {
		fmt.Fprintf(b, "\x1b[%d q", code)
	}
}

func writePrivateModeANSI(b *strings.Builder, mode int, enabled bool) {
	if enabled {
		fmt.Fprintf(b, "\x1b[?%dh", mode)
		return
	}
	fmt.Fprintf(b, "\x1b[?%dl", mode)
}

func cellStyleANSI(style CellStyle) string {
	var b strings.Builder
	b.WriteString(resetHyperlinkANSI())
	b.WriteString("\x1b[0")
	if style.Bold {
		b.WriteString(";1")
	}
	if style.Italic {
		b.WriteString(";3")
	}
	if style.Underline {
		b.WriteString(";4")
	}
	if style.Blink {
		b.WriteString(";5")
	}
	if style.Reverse {
		b.WriteString(";7")
	}
	if style.Strikethrough {
		b.WriteString(";9")
	}
	writeCellStyleColor(&b, style.FG, true)
	writeCellStyleColor(&b, style.BG, false)
	b.WriteByte('m')
	if style.LinkURL != "" || style.LinkParams != "" {
		b.WriteString(setHyperlinkANSI(style.LinkURL, style.LinkParams))
	}
	return b.String()
}

func resetCellStyleANSI() string {
	return resetHyperlinkANSI() + "\x1b[0m"
}

func cellANSI(cell Cell) string {
	style := cell.Style
	style.LinkURL = cell.LinkURL
	style.LinkParams = cell.LinkParams
	return cellStyleANSI(style)
}

func setHyperlinkANSI(linkURL, linkParams string) string {
	return "\x1b]8;" + sanitizeHyperlinkANSI(linkParams) + ";" + sanitizeHyperlinkANSI(linkURL) + "\x07"
}

func resetHyperlinkANSI() string {
	return "\x1b]8;;\x07"
}

func sanitizeHyperlinkANSI(value string) string {
	if value == "" {
		return ""
	}
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
}

func writeCellStyleColor(b *strings.Builder, value string, foreground bool) {
	if b == nil || strings.TrimSpace(value) == "" {
		return
	}
	switch c := decodeTerminalColor(value).(type) {
	case ansi.BasicColor:
		code := int(c)
		if code < 8 {
			if foreground {
				b.WriteString(fmt.Sprintf(";3%d", code))
			} else {
				b.WriteString(fmt.Sprintf(";4%d", code))
			}
			return
		}
		if foreground {
			b.WriteString(fmt.Sprintf(";9%d", code-8))
		} else {
			b.WriteString(fmt.Sprintf(";10%d", code-8))
		}
	case ansi.IndexedColor:
		if foreground {
			b.WriteString(fmt.Sprintf(";38;5;%d", int(c)))
		} else {
			b.WriteString(fmt.Sprintf(";48;5;%d", int(c)))
		}
	case ansi.RGBColor:
		if foreground {
			b.WriteString(fmt.Sprintf(";38;2;%d;%d;%d", c.R, c.G, c.B))
		} else {
			b.WriteString(fmt.Sprintf(";48;2;%d;%d;%d", c.R, c.G, c.B))
		}
	default:
		if rgb := ansi.XParseColor(value); rgb != nil {
			r, g, bl, _ := rgb.RGBA()
			if foreground {
				b.WriteString(fmt.Sprintf(";38;2;%d;%d;%d", uint8(r>>8), uint8(g>>8), uint8(bl>>8)))
			} else {
				b.WriteString(fmt.Sprintf(";48;2;%d;%d;%d", uint8(r>>8), uint8(g>>8), uint8(bl>>8)))
			}
		}
	}
}

func (v *VTerm) Paste(text string) {
	v.emu.Paste(text)
}

func (v *VTerm) SetTitleHandler(handler TitleHandler) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.onTitle = handler
}

func (v *VTerm) SetWorkingDirectoryHandler(handler WorkingDirectoryHandler) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.onCWD = handler
}

func (v *VTerm) SetDefaultColors(fg, bg string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.defaultFG = normalizeColorString(fg)
	v.defaultBG = normalizeColorString(bg)
	v.applyDefaultColorsToEmulator(v.emu)
}

func (v *VTerm) DefaultColors() (fg, bg string) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.defaultFG, v.defaultBG
}

func (v *VTerm) SetIndexedColor(index int, value string) {
	if index < 0 || index > 255 {
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	value = normalizeColorString(value)
	if value == "" {
		if v.palette != nil {
			delete(v.palette, index)
		}
	} else {
		if v.palette == nil {
			v.palette = make(map[int]string)
		}
		v.palette[index] = value
	}
	if v.emu != nil {
		v.emu.SetIndexedColor(index, ansi.XParseColor(value))
	}
}

func (v *VTerm) IndexedColor(index int) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.palette == nil {
		return ""
	}
	return v.palette[index]
}

func (v *VTerm) convertCell(cell *uv.Cell) Cell {
	if cell == nil {
		return Cell{}
	}
	return Cell{
		Content:    cell.Content,
		Width:      cell.Width,
		Style:      v.convertStyle(cell.Style),
		LinkURL:    cell.Link.URL,
		LinkParams: cell.Link.Params,
	}
}

func (v *VTerm) convertStyle(style uv.Style) CellStyle {
	out := CellStyle{}
	if style.Fg != nil {
		out.FG = v.resolveColorString(style.Fg)
	}
	if style.Bg != nil {
		out.BG = v.resolveColorString(style.Bg)
	}
	out.Bold = style.Attrs&uv.AttrBold != 0
	out.Italic = style.Attrs&uv.AttrItalic != 0
	out.Underline = style.Underline != 0
	out.Blink = style.Attrs&uv.AttrBlink != 0
	out.Reverse = style.Attrs&uv.AttrReverse != 0
	out.Strikethrough = style.Attrs&uv.AttrStrikethrough != 0
	return out
}

func (v *VTerm) resolveColorString(c color.Color) string {
	if c == nil {
		return ""
	}
	switch value := c.(type) {
	case ansi.BasicColor:
		return encodeBasicColor(value)
	case ansi.IndexedColor:
		return encodeIndexedColor(value)
	}
	return colorToString(c)
}

func encodeBasicColor(c ansi.BasicColor) string {
	index := int(c)
	if index >= 0 && index < len(basicColorStrings) {
		return basicColorStrings[index]
	}
	return "ansi:" + strconv.Itoa(index)
}

func encodeIndexedColor(c ansi.IndexedColor) string {
	index := int(c)
	if index >= 0 && index < len(indexedColorStrings) {
		return indexedColorStrings[index]
	}
	return "idx:" + strconv.Itoa(index)
}

func buildIndexedColorStrings(prefix string, count int) []string {
	values := make([]string, count)
	for i := range values {
		values[i] = prefix + strconv.Itoa(i)
	}
	return values
}

func decodeTerminalColor(value string) color.Color {
	value = strings.TrimSpace(value)
	switch {
	case strings.HasPrefix(value, "ansi:"):
		index, err := strconv.Atoi(strings.TrimPrefix(value, "ansi:"))
		if err == nil && index >= 0 && index <= 15 {
			return ansi.BasicColor(index)
		}
	case strings.HasPrefix(value, "idx:"):
		index, err := strconv.Atoi(strings.TrimPrefix(value, "idx:"))
		if err == nil && index >= 0 && index <= 255 {
			return ansi.IndexedColor(index)
		}
	}
	return ansi.XParseColor(value)
}

func colorToString(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

func normalizeColorString(value string) string {
	if rgb := ansi.XParseColor(strings.TrimSpace(value)); rgb != nil {
		return colorToString(rgb)
	}
	return ""
}

func (v *VTerm) applyDefaultColorsToEmulator(emu *charmvt.SafeEmulator) {
	if emu == nil {
		return
	}
	emu.SetDefaultForegroundColor(ansi.XParseColor(v.defaultFG))
	emu.SetDefaultBackgroundColor(ansi.XParseColor(v.defaultBG))
	for index, value := range v.palette {
		emu.SetIndexedColor(index, ansi.XParseColor(value))
	}
}

func (v *VTerm) screenRowsLocked() [][]Cell {
	height := v.screenRowCountLocked()
	rows := make([][]Cell, height)
	for y := 0; y < height; y++ {
		rows[y] = v.screenRowViewLocked(y)
	}
	return rows
}

func (v *VTerm) usedScreenRowsLocked() [][]Cell {
	height := v.screenRowCountLocked()
	rows := make([][]Cell, height)
	for y := 0; y < height; y++ {
		rows[y] = v.screenRowUsedViewLocked(y)
	}
	return rows
}

func (v *VTerm) screenWrappedLocked(height int) []bool {
	if height <= 0 {
		return nil
	}
	out := make([]bool, height)
	for y := 0; y < height; y++ {
		out[y] = v.screenRowWrappedAtLocked(y)
	}
	return out
}

func (v *VTerm) screenRowCountLocked() int {
	if v.emu == nil {
		return 0
	}
	return v.emu.Height()
}

func (v *VTerm) scrollbackRowCountLocked() int {
	if v.emu == nil {
		return 0
	}
	return v.emu.ScrollbackLen()
}

func (v *VTerm) screenRowViewLocked(y int) []Cell {
	if v.emu == nil || y < 0 || y >= v.emu.Height() {
		return nil
	}
	if len(v.screenRowCache) != v.emu.Height() {
		v.invalidateRowCachesLocked()
	}
	if cached := v.screenRowCache[y]; len(cached) == v.emu.Width() {
		return cached
	}
	width := v.emu.Width()
	row := make([]Cell, width)
	for x := 0; x < width; x++ {
		row[x] = v.convertCell(v.emu.CellAt(x, y))
	}
	v.screenRowCache[y] = row
	return row
}

func (v *VTerm) screenRowUsedViewLocked(y int) []Cell {
	if v.emu == nil || y < 0 || y >= v.emu.Height() {
		return nil
	}
	used := v.emu.ScreenLineUsed(y)
	if used <= 0 {
		return v.screenRowViewLocked(y)
	}
	if width := v.emu.Width(); used > width {
		used = width
	}
	if len(v.screenRowCache) != v.emu.Height() {
		v.invalidateRowCachesLocked()
	}
	if cached := v.screenRowCache[y]; len(cached) >= used {
		return cached[:used]
	}
	row := make([]Cell, used)
	for x := 0; x < used; x++ {
		row[x] = v.convertCell(v.emu.CellAt(x, y))
	}
	v.screenRowCache[y] = row
	return row
}

func (v *VTerm) trimmedScreenRowCellsLocked(y int) []Cell {
	if v.emu == nil || y < 0 || y >= v.emu.Height() {
		return nil
	}
	width := v.emu.Width()
	end := width
	for end > 0 {
		if !cellIsDefaultBlank(v.convertCell(v.emu.CellAt(end-1, y))) {
			break
		}
		end--
	}
	if end <= 0 {
		return nil
	}
	row := make([]Cell, end)
	for x := 0; x < end; x++ {
		row[x] = v.convertCell(v.emu.CellAt(x, y))
	}
	return row
}

func (v *VTerm) expandedScreenRowSpanLocked(y, start, end int) (int, []Cell) {
	row := v.screenRowViewLocked(y)
	if len(row) == 0 {
		return 0, nil
	}
	if start < 0 {
		start = 0
	}
	if end > len(row) {
		end = len(row)
	}
	if start >= end {
		return 0, nil
	}
	for start > 0 && isWideContinuationCell(row[start]) {
		start--
	}
	for col := start; col < end && col < len(row); col++ {
		cell := row[col]
		if cell.Width > 1 {
			if nextEnd := col + cell.Width; nextEnd > end {
				end = nextEnd
				if end > len(row) {
					end = len(row)
				}
			}
		}
	}
	return start, cloneCellSlice(row[start:end])
}

func spanDamageCellWidth(cells []uv.Cell) int {
	width := 0
	for _, cell := range cells {
		if cell.Width > 0 {
			width += cell.Width
			continue
		}
		width++
	}
	if width <= 0 {
		return 1
	}
	return width
}

func (v *VTerm) scrollbackRowViewLocked(y int) []Cell {
	if v.emu == nil || y < 0 || y >= v.emu.ScrollbackLen() {
		return nil
	}
	v.ensureScrollbackRowCacheLocked(v.emu.ScrollbackLen())
	if cached := v.scrollbackRowCache[y]; cached != nil {
		return cached
	}
	line := v.emu.ScrollbackLine(y)
	row := make([]Cell, len(line))
	for x := 0; x < len(line); x++ {
		row[x] = v.convertCell(&line[x])
	}
	v.scrollbackRowCache[y] = row
	return row
}

func (v *VTerm) invalidateRowCachesLocked() {
	if v.emu == nil {
		v.screenRowCache = nil
		v.scrollbackRowCache = nil
		return
	}
	height := maxInt(v.emu.Height(), 0)
	if cap(v.screenRowCache) >= height {
		v.screenRowCache = v.screenRowCache[:height]
		clear(v.screenRowCache)
	} else {
		v.screenRowCache = make([][]Cell, height)
	}
	v.scrollbackRowCache = nil
}

func (v *VTerm) invalidateFingerprintCacheLocked() {
	v.screenFingerprintCache = nil
}

func (v *VTerm) ensureScreenFingerprintCacheLocked() {
	if v == nil || v.emu == nil {
		v.screenFingerprintCache = nil
		return
	}
	if len(v.screenFingerprintCache) == v.emu.Height() {
		return
	}
	v.screenFingerprintCache = v.screenRowFingerprintsLocked()
}

func (v *VTerm) clearTouchedRowsLocked() {
	if v == nil || v.emu == nil {
		return
	}
	touched := v.emu.Touched()
	for row := range touched {
		touched[row] = nil
	}
}

func (v *VTerm) consumeTouchedRowsLocked() ([]int, bool) {
	if v == nil || v.emu == nil {
		return nil, false
	}
	touched := v.emu.Touched()
	if touched == nil {
		return nil, false
	}
	rows := make([]int, 0, len(touched))
	for row, line := range touched {
		if line == nil {
			continue
		}
		if line.FirstCell == -1 && line.LastCell == -1 {
			touched[row] = nil
			continue
		}
		rows = append(rows, row)
		touched[row] = nil
	}
	return rows, true
}

func (v *VTerm) ensureScrollbackRowCacheLocked(count int) {
	switch {
	case count <= 0:
		v.scrollbackRowCache = nil
	case cap(v.scrollbackRowCache) >= count:
		prevLen := len(v.scrollbackRowCache)
		v.scrollbackRowCache = v.scrollbackRowCache[:count]
		if count > prevLen {
			clear(v.scrollbackRowCache[prevLen:])
		}
	default:
		v.scrollbackRowCache = make([][]Cell, count)
	}
}

func (v *VTerm) screenRowFingerprintsLocked() []rowFingerprint {
	height := v.screenRowCountLocked()
	rows := make([]rowFingerprint, height)
	for y := 0; y < height; y++ {
		rows[y] = v.screenRowFingerprintLocked(y)
	}
	return rows
}

func (v *VTerm) screenRowFingerprintLocked(y int) rowFingerprint {
	if v.emu == nil || y < 0 || y >= v.emu.Height() {
		return rowFingerprint{}
	}
	width := v.emu.ScreenLineUsed(y)
	if width <= 0 {
		width = v.emu.Width()
	}
	return v.rowFingerprintLocked(width, v.screenRowWrappedAtLocked(y), func(x int) *uv.Cell {
		return v.emu.CellAt(x, y)
	})
}

func (v *VTerm) scrollbackTailRowFingerprintsLocked(count int) []rowFingerprint {
	if count <= 0 {
		return nil
	}
	total := v.scrollbackRowCountLocked()
	if total <= 0 {
		return nil
	}
	start := maxInt(0, total-count)
	rows := make([]rowFingerprint, 0, total-start)
	for y := start; y < total; y++ {
		rows = append(rows, v.scrollbackRowFingerprintLocked(y))
	}
	return rows
}

func (v *VTerm) scrollbackRowFingerprintLocked(y int) rowFingerprint {
	if v.emu == nil || y < 0 || y >= v.emu.ScrollbackLen() {
		return rowFingerprint{}
	}
	line := v.emu.ScrollbackLine(y)
	width := len(line)
	return v.rowFingerprintLocked(width, v.scrollbackRowWrappedAtLocked(y), func(x int) *uv.Cell {
		if x < 0 || x >= len(line) {
			return nil
		}
		return &line[x]
	})
}

func (v *VTerm) screenRowWrappedAtLocked(y int) bool {
	if v == nil || v.emu == nil || y < 0 || y >= v.emu.Height() {
		return false
	}
	return v.emu.ScreenLineWrapped(y)
}

func (v *VTerm) scrollbackRowWrappedAtLocked(y int) bool {
	if v == nil || v.emu == nil || y < 0 || y >= v.emu.ScrollbackLen() {
		return false
	}
	return v.emu.ScrollbackLineWrapped(y)
}

func (v *VTerm) rowFingerprintLocked(width int, wrapped bool, cellAt func(int) *uv.Cell) rowFingerprint {
	fingerprint := rowFingerprint{
		hash:  rowFingerprintOffset64,
		blank: true,
	}
	hashUint64(&fingerprint.hash, uint64(width))
	hashBool(&fingerprint.hash, wrapped)
	for x := 0; x < width; x++ {
		if !hashCellFingerprint(&fingerprint.hash, cellAt(x)) {
			fingerprint.blank = false
		}
	}
	return fingerprint
}

func hashVTermCellFingerprint(hash *uint64, cell Cell) bool {
	hashString(hash, cell.Content)
	hashUint64(hash, uint64(cell.Width))
	hashString(hash, cell.Style.FG)
	hashString(hash, cell.Style.BG)
	hashBool(hash, cell.Style.Bold)
	hashBool(hash, cell.Style.Italic)
	hashBool(hash, cell.Style.Underline)
	hashBool(hash, cell.Style.Blink)
	hashBool(hash, cell.Style.Reverse)
	hashBool(hash, cell.Style.Strikethrough)
	hashString(hash, cell.LinkURL)
	hashString(hash, cell.LinkParams)

	return strings.TrimSpace(cell.Content) == "" &&
		cell.Style == (CellStyle{}) &&
		cell.LinkURL == "" &&
		cell.LinkParams == "" &&
		cell.Width <= 1
}

func (v *VTerm) reconcileRowMetadataLocked(beforeScreen []rowFingerprint, beforeScreenTimestamps []time.Time, beforeScreenRowKinds []string, beforeScreenOwnership []string, beforeScrollbackLen int, afterScreen []rowFingerprint, now time.Time) rowCacheReconcilePlan {
	if v.emu == nil {
		v.screenTimestamps = nil
		v.scrollbackTimestamps = nil
		v.screenRowKinds = nil
		v.scrollbackRowKinds = nil
		v.screenOwnership = nil
		v.scrollbackOwnership = nil
		return rowCacheReconcilePlan{}
	}
	scrollShift := detectScreenScrollShift(beforeScreen, afterScreen)
	afterScrollbackLen := v.scrollbackRowCountLocked()
	requiredAppends := scrollShift
	if minAppend := afterScrollbackLen - beforeScrollbackLen; minAppend > requiredAppends {
		requiredAppends = minAppend
	}
	appendedRows := v.scrollbackTailRowFingerprintsLocked(requiredAppends)
	preservedFromBefore := 0
	for preservedFromBefore < len(appendedRows) && preservedFromBefore < len(beforeScreen) && preservedFromBefore < len(beforeScreenTimestamps) {
		if beforeScreenTimestamps[preservedFromBefore].IsZero() && rowFingerprintIsBlank(beforeScreen[preservedFromBefore]) {
			break
		}
		if !rowFingerprintsEqual(beforeScreen[preservedFromBefore], appendedRows[preservedFromBefore]) {
			break
		}
		preservedFromBefore++
	}
	for i := 0; i < preservedFromBefore; i++ {
		ts := beforeScreenTimestamps[i]
		if ts.IsZero() && shouldAssignTimestampToRowFingerprint(beforeScreen[i], i, v.cursor.Row) {
			ts = now
		}
		v.scrollbackTimestamps = append(v.scrollbackTimestamps, ts)
		v.scrollbackRowKinds = append(v.scrollbackRowKinds, stringAt(beforeScreenRowKinds, i))
		v.scrollbackOwnership = append(v.scrollbackOwnership, stringAt(beforeScreenOwnership, i))
	}
	for i := preservedFromBefore; i < requiredAppends; i++ {
		v.scrollbackTimestamps = append(v.scrollbackTimestamps, now)
		v.scrollbackRowKinds = append(v.scrollbackRowKinds, "")
		v.scrollbackOwnership = append(v.scrollbackOwnership, "")
	}
	v.alignScrollbackMetadataLocked()
	screenScrollShift := 0
	if preservedFromBefore == 0 && afterScrollbackLen == beforeScrollbackLen {
		screenScrollShift = scrollShift
	}

	oldScreenTimestamps := v.screenTimestamps
	oldScreenRowKinds := v.screenRowKinds
	nextScreenTimestamps := reuseTimeSlice(v.screenTimestampsScratch, len(afterScreen))
	clear(nextScreenTimestamps)
	nextScreenRowKinds := reuseStringSlice(v.screenRowKindsScratch, len(afterScreen))
	clear(nextScreenRowKinds)
	nextScreenOwnership := make([]string, len(afterScreen))
	claimedPersistedRows := make([]bool, len(beforeScreenOwnership))
	for row := range afterScreen {
		mappedRow := row + preservedFromBefore
		if screenScrollShift > 0 {
			mappedRow = row + screenScrollShift
		}
		if mappedRow < len(beforeScreen) && mappedRow < len(beforeScreenTimestamps) && rowFingerprintsEqual(beforeScreen[mappedRow], afterScreen[row]) {
			nextScreenTimestamps[row] = beforeScreenTimestamps[mappedRow]
			nextScreenRowKinds[row] = stringAt(beforeScreenRowKinds, mappedRow)
			nextScreenOwnership[row] = stringAt(beforeScreenOwnership, mappedRow)
			if nextScreenOwnership[row] == HistoryPersistedOwnership && mappedRow < len(claimedPersistedRows) {
				claimedPersistedRows[mappedRow] = true
			}
		}
		if nextScreenTimestamps[row].IsZero() && shouldAssignTimestampToRowFingerprint(afterScreen[row], row, v.cursor.Row) {
			nextScreenTimestamps[row] = now
		}
	}
	// A zero-capacity emulator has no scrollback length signal. If several rows
	// change in one write, fingerprint shift detection can be inconclusive even
	// though an untouched persisted row moved upward. Recover only strict upward
	// moves, one-to-one; same-position rows were handled above and freshly
	// created rows at the bottom cannot inherit this fallback ownership.
	for row := range afterScreen {
		if nextScreenOwnership[row] != "" {
			continue
		}
		for source := row + 1; source < len(beforeScreen); source++ {
			if source >= len(beforeScreenOwnership) || claimedPersistedRows[source] ||
				beforeScreenOwnership[source] != HistoryPersistedOwnership ||
				!rowFingerprintsEqual(beforeScreen[source], afterScreen[row]) {
				continue
			}
			nextScreenOwnership[row] = HistoryPersistedOwnership
			claimedPersistedRows[source] = true
			break
		}
	}
	v.screenTimestamps = nextScreenTimestamps
	v.screenRowKinds = nextScreenRowKinds
	v.screenOwnership = nextScreenOwnership
	if oldScreenTimestamps == nil {
		v.screenTimestampsScratch = nil
	} else {
		v.screenTimestampsScratch = oldScreenTimestamps[:0]
	}
	if oldScreenRowKinds == nil {
		v.screenRowKindsScratch = nil
	} else {
		v.screenRowKindsScratch = oldScreenRowKinds[:0]
	}
	sourceRows := make([]int, requiredAppends)
	for i := range sourceRows {
		sourceRows[i] = -1
	}
	for i := 0; i < preservedFromBefore && i < len(sourceRows); i++ {
		sourceRows[i] = i
	}
	return rowCacheReconcilePlan{
		afterScreen:               afterScreen,
		preservedFromBefore:       preservedFromBefore,
		requiredScrollbackAppends: requiredAppends,
		beforeScrollbackLen:       beforeScrollbackLen,
		screenScrollShift:         screenScrollShift,
		scrollbackSourceRows:      sourceRows,
	}
}

func (v *VTerm) reconcileRowCachesLocked(beforeScreen []rowFingerprint, plan rowCacheReconcilePlan) {
	if v.emu == nil {
		v.screenRowCache = nil
		v.scrollbackRowCache = nil
		return
	}
	oldScreenCache := v.screenRowCache
	oldScrollbackCache := v.scrollbackRowCache
	var nextScreenCache [][]Cell
	if plan.preservedFromBefore == 0 &&
		plan.screenScrollShift == 0 &&
		len(oldScreenCache) == len(plan.afterScreen) &&
		len(beforeScreen) == len(plan.afterScreen) {
		nextScreenCache = oldScreenCache
		for row := range plan.afterScreen {
			if !rowFingerprintsEqual(beforeScreen[row], plan.afterScreen[row]) {
				nextScreenCache[row] = nil
			}
		}
	} else {
		nextScreenCache = make([][]Cell, len(plan.afterScreen))
		for row := range plan.afterScreen {
			mappedRow := row + plan.preservedFromBefore
			if plan.screenScrollShift > 0 {
				mappedRow = row + plan.screenScrollShift
			}
			if mappedRow >= len(beforeScreen) || mappedRow >= len(oldScreenCache) {
				continue
			}
			if !rowFingerprintsEqual(beforeScreen[mappedRow], plan.afterScreen[row]) {
				continue
			}
			nextScreenCache[row] = oldScreenCache[mappedRow]
		}
	}
	v.screenRowCache = nextScreenCache

	afterScrollbackLen := v.scrollbackRowCountLocked()
	if afterScrollbackLen <= 0 {
		v.scrollbackRowCache = nil
		return
	}
	nextScrollbackCache := make([][]Cell, afterScrollbackLen)
	rowsDroppedFromFront := maxInt(0, plan.beforeScrollbackLen+plan.requiredScrollbackAppends-afterScrollbackLen)
	retainedFromOldScrollback := plan.beforeScrollbackLen - rowsDroppedFromFront
	if retainedFromOldScrollback < 0 {
		retainedFromOldScrollback = 0
	}
	if rowsDroppedFromFront < len(oldScrollbackCache) && retainedFromOldScrollback > 0 {
		available := len(oldScrollbackCache) - rowsDroppedFromFront
		if retainedFromOldScrollback > available {
			retainedFromOldScrollback = available
		}
		copy(nextScrollbackCache[:retainedFromOldScrollback], oldScrollbackCache[rowsDroppedFromFront:rowsDroppedFromFront+retainedFromOldScrollback])
	} else {
		retainedFromOldScrollback = 0
	}
	for i := 0; i < plan.preservedFromBefore && retainedFromOldScrollback+i < afterScrollbackLen; i++ {
		if i >= len(oldScreenCache) {
			break
		}
		nextScrollbackCache[retainedFromOldScrollback+i] = oldScreenCache[i]
	}
	v.scrollbackRowCache = nextScrollbackCache
}

func rowCopiesFromReconcilePlan(beforeScreen []rowFingerprint, plan rowCacheReconcilePlan) []RowCopy {
	if len(beforeScreen) != len(plan.afterScreen) {
		return nil
	}
	var copies []RowCopy
	for destinationRow := range plan.afterScreen {
		sourceRow := destinationRow + plan.preservedFromBefore
		if plan.screenScrollShift > 0 {
			sourceRow = destinationRow + plan.screenScrollShift
		}
		if sourceRow == destinationRow || sourceRow < 0 || sourceRow >= len(beforeScreen) {
			continue
		}
		if !rowFingerprintsEqual(beforeScreen[sourceRow], plan.afterScreen[destinationRow]) {
			continue
		}
		if len(copies) > 0 {
			last := &copies[len(copies)-1]
			if last.SourceRow+last.Count == sourceRow && last.DestinationRow+last.Count == destinationRow {
				last.Count++
				continue
			}
		}
		copies = append(copies, RowCopy{SourceRow: sourceRow, DestinationRow: destinationRow, Count: 1})
	}
	return copies
}

func (v *VTerm) semanticControlOpsFromCharmVTDamagesLocked(damages []charmvt.Damage, timestamps []time.Time, rowKinds []string) []DamageOp {
	if len(damages) == 0 {
		return nil
	}
	ops := make([]DamageOp, 0, len(damages))
	tailRows := make(map[int]*CellStyle)
	for _, raw := range damages {
		switch d := raw.(type) {
		case charmvt.TextDamage:
			if len(d.Cells) == 0 && len(d.Runs) == 0 && d.Text == "" {
				continue
			}
			cells, runs := v.textDamagePayloadLocked(d)
			if len(cells) == 0 && len(runs) == 0 {
				continue
			}
			ops = append(ops, DamageOp{
				Code:  ScreenOpWriteSpan,
				Row:   d.Y,
				Col:   maxInt(0, d.X),
				Cells: cells,
				Runs:  runs,
			})
			recordSemanticTailFillForPayload(tailRows, d.X, d.Y, cells, runs)
		case charmvt.ControlDamage:
			if d.Kind == "" {
				continue
			}
			op := DamageOp{
				Code:      ScreenOpControl,
				Control:   d.Kind,
				Row:       d.Y,
				Col:       d.X,
				Mode:      d.Mode,
				Bottom:    d.Bottom,
				ScrollOut: v.scrollbackRowAppendsFromCharmVTDamages(d.ScrollOut, timestamps, rowKinds),
			}
			if d.HasCell {
				op.Cells = uvCellsToVTermDamageCells(v, []uv.Cell{d.Cell})
			}
			if semanticControlMayCarryTailFill(d.Kind) {
				op.TailFill = cloneCellStylePointer(tailRows[d.Y])
			}
			ops = append(ops, op)
			if semanticControlEndsPhysicalRow(d.Kind) {
				delete(tailRows, d.Y)
			}
		case charmvt.ModeDamage:
			ops = append(ops, DamageOp{
				Code:    ScreenOpModes,
				Mode:    d.Mode,
				Private: d.Private,
				Enabled: d.Enabled,
			})
		}
	}
	return ops
}

func (v *VTerm) semanticBoundaryOpsFromCharmVTDamagesLocked(damages []charmvt.Damage) []DamageOp {
	if len(damages) == 0 {
		return nil
	}
	ops := make([]DamageOp, 0, 4)
	for _, raw := range damages {
		switch d := raw.(type) {
		case charmvt.ModeDamage:
			ops = append(ops, DamageOp{
				Code:    ScreenOpModes,
				Mode:    d.Mode,
				Private: d.Private,
				Enabled: d.Enabled,
			})
		case charmvt.ControlDamage:
			// 中文说明：linehist 生产 ingest 不需要每个 CR/LF/Text op；
			// 但 ED3 是 clear-scrollback 软边界，必须保留下传给 core。
			if d.Kind == "ed" && d.Mode == 3 {
				ops = append(ops, DamageOp{
					Code:    ScreenOpControl,
					Control: d.Kind,
					Row:     d.Y,
					Col:     d.X,
					Mode:    d.Mode,
				})
			}
		}
	}
	if len(ops) == 0 {
		return nil
	}
	return ops
}

func recordSemanticTailFillForPayload(rows map[int]*CellStyle, x int, y int, cells []Cell, runs []CellRun) {
	if rows == nil || len(cells) == 0 {
		if len(runs) == 0 {
			return
		}
		var lastStyled *CellStyle
		width := 0
		for _, run := range runs {
			if run.Text == "" {
				continue
			}
			width += len(run.Text)
			if run.Style.BG != "" && run.Style.FG == "" && !run.Style.Bold && !run.Style.Italic && !run.Style.Underline && !run.Style.Blink && !run.Style.Reverse && !run.Style.Strikethrough {
				cloned := CellStyle{BG: run.Style.BG}
				lastStyled = &cloned
				continue
			}
			lastStyled = nil
		}
		if width <= 0 {
			return
		}
		if lastStyled != nil {
			rows[y] = lastStyled
			return
		}
		delete(rows, y)
		return
	}
	width := 0
	var lastStyled *CellStyle
	for _, cell := range cells {
		if cell.Width == 0 {
			continue
		}
		if style, ok := semanticTailFillStyle(cell); ok {
			cloned := style
			lastStyled = &cloned
		} else if cell.Content != "" {
			lastStyled = nil
		}
		width += maxInt(1, cell.Width)
	}
	if width <= 0 {
		return
	}
	// 中文说明：TextDamage 记录的是同一 parser transaction 的真实写入；
	// 尾部背景延伸作为行语义 metadata 传给下游，不能当作文本空格 payload。
	if lastStyled != nil {
		rows[y] = lastStyled
		return
	}
	delete(rows, y)
}

func semanticTailFillStyle(cell Cell) (CellStyle, bool) {
	if cell.Style.BG == "" || cell.LinkURL != "" || cell.LinkParams != "" {
		return CellStyle{}, false
	}
	return CellStyle{BG: cell.Style.BG}, true
}

func semanticControlMayCarryTailFill(kind string) bool {
	switch kind {
	case "lf", "ind", "soft-wrap":
		return true
	default:
		return false
	}
}

func semanticControlEndsPhysicalRow(kind string) bool {
	switch kind {
	case "lf", "ind", "soft-wrap":
		return true
	default:
		return false
	}
}

func (v *VTerm) writeDamageFromDirectOpsLocked(ops []DamageOp, semanticOps []DamageOp, plan rowCacheReconcilePlan, mode writeDamageMode) WriteDamage {
	damage := v.writeDamageHeaderLocked(plan)
	damage.Ops = ops
	if mode == writeDamageSemanticOnly {
		// 中文说明：semantic-only 是 history tap 的热路径。scroll-out proof 已由
		// recorder 给出，不能再把 live screen scroll rect 当 ordinary history op，
		// 否则普通 stdout 会从 logical-line fast lane 退化成 row ownership diff。
		damage.SemanticOps = semanticOps
		return damage
	}
	damage.SemanticOps = appendSemanticScreenOps(semanticOps, ops)
	v.appendScrollbackDamageLocked(&damage, plan)
	damage.LiveTailAppendRows = trailingWrappedDamageRows(damage.ScrollbackAppend)
	return damage
}

func applySemanticHistoryOpsToDamage(damage *WriteDamage, plan rowCacheReconcilePlan, historyOps []DamageOp, afterScrollbackLen int) {
	if damage == nil || len(historyOps) == 0 {
		return
	}
	damage.ScrollbackAppend = historyOps
	damage.ScrollbackTrim = maxInt(0, plan.beforeScrollbackLen+len(historyOps)-afterScrollbackLen)
	damage.LiveTailAppendRows = trailingWrappedDamageRows(damage.ScrollbackAppend)
}

func (v *VTerm) writeDamageRequiresFullReplaceLocked(plan rowCacheReconcilePlan, reason string) WriteDamage {
	damage := v.writeDamageHeaderLocked(plan)
	damage.RequiresFullReplace = true
	damage.FullReplaceReason = reason
	v.appendScrollbackDamageLocked(&damage, plan)
	damage.LiveTailAppendRows = trailingWrappedDamageRows(damage.ScrollbackAppend)
	return damage
}

func appendSemanticScreenOps(base []DamageOp, ops []DamageOp) []DamageOp {
	if len(ops) == 0 {
		return base
	}
	out := base
	for _, op := range ops {
		switch op.Code {
		case ScreenOpScrollRect:
			out = append(out, op)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (v *VTerm) writeDamageHeaderLocked(plan rowCacheReconcilePlan) WriteDamage {
	damage := WriteDamage{
		Cursor:              v.cursor,
		Modes:               v.modes,
		SizeRows:            len(plan.afterScreen),
		ScreenScroll:        plan.screenScrollShift,
		RequiresFullReplace: false,
	}
	if v.emu != nil {
		damage.SizeCols = v.emu.Width()
	}
	return damage
}

func (v *VTerm) appendScrollbackDamageLocked(damage *WriteDamage, plan rowCacheReconcilePlan) {
	if v == nil || v.emu == nil || damage == nil {
		return
	}
	afterScrollbackLen := v.scrollbackRowCountLocked()
	rowsDroppedFromFront := maxInt(0, plan.beforeScrollbackLen+plan.requiredScrollbackAppends-afterScrollbackLen)
	retainedFromOldScrollback := plan.beforeScrollbackLen - rowsDroppedFromFront
	if retainedFromOldScrollback < 0 {
		retainedFromOldScrollback = 0
	}
	damage.ScrollbackTrim = rowsDroppedFromFront
	if retainedFromOldScrollback > afterScrollbackLen {
		retainedFromOldScrollback = afterScrollbackLen
	}
	for row := retainedFromOldScrollback; row < afterScrollbackLen; row++ {
		sourceIndex := row - retainedFromOldScrollback
		sourceRow := -1
		if sourceIndex >= 0 && sourceIndex < len(plan.scrollbackSourceRows) {
			sourceRow = plan.scrollbackSourceRows[sourceIndex]
		}
		op := DamageOp{
			Row:        row,
			Cells:      cloneCellSlice(v.scrollbackRowViewLocked(row)),
			Timestamp:  timeAt(v.scrollbackTimestamps, row),
			RowKind:    stringAt(v.scrollbackRowKinds, row),
			Wrapped:    v.scrollbackRowWrappedAtLocked(row),
			WrappedSet: true,
		}
		if sourceRow >= 0 {
			op.Row = sourceRow
			op.RowSet = true
		}
		damage.ScrollbackAppend = append(damage.ScrollbackAppend, op)
	}
}

func trailingWrappedDamageRows(rows []DamageOp) int {
	count := 0
	for i := len(rows) - 1; i >= 0; i-- {
		if !(rows[i].WrappedSet && rows[i].Wrapped) {
			break
		}
		count++
	}
	return count
}

func (v *VTerm) screenRowsForResizeLocked(count int) [][]Cell {
	if v == nil || v.emu == nil || count <= 0 {
		return nil
	}
	out := make([][]Cell, count)
	for row := 0; row < count; row++ {
		used := v.emu.ScreenLineUsed(row)
		if used <= 0 {
			used = v.emu.Width()
		}
		if used <= 0 {
			continue
		}
		cells := make([]Cell, used)
		for col := 0; col < used; col++ {
			cells[col] = v.convertCell(v.emu.CellAt(col, row))
		}
		out[row] = cells
	}
	return out
}

func (v *VTerm) screenTailFillsForResizeLocked(count int) []*CellStyle {
	if v == nil || v.emu == nil || count <= 0 {
		return nil
	}
	width := v.emu.Width()
	if width <= 0 {
		return nil
	}
	out := make([]*CellStyle, count)
	for row := 0; row < count; row++ {
		used := v.emu.ScreenLineUsed(row)
		if used < 0 {
			used = 0
		}
		if used >= width {
			continue
		}
		out[row] = v.screenTailFillForResizeLocked(row, used, width)
	}
	return out
}

func (v *VTerm) screenTailFillForResizeLocked(row int, used int, width int) *CellStyle {
	var fill *CellStyle
	for col := used; col < width; col++ {
		cell := v.convertCell(v.emu.CellAt(col, row))
		style, ok := resizeTailFillStyle(cell)
		if !ok {
			return nil
		}
		if fill == nil {
			cloned := style
			fill = &cloned
			continue
		}
		if *fill != style {
			return nil
		}
	}
	return fill
}

func resizeTailFillStyle(cell Cell) (CellStyle, bool) {
	if cell.Content != "" && cell.Content != " " {
		return CellStyle{}, false
	}
	if cell.Width != 0 && cell.Width != 1 {
		return CellStyle{}, false
	}
	// 中文说明：这里识别的是 terminal 当前背景延伸到行尾，不是 logical text；
	// 只保留背景，避免把尾部空白上的临时文本样式误当成可重排内容。
	if cell.Style.BG == "" || cell.LinkURL != "" || cell.LinkParams != "" {
		return CellStyle{}, false
	}
	return CellStyle{BG: cell.Style.BG}, true
}

func resizeReflowRowsForResize(rows [][]Cell, tailFills []*CellStyle, wrapped []bool, timestamps []time.Time, rowKinds []string, newCols, newRows int) []resizeReflowLine {
	if len(rows) == 0 || newRows <= 0 {
		return nil
	}
	if newCols < 1 {
		newCols = 1
	}
	allRows := resizeReflowRowsFromScreen(rows, tailFills, timestamps, rowKinds, wrapped)
	if len(allRows) == 0 {
		return nil
	}
	return resizeReflowLines(allRows, newCols)
}

func shouldTailVisibleRowsForResize(beforeCols, beforeRows, afterCols, afterRows int, reflowed []resizeReflowLine) bool {
	return afterCols > 0 &&
		afterRows > 0 &&
		beforeCols > afterCols &&
		len(reflowed) > afterRows
}

func shouldUseTailFillResizeWriteback(rows []resizeReflowLine) bool {
	for _, row := range rows {
		if row.tailFill != nil {
			return true
		}
	}
	return false
}

func trimTrailingBlankResizeReflowLines(rows []resizeReflowLine) []resizeReflowLine {
	end := len(rows)
	for end > 0 && resizeReflowLineIsBlank(rows[end-1]) && rows[end-1].timestamp.IsZero() && rows[end-1].rowKind == "" {
		end--
	}
	return rows[:end]
}

func resizeReflowLinesUVRows(rows []resizeReflowLine, width int) []uv.Line {
	if len(rows) == 0 {
		return nil
	}
	out := make([]uv.Line, len(rows))
	for i, row := range rows {
		out[i] = uvLineWithTailFill(row.cells, row.tailFill, width)
	}
	return out
}

func uvLineWithTailFill(cells []Cell, tailFill *CellStyle, width int) uv.Line {
	line := uvLine(cells)
	if tailFill == nil || width <= len(line) {
		return line
	}
	fill := *uvCell(Cell{Content: " ", Width: 1, Style: *tailFill})
	for len(line) < width {
		line = append(line, fill)
	}
	return line
}

func resizeReflowLinesWrapped(rows []resizeReflowLine) []bool {
	if len(rows) == 0 {
		return nil
	}
	out := make([]bool, len(rows))
	for i, row := range rows {
		out[i] = row.wrapped
	}
	return out
}

func resizeReflowLinesUsed(rows []resizeReflowLine) []int {
	if len(rows) == 0 {
		return nil
	}
	out := make([]int, len(rows))
	for i, row := range rows {
		out[i] = len(row.cells)
	}
	return out
}

type resizeHistoryPlan struct {
	rows       []resizeReflowLine
	historyEnd int
	planned    bool
}

func planTailResizeHistoryAppend(rows []resizeReflowLine, screenRows int) resizeHistoryPlan {
	if len(rows) == 0 || screenRows <= 0 {
		return resizeHistoryPlan{}
	}
	historyEnd := len(rows) - screenRows
	if historyEnd < 0 {
		historyEnd = 0
	}
	return resizeHistoryPlan{
		rows:       rows,
		historyEnd: historyEnd,
		planned:    true,
	}
}

func (p resizeHistoryPlan) isZero() bool {
	return !p.planned
}

func (p resizeHistoryPlan) scrollbackAppend() []DamageOp {
	if !p.planned || p.historyEnd <= 0 {
		return nil
	}
	out := make([]DamageOp, 0, p.historyEnd)
	for i := 0; i < p.historyEnd && i < len(p.rows); i++ {
		row := p.rows[i]
		if resizeReflowLineIsBlank(row) && row.timestamp.IsZero() {
			continue
		}
		out = append(out, resizeReflowLineDamageOp(row))
	}
	return out
}

func resizeLiveTailRowsFromPlan(plan resizeHistoryPlan, sourceRow int) int {
	if !resizePlanHasRows(plan) || sourceRow < 0 {
		return 0
	}
	count := 0
	for i := minInt(plan.historyEnd, len(plan.rows)) - 1; i >= 0; i-- {
		if plan.rows[i].sourceRow != sourceRow {
			break
		}
		count++
	}
	return count
}

func resizePlanHasRows(plan resizeHistoryPlan) bool {
	return plan.planned && plan.historyEnd > 0 && len(plan.rows) > 0
}

func shouldFallbackResizeHistoryAppend(beforeCols, beforeRows, afterCols, afterRows int, beforeRowsReflowed []resizeReflowLine, plan resizeHistoryPlan) bool {
	return false
}

func resizeScrollbackAppendFromReflowedRows(beforeRows []resizeReflowLine, afterScreenRows [][]Cell, afterScreenTailFills []*CellStyle, afterScreenTimestamps []time.Time, afterScreenRowKinds []string, afterScreenWrapped []bool) []DamageOp {
	if len(beforeRows) == 0 {
		return nil
	}
	visible := resizeReflowRowsFromScreen(afterScreenRows, afterScreenTailFills, afterScreenTimestamps, afterScreenRowKinds, afterScreenWrapped)
	for i := range visible {
		visible[i].fingerprint = resizeReflowLineFingerprint(visible[i])
	}
	visibleStart, visibleCount := resizeVisibleBlockInBeforeRows(beforeRows, visible)
	visibleEnd := visibleStart + visibleCount
	out := make([]DamageOp, 0, maxInt(0, len(beforeRows)-visibleCount))
	appendRow := func(i int) {
		row := beforeRows[i]
		if resizeReflowLineIsBlank(row) && row.timestamp.IsZero() {
			return
		}
		out = append(out, resizeReflowLineDamageOp(row))
	}
	for start := 0; start < len(beforeRows); {
		end := start + 1
		for end < len(beforeRows) && beforeRows[end-1].wrapped {
			end++
		}
		if end <= visibleStart || start >= visibleEnd {
			for i := start; i < end; i++ {
				appendRow(i)
			}
		} else {
			for i := start; i < minInt(end, visibleStart); i++ {
				appendRow(i)
			}
			for i := maxInt(start, visibleEnd); i < end; i++ {
				appendRow(i)
			}
		}
		start = end
	}
	return out
}

func resizeReflowLineDamageOp(row resizeReflowLine) DamageOp {
	return DamageOp{
		Cells:      cloneCellSlice(row.cells),
		Timestamp:  row.timestamp,
		RowKind:    row.rowKind,
		Wrapped:    row.wrapped,
		WrappedSet: true,
	}
}

func resizeVisibleBlockInBeforeRows(beforeRows, visible []resizeReflowLine) (start int, count int) {
	bestStart := len(beforeRows)
	bestCount := 0
	bestScore := -1
	for i := 0; i < len(beforeRows); i++ {
		matched := 0
		for matched < len(visible) && i+matched < len(beforeRows) && resizeReflowLinesEqual(beforeRows[i+matched], visible[matched]) {
			matched++
		}
		score := resizeReflowLineMetadataScore(beforeRows[i:i+matched], visible[:matched])
		if matched > bestCount || (matched == bestCount && score > bestScore) {
			bestStart = i
			bestCount = matched
			bestScore = score
			if bestCount == len(visible) {
				break
			}
		}
	}
	if bestCount == 0 {
		return len(beforeRows), 0
	}
	return bestStart, bestCount
}

func resizeReflowLinesEqual(left, right resizeReflowLine) bool {
	return left.wrapped == right.wrapped &&
		cellStylePointerEqual(left.tailFill, right.tailFill) &&
		resizeReflowLineCellsEqual(left.cells, right.cells)
}

func resizeReflowLineMetadataScore(left, right []resizeReflowLine) int {
	score := 0
	for i := 0; i < len(left) && i < len(right); i++ {
		if !left[i].timestamp.IsZero() && !right[i].timestamp.IsZero() && left[i].timestamp.Equal(right[i].timestamp) {
			score += 2
		}
		if left[i].rowKind != "" && right[i].rowKind != "" && left[i].rowKind == right[i].rowKind {
			score++
		}
	}
	return score
}

func resizeReflowLineCellsEqual(left, right []Cell) bool {
	left = trimResizeReflowDefaultBlankCells(left)
	right = trimResizeReflowDefaultBlankCells(right)
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func trimResizeReflowDefaultBlankCells(cells []Cell) []Cell {
	for len(cells) > 0 {
		cell := cells[len(cells)-1]
		if cell.Content != "" && cell.Content != " " {
			break
		}
		if cell.Width != 0 && cell.Width != 1 {
			break
		}
		if cell.Style != (CellStyle{}) || cell.LinkURL != "" || cell.LinkParams != "" {
			break
		}
		cells = cells[:len(cells)-1]
	}
	return cells
}

func resizeReflowRowsFromScreen(rows [][]Cell, tailFills []*CellStyle, timestamps []time.Time, rowKinds []string, wrapped []bool) []resizeReflowLine {
	out := make([]resizeReflowLine, 0, len(rows))
	for row, cells := range rows {
		line := resizeReflowLine{
			cells:     cloneCellSlice(cells),
			tailFill:  cloneCellStylePointer(pointerAt(tailFills, row)),
			wrapped:   boolAt(wrapped, row),
			sourceRow: row,
			timestamp: timeAt(timestamps, row),
			rowKind:   stringAt(rowKinds, row),
		}
		line.fingerprint = resizeReflowLineFingerprint(line)
		out = append(out, line)
	}
	return out
}

func resizeReflowLines(rows []resizeReflowLine, width int) []resizeReflowLine {
	if width < 1 {
		width = 1
	}
	out := make([]resizeReflowLine, 0, len(rows))
	for i := 0; i < len(rows); i++ {
		cells := resizeReflowCellsForLine(rows[i])
		tailFill := rows[i].tailFill
		for i < len(rows)-1 && rows[i].wrapped {
			i++
			cells = append(cells, resizeReflowCellsForLine(rows[i])...)
			tailFill = rows[i].tailFill
		}
		out = append(out, splitResizeLogicalLine(cells, width, tailFill)...)
	}
	if len(out) == 0 {
		return []resizeReflowLine{{sourceRow: 0}}
	}
	return out
}

func resizeReflowCellsForLine(row resizeReflowLine) []resizeReflowCell {
	out := make([]resizeReflowCell, 0, len(row.cells))
	for _, cell := range row.cells {
		out = append(out, resizeReflowCell{
			cell:      cell,
			sourceRow: row.sourceRow,
			timestamp: row.timestamp,
			rowKind:   row.rowKind,
		})
	}
	return out
}

func splitResizeLogicalLine(cells []resizeReflowCell, width int, tailFill *CellStyle) []resizeReflowLine {
	if width < 1 {
		width = 1
	}
	if width == 1 {
		return resizeReflowApplyTailFill(splitResizeLogicalLineByColumns(cells, width), tailFill)
	}
	if len(cells) == 0 {
		return resizeReflowApplyTailFill([]resizeReflowLine{{sourceRow: 0}}, tailFill)
	}
	out := make([]resizeReflowLine, 0, (resizeReflowCellsWidth(cells)+width-1)/width)
	var current []resizeReflowCell
	currentWidth := 0
	flush := func(wrapped bool) {
		line := resizeReflowLine{
			cells:     cloneResizeReflowLineCells(current),
			wrapped:   wrapped,
			sourceRow: resizeReflowLineSourceRow(current),
			timestamp: resizeReflowLineTimestamp(current),
			rowKind:   resizeReflowLineRowKind(current),
		}
		line.fingerprint = resizeReflowLineFingerprint(line)
		out = append(out, line)
		current = nil
		currentWidth = 0
	}
	for i := 0; i < len(cells); i++ {
		cell := cells[i].cell
		cellWidth := resizeVisibleCellWidth(cell)
		if cellWidth <= 0 {
			continue
		}
		if currentWidth > 0 && currentWidth+cellWidth > width {
			flush(true)
		}
		if cellWidth > width {
			if currentWidth > 0 {
				flush(true)
			}
			line := resizeReflowLine{
				cells:     []Cell{cell},
				wrapped:   false,
				sourceRow: cells[i].sourceRow,
				timestamp: cells[i].timestamp,
				rowKind:   cells[i].rowKind,
			}
			line.fingerprint = resizeReflowLineFingerprint(line)
			out = append(out, line)
			continue
		}
		current = append(current, cells[i])
		for col := 1; col < cell.Width && i+1 < len(cells) && isWideContinuationCell(cells[i+1].cell); col++ {
			i++
			current = append(current, cells[i])
		}
		currentWidth += cellWidth
		if currentWidth == width {
			flush(true)
		}
	}
	if current != nil || len(out) == 0 {
		flush(false)
	} else {
		out[len(out)-1].wrapped = false
	}
	return resizeReflowApplyTailFill(out, tailFill)
}

func resizeReflowApplyTailFill(rows []resizeReflowLine, tailFill *CellStyle) []resizeReflowLine {
	if len(rows) == 0 || tailFill == nil {
		return rows
	}
	rows[len(rows)-1].tailFill = cloneCellStylePointer(tailFill)
	rows[len(rows)-1].fingerprint = resizeReflowLineFingerprint(rows[len(rows)-1])
	return rows
}

func splitResizeLogicalLineByColumns(cells []resizeReflowCell, width int) []resizeReflowLine {
	if width < 1 {
		width = 1
	}
	if len(cells) == 0 {
		return []resizeReflowLine{{sourceRow: 0}}
	}
	out := make([]resizeReflowLine, 0, len(cells))
	for _, cell := range cells {
		if resizeVisibleCellWidth(cell.cell) <= 0 {
			continue
		}
		line := resizeReflowLine{
			cells:     []Cell{cell.cell},
			wrapped:   true,
			sourceRow: cell.sourceRow,
			timestamp: cell.timestamp,
			rowKind:   cell.rowKind,
		}
		line.fingerprint = resizeReflowLineFingerprint(line)
		out = append(out, line)
	}
	if len(out) == 0 {
		return []resizeReflowLine{{sourceRow: 0}}
	}
	out[len(out)-1].wrapped = false
	return out
}

func resizeReflowCellsWidth(cells []resizeReflowCell) int {
	width := 0
	for _, cell := range cells {
		width += resizeVisibleCellWidth(cell.cell)
	}
	return width
}

func resizeReflowLineFingerprint(line resizeReflowLine) rowFingerprint {
	fingerprint := rowFingerprint{blank: true}
	hashUint64(&fingerprint.hash, uint64(len(line.cells)))
	for _, cell := range line.cells {
		if !hashVTermCellFingerprint(&fingerprint.hash, cell) {
			fingerprint.blank = false
		}
	}
	hashCellStylePointerFingerprint(&fingerprint.hash, line.tailFill)
	if line.tailFill != nil {
		fingerprint.blank = false
	}
	return fingerprint
}

func cloneResizeReflowLineCells(cells []resizeReflowCell) []Cell {
	out := make([]Cell, 0, len(cells))
	for _, cell := range cells {
		out = append(out, cell.cell)
	}
	return out
}

func resizeReflowLineSourceRow(cells []resizeReflowCell) int {
	for _, cell := range cells {
		if cell.sourceRow >= 0 {
			return cell.sourceRow
		}
	}
	return 0
}

func resizeReflowLineTimestamp(cells []resizeReflowCell) time.Time {
	for _, cell := range cells {
		if !cell.timestamp.IsZero() {
			return cell.timestamp
		}
	}
	return time.Time{}
}

func resizeReflowLineRowKind(cells []resizeReflowCell) string {
	for i, cell := range cells {
		if i == 0 || cell.rowKind != "" {
			return cell.rowKind
		}
	}
	return ""
}

func resizeVisibleCellWidth(cell Cell) int {
	if cell.Content == "" && cell.Width == 0 && cell.Style == (CellStyle{}) && cell.LinkURL == "" && cell.LinkParams == "" {
		return 0
	}
	if cell.Width <= 0 {
		return 0
	}
	return cell.Width
}

func resizeReflowLineIsBlank(row resizeReflowLine) bool {
	for _, cell := range row.cells {
		if !cellIsBlank(cell) {
			return false
		}
	}
	return true
}

// evictedAppendOpsFromCharmVTDamages 把 direct damage 中的 scrollback eviction 转成
// DamageOp，不丢弃空行——logical-line 无限历史必须保留段落间距。ScrollbackAppend
// 出于旧 proof 消费路径经 filterNonEmptyEvictedAppendOps 继续过滤空行。
//
// damage 流是有序的：用 ModeDamage(47/1047/1049) 在流内跟踪 alt 状态，把每条
// ScrollbackDamage 归属到 primary eviction 或 alternate scroll。这样同一次写入内
// "主屏滚出→进 alt" 或 "进 alt→滚动→退 alt" 都不会把 alt 行误入主历史，
// 也不会丢进 alt 前的主屏 eviction。
//
// ED2（ClearWithScrollback）的滚出行不走 recordScrollbackLine，而是随
// ControlDamage("ed").ScrollOut 携带——它们同样"真正离开可见区且不可再寻址"，
// 必须按流内顺序并入 eviction，否则 `clear` 前的屏幕内容会从历史中凭空消失。
func (v *VTerm) evictedAppendOpsFromCharmVTDamages(damages []charmvt.Damage, altAtStart bool, timestamps []time.Time, rowKinds []string, ownership []string, persistedRows [][]Cell) (evicted []DamageOp, alternate []DamageOp) {
	if len(damages) == 0 {
		return nil, nil
	}
	altActive := altAtStart
	matchedPersistedRows := make([]bool, len(persistedRows))
	appendOp := func(op DamageOp) {
		if altActive {
			alternate = append(alternate, op)
		} else {
			evicted = append(evicted, op)
		}
	}
	for _, raw := range damages {
		switch d := raw.(type) {
		case charmvt.ModeDamage:
			if d.Private && (d.Mode == 47 || d.Mode == 1047 || d.Mode == 1049) {
				altActive = d.Enabled
			}
		case charmvt.ScrollbackDamage:
			row := d.Y
			if row < 0 {
				row = 0
			}
			append := v.scrollbackRowAppendFromCharmVTDamage(d, timestamps, rowKinds, "")
			append.Ownership = matchedPersistedOwnership(append, row, ownership, persistedRows, matchedPersistedRows)
			appendOp(damageOpFromScrollbackRowAppend(append))
		case charmvt.ControlDamage:
			for _, scrollOut := range d.ScrollOut {
				append := v.scrollbackRowAppendFromCharmVTDamage(scrollOut, timestamps, rowKinds, "")
				append.Ownership = matchedPersistedOwnership(append, scrollOut.Y, ownership, persistedRows, matchedPersistedRows)
				appendOp(damageOpFromScrollbackRowAppend(append))
			}
		}
	}
	return evicted, alternate
}

// persistedScreenRowsLocked captures only rows whose content has already been
// committed outside the emulator transaction. Keeping this sparse avoids a
// second full-screen copy on every PTY write.
func (v *VTerm) persistedScreenRowsLocked(ownership []string) [][]Cell {
	var rows [][]Cell
	for row, owner := range ownership {
		if owner != HistoryPersistedOwnership {
			continue
		}
		if rows == nil {
			rows = make([][]Cell, len(ownership))
		}
		rows[row] = cloneCellSlice(v.screenRowUsedViewLocked(row))
	}
	return rows
}

// Damage row indices describe positions in the ordered scroll stream. During
// a large write those positions can be reused by rows created by the same
// write, so an index-only ownership lookup can suppress fresh history. Match
// the immutable proof against the pre-write persisted content and consume each
// persisted row at most once.
func matchedPersistedOwnership(row ScrollbackRowAppend, preferred int, ownership []string, persistedRows [][]Cell, matched []bool) string {
	proofText := scrollbackRowAppendText(row)
	if len(persistedRows) == 0 {
		return ""
	}
	matches := func(index int) bool {
		return index >= 0 && index < len(persistedRows) &&
			index < len(ownership) && ownership[index] == HistoryPersistedOwnership &&
			index < len(matched) && !matched[index] &&
			cellRowText(persistedRows[index]) == proofText
	}
	if matches(preferred) {
		matched[preferred] = true
		return HistoryPersistedOwnership
	}
	for index := range persistedRows {
		if matches(index) {
			matched[index] = true
			return HistoryPersistedOwnership
		}
	}
	return ""
}

func scrollbackRowAppendText(row ScrollbackRowAppend) string {
	if len(row.Runs) > 0 {
		var builder strings.Builder
		for _, run := range row.Runs {
			builder.WriteString(run.Text)
		}
		return builder.String()
	}
	return cellRowText(row.Cells)
}

func cellRowText(cells []Cell) string {
	var builder strings.Builder
	for _, cell := range cells {
		builder.WriteString(cell.Content)
	}
	return builder.String()
}

func filterNonEmptyEvictedAppendOps(ops []DamageOp) []DamageOp {
	if len(ops) == 0 {
		return nil
	}
	out := make([]DamageOp, 0, len(ops))
	for _, op := range ops {
		if len(op.Cells) == 0 && len(op.Runs) == 0 {
			continue
		}
		out = append(out, op)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (v *VTerm) scrollbackRowAppendsFromCharmVTDamages(rows []charmvt.ScrollbackDamage, timestamps []time.Time, rowKinds []string) []ScrollbackRowAppend {
	if len(rows) == 0 {
		return nil
	}
	out := make([]ScrollbackRowAppend, 0, len(rows))
	for _, row := range rows {
		if row.Text == "" && len(row.Runs) == 0 && len(row.Cells) == 0 {
			continue
		}
		out = append(out, v.scrollbackRowAppendFromCharmVTDamage(row, timestamps, rowKinds, ""))
	}
	return out
}

func (v *VTerm) scrollbackRowAppendFromCharmVTDamage(row charmvt.ScrollbackDamage, timestamps []time.Time, rowKinds []string, ownership string) ScrollbackRowAppend {
	cells := uvCellsToVTermDamageCells(v, row.Cells)
	runs := []CellRun(nil)
	switch {
	case len(row.Runs) > 0:
		cells = nil
		runs = scrollbackRunsToVTermRuns(v, row.Runs)
	case row.Text != "":
		cells = nil
		runs = asciiTextToVTermRuns(row.Text)
	}
	return ScrollbackRowAppend{
		Cells:      cells,
		Runs:       runs,
		Timestamp:  timeAt(timestamps, row.Y),
		RowKind:    stringAt(rowKinds, row.Y),
		Ownership:  ownership,
		Row:        row.Y,
		RowSet:     row.Y >= 0,
		Wrapped:    row.Wrapped,
		WrappedSet: true,
	}
}

func damageOpFromScrollbackRowAppend(row ScrollbackRowAppend) DamageOp {
	return DamageOp{
		Cells:      cloneCellSlice(row.Cells),
		Runs:       cloneCellRuns(row.Runs),
		Timestamp:  row.Timestamp,
		RowKind:    row.RowKind,
		Ownership:  row.Ownership,
		Row:        row.Row,
		RowSet:     row.RowSet,
		Wrapped:    row.Wrapped,
		WrappedSet: row.WrappedSet,
	}
}

func asciiTextToVTermRuns(text string) []CellRun {
	if text == "" {
		return nil
	}
	return []CellRun{{Text: text}}
}

func scrollbackRunsToVTermRuns(v *VTerm, runs []charmvt.ScrollbackRun) []CellRun {
	if len(runs) == 0 {
		return nil
	}
	out := make([]CellRun, 0, len(runs))
	for _, run := range runs {
		if run.Text == "" {
			continue
		}
		out = append(out, CellRun{Style: v.convertStyle(run.Style), Text: run.Text})
	}
	return out
}

func (v *VTerm) textDamagePayloadLocked(damage charmvt.TextDamage) ([]Cell, []CellRun) {
	if damage.Text != "" {
		return nil, asciiTextToVTermRuns(damage.Text)
	}
	if len(damage.Runs) > 0 {
		return nil, scrollbackRunsToVTermRuns(v, damage.Runs)
	}
	return uvCellsToVTermDamageCells(v, damage.Cells), nil
}

func detectScreenScrollShift(before, after []rowFingerprint) int {
	limit := minInt(len(before), len(after))
	if limit <= 1 {
		return 0
	}
	bestShift := 0
	bestScore := rowAlignmentScore(before, after, 0)
	for shift := 1; shift < limit; shift++ {
		if limit-shift < bestScore {
			break
		}
		score := rowAlignmentScore(before, after, shift)
		if score > bestScore {
			bestScore = score
			bestShift = shift
			if score == limit-shift {
				break
			}
		}
	}
	if bestShift == 0 {
		return 0
	}
	if bestScore <= rowAlignmentScore(before, after, 0) {
		return 0
	}
	return bestShift
}

func rowAlignmentScore(before, after []rowFingerprint, shift int) int {
	score := 0
	for row := 0; row+shift < len(before) && row < len(after); row++ {
		if rowFingerprintsEqual(before[row+shift], after[row]) {
			score++
		}
	}
	return score
}

func rowFingerprintsEqual(left, right rowFingerprint) bool {
	return left.hash == right.hash && left.blank == right.blank
}

func rowFingerprintIsBlank(row rowFingerprint) bool {
	return row.blank
}

func rowFingerprintVisualHash(row rowFingerprint) uint64 {
	hash := row.hash
	if row.blank {
		hash ^= 1
	}
	return hash
}

func damageChangedRowCount(damage WriteDamage) int {
	seen := make(map[int]struct{}, len(damage.Ops))
	for _, op := range damage.Ops {
		switch op.Code {
		case ScreenOpWriteSpan, ScreenOpClearToEOL:
			seen[op.Row] = struct{}{}
		case ScreenOpScrollRect, ScreenOpClearRect:
			for row := op.Rect.Y; row < op.Rect.Y+op.Rect.Height; row++ {
				seen[row] = struct{}{}
			}
		case ScreenOpCopyRect:
			for row := op.DstY; row < op.DstY+op.Src.Height; row++ {
				seen[row] = struct{}{}
			}
		case ScreenOpControl:
			seen[op.Row] = struct{}{}
		}
	}
	return len(seen) + len(damage.ScrollbackAppend)
}

func damageChangedCellCount(damage WriteDamage) int {
	count := 0
	for _, op := range damage.Ops {
		switch op.Code {
		case ScreenOpWriteSpan:
			count += len(op.Cells)
		case ScreenOpClearRect:
			count += maxInt(0, op.Rect.Width*op.Rect.Height)
		case ScreenOpClearToEOL:
			count++
		case ScreenOpScrollRect, ScreenOpCopyRect:
			count += maxInt(0, op.Rect.Width*op.Rect.Height)
		case ScreenOpControl:
			count++
		}
	}
	for _, row := range damage.ScrollbackAppend {
		if len(row.Runs) > 0 {
			for _, run := range row.Runs {
				count += len(run.Text)
			}
			continue
		}
		count += len(row.Cells)
	}
	return count
}

type directDamageSummary struct {
	Items                  int
	Rows                   int
	Cells                  int
	MaxItemsPerRow         int
	ScreenWidth            int
	ScreenHeight           int
	TouchedRows            []int
	HasScrollOrMove        bool
	HasUnsupported         bool
	HasFullScreenDamage    bool
	HasPartialScreenDamage bool
}

type directDamageInterval struct {
	start int
	end   int
}

func (s directDamageSummary) fullReplaceReason() (string, bool) {
	if s.Items == 0 || s.ScreenWidth <= 0 || s.ScreenHeight <= 0 {
		return "", false
	}
	if s.HasScrollOrMove || s.HasUnsupported || s.HasPartialScreenDamage {
		return "", false
	}
	if s.HasFullScreenDamage {
		return "direct_screen_damage", true
	}
	if s.Cells <= 0 && s.Rows == 0 {
		return "", false
	}
	if s.Items >= repeatedDirectDamageMinItems &&
		s.Cells >= repeatedDirectDamageMinCells &&
		s.Rows > 0 &&
		s.MaxItemsPerRow >= maxInt(1, s.ScreenWidth*repeatedDirectDamageItemRatioNumerator/repeatedDirectDamageItemRatioDenominator) {
		return "repeated_direct_damage", true
	}
	if s.ScreenHeight < broadDirectDamageMinRows || s.Cells < broadDirectDamageMinCells {
		return "", false
	}
	rowThreshold := maxInt(1, s.ScreenHeight*broadDirectDamageRowRatioNumerator/broadDirectDamageRowRatioDenominator)
	totalCells := s.ScreenWidth * s.ScreenHeight
	cellThreshold := maxInt(1, totalCells*broadDirectDamageCellRatioNumerator/broadDirectDamageCellRatioDenominator)
	if s.Rows >= rowThreshold && s.Cells >= cellThreshold {
		return "broad_direct_cell_damage", true
	}
	return "", false
}

func directDamageStats(damages []charmvt.Damage, screenWidth, screenHeight int) directDamageSummary {
	stats := directDamageSummary{
		Items:        len(damages),
		ScreenWidth:  screenWidth,
		ScreenHeight: screenHeight,
	}
	if len(damages) == 0 || screenWidth <= 0 || screenHeight <= 0 {
		return stats
	}
	changedRows := make(map[int]struct{}, minInt(screenHeight, len(damages)))
	rowItems := make(map[int]int, minInt(screenHeight, len(damages)))
	covered := make(map[int][]directDamageInterval, minInt(screenHeight, len(damages)))
	markItemRow := func(y int) {
		if y < 0 || y >= screenHeight {
			return
		}
		rowItems[y]++
		if rowItems[y] > stats.MaxItemsPerRow {
			stats.MaxItemsPerRow = rowItems[y]
		}
	}
	addCoverage := func(y int, x int, width int) {
		if y < 0 || y >= screenHeight || width <= 0 {
			return
		}
		x0 := maxInt(0, x)
		x1 := minInt(screenWidth, x+width)
		if x1 <= x0 {
			return
		}
		covered[y] = append(covered[y], directDamageInterval{start: x0, end: x1})
	}
	for _, raw := range damages {
		switch d := raw.(type) {
		case charmvt.TextDamage:
			continue
		case charmvt.SpanDamage:
			width := spanDamageCellWidth(d.Cells)
			if width <= 0 {
				continue
			}
			stats.Cells += clampedRectArea(d.X, d.Y, width, 1, screenWidth, screenHeight)
			markClampedRows(changedRows, d.Y, 1, screenHeight)
			markItemRow(d.Y)
			addCoverage(d.Y, d.X, width)
		case charmvt.CellDamage:
			width := d.Width
			if width <= 0 {
				width = 1
			}
			stats.Cells += clampedRectArea(d.X, d.Y, width, 1, screenWidth, screenHeight)
			markClampedRows(changedRows, d.Y, 1, screenHeight)
			markItemRow(d.Y)
			addCoverage(d.Y, d.X, width)
		case charmvt.RectDamage:
			rect := uv.Rectangle(d)
			area := clampedRectArea(rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy(), screenWidth, screenHeight)
			if area <= 0 {
				continue
			}
			stats.Cells += area
			markClampedRows(changedRows, rect.Min.Y, rect.Dy(), screenHeight)
			for row := maxInt(0, rect.Min.Y); row < minInt(screenHeight, rect.Max.Y); row++ {
				markItemRow(row)
				addCoverage(row, rect.Min.X, rect.Dx())
			}
		case charmvt.ClearDamage:
			rect := uv.Rectangle(d)
			area := clampedRectArea(rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy(), screenWidth, screenHeight)
			if area <= 0 {
				continue
			}
			stats.Cells += area
			markClampedRows(changedRows, rect.Min.Y, rect.Dy(), screenHeight)
			for row := maxInt(0, rect.Min.Y); row < minInt(screenHeight, rect.Max.Y); row++ {
				markItemRow(row)
				addCoverage(row, rect.Min.X, rect.Dx())
			}
		case charmvt.ScrollbackDamage:
			continue
		case charmvt.ControlDamage:
			continue
		case charmvt.ModeDamage:
			continue
		case charmvt.ScrollDamage, charmvt.MoveDamage:
			stats.HasScrollOrMove = true
		case charmvt.ScreenDamage:
			if d.Width == screenWidth && d.Height == screenHeight {
				stats.HasFullScreenDamage = true
				for row := 0; row < screenHeight; row++ {
					changedRows[row] = struct{}{}
					addCoverage(row, 0, screenWidth)
				}
				stats.Cells = maxInt(stats.Cells, screenWidth*screenHeight)
			} else {
				stats.HasPartialScreenDamage = true
			}
		default:
			stats.HasUnsupported = true
		}
	}
	for _, raw := range damages {
		d, ok := raw.(charmvt.TextDamage)
		if !ok {
			continue
		}
		width := textDamageDisplayWidth(d)
		if width <= 0 {
			continue
		}
		uncovered := uncoveredDirectDamageWidth(covered[d.Y], d.X, width, screenWidth)
		if uncovered <= 0 {
			continue
		}
		stats.Cells += uncovered
		markClampedRows(changedRows, d.Y, 1, screenHeight)
		markItemRow(d.Y)
		addCoverage(d.Y, d.X, width)
	}
	stats.Rows = len(changedRows)
	if len(changedRows) > 0 {
		stats.TouchedRows = make([]int, 0, len(changedRows))
		for row := range changedRows {
			stats.TouchedRows = append(stats.TouchedRows, row)
		}
		sort.Ints(stats.TouchedRows)
	}
	return stats
}

func textDamageDisplayWidth(d charmvt.TextDamage) int {
	if d.Text != "" {
		return len(d.Text)
	}
	if len(d.Runs) > 0 {
		width := 0
		for _, run := range d.Runs {
			width += len(run.Text)
		}
		return width
	}
	return spanDamageCellWidth(d.Cells)
}

func uncoveredDirectDamageWidth(intervals []directDamageInterval, x int, width int, screenWidth int) int {
	if width <= 0 || screenWidth <= 0 {
		return 0
	}
	x0 := maxInt(0, x)
	x1 := minInt(screenWidth, x+width)
	if x1 <= x0 {
		return 0
	}
	if len(intervals) == 0 {
		return x1 - x0
	}
	sorted := append([]directDamageInterval(nil), intervals...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].start == sorted[j].start {
			return sorted[i].end < sorted[j].end
		}
		return sorted[i].start < sorted[j].start
	})
	covered := 0
	cursor := x0
	for _, interval := range sorted {
		if interval.end <= cursor {
			continue
		}
		if interval.start >= x1 {
			break
		}
		start := maxInt(cursor, interval.start)
		end := minInt(x1, interval.end)
		if end > start {
			covered += end - start
			cursor = end
		}
		if cursor >= x1 {
			break
		}
	}
	total := x1 - x0
	if covered >= total {
		return 0
	}
	return total - covered
}

func clampedRectArea(x, y, width, height, maxWidth, maxHeight int) int {
	if width <= 0 || height <= 0 || maxWidth <= 0 || maxHeight <= 0 {
		return 0
	}
	x0 := maxInt(0, x)
	y0 := maxInt(0, y)
	x1 := minInt(maxWidth, x+width)
	y1 := minInt(maxHeight, y+height)
	if x1 <= x0 || y1 <= y0 {
		return 0
	}
	return (x1 - x0) * (y1 - y0)
}

func markClampedRows(rows map[int]struct{}, y, height, maxHeight int) {
	if height <= 0 || maxHeight <= 0 {
		return
	}
	start := maxInt(0, y)
	end := minInt(maxHeight, y+height)
	for row := start; row < end; row++ {
		rows[row] = struct{}{}
	}
}

func cloneIntSlice(in []int) []int {
	if len(in) == 0 {
		return nil
	}
	out := make([]int, len(in))
	copy(out, in)
	return out
}

func (v *VTerm) damageOpsFromCharmVTDamages(damages []charmvt.Damage, screenWidth, screenHeight int, timestamps []time.Time, rowKinds []string) ([]DamageOp, bool) {
	if v == nil {
		return nil, false
	}
	ops := make([]DamageOp, 0, len(damages))
	for _, raw := range damages {
		switch d := raw.(type) {
		case charmvt.TextDamage:
			continue
		case charmvt.SpanDamage:
			if len(d.Cells) == 0 {
				continue
			}
			start, cells := v.spanDamageCellsLocked(d.Y, d.X, d.Cells)
			if len(cells) == 0 {
				continue
			}
			ops = append(ops, DamageOp{
				Code:       ScreenOpWriteSpan,
				Row:        d.Y,
				Col:        start,
				Cells:      cells,
				Timestamp:  timeAt(timestamps, d.Y),
				RowKind:    stringAt(rowKinds, d.Y),
				Wrapped:    v.screenRowWrappedAtLocked(d.Y),
				WrappedSet: true,
			})
		case charmvt.CellDamage:
			row := d.Y
			width := d.Width
			if width <= 0 {
				width = 1
			}
			start, cells := v.expandedCellDamageCellsLocked(row, d.X, width)
			if len(cells) == 0 {
				continue
			}
			ops = append(ops, DamageOp{
				Code:       ScreenOpWriteSpan,
				Row:        row,
				Col:        start,
				Cells:      cells,
				Timestamp:  timeAt(timestamps, row),
				RowKind:    stringAt(rowKinds, row),
				Wrapped:    v.screenRowWrappedAtLocked(row),
				WrappedSet: true,
			})
		case charmvt.ClearDamage:
			rect := uv.Rectangle(d)
			if rect.Dx() <= 0 || rect.Dy() <= 0 {
				continue
			}
			row := rect.Min.Y
			if rect.Dy() == 1 && rect.Min.X >= 0 && rect.Max.X >= screenWidth {
				ops = append(ops, DamageOp{
					Code:       ScreenOpClearToEOL,
					Row:        row,
					Col:        rect.Min.X,
					Timestamp:  timeAt(timestamps, row),
					RowKind:    stringAt(rowKinds, row),
					Wrapped:    v.screenRowWrappedAtLocked(row),
					WrappedSet: true,
				})
				continue
			}
			op := DamageOp{
				Code: ScreenOpClearRect,
				Rect: DamageRect{X: rect.Min.X, Y: rect.Min.Y, Width: rect.Dx(), Height: rect.Dy()},
			}
			if rect.Dy() == 1 {
				op.Timestamp = timeAt(timestamps, row)
				op.RowKind = stringAt(rowKinds, row)
				op.Wrapped = v.screenRowWrappedAtLocked(row)
				op.WrappedSet = true
			}
			ops = append(ops, op)
		case charmvt.ScrollDamage:
			rect := d.Rectangle
			if rect.Dx() <= 0 || rect.Dy() <= 0 {
				continue
			}
			ops = append(ops, DamageOp{
				Code: ScreenOpScrollRect,
				Rect: DamageRect{X: rect.Min.X, Y: rect.Min.Y, Width: rect.Dx(), Height: rect.Dy()},
				Dx:   d.Dx,
				Dy:   d.Dy,
			})
		case charmvt.ControlDamage:
			if d.Kind == "" {
				continue
			}
			op := DamageOp{
				Code:      ScreenOpControl,
				Control:   d.Kind,
				Row:       d.Y,
				Col:       d.X,
				Mode:      d.Mode,
				Bottom:    d.Bottom,
				ScrollOut: v.scrollbackRowAppendsFromCharmVTDamages(d.ScrollOut, timestamps, rowKinds),
			}
			if d.HasCell {
				op.Cells = uvCellsToVTermDamageCells(v, []uv.Cell{d.Cell})
			}
			ops = append(ops, op)
		case charmvt.ModeDamage:
			ops = append(ops, DamageOp{
				Code:    ScreenOpModes,
				Mode:    d.Mode,
				Private: d.Private,
				Enabled: d.Enabled,
			})
		case charmvt.ScrollbackDamage:
			continue
		case charmvt.MoveDamage:
			if d.Src.Dx() <= 0 || d.Src.Dy() <= 0 || d.Dst.Dx() <= 0 || d.Dst.Dy() <= 0 {
				continue
			}
			ops = append(ops, DamageOp{
				Code: ScreenOpCopyRect,
				Src:  DamageRect{X: d.Src.Min.X, Y: d.Src.Min.Y, Width: d.Src.Dx(), Height: d.Src.Dy()},
				DstX: d.Dst.Min.X,
				DstY: d.Dst.Min.Y,
			})
		case charmvt.ScreenDamage:
			if d.Width != screenWidth || d.Height != screenHeight {
				return nil, false
			}
			return nil, false
		case charmvt.RectDamage:
			rect := uv.Rectangle(d)
			if rect.Dx() <= 0 || rect.Dy() <= 0 {
				continue
			}
			for row := rect.Min.Y; row < rect.Max.Y; row++ {
				if row < 0 || row >= screenHeight {
					continue
				}
				start, cells := v.expandedScreenRowSpanLocked(row, maxInt(0, rect.Min.X), minInt(screenWidth, rect.Max.X))
				if len(cells) == 0 {
					continue
				}
				ops = append(ops, DamageOp{
					Code:       ScreenOpWriteSpan,
					Row:        row,
					Col:        start,
					Cells:      cells,
					Timestamp:  timeAt(timestamps, row),
					RowKind:    stringAt(rowKinds, row),
					Wrapped:    v.screenRowWrappedAtLocked(row),
					WrappedSet: true,
				})
			}
		default:
			return nil, false
		}
	}
	return ops, true
}

func (v *VTerm) spanDamageCellsLocked(row, col int, cells []uv.Cell) (int, []Cell) {
	out := uvCellsToVTermDamageCells(v, cells)
	if len(out) == 0 {
		return 0, nil
	}
	if col < 0 {
		col = 0
	}
	if col > 0 && startsInsideWideCell(v.screenRowViewLocked(row), col) {
		return v.expandedScreenRowSpanLocked(row, col, col+spanDamageCellWidth(cells))
	}
	return col, out
}

func (v *VTerm) expandedCellDamageCellsLocked(row, col, width int) (int, []Cell) {
	return v.expandedScreenRowSpanLocked(row, col, col+width)
}

func uvCellsToVTermDamageCells(v *VTerm, row []uv.Cell) []Cell {
	if v == nil || len(row) == 0 {
		return nil
	}
	out := make([]Cell, 0, len(row))
	for i := range row {
		cell := v.convertCell(&row[i])
		out = append(out, cell)
		for col := 1; col < cell.Width; col++ {
			out = append(out, Cell{})
		}
	}
	return out
}

func cloneDenseCellRows(rows [][]Cell, height, width int) [][]Cell {
	out := make([][]Cell, height)
	for row := 0; row < height; row++ {
		out[row] = damagePadRow(cellRowAt(rows, row), width)
	}
	return out
}

func damagePadRow(row []Cell, width int) []Cell {
	out := make([]Cell, width)
	for col := 0; col < width; col++ {
		out[col] = cellAt(row, col)
	}
	return out
}

func damageScreenWidth(beforeRows, afterRows [][]Cell) int {
	width := 0
	for _, rows := range [][][]Cell{beforeRows, afterRows} {
		for _, row := range rows {
			if len(row) > width {
				width = len(row)
			}
		}
	}
	return maxInt(width, 1)
}

func applyDamageScrollRect(rows [][]Cell, op DamageOp) {
	if op.Rect.Width <= 0 || op.Rect.Height <= 0 {
		return
	}
	before := cloneDenseCellRows(rows, len(rows), damageScreenWidth(rows, nil))
	for row := op.Rect.Y; row < op.Rect.Y+op.Rect.Height && row < len(rows); row++ {
		for col := op.Rect.X; col < op.Rect.X+op.Rect.Width && col < len(rows[row]); col++ {
			srcX := col - op.Dx
			srcY := row - op.Dy
			if srcX >= op.Rect.X && srcX < op.Rect.X+op.Rect.Width && srcY >= op.Rect.Y && srcY < op.Rect.Y+op.Rect.Height && srcY >= 0 && srcY < len(before) && srcX >= 0 && srcX < len(before[srcY]) {
				rows[row][col] = before[srcY][srcX]
				continue
			}
			rows[row][col] = Cell{Content: " ", Width: 1}
		}
	}
}

func applyDamageCopyRect(rows [][]Cell, op DamageOp) {
	if op.Src.Width <= 0 || op.Src.Height <= 0 {
		return
	}
	before := cloneDenseCellRows(rows, len(rows), damageScreenWidth(rows, nil))
	for row := 0; row < op.Src.Height; row++ {
		srcY := op.Src.Y + row
		dstY := op.DstY + row
		if srcY < 0 || srcY >= len(before) || dstY < 0 || dstY >= len(rows) {
			continue
		}
		for col := 0; col < op.Src.Width; col++ {
			srcX := op.Src.X + col
			dstX := op.DstX + col
			if srcX < 0 || srcX >= len(before[srcY]) || dstX < 0 || dstX >= len(rows[dstY]) {
				continue
			}
			rows[dstY][dstX] = before[srcY][srcX]
		}
	}
}

func applyDamageClearRect(rows [][]Cell, op DamageOp) {
	if op.Rect.Width <= 0 || op.Rect.Height <= 0 {
		return
	}
	for row := op.Rect.Y; row < op.Rect.Y+op.Rect.Height && row < len(rows); row++ {
		for col := op.Rect.X; col < op.Rect.X+op.Rect.Width && col < len(rows[row]); col++ {
			rows[row][col] = Cell{Content: " ", Width: 1}
		}
	}
}

func cellRowAt(rows [][]Cell, row int) []Cell {
	if row < 0 || row >= len(rows) {
		return nil
	}
	return rows[row]
}

func cellAt(row []Cell, col int) Cell {
	if col < 0 || col >= len(row) {
		return Cell{Content: " ", Width: 1}
	}
	return row[col]
}

func startsInsideWideCell(row []Cell, col int) bool {
	return col > 0 && col < len(row) && isWideContinuationCell(row[col])
}

func isWideContinuationCell(cell Cell) bool {
	return cell.Content == "" && cell.Width == 0
}

func hashCellFingerprint(hash *uint64, cell *uv.Cell) bool {
	content := ""
	width := 0
	var fg color.Color
	var bg color.Color
	var attrs uint8
	var underline uv.Underline
	link := uv.Link{}
	if cell != nil {
		content = cell.Content
		width = cell.Width
		fg = cell.Style.Fg
		bg = cell.Style.Bg
		attrs = cell.Style.Attrs
		underline = cell.Style.Underline
		link = cell.Link
	}

	bold := attrs&uv.AttrBold != 0
	italic := attrs&uv.AttrItalic != 0
	underlined := underline != 0
	blink := attrs&uv.AttrBlink != 0
	reverse := attrs&uv.AttrReverse != 0
	strikethrough := attrs&uv.AttrStrikethrough != 0

	hashString(hash, content)
	hashUint64(hash, uint64(width))
	hashBool(hash, bold)
	hashBool(hash, italic)
	hashBool(hash, underlined)
	hashBool(hash, blink)
	hashBool(hash, reverse)
	hashBool(hash, strikethrough)
	hashColorFingerprint(hash, fg)
	hashColorFingerprint(hash, bg)
	hashString(hash, link.URL)
	hashString(hash, link.Params)

	return strings.TrimSpace(content) == "" &&
		link.IsZero() &&
		fg == nil &&
		bg == nil &&
		!bold &&
		!italic &&
		!underlined &&
		!blink &&
		!reverse &&
		!strikethrough
}

func hashColorFingerprint(hash *uint64, value color.Color) {
	if value == nil {
		hashUint64(hash, 0)
		return
	}
	switch colorValue := value.(type) {
	case ansi.BasicColor:
		hashUint64(hash, 1)
		hashUint64(hash, uint64(colorValue))
	case ansi.IndexedColor:
		hashUint64(hash, 2)
		hashUint64(hash, uint64(colorValue))
	default:
		r, g, b, _ := value.RGBA()
		hashUint64(hash, 3)
		hashUint64(hash, uint64(uint8(r>>8)))
		hashUint64(hash, uint64(uint8(g>>8)))
		hashUint64(hash, uint64(uint8(b>>8)))
	}
}

func hashString(hash *uint64, value string) {
	hashUint64(hash, uint64(len(value)))
	for i := 0; i < len(value); i++ {
		*hash ^= uint64(value[i])
		*hash *= rowFingerprintPrime64
	}
}

func hashBool(hash *uint64, value bool) {
	if value {
		hashUint64(hash, 1)
		return
	}
	hashUint64(hash, 0)
}

func hashUint64(hash *uint64, value uint64) {
	*hash ^= value
	*hash *= rowFingerprintPrime64
}

func cloneTimeSlice(values []time.Time) []time.Time {
	if len(values) == 0 {
		return nil
	}
	return append([]time.Time(nil), values...)
}

func reuseTimeSlice(dst []time.Time, size int) []time.Time {
	if size <= 0 {
		return nil
	}
	if cap(dst) < size {
		return make([]time.Time, size)
	}
	return dst[:size]
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}

func reuseStringSlice(dst []string, size int) []string {
	if size <= 0 {
		return nil
	}
	if cap(dst) < size {
		return make([]string, size)
	}
	return dst[:size]
}

func reuseRowFingerprintSlice(dst, src []rowFingerprint) []rowFingerprint {
	if len(src) == 0 {
		return nil
	}
	if cap(dst) < len(src) {
		dst = make([]rowFingerprint, len(src))
	} else {
		dst = dst[:len(src)]
	}
	copy(dst, src)
	return dst
}

func cloneCellSlice(values []Cell) []Cell {
	if len(values) == 0 {
		return nil
	}
	return append([]Cell(nil), values...)
}

func pointerAt[T any](values []*T, idx int) *T {
	if idx < 0 || idx >= len(values) {
		return nil
	}
	return values[idx]
}

func cloneCellStylePointer(style *CellStyle) *CellStyle {
	if style == nil {
		return nil
	}
	cloned := *style
	return &cloned
}

func cellStylePointerEqual(left *CellStyle, right *CellStyle) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func hashCellStylePointerFingerprint(hash *uint64, style *CellStyle) {
	if style == nil {
		hashUint64(hash, 0)
		return
	}
	hashUint64(hash, 1)
	hashString(hash, style.FG)
	hashString(hash, style.BG)
	hashBool(hash, style.Bold)
	hashBool(hash, style.Italic)
	hashBool(hash, style.Underline)
	hashBool(hash, style.Blink)
	hashBool(hash, style.Reverse)
	hashBool(hash, style.Strikethrough)
	hashString(hash, style.LinkURL)
	hashString(hash, style.LinkParams)
}

func normalizeTimeSlice(values []time.Time, count int) []time.Time {
	if count <= 0 {
		return nil
	}
	out := make([]time.Time, count)
	copy(out, values)
	return out
}

func normalizeStringSlice(values []string, count int) []string {
	if count <= 0 {
		return nil
	}
	out := make([]string, count)
	copy(out, values)
	return out
}

func normalizeBoolSlice(values []bool, count int) []bool {
	if count <= 0 {
		return nil
	}
	out := make([]bool, count)
	copy(out, values)
	return out
}

func stringAt(values []string, idx int) string {
	if idx < 0 || idx >= len(values) {
		return ""
	}
	return values[idx]
}

func timeAt(values []time.Time, idx int) time.Time {
	if idx < 0 || idx >= len(values) {
		return time.Time{}
	}
	return values[idx]
}

func boolAt(values []bool, idx int) bool {
	return idx >= 0 && idx < len(values) && values[idx]
}

func tailTimeSlice(values []time.Time, trim int) []time.Time {
	if trim <= 0 {
		return values
	}
	if trim >= len(values) {
		return nil
	}
	return values[trim:]
}

func tailStringSlice(values []string, trim int) []string {
	if trim <= 0 {
		return values
	}
	if trim >= len(values) {
		return nil
	}
	return values[trim:]
}

func shouldAssignTimestampToRowFingerprint(row rowFingerprint, rowIndex, cursorRow int) bool {
	if !rowFingerprintIsBlank(row) {
		return true
	}
	return rowIndex >= 0 && rowIndex <= cursorRow
}

func (v *VTerm) alignScrollbackMetadataLocked() {
	if v.emu == nil {
		v.scrollbackTimestamps = nil
		v.scrollbackRowKinds = nil
		v.scrollbackOwnership = nil
		return
	}
	alignLen := v.emu.ScrollbackLen()
	switch {
	case alignLen <= 0:
		v.scrollbackTimestamps = nil
		v.scrollbackRowKinds = nil
		v.scrollbackOwnership = nil
	case alignLen < len(v.scrollbackTimestamps):
		v.scrollbackTimestamps = append([]time.Time(nil), v.scrollbackTimestamps[len(v.scrollbackTimestamps)-alignLen:]...)
	case alignLen > len(v.scrollbackTimestamps):
		v.scrollbackTimestamps = append(v.scrollbackTimestamps, make([]time.Time, alignLen-len(v.scrollbackTimestamps))...)
	}
	switch {
	case alignLen <= 0:
		v.scrollbackRowKinds = nil
	case alignLen < len(v.scrollbackRowKinds):
		v.scrollbackRowKinds = append([]string(nil), v.scrollbackRowKinds[len(v.scrollbackRowKinds)-alignLen:]...)
	case alignLen > len(v.scrollbackRowKinds):
		v.scrollbackRowKinds = append(v.scrollbackRowKinds, make([]string, alignLen-len(v.scrollbackRowKinds))...)
	}
	switch {
	case alignLen <= 0:
		v.scrollbackOwnership = nil
	case alignLen < len(v.scrollbackOwnership):
		v.scrollbackOwnership = append([]string(nil), v.scrollbackOwnership[len(v.scrollbackOwnership)-alignLen:]...)
	case alignLen > len(v.scrollbackOwnership):
		v.scrollbackOwnership = append(v.scrollbackOwnership, make([]string, alignLen-len(v.scrollbackOwnership))...)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func normalizeRenderableUTF8(data []byte) []byte {
	if len(data) == 0 || bytes.IndexByte(data, 0x1b) >= 0 || !utf8.Valid(data) {
		return data
	}

	normalized := norm.NFC.Bytes(data)
	if bytes.Equal(normalized, data) {
		return data
	}
	return normalized
}

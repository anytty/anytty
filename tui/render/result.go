package render

import (
	"strconv"
	"strings"
	"sync"

	actiondomain "github.com/anytty/anytty/tui/action"
	xansi "github.com/charmbracelet/x/ansi"
)

// ANSIReset 是 FrameSink 写完 styled frame 后必须输出的 SGR reset。
const ANSIReset = "\x1b[0m"

type CursorShape string

const (
	CursorShapeBlock CursorShape = "block"
	CursorShapeBar   CursorShape = "bar"
)

type Cursor struct {
	Visible bool
	Anchor  bool
	Row     int
	Col     int
	Shape   CursorShape
}

type RenderMetadata struct {
	Width            int
	Height           int
	ForceFullRepaint bool
}

type Cell struct {
	Text       string
	Width      int
	Style      StyleToken
	ANSIStyle  ANSICellStyle
	LinkURL    string
	LinkParams string
	// TerminalContent 标记来自 core-v2 protocol/live/history 的真实 terminal cell。
	TerminalContent bool
	Safe            bool
}

// ANSICellStyle 保留真实 terminal 内容的 SGR 语义；不要映射到 AnyTTY chrome theme token。
type ANSICellStyle struct {
	FG            string
	BG            string
	Bold          bool
	Italic        bool
	Underline     bool
	Blink         bool
	Reverse       bool
	Strikethrough bool
}

func (style ANSICellStyle) IsZero() bool {
	return style == ANSICellStyle{}
}

const ansiCellStyleCacheLimit = 32768

var ansiCellStyleCache = struct {
	sync.RWMutex
	values map[ANSICellStyle]string
}{values: make(map[ANSICellStyle]string)}

type StyleToken string

const (
	StyleAccent              StyleToken = "accent"
	StyleForeground          StyleToken = "foreground"
	StyleStrongForeground    StyleToken = "strong-foreground"
	StyleMuted               StyleToken = "muted"
	StyleStatus              StyleToken = "status"
	StyleStatusAccent        StyleToken = "status-accent"
	StyleStatusMuted         StyleToken = "status-muted"
	StyleStatusWarning       StyleToken = "status-warning"
	StyleHeaderWorkspace     StyleToken = "header-workspace"
	StyleHeaderWorkspaceEdge StyleToken = "header-workspace-edge"
	StyleHeaderSpacer        StyleToken = "header-spacer"
	StyleHeaderInactiveIndex StyleToken = "header-inactive-index"
	StyleHeaderInactiveTitle StyleToken = "header-inactive-title"
	StyleHeaderInactiveClose StyleToken = "header-inactive-close"
	StyleHeaderActiveEdge    StyleToken = "header-active-edge"
	StyleHeaderActiveMarker  StyleToken = "header-active-marker"
	StyleHeaderActiveIndex   StyleToken = "header-active-index"
	StyleHeaderActiveTitle   StyleToken = "header-active-title"
	StyleHeaderActiveClose   StyleToken = "header-active-close"
	StyleHeaderCreate        StyleToken = "header-create"
	StyleFooterChrome        StyleToken = "footer-chrome"
	StyleFooterMuted         StyleToken = "footer-muted"
	StyleFooterAccent        StyleToken = "footer-accent"
	StyleFooterKeyPane       StyleToken = "footer-key-pane"
	StyleFooterKeyResize     StyleToken = "footer-key-resize"
	StyleFooterKeyTab        StyleToken = "footer-key-tab"
	StyleFooterKeyWorkspace  StyleToken = "footer-key-workspace"
	StyleFooterKeyFloat      StyleToken = "footer-key-float"
	StyleFooterKeyCopy       StyleToken = "footer-key-copy"
	StyleFooterKeyPicker     StyleToken = "footer-key-picker"
	StyleFooterKeyGlobal     StyleToken = "footer-key-global"
	StyleInfo                StyleToken = "info"
	StyleSuccess             StyleToken = "success"
	StyleWarning             StyleToken = "warning"
	StyleHistoryBorder       StyleToken = "history-border"
	StyleSearchMatch         StyleToken = "search-match"
	StyleSearchCurrent       StyleToken = "search-current"
	StyleDanger              StyleToken = "danger"
	StyleDangerStrong        StyleToken = "danger-strong"
	StyleOverlay             StyleToken = "overlay"
	StylePicker              StyleToken = "picker"
	StylePickerMuted         StyleToken = "picker-muted"
	StylePickerAccent        StyleToken = "picker-accent"
	StylePickerInfo          StyleToken = "picker-info"
	StylePickerSuccess       StyleToken = "picker-success"
	StylePickerMatch         StyleToken = "picker-match"
	StylePromptSuggestion    StyleToken = "prompt-suggestion"
	StylePromptSuggestionHit StyleToken = "prompt-suggestion-hit"
	StyleToast               StyleToken = "toast"
	StyleToastAccent         StyleToken = "toast-accent"
)

type Line struct {
	Cells     []Cell
	FillStyle StyleToken
}

func NewLine(text string) Line {
	return Line{Cells: []Cell{NewCell(text)}}
}

func NewCell(text string) Cell {
	return Cell{
		Text:  SafeLine(text),
		Width: DisplayWidth(text),
		Safe:  true,
	}
}

func (line Line) String() string {
	if len(line.Cells) == 0 {
		return ""
	}
	var out strings.Builder
	for _, cell := range line.Cells {
		out.WriteString(cellDisplayText(cell))
	}
	return out.String()
}

func (line Line) PlainString() string {
	return xansi.Strip(line.String())
}

func (line Line) ANSIString(theme Theme) string {
	return line.ansiString(theme, 1)
}

func (line Line) ansiString(theme Theme, baseColumn int) string {
	if len(line.Cells) == 0 {
		return ""
	}
	var out strings.Builder
	if baseColumn < 1 {
		baseColumn = 1
	}
	out.Grow(lineANSICapacity(line, baseColumn))
	modelCol := baseColumn
	for index, cell := range line.Cells {
		writeANSIStyledCell(&out, cell, theme, modelCol)
		modelCol += maxInt(0, cell.Width)
		if index < len(line.Cells)-1 {
			// 中文说明：真实 TTY 对 emoji/FE0F 的列宽可能与模型不同；每个 cell 边界按模型列复位。
			out.WriteString(ansiColumn(modelCol))
		}
	}
	return out.String()
}

func lineANSICapacity(line Line, baseColumn int) int {
	capacity := len(ANSIReset)
	modelCol := baseColumn
	for index, cell := range line.Cells {
		capacity += ansiStyledCellCapacity(cell)
		modelCol += maxInt(0, cell.Width)
		if index < len(line.Cells)-1 {
			capacity += ansiColumnCapacity(modelCol)
		}
	}
	return capacity
}

func writeANSIStyledCell(out *strings.Builder, cell Cell, theme Theme, modelCol int) {
	styleSeq := ""
	if !cell.ANSIStyle.IsZero() {
		styleSeq = ansiForCellStyle(cell.ANSIStyle)
	} else if cell.Style != "" {
		styleSeq = ansiForStyleToken(cell.Style, theme)
	}
	linkOpen, linkClose := ansiLinkOpenClose(cell.LinkURL, cell.LinkParams)
	if linkOpen != "" {
		out.WriteString(linkOpen)
	}
	if styleSeq != "" {
		out.WriteString(styleSeq)
	}
	writeANSIText(out, cell, modelCol)
	if styleSeq != "" {
		out.WriteString(ANSIReset)
	}
	if linkClose != "" {
		out.WriteString(linkClose)
	}
}

func ansiStyledCellCapacity(cell Cell) int {
	capacity := ansiTextCapacity(cell)
	if !cell.ANSIStyle.IsZero() {
		capacity += 32 + len(cell.ANSIStyle.FG) + len(cell.ANSIStyle.BG)
	} else if cell.Style != "" {
		capacity += 40
	}
	if !cell.ANSIStyle.IsZero() || cell.Style != "" {
		capacity += len(ANSIReset)
	}
	if cell.LinkURL != "" {
		capacity += 24 + len(cell.LinkURL) + len(cell.LinkParams)
	}
	return capacity
}

func ansiTextCapacity(cell Cell) int {
	textLen := len(cell.Text)
	pad := maxInt(0, cell.Width) - textLen
	if pad > 0 {
		textLen += pad
	}
	if cell.TerminalContent && cell.Width > 1 && strings.Contains(cell.Text, "\ufe0f") {
		textLen += cell.Width * 8
	}
	return textLen
}

func writeANSIText(out *strings.Builder, cell Cell, startModelCol int) {
	text := cellDisplayText(cell)
	if !cell.TerminalContent || cell.Width <= 1 || !strings.Contains(text, "\ufe0f") {
		out.WriteString(text)
		return
	}
	modelCol := startModelCol
	for len(text) > 0 {
		cluster, width := xansi.FirstGraphemeCluster(text, xansi.GraphemeWidth)
		if cluster == "" {
			break
		}
		text = text[len(cluster):]
		out.WriteString(cluster)
		modelWidth := width
		if modelWidth < 0 {
			modelWidth = 0
		}
		eraseContinuation := terminalFE0FContinuationErase(cluster)
		if eraseContinuation && modelWidth < 2 {
			modelWidth = 2
		}
		if modelWidth == 0 {
			continue
		}
		modelCol += modelWidth
		if eraseContinuation {
			// 中文说明：只在 terminal/protocol cell 的 FE0F footprint 边界做 TTY 写出保护；
			// 先清模型 continuation 物理格，再锚回下一模型列，避免后续内容或边框漂移。
			out.WriteString(ansiEraseChars(1))
			if text != "" {
				out.WriteString(ansiColumn(modelCol))
			}
		}
	}
}

func cellDisplayText(cell Cell) string {
	text := SafeLine(cell.Text)
	width := maxInt(0, cell.Width)
	if width <= 0 {
		return text
	}
	pad := width - DisplayWidth(text)
	if pad <= 0 {
		return text
	}
	return text + strings.Repeat(" ", pad)
}

func terminalFE0FContinuationErase(text string) bool {
	return strings.Contains(text, "\ufe0f") && !strings.Contains(text, "\u200d") && !strings.Contains(text, "\u20e3")
}

func ansiEraseChars(count int) string {
	if count <= 0 {
		return ""
	}
	return "\x1b[" + strconv.Itoa(count) + "X"
}

func ansiColumn(col int) string {
	if col < 1 {
		col = 1
	}
	return "\x1b[" + strconv.Itoa(col) + "G"
}

func ansiColumnCapacity(col int) int {
	if col < 1 {
		col = 1
	}
	return 3 + decimalDigits(col)
}

func decimalDigits(value int) int {
	if value < 0 {
		value = -value
	}
	digits := 1
	for value >= 10 {
		value /= 10
		digits++
	}
	return digits
}

func (line Line) Clone() Line {
	if len(line.Cells) == 0 {
		return Line{FillStyle: line.FillStyle}
	}
	cells := make([]Cell, len(line.Cells))
	copy(cells, line.Cells)
	return Line{Cells: cells, FillStyle: line.FillStyle}
}

func (line Line) Width() int {
	width := 0
	for _, cell := range line.Cells {
		width += maxInt(0, cell.Width)
	}
	return width
}

type LayerKind string

const (
	LayerBase     LayerKind = "base"
	LayerPanel    LayerKind = "panel"
	LayerFloating LayerKind = "floating"
	LayerChrome   LayerKind = "chrome"
	LayerOverlay  LayerKind = "overlay"
	LayerPopup    LayerKind = "popup"
	LayerToast    LayerKind = "toast"
)

type Layer struct {
	Kind            LayerKind
	Rect            Rect
	Lines           []Line
	ContentOverflow ContentOverflow
}

type RenderResult struct {
	Content     []Line
	Cursor      Cursor
	CursorRect  Rect
	Blink       bool
	HitRegions  []HitRegion
	LiveTargets []LiveRenderTarget
	Metadata    RenderMetadata
	Layers      []Layer
	Theme       Theme
}

func (result RenderResult) Lines() []string {
	if len(result.Content) == 0 {
		return nil
	}
	lines := make([]string, len(result.Content))
	for i, line := range result.Content {
		lines[i] = line.PlainString()
	}
	return lines
}

func (result RenderResult) StyledLines() []Line {
	if len(result.Content) == 0 {
		return nil
	}
	lines := make([]Line, len(result.Content))
	for i, line := range result.Content {
		lines[i] = line.Clone()
	}
	return lines
}

func (result RenderResult) ANSILines() []string {
	if len(result.Content) == 0 {
		return nil
	}
	lines := make([]string, len(result.Content))
	theme := result.Theme.WithFallback()
	for i, line := range result.Content {
		lines[i] = ensureANSIReset(line.ANSIString(theme))
	}
	return lines
}

func (result RenderResult) Frame() Frame {
	return FrameFromRenderResult(result)
}

func ensureANSIReset(value string) string {
	if strings.HasSuffix(value, ANSIReset) || strings.HasSuffix(value, "\x1b[m") {
		return value
	}
	return value + ANSIReset
}

func ansiForStyleToken(token StyleToken, theme Theme) string {
	theme = theme.WithFallback()
	switch token {
	case StyleAccent:
		return sgrForeground(theme.Accent, true)
	case StyleForeground:
		return sgrForeground(theme.ChromeFG, false)
	case StyleStrongForeground:
		return sgrForeground(theme.ChromeFG, true)
	case StyleMuted:
		return sgrForeground(theme.Muted, false) + "\x1b[2m"
	case StyleStatus:
		return sgrForegroundBackground(theme.StatusFG, theme.StatusBG, false)
	case StyleStatusAccent:
		return sgrForegroundBackground(theme.Accent, theme.StatusBG, true)
	case StyleStatusMuted:
		return sgrForegroundBackground(theme.Muted, theme.StatusBG, false) + "\x1b[2m"
	case StyleStatusWarning:
		return sgrForegroundBackground(theme.Warning, theme.StatusBG, false)
	case StyleHeaderWorkspace:
		return sgrForegroundBackground(headerWorkspaceFG(theme), headerWorkspaceBG(theme), true)
	case StyleHeaderWorkspaceEdge:
		return sgrForegroundBackground(headerWorkspaceBG(theme), theme.StatusBG, false)
	case StyleHeaderSpacer:
		return sgrForegroundBackground(theme.StatusFG, theme.StatusBG, false)
	case StyleHeaderInactiveIndex:
		return sgrForegroundBackground(headerInactiveIndexFG(theme), theme.StatusBG, false)
	case StyleHeaderInactiveTitle:
		return sgrForegroundBackground(theme.InactivePane, theme.StatusBG, false)
	case StyleHeaderInactiveClose:
		return sgrForegroundBackground(headerInactiveCloseFG(theme), theme.StatusBG, false)
	case StyleHeaderActiveEdge:
		return sgrForegroundBackground(headerActiveBG(theme), theme.StatusBG, false)
	case StyleHeaderActiveMarker:
		return sgrForegroundBackground(theme.Accent, headerActiveBG(theme), true)
	case StyleHeaderActiveIndex:
		return sgrForegroundBackground(theme.StatusFG, headerActiveBG(theme), true)
	case StyleHeaderActiveTitle:
		return sgrForegroundBackground(headerActiveFG(theme), headerActiveBG(theme), true)
	case StyleHeaderActiveClose:
		return sgrForegroundBackground(theme.Danger, headerActiveBG(theme), false)
	case StyleHeaderCreate:
		return sgrForegroundBackground(headerCreateFG(theme), headerCreateBG(theme), true)
	case StyleFooterChrome:
		return sgrForeground(theme.Accent, false)
	case StyleFooterMuted:
		return sgrForeground(theme.Muted, false)
	case StyleFooterAccent:
		return sgrForeground(theme.Accent, true)
	case StyleFooterKeyPane:
		return sgrForeground(theme.Accent, true)
	case StyleFooterKeyResize:
		return sgrForeground(theme.Warning, true)
	case StyleFooterKeyTab:
		return sgrForeground(theme.Info, true)
	case StyleFooterKeyWorkspace:
		return sgrForeground(theme.Success, true)
	case StyleFooterKeyFloat:
		return sgrForeground(mixHostColor(theme.Accent, theme.Info, 0.45), true)
	case StyleFooterKeyCopy:
		return sgrForeground(mixHostColor(theme.Success, theme.Info, 0.35), true)
	case StyleFooterKeyPicker:
		return sgrForeground(mixHostColor(theme.Danger, theme.Warning, 0.35), true)
	case StyleFooterKeyGlobal:
		return sgrForeground(mixHostColor(theme.Accent, theme.Warning, 0.35), true)
	case StyleInfo:
		return sgrForeground(theme.Info, false)
	case StyleSuccess:
		return sgrForeground(theme.Success, false)
	case StyleWarning:
		return sgrForeground(theme.Warning, false)
	case StyleHistoryBorder:
		return sgrForeground(theme.Warning, true)
	case StyleSearchMatch:
		return sgrForeground(theme.Warning, false) + "\x1b[2m\x1b[4m"
	case StyleSearchCurrent:
		return sgrForegroundBackground(theme.HostBG, theme.Warning, true)
	case StyleDanger:
		return sgrForeground(theme.Danger, false)
	case StyleDangerStrong:
		return sgrForeground(theme.Danger, true)
	case StyleOverlay:
		return sgrForegroundBackground(theme.ChromeFG, theme.OverlayBG, false)
	case StylePicker:
		return sgrForeground(theme.ChromeFG, false)
	case StylePickerMuted:
		return sgrForeground(theme.Muted, false) + "\x1b[2m"
	case StylePickerAccent:
		return sgrForeground(theme.Accent, true)
	case StylePickerInfo:
		return sgrForeground(theme.Info, true)
	case StylePickerSuccess:
		return sgrForeground(theme.Success, true)
	case StylePickerMatch:
		return sgrForeground(theme.Warning, true)
	case StylePromptSuggestion:
		return sgrForegroundBackground(theme.ChromeFG, promptSuggestionBG(theme), false)
	case StylePromptSuggestionHit:
		return sgrForegroundBackground(theme.ChromeFG, promptSuggestionHitBG(theme), true)
	case StyleToast:
		return sgrForegroundBackground(theme.ChromeFG, theme.ToastBG, false)
	case StyleToastAccent:
		return sgrForegroundBackground(theme.Accent, theme.ToastBG, true)
	default:
		if _, _, _, ok := parseHexColor(string(token)); ok {
			return sgrForeground(string(token), false)
		}
		return "\x1b[1m"
	}
}

func promptSuggestionBG(theme Theme) string {
	return mixHostColor(theme.HostBG, theme.Accent, 0.16)
}

func promptSuggestionHitBG(theme Theme) string {
	return mixHostColor(theme.HostBG, theme.Accent, 0.34)
}

func headerWorkspaceBG(theme Theme) string {
	return mixHostColor(headerChromeAltBG(theme), theme.Accent, 0.24)
}

func headerWorkspaceFG(theme Theme) string {
	return mixHostColor(theme.StatusFG, theme.Accent, 0.30)
}

func headerActiveBG(theme Theme) string {
	return mixHostColor(mixHostColor(theme.StatusBG, theme.StatusFG, 0.06), theme.Accent, 0.14)
}

func headerActiveFG(theme Theme) string {
	return theme.StatusFG
}

func headerInactiveIndexFG(theme Theme) string {
	return mixHostColor(theme.InactivePane, theme.Accent, 0.22)
}

func headerInactiveCloseFG(theme Theme) string {
	return mixHostColor(theme.Danger, theme.InactivePane, 0.62)
}

func headerCreateBG(theme Theme) string {
	return mixHostColor(headerChromeAltBG(theme), theme.Info, 0.18)
}

func headerCreateFG(theme Theme) string {
	return theme.StatusFG
}

func headerChromeAltBG(theme Theme) string {
	return mixHostColor(theme.StatusBG, theme.StatusFG, 0.08)
}

func ansiForCellStyle(style ANSICellStyle) string {
	if style.IsZero() {
		return ""
	}
	ansiCellStyleCache.RLock()
	cached, ok := ansiCellStyleCache.values[style]
	ansiCellStyleCache.RUnlock()
	if ok {
		return cached
	}
	seq := buildANSICellStyle(style)
	ansiCellStyleCache.Lock()
	if cached, ok := ansiCellStyleCache.values[style]; ok {
		ansiCellStyleCache.Unlock()
		return cached
	}
	if len(ansiCellStyleCache.values) >= ansiCellStyleCacheLimit {
		// 中文说明：terminal truecolor 可能是无界输入；缓存只服务重复样式热路径，到上限整体释放。
		ansiCellStyleCache.values = make(map[ANSICellStyle]string)
	}
	ansiCellStyleCache.values[style] = seq
	ansiCellStyleCache.Unlock()
	return seq
}

func buildANSICellStyle(style ANSICellStyle) string {
	var out strings.Builder
	if style.Bold {
		appendSGRParam(&out, "1")
	}
	if style.Italic {
		appendSGRParam(&out, "3")
	}
	if style.Underline {
		appendSGRParam(&out, "4")
	}
	if style.Blink {
		appendSGRParam(&out, "5")
	}
	if style.Reverse {
		appendSGRParam(&out, "7")
	}
	if style.Strikethrough {
		appendSGRParam(&out, "9")
	}
	appendANSIColorParams(&out, style.FG, true)
	appendANSIColorParams(&out, style.BG, false)
	if out.Len() == 0 {
		return ""
	}
	return "\x1b[" + out.String() + "m"
}

func ansiLinkOpenClose(linkURL string, linkParams string) (string, string) {
	linkURL = sanitizeOSC8Field(linkURL)
	if linkURL == "" {
		return "", ""
	}
	linkParams = sanitizeOSC8Field(linkParams)
	return "\x1b]8;" + linkParams + ";" + linkURL + "\x1b\\", "\x1b]8;;\x1b\\"
}

func sanitizeOSC8Field(value string) string {
	value = strings.ReplaceAll(value, "\x1b", "")
	value = strings.ReplaceAll(value, "\x07", "")
	value = strings.ReplaceAll(value, "\x9c", "")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "\r", "")
	return value
}

func appendSGRParam(out *strings.Builder, value string) {
	if value == "" {
		return
	}
	if out.Len() > 0 {
		out.WriteByte(';')
	}
	out.WriteString(value)
}

func appendANSIColorParams(out *strings.Builder, value string, foreground bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if code, ok := ansiPaletteColorCode(value, foreground); ok {
		appendSGRParam(out, strconv.Itoa(code))
		return
	}
	if index, ok := ansiIndexedColor(value); ok {
		prefix := "38"
		if !foreground {
			prefix = "48"
		}
		// 中文说明：terminal cell SGR 是 render 热路径，直接拼参数，避免每格分配 []string 后 Join。
		appendSGRParam(out, prefix)
		appendSGRParam(out, "5")
		appendSGRParam(out, strconv.Itoa(index))
		return
	}
	r, g, b, ok := parseHexColor(value)
	if !ok {
		return
	}
	prefix := "38"
	if !foreground {
		prefix = "48"
	}
	appendSGRParam(out, prefix)
	appendSGRParam(out, "2")
	appendSGRParam(out, r)
	appendSGRParam(out, g)
	appendSGRParam(out, b)
}

func ansiPaletteColorCode(value string, foreground bool) (int, bool) {
	if !strings.HasPrefix(value, "ansi:") {
		return 0, false
	}
	index, err := strconv.Atoi(strings.TrimPrefix(value, "ansi:"))
	if err != nil || index < 0 || index > 15 {
		return 0, false
	}
	if index < 8 {
		if foreground {
			return 30 + index, true
		}
		return 40 + index, true
	}
	if foreground {
		return 90 + (index - 8), true
	}
	return 100 + (index - 8), true
}

func ansiIndexedColor(value string) (int, bool) {
	if !strings.HasPrefix(value, "idx:") {
		return 0, false
	}
	index, err := strconv.Atoi(strings.TrimPrefix(value, "idx:"))
	if err != nil || index < 0 || index > 255 {
		return 0, false
	}
	return index, true
}

func sgrForeground(hex string, bold bool) string {
	prefix := "\x1b["
	if bold {
		prefix += "1;"
	}
	r, g, b, ok := parseHexColor(hex)
	if !ok {
		if bold {
			return "\x1b[1m"
		}
		return ""
	}
	return prefix + "38;2;" + r + ";" + g + ";" + b + "m"
}

func sgrForegroundBackground(fg string, bg string, bold bool) string {
	fgSeq := sgrForeground(fg, bold)
	r, g, b, ok := parseHexColor(bg)
	if !ok {
		return fgSeq
	}
	return fgSeq + "\x1b[48;2;" + r + ";" + g + ";" + b + "m"
}

func parseHexColor(value string) (string, string, string, bool) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(value) != 6 {
		return "", "", "", false
	}
	for _, ch := range value {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return "", "", "", false
		}
	}
	return decimalByte(value[0:2]), decimalByte(value[2:4]), decimalByte(value[4:6]), true
}

func decimalByte(hex string) string {
	value := 0
	for _, ch := range hex {
		value *= 16
		switch {
		case ch >= '0' && ch <= '9':
			value += int(ch - '0')
		case ch >= 'a' && ch <= 'f':
			value += int(ch-'a') + 10
		case ch >= 'A' && ch <= 'F':
			value += int(ch-'A') + 10
		}
	}
	return strconv.Itoa(value)
}

type ContentKind string

const (
	ContentTerminalLive     ContentKind = "terminal-live"
	ContentCopyHistory      ContentKind = "copy-history"
	ContentEmptyPane        ContentKind = "empty-pane"
	ContentExitedPane       ContentKind = "exited-pane"
	ContentTerminalPicker   ContentKind = "terminal-picker"
	ContentTerminalPool     ContentKind = "terminal-pool"
	ContentConnections      ContentKind = "connections"
	ContentWorkbenchTree    ContentKind = "workbench-tree"
	ContentClipboardHistory ContentKind = "clipboard-history"
	ContentFloatingOverview ContentKind = "floating-overview"
	ContentPrompt           ContentKind = "prompt"
	ContentHelp             ContentKind = "help"
	ContentPlaceholder      ContentKind = "placeholder"
)

type ContentVM struct {
	Kind       ContentKind
	Lines      []Line
	Search     SearchBarVM
	Extent     ContentExtent
	Layout     ContentLayoutVM
	Meta       ContentMetaVM
	Status     string
	Pending    bool
	Empty      bool
	Error      string
	Cursor     Cursor
	HitRegions []HitRegion
}

// SearchBarVM replaces the global footer while the active panel owns a search session.
// Prefix/value stay left aligned while status and optional hints stay right aligned.
type SearchBarVM struct {
	Visible      bool
	Prefix       Line
	Value        string
	ValueCursor  int
	CursorPlaced bool
	Status       Line
	WideHint     Line
	WideMinWidth int
	Cursor       Cursor
}

// ContentMetaVM 保存渲染层不能从文本行反推的布局元数据；这些值由对应 content projector 生成，
// overlay chrome、hit region 和 snapshot 装饰只能消费这里的 reducer-owned 投影。
type ContentMetaVM struct {
	LiveEndpointID           string
	LiveTerminalID           string
	LiveRevision             uint64
	ClipboardNameWidth       int
	SplitPageLeftWidth       int
	WorkbenchTreeWidth       int
	WorkbenchBodyRows        int
	WorkbenchActionRow       int
	WorkbenchSnapshotPanel   *PanelVM
	WorkbenchSnapshotRect    Rect
	WorkbenchSnapshotContent Rect
	WorkbenchSnapshots       []WorkbenchSnapshotVM
	WorkbenchSnapshotCanvas  Rect
	WorkbenchSnapshotSplit   SplitVM
}

type WorkbenchSnapshotVM struct {
	Panel         PanelVM
	Rect          Rect
	Content       Rect
	PreviewHeader []Line
	Borderless    bool
	PreviewFrame  bool
	ShowTitle     bool
}

// ContentExtent 表达 terminal 内容在 content rect 内实际占用的 cell 区域。
// Known=false 时，renderer 以当前 content rect 作为 extent，避免把普通旧内容误判为 extent 外占位。
type ContentExtent struct {
	Known bool
	X     int
	Y     int
	Cols  int
	Rows  int
}

// ContentLayoutVM 是 view-local terminal 内容布局投影，只影响当前 pane/floating 的裁切与对齐。
type ContentLayoutVM struct {
	Known  bool
	Mode   string
	PanX   int
	PanY   int
	AlignX string
	AlignY string
}

type ContentOverflow struct {
	Left   bool
	Right  bool
	Top    bool
	Bottom bool
}

type ContentRenderRequest struct {
	Rect    Rect
	Content ContentVM
}

type ContentRenderResult struct {
	Lines      []Line
	Cursor     Cursor
	HitRegions []HitRegion
	Metadata   RenderMetadata
	Overflow   ContentOverflow
}

type ContentRenderer interface {
	RenderContent(ContentRenderRequest) ContentRenderResult
}

type PanelPresentation string

const (
	PanelPresentationCard      PanelPresentation = "card"
	PanelPresentationSplitLine PanelPresentation = "split-line"
)

// PanelVM 是 renderer 消费的单个 tiled pane 投影。
// 它只携带 ShellStore/TerminalViewStore 已经归一化后的展示状态；pane identity、
// terminal binding 和 zoom truth 仍由 state reducer 持有，renderer 不能反写。
type PanelVM struct {
	ID           string
	Title        string
	Rect         Rect
	Presentation PanelPresentation
	Active       bool
	// IsZoomMode 表示该 pane 来自 ShellStore.ZoomedPaneID 的 zoom 投影。
	// 它只用于 chrome 条件展示和 action 模板变量，不改变 pane.zoom 的 toggle 命令链路。
	IsZoomMode bool
	Chrome     PanelChromeVM
	Content    ContentVM
}

type PanelChromeVM struct {
	Title    ChromeSlotVM
	State    ChromeSlotVM
	Meta     []ChromeSlotVM
	Terminal TerminalChromeVM
	Actions  []ChromeActionVM
}

type TerminalChromeVM struct {
	Locked       bool
	LayoutMode   string
	PanX         int
	PanY         int
	AlignX       string
	AlignY       string
	Title        ChromeSlotVM
	State        ChromeSlotVM
	AttachCount  int
	Owner        ChromeSlotVM
	TakeOwner    bool
	CanLockSize  bool
	ResizeRole   string
	CanResize    bool
	TerminalID   string
	TerminalView string
}

type ChromeSlotVM struct {
	Text  string
	Style StyleToken
}

// ChromeActionVM 是 pane/floating chrome 上一个可见 action token 的展示投影。
// ActionID 仍是鼠标命中和 reducer 命令分发的唯一语义来源；Text/Label/IsZoomMode
// 只服务 renderer 和用户配置模板，不能创建新的交互语义。
type ChromeActionVM struct {
	Text     string
	ActionID string
	Label    string
	Style    StyleToken
	// IsZoomMode 把所属 pane 的 zoom 状态传给 chrome 模板。
	// 命中区仍绑定 ActionID，例如 zoom 状态下的 unzoom 图标仍使用 pane.zoom toggle。
	IsZoomMode bool
}

type LayoutVM struct {
	Viewport           Rect
	Body               Rect
	ShellFrame         Rect
	HeaderTopFrame     Rect
	HeaderDividerFrame Rect
	FooterFrame        Rect
	Panels             []PanelVM
	BodyContent        ContentVM
	Floating           []FloatingVM
	Split              SplitVM
	ChromePatches      []ChromePatchVM
}

type SplitDirection string

const (
	SplitHorizontal SplitDirection = "horizontal"
	SplitVertical   SplitDirection = "vertical"
)

type SplitVM struct {
	PaneID      string
	Direction   SplitDirection
	Children    []SplitVM
	Ratio       float64
	BiasCells   int
	FixedPaneID string
	FixedCols   int
	FixedRows   int
}

type FloatingVM struct {
	ID        string
	PaneID    string
	Title     string
	Rect      Rect
	Z         int
	Active    bool
	Collapsed bool
	Chrome    FloatingChromeVM
	Content   ContentVM
}

type FloatingChromeVM struct {
	FillOverlay      bool
	ShowResizeHandle bool
	Terminal         TerminalChromeVM
	Actions          []ChromeActionVM
}

type ChromePatchVM struct {
	Anchor ChromePatchAnchor
	X      int
	Y      int
	W      int
	Text   string
	Style  StyleToken
	Layer  LayerKind
	Owner  string
}

type ChromePatchAnchor string

const (
	ChromePatchAnchorViewport ChromePatchAnchor = "viewport"
	ChromePatchAnchorBody     ChromePatchAnchor = "body"
)

type HeaderTabVM struct {
	ID            string
	Title         string
	Index         int
	Active        bool
	CloseActionID string
	CloseTargetID string
}

// HeaderVM 是顶部 workspace/tab chrome 的渲染投影。
// 它只消费 ShellStore 和本地 TUI 配置生成可点击展示片段；workspace/tab 的真实状态、
// tab close/switch 以及 navigator 打开动作仍由 reducer 消息链路持有。
type HeaderVM struct {
	Visible           bool
	Workspace         string
	Tab               string
	Tabs              []HeaderTabVM
	WorkspaceTemplate string
	TabTemplate       string
	TabCreateIcon     string
	TabCreateTemplate string
	ActivePane        string
	TerminalSummary   string
	FloatingSummary   string
	Notice            string
	Title             string
}

type FooterActionVM struct {
	Key        string
	Label      string
	Icon       string
	ActionID   string
	Style      StyleToken
	Invocation actiondomain.Invocation
	Click      ClickPolicy
}

// ClickPolicy 描述具体 render projection 的点击属性。
// 只有携带唯一 canonical invocation 的 projection 才能是 clickable；该属性不属于 action 或 shortcut 全局 spec。
type ClickPolicy string

const (
	ClickHidden    ClickPolicy = "hidden"
	ClickHintOnly  ClickPolicy = "hint-only"
	ClickClickable ClickPolicy = "clickable"
)

type FooterVM struct {
	Visible                          bool
	Search                           SearchBarVM
	Mode                             string
	ModeIcon                         string
	ModeLabel                        string
	ModeStyle                        StyleToken
	Hint                             string
	ActionTokens                     []FooterActionVM
	KeyTemplate                      string
	KeyTemplateSet                   bool
	ActionTemplate                   string
	ModeBadgeTemplate                string
	ActionSeparator                  string
	WorkspaceSummaryTemplate         string
	FloatingSummaryTemplate          string
	FloatingCollapsedSummaryTemplate string
	TerminalsSummaryTemplate         string
	TabsSummaryTemplate              string
	PanesSummaryTemplate             string
	KeylockOnTemplate                string
	KeylockOffTemplate               string
	ActiveTarget                     string
	GlobalSummary                    string
	FloatingSummaryOpen              bool
}

type ToastSeverity string

const (
	ToastInfo    ToastSeverity = "info"
	ToastSuccess ToastSeverity = "success"
	ToastWarning ToastSeverity = "warning"
	ToastError   ToastSeverity = "error"
)

type ToastVM struct {
	ID       string
	Severity ToastSeverity
	Title    string
	Body     string
	Pending  bool
}

type OverlayKind string

const (
	OverlayNone             OverlayKind = ""
	OverlayTerminalPicker   OverlayKind = "terminal-picker"
	OverlayTerminalPool     OverlayKind = "terminal-pool"
	OverlayConnections      OverlayKind = "connections"
	OverlayWorkbenchTree    OverlayKind = "workbench-tree"
	OverlayClipboardHistory OverlayKind = "clipboard-history"
	OverlayFloatingOverview OverlayKind = "floating-overview"
	OverlayPrompt           OverlayKind = "prompt"
	OverlayHelp             OverlayKind = "help"
)

type OverlayVM struct {
	Kind    OverlayKind
	Opaque  bool
	Content ContentVM
	Popup   OverlayPopupVM
}

type ShellVM struct {
	Header  HeaderVM
	Footer  FooterVM
	Layout  LayoutVM
	Overlay OverlayVM
	Toasts  []ToastVM
	Cursor  Cursor
}

type OverlayPopupKind string

const (
	OverlayPopupNone             OverlayPopupKind = ""
	OverlayPopupPromptSuggestion OverlayPopupKind = "prompt-suggestion"
)

type OverlayPopupVM struct {
	Kind      OverlayPopupKind
	AnchorRow int
	AnchorCol int
	Lines     []Line
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

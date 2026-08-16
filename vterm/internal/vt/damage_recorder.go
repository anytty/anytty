package vt

import (
	"image/color"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

type screenDamageRecorder struct {
	scrollbackOnly  bool
	semanticOnly    bool
	lineHistoryOnly bool
	damages         []Damage
	spanCells       []uv.Cell
	textCells       []uv.Cell
	tailSpan        *SpanDamage
	tailSpanStart   int
	tailSpanEnd     int
}

func (r *screenDamageRecorder) record(d Damage) {
	if r == nil || d == nil {
		return
	}
	if r.scrollbackOnly {
		if _, ok := d.(ScrollbackDamage); !ok {
			return
		}
	}
	if r.lineHistoryOnly {
		switch d.(type) {
		case ControlDamage, ModeDamage, ScrollbackDamage:
			// 中文说明：linehist 生产路径只需要 eviction 与边界；
			// 普通 TextDamage/CR/LF 不应进入高压输出 backlog。
		default:
			return
		}
	}
	if r.semanticOnly {
		switch d.(type) {
		case TextDamage, ControlDamage, ModeDamage, ScrollbackDamage, ScrollDamage, MoveDamage, ScreenDamage:
			// 中文说明：semantic-only recorder 是 core-v2 history 的 PTY 语义来源；
			// 它只保留 ordered text/control/mode/scroll proof，不保留 screen diff cell payload。
		default:
			return
		}
	}
	if text, ok := d.(TextDamage); ok {
		r.recordText(text)
		return
	}
	if span, ok := d.(SpanDamage); ok {
		r.recordSpan(span)
		return
	}
	r.tailSpan = nil
	r.damages = append(r.damages, d)
}

func (r *screenDamageRecorder) recordsScreenDiff() bool {
	return r != nil && !r.scrollbackOnly && !r.semanticOnly && !r.lineHistoryOnly
}

func (r *screenDamageRecorder) recordControl(kind string, x int, y int, mode int) {
	if r == nil || r.scrollbackOnly || r.lineHistoryOnly || kind == "" {
		return
	}
	r.record(ControlDamage{Kind: kind, X: x, Y: y, Mode: mode})
}

func (r *screenDamageRecorder) recordControlWithScrollOut(kind string, x int, y int, mode int, scrollOut []ScrollbackDamage) {
	if r == nil || r.scrollbackOnly || kind == "" {
		return
	}
	r.record(ControlDamage{Kind: kind, X: x, Y: y, Mode: mode, ScrollOut: cloneScrollbackDamages(scrollOut)})
}

func (r *screenDamageRecorder) recordScrollRegion(top int, bottom int) {
	if r == nil || r.scrollbackOnly || r.lineHistoryOnly {
		return
	}
	r.record(ControlDamage{Kind: "decstbm", Mode: top, Bottom: bottom})
}

func (r *screenDamageRecorder) recordHorizontalScrollRegion(left int, right int) {
	if r == nil || r.scrollbackOnly || r.lineHistoryOnly {
		return
	}
	r.record(ControlDamage{Kind: "decslrm", Mode: left, Bottom: right})
}

func (r *screenDamageRecorder) recordControlWithCell(kind string, x int, y int, mode int, cell *uv.Cell) {
	if r == nil || r.scrollbackOnly || r.lineHistoryOnly || kind == "" {
		return
	}
	damage := ControlDamage{Kind: kind, X: x, Y: y, Mode: mode}
	if cell != nil {
		damage.Cell = *cell
		damage.HasCell = true
	}
	r.record(damage)
}

func (r *screenDamageRecorder) recordControlWithCellAndBottom(kind string, x int, y int, mode int, bottom int, cell *uv.Cell) {
	if r == nil || r.scrollbackOnly || r.lineHistoryOnly || kind == "" {
		return
	}
	damage := ControlDamage{Kind: kind, X: x, Y: y, Mode: mode, Bottom: bottom}
	if cell != nil {
		damage.Cell = *cell
		damage.HasCell = true
	}
	r.record(damage)
}

func (r *screenDamageRecorder) recordMode(mode int, private bool, enabled bool) {
	if r == nil || r.scrollbackOnly || mode == 0 {
		return
	}
	r.record(ModeDamage{Mode: mode, Private: private, Enabled: enabled})
}

func (r *screenDamageRecorder) recordTextSpan(x, y int, cells []uv.Cell) {
	if r == nil || r.scrollbackOnly || r.lineHistoryOnly || len(cells) == 0 {
		return
	}
	damage := TextDamage{X: x, Y: y}
	if text, ok := compactASCIIPlainLine(cells); ok {
		damage.ASCII = true
		damage.Text = text
	} else if runs, ok := compactASCIIStyleRuns(cells); ok && len(runs) > 0 {
		damage.Runs = runs
	} else {
		damage.Cells = cells
	}
	r.record(damage)
}

func (r *screenDamageRecorder) recordASCIITextRun(x, y int, data []byte, style uv.Style, link uv.Link) {
	if r == nil || r.scrollbackOnly || r.lineHistoryOnly || len(data) == 0 {
		return
	}
	// 中文说明：semantic-only 是 history 的 ordered text proof；普通 ASCII
	// 压力输出不能先膨胀成 uv.Cell 切片再压缩，否则 100K/1M 输出会把 RSS 顶高。
	if !link.IsZero() {
		cells := make([]uv.Cell, len(data))
		for i, b := range data {
			cells[i] = uv.Cell{
				Content: printableASCIIStrings[b],
				Width:   1,
				Style:   style,
				Link:    link,
			}
		}
		r.recordTextSpan(x, y, cells)
		return
	}
	text := string(data)
	if style.IsZero() {
		r.record(TextDamage{X: x, Y: y, ASCII: true, Text: text})
		return
	}
	r.record(TextDamage{X: x, Y: y, Runs: []ScrollbackRun{{Style: style, Text: text}}})
}

func (r *screenDamageRecorder) recordSpanCell(x, y int, cell uv.Cell) {
	if r == nil || !r.recordsScreenDiff() {
		return
	}
	if r.tailSpan != nil && y == r.tailSpan.Y && x == r.tailSpanEnd {
		r.spanCells = append(r.spanCells, cell)
		r.refreshTailSpan()
		r.tailSpanEnd += cellDamageWidth(cell)
		return
	}
	start := len(r.spanCells)
	r.spanCells = append(r.spanCells, cell)
	span := &SpanDamage{X: x, Y: y, Cells: r.spanCells[start:len(r.spanCells)]}
	r.damages = append(r.damages, span)
	r.tailSpan = span
	r.tailSpanStart = start
	r.tailSpanEnd = x + cellDamageWidth(cell)
}

func (r *screenDamageRecorder) recordRepeatedSpan(x, y, count int, cell uv.Cell) {
	if r == nil || !r.recordsScreenDiff() || count <= 0 {
		return
	}
	start := len(r.spanCells)
	for i := 0; i < count; i++ {
		r.spanCells = append(r.spanCells, cell)
	}
	span := &SpanDamage{X: x, Y: y, Cells: r.spanCells[start:len(r.spanCells)]}
	r.damages = append(r.damages, span)
	r.tailSpan = span
	r.tailSpanStart = start
	r.tailSpanEnd = x + spanCellsWidth(r.spanCells[start:len(r.spanCells)])
}

func (r *screenDamageRecorder) recordScrollbackLine(y int, line uv.Line, wrapped bool) {
	if r == nil {
		return
	}
	r.tailSpan = nil
	r.damages = append(r.damages, compactScrollbackDamage(y, line, wrapped))
}

func compactScrollbackDamage(y int, line uv.Line, wrapped bool) ScrollbackDamage {
	if text, ok := compactASCIIPlainLine(line); ok {
		return ScrollbackDamage{Y: y, ASCII: true, Text: text, Wrapped: wrapped}
	}
	if runs, ok := compactASCIIStyleRuns(line); ok {
		return ScrollbackDamage{Y: y, Runs: runs, Wrapped: wrapped}
	}
	cells := make([]uv.Cell, len(line))
	copy(cells, line)
	return ScrollbackDamage{Y: y, Cells: cells, Wrapped: wrapped}
}

func cloneScrollbackDamages(in []ScrollbackDamage) []ScrollbackDamage {
	if len(in) == 0 {
		return nil
	}
	out := make([]ScrollbackDamage, len(in))
	for i, row := range in {
		out[i] = cloneScrollbackDamage(row)
	}
	return out
}

func cloneScrollbackDamage(row ScrollbackDamage) ScrollbackDamage {
	out := row
	if len(row.Cells) > 0 {
		out.Cells = make([]uv.Cell, len(row.Cells))
		copy(out.Cells, row.Cells)
	}
	if len(row.Runs) > 0 {
		out.Runs = make([]ScrollbackRun, len(row.Runs))
		copy(out.Runs, row.Runs)
	}
	return out
}

func (r *screenDamageRecorder) recordSpan(span SpanDamage) {
	if r == nil || !r.recordsScreenDiff() || len(span.Cells) == 0 {
		return
	}
	if r.mergeTrailingSpan(span) {
		return
	}
	start := len(r.spanCells)
	r.spanCells = append(r.spanCells, span.Cells...)
	next := &SpanDamage{
		X:     span.X,
		Y:     span.Y,
		Cells: r.spanCells[start:len(r.spanCells)],
	}
	r.damages = append(r.damages, next)
	r.tailSpan = next
	r.tailSpanStart = start
	r.tailSpanEnd = span.X + spanDamageWidth(span)
}

func (r *screenDamageRecorder) recordText(text TextDamage) {
	if r == nil || r.scrollbackOnly || r.lineHistoryOnly || (len(text.Cells) == 0 && len(text.Runs) == 0 && text.Text == "") {
		return
	}
	if len(text.Runs) > 0 || text.Text != "" {
		r.damages = append(r.damages, text)
		return
	}
	start := len(r.textCells)
	r.textCells = append(r.textCells, text.Cells...)
	r.damages = append(r.damages, TextDamage{
		X:     text.X,
		Y:     text.Y,
		Cells: r.textCells[start:len(r.textCells)],
	})
}

func (r *screenDamageRecorder) mergeTrailingSpan(next SpanDamage) bool {
	if r == nil || len(next.Cells) == 0 || r.tailSpan == nil || next.Y != r.tailSpan.Y || next.X != r.tailSpanEnd {
		return false
	}
	r.spanCells = append(r.spanCells, next.Cells...)
	r.refreshTailSpan()
	r.tailSpanEnd += spanDamageWidth(next)
	return true
}

func (r *screenDamageRecorder) refreshTailSpan() {
	if r == nil || r.tailSpan == nil || r.tailSpanStart < 0 || r.tailSpanStart > len(r.spanCells) {
		return
	}
	r.tailSpan.Cells = r.spanCells[r.tailSpanStart:len(r.spanCells)]
}

func (r *screenDamageRecorder) snapshot() []Damage {
	if r == nil || len(r.damages) == 0 {
		return nil
	}
	out := make([]Damage, len(r.damages))
	for i, damage := range r.damages {
		if span, ok := damage.(*SpanDamage); ok {
			out[i] = *span
			continue
		}
		if text, ok := damage.(TextDamage); ok {
			if len(text.Cells) > 0 {
				cells := make([]uv.Cell, len(text.Cells))
				copy(cells, text.Cells)
				text.Cells = cells
			}
			if len(text.Runs) > 0 {
				runs := make([]ScrollbackRun, len(text.Runs))
				copy(runs, text.Runs)
				text.Runs = runs
			}
			out[i] = text
			continue
		}
		out[i] = damage
	}
	return out
}

func spanDamageWidth(d SpanDamage) int {
	return spanCellsWidth(d.Cells)
}

func spanCellsWidth(cells []uv.Cell) int {
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

func cellDamageWidth(cell uv.Cell) int {
	if cell.Width > 0 {
		return cell.Width
	}
	return 1
}

func cloneDamageCell(cell *uv.Cell) uv.Cell {
	if cell == nil {
		return uv.Cell{Content: " ", Width: 1}
	}
	return *cell
}

func isClearDamageCell(cell *uv.Cell) bool {
	if cell == nil {
		return true
	}
	if cell.Content != "" && cell.Content != " " {
		return false
	}
	if cell.Width != 0 && cell.Width != 1 {
		return false
	}
	if !cell.Style.IsZero() {
		return false
	}
	if !cell.Link.IsZero() {
		return false
	}
	return true
}

func compactASCIIPlainLine(line uv.Line) (string, bool) {
	if len(line) == 0 {
		return "", true
	}
	out := make([]byte, len(line))
	for i := 0; i < len(line); i++ {
		cell := line[i]
		if !cell.Style.IsZero() || !cell.Link.IsZero() || cell.Width != 1 || len(cell.Content) != 1 || cell.Content[0] >= 0x80 {
			return "", false
		}
		out[i] = cell.Content[0]
	}
	return string(out), true
}

func compactASCIIStyleRuns(line uv.Line) ([]ScrollbackRun, bool) {
	if len(line) == 0 {
		return nil, true
	}
	runs := make([]ScrollbackRun, 0, 4)
	var current ScrollbackRun
	var currentText strings.Builder
	currentText.Grow(len(line))
	runStart := 0
	currentStyleSet := false
	for i := 0; i < len(line); i++ {
		cell := line[i]
		if !cell.Link.IsZero() || cell.Width != 1 || len(cell.Content) != 1 || cell.Content[0] >= 0x80 {
			return nil, false
		}
		if currentStyleSet && compactASCIIStyleEqual(current.Style, cell.Style) {
			currentText.WriteByte(cell.Content[0])
			continue
		}
		if currentStyleSet && currentText.Len() > runStart {
			current.Text = currentText.String()[runStart:]
			runs = append(runs, current)
			runStart = currentText.Len()
		}
		current = ScrollbackRun{Style: cell.Style}
		currentStyleSet = true
		currentText.WriteByte(cell.Content[0])
	}
	if currentStyleSet && currentText.Len() > runStart {
		current.Text = currentText.String()[runStart:]
		runs = append(runs, current)
	}
	return runs, true
}

func compactASCIIStyleEqual(a, b uv.Style) bool {
	return a.Attrs == b.Attrs &&
		a.Underline == b.Underline &&
		compactASCIIColorEqual(a.Fg, b.Fg) &&
		compactASCIIColorEqual(a.Bg, b.Bg) &&
		compactASCIIColorEqual(a.UnderlineColor, b.UnderlineColor)
}

func compactASCIIColorEqual(a, b color.Color) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	switch av := a.(type) {
	case ansi.BasicColor:
		bv, ok := b.(ansi.BasicColor)
		return ok && av == bv
	case ansi.IndexedColor:
		bv, ok := b.(ansi.IndexedColor)
		return ok && av == bv
	case ansi.RGBColor:
		bv, ok := b.(ansi.RGBColor)
		return ok && av == bv
	case color.RGBA:
		bv, ok := b.(color.RGBA)
		return ok && av == bv
	case color.NRGBA:
		bv, ok := b.(color.NRGBA)
		return ok && av == bv
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

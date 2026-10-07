package terminal

import (
	"strconv"
	"strings"

	"github.com/anytty/anytty/clients/tui/render"
)

// Render draws the component at its own origin: lines use coordinates
// relative to (0, 0) with X in [0, width) and Y in [0, height). The runtime
// translates them to the placement rect and blits them into the framebuffer.
//
// The chrome follows the v1 terminal component: a titled border, a focus
// marker, an exit badge and a scrollback badge:
//
//	┌─▎main [↑3] ─────┐
//	│ …content…       │
//	└─────────────────┘
//
// The component declares its own inset (Inset(width, height)): the runtime
// derives the PTY winsize and the cursor offset from it. Chrome colors come
// from the program-declared props (content.props: chrome.border / border_focus
// / border_dead for the frame, chrome.title / chrome.badge for the label) and
// fall back to the built-in semantic tokens when a key is absent. Content
// rows keep their own style tokens.
func (c *Component) Render(width, height int) []render.Line {
	if width <= 0 || height <= 0 {
		return nil
	}
	props := c.props.withDefaults()
	inset := c.Inset(width, height)
	bordered := inset == DefaultInset
	contentW, contentH := width-2*inset, height-2*inset
	if contentH < 0 {
		contentH = 0
	}
	if contentW < 0 {
		contentW = 0
	}
	// Remember the visible content height: it is the window size requested
	// from the history port on scroll.
	c.visible = contentH

	border := borderToken(props)
	lines := make([]render.Line, 0, height)
	if bordered {
		lines = append(lines, topBorderLines(width, props, border)...)
	}
	screen := c.visibleScreen()
	framing := contentFramingFromProps(props)
	for row := 0; row < contentH; row++ {
		y := row + inset
		x := inset
		if bordered {
			lines = append(lines, render.Line{X: 0, Y: y, Text: "│", Style: border})
		}
		lines = append(lines, contentRowLines(screen, row, x, y, contentW, contentH, framing)...)
		if bordered {
			lines = append(lines, render.Line{X: width - 1, Y: y, Text: "│", Style: border})
		}
	}
	lines = append(lines, copyOverlayLines(screen, inset, props, contentW, contentH)...)
	if bordered {
		lines = append(lines, render.Line{
			X:     0,
			Y:     height - 1,
			Text:  "└" + strings.Repeat("─", width-2) + "┘",
			Style: border,
		})
	}
	if props.Dimmed {
		for index := range lines {
			lines[index].Style = dimToken(lines[index].Style)
		}
	}
	return lines
}

func dimToken(token render.Token) render.Token {
	if raw, ok := token.RawSGR(); ok {
		if raw == "" {
			return render.Token("ansi:2")
		}
		return render.Token("ansi:" + raw + ";2")
	}
	if style, ok := render.ParseStyle(string(token)); ok {
		style.Dim = true
		return render.Token(style.String())
	}
	// Host semantic tokens do not expose their color here. Muted is the
	// stable gray fallback and keeps the inactive panel legible.
	return render.TokenMuted
}

// copySpan is one inclusive display-cell range on one viewport row.
type copySpan struct {
	row      int
	startCol int
	endCol   int
}

// parseCopySpans parses "row,c1,c2;row,c1,c2;..." (inclusive columns),
// bounded so a hostile program cannot force unbounded work.
func parseCopySpans(value string) []copySpan {
	const maxSpans = 1024
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	out := make([]copySpan, 0, 8)
	for _, part := range strings.Split(value, ";") {
		if len(out) >= maxSpans {
			break
		}
		fields := strings.Split(part, ",")
		if len(fields) != 3 {
			continue
		}
		row, err1 := strconv.Atoi(strings.TrimSpace(fields[0]))
		start, err2 := strconv.Atoi(strings.TrimSpace(fields[1]))
		end, err3 := strconv.Atoi(strings.TrimSpace(fields[2]))
		if err1 != nil || err2 != nil || err3 != nil || row < 0 || start < 0 || end < start {
			continue
		}
		out = append(out, copySpan{row: row, startCol: start, endCol: end})
	}
	return out
}

// copyOverlayLines paints the program-declared copy cursor, selection spans
// and search matches over the rendered content. Every span is one row of
// inclusive display-cell columns; cells are re-emitted with the highlight
// style so the character survives while its attributes change.
func copyOverlayLines(screen Screen, inset int, props Props, contentW, contentH int) []render.Line {
	if contentW <= 0 || contentH <= 0 {
		return nil
	}
	selectionStyle := chromeStyle(props, PropCopyStyleSelection, render.TokenSelection)
	matchStyle := chromeStyle(props, PropCopyStyleMatch, render.TokenAccent)
	matchCurrentStyle := chromeStyle(props, PropCopyStyleMatchCur, render.TokenWarning)
	cursorStyle := chromeStyle(props, PropCopyStyleCursor, render.Token("reverse"))

	var lines []render.Line
	paint := func(spans []copySpan, style render.Token, fillTail bool) {
		for _, span := range spans {
			if span.row < 0 || span.row >= contentH {
				continue
			}
			cells := screen.Line(span.row)
			col := 0
			runText := ""
			runStart := -1
			flush := func() {
				if runStart >= 0 && runText != "" {
					lines = append(lines, render.Line{X: inset + runStart, Y: inset + span.row, Text: runText, Style: style})
				}
				runText = ""
				runStart = -1
			}
			for _, cell := range cells {
				cellWidth := cell.Width
				if cellWidth <= 0 {
					cellWidth = render.DisplayWidth(cell.Text)
				}
				if cellWidth <= 0 {
					continue
				}
				cellStart, cellEnd := col, col+cellWidth
				col = cellEnd
				if cellStart >= contentW || cellStart > span.endCol {
					flush()
					break
				}
				if cellEnd <= span.startCol {
					flush()
					continue
				}
				if runStart < 0 {
					runStart = cellStart
				}
				runText += cell.Text
			}
			flush()
			if !fillTail {
				continue
			}
			// The old renderer paints the selection background over the blank
			// tail up to the selection column (empty rows included); the copy
			// text itself is unaffected.
			fillFrom := col
			if fillFrom < span.startCol {
				fillFrom = span.startCol
			}
			fillTo := span.endCol
			if fillTo > contentW-1 {
				fillTo = contentW - 1
			}
			if fillTo >= fillFrom && fillFrom < contentW {
				lines = append(lines, render.Line{
					X: inset + fillFrom, Y: inset + span.row,
					Text: strings.Repeat(" ", fillTo-fillFrom+1), Style: style,
				})
			}
		}
	}

	// Paint order follows the old style resolution: plain matches, then the
	// current match, then the selection, with the cursor on top.
	paint(parseCopySpans(props.Chrome[PropCopyMatch]), matchStyle, false)
	paint(parseCopySpans(props.Chrome[PropCopyMatchCurrent]), matchCurrentStyle, false)
	paint(parseCopySpans(props.Chrome[PropCopySelection]), selectionStyle, true)

	if cursor := strings.TrimSpace(props.Chrome[PropCopyCursor]); cursor != "" {
		fields := strings.Split(cursor, ",")
		if len(fields) == 2 {
			row, err1 := strconv.Atoi(strings.TrimSpace(fields[0]))
			col, err2 := strconv.Atoi(strings.TrimSpace(fields[1]))
			if err1 == nil && err2 == nil && row >= 0 && row < contentH {
				if col < 0 {
					col = 0
				} else if col >= contentW {
					col = contentW - 1
				}
				// Legacy parity: copyHistoryCursor is an always-visible block
				// cursor clamped to the frozen viewport, so it must show even
				// over a blank/end-of-line column. cellAtColumn returns not-ok
				// when the target column lies beyond the row's cells (a short
				// history row without TailFill), so fall back to a single
				// space instead of silently dropping the cursor cell.
				text := " "
				if cell, ok := cellAtColumn(screen.Line(row), col); ok && cell.Text != "" {
					text = cell.Text
				}
				lines = append(lines, render.Line{X: inset + col, Y: inset + row, Text: text, Style: cursorStyle})
			}
		}
	}
	return lines
}

// cellAtColumn finds the cell covering one display column.
func cellAtColumn(cells []Cell, target int) (Cell, bool) {
	col := 0
	for _, cell := range cells {
		width := cell.Width
		if width <= 0 {
			width = render.DisplayWidth(cell.Text)
		}
		if width <= 0 {
			continue
		}
		if target >= col && target < col+width {
			return cell, true
		}
		col += width
	}
	return Cell{}, false
}

// placeholderGlyph fills content cells outside the terminal extent footprint
// (the legacy render.ExtentPlaceholder glyph).
const placeholderGlyph = "·"

// contentFraming is the resolved content.offset/content.size for one render: the
// extent origin relative to the content area and its footprint size. cols/rows
// < 0 means "cover the content area" (the props were absent), which keeps the
// legacy output byte-identical.
type contentFraming struct {
	offsetX, offsetY int
	cols, rows       int
	placeholder      render.Token
}

func contentFramingFromProps(props Props) contentFraming {
	parsed := ContentOffsetFromProps(props.Chrome)
	f := contentFraming{offsetX: parsed.X, offsetY: parsed.Y, cols: -1, rows: -1}
	if parsed.Sized {
		f.cols, f.rows = parsed.Cols, parsed.Rows
	}
	f.placeholder = chromeStyle(props, PropPlaceholder, render.TokenMuted)
	return f
}

// contentRowLines renders one content row with the extent framing applied: the
// visible screen is drawn 1:1 at the extent origin and every content cell
// outside the extent footprint is filled with the placeholder. With the default
// framing (offset 0,0, footprint covering the content area) it is exactly the
// legacy rowLines path.
func contentRowLines(screen Screen, contentRow, x, y, maxWidth, contentH int, f contentFraming) []render.Line {
	if maxWidth <= 0 {
		return nil
	}
	footCols, footRows := f.cols, f.rows
	if footCols < 0 {
		footCols = maxWidth
	}
	if footRows < 0 {
		footRows = contentH
	}
	if f.offsetX == 0 && f.offsetY == 0 && footCols >= maxWidth && footRows >= contentH {
		return rowLines(screen.Line(contentRow), x, y, maxWidth)
	}
	sourceRow := contentRow - f.offsetY
	if sourceRow < 0 || sourceRow >= footRows {
		return placeholderLines(x, y, maxWidth, f.placeholder)
	}
	insideStart := max(0, f.offsetX)
	insideEnd := min(maxWidth, f.offsetX+footCols)
	if insideStart >= insideEnd {
		return placeholderLines(x, y, maxWidth, f.placeholder)
	}
	cells := make([]Cell, 0, maxWidth)
	if insideStart > 0 {
		cells = append(cells, placeholderCells(insideStart, f.placeholder)...)
	}
	cells = append(cells, windowCells(screen.Line(sourceRow), insideStart-f.offsetX, insideEnd-insideStart)...)
	if insideEnd < maxWidth {
		cells = append(cells, placeholderCells(maxWidth-insideEnd, f.placeholder)...)
	}
	return rowLines(cells, x, y, maxWidth)
}

// placeholderLines is one full content row of placeholder glyphs.
func placeholderLines(x, y, width int, style render.Token) []render.Line {
	if width <= 0 {
		return nil
	}
	return []render.Line{{X: x, Y: y, Text: strings.Repeat(placeholderGlyph, width), Style: style}}
}

// placeholderCells is width placeholder cells (one glyph each, so rowLines
// merges them into a single run like the legacy outside-extent fill).
func placeholderCells(width int, style render.Token) []Cell {
	if width <= 0 {
		return nil
	}
	cells := make([]Cell, width)
	for i := range cells {
		cells[i] = Cell{Text: placeholderGlyph, Width: 1, Style: style}
	}
	return cells
}

// windowCells extracts exactly width cells for screen columns [start, start+width)
// from a screen row, padding the tail with blanks (the legacy
// contentViewportLineWindow rule: a wide cluster at the window edge becomes
// styled blank columns, never a split).
func windowCells(line []Cell, start, width int) []Cell {
	if width <= 0 {
		return nil
	}
	out := make([]Cell, 0, width)
	col := 0
	for _, cell := range line {
		if len(out) >= width {
			break
		}
		w := cell.Width
		if w <= 0 {
			w = render.DisplayWidth(cell.Text)
		}
		if w <= 0 {
			continue
		}
		cellStart, cellEnd := col, col+w
		col = cellEnd
		if cellEnd <= start {
			continue
		}
		if cellStart >= start+width {
			break
		}
		visStart := max(cellStart, start)
		visEnd := min(cellEnd, start+width)
		if visEnd <= visStart {
			continue
		}
		if cellStart >= start && cellEnd <= start+width && w <= width-len(out) {
			out = append(out, cell)
			continue
		}
		for i := visStart; i < visEnd && len(out) < width; i++ {
			out = append(out, Cell{Text: " ", Width: 1, Style: cell.Style})
		}
	}
	for len(out) < width {
		out = append(out, Cell{Text: " ", Width: 1, Style: render.TokenDefault})
	}
	return out
}

// borderToken follows the v1 precedence (exited wins over focused, focused
// over inactive). A program-declared chrome style (content.props) wins over
// the built-in semantic token; the token remains the default so a program
// that sends no props keeps the golden v1 look.
func borderToken(props Props) render.Token {
	switch {
	case props.Exited:
		return chromeStyle(props, PropBorderDead, render.TokenBorderDead)
	case props.Focused:
		return chromeStyle(props, PropBorderFocus, render.TokenBorderFocus)
	default:
		return chromeStyle(props, PropBorder, render.TokenBorder)
	}
}

// chromeStyle resolves one program-declared chrome prop: the explicit style
// string when present, the built-in default token otherwise. Unknown keys are
// irrelevant here; an unparsable value degrades to no SGR in the renderer.
func chromeStyle(props Props, key string, fallback render.Token) render.Token {
	if value := strings.TrimSpace(props.Chrome[key]); value != "" {
		return render.Token(value)
	}
	return fallback
}

// titleSegments splits the decorated title into the base title and the
// appended badges ([exited N], [↑M]), so each part can carry its own style.
func titleSegments(props Props) (title, badges string) {
	title = strings.TrimSpace(props.Title)
	if title == "" {
		title = "terminal"
	}
	if props.Exited {
		// Legacy parity: a zero exit code prints the bare badge, the code is
		// only shown when it carries information ([exited] vs [exited 3]).
		if props.ExitCode != 0 {
			badges += " [exited " + itoa(props.ExitCode) + "]"
		} else {
			badges += " [exited]"
		}
	}
	if props.Scrolled {
		badges += " [↑" + itoa(props.ScrollOffset) + "]"
	}
	return title, badges
}

// TitleText is the decorated title shown in the border: the base title plus
// the exit badge and the scrollback badge, in that order.
func TitleText(props Props) string {
	title, badges := titleSegments(props)
	return title + badges
}

// titleRun is one styled piece of the border label.
type titleRun struct {
	text  string
	style render.Token
}

// topBorderLines builds the top border as styled runs: "┌", the label pieces
// (base title and badges may use different chrome styles) and the rule.
func topBorderLines(width int, props Props, border render.Token) []render.Line {
	runs := titleRuns(width, props, border)
	lines := []render.Line{{X: 0, Y: 0, Text: "┌", Style: border}}
	labelWidth := 0
	for _, run := range runs {
		lines = append(lines, render.Line{X: 1 + labelWidth, Y: 0, Text: run.text, Style: run.style})
		labelWidth += render.DisplayWidth(run.text)
	}
	dashes := width - 2 - labelWidth
	if dashes < 0 {
		dashes = 0
	}
	lines = append(lines, render.Line{
		X:     1 + labelWidth,
		Y:     0,
		Text:  strings.Repeat("─", dashes) + "┐",
		Style: border,
	})
	return lines
}

// titleRuns mirrors the v1 label algorithm byte for byte: " " (plus a focus
// marker) + decorated title + " ", truncated to always leave at least one
// dash on the top rule. The base title uses the chrome.title prop (default:
// the border style) and the badges use the chrome.badge prop (default: the
// border style); runs with the same style merge, so without props the output
// is the single v1 run. A box narrower than four cells keeps an empty label.
func titleRuns(width int, props Props, border render.Token) []titleRun {
	if width < 4 {
		return nil
	}
	title, badges := titleSegments(props)
	marker := ""
	if props.Focused && !props.Exited {
		marker = "▎"
	}
	prefix := " " + marker
	available := width - 2 - render.DisplayWidth(prefix) - 1
	if available <= 0 {
		return nil
	}
	truncated := render.Truncate(title+badges, available)
	titleStyle := chromeStyle(props, PropTitle, border)
	badgeStyle := chromeStyle(props, PropBadge, border)

	runs := make([]titleRun, 0, 3)
	if prefix != "" {
		runs = append(runs, titleRun{text: prefix, style: titleStyle})
	}
	switch {
	case truncated == "":
	case strings.HasPrefix(title, truncated):
		// Truncation stopped inside (or exactly at) the base title.
		runs = append(runs, titleRun{text: truncated, style: titleStyle})
	case strings.HasPrefix(truncated, title):
		runs = append(runs, titleRun{text: title, style: titleStyle})
		if rest := truncated[len(title):]; rest != "" {
			runs = append(runs, titleRun{text: rest, style: badgeStyle})
		}
	default:
		// Unreachable for a cluster-prefix truncation; keep the title style.
		runs = append(runs, titleRun{text: truncated, style: titleStyle})
	}
	runs = append(runs, titleRun{text: " ", style: titleStyle})
	return mergeTitleRuns(runs)
}

// mergeTitleRuns concatenates adjacent runs that share a style, so a label
// without per-part styles stays one render run (the v1 shape).
func mergeTitleRuns(runs []titleRun) []titleRun {
	out := runs[:0]
	for _, run := range runs {
		if run.text == "" {
			continue
		}
		if n := len(out); n > 0 && out[n-1].style == run.style {
			out[n-1].text += run.text
			continue
		}
		out = append(out, run)
	}
	return out
}

// rowLines converts one screen row into styled runs, clipped to maxWidth
// cells. A wide cluster that does not fully fit is dropped, never split.
func rowLines(cells []Cell, x, y, maxWidth int) []render.Line {
	if maxWidth <= 0 || len(cells) == 0 {
		return nil
	}
	var lines []render.Line
	remaining := maxWidth
	currentX := x
	var run strings.Builder
	runStyle := render.Token("")
	runWidth := 0
	flush := func() {
		if run.Len() == 0 {
			return
		}
		lines = append(lines, render.Line{X: currentX, Y: y, Text: run.String(), Style: runStyle})
		currentX += runWidth
		run.Reset()
		runWidth = 0
	}
	for _, cell := range cells {
		width := cell.Width
		if width <= 0 {
			width = render.DisplayWidth(cell.Text)
		}
		if width <= 0 {
			continue
		}
		if width > remaining {
			break
		}
		style := cell.Style
		if style == "" {
			style = render.TokenDefault
		}
		if runStyle != "" && style != runStyle {
			flush()
		}
		runStyle = style
		run.WriteString(cell.Text)
		runWidth += width
		remaining -= width
	}
	flush()
	return lines
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var digits [20]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		i--
		digits[i] = '-'
	}
	return string(digits[i:])
}

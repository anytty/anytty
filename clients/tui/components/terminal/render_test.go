package terminal

import (
	"reflect"
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/render"
)

func lineAt(t *testing.T, lines []render.Line, x, y int) render.Line {
	t.Helper()
	for _, line := range lines {
		if line.X == x && line.Y == y {
			return line
		}
	}
	t.Fatalf("no line at (%d,%d) in %+v", x, y, lines)
	return render.Line{}
}

func TestRenderUnfocusedGolden(t *testing.T) {
	component := New(nil, nil)
	component.SetProps(Props{Title: "main", Inset: 1})
	component.SetScreen(ScreenFromText([]string{"hi", "yo"}, render.TokenDefault))

	lines := component.Render(12, 4)
	if len(lines) != 10 {
		t.Fatalf("lines = %+v, want 10 runs", lines)
	}
	top := lineAt(t, lines, 0, 0)
	if top.Text != "┌" || top.Style != render.TokenBorder {
		t.Fatalf("top left = %+v", top)
	}
	label := lineAt(t, lines, 1, 0)
	if label.Text != " main " || label.Style != render.TokenBorder {
		t.Fatalf("title = %+v, want muted ' main '", label)
	}
	rule := lineAt(t, lines, 7, 0)
	if rule.Text != "────┐" || rule.Style != render.TokenBorder {
		t.Fatalf("top rule = %+v, want '────┐'", rule)
	}
	if got := lineAt(t, lines, 1, 1); got.Text != "hi" || got.Style != render.TokenDefault {
		t.Fatalf("content row 0 = %+v", got)
	}
	if got := lineAt(t, lines, 0, 1); got.Text != "│" || got.Style != render.TokenBorder {
		t.Fatalf("left border row 0 = %+v", got)
	}
	if got := lineAt(t, lines, 11, 1); got.Text != "│" {
		t.Fatalf("right border row 0 = %+v, want x=11", got)
	}
	if got := lineAt(t, lines, 1, 2); got.Text != "yo" {
		t.Fatalf("content row 1 = %+v", got)
	}
	if got := lineAt(t, lines, 11, 2); got.Text != "│" {
		t.Fatalf("right border row 1 = %+v", got)
	}
	if got := lineAt(t, lines, 0, 3); got.Text != "└──────────┘" || got.Style != render.TokenBorder {
		t.Fatalf("bottom = %+v", got)
	}
}

func TestRenderDimmedPanelUsesMutedStyles(t *testing.T) {
	component := New(nil, nil)
	component.SetProps(Props{Dimmed: true, Inset: 0, InsetSet: true})
	component.SetScreen(ScreenFromText([]string{"hello"}, render.TokenDefault))
	lines := component.Render(5, 1)
	if len(lines) != 1 || lines[0].Style != render.TokenMuted {
		t.Fatalf("dimmed default line = %+v, want muted style", lines)
	}
}

func TestRenderFocusedUsesAccentAndMarker(t *testing.T) {
	component := New(nil, nil)
	component.SetProps(Props{Title: "main", Focused: true, Inset: 1})
	component.SetScreen(ScreenFromText([]string{"x"}, render.TokenDefault))

	lines := component.Render(12, 3)
	label := lineAt(t, lines, 1, 0)
	if label.Text != " ▎main " || label.Style != render.TokenBorderFocus {
		t.Fatalf("focused title = %+v, want accent ' ▎main '", label)
	}
	if got := lineAt(t, lines, 8, 0); got.Text != "───┐" || got.Style != render.TokenBorderFocus {
		t.Fatalf("focused rule = %+v, want accent '───┐'", got)
	}
	if got := lineAt(t, lines, 0, 1); got.Text != "│" || got.Style != render.TokenBorderFocus {
		t.Fatalf("focused left border = %+v", got)
	}
}

func TestRenderExitedBadge(t *testing.T) {
	component := New(nil, nil)
	component.SetProps(Props{Title: "main", Exited: true, ExitCode: 7, Inset: 1})
	component.SetScreen(ScreenFromText([]string{"x"}, render.TokenDefault))

	lines := component.Render(12, 3)
	label := lineAt(t, lines, 1, 0)
	if label.Text != " main [ex " || label.Style != render.TokenBorderDead {
		t.Fatalf("exited title = %+v, want warning clipped ' main [ex '", label)
	}
	if got := lineAt(t, lines, 11, 0); got.Text != "┐" || got.Style != render.TokenBorderDead {
		t.Fatalf("exited top rule = %+v", got)
	}
}

func TestRenderScrolledTitle(t *testing.T) {
	component := New(nil, nil)
	component.SetProps(Props{Title: "main", Scrolled: true, ScrollOffset: 3, Inset: 1})
	component.SetScreen(ScreenFromText([]string{"x"}, render.TokenDefault))

	lines := component.Render(20, 3)
	label := lineAt(t, lines, 1, 0)
	if label.Text != " main [↑3] " || label.Style != render.TokenBorder {
		t.Fatalf("scrolled title = %+v, want ' main [↑3] '", label)
	}
	if got := lineAt(t, lines, 12, 0); got.Text != "───────┐" {
		t.Fatalf("scrolled rule = %+v, want 7 dashes + corner", got)
	}
}

func TestRenderEmojiRowKeepsCluster(t *testing.T) {
	component := New(nil, nil)
	component.SetProps(Props{Title: "", Inset: 1})
	component.SetScreen(ScreenFromText([]string{"a👨‍👩‍👧‍👦"}, render.TokenDefault))

	lines := component.Render(6, 3)
	content := lineAt(t, lines, 1, 1)
	if content.Text != "a👨‍👩‍👧‍👦" {
		t.Fatalf("content = %q, want intact family cluster", content.Text)
	}
	if render.DisplayWidth(content.Text) != 3 {
		t.Fatalf("content width = %d, want 3", render.DisplayWidth(content.Text))
	}
	if got := lineAt(t, lines, 5, 1); got.Text != "│" {
		t.Fatalf("right border = %+v", got)
	}
}

func TestRenderDropsWideClusterAtContentEdge(t *testing.T) {
	component := New(nil, nil)
	component.SetScreen(ScreenFromText([]string{"a👨‍👩‍👧‍👦"}, render.TokenDefault))
	component.SetProps(Props{Inset: 1})

	lines := component.Render(4, 3)
	content := lineAt(t, lines, 1, 1)
	if content.Text != "a" {
		t.Fatalf("content = %q, want family dropped at 2-cell content width", content.Text)
	}

	narrow := component.Render(5, 3)
	fits := lineAt(t, narrow, 1, 1)
	if fits.Text != "a👨‍👩‍👧‍👦" {
		t.Fatalf("content = %q, want family kept at 3-cell content width", fits.Text)
	}
}

func TestRenderWithoutChromeWhenTooSmall(t *testing.T) {
	component := New(nil, nil)
	component.SetScreen(ScreenFromText([]string{"ab"}, render.TokenDefault))
	lines := component.Render(2, 1)
	if len(lines) != 1 || lines[0] != (render.Line{X: 0, Y: 0, Text: "ab", Style: render.TokenDefault}) {
		t.Fatalf("lines = %+v, want raw content without chrome", lines)
	}
	if got := component.Render(0, 1); got != nil {
		t.Fatalf("zero width render = %+v, want nil", got)
	}
}

func TestRenderHonorsDeclaredInset(t *testing.T) {
	component := New(nil, nil)
	component.SetProps(Props{Inset: 2})
	component.SetScreen(ScreenFromText([]string{"hi"}, render.TokenDefault))

	lines := component.Render(8, 5)
	content := lineAt(t, lines, 2, 2)
	if content.Text != "hi" {
		t.Fatalf("content = %+v, want inset 2 offset", content)
	}
	for _, line := range lines {
		switch line.Text {
		case "│", "┌", "┐", "└", "┘":
			t.Fatalf("inset 2 reserves chrome without drawing the 1-cell border: %+v", line)
		}
	}
}

func TestRenderVisibleRowsFeedScrollWindow(t *testing.T) {
	history := &fakeHistory{lines: []string{"h1", "h2", "h3", "h4", "h5", "h6"}}
	component := New(history, nil)
	component.SetScreen(ScreenFromText([]string{"l1", "l2"}, render.TokenDefault))
	component.Render(10, 4) // content height = 2
	if _, err := component.Scroll(1); err != nil {
		t.Fatalf("Scroll: %v", err)
	}
	if len(history.calls) != 1 || history.calls[0] != (historyCall{offset: 1, rows: 2}) {
		t.Fatalf("history calls = %+v, want [{1 2}]", history.calls)
	}
}

// TestRenderChromePropsOverrideDefaults pins the M3 contract: a
// program-declared chrome.border/border_focus/border_dead wins over the
// built-in token, while chrome.title/chrome.badge color the title and the
// badges independently.
func TestRenderChromePropsOverrideDefaults(t *testing.T) {
	chrome := map[string]string{
		PropBorder:      "fg:#565f89",
		PropTitle:       "fg:#c0caf5",
		PropBorderFocus: "fg:#7aa2f7;bold",
		PropBorderDead:  "fg:#db4b4b",
		PropBadge:       "fg:#e0af68",
	}

	unfocused := New(nil, nil)
	unfocused.SetProps(Props{Title: "main", Chrome: chrome})
	unfocused.SetScreen(ScreenFromText([]string{"hi"}, render.TokenDefault))
	lines := unfocused.Render(12, 3)
	if got := lineAt(t, lines, 0, 0); got.Text != "┌" || got.Style != "fg:#565f89" {
		t.Fatalf("prop border corner = %+v, want chrome.border", got)
	}
	if got := lineAt(t, lines, 1, 0); got.Text != " main " || got.Style != "fg:#c0caf5" {
		t.Fatalf("prop title = %+v, want chrome.title", got)
	}
	if got := lineAt(t, lines, 7, 0); got.Text != "────┐" || got.Style != "fg:#565f89" {
		t.Fatalf("prop border rule = %+v, want chrome.border", got)
	}
	if got := lineAt(t, lines, 0, 1); got.Text != "│" || got.Style != "fg:#565f89" {
		t.Fatalf("prop border side = %+v, want chrome.border", got)
	}

	focused := New(nil, nil)
	focused.SetProps(Props{Title: "main", Focused: true, Chrome: chrome})
	focused.SetScreen(ScreenFromText([]string{"hi"}, render.TokenDefault))
	lines = focused.Render(12, 3)
	if got := lineAt(t, lines, 1, 0); got.Text != " ▎main " || got.Style != "fg:#c0caf5" {
		t.Fatalf("focused title = %+v, want chrome.title", got)
	}
	if got := lineAt(t, lines, 8, 0); got.Text != "───┐" || got.Style != "fg:#7aa2f7;bold" {
		t.Fatalf("focused rule = %+v, want chrome.border_focus", got)
	}

	exited := New(nil, nil)
	exited.SetProps(Props{Title: "main", Exited: true, ExitCode: 7, Chrome: chrome})
	exited.SetScreen(ScreenFromText([]string{"hi"}, render.TokenDefault))
	lines = exited.Render(12, 3)
	if got := lineAt(t, lines, 0, 1); got.Text != "│" || got.Style != "fg:#db4b4b" {
		t.Fatalf("exited border = %+v, want chrome.border_dead", got)
	}
}

// TestRenderBadgeUsesChromeBadge checks that the badges keep the badge color
// while the base title keeps the title color (the label splits into runs).
func TestRenderBadgeUsesChromeBadge(t *testing.T) {
	component := New(nil, nil)
	component.SetProps(Props{
		Title:        "main",
		Exited:       true,
		ExitCode:     7,
		Scrolled:     true,
		ScrollOffset: 3,
		Chrome: map[string]string{
			PropTitle: "fg:#c0caf5",
			PropBadge: "fg:#e0af68",
		},
	})
	component.SetScreen(ScreenFromText([]string{"x"}, render.TokenDefault))

	lines := component.Render(24, 3)
	label := lineAt(t, lines, 1, 0)
	if label.Text != " main" || label.Style != "fg:#c0caf5" {
		t.Fatalf("title run = %+v, want ' main' in chrome.title", label)
	}
	badge := lineAt(t, lines, 6, 0)
	if badge.Text != " [exited 7] [↑3]" || badge.Style != "fg:#e0af68" {
		t.Fatalf("badge run = %+v, want both badges in chrome.badge", badge)
	}
}

// TestRenderBadgeTruncationStaysByteIdentical checks the M3 default shape:
// without props the label is the single v1 run, even when truncation cuts
// into the badge.
func TestRenderBadgeTruncationStaysByteIdentical(t *testing.T) {
	component := New(nil, nil)
	component.SetProps(Props{Title: "main", Exited: true, ExitCode: 7})
	component.SetScreen(ScreenFromText([]string{"x"}, render.TokenDefault))

	lines := component.Render(12, 3)
	label := lineAt(t, lines, 1, 0)
	if label.Text != " main [ex " || label.Style != render.TokenBorderDead {
		t.Fatalf("truncated badge label = %+v, want the v1 single run", label)
	}
}

// TestRenderUnknownChromeKeysAreIgnored pins the "component interprets, host
// passes through" rule: unknown keys and junk names change nothing.
func TestRenderUnknownChromeKeysAreIgnored(t *testing.T) {
	plain := New(nil, nil)
	plain.SetProps(Props{Title: "main", Focused: true})
	plain.SetScreen(ScreenFromText([]string{"hi"}, render.TokenDefault))

	withJunk := New(nil, nil)
	withJunk.SetProps(Props{
		Title:   "main",
		Focused: true,
		Chrome:  map[string]string{"chrome.nope": "fg:#ff0000", "not-a-style": "junk"},
	})
	withJunk.SetScreen(ScreenFromText([]string{"hi"}, render.TokenDefault))

	if got, want := withJunk.Render(12, 3), plain.Render(12, 3); !reflect.DeepEqual(got, want) {
		t.Fatalf("unknown chrome keys changed the render:\n got %+v\nwant %+v", got, want)
	}
}

func TestRenderCopyOverlay(t *testing.T) {
	component := New(nil, nil)
	component.SetProps(Props{
		Title: "main",
		Inset: 1,
		Chrome: map[string]string{
			PropCopySelection:      "0,1,3",
			PropCopyMatch:          "1,0,1",
			PropCopyCursor:         "0,2",
			PropCopyStyleSelection: "fg:ansi:8;bg:ansi:3",
			PropCopyStyleMatch:     "fg:#fde68a;underline",
			PropCopyStyleCursor:    "reverse",
		},
	})
	component.SetScreen(ScreenFromText([]string{"abcd", "ef"}, render.TokenDefault))

	lines := component.Render(12, 4)
	// Selection paints "bcd" (cols 1..3) with the program style, and the
	// cursor cell (0,2) repaints "c" on top.
	if got := lineAt(t, lines, 2, 1); got.Text != "bcd" || got.Style != render.Token("fg:ansi:8;bg:ansi:3") {
		t.Fatalf("selection run = %+v", got)
	}
	if got := lineAt(t, lines, 3, 1); got.Text != "c" || got.Style != render.Token("reverse") {
		t.Fatalf("cursor cell = %+v", got)
	}
	// Match paints "ef" on row 1 (cols 0..1) with its own style; the overlay
	// is emitted after the content, so the last run at that cell wins.
	var match render.Line
	for _, line := range lines {
		if line.X == 1 && line.Y == 2 {
			match = line
		}
	}
	if match.Text != "ef" || match.Style != render.Token("fg:#fde68a;underline") {
		t.Fatalf("match run = %+v", match)
	}
	// No copy props: the content keeps its own style.
	plain := New(nil, nil)
	plain.SetProps(Props{Title: "main", Inset: 1})
	plain.SetScreen(ScreenFromText([]string{"abcd"}, render.TokenDefault))
	for _, line := range plain.Render(12, 3) {
		if line.Text == "abcd" && line.Style != render.TokenDefault {
			t.Fatalf("plain content restyled: %+v", line)
		}
	}
}

func TestRenderCopySelectionFillsRowTail(t *testing.T) {
	component := New(nil, nil)
	component.SetProps(Props{
		Title: "main",
		Inset: 1,
		Chrome: map[string]string{
			PropCopySelection:      "0,1,9",
			PropCopyStyleSelection: "fg:ansi:8;bg:ansi:3",
		},
	})
	component.SetScreen(ScreenFromText([]string{"abcd"}, render.TokenDefault))

	lines := component.Render(12, 3)
	// Text cells carry the selection style, then the blank tail up to column
	// 9 is filled with the same background (the old renderer's selection fill).
	var tail render.Line
	for _, line := range lines {
		if line.Y == 1 && line.X == 5 {
			tail = line
		}
	}
	if strings.TrimSpace(tail.Text) != "" || len(tail.Text) != 6 || tail.Style != render.Token("fg:ansi:8;bg:ansi:3") {
		t.Fatalf("selection tail fill = %+v, want 6 styled spaces at x=5", tail)
	}
	delete(component.props.Chrome, PropCopySelection)
	plain := component.Render(12, 3)
	for _, line := range plain {
		if strings.TrimSpace(line.Text) == "" && len(line.Text) > 1 && line.Y == 1 && line.X == 5 {
			t.Fatalf("fill painted without a selection: %+v", line)
		}
	}
}

// TestRenderCopyCursorShowsOverBlankShortRow pins the GAP 2 fix: the legacy
// copy cursor is an always-visible block cursor clamped to the frozen viewport
// (render.copyHistoryCursor), so copy.cursor must still emit a cursor cell when
// the target column lies beyond a short row's cells (no TailFill). Before the
// fix cellAtColumn returned ok=false and the cursor silently vanished.
func TestRenderCopyCursorShowsOverBlankShortRow(t *testing.T) {
	component := New(nil, nil)
	component.SetProps(Props{
		Title: "main",
		Inset: 1,
		Chrome: map[string]string{
			PropCopyCursor:      "0,7",
			PropCopyStyleCursor: "reverse",
		},
	})
	// The row has only four cells; column 7 is past the end.
	component.SetScreen(ScreenFromText([]string{"abcd"}, render.TokenDefault))

	lines := component.Render(12, 3)
	if got := lineAt(t, lines, 1+7, 1); got.Text != " " || got.Style != render.Token("reverse") {
		t.Fatalf("blank-column cursor = %+v, want a reverse space at x=8", got)
	}
}

// TestRenderContentOffsetInertByDefault pins the byte-identical contract: absent
// content.offset/content.size renders exactly as before and FramingFromProps
// reports no shift.
func TestRenderContentOffsetInertByDefault(t *testing.T) {
	plain := New(nil, nil)
	plain.SetProps(Props{Title: "main", Focused: true})
	plain.SetScreen(ScreenFromText([]string{"hello", "world"}, render.TokenDefault))

	explicit := New(nil, nil)
	explicit.SetProps(Props{
		Title:   "main",
		Focused: true,
		Chrome:  map[string]string{PropContentOffset: "0,0"},
	})
	explicit.SetScreen(ScreenFromText([]string{"hello", "world"}, render.TokenDefault))

	if got, want := explicit.Render(12, 4), plain.Render(12, 4); !reflect.DeepEqual(got, want) {
		t.Fatalf("explicit zero offset changed the render:\n got %+v\nwant %+v", got, want)
	}
	for _, chrome := range []map[string]string{nil, {PropContentOffset: "0,0"}} {
		if _, _, shifted := FramingFromProps(chrome); shifted {
			t.Fatalf("chrome %v must be unshifted", chrome)
		}
	}
	if dx, dy, shifted := FramingFromProps(map[string]string{PropContentOffset: "2,0"}); !shifted || dx != 2 || dy != 0 {
		t.Fatalf("FramingFromProps(2,0) = (%d,%d,%v), want (2,0,true)", dx, dy, shifted)
	}
}

// TestRenderContentOffsetShiftsScreenAndFillsPlaceholder pins the framing: the
// screen is drawn at the offset and every cell outside the footprint gets the
// placeholder glyph/style.
func TestRenderContentOffsetShiftsScreenAndFillsPlaceholder(t *testing.T) {
	component := New(nil, nil)
	component.SetProps(Props{
		Inset: 0, InsetSet: true,
		Chrome: map[string]string{
			PropContentOffset: "1,1",
			PropContentSize:   "3,2",
			PropPlaceholder:   "fg:#3b2f63",
		},
	})
	component.SetScreen(ScreenFromText([]string{"abc", "def"}, render.TokenDefault))

	lines := component.Render(5, 4)
	// Row 0 is above the footprint -> one full placeholder run.
	if got := lineAt(t, lines, 0, 0); got.Text != "·····" || got.Style != "fg:#3b2f63" {
		t.Fatalf("above footprint row = %+v, want 5 placeholder cells", got)
	}
	// The screen is drawn at content (1,1): row 0 cols 1..3 = "abc".
	if got := lineAt(t, lines, 1, 1); got.Text != "abc" || got.Style != render.TokenDefault {
		t.Fatalf("shifted screen row 0 = %+v, want abc", got)
	}
	// Row 1 is the screen's second row at content (1,2) = "def".
	if got := lineAt(t, lines, 1, 2); got.Text != "def" || got.Style != render.TokenDefault {
		t.Fatalf("shifted screen row 1 = %+v, want def", got)
	}
	// Column 0 (left margin) and column 4 (right margin) are placeholder.
	if got := lineAt(t, lines, 0, 1); got.Text != "·" || got.Style != "fg:#3b2f63" {
		t.Fatalf("left margin = %+v, want one placeholder cell", got)
	}
	if got := lineAt(t, lines, 4, 1); got.Text != "·" || got.Style != "fg:#3b2f63" {
		t.Fatalf("right margin = %+v, want one placeholder cell", got)
	}
	// Row 3 is below the footprint (rows 1..2) -> full placeholder run.
	if got := lineAt(t, lines, 0, 3); got.Text != "·····" || got.Style != "fg:#3b2f63" {
		t.Fatalf("below footprint row = %+v, want 5 placeholder cells", got)
	}
}

// TestRenderContentOffsetNegativeClipsScreen pins a negative offset: the screen
// is clipped at the top/left and the screen shows its offset cell.
func TestRenderContentOffsetNegativeClipsScreen(t *testing.T) {
	component := New(nil, nil)
	component.SetProps(Props{
		Inset: 0, InsetSet: true,
		Chrome: map[string]string{
			PropContentOffset: "-1,-1",
			PropContentSize:   "3,2",
			PropPlaceholder:   "fg:#3b2f63",
		},
	})
	component.SetScreen(ScreenFromText([]string{"abc", "def"}, render.TokenDefault))

	lines := component.Render(3, 2)
	// Content (0,0) shows screen (row 1, col 1) = "ef".
	if got := lineAt(t, lines, 0, 0); got.Text != "ef" {
		t.Fatalf("negative offset row 0 = %+v, want 'ef'", got)
	}
	// Content col 2 maps to screen col 3, outside the 3-wide footprint -> dot.
	if got := lineAt(t, lines, 2, 0); got.Text != "·" {
		t.Fatalf("negative offset right edge = %+v, want placeholder", got)
	}
	// Content row 1 is below the footprint rows [−1,1) -> full placeholder.
	if got := lineAt(t, lines, 0, 1); got.Text != "···" {
		t.Fatalf("negative offset below row = %+v, want placeholder row", got)
	}
}

// TestRenderContentOffsetLargerThanPane pins a footprint larger than the pane:
// the drawn window shifts with the offset and no placeholder appears where the
// footprint still covers the pane.
func TestRenderContentOffsetLargerThanPane(t *testing.T) {
	component := New(nil, nil)
	component.SetProps(Props{
		Inset: 0, InsetSet: true,
		Chrome: map[string]string{
			PropContentOffset: "2,0",
			PropContentSize:   "10,3",
			PropPlaceholder:   "fg:#3b2f63",
		},
	})
	component.SetScreen(ScreenFromText([]string{"0123456789", "ABCDEFGHIJ"}, render.TokenDefault))

	lines := component.Render(6, 2)
	// Columns 0,1 are the vacated left margin; content col 2 shows screen col 0.
	if got := lineAt(t, lines, 0, 0); got.Text != "··" {
		t.Fatalf("larger-than-pane left margin = %+v, want two dots", got)
	}
	if got := lineAt(t, lines, 2, 0); got.Text != "0123" {
		t.Fatalf("larger-than-pane shifted row 0 = %+v, want '0123'", got)
	}
	if got := lineAt(t, lines, 2, 1); got.Text != "ABCD" {
		t.Fatalf("larger-than-pane shifted row 1 = %+v, want 'ABCD'", got)
	}
}

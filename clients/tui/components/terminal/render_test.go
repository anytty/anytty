package terminal

import (
	"reflect"
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

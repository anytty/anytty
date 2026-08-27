package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/anytty/anytty/tui/render"
)

func TestModuleName(t *testing.T) {
	if ModuleName != "tui" {
		t.Fatalf("unexpected module name %q", ModuleName)
	}
}

func TestSmokeRun(t *testing.T) {
	frame, err := SmokeRun(context.Background())
	if err != nil {
		t.Fatalf("smoke run: %v", err)
	}
	if len(frame.Lines) == 0 || !frameContains(frame.Lines, "tui") {
		t.Fatalf("unexpected smoke frame %#v", frame)
	}
}

func TestSmokeRunDetailedCoversUIFramework(t *testing.T) {
	result, err := SmokeRunDetailed(context.Background())
	if err != nil {
		t.Fatalf("smoke run detailed: %v", err)
	}
	cases := smokeCasesByName(result)
	required := []string{"workbench-live", "split-hidden-toast", "terminal-picker", "terminal-pool-page", "workbench-tree-page", "copy-empty", "copy-history", "prompt-overlay", "help-overlay", "tab-workspace", "pane-command-flow", "visual-audit-current"}
	for _, name := range required {
		if len(cases[name].Lines) == 0 {
			t.Fatalf("missing smoke case %s in %#v", name, result.Cases)
		}
	}
	if !frameContains(cases["workbench-live"].Lines, "anytty live 🚀") ||
		!frameContains(cases["workbench-live"].Lines, "你好 output") ||
		!frameContains(cases["workbench-live"].Lines, " shell ") ||
		!frameContains(cases["workbench-live"].Lines, "["+render.DefaultPaneChromeGlyphs().SizeUnlock+"]") {
		t.Fatalf("workbench live smoke missing shell/live content: %#v", cases["workbench-live"].Lines)
	}
	if !frameContains(cases["workbench-live"].Lines, "◆ owner") {
		t.Fatalf("workbench live smoke missing terminal owner chrome: %#v", cases["workbench-live"].Lines)
	}
	if frameContains(cases["workbench-live"].Lines, "⇄2") ||
		frameContains(cases["workbench-live"].Lines, "1/31") {
		t.Fatalf("workbench live smoke should not render premature pane chrome tokens: %#v", cases["workbench-live"].Lines)
	}
	if !frameContains(cases["workbench-live"].ANSILines, "\x1b[1;38;2;169;112;255m") {
		t.Fatalf("workbench live smoke missing active pane accent ANSI: %#v", cases["workbench-live"].ANSILines)
	}
	if !frameContains(cases["workbench-live"].Lines, " WS main") || !frameContains(cases["workbench-live"].Lines, "▎ 1 main "+render.DefaultPaneChromeGlyphs().Close) || !frameContains(cases["workbench-live"].Lines, render.HeaderTabCreateText) || !frameContains(cases["workbench-live"].Lines, "[Ctrl+P] PANE") || !frameContains(cases["workbench-live"].Lines, "[Ctrl+G] GLOBAL") || !frameContains(cases["workbench-live"].Lines, "ws:main") {
		t.Fatalf("workbench live smoke missing styled shell bar tokens: %#v", cases["workbench-live"].Lines)
	}
	if !frameContains(cases["workbench-live"].ANSILines, "\x1b[48;2;8;8;13m") {
		t.Fatalf("workbench live smoke missing shell bar background ANSI: %#v", cases["workbench-live"].ANSILines)
	}
	assertNoASCIIChrome(t, "workbench-live", cases["workbench-live"])
	assertContinuousCardPaneBorder(t, "workbench-live", cases["workbench-live"])
	if frameContains(cases["split-hidden-toast"].Lines, " ws:") ||
		frameContains(cases["split-hidden-toast"].Lines, " mode:") ||
		!frameContains(cases["split-hidden-toast"].Lines, "│") ||
		frameContains(cases["split-hidden-toast"].Lines, "warn 🚀 ...") ||
		frameContains(cases["split-hidden-toast"].Lines, "warning · pending") ||
		frameContains(cases["split-hidden-toast"].Lines, "世界") {
		t.Fatalf("split hidden toast smoke invalid: %#v", cases["split-hidden-toast"].Lines)
	}
	if !frameContains(cases["split-hidden-toast"].Lines, "┌─ shell") ||
		!frameContains(cases["split-hidden-toast"].Lines, "┌─ logs") ||
		render.SliceCells(cases["split-hidden-toast"].Lines[0], render.DisplayWidth(cases["split-hidden-toast"].Lines[0])-1, render.DisplayWidth(cases["split-hidden-toast"].Lines[0])) != "┐" {
		t.Fatalf("split hidden toast smoke should keep active pane as an L-corner top boundary, got %#v", cases["split-hidden-toast"].Lines)
	}
	if frameContains(cases["split-hidden-toast"].Lines, " "+render.DefaultPaneChromeGlyphs().Running) ||
		frameContains(cases["split-hidden-toast"].Lines, "⇄2") ||
		frameContains(cases["split-hidden-toast"].Lines, "◆ owner") ||
		frameContains(cases["split-hidden-toast"].Lines, "1/31") {
		t.Fatalf("split hidden toast should not render premature pane chrome tokens: %#v", cases["split-hidden-toast"].Lines)
	}
	assertNoASCIIChrome(t, "split-hidden-toast", cases["split-hidden-toast"])
	if !frameContains(cases["terminal-picker"].Lines, "⌕") ||
		!frameContains(cases["terminal-picker"].Lines, "● local 1") ||
		!frameContains(cases["terminal-picker"].Lines, "● All 1") ||
		!frameContains(cases["terminal-picker"].Lines, "Running 1") ||
		!frameContains(cases["terminal-picker"].Lines, "Exited 0") ||
		!frameContains(cases["terminal-picker"].Lines, "Tags") ||
		!frameContains(cases["terminal-picker"].Lines, "anytty-picker shell") ||
		!frameContains(cases["terminal-picker"].Lines, "running") ||
		!frameContains(cases["terminal-picker"].Lines, "80x24") ||
		!frameContains(cases["terminal-picker"].Lines, "local") ||
		!frameContains(cases["terminal-picker"].Lines, "+ New terminal") ||
		frameContains(cases["terminal-picker"].Lines, "filter terminals") ||
		frameContains(cases["terminal-picker"].Lines, "Select terminal source state target") ||
		frameContains(cases["terminal-picker"].Lines, "DETAIL") ||
		frameContains(cases["terminal-picker"].Lines, "[attach]") ||
		frameContains(cases["terminal-picker"].Lines, "[new]") {
		t.Fatalf("terminal picker smoke missing product content: %#v", cases["terminal-picker"].Lines)
	}
	if !frameContains(cases["terminal-pool-page"].Lines, "Terminal Manager") ||
		!frameContains(cases["terminal-pool-page"].Lines, "⌕ search 日志") ||
		!frameContains(cases["terminal-pool-page"].Lines, "▸ ● 日志🚀") ||
		!frameContains(cases["terminal-pool-page"].Lines, "TERMINAL") ||
		!frameContains(cases["terminal-pool-page"].Lines, "CPU") ||
		!frameContains(cases["terminal-pool-page"].Lines, "pool page base") ||
		frameContains(cases["terminal-pool-page"].Lines, "HIST") ||
		frameContains(cases["terminal-pool-page"].Lines, "CONN") ||
		frameContains(cases["terminal-pool-page"].Lines, "Enter Attach") ||
		frameContains(cases["terminal-pool-page"].Lines, "^E Rename") {
		t.Fatalf("terminal pool smoke missing page visual contract: %#v", cases["terminal-pool-page"].Lines)
	}
	if targets := cases["terminal-pool-page"].LiveTargets; len(targets) != 1 || targets[0].TerminalID != "term-logs" {
		t.Fatalf("terminal manager live panel must subscribe to its selected terminal, got %#v", targets)
	}
	if !frameContains(cases["workbench-tree-page"].Lines, "Workbench Navigator") ||
		!frameContains(cases["workbench-tree-page"].Lines, "⌕ search [main]") ||
		!frameContains(cases["workbench-tree-page"].Lines, "WORKBENCH") ||
		!frameContains(cases["workbench-tree-page"].Lines, "日志终端") ||
		!frameContains(cases["workbench-tree-page"].Lines, "shell page base") ||
		!frameContains(cases["workbench-tree-page"].Lines, "tree page base") ||
		frameContains(cases["workbench-tree-page"].Lines, "DETAIL") ||
		frameContains(cases["workbench-tree-page"].Lines, "VIEWS") ||
		frameContains(cases["workbench-tree-page"].Lines, "Open  New  Zoom  Detach  Close") ||
		frameContains(cases["workbench-tree-page"].Lines, "[open]  Open") {
		t.Fatalf("workbench tree smoke missing page visual contract: %#v", cases["workbench-tree-page"].Lines)
	}
	if targets := cases["workbench-tree-page"].LiveTargets; len(targets) != 2 || targets[0].TerminalID != "term-main" || targets[1].TerminalID != "term-logs" {
		t.Fatalf("workbench RootSplit preview must subscribe to every visible terminal, got %#v", targets)
	}
	if !frameContains(cases["copy-empty"].Lines, "copy history empty") {
		t.Fatalf("copy empty smoke missing pending/empty content: %#v", cases["copy-empty"].Lines)
	}
	if !frameContains(cases["copy-history"].Lines, "tui") ||
		!frameContains(cases["copy-history"].Lines, "copy row") {
		t.Fatalf("copy history smoke missing authoritative row: %#v", cases["copy-history"].Lines)
	}
	if !frameContains(cases["prompt-overlay"].Lines, "Command Prompt") ||
		!frameContains(cases["prompt-overlay"].Lines, "重命名") ||
		frameContains(cases["prompt-overlay"].Lines, "[submit]  Submit") ||
		frameContains(cases["prompt-overlay"].Lines, "[cancel]  Cancel") {
		t.Fatalf("prompt overlay smoke missing prompt content: %#v", cases["prompt-overlay"].Lines)
	}
	if !frameContains(cases["help-overlay"].Lines, "Help") ||
		!frameContains(cases["help-overlay"].Lines, "Most used") ||
		!frameContains(cases["help-overlay"].Lines, "[Ctrl+P]") ||
		!frameContains(cases["help-overlay"].Lines, "[PgDn] PAGE") {
		t.Fatalf("help overlay smoke missing help content: %#v", cases["help-overlay"].Lines)
	}
	if !frameContains(cases["tab-workspace"].Lines, " remote ") ||
		!frameContains(cases["tab-workspace"].Lines, "1 main "+render.DefaultPaneChromeGlyphs().Close) ||
		!frameContains(cases["tab-workspace"].Lines, "WORKSPACE") ||
		!frameContains(cases["tab-workspace"].Lines, "workspace live") {
		t.Fatalf("tab/workspace smoke missing product entry content: %#v", cases["tab-workspace"].Lines)
	}
	if frameContains(cases["pane-command-flow"].Lines, "pane.close") ||
		!frameContains(cases["pane-command-flow"].Lines, "pane command live") {
		t.Fatalf("pane command smoke should hide toast feedback and keep live content: %#v", cases["pane-command-flow"].Lines)
	}
	if !frameContains(cases["pane-command-flow"].ANSILines, "\x1b[1;38;2;169;112;255m") {
		t.Fatalf("pane command smoke missing styled active pane ANSI: %#v", cases["pane-command-flow"].ANSILines)
	}
	assertNoASCIIChrome(t, "pane-command-flow", cases["pane-command-flow"])
	if len(cases["visual-audit-current"].Lines) != 40 || render.DisplayWidth(cases["visual-audit-current"].Lines[0]) != 140 {
		t.Fatalf("visual audit smoke must use fixed 140x40 viewport, got lines=%d width=%d", len(cases["visual-audit-current"].Lines), render.DisplayWidth(cases["visual-audit-current"].Lines[0]))
	}
	if !frameContains(cases["visual-audit-current"].Lines, "visual review") ||
		!frameContains(cases["visual-audit-current"].Lines, paneChromeCompactActionMarker()) {
		t.Fatalf("visual review smoke missing fixed visual markers: %#v", cases["visual-audit-current"].Lines)
	}
	if !frameContains(cases["visual-audit-current"].Lines, "◆ owner") ||
		!frameContains(cases["visual-audit-current"].Lines, "◇ follow") {
		t.Fatalf("visual review smoke should render bound terminal role chrome: %#v", cases["visual-audit-current"].Lines)
	}
	if frameContains(cases["visual-audit-current"].Lines, "visual acceptance") {
		t.Fatalf("visual review smoke must not claim acceptance: %#v", cases["visual-audit-current"].Lines)
	}
	if !frameContains(cases["visual-audit-current"].ANSILines, "\x1b[1;38;2;169;112;255m") {
		t.Fatalf("visual audit smoke missing active accent ANSI: %#v", cases["visual-audit-current"].ANSILines)
	}
	assertDefaultVisualReviewChrome(t, cases)
	assertNoASCIIChrome(t, "visual-audit-current", cases["visual-audit-current"])
	for name, frame := range cases {
		assertSmokeWidth(t, name, frame)
	}
}

func smokeCasesByName(result SmokeResult) map[string]render.Frame {
	cases := make(map[string]render.Frame, len(result.Cases))
	for _, item := range result.Cases {
		cases[item.Name] = item.Frame
	}
	return cases
}

func assertSmokeWidth(t *testing.T, name string, frame render.Frame) {
	t.Helper()
	if len(frame.Lines) == 0 {
		t.Fatalf("smoke case %s produced empty frame", name)
	}
	width := render.DisplayWidth(frame.Lines[0])
	for row, line := range frame.Lines {
		if got := render.DisplayWidth(line); got != width {
			t.Fatalf("smoke case %s row %d width=%d want=%d line=%q", name, row, got, width, line)
		}
	}
}

func assertNoASCIIChrome(t *testing.T, name string, frame render.Frame) {
	t.Helper()
	for row, line := range frame.Lines {
		if strings.Contains(line, "|") || strings.Contains(line, "---") {
			t.Fatalf("smoke case %s row %d contains ASCII chrome: %q", name, row, line)
		}
	}
}

func assertContinuousCardPaneBorder(t *testing.T, name string, frame render.Frame) {
	t.Helper()
	if len(frame.Lines) < 3 {
		t.Fatalf("smoke case %s too small for pane border: %#v", name, frame.Lines)
	}
	width := render.DisplayWidth(frame.Lines[0])
	for row := 1; row < len(frame.Lines)-1; row++ {
		left := render.SliceCells(frame.Lines[row], 0, 1)
		right := render.SliceCells(frame.Lines[row], width-1, width)
		previousRight := ""
		if width >= 2 {
			previousRight = render.SliceCells(frame.Lines[row], width-2, width-1)
		}
		if !isLeftPaneBorderOrOverflowGlyph(row, left) || !isRightPaneBorderOrOverflowGlyph(row, len(frame.Lines), previousRight, right) {
			t.Fatalf("smoke case %s pane border discontinuity row=%d left=%q right=%q frame=%#v", name, row, left, right, frame.Lines)
		}
	}
}

func assertDefaultVisualReviewChrome(t *testing.T, cases map[string]render.Frame) {
	t.Helper()
	review := cases["visual-audit-current"]
	closeGlyph := render.DefaultPaneChromeGlyphs().Close
	requiredReview := []string{" WS main", "▎ 1 main " + closeGlyph + "   2 logs " + closeGlyph + render.HeaderTabCreateText, "visual review", floatingChromeFullActionMarker(), "unconnected", "└───────────────────────────┘", "[Ctrl+P] PANE", "[Ctrl+G] GLOBAL", "[PgUp] COPY", "ws:main float:1"}
	for _, marker := range requiredReview {
		if !frameContains(review.Lines, marker) {
			t.Fatalf("visual review smoke missing chrome marker %q: %#v", marker, review.Lines)
		}
	}
	requiredOverlays := map[string][]string{
		"terminal-picker":     {"┌─ Terminal Picker", "⌕", "anytty-picker shell", "running", "80x24"},
		"terminal-pool-page":  {"┌─ Terminal Manager", "⌕ search 日志", "TERMINAL", "CPU", "pool page base"},
		"workbench-tree-page": {"┌─ Workbench Navigator", "⌕ search [main]", "WORKBENCH", "shell page base", "日志终端", "tree page base"},
		"prompt-overlay":      {"┌─ prompt", "Command Prompt", "重命名"},
		"help-overlay":        {"┌─ help", "● open", "Esc", "Most used", "[Ctrl+P]"},
	}
	for name, markers := range requiredOverlays {
		frame := cases[name]
		for _, marker := range markers {
			if !frameContains(frame.Lines, marker) {
				t.Fatalf("smoke case %s missing overlay visual marker %q: %#v", name, marker, frame.Lines)
			}
		}
	}
	picker := cases["terminal-picker"]
	for _, stale := range []string{"filter terminals", "PREVIEW pane:", "Select terminal source state target", "DETAIL", "[attach]", "[new]", "pane:", "selected ", "● open"} {
		if frameContains(picker.Lines, stale) {
			t.Fatalf("terminal picker regressed to engineering label %q: %#v", stale, picker.Lines)
		}
	}
	prompt := cases["prompt-overlay"]
	for _, stale := range []string{"● open", "esc"} {
		if frameContains(prompt.Lines, stale) {
			t.Fatalf("prompt overlay regressed to engineering label %q: %#v", stale, prompt.Lines)
		}
	}
	workbenchTreeTitle := ""
	for _, line := range cases["workbench-tree-page"].Lines {
		if strings.Contains(line, "Workbench Navigator") {
			workbenchTreeTitle = line
			break
		}
	}
	for _, stale := range []string{"● open", "esc"} {
		if strings.Contains(workbenchTreeTitle, stale) {
			t.Fatalf("workbench navigator title regressed to redundant hint %q: %q", stale, workbenchTreeTitle)
		}
	}
	split := cases["split-hidden-toast"]
	for _, marker := range []string{"┌─ shell", "┌─ logs", "└"} {
		if !frameContains(split.Lines, marker) {
			t.Fatalf("split hidden toast missing styled split marker %q: %#v", marker, split.Lines)
		}
	}
	for _, stale := range []string{"⇄2", "◆ owner", "1/31", " " + render.DefaultPaneChromeGlyphs().Running} {
		if frameContains(split.Lines, stale) {
			t.Fatalf("split hidden toast rendered premature pane chrome marker %q: %#v", stale, split.Lines)
		}
	}
}

func floatingChromeFullActionMarker() string {
	glyphs := render.DefaultPaneChromeGlyphs()
	return "[" + glyphs.CenterFloating + "]─[" + glyphs.CollapseFloating + "]─[" + glyphs.Zoom + "]─[" + glyphs.Close + "]"
}

func paneChromeCompactActionMarker() string {
	glyphs := render.DefaultPaneChromeGlyphs()
	return "[" + glyphs.Zoom + "]─[" + glyphs.Close + "]"
}

func isPaneBorderGlyph(value string) bool {
	switch value {
	case "│", "┌", "┐", "└", "┘", "├", "┤", "┼":
		return true
	default:
		return false
	}
}

func isLeftPaneBorderOrOverflowGlyph(row int, left string) bool {
	return isPaneBorderGlyph(left) || left == render.DefaultPaneChromeGlyphs().OverflowLeft
}

func isRightPaneBorderOrOverflowGlyph(row int, lineCount int, previousRight string, right string) bool {
	return isPaneBorderGlyph(right) || right == render.DefaultPaneChromeGlyphs().OverflowRight
}

func frameContains(lines []string, value string) bool {
	for _, line := range lines {
		if strings.Contains(line, value) {
			return true
		}
	}
	return false
}

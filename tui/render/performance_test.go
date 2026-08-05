package render

import (
	"fmt"
	"strings"
	"testing"
)

var benchmarkLargeFrame Frame

func BenchmarkRendererLargeTerminalOutput(b *testing.B) {
	renderer := NewRenderer(DefaultTheme())
	vm := benchmarkLargeOutputVM(180, 60, 240)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkLargeFrame = renderer.Render(vm)
	}
}

func benchmarkLargeOutputVM(width int, height int, rows int) RenderVM {
	liveLines := makeBenchmarkLines(rows, "live")
	logLines := makeBenchmarkLines(rows, "logs")
	return RenderVM{Shell: ShellVM{
		Header: HeaderVM{Visible: true, Workspace: "bench", Tab: "[large]", ActivePane: "pane-live", TerminalSummary: "term:2", FloatingSummary: "float:0"},
		Footer: FooterVM{Visible: true, Mode: "live", Hint: "large terminal output", ActionTokens: []FooterActionVM{
			{Key: "^P", Label: "PANE", ActionID: "menu.panel"},
			{Key: "^V", Label: "COPY", ActionID: "menu.copy"},
		}, ActiveTarget: "pane:large live"},
		Layout: LayoutVM{
			Viewport: Rect{W: width, H: height},
			Panels: []PanelVM{{
				ID:           "pane-live",
				Title:        "large-live",
				Presentation: PanelPresentationSplitLine,
				Active:       true,
				Content:      ContentVM{Kind: ContentTerminalLive, Lines: liveLines},
			}, {
				ID:           "pane-logs",
				Title:        "large-logs",
				Presentation: PanelPresentationSplitLine,
				Active:       false,
				Content:      ContentVM{Kind: ContentPlaceholder, Lines: logLines},
			}},
			Split: SplitVM{Direction: SplitVertical, Children: []SplitVM{{PaneID: "pane-live"}, {PaneID: "pane-logs"}}},
		},
		Toasts: []ToastVM{{ID: "bench-toast", Severity: ToastInfo, Title: "render", Body: "benchmark"}},
	}}
}

func makeBenchmarkLines(rows int, prefix string) []Line {
	lines := make([]Line, rows)
	payload := strings.Repeat(" segment🚀 世界", 8)
	for i := range lines {
		lines[i] = Line{Cells: []Cell{
			styledCell(fmt.Sprintf("%s:%03d ", prefix, i), StyleAccent),
			NewCell(payload),
			styledCell(" ok", StyleSuccess),
		}}
	}
	return lines
}

package app

import (
	"context"
	"fmt"
	"github.com/anytty/anytty/tui/testkit"
	"strings"
	"testing"

	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

func TestPanelContentAcceptanceMatrix(t *testing.T) {
	t.Run("single pane live", func(t *testing.T) {
		frame := renderPanelContentAcceptanceFrame(panelAcceptanceBindPaneTerminal(state.Root{
			Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 24},
			Session:  state.TerminalSessionStore{TerminalID: "term-main", Attached: true, Cols: 78, Rows: 20},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-main",
				Ready:      true,
				Cols:       78,
				Rows:       20,
				Lines:      []string{"top output"},
			},
		}, state.DefaultPaneID, "term-main"))
		assertPanelFrameContains(t, frame, "top output")
		assertPanelFrameWidth(t, frame, 80)
	})

	t.Run("dual pane live and inactive placeholder", func(t *testing.T) {
		shell := state.DefaultShell().
			SplitActivePane(state.PaneState{ID: "pane-logs", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-logs"}, state.SplitDirectionVertical).
			FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
		shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-main"
		frame := renderPanelContentAcceptanceFrame(panelAcceptanceBindPaneTerminal(state.Root{
			Shell:    shell,
			Viewport: state.ViewportStore{Valid: true, Cols: 96, Rows: 24},
			Session:  state.TerminalSessionStore{TerminalID: "term-main", Attached: true, Cols: 45, Rows: 20},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-main",
				Ready:      true,
				Cols:       45,
				Rows:       20,
				Lines:      []string{"main live"},
			},
		}, state.DefaultPaneID, "term-main"))
		assertPanelFrameContains(t, frame, "main live")
		assertPanelFrameContains(t, frame, "logs inactive")
		assertPanelFrameNotContains(t, frame, "live surface pending")
		assertPanelFrameWidth(t, frame, 96)
	})

	t.Run("split resize live clipping", func(t *testing.T) {
		terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-main", Channel: 4}}
		host := NewFakeTerminalHost(16)
		host.SetSize(100, 28)
		shell := state.DefaultShell().
			SetPanelPresentation(state.PanelPresentationSplitLine).
			SplitActivePane(state.PaneState{ID: "pane-right", Title: "right", Kind: state.PaneTerminalLive, TerminalID: "term-main"}, state.SplitDirectionVertical)
		runtime := NewInteractiveRuntime(
			state.Root{Shell: shell},
			host,
			NewSyncEffectRunner(),
			LiveDeps{Terminal: terminal},
			CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
		)
		if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{
			TerminalID:   "term-main",
			Cols:         100,
			Rows:         28,
			ResizePolicy: state.TerminalResizeRoleOwner,
			SurfaceID:    "test-surface",
			ViewID:       state.TerminalPaneViewID("pane-right"),
		}}); err != nil {
			t.Fatalf("post attach: %v", err)
		}
		if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
			Action:   state.PaneCommandSetSize,
			Target:   state.PaneCommandTarget{PaneID: "pane-right"},
			SizeMode: state.PaneSizeCells,
			Cols:     26,
		}}); err != nil {
			t.Fatalf("post set size: %v", err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain resize: %v", err)
		}
		if len(terminal.Resizes) == 0 || terminal.Resizes[len(terminal.Resizes)-1].Cols != 24 {
			t.Fatalf("split resize should resize active terminal to content rect, got %#v", terminal.Resizes)
		}
		if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
			TerminalID: "term-main",
			Revision:   2,
			Cols:       40,
			Rows:       30,
			Lines:      []string{"right pane live content should be clipped by the pane"},
		}}); err != nil {
			t.Fatalf("post oversized surface: %v", err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain surface: %v", err)
		}
		frame := lastFrame(t, host.Frames())
		assertPanelFrameContains(t, frame, "right pane live content")
		result := render.NewRenderer(render.DefaultTheme()).RenderResult(render.NewRenderVMBuilder().Build(runtime.State()))
		if !panelContentAcceptanceHasOverflow(result, render.LayerPanel) {
			t.Fatalf("split resized live content should expose chrome overflow, layers=%#v", result.Layers)
		}
	})

	t.Run("copy mode pending", func(t *testing.T) {
		frame := renderPanelContentAcceptanceFrame(state.Root{
			Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 24},
			Surface:  state.TerminalSurfaceStore{TerminalID: "term-main", Lines: []string{"live should not fallback"}},
			CopyMode: state.CopyModeStore{Active: true, TerminalID: "term-main", BoundCols: 78},
			History:  state.HistoryStore{TerminalID: "term-main"},
		})
		assertPanelFrameContains(t, frame, "copy history pending")
		assertPanelFrameNotContains(t, frame, "live should not fallback")
	})

	t.Run("copy mode loaded styled rows", func(t *testing.T) {
		root := state.Root{
			Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 24},
			CopyMode: state.CopyModeStore{
				Active:     true,
				TerminalID: "term-main",
				BoundToken: "tok-1",
				BoundCols:  78,
			},
			History: state.HistoryStore{
				TerminalID: "term-main",
				Token:      "tok-1",
				Cols:       78,
				Rows: []state.HistoryRow{
					{Text: "git log", LineID: 100},
					{
						Text:      "ERR deploy",
						LineID:    101,
						RowInLine: 1,
						Cells: []state.HistoryCell{
							{Text: "ERR", Width: 3, Style: state.HistoryCellStyle{FG: "ansi:1", Bold: true}},
							{Text: " deploy", Width: 7, Style: state.HistoryCellStyle{FG: "#ffcc00", Underline: true}, LinkURL: "file://deploy.log"},
						},
					},
				},
			},
		}
		vm := render.NewRenderVMBuilder().Build(root)
		content := panelContentAcceptanceActiveContent(vm.Shell)
		if !panelLineHasANSIText(content.Lines, "ERR", render.ANSICellStyle{FG: "ansi:1", Bold: true}) {
			t.Fatalf("styled history row should preserve ANSI palette/bold metadata, got %#v", content.Lines)
		}
		if !panelLineHasANSIText(content.Lines, " deploy", render.ANSICellStyle{FG: "#ffcc00", Underline: true}) {
			t.Fatalf("styled history row should preserve truecolor/underline metadata, got %#v", content.Lines)
		}
		if !panelLineHasLinkText(content.Lines, " deploy", "file://deploy.log", "") {
			t.Fatalf("styled history row should preserve link metadata, got %#v", content.Lines)
		}
		frame := render.NewRenderer(render.DefaultTheme()).Render(vm)
		assertPanelFrameContains(t, frame, "git log")
		assertPanelFrameContains(t, frame, "ERR deploy")
		assertPanelFrameNotContains(t, frame, "● git log")
		assertPanelFrameNotContains(t, frame, "╎ ERR deploy")
	})

	t.Run("floating pane content", func(t *testing.T) {
		shell := state.DefaultShell()
		var result state.FloatingCommandResult
		shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
			Action:   state.FloatingCommandCreate,
			TargetID: "float-1",
			Pane:     state.PaneState{ID: "float-pane", Title: "floating", Kind: state.PaneEmpty},
			Rect:     state.FloatingRect{X: 8, Y: 5, W: 34, H: 8},
		})
		if result.Status != state.FloatingCommandOK {
			t.Fatalf("create floating: %#v", result)
		}
		frame := renderPanelContentAcceptanceFrame(panelAcceptanceBindPaneTerminal(state.Root{
			Shell:    shell,
			Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 24},
			Surface:  state.TerminalSurfaceStore{TerminalID: "term-main", Lines: []string{"tiled live"}},
		}, state.DefaultPaneID, "term-main"))
		assertPanelFrameContains(t, frame, "No terminal connected")
		assertPanelFrameContains(t, frame, "Choose a terminal or create one")
		assertPanelFrameContains(t, frame, "Attach existing terminal")
		assertPanelFrameContains(t, frame, "["+render.DefaultPaneChromeGlyphs().Zoom+"]─["+render.DefaultPaneChromeGlyphs().Close+"]")
	})

	t.Run("exited pane", func(t *testing.T) {
		root := panelAcceptanceBindPaneTerminal(state.Root{
			Shell:    state.DefaultShell(),
			Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 24},
			Surface:  state.TerminalSurfaceStore{TerminalID: "term-done", State: state.TerminalLiveExited, ExitCode: 0, ExitReason: "exited"},
			Session:  state.TerminalSessionStore{TerminalID: "term-done", State: state.TerminalLiveExited, ExitCode: 0, ExitReason: "exited"},
		}, state.DefaultPaneID, "term-done")
		frame := renderPanelContentAcceptanceFrame(root)
		assertPanelFrameContains(t, frame, "terminal exited: term-done code:0 exited")
		assertPanelFrameContains(t, frame, "restart")
	})

	t.Run("error pane", func(t *testing.T) {
		terminal := &testkit.FakeTerminalService{AttachErr: context.Canceled}
		host := NewFakeTerminalHost(4)
		host.SetSize(80, 24)
		runtime := NewLiveRuntime(state.Root{}, host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal})
		if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-main", Cols: 80, Rows: 24}}); err != nil {
			t.Fatalf("post attach: %v", err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain attach error: %v", err)
		}
		frame := lastFrame(t, host.Frames())
		assertPanelFrameContains(t, frame, "context canceled")
	})

	t.Run("live burst latest wins", func(t *testing.T) {
		host := NewFakeTerminalHost(8)
		host.SetSize(80, 24)
		root := state.Root{}
		root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-main", 7, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))
		runtime := NewLiveRuntime(
			root,
			host,
			NewSyncEffectRunner(),
			LiveDeps{Terminal: &testkit.FakeTerminalService{}},
		)
		for revision := uint64(1); revision <= 50; revision++ {
			if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
				TerminalID: "term-main",
				Revision:   revision,
				Cols:       78,
				Rows:       20,
				Lines:      []string{"build " + panelContentAcceptanceRevisionLabel(revision)},
			}}); err != nil {
				t.Fatalf("post revision %d: %v", revision, err)
			}
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain burst: %v", err)
		}
		frame := lastFrame(t, host.Frames())
		assertPanelFrameContains(t, frame, "build r50")
		assertPanelFrameNotContains(t, frame, "build r01")
		renderedBurstFrames := 0
		for _, frame := range host.Frames() {
			if frameContains(frame, "build r") {
				renderedBurstFrames++
			}
		}
		if renderedBurstFrames != 1 {
			t.Fatalf("ordinary live burst should coalesce to one rendered content frame, got %d frames", renderedBurstFrames)
		}
		if runtime.State().Surface.Revision != 50 {
			t.Fatalf("latest surface revision should win, got %#v", runtime.State().Surface)
		}
	})
}

func renderPanelContentAcceptanceFrame(root state.Root) render.Frame {
	return render.NewRenderer(render.DefaultTheme()).Render(render.NewRenderVMBuilder().Build(root))
}

func panelAcceptanceBindPaneTerminal(root state.Root, paneID string, terminalID string) state.Root {
	if paneID == "" {
		paneID = state.DefaultPaneID
	}
	cols, rows := root.Surface.Cols, root.Surface.Rows
	if cols <= 0 {
		cols = root.Session.Cols
	}
	if rows <= 0 {
		rows = root.Session.Rows
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(paneID, terminalID, 7, cols, rows, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(paneID), true))
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: paneID}, terminalID)
	return root
}

func assertPanelFrameContains(t *testing.T, frame render.Frame, value string) {
	t.Helper()
	if !frameContains(frame, value) {
		t.Fatalf("frame missing %q:\n%s", value, strings.Join(frame.Lines, "\n"))
	}
}

func assertPanelFrameNotContains(t *testing.T, frame render.Frame, value string) {
	t.Helper()
	if frameContains(frame, value) {
		t.Fatalf("frame should not contain %q:\n%s", value, strings.Join(frame.Lines, "\n"))
	}
}

func assertPanelFrameWidth(t *testing.T, frame render.Frame, width int) {
	t.Helper()
	for row, line := range frame.Lines {
		if got := render.DisplayWidth(line); got != width {
			t.Fatalf("frame row %d width=%d want=%d line=%q", row, got, width, line)
		}
	}
}

func panelContentAcceptanceHasOverflow(result render.RenderResult, kind render.LayerKind) bool {
	for _, layer := range result.Layers {
		if layer.Kind == kind && (layer.ContentOverflow.Right || layer.ContentOverflow.Bottom) {
			return true
		}
	}
	return false
}

func panelContentAcceptanceActiveContent(shell render.ShellVM) render.ContentVM {
	for _, panel := range shell.Layout.Panels {
		if panel.Active {
			return panel.Content
		}
	}
	return render.ContentVM{}
}

func panelLineHasANSIText(lines []render.Line, text string, style render.ANSICellStyle) bool {
	for _, line := range lines {
		for _, cell := range line.Cells {
			if cell.Text == text && cell.ANSIStyle == style {
				return true
			}
		}
	}
	return false
}

func panelLineHasLinkText(lines []render.Line, text string, linkURL string, linkParams string) bool {
	for _, line := range lines {
		for _, cell := range line.Cells {
			if cell.Text == text && cell.LinkURL == linkURL && cell.LinkParams == linkParams {
				return true
			}
		}
	}
	return false
}

func panelContentAcceptanceRevisionLabel(revision uint64) string {
	return fmt.Sprintf("r%02d", revision)
}

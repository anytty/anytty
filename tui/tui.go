package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/anytty/anytty/tui/app"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

// ModuleName 是 v3 TUI module 的稳定标识。
const ModuleName = "tui"

type SmokeCase struct {
	Name  string
	Frame render.Frame
}

type SmokeResult struct {
	Cases []SmokeCase
}

// SmokeRun 跑通 state -> ShellVM -> render framework -> FrameSink 的默认 smoke。
func SmokeRun(ctx context.Context) (render.Frame, error) {
	result, err := SmokeRunDetailed(ctx)
	if err != nil {
		return render.Frame{}, err
	}
	if len(result.Cases) == 0 {
		return render.Frame{}, nil
	}
	for _, item := range result.Cases {
		if item.Name == "copy-history" {
			return item.Frame, nil
		}
	}
	return result.Cases[len(result.Cases)-1].Frame, nil
}

func SmokeRunDetailed(ctx context.Context) (SmokeResult, error) {
	builder := render.NewRenderVMBuilder()
	renderer := render.NewRenderer(render.DefaultTheme())
	rootCases := []struct {
		name string
		root state.Root
	}{
		{name: "workbench-live", root: smokeLiveRoot()},
		{name: "split-hidden-toast", root: smokeSplitHiddenToastRoot()},
		{name: "terminal-picker", root: smokeTerminalPickerRoot()},
		{name: "terminal-pool-page", root: smokeTerminalPoolRoot()},
		{name: "workbench-tree-page", root: smokeWorkbenchTreeRoot()},
		{name: "copy-empty", root: smokeCopyEmptyRoot()},
		{name: "copy-history", root: smokeCopyHistoryRoot()},
		{name: "prompt-overlay", root: smokePromptRoot()},
		{name: "help-overlay", root: smokeHelpRoot()},
		{name: "tab-workspace", root: smokeTabWorkspaceRoot()},
	}
	result := SmokeResult{Cases: make([]SmokeCase, 0, len(rootCases)+2)}
	for _, item := range rootCases {
		host := app.NewFakeTerminalHost(1)
		if item.root.Viewport.Valid {
			host.SetSize(item.root.Viewport.Cols, item.root.Viewport.Rows)
		}
		runtime := app.NewAppRuntime(item.root, nil, func(root state.Root) render.Frame {
			return renderer.Render(builder.Build(root))
		}, host, nil)
		if err := runtime.Post(app.NoopMsg{}); err != nil {
			return SmokeResult{}, err
		}
		if err := runtime.Drain(ctx); err != nil {
			return SmokeResult{}, err
		}
		frames := host.Frames()
		if len(frames) == 0 {
			return SmokeResult{}, fmt.Errorf("smoke case %s produced no frames", item.name)
		}
		result.Cases = append(result.Cases, SmokeCase{Name: item.name, Frame: frames[len(frames)-1]})
	}
	paneCommandFrame, err := smokePaneCommandFrame(ctx, builder, renderer)
	if err != nil {
		return SmokeResult{}, err
	}
	result.Cases = append(result.Cases, SmokeCase{Name: "pane-command-flow", Frame: paneCommandFrame})
	visualFrame, err := smokeVisualAuditFrame(ctx, builder, renderer)
	if err != nil {
		return SmokeResult{}, err
	}
	result.Cases = append(result.Cases, SmokeCase{Name: "visual-audit-current", Frame: visualFrame})
	return result, nil
}

func smokeLiveRoot() state.Root {
	root := state.Root{
		Shell: state.DefaultShell(),
		Surface: state.TerminalSurfaceStore{
			TerminalID: "anytty-live",
			Cols:       80,
			Rows:       24,
			Lines:      []string{"anytty live 🚀", "你好 output"},
		},
	}
	owner := state.NewPaneTerminalView(state.DefaultPaneID, "anytty-live", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)
	owner.OwnerSurfaceID = owner.SurfaceID
	owner.OwnerViewID = owner.ViewID
	root.TerminalViews = root.TerminalViews.BindPane(owner)
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "anytty-live")
	return root
}

func smokeSplitHiddenToastRoot() state.Root {
	shell := state.DefaultShell().
		SetPanelPresentation(state.PanelPresentationSplitLine).
		SetHeaderVisible(false).
		SetFooterVisible(false).
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive}, state.SplitDirectionVertical).
		AddToast(state.ToastSpec{ID: "toast-1", Severity: state.ToastWarning, Title: "warn 🚀", Body: "世界", Pending: true})
	return state.Root{
		Shell: shell,
		Surface: state.TerminalSurfaceStore{
			TerminalID: "anytty-split",
			Cols:       80,
			Rows:       24,
			Lines:      []string{"split live"},
		},
	}
}

func smokeTerminalPickerRoot() state.Root {
	return state.Root{
		Shell: state.DefaultShell().OpenTerminalPicker(),
		TerminalPool: state.TerminalPoolStore{
			Status: state.TerminalPoolReady,
			Items: []state.TerminalPoolItem{{
				TerminalID: "anytty-picker",
				Title:      "anytty-picker shell",
				State:      "running",
				Cols:       80,
				Rows:       24,
			}},
		},
		Surface: state.TerminalSurfaceStore{
			TerminalID: "anytty-picker",
			Cols:       80,
			Rows:       24,
			Lines:      []string{"picker base"},
		},
	}
}

func smokeTerminalPoolRoot() state.Root {
	shell := state.DefaultShell().OpenTerminalPool().SetTerminalPoolQuery("日志")
	return state.Root{
		Shell:    shell,
		Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 30},
		TerminalPool: state.TerminalPoolStore{
			Status: state.TerminalPoolReady,
			Items: []state.TerminalPoolItem{{
				TerminalID: "term-logs",
				Title:      "日志🚀",
				State:      "running",
				CWD:        "/Users/anytty/project/日志",
				Cols:       120,
				Rows:       36,
				Attached:   true,
				Resources: state.TerminalResourceUsage{
					PID: 4101, CPUPercentX100: 1730, MemoryBytes: 186 * 1024 * 1024,
					SampledAt: time.Date(2026, 8, 10, 3, 40, 0, 0, time.UTC),
				},
				Tags: map[string]string{"role": "logs", "owner": "local"},
			}, {
				TerminalID: "term-worker",
				Title:      "worker",
				State:      "exited",
				CWD:        "/tmp/worker",
			}},
		},
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-logs",
			Cols:       100,
			Rows:       30,
			Lines:      []string{"pool page base"},
		},
	}
}

func smokeWorkbenchTreeRoot() state.Root {
	shell := state.DefaultShell().
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-main").
		SplitActivePane(state.PaneState{ID: "pane-logs", Title: "日志🚀", Kind: state.PaneTerminalLive, TerminalID: "term-logs"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: "pane-logs"}).
		OpenWorkbenchTree()
	surfaces := (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
		TerminalID: "term-main",
		Cols:       80,
		Rows:       30,
		Lines:      []string{"shell page base", "$ make test", "ok   tui/render"},
	})
	surfaces = surfaces.ApplySnapshot(state.LiveSurfaceSnapshot{
		TerminalID: "term-logs",
		Cols:       80,
		Rows:       30,
		Lines:      []string{"tree page base", "api listening :8080", "metrics ready"},
	})
	root := state.Root{
		Shell:    shell,
		Viewport: state.ViewportStore{Valid: true, Cols: 160, Rows: 40},
		TerminalPool: state.TerminalPoolStore{Status: state.TerminalPoolReady, Items: []state.TerminalPoolItem{
			{TerminalID: "term-main", Title: "shell", State: "running", Cols: 80, Rows: 30},
			{TerminalID: "term-logs", Title: "日志终端", State: "running", Cols: 80, Rows: 30},
		}},
		Surface: surfaces,
		Session: state.TerminalSessionStore{TerminalID: "term-logs", Attached: true, Cols: 80, Rows: 30, State: state.TerminalLiveAttached},
	}
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-main", 1, 80, 30, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID(state.DefaultPaneID), false)).
		BindPane(state.NewPaneTerminalView("pane-logs", "term-logs", 2, 80, 30, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID("pane-logs"), true))
	items := state.WorkbenchTreeItems(root)
	for index, item := range items {
		if item.Kind == state.WorkbenchTreeKindTab {
			root.Shell = root.Shell.SetWorkbenchTreeSelectedIndex(index, len(items))
			break
		}
	}
	return root
}

func smokeCopyEmptyRoot() state.Root {
	return state.Root{
		History: state.HistoryStore{
			TerminalID: "anytty-copy-empty",
			Token:      "empty-token",
			Cols:       80,
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "anytty-copy-empty",
			BoundToken: "empty-token",
			BoundCols:  80,
		},
	}
}

func smokeCopyHistoryRoot() state.Root {
	return state.Root{
		History: state.HistoryStore{
			TerminalID: "anytty-smoke",
			Token:      "smoke-token",
			Cols:       80,
			Rows:       []state.HistoryRow{{Text: "tui", LineID: 1}, {Text: "copy row", LineID: 2}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "anytty-smoke",
			BoundToken: "smoke-token",
			BoundCols:  80,
		},
	}
}

func smokePromptRoot() state.Root {
	shell := state.DefaultShell().OpenPrompt(state.PromptState{
		Title:       "Command Prompt",
		Context:     "Rename tab before switching workspace",
		Value:       "重命名",
		Placeholder: "name",
	})
	return state.Root{Shell: shell}
}

func smokeHelpRoot() state.Root {
	return state.Root{Shell: state.DefaultShell().OpenHelp("most-used")}
}

func smokeTabWorkspaceRoot() state.Root {
	shell := state.DefaultShell()
	shell, _ = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: "logs"})
	shell, _ = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceCreate, Name: "remote"})
	shell = shell.SetInteractionMode(state.InteractionModeWorkspace)
	root := state.Root{
		Shell: shell,
		Surface: state.TerminalSurfaceStore{
			TerminalID: "anytty-workspace",
			Cols:       80,
			Rows:       24,
			Lines:      []string{"workspace live"},
		},
	}
	paneID := root.Shell.EnsureDefaults().ActivePaneID
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(paneID, "anytty-workspace", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(paneID), true))
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: paneID}, "anytty-workspace")
	return root
}

func smokePaneCommandFrame(ctx context.Context, builder render.RenderVMBuilder, renderer render.Renderer) (render.Frame, error) {
	host := app.NewFakeTerminalHost(16)
	host.SetSize(64, 16)
	root := state.Root{
		Shell: state.DefaultShell().SetPanelPresentation(state.PanelPresentationSplitLine),
		Surface: state.TerminalSurfaceStore{
			TerminalID: "anytty-pane-command",
			Cols:       64,
			Rows:       16,
			Lines:      []string{"pane command live"},
		},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "anytty-pane-command", 7, 64, 16, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "anytty-pane-command")
	runtime := app.NewAppRuntime(root, app.NewShellReducer(), func(root state.Root) render.Frame {
		return renderer.Render(builder.Build(root))
	}, host, app.NewSyncEffectRunner())
	commands := []state.PaneCommand{
		{
			Action:         state.PaneCommandSplit,
			SplitDirection: state.SplitDirectionVertical,
			NewPane:        state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive},
			Source:         state.PaneCommandSourceTest,
		},
		{Action: state.PaneCommandResize, Target: state.PaneCommandTarget{PaneID: "pane-2"}, ResizeDirection: state.PaneResizeLeft, Delta: 4, Source: state.PaneCommandSourceTest},
		{Action: state.PaneCommandZoom, Target: state.PaneCommandTarget{PaneID: "pane-2"}, Source: state.PaneCommandSourceTest},
		{Action: state.PaneCommandUnzoom, Target: state.PaneCommandTarget{PaneID: "pane-2"}, Source: state.PaneCommandSourceTest},
		{Action: state.PaneCommandClose, Target: state.PaneCommandTarget{PaneID: "pane-2"}, Source: state.PaneCommandSourceTest},
	}
	for _, command := range commands {
		if err := runtime.Post(app.ShellPaneCommandMsg{Command: command}); err != nil {
			return render.Frame{}, err
		}
	}
	if err := runtime.Drain(ctx); err != nil {
		return render.Frame{}, err
	}
	frames := host.Frames()
	if len(frames) == 0 {
		return render.Frame{}, fmt.Errorf("pane command smoke produced no frames")
	}
	return frames[len(frames)-1], nil
}

func smokeVisualAuditFrame(ctx context.Context, builder render.RenderVMBuilder, renderer render.Renderer) (render.Frame, error) {
	host := app.NewFakeTerminalHost(4)
	host.SetSize(140, 40)
	shell := state.DefaultShell()
	shell, _ = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: "logs"})
	shell, _ = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabSwitch, TargetID: state.DefaultTabID})
	activeShellPaneID := shell.ActivePaneID
	shell = shell.
		SetPanelPresentation(state.PanelPresentationSplitLine).
		SplitActivePane(state.PaneState{ID: "pane-logs", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-logs"}, state.SplitDirectionVertical)
	shell = shell.FocusPane(state.PaneCommandTarget{PaneID: activeShellPaneID})
	for tabIndex := range shell.Workspace.Tabs {
		if shell.Workspace.Tabs[tabIndex].ID != shell.Workspace.ActiveTabID {
			continue
		}
		shell.Workspace.Tabs[tabIndex].RootSplit.Ratio = 0.741
		for paneIndex := range shell.Workspace.Tabs[tabIndex].Panes {
			if shell.Workspace.Tabs[tabIndex].Panes[paneIndex].ID == activeShellPaneID {
				shell.Workspace.Tabs[tabIndex].Panes[paneIndex].TerminalID = "term-shell"
			}
		}
	}
	shell, _ = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "float-visual",
		Title:    "quick actions",
		Pane:     state.PaneState{ID: "float-visual-pane", Title: "actions", Kind: state.PaneEmpty},
		Rect:     state.FloatingRect{X: 105, Y: 7, W: 29, H: 8},
		BoundsW:  140,
		BoundsH:  40,
		Source:   state.PaneCommandSourceTest,
	})
	root := state.Root{
		Shell:    shell,
		Viewport: state.ViewportStore{Valid: true, Cols: 140, Rows: 40},
		Surface:  visualAuditSurfaceStore(),
	}
	visualOwner := state.NewPaneTerminalView(activeShellPaneID, "term-shell", 7, 82, 34, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(activeShellPaneID), true)
	visualOwner.OwnerSurfaceID = visualOwner.SurfaceID
	visualOwner.OwnerViewID = visualOwner.ViewID
	root.TerminalViews = root.TerminalViews.
		BindPane(visualOwner).
		BindPane(state.NewPaneTerminalView("pane-logs", "term-logs", 8, 30, 34, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID("pane-logs"), false))
	runtime := app.NewAppRuntime(root, nil, func(root state.Root) render.Frame {
		vm := builder.Build(root)
		vm.Shell.Footer.GlobalSummary = "ws:main float:1 terminals:1"
		for index := range vm.Shell.Layout.Panels {
			panel := &vm.Shell.Layout.Panels[index]
			// 固定视觉审计要同时展示 active pane、inactive pane 与 active floating 的 chrome 样式。
			panel.Chrome.State = render.ChromeSlotVM{}
			switch panel.ID {
			case activeShellPaneID:
				panel.Active = true
				panel.Chrome.Title.Style = render.StyleAccent
				for actionIndex := range panel.Chrome.Actions {
					panel.Chrome.Actions[actionIndex].Style = render.StyleAccent
				}
			case "pane-logs":
				panel.Active = false
				panel.Chrome.Title.Style = render.StyleMuted
				panel.Chrome.Actions = []render.ChromeActionVM{
					{Text: render.DefaultPaneChromeGlyphs().Close, ActionID: render.ActionPaneClose.String(), Style: render.StyleMuted},
				}
			}
		}
		for index := range vm.Shell.Layout.Floating {
			if vm.Shell.Layout.Floating[index].ID == "float-visual" {
				vm.Shell.Layout.Floating[index].Content = visualAuditFloatingContent()
			}
		}
		return renderer.Render(vm)
	}, host, nil)
	if err := runtime.Post(app.NoopMsg{}); err != nil {
		return render.Frame{}, err
	}
	if err := runtime.Drain(ctx); err != nil {
		return render.Frame{}, err
	}
	frames := host.Frames()
	if len(frames) == 0 {
		return render.Frame{}, fmt.Errorf("visual audit smoke produced no frames")
	}
	return frames[len(frames)-1], nil
}

func visualAuditFloatingContent() render.ContentVM {
	return render.ContentVM{
		Kind: render.ContentEmptyPane,
		Lines: []render.Line{
			render.NewLine(" unconnected"),
			render.NewLine(""),
			render.NewLine(" Attach existing"),
			render.NewLine(" New terminal"),
			render.NewLine(" Terminal Manager"),
			render.NewLine(" Close"),
		},
		Empty: true,
	}
}

func visualAuditSurfaceStore() state.TerminalSurfaceStore {
	surface := (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
		TerminalID: "term-shell",
		Cols:       82,
		Rows:       34,
		Lines: []string{
			"anytty git:core-tui-v3-migration  go v1.26.0",
			"> make test",
			"ok   tui/render",
			">",
		},
	})
	return surface.ApplySnapshot(state.LiveSurfaceSnapshot{
		TerminalID: "term-logs",
		Cols:       30,
		Rows:       34,
		Lines: []string{
			" visual review baseline",
			" target visual mismatch",
			" emoji 🚀 and 中文",
		},
	})
}

package render

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/tui/config"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
)

func bindTestPaneTerminal(root state.Root, paneID string, terminalID string) state.Root {
	if paneID == "" {
		paneID = state.DefaultPaneID
	}
	if terminalID == "" {
		terminalID = root.Surface.TerminalID
	}
	if terminalID == "" {
		terminalID = root.Session.TerminalID
	}
	if terminalID == "" {
		return root
	}
	cols, rows := root.Surface.Cols, root.Surface.Rows
	if cols <= 0 {
		cols = root.Session.Cols
	}
	if rows <= 0 {
		rows = root.Session.Rows
	}
	binding := state.NewPaneTerminalView(paneID, terminalID, 7, cols, rows, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(paneID), true)
	binding.LastError = root.Session.LastError
	root.TerminalViews = root.TerminalViews.BindPane(binding)
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: paneID}, terminalID)
	return root
}

func TestDefaultKeylockStatusRendersAsPortableText(t *testing.T) {
	shell := state.DefaultShell()
	shell.ShortcutPassthroughLocked = true
	root := state.Root{
		Shell:    shell,
		Config:   config.Default(),
		Viewport: state.ViewportStore{Valid: true, Cols: 40, Rows: 8},
	}

	frame := NewRenderer(DefaultTheme()).Render(NewRenderVMBuilder().Build(root))
	footer := frame.Lines[len(frame.Lines)-1]
	if !strings.Contains(footer, "LOCK") || DisplayWidth("LOCK") != 4 {
		t.Fatalf("default keylock state must render as portable visible text, footer=%q", footer)
	}
	if DisplayWidth(footer) != 40 {
		t.Fatalf("default keylock footer must preserve viewport width, width=%d footer=%q", DisplayWidth(footer), footer)
	}
}

func TestRenderVMBuilderUsesCopyModeOnlyWhenBoundToHistory(t *testing.T) {
	root := state.Root{
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       80,
			Rows: []state.HistoryRow{
				{Text: "old", LineID: 10},
				{Text: "new", LineID: 11},
			},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  80,
			Cursor:     state.CopyPosition{Row: 1, Col: 2},
		},
	}

	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	if content.Kind != ContentCopyHistory {
		t.Fatalf("expected copy-history content, got %#v", content)
	}
	if len(content.Lines) != 2 || content.Lines[0].PlainString() != "old" || content.Lines[1].PlainString() != "new" {
		t.Fatalf("unexpected copy-history content lines %#v", content.Lines)
	}
	if len(content.HitRegions) != 2 || content.HitRegions[0].LineID != 10 || content.HitRegions[0].Rect != (Rect{X: 0, Y: 0, W: 3, H: 1}) || content.HitRegions[1].Rect != (Rect{X: 0, Y: 1, W: 3, H: 1}) {
		t.Fatalf("unexpected hit regions %#v", content.HitRegions)
	}
	if !vm.Shell.Cursor.Visible || vm.Shell.Cursor.Row != 1 || vm.Shell.Cursor.Col != 2 {
		t.Fatalf("expected copy cursor VM, got %#v", vm.Shell.Cursor)
	}
}

func TestRenderVMBuilderProjectsCopyHistoryTerminalExtentAndLayout(t *testing.T) {
	viewID := state.TerminalPaneViewID(state.DefaultPaneID)
	root := state.Root{
		Surface: state.TerminalSurfaceStore{TerminalID: "term-1", Cols: 6, Rows: 3},
		Session: state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 6, Rows: 3},
		History: state.HistoryStore{
			PaneID: state.DefaultPaneID, ViewID: viewID, TerminalID: "term-1", Token: "tok-1", Cols: 6,
			Rows: []state.HistoryRow{{Text: "old", LineID: 1}},
		},
		CopyMode: state.CopyModeStore{
			Active: true, PaneID: state.DefaultPaneID, ViewID: viewID, TerminalID: "term-1",
			BoundToken: "tok-1", BoundCols: 6, ViewRows: 3,
		},
	}
	binding := state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 6, 3, state.TerminalResizeRoleFollower, "surface", viewID, false)
	binding.Layout = state.TerminalViewLayout{Mode: state.TerminalViewLayoutCenter}
	root.TerminalViews = root.TerminalViews.BindPane(binding)
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1")

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if content.Kind != ContentCopyHistory || content.Extent != (ContentExtent{Known: true, Cols: 6, Rows: 3}) {
		t.Fatalf("copy history must expose the bound terminal extent, got %#v", content)
	}
	if !content.Layout.Known || content.Layout.Mode != state.TerminalViewLayoutCenter {
		t.Fatalf("copy history must preserve the live terminal layout, got %#v", content.Layout)
	}
}

func TestR418RenderVMBuilderHistoryRenderableDoesNotMixLiveSurface(t *testing.T) {
	viewID := state.TerminalPaneViewID(state.DefaultPaneID)
	root := bindTestPaneTerminal(state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Ready:      true,
			Cols:       80,
			Rows:       10,
			Lines:      []string{"live shell prompt", "live codex frame"},
		},
		Session: state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 10},
		History: state.HistoryStore{
			PaneID:     state.DefaultPaneID,
			ViewID:     viewID,
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       80,
			Rows: []state.HistoryRow{
				{Text: "history shell prompt", LineID: 10},
				{Text: "history codex frame", LineID: 11, Segment: state.HistoryCursorSegmentCurrentPrimaryFrame},
			},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     viewID,
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  80,
		},
	}, state.DefaultPaneID, "term-1")

	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	if content.Kind != ContentCopyHistory {
		t.Fatalf("history renderable pane must render copy-history content, got %#v", content)
	}
	rendered := strings.Join(contentPlainLines(content), "\n")
	if strings.Contains(rendered, "live shell prompt") || strings.Contains(rendered, "live codex frame") {
		t.Fatalf("copy-history content must not mix live surface rows, rendered=%q", rendered)
	}
	if !strings.Contains(rendered, "history shell prompt") || !strings.Contains(rendered, "history codex frame") {
		t.Fatalf("copy-history content must come from authoritative history rows, rendered=%q", rendered)
	}
}

func TestRenderVMBuilderKeepsCopyHistoryOnBoundPaneWhenInactive(t *testing.T) {
	shell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: "pane-2"})
	root := state.Root{
		Shell: shell,
		History: state.HistoryStore{
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       80,
			Rows:       []state.HistoryRow{{Text: "frozen history", LineID: 10}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  80,
		},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-1", 7, 80, 10, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
	}

	vm := NewRenderVMBuilder().Build(root)
	boundPanel := panelByID(vm.Shell.Layout.Panels, state.DefaultPaneID)
	activePanel := panelByID(vm.Shell.Layout.Panels, "pane-2")

	if boundPanel == nil || boundPanel.Content.Kind != ContentCopyHistory || boundPanel.Content.Lines[0].PlainString() != "frozen history" {
		t.Fatalf("copy history must stay on bound pane even when inactive, got %#v", vm.Shell.Layout.Panels)
	}
	if activePanel == nil || !activePanel.Active || activePanel.Content.Kind == ContentCopyHistory {
		t.Fatalf("clicked pane should remain active without stealing copy history, got %#v", vm.Shell.Layout.Panels)
	}
}

func TestRenderVMBuilderUsesBoundPaneViewIDForCopyHistory(t *testing.T) {
	viewID := "cmd/anytty-v3-main"
	root := state.Root{
		Shell: state.DefaultShell(),
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Lines:      []string{"live tail"},
		},
		HistoryByView: map[string]state.HistoryStore{
			viewID: {
				PaneID:     state.DefaultPaneID,
				ViewID:     viewID,
				TerminalID: "term-1",
				Token:      "tok-1",
				Cols:       80,
				Rows:       []state.HistoryRow{{Text: "bound view oldest", LineID: 1}},
			},
		},
		CopyModeByView: map[string]state.CopyModeStore{
			viewID: {
				Active:     true,
				PaneID:     state.DefaultPaneID,
				ViewID:     viewID,
				TerminalID: "term-1",
				BoundToken: "tok-1",
				BoundCols:  80,
			},
		},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-1", 7, 80, 10, state.TerminalResizeRoleOwner, "surface", viewID, true,
		)),
	}

	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	if content.Kind != ContentCopyHistory || len(content.Lines) == 0 || content.Lines[0].PlainString() != "bound view oldest" {
		t.Fatalf("copy history must use bound TerminalView ID, got %#v", content)
	}
}

func TestRenderVMBuilderUsesBoundFloatingViewIDForCopyHistory(t *testing.T) {
	viewID := "anytty-floating-view-main"
	shell := state.DefaultShell()
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	root := state.Root{
		Shell: shell,
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-pane",
			Lines:      []string{"pane live tail"},
		},
		HistoryByView: map[string]state.HistoryStore{
			viewID: {
				PaneID:     "float-pane",
				ViewID:     viewID,
				TerminalID: "term-float",
				Token:      "tok-float",
				Cols:       40,
				Rows:       []state.HistoryRow{{Text: "floating bound oldest", LineID: 1}},
			},
			state.TerminalPaneViewID(state.DefaultPaneID): {
				PaneID:     state.DefaultPaneID,
				ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID: "term-pane",
				Token:      "tok-pane",
				Cols:       80,
				Rows:       []state.HistoryRow{{Text: "tiled history", LineID: 2}},
			},
		},
		CopyModeByView: map[string]state.CopyModeStore{
			viewID: {
				Active:     true,
				PaneID:     "float-pane",
				ViewID:     viewID,
				TerminalID: "term-float",
				BoundToken: "tok-float",
				BoundCols:  40,
			},
			state.TerminalPaneViewID(state.DefaultPaneID): {
				Active:     true,
				PaneID:     state.DefaultPaneID,
				ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID: "term-pane",
				BoundToken: "tok-pane",
				BoundCols:  80,
			},
		},
		TerminalViews: state.TerminalViewStore{}.
			BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-pane", 7, 80, 10, state.TerminalResizeRoleOwner, "surface-pane", state.TerminalPaneViewID(state.DefaultPaneID), true)).
			BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-float", 8, 40, 10, state.TerminalResizeRoleFollower, "surface-float", viewID, true)),
	}

	vm := NewRenderVMBuilder().Build(root)
	floating := floatingByID(vm.Shell.Layout.Floating, "floating-1")
	panel := panelByID(vm.Shell.Layout.Panels, state.DefaultPaneID)
	if floating == nil || floating.Content.Kind != ContentCopyHistory || len(floating.Content.Lines) == 0 || floating.Content.Lines[0].PlainString() != "floating bound oldest" {
		t.Fatalf("floating copy history must use bound TerminalView ID, floating=%#v", floating)
	}
	if panel == nil || panel.Content.Kind != ContentCopyHistory || len(panel.Content.Lines) == 0 || panel.Content.Lines[0].PlainString() != "tiled history" {
		t.Fatalf("tiled panel must keep its own copy history, panel=%#v", panel)
	}
}

func TestRenderVMBuilderDoesNotProjectFloatingCopyHistoryIntoLastActiveTiledPane(t *testing.T) {
	floatingView := "anytty-floating-view-main"
	shell := state.DefaultShell()
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	surface := (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
		TerminalID: "term-pane",
		State:      state.TerminalLiveAttached,
		Lines:      []string{"tiled live tail"},
	})
	root := state.Root{
		Shell:   shell,
		Surface: surface,
		HistoryByView: map[string]state.HistoryStore{
			floatingView: {
				PaneID:     "float-pane",
				ViewID:     floatingView,
				TerminalID: "term-float",
				Token:      "tok-float",
				Cols:       40,
				Rows:       []state.HistoryRow{{Text: "floating history", LineID: 1}},
			},
		},
		CopyModeByView: map[string]state.CopyModeStore{
			floatingView: {
				Active:     true,
				PaneID:     "float-pane",
				ViewID:     floatingView,
				TerminalID: "term-float",
				BoundToken: "tok-float",
				BoundCols:  40,
			},
		},
		TerminalViews: state.TerminalViewStore{}.
			BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-pane", 7, 80, 10, state.TerminalResizeRoleOwner, "surface-pane", state.TerminalPaneViewID(state.DefaultPaneID), true)).
			BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-float", 8, 40, 10, state.TerminalResizeRoleFollower, "surface-float", floatingView, true)),
	}

	vm := NewRenderVMBuilder().Build(root)
	floating := floatingByID(vm.Shell.Layout.Floating, "floating-1")
	panel := panelByID(vm.Shell.Layout.Panels, state.DefaultPaneID)
	if floating == nil || floating.Content.Kind != ContentCopyHistory || len(floating.Content.Lines) == 0 || floating.Content.Lines[0].PlainString() != "floating history" {
		t.Fatalf("floating must render its own copy history, floating=%#v", floating)
	}
	if panel == nil || panel.Active || panel.Content.Kind == ContentCopyHistory || len(panel.Content.Lines) == 0 || panel.Content.Lines[0].PlainString() != "tiled live tail" {
		t.Fatalf("last active tiled pane must not steal floating copy history, panel=%#v", panel)
	}
}

func TestRenderVMBuilderBuildsStructuredChromeSlots(t *testing.T) {
	root := state.Root{
		Shell: state.ShellStore{
			HeaderVisible: true,
			FooterVisible: true,
			Workspace: state.WorkspaceState{
				ID:          "ws-main",
				Name:        "main",
				ActiveTabID: "tab-build",
				Tabs: []state.TabState{
					{ID: "tab-shell", Title: "shell", Panes: []state.PaneState{{ID: "pane-shell", Title: "shell", Kind: state.PaneTerminalLive, TerminalID: "term-1"}}},
					{ID: "tab-build", Title: "build", Panes: []state.PaneState{{ID: "pane-build", Title: "build", Kind: state.PaneTerminalLive, TerminalID: "term-2"}}},
				},
			},
			ActivePaneID: "pane-build",
		},
		Session: state.TerminalSessionStore{TerminalID: "term-2", Attached: true},
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-2",
			Ready:      true,
			Lines:      []string{"build live"},
		},
	}
	root = bindTestPaneTerminal(root, "pane-build", "term-2")

	vm := NewRenderVMBuilder().Build(root)
	if len(vm.Shell.Header.Tabs) != 2 || vm.Shell.Header.Tabs[0].Title != "shell" || vm.Shell.Header.Tabs[1].Title != "build" || !vm.Shell.Header.Tabs[1].Active {
		t.Fatalf("header should expose structured tab slots, got %#v", vm.Shell.Header.Tabs)
	}
	if vm.Shell.Header.Tabs[0].CloseTargetID != "tab-shell" || vm.Shell.Header.Tabs[1].CloseTargetID != "tab-build" {
		t.Fatalf("header tab close slots should carry target tab ids, got %#v", vm.Shell.Header.Tabs)
	}
	if len(vm.Shell.Footer.ActionTokens) == 0 || vm.Shell.Footer.ActionTokens[0].Key != "^P" || vm.Shell.Footer.ActionTokens[0].Label != "PANE" {
		t.Fatalf("footer should expose structured action tokens, got %#v", vm.Shell.Footer.ActionTokens)
	}
	if !containsFooterAction(vm.Shell.Footer.ActionTokens, "^G", "GLOBAL", "menu.system") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "PgUp", "COPY", "menu.copy") {
		t.Fatalf("footer structured tokens should keep catalog labels, got %#v", vm.Shell.Footer.ActionTokens)
	}
	if vm.Shell.Footer.ActionTokens[0].ActionID != "menu.panel" {
		t.Fatalf("footer structured tokens should expose semantic action ids, got %#v", vm.Shell.Footer.ActionTokens)
	}
	if len(vm.Shell.Layout.Panels) != 1 || len(vm.Shell.Layout.Panels[0].Chrome.Actions) != 4 {
		t.Fatalf("pane should expose structured chrome actions, got %#v", vm.Shell.Layout.Panels)
	}
	if vm.Shell.Layout.Panels[0].Chrome.Title.Text != "build" || vm.Shell.Layout.Panels[0].Chrome.Title.Style != StyleAccent {
		t.Fatalf("pane should expose structured title slot, got %#v", vm.Shell.Layout.Panels[0].Chrome.Title)
	}
	if vm.Shell.Layout.Panels[0].Chrome.State.Text != "active" || vm.Shell.Layout.Panels[0].Chrome.State.Style != StyleSuccess {
		t.Fatalf("pane should expose structured active state slot, got %#v", vm.Shell.Layout.Panels[0].Chrome.State)
	}
	if vm.Shell.Layout.Panels[0].Chrome.Actions[0].ActionID != ActionPaneZoom.String() || vm.Shell.Layout.Panels[0].Chrome.Actions[1].ActionID != ActionPaneSplitRight.String() ||
		vm.Shell.Layout.Panels[0].Chrome.Actions[2].ActionID != ActionPaneSplitDown.String() || vm.Shell.Layout.Panels[0].Chrome.Actions[3].ActionID != ActionPaneClose.String() {
		t.Fatalf("pane chrome actions should keep semantic action ids, got %#v", vm.Shell.Layout.Panels[0].Chrome.Actions)
	}
}

func TestRenderVMBuilderAppliesFooterConfig(t *testing.T) {
	root := state.Root{
		Shell:    state.DefaultShell(),
		Viewport: state.ViewportStore{Valid: true, Cols: 120, Rows: 20},
		Config: state.TUIConfigStore{
			Version: 1,
			Footer: state.TUIFooterConfig{
				Templates: state.TUIFooterTemplatesConfig{
					ModeBadge:        "{{mode_icon}} {{mode_label}}",
					Key:              "",
					Action:           "{{key}} {{icon}} {{label}}",
					Separator:        " · ",
					WorkspaceSummary: "󰙅 {{workspace}}",
					FloatingSummary:  "󰹙 {{count}}",
					TerminalsSummary: " {{count}}",
				},
				Modes: map[string]state.TUIFooterModeConfig{
					"live": {
						Icon:  "󰆍",
						Label: "TERM",
						Style: string(StyleFooterAccent),
					},
				},
			},
			Shortcuts: state.TUIShortcutConfig{
				Configured: true,
				Actions: map[string]state.TUIShortcutActionConfig{
					"menu.panel":  {Label: "pane custom"},
					"copy.enter":  {Label: "copy custom"},
					"menu.system": {Label: "global custom"},
				},
				Scenes: map[string]state.TUIShortcutSceneConfig{
					"global": {Bindings: map[string]state.TUIShortcutBindingConfig{
						"ctrl-p": {Action: "menu.panel"},
						"ctrl-v": {Action: "copy.enter"},
						"ctrl-g": {Action: "menu.system"},
					}},
				},
			},
		},
	}

	vm := NewRenderVMBuilder().Build(root)
	footer := vm.Shell.Footer
	if footer.ModeIcon != "󰆍" || footer.ModeLabel != "TERM" || footer.KeyTemplate != "" || !footer.KeyTemplateSet || footer.ActionSeparator != " · " ||
		footer.WorkspaceSummaryTemplate != "󰙅 {{workspace}}" || footer.TerminalsSummaryTemplate != " {{count}}" {
		t.Fatalf("footer config metadata not projected, got %#v", footer)
	}
	if len(footer.ActionTokens) != 3 ||
		!containsFooterAction(footer.ActionTokens, "^P", "pane custom", "menu.panel") ||
		!containsFooterAction(footer.ActionTokens, "^V", "copy custom", "menu.copy") ||
		!containsFooterAction(footer.ActionTokens, "^G", "global custom", "menu.system") {
		t.Fatalf("footer actions should come from shortcuts catalog, got %#v", footer.ActionTokens)
	}
	frame := NewRenderer(DefaultTheme()).Render(vm)
	footerLine := frame.Lines[len(frame.Lines)-1]
	if !strings.Contains(footerLine, "󰆍 TERM") ||
		!strings.Contains(footerLine, "PANE CUSTOM") ||
		!strings.Contains(footerLine, "COPY CUSTOM") ||
		!strings.Contains(footerLine, "󰙅 main") ||
		!strings.Contains(footerLine, " 0") ||
		strings.Contains(footerLine, "ws:") ||
		strings.Contains(footerLine, "terminals:") ||
		strings.Contains(footerLine, "RESIZE") {
		t.Fatalf("footer render should combine footer templates with shortcut actions, got %#v", footerLine)
	}
}

func TestRenderVMBuilderAppliesResizeShortcutConfig(t *testing.T) {
	shell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive}, state.SplitDirectionVertical).
		SetInteractionMode(state.InteractionModeResize)
	root := state.Root{
		Shell:    shell,
		Viewport: state.ViewportStore{Valid: true, Cols: 140, Rows: 20},
		Config: state.TUIConfigStore{
			Version: 1,
			Footer: state.TUIFooterConfig{
				Templates: state.TUIFooterTemplatesConfig{
					ModeBadge:        "{{mode_icon}} {{mode_label}}",
					Key:              "{{key}}",
					Action:           "{{key}} {{icon}} {{label}}",
					Separator:        " · ",
					WorkspaceSummary: "ws:{{workspace}}",
					FloatingSummary:  "float:{{count}}",
					TerminalsSummary: "terminals:{{count}}",
				},
				Modes: map[string]state.TUIFooterModeConfig{
					"resize": {
						Icon:  "󰙖",
						Label: "SIZE",
						Style: string(StyleFooterKeyResize),
					},
				},
			},
			Shortcuts: state.TUIShortcutConfig{
				Configured: true,
				Actions: map[string]state.TUIShortcutActionConfig{
					"resize.center": {Label: "middle"},
				},
				Scenes: map[string]state.TUIShortcutSceneConfig{
					"resize": {Bindings: map[string]state.TUIShortcutBindingConfig{
						"h":      {Action: "resize.left"},
						"l":      {Action: "resize.right"},
						"m":      {Action: "resize.center"},
						"r":      {Action: "resize.layout_reset"},
						"ctrl-g": {Action: "menu.system"},
					}},
				},
			},
		},
	}

	vm := NewRenderVMBuilder().Build(root)
	footer := vm.Shell.Footer
	if footer.Mode != "resize" || footer.ModeIcon != "󰙖" || footer.ModeLabel != "SIZE" {
		t.Fatalf("resize footer config metadata not projected, got %#v", footer)
	}
	for _, want := range []struct {
		key      string
		label    string
		actionID string
	}{
		{key: "h", label: "resize left", actionID: "resize.left"},
		{key: "l", label: "resize right", actionID: "resize.right"},
		{key: "m", label: "middle", actionID: "resize.center"},
		{key: "r", label: "RESET", actionID: "resize.layout_reset"},
		{key: "^G", label: "GLOBAL", actionID: "menu.system"},
	} {
		if !containsFooterAction(footer.ActionTokens, want.key, want.label, want.actionID) {
			t.Fatalf("resize footer config should resolve action %#v, got %#v", want, footer.ActionTokens)
		}
	}

	frame := NewRenderer(DefaultTheme()).Render(vm)
	footerLine := frame.Lines[len(frame.Lines)-1]
	if !strings.Contains(footerLine, "󰙖 SIZE") ||
		!strings.Contains(footerLine, "[h] RESIZE LEFT") ||
		!strings.Contains(footerLine, "[l] RESIZE RIGHT") ||
		!strings.Contains(footerLine, "[m] MIDDLE") ||
		!strings.Contains(footerLine, "[r] RESET") ||
		!strings.Contains(footerLine, "[Ctrl+G] GLOBAL") {
		t.Fatalf("resize footer render should include configured resize actions, got %#v", footerLine)
	}
}

func TestRenderVMBuilderUsesOnlyConfiguredShortcutCatalogForResizeFooter(t *testing.T) {
	root := state.Root{
		Shell:    state.DefaultShell().SetInteractionMode(state.InteractionModeResize),
		Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 20},
		Config: state.TUIConfigStore{
			Version: 1,
			Footer: state.TUIFooterConfig{
				Templates: state.TUIFooterTemplatesConfig{
					ModeBadge:        "{{mode_icon}} {{mode_label}}",
					Key:              "{{key}}",
					Action:           "{{key}} {{icon}} {{label}}",
					Separator:        " · ",
					WorkspaceSummary: "ws:{{workspace}}",
					FloatingSummary:  "float:{{count}}",
					TerminalsSummary: "terminals:{{count}}",
				},
				Modes: map[string]state.TUIFooterModeConfig{
					"resize": {
						Icon:  "󰙖",
						Label: "SIZE",
						Style: string(StyleFooterKeyResize),
					},
				},
			},
			Shortcuts: state.TUIShortcutConfig{
				Configured: true,
				Scenes: map[string]state.TUIShortcutSceneConfig{
					"resize": {Bindings: map[string]state.TUIShortcutBindingConfig{
						"m": {Action: "resize.center"},
						"r": {Action: "resize.layout_reset"},
					}},
				},
			},
		},
	}

	vm := NewRenderVMBuilder().Build(root)
	footer := vm.Shell.Footer
	if containsFooterActionID(footer.ActionTokens, "resize.left") ||
		containsFooterActionID(footer.ActionTokens, "panel.balance") ||
		containsFooterActionID(footer.ActionTokens, "menu.system") {
		t.Fatalf("explicit resize shortcuts must not fall back to default actions, got %#v", footer.ActionTokens)
	}
	for _, want := range []struct {
		key      string
		label    string
		actionID string
	}{
		{key: "m", label: "CENTER", actionID: "resize.center"},
		{key: "r", label: "RESET", actionID: "resize.layout_reset"},
	} {
		if !containsFooterAction(footer.ActionTokens, want.key, want.label, want.actionID) {
			t.Fatalf("configured resize footer should keep shortcut action %#v, got %#v", want, footer.ActionTokens)
		}
	}

	frame := NewRenderer(DefaultTheme()).Render(vm)
	footerLine := frame.Lines[len(frame.Lines)-1]
	if !strings.Contains(footerLine, "󰙖 SIZE") ||
		!strings.Contains(footerLine, "CENTER") ||
		!strings.Contains(footerLine, "[r] RESET") ||
		strings.Contains(footerLine, "GLOBAL") {
		t.Fatalf("configured resize footer must not re-add defaults, got %#v", footerLine)
	}
}

func TestTerminalLiveCellsPreserveFE0FFootprintBeforeDots(t *testing.T) {
	line := terminalLiveLineFromCells([]state.LiveCell{
		{Text: "♻️", Width: 2},
		{Text: "", Width: 0},
		{Text: "♻️", Width: 2},
		{Text: "", Width: 0},
		{Text: "♻️", Width: 2},
		{Text: "", Width: 0},
		{Text: "·", Width: 1},
		{Text: "·", Width: 1},
	})

	if got := line.Width(); got != 8 {
		t.Fatalf("live line must keep protocol footprint width, got %d cells=%#v", got, line.Cells)
	}
	ansi := line.ANSIString(DefaultTheme())
	if !strings.Contains(ansi, "♻️\x1b[1X\x1b[3G♻️") ||
		!strings.Contains(ansi, "♻️\x1b[1X\x1b[5G♻️") ||
		!strings.Contains(ansi, "♻️\x1b[1X\x1b[7G·") {
		t.Fatalf("FE0F live cells should erase continuation columns and re-anchor dots at model column 7, got %q", ansi)
	}
}

func TestTerminalLiveCellsUseProtocolWidthWhenLocalMeasureDiffers(t *testing.T) {
	line := terminalLiveLineFromCells([]state.LiveCell{
		{Text: "♻️", Width: 2},
		{Text: "", Width: 0},
		{Text: "♻️", Width: 2},
		{Text: "", Width: 0},
		{Text: "♻️", Width: 2},
		{Text: "", Width: 0},
		{Text: "·", Width: 1},
		{Text: "·", Width: 1},
	})
	c := newCanvas(8, 1)
	c.writeLine(0, 0, 8, line, "pane-1", LayerPanel)

	if got := c.lines()[0].Width(); got != 8 {
		t.Fatalf("canvas must keep protocol cell widths, got %d cells=%#v", got, c.lines()[0].Cells)
	}
	if got := c.rows[0][0]; got.text != "♻️" || got.width != 2 || got.continuation {
		t.Fatalf("first FE0F cell should occupy two protocol columns, got %#v", got)
	}
	if got := c.rows[0][1]; got.text != "" || !got.continuation {
		t.Fatalf("second FE0F column should remain a continuation cell, got %#v row=%#v", got, c.rows[0])
	}
	if got := c.rows[0][6]; got.text != "·" || got.width != 1 || got.continuation {
		t.Fatalf("dots should start immediately after protocol-width FE0F cells, got %#v row=%#v", got, c.rows[0])
	}
}

func TestRenderVMBuilderProjectsTerminalResizeOwnerChrome(t *testing.T) {
	shell := state.DefaultShell().SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical)
	root := state.Root{Shell: shell.FocusPane(state.PaneCommandTarget{PaneID: "pane-2"})}
	root.TerminalPool = state.TerminalPoolStore{Items: []state.TerminalPoolItem{{TerminalID: "term-1", Title: "shell", State: "running", AttachmentCount: 23}}}
	ownerBinding := state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true)
	ownerBinding.OwnerSurfaceID = "surface"
	ownerBinding.OwnerViewID = "view-1"
	root.TerminalViews = root.TerminalViews.BindPane(ownerBinding)
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, state.TerminalResizeRoleFollower, "surface", "view-2", false))

	vm := NewRenderVMBuilder().Build(root)
	var owner PanelVM
	var follower PanelVM
	for _, panel := range vm.Shell.Layout.Panels {
		switch panel.ID {
		case state.DefaultPaneID:
			owner = panel
		case "pane-2":
			follower = panel
		}
	}
	if len(owner.Chrome.Meta) != 0 {
		t.Fatalf("owner pane should keep resize role out of variable meta slots, got %#v", owner.Chrome.Meta)
	}
	if owner.Chrome.Terminal.Title.Text != "shell" || owner.Chrome.Terminal.State.Text != paneChromeRunningGlyph() || owner.Chrome.Terminal.AttachCount != 23 || owner.Chrome.Terminal.Owner.Text != "◆ owner" || !owner.Chrome.Terminal.CanResize || owner.Chrome.Terminal.Locked {
		t.Fatalf("owner pane should expose tuiv2 terminal chrome facts, got %#v", owner.Chrome.Terminal)
	}
	if len(follower.Chrome.Meta) != 0 {
		t.Fatalf("follower pane should keep resize role out of variable meta slots, got %#v", follower.Chrome.Meta)
	}
	if follower.Chrome.Terminal.AttachCount != 23 || follower.Chrome.Terminal.Owner.Text != "◇ follow" || follower.Chrome.Terminal.Locked {
		t.Fatalf("follower pane should expose tuiv2 follower chrome facts, got %#v", follower.Chrome.Terminal)
	}
	root.Shell = root.Shell.ArmOwnerConfirm("view-2")
	vm = NewRenderVMBuilder().Build(root)
	for _, panel := range vm.Shell.Layout.Panels {
		if panel.ID == "pane-2" {
			follower = panel
		}
	}
	if follower.Chrome.Terminal.Owner.Text != "◆ owner?" || follower.Chrome.Terminal.Owner.Style != StyleWarning || !follower.Chrome.Terminal.TakeOwner {
		t.Fatalf("pending follower pane should expose owner confirmation without authoritative owner, got %#v", follower.Chrome.Terminal)
	}
	if len(owner.Chrome.Actions) != 4 || owner.Chrome.Actions[0].ActionID == ActionTerminalTakeResizeOwner.String() {
		t.Fatalf("owner pane should not offer take-owner action, got %#v", owner.Chrome.Actions)
	}
	if len(follower.Chrome.Actions) != 4 || follower.Chrome.Actions[0].ActionID == ActionTerminalTakeResizeOwner.String() {
		t.Fatalf("follower pane should keep structural actions separate from owner token, got %#v", follower.Chrome.Actions)
	}
}

func TestRenderVMBuilderDoesNotTreatLocalOwnerIntentAsConfirmed(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))

	panel := NewRenderVMBuilder().Build(root).Shell.Layout.Panels[0]
	if panel.Chrome.Terminal.Owner.Text != "◇ follow" || panel.Chrome.Terminal.Owner.Style != StyleMuted || !panel.Chrome.Terminal.TakeOwner {
		t.Fatalf("local owner intent without daemon identity must remain unconfirmed, got %#v", panel.Chrome.Terminal)
	}
}

func TestRenderVMBuilderSeparatesTerminalSizeLockFromViewLayoutLock(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	binding := state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true)
	binding.Layout = binding.Layout.Apply(state.TerminalViewLayoutCommand{Action: "toggle-lock"})
	binding.Layout = binding.Layout.Apply(state.TerminalViewLayoutCommand{Action: "center"})
	root.TerminalViews = root.TerminalViews.BindPane(binding)

	panel := NewRenderVMBuilder().Build(root).Shell.Layout.Panels[0]
	if panel.Chrome.Terminal.Locked {
		t.Fatalf("view-local layout lock must not present as terminal size lock, got %#v", panel.Chrome.Terminal)
	}
	if panel.Chrome.Terminal.LayoutMode != state.TerminalViewLayoutCenter || !terminalChromeLayoutAdjusted(panel.Chrome.Terminal) {
		t.Fatalf("view-local layout state should still project as layout adjustment, got %#v", panel.Chrome.Terminal)
	}

	binding.SizeLocked = true
	root.TerminalViews = root.TerminalViews.BindPane(binding)
	panel = NewRenderVMBuilder().Build(root).Shell.Layout.Panels[0]
	if !panel.Chrome.Terminal.Locked {
		t.Fatalf("core terminal size lock must drive terminal lock chrome, got %#v", panel.Chrome.Terminal)
	}
}

func TestRenderVMBuilderKeepsLockedOwnerChromeAsOwner(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	binding := state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", false)
	binding.SizeLocked = true
	binding.ControlReason = "size_locked"
	binding.OwnerSurfaceID = "surface"
	binding.OwnerViewID = "view-1"
	root.TerminalViews = root.TerminalViews.BindPane(binding)

	panel := NewRenderVMBuilder().Build(root).Shell.Layout.Panels[0]
	if panel.Chrome.Terminal.Owner.Text != "◆ owner" || panel.Chrome.Terminal.TakeOwner || panel.Chrome.Terminal.CanResize || !panel.Chrome.Terminal.Locked {
		t.Fatalf("size-locked owner should stay owner without take-owner action, got %#v", panel.Chrome.Terminal)
	}
}

func TestRenderVMBuilderUsesTerminalPoolTitleForSharedTerminalChrome(t *testing.T) {
	shell := state.DefaultShell().SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-main"}, state.SplitDirectionVertical)
	root := state.Root{
		Shell:        shell,
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{TerminalID: "term-main", Title: "main", State: "running"}}},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-main", 7, 80, 24, state.TerminalResizeRoleFollower, "surface", "view-1", false))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-main", 8, 40, 12, state.TerminalResizeRoleOwner, "surface", "view-2", true))

	vm := NewRenderVMBuilder().Build(root)
	for _, panel := range vm.Shell.Layout.Panels {
		if panel.Chrome.Terminal.TerminalID != "term-main" {
			continue
		}
		if panel.Chrome.Terminal.Title.Text != "main" {
			t.Fatalf("shared terminal chrome should use terminal title, panel=%s terminal=%#v", panel.ID, panel.Chrome.Terminal)
		}
	}
}

func TestRenderVMBuilderUsesConfiguredPaneTitleTemplate(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell(),
		Config: state.TUIConfigStore{Chrome: state.TUIChromeConfig{
			PaneTitleTemplate: "{{terminal}}@{{endpoint}}",
		}},
		Endpoints: state.EndpointStore{}.Upsert(state.EndpointItem{ID: "cn_fast", Label: "CN Fast", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true}),
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{
			EndpointID: "cn_fast",
			TerminalID: "term-cn",
			Title:      "deploy",
			State:      "running",
		}}},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView("cn_fast", state.DefaultPaneID, "term-cn", 7, 80, 24, state.TerminalResizeRoleFollower, "surface-cn", state.TerminalPaneViewID(state.DefaultPaneID), false))

	panel := NewRenderVMBuilder().Build(root).Shell.Layout.Panels[0]
	if panel.Chrome.Terminal.Title.Text != "deploy@CN Fast" {
		t.Fatalf("pane title template should include terminal and endpoint labels, got %#v", panel.Chrome.Terminal.Title)
	}

	root.Config.Chrome.PaneTitleTemplate = "{{"
	panel = NewRenderVMBuilder().Build(root).Shell.Layout.Panels[0]
	if panel.Chrome.Terminal.Title.Text != "deploy" {
		t.Fatalf("invalid pane title template should fall back to terminal title, got %#v", panel.Chrome.Terminal.Title)
	}
}

func TestRenderVMBuilderKeepsChromeActionsForEmptyPane(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	root.Shell.Workspace.Tabs[0].Panes[0].Kind = state.PaneEmpty
	root.Shell.Workspace.Tabs[0].Panes[0].TerminalID = ""
	root.Shell.Workspace.Tabs[0].Panes[0].Title = "unconnected"

	panel := NewRenderVMBuilder().Build(root).Shell.Layout.Panels[0]
	if panel.Content.Kind != ContentEmptyPane || !panel.Content.Empty {
		t.Fatalf("expected empty pane content, got %#v", panel.Content)
	}
	if len(panel.Chrome.Actions) != 4 || panel.Chrome.Actions[0].ActionID != ActionPaneZoom.String() || panel.Chrome.Actions[3].ActionID != ActionPaneClose.String() {
		t.Fatalf("empty pane should keep still-available pane chrome actions, got %#v", panel.Chrome.Actions)
	}
}

func TestRenderVMBuilderProjectsZoomPaneChromeAsUnzoomWithoutSplitActions(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-2"}, state.SplitDirectionVertical).
		ZoomPane(state.PaneCommandTarget{PaneID: "pane-2"})}

	vm := NewRenderVMBuilder().Build(root)
	if len(vm.Shell.Layout.Panels) != 1 {
		t.Fatalf("zoom layout should project only one panel, got %#v", vm.Shell.Layout.Panels)
	}
	panel := vm.Shell.Layout.Panels[0]
	if panel.ID != "pane-2" || !panel.IsZoomMode {
		t.Fatalf("zoom panel should carry zoom projection state, got %#v", panel)
	}
	actions := panel.Chrome.Actions
	if len(actions) != 2 ||
		actions[0].ActionID != ActionPaneZoom.String() ||
		actions[0].Text != paneChromeUnzoomGlyph() ||
		actions[0].Label != "unzoom" ||
		!actions[0].IsZoomMode ||
		actions[1].ActionID != ActionPaneClose.String() ||
		!actions[1].IsZoomMode {
		t.Fatalf("zoom chrome should expose unzoom toggle and close only, got %#v", actions)
	}
	for _, action := range actions {
		if action.ActionID == ActionPaneSplitRight.String() || action.ActionID == ActionPaneSplitDown.String() {
			t.Fatalf("zoom chrome must not expose split actions, got %#v", actions)
		}
	}
}

func TestRenderVMBuilderBuildsGlobalFooterActionIDs(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModeGlobal)}
	actions := NewRenderVMBuilder().Build(root).Shell.Footer.ActionTokens
	want := map[string]string{
		"h": "system.toggle_header",
		"f": "system.toggle_footer",
		"?": "menu.help",
		"t": "menu.terminal_pool",
		"w": "menu.workbench_tree",
		"q": "system.quit",
	}
	for _, action := range actions {
		if id, ok := want[action.Key]; ok && action.ActionID != id {
			t.Fatalf("global footer action %s should use %s, got %#v", action.Key, id, actions)
		}
	}
	for _, forbidden := range []string{
		"system.close_toast",
		"system.clear_toasts",
	} {
		if containsFooterActionID(actions, forbidden) {
			t.Fatalf("global footer should keep toast controls out of the primary path, got %#v", actions)
		}
	}
}

func TestRenderVMBuilderUsesStructuredFooterActionCatalog(t *testing.T) {
	builder := NewRenderVMBuilder()
	tests := []struct {
		name string
		root state.Root
		want []FooterActionVM
	}{
		{
			name: "live",
			root: state.Root{Shell: state.DefaultShell()},
			want: []FooterActionVM{
				{Key: "^P", Label: "PANE", ActionID: "menu.panel"},
				{Key: "^R", Label: "RESIZE", ActionID: "menu.resize"},
				{Key: "^T", Label: "TAB", ActionID: "menu.tab"},
				{Key: "^W", Label: "WORKSPACE", ActionID: "menu.workspace"},
				{Key: "^O", Label: "FLOAT", ActionID: "menu.floating"},
				{Key: "^V", Label: "COPY", ActionID: "menu.copy"},
				{Key: "^F", Label: "PICKER", ActionID: "menu.terminal_picker"},
				{Key: "^G", Label: "GLOBAL", ActionID: "menu.system"},
			},
		},
		{
			name: "terminal pool overlay",
			root: state.Root{
				Shell:        state.DefaultShell().OpenTerminalPool(),
				TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{TerminalID: "term-1", Title: "shell", State: "running"}}},
			},
			want: []FooterActionVM{
				{Key: "enter", Label: "ATTACH", ActionID: "terminal_pool.attach"},
				{Key: "^T", Label: "TAB", ActionID: "terminal_pool.attach_tab"},
				{Key: "^O", Label: "FLOAT", ActionID: "terminal_pool.attach_float"},
				{Key: "^R", Label: "RESTART", ActionID: "terminal_pool.restart"},
				{Key: "^E", Label: "RENAME", ActionID: "terminal_pool.edit"},
				{Key: "^K", Label: "KILL", ActionID: "terminal_pool.kill"},
				{Key: "^X", Label: "REMOVE", ActionID: "terminal_pool.delete"},
			},
		},
		{
			name: "prompt overlay",
			root: state.Root{Shell: state.DefaultShell().OpenPrompt(state.PromptState{})},
			want: []FooterActionVM{
				{Key: "enter", Label: "submit", ActionID: "prompt.submit"},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			footer := builder.Build(tc.root).Shell.Footer
			for _, want := range tc.want {
				if !containsFooterAction(footer.ActionTokens, want.Key, want.Label, want.ActionID) {
					t.Fatalf("missing structured footer action %#v in %#v", want, footer.ActionTokens)
				}
			}
		})
	}
}

func TestRenderVMBuilderRendersCompleteCtrlChordsInLiveFooter(t *testing.T) {
	root := state.Root{
		Shell:    state.DefaultShell(),
		Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 24},
	}

	frame := NewRenderer(DefaultTheme()).Render(NewRenderVMBuilder().Build(root))
	footerLine := frame.Lines[len(frame.Lines)-1]
	if !strings.Contains(footerLine, "[Ctrl+P] PANE") || !strings.Contains(footerLine, "[Ctrl+R] RESIZE") {
		t.Fatalf("live footer must show ctrl-r resize shortcut, got %q", footerLine)
	}
	if strings.Contains(footerLine, "[Ctrl] •") || strings.Contains(footerLine, "[^R]") {
		t.Fatalf("live footer must render each ctrl chord as one unambiguous token, got %q", footerLine)
	}
}

func TestRenderVMBuilderKeepsHelpNavigationCompleteAtDefaultWidth(t *testing.T) {
	root := state.Root{
		Shell:    state.DefaultShell().OpenHelp("most-used"),
		Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 24},
	}
	entries := input.ShortcutEntriesForHelp(root.Config.Shortcuts, root.HostCapabilities.KeyboardDisambiguation)
	root.Shell = root.Shell.SetHelpSelection(10, len(entries))

	frame := NewRenderer(DefaultTheme()).Render(NewRenderVMBuilder().Build(root))
	footerLine := frame.Lines[len(frame.Lines)-1]
	for _, want := range []string{"[↑] PREV", "[↓] NEXT", "[PgUp] PAGE", "[PgDn] PAGE", "[Esc] BACK"} {
		if !strings.Contains(footerLine, want) {
			t.Fatalf("help footer must keep %q at default width, got %q", want, footerLine)
		}
	}
	if strings.Contains(footerLine, "ws:") {
		t.Fatalf("help footer must reserve its row for navigation, got %q", footerLine)
	}
}

func TestRenderVMBuilderKeepsTerminalPoolRestartShortcutVisibleAtDefaultWidth(t *testing.T) {
	root := state.Root{
		Shell:    state.DefaultShell().OpenTerminalPool(),
		Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 24},
		TerminalPool: state.TerminalPoolStore{
			Status: state.TerminalPoolReady,
			Items: []state.TerminalPoolItem{{
				TerminalID: "term-restart",
				Title:      "shell",
				State:      "running",
			}},
		},
	}

	frame := NewRenderer(DefaultTheme()).Render(NewRenderVMBuilder().Build(root))
	rendered := strings.Join(frame.Lines, "\n")
	if !strings.Contains(rendered, "[Ctrl+R] RESTART") {
		t.Fatalf("terminal pool page must keep restart action visible, got %#v", frame.Lines)
	}
}

func TestRenderVMBuilderBuildsPaneStateSlotsFromContent(t *testing.T) {
	tests := []struct {
		name    string
		root    state.Root
		paneID  string
		want    string
		wantSty StyleToken
	}{
		{
			name: "active live",
			root: bindTestPaneTerminal(state.Root{
				Shell:   state.DefaultShell(),
				Surface: state.TerminalSurfaceStore{TerminalID: "term-live", Ready: true, Lines: []string{"live"}},
			}, state.DefaultPaneID, "term-live"),
			paneID:  state.DefaultPaneID,
			want:    "active",
			wantSty: StyleSuccess,
		},
		{
			name: "pending live",
			root: state.Root{
				Shell: state.DefaultShell(),
			},
			paneID:  state.DefaultPaneID,
			want:    "pending",
			wantSty: StyleWarning,
		},
		{
			name: "error live",
			root: bindTestPaneTerminal(state.Root{
				Shell:   state.DefaultShell(),
				Session: state.TerminalSessionStore{TerminalID: "term-live", LastError: "boom"},
			}, state.DefaultPaneID, "term-live"),
			paneID:  state.DefaultPaneID,
			want:    "error",
			wantSty: StyleDanger,
		},
		{
			name: "empty live",
			root: bindTestPaneTerminal(state.Root{
				Shell:   state.DefaultShell(),
				Surface: state.TerminalSurfaceStore{TerminalID: "term-empty", Ready: true},
			}, state.DefaultPaneID, "term-empty"),
			paneID:  state.DefaultPaneID,
			want:    "empty",
			wantSty: StyleMuted,
		},
		{
			name: "inactive ready pane",
			root: func() state.Root {
				shell := state.DefaultShell().
					SplitActivePane(state.PaneState{ID: "pane-logs", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-logs"}, state.SplitDirectionVertical).
					FocusPane(state.PaneCommandTarget{PaneID: "pane-logs"})
				shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-main"
				surface := (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{TerminalID: "term-main", Lines: []string{"main live"}})
				return bindTestPaneTerminal(state.Root{Shell: shell, Surface: surface}, state.DefaultPaneID, "term-main")
			}(),
			paneID:  state.DefaultPaneID,
			want:    "idle",
			wantSty: StyleMuted,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewRenderVMBuilder().Build(tt.root)
			panel := panelByID(vm.Shell.Layout.Panels, tt.paneID)
			if panel == nil {
				t.Fatalf("missing panel %s in %#v", tt.paneID, vm.Shell.Layout.Panels)
			}
			if panel.Chrome.State.Text != tt.want || panel.Chrome.State.Style != tt.wantSty {
				t.Fatalf("unexpected pane state slot: got %#v want text=%q style=%s", panel.Chrome.State, tt.want, tt.wantSty)
			}
		})
	}
}

func TestPaneChromeStateSlotUsesIdleForInactiveReadyContent(t *testing.T) {
	slot := paneChromeStateSlot(false, ContentVM{Kind: ContentTerminalLive, Lines: []Line{NewLine("ready")}})
	if slot.Text != "idle" || slot.Style != StyleMuted {
		t.Fatalf("inactive ready content should use idle state slot, got %#v", slot)
	}
}

func panelByID(panels []PanelVM, id string) *PanelVM {
	for index := range panels {
		if panels[index].ID == id {
			return &panels[index]
		}
	}
	return nil
}

func floatingByID(floatings []FloatingVM, id string) *FloatingVM {
	for index := range floatings {
		if floatings[index].ID == id {
			return &floatings[index]
		}
	}
	return nil
}

func TestRenderVMBuilderProjectsCopyHistoryContentRendererState(t *testing.T) {
	root := state.Root{
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       12,
			TotalLines: 11,
			Rows: []state.HistoryRow{
				{Text: "alpha", LineID: 10, RowInLine: 0, ClippedStart: true},
				{Text: "beta好", LineID: 10, RowInLine: 1, ClippedEnd: true},
				{Text: "gamma", LineID: 11, RowInLine: 0},
			},
			Lines: []state.HistoryLineSpan{
				{LineID: 10, StartRow: 0, EndRow: 1, ClippedBefore: true, ClippedAfter: true},
				{LineID: 11, StartRow: 2, EndRow: 2},
			},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  12,
			Cursor:     state.CopyPosition{Row: 1, Col: 4},
			Selection: &state.CopySelection{
				Anchor: state.CopyPosition{Row: 0, Col: 2},
				Focus:  state.CopyPosition{Row: 1, Col: 4},
			},
		},
	}

	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	if got := content.Lines[0].PlainString(); !strings.Contains(got, "⇡ alpha") {
		t.Fatalf("expected clipped-before marker, got %#v", content.Lines)
	}
	if got := content.Lines[1].PlainString(); got != "beta好 ⇣" {
		t.Fatalf("expected continuation row without engineering marker and with clipped-end marker, got %#v", content.Lines)
	}
	if !lineHasANSICell(content.Lines[0], "pha", copyHistorySelectionANSIStyle) || !lineHasANSICell(content.Lines[1], "beta", copyHistorySelectionANSIStyle) {
		t.Fatalf("expected selection rendered as styled cells, got %#v", content.Lines)
	}
	if !content.Cursor.Visible || content.Cursor.Row != 1 || content.Cursor.Col != 4 {
		t.Fatalf("expected cursor in content coordinates without copy marker offset, got %#v", content.Cursor)
	}
	if !strings.Contains(content.Status, "line 10/11 wrap 2/?") {
		t.Fatalf("expected position status, got %q", content.Status)
	}
	if len(content.HitRegions) != 3 || content.HitRegions[1].LineID != 10 ||
		content.HitRegions[1].Rect != (Rect{X: 0, Y: 1, W: 6, H: 1}) {
		t.Fatalf("expected history hit regions to cover only row text display cells, got %#v", content.HitRegions)
	}
}

func TestRenderVMBuilderDoesNotKeepClippedMarkerAfterBoundaryOverlapMerge(t *testing.T) {
	root := state.Root{
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       12,
			Rows: []state.HistoryRow{
				{Text: "defghi", LineID: 10, RowInLine: 0},
			},
			Lines: []state.HistoryLineSpan{
				{LineID: 10, StartRow: 0, EndRow: 0},
			},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  12,
		},
	}

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if got := content.Lines[0].PlainString(); strings.Contains(got, "⇡") || strings.Contains(got, "⇣") {
		t.Fatalf("boundary-overlap merged logical line should not keep clipped markers, got %#v", content.Lines)
	}
}

func TestRenderVMBuilderCopyHistoryHitRegionsUseAuthoritativeDisplayCells(t *testing.T) {
	root := state.Root{
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       12,
			Rows: []state.HistoryRow{{
				Text:   "a好bc",
				LineID: 10,
				Cells: []state.HistoryCell{
					{Text: "a", Width: 1},
					{Text: "好", Width: 2},
					{Text: "bc", Width: 2},
				},
			}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  12,
		},
	}

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if len(content.HitRegions) != 1 {
		t.Fatalf("expected one history hit region, got %#v", content.HitRegions)
	}
	if got, want := content.HitRegions[0].Rect, (Rect{X: 0, Y: 0, W: 5, H: 1}); got != want {
		t.Fatalf("history hit region should use row display width in content coordinates got=%#v want=%#v", got, want)
	}
}

func TestRenderVMBuilderCopyHistoryEmptyRowKeepsOneCellHitRegion(t *testing.T) {
	root := state.Root{
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       12,
			Rows:       []state.HistoryRow{{Text: "", LineID: 10}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  12,
		},
	}

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if len(content.HitRegions) != 1 {
		t.Fatalf("expected one history hit region, got %#v", content.HitRegions)
	}
	if got, want := content.HitRegions[0].Rect, (Rect{X: 0, Y: 0, W: 1, H: 1}); got != want {
		t.Fatalf("empty authoritative row should keep a one-cell selectable region got=%#v want=%#v", got, want)
	}
}

func TestRenderVMBuilderCopyHistoryShowsAuthoritativeBoundaryTokens(t *testing.T) {
	root := state.Root{
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       12,
			Rows: []state.HistoryRow{
				{Text: "one", LineID: 100},
				{Text: "two", LineID: 101},
				{Text: "three", LineID: 102},
				{Text: "four", LineID: 103},
			},
			Cursor:   state.HistoryCursor{Valid: true, BeforeLineID: 100},
			Boundary: state.HistoryBoundary{FirstLineID: 100, LastLineID: 103},
			HasMore:  true,
		},
		CopyMode: state.CopyModeStore{
			Active:      true,
			TerminalID:  "term-1",
			BoundToken:  "tok-1",
			BoundCols:   12,
			ViewRows:    2,
			ViewportTop: 1,
		},
	}

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if strings.Contains(content.Lines[0].PlainString(), "SCROLL") || strings.Contains(content.Lines[0].PlainString(), "search") {
		t.Fatalf("copy history content should only contain history rows, got %#v", content.Lines)
	}
	if !strings.Contains(content.Status, "older:more") || !strings.Contains(content.Status, "loaded") || !strings.Contains(content.Status, "lines:100-103 view:2-3") {
		t.Fatalf("status should include boundary summary, got %q", content.Status)
	}

	root.CopyMode.ViewportTop = 2
	content = activeContent(NewRenderVMBuilder().Build(root).Shell)
	if !strings.Contains(content.Status, "latest") || !strings.Contains(content.Status, "view:3-4") {
		t.Fatalf("bottom token should switch to latest at loaded tail, got %q", content.Status)
	}

	root.History.Pending = &state.HistoryPendingRequest{ID: 7, Kind: state.HistoryRequestOlder, Token: "tok-1"}
	content = activeContent(NewRenderVMBuilder().Build(root).Shell)
	if !strings.Contains(content.Status, "older:loading") {
		t.Fatalf("status should expose pending older request, got %q", content.Status)
	}

	root.History.Pending = nil
	root.History.HasMore = false
	root.History.Exhausted = state.ExhaustedMarker{}
	content = activeContent(NewRenderVMBuilder().Build(root).Shell)
	if !strings.Contains(content.Status, "older:ready") || strings.Contains(content.Status, "older:more") {
		t.Fatalf("cursor-ready window without HasMore should use neutral older token, got %q", content.Status)
	}

	root.History.Exhausted = state.ExhaustedMarker{
		Valid:    true,
		Token:    "tok-1",
		Cols:     12,
		Cursor:   root.History.Cursor,
		Boundary: root.History.Boundary,
	}
	content = activeContent(NewRenderVMBuilder().Build(root).Shell)
	if !strings.Contains(content.Status, "older:top") {
		t.Fatalf("status should expose exhausted top token, got %q", content.Status)
	}
}

func TestFrameworkRendersCopyHistoryOverflowMarkersOnPaneChrome(t *testing.T) {
	root := state.Root{
		Viewport: state.ViewportStore{Valid: true, Cols: 18, Rows: 8},
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       78,
			Rows: []state.HistoryRow{
				{Text: "0123456789abcdefghijkl", LineID: 10},
				{Text: "row-2", LineID: 11},
				{Text: "row-3", LineID: 12},
				{Text: "row-4", LineID: 13},
				{Text: "row-5", LineID: 14},
				{Text: "row-6", LineID: 15},
				{Text: "row-7", LineID: 16},
			},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  78,
		},
	}

	result := NewRenderer(DefaultTheme()).RenderResult(NewRenderVMBuilder().Build(root))
	layer := firstLayer(t, result, LayerPanel)
	if !layer.ContentOverflow.Right || !layer.ContentOverflow.Bottom {
		t.Fatalf("copy history overflow should be exposed through panel layer, got %#v", layer.ContentOverflow)
	}
	lines := result.Lines()
	rightRow := layer.Rect.Y + layer.Rect.H - 2
	rightCol := layer.Rect.X + layer.Rect.W - 1
	if got := SliceCells(lines[rightRow], rightCol, rightCol+1); got != ">" {
		t.Fatalf("right overflow marker should be drawn on pane right edge, got %q frame=%#v", got, lines)
	}
	bottomRow := layer.Rect.Y + layer.Rect.H - 1
	bottomCol := layer.Rect.X + layer.Rect.W - 2
	if got := SliceCells(lines[bottomRow], bottomCol, bottomCol+1); got != "v" {
		t.Fatalf("bottom overflow marker should be drawn on pane bottom edge, got %q frame=%#v", got, lines)
	}
	cornerCol := layer.Rect.X + layer.Rect.W - 1
	if got := SliceCells(lines[bottomRow], cornerCol, cornerCol+1); got != "┘" {
		t.Fatalf("overflow marker should keep pane bottom-right corner, got %q frame=%#v", got, lines)
	}
	if strings.Contains(strings.Join(plainContentViewportLines(layer.Lines), "\n"), ">") ||
		strings.Contains(strings.Join(plainContentViewportLines(layer.Lines), "\n"), "v") {
		t.Fatalf("overflow markers must stay out of copy history content layer, got %#v", layer.Lines)
	}
}

func TestRenderVMBuilderProjectsCopyHistoryStyledCells(t *testing.T) {
	root := state.Root{
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       12,
			Rows: []state.HistoryRow{{
				Text:   "ERR 好",
				LineID: 10,
				Cells: []state.HistoryCell{
					{Text: "ERR", Width: 3, Style: state.HistoryCellStyle{FG: "ansi:1", Bold: true}},
					{Text: " ", Width: 1},
					{Text: "好", Width: 2, Style: state.HistoryCellStyle{FG: "#ffcc00", Underline: true}, LinkURL: "file://build.log", LinkParams: "line=7"},
				},
			}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  12,
		},
	}

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if got := content.Lines[0].PlainString(); !strings.Contains(got, "ERR 好") {
		t.Fatalf("expected copy history plain text from styled cells, got %#v", content.Lines)
	}
	if !lineHasANSICell(content.Lines[0], "ERR", ANSICellStyle{FG: "ansi:1", Bold: true}) ||
		!lineHasANSICell(content.Lines[0], "好", ANSICellStyle{FG: "#ffcc00", Underline: true}) {
		t.Fatalf("copy history render should preserve history ANSI cells, got %#v", content.Lines[0])
	}
	if !lineHasLinkCell(content.Lines[0], "好", "file://build.log", "line=7") {
		t.Fatalf("copy history render should preserve history link metadata, got %#v", content.Lines[0])
	}

	root.CopyMode.Selection = &state.CopySelection{
		Anchor: state.CopyPosition{Row: 0, Col: 0},
		Focus:  state.CopyPosition{Row: 0, Col: 3},
	}
	content = activeContent(NewRenderVMBuilder().Build(root).Shell)
	if !lineHasANSICell(content.Lines[0], "ERR", copyHistorySelectionANSIStyle) {
		t.Fatalf("selection should override history ANSI style for selected cells, got %#v", content.Lines[0])
	}
	if !lineHasLinkCell(content.Lines[0], "好", "file://build.log", "line=7") {
		t.Fatalf("selection should not drop unselected history link metadata, got %#v", content.Lines[0])
	}
}

func TestRenderVMBuilderCopyHistoryMaterializesPaddedCells(t *testing.T) {
	root := state.Root{
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       40,
			Rows: []state.HistoryRow{{
				Text:   "AGENTS.md   go.work  README.md",
				LineID: 10,
				Cells: []state.HistoryCell{
					{Text: "AGENTS.md", Width: 12, Style: state.HistoryCellStyle{FG: "ansi:4"}},
					{Text: "go.work", Width: 9, Style: state.HistoryCellStyle{FG: "ansi:2"}},
					{Text: "README.md", Width: 9},
				},
			}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  40,
		},
	}

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if got := content.Lines[0].PlainString(); got != "AGENTS.md   go.work  README.md" {
		t.Fatalf("copy history should preserve ls-style padded cells, got %q", got)
	}
}

func TestRendererCopyHistoryPreservesStyledTrailingBlankCells(t *testing.T) {
	style := state.HistoryCellStyle{BG: "idx:24"}
	root := state.Root{
		Viewport: state.ViewportStore{Valid: true, Cols: 18, Rows: 7},
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       8,
			Rows: []state.HistoryRow{{
				Text:   "BG  ",
				LineID: 10,
				Cells: []state.HistoryCell{
					{Text: "B", Width: 1, Style: style},
					{Text: "G", Width: 1, Style: style},
					{Text: " ", Width: 1, Style: style},
					{Text: " ", Width: 1, Style: style},
				},
			}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  8,
		},
	}

	result := NewRenderer(DefaultTheme()).RenderResult(NewRenderVMBuilder().Build(root))
	layer := firstLayer(t, result, LayerPanel)
	if got := layer.Lines[0].PlainString(); !strings.Contains(got, "BG  ") {
		t.Fatalf("copy history content layer should keep styled trailing blanks, got %#v", plainContentViewportLines(layer.Lines))
	}
	frameLine := result.Frame().ANSILines[2]
	if strings.Count(frameLine, "\x1b[48;5;24m ") < 2 ||
		!strings.Contains(frameLine, "\x1b[48;5;24mB") ||
		!strings.Contains(frameLine, "\x1b[48;5;24mG") {
		t.Fatalf("final ANSI frame should keep background over trailing spaces, got %q", frameLine)
	}
}

func TestRenderVMBuilderCopyHistorySelectionPreservesEmptyStyledCellFootprint(t *testing.T) {
	blankStyle := state.HistoryCellStyle{BG: "idx:24"}
	root := state.Root{
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       8,
			Rows: []state.HistoryRow{
				{Text: "before", LineID: 9},
				{
					Text:   "X   Y",
					LineID: 10,
					Cells: []state.HistoryCell{
						{Text: "X", Width: 1},
						{Text: "", Width: 3, Style: blankStyle},
						{Text: "Y", Width: 1},
					},
				},
				{Text: "after", LineID: 11},
			},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  8,
		},
	}

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if got := content.Lines[1].PlainString(); got != "X   Y" {
		t.Fatalf("history render should materialize empty styled footprint, got %q cells=%#v", got, content.Lines[1].Cells)
	}
	if got := lineANSIStyleDisplayWidth(content.Lines[1], ANSICellStyle{BG: "idx:24"}); got != 3 {
		t.Fatalf("empty history footprint should keep original background before selection, got width=%d cells=%#v", got, content.Lines[1].Cells)
	}

	root.CopyMode.Selection = &state.CopySelection{
		Anchor: state.CopyPosition{Row: 0, Col: 0},
		Focus:  state.CopyPosition{Row: 2, Col: 1},
	}
	content = activeContent(NewRenderVMBuilder().Build(root).Shell)
	if got := content.Lines[1].PlainString(); got != "X   Y   " {
		t.Fatalf("selection should not drop empty styled footprint, got %q cells=%#v", got, content.Lines[1].Cells)
	}
	if got := lineANSIStyleDisplayWidth(content.Lines[1], copyHistorySelectionANSIStyle); got != 8 {
		t.Fatalf("selection should cover full visual row including tail blanks, got selected width=%d cells=%#v", got, content.Lines[1].Cells)
	}
}

func TestRenderVMBuilderCopyHistorySelectionAndSearchUseDisplayColumns(t *testing.T) {
	root := state.Root{
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       12,
			Rows: []state.HistoryRow{{
				Text:   "a好bc",
				LineID: 10,
				Cells: []state.HistoryCell{
					{Text: "a", Width: 1, Style: state.HistoryCellStyle{FG: "ansi:2"}},
					{Text: "好", Width: 2, Style: state.HistoryCellStyle{FG: "ansi:3"}, LinkURL: "file://wide.txt", LinkParams: "row=1"},
					{Text: "bc", Width: 2, Style: state.HistoryCellStyle{FG: "ansi:4"}, LinkURL: "file://split.txt", LinkParams: "row=1"},
				},
			}},
		},
		CopyMode: state.CopyModeStore{
			Active:      true,
			TerminalID:  "term-1",
			BoundToken:  "tok-1",
			BoundCols:   12,
			Cursor:      state.CopyPosition{Row: 0, Col: 4},
			Query:       "好b",
			Matches:     []state.CopyMatch{{StartRow: 0, StartCol: 1, EndRow: 0, EndCol: 4}},
			ActiveMatch: 0,
			Selection: &state.CopySelection{
				Anchor: state.CopyPosition{Row: 0, Col: 1},
				Focus:  state.CopyPosition{Row: 0, Col: 4},
			},
		},
	}

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if !content.Cursor.Visible || content.Cursor.Col != 4 {
		t.Fatalf("cursor should use display columns, got %#v", content.Cursor)
	}
	if !lineHasANSICell(content.Lines[0], "好", copyHistorySelectionANSIStyle) || !lineHasANSICell(content.Lines[0], "b", copyHistorySelectionANSIStyle) {
		t.Fatalf("selection should split styled cells by display columns, got %#v", content.Lines[0])
	}
	if !lineHasLinkCell(content.Lines[0], "好", "file://wide.txt", "row=1") || !lineHasLinkCell(content.Lines[0], "b", "file://split.txt", "row=1") {
		t.Fatalf("selection split should preserve link metadata on highlighted cells, got %#v", content.Lines[0])
	}
	if lineHasANSICell(content.Lines[0], "a", copyHistorySelectionANSIStyle) || lineHasANSICell(content.Lines[0], "c", copyHistorySelectionANSIStyle) {
		t.Fatalf("selection should not leak outside display-column range, got %#v", content.Lines[0])
	}

	root.CopyMode.Selection = nil
	content = activeContent(NewRenderVMBuilder().Build(root).Shell)
	if !lineHasStyledCell(content.Lines[0], "好", StyleWarning) || !lineHasStyledCell(content.Lines[0], "b", StyleWarning) {
		t.Fatalf("search should split styled cells by display columns, got %#v", content.Lines[0])
	}
	if !lineHasLinkCell(content.Lines[0], "好", "file://wide.txt", "row=1") || !lineHasLinkCell(content.Lines[0], "b", "file://split.txt", "row=1") {
		t.Fatalf("search split should preserve link metadata on highlighted cells, got %#v", content.Lines[0])
	}
	if !lineHasANSICell(content.Lines[0], "a", ANSICellStyle{FG: "ansi:2"}) ||
		!lineHasANSICell(content.Lines[0], "c", ANSICellStyle{FG: "ansi:4"}) {
		t.Fatalf("unmatched history cells should preserve ANSI style, got %#v", content.Lines[0])
	}
}

func TestRenderVMBuilderCopyHistorySearchUsesGraphemeDisplayColumns(t *testing.T) {
	family := "👨‍👩‍👧‍👦"
	root := state.Root{
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       12,
			Rows: []state.HistoryRow{{
				Text:   family + "x",
				LineID: 10,
			}},
		},
		CopyMode: state.CopyModeStore{
			Active:      true,
			TerminalID:  "term-1",
			BoundToken:  "tok-1",
			BoundCols:   12,
			Query:       family,
			Matches:     []state.CopyMatch{{StartRow: 0, StartCol: 0, EndRow: 0, EndCol: 2}},
			ActiveMatch: 0,
		},
	}

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if !lineHasStyledCell(content.Lines[0], family, StyleWarning) {
		t.Fatalf("search should highlight emoji grapheme as one display range, got %#v", content.Lines[0])
	}
	if lineHasStyledCell(content.Lines[0], "x", StyleWarning) {
		t.Fatalf("search highlight should not leak into trailing x, got %#v", content.Lines[0])
	}
}

func TestRenderVMBuilderCopyHistorySearchHighlightsAcrossReflowRows(t *testing.T) {
	root := state.Root{
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       8,
			Rows: []state.HistoryRow{
				{Text: "alphabe", LineID: 10, RowInLine: 0},
				{Text: "tagamma", LineID: 10, RowInLine: 1},
			},
			Lines: []state.HistoryLineSpan{{LineID: 10, StartRow: 0, EndRow: 1}},
		},
		CopyMode: state.CopyModeStore{
			Active:      true,
			TerminalID:  "term-1",
			BoundToken:  "tok-1",
			BoundCols:   8,
			Query:       "beta",
			Matches:     []state.CopyMatch{{StartRow: 0, StartCol: 5, EndRow: 1, EndCol: 2}},
			ActiveMatch: 0,
		},
	}

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if !lineHasStyledCell(content.Lines[0], "be", StyleWarning) {
		t.Fatalf("first reflow row should highlight cross-row search suffix, got %#v", content.Lines[0])
	}
	if !lineHasStyledCell(content.Lines[1], "ta", StyleWarning) {
		t.Fatalf("second reflow row should highlight cross-row search prefix, got %#v", content.Lines[1])
	}
	if lineHasStyledCell(content.Lines[1], "gamma", StyleWarning) {
		t.Fatalf("cross-row search highlight must not leak past match end, got %#v", content.Lines[1])
	}
}

func TestRenderVMBuilderShowsPendingWithoutAuthoritativeHistory(t *testing.T) {
	root := state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Lines:      []string{"live-row"},
		},
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-history",
			Cols:       80,
			Rows:       []state.HistoryRow{{Text: "old", LineID: 10}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-1",
			BoundToken: "tok-copy",
			BoundCols:  80,
		},
	}

	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	if content.Kind != ContentCopyHistory || !content.Pending {
		t.Fatalf("expected pending copy-history content, got %#v", content)
	}
	if len(content.Lines) != 1 || !strings.Contains(content.Lines[0].PlainString(), "stale history token") {
		t.Fatalf("copy mode must not fallback to live rows, got %#v", content.Lines)
	}
}

func TestRenderVMBuilderKeepsLiveFrameWhileCopyModeEntering(t *testing.T) {
	root := state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Lines:      []string{"new-live-row"},
		},
		CopyMode: state.CopyModeStore{
			Entering:   true,
			Phase:      state.CopyModeEnteringPending,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			BoundCols:  80,
		},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-1", 4, 80, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
	}

	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	if content.Kind != ContentTerminalLive || content.Pending {
		t.Fatalf("entering copy mode should keep live content until latest arrives, got %#v", content)
	}
	if len(content.Lines) != 1 || content.Lines[0].PlainString() != "new-live-row" {
		t.Fatalf("entering copy mode should keep ordinary live render outside history path, got %#v", content.Lines)
	}
}

func TestRenderVMBuilderShowsPendingAfterCopyResizeInvalidation(t *testing.T) {
	root := state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Lines:      []string{"live-row"},
		},
		History: state.HistoryStore{
			TerminalID: "term-1",
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-1",
			BoundCols:  98,
		},
	}

	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	if content.Kind != ContentCopyHistory || !content.Pending {
		t.Fatalf("expected pending copy-history after resize invalidation, got %#v", content)
	}
	if len(content.Lines) != 1 || !strings.Contains(content.Lines[0].PlainString(), "window pending") {
		t.Fatalf("copy mode resize pending must not fallback to live rows, got %#v", content.Lines)
	}
}

func TestRenderVMBuilderShowsCopyHistoryEmptyWithoutLiveFallback(t *testing.T) {
	root := state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Lines:      []string{"live-row"},
		},
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       80,
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  80,
		},
	}

	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	if content.Kind != ContentCopyHistory || !content.Empty || content.Pending {
		t.Fatalf("expected empty copy-history content, got %#v", content)
	}
	if len(content.Lines) != 1 || content.Lines[0].PlainString() != "copy history empty" {
		t.Fatalf("copy mode empty must not fallback to live rows, got %#v", content.Lines)
	}
}

func TestRenderVMBuilderShowsPendingWhenCopyPaneBindingMissing(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell(),
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Lines:      []string{"live-row"},
		},
		History: state.HistoryStore{
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       80,
			Rows:       []state.HistoryRow{{Text: "old", LineID: 10}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  80,
		},
	}

	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	if content.Kind != ContentCopyHistory || !content.Pending {
		t.Fatalf("missing pane binding should keep copy-history pending, got %#v", content)
	}
	if len(content.Lines) != 1 || !strings.Contains(content.Lines[0].PlainString(), "copy binding missing") {
		t.Fatalf("missing pane binding must not render stale history rows, got %#v", content.Lines)
	}
}

func TestRenderVMBuilderShowsPendingWhenCopyFloatingBindingMissing(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell(),
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-float",
			Lines:      []string{"live-row"},
		},
		History: state.HistoryStore{
			PaneID:     "float-pane",
			ViewID:     state.TerminalFloatingViewID("floating-1"),
			TerminalID: "term-float",
			Token:      "tok-float",
			Cols:       40,
			Rows:       []state.HistoryRow{{Text: "float-history", LineID: 20}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     "float-pane",
			ViewID:     state.TerminalFloatingViewID("floating-1"),
			TerminalID: "term-float",
			BoundToken: "tok-float",
			BoundCols:  40,
		},
	}

	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	hasStaleHistory := false
	for _, line := range content.Lines {
		if strings.Contains(line.PlainString(), "float-history") {
			hasStaleHistory = true
			break
		}
	}
	if content.Kind == ContentCopyHistory || hasStaleHistory {
		t.Fatalf("missing floating binding must not project stale floating history into active body, got %#v", content)
	}
}

func TestRenderVMBuilderBuildsProductHeaderFooterSummaries(t *testing.T) {
	shell := state.DefaultShell().
		SetInteractionMode(state.InteractionModePane).
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "日志 🚀", Kind: state.PaneTerminalLive}, state.SplitDirectionVertical)
	shell, _ = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "float-1",
		Pane:     state.PaneState{ID: "float-pane", Title: "浮窗", Kind: state.PaneEmpty},
		Rect:     state.FloatingRect{X: 4, Y: 3, W: 30, H: 8},
	})
	root := state.Root{
		Shell: shell,
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-live",
			Title:      "anytty",
			Lines:      []string{"live"},
		},
		Session: state.TerminalSessionStore{
			TerminalID: "term-live",
			Attached:   true,
		},
	}
	root = bindTestPaneTerminal(root, "pane-2", "term-live")

	vm := NewRenderVMBuilder().Build(root)
	header := vm.Shell.Header
	if !header.Visible || header.Workspace != "main" || header.Tab != "[main]" || header.ActivePane != "pane-2" || header.TerminalSummary != "term:1" || header.FloatingSummary != "float:1" {
		t.Fatalf("unexpected product header %#v", header)
	}
	footer := vm.Shell.Footer
	if !footer.Visible || footer.Mode != "pane" || footer.ActiveTarget != "float:浮窗" ||
		containsFooterAction(footer.ActionTokens, "v", "split", "") ||
		!strings.Contains(footer.GlobalSummary, "ws:main") || !strings.Contains(footer.GlobalSummary, "float:1") || !strings.Contains(footer.GlobalSummary, "terminals:1") {
		t.Fatalf("unexpected product footer %#v", footer)
	}
	if len(vm.Shell.Layout.Floating) != 1 || vm.Shell.Layout.Floating[0].Title != "浮窗" || !vm.Shell.Layout.Floating[0].Active {
		t.Fatalf("expected floating VM projection, got %#v", vm.Shell.Layout.Floating)
	}
}

func TestRenderVMBuilderHidesEmptyPaneCursor(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	root.Shell.Workspace.Tabs[0].Panes[0] = state.PaneState{ID: state.DefaultPaneID, Title: "浮窗", Kind: state.PaneEmpty, Active: true}

	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	if content.Kind != ContentEmptyPane || content.Cursor.Visible || content.Cursor.Anchor {
		t.Fatalf("empty pane should not expose cursor or IME anchor, got %#v", content)
	}
}

func TestRenderVMBuilderDimsTiledPaneWhenFloatingOwnsFocus(t *testing.T) {
	shell := state.DefaultShell()
	shell, result := shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "float-1",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneEmpty},
		Rect:     state.FloatingRect{X: 4, Y: 3, W: 30, H: 8},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}

	vm := NewRenderVMBuilder().Build(state.Root{Shell: shell})
	if len(vm.Shell.Layout.Panels) != 1 || vm.Shell.Layout.Panels[0].Active {
		t.Fatalf("active floating should dim tiled pane chrome, panels=%#v floating=%#v", vm.Shell.Layout.Panels, vm.Shell.Layout.Floating)
	}
	if len(vm.Shell.Layout.Floating) != 1 || !vm.Shell.Layout.Floating[0].Active {
		t.Fatalf("expected active floating VM, got %#v", vm.Shell.Layout.Floating)
	}

	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandDeactivate})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("deactivate floating: %#v", result)
	}
	vm = NewRenderVMBuilder().Build(state.Root{Shell: shell})
	if len(vm.Shell.Layout.Panels) != 1 || !vm.Shell.Layout.Panels[0].Active {
		t.Fatalf("tiled pane should regain active visual while inactive floating remains open, got %#v", vm.Shell.Layout.Panels)
	}
	if len(vm.Shell.Layout.Floating) != 1 || vm.Shell.Layout.Floating[0].Active {
		t.Fatalf("floating should remain visible but inactive, got %#v", vm.Shell.Layout.Floating)
	}

	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandFocusRaise, TargetID: "float-1"})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("raise floating: %#v", result)
	}
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandClose, TargetID: "float-1"})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("close floating: %#v", result)
	}
	vm = NewRenderVMBuilder().Build(state.Root{Shell: shell})
	if len(vm.Shell.Layout.Panels) != 1 || !vm.Shell.Layout.Panels[0].Active {
		t.Fatalf("tiled pane active visual should restore after floating closes, got %#v", vm.Shell.Layout.Panels)
	}
}

func TestRenderVMBuilderProjectsTabStripAndWorkspaceSummary(t *testing.T) {
	shell := state.DefaultShell().
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-main").
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-logs"}, state.SplitDirectionVertical)
	shell = shell.SetInteractionMode(state.InteractionModePane)
	root := state.Root{Shell: shell}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-logs", 7, 80, 24, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID("pane-2"), false))
	vm := NewRenderVMBuilder().Build(root)
	if vm.Shell.Footer.Mode != "pane" ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "x/w", "CLOSE", "panel.close") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "d", "DETACH", "panel.detach") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "n/N/h/k/l/j/←/↑/→/↓", "FOCUS", "panel.focus_next") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "%/^D", "VSPLIT", "panel.split_right") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "\"/^E", "HSPLIT", "panel.split_down") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "z", "ZOOM", "panel.toggle_zoom") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "b", "BALANCE", "panel.balance") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "c", "CARD", "panel.presentation_card") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "p", "LINE", "panel.presentation_split_line") {
		t.Fatalf("expected pane footer structural actions, got %#v", vm.Shell.Footer)
	}
	shell = shell.FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	shell = shell.SetInteractionMode(state.InteractionModeResize)
	vm = NewRenderVMBuilder().Build(state.Root{Shell: shell})
	if vm.Shell.Footer.Mode != "resize" ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "←/h/H", "resize left", "resize.left") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "→/l/L", "resize right", "resize.right") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "↑/k/K", "resize up", "resize.up") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "↓/j/J", "resize down", "resize.down") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "b/=", "BALANCE", "panel.balance") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "s", "LOCK", "panel.size_lock") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "space", "LAYOUT", "resize.layout_toggle") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "A/S/W/D/Shift+←/Shift+↓/Shift+↑/Shift+→", "PAN", "resize.pan_left") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "0/$/^/B", "ALIGN", "resize.align_left") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "m/|/_", "CENTER", "resize.center") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "r", "RESET", "resize.layout_reset") {
		t.Fatalf("expected resize footer structural actions, got %#v", vm.Shell.Footer)
	}
	shell, _ = shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandCreate, TargetID: "float-1", Pane: state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"}})
	shell = shell.SetInteractionMode(state.InteractionModeFloating)
	vm = NewRenderVMBuilder().Build(state.Root{Shell: shell})
	if vm.Shell.Footer.Mode != "floating" ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "n", "NEW FLOAT", "floating.new") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "f", "PICKER", "menu.terminal_picker") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "a", "OWNER", "floating.take_owner") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "c", "CENTER", ActionFloatingCenter.String()) ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "z/m", "HIDE", ActionFloatingCollapse.String()) ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "x", "CLOSE", ActionFloatingClose.String()) {
		t.Fatalf("expected floating footer structural actions, got %#v", vm.Shell.Footer)
	}

	shell, _ = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: "logs"})
	shell, _ = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceCreate, Name: "remote"})
	shell = shell.SetInteractionMode(state.InteractionModeWorkspace)

	vm = NewRenderVMBuilder().Build(state.Root{Shell: shell})
	if vm.Shell.Header.Workspace != "remote" || vm.Shell.Header.Tab != "[main]" {
		t.Fatalf("expected active workspace header, got %#v", vm.Shell.Header)
	}
	if vm.Shell.Footer.Mode != "workspace" ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "c", "NEW", "workspace.create") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "p/h/[", "PREV", "workspace.previous") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "n/l/]", "NEXT", "workspace.next") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "r", "RENAME", "workspace.rename") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "t/f/s", "TREE", "menu.workbench_tree") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "x", "DELETE", "workspace.delete") ||
		!strings.Contains(vm.Shell.Footer.GlobalSummary, "ws:remote") {
		t.Fatalf("expected workspace footer summary, got %#v", vm.Shell.Footer)
	}

	shell, _ = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspacePrevious})
	vm = NewRenderVMBuilder().Build(state.Root{Shell: shell})
	if vm.Shell.Header.Workspace != "main" || vm.Shell.Header.Tab != "main [logs]" {
		t.Fatalf("expected original workspace tab strip, got %#v", vm.Shell.Header)
	}
	shell = shell.SetInteractionMode(state.InteractionModeTab)
	vm = NewRenderVMBuilder().Build(state.Root{Shell: shell})
	if vm.Shell.Footer.Mode != "tab" ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "p/h/[", "PREV", "tab.previous") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "n/l/]", "NEXT", "tab.next") ||
		!containsFooterAction(vm.Shell.Footer.ActionTokens, "r", "RENAME", "tab.rename") {
		t.Fatalf("expected tab footer rename action, got %#v", vm.Shell.Footer)
	}
	shell = shell.SetInteractionMode(state.InteractionModeGlobal)
	vm = NewRenderVMBuilder().Build(state.Root{Shell: shell})
	if vm.Shell.Footer.Mode != "global" || !containsFooterAction(vm.Shell.Footer.ActionTokens, "q", "QUIT", "system.quit") {
		t.Fatalf("expected global footer quit action, got %#v", vm.Shell.Footer)
	}
}

func TestRenderVMBuilderFiltersUnavailableFooterActions(t *testing.T) {
	builder := NewRenderVMBuilder()

	paneFooter := builder.Build(state.Root{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModePane)}).Shell.Footer
	if containsFooterActionID(paneFooter.ActionTokens, "panel.close") ||
		containsFooterActionID(paneFooter.ActionTokens, "panel.focus_next") ||
		containsFooterActionID(paneFooter.ActionTokens, "panel.detach") {
		t.Fatalf("single empty pane footer should hide unavailable pane actions, got %#v", paneFooter.ActionTokens)
	}

	resizeFooter := builder.Build(state.Root{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModeResize)}).Shell.Footer
	if containsFooterActionID(resizeFooter.ActionTokens, "resize.left") ||
		containsFooterActionID(resizeFooter.ActionTokens, "panel.balance") {
		t.Fatalf("single pane resize footer should hide unavailable resize actions, got %#v", resizeFooter.ActionTokens)
	}

	tabFooter := builder.Build(state.Root{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModeTab)}).Shell.Footer
	if containsFooterActionID(tabFooter.ActionTokens, "tab.next") ||
		containsFooterActionID(tabFooter.ActionTokens, "tab.previous") {
		t.Fatalf("single tab footer should hide tab switch actions, got %#v", tabFooter.ActionTokens)
	}

	workspaceFooter := builder.Build(state.Root{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModeWorkspace)}).Shell.Footer
	if containsFooterActionID(workspaceFooter.ActionTokens, "workspace.next") ||
		containsFooterActionID(workspaceFooter.ActionTokens, "workspace.previous") {
		t.Fatalf("single workspace footer should hide workspace switch actions, got %#v", workspaceFooter.ActionTokens)
	}

	floatingFooter := builder.Build(state.Root{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModeFloating)}).Shell.Footer
	if containsFooterActionID(floatingFooter.ActionTokens, ActionFloatingClose.String()) ||
		containsFooterActionID(floatingFooter.ActionTokens, ActionFloatingCenter.String()) ||
		containsFooterActionID(floatingFooter.ActionTokens, "floating.toggle_all") {
		t.Fatalf("floating footer without floatings should hide target/group actions, got %#v", floatingFooter.ActionTokens)
	}

	poolFooter := builder.Build(state.Root{Shell: state.DefaultShell().OpenTerminalPool()}).Shell.Footer
	if containsFooterActionID(poolFooter.ActionTokens, "terminal_pool.attach") ||
		containsFooterActionID(poolFooter.ActionTokens, "terminal_pool.attach_tab") ||
		containsFooterActionID(poolFooter.ActionTokens, "terminal_pool.restart") ||
		containsFooterActionID(poolFooter.ActionTokens, "terminal_pool.kill") ||
		containsFooterActionID(poolFooter.ActionTokens, "terminal_pool.delete") {
		t.Fatalf("empty terminal pool footer should hide item actions, got %#v", poolFooter.ActionTokens)
	}
}

func TestRenderVMBuilderDerivesFooterModePrecedence(t *testing.T) {
	builder := NewRenderVMBuilder()
	if got := builder.Build(state.Root{
		CopyMode: state.CopyModeStore{Active: true, TerminalID: "term-copy", BoundCols: 80},
	}).Shell.Footer.Mode; got != "copy" {
		t.Fatalf("expected copy footer mode, got %q", got)
	}
	copyFooter := builder.Build(state.Root{
		CopyMode: state.CopyModeStore{Active: true, TerminalID: "term-copy", BoundCols: 80},
	}).Shell.Footer
	if !containsFooterAction(copyFooter.ActionTokens, "PgUp", "SCROLL", "copy.request_older") {
		t.Fatalf("expected copy footer older action id, got %#v", copyFooter.ActionTokens)
	}
	if !containsFooterAction(copyFooter.ActionTokens, "H", "CLIPBOARD", "menu.clipboard_history") {
		t.Fatalf("expected copy footer clipboard action id, got %#v", copyFooter.ActionTokens)
	}
	if got := builder.Build(state.Root{
		Shell:    state.DefaultShell().OpenTerminalPicker().SetInteractionMode(state.InteractionModeGlobal),
		CopyMode: state.CopyModeStore{Active: true, TerminalID: "term-copy", BoundCols: 80},
	}).Shell.Footer.Mode; got != "terminal-picker" {
		t.Fatalf("overlay should win footer mode, got %q", got)
	}
	if got := builder.Build(state.Root{
		Shell: state.DefaultShell().SetInteractionMode(state.InteractionModeResize),
	}).Shell.Footer.Mode; got != "resize" {
		t.Fatalf("expected interaction footer mode, got %q", got)
	}
	copyBeforeInteraction := builder.Build(state.Root{
		Shell:    state.DefaultShell().SetInteractionMode(state.InteractionModeResize),
		CopyMode: state.CopyModeStore{Active: true, TerminalID: "term-copy", BoundCols: 80},
	}).Shell.Footer
	if copyBeforeInteraction.Mode != "copy" || !containsFooterAction(copyBeforeInteraction.ActionTokens, "Esc", "BACK", "shortcut.exit") {
		t.Fatalf("current copy must own footer before sticky interaction, got %#v", copyBeforeInteraction)
	}

	foreignCopy := state.Root{
		Shell: state.DefaultShell(),
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-live", 4, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
		CopyMode: state.CopyModeStore{Active: true, ViewID: "pane:other", TerminalID: "term-other", BoundCols: 80},
	}
	foreignFooter := builder.Build(foreignCopy).Shell.Footer
	if foreignFooter.Mode != "live" || containsFooterAction(foreignFooter.ActionTokens, "Esc", "BACK", "shortcut.exit") {
		t.Fatalf("non-current copy session must not change active footer, got %#v", foreignFooter)
	}
}

func TestRenderVMBuilderShowsCopyHistoryBindingErrorWithoutLiveFallback(t *testing.T) {
	root := state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-live",
			Lines:      []string{"live-row"},
		},
		History: state.HistoryStore{
			TerminalID: "term-history",
			Token:      "tok-1",
			Cols:       80,
			Rows:       []state.HistoryRow{{Text: "old", LineID: 10}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-copy",
			BoundToken: "tok-1",
			BoundCols:  80,
		},
	}

	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	if content.Kind != ContentCopyHistory || content.Error == "" {
		t.Fatalf("expected copy-history binding error, got %#v", content)
	}
	if len(content.Lines) != 1 || !strings.Contains(content.Lines[0].PlainString(), "terminal mismatch") {
		t.Fatalf("copy mode error must not fallback to live rows, got %#v", content.Lines)
	}
}

func TestRenderVMBuilderBuildsShellPanelToastAndOverlayVM(t *testing.T) {
	shell := state.DefaultShell().
		SetHeaderVisible(false).
		SetFooterVisible(false).
		SetPanelPresentation(state.PanelPresentationSplitLine).
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive}, state.SplitDirectionHorizontal).
		AddToast(state.ToastSpec{ID: "toast-1", Severity: state.ToastWarning, Title: "warn", Body: "body", Pending: true}).
		OpenTerminalPicker()
	root := state.Root{
		Shell: shell,
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-live",
			Lines:      []string{"prompt"},
		},
	}

	vm := NewRenderVMBuilder().Build(root)
	if vm.Shell.Header.Visible || vm.Shell.Footer.Visible {
		t.Fatalf("expected hidden header/footer, got header=%#v footer=%#v", vm.Shell.Header, vm.Shell.Footer)
	}
	if len(vm.Shell.Layout.Panels) != 2 {
		t.Fatalf("expected two panel VMs, got %#v", vm.Shell.Layout.Panels)
	}
	if vm.Shell.Layout.Panels[0].Presentation != PanelPresentationSplitLine || vm.Shell.Layout.Panels[1].Presentation != PanelPresentationSplitLine {
		t.Fatalf("expected split line presentation, got %#v", vm.Shell.Layout.Panels)
	}
	if !vm.Shell.Layout.Panels[1].Active || activeContent(vm.Shell).Kind != ContentTerminalLive {
		t.Fatalf("expected active live content in active panel, got %#v", vm.Shell.Layout.Panels)
	}
	if vm.Shell.Overlay.Kind != OverlayTerminalPicker || vm.Shell.Overlay.Content.Kind != ContentTerminalPicker {
		t.Fatalf("expected terminal picker overlay VM, got %#v", vm.Shell.Overlay)
	}
	if len(vm.Shell.Toasts) != 1 || vm.Shell.Toasts[0].Severity != ToastWarning || !vm.Shell.Toasts[0].Pending {
		t.Fatalf("expected warning toast VM, got %#v", vm.Shell.Toasts)
	}
}

func TestRenderVMBuilderProjectsFloatingOverviewOverlay(t *testing.T) {
	shell := state.DefaultShell()
	shell, _ = shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandCreate, TargetID: "floating-1", Title: "logs", Pane: state.PaneState{ID: "floating-pane-1", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-logs"}, Rect: state.FloatingRect{X: 4, Y: 3, W: 32, H: 9}})
	shell, _ = shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandToggleAutoFit, TargetID: "floating-1", FitCols: 48, FitRows: 14, BoundsW: 100, BoundsH: 30})
	shell = shell.OpenFloatingOverview()

	root := state.Root{
		Shell: shell,
		TerminalViews: state.TerminalViewStore{}.BindFloating(
			state.NewFloatingTerminalView("floating-1", "floating-pane-1", "term-logs", 7, 48, 14, state.TerminalResizeRoleOwner, "surface-1", "", true),
		),
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{TerminalID: "term-logs", Title: "logs", State: "running", Cols: 48, Rows: 14}}},
	}
	vm := NewRenderVMBuilder().Build(root)
	if vm.Shell.Overlay.Kind != OverlayFloatingOverview || vm.Shell.Overlay.Content.Kind != ContentFloatingOverview {
		t.Fatalf("expected floating overview overlay VM, got %#v", vm.Shell.Overlay)
	}
	content := vm.Shell.Overlay.Content
	if content.Status != "floating windows: 1 items" || len(content.HitRegions) == 0 ||
		!strings.Contains(content.Lines[0].PlainString(), "terminal") ||
		!strings.Contains(content.Lines[0].PlainString(), "state") ||
		!strings.Contains(content.Lines[1].PlainString(), "logs") ||
		!strings.Contains(content.Lines[1].PlainString(), "running") ||
		!strings.Contains(content.Lines[1].PlainString(), "48x14") ||
		!strings.Contains(content.Lines[1].PlainString(), "term-logs") {
		t.Fatalf("expected floating overview content with action regions, got %#v", content)
	}
	if strings.Contains(content.Lines[0].PlainString(), "Restore, collapse") ||
		strings.Contains(content.Lines[1].PlainString(), "auto-fit") {
		t.Fatalf("floating overview should render terminal table without old help copy, got %#v", content.Lines)
	}
	if !lineHasStyledANSICell(content.Lines[1], "▸ ", StylePickerAccent, ANSICellStyle{}) {
		t.Fatalf("selected floating row should use picker marker, got %#v", content.Lines[1])
	}
	header := content.Lines[0].PlainString()
	row := content.Lines[1].PlainString()
	for _, pair := range [][2]string{{"state", "running"}, {"size", "48x14"}, {"floating", "term-logs"}} {
		if displayColumnOf(header, pair[0]) != displayColumnOf(row, pair[1]) {
			t.Fatalf("floating overview column %q should align with row value %q, header=%q row=%q", pair[0], pair[1], header, row)
		}
	}
	if !containsFooterAction(vm.Shell.Footer.ActionTokens, "1-9", "SUMMON", ActionFloatingSummon.String()) {
		t.Fatalf("floating overview footer should expose summon action, got %#v", vm.Shell.Footer)
	}
	if !containsFooterAction(vm.Shell.Footer.ActionTokens, "s", "SHOW ALL", "floating_overview.show_all") {
		t.Fatalf("floating overview footer should expose show-all action, got %#v", vm.Shell.Footer)
	}
	if !containsFooterAction(vm.Shell.Footer.ActionTokens, "c", "COLLAPSE ALL", "floating_overview.collapse_all") {
		t.Fatalf("floating overview footer should expose collapse-all action, got %#v", vm.Shell.Footer)
	}
}

func TestRenderVMBuilderProjectsUnconnectedFloatingOverviewRow(t *testing.T) {
	shell := state.DefaultShell()
	shell, result := shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-empty",
		Title:    "floating",
		Pane:     state.PaneState{ID: "floating-empty-pane", Title: "floating", Kind: state.PaneEmpty},
		Rect:     state.FloatingRect{X: 4, Y: 3, W: 32, H: 9},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	shell = shell.OpenFloatingOverview()

	vm := NewRenderVMBuilder().Build(state.Root{Shell: shell})
	content := vm.Shell.Overlay.Content
	if len(content.Lines) < 2 ||
		!strings.Contains(content.Lines[1].PlainString(), "unconnected") ||
		!strings.Contains(content.Lines[1].PlainString(), "empty") ||
		strings.Contains(content.Lines[1].PlainString(), " floating  empty") {
		t.Fatalf("unconnected floating row should show connection state, got %#v", content.Lines)
	}
}

func TestRenderVMBuilderProjectsClipboardHistoryOverlay(t *testing.T) {
	shell := state.DefaultShell().OpenClipboardHistory()
	root := state.Root{
		Shell: shell,
		Clipboard: state.ClipboardStore{
			Entries: []state.ClipboardEntry{
				{ID: "clip:1", Title: "alpha", Text: "alpha", Preview: "alpha"},
				{ID: "clip:2", Title: "build log", Text: "build\nlog\nthird line", Preview: "build …"},
			},
		},
	}

	vm := NewRenderVMBuilder().Build(root)
	if vm.Shell.Overlay.Kind != OverlayClipboardHistory || vm.Shell.Overlay.Content.Kind != ContentClipboardHistory {
		t.Fatalf("expected clipboard history overlay VM, got %#v", vm.Shell.Overlay)
	}
	if !containsFooterAction(vm.Shell.Footer.ActionTokens, "enter", "PASTE", "clipboard_history.paste") {
		t.Fatalf("clipboard history footer should expose paste action, got %#v", vm.Shell.Footer)
	}
	if !containsFooterAction(vm.Shell.Footer.ActionTokens, "^N", "NEW", "clipboard_history.new") {
		t.Fatalf("clipboard history footer should expose new action, got %#v", vm.Shell.Footer)
	}
	content := vm.Shell.Overlay.Content
	if content.Status != "clipboard history: 2" || len(content.HitRegions) != 3 ||
		content.Meta.ClipboardNameWidth != state.DefaultClipboardHistoryNameWidth ||
		!strings.Contains(content.Lines[0].PlainString(), "Search:") ||
		!strings.Contains(content.Lines[2].PlainString(), "› alpha") ||
		!strings.Contains(content.Lines[2].PlainString(), "│alpha") ||
		strings.Contains(plainLines(content.Lines), "[new]") ||
		strings.Contains(plainLines(content.Lines), "copied entries") {
		t.Fatalf("expected clipboard history content, got %#v", content)
	}
	if content.HitRegions[0].Rect.Y != 2 || content.HitRegions[0].ActionID != ActionClipboardHistorySelect.String() {
		t.Fatalf("clipboard rows should expose select hit regions only, got %#v", content.HitRegions)
	}
	if content.HitRegions[2].ActionID != ActionClipboardHistoryDividerDrag.String() || content.HitRegions[2].Rect.X != state.DefaultClipboardHistoryNameWidth {
		t.Fatalf("clipboard divider should expose resize drag hit region, got %#v", content.HitRegions)
	}
	shell = shell.SetClipboardHistorySelectedIndex(1, 2)
	root.Shell = shell
	content = NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if len(content.Lines) < 5 ||
		!strings.Contains(content.Lines[2].PlainString(), "│build") ||
		!strings.Contains(content.Lines[3].PlainString(), "│log") ||
		!strings.Contains(content.Lines[4].PlainString(), "│third line") {
		t.Fatalf("clipboard history should preview selected entry body across lines, got %#v", content.Lines)
	}
	if !styledLinesContainText(content.Lines[2:5], "build", StyleForeground) ||
		!styledLinesContainText(content.Lines[2:5], "log", StyleForeground) ||
		styledLinesContainText(content.Lines[2:5], "build", StylePickerMuted) {
		t.Fatalf("clipboard preview body should render as foreground text, got %#v", content.Lines[2:5])
	}
}

func TestRenderVMBuilderProjectsClipboardHistoryFuzzyMatches(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell().OpenClipboardHistory().SetClipboardHistoryQuery("gft"),
		Clipboard: state.ClipboardStore{
			Entries: []state.ClipboardEntry{
				{ID: "clip:1", Title: "git commit", Text: "git commit -m fix terminal", Preview: "git commit -m fix terminal"},
				{ID: "clip:2", Title: "status check", Text: "status check --watch", Preview: "status check --watch"},
			},
		},
	}

	items := state.ClipboardHistoryItems(root)
	if len(items) != 1 || items[0].Title != "git commit" || len(items[0].PreviewMatchIndexes) != 3 {
		t.Fatalf("expected gft to fuzzy-match git commit preview, got %#v", items)
	}
	content := NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if len(content.Lines) < 3 || !styledLinesContainText(content.Lines[2:3], "g", StylePickerMatch) ||
		!styledLinesContainText(content.Lines[2:3], "f", StylePickerMatch) ||
		!styledLinesContainText(content.Lines[2:3], "t", StylePickerMatch) {
		t.Fatalf("expected clipboard fuzzy match letters highlighted, got %#v", content.Lines)
	}
}

func TestRenderVMBuilderProjectsFloatingFooterGroupActions(t *testing.T) {
	shell := state.DefaultShell()
	shell, _ = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
	})
	shell = shell.SetInteractionMode(state.InteractionModeFloating)
	vm := NewRenderVMBuilder().Build(state.Root{Shell: shell})
	if !containsFooterAction(vm.Shell.Footer.ActionTokens, "v", "ALL", "floating.toggle_all") {
		t.Fatalf("floating footer should expose toggle-all action, got %#v", vm.Shell.Footer)
	}
	if !containsFooterAction(vm.Shell.Footer.ActionTokens, "=", "FIT", "floating.fit") {
		t.Fatalf("floating footer should expose fit action, got %#v", vm.Shell.Footer)
	}
	if !containsFooterAction(vm.Shell.Footer.ActionTokens, "s", "AUTO-FIT", "floating.auto_fit") {
		t.Fatalf("floating footer should expose auto-fit action, got %#v", vm.Shell.Footer)
	}
}

func TestRenderVMBuilderProjectsZoomedPaneOnly(t *testing.T) {
	shell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive}, state.SplitDirectionHorizontal).
		ZoomPane(state.PaneCommandTarget{PaneID: "pane-2"})

	vm := NewRenderVMBuilder().Build(state.Root{Shell: shell, Surface: state.TerminalSurfaceStore{TerminalID: "term-live", Lines: []string{"prompt"}}})
	if len(vm.Shell.Layout.Panels) != 1 {
		t.Fatalf("expected one zoomed panel, got %#v", vm.Shell.Layout.Panels)
	}
	if panel := vm.Shell.Layout.Panels[0]; panel.ID != "pane-2" || !panel.Active || panel.Content.Kind != ContentTerminalLive {
		t.Fatalf("unexpected zoomed panel projection %#v", panel)
	}
	if vm.Shell.Layout.Split.PaneID != "pane-2" || len(vm.Shell.Layout.Split.Children) != 0 {
		t.Fatalf("expected zoom split root to target pane, got %#v", vm.Shell.Layout.Split)
	}
}

func TestRenderVMBuilderDoesNotProjectClosedPane(t *testing.T) {
	shell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive}, state.SplitDirectionHorizontal).
		ClosePane(state.PaneCommandTarget{PaneID: "pane-2"})

	vm := NewRenderVMBuilder().Build(state.Root{Shell: shell})
	if len(vm.Shell.Layout.Panels) != 1 || vm.Shell.Layout.Panels[0].ID != state.DefaultPaneID {
		t.Fatalf("expected closed pane removed from render VM, got %#v", vm.Shell.Layout.Panels)
	}
	if vm.Shell.Layout.Split.PaneID != state.DefaultPaneID {
		t.Fatalf("expected split tree pruned to default pane, got %#v", vm.Shell.Layout.Split)
	}
}

func TestRenderVMBuilderProjectsPaneGeometryHints(t *testing.T) {
	shell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive}, state.SplitDirectionVertical).
		SetPaneSize(state.PaneCommand{Target: state.PaneCommandTarget{PaneID: "pane-2"}, SizeMode: state.PaneSizeCells, Cols: 12})

	vm := NewRenderVMBuilder().Build(state.Root{Shell: shell})
	if vm.Shell.Layout.Split.FixedPaneID != "pane-2" || vm.Shell.Layout.Split.FixedCols != 12 {
		t.Fatalf("expected fixed pane size hint in render VM, got %#v", vm.Shell.Layout.Split)
	}
}

func TestRenderVMBuilderUsesRootViewportAsLayoutTruth(t *testing.T) {
	root := state.Root{
		Viewport: state.ViewportStore{Valid: true, Cols: 37, Rows: 13},
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-live",
			Cols:       80,
			Rows:       24,
			Lines:      []string{"prompt"},
		},
		Session: state.TerminalSessionStore{
			TerminalID: "term-live",
			Attached:   true,
			Cols:       120,
			Rows:       40,
		},
	}

	vm := NewRenderVMBuilder().Build(root)
	if got, want := vm.Shell.Layout.Viewport, (Rect{W: 37, H: 13}); got != want {
		t.Fatalf("expected external viewport as layout truth got=%#v want=%#v", got, want)
	}
}

func TestRenderVMBuilderUsesLiveSurface(t *testing.T) {
	root := state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-live",
			Cols:       80,
			Rows:       24,
			Lines:      []string{"prompt", "output"},
		},
		Session: state.TerminalSessionStore{
			TerminalID: "term-live",
			Attached:   true,
			Cols:       80,
			Rows:       24,
			State:      state.TerminalLiveAttached,
		},
	}
	root = bindTestPaneTerminal(root, state.DefaultPaneID, "term-live")

	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	if content.Kind != ContentTerminalLive || len(content.Lines) != 2 || content.Lines[0].PlainString() != "prompt" || content.Status != "live: term-live attached 80x24" {
		t.Fatalf("unexpected live content %#v", content)
	}
	if vm.Shell.Footer.ActiveTarget != "pane:shell attached" {
		t.Fatalf("expected attached footer active target, got %#v", vm.Shell.Footer)
	}
}

func TestRenderVMBuilderProjectsTerminalLiveANSIStyleCursorAndState(t *testing.T) {
	root := state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-live",
			Cols:       20,
			Rows:       5,
			Ready:      true,
			Screen: [][]state.LiveCell{{
				{Text: "ERR", Width: 3, FG: "ansi:1"},
				{Text: " ok ", Width: 4},
				{Text: "好", Width: 2, FG: "ansi:2"},
				{Text: "🚀", Width: 2, FG: "ansi:2"},
			}},
			Cursor: state.LiveCursor{Visible: true, Row: 0, Col: 10, Shape: "bar"},
		},
		Session: state.TerminalSessionStore{TerminalID: "term-live", Attached: true, Cols: 20, Rows: 5},
	}
	root = bindTestPaneTerminal(root, state.DefaultPaneID, "term-live")

	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	if content.Kind != ContentTerminalLive || len(content.Lines) != 1 {
		t.Fatalf("expected terminal-live content, got %#v", content)
	}
	if got := content.Lines[0].PlainString(); got != "ERR ok 好🚀" {
		t.Fatalf("expected sanitized live text, got %q", got)
	}
	if !lineHasANSICell(content.Lines[0], "ERR", ANSICellStyle{FG: "ansi:1"}) ||
		!lineHasANSICell(content.Lines[0], "好", ANSICellStyle{FG: "ansi:2"}) ||
		!lineHasANSICell(content.Lines[0], "🚀", ANSICellStyle{FG: "ansi:2"}) {
		t.Fatalf("expected live terminal SGR preserved as ANSI cell style, got %#v", content.Lines[0])
	}
	if !content.Cursor.Visible || content.Cursor.Row != 0 || content.Cursor.Col != 10 || content.Cursor.Shape != CursorShapeBar {
		t.Fatalf("expected live cursor projection, got %#v", content.Cursor)
	}
	if content.Extent != (ContentExtent{Known: true, Cols: 20, Rows: 5}) {
		t.Fatalf("live content must expose terminal surface extent, got %#v", content.Extent)
	}

	legacyRoot := bindTestPaneTerminal(state.Root{Surface: state.TerminalSurfaceStore{
		TerminalID: "term-live",
		Ready:      true,
		Lines:      []string{"\x1b[31mERR\x1b[0m"},
	}}, state.DefaultPaneID, "term-live")
	legacy := NewRenderVMBuilder().Build(legacyRoot)
	if content := activeContent(legacy.Shell); !lineHasStyledCell(content.Lines[0], "ERR", StyleDanger) {
		t.Fatalf("expected legacy ANSI line fallback to keep working, got %#v", content.Lines[0])
	}

	pending := NewRenderVMBuilder().Build(bindTestPaneTerminal(state.Root{Surface: state.TerminalSurfaceStore{TerminalID: "term-live"}}, state.DefaultPaneID, "term-live"))
	if content := activeContent(pending.Shell); !content.Pending || content.Empty || content.Lines[0].PlainString() != "live surface pending" {
		t.Fatalf("expected pending live surface content, got %#v", content)
	}

	empty := NewRenderVMBuilder().Build(bindTestPaneTerminal(state.Root{Surface: state.TerminalSurfaceStore{TerminalID: "term-live", Ready: true}}, state.DefaultPaneID, "term-live"))
	if content := activeContent(empty.Shell); !content.Empty || content.Pending || content.Lines[0].PlainString() != "live surface empty" {
		t.Fatalf("expected empty live surface content, got %#v", content)
	}
}

func TestRenderVMBuilderDoesNotSynthesizeLiveCursorFromPreservedTail(t *testing.T) {
	root := bindTestPaneTerminal(state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-live",
			Ready:      true,
			Cols:       80,
			Rows:       24,
			Lines:      []string{"old live tail"},
			Cursor:     state.LiveCursor{},
		},
		Session: state.TerminalSessionStore{TerminalID: "term-live", Attached: true, Cols: 80, Rows: 24},
	}, state.DefaultPaneID, "term-live")

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if content.Kind != ContentTerminalLive || len(content.Lines) != 1 || content.Lines[0].PlainString() != "old live tail" {
		t.Fatalf("expected preserved live tail content, got %#v", content)
	}
	if content.Cursor.Visible || content.Cursor.Anchor {
		t.Fatalf("preserved tail without surface cursor must not synthesize cursor at line end, got %#v", content.Cursor)
	}
}

func TestRenderVMBuilderPreservesHiddenLiveCursorAsAnchor(t *testing.T) {
	root := bindTestPaneTerminal(state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-live",
			Ready:      true,
			Cols:       20,
			Rows:       4,
			Screen:     [][]state.LiveCell{{{Text: "prompt", Width: 6}}},
			Cursor:     state.LiveCursor{Visible: false, Row: 2, Col: 7, Shape: "bar"},
		},
		Session: state.TerminalSessionStore{TerminalID: "term-live", Attached: true, Cols: 20, Rows: 4},
	}, state.DefaultPaneID, "term-live")

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if content.Cursor != (Cursor{Anchor: true, Row: 2, Col: 7, Shape: CursorShapeBar}) {
		t.Fatalf("hidden live cursor must retain its protocol position as a host anchor, got %#v", content.Cursor)
	}
}

func TestRenderVMBuilderMapsLiveCursorFromSurfaceCoordinates(t *testing.T) {
	root := bindTestPaneTerminal(state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-live",
			Ready:      true,
			Cols:       20,
			Rows:       4,
			Screen: [][]state.LiveCell{
				{{Text: "old live tail", Width: 13}},
				{{Text: "$ ", Width: 2}},
			},
			Cursor: state.LiveCursor{Visible: true, Row: 1, Col: 2},
		},
		Session: state.TerminalSessionStore{TerminalID: "term-live", Attached: true, Cols: 20, Rows: 4},
	}, state.DefaultPaneID, "term-live")

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if content.Kind != ContentTerminalLive || len(content.Lines) != 2 {
		t.Fatalf("expected live content from surface screen, got %#v", content)
	}
	if content.Cursor != (Cursor{Visible: true, Row: 1, Col: 2, Shape: CursorShapeBlock}) {
		t.Fatalf("live cursor must use core surface coordinates, got %#v", content.Cursor)
	}
}

func TestRenderVMBuilderDoesNotApplyLiveExtentToStatusFallbacks(t *testing.T) {
	cases := []struct {
		name    string
		root    state.Root
		want    string
		pending bool
		empty   bool
	}{
		{
			name: "pending",
			root: state.Root{
				Surface: state.TerminalSurfaceStore{TerminalID: "term-live", Cols: 6, Rows: 2},
				Session: state.TerminalSessionStore{TerminalID: "term-live", Attached: true, Cols: 6, Rows: 2},
			},
			want:    "live surface pending",
			pending: true,
		},
		{
			name: "empty",
			root: state.Root{
				Surface: state.TerminalSurfaceStore{TerminalID: "term-live", Cols: 6, Rows: 2, Ready: true},
				Session: state.TerminalSessionStore{TerminalID: "term-live", Attached: true, Cols: 6, Rows: 2},
			},
			want:  "live surface empty",
			empty: true,
		},
		{
			name: "exited",
			root: state.Root{
				Surface: state.TerminalSurfaceStore{TerminalID: "term-live", Cols: 6, Rows: 2, State: state.TerminalLiveExited, ExitCode: 0},
				Session: state.TerminalSessionStore{TerminalID: "term-live", Attached: true, Cols: 6, Rows: 2, State: state.TerminalLiveExited, ExitCode: 0},
			},
			want:  "terminal exited: term-live code:0",
			empty: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := bindTestPaneTerminal(tc.root, state.DefaultPaneID, "term-live")
			content := activeContent(NewRenderVMBuilder().Build(root).Shell)
			if content.Pending != tc.pending || content.Empty != tc.empty || content.Lines[0].PlainString() != tc.want {
				t.Fatalf("unexpected fallback content %#v", content)
			}
			if content.Extent.Known {
				t.Fatalf("status fallback must not enable live extent dots, got %#v", content.Extent)
			}
		})
	}
}

func TestRenderVMBuilderDoesNotApplyResizeBoundaryExtentToStatusFallbacks(t *testing.T) {
	cases := []struct {
		name string
		root state.Root
		want string
	}{
		{
			name: "pending after resize",
			root: state.Root{
				Surface: state.TerminalSurfaceStore{
					TerminalID:     "term-live",
					Cols:           98,
					Rows:           36,
					ResizeBoundary: state.LiveResizeBoundary{Active: true, PreviousCols: 78, PreviousRows: 20, Cols: 98, Rows: 36},
				},
				Session: state.TerminalSessionStore{TerminalID: "term-live", Attached: true, Cols: 98, Rows: 36},
			},
			want: "live surface pending",
		},
		{
			name: "exited after resize",
			root: state.Root{
				Surface: state.TerminalSurfaceStore{
					TerminalID: "term-live",
					Cols:       98,
					Rows:       36,
					State:      state.TerminalLiveExited,
					ExitCode:   0,
				},
				Session: state.TerminalSessionStore{TerminalID: "term-live", Attached: true, Cols: 98, Rows: 36, State: state.TerminalLiveExited},
			},
			want: "terminal exited: term-live code:0",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := bindTestPaneTerminal(tc.root, state.DefaultPaneID, "term-live")
			content := activeContent(NewRenderVMBuilder().Build(root).Shell)
			if content.Extent.Known {
				t.Fatalf("resize fallback must not enable live extent dots, got %#v", content.Extent)
			}
			if got := content.Lines[0].PlainString(); got != tc.want {
				t.Fatalf("unexpected fallback content got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestRenderVMBuilderUsesSessionSizeAsLiveExtentWhenSurfaceSizeMissing(t *testing.T) {
	root := state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-live",
			Ready:      true,
			Lines:      []string{"prompt"},
		},
		Session: state.TerminalSessionStore{
			TerminalID: "term-live",
			Attached:   true,
			Cols:       14,
			Rows:       3,
		},
	}

	root = bindTestPaneTerminal(root, state.DefaultPaneID, "term-live")
	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if content.Extent != (ContentExtent{Known: true, Cols: 14, Rows: 3}) {
		t.Fatalf("live content should fall back to session size for extent, got %#v", content.Extent)
	}
}

func TestRenderVMBuilderProjectsLiveExitLifecycle(t *testing.T) {
	exitedAt := time.Date(2026, 6, 17, 12, 30, 0, 0, time.UTC)
	root := state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-live",
			State:      state.TerminalLiveExited,
			ExitCode:   0,
			ExitReason: "done",
			ExitedAt:   exitedAt,
			Command:    []string{"bash", "-lc", "make test"},
		},
		Session: state.TerminalSessionStore{
			TerminalID: "term-live",
			State:      state.TerminalLiveExited,
			ExitCode:   0,
			ExitReason: "done",
			ExitedAt:   exitedAt,
			Command:    []string{"bash", "-lc", "make test"},
		},
	}

	root = bindTestPaneTerminal(root, state.DefaultPaneID, "term-live")
	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	if content.Kind != ContentExitedPane || !content.Empty || content.Pending || content.Cursor.Visible {
		t.Fatalf("expected exited live content lifecycle, got %#v", content)
	}
	if content.Lines[0].PlainString() != "terminal exited: term-live code:0 done" || content.Status != "exited: term-live code:0 done" {
		t.Fatalf("unexpected exited live projection content=%#v status=%q", content.Lines, content.Status)
	}
	if !strings.Contains(plainLines(content.Lines), "exited at: 2026-06-17T12:30:00Z") || !strings.Contains(plainLines(content.Lines), "command: bash -lc make test") {
		t.Fatalf("expected exited lifecycle metadata, got %#v", content.Lines)
	}
	if vm.Shell.Footer.ActiveTarget != "pane:shell exited" {
		t.Fatalf("expected exited footer active target, got %#v", vm.Shell.Footer)
	}
}

func TestRenderVMBuilderProjectsSessionOnlyExitedLifecycle(t *testing.T) {
	exitedAt := time.Date(2026, 7, 5, 1, 1, 34, 0, time.UTC)
	root := state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-session-exited",
			Ready:      true,
			State:      state.TerminalLiveAttached,
			Lines:      []string{"terminal exited: term-session-exited code:130 exited"},
			Cols:       80,
			Rows:       24,
		},
		Session: state.TerminalSessionStore{}.
			AttachWithResizeOwner("term-session-exited", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID)).
			MarkExitedWithMetadata("term-session-exited", 130, "exited", exitedAt, []string{"/bin/zsh"}),
	}

	root = bindTestPaneTerminal(root, state.DefaultPaneID, "term-session-exited")
	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if content.Kind != ContentExitedPane || content.Status != "exited: term-session-exited code:130 exited" {
		t.Fatalf("session-only exited lifecycle should render exited pane, got %#v", content)
	}
	if plain := plainLines(content.Lines); strings.Count(plain, "terminal exited: term-session-exited") != 1 || !strings.Contains(plain, "► restart ◄") {
		t.Fatalf("session-only exited lifecycle should keep core marker once and append restart CTA, got %#v", content.Lines)
	}
	if !contentHasAction(content, ActionExitedRestart.String()) || !contentHasAction(content, ActionExitedReconnect.String()) {
		t.Fatalf("session-only exited lifecycle should expose exited CTA hit regions, got %#v", content.HitRegions)
	}
}

func TestRenderVMBuilderAppendsLiveExitLifecycleAfterFullOutput(t *testing.T) {
	exitedAt := time.Date(2026, 6, 17, 12, 30, 0, 0, time.UTC)
	lines := make([]string, 24)
	for index := range lines {
		lines[index] = "screen row " + string(rune('A'+index))
	}
	root := state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-live",
			Ready:      true,
			Lines:      lines,
			Cols:       80,
			Rows:       24,
			State:      state.TerminalLiveExited,
			ExitCode:   23,
			ExitReason: "done",
			ExitedAt:   exitedAt,
			Command:    []string{"bash", "-lc", "exit 23"},
		},
		Session: state.TerminalSessionStore{
			TerminalID: "term-live",
			Attached:   true,
			Cols:       80,
			Rows:       24,
			State:      state.TerminalLiveExited,
			ExitCode:   23,
			ExitReason: "done",
			ExitedAt:   exitedAt,
			Command:    []string{"bash", "-lc", "exit 23"},
		},
	}

	root = bindTestPaneTerminal(root, state.DefaultPaneID, "term-live")
	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if content.Kind != ContentExitedPane {
		t.Fatalf("live exited content should use exited pane CTA kind, got %#v", content.Kind)
	}
	if got := content.Lines[0].PlainString(); got != "screen row A" {
		t.Fatalf("exited content must keep previous live output at the front, got %q", got)
	}
	if got := content.Lines[23].PlainString(); got != "screen row X" {
		t.Fatalf("exited content must append lifecycle after the last live row, got %q", got)
	}
	if got := content.Lines[24].PlainString(); got != "" {
		t.Fatalf("exited lifecycle should be separated from the last live row, got %q", got)
	}
	rendered := RenderContentViewport(ContentRenderRequest{
		Rect:    Rect{W: 80, H: 8},
		Content: content,
	})
	got := plainContentViewportLines(rendered.Lines)
	want := []string{
		"screen row W",
		"screen row X",
		"",
		"terminal exited: term-live code:23 done",
		"exited at: 2026-06-17T12:30:00Z",
		"command: bash -lc exit 23",
		"► restart ◄",
		"[ reconnect ]",
	}
	for index, wantLine := range want {
		if strings.TrimSpace(got[index]) != wantLine {
			t.Fatalf("exit lifecycle should follow the visible live tail row=%d got=%#v want=%#v all=%#v", index, got[index], wantLine, got)
		}
	}
	if !strings.HasPrefix(got[3], "terminal exited:") || !strings.Contains(got[6], "► restart ◄") {
		t.Fatalf("exit lifecycle text should remain left-aligned and only actions should center, got %#v", got)
	}
	restart := hitRegionByAction(t, rendered.HitRegions, ActionExitedRestart.String())
	restartWidth := DisplayWidth("► restart ◄")
	if restart.Rect != (Rect{X: (80 - restartWidth) / 2, Y: 6, W: restartWidth, H: 1}) {
		t.Fatalf("restart hit region should follow bottom-aligned exited action, got %#v", restart)
	}
	picker := hitRegionByAction(t, rendered.HitRegions, ActionExitedReconnect.String())
	pickerWidth := DisplayWidth("[ reconnect ]")
	if picker.Rect != (Rect{X: (80 - pickerWidth) / 2, Y: 7, W: pickerWidth, H: 1}) {
		t.Fatalf("picker hit region should follow bottom-aligned exited action, got %#v", picker)
	}
	if !contentHasAction(content, ActionExitedRestart.String()) || !contentHasAction(content, ActionExitedReconnect.String()) {
		t.Fatalf("expected exited live content actions, got %#v", content.HitRegions)
	}
	if !strings.Contains(plainLines(content.Lines), "screen row A") || !strings.Contains(plainLines(content.Lines), "screen row X") {
		t.Fatalf("expected exited content to preserve previous live output, got %#v", content.Lines)
	}
}

func TestRenderVMBuilderDoesNotDuplicateCoreExitMarker(t *testing.T) {
	exitedAt := time.Date(2026, 6, 17, 12, 30, 0, 0, time.UTC)
	root := state.Root{
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-live",
			Ready:      true,
			Lines: []string{
				"last output",
				"terminal exited: term-live code:0 exited",
				"exited at: 2026-06-17T12:30:00Z",
				"command: /bin/zsh",
			},
			Cols:       80,
			Rows:       10,
			State:      state.TerminalLiveExited,
			ExitCode:   0,
			ExitReason: "exited",
			ExitedAt:   exitedAt,
			Command:    []string{"/bin/zsh"},
		},
		Session: state.TerminalSessionStore{
			TerminalID: "term-live",
			Attached:   true,
			Cols:       80,
			Rows:       10,
			State:      state.TerminalLiveExited,
			ExitCode:   0,
			ExitReason: "exited",
			ExitedAt:   exitedAt,
			Command:    []string{"/bin/zsh"},
		},
	}

	root = bindTestPaneTerminal(root, state.DefaultPaneID, "term-live")
	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	plain := plainLines(content.Lines)
	if count := strings.Count(plain, "terminal exited: term-live"); count != 1 {
		t.Fatalf("core exit marker should not be duplicated by render overlay, count=%d content=%#v", count, content.Lines)
	}
	if strings.Count(plain, "exited at: 2026-06-17T12:30:00Z") != 1 || strings.Count(plain, "command: /bin/zsh") != 1 {
		t.Fatalf("core exit marker metadata should stay single-source, got %#v", content.Lines)
	}
	if !contentHasAction(content, ActionExitedRestart.String()) || !contentHasAction(content, ActionExitedReconnect.String()) {
		t.Fatalf("render should still append exited actions after core marker, got %#v", content.HitRegions)
	}
}

func TestRenderVMBuilderRespectsActiveEmptyAndExitedPaneContent(t *testing.T) {
	emptyShell := state.DefaultShell()
	emptyShell.Workspace.Tabs[0].Panes[0] = state.PaneState{ID: state.DefaultPaneID, Title: "slot", Kind: state.PaneEmpty, Active: true}
	emptyVM := NewRenderVMBuilder().Build(state.Root{
		Shell:   emptyShell,
		Surface: state.TerminalSurfaceStore{TerminalID: "term-live", Ready: true, Lines: []string{"live must not replace empty"}},
	})
	if content := activeContent(emptyVM.Shell); content.Kind != ContentEmptyPane || !content.Empty || content.Lines[0].PlainString() != "○ No terminal connected" || content.Status != "not connected: choose or create terminal" || !contentPlainContains(content, "► Attach existing terminal ◄") || !contentPlainContains(content, "Choose a terminal or create one.") || content.Cursor.Visible || content.Cursor.Anchor {
		t.Fatalf("expected active empty pane placeholder, got %#v", content)
	} else if !contentHasAction(content, "empty.attach") || !contentHasAction(content, "empty.create") || !contentHasAction(content, "empty.manager") || !contentHasAction(content, "empty.close") {
		t.Fatalf("expected empty pane CTA action regions, got %#v", content.HitRegions)
	}
	emptyShell.EmptyPaneCTA.SelectedIndex = 2
	emptyVM = NewRenderVMBuilder().Build(state.Root{Shell: emptyShell})
	if content := activeContent(emptyVM.Shell); !contentPlainContains(content, "► Open terminal manager ◄") || !contentPlainContains(content, "[ Attach existing terminal ]") {
		t.Fatalf("expected reducer-owned empty pane CTA selection, got %#v", content.Lines)
	}
	floatingShell := state.DefaultShell()
	floatingShell, _ = floatingShell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float", Kind: state.PaneEmpty},
	})
	floatingShell.EmptyPaneCTA.SelectedIndex = 1
	floatingVM := NewRenderVMBuilder().Build(state.Root{Shell: floatingShell})
	if len(floatingVM.Shell.Layout.Floating) != 1 {
		t.Fatalf("expected active floating VM, got %#v", floatingVM.Shell.Layout.Floating)
	}
	if content := floatingVM.Shell.Layout.Floating[0].Content; content.Kind != ContentEmptyPane || !contentPlainContains(content, "► Create new terminal ◄") || !contentPlainContains(content, "[ Attach existing terminal ]") {
		t.Fatalf("expected reducer-owned floating empty CTA selection, got %#v", content.Lines)
	}

	exitedAt := time.Date(2026, 6, 17, 12, 31, 0, 0, time.UTC)
	exitedRoot := bindTestPaneTerminal(state.Root{Shell: state.DefaultShell()}, state.DefaultPaneID, "term-old")
	exitedRoot.Surface = state.TerminalSurfaceStore{
		TerminalID: "term-old",
		Ready:      true,
		State:      state.TerminalLiveExited,
		ExitCode:   23,
		ExitReason: "exited",
		ExitedAt:   exitedAt,
		Command:    []string{"bash", "-lc", "exit 23"},
	}
	exitedRoot.Session = state.TerminalSessionStore{
		TerminalID: "term-old",
		State:      state.TerminalLiveExited,
		ExitCode:   23,
		ExitReason: "exited",
		ExitedAt:   exitedAt,
		Command:    []string{"bash", "-lc", "exit 23"},
	}
	exitedVM := NewRenderVMBuilder().Build(exitedRoot)
	if content := activeContent(exitedVM.Shell); content.Kind != ContentExitedPane || content.Status != "exited: term-old code:23 exited" || !strings.Contains(content.Lines[0].PlainString(), "terminal exited: term-old code:23 exited") {
		t.Fatalf("expected terminal lifecycle exited content, got %#v", content)
	} else if !strings.Contains(plainLines(content.Lines), "exited at: 2026-06-17T12:31:00Z") || !strings.Contains(plainLines(content.Lines), "command: bash -lc exit 23") {
		t.Fatalf("expected exited lifecycle metadata, got %#v", content.Lines)
	} else if !contentHasAction(content, "exited.restart") || !contentHasAction(content, "exited.reconnect") {
		t.Fatalf("expected exited lifecycle CTA action regions, got %#v", content.HitRegions)
	}
}

func TestRenderVMBuilderProjectsCreatedTabAsUnconnectedPane(t *testing.T) {
	shell, result := state.DefaultShell().ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: "logs"})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("create tab: %#v", result)
	}

	vm := NewRenderVMBuilder().Build(state.Root{Shell: shell})
	if len(vm.Shell.Layout.Panels) != 1 {
		t.Fatalf("created tab should project its unconnected pane, got %#v", vm.Shell.Layout.Panels)
	}
	panel := vm.Shell.Layout.Panels[0]
	content := panel.Content
	if panel.ID == "" || panel.ID != shell.ActivePaneID || content.Kind != ContentEmptyPane || !content.Empty || !contentPlainContains(content, "No terminal connected") || !contentPlainContains(content, "Attach existing terminal") {
		t.Fatalf("expected created tab unconnected pane, panel=%#v shell=%#v", panel, shell)
	}
	if content.Cursor.Visible || content.Cursor.Anchor {
		t.Fatalf("unconnected pane must not expose a content cursor, got %#v", content.Cursor)
	}
	if !contentHasAction(content, "empty.attach") || !contentHasAction(content, "empty.create") || !contentHasAction(content, "empty.manager") {
		t.Fatalf("expected unconnected pane CTA action regions, got %#v", content.HitRegions)
	}
}

func TestRenderVMBuilderProjectsEmptyWorkspaceWithoutSyntheticTab(t *testing.T) {
	shell, result := state.DefaultShell().ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabClose})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("close last tab: %#v", result)
	}

	vm := NewRenderVMBuilder().Build(state.Root{Shell: shell})
	content := vm.Shell.Layout.BodyContent
	if len(vm.Shell.Header.Tabs) != 0 || vm.Shell.Header.Tab != "" {
		t.Fatalf("empty workspace must not synthesize header tabs, got %#v", vm.Shell.Header)
	}
	if len(vm.Shell.Layout.Panels) != 0 {
		t.Fatalf("empty workspace must not create synthetic panel VMs, got %#v", vm.Shell.Layout.Panels)
	}
	plain := plainLines(content.Lines)
	if content.Kind != ContentEmptyPane ||
		!content.Empty ||
		!strings.Contains(plain, "No tabs in this workspace") ||
		!strings.Contains(plain, "Choose terminal") ||
		!strings.Contains(plain, "New tab") {
		t.Fatalf("expected empty workspace content, got %#v", content)
	}
	if content.Cursor.Visible || len(content.HitRegions) != 2 {
		t.Fatalf("empty workspace must expose canonical CTA hit regions, got cursor=%#v hitRegions=%#v", content.Cursor, content.HitRegions)
	}
}

func TestRenderVMBuilderProjectsTerminalPickerContentRenderer(t *testing.T) {
	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-main"
	shell = shell.SplitActivePane(state.PaneState{ID: "pane-2", Title: "日志🚀", Kind: state.PaneTerminalLive, TerminalID: "term-2"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}).
		OpenTerminalPicker()
	shell.Overlay.Query = "term"

	root := state.Root{Shell: shell, TerminalPool: state.TerminalPoolStore{Status: state.TerminalPoolReady, Items: []state.TerminalPoolItem{{TerminalID: "term-main", Title: "shell", State: "running", Cols: 80, Rows: 24}, {TerminalID: "term-2", Title: "日志🚀", State: "running", Cols: 100, Rows: 30}}}}
	vm := NewRenderVMBuilder().Build(root)
	content := vm.Shell.Overlay.Content
	if vm.Shell.Overlay.Kind != OverlayTerminalPicker || content.Kind != ContentTerminalPicker {
		t.Fatalf("expected terminal picker content, got %#v", vm.Shell.Overlay)
	}
	plain := plainLines(content.Lines)
	for _, want := range []string{"search: term", "▸ + new terminal", "● shell", "running", "80x24", "● 日志🚀", "100x30", "Create terminal"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("expected compact picker marker %q, got %#v", want, content.Lines)
		}
	}
	for _, forbidden := range []string{"Select terminal source state target", "DETAIL", "TARGET", "TERMINAL", "[attach]", "[new]", "PREVIEW pane:", "pane:", "selected "} {
		if strings.Contains(plain, forbidden) {
			t.Fatalf("picker must not render management table/detail content %q, got %#v", forbidden, content.Lines)
		}
	}
	if content.Status != "terminal picker: 2 items query:term" {
		t.Fatalf("expected picker item count status, got %q", content.Status)
	}
	if !content.Cursor.Visible || content.Cursor.Shape != CursorShapeBar {
		t.Fatalf("expected picker search cursor, got %#v", content.Cursor)
	}
	if !contentHasAction(content, "picker.attach") || !contentHasAction(content, "picker.new") {
		t.Fatalf("expected terminal attach and create hit regions, got %#v", content.HitRegions)
	}
}

func TestRenderVMBuilderFiltersTerminalPickerAndHighlightsSelectedRow(t *testing.T) {
	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-main"
	shell = shell.SplitActivePane(state.PaneState{ID: "pane-2", Title: "日志🚀", Kind: state.PaneTerminalLive, TerminalID: "term-2"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}).
		OpenTerminalPicker().
		SetTerminalPickerQuery("日志")

	root := state.Root{Shell: shell, TerminalPool: state.TerminalPoolStore{Status: state.TerminalPoolReady, Items: []state.TerminalPoolItem{{TerminalID: "term-main", Title: "shell", State: "running", Cols: 80, Rows: 24}, {TerminalID: "term-2", Title: "日志🚀", State: "running", Cols: 100, Rows: 30}}}}
	content := NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if !strings.Contains(content.Lines[0].PlainString(), "search: 日志") ||
		!strings.Contains(content.Lines[1].PlainString(), "▸ ● 日志🚀") ||
		!strings.Contains(content.Lines[1].PlainString(), "running") ||
		!strings.Contains(content.Lines[1].PlainString(), "100x30") ||
		strings.Contains(plainLines(content.Lines), "shell") ||
		strings.Contains(plainLines(content.Lines), "DETAIL") {
		t.Fatalf("expected filtered selected picker row, got %#v", content.Lines)
	}
	if !lineHasStyledCell(content.Lines[1], "日", StylePickerMatch) || !lineHasStyledCell(content.Lines[1], "志", StylePickerMatch) || !lineHasStyledCell(content.Lines[1], "running", StyleSuccess) {
		t.Fatalf("expected picker row text to use picker style and match highlight, got %#v", content.Lines)
	}
	if content.Cursor.Col != DisplayWidth("search: 日志") {
		t.Fatalf("expected cursor after query text, got %#v", content.Cursor)
	}
	if content.Status != "terminal picker: 1 items query:日志" {
		t.Fatalf("expected query status, got %q", content.Status)
	}
}

func TestRenderVMBuilderProjectsTerminalPickerPoolStateAndRows(t *testing.T) {
	shell := state.DefaultShell().OpenTerminalPicker()
	root := state.Root{
		Shell: shell,
		TerminalPool: state.TerminalPoolStore{
			Status: state.TerminalPoolReady,
			Items: []state.TerminalPoolItem{{
				TerminalID: "term-pool",
				Title:      "远程🚀",
				State:      "running",
				Cols:       120,
				Rows:       40,
			}},
		},
	}

	content := NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if !strings.Contains(content.Lines[1].PlainString(), "▸ + new terminal") ||
		!strings.Contains(content.Lines[2].PlainString(), "● 远程🚀") ||
		!strings.Contains(content.Lines[2].PlainString(), "running") ||
		!strings.Contains(content.Lines[2].PlainString(), "120x40") ||
		strings.Contains(plainLines(content.Lines), "DETAIL") {
		t.Fatalf("expected terminal-only pool row, got %#v", content.Lines)
	}
	if len(content.HitRegions) != 2 || content.HitRegions[0].ActionID != ActionPickerNew.String() || content.HitRegions[1].PaneID != "" || content.HitRegions[1].ActionID != ActionPickerAttach.String() {
		t.Fatalf("expected create and terminal attach action regions, got %#v", content.HitRegions)
	}

	root.Shell = root.Shell.SetTerminalPickerQuery("trpl")
	content = NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if len(content.HitRegions) != 1 || content.HitRegions[0].ActionID != ActionPickerAttach.String() ||
		strings.Contains(plainLines(content.Lines), "term-pool") ||
		strings.Contains(plainLines(content.Lines), "DETAIL") {
		t.Fatalf("expected fuzzy terminal id match without id column highlight, got %#v", content.Lines)
	}

	root.TerminalPool = state.TerminalPoolStore{Status: state.TerminalPoolLoading}
	content = NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if !strings.Contains(plainLines(content.Lines), "loading terminals") {
		t.Fatalf("expected loading row, got %#v", content.Lines)
	}
	root.TerminalPool = state.TerminalPoolStore{Status: state.TerminalPoolReady}
	content = NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if terminalPickerSelectableCount(state.TerminalPickerItems(root)) != 0 || strings.Contains(plainLines(content.Lines), "new terminal") {
		t.Fatalf("expected empty row, got %#v", content.Lines)
	}
	root.TerminalPool = state.TerminalPoolStore{Status: state.TerminalPoolError, LastError: "boom"}
	content = NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if !strings.Contains(plainLines(content.Lines), "pool error boom") {
		t.Fatalf("expected error row, got %#v", content.Lines)
	}
}

func TestRenderVMBuilderTerminalPickerCreateRowShowsDefaultEndpointOnly(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell().OpenTerminalPicker(),
		Endpoints: (state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "This Mac", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true}).
			Upsert(state.EndpointItem{ID: "us-west", Label: "US West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true}),
	}
	root.Shell.TerminalCreateDraft = state.TerminalCreateDraft{EndpointID: "us-west"}

	content := NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if len(content.Lines) < 2 || !strings.Contains(content.Lines[1].PlainString(), "US West") || strings.Contains(content.Lines[1].PlainString(), "This Mac") {
		t.Fatalf("create row should display only selected/default endpoint label, got %#v", content.Lines)
	}
}

func TestRenderVMBuilderProjectsTerminalPickerEndpointLabels(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell().OpenTerminalPicker(),
		Endpoints: (state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "This Mac", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true, Status: state.EndpointStatusConnected}).
			Upsert(state.EndpointItem{ID: "manual", Label: "Manual Box", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectManual, Enabled: true}).
			Upsert(state.EndpointItem{ID: "west", Label: "US West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true, Status: state.EndpointStatusOffline, LastError: "ssh timeout"}),
		TerminalPool: state.TerminalPoolStore{
			Status: state.TerminalPoolReady,
			Items: []state.TerminalPoolItem{
				{EndpointID: state.DefaultEndpointID, TerminalID: "term-1", Title: "local shell", State: "running"},
				{EndpointID: "west", TerminalID: "term-1", Title: "west shell", State: "running"},
			},
		},
	}

	content := NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	plain := plainLines(content.Lines)
	for _, want := range []string{"local shell", "This Mac", "west shell", "US West", "running"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("expected flat endpoint label picker content %q in:\n%s", want, plain)
		}
	}
	if !lineHasStyledCell(content.Lines[2], "This Mac", StylePickerInfo) ||
		!lineHasStyledCell(content.Lines[3], "US West", StylePickerInfo) {
		t.Fatalf("terminal picker endpoint ownership should use readable info style, got %#v", content.Lines)
	}
	for _, notWant := range []string{"Manual Box", "manual connect", "ssh timeout"} {
		if strings.Contains(plain, notWant) {
			t.Fatalf("terminal picker should stay a flat terminal table without endpoint-only rows %q in:\n%s", notWant, plain)
		}
	}
	if len(content.HitRegions) != 3 || content.HitRegions[0].Row != 0 || content.HitRegions[1].Row != 1 || content.HitRegions[2].Row != 2 {
		t.Fatalf("flat picker rows should own hit regions, got %#v", content.HitRegions)
	}

	root.Shell = root.Shell.SetTerminalPickerQuery("US West")
	content = NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	plain = plainLines(content.Lines)
	if !strings.Contains(plain, "US West") || !strings.Contains(plain, "new terminal") || !strings.Contains(plain, "west shell") || strings.Contains(plain, "local shell") || strings.Contains(plain, "ssh timeout") || len(content.HitRegions) != 2 || content.HitRegions[0].Row != 0 || content.HitRegions[1].Row != 1 {
		t.Fatalf("endpoint label query should show only west terminal row, lines=%#v hits=%#v", content.Lines, content.HitRegions)
	}
}

func TestRenderVMBuilderTerminalPickerHidesManagedEndpointConnectionState(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell().OpenTerminalPicker(),
		Endpoints: (state.EndpointStore{}).Upsert(state.EndpointItem{
			ID: "studio", Label: "Studio", Transport: state.EndpointTransportHubP2P,
			ConnectMode: state.EndpointConnectOnDemand, Enabled: true, Status: state.EndpointStatusOffline,
			ConnectionPhase: "failed", LastError: "relay unavailable", ObservedPath: "single_relay",
		}),
		TerminalPool: state.TerminalPoolStore{Status: state.TerminalPoolReady, Items: []state.TerminalPoolItem{{
			EndpointID: "studio", TerminalID: "term-studio", Title: "remote shell", State: "running",
		}}},
	}

	content := NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	plain := plainLines(content.Lines)
	for _, want := range []string{"remote shell", "Studio", "running"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("terminal picker should retain terminal facts %q, got:\n%s", want, plain)
		}
	}
	for _, forbidden := range []string{"offline", "on_demand", "managed-webrtc", "failed", "single_relay", "relay unavailable", "1 terminal"} {
		if strings.Contains(plain, forbidden) {
			t.Fatalf("terminal picker should hide endpoint connection metadata %q, got:\n%s", forbidden, plain)
		}
	}

	root.Shell = root.Shell.SetTerminalPickerQuery("offline")
	content = NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if len(content.Lines) != 1 || len(content.HitRegions) != 0 {
		t.Fatalf("hidden endpoint status should not be searchable, lines=%#v hits=%#v", content.Lines, content.HitRegions)
	}
}

func TestRenderVMBuilderAlignsTerminalPickerEndpointColumns(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell().OpenTerminalPicker(),
		Endpoints: (state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "This Mac", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true, Status: state.EndpointStatusConnected}).
			Upsert(state.EndpointItem{ID: "cn_fast", Label: "CN Fast", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true, Status: state.EndpointStatusConnected}).
			Upsert(state.EndpointItem{ID: "us_west", Label: "US West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true, Status: state.EndpointStatusConnected}),
		TerminalPool: state.TerminalPoolStore{
			Status: state.TerminalPoolReady,
			Items: []state.TerminalPoolItem{
				{EndpointID: state.DefaultEndpointID, TerminalID: "term-local", Title: "123", State: "running", Cols: 214, Rows: 94},
				{EndpointID: "cn_fast", TerminalID: "term-cn", Title: "cn-123", State: "running", Cols: 281, Rows: 73},
				{EndpointID: "us_west", TerminalID: "term-pool", Title: "term-pool-1783313349952893063", State: "attached", Cols: 281, Rows: 73},
			},
		},
	}

	content := NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if len(content.Lines) < 5 {
		t.Fatalf("expected search, create and terminal rows, got %#v", content.Lines)
	}
	local := content.Lines[2].PlainString()
	cn := content.Lines[3].PlainString()
	west := content.Lines[4].PlainString()
	endpointCol := displayColumnOf(local, "This Mac")
	if endpointCol < 0 || displayColumnOf(cn, "CN Fast") != endpointCol || displayColumnOf(west, "US West") != endpointCol {
		t.Fatalf("endpoint column should stay aligned:\n%s\n%s\n%s", local, cn, west)
	}
	stateCol := displayColumnOf(local, "running")
	if stateCol < 0 || displayColumnOf(cn, "running") != stateCol || displayColumnOf(west, "running") != stateCol {
		t.Fatalf("state column should stay aligned:\n%s\n%s\n%s", local, cn, west)
	}
	sizeCol := displayColumnOf(local, "214x94")
	if sizeCol < 0 || displayColumnOf(cn, "281x73") != sizeCol || displayColumnOf(west, "281x73") != sizeCol {
		t.Fatalf("size column should stay aligned:\n%s\n%s\n%s", local, cn, west)
	}
	if DisplayWidth(west) > terminalPickerHitRegionWidth {
		t.Fatalf("long terminal title should be clipped inside picker row width=%d line=%q", terminalPickerHitRegionWidth, west)
	}
	if !lineHasStyledCell(content.Lines[2], "This Mac", StylePickerInfo) || !lineHasStyledCell(content.Lines[3], "CN Fast", StylePickerInfo) || !lineHasStyledCell(content.Lines[4], "US West", StylePickerInfo) {
		t.Fatalf("endpoint ownership column should use info style, got %#v", content.Lines[2:5])
	}
}

func TestRenderVMBuilderUsesEndpointSurfaceForBoundTerminal(t *testing.T) {
	ref := state.NewTerminalRef("west", "term-live")
	root := state.Root{
		Shell: state.DefaultShell(),
		Surface: (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
			EndpointID: ref.EndpointID,
			TerminalID: ref.TerminalID,
			Revision:   2,
			Cols:       100,
			Rows:       30,
			Lines:      []string{"west live ready"},
			State:      state.TerminalLiveAttached,
		}),
		Session: (state.TerminalSessionStore{}).AttachRefWithResizeOwner(ref, 7, 100, 30, state.TerminalResizeRoleFollower, "surface-west", state.TerminalPaneViewID(state.DefaultPaneID)),
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView(ref.EndpointID, state.DefaultPaneID, ref.TerminalID, 7, 100, 30, state.TerminalResizeRoleFollower, "surface-west", state.TerminalPaneViewID(state.DefaultPaneID), false))
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, ref.TerminalID)

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if content.Pending || content.Empty || len(content.Lines) == 0 || content.Lines[0].PlainString() != "west live ready" {
		t.Fatalf("endpoint-bound pane must read west live surface instead of local pending, got %#v", content)
	}
}

func TestRenderVMBuilderProjectsDisconnectedPaneActions(t *testing.T) {
	ref := state.NewTerminalRef("west", "remote")
	errText := "remote-daemon: stdio-proxy connect core-v2 daemon socket: connection refused"
	root := state.Root{
		Shell: state.DefaultShell(),
		Surface: state.TerminalSurfaceStore{
			EndpointID: ref.EndpointID,
			TerminalID: ref.TerminalID,
			State:      state.TerminalLiveError,
			Err:        errText,
			Ready:      true,
			Lines:      []string{"last remote output"},
		},
		Session: state.TerminalSessionStore{
			EndpointID: ref.EndpointID,
			TerminalID: ref.TerminalID,
			State:      state.TerminalLiveError,
			LastError:  errText,
			Attached:   false,
		},
	}
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, ref.TerminalID)
	root.Endpoints = root.Endpoints.Upsert(state.EndpointItem{ID: ref.EndpointID, Label: "West Lab", Transport: state.EndpointTransportSSH, Enabled: true, Status: state.EndpointStatusOffline})
	binding := state.NewEndpointPaneTerminalView(ref.EndpointID, state.DefaultPaneID, ref.TerminalID, 0, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)
	binding.LastError = errText
	root.TerminalViews = root.TerminalViews.BindPane(binding)

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if content.Kind != ContentTerminalLive || content.Status != "connection interrupted: reconnect or detach" || !contentHasAction(content, ActionDisconnectedReconnect.String()) || !contentHasAction(content, ActionDisconnectedDisconnect.String()) {
		t.Fatalf("disconnected pane should show local reconnect/disconnect choices, got %#v", content)
	}
	if !strings.Contains(content.Lines[0].PlainString(), "last remote output") || !contentPlainContains(content, "● Connection interrupted") || !contentPlainContains(content, "last terminal frame is preserved") || !contentPlainContains(content, "West Lab (west)") || !contentPlainContains(content, "transport ssh") || !contentPlainContains(content, "Remote anytty daemon unavailable") || !contentPlainContains(content, "remote anytty daemon is running") || !strings.Contains(content.Error, "remote-daemon") {
		t.Fatalf("disconnected pane should preserve content and reason, lines=%#v error=%q", content.Lines, content.Error)
	}
}

func TestRenderVMBuilderProjectsConnectingPaneFromAttachPending(t *testing.T) {
	ref := state.NewTerminalRef("west", "remote")
	root := state.Root{Shell: state.DefaultShell()}
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, ref.TerminalID)
	root.Endpoints = root.Endpoints.Upsert(state.EndpointItem{ID: ref.EndpointID, Label: "West Lab", Transport: state.EndpointTransportSSH, Enabled: true, Status: state.EndpointStatusConnecting})
	binding := state.NewEndpointPaneTerminalView(ref.EndpointID, state.DefaultPaneID, ref.TerminalID, 0, 80, 24, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID(state.DefaultPaneID), false)
	binding.AttachPending = true
	root.TerminalViews = root.TerminalViews.BindPane(binding)
	root.Surface = state.TerminalSurfaceStore{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, Lines: []string{"last remote output"}, Ready: true}

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if !content.Pending || content.Status != "connecting: input paused" || !contentPlainContains(content, "◌ Reconnecting terminal") || !contentPlainContains(content, "Input resumes after attach succeeds") || !contentPlainContains(content, "West Lab (west)") || !contentPlainContains(content, "transport ssh") || contentHasAction(content, ActionDisconnectedReconnect.String()) {
		t.Fatalf("attach pending should render structured connecting state, got %#v", content)
	}
}

func TestEndpointRecoveryHintUsesErrorCategory(t *testing.T) {
	cases := map[state.EndpointErrorKind]string{
		state.EndpointErrorAuth:         "credentials",
		state.EndpointErrorHostKey:      "host identity",
		state.EndpointErrorRemoteDaemon: "anytty daemon",
		state.EndpointErrorProtocol:     "protocol session",
		state.EndpointErrorConfig:       "endpoint configuration",
	}
	for kind, want := range cases {
		if got := endpointRecoveryHint(kind); !strings.Contains(got, want) {
			t.Fatalf("recovery hint kind=%q got=%q want substring=%q", kind, got, want)
		}
	}
}

func TestEndpointIssueLabelUsesReadableCategory(t *testing.T) {
	cases := map[state.EndpointErrorKind]string{
		state.EndpointErrorAuth:            "Authentication failed",
		state.EndpointErrorHostKey:         "Remote host identity changed",
		state.EndpointErrorRemoteDaemon:    "Remote anytty daemon unavailable",
		state.EndpointErrorTransportClosed: "Transport connection closed",
		state.EndpointErrorProtocol:        "Protocol session ended",
	}
	for kind, want := range cases {
		if got := endpointIssueLabel(kind); got != want {
			t.Fatalf("issue label kind=%q got=%q want=%q", kind, got, want)
		}
	}
}

func TestRenderVMBuilderDoesNotProjectErroredBindingAsUnconnected(t *testing.T) {
	ref := state.NewTerminalRef("west", "remote")
	root := state.Root{Shell: state.DefaultShell()}
	root.Shell.Workspace.Tabs[0].Panes[0] = state.PaneState{ID: state.DefaultPaneID, Title: "unconnected", Kind: state.PaneEmpty, Active: true}
	binding := state.NewEndpointPaneTerminalView(ref.EndpointID, state.DefaultPaneID, ref.TerminalID, 0, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)
	binding.LastError = "remote-daemon: daemon socket closed"
	root.TerminalViews = root.TerminalViews.BindPane(binding)
	root.Surface = root.Surface.SetErrorRef(ref, binding.LastError)

	content := activeContent(NewRenderVMBuilder().Build(root).Shell)
	if content.Kind == ContentEmptyPane || len(content.Lines) == 0 || content.Lines[0].PlainString() == "unconnected" || !contentHasAction(content, ActionDisconnectedReconnect.String()) {
		t.Fatalf("errored binding must not collapse into unconnected content, got %#v", content)
	}
}

func snapshotUsesMutedPreviewFrame(snapshot WorkbenchSnapshotVM) bool {
	headerOffset := 0
	if len(snapshot.PreviewHeader) > 0 {
		headerOffset = len(snapshot.PreviewHeader) + 1
	}
	return snapshot.PreviewFrame &&
		!snapshot.Borderless &&
		snapshot.Content.X == snapshot.Rect.X+1 &&
		snapshot.Content.Y == snapshot.Rect.Y+1+headerOffset &&
		snapshot.Content.W == maxInt(0, snapshot.Rect.W-2) &&
		snapshot.Content.H == maxInt(0, snapshot.Rect.H-2-headerOffset)
}

func TestTerminalManagerSparklineRowsUsesTwoRowVerticalResolution(t *testing.T) {
	rows := terminalManagerSparklineRows([]uint64{0, 50, 100}, 3, 0, 100, 2)
	if len(rows) != 2 || rows[0] != "  █" || rows[1] != " ██" {
		t.Fatalf("two-row sparkline should stack samples from the baseline, got %#v", rows)
	}
}

func TestRenderVMBuilderProjectsTerminalPoolPage(t *testing.T) {
	shell := state.DefaultShell().OpenTerminalPool().SetTerminalPoolQuery("日志")
	root := state.Root{
		Shell: shell,
		TerminalPool: state.TerminalPoolStore{
			Status: state.TerminalPoolReady,
			Items: []state.TerminalPoolItem{{
				TerminalID:      "term-pool",
				Title:           "日志🚀",
				State:           "running",
				CWD:             "/tmp/日志",
				Cols:            120,
				Rows:            36,
				AttachmentCount: 27,
				Resources: state.TerminalResourceUsage{
					PID:            4321,
					CPUPercentX100: 1234,
					MemoryBytes:    64 * 1024 * 1024,
					SampledAt:      time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC),
				},
				Tags: map[string]string{"role": "shell"},
			}, {
				TerminalID: "term-other",
				Title:      "worker",
				State:      "exited",
			}},
		},
		TerminalViews: state.TerminalViewStore{}.
			BindPane(state.NewPaneTerminalView("pane-main", "term-pool", 7, 120, 36, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID("pane-main"), true)).
			BindFloating(state.NewFloatingTerminalView("float-main", "float-pane", "term-pool", 8, 120, 36, state.TerminalResizeRoleFollower, "surface", state.TerminalFloatingViewID("float-main"), false)),
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-pool",
			Revision:   23,
			Ready:      true,
			Lines:      []string{"line one", "line two", "line three", "line four"},
		},
	}

	vm := NewRenderVMBuilder().Build(root)
	content := vm.Shell.Overlay.Content
	if vm.Shell.Overlay.Kind != OverlayTerminalPool || content.Kind != ContentTerminalPool {
		t.Fatalf("expected terminal pool content, got %#v", vm.Shell.Overlay)
	}
	rendered := plainLines(content.Lines)
	if !strings.Contains(rendered, "⌕ search 日志") ||
		!strings.Contains(rendered, "TERMINAL") ||
		!strings.Contains(rendered, "CPU") ||
		!strings.Contains(rendered, "MEM") ||
		!strings.Contains(rendered, "▸ ● 日志🚀") ||
		!strings.Contains(rendered, "12%") ||
		!strings.Contains(rendered, "64M") ||
		!strings.Contains(rendered, "27") ||
		strings.Contains(rendered, "running ·") ||
		strings.Contains(rendered, "HIST") ||
		strings.Contains(rendered, "CONN") ||
		strings.Contains(rendered, "id:term-pool") ||
		strings.Contains(rendered, "KEYS Enter attach") ||
		strings.Contains(rendered, "clear pending") ||
		strings.Contains(rendered, "Enter Attach") ||
		strings.Contains(rendered, "^E Rename") ||
		strings.Contains(rendered, "VIEWS 27") ||
		strings.Contains(rendered, "views:27") {
		t.Fatalf("expected terminal manager table without connection detail chrome, got %#v", content.Lines)
	}
	if content.Meta.WorkbenchSnapshotPanel == nil ||
		content.Meta.WorkbenchSnapshotPanel.Title != "日志🚀" ||
		content.Meta.WorkbenchSnapshotPanel.Content.Kind != ContentTerminalLive ||
		!strings.Contains(plainLines(content.Meta.WorkbenchSnapshotPanel.Content.Lines), "line one") ||
		!strings.Contains(plainLines(content.Meta.WorkbenchSnapshotPanel.Content.Lines), "line four") ||
		len(content.Meta.WorkbenchSnapshots) != 1 ||
		!snapshotUsesMutedPreviewFrame(content.Meta.WorkbenchSnapshots[0]) {
		t.Fatalf("expected terminal manager framed live panel meta, got %#v", content.Meta)
	}
	previewHeader := content.Meta.WorkbenchSnapshots[0].PreviewHeader
	previewHeaderText := plainLines(previewHeader)
	if len(previewHeader) != terminalManagerTrendHeaderRows ||
		!strings.Contains(previewHeaderText, "CPU") ||
		!strings.Contains(previewHeaderText, "12%") ||
		!strings.Contains(previewHeaderText, "MEM") ||
		!strings.Contains(previewHeaderText, "64M") {
		t.Fatalf("expected compact CPU and memory trend header, got %#v", previewHeader)
	}
	if strings.Contains(content.Lines[3].PlainString(), "worker") {
		t.Fatalf("query should filter non-matching row, got %#v", content.Lines)
	}
	if !contentHasAction(content, "pool.select") ||
		contentHasAction(content, "pool.attach") ||
		contentHasAction(content, "pool.restart") ||
		contentHasAction(content, "pool.delete") {
		t.Fatalf("expected only pool select in content hit regions, got %#v", content.HitRegions)
	}
	if !content.Cursor.Visible || content.Cursor.Col != DisplayWidth("⌕ search 日志") {
		t.Fatalf("expected pool search cursor, got %#v", content.Cursor)
	}

	root.TerminalPool = state.TerminalPoolStore{Status: state.TerminalPoolLoading}
	content = NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if !content.Pending || !strings.Contains(plainLines(content.Lines), "loading terminals") {
		t.Fatalf("expected pool loading page, got %#v", content)
	}
	root.TerminalPool = state.TerminalPoolStore{Status: state.TerminalPoolError, LastError: "boom"}
	content = NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if content.Error != "boom" || !strings.Contains(plainLines(content.Lines), "list error boom") {
		t.Fatalf("expected pool error page, got %#v", content)
	}
	root.TerminalPool = state.TerminalPoolStore{Status: state.TerminalPoolReady}
	root.Shell = state.DefaultShell().OpenTerminalPool()
	content = NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if !content.Empty || !strings.Contains(plainLines(content.Lines), "list empty") {
		t.Fatalf("expected pool empty page, got %#v", content)
	}
}

func TestRenderVMBuilderProjectsTerminalPoolEndpointGroups(t *testing.T) {
	root := state.Root{
		Shell:    state.DefaultShell().OpenTerminalPool(),
		Viewport: state.ViewportStore{Valid: true, Cols: 120, Rows: 32},
		Endpoints: (state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "This Mac", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true, Status: state.EndpointStatusConnected}).
			Upsert(state.EndpointItem{ID: "west", Label: "US West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectManual, Enabled: true}),
		TerminalPool: state.TerminalPoolStore{
			Status: state.TerminalPoolReady,
			Items: []state.TerminalPoolItem{
				{EndpointID: state.DefaultEndpointID, TerminalID: "term-1", Title: "shell", State: "running", Cols: 80, Rows: 24},
				{EndpointID: "west", TerminalID: "term-1", Title: "shell", State: "running", Cols: 100, Rows: 30},
			},
		},
	}

	content := NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	rendered := plainLines(content.Lines)
	for _, want := range []string{"TERMINAL", "This Mac", "US West", "shell"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected terminal manager endpoint content %q in:\n%s", want, rendered)
		}
	}
	for _, hidden := range []string{"connected", "manual", "ssh"} {
		if strings.Contains(rendered, hidden) {
			t.Fatalf("terminal manager endpoint groups must hide connection state %q in:\n%s", hidden, rendered)
		}
	}
	for _, region := range content.HitRegions {
		if region.ActionID == ActionPoolSelect.String() && region.Row != 0 && region.Row != 1 {
			t.Fatalf("pool select hit regions must point to terminal flat rows, got %#v", content.HitRegions)
		}
	}
}

func TestRenderVMBuilderTerminalManagerProjectsManagedEndpointPhaseAndPath(t *testing.T) {
	root := state.Root{
		Shell:    state.DefaultShell().OpenTerminalPool(),
		Viewport: state.ViewportStore{Valid: true, Cols: 120, Rows: 32},
		Endpoints: (state.EndpointStore{}).Upsert(state.EndpointItem{
			ID: "studio", Label: "Studio", Transport: state.EndpointTransportHubP2P,
			ConnectMode: state.EndpointConnectOnDemand, Enabled: true, Status: state.EndpointStatusConnecting,
			ConnectionPhase: "authorizing", ObservedPath: "direct",
		}),
		TerminalPool: state.TerminalPoolStore{Status: state.TerminalPoolReady},
	}

	content := NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	plain := plainLines(content.Lines)
	if !strings.Contains(plain, "Studio") || strings.Contains(plain, "authorizing") || strings.Contains(plain, "direct") || strings.Contains(plain, "hub-p2p") {
		t.Fatalf("terminal manager should expose only the remote endpoint label, got:\n%s", plain)
	}
}

func TestRenderVMBuilderKeepsEmptyTerminalPoolLivePanel(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell().OpenTerminalPool(),
		TerminalPool: state.TerminalPoolStore{
			Status: state.TerminalPoolReady,
			Items: []state.TerminalPoolItem{{
				TerminalID:      "term-blank",
				Title:           "blank",
				State:           "running",
				Cols:            80,
				Rows:            24,
				AttachmentCount: 1,
			}},
		},
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-blank",
			Revision:   7,
			Ready:      true,
			Lines:      []string{"", "   "},
		},
	}

	content := NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	rendered := plainLines(content.Lines)
	if !strings.Contains(rendered, "blank") ||
		content.Meta.WorkbenchSnapshotPanel == nil ||
		len(content.Meta.WorkbenchSnapshots) != 1 ||
		!snapshotUsesMutedPreviewFrame(content.Meta.WorkbenchSnapshots[0]) ||
		strings.Contains(rendered, "│") {
		t.Fatalf("expected empty terminal to keep a framed live panel, got %#v", content.Meta)
	}
}

func TestRenderVMBuilderTerminalPoolPreviewUsesLiveScreenArea(t *testing.T) {
	liveRows := make([]string, 0, 14)
	for index := 0; index < 14; index++ {
		liveRows = append(liveRows, fmt.Sprintf("screen-%02d", index))
	}
	root := state.Root{
		Shell:    state.DefaultShell().OpenTerminalPool(),
		Viewport: state.ViewportStore{Valid: true, Cols: 140, Rows: 34},
		TerminalPool: state.TerminalPoolStore{
			Status: state.TerminalPoolReady,
			Items: []state.TerminalPoolItem{{
				TerminalID: "term-live-preview",
				Title:      "live-preview",
				State:      "running",
				Cols:       100,
				Rows:       28,
			}},
		},
		Surface: (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
			TerminalID: "term-live-preview",
			Revision:   42,
			Cols:       100,
			Rows:       28,
			Lines:      liveRows,
		}),
	}

	content := NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if content.Meta.WorkbenchSnapshotPanel == nil ||
		len(content.Meta.WorkbenchSnapshots) != 1 ||
		!snapshotUsesMutedPreviewFrame(content.Meta.WorkbenchSnapshots[0]) ||
		!strings.Contains(plainLines(content.Meta.WorkbenchSnapshotPanel.Content.Lines), "screen-00") ||
		!strings.Contains(plainLines(content.Meta.WorkbenchSnapshotPanel.Content.Lines), "screen-13") ||
		strings.Contains(plainLines(content.Lines), "PREVIEW") {
		t.Fatalf("expected live terminal rows to fill manager snapshot area, got lines=%#v meta=%#v", content.Lines, content.Meta)
	}
}

func TestRenderVMBuilderProjectsWorkbenchTreeOverlay(t *testing.T) {
	shell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "日志🚀", Kind: state.PaneTerminalLive, TerminalID: "term-2"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	var floatingResult state.FloatingCommandResult
	shell, floatingResult = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "float-logs",
		Pane:     state.PaneState{ID: "float-logs-pane", Title: "日志浮窗", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Title:    "日志浮窗",
	})
	if floatingResult.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", floatingResult)
	}
	shell = shell.
		OpenWorkbenchTree().
		SetWorkbenchTreeQuery("日志")
	surfaces := state.TerminalSurfaceStore{}
	surfaces = surfaces.ApplySnapshot(state.LiveSurfaceSnapshot{TerminalID: "term-2", Cols: 32, Rows: 8, Lines: []string{"live snapshot row"}})
	surfaces = surfaces.ApplySnapshot(state.LiveSurfaceSnapshot{TerminalID: "term-float", Cols: 30, Rows: 7, Lines: []string{"floating snapshot row"}})
	root := state.Root{
		Shell:        shell,
		TerminalPool: state.TerminalPoolStore{Status: state.TerminalPoolReady, Items: []state.TerminalPoolItem{{TerminalID: "term-main", Title: "shell", State: "running"}, {TerminalID: "term-2", Title: "日志终端", State: "running"}, {TerminalID: "term-float", Title: "浮窗终端", State: "running"}}},
		Surface:      surfaces,
		Session:      state.TerminalSessionStore{TerminalID: "term-2", Attached: true, Cols: 32, Rows: 8, State: state.TerminalLiveAttached},
	}
	root = bindTestPaneTerminal(root, state.DefaultPaneID, "term-main")
	root = bindTestPaneTerminal(root, "pane-2", "term-2")
	root.TerminalViews = root.TerminalViews.BindFloating(state.NewFloatingTerminalView("float-logs", "float-logs-pane", "term-float", 8, 30, 7, state.TerminalResizeRoleFollower, "surface", state.TerminalFloatingViewID("float-logs"), true))

	vm := NewRenderVMBuilder().Build(root)
	content := vm.Shell.Overlay.Content
	if vm.Shell.Overlay.Kind != OverlayWorkbenchTree || !vm.Shell.Overlay.Opaque || content.Kind != ContentWorkbenchTree {
		t.Fatalf("expected workbench tree opaque overlay, got %#v", vm.Shell.Overlay)
	}
	if !strings.Contains(content.Lines[0].PlainString(), "⌕ search 日志") ||
		!strings.Contains(content.Lines[2].PlainString(), "WORKBENCH") ||
		strings.Contains(content.Lines[2].PlainString(), "DETAIL") ||
		!strings.Contains(content.Lines[3].PlainString(), "日志终端") ||
		!lineHasStyledANSICell(content.Lines[3], "日志终端", StyleAccent, ANSICellStyle{Underline: true}) ||
		!strings.Contains(content.Lines[4].PlainString(), "浮窗终端") ||
		!lineHasStyledCell(content.Lines[4], " F ", StyleSuccess) ||
		strings.HasPrefix(content.Lines[3].PlainString(), "▸") ||
		strings.Contains(content.Lines[3].PlainString(), "日志🚀") ||
		strings.Contains(content.Lines[3].PlainString(), "terminal-live") ||
		strings.Contains(content.Lines[3].PlainString(), "term:") ||
		strings.Contains(content.Lines[3].PlainString(), "0.0%") ||
		strings.Contains(content.Lines[3].PlainString(), "│") ||
		strings.Contains(content.Lines[3].PlainString(), "├") ||
		strings.Contains(content.Lines[3].PlainString(), "└") ||
		strings.Contains(content.Lines[3].PlainString(), "›") ||
		strings.Contains(plainLines(content.Lines), "VIEWS") ||
		strings.Contains(plainLines(content.Lines), "Open  New  Zoom") ||
		strings.Contains(plainLines(content.Lines), "[open]") {
		t.Fatalf("expected navigator tree/snapshot summary content, got %#v", content.Lines)
	}
	if content.Meta.WorkbenchTreeWidth < 38 ||
		content.Meta.WorkbenchBodyRows < 8 ||
		content.Meta.WorkbenchSnapshotRect.W < 30 ||
		content.Meta.WorkbenchSnapshotRect.H < 3 ||
		content.Meta.WorkbenchSnapshotRect.X <= content.Meta.WorkbenchTreeWidth ||
		content.Meta.WorkbenchSnapshotPanel == nil ||
		len(content.Meta.WorkbenchSnapshots) != 1 ||
		!snapshotUsesMutedPreviewFrame(content.Meta.WorkbenchSnapshots[0]) ||
		content.Meta.WorkbenchSnapshots[0].ShowTitle ||
		content.Meta.WorkbenchSnapshotPanel.Content.Kind != ContentTerminalLive ||
		!strings.Contains(plainLines(content.Meta.WorkbenchSnapshotPanel.Content.Lines), "live snapshot row") ||
		!contentHasAction(content, "workbench.open") ||
		contentHasAction(content, "workbench.zoom") ||
		contentHasAction(content, "workbench.detach") ||
		contentHasAction(content, "workbench.delete") {
		t.Fatalf("expected workbench hit regions, got %#v", content.HitRegions)
	}
	detailOpenRegion := false
	for _, region := range content.HitRegions {
		if region.ActionID == ActionWorkbenchOpen.String() && region.Row < 0 && region.Rect.X > content.Meta.WorkbenchTreeWidth && region.Rect.H > 1 {
			detailOpenRegion = true
			break
		}
	}
	if !detailOpenRegion {
		t.Fatalf("expected right-side workbench detail open hit region, got %#v", content.HitRegions)
	}
	if !content.Cursor.Visible || content.Cursor.Col != DisplayWidth("⌕ search 日志") {
		t.Fatalf("expected workbench search cursor, got %#v", content.Cursor)
	}
	if content.Cursor.Row != 0 || content.Meta.WorkbenchActionRow != -1 {
		t.Fatalf("expected workbench cursor on search row and no content action row, cursor=%#v meta=%#v lines=%d", content.Cursor, content.Meta, len(content.Lines))
	}
	if len(content.Meta.WorkbenchSnapshots) != 1 ||
		!lineHasStyledCell(content.Lines[3], " P ", StyleAccent) ||
		!lineHasStyledCell(content.Lines[3], "日志终端", StyleAccent) ||
		strings.Contains(content.Lines[3].PlainString(), "terminal-live") ||
		strings.Contains(content.Lines[3].PlainString(), "term:term-2") {
		t.Fatalf("expected selected pane row styled tokens and one snapshot, meta=%#v line=%#v", content.Meta, content.Lines[3])
	}

	root.Shell = root.Shell.SetWorkbenchTreeQuery("")
	root.Shell = root.Shell.MoveWorkbenchTreeSelection(1, len(state.WorkbenchTreeItems(root)))
	content = NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if len(content.Meta.WorkbenchSnapshots) != 2 ||
		!strings.HasPrefix(content.Lines[3].PlainString(), "▾") ||
		!strings.HasPrefix(content.Lines[4].PlainString(), "  ▾") ||
		!strings.HasPrefix(content.Lines[5].PlainString(), "      ") ||
		content.Meta.WorkbenchSnapshots[0].Panel.ID != state.DefaultPaneID ||
		content.Meta.WorkbenchSnapshots[0].Panel.Title != "shell" ||
		content.Meta.WorkbenchSnapshots[1].Panel.ID != "pane-2" ||
		content.Meta.WorkbenchSnapshots[1].Panel.Title != "日志终端" ||
		!snapshotUsesMutedPreviewFrame(content.Meta.WorkbenchSnapshots[0]) ||
		!snapshotUsesMutedPreviewFrame(content.Meta.WorkbenchSnapshots[1]) ||
		!content.Meta.WorkbenchSnapshots[0].ShowTitle ||
		!content.Meta.WorkbenchSnapshots[1].ShowTitle ||
		len(content.Meta.WorkbenchSnapshotSplit.Children) != 2 ||
		content.Meta.WorkbenchSnapshotCanvas.W <= 0 ||
		content.Meta.WorkbenchSnapshotCanvas.H <= 0 ||
		content.Meta.WorkbenchSnapshots[0].Rect.X == content.Meta.WorkbenchSnapshots[1].Rect.X ||
		content.Meta.WorkbenchSnapshots[0].Panel.Chrome.Terminal.Title.Text != "shell" ||
		content.Meta.WorkbenchSnapshots[1].Panel.Chrome.Terminal.Title.Text != "日志终端" ||
		!strings.Contains(plainLines(content.Meta.WorkbenchSnapshots[1].Panel.Content.Lines), "live snapshot row") {
		t.Fatalf("tab selection should project its RootSplit panes without floating windows, meta=%#v", content.Meta)
	}

	root.Shell = state.DefaultShell().OpenWorkbenchTree().SetWorkbenchTreeQuery("missing")
	content = NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if !content.Empty || len(content.Meta.WorkbenchSnapshots) != 0 || strings.Contains(plainLines(content.Lines), "No workbench node selected") {
		t.Fatalf("expected empty tree page, got %#v", content)
	}
}

func TestWorkbenchNavigatorSnapshotsFollowNestedRootSplit(t *testing.T) {
	shell := state.DefaultShell()
	tab := shell.Workspace.Tabs[0]
	tab.Panes = []state.PaneState{
		{ID: "pane-a", Title: "a", Kind: state.PaneTerminalLive, TerminalID: "term-a"},
		{ID: "pane-b", Title: "b", Kind: state.PaneTerminalLive, TerminalID: "term-b"},
		{ID: "pane-c", Title: "c", Kind: state.PaneTerminalLive, TerminalID: "term-c"},
	}
	tab.ActivePaneID = "pane-b"
	tab.RootSplit = state.SplitNode{
		Direction: state.SplitDirectionVertical,
		Ratio:     0.4,
		Children: []state.SplitNode{
			{PaneID: "pane-a"},
			{
				Direction: state.SplitDirectionHorizontal,
				Ratio:     0.5,
				Children:  []state.SplitNode{{PaneID: "pane-b"}, {PaneID: "pane-c"}},
			},
		},
	}
	shell.Workspace.Tabs[0] = tab
	shell.Workspace.ActiveTabID = tab.ID
	shell.ActivePaneID = tab.ActivePaneID
	shell.Workspaces = []state.WorkspaceState{shell.Workspace}
	root := state.Root{Shell: shell}
	selected := state.WorkbenchTreeItem{
		Kind:        state.WorkbenchTreeKindTab,
		WorkspaceID: shell.Workspace.ID,
		TabID:       tab.ID,
	}
	layout := workbenchNavigatorLayout{TreeWidth: 30, RightWidth: 70, BodyRows: 20}

	snapshots, canvas, split := workbenchNavigatorSnapshotVMs(root, selected, layout)
	if canvas != (Rect{X: 31, Y: 2, W: 70, H: 21}) || len(split.Children) != 2 || len(snapshots) != 3 {
		t.Fatalf("expected nested RootSplit canvas, canvas=%#v split=%#v snapshots=%#v", canvas, split, snapshots)
	}
	wantRects := []Rect{
		{X: 31, Y: 2, W: 29, H: 21},
		{X: 59, Y: 2, W: 42, H: 11},
		{X: 59, Y: 12, W: 42, H: 11},
	}
	for index, snapshot := range snapshots {
		if snapshot.Rect != wantRects[index] || !snapshotUsesMutedPreviewFrame(snapshot) || !snapshot.ShowTitle {
			t.Fatalf("snapshot %d must follow nested split geometry, got=%#v wantRect=%#v", index, snapshot, wantRects[index])
		}
	}
}

func TestWorkbenchTreeKindGlyphsUsePortableSingleCellLabels(t *testing.T) {
	for name, item := range map[string]struct {
		kind string
		want string
	}{
		"workspace": {state.WorkbenchTreeKindWorkspace, "W"},
		"tab":       {state.WorkbenchTreeKindTab, "T"},
		"pane":      {state.WorkbenchTreeKindPane, "P"},
		"floating":  {state.WorkbenchTreeKindFloating, "F"},
	} {
		glyph := workbenchTreeKindGlyph(state.WorkbenchTreeItem{Kind: item.kind})
		if glyph != item.want || DisplayWidth(glyph) != 1 {
			t.Fatalf("%s kind glyph must be a visible single-cell label, got=%q want=%q width=%d", name, glyph, item.want, DisplayWidth(glyph))
		}
	}
}

func TestRenderVMBuilderProjectsWorkbenchTreeFloatingSnapshot(t *testing.T) {
	shell := state.DefaultShell()
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "float-1",
		Pane:     state.PaneState{ID: "float-pane", Title: "float pane", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	shell = shell.OpenWorkbenchTree().SetWorkbenchTreeQuery("float pane")
	root := state.Root{
		Shell: shell,
		Surface: (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
			TerminalID: "term-float",
			Cols:       32,
			Rows:       8,
			Lines:      []string{"floating live row"},
		}),
		Session: state.TerminalSessionStore{TerminalID: "term-float", Attached: true, Cols: 32, Rows: 8, State: state.TerminalLiveAttached},
	}
	root.TerminalViews = root.TerminalViews.BindFloating(state.NewFloatingTerminalView("float-1", "float-pane", "term-float", 8, 32, 8, state.TerminalResizeRoleFollower, "surface", state.TerminalFloatingViewID("float-1"), true))

	content := NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	if !strings.Contains(content.Lines[3].PlainString(), "term-float") ||
		!lineHasStyledCell(content.Lines[3], " F ", StyleAccent) ||
		strings.Contains(content.Lines[3].PlainString(), "terminal-live") ||
		len(content.Meta.WorkbenchSnapshots) != 1 ||
		content.Meta.WorkbenchSnapshots[0].Panel.Title != "term-float" ||
		!snapshotUsesMutedPreviewFrame(content.Meta.WorkbenchSnapshots[0]) ||
		content.Meta.WorkbenchSnapshots[0].ShowTitle ||
		!strings.Contains(plainLines(content.Meta.WorkbenchSnapshots[0].Panel.Content.Lines), "floating live row") {
		t.Fatalf("expected floating tree row and snapshot, line=%#v meta=%#v", content.Lines[3], content.Meta)
	}
}

func TestRenderVMBuilderProjectsPromptAndHelpOverlay(t *testing.T) {
	shell := state.DefaultShell().OpenPrompt(state.PromptState{
		Title:       "Rename Pane",
		Context:     "输入新名称",
		Value:       "日志🚀",
		Placeholder: "name",
	})
	content := NewRenderVMBuilder().Build(state.Root{Shell: shell}).Shell.Overlay.Content
	if content.Kind != ContentPrompt ||
		!strings.Contains(content.Lines[0].PlainString(), "Rename Pane") ||
		!strings.Contains(content.Lines[1].PlainString(), "NAME 日志🚀") ||
		contentHasAction(content, "prompt.submit") ||
		contentHasAction(content, "prompt.cancel") {
		t.Fatalf("expected prompt content, got %#v", content)
	}
	if !content.Cursor.Visible || content.Cursor.Row != 1 || content.Cursor.Col != DisplayWidth("Name 日志🚀") {
		t.Fatalf("expected prompt cursor after input, got %#v", content.Cursor)
	}

	content = NewRenderVMBuilder().Build(state.Root{Shell: state.DefaultShell().OpenPrompt(state.PromptState{
		Title:       "Rename Terminal",
		Purpose:     "terminal.rename",
		Value:       "shell",
		Placeholder: "terminal name",
	})}).Shell.Overlay.Content
	if len(content.Lines) != 1 || content.Lines[0].PlainString() != "shell" {
		t.Fatalf("expected compact terminal rename prompt, got %#v", content.Lines)
	}
	if content.Cursor.Row != 0 || content.Cursor.Col != DisplayWidth("shell") {
		t.Fatalf("expected compact prompt cursor after input, got %#v", content.Cursor)
	}

	content = NewRenderVMBuilder().Build(state.Root{Shell: state.DefaultShell().OpenPrompt(state.PromptState{
		Title:       "Create Terminal",
		ActiveField: 1,
		Fields: []state.PromptFieldState{
			{Key: "name", Label: "name", Value: "shell", Required: true},
			{Key: "command", Label: "command", Value: "/bin/sh"},
			{Key: "workdir", Label: "workdir", Value: "/tmp"},
		},
	})}).Shell.Overlay.Content
	if !strings.Contains(content.Lines[0].PlainString(), "Create Terminal") ||
		!strings.Contains(content.Lines[1].PlainString(), "name*: shell") ||
		!strings.Contains(content.Lines[2].PlainString(), "command: /bin/sh") ||
		strings.Contains(plainLines(content.Lines), "name is required") ||
		contentHasAction(content, "prompt.submit") ||
		contentHasAction(content, "prompt.cancel") {
		t.Fatalf("expected create terminal form content, got %#v", content.Lines)
	}
	if !lineHasStyledCell(content.Lines[1], "name*: ", StyleStrongForeground) {
		t.Fatalf("expected prompt labels to use strong foreground, got %#v", content.Lines[1])
	}
	if content.Cursor.Row != 2 || content.Cursor.Col != DisplayWidth("command: /bin/sh") {
		t.Fatalf("expected form cursor on active command field, got %#v", content.Cursor)
	}

	promptVM := NewRenderVMBuilder().Build(state.Root{Shell: state.DefaultShell().OpenPrompt(state.PromptState{
		Title:              "Create Terminal",
		ActiveField:        2,
		SuggestionFocused:  true,
		SuggestionSelected: 6,
		SuggestionOffset:   1,
		Fields: []state.PromptFieldState{
			{Key: "name", Label: "name", Value: "shell", Required: true},
			{Key: "command", Label: "command", Value: "/bin/sh"},
			{
				Key:             "workdir",
				Label:           "workdir",
				Value:           "/tmp/de",
				Cursor:          len([]rune("/tmp/de")),
				SuggestionTitle: "path: /tmp",
				SuggestionItems: []string{
					"/tmp/d0/",
					"/tmp/d1/",
					"/tmp/d2/",
					"/tmp/d3/",
					"/tmp/d4/",
					"/tmp/d5/",
					"/tmp/dev/",
					"/tmp/d7/",
				},
			},
		},
	})})
	content = promptVM.Shell.Overlay.Content
	popup := promptVM.Shell.Overlay.Popup
	if strings.Contains(plainLines(content.Lines), "path: /tmp") ||
		strings.Contains(plainLines(content.Lines), "/tmp/dev/") {
		t.Fatalf("prompt content should not own suggestion popup rows, got %#v", content.Lines)
	}
	if popup.Kind != OverlayPopupPromptSuggestion ||
		!strings.Contains(plainLines(popup.Lines), "path: /tmp") ||
		!strings.Contains(plainLines(popup.Lines), "▸   dev/") ||
		!strings.Contains(plainLines(popup.Lines), "2-7/8") ||
		strings.Contains(plainLines(popup.Lines), "/tmp/d0/") ||
		strings.Contains(plainLines(popup.Lines), "│") ||
		!lineHasStyledCell(popup.Lines[6], "dev/", StylePromptSuggestionHit) {
		t.Fatalf("expected workdir suggestion popup rows, got %#v", popup)
	}
	if content.Cursor.Row != 3 || content.Cursor.Col != DisplayWidth("workdir: /tmp/de") {
		t.Fatalf("expected cursor on workdir field before suggestion rows, got %#v", content.Cursor)
	}

	helpRoot := state.Root{Shell: state.DefaultShell().OpenHelp("most-used")}
	content = NewRenderVMBuilder().Build(helpRoot).Shell.Overlay.Content
	helpPlain := plainLines(content.Lines)
	if content.Kind != ContentHelp ||
		!strings.Contains(content.Lines[0].PlainString(), "Help") ||
		!strings.Contains(helpPlain, "Most used") ||
		!strings.Contains(helpPlain, "[Ctrl+P]") ||
		!strings.Contains(helpPlain, "[Ctrl+R]") ||
		strings.Contains(helpPlain, "[Ctrl] •") ||
		!contentHasAction(content, "help.close") {
		t.Fatalf("expected help content, got %#v", content)
	}
	helpWidth, helpHeight := helpContentDimensions(helpRoot)
	if len(content.Lines) > helpHeight {
		t.Fatalf("help page must fit its content viewport: lines=%d height=%d", len(content.Lines), helpHeight)
	}
	for _, line := range content.Lines {
		if line.Width() > helpWidth {
			t.Fatalf("help row must fit without horizontal clipping: width=%d limit=%d row=%q", line.Width(), helpWidth, line.PlainString())
		}
	}
	entries := input.ShortcutEntriesForHelp(helpRoot.Config.Shortcuts, helpRoot.HostCapabilities.KeyboardDisambiguation)
	helpRoot.Shell = helpRoot.Shell.SetHelpSelection(len(entries)-1, len(entries))
	lastPage := plainLines(NewRenderVMBuilder().Build(helpRoot).Shell.Overlay.Content.Lines)
	if !strings.Contains(lastPage, fmt.Sprintf("shortcuts %d/%d", len(entries), len(entries))) ||
		!strings.Contains(lastPage, "[enter]") || !strings.Contains(lastPage, "Close Help") {
		t.Fatalf("help navigation must expose the final configured binding, got %q", lastPage)
	}
	if content.Cursor.Visible {
		t.Fatalf("help overlay should not show input cursor, got %#v", content.Cursor)
	}
}

func TestRenderVMBuilderShowsLiveError(t *testing.T) {
	root := state.Root{
		Surface: state.TerminalSurfaceStore{TerminalID: "term-live", Err: "boom"},
	}
	root = bindTestPaneTerminal(root, state.DefaultPaneID, "term-live")

	vm := NewRenderVMBuilder().Build(root)
	content := activeContent(vm.Shell)
	if content.Status != "error: boom" || content.Error != "boom" {
		t.Fatalf("expected error content, got %#v", content)
	}
}

func TestRendererConsumesShellVMAndSanitizesContent(t *testing.T) {
	renderer := NewRenderer(DefaultTheme())
	result := renderer.RenderResult(RenderVM{
		Shell: ShellVM{
			Cursor: Cursor{Visible: true, Row: 2, Col: 3, Shape: CursorShapeBlock},
			Footer: FooterVM{Visible: true, Mode: "copy", Hint: "copy"},
			Layout: LayoutVM{Panels: []PanelVM{{
				ID:           "pane-1",
				Title:        "copy",
				Presentation: PanelPresentationCard,
				Active:       true,
				Content: ContentVM{
					Kind:  ContentCopyHistory,
					Lines: []Line{NewLine("hello\nworld")},
				},
			}}},
		},
	})
	frame := result.Frame()

	if len(frame.Lines) != defaultHeight {
		t.Fatalf("expected default frame height %d, got %d", defaultHeight, len(frame.Lines))
	}
	if !frameContains(frame, "hello world") {
		t.Fatalf("expected sanitized content inside panel, got %#v", frame.Lines)
	}
	if !frameContains(frame, "copy") {
		t.Fatalf("expected styled status line to contain text, got %q", frame.Lines[1])
	}
	if result.Metadata.Height != defaultHeight || result.Metadata.Width != defaultWidth {
		t.Fatalf("expected render metadata, got %#v", result.Metadata)
	}
	if !result.Cursor.Visible || result.Cursor.Row != 2 || result.Cursor.Col != 3 {
		t.Fatalf("expected cursor passthrough, got %#v", result.Cursor)
	}
}

func frameContains(frame Frame, value string) bool {
	for _, line := range frame.Lines {
		if strings.Contains(line, value) {
			return true
		}
	}
	return false
}

func containsFooterAction(actions []FooterActionVM, key string, label string, actionID string) bool {
	for _, action := range actions {
		if action.Key == key && action.Label == label && action.ActionID == actionID {
			return true
		}
	}
	return false
}

func containsFooterActionID(actions []FooterActionVM, actionID string) bool {
	for _, action := range actions {
		if action.ActionID == actionID {
			return true
		}
	}
	return false
}

func lineHasStyledCell(line Line, text string, style StyleToken) bool {
	for _, cell := range line.Cells {
		if cell.Text == text && cell.Style == style {
			return true
		}
	}
	return false
}

func lineHasStyledANSICell(line Line, text string, style StyleToken, ansiStyle ANSICellStyle) bool {
	for _, cell := range line.Cells {
		if cell.Text == text && cell.Style == style && cell.ANSIStyle == ansiStyle {
			return true
		}
	}
	return false
}

func lineHasANSICell(line Line, text string, style ANSICellStyle) bool {
	for _, cell := range line.Cells {
		if cell.Text == text && cell.ANSIStyle == style && cell.Style == "" {
			return true
		}
	}
	return false
}

func lineANSIStyleDisplayWidth(line Line, style ANSICellStyle) int {
	width := 0
	for _, cell := range line.Cells {
		if cell.ANSIStyle == style && cell.Style == "" {
			width += maxInt(0, cell.Width)
		}
	}
	return width
}

func displayColumnOf(line string, needle string) int {
	index := strings.Index(line, needle)
	if index < 0 {
		return -1
	}
	return DisplayWidth(line[:index])
}

func lineHasLinkCell(line Line, text string, linkURL string, linkParams string) bool {
	for _, cell := range line.Cells {
		if cell.Text == text && cell.LinkURL == linkURL && cell.LinkParams == linkParams {
			return true
		}
	}
	return false
}

func contentHasAction(content ContentVM, actionID string) bool {
	for _, region := range content.HitRegions {
		if region.Kind == HitRegionContentAction && region.ActionID == actionID {
			return true
		}
	}
	return false
}

func contentPlainContains(content ContentVM, text string) bool {
	for _, line := range content.Lines {
		if strings.Contains(line.PlainString(), text) {
			return true
		}
	}
	return false
}

func TestStyleHelpersWidthAndTruncateANSI(t *testing.T) {
	styled := StatusStyle(DefaultTheme()).Render("abcdef")
	if Width(styled) != 6 {
		t.Fatalf("expected ANSI-safe width 6, got %d", Width(styled))
	}
	if got := Truncate(styled, 3); Width(got) != 3 {
		t.Fatalf("expected truncated width 3, got width=%d value=%q", Width(got), got)
	}
	if got := SafeLine("a\nb"); got != "a b" {
		t.Fatalf("unexpected safe line %q", got)
	}
}

func TestWidthSafeHelpersKeepRowsAtTargetDisplayWidth(t *testing.T) {
	cases := []struct {
		name  string
		value string
		width int
	}{
		{name: "cjk", value: "世界", width: 6},
		{name: "emoji", value: "ok 🚀", width: 8},
		{name: "combining", value: "e\u0301cho", width: 7},
		{name: "ansi", value: StatusStyle(DefaultTheme()).Render("warn 🚀"), width: 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fitted := FitText(tc.value, tc.width)
			if got := DisplayWidth(fitted); got != tc.width {
				t.Fatalf("expected fitted width %d, got %d for %q", tc.width, got, fitted)
			}
			line := LineFromText(tc.value, tc.width)
			if got := line.Width(); got != tc.width {
				t.Fatalf("expected line width %d, got %d for %#v", tc.width, got, line)
			}
		})
	}
}

func TestWidthSafeTruncateDoesNotExceedTargetWidth(t *testing.T) {
	value := StatusStyle(DefaultTheme()).Render("title 🚀 世界")
	for width := 1; width <= 8; width++ {
		truncated := TruncateCells(value, width)
		if got := DisplayWidth(truncated); got > width {
			t.Fatalf("truncated width exceeds target width=%d got=%d value=%q", width, got, truncated)
		}
	}
}

func TestRenderVMBuilderCarriesHostAwareTheme(t *testing.T) {
	root := state.Root{}
	root.HostTheme = root.HostTheme.ApplyUpdate(state.HostThemeUpdate{DefaultFG: "#eeeeee"})
	root.HostTheme = root.HostTheme.ApplyUpdate(state.HostThemeUpdate{DefaultBG: "#101010"})
	root.HostTheme = root.HostTheme.ApplyUpdate(state.HostThemeUpdate{PaletteIndex: 5, PaletteColor: "#bb66ff"})
	vm := NewRenderVMBuilder().Build(root)
	if vm.Theme.HostFG != "#eeeeee" || vm.Theme.HostBG != "#101010" || vm.Theme.Accent != "#bb66ff" {
		t.Fatalf("expected host-aware theme in render VM, got %#v", vm.Theme)
	}
	frame := NewRenderer(DefaultTheme()).Render(RenderVM{
		Shell: ShellVM{
			Layout: LayoutVM{
				Viewport: Rect{W: 20, H: 3},
				Panels: []PanelVM{{
					ID:           "pane-1",
					Title:        "pane",
					Active:       true,
					Presentation: PanelPresentationCard,
					Content:      ContentVM{Kind: ContentTerminalLive, Lines: []Line{NewLine("ok")}},
				}},
			},
		},
		Theme: vm.Theme,
	})
	if frame.Theme.Accent != "#bb66ff" {
		t.Fatalf("renderer should prefer VM host-aware theme, got %#v", frame.Theme)
	}
	if !strings.Contains(strings.Join(frame.ANSILines, "\n"), "38;2;187;102;255") {
		t.Fatalf("styled frame should use host-aware accent, got %#v", frame.ANSILines)
	}
}

func TestRenderVMBuilderCarriesUserConfiguredTheme(t *testing.T) {
	root := state.Root{}
	root.HostTheme = root.HostTheme.ApplyUpdate(state.HostThemeUpdate{PaletteIndex: 5, PaletteColor: "#bb66ff"})
	root.Config = state.TUIConfigStore{
		Theme: state.TUIThemeConfig{
			Palette: "host",
			Primary: "#d65cff",
		},
	}
	vm := NewRenderVMBuilder().Build(root)
	if vm.Theme.Accent != "#d65cff" || vm.Theme.ActivePaneBorder != "#d65cff" {
		t.Fatalf("configured primary should override host accent in VM, got %#v", vm.Theme)
	}
	frame := NewRenderer(DefaultTheme()).Render(RenderVM{
		Shell: ShellVM{
			Layout: LayoutVM{
				Viewport: Rect{W: 20, H: 3},
				Panels: []PanelVM{{
					ID:           "pane-1",
					Title:        "pane",
					Active:       true,
					Presentation: PanelPresentationCard,
					Content:      ContentVM{Kind: ContentTerminalLive, Lines: []Line{NewLine("ok")}},
				}},
			},
		},
		Theme: vm.Theme,
	})
	if frame.Theme.Accent != "#d65cff" {
		t.Fatalf("renderer should prefer configured VM theme, got %#v", frame.Theme)
	}
	if !strings.Contains(strings.Join(frame.ANSILines, "\n"), "38;2;214;92;255") {
		t.Fatalf("styled frame should use configured primary, got %#v", frame.ANSILines)
	}
}

func TestRenderVMBuilderCarriesUserConfiguredHeaderChrome(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	root.Config = state.TUIConfigStore{Chrome: state.TUIChromeConfig{
		TabCreateIcon: "+",
		TabTemplate:   "{{tab_id}} {{if active}}▎{{else}}|{{end}} {{title | truncate 8}} [action:tab.close]{{close_icon}}[/action]",
	}}
	vm := NewRenderVMBuilder().Build(root)
	if vm.Shell.Header.TabCreateIcon != "+" || !strings.Contains(vm.Shell.Header.TabTemplate, "{{tab_id}}") {
		t.Fatalf("header chrome config should be carried into VM, header=%#v", vm.Shell.Header)
	}
	frame := NewRenderer(DefaultTheme()).RenderResult(vm)
	if !strings.Contains(frame.Lines()[0], ""+state.DefaultTabID) || !strings.Contains(frame.Lines()[0], "+") {
		t.Fatalf("configured header template/create icon should render in top bar, got %#v", frame.Lines()[0])
	}
}

func plainLines(lines []Line) string {
	values := make([]string, len(lines))
	for i, line := range lines {
		values[i] = line.PlainString()
	}
	return strings.Join(values, "\n")
}

func contentPlainLines(content ContentVM) []string {
	values := make([]string, len(content.Lines))
	for i, line := range content.Lines {
		values[i] = line.PlainString()
	}
	return values
}

package state

import (
	"strings"
	"testing"
	"time"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
)

func TestDefaultShellOwnsWorkbenchTreeAndChromeState(t *testing.T) {
	shell := DefaultShell()

	if shell.Workspace.ID != DefaultWorkspaceID || shell.Workspace.ActiveTabID != DefaultTabID {
		t.Fatalf("unexpected default workspace %#v", shell.Workspace)
	}
	if len(shell.Workspace.Tabs) != 1 || len(shell.Workspace.Tabs[0].Panes) != 1 {
		t.Fatalf("expected one tab and one pane, got %#v", shell.Workspace)
	}
	if shell.ActivePaneID != DefaultPaneID {
		t.Fatalf("unexpected active pane %q", shell.ActivePaneID)
	}
	if shell.PanelPresentation != PanelPresentationCard {
		t.Fatalf("expected card presentation, got %q", shell.PanelPresentation)
	}
	if !shell.HeaderVisible || !shell.FooterVisible {
		t.Fatalf("header/footer must default visible, got header=%v footer=%v", shell.HeaderVisible, shell.FooterVisible)
	}
	if !shell.Workspace.Tabs[0].Panes[0].Active {
		t.Fatalf("default pane must be active %#v", shell.Workspace.Tabs[0].Panes[0])
	}
}

func TestShellPanelPresentationCanToggleBetweenCardAndSplit(t *testing.T) {
	shell := DefaultShell()

	shell = shell.TogglePanelPresentation()
	if shell.PanelPresentation != PanelPresentationSplitLine {
		t.Fatalf("expected split line, got %q", shell.PanelPresentation)
	}
	shell = shell.TogglePanelPresentation()
	if shell.PanelPresentation != PanelPresentationCard {
		t.Fatalf("expected card, got %q", shell.PanelPresentation)
	}
	shell = shell.SetPanelPresentation("invalid")
	if shell.PanelPresentation != PanelPresentationCard {
		t.Fatalf("invalid presentation should be ignored, got %q", shell.PanelPresentation)
	}
}

func TestShellHeaderFooterVisibilityCanBeHiddenAndRestored(t *testing.T) {
	shell := DefaultShell()

	shell = shell.SetHeaderVisible(false).SetFooterVisible(false)
	if shell.HeaderVisible || shell.FooterVisible {
		t.Fatalf("expected hidden header/footer, got header=%v footer=%v", shell.HeaderVisible, shell.FooterVisible)
	}
	shell = shell.ToggleHeaderVisible().ToggleFooterVisible()
	if !shell.HeaderVisible || !shell.FooterVisible {
		t.Fatalf("expected restored header/footer, got header=%v footer=%v", shell.HeaderVisible, shell.FooterVisible)
	}
}

func TestShellWorkbenchTabCommandsManageActiveTab(t *testing.T) {
	shell := DefaultShell()

	var result WorkbenchCommandResult
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandTabCreate, Name: "build"})
	if result.Status != WorkbenchCommandOK || result.ID == "" {
		t.Fatalf("expected tab create ok, result=%#v shell=%#v", result, shell)
	}
	if len(shell.Workspace.Tabs) != 2 || shell.Workspace.ActiveTabID != result.ID || shell.ActivePaneID == "" {
		t.Fatalf("expected new unconnected tab active, result=%#v shell=%#v", result, shell)
	}
	if active := shell.activeTab(); len(active.Panes) != 1 || active.Panes[0].Kind != PaneEmpty || active.Panes[0].Title != "unconnected" || active.ActivePaneID != active.Panes[0].ID || active.RootSplit.PaneID != active.Panes[0].ID {
		t.Fatalf("new tab must create an unconnected pane, got %#v", active)
	}
	buildTabID := result.ID
	buildPaneID := shell.ActivePaneID
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandTabRename, Name: "构建🚀"})
	if result.Status != WorkbenchCommandOK || shell.activeTab().Title != "构建🚀" {
		t.Fatalf("expected tab rename, result=%#v tab=%#v", result, shell.activeTab())
	}
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandTabPrevious})
	if result.Status != WorkbenchCommandOK || shell.Workspace.ActiveTabID != DefaultTabID || shell.ActivePaneID != DefaultPaneID {
		t.Fatalf("expected previous tab active, result=%#v shell=%#v", result, shell)
	}
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandTabSwitch, TargetID: buildTabID})
	if result.Status != WorkbenchCommandOK || shell.Workspace.ActiveTabID != buildTabID || shell.ActivePaneID != buildPaneID {
		t.Fatalf("expected switch tab active, result=%#v shell=%#v", result, shell)
	}
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandTabSwitch, TargetID: "missing-tab"})
	if result.Status != WorkbenchCommandInvalid || shell.Workspace.ActiveTabID != buildTabID {
		t.Fatalf("missing tab switch must be rejected without changing active tab, result=%#v shell=%#v", result, shell)
	}
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandTabSwitch, TargetID: DefaultTabID})
	if result.Status != WorkbenchCommandOK || shell.Workspace.ActiveTabID != DefaultTabID {
		t.Fatalf("expected switch back to main tab, result=%#v shell=%#v", result, shell)
	}
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandTabClose})
	if result.Status != WorkbenchCommandOK || len(shell.Workspace.Tabs) != 1 || shell.Workspace.ActiveTabID != buildTabID {
		t.Fatalf("expected close current tab and keep remaining active, result=%#v shell=%#v", result, shell)
	}
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandTabClose})
	if result.Status != WorkbenchCommandOK || len(shell.Workspace.Tabs) != 0 || shell.Workspace.ActiveTabID != "" || shell.ActivePaneID != "" {
		t.Fatalf("last tab close must leave an empty workspace, result=%#v shell=%#v", result, shell)
	}
	shell = shell.EnsureDefaults()
	if len(shell.Workspace.Tabs) != 0 || shell.Workspace.ActiveTabID != "" || shell.ActivePaneID != "" {
		t.Fatalf("empty workspace must survive defaults, got %#v", shell)
	}
}

func TestShellWorkbenchTabCommandsRespectTargetWorkspace(t *testing.T) {
	shell := DefaultShell()

	var result WorkbenchCommandResult
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandWorkspaceCreate, Name: "remote"})
	if result.Status != WorkbenchCommandOK {
		t.Fatalf("expected workspace create, result=%#v", result)
	}
	remoteID := result.ID
	remotePaneID := shell.ActivePaneID
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandWorkspaceSwitch, TargetID: DefaultWorkspaceID})
	if result.Status != WorkbenchCommandOK {
		t.Fatalf("expected switch to default workspace, result=%#v", result)
	}

	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{
		Action: WorkbenchCommandTabCreate,
		Target: PaneCommandTarget{WorkspaceID: remoteID},
		Name:   "remote-build",
	})
	if result.Status != WorkbenchCommandOK || shell.Workspace.ID != remoteID {
		t.Fatalf("targeted tab create should switch and create in remote workspace, result=%#v shell=%#v", result, shell)
	}
	remoteTabID := result.ID
	if shell.Workspace.Tabs[1].Title != "remote-build" {
		t.Fatalf("expected remote tab created, workspace=%#v", shell.Workspace)
	}

	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandWorkspaceSwitch, TargetID: DefaultWorkspaceID})
	if result.Status != WorkbenchCommandOK {
		t.Fatalf("expected switch to default workspace, result=%#v", result)
	}
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{
		Action:   WorkbenchCommandTabRename,
		TargetID: DefaultTabID,
		Target:   PaneCommandTarget{WorkspaceID: remoteID},
		Name:     "remote-main",
	})
	if result.Status != WorkbenchCommandOK || shell.Workspace.ID != remoteID || shell.Workspace.Tabs[0].Title != "remote-main" {
		t.Fatalf("targeted tab rename should rename remote workspace tab, result=%#v shell=%#v", result, shell)
	}
	defaultWorkspace, ok := workspaceByID(shell.Workspaces, DefaultWorkspaceID)
	if !ok || defaultWorkspace.Tabs[0].Title != "main" {
		t.Fatalf("targeted tab rename must not change default workspace, ok=%v workspace=%#v", ok, defaultWorkspace)
	}

	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandWorkspaceSwitch, TargetID: DefaultWorkspaceID})
	if result.Status != WorkbenchCommandOK {
		t.Fatalf("expected switch to default workspace, result=%#v", result)
	}
	shell = shell.FocusPane(PaneCommandTarget{WorkspaceID: remoteID, TabID: DefaultTabID, PaneID: remotePaneID})
	if shell.Workspace.ID != remoteID || shell.ActivePaneID != remotePaneID {
		t.Fatalf("targeted focus should switch to remote pane, shell=%#v", shell)
	}

	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandWorkspaceSwitch, TargetID: DefaultWorkspaceID})
	if result.Status != WorkbenchCommandOK {
		t.Fatalf("expected switch to default workspace, result=%#v", result)
	}
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{
		Action:   WorkbenchCommandTabClose,
		TargetID: remoteTabID,
		Target:   PaneCommandTarget{WorkspaceID: remoteID},
	})
	if result.Status != WorkbenchCommandOK || shell.Workspace.ID != remoteID || len(shell.Workspace.Tabs) != 1 || shell.Workspace.Tabs[0].Title != "remote-main" {
		t.Fatalf("targeted tab close should close remote tab only, result=%#v shell=%#v", result, shell)
	}
	defaultWorkspace, ok = workspaceByID(shell.Workspaces, DefaultWorkspaceID)
	if !ok || len(defaultWorkspace.Tabs) != 1 || defaultWorkspace.Tabs[0].Title != "main" {
		t.Fatalf("targeted tab close must not change default workspace, ok=%v workspace=%#v", ok, defaultWorkspace)
	}
}

func TestShellEmptyWorkspaceCreatesTabForTerminalAttach(t *testing.T) {
	shell := DefaultShell()
	var result WorkbenchCommandResult
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandTabClose})
	if result.Status != WorkbenchCommandOK || len(shell.Workspace.Tabs) != 0 {
		t.Fatalf("expected empty workspace, result=%#v shell=%#v", result, shell)
	}

	shell = shell.EnsureActiveTabForAttach().BindPaneTerminal(PaneCommandTarget{}, "term-main")
	if len(shell.Workspace.Tabs) != 1 || shell.Workspace.ActiveTabID == "" || shell.ActivePaneID == "" {
		t.Fatalf("attach must create a real tab and pane, got %#v", shell)
	}
	pane, ok := shell.Pane(PaneCommandTarget{PaneID: shell.ActivePaneID})
	if !ok || pane.TerminalID != "term-main" || pane.Kind != PaneTerminalLive {
		t.Fatalf("expected terminal-bound first pane, pane=%#v ok=%v shell=%#v", pane, ok, shell)
	}
}

func TestShellWorkbenchWorkspaceCommandsSwitchAndRename(t *testing.T) {
	shell := DefaultShell()

	var result WorkbenchCommandResult
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandWorkspaceCreate, Name: "work-2"})
	if result.Status != WorkbenchCommandOK || result.ID == "" {
		t.Fatalf("expected workspace create ok, result=%#v shell=%#v", result, shell)
	}
	if shell.Workspace.ID != result.ID || shell.Workspace.Name != "work-2" || len(shell.Workspaces) != 2 {
		t.Fatalf("expected new workspace active and stored, result=%#v shell=%#v", result, shell)
	}
	if len(shell.Workspace.Tabs) != 1 || shell.Workspace.ActiveTabID != DefaultTabID || shell.ActivePaneID == "" {
		t.Fatalf("new workspace must create one active tab and pane slot, result=%#v shell=%#v", result, shell)
	}
	activeTab := shell.Workspace.Tabs[0]
	if len(activeTab.Panes) != 1 || activeTab.RootSplit.PaneID != activeTab.Panes[0].ID {
		t.Fatalf("new workspace must create one fullscreen pane slot, tab=%#v shell=%#v", activeTab, shell)
	}
	if pane := activeTab.Panes[0]; pane.Kind != PaneEmpty || pane.TerminalID != "" || pane.Title != "unconnected" || !pane.Active {
		t.Fatalf("new workspace pane must be unconnected without terminal, pane=%#v shell=%#v", pane, shell)
	}
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandWorkspaceRename, Name: "云端🚀"})
	if result.Status != WorkbenchCommandOK || shell.Workspace.Name != "云端🚀" {
		t.Fatalf("expected active workspace rename, result=%#v shell=%#v", result, shell)
	}
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandWorkspacePrevious})
	if result.Status != WorkbenchCommandOK || shell.Workspace.ID != DefaultWorkspaceID || shell.ActivePaneID != DefaultPaneID {
		t.Fatalf("expected previous workspace active, result=%#v shell=%#v", result, shell)
	}
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandWorkspaceNext})
	if result.Status != WorkbenchCommandOK || shell.Workspace.Name != "云端🚀" {
		t.Fatalf("expected next workspace to restore renamed workspace, result=%#v shell=%#v", result, shell)
	}

	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandWorkspaceSwitch, TargetID: DefaultWorkspaceID})
	if result.Status != WorkbenchCommandOK || shell.Workspace.ID != DefaultWorkspaceID {
		t.Fatalf("expected explicit workspace switch, result=%#v shell=%#v", result, shell)
	}
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandWorkspaceDelete, TargetID: "workspace-2"})
	if result.Status != WorkbenchCommandNeedsConfirmation || len(shell.Workspaces) != 2 {
		t.Fatalf("workspace delete must require confirmation, result=%#v shell=%#v", result, shell)
	}
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandWorkspaceDelete, TargetID: "workspace-2", Confirm: PaneConfirmAccepted})
	if result.Status != WorkbenchCommandOK || len(shell.Workspaces) != 1 || shell.Workspaces[0].ID != DefaultWorkspaceID {
		t.Fatalf("expected confirmed workspace delete, result=%#v shell=%#v", result, shell)
	}

	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandWorkspaceCreate, Name: "ephemeral"})
	if result.Status != WorkbenchCommandOK || shell.Workspace.ID == DefaultWorkspaceID {
		t.Fatalf("expected new active workspace for active-delete regression, result=%#v shell=%#v", result, shell)
	}
	activeID := shell.Workspace.ID
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandWorkspaceDelete, TargetID: activeID, Confirm: PaneConfirmAccepted})
	if result.Status != WorkbenchCommandOK || result.ID != activeID {
		t.Fatalf("expected active workspace delete ok, result=%#v shell=%#v", result, shell)
	}
	if shell.Workspace.ID == activeID || workspaceIndexByID(shell.Workspaces, activeID) >= 0 || len(shell.Workspaces) != 1 {
		t.Fatalf("active workspace delete must not reinsert deleted workspace, active=%q shell=%#v", activeID, shell)
	}
}

func TestShellWorkbenchTabKillAndPaneCRUDDistinguishTerminalLifecycle(t *testing.T) {
	shell := DefaultShell().
		SplitActivePane(PaneState{ID: "pane-logs", Title: "logs", Kind: PaneTerminalLive, TerminalID: "term-logs"}, SplitDirectionVertical).
		FocusPane(PaneCommandTarget{PaneID: "pane-logs"})

	var result WorkbenchCommandResult
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandPaneDetach, Target: PaneCommandTarget{PaneID: "pane-logs"}})
	pane, _ := shell.Pane(PaneCommandTarget{PaneID: "pane-logs"})
	if result.Status != WorkbenchCommandOK || pane.Kind != PaneEmpty || pane.TerminalID != "" {
		t.Fatalf("pane detach should keep pane and drop terminal binding only, result=%#v pane=%#v", result, pane)
	}
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandPaneRename, Target: PaneCommandTarget{PaneID: "pane-logs"}, Name: "日志"})
	pane, _ = shell.Pane(PaneCommandTarget{PaneID: "pane-logs"})
	if result.Status != WorkbenchCommandOK || pane.Title != "日志" {
		t.Fatalf("pane rename should update workbench schema, result=%#v pane=%#v", result, pane)
	}

	shell = shell.BindPaneTerminal(PaneCommandTarget{PaneID: "pane-logs"}, "term-logs")
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandPaneKill, Target: PaneCommandTarget{PaneID: "pane-logs"}})
	if result.Status != WorkbenchCommandNeedsConfirmation || shell.ActivePaneID != "pane-logs" {
		t.Fatalf("pane kill must require confirmation without mutation, result=%#v shell=%#v", result, shell)
	}
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandPaneKill, Target: PaneCommandTarget{PaneID: "pane-logs"}, Confirm: PaneConfirmAccepted})
	if result.Status != WorkbenchCommandOK || result.ID != "pane-logs" || len(result.Killed) != 1 || result.Killed[0] != "term-logs" || shell.HasPane(PaneCommandTarget{PaneID: "pane-logs"}) {
		t.Fatalf("pane kill should close pane and report terminal kill boundary, result=%#v shell=%#v", result, shell)
	}

	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandTabCreate, TargetID: "tab-logs", Name: "logs"})
	if result.Status != WorkbenchCommandOK {
		t.Fatalf("expected tab create, result=%#v", result)
	}
	shell = shell.BindPaneTerminal(PaneCommandTarget{TabID: "tab-logs"}, "term-tab")
	shell, result = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandTabKill, TargetID: "tab-logs", Confirm: PaneConfirmAccepted})
	if result.Status != WorkbenchCommandOK || result.ID != "tab-logs" || len(result.Killed) != 1 || result.Killed[0] != "term-tab" || shell.Workspace.ActiveTabID == "tab-logs" {
		t.Fatalf("tab kill should close tab and report terminal ids, result=%#v shell=%#v", result, shell)
	}
}

func TestShellToastLifecycle(t *testing.T) {
	shell := DefaultShell().
		AddToast(ToastSpec{ID: "keep", Severity: ToastInfo, Title: "sync"}).
		AddToast(ToastSpec{ID: "drop", Severity: ToastWarning, Title: "warn", DismissAfterTicks: 2})

	shell = shell.TickToasts(1)
	if len(shell.Toasts) != 2 || shell.Toasts[1].AgeTicks != 1 {
		t.Fatalf("expected both toasts after one tick, got %#v", shell.Toasts)
	}
	shell = shell.TickToasts(1)
	if len(shell.Toasts) != 1 || shell.Toasts[0].ID != "keep" {
		t.Fatalf("expected auto-dismissed toast, got %#v", shell.Toasts)
	}
	shell = shell.AddToast(ToastSpec{ID: "close", Severity: ToastSuccess, Title: "done"})
	shell = shell.CloseCurrentToast()
	if len(shell.Toasts) != 1 || shell.Toasts[0].ID != "keep" {
		t.Fatalf("expected close current to remove latest toast, got %#v", shell.Toasts)
	}
	shell = shell.ClearToasts()
	if len(shell.Toasts) != 0 {
		t.Fatalf("expected clear all toasts, got %#v", shell.Toasts)
	}
}

func TestShellToastDefaultDismissTicks(t *testing.T) {
	shell := DefaultShell().
		AddToast(ToastSpec{ID: "info", Severity: ToastInfo, Title: "info"}).
		AddToast(ToastSpec{ID: "error", Severity: ToastError, Title: "error"}).
		AddToast(ToastSpec{ID: "pending", Severity: ToastWarning, Title: "pending", Pending: true})

	if shell.Toasts[0].DismissAfterTicks != defaultToastDismissTicks {
		t.Fatalf("expected info default dismiss ticks, got %#v", shell.Toasts[0])
	}
	if shell.Toasts[1].DismissAfterTicks != attentionToastDismissTicks {
		t.Fatalf("expected error default dismiss ticks, got %#v", shell.Toasts[1])
	}
	if shell.Toasts[2].DismissAfterTicks != pendingToastDismissTicks {
		t.Fatalf("expected pending default dismiss ticks, got %#v", shell.Toasts[2])
	}

	shell = shell.TickToasts(defaultToastDismissTicks)
	if len(shell.Toasts) != 2 || shell.Toasts[0].ID != "error" || shell.Toasts[1].ID != "pending" {
		t.Fatalf("expected short info toast to disappear first, got %#v", shell.Toasts)
	}
	shell = shell.TickToasts(attentionToastDismissTicks - defaultToastDismissTicks)
	if len(shell.Toasts) != 1 || shell.Toasts[0].ID != "pending" {
		t.Fatalf("expected error toast to disappear before pending, got %#v", shell.Toasts)
	}
	shell = shell.TickToasts(pendingToastDismissTicks - attentionToastDismissTicks)
	if len(shell.Toasts) != 0 {
		t.Fatalf("expected pending toast to have explicit lifecycle, got %#v", shell.Toasts)
	}
}

func TestShellToastDeduplicatesByVisibleMessage(t *testing.T) {
	shell := DefaultShell().
		AddToast(ToastSpec{ID: "first", Severity: ToastInfo, Title: "floating.move", Body: "floating-1"}).
		TickToasts(2).
		AddToast(ToastSpec{Severity: ToastInfo, Title: "floating.move", Body: "floating-1"})

	if len(shell.Toasts) != 1 {
		t.Fatalf("expected duplicate toast to be refreshed, got %#v", shell.Toasts)
	}
	if shell.Toasts[0].ID != "first" || shell.Toasts[0].AgeTicks != 0 {
		t.Fatalf("expected refreshed duplicate to keep stable id and reset age, got %#v", shell.Toasts[0])
	}

	shell = shell.
		AddToast(ToastSpec{Severity: ToastWarning, Title: "floating.move", Body: "blocked"}).
		AddToast(ToastSpec{Severity: ToastWarning, Title: "floating.move", Body: "blocked"})
	if len(shell.Toasts) != 2 || shell.Toasts[1].AgeTicks != 0 {
		t.Fatalf("expected severity/body scoped dedupe without losing distinct messages, got %#v", shell.Toasts)
	}
}

func TestShellToastAutoIDAndDefaultSeverity(t *testing.T) {
	shell := DefaultShell().AddToast(ToastSpec{Title: "hello"})
	if len(shell.Toasts) != 1 {
		t.Fatalf("expected one toast, got %#v", shell.Toasts)
	}
	if shell.Toasts[0].ID != "toast-1" || shell.Toasts[0].Severity != ToastInfo {
		t.Fatalf("unexpected generated toast %#v", shell.Toasts[0])
	}
}

func TestShellTerminalPickerOverlayTargetsActivePane(t *testing.T) {
	shell := DefaultShell().OpenTerminalPicker()

	if !shell.Overlay.Open || shell.Overlay.Kind != OverlayTerminalPicker {
		t.Fatalf("expected terminal picker overlay, got %#v", shell.Overlay)
	}
	if shell.Overlay.TargetID != DefaultPaneID {
		t.Fatalf("expected overlay target active pane, got %#v", shell.Overlay)
	}
	shell = shell.CloseOverlay()
	if shell.Overlay.Open || shell.Overlay.Kind != OverlayNone {
		t.Fatalf("expected closed overlay, got %#v", shell.Overlay)
	}
}

func TestTerminalPickerStateFiltersAndMovesSelection(t *testing.T) {
	shell := DefaultShell().
		SplitActivePane(PaneState{ID: "pane-2", Title: "日志🚀", Kind: PaneTerminalLive, TerminalID: "term-2"}, SplitDirectionVertical).
		FocusPane(PaneCommandTarget{PaneID: DefaultPaneID}).
		OpenTerminalPicker()
	root := Root{
		Shell:        shell,
		TerminalPool: TerminalPoolStore{Items: []TerminalPoolItem{{TerminalID: "term-main", Title: "term-main", State: "running"}}},
	}

	items := TerminalPickerItems(root)
	if len(items) != 2 || !items[0].Selected || !items[0].CreateNew || items[1].TerminalID != "term-main" || items[1].PaneID != "" {
		t.Fatalf("expected create row plus terminal row, got %#v", items)
	}
	root.Shell = root.Shell.SetTerminalPickerQuery("日志")
	items = TerminalPickerItems(root)
	if len(items) != 0 {
		t.Fatalf("query should filter terminal-only rows, got %#v", items)
	}
	root.Shell = root.Shell.SetTerminalPickerQuery("")
	root.Shell = root.Shell.MoveTerminalPickerSelection(1, len(TerminalPickerItems(root)))
	items = TerminalPickerItems(root)
	if len(items) != 2 || !items[1].Selected || items[1].TerminalID != "term-main" {
		t.Fatalf("selection should move to terminal row, got %#v", items)
	}
}

func TestTerminalPickerItemsDoNotAppendStaleLocalBindings(t *testing.T) {
	root := Root{
		Shell: DefaultShell().OpenTerminalPicker(),
		TerminalPool: TerminalPoolStore{Items: []TerminalPoolItem{
			{EndpointID: "cn-fast", TerminalID: "111", Title: "111", State: "running"},
			{EndpointID: DefaultEndpointID, TerminalID: "123", Title: "123", State: "running"},
		}},
		TerminalViews: TerminalViewStore{}.BindPane(NewEndpointPaneTerminalView("cn-fast", "pane-old", "term-pool-legacy", 7, 218, 94, TerminalResizeRoleOwner, "surface-old", TerminalPaneViewID("pane-old"), true)),
		Session:       TerminalSessionStore{EndpointID: "cn-fast", TerminalID: "term-pool-legacy", Attached: true, Cols: 218, Rows: 94, State: TerminalLiveAttached},
	}

	items := TerminalPickerItems(root)
	if len(items) != 3 || !items[0].CreateNew || items[1].TerminalID != "111" || items[2].TerminalID != "123" {
		t.Fatalf("picker must only show daemon-listed terminals plus create row, got %#v", items)
	}
	for _, item := range items {
		if item.TerminalID == "term-pool-legacy" {
			t.Fatalf("stale binding/session must not create picker terminal row, got %#v", items)
		}
	}
}

func TestTerminalPickerItemsShowOnlyTerminalPoolInfo(t *testing.T) {
	shell := DefaultShell().OpenTerminalPicker()
	shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-main"
	root := Root{
		Shell: shell,
		TerminalPool: TerminalPoolStore{
			Status: TerminalPoolReady,
			Items: []TerminalPoolItem{
				{TerminalID: "term-main", Title: "main", State: "running", Cols: 80, Rows: 24},
				{TerminalID: "term-pool", Title: "远程🚀", State: "running", Cols: 100, Rows: 30},
			},
		},
	}

	items := TerminalPickerItems(root)
	if len(items) != 3 || !items[0].CreateNew || items[1].PaneID != "" || items[1].Location != "" || items[1].CreateNew || !items[1].FromPool || items[2].TerminalID != "term-pool" || items[2].Cols != 100 || items[2].Rows != 30 {
		t.Fatalf("expected create row plus terminal-only pool rows, got %#v", items)
	}
	root.Shell = root.Shell.SetTerminalPickerQuery("远程")
	items = TerminalPickerItems(root)
	if len(items) != 1 || !items[0].Selected || items[0].TerminalID != "term-pool" {
		t.Fatalf("expected query to match pool terminal item, got %#v", items)
	}
	root.Shell = root.Shell.SetTerminalPickerQuery("trpl")
	items = TerminalPickerItems(root)
	if len(items) != 1 || !items[0].Selected || items[0].TerminalID != "term-pool" {
		t.Fatalf("expected fuzzy query to match term-pool, got %#v", items)
	}
	if got := TerminalPickerQueryMatchIndexes("term-pool", "trpl"); len(got) != 4 || got[0] != 0 || got[1] != 2 || got[2] != 5 || got[3] != 8 {
		t.Fatalf("unexpected fuzzy match indexes for term-pool/trpl: %#v", got)
	}
}

func TestTerminalPickerAndManagerFilterTerminalTags(t *testing.T) {
	root := Root{
		Shell: DefaultShell().OpenTerminalPicker(),
		TerminalPool: TerminalPoolStore{Status: TerminalPoolReady, Items: []TerminalPoolItem{
			{TerminalID: "task-tests", Title: "tests", State: "running", Tags: map[string]string{
				"task": "unit-tests", "kind": "task", "run": "fix-login", "agent": "codex",
			}},
			{TerminalID: "shell", Title: "shell", State: "running", Tags: map[string]string{"kind": "shell"}},
		}},
	}

	root.Shell = root.Shell.SetTerminalPickerQuery("kind=task")
	picker := TerminalPickerItems(root)
	if len(picker) != 1 || picker[0].TerminalID != "task-tests" || picker[0].Tags["run"] != "fix-login" {
		t.Fatalf("picker should project and filter canonical key=value tags, got %#v", picker)
	}

	root.Shell = root.Shell.OpenTerminalPool().SetTerminalPoolQuery("run=fix-login")
	manager := TerminalPoolPageItems(root)
	if len(manager) != 1 || manager[0].TerminalID != "task-tests" {
		t.Fatalf("manager should filter canonical key=value tags, got %#v", manager)
	}
	if got := TerminalTagsText(manager[0].Tags); got != "agent=codex, kind=task, run=fix-login, task=unit-tests" {
		t.Fatalf("tag text should be stable and sorted, got %q", got)
	}
}

func TestTerminalPickerItemsKeepSameTerminalIDAcrossEndpoints(t *testing.T) {
	root := Root{
		Shell: DefaultShell().OpenTerminalPicker(),
		TerminalPool: TerminalPoolStore{
			Status: TerminalPoolReady,
			Items: []TerminalPoolItem{
				{EndpointID: DefaultEndpointID, TerminalID: "term-1", Title: "local shell"},
				{EndpointID: "west", TerminalID: "term-1", Title: "west shell"},
			},
		},
	}

	items := TerminalPickerItems(root)
	if len(items) != 3 {
		t.Fatalf("expected create row plus both endpoint terminal rows, got %#v", items)
	}
	if items[1].EndpointID != DefaultEndpointID || items[1].TerminalID != "term-1" || items[2].EndpointID != "west" || items[2].TerminalID != "term-1" {
		t.Fatalf("terminal picker must keep endpoint identity for duplicate terminal ids, got %#v", items)
	}
}

func TestTerminalPickerItemsUseCurrentViewBindingForAttachedState(t *testing.T) {
	root := Root{
		Shell:         DefaultShell().OpenTerminalPicker(),
		TerminalViews: TerminalViewStore{}.BindPane(NewEndpointPaneTerminalView(DefaultEndpointID, "pane-local", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface-local", TerminalPaneViewID("pane-local"), true)),
		TerminalPool: TerminalPoolStore{
			Status: TerminalPoolReady,
			Items: []TerminalPoolItem{
				{EndpointID: DefaultEndpointID, TerminalID: "term-1", Title: "local shell", State: "running"},
				{EndpointID: "west", TerminalID: "term-1", Title: "west shell", State: "attached", Attached: true, AttachmentCount: 2},
			},
		},
	}

	items := TerminalPickerItems(root)
	if len(items) != 3 {
		t.Fatalf("expected create row plus local and west rows, got %#v", items)
	}
	if !items[1].Active || items[1].PoolState != string(TerminalLiveAttached) {
		t.Fatalf("local exact binding should be projected as attached, got %#v", items[1])
	}
	if items[2].Active || items[2].PoolState != "running" {
		t.Fatalf("remote daemon attached metadata must not become current TUI attached state, got %#v", items[2])
	}

	root.Shell = root.Shell.SetTerminalPickerQuery("attached")
	items = TerminalPickerItems(root)
	if len(items) != 1 || items[0].EndpointID != DefaultEndpointID || items[0].TerminalID != "term-1" {
		t.Fatalf("attached search should only match current TUI exact binding, got %#v", items)
	}

	root.Shell = root.Shell.SetTerminalPickerQuery("")
	root.TerminalViews = root.TerminalViews.BindPane(NewEndpointPaneTerminalView("west", "pane-west", "term-1", 8, 100, 30, TerminalResizeRoleFollower, "surface-west", TerminalPaneViewID("pane-west"), false))
	items = TerminalPickerItems(root)
	if len(items) != 3 || !items[2].Active || items[2].PoolState != string(TerminalLiveAttached) {
		t.Fatalf("west exact binding should be projected as attached only after this TUI attaches it, got %#v", items)
	}
}

func TestTerminalPickerAndPoolRowsSortByName(t *testing.T) {
	root := Root{
		Shell: DefaultShell().OpenTerminalPicker(),
		Endpoints: (EndpointStore{}).
			Upsert(EndpointItem{ID: DefaultEndpointID, Label: "This Mac", Transport: EndpointTransportLocal, ConnectMode: EndpointConnectAuto, Enabled: true}).
			Upsert(EndpointItem{ID: "west", Label: "US West", Transport: EndpointTransportSSH, ConnectMode: EndpointConnectOnDemand, Enabled: true}),
		TerminalPool: TerminalPoolStore{
			Status: TerminalPoolReady,
			Items: []TerminalPoolItem{
				{EndpointID: "west", TerminalID: "term-z", Title: "zeta"},
				{EndpointID: DefaultEndpointID, TerminalID: "term-beta", Title: "Beta"},
				{EndpointID: "west", TerminalID: "term-alpha", Title: "alpha"},
			},
		},
	}

	picker := TerminalPickerItems(root)
	if len(picker) != 4 || !picker[0].CreateNew || picker[1].Title != "alpha" || picker[2].Title != "Beta" || picker[3].Title != "zeta" {
		t.Fatalf("terminal picker rows should sort by display name after create row, got %#v", picker)
	}

	root.Shell = root.Shell.OpenTerminalPool()
	pool := TerminalPoolPageItems(root)
	if len(pool) != 3 || pool[0].Title != "alpha" || pool[1].Title != "Beta" || pool[2].Title != "zeta" {
		t.Fatalf("terminal manager rows should sort by display name, got %#v", pool)
	}
}

func TestTerminalPickerCreateRowIsGlobalButEndpointSearchable(t *testing.T) {
	root := Root{
		Shell: DefaultShell().OpenTerminalPicker().SetTerminalPickerQuery("us west"),
		Endpoints: (EndpointStore{}).
			Upsert(EndpointItem{ID: DefaultEndpointID, Label: "This Mac", Transport: EndpointTransportLocal, ConnectMode: EndpointConnectAuto, Enabled: true}).
			Upsert(EndpointItem{ID: "us-west", Label: "US West", Transport: EndpointTransportSSH, ConnectMode: EndpointConnectOnDemand, Enabled: true}),
	}

	picker := TerminalPickerItems(root)
	if len(picker) != 1 || !picker[0].CreateNew || picker[0].EndpointID != DefaultEndpointID || picker[0].EndpointLabel != "This Mac" || !strings.Contains(picker[0].EndpointSearchText, "US West") {
		t.Fatalf("endpoint query should expose one global create row, got %#v", picker)
	}
}

func TestTerminalCreateEndpointItemsUseAvailableEndpoints(t *testing.T) {
	root := Root{
		Shell: DefaultShell().OpenTerminalPicker(),
		Endpoints: (EndpointStore{}).
			Upsert(EndpointItem{ID: DefaultEndpointID, Label: "This Mac", Transport: EndpointTransportLocal, ConnectMode: EndpointConnectAuto, Enabled: true, Status: EndpointStatusConnected}).
			Upsert(EndpointItem{ID: "disabled", Label: "Disabled Box", Transport: EndpointTransportSSH, ConnectMode: EndpointConnectAuto, Enabled: false}).
			Upsert(EndpointItem{ID: "manual", Label: "Manual Box", Transport: EndpointTransportSSH, ConnectMode: EndpointConnectManual, Enabled: true}).
			Upsert(EndpointItem{ID: "reconnect", Label: "Moved Box", Transport: EndpointTransportSSH, ConnectMode: EndpointConnectOnDemand, Enabled: true, ReconnectRequired: true}).
			Upsert(EndpointItem{ID: "west", Label: "US West", Transport: EndpointTransportSSH, ConnectMode: EndpointConnectOnDemand, Enabled: true, Status: EndpointStatusOffline}),
	}

	items := TerminalCreateEndpointItems(root)
	if len(items) != 2 || items[0].DisplayLabel() != "This Mac" || items[1].DisplayLabel() != "US West" {
		t.Fatalf("create endpoint choices should only use available endpoints, got %#v", items)
	}
}

func TestEndpointStoreRegistryReloadClassifiesRuntimeDisplayState(t *testing.T) {
	registry, err := (endpointdomain.Registry{Default: endpointdomain.DefaultEndpointID, Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{
		endpointdomain.DefaultEndpointID: endpointdomain.NewLocalEndpoint(endpointdomain.DefaultEndpointID, "This Mac", "auto", endpointdomain.ConnectAuto),
		"west":                           endpointdomain.NewSSHEndpoint("west", "US West", "root@155.94.155.192", "ssh:west", "127.0.0.1:41120", "127.0.0.1:41121", endpointdomain.ConnectOnDemand),
	}}).Normalize()
	if err != nil {
		t.Fatalf("normalize registry: %v", err)
	}
	store := (EndpointStore{}).ApplyConnectionRegistry(registry)
	store = store.MarkTerminalListResult("west", 0, "ssh timeout")
	store = store.ApplyDefaults("west", []string{"/bin/bash", "-l"}, "/srv/app", "")

	if west, ok := store.Endpoint("west"); !ok || west.DisplayStatus() != EndpointStatusOffline || west.LastError != "ssh timeout" || west.LastErrorKind != EndpointErrorUnavailable {
		t.Fatalf("west endpoint should be offline only in endpoint store, got %#v ok=%v", west, ok)
	}

	renamed := cloneStateTestRegistry(registry)
	cfg := renamed.Endpoints["west"]
	cfg.Label = "West Renamed"
	renamed.Endpoints["west"] = cfg
	store = store.ApplyConnectionRegistry(renamed)
	if west, _ := store.Endpoint("west"); west.DisplayLabel() != "West Renamed" || west.DisplayStatus() != EndpointStatusOffline || west.ReconnectRequired || !west.DefaultsLoaded || west.DefaultCWD != "/srv/app" || strings.Join(west.DefaultCommand, " ") != "/bin/bash -l" {
		t.Fatalf("label reload should update display only, got %#v", west)
	}

	moved := cloneStateTestRegistry(renamed)
	cfg = moved.Endpoints["west"]
	route := cfg.Routes["ssh"]
	route.Host = "root@155.94.155.193"
	cfg.Routes["ssh"] = route
	moved.Endpoints["west"] = cfg
	store = store.ApplyConnectionRegistry(moved)
	if west, _ := store.Endpoint("west"); west.DisplayStatus() != EndpointStatusReconnectRequired {
		t.Fatalf("address reload should mark reconnect required, got %#v", west)
	}

	deleted := moved
	delete(deleted.Endpoints, "west")
	store = store.ApplyConnectionRegistry(deleted)
	if west, ok := store.Endpoint("west"); !ok || west.DisplayStatus() != EndpointStatusUnregistered {
		t.Fatalf("deleted active endpoint should remain unregistered, got %#v ok=%v", west, ok)
	}
}

func TestEndpointStoreHubIdentityReloadRequiresReconnect(t *testing.T) {
	registry, err := (endpointdomain.Registry{Default: "studio", Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{
		"studio": endpointdomain.NewManagedEndpoint("studio", "Studio Mac", endpointdomain.DaemonIdentity{DeviceID: "device_ed25519:studio", DeviceFingerprint: "SHA256:studio"}, "device_ed25519:studio", "grant:studio", endpointdomain.RelayAuto, endpointdomain.ConnectOnDemand),
	}}).Normalize()
	if err != nil {
		t.Fatalf("normalize hub registry: %v", err)
	}
	store := (EndpointStore{}).ApplyConnectionRegistry(registry)
	store = store.MarkManagedRoute("studio", "single_relay", "lower_loss")
	studio, ok := store.Endpoint("studio")
	if !ok || studio.Transport != EndpointTransportHubP2P || studio.DeviceID != "device_ed25519:studio" || studio.DeviceFingerprint != "SHA256:studio" || len(studio.Routes) != 1 || studio.Routes[0].DialIdentity.CredentialRef != "grant:studio" || studio.Routes[0].DialIdentity.RelayMode != endpointdomain.RelayAuto {
		t.Fatalf("hub endpoint should project identity fields, got %#v ok=%v", studio, ok)
	}

	renamed := cloneStateTestRegistry(registry)
	cfg := renamed.Endpoints["studio"]
	cfg.Label = "Desk"
	renamed.Endpoints["studio"] = cfg
	store = store.ApplyConnectionRegistry(renamed)
	if studio, _ := store.Endpoint("studio"); studio.DisplayLabel() != "Desk" || studio.ReconnectRequired || studio.ObservedPath != "single_relay" || studio.RouteSelectionReason != "lower_loss" {
		t.Fatalf("hub label change should update display only, got %#v", studio)
	}

	identityChanged := cloneStateTestRegistry(renamed)
	cfg = identityChanged.Endpoints["studio"]
	cfg.DaemonIdentity.DeviceFingerprint = "SHA256:other"
	identityChanged.Endpoints["studio"] = cfg
	store = store.ApplyConnectionRegistry(identityChanged)
	if studio, _ := store.Endpoint("studio"); studio.DisplayStatus() != EndpointStatusReconnectRequired {
		t.Fatalf("hub device fingerprint change should mark reconnect required, got %#v", studio)
	}

	grantChanged := cloneStateTestRegistry(renamed)
	cfg = grantChanged.Endpoints["studio"]
	managedRoute := cfg.Routes["cloud"]
	managedRoute.CredentialRef = "grant:other"
	cfg.Routes["cloud"] = managedRoute
	grantChanged.Endpoints["studio"] = cfg
	store = (EndpointStore{}).ApplyConnectionRegistry(registry).ApplyConnectionRegistry(grantChanged)
	if studio, _ := store.Endpoint("studio"); studio.DisplayStatus() != EndpointStatusReconnectRequired {
		t.Fatalf("hub grant ref change should mark reconnect required, got %#v", studio)
	}
}

func cloneStateTestRegistry(src endpointdomain.Registry) endpointdomain.Registry {
	out := src
	out.Endpoints = map[endpointdomain.EndpointID]endpointdomain.Endpoint{}
	for id, endpoint := range src.Endpoints {
		cloned := endpoint
		cloned.Routes = map[endpointdomain.RouteID]endpointdomain.AccessRoute{}
		for routeID, route := range endpoint.Routes {
			route.HostKeyFingerprints = append([]string(nil), route.HostKeyFingerprints...)
			route.SignalingAddresses = append([]string(nil), route.SignalingAddresses...)
			route.ICETCPAddresses = append([]string(nil), route.ICETCPAddresses...)
			route.AdvertisedAddresses = append([]string(nil), route.AdvertisedAddresses...)
			cloned.Routes[routeID] = route
		}
		out.Endpoints[id] = cloned
	}
	return out
}

func TestClassifyEndpointErrorTextPrefersRemoteDaemonDetail(t *testing.T) {
	text := "ssh transport closed: exit status 255: stdio-proxy connect core-v2 daemon socket: connection refused"
	if got := ClassifyEndpointErrorText(text); got != EndpointErrorRemoteDaemon {
		t.Fatalf("daemon proxy detail should classify as remote-daemon, got %q", got)
	}
}

func TestClassifyEndpointErrorTextRecognizesRouteConfigurationFailure(t *testing.T) {
	if got := ClassifyEndpointErrorText("route kind direct-webrtc-tcp is not connected"); got != EndpointErrorConfig {
		t.Fatalf("route configuration failure should classify as config, got %q", got)
	}
}

func TestEndpointScopedTerminalListFailurePreservesOtherEndpoints(t *testing.T) {
	root := Root{Endpoints: (EndpointStore{}).
		Upsert(EndpointItem{ID: DefaultEndpointID, Label: "This Mac", Transport: EndpointTransportLocal, ConnectMode: EndpointConnectAuto, Enabled: true}).
		Upsert(EndpointItem{ID: "west", Label: "US West", Transport: EndpointTransportSSH, ConnectMode: EndpointConnectOnDemand, Enabled: true})}

	root = root.ApplyEndpointTerminalList(DefaultEndpointID, []TerminalPoolItem{{TerminalID: "term-1", Title: "local"}}, "")
	root = root.ApplyEndpointTerminalList("west", nil, "ssh timeout")

	if len(root.TerminalPool.Items) != 1 || root.TerminalPool.Items[0].EndpointID != DefaultEndpointID || root.TerminalPool.Items[0].TerminalID != "term-1" {
		t.Fatalf("west failure must not clear local terminal list, pool=%#v", root.TerminalPool)
	}
	if root.TerminalPool.Status != TerminalPoolReady || root.TerminalPool.LastError != "" {
		t.Fatalf("partial endpoint failure must not become global pool error, pool=%#v", root.TerminalPool)
	}
	if west, ok := root.Endpoints.Endpoint("west"); !ok || west.DisplayStatus() != EndpointStatusOffline || west.LastError != "ssh timeout" || west.LastErrorKind != EndpointErrorUnavailable {
		t.Fatalf("west endpoint should hold offline state, got %#v ok=%v", west, ok)
	}
	if local, ok := root.Endpoints.Endpoint(DefaultEndpointID); !ok || local.DisplayStatus() != EndpointStatusConnected {
		t.Fatalf("local endpoint should remain connected, got %#v ok=%v", local, ok)
	}
}

func TestTerminalPickerGroupsExposeEndpointMetadataAndSearch(t *testing.T) {
	root := Root{
		Shell: DefaultShell().OpenTerminalPicker(),
		Endpoints: (EndpointStore{}).
			Upsert(EndpointItem{ID: DefaultEndpointID, Label: "This Mac", Transport: EndpointTransportLocal, ConnectMode: EndpointConnectAuto, Enabled: true, Status: EndpointStatusConnected}).
			Upsert(EndpointItem{ID: "disabled", Label: "Disabled Box", Transport: EndpointTransportSSH, ConnectMode: EndpointConnectAuto, Enabled: false}).
			Upsert(EndpointItem{ID: "manual", Label: "Manual Box", Transport: EndpointTransportSSH, ConnectMode: EndpointConnectManual, Enabled: true}).
			Upsert(EndpointItem{ID: "reconnect", Label: "Moved Box", Transport: EndpointTransportSSH, ConnectMode: EndpointConnectOnDemand, Enabled: true, ReconnectRequired: true}).
			Upsert(EndpointItem{ID: "west", Label: "US West", Transport: EndpointTransportHubP2P, ConnectMode: EndpointConnectOnDemand, Enabled: true, Status: EndpointStatusOffline, LastError: "route unavailable", LastErrorKind: EndpointErrorTransportClosed, ConnectionPhase: "failed", ObservedPath: "single_relay", RouteSelectionReason: "lower_loss"}),
		TerminalPool: TerminalPoolStore{
			Status: TerminalPoolReady,
			Items: []TerminalPoolItem{
				{EndpointID: DefaultEndpointID, TerminalID: "term-1", Title: "shell"},
				{EndpointID: "west", TerminalID: "term-1", Title: "shell"},
				{EndpointID: "orphan", TerminalID: "term-9", Title: "orphan shell"},
			},
		},
	}

	groups := TerminalPickerGroups(root)
	if len(groups) != 6 {
		t.Fatalf("expected configured plus unregistered endpoint groups, got %#v", groups)
	}
	statusByID := map[EndpointID]EndpointStatusKind{}
	rowCountByID := map[EndpointID]int{}
	for _, group := range groups {
		statusByID[group.EndpointID] = group.Status
		rowCountByID[group.EndpointID] = len(group.VisibleTerminalRows)
	}
	if statusByID[DefaultEndpointID] != EndpointStatusConnected ||
		statusByID["disabled"] != EndpointStatusDisabled ||
		statusByID["manual"] != EndpointStatusManual ||
		statusByID["reconnect"] != EndpointStatusReconnectRequired ||
		statusByID["west"] != EndpointStatusOffline ||
		statusByID["orphan"] != EndpointStatusUnregistered {
		t.Fatalf("unexpected endpoint group statuses %#v", statusByID)
	}
	for _, group := range groups {
		if group.EndpointID == "west" && group.ErrorKind != EndpointErrorTransportClosed {
			t.Fatalf("west group should carry endpoint error kind, got %#v", group)
		}
		if group.EndpointID == "west" && group.ObservedPath != "single_relay" {
			t.Fatalf("west group should carry managed observed path, got %#v", group)
		}
		if group.EndpointID == "west" && group.RouteSelectionReason != "lower_loss" {
			t.Fatalf("west group should carry managed route reason, got %#v", group)
		}
		if group.EndpointID == "west" && group.ConnectionPhase != "failed" {
			t.Fatalf("west group should carry managed connection phase, got %#v", group)
		}
	}
	if rowCountByID[DefaultEndpointID] != 1 || rowCountByID["west"] != 1 || rowCountByID["orphan"] != 1 || rowCountByID["manual"] != 0 {
		t.Fatalf("unexpected endpoint group terminal rows %#v", rowCountByID)
	}

	root.Shell = root.Shell.SetTerminalPickerQuery("US West")
	items := TerminalPickerItems(root)
	if len(items) != 2 || !items[0].CreateNew || items[1].EndpointID != "west" || items[1].TerminalID != "term-1" {
		t.Fatalf("endpoint label query should match west terminal row, got %#v", items)
	}
}

func TestTerminalPoolPageGroupsExposeEndpointMetadata(t *testing.T) {
	root := Root{
		Shell: DefaultShell().OpenTerminalPool(),
		Endpoints: (EndpointStore{}).
			Upsert(EndpointItem{ID: DefaultEndpointID, Label: "This Mac", Transport: EndpointTransportLocal, ConnectMode: EndpointConnectAuto, Enabled: true, Status: EndpointStatusConnected}).
			Upsert(EndpointItem{ID: "west", Label: "US West", Transport: EndpointTransportSSH, ConnectMode: EndpointConnectManual, Enabled: true}),
		TerminalPool: TerminalPoolStore{Items: []TerminalPoolItem{
			{EndpointID: DefaultEndpointID, TerminalID: "term-1", Title: "shell"},
			{EndpointID: "west", TerminalID: "term-1", Title: "shell"},
		}},
	}

	rows := TerminalPoolPageItems(root)
	if len(rows) != 2 || rows[0].EndpointLabel != "This Mac" || rows[1].EndpointLabel != "US West" || rows[1].EndpointStatus != EndpointStatusManual {
		t.Fatalf("terminal pool rows should carry endpoint metadata, got %#v", rows)
	}
	groups := TerminalPoolPageGroups(root)
	if len(groups) != 2 || groups[0].EndpointID != DefaultEndpointID || groups[1].EndpointID != "west" || len(groups[1].VisibleTerminalRows) != 1 {
		t.Fatalf("terminal pool should group rows by endpoint, got %#v", groups)
	}
}

func TestWorkbenchTreeItemsExposeEndpointOfflineMetadata(t *testing.T) {
	shell := DefaultShell().
		SplitActivePane(PaneState{ID: "pane-west", Title: "remote", Kind: PaneTerminalLive, TerminalID: "remote"}, SplitDirectionVertical).
		OpenWorkbenchTree()
	root := Root{
		Shell: shell,
		Endpoints: (EndpointStore{}).
			Upsert(EndpointItem{ID: DefaultEndpointID, Label: "This Mac", Transport: EndpointTransportLocal, ConnectMode: EndpointConnectAuto, Enabled: true, Status: EndpointStatusConnected}).
			Upsert(EndpointItem{ID: "west", Label: "US West", Transport: EndpointTransportSSH, ConnectMode: EndpointConnectOnDemand, Enabled: true, Status: EndpointStatusOffline, LastError: "ssh transport closed: exit status 255", LastErrorKind: EndpointErrorTransportClosed}),
		TerminalPool: TerminalPoolStore{Items: []TerminalPoolItem{
			{EndpointID: "west", TerminalID: "remote", Title: "remote shell"},
		}},
	}
	root.TerminalViews = root.TerminalViews.BindPane(NewEndpointPaneTerminalView("west", "pane-west", "remote", 7, 80, 24, TerminalResizeRoleFollower, "surface", TerminalPaneViewID("pane-west"), false))

	items := WorkbenchTreeItems(root)
	var pane WorkbenchTreeItem
	for _, item := range items {
		if item.Kind == WorkbenchTreeKindPane && item.PaneID == "pane-west" {
			pane = item
			break
		}
	}
	if pane.EndpointID != "west" || pane.EndpointLabel != "US West" || pane.EndpointStatus != EndpointStatusOffline || pane.EndpointErrorKind != EndpointErrorTransportClosed || pane.DisplayTitle != "remote shell" {
		t.Fatalf("workbench pane should carry endpoint offline metadata, got %#v", pane)
	}
}

func TestWorkbenchTreeItemsKeepErroredBindingWhenPaneLooksEmpty(t *testing.T) {
	shell := DefaultShell().OpenWorkbenchTree()
	shell.Workspace.Tabs[0].Panes[0] = PaneState{ID: DefaultPaneID, Title: "unconnected", Kind: PaneEmpty, Active: true}
	root := Root{
		Shell: shell,
		Endpoints: EndpointStore{}.
			Upsert(EndpointItem{ID: "west", Label: "US West", Transport: EndpointTransportSSH, ConnectMode: EndpointConnectOnDemand, Enabled: true, Status: EndpointStatusOffline, LastError: "daemon socket closed", LastErrorKind: EndpointErrorRemoteDaemon}),
		TerminalPool: TerminalPoolStore{Items: []TerminalPoolItem{{EndpointID: "west", TerminalID: "remote", Title: "remote shell"}}},
	}
	binding := NewEndpointPaneTerminalView("west", DefaultPaneID, "remote", 0, 80, 24, TerminalResizeRoleFollower, "surface", TerminalPaneViewID(DefaultPaneID), false)
	binding.LastError = "remote-daemon: daemon socket closed"
	root.TerminalViews = root.TerminalViews.BindPane(binding)

	items := WorkbenchTreeItems(root)
	var pane WorkbenchTreeItem
	for _, item := range items {
		if item.Kind == WorkbenchTreeKindPane && item.PaneID == DefaultPaneID {
			pane = item
			break
		}
	}
	if pane.TerminalID != "remote" || pane.EndpointID != "west" || pane.DisplayTitle != "remote shell" || pane.EndpointErrorKind != EndpointErrorRemoteDaemon {
		t.Fatalf("workbench pane should preserve errored binding metadata, got %#v", pane)
	}
}

func TestTerminalPoolStoreAppliesListWithStaleGuardAndError(t *testing.T) {
	pool := TerminalPoolStore{}
	pool = pool.RequestList()
	firstSeq := pool.RequestSeq
	pool = pool.RequestList()
	secondSeq := pool.RequestSeq

	next, applied := pool.ApplyList(firstSeq, []TerminalPoolItem{{TerminalID: "old"}}, "")
	if applied || len(next.Items) != 0 {
		t.Fatalf("stale list result must be ignored, got applied=%v pool=%#v", applied, next)
	}
	next, applied = pool.ApplyList(secondSeq, []TerminalPoolItem{{TerminalID: "term-1", Title: "main"}}, "")
	if !applied || next.Status != TerminalPoolReady || len(next.Items) != 1 || next.Items[0].TerminalID != "term-1" {
		t.Fatalf("expected fresh list applied, got applied=%v pool=%#v", applied, next)
	}
	next, applied = next.ApplyList(secondSeq, nil, "boom")
	if !applied || next.Status != TerminalPoolError || next.LastError != "boom" {
		t.Fatalf("expected error state, got applied=%v pool=%#v", applied, next)
	}
}

func TestTerminalPoolStoreKeepsBoundedResourceHistoryPerProcess(t *testing.T) {
	ref := LocalTerminalRef("term-1")
	pool := TerminalPoolStore{}
	base := time.Date(2026, 8, 10, 3, 0, 0, 0, time.UTC)
	for index := 0; index < terminalResourceHistoryLimit+5; index++ {
		pool = pool.RequestRefresh()
		var applied bool
		pool, applied = pool.ApplyList(pool.RequestSeq, []TerminalPoolItem{{
			TerminalID: ref.TerminalID,
			Resources: TerminalResourceUsage{
				PID: 100, CPUPercentX100: index * 100, MemoryBytes: uint64(index+1) * 1024,
				SampledAt: base.Add(time.Duration(index) * time.Second),
			},
		}}, "")
		if !applied {
			t.Fatalf("resource sample %d was not applied", index)
		}
	}
	samples := pool.ResourceSamples(ref)
	if len(samples) != terminalResourceHistoryLimit || samples[0].CPUPercentX100 != 500 || samples[len(samples)-1].CPUPercentX100 != (terminalResourceHistoryLimit+4)*100 {
		t.Fatalf("resource history should retain the latest bounded window, got %#v", samples)
	}

	latest := samples[len(samples)-1]
	pool, _ = pool.ApplyList(pool.RequestSeq, []TerminalPoolItem{{TerminalID: ref.TerminalID, Resources: latest}}, "")
	if got := len(pool.ResourceSamples(ref)); got != terminalResourceHistoryLimit {
		t.Fatalf("duplicate sample timestamp should not extend history, got %d", got)
	}

	pool, _ = pool.ApplyList(pool.RequestSeq, []TerminalPoolItem{{TerminalID: ref.TerminalID, Resources: TerminalResourceUsage{
		PID: 200, CPUPercentX100: 25, MemoryBytes: 2048, SampledAt: latest.SampledAt.Add(time.Second),
	}}}, "")
	samples = pool.ResourceSamples(ref)
	if len(samples) != 1 || samples[0].PID != 200 {
		t.Fatalf("new terminal process should start a fresh resource history, got %#v", samples)
	}

	pool = pool.ApplyRemovedRef(ref, "")
	if samples := pool.ResourceSamples(ref); len(samples) != 0 {
		t.Fatalf("removed terminal should clear resource history, got %#v", samples)
	}
}

func TestTerminalPoolStoreSeedsResourceHistoryFromFirstDaemonList(t *testing.T) {
	ref := NewTerminalRef("west", "term-1")
	base := time.Date(2026, 8, 10, 4, 0, 0, 0, time.UTC)
	history := make([]TerminalResourceUsage, terminalResourceHistoryLimit)
	for index := range history {
		history[index] = TerminalResourceUsage{
			PID: 100, CPUPercentX100: index * 100, MemoryBytes: uint64(index+1) * 1024,
			SampledAt: base.Add(time.Duration(index) * time.Second),
		}
	}
	pool := TerminalPoolStore{}.RequestList()
	pool, applied := pool.ApplyList(pool.RequestSeq, []TerminalPoolItem{{
		EndpointID: ref.EndpointID, TerminalID: ref.TerminalID,
		Resources: history[len(history)-1], ResourceHistory: history,
	}}, "")
	if !applied {
		t.Fatal("first daemon list was not applied")
	}
	samples := pool.ResourceSamples(ref)
	if len(samples) != terminalResourceHistoryLimit || samples[0] != history[0] || samples[len(samples)-1] != history[len(history)-1] {
		t.Fatalf("first list should seed the complete daemon history, got %#v", samples)
	}
	history[0].CPUPercentX100 = -1
	if pool.ResourceSamples(ref)[0].CPUPercentX100 == -1 {
		t.Fatal("terminal pool aliases daemon history input")
	}
}

func TestTerminalPoolStoreScopesActionsByEndpoint(t *testing.T) {
	pool := TerminalPoolStore{}
	pool = pool.RequestList()
	var applied bool
	pool, applied = pool.ApplyList(pool.RequestSeq, []TerminalPoolItem{
		{EndpointID: DefaultEndpointID, TerminalID: "term-1", Title: "local"},
		{EndpointID: "west", TerminalID: "term-1", Title: "west"},
	}, "")
	if !applied || len(pool.Items) != 2 {
		t.Fatalf("expected two endpoint-scoped items, applied=%v pool=%#v", applied, pool)
	}

	pool = pool.ApplyAttachedRef(NewTerminalRef("west", "term-1"), "")
	if pool.LastAttachedID != "term-1" || !pool.LastAttachedRef.Equal(NewTerminalRef("west", "term-1")) {
		t.Fatalf("attached result should record endpoint ref, got %#v", pool)
	}
	if pool.Items[0].Attached || !pool.Items[1].Attached {
		t.Fatalf("attach should mark only west item, got %#v", pool.Items)
	}

	pool = pool.ApplyEditedRef(NewTerminalRef("west", "term-1"), "west renamed", map[string]string{"site": "us-west"}, "")
	if pool.Items[0].Title != "local" || pool.Items[1].Title != "west renamed" || pool.Items[1].Tags["site"] != "us-west" {
		t.Fatalf("metadata edit should only update west item, got %#v", pool.Items)
	}

	pool = pool.ApplyAttachmentProjectionRef(LocalTerminalRef("term-1"), 3)
	if pool.Items[0].AttachmentCount != 3 || pool.Items[1].AttachmentCount != 0 {
		t.Fatalf("attachment count should only update local item, got %#v", pool.Items)
	}

	pool = pool.ApplyRemovedRef(NewTerminalRef("west", "term-1"), "")
	if len(pool.Items) != 1 || pool.Items[0].EndpointID != DefaultEndpointID || pool.Items[0].TerminalID != "term-1" {
		t.Fatalf("remove should only delete west item, got %#v", pool.Items)
	}
}

func TestWorkbenchTreeItemsProjectStructureSearchAndSelection(t *testing.T) {
	shell := DefaultShell().
		SplitActivePane(PaneState{ID: "pane-2", Title: "日志🚀", Kind: PaneTerminalLive, TerminalID: "term-2"}, SplitDirectionVertical).
		FocusPane(PaneCommandTarget{PaneID: DefaultPaneID}).
		OpenWorkbenchTree()
	root := Root{
		Shell:        shell.BindPaneTerminal(PaneCommandTarget{PaneID: DefaultPaneID}, "term-main"),
		TerminalPool: TerminalPoolStore{Items: []TerminalPoolItem{{TerminalID: "term-main", Title: "main terminal"}, {TerminalID: "term-2", Title: "日志终端"}}},
	}

	items := WorkbenchTreeItems(root)
	if len(items) != 4 {
		t.Fatalf("expected workspace/tab/two pane rows before floating, got %#v", items)
	}
	if items[0].Kind != WorkbenchTreeKindWorkspace || !items[0].Selected {
		t.Fatalf("expected workspace selected first, got %#v", items)
	}
	if items[3].Kind != WorkbenchTreeKindPane || items[3].PaneID != "pane-2" || items[3].TerminalID != "term-2" {
		t.Fatalf("expected pane-2 row with terminal binding, got %#v", items[3])
	}
	if items[2].DisplayTitle != "main terminal" || items[3].DisplayTitle != "日志终端" || items[3].PaneTitle != "日志🚀" {
		t.Fatalf("tab children should display connected terminal names, got %#v %#v", items[2], items[3])
	}
	var result FloatingCommandResult
	root.Shell, result = root.Shell.ApplyFloatingCommand(FloatingCommand{
		Action:   FloatingCommandCreate,
		TargetID: "float-1",
		Pane:     PaneState{ID: "float-pane", Title: "浮窗", Kind: PaneTerminalLive, TerminalID: "term-float"},
	})
	if result.Status != FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	items = WorkbenchTreeItems(root)
	if len(items) != 5 || items[4].Kind != WorkbenchTreeKindFloating || items[4].FloatingID != "float-1" || items[4].PaneID != "float-pane" || items[4].TerminalID != "term-float" || !strings.Contains(items[4].Summary, "floating") {
		t.Fatalf("expected actual floating row, got %#v", items)
	}
	var workbenchResult WorkbenchCommandResult
	root.Shell, workbenchResult = root.Shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandWorkspaceCreate, TargetID: "workspace-2", Name: "remote"})
	if workbenchResult.Status != WorkbenchCommandOK {
		t.Fatalf("create workspace: %#v", workbenchResult)
	}
	root.Shell = root.Shell.OpenWorkbenchTree()
	items = WorkbenchTreeItems(root)
	workspaceRows := 0
	for _, item := range items {
		if item.Kind == WorkbenchTreeKindWorkspace {
			workspaceRows++
		}
	}
	if workspaceRows != 2 || items[0].WorkspaceID != DefaultWorkspaceID || items[5].WorkspaceID != "workspace-2" || !items[5].Active {
		t.Fatalf("workbench tree should show all workspaces with active marker, items=%#v", items)
	}

	root.Shell = root.Shell.SetWorkbenchTreeQuery("日志")
	items = WorkbenchTreeItems(root)
	if len(items) != 1 || items[0].PaneID != "pane-2" || !items[0].Selected {
		t.Fatalf("query should filter to matching terminal-name pane row, got %#v", items)
	}
	root.Shell = root.Shell.SetWorkbenchTreeQuery("")
	root.Shell = root.Shell.MoveWorkbenchTreeSelection(2, len(WorkbenchTreeItems(root)))
	items = WorkbenchTreeItems(root)
	if len(items) != 8 || !items[2].Selected || items[2].PaneID != DefaultPaneID {
		t.Fatalf("selection should move to default pane row, got %#v", items)
	}
}

func TestWorkbenchTreeItemsRespectOverlayCollapseState(t *testing.T) {
	shell := DefaultShell().
		SplitActivePane(PaneState{ID: "pane-2", Title: "two", Kind: PaneTerminalLive, TerminalID: "term-2"}, SplitDirectionVertical).
		OpenWorkbenchTree()
	root := Root{Shell: shell}

	items := WorkbenchTreeItems(root)
	if len(items) != 4 || !items[0].Expandable || !items[1].Expandable {
		t.Fatalf("expected expandable workspace and tab rows, got %#v", items)
	}
	root.Shell = root.Shell.SetWorkbenchTreeItemCollapsed(items[1], true)
	items = WorkbenchTreeItems(root)
	if len(items) != 2 || !items[1].Collapsed || items[1].Kind != WorkbenchTreeKindTab {
		t.Fatalf("collapsed tab should hide pane children, got %#v", items)
	}

	root.Shell = root.Shell.SetWorkbenchTreeQuery("two")
	items = WorkbenchTreeItems(root)
	if len(items) != 1 || items[0].PaneID != "pane-2" {
		t.Fatalf("query should still see descendants hidden by collapse, got %#v", items)
	}

	root.Shell = root.Shell.SetWorkbenchTreeQuery("")
	items = WorkbenchTreeItems(root)
	root.Shell = root.Shell.SetWorkbenchTreeItemCollapsed(items[0], true)
	items = WorkbenchTreeItems(root)
	if len(items) != 1 || !items[0].Collapsed || items[0].Kind != WorkbenchTreeKindWorkspace {
		t.Fatalf("collapsed workspace should hide tab subtree, got %#v", items)
	}
}

func TestClipboardHistoryItemsFilterAndSelection(t *testing.T) {
	shell := DefaultShell().OpenClipboardHistory()
	root := Root{
		Shell: shell,
		Clipboard: ClipboardStore{
			Entries: []ClipboardEntry{
				{ID: "clip:1", Title: "alpha", Text: "alpha", Preview: "alpha"},
				{ID: "clip:2", Title: "build log", Text: "build\nlog", Preview: "build …"},
			},
		},
	}

	items := ClipboardHistoryItems(root)
	if len(items) != 2 || !items[0].Selected || items[0].Title != "alpha" {
		t.Fatalf("expected first clipboard item selected, got %#v", items)
	}
	root.Shell = root.Shell.SetClipboardHistoryQuery("build")
	items = ClipboardHistoryItems(root)
	if len(items) != 1 || !items[0].Selected || items[0].Title != "build log" {
		t.Fatalf("expected filtered clipboard item, got %#v", items)
	}
	root.Clipboard.Entries = append([]ClipboardEntry{{ID: "clip:3", Title: "git commit", Text: "git commit -m fix terminal", Preview: "git commit -m fix terminal"}}, root.Clipboard.Entries...)
	root.Shell = root.Shell.SetClipboardHistoryQuery("gft")
	items = ClipboardHistoryItems(root)
	if len(items) != 1 || items[0].Title != "git commit" || len(items[0].PreviewMatchIndexes) != 3 {
		t.Fatalf("expected fuzzy clipboard query to match git commit preview, got %#v", items)
	}
	root.Shell = root.Shell.SetClipboardHistoryQuery("")
	root.Shell = root.Shell.MoveClipboardHistorySelection(1, len(ClipboardHistoryItems(root)))
	items = ClipboardHistoryItems(root)
	if len(items) != 3 || !items[1].Selected || items[1].Title != "alpha" {
		t.Fatalf("expected selection moved to second clipboard item, got %#v", items)
	}
}

func TestClipboardHistoryNameWidthDefaultsAndClamps(t *testing.T) {
	shell := DefaultShell().OpenClipboardHistory()
	if got := ClipboardHistoryNameWidth(shell.Overlay); got != DefaultClipboardHistoryNameWidth {
		t.Fatalf("expected default clipboard name width %d, got %d", DefaultClipboardHistoryNameWidth, got)
	}

	shell = shell.MoveClipboardHistoryNameWidth(8)
	if got := ClipboardHistoryNameWidth(shell.Overlay); got != DefaultClipboardHistoryNameWidth+8 {
		t.Fatalf("expected dragged clipboard name width, got %d", got)
	}
	shell = shell.SetClipboardHistoryNameWidth(-100)
	if got := ClipboardHistoryNameWidth(shell.Overlay); got != DefaultClipboardHistoryNameWidth {
		t.Fatalf("non-positive explicit width should fall back to default, got %d", got)
	}
	shell = shell.SetClipboardHistoryNameWidth(1)
	if got := ClipboardHistoryNameWidth(shell.Overlay); got != MinClipboardHistoryNameWidth {
		t.Fatalf("expected min clipboard name width %d, got %d", MinClipboardHistoryNameWidth, got)
	}
	shell = shell.SetClipboardHistoryNameWidth(999)
	if got := ClipboardHistoryNameWidth(shell.Overlay); got != MaxClipboardHistoryNameWidth {
		t.Fatalf("expected max clipboard name width %d, got %d", MaxClipboardHistoryNameWidth, got)
	}
}

func TestShellSplitActivePaneCreatesMinimalPaneTree(t *testing.T) {
	shell := DefaultShell().SplitActivePane(PaneState{
		ID:         "pane-2",
		Title:      "logs",
		Kind:       PaneTerminalLive,
		TerminalID: "term-2",
	}, SplitDirectionHorizontal)

	tab := shell.Workspace.Tabs[0]
	if len(tab.Panes) != 2 {
		t.Fatalf("expected two panes, got %#v", tab.Panes)
	}
	if shell.ActivePaneID != "pane-2" || tab.ActivePaneID != "pane-2" {
		t.Fatalf("expected new pane active, shell=%#v tab=%#v", shell, tab)
	}
	if tab.RootSplit.Direction != SplitDirectionHorizontal || len(tab.RootSplit.Children) != 2 {
		t.Fatalf("expected horizontal root split, got %#v", tab.RootSplit)
	}
	if tab.RootSplit.Children[0].PaneID != DefaultPaneID || tab.RootSplit.Children[1].PaneID != "pane-2" {
		t.Fatalf("unexpected split children %#v", tab.RootSplit.Children)
	}
	if !tab.Panes[1].Active || tab.Panes[0].Active {
		t.Fatalf("expected only new pane active, got %#v", tab.Panes)
	}
}

func TestShellSplitActivePaneSupportsVerticalDirection(t *testing.T) {
	shell := DefaultShell().SplitActivePane(PaneState{ID: "pane-2"}, SplitDirectionVertical)
	if shell.Workspace.Tabs[0].RootSplit.Direction != SplitDirectionVertical {
		t.Fatalf("expected vertical split, got %#v", shell.Workspace.Tabs[0].RootSplit)
	}
}

func TestShellFocusCloseAndZoomPaneState(t *testing.T) {
	shell := DefaultShell().
		SplitActivePane(PaneState{ID: "pane-2", Title: "logs", Kind: PaneTerminalLive, TerminalID: "term-2"}, SplitDirectionVertical).
		SplitActivePane(PaneState{ID: "pane-3", Title: "build", Kind: PaneTerminalLive, TerminalID: "term-3"}, SplitDirectionHorizontal)

	shell = shell.FocusPane(PaneCommandTarget{PaneID: DefaultPaneID})
	if shell.ActivePaneID != DefaultPaneID || !shell.Workspace.Tabs[0].Panes[0].Active {
		t.Fatalf("expected focused default pane, got %#v", shell)
	}

	shell = shell.ZoomPane(PaneCommandTarget{PaneID: "pane-2"})
	if shell.ZoomedPaneID != "pane-2" || shell.ActivePaneID != "pane-2" {
		t.Fatalf("expected zoomed pane-2, got %#v", shell)
	}
	shell = shell.UnzoomPane()
	if shell.ZoomedPaneID != "" {
		t.Fatalf("expected unzoomed shell, got %#v", shell)
	}

	shell = shell.ClosePane(PaneCommandTarget{PaneID: "pane-2"})
	tab := shell.Workspace.Tabs[0]
	if len(tab.Panes) != 2 || shell.HasPane(PaneCommandTarget{PaneID: "pane-2"}) {
		t.Fatalf("expected pane-2 closed, got %#v", shell)
	}
	if containsPaneID(tab.RootSplit, "pane-2") {
		t.Fatalf("closed pane must be removed from split tree, got %#v", tab.RootSplit)
	}
}

func TestShellCloseLastPaneIsIgnoredByStateMethod(t *testing.T) {
	shell := DefaultShell().ClosePane(PaneCommandTarget{PaneID: DefaultPaneID})
	if len(shell.Workspace.Tabs[0].Panes) != 1 || shell.ActivePaneID != DefaultPaneID {
		t.Fatalf("last pane close must be ignored, got %#v", shell)
	}
}

func TestShellResizeSetSizeAndBalancePaneGeometry(t *testing.T) {
	shell := DefaultShell().SplitActivePane(PaneState{ID: "pane-2"}, SplitDirectionVertical)

	shell = shell.ResizePane(PaneCommandTarget{PaneID: DefaultPaneID}, PaneResizeRight, 4)
	if shell.Workspace.Tabs[0].RootSplit.BiasCells != 4 {
		t.Fatalf("expected resize bias for first pane, got %#v", shell.Workspace.Tabs[0].RootSplit)
	}

	shell = shell.SetPaneSize(PaneCommand{
		Action:   PaneCommandSetSize,
		Target:   PaneCommandTarget{PaneID: "pane-2"},
		SizeMode: PaneSizeRatio,
		Ratio:    0.25,
	})
	if got := shell.Workspace.Tabs[0].RootSplit.Ratio; got != 0.75 {
		t.Fatalf("ratio is stored as first child ratio, got %#v", shell.Workspace.Tabs[0].RootSplit)
	}

	shell = shell.SetPaneSize(PaneCommand{
		Action:   PaneCommandSetSize,
		Target:   PaneCommandTarget{PaneID: "pane-2"},
		SizeMode: PaneSizeCells,
		Cols:     12,
	})
	root := shell.Workspace.Tabs[0].RootSplit
	if root.FixedPaneID != "pane-2" || root.FixedCols != 12 || root.Ratio != 0 {
		t.Fatalf("expected fixed cols for pane-2, got %#v", root)
	}

	shell = shell.BalancePanes(PaneCommandTarget{PaneID: "pane-2"})
	root = shell.Workspace.Tabs[0].RootSplit
	if root.BiasCells != 0 || root.Ratio != 0 || root.FixedPaneID != "" || root.FixedCols != 0 {
		t.Fatalf("expected balanced split hints, got %#v", root)
	}
}

func TestShellResizeSplitPathOnlyChangesExactNestedDivider(t *testing.T) {
	shell := DefaultShell().
		SplitActivePane(PaneState{ID: "pane-middle"}, SplitDirectionVertical).
		SplitActivePane(PaneState{ID: "pane-right"}, SplitDirectionVertical)

	shell = shell.ResizeSplitPath(PaneCommandTarget{PaneID: "pane-middle"}, "root/1", PaneResizeRight, -4)
	root := shell.Workspace.Tabs[0].RootSplit
	if root.BiasCells != 0 {
		t.Fatalf("exact nested divider resize must not change root split, got %#v", root)
	}
	if len(root.Children) < 2 || root.Children[1].BiasCells != -4 {
		t.Fatalf("expected right nested split bias only, got %#v", root)
	}

	unchanged := shell.ResizeSplitPath(PaneCommandTarget{PaneID: "pane-middle"}, "root/9", PaneResizeRight, 3)
	if unchanged.Workspace.Tabs[0].RootSplit.Children[1].BiasCells != -4 {
		t.Fatalf("invalid split path must not mutate layout, got %#v", unchanged.Workspace.Tabs[0].RootSplit)
	}
}

func TestShellResizePaneGroupKeepsNonAdjacentColumnsFixed(t *testing.T) {
	shell := DefaultShell().
		SplitActivePane(PaneState{ID: "pane-2"}, SplitDirectionVertical).
		SplitActivePane(PaneState{ID: "pane-3"}, SplitDirectionVertical).
		SplitActivePane(PaneState{ID: "pane-4"}, SplitDirectionVertical)

	shell = shell.ResizePaneGroup(PaneCommandTarget{PaneID: "pane-2"}, PaneResizeRight, []PaneResizeGroupItem{
		{PaneID: "pane-2", Cells: 25},
		{PaneID: "pane-3", Cells: 15},
		{PaneID: "pane-4", Cells: 20},
	})
	root := shell.Workspace.Tabs[0].RootSplit
	right := root.Children[1]
	if right.FixedPaneID != "pane-2" || right.FixedCols != 25 {
		t.Fatalf("right subtree should fix pane-2 width, got %#v", right)
	}
	if right.Children[1].FixedPaneID != "pane-3" || right.Children[1].FixedCols != 15 {
		t.Fatalf("right subtree should fix pane-3 width without resizing pane-4, got %#v", right.Children[1])
	}
	if right.Children[1].Children[1].PaneID != "pane-4" {
		t.Fatalf("pane-4 should remain an independent leaf, got %#v", right)
	}
}

func TestShellResizePaneGroupKeepsStackedRightColumnSharedWidth(t *testing.T) {
	shell := DefaultShell()
	tab := &shell.Workspace.Tabs[0]
	tab.Panes = []PaneState{
		{ID: "left", Kind: PaneTerminalLive},
		{ID: "top", Kind: PaneTerminalLive},
		{ID: "middle-left", Kind: PaneTerminalLive},
		{ID: "middle-right", Kind: PaneTerminalLive},
		{ID: "bottom", Kind: PaneTerminalLive},
	}
	tab.ActivePaneID = "middle-left"
	tab.RootSplit = SplitNode{
		Direction: SplitDirectionVertical,
		Children: []SplitNode{
			{PaneID: "left"},
			{
				Direction: SplitDirectionHorizontal,
				Children: []SplitNode{
					{PaneID: "top"},
					{
						Direction: SplitDirectionHorizontal,
						Children: []SplitNode{
							{
								Direction: SplitDirectionVertical,
								Children:  []SplitNode{{PaneID: "middle-left"}, {PaneID: "middle-right"}},
							},
							{PaneID: "bottom"},
						},
					},
				},
			},
		},
	}
	shell.ActivePaneID = "middle-left"
	shell = shell.EnsureDefaults()

	shell = shell.ResizePaneGroup(PaneCommandTarget{PaneID: "left"}, PaneResizeRight, []PaneResizeGroupItem{
		{PaneID: "left", Cells: 36, DeltaSign: 1},
		{PaneID: "top", Cells: 44, DeltaSign: -1},
		{PaneID: "middle-left", Cells: 24, DeltaSign: -1},
		{PaneID: "middle-right", Cells: 20},
		{PaneID: "bottom", Cells: 44, DeltaSign: -1},
	})

	root := shell.Workspace.Tabs[0].RootSplit
	if root.FixedPaneID != "left" || root.FixedCols != 36 {
		t.Fatalf("root divider should fix the left column width, got %#v", root)
	}
	rightColumn := root.Children[1]
	if rightColumn.FixedCols != 0 || rightColumn.FixedRows != 0 {
		t.Fatalf("orthogonal stacked column should keep height geometry independent, got %#v", rightColumn)
	}
	middleRow := rightColumn.Children[1].Children[0]
	if middleRow.FixedPaneID != "middle-left" || middleRow.FixedCols != 24 {
		t.Fatalf("nested middle row should resize its left child while preserving right child anchor, got %#v", middleRow)
	}
}

func TestShellFloatingCommandsManageRectZOrderAndCollapse(t *testing.T) {
	shell := DefaultShell()
	var result FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(FloatingCommand{
		Action:   FloatingCommandCreate,
		TargetID: "float-1",
		Pane:     PaneState{ID: "float-pane", Title: "日志🚀", Kind: PaneEmpty},
		BoundsW:  80,
		BoundsH:  24,
	})
	floatings := shell.ActiveFloatings()
	if result.Status != FloatingCommandOK || len(floatings) != 1 || shell.ActiveFloatingID() != "float-1" {
		t.Fatalf("expected created active floating, result=%#v shell=%#v", result, shell)
	}
	created := floatings[0]
	if created.Rect != (FloatingRect{X: 8, Y: 3, W: 64, H: 18}) || !created.Active {
		t.Fatalf("expected centered clamped floating, got %#v", created)
	}

	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandMove, TargetID: "float-1", DeltaX: -999, DeltaY: -999, BoundsW: 80, BoundsH: 24})
	floatings = shell.ActiveFloatings()
	if result.Status != FloatingCommandOK || floatings[0].Rect.X != 0 || floatings[0].Rect.Y != 0 {
		t.Fatalf("move should clamp to viewport, result=%#v floating=%#v", result, floatings[0])
	}
	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandResize, TargetID: "float-1", DeltaW: -999, DeltaH: -999, BoundsW: 80, BoundsH: 24})
	floatings = shell.ActiveFloatings()
	if result.Status != FloatingCommandOK || floatings[0].Rect.W != 16 || floatings[0].Rect.H != 4 {
		t.Fatalf("resize should keep minimum floating size, result=%#v floating=%#v", result, floatings[0])
	}
	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandToggleCollapse, TargetID: "float-1"})
	floatings = shell.ActiveFloatings()
	if result.Status != FloatingCommandOK || shell.ActiveFloatingID() != "" || !floatings[0].Collapsed || floatings[0].Active {
		t.Fatalf("expected collapsed floating to clear active target, result=%#v shell=%#v floating=%#v", result, shell, floatings[0])
	}
	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandFocusRaise, TargetID: "float-1"})
	floatings = shell.ActiveFloatings()
	if result.Status != FloatingCommandInvalid || shell.ActiveFloatingID() != "" || floatings[0].Active {
		t.Fatalf("hidden floating should not be focus-raised, result=%#v shell=%#v", result, shell)
	}
	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandSummon, TargetID: "float-1"})
	floatings = shell.ActiveFloatings()
	if result.Status != FloatingCommandOK || shell.ActiveFloatingID() != "float-1" || floatings[0].Collapsed || !floatings[0].Active {
		t.Fatalf("summon should expand and raise hidden floating, result=%#v shell=%#v", result, shell)
	}
	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandDeactivate})
	floatings = shell.ActiveFloatings()
	if result.Status != FloatingCommandOK || shell.ActiveFloatingID() != "" || floatings[0].Active {
		t.Fatalf("deactivate should keep floating but clear active state, result=%#v shell=%#v", result, shell)
	}
	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandClose, TargetID: "float-1"})
	if result.Status != FloatingCommandOK || len(shell.ActiveFloatings()) != 0 || shell.ActiveFloatingID() != "" {
		t.Fatalf("expected closed floating, result=%#v shell=%#v", result, shell)
	}
}

func TestShellFloatingCreateCascadesDefaultRects(t *testing.T) {
	shell := DefaultShell()
	var result FloatingCommandResult
	for i, id := range []string{"float-1", "float-2", "float-3"} {
		shell, result = shell.ApplyFloatingCommand(FloatingCommand{
			Action:   FloatingCommandCreate,
			TargetID: id,
			Pane:     PaneState{ID: id + "-pane", Title: id, Kind: PaneEmpty},
			BoundsW:  80,
			BoundsH:  24,
		})
		if result.Status != FloatingCommandOK {
			t.Fatalf("create floating %d: %#v", i, result)
		}
	}

	floatings := shell.ActiveFloatings()
	want := []FloatingRect{
		{X: 8, Y: 3, W: 64, H: 18},
		{X: 12, Y: 4, W: 64, H: 18},
		{X: 16, Y: 5, W: 64, H: 18},
	}
	if len(floatings) != len(want) {
		t.Fatalf("unexpected floating count: %#v", floatings)
	}
	for index, rect := range want {
		if floatings[index].Rect != rect {
			t.Fatalf("floating %d should be center-cascaded, got %#v want %#v all=%#v", index, floatings[index].Rect, rect, floatings)
		}
	}

	shell, result = shell.ApplyFloatingCommand(FloatingCommand{
		Action:   FloatingCommandCreate,
		TargetID: "manual",
		Pane:     PaneState{ID: "manual-pane", Title: "manual", Kind: PaneEmpty},
		Rect:     FloatingRect{X: 1, Y: 2, W: 30, H: 8},
		BoundsW:  80,
		BoundsH:  24,
	})
	if result.Status != FloatingCommandOK {
		t.Fatalf("create manual floating: %#v", result)
	}
	if got := shell.ActiveFloatings()[3].Rect; got != (FloatingRect{X: 1, Y: 2, W: 30, H: 8}) {
		t.Fatalf("manual floating rect must not be cascaded, got %#v", got)
	}
}

func TestShellFloatingGroupCommandsManageCollapseAndFitMode(t *testing.T) {
	shell := DefaultShell()
	var result FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(FloatingCommand{
		Action:   FloatingCommandCreate,
		TargetID: "float-1",
		Pane:     PaneState{ID: "float-pane-1", Title: "one", Kind: PaneEmpty},
		Rect:     FloatingRect{X: 1, Y: 1, W: 20, H: 8},
		BoundsW:  100,
		BoundsH:  40,
	})
	if result.Status != FloatingCommandOK {
		t.Fatalf("create first floating: %#v", result)
	}
	shell, result = shell.ApplyFloatingCommand(FloatingCommand{
		Action:   FloatingCommandCreate,
		TargetID: "float-2",
		Pane:     PaneState{ID: "float-pane-2", Title: "two", Kind: PaneEmpty},
		Rect:     FloatingRect{X: 10, Y: 5, W: 22, H: 9},
		BoundsW:  100,
		BoundsH:  40,
	})
	if result.Status != FloatingCommandOK {
		t.Fatalf("create second floating: %#v", result)
	}

	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandCollapseAll})
	floatings := shell.ActiveFloatings()
	if result.Status != FloatingCommandOK || shell.ActiveFloatingID() != "" || !floatings[0].Collapsed || !floatings[1].Collapsed {
		t.Fatalf("collapse all should collapse every floating, result=%#v shell=%#v", result, shell)
	}

	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandShowAll})
	floatings = shell.ActiveFloatings()
	if result.Status != FloatingCommandOK || shell.ActiveFloatingID() == "" || floatings[0].Collapsed || floatings[1].Collapsed {
		t.Fatalf("show all should restore every floating, result=%#v shell=%#v", result, shell)
	}

	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandToggleAll})
	floatings = shell.ActiveFloatings()
	if result.Status != FloatingCommandOK || !floatings[0].Collapsed || !floatings[1].Collapsed {
		t.Fatalf("toggle all should collapse open floatings, result=%#v shell=%#v", result, shell)
	}
	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandToggleAll})
	floatings = shell.ActiveFloatings()
	if result.Status != FloatingCommandOK || floatings[0].Collapsed || floatings[1].Collapsed {
		t.Fatalf("toggle all should restore collapsed floatings, result=%#v shell=%#v", result, shell)
	}

	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandFit, TargetID: "float-1", FitCols: 50, FitRows: 16, BoundsW: 100, BoundsH: 40})
	if result.Status != FloatingCommandOK {
		t.Fatalf("fit floating: %#v", result)
	}
	if got := shell.ActiveFloatings()[0]; got.Rect.W != 52 || got.Rect.H != 18 || got.FitMode != FloatingFitManual || got.AutoFit != (FloatingAutoFitState{}) {
		t.Fatalf("fit should resize to content extent plus chrome, got %#v", got)
	}

	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandToggleAutoFit, TargetID: "float-1", FitCols: 60, FitRows: 20, BoundsW: 100, BoundsH: 40})
	if result.Status != FloatingCommandOK {
		t.Fatalf("enable auto-fit: %#v", result)
	}
	if got := shell.ActiveFloatings()[0]; got.FitMode != FloatingFitAuto || got.AutoFit.Cols != 60 || got.AutoFit.Rows != 20 || got.Rect.W != 62 || got.Rect.H != 22 {
		t.Fatalf("auto-fit should save latest fit size and update rect, got %#v", got)
	}

	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandRefreshAutoFit, TargetID: "float-1", FitCols: 70, FitRows: 18, BoundsW: 100, BoundsH: 40})
	if result.Status != FloatingCommandOK {
		t.Fatalf("refresh auto-fit: %#v", result)
	}
	if got := shell.ActiveFloatings()[0]; got.AutoFit.Cols != 70 || got.AutoFit.Rows != 18 || got.Rect.W != 72 || got.Rect.H != 20 {
		t.Fatalf("refresh auto-fit should update metadata and rect, got %#v", got)
	}

	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandResize, TargetID: "float-1", DeltaW: -2, DeltaH: -2, BoundsW: 100, BoundsH: 40})
	if result.Status != FloatingCommandOK {
		t.Fatalf("manual resize after auto-fit: %#v", result)
	}
	if got := shell.ActiveFloatings()[0]; got.FitMode != FloatingFitManual || got.AutoFit != (FloatingAutoFitState{}) {
		t.Fatalf("manual resize should clear auto-fit state, got %#v", got)
	}
}

func TestShellBindFloatingTerminal(t *testing.T) {
	shell := DefaultShell().BindPaneTerminal(PaneCommandTarget{PaneID: DefaultPaneID}, "term-main")
	var result FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(FloatingCommand{
		Action:   FloatingCommandCreate,
		TargetID: "float-1",
		Pane:     PaneState{ID: "float-pane", Title: "float", Kind: PaneEmpty},
	})
	if result.Status != FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}

	shell = shell.BindFloatingTerminal("float-1", "term-float")
	floatings := shell.ActiveFloatings()
	if shell.ActiveFloatingID() != "float-1" || len(floatings) != 1 || !floatings[0].Active {
		t.Fatalf("floating bind should focus target floating, got %#v", floatings)
	}
	if pane := floatings[0].Pane; pane.Kind != PaneTerminalLive || pane.TerminalID != "term-float" {
		t.Fatalf("floating pane should bind terminal-live target, got %#v", pane)
	}
	tiled, ok := shell.Pane(PaneCommandTarget{PaneID: DefaultPaneID})
	if !ok || tiled.TerminalID != "term-main" {
		t.Fatalf("floating bind must not rewrite tiled pane terminal, got %#v ok=%v", tiled, ok)
	}
}

func TestShellFloatingsAreScopedToActiveTab(t *testing.T) {
	shell := DefaultShell()
	var result FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(FloatingCommand{
		Action:   FloatingCommandCreate,
		TargetID: "float-main",
		Pane:     PaneState{ID: "float-main-pane", Title: "main float", Kind: PaneEmpty},
	})
	if result.Status != FloatingCommandOK || len(shell.ActiveFloatings()) != 1 {
		t.Fatalf("expected floating under default tab, result=%#v shell=%#v", result, shell)
	}

	var workbenchResult WorkbenchCommandResult
	shell, workbenchResult = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandTabCreate, TargetID: "tab-build", Name: "build"})
	if workbenchResult.Status != WorkbenchCommandOK {
		t.Fatalf("create tab: %#v", workbenchResult)
	}
	if shell.Workspace.ActiveTabID != "tab-build" || len(shell.ActiveFloatings()) != 0 || shell.ActiveFloatingID() != "" {
		t.Fatalf("new active tab should not inherit previous tab floating, shell=%#v", shell)
	}

	shell, result = shell.ApplyFloatingCommand(FloatingCommand{
		Action:   FloatingCommandCreate,
		TargetID: "float-build",
		Pane:     PaneState{ID: "float-build-pane", Title: "build float", Kind: PaneEmpty},
	})
	if result.Status != FloatingCommandOK || shell.ActiveFloatingID() != "float-build" {
		t.Fatalf("expected floating under build tab, result=%#v shell=%#v", result, shell)
	}

	shell, workbenchResult = shell.ApplyWorkbenchCommand(WorkbenchCommand{Action: WorkbenchCommandTabSwitch, TargetID: DefaultTabID})
	if workbenchResult.Status != WorkbenchCommandOK {
		t.Fatalf("switch tab: %#v", workbenchResult)
	}
	floatings := shell.ActiveFloatings()
	if len(floatings) != 1 || floatings[0].ID != "float-main" || shell.ActiveFloatingID() != "float-main" {
		t.Fatalf("switching back should restore default tab floating only, floatings=%#v active=%q", floatings, shell.ActiveFloatingID())
	}
}

func TestShellTerminalOverlaysTargetActiveFloating(t *testing.T) {
	shell := DefaultShell()
	var result FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(FloatingCommand{Action: FloatingCommandCreate, TargetID: "float-1", Pane: PaneState{ID: "float-pane", Kind: PaneEmpty}})
	if result.Status != FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}

	picker := shell.OpenTerminalPicker()
	if picker.Overlay.TargetID != "float-1" {
		t.Fatalf("terminal picker should target active floating, got %#v", picker.Overlay)
	}
	pool := shell.OpenTerminalPool()
	if pool.Overlay.TargetID != "float-1" {
		t.Fatalf("terminal pool should target active floating, got %#v", pool.Overlay)
	}
}

func TestShellPromptOverlaySubmitCancelAndConfirm(t *testing.T) {
	shell := DefaultShell().OpenPrompt(PromptState{Title: "Danger", Destructive: true, ConfirmText: "DELETE"})
	if !shell.Overlay.Open || shell.Overlay.Kind != OverlayPrompt || !shell.Overlay.Prompt.Destructive {
		t.Fatalf("expected prompt overlay, got %#v", shell.Overlay)
	}
	shell = shell.SetPromptValue("nope").SubmitPrompt()
	if shell.Overlay.Prompt.Submitted || shell.Overlay.Prompt.LastResult != "confirm required: DELETE" {
		t.Fatalf("expected destructive confirm guard, got %#v", shell.Overlay.Prompt)
	}
	shell = shell.SetPromptValue("DELETE").SubmitPrompt()
	if !shell.Overlay.Prompt.Submitted || shell.Overlay.Prompt.LastResult != "DELETE" {
		t.Fatalf("expected prompt submitted, got %#v", shell.Overlay.Prompt)
	}
	shell = shell.OpenPrompt(PromptState{Title: "Rename"}).CancelPrompt()
	if !shell.Overlay.Prompt.Canceled {
		t.Fatalf("expected prompt canceled, got %#v", shell.Overlay.Prompt)
	}
}

func TestShellUpdatesDoNotMutatePreviousSlices(t *testing.T) {
	original := DefaultShell().AddToast(ToastSpec{ID: "old"})
	next := original.AddToast(ToastSpec{ID: "new"})
	next.Workspace.Tabs[0].Panes[0].Title = "mutated"

	if len(original.Toasts) != 1 || original.Toasts[0].ID != "old" {
		t.Fatalf("original toasts mutated: %#v next=%#v", original.Toasts, next.Toasts)
	}
	if original.Workspace.Tabs[0].Panes[0].Title != "shell" {
		t.Fatalf("original pane mutated: %#v", original.Workspace.Tabs[0].Panes[0])
	}
}

func TestShellReadonlyDefaultsDoesNotCloneInitializedSlices(t *testing.T) {
	shell := DefaultShell().AddToast(ToastSpec{ID: "one"})
	view := shell.ReadonlyDefaults()

	if len(view.Workspace.Tabs) == 0 || len(shell.Workspace.Tabs) == 0 {
		t.Fatalf("test expects default tab, view=%#v shell=%#v", view.Workspace, shell.Workspace)
	}
	if &view.Workspace.Tabs[0] != &shell.Workspace.Tabs[0] {
		t.Fatalf("readonly defaults should reuse initialized workspace tab slice")
	}
	if len(view.Workspace.Tabs[0].Panes) == 0 || len(shell.Workspace.Tabs[0].Panes) == 0 {
		t.Fatalf("test expects default pane, view=%#v shell=%#v", view.Workspace.Tabs[0], shell.Workspace.Tabs[0])
	}
	if &view.Workspace.Tabs[0].Panes[0] != &shell.Workspace.Tabs[0].Panes[0] {
		t.Fatalf("readonly defaults should reuse initialized pane slice")
	}
	if len(view.Toasts) == 0 || len(shell.Toasts) == 0 || &view.Toasts[0] != &shell.Toasts[0] {
		t.Fatalf("readonly defaults should reuse toast slice, view=%#v shell=%#v", view.Toasts, shell.Toasts)
	}
}

func TestShellReadonlyDefaultsFallsBackForZeroValue(t *testing.T) {
	view := (ShellStore{}).ReadonlyDefaults()

	if view.Workspace.ID != DefaultWorkspaceID || view.ActivePaneID != DefaultPaneID {
		t.Fatalf("zero shell should be normalized, got %#v", view)
	}
	if len(view.Workspace.Tabs) != 1 || len(view.Workspace.Tabs[0].Panes) != 1 {
		t.Fatalf("zero shell should seed default tab and pane, got %#v", view.Workspace)
	}
}

func TestShellReadonlyDefaultsFallsBackForStaleWorkspaceList(t *testing.T) {
	shell := DefaultShell()
	shell.Workspace.Tabs[0].Title = "active-title"
	shell.Workspaces = cloneWorkspaces(shell.Workspaces)
	shell.Workspaces[0].Tabs[0].Title = "stale-title"

	view := shell.ReadonlyDefaults()

	if view.Workspaces[0].Tabs[0].Title != "active-title" {
		t.Fatalf("stale workspace list should be repaired, got %#v", view.Workspaces[0].Tabs[0])
	}
	if &view.Workspace.Tabs[0] == &shell.Workspace.Tabs[0] {
		t.Fatalf("fallback normalization should not reuse stale workspace backing")
	}
}

func containsPaneID(node SplitNode, paneID string) bool {
	if node.PaneID == paneID {
		return true
	}
	for _, child := range node.Children {
		if containsPaneID(child, paneID) {
			return true
		}
	}
	return false
}

package state

import "strings"

func (store ShellStore) splitPaneWorkbench(command WorkbenchCommand) (ShellStore, WorkbenchCommandResult) {
	paneCommand := command.Pane
	paneCommand.Action = PaneCommandSplit
	if paneCommand.Source == "" {
		paneCommand.Source = command.Source
	}
	if paneCommand.Target.WorkspaceID != "" && paneCommand.Target.WorkspaceID != store.EnsureDefaults().Workspace.ID {
		var ok bool
		store, ok = store.withCommandWorkspace(WorkbenchCommand{Action: command.Action, Target: paneCommand.Target})
		if !ok {
			return store, workbenchCommandInvalid(command.Action, "workspace not found")
		}
	}
	if paneCommand.Target.PaneID == "" || paneCommand.Target.TabID == "" || paneCommand.Target.WorkspaceID == "" {
		paneCommand.Target = store.workbenchPaneTarget(WorkbenchCommand{Target: paneCommand.Target, TargetID: command.TargetID})
	}
	next, result := store.ApplyPaneCommand(paneCommand)
	if result.Status != PaneCommandOK {
		return store, workbenchCommandInvalid(command.Action, result.Reason)
	}
	next.Workspaces = upsertWorkspace(next.Workspaces, next.Workspace)
	return next, WorkbenchCommandResult{Status: WorkbenchCommandOK, Action: command.Action, ID: paneCommand.NewPane.ID}
}

func (store ShellStore) renamePane(command WorkbenchCommand) (ShellStore, WorkbenchCommandResult) {
	var ok bool
	store, ok = store.withCommandWorkspace(command)
	if !ok {
		return store, workbenchCommandInvalid(command.Action, "workspace not found")
	}
	name := strings.TrimSpace(command.Name)
	if name == "" {
		return store, workbenchCommandInvalid(command.Action, "missing pane name")
	}
	target := store.workbenchPaneTarget(command)
	tabIndex := store.tabIndexForTarget(target)
	if tabIndex < 0 {
		return store, workbenchCommandInvalid(command.Action, "pane not found")
	}
	paneID := target.PaneID
	if paneID == "" {
		paneID = store.ActivePaneID
	}
	for index := range store.Workspace.Tabs[tabIndex].Panes {
		if store.Workspace.Tabs[tabIndex].Panes[index].ID != paneID {
			continue
		}
		store.Workspace.Tabs[tabIndex].Panes[index].Title = name
		store.Workspace = store.Workspace.ensureActive(store.ActivePaneID)
		store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
		return store.EnsureDefaults(), WorkbenchCommandResult{Status: WorkbenchCommandOK, Action: command.Action, ID: paneID}
	}
	return store, workbenchCommandInvalid(command.Action, "pane not found")
}

func (store ShellStore) detachPane(command WorkbenchCommand) (ShellStore, WorkbenchCommandResult) {
	var targetOK bool
	store, targetOK = store.withCommandWorkspace(command)
	if !targetOK {
		return store, workbenchCommandInvalid(command.Action, "workspace not found")
	}
	target := store.workbenchPaneTarget(command)
	pane, ok := store.Pane(target)
	if !ok {
		return store, workbenchCommandInvalid(command.Action, "pane not found")
	}
	// detach 只断开 workbench pane 与 terminal 的绑定，不销毁 daemon terminal。
	store = store.setPaneDetached(target)
	return store, WorkbenchCommandResult{Status: WorkbenchCommandOK, Action: command.Action, ID: pane.ID}
}

func (store ShellStore) closePaneWorkbench(command WorkbenchCommand) (ShellStore, WorkbenchCommandResult) {
	target := store.workbenchPaneTarget(command)
	return store.removePaneWorkbench(command.Action, target, nil)
}

func (store ShellStore) killPane(command WorkbenchCommand) (ShellStore, WorkbenchCommandResult) {
	if command.Confirm != PaneConfirmAccepted {
		return store, WorkbenchCommandResult{Status: WorkbenchCommandNeedsConfirmation, Action: command.Action, Reason: "confirm pane kill"}
	}
	var targetOK bool
	store, targetOK = store.withCommandWorkspace(command)
	if !targetOK {
		return store, workbenchCommandInvalid(command.Action, "workspace not found")
	}
	target := store.workbenchPaneTarget(command)
	pane, ok := store.Pane(target)
	if !ok {
		return store, workbenchCommandInvalid(command.Action, "pane not found")
	}
	killed := []string{}
	if pane.TerminalID != "" {
		killed = []string{pane.TerminalID}
	}
	return store.removePaneWorkbench(command.Action, target, killed)
}

func (store ShellStore) workbenchPaneTarget(command WorkbenchCommand) PaneCommandTarget {
	target := command.Target
	if target.PaneID == "" {
		target.PaneID = strings.TrimSpace(command.TargetID)
	}
	if target.PaneID == "" {
		target.PaneID = store.EnsureDefaults().ActivePaneID
	}
	if target.WorkspaceID == "" {
		target.WorkspaceID = store.EnsureDefaults().Workspace.ID
	}
	if target.TabID == "" {
		target.TabID = store.EnsureDefaults().Workspace.ActiveTabID
	}
	return target
}

func (store ShellStore) setPaneDetached(target PaneCommandTarget) ShellStore {
	store = store.EnsureDefaults()
	tabIndex := store.tabIndexForTarget(target)
	if tabIndex < 0 {
		return store
	}
	paneID := target.PaneID
	if paneID == "" {
		paneID = store.ActivePaneID
	}
	for index := range store.Workspace.Tabs[tabIndex].Panes {
		pane := &store.Workspace.Tabs[tabIndex].Panes[index]
		if pane.ID != paneID {
			continue
		}
		pane.TerminalID = ""
		pane.Kind = PaneEmpty
		if pane.Title == "" {
			pane.Title = "empty"
		}
		store.Workspace = store.Workspace.ensureActive(store.ActivePaneID)
		store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
		return store
	}
	return store
}

func (store ShellStore) removePaneWorkbench(action WorkbenchCommandAction, target PaneCommandTarget, killed []string) (ShellStore, WorkbenchCommandResult) {
	if target.WorkspaceID != "" {
		var ok bool
		store, ok = store.withCommandWorkspace(WorkbenchCommand{Action: action, Target: target})
		if !ok {
			return store, workbenchCommandInvalid(action, "workspace not found")
		}
	}
	target = store.workbenchPaneTarget(WorkbenchCommand{Target: target})
	pane, ok := store.Pane(target)
	if !ok {
		return store, workbenchCommandInvalid(action, "pane not found")
	}
	if store.paneCountForTarget(target) <= 1 {
		return store, workbenchCommandInvalid(action, "cannot close last pane")
	}
	store = store.ClosePane(target)
	return store, WorkbenchCommandResult{Status: WorkbenchCommandOK, Action: action, ID: pane.ID, Killed: killed}
}

package state

import (
	"fmt"
	"strings"
)

func (store ShellStore) ApplyWorkbenchCommand(command WorkbenchCommand) (ShellStore, WorkbenchCommandResult) {
	store = store.EnsureDefaults()
	if command.Source == "" {
		command.Source = PaneCommandSourceKeyboard
	}
	switch command.Action {
	case WorkbenchCommandTabCreate:
		return store.createTab(command)
	case WorkbenchCommandTabSwitch:
		return store.switchTab(command)
	case WorkbenchCommandTabNext:
		return store.switchRelativeTab(1, command.Action)
	case WorkbenchCommandTabPrevious:
		return store.switchRelativeTab(-1, command.Action)
	case WorkbenchCommandTabRename:
		return store.renameTab(command)
	case WorkbenchCommandTabClose:
		return store.closeTab(command)
	case WorkbenchCommandTabKill:
		return store.killTab(command)
	case WorkbenchCommandWorkspaceCreate:
		return store.createWorkspace(command)
	case WorkbenchCommandWorkspaceSwitch:
		return store.switchWorkspace(command.TargetID, command.Action)
	case WorkbenchCommandWorkspaceNext:
		return store.switchRelativeWorkspace(1, command.Action)
	case WorkbenchCommandWorkspacePrevious:
		return store.switchRelativeWorkspace(-1, command.Action)
	case WorkbenchCommandWorkspaceRename:
		return store.renameWorkspace(command)
	case WorkbenchCommandWorkspaceDelete:
		return store.deleteWorkspace(command)
	case WorkbenchCommandPaneSplit:
		return store.splitPaneWorkbench(command)
	case WorkbenchCommandPaneRename:
		return store.renamePane(command)
	case WorkbenchCommandPaneDetach:
		return store.detachPane(command)
	case WorkbenchCommandPaneClose:
		return store.closePaneWorkbench(command)
	case WorkbenchCommandPaneKill:
		return store.killPane(command)
	default:
		return store, workbenchCommandInvalid(command.Action, "unknown action")
	}
}

func (store ShellStore) createTab(command WorkbenchCommand) (ShellStore, WorkbenchCommandResult) {
	var ok bool
	store, ok = store.withCommandWorkspace(command)
	if !ok {
		return store, workbenchCommandInvalid(command.Action, "workspace not found")
	}
	id := strings.TrimSpace(command.TargetID)
	if id == "" {
		id = nextTabID(store.Workspace)
	}
	if store.tabIndexByID(id) >= 0 {
		return store, workbenchCommandInvalid(command.Action, "tab already exists")
	}
	name := strings.TrimSpace(command.Name)
	if name == "" {
		name = id
	}
	paneID := id + "-pane"
	tab := TabState{
		ID:           id,
		Title:        name,
		ActivePaneID: paneID,
		// 新 tab 对齐 tuiv2：先创建可连接槽位，terminal 由 picker/create 后续显式绑定。
		Panes:     []PaneState{{ID: paneID, Title: "unconnected", Kind: PaneEmpty, Active: true}},
		RootSplit: SplitNode{PaneID: paneID},
	}
	store.Workspace.Tabs = append(cloneTabs(store.Workspace.Tabs), tab)
	return store.focusTabByIndex(len(store.Workspace.Tabs) - 1), WorkbenchCommandResult{Status: WorkbenchCommandOK, Action: command.Action, ID: id}
}

func (store ShellStore) EnsureActiveTabForAttach() ShellStore {
	store = store.EnsureDefaults()
	if len(store.Workspace.Tabs) > 0 {
		return store
	}
	store.Workspace.Tabs = []TabState{{ID: DefaultTabID, Title: "main"}}
	store.Workspace.ActiveTabID = DefaultTabID
	store.ActivePaneID = ""
	store.ZoomedPaneID = ""
	store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
	return store.EnsureDefaults()
}

func (store ShellStore) switchRelativeTab(offset int, action WorkbenchCommandAction) (ShellStore, WorkbenchCommandResult) {
	if len(store.Workspace.Tabs) == 0 {
		return store, workbenchCommandInvalid(action, "no tab")
	}
	index := store.activeTabIndex()
	if index < 0 {
		index = 0
	}
	next := (index + offset) % len(store.Workspace.Tabs)
	if next < 0 {
		next += len(store.Workspace.Tabs)
	}
	store = store.focusTabByIndex(next)
	return store, WorkbenchCommandResult{Status: WorkbenchCommandOK, Action: action, ID: store.Workspace.ActiveTabID}
}

func (store ShellStore) switchTab(command WorkbenchCommand) (ShellStore, WorkbenchCommandResult) {
	var ok bool
	store, ok = store.withCommandWorkspace(command)
	if !ok {
		return store, workbenchCommandInvalid(command.Action, "workspace not found")
	}
	if len(store.Workspace.Tabs) == 0 {
		return store, workbenchCommandInvalid(command.Action, "no tab")
	}
	index := store.tabIndexByID(command.TargetID)
	if index < 0 {
		return store, workbenchCommandInvalid(command.Action, "tab not found")
	}
	store = store.focusTabByIndex(index)
	return store, WorkbenchCommandResult{Status: WorkbenchCommandOK, Action: command.Action, ID: store.Workspace.ActiveTabID}
}

func (store ShellStore) renameTab(command WorkbenchCommand) (ShellStore, WorkbenchCommandResult) {
	var ok bool
	store, ok = store.withCommandWorkspace(command)
	if !ok {
		return store, workbenchCommandInvalid(command.Action, "workspace not found")
	}
	name := strings.TrimSpace(command.Name)
	if name == "" {
		return store, workbenchCommandInvalid(command.Action, "missing tab name")
	}
	index := store.activeTabIndex()
	if command.TargetID != "" {
		index = store.tabIndexByID(command.TargetID)
	}
	if index < 0 {
		return store, workbenchCommandInvalid(command.Action, "tab not found")
	}
	store.Workspace.Tabs[index].Title = name
	store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
	return store.EnsureDefaults(), WorkbenchCommandResult{Status: WorkbenchCommandOK, Action: command.Action, ID: store.Workspace.Tabs[index].ID}
}

func (store ShellStore) closeTab(command WorkbenchCommand) (ShellStore, WorkbenchCommandResult) {
	var ok bool
	store, ok = store.withCommandWorkspace(command)
	if !ok {
		return store, workbenchCommandInvalid(command.Action, "workspace not found")
	}
	index := store.activeTabIndex()
	if command.TargetID != "" {
		index = store.tabIndexByID(command.TargetID)
	}
	if index < 0 {
		return store, workbenchCommandInvalid(command.Action, "tab not found")
	}
	closedID := store.Workspace.Tabs[index].ID
	nextTabs := make([]TabState, 0, len(store.Workspace.Tabs)-1)
	for i, tab := range store.Workspace.Tabs {
		if i != index {
			nextTabs = append(nextTabs, tab)
		}
	}
	store.Workspace.Tabs = nextTabs
	if len(nextTabs) == 0 {
		store.Workspace.ActiveTabID = ""
		store.ActivePaneID = ""
		store.ZoomedPaneID = ""
		store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
		return store.EnsureDefaults(), WorkbenchCommandResult{Status: WorkbenchCommandOK, Action: command.Action, ID: closedID}
	}
	if index >= len(nextTabs) {
		index = len(nextTabs) - 1
	}
	store = store.focusTabByIndex(index)
	store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
	return store, WorkbenchCommandResult{Status: WorkbenchCommandOK, Action: command.Action, ID: closedID}
}

func (store ShellStore) killTab(command WorkbenchCommand) (ShellStore, WorkbenchCommandResult) {
	if command.Confirm != PaneConfirmAccepted {
		return store, WorkbenchCommandResult{Status: WorkbenchCommandNeedsConfirmation, Action: command.Action, Reason: "confirm tab kill"}
	}
	var ok bool
	store, ok = store.withCommandWorkspace(command)
	if !ok {
		return store, workbenchCommandInvalid(command.Action, "workspace not found")
	}
	index := store.activeTabIndex()
	if command.TargetID != "" {
		index = store.tabIndexByID(command.TargetID)
	}
	if index < 0 {
		return store, workbenchCommandInvalid(command.Action, "tab not found")
	}
	killed := terminalIDsForTab(store.Workspace.Tabs[index])
	next, result := store.closeTab(WorkbenchCommand{
		Action:   WorkbenchCommandTabClose,
		TargetID: command.TargetID,
		Target:   command.Target,
		Source:   command.Source,
	})
	if result.Status != WorkbenchCommandOK {
		result.Action = command.Action
		result.Killed = nil
		return store, result
	}
	result.Action = command.Action
	result.Killed = killed
	return next, result
}

func (store ShellStore) createWorkspace(command WorkbenchCommand) (ShellStore, WorkbenchCommandResult) {
	id := strings.TrimSpace(command.TargetID)
	if id == "" {
		id = nextWorkspaceID(store.Workspaces)
	}
	if _, ok := workspaceByID(store.Workspaces, id); ok {
		return store, workbenchCommandInvalid(command.Action, "workspace already exists")
	}
	name := strings.TrimSpace(command.Name)
	if name == "" {
		name = id
	}
	paneID := id + "-pane"
	workspace := WorkspaceState{
		ID:          id,
		Name:        name,
		ActiveTabID: DefaultTabID,
		Tabs: []TabState{{
			ID:           DefaultTabID,
			Title:        "main",
			ActivePaneID: paneID,
			// 新 workspace 只创建可连接槽位；terminal 必须由用户显式 attach/create。
			Panes:     []PaneState{{ID: paneID, Title: "unconnected", Kind: PaneEmpty, Active: true}},
			RootSplit: SplitNode{PaneID: paneID},
		}},
	}
	store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
	store.Workspaces = appendWorkspaceIfMissing(store.Workspaces, workspace.ensureDefaults())
	return store.switchWorkspace(id, command.Action)
}

func (store ShellStore) switchRelativeWorkspace(offset int, action WorkbenchCommandAction) (ShellStore, WorkbenchCommandResult) {
	store.Workspaces = upsertWorkspace(ensureWorkspaceList(store.Workspaces, store.Workspace), store.Workspace)
	if len(store.Workspaces) == 0 {
		return store, workbenchCommandInvalid(action, "no workspace")
	}
	current := workspaceIndexByID(store.Workspaces, store.Workspace.ID)
	if current < 0 {
		current = 0
	}
	next := (current + offset) % len(store.Workspaces)
	if next < 0 {
		next += len(store.Workspaces)
	}
	return store.switchWorkspace(store.Workspaces[next].ID, action)
}

func (store ShellStore) renameWorkspace(command WorkbenchCommand) (ShellStore, WorkbenchCommandResult) {
	name := strings.TrimSpace(command.Name)
	if name == "" {
		return store, workbenchCommandInvalid(command.Action, "missing workspace name")
	}
	id := command.TargetID
	if id == "" {
		id = store.Workspace.ID
	}
	index := workspaceIndexByID(store.Workspaces, id)
	if index < 0 {
		return store, workbenchCommandInvalid(command.Action, "workspace not found")
	}
	store.Workspaces[index].Name = name
	if store.Workspace.ID == id {
		store.Workspace.Name = name
	}
	store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
	return store.EnsureDefaults(), WorkbenchCommandResult{Status: WorkbenchCommandOK, Action: command.Action, ID: id}
}

func (store ShellStore) deleteWorkspace(command WorkbenchCommand) (ShellStore, WorkbenchCommandResult) {
	if command.Confirm != PaneConfirmAccepted {
		return store, WorkbenchCommandResult{Status: WorkbenchCommandNeedsConfirmation, Action: command.Action, Reason: "confirm workspace delete"}
	}
	store.Workspaces = upsertWorkspace(ensureWorkspaceList(store.Workspaces, store.Workspace), store.Workspace)
	if len(store.Workspaces) <= 1 {
		return store, workbenchCommandInvalid(command.Action, "cannot delete last workspace")
	}
	id := strings.TrimSpace(command.TargetID)
	if id == "" {
		id = store.Workspace.ID
	}
	index := workspaceIndexByID(store.Workspaces, id)
	if index < 0 {
		return store, workbenchCommandInvalid(command.Action, "workspace not found")
	}
	deletedID := store.Workspaces[index].ID
	nextWorkspaces := make([]WorkspaceState, 0, len(store.Workspaces)-1)
	for i, workspace := range store.Workspaces {
		if i != index {
			nextWorkspaces = append(nextWorkspaces, workspace)
		}
	}
	store.Workspaces = nextWorkspaces
	if store.Workspace.ID == deletedID {
		if index >= len(nextWorkspaces) {
			index = len(nextWorkspaces) - 1
		}
		// 删除当前 workspace 时不能走 switchWorkspace：它会先把旧 active workspace upsert 回列表。
		next := cloneWorkspace(nextWorkspaces[index]).ensureDefaults()
		store.Workspace = next
		store.ActivePaneID = next.activeTab().ActivePaneID
		store.ZoomedPaneID = ""
		store.Workspace = store.Workspace.ensureActive(store.ActivePaneID)
		store.Workspaces = upsertWorkspace(nextWorkspaces, store.Workspace)
		return store.EnsureDefaults(), WorkbenchCommandResult{Status: WorkbenchCommandOK, Action: command.Action, ID: deletedID}
	}
	store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
	return store.EnsureDefaults(), WorkbenchCommandResult{Status: WorkbenchCommandOK, Action: command.Action, ID: deletedID}
}

func (store ShellStore) switchWorkspace(id string, action WorkbenchCommandAction) (ShellStore, WorkbenchCommandResult) {
	index := workspaceIndexByID(store.Workspaces, id)
	if index < 0 {
		return store, workbenchCommandInvalid(action, "workspace not found")
	}
	store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
	next := cloneWorkspace(store.Workspaces[index]).ensureDefaults()
	store.Workspace = next
	store.ActivePaneID = next.activeTab().ActivePaneID
	store.ZoomedPaneID = ""
	store.Workspace = store.Workspace.ensureActive(store.ActivePaneID)
	store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
	return store.EnsureDefaults(), WorkbenchCommandResult{Status: WorkbenchCommandOK, Action: action, ID: id}
}

func (store ShellStore) withCommandWorkspace(command WorkbenchCommand) (ShellStore, bool) {
	store = store.EnsureDefaults()
	workspaceID := strings.TrimSpace(command.Target.WorkspaceID)
	if workspaceID == "" || workspaceID == store.Workspace.ID {
		return store, true
	}
	// Workbench Navigator 会对非当前 workspace 的节点发命令，执行前必须先切到目标 workspace。
	next, result := store.switchWorkspace(workspaceID, command.Action)
	if result.Status != WorkbenchCommandOK {
		return store, false
	}
	return next, true
}

func (store ShellStore) focusTabByIndex(index int) ShellStore {
	if index < 0 || index >= len(store.Workspace.Tabs) {
		store.Workspace.ActiveTabID = ""
		store.ActivePaneID = ""
		store.ZoomedPaneID = ""
		store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
		return store.EnsureDefaults()
	}
	tab := store.Workspace.Tabs[index]
	store.Workspace.ActiveTabID = tab.ID
	if tab.ActivePaneID == "" && len(tab.Panes) > 0 {
		tab.ActivePaneID = tab.Panes[0].ID
		store.Workspace.Tabs[index] = tab
	}
	store.ActivePaneID = tab.ActivePaneID
	store.ZoomedPaneID = ""
	store.Workspace = store.Workspace.ensureActive(store.ActivePaneID)
	store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
	return store.EnsureDefaults()
}

func (store ShellStore) tabIndexByID(id string) int {
	for index, tab := range store.Workspace.Tabs {
		if tab.ID == id {
			return index
		}
	}
	return -1
}

func ensureWorkspaceList(workspaces []WorkspaceState, active WorkspaceState) []WorkspaceState {
	active = active.ensureDefaults()
	if len(workspaces) == 0 {
		return []WorkspaceState{active}
	}
	out := cloneWorkspaces(workspaces)
	for index := range out {
		out[index] = out[index].ensureDefaults()
	}
	return out
}

func workspaceByID(workspaces []WorkspaceState, id string) (WorkspaceState, bool) {
	index := workspaceIndexByID(workspaces, id)
	if index < 0 {
		return WorkspaceState{}, false
	}
	return cloneWorkspace(workspaces[index]).ensureDefaults(), true
}

func workspaceIndexByID(workspaces []WorkspaceState, id string) int {
	for index, workspace := range workspaces {
		if workspace.ID == id {
			return index
		}
	}
	return -1
}

func upsertWorkspace(workspaces []WorkspaceState, workspace WorkspaceState) []WorkspaceState {
	workspace = workspace.ensureDefaults()
	out := cloneWorkspaces(workspaces)
	if len(out) == 0 {
		return []WorkspaceState{workspace}
	}
	for index := range out {
		if out[index].ID == workspace.ID {
			out[index] = workspace
			return out
		}
	}
	return append(out, workspace)
}

func appendWorkspaceIfMissing(workspaces []WorkspaceState, workspace WorkspaceState) []WorkspaceState {
	if _, ok := workspaceByID(workspaces, workspace.ID); ok {
		return cloneWorkspaces(workspaces)
	}
	return append(cloneWorkspaces(workspaces), workspace.ensureDefaults())
}

func terminalIDsForTab(tab TabState) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, pane := range tab.Panes {
		if pane.TerminalID == "" {
			continue
		}
		if _, ok := seen[pane.TerminalID]; ok {
			continue
		}
		seen[pane.TerminalID] = struct{}{}
		out = append(out, pane.TerminalID)
	}
	return out
}

func nextTabID(workspace WorkspaceState) string {
	for i := 2; ; i++ {
		id := fmt.Sprintf("tab-%d", i)
		exists := false
		for _, tab := range workspace.Tabs {
			if tab.ID == id {
				exists = true
				break
			}
		}
		if !exists {
			return id
		}
	}
}

func nextWorkspaceID(workspaces []WorkspaceState) string {
	for i := 2; ; i++ {
		id := fmt.Sprintf("workspace-%d", i)
		if workspaceIndexByID(workspaces, id) < 0 {
			return id
		}
	}
}

func paneTitle(pane PaneState) string {
	if pane.Title != "" {
		return pane.Title
	}
	if pane.TerminalID != "" {
		return pane.TerminalID
	}
	if pane.ID != "" {
		return pane.ID
	}
	return "pane"
}

func terminalPoolTitle(item TerminalPoolItem) string {
	if item.Title != "" {
		return item.Title
	}
	if item.TerminalID != "" {
		return item.TerminalID
	}
	return "terminal"
}

func workbenchCommandInvalid(action WorkbenchCommandAction, reason string) WorkbenchCommandResult {
	return WorkbenchCommandResult{Status: WorkbenchCommandInvalid, Action: action, Reason: reason}
}

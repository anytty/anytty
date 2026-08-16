package testkit

import "github.com/anytty/anytty/tui/state"

func cloneWorkbenchStorageSnapshot(snapshot state.WorkbenchStorageSnapshot) state.WorkbenchStorageSnapshot {
	clone := snapshot
	clone.Workspace = cloneWorkspaceState(snapshot.Workspace)
	clone.Workspaces = cloneWorkspaceStates(snapshot.Workspaces)
	return clone
}

func cloneClipboardStorageSnapshot(snapshot state.ClipboardStorageSnapshot) state.ClipboardStorageSnapshot {
	clone := snapshot
	if len(snapshot.Entries) > 0 {
		clone.Entries = append([]state.ClipboardEntry(nil), snapshot.Entries...)
	}
	return clone
}

func cloneWorkspaceState(workspace state.WorkspaceState) state.WorkspaceState {
	workspace.Tabs = cloneTabStates(workspace.Tabs)
	return workspace
}

func cloneWorkspaceStates(workspaces []state.WorkspaceState) []state.WorkspaceState {
	if len(workspaces) == 0 {
		return nil
	}
	out := make([]state.WorkspaceState, len(workspaces))
	for i, workspace := range workspaces {
		out[i] = cloneWorkspaceState(workspace)
	}
	return out
}

func cloneTabStates(tabs []state.TabState) []state.TabState {
	if len(tabs) == 0 {
		return nil
	}
	out := make([]state.TabState, len(tabs))
	for i, tab := range tabs {
		out[i] = tab
		out[i].Panes = clonePaneStates(tab.Panes)
		out[i].RootSplit = cloneSplitNode(tab.RootSplit)
		out[i].Floatings = cloneFloatingPaneStates(tab.Floatings)
	}
	return out
}

func clonePaneStates(panes []state.PaneState) []state.PaneState {
	if len(panes) == 0 {
		return nil
	}
	out := make([]state.PaneState, len(panes))
	copy(out, panes)
	return out
}

func cloneFloatingPaneStates(floatings []state.FloatingPaneState) []state.FloatingPaneState {
	if len(floatings) == 0 {
		return nil
	}
	out := make([]state.FloatingPaneState, len(floatings))
	for i, floating := range floatings {
		out[i] = floating
	}
	return out
}

func cloneSplitNode(node state.SplitNode) state.SplitNode {
	if len(node.Children) == 0 {
		return node
	}
	node.Children = append([]state.SplitNode(nil), node.Children...)
	for i := range node.Children {
		node.Children[i] = cloneSplitNode(node.Children[i])
	}
	return node
}

package state

func cloneToasts(toasts []ToastState) []ToastState {
	if len(toasts) == 0 {
		return nil
	}
	cloned := make([]ToastState, len(toasts))
	copy(cloned, toasts)
	return cloned
}

func cloneWorkspace(workspace WorkspaceState) WorkspaceState {
	workspace.Tabs = cloneTabs(workspace.Tabs)
	return workspace
}

func cloneWorkspaces(workspaces []WorkspaceState) []WorkspaceState {
	if len(workspaces) == 0 {
		return nil
	}
	cloned := make([]WorkspaceState, len(workspaces))
	for index, workspace := range workspaces {
		cloned[index] = cloneWorkspace(workspace)
	}
	return cloned
}

func cloneFloatings(floatings []FloatingPaneState) []FloatingPaneState {
	if len(floatings) == 0 {
		return nil
	}
	cloned := make([]FloatingPaneState, len(floatings))
	copy(cloned, floatings)
	return cloned
}

func cloneTabs(tabs []TabState) []TabState {
	if len(tabs) == 0 {
		return nil
	}
	cloned := make([]TabState, len(tabs))
	for i, tab := range tabs {
		cloned[i] = tab
		cloned[i].Panes = clonePanes(tab.Panes)
		cloned[i].RootSplit = cloneSplitNode(tab.RootSplit)
		cloned[i].Floatings = cloneFloatings(tab.Floatings)
	}
	return cloned
}

func clonePanes(panes []PaneState) []PaneState {
	if len(panes) == 0 {
		return nil
	}
	cloned := make([]PaneState, len(panes))
	copy(cloned, panes)
	return cloned
}

func cloneSplitNode(node SplitNode) SplitNode {
	node.Children = cloneSplitNodes(node.Children)
	return node
}

func cloneSplitNodes(nodes []SplitNode) []SplitNode {
	if len(nodes) == 0 {
		return nil
	}
	cloned := make([]SplitNode, len(nodes))
	for i, node := range nodes {
		cloned[i] = cloneSplitNode(node)
	}
	return cloned
}

package state

// RemoveTerminalRefBindings 按 TerminalRef 清理 pane/floating 的连接槽位。
// Shell 自身只保存 daemon-local TerminalID，跨 endpoint 判断必须借助调用时仍存在的
// TerminalViewStore；缺少 view binding 的旧数据只按默认 local endpoint 兼容清理。
func (store ShellStore) RemoveTerminalRefBindings(ref TerminalRef, views TerminalViewStore) ShellStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store.EnsureDefaults()
	}
	store = store.EnsureDefaults()
	store.Workspace = store.Workspace.removeTerminalRefBindings(ref, views)
	for index := range store.Workspaces {
		store.Workspaces[index] = store.Workspaces[index].removeTerminalRefBindings(ref, views)
		if store.Workspaces[index].ID == store.Workspace.ID {
			store.Workspaces[index] = store.Workspace
		}
	}
	return store.EnsureDefaults()
}

func (workspace WorkspaceState) removeTerminalRefBindings(ref TerminalRef, views TerminalViewStore) WorkspaceState {
	workspace = cloneWorkspace(workspace)
	for tabIndex := range workspace.Tabs {
		for paneIndex := range workspace.Tabs[tabIndex].Panes {
			pane := &workspace.Tabs[tabIndex].Panes[paneIndex]
			if !paneMatchesTerminalRef(*pane, ref, views) {
				continue
			}
			pane.TerminalID = ""
			pane.Kind = PaneEmpty
		}
		for floatingIndex := range workspace.Tabs[tabIndex].Floatings {
			floating := &workspace.Tabs[tabIndex].Floatings[floatingIndex]
			if !floatingMatchesTerminalRef(*floating, ref, views) {
				continue
			}
			floating.Pane.TerminalID = ""
			floating.Pane.Kind = PaneEmpty
		}
	}
	return workspace
}

func paneMatchesTerminalRef(pane PaneState, ref TerminalRef, views TerminalViewStore) bool {
	if binding, ok := views.PaneBinding(pane.ID); ok {
		return binding.TerminalRef().Equal(ref)
	}
	return ref.EndpointID == DefaultEndpointID && pane.TerminalID == ref.TerminalID
}

func floatingMatchesTerminalRef(floating FloatingPaneState, ref TerminalRef, views TerminalViewStore) bool {
	if binding, ok := views.FloatingBinding(floating.ID); ok {
		return binding.TerminalRef().Equal(ref)
	}
	return ref.EndpointID == DefaultEndpointID && floating.Pane.TerminalID == ref.TerminalID
}

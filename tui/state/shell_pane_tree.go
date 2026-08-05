package state

import (
	"strconv"
	"strings"
)

func (store ShellStore) SplitActivePane(newPane PaneState, direction SplitDirection) ShellStore {
	if direction != SplitDirectionHorizontal && direction != SplitDirectionVertical {
		return store.EnsureDefaults()
	}
	store = store.EnsureDefaults()
	if newPane.ID == "" {
		return store
	}
	tabIndex := store.activeTabIndex()
	if tabIndex < 0 {
		return store
	}
	tab := &store.Workspace.Tabs[tabIndex]
	for _, pane := range tab.Panes {
		if pane.ID == newPane.ID {
			return store
		}
	}
	if newPane.Title == "" {
		newPane.Title = newPane.ID
	}
	if newPane.Kind == "" {
		newPane.Kind = PaneEmpty
	}
	previousActive := tab.ActivePaneID
	if previousActive == "" {
		previousActive = store.ActivePaneID
	}
	if len(tab.Panes) == 0 {
		tab.Panes = []PaneState{newPane}
		tab.RootSplit = SplitNode{PaneID: newPane.ID}
		tab.ActivePaneID = newPane.ID
		store.ActivePaneID = newPane.ID
		store.ZoomedPaneID = ""
		store.Workspace = store.Workspace.ensureActive(store.ActivePaneID)
		return store
	}
	tab.Panes = append(clonePanes(tab.Panes), newPane)
	tab.RootSplit = insertSplitNode(tab.RootSplit, previousActive, newPane.ID, direction)
	tab.ActivePaneID = newPane.ID
	store.ActivePaneID = newPane.ID
	store.ZoomedPaneID = ""
	store.Workspace = store.Workspace.ensureActive(store.ActivePaneID)
	return store
}

func (store ShellStore) FocusPane(target PaneCommandTarget) ShellStore {
	store = store.EnsureDefaults()
	if target.WorkspaceID != "" && target.WorkspaceID != store.Workspace.ID {
		// focus 是用户显式打开节点，跨 workspace 时先切 workspace 再定位 pane。
		next, result := store.switchWorkspace(target.WorkspaceID, WorkbenchCommandWorkspaceSwitch)
		if result.Status != WorkbenchCommandOK {
			return store
		}
		store = next
	}
	if !store.HasPane(target) {
		return store
	}
	tabIndex := store.tabIndexForTarget(target)
	if tabIndex < 0 {
		return store
	}
	paneID := target.PaneID
	store.Workspace.ActiveTabID = store.Workspace.Tabs[tabIndex].ID
	store.Workspace.Tabs[tabIndex].ActivePaneID = paneID
	store.Workspace.Tabs[tabIndex].ActiveFloatingID = ""
	store.ActivePaneID = paneID
	if store.ZoomedPaneID != "" {
		store.ZoomedPaneID = paneID
	}
	store.Workspace = store.Workspace.ensureActive(store.ActivePaneID)
	return store
}

func (store ShellStore) BindPaneTerminal(target PaneCommandTarget, terminalID string) ShellStore {
	store = store.EnsureDefaults()
	if terminalID == "" {
		return store
	}
	tabIndex := store.tabIndexForTarget(target)
	if tabIndex < 0 && target.PaneID == "" {
		tabIndex = store.activeTabIndex()
	}
	if tabIndex < 0 {
		return store
	}
	paneID := target.PaneID
	if paneID == "" {
		paneID = store.ActivePaneID
	}
	if paneID == "" && len(store.Workspace.Tabs[tabIndex].Panes) == 0 {
		paneID = store.Workspace.Tabs[tabIndex].ID + "-pane"
		store.Workspace.Tabs[tabIndex].Panes = []PaneState{{ID: paneID, Title: terminalID, Kind: PaneTerminalLive, TerminalID: terminalID}}
		store.Workspace.Tabs[tabIndex].ActivePaneID = paneID
		store.Workspace.Tabs[tabIndex].RootSplit = SplitNode{PaneID: paneID}
		store.ActivePaneID = paneID
		store.Workspace = store.Workspace.ensureActive(store.ActivePaneID)
		store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
		return store
	}
	for index := range store.Workspace.Tabs[tabIndex].Panes {
		if store.Workspace.Tabs[tabIndex].Panes[index].ID != paneID {
			continue
		}
		store.Workspace.Tabs[tabIndex].Panes[index].TerminalID = terminalID
		store.Workspace.Tabs[tabIndex].Panes[index].Kind = PaneTerminalLive
		if store.Workspace.Tabs[tabIndex].Panes[index].Title == "" {
			store.Workspace.Tabs[tabIndex].Panes[index].Title = terminalID
		}
		store.Workspace = store.Workspace.ensureActive(store.ActivePaneID)
		store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
		return store
	}
	return store
}

func (store ShellStore) RemoveTerminalBindings(terminalID string) ShellStore {
	if terminalID == "" {
		return store.EnsureDefaults()
	}
	store = store.EnsureDefaults()
	store.Workspace = store.Workspace.removeTerminalBindings(terminalID)
	for index := range store.Workspaces {
		store.Workspaces[index] = store.Workspaces[index].removeTerminalBindings(terminalID)
		if store.Workspaces[index].ID == store.Workspace.ID {
			store.Workspaces[index] = store.Workspace
		}
	}
	return store.EnsureDefaults()
}

func (workspace WorkspaceState) removeTerminalBindings(terminalID string) WorkspaceState {
	workspace = cloneWorkspace(workspace)
	for tabIndex := range workspace.Tabs {
		for paneIndex := range workspace.Tabs[tabIndex].Panes {
			if workspace.Tabs[tabIndex].Panes[paneIndex].TerminalID != terminalID {
				continue
			}
			workspace.Tabs[tabIndex].Panes[paneIndex].TerminalID = ""
			workspace.Tabs[tabIndex].Panes[paneIndex].Kind = PaneEmpty
		}
		for floatingIndex := range workspace.Tabs[tabIndex].Floatings {
			if workspace.Tabs[tabIndex].Floatings[floatingIndex].Pane.TerminalID != terminalID {
				continue
			}
			workspace.Tabs[tabIndex].Floatings[floatingIndex].Pane.TerminalID = ""
			workspace.Tabs[tabIndex].Floatings[floatingIndex].Pane.Kind = PaneEmpty
		}
	}
	return workspace
}

func (store ShellStore) FocusRelativePane(offset int) ShellStore {
	store = store.EnsureDefaults()
	if offset == 0 {
		return store
	}
	tabIndex := store.activeTabIndex()
	if tabIndex < 0 {
		return store
	}
	tab := store.Workspace.Tabs[tabIndex]
	if len(tab.Panes) <= 1 {
		return store
	}
	current := 0
	for i, pane := range tab.Panes {
		if pane.ID == store.ActivePaneID {
			current = i
			break
		}
	}
	next := (current + offset) % len(tab.Panes)
	if next < 0 {
		next += len(tab.Panes)
	}
	return store.FocusPane(PaneCommandTarget{WorkspaceID: store.Workspace.ID, TabID: tab.ID, PaneID: tab.Panes[next].ID})
}

func (store ShellStore) ClosePane(target PaneCommandTarget) ShellStore {
	store = store.EnsureDefaults()
	tabIndex := store.tabIndexForTarget(target)
	if tabIndex < 0 {
		return store
	}
	tab := &store.Workspace.Tabs[tabIndex]
	if len(tab.Panes) <= 1 {
		return store
	}
	paneID := target.PaneID
	nextPanes := make([]PaneState, 0, len(tab.Panes)-1)
	for _, pane := range tab.Panes {
		if pane.ID != paneID {
			nextPanes = append(nextPanes, pane)
		}
	}
	if len(nextPanes) == len(tab.Panes) || len(nextPanes) == 0 {
		return store
	}
	tab.Panes = nextPanes
	if nextRoot, ok := removePaneFromSplit(tab.RootSplit, paneID); ok {
		tab.RootSplit = nextRoot
	} else {
		tab.RootSplit = SplitNode{PaneID: nextPanes[0].ID}
	}
	if tab.ActivePaneID == paneID || store.ActivePaneID == paneID {
		tab.ActivePaneID = firstPaneIDInSplit(tab.RootSplit)
		if tab.ActivePaneID == "" {
			tab.ActivePaneID = nextPanes[0].ID
		}
		store.ActivePaneID = tab.ActivePaneID
	}
	if store.ZoomedPaneID == paneID {
		store.ZoomedPaneID = ""
	}
	store.Workspace = store.Workspace.ensureActive(store.ActivePaneID)
	store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
	return store
}

func (store ShellStore) ZoomPane(target PaneCommandTarget) ShellStore {
	store = store.FocusPane(target)
	if store.HasPane(target) {
		store.ZoomedPaneID = target.PaneID
	}
	return store
}

func (store ShellStore) UnzoomPane() ShellStore {
	store = store.EnsureDefaults()
	store.ZoomedPaneID = ""
	return store
}

func (store ShellStore) ToggleZoomPane(target PaneCommandTarget) ShellStore {
	store = store.EnsureDefaults()
	if store.ZoomedPaneID == target.PaneID && store.ZoomedPaneID != "" {
		return store.UnzoomPane()
	}
	return store.ZoomPane(target)
}

func (store ShellStore) ResizePane(target PaneCommandTarget, direction PaneResizeDirection, delta int) ShellStore {
	store = store.EnsureDefaults()
	if delta == 0 {
		return store
	}
	tabIndex := store.tabIndexForTarget(target)
	if tabIndex < 0 {
		return store
	}
	tab := &store.Workspace.Tabs[tabIndex]
	tab.RootSplit, _ = resizeSplitNode(tab.RootSplit, target.PaneID, direction, delta)
	return store
}

func (store ShellStore) ResizeSplitPath(target PaneCommandTarget, splitPath string, direction PaneResizeDirection, delta int) ShellStore {
	store = store.EnsureDefaults()
	if delta == 0 {
		return store
	}
	tabIndex := store.tabIndexForTarget(target)
	if tabIndex < 0 {
		return store
	}
	path, ok := parseSplitPath(splitPath)
	if !ok {
		return store
	}
	tab := &store.Workspace.Tabs[tabIndex]
	tab.RootSplit, _ = resizeSplitNodeByPath(tab.RootSplit, path, direction, delta)
	return store
}

func (store ShellStore) ResizePaneGroup(target PaneCommandTarget, direction PaneResizeDirection, group []PaneResizeGroupItem) ShellStore {
	store = store.EnsureDefaults()
	if len(group) < 2 {
		return store
	}
	tabIndex := store.tabIndexForTarget(target)
	if tabIndex < 0 {
		return store
	}
	axis, ok := splitDirectionForResize(direction)
	if !ok {
		return store
	}
	items := clonePaneResizeGroupItems(group)
	if !validPaneResizeGroup(items) {
		return store
	}
	tab := &store.Workspace.Tabs[tabIndex]
	next, changed := resizePaneGroupNode(tab.RootSplit, axis, items)
	if !changed {
		return store
	}
	tab.RootSplit = next
	return store
}

func (store ShellStore) SetPaneSize(command PaneCommand) ShellStore {
	store = store.EnsureDefaults()
	tabIndex := store.tabIndexForTarget(command.Target)
	if tabIndex < 0 {
		return store
	}
	tab := &store.Workspace.Tabs[tabIndex]
	tab.RootSplit, _ = setSplitNodeSize(tab.RootSplit, command)
	return store
}

func (store ShellStore) BalancePanes(target PaneCommandTarget) ShellStore {
	store = store.EnsureDefaults()
	tabIndex := store.tabIndexForTarget(target)
	if tabIndex < 0 {
		return store
	}
	tab := &store.Workspace.Tabs[tabIndex]
	tab.RootSplit = balanceSplitNode(tab.RootSplit)
	return store
}

func (store ShellStore) activeTab() TabState {
	for _, tab := range store.Workspace.Tabs {
		if tab.ID == store.Workspace.ActiveTabID {
			return tab
		}
	}
	if len(store.Workspace.Tabs) > 0 {
		return store.Workspace.Tabs[0]
	}
	return TabState{}
}

func (store ShellStore) activeTabIndex() int {
	for index, tab := range store.Workspace.Tabs {
		if tab.ID == store.Workspace.ActiveTabID {
			return index
		}
	}
	if len(store.Workspace.Tabs) > 0 {
		return 0
	}
	return -1
}

func (store ShellStore) PaneByID(paneID string) (PaneState, bool) {
	store = store.EnsureDefaults()
	for _, tab := range store.Workspace.Tabs {
		for _, pane := range tab.Panes {
			if pane.ID == paneID {
				return pane, true
			}
		}
	}
	return PaneState{}, false
}

func (store ShellStore) tabIndexForTarget(target PaneCommandTarget) int {
	store = store.EnsureDefaults()
	for index, tab := range store.Workspace.Tabs {
		if target.TabID != "" && tab.ID != target.TabID {
			continue
		}
		for _, pane := range tab.Panes {
			if pane.ID == target.PaneID {
				return index
			}
		}
	}
	return -1
}

func (store ShellStore) paneCountForTarget(target PaneCommandTarget) int {
	index := store.tabIndexForTarget(target)
	if index < 0 {
		return 0
	}
	return len(store.Workspace.Tabs[index].Panes)
}

func (store ShellStore) hasPaneInActiveTab(paneID string) bool {
	activeTabID := store.Workspace.ActiveTabID
	for _, tab := range store.Workspace.Tabs {
		if activeTabID != "" && tab.ID != activeTabID {
			continue
		}
		for _, pane := range tab.Panes {
			if pane.ID == paneID {
				return true
			}
		}
	}
	return false
}

func insertSplitNode(node SplitNode, targetPaneID string, newPaneID string, direction SplitDirection) SplitNode {
	if node.PaneID == targetPaneID || (node.PaneID == "" && len(node.Children) == 0) {
		return SplitNode{
			Direction: direction,
			Children: []SplitNode{
				{PaneID: targetPaneID},
				{PaneID: newPaneID},
			},
		}
	}
	children := cloneSplitNodes(node.Children)
	for i, child := range children {
		children[i] = insertSplitNode(child, targetPaneID, newPaneID, direction)
	}
	node.Children = children
	return node
}

func removePaneFromSplit(node SplitNode, paneID string) (SplitNode, bool) {
	if node.PaneID != "" || len(node.Children) == 0 {
		return node, node.PaneID != paneID
	}
	children := make([]SplitNode, 0, len(node.Children))
	for _, child := range node.Children {
		if next, keep := removePaneFromSplit(child, paneID); keep {
			children = append(children, next)
		}
	}
	if len(children) == 0 {
		return SplitNode{}, false
	}
	if len(children) == 1 {
		return children[0], true
	}
	node.Children = children
	return node, true
}

func firstPaneIDInSplit(node SplitNode) string {
	if node.PaneID != "" {
		return node.PaneID
	}
	for _, child := range node.Children {
		if paneID := firstPaneIDInSplit(child); paneID != "" {
			return paneID
		}
	}
	return ""
}

func resizeSplitNode(node SplitNode, paneID string, direction PaneResizeDirection, delta int) (SplitNode, bool) {
	if node.PaneID != "" || len(node.Children) < 2 {
		return node, node.PaneID == paneID
	}
	firstContains := splitContainsPane(node.Children[0], paneID)
	secondContains := splitContainsPane(node.Children[1], paneID)
	if firstContains || secondContains {
		if splitDirectionMatchesResize(node.Direction, direction) {
			node.BiasCells += resizeBiasDelta(node.Direction, direction, firstContains, delta)
			node.FixedPaneID = ""
			node.FixedCols = 0
			node.FixedRows = 0
			node.Ratio = 0
			return node, true
		}
		childIndex := 0
		if secondContains {
			childIndex = 1
		}
		children := cloneSplitNodes(node.Children)
		children[childIndex], _ = resizeSplitNode(children[childIndex], paneID, direction, delta)
		node.Children = children
		return node, true
	}
	children := cloneSplitNodes(node.Children)
	changed := false
	for i, child := range children {
		children[i], changed = resizeSplitNode(child, paneID, direction, delta)
		if changed {
			break
		}
	}
	node.Children = children
	return node, changed
}

func resizeSplitNodeByPath(node SplitNode, path []int, direction PaneResizeDirection, delta int) (SplitNode, bool) {
	if len(path) == 0 {
		if node.PaneID != "" || len(node.Children) < 2 || !splitDirectionMatchesResize(node.Direction, direction) {
			return node, false
		}
		// 鼠标拖拽 divider 时目标就是当前 split，delta 已按拖拽方向带符号；不能再按 pane 所在侧反推祖先。
		node.BiasCells += delta
		node.FixedPaneID = ""
		node.FixedCols = 0
		node.FixedRows = 0
		node.Ratio = 0
		return node, true
	}
	index := path[0]
	if index < 0 || index >= len(node.Children) {
		return node, false
	}
	children := cloneSplitNodes(node.Children)
	next, changed := resizeSplitNodeByPath(children[index], path[1:], direction, delta)
	if !changed {
		return node, false
	}
	children[index] = next
	node.Children = children
	return node, true
}

func parseSplitPath(path string) ([]int, bool) {
	if path == PaneResizeRootSplitPath {
		return nil, true
	}
	prefix := PaneResizeRootSplitPath + "/"
	if !strings.HasPrefix(path, prefix) {
		return nil, false
	}
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) == 0 {
		return nil, false
	}
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		index, err := strconv.Atoi(part)
		if err != nil || (index != 0 && index != 1) {
			return nil, false
		}
		out = append(out, index)
	}
	return out, true
}

func resizePaneGroupNode(node SplitNode, axis SplitDirection, group []PaneResizeGroupItem) (SplitNode, bool) {
	panes := paneIDsInSplitOrder(node, nil)
	if samePaneOrder(panes, group) {
		// 鼠标命中的是一条真实 divider；group 带有每个叶子在该轴上的目标尺寸。
		// 这里保留原有异轴 stacked 结构，只给同轴 split 写入固定尺寸 hint。
		return applyPaneResizeGroupNode(node, axis, paneResizeGroupByPane(group))
	}
	if node.PaneID != "" || len(node.Children) == 0 {
		return node, false
	}
	children := cloneSplitNodes(node.Children)
	for i, child := range children {
		next, changed := resizePaneGroupNode(child, axis, group)
		if changed {
			children[i] = next
			node.Children = children
			return node, true
		}
	}
	return node, false
}

func paneIDsInSplitOrder(node SplitNode, out []string) []string {
	if node.PaneID != "" {
		return append(out, node.PaneID)
	}
	for _, child := range node.Children {
		out = paneIDsInSplitOrder(child, out)
	}
	return out
}

func paneResizeGroupByPane(group []PaneResizeGroupItem) map[string]PaneResizeGroupItem {
	out := make(map[string]PaneResizeGroupItem, len(group))
	for _, item := range group {
		out[item.PaneID] = item
	}
	return out
}

func applyPaneResizeGroupNode(node SplitNode, axis SplitDirection, group map[string]PaneResizeGroupItem) (SplitNode, bool) {
	if node.PaneID != "" {
		_, ok := group[node.PaneID]
		return node, ok
	}
	children := cloneSplitNodes(node.Children)
	changed := false
	for i, child := range children {
		next, childChanged := applyPaneResizeGroupNode(child, axis, group)
		if childChanged {
			children[i] = next
			changed = true
		}
	}
	if !changed {
		return node, false
	}
	node.Children = children
	if node.Direction == axis && len(node.Children) >= 2 {
		firstExtent, firstOK := paneResizeGroupExtent(node.Children[0], axis, group)
		secondExtent, secondOK := paneResizeGroupExtent(node.Children[1], axis, group)
		if firstOK && secondOK && firstExtent > 0 && secondExtent > 0 {
			node.BiasCells = 0
			node.Ratio = 0
			node.FixedPaneID = firstPaneIDInSplit(node.Children[0])
			if axis == SplitDirectionVertical {
				node.FixedCols = firstExtent
				node.FixedRows = 0
			} else {
				node.FixedRows = firstExtent
				node.FixedCols = 0
			}
		}
	}
	return node, true
}

func paneResizeGroupExtent(node SplitNode, axis SplitDirection, group map[string]PaneResizeGroupItem) (int, bool) {
	if node.PaneID != "" {
		item, ok := group[node.PaneID]
		return item.Cells, ok && item.Cells > 0
	}
	if len(node.Children) == 0 {
		return 0, false
	}
	extent := 0
	for _, child := range node.Children {
		childExtent, ok := paneResizeGroupExtent(child, axis, group)
		if !ok {
			return 0, false
		}
		if node.Direction == axis {
			extent += childExtent
		} else if childExtent > extent {
			extent = childExtent
		}
	}
	return extent, extent > 0
}

func samePaneOrder(panes []string, group []PaneResizeGroupItem) bool {
	if len(panes) != len(group) {
		return false
	}
	for i, paneID := range panes {
		if paneID != group[i].PaneID {
			return false
		}
	}
	return true
}

func clonePaneResizeGroupItems(items []PaneResizeGroupItem) []PaneResizeGroupItem {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]PaneResizeGroupItem, len(items))
	copy(cloned, items)
	return cloned
}

func validPaneResizeGroup(items []PaneResizeGroupItem) bool {
	for _, item := range items {
		if item.PaneID == "" || item.Cells <= 0 {
			return false
		}
	}
	return true
}

func setSplitNodeSize(node SplitNode, command PaneCommand) (SplitNode, bool) {
	if node.PaneID != "" || len(node.Children) < 2 {
		return node, node.PaneID == command.Target.PaneID
	}
	firstContains := splitContainsPane(node.Children[0], command.Target.PaneID)
	secondContains := splitContainsPane(node.Children[1], command.Target.PaneID)
	if firstContains || secondContains {
		node.BiasCells = 0
		node.FixedPaneID = ""
		node.FixedCols = 0
		node.FixedRows = 0
		node.Ratio = 0
		switch command.SizeMode {
		case PaneSizeRatio:
			if firstContains {
				node.Ratio = command.Ratio
			} else {
				node.Ratio = 1 - command.Ratio
			}
		case PaneSizeCells:
			node.FixedPaneID = command.Target.PaneID
			if node.Direction == SplitDirectionVertical {
				node.FixedCols = command.Cols
				if node.FixedCols <= 0 {
					node.FixedCols = command.Rows
				}
			} else {
				node.FixedRows = command.Rows
				if node.FixedRows <= 0 {
					node.FixedRows = command.Cols
				}
			}
		}
		return node, true
	}
	children := cloneSplitNodes(node.Children)
	changed := false
	for i, child := range children {
		children[i], changed = setSplitNodeSize(child, command)
		if changed {
			break
		}
	}
	node.Children = children
	return node, changed
}

func balanceSplitNode(node SplitNode) SplitNode {
	node.Ratio = 0
	node.BiasCells = 0
	node.FixedPaneID = ""
	node.FixedCols = 0
	node.FixedRows = 0
	for i, child := range node.Children {
		node.Children[i] = balanceSplitNode(child)
	}
	return node
}

func splitContainsPane(node SplitNode, paneID string) bool {
	if node.PaneID == paneID {
		return true
	}
	for _, child := range node.Children {
		if splitContainsPane(child, paneID) {
			return true
		}
	}
	return false
}

func splitDirectionMatchesResize(splitDirection SplitDirection, resizeDirection PaneResizeDirection) bool {
	switch splitDirection {
	case SplitDirectionVertical:
		return resizeDirection == PaneResizeLeft || resizeDirection == PaneResizeRight
	case SplitDirectionHorizontal:
		return resizeDirection == PaneResizeUp || resizeDirection == PaneResizeDown
	default:
		return false
	}
}

func splitDirectionForResize(resizeDirection PaneResizeDirection) (SplitDirection, bool) {
	switch resizeDirection {
	case PaneResizeLeft, PaneResizeRight:
		return SplitDirectionVertical, true
	case PaneResizeUp, PaneResizeDown:
		return SplitDirectionHorizontal, true
	default:
		return "", false
	}
}

func resizeBiasDelta(splitDirection SplitDirection, resizeDirection PaneResizeDirection, firstContains bool, delta int) int {
	switch splitDirection {
	case SplitDirectionVertical:
		if firstContains && resizeDirection == PaneResizeRight {
			return delta
		}
		if firstContains && resizeDirection == PaneResizeLeft {
			return -delta
		}
		if !firstContains && resizeDirection == PaneResizeLeft {
			return -delta
		}
		return delta
	case SplitDirectionHorizontal:
		if firstContains && resizeDirection == PaneResizeDown {
			return delta
		}
		if firstContains && resizeDirection == PaneResizeUp {
			return -delta
		}
		if !firstContains && resizeDirection == PaneResizeUp {
			return -delta
		}
		return delta
	default:
		return 0
	}
}

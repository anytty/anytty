package state

import (
	"fmt"
	"strings"
	"time"
)

func WorkbenchTreeItems(root Root) []WorkbenchTreeItem {
	shell := root.Shell.ReadonlyDefaults()
	query := strings.ToLower(strings.TrimSpace(shell.Overlay.Query))
	workspaces := shell.Workspaces
	if len(workspaces) == 0 {
		workspaces = []WorkspaceState{shell.Workspace}
	}
	items := make([]WorkbenchTreeItem, 0, len(workspaces)*4)
	appendItem := func(item WorkbenchTreeItem) {
		if matchesWorkbenchTreeQuery(item, query) {
			items = append(items, item)
		}
	}

	for _, workspace := range workspaces {
		workspace = workspace.ensureDefaults()
		workspaceActive := workspace.ID == shell.Workspace.ID
		workspaceItem := WorkbenchTreeItem{
			Kind:          WorkbenchTreeKindWorkspace,
			WorkspaceID:   workspace.ID,
			WorkspaceName: workspace.Name,
			Depth:         0,
			Expandable:    len(workspace.Tabs) > 0,
			Active:        workspaceActive,
			Summary:       workbenchWorkspaceSummary(workspace),
		}
		workspaceItem.Collapsed = workbenchTreeItemCollapsed(shell.Overlay, workspaceItem)
		appendItem(workspaceItem)
		if query == "" && workspaceItem.Collapsed {
			continue
		}
		for _, tab := range workspace.Tabs {
			tabActive := workspaceActive && tab.ID == workspace.ActiveTabID
			tabItem := WorkbenchTreeItem{
				Kind:          WorkbenchTreeKindTab,
				WorkspaceID:   workspace.ID,
				WorkspaceName: workspace.Name,
				TabID:         tab.ID,
				TabTitle:      tab.Title,
				PaneID:        tab.ActivePaneID,
				Depth:         1,
				Expandable:    len(tab.Panes)+len(tab.Floatings) > 0,
				Active:        tabActive,
				Summary:       workbenchTabSummary(tab),
			}
			tabItem.Collapsed = workbenchTreeItemCollapsed(shell.Overlay, tabItem)
			appendItem(tabItem)
			if query == "" && tabItem.Collapsed {
				continue
			}
			for _, pane := range tab.Panes {
				terminalRef := workbenchPaneTerminalRef(root, pane)
				terminalID := terminalRef.TerminalID
				displayTitle := ""
				if terminalID != "" {
					displayTitle = workbenchTerminalTitle(root.TerminalPool, terminalRef)
				}
				item := WorkbenchTreeItem{
					Kind:          WorkbenchTreeKindPane,
					WorkspaceID:   workspace.ID,
					WorkspaceName: workspace.Name,
					TabID:         tab.ID,
					TabTitle:      tab.Title,
					PaneID:        pane.ID,
					PaneTitle:     paneTitle(pane),
					DisplayTitle:  displayTitle,
					PaneKind:      pane.Kind,
					EndpointID:    terminalRef.EndpointID,
					TerminalID:    terminalID,
					LastOutputAt:  workbenchTerminalLastOutputAt(root.TerminalPool, terminalRef),
					Depth:         2,
					Active:        tabActive && pane.ID == shell.ActivePaneID,
					Summary:       workbenchPaneSummary(pane, terminalID),
				}
				item = workbenchTreeItemWithEndpoint(root, item)
				appendItem(item)
			}
			for _, floating := range tab.Floatings {
				pane := floating.Pane
				terminalRef := workbenchFloatingTerminalRef(root, floating)
				terminalID := terminalRef.TerminalID
				displayTitle := ""
				if terminalID != "" {
					displayTitle = workbenchTerminalTitle(root.TerminalPool, terminalRef)
				}
				item := WorkbenchTreeItem{
					Kind:          WorkbenchTreeKindFloating,
					WorkspaceID:   workspace.ID,
					WorkspaceName: workspace.Name,
					TabID:         tab.ID,
					TabTitle:      tab.Title,
					FloatingID:    floating.ID,
					FloatingTitle: floating.Title,
					PaneID:        pane.ID,
					PaneTitle:     paneTitle(pane),
					DisplayTitle:  displayTitle,
					PaneKind:      pane.Kind,
					EndpointID:    terminalRef.EndpointID,
					TerminalID:    terminalID,
					LastOutputAt:  workbenchTerminalLastOutputAt(root.TerminalPool, terminalRef),
					Depth:         2,
					Active:        tabActive && floating.Active,
					Summary:       workbenchFloatingSummary(floating, terminalID),
				}
				item = workbenchTreeItemWithEndpoint(root, item)
				appendItem(item)
			}
		}
	}
	if len(items) > 0 {
		selected := shell.Overlay.SelectedIndex
		if selected < 0 {
			selected = 0
		}
		if selected >= len(items) {
			selected = len(items) - 1
		}
		items[selected].Selected = true
	}
	return items
}

func workbenchTreeItemCollapsed(overlay OverlayState, item WorkbenchTreeItem) bool {
	key, ok := workbenchTreeCollapseKey(item)
	return ok && overlay.WorkbenchCollapsed[key]
}

func workbenchFloatingTerminalRef(root Root, floating FloatingPaneState) TerminalRef {
	if binding, ok := root.TerminalViews.FloatingBinding(floating.ID); ok && binding.TerminalID != "" {
		return binding.TerminalRef()
	}
	return workbenchPaneTerminalRef(root, floating.Pane)
}

func workbenchPaneTerminalRef(root Root, pane PaneState) TerminalRef {
	if binding, ok := root.TerminalViews.PaneBinding(pane.ID); ok && binding.TerminalID != "" {
		return binding.TerminalRef()
	}
	if pane.TerminalID != "" {
		return LocalTerminalRef(pane.TerminalID)
	}
	return TerminalRef{}
}

func pickerTerminalID(root Root, pane PaneState) string {
	if pane.TerminalID != "" {
		return pane.TerminalID
	}
	if pane.Active && root.Session.TerminalID != "" {
		return root.Session.TerminalID
	}
	if pane.Active && root.Surface.TerminalID != "" {
		return root.Surface.TerminalID
	}
	if pane.Active && root.History.TerminalID != "" {
		return root.History.TerminalID
	}
	return ""
}

func workbenchTerminalTitle(pool TerminalPoolStore, ref TerminalRef) string {
	ref = ref.Normalize()
	for _, item := range pool.Items {
		if !item.TerminalRef().Equal(ref) {
			continue
		}
		return terminalPoolTitle(item)
	}
	return ref.TerminalID
}

func workbenchTerminalLastOutputAt(pool TerminalPoolStore, ref TerminalRef) time.Time {
	ref = ref.Normalize()
	for _, item := range pool.Items {
		if !item.TerminalRef().Equal(ref) {
			continue
		}
		return item.LastOutputAt
	}
	return time.Time{}
}

func workbenchTreeItemWithEndpoint(root Root, item WorkbenchTreeItem) WorkbenchTreeItem {
	if item.TerminalID == "" {
		return item
	}
	item.EndpointID = NormalizeEndpointID(item.EndpointID)
	if endpoint, ok := root.Endpoints.DisplayEndpoint(item.EndpointID); ok {
		item.EndpointLabel = endpoint.DisplayLabel()
		item.EndpointStatus = endpoint.DisplayStatus()
		item.EndpointLastError = endpoint.LastError
		item.EndpointErrorKind = endpoint.LastErrorKind
	}
	return item
}

func matchesWorkbenchTreeQuery(item WorkbenchTreeItem, query string) bool {
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(item.Kind), query) ||
		strings.Contains(strings.ToLower(item.WorkspaceName), query) ||
		strings.Contains(strings.ToLower(item.WorkspaceID), query) ||
		strings.Contains(strings.ToLower(item.TabTitle), query) ||
		strings.Contains(strings.ToLower(item.TabID), query) ||
		strings.Contains(strings.ToLower(item.FloatingTitle), query) ||
		strings.Contains(strings.ToLower(item.FloatingID), query) ||
		strings.Contains(strings.ToLower(item.PaneTitle), query) ||
		strings.Contains(strings.ToLower(item.DisplayTitle), query) ||
		strings.Contains(strings.ToLower(item.PaneID), query) ||
		strings.Contains(strings.ToLower(string(item.PaneKind)), query) ||
		strings.Contains(strings.ToLower(item.TerminalID), query) ||
		strings.Contains(strings.ToLower(item.Summary), query)
}

func workbenchWorkspaceSummary(workspace WorkspaceState) string {
	return fmt.Sprintf("tabs:%d panes:%d", len(workspace.Tabs), workspacePaneCount(workspace))
}

func workbenchTabSummary(tab TabState) string {
	if len(tab.Floatings) == 0 {
		return fmt.Sprintf("panes:%d active:%s", len(tab.Panes), tab.ActivePaneID)
	}
	return fmt.Sprintf("panes:%d floating:%d active:%s", len(tab.Panes), len(tab.Floatings), tab.ActivePaneID)
}

func workbenchPaneSummary(pane PaneState, terminalID string) string {
	summary := string(pane.Kind)
	if terminalID != "" {
		summary += " term:" + terminalID
	}
	return summary
}

func workbenchFloatingSummary(floating FloatingPaneState, terminalID string) string {
	tags := []string{"floating"}
	if floating.Pane.Kind != "" {
		tags = append(tags, string(floating.Pane.Kind))
	}
	if floating.Collapsed {
		tags = append(tags, "collapsed")
	} else {
		tags = append(tags, "open")
	}
	if floating.FitMode == FloatingFitAuto {
		tags = append(tags, "auto-fit")
	} else {
		tags = append(tags, "manual")
	}
	if terminalID != "" {
		tags = append(tags, "term:"+terminalID)
	}
	return strings.Join(tags, " ")
}

func workspacePaneCount(workspace WorkspaceState) int {
	count := 0
	for _, tab := range workspace.Tabs {
		count += len(tab.Panes)
	}
	return count
}

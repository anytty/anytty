package state

import (
	"errors"
	"github.com/anytty/anytty/proto/apipb"
	"sort"
	"strings"
)

// PluginOwner is a complete, immutable container address. A floating belongs to a tab.
type PluginOwner struct{ Kind, WorkspaceID, TabID, PaneID, FloatingID string }
type PluginNodeStyle struct {
	ForegroundRole string
	BackgroundRole string
	Bold           bool
	Dim            bool
}
type PluginNodeLayout struct {
	PaddingTop    int
	PaddingRight  int
	PaddingBottom int
	PaddingLeft   int
	GapAfter      int
}
type PluginNode struct {
	ID, Kind, Text, Detail, Action, Value, Status, Description string
	ItemID, Placeholder                                        string
	Progress                                                   float64
	Depth                                                      int
	EndpointID                                                 EndpointID
	DaemonID, TerminalID                                       string
	Disabled                                                   bool
	Style, SelectedStyle                                       PluginNodeStyle
	Layout                                                     PluginNodeLayout
	Children                                                   []PluginNode
}
type PluginMount struct {
	Collapsed                             map[string]bool
	InputValues                           map[string]string
	EditingID, EditOriginal               string
	EditCursor                            int
	Source                                *apipb.PluginAddress
	Actions                               []*apipb.PluginUiAction
	ID, PluginID, DaemonID                string
	EndpointID                            EndpointID
	Owner                                 PluginOwner
	Slot, Title                           string
	Revision                              uint64
	Nodes                                 []PluginNode
	SelectedID, Search                    string
	Searching, Interactive, Hidden, Stale bool
	Width                                 int
}
type PluginInteractionContext struct {
	Pending    bool
	Context    *apipb.PluginTargetContext
	PluginID   string
	EndpointID EndpointID
	Expires    int64
}
type PluginPeer struct {
	EndpointID    EndpointID
	Address, Host *apipb.PluginAddress
}
type PluginStore struct {
	Peers             []PluginPeer
	OwnersKey         string
	Contexts          map[string]PluginInteractionContext
	LastClickID       string
	LastClickMillis   int64
	Mounts            map[string]PluginMount
	FocusedMountID    string
	LastContentPaneID string
	Error             string
}

func (s PluginStore) clone() PluginStore {
	next := make(map[string]PluginMount, len(s.Mounts))
	for k, v := range s.Mounts {
		next[k] = v
	}
	s.Mounts = next
	return s
}
func clonePluginNodes(nodes []PluginNode) []PluginNode {
	out := append([]PluginNode(nil), nodes...)
	for i := range out {
		out[i].Children = clonePluginNodes(out[i].Children)
	}
	return out
}
func ValidPluginSlot(owner, slot string) bool {
	switch owner {
	case "workspace":
		return slot == "sidebar" || slot == "statusbar" || slot == "menu" || slot == "overlay"
	case "tab":
		return slot == "header" || slot == "menu" || slot == "overlay" || slot == "content"
	case "panel", "floating":
		return slot == "header" || slot == "menu" || slot == "content"
	}
	return false
}
func (o PluginOwner) Exists(shell ShellStore) bool { _, exists := o.visibility(shell); return exists }
func (o PluginOwner) Visible(shell ShellStore) bool {
	visible, _ := o.visibility(shell)
	return visible
}
func (o PluginOwner) visibility(shell ShellStore) (bool, bool) {
	shell = shell.ReadonlyDefaults()
	workspaces := append([]WorkspaceState(nil), shell.Workspaces...)
	found := false
	for i := range workspaces {
		if workspaces[i].ID == shell.Workspace.ID {
			workspaces[i] = shell.Workspace
			found = true
		}
	}
	if !found {
		workspaces = append(workspaces, shell.Workspace)
	}
	for _, w := range workspaces {
		if w.ID != o.WorkspaceID {
			continue
		}
		active := w.ID == shell.Workspace.ID
		if o.Kind == "workspace" {
			return active, true
		}
		for _, t := range w.Tabs {
			if t.ID != o.TabID {
				continue
			}
			active = active && t.ID == w.ActiveTabID
			switch o.Kind {
			case "tab":
				return active, true
			case "panel":
				for _, p := range t.Panes {
					if p.ID == o.PaneID {
						return active, true
					}
				}
			case "floating":
				for _, f := range t.Floatings {
					if f.ID == o.FloatingID {
						return active && !f.Collapsed, true
					}
				}
			}
			return false, false
		}
		return false, false
	}
	return false, false
}
func (s PluginStore) Apply(shell ShellStore, m PluginMount, base uint64) (PluginStore, error) {
	if m.ID == "" || m.PluginID == "" || m.DaemonID == "" || !ValidPluginSlot(m.Owner.Kind, m.Slot) || !m.Owner.Exists(shell) {
		return s, errors.New("invalid plugin mount owner or slot")
	}
	old, exists := s.Mounts[m.ID]
	if exists && (old.PluginID != m.PluginID || old.DaemonID != m.DaemonID || old.Owner != m.Owner) {
		return s, errors.New("plugin mount identity conflict")
	}
	if (exists && base != old.Revision) || (!exists && base != 0) || m.Revision <= base {
		return s, errors.New("plugin mount revision conflict")
	}
	m.Nodes = clonePluginNodes(m.Nodes)
	if exists {
		m.SelectedID = old.SelectedID
		m.Search = old.Search
		m.Searching = old.Searching
		m.Collapsed = old.Collapsed
		m.InputValues = old.InputValues
		m.EditingID = old.EditingID
		m.EditOriginal = old.EditOriginal
		m.EditCursor = old.EditCursor
		m.Hidden = old.Hidden
	}
	ids := make(map[string]bool)
	var collect func([]PluginNode)
	collect = func(nodes []PluginNode) {
		for _, n := range nodes {
			ids[n.ID] = true
			collect(n.Children)
		}
	}
	collect(m.Nodes)
	values := make(map[string]string)
	for id, value := range m.InputValues {
		if ids[id] {
			values[id] = value
		}
	}
	m.InputValues = values
	collapsed := make(map[string]bool)
	for id, value := range m.Collapsed {
		if ids[id] {
			collapsed[id] = value
		}
	}
	m.Collapsed = collapsed
	if !ids[m.EditingID] {
		m.EditingID = ""
	}
	rows := m.Rows()
	valid := false
	for _, n := range rows {
		if n.ID == m.SelectedID {
			valid = true
		}
	}
	if !valid {
		m.SelectedID = ""
		if len(rows) > 0 {
			m.SelectedID = rows[0].ID
		}
	}
	s = s.clone()
	s.Mounts[m.ID] = m
	return s, nil
}
func (s PluginStore) Set(m PluginMount) PluginStore { s = s.clone(); s.Mounts[m.ID] = m; return s }
func (s PluginStore) Remove(id string) PluginStore {
	s = s.clone()
	delete(s.Mounts, id)
	if s.FocusedMountID == id {
		s.FocusedMountID = ""
	}
	return s
}
func (s PluginStore) Prune(shell ShellStore) (PluginStore, []PluginMount) {
	var removed []PluginMount
	for id, m := range s.Mounts {
		if !m.Owner.Exists(shell) {
			removed = append(removed, m)
			s = s.Remove(id)
		}
	}
	if m, ok := s.Mounts[s.FocusedMountID]; ok && (!m.Owner.Visible(shell) || m.Hidden) {
		s.FocusedMountID = ""
	}
	return s, removed
}
func (s PluginStore) Visible(shell ShellStore) []PluginMount {
	var out []PluginMount
	for _, m := range s.Mounts {
		if !m.Hidden && m.Owner.Visible(shell) {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func (m PluginMount) Rows() []PluginNode {
	var rows []PluginNode
	var walk func([]PluginNode, int)
	walk = func(nodes []PluginNode, depth int) {
		for _, n := range nodes {
			if value, ok := m.InputValues[n.ID]; ok {
				n.Value = value
			}
			if n.Kind == "row" && len(n.Children) > 0 {
				var cells []string
				for _, cell := range n.Children {
					cells = append(cells, cell.Text)
				}
				if n.Text != "" {
					cells = append([]string{n.Text}, cells...)
				}
				n.Text = strings.Join(cells, " │ ")
			}
			if n.Kind == "gap" {
				continue
			}
			if n.Kind == "card" {
				if m.Search == "" || strings.Contains(strings.ToLower(n.Text+" "+n.Description+" "+n.Status), strings.ToLower(m.Search)) {
					n.Depth = depth
					rows = append(rows, n)
				}
				continue
			}
			if n.Kind == "tree" {
				if n.Text != "" {
					n.Depth = depth
					rows = append(rows, n)
				}
				if !m.Collapsed[n.ID] || m.Search != "" {
					walk(n.Children, depth+1)
				}
				continue
			}
			if n.Kind == "list" || n.Kind == "layout" || n.Kind == "table" || n.Kind == "column" || n.Kind == "form" {
				walk(n.Children, depth)
				continue
			}
			if m.Search == "" || strings.Contains(strings.ToLower(n.Text+" "+n.Detail+" "+n.Description+" "+n.Status), strings.ToLower(m.Search)) {
				n.Depth = depth
				rows = append(rows, n)
			}
		}
	}
	walk(m.Nodes, 0)
	return rows
}
func (m PluginMount) Selected() (PluginNode, bool) {
	for _, r := range m.Rows() {
		if r.ID == m.SelectedID {
			return r, true
		}
	}
	return PluginNode{}, false
}
func (m PluginMount) Move(delta int) PluginMount {
	rows := m.Rows()
	if len(rows) == 0 {
		m.SelectedID = ""
		return m
	}
	index := 0
	for i, n := range rows {
		if n.ID == m.SelectedID {
			index = i
			break
		}
	}
	index += delta
	if index < 0 {
		index = 0
	}
	if index >= len(rows) {
		index = len(rows) - 1
	}
	m.SelectedID = rows[index].ID
	return m
}

func (m PluginMount) WithValue(id, value string) PluginMount {
	values := make(map[string]string, len(m.InputValues)+1)
	for k, v := range m.InputValues {
		values[k] = v
	}
	values[id] = value
	m.InputValues = values
	return m
}
func (m PluginMount) FormValues() map[string]string {
	values := make(map[string]string)
	m.Search = ""
	m.Collapsed = nil
	for _, n := range m.Rows() {
		switch n.Kind {
		case "input", "select", "checkbox":
			values[n.ID] = n.Value
		}
	}
	return values
}
func (m PluginMount) FormAction() string {
	var walk func([]PluginNode) string
	walk = func(nodes []PluginNode) string {
		for _, n := range nodes {
			if n.Kind == "form" && n.Action != "" {
				return n.Action
			}
			if a := walk(n.Children); a != "" {
				return a
			}
		}
		return ""
	}
	return walk(m.Nodes)
}

func (m PluginMount) MoveFocus(delta int) PluginMount {
	rows := m.Rows()
	if len(rows) == 0 {
		return m
	}
	selected := 0
	for i, n := range rows {
		if n.ID == m.SelectedID {
			selected = i
		}
	}
	for attempts := 0; attempts < len(rows); attempts++ {
		selected = (selected + delta + len(rows)) % len(rows)
		n := rows[selected]
		if !n.Disabled && (n.Action != "" || n.Kind == "input" || n.Kind == "select" || n.Kind == "checkbox") {
			m.SelectedID = n.ID
			break
		}
	}
	return m
}

func (m PluginMount) SetCollapsed(id string, collapsed bool) PluginMount {
	values := make(map[string]bool, len(m.Collapsed)+1)
	for k, v := range m.Collapsed {
		values[k] = v
	}
	values[id] = collapsed
	m.Collapsed = values
	return m
}

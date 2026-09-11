package agents

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

type Entry struct {
	EndpointID string
	Report     *apipb.PluginAgentReport
	Stale      bool
}

func ItemID(report *apipb.PluginAgentReport) string {
	return base64.RawURLEncoding.EncodeToString([]byte(report.GetTerminal().GetDaemonId() + "\x00" + report.GetProvider() + "\x00" + report.GetSessionId()))
}

func BuildMount(entries []Entry, owner *apipb.PluginMountOwner, mountID string, revision uint64, attentionOnly bool) *apipb.PluginUiMountUpdate {
	if revision == 0 {
		revision = 1
	}
	entries = append([]Entry(nil), entries...)
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		ra, rb := attentionRank(a), attentionRank(b)
		if ra != rb {
			return ra < rb
		}
		if a.EndpointID != b.EndpointID {
			return a.EndpointID < b.EndpointID
		}
		return ItemID(a.Report) < ItemID(b.Report)
	})
	root := &apipb.PluginUiNode{Id: "agents", Kind: "list"}
	attention := 0
	for _, entry := range entries {
		r := entry.Report
		if r == nil || r.Terminal == nil {
			continue
		}
		if attentionRank(entry) == 0 {
			attention++
		} else if attentionOnly {
			continue
		}
		status := r.State
		if entry.Stale {
			status = "stale"
		}
		title := r.Title
		if title == "" {
			title = r.Cwd
		}
		if title == "" {
			title = r.SessionId
		}
		title = strings.Map(func(r rune) rune {
			if r < 32 || r == 127 {
				return ' '
			}
			return r
		}, title)
		icon := "○"
		switch status {
		case "working":
			icon = "●"
		case "blocked", "error":
			icon = "!"
		case "stale":
			icon = "…"
		case "exited":
			icon = "×"
		}
		foregroundRole := "primary"
		switch status {
		case "working":
			foregroundRole = "success"
		case "blocked":
			foregroundRole = "warning"
		case "error":
			foregroundRole = "danger"
		case "idle", "stale", "exited":
			foregroundRole = "muted"
		}
		root.Children = append(root.Children, &apipb.PluginUiNode{
			Id: ItemID(r), Kind: "card", Text: fmt.Sprintf("%s  %s", icon, title),
			Description: fmt.Sprintf("%s · %s · %s", entry.EndpointID, r.Provider, time.UnixMilli(r.ObservedUnixMillis).Format("15:04:05")),
			Status:      status, ItemId: ItemID(r), ActionId: "agents.open", Disabled: entry.Stale || r.State == "exited",
			Terminal:      proto.Clone(r.Terminal).(*apipb.PluginTerminalRef),
			Style:         &apipb.PluginUiStyle{ForegroundRole: foregroundRole, BackgroundRole: "surface"},
			SelectedStyle: &apipb.PluginUiStyle{ForegroundRole: "primary", BackgroundRole: "selected", Bold: true},
			Layout:        &apipb.PluginUiLayout{PaddingTop: 1, PaddingBottom: 1, PaddingLeft: 1, GapAfter: 1},
		})
	}
	if len(root.Children) == 0 {
		root.Children = append(root.Children, &apipb.PluginUiNode{Id: "empty", Kind: "text", Text: "No Agents yet. Install Codex / OpenCode hooks.", Disabled: true})
	}
	filterLabel := "Show needs attention"
	if attentionOnly {
		filterLabel = "Show all Agents"
	}
	title := fmt.Sprintf("AGENTS  ·  %d need attention", attention)
	if owner.GetPanel() != nil {
		root.Kind = "column"
		for _, child := range root.Children {
			child.Kind = "badge"
			child.Text = child.Status
			child.ActionId = ""
		}
		return &apipb.PluginUiMountUpdate{MountId: mountID, Owner: owner, Slot: "header", Revision: revision, ExpectedRevision: revision - 1, Title: "Agent", Root: root}
	}
	return &apipb.PluginUiMountUpdate{MountId: mountID, Owner: owner, Slot: "sidebar", Revision: revision, ExpectedRevision: revision - 1, Title: title, Root: root, PreferredWidth: 42, MinWidth: 24, Actions: []*apipb.PluginUiAction{
		{Id: "agents.open", Label: "Open terminal", DefaultKey: "enter", Enabled: true, Scope: "mount"},
		{Id: "agents.filter", Label: filterLabel, DefaultKey: "f", Enabled: true, Scope: "mount"},
	}}
}

func attentionRank(e Entry) int {
	if e.Stale || e.Report.GetState() == "blocked" || e.Report.GetState() == "error" {
		return 0
	}
	return 1
}

func MountID(owner *apipb.PluginMountOwner) string {
	if w := owner.GetWorkspace(); w != nil {
		return "agents.workspace." + base64.RawURLEncoding.EncodeToString([]byte(w.WorkspaceId))
	}
	if p := owner.GetPanel(); p != nil {
		return "agents.panel." + base64.RawURLEncoding.EncodeToString([]byte(strings.Join([]string{p.WorkspaceId, p.TabId, p.PaneId}, "\x00")))
	}
	return ""
}

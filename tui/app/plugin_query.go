package app

import (
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

// A read snapshot is a public projection, never an export of reducer internals.
// ActiveContext describes focus only: it contains no minted context_id and does
// not authorize a later write. Writes still need a real interaction context.
func reducePluginQuery(root state.Root, delivery port.PluginDelivery, deps PluginDeps) (state.Root, []Effect) {
	query := delivery.Message.GetUiQuery()
	if query.GetSnapshot() == nil {
		return root, []Effect{pluginReply(deps, delivery, "UNSUPPORTED", "unsupported UI query")}
	}
	if delivery.Message.GetRequestId() == "" {
		return root, nil
	}
	snapshot := pluginUISnapshot(root, deps)
	reply := &apipb.PluginReply{RequestId: delivery.Message.RequestId, UiSnapshot: snapshot}
	return root, []Effect{pluginSend(deps, delivery.EndpointID, &apipb.PluginMessage{RequestId: pluginID(), TraceId: delivery.Message.TraceId, Destination: delivery.Message.Source, Body: &apipb.PluginMessage_Reply{Reply: reply}})}
}

type pluginDaemonIdentity interface {
	DaemonIdentity(state.EndpointID) (string, bool)
}

func pluginDaemonID(root state.Root, deps PluginDeps, endpoint state.EndpointID) (string, bool) {
	endpoint = state.NormalizeEndpointID(endpoint)
	if identities, ok := deps.Service.(pluginDaemonIdentity); ok {
		if id, found := identities.DaemonIdentity(endpoint); found && id != "" {
			return id, true
		}
	}
	for _, peer := range root.Plugins.Peers {
		if state.NormalizeEndpointID(peer.EndpointID) == endpoint && peer.Host.GetDaemonId() != "" {
			return peer.Host.DaemonId, true
		}
	}
	return "", false
}

func pluginUISnapshot(root state.Root, deps PluginDeps) *apipb.PluginUiSnapshot {
	init := pluginInit(root, deps.TUIInstanceID, nil)
	snapshot := &apipb.PluginUiSnapshot{TuiInstanceId: deps.TUIInstanceID, Owners: init.Owners, ActiveContext: init.Context, Revision: root.Generation}
	shell := root.Shell.ReadonlyDefaults()
	activeFloating := ""
	for _, tab := range shell.Workspace.Tabs {
		if tab.ID == shell.Workspace.ActiveTabID {
			activeFloating = tab.ActiveFloatingID
			break
		}
	}
	snapshot.ActiveContext.FloatingId = activeFloating
	snapshot.ActiveContext.MountId = root.Plugins.FocusedMountID
	for _, owner := range snapshot.Owners {
		var binding state.TerminalViewBinding
		var bound bool
		var focused bool
		switch o := owner.Owner.(type) {
		case *apipb.PluginMountOwner_Panel:
			binding, bound = root.TerminalViews.PaneBinding(o.Panel.PaneId)
			focused = o.Panel.WorkspaceId == shell.Workspace.ID && o.Panel.TabId == shell.Workspace.ActiveTabID && o.Panel.PaneId == shell.ActivePaneID && activeFloating == "" && root.Plugins.FocusedMountID == ""
		case *apipb.PluginMountOwner_Floating:
			binding, bound = root.TerminalViews.FloatingBinding(o.Floating.FloatingId)
			focused = o.Floating.WorkspaceId == shell.Workspace.ID && o.Floating.TabId == shell.Workspace.ActiveTabID && o.Floating.FloatingId == activeFloating && root.Plugins.FocusedMountID == ""
		default:
			continue
		}
		panel := &apipb.PluginPanelSnapshot{Owner: owner, Focused: focused}
		if bound {
			panel.ViewId = binding.ViewID
			if daemon, verified := pluginDaemonID(root, deps, binding.EndpointID); verified {
				panel.Terminal = &apipb.PluginTerminalRef{DaemonId: daemon, TerminalId: binding.TerminalID}
			}
		}
		if focused {
			snapshot.ActiveContext.BindingRevision = pluginBindingRevision(root, pluginOwner(owner))
		}
		snapshot.Panels = append(snapshot.Panels, panel)
	}
	return snapshot
}

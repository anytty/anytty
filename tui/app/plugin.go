package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"time"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
	"google.golang.org/protobuf/proto"
)

type PluginDeps struct {
	Service       port.PluginService
	TUIInstanceID string
	Live          LiveDeps
}
type PluginStartMsg struct{}

func (PluginStartMsg) isMsg() {}

type PluginDeliveryMsg struct{ Delivery port.PluginDelivery }

func (PluginDeliveryMsg) isMsg() {}

type PluginInputMsg struct {
	MountID, NodeID string
	Event           input.InputEvent
}

func (PluginInputMsg) isMsg() {}

type PluginSendResultMsg struct{ Err error }

func (PluginSendResultMsg) isMsg() {}

type PluginBindResultMsg struct {
	Delivery port.PluginDelivery
	Attach   LiveAttachResultMsg
	Target   state.PluginOwner
	Revision uint64
}

func (PluginBindResultMsg) isMsg() {}

func NewPluginReducer(deps PluginDeps) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {

		switch m := msg.(type) {
		case ShellActivateTerminalInputMsg, ShellPaneCommandMsg, ShellFloatingCommandMsg, ShellWorkbenchCommandMsg:
			root.Plugins.FocusedMountID = ""
			root.Plugins.LastContentPaneID = root.Shell.ReadonlyDefaults().ActivePaneID
			return root, nil
		case PluginStartMsg:
			if deps.Service == nil {
				return root, nil
			}
			return root, []Effect{StreamEffect{Token: "plugins.watch", Run: func(ctx context.Context, post func(Msg)) {
				stream, err := deps.Service.Watch(ctx)
				if err != nil {
					post(PluginSendResultMsg{Err: err})
					return
				}
				for {
					select {
					case <-ctx.Done():
						return
					case d, ok := <-stream:
						if !ok {
							return
						}
						post(PluginDeliveryMsg{Delivery: d})
					}
				}
			}}}
		case PluginSendResultMsg:
			if m.Err != nil {
				root.Plugins.Error = m.Err.Error()
				return root.Advance(), nil
			}
			return root, nil
		case PluginDeliveryMsg:
			return reducePluginDelivery(root, m.Delivery, deps)
		case PluginBindResultMsg:
			return reducePluginBindResult(root, m, deps)
		case PluginInputMsg:
			return reducePluginInput(root, m, deps)
		case InputMsg:
			if m.Event.Kind == input.EventKindMouse && m.Event.Mouse == input.MouseLeft {
				root.Plugins.FocusedMountID = ""
				return root, nil
			}
			if next, effects, handled := pluginShortcut(root, m.Event, deps); handled {
				return next, effects
			}
			if root.Shell.Overlay.Open || root.Shell.InteractionMode != state.InteractionModeNormal || root.Plugins.FocusedMountID == "" || m.Event.Kind != input.EventKindKey && m.Event.Kind != input.EventKindPaste {
				return root, nil
			}
			return reducePluginInput(root, PluginInputMsg{MountID: root.Plugins.FocusedMountID, Event: m.Event}, deps)
		}
		return root, nil
	}
}
func pluginID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func pluginSend(deps PluginDeps, endpoint state.EndpointID, msg *apipb.PluginMessage) Effect {
	return FuncEffect{Async: true, SerialKey: "plugin:" + string(endpoint), Run: func(ctx context.Context) Msg {
		if deps.Service == nil {
			return PluginSendResultMsg{Err: errors.New("plugin daemon unavailable")}
		}
		return PluginSendResultMsg{Err: deps.Service.Send(ctx, endpoint, msg)}
	}}
}
func pluginReply(deps PluginDeps, d port.PluginDelivery, code, detail string) Effect {
	reply := &apipb.PluginReply{RequestId: d.Message.GetRequestId()}
	if code != "" {
		reply.Error = &apipb.PluginError{Code: code, Message: detail}
	}
	return pluginSend(deps, d.EndpointID, &apipb.PluginMessage{RequestId: pluginID(), TraceId: d.Message.GetTraceId(), Destination: d.Message.GetSource(), Body: &apipb.PluginMessage_Reply{Reply: reply}})
}
func pluginOwner(p *apipb.PluginMountOwner) state.PluginOwner {
	switch o := p.GetOwner().(type) {
	case *apipb.PluginMountOwner_Workspace:
		return state.PluginOwner{Kind: "workspace", WorkspaceID: o.Workspace.GetWorkspaceId()}
	case *apipb.PluginMountOwner_Tab:
		return state.PluginOwner{Kind: "tab", WorkspaceID: o.Tab.GetWorkspaceId(), TabID: o.Tab.GetTabId()}
	case *apipb.PluginMountOwner_Panel:
		return state.PluginOwner{Kind: "panel", WorkspaceID: o.Panel.GetWorkspaceId(), TabID: o.Panel.GetTabId(), PaneID: o.Panel.GetPaneId()}
	case *apipb.PluginMountOwner_Floating:
		return state.PluginOwner{Kind: "floating", WorkspaceID: o.Floating.GetWorkspaceId(), TabID: o.Floating.GetTabId(), FloatingID: o.Floating.GetFloatingId()}
	}
	return state.PluginOwner{}
}

func pluginMountOwner(root state.Root, update *apipb.PluginUiMountUpdate) (state.PluginOwner, string, error) {
	owner := pluginOwner(update.GetOwner())
	dynamic := state.IsDynamicPluginScope(update.GetScope())
	if update.GetScope() != "" || owner.Kind == "" {
		scope := update.GetScope()
		if scope == "" {
			scope = "workspace"
		}
		var ok bool
		owner, ok = state.PluginOwnerForScope(root.Shell, scope)
		if !ok {
			// Keep a typed, incomplete owner while an active tab/panel is absent.
			// ReconcileDynamicOwners fills its IDs as soon as the shell creates it.
			shell := root.Shell.ReadonlyDefaults()
			if shell.Workspace.ID == "" {
				return state.PluginOwner{}, "", fmt.Errorf("plugin surface scope %q has no workspace", scope)
			}
			switch scope {
			case "active_tab":
				owner = state.PluginOwner{Kind: "tab", WorkspaceID: shell.Workspace.ID}
			case "active_panel":
				owner = state.PluginOwner{Kind: "panel", WorkspaceID: shell.Workspace.ID}
			default:
				return state.PluginOwner{}, "", fmt.Errorf("plugin surface scope %q has no active target", scope)
			}
		}
	}
	slot := update.GetSlot()
	if placement := update.GetPlacement(); placement != "" {
		slot = map[string]string{"sidebar": "sidebar", "statusbar": "statusbar", "floating": "overlay", "overlay": "overlay", "menu": "menu", "header": "header", "content": "content"}[placement]
	}
	if !state.ValidPluginSlot(owner.Kind, slot) {
		return state.PluginOwner{}, "", fmt.Errorf("plugin surface %q is invalid for %s owner", slot, owner.Kind)
	}
	if !owner.Exists(root.Shell) && !dynamic {
		return state.PluginOwner{}, "", fmt.Errorf("plugin surface owner is unavailable")
	}
	return owner, slot, nil
}
func pluginNodes(node *apipb.PluginUiNode, deps PluginDeps) []state.PluginNode {
	if node == nil {
		return nil
	}
	n := state.PluginNode{ID: node.GetId(), Kind: node.GetKind(), Text: node.GetText(), Action: node.GetActionId(), ItemID: node.GetItemId(), Value: node.GetValue(), Description: node.GetDescription(), Placeholder: node.GetPlaceholder(), Status: node.GetStatus(), Disabled: node.GetDisabled(), Progress: node.GetProgress(), Style: pluginNodeStyle(node.GetStyle()), SelectedStyle: pluginNodeStyle(node.GetSelectedStyle()), Layout: pluginNodeLayout(node.GetLayout())}
	if ref := node.GetTerminal(); ref != nil {
		n.DaemonID = ref.GetDaemonId()
		n.TerminalID = ref.GetTerminalId()
		if deps.Service != nil {
			n.EndpointID, _ = deps.Service.ResolveDaemon(n.DaemonID)
		}
	}
	for _, c := range node.GetChildren() {
		n.Children = append(n.Children, pluginNodes(c, deps)...)
	}
	return []state.PluginNode{n}
}

func pluginNodeStyle(style *apipb.PluginUiStyle) state.PluginNodeStyle {
	if style == nil {
		return state.PluginNodeStyle{}
	}
	return state.PluginNodeStyle{ForegroundRole: style.GetForegroundRole(), BackgroundRole: style.GetBackgroundRole(), Bold: style.GetBold(), Dim: style.GetDim()}
}

func pluginNodeLayout(layout *apipb.PluginUiLayout) state.PluginNodeLayout {
	if layout == nil {
		return state.PluginNodeLayout{}
	}
	return state.PluginNodeLayout{PaddingTop: int(layout.GetPaddingTop()), PaddingRight: int(layout.GetPaddingRight()), PaddingBottom: int(layout.GetPaddingBottom()), PaddingLeft: int(layout.GetPaddingLeft()), GapAfter: int(layout.GetGapAfter())}
}
func reducePluginDelivery(root state.Root, d port.PluginDelivery, deps PluginDeps) (state.Root, []Effect) {
	if d.Err != nil {
		for _, m := range root.Plugins.Mounts {
			if m.EndpointID == d.EndpointID && (d.PluginID == "" || m.PluginID == d.PluginID) {
				m.Stale = true
				root.Plugins = root.Plugins.Set(m)
			}
		}
		contexts := make(map[string]state.PluginInteractionContext)
		for id, c := range root.Plugins.Contexts {
			if c.EndpointID != d.EndpointID || (d.PluginID != "" && c.PluginID != d.PluginID) {
				contexts[id] = c
			}
		}
		root.Plugins.Contexts = contexts
		root.Plugins.Error = d.Err.Error()
		return root.Advance(), nil
	}
	p := d.Message
	if p == nil || p.GetSource() == nil || p.GetSource().GetDaemonId() == "" || p.GetDestination().GetTuiInstanceId() != deps.TUIInstanceID {
		return root, nil
	}
	if p.GetDeadlineUnixMillis() > 0 && p.GetDeadlineUnixMillis() < time.Now().UnixMilli() {
		return root, []Effect{pluginReply(deps, d, "DEADLINE_EXCEEDED", "request expired")}
	}
	if p.GetUiQuery() != nil {
		return reducePluginQuery(root, d, deps)
	}
	if init := p.GetInit(); init != nil && len(init.Owners) == 0 {
		response := pluginInitForPeer(root, deps.TUIInstanceID, p.GetDestination(), p.GetSource())
		response.ReconcileRequestId = init.GetReconcileRequestId()
		peers := append([]state.PluginPeer(nil), root.Plugins.Peers...)
		found := false
		for i, peer := range peers {
			if peer.EndpointID == d.EndpointID && peer.Address.GetPluginId() == p.GetSource().GetPluginId() {
				peers[i] = state.PluginPeer{EndpointID: d.EndpointID, Address: p.GetSource(), Host: p.GetDestination()}
				found = true
			}
		}
		if !found {
			peers = append(peers, state.PluginPeer{EndpointID: d.EndpointID, Address: p.GetSource(), Host: p.GetDestination()})
		}
		root.Plugins.Peers = peers
		root.Plugins.OwnersKey = pluginOwnersKey(response.Owners)
		effects := []Effect{pluginSend(deps, d.EndpointID, &apipb.PluginMessage{RequestId: pluginID(), Destination: p.GetSource(), Body: &apipb.PluginMessage_Init{Init: response}})}
		if p.GetRequestId() != "" {
			effects = append(effects, pluginReply(deps, d, "", ""))
		}
		return root, effects
	}
	if update := p.GetMountUpdate(); update != nil {
		old, exists := root.Plugins.Mounts[update.GetMountId()]
		if exists && (old.PluginID != p.GetSource().GetPluginId() || old.DaemonID != p.GetSource().GetDaemonId()) {
			return root, []Effect{pluginReply(deps, d, "PERMISSION_DENIED", "mount belongs to another plugin")}
		}
		if update.GetClose() {
			if exists && update.GetExpectedRevision() != old.Revision {
				return root, []Effect{pluginReply(deps, d, "CONFLICT", "stale close revision")}
			}
			if exists {
				root.Plugins = root.Plugins.Remove(old.ID)
			}
			return root.Advance(), []Effect{pluginReply(deps, d, "", "")}
		}
		if err := validatePluginMount(update); err != nil {
			return root, []Effect{pluginReply(deps, d, "UNSUPPORTED", err.Error())}
		}
		owner, slot, err := pluginMountOwner(root, update)
		if err != nil {
			return root, []Effect{pluginReply(deps, d, "UNSUPPORTED", err.Error())}
		}
		mount := state.PluginMount{ID: update.GetMountId(), PluginID: p.GetSource().GetPluginId(), DaemonID: p.GetSource().GetDaemonId(), EndpointID: d.EndpointID, Owner: owner, SurfaceID: update.GetSurfaceId(), Placement: update.GetPlacement(), Scope: update.GetScope(), Slot: slot, Title: update.GetTitle(), Revision: update.GetRevision(), Nodes: pluginNodes(update.GetRoot(), deps), Interactive: true, Width: int(update.GetPreferredWidth()), Source: proto.Clone(p.GetSource()).(*apipb.PluginAddress), Actions: update.GetActions(), Hideable: update.GetHideable(), Closeable: update.GetCloseable()}
		if exists && old.Stale && update.GetExpectedRevision() == 0 {
			root.Plugins = root.Plugins.Remove(old.ID)
		}
		next, err := root.Plugins.Apply(root.Shell, mount, update.GetExpectedRevision())
		if err != nil {
			return root, []Effect{pluginReply(deps, d, "CONFLICT", err.Error())}
		}
		root.Plugins = next
		if update.GetFocus() && mount.Owner.Visible(root.Shell) && !root.Plugins.Mounts[mount.ID].Hidden {
			root.Plugins.LastContentPaneID = root.Shell.ReadonlyDefaults().ActivePaneID
			root.Plugins.FocusedMountID = mount.ID
		}
		return root.Advance(), []Effect{pluginReply(deps, d, "", "")}
	}
	if reply := p.GetReply(); reply != nil && reply.GetError() != nil {
		root.Plugins.Error = reply.GetError().GetMessage()
		return root.Advance(), nil
	}
	if op := p.GetOperation(); op != nil {
		return reducePluginOperation(root, d, deps)
	}
	return root, nil
}
func pluginBindingRevision(root state.Root, owner state.PluginOwner) uint64 {
	h := fnv.New64a()
	var b state.TerminalViewBinding
	if owner.FloatingID != "" {
		b, _ = root.TerminalViews.FloatingBinding(owner.FloatingID)
	} else {
		b, _ = root.TerminalViews.PaneBinding(owner.PaneID)
	}
	if b.AttachCandidate != nil && !b.AttachCandidate.HadBinding {
		b = state.TerminalViewBinding{}
	}
	fmt.Fprintf(h, "%s|%s|%s|%s|%s|%d", owner.WorkspaceID, owner.TabID, b.ViewID, b.EndpointID, b.TerminalID, b.Channel)
	return h.Sum64()
}
func pluginActionForID(m state.PluginMount, id string) *apipb.PluginUiAction {
	for _, action := range m.Actions {
		if action.GetId() == id {
			return action
		}
	}
	return nil
}

func pluginActionTargetPolicy(m state.PluginMount, id string) string {
	if action := pluginActionForID(m, id); action != nil && action.GetTargetPolicy() != "" {
		return action.GetTargetPolicy()
	}
	return "focused_panel"
}

func pluginCapture(root state.Root, m state.PluginMount, node state.PluginNode, deps PluginDeps) (state.Root, *apipb.PluginTargetContext, state.EndpointID) {
	shell := root.Shell.ReadonlyDefaults()
	policy := pluginActionTargetPolicy(m, node.Action)
	pane := ""
	owner := state.PluginOwner{}
	switch policy {
	case "active_panel":
		pane = shell.ActivePaneID
	case "source_panel":
		if m.Owner.Kind == "panel" {
			owner = m.Owner
			pane = owner.PaneID
		} else {
			pane = root.Plugins.LastContentPaneID
			if pane == "" {
				pane = shell.ActivePaneID
			}
		}
	case "none":
		// UI-only actions still carry a daemon-routed context, but do not pin
		// themselves to a terminal panel.
	default:
		pane = root.Plugins.LastContentPaneID
		if pane == "" {
			pane = shell.ActivePaneID
		}
	}
	if pane != "" && owner.Kind == "" {
		owner = state.PluginOwner{Kind: "panel", WorkspaceID: shell.Workspace.ID, TabID: shell.Workspace.ActiveTabID, PaneID: pane}
	}
	endpoint := m.EndpointID
	if node.DaemonID != "" && deps.Service != nil {
		if ep, ok := deps.Service.ResolveDaemon(node.DaemonID); ok {
			endpoint = ep
		}
	}
	c := &apipb.PluginTargetContext{ContextId: pluginID(), TuiInstanceId: deps.TUIInstanceID, WorkspaceId: owner.WorkspaceID, TabId: owner.TabID, PaneId: pane, MountId: m.ID, TargetPolicy: policy, SurfaceId: m.SurfaceID}
	if pane != "" {
		c.BindingRevision = pluginBindingRevision(root, owner)
		if owner.FloatingID != "" {
			c.ViewId = liveAttachFloatingViewID(root, owner.FloatingID)
		} else {
			c.ViewId = liveAttachPaneViewID(root, owner.PaneID)
		}
	}
	contexts := make(map[string]state.PluginInteractionContext)
	for k, v := range root.Plugins.Contexts {
		if v.Expires > time.Now().UnixMilli() {
			contexts[k] = v
		}
	}
	contexts[c.ContextId] = state.PluginInteractionContext{Context: c, PluginID: m.PluginID, EndpointID: endpoint, Expires: time.Now().Add(30 * time.Second).UnixMilli()}
	root.Plugins.Contexts = contexts
	return root, c, endpoint
}
func reducePluginInput(root state.Root, msg PluginInputMsg, deps PluginDeps) (state.Root, []Effect) {
	m, ok := root.Plugins.Mounts[msg.MountID]
	if !ok || !m.Owner.Visible(root.Shell) || m.Hidden || !m.Interactive {
		return root, nil
	}
	e := msg.Event
	handled := []Effect{handledEffect{}}
	if e.Kind == input.EventKindMouse {
		if root.Plugins.FocusedMountID != m.ID {
			root.Plugins.LastContentPaneID = root.Shell.ReadonlyDefaults().ActivePaneID
		}
		root.Plugins.FocusedMountID = m.ID
		switch e.Mouse {
		case input.MouseWheelUp:
			m = m.Move(-1)
		case input.MouseWheelDown:
			m = m.Move(1)
		case input.MouseLeft:
			if msg.NodeID != "" {
				m.SelectedID = msg.NodeID
			}
			if next, effects, handled := pluginFormInput(root, m, e, deps); handled {
				return next, effects
			}
			// Cards represent directly actionable items. A single click should
			// execute the card action after selecting it; plain rows retain the
			// traditional select-then-double-click behavior.
			if node, ok := m.Selected(); ok && node.Kind == "card" && node.Action != "" {
				return pluginActivate(root, m, deps)
			}
			now := time.Now().UnixMilli()
			double := root.Plugins.LastClickID == m.ID+"/"+msg.NodeID && now-root.Plugins.LastClickMillis < 450
			root.Plugins.LastClickID = m.ID + "/" + msg.NodeID
			root.Plugins.LastClickMillis = now
			if double {
				return pluginActivate(root, m, deps)
			}
		}
		root.Plugins = root.Plugins.Set(m)
		return root.Advance(), handled
	}
	if next, effects, handled := pluginFormInput(root, m, e, deps); handled {
		return next, effects
	}
	if e.Kind != input.EventKindKey {
		return root, handled
	}
	if e.Ctrl || e.Alt {
		if _, ok := input.ShortcutEntryForEvent(root.Config.Shortcuts, "global", e); ok {
			return root, nil
		}
		return root, handled
	}
	switch e.Key {
	case input.KeyEsc:
		if m.Search != "" || m.Searching {
			m.Search = ""
			m.Searching = false
		} else {
			root.Plugins.FocusedMountID = ""
		}
	case input.KeyTab:
		m = m.MoveFocus(1)
	case input.KeyShiftTab:
		m = m.MoveFocus(-1)
	case input.KeyUp:
		m = m.Move(-1)
	case input.KeyDown:
		m = m.Move(1)
	case input.KeyEnter:
		return pluginActivate(root, m, deps)
	case input.KeyBackspace:
		if m.Searching {
			r := []rune(m.Search)
			if len(r) > 0 {
				m.Search = string(r[:len(r)-1])
			}
		}
	case input.KeyChar:
		if m.Searching {
			m.Search += e.Char
		} else {
			switch e.Char {
			case "j":
				m = m.Move(1)
			case "k":
				m = m.Move(-1)
			case "/":
				m.Searching = true
			}
		}
	}
	root.Plugins = root.Plugins.Set(m)
	return root.Advance(), handled
}
func pluginActivate(root state.Root, m state.PluginMount, deps PluginDeps) (state.Root, []Effect) {
	return pluginActivateAction(root, m, "", deps)
}
func pluginActivateAction(root state.Root, m state.PluginMount, action string, deps PluginDeps, events ...input.InputEvent) (state.Root, []Effect) {
	meta := pluginActionForID(m, action)
	behavior := "activate"
	if meta != nil && meta.GetBehavior() != "" {
		behavior = meta.GetBehavior()
	}
	switch behavior {
	case "hide":
		if !m.Hideable {
			return root, []Effect{handledEffect{}}
		}
		m.Hidden = true
		root.Plugins.FocusedMountID = ""
		root.Plugins = root.Plugins.Set(m)
		return root.Advance(), []Effect{handledEffect{}}
	case "show":
		m.Hidden = false
		root.Plugins = root.Plugins.Set(m)
		if m.Owner.Visible(root.Shell) {
			root.Plugins.FocusedMountID = m.ID
		}
		return root.Advance(), []Effect{handledEffect{}}
	case "toggle":
		if !m.Hideable {
			return root, []Effect{handledEffect{}}
		}
		m.Hidden = !m.Hidden
		root.Plugins = root.Plugins.Set(m)
		if m.Hidden {
			if root.Plugins.FocusedMountID == m.ID {
				root.Plugins.FocusedMountID = ""
			}
		} else if m.Owner.Visible(root.Shell) {
			root.Plugins.FocusedMountID = m.ID
		}
		return root.Advance(), []Effect{handledEffect{}}
	case "close":
		if !m.Closeable {
			return root, []Effect{handledEffect{}}
		}
		root.Plugins = root.Plugins.Remove(m.ID)
		if m.Source == nil {
			return root.Advance(), []Effect{handledEffect{}}
		}
		return root.Advance(), []Effect{handledEffect{}, pluginSend(deps, m.EndpointID, &apipb.PluginMessage{RequestId: pluginID(), Destination: m.Source, Body: &apipb.PluginMessage_Interaction{Interaction: &apipb.PluginUiInteraction{MountId: m.ID, MountRevision: m.Revision, ActionId: action, Kind: "close"}}})}
	}
	node, ok := m.Selected()
	if action != "" {
		node.Action = action
		node.Disabled = false
		if !ok {
			node.ID = "action:" + action
			ok = true
		}
	}
	if !ok || node.Disabled || node.Action == "" || m.Stale {
		return root, []Effect{handledEffect{}}
	}
	event := input.InputEvent{}
	if len(events) > 0 {
		event = events[0]
	}
	return pluginEmitInteraction(root, m, node, "submit", deps, event)
}
func pluginEmitInteraction(root state.Root, m state.PluginMount, node state.PluginNode, kind string, deps PluginDeps, event input.InputEvent) (state.Root, []Effect) {
	if m.Stale || m.Source == nil {
		return root, []Effect{handledEffect{}}
	}
	root.Plugins = root.Plugins.Set(m)
	root, c, ep := pluginCapture(root, m, node, deps)
	dest := proto.Clone(m.Source).(*apipb.PluginAddress)
	if node.DaemonID != "" && node.DaemonID != dest.DaemonId {
		dest.DaemonId = node.DaemonID
		dest.RegistrationEpoch = 0
	}
	interaction := &apipb.PluginUiInteraction{MountId: m.ID, MountRevision: m.Revision, NodeId: node.ID, ActionId: node.Action, ItemId: node.ItemID, Value: node.Value, Values: m.FormValues(), Context: c, Kind: kind}
	if event.Ctrl {
		interaction.Modifiers = append(interaction.Modifiers, "ctrl")
	}
	if event.Alt {
		interaction.Modifiers = append(interaction.Modifiers, "alt")
	}
	if event.Shift {
		interaction.Modifiers = append(interaction.Modifiers, "shift")
	}
	return root.Advance(), []Effect{handledEffect{}, pluginSend(deps, ep, &apipb.PluginMessage{RequestId: pluginID(), Destination: dest, DeadlineUnixMillis: time.Now().Add(30 * time.Second).UnixMilli(), Body: &apipb.PluginMessage_Interaction{Interaction: interaction}})}
}

func reducePluginOperation(root state.Root, d port.PluginDelivery, deps PluginDeps) (state.Root, []Effect) {
	op := d.Message.GetOperation()
	c := op.GetContext()
	saved, ok := root.Plugins.Contexts[c.GetContextId()]
	reject := func(code, detail string) (state.Root, []Effect) {
		root.Plugins.Error = detail
		return root.Advance(), []Effect{pluginReply(deps, d, code, detail)}
	}
	switch {
	case !ok:
		return reject("STALE_CONTEXT", "interaction context is no longer registered")
	case saved.Pending:
		return reject("STALE_CONTEXT", "interaction context was already consumed")
	case saved.Expires < time.Now().UnixMilli():
		return reject("STALE_CONTEXT", "interaction context expired")
	case saved.PluginID != d.Message.GetSource().GetPluginId():
		return reject("STALE_CONTEXT", "interaction context belongs to another plugin")
	case saved.EndpointID != d.EndpointID:
		return reject("STALE_CONTEXT", "interaction context belongs to another endpoint")
	case !proto.Equal(saved.Context, c):
		return reject("STALE_CONTEXT", "interaction context was modified")
	}
	if n := op.GetNotification(); n != nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: n.GetTitle(), Body: n.GetBody()})
		return root.Advance(), []Effect{pluginReply(deps, d, "", "")}
	}
	if c.GetTargetPolicy() == "none" {
		return reject("NO_TARGET", "this plugin action does not target a panel")
	}
	owner := state.PluginOwner{Kind: "panel", WorkspaceID: c.GetWorkspaceId(), TabID: c.GetTabId(), PaneID: c.GetPaneId()}
	if c.GetFloatingId() != "" {
		owner.Kind = "floating"
		owner.FloatingID = c.GetFloatingId()
	}
	if !owner.Exists(root.Shell) || pluginBindingRevision(root, owner) != c.GetBindingRevision() {
		return reject("CONFLICT", "target binding changed")
	}
	bind := op.GetBind()
	if bind == nil {
		return reject("UNSUPPORTED", "unsupported UI operation")
	}
	if deps.Service == nil || deps.Live.Terminal == nil {
		return reject("TARGET_OFFLINE", "terminal service unavailable")
	}
	ep, ok := deps.Service.ResolveDaemon(bind.GetTerminal().GetDaemonId())
	if !ok {
		return reject("TARGET_OFFLINE", "terminal daemon is not connected")
	}
	viewID := liveAttachPaneViewID(root, owner.PaneID)
	if owner.FloatingID != "" {
		viewID = liveAttachFloatingViewID(root, owner.FloatingID)
	}
	cfg := LiveConfig{EndpointID: ep, TerminalID: bind.GetTerminal().GetTerminalId(), ViewID: viewID, SurfaceID: runtimeSurfaceID(root), Mode: "raw", ResizePolicy: state.TerminalResizeRoleFollower}
	contexts := make(map[string]state.PluginInteractionContext, len(root.Plugins.Contexts))
	for id, value := range root.Plugins.Contexts {
		contexts[id] = value
	}
	saved.Pending = true
	contexts[c.GetContextId()] = saved
	root.Plugins.Contexts = contexts
	next, effects := reduceLiveAttach(root, LiveAttachMsg{Config: cfg}, deps.Live)
	if len(effects) == 0 {
		return reject("CONFLICT", "target cannot attach")
	}
	for i, e := range effects {
		if f, ok := e.(FuncEffect); ok {
			run := f.Run
			f.Run = func(ctx context.Context) Msg {
				r := run(ctx)
				if a, ok := r.(LiveAttachResultMsg); ok {
					return PluginBindResultMsg{Delivery: d, Attach: a, Target: owner, Revision: c.GetBindingRevision()}
				}
				return r
			}
			effects[i] = f
		}
	}
	return next, effects
}
func reducePluginBindResult(root state.Root, m PluginBindResultMsg, deps PluginDeps) (state.Root, []Effect) {
	if !m.Target.Exists(root.Shell) || pluginBindingRevision(root, m.Target) != m.Revision {
		root.TerminalViews, _ = root.TerminalViews.FailAttach(m.Attach.ViewID, m.Attach.OperationID, "plugin target changed")
		effects := cleanupAttachResultEffects(m.Attach.Result)
		return root, append(effects, pluginReply(deps, m.Delivery, "CONFLICT", "target changed during attach"))
	}
	b, ok := root.TerminalViews.Views[m.Attach.ViewID]
	if !ok || b.AttachCandidate == nil || b.AttachCandidate.OperationID != m.Attach.OperationID {
		return root, append(cleanupAttachResultEffects(m.Attach.Result), pluginReply(deps, m.Delivery, "CONFLICT", "attach superseded"))
	}
	next, effects := reduceLiveAttachResult(root, m.Attach, deps.Live)
	code, detail := "", ""
	if m.Attach.Err != nil {
		code = "ATTACH_FAILED"
		detail = m.Attach.Err.Error()
		next.Plugins.Error = detail
	}
	if code == "" {
		next.Plugins.FocusedMountID = ""
		effects = append(effects, FuncEffect{Run: func(context.Context) Msg { return WorkbenchStoragePersistRequestMsg{Reason: "plugin.bind"} }})
	}
	return next, append(effects, pluginReply(deps, m.Delivery, code, detail))
}

// pluginInitForPeer exposes authoritative revisions only within the requesting
// plugin's daemon domain. Reconciliation does not transfer mount ownership.
func pluginInitForPeer(root state.Root, tuiID string, host, source *apipb.PluginAddress) *apipb.PluginUiInit {
	init := pluginInit(root, tuiID, host)
	init.MountRevisions = make(map[string]uint64)
	if source.GetPluginId() == "" || source.GetDaemonId() == "" {
		return init
	}
	for id, mount := range root.Plugins.Mounts {
		if mount.PluginID == source.GetPluginId() && mount.DaemonID == source.GetDaemonId() {
			init.MountRevisions[id] = mount.Revision
		}
	}
	return init
}

func pluginInit(root state.Root, tuiID string, host *apipb.PluginAddress) *apipb.PluginUiInit {
	shell := root.Shell.ReadonlyDefaults()
	init := &apipb.PluginUiInit{Host: host, Context: &apipb.PluginTargetContext{TuiInstanceId: tuiID, WorkspaceId: shell.Workspace.ID, TabId: shell.Workspace.ActiveTabID, PaneId: shell.ActivePaneID}}
	workspaces := append([]state.WorkspaceState(nil), shell.Workspaces...)
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
		init.Owners = append(init.Owners, &apipb.PluginMountOwner{Owner: &apipb.PluginMountOwner_Workspace{Workspace: &apipb.PluginWorkspaceOwner{WorkspaceId: w.ID}}})
		for _, t := range w.Tabs {
			init.Owners = append(init.Owners, &apipb.PluginMountOwner{Owner: &apipb.PluginMountOwner_Tab{Tab: &apipb.PluginTabOwner{WorkspaceId: w.ID, TabId: t.ID}}})
			for _, p := range t.Panes {
				init.Owners = append(init.Owners, &apipb.PluginMountOwner{Owner: &apipb.PluginMountOwner_Panel{Panel: &apipb.PluginPanelOwner{WorkspaceId: w.ID, TabId: t.ID, PaneId: p.ID}}})
			}
			for _, f := range t.Floatings {
				init.Owners = append(init.Owners, &apipb.PluginMountOwner{Owner: &apipb.PluginMountOwner_Floating{Floating: &apipb.PluginFloatingOwner{WorkspaceId: w.ID, TabId: t.ID, FloatingId: f.ID}}})
			}
		}
	}
	return init
}
func pluginShortcut(root state.Root, event input.InputEvent, deps PluginDeps) (state.Root, []Effect, bool) {
	if event.Kind != input.EventKindKey || root.Shell.Overlay.Open || root.Shell.InteractionMode != state.InteractionModeNormal || event.Key == input.KeyEsc {
		return root, nil, false
	}
	if m, ok := root.Plugins.Mounts[root.Plugins.FocusedMountID]; ok && (m.EditingID != "" || m.Searching) {
		return root, nil, false
	}
	// Configured and built-in host command prefixes always take precedence.
	if _, ok := input.ShortcutEntryForEvent(root.Config.Shortcuts, "global", event); ok {
		return root, nil, false
	}
	type match struct {
		mount  state.PluginMount
		action *apipb.PluginUiAction
		rank   int
	}
	var matches []match
	for _, m := range root.Plugins.Mounts {
		if !m.Owner.Exists(root.Shell) {
			continue
		}
		for _, a := range m.Actions {
			if !a.GetEnabled() {
				continue
			}
			scope := a.GetScope()
			key := a.GetDefaultKey()
			override, configured := root.Config.PluginShortcuts[m.PluginID+"/"+a.GetId()]
			if configured {
				key = override
			}
			if key == "" {
				continue
			}
			if scope != "global" && (!m.Owner.Visible(root.Shell) || m.Hidden) {
				continue
			}
			rank := 0
			switch scope {
			case "", "mount":
				if root.Plugins.FocusedMountID != m.ID {
					continue
				}
			case "panel":
				rank = 1
				if m.Owner.PaneID != root.Shell.ReadonlyDefaults().ActivePaneID {
					continue
				}
			case "floating":
				rank = 1
				if m.Owner.FloatingID != root.Shell.ReadonlyDefaults().ActiveFloatingID() {
					continue
				}
			case "tab":
				rank = 2
				if m.Owner.TabID != root.Shell.ReadonlyDefaults().Workspace.ActiveTabID {
					continue
				}
			case "workspace":
				rank = 3
			case "global":
				if !configured {
					continue
				}
				rank = 4
			default:
				continue
			}
			if input.MatchPluginKey(key, event) {
				if len(matches) == 0 || rank < matches[0].rank {
					matches = nil
				}
				if len(matches) == 0 || rank == matches[0].rank {
					matches = append(matches, match{m, a, rank})
				}
			}
		}
	}
	if len(matches) > 1 {
		root.Plugins.Error = "Conflicting plugin shortcuts; binding disabled"
		return root.Advance(), []Effect{handledEffect{}}, true
	}
	if len(matches) == 0 {
		return root, nil, false
	}
	m := matches[0].mount
	a := matches[0].action
	next, effects := pluginActivateAction(root, m, a.GetId(), deps, event)
	return next, effects, true
}

// NewPluginMaintenanceReducer runs after shell reducers so closing an owner
// releases its mounts immediately, without waiting for the next user input.
func NewPluginMaintenanceReducer(deps PluginDeps) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		var changed []string
		root.Plugins, changed = root.Plugins.ReconcileDynamicOwners(root.Shell)
		if len(changed) > 0 {
			contexts := make(map[string]state.PluginInteractionContext, len(root.Plugins.Contexts))
			for id, context := range root.Plugins.Contexts {
				remove := false
				for _, mountID := range changed {
					if context.Context != nil && context.Context.GetMountId() == mountID {
						remove = true
						break
					}
				}
				if !remove {
					contexts[id] = context
				}
			}
			root.Plugins.Contexts = contexts
		}
		var removed []state.PluginMount
		root.Plugins, removed = root.Plugins.Prune(root.Shell)
		contexts := make(map[string]state.PluginInteractionContext)
		for id, c := range root.Plugins.Contexts {
			if _, ok := root.Plugins.Mounts[c.Context.GetMountId()]; ok {
				contexts[id] = c
			}
		}
		root.Plugins.Contexts = contexts
		var effects []Effect
		for _, m := range removed {
			if m.Source != nil {
				effects = append(effects, pluginSend(deps, m.EndpointID, &apipb.PluginMessage{RequestId: pluginID(), Destination: m.Source, Body: &apipb.PluginMessage_Interaction{Interaction: &apipb.PluginUiInteraction{MountId: m.ID, MountRevision: m.Revision, Kind: "close"}}}))
			}
		}
		if len(root.Plugins.Peers) > 0 {
			init := pluginInit(root, deps.TUIInstanceID, nil)
			key := pluginOwnersKey(init.Owners)
			if key != root.Plugins.OwnersKey {
				root.Plugins.OwnersKey = key
				for _, peer := range root.Plugins.Peers {
					response := pluginInitForPeer(root, deps.TUIInstanceID, peer.Host, peer.Address)
					effects = append(effects, pluginSend(deps, peer.EndpointID, &apipb.PluginMessage{RequestId: pluginID(), Destination: peer.Address, Body: &apipb.PluginMessage_Init{Init: response}}))
				}
			}
		}
		return root, effects
	}
}

func pluginOwnersKey(owners []*apipb.PluginMountOwner) string {
	h := fnv.New64a()
	for _, o := range owners {
		b, _ := proto.Marshal(o)
		h.Write(b)
		h.Write([]byte{0})
	}
	return fmt.Sprint(h.Sum64())
}

func pluginFocusNext(root state.Root, delta int) state.Root {
	var mounts []state.PluginMount
	for _, m := range root.Plugins.Visible(root.Shell) {
		if m.Interactive && m.Slot != "header" && m.Slot != "statusbar" {
			mounts = append(mounts, m)
		}
	}
	// A hidden surface must remain recoverable through the host's generic
	// plugin-focus command even when its plugin has no global toggle action.
	if len(mounts) == 0 {
		for _, m := range root.Plugins.Mounts {
			if m.Hidden && m.Interactive && m.Owner.Visible(root.Shell) && m.Slot != "header" && m.Slot != "statusbar" {
				mounts = append(mounts, m)
			}
		}
		sort.Slice(mounts, func(i, j int) bool { return mounts[i].ID < mounts[j].ID })
	}
	if len(mounts) == 0 {
		root.Plugins.Error = "No interactive plugin views available"
		return root.Advance()
	}
	selected := -1
	for i, m := range mounts {
		if m.ID == root.Plugins.FocusedMountID {
			selected = i
			break
		}
	}
	if selected < 0 && delta < 0 {
		selected = 0
	}
	selected = (selected + delta + len(mounts)) % len(mounts)
	if root.Plugins.FocusedMountID == "" {
		root.Plugins.LastContentPaneID = root.Shell.ReadonlyDefaults().ActivePaneID
	}
	if mounts[selected].Hidden {
		mounts[selected].Hidden = false
		root.Plugins = root.Plugins.Set(mounts[selected])
	}
	root.Plugins.FocusedMountID = mounts[selected].ID
	root.Shell = root.Shell.SetInteractionMode(state.InteractionModeNormal)
	return root.Advance()
}

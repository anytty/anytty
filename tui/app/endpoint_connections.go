package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

// ShellOpenConnectionsMsg 请求打开独立 Connections 页面；真实策略由随后触发的 LoadConnections effect 刷新。
type ShellOpenConnectionsMsg struct{}

func (ShellOpenConnectionsMsg) isMsg() {}

// ConnectionsLoadRequestMsg 请求 effect 从 Go Endpoint registry/planner 读取最新连接投影。
type ConnectionsLoadRequestMsg struct{}

func (ConnectionsLoadRequestMsg) isMsg() {}

// ConnectionsLoadResultMsg 把 adapter 返回的连接投影送回 reducer；错误不会清空当前页面数据。
type ConnectionsLoadResultMsg struct {
	Store state.EndpointStore
	Err   error
}

func (ConnectionsLoadResultMsg) isMsg() {}

const (
	connectionsSnapshotRefreshToken    CancelToken   = "connections.snapshot.refresh"
	connectionsSnapshotRefreshInterval time.Duration = time.Second
	connectionsRegistryWatchToken      CancelToken   = "connections.registry.watch"
)

// ConnectionsWatchRequestMsg starts the shared endpoint registry subscription for this TUI process.
type ConnectionsWatchRequestMsg struct{}

func (ConnectionsWatchRequestMsg) isMsg()           {}
func (ConnectionsWatchRequestMsg) SkipRender() bool { return true }

// ConnectionsWatchUpdateMsg carries one externally committed registry projection.
type ConnectionsWatchUpdateMsg struct {
	Update port.EndpointConnectionUpdate
}

func (ConnectionsWatchUpdateMsg) isMsg() {}

// ConnectionsApplySettingsRequestMsg requests one atomic policy and priority update.
type ConnectionsApplySettingsRequestMsg struct {
	EndpointID state.EndpointID
	Policy     state.EndpointConnectionPolicy
	Priorities map[string]*int
}

func (ConnectionsApplySettingsRequestMsg) isMsg() {}

type ConnectionsApplySettingsResultMsg struct {
	EndpointID state.EndpointID
	Store      state.EndpointStore
	Err        error
}

func (ConnectionsApplySettingsResultMsg) isMsg() {}

// ConnectionsSetEnabledRequestMsg persists one Endpoint's participation in automatic observation and new connections.
type ConnectionsSetEnabledRequestMsg struct {
	EndpointID state.EndpointID
	Enabled    bool
}

func (ConnectionsSetEnabledRequestMsg) isMsg() {}

// ConnectionsSetEnabledResultMsg carries both the durable registry projection and the runtime drain outcome.
type ConnectionsSetEnabledResultMsg struct {
	EndpointID state.EndpointID
	Enabled    bool
	Store      state.EndpointStore
	Err        error
}

func (ConnectionsSetEnabledResultMsg) isMsg() {}

type ConnectionsSnapshotRefreshMsg struct{}

func (ConnectionsSnapshotRefreshMsg) isMsg()           {}
func (ConnectionsSnapshotRefreshMsg) SkipRender() bool { return true }

type ConnectionsSnapshotResultMsg struct {
	EndpointID state.EndpointID
	Snapshot   state.EndpointConnectionSnapshot
	Valid      bool
	Err        error
}

func (ConnectionsSnapshotResultMsg) isMsg() {}

// NewEndpointConnectionsReducer 只编排 Connections 的 load/apply effect 与结果投影。
// priority 校验、registry 原子事务和 planner availability 均由 port 实现负责，reducer 不建立第二份连接策略。
func NewEndpointConnectionsReducer(deps LiveDeps) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		switch msg := msg.(type) {
		case ConnectionsLoadRequestMsg:
			return root, []Effect{connectionsLoadEffect(deps)}
		case ConnectionsWatchRequestMsg:
			return reduceConnectionsWatchRequest(root, deps)
		case ConnectionsWatchUpdateMsg:
			return reduceConnectionsWatchUpdate(root, msg)
		case ConnectionsLoadResultMsg:
			if msg.Err != nil {
				root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastError, Title: "Connections", Body: msg.Err.Error()})
				return root.Advance(), nil
			}
			selectedID := selectedConnectionsEndpointID(root)
			root.Endpoints = root.Endpoints.ApplyConnectionProjection(msg.Store)
			root.Shell = restoreConnectionsSelection(root.Shell, root.Endpoints, selectedID)
			return root.Advance(), connectionsSnapshotEffects(root, deps)
		case ConnectionsApplySettingsRequestMsg:
			return root, []Effect{connectionsApplyEffect(deps, msg)}
		case ConnectionsApplySettingsResultMsg:
			return reduceConnectionsApplyResult(root, msg, deps)
		case ConnectionsSetEnabledRequestMsg:
			return root, []Effect{connectionsSetEnabledEffect(deps, msg)}
		case ConnectionsSetEnabledResultMsg:
			return reduceConnectionsSetEnabledResult(root, msg, deps)
		case ConnectionsSnapshotRefreshMsg:
			return root, connectionsSnapshotEffects(root, deps)
		case ConnectionsSnapshotResultMsg:
			if msg.Err == nil && msg.Valid {
				root.Endpoints = root.Endpoints.ApplyConnectionSnapshot(msg.EndpointID, msg.Snapshot)
				root = root.Advance()
			}
			return root, connectionsSnapshotLoopEffects(root)
		default:
			return root, nil
		}
	}
}

func reduceConnectionsWatchRequest(root state.Root, deps LiveDeps) (state.Root, []Effect) {
	source, ok := deps.EndpointConnections.(port.EndpointConnectionWatchService)
	if !ok {
		return root, nil
	}
	return root, []Effect{StreamEffect{Token: connectionsRegistryWatchToken, Run: func(ctx context.Context, post func(Msg)) {
		updates, err := source.WatchConnections(ctx)
		if err != nil {
			logEffectError(deps.Logger, "connections.registry.watch", err)
			if !isContextLifecycleError(err) {
				post(ConnectionsWatchUpdateMsg{Update: port.EndpointConnectionUpdate{Err: err}})
			}
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			case update, open := <-updates:
				if !open {
					return
				}
				logEffectError(deps.Logger, "connections.registry.watch", update.Err)
				post(ConnectionsWatchUpdateMsg{Update: update})
			}
		}
	}}}
}

func reduceConnectionsWatchUpdate(root state.Root, msg ConnectionsWatchUpdateMsg) (state.Root, []Effect) {
	var effects []Effect
	if msg.Update.Store.HasItems() {
		previous := root.Endpoints
		selectedID := selectedConnectionsEndpointID(root)
		root.Endpoints = root.Endpoints.ApplyConnectionProjection(msg.Update.Store)
		root.Shell = restoreConnectionsSelection(root.Shell, root.Endpoints, selectedID)
		for _, endpointID := range newlyEnabledConnectionEndpoints(previous, root.Endpoints) {
			effects = append(effects, FuncEffect{Run: func(context.Context) Msg {
				return TerminalPoolListRequestMsg{EndpointID: endpointID, Refresh: true}
			}})
		}
		root = root.Advance()
	}
	if msg.Update.Err != nil && !isContextLifecycleError(msg.Update.Err) {
		root.Shell = root.Shell.AddToast(state.ToastSpec{
			Severity: state.ToastWarning,
			Title:    "Endpoint registry",
			Body:     msg.Update.Err.Error(),
		})
		root = root.Advance()
	}
	return root, effects
}

func newlyEnabledConnectionEndpoints(previous state.EndpointStore, current state.EndpointStore) []state.EndpointID {
	var endpointIDs []state.EndpointID
	for _, item := range current.Normalize().Items {
		if !item.Enabled || item.ConnectMode == state.EndpointConnectManual {
			continue
		}
		old, existed := previous.Endpoint(item.ID)
		if existed && old.Enabled {
			continue
		}
		endpointIDs = append(endpointIDs, item.ID)
	}
	return endpointIDs
}

func connectionsLoadEffect(deps LiveDeps) Effect {
	return FuncEffect{Run: func(ctx context.Context) Msg {
		if deps.EndpointConnections == nil {
			return ConnectionsLoadResultMsg{Err: fmt.Errorf("connection settings are unavailable")}
		}
		store, err := deps.EndpointConnections.LoadConnections(ctx)
		return ConnectionsLoadResultMsg{Store: store, Err: err}
	}}
}

func connectionsApplyEffect(deps LiveDeps, request ConnectionsApplySettingsRequestMsg) Effect {
	return FuncEffect{Run: func(ctx context.Context) Msg {
		if deps.EndpointConnections == nil {
			return ConnectionsApplySettingsResultMsg{EndpointID: request.EndpointID, Err: fmt.Errorf("connection settings are unavailable")}
		}
		store, err := deps.EndpointConnections.ApplyConnectionSettings(ctx, request.EndpointID, request.Policy, request.Priorities)
		return ConnectionsApplySettingsResultMsg{EndpointID: request.EndpointID, Store: store, Err: err}
	}}
}

func connectionsSetEnabledEffect(deps LiveDeps, request ConnectionsSetEnabledRequestMsg) Effect {
	return FuncEffect{Run: func(ctx context.Context) Msg {
		if deps.EndpointConnections == nil {
			return ConnectionsSetEnabledResultMsg{
				EndpointID: request.EndpointID, Enabled: request.Enabled,
				Err: fmt.Errorf("connection settings are unavailable"),
			}
		}
		store, err := deps.EndpointConnections.SetEndpointEnabled(ctx, request.EndpointID, request.Enabled)
		return ConnectionsSetEnabledResultMsg{
			EndpointID: request.EndpointID, Enabled: request.Enabled,
			Store: store, Err: err,
		}
	}}
}

func reduceConnectionsApplyResult(root state.Root, msg ConnectionsApplySettingsResultMsg, deps LiveDeps) (state.Root, []Effect) {
	if msg.Err != nil {
		shell := root.Shell.EnsureDefaults()
		if shell.Overlay.Kind == state.OverlayPrompt && shell.Overlay.Prompt.Purpose == "connections.settings" {
			prompt := shell.Overlay.Prompt
			prompt.Submitted = false
			prompt.LastResult = msg.Err.Error()
			shell.Overlay.Prompt = prompt
		}
		root.Shell = shell.AddToast(state.ToastSpec{Severity: state.ToastError, Title: "Connection settings", Body: msg.Err.Error()})
		return root.Advance(), nil
	}
	root.Endpoints = root.Endpoints.ApplyConnectionProjection(msg.Store)
	root.Shell = restoreConnectionsSelection(root.Shell.OpenConnections(), root.Endpoints, msg.EndpointID)
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastSuccess, Title: "Connection settings saved", Body: "Applies to the next connection"})
	return root.Advance(), connectionsSnapshotEffects(root, deps)
}

func reduceConnectionsSetEnabledResult(root state.Root, msg ConnectionsSetEnabledResultMsg, deps LiveDeps) (state.Root, []Effect) {
	if msg.Store.HasItems() {
		root.Endpoints = root.Endpoints.ApplyConnectionProjection(msg.Store)
		root.Shell = restoreConnectionsSelection(root.Shell, root.Endpoints, msg.EndpointID)
	}
	if msg.Err != nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastError, Title: "Endpoint switch", Body: msg.Err.Error()})
		return root.Advance(), connectionsSnapshotEffects(root, deps)
	}

	body := "Disabled and disconnected"
	activeViews := root.TerminalViews.AttachedBindingCountForEndpoint(msg.EndpointID)
	if msg.Enabled {
		body = "Enabled for new connections"
	} else if activeViews > 0 {
		body = fmt.Sprintf("Disabled; waiting for %d active view(s) to close", activeViews)
	}
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastSuccess, Title: "Endpoint switch", Body: body})
	effects := connectionsSnapshotEffects(root, deps)
	if item, ok := root.Endpoints.Endpoint(msg.EndpointID); msg.Enabled && ok && item.ConnectMode != state.EndpointConnectManual {
		effects = append(effects, FuncEffect{Run: func(context.Context) Msg {
			return TerminalPoolListRequestMsg{EndpointID: msg.EndpointID, Refresh: true}
		}})
	}
	return root.Advance(), effects
}

func connectionsSnapshotEffects(root state.Root, deps LiveDeps) []Effect {
	if root.Shell.EnsureDefaults().Overlay.Kind != state.OverlayConnections || deps.EndpointConnections == nil {
		return nil
	}
	selected, ok := selectedConnectionsEndpoint(root)
	if !ok {
		return connectionsSnapshotLoopEffects(root)
	}
	return []Effect{FuncEffect{Async: true, Run: func(ctx context.Context) Msg {
		snapshot, valid, err := deps.EndpointConnections.SampleConnection(ctx, selected.ID)
		return ConnectionsSnapshotResultMsg{EndpointID: selected.ID, Snapshot: snapshot, Valid: valid, Err: err}
	}}}
}

func connectionsSnapshotLoopEffects(root state.Root) []Effect {
	if root.Shell.EnsureDefaults().Overlay.Kind != state.OverlayConnections {
		return nil
	}
	return []Effect{FuncEffect{Token: connectionsSnapshotRefreshToken, Async: true, Run: func(ctx context.Context) Msg {
		timer := time.NewTimer(connectionsSnapshotRefreshInterval)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			return ConnectionsSnapshotRefreshMsg{}
		}
	}}}
}

func reduceConnectionsInput(root state.Root, event input.InputEvent) (state.Root, []Effect) {
	if event.Kind != input.EventKindKey {
		return root, nil
	}
	if entry, ok := input.ShortcutEntryForEvent(root.Config.Shortcuts, "connections", event); ok {
		return reduceOverlayShortcutAction(root, entry, event)
	}
	switch event.Key {
	case input.KeyUp:
		root.Shell = root.Shell.MoveConnectionsSelection(-1, len(root.Endpoints.Items))
		return root.Advance(), []Effect{handledEffect{}}
	case input.KeyDown:
		root.Shell = root.Shell.MoveConnectionsSelection(1, len(root.Endpoints.Items))
		return root.Advance(), []Effect{handledEffect{}}
	default:
		return root, []Effect{handledEffect{}}
	}
}

func reduceConnectionsEdit(root state.Root, row int) (state.Root, []Effect) {
	if row >= 0 {
		root.Shell = root.Shell.SetConnectionsSelectedIndex(row, len(root.Endpoints.Items))
	}
	selected, ok := selectedConnectionsEndpoint(root)
	if !ok {
		return shortcutUnavailable(root, "connections.edit", "no endpoint")
	}
	preference := selected.RoutePreference
	if preference == "" {
		preference = endpointdomain.RoutePreferenceAuto
	}
	relayMode, relayTransport := endpointdomain.RelayAuto, endpointdomain.RelayTransportAuto
	for _, route := range selected.Routes {
		if route.Kind == state.EndpointTransportHubP2P {
			if route.RelayMode != "" {
				relayMode = route.RelayMode
			}
			if route.RelayTransport != "" {
				relayTransport = route.RelayTransport
			}
			break
		}
	}
	fields := []state.PromptFieldState{
		{Key: "policy:route", Label: "Route", Value: string(preference), Placeholder: "auto/direct/ssh/managed_cloud", SuggestionItems: []string{"auto", "direct", "ssh", "managed_cloud"}},
		{Key: "policy:cloud", Label: "Cloud path", Value: string(relayMode), Placeholder: "auto/direct/relay_only/smart_route", SuggestionItems: []string{"auto", "direct", "relay_only", "smart_route"}},
		{Key: "policy:relay_transport", Label: "Relay transport", Value: string(relayTransport), Placeholder: "auto/udp/tcp", SuggestionItems: []string{"auto", "udp", "tcp"}},
	}
	for _, route := range selected.Routes {
		if !route.Enabled || route.ManualOnly {
			continue
		}
		value := ""
		if route.Priority != nil {
			value = strconv.Itoa(*route.Priority)
		}
		fields = append(fields, state.PromptFieldState{
			Key: "route:" + string(route.ID), Label: string(route.ID) + " (" + string(route.Kind) + ")", Value: value, Placeholder: "Full race",
		})
	}
	root.Shell = root.Shell.OpenPrompt(state.PromptState{
		Title: "Connection settings", Context: "Next connection only. Priorities: lower first, equal values race, all blank is Full race.",
		Purpose: "connections.settings", TargetEndpointID: selected.ID, Fields: fields,
	})
	return root.Advance(), []Effect{handledEffect{}}
}

func reduceConnectionsToggle(root state.Root, row int) (state.Root, []Effect) {
	if row >= 0 {
		root.Shell = root.Shell.SetConnectionsSelectedIndex(row, len(root.Endpoints.Items))
	}
	selected, ok := selectedConnectionsEndpoint(root)
	if !ok {
		return shortcutUnavailable(root, "connections.toggle", "no endpoint")
	}
	request := ConnectionsSetEnabledRequestMsg{
		EndpointID: selected.ID,
		Enabled:    !selected.Enabled,
	}
	return root.Advance(), []Effect{
		handledEffect{},
		FuncEffect{Run: func(context.Context) Msg { return request }},
	}
}

func connectionSettingsRequestFromPrompt(root state.Root, prompt state.PromptState) (ConnectionsApplySettingsRequestMsg, error) {
	if prompt.Purpose != "connections.settings" || prompt.TargetEndpointID == "" {
		return ConnectionsApplySettingsRequestMsg{}, fmt.Errorf("connection settings endpoint is required")
	}
	target, ok := root.Endpoints.Endpoint(prompt.TargetEndpointID)
	if !ok {
		return ConnectionsApplySettingsRequestMsg{}, fmt.Errorf("endpoint %q is unavailable", prompt.TargetEndpointID)
	}
	policy, err := connectionPolicyFromPrompt(prompt)
	if err != nil {
		return ConnectionsApplySettingsRequestMsg{}, err
	}
	automatic := make(map[string]struct{})
	for _, route := range target.Routes {
		if route.Enabled && !route.ManualOnly {
			automatic[string(route.ID)] = struct{}{}
		}
	}
	priorities := make(map[string]*int, len(prompt.Fields))
	anyValue, allValue := false, true
	for _, field := range prompt.Fields {
		if !strings.HasPrefix(field.Key, "route:") {
			continue
		}
		routeID := strings.TrimPrefix(field.Key, "route:")
		if _, ok := automatic[routeID]; !ok {
			return ConnectionsApplySettingsRequestMsg{}, fmt.Errorf("route %q is no longer automatic", routeID)
		}
		value := strings.TrimSpace(field.Value)
		anyValue = anyValue || value != ""
		allValue = allValue && value != ""
		if value == "" {
			priorities[routeID] = nil
			continue
		}
		priority, err := strconv.Atoi(value)
		if err != nil || priority < 0 {
			return ConnectionsApplySettingsRequestMsg{}, fmt.Errorf("route %q priority must be a non-negative integer", routeID)
		}
		priorityCopy := priority
		priorities[routeID] = &priorityCopy
	}
	if len(priorities) != len(automatic) {
		return ConnectionsApplySettingsRequestMsg{}, fmt.Errorf("route priority form must include every automatic route")
	}
	if anyValue && !allValue {
		return ConnectionsApplySettingsRequestMsg{}, fmt.Errorf("set every route priority or leave every field blank for Full race")
	}
	return ConnectionsApplySettingsRequestMsg{EndpointID: prompt.TargetEndpointID, Policy: policy, Priorities: priorities}, nil
}

func connectionPolicyFromPrompt(prompt state.PromptState) (state.EndpointConnectionPolicy, error) {
	policy := state.EndpointConnectionPolicy{
		RoutePreference: endpointdomain.RoutePreference(strings.TrimSpace(prompt.FieldValue("policy:route"))),
		CloudRelayMode:  endpointdomain.RelayMode(strings.TrimSpace(prompt.FieldValue("policy:cloud"))),
		RelayTransport:  endpointdomain.RelayTransport(strings.TrimSpace(prompt.FieldValue("policy:relay_transport"))),
	}
	switch policy.RoutePreference {
	case endpointdomain.RoutePreferenceAuto, endpointdomain.RoutePreferenceDirect, endpointdomain.RoutePreferenceSSH, endpointdomain.RoutePreferenceManagedCloud:
	default:
		return state.EndpointConnectionPolicy{}, fmt.Errorf("route must be auto, direct, ssh, or managed_cloud")
	}
	switch policy.CloudRelayMode {
	case endpointdomain.RelayAuto, endpointdomain.RelayDirect, endpointdomain.RelayOnly, endpointdomain.RelaySmart:
	default:
		return state.EndpointConnectionPolicy{}, fmt.Errorf("Cloud path must be auto, direct, relay_only, or smart_route")
	}
	switch policy.RelayTransport {
	case endpointdomain.RelayTransportAuto, endpointdomain.RelayTransportUDP, endpointdomain.RelayTransportTCP:
	default:
		return state.EndpointConnectionPolicy{}, fmt.Errorf("Relay transport must be auto, udp, or tcp")
	}
	return policy, nil
}

func selectedConnectionsEndpoint(root state.Root) (state.EndpointItem, bool) {
	items := root.Endpoints.Normalize().Items
	if len(items) == 0 {
		return state.EndpointItem{}, false
	}
	index := root.Shell.EnsureDefaults().Overlay.SelectedIndex
	if index < 0 || index >= len(items) {
		index = 0
	}
	return items[index], true
}

func selectedConnectionsEndpointID(root state.Root) state.EndpointID {
	item, _ := selectedConnectionsEndpoint(root)
	return item.ID
}

func restoreConnectionsSelection(shell state.ShellStore, store state.EndpointStore, endpointID state.EndpointID) state.ShellStore {
	if shell.EnsureDefaults().Overlay.Kind != state.OverlayConnections {
		return shell
	}
	for index, item := range store.Normalize().Items {
		if item.ID == endpointID {
			return shell.SetConnectionsSelectedIndex(index, len(store.Items))
		}
	}
	return shell.SetConnectionsSelectedIndex(0, len(store.Items))
}

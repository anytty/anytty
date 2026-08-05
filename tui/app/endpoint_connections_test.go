package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
)

func TestConnectionsPriorityFormIsCompleteAtomicAndPreservesRuntimeTruth(t *testing.T) {
	root := connectionsTestRoot()
	next, effects := NewShellReducer()(root, ShellOpenConnectionsMsg{})
	if next.Shell.Overlay.Kind != state.OverlayConnections || len(effects) != 1 {
		t.Fatalf("connections page did not open: shell=%#v effects=%#v", next.Shell.Overlay, effects)
	}
	next, _ = reduceConnectionsEdit(next, -1)
	prompt := next.Shell.EnsureDefaults().Overlay.Prompt
	if prompt.Purpose != "connections.settings" || len(prompt.Fields) != 5 || prompt.Fields[3].Placeholder != "Full race" || !strings.Contains(prompt.Context, "Next connection only") {
		t.Fatalf("priority prompt = %#v", prompt)
	}
	next.Shell = next.Shell.MovePromptField(3).SetPromptValue("1")
	invalid, effects := reducePromptSubmit(next)
	if len(effects) != 0 || invalid.Shell.Overlay.Kind != state.OverlayPrompt || !strings.Contains(invalid.Shell.Overlay.Prompt.LastResult, "every route") {
		t.Fatalf("partial priority should remain editable: overlay=%#v effects=%#v", invalid.Shell.Overlay, effects)
	}
	invalid.Shell = invalid.Shell.MovePromptField(1).SetPromptValue("1")
	valid, effects := reducePromptSubmit(invalid)
	if len(effects) != 1 {
		t.Fatalf("complete priority did not schedule atomic apply: %#v", effects)
	}
	request, ok := effects[0].(FuncEffect).Run(context.Background()).(ConnectionsApplySettingsRequestMsg)
	if !ok || request.Policy.RoutePreference != endpointdomain.RoutePreferenceAuto || len(request.Priorities) != 2 || request.Priorities["cloud"] == nil || request.Priorities["local"] == nil {
		t.Fatalf("priority request = %#v", request)
	}
	projection := root.Endpoints
	connected, _ := root.Endpoints.Endpoint("studio")
	connected.Label = "Renamed Studio"
	projection = projection.Upsert(connected)
	result, _ := NewEndpointConnectionsReducer(LiveDeps{})(valid, ConnectionsApplySettingsResultMsg{EndpointID: "studio", Store: projection})
	studio, _ := result.Endpoints.Endpoint("studio")
	if result.Shell.Overlay.Kind != state.OverlayConnections || studio.ActiveRouteID != "cloud" || studio.ConnectionGeneration != 9 || studio.ObservedPath != "single_relay" {
		t.Fatalf("save result overwrote runtime truth: shell=%#v endpoint=%#v", result.Shell.Overlay, studio)
	}
}

func TestConnectionsApplyFailureKeepsDraft(t *testing.T) {
	root := connectionsTestRoot()
	root.Shell = root.Shell.OpenConnections()
	root, _ = reduceConnectionsEdit(root, -1)
	root.Shell = root.Shell.MovePromptField(3).SetPromptValue("4")
	next, _ := NewEndpointConnectionsReducer(LiveDeps{})(root, ConnectionsApplySettingsResultMsg{EndpointID: "studio", Err: errors.New("registry locked")})
	if next.Shell.Overlay.Kind != state.OverlayPrompt || next.Shell.Overlay.Prompt.FieldValue("route:cloud") != "4" || !strings.Contains(next.Shell.Overlay.Prompt.LastResult, "registry locked") {
		t.Fatalf("failed apply lost prompt draft: %#v", next.Shell.Overlay.Prompt)
	}
}

func TestConnectionsSpaceTogglesSelectedEndpointAndCountsActiveViews(t *testing.T) {
	root := connectionsTestRoot()
	root.Shell = root.Shell.OpenConnections()
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView(
		"studio", "pane-1", "term-1", 7, 80, 24, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID("pane-1"), false,
	))
	_, effects := reduceConnectionsInput(root, input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: " "})
	if len(effects) != 2 {
		t.Fatalf("toggle effects = %#v", effects)
	}
	request, ok := effects[1].(FuncEffect).Run(context.Background()).(ConnectionsSetEnabledRequestMsg)
	if !ok || request.EndpointID != "studio" || request.Enabled {
		t.Fatalf("toggle request = %#v", request)
	}
}

func TestConnectionsDisableProjectsDurableStateAndReportsDraining(t *testing.T) {
	root := connectionsTestRoot()
	root.Shell = root.Shell.OpenConnections()
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView(
		"studio", "pane-1", "term-1", 7, 80, 24, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID("pane-1"), false,
	))
	projection := root.Endpoints
	studio, _ := projection.Endpoint("studio")
	studio.Enabled = false
	projection = projection.Upsert(studio)
	next, effects := NewEndpointConnectionsReducer(LiveDeps{})(root, ConnectionsSetEnabledResultMsg{
		EndpointID: "studio", Enabled: false, Store: projection,
	})
	studio, _ = next.Endpoints.Endpoint("studio")
	if studio.Enabled || studio.ActiveRouteID != "cloud" || studio.ConnectionGeneration != 9 || studio.ObservedPath != "single_relay" {
		t.Fatalf("disable projection lost durable or runtime truth: %#v", studio)
	}
	if len(effects) != 0 || len(next.Shell.Toasts) == 0 || !strings.Contains(next.Shell.Toasts[len(next.Shell.Toasts)-1].Body, "waiting for 1 active view") {
		t.Fatalf("draining result effects=%#v toasts=%#v", effects, next.Shell.Toasts)
	}
}

func TestConnectionsEnableSchedulesEndpointRefresh(t *testing.T) {
	root := connectionsTestRoot()
	root.Shell = root.Shell.OpenConnections()
	studio, _ := root.Endpoints.Endpoint("studio")
	studio.Enabled = false
	root.Endpoints = root.Endpoints.Upsert(studio)
	studio.Enabled = true
	projection := (state.EndpointStore{}).Upsert(studio)
	next, effects := NewEndpointConnectionsReducer(LiveDeps{})(root, ConnectionsSetEnabledResultMsg{
		EndpointID: "studio", Enabled: true, Store: projection,
	})
	if len(effects) != 1 {
		t.Fatalf("enable effects = %#v", effects)
	}
	request, ok := effects[0].(FuncEffect).Run(context.Background()).(TerminalPoolListRequestMsg)
	if !ok || request.EndpointID != "studio" || !request.Refresh {
		t.Fatalf("enable refresh = %#v", request)
	}
	if item, _ := next.Endpoints.Endpoint("studio"); !item.Enabled {
		t.Fatalf("enabled projection = %#v", item)
	}
}

func TestConnectionsLifecycleFailureStillProjectsSavedRegistry(t *testing.T) {
	root := connectionsTestRoot()
	root.Shell = root.Shell.OpenConnections()
	studio, _ := root.Endpoints.Endpoint("studio")
	studio.Enabled = false
	projection := (state.EndpointStore{}).Upsert(studio)
	next, _ := NewEndpointConnectionsReducer(LiveDeps{})(root, ConnectionsSetEnabledResultMsg{
		EndpointID: "studio", Enabled: false, Store: projection, Err: errors.New("disconnect failed"),
	})
	item, _ := next.Endpoints.Endpoint("studio")
	if item.Enabled || len(next.Shell.Toasts) == 0 || next.Shell.Toasts[len(next.Shell.Toasts)-1].Severity != state.ToastError {
		t.Fatalf("saved registry result was discarded: item=%#v toasts=%#v", item, next.Shell.Toasts)
	}
}

func connectionsTestRoot() state.Root {
	item := state.EndpointItem{
		ID: "studio", Label: "Studio", Enabled: true, Status: state.EndpointStatusConnected,
		ActiveRouteID: "cloud", ConnectionGeneration: 9, ObservedPath: "single_relay", RouteSelectionReason: "first_ready",
		Routes: []state.EndpointRouteItem{
			{ID: "cloud", Kind: state.EndpointTransportHubP2P, Enabled: true},
			{ID: "local", Kind: state.EndpointTransportLocal, Enabled: true},
		},
	}
	return state.Root{Shell: state.DefaultShell(), Endpoints: (state.EndpointStore{}).Upsert(item)}
}

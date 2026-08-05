package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

func TestEndpointRuntimeStatusMarksPanesWithoutGlobalToast(t *testing.T) {
	root := state.Root{
		Session: state.TerminalSessionStore{
			EndpointID:       state.DefaultEndpointID,
			TerminalID:       "local",
			Channel:          3,
			Attached:         true,
			State:            state.TerminalLiveAttached,
			InputChannels:    map[string]uint16{"local": 3, "west/remote": 9},
			ResizePolicy:     state.TerminalResizeRoleOwner,
			SurfaceID:        "surface",
			ViewID:           state.TerminalPaneViewID(state.DefaultPaneID),
			DesiredCols:      80,
			DesiredRows:      24,
			ResizeRequestSeq: 1,
		},
		Surface: state.TerminalSurfaceStore{EndpointID: state.DefaultEndpointID, TerminalID: "local", Lines: []string{"local"}, State: state.TerminalLiveAttached, Ready: true},
		Endpoints: (state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "This Mac", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true, Status: state.EndpointStatusConnected}).
			Upsert(state.EndpointItem{ID: "west", Label: "US West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true, Status: state.EndpointStatusConnected}),
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{
			{EndpointID: state.DefaultEndpointID, TerminalID: "local", Title: "local"},
			{EndpointID: "west", TerminalID: "remote", Title: "remote"},
		}},
		Shell: state.DefaultShell(),
	}
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "local", 3, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindPane(state.NewEndpointPaneTerminalView("west", "pane-west", "remote", 9, 80, 24, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID("pane-west"), false))

	reducer := NewEndpointStatusReducer(LiveDeps{})
	next, effects := reducer(root, EndpointRuntimeStatusMsg{Event: port.EndpointRuntimeEvent{
		EndpointID: "west",
		Status:     state.EndpointStatusOffline,
		ErrorKind:  state.EndpointErrorTransportClosed,
		Err:        errors.New("ssh transport closed: exit status 255"),
	}})

	if len(effects) != 0 || len(next.Shell.Toasts) != 0 {
		t.Fatalf("endpoint disconnect should not create effects or global toasts, effects=%#v toasts=%#v", effects, next.Shell.Toasts)
	}
	west, ok := next.Endpoints.Endpoint("west")
	if !ok || west.DisplayStatus() != state.EndpointStatusOffline || west.LastErrorKind != state.EndpointErrorTransportClosed || !strings.Contains(west.LastError, "ssh transport closed") {
		t.Fatalf("west endpoint should record offline transport error, got %#v ok=%v", west, ok)
	}
	if local, ok := next.Endpoints.Endpoint(state.DefaultEndpointID); !ok || local.DisplayStatus() != state.EndpointStatusConnected || local.LastError != "" {
		t.Fatalf("local endpoint should stay connected, got %#v ok=%v", local, ok)
	}
	if binding, ok := next.TerminalViews.PaneBinding("pane-west"); !ok || binding.Channel != 0 || binding.Attached || !strings.Contains(binding.LastError, "transport-closed") {
		t.Fatalf("west pane should show endpoint error and clear stale channel, binding=%#v ok=%v", binding, ok)
	}
	if binding, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID); !ok || binding.Channel != 3 || !binding.Attached || binding.LastError != "" {
		t.Fatalf("local pane should stay attached, binding=%#v ok=%v", binding, ok)
	}
	remoteSurface := next.Surface.SurfaceForTerminalRef(state.NewTerminalRef("west", "remote"))
	if remoteSurface.State != state.TerminalLiveError || !strings.Contains(remoteSurface.Err, "transport-closed") {
		t.Fatalf("west surface should show endpoint error, got %#v", remoteSurface)
	}
	if !next.Surface.TerminalRef().Equal(state.LocalTerminalRef("local")) || next.Surface.Err != "" || next.Surface.Lines[0] != "local" {
		t.Fatalf("active local surface should not be poisoned, got %#v", next.Surface)
	}
	if _, ok := next.Session.InputChannelForRef(state.NewTerminalRef("west", "remote")); ok {
		t.Fatalf("west input channel should be cleared, channels=%#v", next.Session.InputChannels)
	}
	if channel, ok := next.Session.InputChannelForRef(state.LocalTerminalRef("local")); !ok || channel != 3 {
		t.Fatalf("local input channel should survive, channel=%d ok=%v channels=%#v", channel, ok, next.Session.InputChannels)
	}
}

func TestEndpointRuntimeStatusStoresAndClearsManagedRouteProjection(t *testing.T) {
	root := state.Root{Endpoints: (state.EndpointStore{}).Upsert(state.EndpointItem{
		ID: "studio", Label: "Studio", Transport: state.EndpointTransportHubP2P,
		ConnectMode: state.EndpointConnectOnDemand, Enabled: true,
	})}
	reducer := NewEndpointStatusReducer(LiveDeps{})
	connected, _ := reducer(root, EndpointRuntimeStatusMsg{Event: port.EndpointRuntimeEvent{
		EndpointID: "studio", Status: state.EndpointStatusConnected, ObservedPath: "single_relay",
		RouteID: "cloud", Generation: 7, RouteSelectionReason: "first_ready",
	}})
	studio, ok := connected.Endpoints.Endpoint("studio")
	if !ok || studio.ActiveRouteID != "cloud" || studio.ConnectionGeneration != 7 || studio.ObservedPath != "single_relay" || studio.RouteSelectionReason != "first_ready" || studio.ConnectionPhase != "connected" || studio.DisplayStatus() != state.EndpointStatusConnected {
		t.Fatalf("connected managed path not stored: %#v ok=%v", studio, ok)
	}
	offline, _ := reducer(connected, EndpointRuntimeStatusMsg{Event: port.EndpointRuntimeEvent{
		EndpointID: "studio", RouteID: "cloud", Generation: 7, Status: state.EndpointStatusOffline, Err: errors.New("route closed"),
	}})
	studio, ok = offline.Endpoints.Endpoint("studio")
	if !ok || studio.ActiveRouteID != "" || studio.ConnectionGeneration != 7 || studio.ObservedPath != "" || studio.RouteSelectionReason != "" || studio.ConnectionPhase != "failed" || studio.DisplayStatus() != state.EndpointStatusOffline {
		t.Fatalf("offline managed path not cleared: %#v ok=%v", studio, ok)
	}
	stale, _ := reducer(offline, EndpointRuntimeStatusMsg{Event: port.EndpointRuntimeEvent{
		EndpointID: "studio", RouteID: "cloud", Generation: 6, Status: state.EndpointStatusConnected, ObservedPath: "direct", RouteSelectionReason: "current_winner",
	}})
	if got, _ := stale.Endpoints.Endpoint("studio"); got.DisplayStatus() != state.EndpointStatusOffline || got.ActiveRouteID != "" || got.ConnectionGeneration != 7 {
		t.Fatalf("stale generation revived endpoint: %#v", got)
	}
	zeroGeneration, _ := reducer(connected, EndpointRuntimeStatusMsg{Event: port.EndpointRuntimeEvent{
		EndpointID: "studio", Generation: 0, Status: state.EndpointStatusOffline, Err: errors.New("registry unavailable"),
	}})
	if got, _ := zeroGeneration.Endpoints.Endpoint("studio"); got.DisplayStatus() != state.EndpointStatusConnected || got.ActiveRouteID != "cloud" || got.ConnectionGeneration != 7 {
		t.Fatalf("unbound resolution failure replaced current winner: %#v", got)
	}
}

func TestEndpointRuntimeStatusKeepsActiveRemotePaneError(t *testing.T) {
	ref := state.NewTerminalRef("west", "remote")
	root := state.Root{
		Session: state.TerminalSessionStore{
			EndpointID:    ref.EndpointID,
			TerminalID:    ref.TerminalID,
			Channel:       9,
			Attached:      true,
			State:         state.TerminalLiveAttached,
			InputChannels: map[string]uint16{ref.Key(): 9},
			SurfaceID:     "surface",
			ViewID:        state.TerminalPaneViewID(state.DefaultPaneID),
			DesiredCols:   80,
			DesiredRows:   24,
		},
		Surface: state.TerminalSurfaceStore{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, Lines: []string{"remote prompt"}, State: state.TerminalLiveAttached, Ready: true},
		Endpoints: (state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: "west", Label: "US West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true, Status: state.EndpointStatusConnected}),
		Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, ref.TerminalID),
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView(ref.EndpointID, state.DefaultPaneID, ref.TerminalID, 9, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))

	reducer := NewEndpointStatusReducer(LiveDeps{})
	next, effects := reducer(root, EndpointRuntimeStatusMsg{Event: port.EndpointRuntimeEvent{
		EndpointID: "west",
		Status:     state.EndpointStatusOffline,
		ErrorKind:  state.EndpointErrorRemoteDaemon,
		Message:    "ssh transport closed: exit status 255: stdio-proxy connect core-v2 daemon socket: connection refused",
	}})

	if len(effects) != 0 || len(next.Shell.Toasts) != 0 {
		t.Fatalf("active endpoint disconnect should stay local, effects=%#v toasts=%#v", effects, next.Shell.Toasts)
	}
	binding, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || binding.TerminalID != ref.TerminalID || binding.Channel != 0 || binding.LastError == "" {
		t.Fatalf("active remote pane should keep terminal intent and error, binding=%#v ok=%v", binding, ok)
	}
	if next.Session.LastError == "" || next.Session.State != state.TerminalLiveError || next.Session.Attached {
		t.Fatalf("active session should keep local error projection, session=%#v", next.Session)
	}
	if next.Surface.Err == "" || next.Surface.State != state.TerminalLiveError || !next.Surface.TerminalRef().Equal(ref) {
		t.Fatalf("active surface should show remote daemon error, surface=%#v", next.Surface)
	}
	west, _ := next.Endpoints.Endpoint("west")
	if west.LastErrorKind != state.EndpointErrorRemoteDaemon {
		t.Fatalf("endpoint should classify daemon failure, got %#v", west)
	}
}

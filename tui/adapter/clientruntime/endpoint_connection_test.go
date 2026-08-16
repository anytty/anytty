package clientruntimeadapter

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

type endpointPlanSnapshotStub struct {
	registry endpoint.Registry
}

type endpointEnableLifecycleStub struct {
	endpointID state.EndpointID
	enabled    bool
	calls      int
	callEvents chan endpointEnableLifecycleCall
}

type endpointEnableLifecycleCall struct {
	endpointID state.EndpointID
	enabled    bool
}

func (stub *endpointEnableLifecycleStub) SetEndpointEnabled(_ context.Context, endpointID state.EndpointID, enabled bool) error {
	stub.endpointID = endpointID
	stub.enabled = enabled
	stub.calls++
	if stub.callEvents != nil {
		stub.callEvents <- endpointEnableLifecycleCall{endpointID: endpointID, enabled: enabled}
	}
	return nil
}

func (stub endpointPlanSnapshotStub) PlanSnapshot(_ context.Context, endpointID endpoint.EndpointID) (clientruntime.EndpointPlanSnapshot, error) {
	target := stub.registry.Endpoints[endpointID]
	return clientruntime.EndpointPlanSnapshot{
		Endpoint: target, ConfigKey: "test",
		Environment: clientruntime.RoutePlanEnvironment{
			SupportedRouteKinds:     []endpoint.RouteKind{endpoint.RouteLocalUnix, endpoint.RouteManagedWebRTC},
			AvailableCredentialRefs: []string{"credential:studio"},
		},
	}, nil
}

func TestEndpointConnectionControlLoadsAndAtomicallyAppliesPriorities(t *testing.T) {
	path := filepath.Join(t.TempDir(), "endpoints.yaml")
	registry := connectionControlRegistry()
	if err := endpoint.Save(path, registry); err != nil {
		t.Fatal(err)
	}
	control := EndpointConnectionControl{RegistryPath: path, Runtime: endpointPlanSnapshotStub{registry: registry}}
	store, err := control.LoadConnections(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	studio, ok := store.Endpoint("studio")
	if !ok || len(studio.Routes) != 2 || !studio.Routes[0].AvailabilityKnown {
		t.Fatalf("Go planner availability was not projected: %#v ok=%v", studio, ok)
	}
	zero, ten := 0, 10
	policy := state.EndpointConnectionPolicy{RoutePreference: endpoint.RoutePreferenceManagedCloud, CloudRelayMode: endpoint.RelayOnly, RelayTransport: endpoint.RelayTransportTCP}
	updated, err := control.ApplyConnectionSettings(context.Background(), "studio", policy, map[string]*int{"cloud": &ten, "local": &zero})
	if err != nil {
		t.Fatal(err)
	}
	studio, _ = updated.Endpoint("studio")
	if studio.Routes[0].Priority == nil || studio.Routes[1].Priority == nil {
		t.Fatalf("submitted projection lost priorities: %#v", studio.Routes)
	}
	persisted, err := endpoint.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := *persisted.Endpoints["studio"].Routes["local"].Priority; got != 0 {
		t.Fatalf("persisted local priority = %d", got)
	}
	if persisted.Endpoints["studio"].SelectionPolicy.RoutePreference != endpoint.RoutePreferenceManagedCloud || persisted.Endpoints["studio"].Routes["cloud"].RelayTransport != endpoint.RelayTransportTCP {
		t.Fatalf("persisted connection policy = %#v", persisted.Endpoints["studio"])
	}
	if _, err := control.ApplyConnectionSettings(context.Background(), state.EndpointID("studio"), policy, map[string]*int{"local": &zero}); err == nil {
		t.Fatal("partial priority transaction succeeded")
	}
	persisted, _ = endpoint.Load(path)
	if got := *persisted.Endpoints["studio"].Routes["cloud"].Priority; got != 10 {
		t.Fatalf("failed transaction changed persisted registry: %d", got)
	}
}

func TestEndpointConnectionControlPersistsEnabledAndNotifiesLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "endpoints.yaml")
	registry := connectionControlRegistry()
	local := endpoint.NewLocalEndpoint("local", "Local", "auto", endpoint.ConnectAuto)
	registry.Endpoints[local.ID] = local
	registry.Default = local.ID
	if err := endpoint.Save(path, registry); err != nil {
		t.Fatal(err)
	}
	lifecycle := &endpointEnableLifecycleStub{}
	control := EndpointConnectionControl{
		RegistryPath: path,
		Runtime:      endpointPlanSnapshotStub{registry: registry},
		Lifecycle:    lifecycle,
	}
	store, err := control.SetEndpointEnabled(context.Background(), "studio", false)
	if err != nil {
		t.Fatal(err)
	}
	studio, ok := store.Endpoint("studio")
	if !ok || studio.Enabled {
		t.Fatalf("disabled projection = %#v ok=%t", studio, ok)
	}
	persisted, err := endpoint.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Endpoints["studio"].Enabled || lifecycle.calls != 1 || lifecycle.endpointID != "studio" || lifecycle.enabled {
		t.Fatalf("persisted=%#v lifecycle=%#v", persisted.Endpoints["studio"], lifecycle)
	}
}

func TestEndpointConnectionControlWatchesExternalToggleAndRecoversAfterInvalidRegistry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "endpoints.yaml")
	initial := connectionControlRegistry()
	local := endpoint.NewLocalEndpoint("local", "Local", "auto", endpoint.ConnectAuto)
	initial.Endpoints[local.ID] = local
	initial.Default = local.ID
	if err := endpoint.Save(path, initial); err != nil {
		t.Fatal(err)
	}

	lifecycle := &endpointEnableLifecycleStub{callEvents: make(chan endpointEnableLifecycleCall, 4)}
	control := EndpointConnectionControl{
		RegistryPath: path, InitialRegistry: initial, Lifecycle: lifecycle, watchInterval: 5 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	updates, err := control.WatchConnections(ctx)
	if err != nil {
		t.Fatal(err)
	}

	disabled, err := endpoint.Update(path, false, func(registry endpoint.Registry) (endpoint.Registry, error) {
		return endpoint.SetEndpointEnabled(registry, "studio", false)
	})
	if err != nil {
		t.Fatal(err)
	}
	assertEndpointLifecycleCall(t, lifecycle.callEvents, "studio", false)
	update := awaitEndpointConnectionUpdate(t, updates)
	studio, ok := update.Store.Endpoint("studio")
	if update.Err != nil || !ok || studio.Enabled {
		t.Fatalf("external disable update = %#v studio=%#v ok=%t", update, studio, ok)
	}

	if err := os.WriteFile(path, []byte("not: [valid"), 0o600); err != nil {
		t.Fatal(err)
	}
	invalid := awaitEndpointConnectionUpdate(t, updates)
	if invalid.Err == nil || invalid.Store.HasItems() {
		t.Fatalf("invalid registry update = %#v", invalid)
	}

	reenabled, err := endpoint.SetEndpointEnabled(disabled, "studio", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := endpoint.Save(path, reenabled); err != nil {
		t.Fatal(err)
	}
	assertEndpointLifecycleCall(t, lifecycle.callEvents, "studio", true)
	update = awaitEndpointConnectionUpdate(t, updates)
	studio, ok = update.Store.Endpoint("studio")
	if update.Err != nil || !ok || !studio.Enabled {
		t.Fatalf("recovered enable update = %#v studio=%#v ok=%t", update, studio, ok)
	}

	cancel()
	select {
	case _, open := <-updates:
		if open {
			t.Fatal("registry watch emitted an update after cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("registry watch did not close after cancellation")
	}
}

func assertEndpointLifecycleCall(t *testing.T, calls <-chan endpointEnableLifecycleCall, endpointID state.EndpointID, enabled bool) {
	t.Helper()
	select {
	case call := <-calls:
		if call.endpointID != endpointID || call.enabled != enabled {
			t.Fatalf("lifecycle call = %#v, want endpoint=%q enabled=%t", call, endpointID, enabled)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for endpoint=%q enabled=%t lifecycle call", endpointID, enabled)
	}
}

func awaitEndpointConnectionUpdate(t *testing.T, updates <-chan port.EndpointConnectionUpdate) port.EndpointConnectionUpdate {
	t.Helper()
	select {
	case update, open := <-updates:
		if !open {
			t.Fatal("registry watch closed before the expected update")
		}
		return update
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for registry update")
		return port.EndpointConnectionUpdate{}
	}
}

func connectionControlRegistry() endpoint.Registry {
	identity := endpoint.DaemonIdentity{DeviceID: "device-studio", DeviceFingerprint: "SHA256:studio"}
	target := endpoint.Endpoint{
		ID: "studio", Label: "Studio", LabelSource: endpoint.SourceUser, DaemonIdentity: identity,
		ConnectMode: endpoint.ConnectOnDemand, Enabled: true,
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{
			"local": {ID: "local", Kind: endpoint.RouteLocalUnix, Enabled: true, Source: endpoint.SourceLocal, PolicySource: endpoint.SourceUser, Socket: "auto"},
			"cloud": {ID: "cloud", Kind: endpoint.RouteManagedWebRTC, Enabled: true, Source: endpoint.SourceCloud, PolicySource: endpoint.SourceUser, TargetDeviceID: identity.DeviceID, CredentialRef: "credential:studio", RelayMode: endpoint.RelayAuto},
		},
	}
	return endpoint.Registry{Version: endpoint.RegistryVersion, Default: target.ID, Endpoints: map[endpoint.EndpointID]endpoint.Endpoint{target.ID: target}}
}

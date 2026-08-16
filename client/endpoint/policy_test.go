package endpoint

import "testing"

func TestSetEndpointEnabledReassignsDisabledDefaultToLocalEndpoint(t *testing.T) {
	registry := Registry{
		Version: RegistryVersion,
		Default: "alpha",
		Endpoints: map[EndpointID]Endpoint{
			"alpha":  NewSSHEndpoint("alpha", "Alpha", "alpha", "ssh:alpha", "127.0.0.1:41120", "127.0.0.1:41121", ConnectOnDemand),
			"beta":   NewSSHEndpoint("beta", "Beta", "beta", "ssh:beta", "127.0.0.1:41120", "127.0.0.1:41121", ConnectOnDemand),
			"zlocal": NewLocalEndpoint("zlocal", "Local", "auto", ConnectAuto),
		},
	}
	next, err := SetEndpointEnabled(registry, "alpha", false)
	if err != nil {
		t.Fatal(err)
	}
	if next.Endpoints["alpha"].Enabled || next.Default != "zlocal" {
		t.Fatalf("disabled registry = %#v", next)
	}
	if !registry.Endpoints["alpha"].Enabled || registry.Default != "alpha" {
		t.Fatal("input registry was mutated")
	}

	next, err = SetEndpointEnabled(next, "alpha", true)
	if err != nil {
		t.Fatal(err)
	}
	if !next.Endpoints["alpha"].Enabled || next.Default != "zlocal" {
		t.Fatalf("enabled registry = %#v", next)
	}
}

func TestSetEndpointEnabledRejectsLastEnabledEndpoint(t *testing.T) {
	registry := DefaultRegistry()
	if _, err := SetEndpointEnabled(registry, registry.Default, false); err == nil {
		t.Fatal("last enabled endpoint was disabled")
	}
	if !registry.Endpoints[registry.Default].Enabled {
		t.Fatal("failed update mutated input registry")
	}
}

func TestSetConnectionPolicyUpdatesEveryManagedRouteAtomically(t *testing.T) {
	registry := Registry{Version: RegistryVersion, Default: "studio", Endpoints: map[EndpointID]Endpoint{"studio": plannerEndpoint()}}
	registry.Endpoints["studio"] = cloneEndpoint(registry.Endpoints["studio"])
	target := registry.Endpoints["studio"]
	target.Routes["cloud-backup"] = AccessRoute{ID: "cloud-backup", Kind: RouteManagedWebRTC, Enabled: true, TargetDeviceID: target.DaemonIdentity.DeviceID, CredentialRef: "credential:studio"}
	registry.Endpoints["studio"] = target

	next, err := SetConnectionPolicy(registry, "studio", ConnectionPolicy{
		RoutePreference: RoutePreferenceManagedCloud, CloudRelayMode: RelayOnly, RelayTransport: RelayTransportTCP,
	})
	if err != nil {
		t.Fatal(err)
	}
	updated := next.Endpoints["studio"]
	if updated.SelectionPolicy.RoutePreference != RoutePreferenceManagedCloud {
		t.Fatalf("route preference = %q", updated.SelectionPolicy.RoutePreference)
	}
	for _, routeID := range []RouteID{"cloud", "cloud-backup"} {
		route := updated.Routes[routeID]
		if route.RelayMode != RelayOnly || route.RelayTransport != RelayTransportTCP || route.PolicySource != SourceUser {
			t.Fatalf("managed route %q = %#v", routeID, route)
		}
	}
	if registry.Endpoints["studio"].SelectionPolicy.RoutePreference == RoutePreferenceManagedCloud {
		t.Fatal("input registry was mutated")
	}
}

func TestSetConnectionPolicyRejectsUnknownValues(t *testing.T) {
	for _, policy := range []ConnectionPolicy{
		{RoutePreference: "bad", CloudRelayMode: RelayAuto, RelayTransport: RelayTransportAuto},
		{RoutePreference: RoutePreferenceAuto, CloudRelayMode: "bad", RelayTransport: RelayTransportAuto},
		{RoutePreference: RoutePreferenceAuto, CloudRelayMode: RelayAuto, RelayTransport: "bad"},
	} {
		registry := Registry{Version: RegistryVersion, Default: "studio", Endpoints: map[EndpointID]Endpoint{"studio": plannerEndpoint()}}
		if _, err := SetConnectionPolicy(registry, "studio", policy); err == nil {
			t.Fatalf("invalid policy accepted: %#v", policy)
		}
	}
}

func TestSetAutomaticRoutePrioritiesIsAtomicAndPreservesFullRace(t *testing.T) {
	registry := Registry{Version: RegistryVersion, Default: "studio", Endpoints: map[EndpointID]Endpoint{"studio": plannerEndpoint()}}
	fullRace, err := SetAutomaticRoutePriorities(registry, "studio", map[RouteID]*int{"cloud": nil, "local": nil, "ssh": nil})
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range fullRace.Endpoints["studio"].RouteList() {
		if route.Enabled && !route.ManualOnly && (route.Priority != nil || route.PolicySource != SourceUser) {
			t.Fatalf("full race route was not normalized through user policy: %#v", route)
		}
	}
	zero, ten := 0, 10
	grouped, err := SetAutomaticRoutePriorities(fullRace, "studio", map[RouteID]*int{"cloud": &ten, "local": &zero, "ssh": &ten})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := (RouteSelectionPlanner{}).Plan(RouteSelectionRequest{
		Endpoint: grouped.Endpoints["studio"], Intent: ConnectIntent{Kind: "interactive"}, Generation: 1,
		SupportedRouteKinds:     []RouteKind{RouteLocalUnix, RouteSSHWebRTCTCP, RouteManagedWebRTC},
		AvailableCredentialRefs: []string{"credential:ssh-studio", "ssh:studio", "credential:studio"},
	})
	if err != nil {
		t.Fatal(err)
	}
	groups := plan.Groups()
	if len(groups) != 2 || groups[0].Priority() == nil || *groups[0].Priority() != 0 || len(groups[1].Attempts()) != 2 {
		t.Fatalf("priority groups do not preserve smaller-first/equal-concurrent semantics: %#v", groups)
	}
}

func TestSetAutomaticRoutePrioritiesRejectsPartialOrUnknownRoute(t *testing.T) {
	registry := Registry{Version: RegistryVersion, Default: "studio", Endpoints: map[EndpointID]Endpoint{"studio": plannerEndpoint()}}
	one := 1
	for name, priorities := range map[string]map[RouteID]*int{
		"partial": {"local": &one},
		"unknown": {"cloud": &one, "local": &one, "missing": &one},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := SetAutomaticRoutePriorities(registry, "studio", priorities); err == nil {
				t.Fatal("invalid priority transaction succeeded")
			}
			if registry.Endpoints["studio"].Routes["local"].Priority != nil {
				t.Fatal("failed transaction mutated input registry")
			}
		})
	}
}

func TestEvaluateRouteAvailabilityUsesSharedPlannerPrerequisites(t *testing.T) {
	target := plannerEndpoint()
	planning := cloneEndpoint(target)
	cloud := planning.Routes["cloud"]
	cloud.Enabled = false
	planning.Routes["cloud"] = cloud
	availability, err := EvaluateRouteAvailability(RouteAvailabilityRequest{
		Endpoint: target, PlanningEndpoint: planning,
		SupportedRouteKinds:     []RouteKind{RouteLocalUnix, RouteSSHWebRTCTCP, RouteManagedWebRTC},
		AvailableCredentialRefs: []string{"credential:ssh-studio", "ssh:studio"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := map[RouteID]RouteAvailability{}
	for _, item := range availability {
		got[item.RouteID] = item
	}
	if !got["local"].Available || !got["ssh"].Available {
		t.Fatalf("eligible routes unavailable: %#v", got)
	}
	if got["cloud"].Reason != RouteAvailabilityCloudUnavailable {
		t.Fatalf("managed eligibility loss not projected: %#v", got["cloud"])
	}
}

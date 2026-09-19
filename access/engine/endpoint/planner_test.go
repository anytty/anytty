package endpoint

import (
	"testing"
	"time"
)

func TestRouteSelectionPlannerFullRaceAndImmutableOutput(t *testing.T) {
	target := plannerEndpoint()
	plan, err := (RouteSelectionPlanner{}).Plan(RouteSelectionRequest{
		Endpoint: target, Intent: ConnectIntent{Kind: "interactive"}, Generation: 7,
		SupportedRouteKinds:     []RouteKind{RouteLocalUnix, RouteSSHWebRTCTCP, RouteManagedWebRTC},
		AvailableCredentialRefs: []string{"credential:ssh-studio", "ssh:studio", "credential:studio"},
	})
	if err != nil {
		t.Fatal(err)
	}
	groups := plan.Groups()
	if len(groups) != 1 || groups[0].StartDelay() != 0 || groups[0].Priority() != nil {
		t.Fatalf("full-race groups = %#v", groups)
	}
	if diagnostics := plan.Diagnostics(); len(diagnostics) != 0 {
		t.Fatalf("full-race diagnostics = %#v", diagnostics)
	}
	attempts := groups[0].Attempts()
	if len(attempts) != 3 || attempts[0].Route.ID != "cloud" || attempts[1].Route.ID != "local" || attempts[2].Route.ID != "ssh" {
		t.Fatalf("full-race attempts = %#v", attempts)
	}
	if attempts[1].Generation != 7 || attempts[1].AttemptID != "studio:7:local" {
		t.Fatalf("attempt identity = %#v", attempts[1])
	}
	attempts[1].Route.Socket = "changed"
	attempts[1].Intent.RequiredScopes = append(attempts[1].Intent.RequiredScopes, "changed")
	if current := plan.Groups()[0].Attempts()[1]; current.Route.Socket != "auto" || len(current.Intent.RequiredScopes) != 0 {
		t.Fatalf("plan output was mutated through getter: %#v", current)
	}
	local := target.Routes["local"]
	local.Socket = "registry-change"
	target.Routes["local"] = local
	if current := plan.Groups()[0].Attempts()[1]; current.Route.Socket != "auto" {
		t.Fatalf("plan output followed registry mutation: %#v", current)
	}
}

func TestRouteSelectionPlannerGroupsPriorityWithCumulativeHedge(t *testing.T) {
	target := plannerEndpoint()
	priority10, priority20 := 10, 20
	local := target.Routes["local"]
	local.Priority = &priority10
	target.Routes["local"] = local
	ssh := target.Routes["ssh"]
	ssh.Priority = &priority20
	target.Routes["ssh"] = ssh
	cloud := target.Routes["cloud"]
	cloud.Priority = &priority20
	target.Routes["cloud"] = cloud
	target.SelectionPolicy = SelectionPolicy{HedgeDelay: 250 * time.Millisecond, HedgeDelayConfigured: true}

	plan, err := (RouteSelectionPlanner{}).Plan(RouteSelectionRequest{
		Endpoint: target, Intent: ConnectIntent{Kind: "background"}, Generation: 3,
		SupportedRouteKinds:     []RouteKind{RouteLocalUnix, RouteSSHWebRTCTCP, RouteManagedWebRTC},
		AvailableCredentialRefs: []string{"credential:ssh-studio", "ssh:studio", "credential:studio"},
	})
	if err != nil {
		t.Fatal(err)
	}
	groups := plan.Groups()
	if len(groups) != 2 || *groups[0].Priority() != 10 || groups[0].StartDelay() != 0 || *groups[1].Priority() != 20 || groups[1].StartDelay() != 250*time.Millisecond {
		t.Fatalf("priority groups = %#v", groups)
	}
	if attempts := groups[1].Attempts(); len(attempts) != 2 || attempts[0].Route.ID != "cloud" || attempts[1].Route.ID != "ssh" {
		t.Fatalf("priority group attempts = %#v", attempts)
	}
}

func TestRouteSelectionPlannerExplicitManualRouteAndFailures(t *testing.T) {
	target := plannerEndpoint()
	ssh := target.Routes["ssh"]
	ssh.ManualOnly = true
	target.Routes["ssh"] = ssh
	planner := RouteSelectionPlanner{}

	plan, err := planner.Plan(RouteSelectionRequest{
		Endpoint: target, Intent: ConnectIntent{Kind: "probe"}, RouteOverride: "ssh", Generation: 2,
		SupportedRouteKinds: []RouteKind{RouteSSHWebRTCTCP}, AvailableCredentialRefs: []string{"credential:ssh-studio", "ssh:studio"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts := plan.Groups()[0].Attempts(); len(attempts) != 1 || attempts[0].Route.ID != "ssh" {
		t.Fatalf("override attempts = %#v", attempts)
	}

	request := RouteSelectionRequest{
		Endpoint: target, Intent: ConnectIntent{Kind: "interactive"}, RouteOverride: "ssh", Generation: 2,
		SupportedRouteKinds: []RouteKind{RouteSSHWebRTCTCP},
	}
	if _, err := planner.Plan(request); !IsCode(err, ErrorCredentialRequired) {
		t.Fatalf("missing credential error = %v", err)
	}
	request.AvailableCredentialRefs = []string{"credential:ssh-studio", "ssh:studio"}
	request.SupportedRouteKinds = []RouteKind{RouteLocalUnix}
	if _, err := planner.Plan(request); !IsCode(err, ErrorRouteUnavailable) {
		t.Fatalf("unsupported platform error = %v", err)
	}
	request.SupportedRouteKinds = []RouteKind{RouteSSHWebRTCTCP}
	request.RouteOverride = "missing"
	if _, err := planner.Plan(request); !IsCode(err, ErrorRouteUnavailable) {
		t.Fatalf("missing override error = %v", err)
	}
}

func TestRouteSelectionPlannerAllowsExplicitDirectRoute(t *testing.T) {
	target := plannerEndpoint()
	target.Routes["direct"] = AccessRoute{
		ID: "direct", Kind: RouteDirectWebRTCTCP, Enabled: true, Source: SourceManual, PolicySource: SourceUser,
		CredentialRef: "credential:studio", SignalingAddresses: []string{"studio:41120"}, ICETCPAddresses: []string{"studio:41121"},
	}
	plan, err := (RouteSelectionPlanner{}).Plan(RouteSelectionRequest{
		Endpoint: target, Intent: ConnectIntent{Kind: "interactive"}, RouteOverride: "direct", Generation: 2,
		SupportedRouteKinds: []RouteKind{RouteDirectWebRTCTCP}, AvailableCredentialRefs: []string{"credential:studio"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts := plan.Groups()[0].Attempts(); len(attempts) != 1 || attempts[0].Route.ID != "direct" {
		t.Fatalf("Direct override attempts = %#v", attempts)
	}
}

func TestRouteSelectionPlannerKeepsSingleManagedRouteOutOfCommonRace(t *testing.T) {
	target := plannerEndpoint()
	for _, routeID := range []RouteID{"local", "ssh"} {
		route := target.Routes[routeID]
		route.Enabled = false
		target.Routes[routeID] = route
	}
	plan, err := (RouteSelectionPlanner{}).Plan(RouteSelectionRequest{
		Endpoint: target, Intent: ConnectIntent{Kind: "interactive"}, Generation: 4,
		SupportedRouteKinds: []RouteKind{RouteManagedWebRTC}, AvailableCredentialRefs: []string{"credential:studio"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts := plan.Groups()[0].Attempts(); len(attempts) != 1 || attempts[0].Route.Kind != RouteManagedWebRTC {
		t.Fatalf("single managed plan = %#v", attempts)
	}
}

func TestRouteSelectionPlannerEnforcesRoutePreferenceWithoutFallback(t *testing.T) {
	target := plannerEndpoint()
	target.SelectionPolicy.RoutePreference = RoutePreferenceSSH
	plan, err := (RouteSelectionPlanner{}).Plan(RouteSelectionRequest{
		Endpoint: target, Intent: ConnectIntent{Kind: "interactive"}, Generation: 5,
		SupportedRouteKinds:     []RouteKind{RouteDirectWebRTCTCP, RouteSSHWebRTCTCP, RouteManagedWebRTC},
		AvailableCredentialRefs: []string{"credential:ssh-studio", "ssh:studio", "credential:studio"},
	})
	if err != nil {
		t.Fatal(err)
	}
	attempts := plan.Groups()[0].Attempts()
	if len(attempts) != 1 || attempts[0].Route.Kind != RouteSSHWebRTCTCP {
		t.Fatalf("forced SSH attempts = %#v", attempts)
	}

	target.SelectionPolicy.RoutePreference = RoutePreferenceDirect
	if _, err := (RouteSelectionPlanner{}).Plan(RouteSelectionRequest{
		Endpoint: target, Intent: ConnectIntent{Kind: "interactive"}, Generation: 6,
		SupportedRouteKinds:     []RouteKind{RouteSSHWebRTCTCP, RouteManagedWebRTC},
		AvailableCredentialRefs: []string{"credential:ssh-studio", "ssh:studio", "credential:studio"},
	}); !IsCode(err, ErrorRouteUnavailable) {
		t.Fatalf("unavailable forced Direct error = %v", err)
	}
}

func plannerEndpoint() Endpoint {
	identity := DaemonIdentity{DeviceID: "device-studio", DeviceFingerprint: "SHA256:studio"}
	return Endpoint{
		ID: "studio", Label: "Studio", LabelSource: SourceUser, DaemonIdentity: identity, ConnectMode: ConnectOnDemand, Enabled: true,
		Routes: map[RouteID]AccessRoute{
			"local": {ID: "local", Kind: RouteLocalUnix, Enabled: true, Source: SourceLocal, PolicySource: SourceUser, Socket: "auto"},
			"ssh":   {ID: "ssh", Kind: RouteSSHWebRTCTCP, Enabled: true, Source: SourceManual, PolicySource: SourceUser, Host: "studio", RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121", CredentialRef: "credential:ssh-studio", SSHCredentialRef: "ssh:studio"},
			"cloud": {ID: "cloud", Kind: RouteManagedWebRTC, Enabled: true, Source: SourceCloud, PolicySource: SourceUser, TargetDeviceID: identity.DeviceID, RelayMode: RelayAuto, CredentialRef: "credential:studio"},
		},
	}
}

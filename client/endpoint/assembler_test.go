package endpoint

import (
	"reflect"
	"testing"
	"time"
)

func TestAssemblerMergesCloudBootstrapAndManualByFingerprint(t *testing.T) {
	identity := DaemonIdentity{DeviceID: "device-studio", DeviceFingerprint: "SHA256:studio"}
	cloud := EndpointCandidate{Source: SourceCloud, Identity: identity, SuggestedLabel: "Cloud Studio", Routes: []AccessRoute{
		{ID: "cloud", Kind: RouteManagedWebRTC, Enabled: true, Source: SourceCloud, TargetDeviceID: identity.DeviceID, RelayMode: RelayAuto},
	}}
	bootstrap := EndpointCandidate{Source: SourceBootstrap, Identity: identity, SuggestedLabel: "Studio", Routes: []AccessRoute{
		{ID: "lan", Kind: RouteDirectWebRTCTCP, Enabled: true, Source: SourceBootstrap, SignalingAddresses: []string{"studio.local:41120"}, ICETCPAddresses: []string{"studio.local:41120"}},
	}}
	manual := EndpointCandidate{Source: SourceManual, Identity: identity, SuggestedLabel: "Studio SSH", Routes: []AccessRoute{
		{ID: "ssh", Kind: RouteSSHWebRTCTCP, Enabled: true, Source: SourceManual, Host: "studio-host", RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121"},
	}}
	left := assembleSequence(t, cloud, bootstrap, manual)
	right := assembleSequence(t, manual, bootstrap, cloud)
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("assembler is not commutative\nleft=%#v\nright=%#v", left, right)
	}
	if len(left.Endpoints) != 2 {
		t.Fatalf("default local plus one daemon expected, got %#v", left.Endpoints)
	}
	var studio Endpoint
	for _, endpoint := range left.Endpoints {
		if endpoint.DaemonIdentity == identity {
			studio = endpoint
		}
	}
	if len(studio.Routes) != 3 || studio.ID == "" {
		t.Fatalf("expected one three-route endpoint, got %#v", studio)
	}
	for _, routeID := range []RouteID{"cloud", "lan"} {
		if route := studio.Routes[routeID]; route.PolicySource != SourceLocal {
			t.Fatalf("external route %q claimed client policy ownership: %#v", routeID, route)
		}
	}
}

func TestAssemblerPublishesFirstImportedEndpointWithoutInventingLocalEndpoint(t *testing.T) {
	identity := DaemonIdentity{DeviceID: "device-studio", DeviceFingerprint: "SHA256:studio"}
	result, err := AssembleEndpoints(EndpointAssemblerInput{Registry: Registry{}, Candidates: []EndpointCandidate{{
		Source: SourceManual, Identity: identity, Routes: []AccessRoute{{ID: "ssh", Kind: RouteSSHWebRTCTCP, Enabled: true, Host: "studio", RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121"}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Registry.Endpoints) != 1 || result.Registry.Default != result.ResolvedEndpointIDs[0] {
		t.Fatalf("first import did not become the sole default endpoint: %#v", result)
	}
}

func TestAssemblerRejectsIdentityAndRouteConflicts(t *testing.T) {
	identity := DaemonIdentity{DeviceID: "device-studio", DeviceFingerprint: "SHA256:studio"}
	base, err := AssembleEndpoints(EndpointAssemblerInput{Registry: DefaultRegistry(), Candidates: []EndpointCandidate{{
		Source: SourceBootstrap, Identity: identity, Routes: []AccessRoute{{ID: "lan", Kind: RouteDirectWebRTCTCP, Enabled: true, SignalingAddresses: []string{"studio:41120"}, ICETCPAddresses: []string{"studio:41120"}}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = AssembleEndpoints(EndpointAssemblerInput{Registry: base.Registry, Candidates: []EndpointCandidate{{
		Source: SourceCloud, Identity: DaemonIdentity{DeviceID: identity.DeviceID, DeviceFingerprint: "SHA256:other"},
		Routes: []AccessRoute{{ID: "cloud", Kind: RouteManagedWebRTC, Enabled: true, TargetDeviceID: identity.DeviceID, RelayMode: RelayAuto}},
	}}})
	if !IsCode(err, ErrorIdentityConflict) {
		t.Fatalf("identity conflict error=%v", err)
	}
	_, err = AssembleEndpoints(EndpointAssemblerInput{Registry: base.Registry, Candidates: []EndpointCandidate{{
		Source: SourceManual, Identity: identity, Routes: []AccessRoute{{ID: "lan", Kind: RouteSSHWebRTCTCP, Enabled: true, Host: "studio", RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121"}},
	}}})
	if !IsCode(err, ErrorRouteConflict) {
		t.Fatalf("route conflict error=%v", err)
	}
}

func TestAssemblerBindsUnverifiedSSHEndpointOnlyWithConfirmedIdentity(t *testing.T) {
	identity := DaemonIdentity{DeviceID: "device-studio", DeviceFingerprint: "SHA256:studio"}
	ssh := NewSSHEndpoint("studio", "Studio SSH", "studio-host", "ssh:studio", "127.0.0.1:41120", "127.0.0.1:41121", ConnectOnDemand)
	registry, err := (Registry{Version: RegistryVersion, Default: "studio", Endpoints: map[EndpointID]Endpoint{"studio": ssh}}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	candidate := EndpointCandidate{Source: SourceBootstrap, Identity: identity, Routes: []AccessRoute{
		{ID: "lan", Kind: RouteDirectWebRTCTCP, Enabled: true, SignalingAddresses: []string{"studio.local:41120"}, ICETCPAddresses: []string{"studio.local:41120"}},
	}}
	result, err := AssembleEndpoints(EndpointAssemblerInput{
		Registry: registry, Candidates: []EndpointCandidate{candidate},
		ConfirmedIdentityBindings: []ConfirmedIdentityBinding{{EndpointID: "studio", Identity: identity}},
	})
	if err != nil {
		t.Fatal(err)
	}
	endpoint := result.Registry.Endpoints["studio"]
	if len(result.Registry.Endpoints) != 1 || endpoint.DaemonIdentity != identity || len(endpoint.Routes) != 2 || result.ResolvedEndpointIDs[0] != "studio" {
		t.Fatalf("confirmed binding did not preserve one endpoint: %#v", result)
	}
	_, err = AssembleEndpoints(EndpointAssemblerInput{
		Registry:                  registry,
		ConfirmedIdentityBindings: []ConfirmedIdentityBinding{{EndpointID: "studio", Identity: identity}},
	})
	if !IsCode(err, ErrorConfig) {
		t.Fatalf("binding without a verified candidate error=%v", err)
	}
}

func TestAssemblerPreservesUserPolicyUnlessConfirmedShare(t *testing.T) {
	identity := DaemonIdentity{DeviceID: "device-studio", DeviceFingerprint: "SHA256:studio"}
	priority := 10
	base, err := AssembleEndpoints(EndpointAssemblerInput{Registry: DefaultRegistry(), Candidates: []EndpointCandidate{{
		Source: SourceBootstrap, Identity: identity, SuggestedLabel: "User Studio", Routes: []AccessRoute{{
			ID: "ssh", Kind: RouteSSHWebRTCTCP, Enabled: true, Host: "studio", RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121",
		}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	endpoint := endpointByIdentity(t, base.Registry, identity)
	route := endpoint.Routes["ssh"]
	route.Enabled, route.ManualOnly, route.Priority, route.PolicySource = false, true, &priority, SourceUser
	endpoint.Routes["ssh"] = route
	endpoint.Label, endpoint.LabelSource = "User Studio", SourceUser
	base.Registry.Endpoints[endpoint.ID] = endpoint
	base.Registry, err = base.Registry.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	result, err := AssembleEndpoints(EndpointAssemblerInput{Registry: base.Registry, Candidates: []EndpointCandidate{{
		Source: SourceBootstrap, Identity: identity, SuggestedLabel: "Bootstrap Label", Routes: []AccessRoute{{
			ID: "ssh", Kind: RouteSSHWebRTCTCP, Enabled: true, Source: SourceBootstrap, Host: "new-studio", RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121",
		}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	endpoint = endpointByIdentity(t, result.Registry, identity)
	route = endpoint.Routes["ssh"]
	if endpoint.Label != "User Studio" || route.Enabled || !route.ManualOnly || route.Priority == nil || *route.Priority != 10 || route.Host != "new-studio" {
		t.Fatalf("bootstrap overwrote user policy or failed to refresh config: endpoint=%#v route=%#v", endpoint, route)
	}
	unconfirmedShare, err := AssembleEndpoints(EndpointAssemblerInput{Registry: result.Registry, Candidates: []EndpointCandidate{{
		Source: SourceShare, Identity: identity, SuggestedLabel: "Unconfirmed Share Label",
		Routes: []AccessRoute{{ID: "ssh", Kind: RouteSSHWebRTCTCP, Enabled: true, Source: SourceShare, PolicySource: SourceShare, Host: "shared-config", RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121"}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	endpoint = endpointByIdentity(t, unconfirmedShare.Registry, identity)
	if endpoint.Label != "User Studio" || endpoint.Routes["ssh"].Host != "shared-config" {
		t.Fatalf("unconfirmed share changed label policy or failed to refresh route config: %#v", endpoint)
	}
	sharePriority := 5
	sharePolicy := SelectionPolicy{HedgeDelay: 450 * time.Millisecond, HedgeDelayConfigured: true}
	shared, err := AssembleEndpoints(EndpointAssemblerInput{Registry: unconfirmedShare.Registry, Candidates: []EndpointCandidate{{
		Source: SourceShare, Identity: identity, ConnectMode: ConnectManual, SelectionPolicy: &sharePolicy, ApplyClientPolicy: true,
		Routes: []AccessRoute{{
			ID: "ssh", Kind: RouteSSHWebRTCTCP, Enabled: true, Priority: &sharePriority, Source: SourceShare, PolicySource: SourceShare, Host: "shared-studio", RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121",
		}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	endpoint = endpointByIdentity(t, shared.Registry, identity)
	route = endpoint.Routes["ssh"]
	if endpoint.ConnectMode != ConnectManual || endpoint.SelectionPolicy != sharePolicy || !route.Enabled || route.ManualOnly || route.Priority == nil || *route.Priority != sharePriority {
		t.Fatalf("confirmed share did not apply client policy: endpoint=%#v route=%#v", endpoint, route)
	}
	withExternalRoute, err := AssembleEndpoints(EndpointAssemblerInput{Registry: shared.Registry, Candidates: []EndpointCandidate{{
		Source: SourceBootstrap, Identity: identity,
		Routes: []AccessRoute{{ID: "lan", Kind: RouteDirectWebRTCTCP, Enabled: true, SignalingAddresses: []string{"studio:41120"}, ICETCPAddresses: []string{"studio:41120"}}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	lanRoute := endpointByIdentity(t, withExternalRoute.Registry, identity).Routes["lan"]
	if !lanRoute.Enabled || !lanRoute.ManualOnly || lanRoute.Priority != nil || lanRoute.PolicySource != SourceLocal {
		t.Fatalf("external route should wait for explicit priority policy: %#v", lanRoute)
	}
	if _, err := AssembleEndpoints(EndpointAssemblerInput{Registry: result.Registry, Candidates: []EndpointCandidate{{
		Source: SourceBootstrap, Identity: identity, ConnectMode: ConnectAuto,
		Routes: []AccessRoute{{ID: "lan", Kind: RouteDirectWebRTCTCP, Enabled: true, SignalingAddresses: []string{"studio:41120"}, ICETCPAddresses: []string{"studio:41120"}}},
	}}}); !IsCode(err, ErrorConfig) {
		t.Fatalf("bootstrap client policy error=%v", err)
	}
}

func TestAssemblerReturnsDeterministicCredentialDescriptors(t *testing.T) {
	identity := DaemonIdentity{DeviceID: "device-studio", DeviceFingerprint: "SHA256:studio"}
	result, err := AssembleEndpoints(EndpointAssemblerInput{Registry: DefaultRegistry(), Candidates: []EndpointCandidate{{
		Source: SourceShare, Identity: identity,
		Routes: []AccessRoute{{ID: "ssh", Kind: RouteSSHWebRTCTCP, Enabled: true, Host: "studio", RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121"}},
		CredentialDescriptors: []CredentialDescriptor{
			{DescriptorID: "ssh-password", Kind: CredentialSSHPassword, Exportable: true},
			{DescriptorID: "ssh-key", Kind: CredentialSSHPrivateKey},
			{DescriptorID: "ssh-key", Kind: CredentialSSHPrivateKey},
		},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	want := []CredentialDescriptor{
		{DescriptorID: "ssh-key", Kind: CredentialSSHPrivateKey},
		{DescriptorID: "ssh-password", Kind: CredentialSSHPassword, Exportable: true},
	}
	if !reflect.DeepEqual(result.CredentialDescriptors, want) {
		t.Fatalf("credential descriptors=%#v want=%#v", result.CredentialDescriptors, want)
	}
	_, err = AssembleEndpoints(EndpointAssemblerInput{Registry: DefaultRegistry(), Candidates: []EndpointCandidate{{
		Source: SourceShare, Identity: identity,
		Routes: []AccessRoute{{ID: "ssh", Kind: RouteSSHWebRTCTCP, Enabled: true, Host: "studio", RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121"}},
		CredentialDescriptors: []CredentialDescriptor{
			{DescriptorID: "credential", Kind: CredentialSSHPrivateKey},
			{DescriptorID: "credential", Kind: CredentialSSHPassword},
		},
	}}})
	if !IsCode(err, ErrorConfig) {
		t.Fatalf("inconsistent credential descriptor error=%v", err)
	}
}

func assembleSequence(t *testing.T, candidates ...EndpointCandidate) Registry {
	t.Helper()
	registry := DefaultRegistry()
	for _, candidate := range candidates {
		result, err := AssembleEndpoints(EndpointAssemblerInput{Registry: registry, Candidates: []EndpointCandidate{candidate}})
		if err != nil {
			t.Fatal(err)
		}
		registry = result.Registry
	}
	return registry
}

func endpointByIdentity(t *testing.T, registry Registry, identity DaemonIdentity) Endpoint {
	t.Helper()
	for _, endpoint := range registry.Endpoints {
		if endpoint.DaemonIdentity == identity {
			return endpoint
		}
	}
	t.Fatalf("identity %#v not found in %#v", identity, registry)
	return Endpoint{}
}

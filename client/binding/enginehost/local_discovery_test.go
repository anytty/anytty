package enginehost

import (
	"context"
	"testing"
	"time"

	"github.com/anytty/anytty/client/binding"
	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/bindingpb"
	"github.com/anytty/anytty/proto/wire"
	"google.golang.org/protobuf/proto"
)

func TestApplyPlatformLocalDiscoveryUpdatesOnlyPlanningTarget(t *testing.T) {
	broker := binding.NewPlatformBroker()
	defer broker.Close()
	now := time.Date(2026, 8, 13, 8, 0, 0, 0, time.UTC)
	go answerLocalDiscoveryRequest(t, broker, &bindingpb.LocalDiscoveryLookupResult{Candidates: []*bindingpb.LocalDiscoveryCandidate{
		{Address: "192.168.1.8", Port: 41120, ProtocolVersion: uint32(wire.Version), ExpiresAtUnixNano: now.Add(time.Minute).UnixNano(), NetworkHandle: 42},
	}})
	original := endpoint.Endpoint{
		ID: "studio", Label: "Studio", Enabled: true,
		DaemonIdentity: endpoint.DaemonIdentity{DeviceID: "device-1", DeviceFingerprint: "ed25519-sha256:test"},
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{"cloud": {
			ID: "cloud", Kind: endpoint.RouteManagedWebRTC, Enabled: true, CredentialRef: "grant:1", Source: endpoint.SourceCloud, PolicySource: endpoint.SourceCloud,
			TargetDeviceID: "device-1", AccountProfileRef: "default", RelayMode: endpoint.RelayAuto,
		}},
	}
	planning := applyPlatformLocalDiscovery(context.Background(), original, Options{Broker: broker, DirectPeers: fakeCloudPairingPeerFactory{}, Now: func() time.Time { return now }, EnableLocalDiscovery: true})
	route, ok := planning.Routes[localDiscoveryRouteID]
	if !ok || route.CredentialRef != "grant:1" || route.NetworkHandle != 42 || len(route.SignalingAddresses) != 1 || route.SignalingAddresses[0] != "192.168.1.8:41120" {
		t.Fatalf("LAN route = %#v", route)
	}
	if _, persisted := original.Routes[localDiscoveryRouteID]; persisted {
		t.Fatal("ephemeral LAN route mutated persisted endpoint")
	}
}

func TestApplyPlatformLocalDiscoveryKeepsSourceNetworksSeparate(t *testing.T) {
	broker := binding.NewPlatformBroker()
	defer broker.Close()
	now := time.Now().UTC()
	go answerLocalDiscoveryRequest(t, broker, &bindingpb.LocalDiscoveryLookupResult{Candidates: []*bindingpb.LocalDiscoveryCandidate{
		{Address: "192.168.1.8", Port: 41120, ProtocolVersion: uint32(wire.Version), ExpiresAtUnixNano: now.Add(time.Minute).UnixNano(), NetworkHandle: 42},
		{Address: "192.168.50.8", Port: 41120, ProtocolVersion: uint32(wire.Version), ExpiresAtUnixNano: now.Add(time.Minute).UnixNano(), NetworkHandle: 77},
	}})
	target := endpoint.Endpoint{
		ID: "studio", DaemonIdentity: endpoint.DaemonIdentity{DeviceID: "device-1", DeviceFingerprint: "ed25519-sha256:test"},
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{"cloud": {
			ID: "cloud", Kind: endpoint.RouteManagedWebRTC, Enabled: true, CredentialRef: "grant:1", Source: endpoint.SourceCloud, PolicySource: endpoint.SourceCloud,
			TargetDeviceID: "device-1", AccountProfileRef: "default", RelayMode: endpoint.RelayAuto,
		}},
	}
	planning := applyPlatformLocalDiscovery(context.Background(), target, Options{Broker: broker, DirectPeers: fakeCloudPairingPeerFactory{}, Now: func() time.Time { return now }, EnableLocalDiscovery: true})
	first, second := planning.Routes["lan-discovery"], planning.Routes["lan-discovery-2"]
	if first.NetworkHandle != 42 || second.NetworkHandle != 77 || first.SignalingAddresses[0] != "192.168.1.8:41120" || second.SignalingAddresses[0] != "192.168.50.8:41120" {
		t.Fatalf("LAN network groups = %#v / %#v", first, second)
	}
}

func TestApplyPlatformLocalDiscoveryRejectsExpiredOrWrongProtocol(t *testing.T) {
	broker := binding.NewPlatformBroker()
	defer broker.Close()
	now := time.Now().UTC()
	go answerLocalDiscoveryRequest(t, broker, &bindingpb.LocalDiscoveryLookupResult{Candidates: []*bindingpb.LocalDiscoveryCandidate{
		{Address: "192.168.1.9", Port: 41120, ProtocolVersion: uint32(wire.Version + 1), ExpiresAtUnixNano: now.Add(time.Minute).UnixNano()},
		{Address: "192.168.1.10", Port: 41120, ProtocolVersion: uint32(wire.Version), ExpiresAtUnixNano: now.Add(-time.Second).UnixNano()},
	}})
	target := endpoint.Endpoint{ID: "studio", DaemonIdentity: endpoint.DaemonIdentity{DeviceID: "device-1", DeviceFingerprint: "ed25519-sha256:test"}, Routes: map[endpoint.RouteID]endpoint.AccessRoute{}}
	planning := applyPlatformLocalDiscovery(context.Background(), target, Options{Broker: broker, DirectPeers: fakeCloudPairingPeerFactory{}, Now: func() time.Time { return now }, EnableLocalDiscovery: true})
	if len(planning.Routes) != 0 {
		t.Fatalf("invalid discovery routes = %#v", planning.Routes)
	}
}

func TestApplyPlatformLocalDiscoveryPrependsCandidateToPairingDirectRoute(t *testing.T) {
	broker := binding.NewPlatformBroker()
	defer broker.Close()
	now := time.Now().UTC()
	go answerLocalDiscoveryRequest(t, broker, &bindingpb.LocalDiscoveryLookupResult{Candidates: []*bindingpb.LocalDiscoveryCandidate{
		{Address: "192.168.1.12", Port: 41120, ProtocolVersion: uint32(wire.Version), ExpiresAtUnixNano: now.Add(time.Minute).UnixNano()},
	}})
	target := endpoint.Endpoint{
		ID: "studio", DaemonIdentity: endpoint.DaemonIdentity{DeviceID: "device-1", DeviceFingerprint: "ed25519-sha256:test"},
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{"direct": {
			ID: "direct", Kind: endpoint.RouteDirectWebRTCTCP, Enabled: true, CredentialRef: "grant:1", Source: endpoint.SourceBootstrap, PolicySource: endpoint.SourceBootstrap,
			SignalingAddresses: []string{"192.168.1.8:41120"}, ICETCPAddresses: []string{"192.168.1.8:41120"},
		}},
	}
	planning := applyPlatformLocalDiscovery(context.Background(), target, Options{Broker: broker, DirectPeers: fakeCloudPairingPeerFactory{}, Now: func() time.Time { return now }, EnableLocalDiscovery: true})
	route := planning.Routes[localDiscoveryRouteID]
	if len(route.SignalingAddresses) != 1 || route.SignalingAddresses[0] != "192.168.1.12:41120" {
		t.Fatalf("pairing LAN route = %#v", route)
	}
	if target.Routes["direct"].SignalingAddresses[0] != "192.168.1.8:41120" {
		t.Fatal("pairing discovery mutated signed bootstrap route")
	}
	if planning.Routes["direct"].SignalingAddresses[0] != "192.168.1.8:41120" {
		t.Fatal("pairing discovery overwrote configured Direct fallback")
	}
}

func TestApplyPlatformLocalDiscoveryDoesNotOverwriteAConfiguredRouteID(t *testing.T) {
	broker := binding.NewPlatformBroker()
	defer broker.Close()
	now := time.Now().UTC()
	go answerLocalDiscoveryRequest(t, broker, &bindingpb.LocalDiscoveryLookupResult{Candidates: []*bindingpb.LocalDiscoveryCandidate{
		{Address: "192.168.1.13", Port: 41120, ProtocolVersion: uint32(wire.Version), ExpiresAtUnixNano: now.Add(time.Minute).UnixNano()},
	}})
	target := endpoint.Endpoint{
		ID: "studio", DaemonIdentity: endpoint.DaemonIdentity{DeviceID: "device-1", DeviceFingerprint: "ed25519-sha256:test"},
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{
			"cloud":               {ID: "cloud", Kind: endpoint.RouteManagedWebRTC, Enabled: true, CredentialRef: "grant:1", Source: endpoint.SourceCloud, PolicySource: endpoint.SourceCloud, TargetDeviceID: "device-1", AccountProfileRef: "default", RelayMode: endpoint.RelayAuto},
			localDiscoveryRouteID: {ID: localDiscoveryRouteID, Kind: endpoint.RouteSSHWebRTCTCP, Enabled: true, Source: endpoint.SourceUser, PolicySource: endpoint.SourceUser},
		},
	}
	planning := applyPlatformLocalDiscovery(context.Background(), target, Options{Broker: broker, DirectPeers: fakeCloudPairingPeerFactory{}, Now: func() time.Time { return now }, EnableLocalDiscovery: true})
	if planning.Routes[localDiscoveryRouteID].Kind != endpoint.RouteSSHWebRTCTCP {
		t.Fatal("configured route was overwritten by local discovery")
	}
	if planning.Routes["lan-discovery-2"].Kind != endpoint.RouteDirectWebRTCTCP {
		t.Fatalf("collision-safe LAN route missing: %#v", planning.Routes)
	}
}

func answerLocalDiscoveryRequest(t *testing.T, broker *binding.PlatformBroker, result *bindingpb.LocalDiscoveryLookupResult) {
	t.Helper()
	payload, err := broker.NextRequest(context.Background())
	if err != nil {
		t.Error(err)
		return
	}
	request := &bindingpb.PlatformRequest{}
	if err := proto.Unmarshal(payload, request); err != nil {
		t.Error(err)
		return
	}
	if request.GetLocalDiscoveryLookup() == nil {
		t.Error("missing local discovery request")
		return
	}
	response := &bindingpb.PlatformResponse{RequestId: request.GetRequestId(), Response: &bindingpb.PlatformResponse_LocalDiscovery{LocalDiscovery: result}}
	encoded, err := proto.Marshal(response)
	if err != nil {
		t.Error(err)
		return
	}
	if err := broker.Complete(encoded); err != nil {
		t.Error(err)
	}
}

package enginehost

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"sync/atomic"
	"testing"

	"github.com/anytty/anytty/client/adapter/direct"
	"github.com/anytty/anytty/client/binding"
	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	clientruntime "github.com/anytty/anytty/client/runtime"
	cloudclient "github.com/anytty/anytty/cloud/client"
	"github.com/anytty/anytty/proto/bindingpb"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

func TestDecodeBootstrapAcceptsManualPairingClaimCode(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04}
	decoded, err := decodeBootstrap(remoteauth.EncodePairingClaimCode(payload))
	if err != nil || !bytes.Equal(decoded, payload) {
		t.Fatalf("manual pairing code decoded=%x err=%v", decoded, err)
	}
}

func TestPairingClaimCandidateProducesValidDirectAttempt(t *testing.T) {
	publicKey := bytes.Repeat([]byte{0x21}, 32)
	route := &remoteauthpb.PairingRouteSeed{RouteId: "direct", Route: &remoteauthpb.PairingRouteSeed_DirectWebrtcTcp{DirectWebrtcTcp: &remoteauthpb.PairingDirectRouteSeed{SignalingAddress: "127.0.0.1:41001", IceTcpAddress: "127.0.0.1:41002"}}}
	candidate, err := remoteauth.PairingClaimEndpointCandidate(&remoteauthpb.PairingClaimOffer{SchemaVersion: remoteauth.PairingClaimOfferVersion, Claim: bytes.Repeat([]byte{0x31}, 16), DeviceId: "device-1", DevicePublicKey: publicKey, ExpiresAtUnixNano: 1, Routes: []*remoteauthpb.PairingRouteSeed{route}})
	if err != nil {
		t.Fatal(err)
	}
	selected, err := pairingClaimRoutes(candidate, Options{DirectPeers: fakeDirectPeerFactory{}})
	if err != nil {
		t.Fatal(err)
	}
	target := pairingTarget("device-1", candidate.Identity, selected, "credential:device-1")
	if _, err := clientruntime.NewAttemptRequest(target, selected[0].ID, 1, clientruntime.ConnectIntentInteractive); err != nil {
		t.Fatalf("claim attempt is invalid: %#v err=%v", target, err)
	}
}

type credentialAvailability map[string]bool

func (values credentialAvailability) Available(_ context.Context, _, reference string) bool {
	return values[reference]
}

func TestPairingTargetKeepsManagedFieldsOutOfDirectRoute(t *testing.T) {
	identity := endpoint.DaemonIdentity{DeviceID: "daemon-1", DeviceFingerprint: "ed25519-sha256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}
	route := endpoint.AccessRoute{
		ID: "direct", Kind: endpoint.RouteDirectWebRTCTCP, Enabled: true, Source: endpoint.SourceBootstrap, PolicySource: endpoint.SourceBootstrap,
		SignalingAddresses: []string{"127.0.0.1:41120"}, ICETCPAddresses: []string{"127.0.0.1:41121"},
	}
	target := pairingTarget("daemon-1", identity, []endpoint.AccessRoute{route}, "android-access-daemon-1")
	direct, ok := target.Route("direct")
	if !ok {
		t.Fatal("Direct pairing target route is missing")
	}
	if direct.TargetDeviceID != "" {
		t.Fatalf("Direct pairing target leaked managed target_device_id %q", direct.TargetDeviceID)
	}
	if direct.CredentialRef != "android-access-daemon-1" {
		t.Fatalf("Direct pairing credential_ref = %q", direct.CredentialRef)
	}
	if _, err := clientruntime.NewAttemptRequest(target, "direct", 1, clientruntime.ConnectIntentInteractive); err != nil {
		t.Fatalf("Direct pairing attempt rejected: %v", err)
	}
}

func TestPairingClaimRoutesRequireCloudBootstrapDependencies(t *testing.T) {
	direct := endpoint.AccessRoute{ID: "direct", Kind: endpoint.RouteDirectWebRTCTCP, Enabled: true}
	cloud := endpoint.AccessRoute{ID: "cloud", Kind: endpoint.RouteManagedWebRTC, Enabled: true}
	target := endpoint.EndpointCandidate{Routes: []endpoint.AccessRoute{direct, cloud}}
	routes, err := pairingClaimRoutes(target, Options{DirectPeers: fakeDirectPeerFactory{}})
	if err != nil || len(routes) != 1 || routes[0].ID != direct.ID {
		t.Fatalf("pairing routes = %#v err=%v", routes, err)
	}
	if _, err := pairingClaimRoutes(endpoint.EndpointCandidate{Routes: []endpoint.AccessRoute{{ID: direct.ID, Kind: direct.Kind}}}, Options{DirectPeers: fakeDirectPeerFactory{}}); err == nil {
		t.Fatal("disabled pairing route was accepted")
	}
	broker := binding.NewPlatformBroker()
	defer broker.Close()
	routes, err = pairingClaimRoutes(endpoint.EndpointCandidate{Routes: []endpoint.AccessRoute{cloud}}, Options{
		Broker: broker, DirectPeers: fakeCloudPairingPeerFactory{}, CloudProduct: cloudv1.ClientProduct_CLIENT_PRODUCT_ANDROID,
	})
	if err != nil || len(routes) != 1 || routes[0].ID != cloud.ID {
		t.Fatalf("Cloud bootstrap pairing routes = %#v err=%v", routes, err)
	}
}

func TestRoutePlanEnvironmentDisablesCloudWithoutAConnector(t *testing.T) {
	directPriority, cloudPriority := 10, 20
	target := endpoint.Endpoint{
		ID: "studio", Label: "Studio", LabelSource: endpoint.SourceUser, Enabled: true, ConnectMode: endpoint.ConnectOnDemand,
		DaemonIdentity: endpoint.DaemonIdentity{DeviceID: "daemon-1", DeviceFingerprint: "ed25519-sha256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"},
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{
			"direct": {ID: "direct", Kind: endpoint.RouteDirectWebRTCTCP, Enabled: true, Priority: &directPriority, CredentialRef: "grant:studio", Source: endpoint.SourceManual, PolicySource: endpoint.SourceUser, SignalingAddresses: []string{"studio:41120"}, ICETCPAddresses: []string{"studio:41121"}},
			"cloud":  {ID: "cloud", Kind: endpoint.RouteManagedWebRTC, Enabled: true, Priority: &cloudPriority, CredentialRef: "grant:studio", Source: endpoint.SourceCloud, PolicySource: endpoint.SourceUser, TargetDeviceID: "daemon-1", RelayMode: endpoint.RelayAuto},
		},
	}
	planning, environment, err := routePlanEnvironment(context.Background(), target, Options{DirectPeers: fakeDirectPeerFactory{}}, credentialAvailability{"grant:studio": true}, platformCloudProfiles{})
	if err != nil {
		t.Fatal(err)
	}
	if route, _ := planning.Route("cloud"); route.Enabled {
		t.Fatal("logged-out managed route remained eligible")
	}
	if route, _ := target.Route("cloud"); !route.Enabled {
		t.Fatal("eligibility projection mutated persistent endpoint")
	}
	if len(environment.SupportedRouteKinds) != 1 || environment.SupportedRouteKinds[0] != endpoint.RouteDirectWebRTCTCP {
		t.Fatalf("supported route kinds = %#v", environment.SupportedRouteKinds)
	}
	plan, err := (endpoint.RouteSelectionPlanner{}).Plan(endpoint.RouteSelectionRequest{
		Endpoint: planning, Intent: endpoint.ConnectIntent{Kind: "interactive"}, Generation: 1,
		SupportedRouteKinds: environment.SupportedRouteKinds, AvailableCredentialRefs: environment.AvailableCredentialRefs,
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts := plan.Groups()[0].Attempts(); len(attempts) != 1 || attempts[0].Route.ID != "direct" {
		t.Fatalf("Cloud failure affected Direct plan: %#v", attempts)
	}
}

func TestAttemptCloudProfileSnapshotAvoidsDuplicatePlatformResolve(t *testing.T) {
	broker := binding.NewPlatformBroker()
	defer broker.Close()
	var calls atomic.Int32
	pumpPlatformResponses(t, broker, func(request *bindingpb.PlatformRequest) *bindingpb.PlatformResponse {
		calls.Add(1)
		return &bindingpb.PlatformResponse{RequestId: request.GetRequestId(), Response: &bindingpb.PlatformResponse_CloudProfile{CloudProfile: &bindingpb.CloudProfileRecord{
			AccountProfileRef: "account:test", ControllerAddress: "controller.test:443", ControllerServerName: "controller.test",
		}}}
	})
	profiles := platformCloudProfiles{
		broker: broker,
		bootID: uuid.NewString(),
		cache:  &platformCloudProfileCache{clients: make(map[string]*cloudclient.Client)},
	}
	first, err := profiles.Resolve(context.Background(), "account:test")
	if err != nil {
		t.Fatal(err)
	}
	second, err := profiles.Resolve(context.Background(), "account:test")
	if err != nil {
		t.Fatal(err)
	}
	if first != second || calls.Load() != 1 {
		t.Fatalf("profile clients same=%t platform resolves=%d", first == second, calls.Load())
	}
}

func TestAttemptCredentialSnapshotAvoidsDuplicatePlatformResolve(t *testing.T) {
	identity, err := remoteauth.GenerateClientAccessIdentity("studio", nil)
	if err != nil {
		t.Fatal(err)
	}
	broker := binding.NewPlatformBroker()
	defer broker.Close()
	var calls atomic.Int32
	pumpPlatformResponses(t, broker, func(request *bindingpb.PlatformRequest) *bindingpb.PlatformResponse {
		calls.Add(1)
		return &bindingpb.PlatformResponse{RequestId: request.GetRequestId(), Response: &bindingpb.PlatformResponse_Credential{Credential: &bindingpb.CredentialRecord{
			EndpointId: "studio", CredentialRef: "credential:studio", PublicKey: append([]byte(nil), identity.PublicKey...),
			KeyFingerprint: identity.Fingerprint, CapabilityGrant: "grant", CloudRouteGrant: []byte("route"),
		}}}
	})
	credentials := newPlatformCredentials(broker)
	first, err := credentials.ResolveClientCredential(context.Background(), "studio", "credential:studio")
	if err != nil {
		t.Fatal(err)
	}
	first.Identity.PublicKey[0] ^= 0xff
	second, err := credentials.ResolveClientCredential(context.Background(), "studio", "credential:studio")
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 || !ed25519.PublicKey(second.Identity.PublicKey).Equal(identity.PublicKey) {
		t.Fatalf("credential platform resolves=%d clone matches=%t", calls.Load(), ed25519.PublicKey(second.Identity.PublicKey).Equal(identity.PublicKey))
	}
}

func pumpPlatformResponses(t *testing.T, broker *binding.PlatformBroker, response func(*bindingpb.PlatformRequest) *bindingpb.PlatformResponse) {
	t.Helper()
	go func() {
		for {
			payload, err := broker.NextRequest(context.Background())
			if err != nil {
				return
			}
			request := &bindingpb.PlatformRequest{}
			if err := proto.Unmarshal(payload, request); err != nil {
				t.Errorf("decode platform request: %v", err)
				return
			}
			encoded, err := proto.Marshal(response(request))
			if err != nil {
				t.Errorf("encode platform response: %v", err)
				return
			}
			if err := broker.Complete(encoded); err != nil {
				return
			}
		}
	}()
}

type fakeDirectPeerFactory struct{ direct.PeerFactory }

type fakeCloudPairingPeerFactory struct{ fakeDirectPeerFactory }

func (fakeCloudPairingPeerFactory) OpenCloudPeer(context.Context, port.WebRTCConfig) (port.WebRTCPeer, error) {
	return nil, nil
}

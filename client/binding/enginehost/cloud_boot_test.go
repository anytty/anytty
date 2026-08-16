package enginehost

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	peeradapter "github.com/anytty/anytty/client/adapter/peer"
	"github.com/anytty/anytty/client/binding"
	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	clientruntime "github.com/anytty/anytty/client/runtime"
	cloudclient "github.com/anytty/anytty/cloud/client"
	cloudprotocol "github.com/anytty/anytty/cloud/protocol"
	"github.com/anytty/anytty/cloud/securetransport"
	"github.com/anytty/anytty/cloud/ticket"
	"github.com/anytty/anytty/proto/bindingpb"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var errCloudBootSignalingComplete = errors.New("stop after Cloud signaling")

func TestHostCloudBootIdentitySpansReconnectAttempts(t *testing.T) {
	gateway, address, serverName, caPEM := startCloudBootGateway(t)
	profile := &bindingpb.CloudProfileRecord{AccountProfileRef: "account:test", ControllerAddress: address, ControllerServerName: serverName, ControllerCaPem: caPEM}
	host := newCloudBootHost(t, profile)
	otherHost := newCloudBootHost(t, profile)
	if host.cloudBootID == otherHost.cloudBootID {
		t.Fatal("new engine Hosts shared a Cloud boot ID")
	}

	profiles := cloudProfilesFromHost(t, host)
	otherProfiles := cloudProfilesFromHost(t, otherHost)
	identity, err := remoteauth.GenerateClientAccessIdentity("engine-cloud", nil)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := remoteauth.NewPrivateClientAccessSigner(identity)
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := cloudclient.NewCachedRoute(
		&cloudv1.EdgeLocator{EdgeId: gateway.edgeID, PublicEndpoint: address, ServerName: serverName, CaCertificatePem: caPEM},
		&cloudv1.SignedEnvelope{KeyId: "daemon-route", Payload: []byte("route-grant"), Signature: bytes.Repeat([]byte{0x41}, ed25519.SignatureSize)},
	)
	if err != nil {
		t.Fatal(err)
	}
	target := cloudBootTarget()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	const sequentialAttempts = 3
	var previous uint64
	for range sequentialAttempts {
		generation, allocationErr := allocateCloudAttempt(host, target)
		if allocationErr != nil {
			t.Fatal(allocationErr)
		}
		if generation != previous+1 {
			t.Fatalf("sequential attempt generation=%d after %d", generation, previous)
		}
		previous = generation
		if err := exchangeCloudHello(ctx, profiles, resolution, identity, signer, generation); err != nil {
			t.Fatalf("sequential Cloud attempt %d: %v", generation, err)
		}
	}

	otherGeneration, err := allocateCloudAttempt(otherHost, target)
	if err != nil {
		t.Fatal(err)
	}
	if err := exchangeCloudHello(ctx, otherProfiles, resolution, identity, signer, otherGeneration); err != nil {
		t.Fatalf("new Host Cloud attempt: %v", err)
	}

	const concurrentAttempts = 8
	generations := make([]uint64, concurrentAttempts)
	for index := range generations {
		generation, allocationErr := allocateCloudAttempt(host, target)
		if allocationErr != nil {
			t.Fatal(allocationErr)
		}
		if generation != previous+1 {
			t.Fatalf("concurrent attempt allocation generation=%d after %d", generation, previous)
		}
		previous = generation
		generations[index] = generation
	}
	errors := make(chan error, concurrentAttempts)
	var attempts sync.WaitGroup
	for _, generation := range generations {
		generation := generation
		attempts.Add(1)
		go func() {
			defer attempts.Done()
			errors <- exchangeCloudHello(ctx, profiles, resolution, identity, signer, generation)
		}()
	}
	attempts.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent Cloud attempt: %v", err)
		}
	}

	hostCaptures := gateway.capturesForBoot(host.cloudBootID)
	if len(hostCaptures) != sequentialAttempts+concurrentAttempts {
		t.Fatalf("same-Host Hello count=%d want %d", len(hostCaptures), sequentialAttempts+concurrentAttempts)
	}
	for index := range sequentialAttempts {
		if got := hostCaptures[index].hello.GetHello().GetAttemptGeneration(); got != uint64(index+1) {
			t.Fatalf("sequential wire attempt_generation[%d]=%d", index, got)
		}
	}
	allGenerations := make([]uint64, 0, len(hostCaptures))
	for _, capture := range hostCaptures {
		if capture.hello.GetBootId() != host.cloudBootID {
			t.Fatalf("same Host emitted boot_id %q want %q", capture.hello.GetBootId(), host.cloudBootID)
		}
		allGenerations = append(allGenerations, capture.hello.GetHello().GetAttemptGeneration())
	}
	sort.Slice(allGenerations, func(i, j int) bool { return allGenerations[i] < allGenerations[j] })
	for index, generation := range allGenerations {
		if generation != uint64(index+1) {
			t.Fatalf("same-Host wire generations=%v", allGenerations)
		}
	}

	otherCaptures := gateway.capturesForBoot(otherHost.cloudBootID)
	if len(otherCaptures) != 1 || otherCaptures[0].hello.GetBootId() == hostCaptures[0].hello.GetBootId() {
		t.Fatalf("new Host wire boot IDs did not differ: first=%q second=%q", hostCaptures[0].hello.GetBootId(), otherCaptures[0].hello.GetBootId())
	}
	tampered := proto.Clone(hostCaptures[0].hello).(*cloudv1.ClientSignal)
	tampered.BootId = otherCaptures[0].hello.GetBootId()
	if err := ticket.VerifyClientHelloProof(identity.PublicKey, tampered.GetHello().GetClientProof(), hostCaptures[0].challenge, tampered, hostCaptures[0].challenge.GetIssuedAt().AsTime()); err == nil {
		t.Fatal("ClientHello proof accepted a cross-boot transcript")
	}
}

func TestHostPublicCloudEntriesShareBootIdentity(t *testing.T) {
	gateway, address, serverName, caPEM := startCloudBootGateway(t)
	platform, target, pairingPayload := newCloudEntryFixture(t, gateway.edgeID, address, serverName, caPEM)
	host := newPublicCloudBootHost(t, platform, target)
	otherHost := newPublicCloudBootHost(t, platform, target)
	if host.cloudBootID == otherHost.cloudBootID {
		t.Fatal("new public-entry Hosts shared a Cloud boot ID")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	open := func(host *Host) error {
		_, err := host.OpenSession(ctx, &bindingpb.OpenSessionRequest{
			EndpointId: string(target.ID), RouteOverride: "cloud", Intent: bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE,
		})
		return err
	}
	pair := func(host *Host) error {
		_, err := host.ImportPairing(ctx, &bindingpb.ImportPairingRequest{
			PortablePayload: pairingPayload, ExpectedEndpointId: string(target.ID),
		})
		return err
	}
	assertReachedSignaling := func(entry string, err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s unexpectedly completed its test transport", entry)
		}
	}

	for attempt := range 2 {
		assertReachedSignaling(fmt.Sprintf("OpenSession %d", attempt), open(host))
		assertReachedSignaling(fmt.Sprintf("ImportPairing %d", attempt), pair(host))
	}

	const concurrentAttempts = 8
	errors := make(chan error, concurrentAttempts)
	var attempts sync.WaitGroup
	for index := range concurrentAttempts {
		index := index
		attempts.Add(1)
		go func() {
			defer attempts.Done()
			if index%2 == 0 {
				errors <- open(host)
				return
			}
			errors <- pair(host)
		}()
	}
	attempts.Wait()
	close(errors)
	for err := range errors {
		assertReachedSignaling("concurrent public Cloud entry", err)
	}

	assertReachedSignaling("new Host OpenSession", open(otherHost))
	assertReachedSignaling("new Host ImportPairing", pair(otherHost))

	hostCaptures := gateway.capturesForBoot(host.cloudBootID)
	if len(hostCaptures) != 4+concurrentAttempts {
		t.Fatalf("public same-Host Hello count=%d want %d", len(hostCaptures), 4+concurrentAttempts)
	}
	assertCloudBootGenerations(t, hostCaptures, host.cloudBootID)
	capability, pairing := 0, 0
	for _, capture := range hostCaptures {
		switch {
		case capture.hello.GetHello().GetCloudRouteGrant() != nil:
			capability++
		case capture.hello.GetHello().GetPairingAdmission() != nil:
			pairing++
		default:
			t.Fatal("public Cloud entry emitted Hello without authorization")
		}
	}
	if capability != 2+concurrentAttempts/2 || pairing != 2+concurrentAttempts/2 {
		t.Fatalf("public Cloud Hello modes capability=%d pairing=%d", capability, pairing)
	}

	otherCaptures := gateway.capturesForBoot(otherHost.cloudBootID)
	if len(otherCaptures) != 2 {
		t.Fatalf("new Host public Hello count=%d want 2", len(otherCaptures))
	}
	assertCloudBootGenerations(t, otherCaptures, otherHost.cloudBootID)
	if hostCaptures[0].hello.GetBootId() == otherCaptures[0].hello.GetBootId() {
		t.Fatal("public Cloud entries reused boot ID across Hosts")
	}
}

func TestEndpointCloudPresenceRefreshesStaleLocatorAndStoresCurrentEdge(t *testing.T) {
	oldGateway, oldAddress, oldServerName, oldCAPEM := startCloudPresenceGateway(t, "edge-old", false)
	currentGateway, currentAddress, currentServerName, currentCAPEM := startCloudPresenceGateway(t, "edge-current", true)
	currentLocator := &cloudv1.EdgeLocator{EdgeId: currentGateway.edgeID, Name: "Current Edge", Region: "TEST", PublicEndpoint: currentAddress, ServerName: currentServerName, CaCertificatePem: currentCAPEM}
	directory, controllerAddress, controllerServerName, controllerCAPEM := startCloudPresenceDirectory(t, currentLocator)
	platform, target, _ := newCloudEntryFixture(t, oldGateway.edgeID, oldAddress, oldServerName, oldCAPEM)
	platform.profile = &bindingpb.CloudProfileRecord{
		AccountProfileRef: "account:test", ControllerAddress: controllerAddress, ControllerServerName: controllerServerName, ControllerCaPem: controllerCAPEM,
	}
	host := newPublicCloudBootHost(t, platform, target)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	presence, err := host.GetEndpointCloudPresence(ctx, &bindingpb.EndpointCloudPresenceGetRequest{EndpointId: string(target.ID)})
	if err != nil || !presence.GetOnline() {
		t.Fatalf("presence after refresh online=%t err=%v", presence.GetOnline(), err)
	}
	if oldGateway.calls.Load() != 1 || currentGateway.calls.Load() != 1 || directory.resolves.Load() != 1 {
		t.Fatalf("presence calls old=%d current=%d resolves=%d", oldGateway.calls.Load(), currentGateway.calls.Load(), directory.resolves.Load())
	}
	if presence.GetLocatorSource() != "controller" || !presence.GetRefreshedFromController() ||
		presence.GetEdgeId() != currentGateway.edgeID || presence.GetEdgePublicEndpoint() != currentAddress ||
		presence.GetEdgeName() != "Current Edge" || presence.GetEdgeRegion() != "TEST" ||
		presence.GetDaemonId() != "daemon-enginehost-presence" ||
		presence.GetDeviceId() != target.DaemonIdentity.DeviceID ||
		presence.GetDeviceFingerprint() != target.DaemonIdentity.DeviceFingerprint {
		t.Fatalf("presence diagnostics = %#v", presence)
	}
	stored, err := cloudclient.DecodeEdgeLocator(platform.storedCloudEdgeLocator())
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetEdgeId() != currentGateway.edgeID || stored.GetPublicEndpoint() != currentAddress {
		t.Fatalf("stored locator = %#v, want edge %q at %q", stored, currentGateway.edgeID, currentAddress)
	}
}

func assertCloudBootGenerations(t *testing.T, captures []cloudBootCapture, bootID string) {
	t.Helper()
	generations := make([]uint64, 0, len(captures))
	for _, capture := range captures {
		if capture.hello.GetBootId() != bootID {
			t.Fatalf("wire boot_id=%q want %q", capture.hello.GetBootId(), bootID)
		}
		generations = append(generations, capture.hello.GetHello().GetAttemptGeneration())
	}
	sort.Slice(generations, func(i, j int) bool { return generations[i] < generations[j] })
	for index, generation := range generations {
		if generation != uint64(index+1) {
			t.Fatalf("wire attempt generations=%v", generations)
		}
	}
}

type cloudEntryPlatform struct {
	mu               sync.Mutex
	t                *testing.T
	profile          *bindingpb.CloudProfileRecord
	clientIdentity   remoteauth.ClientAccessIdentity
	capabilityGrant  string
	cloudRouteGrant  []byte
	cloudEdgeLocator []byte
}

func newCloudEntryFixture(t *testing.T, edgeID, address, serverName string, caPEM []byte) (*cloudEntryPlatform, endpoint.Endpoint, string) {
	t.Helper()
	const daemonID = "daemon-enginehost-presence"
	_, daemonPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	daemonIdentity, err := remoteauth.NewIdentity("daemon-engine", daemonPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	clientIdentity, err := remoteauth.GenerateClientAccessIdentity("engine-cloud", nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	grant, err := remoteauth.Issue(daemonIdentity.PrivateKey, remoteauth.Claims{
		GrantID: "enginehost-public-grant", IssuerDeviceID: daemonIdentity.DeviceID, SubjectKeyFingerprint: clientIdentity.Fingerprint,
		Scope: remoteauth.FullDaemonScope(), IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	routeClaims, err := proto.Marshal(&cloudv1.CloudRouteGrantClaims{
		GrantId: "route-grant", DaemonId: daemonID, ClientPublicKey: append([]byte(nil), clientIdentity.PublicKey...),
		Product: cloudv1.ClientProduct_CLIENT_PRODUCT_ANDROID, IssuedAt: timestamppb.New(now.Add(-time.Minute)), ExpiresAt: timestamppb.New(now.Add(time.Hour)),
	})
	if err != nil {
		t.Fatal(err)
	}
	routeGrant, err := proto.Marshal(&cloudv1.SignedEnvelope{KeyId: "daemon-route", Payload: routeClaims, Signature: bytes.Repeat([]byte{0x51}, ed25519.SignatureSize)})
	if err != nil {
		t.Fatal(err)
	}
	edgeLocator, err := cloudclient.EncodeEdgeLocator(&cloudv1.EdgeLocator{EdgeId: edgeID, PublicEndpoint: address, ServerName: serverName, CaCertificatePem: caPEM})
	if err != nil {
		t.Fatal(err)
	}
	caFingerprint, err := securetransport.EdgeCACertificateDERFingerprint(caPEM)
	if err != nil {
		t.Fatal(err)
	}
	offer, err := remoteauth.EncodePairingClaimOffer(&remoteauthpb.PairingClaimOffer{
		SchemaVersion: remoteauth.PairingClaimOfferVersion, Claim: bytes.Repeat([]byte{0x61}, 16), DeviceId: daemonIdentity.DeviceID,
		DevicePublicKey: daemonIdentity.PublicKey, ExpiresAtUnixNano: now.Add(time.Hour).UnixNano(),
		Routes: []*remoteauthpb.PairingRouteSeed{{RouteId: "cloud", Route: &remoteauthpb.PairingRouteSeed_ManagedWebrtc{ManagedWebrtc: &remoteauthpb.PairingManagedRouteSeed{
			DaemonId: daemonIdentity.DeviceID, EdgeId: edgeID, PublicEndpoint: address, ServerName: serverName, CaCertificateDerSha256: caFingerprint,
		}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	target := endpoint.Endpoint{
		ID: "engine-cloud", Label: "Engine Cloud", LabelSource: endpoint.SourceUser, ConnectMode: endpoint.ConnectOnDemand, Enabled: true,
		DaemonIdentity: endpoint.DaemonIdentity{DeviceID: daemonIdentity.DeviceID, DeviceFingerprint: daemonIdentity.Fingerprint},
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{"cloud": {
			ID: "cloud", Kind: endpoint.RouteManagedWebRTC, Enabled: true, CredentialRef: "credential:engine", Source: endpoint.SourceCloud,
			PolicySource: endpoint.SourceUser, TargetDeviceID: daemonIdentity.DeviceID, AccountProfileRef: "account:test", RelayMode: endpoint.RelayAuto,
		}},
	}
	return &cloudEntryPlatform{
		t: t, profile: &bindingpb.CloudProfileRecord{AccountProfileRef: "account:test", ControllerAddress: address, ControllerServerName: serverName, ControllerCaPem: caPEM},
		clientIdentity: clientIdentity, capabilityGrant: grant, cloudRouteGrant: routeGrant, cloudEdgeLocator: edgeLocator,
	}, target, base64.RawURLEncoding.EncodeToString(offer)
}

func newPublicCloudBootHost(t *testing.T, platform *cloudEntryPlatform, target endpoint.Endpoint) *Host {
	t.Helper()
	broker := binding.NewPlatformBroker()
	go platform.pump(broker)
	host, err := New(Options{
		Broker: broker, DirectPeers: signalingOnlyCloudPeerFactory{}, ClientName: "enginehost-public-test", CredentialPrefix: "enginehost:test:",
		CloudProduct: cloudv1.ClientProduct_CLIENT_PRODUCT_ANDROID,
	})
	if err != nil {
		t.Fatal(err)
	}
	host.registry = endpoint.Registry{Version: endpoint.RegistryVersion, Default: target.ID, Endpoints: map[endpoint.EndpointID]endpoint.Endpoint{target.ID: target}}
	host.registryLoaded = true
	t.Cleanup(func() { _ = host.Close() })
	return host
}

func (platform *cloudEntryPlatform) pump(broker *binding.PlatformBroker) {
	for {
		payload, err := broker.NextRequest(context.Background())
		if err != nil {
			return
		}
		request := &bindingpb.PlatformRequest{}
		if err := proto.Unmarshal(payload, request); err != nil {
			platform.t.Errorf("decode public Cloud platform request: %v", err)
			return
		}
		response := &bindingpb.PlatformResponse{RequestId: request.GetRequestId()}
		switch value := request.GetRequest().(type) {
		case *bindingpb.PlatformRequest_CloudProfileResolve:
			profile := proto.Clone(platform.profile).(*bindingpb.CloudProfileRecord)
			profile.AccountProfileRef = value.CloudProfileResolve.GetAccountProfileRef()
			response.Response = &bindingpb.PlatformResponse_CloudProfile{CloudProfile: profile}
		case *bindingpb.PlatformRequest_CredentialResolve:
			response.Response = &bindingpb.PlatformResponse_Credential{Credential: platform.credential(value.CredentialResolve.GetEndpointId(), value.CredentialResolve.GetCredentialRef(), true)}
		case *bindingpb.PlatformRequest_CredentialPrepare:
			response.Response = &bindingpb.PlatformResponse_Credential{Credential: platform.credential(value.CredentialPrepare.GetEndpointId(), value.CredentialPrepare.GetCredentialRef(), false)}
		case *bindingpb.PlatformRequest_CredentialSign:
			response.Response = &bindingpb.PlatformResponse_CredentialSign{CredentialSign: &bindingpb.CredentialSignResponse{Signature: ed25519.Sign(platform.clientIdentity.PrivateKey, value.CredentialSign.GetPayload())}}
		case *bindingpb.PlatformRequest_CredentialBind:
			response.Response = &bindingpb.PlatformResponse_Credential{Credential: platform.bindCredential(value.CredentialBind.GetEndpointId(), value.CredentialBind.GetCredentialRef(), value.CredentialBind.GetCapabilityGrant(), value.CredentialBind.GetCloudRouteGrant(), value.CredentialBind.GetCloudEdgeLocator())}
		default:
			platform.t.Errorf("unexpected public Cloud platform request: %T", value)
			return
		}
		encoded, err := proto.Marshal(response)
		if err != nil {
			platform.t.Errorf("encode public Cloud platform response: %v", err)
			return
		}
		if err := broker.Complete(encoded); err != nil {
			return
		}
	}
}

func (platform *cloudEntryPlatform) credential(endpointID, reference string, bound bool) *bindingpb.CredentialRecord {
	platform.mu.Lock()
	defer platform.mu.Unlock()
	return platform.credentialLocked(endpointID, reference, bound)
}

func (platform *cloudEntryPlatform) bindCredential(endpointID, reference, capabilityGrant string, cloudRouteGrant, cloudEdgeLocator []byte) *bindingpb.CredentialRecord {
	platform.mu.Lock()
	defer platform.mu.Unlock()
	platform.capabilityGrant = capabilityGrant
	platform.cloudRouteGrant = append([]byte(nil), cloudRouteGrant...)
	platform.cloudEdgeLocator = append([]byte(nil), cloudEdgeLocator...)
	return platform.credentialLocked(endpointID, reference, true)
}

func (platform *cloudEntryPlatform) storedCloudEdgeLocator() []byte {
	platform.mu.Lock()
	defer platform.mu.Unlock()
	return append([]byte(nil), platform.cloudEdgeLocator...)
}

func (platform *cloudEntryPlatform) credentialLocked(endpointID, reference string, bound bool) *bindingpb.CredentialRecord {
	record := &bindingpb.CredentialRecord{
		EndpointId: endpointID, CredentialRef: reference, PublicKey: append([]byte(nil), platform.clientIdentity.PublicKey...),
		KeyFingerprint: platform.clientIdentity.Fingerprint,
	}
	if bound {
		record.CapabilityGrant = platform.capabilityGrant
		record.CloudRouteGrant = append([]byte(nil), platform.cloudRouteGrant...)
		record.CloudEdgeLocator = append([]byte(nil), platform.cloudEdgeLocator...)
	}
	return record
}

type signalingOnlyCloudPeerFactory struct{ fakeDirectPeerFactory }

func (signalingOnlyCloudPeerFactory) OpenCloudPeer(context.Context, port.WebRTCConfig) (port.WebRTCPeer, error) {
	return &signalingOnlyCloudPeer{channel: &signalingOnlyCloudChannel{}}, nil
}

type signalingOnlyCloudPeer struct {
	channel *signalingOnlyCloudChannel
}

func (peer *signalingOnlyCloudPeer) Channel() port.WebRTCMessageChannel { return peer.channel }
func (*signalingOnlyCloudPeer) CreateOffer(context.Context) (string, error) {
	return "enginehost-public-offer", nil
}
func (*signalingOnlyCloudPeer) ApplyAnswer(context.Context, string, []port.ICECandidate) error {
	return nil
}
func (*signalingOnlyCloudPeer) WaitReady(context.Context) error { return nil }
func (*signalingOnlyCloudPeer) RemoteCertificateFingerprint() (string, error) {
	parts := make([]string, 32)
	for index := range parts {
		parts[index] = "11"
	}
	return "sha-256:" + strings.Join(parts, ":"), nil
}
func (*signalingOnlyCloudPeer) ObservedPath() endpoint.Path { return endpoint.PathDirect }
func (*signalingOnlyCloudPeer) Snapshot(time.Time) (port.WebRTCPeerSnapshot, bool) {
	return port.WebRTCPeerSnapshot{}, false
}
func (peer *signalingOnlyCloudPeer) Close() error { return peer.channel.Close() }

type signalingOnlyCloudChannel struct {
	mu           sync.Mutex
	closeHandler func()
	closed       bool
}

func (*signalingOnlyCloudChannel) SetMessageHandler(func([]byte)) {}
func (channel *signalingOnlyCloudChannel) SetCloseHandler(handler func()) {
	channel.mu.Lock()
	channel.closeHandler = handler
	channel.closed = true
	channel.mu.Unlock()
	handler()
}
func (*signalingOnlyCloudChannel) BufferedAmount() uint64               { return 0 }
func (*signalingOnlyCloudChannel) SetBufferedAmountLowThreshold(uint64) {}
func (*signalingOnlyCloudChannel) SetBufferedAmountLowHandler(func())   {}
func (*signalingOnlyCloudChannel) Send([]byte) error                    { return errCloudBootSignalingComplete }
func (channel *signalingOnlyCloudChannel) Close() error {
	channel.mu.Lock()
	if channel.closed {
		channel.mu.Unlock()
		return nil
	}
	channel.closed = true
	handler := channel.closeHandler
	channel.mu.Unlock()
	if handler != nil {
		handler()
	}
	return nil
}

func cloudProfilesFromHost(t *testing.T, host *Host) platformCloudProfiles {
	t.Helper()
	connector, ok := host.routeConnectors(peeradapter.CapabilityAuthorizer{})[endpoint.RouteManagedWebRTC].(*platformCloudConnector)
	if !ok {
		t.Fatal("Host did not construct its Cloud connector")
	}
	return connector.profiles
}

func newCloudBootHost(t *testing.T, profile *bindingpb.CloudProfileRecord) *Host {
	t.Helper()
	broker := binding.NewPlatformBroker()
	pumpCloudProfile(t, broker, profile)
	host, err := New(Options{
		Broker: broker, DirectPeers: fakeCloudPairingPeerFactory{}, ClientName: "enginehost-test", CredentialPrefix: "enginehost:test",
		CloudProduct: cloudv1.ClientProduct_CLIENT_PRODUCT_ANDROID,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = host.Close() })
	return host
}

func pumpCloudProfile(t *testing.T, broker *binding.PlatformBroker, profile *bindingpb.CloudProfileRecord) {
	t.Helper()
	go func() {
		for {
			payload, err := broker.NextRequest(context.Background())
			if err != nil {
				return
			}
			request := &bindingpb.PlatformRequest{}
			if err := proto.Unmarshal(payload, request); err != nil {
				t.Errorf("decode Cloud profile request: %v", err)
				return
			}
			resolved := request.GetCloudProfileResolve()
			if resolved == nil || resolved.GetAccountProfileRef() != profile.GetAccountProfileRef() {
				t.Errorf("unexpected Cloud profile request: %v", request)
				return
			}
			response, err := proto.Marshal(&bindingpb.PlatformResponse{
				RequestId: request.GetRequestId(),
				Response:  &bindingpb.PlatformResponse_CloudProfile{CloudProfile: proto.Clone(profile).(*bindingpb.CloudProfileRecord)},
			})
			if err != nil {
				t.Errorf("encode Cloud profile response: %v", err)
				return
			}
			if err := broker.Complete(response); err != nil {
				t.Errorf("complete Cloud profile response: %v", err)
				return
			}
		}
	}()
}

func cloudBootTarget() endpoint.Endpoint {
	identity := endpoint.DaemonIdentity{DeviceID: "daemon-engine", DeviceFingerprint: "SHA256:daemon-engine"}
	return endpoint.Endpoint{
		ID: "engine-cloud", DaemonIdentity: identity,
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{
			"cloud": {
				ID: "cloud", Kind: endpoint.RouteManagedWebRTC, Enabled: true, CredentialRef: "credential:engine",
				Source: endpoint.SourceCloud, PolicySource: endpoint.SourceUser, TargetDeviceID: identity.DeviceID,
				AccountProfileRef: "account:test", RelayMode: endpoint.RelayAuto,
			},
		},
	}
}

func allocateCloudAttempt(host *Host, target endpoint.Endpoint) (uint64, error) {
	attempt, err := host.owner.BeginRouteAttempt(target, "cloud", clientruntime.ConnectIntentInteractive)
	if err != nil {
		return 0, err
	}
	return uint64(attempt.Stamp().Generation), nil
}

func exchangeCloudHello(ctx context.Context, profiles platformCloudProfiles, resolution *cloudclient.RouteResolution, identity remoteauth.ClientAccessIdentity, signer remoteauth.ClientAccessSigner, generation uint64) error {
	client, err := profiles.Resolve(ctx, "account:test")
	if err != nil {
		return err
	}
	session, err := client.Exchange(ctx, resolution, identity, signer, cloudv1.ClientProduct_CLIENT_PRODUCT_ANDROID, generation, cloudv1.RelayPreference_RELAY_PREFERENCE_AUTO, func(context.Context, *cloudv1.ClientReady) (string, error) {
		return "enginehost-offer", nil
	})
	if err != nil {
		return err
	}
	return session.Close()
}

type cloudBootCapture struct {
	challenge *cloudv1.EdgeChallenge
	hello     *cloudv1.ClientSignal
}

type cloudBootGateway struct {
	cloudv1.UnimplementedClientGatewayServer
	edgeID   string
	edgeBoot string
	sequence atomic.Uint64
	mu       sync.Mutex
	captures []cloudBootCapture
}

func (gateway *cloudBootGateway) Connect(stream grpc.BidiStreamingServer[cloudv1.ClientSignal, cloudv1.EdgeSignal]) error {
	sequence := gateway.sequence.Add(1)
	issuedAt := time.Now().UTC()
	challenge := &cloudv1.EdgeChallenge{
		Nonce: bytes.Repeat([]byte{byte(sequence)}, ticket.EdgeChallengeNonceSize), EdgeId: gateway.edgeID, EdgeBootId: gateway.edgeBoot,
		StreamId: fmt.Sprintf("enginehost-stream-%d", sequence), IssuedAt: timestamppb.New(issuedAt), ExpiresAt: timestamppb.New(issuedAt.Add(ticket.EdgeChallengeLifetime)),
		Target: cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY,
	}
	if err := stream.Send(&cloudv1.EdgeSignal{
		ProtocolVersion: cloudprotocol.ClientGatewayVersion, MessageId: fmt.Sprintf("challenge-%d", sequence), SenderId: gateway.edgeID,
		BootId: gateway.edgeBoot, ConnectionId: challenge.GetStreamId(), StreamSeq: 1, SentAt: timestamppb.New(issuedAt),
		Payload: &cloudv1.EdgeSignal_Challenge{Challenge: challenge},
	}); err != nil {
		return err
	}
	hello, err := stream.Recv()
	if err != nil {
		return err
	}
	if err := ticket.VerifyClientHelloProof(hello.GetHello().GetClientPublicKey(), hello.GetHello().GetClientProof(), challenge, hello, issuedAt); err != nil {
		return fmt.Errorf("verify enginehost ClientHello proof: %w", err)
	}
	gateway.mu.Lock()
	gateway.captures = append(gateway.captures, cloudBootCapture{challenge: proto.Clone(challenge).(*cloudv1.EdgeChallenge), hello: proto.Clone(hello).(*cloudv1.ClientSignal)})
	gateway.mu.Unlock()
	sessionID := hello.GetConnectionId()
	generation := hello.GetHello().GetAttemptGeneration()
	if err := stream.Send(&cloudv1.EdgeSignal{
		ProtocolVersion: cloudprotocol.ClientGatewayVersion, MessageId: fmt.Sprintf("ready-%d", sequence), SenderId: gateway.edgeID, BootId: gateway.edgeBoot,
		ConnectionId: sessionID, StreamSeq: 2, SentAt: timestamppb.Now(), Payload: &cloudv1.EdgeSignal_Ready{Ready: &cloudv1.ClientReady{SessionId: sessionID, Generation: generation}},
	}); err != nil {
		return err
	}
	offer, err := stream.Recv()
	if err != nil {
		return err
	}
	if offer.GetOffer() == nil || offer.GetOffer().GetSessionId() != sessionID {
		return fmt.Errorf("enginehost ClientOffer is invalid")
	}
	return stream.Send(&cloudv1.EdgeSignal{
		ProtocolVersion: cloudprotocol.ClientGatewayVersion, MessageId: fmt.Sprintf("answer-%d", sequence), SenderId: gateway.edgeID, BootId: gateway.edgeBoot,
		ConnectionId: sessionID, StreamSeq: 3, SentAt: timestamppb.Now(), Payload: &cloudv1.EdgeSignal_Answer{Answer: &cloudv1.EdgeAnswer{SessionId: sessionID, AnswerSdp: "enginehost-answer"}},
	})
}

func (gateway *cloudBootGateway) capturesForBoot(bootID string) []cloudBootCapture {
	gateway.mu.Lock()
	defer gateway.mu.Unlock()
	var captures []cloudBootCapture
	for _, capture := range gateway.captures {
		if capture.hello.GetBootId() == bootID {
			captures = append(captures, cloudBootCapture{
				challenge: proto.Clone(capture.challenge).(*cloudv1.EdgeChallenge),
				hello:     proto.Clone(capture.hello).(*cloudv1.ClientSignal),
			})
		}
	}
	return captures
}

type cloudPresenceGateway struct {
	cloudv1.UnimplementedClientGatewayServer
	edgeID   string
	edgeBoot string
	online   bool
	calls    atomic.Uint64
}

func (gateway *cloudPresenceGateway) Connect(stream grpc.BidiStreamingServer[cloudv1.ClientSignal, cloudv1.EdgeSignal]) error {
	sequence := gateway.calls.Add(1)
	issuedAt := time.Now().UTC()
	challenge := &cloudv1.EdgeChallenge{
		Nonce: bytes.Repeat([]byte{byte(sequence)}, ticket.EdgeChallengeNonceSize), EdgeId: gateway.edgeID, EdgeBootId: gateway.edgeBoot,
		StreamId: fmt.Sprintf("%s-stream-%d", gateway.edgeID, sequence), IssuedAt: timestamppb.New(issuedAt), ExpiresAt: timestamppb.New(issuedAt.Add(ticket.EdgeChallengeLifetime)),
		Target: cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY,
	}
	if err := stream.Send(&cloudv1.EdgeSignal{
		ProtocolVersion: cloudprotocol.ClientGatewayVersion, MessageId: fmt.Sprintf("%s-challenge-%d", gateway.edgeID, sequence), SenderId: gateway.edgeID,
		BootId: gateway.edgeBoot, ConnectionId: challenge.GetStreamId(), StreamSeq: 1, SentAt: timestamppb.New(issuedAt),
		Payload: &cloudv1.EdgeSignal_Challenge{Challenge: challenge},
	}); err != nil {
		return err
	}
	hello, err := stream.Recv()
	if err != nil {
		return err
	}
	if !hello.GetHello().GetPresenceProbe() {
		return fmt.Errorf("presence test received non-presence ClientHello")
	}
	if err := ticket.VerifyClientHelloProof(hello.GetHello().GetClientPublicKey(), hello.GetHello().GetClientProof(), challenge, hello, issuedAt); err != nil {
		return fmt.Errorf("verify presence ClientHello proof: %w", err)
	}
	return stream.Send(&cloudv1.EdgeSignal{
		ProtocolVersion: cloudprotocol.ClientGatewayVersion, MessageId: fmt.Sprintf("%s-presence-%d", gateway.edgeID, sequence), SenderId: gateway.edgeID,
		BootId: gateway.edgeBoot, ConnectionId: hello.GetConnectionId(), StreamSeq: 2, SentAt: timestamppb.Now(),
		Payload: &cloudv1.EdgeSignal_Presence{Presence: &cloudv1.DaemonPresence{Online: gateway.online}},
	})
}

type cloudPresenceDirectory struct {
	cloudv1.UnimplementedDirectoryServiceServer
	locator  *cloudv1.EdgeLocator
	begins   atomic.Uint64
	resolves atomic.Uint64
}

func (directory *cloudPresenceDirectory) BeginClientRoute(context.Context, *cloudv1.BeginClientRouteRequest) (*cloudv1.IdentityChallenge, error) {
	sequence := directory.begins.Add(1)
	return &cloudv1.IdentityChallenge{
		ChallengeId: fmt.Sprintf("presence-challenge-%d", sequence),
		Challenge:   bytes.Repeat([]byte{byte(sequence)}, remoteauth.DeviceIdentityChallengeBytes),
		ExpiresAt:   timestamppb.New(time.Now().UTC().Add(time.Minute)),
	}, nil
}

func (directory *cloudPresenceDirectory) ResolveClientRoute(_ context.Context, request *cloudv1.ResolveClientRouteRequest) (*cloudv1.ResolveClientRouteResponse, error) {
	directory.resolves.Add(1)
	if strings.TrimSpace(request.GetRequestId()) == "" || len(request.GetClientProof()) == 0 {
		return nil, fmt.Errorf("presence route resolution proof is incomplete")
	}
	return &cloudv1.ResolveClientRouteResponse{EdgeLocator: proto.Clone(directory.locator).(*cloudv1.EdgeLocator)}, nil
}

func startCloudPresenceGateway(t *testing.T, edgeID string, online bool) (*cloudPresenceGateway, string, string, []byte) {
	t.Helper()
	serverName := edgeID + ".enginehost-edge.test"
	certificate, caPEM := cloudBootCertificate(t, serverName)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	gateway := &cloudPresenceGateway{edgeID: edgeID, edgeBoot: edgeID + "-boot", online: online}
	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{certificate}})))
	cloudv1.RegisterClientGatewayServer(server, gateway)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})
	return gateway, listener.Addr().String(), serverName, caPEM
}

func startCloudPresenceDirectory(t *testing.T, locator *cloudv1.EdgeLocator) (*cloudPresenceDirectory, string, string, []byte) {
	t.Helper()
	const serverName = "presence-controller.enginehost.test"
	certificate, caPEM := cloudBootCertificate(t, serverName)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	directory := &cloudPresenceDirectory{locator: proto.Clone(locator).(*cloudv1.EdgeLocator)}
	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{certificate}})))
	cloudv1.RegisterDirectoryServiceServer(server, directory)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})
	return directory, listener.Addr().String(), serverName, caPEM
}

func startCloudBootGateway(t *testing.T) (*cloudBootGateway, string, string, []byte) {
	t.Helper()
	const serverName = "enginehost-edge.test"
	certificate, caPEM := cloudBootCertificate(t, serverName)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	gateway := &cloudBootGateway{edgeID: "edge-enginehost", edgeBoot: "edge-enginehost-boot"}
	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{certificate}})))
	cloudv1.RegisterClientGatewayServer(server, gateway)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})
	return gateway, listener.Addr().String(), serverName, caPEM
}

func cloudBootCertificate(t *testing.T, serverName string) (tls.Certificate, []byte) {
	t.Helper()
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	rootTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "enginehost test root"}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: serverName}, DNSNames: []string{serverName}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, root, &serverKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(serverKey)
	if err != nil {
		t.Fatal(err)
	}
	rootPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER})
	certificate, err := tls.X509KeyPair(
		append(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER}), rootPEM...),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return certificate, rootPEM
}

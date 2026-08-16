package enginehost

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/client/binding"
	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/bindingpb"
	"github.com/anytty/anytty/proto/remoteauthpb"
	golangssh "golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/proto"
)

func TestEndpointRegistryPersistsAcrossEngineRecreation(t *testing.T) {
	platform := newRegistryPlatform(t)
	first := platform.host()
	if _, err := first.UpsertEndpoint(context.Background(), &bindingpb.EndpointUpsertRequest{Endpoint: testEndpointProto(t, "studio", "daemon-studio", "grant-studio"), MakeDefault: true}); err != nil {
		t.Fatal(err)
	}
	second := platform.host()
	result, err := second.GetEndpointRegistry(context.Background(), &bindingpb.EndpointRegistryGetRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if result.GetRegistry().GetDefaultEndpointId() != "studio" || len(result.GetRegistry().GetEndpoints()) != 1 {
		t.Fatalf("recreated registry = %#v", result.GetRegistry())
	}
}

func TestConnectionPolicyUsesGoRegistryTransactionAndAvailability(t *testing.T) {
	platform := newRegistryPlatform(t)
	host := platform.host()
	if _, err := host.UpsertEndpoint(context.Background(), &bindingpb.EndpointUpsertRequest{Endpoint: testEndpointProto(t, "studio", "daemon-studio", "grant-studio")}); err != nil {
		t.Fatal(err)
	}
	initial, err := host.GetConnectionPolicy(context.Background(), &bindingpb.ConnectionPolicyGetRequest{EndpointId: "studio"})
	if err != nil {
		t.Fatal(err)
	}
	if initial.GetState().GetPolicy().GetRoutePreference() != remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_AUTO {
		t.Fatalf("initial policy = %#v", initial.GetState())
	}
	availability := initial.GetState().GetRoutes()
	if len(availability) != 3 || availability[0].GetRouteKind() != bindingpb.ConnectionRouteKind_CONNECTION_ROUTE_KIND_DIRECT || availability[0].GetReason() != bindingpb.ConnectionPolicyAvailabilityReason_CONNECTION_POLICY_AVAILABILITY_REASON_PLATFORM_UNSUPPORTED ||
		availability[2].GetRouteKind() != bindingpb.ConnectionRouteKind_CONNECTION_ROUTE_KIND_CLOUD || availability[2].GetReason() != bindingpb.ConnectionPolicyAvailabilityReason_CONNECTION_POLICY_AVAILABILITY_REASON_ROUTE_NOT_CONFIGURED {
		t.Fatalf("route availability = %#v", availability)
	}
	if _, err := host.UpsertEndpoint(context.Background(), &bindingpb.EndpointUpsertRequest{Endpoint: testManagedEndpointProto(t, "studio", "daemon-studio", "grant-studio")}); err != nil {
		t.Fatal(err)
	}
	applied, err := host.ApplyConnectionPolicy(context.Background(), &bindingpb.ConnectionPolicyApplyRequest{
		EndpointId: "studio",
		Policy: &bindingpb.ConnectionPolicy{
			RoutePreference: remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_DIRECT,
			CloudRelayMode:  remoteauthpb.ManagedWebRTCRelayMode_MANAGED_WEBRTC_RELAY_MODE_RELAY_ONLY,
			RelayTransport:  remoteauthpb.ManagedWebRTCRelayTransport_MANAGED_WEBRTC_RELAY_TRANSPORT_TCP,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied.GetState().GetPolicy().GetRoutePreference() != remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_DIRECT {
		t.Fatalf("applied state = %#v", applied.GetState())
	}
	registry, err := host.GetEndpointRegistry(context.Background(), &bindingpb.EndpointRegistryGetRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got := registry.GetRegistry().GetEndpoints()[0].GetSelectionPolicy().GetRoutePreference(); got != remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_DIRECT {
		t.Fatalf("persisted route preference = %v", got)
	}
	model, err := endpoint.RegistryFromProto(registry.GetRegistry())
	if err != nil {
		t.Fatal(err)
	}
	cloudRoute := model.Endpoints["studio"].Routes["cloud"]
	if cloudRoute.RelayMode != endpoint.RelayOnly || cloudRoute.RelayTransport != endpoint.RelayTransportTCP {
		t.Fatalf("persisted Cloud policy = %#v", cloudRoute)
	}
	applied, err = host.ApplyConnectionPolicy(context.Background(), &bindingpb.ConnectionPolicyApplyRequest{
		EndpointId: "studio",
		Policy: &bindingpb.ConnectionPolicy{
			RoutePreference: remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD,
			CloudRelayMode:  remoteauthpb.ManagedWebRTCRelayMode_MANAGED_WEBRTC_RELAY_MODE_RELAY_ONLY,
			RelayTransport:  remoteauthpb.ManagedWebRTCRelayTransport_MANAGED_WEBRTC_RELAY_TRANSPORT_TCP,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied.GetState().GetPolicy().GetRoutePreference() != remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD {
		t.Fatalf("applied Cloud state = %#v", applied.GetState())
	}
	registry, err = host.GetEndpointRegistry(context.Background(), &bindingpb.EndpointRegistryGetRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got := registry.GetRegistry().GetEndpoints()[0].GetSelectionPolicy().GetRoutePreference(); got != remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD {
		t.Fatalf("persisted Cloud route preference = %v", got)
	}
}

func TestEndpointRegistryRejectsIdentityReplacementWithoutPublishing(t *testing.T) {
	platform := newRegistryPlatform(t)
	host := platform.host()
	if _, err := host.UpsertEndpoint(context.Background(), &bindingpb.EndpointUpsertRequest{Endpoint: testEndpointProto(t, "studio", "daemon-one", "grant-one")}); err != nil {
		t.Fatal(err)
	}
	if _, err := host.UpsertEndpoint(context.Background(), &bindingpb.EndpointUpsertRequest{Endpoint: testEndpointProto(t, "studio", "daemon-two", "grant-two")}); err == nil {
		t.Fatal("identity replacement unexpectedly succeeded")
	}
	result, err := host.GetEndpointRegistry(context.Background(), &bindingpb.EndpointRegistryGetRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.GetRegistry().GetEndpoints()[0].GetIdentity().GetDeviceId(); got != "daemon-one" {
		t.Fatalf("identity conflict published %q", got)
	}
}

func TestEndpointRegistryConcurrentUpsertsDoNotLoseEndpoints(t *testing.T) {
	platform := newRegistryPlatform(t)
	host := platform.host()
	const count = 32
	var wait sync.WaitGroup
	errors := make(chan error, count)
	for index := 0; index < count; index++ {
		index := index
		wait.Add(1)
		go func() {
			defer wait.Done()
			id := fmt.Sprintf("endpoint-%02d", index)
			_, err := host.UpsertEndpoint(context.Background(), &bindingpb.EndpointUpsertRequest{Endpoint: testEndpointProto(t, id, "daemon-"+id, "grant-"+id)})
			errors <- err
		}()
	}
	wait.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	result, err := host.GetEndpointRegistry(context.Background(), &bindingpb.EndpointRegistryGetRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(result.GetRegistry().GetEndpoints()); got != count {
		t.Fatalf("registry contains %d endpoints, want %d", got, count)
	}
}

func TestEndpointRegistryStoreFailureKeepsPreviousSnapshot(t *testing.T) {
	platform := newRegistryPlatform(t)
	host := platform.host()
	if _, err := host.UpsertEndpoint(context.Background(), &bindingpb.EndpointUpsertRequest{Endpoint: testEndpointProto(t, "one", "daemon-one", "grant-one")}); err != nil {
		t.Fatal(err)
	}
	platform.failNextStore()
	if _, err := host.UpsertEndpoint(context.Background(), &bindingpb.EndpointUpsertRequest{Endpoint: testEndpointProto(t, "two", "daemon-two", "grant-two")}); err == nil {
		t.Fatal("failed platform store unexpectedly succeeded")
	}
	result, err := host.GetEndpointRegistry(context.Background(), &bindingpb.EndpointRegistryGetRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(result.GetRegistry().GetEndpoints()); got != 1 || result.GetRegistry().GetEndpoints()[0].GetEndpointId() != "one" {
		t.Fatalf("failed store published registry %#v", result.GetRegistry())
	}
}

func TestEndpointDeleteCommitsRegistryAndCredentialCleanupTogether(t *testing.T) {
	platform := newRegistryPlatform(t)
	platform.credentials["grant-studio"] = "bound"
	host := platform.host()
	if _, err := host.UpsertEndpoint(context.Background(), &bindingpb.EndpointUpsertRequest{Endpoint: testEndpointProto(t, "studio", "daemon-studio", "grant-studio")}); err != nil {
		t.Fatal(err)
	}
	deleted, err := host.DeleteEndpoint(context.Background(), &bindingpb.EndpointDeleteRequest{EndpointId: "studio"})
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted.GetRegistry().GetEndpoints()) != 0 {
		t.Fatalf("deleted registry = %#v", deleted.GetRegistry())
	}
	platform.mu.Lock()
	_, credentialExists := platform.credentials["grant-studio"]
	platform.mu.Unlock()
	if credentialExists {
		t.Fatal("unreferenced credential was not deleted in registry transaction")
	}
}

func TestPairingCredentialRollbackRestoresPreparedState(t *testing.T) {
	platform := newRegistryPlatform(t)
	host := platform.host()
	platform.credentials["grant-new"] = "bound-new"
	if err := host.rollbackPreparedCredential(context.Background(), &bindingpb.CredentialRecord{
		EndpointId: "new", CredentialRef: "grant-new", NewlyCreated: true,
	}, "bound-new", nil, nil); err != nil {
		t.Fatal(err)
	}
	platform.credentials["grant-existing"] = "bound-new"
	if err := host.rollbackPreparedCredential(context.Background(), &bindingpb.CredentialRecord{
		EndpointId: "existing", CredentialRef: "grant-existing", CapabilityGrant: "bound-old",
	}, "bound-new", nil, nil); err != nil {
		t.Fatal(err)
	}
	platform.mu.Lock()
	defer platform.mu.Unlock()
	if _, exists := platform.credentials["grant-new"]; exists {
		t.Fatal("newly created credential survived rollback")
	}
	if got := platform.credentials["grant-existing"]; got != "bound-old" {
		t.Fatalf("existing credential grant after rollback = %q", got)
	}
}

func TestPairingBindsGrantToExistingShareRoutesAfterAssembly(t *testing.T) {
	platform := newRegistryPlatform(t)
	host := platform.host()
	identity := endpoint.DaemonIdentity{DeviceID: "daemon-shared", DeviceFingerprint: "SHA256:shared"}
	shared := endpoint.Endpoint{
		ID: "shared", Label: "Shared", LabelSource: endpoint.SourceShare, DaemonIdentity: identity,
		ConnectMode: endpoint.ConnectOnDemand, Enabled: true,
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{
			"direct": {ID: "direct", Kind: endpoint.RouteDirectWebRTCTCP, Enabled: true, Source: endpoint.SourceShare, PolicySource: endpoint.SourceShare, SignalingAddresses: []string{"127.0.0.1:41120"}, ICETCPAddresses: []string{"127.0.0.1:41121"}},
			"ssh":    {ID: "ssh", Kind: endpoint.RouteSSHWebRTCTCP, Enabled: true, Source: endpoint.SourceShare, PolicySource: endpoint.SourceShare, Host: "127.0.0.1", User: "anytty", HostKeyFingerprints: []string{"SHA256:test"}, CredentialDescriptor: &endpoint.CredentialDescriptor{DescriptorID: "ssh-key", Kind: endpoint.CredentialSSHPrivateKey}, SSHCredentialRef: "ssh-platform-existing", RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121"},
			"cloud":  {ID: "cloud", Kind: endpoint.RouteManagedWebRTC, Enabled: true, Source: endpoint.SourceShare, PolicySource: endpoint.SourceShare, TargetDeviceID: identity.DeviceID, RelayMode: endpoint.RelayAuto},
		},
	}
	wireShared, err := endpoint.EndpointToProto(shared)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := host.UpsertEndpoint(context.Background(), &bindingpb.EndpointUpsertRequest{Endpoint: wireShared}); err != nil {
		t.Fatal(err)
	}
	candidate := endpoint.EndpointCandidate{Source: endpoint.SourceBootstrap, Identity: identity, Routes: []endpoint.AccessRoute{{
		ID: "direct", Kind: endpoint.RouteDirectWebRTCTCP, Enabled: true, Source: endpoint.SourceBootstrap, PolicySource: endpoint.SourceBootstrap,
		SignalingAddresses: []string{"127.0.0.1:41120"}, ICETCPAddresses: []string{"127.0.0.1:41121"},
	}}}
	paired, _, err := host.commitPairingEndpoint(context.Background(), "shared", candidate, "grant-shared")
	if err != nil {
		t.Fatal(err)
	}
	if len(paired.GetRoutes()) != 3 {
		t.Fatalf("paired routes = %#v", paired.GetRoutes())
	}
	for _, route := range paired.GetRoutes() {
		if route.GetCredentialRef() != "grant-shared" {
			t.Fatalf("route %q credential_ref = %q", route.GetRouteId(), route.GetCredentialRef())
		}
	}
	for _, route := range paired.GetRoutes() {
		if route.GetRouteId() == "ssh" && route.GetSshWebrtcTcp().GetSshCredentialRef() != "ssh-platform-existing" {
			t.Fatalf("SSH signer ref was replaced: %#v", route)
		}
	}
}

func TestEndpointSharePreviewCommitsAtomicallyAndTokenIsSingleUse(t *testing.T) {
	platform := newRegistryPlatform(t)
	host := platform.host()
	now := time.Now().UTC()
	bundle, err := endpoint.NewClientEndpointShareBundle(endpoint.Endpoint{
		ID: "source", Label: "Shared Studio", LabelSource: endpoint.SourceUser,
		DaemonIdentity: endpoint.DaemonIdentity{DeviceID: "daemon-shared", DeviceFingerprint: "SHA256:shared"},
		ConnectMode:    endpoint.ConnectOnDemand, Enabled: true,
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{
			"direct": {ID: "direct", Kind: endpoint.RouteDirectWebRTCTCP, Enabled: true, Source: endpoint.SourceManual, PolicySource: endpoint.SourceManual, SignalingAddresses: []string{"shared:41120"}, ICETCPAddresses: []string{"shared:41121"}},
		},
	}, "share-binding", now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	host.options.Now = func() time.Time { return now }
	host.options.ShareReceive = func(context.Context, *remoteauthpb.ShareSessionOffer) (*remoteauthpb.ClientEndpointShareBundleV1, error) {
		return proto.Clone(bundle).(*remoteauthpb.ClientEndpointShareBundleV1), nil
	}
	host.pendingShares = make(map[string]*remoteauthpb.ClientEndpointShareBundleV1)
	offer := &remoteauthpb.ShareSessionOffer{
		SchemaVersion: endpoint.ShareSessionOfferVersion, TransferId: bundle.GetTransferId(), ListenerAddresses: []string{"127.0.0.1:41130"},
		EphemeralCertificateSha256: "sha256:" + base64.RawURLEncoding.EncodeToString(make([]byte, 32)), OneTimeSessionSecret: make([]byte, 32), ExpiresAtUnixNano: now.Add(time.Minute).UnixNano(),
	}
	offerPayload, err := endpoint.MarshalShareSessionOffer(offer)
	if err != nil {
		t.Fatal(err)
	}
	received, err := host.ReceiveEndpointShare(context.Background(), &bindingpb.EndpointShareReceiveRequest{PortableOffer: endpointShareURIPrefix + base64.RawURLEncoding.EncodeToString(offerPayload)})
	if err != nil {
		t.Fatal(err)
	}
	preview := received.GetPreview()
	if preview.GetImportToken() == "" || len(preview.GetRouteDiffs()) != 1 || preview.GetRouteDiffs()[0].GetAction() != "add" {
		t.Fatalf("share preview=%#v", preview)
	}
	committed, err := host.CommitEndpointShare(context.Background(), &bindingpb.EndpointShareCommitRequest{ImportToken: preview.GetImportToken()})
	if err != nil {
		t.Fatal(err)
	}
	if !committed.GetAuthorizationRequired() || committed.GetEndpoint().GetRoutes()[0].GetCredentialRef() != "" {
		t.Fatalf("share commit was not config-only: %#v", committed)
	}
	if _, err := host.CommitEndpointShare(context.Background(), &bindingpb.EndpointShareCommitRequest{ImportToken: preview.GetImportToken()}); err == nil {
		t.Fatal("share import token unexpectedly committed twice")
	}
	retryPreview, err := host.ReceiveEndpointShare(context.Background(), &bindingpb.EndpointShareReceiveRequest{PortableOffer: endpointShareURIPrefix + base64.RawURLEncoding.EncodeToString(offerPayload)})
	if err != nil {
		t.Fatal(err)
	}
	platform.failNextStore()
	retryToken := retryPreview.GetPreview().GetImportToken()
	if _, err := host.CommitEndpointShare(context.Background(), &bindingpb.EndpointShareCommitRequest{ImportToken: retryToken}); err == nil {
		t.Fatal("share commit unexpectedly ignored platform store failure")
	}
	if _, err := host.CommitEndpointShare(context.Background(), &bindingpb.EndpointShareCommitRequest{ImportToken: retryToken}); err != nil {
		t.Fatalf("share token did not survive unpublished store failure: %v", err)
	}
}

func TestSSHCredentialProvisionCommitsRouteAndRollsBackNewKeyOnStoreFailure(t *testing.T) {
	platform := newRegistryPlatform(t)
	host := platform.host()
	if _, err := host.UpsertEndpoint(context.Background(), &bindingpb.EndpointUpsertRequest{Endpoint: testSSHEndpointProto(t, "studio")}); err != nil {
		t.Fatal(err)
	}
	result, err := host.ProvisionSSHCredential(context.Background(), &bindingpb.SSHCredentialProvisionRequest{EndpointId: "studio", RouteId: "ssh"})
	if err != nil {
		t.Fatal(err)
	}
	route := result.GetEndpoint().GetRoutes()[0].GetSshWebrtcTcp()
	if route.GetSshCredentialRef() != result.GetCredentialRef() || !strings.HasPrefix(result.GetCredentialRef(), platformSSHCredentialPrefix) {
		t.Fatalf("provisioned SSH route = %#v result=%#v", route, result)
	}
	if result.GetEndpoint().GetRoutes()[0].GetCredentialRef() != "" {
		t.Fatalf("SSH signer provisioning unexpectedly authorized config-only endpoint: %#v", result.GetEndpoint())
	}
	publicKey, _, _, _, err := golangssh.ParseAuthorizedKey([]byte(result.GetAuthorizedKey()))
	if err != nil || golangssh.FingerprintSHA256(publicKey) != result.GetKeyFingerprint() {
		t.Fatalf("provisioned authorized key is invalid: key=%v err=%v result=%#v", publicKey, err, result)
	}

	if _, err := host.UpsertEndpoint(context.Background(), &bindingpb.EndpointUpsertRequest{Endpoint: testSSHEndpointProto(t, "backup")}); err != nil {
		t.Fatal(err)
	}
	platform.failNextStore()
	if _, err := host.ProvisionSSHCredential(context.Background(), &bindingpb.SSHCredentialProvisionRequest{EndpointId: "backup", RouteId: "ssh"}); err == nil {
		t.Fatal("failed registry store unexpectedly kept provisioned SSH credential")
	}
	failedRef := platformSSHCredentialRef("backup", "ssh")
	platform.mu.Lock()
	_, keySurvived := platform.sshKeys[failedRef]
	platform.mu.Unlock()
	if keySurvived {
		t.Fatal("new platform SSH key survived failed registry transaction")
	}
	registry, err := host.GetEndpointRegistry(context.Background(), &bindingpb.EndpointRegistryGetRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range registry.GetRegistry().GetEndpoints() {
		if value.GetEndpointId() == "backup" && value.GetRoutes()[0].GetSshWebrtcTcp().GetSshCredentialRef() != "" {
			t.Fatalf("failed provision published SSH credential ref: %#v", value)
		}
	}
}

func TestEndpointUpsertRemovingRouteDeletesUnreferencedCredentials(t *testing.T) {
	platform := newRegistryPlatform(t)
	host := platform.host()
	configured := testSSHEndpointProto(t, "studio")
	configured.Routes[0].CredentialRef = "credential:studio"
	configured.Routes[0].GetSshWebrtcTcp().SshCredentialRef = platformSSHCredentialRef("studio", "ssh")
	platform.credentials["credential:studio"] = "grant"
	platform.sshKeys[platformSSHCredentialRef("studio", "ssh")] = []byte("private-key")
	if _, err := host.UpsertEndpoint(context.Background(), &bindingpb.EndpointUpsertRequest{Endpoint: configured}); err != nil {
		t.Fatal(err)
	}

	replacement := testEndpointProto(t, "studio", configured.GetIdentity().GetDeviceId(), "credential:replacement")
	replacement.Identity = proto.Clone(configured.GetIdentity()).(*remoteauthpb.EndpointDaemonIdentity)
	if _, err := host.UpsertEndpoint(context.Background(), &bindingpb.EndpointUpsertRequest{Endpoint: replacement}); err != nil {
		t.Fatal(err)
	}
	platform.mu.Lock()
	defer platform.mu.Unlock()
	if _, ok := platform.credentials["credential:studio"]; ok {
		t.Fatal("removed Route capability credential survived registry transaction")
	}
	if _, ok := platform.sshKeys[platformSSHCredentialRef("studio", "ssh")]; ok {
		t.Fatal("removed SSH Route credential survived registry transaction")
	}
}

func testEndpointProto(t *testing.T, id, deviceID, credentialRef string) *remoteauthpb.EndpointConfigV1 {
	t.Helper()
	digest := sha256.Sum256([]byte(deviceID))
	model := endpoint.Endpoint{
		ID: endpoint.EndpointID(id), Label: id, LabelSource: endpoint.SourceUser,
		DaemonIdentity: endpoint.DaemonIdentity{DeviceID: deviceID, DeviceFingerprint: "ed25519-sha256:" + base64.RawURLEncoding.EncodeToString(digest[:])},
		ConnectMode:    endpoint.ConnectOnDemand, Enabled: true,
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{
			"direct": {
				ID: "direct", Kind: endpoint.RouteDirectWebRTCTCP, Enabled: true, Source: endpoint.SourceBootstrap, PolicySource: endpoint.SourceBootstrap,
				CredentialRef: credentialRef, SignalingAddresses: []string{"127.0.0.1:41120"}, ICETCPAddresses: []string{"127.0.0.1:41121"},
			},
		},
	}
	wire, err := endpoint.EndpointToProto(model)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func testManagedEndpointProto(t *testing.T, id, deviceID, credentialRef string) *remoteauthpb.EndpointConfigV1 {
	t.Helper()
	wire := testEndpointProto(t, id, deviceID, credentialRef)
	model, err := endpoint.EndpointFromProto(wire)
	if err != nil {
		t.Fatal(err)
	}
	model.Routes["cloud"] = endpoint.AccessRoute{
		ID: "cloud", Kind: endpoint.RouteManagedWebRTC, Enabled: true,
		Source: endpoint.SourceCloud, PolicySource: endpoint.SourceBootstrap,
		TargetDeviceID: deviceID, RelayMode: endpoint.RelayAuto, RelayTransport: endpoint.RelayTransportAuto,
	}
	wire, err = endpoint.EndpointToProto(model)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func testSSHEndpointProto(t *testing.T, id string) *remoteauthpb.EndpointConfigV1 {
	t.Helper()
	digest := sha256.Sum256([]byte(id))
	model := endpoint.Endpoint{
		ID: endpoint.EndpointID(id), Label: id, LabelSource: endpoint.SourceUser,
		DaemonIdentity: endpoint.DaemonIdentity{DeviceID: "daemon-" + id, DeviceFingerprint: "ed25519-sha256:" + base64.RawURLEncoding.EncodeToString(digest[:])},
		ConnectMode:    endpoint.ConnectOnDemand, Enabled: true,
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{
			"ssh": {
				ID: "ssh", Kind: endpoint.RouteSSHWebRTCTCP, Enabled: true, Source: endpoint.SourceShare, PolicySource: endpoint.SourceShare,
				Host: "127.0.0.1", User: "anytty", HostKeyFingerprints: []string{"SHA256:test"},
				CredentialDescriptor:   &endpoint.CredentialDescriptor{DescriptorID: "ssh-key", Kind: endpoint.CredentialSSHPrivateKey},
				RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121",
			},
		},
	}
	wire, err := endpoint.EndpointToProto(model)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

type registryPlatform struct {
	t           *testing.T
	mu          sync.Mutex
	registry    []byte
	credentials map[string]string
	sshKeys     map[string][]byte
	failStore   bool
}

func newRegistryPlatform(t *testing.T) *registryPlatform {
	return &registryPlatform{t: t, credentials: make(map[string]string), sshKeys: make(map[string][]byte)}
}

func (platform *registryPlatform) host() *Host {
	broker := binding.NewPlatformBroker()
	platform.t.Cleanup(func() { _ = broker.Close() })
	go platform.pump(broker)
	return &Host{options: Options{Broker: broker}}
}

func (platform *registryPlatform) failNextStore() {
	platform.mu.Lock()
	platform.failStore = true
	platform.mu.Unlock()
}

func (platform *registryPlatform) pump(broker *binding.PlatformBroker) {
	for {
		payload, err := broker.NextRequest(context.Background())
		if err != nil {
			return
		}
		request := &bindingpb.PlatformRequest{}
		if err := proto.Unmarshal(payload, request); err != nil {
			platform.t.Errorf("decode platform request: %v", err)
			return
		}
		response := &bindingpb.PlatformResponse{RequestId: request.GetRequestId()}
		platform.mu.Lock()
		switch value := request.GetRequest().(type) {
		case *bindingpb.PlatformRequest_EndpointRegistryLoad:
			response.Response = &bindingpb.PlatformResponse_EndpointRegistry{EndpointRegistry: &bindingpb.EndpointRegistryLoaded{RegistryProto: append([]byte(nil), platform.registry...)}}
		case *bindingpb.PlatformRequest_EndpointRegistryStore:
			if platform.failStore {
				platform.failStore = false
				response.Error = &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE, Message: "registry store failed"}
				break
			}
			platform.registry = append([]byte(nil), value.EndpointRegistryStore.GetRegistryProto()...)
			for _, ref := range value.EndpointRegistryStore.GetDeleteCredentialRefs() {
				if strings.HasPrefix(ref, platformSSHCredentialPrefix) {
					delete(platform.sshKeys, ref)
				} else {
					delete(platform.credentials, ref)
				}
			}
		case *bindingpb.PlatformRequest_CredentialDelete:
			delete(platform.credentials, value.CredentialDelete.GetCredentialRef())
		case *bindingpb.PlatformRequest_CredentialBind:
			platform.credentials[value.CredentialBind.GetCredentialRef()] = value.CredentialBind.GetCapabilityGrant()
			response.Response = &bindingpb.PlatformResponse_Credential{Credential: &bindingpb.CredentialRecord{
				EndpointId: value.CredentialBind.GetEndpointId(), CredentialRef: value.CredentialBind.GetCredentialRef(), CapabilityGrant: value.CredentialBind.GetCapabilityGrant(),
			}}
		case *bindingpb.PlatformRequest_SshCredentialLookup:
			ref := value.SshCredentialLookup.GetCredentialRef()
			publicKey := platform.sshKeys[ref]
			newlyCreated := false
			if len(publicKey) == 0 && value.SshCredentialLookup.GetCreateIfMissing() {
				privateKey, keyErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				if keyErr != nil {
					platform.t.Errorf("generate test SSH key: %v", keyErr)
					platform.mu.Unlock()
					return
				}
				publicKey, keyErr = x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
				if keyErr != nil {
					platform.t.Errorf("marshal test SSH key: %v", keyErr)
					platform.mu.Unlock()
					return
				}
				platform.sshKeys[ref] = publicKey
				newlyCreated = true
			}
			if len(publicKey) == 0 {
				response.Error = &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_UNAUTHORIZED, Message: "SSH credential is missing"}
				break
			}
			response.Response = &bindingpb.PlatformResponse_SshCredential{SshCredential: &bindingpb.SSHCredentialRecord{
				CredentialRef: ref, PublicKeyPkix: append([]byte(nil), publicKey...), NewlyCreated: newlyCreated,
			}}
		case *bindingpb.PlatformRequest_SshCredentialDelete:
			delete(platform.sshKeys, value.SshCredentialDelete.GetCredentialRef())
		default:
			response.Error = &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST, Message: "unexpected platform request"}
		}
		platform.mu.Unlock()
		encoded, err := proto.Marshal(response)
		if err != nil {
			platform.t.Errorf("encode platform response: %v", err)
			return
		}
		if err := broker.Complete(encoded); err != nil {
			return
		}
	}
}

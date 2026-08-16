package endpoint

import (
	"testing"

	"github.com/anytty/anytty/proto/remoteauthpb"
	"google.golang.org/protobuf/proto"
)

func TestEndpointRegistryProtoRoundTripCoversAllRouteKinds(t *testing.T) {
	identity := DaemonIdentity{DeviceID: "device-studio", DeviceFingerprint: "SHA256:studio"}
	priority := func(value int) *int { return &value }
	registry := Registry{
		Version: RegistryVersion,
		Default: "studio",
		Endpoints: map[EndpointID]Endpoint{"studio": {
			ID: "studio", Label: "Studio", LabelSource: SourceUser, DaemonIdentity: identity, Platform: "darwin",
			ConnectMode: ConnectOnDemand, Enabled: true,
			SelectionPolicy: SelectionPolicy{RoutePreference: RoutePreferenceManagedCloud},
			Routes: map[RouteID]AccessRoute{
				"local": {
					ID: "local", Kind: RouteLocalUnix, Enabled: true, Priority: priority(10), Source: SourceLocal, PolicySource: SourceUser, Socket: "auto",
				},
				"direct": {
					ID: "direct", Kind: RouteDirectWebRTCTCP, Enabled: true, Priority: priority(20), CredentialRef: "grant:studio",
					Source: SourceBootstrap, PolicySource: SourceUser, SignalingAddresses: []string{"studio.local:41120"},
					ICETCPAddresses: []string{"studio.local:41121"}, AdvertisedAddresses: []string{"203.0.113.10:41121"}, ServerName: "studio.local",
				},
				"ssh": {
					ID: "ssh", Kind: RouteSSHWebRTCTCP, Enabled: true, Priority: priority(30), CredentialRef: "grant:ssh-studio", SSHCredentialRef: "ssh:studio",
					Source: SourceManual, PolicySource: SourceUser, Host: "studio-ssh", Port: 22, User: "build",
					HostKeyFingerprints: []string{"SHA256:ssh-host"}, RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121",
					CredentialDescriptor: &CredentialDescriptor{DescriptorID: "ssh-key", Kind: CredentialSSHPrivateKey, Exportable: true},
				},
				"cloud": {
					ID: "cloud", Kind: RouteManagedWebRTC, Enabled: true, Priority: priority(40), CredentialRef: "grant:studio",
					Source: SourceCloud, PolicySource: SourceUser, TargetDeviceID: identity.DeviceID, AccountProfileRef: "default", RelayMode: RelayOnly, RelayTransport: RelayTransportTCP,
				},
			},
		}},
	}
	wire, err := RegistryToProto(registry)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := (proto.MarshalOptions{Deterministic: true}).Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &remoteauthpb.EndpointRegistryV1{}
	if err := proto.Unmarshal(payload, decoded); err != nil {
		t.Fatal(err)
	}
	roundTripped, err := RegistryFromProto(decoded)
	if err != nil {
		t.Fatal(err)
	}
	wireAgain, err := RegistryToProto(roundTripped)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(wire, wireAgain) {
		t.Fatalf("endpoint registry proto round-trip mismatch:\nfirst=%v\nsecond=%v", wire, wireAgain)
	}
	if roundTripped.Endpoints["studio"].Platform != "darwin" {
		t.Fatalf("platform was not preserved: %#v", roundTripped.Endpoints["studio"])
	}
}

func TestEndpointRegistryProtoRejectsUnknownOldAndDuplicateContracts(t *testing.T) {
	valid := &remoteauthpb.EndpointRegistryV1{
		SchemaVersion:     EndpointRegistryContractVersion,
		DefaultEndpointId: "local",
		Endpoints: []*remoteauthpb.EndpointConfigV1{{
			SchemaVersion: EndpointConfigVersion, EndpointId: "local", Label: "Local",
			LabelSource: remoteauthpb.EndpointSource_ENDPOINT_SOURCE_LOCAL,
			ConnectMode: remoteauthpb.EndpointConnectMode_ENDPOINT_CONNECT_MODE_AUTO, Enabled: true,
			Routes: []*remoteauthpb.EndpointRouteConfigV1{{
				SchemaVersion: RouteConfigVersion, RouteId: "local", Enabled: true,
				Source: remoteauthpb.EndpointSource_ENDPOINT_SOURCE_LOCAL, PolicySource: remoteauthpb.EndpointSource_ENDPOINT_SOURCE_LOCAL,
				Route: &remoteauthpb.EndpointRouteConfigV1_LocalUnix{LocalUnix: &remoteauthpb.LocalUnixRouteConfig{Socket: "auto"}},
			}},
		}},
	}
	withUnknown := proto.Clone(valid).(*remoteauthpb.EndpointRegistryV1)
	withUnknown.Endpoints[0].Routes[0].GetLocalUnix().ProtoReflect().SetUnknown([]byte{0xa0, 0x06, 0x01})
	if _, err := RegistryFromProto(withUnknown); !IsCode(err, ErrorUnsupportedVersion) {
		t.Fatalf("nested unknown field error = %v", err)
	}
	oldVersion := proto.Clone(valid).(*remoteauthpb.EndpointRegistryV1)
	oldVersion.SchemaVersion = 0
	if _, err := RegistryFromProto(oldVersion); !IsCode(err, ErrorUnsupportedVersion) {
		t.Fatalf("old registry version error = %v", err)
	}
	oldRouteVersion := proto.Clone(valid).(*remoteauthpb.EndpointRegistryV1)
	oldRouteVersion.Endpoints[0].Routes[0].SchemaVersion = 0
	if _, err := RegistryFromProto(oldRouteVersion); !IsCode(err, ErrorUnsupportedVersion) {
		t.Fatalf("old route version error = %v", err)
	}
	duplicate := proto.Clone(valid).(*remoteauthpb.EndpointRegistryV1)
	duplicate.Endpoints = append(duplicate.Endpoints, proto.Clone(duplicate.Endpoints[0]).(*remoteauthpb.EndpointConfigV1))
	if _, err := RegistryFromProto(duplicate); !IsCode(err, ErrorConfig) {
		t.Fatalf("duplicate endpoint error = %v", err)
	}
}

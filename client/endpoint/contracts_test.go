package endpoint

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/remoteauthpb"
	"google.golang.org/protobuf/proto"
)

func TestPortableContractsUseDeterministicStrictProtobuf(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	identity := &remoteauthpb.EndpointDaemonIdentity{
		DeviceId: "device-studio", DevicePublicKey: publicKey, DeviceFingerprint: daemonPublicKeyFingerprint(publicKey),
	}
	ticket := &remoteauthpb.PairingTicketDescriptor{
		TicketId: "ticket-1", ScopeCeiling: []string{"terminal"}, ExpiresAtUnixNano: now.Add(time.Minute).UnixNano(),
		Nonce: bytes.Repeat([]byte{1}, 16), MaxRedemptions: 1, IssuedAtUnixNano: now.UnixNano(), GrantLifetimeSeconds: 3600,
	}
	ticketSigningBytes, err := PairingTicketSigningBytes(identity, ticket)
	if err != nil {
		t.Fatal(err)
	}
	ticket.Signature = ed25519.Sign(privateKey, ticketSigningBytes)
	bundle := &remoteauthpb.EndpointBootstrapBundleV2{
		SchemaVersion:     EndpointBootstrapBundleVersion,
		BundleId:          "bundle-1",
		Identity:          identity,
		Routes:            []*remoteauthpb.EndpointRouteConfigV1{testDirectRoute("lan", "studio.local:41120")},
		Authorization:     &remoteauthpb.EndpointAuthorizationBootstrap{Payload: &remoteauthpb.EndpointAuthorizationBootstrap_PairingTicket{PairingTicket: ticket}},
		IssuedAtUnixNano:  now.UnixNano(),
		ExpiresAtUnixNano: now.Add(time.Minute).UnixNano(),
	}
	bundleSigningBytes, err := EndpointBootstrapSigningBytes(bundle)
	if err != nil {
		t.Fatal(err)
	}
	bundle.BundleSignature = ed25519.Sign(privateKey, bundleSigningBytes)
	first, err := MarshalEndpointBootstrapBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	second, err := MarshalEndpointBootstrapBundle(bundle)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("deterministic bootstrap encoding mismatch: err=%v", err)
	}
	parsed, err := ParseEndpointBootstrapBundle(first)
	if err != nil || parsed.GetBundleId() != bundle.GetBundleId() {
		t.Fatalf("parse bootstrap: bundle=%v err=%v", parsed, err)
	}

	withNestedUnknown := proto.Clone(bundle).(*remoteauthpb.EndpointBootstrapBundleV2)
	withNestedUnknown.Routes[0].ProtoReflect().SetUnknown([]byte{0xa0, 0x06, 0x01})
	unknownPayload, err := proto.Marshal(withNestedUnknown)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseEndpointBootstrapBundle(unknownPayload); !IsCode(err, ErrorConfig) {
		t.Fatalf("nested unknown field error = %v", err)
	}
	if _, err := ParseEndpointBootstrapBundle(bytes.Repeat([]byte{1}, MaxPortableContractBytes+1)); !IsCode(err, ErrorSizeLimit) {
		t.Fatalf("oversize portable contract error = %v", err)
	}

	forbiddenPolicy := proto.Clone(bundle).(*remoteauthpb.EndpointBootstrapBundleV2)
	forbiddenPolicy.Routes[0].Priority = proto.Int32(10)
	if _, err := MarshalEndpointBootstrapBundle(forbiddenPolicy); !IsCode(err, ErrorConfig) {
		t.Fatalf("bootstrap priority error = %v", err)
	}
	tampered := proto.Clone(bundle).(*remoteauthpb.EndpointBootstrapBundleV2)
	tampered.Routes[0].GetDirectWebrtcTcp().SignalingAddresses[0] = "attacker.example:41120"
	if _, err := MarshalEndpointBootstrapBundle(tampered); !IsCode(err, ErrorIdentityConflict) {
		t.Fatalf("tampered bootstrap signature error = %v", err)
	}
	tamperedTicket := proto.Clone(bundle).(*remoteauthpb.EndpointBootstrapBundleV2)
	tamperedTicket.GetAuthorization().GetPairingTicket().Nonce[0] ^= 0xff
	tamperedTicket.BundleSignature = nil
	if _, err := EndpointBootstrapSigningBytes(tamperedTicket); !IsCode(err, ErrorIdentityConflict) {
		t.Fatalf("tampered ticket signature error = %v", err)
	}
	expired := proto.Clone(bundle).(*remoteauthpb.EndpointBootstrapBundleV2)
	expired.IssuedAtUnixNano = now.Add(-2 * time.Minute).UnixNano()
	expired.ExpiresAtUnixNano = now.Add(-time.Second).UnixNano()
	expired.Authorization.GetPairingTicket().IssuedAtUnixNano = expired.IssuedAtUnixNano
	expired.Authorization.GetPairingTicket().ExpiresAtUnixNano = expired.ExpiresAtUnixNano
	expired.Authorization.GetPairingTicket().Signature = nil
	ticketBytes, err := PairingTicketSigningBytes(expired.Identity, expired.Authorization.GetPairingTicket())
	if err != nil {
		t.Fatal(err)
	}
	expired.Authorization.GetPairingTicket().Signature = ed25519.Sign(privateKey, ticketBytes)
	expired.BundleSignature = nil
	expiredBytes, err := EndpointBootstrapSigningBytes(expired)
	if err != nil {
		t.Fatal(err)
	}
	expired.BundleSignature = ed25519.Sign(privateKey, expiredBytes)
	payload, err := MarshalEndpointBootstrapBundle(expired)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseEndpointBootstrapBundleAt(payload, now); !IsCode(err, ErrorConfig) {
		t.Fatalf("expired bootstrap error = %v", err)
	}
	opaqueBoundGrant := proto.Clone(bundle).(*remoteauthpb.EndpointBootstrapBundleV2)
	opaqueBoundGrant.Authorization = &remoteauthpb.EndpointAuthorizationBootstrap{Payload: &remoteauthpb.EndpointAuthorizationBootstrap_BoundGrant{BoundGrant: []byte("legacy-bearer")}}
	if _, err := MarshalEndpointBootstrapBundle(opaqueBoundGrant); !IsCode(err, ErrorAuthorizationRequired) {
		t.Fatalf("unverified bootstrap bound grant error = %v", err)
	}
}

func TestShareBundleAndOfferRejectNonPortableOrSecretBearingFields(t *testing.T) {
	now := time.Now().UTC()
	identity := &remoteauthpb.EndpointDaemonIdentity{DeviceId: "device-studio", DeviceFingerprint: "SHA256:studio"}
	priority := int32(20)
	share := &remoteauthpb.ClientEndpointShareBundleV1{
		SchemaVersion: ClientEndpointShareBundleVersion, TransferId: "transfer-1", Identity: identity, SuggestedLabel: "Studio",
		Routes: []*remoteauthpb.EndpointRouteConfigV1{{
			SchemaVersion: RouteConfigVersion, RouteId: "ssh", Enabled: false, ManualOnly: true, Priority: &priority,
			Route: &remoteauthpb.EndpointRouteConfigV1_SshWebrtcTcp{SshWebrtcTcp: &remoteauthpb.SSHWebRTCTCPRouteConfig{
				Host: "studio", Port: 22, RemoteSignalingAddress: "127.0.0.1:41120", RemoteIceTcpAddress: "127.0.0.1:41121",
				CredentialDescriptor: &remoteauthpb.EndpointCredentialDescriptor{DescriptorId: "ssh-key", Kind: remoteauthpb.EndpointCredentialKind_ENDPOINT_CREDENTIAL_KIND_SSH_AGENT},
			}},
		}},
		ConnectMode:     remoteauthpb.EndpointConnectMode_ENDPOINT_CONNECT_MODE_ON_DEMAND,
		SelectionPolicy: &remoteauthpb.EndpointSelectionPolicy{HedgeDelayConfigured: true, HedgeDelayMillis: 250},
		CredentialDescriptors: []*remoteauthpb.EndpointCredentialDescriptor{{
			DescriptorId: "ssh-key", Kind: remoteauthpb.EndpointCredentialKind_ENDPOINT_CREDENTIAL_KIND_SSH_AGENT,
		}},
		IssuedAtUnixNano: now.UnixNano(), ExpiresAtUnixNano: now.Add(time.Minute).UnixNano(),
	}
	payload, err := MarshalClientEndpointShareBundle(share)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseClientEndpointShareBundle(payload)
	if err != nil || parsed.GetRoutes()[0].GetEnabled() {
		t.Fatalf("parse share bundle: parsed=%v err=%v", parsed, err)
	}

	withCredentialRef := proto.Clone(share).(*remoteauthpb.ClientEndpointShareBundleV1)
	withCredentialRef.Routes[0].CredentialRef = "source-secret-ref"
	if _, err := MarshalClientEndpointShareBundle(withCredentialRef); !IsCode(err, ErrorConfig) {
		t.Fatalf("source credential ref error = %v", err)
	}
	whitespaceCredentialRef := proto.Clone(share).(*remoteauthpb.ClientEndpointShareBundleV1)
	whitespaceCredentialRef.Routes[0].CredentialRef = " "
	if _, err := MarshalClientEndpointShareBundle(whitespaceCredentialRef); !IsCode(err, ErrorConfig) {
		t.Fatalf("whitespace source credential ref error = %v", err)
	}
	nonCanonicalHost := proto.Clone(share).(*remoteauthpb.ClientEndpointShareBundleV1)
	nonCanonicalHost.Routes[0].GetSshWebrtcTcp().Host = " studio"
	if _, err := MarshalClientEndpointShareBundle(nonCanonicalHost); !IsCode(err, ErrorConfig) {
		t.Fatalf("non-canonical portable host error = %v", err)
	}
	controlProxyJump := proto.Clone(share).(*remoteauthpb.ClientEndpointShareBundleV1)
	controlProxyJump.Routes[0].GetSshWebrtcTcp().ProxyJump = "bad\njump"
	if _, err := MarshalClientEndpointShareBundle(controlProxyJump); !IsCode(err, ErrorConfig) {
		t.Fatalf("portable proxy jump control character error = %v", err)
	}
	unknownManagedRelay := proto.Clone(share).(*remoteauthpb.ClientEndpointShareBundleV1)
	unknownManagedRelay.Routes[0] = &remoteauthpb.EndpointRouteConfigV1{
		SchemaVersion: RouteConfigVersion, RouteId: "cloud", Enabled: true,
		Route: &remoteauthpb.EndpointRouteConfigV1_ManagedWebrtc{ManagedWebrtc: &remoteauthpb.ManagedWebRTCRouteConfig{
			TargetDeviceId: identity.GetDeviceId(), RelayMode: remoteauthpb.ManagedWebRTCRelayMode(99),
		}},
	}
	if _, err := MarshalClientEndpointShareBundle(unknownManagedRelay); !IsCode(err, ErrorConfig) {
		t.Fatalf("unknown managed relay mode error = %v", err)
	}
	duplicateAddress := proto.Clone(share).(*remoteauthpb.ClientEndpointShareBundleV1)
	duplicateAddress.Routes[0] = testDirectRoute("lan", "studio:41120", "studio:41120")
	if _, err := MarshalClientEndpointShareBundle(duplicateAddress); !IsCode(err, ErrorConfig) {
		t.Fatalf("duplicate portable address error = %v", err)
	}
	withLocal := proto.Clone(share).(*remoteauthpb.ClientEndpointShareBundleV1)
	withLocal.Routes[0] = &remoteauthpb.EndpointRouteConfigV1{
		SchemaVersion: RouteConfigVersion, RouteId: "local", Enabled: true,
		Route: &remoteauthpb.EndpointRouteConfigV1_LocalUnix{LocalUnix: &remoteauthpb.LocalUnixRouteConfig{Socket: "auto"}},
	}
	if _, err := MarshalClientEndpointShareBundle(withLocal); !IsCode(err, ErrorConfig) {
		t.Fatalf("local share route error = %v", err)
	}
	exportableAgent := proto.Clone(share).(*remoteauthpb.ClientEndpointShareBundleV1)
	exportableAgent.CredentialDescriptors[0].Exportable = true
	if _, err := MarshalClientEndpointShareBundle(exportableAgent); !IsCode(err, ErrorConfig) {
		t.Fatalf("exportable agent descriptor error = %v", err)
	}
	duplicateDescriptor := proto.Clone(share).(*remoteauthpb.ClientEndpointShareBundleV1)
	duplicateDescriptor.CredentialDescriptors = append(duplicateDescriptor.CredentialDescriptors, proto.Clone(duplicateDescriptor.CredentialDescriptors[0]).(*remoteauthpb.EndpointCredentialDescriptor))
	if _, err := MarshalClientEndpointShareBundle(duplicateDescriptor); !IsCode(err, ErrorConfig) {
		t.Fatalf("duplicate credential descriptor error = %v", err)
	}
	claimedSource := proto.Clone(share).(*remoteauthpb.ClientEndpointShareBundleV1)
	claimedSource.Routes[0].Source = remoteauthpb.EndpointSource_ENDPOINT_SOURCE_USER
	if _, err := MarshalClientEndpointShareBundle(claimedSource); !IsCode(err, ErrorConfig) {
		t.Fatalf("portable source provenance error = %v", err)
	}
	sharedBearer := proto.Clone(share).(*remoteauthpb.ClientEndpointShareBundleV1)
	sharedBearer.BoundGrant = []byte("legacy-bearer")
	if _, err := MarshalClientEndpointShareBundle(sharedBearer); !IsCode(err, ErrorAuthorizationRequired) {
		t.Fatalf("unverified shared bound grant error = %v", err)
	}
	whitespaceIdentity := proto.Clone(share).(*remoteauthpb.ClientEndpointShareBundleV1)
	whitespaceIdentity.Identity.DeviceId = " device-studio"
	if _, err := MarshalClientEndpointShareBundle(whitespaceIdentity); !IsCode(err, ErrorConfig) {
		t.Fatalf("whitespace identity error = %v", err)
	}

	offer := &remoteauthpb.ShareSessionOffer{
		SchemaVersion: ShareSessionOfferVersion, TransferId: "transfer-1", ListenerAddresses: []string{"192.0.2.10:41130"},
		EphemeralCertificateSha256: "sha256:" + base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, sha256.Size)), OneTimeSessionSecret: bytes.Repeat([]byte{4}, 32), ExpiresAtUnixNano: now.Add(time.Minute).UnixNano(),
	}
	offerPayload, err := MarshalShareSessionOffer(offer)
	if err != nil {
		t.Fatal(err)
	}
	if parsedOffer, err := ParseShareSessionOffer(offerPayload); err != nil || parsedOffer.GetTransferId() != offer.GetTransferId() {
		t.Fatalf("parse share offer: offer=%v err=%v", parsedOffer, err)
	}
	expiredOffer := proto.Clone(offer).(*remoteauthpb.ShareSessionOffer)
	expiredOffer.ExpiresAtUnixNano = now.Add(-time.Second).UnixNano()
	if _, err := MarshalShareSessionOffer(expiredOffer); !IsCode(err, ErrorConfig) {
		t.Fatalf("expired share offer error = %v", err)
	}
}

func testDirectRoute(routeID string, addresses ...string) *remoteauthpb.EndpointRouteConfigV1 {
	return &remoteauthpb.EndpointRouteConfigV1{
		SchemaVersion: RouteConfigVersion,
		RouteId:       routeID,
		Enabled:       true,
		Route: &remoteauthpb.EndpointRouteConfigV1_DirectWebrtcTcp{DirectWebrtcTcp: &remoteauthpb.DirectWebRTCTCPRouteConfig{
			SignalingAddresses: append([]string(nil), addresses...),
			IceTcpAddresses:    append([]string(nil), addresses...),
		}},
	}
}

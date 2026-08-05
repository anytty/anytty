package client

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
)

func TestPairingRouteBuildsOnlineAdmissionWithoutGrantOrFullCA(t *testing.T) {
	claim := bytes.Repeat([]byte{0x41}, 16)
	offer := &remoteauthpb.PairingClaimOffer{
		SchemaVersion: remoteauth.PairingClaimOfferVersion, Claim: claim, DeviceId: "device-1", DevicePublicKey: bytes.Repeat([]byte{0x42}, ed25519.PublicKeySize),
		ExpiresAtUnixNano: time.Now().Add(time.Minute).UnixNano(),
		Routes: []*remoteauthpb.PairingRouteSeed{{RouteId: "cloud", Route: &remoteauthpb.PairingRouteSeed_ManagedWebrtc{ManagedWebrtc: &remoteauthpb.PairingManagedRouteSeed{
			DaemonId: "daemon-1", EdgeId: "edge-1", PublicEndpoint: "edge.example:443", ServerName: "edge.example", CaCertificateDerSha256: bytes.Repeat([]byte{0x43}, sha256.Size),
		}}}},
	}
	payload, err := remoteauth.EncodePairingClaimOffer(offer)
	if err != nil {
		t.Fatal(err)
	}
	network, err := NewClient(Config{ControllerAddress: "controller.example:443", ControllerServerName: "controller.example", BootID: "00000000-0000-4000-8000-000000000001"})
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := network.PairingRoute(payload)
	if err != nil {
		t.Fatal(err)
	}
	wantDigest := sha256.Sum256(claim)
	if resolution.routeGrant != nil || resolution.locator != nil || resolution.pairingBootstrap == nil || resolution.pairingAdmission == nil ||
		resolution.pairingAdmission.GetDaemonId() != "daemon-1" || resolution.pairingAdmission.GetDeviceId() != "device-1" || !bytes.Equal(resolution.pairingAdmission.GetPairingClaimSha256(), wantDigest[:]) {
		t.Fatalf("pairing resolution = %#v", resolution)
	}
}

func TestNewClientRequiresOwnerBootID(t *testing.T) {
	base := Config{ControllerAddress: "controller.example:443", ControllerServerName: "controller.example"}
	for _, bootID := range []string{"", "not-a-uuid", "00000000-0000-0000-0000-000000000000", "00000000-0000-4000-8000-00000000000A"} {
		config := base
		config.BootID = bootID
		if _, err := NewClient(config); err == nil {
			t.Fatalf("NewClient accepted boot ID %q", bootID)
		}
	}
	base.BootID = "00000000-0000-4000-8000-00000000000a"
	if _, err := NewClient(base); err != nil {
		t.Fatalf("NewClient rejected canonical owner boot ID: %v", err)
	}
}

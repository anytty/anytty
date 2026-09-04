package enginehost

import (
	"testing"

	"github.com/anytty/anytty/client/endpoint"
)

func TestPairingTargetKeepsClaimRoutesIsolatedFromLocalDiscovery(t *testing.T) {
	claimRoute := endpoint.AccessRoute{
		ID: "direct", Kind: endpoint.RouteDirectWebRTCTCP, Enabled: true,
		Source: endpoint.SourceBootstrap, PolicySource: endpoint.SourceBootstrap,
		SignalingAddresses: []string{"47.108.66.192:41120"},
		ICETCPAddresses:    []string{"47.108.66.192:41120"},
	}
	target := pairingTarget(
		"device-cn2",
		endpoint.DaemonIdentity{DeviceID: "device-cn2", DeviceFingerprint: "ed25519-sha256:test"},
		[]endpoint.AccessRoute{claimRoute},
		"credential:device-cn2",
	)
	if len(target.Routes) != 1 {
		t.Fatalf("pairing route count = %d, want 1", len(target.Routes))
	}
	route, ok := target.Routes["direct"]
	if !ok || len(route.SignalingAddresses) != 1 || route.SignalingAddresses[0] != "47.108.66.192:41120" {
		t.Fatalf("pairing direct route = %#v", route)
	}
	if route.CredentialRef != "credential:device-cn2" {
		t.Fatalf("pairing credential ref = %q", route.CredentialRef)
	}
}

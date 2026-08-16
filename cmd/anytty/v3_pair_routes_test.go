package main

import (
	"testing"

	remoteauthpb "github.com/anytty/anytty/proto/remoteauthpb"
)

func TestV3PairingRoutesAcceptOrdinaryDirectParameters(t *testing.T) {
	routes, err := v3PairingRoutes(v3PairRouteFlags{
		Routes: []string{"direct"}, DirectID: "frp", DirectName: "FRP Public",
		SignalingAddresses: []string{"frp.example.com:443"}, ICETCPAddresses: []string{"frp.example.com:444"}, ServerName: "mac.example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || routes[0].GetRouteId() != "direct-frp" || routes[0].GetDisplayName() != "FRP Public" || routes[0].GetDirectWebrtcTcp().GetServerName() != "mac.example.com" {
		t.Fatalf("routes = %#v", routes)
	}
}

func TestV3PairingRoutesUseOneDirectAddressForBothProtocols(t *testing.T) {
	routes, err := v3PairingRoutes(v3PairRouteFlags{Routes: []string{"direct"}, DirectAddresses: []string{"direct.example.com:443"}})
	if err != nil {
		t.Fatal(err)
	}
	direct := routes[0].GetDirectWebrtcTcp()
	if got := direct.GetSignalingAddresses(); len(got) != 1 || got[0] != "direct.example.com:443" {
		t.Fatalf("signaling addresses = %v", got)
	}
	if got := direct.GetIceTcpAddresses(); len(got) != 1 || got[0] != "direct.example.com:443" {
		t.Fatalf("ICE TCP addresses = %v", got)
	}
	if _, err := v3PairingRoutes(v3PairRouteFlags{
		Routes: []string{"direct"}, DirectAddresses: []string{"direct.example.com:443"},
		SignalingAddresses: []string{"direct.example.com:41120"}, ICETCPAddresses: []string{"direct.example.com:41121"},
	}); err == nil {
		t.Fatal("shared and legacy split Direct flags were accepted together")
	}
}

func TestV3PairingRoutesAcceptStrictURIAndOrdinarySSH(t *testing.T) {
	routes, err := v3PairingRoutes(v3PairRouteFlags{Routes: []string{
		"direct://frp?name=FRP%20Public&address=frp.example.com:443",
		"ssh",
	}, SSHID: "office", SSHName: "Office SSH", SSHHost: "mac.example.com", SSHPort: 2222, SSHUser: "lozzow", SSHHostKeys: []string{"SHA256:abc"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 2 || routes[0].GetRouteId() != "direct-frp" || routes[1].GetRouteId() != "ssh-office" || routes[1].GetSshWebrtcTcp().GetPort() != 2222 {
		t.Fatalf("routes = %#v", routes)
	}
}

func TestV3PairingRoutesRejectFieldsWithoutExplicitRoute(t *testing.T) {
	if _, err := v3PairingRoutes(v3PairRouteFlags{SignalingAddresses: []string{"frp.example.com:443"}, ICETCPAddresses: []string{"frp.example.com:444"}}); err == nil {
		t.Fatal("implicit parameterized Direct Route was accepted")
	}
	if _, err := v3PairingRoutes(v3PairRouteFlags{Routes: []string{"direct://lan?unknown=value"}}); err == nil {
		t.Fatal("unknown URI query was accepted")
	}
	if routes, err := v3PairingRoutes(v3PairRouteFlags{Routes: []string{"cloud"}}); err != nil || len(routes) != 1 || routes[0].GetManagedWebrtc() == nil {
		t.Fatalf("Cloud-only pairing route = %#v err=%v", routes, err)
	}
	if _, err := v3PairingRoutes(v3PairRouteFlags{Routes: []string{"direct"}, SSHHost: "ignored"}); err == nil {
		t.Fatal("out-of-scope SSH fields were silently ignored")
	}
}

func TestV3PairingRoutesAddsCloudAlongsideDirect(t *testing.T) {
	routes, err := v3PairingRoutes(v3PairRouteFlags{
		Routes:             []string{"direct", "cloud"},
		SignalingAddresses: []string{"127.0.0.1:44111"},
		ICETCPAddresses:    []string{"127.0.0.1:44112"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 2 || routes[1].GetRouteId() != "cloud" || routes[1].GetManagedWebrtc().GetAccountProfileRef() != "default" {
		t.Fatalf("routes = %#v", routes)
	}
	if routes[1].GetManagedWebrtc().GetRelayMode() != remoteauthpb.ManagedWebRTCRelayMode_MANAGED_WEBRTC_RELAY_MODE_AUTO {
		t.Fatalf("portable Cloud route did not default to auto: %#v", routes[1].GetManagedWebrtc())
	}
	if routes[1].GetManagedWebrtc().GetTargetDeviceId() != "" {
		t.Fatalf("pair flags invented daemon identity %q", routes[1].GetManagedWebrtc().GetTargetDeviceId())
	}
	if routes[1].GetSource() != 0 || routes[1].GetPolicySource() != 0 {
		t.Fatalf("portable Cloud route claimed source provenance: %#v", routes[1])
	}
}

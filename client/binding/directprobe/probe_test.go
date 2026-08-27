package directprobe

import (
	"context"
	"net"
	"testing"

	"github.com/anytty/anytty/proto/remoteauthpb"
)

func TestReachableProbesUniqueSignalingAddresses(t *testing.T) {
	route := &remoteauthpb.DirectWebRTCTCPRouteConfig{
		SignalingAddresses:  []string{"direct.example.test:443", "direct.example.test:443", "[2001:db8::1]:41120"},
		IceTcpAddresses:     []string{"ice.example.test:444"},
		AdvertisedAddresses: []string{"advertised.example.test:445"},
	}
	addresses := normalizedAddresses(route.GetSignalingAddresses())
	if len(addresses) != 2 || addresses[0] != "direct.example.test:443" || addresses[1] != "[2001:db8::1]:41120" {
		t.Fatalf("normalized addresses = %v", addresses)
	}
	reachable := Reachable(context.Background(), route, func(_ context.Context, _ string) (net.Conn, error) {
		left, right := net.Pipe()
		_ = right.Close()
		return left, nil
	})
	if !reachable {
		t.Fatal("Reachable returned false")
	}
}

func TestReachableRejectsInvalidAddresses(t *testing.T) {
	route := &remoteauthpb.DirectWebRTCTCPRouteConfig{SignalingAddresses: []string{
		"", "missing-port", "https://direct.example.test:443", "direct.example.test:0", "direct.example.test:65536",
	}}
	called := false
	if Reachable(context.Background(), route, func(context.Context, string) (net.Conn, error) {
		called = true
		return nil, nil
	}) {
		t.Fatal("invalid addresses were reachable")
	}
	if called {
		t.Fatal("dialer was called for invalid addresses")
	}
}

func TestReachableReturnsFalseWithoutAConfiguredRoute(t *testing.T) {
	if Reachable(context.Background(), nil, nil) {
		t.Fatal("nil route was reachable")
	}
	if Reachable(context.Background(), &remoteauthpb.DirectWebRTCTCPRouteConfig{}, nil) {
		t.Fatal("empty route was reachable")
	}
}

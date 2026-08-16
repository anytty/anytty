package main

import (
	"os"
	"reflect"
	"testing"
)

func TestDirectListenerSeedsProjectWildcardToPrivateLAN(t *testing.T) {
	previous := v3PrivateLANAddresses
	v3PrivateLANAddresses = func() ([]string, error) { return []string{"192.168.1.20", "10.0.0.8"}, nil }
	t.Cleanup(func() { v3PrivateLANAddresses = previous })
	signaling, ice, err := directListenerSeeds("0.0.0.0:41120", "[::]:41121")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(signaling, []string{"10.0.0.8:41120", "192.168.1.20:41120"}) ||
		!reflect.DeepEqual(ice, []string{"10.0.0.8:41121", "192.168.1.20:41121"}) {
		t.Fatalf("LAN seeds signaling=%#v ice=%#v", signaling, ice)
	}
}

func TestDirectAddressesPreferSharedListenerAndKeepLegacySplit(t *testing.T) {
	for _, key := range []string{"ANYTTY_DIRECT_LISTEN", "ANYTTY_DIRECT_SIGNALING_LISTEN", "ANYTTY_DIRECT_ICE_TCP_LISTEN"} {
		previous, present := os.LookupEnv(key)
		_ = os.Unsetenv(key)
		t.Cleanup(func() {
			if present {
				_ = os.Setenv(key, previous)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}
	t.Setenv("ANYTTY_DIRECT_LISTEN", "0.0.0.0:45000")
	if signaling, ice := v3DirectAddresses(); signaling != "0.0.0.0:45000" || ice != signaling {
		t.Fatalf("shared listener = (%q, %q)", signaling, ice)
	}
	t.Setenv("ANYTTY_DIRECT_SIGNALING_LISTEN", "127.0.0.1:45001")
	t.Setenv("ANYTTY_DIRECT_ICE_TCP_LISTEN", "127.0.0.1:45002")
	if signaling, ice := v3DirectAddresses(); signaling != "127.0.0.1:45001" || ice != "127.0.0.1:45002" {
		t.Fatalf("legacy split listeners = (%q, %q)", signaling, ice)
	}
}

func TestDirectListenRouteAcceptsWildcardAndExplicitHost(t *testing.T) {
	for _, address := range []string{"0.0.0.0:41120", "[::]:41120", "192.168.1.8:41120", "direct.example.test:443"} {
		if err := validateDirectListenAddress(address); err != nil {
			t.Fatalf("route %q: %v", address, err)
		}
	}
	for _, address := range []string{"0.0.0.0", "0.0.0.0:0", "0.0.0.0:nope"} {
		if err := validateDirectListenAddress(address); err == nil {
			t.Fatalf("invalid route %q accepted", address)
		}
	}
}

func TestDirectPairingRouteExplicitAddressesAreCanonicalAndDeterministic(t *testing.T) {
	route, err := v3DirectPairingRoute(v3DirectPairingRouteOptions{
		SignalingAddresses: []string{"frp.example:51020", "frp.example:51020"},
		ICETCPAddresses:    []string{"203.0.113.8:51021"}, ServerName: "frp.example",
	})
	if err != nil {
		t.Fatal(err)
	}
	direct := route.GetDirectWebrtcTcp()
	if !reflect.DeepEqual(direct.GetSignalingAddresses(), []string{"frp.example:51020"}) ||
		!reflect.DeepEqual(direct.GetIceTcpAddresses(), []string{"203.0.113.8:51021"}) ||
		!reflect.DeepEqual(direct.GetAdvertisedAddresses(), []string{"203.0.113.8:51021", "frp.example:51020"}) {
		t.Fatalf("explicit route = %#v", direct)
	}
}

func TestDirectPairingRouteUsesRecordedDaemonListenerByDefault(t *testing.T) {
	previous := v3PrivateLANAddresses
	v3PrivateLANAddresses = func() ([]string, error) { return []string{"192.168.1.28"}, nil }
	t.Cleanup(func() { v3PrivateLANAddresses = previous })
	route, err := v3DirectPairingRoute(v3DirectPairingRouteOptions{DefaultListen: "0.0.0.0:45000"})
	if err != nil {
		t.Fatal(err)
	}
	direct := route.GetDirectWebrtcTcp()
	if !reflect.DeepEqual(direct.GetSignalingAddresses(), []string{"192.168.1.28:45000"}) || !reflect.DeepEqual(direct.GetIceTcpAddresses(), []string{"192.168.1.28:45000"}) {
		t.Fatalf("recorded listener route = %#v", direct)
	}
}

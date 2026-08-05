package main

import (
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

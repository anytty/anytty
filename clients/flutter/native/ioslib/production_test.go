//go:build cgo

package main

import "testing"

func TestIOSPeerFactoryUsesNativeInterfaceEnumeration(t *testing.T) {
	factory := newIOSPeerFactory()
	if factory.Network != nil || factory.NetworkFactory != nil || factory.RouteNetworkFactory != nil {
		t.Fatal("iOS peer factory must leave Pion network enumeration enabled")
	}
}

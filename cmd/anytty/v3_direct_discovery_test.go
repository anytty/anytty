package main

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/wire"
)

type fakeDirectDiscoveryServer struct{ closed bool }

func (server *fakeDirectDiscoveryServer) Shutdown() { server.closed = true }

func TestStartDirectDiscoveryPublishesWildcardListenerIdentity(t *testing.T) {
	originalRegister, originalAddresses := registerDirectDiscovery, v3PrivateLANAddresses
	defer func() { registerDirectDiscovery, v3PrivateLANAddresses = originalRegister, originalAddresses }()
	v3PrivateLANAddresses = func() ([]string, error) { return []string{"192.168.1.9", "192.168.1.8"}, nil }
	server := &fakeDirectDiscoveryServer{}
	var gotPort int
	var gotText, gotAddresses []string
	registerDirectDiscovery = func(_ string, port int, text []string, addresses []string) (directDiscoveryServer, error) {
		gotPort, gotText, gotAddresses = port, append([]string(nil), text...), append([]string(nil), addresses...)
		return server, nil
	}
	deviceID, fingerprint := "device-1234567890", "ed25519-sha256:test"
	closeDiscovery, err := startDirectDiscovery(context.Background(), deviceID, fingerprint, "0.0.0.0:41120", "[::]:41120")
	if err != nil {
		t.Fatal(err)
	}
	if gotPort != 41120 || !reflect.DeepEqual(gotAddresses, []string{"192.168.1.8", "192.168.1.9"}) {
		t.Fatalf("published port=%d addresses=%v", gotPort, gotAddresses)
	}
	wantText := []string{"v=1", fmt.Sprintf("p=%d", wire.Version), "k=" + directDiscoveryKey(deviceID, fingerprint)}
	if !reflect.DeepEqual(gotText, wantText) {
		t.Fatalf("TXT = %v, want %v", gotText, wantText)
	}
	for _, value := range gotText {
		if value == "id="+deviceID || value == "fp="+fingerprint {
			t.Fatalf("discovery exposed a stable device identity: %v", gotText)
		}
	}
	closeDiscovery()
	if !server.closed {
		t.Fatal("discovery server was not shut down")
	}
}

func TestStartDirectDiscoverySkipsExplicitListener(t *testing.T) {
	original := registerDirectDiscovery
	defer func() { registerDirectDiscovery = original }()
	called := false
	registerDirectDiscovery = func(string, int, []string, []string) (directDiscoveryServer, error) {
		called = true
		return &fakeDirectDiscoveryServer{}, nil
	}
	closeDiscovery, err := startDirectDiscovery(context.Background(), "", "", "192.168.1.8:41120", "192.168.1.8:41120")
	if err != nil {
		t.Fatal(err)
	}
	closeDiscovery()
	if called {
		t.Fatal("explicit listener unexpectedly enabled discovery")
	}
}

func TestStartDirectDiscoveryRegistersWhenLANAppearsAfterStartup(t *testing.T) {
	originalRegister, originalAddresses, originalPeriod := registerDirectDiscovery, v3PrivateLANAddresses, directDiscoveryRefreshPeriod
	defer func() {
		registerDirectDiscovery, v3PrivateLANAddresses, directDiscoveryRefreshPeriod = originalRegister, originalAddresses, originalPeriod
	}()
	directDiscoveryRefreshPeriod = time.Millisecond
	var lookups atomic.Int32
	v3PrivateLANAddresses = func() ([]string, error) {
		if lookups.Add(1) == 1 {
			return nil, nil
		}
		return []string{"192.168.1.15"}, nil
	}
	registered := make(chan *fakeDirectDiscoveryServer, 1)
	registerDirectDiscovery = func(_ string, _ int, _ []string, addresses []string) (directDiscoveryServer, error) {
		if !reflect.DeepEqual(addresses, []string{"192.168.1.15"}) {
			t.Errorf("addresses = %v", addresses)
		}
		server := &fakeDirectDiscoveryServer{}
		registered <- server
		return server, nil
	}
	closeDiscovery, err := startDirectDiscovery(context.Background(), "device", "fingerprint", "0.0.0.0:41120", "0.0.0.0:41120")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case server := <-registered:
		closeDiscovery()
		if !server.closed {
			t.Fatal("late discovery server was not shut down")
		}
	case <-time.After(time.Second):
		t.Fatal("LAN discovery did not register after an address appeared")
	}
}

func TestStartDirectDiscoveryStopsPreviousAdvertisementBeforeRefreshing(t *testing.T) {
	originalRegister, originalAddresses, originalPeriod := registerDirectDiscovery, v3PrivateLANAddresses, directDiscoveryRefreshPeriod
	defer func() {
		registerDirectDiscovery, v3PrivateLANAddresses, directDiscoveryRefreshPeriod = originalRegister, originalAddresses, originalPeriod
	}()
	directDiscoveryRefreshPeriod = time.Millisecond
	var lookups atomic.Int32
	v3PrivateLANAddresses = func() ([]string, error) {
		if lookups.Add(1) == 1 {
			return []string{"192.168.1.15"}, nil
		}
		return []string{"192.168.1.16"}, nil
	}
	first := &fakeDirectDiscoveryServer{}
	second := &fakeDirectDiscoveryServer{}
	refreshed := make(chan struct{}, 1)
	var registrations atomic.Int32
	registerDirectDiscovery = func(_ string, _ int, _ []string, addresses []string) (directDiscoveryServer, error) {
		switch registrations.Add(1) {
		case 1:
			return first, nil
		case 2:
			if !first.closed {
				t.Error("replacement registered before the previous DNS-SD advertisement stopped")
			}
			if !reflect.DeepEqual(addresses, []string{"192.168.1.16"}) {
				t.Errorf("replacement addresses = %v", addresses)
			}
			refreshed <- struct{}{}
			return second, nil
		default:
			return second, nil
		}
	}
	closeDiscovery, err := startDirectDiscovery(context.Background(), "device", "fingerprint", "0.0.0.0:41120", "0.0.0.0:41120")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-refreshed:
		closeDiscovery()
		if !second.closed {
			t.Fatal("replacement discovery server was not shut down")
		}
	case <-time.After(time.Second):
		closeDiscovery()
		t.Fatal("LAN discovery advertisement was not refreshed")
	}
}

func TestDirectDiscoveryKeyContract(t *testing.T) {
	if got, want := directDiscoveryKey(" device-1234567890 ", " ed25519-sha256:test "), "f53482982aaa564cffa0b7a6964bf7694745e2f379d6a85f871b60d2585dde8f"; got != want {
		t.Fatalf("discovery key = %q, want %q", got, want)
	}
}

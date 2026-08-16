//go:build cgo

package main

import (
	"errors"
	"testing"

	pionadapter "github.com/anytty/anytty/client/adapter/webrtc/pion"
	"github.com/pion/transport/v4"
)

func TestAndroidProductionHostStartsWhileNetworkIsOffline(t *testing.T) {
	calls := 0
	host, err := newAndroidProductionHostWithPeers(pionadapter.Factory{NetworkFactory: func() (transport.Net, error) {
		calls++
		return nil, errors.New("network is offline")
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer host.close()
	if calls != 0 {
		t.Fatalf("Android host startup created %d network snapshots, want 0", calls)
	}
}

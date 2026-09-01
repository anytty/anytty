//go:build cgo

package main

import (
	"context"
	"errors"
	"testing"

	pionadapter "github.com/anytty/anytty/client/adapter/webrtc/pion"
	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
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

func TestAndroidSupervisorWaitReadyReportsTimeout(t *testing.T) {
	if got := waitEndpointDemandReadyStatus(timeoutSupervisorHost{}, 1); got == 0 {
		t.Fatal("supervisor readiness timeout was reported as OK")
	}
}

type timeoutSupervisorHost struct{}

func (timeoutSupervisorHost) ReplaceEndpointDemand(clientruntime.EndpointDemandSnapshot) error {
	return nil
}
func (timeoutSupervisorHost) SignalEndpointHost(clientruntime.EndpointHostSignal) error { return nil }
func (timeoutSupervisorHost) RepairEndpoint(endpoint.EndpointID) error                  { return nil }
func (timeoutSupervisorHost) WaitEndpointDemandReady(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
func (timeoutSupervisorHost) EndpointSupervisorSnapshot() []clientruntime.EndpointSupervisorProjection {
	return nil
}

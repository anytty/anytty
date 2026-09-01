//go:build cgo

package main

import (
	"context"
	"testing"

	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
)

func TestIOSPeerFactoryUsesNativeInterfaceEnumeration(t *testing.T) {
	factory := newIOSPeerFactory()
	if factory.Network != nil || factory.NetworkFactory != nil || factory.RouteNetworkFactory != nil {
		t.Fatal("iOS peer factory must leave Pion network enumeration enabled")
	}
}

func TestIOSSupervisorWaitReadyReportsTimeout(t *testing.T) {
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

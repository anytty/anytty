package cloud

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
)

func TestCloudPeerAttemptsUseDirectFirstForAutomaticModes(t *testing.T) {
	for _, mode := range []endpoint.RelayMode{"", endpoint.RelayAuto, endpoint.RelaySmart} {
		attempts, err := cloudPeerAttempts(mode)
		if err != nil {
			t.Fatalf("cloudPeerAttempts(%q): %v", mode, err)
		}
		if len(attempts) != 2 ||
			attempts[0].preference != cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY || attempts[0].icePolicy != port.ICETransportAll || attempts[0].readyTimeout != 3*time.Second ||
			attempts[1].preference != cloudv1.RelayPreference_RELAY_PREFERENCE_RELAY_ONLY || attempts[1].icePolicy != port.ICETransportRelayOnly || attempts[1].readyTimeout != 0 {
			t.Fatalf("cloudPeerAttempts(%q) = %#v", mode, attempts)
		}
	}
}

func TestCloudPeerAttemptsPreserveExplicitPolicies(t *testing.T) {
	direct, err := cloudPeerAttempts(endpoint.RelayDirect)
	if err != nil || len(direct) != 1 || direct[0].preference != cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY || direct[0].readyTimeout != 0 {
		t.Fatalf("direct attempts=%#v err=%v", direct, err)
	}
	relay, err := cloudPeerAttempts(endpoint.RelayOnly)
	if err != nil || len(relay) != 1 || relay[0].preference != cloudv1.RelayPreference_RELAY_PREFERENCE_RELAY_ONLY || relay[0].icePolicy != port.ICETransportRelayOnly {
		t.Fatalf("relay attempts=%#v err=%v", relay, err)
	}
	if _, err := cloudPeerAttempts("invalid"); err == nil {
		t.Fatal("invalid relay mode was accepted")
	}
}

func TestOnlyPeerConnectivityFailuresPermitRelayFallback(t *testing.T) {
	connectivity := &cloudPeerConnectivityError{err: errors.New("ICE failed")}
	if !isCloudPeerConnectivityError(connectivity) || !isCloudPeerConnectivityError(errors.Join(errors.New("context"), connectivity)) {
		t.Fatal("wrapped connectivity failure was not classified")
	}
	if isCloudPeerConnectivityError(errors.New("client proof rejected")) {
		t.Fatal("authorization failure was classified as connectivity")
	}
}

func TestRunCloudPeerAttemptsFallsBackOnlyAfterConnectivityFailure(t *testing.T) {
	attempts, err := cloudPeerAttempts(endpoint.RelayAuto)
	if err != nil {
		t.Fatal(err)
	}
	opened := &openedCloudPeer{}
	var called []cloudv1.RelayPreference
	fallbacks := 0
	result, err := runCloudPeerAttempts(context.Background(), attempts, func(attempt cloudPeerAttempt) (*openedCloudPeer, error) {
		called = append(called, attempt.preference)
		if len(called) == 1 {
			return nil, &cloudPeerConnectivityError{err: errors.New("ICE failed")}
		}
		return opened, nil
	}, func() { fallbacks++ })
	if err != nil || result != opened || fallbacks != 1 || len(called) != 2 || called[0] != cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY || called[1] != cloudv1.RelayPreference_RELAY_PREFERENCE_RELAY_ONLY {
		t.Fatalf("result=%p err=%v fallbacks=%d called=%v", result, err, fallbacks, called)
	}
}

func TestRunCloudPeerAttemptsDoesNotFallbackAfterAuthorizationOrCancellation(t *testing.T) {
	attempts, err := cloudPeerAttempts(endpoint.RelayAuto)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		ctx  context.Context
		err  error
	}{
		{name: "authorization", ctx: context.Background(), err: errors.New("client proof rejected")},
		{name: "canceled connectivity", ctx: canceledContext(), err: &cloudPeerConnectivityError{err: context.Canceled}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			_, gotErr := runCloudPeerAttempts(test.ctx, attempts, func(cloudPeerAttempt) (*openedCloudPeer, error) {
				calls++
				return nil, test.err
			}, nil)
			if !errors.Is(gotErr, test.err) || calls != 1 {
				t.Fatalf("err=%v calls=%d", gotErr, calls)
			}
		})
	}
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

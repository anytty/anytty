package cloud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
)

type cloudPathObserverFunc func() endpoint.Path

func (observe cloudPathObserverFunc) ObservedPath() endpoint.Path { return observe() }

func TestObserveCloudPeerPathReadsCurrentSelectedPair(t *testing.T) {
	current := endpoint.PathDirect
	observe := cloudPathObserverFunc(func() endpoint.Path { return current })
	if got, err := observeCloudPeerPath(observe, port.ICETransportAll); err != nil || got != endpoint.PathDirect {
		t.Fatalf("initial path = %q, err=%v", got, err)
	}
	current = endpoint.PathSingleRelay
	if got, err := observeCloudPeerPath(observe, port.ICETransportAll); err != nil || got != endpoint.PathSingleRelay {
		t.Fatalf("reselected path = %q, err=%v", got, err)
	}
	current = endpoint.PathDirect
	if _, err := observeCloudPeerPath(observe, port.ICETransportRelayOnly); err == nil {
		t.Fatal("Relay-only peer accepted a reselected direct path")
	}
	current = ""
	if _, err := observeCloudPeerPath(observe, port.ICETransportAll); err == nil {
		t.Fatal("peer accepted an unavailable selected path")
	}
}

func TestCloudPeerAttemptUsesSingleICECompetitionForAutomaticModes(t *testing.T) {
	for _, mode := range []endpoint.RelayMode{"", endpoint.RelayAuto, endpoint.RelaySmart} {
		attempt, err := planCloudPeerAttempt(mode)
		if err != nil {
			t.Fatalf("planCloudPeerAttempt(%q): %v", mode, err)
		}
		if attempt.preference != cloudv1.RelayPreference_RELAY_PREFERENCE_AUTO || attempt.icePolicy != port.ICETransportAll {
			t.Fatalf("planCloudPeerAttempt(%q) = %#v", mode, attempt)
		}
	}
}

func TestCloudPeerAttemptPreservesExplicitPolicies(t *testing.T) {
	direct, err := planCloudPeerAttempt(endpoint.RelayDirect)
	if err != nil || direct.preference != cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY || direct.icePolicy != port.ICETransportAll {
		t.Fatalf("direct attempts=%#v err=%v", direct, err)
	}
	relay, err := planCloudPeerAttempt(endpoint.RelayOnly)
	if err != nil || relay.preference != cloudv1.RelayPreference_RELAY_PREFERENCE_RELAY_ONLY || relay.icePolicy != port.ICETransportRelayOnly {
		t.Fatalf("relay attempts=%#v err=%v", relay, err)
	}
	if _, err := planCloudPeerAttempt("invalid"); err == nil {
		t.Fatal("invalid relay mode was accepted")
	}
}

func TestCloudPeerTransportFallbackRequiresAutomaticPolicyAndTransportEOF(t *testing.T) {
	transportEOF := &remoteauth.HandshakeError{
		Code:   remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL,
		Detail: "receive remote auth frame",
		Cause:  io.EOF,
	}
	for _, mode := range []endpoint.RelayMode{"", endpoint.RelayAuto, endpoint.RelaySmart} {
		attempt, ok := planCloudPeerTransportFallback(endpoint.AccessRoute{RelayMode: mode}, transportEOF)
		if !ok || attempt.preference != cloudv1.RelayPreference_RELAY_PREFERENCE_RELAY_ONLY ||
			attempt.icePolicy != port.ICETransportRelayOnly || attempt.relayTransport != endpoint.RelayTransportTCP {
			t.Fatalf("fallback for mode %q = %#v, %t", mode, attempt, ok)
		}
	}

	for name, test := range map[string]struct {
		route endpoint.AccessRoute
		err   error
	}{
		"explicit direct": {route: endpoint.AccessRoute{RelayMode: endpoint.RelayDirect}, err: transportEOF},
		"explicit relay":  {route: endpoint.AccessRoute{RelayMode: endpoint.RelayOnly}, err: transportEOF},
		"explicit udp":    {route: endpoint.AccessRoute{RelayMode: endpoint.RelayAuto, RelayTransport: endpoint.RelayTransportUDP}, err: transportEOF},
		"context cancel":  {route: endpoint.AccessRoute{RelayMode: endpoint.RelayAuto}, err: context.Canceled},
		"malformed frame": {route: endpoint.AccessRoute{RelayMode: endpoint.RelayAuto}, err: &remoteauth.HandshakeError{Code: remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL, Detail: "invalid remote auth frame"}},
		"auth rejection":  {route: endpoint.AccessRoute{RelayMode: endpoint.RelayAuto}, err: &remoteauth.HandshakeError{Code: remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_REVOKED, Detail: "revoked"}},
	} {
		t.Run(name, func(t *testing.T) {
			if attempt, ok := planCloudPeerTransportFallback(test.route, test.err); ok {
				t.Fatalf("unsafe fallback = %#v", attempt)
			}
		})
	}
}

func TestCloudPeerPostAuthFallbackRequiresLiveAutomaticDirectPath(t *testing.T) {
	for _, mode := range []endpoint.RelayMode{"", endpoint.RelayAuto, endpoint.RelaySmart} {
		for _, failure := range []error{io.EOF, io.ErrUnexpectedEOF, context.Canceled, context.DeadlineExceeded} {
			attempt, ok := planCloudPeerPostAuthFallback(
				context.Background(),
				endpoint.AccessRoute{RelayMode: mode},
				endpoint.PathDirect,
				true,
				fmt.Errorf("wrapped: %w", failure),
			)
			if !ok || attempt.preference != cloudv1.RelayPreference_RELAY_PREFERENCE_RELAY_ONLY ||
				attempt.icePolicy != port.ICETransportRelayOnly || attempt.relayTransport != endpoint.RelayTransportTCP {
				t.Fatalf("fallback for mode %q and failure %v = %#v, %t", mode, failure, attempt, ok)
			}
		}
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for name, test := range map[string]struct {
		ctx       context.Context
		route     endpoint.AccessRoute
		path      endpoint.Path
		available bool
		err       error
	}{
		"parent canceled":  {ctx: cancelled, route: endpoint.AccessRoute{RelayMode: endpoint.RelayAuto}, path: endpoint.PathDirect, available: true, err: context.Canceled},
		"explicit direct":  {ctx: context.Background(), route: endpoint.AccessRoute{RelayMode: endpoint.RelayDirect}, path: endpoint.PathDirect, available: true, err: context.DeadlineExceeded},
		"explicit relay":   {ctx: context.Background(), route: endpoint.AccessRoute{RelayMode: endpoint.RelayOnly}, path: endpoint.PathDirect, available: true, err: context.DeadlineExceeded},
		"explicit udp":     {ctx: context.Background(), route: endpoint.AccessRoute{RelayMode: endpoint.RelayAuto, RelayTransport: endpoint.RelayTransportUDP}, path: endpoint.PathDirect, available: true, err: context.DeadlineExceeded},
		"selected relay":   {ctx: context.Background(), route: endpoint.AccessRoute{RelayMode: endpoint.RelayAuto}, path: endpoint.PathSingleRelay, available: true, err: context.DeadlineExceeded},
		"no relay tcp":     {ctx: context.Background(), route: endpoint.AccessRoute{RelayMode: endpoint.RelayAuto}, path: endpoint.PathDirect, available: false, err: context.DeadlineExceeded},
		"semantic failure": {ctx: context.Background(), route: endpoint.AccessRoute{RelayMode: endpoint.RelayAuto}, path: endpoint.PathDirect, available: true, err: errors.New("forbidden")},
	} {
		t.Run(name, func(t *testing.T) {
			if attempt, ok := planCloudPeerPostAuthFallback(test.ctx, test.route, test.path, test.available, test.err); ok {
				t.Fatalf("unsafe fallback = %#v", attempt)
			}
		})
	}
}

package endpoint

import (
	"context"
	"reflect"
	"strings"
	"testing"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
)

// TestParseRouteKindsNamesAndAliases pins the operator-facing switch:
// local/tcp/lunix aliases, ssh, direct, cloud, all, dedup and readable errors.
func TestParseRouteKindsNamesAndAliases(t *testing.T) {
	got, err := ParseRouteKinds("local,tcp,ssh,ssh-webrtc-tcp")
	if err != nil {
		t.Fatalf("ParseRouteKinds: %v", err)
	}
	want := []clientendpoint.RouteKind{clientendpoint.RouteLocalUnix, clientendpoint.RouteSSHWebRTCTCP}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseRouteKinds(local,tcp,ssh) = %v, want %v", got, want)
	}
	got, err = ParseRouteKinds("all")
	if err != nil {
		t.Fatalf("ParseRouteKinds(all): %v", err)
	}
	want = []clientendpoint.RouteKind{
		clientendpoint.RouteLocalUnix, clientendpoint.RouteSSHWebRTCTCP,
		clientendpoint.RouteDirectWebRTCTCP, clientendpoint.RouteManagedWebRTC,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseRouteKinds(all) = %v, want %v", got, want)
	}
	if _, err := ParseRouteKinds("bogus"); err == nil || !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("ParseRouteKinds(bogus) error = %v; want a readable unknown-route error", err)
	}
}

// TestResolveRouteKindsFlagEnvDefault pins flag > TUI2_ROUTES > default.
func TestResolveRouteKindsFlagEnvDefault(t *testing.T) {
	t.Setenv(RoutesEnvVar, "direct")
	got, err := ResolveRouteKinds("")
	if err != nil || !reflect.DeepEqual(got, []clientendpoint.RouteKind{clientendpoint.RouteDirectWebRTCTCP}) {
		t.Fatalf("ResolveRouteKinds(env) = %v, %v; want just direct", got, err)
	}
	got, err = ResolveRouteKinds("local")
	if err != nil || !reflect.DeepEqual(got, []clientendpoint.RouteKind{clientendpoint.RouteLocalUnix}) {
		t.Fatalf("ResolveRouteKinds(flag) = %v, %v; want the flag to win", got, err)
	}
	t.Setenv(RoutesEnvVar, "")
	got, err = ResolveRouteKinds("")
	if err != nil || !reflect.DeepEqual(got, DefaultRouteKinds()) {
		t.Fatalf("ResolveRouteKinds(default) = %v, %v; want %v", got, err, DefaultRouteKinds())
	}
}

// TestSharedRouteEnvironmentPrunesOptInWebRTC is the M2 route-pruning
// contract: the default policy only ever offers local-unix (plus a
// credential-backed ssh), so a direct/cloud route in an old registry is not
// attempted; explicitly opting in adds direct to the plan.
func TestSharedRouteEnvironmentPrunesOptInWebRTC(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	target := clientendpoint.Endpoint{
		ID: "old", Label: "old", Enabled: true, ConnectMode: clientendpoint.ConnectAuto,
		DaemonIdentity: clientendpoint.DaemonIdentity{DeviceID: "dev-old", DeviceFingerprint: "ed25519-sha256:old"},
		Routes: map[clientendpoint.RouteID]clientendpoint.AccessRoute{
			"local": {
				ID: "local", Kind: clientendpoint.RouteLocalUnix, Enabled: true,
				Source: clientendpoint.SourceManual, PolicySource: clientendpoint.SourceManual, Socket: "/tmp/old.sock",
			},
			"direct": {
				ID: "direct", Kind: clientendpoint.RouteDirectWebRTCTCP, Enabled: true,
				Source: clientendpoint.SourceManual, PolicySource: clientendpoint.SourceManual,
				SignalingAddresses: []string{"127.0.0.1:1"}, ICETCPAddresses: []string{"127.0.0.1:1"},
			},
			"cloud": {
				ID: "cloud", Kind: clientendpoint.RouteManagedWebRTC, Enabled: true,
				Source: clientendpoint.SourceCloud, PolicySource: clientendpoint.SourceLocal, TargetDeviceID: "dev-old",
			},
		},
	}
	environment := sharedRouteEnvironment(context.Background(), target, nil, true)
	if !reflect.DeepEqual(environment.SupportedRouteKinds, []clientendpoint.RouteKind{clientendpoint.RouteLocalUnix}) {
		t.Fatalf("default supported kinds = %v; want local-unix only (direct/cloud pruned)", environment.SupportedRouteKinds)
	}

	environment = sharedRouteEnvironment(context.Background(), target, []clientendpoint.RouteKind{
		clientendpoint.RouteLocalUnix, clientendpoint.RouteDirectWebRTCTCP,
	}, true)
	if !routeKindEnabled(environment.SupportedRouteKinds, clientendpoint.RouteDirectWebRTCTCP) {
		t.Fatalf("opt-in supported kinds = %v; want direct-webrtc-tcp enabled", environment.SupportedRouteKinds)
	}
	if routeKindEnabled(environment.SupportedRouteKinds, clientendpoint.RouteManagedWebRTC) {
		t.Fatalf("managed-webrtc must stay pruned without a Cloud credential: %v", environment.SupportedRouteKinds)
	}
}

// TestConfigFromSharedEndpointDegradesToLocal pins the projection order: an
// old endpoint carrying direct+local routes projects the dialable local-unix
// route instead of the WebRTC one.
func TestConfigFromSharedEndpointDegradesToLocal(t *testing.T) {
	target := clientendpoint.Endpoint{
		ID: "old", Label: "old", Enabled: true, ConnectMode: clientendpoint.ConnectAuto,
		DaemonIdentity: clientendpoint.DaemonIdentity{DeviceID: "dev-old", DeviceFingerprint: "ed25519-sha256:old"},
		Routes: map[clientendpoint.RouteID]clientendpoint.AccessRoute{
			"direct": {
				ID: "direct", Kind: clientendpoint.RouteDirectWebRTCTCP, Enabled: true,
				Source: clientendpoint.SourceManual, PolicySource: clientendpoint.SourceManual,
				SignalingAddresses: []string{"127.0.0.1:1"}, ICETCPAddresses: []string{"127.0.0.1:1"},
			},
			"local": {
				ID: "local", Kind: clientendpoint.RouteLocalUnix, Enabled: true,
				Source: clientendpoint.SourceManual, PolicySource: clientendpoint.SourceManual, Socket: "/tmp/old.sock",
			},
		},
	}
	cfg, ok := configFromSharedEndpoint(target)
	if !ok {
		t.Fatal("configFromSharedEndpoint returned no config for a local+direct endpoint")
	}
	if cfg.ConnectMode != ConnectLocalUnix || cfg.Socket != "/tmp/old.sock" {
		t.Fatalf("degraded config = %+v; want local-unix /tmp/old.sock", cfg)
	}
}

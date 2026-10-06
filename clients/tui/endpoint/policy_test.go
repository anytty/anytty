package endpoint

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	"github.com/anytty/anytty/proto/access/wire"
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

// TestSharedRouteEnvironmentRacesConfiguredKinds pins the default policy:
// every configured, enabled kind races (direct included) and the first ready
// one wins; narrowing is an explicit -routes/TUI2_ROUTES choice, while
// credential and cloud gates still prune kinds that cannot connect.
func TestSharedRouteEnvironmentRacesConfiguredKinds(t *testing.T) {
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

	// Default: local and direct are offered; ssh and managed stay pruned because
	// no credential/cloud account is available.
	environment := sharedRouteEnvironment(context.Background(), target, nil, true)
	if !routeKindEnabled(environment.SupportedRouteKinds, clientendpoint.RouteLocalUnix) ||
		!routeKindEnabled(environment.SupportedRouteKinds, clientendpoint.RouteDirectWebRTCTCP) {
		t.Fatalf("default supported kinds = %v; want local-unix and direct-webrtc-tcp to race", environment.SupportedRouteKinds)
	}
	if routeKindEnabled(environment.SupportedRouteKinds, clientendpoint.RouteSSHWebRTCTCP) {
		t.Fatalf("ssh must stay pruned without a credential: %v", environment.SupportedRouteKinds)
	}
	if routeKindEnabled(environment.SupportedRouteKinds, clientendpoint.RouteManagedWebRTC) {
		t.Fatalf("managed-webrtc must stay pruned without a Cloud credential: %v", environment.SupportedRouteKinds)
	}

	// Explicit narrowing wins: only the named kinds are offered.
	environment = sharedRouteEnvironment(context.Background(), target, []clientendpoint.RouteKind{
		clientendpoint.RouteLocalUnix,
	}, true)
	if !reflect.DeepEqual(environment.SupportedRouteKinds, []clientendpoint.RouteKind{clientendpoint.RouteLocalUnix}) {
		t.Fatalf("narrowed supported kinds = %v; want local-unix only", environment.SupportedRouteKinds)
	}

	// Managed needs Cloud availability even when requested.
	environment = sharedRouteEnvironment(context.Background(), target, []clientendpoint.RouteKind{
		clientendpoint.RouteLocalUnix, clientendpoint.RouteManagedWebRTC,
	}, false)
	if routeKindEnabled(environment.SupportedRouteKinds, clientendpoint.RouteManagedWebRTC) {
		t.Fatalf("managed-webrtc must stay pruned without a Cloud client: %v", environment.SupportedRouteKinds)
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

// TestConfigFromSharedEndpointResolvesAutoLocalSocket pins the "all local
// connections go through access" rule: socket "auto" projects to the default
// canonical access socket instead of degrading to the host PTY fallback.
func TestConfigFromSharedEndpointResolvesAutoLocalSocket(t *testing.T) {
	runtimeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	target := clientendpoint.Endpoint{
		ID: "local", Label: "local", Enabled: true, ConnectMode: clientendpoint.ConnectAuto,
		Routes: map[clientendpoint.RouteID]clientendpoint.AccessRoute{
			"local": {
				ID: "local", Kind: clientendpoint.RouteLocalUnix, Enabled: true,
				Source: clientendpoint.SourceLocal, PolicySource: clientendpoint.SourceLocal, Socket: "auto",
			},
		},
	}
	cfg, ok := configFromSharedEndpoint(target)
	if !ok {
		t.Fatal("auto local-unix must project to the default access socket")
	}
	want := filepath.Join(runtimeDir, fmt.Sprintf("anytty-v3-wire%d.sock", wire.Version))
	if cfg.ConnectMode != ConnectLocalUnix || cfg.Socket != want {
		t.Fatalf("auto local config = %+v, want local-unix %s", cfg, want)
	}
}

// TestDefaultRouteKindsRacesAllTransports pins the out-of-the-box policy: a
// configured route of any kind enters the race; the order decides the hedge
// stagger (local first, cloud last).
func TestDefaultRouteKindsRacesAllTransports(t *testing.T) {
	want := []clientendpoint.RouteKind{
		clientendpoint.RouteLocalUnix,
		clientendpoint.RouteSSHWebRTCTCP,
		clientendpoint.RouteDirectWebRTCTCP,
		clientendpoint.RouteManagedWebRTC,
	}
	if got := DefaultRouteKinds(); !reflect.DeepEqual(got, want) {
		t.Fatalf("DefaultRouteKinds() = %v, want all transports %v", got, want)
	}
}

package endpoint

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/anytty/anytty/shared/securefs"
)

func TestLoadMissingDefaultPathReturnsLocalEndpoint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	registry, err := Load("")
	if err != nil {
		t.Fatalf("missing default registry: %v", err)
	}
	endpoint, ok := registry.DefaultEndpoint()
	if !ok || endpoint.ID != DefaultEndpointID || endpoint.ConnectMode != ConnectAuto || !endpoint.Enabled {
		t.Fatalf("unexpected default endpoint %#v, ok=%v", endpoint, ok)
	}
	routes := endpoint.RouteList()
	if len(routes) != 1 || routes[0].Kind != RouteLocalUnix || routes[0].Socket != "auto" {
		t.Fatalf("unexpected default routes %#v", routes)
	}
}

func TestLoadExplicitMissingPathFails(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("explicit missing registry path should fail")
	}
}

func TestNormalizeEmptyRegistryDoesNotInventLocalEndpoint(t *testing.T) {
	registry, err := (Registry{}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if registry.Version != RegistryVersion || registry.Default != "" || len(registry.Endpoints) != 0 {
		t.Fatalf("empty registry changed unexpectedly: %#v", registry)
	}
	if _, err := (Registry{Version: RegistryVersion, Default: "local"}).Normalize(); !IsCode(err, ErrorConfig) {
		t.Fatalf("empty registry with default error=%v", err)
	}
}

func TestNormalizeRejectsNonSerializablePolicyState(t *testing.T) {
	endpoint := NewSSHEndpoint("peer", "Peer", "peer", "", "127.0.0.1:41120", "127.0.0.1:41121", ConnectOnDemand)
	endpoint.SelectionPolicy = SelectionPolicy{HedgeDelay: time.Second, HedgeDelayConfigured: false}
	registry := Registry{Version: RegistryVersion, Default: "peer", Endpoints: map[EndpointID]Endpoint{"peer": endpoint}}
	if _, err := registry.Normalize(); !IsCode(err, ErrorConfig) {
		t.Fatalf("unconfigured non-zero hedge delay error=%v", err)
	}
	endpoint.SelectionPolicy = SelectionPolicy{}
	priority64 := int64(1) << 31
	priority := int(priority64)
	route := endpoint.Routes["ssh"]
	route.Priority = &priority
	endpoint.Routes["ssh"] = route
	registry.Endpoints["peer"] = endpoint
	if _, err := registry.Normalize(); !IsCode(err, ErrorConfig) {
		t.Fatalf("priority overflow error=%v", err)
	}
}

func TestParseEndpointRoutesV2(t *testing.T) {
	registry, err := Parse([]byte(`
version: 3
default: studio
endpoints:
  studio:
    label: Studio
    label_source: user
    device_id: device-studio
    device_fingerprint: SHA256:studio
    enabled: true
    connect_mode: on_demand
    selection:
      hedge_delay: 250ms
    routes:
      cloud:
        kind: managed-webrtc
        enabled: true
        priority: 30
        credential_ref: grant:studio
        source: cloud
        target_device_id: device-studio
        relay_mode: relay_only
      lan:
        kind: direct-webrtc-tcp
        enabled: true
        priority: 10
        credential_ref: grant:studio
        source: bootstrap
        signaling_addresses:
          - 192.0.2.10:41120
          - studio.local:41120
        ice_tcp_addresses:
          - 192.0.2.10:41120
          - studio.local:41120
        server_name: studio.local
      ssh:
        kind: ssh-webrtc-tcp
        enabled: true
        priority: 20
        credential_ref: ssh:studio
        source: manual
        host: studio-host
        port: 22
        user: build
        proxy_jump: bastion
        remote_signaling_address: 127.0.0.1:41120
        remote_ice_tcp_address: 127.0.0.1:41121
        host_key_fingerprints:
          - SHA256:ssh-host
`))
	if err != nil {
		t.Fatalf("parse registry: %v", err)
	}
	studio := registry.Endpoints["studio"]
	if studio.DaemonIdentity != (DaemonIdentity{DeviceID: "device-studio", DeviceFingerprint: "SHA256:studio"}) || !studio.SelectionPolicy.HedgeDelayConfigured || studio.SelectionPolicy.HedgeDelay != 250*time.Millisecond {
		t.Fatalf("unexpected endpoint identity/policy %#v", studio)
	}
	if got := studio.RouteList(); len(got) != 3 || got[0].ID != "cloud" || got[1].ID != "lan" || got[2].ID != "ssh" {
		t.Fatalf("route list is not stable: %#v", got)
	}
	if route := studio.Routes["ssh"]; route.Kind != RouteSSHWebRTCTCP || route.Host != "studio-host" || route.User != "build" || route.ProxyJump != "bastion" || route.RemoteSignalingAddress != "127.0.0.1:41120" || route.RemoteICETCPAddress != "127.0.0.1:41121" {
		t.Fatalf("unexpected ssh route %#v", route)
	}
}

func TestParseRejectsOldUnknownOversizeAndInvalidRegistry(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		code ErrorCode
	}{
		{name: "empty document", data: nil, code: ErrorConfig},
		{name: "missing endpoints", data: []byte("version: 3\ndefault: ''\n"), code: ErrorConfig},
		{name: "missing default", data: []byte("version: 3\nendpoints: {}\n"), code: ErrorConfig},
		{name: "non portable hedge delay", data: []byte("version: 3\ndefault: lab\nendpoints:\n  lab:\n    selection:\n      hedge_delay: 1.5s\n    routes:\n      ssh:\n        kind: ssh-webrtc-tcp\n        host: lab\n"), code: ErrorConfig},
		{name: "old schema", data: []byte("version: 1\nconnections:\n  local:\n    transport: local\n"), code: ErrorConfig},
		{name: "unknown field", data: []byte("version: 3\nendpoints:\n  local:\n    routes:\n      local:\n        kind: local-unix\n        socket: auto\n        surprise: true\n"), code: ErrorConfig},
		{name: "multiple documents", data: []byte("version: 3\nendpoints: {}\n---\nversion: 3\n"), code: ErrorConfig},
		{name: "oversize", data: bytes.Repeat([]byte("x"), MaxRegistryBytes+1), code: ErrorSizeLimit},
		{name: "identity half", data: []byte("version: 3\nendpoints:\n  peer:\n    device_id: device\n    routes:\n      ssh:\n        kind: ssh-webrtc-tcp\n        host: peer\n"), code: ErrorConfig},
		{name: "identity whitespace", data: []byte("version: 3\ndefault: peer\nendpoints:\n  peer:\n    device_id: ' device'\n    device_fingerprint: SHA256:device\n    routes:\n      ssh:\n        kind: ssh-webrtc-tcp\n        host: peer\n"), code: ErrorConfig},
		{name: "direct missing identity", data: []byte("version: 3\nendpoints:\n  peer:\n    routes:\n      lan:\n        kind: direct-webrtc-tcp\n        signaling_addresses: [peer:41120]\n        ice_tcp_addresses: [peer:41120]\n"), code: ErrorConfig},
		{name: "mixed priority", data: []byte("version: 3\nendpoints:\n  peer:\n    routes:\n      one:\n        kind: ssh-webrtc-tcp\n        host: one\n        priority: 10\n      two:\n        kind: ssh-webrtc-tcp\n        host: two\n"), code: ErrorConfig},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.data)
			if err == nil {
				t.Fatal("expected parse failure")
			}
			var connectionErr *Error
			for current := err; current != nil; {
				if value, ok := current.(*Error); ok {
					connectionErr = value
					break
				}
				unwrapper, ok := current.(interface{ Unwrap() error })
				if !ok {
					break
				}
				current = unwrapper.Unwrap()
			}
			if connectionErr == nil || connectionErr.Code != tc.code {
				t.Fatalf("error=%v, want code=%s", err, tc.code)
			}
		})
	}
}

func TestEndpointRuntimeChangeClassification(t *testing.T) {
	endpoint := NewSSHEndpoint("lab", "Lab", "lab.example", "ssh:lab", "127.0.0.1:41120", "127.0.0.1:41121", ConnectOnDemand)
	normalized, err := (Registry{Version: RegistryVersion, Default: "lab", Endpoints: map[EndpointID]Endpoint{"lab": endpoint}}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	endpoint = normalized.Endpoints["lab"]
	renamed := cloneEndpoint(endpoint)
	renamed.Label = "Lab Renamed"
	if !endpoint.DisplayChanged(renamed) || endpoint.RequiresReconnect(renamed) {
		t.Fatal("label change should be display-only")
	}
	policyChanged := cloneEndpoint(endpoint)
	priority := 10
	route := policyChanged.Routes["ssh"]
	route.Priority = &priority
	policyChanged.Routes["ssh"] = route
	if endpoint.RequiresReconnect(policyChanged) {
		t.Fatal("priority change must only affect future selection")
	}
	moved := cloneEndpoint(endpoint)
	route = moved.Routes["ssh"]
	route.Host = "new.example"
	moved.Routes["ssh"] = route
	if !endpoint.RequiresReconnect(moved) {
		t.Fatal("route dial identity change must require reconnect")
	}
}

func TestSaveRoundTripV2AndFileMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "anytty", "endpoints.yaml")
	priority := 10
	endpoint := NewManagedEndpoint("studio", "Studio", DaemonIdentity{DeviceID: "device-studio", DeviceFingerprint: "SHA256:studio"}, "device-studio", "grant:studio", RelayDirect, ConnectOnDemand)
	endpoint.SelectionPolicy = SelectionPolicy{HedgeDelay: 1500 * time.Millisecond, HedgeDelayConfigured: true, RoutePreference: RoutePreferenceManagedCloud}
	endpoint.Routes["lan"] = AccessRoute{ID: "lan", Kind: RouteDirectWebRTCTCP, Enabled: true, Priority: &priority, Source: SourceBootstrap, CredentialRef: "grant:studio", SignalingAddresses: []string{"studio.local:41120"}, ICETCPAddresses: []string{"studio.local:41120"}}
	cloud := endpoint.Routes["cloud"]
	cloud.Priority = intPointer(20)
	cloud.RelayTransport = RelayTransportTCP
	endpoint.Routes["cloud"] = cloud
	registry := Registry{Version: RegistryVersion, Default: "studio", Endpoints: map[EndpointID]Endpoint{"studio": endpoint}}
	if err := Save(path, registry); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range [][]byte{[]byte("connections:"), []byte("capability_grant"), []byte("cloud_token"), []byte("hub_url")} {
		if bytes.Contains(payload, forbidden) {
			t.Fatalf("registry contains forbidden value %q: %s", forbidden, payload)
		}
	}
	if !bytes.Contains(payload, []byte("hedge_delay: 1500ms")) {
		t.Fatalf("registry did not use canonical millisecond duration: %s", payload)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := registry.Normalize()
	if !reflect.DeepEqual(loaded, want) {
		t.Fatalf("round trip mismatch\ngot=%#v\nwant=%#v", loaded, want)
	}
	info, err := os.Stat(path)
	if err != nil || !securefs.IsPrivateFile(path, info) {
		t.Fatalf("registry mode=%v err=%v", info.Mode().Perm(), err)
	}
}

func TestRegistryWritePublishedClassification(t *testing.T) {
	published := &registryWriteError{err: os.ErrInvalid, published: true}
	if !RegistryWritePublished(fmt.Errorf("save registry: %w", published)) {
		t.Fatal("published registry write error lost commit classification")
	}
	if RegistryWritePublished(os.ErrPermission) {
		t.Fatal("pre-publish write error was classified as committed")
	}
}

func TestLoadReadsDefaultPathV2(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	path := filepath.Join(configHome, "anytty", DefaultFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("version: 3\ndefault: local\nendpoints:\n  local:\n    label: Configured Local\n    enabled: true\n    connect_mode: auto\n    routes:\n      local:\n        kind: local-unix\n        enabled: true\n        socket: auto\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	registry, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if registry.Endpoints[DefaultEndpointID].Label != "Configured Local" {
		t.Fatalf("unexpected registry %#v", registry)
	}
}

func TestLoadDefaultPathIgnoresLegacyConnectionsV1(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	dir := filepath.Join(configHome, "anytty")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := []byte("version: 1\ndefault: local\nconnections:\n  local:\n    transport: local\n")
	if err := os.WriteFile(filepath.Join(dir, "connections.yaml"), legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	registry, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	local, ok := registry.Endpoints[DefaultEndpointID]
	if !ok || registry.Default != DefaultEndpointID {
		t.Fatalf("legacy file blocked default local registry: %#v", registry)
	}
	if route, exists := local.Route(DefaultLocalRouteID); !exists || route.Kind != RouteLocalUnix {
		t.Fatalf("default local route = %#v", local.Routes)
	}
	if got := filepath.Base(DefaultPath()); got != "endpoints.yaml" {
		t.Fatalf("default registry file = %q", got)
	}
}

func intPointer(value int) *int { return &value }

package endpoint

import (
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
)

// TestSharedRegistryPathIsClientOwned pins the CLI/TUI single connection
// source of truth: tui2 must read client/endpoint's path, not invent one.
func TestSharedRegistryPathIsClientOwned(t *testing.T) {
	if got, want := SharedRegistryPath(), clientendpoint.DefaultPath(); got != want {
		t.Fatalf("SharedRegistryPath() = %q, want client/endpoint.DefaultPath() %q", got, want)
	}
	if !strings.HasSuffix(SharedRegistryPath(), filepath.Join("anytty", clientendpoint.DefaultFileName)) {
		t.Fatalf("shared registry path %q does not end in anytty/%s", SharedRegistryPath(), clientendpoint.DefaultFileName)
	}
}

// TestSharedConfigForEndpointReadsRegistry proves the tui2 projection of a
// CLI-written endpoints.yaml: direct routes carry the daemon pin and
// signaling/ICE-TCP locators, local-unix routes fall back to their socket.
func TestSharedConfigForEndpointReadsRegistry(t *testing.T) {
	path := writeTestRegistry(t)
	direct, ok, err := sharedConfigForEndpointAt(path, "dev")
	if err != nil || !ok {
		t.Fatalf("sharedConfigForEndpointAt(dev) = %+v, %v, %v", direct, ok, err)
	}
	if direct.Kind != KindDaemon || direct.ConnectMode != ConnectDirectWebRTC {
		t.Fatalf("dev config = %+v, want daemon/direct-webrtc-tcp", direct)
	}
	if direct.DaemonDeviceID != "device-dev" || direct.DaemonFingerprint != "ed25519-sha256:dev" {
		t.Fatalf("dev identity = %q/%q", direct.DaemonDeviceID, direct.DaemonFingerprint)
	}
	if len(direct.SignalingAddresses) != 1 || direct.SignalingAddresses[0] != "127.0.0.1:17001" ||
		len(direct.ICETCPAddresses) != 1 || direct.ICETCPAddresses[0] != "127.0.0.1:17002" {
		t.Fatalf("dev locators = %v / %v", direct.SignalingAddresses, direct.ICETCPAddresses)
	}
	local, ok, err := sharedConfigForEndpointAt(path, "local")
	if err != nil || !ok {
		t.Fatalf("sharedConfigForEndpointAt(local) = %+v, %v, %v", local, ok, err)
	}
	if local.ConnectMode != ConnectLocalUnix || local.Socket != "/tmp/shared-local.sock" {
		t.Fatalf("local config = %+v", local)
	}
	if _, ok, err := sharedConfigForEndpointAt(path, "absent"); err != nil || ok {
		t.Fatalf("absent endpoint = ok %v, err %v; want false/nil", ok, err)
	}
}

// TestSharedConfigForEndpointMissingFileIsNotImplicitDefault: tui2 must not
// inject client/endpoint.DefaultRegistry into the UI when endpoints.yaml is
// absent; only an explicit user registry is read.
func TestSharedConfigForEndpointMissingFileIsNotImplicitDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", clientendpoint.DefaultFileName)
	if cfg, ok, err := sharedConfigForEndpointAt(path, "local"); err != nil || ok {
		t.Fatalf("missing registry = %+v, %v, %v; want zero/false/nil", cfg, ok, err)
	}
	configs, warnings := LoadSharedEndpointConfigs(path)
	if len(configs) != 0 || len(warnings) != 1 || !strings.Contains(warnings[0], "does not exist") {
		t.Fatalf("LoadSharedEndpointConfigs(missing) = %v, %v; want an empty list and one readable missing-file warning", configs, warnings)
	}
}

// TestLoadSharedEndpointConfigsListsRepresentableRoutes covers the read-only
// listing used by the shared-registry compatibility path.
func TestLoadSharedEndpointConfigsListsRepresentableRoutes(t *testing.T) {
	configs, warnings := LoadSharedEndpointConfigs(writeTestRegistry(t))
	if len(warnings) != 0 {
		t.Fatalf("LoadSharedEndpointConfigs: %v", warnings)
	}
	byName := map[string]Config{}
	for _, cfg := range configs {
		byName[cfg.Name] = cfg
	}
	if len(byName) != 2 || byName["dev"].ConnectMode != ConnectDirectWebRTC || byName["local"].Socket != "/tmp/shared-local.sock" {
		t.Fatalf("shared configs = %+v", configs)
	}
}

// TestAllDaemonModesDialThroughSharedLayer pins the final M1 boundary: every
// recognized daemon connect mode is dialable through the shared client layer.
// Missing direct/cloud route parameters are not a tui2 scope error anymore:
// endpointFromConfig surfaces a readable route validation error instead.
func TestAllDaemonModesDialThroughSharedLayer(t *testing.T) {
	for _, mode := range []string{ConnectLocalUnix, ConnectDirectTCP, ConnectDirectWebRTC, ConnectManagedWebRTC} {
		cfg := Config{Name: "remote", Kind: KindDaemon, ConnectMode: mode}
		if err := cfg.UnsupportedModeError(); err != nil {
			t.Fatalf("%s UnsupportedModeError = %v, want nil (shared layer owns all modes)", mode, err)
		}
	}
	if err := (Config{Name: "bogus", Kind: KindDaemon, ConnectMode: "bogus"}).UnsupportedModeError(); err == nil {
		t.Fatal("unknown connect_mode must stay a readable error")
	}

	if _, err := endpointFromConfig(Config{Name: "local", Kind: KindDaemon, ConnectMode: ConnectLocalUnix}, ""); err == nil || !strings.Contains(err.Error(), "socket") {
		t.Fatalf("local-unix without socket error = %v", err)
	}
	target, err := endpointFromConfig(Config{Name: "local", Kind: KindDaemon, ConnectMode: ConnectLocalUnix, Socket: "/tmp/dev.sock"}, "")
	if err != nil {
		t.Fatalf("local endpointFromConfig: %v", err)
	}
	if route, ok := target.Route(clientendpoint.RouteID("local")); !ok || route.Kind != clientendpoint.RouteLocalUnix || route.Socket != "/tmp/dev.sock" {
		t.Fatalf("local target = %+v", target)
	}
	tcpTarget, err := endpointFromConfig(Config{Name: "tcp", Kind: KindDaemon, ConnectMode: ConnectDirectTCP, Address: "127.0.0.1:1"}, "/tmp/bridge.sock")
	if err != nil {
		t.Fatalf("tcp endpointFromConfig: %v", err)
	}
	if route := tcpTarget.Routes[clientendpoint.RouteID("local")]; route.Kind != clientendpoint.RouteLocalUnix || route.Socket != "/tmp/bridge.sock" {
		t.Fatalf("tcp target route = %+v; want the bridge socket on the shared local route", route)
	}
	if _, err := endpointFromConfig(Config{Name: "direct", Kind: KindDaemon, ConnectMode: ConnectDirectWebRTC}, ""); err == nil || !strings.Contains(err.Error(), "signaling") {
		t.Fatalf("direct without locators error = %v; want readable signaling/ICE-TCP error", err)
	}
}

// TestSharedPlanPrefersCLIRegistryOverTui2Config is the M2 boundary: an
// endpoint written by the CLI wins over the same-name tui2.json fields.
func TestSharedPlanPrefersCLIRegistryOverTui2Config(t *testing.T) {
	path := writeTestRegistry(t)
	cfg := Config{Name: "local", Kind: KindDaemon, ConnectMode: ConnectLocalUnix, Socket: "/tmp/tui2-json.sock"}
	snapshot, err := sharedPlanSnapshot(t.Context(), []string{path}, cfg, "", false, nil)
	if err != nil {
		t.Fatalf("sharedPlanSnapshot: %v", err)
	}
	if snapshot.Endpoint.ID != "local" {
		t.Fatalf("target id = %q", snapshot.Endpoint.ID)
	}
	route, ok := snapshot.Endpoint.Route("local")
	if !ok || route.Socket != "/tmp/shared-local.sock" {
		t.Fatalf("route = %+v; want the CLI registry socket", route)
	}
	if snapshot.ConfigKey == "" {
		t.Fatal("plan snapshot must carry a config key")
	}
}

// TestTCPBridgeRelaysBytesAndCleansUp proves the tcp compatibility adapter is
// byte-transparent and releases its unix socket directory.
func TestTCPBridgeRelaysBytesAndCleansUp(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 4)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return
		}
		_, _ = conn.Write([]byte("pong"))
	}()
	bridge, err := startTCPBridge(t.Context(), ln.Addr().String(), "tcp")
	if err != nil {
		t.Fatalf("startTCPBridge: %v", err)
	}
	conn, err := net.Dial("unix", bridge.Socket())
	if err != nil {
		t.Fatalf("dial bridge socket: %v", err)
	}
	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}
	reply := make([]byte, 4)
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatalf("read: %v", err)
	}
	_ = conn.Close()
	if string(reply) != "pong" {
		t.Fatalf("reply = %q", reply)
	}
	dir := filepath.Dir(bridge.Socket())
	if err := bridge.Close(); err != nil {
		t.Fatalf("bridge close: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("bridge dir %s still exists after close", dir)
	}
}

// TestTui2HasNoRawProtocolDialer is the M5 anti-regression guard: the tui2
// endpoint package must import the shared connection packages and must not
// import the raw framing/dialer implementation it used to duplicate.
func TestTui2HasNoRawProtocolDialer(t *testing.T) {
	goTool := goToolPath(t)
	root := repositoryRoot(t)
	command := exec.Command(goTool, "list", "-f", "{{join .Imports \"\\n\"}}", "./clients/tui/endpoint", "./clients/tui/cmd/tui2")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("go list imports: %v\n%s", err, exitErr.Stderr)
		}
		t.Fatalf("go list imports: %v", err)
	}
	imports := string(output)
	for _, required := range []string{
		"github.com/anytty/anytty/access/engine/adapter/protocol",
		"github.com/anytty/anytty/access/engine/adapter/local",
		"github.com/anytty/anytty/access/engine/endpoint",
		"github.com/anytty/anytty/access/engine/runtime",
	} {
		if !strings.Contains(imports, required) {
			t.Errorf("tui2 must import shared connection package %q", required)
		}
	}
	for _, forbidden := range []string{
		"github.com/anytty/anytty/internal/protocol",
		"github.com/anytty/anytty/proto/access/wirepb",
		"github.com/anytty/anytty/shared/transport/unix",
		"github.com/klauspost/compress/zstd",
		"golang.org/x/crypto/ssh",
	} {
		for _, line := range strings.Split(imports, "\n") {
			if strings.TrimSpace(line) == forbidden {
				t.Errorf("tui2 imports %q directly; raw dialing/framing belongs to client/ (see CLIENT_SHARING.zh-CN.md)", forbidden)
			}
		}
	}
}

// TestTui2DependencyClosureUsesSharedAdapters proves the linked dependency
// closure really contains the shared dialers (not a tui2 reimplementation).
func TestTui2DependencyClosureUsesSharedAdapters(t *testing.T) {
	goTool := goToolPath(t)
	root := repositoryRoot(t)
	command := exec.Command(goTool, "list", "-deps", "./clients/tui/endpoint", "./clients/tui/cmd/tui2")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("go list -deps: %v\n%s", err, exitErr.Stderr)
		}
		t.Fatalf("go list -deps: %v", err)
	}
	deps := "\n" + string(output) + "\n"
	for _, required := range []string{
		"github.com/anytty/anytty/access/engine/adapter/local",
		"github.com/anytty/anytty/access/engine/adapter/protocol",
		"github.com/anytty/anytty/access/engine/runtime",
	} {
		if !strings.Contains(deps, "\n"+required+"\n") {
			t.Errorf("dependency closure misses shared adapter %q", required)
		}
	}
}

// TestTui2EndpointDeduplicationEvidence records the final M1 line-count drop:
// the raw wire client and the tui2-owned tcp framing were replaced by the
// shared session adapter plus a byte-transparent tcp relay.
func TestTui2EndpointDeduplicationEvidence(t *testing.T) {
	const (
		baselineTotalLines = 4952
		baselineProdLines  = 3642
	)
	dir := packageDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	total, prod := 0, 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		switch entry.Name() {
		case "client.go", "transport_tcp.go", "transport_direct.go", "transport_cloud.go", "transport_direct_pion.go":
			t.Errorf("removed dialer file %s must not return; daemon dialing is owned by client/", entry.Name())
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		lines := len(strings.Split(strings.TrimSuffix(string(data), "\n"), "\n"))
		total += lines
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			prod += lines
		}
	}
	for _, required := range []string{"shared_session.go", "shared_runtime.go", "session.go", "tcp_bridge.go"} {
		if _, err := os.Stat(filepath.Join(dir, required)); err != nil {
			t.Errorf("shared adapter file %s is missing: %v", required, err)
		}
	}
	if prod >= baselineProdLines {
		t.Fatalf("clients/tui/endpoint production code is %d lines; want fewer than the %d-line baseline", prod, baselineProdLines)
	}
	t.Logf("clients/tui/endpoint dedup: total %d -> %d, production %d -> %d (-%d)",
		baselineTotalLines, total, baselineProdLines, prod, baselineProdLines-prod)
}

func writeTestRegistry(t *testing.T) string {
	t.Helper()
	registry := clientendpoint.Registry{
		Version: clientendpoint.RegistryVersion,
		Default: "dev",
		Endpoints: map[clientendpoint.EndpointID]clientendpoint.Endpoint{
			"dev": {
				ID: "dev", Label: "dev", LabelSource: clientendpoint.SourceManual,
				ConnectMode: clientendpoint.ConnectAuto, Enabled: true,
				DaemonIdentity: clientendpoint.DaemonIdentity{DeviceID: "device-dev", DeviceFingerprint: "ed25519-sha256:dev"},
				Routes: map[clientendpoint.RouteID]clientendpoint.AccessRoute{
					"direct": {
						ID: "direct", Kind: clientendpoint.RouteDirectWebRTCTCP, Enabled: true,
						Source: clientendpoint.SourceManual, PolicySource: clientendpoint.SourceManual,
						SignalingAddresses: []string{"127.0.0.1:17001"}, ICETCPAddresses: []string{"127.0.0.1:17002"},
					},
				},
			},
			"local": {
				ID: "local", Label: "local", LabelSource: clientendpoint.SourceManual,
				ConnectMode: clientendpoint.ConnectAuto, Enabled: true,
				Routes: map[clientendpoint.RouteID]clientendpoint.AccessRoute{
					"local": {
						ID: "local", Kind: clientendpoint.RouteLocalUnix, Enabled: true,
						Source: clientendpoint.SourceManual, PolicySource: clientendpoint.SourceManual, Socket: "/tmp/shared-local.sock",
					},
				},
			},
		},
	}
	normalized, err := registry.Normalize()
	if err != nil {
		t.Fatalf("normalize test registry: %v", err)
	}
	payload, err := clientendpoint.Encode(normalized)
	if err != nil {
		t.Fatalf("encode test registry: %v", err)
	}
	path := filepath.Join(t.TempDir(), clientendpoint.DefaultFileName)
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func goToolPath(t *testing.T) string {
	t.Helper()
	if tool, err := exec.LookPath("go"); err == nil {
		return tool
	}
	tool := filepath.Join(runtime.GOROOT(), "bin", "go")
	if _, err := os.Stat(tool); err != nil {
		t.Skipf("go tool unavailable: %v", err)
	}
	return tool
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root (go.mod) not found")
		}
		dir = parent
	}
}

func packageDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

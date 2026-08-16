package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/shared/filelock"
	"github.com/anytty/anytty/shared/securefs"
)

func TestEndpointMutationHonorsRootTimeoutWhileRegistryLocked(t *testing.T) {
	registryPath := filepath.Join(t.TempDir(), "endpoints.yaml")
	owner, err := filelock.Acquire(registryPath+".lock", true)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	command := newRootCmd()
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs([]string{
		"--timeout", "100ms", "endpoint", "--registry", registryPath,
		"add", "ssh", "blocked", "--host", "blocked.example",
	})
	started := time.Now()
	err = command.Execute()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("locked registry timeout error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("locked registry ignored root timeout: %s", elapsed)
	}
}

func TestEndpointRegistryCommandLifecycle(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	run := func(args ...string) string {
		t.Helper()
		command := newRootCmd()
		var output bytes.Buffer
		command.SetOut(&output)
		command.SetErr(io.Discard)
		command.SetArgs(args)
		if err := command.Execute(); err != nil {
			t.Fatalf("anytty %s: %v", strings.Join(args, " "), err)
		}
		return output.String()
	}

	run("endpoint", "add", "ssh", "west", "--host", "root@west.example", "--credential-ref", "ssh:west")
	run("endpoint", "add", "cloud", "studio", "--device-id", "device-studio", "--device-fingerprint", "SHA256:studio", "--target-device-id", "device-studio", "--credential-ref", "grant:studio", "--relay", "relay_only")
	run("endpoint", "update", "west", "--label", "West build host")
	identityUpdate := newRootCmd()
	identityUpdate.SetOut(io.Discard)
	identityUpdate.SetErr(io.Discard)
	identityUpdate.SetArgs([]string{"endpoint", "update", "west", "--device-id", "device-west", "--device-fingerprint", "SHA256:west"})
	if err := identityUpdate.Execute(); err == nil || !strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("generic endpoint update must not mutate daemon identity: %v", err)
	}
	run("endpoint", "route", "update", "west", "ssh", "--remote-signaling-address", "127.0.0.1:42120", "--remote-ice-tcp-address", "127.0.0.1:42121")
	run("endpoint", "add", "direct", "lan", "--device-id", "device-lan", "--device-fingerprint", "SHA256:lan", "--credential-ref", "grant:lan", "--direct-address", "192.168.1.8:41120")
	run("endpoint", "set-default", "west")

	listing := run("endpoint", "list", "--json")
	for _, expected := range []string{`"id":"local"`, `"id":"west"`, `"id":"studio"`, `"default":true`, `"kind":"managed-webrtc"`} {
		if !strings.Contains(listing, expected) {
			t.Fatalf("endpoint list missing %s: %s", expected, listing)
		}
	}
	humanListing := run("endpoint", "list")
	if strings.Contains(humanListing, "\t") || !strings.Contains(humanListing, "STATUS") || !strings.Contains(humanListing, "DEFAULT") || !strings.Contains(humanListing, "West build host") {
		t.Fatalf("endpoint human list is not aligned table output:\n%s", humanListing)
	}
	shown := run("endpoint", "show", "west", "--json")
	if !strings.Contains(shown, `"label":"West build host"`) || !strings.Contains(shown, `"credential_ref":"ssh:west"`) {
		t.Fatalf("unexpected endpoint show: %s", shown)
	}
	lan := run("endpoint", "show", "lan", "--json")
	for _, expected := range []string{`"signaling_addresses":["192.168.1.8:41120"]`, `"ice_tcp_addresses":["192.168.1.8:41120"]`} {
		if !strings.Contains(lan, expected) {
			t.Fatalf("single-address Direct endpoint missing %s: %s", expected, lan)
		}
	}
	policy := run("endpoint", "policy", "set", "studio", "--route", "cloud", "--cloud-path", "relay", "--relay-transport", "tcp", "--json")
	for _, expected := range []string{`"route":"cloud"`, `"cloud_path":"relay"`, `"relay_transport":"tcp"`} {
		if !strings.Contains(policy, expected) {
			t.Fatalf("endpoint policy output missing %s: %s", expected, policy)
		}
	}
	shownPolicy := run("endpoint", "policy", "show", "studio", "--json")
	if shownPolicy != policy {
		t.Fatalf("persisted policy changed after reload: set=%s show=%s", policy, shownPolicy)
	}
	run("endpoint", "update", "studio", "--hedge-delay", "750ms")
	if afterUpdate := run("endpoint", "policy", "show", "studio", "--json"); afterUpdate != policy {
		t.Fatalf("unrelated endpoint update reset connection policy: set=%s show=%s", policy, afterUpdate)
	}

	run("endpoint", "disable", "west")
	registry, err := endpointdomain.Load("")
	if err != nil {
		t.Fatal(err)
	}
	registry, err = registry.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if registry.Default != endpointdomain.DefaultEndpointID || registry.Endpoints["west"].Enabled {
		t.Fatalf("disable did not select enabled default: %#v", registry)
	}
	run("endpoint", "enable", "west")
	run("endpoint", "remove", "studio")
	if _, ok := mustLoadEndpointRegistry(t).Endpoints["studio"]; ok {
		t.Fatal("removed endpoint is still present")
	}

	info, err := os.Stat(filepath.Join(configHome, "anytty", endpointdomain.DefaultFileName))
	if err != nil {
		t.Fatal(err)
	}
	registryPath := filepath.Join(configHome, "anytty", endpointdomain.DefaultFileName)
	if !securefs.IsPrivateFile(registryPath, info) {
		t.Fatalf("registry permissions are not private: %v", info.Mode())
	}
}

func TestEndpointTestViewIncludesSelectedPairAddresses(t *testing.T) {
	sampledAt := time.Date(2026, 7, 27, 12, 0, 0, 123, time.UTC)
	view := endpointTestViewFromSnapshot("studio", endpointdomain.AccessRoute{ID: "cloud", Kind: endpointdomain.RouteManagedWebRTC}, "single_relay", "only_viable", clientruntime.ConnectionSnapshot{
		ObservedPath: "direct", SampledAt: sampledAt, RoundTrip: 42*time.Millisecond + 500*time.Microsecond,
		LocalAddress: "192.0.2.10", RemoteAddress: "2001:db8::20", LocalPort: 41000, RemotePort: 41121,
		LocalCandidateType: "srflx", RemoteCandidateType: "relay", LocalProtocol: "udp", RemoteProtocol: "udp", RelayTransport: "tcp",
		BytesSent: 10, BytesReceived: 20, Connected: true,
	}, true)
	if !view.SnapshotAvailable || view.ObservedPath != "direct" || view.LocalIP != "192.0.2.10" || view.RemoteIP != "2001:db8::20" || view.RoundTripMillis != 42.5 {
		t.Fatalf("endpoint test view = %#v", view)
	}
	if got := formatCLIEndpoint(view.RemoteIP, view.RemotePort); got != "[2001:db8::20]:41121" {
		t.Fatalf("formatted remote endpoint = %q", got)
	}
}

func TestEndpointAddCreatesExplicitRegistryWithoutInventingLocalEndpoint(t *testing.T) {
	registryPath := filepath.Join(t.TempDir(), "custom", "endpoints.yaml")
	command := newRootCmd()
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"endpoint", "--registry", registryPath, "add", "ssh", "west", "--host", "west.example"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	registry, err := endpointdomain.Load(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Endpoints) != 1 || registry.Default != "west" {
		t.Fatalf("explicit registry should contain only the imported endpoint: %#v", registry)
	}
	if _, exists := registry.Endpoints[endpointdomain.DefaultEndpointID]; exists {
		t.Fatal("explicit registry creation invented a local endpoint")
	}
}

func TestTerminalCommandsRouteDuplicateIDsToOwningLocalEndpoint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	localSocket, localClient, closeLocal := startCLIEndpointServer(t)
	defer closeLocal()
	westSocket, westClient, closeWest := startCLIEndpointServer(t)
	defer closeWest()

	for _, client := range []*protocol.Client{localClient, westClient} {
		if _, err := createCLIProtoTerminal(context.Background(), client, &apipb.TerminalCreateSpec{
			TerminalId: "same", Name: "same", Command: testShellSleepCommand(), Size: &apipb.TerminalSize{Cols: 80, Rows: 24},
		}); err != nil {
			t.Fatal(err)
		}
	}
	registry := endpointdomain.Registry{
		Version: endpointdomain.RegistryVersion, Default: "local",
		Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{
			"local": testLocalEndpoint("local", "Local", localSocket, endpointdomain.ConnectAuto, true),
			"west":  testLocalEndpoint("west", "West", westSocket, endpointdomain.ConnectOnDemand, true),
		},
	}
	if err := endpointdomain.Save("", registry); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) string {
		t.Helper()
		command := newRootCmd()
		var output bytes.Buffer
		command.SetOut(&output)
		command.SetErr(io.Discard)
		command.SetArgs(args)
		if err := command.Execute(); err != nil {
			t.Fatalf("anytty %s: %v", strings.Join(args, " "), err)
		}
		return output.String()
	}
	listing := run("terminal", "list", "--all-endpoints", "--json")
	if strings.Count(listing, `"terminal_id":"same"`) != 2 || !strings.Contains(listing, `"target":"local:same"`) || !strings.Contains(listing, `"target":"west:same"`) {
		t.Fatalf("duplicate terminal IDs lost endpoint ownership: %s", listing)
	}
	shown := run("terminal", "show", "west:same", "--json")
	if !strings.Contains(shown, `"target":"west:same"`) || strings.Contains(shown, `"target":"local:same"`) {
		t.Fatalf("show routed to wrong endpoint: %s", shown)
	}
}

func startCLIEndpointServer(t *testing.T) (string, *protocol.Client, func()) {
	t.Helper()
	socketPath := filepath.Join(t.TempDir(), "anytty.sock")
	server := newCoreV2TestServer(corev2.WithSocketPath(socketPath))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe(ctx) }()
	if err := waitForSocket(socketPath, 2*time.Second, func() error {
		client, err := dialV3Client(socketPath)
		if err != nil {
			return err
		}
		return client.Close()
	}); err != nil {
		cancel()
		t.Fatalf("endpoint server did not become ready: %v", err)
	}
	client, err := dialV3Client(socketPath)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	return socketPath, client, func() {
		_ = client.Close()
		cancel()
		_ = server.Shutdown(context.Background())
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("endpoint server did not stop")
		}
	}
}

func mustLoadEndpointRegistry(t *testing.T) endpointdomain.Registry {
	t.Helper()
	registry, err := endpointdomain.Load("")
	if err != nil {
		t.Fatal(err)
	}
	registry, err = registry.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

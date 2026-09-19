package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	endpointdomain "github.com/anytty/anytty/access/engine/endpoint"
	"github.com/anytty/anytty/proto/access/wire"
	pionlogging "github.com/pion/logging"
	"google.golang.org/grpc/grpclog"
)

func TestResolveLogFilePathPrefersExplicitValue(t *testing.T) {
	t.Setenv("ANYTTY_LOG_FILE", filepath.Join(t.TempDir(), "ignored.log"))
	got := resolveLogFilePath("/tmp/anytty-explicit.log")
	if got != "/tmp/anytty-explicit.log" {
		t.Fatalf("expected explicit log path to win, got %q", got)
	}
}

func TestResolveLogFilePathUsesEnvironmentOverride(t *testing.T) {
	want := filepath.Join(t.TempDir(), "anytty-env.log")
	t.Setenv("ANYTTY_LOG_FILE", want)
	if got := resolveLogFilePath(""); got != want {
		t.Fatalf("expected ANYTTY_LOG_FILE path %q, got %q", want, got)
	}
}

func TestResolveLogFilePathFallsBackToXDGStateHome(t *testing.T) {
	base := t.TempDir()
	t.Setenv("ANYTTY_LOG_FILE", "")
	t.Setenv("XDG_STATE_HOME", base)
	got := resolveLogFilePath("")
	want := filepath.Join(base, "anytty", "anytty.log")
	if got != want {
		t.Fatalf("expected XDG fallback %q, got %q", want, got)
	}
}

func TestResolveWorkspaceStatePathFallsBackToXDGStateHome(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", base)
	got := resolveWorkspaceStatePath()
	want := filepath.Join(base, "anytty", "workspace-state.json")
	if got != want {
		t.Fatalf("expected workspace state path %q, got %q", want, got)
	}
}

func TestResolveGridStatePathPrefersEnvironmentOverride(t *testing.T) {
	want := filepath.Join(t.TempDir(), "grid")
	t.Setenv("ANYTTY_GRID_DIR", want)
	if got := resolveGridStatePath(); got != want {
		t.Fatalf("expected ANYTTY_GRID_DIR path %q, got %q", want, got)
	}
}

func TestResolveGridStatePathFallsBackToXDGStateHome(t *testing.T) {
	base := t.TempDir()
	t.Setenv("ANYTTY_GRID_DIR", "")
	t.Setenv("XDG_STATE_HOME", base)
	got := resolveGridStatePath()
	want := filepath.Join(base, "anytty", "grid")
	if got != want {
		t.Fatalf("expected grid state path %q, got %q", want, got)
	}
}

func TestV3PathPolicy(t *testing.T) {
	runtimeDir := t.TempDir()
	stateHome := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	t.Setenv("XDG_STATE_HOME", stateHome)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if got := resolveV3Socket(""); got != filepath.Join(runtimeDir, fmt.Sprintf("anytty-v2-wire%d.sock", wire.Version)) {
		t.Fatalf("expected v3 socket in runtime dir, got %q", got)
	}
	explicitSocket := filepath.Join(t.TempDir(), "explicit.sock")
	if got := resolveV3Socket(explicitSocket); got != explicitSocket {
		t.Fatalf("expected explicit v3 socket to win, got %q", got)
	}
	if got := resolveV3LogFilePath(""); got != filepath.Join(stateHome, "anytty", "anytty.log") {
		t.Fatalf("expected v3 log path to reuse global log policy, got %q", got)
	}
	if got := resolveV3ClipboardStoragePath(); got != filepath.Join(stateHome, "anytty", "clipboard-history.json") {
		t.Fatalf("expected clipboard history in host state dir, got %q", got)
	}
	if got := poolConfigDefaultPath(); got != filepath.Join(configHome, "anytty", "tui-v3.yaml") {
		t.Fatalf("expected pool config path to stay tui-v3.yaml, got %q", got)
	}
}

func TestResolveV3SocketRequiresRegisteredLocalRoute(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	if socket, ok, err := resolveTUI2LocalSocket(endpointdomain.DefaultRegistry(), string(endpointdomain.DefaultEndpointID)); err != nil || !ok || socket == "" {
		t.Fatalf("default local registry socket = %q ok=%v err=%v", socket, ok, err)
	}
	if _, ok, err := resolveTUI2LocalSocket(endpointdomain.Registry{}, string(endpointdomain.DefaultEndpointID)); err != nil || ok {
		t.Fatalf("empty registry must not resolve a local pool socket: ok=%v err=%v", ok, err)
	}
	remote := endpointdomain.NewSSHEndpoint("remote", "Remote", "remote.example", "", "127.0.0.1:41120", "127.0.0.1:41121", endpointdomain.ConnectOnDemand)
	registry := endpointdomain.Registry{Version: endpointdomain.RegistryVersion, Default: "remote", Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{"remote": remote}}
	if _, ok, err := resolveTUI2LocalSocket(registry, string(registry.Default)); err != nil || ok {
		t.Fatal("remote-only registry must not fall back to the default local socket")
	}
	mixed := endpointdomain.DefaultRegistry()
	mixed.Endpoints["remote"] = remote
	mixed.Default = "remote"
	if _, ok, err := resolveTUI2LocalSocket(mixed, string(mixed.Default)); err != nil || ok {
		t.Fatal("mixed registry must honor its remote default instead of falling back to the local endpoint")
	}
}

func TestV3AttachDoesNotCreateTuiv2Config(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRun := runTUI2
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runTUI2 = oldRun
	})

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	isInteractiveTerminal = func() bool { return true }
	runTUI2 = func(context.Context, tui2EntryConfig) error { return nil }

	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"--log-file", filepath.Join(t.TempDir(), "anytty.log"), "v3", "attach", "term-1"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("v3 attach returned error: %v", err)
	}
	configPath := filepath.Join(configHome, "anytty", "anytty.yaml")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("v3 attach must not create tuiv2 config at %s, stat err=%v", configPath, err)
	}
}
func TestV3AttachForwardsConfigPathToTUI2(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRun := runTUI2
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runTUI2 = oldRun
	})

	isInteractiveTerminal = func() bool { return true }
	configPath := filepath.Join(t.TempDir(), "tui2.json")
	if err := os.WriteFile(configPath, []byte("{\"theme\":\"dark\"}\n"), 0o600); err != nil {
		t.Fatalf("write tui2 config: %v", err)
	}
	var got tui2EntryConfig
	runTUI2 = func(_ context.Context, cfg tui2EntryConfig) error {
		got = cfg
		return nil
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--config", configPath, "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "attach", "term-1"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("v3 attach returned error: %v", err)
	}
	if got.ConfigPath != configPath {
		t.Fatalf("expected v3 attach to forward --config %q, got %#v", configPath, got)
	}
}
func TestV3RootForwardsConfigPathToTUI2(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRun := runTUI2
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runTUI2 = oldRun
	})

	isInteractiveTerminal = func() bool { return true }
	configPath := filepath.Join(t.TempDir(), "tui2.json")
	var got tui2EntryConfig
	runTUI2 = func(_ context.Context, cfg tui2EntryConfig) error {
		got = cfg
		return nil
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--config", configPath, "--log-file", filepath.Join(t.TempDir(), "anytty.log")})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("root command returned error: %v", err)
	}
	if got.ConfigPath != configPath {
		t.Fatalf("expected root to forward --config %q, got %#v", configPath, got)
	}
}

func TestOpenLogFileLoggerCreatesFileAndWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "anytty.log")
	logger, closeFn, resolved, err := openLogFileLogger(path)
	if err != nil {
		t.Fatalf("openLogFileLogger returned error: %v", err)
	}
	defer closeFn()

	if resolved != path {
		t.Fatalf("expected resolved path %q, got %q", path, resolved)
	}

	logger.Info("hello-log", "component", "test")
	if err := closeFn(); err != nil {
		t.Fatalf("closeFn returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "hello-log") || !strings.Contains(text, "component=test") {
		t.Fatalf("expected log file to contain structured record, got:\n%s", text)
	}
}

func TestOpenLogFileLoggerRoutesProcessLogAndRestoresOutput(t *testing.T) {
	var originalOutput bytes.Buffer
	previousOutput := log.Writer()
	previousStderr := os.Stderr
	log.SetOutput(&originalOutput)
	t.Cleanup(func() { log.SetOutput(previousOutput) })

	path := filepath.Join(t.TempDir(), "logs", "anytty.log")
	_, closeFn, _, err := openLogFileLogger(path)
	if err != nil {
		t.Fatalf("openLogFileLogger returned error: %v", err)
	}
	log.Print("connection-stage-in-file")
	grpclog.Error("grpc-connection-error-in-file")
	fmt.Fprintln(os.Stderr, "direct-stderr-in-file")
	pionlogging.NewDefaultLoggerFactory().NewLogger("turnc").Error("Fail to refresh permissions: retransmissions failed")
	if err := closeFn(); err != nil {
		t.Fatalf("closeFn returned error: %v", err)
	}
	if os.Stderr != previousStderr {
		t.Fatal("process stderr was not restored")
	}
	log.Print("command-finished-on-original-output")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(data), "connection-stage-in-file") {
		t.Fatalf("standard library log was not routed to file: %s", data)
	}
	if !strings.Contains(string(data), "grpc-connection-error-in-file") {
		t.Fatalf("gRPC log was not routed to file: %s", data)
	}
	if !strings.Contains(string(data), "direct-stderr-in-file") {
		t.Fatalf("direct stderr was not routed to file: %s", data)
	}
	if !strings.Contains(string(data), "turnc ERROR:") || !strings.Contains(string(data), "Fail to refresh permissions") {
		t.Fatalf("Pion default logger was not routed to file: %s", data)
	}
	if strings.Contains(originalOutput.String(), "connection-stage-in-file") {
		t.Fatalf("standard library log leaked to original output: %s", originalOutput.String())
	}
	if !strings.Contains(originalOutput.String(), "command-finished-on-original-output") {
		t.Fatalf("standard library output was not restored: %s", originalOutput.String())
	}
}

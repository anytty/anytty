package main

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

	endpointdomain "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/wire"
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
	if got := v3ConfigPathPolicy(); got != filepath.Join(configHome, "anytty", "tui-v3.yaml") {
		t.Fatalf("expected v3 config path policy to use tui-v3.yaml, got %q", got)
	}
	if got := v3StatePathPolicy(); got != "unused" {
		t.Fatalf("expected v3 state path policy unused, got %q", got)
	}
}

func TestResolveV3SocketRequiresRegisteredLocalRoute(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	if socket, err := resolveV3SocketForConnectionRegistry("", endpointdomain.DefaultRegistry()); err != nil || socket == "" {
		t.Fatalf("default local registry socket = %q, err=%v", socket, err)
	}
	if _, err := resolveV3SocketForConnectionRegistry("", endpointdomain.Registry{}); err == nil {
		t.Fatal("empty registry must not start an unregistered local daemon")
	}
	remote := endpointdomain.NewSSHEndpoint("remote", "Remote", "remote.example", "", "127.0.0.1:41120", "127.0.0.1:41121", endpointdomain.ConnectOnDemand)
	registry := endpointdomain.Registry{Version: endpointdomain.RegistryVersion, Default: "remote", Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{"remote": remote}}
	if _, err := resolveV3SocketForConnectionRegistry("", registry); err == nil {
		t.Fatal("remote-only registry must not fall back to the default local socket")
	}
	mixed := endpointdomain.DefaultRegistry()
	mixed.Endpoints["remote"] = remote
	mixed.Default = "remote"
	if _, err := resolveV3SocketForConnectionRegistry("", mixed); err == nil {
		t.Fatal("mixed registry must honor its remote default instead of falling back to the local endpoint")
	}
}

func TestV3AttachDoesNotCreateTuiv2Config(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRunAttach := runV3Attach
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runV3Attach = oldRunAttach
	})

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	isInteractiveTerminal = func() bool { return true }
	runV3Attach = func(ctx context.Context, cfg v3AttachConfig) error {
		return nil
	}

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

func TestV3AttachLoadsTUIV3Config(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRunAttach := runV3Attach
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runV3Attach = oldRunAttach
	})

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	configDir := filepath.Join(configHome, "anytty")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "tui-v3.yaml"), []byte(`
version: 1
tui:
  theme:
    primary: "#d65cff"
`), 0o644); err != nil {
		t.Fatalf("write tui-v3 config: %v", err)
	}

	isInteractiveTerminal = func() bool { return true }
	var got v3AttachConfig
	runV3Attach = func(ctx context.Context, cfg v3AttachConfig) error {
		got = cfg
		return nil
	}

	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"--log-file", filepath.Join(t.TempDir(), "anytty.log"), "v3", "attach", "term-1"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("v3 attach returned error: %v", err)
	}
	if got.TUIConfig.Theme.Primary != "#d65cff" {
		t.Fatalf("expected v3 attach to load tui-v3 config, got %#v", got.TUIConfig)
	}
}

func TestV3RootLoadsTUIV3Config(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRunRoot := runV3Root
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runV3Root = oldRunRoot
	})

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	configDir := filepath.Join(configHome, "anytty")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "tui-v3.yaml"), []byte(`
version: 1
tui:
  theme:
    secondary: "#66e3ff"
`), 0o644); err != nil {
		t.Fatalf("write tui-v3 config: %v", err)
	}

	isInteractiveTerminal = func() bool { return true }
	var got v3RootConfig
	runV3Root = func(ctx context.Context, cfg v3RootConfig) error {
		got = cfg
		return nil
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--log-file", filepath.Join(t.TempDir(), "anytty.log")})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("root command returned error: %v", err)
	}
	if got.TUIConfig.Theme.Secondary != "#66e3ff" {
		t.Fatalf("expected root command to load tui-v3 config, got %#v", got.TUIConfig)
	}
}

func TestV3AttachRejectsInvalidTUIV3Config(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRunAttach := runV3Attach
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runV3Attach = oldRunAttach
	})

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	configDir := filepath.Join(configHome, "anytty")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "tui-v3.yaml"), []byte(`
version: 1
tui:
  theme:
    primary: red
`), 0o644); err != nil {
		t.Fatalf("write bad tui-v3 config: %v", err)
	}

	isInteractiveTerminal = func() bool { return true }
	runV3Attach = func(ctx context.Context, cfg v3AttachConfig) error {
		t.Fatalf("runV3Attach should not be called for invalid config")
		return nil
	}

	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"--log-file", filepath.Join(t.TempDir(), "anytty.log"), "v3", "attach", "term-1"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "must be empty or #RRGGBB") {
		t.Fatalf("expected invalid config error, got %v", err)
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

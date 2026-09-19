package cli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	localadapter "github.com/anytty/anytty/access/engine/adapter/local"
	clientprotocol "github.com/anytty/anytty/access/engine/adapter/protocol"
	endpointdomain "github.com/anytty/anytty/access/engine/endpoint"
	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	"github.com/anytty/anytty/access/files"
	poolprovider "github.com/anytty/anytty/access/provider/pool"
	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	accessruntime "github.com/anytty/anytty/access/runtime"
	accessserver "github.com/anytty/anytty/access/server"
	"github.com/anytty/anytty/internal/protocol"
	corev2 "github.com/anytty/anytty/pool/core"
	"github.com/anytty/anytty/pool/core/history"
	providercore "github.com/anytty/anytty/pool/provider"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport"
)

func TestRootCmdRoutesToTUIv3ByDefault(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRun := runTUI2
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runTUI2 = oldRun
	})

	isInteractiveTerminal = func() bool { return true }
	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	logPath := filepath.Join(t.TempDir(), "anytty.log")
	explicitConfig := filepath.Join(t.TempDir(), "tui2.json")
	if err := os.WriteFile(explicitConfig, []byte("{\"theme\":\"dark\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var gotCfg tui2EntryConfig
	calledRoot := false
	runTUI2 = func(_ context.Context, cfg tui2EntryConfig) error {
		calledRoot = true
		gotCfg = cfg
		return nil
	}

	cmd := newRootCmd()
	if !cmd.SilenceUsage {
		t.Fatal("runtime failures must not print the full command usage")
	}
	cmd.SetArgs([]string{"--socket", socketPath, "--log-file", logPath, "--config", explicitConfig})
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !calledRoot {
		t.Fatal("expected default root command to call the tui2 runner")
	}
	if gotCfg.SocketOverride != socketPath || gotCfg.LocalSocket != socketPath {
		t.Fatalf("unexpected socket routing %#v", gotCfg)
	}
	if gotCfg.LogFile != logPath || gotCfg.ConfigPath != explicitConfig || gotCfg.TerminalID != "" {
		t.Fatalf("unexpected tui2 entry config %#v", gotCfg)
	}
}
func TestRootCmdExposesBuildVersion(t *testing.T) {
	if got := newRootCmd().Version; got != version {
		t.Fatalf("root version = %q, want %q", got, version)
	}
}

func TestRootCmdLoadsConnectionRegistryLocalSocket(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRun := runTUI2
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runTUI2 = oldRun
	})

	isInteractiveTerminal = func() bool { return true }
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	socketPath := filepath.Join(t.TempDir(), "configured.sock")
	writeCLIConnectionRegistry(t, configHome, `
version: 3
default: local
endpoints:
  local:
    label: "Configured Local"
    enabled: true
    connect_mode: auto
    routes:
      local:
        kind: local-unix
        enabled: true
        socket: `+yamlTestString(socketPath)+`
`)

	var gotCfg tui2EntryConfig
	runTUI2 = func(_ context.Context, cfg tui2EntryConfig) error {
		gotCfg = cfg
		return nil
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--log-file", filepath.Join(t.TempDir(), "anytty.log")})
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if gotCfg.LocalSocket != socketPath || gotCfg.SocketOverride != "" {
		t.Fatalf("expected registry local socket %q, got %#v", socketPath, gotCfg)
	}
}
func TestRootCmdSocketFlagOverridesConnectionRegistrySocket(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRun := runTUI2
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runTUI2 = oldRun
	})

	isInteractiveTerminal = func() bool { return true }
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	registrySocket := filepath.Join(t.TempDir(), "registry.sock")
	flagSocket := filepath.Join(t.TempDir(), "flag.sock")
	writeCLIConnectionRegistry(t, configHome, `
version: 3
default: local
endpoints:
  local:
    label: "Configured Local"
    enabled: true
    connect_mode: auto
    routes:
      local:
        kind: local-unix
        enabled: true
        socket: `+yamlTestString(registrySocket)+`
`)

	var gotCfg tui2EntryConfig
	runTUI2 = func(_ context.Context, cfg tui2EntryConfig) error {
		gotCfg = cfg
		return nil
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--socket", flagSocket, "--log-file", filepath.Join(t.TempDir(), "anytty.log")})
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if gotCfg.SocketOverride != flagSocket || gotCfg.LocalSocket != flagSocket {
		t.Fatalf("expected --socket to override registry socket %q, got %#v", flagSocket, gotCfg)
	}
}
func writeCLIConnectionRegistry(t *testing.T, configHome string, content string) {
	t.Helper()
	dir := filepath.Join(configHome, "anytty")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir connection registry dir: %v", err)
	}
	path := filepath.Join(dir, endpointdomain.DefaultFileName)
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("write connection registry: %v", err)
	}
}

func yamlTestString(value string) string {
	return strconv.Quote(value)
}

func TestRootCmdBlocksNestedTUIByDefault(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRun := runTUI2
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runTUI2 = oldRun
	})

	isInteractiveTerminal = func() bool { return true }
	runTUI2 = func(context.Context, tui2EntryConfig) error {
		t.Fatal("runTUI2 should not be called when nested TUI is blocked")
		return nil
	}

	t.Setenv("ANYTTY", "1")
	t.Setenv("ANYTTY_ALLOW_NESTED", "")

	cmd := newRootCmd()
	cmd.SetArgs([]string{})
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "refusing to start anytty TUI inside a anytty remote terminal") {
		t.Fatalf("expected nested TUI rejection, got %v", err)
	}
}
func TestAttachCmdRoutesToTUIv3ByDefault(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRun := runTUI2
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runTUI2 = oldRun
	})

	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	logPath := filepath.Join(t.TempDir(), "anytty.log")
	isInteractiveTerminal = func() bool { return true }
	var got tui2EntryConfig
	runTUI2 = func(_ context.Context, cfg tui2EntryConfig) error {
		got = cfg
		return nil
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--socket", socketPath, "--log-file", logPath, "attach", "term-001"})
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected attach command to succeed, got %v", err)
	}
	if got.TerminalID != "term-001" || got.EndpointID != string(endpointdomain.DefaultEndpointID) ||
		got.SocketOverride != socketPath || got.LogFile != logPath {
		t.Fatalf("unexpected tui2 attach config %#v", got)
	}
}
func TestAttachCmdAllowsNestedTUIWhenOverrideIsSet(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRun := runTUI2
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runTUI2 = oldRun
	})

	isInteractiveTerminal = func() bool { return true }
	called := false
	runTUI2 = func(_ context.Context, cfg tui2EntryConfig) error {
		called = true
		return nil
	}

	t.Setenv("ANYTTY", "1")
	t.Setenv("ANYTTY_ALLOW_NESTED", "1")

	cmd := newRootCmd()
	cmd.SetArgs([]string{"attach", "term-001"})
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected attach override to succeed, got %v", err)
	}
	if !called {
		t.Fatal("expected attach command to reach the tui2 runner when override is set")
	}
}
func TestAttachCmdBlocksNestedTUIByDefault(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRun := runTUI2
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runTUI2 = oldRun
	})

	isInteractiveTerminal = func() bool { return true }
	runTUI2 = func(context.Context, tui2EntryConfig) error {
		t.Fatal("runTUI2 should not be called when nested attach is blocked")
		return nil
	}

	t.Setenv("ANYTTY", "1")
	t.Setenv("ANYTTY_ALLOW_NESTED", "")

	cmd := newRootCmd()
	cmd.SetArgs([]string{"attach", "term-001"})
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "refusing to start anytty TUI inside a anytty remote terminal") {
		t.Fatalf("expected nested attach rejection, got %v", err)
	}
}
func TestPoolAppliesHistoryStorageConfigFile(t *testing.T) {
	// `pool:` 是当前段；`daemon:` 是升级兼容段，两者都必须被解析。
	for _, section := range []string{"pool", "daemon"} {
		t.Run(section, func(t *testing.T) {
			t.Setenv("XDG_STATE_HOME", t.TempDir())
			oldNewCoreV2Server := newCoreV2Server
			t.Cleanup(func() { newCoreV2Server = oldNewCoreV2Server })
			configPath := filepath.Join(t.TempDir(), "anytty.yaml")
			config := "version: 1\n" + section + ":\n  history:\n    max_size_mb: 64\n    max_age_days: 14\n    compression: s2\n    compression_level: best\n  resource_sampling:\n    interval_ms: 750\n    max_samples: 1024\n"
			if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
				t.Fatal(err)
			}
			fakeV3 := &fakeCoreV2Server{}
			newCoreV2Server = func(opts ...corev2.ServerOption) coreV2Server {
				server := newCoreV2TestServer(opts...)
				if got := server.HistoryStorageConfig(); got.MaxBytesPerTerminal != 64<<20 || got.MaxAge != 14*24*time.Hour || got.Compression != corev2.HistoryCompressionS2 || got.CompressionLevel != corev2.HistoryCompressionLevelBest {
					t.Fatalf("pool did not apply %s history storage config: %#v", section, got)
				}
				if got := server.TerminalResourceSamplingConfig(); got.Interval != 750*time.Millisecond || got.MaxSamples != 1024 {
					t.Fatalf("pool did not apply %s resource sampling config: %#v", section, got)
				}
				return fakeV3
			}
			cmd := newRootCmd()
			cmd.SetArgs([]string{"--config", configPath, "--socket", filepath.Join(t.TempDir(), "anytty-v2.sock"), "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "pool"})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
		})
	}
}

func TestPoolConfigSectionOverridesLegacyDaemonSection(t *testing.T) {
	t.Setenv("ANYTTY_HISTORY_MAX_SIZE_MB", "")
	configPath := filepath.Join(t.TempDir(), "anytty.yaml")
	config := "version: 1\ndaemon:\n  history:\n    max_size_mb: 64\npool:\n  history:\n    max_size_mb: 32\n"
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadPoolRuntimeConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.History.MaxSizeMB != 32 {
		t.Fatalf("pool section must override legacy daemon section, got %d", loaded.History.MaxSizeMB)
	}
}

func TestPoolCanDisableHistoryFromEnv(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	oldNewCoreV2Server := newCoreV2Server
	t.Cleanup(func() {
		newCoreV2Server = oldNewCoreV2Server
	})
	t.Setenv("ANYTTY_HISTORY_DISABLE", "1")

	fakeV3 := &fakeCoreV2Server{}
	newCoreV2Server = func(opts ...corev2.ServerOption) coreV2Server {
		fakeV3.newServerCalls++
		opts = append(opts, corev2.WithProcessFactory(newCoreV2ResizeRecordingProcessFactory()))
		server := newCoreV2TestServer(opts...)
		if server.HistoryStorageDir() != "" {
			t.Fatalf("history disabled pool must not configure history storage dir, got %q", server.HistoryStorageDir())
		}
		if _, err := server.RegisterTerminal(corev2.TerminalRecord{ID: "term-disabled", Command: []string{"shell"}}); err != nil {
			t.Fatalf("register disabled-history terminal: %v", err)
		}
		if _, err := server.TerminalHistoryWindow(context.Background(), "term-disabled", history.HistoryWindowRequest{
			TerminalID: "term-disabled",
			Mode:       history.HistoryWindowModeLatest,
			Limit:      1,
			Cols:       20,
		}); !errors.Is(err, corev2.ErrHistoryDisabled) {
			t.Fatalf("expected disabled history window, got %v", err)
		}
		return fakeV3
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--socket", filepath.Join(t.TempDir(), "anytty-v2.sock"), "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "pool"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if fakeV3.newServerCalls != 1 {
		t.Fatalf("expected one core-v2 server construction, got %d", fakeV3.newServerCalls)
	}
}

func TestPoolConfiguresTerminalOutputBufferFromEnv(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	oldNewCoreV2Server := newCoreV2Server
	t.Cleanup(func() {
		newCoreV2Server = oldNewCoreV2Server
	})
	t.Setenv("ANYTTY_OUTPUT_BUFFER_OVERFLOW", "block")
	t.Setenv("ANYTTY_OUTPUT_BUFFER_CAPACITY_BYTES", "12582912")
	t.Setenv("ANYTTY_OUTPUT_RESIDENT_BUDGET_BYTES", "268435456")
	t.Setenv("ANYTTY_RESOURCE_SAMPLING_INTERVAL_MS", "750")
	t.Setenv("ANYTTY_RESOURCE_SAMPLING_MAX_SAMPLES", "1024")

	fakeV3 := &fakeCoreV2Server{}
	newCoreV2Server = func(opts ...corev2.ServerOption) coreV2Server {
		fakeV3.newServerCalls++
		server := newCoreV2TestServer(opts...)
		got := server.TerminalOutputBufferConfig()
		if got.Overflow != corev2.TerminalOutputOverflowBlock || got.CapacityBytes != 12<<20 {
			t.Fatalf("pool did not pass output buffer env to core: %#v", got)
		}
		if budget := server.TerminalOutputResidentBudget(); budget != 256<<20 {
			t.Fatalf("pool did not pass resident budget env to core: %d", budget)
		}
		if resources := server.TerminalResourceSamplingConfig(); resources.Interval != 750*time.Millisecond || resources.MaxSamples != 1024 {
			t.Fatalf("pool did not pass resource sampling env to core: %#v", resources)
		}
		return fakeV3
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--socket", filepath.Join(t.TempDir(), "anytty-v2.sock"), "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "pool"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if fakeV3.newServerCalls != 1 {
		t.Fatalf("expected one core-v2 server construction, got %d", fakeV3.newServerCalls)
	}
}

func TestV3PingConnectsExistingCoreV2Pool(t *testing.T) {
	oldConnect := connectV3EndpointApplication
	oldStart := startV3Pool
	t.Cleanup(func() {
		connectV3EndpointApplication = oldConnect
		startV3Pool = oldStart
	})

	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	dialed := false
	connectV3EndpointApplication = func(_ context.Context, _ *clientruntime.SessionOwner, _ endpointdomain.Endpoint, _ endpointdomain.RouteID, _ clientruntime.ConnectIntent, options localadapter.Options, _ *slog.Logger) (*clientprotocol.ApplicationClient, endpointdomain.AccessRoute, error) {
		if options.SocketOverride != socketPath {
			t.Fatalf("expected v3 ping to dial socket %q, got %q", socketPath, options.SocketOverride)
		}
		dialed = true
		return nil, endpointdomain.AccessRoute{}, nil
	}
	startV3Pool = func(path string, logFile string) error {
		t.Fatal("v3 ping must not auto-start when existing pool is reachable")
		return nil
	}

	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"--socket", socketPath, "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "v3", "ping"})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !dialed {
		t.Fatal("expected v3 ping to dial core-v2 pool")
	}
	if !strings.Contains(out.String(), "anytty v3 pool ok") || !strings.Contains(out.String(), socketPath) {
		t.Fatalf("unexpected v3 ping output:\n%s", out.String())
	}
}

func TestV3PingAutoStartsCoreV2Pool(t *testing.T) {
	oldConnect := connectV3EndpointApplication
	oldStart := startV3Pool
	oldAccessStart := startV3Access
	t.Cleanup(func() {
		connectV3EndpointApplication = oldConnect
		startV3Pool = oldStart
		startV3Access = oldAccessStart
	})

	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	logPath := filepath.Join(t.TempDir(), "anytty.log")
	connectCalls := 0
	startCalls := 0
	accessStartCalls := 0
	var startedSocket string
	var startedLog string
	connectV3EndpointApplication = func(ctx context.Context, _ *clientruntime.SessionOwner, _ endpointdomain.Endpoint, _ endpointdomain.RouteID, _ clientruntime.ConnectIntent, options localadapter.Options, _ *slog.Logger) (*clientprotocol.ApplicationClient, endpointdomain.AccessRoute, error) {
		connectCalls++
		if options.SocketOverride != socketPath {
			t.Fatalf("expected dial socket %q, got %q", socketPath, options.SocketOverride)
		}
		if err := options.Start(ctx, socketPath); err != nil {
			return nil, endpointdomain.AccessRoute{}, err
		}
		return nil, endpointdomain.AccessRoute{}, nil
	}
	startV3Pool = func(path string, logFile string) error {
		startCalls++
		startedSocket = path
		startedLog = logFile
		return nil
	}
	startV3Access = func(path string, logFile string) error {
		accessStartCalls++
		return nil
	}

	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"--socket", socketPath, "--log-file", logPath, "v3", "ping"})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if connectCalls != 1 {
		t.Fatalf("expected one owner-managed connect attempt, got %d", connectCalls)
	}
	if startCalls != 1 || startedSocket != socketPath || startedLog != logPath {
		t.Fatalf("unexpected v3 pool auto-start: calls=%d socket=%q log=%q", startCalls, startedSocket, startedLog)
	}
	if accessStartCalls != 1 {
		t.Fatalf("expected local stack auto-start to start access once, got %d", accessStartCalls)
	}
	if !strings.Contains(out.String(), "anytty v3 pool ok") {
		t.Fatalf("unexpected v3 ping output:\n%s", out.String())
	}
}

func TestV3PingReturnsAutoStartError(t *testing.T) {
	oldConnect := connectV3EndpointApplication
	oldStart := startV3Pool
	t.Cleanup(func() {
		connectV3EndpointApplication = oldConnect
		startV3Pool = oldStart
	})

	connectV3EndpointApplication = func(ctx context.Context, _ *clientruntime.SessionOwner, _ endpointdomain.Endpoint, _ endpointdomain.RouteID, _ clientruntime.ConnectIntent, options localadapter.Options, _ *slog.Logger) (*clientprotocol.ApplicationClient, endpointdomain.AccessRoute, error) {
		return nil, endpointdomain.AccessRoute{}, options.Start(ctx, options.SocketOverride)
	}
	startV3Pool = func(path string, logFile string) error {
		return os.ErrPermission
	}

	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"--socket", filepath.Join(t.TempDir(), "anytty-v2.sock"), "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "v3", "ping"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "start local access stack") || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected auto-start error, got %v", err)
	}
}

func TestV3PingConnectsRealCoreV2Pool(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	server := newCoreV2TestServer(corev2.WithSocketPath(socketPath + ".provider"))
	stopProvider := startCoreV2ProviderServer(t, server, socketPath+".provider")
	defer func() {
		stopProvider()
		_ = server.Shutdown(context.Background())
	}()
	startCLIAccessServer(t, socketPath)

	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"--socket", socketPath, "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "v3", "ping"})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(out.String(), "anytty v3 pool ok") || !strings.Contains(out.String(), socketPath) {
		t.Fatalf("unexpected v3 ping output:\n%s", out.String())
	}
}

func TestStartCoreV2PoolCommandUsesV3Pool(t *testing.T) {
	oldExecutable := osExecutable
	t.Cleanup(func() {
		osExecutable = oldExecutable
	})

	exe := filepath.Join(t.TempDir(), "anytty")
	osExecutable = func() (string, error) {
		return exe, nil
	}
	got, err := buildStartCoreV2PoolCommand("/tmp/anytty-v2.sock", "/tmp/anytty.log")
	if err != nil {
		t.Fatalf("buildStartCoreV2PoolCommand returned error: %v", err)
	}
	if got.Path != exe {
		t.Fatalf("expected executable %q, got %q", exe, got.Path)
	}
	wantArgs := []string{exe, "--socket", "/tmp/anytty-v2.sock", "--log-file", "/tmp/anytty.log", "pool"}
	if !reflect.DeepEqual(got.Args, wantArgs) {
		t.Fatalf("unexpected v3 pool args: %#v", got.Args)
	}
}

func TestStartCoreV2PoolCommandCarriesHistoryDisableEnv(t *testing.T) {
	oldExecutable := osExecutable
	t.Cleanup(func() {
		osExecutable = oldExecutable
	})
	t.Setenv("ANYTTY_HISTORY_DISABLE", "1")

	exe := filepath.Join(t.TempDir(), "anytty")
	osExecutable = func() (string, error) {
		return exe, nil
	}
	got, err := buildStartCoreV2PoolCommand("/tmp/anytty-v2.sock", "/tmp/anytty.log")
	if err != nil {
		t.Fatalf("buildStartCoreV2PoolCommand returned error: %v", err)
	}
	if got.Path != exe {
		t.Fatalf("expected executable %q, got %q", exe, got.Path)
	}
	if !containsEnv(got.Env, "ANYTTY_HISTORY_DISABLE=1") {
		t.Fatalf("auto-start pool command must carry history disabled env, env=%#v", got.Env)
	}
}

func containsEnv(env []string, want string) bool {
	for _, item := range env {
		if item == want {
			return true
		}
	}
	return false
}

func TestStartCoreV2PoolCommandCanCarryExplicitConfigPath(t *testing.T) {
	oldExecutable := osExecutable
	t.Cleanup(func() {
		osExecutable = oldExecutable
	})

	exe := filepath.Join(t.TempDir(), "anytty")
	configPath := filepath.Join(t.TempDir(), "anytty.yaml")
	osExecutable = func() (string, error) {
		return exe, nil
	}
	got, err := buildStartCoreV2PoolCommandWithConfig("/tmp/anytty-v2.sock", "/tmp/anytty.log", configPath)
	if err != nil {
		t.Fatalf("buildStartCoreV2PoolCommandWithConfig returned error: %v", err)
	}
	wantArgs := []string{exe, "--socket", "/tmp/anytty-v2.sock", "--log-file", "/tmp/anytty.log", "--config", configPath, "pool"}
	if !reflect.DeepEqual(got.Args, wantArgs) {
		t.Fatalf("unexpected v3 pool args with config: %#v", got.Args)
	}
}

func TestDialOrStartV3ClientUsesConfigStarterWhenConfigPathIsExplicit(t *testing.T) {
	oldDial := v3DialClient
	oldStart := startV3Pool
	oldStartWithConfig := startV3PoolWithConfig
	oldAccessStart := startV3Access
	oldConnect := connectV3EndpointApplication
	t.Cleanup(func() {
		v3DialClient = oldDial
		startV3Pool = oldStart
		startV3PoolWithConfig = oldStartWithConfig
		startV3Access = oldAccessStart
		connectV3EndpointApplication = oldConnect
	})
	configPath := filepath.Join(t.TempDir(), "anytty.yaml")
	socketPath := filepath.Join(t.TempDir(), "anytty.sock")
	logPath := filepath.Join(t.TempDir(), "anytty.log")
	startV3Pool = func(path string, logFile string) error {
		t.Fatal("plain pool starter must not be used when explicit config path is present")
		return nil
	}
	var gotSocket, gotLog, gotConfig string
	startV3PoolWithConfig = func(path string, logFile string, cfg string) error {
		gotSocket, gotLog, gotConfig = path, logFile, cfg
		return nil
	}
	startV3Access = func(path string, logFile string) error { return nil }
	connectV3EndpointApplication = func(ctx context.Context, _ *clientruntime.SessionOwner, _ endpointdomain.Endpoint, _ endpointdomain.RouteID, _ clientruntime.ConnectIntent, options localadapter.Options, _ *slog.Logger) (*clientprotocol.ApplicationClient, endpointdomain.AccessRoute, error) {
		if err := options.Start(ctx, socketPath); err != nil {
			return nil, endpointdomain.AccessRoute{}, err
		}
		return nil, endpointdomain.AccessRoute{}, nil
	}

	client, err := dialOrStartV3ClientWithConfig(socketPath, logPath, configPath, nil)
	if err != nil {
		t.Fatalf("dialOrStartV3ClientWithConfig returned error: %v", err)
	}
	if client != nil {
		_ = client.Close()
	}
	if gotSocket != socketPath || gotLog != logPath || gotConfig != configPath {
		t.Fatalf("config starter got socket=%q log=%q config=%q", gotSocket, gotLog, gotConfig)
	}
}

func TestDefaultLocalControlCommandsUseCoreV2Protocol(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	server := newCoreV2TestServer(corev2.WithSocketPath(socketPath + ".provider"))
	stopProvider := startCoreV2ProviderServer(t, server, socketPath+".provider")
	defer func() {
		stopProvider()
		_ = server.Shutdown(context.Background())
	}()
	startCLIAccessServer(t, socketPath)

	logPath := filepath.Join(t.TempDir(), "anytty.log")
	var newOut bytes.Buffer
	newCmd := newRootCmd()
	newCmd.SetArgs(append([]string{"--socket", socketPath, "--log-file", logPath, "new", "--name", "v3-demo", "--"}, testShellSleepCommand()...))
	newCmd.SetOut(&newOut)
	newCmd.SetErr(io.Discard)
	if err := newCmd.Execute(); err != nil {
		t.Fatalf("new returned error: %v", err)
	}
	terminalTarget := strings.TrimSpace(newOut.String())
	if terminalTarget != "local:v3-demo" {
		t.Fatalf("expected terminal create to print stable target, got %q", newOut.String())
	}
	terminalID := strings.TrimPrefix(terminalTarget, "local:")

	var lsOut bytes.Buffer
	lsCmd := newRootCmd()
	lsCmd.SetArgs([]string{"--socket", socketPath, "--log-file", logPath, "ls"})
	lsCmd.SetOut(&lsOut)
	lsCmd.SetErr(io.Discard)
	if err := lsCmd.Execute(); err != nil {
		t.Fatalf("ls returned error: %v", err)
	}
	lsText := lsOut.String()
	if !strings.Contains(lsText, terminalID) ||
		!strings.Contains(lsText, "v3-demo") ||
		!strings.Contains(lsText, testShellSleepCommand()[0]) ||
		!strings.Contains(lsText, "running") {
		t.Fatalf("unexpected v3 ls output:\n%s", lsText)
	}

	var listJSON bytes.Buffer
	listJSONCmd := newRootCmd()
	listJSONCmd.SetArgs([]string{"--socket", socketPath, "--log-file", logPath, "terminal", "list", "--json"})
	listJSONCmd.SetOut(&listJSON)
	listJSONCmd.SetErr(io.Discard)
	if err := listJSONCmd.Execute(); err != nil {
		t.Fatalf("terminal list JSON returned error: %v", err)
	}
	for _, expected := range []string{`"schema_version":1`, `"kind":"terminal_list"`, `"target":"local:v3-demo"`} {
		if !strings.Contains(listJSON.String(), expected) {
			t.Fatalf("terminal list JSON missing %s: %s", expected, listJSON.String())
		}
	}

	runningRemoveCmd := newRootCmd()
	runningRemoveCmd.SetArgs([]string{"--socket", socketPath, "--log-file", logPath, "terminal", "remove", terminalTarget})
	runningRemoveCmd.SetOut(io.Discard)
	runningRemoveCmd.SetErr(io.Discard)
	if err := runningRemoveCmd.Execute(); cliExitCode(err) != 4 {
		t.Fatalf("running terminal remove error = %v, exit=%d", err, cliExitCode(err))
	}

	renameCmd := newRootCmd()
	renameCmd.SetArgs([]string{"--socket", socketPath, "--log-file", logPath, "terminal", "rename", terminalTarget, "renamed-demo"})
	renameCmd.SetOut(io.Discard)
	renameCmd.SetErr(io.Discard)
	if err := renameCmd.Execute(); err != nil {
		t.Fatalf("terminal rename returned error: %v", err)
	}

	tagCmd := newRootCmd()
	tagCmd.SetArgs([]string{"--socket", socketPath, "--log-file", logPath, "terminal", "tag", terminalTarget, "role=test"})
	tagCmd.SetOut(io.Discard)
	tagCmd.SetErr(io.Discard)
	if err := tagCmd.Execute(); err != nil {
		t.Fatalf("terminal tag returned error: %v", err)
	}

	var showOut bytes.Buffer
	showCmd := newRootCmd()
	showCmd.SetArgs([]string{"--socket", socketPath, "--log-file", logPath, "terminal", "show", terminalTarget, "--json"})
	showCmd.SetOut(&showOut)
	showCmd.SetErr(io.Discard)
	if err := showCmd.Execute(); err != nil {
		t.Fatalf("terminal show returned error: %v", err)
	}
	for _, expected := range []string{`"schema_version":1`, `"kind":"terminal"`, `"target":"local:v3-demo"`, `"name":"renamed-demo"`, `"role":"test"`} {
		if !strings.Contains(showOut.String(), expected) {
			t.Fatalf("terminal show JSON missing %s: %s", expected, showOut.String())
		}
	}

	restartCmd := newRootCmd()
	restartCmd.SetArgs([]string{"--socket", socketPath, "--log-file", logPath, "terminal", "restart", terminalTarget, "--quiet"})
	restartCmd.SetOut(io.Discard)
	restartCmd.SetErr(io.Discard)
	if err := restartCmd.Execute(); err != nil {
		t.Fatalf("terminal restart returned error: %v", err)
	}

	killCmd := newRootCmd()
	killCmd.SetArgs([]string{"--socket", socketPath, "--log-file", logPath, "kill", terminalTarget})
	killCmd.SetOut(io.Discard)
	killCmd.SetErr(io.Discard)
	if err := killCmd.Execute(); err != nil {
		t.Fatalf("kill returned error: %v", err)
	}
	// kill 是异步进程终止；rm 前必须等 exited，否则 race/高负载下会误报 running。
	waitForCLITerminalState(t, server, terminalID, corev2.TerminalStateExited)

	rmCmd := newRootCmd()
	rmCmd.SetArgs([]string{"--socket", socketPath, "--log-file", logPath, "rm", terminalID})
	rmCmd.SetOut(io.Discard)
	rmCmd.SetErr(io.Discard)
	if err := rmCmd.Execute(); err != nil {
		t.Fatalf("rm returned error: %v", err)
	}
	if _, err := server.GetTerminal(terminalID); err == nil || !strings.Contains(err.Error(), "terminal not found") {
		t.Fatalf("expected removed terminal lookup to fail, got %v", err)
	}

	missingShowCmd := newRootCmd()
	missingShowCmd.SetArgs([]string{"--socket", socketPath, "--log-file", logPath, "terminal", "show", terminalTarget, "--json"})
	missingShowCmd.SetOut(io.Discard)
	missingShowCmd.SetErr(io.Discard)
	if err := missingShowCmd.Execute(); cliExitCode(err) != 3 {
		t.Fatalf("removed terminal show error = %v, exit=%d", err, cliExitCode(err))
	}

	var emptyLs bytes.Buffer
	emptyCmd := newRootCmd()
	emptyCmd.SetArgs([]string{"--socket", socketPath, "--log-file", logPath, "ls"})
	emptyCmd.SetOut(&emptyLs)
	emptyCmd.SetErr(io.Discard)
	if err := emptyCmd.Execute(); err != nil {
		t.Fatalf("ls after remove returned error: %v", err)
	}
	if strings.Contains(emptyLs.String(), terminalID) {
		t.Fatalf("removed terminal still listed:\n%s", emptyLs.String())
	}
}

func TestV3LocalControlCommandsRemainAvailable(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	server := newCoreV2TestServer(corev2.WithSocketPath(socketPath + ".provider"))
	stopProvider := startCoreV2ProviderServer(t, server, socketPath+".provider")
	defer func() {
		stopProvider()
		_ = server.Shutdown(context.Background())
	}()
	startCLIAccessServer(t, socketPath)

	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs(append([]string{"--socket", socketPath, "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "v3", "new", "--name", "v3-demo", "--"}, testShellSleepCommand()...))
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("v3 new returned error after default switch: %v", err)
	}
	if strings.TrimSpace(out.String()) != "v3-demo" {
		t.Fatalf("expected v3 new to print terminal id, got %q", out.String())
	}
}

func TestR448V3HistoryBacklogWritesDiagnostics(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	server := newCoreV2TestServer(
		corev2.WithSocketPath(socketPath+".provider"),
		corev2.WithProcessFactory(newCoreV2ResizeRecordingProcessFactory()),
		corev2.WithTerminalOutputBufferConfig(corev2.TerminalOutputBufferConfig{
			Overflow:      corev2.TerminalOutputOverflowBlock,
			CapacityBytes: 64 << 10,
		}),
	)
	stopProvider := startCoreV2ProviderServer(t, server, socketPath+".provider")
	defer func() {
		stopProvider()
		_ = server.Shutdown(context.Background())
	}()
	startCLIAccessServer(t, socketPath)
	client, err := dialV3Client(socketPath)
	if err != nil {
		t.Fatalf("dial core-v2 pool: %v", err)
	}
	if _, err := createCLIProtoTerminal(context.Background(), client, &apipb.TerminalCreateSpec{TerminalId: "term-backlog", Command: []string{"shell"}, Size: &apipb.TerminalSize{Cols: 24, Rows: 4}}); err != nil {
		t.Fatalf("create terminal: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-backlog", "alpha\r\nbeta\r\n"); err != nil {
		t.Fatalf("ingest output: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("close setup client: %v", err)
	}

	var out bytes.Buffer
	outPath := filepath.Join(t.TempDir(), "history-backlog.tsv")
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"--socket", socketPath, "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "v3", "history-backlog", "term-backlog", "--out", outPath})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("history-backlog returned error: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read history backlog output: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"terminal_id\thistory_enabled",
		"term-backlog\ttrue",
		"\tblock\t65536\t",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("history backlog output missing %q:\n%s", want, text)
		}
	}
	if !strings.Contains(out.String(), "anytty v3 history backlog ok") || !strings.Contains(out.String(), outPath) {
		t.Fatalf("unexpected command output:\n%s", out.String())
	}

	dumpPath := filepath.Join(t.TempDir(), "history-dump.txt")
	dumpCmd := newDevelopmentRootCmd()
	dumpCmd.SetArgs([]string{"--socket", socketPath, "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "v3", "history-dump", "term-backlog", "--out", dumpPath, "--cols", "24", "--limit", "1"})
	var dumpOut bytes.Buffer
	dumpCmd.SetOut(&dumpOut)
	dumpCmd.SetErr(io.Discard)
	if err := dumpCmd.Execute(); err != nil {
		t.Fatalf("history-dump returned error: %v", err)
	}
	dump, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dump), " page_row=") || !strings.Contains(string(dump), "alpha") || !strings.Contains(string(dump), "beta") {
		t.Fatalf("history dump did not paginate authoritative rows:\n%s", dump)
	}
	if !strings.Contains(dumpOut.String(), "anytty v3 history dump ok") {
		t.Fatalf("unexpected history dump command output: %s", dumpOut.String())
	}
}

func TestV3AttachRejectsNonInteractiveTerminal(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRun := runTUI2
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runTUI2 = oldRun
	})

	isInteractiveTerminal = func() bool { return false }
	runTUI2 = func(context.Context, tui2EntryConfig) error {
		t.Fatal("runTUI2 should not be called for a non-interactive terminal")
		return nil
	}

	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "attach", "term-001"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "requires an interactive terminal") {
		t.Fatalf("expected non-interactive attach rejection, got %v", err)
	}
}
func TestRemoteCommandsAreNotMounted(t *testing.T) {
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "remote", "status"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected v3 remote to remain unavailable, got %v", err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"remote", "status"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	err = root.Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected root remote to remain unavailable, got %v", err)
	}
}

func buildAnyTTYBinaryForTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "anytty-test-bin")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	build := func(output string, pkg string, tags ...string) {
		args := []string{"build"}
		if len(tags) > 0 {
			args = append(args, "-tags", strings.Join(tags, ","))
		}
		args = append(args, "-o", output, pkg)
		cmd := exec.Command("go", args...)
		cmd.Dir = filepath.Join("..", "..")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", pkg, err, out)
		}
	}
	build(path, "./cmd/anytty", "anytty_dev_commands")
	tui2Path := filepath.Join(dir, "tui2")
	shellPath := filepath.Join(dir, "tui2-shell")
	if runtime.GOOS == "windows" {
		tui2Path += ".exe"
		shellPath += ".exe"
	}
	build(tui2Path, "./clients/tui/cmd/tui2")
	build(shellPath, "./clients/tui/cmd/tui2-shell")
	// CLI 默认入口查找同目录的 tui2/tui2-shell；测试显式指定，避免依赖构建目录布局。
	t.Setenv("TUI2_BIN", tui2Path)
	t.Setenv("TUI2_SHELL", shellPath)
	return path
}

type fakeCoreV2Server struct {
	newServerCalls int
	listenCalls    int
	shutdownCalls  int
}

func (s *fakeCoreV2Server) Start(context.Context) error {
	s.listenCalls++
	return nil
}

func (s *fakeCoreV2Server) Shutdown(context.Context) error {
	s.shutdownCalls++
	return nil
}

func (s *fakeCoreV2Server) ServeTransport(context.Context, transport.Transport) error {
	return nil
}

func newCoreV2ProtocolClientForCLITest(t *testing.T) (*corev2.Server, *protocol.Client, func()) {
	return newCoreV2ProtocolClientForCLITestWithOptions(t)
}

func installV3LocalApplicationTestClient(t *testing.T, socketPath string, client *protocol.Client) {
	t.Helper()
	connectV3EndpointApplication = func(_ context.Context, owner *clientruntime.SessionOwner, target endpointdomain.Endpoint, requested endpointdomain.RouteID, intent clientruntime.ConnectIntent, options localadapter.Options, _ *slog.Logger) (*clientprotocol.ApplicationClient, endpointdomain.AccessRoute, error) {
		if options.SocketOverride != socketPath {
			return nil, endpointdomain.AccessRoute{}, fmt.Errorf("test local socket = %q, want %q", options.SocketOverride, socketPath)
		}
		if requested == "" {
			requested = endpointdomain.DefaultLocalRouteID
		}
		route, ok := target.Route(requested)
		if !ok {
			return nil, endpointdomain.AccessRoute{}, fmt.Errorf("test route %q is unavailable", requested)
		}
		attempt, err := owner.BeginRouteAttempt(target, route.ID, intent)
		if err != nil {
			return nil, endpointdomain.AccessRoute{}, err
		}
		ready, err := clientprotocol.NewApplicationClient(client, attempt.Stamp())
		if err != nil {
			return nil, endpointdomain.AccessRoute{}, err
		}
		if err := ready.MarkReady(clientruntime.ReadyPeerSessionEvidence{Identity: endpointdomain.DaemonIdentity{DeviceID: "device-cli-fixture", DeviceFingerprint: "SHA256:device-cli-fixture"}, IdentityVerified: true, AuthorizationVerified: true, ProtocolVersion: wire.Version}); err != nil {
			return nil, endpointdomain.AccessRoute{}, err
		}
		lease, err := owner.AdoptReadyPeerSession(attempt, ready)
		if err != nil {
			return nil, endpointdomain.AccessRoute{}, err
		}
		owned, err := owner.ApplicationSession(lease)
		if err != nil {
			return nil, endpointdomain.AccessRoute{}, err
		}
		application, err := clientprotocol.NewReadyApplicationClient(owned)
		if err != nil {
			return nil, endpointdomain.AccessRoute{}, err
		}
		return application, route, nil
	}
}

func wrapCLIProtocolClientForTest(t *testing.T, client *protocol.Client) *clientprotocol.ApplicationClient {
	t.Helper()
	application, err := wrapCLIProtocolClientForTestContext(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	return application
}

func wrapCLIProtocolClientForTestContext(ctx context.Context, client *protocol.Client) (*clientprotocol.ApplicationClient, error) {
	owner := clientruntime.NewSessionOwner()
	target, _ := endpointdomain.DefaultRegistry().DefaultEndpoint()
	attempt, err := owner.BeginRouteAttempt(target, endpointdomain.DefaultLocalRouteID, clientruntime.ConnectIntentInteractive)
	if err != nil {
		return nil, err
	}
	ready, err := clientprotocol.NewApplicationClient(client, attempt.Stamp())
	if err != nil {
		return nil, err
	}
	proofCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	identity, err := clientprotocol.VerifyDaemonIdentity(proofCtx, ready.ApplicationSession, endpointdomain.DaemonIdentity{})
	cancel()
	if err != nil {
		return nil, err
	}
	if err := ready.MarkReady(clientruntime.ReadyPeerSessionEvidence{Identity: identity, IdentityVerified: true, AuthorizationVerified: true, ProtocolVersion: wire.Version}); err != nil {
		return nil, err
	}
	lease, err := owner.AdoptReadyPeerSession(attempt, ready)
	if err != nil {
		return nil, err
	}
	owned, err := owner.ApplicationSession(lease)
	if err != nil {
		return nil, err
	}
	application, err := clientprotocol.NewReadyApplicationClient(owned)
	if err != nil {
		return nil, err
	}
	return application, nil
}

func newCoreV2TestServer(opts ...corev2.ServerOption) *corev2.Server {
	return corev2.NewServer(opts...)
}

// startCLIAccessServer 在 canonical socket 上启动 access（Auth 直答 + provider 路由）。
// 调用方必须已经启动 pool provider（<socketPath>.provider）。
func startCLIAccessServer(t *testing.T, socketPath string) *accessserver.Server {
	t.Helper()
	access, err := accessserver.New(accessserver.Config{
		Socket: socketPath,
		Files:  files.Config{TransferDir: filepath.Join(t.TempDir(), "transfers")},
		Auth:   &accessserver.AuthServices{Access: accessruntime.Service{DeviceIdentity: testCoreV2Identity()}},
		Provider: func(dialCtx context.Context) (terminalprovider.Provider, error) {
			return poolprovider.DialTerminal(dialCtx, socketPath+".provider")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		_ = access.Close()
		cancel()
	})
	go func() { _ = access.Serve(ctx) }()
	if err := waitForSocket(socketPath, 2*time.Second, func() error {
		return probeV3Socket(socketPath)
	}); err != nil {
		t.Fatalf("access did not become ready: %v", err)
	}
	return access
}

// testCoreV2Identity 是 CLI 测试固定的 DeviceIdentity；Phase 4 后 identity
// 由 access 直答，pool 不再挂载 client access service。
func testCoreV2Identity() remoteauth.Identity {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, ed25519.SeedSize))
	identity, err := remoteauth.NewIdentity("device-cli-test", privateKey)
	if err != nil {
		panic(err)
	}
	return identity
}

func newCoreV2ProtocolClientForCLITestWithOptions(t *testing.T, opts ...corev2.ServerOption) (*corev2.Server, *protocol.Client, func()) {
	t.Helper()
	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	serverOpts := append([]corev2.ServerOption{corev2.WithSocketPath(socketPath + ".provider")}, opts...)
	server := newCoreV2TestServer(serverOpts...)
	stopProvider := startCoreV2ProviderServer(t, server, socketPath+".provider")
	startCLIAccessServer(t, socketPath)
	client, err := dialV3Client(socketPath)
	if err != nil {
		stopProvider()
		_ = server.Shutdown(context.Background())
		t.Fatalf("dial core-v2 pool: %v", err)
	}
	closeFn := func() {
		_ = client.Close()
		stopProvider()
		_ = server.Shutdown(context.Background())
	}
	return server, client, closeFn
}

// startCoreV2ProviderServer 在 providerSocket 上启动 provider 协议 listener，
// 返回停止函数；pool provider 就绪后才能启动 access。
func startCoreV2ProviderServer(t *testing.T, server *corev2.Server, providerSocket string) func() {
	t.Helper()
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("start core-v2 terminal server: %v", err)
	}
	providerServer, err := providercore.New(server, providercore.Config{Socket: providerSocket})
	if err != nil {
		t.Fatalf("create terminal provider server: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- providerServer.ListenAndServe(ctx) }()
	if err := waitForSocket(providerSocket, 2*time.Second, func() error {
		return probeProviderSocket(providerSocket)
	}); err != nil {
		cancel()
		_ = providerServer.Shutdown(context.Background())
		t.Fatalf("pool provider did not become ready: %v", err)
	}
	return func() {
		cancel()
		_ = providerServer.Shutdown(context.Background())
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("provider server did not stop in time")
		}
	}
}

type coreV2ResizeRecordingProcessFactory struct {
	mu        sync.Mutex
	processes map[string]*coreV2ResizeRecordingProcess
}

func newCoreV2ResizeRecordingProcessFactory() *coreV2ResizeRecordingProcessFactory {
	return &coreV2ResizeRecordingProcessFactory{processes: make(map[string]*coreV2ResizeRecordingProcess)}
}

func (factory *coreV2ResizeRecordingProcessFactory) Spawn(_ context.Context, spec corev2.ProcessSpec) (corev2.TerminalProcess, error) {
	process := &coreV2ResizeRecordingProcess{
		outputCh: make(chan []byte, 16),
		waitCh:   make(chan corev2.ProcessExit, 1),
	}
	factory.mu.Lock()
	factory.processes[spec.TerminalID] = process
	factory.mu.Unlock()
	return process, nil
}

func (factory *coreV2ResizeRecordingProcessFactory) process(id string) *coreV2ResizeRecordingProcess {
	factory.mu.Lock()
	defer factory.mu.Unlock()
	return factory.processes[id]
}

type coreV2ResizeRecordingProcess struct {
	mu         sync.Mutex
	inputs     [][]byte
	resizes    []corev2.Size
	resizeErr  error
	outputCh   chan []byte
	waitCh     chan corev2.ProcessExit
	exitOnce   sync.Once
	outputOnce sync.Once
	closed     bool
}

func (process *coreV2ResizeRecordingProcess) Input(data []byte) error {
	process.mu.Lock()
	defer process.mu.Unlock()
	if process.closed {
		return io.ErrClosedPipe
	}
	process.inputs = append(process.inputs, append([]byte(nil), data...))
	return nil
}

func (process *coreV2ResizeRecordingProcess) Resize(size corev2.Size) error {
	process.mu.Lock()
	defer process.mu.Unlock()
	if process.closed {
		return io.ErrClosedPipe
	}
	process.resizes = append(process.resizes, size)
	if process.resizeErr != nil {
		return process.resizeErr
	}
	return nil
}

func (process *coreV2ResizeRecordingProcess) Output() <-chan []byte {
	return process.outputCh
}

func (process *coreV2ResizeRecordingProcess) CancelOutput() {
	process.closeOutput()
}

func (process *coreV2ResizeRecordingProcess) Kill() error {
	process.mu.Lock()
	process.closed = true
	process.mu.Unlock()
	process.exit(-1)
	return nil
}

func (process *coreV2ResizeRecordingProcess) Wait() <-chan corev2.ProcessExit {
	return process.waitCh
}

func (process *coreV2ResizeRecordingProcess) Close() error {
	process.mu.Lock()
	process.closed = true
	process.mu.Unlock()
	process.exit(-1)
	return nil
}

func (process *coreV2ResizeRecordingProcess) exit(code int) {
	process.mu.Lock()
	process.closed = true
	process.mu.Unlock()
	process.exitOnce.Do(func() {
		process.closeOutput()
		process.waitCh <- corev2.ProcessExit{Code: code}
		close(process.waitCh)
	})
}

func (process *coreV2ResizeRecordingProcess) closeOutput() {
	process.outputOnce.Do(func() {
		close(process.outputCh)
	})
}

func (process *coreV2ResizeRecordingProcess) setResizeErr(err error) {
	process.mu.Lock()
	defer process.mu.Unlock()
	process.resizeErr = err
}

func (process *coreV2ResizeRecordingProcess) resizeCount() int {
	process.mu.Lock()
	defer process.mu.Unlock()
	return len(process.resizes)
}

func (process *coreV2ResizeRecordingProcess) snapshot() ([][]byte, []corev2.Size) {
	process.mu.Lock()
	defer process.mu.Unlock()
	inputs := make([][]byte, len(process.inputs))
	for i, input := range process.inputs {
		inputs[i] = append([]byte(nil), input...)
	}
	resizes := append([]corev2.Size(nil), process.resizes...)
	return inputs, resizes
}

func waitForCoreV2ResizeRecordingProcess(t *testing.T, factory *coreV2ResizeRecordingProcessFactory, terminalID string) *coreV2ResizeRecordingProcess {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if process := factory.process(terminalID); process != nil {
			return process
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for core-v2 recording process %s", terminalID)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitForCLITerminalState(t *testing.T, server *corev2.Server, terminalID string, want corev2.TerminalState) corev2.TerminalInfo {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		info, err := server.GetTerminal(terminalID)
		if err == nil && info.State == want {
			return info
		}
		if time.Now().After(deadline) {
			if err != nil {
				t.Fatalf("timed out waiting for terminal %s state %s: %v", terminalID, want, err)
			}
			t.Fatalf("timed out waiting for terminal %s state %s, got %#v", terminalID, want, info)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitForCoreV2ProcessResize(t *testing.T, process *coreV2ResizeRecordingProcess, want corev2.Size) int {
	t.Helper()
	return waitForCoreV2ProcessResizeAfter(t, process, want, 0)
}

func waitForCoreV2ProcessResizeAfter(t *testing.T, process *coreV2ResizeRecordingProcess, want corev2.Size, after int) int {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		_, resizes := process.snapshot()
		if after < 0 {
			after = 0
		}
		for index := after; index < len(resizes); index++ {
			if resizes[index] == want {
				return index + 1
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for process resize %#v after %d, got %#v", want, after, resizes)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func parseV3TmuxSize(value string) (int, int, bool) {
	cols, rows, ok := strings.Cut(value, "x")
	if !ok {
		return 0, 0, false
	}
	parsedCols, errCols := strconv.Atoi(cols)
	parsedRows, errRows := strconv.Atoi(rows)
	if errCols != nil || errRows != nil {
		return 0, 0, false
	}
	return parsedCols, parsedRows, true
}

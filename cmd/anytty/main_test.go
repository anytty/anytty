package main

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

	apilayer "github.com/anytty/anytty/api_layer"
	localadapter "github.com/anytty/anytty/client/adapter/local"
	clientprotocol "github.com/anytty/anytty/client/adapter/protocol"
	endpointdomain "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport"
	tuiv3 "github.com/anytty/anytty/tui"
	actiondomain "github.com/anytty/anytty/tui/action"
	clientruntimeadapter "github.com/anytty/anytty/tui/adapter/clientruntime"
	protocoladapter "github.com/anytty/anytty/tui/adapter/protocol"
	"github.com/anytty/anytty/tui/app"
	tuiinput "github.com/anytty/anytty/tui/input"
	tuiservices "github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	tuistate "github.com/anytty/anytty/tui/state"
)

func TestRootCmdRoutesToTUIv3ByDefault(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRunRoot := runV3Root
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runV3Root = oldRunRoot
	})

	isInteractiveTerminal = func() bool { return true }
	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	logPath := filepath.Join(t.TempDir(), "anytty.log")
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	legacyDir := filepath.Join(configHome, "anytty")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "connections.yaml"), []byte("version: 1\ndefault: local\nconnections:\n  local:\n    transport: local\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var gotCfg v3RootConfig
	calledRoot := false
	runV3Root = func(ctx context.Context, cfg v3RootConfig) error {
		calledRoot = true
		gotCfg = cfg
		return nil
	}

	explicitConfig := filepath.Join(t.TempDir(), "tui-v3.yaml")
	if err := os.WriteFile(explicitConfig, []byte("version: 1\n"), 0o600); err != nil {
		t.Fatal(err)
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
		t.Fatal("expected default root command to call tui-v3 root runner")
	}
	if gotCfg.SocketPath != socketPath || gotCfg.LogFile != logPath {
		t.Fatalf("unexpected v3 root config %#v", gotCfg)
	}
	if gotCfg.ConnectionRegistry.Default != endpointdomain.DefaultEndpointID {
		t.Fatalf("legacy connections.yaml blocked default local TUI registry: %#v", gotCfg.ConnectionRegistry)
	}
	configPath := filepath.Join(configHome, "anytty", "anytty.yaml")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("default root must not create tuiv2 config at %s, stat err=%v", configPath, err)
	}
}

func TestRootCmdExposesBuildVersion(t *testing.T) {
	if got := newRootCmd().Version; got != version {
		t.Fatalf("root version = %q, want %q", got, version)
	}
}

func TestRootCmdLoadsConnectionRegistryLocalSocket(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRunRoot := runV3Root
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runV3Root = oldRunRoot
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

	var gotCfg v3RootConfig
	runV3Root = func(ctx context.Context, cfg v3RootConfig) error {
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
	if gotCfg.SocketPath != socketPath {
		t.Fatalf("expected registry socket %q, got %q", socketPath, gotCfg.SocketPath)
	}
	local, ok := gotCfg.ConnectionRegistry.Endpoints[endpointdomain.DefaultEndpointID]
	if !ok {
		t.Fatalf("local connection missing from registry %#v", gotCfg.ConnectionRegistry)
	}
	if route, ok := local.Route(endpointdomain.DefaultLocalRouteID); local.Label != "Configured Local" || !ok || route.Socket != socketPath {
		t.Fatalf("unexpected local connection %#v", local)
	}
}

func TestRootCmdSocketFlagOverridesConnectionRegistrySocket(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRunRoot := runV3Root
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runV3Root = oldRunRoot
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

	var gotCfg v3RootConfig
	runV3Root = func(ctx context.Context, cfg v3RootConfig) error {
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
	if gotCfg.SocketPath != flagSocket {
		t.Fatalf("expected --socket to override registry socket %q, got %q", flagSocket, gotCfg.SocketPath)
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

func TestV3InteractiveRuntimeInitializesEndpointStoreFromRegistry(t *testing.T) {
	registry := endpointdomain.Registry{
		Version: endpointdomain.RegistryVersion,
		Default: endpointdomain.DefaultEndpointID,
		Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{
			endpointdomain.DefaultEndpointID: testLocalEndpoint(endpointdomain.DefaultEndpointID, "Runtime Local", "auto", endpointdomain.ConnectAuto, true),
		},
	}
	runtime := newV3InteractiveRuntimeWithOptions("", 80, 24, nil, app.NewFakeTerminalHost(8), nil, v3InteractiveRuntimeOptions{
		SkipWorkbenchInitialLoad: true,
		ConnectionRegistry:       registry,
	})
	local, ok := runtime.State().Endpoints.Endpoint(tuistate.DefaultEndpointID)
	if !ok {
		t.Fatalf("local endpoint missing from runtime state %#v", runtime.State().Endpoints)
	}
	if local.DisplayLabel() != "Runtime Local" || local.DisplayStatus() != tuistate.EndpointStatusAuto {
		t.Fatalf("unexpected local endpoint projection %#v", local)
	}
}

func TestV3InteractiveRuntimePreservesInitialRemoteTerminalRef(t *testing.T) {
	registry := endpointdomain.Registry{
		Version: endpointdomain.RegistryVersion,
		Default: "west",
		Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{
			"west": testSSHEndpoint("west", "West", "west.example", "", endpointdomain.ConnectOnDemand, true),
		},
	}
	runtime := newV3InteractiveRuntimeWithOptions("term-1", 80, 24, nil, app.NewFakeTerminalHost(8), nil, v3InteractiveRuntimeOptions{
		SkipWorkbenchInitialLoad: true,
		InitialEndpointID:        "west",
		ConnectionRegistry:       registry,
	})
	root := runtime.State()
	if root.Session.EndpointID != "west" || root.Session.TerminalID != "term-1" || root.Surface.EndpointID != "west" || root.Surface.TerminalID != "term-1" {
		t.Fatalf("remote terminal ref was rewritten during initialization: session=%#v surface=%#v", root.Session.TerminalRef(), root.Surface.TerminalRef())
	}
}

func TestRootCmdBlocksNestedTUIByDefault(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRunRoot := runV3Root
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runV3Root = oldRunRoot
	})

	isInteractiveTerminal = func() bool { return true }
	runV3Root = func(ctx context.Context, cfg v3RootConfig) error {
		t.Fatal("runV3Root should not be called when nested TUI is blocked")
		return nil
	}

	t.Setenv("ANYTTY", "1")
	t.Setenv("ANYTTY_ALLOW_NESTED", "")

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--log-file", filepath.Join(t.TempDir(), "anytty.log")})
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
	oldRunAttach := runV3Attach
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runV3Attach = oldRunAttach
	})

	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	logPath := filepath.Join(t.TempDir(), "anytty.log")
	isInteractiveTerminal = func() bool { return true }
	var got v3AttachConfig
	runV3Attach = func(ctx context.Context, cfg v3AttachConfig) error {
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
	if got.TerminalID != "term-001" || got.SocketPath != socketPath || got.LogFile != logPath {
		t.Fatalf("unexpected v3 attach config %#v", got)
	}
}
func TestAttachCmdAllowsNestedTUIWhenOverrideIsSet(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRunAttach := runV3Attach
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runV3Attach = oldRunAttach
	})

	isInteractiveTerminal = func() bool { return true }
	called := false
	runV3Attach = func(ctx context.Context, cfg v3AttachConfig) error {
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
		t.Fatal("expected attach command to reach tui-v3 runner when override is set")
	}
}

func TestAttachCmdBlocksNestedTUIByDefault(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRunAttach := runV3Attach
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runV3Attach = oldRunAttach
	})

	isInteractiveTerminal = func() bool { return true }
	runV3Attach = func(ctx context.Context, cfg v3AttachConfig) error {
		t.Fatal("runV3Attach should not be called when nested attach is blocked")
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
func TestDefaultRootDoesNotRunTUIv2OrV3Smoke(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRunRoot := runV3Root
	oldRunSmoke := runTUIv3Smoke
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runV3Root = oldRunRoot
		runTUIv3Smoke = oldRunSmoke
	})

	isInteractiveTerminal = func() bool { return true }
	calledRoot := false
	calledV3Smoke := false
	runV3Root = func(ctx context.Context, cfg v3RootConfig) error {
		calledRoot = true
		return nil
	}

	runTUIv3Smoke = func(ctx context.Context) (render.Frame, error) {
		calledV3Smoke = true
		return render.Frame{}, nil
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--log-file", filepath.Join(t.TempDir(), "anytty.log")})
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !calledRoot {
		t.Fatal("expected default root command to call tui-v3 root runner")
	}
	if calledV3Smoke {
		t.Fatal("default root command must not run tui-v3 smoke")
	}
}

func TestV3SmokeRunsTUIv3Smoke(t *testing.T) {
	oldRunSmoke := runTUIv3SmokeDetailed
	t.Cleanup(func() {
		runTUIv3SmokeDetailed = oldRunSmoke
	})

	called := false
	runTUIv3SmokeDetailed = func(ctx context.Context) (tuiv3.SmokeResult, error) {
		called = true
		return tuiv3.SmokeResult{Cases: []tuiv3.SmokeCase{{Name: "case-a", Frame: render.Frame{Lines: []string{"v3-line"}}}}}, nil
	}

	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "smoke"})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !called {
		t.Fatal("expected v3 smoke to call tui-v3 smoke runner")
	}
	text := out.String()
	if !strings.Contains(text, "anytty v3 smoke ok") ||
		!strings.Contains(text, "tui") ||
		!strings.Contains(text, "case: case-a") ||
		!strings.Contains(text, "v3-line") {
		t.Fatalf("unexpected v3 smoke output:\n%s", text)
	}
}

func TestV3SmokeCommandIncludesVisualReviewCases(t *testing.T) {
	oldRunSmoke := runTUIv3SmokeDetailed
	t.Cleanup(func() {
		runTUIv3SmokeDetailed = oldRunSmoke
	})
	runTUIv3SmokeDetailed = tuiv3.SmokeRunDetailed

	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "smoke"})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"anytty v3 smoke ok: tui=tui cases=12",
		"case: terminal-pool-page",
		"Terminal Manager",
		"⌕ search 日志",
		"[Ctrl+R] RESTART",
		"case: workbench-tree-page",
		"Workbench Navigator",
		"shell page base",
		"WORKBENCH-TREE  • [enter] OPEN • [Ctrl+N] NEW • [Ctrl+R] RENAME",
		"case: visual-audit-current",
		"visual review",
		"[↗]─[×]",
		"case: copy-history",
		"copy row",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v3 smoke output missing visual review marker %q:\n%s", want, text)
		}
	}
	for _, stale := range []string{"visual acceptance"} {
		if strings.Contains(text, stale) {
			t.Fatalf("v3 smoke output must not claim visual acceptance %q:\n%s", stale, text)
		}
	}
}

func TestV3PaneCommandAdapterParsesMiniCommand(t *testing.T) {
	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "pane-command", "pane", "resize", "right", "delta=4", "pane=main"})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "action=pane.resize") || !strings.Contains(text, "pane=main") || !strings.Contains(text, "source=cli-mini") {
		t.Fatalf("unexpected pane command adapter output:\n%s", text)
	}
}
func TestDefaultDaemonUsesCoreV2Server(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	obsoleteDir := filepath.Join(stateHome, "anytty", "core-v2-history")
	if err := os.MkdirAll(obsoleteDir, 0o755); err != nil {
		t.Fatal(err)
	}
	obsoletePath := filepath.Join(obsoleteDir, "old.compact")
	if err := os.WriteFile(obsoletePath, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldNewCoreV2Server := newCoreV2Server
	t.Cleanup(func() {
		newCoreV2Server = oldNewCoreV2Server
	})

	fakeV3 := &fakeCoreV2Server{}
	newCoreV2Server = func(opts ...corev2.ServerOption) coreV2Server {
		fakeV3.newServerCalls++
		server := newCoreV2TestServer(opts...)
		if server.HistoryStorageDir() == "" {
			t.Fatal("default daemon must configure file-backed core-v2 history storage dir")
		}
		if got := server.HistoryStorageConfig(); got.MaxBytesPerTerminal != corev2.DefaultHistoryMaxBytesPerTerminal || got.Compression != corev2.HistoryCompressionZstd || got.CompressionLevel != corev2.HistoryCompressionLevelFast {
			t.Fatalf("unexpected default daemon history storage config: %#v", got)
		}
		if got := server.TerminalResourceSamplingConfig(); got.Interval != 500*time.Millisecond || got.MaxSamples != 512 {
			t.Fatalf("unexpected default daemon resource sampling config: %#v", got)
		}
		return fakeV3
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--socket", filepath.Join(t.TempDir(), "anytty-v2.sock"), "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "daemon"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if fakeV3.newServerCalls != 1 || fakeV3.listenCalls != 1 || fakeV3.shutdownCalls != 1 {
		t.Fatalf("unexpected core-v2 fake server calls: new=%d listen=%d shutdown=%d", fakeV3.newServerCalls, fakeV3.listenCalls, fakeV3.shutdownCalls)
	}
	if _, err := os.Stat(obsoletePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("obsolete compact history remains: %v", err)
	}
}

func TestDaemonAppliesHistoryStorageConfigFile(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	oldNewCoreV2Server := newCoreV2Server
	t.Cleanup(func() { newCoreV2Server = oldNewCoreV2Server })
	configPath := filepath.Join(t.TempDir(), "anytty.yaml")
	if err := os.WriteFile(configPath, []byte("version: 1\ndaemon:\n  history:\n    max_size_mb: 64\n    max_age_days: 14\n    compression: s2\n    compression_level: best\n  resource_sampling:\n    interval_ms: 750\n    max_samples: 1024\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fakeV3 := &fakeCoreV2Server{}
	newCoreV2Server = func(opts ...corev2.ServerOption) coreV2Server {
		server := newCoreV2TestServer(opts...)
		if got := server.HistoryStorageConfig(); got.MaxBytesPerTerminal != 64<<20 || got.MaxAge != 14*24*time.Hour || got.Compression != corev2.HistoryCompressionS2 || got.CompressionLevel != corev2.HistoryCompressionLevelBest {
			t.Fatalf("daemon did not apply history storage config: %#v", got)
		}
		if got := server.TerminalResourceSamplingConfig(); got.Interval != 750*time.Millisecond || got.MaxSamples != 1024 {
			t.Fatalf("daemon did not apply resource sampling config: %#v", got)
		}
		return fakeV3
	}
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--config", configPath, "--socket", filepath.Join(t.TempDir(), "anytty-v2.sock"), "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "daemon"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestDaemonCanDisableHistoryFromEnv(t *testing.T) {
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
			t.Fatalf("history disabled daemon must not configure history storage dir, got %q", server.HistoryStorageDir())
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
	cmd.SetArgs([]string{"--socket", filepath.Join(t.TempDir(), "anytty-v2.sock"), "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "daemon"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if fakeV3.newServerCalls != 1 {
		t.Fatalf("expected one core-v2 server construction, got %d", fakeV3.newServerCalls)
	}
}

func TestDaemonConfiguresTerminalOutputBufferFromEnv(t *testing.T) {
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
			t.Fatalf("daemon did not pass output buffer env to core: %#v", got)
		}
		if budget := server.TerminalOutputResidentBudget(); budget != 256<<20 {
			t.Fatalf("daemon did not pass resident budget env to core: %d", budget)
		}
		if resources := server.TerminalResourceSamplingConfig(); resources.Interval != 750*time.Millisecond || resources.MaxSamples != 1024 {
			t.Fatalf("daemon did not pass resource sampling env to core: %#v", resources)
		}
		return fakeV3
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--socket", filepath.Join(t.TempDir(), "anytty-v2.sock"), "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "daemon"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if fakeV3.newServerCalls != 1 {
		t.Fatalf("expected one core-v2 server construction, got %d", fakeV3.newServerCalls)
	}
}

func TestV3PingConnectsExistingCoreV2Daemon(t *testing.T) {
	oldConnect := connectV3EndpointApplication
	oldStart := startV3Daemon
	t.Cleanup(func() {
		connectV3EndpointApplication = oldConnect
		startV3Daemon = oldStart
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
	startV3Daemon = func(path string, logFile string) error {
		t.Fatal("v3 ping must not auto-start when existing daemon is reachable")
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
		t.Fatal("expected v3 ping to dial core-v2 daemon")
	}
	if !strings.Contains(out.String(), "anytty v3 daemon ok") || !strings.Contains(out.String(), socketPath) {
		t.Fatalf("unexpected v3 ping output:\n%s", out.String())
	}
}

func TestV3PingAutoStartsCoreV2Daemon(t *testing.T) {
	oldConnect := connectV3EndpointApplication
	oldStart := startV3Daemon
	t.Cleanup(func() {
		connectV3EndpointApplication = oldConnect
		startV3Daemon = oldStart
	})

	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	logPath := filepath.Join(t.TempDir(), "anytty.log")
	connectCalls := 0
	startCalls := 0
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
	startV3Daemon = func(path string, logFile string) error {
		startCalls++
		startedSocket = path
		startedLog = logFile
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
		t.Fatalf("unexpected v3 daemon auto-start: calls=%d socket=%q log=%q", startCalls, startedSocket, startedLog)
	}
	if !strings.Contains(out.String(), "anytty v3 daemon ok") {
		t.Fatalf("unexpected v3 ping output:\n%s", out.String())
	}
}

func TestV3PingReturnsAutoStartError(t *testing.T) {
	oldConnect := connectV3EndpointApplication
	oldStart := startV3Daemon
	t.Cleanup(func() {
		connectV3EndpointApplication = oldConnect
		startV3Daemon = oldStart
	})

	connectV3EndpointApplication = func(ctx context.Context, _ *clientruntime.SessionOwner, _ endpointdomain.Endpoint, _ endpointdomain.RouteID, _ clientruntime.ConnectIntent, options localadapter.Options, _ *slog.Logger) (*clientprotocol.ApplicationClient, endpointdomain.AccessRoute, error) {
		return nil, endpointdomain.AccessRoute{}, options.Start(ctx, options.SocketOverride)
	}
	startV3Daemon = func(path string, logFile string) error {
		return os.ErrPermission
	}

	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"--socket", filepath.Join(t.TempDir(), "anytty-v2.sock"), "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "v3", "ping"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "start core-v2 daemon") || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected auto-start error, got %v", err)
	}
}

func TestV3PingConnectsRealCoreV2Daemon(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	server := newCoreV2TestServer(corev2.WithSocketPath(socketPath))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- server.ListenAndServe(ctx)
	}()
	defer func() {
		cancel()
		_ = server.Shutdown(context.Background())
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("core-v2 server did not stop in time")
		}
	}()
	if err := waitForSocket(socketPath, 2*time.Second, func() error {
		client, err := dialV3Client(socketPath)
		if err != nil {
			return err
		}
		return client.Close()
	}); err != nil {
		t.Fatalf("core-v2 daemon did not become ready: %v", err)
	}

	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"--socket", socketPath, "--log-file", filepath.Join(t.TempDir(), "anytty.log"), "v3", "ping"})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(out.String(), "anytty v3 daemon ok") || !strings.Contains(out.String(), socketPath) {
		t.Fatalf("unexpected v3 ping output:\n%s", out.String())
	}
}

func TestStartCoreV2DaemonCommandUsesV3Daemon(t *testing.T) {
	oldExecutable := osExecutable
	t.Cleanup(func() {
		osExecutable = oldExecutable
	})

	exe := filepath.Join(t.TempDir(), "anytty")
	osExecutable = func() (string, error) {
		return exe, nil
	}
	got, err := buildStartCoreV2DaemonCommand("/tmp/anytty-v2.sock", "/tmp/anytty.log")
	if err != nil {
		t.Fatalf("buildStartCoreV2DaemonCommand returned error: %v", err)
	}
	if got.Path != exe {
		t.Fatalf("expected executable %q, got %q", exe, got.Path)
	}
	wantArgs := []string{exe, "--socket", "/tmp/anytty-v2.sock", "--log-file", "/tmp/anytty.log", "daemon"}
	if !reflect.DeepEqual(got.Args, wantArgs) {
		t.Fatalf("unexpected v3 daemon args: %#v", got.Args)
	}
}

func TestStartCoreV2DaemonCommandCarriesHistoryDisableEnv(t *testing.T) {
	oldExecutable := osExecutable
	t.Cleanup(func() {
		osExecutable = oldExecutable
	})
	t.Setenv("ANYTTY_HISTORY_DISABLE", "1")

	exe := filepath.Join(t.TempDir(), "anytty")
	osExecutable = func() (string, error) {
		return exe, nil
	}
	got, err := buildStartCoreV2DaemonCommand("/tmp/anytty-v2.sock", "/tmp/anytty.log")
	if err != nil {
		t.Fatalf("buildStartCoreV2DaemonCommand returned error: %v", err)
	}
	if got.Path != exe {
		t.Fatalf("expected executable %q, got %q", exe, got.Path)
	}
	if !containsEnv(got.Env, "ANYTTY_HISTORY_DISABLE=1") {
		t.Fatalf("auto-start daemon command must carry history disabled env, env=%#v", got.Env)
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

func TestStartCoreV2DaemonCommandCanCarryExplicitConfigPath(t *testing.T) {
	oldExecutable := osExecutable
	t.Cleanup(func() {
		osExecutable = oldExecutable
	})

	exe := filepath.Join(t.TempDir(), "anytty")
	configPath := filepath.Join(t.TempDir(), "anytty.yaml")
	osExecutable = func() (string, error) {
		return exe, nil
	}
	got, err := buildStartCoreV2DaemonCommandWithConfig("/tmp/anytty-v2.sock", "/tmp/anytty.log", configPath)
	if err != nil {
		t.Fatalf("buildStartCoreV2DaemonCommandWithConfig returned error: %v", err)
	}
	wantArgs := []string{exe, "--socket", "/tmp/anytty-v2.sock", "--log-file", "/tmp/anytty.log", "--config", configPath, "daemon"}
	if !reflect.DeepEqual(got.Args, wantArgs) {
		t.Fatalf("unexpected v3 daemon args with config: %#v", got.Args)
	}
}

func TestDialOrStartV3ClientUsesConfigStarterWhenConfigPathIsExplicit(t *testing.T) {
	oldDial := v3DialClient
	oldStart := startV3Daemon
	oldStartWithConfig := startV3DaemonWithConfig
	oldConnect := connectV3EndpointApplication
	t.Cleanup(func() {
		v3DialClient = oldDial
		startV3Daemon = oldStart
		startV3DaemonWithConfig = oldStartWithConfig
		connectV3EndpointApplication = oldConnect
	})
	configPath := filepath.Join(t.TempDir(), "anytty.yaml")
	socketPath := filepath.Join(t.TempDir(), "anytty.sock")
	logPath := filepath.Join(t.TempDir(), "anytty.log")
	startV3Daemon = func(path string, logFile string) error {
		t.Fatal("plain daemon starter must not be used when explicit config path is present")
		return nil
	}
	var gotSocket, gotLog, gotConfig string
	startV3DaemonWithConfig = func(path string, logFile string, cfg string) error {
		gotSocket, gotLog, gotConfig = path, logFile, cfg
		return nil
	}
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

func TestDialOrStartV3TUIClientUsesTUIEndpointClientName(t *testing.T) {
	oldConnect := connectV3EndpointApplication
	t.Cleanup(func() { connectV3EndpointApplication = oldConnect })

	socketPath := filepath.Join(t.TempDir(), "anytty.sock")
	var receivedClientName string
	connectV3EndpointApplication = func(_ context.Context, _ *clientruntime.SessionOwner, _ endpointdomain.Endpoint, _ endpointdomain.RouteID, _ clientruntime.ConnectIntent, options localadapter.Options, _ *slog.Logger) (*clientprotocol.ApplicationClient, endpointdomain.AccessRoute, error) {
		receivedClientName = options.ClientName
		return nil, endpointdomain.AccessRoute{}, nil
	}

	client, err := dialOrStartV3TUIClient(socketPath, filepath.Join(t.TempDir(), "anytty.log"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if client != nil {
		t.Fatal("client name fixture unexpectedly returned a protocol client")
	}
	if receivedClientName != tuiEndpointClientName {
		t.Fatalf("TUI root client name = %q, want %q", receivedClientName, tuiEndpointClientName)
	}
}

func TestDefaultLocalControlCommandsUseCoreV2Protocol(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	server := newCoreV2TestServer(corev2.WithSocketPath(socketPath))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- server.ListenAndServe(ctx)
	}()
	defer func() {
		cancel()
		_ = server.Shutdown(context.Background())
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("core-v2 server did not stop in time")
		}
	}()
	if err := waitForSocket(socketPath, 2*time.Second, func() error {
		client, err := dialV3Client(socketPath)
		if err != nil {
			return err
		}
		return client.Close()
	}); err != nil {
		t.Fatalf("core-v2 daemon did not become ready: %v", err)
	}

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
	server := newCoreV2TestServer(corev2.WithSocketPath(socketPath))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- server.ListenAndServe(ctx)
	}()
	defer func() {
		cancel()
		_ = server.Shutdown(context.Background())
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("core-v2 server did not stop in time")
		}
	}()
	if err := waitForSocket(socketPath, 2*time.Second, func() error {
		client, err := dialV3Client(socketPath)
		if err != nil {
			return err
		}
		return client.Close()
	}); err != nil {
		t.Fatalf("core-v2 daemon did not become ready: %v", err)
	}

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
		corev2.WithSocketPath(socketPath),
		corev2.WithProcessFactory(newCoreV2ResizeRecordingProcessFactory()),
		corev2.WithTerminalOutputBufferConfig(corev2.TerminalOutputBufferConfig{
			Overflow:      corev2.TerminalOutputOverflowBlock,
			CapacityBytes: 64 << 10,
		}),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- server.ListenAndServe(ctx)
	}()
	defer func() {
		cancel()
		_ = server.Shutdown(context.Background())
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("core-v2 server did not stop in time")
		}
	}()
	if err := waitForSocket(socketPath, 2*time.Second, func() error {
		client, err := dialV3Client(socketPath)
		if err != nil {
			return err
		}
		return client.Close()
	}); err != nil {
		t.Fatalf("core-v2 daemon did not become ready: %v", err)
	}
	client, err := dialV3Client(socketPath)
	if err != nil {
		t.Fatalf("dial core-v2 daemon: %v", err)
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
	oldRunAttach := runV3Attach
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runV3Attach = oldRunAttach
	})

	isInteractiveTerminal = func() bool { return false }
	runV3Attach = func(ctx context.Context, cfg v3AttachConfig) error {
		t.Fatal("non-interactive v3 attach must not start runtime")
		return nil
	}

	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "attach", "term-1"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "anytty terminal attach requires an interactive terminal") {
		t.Fatalf("expected non-interactive attach error, got %v", err)
	}
}

func TestV3AttachRoutesToTUIv3Runtime(t *testing.T) {
	oldInteractive := isInteractiveTerminal
	oldRunAttach := runV3Attach
	t.Cleanup(func() {
		isInteractiveTerminal = oldInteractive
		runV3Attach = oldRunAttach
	})

	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	logPath := filepath.Join(t.TempDir(), "anytty.log")
	isInteractiveTerminal = func() bool { return true }
	var got v3AttachConfig
	runV3Attach = func(ctx context.Context, cfg v3AttachConfig) error {
		got = cfg
		return nil
	}

	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"--socket", socketPath, "--log-file", logPath, "v3", "attach", "term-1"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got.TerminalID != "term-1" || got.SocketPath != socketPath || got.LogFile != logPath {
		t.Fatalf("unexpected v3 attach config %#v", got)
	}
}

func TestV3AttachRoutesFileLoggerIntoEndpointRuntime(t *testing.T) {
	oldConnect := connectV3EndpointApplication
	t.Cleanup(func() { connectV3EndpointApplication = oldConnect })

	expectedLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	var receivedLogger *slog.Logger
	connectV3EndpointApplication = func(_ context.Context, _ *clientruntime.SessionOwner, _ endpointdomain.Endpoint, _ endpointdomain.RouteID, _ clientruntime.ConnectIntent, _ localadapter.Options, logger *slog.Logger) (*clientprotocol.ApplicationClient, endpointdomain.AccessRoute, error) {
		receivedLogger = logger
		return nil, endpointdomain.AccessRoute{}, nil
	}

	registry := endpointdomain.DefaultRegistry()
	client, _, err := openV3AttachProtocolClientsWithClientRuntime(context.Background(), v3AttachConfig{
		EndpointID:         tuistate.DefaultEndpointID,
		ConnectionRegistry: registry,
	}, filepath.Join(t.TempDir(), "anytty.log"), expectedLogger)
	if err != nil {
		t.Fatal(err)
	}
	if client != nil {
		t.Fatal("logger routing fixture unexpectedly returned a protocol client")
	}
	if receivedLogger != expectedLogger {
		t.Fatal("TUI attach did not pass its file logger into the endpoint runtime")
	}
}

func TestV3AttachUsesTUIEndpointClientName(t *testing.T) {
	oldConnect := connectV3EndpointApplication
	t.Cleanup(func() { connectV3EndpointApplication = oldConnect })

	var receivedClientName string
	connectV3EndpointApplication = func(_ context.Context, _ *clientruntime.SessionOwner, _ endpointdomain.Endpoint, _ endpointdomain.RouteID, _ clientruntime.ConnectIntent, options localadapter.Options, _ *slog.Logger) (*clientprotocol.ApplicationClient, endpointdomain.AccessRoute, error) {
		receivedClientName = options.ClientName
		return nil, endpointdomain.AccessRoute{}, nil
	}

	registry := endpointdomain.DefaultRegistry()
	client, _, err := openV3AttachProtocolClientsWithClientRuntime(context.Background(), v3AttachConfig{
		EndpointID:         tuistate.DefaultEndpointID,
		ConnectionRegistry: registry,
	}, filepath.Join(t.TempDir(), "anytty.log"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if client != nil {
		t.Fatal("client name fixture unexpectedly returned a protocol client")
	}
	if receivedClientName != tuiEndpointClientName {
		t.Fatalf("TUI attach client name = %q, want %q", receivedClientName, tuiEndpointClientName)
	}
}

func TestV3RootRuntimeWithoutTerminalOpensPickerWithoutCreatingTerminal(t *testing.T) {
	server, client, closeClient := newCoreV2ProtocolClientForCLITest(t)
	defer closeClient()
	oldConnect := connectV3EndpointApplication
	oldStart := startV3Daemon
	oldRunAttach := runV3Attach
	oldRunEmpty := runV3RootEmpty
	t.Cleanup(func() {
		connectV3EndpointApplication = oldConnect
		startV3Daemon = oldStart
		runV3Attach = oldRunAttach
		runV3RootEmpty = oldRunEmpty
	})

	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	logPath := filepath.Join(t.TempDir(), "anytty.log")
	installV3LocalApplicationTestClient(t, socketPath, client)
	startV3Daemon = func(path string, logFile string) error {
		t.Fatal("v3 root must not auto-start when injected client is available")
		return nil
	}
	var gotEmpty v3RootEmptyConfig
	runV3Attach = func(ctx context.Context, cfg v3AttachConfig) error {
		t.Fatalf("v3 root must not attach when the core-v2 terminal list is empty, cfg=%#v", cfg)
		return nil
	}
	runV3RootEmpty = func(ctx context.Context, cfg v3RootEmptyConfig) error {
		gotEmpty = cfg
		return nil
	}

	if err := runV3RootRuntime(context.Background(), v3RootConfig{SocketPath: socketPath, LogFile: logPath}); err != nil {
		t.Fatalf("runV3RootRuntime returned error: %v", err)
	}
	if gotEmpty.SocketPath != socketPath || gotEmpty.LogFile != logPath {
		t.Fatalf("unexpected v3 root empty config %#v", gotEmpty)
	}
	if _, err := server.GetTerminal(v3RootTerminalID); err == nil {
		t.Fatal("v3 root should not auto-create the default terminal on empty startup")
	}
}

func TestV3RootEmptyRuntimeEnablesPerfTrace(t *testing.T) {
	oldDial := v3DialClient
	oldStart := startV3Daemon
	t.Cleanup(func() {
		v3DialClient = oldDial
		startV3Daemon = oldStart
	})

	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	logPath := filepath.Join(t.TempDir(), "anytty.log")
	tracePath := filepath.Join(t.TempDir(), "perftrace.jsonl")
	t.Setenv("ANYTTY_PERF_TRACE", tracePath)
	t.Setenv("ANYTTY_PERF_TRACE_INTERVAL_MS", "10000")

	v3DialClient = func(path string) (*protocol.Client, error) {
		return nil, errors.New("dial disabled")
	}
	startV3Daemon = func(path string, logFile string) error {
		return errors.New("start disabled")
	}

	err := runV3RootEmptyRuntime(context.Background(), v3RootEmptyConfig{
		SocketPath: socketPath,
		LogFile:    logPath,
		TUIConfig:  tuistate.TUIConfigStore{},
	})
	if err == nil || clientruntime.CodeOf(err) != clientruntime.ErrorUnavailable || !clientruntime.WasAttempted(err) {
		t.Fatalf("expected daemon start failure after perftrace enable, got %v", err)
	}
	data, readErr := os.ReadFile(tracePath)
	if readErr != nil {
		t.Fatalf("read perftrace jsonl: %v", readErr)
	}
	if !strings.Contains(string(data), `"process":"tui-v3"`) {
		t.Fatalf("root empty runtime must write tui-v3 perftrace record, got %s", data)
	}
}

func TestV3RootEmptyRuntimeInstallsEndpointApplicationRouter(t *testing.T) {
	_, protocolClient, closeClient := newCoreV2ProtocolClientForCLITest(t)
	defer closeClient()
	oldConnect := connectV3EndpointApplication
	oldRouter := newV3EndpointApplicationRouter
	oldHost := newV3TerminalHost
	t.Cleanup(func() {
		connectV3EndpointApplication = oldConnect
		newV3EndpointApplicationRouter = oldRouter
		newV3TerminalHost = oldHost
	})

	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	installV3LocalApplicationTestClient(t, socketPath, protocolClient)
	routerCalled := false
	newV3EndpointApplicationRouter = func(endpointID tuistate.EndpointID, client *clientprotocol.ApplicationClient) (v3EndpointApplicationServices, error) {
		routerCalled = true
		if endpointID != tuistate.DefaultEndpointID || client == nil {
			t.Fatalf("root empty router input: endpoint=%q client=%v", endpointID, client)
		}
		return &clientruntimeadapter.EndpointApplicationRouter{}, nil
	}
	enterErr := errors.New("stop after composition")
	newV3TerminalHost = func() v3TerminalHost {
		return &enterFailingV3TerminalHost{FakeTerminalHost: app.NewFakeTerminalHost(1), err: enterErr}
	}

	err := runV3RootEmptyRuntime(context.Background(), v3RootEmptyConfig{
		SocketPath:         socketPath,
		LogFile:            filepath.Join(t.TempDir(), "anytty.log"),
		ConnectionRegistry: endpointdomain.DefaultRegistry(),
	})
	if !errors.Is(err, enterErr) {
		t.Fatalf("root empty runtime error = %v", err)
	}
	if !routerCalled {
		t.Fatal("root empty runtime did not install endpoint application router")
	}
}

type enterFailingV3TerminalHost struct {
	*app.FakeTerminalHost
	err error
}

func (host *enterFailingV3TerminalHost) Enter(context.Context) error { return host.err }
func (*enterFailingV3TerminalHost) Close() error                     { return nil }

func TestV3RootRuntimeReusesRunningTerminal(t *testing.T) {
	server, client, closeClient := newCoreV2ProtocolClientForCLITest(t)
	defer closeClient()
	if _, err := createCLIProtoTerminal(context.Background(), client, &apipb.TerminalCreateSpec{
		TerminalId: "term-existing",
		Name:       "existing",
		Command:    testShellSleepCommand(),
		Size:       &apipb.TerminalSize{Cols: 100, Rows: 30},
	}); err != nil {
		t.Fatalf("create existing terminal: %v", err)
	}
	oldConnect := connectV3EndpointApplication
	oldRunAttach := runV3Attach
	t.Cleanup(func() {
		connectV3EndpointApplication = oldConnect
		runV3Attach = oldRunAttach
	})

	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	logPath := filepath.Join(t.TempDir(), "anytty.log")
	installV3LocalApplicationTestClient(t, socketPath, client)
	var gotAttach v3AttachConfig
	runV3Attach = func(ctx context.Context, cfg v3AttachConfig) error {
		gotAttach = cfg
		return nil
	}

	if err := runV3RootRuntime(context.Background(), v3RootConfig{SocketPath: socketPath, LogFile: logPath}); err != nil {
		t.Fatalf("runV3RootRuntime returned error: %v", err)
	}
	if gotAttach.TerminalID != "term-existing" {
		t.Fatalf("expected v3 root to attach existing terminal, got %#v", gotAttach)
	}
	if _, err := server.GetTerminal(v3RootTerminalID); err == nil {
		t.Fatal("v3 root should not create default terminal when a running terminal exists")
	}
}

func TestV3RootRuntimeAttachesExitedRootWithoutAutoRestart(t *testing.T) {
	processes := newCoreV2ResizeRecordingProcessFactory()
	server, client, closeClient := newCoreV2ProtocolClientForCLITestWithOptions(t, corev2.WithProcessFactory(processes))
	defer closeClient()
	if _, err := createCLIProtoTerminal(context.Background(), client, &apipb.TerminalCreateSpec{
		TerminalId: v3RootTerminalID,
		Name:       "main",
		Command:    testShellSleepCommand(),
		Size:       &apipb.TerminalSize{Cols: 100, Rows: 30},
	}); err != nil {
		t.Fatalf("create exited root terminal: %v", err)
	}
	process := waitForCoreV2ResizeRecordingProcess(t, processes, v3RootTerminalID)
	process.exit(23)
	waitForCLITerminalState(t, server, v3RootTerminalID, corev2.TerminalStateExited)

	oldConnect := connectV3EndpointApplication
	oldRunAttach := runV3Attach
	t.Cleanup(func() {
		connectV3EndpointApplication = oldConnect
		runV3Attach = oldRunAttach
	})
	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	logPath := filepath.Join(t.TempDir(), "anytty.log")
	installV3LocalApplicationTestClient(t, socketPath, client)
	var gotAttach v3AttachConfig
	runV3Attach = func(ctx context.Context, cfg v3AttachConfig) error {
		gotAttach = cfg
		return nil
	}

	if err := runV3RootRuntime(context.Background(), v3RootConfig{SocketPath: socketPath, LogFile: logPath}); err != nil {
		t.Fatalf("runV3RootRuntime returned error: %v", err)
	}
	if gotAttach.TerminalID != v3RootTerminalID {
		t.Fatalf("expected exited root terminal to attach, got %#v", gotAttach)
	}
	info, err := server.GetTerminal(v3RootTerminalID)
	if err != nil {
		t.Fatalf("get root terminal: %v", err)
	}
	if info.State != corev2.TerminalStateExited || info.ExitCode == nil || info.ExitedAt.IsZero() {
		t.Fatalf("root entry must not auto restart or mutate exited lifecycle, got %#v", info)
	}
}

func TestV3RootEmptyInteractiveRuntimeStartsPickerAndEscLeavesUnconnectedPane(t *testing.T) {
	_, client, closeClient := newCoreV2ProtocolClientForCLITest(t)
	defer closeClient()
	host := app.NewFakeTerminalHost(8)
	host.SetSize(100, 30)
	runtime := newV3InteractiveRuntime("", 100, 30, wrapCLIProtocolClientForTest(t, client), host, nil)

	if err := runtime.Post(app.ShellOpenTerminalPickerMsg{}); err != nil {
		t.Fatalf("post terminal picker: %v", err)
	}
	root := waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return root.Shell.Overlay.Open && root.Shell.Overlay.Kind == tuistate.OverlayTerminalPicker
	}, "empty root opens terminal picker")
	if root.Session.TerminalID != "" || root.Session.Attached {
		t.Fatalf("empty root must not attach a terminal before user choice, session=%#v", root.Session)
	}

	if err := host.SendInput(tuiinput.InputEvent{Kind: tuiinput.EventKindKey, Key: tuiinput.KeyEsc}); err != nil {
		t.Fatalf("send esc: %v", err)
	}
	root = waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		pane, ok := root.Shell.EnsureDefaults().Pane(tuistate.PaneCommandTarget{PaneID: tuistate.DefaultPaneID})
		return !root.Shell.Overlay.Open && ok && pane.Kind == tuistate.PaneEmpty && pane.TerminalID == "" && pane.Title == "unconnected"
	}, "esc closes picker and leaves unconnected pane")
	if root.Session.TerminalID != "" || root.Session.Attached {
		t.Fatalf("esc from empty root picker must not attach terminal, session=%#v", root.Session)
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

func TestV3InteractiveRuntimeAttachesThroughProtocolClient(t *testing.T) {
	server, client, closeClient := newCoreV2ProtocolClientForCLITest(t)
	defer closeClient()
	if _, err := createCLIProtoTerminal(context.Background(), client, &apipb.TerminalCreateSpec{
		TerminalId: "term-1",
		Name:       "attach-demo",
		Command:    testShellSleepCommand(),
		Size:       &apipb.TerminalSize{Cols: 100, Rows: 30},
	}); err != nil {
		t.Fatalf("create terminal: %v", err)
	}
	_ = server
	host := app.NewFakeTerminalHost(8)
	runtime := newV3InteractiveRuntime("term-1", 100, 30, wrapCLIProtocolClientForTest(t, client), host, nil)

	if err := runtime.Post(app.LiveAttachMsg{Config: app.LiveConfig{
		TerminalID:   "term-1",
		Cols:         100,
		Rows:         30,
		Mode:         "collaborator",
		ResizePolicy: tuistate.TerminalResizeRoleOwner,
		SurfaceID:    "test-surface",
		ViewID:       tuistate.TerminalPaneViewID(tuistate.DefaultPaneID),
	}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return root.Session.Attached &&
			root.Session.TerminalID == "term-1" &&
			root.Session.Channel != 0 &&
			root.Session.Cols == 100 &&
			root.Session.Rows == 30
	}, "protocol attach result")
	if len(host.Frames()) == 0 {
		t.Fatal("expected attach drain to render at least one frame")
	}
}

func TestV3InteractiveRuntimeRestoresWorkbenchFromCoreV2Storage(t *testing.T) {
	_, client, closeClient := newCoreV2ProtocolClientForCLITest(t)
	defer closeClient()
	if _, err := createCLIProtoTerminal(context.Background(), client, &apipb.TerminalCreateSpec{
		TerminalId: "term-restored",
		Name:       "restored",
		Command:    testShellSleepCommand(),
		Size:       &apipb.TerminalSize{Cols: 80, Rows: 24},
	}); err != nil {
		t.Fatalf("create restored terminal: %v", err)
	}
	shell := tuistate.DefaultShell().
		SplitActivePane(tuistate.PaneState{ID: "pane-restored", Title: "restored", Kind: tuistate.PaneTerminalLive, TerminalID: "term-restored"}, tuistate.SplitDirectionVertical).
		FocusPane(tuistate.PaneCommandTarget{PaneID: "pane-restored"})
	views := tuistate.TerminalViewStore{}.
		BindPane(tuistate.NewPaneTerminalView("pane-restored", "term-restored", 3, 80, 24, tuistate.TerminalResizeRoleOwner, "surface-restored", tuistate.TerminalPaneViewID("pane-restored"), true))
	value, err := tuistate.EncodeWorkbenchStorageSnapshotValue(tuistate.SnapshotRootWorkbenchForStorage(tuistate.Root{Shell: shell, TerminalViews: views}))
	if err != nil {
		t.Fatalf("encode workbench snapshot: %v", err)
	}
	ref := tuistate.DefaultWorkbenchStorageRef(tuistate.DefaultWorkspaceID)
	storageApplication, err := newLocalApplicationSession(wrapCLIProtocolClientForTest(t, client))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := storageApplication.StoragePut(context.Background(), &apipb.StoragePutCommand{
		Key:   &apipb.StorageKey{AppId: ref.AppID, Scope: apipb.StorageScope_STORAGE_SCOPE_PUBLIC, OwnerId: ref.OwnerID, Key: ref.Key},
		Value: value,
	}); err != nil {
		t.Fatalf("put workbench snapshot: %v", err)
	}
	host := app.NewFakeTerminalHost(8)
	runtime := newV3InteractiveRuntime("term-restored", 100, 30, wrapCLIProtocolClientForTest(t, client), host, nil)

	root := waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		if root.Shell.ActivePaneID != "pane-restored" || root.WorkbenchSync.SaveVersion() < 1 {
			return false
		}
		binding, ok := root.TerminalViews.PaneBinding("pane-restored")
		return ok && binding.TerminalID == "term-restored"
	}, "workbench storage restore")
	if root.Shell.ActivePaneID != "pane-restored" || root.WorkbenchSync.SaveVersion() < 1 {
		t.Fatalf("runtime did not restore workbench from core-v2 storage %#v", root)
	}
	if binding, ok := root.TerminalViews.PaneBinding("pane-restored"); !ok || binding.TerminalID != "term-restored" {
		t.Fatalf("runtime did not restore terminal view binding binding=%#v ok=%v", binding, ok)
	}
}

func TestV3InteractiveRuntimeCorrectsProtocolResizeToContentRect(t *testing.T) {
	processes := newCoreV2ResizeRecordingProcessFactory()
	server, client, closeClient := newCoreV2ProtocolClientForCLITestWithOptions(t, corev2.WithProcessFactory(processes))
	defer closeClient()
	if _, err := createCLIProtoTerminal(context.Background(), client, &apipb.TerminalCreateSpec{
		TerminalId: "term-1",
		Name:       "resize-demo",
		Command:    testShellSleepCommand(),
		Size:       &apipb.TerminalSize{Cols: 100, Rows: 30},
	}); err != nil {
		t.Fatalf("create terminal: %v", err)
	}
	_ = server
	host := app.NewFakeTerminalHost(8)
	host.SetSize(100, 30)
	runtime := newV3InteractiveRuntime("term-1", 100, 30, wrapCLIProtocolClientForTest(t, client), host, nil)

	if err := runtime.Post(app.LiveAttachMsg{Config: app.LiveConfig{
		TerminalID:   "term-1",
		Cols:         100,
		Rows:         30,
		Mode:         "collaborator",
		ResizePolicy: tuistate.TerminalResizeRoleOwner,
		SurfaceID:    "test-surface",
		ViewID:       tuistate.TerminalPaneViewID(tuistate.DefaultPaneID),
	}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	current := waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return root.Session.Cols == 98 && root.Session.Rows == 26
	}, "initial attach content rect correction")
	if current.Session.SurfaceID != "test-surface" || current.Session.ViewID != tuistate.TerminalPaneViewID(tuistate.DefaultPaneID) {
		t.Fatalf("runtime must keep protocol resize owner metadata, session=%#v", current.Session)
	}
	process := waitForCoreV2ResizeRecordingProcess(t, processes, "term-1")
	waitForCoreV2ProcessResize(t, process, corev2.Size{Cols: 98, Rows: 26})
}

func TestV3InteractiveRuntimeLayoutResizeReachesCoreV2Process(t *testing.T) {
	processes := newCoreV2ResizeRecordingProcessFactory()
	server, client, closeClient := newCoreV2ProtocolClientForCLITestWithOptions(t, corev2.WithProcessFactory(processes))
	defer closeClient()
	if _, err := createCLIProtoTerminal(context.Background(), client, &apipb.TerminalCreateSpec{
		TerminalId: "term-1",
		Name:       "resize-flow",
		Command:    testShellSleepCommand(),
		Size:       &apipb.TerminalSize{Cols: 100, Rows: 30},
	}); err != nil {
		t.Fatalf("create terminal: %v", err)
	}
	_ = server
	process := waitForCoreV2ResizeRecordingProcess(t, processes, "term-1")
	host := app.NewFakeTerminalHost(32)
	host.SetSize(100, 30)
	runtime := newV3InteractiveRuntime("term-1", 100, 30, wrapCLIProtocolClientForTest(t, client), host, nil)

	resizeStart := process.resizeCount()
	if err := runtime.Post(app.LiveAttachMsg{Config: app.LiveConfig{
		TerminalID:   "term-1",
		Cols:         100,
		Rows:         30,
		Mode:         "collaborator",
		ResizePolicy: tuistate.TerminalResizeRoleOwner,
		SurfaceID:    "test-surface",
		ViewID:       tuistate.TerminalPaneViewID(tuistate.DefaultPaneID),
	}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return root.Session.Cols == 98 && root.Session.Rows == 26
	}, "initial attach content rect correction")
	waitForCoreV2ProcessResizeAfter(t, process, corev2.Size{Cols: 98, Rows: 26}, resizeStart)

	resizeStart = process.resizeCount()
	if err := host.SendResize(120, 40); err != nil {
		t.Fatalf("host resize: %v", err)
	}
	drainV3RuntimeForCLITest(t, runtime)
	waitForCoreV2ProcessResizeAfter(t, process, corev2.Size{Cols: 118, Rows: 36}, resizeStart)
	waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return root.Surface.Cols == 118 && root.Surface.Rows == 36
	}, "host resize content rect")

	resizeStart = process.resizeCount()
	if err := runtime.Post(app.ShellSetHeaderVisibleMsg{Visible: false}); err != nil {
		t.Fatalf("post header hide: %v", err)
	}
	drainV3RuntimeForCLITest(t, runtime)
	waitForCoreV2ProcessResizeAfter(t, process, corev2.Size{Cols: 118, Rows: 37}, resizeStart)
	waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return root.Surface.Cols == 118 && root.Surface.Rows == 37
	}, "header hide content rect")
	resizeStart = process.resizeCount()
	if err := runtime.Post(app.ShellSetFooterVisibleMsg{Visible: false}); err != nil {
		t.Fatalf("post footer hide: %v", err)
	}
	drainV3RuntimeForCLITest(t, runtime)
	waitForCoreV2ProcessResizeAfter(t, process, corev2.Size{Cols: 118, Rows: 38}, resizeStart)
	waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return root.Surface.Cols == 118 && root.Surface.Rows == 38
	}, "footer hide content rect")

	resizeStart = process.resizeCount()
	if err := runtime.Post(app.ShellPaneCommandMsg{Command: tuistate.PaneCommand{
		Action:         tuistate.PaneCommandSplit,
		SplitDirection: tuistate.SplitDirectionVertical,
		NewPane:        tuistate.PaneState{ID: "pane-2", Title: "right", Kind: tuistate.PaneTerminalLive},
		Source:         tuistate.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post split: %v", err)
	}
	drainV3RuntimeForCLITest(t, runtime)
	attachV3PaneOwnerForCLITest(t, runtime, "term-1", "pane-2", 58, 38)
	waitForCoreV2ProcessResizeAfter(t, process, corev2.Size{Cols: 58, Rows: 38}, resizeStart)
	waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return root.Surface.Cols == 58 && root.Surface.Rows == 38
	}, "split pane content rect")

	resizeStart = process.resizeCount()
	if err := runtime.Post(app.ShellPaneCommandMsg{Command: tuistate.PaneCommand{
		Action:          tuistate.PaneCommandResize,
		Target:          tuistate.PaneCommandTarget{PaneID: "pane-2"},
		ResizeDirection: tuistate.PaneResizeLeft,
		Delta:           6,
		Source:          tuistate.PaneCommandSourceKeyboard,
	}}); err != nil {
		t.Fatalf("post keyboard resize: %v", err)
	}
	drainV3RuntimeForCLITest(t, runtime)
	waitForCoreV2ProcessResizeAfter(t, process, corev2.Size{Cols: 64, Rows: 38}, resizeStart)
	waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return root.Surface.Cols == 64 && root.Surface.Rows == 38
	}, "keyboard resize content rect")

	resizeStart = process.resizeCount()
	if err := runtime.Post(app.ShellPaneCommandMsg{Command: tuistate.PaneCommand{
		Action: tuistate.PaneCommandZoom,
		Target: tuistate.PaneCommandTarget{PaneID: "pane-2"},
		Source: tuistate.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post zoom: %v", err)
	}
	drainV3RuntimeForCLITest(t, runtime)
	waitForCoreV2ProcessResizeAfter(t, process, corev2.Size{Cols: 120, Rows: 40}, resizeStart)
	waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return root.Surface.Cols == 120 && root.Surface.Rows == 40
	}, "zoom content rect")

	resizeStart = process.resizeCount()
	if err := runtime.Post(app.ShellPaneCommandMsg{Command: tuistate.PaneCommand{
		Action: tuistate.PaneCommandUnzoom,
		Source: tuistate.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post unzoom: %v", err)
	}
	drainV3RuntimeForCLITest(t, runtime)
	waitForCoreV2ProcessResizeAfter(t, process, corev2.Size{Cols: 64, Rows: 38}, resizeStart)
	waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return root.Surface.Cols == 64 && root.Surface.Rows == 38
	}, "unzoom content rect")

	resizeStart = process.resizeCount()
	if err := runtime.Post(app.ShellPaneCommandMsg{Command: tuistate.PaneCommand{
		Action: tuistate.PaneCommandClose,
		Target: tuistate.PaneCommandTarget{PaneID: "pane-2"},
		Source: tuistate.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post close: %v", err)
	}
	drainV3RuntimeForCLITest(t, runtime)
	waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return root.Session.Cols == 118 && root.Session.Rows == 38
	}, "close pane content rect restore")
	waitForCoreV2ProcessResizeAfter(t, process, corev2.Size{Cols: 118, Rows: 38}, resizeStart)
}

func TestV3InteractiveRuntimeMouseDividerResizeReachesCoreV2Process(t *testing.T) {
	processes := newCoreV2ResizeRecordingProcessFactory()
	server, client, closeClient := newCoreV2ProtocolClientForCLITestWithOptions(t, corev2.WithProcessFactory(processes))
	defer closeClient()
	if _, err := createCLIProtoTerminal(context.Background(), client, &apipb.TerminalCreateSpec{
		TerminalId: "term-1",
		Name:       "mouse-resize-flow",
		Command:    testShellSleepCommand(),
		Size:       &apipb.TerminalSize{Cols: 100, Rows: 30},
	}); err != nil {
		t.Fatalf("create terminal: %v", err)
	}
	_ = server
	process := waitForCoreV2ResizeRecordingProcess(t, processes, "term-1")
	host := app.NewFakeTerminalHost(32)
	host.SetSize(100, 30)
	runtime := newV3InteractiveRuntime("term-1", 100, 30, wrapCLIProtocolClientForTest(t, client), host, nil)

	resizeStart := process.resizeCount()
	if err := runtime.Post(app.LiveAttachMsg{Config: app.LiveConfig{
		TerminalID:   "term-1",
		Cols:         100,
		Rows:         30,
		Mode:         "collaborator",
		ResizePolicy: tuistate.TerminalResizeRoleOwner,
		SurfaceID:    "test-surface",
		ViewID:       tuistate.TerminalPaneViewID(tuistate.DefaultPaneID),
	}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return root.Session.Cols == 98 && root.Session.Rows == 26
	}, "initial attach content rect correction")
	waitForCoreV2ProcessResizeAfter(t, process, corev2.Size{Cols: 98, Rows: 26}, resizeStart)

	resizeStart = process.resizeCount()
	if err := runtime.Post(app.ShellPaneCommandMsg{Command: tuistate.PaneCommand{
		Action:         tuistate.PaneCommandSplit,
		SplitDirection: tuistate.SplitDirectionVertical,
		NewPane:        tuistate.PaneState{ID: "pane-2", Title: "right", Kind: tuistate.PaneTerminalLive},
		Source:         tuistate.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post split: %v", err)
	}
	drainV3RuntimeForCLITest(t, runtime)
	attachV3PaneOwnerForCLITest(t, runtime, "term-1", "pane-2", 48, 26)
	waitForCoreV2ProcessResizeAfter(t, process, corev2.Size{Cols: 48, Rows: 26}, resizeStart)
	waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return root.Surface.Cols == 48 && root.Surface.Rows == 26
	}, "split pane content rect")
	inputsBefore, _ := process.snapshot()

	divider := lastFramePaneResizeRegionForCLITest(t, host, tuistate.DefaultPaneID, tuistate.PaneResizeRight)
	start := mouseEventAtCLITest(divider.Rect)
	if err := host.SendInput(start); err != nil {
		t.Fatalf("send mouse drag start: %v", err)
	}
	drainV3RuntimeForCLITest(t, runtime)
	drag := start
	drag.Mouse = tuiinput.MouseLeftDrag
	drag.Col += 6
	resizeStart = process.resizeCount()
	if err := host.SendInput(drag); err != nil {
		t.Fatalf("send mouse drag move: %v", err)
	}
	drainV3RuntimeForCLITest(t, runtime)
	waitForCoreV2ProcessResizeAfter(t, process, corev2.Size{Cols: 42, Rows: 26}, resizeStart)

	inputsAfter, _ := process.snapshot()
	if len(inputsAfter) != len(inputsBefore) {
		t.Fatalf("mouse divider drag must not leak terminal input, before=%#v after=%#v", inputsBefore, inputsAfter)
	}
}

func TestV3InteractiveRuntimeCoreV2ResizeFailureSurfacesInSession(t *testing.T) {
	processes := newCoreV2ResizeRecordingProcessFactory()
	server, client, closeClient := newCoreV2ProtocolClientForCLITestWithOptions(t, corev2.WithProcessFactory(processes))
	defer closeClient()
	if _, err := createCLIProtoTerminal(context.Background(), client, &apipb.TerminalCreateSpec{
		TerminalId: "term-1",
		Name:       "resize-failure-flow",
		Command:    testShellSleepCommand(),
		Size:       &apipb.TerminalSize{Cols: 100, Rows: 30},
	}); err != nil {
		t.Fatalf("create terminal: %v", err)
	}
	_ = server
	host := app.NewFakeTerminalHost(32)
	host.SetSize(100, 30)
	runtime := newV3InteractiveRuntime("term-1", 100, 30, wrapCLIProtocolClientForTest(t, client), host, nil)

	if err := runtime.Post(app.LiveAttachMsg{Config: app.LiveConfig{
		TerminalID:   "term-1",
		Cols:         100,
		Rows:         30,
		Mode:         "collaborator",
		ResizePolicy: tuistate.TerminalResizeRoleOwner,
		SurfaceID:    "test-surface",
		ViewID:       tuistate.TerminalPaneViewID(tuistate.DefaultPaneID),
	}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return root.Session.Cols == 98 && root.Session.Rows == 26
	}, "initial attach content rect correction")
	process := waitForCoreV2ResizeRecordingProcess(t, processes, "term-1")
	seenResize := waitForCoreV2ProcessResize(t, process, corev2.Size{Cols: 98, Rows: 26})
	process.setResizeErr(errors.New("pty resize failed"))

	if err := host.SendResize(120, 40); err != nil {
		t.Fatalf("host resize: %v", err)
	}
	drainV3RuntimeForCLITest(t, runtime)
	waitForCoreV2ProcessResizeAfter(t, process, corev2.Size{Cols: 118, Rows: 36}, seenResize)
	current := waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		return strings.Contains(root.Session.LastError, "pty resize failed") && !root.Session.Attached
	}, "resize failure surfaced in session")
	if !strings.Contains(current.Session.LastError, "pty resize failed") || current.Session.Attached {
		t.Fatalf("core-v2 process resize failure must surface in live session, session=%#v", current.Session)
	}
}

func TestV3TerminalServiceCreateRejectsMissingCommandAgainstCoreV2(t *testing.T) {
	_, client, closeClient := newCoreV2ProtocolClientForCLITest(t)
	defer closeClient()
	adapter, err := protocoladapter.NewProtocolTerminalServiceAdapter(client, clientruntime.EndpointSessionStamp{EndpointID: endpointdomain.DefaultEndpointID, RouteID: endpointdomain.DefaultLocalRouteID, Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.Create(context.Background(), tuiservices.TerminalCreateRequest{
		TerminalID: "term-default-command",
		Title:      "default command",
		Cols:       80,
		Rows:       24,
	})
	if err == nil || !strings.Contains(err.Error(), "terminal create command is required") {
		t.Fatalf("adapter must reject empty command and rely on endpoint defaults before create, got %v", err)
	}
}

func TestV3VisualSnapshotCommandPrintsFixedVisualFrame(t *testing.T) {
	oldRunSmoke := runTUIv3SmokeDetailed
	t.Cleanup(func() {
		runTUIv3SmokeDetailed = oldRunSmoke
	})
	runTUIv3SmokeDetailed = tuiv3.SmokeRunDetailed

	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "visual-snapshot"})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	text := out.String()
	for _, want := range []string{" WS main", "▎ 1 main ×   2 logs ×  +", "visual review", "[↗]─[×]", "└───────────────────────────┘", "[Ctrl+P] PANE", "[PgUp] COPY", "ws:main float:1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("visual snapshot missing %q:\n%s", want, text)
		}
	}
}

func TestV3VisualSnapshotCommandCanWriteANSIScreen(t *testing.T) {
	oldRunSmoke := runTUIv3SmokeDetailed
	t.Cleanup(func() {
		runTUIv3SmokeDetailed = oldRunSmoke
	})
	runTUIv3SmokeDetailed = func(ctx context.Context) (tuiv3.SmokeResult, error) {
		return tuiv3.SmokeResult{Cases: []tuiv3.SmokeCase{{
			Name: "visual-audit-current",
			Frame: render.Frame{
				Lines:     []string{"plain"},
				ANSILines: []string{"\x1b[31mstyled\x1b[0m"},
			},
		}}}, nil
	}

	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "visual-snapshot", "--ansi"})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "\x1b[?2026h") || !strings.Contains(text, "\x1b[31mstyled\x1b[0m") || !strings.Contains(text, "\x1b[?2026l") {
		t.Fatalf("ANSI visual snapshot should use FrameSink repaint output, got %q", text)
	}
}

func TestV3VisualSnapshotCommandCanSelectSmokeCase(t *testing.T) {
	oldRunSmoke := runTUIv3SmokeDetailed
	t.Cleanup(func() {
		runTUIv3SmokeDetailed = oldRunSmoke
	})
	runTUIv3SmokeDetailed = func(ctx context.Context) (tuiv3.SmokeResult, error) {
		return tuiv3.SmokeResult{Cases: []tuiv3.SmokeCase{
			{Name: "visual-audit-current", Frame: render.Frame{Lines: []string{"default"}}},
			{Name: "workbench-tree-page", Frame: render.Frame{Lines: []string{"workbench frame"}}},
		}}, nil
	}

	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "visual-snapshot", "--case", "workbench-tree-page"})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "workbench frame") || strings.Contains(got, "default") {
		t.Fatalf("visual snapshot should print selected smoke case, got %q", got)
	}
}

func TestV3E2ESmokeCommandRunsLocalCoreAndTUIPath(t *testing.T) {
	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "e2e-smoke"})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("v3 e2e-smoke returned error: %v", err)
	}
	text := out.String()
	for _, want := range []string{"anytty v3 e2e smoke ok", "copy_cols=98", "zoom_checked=true"} {
		if !strings.Contains(text, want) {
			t.Fatalf("v3 e2e-smoke output missing %q:\n%s", want, text)
		}
	}
}

func TestV3TmuxSmokeHarnessCapturesArtifact(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not installed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := runV3TmuxSmoke(ctx)
	if err != nil {
		t.Fatalf("tmux smoke: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(result.ArtifactDir)
	})
	if result.Session == "" || result.ArtifactDir == "" || result.ANSIPath == "" || result.PlainPath == "" {
		t.Fatalf("tmux smoke should return session and artifact paths, got %#v", result)
	}
	if !strings.Contains(result.Captured, "anytty tmux harness ready") || !strings.Contains(result.Captured, "anytty tmux input:hello-from-tmux") {
		t.Fatalf("tmux capture missing markers:\n%s", result.Captured)
	}
	for _, path := range []string{result.ANSIPath, result.PlainPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read artifact %s: %v", path, err)
		}
		if len(data) == 0 {
			t.Fatalf("artifact %s should not be empty", path)
		}
	}
	if err := runTmuxCommand(context.Background(), "has-session", "-t", result.Session); err == nil {
		t.Fatalf("tmux session %s should be cleaned up", result.Session)
	}
}

func TestV3TmuxSmokeCommandReportsArtifactPaths(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not installed: %v", err)
	}
	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "tmux-smoke"})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("v3 tmux-smoke returned error: %v", err)
	}
	text := out.String()
	if artifactDir := v3TmuxSmokeOutputValue(text, "artifact_dir"); artifactDir != "" {
		t.Cleanup(func() {
			_ = os.RemoveAll(artifactDir)
		})
	}
	if !strings.Contains(text, "anytty v3 tmux smoke ok") ||
		!strings.Contains(text, "session=anytty-v3-smoke-") ||
		!strings.Contains(text, "input=hello-from-tmux") ||
		!strings.Contains(text, "artifact_dir=") ||
		!strings.Contains(text, "ansi=") ||
		!strings.Contains(text, "plain=") {
		t.Fatalf("unexpected tmux smoke output:\n%s", text)
	}
}

func TestV3TmuxTerminalSmokeCreatesAttachesAndSendsInput(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not installed: %v", err)
	}
	anyttyBin := buildAnyTTYBinaryForTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, err := runV3TmuxTerminalSmoke(ctx, anyttyBin)
	if err != nil {
		t.Fatalf("tmux terminal smoke: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(result.ArtifactDir)
	})
	if result.TerminalID == "" || result.Session == "" || result.SocketPath == "" || result.DaemonLog == "" || result.TimelinePath == "" {
		t.Fatalf("tmux terminal smoke should return ids and artifact paths, got %#v", result)
	}
	if !strings.Contains(result.Captured, "anytty-pty-ready") || !strings.Contains(result.Captured, "anytty-pty-echo:tmux-live-input") {
		t.Fatalf("tmux terminal capture missing live surface markers:\n%s", result.Captured)
	}
	for _, path := range []string{result.ANSIPath, result.PlainPath, result.DaemonLog, result.TimelinePath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read artifact %s: %v", path, err)
		}
		if len(data) == 0 {
			t.Fatalf("artifact %s should not be empty", path)
		}
	}
	if err := runTmuxCommand(context.Background(), "has-session", "-t", result.Session); err == nil {
		t.Fatalf("tmux session %s should be cleaned up", result.Session)
	}
}

func TestV3TmuxTerminalSmokeCommandReportsArtifacts(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not installed: %v", err)
	}
	anyttyBin := buildAnyTTYBinaryForTest(t)
	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "tmux-terminal-smoke", "--anytty-bin", anyttyBin})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("v3 tmux-terminal-smoke returned error: %v", err)
	}
	text := out.String()
	if artifactDir := v3TmuxSmokeOutputValue(text, "artifact_dir"); artifactDir != "" {
		t.Cleanup(func() {
			_ = os.RemoveAll(artifactDir)
		})
	}
	if !strings.Contains(text, "anytty v3 tmux terminal smoke ok") ||
		!strings.Contains(text, "terminal=tmux-e2e") ||
		!strings.Contains(text, "session=anytty-v3-terminal-") ||
		!strings.Contains(text, "input=tmux-live-input") ||
		!strings.Contains(text, "artifact_dir=") ||
		!strings.Contains(text, "daemon_log=") ||
		!strings.Contains(text, "socket=") ||
		!strings.Contains(text, "timeline=") {
		t.Fatalf("unexpected tmux terminal smoke output:\n%s", text)
	}
}

func TestV3TmuxResizeSmokePropagatesHostResizeToPTY(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not installed: %v", err)
	}
	anyttyBin := buildAnyTTYBinaryForTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, err := runV3TmuxResizeSmoke(ctx, anyttyBin)
	if err != nil {
		t.Fatalf("tmux resize smoke: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(result.ArtifactDir)
	})
	if result.TerminalID == "" || result.BeforeSize == "" || result.AfterSize == "" || result.WindowSize != "120x40" {
		t.Fatalf("tmux resize smoke should return sizes and ids, got %#v", result)
	}
	if result.BeforeSize == result.AfterSize {
		t.Fatalf("resize should change PTY size, before=%s after=%s", result.BeforeSize, result.AfterSize)
	}
	if result.AfterSize != "118x36" {
		t.Fatalf("120x40 host viewport should resize PTY to tuiv2-style active content rect 118x36, got %s", result.AfterSize)
	}
	if !strings.Contains(result.Captured, "anytty-pty-size:size-before:") || !strings.Contains(result.Captured, "anytty-pty-size:size-after:"+result.AfterSize) {
		t.Fatalf("tmux resize capture missing PTY size markers:\n%s", result.Captured)
	}
	for _, path := range []string{result.ANSIPath, result.PlainPath, result.DaemonLog, result.TimelinePath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read artifact %s: %v", path, err)
		}
		if len(data) == 0 {
			t.Fatalf("artifact %s should not be empty", path)
		}
	}
	if err := runTmuxCommand(context.Background(), "has-session", "-t", result.Session); err == nil {
		t.Fatalf("tmux session %s should be cleaned up", result.Session)
	}
}

func TestV3TmuxResizeSmokeCommandReportsArtifacts(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not installed: %v", err)
	}
	anyttyBin := buildAnyTTYBinaryForTest(t)
	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "tmux-resize-smoke", "--anytty-bin", anyttyBin})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("v3 tmux-resize-smoke returned error: %v", err)
	}
	text := out.String()
	if artifactDir := v3TmuxSmokeOutputValue(text, "artifact_dir"); artifactDir != "" {
		t.Cleanup(func() {
			_ = os.RemoveAll(artifactDir)
		})
	}
	if !strings.Contains(text, "anytty v3 tmux resize smoke ok") ||
		!strings.Contains(text, "terminal=tmux-resize") ||
		!strings.Contains(text, "session=anytty-v3-resize-") ||
		!strings.Contains(text, "window=120x40") ||
		!strings.Contains(text, "after=118x36") ||
		!strings.Contains(text, "artifact_dir=") ||
		!strings.Contains(text, "timeline=") {
		t.Fatalf("unexpected tmux resize smoke output:\n%s", text)
	}
}

func TestV3TmuxANSISmokeCapturesLiveSurfaceSGR(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not installed: %v", err)
	}
	anyttyBin := buildAnyTTYBinaryForTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, err := runV3TmuxANSISmoke(ctx, anyttyBin)
	if err != nil {
		t.Fatalf("tmux ansi smoke: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(result.ArtifactDir)
	})
	for _, marker := range []string{"ANSI16_RED", "ANSI16_BLUE_BOLD", "ANSI256_ORANGE", "ANSI256_BG", "TRUECOLOR_MINT", "CR_REPLACED", "PRIMARY_AFTER_ALT"} {
		if !strings.Contains(result.Captured, marker) {
			t.Fatalf("plain capture missing marker %q:\n%s", marker, result.Captured)
		}
	}
	for _, sgr := range []string{"\x1b[31m", "\x1b[1m", "\x1b[34m", "\x1b[38;5;202m", "\x1b[48;5;24m", "\x1b[38;2;12;200;155m"} {
		if !strings.Contains(result.ANSICaptured, sgr) {
			t.Fatalf("ANSI capture missing SGR %q:\n%q", sgr, result.ANSICaptured)
		}
	}
	if strings.Contains(result.Captured, "\x1b]") {
		t.Fatalf("OSC theme responses must not leak into plain terminal capture:\n%q", result.Captured)
	}
	for _, path := range []string{result.ANSIPath, result.PlainPath, result.DaemonLog, result.TimelinePath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read artifact %s: %v", path, err)
		}
		if len(data) == 0 {
			t.Fatalf("artifact %s should not be empty", path)
		}
	}
	if err := runTmuxCommand(context.Background(), "has-session", "-t", result.Session); err == nil {
		t.Fatalf("tmux session %s should be cleaned up", result.Session)
	}
}

func TestV3TmuxANSISmokeCommandReportsArtifacts(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not installed: %v", err)
	}
	anyttyBin := buildAnyTTYBinaryForTest(t)
	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "tmux-ansi-smoke", "--anytty-bin", anyttyBin})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("v3 tmux-ansi-smoke returned error: %v", err)
	}
	text := out.String()
	if artifactDir := v3TmuxSmokeOutputValue(text, "artifact_dir"); artifactDir != "" {
		t.Cleanup(func() {
			_ = os.RemoveAll(artifactDir)
		})
	}
	if !strings.Contains(text, "anytty v3 tmux ansi smoke ok") ||
		!strings.Contains(text, "terminal=tmux-ansi") ||
		!strings.Contains(text, "session=anytty-v3-ansi-") ||
		!strings.Contains(text, "artifact_dir=") ||
		!strings.Contains(text, "ansi=") ||
		!strings.Contains(text, "plain=") ||
		!strings.Contains(text, "timeline=") {
		t.Fatalf("unexpected tmux ansi smoke output:\n%s", text)
	}
}

func TestV3TmuxEmojiDotsSmokeReproducesOwnerFollowerGeometry(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not installed: %v", err)
	}
	anyttyBin := buildAnyTTYBinaryForTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, err := runV3TmuxEmojiDotsSmoke(ctx, anyttyBin)
	if err != nil {
		t.Fatalf("tmux emoji dots smoke: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(result.ArtifactDir)
	})
	if result.TerminalID == "" || result.BeforeSize == "" || result.AfterSize == "" || !result.DotsVisible {
		t.Fatalf("tmux emoji dots smoke should return terminal ids, PTY sizes and visible dots, got %#v", result)
	}
	if result.BeforeSize == result.AfterSize {
		t.Fatalf("owner/follower resize should change PTY size, before=%s after=%s", result.BeforeSize, result.AfterSize)
	}
	for _, marker := range []string{"owner", "follow", "anytty-pty-size:size-before:", "anytty-pty-size:size-after:", "anytty-pty-echo:", "····"} {
		if !strings.Contains(result.Captured, marker) {
			t.Fatalf("tmux emoji dots capture missing marker %q:\n%s", marker, result.Captured)
		}
	}
	for _, path := range []string{result.ANSIPath, result.PlainPath, result.DaemonLog, result.TimelinePath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read artifact %s: %v", path, err)
		}
		if len(data) == 0 {
			t.Fatalf("artifact %s should not be empty", path)
		}
	}
	if err := runTmuxCommand(context.Background(), "has-session", "-t", result.Session); err == nil {
		t.Fatalf("tmux session %s should be cleaned up", result.Session)
	}
}

func TestV3TmuxEmojiDotsSmokeCommandReportsArtifacts(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not installed: %v", err)
	}
	anyttyBin := buildAnyTTYBinaryForTest(t)
	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "tmux-emoji-dots-smoke", "--anytty-bin", anyttyBin})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("v3 tmux-emoji-dots-smoke returned error: %v", err)
	}
	text := out.String()
	if artifactDir := v3TmuxSmokeOutputValue(text, "artifact_dir"); artifactDir != "" {
		t.Cleanup(func() {
			_ = os.RemoveAll(artifactDir)
		})
	}
	if !strings.Contains(text, "anytty v3 tmux emoji dots smoke ok") ||
		!strings.Contains(text, "terminal=tmux-emoji-dots") ||
		!strings.Contains(text, "session=anytty-v3-emoji-dots-") ||
		!strings.Contains(text, "dots=true") ||
		!strings.Contains(text, "artifact_dir=") ||
		!strings.Contains(text, "timeline=") {
		t.Fatalf("unexpected tmux emoji dots smoke output:\n%s", text)
	}
}

func TestV3TmuxVisualCompareCapturesTargetAndDiffArtifacts(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not installed: %v", err)
	}
	anyttyBin := buildAnyTTYBinaryForTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result, err := runV3TmuxVisualCompare(ctx, anyttyBin)
	if err != nil {
		t.Fatalf("tmux visual compare: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(result.ArtifactDir)
	})
	if result.Session == "" || result.ArtifactDir == "" || result.CurrentPlainPath == "" || result.CurrentANSIPath == "" || result.TargetPath == "" || result.DiffPath == "" || result.StylePath == "" || result.StyleDiffPath == "" || result.CurrentStyleMapPath == "" || result.TargetStyleMapPath == "" || result.StyleMapDiffPath == "" || result.SummaryPath == "" {
		t.Fatalf("tmux visual compare should return artifact paths, got %#v", result)
	}
	for _, path := range []string{result.CurrentPlainPath, result.CurrentANSIPath, result.TargetPath, result.DiffPath, result.StylePath, result.StyleDiffPath, result.CurrentStyleMapPath, result.TargetStyleMapPath, result.StyleMapDiffPath, result.SummaryPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read artifact %s: %v", path, err)
		}
		if len(data) == 0 {
			t.Fatalf("artifact %s should not be empty", path)
		}
	}
	current, err := os.ReadFile(result.CurrentPlainPath)
	if err != nil {
		t.Fatalf("read current plain: %v", err)
	}
	target, err := os.ReadFile(result.TargetPath)
	if err != nil {
		t.Fatalf("read target plain: %v", err)
	}
	diff, err := os.ReadFile(result.DiffPath)
	if err != nil {
		t.Fatalf("read diff: %v", err)
	}
	style, err := os.ReadFile(result.StylePath)
	if err != nil {
		t.Fatalf("read style report: %v", err)
	}
	styleDiff, err := os.ReadFile(result.StyleDiffPath)
	if err != nil {
		t.Fatalf("read style diff: %v", err)
	}
	currentStyleMap, err := os.ReadFile(result.CurrentStyleMapPath)
	if err != nil {
		t.Fatalf("read current style map: %v", err)
	}
	targetStyleMap, err := os.ReadFile(result.TargetStyleMapPath)
	if err != nil {
		t.Fatalf("read target style map: %v", err)
	}
	styleMapDiff, err := os.ReadFile(result.StyleMapDiffPath)
	if err != nil {
		t.Fatalf("read style map diff: %v", err)
	}
	if !strings.Contains(string(current), "[↗]─[×]") ||
		!strings.Contains(string(target), "[Ctrl+G] GLOBAL • [Ctrl+O] FLOAT • [Ctrl+T] TAB") ||
		!strings.Contains(string(target), "[PgUp] COPY") ||
		!strings.Contains(string(target), "float:1") ||
		!strings.Contains(string(diff), "tmux visual diff") {
		t.Fatalf("visual compare artifacts missing expected markers current=%q target=%q diff=%q", current, target, diff)
	}
	currentNormalized := normalizeVisualText(string(current), 140, 40)
	targetNormalized := normalizeVisualText(string(target), 140, 40)
	if result.Mismatches != 0 || !strings.Contains(string(diff), "no row mismatches") || currentNormalized != targetNormalized {
		t.Fatalf("visual compare should match target exactly, mismatches=%d diff=\n%s", result.Mismatches, diff)
	}
	if result.StyleMismatches != 0 || !strings.Contains(string(styleDiff), "no style mismatches") {
		t.Fatalf("visual compare style contract should match, style_mismatches=%d diff=\n%s", result.StyleMismatches, styleDiff)
	}
	if result.StyleMapMismatches != 0 || !strings.Contains(string(styleMapDiff), "no style map mismatches") {
		t.Fatalf("visual compare style map should match, stylemap_mismatches=%d diff=\n%s", result.StyleMapMismatches, styleMapDiff)
	}
	for _, marker := range []string{"legend: S=status A=accent M=muted W=warning G=success P=plain .=transparent ?=unknown", "01 SSSSSS"} {
		if !strings.Contains(string(currentStyleMap), marker) || !strings.Contains(string(targetStyleMap), marker) {
			t.Fatalf("style maps missing marker %q current=\n%s\ntarget=\n%s", marker, currentStyleMap, targetStyleMap)
		}
	}
	for _, marker := range []string{"pane-action-accent", "active-pane-right-border-accent", "right-pane-border-muted", "footer-no-bg", "footer-float-accent"} {
		if !strings.Contains(string(style), marker+" row=") {
			t.Fatalf("style report missing marker %q: %s", marker, style)
		}
	}
	if err := runTmuxCommand(context.Background(), "has-session", "-t", result.Session); err == nil {
		t.Fatalf("tmux session %s should be cleaned up", result.Session)
	}
}

func TestV3TmuxVisualCompareCommandReportsArtifacts(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not installed: %v", err)
	}
	anyttyBin := buildAnyTTYBinaryForTest(t)
	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "tmux-visual-compare", "--anytty-bin", anyttyBin})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("v3 tmux-visual-compare returned error: %v", err)
	}
	text := out.String()
	if artifactDir := v3TmuxSmokeOutputValue(text, "artifact_dir"); artifactDir != "" {
		t.Cleanup(func() {
			_ = os.RemoveAll(artifactDir)
		})
	}
	if !strings.Contains(text, "anytty v3 tmux visual compare ok") ||
		!strings.Contains(text, "session=anytty-v3-visual-") ||
		!strings.Contains(text, "mismatches=") ||
		!strings.Contains(text, "style_mismatches=") ||
		!strings.Contains(text, "stylemap_mismatches=") ||
		!strings.Contains(text, "current_plain=") ||
		!strings.Contains(text, "current_ansi=") ||
		!strings.Contains(text, "target=") ||
		!strings.Contains(text, "diff=") ||
		!strings.Contains(text, "style=") ||
		!strings.Contains(text, "style_diff=") ||
		!strings.Contains(text, "current_stylemap=") ||
		!strings.Contains(text, "target_stylemap=") ||
		!strings.Contains(text, "stylemap_diff=") {
		t.Fatalf("unexpected tmux visual compare output:\n%s", text)
	}
}

func TestV3TmuxStabilitySmokeRunsShortRound(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not installed: %v", err)
	}
	anyttyBin := buildAnyTTYBinaryForTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	result, err := runV3TmuxStabilitySmoke(ctx, anyttyBin, 1)
	if err != nil {
		t.Fatalf("tmux stability smoke: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(result.ArtifactDir)
		for _, artifact := range result.Artifacts {
			_ = os.RemoveAll(artifact)
		}
	})
	if result.Rounds != 1 || len(result.Artifacts) != 3 || result.ArtifactDir == "" || result.TimelinePath == "" {
		t.Fatalf("unexpected stability result %#v", result)
	}
	timeline, err := os.ReadFile(result.TimelinePath)
	if err != nil {
		t.Fatalf("read stability timeline: %v", err)
	}
	if !strings.Contains(string(timeline), "terminal smoke ok") ||
		!strings.Contains(string(timeline), "resize smoke ok") ||
		!strings.Contains(string(timeline), "ansi smoke ok") {
		t.Fatalf("stability timeline missing child smoke results:\n%s", timeline)
	}
	for _, artifact := range result.Artifacts {
		if _, err := os.Stat(filepath.Join(artifact, "capture.txt")); err != nil {
			t.Fatalf("child artifact %s missing capture.txt: %v", artifact, err)
		}
	}
}

func TestV3TmuxStabilitySmokeCommandReportsArtifacts(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not installed: %v", err)
	}
	anyttyBin := buildAnyTTYBinaryForTest(t)
	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "tmux-stability-smoke", "--anytty-bin", anyttyBin, "--rounds", "1"})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("v3 tmux-stability-smoke returned error: %v", err)
	}
	text := out.String()
	if artifactDir := v3TmuxSmokeOutputValue(text, "artifact_dir"); artifactDir != "" {
		t.Cleanup(func() {
			_ = os.RemoveAll(artifactDir)
		})
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "artifact: ") {
			artifact := strings.TrimSpace(strings.TrimPrefix(line, "artifact: "))
			t.Cleanup(func() {
				_ = os.RemoveAll(artifact)
			})
		}
	}
	if !strings.Contains(text, "anytty v3 tmux stability smoke ok") ||
		!strings.Contains(text, "rounds=1") ||
		!strings.Contains(text, "artifacts=3") ||
		!strings.Contains(text, "artifact_dir=") ||
		!strings.Contains(text, "timeline=") ||
		!strings.Contains(text, "artifact: ") {
		t.Fatalf("unexpected tmux stability smoke output:\n%s", text)
	}
}

func v3TmuxSmokeOutputValue(output string, key string) string {
	prefix := key + "="
	for _, field := range strings.Fields(output) {
		if strings.HasPrefix(field, prefix) {
			return strings.TrimPrefix(field, prefix)
		}
	}
	return ""
}

func buildAnyTTYBinaryForTest(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "anytty-test-bin")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	cmd := exec.Command("go", "build", "-tags", "anytty_dev_commands", "-o", path, ".")
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build anytty test binary: %v\n%s", err, output)
	}
	return path
}

type fakeCoreV2Server struct {
	newServerCalls int
	listenCalls    int
	shutdownCalls  int
}

func (s *fakeCoreV2Server) ListenAndServe(context.Context) error {
	s.listenCalls++
	return nil
}

func (s *fakeCoreV2Server) Shutdown(context.Context) error {
	s.shutdownCalls++
	return nil
}

func (s *fakeCoreV2Server) ServeScopedTransport(context.Context, transport.Transport, corev2.TransportScope) error {
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
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, ed25519.SeedSize))
	identity, err := remoteauth.NewIdentity("device-cli-test", privateKey)
	if err != nil {
		panic(err)
	}
	defaults := []corev2.ServerOption{
		corev2.WithApplicationExecutorFactory(apilayer.CoreApplicationExecutorFactory),
		corev2.WithClientAccessService(v3ClientAccessService{identity: identity}),
	}
	return corev2.NewServer(append(defaults, opts...)...)
}

func newCoreV2ProtocolClientForCLITestWithOptions(t *testing.T, opts ...corev2.ServerOption) (*corev2.Server, *protocol.Client, func()) {
	t.Helper()
	socketPath := filepath.Join(t.TempDir(), "anytty-v2.sock")
	serverOpts := append([]corev2.ServerOption{corev2.WithSocketPath(socketPath)}, opts...)
	server := newCoreV2TestServer(serverOpts...)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- server.ListenAndServe(ctx)
	}()
	if err := waitForSocket(socketPath, 2*time.Second, func() error {
		client, err := dialV3Client(socketPath)
		if err != nil {
			return err
		}
		return client.Close()
	}); err != nil {
		cancel()
		_ = server.Shutdown(context.Background())
		t.Fatalf("core-v2 daemon did not become ready: %v", err)
	}
	client, err := dialV3Client(socketPath)
	if err != nil {
		cancel()
		_ = server.Shutdown(context.Background())
		t.Fatalf("dial core-v2 daemon: %v", err)
	}
	closeFn := func() {
		_ = client.Close()
		cancel()
		_ = server.Shutdown(context.Background())
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("core-v2 server did not stop in time")
		}
	}
	return server, client, closeFn
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

func drainV3RuntimeForCLITest(t *testing.T, runtime *app.AppRuntime) {
	t.Helper()
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain runtime: %v", err)
	}
}

func attachV3PaneOwnerForCLITest(t *testing.T, runtime *app.AppRuntime, terminalID string, paneID string, cols int, rows int) {
	t.Helper()
	root := runtime.State()
	binding, ok := root.TerminalViews.PaneBinding(paneID)
	if !ok || binding.TerminalID != terminalID {
		t.Fatalf("missing pane binding for owner transfer pane=%s terminal=%s root=%#v", paneID, terminalID, root)
	}
	if err := runtime.Post(app.ShellShortcutActionMsg{
		Invocation: actiondomain.Invocation{ID: "panel.take_owner", SourceActionID: "panel.take_owner"},
		Surface:    &app.ShortcutSurfaceContext{ExplicitTarget: true, PaneID: paneID, Row: -1},
	}); err != nil {
		t.Fatalf("post pane owner action: %v", err)
	}
	waitForV3RuntimeState(t, runtime, func(root tuistate.Root) bool {
		binding, ok := root.TerminalViews.PaneBinding(paneID)
		return ok &&
			binding.TerminalID == terminalID &&
			binding.ViewID == tuistate.TerminalPaneViewID(paneID) &&
			binding.ResizeRole == tuistate.TerminalResizeRoleOwner &&
			binding.CanResize &&
			binding.DesiredCols == cols &&
			binding.DesiredRows == rows
	}, "pane owner attach")
}

func waitForV3RuntimeState(t *testing.T, runtime *app.AppRuntime, predicate func(tuistate.Root) bool, label string) tuistate.Root {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		drainV3RuntimeForCLITest(t, runtime)
		current := runtime.State()
		if predicate(current) {
			return current
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for runtime state: %s current=%#v", label, current)
		}
		time.Sleep(10 * time.Millisecond)
	}
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

func mouseEventAtCLITest(rect render.Rect) tuiinput.InputEvent {
	return tuiinput.InputEvent{Kind: tuiinput.EventKindMouse, Mouse: tuiinput.MouseLeft, Row: rect.Y + 1, Col: rect.X + 1}
}

func lastFramePaneResizeRegionForCLITest(t *testing.T, host *app.FakeTerminalHost, paneID string, direction tuistate.PaneResizeDirection) render.HitRegion {
	t.Helper()
	frames := host.Frames()
	if len(frames) == 0 {
		t.Fatal("expected runtime frame")
	}
	for _, region := range frames[len(frames)-1].HitRegions {
		if region.Kind == render.HitRegionPaneResize && region.ActionID == "pane.resize" && region.PaneID == paneID && region.Direction == string(direction) {
			return region
		}
	}
	t.Fatalf("missing pane resize region pane=%s direction=%s in %#v", paneID, direction, frames[len(frames)-1].HitRegions)
	return render.HitRegion{}
}

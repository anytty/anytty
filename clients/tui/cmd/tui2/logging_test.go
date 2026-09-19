package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSetupLoggingRedirectsStandardLoggerToFile pins the M1 contract: setting
// up the host logger points the Go standard logger (shared client layer:
// client/runtime, client/adapter/*, shared/netpath, shared/connecttrace) at
// the log file, so a shared-layer diagnostic can never reach stderr.
func TestSetupLoggingRedirectsStandardLoggerToFile(t *testing.T) {
	previousOutput := log.Writer()
	previousFlags := log.Flags()
	t.Cleanup(func() {
		log.SetOutput(previousOutput)
		log.SetFlags(previousFlags)
		closeLogging()
	})

	path := filepath.Join(t.TempDir(), "state", "anytty", "tui2.log")
	t.Setenv(LogFileEnvVar, path)
	resolved, err := setupLogging("")
	if err != nil {
		t.Fatalf("setupLogging: %v", err)
	}
	if resolved != path {
		t.Fatalf("resolved log path = %q, want %q", resolved, path)
	}
	log.Printf("anytty connect trace_id=trace-1 cloud connect generation=1")

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(payload), "trace_id=trace-1") {
		t.Fatalf("log file = %q; want the shared-layer diagnostic", payload)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat log file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("log file mode = %v, want 0600", info.Mode().Perm())
	}
}

// TestResolveLogPathPriority pins flag > environment > XDG state default.
func TestResolveLogPathPriority(t *testing.T) {
	t.Setenv(LogFileEnvVar, filepath.FromSlash("/tmp/env-anytty-tui2.log"))
	if got := resolveLogPath(filepath.FromSlash("/tmp/flag-anytty-tui2.log")); got != filepath.FromSlash("/tmp/flag-anytty-tui2.log") {
		t.Fatalf("resolveLogPath(flag) = %q, want the flag path", got)
	}
	if got := resolveLogPath(""); got != filepath.FromSlash("/tmp/env-anytty-tui2.log") {
		t.Fatalf("resolveLogPath(env) = %q, want the env path", got)
	}
	t.Setenv(LogFileEnvVar, "")
	t.Setenv("XDG_STATE_HOME", filepath.FromSlash("/tmp/anytty-state"))
	if got, want := resolveLogPath(""), filepath.Join(filepath.FromSlash("/tmp/anytty-state"), "anytty", logFileName); got != want {
		t.Fatalf("resolveLogPath(default) = %q, want %q", got, want)
	}
}

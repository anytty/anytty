//go:build darwin || linux || windows

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/shared/securefs"
)

func TestPoolServiceLifecycleUsesExactRuntimeRecord(t *testing.T) {
	binary := buildAnyTTYBinaryForTest(t)
	runtimeDir := t.TempDir()
	socketPath := filepath.Join(runtimeDir, "anytty.sock")
	logPath := filepath.Join(runtimeDir, "anytty.log")
	t.Cleanup(func() {
		stop := exec.Command(binary, "--socket", socketPath, "--log-file", logPath, "pool", "stop")
		_, _ = stop.CombinedOutput()
	})

	stopped := executeAnyTTYBinary(t, binary, "--socket", socketPath, "--log-file", logPath, "pool", "status", "--json")
	if stopped.State != "stopped" || stopped.PID != 0 {
		t.Fatalf("initial pool status = %#v", stopped)
	}
	started := executeAnyTTYBinary(t, binary, "--socket", socketPath, "--log-file", logPath, "pool", "start", "--json")
	if started.State != "running" || started.PID <= 0 {
		t.Fatalf("started pool status = %#v", started)
	}
	recordInfo, err := os.Stat(poolRecordPath(socketPath))
	if err != nil {
		t.Fatal(err)
	}
	if !securefs.IsPrivateFile(poolRecordPath(socketPath), recordInfo) {
		t.Fatal("pool record is not protected for the current user")
	}

	restart := exec.Command(binary, "--socket", socketPath, "--log-file", logPath, "pool", "restart")
	if output, err := restart.CombinedOutput(); err != nil {
		t.Fatalf("pool restart: %v\n%s", err, output)
	}
	restarted := executeAnyTTYBinary(t, binary, "--socket", socketPath, "--log-file", logPath, "pool", "status", "--json")
	if restarted.State != "running" || restarted.PID <= 0 || restarted.PID == started.PID {
		t.Fatalf("restarted pool status = %#v, old pid=%d", restarted, started.PID)
	}

	logCommand := exec.Command(binary, "--log-file", logPath, "pool", "logs", "--lines", "5")
	if output, err := logCommand.CombinedOutput(); err != nil || !bytes.Contains(output, []byte("starting terminal pool")) {
		t.Fatalf("pool logs = %v\n%s", err, output)
	}
	stopped = executeAnyTTYBinary(t, binary, "--socket", socketPath, "--log-file", logPath, "pool", "stop", "--json")
	if stopped.State != "stopped" {
		t.Fatalf("stopped pool status = %#v", stopped)
	}
	if _, err := os.Stat(poolRecordPath(socketPath)); !os.IsNotExist(err) {
		t.Fatalf("pool record remains after stop: %v", err)
	}
}

func TestDeprecatedDaemonAliasStillServesPoolStatus(t *testing.T) {
	binary := buildAnyTTYBinaryForTest(t)
	runtimeDir := t.TempDir()
	socketPath := filepath.Join(runtimeDir, "anytty.sock")
	logPath := filepath.Join(runtimeDir, "anytty.log")
	command := exec.Command(binary, "--socket", socketPath, "--log-file", logPath, "daemon", "status", "--json")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("deprecated daemon status: %v\nstderr:\n%s", err, stderr.String())
	}
	if !strings.Contains(stderr.String(), "anytty daemon is deprecated; use `anytty pool` instead") {
		t.Fatalf("missing deprecation notice: %q", stderr.String())
	}
	var view poolStatusView
	if err := json.Unmarshal(stdout.Bytes(), &view); err != nil {
		t.Fatalf("decode deprecated daemon status %q: %v", stdout.String(), err)
	}
	if view.State != "stopped" {
		t.Fatalf("deprecated daemon status = %#v", view)
	}
}

func TestPoolStatusReadsLegacyDaemonRecordAfterUpgrade(t *testing.T) {
	runtimeDir := t.TempDir()
	socketPath := filepath.Join(runtimeDir, "anytty.sock")
	identity, err := poolProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	legacy := poolRuntimeRecord{
		SchemaVersion: poolRecordSchemaVersion, PID: os.Getpid(), ProcessID: identity,
		InstanceToken: strings.Repeat("a", 64), Executable: "/legacy/anytty",
		SocketPath: socketPath, LogPath: filepath.Join(runtimeDir, "anytty.log"), StartedAt: time.Now().UTC(),
	}
	payload, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPoolRecordPath(socketPath), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	view, record, err := poolStatus(socketPath, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if record.PID != os.Getpid() || view.PID != os.Getpid() || view.State != "starting" {
		t.Fatalf("legacy record not honored: view=%#v record=%#v", view, record)
	}
	if _, path, err := readPoolRuntimeRecordForSocket(socketPath); err != nil || path != legacyPoolRecordPath(socketPath) {
		t.Fatalf("legacy record path = %q err=%v", path, err)
	}
}

func TestPoolRuntimeRecordRejectsUnsafePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		path := filepath.Join(os.Getenv("SystemRoot"), "System32", "kernel32.dll")
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if securefs.IsPrivateFile(path, info) {
			t.Fatal("system-owned file must not pass current-user pool record ownership")
		}
		return
	}
	path := poolRecordPath(filepath.Join(t.TempDir(), "anytty.sock"))
	if err := os.WriteFile(path, []byte(`{"schema_version":1}`), 0o666); err != nil {
		t.Fatal(err)
	}
	if _, err := readPoolRuntimeRecord(path); cliExitCode(err) != 5 {
		t.Fatalf("unsafe pool record error = %v, exit=%d", err, cliExitCode(err))
	}
}

func TestPoolRuntimeRecordKeepsDirectListenerForPairing(t *testing.T) {
	t.Setenv("ANYTTY_DIRECT_LISTEN", "0.0.0.0:45123")
	runtimeDir := t.TempDir()
	socketPath := filepath.Join(runtimeDir, "anytty.sock")
	release, err := acquirePoolRuntimeRecord(socketPath, filepath.Join(runtimeDir, "anytty.log"), filepath.Join(runtimeDir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	record, err := readPoolRuntimeRecord(poolRecordPath(socketPath))
	if err != nil {
		t.Fatal(err)
	}
	if record.DirectListen != "0.0.0.0:45123" {
		t.Fatalf("record Direct listener = %q", record.DirectListen)
	}
	if got := runningPoolDirectListen(socketPath); got != record.DirectListen {
		t.Fatalf("pairing Direct listener = %q, want %q", got, record.DirectListen)
	}
}

func TestPrivatePoolLogCreatesProtectedParent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "nested", "anytty.log")
	file, err := openPrivatePoolLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !securefs.IsPrivateFile(path, info) {
		t.Fatalf("pool log permissions are not private: %v", info.Mode())
	}
}

func executeAnyTTYBinary(t *testing.T, binary string, args ...string) poolStatusView {
	t.Helper()
	command := exec.Command(binary, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		var poolLog []byte
		for index, arg := range args {
			if arg == "--log-file" && index+1 < len(args) {
				poolLog, _ = os.ReadFile(args[index+1])
				break
			}
		}
		t.Fatalf("anytty %s: %v\noutput:\n%s\nlog:\n%s", strings.Join(args, " "), err, output, poolLog)
	}
	var view poolStatusView
	if err := json.Unmarshal(output, &view); err != nil {
		t.Fatalf("decode pool status %q: %v", output, err)
	}
	return view
}

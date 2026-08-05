//go:build darwin || linux || windows

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/anytty/anytty/shared/securefs"
)

func TestDaemonServiceLifecycleUsesExactRuntimeRecord(t *testing.T) {
	binary := buildAnyTTYBinaryForTest(t)
	runtimeDir := t.TempDir()
	socketPath := filepath.Join(runtimeDir, "anytty.sock")
	logPath := filepath.Join(runtimeDir, "anytty.log")
	t.Cleanup(func() {
		stop := exec.Command(binary, "--socket", socketPath, "--log-file", logPath, "daemon", "stop")
		_, _ = stop.CombinedOutput()
	})

	stopped := executeAnyTTYBinary(t, binary, "--socket", socketPath, "--log-file", logPath, "daemon", "status", "--json")
	if stopped.State != "stopped" || stopped.PID != 0 {
		t.Fatalf("initial daemon status = %#v", stopped)
	}
	started := executeAnyTTYBinary(t, binary, "--socket", socketPath, "--log-file", logPath, "daemon", "start", "--json")
	if started.State != "running" || started.PID <= 0 {
		t.Fatalf("started daemon status = %#v", started)
	}
	recordInfo, err := os.Stat(daemonRecordPath(socketPath))
	if err != nil {
		t.Fatal(err)
	}
	if !securefs.IsPrivateFile(daemonRecordPath(socketPath), recordInfo) {
		t.Fatal("daemon record is not protected for the current user")
	}

	restart := exec.Command(binary, "--socket", socketPath, "--log-file", logPath, "daemon", "restart")
	if output, err := restart.CombinedOutput(); err != nil {
		t.Fatalf("daemon restart: %v\n%s", err, output)
	}
	restarted := executeAnyTTYBinary(t, binary, "--socket", socketPath, "--log-file", logPath, "daemon", "status", "--json")
	if restarted.State != "running" || restarted.PID <= 0 || restarted.PID == started.PID {
		t.Fatalf("restarted daemon status = %#v, old pid=%d", restarted, started.PID)
	}

	logCommand := exec.Command(binary, "--log-file", logPath, "daemon", "logs", "--lines", "5")
	if output, err := logCommand.CombinedOutput(); err != nil || !bytes.Contains(output, []byte("core-v2 daemon")) {
		t.Fatalf("daemon logs = %v\n%s", err, output)
	}
	stopped = executeAnyTTYBinary(t, binary, "--socket", socketPath, "--log-file", logPath, "daemon", "stop", "--json")
	if stopped.State != "stopped" {
		t.Fatalf("stopped daemon status = %#v", stopped)
	}
	if _, err := os.Stat(daemonRecordPath(socketPath)); !os.IsNotExist(err) {
		t.Fatalf("daemon record remains after stop: %v", err)
	}
}

func TestDaemonRuntimeRecordRejectsUnsafePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		path := filepath.Join(os.Getenv("SystemRoot"), "System32", "kernel32.dll")
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if securefs.IsPrivateFile(path, info) {
			t.Fatal("system-owned file must not pass current-user daemon record ownership")
		}
		return
	}
	path := daemonRecordPath(filepath.Join(t.TempDir(), "anytty.sock"))
	if err := os.WriteFile(path, []byte(`{"schema_version":1}`), 0o666); err != nil {
		t.Fatal(err)
	}
	if _, err := readDaemonRuntimeRecord(path); cliExitCode(err) != 5 {
		t.Fatalf("unsafe daemon record error = %v, exit=%d", err, cliExitCode(err))
	}
}

func TestPrivateDaemonLogCreatesProtectedParent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "nested", "anytty.log")
	file, err := openPrivateDaemonLog(path)
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
		t.Fatalf("daemon log permissions are not private: %v", info.Mode())
	}
}

func executeAnyTTYBinary(t *testing.T, binary string, args ...string) daemonStatusView {
	t.Helper()
	command := exec.Command(binary, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		var daemonLog []byte
		for index, arg := range args {
			if arg == "--log-file" && index+1 < len(args) {
				daemonLog, _ = os.ReadFile(args[index+1])
				break
			}
		}
		t.Fatalf("anytty %s: %v\noutput:\n%s\nlog:\n%s", strings.Join(args, " "), err, output, daemonLog)
	}
	var view daemonStatusView
	if err := json.Unmarshal(output, &view); err != nil {
		t.Fatalf("decode daemon status %q: %v", output, err)
	}
	return view
}

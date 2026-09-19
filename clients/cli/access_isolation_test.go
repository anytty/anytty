//go:build !windows

package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestAccessLifecycleIsolationAndIndependentUpgrade 验证同一二进制下的角色隔离：
// access 崩溃/重启/独立升级都不影响 pool 与终端；pool 重启可保留 access。
func TestAccessLifecycleIsolationAndIndependentUpgrade(t *testing.T) {
	binary := buildAnyTTYBinaryForTest(t)
	root := t.TempDir()
	socketPath := filepath.Join(root, "anytty.sock")
	logPath := filepath.Join(root, "anytty.log")
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_RUNTIME_DIR", root)
	t.Setenv("ANYTTY_DIRECT_LISTEN", "")
	t.Setenv("ANYTTY_ACCESS_LOG_FILE", filepath.Join(root, "access.log"))

	run := func(args ...string) (string, error) {
		full := append([]string{"--socket", socketPath, "--log-file", logPath}, args...)
		command := exec.Command(binary, full...)
		output, err := command.CombinedOutput()
		return string(output), err
	}
	mustRun := func(args ...string) string {
		t.Helper()
		output, err := run(args...)
		if err != nil {
			t.Fatalf("anytty %s: %v\n%s", strings.Join(args, " "), err, output)
		}
		return output
	}
	mustListTerminal := func(marker string) {
		t.Helper()
		if output := mustRun("terminal", "list", "--json"); !strings.Contains(output, marker) {
			t.Fatalf("terminal %q missing after lifecycle step:\n%s", marker, output)
		}
	}
	listAfterAccess := func(marker string) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			output, err := run("terminal", "list", "--json")
			if err == nil && strings.Contains(output, marker) {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatalf("terminal %q did not come back after access restart", marker)
	}
	t.Cleanup(func() { _, _ = run("pool", "stop") })

	mustRun("pool", "start", "--json")
	started := decodePoolStatus(t, mustRun("pool", "status", "--json"))
	if started.State != "running" || started.PID <= 0 || started.AccessPID <= 0 {
		t.Fatalf("pool+access stack did not start: %#v", started)
	}
	mustRun("new", "--name", "iso-term", "--", "/bin/sh")
	mustListTerminal("iso-term")

	// access 崩溃：pool 与终端必须存活。
	accessProcess, err := os.FindProcess(started.AccessPID)
	if err != nil {
		t.Fatal(err)
	}
	if err := accessProcess.Signal(syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	waitForAccessNotRunning(t, run)
	afterCrash := decodePoolStatus(t, mustRun("pool", "status", "--json"))
	if afterCrash.State != "running" || afterCrash.PID != started.PID {
		t.Fatalf("pool did not survive access crash: %#v (was %#v)", afterCrash, started)
	}

	// access 独立恢复：pool PID 不变，终端记录仍在。
	recovered := decodeAccessStatus(t, mustRun("access", "start", "--json"))
	if recovered.State != "running" || recovered.PID <= 0 || recovered.PID == started.AccessPID {
		t.Fatalf("access did not recover with a new pid: %#v", recovered)
	}
	listAfterAccess("iso-term")
	if current := decodePoolStatus(t, mustRun("pool", "status", "--json")); current.PID != started.PID {
		t.Fatalf("pool pid changed during access recovery: %#v", current)
	}

	// access 独立重启（升级入口）：pool 与终端不受影响。
	restarted := decodeAccessStatus(t, mustRun("access", "restart", "--json"))
	if restarted.State != "running" || restarted.PID == recovered.PID {
		t.Fatalf("access restart did not replace the process: %#v", restarted)
	}
	if current := decodePoolStatus(t, mustRun("pool", "status", "--json")); current.PID != started.PID {
		t.Fatalf("pool pid changed during access restart: %#v", current)
	}
	listAfterAccess("iso-term")

	// pool 重启并保留 access：access PID 不变（终端随 pool 重建属于预期）。
	mustRun("pool", "restart", "--keep-access")
	afterPoolRestart := decodePoolStatus(t, mustRun("pool", "status", "--json"))
	if afterPoolRestart.State != "running" || afterPoolRestart.PID == started.PID {
		t.Fatalf("pool restart did not replace the pool: %#v", afterPoolRestart)
	}
	if afterPoolRestart.AccessPID != restarted.PID || afterPoolRestart.AccessState != "running" {
		t.Fatalf("pool restart --keep-access replaced access: %#v (was %#v)", afterPoolRestart, restarted)
	}

	// access 独立停止：pool 继续运行。
	mustRun("access", "stop")
	stoppedAccess := decodeAccessStatus(t, mustRun("access", "status", "--json"))
	if stoppedAccess.State == "running" || stoppedAccess.State == "starting" {
		t.Fatalf("access still running after stop: %#v", stoppedAccess)
	}
	if current := decodePoolStatus(t, mustRun("pool", "status", "--json")); current.State != "running" || current.PID != afterPoolRestart.PID {
		t.Fatalf("pool did not survive access stop: %#v", current)
	}
}

func decodePoolStatus(t *testing.T, output string) poolStatusView {
	t.Helper()
	var view poolStatusView
	if err := json.Unmarshal([]byte(output), &view); err != nil {
		t.Fatalf("decode pool status %q: %v", output, err)
	}
	return view
}

func decodeAccessStatus(t *testing.T, output string) accessStatusView {
	t.Helper()
	var view accessStatusView
	if err := json.Unmarshal([]byte(output), &view); err != nil {
		t.Fatalf("decode access status %q: %v", output, err)
	}
	return view
}

func waitForAccessNotRunning(t *testing.T, run func(...string) (string, error)) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		output, err := run("access", "status", "--json")
		if err == nil {
			view := decodeAccessStatus(t, output)
			if view.State != "running" && view.State != "starting" {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("access stayed running after kill")
}

package harness

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/pty"
)

// waitSessionText polls the captured screen until want appears.
func waitSessionText(t *testing.T, session *Session, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, line := range session.CaptureText() {
			if strings.Contains(line, want) {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("screen never contained %q: %#v", want, session.CaptureText())
}

func waitSessionClipboard(t *testing.T, session *Session, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if got, ok := session.Clipboard(); ok && got == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	got, ok := session.Clipboard()
	t.Fatalf("clipboard = %q, %v, want %q", got, ok, want)
}

func TestSessionPTYRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("pty integration test skipped in short mode")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found")
	}
	script := `printf 'READY\n'; read line; printf 'GOT:%s\n' "$line"; printf '\033]52;c;aGVsbG8=\007'; sleep 10`
	session, err := Spawn(Config{Argv: []string{sh, "-c", script}, Cols: 40, Rows: 8})
	if errors.Is(err, pty.ErrUnsupported) {
		t.Skip("pty unsupported on this platform")
	}
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer func() { _ = session.Kill() }()

	waitSessionText(t, session, "READY", 5*time.Second)
	if err := session.SendBytes([]byte("hi\r")); err != nil {
		t.Fatalf("SendBytes: %v", err)
	}
	waitSessionText(t, session, "GOT:hi", 5*time.Second)
	waitSessionClipboard(t, session, "hello", 5*time.Second)
	if err := session.SendKeys("Escape"); err != nil {
		t.Fatalf("SendKeys: %v", err)
	}
	if err := session.Resize(60, 12); err != nil {
		t.Fatalf("Resize: %v", err)
	}
}

// TestSessionExitCode pins the graceful-exit readback: a command that exits
// without a wrapper leaves its status readable through Wait/ExitCode.
func TestSessionExitCode(t *testing.T) {
	if testing.Short() {
		t.Skip("pty integration test skipped in short mode")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found")
	}
	session, err := Spawn(Config{Argv: []string{sh, "-c", "exit 7"}, Cols: 20, Rows: 5})
	if errors.Is(err, pty.ErrUnsupported) {
		t.Skip("pty unsupported on this platform")
	}
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer func() { _ = session.Kill() }()
	code, ok := session.Wait(5 * time.Second)
	if !ok || code != 7 {
		t.Fatalf("Wait = %d, %v, want 7, true", code, ok)
	}
}

func TestManagerReuseAndKill(t *testing.T) {
	if testing.Short() {
		t.Skip("pty integration test skipped in short mode")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found")
	}
	manager := NewManager()
	if err := manager.Spawn("s1", Config{Argv: []string{sh, "-c", "sleep 10"}, Cols: 20, Rows: 5}); err != nil {
		if errors.Is(err, pty.ErrUnsupported) {
			t.Skip("pty unsupported on this platform")
		}
		t.Fatalf("Spawn: %v", err)
	}
	if err := manager.Spawn("s1", Config{Argv: []string{sh, "-c", "sleep 10"}, Cols: 20, Rows: 5}); err != nil {
		t.Fatalf("respawn same name: %v", err)
	}
	if err := manager.Kill("s1"); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	if _, err := manager.Session("s1"); !errors.Is(err, ErrNoSession) {
		t.Fatalf("Session after Kill = %v, want ErrNoSession", err)
	}
	manager.KillAll()
}

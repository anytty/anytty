//go:build linux || darwin

package pty

import (
	"errors"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLifecyclePreconditions(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found")
	}
	p := New(Config{Argv: []string{sh, "-c", "exit 0"}})
	if _, err := p.Read(make([]byte, 1)); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("Read before Start = %v, want ErrNotStarted", err)
	}
	if _, err := p.Write([]byte("x")); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("Write before Start = %v, want ErrNotStarted", err)
	}
	if err := p.Resize(10, 5); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("Resize before Start = %v, want ErrNotStarted", err)
	}
	if _, _, err := p.Size(); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("Size before Start = %v, want ErrNotStarted", err)
	}
	if err := p.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := p.Start(); !errors.Is(err, ErrStarted) {
		t.Fatalf("second Start = %v, want ErrStarted", err)
	}
	if err := p.Resize(0, 5); !errors.Is(err, ErrInvalidSize) {
		t.Fatalf("Resize(0,5) = %v, want ErrInvalidSize", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if _, _, err := p.Size(); !errors.Is(err, ErrClosed) {
		t.Fatalf("Size after Close = %v, want ErrClosed", err)
	}
	if _, err := p.Write([]byte("x")); !errors.Is(err, ErrClosed) {
		t.Fatalf("Write after Close = %v, want ErrClosed", err)
	}
}

func TestStartEmptyArgv(t *testing.T) {
	p := New(Config{})
	if err := p.Start(); !errors.Is(err, ErrNoCommand) {
		t.Fatalf("Start with empty argv = %v, want ErrNoCommand", err)
	}
}

// TestShellEchoAndResize drives a real /bin/sh over a PTY: it starts a
// reader, checks the startup output, verifies that typed input is echoed,
// resizes the window and finally asks the shell for `stty size` to prove the
// kernel saw the new dimensions.
func TestShellEchoAndResize(t *testing.T) {
	if testing.Short() {
		t.Skip("pty integration test skipped in short mode")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found")
	}
	p := New(Config{
		Argv: []string{sh, "-c", "printf ready; cat -v; stty size"},
		Cols: 80,
		Rows: 24,
	})
	if err := p.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	chunks, stop := streamPTY(p)
	defer func() {
		_ = p.Close()
		stop()
	}()

	readUntil(t, chunks, "ready", 5*time.Second)

	if _, err := p.Write([]byte("hi\n")); err != nil {
		t.Fatalf("Write hi: %v", err)
	}
	readUntilCount(t, chunks, "hi", 2, 5*time.Second)

	if err := p.Resize(100, 30); err != nil {
		t.Fatalf("Resize: %v", err)
	}
	cols, rows, err := p.Size()
	if err != nil {
		t.Fatalf("Size: %v", err)
	}
	if cols != 100 || rows != 30 {
		t.Fatalf("Size = %dx%d, want 100x30", cols, rows)
	}
	time.Sleep(100 * time.Millisecond)

	if _, err := p.Write([]byte{0x04}); err != nil {
		t.Fatalf("Write EOT: %v", err)
	}
	out := readUntil(t, chunks, "30 100", 5*time.Second)
	if !strings.Contains(out, "30 100") {
		t.Fatalf("stty size output missing: %q", out)
	}
}

type ptyRead struct {
	data string
	err  error
}

// streamPTY pumps Read into a channel. The reader exits when the PTY is
// closed; stop waits for it so tests never leak the goroutine.
func streamPTY(p PTY) (<-chan ptyRead, func()) {
	chunks := make(chan ptyRead, 32)
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(chunks)
		buf := make([]byte, 4096)
		for {
			n, err := p.Read(buf)
			if n > 0 {
				select {
				case chunks <- ptyRead{data: string(buf[:n])}:
				case <-done:
					return
				}
			}
			if err != nil {
				select {
				case chunks <- ptyRead{err: err}:
				default:
				}
				return
			}
		}
	}()
	return chunks, func() {
		close(done)
		wg.Wait()
	}
}

func readUntil(t *testing.T, chunks <-chan ptyRead, want string, timeout time.Duration) string {
	t.Helper()
	return readUntilCount(t, chunks, want, 1, timeout)
}

func readUntilCount(t *testing.T, chunks <-chan ptyRead, want string, count int, timeout time.Duration) string {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	var b strings.Builder
	for {
		if strings.Count(b.String(), want) >= count {
			return b.String()
		}
		select {
		case chunk, ok := <-chunks:
			if !ok {
				t.Fatalf("pty stream closed after %q, want %d×%q", b.String(), count, want)
			}
			b.WriteString(chunk.data)
		case <-timer.C:
			t.Fatalf("timeout after %q, want %d×%q", b.String(), count, want)
		}
	}
}

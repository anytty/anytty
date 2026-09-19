//go:build linux || darwin

package pty

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	creackpty "github.com/creack/pty"
)

// unixPTY implements PTY with creack/pty on Linux and macOS.
type unixPTY struct {
	cfg Config

	mu       sync.Mutex
	started  bool
	closed   bool
	file     *os.File
	cmd      *exec.Cmd
	exitCode int
	done     chan struct{}
}

func newPlatform(cfg Config) PTY { return &unixPTY{cfg: cfg} }

// Start allocates the PTY and spawns cfg.Argv with the PTY as its
// controlling terminal, using the configured initial window size.
func (p *unixPTY) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started {
		return ErrStarted
	}
	if len(p.cfg.Argv) == 0 {
		return ErrNoCommand
	}
	cmd := exec.Command(p.cfg.Argv[0], p.cfg.Argv[1:]...)
	cmd.Dir = p.cfg.Cwd
	if len(p.cfg.Env) > 0 {
		cmd.Env = p.cfg.Env
	} else {
		cmd.Env = os.Environ()
	}
	file, err := creackpty.StartWithSize(cmd, &creackpty.Winsize{
		Rows: uint16(p.cfg.Rows),
		Cols: uint16(p.cfg.Cols),
	})
	if err != nil {
		return err
	}
	p.file = file
	p.cmd = cmd
	p.started = true
	p.exitCode = -1
	p.done = make(chan struct{})
	go p.reap()
	return nil
}

// reap waits for the child once and records its exit status (a signal maps to
// 128+signal, matching shell conventions).
func (p *unixPTY) reap() {
	_ = p.cmd.Wait()
	code := -1
	if ps := p.cmd.ProcessState; ps != nil {
		if status, ok := ps.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			code = 128 + int(status.Signal())
		} else if exit := ps.ExitCode(); exit >= 0 {
			code = exit
		}
	}
	p.mu.Lock()
	p.exitCode = code
	p.mu.Unlock()
	close(p.done)
}

// ExitCode returns the recorded child exit status, or -1 while unknown.
func (p *unixPTY) ExitCode() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitCode
}

// PID returns the child process id, or 0 before Start and after Close. The
// platform start path puts the child in its own session, so the pid doubles as
// its process-group id; callers can signal the whole tree with
// syscall.Kill(-pid, sig).
func (p *unixPTY) PID() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.cmd == nil || p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

// Read reads child output from the PTY master.
func (p *unixPTY) Read(buf []byte) (int, error) {
	file, err := p.openFile()
	if err != nil {
		return 0, err
	}
	return file.Read(buf)
}

// Write writes bytes to the child, as if typed on the terminal.
func (p *unixPTY) Write(buf []byte) (int, error) {
	file, err := p.openFile()
	if err != nil {
		return 0, err
	}
	return file.Write(buf)
}

// Resize sets the terminal window size and lets the kernel signal SIGWINCH
// to the child's foreground process group.
func (p *unixPTY) Resize(cols, rows int) error {
	if cols <= 0 || rows <= 0 {
		return ErrInvalidSize
	}
	file, err := p.openFile()
	if err != nil {
		return err
	}
	if cols > 0xffff {
		cols = 0xffff
	}
	if rows > 0xffff {
		rows = 0xffff
	}
	return creackpty.Setsize(file, &creackpty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
}

// Size reports the current terminal window size.
func (p *unixPTY) Size() (int, int, error) {
	file, err := p.openFile()
	if err != nil {
		return 0, 0, err
	}
	size, err := creackpty.GetsizeFull(file)
	if err != nil {
		return 0, 0, err
	}
	return int(size.Cols), int(size.Rows), nil
}

// Close kills the child, reaps it and releases the PTY. It is idempotent.
func (p *unixPTY) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	file := p.file
	cmd := p.cmd
	p.file = nil
	p.mu.Unlock()

	var closeErr error
	if file != nil {
		closeErr = file.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		// Let the reaper record the signal exit status instead of racing it
		// with a second wait on the process.
		if done := p.done; done != nil {
			select {
			case <-done:
			case <-time.After(2 * time.Second):
			}
		}
	}
	if closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
		return closeErr
	}
	return nil
}

func (p *unixPTY) openFile() (*os.File, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, ErrClosed
	}
	if !p.started || p.file == nil {
		return nil, ErrNotStarted
	}
	return p.file, nil
}

// Package pty owns the host side of one terminal process: allocating a
// pseudo-terminal, starting the child, resizing its window and closing it.
// It is deliberately protocol-free; tui2/runtime and the terminal component
// call it through the PTY interface so tests can substitute a fake.
package pty

import (
	"errors"
)

// Config describes the process behind a PTY. Argv is required; Cwd, Env,
// Cols and Rows fall back to the host defaults (row/col defaults are 24/80).
//
// Endpoint and ID carry the protocol identity of the attach call
// ("terminal:<endpoint>:<id>"). They are host metadata: the platform PTY
// implementations ignore them, while a host can dispatch on Endpoint to build
// a daemon-backed PTY instead (ENDPOINTS.zh-CN.md §2).
type Config struct {
	Argv     []string
	Cwd      string
	Env      []string
	Cols     int
	Rows     int
	Endpoint string
	ID       string
}

// DefaultCols and DefaultRows are the fallback window size.
const (
	DefaultCols = 80
	DefaultRows = 24
)

// Errors reported by every PTY implementation.
var (
	// ErrNoCommand means Start was called with an empty argv.
	ErrNoCommand = errors.New("pty: empty argv")
	// ErrStarted means Start was called twice.
	ErrStarted = errors.New("pty: already started")
	// ErrNotStarted means the operation needs a started PTY.
	ErrNotStarted = errors.New("pty: not started")
	// ErrClosed means the PTY has already been closed.
	ErrClosed = errors.New("pty: closed")
	// ErrInvalidSize means cols/rows were not positive.
	ErrInvalidSize = errors.New("pty: invalid size")
	// ErrUnsupported means the platform has no PTY implementation.
	ErrUnsupported = errors.New("pty: unsupported platform")
)

// PTY is one pseudo-terminal attached to one child process.
//
// Lifecycle: New returns an unstarted value, Start spawns the child, Read and
// Write carry bytes in both directions, Resize updates the window (SIGWINCH
// included on Unix), Size reports it and Close kills the child and releases
// the PTY. Close is idempotent; Read/Write/Resize return ErrClosed after it.
type PTY interface {
	Start() error
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	Resize(cols, rows int) error
	Size() (cols, rows int, err error)
	// ExitCode returns the child's exit status once it has exited (signals
	// report 128+signal). It is -1 while the status is still unknown.
	ExitCode() int
	Close() error
}

// New returns the platform PTY implementation for cfg. It never fails here;
// Start reports missing argv or platform errors.
func New(cfg Config) PTY {
	cfg = normalize(cfg)
	return newPlatform(cfg)
}

func normalize(cfg Config) Config {
	if cfg.Cols <= 0 {
		cfg.Cols = DefaultCols
	}
	if cfg.Rows <= 0 {
		cfg.Rows = DefaultRows
	}
	if cfg.Cols > 0xffff {
		cfg.Cols = 0xffff
	}
	if cfg.Rows > 0xffff {
		cfg.Rows = 0xffff
	}
	return cfg
}

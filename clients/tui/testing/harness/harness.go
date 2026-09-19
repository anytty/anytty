// Package harness drives a terminal program through a real PTY for black-box
// tests. It spawns a command at a fixed window size, parses its ANSI output
// into a screen, forwards typed input, resizes the window, and captures the
// screen as plain text, as tmux-style SGR runs, and as OSC 52 clipboard
// writes. The tui2-harness command exposes the same primitives over a line
// protocol so the smoke/acceptance shell scripts can run without tmux.
package harness

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/clients/tui/render/ansi"
)

// Errors reported by the harness.
var (
	// ErrSessionExists means Spawn used a name that is already live.
	ErrSessionExists = errors.New("harness: session already exists")
	// ErrNoSession means the named session is unknown.
	ErrNoSession = errors.New("harness: session not found")
)

// Config describes one PTY session.
type Config struct {
	// Argv is the command to run. It is required.
	Argv []string
	// Dir is the working directory (empty inherits the harness cwd).
	Dir string
	// Env overrides the environment (nil inherits the harness environment).
	Env []string
	// Cols and Rows are the initial window size (zero uses 80x24).
	Cols int
	Rows int
}

// Session is one spawned command plus its parsed screen state. It is safe for
// concurrent use; the reader goroutine updates the screen while callers
// snapshot it.
type Session struct {
	proc   pty.PTY
	parser *ansi.Parser

	mu      sync.Mutex
	rawTail []byte
	clip    string
	hasClip bool
	readErr error
	exited  chan struct{}
	killed  bool
}

// Spawn starts cfg under a fresh PTY and begins parsing its output.
func Spawn(cfg Config) (*Session, error) {
	if len(cfg.Argv) == 0 {
		return nil, pty.ErrNoCommand
	}
	cols, rows := cfg.Cols, cfg.Rows
	if cols <= 0 {
		cols = pty.DefaultCols
	}
	if rows <= 0 {
		rows = pty.DefaultRows
	}
	proc := pty.New(pty.Config{
		Argv: cfg.Argv,
		Cwd:  cfg.Dir,
		Env:  cfg.Env,
		Cols: cols,
		Rows: rows,
	})
	if err := proc.Start(); err != nil {
		return nil, err
	}
	s := &Session{
		proc:   proc,
		parser: ansi.New(cols, rows),
		exited: make(chan struct{}),
	}
	go s.readLoop()
	return s, nil
}

// readLoop pumps PTY output into the parser and the OSC 52 scanner until the
// PTY reports an error (EIO after the child exits, or a closed master).
func (s *Session) readLoop() {
	defer close(s.exited)
	buf := make([]byte, 64*1024)
	for {
		n, err := s.proc.Read(buf)
		if n > 0 {
			s.mu.Lock()
			s.parser.Write(buf[:n])
			s.scanOSC52Locked(buf[:n])
			s.mu.Unlock()
		}
		if err != nil {
			s.mu.Lock()
			s.readErr = err
			s.mu.Unlock()
			return
		}
	}
}

// SendBytes writes raw bytes to the child, as if typed on the terminal.
func (s *Session) SendBytes(data []byte) error {
	_, err := s.proc.Write(data)
	return err
}

// SendKeys translates tmux-style key names (Enter, C-p, Down, ...) and sends
// the resulting bytes.
func (s *Session) SendKeys(keys ...string) error {
	var data []byte
	for _, key := range keys {
		data = append(data, KeyBytes(key)...)
	}
	return s.SendBytes(data)
}

// Resize changes the PTY window size. The parser grid is resized first so
// output triggered by SIGWINCH lands on a screen of the right shape.
func (s *Session) Resize(cols, rows int) error {
	if cols <= 0 || rows <= 0 {
		return pty.ErrInvalidSize
	}
	s.mu.Lock()
	s.parser.Resize(cols, rows)
	s.mu.Unlock()
	return s.proc.Resize(cols, rows)
}

// CaptureText returns the visible screen as plain text rows, trailing blanks
// trimmed like `tmux capture-pane -p`.
func (s *Session) CaptureText() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return RenderText(s.parser.Screen())
}

// CaptureRaw returns the visible screen as rows carrying SGR escape runs,
// close to `tmux capture-pane -p -e` (each attribute change starts a new CSI).
func (s *Session) CaptureRaw() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return RenderRaw(s.parser.Screen())
}

// Cursor returns the zero-based cursor position, matching tmux's cursor_x and
// cursor_y format targets.
func (s *Session) Cursor() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	screen := s.parser.Screen()
	return screen.CursorX, screen.CursorY
}

// Clipboard returns the most recent OSC 52 payload, decoded.
func (s *Session) Clipboard() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clip, s.hasClip
}

// ExitCode returns the child exit status, or -1 while it is still running.
func (s *Session) ExitCode() int { return s.proc.ExitCode() }

// Wait blocks until the child exits or the timeout elapses and reports the
// exit code. ok is false on timeout.
func (s *Session) Wait(timeout time.Duration) (code int, ok bool) {
	deadline := time.Now().Add(timeout)
	for {
		if code := s.proc.ExitCode(); code >= 0 {
			return code, true
		}
		if time.Now().After(deadline) {
			return -1, false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Kill terminates the child's whole process group and releases the PTY. It is
// idempotent and safe to call after the child already exited.
func (s *Session) Kill() error {
	s.mu.Lock()
	if s.killed {
		s.mu.Unlock()
		return nil
	}
	s.killed = true
	s.mu.Unlock()
	if pid := ptyPID(s.proc); pid > 0 {
		killGroup(pid)
	}
	err := s.proc.Close()
	select {
	case <-s.exited:
	case <-time.After(3 * time.Second):
	}
	return err
}

// pidProvider is implemented by PTY backends that can report the child pid.
type pidProvider interface{ PID() int }

func ptyPID(p pty.PTY) int {
	if provider, ok := p.(pidProvider); ok {
		return provider.PID()
	}
	return 0
}

// Manager owns named sessions for one harness process.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*Session
}

// NewManager returns an empty session manager.
func NewManager() *Manager {
	return &Manager{sessions: make(map[string]*Session)}
}

// Spawn starts a new session under name, replacing a previous session with the
// same name (its process group is killed first).
func (m *Manager) Spawn(name string, cfg Config) error {
	if name == "" {
		return errors.New("harness: empty session name")
	}
	m.mu.Lock()
	if old, ok := m.sessions[name]; ok {
		delete(m.sessions, name)
		m.mu.Unlock()
		_ = old.Kill()
		m.mu.Lock()
	}
	m.mu.Unlock()
	session, err := Spawn(cfg)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[name]; ok {
		_ = session.Kill()
		return ErrSessionExists
	}
	m.sessions[name] = session
	return nil
}

// Session returns the named live session.
func (m *Manager) Session(name string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoSession, name)
	}
	return session, nil
}

// Kill terminates and forgets the named session. An unknown name is a no-op
// so shell drivers can mirror `tmux kill-session ... || true`.
func (m *Manager) Kill(name string) error {
	m.mu.Lock()
	session, ok := m.sessions[name]
	delete(m.sessions, name)
	m.mu.Unlock()
	if !ok {
		return nil
	}
	return session.Kill()
}

// KillAll terminates every live session.
func (m *Manager) KillAll() {
	m.mu.Lock()
	sessions := m.sessions
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()
	for _, session := range sessions {
		_ = session.Kill()
	}
}

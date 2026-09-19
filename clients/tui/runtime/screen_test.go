package runtime

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/components/terminal"
	"github.com/anytty/anytty/clients/tui/kernel"
	"github.com/anytty/anytty/clients/tui/render"
)

func frameText(frame *render.Frame) string {
	var b strings.Builder
	for y := 0; y < frame.Rows(); y++ {
		for x := 0; x < frame.Cols(); x++ {
			b.WriteString(frame.CellAt(x, y).Text)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// TestScreenRevisionRepaintsWithoutInput proves M2: PTY output alone advances
// the screen revision and a caller that waits on it composes a non-nil frame
// carrying the new text, with no input event involved.
func TestScreenRevisionRepaintsWithoutInput(t *testing.T) {
	s := NewSession(Options{ViewID: "view:test:1", Cols: 40, Rows: 6}, &bytes.Buffer{}, &bytes.Buffer{})
	h := NewTerminalHandler(TerminalOptions{
		Cols:     40,
		Rows:     6,
		OnOutput: func(string) { s.MarkOutput() },
	})
	defer h.Close()

	seen := s.ScreenRevision()
	term := attachTerminal(t, h, "main", "sh", "-c", "printf 'wake up\\n'; sleep 5")

	if rev := s.WaitScreen(seen, 5*time.Second); rev == seen {
		t.Fatal("WaitScreen timed out: PTY output did not notify the session")
	}
	waitFor(t, "printf output", func() bool { return screenContains(term, "wake up") })

	component := terminal.New(nil, nil)
	component.SetScreen(term.Screen())
	placement := term.Placement(component, kernel.Rect{Width: 40, Height: 6}, true)

	update := s.FrameBytes([]Placement{placement}, nil)
	if len(update) == 0 {
		t.Fatal("FrameBytes returned no bytes after an output notification")
	}
	if !strings.Contains(frameText(s.ComposeFrame([]Placement{placement}, nil)), "wake up") {
		t.Fatalf("composed frame does not carry the new output:\n%s", frameText(s.ComposeFrame([]Placement{placement}, nil)))
	}
}

func TestWaitScreenTimeoutAndRevision(t *testing.T) {
	s := NewSession(Options{ViewID: "view:test:2", Cols: 10, Rows: 2}, &bytes.Buffer{}, &bytes.Buffer{})
	seen := s.ScreenRevision()
	start := time.Now()
	if rev := s.WaitScreen(seen, 20*time.Millisecond); rev != seen {
		t.Fatalf("WaitScreen without output = %d, want timeout %d", rev, seen)
	}
	if elapsed := time.Since(start); elapsed < 15*time.Millisecond {
		t.Fatalf("WaitScreen returned early after %v", elapsed)
	}
	s.MarkOutput()
	if rev := s.WaitScreen(seen, time.Second); rev != seen+1 {
		t.Fatalf("WaitScreen after MarkOutput = %d, want %d", rev, seen+1)
	}
	if s.ScreenRevision() != seen+1 {
		t.Fatalf("revision = %d, want %d", s.ScreenRevision(), seen+1)
	}
}

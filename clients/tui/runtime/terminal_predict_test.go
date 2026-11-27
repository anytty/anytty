package runtime

import (
	"testing"
)

// predictTestTerminal builds a terminal with prediction forced on (the env
// gate is opt-in) and a parser primed with prompt text.
func predictTestTerminal(t *testing.T, prompt string) *Terminal {
	t.Helper()
	proc := &bottomTracePTY{cols: 20, rows: 3}
	term := newTerminal("terminal:local:predict", "predict", nil, proc, 20, 3, nil)
	term.predictEnabled = true
	term.mu.Lock()
	term.parser.Write([]byte(prompt))
	term.invalidateSnapshotLocked()
	term.mu.Unlock()
	return term
}

func TestPredictionOverlayFollowsCursorAndRollsBack(t *testing.T) {
	term := predictTestTerminal(t, "hi> ")

	term.observeInput([]byte("a"))
	if pending := term.PendingPredictions(); pending != 1 {
		t.Fatalf("pending predictions = %d, want 1", pending)
	}
	state := term.State()
	line := state.Screen.Line(0)
	if len(line) < 5 || line[4].Text != "a" {
		t.Fatalf("predicted cell = %#v, want 'a' at column 4", line)
	}
	if state.CursorX != 5 {
		t.Fatalf("cursor x = %d, want 5 (advanced past the prediction)", state.CursorX)
	}

	// The real echo arrives: speculation is dropped and the parser wins.
	term.mu.Lock()
	term.parser.Write([]byte("a"))
	term.clearPredictionsLocked()
	term.invalidateSnapshotLocked()
	term.mu.Unlock()
	if pending := term.PendingPredictions(); pending != 0 {
		t.Fatalf("pending predictions after echo = %d, want 0", pending)
	}
	state = term.State()
	if got := state.Screen.Line(0)[4].Text; got != "a" {
		t.Fatalf("echoed cell = %q, want 'a'", got)
	}
	if state.CursorX != 5 {
		t.Fatalf("cursor x after echo = %d, want 5", state.CursorX)
	}
}

func TestPredictionSkippedForFullScreenAndControlKeys(t *testing.T) {
	term := predictTestTerminal(t, "")
	term.observeInput([]byte{'\r'})
	term.observeInput([]byte{0x1b, '[', 'A'})
	term.observeInput([]byte("中文"))
	if pending := term.PendingPredictions(); pending != 0 {
		t.Fatalf("control/UTF-8 input predicted %d chars, want 0", pending)
	}

	// Alternate screen (full-screen app) or mouse tracking turns it off.
	term.mu.Lock()
	term.parser.Write([]byte("\x1b[?1049h"))
	term.invalidateSnapshotLocked()
	term.mu.Unlock()
	term.observeInput([]byte("a"))
	if pending := term.PendingPredictions(); pending != 0 {
		t.Fatalf("alt-screen input predicted %d chars, want 0", pending)
	}

	term.mu.Lock()
	term.parser.Write([]byte("\x1b[?1049l\x1b[?1000h"))
	term.invalidateSnapshotLocked()
	term.mu.Unlock()
	term.observeInput([]byte("a"))
	if pending := term.PendingPredictions(); pending != 0 {
		t.Fatalf("mouse-tracking input predicted %d chars, want 0", pending)
	}
}

func TestPredictionDisabledByDefault(t *testing.T) {
	proc := &bottomTracePTY{cols: 20, rows: 3}
	term := newTerminal("terminal:local:predict-off", "predict-off", nil, proc, 20, 3, nil)
	term.predictEnabled = false
	term.observeInput([]byte("a"))
	if pending := term.PendingPredictions(); pending != 0 {
		t.Fatalf("prediction enabled without the opt-in, pending = %d", pending)
	}
}

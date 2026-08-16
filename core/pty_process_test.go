package core

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPTYProcessFactoryFeedsLiveSurface(t *testing.T) {
	command, inputLine := ptyInteractiveFixture()
	server := NewServer()
	events := server.Events(context.Background(), EventFilter{Types: []EventType{EventTerminalLiveInvalidated, EventTerminalResized, EventTerminalExited}})
	info, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-pty",
		Command: command,
		Size:    Size{Cols: 40, Rows: 8},
	})
	if err != nil {
		t.Fatalf("register pty terminal: %v", err)
	}
	if info.State != TerminalStateRunning {
		t.Fatalf("unexpected terminal state %#v", info)
	}
	waitForLiveRow(t, server, "term-pty", "alpha")
	if err := server.ResizeTerminal(context.Background(), "term-pty", 50, 10); err != nil {
		t.Fatalf("resize pty terminal: %v", err)
	}
	if err := server.WriteInput(context.Background(), "term-pty", inputLine); err != nil {
		t.Fatalf("write pty input: %v", err)
	}
	waitForLiveRow(t, server, "term-pty", "echo:beta")
	assertEventuallyEvent(t, events, EventTerminalResized, "term-pty")
	assertEventuallyEvent(t, events, EventTerminalLiveInvalidated, "term-pty")
	assertEventuallyEvent(t, events, EventTerminalExited, "term-pty")
}

func TestPTYProcessOutputHandoffHasNoHiddenQueueAndCloseUnblocksWait(t *testing.T) {
	command, _ := ptyInteractiveFixture()
	processValue, err := newPTYProcessFactory().Spawn(context.Background(), ProcessSpec{
		TerminalID: "term-pty-unbuffered", Command: command, Size: Size{Cols: 40, Rows: 8},
	})
	if err != nil {
		t.Fatal(err)
	}
	process := processValue.(*ptyProcess)
	if capacity := cap(process.outputCh); capacity != 0 {
		t.Fatalf("PTY output handoff retained a %d-chunk queue outside the shared output budget", capacity)
	}
	if ptyReadBufferBytes != int(MinTerminalOutputBufferCapacityBytes) {
		t.Fatalf("one in-flight PTY read=%d, want minimum buffer capacity %d", ptyReadBufferBytes, MinTerminalOutputBufferCapacityBytes)
	}
	if err := process.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-process.Wait():
	case <-time.After(3 * time.Second):
		t.Fatal("PTY waitLoop remained blocked after output cancellation")
	}
}

func TestPTYProcessFactoryUsesCreateDirAndEnv(t *testing.T) {
	dir, err := os.MkdirTemp("", "anytty-pty-env-")
	if err != nil {
		t.Fatalf("create pty cwd: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	server := NewServer()
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-pty-env",
		Command: ptyEnvironmentFixture(),
		Size:    Size{Cols: 120, Rows: 4},
		Options: TerminalCreateOptions{
			Dir: dir,
			Env: []string{"ANYTTY_REMOTE_TEST=ok"},
		},
	}); err != nil {
		t.Fatalf("register pty terminal: %v", err)
	}
	waitForTerminalState(t, server, "term-pty-env", TerminalStateExited)
	info, err := server.GetTerminal("term-pty-env")
	if err != nil {
		t.Fatalf("get pty env terminal: %v", err)
	}
	if info.CWD != dir || info.LiveCWD != dir {
		t.Fatalf("terminal info should keep requested cwd, got %#v want %q", info, dir)
	}
}

func TestParseProcessResourceUsage(t *testing.T) {
	sampledAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	usage, ok := parseProcessResourceUsage(4321, []byte(" 12.34 65536\n"), sampledAt)
	if !ok {
		t.Fatal("expected ps resource output to parse")
	}
	if usage.PID != 4321 || usage.CPUPercentX100 != 1234 || usage.MemoryBytes != 65536*1024 || !usage.SampledAt.Equal(sampledAt) {
		t.Fatalf("unexpected resource usage %#v", usage)
	}
	if _, ok := parseProcessResourceUsage(4321, []byte("bad\n"), sampledAt); ok {
		t.Fatal("invalid ps resource output should not parse")
	}
}

func TestParseProcessTreeResourceUsageAggregatesDescendants(t *testing.T) {
	sampledAt := time.Date(2026, 8, 10, 3, 0, 0, 0, time.UTC)
	usage, ok := parseProcessTreeResourceUsage(100, []byte(`
  1   0  9.9 5000
100   1  1.25 1000
101 100  2.50 2000
102 101  3.75 3000
200   1 80.00 9000
bad row
`), sampledAt)
	if !ok {
		t.Fatal("expected terminal process tree resource output to parse")
	}
	if usage.PID != 100 || usage.CPUPercentX100 != 750 || usage.MemoryBytes != 6000*1024 || !usage.SampledAt.Equal(sampledAt) {
		t.Fatalf("unexpected process tree resource usage %#v", usage)
	}
	if _, ok := parseProcessTreeResourceUsage(999, []byte("100 1 1.0 1000\n"), sampledAt); ok {
		t.Fatal("missing terminal root process must not produce a resource sample")
	}
}

func assertEventuallyEvent(t *testing.T, events <-chan Event, typ EventType, terminalID string) Event {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case event, ok := <-events:
			if !ok {
				t.Fatal("event channel closed")
			}
			if event.Type == typ && event.TerminalID == terminalID {
				return event
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %s event", typ)
		}
	}
}

func TestPTYProcessFactoryKillExitsProcess(t *testing.T) {
	server := NewServer()
	events := server.Events(context.Background(), EventFilter{Types: []EventType{EventTerminalExited}})
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-kill",
		Command: ptyLongRunningFixture(),
		Size:    Size{Cols: 20, Rows: 4},
	}); err != nil {
		t.Fatalf("register pty terminal: %v", err)
	}
	if err := server.KillTerminal(context.Background(), "term-kill"); err != nil {
		t.Fatalf("kill pty terminal: %v", err)
	}
	event := assertEventValue(t, events, EventTerminalExited, "term-kill")
	if event.Terminal == nil || event.Terminal.State != TerminalStateExited {
		t.Fatalf("expected exited terminal event, got %#v", event)
	}
}

func waitForLiveRow(t *testing.T, server *Server, terminalID string, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rows, err := server.LiveRows(terminalID)
		if err == nil {
			for _, row := range rows {
				if strings.Contains(row, want) {
					return
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	rows, _ := server.LiveRows(terminalID)
	t.Fatalf("timed out waiting for live row %q, got %#v", want, rows)
}

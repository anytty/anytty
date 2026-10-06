//go:build !windows

package core

import (
	"context"
	"testing"
	"time"
)

// TestPTYProcessKillEscalatesWhenSIGHUPIgnored 固定 Kill 的兜底语义：
// 子进程显式忽略 SIGHUP 时，宽限后必须升级为 SIGKILL 并派发 terminal.exited，
// 否则 terminal 会永久停在 running。
func TestPTYProcessKillEscalatesWhenSIGHUPIgnored(t *testing.T) {
	server := NewServer()
	events := server.Events(context.Background(), EventFilter{Types: []EventType{EventTerminalExited}})
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-kill-escalation",
		Command: ptyHUPIgnoringFixture(),
		Size:    Size{Cols: 20, Rows: 4},
	}); err != nil {
		t.Fatalf("register pty terminal: %v", err)
	}
	started := time.Now()
	if err := server.KillTerminal(context.Background(), "term-kill-escalation"); err != nil {
		t.Fatalf("kill pty terminal: %v", err)
	}
	event := assertEventValue(t, events, EventTerminalExited, "term-kill-escalation")
	if event.Terminal == nil || event.Terminal.State != TerminalStateExited {
		t.Fatalf("expected exited terminal event, got %#v", event)
	}
	if elapsed := time.Since(started); elapsed > 4*time.Second {
		t.Fatalf("kill escalation took too long: %v", elapsed)
	}
}

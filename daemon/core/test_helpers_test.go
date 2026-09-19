package core

import (
	"testing"
	"time"
)

func waitForTerminalState(t *testing.T, server *Server, terminalID string, want TerminalState) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		info, err := server.GetTerminal(terminalID)
		if err == nil && info.State == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	info, err := server.GetTerminal(terminalID)
	if err != nil {
		t.Fatalf("timed out waiting for terminal %q state %q: %v", terminalID, want, err)
	}
	t.Fatalf("timed out waiting for terminal %q state %q, got %#v", terminalID, want, info)
}

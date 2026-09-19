//go:build !windows

package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestServerListTerminalsReportsForegroundProcessWorkingDirectory(t *testing.T) {
	dir, err := os.MkdirTemp("", "anytty-foreground-cwd-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer()
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-foreground-cwd",
		Command: ptyLongRunningFixture(),
		Size:    Size{Cols: 20, Rows: 4},
		Options: TerminalCreateOptions{Dir: dir},
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = server.KillTerminal(context.Background(), "term-foreground-cwd")
	})

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		for _, terminal := range server.ListTerminals() {
			if terminal.ID == "term-foreground-cwd" &&
				terminal.ForegroundCWD == want {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("foreground cwd was not reported, terminals=%#v", server.ListTerminals())
}

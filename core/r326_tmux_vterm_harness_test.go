package core

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/core/history"
)

func TestR326TmuxAndCoreAuthoritativeHistoryAlignForScreenAppHistory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tmux/vterm harness requires a Unix-like PTY")
	}
	script := r326ScreenAppHistoryScript()
	tmuxCapture := r326TmxCapture(t, script, 12, 3)
	for _, want := range []string{"sync01", "sync12", "final-frame"} {
		if !strings.Contains(tmuxCapture, want) {
			t.Fatalf("tmux observable history missing %q:\n%s", want, tmuxCapture)
		}
	}
	if strings.Contains(tmuxCapture, "ALT-TRANSIENT") {
		t.Fatalf("tmux primary capture must not include pure alt-screen transient content:\n%s", tmuxCapture)
	}

	server := NewServer()
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r326",
		Command: []string{"/bin/sh", "-c", script},
		Size:    Size{Cols: 12, Rows: 3},
	}); err != nil {
		t.Fatalf("register r326 terminal: %v", err)
	}
	waitForLiveRow(t, server, "term-r326", "sync12")
	if err := server.ResizeTerminal(context.Background(), "term-r326", 16, 4); err != nil {
		t.Fatalf("resize r326 terminal: %v", err)
	}
	r326WaitForTerminalState(t, server, "term-r326", TerminalStateExited)

	rows, pageCount := r326CollectAllHistoryRows(t, server, "term-r326", 16, 4)
	text := strings.Join(historyRowTexts(rows), "\n")
	for _, want := range []string{"sync01", "sync12", "final-frame"} {
		if !strings.Contains(text, want) {
			t.Fatalf("core authoritative history missing %q after tmux-aligned PTY flow:\n%s\nrows=%#v", want, text, rows)
		}
	}
	if strings.Contains(text, "ALT-TRANSIENT") {
		t.Fatalf("core authoritative history must not seal pure alt-screen transient content:\n%s", text)
	}
	if pageCount < 2 {
		t.Fatalf("long synchronized output should require older paging beyond latest frame, page_count=%d rows=%#v", pageCount, rows)
	}
	if !historyRowsContain(rows, "post-alt") {
		t.Fatalf("core authoritative history must include post-alt primary content, rows=%#v", rows)
	}
}

func r326ScreenAppHistoryScript() string {
	return strings.Join([]string{
		"printf 'prelude\\r\\n'",
		"printf '\\033[?2026h'",
		"i=1; while [ \"$i\" -le 12 ]; do printf 'sync%02d\\r\\n' \"$i\"; i=$((i+1)); done",
		"printf '\\033[?2026l'",
		"printf '\\033[?1049hALT-TRANSIENT\\033[?1049l'",
		"sleep 1",
		"printf '\\033[?2026hpost-alt\\r\\nfinal-frame\\033[?2026l'",
	}, "; ")
}

func r326TmxCapture(t *testing.T, script string, cols int, rows int) string {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux binary not available for R326 harness")
	}
	session := fmt.Sprintf("anytty-r326-%d", time.Now().UnixNano())
	target := session + ":0.0"
	command := "/bin/sh -c " + shellQuote(script+"; sleep 2")
	if output, err := exec.Command("tmux", "new-session", "-d", "-s", session, "-x", fmt.Sprint(cols), "-y", fmt.Sprint(rows), command).CombinedOutput(); err != nil {
		t.Fatalf("start tmux r326 session: %v\n%s", err, string(output))
	}
	t.Cleanup(func() { _ = exec.Command("tmux", "kill-session", "-t", session).Run() })

	deadline := time.Now().Add(4 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		output, err := exec.Command("tmux", "capture-pane", "-t", target, "-p", "-S", "-").CombinedOutput()
		if err == nil {
			last = string(output)
			if strings.Contains(last, "final-frame") {
				return last
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for tmux r326 capture, last:\n%s", last)
	return ""
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func r326CollectAllHistoryRows(t *testing.T, server *Server, terminalID string, cols int, limit int) ([]history.HistoryRow, int) {
	t.Helper()
	snapshot, err := server.TerminalHistoryFreeze(context.Background(), terminalID, history.FreezeHistoryRequest{
		TerminalID: terminalID,
		Cols:       cols,
		Limit:      limit,
	})
	if err != nil {
		t.Fatalf("freeze history window: %v", err)
	}
	t.Cleanup(func() {
		_ = server.TerminalHistoryRelease(context.Background(), terminalID, snapshot.Token)
	})
	latest, err := server.TerminalHistoryWindow(context.Background(), terminalID, history.HistoryWindowRequest{
		TerminalID: terminalID,
		Mode:       history.HistoryWindowModeLatest,
		Cols:       cols,
		Limit:      limit,
		Token:      snapshot.Token,
	})
	if err != nil {
		t.Fatalf("latest history window: %v", err)
	}
	rows := append([]history.HistoryRow(nil), latest.Rows...)
	pageCount := 1
	cursor := latest.Boundary.Cursor
	for cursor.Valid {
		older, err := server.TerminalHistoryWindow(context.Background(), terminalID, history.HistoryWindowRequest{
			TerminalID: terminalID,
			Mode:       history.HistoryWindowModeOlder,
			Cols:       cols,
			Limit:      limit,
			Token:      snapshot.Token,
			Cursor:     cursor,
		})
		if err != nil {
			t.Fatalf("older history window: %v", err)
		}
		if len(older.Rows) == 0 {
			break
		}
		pageCount++
		rows = append(append([]history.HistoryRow(nil), older.Rows...), rows...)
		cursor = older.Boundary.Cursor
	}
	return rows, pageCount
}

func r326WaitForTerminalState(t *testing.T, server *Server, terminalID string, want TerminalState) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var last TerminalInfo
	for time.Now().Before(deadline) {
		info, err := server.GetTerminal(terminalID)
		if err == nil {
			last = info
			if info.State == want {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for terminal %q state %q, got %#v", terminalID, want, last)
}

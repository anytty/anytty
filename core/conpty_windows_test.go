//go:build windows

package core

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/anytty/anytty/core/history"
	"golang.org/x/sys/windows"
)

const (
	windowsConPTYTestHelperEnv = "ANYTTY_CONPTY_TEST_HELPER"
	windowsConPTYDLLOverride   = "ANYTTY_CONPTY_DLL"
)

func TestBundledConPTYPreservesRawVTOutput(t *testing.T) {
	useBundledConPTYForTest(t)
	process, err := newPTYProcessFactory().Spawn(context.Background(), ProcessSpec{
		TerminalID: "conpty-raw-vt",
		Command:    windowsConPTYTestCommand(),
		Size:       Size{Cols: 20, Rows: 6},
		Env:        []string{windowsConPTYTestHelperEnv + "=1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = process.Close() })

	var output bytes.Buffer
	for chunk := range process.Output() {
		_, _ = output.Write(chunk)
	}
	exit := <-process.Wait()
	if exit.Code != 0 || exit.Err != nil {
		t.Fatalf("helper exit = %#v", exit)
	}
	raw := output.String()
	if strings.Count(raw, "\x1b[47m") != 2 || strings.Count(raw, "            ") != 2 {
		t.Fatalf("ConPTY did not preserve styled blank rows: %q", raw)
	}
	for index := range 40 {
		marker := fmt.Sprintf("history-%02d", index)
		if !strings.Contains(raw, marker) {
			t.Fatalf("ConPTY output lost %q: %q", marker, raw)
		}
	}
}

func TestBundledConPTYFeedsCompleteLineHistory(t *testing.T) {
	useBundledConPTYForTest(t)
	server := NewServer(WithHistoryStorageDir(t.TempDir()))
	const terminalID = "conpty-line-history"
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      terminalID,
		Command: windowsConPTYTestCommand(),
		Size:    Size{Cols: 20, Rows: 6},
		Options: TerminalCreateOptions{Env: []string{windowsConPTYTestHelperEnv + "=1"}},
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.RemoveTerminal(terminalID) })
	waitForTerminalState(t, server, terminalID, TerminalStateExited)
	terminal, err := server.Terminal(terminalID)
	if err != nil {
		t.Fatal(err)
	}
	if err := terminal.FlushHistory(context.Background()); err != nil {
		t.Fatal(err)
	}
	window, err := server.TerminalHistoryWindow(context.Background(), terminalID, history.HistoryWindowRequest{
		TerminalID: terminalID,
		Mode:       history.HistoryWindowModeLatest,
		Cols:       20,
		Limit:      100,
	})
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(historyRowTexts(window.Rows), "\n")
	for index := range 40 {
		marker := fmt.Sprintf("history-%02d", index)
		if !strings.Contains(text, marker) {
			t.Fatalf("line history lost %q: %q", marker, text)
		}
	}
}

func TestWindowsConPTYOutputHelper(t *testing.T) {
	if os.Getenv(windowsConPTYTestHelperEnv) != "1" {
		return
	}
	var mode uint32
	handle := windows.Handle(os.Stdout.Fd())
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		panic(err)
	}
	if err := windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		panic(err)
	}
	_, _ = fmt.Fprint(os.Stdout, "\x1b[47m            \x1b[0m\r\n")
	_, _ = fmt.Fprint(os.Stdout, "\x1b[47m            \x1b[0m\r\n")
	for index := range 40 {
		_, _ = fmt.Fprintf(os.Stdout, "history-%02d\r\n", index)
	}
	os.Exit(0)
}

func useBundledConPTYForTest(t *testing.T) {
	t.Helper()
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skipf("bundled ConPTY is not published for %s", runtime.GOARCH)
	}
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv(windowsConPTYDLLOverride, "")
}

func windowsConPTYTestCommand() []string {
	return []string{os.Args[0], "-test.run=^TestWindowsConPTYOutputHelper$"}
}

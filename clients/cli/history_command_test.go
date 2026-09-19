package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anytty/anytty/pool/core/history/linehist"
)

func TestHistoryDeleteCommandRemovesTerminalHistoryWhilePoolStopped(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := resolveV3HistoryStorageDir()
	file, err := linehist.OpenCompressedLineFile(dir, "manual-delete", linehist.CompressedLineFileOptions{Compression: "zstd"})
	if err != nil {
		t.Fatal(err)
	}
	if err := file.AppendLines([]linehist.Line{{Runs: []linehist.Run{{Text: "payload"}}, HardEnd: true}}); err != nil {
		t.Fatal(err)
	}
	path := file.Path()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	command := newRootCmd()
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{
		"--socket", filepath.Join(t.TempDir(), "daemon.sock"),
		"--log-file", filepath.Join(t.TempDir(), "daemon.log"),
		"history", "delete", "manual-delete",
	})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "deleted 2 history file") {
		t.Fatalf("unexpected output %q", output.String())
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("history file remains: %v", err)
	}
}

func TestHistoryDeleteCommandRefusesWhilePoolOwnsRuntimeRecord(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	socketPath := filepath.Join(t.TempDir(), "daemon.sock")
	release, err := acquirePoolRuntimeRecord(socketPath, filepath.Join(t.TempDir(), "daemon.log"), "")
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	command := newRootCmd()
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"--socket", socketPath, "history", "delete", "--all"})
	err = command.Execute()
	if err == nil || !strings.Contains(err.Error(), "requires the terminal pool to be stopped") {
		t.Fatalf("unexpected running-pool error: %v", err)
	}
}

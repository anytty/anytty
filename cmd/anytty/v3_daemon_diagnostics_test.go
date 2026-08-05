package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDaemonMemstatsStageFileIsSanitized(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stage.txt")
	if err := os.WriteFile(path, []byte("copy oldest!\n"), 0o644); err != nil {
		t.Fatalf("write stage file: %v", err)
	}
	t.Setenv(daemonMemstatsStageFileEnv, path)

	if got := daemonHeapProfileReason(readDaemonMemstatsStageFile()); got != "copyoldest" {
		t.Fatalf("unexpected sanitized stage %q", got)
	}
}

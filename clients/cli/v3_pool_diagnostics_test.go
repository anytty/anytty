package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPoolMemstatsStageFileIsSanitized(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stage.txt")
	if err := os.WriteFile(path, []byte("copy oldest!\n"), 0o644); err != nil {
		t.Fatalf("write stage file: %v", err)
	}
	t.Setenv(poolMemstatsStageFileEnv, path)

	if got := poolHeapProfileReason(readPoolMemstatsStageFile()); got != "copyoldest" {
		t.Fatalf("unexpected sanitized stage %q", got)
	}
}

func TestPoolEnvPrefersNewNameThenLegacy(t *testing.T) {
	t.Setenv(poolMemstatsDirEnv, "")
	t.Setenv(legacyPoolMemstatsDirEnv, "/legacy")
	if got := poolEnv(poolMemstatsDirEnv, legacyPoolMemstatsDirEnv); got != "/legacy" {
		t.Fatalf("legacy fallback = %q", got)
	}
	t.Setenv(poolMemstatsDirEnv, "/new")
	if got := poolEnv(poolMemstatsDirEnv, legacyPoolMemstatsDirEnv); got != "/new" {
		t.Fatalf("new env must win = %q", got)
	}
}

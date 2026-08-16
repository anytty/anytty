package runtimepath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSocketPathUsesRuntimeDirectoryAndNativeSeparators(t *testing.T) {
	runtimeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	if got, want := SocketPath("anytty.sock"), filepath.Join(runtimeDir, "anytty.sock"); got != want {
		t.Fatalf("socket path = %q, want %q", got, want)
	}
}

func TestSocketPathFallsBackToUserScopedTemporaryPath(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	got := SocketPath("anytty.sock")
	want := filepath.Join(os.TempDir(), "anytty-"+userDiscriminator(), "anytty.sock")
	if got != want {
		t.Fatalf("fallback socket path = %q, want user-scoped path %q", got, want)
	}
}

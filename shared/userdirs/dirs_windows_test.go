//go:build windows

package userdirs

import (
	"path/filepath"
	"testing"
)

func TestWindowsDefaultsUseApplicationDataDirectories(t *testing.T) {
	config := filepath.Join(t.TempDir(), "roaming")
	state := filepath.Join(t.TempDir(), "local")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("APPDATA", config)
	t.Setenv("LOCALAPPDATA", state)
	if got := ConfigHome(); got != config {
		t.Fatalf("config home = %q, want %q", got, config)
	}
	if got := StateHome(); got != state {
		t.Fatalf("state home = %q, want %q", got, state)
	}
}

package userdirs

import (
	"path/filepath"
	"testing"
)

func TestExplicitXDGDirectoriesOverridePlatformDefaults(t *testing.T) {
	config := filepath.Join(t.TempDir(), "config")
	state := filepath.Join(t.TempDir(), "state")
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("XDG_STATE_HOME", state)
	if got := ConfigHome(); got != config {
		t.Fatalf("config home = %q, want %q", got, config)
	}
	if got := StateHome(); got != state {
		t.Fatalf("state home = %q, want %q", got, state)
	}
}

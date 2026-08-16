package core

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPathDefaultsUseDaemonUserShellAndHome(t *testing.T) {
	home := t.TempDir()
	working := t.TempDir()
	t.Setenv("SHELL", "/bin/anytty-test-shell")
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	} else {
		t.Setenv("HOME", home)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(working); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	defaults := pathDefaults()
	if len(defaults.DefaultCommand) != 1 || defaults.DefaultCommand[0] != "/bin/anytty-test-shell" {
		t.Fatalf("default command = %#v", defaults.DefaultCommand)
	}
	if defaults.DefaultCWD != filepath.Clean(home) {
		t.Fatalf("default cwd = %q, want home %q", defaults.DefaultCWD, home)
	}
	if defaults.Platform != runtime.GOOS {
		t.Fatalf("platform = %q, want %q", defaults.Platform, runtime.GOOS)
	}
}

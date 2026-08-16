//go:build unix

package securefs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenOrCreatePrivateDirectoryRejectsUnsafeOrSymlink(t *testing.T) {
	unsafeDirectory := filepath.Join(t.TempDir(), "unsafe")
	if err := os.Mkdir(unsafeDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unsafeDirectory, 0o750); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenOrCreatePrivateDirectory(unsafeDirectory); err == nil {
		t.Fatal("group-accessible directory was accepted")
	}

	privateDirectory := filepath.Join(t.TempDir(), "private")
	handle, err := OpenOrCreatePrivateDirectory(privateDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "private-link")
	if err := os.Symlink(privateDirectory, link); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenOrCreatePrivateDirectory(link); err == nil {
		t.Fatal("directory symlink was accepted")
	}
}

package securefs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecureFileAndDirectoryArePrivate(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "private")
	if err := os.Mkdir(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := SecureDirectory(dir); err != nil {
		t.Fatalf("secure directory: %v", err)
	}
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("secret"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := SecureFile(path); err != nil {
		t.Fatalf("secure file: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !IsPrivateFile(path, info) {
		t.Fatal("secured file did not satisfy the platform privacy contract")
	}
}

func TestSecureHandlePermissionsArePrivate(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "handle-private")
	if err := os.Mkdir(directory, 0o777); err != nil {
		t.Fatal(err)
	}
	directoryHandle, err := os.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := SecureDirectoryHandle(directoryHandle); err != nil {
		_ = directoryHandle.Close()
		t.Fatal(err)
	}
	if err := ValidatePrivateDirectoryHandle(directoryHandle); err != nil {
		_ = directoryHandle.Close()
		t.Fatal(err)
	}
	if err := directoryHandle.Close(); err != nil {
		t.Fatal(err)
	}

	file, err := os.OpenFile(filepath.Join(directory, "secret"), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		t.Fatal(err)
	}
	if err := SecureFileHandle(file); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := ValidatePrivateFileHandle(file); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

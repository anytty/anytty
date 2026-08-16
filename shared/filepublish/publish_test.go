package filepublish

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenameReplacesTargetAndSyncsDirectory(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "state.json")
	temporary := filepath.Join(dir, "state.tmp")
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(temporary, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Rename(temporary, target); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if err := SyncDirectory(dir); err != nil {
		t.Fatalf("sync directory: %v", err)
	}
	payload, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "new" {
		t.Fatalf("published payload = %q", payload)
	}
	if _, err := os.Stat(temporary); !os.IsNotExist(err) {
		t.Fatalf("temporary file remains: %v", err)
	}
}

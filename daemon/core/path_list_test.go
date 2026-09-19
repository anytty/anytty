package core

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestListPathDirectoriesUsesDaemonFilesystem(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"demo", "dev", "delta", ".dot"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "doc.txt"), []byte("file"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolve temp dir: %v", err)
	}

	result, err := listPathDirectories("d", 10)
	if err != nil {
		t.Fatalf("list path dirs: %v", err)
	}
	got := make([]string, 0, len(result.Entries))
	for _, entry := range result.Entries {
		got = append(got, entry.Path)
	}
	want := []string{"delta" + string(os.PathSeparator), "demo" + string(os.PathSeparator), "dev" + string(os.PathSeparator)}
	if !reflect.DeepEqual(got, want) || result.BasePath != resolvedDir || result.Missing || result.Truncated {
		t.Fatalf("unexpected directory candidates got=%#v result=%#v want=%#v", got, result, want)
	}

	hidden, err := listPathDirectories(".", 10)
	if err != nil {
		t.Fatalf("list hidden dirs: %v", err)
	}
	if len(hidden.Entries) != 1 || hidden.Entries[0].Path != ".dot"+string(os.PathSeparator) {
		t.Fatalf("hidden directories require dot prefix, got %#v", hidden)
	}

	missing, err := listPathDirectories(filepath.Join(dir, "missing", "x"), 10)
	if err != nil {
		t.Fatalf("missing path should be prompt empty state, got err=%v", err)
	}
	if !missing.Missing || len(missing.Entries) != 0 {
		t.Fatalf("missing path should return missing empty result, got %#v", missing)
	}
}

package host

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func packageFixture(t *testing.T, id string) string {
	t.Helper()
	dir := t.TempDir()
	manifest := fmt.Sprintf("id = %q\nversion = \"1\"\napi = \"anytty.plugin/1\"\n[daemon]\ncommand = [\"./plugin\"]\n", id)
	if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin"), []byte("original"), 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestInstallSnapshotsPackageAndPreservesExecutability(t *testing.T) {
	source := packageFixture(t, "org.test.one")
	registry := Registry{Version: 1}
	if _, err := registry.Install(source, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	installed := filepath.Join(registry.Plugins[0].Directory, "plugin")
	if err := os.WriteFile(filepath.Join(source, "plugin"), []byte("edited"), 0700); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(installed)
	if err != nil || string(data) != "original" {
		t.Fatalf("installation aliases source: %q %v", data, err)
	}
	info, err := os.Stat(installed)
	if err != nil || info.Mode().Perm()&0100 == 0 {
		t.Fatal("installed executable lost executable bit")
	}
	if _, err := registry.Install(source, filepath.Join(source, "nested")); err == nil {
		t.Fatal("accepted recursive installation")
	}
}

func TestInstallRejectsSymlinkWithoutPublishing(t *testing.T) {
	source := packageFixture(t, "org.test.one")
	if err := os.Symlink(filepath.Join(source, "plugin"), filepath.Join(source, "symlink")); err != nil {
		t.Skip(err)
	}
	registry := Registry{Version: 1}
	destination := t.TempDir()
	if _, err := registry.Install(source, destination); err == nil {
		t.Fatal("accepted symlink")
	}
	entries, err := os.ReadDir(destination)
	if err != nil || len(entries) != 0 || len(registry.Plugins) != 0 {
		t.Fatal("failed install published partial package")
	}
}

func TestRegistryConcurrentMutationsAndFailureAtomicity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plugins.yaml")
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		directory := packageFixture(t, fmt.Sprintf("org.test.plugin%d", i))
		workers.Add(1)
		go func() {
			defer workers.Done()
			if err := UpdateRegistry(context.Background(), path, func(r *Registry) error { _, err := r.Link(directory); return err }); err != nil {
				t.Error(err)
			}
		}()
	}
	workers.Wait()
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	r, err := LoadRegistry(path)
	if err != nil || len(r.Plugins) != 8 {
		t.Fatalf("lost concurrent update: %v %v", r, err)
	}
	err = UpdateRegistry(context.Background(), path, func(r *Registry) error { r.Plugins = nil; return fmt.Errorf("aborted") })
	if err == nil {
		t.Fatal("mutation error omitted")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(before) {
		t.Fatal("aborted transaction changed registry")
	}
}

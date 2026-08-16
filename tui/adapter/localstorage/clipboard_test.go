package localstorage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

func TestClipboardStoragePersistsMaterializedText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "clipboard-history.json")
	storage := ClipboardStorage{Path: path}
	sourceRef := state.DefaultClipboardStorageRef("endpoint-a")
	targetRef := state.DefaultClipboardStorageRef("endpoint-b")
	snapshot := state.SnapshotClipboardForStorage(state.ClipboardStore{}.WithCopiedText("copied across endpoints"))

	saved, err := storage.SaveClipboard(context.Background(), port.ClipboardStorageSaveRequest{
		Ref: sourceRef, Snapshot: snapshot, CheckVersion: true, ExpectedVersion: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Version != 1 {
		t.Fatalf("saved version = %d, want 1", saved.Version)
	}
	loaded, err := storage.LoadClipboard(context.Background(), targetRef)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Found || loaded.Version != 1 || len(loaded.Snapshot.Entries) != 1 || loaded.Snapshot.Entries[0].Text != "copied across endpoints" {
		t.Fatalf("loaded clipboard = %#v", loaded)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("clipboard file mode = %#o, want private", info.Mode().Perm())
	}
}

func TestClipboardStorageRejectsStaleWriter(t *testing.T) {
	storage := ClipboardStorage{Path: filepath.Join(t.TempDir(), "clipboard-history.json")}
	ref := state.DefaultClipboardStorageRef(state.DefaultWorkspaceID)
	first := state.SnapshotClipboardForStorage(state.ClipboardStore{}.WithCopiedText("first"))
	if _, err := storage.SaveClipboard(context.Background(), port.ClipboardStorageSaveRequest{
		Ref: ref, Snapshot: first, CheckVersion: true, ExpectedVersion: 0,
	}); err != nil {
		t.Fatal(err)
	}
	second := state.SnapshotClipboardForStorage(state.ClipboardStore{}.WithCopiedText("second"))
	_, err := storage.SaveClipboard(context.Background(), port.ClipboardStorageSaveRequest{
		Ref: ref, Snapshot: second, CheckVersion: true, ExpectedVersion: 0,
	})
	if !errors.Is(err, port.ErrClipboardStorageConflict) {
		t.Fatalf("stale save error = %v", err)
	}
	loaded, loadErr := storage.LoadClipboard(context.Background(), ref)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if len(loaded.Snapshot.Entries) != 1 || loaded.Snapshot.Entries[0].Text != "first" {
		t.Fatalf("stale writer replaced clipboard: %#v", loaded)
	}
}

func TestClipboardStorageWatchReportsExternalVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboard-history.json")
	storage := ClipboardStorage{Path: path, PollInterval: 5 * time.Millisecond}
	ref := state.DefaultClipboardStorageRef(state.DefaultWorkspaceID)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events, err := storage.WatchClipboard(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := state.SnapshotClipboardForStorage(state.ClipboardStore{}.WithCopiedText("watched"))
	if _, err := storage.SaveClipboard(ctx, port.ClipboardStorageSaveRequest{
		Ref: ref, Snapshot: snapshot, CheckVersion: true, ExpectedVersion: 0,
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.Version != 1 || event.Ref.Version != 1 {
			t.Fatalf("watch event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for clipboard storage event")
	}
}

func TestClipboardStorageRequiresExplicitPath(t *testing.T) {
	storage := ClipboardStorage{}
	if _, err := storage.LoadClipboard(context.Background(), state.ClipboardStorageRef{}); err == nil {
		t.Fatal("missing path was accepted")
	}
}

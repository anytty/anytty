package transfer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStorePersistsAndReopensAcrossProcesses(t *testing.T) {
	dir := t.TempDir()
	token, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	targetModified := time.Unix(1700000000, 0).UTC()
	record := Record{
		ID: "upload-1", Token: token, Direction: DirectionUpload, Path: "/data/big.bin",
		OwnerID: "owner", Size: 1 << 20, Offset: 4096, TargetSize: 1 << 20, TargetModifiedAt: targetModified,
		TempPath: filepath.Join(dir, "big.bin.part"), ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := store.Put(record); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	resumed, ok := reopened.Lookup(token)
	if !ok || resumed.ID != "upload-1" || resumed.Offset != 4096 {
		t.Fatalf("reopened lookup = %#v, ok=%v", resumed, ok)
	}
	if !resumed.MatchesTarget(1<<20, targetModified) {
		t.Fatal("reopened record must match unchanged target")
	}
	if resumed.MatchesTarget(2<<20, targetModified) || resumed.MatchesTarget(1<<20, targetModified.Add(time.Second)) {
		t.Fatal("record must reject changed size or mtime")
	}
	info, err := os.Stat(filepath.Join(dir, "transfers.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("store file mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestStorePruneAndDeleteByToken(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	expired, _ := NewToken()
	active, _ := NewToken()
	if err := store.Put(Record{ID: "expired", Token: expired, Direction: DirectionDownload, Path: "/x", TargetSize: 1, ExpiresAt: time.Now().Add(-time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(Record{ID: "active", Token: active, Direction: DirectionDownload, Path: "/y", TargetSize: 1, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if removed := store.Prune(time.Now()); removed != 1 {
		t.Fatalf("prune removed %d, want 1", removed)
	}
	if _, ok := store.Lookup(expired); ok {
		t.Fatal("expired record survived prune")
	}
	if err := store.DeleteByToken(active); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Lookup(active); ok {
		t.Fatal("deleted record survived DeleteByToken")
	}
}

func TestStoreRejectsCorruptState(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "transfers.json"), []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir); err == nil {
		t.Fatal("corrupt store must fail closed")
	}
}

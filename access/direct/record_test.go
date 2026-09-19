package direct

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListenerRecordRoundTripAndRemove(t *testing.T) {
	path := RecordPath(filepath.Join(t.TempDir(), "daemon.sock"))
	record := ListenerRecord{Listen: "0.0.0.0:41120", Signaling: "192.168.1.8:41120", ICETCP: "192.168.1.8:41120", UpdatedAt: time.Now().UTC()}
	if err := WriteListenerRecord(path, record); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("record permissions = %o", info.Mode().Perm())
	}
	loaded, err := ReadListenerRecord(path)
	if err != nil || loaded.Listen != record.Listen || loaded.Signaling != record.Signaling {
		t.Fatalf("record = %#v err=%v", loaded, err)
	}
	if err := RemoveListenerRecord(path); err != nil {
		t.Fatal(err)
	}
	if err := RemoveListenerRecord(path); err != nil {
		t.Fatalf("repeated remove: %v", err)
	}
	if _, err := ReadListenerRecord(path); err == nil {
		t.Fatal("removed record still readable")
	}
}

func TestListenerRecordRejectsCorruptPayload(t *testing.T) {
	path := RecordPath(filepath.Join(t.TempDir(), "daemon.sock"))
	if err := os.WriteFile(path, []byte(`{"version":1,"signaling":"a"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadListenerRecord(path); err == nil {
		t.Fatal("incomplete record accepted")
	}
	if err := WriteListenerRecord(path, ListenerRecord{Signaling: "a"}); err == nil {
		t.Fatal("incomplete record written")
	}
}

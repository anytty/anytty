//go:build windows

package core

import (
	"strings"
	"testing"
)

func TestFileSystemRootListReturnsWindowsDrives(t *testing.T) {
	result, handled, err := fileSystemRootList(FileListRequest{Path: "/"})
	if err != nil {
		t.Fatal(err)
	}
	if !handled || result.Path != "/" || len(result.Entries) == 0 {
		t.Fatalf("unexpected Windows root result: handled=%v result=%+v", handled, result)
	}
	for _, entry := range result.Entries {
		if entry.Type != "dir" || !strings.HasSuffix(entry.Name, ":") || entry.Path != entry.Name+"/" {
			t.Fatalf("invalid drive entry: %+v", entry)
		}
	}
}

func TestFileSystemRootListRejectsCursor(t *testing.T) {
	_, handled, err := fileSystemRootList(FileListRequest{Path: "/", Cursor: "stale"})
	if !handled || err == nil {
		t.Fatalf("expected handled cursor error, got handled=%v err=%v", handled, err)
	}
}

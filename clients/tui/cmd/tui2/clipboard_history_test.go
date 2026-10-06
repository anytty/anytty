package main

import (
	"path/filepath"
	"testing"
)

func TestClipboardHistoryPersistsAndDeduplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboard.json")
	store := newClipboardHistory(path, 2)
	first, err := store.Add("one")
	if err != nil || first.ID == "" {
		t.Fatalf("add first = %+v, err=%v", first, err)
	}
	if _, err := store.Add("two"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Add("three"); err != nil {
		t.Fatal(err)
	}
	reloaded := newClipboardHistory(path, 2)
	entries := reloaded.List()
	if len(entries) != 2 || entries[0].Text != "three" || entries[1].Text != "two" {
		t.Fatalf("reloaded entries = %+v", entries)
	}
	entry, err := reloaded.Add("two")
	if err != nil || entry.Text != "two" {
		t.Fatalf("dedupe add = %+v, err=%v", entry, err)
	}
	entries = reloaded.List()
	if len(entries) != 2 || entries[0].Text != "two" || entries[1].Text != "three" {
		t.Fatalf("dedupe order = %+v", entries)
	}
}

func TestClipboardHistoryRowsAreJSONEntries(t *testing.T) {
	store := newClipboardHistory(filepath.Join(t.TempDir(), "clipboard.json"), 4)
	if _, err := store.Add("line one\nline two"); err != nil {
		t.Fatal(err)
	}
	rows := store.Rows()
	if len(rows) != 1 || rows[0] == "" || rows[0][0] != '{' {
		t.Fatalf("rows = %q", rows)
	}
}

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCatalogIsUpToDate is the drift guard: the committed catalog must be
// byte-for-byte what the generator produces right now.
func TestCatalogIsUpToDate(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	onDisk, err := os.ReadFile(filepath.Join(root, catalogPath))
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}
	want, err := generate(root)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !bytes.Equal(onDisk, want.Bytes()) {
		t.Fatalf("catalog %s is stale\nrun: go run ./clients/tui/cmd/tui2-widgetdoc", catalogPath)
	}
}

// TestEveryEntryRendersContent renders every visual entry through the real
// host pipeline: a blank picture or a missing expected glyph fails.
func TestEveryEntryRendersContent(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	sections, err := catalog(root)
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	visual := 0
	for _, section := range sections {
		for _, entry := range section.entries {
			if entry.build == nil {
				continue
			}
			visual++
			if len(entry.glyphs) == 0 {
				t.Errorf("%s: visual entry declares no expected glyphs", entry.name)
			}
			text, err := renderEntry(entry)
			if err != nil {
				t.Fatalf("%s: render: %v", entry.name, err)
			}
			if strings.TrimSpace(text) == "" {
				t.Fatalf("%s: rendered a blank picture", entry.name)
			}
			for _, glyph := range entry.glyphs {
				if !strings.Contains(text, glyph) {
					t.Fatalf("%s: picture is missing glyph %q:\n%s", entry.name, glyph, text)
				}
			}
		}
	}
	if visual == 0 {
		t.Fatal("catalog has no visual entries")
	}
}

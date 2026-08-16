package render

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderPackageDoesNotImportBubbleTea(t *testing.T) {
	assertRenderImports(t, func(file string, path string) {
		if path == "github.com/charmbracelet/bubbletea" || strings.Contains(path, "/bubbles") {
			t.Fatalf("%s imports Bubble Tea contract package %s", file, path)
		}
	})
}

func TestRenderPackageDoesNotImportRuntimeOrServices(t *testing.T) {
	forbidden := []string{
		"github.com/anytty/anytty/tui/app",
		"github.com/anytty/anytty/tui/port",
		"github.com/anytty/anytty/tui/terminalhost",
	}
	assertRenderImports(t, func(file string, path string) {
		for _, prefix := range forbidden {
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				t.Fatalf("%s imports runtime/service boundary package %s", file, path)
			}
		}
	})
}

func assertRenderImports(t *testing.T, check func(file string, path string)) {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob render files: %v", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, imported := range parsed.Imports {
			path := strings.Trim(imported.Path.Value, `"`)
			check(file, path)
		}
	}
}

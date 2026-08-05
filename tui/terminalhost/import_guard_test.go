package terminalhost

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTerminalHostDoesNotImportBubbleTea(t *testing.T) {
	for _, pattern := range []string{"*.go"} {
		files, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("glob %s: %v", pattern, err)
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
				if path == "github.com/charmbracelet/bubbletea" || strings.Contains(path, "/bubbles") {
					t.Fatalf("%s imports Bubble Tea contract package %s", file, path)
				}
			}
		}
	}
}

func TestTerminalHostDoesNotImportForbiddenStateOrServices(t *testing.T) {
	root := ".."
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, dir := range []string{"state", "services"} {
			forbidden := "github.com/anytty/anytty/tui/" + dir
			if strings.Contains(string(source), forbidden) {
				t.Fatalf("%s must not import %s under %s", file, dir, root)
			}
		}
	}
}

package apilayer

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPILayerAndAPIMappingRespectDependencyDirection(t *testing.T) {
	assertImportsExclude(t, ".", []string{
		"github.com/anytty/anytty/client",
		"github.com/anytty/anytty/cmd",
		"github.com/anytty/anytty/internal/protocol",
		"github.com/anytty/anytty/private",
		"github.com/anytty/anytty/proto/runtimepb",
		"github.com/anytty/anytty/proto/wirepb",
		"github.com/anytty/anytty/remote",
		"github.com/anytty/anytty/shared/transport",
		"github.com/anytty/anytty/tui",
	})
	assertImportsExclude(t, "../api_mapping", []string{
		"github.com/anytty/anytty/api_layer",
		"github.com/anytty/anytty/client",
		"github.com/anytty/anytty/cmd",
		"github.com/anytty/anytty/internal/protocol",
		"github.com/anytty/anytty/private",
		"github.com/anytty/anytty/proto/runtimepb",
		"github.com/anytty/anytty/proto/wirepb",
		"github.com/anytty/anytty/remote",
		"github.com/anytty/anytty/shared",
		"github.com/anytty/anytty/tui",
	})
	assertImportsExclude(t, "../core", []string{
		"github.com/anytty/anytty/api_layer",
		"github.com/anytty/anytty/api_mapping",
	})
}

func assertImportsExclude(t *testing.T, root string, forbidden []string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range parsed.Imports {
			importPath := strings.Trim(imported.Path.Value, `"`)
			for _, prefix := range forbidden {
				if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
					t.Errorf("%s imports forbidden owner %s", path, importPath)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

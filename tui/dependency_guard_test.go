package tui_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTUIDirectoryDependencies 保证 TUI 的 application port、adapter 和测试资产保持单向依赖。
func TestTUIDirectoryDependencies(t *testing.T) {
	assertImportsExclude(t, "port", []string{
		"github.com/anytty/anytty/internal/protocol",
		"github.com/anytty/anytty/tui/adapter",
		"github.com/anytty/anytty/tui/testkit",
		"os/exec",
	})
	for _, root := range []string{"state", "app", "render"} {
		assertImportsExclude(t, root, []string{
			"github.com/anytty/anytty/internal/protocol",
			"github.com/anytty/anytty/shared/transport",
			"github.com/anytty/anytty/shared/remoteauth",
			"github.com/anytty/anytty/remote/client",
			"github.com/anytty/anytty/remote/webrtc",
			"github.com/anytty/anytty/tui/adapter",
			"github.com/anytty/anytty/tui/testkit",
		})
	}
}

// TestTUIPortContainsOnlyProductionContracts 防止 fake 或宿主 IO 实现重新进入 application port。
func TestTUIPortContainsOnlyProductionContracts(t *testing.T) {
	allowedImports := map[string]struct{}{
		"context": {}, "errors": {}, "time": {},
		"github.com/anytty/anytty/proto/apipb": {},
		"github.com/anytty/anytty/tui/input":   {},
		"github.com/anytty/anytty/tui/state":   {},
	}
	err := filepath.WalkDir("port", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, forbidden := range []string{"type Fake", "SystemClipboardService", "exec.Command"} {
			if strings.Contains(string(content), forbidden) {
				t.Errorf("%s contains forbidden port implementation %q", path, forbidden)
			}
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, content, 0)
		if err != nil {
			return err
		}
		for _, imported := range parsed.Imports {
			importPath := strings.Trim(imported.Path.Value, `"`)
			if _, ok := allowedImports[importPath]; !ok {
				t.Errorf("%s imports non-contract dependency %s", path, importPath)
			}
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			if function.Recv == nil {
				t.Errorf("%s implements function %s in contract-only port", path, function.Name.Name)
				continue
			}
			if receiverUsesStructType(parsed, function.Recv.List[0].Type) {
				t.Errorf("%s implements method %s on a struct-owned service in contract-only port", path, function.Name.Name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func receiverUsesStructType(file *ast.File, receiver ast.Expr) bool {
	if pointer, ok := receiver.(*ast.StarExpr); ok {
		receiver = pointer.X
	}
	name, ok := receiver.(*ast.Ident)
	if !ok {
		return true
	}
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok.String() != "type" {
			continue
		}
		for _, spec := range general.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if ok && typeSpec.Name.Name == name.Name {
				_, isStruct := typeSpec.Type.(*ast.StructType)
				return isStruct
			}
		}
	}
	return true
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
					t.Errorf("%s imports forbidden dependency %s", path, importPath)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

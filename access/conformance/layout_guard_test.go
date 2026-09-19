package conformance_test

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot resolves the repository root from this package directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// walkProductionImports calls fn with every import path of every non-test Go
// file below root.
func walkProductionImports(t *testing.T, root string, fn func(path, importPath string)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range parsed.Imports {
			fn(path, strings.Trim(imported.Path.Value, `"`))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan %s: %v", root, err)
	}
}

func hasPrefix(importPath string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}
	return false
}

// TestPoolProductPackagesDoNotImportClientSide enforces the dependency
// direction of the terminal pool product line: pool/core, access/remote and
// access/cloud never depend on the access engine/gateway, the clients tree or
// the old TUI trees. access/transport is a shared wire/transport layer and is
// deliberately allowed (access/cloud uses the webrtc/cloud transports).
func TestPoolProductPackagesDoNotImportClientSide(t *testing.T) {
	forbidden := []string{
		"github.com/anytty/anytty/access/engine",
		"github.com/anytty/anytty/access/localweb",
		"github.com/anytty/anytty/access/provider",
		"github.com/anytty/anytty/access/sdk",
		"github.com/anytty/anytty/access/cmd",
		"github.com/anytty/anytty/clients",
		"github.com/anytty/anytty/tui",
		"github.com/anytty/anytty/tui2",
	}
	root := repoRoot(t)
	for _, pkg := range []string{
		filepath.Join("pool", "core"),
		filepath.Join("access", "remote"),
		filepath.Join("access", "cloud"),
	} {
		root := filepath.Join(root, pkg)
		walkProductionImports(t, root, func(path, importPath string) {
			if hasPrefix(importPath, forbidden...) {
				t.Errorf("%s imports forbidden client-side package %s", path, importPath)
			}
		})
	}
}

// TestSingleBinaryPolicyKeepsDaemonEntryDeleted pins the single-binary policy:
// `anytty pool run` is the terminal pool entry, so the old anyttyd binary must
// not reappear.
func TestSingleBinaryPolicyKeepsDaemonEntryDeleted(t *testing.T) {
	for _, path := range []string{
		filepath.Join(repoRoot(t), "daemon", "cmd", "anyttyd"),
		filepath.Join(repoRoot(t), "cmd", "anyttyd"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("single-binary policy violated: %s must stay deleted", path)
		}
	}
}

// TestAccessDoesNotImportTheTUIFrontend keeps the access product line
// frontend-agnostic: no access package may import clients/tui or the removed
// tui2 path.
func TestAccessDoesNotImportTheTUIFrontend(t *testing.T) {
	root := filepath.Join(repoRoot(t), "access")
	forbidden := []string{
		"github.com/anytty/anytty/clients/tui",
		"github.com/anytty/anytty/tui2",
	}
	walkProductionImports(t, root, func(path, importPath string) {
		if hasPrefix(importPath, forbidden...) {
			t.Errorf("%s imports forbidden TUI frontend %s", path, importPath)
		}
	})
}

// TestLayoutProgramOnlyDependsOnTheUISDK pins the layout-program contract:
// the host binary is built from the SDK, the layout config and the UI wire
// packages only.
func TestLayoutProgramOnlyDependsOnTheUISDK(t *testing.T) {
	allowed := []string{
		"github.com/anytty/anytty/clients/tui/sdk",
		"github.com/anytty/anytty/clients/tui/config",
		"github.com/anytty/anytty/proto/ui",
		"google.golang.org/protobuf",
	}
	command := exec.Command("go", "list", "-deps", "./clients/tui/cmd/tui2-shell")
	command.Dir = repoRoot(t)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("go list -deps layout program: %v", err)
	}
	for _, dep := range strings.Fields(string(output)) {
		if !strings.Contains(dep, ".") || dep == "github.com/anytty/anytty/clients/tui/cmd/tui2-shell" {
			continue
		}
		if !hasPrefix(dep, allowed...) {
			t.Errorf("layout program depends on %s; only the UI SDK, layout config and UI wire are allowed", dep)
		}
	}
}

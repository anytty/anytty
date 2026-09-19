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

// TestDaemonProductPackagesDoNotImportClientSide enforces the dependency
// direction of the daemon product line: daemon/core, daemon/remote and
// daemon/cloud never depend on the access engine/gateway, the clients tree or
// the old TUI trees. access/transport is a shared wire/transport layer and is
// deliberately allowed (daemon/cloud uses the webrtc/cloud transports).
// daemon/cmd/anyttyd is the composition entry and is checked separately.
func TestDaemonProductPackagesDoNotImportClientSide(t *testing.T) {
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
	for _, pkg := range []string{"core", "remote", "cloud"} {
		root := filepath.Join(root, "daemon", pkg)
		walkProductionImports(t, root, func(path, importPath string) {
			if hasPrefix(importPath, forbidden...) {
				t.Errorf("%s imports forbidden client-side package %s", path, importPath)
			}
		})
	}
}

// TestAnyTTYDaemonEntryOnlyComposesCLI documents the one deliberate exception
// of the daemon tree: the anyttyd entry binary composes the shared CLI
// implementation while the daemon product packages stay independent.
func TestAnyTTYDaemonEntryOnlyComposesCLI(t *testing.T) {
	root := filepath.Join(repoRoot(t), "daemon", "cmd", "anyttyd")
	walkProductionImports(t, root, func(path, importPath string) {
		if !strings.HasPrefix(importPath, "github.com/anytty/anytty/") {
			return
		}
		if !hasPrefix(importPath, "github.com/anytty/anytty/clients/cli") {
			t.Errorf("daemon entry %s grows an unexpected internal dependency on %s", path, importPath)
		}
	})
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

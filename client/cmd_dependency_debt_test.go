package client_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandConcreteDependencyDebtDoesNotGrow(t *testing.T) {
	expectedImports := map[string]struct{}{
		"file_command.go|github.com/anytty/anytty/internal/protocol":         {},
		"terminal_command.go|github.com/anytty/anytty/internal/protocol":     {},
		"v3_client_access.go|github.com/anytty/anytty/shared/remoteauth":     {},
		"v3_client_access.go|github.com/anytty/anytty/shared/transport":      {},
		"v3_client_access.go|github.com/anytty/anytty/shared/transport/unix": {},
		"v3_direct_daemon.go|github.com/anytty/anytty/remote/webrtc":         {},
		"v3_pair_command.go|github.com/anytty/anytty/shared/remoteauth":      {},
		"v3_pair_command.go|github.com/anytty/anytty/shared/transport/unix":  {},
	}
	expectedHelpers := map[string]struct{}{
		"v3_daemon_client.go|dialOrStartV3Client":           {},
		"v3_daemon_client.go|dialOrStartV3ClientContext":    {},
		"v3_daemon_client.go|v3DialClient":                  {},
		"v3_endpoint_client.go|openEndpointProtocolClient":  {},
		"v3_endpoint_client.go|probeEndpointProtocolClient": {},
		"daemon_lifecycle.go|v3DialClient":                  {},
		"endpoint_command.go|probeEndpointProtocolClient":   {},
		"file_command.go|openEndpointProtocolClient":        {},
		"terminal_command.go|openEndpointProtocolClient":    {},
		"v3_access_command.go|dialOrStartV3ClientContext":   {},
		"v3_command.go|dialOrStartV3Client":                 {},
		"v3_control_commands.go|dialOrStartV3Client":        {},
		"v3_history_backlog_command.go|dialOrStartV3Client": {},
		"v3_pair_command.go|dialOrStartV3ClientContext":     {},
		"v3_root_command.go|dialOrStartV3Client":            {},
	}
	seenImports := map[string]struct{}{}
	seenHelpers := map[string]struct{}{}
	allowedCompositionImports := map[string]struct{}{
		"v3_client_runtime.go|github.com/anytty/anytty/shared/remoteauth": {},
	}
	concretePrefixes := []string{
		"github.com/anytty/anytty/internal/protocol",
		"github.com/anytty/anytty/shared/transport",
		"github.com/anytty/anytty/shared/remoteauth",
		"github.com/anytty/anytty/remote/client",
		"github.com/anytty/anytty/remote/webrtc",
	}
	helperNames := map[string]struct{}{
		"v3DialClient": {}, "probeEndpointProtocolClient": {}, "openEndpointProtocolClient": {},
		"dialOrStartV3Client": {}, "dialOrStartV3ClientContext": {},
	}
	err := filepath.WalkDir("../cmd/anytty", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		file := filepath.Base(path)
		for _, imported := range parsed.Imports {
			importPath := strings.Trim(imported.Path.Value, `"`)
			if hasImportPrefix(importPath, concretePrefixes) {
				key := file + "|" + importPath
				if _, composition := allowedCompositionImports[key]; composition {
					continue
				}
				seenImports[key] = struct{}{}
				if _, ok := expectedImports[key]; !ok {
					t.Errorf("new command concrete import debt: %s", key)
				}
			}
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if !ok {
				return true
			}
			if _, tracked := helperNames[identifier.Name]; tracked {
				key := file + "|" + identifier.Name
				seenHelpers[key] = struct{}{}
				if _, ok := expectedHelpers[key]; !ok {
					t.Errorf("new command direct helper debt: %s", key)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	assertDebtSetMatches(t, "concrete import", expectedImports, seenImports)
	assertDebtSetMatches(t, "direct helper", expectedHelpers, seenHelpers)
}

func hasImportPrefix(importPath string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}
	return false
}

func assertDebtSetMatches(t *testing.T, kind string, expected, seen map[string]struct{}) {
	t.Helper()
	for key := range expected {
		if _, ok := seen[key]; !ok {
			t.Errorf("%s debt was removed; delete the obsolete guard entry: %s", kind, key)
		}
	}
}

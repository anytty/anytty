package conformance_test

import (
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/conformance"
)

// repoPath resolves a path relative to the repository root, independent of
// the package working directory.
func repoPath(t *testing.T, parts ...string) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(append([]string{root}, parts...)...)
}

// checkInterpreter runs the whole fixture suite against an external SDK
// conformance program, exactly like `tui2-sdk-verify --cmd`.
func checkInterpreter(t *testing.T, argv []string, fixtures []conformance.Fixture) {
	t.Helper()
	if _, err := exec.LookPath(argv[0]); err != nil {
		t.Skipf("%s not installed", argv[0])
	}
	results := conformance.Run(fixtures, conformance.CommandFactory{Argv: argv}, 10*time.Second)
	for _, result := range results {
		if !result.Passed {
			t.Errorf("fixture %s failed at step %d: %v", result.Name, result.Steps, result.Err)
		}
	}
}

// TestPythonSDKConformance certifies the official Python SDK against the same
// fixtures as the Go SDK (the "any language SDK can self-verify" contract).
func TestPythonSDKConformance(t *testing.T) {
	fixtures := loadFixtures(t)
	checkInterpreter(t, []string{"python3", repoPath(t, "clients", "tui", "sdk", "python", "conformance.py")}, fixtures)
}

// TestTSSDKConformance certifies the official TS/JS SDK against the same
// fixtures.
func TestTSSDKConformance(t *testing.T) {
	fixtures := loadFixtures(t)
	checkInterpreter(t, []string{"node", repoPath(t, "clients", "tui", "sdk", "ts", "conformance.js")}, fixtures)
}

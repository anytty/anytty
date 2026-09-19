package sdk_test

import (
	"os/exec"
	"strings"
	"testing"
)

// forbiddenDeps are the host-internal packages a layout program must never
// depend on (ARCHITECTURE §1): the wire protocol is the only contract.
var forbiddenDeps = []string{
	"github.com/anytty/anytty/clients/tui/kernel",
	"github.com/anytty/anytty/clients/tui/runtime",
	"github.com/anytty/anytty/clients/tui/render",
	"github.com/anytty/anytty/clients/tui/components",
}

// listDeps returns the transitive, non-test dependencies of the package
// pattern (run from this package directory).
func listDeps(t *testing.T, pattern string) []string {
	t.Helper()
	out, err := exec.Command("go", "list", "-deps", pattern).Output()
	if err != nil {
		t.Fatalf("go list -deps %s: %v", pattern, err)
	}
	return strings.Fields(string(out))
}

func assertFrameworkFree(t *testing.T, dep string) {
	t.Helper()
	for _, bad := range forbiddenDeps {
		if dep == bad || strings.HasPrefix(dep, bad+"/") {
			t.Errorf("layout program depends on host-internal package %s; the protocol is the only contract", dep)
		}
	}
}

// TestSDKIsFrameworkFree is the dependency guard of the "layout programs are
// host-independent" contract: the Go SDK (including sdk/widgets) may build
// against the standard library plus the wire protocol packages (proto/ui,
// proto/ui/protobuf) and nothing else. If this test fails, the decoupling
// regressed.
func TestSDKIsFrameworkFree(t *testing.T) {
	allowed := []string{
		"github.com/anytty/anytty/clients/tui/sdk",
		"github.com/anytty/anytty/proto/ui",
		"github.com/anytty/anytty/proto/ui/protobuf",
		"google.golang.org/protobuf",
	}
	for _, dep := range listDeps(t, "./...") {
		assertFrameworkFree(t, dep)
		if !strings.Contains(dep, ".") {
			continue // stdlib
		}
		ok := false
		for _, prefix := range allowed {
			if dep == prefix || strings.HasPrefix(dep, prefix+"/") {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("sdk has non-protocol dependency %s; want stdlib + proto + protobuf only", dep)
		}
	}
}

// TestGoShellIsFrameworkFree pins the same contract for the reference Go
// layout program: cmd/tui2-shell speaks the protocol through sdk/config only
// and never imports host internals (no kernel types, no render palette, no
// component packages).
func TestGoShellIsFrameworkFree(t *testing.T) {
	shell := "github.com/anytty/anytty/clients/tui/cmd/tui2-shell"
	deps := listDeps(t, shell)
	found := false
	for _, dep := range deps {
		if dep == shell {
			found = true
		}
		assertFrameworkFree(t, dep)
	}
	if !found {
		t.Fatalf("go list -deps %s did not report the package itself", shell)
	}
}

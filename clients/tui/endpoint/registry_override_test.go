package endpoint

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
)

// registryFixtureEntry is one local-unix endpoint of a test registry file.
type registryFixtureEntry struct {
	ID     string
	Label  string
	Socket string
}

// writeRegistryFixture writes a normalized endpoints.yaml with one enabled
// local-unix route per entry; the first entry becomes the registry default.
func writeRegistryFixture(t *testing.T, path string, entries ...registryFixtureEntry) string {
	t.Helper()
	if len(entries) == 0 {
		t.Fatal("registry fixture needs at least one endpoint")
	}
	registry := clientendpoint.Registry{Version: clientendpoint.RegistryVersion, Endpoints: make(map[clientendpoint.EndpointID]clientendpoint.Endpoint, len(entries))}
	for _, entry := range entries {
		id := clientendpoint.EndpointID(entry.ID)
		label := entry.Label
		if strings.TrimSpace(label) == "" {
			label = entry.ID
		}
		registry.Endpoints[id] = clientendpoint.Endpoint{
			ID: id, Label: label, LabelSource: clientendpoint.SourceManual, ConnectMode: clientendpoint.ConnectAuto, Enabled: true,
			Routes: map[clientendpoint.RouteID]clientendpoint.AccessRoute{
				"local": {ID: "local", Kind: clientendpoint.RouteLocalUnix, Enabled: true, Source: clientendpoint.SourceManual, PolicySource: clientendpoint.SourceManual, Socket: entry.Socket},
			},
		}
	}
	registry.Default = clientendpoint.EndpointID(entries[0].ID)
	normalized, err := registry.Normalize()
	if err != nil {
		t.Fatalf("normalize registry fixture: %v", err)
	}
	payload, err := clientendpoint.Encode(normalized)
	if err != nil {
		t.Fatalf("encode registry fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir registry fixture: %v", err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatalf("write registry fixture: %v", err)
	}
	return path
}

func configsByName(configs []Config) map[string]Config {
	byName := make(map[string]Config, len(configs))
	for _, cfg := range configs {
		byName[cfg.Name] = cfg
	}
	return byName
}

// TestRegistryPathsExplicitFirstThenDefault pins the resolution order used by
// both the host flag/env and the manager: explicit files first, the default
// client/endpoint registry last, path-list splitting and de-duplication.
func TestRegistryPathsExplicitFirstThenDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	def := SharedRegistryPath()
	a := filepath.Join(dir, "a.yaml")
	b := filepath.Join(dir, "b.yaml")
	if got, want := RegistryPaths(a, b), []string{a, b, def}; !reflect.DeepEqual(got, want) {
		t.Fatalf("RegistryPaths(a, b) = %v, want %v", got, want)
	}
	if got, want := RegistryPaths(def, a), []string{def, a}; !reflect.DeepEqual(got, want) {
		t.Fatalf("RegistryPaths(default, a) = %v, want %v (default must not repeat)", got, want)
	}
	t.Setenv(RegistryEnvVar, strings.Join([]string{a, b, a}, string(os.PathListSeparator)))
	if got, want := RegistryPathsFromEnv(), []string{a, b, def}; !reflect.DeepEqual(got, want) {
		t.Fatalf("RegistryPathsFromEnv() = %v, want %v", got, want)
	}
}

// TestLoadSharedEndpointConfigsMergesByNameExplicitFirst is the M3 merge
// contract: different names coexist, a same-name entry from the explicit
// (legacy) registry wins, and the shadowed duplicate is reported.
func TestLoadSharedEndpointConfigsMergesByNameExplicitFirst(t *testing.T) {
	dir := t.TempDir()
	explicit := writeRegistryFixture(t, filepath.Join(dir, "legacy", clientendpoint.DefaultFileName),
		registryFixtureEntry{ID: "legacy", Socket: "/tmp/legacy.sock"},
		registryFixtureEntry{ID: "both", Socket: "/tmp/explicit-both.sock"})
	def := writeRegistryFixture(t, filepath.Join(dir, "dev", clientendpoint.DefaultFileName),
		registryFixtureEntry{ID: "dev", Socket: "/tmp/dev.sock"},
		registryFixtureEntry{ID: "both", Socket: "/tmp/default-both.sock"})

	configs, warnings := LoadSharedEndpointConfigs(explicit, def)
	byName := configsByName(configs)
	if len(byName) != 3 {
		t.Fatalf("merged configs = %+v; want legacy, dev and both", configs)
	}
	if byName["legacy"].Socket != "/tmp/legacy.sock" || byName["dev"].Socket != "/tmp/dev.sock" {
		t.Fatalf("merged configs = %+v; want both names to coexist", configs)
	}
	if byName["both"].Socket != "/tmp/explicit-both.sock" {
		t.Fatalf("same-name endpoint socket = %q, want the explicit registry value", byName["both"].Socket)
	}
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "both") || !strings.Contains(joined, "shadowed") || !strings.Contains(joined, explicit) {
		t.Fatalf("merge warnings = %q; want a readable shadowed-duplicate warning naming %s", warnings, explicit)
	}
}

// TestLoadSharedEndpointConfigsMissingAndCorruptStayReadable: a missing
// explicit file and a corrupt file each produce one readable warning, and the
// lower-priority registry still loads; nothing panics.
func TestLoadSharedEndpointConfigsMissingAndCorruptStayReadable(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.yaml")
	corrupt := filepath.Join(dir, "corrupt.yaml")
	if err := os.WriteFile(corrupt, []byte("version: 3\nendpoints: [broken\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	def := writeRegistryFixture(t, filepath.Join(dir, "dev", clientendpoint.DefaultFileName),
		registryFixtureEntry{ID: "dev", Socket: "/tmp/dev.sock"})

	configs, warnings := LoadSharedEndpointConfigs(missing, corrupt, def)
	if byName := configsByName(configs); len(byName) != 1 || byName["dev"].Socket != "/tmp/dev.sock" {
		t.Fatalf("configs = %+v; want the default registry to stay usable", configs)
	}
	if len(warnings) != 2 {
		t.Fatalf("warnings = %q; want one missing-file and one corrupt-file warning", warnings)
	}
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, missing) || !strings.Contains(joined, "does not exist") {
		t.Fatalf("warnings = %q; want a readable missing-file warning for %s", warnings, missing)
	}
	if !strings.Contains(joined, corrupt) {
		t.Fatalf("warnings = %q; want a readable corrupt-file warning for %s", warnings, corrupt)
	}
}

// TestSharedConfigForEndpointPriorityAndCorruptFallback covers dial-time
// resolution: explicit wins, a missing explicit file never masks the default,
// a corrupt explicit file falls through when the default defines the endpoint,
// and a name that only lives in a corrupt registry yields a readable error.
func TestSharedConfigForEndpointPriorityAndCorruptFallback(t *testing.T) {
	dir := t.TempDir()
	explicit := writeRegistryFixture(t, filepath.Join(dir, "legacy", clientendpoint.DefaultFileName),
		registryFixtureEntry{ID: "both", Socket: "/tmp/explicit-both.sock"})
	def := writeRegistryFixture(t, filepath.Join(dir, "dev", clientendpoint.DefaultFileName),
		registryFixtureEntry{ID: "both", Socket: "/tmp/default-both.sock"},
		registryFixtureEntry{ID: "dev", Socket: "/tmp/dev.sock"})
	corrupt := filepath.Join(dir, "corrupt.yaml")
	if err := os.WriteFile(corrupt, []byte("version: nope\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing.yaml")

	cfg, ok, err := SharedConfigForEndpointIn([]string{explicit, def}, "both")
	if err != nil || !ok || cfg.Socket != "/tmp/explicit-both.sock" {
		t.Fatalf("explicit priority = %+v, %v, %v; want the explicit socket", cfg, ok, err)
	}
	if _, ok, err := SharedConfigForEndpointIn([]string{explicit, def}, "dev"); err != nil || !ok {
		t.Fatalf("lower-priority lookup = ok %v, err %v; want the default registry value", ok, err)
	}
	cfg, ok, err = SharedConfigForEndpointIn([]string{missing, def}, "dev")
	if err != nil || !ok || cfg.Socket != "/tmp/dev.sock" {
		t.Fatalf("missing explicit lookup = %+v, %v, %v; want the default registry value", cfg, ok, err)
	}
	cfg, ok, err = SharedConfigForEndpointIn([]string{corrupt, def}, "dev")
	if err != nil || !ok || cfg.Socket != "/tmp/dev.sock" {
		t.Fatalf("corrupt explicit fallback = %+v, %v, %v; want the default registry value", cfg, ok, err)
	}
	if _, ok, err := SharedConfigForEndpointIn([]string{corrupt}, "ghost"); ok || err == nil || !strings.Contains(err.Error(), corrupt) {
		t.Fatalf("corrupt-only lookup = ok %v, err %v; want a readable corrupt-registry error", ok, err)
	}
}

// TestSharedConfigForEndpointResolvesEnvRegistry is the runbook behavior:
// TUI2_ENDPOINTS points at the legacy production registry while the dev
// default path stays merged for its own endpoints.
func TestSharedConfigForEndpointResolvesEnvRegistry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "dev-xdg"))
	explicit := writeRegistryFixture(t, filepath.Join(dir, "legacy", clientendpoint.DefaultFileName),
		registryFixtureEntry{ID: "legacy-env", Socket: "/tmp/legacy-env.sock"})
	writeRegistryFixture(t, SharedRegistryPath(), registryFixtureEntry{ID: "dev-env", Socket: "/tmp/dev-env.sock"})
	t.Setenv(RegistryEnvVar, explicit)

	cfg, ok, err := SharedConfigForEndpoint("legacy-env")
	if err != nil || !ok || cfg.Socket != "/tmp/legacy-env.sock" {
		t.Fatalf("SharedConfigForEndpoint(legacy-env) = %+v, %v, %v", cfg, ok, err)
	}
	configs, warnings := LoadSharedEndpointConfigs()
	if len(warnings) != 0 {
		t.Fatalf("LoadSharedEndpointConfigs() warnings = %q; want none", warnings)
	}
	byName := configsByName(configs)
	if byName["legacy-env"].Socket != "/tmp/legacy-env.sock" || byName["dev-env"].Socket != "/tmp/dev-env.sock" {
		t.Fatalf("merged configs = %+v; want legacy and dev endpoints", configs)
	}
}

// TestRegistryReadsNeverWriteExplicitFile pins the read-only contract: no
// load/lookup path may touch the legacy registry bytes.
func TestRegistryReadsNeverWriteExplicitFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "dev-xdg"))
	explicit := writeRegistryFixture(t, filepath.Join(dir, "legacy", clientendpoint.DefaultFileName),
		registryFixtureEntry{ID: "readonly", Socket: "/tmp/readonly.sock"})
	before, err := os.ReadFile(explicit)
	if err != nil {
		t.Fatal(err)
	}

	if _, warnings := LoadSharedEndpointConfigs(explicit, SharedRegistryPath()); len(warnings) != 0 {
		t.Fatalf("warnings = %q; want none", warnings)
	}
	if _, ok, err := SharedConfigForEndpointIn(RegistryPaths(explicit), "readonly"); err != nil || !ok {
		t.Fatalf("lookup = ok %v, err %v; want the explicit endpoint", ok, err)
	}
	after, err := os.ReadFile(explicit)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("explicit registry bytes changed; tui2 registry reads must stay read-only")
	}
}

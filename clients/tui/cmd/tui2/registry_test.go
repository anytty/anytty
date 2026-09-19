package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	"github.com/anytty/anytty/clients/tui/endpoint"
)

// hostRegistryEndpoint is one enabled local-unix endpoint for the host-level
// registry fixtures.
func hostRegistryEndpoint(id, label, socket string) clientendpoint.Endpoint {
	return clientendpoint.Endpoint{
		ID: clientendpoint.EndpointID(id), Label: label, LabelSource: clientendpoint.SourceManual,
		ConnectMode: clientendpoint.ConnectAuto, Enabled: true,
		Routes: map[clientendpoint.RouteID]clientendpoint.AccessRoute{
			"local": {ID: "local", Kind: clientendpoint.RouteLocalUnix, Enabled: true, Source: clientendpoint.SourceManual, PolicySource: clientendpoint.SourceManual, Socket: socket},
		},
	}
}

func writeHostRegistry(t *testing.T, path string, endpoints ...clientendpoint.Endpoint) string {
	t.Helper()
	if len(endpoints) == 0 {
		t.Fatal("registry fixture needs at least one endpoint")
	}
	registry := clientendpoint.Registry{Version: clientendpoint.RegistryVersion, Endpoints: make(map[clientendpoint.EndpointID]clientendpoint.Endpoint, len(endpoints))}
	for _, item := range endpoints {
		registry.Endpoints[item.ID] = item
	}
	registry.Default = endpoints[0].ID
	normalized, err := registry.Normalize()
	if err != nil {
		t.Fatalf("normalize host registry fixture: %v", err)
	}
	payload, err := clientendpoint.Encode(normalized)
	if err != nil {
		t.Fatalf("encode host registry fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestHostReadsTUI2EndpointsEnvRegistry is the dev runbook behavior: starting
// the host with only TUI2_ENDPOINTS set (no pairing, no copies) lists the
// legacy endpoint, keeps the dev default registry merged, and lets the
// explicit file win when both define the same name.
func TestHostReadsTUI2EndpointsEnvRegistry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "dev-xdg"))
	legacy := writeHostRegistry(t, filepath.Join(dir, "legacy", clientendpoint.DefaultFileName),
		hostRegistryEndpoint("legacy-host", "legacy host", "/tmp/legacy-host.sock"),
		hostRegistryEndpoint("shadow-host", "legacy shadow", "/tmp/legacy-shadow.sock"))
	writeHostRegistry(t, filepath.Join(dir, "dev-xdg", "anytty", clientendpoint.DefaultFileName),
		hostRegistryEndpoint("dev-host", "dev host", "/tmp/dev-host.sock"),
		hostRegistryEndpoint("shadow-host", "dev shadow", "/tmp/dev-shadow.sock"))
	t.Setenv(endpoint.RegistryEnvVar, legacy)

	host := NewHost(Options{LoadSharedRegistry: true})
	t.Cleanup(func() { _ = host.endpoints.Close() })

	for _, name := range []string{"legacy-host", "dev-host", "shadow-host"} {
		if kind, ok := host.endpoints.Kind(name); !ok || kind != endpoint.KindDaemon {
			t.Fatalf("endpoint %q not registered as daemon (kind %q, ok %v)", name, kind, ok)
		}
	}
	shadow, ok := host.endpoints.Config("shadow-host")
	if !ok || shadow.Socket != "/tmp/legacy-shadow.sock" {
		t.Fatalf("shadow-host config = %+v, %v; want the explicit legacy socket", shadow, ok)
	}
	host.noticeMu.Lock()
	notices := append([][2]string(nil), host.notices...)
	host.noticeMu.Unlock()
	joined := make([]string, 0, len(notices))
	for _, notice := range notices {
		joined = append(joined, notice[0]+":"+notice[1])
	}
	text := strings.Join(joined, "\n")
	if !strings.Contains(text, "shadow-host") || !strings.Contains(text, "shadowed") {
		t.Fatalf("host notices = %q; want a shadowed-duplicate warning", text)
	}
}

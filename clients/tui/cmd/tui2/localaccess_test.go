package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/endpoint"
	"github.com/anytty/anytty/proto/access/wire"
)

func newLocalAccessTestHost(t *testing.T) (*Host, string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)
	manager := endpoint.NewManager(endpoint.Options{
		DialTimeout: 50 * time.Millisecond,
		CallTimeout: 50 * time.Millisecond,
		BackoffMin:  10 * time.Millisecond,
		BackoffMax:  20 * time.Millisecond,
	})
	t.Cleanup(func() { _ = manager.Close() })
	socket := filepath.Join(dir, fmt.Sprintf("anytty-v3-wire%d.sock", wire.Version))
	return &Host{endpoints: manager}, socket
}

func TestRegisterLocalAccessEndpoint(t *testing.T) {
	host, socket := newLocalAccessTestHost(t)
	host.registerLocalAccessEndpoint()
	if _, ok := host.endpoints.Config(localAccessEndpointName); ok {
		t.Fatal("local-access must not register without a socket")
	}

	if err := os.WriteFile(socket, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	host.registerLocalAccessEndpoint()
	cfg, ok := host.endpoints.Config(localAccessEndpointName)
	if !ok {
		t.Fatal("local-access was not registered")
	}
	if cfg.Kind != endpoint.KindDaemon || cfg.Socket != socket || cfg.ConnectMode != endpoint.ConnectLocalUnix {
		t.Fatalf("local-access config = %+v", cfg)
	}
	host.registerLocalAccessEndpoint() // idempotent
}

func TestRegisterLocalAccessEndpointSkipsSharedSocket(t *testing.T) {
	host, socket := newLocalAccessTestHost(t)
	if err := os.WriteFile(socket, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := host.endpoints.Register(endpoint.Config{
		Name: "local", Kind: endpoint.KindDaemon, ConnectMode: endpoint.ConnectLocalUnix, Socket: socket,
	}); err != nil {
		t.Fatalf("register local: %v", err)
	}
	host.registerLocalAccessEndpoint()
	if _, ok := host.endpoints.Config(localAccessEndpointName); ok {
		t.Fatal("local-access must not duplicate an endpoint with the same socket")
	}
}

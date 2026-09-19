package endpoint

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/pty"
)

// hermeticRegistryPath points the shared runtime at an absent registry so the
// unit tests never read a developer's paired endpoints.yaml.
func hermeticRegistryPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "endpoints.yaml")
}

// TestSharedStackLocalUnixLifecycle runs the real shared connection stack
// (client/runtime + client/adapter/protocol + shared local route) against the
// identity-proving framed fake daemon: health, list, create, attach, input,
// resize-owner CAS, a connection loss that must NOT look like a terminal
// exit, snapshot re-seed and kill. It is the M1/G1 regression harness for the
// sharedClient adapter.
func TestSharedStackLocalUnixLifecycle(t *testing.T) {
	d := newFramedFakeDaemon(t)
	m := NewManager(Options{
		DialTimeout:  2 * time.Second,
		CallTimeout:  2 * time.Second,
		BackoffMin:   30 * time.Millisecond,
		BackoffMax:   120 * time.Millisecond,
		RegistryPath: hermeticRegistryPath(t),
	})
	t.Cleanup(func() { _ = m.Close() })
	if err := m.Register(Config{Name: "shareddev", Kind: KindDaemon, Socket: d.socket}); err != nil {
		t.Fatalf("register shared endpoint: %v", err)
	}
	m.SetOnNotice(func(level, message string) { t.Logf("notice %s: %s", level, message) })
	waitFor(t, 5*time.Second, func() bool { return m.Health("shareddev") == HealthOK }, "shared endpoint health ok")
	id := createTerminal(t, m, d, "shareddev")

	p := m.NewRemotePTY(pty.Config{Endpoint: "shareddev", ID: id, Cols: 80, Rows: 24})
	if err := p.Start(); err != nil {
		t.Fatalf("shared remote pty start: %v", err)
	}
	reader := startReader(p)
	if _, err := p.Write([]byte("echo SHARED-OK\r")); err != nil {
		t.Fatalf("write: %v", err)
	}
	waitText(t, reader, "SHARED-OK")

	if err := p.Resize(100, 30); err != nil {
		t.Fatalf("resize over shared session: %v", err)
	}
	waitFor(t, time.Second, func() bool {
		size := d.terminal(id).sizeProto()
		return size.GetCols() == 100 && size.GetRows() == 30
	}, "daemon resize over shared session")

	// A dropped connection is transient: health flips offline and back, the
	// RemotePTY is NOT marked exited, and the rebound attachment continues.
	d.disconnect()
	waitFor(t, 5*time.Second, func() bool { return m.Health("shareddev") == HealthOffline }, "shared health offline")
	waitFor(t, 5*time.Second, func() bool { return m.Health("shareddev") == HealthOK }, "shared health back online")
	if p.Exited() {
		t.Fatal("shared remote pty must survive a reconnect")
	}
	waitFor(t, 5*time.Second, func() bool { return strings.Count(reader.text(), "SHARED-OK") >= 2 }, "shared snapshot re-seed")
	// The rebound attachment is published just after the snapshot seed; retry
	// the first post-reconnect keystroke within a bounded window.
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := p.Write([]byte("echo SHARED-AGAIN\r")); err == nil {
			break
		} else if time.Now().After(deadline) {
			t.Fatalf("write after reconnect: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	waitText(t, reader, "SHARED-AGAIN")

	if err := m.Kill(context.Background(), "shareddev", id); err != nil {
		t.Fatalf("kill over shared session: %v", err)
	}
	waitFor(t, 5*time.Second, func() bool { return p.Exited() }, "shared pty exit")
}

// TestSharedStackDirectMissingRouteIsReadable pins the M3 error surface for
// direct endpoints that are not in the shared registry: a readable route
// validation error, never a crash or a silent fallback.
func TestSharedStackDirectMissingRouteIsReadable(t *testing.T) {
	m := NewManager(Options{DialTimeout: 500 * time.Millisecond, CallTimeout: 500 * time.Millisecond, BackoffMin: 30 * time.Millisecond, BackoffMax: 120 * time.Millisecond, RegistryPath: hermeticRegistryPath(t)})
	t.Cleanup(func() { _ = m.Close() })
	if err := m.Register(Config{Name: "no-direct", Kind: KindDaemon, ConnectMode: ConnectDirectWebRTC}); err != nil {
		t.Fatalf("register direct endpoint: %v", err)
	}
	waitFor(t, 5*time.Second, func() bool { return m.Health("no-direct") == HealthOffline }, "direct endpoint offline")
	p := m.NewRemotePTY(pty.Config{Endpoint: "no-direct", ID: "x", Cols: 80, Rows: 24})
	err := p.Start()
	if err == nil || !strings.Contains(err.Error(), "signaling_addresses") {
		t.Fatalf("direct start error = %v; want readable missing-route error", err)
	}
}

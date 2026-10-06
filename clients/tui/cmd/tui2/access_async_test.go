package main

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/endpoint"
	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/clients/tui/runtime"
	"github.com/anytty/anytty/proto/access/apipb"

	gproto "google.golang.org/protobuf/proto"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// TestAccessCallIsAsync pins the isolation contract of the access bridge: the
// session loop must never run endpoint I/O inline. accessCall therefore
// returns pending=true (the RESPONSE is delivered later by Complete) instead of
// blocking until the endpoint answers. Before the async fix it ran the dial and
// the call inline while the session mutex was held, which froze rendering for
// the whole call timeout whenever an endpoint was slow or offline.
func TestAccessCallIsAsync(t *testing.T) {
	mgr := endpoint.NewManager(endpoint.Options{RegistryPath: t.TempDir() + "/none.yaml"})
	defer mgr.Close()

	host := &Host{endpoints: mgr}
	host.session = runtime.NewSession(runtime.Options{ViewID: "v", Cols: 80, Rows: 24}, bytes.NewReader(nil), io.Discard)
	inner := runtime.NewTerminalHandler(runtime.TerminalOptions{
		Cols: 80, Rows: 24,
		NewPTY: func(cfg pty.Config) pty.PTY { return mgr.NewRemotePTY(cfg) },
	})
	defer inner.Close()
	gate := &gateHandler{host: host, inner: inner}

	command, err := gproto.Marshal(&apipb.CommandEnvelope{
		Command: &apipb.CommandEnvelope_StorageGet{StorageGet: &apipb.StorageGetCommand{
			Key: &apipb.StorageKey{AppId: "herdr", Key: "pinned", Scope: apipb.StorageScope_STORAGE_SCOPE_PRIVATE},
		}},
	})
	if err != nil {
		t.Fatalf("marshal command: %v", err)
	}

	start := time.Now()
	outcome, pending := gate.accessCall(runtime.Request{
		Epoch:     1,
		RequestID: 7,
		Params:    &pb.MethodParams{Endpoint: "local", AccessCommand: command},
	})
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("accessCall blocked for %v; endpoint I/O must run on a background goroutine", elapsed)
	}
	_ = outcome
	if !pending {
		t.Fatal("accessCall returned pending=false; it must defer the RESPONSE to Complete")
	}

	// The background call fails fast against an unregistered endpoint and
	// completes the request, which removes it from the in-flight set.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if host.session.Pending() == 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("access.call never completed; the RESPONSE would never be sent")
}

// TestOpenAccessStreamIsAsync is the same isolation contract for
// access.stream.open: the open dial must not run on the session loop.
func TestOpenAccessStreamIsAsync(t *testing.T) {
	mgr := endpoint.NewManager(endpoint.Options{RegistryPath: t.TempDir() + "/none.yaml"})
	defer mgr.Close()

	host := &Host{endpoints: mgr}
	host.session = runtime.NewSession(runtime.Options{ViewID: "v", Cols: 80, Rows: 24}, bytes.NewReader(nil), io.Discard)
	gate := &gateHandler{host: host}

	resource, err := gproto.Marshal(&apipb.ResourceHandle{OpaqueToken: []byte("tok")})
	if err != nil {
		t.Fatalf("marshal resource: %v", err)
	}
	start := time.Now()
	_, pending := gate.openAccessStream(runtime.Request{
		Epoch:     1,
		RequestID: 9,
		Params:    &pb.MethodParams{Endpoint: "local", StreamId: 3, AccessResource: resource},
	})
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("openAccessStream blocked for %v; endpoint I/O must run on a background goroutine", elapsed)
	}
	if !pending {
		t.Fatal("openAccessStream returned pending=false; it must defer the RESPONSE to Complete")
	}
	// The reserved id is released when the background open fails.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if host.session.Pending() == 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("access.stream.open never completed")
}

// TestDaemonCreateAndAttachAreAsync pins the same isolation contract for the
// remote terminal path: create/restart/attach on a daemon endpoint dial and
// attach over the network, so they must run on a background worker instead of
// the protocol reader. Before the fix, a slow or offline endpoint froze the
// frame loop (and every other pane) for the whole call timeout.
func TestDaemonCreateAndAttachAreAsync(t *testing.T) {
	mgr := endpoint.NewManager(endpoint.Options{
		RegistryPath: t.TempDir() + "/none.yaml",
		// A dial that never succeeds quickly: this is the slow/offline case.
		DialTimeout: 2 * time.Second,
		CallTimeout: 2 * time.Second,
	})
	defer mgr.Close()
	if err := mgr.Register(endpoint.Config{
		Name: "slowdev", Kind: endpoint.KindDaemon,
		ConnectMode: endpoint.ConnectLocalUnix, Socket: t.TempDir() + "/missing.sock",
	}); err != nil {
		t.Fatal(err)
	}

	host := &Host{endpoints: mgr}
	host.session = runtime.NewSession(runtime.Options{ViewID: "v", Cols: 80, Rows: 24}, bytes.NewReader(nil), io.Discard)
	inner := runtime.NewTerminalHandler(runtime.TerminalOptions{
		Cols: 80, Rows: 24,
		NewPTY: func(cfg pty.Config) pty.PTY { return mgr.NewRemotePTY(cfg) },
	})
	defer inner.Close()
	gate := &gateHandler{host: host, inner: inner}

	for _, method := range []string{"terminal.create", "terminal.attach"} {
		start := time.Now()
		_, pending := gate.Handle(runtime.Request{
			Epoch: 1, RequestID: 1, Method: runtime.Method{Name: method},
			Params: &pb.MethodParams{Endpoint: "slowdev", Id: "term-x"},
		})
		if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
			t.Fatalf("%s blocked for %v; daemon attach must run on a background worker", method, elapsed)
		}
		if !pending {
			t.Fatalf("%s returned pending=false; it must defer the RESPONSE", method)
		}
	}
	// The background workers fail against the missing socket and complete.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if host.session.Pending() == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("daemon create/attach never completed")
}

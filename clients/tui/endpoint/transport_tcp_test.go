package endpoint

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/pty"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
)

// tcpTestManager registers endpoints through the default dialTransport, which
// is the production path: ConnectMode tcp dials the framed transport.
func tcpTestManager(t *testing.T) *Manager {
	t.Helper()
	m := NewManager(Options{
		DialTimeout: time.Second,
		CallTimeout: time.Second,
		BackoffMin:  20 * time.Millisecond,
		BackoffMax:  60 * time.Millisecond,
		Dial:        dialRawSession,
	})
	t.Cleanup(func() { _ = m.Close() })
	return m
}

// startUnixTCPBridge forwards TCP connections byte for byte to one unix
// socket, standing in for `ssh -L 127.0.0.1:PORT:/remote/daemon.sock` or a
// socat bridge in front of a remote daemon.
func startUnixTCPBridge(t *testing.T, unixPath string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen bridge: %v", err)
	}
	var wg sync.WaitGroup
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer conn.Close()
				upstream, err := net.Dial("unix", unixPath)
				if err != nil {
					return
				}
				defer upstream.Close()
				done := make(chan struct{})
				go func() {
					_, _ = io.Copy(upstream, conn)
					close(done)
				}()
				_, _ = io.Copy(conn, upstream)
				<-done
			}()
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		wg.Wait()
	})
	return ln.Addr().String()
}

// TestTCPFramingMatchesSharedUnixTransport proves the tui2 tcp transport is
// byte compatible with the production shared/transport/unix framing: a unix
// listener from the shared package sits behind a TCP bridge, and frames of
// every size (small, exact packet limit, fragmented) cross both ways.
func TestTCPFramingMatchesSharedUnixTransport(t *testing.T) {
	listener, err := unixtransport.NewListener(t.TempDir() + "/shared.sock")
	if err != nil {
		t.Fatalf("shared unix listener: %v", err)
	}
	defer listener.Close()
	address := startUnixTCPBridge(t, listener.Addr())

	serverErr := make(chan error, 1)
	serverFrames := make(chan string, 2)
	go func() {
		server, err := listener.Accept(context.Background())
		if err != nil {
			serverErr <- err
			return
		}
		defer server.Close()
		small := []byte("shared-small")
		big := bytes.Repeat([]byte("fragment-"), 9000) // ~72KiB > 64KiB packet cap
		if err := server.Send(small); err != nil {
			serverErr <- err
			return
		}
		if err := server.Send(big); err != nil {
			serverErr <- err
			return
		}
		serverFrames <- string(small)
		serverFrames <- string(big)
		reply, err := server.Recv()
		if err != nil {
			serverErr <- err
			return
		}
		if string(reply) != "client-payload" {
			serverErr <- fmt.Errorf("server received %q", reply)
			return
		}
		serverErr <- nil
	}()

	client, err := dialTCPTransport(context.Background(), address)
	if err != nil {
		t.Fatalf("dial tcp transport: %v", err)
	}
	defer client.Close()

	for i := 0; i < 2; i++ {
		want := <-serverFrames
		got, err := client.Recv()
		if err != nil {
			t.Fatalf("client recv %d: %v", i, err)
		}
		if string(got) != want {
			t.Fatalf("frame %d length = %d, want %d", i, len(got), len(want))
		}
	}
	if err := client.Send([]byte("client-payload")); err != nil {
		t.Fatalf("client send: %v", err)
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("server side: %v", err)
	}
}

// TestTCPEndpointLifecycle runs the full manager/RemotePTY path over tcp:
// health, list, attach, input, resize-owner CAS, reconnect resubscribe and
// snapshot re-seed, then kill.
func TestTCPEndpointLifecycle(t *testing.T) {
	d := newTCPFakeDaemon(t)
	m := tcpTestManager(t)
	var noticeMu sync.Mutex
	var notices []string
	m.SetOnNotice(func(level, message string) {
		noticeMu.Lock()
		notices = append(notices, level+":"+message)
		noticeMu.Unlock()
	})
	if err := m.Register(Config{Name: "remote", Kind: KindDaemon, ConnectMode: ConnectDirectTCP, Address: d.address}); err != nil {
		t.Fatalf("register tcp endpoint: %v", err)
	}
	waitFor(t, 3*time.Second, func() bool { return m.Health("remote") == HealthOK }, "tcp endpoint health ok")
	if kind, _ := m.Kind("remote"); kind != KindDaemon {
		t.Fatalf("kind = %q, want daemon", kind)
	}

	id := createTerminal(t, m, d, "remote")
	p := m.NewRemotePTY(pty.Config{Endpoint: "remote", ID: id, Cols: 80, Rows: 24})
	if err := p.Start(); err != nil {
		t.Fatalf("remote pty start over tcp: %v", err)
	}
	reader := startReader(p)
	if _, err := p.Write([]byte("echo TCP-OK\r")); err != nil {
		t.Fatalf("write: %v", err)
	}
	waitText(t, reader, "TCP-OK")

	if _, err := p.Write([]byte("stty size\r")); err != nil {
		t.Fatalf("write stty: %v", err)
	}
	waitText(t, reader, "24 80")
	if err := p.Resize(100, 30); err != nil {
		t.Fatalf("resize over tcp: %v", err)
	}
	waitFor(t, time.Second, func() bool {
		size := d.terminal(id).sizeProto()
		return size.GetCols() == 100 && size.GetRows() == 30
	}, "daemon resize over tcp")
	if _, err := p.Write([]byte("stty size\r")); err != nil {
		t.Fatalf("write stty after resize: %v", err)
	}
	waitText(t, reader, "30 100")

	// A dropped tcp connection flips health offline and the supervisor
	// reconnects, re-subscribes and re-seeds the snapshot.
	d.disconnect()
	waitFor(t, 3*time.Second, func() bool { return m.Health("remote") == HealthOffline }, "tcp health offline")
	waitFor(t, 3*time.Second, func() bool { return m.Health("remote") == HealthOK }, "tcp health back online")
	noticeMu.Lock()
	joined := strings.Join(notices, "\n")
	noticeMu.Unlock()
	if !strings.Contains(joined, "remote offline") || !strings.Contains(joined, "remote connected") {
		t.Fatalf("notices = %q", joined)
	}
	if p.Exited() {
		t.Fatal("remote pty must survive a tcp reconnect")
	}
	waitFor(t, 3*time.Second, func() bool { return strings.Count(reader.text(), "TCP-OK") >= 2 }, "snapshot re-seed over tcp")
	if _, err := p.Write([]byte("echo TCP-AGAIN\r")); err != nil {
		t.Fatalf("write after reconnect: %v", err)
	}
	waitText(t, reader, "TCP-AGAIN")

	if err := m.Kill(context.Background(), "remote", id); err != nil {
		t.Fatalf("kill over tcp: %v", err)
	}
	waitFor(t, 3*time.Second, func() bool { return p.Exited() }, "remote pty exit over tcp")
}

// TestTCPDialErrorsAreReadable checks config validation and the dial error
// surface: a missing address is a config error, a refused address is an
// offline health notice naming the target.
func TestTCPDialErrorsAreReadable(t *testing.T) {
	if err := (Config{Name: "tcp", Kind: KindDaemon, ConnectMode: ConnectDirectTCP}).Validate(); err == nil || !strings.Contains(err.Error(), "requires an address") {
		t.Fatalf("missing address error = %v", err)
	}
	if err := (Config{Name: "tcp", Kind: KindDaemon, ConnectMode: ConnectDirectTCP, Address: "127.0.0.1"}).Validate(); err == nil || !strings.Contains(err.Error(), "HOST:PORT") {
		t.Fatalf("bad address error = %v", err)
	}

	closed, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	address := closed.Addr().String()
	_ = closed.Close()

	m := tcpTestManager(t)
	var noticeMu sync.Mutex
	var notices []string
	m.SetOnNotice(func(level, message string) {
		noticeMu.Lock()
		notices = append(notices, message)
		noticeMu.Unlock()
	})
	if err := m.Register(Config{Name: "remote", Kind: KindDaemon, ConnectMode: ConnectDirectTCP, Address: address}); err != nil {
		t.Fatalf("register refused endpoint: %v", err)
	}
	waitFor(t, 3*time.Second, func() bool { return m.Health("remote") == HealthOffline }, "refused endpoint offline")
	waitFor(t, 3*time.Second, func() bool {
		noticeMu.Lock()
		defer noticeMu.Unlock()
		return len(notices) > 0
	}, "offline notice")
	noticeMu.Lock()
	message := strings.Join(notices, "\n")
	noticeMu.Unlock()
	if !strings.Contains(message, "dial "+address) || !strings.Contains(message, "refused") {
		t.Fatalf("offline notice = %q", message)
	}
}

// TestTCPConnectModeSurface pins the tcp config surface used by the picker
// and dialer: readable target, dialable mode, and every recognized shared
// mode accepted (missing route params fail later with a readable error).
func TestTCPConnectModeSurface(t *testing.T) {
	cfg := Config{Name: "remote", Kind: KindDaemon, ConnectMode: ConnectDirectTCP, Address: "127.0.0.1:17777"}
	if got := cfg.DialTarget(); got != "127.0.0.1:17777" {
		t.Fatalf("DialTarget = %q", got)
	}
	if err := cfg.UnsupportedModeError(); err != nil {
		t.Fatalf("tcp UnsupportedModeError = %v", err)
	}
	webrtc := Config{Name: "webrtc", Kind: KindDaemon, ConnectMode: ConnectDirectWebRTC}
	if err := webrtc.UnsupportedModeError(); err != nil {
		t.Fatalf("webrtc UnsupportedModeError = %v; want nil (shared layer owns direct/cloud)", err)
	}
	if _, err := endpointFromConfig(webrtc, ""); err == nil || !strings.Contains(err.Error(), "signaling") {
		t.Fatalf("webrtc endpointFromConfig error = %v; want readable missing-route error", err)
	}
}

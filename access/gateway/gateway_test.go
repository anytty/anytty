package gateway_test

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/access/gateway"
	"github.com/anytty/anytty/access/provider"
)

const testTimeout = 5 * time.Second

// startEchoServer accepts unix connections and echoes every byte back.
func startEchoServer(t *testing.T, socket string) {
	t.Helper()
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("listen echo server: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}()
		}
	}()
}

// startFinalReplyServer reads until the client half-closes, then replies once
// and closes. It proves the gateway propagates FIN in both directions.
func startFinalReplyServer(t *testing.T, socket string) {
	t.Helper()
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("listen final-reply server: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				payload, _ := io.ReadAll(conn)
				_, _ = conn.Write(append([]byte("reply:"), payload...))
			}()
		}
	}()
}

func unixProvider(socket string) provider.SessionProvider {
	return provider.DialFunc(func(ctx context.Context) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	})
}

func startGateway(t *testing.T, cfg gateway.Config) *gateway.Gateway {
	t.Helper()
	gw, err := gateway.New(cfg)
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- gw.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		_ = gw.Close()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("gateway serve: %v", err)
			}
		case <-time.After(testTimeout):
			t.Error("gateway did not stop")
		}
	})
	if err := gw.WaitReady(ctx); err != nil {
		t.Fatalf("gateway not ready: %v", err)
	}
	return gw
}

func tcpSpec() gateway.ListenerSpec {
	return gateway.ListenerSpec{Network: "tcp", Address: "127.0.0.1:0"}
}

func dialTCP(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, testTimeout)
	if err != nil {
		t.Fatalf("dial gateway %s: %v", addr, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func readExact(t *testing.T, reader io.Reader, size int) []byte {
	t.Helper()
	buf := make([]byte, size)
	if _, err := io.ReadFull(reader, buf); err != nil {
		t.Fatalf("read %d bytes: %v", size, err)
	}
	return buf
}

func TestGatewayProxiesBytesBothDirections(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "daemon.sock")
	startEchoServer(t, socket)
	gw := startGateway(t, gateway.Config{Provider: unixProvider(socket), Listeners: []gateway.ListenerSpec{tcpSpec()}})
	conn := dialTCP(t, gw.Addrs()[0])
	reader := bufio.NewReader(conn)

	if _, err := conn.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if got := readExact(t, reader, len("hello")); string(got) != "hello" {
		t.Fatalf("first roundtrip = %q", got)
	}
	if _, err := conn.Write([]byte("world")); err != nil {
		t.Fatal(err)
	}
	if got := readExact(t, reader, len("world")); string(got) != "world" {
		t.Fatalf("second roundtrip = %q", got)
	}
}

func TestGatewayPropagatesHalfClose(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "daemon.sock")
	startFinalReplyServer(t, socket)
	gw := startGateway(t, gateway.Config{Provider: unixProvider(socket), Listeners: []gateway.ListenerSpec{tcpSpec()}})
	conn := dialTCP(t, gw.Addrs()[0])

	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	if err := conn.(*net.TCPConn).CloseWrite(); err != nil {
		t.Fatalf("client CloseWrite: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(testTimeout))
	reply, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("read reply after half-close: %v", err)
	}
	if string(reply) != "reply:ping" {
		t.Fatalf("half-close reply = %q, want reply:ping", reply)
	}
}

func TestGatewayClosesUpstreamWhenClientDisconnects(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "daemon.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	clientEOF := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			clientEOF <- err
			return
		}
		defer conn.Close()
		buf := make([]byte, 32)
		for {
			if _, err := conn.Read(buf); err != nil {
				clientEOF <- err
				return
			}
		}
	}()
	gw := startGateway(t, gateway.Config{Provider: unixProvider(socket), Listeners: []gateway.ListenerSpec{tcpSpec()}})
	conn := dialTCP(t, gw.Addrs()[0])
	if _, err := conn.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	select {
	case err := <-clientEOF:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("upstream read error = %v, want EOF", err)
		}
	case <-time.After(testTimeout):
		t.Fatal("upstream did not observe client disconnect")
	}
}

func TestGatewayUnixListenerIsOwnerOnly(t *testing.T) {
	daemonSocket := filepath.Join(t.TempDir(), "daemon.sock")
	startEchoServer(t, daemonSocket)
	proxySocket := filepath.Join(t.TempDir(), "access.sock")
	gw := startGateway(t, gateway.Config{
		Provider:  unixProvider(daemonSocket),
		Listeners: []gateway.ListenerSpec{{Network: "unix", Address: proxySocket}},
	})
	info, err := os.Stat(proxySocket)
	if err != nil {
		t.Fatalf("stat gateway socket: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("gateway socket mode = %o, want 600", perm)
	}
	conn, err := net.Dial("unix", gw.Addrs()[0])
	if err != nil {
		t.Fatalf("dial gateway unix socket: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("unix-path")); err != nil {
		t.Fatal(err)
	}
	if got := readExact(t, conn, len("unix-path")); string(got) != "unix-path" {
		t.Fatalf("unix listener roundtrip = %q", got)
	}
}

func TestGatewayRefusesLiveUnixSocket(t *testing.T) {
	daemonSocket := filepath.Join(t.TempDir(), "daemon.sock")
	startEchoServer(t, daemonSocket)
	live := filepath.Join(t.TempDir(), "live.sock")
	listener, err := net.Listen("unix", live)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if _, err := gateway.New(gateway.Config{
		Provider:  unixProvider(daemonSocket),
		Listeners: []gateway.ListenerSpec{{Network: "unix", Address: live}},
	}); err != nil {
		t.Fatalf("config rejected: %v", err)
	}
	spec := gateway.ListenerSpec{Network: "unix", Address: live}
	if _, err := spec.Listen(); err == nil || !strings.Contains(err.Error(), "active listener") {
		t.Fatalf("live socket bind error = %v, want active-listener refusal", err)
	}
}

func TestGatewayPairTokenHandshake(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "daemon.sock")
	startEchoServer(t, socket)
	token := []byte("s3cret-token")
	gw := startGateway(t, gateway.Config{
		Provider:  unixProvider(socket),
		Listeners: []gateway.ListenerSpec{tcpSpec()},
		PairToken: token,
	})
	addr := gw.Addrs()[0]

	t.Run("correct token with pipelined payload", func(t *testing.T) {
		conn := dialTCP(t, addr)
		reader := bufio.NewReader(conn)
		if _, err := conn.Write([]byte("ANYTTY-PAIR s3cret-token\npayload")); err != nil {
			t.Fatal(err)
		}
		line, err := reader.ReadString('\n')
		if err != nil || line != "ANYTTY-PAIR OK\n" {
			t.Fatalf("ack = %q err=%v", line, err)
		}
		if got := readExact(t, reader, len("payload")); string(got) != "payload" {
			t.Fatalf("pipelined payload = %q", got)
		}
	})

	t.Run("wrong token is denied", func(t *testing.T) {
		conn := dialTCP(t, addr)
		reader := bufio.NewReader(conn)
		if _, err := conn.Write([]byte("ANYTTY-PAIR wrong\n")); err != nil {
			t.Fatal(err)
		}
		line, err := reader.ReadString('\n')
		if err != nil || line != "ANYTTY-PAIR DENIED\n" {
			t.Fatalf("denied ack = %q err=%v", line, err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(testTimeout))
		if _, err := reader.ReadByte(); !errors.Is(err, io.EOF) {
			t.Fatalf("denied connection error = %v, want EOF", err)
		}
	})

	t.Run("missing token is denied", func(t *testing.T) {
		conn := dialTCP(t, addr)
		if _, err := conn.Write([]byte("GET / HTTP/1.0\r\n\r\n")); err != nil {
			t.Fatal(err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(testTimeout))
		reply, _ := io.ReadAll(conn)
		if string(reply) != "ANYTTY-PAIR DENIED\n" {
			t.Fatalf("missing-token reply = %q", reply)
		}
	})
}

func TestGatewayAllowListRejectsUnlistedPeer(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "daemon.sock")
	startEchoServer(t, socket)
	allow := mustNetworks(t, "10.0.0.0/8")
	gw := startGateway(t, gateway.Config{Provider: unixProvider(socket), Listeners: []gateway.ListenerSpec{tcpSpec()}, Allow: allow})
	conn := dialTCP(t, gw.Addrs()[0])
	_ = conn.SetReadDeadline(time.Now().Add(testTimeout))
	if _, err := conn.Write([]byte("hi")); err != nil {
		t.Fatal(err)
	}
	if reply, _ := io.ReadAll(conn); len(reply) != 0 {
		t.Fatalf("unlisted peer received %q, want closed connection", reply)
	}

	allowed := mustNetworks(t, "127.0.0.1/32")
	gwAllowed := startGateway(t, gateway.Config{Provider: unixProvider(socket), Listeners: []gateway.ListenerSpec{tcpSpec()}, Allow: allowed})
	connAllowed := dialTCP(t, gwAllowed.Addrs()[0])
	if _, err := connAllowed.Write([]byte("ok")); err != nil {
		t.Fatal(err)
	}
	if got := readExact(t, connAllowed, len("ok")); string(got) != "ok" {
		t.Fatalf("allow-listed peer roundtrip = %q", got)
	}
}

func TestGatewayProviderFailureClosesClient(t *testing.T) {
	failing := provider.DialFunc(func(context.Context) (net.Conn, error) {
		return nil, errors.New("daemon unavailable")
	})
	gw := startGateway(t, gateway.Config{Provider: failing, Listeners: []gateway.ListenerSpec{tcpSpec()}})
	conn := dialTCP(t, gw.Addrs()[0])
	_ = conn.SetReadDeadline(time.Now().Add(testTimeout))
	reply, _ := io.ReadAll(conn)
	if len(reply) != 0 {
		t.Fatalf("provider failure delivered %q, want closed connection", reply)
	}
}

func TestGatewayCloseUnblocksActiveConnections(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "daemon.sock")
	startEchoServer(t, socket)
	gw, err := gateway.New(gateway.Config{Provider: unixProvider(socket), Listeners: []gateway.ListenerSpec{tcpSpec()}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- gw.Serve(ctx) }()
	if err := gw.WaitReady(ctx); err != nil {
		t.Fatal(err)
	}
	conn := dialTCP(t, gw.Addrs()[0])
	if _, err := conn.Write([]byte("hold")); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(conn, make([]byte, 4)); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gateway: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(testTimeout))
	if _, err := conn.Read(make([]byte, 1)); !errors.Is(err, io.EOF) && err == nil {
		t.Fatalf("active connection survived gateway close: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve after close: %v", err)
		}
	case <-time.After(testTimeout):
		t.Fatal("serve did not return after close")
	}
}

func TestParseListenerSpec(t *testing.T) {
	valid := []struct {
		value   string
		network string
		address string
	}{
		{"tcp:127.0.0.1:7331", "tcp", "127.0.0.1:7331"},
		{"unix:/tmp/access.sock", "unix", "/tmp/access.sock"},
		{"127.0.0.1:7331", "tcp", "127.0.0.1:7331"},
	}
	for _, test := range valid {
		spec, err := gateway.ParseListenerSpec(test.value)
		if err != nil || spec.Network != test.network || spec.Address != test.address {
			t.Fatalf("ParseListenerSpec(%q) = %+v err=%v", test.value, spec, err)
		}
	}
	for _, value := range []string{"", "tcp:", "unix:", "udp:127.0.0.1:1", "tcp:not-a-host-port"} {
		if _, err := gateway.ParseListenerSpec(value); err == nil {
			t.Fatalf("ParseListenerSpec(%q) unexpectedly succeeded", value)
		}
	}
}

func TestParseNetworks(t *testing.T) {
	networks, err := gateway.ParseNetworks([]string{"127.0.0.1", "10.0.0.0/8", "::1", "", " ", "fd00::/8"})
	if err != nil || len(networks) != 4 {
		t.Fatalf("ParseNetworks = %v err=%v", networks, err)
	}
	if !networks[0].Contains(net.ParseIP("127.0.0.1")) {
		t.Fatal("bare IPv4 host was not treated as a single host")
	}
	if !networks[2].Contains(net.ParseIP("::1")) {
		t.Fatal("bare IPv6 host was not treated as a single host")
	}
	if _, err := gateway.ParseNetworks([]string{"not-an-ip"}); err == nil {
		t.Fatal("invalid allow entry unexpectedly parsed")
	}
}

func mustNetworks(t *testing.T, values ...string) []*net.IPNet {
	t.Helper()
	networks, err := gateway.ParseNetworks(values)
	if err != nil {
		t.Fatal(err)
	}
	return networks
}

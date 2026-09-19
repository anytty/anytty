package gateway_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	accesscontract "github.com/anytty/anytty/access/contract"
	localadapter "github.com/anytty/anytty/access/engine/adapter/local"
	protocoladapter "github.com/anytty/anytty/access/engine/adapter/protocol"
	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	"github.com/anytty/anytty/access/gateway"
	daemonprovider "github.com/anytty/anytty/access/provider/daemon"
	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	accessserver "github.com/anytty/anytty/access/server"
	core "github.com/anytty/anytty/daemon/core"
	providercore "github.com/anytty/anytty/daemon/provider"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
	"github.com/anytty/anytty/shared/remoteauth"
)

// e2eTimeout bounds one engine operation in the gateway acceptance suite. A
// plain daemon start plus PTY spawn is fast on CI, but -race needs headroom.
const e2eTimeout = 20 * time.Second

// testDaemon is one real daemon core with an identity service so the engine's
// local adapter can complete its Hello and identity proof exactly like it does
// against anyttyd.
type testDaemon struct {
	socket string
	server *core.Server
	cancel context.CancelFunc
	done   chan error
}

func startTestDaemon(t *testing.T) *testDaemon {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := remoteauth.NewIdentity("anytty-access-e2e", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	providerSocket := filepath.Join(dir, "anytty-v2-wire7.sock.provider")
	accessSocket := filepath.Join(dir, "anytty-v2-wire7.sock")
	server := core.NewServer(
		core.WithSocketPath(providerSocket),
		core.WithHistoryDisabled(),
	)
	providerServer, err := providercore.New(server, providercore.Config{Socket: providerSocket})
	if err != nil {
		t.Fatal(err)
	}
	access, err := accessserver.New(accessserver.Config{
		Socket: accessSocket,
		Auth:   &accessserver.AuthServices{Access: testAccessService{identity: identity}},
		Provider: func(dialCtx context.Context) (terminalprovider.Provider, error) {
			return daemonprovider.DialTerminal(dialCtx, providerSocket)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- providerServer.ListenAndServe(ctx) }()
	go func() { _ = access.Serve(ctx) }()
	deadline := time.Now().Add(e2eTimeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("unix", accessSocket, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(accessSocket); err != nil {
		cancel()
		t.Fatalf("access socket never appeared: %v", err)
	}
	daemon := &testDaemon{socket: accessSocket, server: server, cancel: cancel, done: done}
	t.Cleanup(func() {
		cancel()
		_ = access.Close()
		_ = providerServer.Shutdown(context.Background())
		_ = server.Shutdown(context.Background())
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("daemon shutdown: %v", err)
			}
		case <-time.After(e2eTimeout):
			t.Error("daemon did not stop")
		}
	})
	return daemon
}

type testAccessService struct {
	identity remoteauth.Identity
}

func (service testAccessService) Identity(_ context.Context, challenge []byte) (accesscontract.ClientAccessIdentity, error) {
	proof, err := remoteauth.SignDeviceIdentityProof(service.identity, challenge)
	if err != nil {
		return accesscontract.ClientAccessIdentity{}, err
	}
	return accesscontract.ClientAccessIdentity{
		DeviceID: service.identity.DeviceID, DeviceFingerprint: service.identity.Fingerprint,
		DevicePublicKey: append([]byte(nil), service.identity.PublicKey...),
		Challenge:       append([]byte(nil), challenge...),
		Proof:           proof,
	}, nil
}

func (testAccessService) CreateTicket(context.Context, accesscontract.ClientAccessTicketRequest) (accesscontract.ClientAccessTicket, error) {
	return accesscontract.ClientAccessTicket{}, errors.New("pairing tickets are not part of this acceptance")
}

func (testAccessService) List(context.Context) ([]accesscontract.ClientAccessRecord, error) {
	return nil, nil
}

func (testAccessService) GrantActive(context.Context, string, time.Time, time.Time) bool {
	return false
}

func (testAccessService) Revoke(context.Context, string) (accesscontract.ClientAccessRecord, error) {
	return accesscontract.ClientAccessRecord{}, errors.New("revocation is not part of this acceptance")
}

// gatewayFixture is one daemon plus one running gateway and the client-side
// unix relay that stands in for the shipped ssh -L / TUI tcp bridge.
type gatewayFixture struct {
	daemon  *testDaemon
	gateway *gateway.Gateway
	address string
	relay   string
	token   []byte
}

func newGatewayFixture(t *testing.T, token []byte) *gatewayFixture {
	t.Helper()
	daemon := startTestDaemon(t)
	fixture := &gatewayFixture{daemon: daemon, token: token}
	fixture.startGateway(t, tcpSpec())
	fixture.relay = startTCPRelay(t, fixture.address, token)
	return fixture
}

func (fixture *gatewayFixture) startGateway(t *testing.T, specs ...gateway.ListenerSpec) {
	t.Helper()
	gw, err := gateway.New(gateway.Config{
		Provider:  daemonprovider.New(fixture.daemon.socket),
		Listeners: specs,
		PairToken: fixture.token,
	})
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- gw.Serve(ctx) }()
	if err := gw.WaitReady(ctx); err != nil {
		cancel()
		t.Fatalf("gateway not ready: %v", err)
	}
	fixture.gateway = gw
	if specs[0].Network == "tcp" {
		fixture.address = gw.Addrs()[0]
	}
	t.Cleanup(func() {
		cancel()
		_ = gw.Close()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("gateway serve: %v", err)
			}
		case <-time.After(e2eTimeout):
			t.Error("gateway did not stop")
		}
	})
}

// startTCPRelay mirrors the shipped client-side tcp compatibility bridge: a
// private unix socket byte-relays to the gateway TCP listener, optionally
// speaking the pair-token pre-handshake first.
func startTCPRelay(t *testing.T, address string, token []byte) string {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "gateway-relay.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("listen relay: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			client, err := listener.Accept()
			if err != nil {
				return
			}
			go relayConnection(client, address, token)
		}
	}()
	return socket
}

func relayConnection(client net.Conn, address string, token []byte) {
	defer client.Close()
	upstream, err := net.DialTimeout("tcp", address, e2eTimeout)
	if err != nil {
		return
	}
	defer upstream.Close()
	source := io.Reader(upstream)
	if len(token) > 0 {
		_ = upstream.SetDeadline(time.Now().Add(e2eTimeout))
		if _, err := upstream.Write([]byte("ANYTTY-PAIR " + string(token) + "\n")); err != nil {
			return
		}
		reader := bufio.NewReader(upstream)
		line, err := reader.ReadString('\n')
		if err != nil || line != "ANYTTY-PAIR OK\n" {
			return
		}
		_ = upstream.SetDeadline(time.Time{})
		source = reader
	}
	done := make(chan struct{}, 2)
	closeBoth := func() { _ = client.Close(); _ = upstream.Close() }
	go func() { _, _ = io.Copy(upstream, client); done <- struct{}{} }()
	go func() { _, _ = io.Copy(client, source); done <- struct{}{} }()
	<-done
	// Mirror the shipped tcp bridge: when either direction ends, close both
	// ends so the engine session observes the loss immediately.
	closeBoth()
	<-done
}

// connectEngine dials one unix socket through the shipped local route adapter
// (identity proof, Hello v7, application session) and returns the typed
// application client.
func connectEngine(t *testing.T, ctx context.Context, socket string) *protocoladapter.ApplicationClient {
	t.Helper()
	target := clientendpoint.Endpoint{
		ID: "gateway",
		Routes: map[clientendpoint.RouteID]clientendpoint.AccessRoute{
			"gateway": {
				ID: "gateway", Kind: clientendpoint.RouteLocalUnix, Enabled: true,
				Source: clientendpoint.SourceManual, PolicySource: clientendpoint.SourceUser, Socket: socket,
			},
		},
	}
	attempt, err := clientruntime.NewAttemptRequest(target, "gateway", clientruntime.SessionGeneration(1), clientruntime.ConnectIntentInteractive)
	if err != nil {
		t.Fatalf("build attempt: %v", err)
	}
	ready, err := localadapter.NewDialer(localadapter.Options{ClientName: "anytty-access-e2e"}).Connect(ctx, attempt)
	if err != nil {
		t.Fatalf("connect engine through %s: %v", socket, err)
	}
	app, ok := ready.(*protocoladapter.ApplicationClient)
	if !ok {
		t.Fatalf("local route returned %T, want *protocoladapter.ApplicationClient", ready)
	}
	t.Cleanup(func() { _ = app.Close() })
	return app
}

func createTerminal(t *testing.T, ctx context.Context, app *protocoladapter.ApplicationClient, id string, cols, rows uint32) *apipb.TerminalRef {
	t.Helper()
	result, err := app.TerminalCreate(ctx, &apipb.TerminalCreateCommand{Terminal: &apipb.TerminalCreateSpec{
		TerminalId: id, Name: id, Command: []string{"/bin/sh"},
		Size: &apipb.TerminalSize{Cols: cols, Rows: rows},
	}})
	if err != nil {
		t.Fatalf("create terminal %s: %v", id, err)
	}
	ref := result.GetTerminal().GetRef()
	if ref.GetTerminalId() != id {
		t.Fatalf("created ref = %+v", ref)
	}
	return ref
}

type attachment struct {
	resource *apipb.ResourceHandle
	stream   clientruntime.ResourceStream
}

func attachTerminal(t *testing.T, ctx context.Context, app *protocoladapter.ApplicationClient, ref *apipb.TerminalRef) *attachment {
	t.Helper()
	result, err := app.TerminalAttach(ctx, &apipb.TerminalAttachCommand{
		Terminal: ref, Mode: apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR,
		ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER, SurfaceId: "gateway-e2e", ViewId: "e2e",
	})
	if err != nil {
		t.Fatalf("attach %s: %v", ref.GetTerminalId(), err)
	}
	resource := result.GetAttachment().GetResource()
	stream, err := app.OpenResourceStream(resource)
	if err != nil {
		t.Fatalf("open attachment stream: %v", err)
	}
	att := &attachment{resource: resource, stream: stream}
	t.Cleanup(func() {
		// The session may already be gone (gateway stopped); detach must not
		// block cleanup on a dead connection.
		detachCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = app.TerminalDetach(detachCtx, &apipb.TerminalDetachCommand{Attachment: resource})
	})
	return att
}

func (att *attachment) input(t *testing.T, ctx context.Context, app *protocoladapter.ApplicationClient, data string) {
	t.Helper()
	if err := app.TerminalInput(ctx, &apipb.TerminalInputCommand{Attachment: att.resource, Data: []byte(data)}); err != nil {
		t.Fatalf("terminal input %q: %v", data, err)
	}
}

// expectOutput reads PTY frames until marker appears; the echoed command line
// is deliberately built so only execution (not echo) can contain the marker.
func (att *attachment) expectOutput(t *testing.T, ctx context.Context, marker string) string {
	t.Helper()
	deadlineCtx, cancel := context.WithTimeout(ctx, e2eTimeout)
	defer cancel()
	var collected bytes.Buffer
	for deadlineCtx.Err() == nil {
		typ, payload, err := att.stream.Receive(deadlineCtx)
		if err != nil {
			if deadlineCtx.Err() != nil {
				break
			}
			t.Fatalf("receive attachment frame: %v", err)
		}
		switch typ {
		case wire.TypePTYOutput:
			collected.Write(payload)
			if bytes.Contains(payload, []byte(marker)) {
				return collected.String()
			}
		case wire.TypeClosed:
			t.Fatalf("attachment stream closed before %q; output=%q", marker, collected.String())
		}
	}
	t.Fatalf("marker %q not observed; output=%q", marker, collected.String())
	return ""
}

func terminalState(t *testing.T, ctx context.Context, app *protocoladapter.ApplicationClient, ref *apipb.TerminalRef) *apipb.TerminalInfo {
	t.Helper()
	result, err := app.TerminalGet(ctx, &apipb.TerminalGetCommand{Terminal: ref})
	if err != nil {
		t.Fatalf("terminal get %s: %v", ref.GetTerminalId(), err)
	}
	return result.GetTerminal()
}

func waitTerminalState(t *testing.T, ctx context.Context, app *protocoladapter.ApplicationClient, ref *apipb.TerminalRef, want apipb.TerminalState) {
	t.Helper()
	deadline := time.Now().Add(e2eTimeout)
	for time.Now().Before(deadline) {
		if terminalState(t, ctx, app, ref).GetState() == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("terminal %s never reached %s", ref.GetTerminalId(), want)
}

// TestGatewayEngineAttachInputOutput is the M5 core acceptance: with the
// daemon reachable only through the gateway TCP listener, an existing engine
// session creates, attaches, types into and reads a terminal end to end.
func TestGatewayEngineAttachInputOutput(t *testing.T) {
	fixture := newGatewayFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), e2eTimeout)
	defer cancel()
	app := connectEngine(t, ctx, fixture.relay)
	ref := createTerminal(t, ctx, app, "gw-echo", 80, 24)
	att := attachTerminal(t, ctx, app, ref)
	att.input(t, ctx, app, "echo GW-OK-$(echo e2e)\r")
	output := att.expectOutput(t, ctx, "GW-OK-e2e")
	t.Logf("GW-OK output over gateway tcp: %q", output)
}

// TestGatewayEngineResize proves the resize command crosses the gateway and
// reaches the daemon terminal truth.
func TestGatewayEngineResize(t *testing.T) {
	fixture := newGatewayFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), e2eTimeout)
	defer cancel()
	app := connectEngine(t, ctx, fixture.relay)
	ref := createTerminal(t, ctx, app, "gw-resize", 80, 24)
	att := attachTerminal(t, ctx, app, ref)
	result, err := app.TerminalResize(ctx, &apipb.TerminalResizeCommand{
		Attachment: att.resource, Size: &apipb.TerminalSize{Cols: 100, Rows: 30},
		ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER,
	})
	if err != nil {
		t.Fatalf("resize through gateway: %v", err)
	}
	if got := result.GetSize(); got.GetCols() != 100 || got.GetRows() != 30 {
		t.Fatalf("resize result = %v, want 100x30", got)
	}
	info := terminalState(t, ctx, app, ref)
	if info.GetSize().GetCols() != 100 || info.GetSize().GetRows() != 30 {
		t.Fatalf("daemon terminal size = %v, want 100x30", info.GetSize())
	}
	att.input(t, ctx, app, "echo GW-OK-$(echo resize)\r")
	att.expectOutput(t, ctx, "GW-OK-resize")
}

// TestGatewayEngineKill proves kill crosses the gateway and stops the terminal.
func TestGatewayEngineKill(t *testing.T) {
	fixture := newGatewayFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), e2eTimeout)
	defer cancel()
	app := connectEngine(t, ctx, fixture.relay)
	ref := createTerminal(t, ctx, app, "gw-kill", 80, 24)
	att := attachTerminal(t, ctx, app, ref)
	att.input(t, ctx, app, "echo GW-OK-$(echo kill)\r")
	att.expectOutput(t, ctx, "GW-OK-kill")
	if err := app.TerminalKill(ctx, &apipb.TerminalKillCommand{Terminal: ref}); err != nil {
		t.Fatalf("kill through gateway: %v", err)
	}
	waitTerminalState(t, ctx, app, ref, apipb.TerminalState_TERMINAL_STATE_EXITED)
}

// TestGatewayRestartKeepsDaemonAndTerminal kills the gateway, proves the
// daemon and its terminal survive through a direct engine connection, then
// restarts the gateway and reattaches through the new TCP listener.
func TestGatewayRestartKeepsDaemonAndTerminal(t *testing.T) {
	fixture := newGatewayFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), e2eTimeout)
	defer cancel()

	app := connectEngine(t, ctx, fixture.relay)
	ref := createTerminal(t, ctx, app, "gw-survive", 80, 24)
	att := attachTerminal(t, ctx, app, ref)
	att.input(t, ctx, app, "echo GW-OK-$(echo before)\r")
	att.expectOutput(t, ctx, "GW-OK-before")

	if err := fixture.gateway.Close(); err != nil {
		t.Fatalf("close gateway: %v", err)
	}

	direct := connectEngine(t, ctx, fixture.daemon.socket)
	list, err := direct.TerminalList(ctx, &apipb.TerminalListCommand{})
	if err != nil {
		t.Fatalf("direct list after gateway close: %v", err)
	}
	found := false
	for _, terminal := range list.GetTerminals() {
		if terminal.GetRef().GetTerminalId() == "gw-survive" {
			found = true
			if terminal.GetState() != apipb.TerminalState_TERMINAL_STATE_RUNNING {
				t.Fatalf("terminal state after gateway close = %v, want running", terminal.GetState())
			}
		}
	}
	if !found {
		t.Fatal("terminal vanished when the gateway stopped")
	}
	directAtt := attachTerminal(t, ctx, direct, ref)
	directAtt.input(t, ctx, direct, "echo GW-OK-$(echo survive)\r")
	directAtt.expectOutput(t, ctx, "GW-OK-survive")

	fixture.startGateway(t, tcpSpec())
	restartedRelay := startTCPRelay(t, fixture.address, nil)
	restarted := connectEngine(t, ctx, restartedRelay)
	reattached := attachTerminal(t, ctx, restarted, ref)
	reattached.input(t, ctx, restarted, "echo GW-OK-$(echo restart)\r")
	reattached.expectOutput(t, ctx, "GW-OK-restart")
}

// TestDirectDaemonUnixPathUnaffected proves the local fast path still works
// while the gateway is stopped: the engine dials the daemon socket directly.
func TestDirectDaemonUnixPathUnaffected(t *testing.T) {
	daemon := startTestDaemon(t)
	ctx, cancel := context.WithTimeout(context.Background(), e2eTimeout)
	defer cancel()
	app := connectEngine(t, ctx, daemon.socket)
	ref := createTerminal(t, ctx, app, "direct-echo", 80, 24)
	att := attachTerminal(t, ctx, app, ref)
	att.input(t, ctx, app, "echo GW-OK-$(echo direct)\r")
	att.expectOutput(t, ctx, "GW-OK-direct")
}

// TestGatewayPairTokenAuthorizedAndRejected proves the optional pre-handshake
// is both enforced (readable denial) and usable by the existing engine via a
// client bridge that speaks it.
func TestGatewayPairTokenAuthorizedAndRejected(t *testing.T) {
	fixture := newGatewayFixture(t, []byte("e2e-pair-token"))
	ctx, cancel := context.WithTimeout(context.Background(), e2eTimeout)
	defer cancel()

	t.Run("wrong token denied", func(t *testing.T) {
		conn, err := net.DialTimeout("tcp", fixture.address, e2eTimeout)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		if _, err := conn.Write([]byte("ANYTTY-PAIR wrong-token\n")); err != nil {
			t.Fatal(err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(e2eTimeout))
		reply, _ := io.ReadAll(conn)
		if string(reply) != "ANYTTY-PAIR DENIED\n" {
			t.Fatalf("denied reply = %q", reply)
		}
	})
	t.Run("missing token denied", func(t *testing.T) {
		conn, err := net.DialTimeout("tcp", fixture.address, e2eTimeout)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		_ = conn.SetReadDeadline(time.Now().Add(e2eTimeout))
		if _, err := conn.Write([]byte("\x28\xb5\x2f\xfd\n")); err != nil {
			t.Fatal(err)
		}
		reply, _ := io.ReadAll(conn)
		if !strings.Contains(string(reply), "DENIED") {
			t.Fatalf("missing-token reply = %q", reply)
		}
	})
	t.Run("authorized engine roundtrip", func(t *testing.T) {
		app := connectEngine(t, ctx, fixture.relay)
		ref := createTerminal(t, ctx, app, "gw-token", 80, 24)
		att := attachTerminal(t, ctx, app, ref)
		att.input(t, ctx, app, "echo GW-OK-$(echo token)\r")
		att.expectOutput(t, ctx, "GW-OK-token")
	})
}

// TestGatewayMultipleListeners proves tcp and unix listeners serve the same
// daemon byte stream.
func TestGatewayMultipleListeners(t *testing.T) {
	daemon := startTestDaemon(t)
	unixSocket := filepath.Join(t.TempDir(), "gateway-access.sock")
	gw, err := gateway.New(gateway.Config{
		Provider: daemonprovider.New(daemon.socket),
		Listeners: []gateway.ListenerSpec{
			{Network: "tcp", Address: "127.0.0.1:0"},
			{Network: "unix", Address: unixSocket},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- gw.Serve(ctx) }()
	if err := gw.WaitReady(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		_ = gw.Close()
		select {
		case <-done:
		case <-time.After(e2eTimeout):
			t.Error("multi-listener gateway did not stop")
		}
	})
	ctx, cancelTimeout := context.WithTimeout(context.Background(), e2eTimeout)
	defer cancelTimeout()

	tcpApp := connectEngine(t, ctx, startTCPRelay(t, fmt.Sprintf("127.0.0.1:%d", portOf(t, gw.Addrs()[0])), nil))
	tcpRef := createTerminal(t, ctx, tcpApp, "multi-tcp", 80, 24)
	tcpAtt := attachTerminal(t, ctx, tcpApp, tcpRef)
	tcpAtt.input(t, ctx, tcpApp, "echo GW-OK-$(echo tcp)\r")
	tcpAtt.expectOutput(t, ctx, "GW-OK-tcp")

	unixApp := connectEngine(t, ctx, unixSocket)
	unixRef := createTerminal(t, ctx, unixApp, "multi-unix", 80, 24)
	unixAtt := attachTerminal(t, ctx, unixApp, unixRef)
	unixAtt.input(t, ctx, unixApp, "echo GW-OK-$(echo unix)\r")
	unixAtt.expectOutput(t, ctx, "GW-OK-unix")
}

// TestDaemonHasNoTCPListeners asserts the posture invariant on Linux: this
// process owns only the gateway's TCP listeners, so the in-process daemon
// contributes no network listener and cannot be reached by tcp except through
// the gateway.
func TestDaemonHasNoTCPListeners(t *testing.T) {
	_ = startTestDaemon(t)
	if len(processTCPListeners(t)) != 0 {
		t.Fatalf("daemon-only process owns tcp listeners: %v", processTCPListeners(t))
	}
	gw := func() *gateway.Gateway {
		gw, err := gateway.New(gateway.Config{
			Provider:  daemonprovider.New("unused-for-this-assertion"),
			Listeners: []gateway.ListenerSpec{tcpSpec()},
		})
		if err != nil {
			t.Fatal(err)
		}
		return gw
	}()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- gw.Serve(ctx) }()
	if err := gw.WaitReady(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		_ = gw.Close()
		<-done
	}()
	allowed := map[uint16]bool{}
	for _, address := range gw.Addrs() {
		allowed[portOf(t, address)] = true
	}
	listeners := processTCPListeners(t)
	if len(listeners) != len(allowed) {
		t.Fatalf("tcp listeners = %v, want exactly the gateway addresses %v", listeners, allowed)
	}
	for inode, port := range listeners {
		if !allowed[port] {
			t.Fatalf("unexpected tcp listener inode=%s port=%d; daemon exposed a network listener", inode, port)
		}
	}
}

// TestGatewayProviderDialUsesDaemonSocket proves a misconfigured daemon socket
// is a readable, contained failure: the client is closed, the gateway keeps
// serving other connections.
func TestGatewayProviderDialUsesDaemonSocket(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "missing-daemon.sock")
	gw := startGateway(t, gateway.Config{Provider: daemonprovider.New(socket), Listeners: []gateway.ListenerSpec{tcpSpec()}})
	conn := dialTCP(t, gw.Addrs()[0])
	_ = conn.SetReadDeadline(time.Now().Add(e2eTimeout))
	if _, err := io.ReadAll(conn); err != nil {
		t.Fatalf("client read after failed dial: %v", err)
	}
}

func portOf(t *testing.T, address string) uint16 {
	t.Helper()
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatalf("split %q: %v", address, err)
	}
	value, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		t.Fatalf("parse port %q: %v", port, err)
	}
	return uint16(value)
}

// processTCPListeners returns the inode -> port map of TCP sockets in LISTEN
// state owned by this process. It skips the test when /proc is unavailable
// (non-Linux) so other platforms keep running the rest of the suite.
func processTCPListeners(t *testing.T) map[string]uint16 {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Skipf("process fd table unavailable: %v", err)
	}
	owned := map[string]bool{}
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
		if err != nil {
			continue
		}
		if rest, ok := strings.CutPrefix(target, "socket:["); ok {
			owned[strings.TrimSuffix(rest, "]")] = true
		}
	}
	listeners := map[string]uint16{}
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		payload, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(payload), "\n")[1:] {
			fields := strings.Fields(line)
			if len(fields) < 10 || fields[3] != "0A" {
				continue
			}
			inode := fields[9]
			if !owned[inode] {
				continue
			}
			_, portHex, ok := strings.Cut(fields[1], ":")
			if !ok {
				continue
			}
			port, err := strconv.ParseUint(portHex, 16, 16)
			if err != nil {
				continue
			}
			listeners[inode] = uint16(port)
		}
	}
	return listeners
}

package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport/memory"
)

func TestBrowserProxyStreamsLargeUploadToRealTCPTarget(t *testing.T) {
	const totalBytes = 32 << 20
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	targetDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			targetDone <- err
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		digest := sha256.New()
		if _, err := io.CopyN(digest, conn, totalBytes); err != nil {
			targetDone <- err
			return
		}
		_, err = conn.Write(digest.Sum(nil))
		targetDone <- err
	}()
	client, server := memory.NewPair()
	defer client.Close()
	defer server.Close()
	session := newProtocolSession(NewServer(), server, fullDaemonTransportScope())
	defer session.releaseAllBrowserProxies()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resource, err := session.ApplicationBrowserProxyOpen(ctx, "127.0.0.1", uint16(listener.Addr().(*net.TCPAddr).Port))
	if err != nil {
		t.Fatal(err)
	}
	channel := uint16(resource.Token[0])<<8 | uint16(resource.Token[1])
	chunk := bytes.Repeat([]byte{0x5a}, 32<<10)
	want := sha256.New()
	for sent := 0; sent < totalBytes; sent += len(chunk) {
		_, _ = want.Write(chunk)
		if err := session.handleStreamFrame(ctx, channel, wire.TypeBrowserData, chunk); err != nil {
			t.Fatal(err)
		}
	}
	var response []byte
	for {
		gotChannel, typ, payload := receiveProtocolFrame(t, client)
		if gotChannel != channel {
			t.Fatalf("wrong response channel %d", gotChannel)
		}
		if typ == wire.TypeBrowserClosed {
			break
		}
		if typ != wire.TypeBrowserData {
			t.Fatalf("unexpected response type %d", typ)
		}
		response = append(response, payload...)
	}
	if !bytes.Equal(response, want.Sum(nil)) {
		t.Fatal("32 MiB upload checksum or final response was corrupted")
	}
	select {
	case err := <-targetDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("TCP target did not finish")
	}
	if err := session.handleStreamFrame(ctx, channel, wire.TypeClosed, nil); err != nil {
		t.Fatalf("late close failed: %v", err)
	}
}

type shortBrowserDeadlineConn struct {
	net.Conn
	requested time.Time
}

func (conn *shortBrowserDeadlineConn) SetWriteDeadline(deadline time.Time) error {
	conn.requested = deadline
	return conn.Conn.SetWriteDeadline(time.Now().Add(20 * time.Millisecond))
}

func TestBrowserProxyTargetWriteHasBoundedDeadline(t *testing.T) {
	local, remote := net.Pipe()
	defer local.Close()
	defer remote.Close()
	conn := &shortBrowserDeadlineConn{Conn: local}
	start := time.Now()
	written, err := writeBrowserProxyData(conn, []byte("upload"))
	var timeout net.Error
	if written != 0 || !errors.As(err, &timeout) || !timeout.Timeout() {
		t.Fatalf("blocked write = %d, %v", written, err)
	}
	if duration := conn.requested.Sub(start); duration < browserProxyWriteTimeout || duration > browserProxyWriteTimeout+time.Second {
		t.Fatalf("unexpected production write deadline: %s", duration)
	}
}

func TestBrowserProxyWriteFailureOnlyClosesItsResource(t *testing.T) {
	client, server := memory.NewPair()
	defer client.Close()
	defer server.Close()
	session := newProtocolSession(NewServer(), server, fullDaemonTransportScope())
	channel, err := session.reserveBrowserChannel()
	if err != nil {
		t.Fatal(err)
	}
	local, remote := net.Pipe()
	_ = remote.Close()
	proxy := &sessionBrowserProxy{channel: channel, conn: local}
	session.browserChannels[channel] = proxy
	if err := session.handleStreamFrame(context.Background(), channel, wire.TypeBrowserData, []byte("upload")); err != nil {
		t.Fatalf("website failure escaped into protocol session: %v", err)
	}
	gotChannel, typ, _ := receiveProtocolFrame(t, client)
	if gotChannel != channel || typ != wire.TypeBrowserClosed {
		t.Fatalf("resource closure = channel %d type %d", gotChannel, typ)
	}
	if session.browserCount != 0 || session.browserProxyForChannel(channel) != nil {
		t.Fatal("failed browser resource was not released")
	}
	for _, typ := range []uint8{wire.TypeBrowserData, wire.TypeClosed} {
		if err := session.handleStreamFrame(context.Background(), channel, typ, nil); err != nil {
			t.Fatalf("late browser frame terminated session: %v", err)
		}
	}
	if err := session.handleStreamFrame(context.Background(), channel+1, wire.TypeBrowserData, nil); err == nil {
		t.Fatal("unknown channel was mistaken for a closed browser resource")
	}
	if err := session.handleStreamFrame(context.Background(), channel, wire.TypeClosed, []byte("invalid")); err == nil {
		t.Fatal("malformed close was accepted")
	}
	if err := session.sendFrame(0, wire.TypeResponse, []byte("still-connected")); err != nil {
		t.Fatal(err)
	}
	_, typ, payload := receiveProtocolFrame(t, client)
	if typ != wire.TypeResponse || string(payload) != "still-connected" {
		t.Fatal("other protocol traffic stopped")
	}
}

func TestBrowserProxyResourceDialsFromDaemonAndReturnsRemoteBytes(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan error, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			accepted <- acceptErr
			return
		}
		defer conn.Close()
		_, copyErr := io.Copy(conn, conn)
		accepted <- copyErr
	}()

	server := NewServer(WithApplicationExecutorFactory(browserProxyTestExecutorFactory))
	clientTransport, serverTransport := memory.NewPair()
	done := make(chan error, 1)
	go func() {
		done <- newProtocolSession(server, serverTransport, fullDaemonTransportScope()).run(context.Background())
	}()
	completeServerProtocolHello(t, clientTransport)

	stamp := &apipb.EndpointSessionStamp{EndpointId: "endpoint-a", RouteId: "local", Generation: 1}
	commandPayload, err := protocol.EncodeApplicationCommand(&apipb.CommandEnvelope{
		Context: &apipb.RequestContext{
			RequestId:  "browser-open-1",
			ApiVersion: &apipb.ApiVersion{Major: 1},
			Session:    stamp,
		},
		Command: &apipb.CommandEnvelope_BrowserProxyOpen{
			BrowserProxyOpen: &apipb.BrowserProxyOpenCommand{
				Host: "127.0.0.1",
				Port: uint32(listener.Addr().(*net.TCPAddr).Port),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	sendProtocolRequest(t, clientTransport, protocol.Request{ID: 1, Method: "api.execute", Params: commandPayload})
	_, typ, payload := receiveProtocolFrame(t, clientTransport)
	if typ != wire.TypeResponse {
		t.Fatalf("browser open response type = %d", typ)
	}
	response, err := protocol.DecodeResponsePayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	result, err := protocol.DecodeApplicationResult(response.Result)
	if err != nil {
		t.Fatal(err)
	}
	resource := result.GetBrowserProxyOpen().GetResource()
	if resource == nil || resource.GetKind() != apipb.ResourceKind_RESOURCE_KIND_BROWSER_PROXY || len(resource.GetOpaqueToken()) < 2 {
		t.Fatalf("browser result = %#v error = %#v", result.GetBrowserProxyOpen(), result.GetError())
	}
	channel := uint16(resource.GetOpaqueToken()[0])<<8 | uint16(resource.GetOpaqueToken()[1])
	if err := sendBrowserTestFrame(clientTransport, channel, wire.TypeBrowserData, []byte("from-client")); err != nil {
		t.Fatal(err)
	}
	_, typ, payload = receiveProtocolFrame(t, clientTransport)
	if typ != wire.TypeBrowserData || string(payload) != "from-client" {
		t.Fatalf("browser data response = type %d payload %q", typ, payload)
	}
	if err := sendBrowserTestFrame(clientTransport, channel, wire.TypeClosed, nil); err != nil {
		t.Fatal(err)
	}
	if err := clientTransport.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-accepted:
		if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) {
			t.Fatalf("echo connection = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("daemon did not connect to the target")
	}
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, io.EOF) {
			t.Fatalf("protocol session = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("protocol session did not stop")
	}
}

func browserProxyTestExecutorFactory(port ApplicationSessionPort) ApplicationExecutor {
	return browserProxyTestExecutor{port: port}
}

type browserProxyTestExecutor struct {
	port ApplicationSessionPort
}

func (executor browserProxyTestExecutor) Execute(ctx context.Context, command *apipb.CommandEnvelope) *apipb.ResultEnvelope {
	request := command.GetContext()
	result := &apipb.ResultEnvelope{
		RequestId:     request.GetRequestId(),
		OriginSession: cloneApplicationTestSession(request.GetSession()),
	}
	lease, err := executor.port.AcquireApplication(ctx, ApplicationAdmission{Capability: ApplicationCapabilityBrowserProxy})
	if err != nil {
		result.Result = &apipb.ResultEnvelope_Error{Error: &apipb.ApiError{Message: err.Error()}}
		return result
	}
	defer lease.Release()
	proxy, err := executor.port.ApplicationBrowserProxyOpen(ctx, command.GetBrowserProxyOpen().GetHost(), uint16(command.GetBrowserProxyOpen().GetPort()))
	if err != nil {
		result.Result = &apipb.ResultEnvelope_Error{Error: &apipb.ApiError{Message: err.Error()}}
		return result
	}
	result.Result = &apipb.ResultEnvelope_BrowserProxyOpen{BrowserProxyOpen: &apipb.BrowserProxyOpenResult{Resource: &apipb.ResourceHandle{
		OpaqueToken: proxy.Token,
		Kind:        apipb.ResourceKind_RESOURCE_KIND_BROWSER_PROXY,
		Session:     cloneApplicationTestSession(request.GetSession()),
		Generation:  1,
	}}}
	return result
}

func sendBrowserTestFrame(connection interface{ Send([]byte) error }, channel uint16, typ uint8, payload []byte) error {
	frame, err := wire.EncodeFrame(channel, typ, payload)
	if err != nil {
		return err
	}
	return connection.Send(frame)
}

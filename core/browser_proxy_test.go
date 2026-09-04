package core

import (
	"context"
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

package localweb

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	protocoladapter "github.com/anytty/anytty/client/adapter/protocol"
	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/bindingpb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport"
	"github.com/anytty/anytty/shared/transport/memory"
)

const localEndpointID = "local"

type Core interface {
	ServeTransport(context.Context, transport.Transport) error
}

type bindingHost struct {
	core       Core
	generation atomic.Uint64
}

type readySession struct {
	*protocoladapter.ApplicationClient
}

func (session *readySession) ExecuteApplication(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return session.ApplicationSession.Execute(ctx, command)
}

func (session *readySession) ExecuteApplicationTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return session.ApplicationSession.ExecuteTerminal(ctx, command)
}

func (host *bindingHost) OpenSession(ctx context.Context, request *bindingpb.OpenSessionRequest) (clientruntime.ApplicationReadyPeerSession, error) {
	if host == nil || host.core == nil {
		return nil, fmt.Errorf("local web core is unavailable")
	}
	if request == nil || strings.TrimSpace(request.GetEndpointId()) != localEndpointID {
		return nil, fmt.Errorf("local web endpoint must be %q", localEndpointID)
	}
	clientTransport, daemonTransport := memory.NewPair()
	serveCtx, cancelServe := context.WithCancel(context.Background())
	serveDone := make(chan struct{})
	go func() {
		defer close(serveDone)
		_ = host.core.ServeTransport(serveCtx, daemonTransport)
	}()

	client := internalprotocol.NewClient(clientTransport)
	closeFailed := func() {
		_ = client.Close()
		cancelServe()
		<-serveDone
	}
	if err := client.Hello(ctx, internalprotocol.Hello{Version: wire.Version, Client: "anytty-local-web"}); err != nil {
		closeFailed()
		return nil, fmt.Errorf("local web protocol hello: %w", err)
	}
	generation := host.generation.Add(1)
	stamp := clientruntime.EndpointSessionStamp{
		EndpointID: endpoint.EndpointID(localEndpointID),
		RouteID:    endpoint.RouteID("local-web"),
		Generation: clientruntime.SessionGeneration(generation),
	}
	ready, err := protocoladapter.NewApplicationClientWithObservedPath(client, stamp, "local_web")
	if err != nil {
		closeFailed()
		return nil, err
	}
	identity, err := protocoladapter.VerifyDaemonIdentity(ctx, ready.ApplicationSession, endpoint.DaemonIdentity{})
	if err != nil {
		closeFailed()
		return nil, fmt.Errorf("verify local daemon identity: %w", err)
	}
	if err := ready.MarkReady(clientruntime.ReadyPeerSessionEvidence{
		Identity: identity, IdentityVerified: true, AuthorizationVerified: true, ProtocolVersion: wire.Version,
	}); err != nil {
		closeFailed()
		return nil, err
	}
	go func() {
		<-ready.Done()
		cancelServe()
	}()
	return &readySession{ApplicationClient: ready}, nil
}

var _ clientruntime.ApplicationReadyPeerSession = (*readySession)(nil)
var _ clientruntime.TerminalResponseApplicationExecutor = (*readySession)(nil)

package binding

import (
	"context"
	"fmt"

	"github.com/anytty/anytty/client/browserproxy"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/bindingpb"
	"google.golang.org/protobuf/proto"
)

func (engine *Engine) startBrowserProxyListen(request *bindingpb.BrowserProxyListenRequest) (uint64, error) {
	if request == nil || request.GetRequestId() == "" || request.GetSessionHandle() == 0 || request.GetStop() && request.GetPort() == 0 || !request.GetStop() && request.GetPort() != 0 {
		return 0, fmt.Errorf("browser proxy listen request is incomplete")
	}
	handle, ctx, session, err := engine.startSessionOperation(request.GetSessionHandle())
	if err != nil {
		return 0, err
	}
	go engine.runBrowserProxyListen(handle, ctx, session, proto.Clone(request).(*bindingpb.BrowserProxyListenRequest))
	return handle, nil
}

func (engine *Engine) runBrowserProxyListen(handle uint64, ctx context.Context, session clientruntime.ApplicationReadyPeerSession, request *bindingpb.BrowserProxyListenRequest) {
	port, err := engine.browserProxyListen(ctx, session, request)
	engine.markOperationDone(handle)
	result := &bindingpb.BrowserProxyListenResult{RequestId: request.GetRequestId(), OperationHandle: handle, SessionHandle: request.GetSessionHandle(), Port: port}
	if err != nil {
		result.Error = apiError(err)
	}
	engine.emitForHandle(handle, &bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_BrowserProxyListen{BrowserProxyListen: result}})
}

func (engine *Engine) browserProxyListen(ctx context.Context, session clientruntime.ApplicationReadyPeerSession, request *bindingpb.BrowserProxyListenRequest) (uint32, error) {
	// Only local listener setup is performed under the registry lock.
	engine.mu.Lock()
	defer engine.mu.Unlock()
	record := engine.sessions[request.GetSessionHandle()]
	if engine.closed || record == nil || record.closed || record.closing || record.session != session || record.rendererID != engine.activeRenderer {
		return 0, ErrInvalidHandle
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if request.GetStop() {
		if record.browserProxy != nil && record.browserProxy.Port() == request.GetPort() {
			_ = record.browserProxy.Close()
			record.browserProxy = nil
		}
		return 0, nil
	}
	if record.browserProxy != nil {
		return record.browserProxy.Port(), nil
	}
	opener, err := browserproxy.OpenSession(session)
	if err != nil {
		return 0, err
	}
	parent := engine.renderers[record.rendererID].ctx
	server, err := browserproxy.Start(parent, opener)
	if err != nil {
		return 0, err
	}
	if err = ctx.Err(); err != nil {
		_ = server.Close()
		return 0, err
	}
	record.browserProxy = server
	go func() {
		select {
		case <-parent.Done():
		case <-session.Done():
		case <-server.Done():
		}
		_ = server.Close()
	}()
	return server.Port(), nil
}

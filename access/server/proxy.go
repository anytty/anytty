package server

import (
	"context"

	"github.com/anytty/anytty/access/proxy"
	apimapping "github.com/anytty/anytty/api_mapping"
	"github.com/anytty/anytty/proto/access/apipb"
)

// executeProxy 在 access 本地终结 browser proxy open（Phase 2 迁移）：
// 拨号出口是 access 主机，双向流绑定到客户端 channel。
func (session *session) executeProxy(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	if err := apimapping.ValidateBrowserProxyCommand(command); err != nil {
		return nil, &routeError{apiError: apimapping.ErrorToProto(err, false)}
	}
	request := command.GetBrowserProxyOpen()
	channel, err := session.allocateLocalChannel()
	if err != nil {
		return nil, &routeError{apiError: &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_RESOURCE_EXHAUSTED, Message: err.Error(), Retryable: true}}
	}
	metadata, handle, err := session.server.proxy.Open(ctx, proxy.OpenRequest{
		Host:          request.GetHost(),
		Port:          uint16(request.GetPort()),
		ReceiveWindow: request.GetReceiveWindowBytes(),
		SendWindow:    request.GetSendWindowBytes(),
		Channel:       channel,
	}, sessionStream{session: session, channel: channel})
	if err != nil {
		session.releaseLocalChannel(channel)
		return nil, fileRouteError(err)
	}
	session.registerLocalBinding(apipb.ResourceKind_RESOURCE_KIND_BROWSER_PROXY, metadata.Token, channel, "", handle)
	envelope := newResultEnvelope(command)
	envelope.Result = &apipb.ResultEnvelope_BrowserProxyOpen{BrowserProxyOpen: &apipb.BrowserProxyOpenResult{
		Resource: &apipb.ResourceHandle{
			OpaqueToken: append([]byte(nil), metadata.Token...),
			Kind:        apipb.ResourceKind_RESOURCE_KIND_BROWSER_PROXY,
			Session:     cloneSessionStamp(command.GetContext().GetSession()),
			Generation:  1,
		},
		ReceiveWindowBytes: metadata.ReceiveWindowBytes,
		SendWindowBytes:    metadata.SendWindowBytes,
	}}
	return envelope, nil
}

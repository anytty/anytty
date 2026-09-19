package server

import (
	"context"

	accesscontract "github.com/anytty/anytty/access/contract"
	apimapping "github.com/anytty/anytty/api_mapping"
	"github.com/anytty/anytty/proto/access/apipb"
)

// AuthServices 是 access 直答 client_access.* / cloud.* 的进程内边界。
// daemon 不再挂载 ClientAccessService/RemoteService，也不再走反向 control RPC；
// access/server 直接把命令映射到 access runtime 服务。
type AuthServices struct {
	Access accesscontract.ClientAccessService
	Remote accesscontract.RemoteService
}

// executeAuth 在 access 本地终结鉴权/配对/Cloud 命令。
func (session *session) executeAuth(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	if err := apimapping.ValidateAccessRemoteCommand(command); err != nil {
		return nil, &routeError{apiError: apimapping.ErrorToProto(err, false)}
	}
	auth := session.server.cfg.Auth
	if auth == nil || (auth.Access == nil && auth.Remote == nil) {
		return nil, &routeError{apiError: &apipb.ApiError{
			Code: apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE, Message: "access runtime is unavailable", Retryable: true,
		}}
	}
	envelope := newResultEnvelope(command)
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_ClientAccessIdentity:
		if auth.Access == nil {
			return nil, errUnsupportedCommand
		}
		result, err := auth.Access.Identity(ctx, value.ClientAccessIdentity.GetChallenge())
		if err != nil {
			return nil, authRouteError(err)
		}
		envelope.Result = &apipb.ResultEnvelope_ClientAccessIdentity{ClientAccessIdentity: apimapping.ClientAccessIdentityResultToProto(result)}
	case *apipb.CommandEnvelope_ClientAccessList:
		if auth.Access == nil {
			return nil, errUnsupportedCommand
		}
		records, err := auth.Access.List(ctx)
		if err != nil {
			return nil, authRouteError(err)
		}
		envelope.Result = &apipb.ResultEnvelope_ClientAccessList{ClientAccessList: apimapping.ClientAccessListToProto(records)}
	case *apipb.CommandEnvelope_ClientAccessTicketCreate:
		if auth.Access == nil {
			return nil, errUnsupportedCommand
		}
		result, err := auth.Access.CreateTicket(ctx, apimapping.ClientAccessTicketRequestFromProto(value.ClientAccessTicketCreate))
		if err != nil {
			return nil, authRouteError(err)
		}
		envelope.Result = &apipb.ResultEnvelope_ClientAccessTicketCreate{ClientAccessTicketCreate: apimapping.ClientAccessTicketResultToProto(result)}
	case *apipb.CommandEnvelope_ClientAccessRevoke:
		if auth.Access == nil {
			return nil, errUnsupportedCommand
		}
		result, err := auth.Access.Revoke(ctx, value.ClientAccessRevoke.GetRequest().GetGrantId())
		if err != nil {
			return nil, authRouteError(err)
		}
		envelope.Result = &apipb.ResultEnvelope_ClientAccessRevoke{ClientAccessRevoke: apimapping.ClientAccessRevokeToProto(result)}
	case *apipb.CommandEnvelope_RemoteStatus:
		if auth.Remote == nil {
			return nil, errUnsupportedCommand
		}
		status, err := auth.Remote.Status(ctx)
		if err != nil {
			return nil, authRouteError(err)
		}
		envelope.Result = &apipb.ResultEnvelope_RemoteStatus{RemoteStatus: apimapping.RemoteStatusToProto(status)}
	case *apipb.CommandEnvelope_RemotePairStart:
		if auth.Remote == nil {
			return nil, errUnsupportedCommand
		}
		result, err := auth.Remote.PairStart(ctx, apimapping.RemotePairStartRequestFromProto(value.RemotePairStart))
		if err != nil {
			return nil, authRouteError(err)
		}
		envelope.Result = &apipb.ResultEnvelope_RemotePairStart{RemotePairStart: apimapping.RemotePairStartToProto(result)}
	case *apipb.CommandEnvelope_RemoteLocalEnable:
		if auth.Remote == nil {
			return nil, errUnsupportedCommand
		}
		result, err := auth.Remote.LocalEnable(ctx, apimapping.RemoteLocalEnableRequestFromProto(value.RemoteLocalEnable))
		if err != nil {
			return nil, authRouteError(err)
		}
		envelope.Result = &apipb.ResultEnvelope_RemoteLocalStatus{RemoteLocalStatus: apimapping.RemoteLocalStatusToProto(result)}
	case *apipb.CommandEnvelope_RemoteLocalStatus:
		if auth.Remote == nil {
			return nil, errUnsupportedCommand
		}
		result, err := auth.Remote.LocalStatus(ctx)
		if err != nil {
			return nil, authRouteError(err)
		}
		envelope.Result = &apipb.ResultEnvelope_RemoteLocalStatus{RemoteLocalStatus: apimapping.RemoteLocalStatusToProto(result)}
	case *apipb.CommandEnvelope_RemoteLocalDisable:
		if auth.Remote == nil {
			return nil, errUnsupportedCommand
		}
		result, err := auth.Remote.LocalDisable(ctx)
		if err != nil {
			return nil, authRouteError(err)
		}
		envelope.Result = &apipb.ResultEnvelope_RemoteLocalStatus{RemoteLocalStatus: apimapping.RemoteLocalStatusToProto(result)}
	case *apipb.CommandEnvelope_RemoteCloudStatus:
		if auth.Remote == nil {
			return nil, errUnsupportedCommand
		}
		result, err := auth.Remote.CloudStatus(ctx)
		if err != nil {
			return nil, authRouteError(err)
		}
		envelope.Result = &apipb.ResultEnvelope_RemoteCloudStatus{RemoteCloudStatus: apimapping.RemoteCloudStatusToProto(result)}
	case *apipb.CommandEnvelope_RemoteCloudEnable:
		if auth.Remote == nil {
			return nil, errUnsupportedCommand
		}
		result, err := auth.Remote.CloudEnable(ctx)
		if err != nil {
			return nil, authRouteError(err)
		}
		envelope.Result = &apipb.ResultEnvelope_RemoteCloudStatus{RemoteCloudStatus: apimapping.RemoteCloudStatusToProto(result)}
	case *apipb.CommandEnvelope_RemoteCloudDisable:
		if auth.Remote == nil {
			return nil, errUnsupportedCommand
		}
		result, err := auth.Remote.CloudDisable(ctx)
		if err != nil {
			return nil, authRouteError(err)
		}
		envelope.Result = &apipb.ResultEnvelope_RemoteCloudStatus{RemoteCloudStatus: apimapping.RemoteCloudStatusToProto(result)}
	case *apipb.CommandEnvelope_RemoteCloudEdges:
		if auth.Remote == nil {
			return nil, errUnsupportedCommand
		}
		result, err := auth.Remote.CloudEdges(ctx)
		if err != nil {
			return nil, authRouteError(err)
		}
		envelope.Result = &apipb.ResultEnvelope_RemoteCloudEdges{RemoteCloudEdges: apimapping.RemoteCloudEdgeSelectionToProto(result)}
	case *apipb.CommandEnvelope_RemoteCloudPreferEdge:
		if auth.Remote == nil {
			return nil, errUnsupportedCommand
		}
		result, err := auth.Remote.CloudPreferEdge(ctx, value.RemoteCloudPreferEdge.GetEdgeId(), value.RemoteCloudPreferEdge.GetExpectedPreferenceRevision())
		if err != nil {
			return nil, authRouteError(err)
		}
		envelope.Result = &apipb.ResultEnvelope_RemoteCloudEdges{RemoteCloudEdges: apimapping.RemoteCloudEdgeSelectionToProto(result)}
	case *apipb.CommandEnvelope_RemoteCloudReselectEdge:
		if auth.Remote == nil {
			return nil, errUnsupportedCommand
		}
		result, err := auth.Remote.CloudReselectEdge(ctx)
		if err != nil {
			return nil, authRouteError(err)
		}
		envelope.Result = &apipb.ResultEnvelope_RemoteCloudEdges{RemoteCloudEdges: apimapping.RemoteCloudEdgeSelectionToProto(result)}
	default:
		return nil, errUnsupportedCommand
	}
	return envelope, nil
}

// authRouteError 把 access runtime 错误映射为公共 typed error。
func authRouteError(err error) *routeError {
	return &routeError{apiError: apimapping.ErrorToProto(apimapping.CoreError(err), false)}
}

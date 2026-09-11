package apilayer

import (
	"context"
	apimapping "github.com/anytty/anytty/api_mapping"
	"github.com/anytty/anytty/proto/apipb"
)

// PluginController is optional so older embedding hosts fail closed.
type PluginController interface {
	Plugin(context.Context, *apipb.PluginCommand) (*apipb.PluginResult, error)
}

func (s *Service) executePlugin(ctx context.Context, session *apipb.EndpointSessionStamp, request *apipb.RequestContext, command *apipb.PluginCommand) *apipb.ResultEnvelope {
	controller, ok := s.platform.(PluginController)
	if !ok {
		return unavailable(request.GetRequestId(), session, "plugin controller is unavailable")
	}
	if command == nil || command.GetCommand() == nil {
		return errorResult(request.GetRequestId(), session, &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST, Message: "plugin command is required"})
	}
	result, err := controller.Plugin(ctx, command)
	if err != nil {
		return errorResult(request.GetRequestId(), session, apimapping.ErrorToProto(err, true))
	}
	return &apipb.ResultEnvelope{RequestId: request.GetRequestId(), OriginSession: cloneSession(session), Result: &apipb.ResultEnvelope_Plugin{Plugin: result}}
}
func (adapter *coreApplicationAdapter) Plugin(ctx context.Context, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
	port, ok := adapter.port.(interface {
		ApplicationPlugin(context.Context, *apipb.PluginCommand) (*apipb.PluginResult, error)
	})
	if !ok {
		return nil, ErrAdmissionUnsupportedCapability
	}
	result, err := port.ApplicationPlugin(ctx, command)
	return result, apimapping.CoreError(err)
}

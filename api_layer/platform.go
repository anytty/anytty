package apilayer

import (
	"context"

	apimapping "github.com/anytty/anytty/api_mapping"
	"github.com/anytty/anytty/proto/apipb"
)

func (service *Service) executePlatform(ctx context.Context, session *apipb.EndpointSessionStamp, command *apipb.CommandEnvelope, requestContext *apipb.RequestContext) *apipb.ResultEnvelope {
	requestID := requestContext.GetRequestId()
	if service == nil || service.platform == nil {
		return unavailable(requestID, session, "platform controller is unavailable")
	}
	if err := validateApplicationCommand(command); err != nil {
		return errorResult(requestID, session, apimapping.ErrorToProto(err, false))
	}
	return service.dispatchPlatformCommand(ctx, requestID, session, command)
}

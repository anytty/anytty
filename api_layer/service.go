// Package apilayer 实现 generated Proto API 到 core-facing controller 的 application 边界。
package apilayer

import (
	"context"
	"errors"
	"fmt"

	"github.com/anytty/anytty/api_mapping"
	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

const supportedAPIMajor uint32 = 1

var (
	// ErrAdmissionUnauthorized 表示当前连接没有可认证的 application session 身份。
	ErrAdmissionUnauthorized = errors.New("API request admission is unauthorized")
	// ErrAdmissionForbidden 表示身份有效，但 command/resource 不在授权 scope 内。
	ErrAdmissionForbidden = errors.New("API request admission is forbidden")
	// ErrAdmissionUnsupportedCapability 表示连接没有协商当前 command 所需 capability。
	ErrAdmissionUnsupportedCapability = errors.New("API request capability was not negotiated")
)

// AdmissionLease 保证一次已授权 command 执行期间，其连接、channel binding 和 authorization scope 仍然有效。
// Release 必须幂等；API Layer 在 controller 返回后立即释放，不得把 lease 暴露给客户端。
type AdmissionLease interface {
	Release()
}

// RequestAdmission 是 protocol connection 到 API Layer 的原子准入边界。
// Acquire 必须在同一临界区校验连接存活、已协商 capability 及具体 command/resource authorization，
// 并返回覆盖 controller 执行期的 lease。EndpointSessionStamp 属于客户端 runtime correlation，不是 daemon authority truth。
type RequestAdmission interface {
	Acquire(context.Context, *apipb.CommandEnvelope, apipb.ApiCapability) (AdmissionLease, error)
}

// OperationController 是 API Layer 取消 core-owned operation 的窄内部边界。
// 跨层参数仍使用 generated proto，controller adapter 不得另建 Go request DTO。
// Proto 参数仅在调用期间借用，controller 必须只读且不得保留。
type OperationController interface {
	CancelOperation(context.Context, *apipb.OperationStamp) error
}

// ResourceController 是 API Layer 释放 core-owned 长期资源的窄内部边界。
// 跨层参数仍使用 generated proto；实现必须通过 owning registry 验证 opaque token、kind、generation 与 session ownership。
// Proto 参数仅在调用期间借用，controller 必须只读且不得保留。
type ResourceController interface {
	ReleaseResource(context.Context, *apipb.ResourceHandle) error
}

// Service 执行公共 Proto command，并保证所有失败返回带 request/session correlation 的 typed ResultEnvelope。
// Service 不拥有 transport framing；调用方只负责完整传递 envelope payload。
type Service struct {
	admission  RequestAdmission
	operations OperationController
	resources  ResourceController
	terminals  TerminalController
	platform   PlatformController
}

// NewService 创建 application API service；缺失 request admission 或 controller 的请求会 fail closed。
func NewService(admission RequestAdmission, operations OperationController, resources ResourceController, terminals TerminalController) *Service {
	return &Service{admission: admission, operations: operations, resources: resources, terminals: terminals}
}

// NewPlatformService 创建覆盖全部公共 application domain 的 API service。
// platform controller 缺失时对应 command fail closed，不回退旧 protocol method。
func NewPlatformService(admission RequestAdmission, operations OperationController, resources ResourceController, terminals TerminalController, platform PlatformController) *Service {
	return &Service{admission: admission, operations: operations, resources: resources, terminals: terminals, platform: platform}
}

// Execute 校验并执行单个 typed command。返回值始终非 nil，领域失败不会泄露成 Go error。
func (service *Service) Execute(ctx context.Context, command *apipb.CommandEnvelope) *apipb.ResultEnvelope {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := apimapping.ValidateCommandEnvelopeSize(command); err != nil {
		requestContext := apimapping.RequestContextForCommand(command)
		requestID, originSession := apimapping.SafeRequestCorrelation(requestContext)
		return errorResult(requestID, originSession, apimapping.ErrorToProto(err, false))
	}
	// admission、validation 和 controller dispatch 必须共享同一私有快照，禁止调用方并发修改 Proto 绕过授权。
	command = proto.Clone(command).(*apipb.CommandEnvelope)
	requestContext := apimapping.RequestContextForCommand(command)
	requestID, originSession := apimapping.SafeRequestCorrelation(requestContext)
	if err := apimapping.ValidateRequestContext(requestContext); err != nil {
		return errorResult(requestID, originSession, apimapping.ErrorToProto(err, false))
	}
	requestID = requestContext.GetRequestId()
	originSession = requestContext.GetSession()
	if requestContext.GetApiVersion().GetMajor() != supportedAPIMajor {
		return errorResult(requestID, originSession, &apipb.ApiError{
			Code:    apipb.ApiErrorCode_API_ERROR_CODE_UNSUPPORTED_VERSION,
			Message: fmt.Sprintf("unsupported API major version %d", requestContext.GetApiVersion().GetMajor()),
		})
	}
	if err := ctx.Err(); err != nil {
		return errorResult(requestID, originSession, apimapping.ErrorToProto(err, false))
	}
	requiredCapability := apimapping.RequiredCapabilityForCommand(command)
	lease, apiError := service.acquireAdmission(ctx, command, requiredCapability)
	if apiError != nil {
		return errorResult(requestID, originSession, apiError)
	}
	defer lease.Release()

	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_CancelOperation:
		return service.cancelOperation(ctx, originSession, requestContext, value.CancelOperation)
	case *apipb.CommandEnvelope_ReleaseResource:
		return service.releaseResource(ctx, originSession, requestContext, value.ReleaseResource)
	default:
		if requiredCapability != apipb.ApiCapability_API_CAPABILITY_UNSPECIFIED {
			if isTerminalCommand(command) {
				return service.executeTerminal(ctx, originSession, command, requestContext)
			}
			return service.executePlatform(ctx, originSession, command, requestContext)
		}
		return errorResult(requestID, originSession, &apipb.ApiError{
			Code:    apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST,
			Message: "command is unsupported or missing",
			Detail:  &apipb.ApiError_Validation{Validation: &apipb.ValidationErrorDetail{Field: "command", Reason: "is unsupported or missing"}},
		})
	}
}

func (service *Service) cancelOperation(ctx context.Context, originSession *apipb.EndpointSessionStamp, requestContext *apipb.RequestContext, command *apipb.CancelOperationCommand) *apipb.ResultEnvelope {
	requestID := requestContext.GetRequestId()
	if err := apimapping.ValidateOperationStamp(command.GetOperation(), requestContext.GetSession()); err != nil {
		return errorResult(requestID, originSession, apimapping.ErrorToProto(err, false))
	}
	if service == nil || service.operations == nil {
		return unavailable(requestID, originSession, "operation controller is unavailable")
	}
	if err := service.operations.CancelOperation(ctx, command.GetOperation()); err != nil {
		return errorResult(requestID, originSession, apimapping.ErrorToProto(err, true))
	}
	return acknowledge(requestID, originSession)
}

func (service *Service) releaseResource(ctx context.Context, originSession *apipb.EndpointSessionStamp, requestContext *apipb.RequestContext, command *apipb.ReleaseResourceCommand) *apipb.ResultEnvelope {
	requestID := requestContext.GetRequestId()
	if err := apimapping.ValidateResourceHandle(command.GetResource()); err != nil {
		return errorResult(requestID, originSession, apimapping.ErrorToProto(err, false))
	}
	if !apimapping.SessionStampsEqual(command.GetResource().GetSession(), originSession) {
		return errorResult(requestID, originSession, apimapping.ErrorToProto(&apimapping.ValidationError{Field: "resource.session", Reason: "must match request origin session"}, false))
	}
	if service == nil || service.resources == nil {
		return unavailable(requestID, originSession, "resource controller is unavailable")
	}
	if err := service.resources.ReleaseResource(ctx, command.GetResource()); err != nil {
		return errorResult(requestID, originSession, apimapping.ErrorToProto(err, true))
	}
	return acknowledge(requestID, originSession)
}

func (service *Service) acquireAdmission(ctx context.Context, command *apipb.CommandEnvelope, required apipb.ApiCapability) (AdmissionLease, *apipb.ApiError) {
	if service == nil || service.admission == nil {
		return nil, &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE, Message: "request admission is unavailable", Retryable: true}
	}
	lease, err := service.admission.Acquire(ctx, command, required)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return nil, apimapping.ErrorToProto(err, false)
		case errors.Is(err, ErrAdmissionUnauthorized):
			return nil, &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_UNAUTHORIZED, Message: err.Error()}
		case errors.Is(err, ErrAdmissionForbidden):
			return nil, &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_FORBIDDEN, Message: err.Error()}
		case errors.Is(err, ErrAdmissionUnsupportedCapability):
			return nil, &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_UNSUPPORTED_CAPABILITY, Message: fmt.Sprintf("required API capability %s was not negotiated", required.String())}
		default:
			return nil, &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE, Message: "request admission failed", Retryable: true}
		}
	}
	if lease == nil {
		return nil, &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL, Message: "request admission returned a nil lease"}
	}
	return lease, nil
}

func acknowledge(requestID string, session *apipb.EndpointSessionStamp) *apipb.ResultEnvelope {
	return &apipb.ResultEnvelope{RequestId: requestID, OriginSession: cloneSession(session), Result: &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}}
}

func errorResult(requestID string, session *apipb.EndpointSessionStamp, apiError *apipb.ApiError) *apipb.ResultEnvelope {
	return &apipb.ResultEnvelope{RequestId: requestID, OriginSession: cloneSession(session), Result: &apipb.ResultEnvelope_Error{Error: apiError}}
}

func unavailable(requestID string, session *apipb.EndpointSessionStamp, message string) *apipb.ResultEnvelope {
	return errorResult(requestID, session, &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE, Message: message, Retryable: true})
}

func cloneSession(session *apipb.EndpointSessionStamp) *apipb.EndpointSessionStamp {
	if session == nil {
		return nil
	}
	return proto.Clone(session).(*apipb.EndpointSessionStamp)
}

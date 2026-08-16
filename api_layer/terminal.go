package apilayer

import (
	"context"
	"errors"
	"time"

	"github.com/anytty/anytty/api_mapping"
	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

const attachmentCleanupTimeout = 5 * time.Second

// TerminalAttachTransaction 持有尚未发布的 attachment 资源。
// API Layer 校验 Result 后调用 Commit；任何校验或提交失败都调用 Rollback，防止客户端不可见资源泄漏。
type TerminalAttachTransaction interface {
	// Result 返回 pending attachment 的不可变 Proto projection；不得在返回后继续修改。
	Result() *apipb.TerminalAttachResult
	// Commit 原子发布 attachment，使后续 handle 操作可见。
	Commit(context.Context) error
	// Rollback 无条件释放 pending attachment；实现必须幂等并接受独立 cleanup context。
	Rollback(context.Context) error
}

func (service *Service) executeTerminal(ctx context.Context, currentSession *apipb.EndpointSessionStamp, command *apipb.CommandEnvelope, requestContext *apipb.RequestContext) *apipb.ResultEnvelope {
	requestID := requestContext.GetRequestId()
	if err := validateApplicationCommand(command); err != nil {
		return errorResult(requestID, currentSession, apimapping.ErrorToProto(err, false))
	}
	if service == nil || service.terminals == nil {
		return unavailable(requestID, currentSession, "terminal controller is unavailable")
	}
	return service.dispatchTerminalCommand(ctx, requestID, currentSession, command)
}

func cloneMessage[T proto.Message](message T) T {
	return proto.Clone(message).(T)
}

func (service *Service) terminalAttachResult(ctx context.Context, requestID string, session *apipb.EndpointSessionStamp, command *apipb.TerminalAttachCommand, transaction TerminalAttachTransaction, err error) *apipb.ResultEnvelope {
	if err != nil {
		if transaction != nil {
			if rollbackErr := rollbackAttachment(ctx, transaction); rollbackErr != nil {
				return errorResult(requestID, session, &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL, Message: "terminal attachment failed and rollback failed", Attempted: true})
			}
		}
		return errorResult(requestID, session, apimapping.ErrorToProto(err, true))
	}
	if transaction == nil {
		return terminalResultError(requestID, session, err)
	}
	result := transaction.Result()
	if err := apimapping.ValidateTerminalAttachResult(command, result, session); err != nil {
		if rollbackErr := rollbackAttachment(ctx, transaction); rollbackErr != nil {
			return errorResult(requestID, session, &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL, Message: "terminal attachment publication failed and rollback failed", Attempted: true})
		}
		return errorResult(requestID, session, &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL, Message: "terminal controller returned an invalid attachment result", Attempted: true})
	}
	if err := transaction.Commit(ctx); err != nil {
		if rollbackErr := rollbackAttachment(ctx, transaction); rollbackErr != nil {
			return errorResult(requestID, session, &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL, Message: "terminal attachment commit and rollback failed", Attempted: true})
		}
		return errorResult(requestID, session, apimapping.ErrorToProto(err, true))
	}
	return &apipb.ResultEnvelope{RequestId: requestID, OriginSession: cloneSession(session), Result: &apipb.ResultEnvelope_TerminalAttach{TerminalAttach: cloneMessage(result)}}
}

func rollbackAttachment(ctx context.Context, transaction TerminalAttachTransaction) error {
	cleanupContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), attachmentCleanupTimeout)
	defer cancel()
	return transaction.Rollback(cleanupContext)
}

func terminalResultError(requestID string, session *apipb.EndpointSessionStamp, err error) *apipb.ResultEnvelope {
	if err == nil {
		err = errors.New("terminal controller returned nil result")
	}
	return errorResult(requestID, session, apimapping.ErrorToProto(err, true))
}

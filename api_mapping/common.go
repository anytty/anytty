// Package apimapping 提供 core domain 与公共 Proto API 之间的无状态确定性映射。
// 它不建立连接、不处理 framing，也不拥有授权、session 或应用状态。
package apimapping

import (
	"context"
	"errors"
	"fmt"

	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

const (
	maxAPIIdentityBytes     = 256
	maxCommandEnvelopeBytes = 2 << 20
)

// ValidationError 描述 proto 字段不满足 API contract；Field 使用 proto field path。
type ValidationError struct {
	Field  string
	Reason string
}

// ClassifiedError 是 core adapter 向公共 typed error 映射层提交的稳定内部错误分类。
// 它不携带 request/result DTO，只描述 code、retryability 与原始诊断错误。
type ClassifiedError struct {
	Err       error
	Code      apipb.ApiErrorCode
	Retryable bool
	SyncLost  *apipb.OutputSyncLostErrorDetail
}

// Error 返回原始领域错误文本；公共响应仍由 ErrorToProto 统一生成。
func (err *ClassifiedError) Error() string {
	if err == nil || err.Err == nil {
		return "classified application error"
	}
	return err.Err.Error()
}

// Unwrap 保留 errors.Is/errors.As 诊断链路。
func (err *ClassifiedError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Err
}

// ValidateCommandEnvelopeSize 在 API Layer 深拷贝前执行总量门禁，避免超大 in-process/JNI/WASM 请求造成二次内存放大。
// 调用方在 Execute 返回前不得并发修改传入消息；API Layer 通过门禁后立即建立私有快照。
func ValidateCommandEnvelopeSize(command *apipb.CommandEnvelope) error {
	if command == nil {
		return validation("command", "is required")
	}
	if proto.Size(command) > maxCommandEnvelopeBytes {
		return validation("command", "serialized envelope exceeds 2 MiB")
	}
	return nil
}

// SafeRequestCorrelation 返回可以安全写入失败响应的 request ID 与 origin session。
// 各字段独立通过长度和结构校验后才回显，避免非法请求利用错误响应放大内存或网络输出。
func SafeRequestCorrelation(contextMessage *apipb.RequestContext) (string, *apipb.EndpointSessionStamp) {
	if contextMessage == nil {
		return "", nil
	}
	requestID := contextMessage.GetRequestId()
	if len(requestID) > maxAPIIdentityBytes {
		requestID = ""
	}
	session := contextMessage.GetSession()
	if ValidateSessionStamp(session) != nil {
		session = nil
	}
	return requestID, session
}

// Error 返回适合日志诊断的 validation 文本；客户端应读取转换后的 typed ApiError。
func (err *ValidationError) Error() string {
	if err == nil {
		return "invalid API request"
	}
	return fmt.Sprintf("%s: %s", err.Field, err.Reason)
}

// ValidateRequestContext 校验 envelope 顶层 request context 的版本与可选 session 结构。
// capability 不来自客户端请求；API Layer 必须向权威 session owner 查询已协商能力。
func ValidateRequestContext(contextMessage *apipb.RequestContext) error {
	if contextMessage == nil {
		return validation("context", "is required")
	}
	if contextMessage.GetRequestId() == "" {
		return validation("context.request_id", "is required")
	}
	if len(contextMessage.GetRequestId()) > maxAPIIdentityBytes {
		return validation("context.request_id", "exceeds 256 bytes")
	}
	version := contextMessage.GetApiVersion()
	if version == nil || version.GetMajor() == 0 {
		return validation("context.api_version.major", "must be greater than zero")
	}
	if contextMessage.GetSession() != nil {
		if err := ValidateSessionStamp(contextMessage.GetSession()); err != nil {
			return err
		}
	}
	return nil
}

// ValidateSessionStamp 校验 endpoint、route 和 generation fence。
func ValidateSessionStamp(stamp *apipb.EndpointSessionStamp) error {
	if stamp == nil {
		return validation("session", "is required")
	}
	if stamp.GetEndpointId() == "" {
		return validation("session.endpoint_id", "is required")
	}
	if stamp.GetRouteId() == "" {
		return validation("session.route_id", "is required")
	}
	if len(stamp.GetEndpointId()) > maxAPIIdentityBytes || len(stamp.GetRouteId()) > maxAPIIdentityBytes {
		return validation("session", "endpoint_id or route_id exceeds 256 bytes")
	}
	if stamp.GetGeneration() == 0 {
		return validation("session.generation", "must be greater than zero")
	}
	return nil
}

// ValidateOperationStamp 校验 operation stamp，并要求它与 request session fence 一致。
func ValidateOperationStamp(stamp *apipb.OperationStamp, requestSession *apipb.EndpointSessionStamp) error {
	if stamp == nil {
		return validation("operation", "is required")
	}
	if stamp.GetOperationId() == "" {
		return validation("operation.operation_id", "is required")
	}
	if len(stamp.GetOperationId()) > maxAPIIdentityBytes {
		return validation("operation.operation_id", "exceeds 256 bytes")
	}
	if err := ValidateSessionStamp(stamp.GetSession()); err != nil {
		return err
	}
	if requestSession == nil || !SessionStampsEqual(stamp.GetSession(), requestSession) {
		return validation("operation.session", "must match context.session")
	}
	return nil
}

// ValidateResourceHandle 校验 opaque、typed、session-bound resource handle 的结构。
// token 真伪和资源 registry ownership 仍由 owning controller 验证。
func ValidateResourceHandle(handle *apipb.ResourceHandle) error {
	if handle == nil {
		return validation("resource", "is required")
	}
	if len(handle.GetOpaqueToken()) == 0 {
		return validation("resource.opaque_token", "is required")
	}
	if len(handle.GetOpaqueToken()) > 256 {
		return validation("resource.opaque_token", "exceeds 256 bytes")
	}
	if !validResourceKind(handle.GetKind()) {
		return validation("resource.kind", "must be a known resource kind")
	}
	if err := ValidateSessionStamp(handle.GetSession()); err != nil {
		return validation("resource.session", err.Error())
	}
	if handle.GetGeneration() == 0 {
		return validation("resource.generation", "must be greater than zero")
	}
	return nil
}

// ErrorToProto 把 API mapping/core adapter 失败映射为公共 typed error。
// attempted 必须由 API Layer 在 adapter 调用边界确定，API mapping 不推断是否已执行副作用。
func ErrorToProto(err error, attempted bool) *apipb.ApiError {
	if err == nil {
		return nil
	}
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return &apipb.ApiError{
			Code:      apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST,
			Message:   validationErr.Error(),
			Attempted: attempted,
			Detail: &apipb.ApiError_Validation{Validation: &apipb.ValidationErrorDetail{
				Field: validationErr.Field, Reason: validationErr.Reason,
			}},
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_CANCELLED, Message: err.Error(), Attempted: attempted}
	}
	var classified *ClassifiedError
	if errors.As(err, &classified) {
		code := classified.Code
		if code == apipb.ApiErrorCode_API_ERROR_CODE_UNSPECIFIED {
			code = apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL
		}
		result := &apipb.ApiError{Code: code, Message: classified.Error(), Retryable: classified.Retryable, Attempted: attempted}
		if classified.SyncLost != nil {
			result.Detail = &apipb.ApiError_OutputSyncLost{OutputSyncLost: classified.SyncLost}
		}
		return result
	}
	return &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL, Message: "application operation failed", Attempted: attempted}
}

func validation(field string, reason string) error {
	return &ValidationError{Field: field, Reason: reason}
}

// SessionStampsEqual 只比较当前 schema 已知的 endpoint、route 和 generation 语义字段。
// 它故意忽略 protobuf unknown fields，避免未来扩字段导致旧服务误判 session fence。
func SessionStampsEqual(left *apipb.EndpointSessionStamp, right *apipb.EndpointSessionStamp) bool {
	return left.GetEndpointId() == right.GetEndpointId() && left.GetRouteId() == right.GetRouteId() && left.GetGeneration() == right.GetGeneration()
}

func validResourceKind(kind apipb.ResourceKind) bool {
	switch kind {
	case apipb.ResourceKind_RESOURCE_KIND_OPERATION,
		apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION,
		apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT,
		apipb.ResourceKind_RESOURCE_KIND_HISTORY_WINDOW,
		apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER:
		return true
	default:
		return false
	}
}

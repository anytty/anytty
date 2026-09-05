package runtime

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

// ConnectionSnapshot 是同一 ReadyPeerSession 的即时网络投影。
// Route/generation 来自 runtime；candidate/RTT 来自实际 peer stats，未知字段必须保持空值而不是推断。
type ConnectionSnapshot struct {
	RouteID              endpoint.RouteID
	RouteKind            endpoint.RouteKind
	ObservedPath         string
	SelectionReason      string
	SampledAt            time.Time
	RoundTrip            time.Duration
	LocalCandidateType   string
	RemoteCandidateType  string
	LocalAddress         string
	RemoteAddress        string
	LocalPort            uint16
	RemotePort           uint16
	PairID               string
	LocalRelatedAddress  string
	RemoteRelatedAddress string
	LocalRelatedPort     uint16
	RemoteRelatedPort    uint16
	LocalProtocol        string
	RemoteProtocol       string
	RelayTransport       string
	NetworkClass         string
	BytesSent            uint64
	BytesReceived        uint64
	PacketsSent          uint64
	LossEvents           uint64
	Connected            bool
}

// ConnectionSnapshotProvider 由持有实际 transport 的 ReadySession 实现。
// 调用方只能采样当前 session，不能用快照驱动路由、鉴权或 generation 状态机。
type ConnectionSnapshotProvider interface {
	ConnectionSnapshot(time.Time) (ConnectionSnapshot, bool)
}

// ProtoApplicationExecutor 是 client runtime 到 protocol/platform binding 的唯一公共 API 执行边界。
// 实现只运输完整 CommandEnvelope/ResultEnvelope，不解释 terminal 字段，也不选择 route 或 generation。
type ProtoApplicationExecutor interface {
	ExecuteApplication(context.Context, *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error)
}

// ApplicationSessionValidator 在任何具体 adapter 副作用前校验请求仍属于当前 EndpointSessionStamp。
// owner/shared lease 实现必须在 generation 被替换或 consumer lease 关闭后返回 Attempted=false 的 stale error。
type ApplicationSessionValidator interface {
	ValidateApplicationSession(EndpointSessionStamp) error
}

// ApplicationSessionInvalidator 在无法确认 session-owned 远端资源已销毁时撤销精确 generation。
// 实现必须使同一底层 ReadyPeerSession 的全部 consumer lease 失效并关闭 transport；调用方不得把它用于普通 consumer release。
type ApplicationSessionInvalidator interface {
	InvalidateApplicationSession(error) error
}

// TerminalResponseApplicationExecutor 在调用 context 取消后仍等待一个有界 terminal response。
// 只有会创建远端资源、且必须取得迟到结果完成销毁的 resource owner 可以选择该能力；transport 不得自行解释业务 command。
type TerminalResponseApplicationExecutor interface {
	ExecuteApplicationTerminal(context.Context, *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error)
}

// ApplicationReadyPeerSession 是已经完成 transport、授权与 protocol Hello 的可执行 session。
// route adapter 返回该接口后，runtime/binding 只能通过 generated Proto command/event 访问 daemon；关闭和 generation fence 仍由 ReadyPeerSession 约束。
type ApplicationReadyPeerSession interface {
	ReadyPeerSession
	ProtoApplicationExecutor
	ApplicationEvents(context.Context) (<-chan *apipb.EventEnvelope, error)
}

// ResourceStreamSession 是 ready connection 对 session-bound stream resource 的内部 framing 能力。
// resource 真值来自 apipb.ResourceHandle；frame type 与 payload 只在 protocol/binding adapter 间运输，不成为第二套业务 API。
type ResourceStreamSession interface {
	OpenResourceStream(*apipb.ResourceHandle) (ResourceStream, error)
}

// ApplicationAttachmentSession 是 protocol connection 对 attachment resource 与私有 stream channel 的相关性投影。
// channel 只在当前 ReadyPeerSession 内有效；TUI 必须携带原 resource/stamp，不能把 uint16 channel 当成跨 generation API identity。
type ApplicationAttachmentSession interface {
	ApplicationAttachmentChannel(*apipb.ResourceHandle) (uint16, bool)
	ApplicationAttachment(uint16) (*apipb.ResourceHandle, bool)
}

// ResourceStream 是 attachment/file resource 对应的有界有序 frame stream。
// Receive 必须响应 context 取消；Send 必须拒绝不属于该 resource kind 的 frame type；Close 必须幂等并解除 framing registry。
type ResourceStream interface {
	Receive(context.Context) (uint8, []byte, error)
	Send(context.Context, uint8, []byte) error
	Close() error
}

type protoApplicationEventSource interface {
	ApplicationEvents(context.Context) (<-chan *apipb.EventEnvelope, error)
}

// ApplicationSession 把 generated Proto command 绑定到一个不可变 ReadyPeerSession generation。
// request ID 与 operation ID 由该对象单调分配；调用方不得自行重建 session stamp 或跨 generation 复用资源。
type ApplicationSession struct {
	stamp         EndpointSessionStamp
	executor      ProtoApplicationExecutor
	requestPrefix string
	nextID        atomic.Uint64
}

// preparedApplicationRequest is an in-process trust marker created by the
// outermost ApplicationSession. Nested route adapters must preserve its exact
// request and session identity instead of allocating another correlation ID.
type preparedApplicationRequest struct {
	requestID string
	stamp     EndpointSessionStamp
}

type preparedApplicationRequestContextKey struct{}

// NewApplicationSession 建立 connection-bound Proto API session。
// stamp 不完整或 executor 缺失时立即失败，禁止创建可在运行期 fallback 的半初始化对象。
func NewApplicationSession(stamp EndpointSessionStamp, executor ProtoApplicationExecutor) (*ApplicationSession, error) {
	if err := stamp.Validate(); err != nil {
		return nil, err
	}
	if executor == nil {
		return nil, runtimeError(ErrorUnavailable, "application executor is required", nil)
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, runtimeError(ErrorUnavailable, "generate application request identity", err)
	}
	return &ApplicationSession{stamp: stamp, executor: executor, requestPrefix: fmt.Sprintf("app-%x", nonce)}, nil
}

// Stamp 返回该 application session 的不可变 generation fence。
func (session *ApplicationSession) Stamp() EndpointSessionStamp {
	if session == nil {
		return EndpointSessionStamp{}
	}
	return session.stamp
}

// ProtoStamp 返回当前 application session 的 generated Proto stamp 快照。
// 调用方可以把它保存为 attachment/view projection，但不得修改后再作为新的 generation truth。
func (session *ApplicationSession) ProtoStamp() *apipb.EndpointSessionStamp {
	if session == nil {
		return nil
	}
	return session.protoStamp()
}

// ValidateCurrent 在进入 protocol/resource adapter 前验证 executor 仍绑定当前 generation。
// 不支持动态 generation 的单连接 executor 只校验本地 stamp；runtime-owned executor 必须实现 ApplicationSessionValidator。
func (session *ApplicationSession) ValidateCurrent() error {
	if session == nil || session.executor == nil {
		return runtimeError(ErrorUnavailable, "application session is unavailable", nil)
	}
	if validator, ok := session.executor.(ApplicationSessionValidator); ok {
		return validator.ValidateApplicationSession(session.stamp)
	}
	return session.stamp.Validate()
}

// Execute 克隆 command、写入当前 session context 和每次调用唯一的 operation stamp，再交给 protocol binding。
// transport 失败与 ResultEnvelope typed error 都转换为 runtime error；失败不会自动重放 command。
func (session *ApplicationSession) Execute(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return session.execute(ctx, command, false)
}

// ExecuteTerminal 为必须取得迟到资源结果的调用保留有界 terminal response，同时复用本 session 的 request/operation stamp 与结果校验。
// executor 不支持该能力时立即失败，禁止绕过 ApplicationSession 自行复制 generation 或 correlation 逻辑。
func (session *ApplicationSession) ExecuteTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return session.execute(ctx, command, true)
}

func (session *ApplicationSession) execute(ctx context.Context, command *apipb.CommandEnvelope, terminal bool) (*apipb.ResultEnvelope, error) {
	if session == nil || session.executor == nil {
		return nil, runtimeError(ErrorUnavailable, "application session is unavailable", nil)
	}
	if ctx == nil {
		return nil, runtimeError(ErrorInvalidRequest, "application context is required", nil)
	}
	if command == nil {
		return nil, runtimeError(ErrorInvalidRequest, "application command is required", nil)
	}
	if err := session.ValidateCurrent(); err != nil {
		return nil, err
	}
	stamp := session.protoStamp()
	snapshot := proto.Clone(command).(*apipb.CommandEnvelope)
	requestID := ""
	if prepared, ok := ctx.Value(preparedApplicationRequestContextKey{}).(preparedApplicationRequest); ok {
		if prepared.stamp != session.stamp {
			return nil, &Error{Code: ErrorStaleSession, Message: "prepared application request belongs to a different endpoint session"}
		}
		requestContext := snapshot.GetContext()
		if prepared.requestID == "" || requestContext.GetRequestId() != prepared.requestID {
			return nil, &Error{Code: ErrorInvalidRequest, Message: "prepared application request correlation mismatch"}
		}
		if requestContext.GetApiVersion().GetMajor() != 1 {
			return nil, &Error{Code: ErrorInvalidRequest, Message: "prepared application request API version mismatch"}
		}
		if !applicationSessionStampsEqual(requestContext.GetSession(), stamp) {
			return nil, &Error{Code: ErrorStaleSession, Message: "prepared application request session mismatch"}
		}
		requestID = prepared.requestID
	} else {
		sequence := session.nextID.Add(1)
		if sequence == 0 {
			return nil, runtimeError(ErrorUnavailable, "application request sequence is exhausted", nil)
		}
		requestID = fmt.Sprintf("%s-%d", session.requestPrefix, sequence)
		snapshot.Context = &apipb.RequestContext{RequestId: requestID, ApiVersion: &apipb.ApiVersion{Major: 1}, Session: stamp}
		bindOperationStamp(snapshot, stamp, requestID)
		ctx = context.WithValue(ctx, preparedApplicationRequestContextKey{}, preparedApplicationRequest{requestID: requestID, stamp: session.stamp})
	}
	var result *apipb.ResultEnvelope
	var err error
	if terminal {
		executor, ok := session.executor.(TerminalResponseApplicationExecutor)
		if !ok {
			return nil, runtimeError(ErrorUnavailable, "application executor does not support terminal responses", nil)
		}
		result, err = executor.ExecuteApplicationTerminal(ctx, snapshot)
	} else {
		result, err = session.executor.ExecuteApplication(ctx, snapshot)
	}
	if err != nil {
		return nil, &Error{Code: CodeOf(err), Message: err.Error(), Cause: err, Attempted: WasAttempted(err)}
	}
	if result == nil {
		return nil, &Error{Code: ErrorUnavailable, Message: "application executor returned no result", Attempted: true}
	}
	if result.GetRequestId() != requestID {
		return nil, &Error{Code: ErrorUnavailable, Message: "application result request correlation mismatch", Attempted: true}
	}
	if !applicationSessionStampsEqual(result.GetOriginSession(), stamp) {
		return nil, &Error{Code: ErrorStaleSession, Message: "application result belongs to a different endpoint session", Attempted: true}
	}
	if apiError := result.GetError(); apiError != nil {
		return nil, runtimeErrorFromProto(apiError)
	}
	return result, nil
}

// HistoryWindow owns newly created frozen snapshots across cancellation. Page
// requests reuse a caller-owned token and therefore stay on the normal path.
func (session *ApplicationSession) HistoryWindow(ctx context.Context, command *apipb.HistoryWindowCommand) (*apipb.HistoryWindowResult, error) {
	mode := command.GetMode()
	if mode != apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_UNSPECIFIED && mode != apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST {
		return session.executeHistoryWindow(ctx, command)
	}
	resultEnvelope, err := session.ExecuteTerminal(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_HistoryWindow{HistoryWindow: command}})
	if err != nil {
		return nil, err
	}
	result := resultEnvelope.GetHistoryWindow()
	if result == nil {
		return nil, missingApplicationResult("history_window")
	}
	if ctx.Err() == nil {
		return result, nil
	}
	if result.GetTerminal() != nil && result.GetToken() != "" {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		cleanupErr := session.HistoryRelease(cleanupCtx, &apipb.HistoryReleaseCommand{
			Terminal:          proto.Clone(result.GetTerminal()).(*apipb.TerminalRef),
			Token:             result.GetToken(),
			HistoryGeneration: result.GetHistoryGeneration(),
		})
		cancel()
		if cleanupErr != nil {
			return nil, fmt.Errorf("cancelled history window cleanup: %w", cleanupErr)
		}
	}
	return nil, ctx.Err()
}

// EventSubscribe 建立 daemon session-owned subscription，并返回对应的 Proto event stream。
func (session *ApplicationSession) EventSubscribe(ctx context.Context, command *apipb.EventSubscribeCommand) (*apipb.EventSubscriptionResult, <-chan *apipb.EventEnvelope, error) {
	source, ok := session.executor.(protoApplicationEventSource)
	if !ok {
		return nil, nil, runtimeError(ErrorUnavailable, "application event source is unavailable", nil)
	}
	eventCtx, cancel := context.WithCancel(ctx)
	events, err := source.ApplicationEvents(eventCtx)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	resultEnvelope, err := session.ExecuteTerminal(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_EventSubscribe{EventSubscribe: command}})
	if err != nil {
		cancel()
		return nil, nil, err
	}
	result := resultEnvelope.GetEventSubscription()
	if result == nil {
		cancel()
		return nil, nil, missingApplicationResult("event_subscription")
	}
	subscription := result.GetSubscription()
	if ctx.Err() != nil {
		cancel()
		if subscription != nil {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			_ = session.ReleaseResource(cleanupCtx, &apipb.ReleaseResourceCommand{Resource: proto.Clone(subscription).(*apipb.ResourceHandle)})
			cleanupCancel()
		}
		return nil, nil, ctx.Err()
	}
	filtered := make(chan *apipb.EventEnvelope, 64)
	go func() {
		defer cancel()
		defer close(filtered)
		defer func() {
			if subscription == nil {
				return
			}
			cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			_ = session.ReleaseResource(cleanupCtx, &apipb.ReleaseResourceCommand{Resource: proto.Clone(subscription).(*apipb.ResourceHandle)})
			cleanupCancel()
		}()
		for {
			select {
			case <-eventCtx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				if !sameApplicationResource(event.GetSubscription(), subscription) {
					continue
				}
				select {
				case filtered <- event:
				case <-eventCtx.Done():
					return
				}
			}
		}
	}()
	return result, filtered, nil
}

func sameApplicationResource(left, right *apipb.ResourceHandle) bool {
	return left != nil && right != nil && left.GetKind() == right.GetKind() && left.GetGeneration() == right.GetGeneration() && applicationSessionStampsEqual(left.GetSession(), right.GetSession()) && bytes.Equal(left.GetOpaqueToken(), right.GetOpaqueToken())
}

func missingApplicationResult(kind string) error {
	return &Error{Code: ErrorUnavailable, Message: fmt.Sprintf("application result %s is missing", kind), Attempted: true}
}

func (session *ApplicationSession) protoStamp() *apipb.EndpointSessionStamp {
	return &apipb.EndpointSessionStamp{
		EndpointId: string(session.stamp.EndpointID),
		RouteId:    string(session.stamp.RouteID),
		Generation: uint64(session.stamp.Generation),
	}
}

func applicationSessionStampsEqual(left, right *apipb.EndpointSessionStamp) bool {
	return left != nil && right != nil &&
		left.GetEndpointId() == right.GetEndpointId() &&
		left.GetRouteId() == right.GetRouteId() &&
		left.GetGeneration() == right.GetGeneration()
}

func bindOperationStamp(command *apipb.CommandEnvelope, stamp *apipb.EndpointSessionStamp, requestID string) {
	operationID := commandOperationID(command)
	if operationID == "" {
		operationID = requestID
	}
	operation := &apipb.OperationStamp{Session: proto.Clone(stamp).(*apipb.EndpointSessionStamp), OperationId: operationID}
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_TerminalAttach:
		value.TerminalAttach.Operation = operation
	case *apipb.CommandEnvelope_TerminalDetach:
		value.TerminalDetach.Operation = operation
	case *apipb.CommandEnvelope_TerminalInput:
		value.TerminalInput.Operation = operation
	case *apipb.CommandEnvelope_TerminalResize:
		value.TerminalResize.Operation = operation
	case *apipb.CommandEnvelope_TerminalResizeLock:
		value.TerminalResizeLock.Operation = operation
	case *apipb.CommandEnvelope_FileDownloadOpen:
		value.FileDownloadOpen.Operation = operation
	case *apipb.CommandEnvelope_FileUploadOpen:
		value.FileUploadOpen.Operation = operation
	case *apipb.CommandEnvelope_FileTransferCancel:
		value.FileTransferCancel.Operation = operation
	}
}

func commandOperationID(command *apipb.CommandEnvelope) string {
	if command == nil {
		return ""
	}
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_TerminalAttach:
		return value.TerminalAttach.GetOperation().GetOperationId()
	case *apipb.CommandEnvelope_TerminalDetach:
		return value.TerminalDetach.GetOperation().GetOperationId()
	case *apipb.CommandEnvelope_TerminalInput:
		return value.TerminalInput.GetOperation().GetOperationId()
	case *apipb.CommandEnvelope_TerminalResize:
		return value.TerminalResize.GetOperation().GetOperationId()
	case *apipb.CommandEnvelope_TerminalResizeLock:
		return value.TerminalResizeLock.GetOperation().GetOperationId()
	case *apipb.CommandEnvelope_FileDownloadOpen:
		return value.FileDownloadOpen.GetOperation().GetOperationId()
	case *apipb.CommandEnvelope_FileUploadOpen:
		return value.FileUploadOpen.GetOperation().GetOperationId()
	case *apipb.CommandEnvelope_FileTransferCancel:
		return value.FileTransferCancel.GetOperation().GetOperationId()
	default:
		return ""
	}
}

func runtimeErrorFromProto(apiError *apipb.ApiError) error {
	code := ErrorUnavailable
	switch apiError.GetCode() {
	case apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST,
		apipb.ApiErrorCode_API_ERROR_CODE_UNSUPPORTED_VERSION,
		apipb.ApiErrorCode_API_ERROR_CODE_UNSUPPORTED_CAPABILITY,
		apipb.ApiErrorCode_API_ERROR_CODE_CONFLICT:
		code = ErrorInvalidRequest
	case apipb.ApiErrorCode_API_ERROR_CODE_UNAUTHORIZED,
		apipb.ApiErrorCode_API_ERROR_CODE_FORBIDDEN:
		code = ErrorAuthorization
	case apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND:
		code = ErrorNotFound
	case apipb.ApiErrorCode_API_ERROR_CODE_STALE_SESSION:
		code = ErrorStaleSession
	case apipb.ApiErrorCode_API_ERROR_CODE_STALE_RESOURCE:
		code = ErrorStaleResource
	case apipb.ApiErrorCode_API_ERROR_CODE_RESOURCE_EXHAUSTED:
		code = ErrorResourceExhausted
	case apipb.ApiErrorCode_API_ERROR_CODE_CANCELLED:
		code = ErrorCanceled
	case apipb.ApiErrorCode_API_ERROR_CODE_ENTITLEMENT_DENIED:
		code = ErrorEntitlement
	case apipb.ApiErrorCode_API_ERROR_CODE_DAEMON_BLOCKED:
		code = ErrorDaemonBlocked
	case apipb.ApiErrorCode_API_ERROR_CODE_DAEMON_DELETED:
		code = ErrorDaemonDeleted
	case apipb.ApiErrorCode_API_ERROR_CODE_RELAY_NOT_IN_PLAN:
		code = ErrorRelayNotInPlan
	case apipb.ApiErrorCode_API_ERROR_CODE_RELAY_QUOTA_EXHAUSTED:
		code = ErrorRelayQuotaExhausted
	case apipb.ApiErrorCode_API_ERROR_CODE_RELAY_CONCURRENCY_EXHAUSTED:
		code = ErrorRelayConcurrencyExhausted
	case apipb.ApiErrorCode_API_ERROR_CODE_SUBSCRIPTION_INACTIVE:
		code = ErrorSubscriptionInactive
	case apipb.ApiErrorCode_API_ERROR_CODE_RELAY_REGION_UNAVAILABLE:
		code = ErrorRelayRegionUnavailable
	}
	return &Error{Code: code, Message: apiError.GetMessage(), Attempted: apiError.GetAttempted(), Retryable: apiError.GetRetryable()}
}

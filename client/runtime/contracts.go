// Package runtime 定义跨平台客户端连接运行时的控制面契约。
// 本包拥有 session generation、attempt lifecycle 和 endpoint connection event；
// transport、credential、protocol 与 UI 只能通过 port/adapter 消费这些契约，不能建立第二份 session 真值。
package runtime

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/anytty/anytty/client/endpoint"
)

// SessionGeneration 标识某个 Endpoint 当前连接世代。
// generation 只由 client runtime 分配；adapter、TUI、CLI 和 storage 不得自行递增或持久化它。
type SessionGeneration uint64

// Valid 表示 generation 是否可以参与 session stamp 比较。
func (generation SessionGeneration) Valid() bool {
	return generation != 0
}

// ConnectIntent 描述调用方建立连接的稳定目的，而不是 UI scene 或命令名称。
type ConnectIntent string

const (
	// ConnectIntentInteractive 表示用户正在等待可交互 session。
	ConnectIntentInteractive ConnectIntent = "interactive"
	// ConnectIntentBackground 表示恢复、预热或后台同步需要 session。
	ConnectIntentBackground ConnectIntent = "background"
	// ConnectIntentProbe 表示只验证 endpoint 可达性，不建立长期 consumer binding。
	ConnectIntentProbe ConnectIntent = "probe"
)

// EndpointSessionStamp 是所有 endpoint-scoped runtime 操作的 generation fence。
// 它来自 ReadyPeerSession winner，必须原样传递到后续 operation；任何层都不得用当前 registry 推断或重建 stamp。
type EndpointSessionStamp struct {
	EndpointID endpoint.EndpointID
	RouteID    endpoint.RouteID
	Generation SessionGeneration
}

// Validate 校验 stamp 是否完整；失败表示调用方没有绑定到真实 ReadyPeerSession。
func (stamp EndpointSessionStamp) Validate() error {
	if strings.TrimSpace(string(stamp.EndpointID)) == "" {
		return runtimeError(ErrorInvalidRequest, "session stamp endpoint_id is required", nil)
	}
	if strings.TrimSpace(string(stamp.RouteID)) == "" {
		return runtimeError(ErrorInvalidRequest, "session stamp route_id is required", nil)
	}
	if !stamp.Generation.Valid() {
		return runtimeError(ErrorInvalidRequest, "session stamp generation is required", nil)
	}
	return nil
}

// ConnectRequest 是 application consumer 请求 client runtime 确保 endpoint session 的输入。
// RouteOverride 为空时由 planner 生成自动计划；非空时只允许选择该 route，不得静默 fallback。
type ConnectRequest struct {
	EndpointID    endpoint.EndpointID
	RouteOverride endpoint.RouteID
	Intent        ConnectIntent
}

// Validate 校验 application 请求，不读取 registry，也不选择 route。
func (request ConnectRequest) Validate() error {
	if strings.TrimSpace(string(request.EndpointID)) == "" {
		return runtimeError(ErrorInvalidRequest, "connect endpoint_id is required", nil)
	}
	switch request.Intent {
	case ConnectIntentInteractive, ConnectIntentBackground, ConnectIntentProbe:
		return nil
	default:
		return runtimeError(ErrorInvalidRequest, fmt.Sprintf("unsupported connect intent %q", request.Intent), nil)
	}
}

// SessionLease 是 runtime 返回给 application consumer 的不可变连接租约。
// lease 不暴露 transport 或 protocol client；消费者只能把 Stamp 带回 runtime operation API。
type SessionLease struct {
	Stamp EndpointSessionStamp
}

// Validate 校验 lease 是否来自完整 session stamp。
func (lease SessionLease) Validate() error {
	return lease.Stamp.Validate()
}

// DisconnectRequest 描述显式释放某个 endpoint generation 的请求。
// Stamp 不匹配当前 generation 时 runtime 必须返回 stale，不能关闭新的 winner。
type DisconnectRequest struct {
	Stamp EndpointSessionStamp
}

// Validate 校验断开请求的 generation fence。
func (request DisconnectRequest) Validate() error {
	return request.Stamp.Validate()
}

// EndpointPhase 是 runtime 对 application 发布的连接控制面阶段。
// 它不表示 terminal lifecycle，也不暴露 transport 内部状态。
type EndpointPhase string

const (
	EndpointPhaseIdle        EndpointPhase = "idle"
	EndpointPhasePlanning    EndpointPhase = "planning"
	EndpointPhaseResolving   EndpointPhase = "resolving"
	EndpointPhaseSignaling   EndpointPhase = "signaling"
	EndpointPhaseConnecting  EndpointPhase = "connecting"
	EndpointPhaseAuthorizing EndpointPhase = "authorizing"
	EndpointPhaseReady       EndpointPhase = "ready"
	EndpointPhaseOffline     EndpointPhase = "offline"
)

// EndpointEvent 是 runtime lifecycle mailbox 发布的 endpoint-scoped 事件。
// 事件不携带 credential、transport、protocol payload 或 terminal 内容。
type EndpointEvent struct {
	EndpointID           endpoint.EndpointID
	Stamp                EndpointSessionStamp
	Phase                EndpointPhase
	ObservedPath         string
	RouteSelectionReason string
	ErrorCode            ErrorCode
	Message              string
}

// Runtime 是 TUI、CLI 和未来平台 binding 消费的连接控制面 application interface。
// 实现是每个 Endpoint session/generation 的唯一 owner；consumer 不得缓存 ReadyPeerSession 或自行 dial。
type Runtime interface {
	EnsureSession(context.Context, ConnectRequest) (SessionLease, error)
	Disconnect(context.Context, DisconnectRequest) error
	WatchEndpoint(context.Context, endpoint.EndpointID) (<-chan EndpointEvent, error)
}

// ApplicationRuntime 为需要执行 endpoint-scoped Proto API 的 adapter 提供独立 consumer lease。
// 返回值仍由 SessionOwner 持有 generation 真值；调用方只持有共享 lease，并在不再使用时 Close。
type ApplicationRuntime interface {
	Runtime
	AcquireSession(context.Context, ConnectRequest) (ApplicationReadyPeerSession, error)
}

// AttemptRequest 是 planner 产生、runtime 交给单 route adapter 的不可变 attempt 描述。
// Target 与 Route 只返回克隆值；adapter 不能修改 registry，也不能选择其它 route。
type AttemptRequest struct {
	endpointID endpoint.EndpointID
	identity   endpoint.DaemonIdentity
	route      endpoint.AccessRoute
	generation SessionGeneration
	intent     ConnectIntent
}

// NewAttemptRequest 为已选定 route 建立 attempt contract。
// 构造函数不判断 race policy，只验证 route 属于 target 且 generation/intent 完整。
func NewAttemptRequest(target endpoint.Endpoint, routeID endpoint.RouteID, generation SessionGeneration, intent ConnectIntent) (AttemptRequest, error) {
	route, ok := target.Route(routeID)
	if !ok {
		return AttemptRequest{}, runtimeError(ErrorInvalidRequest, fmt.Sprintf("endpoint %q has no route %q", target.ID, routeID), nil)
	}
	request := AttemptRequest{endpointID: target.ID, identity: target.DaemonIdentity, route: cloneAttemptRoute(route), generation: generation, intent: intent}
	if err := request.Validate(); err != nil {
		return AttemptRequest{}, err
	}
	return request, nil
}

// Validate 校验 attempt 的身份、route、generation 与 intent 边界。
func (request AttemptRequest) Validate() error {
	if strings.TrimSpace(string(request.endpointID)) == "" {
		return runtimeError(ErrorInvalidRequest, "attempt endpoint_id is required", nil)
	}
	if err := request.route.Validate(request.identity); err != nil {
		return runtimeError(ErrorInvalidRequest, "invalid attempt route", err)
	}
	if !request.generation.Valid() {
		return runtimeError(ErrorInvalidRequest, "attempt generation is required", nil)
	}
	switch request.intent {
	case ConnectIntentInteractive, ConnectIntentBackground, ConnectIntentProbe:
		return nil
	default:
		return runtimeError(ErrorInvalidRequest, fmt.Sprintf("unsupported attempt intent %q", request.intent), nil)
	}
}

// Stamp 返回该 attempt 成功后必须写入 ReadyPeerSession 的 generation fence。
func (request AttemptRequest) Stamp() EndpointSessionStamp {
	return EndpointSessionStamp{EndpointID: request.endpointID, RouteID: request.route.ID, Generation: request.generation}
}

// EndpointID 返回 attempt 唯一允许连接的 daemon endpoint identity。
func (request AttemptRequest) EndpointID() endpoint.EndpointID {
	return request.endpointID
}

// DaemonIdentity 返回 attempt 必须验证的 daemon identity pin。
func (request AttemptRequest) DaemonIdentity() endpoint.DaemonIdentity {
	return request.identity
}

// Route 返回 attempt 唯一允许拨号的 Route 快照。
func (request AttemptRequest) Route() endpoint.AccessRoute {
	return cloneAttemptRoute(request.route)
}

// Intent 返回 attempt 的调用目的。
func (request AttemptRequest) Intent() ConnectIntent {
	return request.intent
}

// ReadyPeerSession 表示单 route attempt 已完成 transport、身份、授权和 protocol Hello。
// Close 必须释放 protocol、transport 和子进程资源；Done 关闭后 Err 返回最终失败，正常关闭返回 nil。
type ReadyPeerSession interface {
	Stamp() EndpointSessionStamp
	ObservedPath() string
	Readiness() ReadyPeerSessionEvidence
	Done() <-chan struct{}
	Err() error
	Close() error
}

// ReadyPeerSessionEvidence 冻结单次 route attempt 成为 ReadyPeerSession 前已经完成的安全边界。
// IdentityVerified 表示 adapter 已按 route 类型完成身份边界并取得 fresh daemon identity；managed 使用 channel-bound DeviceHello，local/SSH 使用 Proto challenge proof。
// AuthorizationVerified 与 ProtocolVersion 分别证明该 route 的授权条件和 protocol Hello 已完成；Identity 始终非空，并在 Endpoint 已有 pin 时要求精确匹配。
type ReadyPeerSessionEvidence struct {
	Identity              endpoint.DaemonIdentity
	IdentityVerified      bool
	AuthorizationVerified bool
	ProtocolVersion       uint32
}

// Validate 校验 evidence 是否足以参加 winner 线性化，并在 Endpoint 已有 pin 时要求精确匹配。
func (evidence ReadyPeerSessionEvidence) Validate(expected endpoint.DaemonIdentity) error {
	if !evidence.IdentityVerified {
		return runtimeError(ErrorIdentity, "route attempt did not verify daemon identity", nil)
	}
	if !evidence.AuthorizationVerified {
		return runtimeError(ErrorAuthorization, "route attempt did not complete authorization", nil)
	}
	if evidence.ProtocolVersion == 0 {
		return runtimeError(ErrorUnavailable, "route attempt did not complete protocol Hello", nil)
	}
	if err := evidence.Identity.Validate(true); err != nil {
		return runtimeError(ErrorIdentity, "route attempt returned invalid daemon identity", err)
	}
	if !expected.Empty() && evidence.Identity != expected {
		return runtimeError(ErrorIdentity, "route attempt daemon identity does not match endpoint pin", nil)
	}
	return nil
}

// PeerConnector 是具体 route adapter 提供给 runtime 的单 attempt 边界。
// Connect 只能尝试 request.Route；context 取消后必须停止建连并释放已创建资源，不能 fallback 到其它 route。
type PeerConnector interface {
	Connect(context.Context, AttemptRequest) (ReadyPeerSession, error)
}

// ValidateReadyPeerSession 校验 adapter 产出的 session 是否严格属于原始 attempt。
// stamp 不一致、缺失 Done signal 或 nil session 都必须在 winner CAS 前失败并关闭该资源，不能进入 runtime 当前 generation。
func ValidateReadyPeerSession(request AttemptRequest, session ReadyPeerSession) error {
	if session == nil || (reflect.ValueOf(session).Kind() == reflect.Pointer && reflect.ValueOf(session).IsNil()) {
		return runtimeError(ErrorUnavailable, "route attempt returned no ready session", nil)
	}
	stamp := session.Stamp()
	if err := stamp.Validate(); err != nil {
		return runtimeError(ErrorUnavailable, "route attempt returned invalid session stamp", err)
	}
	if stamp != request.Stamp() {
		return runtimeError(ErrorStaleSession, fmt.Sprintf("ready session stamp %#v does not match attempt %#v", stamp, request.Stamp()), nil)
	}
	if err := session.Readiness().Validate(request.DaemonIdentity()); err != nil {
		return err
	}
	if session.Done() == nil {
		return runtimeError(ErrorUnavailable, "ready session lifecycle signal is required", nil)
	}
	select {
	case <-session.Done():
		return runtimeError(ErrorUnavailable, "route attempt session ended before winner publication", session.Err())
	default:
	}
	return nil
}

// ErrorCode 是 runtime 对 application 暴露的稳定失败分类。
type ErrorCode string

const (
	ErrorInvalidRequest            ErrorCode = "invalid_request"
	ErrorUnsupportedRoute          ErrorCode = "unsupported_route"
	ErrorIdentity                  ErrorCode = "identity"
	ErrorAuthorization             ErrorCode = "authorization"
	ErrorNotFound                  ErrorCode = "not_found"
	ErrorUnavailable               ErrorCode = "unavailable"
	ErrorCanceled                  ErrorCode = "canceled"
	ErrorUserStopped               ErrorCode = "user_stopped"
	ErrorStaleSession              ErrorCode = "stale_session"
	ErrorStaleResource             ErrorCode = "stale_resource"
	ErrorResourceExhausted         ErrorCode = "resource_exhausted"
	ErrorEntitlement               ErrorCode = "entitlement_denied"
	ErrorDaemonBlocked             ErrorCode = "daemon_blocked"
	ErrorDaemonDeleted             ErrorCode = "daemon_deleted"
	ErrorRelayNotInPlan            ErrorCode = "relay_not_in_plan"
	ErrorRelayQuotaExhausted       ErrorCode = "relay_quota_exhausted"
	ErrorRelayConcurrencyExhausted ErrorCode = "relay_concurrency_exhausted"
	ErrorSubscriptionInactive      ErrorCode = "subscription_inactive"
	ErrorRelayRegionUnavailable    ErrorCode = "relay_region_unavailable"
)

// Error 是 runtime 边界返回的稳定错误；Cause 只用于日志和 errors.Is/As，不作为 UI 文本协议。
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
	// Attempted 表示请求是否已经越过 runtime generation guard 并调用 concrete adapter。
	// input/paste 等非幂等操作只有在 false 时才允许 consumer 发起新的显式 recovery，不能自动重放 payload。
	Attempted bool
	// Retryable is copied from the typed API error. Consumers must not infer it
	// from the display message.
	Retryable bool
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Message
}

// Unwrap 返回底层失败，供取消和 adapter 原始错误分类使用。
func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

// CodeOf 返回 runtime 稳定错误码；未知错误统一归 unavailable，不根据错误文本猜测安全语义。
func CodeOf(err error) ErrorCode {
	if err == nil {
		return ""
	}
	var runtimeErr *Error
	if errors.As(err, &runtimeErr) {
		return runtimeErr.Code
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return ErrorCanceled
	}
	return ErrorUnavailable
}

// WasAttempted 返回失败是否已经调用 concrete adapter；未知错误保守返回 true。
func WasAttempted(err error) bool {
	if err == nil {
		return false
	}
	var runtimeErr *Error
	if errors.As(err, &runtimeErr) {
		return runtimeErr.Attempted
	}
	return true
}

func IsRetryable(err error) bool {
	var runtimeErr *Error
	return errors.As(err, &runtimeErr) && runtimeErr.Retryable
}

func runtimeError(code ErrorCode, message string, cause error) error {
	return &Error{Code: code, Message: message, Cause: cause}
}

func cloneAttemptRoute(route endpoint.AccessRoute) endpoint.AccessRoute {
	if route.Priority != nil {
		priority := *route.Priority
		route.Priority = &priority
	}
	route.HostKeyFingerprints = append([]string(nil), route.HostKeyFingerprints...)
	route.SignalingAddresses = append([]string(nil), route.SignalingAddresses...)
	route.ICETCPAddresses = append([]string(nil), route.ICETCPAddresses...)
	route.AdvertisedAddresses = append([]string(nil), route.AdvertisedAddresses...)
	if route.CredentialDescriptor != nil {
		descriptor := *route.CredentialDescriptor
		route.CredentialDescriptor = &descriptor
	}
	return route
}

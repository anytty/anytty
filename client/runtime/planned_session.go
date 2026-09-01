package runtime

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
)

// RoutePlanEnvironment 是 composition root 提供给纯 planner 的当前平台能力快照。
// 它只包含 route kind 与 credential ref 索引，不携带 credential body、transport handle、Cloud response 或 runtime session state。
type RoutePlanEnvironment struct {
	SupportedRouteKinds     []endpoint.RouteKind
	AvailableCredentialRefs []string
}

// PeerConnectorResolver 按 planner 已选定的 RouteKind 返回 concrete single-route adapter。
// resolver 不得选择 route、执行 fallback 或缓存 ReadyPeerSession；同一种 kind 的 adapter 必须严格执行 AttemptRequest.Route。
type PeerConnectorResolver interface {
	Connector(endpoint.RouteKind) (PeerConnector, bool)
}

// PeerConnectorMap 是 composition root 使用的不可变 RouteKind 到 adapter 映射。
// NewPeerConnectorMap 会复制输入 map，后续调用方修改原 map 不得改变正在运行的 race。
type PeerConnectorMap struct {
	connectors map[endpoint.RouteKind]PeerConnector
}

// NewPeerConnectorMap 校验并复制 route adapter 映射；nil adapter 会在启动 race 前失败，不能变成运行期 fallback。
func NewPeerConnectorMap(values map[endpoint.RouteKind]PeerConnector) (PeerConnectorMap, error) {
	connectors := make(map[endpoint.RouteKind]PeerConnector, len(values))
	for kind, connector := range values {
		if connector == nil || (reflect.ValueOf(connector).Kind() == reflect.Pointer && reflect.ValueOf(connector).IsNil()) {
			return PeerConnectorMap{}, runtimeError(ErrorInvalidRequest, fmt.Sprintf("route %q connector is required", kind), nil)
		}
		connectors[kind] = connector
	}
	return PeerConnectorMap{connectors: connectors}, nil
}

// Connector 返回指定 route kind 的 adapter；返回值只读，不能借此修改 registry 或其它 kind 的 adapter。
func (registry PeerConnectorMap) Connector(kind endpoint.RouteKind) (PeerConnector, bool) {
	connector, ok := registry.connectors[kind]
	return connector, ok
}

// ConnectPlanned 为一个 Endpoint generation 执行 C3B attempt groups，并只发布首个完整 ReadyPeerSession。
// winner 选定后立即返回；loser 的取消和迟到 session 关闭由后台 drain 完成，不能阻塞已鉴权 winner 的发布。
func (owner *SessionOwner) ConnectPlanned(
	ctx context.Context,
	target endpoint.Endpoint,
	routeOverride endpoint.RouteID,
	intent ConnectIntent,
	environment RoutePlanEnvironment,
	clock port.Clock,
	dialers PeerConnectorResolver,
) (SessionLease, error) {
	if owner == nil || ctx == nil || clock == nil || dialers == nil {
		return SessionLease{}, runtimeError(ErrorInvalidRequest, "session owner, context, clock, and route dialers are required", nil)
	}
	unlock, err := owner.acquireEndpoint(ctx, target.ID)
	if err != nil {
		return SessionLease{}, err
	}
	defer unlock()
	return owner.connectPlanned(ctx, target, routeOverride, "", intent, environment, clock, dialers)
}

func (owner *SessionOwner) connectPlanned(ctx context.Context, target endpoint.Endpoint, routeOverride endpoint.RouteID, configKey string, intent ConnectIntent, environment RoutePlanEnvironment, clock port.Clock, dialers PeerConnectorResolver) (SessionLease, error) {
	override := owner.effectiveRouteOverride(target.ID, routeOverride, configKey)
	preflight := endpoint.RouteSelectionRequest{
		Endpoint: target, Intent: endpoint.ConnectIntent{Kind: string(intent)}, RouteOverride: override, Generation: 1,
		SupportedRouteKinds:     append([]endpoint.RouteKind(nil), environment.SupportedRouteKinds...),
		AvailableCredentialRefs: append([]string(nil), environment.AvailableCredentialRefs...),
	}
	if _, err := (endpoint.RouteSelectionPlanner{}).Plan(preflight); err != nil {
		return SessionLease{}, plannerRuntimeError(err)
	}
	generation, err := owner.beginEndpointGeneration(target.ID)
	if err != nil {
		return SessionLease{}, err
	}
	request := preflight
	request.Generation = endpoint.SessionGeneration(generation)
	plan, err := (endpoint.RouteSelectionPlanner{}).Plan(request)
	if err != nil {
		return SessionLease{}, plannerRuntimeError(err)
	}
	owner.publishEndpointEvent(EndpointEvent{EndpointID: target.ID, Stamp: EndpointSessionStamp{EndpointID: target.ID, Generation: generation}, Phase: EndpointPhasePlanning})
	selectionReason := "first_ready"
	if override != "" {
		selectionReason = "route_override"
	}
	lease, err := owner.runRoutePlan(ctx, target, intent, plan, selectionReason, clock, dialers)
	if err != nil {
		owner.publishEndpointEvent(EndpointEvent{EndpointID: target.ID, Stamp: EndpointSessionStamp{EndpointID: target.ID, Generation: generation}, Phase: EndpointPhaseOffline, ErrorCode: CodeOf(err), Message: err.Error()})
		return SessionLease{}, err
	}
	if routeOverride != "" {
		owner.mu.Lock()
		owner.stickyRoutes[target.ID] = stickyRouteSelection{routeID: routeOverride, configKey: configKey}
		owner.mu.Unlock()
	}
	return lease, nil
}

// AcquirePlanned 复用同 Endpoint、同 config key 的当前 winner，并为每个 consumer 返回独立 lease。
// replacement race 仍由 SessionOwner 单点提升 generation；consumer Close 只释放自己的订阅与资源视图。
func (owner *SessionOwner) AcquirePlanned(
	ctx context.Context,
	target endpoint.Endpoint,
	routeOverride endpoint.RouteID,
	intent ConnectIntent,
	configKey string,
	environment RoutePlanEnvironment,
	clock port.Clock,
	dialers PeerConnectorResolver,
) (ApplicationReadyPeerSession, error) {
	if owner == nil || ctx == nil || configKey == "" || clock == nil || dialers == nil {
		return nil, runtimeError(ErrorInvalidRequest, "session owner, context, config key, clock, and route dialers are required", nil)
	}
	unlock, err := owner.acquireEndpoint(ctx, target.ID)
	if err != nil {
		return nil, err
	}
	defer unlock()
	owner.mu.Lock()
	if owner.closed {
		owner.mu.Unlock()
		return nil, runtimeError(ErrorUnavailable, "session owner is closed", nil)
	}
	current := owner.current[target.ID]
	if current != nil && owner.configs[target.ID] == configKey && routeOverrideAllowsReuse(current.Stamp().RouteID, routeOverride) && owner.authority.isCurrent(target.ID, current.Stamp().Generation, current) {
		select {
		case <-current.Done():
		default:
			if routeOverride != "" {
				owner.stickyRoutes[target.ID] = stickyRouteSelection{routeID: routeOverride, configKey: configKey}
			}
			lease := owner.newSharedLeaseLocked(target.ID, current)
			owner.mu.Unlock()
			return lease, nil
		}
	}
	owner.mu.Unlock()
	lease, err := owner.connectPlanned(ctx, target, routeOverride, configKey, intent, environment, clock, dialers)
	if err != nil {
		return nil, err
	}
	ready, err := owner.ApplicationSession(lease)
	if err != nil {
		return nil, err
	}
	owner.mu.Lock()
	if owner.closed {
		owner.mu.Unlock()
		_ = ready.Close()
		return nil, runtimeError(ErrorUnavailable, "session owner is closed", nil)
	}
	owner.configs[target.ID] = configKey
	shared := owner.newSharedLeaseLocked(target.ID, ready)
	owner.mu.Unlock()
	return shared, nil
}

// EnsurePlanned 返回 runtime application 使用的不可变 SessionLease，并按 config key 复用当前 winner。
// 与 AcquirePlanned 不同，它不创建 consumer resource lease；调用方的 command/event 仍必须携带返回 stamp 回到 owner。
func (owner *SessionOwner) EnsurePlanned(
	ctx context.Context,
	target endpoint.Endpoint,
	routeOverride endpoint.RouteID,
	intent ConnectIntent,
	configKey string,
	environment RoutePlanEnvironment,
	clock port.Clock,
	dialers PeerConnectorResolver,
) (SessionLease, error) {
	if owner == nil || ctx == nil || configKey == "" || clock == nil || dialers == nil {
		return SessionLease{}, runtimeError(ErrorInvalidRequest, "session owner, context, config key, clock, and route dialers are required", nil)
	}
	unlock, err := owner.acquireEndpoint(ctx, target.ID)
	if err != nil {
		return SessionLease{}, err
	}
	defer unlock()
	owner.mu.Lock()
	if owner.closed {
		owner.mu.Unlock()
		return SessionLease{}, runtimeError(ErrorUnavailable, "session owner is closed", nil)
	}
	current := owner.current[target.ID]
	if current != nil && owner.configs[target.ID] == configKey && routeOverrideAllowsReuse(current.Stamp().RouteID, routeOverride) && owner.authority.isCurrent(target.ID, current.Stamp().Generation, current) {
		select {
		case <-current.Done():
		default:
			if routeOverride != "" {
				owner.stickyRoutes[target.ID] = stickyRouteSelection{routeID: routeOverride, configKey: configKey}
			}
			lease := SessionLease{Stamp: current.Stamp()}
			owner.mu.Unlock()
			return lease, nil
		}
	}
	owner.mu.Unlock()
	lease, err := owner.connectPlanned(ctx, target, routeOverride, configKey, intent, environment, clock, dialers)
	if err != nil {
		return SessionLease{}, err
	}
	owner.mu.Lock()
	if owner.closed {
		owner.mu.Unlock()
		_ = owner.Disconnect(context.Background(), DisconnectRequest{Stamp: lease.Stamp})
		return SessionLease{}, runtimeError(ErrorUnavailable, "session owner is closed", nil)
	}
	owner.configs[target.ID] = configKey
	owner.mu.Unlock()
	return lease, nil
}

// ClearRouteOverride 清除当前进程内 Endpoint sticky route；它不修改 connections registry，也不关闭当前 winner。
func (owner *SessionOwner) ClearRouteOverride(endpointID endpoint.EndpointID) {
	if owner == nil {
		return
	}
	owner.mu.Lock()
	delete(owner.stickyRoutes, endpointID)
	owner.mu.Unlock()
}

func routeOverrideAllowsReuse(current, explicit endpoint.RouteID) bool {
	return explicit == "" || explicit == current
}

// WatchEndpoint 订阅 bounded endpoint lifecycle mailbox。
// producer 使用非阻塞发送；慢 consumer 只丢弃中间投影，不得阻塞 winner/cleanup 或获得 transport/protocol payload。
func (owner *SessionOwner) WatchEndpoint(ctx context.Context, endpointID endpoint.EndpointID) (<-chan EndpointEvent, error) {
	if owner == nil || ctx == nil || endpointID == "" {
		return nil, runtimeError(ErrorInvalidRequest, "session owner and endpoint_id are required", nil)
	}
	watcher := make(chan EndpointEvent, 16)
	owner.mu.Lock()
	if owner.closed {
		owner.mu.Unlock()
		return nil, runtimeError(ErrorUnavailable, "session owner is closed", nil)
	}
	if owner.watchers[endpointID] == nil {
		owner.watchers[endpointID] = make(map[chan EndpointEvent]struct{})
	}
	owner.watchers[endpointID][watcher] = struct{}{}
	if current := owner.current[endpointID]; current != nil && owner.authority.isCurrent(endpointID, current.Stamp().Generation, current) {
		watcher <- EndpointEvent{EndpointID: endpointID, Stamp: current.Stamp(), Phase: EndpointPhaseReady, ObservedPath: current.ObservedPath(), RouteSelectionReason: "current_winner"}
	} else {
		watcher <- EndpointEvent{EndpointID: endpointID, Phase: EndpointPhaseIdle}
	}
	owner.mu.Unlock()
	go func() {
		<-ctx.Done()
		owner.mu.Lock()
		if watchers := owner.watchers[endpointID]; watchers != nil {
			if _, ok := watchers[watcher]; ok {
				delete(watchers, watcher)
				close(watcher)
			}
			if len(watchers) == 0 {
				delete(owner.watchers, endpointID)
			}
		}
		owner.mu.Unlock()
	}()
	return watcher, nil
}

type routeAttemptResult struct {
	index   int
	group   int
	request AttemptRequest
	ready   ReadyPeerSession
	err     error
}

func (owner *SessionOwner) runRoutePlan(ctx context.Context, target endpoint.Endpoint, intent ConnectIntent, plan endpoint.RouteSelectionPlan, selectionReason string, clock port.Clock, dialers PeerConnectorResolver) (SessionLease, error) {
	type scheduledAttempt struct {
		request AttemptRequest
		delay   time.Duration
		group   int
		cancel  context.CancelFunc
	}
	groups := plan.Groups()
	scheduled := make([]scheduledAttempt, 0)
	groupRemaining := make([]int, len(groups))
	groupFailed := make([]bool, len(groups))
	expedite := make([]chan struct{}, len(groups))
	for groupIndex, group := range groups {
		expedite[groupIndex] = make(chan struct{})
		for _, attempt := range group.Attempts() {
			request, err := NewAttemptRequest(target, attempt.Route.ID, SessionGeneration(plan.Generation()), intent)
			if err != nil {
				return SessionLease{}, err
			}
			groupRemaining[groupIndex]++
			scheduled = append(scheduled, scheduledAttempt{request: request, delay: group.StartDelay(), group: groupIndex})
		}
	}
	results := make(chan routeAttemptResult, len(scheduled))
	for index := range scheduled {
		item := &scheduled[index]
		attemptCtx, cancel := context.WithCancel(ctx)
		item.cancel = cancel
		go func(index int, item scheduledAttempt, attemptCtx context.Context) {
			results <- owner.executeScheduledAttempt(attemptCtx, index, item.group, item.request, item.delay, expedite[item.group], clock, dialers)
		}(index, *item, attemptCtx)
	}
	errorsByIndex := make([]error, len(scheduled))
	nextExpediteGroup := 1
	for received := 0; received < len(scheduled); received++ {
		result := <-results
		if result.err == nil {
			for index := range scheduled {
				scheduled[index].cancel()
			}
			remaining := len(scheduled) - received - 1
			if remaining > 0 {
				// 迟到结果只能被关闭，不能再次参与 winner 或 generation 判定。
				go closeLateRouteAttempts(results, remaining)
			}
			if ctx.Err() != nil {
				_ = result.ready.Close()
				return SessionLease{}, &Error{Code: ErrorCanceled, Message: "route race was canceled before winner publication", Cause: ctx.Err(), Attempted: true}
			}
			lease, err := owner.adoptReadyPeerSession(result.request, result.ready, selectionReason)
			if err != nil {
				return SessionLease{}, err
			}
			owner.publishEndpointEvent(EndpointEvent{EndpointID: target.ID, Stamp: lease.Stamp, Phase: EndpointPhaseReady, ObservedPath: result.ready.ObservedPath(), RouteSelectionReason: selectionReason})
			return lease, nil
		}
		if result.ready != nil {
			_ = result.ready.Close()
		}
		errorsByIndex[result.index] = result.err
		groupRemaining[result.group]--
		if groupRemaining[result.group] == 0 {
			groupFailed[result.group] = true
		}
		for nextExpediteGroup < len(groups) && groupFailed[nextExpediteGroup-1] {
			close(expedite[nextExpediteGroup])
			nextExpediteGroup++
		}
	}
	for index := range scheduled {
		scheduled[index].cancel()
	}
	attempted := false
	for _, err := range errorsByIndex {
		attempted = attempted || WasAttempted(err)
	}
	for _, code := range []ErrorCode{ErrorDaemonDeleted, ErrorDaemonBlocked} {
		for _, err := range errorsByIndex {
			if err != nil && CodeOf(err) == code {
				return SessionLease{}, err
			}
		}
	}
	for _, err := range errorsByIndex {
		if err != nil && CodeOf(err) != ErrorCanceled {
			return SessionLease{}, err
		}
	}
	return SessionLease{}, &Error{Code: ErrorCanceled, Message: "route race was canceled", Cause: ctx.Err(), Attempted: attempted}
}

func closeLateRouteAttempts(results <-chan routeAttemptResult, remaining int) {
	for ; remaining > 0; remaining-- {
		result := <-results
		if result.ready != nil {
			_ = result.ready.Close()
		}
	}
}

func (owner *SessionOwner) executeScheduledAttempt(ctx context.Context, index, group int, request AttemptRequest, delay time.Duration, expedite <-chan struct{}, clock port.Clock, dialers PeerConnectorResolver) routeAttemptResult {
	if delay > 0 {
		timer := clock.NewTimer(delay)
		if timer == nil {
			return routeAttemptResult{index: index, group: group, request: request, err: runtimeError(ErrorUnavailable, "route hedge timer is unavailable", nil)}
		}
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return routeAttemptResult{index: index, group: group, request: request, err: runtimeError(ErrorCanceled, "route attempt canceled before start", ctx.Err())}
		case <-timer.C():
		case <-expedite:
		}
	}
	connector, ok := dialers.Connector(request.Route().Kind)
	if !ok || connector == nil {
		return routeAttemptResult{index: index, group: group, request: request, err: runtimeError(ErrorUnsupportedRoute, fmt.Sprintf("route kind %q has no adapter", request.Route().Kind), nil)}
	}
	routeKind := request.Route().Kind
	publishProgress := func(phase EndpointPhase, stage EndpointConnectionStage) {
		owner.publishEndpointEvent(EndpointEvent{
			EndpointID:         request.EndpointID(),
			Stamp:              request.Stamp(),
			Phase:              phase,
			AttemptedRouteKind: routeKind,
			ConnectionStage:    stage,
		})
	}
	publishProgress(EndpointPhaseConnecting, EndpointStageAttemptStarting)
	ctx = withEndpointProgressReporter(ctx, publishProgress)
	ready, err := connector.Connect(ctx, request)
	if err != nil {
		if ready != nil {
			_ = ready.Close()
		}
		attemptErr := attemptedRuntimeError(err)
		owner.publishEndpointEvent(EndpointEvent{
			EndpointID:         request.EndpointID(),
			Stamp:              request.Stamp(),
			Phase:              EndpointPhaseConnecting,
			AttemptedRouteKind: routeKind,
			ConnectionStage:    EndpointStageAttemptFailed,
			ErrorCode:          CodeOf(attemptErr),
			Message:            errorMessage(attemptErr),
		})
		return routeAttemptResult{index: index, group: group, request: request, err: attemptErr}
	}
	if err := ValidateReadyPeerSession(request, ready); err != nil {
		if ready != nil {
			_ = ready.Close()
		}
		return routeAttemptResult{index: index, group: group, request: request, err: attemptedRuntimeError(err)}
	}
	if _, ok := ready.(ApplicationReadyPeerSession); !ok {
		_ = ready.Close()
		return routeAttemptResult{index: index, group: group, request: request, err: attemptedRuntimeError(runtimeError(ErrorUnavailable, "route attempt returned no Proto application session", nil))}
	}
	return routeAttemptResult{index: index, group: group, request: request, ready: ready}
}

func (owner *SessionOwner) effectiveRouteOverride(endpointID endpoint.EndpointID, explicit endpoint.RouteID, configKey string) endpoint.RouteID {
	if explicit != "" || owner == nil {
		return explicit
	}
	owner.mu.Lock()
	defer owner.mu.Unlock()
	sticky, ok := owner.stickyRoutes[endpointID]
	if !ok {
		return ""
	}
	if sticky.configKey != configKey {
		delete(owner.stickyRoutes, endpointID)
		return ""
	}
	return sticky.routeID
}

func (owner *SessionOwner) publishEndpointEvent(event EndpointEvent) {
	owner.mu.Lock()
	defer owner.mu.Unlock()
	owner.publishEndpointEventLocked(event)
}

func (owner *SessionOwner) publishEndpointEventLocked(event EndpointEvent) {
	for watcher := range owner.watchers[event.EndpointID] {
		select {
		case watcher <- event:
		default:
			select {
			case <-watcher:
			default:
			}
			select {
			case watcher <- event:
			default:
			}
		}
	}
}

func attemptedRuntimeError(err error) error {
	code := CodeOf(err)
	message := "route attempt failed"
	retryable := false
	var value *Error
	if errors.As(err, &value) && value.Message != "" {
		message = value.Message
		retryable = value.Retryable
	}
	return &Error{Code: code, Message: message, Cause: err, Attempted: true, Retryable: retryable}
}

func plannerRuntimeError(err error) error {
	var value *endpoint.Error
	if !errors.As(err, &value) {
		return runtimeError(ErrorInvalidRequest, "route planner rejected request", err)
	}
	switch value.Code {
	case endpoint.ErrorCredentialRequired, endpoint.ErrorAuthorizationRequired:
		return runtimeError(ErrorAuthorization, value.Message, err)
	case endpoint.ErrorRouteUnavailable:
		return runtimeError(ErrorUnsupportedRoute, value.Message, err)
	default:
		return runtimeError(ErrorInvalidRequest, value.Message, err)
	}
}

var _ PeerConnectorResolver = PeerConnectorMap{}

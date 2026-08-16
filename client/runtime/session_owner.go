package runtime

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/apipb"
)

// SessionOwner 是跨端 Go Client Engine 的 endpoint generation、planner race 与当前 ready session 真值。
// 单 route adapter 只能通过 AttemptRequest 参与 owner 启动的 race；Android、Web、TUI 和 CLI 不得在 owner 外缓存可执行 protocol client。
type SessionOwner struct {
	mu           sync.Mutex
	authority    *SessionGenerationAuthority
	current      map[endpoint.EndpointID]ApplicationReadyPeerSession
	configs      map[endpoint.EndpointID]string
	stickyRoutes map[endpoint.EndpointID]stickyRouteSelection
	selections   map[endpoint.EndpointID]routeSelection
	acquireLocks map[endpoint.EndpointID]*endpointAcquireEntry
	sharedLeases map[endpoint.EndpointID]map[*sharedApplicationLease]struct{}
	watchers     map[endpoint.EndpointID]map[chan EndpointEvent]struct{}
	ownerDone    chan struct{}
	closed       bool
}

type routeSelection struct {
	generation SessionGeneration
	reason     string
}

type endpointAcquireEntry struct {
	token chan struct{}
	refs  int
}

// stickyRouteSelection 只在产生显式选择时的连接配置仍未变化时复用 route。
// configKey 由 Endpoint、策略和平台能力共同生成，配置变化后必须回到 planner，不能让旧测试连接覆盖用户的新策略。
type stickyRouteSelection struct {
	routeID   endpoint.RouteID
	configKey string
}

// SessionGenerationAuthority 是 Go Client Engine 进程级的 endpoint generation 真值。
// engine/host 可以重建，但同一进程内后创建的 owner 必须继续递增；平台层只能持有该对象引用，不能读取或生成数值。
type SessionGenerationAuthority struct {
	mu          sync.Mutex
	generations map[endpoint.EndpointID]SessionGeneration
	current     map[endpoint.EndpointID]ApplicationReadyPeerSession
}

// NewSessionGenerationAuthority 创建空的进程级 generation authority。
func NewSessionGenerationAuthority() *SessionGenerationAuthority {
	return &SessionGenerationAuthority{generations: make(map[endpoint.EndpointID]SessionGeneration), current: make(map[endpoint.EndpointID]ApplicationReadyPeerSession)}
}

// NewSessionOwner 创建空的客户端 session owner。
// generation 从每个 endpoint 的 1 开始单调递增，进程内溢出时 fail closed，不能回绕后复活旧 resource stamp。
func NewSessionOwner() *SessionOwner {
	return NewSessionOwnerWithAuthority(NewSessionGenerationAuthority())
}

// NewSessionOwnerWithAuthority 创建共享进程级 generation 真值的 session owner。
// authority 只跨 host 重建复用，不改变每个 owner 对其 ready session 生命周期的独占责任。
func NewSessionOwnerWithAuthority(authority *SessionGenerationAuthority) *SessionOwner {
	if authority == nil {
		authority = NewSessionGenerationAuthority()
	}
	return &SessionOwner{
		authority:    authority,
		current:      make(map[endpoint.EndpointID]ApplicationReadyPeerSession),
		configs:      make(map[endpoint.EndpointID]string),
		stickyRoutes: make(map[endpoint.EndpointID]stickyRouteSelection),
		selections:   make(map[endpoint.EndpointID]routeSelection),
		acquireLocks: make(map[endpoint.EndpointID]*endpointAcquireEntry),
		sharedLeases: make(map[endpoint.EndpointID]map[*sharedApplicationLease]struct{}),
		watchers:     make(map[endpoint.EndpointID]map[chan EndpointEvent]struct{}),
		ownerDone:    make(chan struct{}),
	}
}

// AcquireRoute 复用同 endpoint、同 config key 的当前 ready session，并返回独立 consumer lease。
// config key 由 adapter 对已验证连接配置生成，runtime 只比较 opaque key；lease Close 只释放 consumer，配置变化或 owner teardown 才提升 generation。
func (owner *SessionOwner) AcquireRoute(ctx context.Context, target endpoint.Endpoint, routeID endpoint.RouteID, intent ConnectIntent, configKey string, dialer PeerConnector) (ApplicationReadyPeerSession, error) {
	if owner == nil || ctx == nil || dialer == nil || configKey == "" {
		return nil, runtimeError(ErrorInvalidRequest, "session owner, context, route dialer, and config key are required", nil)
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
	if current != nil && owner.configs[target.ID] == configKey && owner.authority.isCurrent(target.ID, current.Stamp().Generation, current) {
		select {
		case <-current.Done():
		default:
			lease := owner.newSharedLeaseLocked(target.ID, current)
			owner.mu.Unlock()
			return lease, nil
		}
	}
	owner.mu.Unlock()
	lease, err := owner.ConnectRoute(ctx, target, routeID, intent, dialer)
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

func (owner *SessionOwner) acquireEndpoint(ctx context.Context, endpointID endpoint.EndpointID) (func(), error) {
	owner.mu.Lock()
	entry := owner.acquireLocks[endpointID]
	if entry == nil {
		entry = &endpointAcquireEntry{token: make(chan struct{}, 1)}
		entry.token <- struct{}{}
		owner.acquireLocks[endpointID] = entry
	}
	// refs includes the current holder and every waiter that will use this entry.
	entry.refs++
	owner.mu.Unlock()

	select {
	case <-entry.token:
		if err := ctx.Err(); err != nil {
			entry.token <- struct{}{}
			owner.releaseEndpointAcquireRef(endpointID, entry)
			return nil, runtimeError(ErrorCanceled, "endpoint acquire was canceled", err)
		}
		owner.mu.Lock()
		closed := owner.closed
		owner.mu.Unlock()
		if closed {
			entry.token <- struct{}{}
			owner.releaseEndpointAcquireRef(endpointID, entry)
			return nil, runtimeError(ErrorUnavailable, "session owner is closed", nil)
		}
		return func() {
			entry.token <- struct{}{}
			owner.releaseEndpointAcquireRef(endpointID, entry)
		}, nil
	case <-ctx.Done():
		owner.releaseEndpointAcquireRef(endpointID, entry)
		return nil, runtimeError(ErrorCanceled, "endpoint acquire was canceled", ctx.Err())
	case <-owner.ownerDone:
		owner.releaseEndpointAcquireRef(endpointID, entry)
		return nil, runtimeError(ErrorUnavailable, "session owner is closed", nil)
	}
}

func (owner *SessionOwner) releaseEndpointAcquireRef(endpointID endpoint.EndpointID, entry *endpointAcquireEntry) {
	owner.mu.Lock()
	defer owner.mu.Unlock()
	entry.refs--
	if entry.refs == 0 && owner.acquireLocks[endpointID] == entry {
		delete(owner.acquireLocks, endpointID)
	}
}

// ConnectRoute 为调用方已经选定的唯一 route 建立新 generation。
// 新 generation 在拨号前即使旧 session 失效；并发连接只有最后分配的 generation 可以发布，迟到成功必须关闭并返回 stale。
func (owner *SessionOwner) ConnectRoute(
	ctx context.Context,
	target endpoint.Endpoint,
	routeID endpoint.RouteID,
	intent ConnectIntent,
	dialer PeerConnector,
) (SessionLease, error) {
	if owner == nil || dialer == nil {
		return SessionLease{}, runtimeError(ErrorUnavailable, "session owner or route dialer is unavailable", nil)
	}
	attempt, err := owner.BeginRouteAttempt(target, routeID, intent)
	if err != nil {
		return SessionLease{}, err
	}
	ready, err := dialer.Connect(ctx, attempt)
	if err != nil {
		return SessionLease{}, err
	}
	return owner.adoptReadyPeerSession(attempt, ready, "route_override")
}

// AdoptReadyPeerSession 把指定 attempt 已完成 Hello/auth 的 ready session 发布为当前 winner。
// 该入口只供 composition 迁移已经建立的 transport；attempt generation 必须先由 BeginRouteAttempt 分配，调用方不能伪造 stamp。
func (owner *SessionOwner) AdoptReadyPeerSession(attempt AttemptRequest, ready ReadyPeerSession) (SessionLease, error) {
	return owner.adoptReadyPeerSession(attempt, ready, "route_override")
}

func (owner *SessionOwner) adoptReadyPeerSession(attempt AttemptRequest, ready ReadyPeerSession, selectionReason string) (SessionLease, error) {
	if owner == nil {
		if ready != nil {
			_ = ready.Close()
		}
		return SessionLease{}, runtimeError(ErrorUnavailable, "session owner is unavailable", nil)
	}
	if err := ValidateReadyPeerSession(attempt, ready); err != nil {
		_ = ready.Close()
		return SessionLease{}, err
	}
	application, ok := ready.(ApplicationReadyPeerSession)
	if !ok {
		_ = ready.Close()
		return SessionLease{}, runtimeError(ErrorUnavailable, "route attempt returned no Proto application session", nil)
	}
	stamp := attempt.Stamp()
	endpointID := stamp.EndpointID
	generation := stamp.Generation

	owner.mu.Lock()
	owner.authority.mu.Lock()
	if owner.closed || owner.authority.generations[endpointID] != generation || owner.authority.current[endpointID] != nil {
		owner.authority.mu.Unlock()
		owner.mu.Unlock()
		_ = application.Close()
		return SessionLease{}, runtimeError(ErrorStaleSession, "route attempt completed after its endpoint generation was replaced", nil)
	}
	owner.authority.current[endpointID] = application
	owner.authority.mu.Unlock()
	owner.current[endpointID] = application
	owner.selections[endpointID] = routeSelection{generation: generation, reason: selectionReason}
	owner.mu.Unlock()
	go owner.removeWhenDone(endpointID, generation, application)
	return SessionLease{Stamp: attempt.Stamp()}, nil
}

// BeginRouteAttempt 为非 session 路由操作分配 endpoint generation，并立即撤销旧 session。
// Pairing 等认证握手必须通过该入口取得 AttemptRequest，平台 binding 不得自行生成 generation；调用方若不发布 ready session，
// 本次 generation 仍然有效地使旧 lease 失效，后续正式连接会再分配下一代。
func (owner *SessionOwner) BeginRouteAttempt(target endpoint.Endpoint, routeID endpoint.RouteID, intent ConnectIntent) (AttemptRequest, error) {
	attempts, err := owner.BeginRouteAttempts(target, []endpoint.RouteID{routeID}, intent)
	if err != nil {
		return AttemptRequest{}, err
	}
	return attempts[0], nil
}

// BeginRouteAttempts 为同一次 pairing/race 的多条 Route 分配同一个 endpoint generation。
// 所有 Route 会在推进 generation 前完成静态校验；调用方只能把这些 attempt 用于同一逻辑事务，不能拆分缓存或跨重连复用。
func (owner *SessionOwner) BeginRouteAttempts(target endpoint.Endpoint, routeIDs []endpoint.RouteID, intent ConnectIntent) ([]AttemptRequest, error) {
	if owner == nil {
		return nil, runtimeError(ErrorUnavailable, "session owner is unavailable", nil)
	}
	if len(routeIDs) == 0 {
		return nil, runtimeError(ErrorInvalidRequest, "at least one route attempt is required", nil)
	}
	seen := make(map[endpoint.RouteID]struct{}, len(routeIDs))
	for _, routeID := range routeIDs {
		if _, exists := seen[routeID]; exists {
			return nil, runtimeError(ErrorInvalidRequest, "route attempts must be unique", nil)
		}
		seen[routeID] = struct{}{}
		// 静态 route/identity/intent 必须在推进 generation 或关闭健康 session 前完成验证。
		if _, err := NewAttemptRequest(target, routeID, 1, intent); err != nil {
			return nil, err
		}
	}
	generation, err := owner.beginEndpointGeneration(target.ID)
	if err != nil {
		return nil, err
	}
	attempts := make([]AttemptRequest, 0, len(routeIDs))
	for _, routeID := range routeIDs {
		attempt, err := NewAttemptRequest(target, routeID, generation, intent)
		if err != nil {
			return nil, err
		}
		attempts = append(attempts, attempt)
	}
	return attempts, nil
}

func (owner *SessionOwner) beginEndpointGeneration(endpointID endpoint.EndpointID) (SessionGeneration, error) {
	owner.mu.Lock()
	if owner.closed {
		owner.mu.Unlock()
		return 0, runtimeError(ErrorUnavailable, "session owner is closed", nil)
	}
	owner.authority.mu.Lock()
	previousGeneration := owner.authority.generations[endpointID]
	if previousGeneration == SessionGeneration(math.MaxUint64) {
		owner.authority.mu.Unlock()
		owner.mu.Unlock()
		return 0, runtimeError(ErrorUnavailable, "endpoint session generation is exhausted", nil)
	}
	generation := previousGeneration + 1
	owner.authority.generations[endpointID] = generation
	previousGlobal := owner.authority.current[endpointID]
	delete(owner.authority.current, endpointID)
	owner.authority.mu.Unlock()
	previous := owner.current[endpointID]
	delete(owner.current, endpointID)
	delete(owner.configs, endpointID)
	delete(owner.selections, endpointID)
	shared := owner.takeSharedLeasesLocked(endpointID)
	owner.mu.Unlock()
	for _, lease := range shared {
		lease.finish(runtimeError(ErrorStaleSession, "endpoint session generation was replaced", nil))
	}
	if previousGlobal != nil {
		_ = previousGlobal.Close()
	} else if previous != nil {
		_ = previous.Close()
	}
	return generation, nil
}

// ApplicationSession 把 runtime 返回的 lease 投影为 generation-fenced application ready session。
// wrapper 不缓存可执行 protocol client；每次 command/event/resource 操作都回到 owner 校验 current winner，Close 也只断开精确 generation。
func (owner *SessionOwner) ApplicationSession(lease SessionLease) (ApplicationReadyPeerSession, error) {
	if err := lease.Validate(); err != nil {
		return nil, err
	}
	current, err := owner.session(lease.Stamp)
	if err != nil {
		return nil, err
	}
	return &ownedApplicationSession{
		owner: owner, stamp: lease.Stamp, observedPath: current.ObservedPath(),
		done: current.Done(), terminal: current,
	}, nil
}

// ExecuteApplication 把 generated Proto command 路由到 stamp 指定的当前 ready session。
// generation 不匹配时在进入 protocol adapter 前失败，非幂等 command 不会被自动重放到新 session。
func (owner *SessionOwner) ExecuteApplication(ctx context.Context, stamp EndpointSessionStamp, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	session, err := owner.session(stamp)
	if err != nil {
		return nil, err
	}
	return session.ExecuteApplication(ctx, command)
}

// ApplicationEvents 返回 stamp 指定 session 的 generated Proto event stream。
// session 被替换或关闭后底层 stream 必须结束；调用方必须用新 lease 显式重新订阅，不能跨 generation 拼接事件。
func (owner *SessionOwner) ApplicationEvents(ctx context.Context, stamp EndpointSessionStamp) (<-chan *apipb.EventEnvelope, error) {
	session, err := owner.session(stamp)
	if err != nil {
		return nil, err
	}
	return session.ApplicationEvents(ctx)
}

// Disconnect 只关闭 stamp 精确匹配的当前 generation。
// stale disconnect 不能误关后来建立的 session；关闭错误只属于被释放 session，不触发其他 route fallback。
func (owner *SessionOwner) Disconnect(_ context.Context, request DisconnectRequest) error {
	if err := request.Validate(); err != nil {
		return err
	}
	owner.mu.Lock()
	session := owner.current[request.Stamp.EndpointID]
	if session == nil || session.Stamp() != request.Stamp || !owner.authority.isCurrent(request.Stamp.EndpointID, request.Stamp.Generation, session) {
		owner.mu.Unlock()
		return runtimeError(ErrorStaleSession, "disconnect session stamp is not current", nil)
	}
	delete(owner.current, request.Stamp.EndpointID)
	delete(owner.selections, request.Stamp.EndpointID)
	owner.authority.removeCurrent(request.Stamp.EndpointID, session)
	owner.mu.Unlock()
	return session.Close()
}

// DisconnectEndpoint is the explicit user disconnect boundary. It serializes
// behind an in-flight acquire so a late ready winner cannot remain pooled, then
// advances the endpoint generation before closing the physical session.
func (owner *SessionOwner) DisconnectEndpoint(ctx context.Context, endpointID endpoint.EndpointID) error {
	if owner == nil || ctx == nil || strings.TrimSpace(string(endpointID)) == "" {
		return runtimeError(ErrorInvalidRequest, "session owner, context, and endpoint_id are required", nil)
	}
	unlock, err := owner.acquireEndpoint(ctx, endpointID)
	if err != nil {
		return err
	}
	defer unlock()

	owner.mu.Lock()
	owner.authority.mu.Lock()
	generation := owner.authority.generations[endpointID]
	if generation == SessionGeneration(math.MaxUint64) {
		owner.authority.mu.Unlock()
		owner.mu.Unlock()
		return runtimeError(ErrorUnavailable, "endpoint session generation is exhausted", nil)
	}
	session := owner.current[endpointID]
	globalSession := owner.authority.current[endpointID]
	owner.authority.generations[endpointID] = generation + 1
	delete(owner.authority.current, endpointID)
	owner.authority.mu.Unlock()
	delete(owner.current, endpointID)
	delete(owner.configs, endpointID)
	delete(owner.selections, endpointID)
	shared := owner.takeSharedLeasesLocked(endpointID)
	owner.mu.Unlock()

	cause := runtimeError(ErrorCanceled, "endpoint was disconnected by the user", nil)
	for _, lease := range shared {
		lease.finish(cause)
	}
	if globalSession != nil {
		return globalSession.Close()
	}
	if session != nil {
		return session.Close()
	}
	return nil
}

// InvalidateSessionFast removes one exact generation before closing its stale transport in
// the background. Network recovery uses this path so a dead route cannot delay the next
// OpenSession; normal Disconnect remains a synchronous, graceful user action.
func (owner *SessionOwner) InvalidateSessionFast(stamp EndpointSessionStamp, cause error) error {
	if err := stamp.Validate(); err != nil {
		return err
	}
	if cause == nil {
		cause = runtimeError(ErrorUnavailable, "application session was invalidated", nil)
	}
	owner.mu.Lock()
	owner.authority.mu.Lock()
	session := owner.current[stamp.EndpointID]
	if session == nil || session.Stamp() != stamp || owner.authority.generations[stamp.EndpointID] != stamp.Generation || owner.authority.current[stamp.EndpointID] != session {
		owner.authority.mu.Unlock()
		owner.mu.Unlock()
		return runtimeError(ErrorStaleSession, "invalidate session stamp is not current", nil)
	}
	if stamp.Generation == SessionGeneration(math.MaxUint64) {
		owner.authority.mu.Unlock()
		owner.mu.Unlock()
		return runtimeError(ErrorUnavailable, "endpoint session generation is exhausted", nil)
	}
	owner.authority.generations[stamp.EndpointID] = stamp.Generation + 1
	delete(owner.authority.current, stamp.EndpointID)
	owner.authority.mu.Unlock()
	delete(owner.current, stamp.EndpointID)
	delete(owner.configs, stamp.EndpointID)
	delete(owner.selections, stamp.EndpointID)
	shared := owner.takeSharedLeasesForSessionLocked(stamp.EndpointID, session)
	owner.mu.Unlock()
	for _, lease := range shared {
		lease.finish(cause)
	}
	go func() { _ = session.Close() }()
	return nil
}

// invalidateApplicationSession 在资源清理无法确认时撤销精确 session generation。
// 它关闭底层 ReadyPeerSession 并终止同 generation 的全部 shared lease，避免其它 consumer 让未知 daemon resource 继续存活。
func (owner *SessionOwner) invalidateApplicationSession(stamp EndpointSessionStamp, cause error) error {
	if err := stamp.Validate(); err != nil {
		return err
	}
	if cause == nil {
		cause = runtimeError(ErrorUnavailable, "application session was invalidated", nil)
	}
	owner.mu.Lock()
	session := owner.current[stamp.EndpointID]
	if session == nil || session.Stamp() != stamp || !owner.authority.isCurrent(stamp.EndpointID, stamp.Generation, session) {
		owner.mu.Unlock()
		return runtimeError(ErrorStaleSession, "invalidate session stamp is not current", nil)
	}
	delete(owner.current, stamp.EndpointID)
	delete(owner.configs, stamp.EndpointID)
	delete(owner.selections, stamp.EndpointID)
	shared := owner.takeSharedLeasesForSessionLocked(stamp.EndpointID, session)
	owner.authority.removeCurrent(stamp.EndpointID, session)
	owner.mu.Unlock()
	for _, lease := range shared {
		lease.finish(cause)
	}
	return session.Close()
}

// Close 关闭 owner 持有的全部当前 session 并禁止新连接。
// 方法可重复调用；它不修改 endpoint registry、credential store 或 daemon terminal lifecycle。
func (owner *SessionOwner) Close() error {
	if owner == nil {
		return nil
	}
	owner.mu.Lock()
	if owner.closed {
		owner.mu.Unlock()
		return nil
	}
	owner.closed = true
	close(owner.ownerDone)
	sessions := make([]ApplicationReadyPeerSession, 0, len(owner.current))
	for endpointID, session := range owner.current {
		sessions = append(sessions, session)
		owner.authority.removeCurrent(endpointID, session)
		delete(owner.current, endpointID)
		delete(owner.configs, endpointID)
		delete(owner.selections, endpointID)
	}
	shared := make([]*sharedApplicationLease, 0)
	for endpointID := range owner.sharedLeases {
		shared = append(shared, owner.takeSharedLeasesLocked(endpointID)...)
	}
	for endpointID, watchers := range owner.watchers {
		for watcher := range watchers {
			close(watcher)
		}
		delete(owner.watchers, endpointID)
	}
	owner.mu.Unlock()
	for _, lease := range shared {
		lease.finish(runtimeError(ErrorUnavailable, "session owner is closed", nil))
	}
	var first error
	for _, session := range sessions {
		if err := session.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (owner *SessionOwner) session(stamp EndpointSessionStamp) (ApplicationReadyPeerSession, error) {
	if owner == nil {
		return nil, runtimeError(ErrorUnavailable, "session owner is unavailable", nil)
	}
	if err := stamp.Validate(); err != nil {
		return nil, err
	}
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.closed {
		return nil, runtimeError(ErrorUnavailable, "session owner is closed", nil)
	}
	session := owner.current[stamp.EndpointID]
	if session == nil || session.Stamp() != stamp || !owner.authority.isCurrent(stamp.EndpointID, stamp.Generation, session) {
		return nil, runtimeError(ErrorStaleSession, fmt.Sprintf("endpoint %q session generation is stale", stamp.EndpointID), nil)
	}
	return session, nil
}

func (owner *SessionOwner) removeWhenDone(endpointID endpoint.EndpointID, generation SessionGeneration, session ApplicationReadyPeerSession) {
	<-session.Done()
	stamp := session.Stamp()
	err := session.Err()
	owner.mu.Lock()
	wasCurrent := generation == stamp.Generation && owner.current[endpointID] == session && owner.authority.isCurrent(endpointID, generation, session)
	if wasCurrent {
		delete(owner.current, endpointID)
		delete(owner.configs, endpointID)
		delete(owner.selections, endpointID)
	}
	// 旧 session 的 Done 可能晚于新 generation 发布；这里只能回收精确绑定
	// 当前 ready session 的 consumer lease，不能触碰后来 generation。
	shared := owner.takeSharedLeasesForSessionLocked(endpointID, session)
	owner.authority.removeCurrent(endpointID, session)
	owner.mu.Unlock()
	for _, lease := range shared {
		lease.finish(err)
	}
	_ = session.Close()
	if wasCurrent {
		owner.mu.Lock()
		owner.authority.mu.Lock()
		stillLatest := owner.authority.generations[endpointID] == generation && owner.authority.current[endpointID] == nil && owner.current[endpointID] == nil
		owner.authority.mu.Unlock()
		if stillLatest {
			owner.publishEndpointEventLocked(EndpointEvent{EndpointID: endpointID, Stamp: stamp, Phase: EndpointPhaseOffline, ErrorCode: CodeOf(err), Message: errorMessage(err)})
		}
		owner.mu.Unlock()
	}
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (owner *SessionOwner) newSharedLeaseLocked(endpointID endpoint.EndpointID, ready ApplicationReadyPeerSession) *sharedApplicationLease {
	lease := &sharedApplicationLease{owner: owner, endpointID: endpointID, ready: ready, done: make(chan struct{})}
	if owner.sharedLeases[endpointID] == nil {
		owner.sharedLeases[endpointID] = make(map[*sharedApplicationLease]struct{})
	}
	owner.sharedLeases[endpointID][lease] = struct{}{}
	go func() {
		select {
		case <-ready.Done():
			lease.finish(ready.Err())
		case <-lease.done:
		}
	}()
	return lease
}

func (owner *SessionOwner) releaseSharedLease(lease *sharedApplicationLease) {
	owner.mu.Lock()
	leases := owner.sharedLeases[lease.endpointID]
	delete(leases, lease)
	if len(leases) == 0 {
		delete(owner.sharedLeases, lease.endpointID)
	}
	owner.mu.Unlock()
}

func (owner *SessionOwner) takeSharedLeasesLocked(endpointID endpoint.EndpointID) []*sharedApplicationLease {
	leasing := owner.sharedLeases[endpointID]
	result := make([]*sharedApplicationLease, 0, len(leasing))
	for lease := range leasing {
		result = append(result, lease)
	}
	delete(owner.sharedLeases, endpointID)
	return result
}

func (owner *SessionOwner) takeSharedLeasesForSessionLocked(endpointID endpoint.EndpointID, ready ApplicationReadyPeerSession) []*sharedApplicationLease {
	leasing := owner.sharedLeases[endpointID]
	result := make([]*sharedApplicationLease, 0, len(leasing))
	for lease := range leasing {
		// 首个 AcquireRoute consumer 可能包装 ownedApplicationSession，后续 consumer 直接包装底层 ready；
		// stamp 才是两者共享的精确 generation identity，不能用 wrapper pointer 判断 ownership。
		if lease.ready.Stamp() != ready.Stamp() {
			continue
		}
		result = append(result, lease)
		delete(leasing, lease)
	}
	if len(leasing) == 0 {
		delete(owner.sharedLeases, endpointID)
	}
	return result
}

func (authority *SessionGenerationAuthority) isCurrent(endpointID endpoint.EndpointID, generation SessionGeneration, session ApplicationReadyPeerSession) bool {
	if authority == nil {
		return false
	}
	authority.mu.Lock()
	defer authority.mu.Unlock()
	return authority.generations[endpointID] == generation && authority.current[endpointID] == session
}

func (authority *SessionGenerationAuthority) removeCurrent(endpointID endpoint.EndpointID, session ApplicationReadyPeerSession) {
	if authority == nil {
		return
	}
	authority.mu.Lock()
	if authority.current[endpointID] == session {
		delete(authority.current, endpointID)
	}
	authority.mu.Unlock()
}

type ownedApplicationSession struct {
	owner        *SessionOwner
	stamp        EndpointSessionStamp
	observedPath string
	done         <-chan struct{}
	terminal     ApplicationReadyPeerSession
}

func (session *ownedApplicationSession) Stamp() EndpointSessionStamp { return session.stamp }
func (session *ownedApplicationSession) ObservedPath() string        { return session.observedPath }
func (session *ownedApplicationSession) Readiness() ReadyPeerSessionEvidence {
	return session.terminal.Readiness()
}
func (session *ownedApplicationSession) Done() <-chan struct{} { return session.done }
func (session *ownedApplicationSession) Err() error            { return session.terminal.Err() }

// ConnectionSnapshot 转发当前 owner winner 的即时诊断；stale wrapper 不返回旧 transport 数据。
func (session *ownedApplicationSession) ConnectionSnapshot(at time.Time) (ConnectionSnapshot, bool) {
	current, err := session.owner.session(session.stamp)
	if err != nil {
		return ConnectionSnapshot{}, false
	}
	provider, ok := current.(ConnectionSnapshotProvider)
	if !ok {
		return ConnectionSnapshot{}, false
	}
	snapshot, valid := provider.ConnectionSnapshot(at)
	if !valid {
		return ConnectionSnapshot{}, false
	}
	snapshot.SelectionReason = session.owner.selectionReason(session.stamp)
	return snapshot, true
}

func (owner *SessionOwner) selectionReason(stamp EndpointSessionStamp) string {
	owner.mu.Lock()
	defer owner.mu.Unlock()
	selection := owner.selections[stamp.EndpointID]
	if selection.generation != stamp.Generation {
		return ""
	}
	return selection.reason
}

func (session *ownedApplicationSession) Close() error {
	return session.owner.Disconnect(context.Background(), DisconnectRequest{Stamp: session.stamp})
}

// InvalidateApplicationSession 在资源清理失败时撤销 owned wrapper 绑定的精确 generation。
func (session *ownedApplicationSession) InvalidateApplicationSession(cause error) error {
	return session.owner.invalidateApplicationSession(session.stamp, cause)
}

func (session *ownedApplicationSession) ExecuteApplication(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return session.owner.ExecuteApplication(ctx, session.stamp, command)
}

// ValidateApplicationSession 在具体 protocol/resource 调用前确认 owned wrapper 仍是 owner 当前 winner。
func (session *ownedApplicationSession) ValidateApplicationSession(stamp EndpointSessionStamp) error {
	if stamp != session.stamp {
		return runtimeError(ErrorStaleSession, "application operation session stamp does not match owned session", nil)
	}
	_, err := session.owner.session(stamp)
	return err
}

// ExecuteApplicationTerminal 把 terminal-response 选择路由到同一 generation 的底层 ready session。
func (session *ownedApplicationSession) ExecuteApplicationTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	current, err := session.owner.session(session.stamp)
	if err != nil {
		return nil, err
	}
	executor, ok := current.(TerminalResponseApplicationExecutor)
	if !ok {
		return nil, runtimeError(ErrorUnavailable, "current session does not support terminal application responses", nil)
	}
	return executor.ExecuteApplicationTerminal(ctx, command)
}

func (session *ownedApplicationSession) ApplicationEvents(ctx context.Context) (<-chan *apipb.EventEnvelope, error) {
	return session.owner.ApplicationEvents(ctx, session.stamp)
}

func (session *ownedApplicationSession) OpenResourceStream(resource *apipb.ResourceHandle) (ResourceStream, error) {
	current, err := session.owner.session(session.stamp)
	if err != nil {
		return nil, err
	}
	provider, ok := current.(ResourceStreamSession)
	if !ok {
		return nil, runtimeError(ErrorUnavailable, "current session does not support resource streams", nil)
	}
	return provider.OpenResourceStream(resource)
}

func (session *ownedApplicationSession) ApplicationAttachmentChannel(resource *apipb.ResourceHandle) (uint16, bool) {
	current, err := session.owner.session(session.stamp)
	if err != nil {
		return 0, false
	}
	provider, ok := current.(ApplicationAttachmentSession)
	if !ok {
		return 0, false
	}
	return provider.ApplicationAttachmentChannel(resource)
}

func (session *ownedApplicationSession) ApplicationAttachment(channel uint16) (*apipb.ResourceHandle, bool) {
	current, err := session.owner.session(session.stamp)
	if err != nil {
		return nil, false
	}
	provider, ok := current.(ApplicationAttachmentSession)
	if !ok {
		return nil, false
	}
	return provider.ApplicationAttachment(channel)
}

var _ ApplicationReadyPeerSession = (*ownedApplicationSession)(nil)
var _ ResourceStreamSession = (*ownedApplicationSession)(nil)
var _ ApplicationAttachmentSession = (*ownedApplicationSession)(nil)
var _ ApplicationSessionValidator = (*ownedApplicationSession)(nil)
var _ ApplicationSessionInvalidator = (*ownedApplicationSession)(nil)

type sharedApplicationLease struct {
	owner      *SessionOwner
	endpointID endpoint.EndpointID
	ready      ApplicationReadyPeerSession
	done       chan struct{}
	closeOnce  sync.Once
	errMu      sync.Mutex
	err        error
}

func (lease *sharedApplicationLease) Stamp() EndpointSessionStamp { return lease.ready.Stamp() }
func (lease *sharedApplicationLease) ObservedPath() string        { return lease.ready.ObservedPath() }
func (lease *sharedApplicationLease) Readiness() ReadyPeerSessionEvidence {
	return lease.ready.Readiness()
}
func (lease *sharedApplicationLease) Done() <-chan struct{} { return lease.done }
func (lease *sharedApplicationLease) Err() error {
	lease.errMu.Lock()
	defer lease.errMu.Unlock()
	return lease.err
}

// ConnectionSnapshot 只在 consumer lease 仍有效时转发底层 ReadySession 的即时诊断。
func (lease *sharedApplicationLease) ConnectionSnapshot(at time.Time) (ConnectionSnapshot, bool) {
	if lease.active() != nil {
		return ConnectionSnapshot{}, false
	}
	provider, ok := lease.ready.(ConnectionSnapshotProvider)
	if !ok {
		return ConnectionSnapshot{}, false
	}
	snapshot, valid := provider.ConnectionSnapshot(at)
	if !valid {
		return ConnectionSnapshot{}, false
	}
	snapshot.SelectionReason = lease.owner.selectionReason(lease.ready.Stamp())
	return snapshot, true
}
func (lease *sharedApplicationLease) Close() error {
	lease.closeOnce.Do(func() {
		lease.owner.releaseSharedLease(lease)
		close(lease.done)
	})
	return nil
}

// InvalidateApplicationSession 在资源清理失败时撤销 shared lease 所属的完整 generation，而不是只释放当前 consumer。
func (lease *sharedApplicationLease) InvalidateApplicationSession(cause error) error {
	return lease.owner.invalidateApplicationSession(lease.ready.Stamp(), cause)
}
func (lease *sharedApplicationLease) finish(err error) {
	lease.closeOnce.Do(func() {
		lease.errMu.Lock()
		lease.err = err
		lease.errMu.Unlock()
		lease.owner.releaseSharedLease(lease)
		close(lease.done)
	})
}
func (lease *sharedApplicationLease) ExecuteApplication(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	if err := lease.active(); err != nil {
		return nil, err
	}
	return lease.ready.ExecuteApplication(ctx, command)
}

// ValidateApplicationSession 在副作用前确认 consumer lease 未关闭且请求 stamp 精确属于共享 ready generation。
func (lease *sharedApplicationLease) ValidateApplicationSession(stamp EndpointSessionStamp) error {
	if err := lease.active(); err != nil {
		return err
	}
	if stamp != lease.ready.Stamp() {
		return runtimeError(ErrorStaleSession, "application operation session stamp does not match shared session", nil)
	}
	return nil
}

// ExecuteApplicationTerminal 保持 shared lease generation fence，并把 terminal response 交给同一 ready session。
func (lease *sharedApplicationLease) ExecuteApplicationTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	if err := lease.active(); err != nil {
		return nil, err
	}
	executor, ok := lease.ready.(TerminalResponseApplicationExecutor)
	if !ok {
		return nil, runtimeError(ErrorUnavailable, "shared session does not support terminal application responses", nil)
	}
	return executor.ExecuteApplicationTerminal(ctx, command)
}
func (lease *sharedApplicationLease) ApplicationEvents(ctx context.Context) (<-chan *apipb.EventEnvelope, error) {
	if err := lease.active(); err != nil {
		return nil, err
	}
	eventCtx, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-lease.done:
			cancel()
		case <-eventCtx.Done():
		}
	}()
	return lease.ready.ApplicationEvents(eventCtx)
}
func (lease *sharedApplicationLease) OpenResourceStream(resource *apipb.ResourceHandle) (ResourceStream, error) {
	if err := lease.active(); err != nil {
		return nil, err
	}
	provider, ok := lease.ready.(ResourceStreamSession)
	if !ok {
		return nil, runtimeError(ErrorUnavailable, "shared session does not support resource streams", nil)
	}
	return provider.OpenResourceStream(resource)
}
func (lease *sharedApplicationLease) ApplicationAttachmentChannel(resource *apipb.ResourceHandle) (uint16, bool) {
	if err := lease.active(); err != nil {
		return 0, false
	}
	provider, ok := lease.ready.(ApplicationAttachmentSession)
	if !ok {
		return 0, false
	}
	return provider.ApplicationAttachmentChannel(resource)
}
func (lease *sharedApplicationLease) ApplicationAttachment(channel uint16) (*apipb.ResourceHandle, bool) {
	if err := lease.active(); err != nil {
		return nil, false
	}
	provider, ok := lease.ready.(ApplicationAttachmentSession)
	if !ok {
		return nil, false
	}
	return provider.ApplicationAttachment(channel)
}

var _ ApplicationSessionInvalidator = (*sharedApplicationLease)(nil)

func (lease *sharedApplicationLease) active() error {
	select {
	case <-lease.done:
		return runtimeError(ErrorStaleSession, "shared session lease is closed", lease.Err())
	default:
		return nil
	}
}

var _ ApplicationReadyPeerSession = (*sharedApplicationLease)(nil)
var _ ResourceStreamSession = (*sharedApplicationLease)(nil)
var _ ApplicationAttachmentSession = (*sharedApplicationLease)(nil)
var _ ApplicationSessionValidator = (*sharedApplicationLease)(nil)

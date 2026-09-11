package core

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

const pluginQueueLimit = 64
const pluginRegistrationLimit = 256
const pluginMessageLimit = 256 * 1024
const pluginPendingLimit = 2048

// WithPluginRuntime sets the public daemon identity and state directory for an
// embedding host (tests use private temporary directories). Production normally
// obtains identity from the already configured ClientAccessService.
func WithPluginRuntime(daemonID, stateDir string) ServerOption {
	return func(c *serverConfig) { c.pluginIdentity = daemonID; c.pluginStateDir = stateDir }
}

type pluginRegistration struct {
	address             *apipb.PluginAddress
	lease               []byte
	owner               *protocolSession
	principal           string
	service, allowPeers bool
	topics              map[string]bool
	watches             map[string]bool
	queue               []*apipb.PluginMessage
	notify              chan struct{}
	closed              bool
	resync              bool
	receiving           bool
	queuedBytes         int
}
type pluginPending struct {
	source, target *pluginRegistration
	expires        time.Time
}
type pluginContext struct {
	source, target *pluginRegistration
	value          *apipb.PluginTargetContext
	expires        time.Time
}
type pluginDedupe struct {
	hash    [32]byte
	result  *apipb.PluginResult
	expires time.Time
}
type pluginRouter struct {
	mu                       sync.Mutex
	daemonID, boot, stateDir string
	epoch, sequence          uint64
	queuedBytes              int
	registrations            map[string]*pluginRegistration
	pending                  map[string]pluginPending
	contexts                 map[string]pluginContext
	dedupe                   map[string]pluginDedupe
	states                   map[string]*apipb.PluginStateSnapshot
}

func randomPluginToken() []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}
func randomPluginID() string { return hex.EncodeToString(randomPluginToken()) }
func pluginError(code, message string) *apipb.PluginResult {
	return &apipb.PluginResult{Result: &apipb.PluginResult_Error{Error: &apipb.PluginError{Code: code, Message: message}}}
}
func pluginAck(count int) *apipb.PluginResult {
	return &apipb.PluginResult{Result: &apipb.PluginResult_Ack{Ack: &apipb.PluginAck{Delivered: uint32(count)}}}
}
func pluginName(s string) bool {
	if len(s) == 0 || len(s) > 160 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-') {
			return false
		}
	}
	return s != "." && s != ".."
}
func cloneAddress(a *apipb.PluginAddress) *apipb.PluginAddress {
	if a == nil {
		return nil
	}
	return proto.Clone(a).(*apipb.PluginAddress)
}
func (s *Server) pluginRuntime(ctx context.Context) (*pluginRouter, error) {
	s.pluginMu.Lock()
	defer s.pluginMu.Unlock()
	if s.plugins != nil {
		return s.plugins, nil
	}
	identity := s.cfg.pluginIdentity
	if identity == "" {
		service := s.ClientAccessService()
		if service == nil {
			return nil, ErrClientAccessServiceUnavailable
		}
		value, err := service.Identity(ctx, randomPluginToken())
		if err != nil {
			return nil, err
		}
		identity = value.DeviceID
	}
	if identity == "" {
		return nil, errors.New("daemon identity unavailable")
	}
	dir := s.cfg.pluginStateDir
	if dir == "" {
		sum := sha256.Sum256([]byte(identity))
		dir = filepath.Join(filepath.Dir(s.cfg.socketPath), "plugin-state", hex.EncodeToString(sum[:]))
	}
	s.plugins = &pluginRouter{daemonID: identity, boot: randomPluginID(), stateDir: dir, registrations: map[string]*pluginRegistration{}, pending: map[string]pluginPending{}, contexts: map[string]pluginContext{}, dedupe: map[string]pluginDedupe{}, states: map[string]*apipb.PluginStateSnapshot{}}
	return s.plugins, nil
}
func (s *protocolSession) ApplicationPlugin(ctx context.Context, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
	if _, err := s.AcquireApplication(ctx, ApplicationAdmission{Capability: ApplicationCapabilityPlugin}); err != nil {
		return nil, err
	}
	if command == nil || command.GetCommand() == nil || proto.Size(command) > pluginMessageLimit {
		return pluginError("INVALID_REQUEST", "missing or oversized plugin command"), nil
	}
	router, err := s.server.pluginRuntime(ctx)
	if err != nil {
		return pluginError("UNAVAILABLE", err.Error()), nil
	}
	if receive := command.GetReceive(); receive != nil {
		return router.receive(ctx, s, receive), nil
	}
	router.mu.Lock()
	defer router.mu.Unlock()
	router.pruneLocked()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch request := command.GetCommand().(type) {
	case *apipb.PluginCommand_Register:
		return router.register(s, request.Register), nil
	case *apipb.PluginCommand_Send:
		return router.send(s, request.Send), nil
	case *apipb.PluginCommand_Unregister:
		reg := router.authorized(s, request.Unregister.GetSourceLease())
		if reg == nil {
			return pluginError("STALE_CONTEXT", "registration is not current"), nil
		}
		router.remove(reg)
		return pluginAck(0), nil
	case *apipb.PluginCommand_State:
		return router.state(s, request.State), nil
	default:
		return pluginError("UNSUPPORTED", "plugin command is unsupported"), nil
	}
}
func (r *pluginRouter) authorized(s *protocolSession, lease []byte) *pluginRegistration {
	reg := r.registrations[string(lease)]
	if reg == nil || reg.owner != s || reg.closed || (s.sessionCtx != nil && s.sessionCtx.Err() != nil) {
		return nil
	}
	return reg
}
func (r *pluginRouter) register(s *protocolSession, request *apipb.PluginRegisterRequest) *apipb.PluginResult {
	address := request.GetAddress()
	if address == nil || !pluginName(address.GetPluginId()) || !pluginName(address.GetPluginInstanceId()) || address.GetTuiInstanceId() != "" && !pluginName(address.GetTuiInstanceId()) || len(request.GetTopics()) > 32 {
		return pluginError("INVALID_REQUEST", "valid plugin and instance identities are required")
	}
	if address.GetDaemonId() != "" && address.GetDaemonId() != r.daemonID {
		return pluginError("INVALID_REQUEST", "wrong daemon routing domain")
	}
	if request.GetDaemonService() && (!s.scope.LocalOwner || address.GetTuiInstanceId() != "") {
		return pluginError("PERMISSION_DENIED", "daemon service requires local owner and no TUI identity")
	}
	for _, topic := range request.GetTopics() {
		if !pluginName(topic) || !strings.HasPrefix(topic, "plugin."+address.GetPluginId()+".") {
			return pluginError("PERMISSION_DENIED", "topic must belong to plugin namespace")
		}
	}
	var replacing *pluginRegistration
	for _, old := range r.registrations {
		same := old.address.GetPluginId() == address.GetPluginId() && old.address.GetPluginInstanceId() == address.GetPluginInstanceId() && old.address.GetTuiInstanceId() == address.GetTuiInstanceId()
		if !same {
			continue
		}
		if old.owner == s {
			return r.registrationResult(old)
		}
		if old.principal != s.scope.PrincipalID || string(request.GetPreviousLease()) != string(old.lease) {
			return pluginError("CONFLICT", "registration exists; authenticated handoff required")
		}
		replacing = old
	}
	for _, old := range r.registrations {
		if old == replacing {
			continue
		}
		if request.GetDaemonService() && old.service && old.address.GetPluginId() == address.GetPluginId() {
			return pluginError("CONFLICT", "daemon plugin service already registered")
		}
		if replacing == nil && address.GetTuiInstanceId() != "" && old.address.GetTuiInstanceId() == address.GetTuiInstanceId() && old.owner != s {
			return pluginError("CONFLICT", "TUI identity belongs to another connection")
		}
	}
	if replacing != nil {
		r.remove(replacing)
	}

	if len(r.registrations) >= pluginRegistrationLimit {
		return pluginError("RESOURCE_EXHAUSTED", "registration capacity reached")
	}
	r.epoch++
	a := cloneAddress(address)
	a.DaemonId = r.daemonID
	a.RegistrationEpoch = r.epoch
	reg := &pluginRegistration{address: a, lease: randomPluginToken(), owner: s, principal: s.scope.PrincipalID, service: request.GetDaemonService(), allowPeers: request.GetAllowPeerMessages(), topics: map[string]bool{}, watches: map[string]bool{}, notify: make(chan struct{}, 1)}
	for _, topic := range request.GetTopics() {
		reg.topics[topic] = true
	}
	r.registrations[string(reg.lease)] = reg
	return r.registrationResult(reg)
}
func (r *pluginRouter) registrationResult(reg *pluginRegistration) *apipb.PluginResult {
	value := &apipb.PluginRegistration{Address: cloneAddress(reg.address), SourceLease: append([]byte(nil), reg.lease...), BootEpoch: r.boot}
	for _, peer := range r.registrations {
		if peer != reg && peer.owner == reg.owner && peer.address.GetTuiInstanceId() == reg.address.GetTuiInstanceId() && peer.address.GetPluginId() == reg.address.GetPluginId() {
			value.Peers = append(value.Peers, cloneAddress(peer.address))
		}
	}
	return &apipb.PluginResult{Result: &apipb.PluginResult_Registration{Registration: value}}
}
func (r *pluginRouter) remove(reg *pluginRegistration) {
	delete(r.registrations, string(reg.lease))
	reg.closed = true
	r.queuedBytes -= reg.queuedBytes
	reg.queuedBytes = 0
	reg.queue = nil
	r.signal(reg)
	for key, p := range r.pending {
		if p.source == reg || p.target == reg {
			delete(r.pending, key)
		}
	}
	for key, c := range r.contexts {
		if c.source == reg || c.target == reg {
			delete(r.contexts, key)
		}
	}
}
func (r *pluginRouter) signal(reg *pluginRegistration) {
	select {
	case reg.notify <- struct{}{}:
	default:
	}
}
func (r *pluginRouter) pruneLocked() {
	now := time.Now()
	for key, p := range r.pending {
		if now.After(p.expires) {
			delete(r.pending, key)
		}
	}
	for key, c := range r.contexts {
		if now.After(c.expires) {
			delete(r.contexts, key)
		}
	}
	for key, d := range r.dedupe {
		if now.After(d.expires) {
			delete(r.dedupe, key)
		}
	}
}
func (s *protocolSession) releasePluginRegistrations() {
	s.server.pluginMu.Lock()
	router := s.server.plugins
	s.server.pluginMu.Unlock()
	if router == nil {
		return
	}
	router.mu.Lock()
	defer router.mu.Unlock()
	for _, reg := range router.registrations {
		if reg.owner == s {
			router.remove(reg)
		}
	}
}
func (r *pluginRouter) target(a *apipb.PluginAddress) *pluginRegistration {
	if a == nil || a.GetDaemonId() != r.daemonID {
		return nil
	}
	for _, reg := range r.registrations {
		if reg.owner.sessionCtx != nil && reg.owner.sessionCtx.Err() != nil {
			continue
		}
		if reg.address.GetPluginId() != a.GetPluginId() || reg.address.GetTuiInstanceId() != a.GetTuiInstanceId() {
			continue
		}
		if a.GetPluginInstanceId() == "" {
			if reg.service && a.GetRegistrationEpoch() == 0 {
				return reg
			}
			continue
		}
		if reg.address.GetPluginInstanceId() == a.GetPluginInstanceId() && reg.address.GetRegistrationEpoch() == a.GetRegistrationEpoch() {
			return reg
		}
	}
	return nil
}
func (r *pluginRouter) enqueue(reg *pluginRegistration, message *apipb.PluginMessage) bool {
	size := proto.Size(message) + 16
	if reg.closed || len(reg.queue) >= pluginQueueLimit || reg.queuedBytes+size > 1024*1024 || r.queuedBytes+size > 32*1024*1024 {
		return false
	}
	r.sequence++
	copy := proto.Clone(message).(*apipb.PluginMessage)
	copy.DeliverySequence = r.sequence
	reg.queue = append(reg.queue, copy)
	reg.queuedBytes += proto.Size(copy) + 16
	r.queuedBytes += proto.Size(copy) + 16
	r.signal(reg)
	return true
}
func pluginRequestKey(reg *pluginRegistration, id string) string { return string(reg.lease) + "/" + id }
func (r *pluginRouter) send(s *protocolSession, request *apipb.PluginSendRequest) *apipb.PluginResult {
	source := r.authorized(s, request.GetSourceLease())
	if source == nil {
		return pluginError("STALE_CONTEXT", "registration is not current")
	}
	original := request.GetMessage()
	if original == nil || original.GetBody() == nil || len(original.GetRequestId()) > 160 || len(original.GetIdempotencyKey()) > 160 || len(original.GetTraceId()) > 160 {
		return pluginError("INVALID_REQUEST", "typed message body and bounded identifiers required")
	}
	if original.GetSource() != nil {
		return pluginError("PERMISSION_DENIED", "source identity is daemon-stamped")
	}
	message := proto.Clone(original).(*apipb.PluginMessage)
	message.Source = cloneAddress(source.address)
	message.DeliverySequence = 0
	now := time.Now()
	if message.GetDeadlineUnixMillis() == 0 {
		message.DeadlineUnixMillis = now.Add(30 * time.Second).UnixMilli()
	}
	if message.GetDeadlineUnixMillis() <= now.UnixMilli() {
		return pluginError("DEADLINE_EXCEEDED", "message deadline expired")
	}
	if message.GetDeadlineUnixMillis() > now.Add(2*time.Minute).UnixMilli() {
		return pluginError("INVALID_REQUEST", "message deadline exceeds two minutes")
	}
	hashBytes, _ := proto.MarshalOptions{Deterministic: true}.Marshal(original)
	hash := sha256.Sum256(hashBytes)
	dedupeKey := pluginRequestKey(source, message.GetIdempotencyKey())
	if message.GetIdempotencyKey() != "" {
		if old, ok := r.dedupe[dedupeKey]; ok {
			if old.hash != hash {
				return pluginError("CONFLICT", "idempotency key reused with different content")
			}
			return proto.Clone(old.result).(*apipb.PluginResult)
		}
		if len(r.dedupe) >= pluginPendingLimit {
			return pluginError("RESOURCE_EXHAUSTED", "idempotency capacity reached")
		}
	}
	result := r.route(source, message)
	if message.GetIdempotencyKey() != "" && result.GetError() == nil {
		r.dedupe[dedupeKey] = pluginDedupe{hash: hash, result: proto.Clone(result).(*apipb.PluginResult), expires: now.Add(2 * time.Minute)}
	}
	return result
}
func (r *pluginRouter) route(source *pluginRegistration, m *apipb.PluginMessage) *apipb.PluginResult {
	if m.GetTopic() != "" {
		if m.GetDestination() != nil || !strings.HasPrefix(m.GetTopic(), "plugin."+source.address.GetPluginId()+".") || !pluginName(m.GetTopic()) {
			return pluginError("PERMISSION_DENIED", "invalid broadcast namespace")
		}
		if m.GetAgentSnapshot() == nil && m.GetPayload() == nil {
			return pluginError("PERMISSION_DENIED", "commands and replies cannot be broadcast")
		}
		count := 0
		for _, target := range r.registrations {
			if target.topics[m.GetTopic()] && (source.service || source.principal == target.principal) {
				if r.enqueue(target, m) {
					count++
				} else {
					target.resync = true
					r.signal(target)
				}
			}
		}
		return pluginAck(count)
	}
	target := r.target(m.GetDestination())
	if target == nil {
		return pluginError("TARGET_OFFLINE", "target registration is unavailable or stale")
	}
	if target.address.GetPluginId() != source.address.GetPluginId() {
		return pluginError("PERMISSION_DENIED", "cross-plugin delivery requires an explicit capability")
	}
	reply := m.GetReply()
	if reply != nil {
		key := pluginRequestKey(target, reply.GetRequestId())
		pending, ok := r.pending[key]
		if !ok || pending.target != source || pending.source != target {
			return pluginError("PERMISSION_DENIED", "reply does not match this source's pending request")
		}
		if !r.enqueue(target, m) {
			return pluginError("RESOURCE_EXHAUSTED", "target mailbox is full")
		}
		delete(r.pending, key)
		return pluginAck(1)
	}
	sameTUI := source.address.GetTuiInstanceId() != "" && source.address.GetTuiInstanceId() == target.address.GetTuiInstanceId() && source.owner == target.owner
	if !sameTUI && !target.service && !(target.allowPeers && source.principal == target.principal) {
		// An interaction's short-lived context grants only the captured UI target.
		c, ok := r.contexts[m.GetOperation().GetContext().GetContextId()]
		if !ok || c.target != source || c.source != target {
			return pluginError("PERMISSION_DENIED", "target has not authorized peer messages")
		}
	}
	if report := m.GetAgentReport(); report != nil && (!target.service || report.GetTerminal().GetDaemonId() != r.daemonID) {
		return pluginError("PERMISSION_DENIED", "agent report must target owning daemon service")
	}
	var newContext *pluginContext
	if interaction := m.GetInteraction(); interaction != nil {
		if source.address.GetTuiInstanceId() == "" || interaction.GetContext() == nil || interaction.GetContext().GetTuiInstanceId() != source.address.GetTuiInstanceId() {
			return pluginError("PERMISSION_DENIED", "interaction must originate in its registered TUI")
		}
		// The TUI already captured and indexed this context before sending it.
		// Its ID is correlation, not authority: the registration-bound record
		// below grants the exact responder access to the immutable target only.
		contextID := interaction.GetContext().GetContextId()
		if !pluginName(contextID) {
			return pluginError("INVALID_REQUEST", "a bounded host interaction context ID is required")
		}
		if _, exists := r.contexts[contextID]; exists {
			return pluginError("CONFLICT", "interaction context ID is already registered")
		}
		if len(r.contexts) >= pluginPendingLimit {
			return pluginError("RESOURCE_EXHAUSTED", "interaction context capacity reached")
		}
		newContext = &pluginContext{source: source, target: target, value: proto.Clone(interaction.Context).(*apipb.PluginTargetContext), expires: time.UnixMilli(m.GetDeadlineUnixMillis())}
	}
	if operation := m.GetOperation(); operation != nil && operation.GetBind() != nil {
		c, ok := r.contexts[operation.GetContext().GetContextId()]
		if !ok || c.target != source || c.source != target || !proto.Equal(c.value, operation.GetContext()) {
			return pluginError("STALE_CONTEXT", "bind requires the original daemon-registered interaction context")
		}
		if operation.GetBind().GetTerminal().GetDaemonId() != r.daemonID {
			return pluginError("PERMISSION_DENIED", "terminal must belong to routing daemon")
		}
	}
	if update := m.GetMountUpdate(); update != nil {
		if !sameTUI || target.address.GetTuiInstanceId() == "" {
			return pluginError("PERMISSION_DENIED", "mount updates target only their owning TUI")
		}
		if !pluginName(update.GetMountId()) || update.GetOwner() == nil || update.GetOwner().GetOwner() == nil {
			return pluginError("INVALID_REQUEST", "mount identity and owner are required")
		}
	}
	if m.GetStateChanged() != nil {
		return pluginError("PERMISSION_DENIED", "state events are daemon generated")
	}
	if m.GetRequestId() != "" {
		key := pluginRequestKey(source, m.GetRequestId())
		if _, exists := r.pending[key]; exists {
			return pluginError("CONFLICT", "request ID already pending")
		}
		if len(r.pending) >= pluginPendingLimit {
			return pluginError("RESOURCE_EXHAUSTED", "pending request capacity reached")
		}
	}
	if !r.enqueue(target, m) {
		return pluginError("RESOURCE_EXHAUSTED", "target mailbox is full")
	}
	if newContext != nil {
		r.contexts[newContext.value.ContextId] = *newContext
	}
	if m.GetRequestId() != "" {
		r.pending[pluginRequestKey(source, m.GetRequestId())] = pluginPending{source: source, target: target, expires: time.UnixMilli(m.GetDeadlineUnixMillis())}
	}
	return pluginAck(1)
}
func (r *pluginRouter) receive(ctx context.Context, s *protocolSession, request *apipb.PluginReceiveRequest) *apipb.PluginResult {
	r.mu.Lock()
	reg := r.authorized(s, request.GetSourceLease())
	if reg == nil {
		r.mu.Unlock()
		return pluginError("STALE_CONTEXT", "registration is not current")
	}
	if reg.receiving {
		r.mu.Unlock()
		return pluginError("CONFLICT", "only one receive per registration may be active")
	}
	reg.receiving = true
	r.mu.Unlock()
	defer func() { r.mu.Lock(); reg.receiving = false; r.mu.Unlock() }()
	wait := request.GetWaitMillis()
	if wait > 25000 {
		wait = 25000
	}
	timer := time.NewTimer(time.Duration(wait) * time.Millisecond)
	defer timer.Stop()
	max := int(request.GetMaxMessages())
	if max <= 0 || max > 32 {
		max = 32
	}
	for {
		r.mu.Lock()
		if reg.closed {
			r.mu.Unlock()
			return pluginError("STALE_CONTEXT", "registration closed")
		}
		if len(reg.queue) > 0 || reg.resync {
			n := len(reg.queue)
			if n > max {
				n = max
			}
			messages := append([]*apipb.PluginMessage(nil), reg.queue[:n]...)
			for _, delivered := range reg.queue[:n] {
				size := proto.Size(delivered) + 16
				reg.queuedBytes -= size
				r.queuedBytes -= size
			}
			copy(reg.queue, reg.queue[n:])
			clear(reg.queue[len(reg.queue)-n:])
			reg.queue = reg.queue[:len(reg.queue)-n]
			batch := &apipb.PluginBatch{Messages: messages, ResyncRequired: reg.resync}
			reg.resync = false
			r.mu.Unlock()
			return &apipb.PluginResult{Result: &apipb.PluginResult_Batch{Batch: batch}}
		}
		r.mu.Unlock()
		select {
		case <-ctx.Done():
			return pluginError("CANCELLED", ctx.Err().Error())
		case <-timer.C:
			return &apipb.PluginResult{Result: &apipb.PluginResult_Batch{Batch: &apipb.PluginBatch{}}}
		case <-reg.notify:
		}
	}
}

// PluginRouteCount is a diagnostic watermark, not an event recovery cursor.
func (s *Server) PluginRouteCount() uint64 {
	s.pluginMu.Lock()
	r := s.plugins
	s.pluginMu.Unlock()
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sequence
}

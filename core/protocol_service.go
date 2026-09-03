package core

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/perftrace"
	"github.com/anytty/anytty/shared/terminalmeta"
	"github.com/anytty/anytty/shared/transport"
)

const (
	protocolErrorBadRequest  = 400
	protocolErrorForbidden   = 403
	protocolErrorNotFound    = 404
	protocolErrorExhausted   = 429
	protocolErrorUnavailable = 503
	protocolErrorInternal    = 500
)

const protocolRequestCapacityExhaustedMessage = "protocol in-flight request capacity is exhausted"

const daemonBoundaryReclaimMinHeapMBEnv = "ANYTTY_DAEMON_REQUEST_RECLAIM_MIN_HEAP_MB"
const daemonBoundaryReclaimDefaultMinHeapBytes = 0

var errProtocolAttachmentMismatch = errors.New("protocol attachment mismatch")
var daemonBoundaryReclaimMinHeapBytes = parseDaemonBoundaryReclaimMinHeapBytes()
var daemonBoundaryReclaimLastHeapSys atomic.Uint64

func parseDaemonBoundaryReclaimMinHeapBytes() uint64 {
	raw := strings.TrimSpace(os.Getenv(daemonBoundaryReclaimMinHeapMBEnv))
	if raw == "" {
		// 中文说明：protocol request 是交互热路径，默认不能在 response 边界
		// 同步执行 debug.FreeOSMemory；大历史压测后它会把 create/attach/list
		// 这类后续请求串行卡住。需要诊断 RSS 回收时显式设置 env 开启。
		return daemonBoundaryReclaimDefaultMinHeapBytes
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return value << 20
}

func maybeReclaimDaemonBoundaryHeap() {
	if daemonBoundaryReclaimMinHeapBytes == 0 {
		return
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	idleUnreleased := uint64(0)
	if mem.HeapIdle > mem.HeapReleased {
		idleUnreleased = mem.HeapIdle - mem.HeapReleased
	}
	if mem.HeapAlloc < daemonBoundaryReclaimMinHeapBytes && idleUnreleased < daemonBoundaryReclaimMinHeapBytes {
		return
	}
	if !claimDaemonBoundaryReclaimHeapSys(mem.HeapSys) {
		return
	}
	// 中文说明：这是 request/terminal 批次边界上的显式 runtime page 归还，只在
	// history/snapshot response 已发送或 history 批次已落盘后运行；它不删除任何
	// history truth，也不是后台定时 scrub。
	debug.FreeOSMemory()
}

func claimDaemonBoundaryReclaimHeapSys(heapSys uint64) bool {
	for {
		last := daemonBoundaryReclaimLastHeapSys.Load()
		if last != 0 && heapSys < last+daemonBoundaryReclaimMinHeapBytes {
			return false
		}
		if daemonBoundaryReclaimLastHeapSys.CompareAndSwap(last, heapSys) {
			return true
		}
	}
}

type protocolSession struct {
	server                        *Server
	conn                          transport.Transport
	scope                         TransportScope
	application                   ApplicationExecutor
	sessionID                     uint64
	sessionCtx                    context.Context
	sendMu                        sync.Mutex
	attachmentMu                  sync.Mutex
	mu                            sync.RWMutex
	attachments                   map[uint16]protocolAttachment
	attachmentTokens              map[string]uint16
	eventSubscriptions            map[uint64]applicationEventSubscription
	rawPTYStreams                 map[uint16]*protocolRawPTYStream
	nextEventSub                  uint64
	requests                      sync.WaitGroup
	requestSlots                  chan struct{}
	requestMu                     sync.Mutex
	activeRequests                map[uint64]context.CancelFunc
	liveBaselineMu                sync.Mutex
	liveBaselines                 map[string]*sessionLiveScreenBaselines
	liveBaselineBytes             int64
	liveBaselineTimer             *time.Timer
	liveBaselineClosed            bool
	resourceMu                    sync.Mutex
	nextChannel                   uint32
	channelKinds                  map[uint16]protocolChannelKind
	attachmentCount               int
	fileTransferCount             int
	eventSubscriptionCount        int
	historyTokenReservations      int
	historyTokens                 map[protocolHistoryResourceKey]struct{}
	fileMu                        sync.Mutex
	fileChannels                  map[uint16]*sessionFileTransfer
	fileIDs                       map[string]uint16
	browserMu                     sync.Mutex
	browserChannels               map[uint16]*sessionBrowserProxy
	browserTokens                 map[string]uint16
	browserCount                  int
	lifecycleObserver             TransportLifecycleObserver
	helloAccepted                 bool
	beforeGlobalAttachmentPublish func(protocolAttachment)
}

type protocolChannelKind uint8

const (
	protocolChannelAttachment protocolChannelKind = iota + 1
	protocolChannelFileTransfer
	protocolChannelBrowserProxy
)

type protocolHistoryResourceKey struct {
	TerminalID string
	Token      history.HistoryToken
}

type applicationEventSubscription struct {
	cancel context.CancelFunc
	filter EventFilter
}

type sessionBrowserProxy struct {
	channel     uint16
	token       []byte
	conn        net.Conn
	writeMu     sync.Mutex
	forwardOnce sync.Once
	closeOnce   sync.Once
}

func (proxy *sessionBrowserProxy) close() {
	if proxy == nil || proxy.conn == nil {
		return
	}
	proxy.closeOnce.Do(func() { _ = proxy.conn.Close() })
}

// protocolAttachment 是 daemon-side channel/view registry；它不保存 TUI workspace/pane truth。
type protocolAttachment struct {
	SessionID    uint64
	TerminalID   string
	Channel      uint16
	Mode         string
	ResizePolicy string
	SurfaceID    string
	ViewID       string
	Epoch        uint64
	Token        []byte
}

type protocolAttachmentKey struct {
	SessionID uint64
	Channel   uint16
}

type attachmentRequest struct {
	TerminalID   string
	Mode         string
	ResizePolicy string
	SurfaceID    string
	ViewID       string
}

type attachmentDetachRequest struct {
	TerminalID string
	Channel    uint16
	SurfaceID  string
	ViewID     string
}

type attachmentInputRequest struct {
	TerminalID string
	Channel    uint16
	SurfaceID  string
	ViewID     string
	Data       []byte
}

type attachmentResizeControlRequest struct {
	TerminalID   string
	Channel      uint16
	ResizePolicy string
	SurfaceID    string
	ViewID       string
}

type attachmentResizeOwnership struct {
	OwnerAttachmentID string
	OwnerSurfaceID    string
	OwnerViewID       string
	Size              Size
	SizeLocked        bool
	Epoch             uint64
}

type attachmentResizeControl struct {
	CanResize       bool
	Reason          string
	SizeLocked      bool
	SurfaceID       string
	OwnerSurfaceID  string
	OwnerViewID     string
	ResizeOwnership *attachmentResizeOwnership
}

const (
	attachmentResizePolicyOwner    = "owner"
	attachmentResizePolicyFollower = "follower"
	attachmentResizePolicyObserver = "observer"
)

const (
	attachmentResizeReasonOwner      = "owner"
	attachmentResizeReasonFollower   = "follower"
	attachmentResizeReasonObserver   = "observer"
	attachmentResizeReasonSizeLocked = "size_locked"
)

func protocolAttachmentEqual(left, right protocolAttachment) bool {
	return left.SessionID == right.SessionID &&
		left.TerminalID == right.TerminalID &&
		left.Channel == right.Channel &&
		left.Mode == right.Mode &&
		left.ResizePolicy == right.ResizePolicy &&
		left.SurfaceID == right.SurfaceID &&
		left.ViewID == right.ViewID &&
		left.Epoch == right.Epoch &&
		bytes.Equal(left.Token, right.Token)
}

func newProtocolSession(server *Server, conn transport.Transport, scope TransportScope) *protocolSession {
	return newProtocolSessionObserved(server, conn, scope, nil)
}

func newProtocolSessionObserved(server *Server, conn transport.Transport, scope TransportScope, observer TransportLifecycleObserver) *protocolSession {
	session := &protocolSession{
		server:             server,
		conn:               conn,
		scope:              scope.normalized(),
		sessionID:          server.nextProtocolSessionID.Add(1),
		attachments:        make(map[uint16]protocolAttachment),
		attachmentTokens:   make(map[string]uint16),
		fileChannels:       make(map[uint16]*sessionFileTransfer),
		fileIDs:            make(map[string]uint16),
		browserChannels:    make(map[uint16]*sessionBrowserProxy),
		browserTokens:      make(map[string]uint16),
		eventSubscriptions: make(map[uint64]applicationEventSubscription),
		rawPTYStreams:      make(map[uint16]*protocolRawPTYStream),
		requestSlots:       make(chan struct{}, server.cfg.protocolLimits.MaxInFlightRequests),
		activeRequests:     make(map[uint64]context.CancelFunc),
		liveBaselines:      make(map[string]*sessionLiveScreenBaselines),
		channelKinds:       make(map[uint16]protocolChannelKind),
		historyTokens:      make(map[protocolHistoryResourceKey]struct{}),
		lifecycleObserver:  observer,
	}
	// 中文说明：application executor 与当前连接 session 同寿命；具体 API Layer 装配由 composition root 注入。
	if server.cfg.applicationFactory != nil {
		session.application = server.cfg.applicationFactory(session)
	}
	session.nextChannel = 6
	return session
}

func (session *protocolSession) run(ctx context.Context) error {
	sessionCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	session.mu.Lock()
	session.sessionCtx = sessionCtx
	session.mu.Unlock()
	defer func() {
		cancel()
		session.requests.Wait()
		session.releaseAllHistorySnapshots()
		session.releaseAllFileTransfers()
		session.releaseAllBrowserProxies()
		session.stopEvents()
		session.releaseProtocolAttachments()
		session.clearLiveScreenBaselines()
	}()
	for {
		frame, err := session.conn.Recv()
		if err != nil {
			return err
		}
		channel, typ, payload, err := wire.DecodeFrame(frame)
		if err != nil {
			return err
		}
		if channel == 0 {
			if err := session.handleControlFrame(sessionCtx, typ, payload); err != nil {
				return err
			}
			continue
		}
		if !session.helloAccepted {
			if err := session.sendStreamError(channel, protocolErrorBadRequest, "protocol Hello is required before stream frames"); err != nil {
				return err
			}
			continue
		}
		if err := session.handleStreamFrame(sessionCtx, channel, typ, payload); err != nil {
			if sendErr := session.sendStreamError(channel, protocolErrorBadRequest, err.Error()); sendErr != nil {
				return sendErr
			}
		}
	}
}

func (session *protocolSession) lifetimeContext(fallback context.Context) context.Context {
	session.mu.RLock()
	ctx := session.sessionCtx
	session.mu.RUnlock()
	if ctx != nil {
		return ctx
	}
	return fallback
}

func (session *protocolSession) handleControlFrame(ctx context.Context, typ uint8, payload []byte) error {
	switch typ {
	case wire.TypeHello:
		if session.helloAccepted {
			return session.sendError(0, protocolErrorBadRequest, "protocol Hello was already accepted")
		}
		hello, err := protocol.DecodeHelloPayload(payload)
		if err != nil {
			return session.sendError(0, protocolErrorBadRequest, err.Error())
		}
		if hello.Version != 0 && hello.Version != wire.Version {
			return session.sendError(0, protocolErrorBadRequest, fmt.Sprintf("unsupported wire version %d", hello.Version))
		}
		response, err := protocol.EncodeHelloPayload(protocol.Hello{Version: wire.Version, Server: ModuleName})
		if err != nil {
			return err
		}
		if err := session.sendFrame(0, wire.TypeHello, response); err != nil {
			return err
		}
		session.helloAccepted = true
		if session.lifecycleObserver != nil {
			session.lifecycleObserver.HelloAccepted()
		}
		return nil
	case wire.TypeRequest:
		if !session.helloAccepted {
			return session.sendError(0, protocolErrorBadRequest, "protocol Hello is required before requests")
		}
		req, err := protocol.DecodeRequestPayload(payload)
		if err != nil {
			return session.sendError(0, protocolErrorBadRequest, err.Error())
		}
		if req.ID == 0 || req.Method == "" {
			return session.sendError(req.ID, protocolErrorBadRequest, "protocol request ID and method are required")
		}
		requestCtx, requestCancel := context.WithCancel(ctx)
		if !session.claimActiveRequest(req.ID, requestCancel) {
			requestCancel()
			_ = session.conn.Close()
			return fmt.Errorf("duplicate in-flight protocol request ID %d", req.ID)
		}
		select {
		case session.requestSlots <- struct{}{}:
		default:
			session.releaseActiveRequest(req.ID)
			return session.sendError(req.ID, protocolErrorExhausted, protocolRequestCapacityExhaustedMessage)
		}
		// 中文说明：control request 不能在同一 client 上互相 head-of-line blocking。
		// history.window latest 可能短暂等待 history 追平，普通 input ack 仍要能并发处理。
		session.requests.Add(1)
		if !session.server.startProtocolRequest(func() {
			defer session.requests.Done()
			defer session.releaseActiveRequest(req.ID)
			defer func() { <-session.requestSlots }()
			defer session.server.releaseProtocolRequest()
			if err := session.handleRequest(requestCtx, req); err != nil {
				_ = session.conn.Close()
			}
		}) {
			session.requests.Done()
			<-session.requestSlots
			session.releaseActiveRequest(req.ID)
			return session.sendError(req.ID, protocolErrorExhausted, protocolRequestCapacityExhaustedMessage)
		}
		return nil
	case wire.TypeRequestCancel:
		if !session.helloAccepted {
			return session.sendError(0, protocolErrorBadRequest, "protocol Hello is required before request cancellation")
		}
		id, err := protocol.DecodeRequestCancelPayload(payload)
		if err != nil {
			return session.sendError(0, protocolErrorBadRequest, err.Error())
		}
		session.cancelActiveRequest(id)
		return nil
	case wire.TypeSessionClose:
		if err := protocol.DecodeSessionClosePayload(payload); err != nil {
			return session.sendError(0, protocolErrorBadRequest, err.Error())
		}
		return io.EOF
	default:
		return session.sendError(0, protocolErrorBadRequest, fmt.Sprintf("unsupported control frame type %d", typ))
	}
}

// startProtocolRequest performs non-blocking admission against the server-wide execution budget.
func (server *Server) startProtocolRequest(run func()) bool {
	if server.closed.Load() {
		return false
	}
	select {
	case server.protocolRequestSlots <- struct{}{}:
		go run()
		return true
	default:
		return false
	}
}

func (server *Server) releaseProtocolRequest() {
	<-server.protocolRequestSlots
}

func (session *protocolSession) claimActiveRequest(id uint64, cancel context.CancelFunc) bool {
	session.requestMu.Lock()
	defer session.requestMu.Unlock()
	if _, exists := session.activeRequests[id]; exists {
		return false
	}
	session.activeRequests[id] = cancel
	return true
}

func (session *protocolSession) releaseActiveRequest(id uint64) {
	session.requestMu.Lock()
	cancel := session.activeRequests[id]
	delete(session.activeRequests, id)
	session.requestMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (session *protocolSession) cancelActiveRequest(id uint64) {
	session.requestMu.Lock()
	cancel := session.activeRequests[id]
	session.requestMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (session *protocolSession) handleRequest(ctx context.Context, req protocol.Request) error {
	finishTotal := perftrace.Measure("core.protocol.request." + req.Method + ".total")
	responseBytes := 0
	defer func() { finishTotal(responseBytes) }()
	finishDispatch := perftrace.Measure("core.protocol.request." + req.Method + ".dispatch")
	result, binary, code, err := session.dispatchRequest(ctx, req)
	finishDispatch(len(result))
	if err != nil {
		return session.sendError(req.ID, code, err.Error())
	}
	if binary {
		responseBytes = len(result)
		payload, err := protocol.EncodeBinaryResponsePayload(req.ID, result)
		if err != nil {
			return err
		}
		finishSend := perftrace.Measure("core.protocol.request." + req.Method + ".send")
		err = session.sendFrame(0, wire.TypeResponseBinary, payload)
		finishSend(len(payload))
		maybeReclaimDaemonBoundaryHeap()
		return err
	}
	payload, err := protocol.EncodeResponsePayload(protocol.Response{ID: req.ID, Result: result})
	if err != nil {
		return err
	}
	responseBytes = len(payload)
	finishSend := perftrace.Measure("core.protocol.request." + req.Method + ".send")
	err = session.sendFrame(0, wire.TypeResponse, payload)
	finishSend(len(payload))
	maybeReclaimDaemonBoundaryHeap()
	return err
}

func (session *protocolSession) dispatchRequest(ctx context.Context, req protocol.Request) ([]byte, bool, int, error) {
	if req.Method == "api.execute" {
		return session.dispatchApplicationPayload(ctx, req.Params)
	}
	return nil, false, protocolErrorNotFound, fmt.Errorf("unknown method: %s", req.Method)
}

func (session *protocolSession) clientAccessService() (ClientAccessService, error) {
	service := session.server.ClientAccessService()
	if service == nil {
		return nil, ErrClientAccessServiceUnavailable
	}
	return service, nil
}

func (session *protocolSession) remoteService() (RemoteService, error) {
	service := session.server.RemoteService()
	if service == nil {
		return nil, ErrRemoteServiceUnavailable
	}
	return service, nil
}

func (session *protocolSession) handleStreamFrame(ctx context.Context, channel uint16, typ uint8, payload []byte) error {
	if proxy := session.browserProxyForChannel(channel); proxy != nil {
		return session.handleBrowserProxyFrame(proxy, typ, payload)
	}
	if transfer := session.fileTransferForChannel(channel); transfer != nil {
		return session.handleFileTransferFrame(ctx, transfer, typ, payload)
	}
	attachment, err := session.attachmentForChannel(channel)
	if err != nil {
		return err
	}
	if err := session.scope.allowsAttachment(attachment); err != nil {
		return err
	}
	switch typ {
	case wire.TypeBootstrapDone:
		return session.startRawPTYStream(ctx, attachment)
	case wire.TypeClosed:
		if len(payload) != 0 {
			return fmt.Errorf("terminal attachment stream close payload must be empty")
		}
		session.stopRawPTYStream(attachment.Channel)
		return nil
	default:
		return fmt.Errorf("unsupported stream frame type %d", typ)
	}
}

func (session *protocolSession) sendStreamError(channel uint16, code int, message string) error {
	payload, err := protocol.EncodeErrorPayload(protocol.ErrorMessage{Error: protocol.ProtocolError{Code: code, Message: message}})
	if err != nil {
		return err
	}
	return session.sendFrame(channel, wire.TypeError, payload)
}

func (session *protocolSession) stopEvents() {
	session.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(session.eventSubscriptions))
	for id, subscription := range session.eventSubscriptions {
		cancels = append(cancels, subscription.cancel)
		delete(session.eventSubscriptions, id)
	}
	session.mu.Unlock()
	for range cancels {
		session.releaseEventSubscription()
	}
	for _, cancel := range cancels {
		cancel()
	}
}

func (session *protocolSession) clearEventSubscription(id uint64) {
	session.mu.Lock()
	_, exists := session.eventSubscriptions[id]
	if exists {
		delete(session.eventSubscriptions, id)
	}
	session.mu.Unlock()
	if exists {
		session.releaseEventSubscription()
	}
}

func (session *protocolSession) attach(params attachmentRequest) (protocolAttachment, *attachmentResizeControl, error) {
	info, err := session.server.GetTerminal(params.TerminalID)
	if err != nil {
		coreLifecycleTrace(session.server.cfg.logger, "protocol.attach.request",
			"terminal_id", params.TerminalID,
			"view_id", params.ViewID,
			"surface_id", params.SurfaceID,
			"resize_policy", params.ResizePolicy,
			"error", err.Error(),
		)
		return protocolAttachment{}, nil, err
	}
	attrs := coreTerminalInfoAttrs(info)
	attrs = append(attrs,
		"view_id", params.ViewID,
		"surface_id", params.SurfaceID,
		"resize_policy", params.ResizePolicy,
		"mode", params.Mode,
	)
	coreLifecycleTrace(session.server.cfg.logger, "protocol.attach.request", attrs...)
	mode := normalizeAttachMode(params.Mode)
	resizePolicy := normalizeAttachResizePolicy(params.ResizePolicy)
	if mode == string(TerminalAttachmentModeObserver) && resizePolicy != attachmentResizePolicyObserver {
		return protocolAttachment{}, nil, fmt.Errorf("observer attachment cannot request resize ownership")
	}
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return protocolAttachment{}, nil, fmt.Errorf("allocate attachment token: %w", err)
	}
	session.attachmentMu.Lock()
	defer session.attachmentMu.Unlock()
	attachment := protocolAttachment{
		SessionID:    session.sessionID,
		TerminalID:   params.TerminalID,
		Mode:         mode,
		ResizePolicy: resizePolicy,
		SurfaceID:    params.SurfaceID,
		ViewID:       params.ViewID,
		Token:        token,
	}
	replaced := session.protocolAttachmentsForView(attachment)
	channel, err := session.reserveAttachmentChannel(replaced...)
	if err != nil {
		return protocolAttachment{}, nil, err
	}
	// 中文说明：opaque token 的 channel 前缀只属于 protocol binding，用于把 Proto resource handle
	// 重新绑定到现有 stream frame；API Layer、TUI 和第三方客户端不得解析这个内部格式。
	binary.BigEndian.PutUint16(token[:2], channel)
	attachment.Channel = channel
	session.replaceProtocolAttachmentsForView(attachment, replaced)
	session.unregisterProtocolAttachments(replaced)
	if session.beforeGlobalAttachmentPublish != nil {
		session.beforeGlobalAttachmentPublish(attachment)
	}
	control := session.registerProtocolAttachment(attachment, info.Size, terminalmeta.SizeLocked(info.Tags))
	if control != nil && (control.Reason == attachmentResizeReasonOwner || control.Reason == attachmentResizeReasonSizeLocked) {
		if control.ResizeOwnership != nil {
			attachment.Epoch = control.ResizeOwnership.Epoch
		}
		attachment.ResizePolicy = attachmentResizePolicyOwner
		session.mu.Lock()
		session.attachments[channel] = attachment
		session.mu.Unlock()
	}
	coreLifecycleTrace(session.server.cfg.logger, "protocol.attach.result",
		"terminal_id", attachment.TerminalID,
		"channel", attachment.Channel,
		"view_id", attachment.ViewID,
		"surface_id", attachment.SurfaceID,
		"resize_policy", attachment.ResizePolicy,
		"can_resize", control.CanResize,
		"control_reason", control.Reason,
		"owner_view_id", control.OwnerViewID,
		"owner_surface_id", control.OwnerSurfaceID,
		"state", string(info.State),
	)
	return attachment, control, nil
}

func (session *protocolSession) publishAttachmentToken(attachment protocolAttachment) error {
	session.attachmentMu.Lock()
	defer session.attachmentMu.Unlock()
	session.mu.Lock()
	defer session.mu.Unlock()
	current, ok := session.attachments[attachment.Channel]
	if !ok || !bytes.Equal(current.Token, attachment.Token) {
		return fmt.Errorf("%w: pending attachment is no longer active", errProtocolAttachmentMismatch)
	}
	session.attachmentTokens[string(attachment.Token)] = attachment.Channel
	return nil
}

func (session *protocolSession) protocolAttachmentsForView(next protocolAttachment) []protocolAttachment {
	if next.ViewID == "" {
		return nil
	}
	session.mu.RLock()
	defer session.mu.RUnlock()
	detached := make([]protocolAttachment, 0)
	for _, current := range session.attachments {
		if !sameProtocolViewAttachment(next, current) {
			continue
		}
		detached = append(detached, current)
	}
	return detached
}

func (session *protocolSession) replaceProtocolAttachmentsForView(next protocolAttachment, replaced []protocolAttachment) {
	session.mu.Lock()
	defer session.mu.Unlock()
	for _, current := range replaced {
		// 中文说明：同一个 client view 重新 attach 时，新 channel 才是输入/resize
		// 真值；旧 channel 必须从 daemon attachment registry 释放，避免 chrome 计数膨胀。
		delete(session.attachments, current.Channel)
		delete(session.attachmentTokens, string(current.Token))
	}
	session.attachments[next.Channel] = next
}

func sameProtocolViewAttachment(next protocolAttachment, current protocolAttachment) bool {
	if next.ViewID == "" || current.ViewID != next.ViewID {
		return false
	}
	if next.SurfaceID != "" {
		return current.SurfaceID == next.SurfaceID
	}
	return true
}

func (session *protocolSession) detach(params attachmentDetachRequest) {
	session.attachmentMu.Lock()
	defer session.attachmentMu.Unlock()
	session.detachLocked(params, nil)
}

func (session *protocolSession) detachExact(attachment protocolAttachment) {
	session.attachmentMu.Lock()
	defer session.attachmentMu.Unlock()
	session.detachLocked(attachmentDetachRequest{Channel: attachment.Channel}, attachment.Token)
}

func (session *protocolSession) detachLocked(params attachmentDetachRequest, exactToken []byte) {
	session.mu.Lock()
	var detached []protocolAttachment
	for channel, attachment := range session.attachments {
		if !detachMatches(params, channel, attachment) {
			continue
		}
		if exactToken != nil && !bytes.Equal(attachment.Token, exactToken) {
			continue
		}
		delete(session.attachments, channel)
		delete(session.attachmentTokens, string(attachment.Token))
		detached = append(detached, attachment)
	}
	session.mu.Unlock()
	session.unregisterProtocolAttachments(detached)
}

func detachMatches(params attachmentDetachRequest, channel uint16, attachment protocolAttachment) bool {
	if params.Channel != 0 {
		return params.Channel == channel
	}
	if params.TerminalID != "" && params.TerminalID != attachment.TerminalID {
		return false
	}
	if params.SurfaceID != "" && params.SurfaceID != attachment.SurfaceID {
		return false
	}
	if params.ViewID != "" && params.ViewID != attachment.ViewID {
		return false
	}
	return params.TerminalID != "" || params.SurfaceID != "" || params.ViewID != ""
}

func (session *protocolSession) input(ctx context.Context, params attachmentInputRequest) error {
	attachment, err := session.attachmentForChannel(params.Channel)
	if err != nil {
		coreLifecycleTrace(session.server.cfg.logger, "protocol.input",
			"terminal_id", params.TerminalID,
			"channel", params.Channel,
			"view_id", params.ViewID,
			"surface_id", params.SurfaceID,
			"bytes", len(params.Data),
			"error", err.Error(),
		)
		return err
	}
	if attachment.Mode == string(TerminalAttachmentModeObserver) {
		return fmt.Errorf("observer attachment cannot send terminal input")
	}
	if attachment.TerminalID != params.TerminalID {
		err := fmt.Errorf("%w: input channel %d is attached to %s, not %s", errProtocolAttachmentMismatch, params.Channel, attachment.TerminalID, params.TerminalID)
		coreLifecycleTrace(session.server.cfg.logger, "protocol.input",
			"terminal_id", params.TerminalID,
			"attached_terminal", attachment.TerminalID,
			"channel", params.Channel,
			"view_id", params.ViewID,
			"attached_view_id", attachment.ViewID,
			"surface_id", params.SurfaceID,
			"attached_surface_id", attachment.SurfaceID,
			"bytes", len(params.Data),
			"error", err.Error(),
		)
		return err
	}
	if params.SurfaceID != "" && attachment.SurfaceID != params.SurfaceID {
		err := fmt.Errorf("%w: input channel %d surface mismatch: %s != %s", errProtocolAttachmentMismatch, params.Channel, attachment.SurfaceID, params.SurfaceID)
		coreLifecycleTrace(session.server.cfg.logger, "protocol.input",
			"terminal_id", params.TerminalID,
			"channel", params.Channel,
			"view_id", params.ViewID,
			"attached_view_id", attachment.ViewID,
			"surface_id", params.SurfaceID,
			"attached_surface_id", attachment.SurfaceID,
			"bytes", len(params.Data),
			"error", err.Error(),
		)
		return err
	}
	if params.ViewID != "" && attachment.ViewID != params.ViewID {
		err := fmt.Errorf("%w: input channel %d view mismatch: %s != %s", errProtocolAttachmentMismatch, params.Channel, attachment.ViewID, params.ViewID)
		coreLifecycleTrace(session.server.cfg.logger, "protocol.input",
			"terminal_id", params.TerminalID,
			"channel", params.Channel,
			"view_id", params.ViewID,
			"attached_view_id", attachment.ViewID,
			"surface_id", params.SurfaceID,
			"attached_surface_id", attachment.SurfaceID,
			"bytes", len(params.Data),
			"error", err.Error(),
		)
		return err
	}
	err = session.server.WriteInput(ctx, attachment.TerminalID, params.Data)
	if err != nil {
		coreLifecycleTrace(session.server.cfg.logger, "protocol.input",
			"terminal_id", params.TerminalID,
			"channel", params.Channel,
			"view_id", params.ViewID,
			"surface_id", params.SurfaceID,
			"bytes", len(params.Data),
			"error", err.Error(),
		)
		return err
	}
	coreLifecycleTrace(session.server.cfg.logger, "protocol.input",
		"terminal_id", params.TerminalID,
		"channel", params.Channel,
		"view_id", params.ViewID,
		"surface_id", params.SurfaceID,
		"bytes", len(params.Data),
		"result", "ok",
	)
	return nil
}

func (session *protocolSession) attachmentForChannel(channel uint16) (protocolAttachment, error) {
	session.mu.RLock()
	defer session.mu.RUnlock()
	attachment, ok := session.attachments[channel]
	if !ok {
		return protocolAttachment{}, ErrTerminalNotFound
	}
	return attachment, nil
}

func (session *protocolSession) attachmentForToken(token []byte) (protocolAttachment, error) {
	if len(token) == 0 {
		return protocolAttachment{}, ErrTerminalNotFound
	}
	session.mu.RLock()
	channel, ok := session.attachmentTokens[string(token)]
	attachment := session.attachments[channel]
	session.mu.RUnlock()
	if !ok || attachment.Channel == 0 {
		return protocolAttachment{}, ErrTerminalNotFound
	}
	return attachment, nil
}

func (session *protocolSession) resizeControlForRequest(attachment protocolAttachment, policy string, surfaceID string, viewID string, takeOwnership bool, expectedOwnerEpoch uint64) (*attachmentResizeControl, bool, error) {
	info, err := session.server.GetTerminal(attachment.TerminalID)
	if err != nil {
		return nil, false, err
	}
	session.attachmentMu.Lock()
	defer session.attachmentMu.Unlock()
	if policy != "" {
		attachment.ResizePolicy = normalizeResizePolicy(policy)
	}
	if attachment.Mode == string(TerminalAttachmentModeObserver) && (takeOwnership || attachment.ResizePolicy != attachmentResizePolicyObserver) {
		return nil, false, fmt.Errorf("observer attachment cannot request resize ownership")
	}
	if surfaceID != "" {
		attachment.SurfaceID = surfaceID
	}
	if viewID != "" {
		attachment.ViewID = viewID
	}
	session.mu.Lock()
	current, ok := session.attachments[attachment.Channel]
	if !ok || !bytes.Equal(current.Token, attachment.Token) {
		session.mu.Unlock()
		return nil, false, fmt.Errorf("%w: attachment is no longer active", errProtocolAttachmentMismatch)
	}
	current.ResizePolicy = attachment.ResizePolicy
	current.SurfaceID = attachment.SurfaceID
	current.ViewID = attachment.ViewID
	attachment = current
	session.attachments[attachment.Channel] = current
	session.mu.Unlock()
	control := session.updateProtocolAttachmentControl(attachment, info.Size, takeOwnership, expectedOwnerEpoch)
	return control, control.CanResize, nil
}

func (session *protocolSession) setResizeLock(params attachmentResizeControlRequest, locked bool) (*attachmentResizeControl, error) {
	attachment, err := session.attachmentForChannel(params.Channel)
	if err != nil {
		return nil, err
	}
	if attachment.TerminalID != params.TerminalID {
		return nil, fmt.Errorf("resize control channel %d is attached to %s, not %s", params.Channel, attachment.TerminalID, params.TerminalID)
	}
	control, _, err := session.resizeControlForRequest(attachment, params.ResizePolicy, params.SurfaceID, params.ViewID, false, 0)
	if err != nil {
		return nil, err
	}
	ownerAttachmentID := ""
	if control.ResizeOwnership != nil {
		ownerAttachmentID = control.ResizeOwnership.OwnerAttachmentID
	}
	if ownerAttachmentID != protocolAttachmentOwnerID(attachment) || attachment.ResizePolicy != attachmentResizePolicyOwner {
		return control, nil
	}
	info, err := session.server.GetTerminal(params.TerminalID)
	if err != nil {
		return nil, err
	}
	return session.setGlobalResizeLock(attachment, info.Size, locked)
}

func (session *protocolSession) registerProtocolAttachment(attachment protocolAttachment, size Size, initialSizeLocked bool) *attachmentResizeControl {
	session.server.protocolAttachmentMu.Lock()
	defer session.server.protocolAttachmentMu.Unlock()
	if attachment.SessionID == 0 {
		attachment.SessionID = session.sessionID
	}
	if _, initialized := session.server.protocolSizeLocks[attachment.TerminalID]; !initialized && initialSizeLocked {
		session.server.protocolSizeLocks[attachment.TerminalID] = true
	}
	key := attachmentKey(attachment)
	changed := true
	if existing, ok := session.server.protocolAttachments[key]; ok {
		changed = !protocolAttachmentEqual(existing, attachment)
	}
	takeOwner := attachment.ResizePolicy == attachmentResizePolicyOwner ||
		(session.server.protocolResizeOwners[attachment.TerminalID] == "" && attachment.ResizePolicy != attachmentResizePolicyObserver)
	if takeOwner {
		attachment.ResizePolicy = attachmentResizePolicyOwner
		attachment.Epoch = session.server.advanceProtocolResizeEpochLocked(attachment.TerminalID)
		session.server.protocolResizeOwners[attachment.TerminalID] = key
		changed = true
	}
	session.server.protocolAttachments[key] = attachment
	session.server.protocolChannelIndex[protocolAttachmentKey{SessionID: attachment.SessionID, Channel: attachment.Channel}] = key
	control := session.resizeControlForGlobalAttachmentLocked(attachment, size)
	session.publishProtocolAttachmentChangedLocked(attachment.TerminalID, changed)
	return control
}

func (session *protocolSession) updateProtocolAttachmentControl(attachment protocolAttachment, size Size, takeOwner bool, expectedOwnerEpoch uint64) *attachmentResizeControl {
	session.server.protocolAttachmentMu.Lock()
	defer session.server.protocolAttachmentMu.Unlock()
	if attachment.SessionID == 0 {
		attachment.SessionID = session.sessionID
	}
	key := attachmentKey(attachment)
	if current, ok := session.server.protocolAttachments[key]; ok {
		current.ResizePolicy = attachment.ResizePolicy
		current.SurfaceID = attachment.SurfaceID
		current.ViewID = attachment.ViewID
		attachment = current
	} else {
		session.server.protocolChannelIndex[protocolAttachmentKey{SessionID: attachment.SessionID, Channel: attachment.Channel}] = key
	}
	existing, existed := session.server.protocolAttachments[key]
	ownerKey := session.server.protocolResizeOwners[attachment.TerminalID]
	ownerEpoch := session.server.protocolResizeEpochs[attachment.TerminalID]
	if ownerEpoch == 0 {
		if owner, ok := session.server.protocolAttachments[ownerKey]; ok {
			ownerEpoch = owner.Epoch
		}
	}
	canTakeOwner := takeOwner && (expectedOwnerEpoch == 0 || expectedOwnerEpoch == ownerEpoch)
	if canTakeOwner && ownerKey != key {
		attachment.ResizePolicy = attachmentResizePolicyOwner
		attachment.Epoch = session.server.advanceProtocolResizeEpochLocked(attachment.TerminalID)
		session.server.protocolResizeOwners[attachment.TerminalID] = key
	} else if ownerKey == key && attachment.ResizePolicy != attachmentResizePolicyOwner {
		delete(session.server.protocolResizeOwners, attachment.TerminalID)
		session.server.protocolAttachments[key] = attachment
		if !session.promoteGlobalResizeOwnerExceptLocked(attachment.TerminalID, key) {
			session.server.advanceProtocolResizeEpochLocked(attachment.TerminalID)
		}
	} else if ownerKey == key {
		attachment.ResizePolicy = attachmentResizePolicyOwner
		attachment.Epoch = ownerEpoch
	}
	changed := !existed || !protocolAttachmentEqual(existing, attachment) || (canTakeOwner && ownerKey != key)
	session.server.protocolAttachments[key] = attachment
	control := session.resizeControlForGlobalAttachmentLocked(attachment, size)
	session.publishProtocolAttachmentChangedLocked(attachment.TerminalID, changed)
	return control
}

func (session *protocolSession) unregisterProtocolAttachments(detached []protocolAttachment) {
	if len(detached) == 0 {
		return
	}
	for _, attachment := range detached {
		session.stopRawPTYStream(attachment.Channel)
	}
	session.server.protocolAttachmentMu.Lock()
	changedTerminals := make(map[string]bool)
	for _, attachment := range detached {
		key := attachmentKey(attachment)
		if _, ok := session.server.protocolAttachments[key]; ok {
			delete(session.server.protocolAttachments, key)
			changedTerminals[attachment.TerminalID] = true
		}
		delete(session.server.protocolChannelIndex, protocolAttachmentKey{SessionID: attachment.SessionID, Channel: attachment.Channel})
		if session.server.protocolResizeOwners[attachment.TerminalID] == key {
			delete(session.server.protocolResizeOwners, attachment.TerminalID)
			if !session.promoteGlobalResizeOwnerLocked(attachment.TerminalID) {
				session.server.advanceProtocolResizeEpochLocked(attachment.TerminalID)
			}
			changedTerminals[attachment.TerminalID] = true
		}
	}
	for terminalID := range changedTerminals {
		session.publishProtocolAttachmentChangedLocked(terminalID, true)
	}
	session.server.protocolAttachmentMu.Unlock()
	for _, attachment := range detached {
		session.releaseChannel(attachment.Channel, protocolChannelAttachment)
	}
}

func (session *protocolSession) releaseProtocolAttachments() {
	session.attachmentMu.Lock()
	defer session.attachmentMu.Unlock()
	session.mu.Lock()
	detached := make([]protocolAttachment, 0, len(session.attachments))
	for channel, attachment := range session.attachments {
		delete(session.attachments, channel)
		delete(session.attachmentTokens, string(attachment.Token))
		detached = append(detached, attachment)
	}
	session.mu.Unlock()
	session.unregisterProtocolAttachments(detached)
}

func (session *protocolSession) promoteGlobalResizeOwnerLocked(terminalID string) bool {
	return session.promoteGlobalResizeOwnerExceptLocked(terminalID, "")
}

func (session *protocolSession) promoteGlobalResizeOwnerExceptLocked(terminalID string, excludedKey string) bool {
	selectedKey := ""
	selected := protocolAttachment{}
	for key, attachment := range session.server.protocolAttachments {
		if key == excludedKey || attachment.TerminalID != terminalID || attachment.ResizePolicy == attachmentResizePolicyObserver {
			continue
		}
		if selectedKey == "" || protocolAttachmentPromotionLess(attachment, selected) {
			selectedKey = key
			selected = attachment
		}
	}
	if selectedKey == "" {
		return false
	}
	selected.ResizePolicy = attachmentResizePolicyOwner
	selected.Epoch = session.server.advanceProtocolResizeEpochLocked(terminalID)
	session.server.protocolAttachments[selectedKey] = selected
	session.server.protocolResizeOwners[terminalID] = selectedKey
	return true
}

func (server *Server) advanceProtocolResizeEpochLocked(terminalID string) uint64 {
	server.protocolOwnerEpoch++
	server.protocolResizeEpochs[terminalID] = server.protocolOwnerEpoch
	return server.protocolOwnerEpoch
}

func protocolAttachmentPromotionLess(left protocolAttachment, right protocolAttachment) bool {
	if left.SessionID != right.SessionID {
		return left.SessionID < right.SessionID
	}
	if left.Channel != right.Channel {
		return left.Channel < right.Channel
	}
	if left.SurfaceID != right.SurfaceID {
		return left.SurfaceID < right.SurfaceID
	}
	return left.ViewID < right.ViewID
}

func (session *protocolSession) setGlobalResizeLock(attachment protocolAttachment, size Size, locked bool) (*attachmentResizeControl, error) {
	session.attachmentMu.Lock()
	defer session.attachmentMu.Unlock()
	session.mu.RLock()
	current, ok := session.attachments[attachment.Channel]
	session.mu.RUnlock()
	if !ok || !bytes.Equal(current.Token, attachment.Token) {
		return nil, fmt.Errorf("%w: attachment is no longer active", errProtocolAttachmentMismatch)
	}
	attachment = current
	session.server.protocolAttachmentMu.Lock()
	defer session.server.protocolAttachmentMu.Unlock()
	terminalID := attachment.TerminalID
	key := attachmentKey(attachment)
	global, ok := session.server.protocolAttachments[key]
	if !ok || session.server.protocolResizeOwners[terminalID] != key || global.ResizePolicy != attachmentResizePolicyOwner {
		if ok {
			attachment = global
		}
		return session.resizeControlForGlobalAttachmentLocked(attachment, size), nil
	}
	if session.server.protocolSizeLocks[terminalID] != locked {
		epoch := session.server.advanceProtocolResizeEpochLocked(terminalID)
		session.server.protocolSizeLocks[terminalID] = locked
		if ownerKey := session.server.protocolResizeOwners[terminalID]; ownerKey != "" {
			if owner, ok := session.server.protocolAttachments[ownerKey]; ok {
				owner.Epoch = epoch
				session.server.protocolAttachments[ownerKey] = owner
			}
		}
	}
	if current, ok := session.server.protocolAttachments[attachmentKey(attachment)]; ok {
		attachment = current
	}
	control := session.resizeControlForGlobalAttachmentLocked(attachment, size)
	session.publishProtocolAttachmentChangedLocked(terminalID, true)
	return control, nil
}

func (session *protocolSession) resizeControlForGlobalAttachmentLocked(attachment protocolAttachment, size Size) *attachmentResizeControl {
	key := attachmentKey(attachment)
	ownerKey := session.server.protocolResizeOwners[attachment.TerminalID]
	owner, hasOwner := session.server.protocolAttachments[ownerKey]
	if !hasOwner || owner.TerminalID != attachment.TerminalID {
		control := &attachmentResizeControl{
			Reason:         attachmentResizeReasonFollower,
			SizeLocked:     session.server.protocolSizeLocks[attachment.TerminalID],
			SurfaceID:      attachment.SurfaceID,
			OwnerSurfaceID: "",
			OwnerViewID:    "",
		}
		if attachment.ResizePolicy == attachmentResizePolicyObserver {
			control.Reason = attachmentResizeReasonObserver
		}
		return control
	}
	ownership := &attachmentResizeOwnership{
		OwnerAttachmentID: protocolAttachmentOwnerID(owner),
		OwnerSurfaceID:    owner.SurfaceID,
		OwnerViewID:       owner.ViewID,
		Size:              size,
		SizeLocked:        session.server.protocolSizeLocks[attachment.TerminalID],
		Epoch:             owner.Epoch,
	}
	control := &attachmentResizeControl{
		CanResize:       ownerKey == key && attachment.ResizePolicy == attachmentResizePolicyOwner && !ownership.SizeLocked,
		Reason:          attachmentResizeReasonFollower,
		SizeLocked:      ownership.SizeLocked,
		SurfaceID:       attachment.SurfaceID,
		OwnerSurfaceID:  owner.SurfaceID,
		OwnerViewID:     owner.ViewID,
		ResizeOwnership: ownership,
	}
	if ownership.SizeLocked && ownerKey == key && attachment.ResizePolicy == attachmentResizePolicyOwner {
		control.Reason = attachmentResizeReasonSizeLocked
	} else if attachment.ResizePolicy == attachmentResizePolicyObserver {
		control.Reason = attachmentResizeReasonObserver
	} else if control.CanResize {
		control.Reason = attachmentResizeReasonOwner
	}
	return control
}

func (server *Server) protocolAttachmentCount(terminalID string) int {
	server.protocolAttachmentMu.Lock()
	defer server.protocolAttachmentMu.Unlock()
	return server.protocolAttachmentViewCountLocked(terminalID)
}

func (server *Server) protocolAttachmentViewCountLocked(terminalID string) int {
	seen := make(map[string]struct{})
	for _, attachment := range server.protocolAttachments {
		if attachment.TerminalID != terminalID {
			continue
		}
		seen[protocolAttachmentViewCountKey(attachment)] = struct{}{}
	}
	return len(seen)
}

func protocolAttachmentViewCountKey(attachment protocolAttachment) string {
	if attachment.ViewID == "" {
		return attachmentKey(attachment)
	}
	// A connection count represents attached views. SessionID prevents two
	// clients that happen to reuse the same surface/view IDs from collapsing,
	// while distinct panels in one client remain distinct through ViewID.
	return fmt.Sprintf("%d\x00%s\x00%s", attachment.SessionID, attachment.SurfaceID, attachment.ViewID)
}

func (session *protocolSession) publishProtocolAttachmentChangedLocked(terminalID string, changed bool) {
	if !changed || terminalID == "" {
		return
	}
	info, err := session.server.GetTerminal(terminalID)
	if err != nil {
		return
	}
	terminal := info.Clone()
	projection := session.protocolAttachmentProjectionLocked(terminalID, terminal.Size)
	session.server.events.publish(Event{
		Type:       EventTerminalMetadataChanged,
		TerminalID: terminalID,
		Terminal:   &terminal,
		Attachment: projection,
	})
}

func (session *protocolSession) protocolAttachmentProjectionLocked(terminalID string, size Size) *TerminalAttachmentProjection {
	projection := &TerminalAttachmentProjection{
		AttachmentCount: session.server.protocolAttachmentViewCountLocked(terminalID),
		ResizeEpoch:     session.server.protocolResizeEpochs[terminalID],
		ResizeControl: &TerminalResizeControl{
			Reason:     TerminalResizeReasonFollower,
			SizeLocked: session.server.protocolSizeLocks[terminalID],
		},
	}
	if ownerKey := session.server.protocolResizeOwners[terminalID]; ownerKey != "" {
		if owner, ok := session.server.protocolAttachments[ownerKey]; ok {
			if projection.ResizeEpoch == 0 {
				projection.ResizeEpoch = owner.Epoch
			}
			projection.ResizeControl = applicationResizeControl(session.resizeControlForGlobalAttachmentLocked(owner, size))
		}
	}
	return projection
}

func (session *protocolSession) protocolAttachmentSnapshotEvents(filter EventFilter) []Event {
	terminals := session.server.ListTerminals()
	if len(terminals) == 0 {
		return nil
	}
	now := time.Now().UTC()
	session.server.protocolAttachmentMu.Lock()
	defer session.server.protocolAttachmentMu.Unlock()
	events := make([]Event, 0, len(terminals))
	for index := range terminals {
		terminal := terminals[index].Clone()
		event := Event{
			Type:       EventTerminalMetadataChanged,
			TerminalID: terminal.ID,
			Terminal:   &terminal,
			Attachment: session.protocolAttachmentProjectionLocked(terminal.ID, terminal.Size),
			Timestamp:  now,
		}
		if eventMatchesFilter(event, filter) {
			events = append(events, event)
		}
	}
	return events
}

func attachmentKey(attachment protocolAttachment) string {
	return fmt.Sprintf("%d:%d", attachment.SessionID, attachment.Channel)
}

func protocolAttachmentOwnerID(attachment protocolAttachment) string {
	return attachmentKey(attachment)
}

func normalizeResizePolicy(policy string) string {
	switch policy {
	case attachmentResizePolicyFollower, attachmentResizePolicyObserver:
		return policy
	default:
		return attachmentResizePolicyOwner
	}
}

func normalizeAttachResizePolicy(policy string) string {
	switch policy {
	case attachmentResizePolicyOwner, attachmentResizePolicyObserver:
		return policy
	default:
		return attachmentResizePolicyFollower
	}
}

func (session *protocolSession) sendError(id uint64, code int, message string) error {
	payload, err := protocol.EncodeErrorPayload(protocol.ErrorMessage{
		ID: id,
		Error: protocol.ProtocolError{
			Code:    code,
			Message: message,
		},
	})
	if err != nil {
		return err
	}
	return session.sendFrame(0, wire.TypeError, payload)
}

func (session *protocolSession) sendFrame(channel uint16, typ uint8, payload []byte) error {
	frame, err := wire.EncodeFrame(channel, typ, payload)
	if err != nil {
		return err
	}
	session.sendMu.Lock()
	defer session.sendMu.Unlock()
	return session.conn.Send(frame)
}

func (session *protocolSession) protocolLimits() ProtocolSessionLimits {
	if session.server == nil {
		return DefaultProtocolSessionLimits()
	}
	return session.server.cfg.protocolLimits.normalized()
}

func (session *protocolSession) reserveAttachmentChannel(replaced ...protocolAttachment) (uint16, error) {
	session.resourceMu.Lock()
	defer session.resourceMu.Unlock()
	limits := session.protocolLimits()
	replacedCount := 0
	for _, attachment := range replaced {
		if session.channelKinds[attachment.Channel] == protocolChannelAttachment {
			replacedCount++
		}
	}
	nextAttachmentCount := session.attachmentCount - replacedCount + 1
	nextResourceCount := session.totalResourcesLocked() - replacedCount + 1
	if nextAttachmentCount > limits.MaxAttachments || nextResourceCount > limits.MaxResources {
		return 0, fmt.Errorf("%w: attachment limit reached", ErrProtocolResourceExhausted)
	}
	channel, err := session.allocateChannelLocked()
	if err != nil {
		return 0, err
	}
	for _, attachment := range replaced {
		if session.channelKinds[attachment.Channel] == protocolChannelAttachment {
			delete(session.channelKinds, attachment.Channel)
		}
	}
	session.channelKinds[channel] = protocolChannelAttachment
	session.attachmentCount = nextAttachmentCount
	return channel, nil
}

func (session *protocolSession) reserveFileChannel() (uint16, error) {
	session.resourceMu.Lock()
	defer session.resourceMu.Unlock()
	limits := session.protocolLimits()
	if session.fileTransferCount >= limits.MaxFileTransfers || session.totalResourcesLocked() >= limits.MaxResources {
		return 0, fmt.Errorf("%w: file transfer limit reached", ErrProtocolResourceExhausted)
	}
	channel, err := session.allocateChannelLocked()
	if err != nil {
		return 0, err
	}
	session.channelKinds[channel] = protocolChannelFileTransfer
	session.fileTransferCount++
	return channel, nil
}

func (session *protocolSession) reserveBrowserChannel() (uint16, error) {
	session.resourceMu.Lock()
	defer session.resourceMu.Unlock()
	if session.totalResourcesLocked() >= session.protocolLimits().MaxResources {
		return 0, fmt.Errorf("%w: browser proxy resource limit reached", ErrProtocolResourceExhausted)
	}
	channel, err := session.allocateChannelLocked()
	if err != nil {
		return 0, err
	}
	session.channelKinds[channel] = protocolChannelBrowserProxy
	session.browserCount++
	return channel, nil
}

func (session *protocolSession) reserveEventSubscription() error {
	session.resourceMu.Lock()
	defer session.resourceMu.Unlock()
	limits := session.protocolLimits()
	if session.eventSubscriptionCount >= limits.MaxEventSubscriptions || session.totalResourcesLocked() >= limits.MaxResources {
		return fmt.Errorf("%w: event subscription limit reached", ErrProtocolResourceExhausted)
	}
	session.eventSubscriptionCount++
	return nil
}

const maxProtocolChannelID uint32 = 1<<16 - 1

func (session *protocolSession) allocateChannelLocked() (uint16, error) {
	if session.channelKinds == nil {
		session.channelKinds = make(map[uint16]protocolChannelKind)
	}
	if session.nextChannel >= maxProtocolChannelID {
		return 0, fmt.Errorf("%w: stream channel IDs are exhausted", ErrProtocolResourceExhausted)
	}
	session.nextChannel++
	return uint16(session.nextChannel), nil
}

func (session *protocolSession) releaseChannel(channel uint16, kind protocolChannelKind) {
	if channel == 0 {
		return
	}
	session.resourceMu.Lock()
	if session.channelKinds[channel] == kind {
		delete(session.channelKinds, channel)
		switch kind {
		case protocolChannelAttachment:
			if session.attachmentCount > 0 {
				session.attachmentCount--
			}
		case protocolChannelFileTransfer:
			if session.fileTransferCount > 0 {
				session.fileTransferCount--
			}
		case protocolChannelBrowserProxy:
			if session.browserCount > 0 {
				session.browserCount--
			}
		}
	}
	session.resourceMu.Unlock()
}

func (session *protocolSession) releaseEventSubscription() {
	session.resourceMu.Lock()
	if session.eventSubscriptionCount > 0 {
		session.eventSubscriptionCount--
	}
	session.resourceMu.Unlock()
}

func (session *protocolSession) reserveHistoryToken() error {
	session.resourceMu.Lock()
	defer session.resourceMu.Unlock()
	if session.totalResourcesLocked() >= session.protocolLimits().MaxResources {
		return fmt.Errorf("%w: history token limit reached", ErrProtocolResourceExhausted)
	}
	session.historyTokenReservations++
	return nil
}

func (session *protocolSession) rollbackHistoryTokenReservation() {
	session.resourceMu.Lock()
	if session.historyTokenReservations > 0 {
		session.historyTokenReservations--
	}
	session.resourceMu.Unlock()
}

func (session *protocolSession) commitHistoryToken(terminalID string, token history.HistoryToken) {
	session.resourceMu.Lock()
	if session.historyTokenReservations > 0 {
		session.historyTokenReservations--
	}
	if session.historyTokens == nil {
		session.historyTokens = make(map[protocolHistoryResourceKey]struct{})
	}
	session.historyTokens[protocolHistoryResourceKey{TerminalID: terminalID, Token: token}] = struct{}{}
	session.resourceMu.Unlock()
}

func (session *protocolSession) ownsHistoryToken(terminalID string, token history.HistoryToken) bool {
	session.resourceMu.Lock()
	_, ok := session.historyTokens[protocolHistoryResourceKey{TerminalID: terminalID, Token: token}]
	session.resourceMu.Unlock()
	return ok
}

func (session *protocolSession) forgetHistoryToken(terminalID string, token history.HistoryToken) bool {
	key := protocolHistoryResourceKey{TerminalID: terminalID, Token: token}
	session.resourceMu.Lock()
	_, ok := session.historyTokens[key]
	if ok {
		delete(session.historyTokens, key)
	}
	session.resourceMu.Unlock()
	return ok
}

func (session *protocolSession) releaseOwnedHistoryToken(terminalID string, token history.HistoryToken) {
	if !session.forgetHistoryToken(terminalID, token) {
		return
	}
	_ = session.server.TerminalHistoryRelease(context.Background(), terminalID, token)
}

func (session *protocolSession) releaseAllHistorySnapshots() {
	session.resourceMu.Lock()
	keys := make([]protocolHistoryResourceKey, 0, len(session.historyTokens))
	for key := range session.historyTokens {
		keys = append(keys, key)
	}
	clear(session.historyTokens)
	session.historyTokenReservations = 0
	session.resourceMu.Unlock()
	for _, key := range keys {
		_ = session.server.TerminalHistoryRelease(context.Background(), key.TerminalID, key.Token)
	}
}

func (session *protocolSession) totalResourcesLocked() int {
	return session.attachmentCount + session.fileTransferCount + session.browserCount + session.eventSubscriptionCount + session.historyTokenReservations + len(session.historyTokens)
}

func normalizeAttachMode(mode string) string {
	if mode == "" {
		return "collaborator"
	}
	return mode
}

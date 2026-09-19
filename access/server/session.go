package server

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"sync"

	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
	"github.com/anytty/anytty/shared/transport"
	"google.golang.org/protobuf/proto"
)

const (
	protocolErrorBadRequest  = 400
	protocolErrorForbidden   = 403
	protocolErrorNotFound    = 404
	protocolErrorExhausted   = 429
	protocolErrorUnavailable = 503
	protocolErrorInternal    = 500
)

// defaultMaxInFlightRequests 是一条客户端连接上并发 control request 的上限。
const defaultMaxInFlightRequests = 64

// maxProtocolChannelID 与 framing uint16 channel 空间一致。
const maxProtocolChannelID uint32 = math.MaxUint16

// accessTokenBytes 是 access 重写后 resource token 的长度，保持与 daemon token 一致的不透明性。
const accessTokenBytes = 32

type session struct {
	server *Server
	conn   transport.Transport
	logger *slog.Logger

	sendMu sync.Mutex

	stateMu       sync.RWMutex
	helloAccepted bool
	sessionCtx    context.Context

	providerMu sync.Mutex
	provider   terminalprovider.Provider

	requestSlots   chan struct{}
	requests       sync.WaitGroup
	requestMu      sync.Mutex
	activeRequests map[uint64]context.CancelFunc

	streamsMu   sync.Mutex
	channels    map[uint16]*streamBinding
	tokens      map[string]*streamBinding
	retired     map[uint16]struct{}
	nextChannel uint32

	localMu            sync.Mutex
	localSubscriptions map[string]*localSubscription

	eventsMu      sync.Mutex
	eventProvider terminalprovider.Provider
	eventCancel   context.CancelFunc
	eventWG       sync.WaitGroup
}

// streamBinding 把一个 access-issued channel/token 映射到 provider resource
// 或 access-local resource（文件传输、browser proxy）。
// attachment 只有 bootstrap 后才打开 provider stream；file/browser 在资源发布后立即桥接。
type streamBinding struct {
	clientChannel    uint16
	kind             apipb.ResourceKind
	accessToken      []byte
	providerToken    []byte
	providerChannel  uint16
	providerResource *apipb.ResourceHandle
	localID          string
	local            localResource

	mu         sync.Mutex
	generation uint64
	stream     terminalprovider.Stream
}

// localResource 是 access-local stream 的 session 侧契约：
// HandleFrame 返回 done=true 表示终态；Close 必须幂等并释放 OS 资源。
type localResource interface {
	HandleFrame(ctx context.Context, typ uint8, payload []byte) (bool, error)
	Done() <-chan struct{}
	Close() error
}

// sessionStream 是 access session 暴露给本地文件/转发服务的帧通道。
type sessionStream struct {
	session *session
	channel uint16
}

func (stream sessionStream) Send(typ uint8, payload []byte) error {
	return stream.session.sendFrame(stream.channel, typ, payload)
}

func (stream sessionStream) OutboundBuffered() uint64 {
	if reporter, ok := stream.session.conn.(transport.OutboundBufferReporter); ok {
		return reporter.OutboundBufferedAmount()
	}
	return 0
}

func (stream sessionStream) Done() <-chan struct{} {
	return stream.session.lifetimeContext().Done()
}

func newSession(server *Server, connection transport.Transport) (*session, error) {
	if server == nil || connection == nil {
		return nil, errors.New("access/server: session requires server and transport")
	}
	return &session{
		server:         server,
		conn:           connection,
		logger:         server.cfg.Logger,
		requestSlots:   make(chan struct{}, defaultMaxInFlightRequests),
		activeRequests: make(map[uint64]context.CancelFunc),
		channels:       make(map[uint16]*streamBinding),
		tokens:         make(map[string]*streamBinding),
		retired:        make(map[uint16]struct{}),
		nextChannel:    6,
	}, nil
}

func (session *session) run(ctx context.Context) error {
	sessionCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	session.stateMu.Lock()
	session.sessionCtx = sessionCtx
	session.stateMu.Unlock()

	defer func() {
		cancel()
		session.requests.Wait()
		session.stopEventRelay()
		session.releaseAllLocalSubscriptions()
		session.releaseAllStreams()
		session.closeProvider()
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
		if !session.isHelloAccepted() {
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

func (session *session) handleControlFrame(ctx context.Context, typ uint8, payload []byte) error {
	switch typ {
	case wire.TypeHello:
		if session.isHelloAccepted() {
			return session.sendError(0, protocolErrorBadRequest, "protocol Hello was already accepted")
		}
		hello, err := internalprotocol.DecodeHelloPayload(payload)
		if err != nil {
			return session.sendError(0, protocolErrorBadRequest, err.Error())
		}
		if hello.Version != 0 && hello.Version != wire.Version {
			return session.sendError(0, protocolErrorBadRequest, fmt.Sprintf("unsupported wire version %d", hello.Version))
		}
		response, err := internalprotocol.EncodeHelloPayload(internalprotocol.Hello{Version: wire.Version, Server: ModuleName})
		if err != nil {
			return err
		}
		if err := session.sendFrame(0, wire.TypeHello, response); err != nil {
			return err
		}
		session.stateMu.Lock()
		session.helloAccepted = true
		session.stateMu.Unlock()
		return nil
	case wire.TypeRequest:
		if !session.isHelloAccepted() {
			return session.sendError(0, protocolErrorBadRequest, "protocol Hello is required before requests")
		}
		request, err := internalprotocol.DecodeRequestPayload(payload)
		if err != nil {
			return session.sendError(0, protocolErrorBadRequest, err.Error())
		}
		if request.ID == 0 || request.Method == "" {
			return session.sendError(request.ID, protocolErrorBadRequest, "protocol request ID and method are required")
		}
		requestCtx, requestCancel := context.WithCancel(ctx)
		if !session.claimActiveRequest(request.ID, requestCancel) {
			requestCancel()
			_ = session.conn.Close()
			return fmt.Errorf("duplicate in-flight protocol request ID %d", request.ID)
		}
		select {
		case session.requestSlots <- struct{}{}:
		default:
			session.releaseActiveRequest(request.ID)
			return session.sendError(request.ID, protocolErrorExhausted, "protocol in-flight request capacity is exhausted")
		}
		session.requests.Add(1)
		go func() {
			defer session.requests.Done()
			defer session.releaseActiveRequest(request.ID)
			defer func() { <-session.requestSlots }()
			if err := session.handleRequest(requestCtx, request); err != nil {
				_ = session.conn.Close()
			}
		}()
		return nil
	case wire.TypeRequestCancel:
		if !session.isHelloAccepted() {
			return session.sendError(0, protocolErrorBadRequest, "protocol Hello is required before request cancellation")
		}
		id, err := internalprotocol.DecodeRequestCancelPayload(payload)
		if err != nil {
			return session.sendError(0, protocolErrorBadRequest, err.Error())
		}
		session.cancelActiveRequest(id)
		return nil
	case wire.TypeSessionClose:
		if err := internalprotocol.DecodeSessionClosePayload(payload); err != nil {
			return session.sendError(0, protocolErrorBadRequest, err.Error())
		}
		return io.EOF
	default:
		return session.sendError(0, protocolErrorBadRequest, fmt.Sprintf("unsupported control frame type %d", typ))
	}
}

func (session *session) handleRequest(ctx context.Context, request internalprotocol.Request) error {
	if request.Method != "api.execute" {
		return session.sendError(request.ID, protocolErrorNotFound, fmt.Sprintf("unknown method: %s", request.Method))
	}
	command, err := internalprotocol.DecodeApplicationCommand(request.Params)
	if err != nil {
		return session.sendError(request.ID, protocolErrorBadRequest, err.Error())
	}
	result, executeErr := session.routeCommand(ctx, command)
	if executeErr != nil {
		result = providerErrorResult(command, executeErr)
	}
	payload, err := internalprotocol.EncodeApplicationResult(result)
	if err != nil {
		return session.sendError(request.ID, protocolErrorInternal, err.Error())
	}
	response, err := internalprotocol.EncodeResponsePayload(internalprotocol.Response{ID: request.ID, Result: payload})
	if err != nil {
		return session.sendError(request.ID, protocolErrorInternal, err.Error())
	}
	return session.sendFrame(0, wire.TypeResponse, response)
}

// executeViaProvider 把一个 application command 交给 terminal provider：
//  1. 把 access-issued stream token 还原成 provider token；
//  2. 交给 provider 执行；
//  3. 对成功的 stream resource 发布结果分配 access channel/token 并桥接。
func (session *session) executeViaProvider(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	retired := session.rewriteRequestTokens(command)
	provider, err := session.ensureProvider(ctx)
	if err != nil {
		return nil, err
	}
	result, err := provider.Execute(ctx, command)
	if err != nil {
		return nil, err
	}
	if result.GetError() != nil {
		return result, nil
	}
	session.retireBindings(retired)
	if err := session.publishResult(result); err != nil {
		return nil, err
	}
	session.registerLocalRoutes(command, result)
	return result, nil
}

// rewriteRequestTokens 把命令中引用的 access token 还原成 provider token，
// 并返回需要在成功执行后释放的 access stream binding。
func (session *session) rewriteRequestTokens(command *apipb.CommandEnvelope) []*streamBinding {
	if command == nil {
		return nil
	}
	type tokenRef struct {
		resource *apipb.ResourceHandle
		retire   bool
	}
	var resources []tokenRef
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_TerminalDetach:
		resources = append(resources, tokenRef{value.TerminalDetach.GetAttachment(), true})
	case *apipb.CommandEnvelope_TerminalInput:
		resources = append(resources, tokenRef{value.TerminalInput.GetAttachment(), false})
	case *apipb.CommandEnvelope_TerminalResize:
		resources = append(resources, tokenRef{value.TerminalResize.GetAttachment(), false})
	case *apipb.CommandEnvelope_TerminalResizeLock:
		resources = append(resources, tokenRef{value.TerminalResizeLock.GetAttachment(), false})
	case *apipb.CommandEnvelope_ReleaseResource:
		resources = append(resources, tokenRef{value.ReleaseResource.GetResource(), true})
	case *apipb.CommandEnvelope_FileTransferCancel:
		resources = append(resources, tokenRef{value.FileTransferCancel.GetTransfer(), true})
	}
	retired := make([]*streamBinding, 0, len(resources))
	for _, ref := range resources {
		if ref.resource == nil || len(ref.resource.GetOpaqueToken()) == 0 {
			continue
		}
		session.streamsMu.Lock()
		binding := session.tokens[string(ref.resource.GetOpaqueToken())]
		session.streamsMu.Unlock()
		if binding == nil || binding.local != nil {
			// 本地资源不进入 provider 路径；ReleaseResource 已在 routeCommand 本地应答。
			continue
		}
		ref.resource.OpaqueToken = append([]byte(nil), binding.providerToken...)
		if ref.retire {
			retired = append(retired, binding)
		}
	}
	return retired
}

// publishResult 处理 provider 成功结果：为 stream resource 分配 access channel/token，
// 启动桥接；非 stream resource（history token、event subscription）原样透传。
func (session *session) publishResult(result *apipb.ResultEnvelope) error {
	resource, eager := streamResourceFromResult(result)
	if resource == nil {
		return nil
	}
	binding, err := session.registerStreamResource(resource)
	if err != nil {
		return err
	}
	if !eager {
		return nil
	}
	ctx := session.lifetimeContext()
	stream, err := session.providerForBinding().OpenStream(ctx, binding.providerResource)
	if err != nil {
		session.retireBinding(binding)
		return err
	}
	binding.mu.Lock()
	binding.stream = stream
	binding.mu.Unlock()
	go session.forwardProviderFrames(binding)
	return nil
}

func streamResourceFromResult(result *apipb.ResultEnvelope) (*apipb.ResourceHandle, bool) {
	switch value := result.GetResult().(type) {
	case *apipb.ResultEnvelope_TerminalAttach:
		return value.TerminalAttach.GetAttachment().GetResource(), false
	case *apipb.ResultEnvelope_FileTransferOpen:
		return value.FileTransferOpen.GetTransfer().GetResource(), true
	case *apipb.ResultEnvelope_BrowserProxyOpen:
		return value.BrowserProxyOpen.GetResource(), true
	default:
		return nil, false
	}
}

func (session *session) registerStreamResource(resource *apipb.ResourceHandle) (*streamBinding, error) {
	providerToken := resource.GetOpaqueToken()
	if len(providerToken) < 2 {
		return nil, fmt.Errorf("access/server: %s resource token is malformed", resource.GetKind())
	}
	providerChannel := binary.BigEndian.Uint16(providerToken[:2])
	if providerChannel == 0 {
		return nil, fmt.Errorf("access/server: %s resource channel is missing", resource.GetKind())
	}
	token := make([]byte, accessTokenBytes)
	if _, err := rand.Read(token); err != nil {
		return nil, fmt.Errorf("access/server: allocate resource token: %w", err)
	}
	session.streamsMu.Lock()
	clientChannel, err := session.allocateChannelLocked()
	if err != nil {
		session.streamsMu.Unlock()
		return nil, err
	}
	binary.BigEndian.PutUint16(token[:2], clientChannel)
	binding := &streamBinding{
		clientChannel:    clientChannel,
		kind:             resource.GetKind(),
		accessToken:      token,
		providerToken:    append([]byte(nil), providerToken...),
		providerChannel:  providerChannel,
		providerResource: proto.Clone(resource).(*apipb.ResourceHandle),
	}
	session.channels[clientChannel] = binding
	session.tokens[string(token)] = binding
	session.streamsMu.Unlock()
	resource.OpaqueToken = token
	return binding, nil
}

func (session *session) allocateChannelLocked() (uint16, error) {
	if session.nextChannel >= maxProtocolChannelID {
		return 0, fmt.Errorf("access/server: stream channel IDs are exhausted")
	}
	session.nextChannel++
	return uint16(session.nextChannel), nil
}

// allocateLocalChannel 为 access-local resource 分配客户端 channel。
func (session *session) allocateLocalChannel() (uint16, error) {
	session.streamsMu.Lock()
	defer session.streamsMu.Unlock()
	return session.allocateChannelLocked()
}

// releaseLocalChannel 在本地资源建立失败时归还尚未发布的 channel。
func (session *session) releaseLocalChannel(channel uint16) {
	if channel == 0 {
		return
	}
	session.streamsMu.Lock()
	delete(session.channels, channel)
	session.streamsMu.Unlock()
}

// registerLocalBinding 发布一个 access-local stream：channel/token 归 access 所有，
// 资源终态或 session 结束时自动回收。
func (session *session) registerLocalBinding(kind apipb.ResourceKind, token []byte, channel uint16, localID string, resource localResource) *streamBinding {
	binding := &streamBinding{
		clientChannel: channel,
		kind:          kind,
		accessToken:   append([]byte(nil), token...),
		localID:       localID,
		local:         resource,
	}
	session.streamsMu.Lock()
	session.channels[channel] = binding
	session.tokens[string(token)] = binding
	session.streamsMu.Unlock()
	if resource != nil {
		go func() {
			select {
			case <-resource.Done():
				session.retireBinding(binding)
			case <-session.lifetimeContext().Done():
			}
		}()
	}
	return binding
}

// localBindingForToken 返回当前 session 持有的本地资源 binding。
func (session *session) localBindingForToken(token []byte) *streamBinding {
	if len(token) == 0 {
		return nil
	}
	session.streamsMu.Lock()
	binding := session.tokens[string(token)]
	session.streamsMu.Unlock()
	if binding == nil || binding.local == nil {
		return nil
	}
	return binding
}

// handleStreamFrame 处理客户端 stream frame：attachment 走 bootstrap/close 握手，
// provider file/browser 原样双向转发，access-local 资源交给资源自己处理。
func (session *session) handleStreamFrame(ctx context.Context, channel uint16, typ uint8, payload []byte) error {
	session.streamsMu.Lock()
	binding := session.channels[channel]
	session.streamsMu.Unlock()
	if binding == nil {
		// channel ID 在同一 session 内单调分配、从不重用；终态后迟到的 ack/close
		// 是幂等帧，静默忽略，避免把已经完成的 transfer 变成 stream error。
		session.streamsMu.Lock()
		_, wasRetired := session.retired[channel]
		session.streamsMu.Unlock()
		if wasRetired || typ == wire.TypeClosed && len(payload) == 0 {
			return nil
		}
		return fmt.Errorf("stream channel %d is not bound to a resource", channel)
	}
	if binding.local != nil {
		done, err := binding.local.HandleFrame(ctx, typ, payload)
		if done {
			session.retireBinding(binding)
		}
		return err
	}
	switch binding.kind {
	case apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT:
		return session.handleAttachmentFrame(binding, typ, payload)
	case apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER, apipb.ResourceKind_RESOURCE_KIND_BROWSER_PROXY:
		stream := binding.currentStream()
		if stream == nil {
			return fmt.Errorf("stream channel %d is not ready", channel)
		}
		return stream.Send(ctx, typ, payload)
	default:
		return fmt.Errorf("unsupported stream resource kind %s", binding.kind)
	}
}

func (session *session) handleAttachmentFrame(binding *streamBinding, typ uint8, payload []byte) error {
	switch typ {
	case wire.TypeBootstrapDone:
		if len(payload) != 0 {
			return errors.New("terminal attachment bootstrap payload must be empty")
		}
		session.startAttachmentStream(binding)
		return nil
	case wire.TypeClosed:
		if len(payload) != 0 {
			return errors.New("terminal attachment stream close payload must be empty")
		}
		session.stopAttachmentStream(binding)
		return nil
	default:
		return fmt.Errorf("unsupported attachment stream frame type %d", typ)
	}
}

// startAttachmentStream 在客户端 bootstrap 后打开 provider stream：
// 等待 provider ready 后回写 TypeStreamReady，再开始转发 PTY 输出。
func (session *session) startAttachmentStream(binding *streamBinding) {
	binding.mu.Lock()
	binding.generation++
	generation := binding.generation
	binding.mu.Unlock()

	provider, err := session.ensureProvider(session.lifetimeContext())
	if err != nil {
		_ = session.sendStreamError(binding.clientChannel, protocolErrorUnavailable, err.Error())
		return
	}
	go func() {
		stream, err := provider.OpenStream(session.lifetimeContext(), binding.providerResource)
		binding.mu.Lock()
		if err != nil || binding.generation != generation || binding.stream != nil {
			binding.mu.Unlock()
			if stream != nil {
				_ = stream.Close()
			}
			if err != nil {
				_ = session.sendStreamError(binding.clientChannel, protocolErrorUnavailable, err.Error())
			}
			return
		}
		binding.stream = stream
		binding.mu.Unlock()
		if err := session.sendFrame(binding.clientChannel, wire.TypeStreamReady, nil); err != nil {
			return
		}
		session.forwardProviderFrames(binding)
	}()
}

// stopAttachmentStream 停止当前 provider 输出流，但保留 attachment resource，
// 客户端可以重新 bootstrap。
func (session *session) stopAttachmentStream(binding *streamBinding) {
	binding.mu.Lock()
	binding.generation++
	stream := binding.stream
	binding.stream = nil
	binding.mu.Unlock()
	if stream != nil {
		_ = stream.Close()
	}
}

// forwardProviderFrames 把 provider stream 的 frame 原样转发到客户端 channel。
func (session *session) forwardProviderFrames(binding *streamBinding) {
	stream := binding.currentStream()
	if stream == nil {
		return
	}
	ctx := session.lifetimeContext()
	for {
		typ, payload, err := stream.Receive(ctx)
		if err != nil {
			session.notifyStreamFailure(binding)
			return
		}
		if err := session.sendFrame(binding.clientChannel, typ, payload); err != nil {
			return
		}
		if streamFrameTerminal(binding.kind, typ) {
			return
		}
	}
}

func (session *session) notifyStreamFailure(binding *streamBinding) {
	if session.lifetimeContext().Err() != nil {
		return
	}
	// provider 连接中断时，客户端只关心 stream 终止；显式发送 closed 让上层
	// 不用等待 transport 超时。
	payload := wire.EncodeClosedPayload(-1)
	_ = session.sendFrame(binding.clientChannel, wire.TypeClosed, payload)
}

func streamFrameTerminal(kind apipb.ResourceKind, typ uint8) bool {
	switch kind {
	case apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT:
		return typ == wire.TypeClosed || typ == wire.TypeSyncLost
	case apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER:
		return typ == wire.TypeFileFinish || typ == wire.TypeFileResult || typ == wire.TypeClosed || typ == wire.TypeSyncLost
	case apipb.ResourceKind_RESOURCE_KIND_BROWSER_PROXY:
		return typ == wire.TypeBrowserClosed || typ == wire.TypeClosed || typ == wire.TypeSyncLost
	default:
		return true
	}
}

func (binding *streamBinding) currentStream() terminalprovider.Stream {
	if binding == nil {
		return nil
	}
	binding.mu.Lock()
	defer binding.mu.Unlock()
	return binding.stream
}

// ensureProvider 返回当前可用的 provider；连接缺失或已终止时按需重连。
func (session *session) ensureProvider(ctx context.Context) (terminalprovider.Provider, error) {
	session.providerMu.Lock()
	defer session.providerMu.Unlock()
	if session.provider != nil {
		select {
		case <-session.provider.Done():
			// provider 连接终止时，旧 session 发布的 attachment/file resource
			// 已经失效；先释放 bridge，避免客户端继续拿到 stale token。
			_ = session.provider.Close()
			session.provider = nil
			session.releaseAllStreams()
		default:
			return session.provider, nil
		}
	}
	provider, err := session.server.cfg.Provider(ctx)
	if err != nil {
		return nil, fmt.Errorf("terminal provider unavailable: %w", err)
	}
	session.provider = provider
	session.startEventRelay(provider)
	return provider, nil
}

func (session *session) providerForBinding() terminalprovider.Provider {
	session.providerMu.Lock()
	defer session.providerMu.Unlock()
	return session.provider
}

// startEventRelay 把 provider application event 原样转发成 channel-0 event frame。
// 每个 provider 连接最多一个 relay；provider 更换时旧 relay 随 ctx 结束。
func (session *session) startEventRelay(provider terminalprovider.Provider) {
	session.eventsMu.Lock()
	if session.eventProvider == provider {
		session.eventsMu.Unlock()
		return
	}
	if session.eventCancel != nil {
		session.eventCancel()
	}
	ctx, cancel := context.WithCancel(session.lifetimeContext())
	session.eventCancel = cancel
	session.eventProvider = provider
	session.eventsMu.Unlock()

	events, err := provider.Events(ctx)
	if err != nil {
		return
	}
	session.eventWG.Add(1)
	go func() {
		defer session.eventWG.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(event)
				if err != nil {
					continue
				}
				if err := session.sendFrame(0, wire.TypeEvent, payload); err != nil {
					return
				}
			}
		}
	}()
}

func (session *session) stopEventRelay() {
	session.eventsMu.Lock()
	cancel := session.eventCancel
	session.eventCancel = nil
	session.eventProvider = nil
	session.eventsMu.Unlock()
	if cancel != nil {
		cancel()
	}
	session.eventWG.Wait()
}

func (session *session) closeProvider() {
	session.providerMu.Lock()
	provider := session.provider
	session.provider = nil
	session.providerMu.Unlock()
	if provider != nil {
		_ = provider.Close()
	}
}

// retireBinding 释放一个 access stream binding：解除 registry 并停止桥接，
// 但 provider resource 的释放由 provider 命令负责。
func (session *session) retireBinding(binding *streamBinding) {
	if binding == nil {
		return
	}
	session.streamsMu.Lock()
	if session.channels[binding.clientChannel] == binding {
		delete(session.channels, binding.clientChannel)
	}
	if session.tokens[string(binding.accessToken)] == binding {
		delete(session.tokens, string(binding.accessToken))
	}
	session.retired[binding.clientChannel] = struct{}{}
	session.streamsMu.Unlock()
	if binding.local != nil {
		_ = binding.local.Close()
		return
	}
	binding.mu.Lock()
	binding.generation++
	stream := binding.stream
	binding.stream = nil
	binding.mu.Unlock()
	if stream != nil {
		_ = stream.Close()
	}
}

func (session *session) retireBindings(bindings []*streamBinding) {
	for _, binding := range bindings {
		session.retireBinding(binding)
	}
}

func (session *session) releaseAllStreams() {
	session.streamsMu.Lock()
	bindings := make([]*streamBinding, 0, len(session.channels))
	for _, binding := range session.channels {
		bindings = append(bindings, binding)
	}
	for channel := range session.channels {
		session.retired[channel] = struct{}{}
	}
	session.channels = make(map[uint16]*streamBinding)
	session.tokens = make(map[string]*streamBinding)
	session.streamsMu.Unlock()
	for _, binding := range bindings {
		if binding.local != nil {
			_ = binding.local.Close()
			continue
		}
		binding.mu.Lock()
		binding.generation++
		stream := binding.stream
		binding.stream = nil
		binding.mu.Unlock()
		if stream != nil {
			_ = stream.Close()
		}
	}
}

func (session *session) claimActiveRequest(id uint64, cancel context.CancelFunc) bool {
	session.requestMu.Lock()
	defer session.requestMu.Unlock()
	if _, exists := session.activeRequests[id]; exists {
		return false
	}
	session.activeRequests[id] = cancel
	return true
}

func (session *session) releaseActiveRequest(id uint64) {
	session.requestMu.Lock()
	cancel := session.activeRequests[id]
	delete(session.activeRequests, id)
	session.requestMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (session *session) cancelActiveRequest(id uint64) {
	session.requestMu.Lock()
	cancel := session.activeRequests[id]
	session.requestMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (session *session) isHelloAccepted() bool {
	session.stateMu.RLock()
	defer session.stateMu.RUnlock()
	return session.helloAccepted
}

func (session *session) lifetimeContext() context.Context {
	session.stateMu.RLock()
	defer session.stateMu.RUnlock()
	if session.sessionCtx != nil {
		return session.sessionCtx
	}
	return context.Background()
}

func (session *session) sendFrame(channel uint16, typ uint8, payload []byte) error {
	frame, err := wire.EncodeFrame(channel, typ, payload)
	if err != nil {
		return err
	}
	session.sendMu.Lock()
	defer session.sendMu.Unlock()
	return session.conn.Send(frame)
}

func (session *session) sendError(id uint64, code int, message string) error {
	payload, err := internalprotocol.EncodeErrorPayload(internalprotocol.ErrorMessage{
		ID:    id,
		Error: internalprotocol.ProtocolError{Code: code, Message: message},
	})
	if err != nil {
		return err
	}
	return session.sendFrame(0, wire.TypeError, payload)
}

func (session *session) sendStreamError(channel uint16, code int, message string) error {
	payload, err := internalprotocol.EncodeErrorPayload(internalprotocol.ErrorMessage{Error: internalprotocol.ProtocolError{Code: code, Message: message}})
	if err != nil {
		return err
	}
	return session.sendFrame(channel, wire.TypeError, payload)
}

// providerErrorResult 把 provider/路由 Go error 映射为带 correlation 的 typed envelope。
// 错误码沿用公共 ApiErrorCode，不新增客户端可见错误码。
func providerErrorResult(command *apipb.CommandEnvelope, err error) *apipb.ResultEnvelope {
	if command == nil {
		command = &apipb.CommandEnvelope{}
	}
	requestContext := command.GetContext()
	result := &apipb.ResultEnvelope{
		RequestId:     requestContext.GetRequestId(),
		OriginSession: cloneSessionStamp(requestContext.GetSession()),
	}
	var routeErr *routeError
	if errors.As(err, &routeErr) {
		result.Result = &apipb.ResultEnvelope_Error{Error: routeErr.apiError}
		return result
	}
	apiError := &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE, Message: "terminal provider is unavailable", Retryable: true}
	switch {
	case errors.Is(err, errUnsupportedCommand):
		apiError = &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST, Message: err.Error()}
	case errors.Is(err, terminalprovider.ErrUnsupported):
		apiError.Message = "terminal provider capability is unsupported"
	case errors.Is(err, context.Canceled):
		apiError = &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_CANCELLED, Message: "request was cancelled"}
	default:
		apiError.Message = err.Error()
	}
	var requestErr *internalprotocol.RequestError
	if errors.As(err, &requestErr) {
		apiError = apiErrorFromRequestError(requestErr)
	}
	result.Result = &apipb.ResultEnvelope_Error{Error: apiError}
	return result
}

// routeError 是 access-local handler 返回的 typed command 错误。
// 它已经带有最终 ApiError，不再经过 provider 错误分类。
type routeError struct {
	apiError *apipb.ApiError
}

func (err *routeError) Error() string {
	if err == nil || err.apiError == nil {
		return "access route error"
	}
	return err.apiError.GetMessage()
}

func cloneSessionStamp(stamp *apipb.EndpointSessionStamp) *apipb.EndpointSessionStamp {
	if stamp == nil {
		return nil
	}
	return proto.Clone(stamp).(*apipb.EndpointSessionStamp)
}

func apiErrorFromRequestError(err *internalprotocol.RequestError) *apipb.ApiError {
	apiError := &apipb.ApiError{Message: err.Message}
	switch err.Code {
	case protocolErrorBadRequest:
		apiError.Code = apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST
	case protocolErrorForbidden:
		apiError.Code = apipb.ApiErrorCode_API_ERROR_CODE_FORBIDDEN
	case protocolErrorNotFound:
		apiError.Code = apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND
	case protocolErrorExhausted:
		apiError.Code = apipb.ApiErrorCode_API_ERROR_CODE_RESOURCE_EXHAUSTED
		apiError.Retryable = true
	case protocolErrorUnavailable:
		apiError.Code = apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE
		apiError.Retryable = true
	default:
		apiError.Code = apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL
	}
	return apiError
}

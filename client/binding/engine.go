package binding

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/bindingpb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/proto/wirepb"
	"google.golang.org/protobuf/proto"
)

const (
	// ABIVersion 是 C/JNI/WASM binding 符号与 EventEnvelope 语义版本。
	// 不兼容的 handle ownership、函数签名或事件 oneof 变更必须递增该值。
	ABIVersion uint32 = 6
	// MaxPayloadBytes 限制跨语言单次 protobuf 输入，防止 JNI/WASM 分配无界内存。
	MaxPayloadBytes      = 4 << 20
	defaultEventCapacity = 256
	maxLiveHandles       = 4096
	// defaultOpenTimeout 覆盖 signaling/answer、ICE/DataChannel、端到端鉴权和 Hello。
	defaultOpenTimeout = 40 * time.Second
)

var (
	// ErrClosed 表示 binding engine 已关闭，任何新操作都不能创建 session 或复活旧 handle。
	ErrClosed = errors.New("binding engine is closed")
	// ErrInvalidHandle 表示 opaque handle 不属于当前 engine 或类型不匹配。
	ErrInvalidHandle = errors.New("binding handle is invalid")
	// ErrHandleActive 表示调用方试图 release 仍在执行或仍未关闭的 handle。
	ErrHandleActive = errors.New("binding handle is still active")
)

// Host 是 binding 到跨端 Go Client Engine composition root 的唯一依赖。
// request 来自 bindingpb；返回 session 必须已经完成 transport、remote auth、Hello，并由 client/runtime 持有 generation truth。
type Host interface {
	// OpenSession 按 generated binding request 建立一个 ApplicationReadyPeerSession；context 取消必须释放未发布资源。
	OpenSession(context.Context, *bindingpb.OpenSessionRequest) (clientruntime.ApplicationReadyPeerSession, error)
}

type endpointLifecycleHost interface {
	WatchEndpoint(context.Context, endpoint.EndpointID) (<-chan clientruntime.EndpointEvent, error)
}

// Engine 持有当前跨语言实例的 operation/session handle registry 和有界事件队列。
// handle 只在单个 Engine 内有效且永不复用；业务资源仍由 apipb.ResourceHandle 与 API Layer 管理，不能混入本 registry。
type Engine struct {
	host Host

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	mu             sync.Mutex
	closed         bool
	nextHandle     uint64
	nextRenderer   uint64
	activeRenderer uint64
	operations     map[uint64]*operation
	sessions       map[uint64]*sessionRecord
	streams        map[uint64]*streamRecord
	renderers      map[uint64]*rendererRecord
	eventCapacity  int
	cleanupTTL     time.Duration
	openTimeout    time.Duration
	closeOnce      sync.Once
}

type operation struct {
	cancel        context.CancelFunc
	sessionHandle uint64
	rendererID    uint64
	done          bool
}

type sessionRecord struct {
	session      clientruntime.ApplicationReadyPeerSession
	rendererID   uint64
	activeOps    int
	closing      bool
	closeStarted bool
	closed       bool
}

type streamRecord struct {
	sendMu        sync.Mutex
	stream        clientruntime.ResourceStream
	sessionHandle uint64
	rendererID    uint64
	closed        bool
	uploadHash    hash.Hash
	uploadBytes   int64
	uploadStart   int64
}

type rendererRecord struct {
	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
	events    chan []byte
	emitMu    sync.Mutex
	sequence  uint64
	closeOnce sync.Once
}

type rendererCleanup struct {
	renderer   *rendererRecord
	operations []context.CancelFunc
	sessions   []clientruntime.ApplicationReadyPeerSession
	streams    []*streamRecord
}

// NewEngine 创建使用默认有界事件容量的 binding engine。
// host 缺失时立即失败，禁止创建运行期才 fallback 的半初始化实例。
func NewEngine(host Host) (*Engine, error) {
	return NewEngineWithEventCapacity(host, defaultEventCapacity)
}

// NewEngineWithEventCapacity 创建测试或平台指定容量的 binding engine。
// capacity 必须为正；队列满时生产者施加背压，结果和事件不得静默丢弃。
func NewEngineWithEventCapacity(host Host, capacity int) (*Engine, error) {
	if host == nil {
		return nil, fmt.Errorf("binding host is required")
	}
	if capacity <= 0 {
		return nil, fmt.Errorf("binding event capacity must be positive")
	}
	ctx, cancel := context.WithCancel(context.Background())
	engine := &Engine{
		host: host, ctx: ctx, cancel: cancel, done: make(chan struct{}),
		operations: make(map[uint64]*operation), sessions: make(map[uint64]*sessionRecord), streams: make(map[uint64]*streamRecord),
		renderers: make(map[uint64]*rendererRecord), eventCapacity: capacity,
		cleanupTTL: 5 * time.Second, openTimeout: defaultOpenTimeout,
	}
	engine.activeRenderer = engine.newRendererLocked()
	return engine, nil
}

// AttachRenderer creates a fresh handle and event arena, fencing and cleaning the previous renderer.
// The Go runtime and its pooled physical endpoint sessions remain owned by Host.
func (engine *Engine) AttachRenderer() (uint64, error) {
	if engine == nil {
		return 0, ErrClosed
	}
	engine.mu.Lock()
	if engine.closed {
		engine.mu.Unlock()
		return 0, ErrClosed
	}
	if engine.nextRenderer == ^uint64(0) {
		engine.mu.Unlock()
		return 0, fmt.Errorf("binding renderer sequence is exhausted")
	}
	previous := engine.activeRenderer
	rendererID := engine.newRendererLocked()
	cleanup := engine.detachRendererLocked(previous)
	engine.mu.Unlock()
	_ = cleanup.run()
	return rendererID, nil
}

// DetachRenderer atomically revokes every handle created by rendererID.
// A stale detach is harmless and cannot affect a newer renderer.
func (engine *Engine) DetachRenderer(rendererID uint64) error {
	if engine == nil || rendererID == 0 {
		return nil
	}
	engine.mu.Lock()
	cleanup := engine.detachRendererLocked(rendererID)
	engine.mu.Unlock()
	return cleanup.run()
}

func (engine *Engine) newRendererLocked() uint64 {
	engine.nextRenderer++
	ctx, cancel := context.WithCancel(engine.ctx)
	engine.renderers[engine.nextRenderer] = &rendererRecord{
		ctx: ctx, cancel: cancel, done: make(chan struct{}), events: make(chan []byte, engine.eventCapacity),
	}
	engine.activeRenderer = engine.nextRenderer
	return engine.nextRenderer
}

func (engine *Engine) detachRendererLocked(rendererID uint64) rendererCleanup {
	record := engine.renderers[rendererID]
	if record == nil {
		return rendererCleanup{}
	}
	delete(engine.renderers, rendererID)
	if engine.activeRenderer == rendererID {
		engine.activeRenderer = 0
	}
	cleanup := rendererCleanup{renderer: record}
	for handle, operation := range engine.operations {
		if operation.rendererID != rendererID {
			continue
		}
		cleanup.operations = append(cleanup.operations, operation.cancel)
		delete(engine.operations, handle)
	}
	for handle, stream := range engine.streams {
		if stream.rendererID != rendererID {
			continue
		}
		stream.closed = true
		cleanup.streams = append(cleanup.streams, stream)
		delete(engine.streams, handle)
	}
	for handle, session := range engine.sessions {
		if session.rendererID != rendererID {
			continue
		}
		session.closed = true
		cleanup.sessions = append(cleanup.sessions, session.session)
		delete(engine.sessions, handle)
	}
	return cleanup
}

func (cleanup rendererCleanup) run() error {
	if cleanup.renderer == nil {
		return nil
	}
	cleanup.renderer.closeOnce.Do(func() {
		cleanup.renderer.cancel()
		close(cleanup.renderer.done)
	})
	for _, cancel := range cleanup.operations {
		cancel()
	}
	var first error
	for _, stream := range cleanup.streams {
		stream.sendMu.Lock()
		err := stream.stream.Close()
		stream.sendMu.Unlock()
		if err != nil && first == nil {
			first = err
		}
	}
	for _, session := range cleanup.sessions {
		if err := session.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// OpenResourceStream 解析 bindingpb.OpenResourceStreamRequest，并把 session-bound framing stream 注册为 opaque handle。
// resource 必须由同一 ready session 的 API result 签发；binding 不解析 opaque token 内部 channel 编号。
func (engine *Engine) OpenResourceStream(sessionHandle uint64, payload []byte) (uint64, error) {
	if err := validatePayload(payload); err != nil {
		return 0, err
	}
	request := &bindingpb.OpenResourceStreamRequest{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, request); err != nil {
		return 0, fmt.Errorf("decode resource stream request: %w", err)
	}
	if request.GetResource() == nil {
		return 0, fmt.Errorf("resource stream handle is required")
	}
	if request.GetInitialUploadOffset() < 0 {
		return 0, fmt.Errorf("initial upload offset must not be negative")
	}
	if request.GetResource().GetKind() == apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT {
		if request.GetInitialUploadOffset() != 0 {
			return 0, fmt.Errorf("terminal attachment stream does not accept an upload offset")
		}
	}
	session, err := engine.activeSession(sessionHandle)
	if err != nil {
		return 0, err
	}
	provider, ok := session.(clientruntime.ResourceStreamSession)
	if !ok {
		return 0, fmt.Errorf("application session does not support resource streams")
	}
	stream, err := provider.OpenResourceStream(proto.Clone(request.GetResource()).(*apipb.ResourceHandle))
	if err != nil {
		return 0, err
	}
	if stream == nil {
		return 0, fmt.Errorf("application session returned no resource stream")
	}
	engine.mu.Lock()
	if engine.closed {
		engine.mu.Unlock()
		_ = stream.Close()
		return 0, ErrClosed
	}
	sessionRecord := engine.sessions[sessionHandle]
	if sessionRecord == nil || sessionRecord.session != session || sessionRecord.closing || sessionRecord.closed {
		engine.mu.Unlock()
		_ = stream.Close()
		return 0, ErrInvalidHandle
	}
	handle, err := engine.allocateHandleLocked()
	if err == nil {
		engine.streams[handle] = &streamRecord{
			stream: stream, sessionHandle: sessionHandle, rendererID: sessionRecord.rendererID, uploadHash: sha256.New(),
			uploadBytes: request.GetInitialUploadOffset(), uploadStart: request.GetInitialUploadOffset(),
		}
	}
	engine.mu.Unlock()
	if err != nil {
		_ = stream.Close()
		return 0, err
	}
	go engine.forwardResourceStream(handle, stream)
	return handle, nil
}

// SendResourceStreamFrame 发送 serialized bindingpb.ResourceStreamFrame。
// frame handle 必须与参数一致，且只允许受控 file stream frame type，防止借 binding 写 terminal framing。
func (engine *Engine) SendResourceStreamFrame(streamHandle uint64, payload []byte) error {
	if err := validatePayload(payload); err != nil {
		return err
	}
	frame := &bindingpb.ResourceStreamFrame{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, frame); err != nil {
		return fmt.Errorf("decode resource stream frame: %w", err)
	}
	if frame.GetStreamHandle() != 0 && frame.GetStreamHandle() != streamHandle {
		return fmt.Errorf("resource stream frame handle mismatch")
	}
	record, err := engine.activeStreamRecord(streamHandle)
	if err != nil {
		return err
	}
	record.sendMu.Lock()
	defer record.sendMu.Unlock()
	typ, err := bindingFrameTypeToWire(frame.GetType(), true)
	if err != nil {
		return err
	}
	payloadSnapshot := append([]byte(nil), frame.GetPayload()...)
	if frame.GetType() == bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_DATA {
		var data wirepb.FileTransferData
		if err := proto.Unmarshal(payloadSnapshot, &data); err != nil {
			return fmt.Errorf("decode upload data frame: %w", err)
		}
		engine.mu.Lock()
		if err := engine.validateStreamSendLocked(streamHandle, record); err != nil {
			engine.mu.Unlock()
			return err
		}
		if data.GetOffset() != record.uploadBytes || len(data.GetData()) == 0 {
			engine.mu.Unlock()
			return fmt.Errorf("upload data frame offset is invalid")
		}
		engine.mu.Unlock()
		if err := record.stream.Send(engine.ctx, typ, payloadSnapshot); err != nil {
			return err
		}
		engine.mu.Lock()
		_, _ = record.uploadHash.Write(data.GetData())
		record.uploadBytes += int64(len(data.GetData()))
		engine.mu.Unlock()
		return nil
	}
	if frame.GetType() == bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH_AUTO {
		if len(payloadSnapshot) != 0 {
			return fmt.Errorf("automatic upload finish payload must be empty")
		}
		engine.mu.Lock()
		if record.uploadStart != 0 {
			engine.mu.Unlock()
			return fmt.Errorf("automatic upload finish is unavailable for resumed streams")
		}
		payloadSnapshot, err = proto.Marshal(&wirepb.FileTransferFinish{Size: record.uploadBytes, Sha256: record.uploadHash.Sum(nil)})
		engine.mu.Unlock()
		if err != nil {
			return err
		}
	}
	if frame.GetType() == bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH {
		var finish wirepb.FileTransferFinish
		if err := proto.Unmarshal(payloadSnapshot, &finish); err != nil {
			return fmt.Errorf("decode upload finish frame: %w", err)
		}
		engine.mu.Lock()
		uploadBytes := record.uploadBytes
		engine.mu.Unlock()
		if finish.GetSize() != uploadBytes || len(finish.GetSha256()) != sha256.Size {
			return fmt.Errorf("upload finish does not match sent bytes")
		}
	}
	engine.mu.Lock()
	err = engine.validateStreamSendLocked(streamHandle, record)
	engine.mu.Unlock()
	if err != nil {
		return err
	}
	return record.stream.Send(engine.ctx, typ, payloadSnapshot)
}

// CloseResourceStream 关闭 opaque stream handle；关闭后 handle 仍需 Release 才从 registry 删除。
func (engine *Engine) CloseResourceStream(handle uint64) error {
	engine.mu.Lock()
	if engine.closed {
		engine.mu.Unlock()
		return ErrClosed
	}
	record := engine.streams[handle]
	if record == nil || record.rendererID != engine.activeRenderer {
		engine.mu.Unlock()
		return ErrInvalidHandle
	}
	if record.closed {
		engine.mu.Unlock()
		return nil
	}
	record.closed = true
	engine.mu.Unlock()
	record.sendMu.Lock()
	defer record.sendMu.Unlock()
	return record.stream.Close()
}

// OpenSession 解析 bindingpb.OpenSessionRequest 并异步请求 Host 建立 session。
// 返回 operation handle；完成结果只能通过 NextEvent 的 bindingpb.EventEnvelope 获取。
func (engine *Engine) OpenSession(payload []byte) (uint64, error) {
	if err := validatePayload(payload); err != nil {
		return 0, err
	}
	request := &bindingpb.OpenSessionRequest{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, request); err != nil {
		return 0, fmt.Errorf("decode open session request: %w", err)
	}
	if request.GetRequestId() == "" || request.GetEndpointId() == "" {
		return 0, fmt.Errorf("open session request is incomplete")
	}
	switch request.GetIntent() {
	case bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE,
		bindingpb.ConnectIntent_CONNECT_INTENT_BACKGROUND,
		bindingpb.ConnectIntent_CONNECT_INTENT_PROBE:
	default:
		return 0, fmt.Errorf("open session intent is unsupported")
	}
	handle, operationContext, err := engine.startOperation()
	if err != nil {
		return 0, err
	}
	go engine.runOpen(handle, operationContext, request)
	return handle, nil
}

// Execute 解析完整 apipb.CommandEnvelope，并在指定 session 上异步执行。
// command unknown fields 保留到 Go Client Engine；binding 不解释业务 oneof，也不按 command 类型扩张导出函数。
func (engine *Engine) Execute(sessionHandle uint64, payload []byte) (uint64, error) {
	if err := validatePayload(payload); err != nil {
		return 0, err
	}
	command := &apipb.CommandEnvelope{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, command); err != nil {
		return 0, fmt.Errorf("decode application command: %w", err)
	}
	if command.GetCommand() == nil {
		return 0, fmt.Errorf("application command is required")
	}
	handle, operationContext, session, err := engine.startSessionOperation(sessionHandle)
	if err != nil {
		return 0, err
	}
	go engine.runExecute(handle, sessionHandle, operationContext, session, command)
	return handle, nil
}

// NextEvent 阻塞读取下一条 bindingpb.EventEnvelope protobuf bytes。
// 返回内容是独立副本；调用方拥有该 buffer，C/WASM wrapper 必须提供匹配的显式 free/release 语义。
func (engine *Engine) NextEvent(ctx context.Context) ([]byte, error) {
	if engine == nil {
		return nil, ErrClosed
	}
	engine.mu.Lock()
	rendererID := engine.activeRenderer
	engine.mu.Unlock()
	return engine.NextEventForRenderer(ctx, rendererID)
}

// NextEventForRenderer reads only events emitted by rendererID's arena.
func (engine *Engine) NextEventForRenderer(ctx context.Context, rendererID uint64) ([]byte, error) {
	if engine == nil {
		return nil, ErrClosed
	}
	engine.mu.Lock()
	if engine.closed {
		engine.mu.Unlock()
		return nil, ErrClosed
	}
	renderer := engine.renderers[rendererID]
	engine.mu.Unlock()
	if renderer == nil {
		return nil, ErrInvalidHandle
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-engine.done:
		return nil, ErrClosed
	case <-renderer.done:
		return nil, ErrInvalidHandle
	case payload := <-renderer.events:
		return append([]byte(nil), payload...), nil
	}
}

// Cancel 取消仍在执行的 open/execute operation。
// cancellation 只影响该 operation；最终 CANCELLED 结果仍通过事件队列交付，调用方之后必须 Release operation handle。
func (engine *Engine) Cancel(operationHandle uint64) error {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if engine.closed {
		return ErrClosed
	}
	operation := engine.operations[operationHandle]
	if operation == nil || operation.rendererID != engine.activeRenderer {
		return ErrInvalidHandle
	}
	if operation.done {
		return ErrHandleActive
	}
	operation.cancel()
	return nil
}

// CloseSession 请求关闭指定 session，但保留已关闭 handle 直到显式 Release。
// lifecycle goroutine 负责发布唯一 SessionClosedEvent；重复 close 不创建第二个事件或新 generation。
func (engine *Engine) CloseSession(sessionHandle uint64) error {
	engine.mu.Lock()
	if engine.closed {
		engine.mu.Unlock()
		return ErrClosed
	}
	record := engine.sessions[sessionHandle]
	if record == nil || record.rendererID != engine.activeRenderer {
		engine.mu.Unlock()
		return ErrInvalidHandle
	}
	if record.closed || record.closing {
		engine.mu.Unlock()
		return nil
	}
	record.closing = true
	session := record.session
	closeSession := record.activeOps == 0
	if closeSession {
		record.closeStarted = true
	}
	streams := make([]*streamRecord, 0)
	for _, streamRecord := range engine.streams {
		if streamRecord.sessionHandle == sessionHandle && !streamRecord.closed {
			streamRecord.closed = true
			streams = append(streams, streamRecord)
		}
	}
	engine.mu.Unlock()
	var first error
	for _, stream := range streams {
		stream.sendMu.Lock()
		err := stream.stream.Close()
		stream.sendMu.Unlock()
		if err != nil && first == nil {
			first = err
		}
	}
	if closeSession {
		if err := session.Close(); err != nil {
			if first == nil {
				first = err
			}
			engine.finishSession(sessionHandle, session, err)
		}
	}
	return first
}

// Release 从本地 handle registry 删除已经完成的 operation 或已经关闭的 session。
// 活动 handle 不能释放；该函数不替代 apipb.ReleaseResourceCommand，daemon-owned resource 必须继续走 Proto API。
func (engine *Engine) Release(handle uint64) error {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if engine.closed {
		return ErrClosed
	}
	if operation := engine.operations[handle]; operation != nil {
		if operation.rendererID != engine.activeRenderer {
			return ErrInvalidHandle
		}
		if !operation.done {
			return ErrHandleActive
		}
		delete(engine.operations, handle)
		return nil
	}
	if record := engine.sessions[handle]; record != nil {
		if record.rendererID != engine.activeRenderer {
			return ErrInvalidHandle
		}
		if !record.closed {
			return ErrHandleActive
		}
		delete(engine.sessions, handle)
		return nil
	}
	if record := engine.streams[handle]; record != nil {
		if record.rendererID != engine.activeRenderer {
			return ErrInvalidHandle
		}
		if !record.closed {
			return ErrHandleActive
		}
		delete(engine.streams, handle)
		return nil
	}
	return ErrInvalidHandle
}

// Close 取消全部 operation、关闭全部 session，并解除 NextEvent/背压等待。
// 方法幂等；它不释放平台 wrapper 自己分配的输出 buffer，wrapper 仍须执行对应 buffer free。
func (engine *Engine) Close() error {
	if engine == nil {
		return nil
	}
	var first error
	engine.closeOnce.Do(func() {
		engine.mu.Lock()
		engine.closed = true
		renderers := make([]*rendererRecord, 0, len(engine.renderers))
		for rendererID, renderer := range engine.renderers {
			renderers = append(renderers, renderer)
			delete(engine.renderers, rendererID)
		}
		engine.activeRenderer = 0
		for _, operation := range engine.operations {
			operation.cancel()
		}
		sessions := make([]clientruntime.ApplicationReadyPeerSession, 0, len(engine.sessions))
		for _, record := range engine.sessions {
			if !record.closed && !record.closeStarted {
				record.closeStarted = true
				sessions = append(sessions, record.session)
			}
		}
		streams := make([]*streamRecord, 0, len(engine.streams))
		for _, record := range engine.streams {
			streams = append(streams, record)
			record.closed = true
		}
		engine.mu.Unlock()
		engine.cancel()
		close(engine.done)
		for _, renderer := range renderers {
			renderer.closeOnce.Do(func() {
				renderer.cancel()
				close(renderer.done)
			})
		}
		for _, stream := range streams {
			stream.sendMu.Lock()
			err := stream.stream.Close()
			stream.sendMu.Unlock()
			if err != nil && first == nil {
				first = err
			}
		}
		for _, session := range sessions {
			if err := session.Close(); err != nil && first == nil {
				first = err
			}
		}
	})
	return first
}

func (engine *Engine) forwardResourceStream(handle uint64, stream clientruntime.ResourceStream) {
	var terminalError error
	defer func() {
		engine.mu.Lock()
		if record := engine.streams[handle]; record != nil {
			record.closed = true
		}
		engine.mu.Unlock()
		event := &bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_ResourceStreamClosed{ResourceStreamClosed: &bindingpb.ResourceStreamClosedEvent{StreamHandle: handle}}}
		if terminalError != nil && !errors.Is(terminalError, io.EOF) && !errors.Is(terminalError, context.Canceled) {
			event.GetResourceStreamClosed().Error = apiError(terminalError)
		}
		engine.emitForHandle(handle, event)
	}()
	for {
		typ, payload, err := stream.Receive(engine.ctx)
		if err != nil {
			terminalError = err
			return
		}
		bindingType, err := wireFrameTypeToBinding(typ)
		if err != nil {
			terminalError = err
			return
		}
		payload, err = bindingPayloadForWireFrame(typ, payload)
		if err != nil {
			terminalError = err
			return
		}
		engine.emitForHandle(handle, &bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_ResourceStreamFrame{ResourceStreamFrame: &bindingpb.ResourceStreamFrame{
			StreamHandle: handle, Type: bindingType, Payload: append([]byte(nil), payload...),
		}}})
	}
}

func (engine *Engine) runOpen(handle uint64, ctx context.Context, request *bindingpb.OpenSessionRequest) {
	// OpenSession 的完成条件是 transport、端到端鉴权和 Protocol Hello 全部 ready；
	// 任一阶段失去回调都必须由 Go binding operation 统一取消，不能让平台 UI 永久等待。
	openCtx, cancel := context.WithTimeout(ctx, engine.openTimeout)
	defer cancel()
	stopLifecycle := engine.forwardOpenLifecycle(handle, request)
	session, err := engine.host.OpenSession(openCtx, request)
	if err == nil {
		if session == nil || session.Done() == nil || session.Stamp().Validate() != nil {
			if session != nil {
				_ = session.Close()
			}
			session = nil
			err = fmt.Errorf("binding host returned an invalid ready session")
		}
	}
	var sessionHandle uint64
	if err == nil {
		engine.mu.Lock()
		operation := engine.operations[handle]
		if engine.closed || operation == nil || openCtx.Err() != nil {
			engine.mu.Unlock()
			_ = session.Close()
			err = context.Canceled
			session = nil
		} else {
			var allocateErr error
			sessionHandle, allocateErr = engine.allocateHandleLocked()
			if allocateErr == nil {
				engine.sessions[sessionHandle] = &sessionRecord{session: session, rendererID: operation.rendererID}
			}
			engine.mu.Unlock()
			if allocateErr != nil {
				_ = session.Close()
				err = allocateErr
				session = nil
			}
		}
	}
	engine.markOperationDone(handle)
	stopLifecycle()
	event := &bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_OpenSession{OpenSession: &bindingpb.OpenSessionResult{
		RequestId: request.GetRequestId(), OperationHandle: handle, SessionHandle: sessionHandle,
	}}}
	if session != nil {
		event.GetOpenSession().Session = runtimeStampToProto(session.Stamp())
		if provider, ok := session.(clientruntime.ConnectionSnapshotProvider); ok {
			if snapshot, valid := provider.ConnectionSnapshot(time.Now().UTC()); valid {
				event.GetOpenSession().Connection = connectionSnapshotToProto(snapshot)
			}
		}
	} else {
		event.GetOpenSession().Error = apiError(err)
	}
	engine.emitForHandle(handle, event)
	if session != nil {
		go engine.forwardSession(sessionHandle, session)
	}
}

func (engine *Engine) forwardOpenLifecycle(handle uint64, request *bindingpb.OpenSessionRequest) func() {
	host, ok := engine.host.(endpointLifecycleHost)
	if !ok || request == nil || strings.TrimSpace(request.GetEndpointId()) == "" {
		return func() {}
	}
	ctx, cancel := context.WithCancel(engine.ctx)
	events, err := host.WatchEndpoint(ctx, endpoint.EndpointID(strings.TrimSpace(request.GetEndpointId())))
	if err != nil {
		cancel()
		return func() {}
	}
	done := make(chan struct{})
	stop := make(chan struct{})
	var stopOnce sync.Once
	go func() {
		defer close(done)
		for {
			var event clientruntime.EndpointEvent
			var ok bool
			select {
			case <-stop:
				return
			case event, ok = <-events:
				if !ok {
					return
				}
			}
			// WatchEndpoint starts with the state before this open operation. An
			// idle snapshot would make a newly-started connection jump backwards.
			if event.Phase == clientruntime.EndpointPhaseIdle {
				continue
			}
			engine.emitForHandle(handle, endpointConnectionEvent(handle, request.GetRequestId(), event))
		}
	}()
	return func() {
		stopOnce.Do(func() { close(stop) })
		cancel()
		<-done
	}
}

func endpointConnectionEvent(handle uint64, requestID string, event clientruntime.EndpointEvent) *bindingpb.EventEnvelope {
	value := &bindingpb.EndpointConnectionEvent{
		RequestId: requestID, OperationHandle: handle, EndpointId: string(event.EndpointID),
		Phase: endpointConnectionPhase(event.Phase), ObservedPath: bindingObservedPath(event.ObservedPath),
		RouteSelectionReason: event.RouteSelectionReason,
	}
	if event.Stamp.Generation.Valid() {
		value.Session = runtimeStampToProto(event.Stamp)
	}
	if event.ErrorCode != "" || event.Message != "" {
		value.Error = apiError(&clientruntime.Error{Code: event.ErrorCode, Message: event.Message, Attempted: true})
	}
	return &bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_EndpointConnection{EndpointConnection: value}}
}

func endpointConnectionPhase(phase clientruntime.EndpointPhase) bindingpb.EndpointConnectionPhase {
	switch phase {
	case clientruntime.EndpointPhaseIdle:
		return bindingpb.EndpointConnectionPhase_ENDPOINT_CONNECTION_PHASE_IDLE
	case clientruntime.EndpointPhasePlanning:
		return bindingpb.EndpointConnectionPhase_ENDPOINT_CONNECTION_PHASE_PLANNING
	case clientruntime.EndpointPhaseResolving:
		return bindingpb.EndpointConnectionPhase_ENDPOINT_CONNECTION_PHASE_RESOLVING
	case clientruntime.EndpointPhaseSignaling:
		return bindingpb.EndpointConnectionPhase_ENDPOINT_CONNECTION_PHASE_SIGNALING
	case clientruntime.EndpointPhaseConnecting:
		return bindingpb.EndpointConnectionPhase_ENDPOINT_CONNECTION_PHASE_CONNECTING
	case clientruntime.EndpointPhaseAuthorizing:
		return bindingpb.EndpointConnectionPhase_ENDPOINT_CONNECTION_PHASE_AUTHORIZING
	case clientruntime.EndpointPhaseReady:
		return bindingpb.EndpointConnectionPhase_ENDPOINT_CONNECTION_PHASE_READY
	case clientruntime.EndpointPhaseOffline:
		return bindingpb.EndpointConnectionPhase_ENDPOINT_CONNECTION_PHASE_OFFLINE
	default:
		return bindingpb.EndpointConnectionPhase_ENDPOINT_CONNECTION_PHASE_UNSPECIFIED
	}
}

func connectionSnapshotToProto(snapshot clientruntime.ConnectionSnapshot) *bindingpb.ConnectionSnapshot {
	return &bindingpb.ConnectionSnapshot{
		RouteId: string(snapshot.RouteID), RouteKind: bindingRouteKind(snapshot.RouteKind),
		ObservedPath: bindingObservedPath(snapshot.ObservedPath), SelectionReason: snapshot.SelectionReason,
		SampledAtUnixNano: snapshot.SampledAt.UTC().UnixNano(), RoundTripNanos: int64(snapshot.RoundTrip),
		LocalCandidateType: bindingCandidateType(snapshot.LocalCandidateType), RemoteCandidateType: bindingCandidateType(snapshot.RemoteCandidateType),
		LocalIp: snapshot.LocalAddress, RemoteIp: snapshot.RemoteAddress,
		LocalPort: uint32(snapshot.LocalPort), RemotePort: uint32(snapshot.RemotePort),
		CandidatePairId: snapshot.PairID,
		LocalRelatedIp:  snapshot.LocalRelatedAddress, LocalRelatedPort: uint32(snapshot.LocalRelatedPort),
		RemoteRelatedIp: snapshot.RemoteRelatedAddress, RemoteRelatedPort: uint32(snapshot.RemoteRelatedPort),
		LocalProtocol: bindingConnectionTransport(snapshot.LocalProtocol), RemoteProtocol: bindingConnectionTransport(snapshot.RemoteProtocol),
		RelayTransport: bindingConnectionTransport(snapshot.RelayTransport), NetworkClass: snapshot.NetworkClass,
		BytesSent: snapshot.BytesSent, BytesReceived: snapshot.BytesReceived, PacketsSent: snapshot.PacketsSent,
		LossEvents: snapshot.LossEvents, Connected: snapshot.Connected,
	}
}

func bindingRouteKind(kind endpoint.RouteKind) bindingpb.ConnectionRouteKind {
	switch kind {
	case endpoint.RouteLocalUnix:
		return bindingpb.ConnectionRouteKind_CONNECTION_ROUTE_KIND_LOCAL
	case endpoint.RouteDirectWebRTCTCP:
		return bindingpb.ConnectionRouteKind_CONNECTION_ROUTE_KIND_DIRECT
	case endpoint.RouteSSHWebRTCTCP:
		return bindingpb.ConnectionRouteKind_CONNECTION_ROUTE_KIND_SSH
	case endpoint.RouteManagedWebRTC:
		return bindingpb.ConnectionRouteKind_CONNECTION_ROUTE_KIND_CLOUD
	default:
		return bindingpb.ConnectionRouteKind_CONNECTION_ROUTE_KIND_UNSPECIFIED
	}
}

func bindingObservedPath(path string) bindingpb.ConnectionObservedPath {
	switch endpoint.Path(path) {
	case endpoint.PathDirect:
		return bindingpb.ConnectionObservedPath_CONNECTION_OBSERVED_PATH_DIRECT
	case endpoint.PathSingleRelay:
		return bindingpb.ConnectionObservedPath_CONNECTION_OBSERVED_PATH_SINGLE_RELAY
	default:
		return bindingpb.ConnectionObservedPath_CONNECTION_OBSERVED_PATH_UNSPECIFIED
	}
}

func bindingCandidateType(value string) bindingpb.ConnectionCandidateType {
	switch value {
	case "host":
		return bindingpb.ConnectionCandidateType_CONNECTION_CANDIDATE_TYPE_HOST
	case "srflx":
		return bindingpb.ConnectionCandidateType_CONNECTION_CANDIDATE_TYPE_SERVER_REFLEXIVE
	case "prflx":
		return bindingpb.ConnectionCandidateType_CONNECTION_CANDIDATE_TYPE_PEER_REFLEXIVE
	case "relay":
		return bindingpb.ConnectionCandidateType_CONNECTION_CANDIDATE_TYPE_RELAY
	default:
		return bindingpb.ConnectionCandidateType_CONNECTION_CANDIDATE_TYPE_UNSPECIFIED
	}
}

func bindingConnectionTransport(value string) bindingpb.ConnectionTransport {
	switch value {
	case "udp":
		return bindingpb.ConnectionTransport_CONNECTION_TRANSPORT_UDP
	case "tcp", "tls":
		return bindingpb.ConnectionTransport_CONNECTION_TRANSPORT_TCP
	default:
		return bindingpb.ConnectionTransport_CONNECTION_TRANSPORT_UNSPECIFIED
	}
}

func (engine *Engine) runExecute(handle, sessionHandle uint64, ctx context.Context, session clientruntime.ApplicationReadyPeerSession, command *apipb.CommandEnvelope) {
	var result *apipb.ResultEnvelope
	var err error
	if requiresTerminalResponse(command) {
		if executor, ok := session.(clientruntime.TerminalResponseApplicationExecutor); ok {
			result, err = executor.ExecuteApplicationTerminal(ctx, command)
		} else {
			result, err = session.ExecuteApplication(ctx, command)
		}
	} else {
		result, err = session.ExecuteApplication(ctx, command)
	}
	if ctx.Err() != nil {
		if cleanupErr := engine.cleanupCancelledApplicationResult(session, command, result); cleanupErr != nil {
			cleanupErr = fmt.Errorf("cancelled application resource cleanup: %w", cleanupErr)
			if invalidateErr := invalidateSessionAfterCleanupFailure(session, cleanupErr); invalidateErr != nil {
				err = fmt.Errorf("%v; invalidate session: %w", cleanupErr, invalidateErr)
			} else {
				err = cleanupErr
			}
		} else {
			err = ctx.Err()
		}
		result = nil
	}
	engine.markOperationDone(handle)
	event := &bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_Execute{Execute: &bindingpb.ExecuteResult{
		OperationHandle: handle, SessionHandle: sessionHandle,
	}}}
	if err != nil {
		event.GetExecute().Error = apiError(err)
	} else if result == nil {
		event.GetExecute().Error = apiError(fmt.Errorf("application session returned no result"))
	} else {
		event.GetExecute().Result = proto.Clone(result).(*apipb.ResultEnvelope)
	}
	engine.emitForHandle(handle, event)
}

func invalidateSessionAfterCleanupFailure(session clientruntime.ApplicationReadyPeerSession, cause error) error {
	if invalidator, ok := session.(clientruntime.ApplicationSessionInvalidator); ok {
		return invalidator.InvalidateApplicationSession(cause)
	}
	// 非 runtime owner 的测试或嵌入实现没有 generation invalidator 时，至少必须关闭其完整 session。
	return session.Close()
}

// cleanupCancelledApplicationResult destroys a resource that reached the binding
// after its cross-language consumer had already cancelled the operation.
func (engine *Engine) cleanupCancelledApplicationResult(session clientruntime.ApplicationReadyPeerSession, original *apipb.CommandEnvelope, result *apipb.ResultEnvelope) error {
	cleanupCommand, confirmUpload := cancelledApplicationCleanupCommand(original, result)
	if cleanupCommand == nil {
		return nil
	}
	cleanupCtx, cancel := context.WithTimeout(engine.ctx, engine.cleanupTTL)
	defer cancel()
	cleanup, err := session.ExecuteApplication(cleanupCtx, cleanupCommand)
	if err != nil {
		if cleanupCtx.Err() != nil {
			return fmt.Errorf("cancelled application resource cleanup timed out")
		}
		return err
	}
	if confirmUpload && (cleanup.GetFileTransferCancel() == nil || !cleanup.GetFileTransferCancel().GetCancelled()) {
		return fmt.Errorf("upload cancellation was not confirmed")
	}
	return nil
}

func cancelledApplicationCleanupCommand(original *apipb.CommandEnvelope, result *apipb.ResultEnvelope) (*apipb.CommandEnvelope, bool) {
	if transfer := result.GetFileTransferOpen().GetTransfer(); transfer != nil && transfer.GetResource() != nil {
		if transfer.GetResume() != nil {
			return &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_FileTransferCancel{FileTransferCancel: &apipb.FileTransferCancelCommand{UploadResume: proto.Clone(transfer.GetResume()).(*apipb.FileUploadResumeHandle)}}}, true
		}
		return &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_ReleaseResource{ReleaseResource: &apipb.ReleaseResourceCommand{Resource: proto.Clone(transfer.GetResource()).(*apipb.ResourceHandle)}}}, false
	}
	if window := result.GetHistoryWindow(); requiresTerminalResponse(original) && window != nil && window.GetTerminal() != nil && window.GetToken() != "" {
		return &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_HistoryRelease{HistoryRelease: &apipb.HistoryReleaseCommand{
			Terminal: proto.Clone(window.GetTerminal()).(*apipb.TerminalRef), Token: window.GetToken(), HistoryGeneration: window.GetHistoryGeneration(),
		}}}, false
	}
	if subscription := result.GetEventSubscription().GetSubscription(); subscription != nil {
		return &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_ReleaseResource{ReleaseResource: &apipb.ReleaseResourceCommand{Resource: proto.Clone(subscription).(*apipb.ResourceHandle)}}}, false
	}
	return nil, false
}

func (engine *Engine) forwardSession(handle uint64, session clientruntime.ApplicationReadyPeerSession) {
	eventCtx, cancelEvents := context.WithCancel(engine.ctx)
	defer cancelEvents()
	events, err := session.ApplicationEvents(eventCtx)
	if err != nil {
		_ = session.Close()
		engine.finishSession(handle, session, err)
		return
	}
	for {
		select {
		case <-engine.done:
			return
		case <-session.Done():
			engine.finishSession(handle, session, sessionTerminalError(session))
			return
		case event, ok := <-events:
			if !ok {
				terminalErr, observed := awaitSessionTerminalError(engine.done, session)
				if observed {
					engine.finishSession(handle, session, terminalErr)
				}
				return
			}
			if event != nil {
				engine.emitForHandle(handle, &bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_Application{Application: &bindingpb.ApplicationEvent{
					SessionHandle: handle, Event: proto.Clone(event).(*apipb.EventEnvelope),
				}}})
			}
		}
	}
}

func awaitSessionTerminalError(engineDone <-chan struct{}, session clientruntime.ApplicationReadyPeerSession) (error, bool) {
	select {
	case <-engineDone:
		return nil, false
	case <-session.Done():
		return sessionTerminalError(session), true
	}
}

func sessionTerminalError(session clientruntime.ApplicationReadyPeerSession) error {
	terminalErr := session.Err()
	closeErr := session.Close()
	if terminalErr != nil {
		return terminalErr
	}
	if terminalErr = session.Err(); terminalErr != nil {
		return terminalErr
	}
	if closeErr != nil {
		return closeErr
	}
	return io.EOF
}

func (engine *Engine) finishSession(handle uint64, session clientruntime.ApplicationReadyPeerSession, err error) {
	engine.mu.Lock()
	record := engine.sessions[handle]
	if record == nil || record.session != session || record.closed {
		engine.mu.Unlock()
		return
	}
	record.closed = true
	engine.mu.Unlock()
	event := &bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_SessionClosed{SessionClosed: &bindingpb.SessionClosedEvent{
		SessionHandle: handle, Session: runtimeStampToProto(session.Stamp()),
	}}}
	if err != nil && !errors.Is(err, io.EOF) {
		event.GetSessionClosed().Error = apiError(err)
	}
	engine.emitForHandle(handle, event)
}

func (engine *Engine) startOperation() (uint64, context.Context, error) {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	renderer := engine.renderers[engine.activeRenderer]
	if engine.closed {
		return 0, nil, ErrClosed
	}
	if renderer == nil {
		return 0, nil, ErrInvalidHandle
	}
	handle, err := engine.allocateHandleLocked()
	if err != nil {
		return 0, nil, err
	}
	ctx, cancel := context.WithCancel(renderer.ctx)
	engine.operations[handle] = &operation{cancel: cancel, rendererID: engine.activeRenderer}
	return handle, ctx, nil
}

func (engine *Engine) startSessionOperation(sessionHandle uint64) (uint64, context.Context, clientruntime.ApplicationReadyPeerSession, error) {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if engine.closed {
		return 0, nil, nil, ErrClosed
	}
	record := engine.sessions[sessionHandle]
	if record == nil || record.rendererID != engine.activeRenderer || record.closing || record.closed {
		return 0, nil, nil, ErrInvalidHandle
	}
	renderer := engine.renderers[record.rendererID]
	if renderer == nil {
		return 0, nil, nil, ErrInvalidHandle
	}
	handle, err := engine.allocateHandleLocked()
	if err != nil {
		return 0, nil, nil, err
	}
	ctx, cancel := context.WithCancel(renderer.ctx)
	engine.operations[handle] = &operation{cancel: cancel, sessionHandle: sessionHandle, rendererID: record.rendererID}
	record.activeOps++
	return handle, ctx, record.session, nil
}

func (engine *Engine) allocateHandleLocked() (uint64, error) {
	if len(engine.operations)+len(engine.sessions)+len(engine.streams) >= maxLiveHandles || engine.nextHandle == ^uint64(0) {
		return 0, fmt.Errorf("binding handle capacity is exhausted")
	}
	engine.nextHandle++
	if engine.nextHandle == 0 {
		return 0, fmt.Errorf("binding handle sequence is exhausted")
	}
	return engine.nextHandle, nil
}

func (engine *Engine) markOperationDone(handle uint64) {
	var closeSession clientruntime.ApplicationReadyPeerSession
	var closeSessionHandle uint64
	engine.mu.Lock()
	if operation := engine.operations[handle]; operation != nil {
		operation.done = true
		operation.cancel()
		if operation.sessionHandle != 0 {
			if record := engine.sessions[operation.sessionHandle]; record != nil {
				if record.activeOps > 0 {
					record.activeOps--
				}
				if record.closing && record.activeOps == 0 && !record.closeStarted && !record.closed {
					record.closeStarted = true
					closeSession = record.session
					closeSessionHandle = operation.sessionHandle
				}
			}
		}
	}
	engine.mu.Unlock()
	if closeSession != nil {
		if err := closeSession.Close(); err != nil {
			engine.finishSession(closeSessionHandle, closeSession, err)
		}
	}
}

func (engine *Engine) activeSession(handle uint64) (clientruntime.ApplicationReadyPeerSession, error) {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if engine.closed {
		return nil, ErrClosed
	}
	record := engine.sessions[handle]
	if record == nil || record.rendererID != engine.activeRenderer || record.closing || record.closed {
		return nil, ErrInvalidHandle
	}
	return record.session, nil
}

func (engine *Engine) activeStreamRecord(handle uint64) (*streamRecord, error) {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if engine.closed {
		return nil, ErrClosed
	}
	record := engine.streams[handle]
	if record == nil || record.rendererID != engine.activeRenderer || record.closed {
		return nil, ErrInvalidHandle
	}
	return record, nil
}

func (engine *Engine) validateStreamSendLocked(handle uint64, record *streamRecord) error {
	if engine.closed {
		return ErrClosed
	}
	if record == nil || engine.streams[handle] != record || record.rendererID != engine.activeRenderer || record.closed {
		return ErrInvalidHandle
	}
	session := engine.sessions[record.sessionHandle]
	if session == nil || session.rendererID != record.rendererID || session.closing || session.closed {
		return ErrInvalidHandle
	}
	return nil
}

func bindingFrameTypeToWire(typ bindingpb.ResourceStreamFrameType, outbound bool) (uint8, error) {
	switch typ {
	case bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_DATA:
		return wire.TypeFileData, nil
	case bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_ACK:
		return wire.TypeFileAck, nil
	case bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH:
		return wire.TypeFileFinish, nil
	case bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_RESULT:
		if !outbound {
			return wire.TypeFileResult, nil
		}
	case bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_ERROR:
		if !outbound {
			return wire.TypeError, nil
		}
	case bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH_AUTO:
		if outbound {
			return wire.TypeFileFinish, nil
		}
	case bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_PTY_OUTPUT:
		if !outbound {
			return wire.TypePTYOutput, nil
		}
	case bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_PTY_SYNC_LOST:
		if !outbound {
			return wire.TypeSyncLost, nil
		}
	case bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_PTY_CLOSED:
		if !outbound {
			return wire.TypeClosed, nil
		}
	}
	return 0, fmt.Errorf("resource stream frame type is unsupported")
}

func wireFrameTypeToBinding(typ uint8) (bindingpb.ResourceStreamFrameType, error) {
	for _, candidate := range []bindingpb.ResourceStreamFrameType{
		bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_DATA,
		bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_ACK,
		bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH,
		bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_RESULT,
		bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_ERROR,
		bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_PTY_OUTPUT,
		bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_PTY_SYNC_LOST,
		bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_PTY_CLOSED,
	} {
		wireType, _ := bindingFrameTypeToWire(candidate, false)
		if wireType == typ {
			return candidate, nil
		}
	}
	return bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_UNSPECIFIED, fmt.Errorf("resource stream received unsupported frame type %d", typ)
}

func bindingPayloadForWireFrame(typ uint8, payload []byte) ([]byte, error) {
	switch typ {
	case wire.TypeSyncLost:
		droppedBytes, err := wire.DecodeSyncLostPayload(payload)
		if err != nil {
			return nil, err
		}
		return proto.Marshal(&bindingpb.PTYStreamSyncLost{DroppedBytes: droppedBytes})
	case wire.TypeClosed:
		exitCode, err := wire.DecodeClosedPayload(payload)
		if err != nil {
			return nil, err
		}
		return proto.Marshal(&bindingpb.PTYStreamClosed{ExitCode: int32(exitCode)})
	default:
		return append([]byte(nil), payload...), nil
	}
}

func (engine *Engine) emitForHandle(handle uint64, event *bindingpb.EventEnvelope) {
	if engine == nil || handle == 0 || event == nil {
		return
	}
	engine.mu.Lock()
	rendererID := uint64(0)
	if operation := engine.operations[handle]; operation != nil {
		rendererID = operation.rendererID
	} else if session := engine.sessions[handle]; session != nil {
		rendererID = session.rendererID
	} else if stream := engine.streams[handle]; stream != nil {
		rendererID = stream.rendererID
	}
	renderer := engine.renderers[rendererID]
	engine.mu.Unlock()
	if renderer == nil {
		return
	}
	renderer.emitMu.Lock()
	defer renderer.emitMu.Unlock()
	renderer.sequence++
	event.AbiVersion = ABIVersion
	event.Sequence = renderer.sequence
	payload, err := (proto.MarshalOptions{Deterministic: true}).Marshal(event)
	if err != nil {
		return
	}
	select {
	case <-engine.done:
	case <-renderer.done:
	case renderer.events <- payload:
	}
}

func validatePayload(payload []byte) error {
	if len(payload) == 0 {
		return fmt.Errorf("binding protobuf payload is empty")
	}
	if len(payload) > MaxPayloadBytes {
		return fmt.Errorf("binding protobuf payload exceeds %d bytes", MaxPayloadBytes)
	}
	return nil
}

func runtimeStampToProto(stamp clientruntime.EndpointSessionStamp) *apipb.EndpointSessionStamp {
	return &apipb.EndpointSessionStamp{
		EndpointId: string(stamp.EndpointID), RouteId: string(stamp.RouteID), Generation: uint64(stamp.Generation),
	}
}

func apiError(err error) *apipb.ApiError {
	if err == nil {
		return nil
	}
	code := apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE
	message := "binding operation failed"
	retryable := false
	attempted := true
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		code, message, retryable = apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE, "client session timed out", true
	case errors.Is(err, context.Canceled):
		code, message = apipb.ApiErrorCode_API_ERROR_CODE_CANCELLED, "binding operation was cancelled"
	case errors.Is(err, ErrInvalidHandle):
		code, message, attempted = apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST, "binding handle is invalid", false
	default:
		switch clientruntime.CodeOf(err) {
		case clientruntime.ErrorInvalidRequest:
			code, message, attempted = apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST, "client runtime rejected the request", false
		case clientruntime.ErrorIdentity, clientruntime.ErrorAuthorization:
			code, message = apipb.ApiErrorCode_API_ERROR_CODE_UNAUTHORIZED, "client authorization failed"
		case clientruntime.ErrorCanceled:
			code, message = apipb.ApiErrorCode_API_ERROR_CODE_CANCELLED, "client operation was cancelled"
		case clientruntime.ErrorStaleSession:
			code, message = apipb.ApiErrorCode_API_ERROR_CODE_STALE_SESSION, "client session is stale"
		case clientruntime.ErrorStaleResource:
			code, message = apipb.ApiErrorCode_API_ERROR_CODE_STALE_RESOURCE, "client resource is stale"
		case clientruntime.ErrorResourceExhausted:
			code, message = apipb.ApiErrorCode_API_ERROR_CODE_RESOURCE_EXHAUSTED, "client resource capacity is exhausted"
		case clientruntime.ErrorEntitlement:
			code, message = apipb.ApiErrorCode_API_ERROR_CODE_ENTITLEMENT_DENIED, "Relay is not included in the current AnyTTY Cloud plan"
		case clientruntime.ErrorRelayNotInPlan:
			code = apipb.ApiErrorCode_API_ERROR_CODE_RELAY_NOT_IN_PLAN
		case clientruntime.ErrorRelayQuotaExhausted:
			code = apipb.ApiErrorCode_API_ERROR_CODE_RELAY_QUOTA_EXHAUSTED
		case clientruntime.ErrorRelayConcurrencyExhausted:
			code = apipb.ApiErrorCode_API_ERROR_CODE_RELAY_CONCURRENCY_EXHAUSTED
		case clientruntime.ErrorSubscriptionInactive:
			code = apipb.ApiErrorCode_API_ERROR_CODE_SUBSCRIPTION_INACTIVE
		case clientruntime.ErrorRelayRegionUnavailable:
			code = apipb.ApiErrorCode_API_ERROR_CODE_RELAY_REGION_UNAVAILABLE
		case clientruntime.ErrorDaemonBlocked:
			code, message, retryable = apipb.ApiErrorCode_API_ERROR_CODE_DAEMON_BLOCKED, "daemon Cloud access is temporarily disabled", true
		case clientruntime.ErrorDaemonDeleted:
			code, message = apipb.ApiErrorCode_API_ERROR_CODE_DAEMON_DELETED, "daemon Cloud enrollment was deleted"
		case clientruntime.ErrorUnavailable, clientruntime.ErrorUnsupportedRoute:
			code, message, retryable = apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE, "client session is unavailable", true
		}
		var runtimeErr *clientruntime.Error
		if errors.As(err, &runtimeErr) {
			attempted = runtimeErr.Attempted
			retryable = runtimeErr.Retryable
			if strings.TrimSpace(runtimeErr.Message) != "" && (runtimeErr.Code == clientruntime.ErrorResourceExhausted || runtimeErr.Code == clientruntime.ErrorEntitlement || runtimeErr.Code == clientruntime.ErrorUnavailable ||
				runtimeErr.Code == clientruntime.ErrorRelayNotInPlan || runtimeErr.Code == clientruntime.ErrorRelayQuotaExhausted || runtimeErr.Code == clientruntime.ErrorRelayConcurrencyExhausted ||
				runtimeErr.Code == clientruntime.ErrorSubscriptionInactive || runtimeErr.Code == clientruntime.ErrorRelayRegionUnavailable) {
				message = runtimeErr.Message
			}
		}
	}
	return &apipb.ApiError{Code: code, Message: message, Retryable: retryable, Attempted: attempted}
}

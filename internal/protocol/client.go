package protocol

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"

	"github.com/anytty/anytty/shared/perftrace"
	"github.com/anytty/anytty/shared/transport"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	transport           transport.Transport
	nextID              atomic.Uint64
	streamPayloadBudget *streamPayloadBudget

	doneErrMu sync.Mutex
	doneErr   error
	closeOnce sync.Once
	closeErr  error

	mu                          sync.Mutex
	sendMu                      sync.Mutex
	waiters                     map[uint64]*responseWaiter
	abandonedWaiters            map[uint64]struct{}
	streams                     map[uint16]*clientStream
	pending                     map[uint16][]StreamFrame
	reused                      map[uint16][]StreamFrame
	pendingFrameCount           int
	pendingByteCount            int
	pendingLimits               clientPendingLimits
	dropped                     map[uint16]struct{}
	applicationEventSubscribers map[uint64]chan *apipb.EventEnvelope
	pendingApplicationEvents    []*apipb.EventEnvelope
	applicationAttachments      map[uint16]*apipb.ResourceHandle
	nextEventSubID              uint64
	helloReceived               bool

	helloCh chan result
	done    chan struct{}
}

type result struct {
	payload []byte
	err     error
}

type responseWaiter struct {
	ch        chan result
	delivered bool
}

type clientPendingLimits struct {
	channels         int
	frames           int
	framesPerChannel int
	bytes            int
}

var defaultClientPendingLimits = clientPendingLimits{
	channels:         64,
	frames:           1024,
	framesPerChannel: 256,
	bytes:            8 << 20,
}

const maxAbandonedWaiters = 256

type StreamFrame struct {
	Type    uint8
	Payload []byte
}

const (
	maxClientStreamPayloadBytes = 8 << 20
	maxClientPayloadBytes       = 64 << 20
)

type streamPayloadBudget struct {
	mu       sync.Mutex
	limit    int
	retained int
}

func newStreamPayloadBudget(limit int) *streamPayloadBudget {
	return &streamPayloadBudget{limit: limit}
}

func (b *streamPayloadBudget) reserve(size int) bool {
	if b == nil || size == 0 {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if size > b.limit-b.retained {
		return false
	}
	b.retained += size
	return true
}

func (b *streamPayloadBudget) release(size int) {
	if b == nil || size == 0 {
		return
	}
	b.mu.Lock()
	b.retained -= size
	if b.retained < 0 {
		b.mu.Unlock()
		panic("protocol: negative stream payload budget")
	}
	b.mu.Unlock()
}

type retainedStreamFrame struct {
	frame         StreamFrame
	payloadBytes  int
	countsAsFrame bool
}

type clientStream struct {
	mu                  sync.Mutex
	cond                *sync.Cond
	ch                  chan StreamFrame
	done                chan struct{}
	queue               []retainedStreamFrame
	queueLimit          int
	retainedFrames      int
	payloadBudget       *streamPayloadBudget
	sharedPayloadBudget *streamPayloadBudget
	closed              bool
	rawOverflow         bool
	rawSyncLostSent     bool
	rawDroppedBytes     uint64
	terminalAfterQueue  bool
}

func newClientStreamWithConfig(queueLimit, channelCapacity int) *clientStream {
	return newClientStreamWithLimits(queueLimit, channelCapacity, maxClientStreamPayloadBytes, newStreamPayloadBudget(maxClientPayloadBytes))
}

func newClientStreamWithSharedBudget(shared *streamPayloadBudget) *clientStream {
	return newClientStreamWithLimits(256, 0, maxClientStreamPayloadBytes, shared)
}

func newClientStreamWithLimits(queueLimit, channelCapacity, payloadLimit int, shared *streamPayloadBudget) *clientStream {
	if queueLimit <= 0 {
		queueLimit = 1
	}
	if channelCapacity < 0 {
		channelCapacity = 0
	}
	s := &clientStream{
		ch:                  make(chan StreamFrame, channelCapacity),
		done:                make(chan struct{}),
		queueLimit:          queueLimit,
		queue:               make([]retainedStreamFrame, 0, min(queueLimit, 16)),
		payloadBudget:       newStreamPayloadBudget(payloadLimit),
		sharedPayloadBudget: shared,
	}
	s.cond = sync.NewCond(&s.mu)
	go s.run()
	return s
}

func (s *clientStream) channel() chan StreamFrame {
	return s.ch
}

func (s *clientStream) send(frame StreamFrame) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.rawSyncLostSent {
		return
	}
	if s.rawOverflow {
		if frame.Type == wire.TypePTYOutput {
			s.addRawDroppedBytesLocked(len(frame.Payload))
		}
		return
	}
	if s.terminalAfterQueue {
		return
	}
	if s.retainedFrames >= s.queueLimit {
		s.overflowLocked(frame, "resource stream frame capacity is exhausted")
		return
	}
	if !s.reservePayloadLocked(len(frame.Payload)) {
		s.overflowLocked(frame, "resource stream payload capacity is exhausted")
		return
	}
	payload := frame.Payload
	if len(payload) > 0 {
		payload = append([]byte(nil), payload...)
	}
	s.queue = append(s.queue, retainedStreamFrame{
		frame:         StreamFrame{Type: frame.Type, Payload: payload},
		payloadBytes:  len(payload),
		countsAsFrame: true,
	})
	s.retainedFrames++
	if frame.Type == wire.TypeBrowserClosed {
		s.terminalAfterQueue = true
	}
	s.cond.Signal()
}

func (s *clientStream) overflowLocked(frame StreamFrame, message string) {
	if frame.Type == wire.TypePTYOutput {
		s.rawOverflow = true
		s.addRawDroppedBytesLocked(len(frame.Payload))
		s.cond.Signal()
		return
	}
	s.failLocked(ProtocolErrorCodeResourceExhausted, message)
}

func (s *clientStream) fail(code int, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.terminalAfterQueue || s.rawOverflow {
		return
	}
	s.failLocked(code, message)
}

func (s *clientStream) failLocked(code int, message string) {
	payload, err := EncodeErrorPayload(ErrorMessage{Error: ProtocolError{Code: code, Message: message}})
	if err != nil {
		s.closed = true
		close(s.done)
		s.resetQueueLocked()
		s.cond.Broadcast()
		return
	}
	s.resetQueueLocked()
	// The terminal error is locally created and bounded; it does not retain peer payload.
	s.queue = append(s.queue, retainedStreamFrame{frame: StreamFrame{Type: wire.TypeError, Payload: payload}})
	s.terminalAfterQueue = true
	s.cond.Signal()
}

func (s *clientStream) addRawDroppedBytesLocked(count int) {
	const maxSyncLostBytes = uint64(^uint32(0))
	if count <= 0 || s.rawDroppedBytes >= maxSyncLostBytes {
		return
	}
	next := uint64(count)
	if next > maxSyncLostBytes-s.rawDroppedBytes {
		s.rawDroppedBytes = maxSyncLostBytes
		return
	}
	s.rawDroppedBytes += next
}

func (s *clientStream) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	close(s.done)
	s.resetQueueLocked()
	s.cond.Broadcast()
}

func (s *clientStream) run() {
	ch := s.channel()
	defer close(ch)
	for {
		frame, ok := s.nextFrame()
		if !ok {
			return
		}
		select {
		case ch <- frame.frame:
			s.releaseFrame(frame)
		case <-s.done:
			s.releaseFrame(frame)
			return
		}
	}
}

func (s *clientStream) nextFrame() (retainedStreamFrame, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		if len(s.queue) > 0 {
			frame := s.queue[0]
			copy(s.queue, s.queue[1:])
			last := len(s.queue) - 1
			s.queue[last] = retainedStreamFrame{}
			s.queue = s.queue[:last]
			return frame, true
		}
		if s.closed {
			return retainedStreamFrame{}, false
		}
		if s.rawOverflow && !s.rawSyncLostSent {
			s.rawSyncLostSent = true
			return retainedStreamFrame{frame: StreamFrame{Type: wire.TypeSyncLost, Payload: wire.EncodeSyncLostPayload(s.rawDroppedBytes)}}, true
		}
		if s.rawSyncLostSent {
			s.closed = true
			close(s.done)
			return retainedStreamFrame{}, false
		}
		if s.terminalAfterQueue {
			s.closed = true
			close(s.done)
			return retainedStreamFrame{}, false
		}
		s.cond.Wait()
	}
}

func (s *clientStream) reservePayloadLocked(size int) bool {
	if !s.payloadBudget.reserve(size) {
		return false
	}
	if !s.sharedPayloadBudget.reserve(size) {
		s.payloadBudget.release(size)
		return false
	}
	return true
}

func (s *clientStream) releasePayloadLocked(size int) {
	s.payloadBudget.release(size)
	s.sharedPayloadBudget.release(size)
}

func (s *clientStream) releaseFrame(frame retainedStreamFrame) {
	if !frame.countsAsFrame {
		return
	}
	s.mu.Lock()
	s.retainedFrames--
	s.releasePayloadLocked(frame.payloadBytes)
	s.mu.Unlock()
}

func (s *clientStream) resetQueueLocked() {
	for index := range s.queue {
		frame := s.queue[index]
		if frame.countsAsFrame {
			s.retainedFrames--
			s.releasePayloadLocked(frame.payloadBytes)
		}
		s.queue[index] = retainedStreamFrame{}
	}
	s.queue = nil
}

func NewClient(t transport.Transport) *Client {
	return newClientWithPendingLimits(t, defaultClientPendingLimits)
}

func newClientWithPendingLimits(t transport.Transport, limits clientPendingLimits) *Client {
	c := &Client{
		transport:                   t,
		streamPayloadBudget:         newStreamPayloadBudget(maxClientPayloadBytes),
		waiters:                     make(map[uint64]*responseWaiter),
		abandonedWaiters:            make(map[uint64]struct{}),
		streams:                     make(map[uint16]*clientStream),
		pending:                     make(map[uint16][]StreamFrame),
		reused:                      make(map[uint16][]StreamFrame),
		pendingLimits:               limits.normalized(),
		dropped:                     make(map[uint16]struct{}),
		applicationEventSubscribers: make(map[uint64]chan *apipb.EventEnvelope),
		applicationAttachments:      make(map[uint16]*apipb.ResourceHandle),
		helloCh:                     make(chan result, 1),
		done:                        make(chan struct{}),
	}
	go c.readLoop()
	return c
}

// ApplicationEvents 注册 generated Proto application event 消费者。
// 事件 frame 只包含 EventEnvelope；filter 与 subscription lifecycle 由 daemon resource 管理。
func (c *Client) ApplicationEvents(ctx context.Context) (<-chan *apipb.EventEnvelope, error) {
	if c == nil || c.doneClosed() {
		return nil, io.EOF
	}
	out := make(chan *apipb.EventEnvelope, 64)
	c.mu.Lock()
	c.nextEventSubID++
	id := c.nextEventSubID
	pending := c.pendingApplicationEvents
	c.pendingApplicationEvents = nil
	raw := make(chan *apipb.EventEnvelope, len(pending)+64)
	for _, event := range pending {
		raw <- event
	}
	c.applicationEventSubscribers[id] = raw
	c.mu.Unlock()
	go func() {
		defer close(out)
		defer func() {
			c.mu.Lock()
			delete(c.applicationEventSubscribers, id)
			c.mu.Unlock()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.done:
				return
			case event := <-raw:
				select {
				case out <- event:
				case <-ctx.Done():
					return
				case <-c.done:
					return
				}
			}
		}
	}()
	return out, nil
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.closeOnce.Do(func() {
		select {
		case <-c.done:
		default:
			if drainer, ok := c.transport.(interface{ Drain(context.Context) error }); ok {
				if payload, err := EncodeSessionClosePayload(); err == nil {
					if frame, err := wire.EncodeFrame(0, wire.TypeSessionClose, payload); err == nil && c.send(frame) == nil {
						drainCtx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
						_ = drainer.Drain(drainCtx)
						cancel()
						select {
						case <-c.done:
						case <-time.After(250 * time.Millisecond):
						}
					}
				}
			}
		}
		c.closeErr = c.transport.Close()
	})
	<-c.done
	return c.closeErr
}

// Done 返回 protocol client read loop 的关闭信号。
// 调用方只能把它作为连接生命周期事件源，不能从这里推断 terminal lifecycle 或 history truth。
func (c *Client) Done() <-chan struct{} {
	if c == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return c.done
}

// Err 返回 read loop 结束时记录的 transport/protocol 错误。
// 该错误用于 client/TUI 展示 endpoint 断线原因；如果连接尚未关闭或正常关闭则返回 nil。
func (c *Client) Err() error {
	if c == nil {
		return io.EOF
	}
	select {
	case <-c.done:
	default:
		return nil
	}
	c.doneErrMu.Lock()
	defer c.doneErrMu.Unlock()
	return c.doneErr
}

func (c *Client) Hello(ctx context.Context, hello Hello) error {
	payload, err := EncodeHelloPayload(hello)
	if err != nil {
		return err
	}
	frame, err := wire.EncodeFrame(0, wire.TypeHello, payload)
	if err != nil {
		return err
	}
	if err := c.send(frame); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-c.helloCh:
		return res.err
	}
}

// ExecuteApplication 通过唯一 api.execute framing 发送公共 Proto API command。
// protocol client 只负责 correlation 与 payload transport，不解释 terminal application 字段。
func (c *Client) ExecuteApplication(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	payload, err := c.doApplicationRequest(ctx, command, false)
	if err != nil {
		return nil, err
	}
	result, err := DecodeApplicationResult(payload)
	if err != nil {
		return nil, err
	}
	c.updateApplicationAttachmentBinding(command, result)
	return result, nil
}

// ExecuteApplicationTerminal 使用通用 correlation 选项，在调用 context 取消后继续等待一个有界 terminal response。
// 是否选择该能力由上层 resource owner 决定；protocol 不检查 application oneof，也不负责业务清理。
func (c *Client) ExecuteApplicationTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	payload, err := c.doApplicationRequest(ctx, command, true)
	if err != nil {
		return nil, err
	}
	result, err := DecodeApplicationResult(payload)
	if err != nil {
		return nil, err
	}
	c.updateApplicationAttachmentBinding(command, result)
	return result, nil
}

// ApplicationAttachmentChannel 返回当前 protocol connection 为 attachment resource 分配的 stream channel。
// opaque token 的内部格式和 channel registry 只属于 framing binding，不得暴露为公共 API schema。
func (c *Client) ApplicationAttachmentChannel(resource *apipb.ResourceHandle) (uint16, bool) {
	if resource.GetKind() != apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT {
		return 0, false
	}
	return c.ApplicationResourceChannel(resource)
}

// ApplicationResourceChannel 返回当前 connection 为 stream resource 绑定的内部 channel。
// channel 只供 framing adapter 使用，不进入公共 Proto result。
func (c *Client) ApplicationResourceChannel(resource *apipb.ResourceHandle) (uint16, bool) {
	if resource == nil || len(resource.GetOpaqueToken()) < 2 {
		return 0, false
	}
	channel := binary.BigEndian.Uint16(resource.GetOpaqueToken()[:2])
	if channel == 0 {
		return 0, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	bound := c.applicationAttachments[channel]
	return channel, bound != nil && string(bound.GetOpaqueToken()) == string(resource.GetOpaqueToken())
}

// ApplicationAttachment 返回 stream channel 当前绑定的 Proto resource handle 副本。
func (c *Client) ApplicationAttachment(channel uint16) (*apipb.ResourceHandle, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	resource := c.applicationAttachments[channel]
	if resource == nil {
		return nil, false
	}
	return proto.Clone(resource).(*apipb.ResourceHandle), true
}

func (c *Client) updateApplicationAttachmentBinding(command *apipb.CommandEnvelope, result *apipb.ResultEnvelope) {
	if command == nil || result == nil || result.GetError() != nil {
		return
	}
	boundResource := result.GetTerminalAttach().GetAttachment().GetResource()
	if transfer := result.GetFileTransferOpen().GetTransfer(); transfer != nil {
		boundResource = transfer.GetResource()
	}
	if browser := result.GetBrowserProxyOpen().GetResource(); browser != nil {
		boundResource = browser
	}
	if boundResource != nil && len(boundResource.GetOpaqueToken()) >= 2 {
		channel := binary.BigEndian.Uint16(boundResource.GetOpaqueToken()[:2])
		if channel != 0 {
			c.bindApplicationAttachment(channel, boundResource)
		}
		return
	}
	var released *apipb.ResourceHandle
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_TerminalDetach:
		released = value.TerminalDetach.GetAttachment()
	case *apipb.CommandEnvelope_ReleaseResource:
		released = value.ReleaseResource.GetResource()
	case *apipb.CommandEnvelope_FileTransferCancel:
		released = value.FileTransferCancel.GetTransfer()
	}
	if channel, ok := c.ApplicationResourceChannel(released); ok {
		c.mu.Lock()
		delete(c.applicationAttachments, channel)
		c.mu.Unlock()
	}
}

func (c *Client) bindApplicationAttachment(channel uint16, resource *apipb.ResourceHandle) {
	c.mu.Lock()
	stream := c.streams[channel]
	if stream == nil {
		stream = newClientStreamWithSharedBudget(c.streamPayloadBudget)
		c.streams[channel] = stream
	}
	c.applicationAttachments[channel] = proto.Clone(resource).(*apipb.ResourceHandle)
	delete(c.dropped, channel)
	pending := c.takePendingFramesLocked(c.pending, channel)
	reused := c.takePendingFramesLocked(c.reused, channel)
	c.mu.Unlock()
	for _, frame := range pending {
		stream.send(frame)
	}
	for _, frame := range reused {
		stream.send(frame)
	}
}

// SendFileFrame 在已由 daemon 分配的 transfer channel 上发送流控 frame。
// typ 只允许 file data/ack/finish，避免调用方借该入口写 terminal channel。
func (c *Client) SendFileFrame(channel uint16, typ uint8, payload []byte) error {
	switch typ {
	case wire.TypeFileData, wire.TypeFileAck, wire.TypeFileFinish:
	default:
		return fmt.Errorf("unsupported client file frame type %d", typ)
	}
	frame, err := wire.EncodeFrame(channel, typ, payload)
	if err != nil {
		return err
	}
	return c.send(frame)
}

// SendBrowserFrame writes bytes or a close marker to a daemon-side TCP resource.
func (c *Client) SendBrowserFrame(channel uint16, typ uint8, payload []byte) error {
	switch typ {
	case wire.TypeBrowserData:
	case wire.TypeClosed:
		if len(payload) != 0 {
			return fmt.Errorf("browser proxy close payload must be empty")
		}
	default:
		return fmt.Errorf("unsupported client browser frame type %d", typ)
	}
	frame, err := wire.EncodeFrame(channel, typ, payload)
	if err != nil {
		return err
	}
	return c.send(frame)
}

// SendAttachmentReady 启动已绑定 attachment channel 的实时输出。
// ready 之前 daemon 不发送 PTY bytes，避免 attach result 与本地 stream consumer 建立之间出现竞态。
func (c *Client) SendAttachmentReady(channel uint16) error {
	if channel == 0 {
		return fmt.Errorf("attachment channel is required")
	}
	frame, err := wire.EncodeFrame(channel, wire.TypeBootstrapDone, nil)
	if err != nil {
		return err
	}
	return c.send(frame)
}

// SendAttachmentStreamClose 停止 attachment channel 的实时 PTY 输出，但不释放
// attachment resource；调用方之后可以在同一 resource 上重新打开输出流。
func (c *Client) SendAttachmentStreamClose(channel uint16) error {
	if channel == 0 {
		return fmt.Errorf("attachment channel is required")
	}
	frame, err := wire.EncodeFrame(channel, wire.TypeClosed, nil)
	if err != nil {
		return err
	}
	return c.send(frame)
}

func (c *Client) Stream(channel uint16) (<-chan StreamFrame, func()) {
	c.mu.Lock()
	stream := c.streams[channel]
	if stream == nil {
		if _, dropped := c.dropped[channel]; dropped {
			// attachment raw stream 在 process exit 后可以用同一个 attachment
			// resource 重新打开；ready 握手保证丢弃旧 channel 尾帧后不会漏掉新输出。
			delete(c.dropped, channel)
			c.discardPendingFramesLocked(c.reused, channel)
		}
		stream = newClientStreamWithSharedBudget(c.streamPayloadBudget)
		c.streams[channel] = stream
	}
	pending := c.takePendingFramesLocked(c.pending, channel)
	c.mu.Unlock()
	for _, frame := range pending {
		stream.send(frame)
	}

	return stream.channel(), func() {
		c.mu.Lock()
		if current, ok := c.streams[channel]; ok {
			delete(c.streams, channel)
			c.dropped[channel] = struct{}{}
			current.close()
		}
		c.discardPendingFramesLocked(c.pending, channel)
		c.mu.Unlock()
	}
}

func (c *Client) doApplicationRequest(ctx context.Context, command *apipb.CommandEnvelope, waitTerminalOnCancel bool) ([]byte, error) {
	const method = "api.execute"
	finish := perftrace.Measure("protocol.request." + method)
	payload, err := EncodeApplicationCommand(command)
	if err != nil {
		finish(0)
		return nil, err
	}
	id := c.nextID.Add(1)
	requestPayload, err := EncodeRequestPayload(Request{ID: id, Method: method, Params: payload})
	if err != nil {
		finish(len(payload))
		return nil, err
	}
	frame, err := wire.EncodeFrame(0, wire.TypeRequest, requestPayload)
	if err != nil {
		finish(len(payload))
		return nil, err
	}

	response := make(chan result, 1)
	c.mu.Lock()
	c.waiters[id] = &responseWaiter{ch: response}
	c.mu.Unlock()
	abandonResponse := false
	defer func() {
		c.removeWaiter(id, abandonResponse)
	}()
	if err := c.send(frame); err != nil {
		finish(len(frame))
		return nil, err
	}
	select {
	case <-ctx.Done():
		if waitTerminalOnCancel {
			timer := time.NewTimer(5 * time.Second)
			defer timer.Stop()
			select {
			case result := <-response:
				if result.err != nil {
					finish(len(frame))
					return nil, result.err
				}
				finish(len(result.payload))
				return result.payload, nil
			case <-c.done:
				finish(len(frame))
				return nil, c.Err()
			case <-timer.C:
				_ = c.Close()
				finish(len(frame))
				return nil, fmt.Errorf("cancelled file resource open did not reach a terminal response")
			}
		}
		_ = c.sendRequestCancel(id)
		abandonResponse = true
		finish(len(frame))
		return nil, ctx.Err()
	case result := <-response:
		if result.err != nil {
			finish(len(frame))
			return nil, result.err
		}
		finish(len(result.payload))
		return result.payload, nil
	}
}

func (c *Client) sendRequestCancel(id uint64) error {
	payload, err := EncodeRequestCancelPayload(id)
	if err != nil {
		return err
	}
	frame, err := wire.EncodeFrame(0, wire.TypeRequestCancel, payload)
	if err != nil {
		return err
	}
	return c.send(frame)
}

func (c *Client) send(frame []byte) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	return c.transport.Send(frame)
}

func (limits clientPendingLimits) normalized() clientPendingLimits {
	defaults := defaultClientPendingLimits
	if limits.channels <= 0 {
		limits.channels = defaults.channels
	}
	if limits.frames <= 0 {
		limits.frames = defaults.frames
	}
	if limits.framesPerChannel <= 0 {
		limits.framesPerChannel = defaults.framesPerChannel
	}
	if limits.bytes <= 0 {
		limits.bytes = defaults.bytes
	}
	return limits
}

func (c *Client) handleControlFrame(typ uint8, payload []byte) error {
	if typ != wire.TypeHello && typ != wire.TypeError && !c.helloReceived {
		return &PeerError{Code: ProtocolErrorCodeBadRequest, Message: "control frame received before Hello"}
	}
	switch typ {
	case wire.TypeHello:
		if c.helloReceived {
			return &PeerError{Code: ProtocolErrorCodeBadRequest, Message: "duplicate protocol Hello"}
		}
		hello, err := DecodeHelloPayload(payload)
		if err != nil {
			return &PeerError{Code: ProtocolErrorCodeBadRequest, Message: "malformed protocol Hello"}
		}
		if hello.Version != wire.Version {
			return &PeerError{Code: ProtocolErrorCodeBadRequest, Message: fmt.Sprintf("unsupported wire version %d", hello.Version)}
		}
		c.helloReceived = true
		select {
		case c.helloCh <- result{}:
		default:
			return &PeerError{Code: ProtocolErrorCodeBadRequest, Message: "duplicate protocol Hello result"}
		}
		return nil
	case wire.TypeEvent:
		var applicationEvent apipb.EventEnvelope
		if err := proto.Unmarshal(payload, &applicationEvent); err != nil {
			return &PeerError{Code: ProtocolErrorCodeBadRequest, Message: "malformed application event"}
		}
		if applicationEvent.GetEvent() == nil {
			return &PeerError{Code: ProtocolErrorCodeBadRequest, Message: "application event payload has no event"}
		}
		return c.publishApplicationEvent(&applicationEvent)
	case wire.TypeResponse:
		response, err := DecodeResponsePayload(payload)
		if err != nil {
			return &PeerError{Code: ProtocolErrorCodeBadRequest, Message: "malformed protocol response"}
		}
		return c.deliverResponse(response.ID, result{payload: response.Result})
	case wire.TypeResponseBinary:
		id, responsePayload, err := DecodeBinaryResponsePayload(payload)
		if err != nil {
			return &PeerError{Code: ProtocolErrorCodeBadRequest, Message: "malformed binary protocol response"}
		}
		return c.deliverResponse(id, result{payload: responsePayload})
	case wire.TypeError:
		message, err := DecodeErrorPayload(payload)
		if err != nil {
			return &PeerError{Code: ProtocolErrorCodeBadRequest, Message: "malformed protocol error"}
		}
		requestErr := &RequestError{Code: message.Error.Code, Message: message.Error.Message}
		if message.ID == 0 {
			return requestErr
		}
		return c.deliverResponse(message.ID, result{err: requestErr})
	default:
		return &PeerError{Code: ProtocolErrorCodeBadRequest, Message: fmt.Sprintf("unsupported control frame type %d", typ)}
	}
}

func (c *Client) deliverResponse(id uint64, response result) error {
	if id == 0 {
		return &PeerError{Code: ProtocolErrorCodeBadRequest, Message: "protocol response ID is required"}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, abandoned := c.abandonedWaiters[id]; abandoned {
		delete(c.abandonedWaiters, id)
		return nil
	}
	waiter := c.waiters[id]
	if waiter == nil {
		return &PeerError{Code: ProtocolErrorCodeBadRequest, Message: fmt.Sprintf("unsolicited protocol response %d", id)}
	}
	if waiter.delivered {
		return &PeerError{Code: ProtocolErrorCodeBadRequest, Message: fmt.Sprintf("duplicate protocol response %d", id)}
	}
	waiter.delivered = true
	waiter.ch <- response
	return nil
}

func (c *Client) removeWaiter(id uint64, abandoned bool) {
	c.mu.Lock()
	waiter := c.waiters[id]
	delete(c.waiters, id)
	overflow := false
	if abandoned && waiter != nil && !waiter.delivered {
		if len(c.abandonedWaiters) >= maxAbandonedWaiters {
			overflow = true
		} else {
			c.abandonedWaiters[id] = struct{}{}
		}
	}
	c.mu.Unlock()
	if overflow {
		c.failPeer(&PeerError{Code: ProtocolErrorCodeResourceExhausted, Message: "abandoned response capacity is exhausted"})
	}
}

func validInboundStreamFrameType(typ uint8) bool {
	switch typ {
	case wire.TypeStreamReady, wire.TypeSyncLost, wire.TypeClosed, wire.TypePTYOutput,
		wire.TypeFileData, wire.TypeFileAck, wire.TypeFileFinish, wire.TypeFileResult, wire.TypeBrowserData, wire.TypeBrowserClosed, wire.TypeError:
		return true
	default:
		return false
	}
}

func (c *Client) queuePendingFrameLocked(queues map[uint16][]StreamFrame, channel uint16, typ uint8, payload []byte) error {
	queue, exists := queues[channel]
	if !exists && len(c.pending)+len(c.reused) >= c.pendingLimits.channels {
		return &PeerError{Code: ProtocolErrorCodeResourceExhausted, Message: "pending stream channel capacity is exhausted"}
	}
	if len(queue) >= c.pendingLimits.framesPerChannel || c.pendingFrameCount >= c.pendingLimits.frames {
		return &PeerError{Code: ProtocolErrorCodeResourceExhausted, Message: "pending stream frame capacity is exhausted"}
	}
	if len(payload) > c.pendingLimits.bytes-c.pendingByteCount {
		return &PeerError{Code: ProtocolErrorCodeResourceExhausted, Message: "pending stream byte capacity is exhausted"}
	}
	framePayload := append([]byte(nil), payload...)
	queues[channel] = append(queue, StreamFrame{Type: typ, Payload: framePayload})
	c.pendingFrameCount++
	c.pendingByteCount += len(framePayload)
	return nil
}

func (c *Client) takePendingFramesLocked(queues map[uint16][]StreamFrame, channel uint16) []StreamFrame {
	frames := queues[channel]
	delete(queues, channel)
	for _, frame := range frames {
		c.pendingFrameCount--
		c.pendingByteCount -= len(frame.Payload)
	}
	return frames
}

func (c *Client) discardPendingFramesLocked(queues map[uint16][]StreamFrame, channel uint16) {
	_ = c.takePendingFramesLocked(queues, channel)
}

func (c *Client) failPeer(err error) {
	if err == nil {
		return
	}
	c.setDoneErr(err)
	c.failAll(err)
	_ = c.transport.Close()
}

func (c *Client) readLoop() {
	defer close(c.done)
	for {
		frame, err := c.transport.Recv()
		if err != nil {
			c.setDoneErr(err)
			c.failAll(err)
			return
		}
		channel, typ, payload, err := wire.DecodeFrame(frame)
		if err != nil {
			c.failPeer(&PeerError{Code: ProtocolErrorCodeBadRequest, Message: err.Error()})
			return
		}
		if channel == 0 {
			if err := c.handleControlFrame(typ, payload); err != nil {
				c.failPeer(err)
				return
			}
			continue
		}

		c.mu.Lock()
		stream := c.streams[channel]
		if !validInboundStreamFrameType(typ) {
			c.mu.Unlock()
			if stream == nil {
				c.failPeer(&PeerError{Code: ProtocolErrorCodeBadRequest, Message: fmt.Sprintf("unsupported stream frame type %d", typ)})
				return
			}
			stream.fail(ProtocolErrorCodeBadRequest, fmt.Sprintf("unsupported stream frame type %d", typ))
			continue
		}
		if typ == wire.TypeError {
			if _, err := DecodeErrorPayload(payload); err != nil {
				c.mu.Unlock()
				if stream == nil {
					c.failPeer(&PeerError{Code: ProtocolErrorCodeBadRequest, Message: "malformed error stream frame"})
					return
				}
				stream.fail(ProtocolErrorCodeBadRequest, "malformed error stream frame")
				continue
			}
		}
		if stream == nil {
			if _, dropped := c.dropped[channel]; dropped {
				if resource := c.applicationAttachments[channel]; resource != nil &&
					resource.GetKind() == apipb.ResourceKind_RESOURCE_KIND_BROWSER_PROXY {
					// Browser resources are never reopened. A response that races
					// with local socket teardown must not consume the reusable
					// attachment queue or eventually tear down the whole peer.
					c.mu.Unlock()
					continue
				}
				err = c.queuePendingFrameLocked(c.reused, channel, typ, payload)
			} else {
				err = c.queuePendingFrameLocked(c.pending, channel, typ, payload)
			}
			c.mu.Unlock()
			if err != nil {
				c.failPeer(err)
				return
			}
			continue
		}
		c.mu.Unlock()
		stream.send(StreamFrame{Type: typ, Payload: payload})
	}
}

func (c *Client) setDoneErr(err error) {
	if err == nil {
		return
	}
	c.doneErrMu.Lock()
	defer c.doneErrMu.Unlock()
	if c.doneErr == nil {
		c.doneErr = err
	}
}

func (c *Client) publishApplicationEvent(event *apipb.EventEnvelope) error {
	if event == nil {
		return nil
	}
	snapshot := proto.Clone(event).(*apipb.EventEnvelope)
	c.mu.Lock()
	if len(c.applicationEventSubscribers) == 0 {
		if len(c.pendingApplicationEvents) >= 64 {
			c.mu.Unlock()
			return &PeerError{Code: ProtocolErrorCodeResourceExhausted, Message: "pending application event capacity is exhausted"}
		}
		c.pendingApplicationEvents = append(c.pendingApplicationEvents, snapshot)
		c.mu.Unlock()
		return nil
	}
	subscribers := make([]chan *apipb.EventEnvelope, 0, len(c.applicationEventSubscribers))
	for _, subscriber := range c.applicationEventSubscribers {
		subscribers = append(subscribers, subscriber)
	}
	c.mu.Unlock()
	for _, subscriber := range subscribers {
		select {
		case subscriber <- proto.Clone(snapshot).(*apipb.EventEnvelope):
		default:
			return &PeerError{Code: ProtocolErrorCodeResourceExhausted, Message: "application event subscriber capacity is exhausted"}
		}
	}
	return nil
}

func (c *Client) doneClosed() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}

func (c *Client) failAll(err error) {
	if err == nil {
		err = io.EOF
	}
	c.mu.Lock()
	waiters := make([]chan result, 0, len(c.waiters))
	for id, waiter := range c.waiters {
		waiters = append(waiters, waiter.ch)
		delete(c.waiters, id)
	}
	for id := range c.abandonedWaiters {
		delete(c.abandonedWaiters, id)
	}
	streams := make([]*clientStream, 0, len(c.streams))
	for id, stream := range c.streams {
		streams = append(streams, stream)
		delete(c.streams, id)
	}
	for id := range c.pending {
		delete(c.pending, id)
	}
	for id := range c.reused {
		delete(c.reused, id)
	}
	for id := range c.dropped {
		delete(c.dropped, id)
	}
	for id := range c.applicationEventSubscribers {
		delete(c.applicationEventSubscribers, id)
	}
	c.pendingApplicationEvents = nil
	c.pendingFrameCount = 0
	c.pendingByteCount = 0
	c.mu.Unlock()

	// waiter 可能已经收到 terminal result 并正等待 c.mu 删除自身；通知与 stream close 必须在 registry 锁外执行。
	for _, ch := range waiters {
		select {
		case ch <- result{err: err}:
		default:
		}
	}
	select {
	case c.helloCh <- result{err: err}:
	default:
	}
	for _, stream := range streams {
		stream.close()
	}
}

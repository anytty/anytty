package provider

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/anytty/anytty/internal/providerproto"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
	"github.com/anytty/anytty/shared/transport"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
)

// Client 是 provider 协议的同步客户端（access 侧与测试使用）。
// 每个请求分配一个 id；读循环按 id 分发响应。
type Client struct {
	conn   transport.Transport
	nextID atomic.Uint64

	sendMu  sync.Mutex
	mu      sync.Mutex
	waiters map[uint64]chan *providerv1.Response
	streams map[uint16]chan StreamFrame
	pending map[uint16][]StreamFrame

	eventMu          sync.Mutex
	nextEventSub     uint64
	eventSubscribers map[uint64]chan *providerv1.TerminalEvent
	closed           bool
	doneErr          error
	done             chan struct{}
	hello            chan struct{}
}

// StreamFrame 是 attachment channel 上的一个 provider stream frame。
type StreamFrame struct {
	Type    uint8
	Payload []byte
}

// Dial 连接 provider socket 并完成 Hello。
func Dial(ctx context.Context, socket string) (*Client, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	connection, err := unixtransport.DialContext(ctx, socket)
	if err != nil {
		return nil, err
	}
	client := NewClient(connection)
	if err := client.Hello(ctx); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

// NewClient 在已有 transport 上创建 provider 客户端；调用方必须自行 Hello。
func NewClient(connection transport.Transport) *Client {
	client := &Client{
		conn:             connection,
		waiters:          make(map[uint64]chan *providerv1.Response),
		streams:          make(map[uint16]chan StreamFrame),
		pending:          make(map[uint16][]StreamFrame),
		eventSubscribers: make(map[uint64]chan *providerv1.TerminalEvent),
		done:             make(chan struct{}),
		hello:            make(chan struct{}),
	}
	go client.readLoop()
	return client
}

// Hello 完成协议握手。
func (client *Client) Hello(ctx context.Context) error {
	payload, err := providerproto.EncodeHelloPayload(&providerv1.Hello{Version: providerproto.Version, Client: "anytty-access"})
	if err != nil {
		return err
	}
	if err := client.sendFrame(0, providerproto.TypeHello, payload); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-client.done:
		return client.Err()
	case <-client.hello:
		return nil
	}
}

// Err 返回客户端终止原因。
func (client *Client) Err() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.doneErr == nil {
		return io.EOF
	}
	return client.doneErr
}

// Done 在连接终止时关闭。
func (client *Client) Done() <-chan struct{} { return client.done }

// Close 释放连接；幂等。
func (client *Client) Close() error {
	return client.conn.Close()
}

func (client *Client) request(ctx context.Context, build func(id uint64) *providerv1.Request) (*providerv1.Response, error) {
	id := client.nextID.Add(1)
	request := build(id)
	payload, err := providerproto.EncodeRequestPayload(request)
	if err != nil {
		return nil, err
	}
	waiter := make(chan *providerv1.Response, 1)
	client.mu.Lock()
	if client.doneErr != nil {
		err := client.doneErr
		client.mu.Unlock()
		return nil, err
	}
	client.waiters[id] = waiter
	client.mu.Unlock()
	defer func() {
		client.mu.Lock()
		delete(client.waiters, id)
		client.mu.Unlock()
	}()
	if err := client.sendFrame(0, providerproto.TypeRequest, payload); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-client.done:
		return nil, client.Err()
	case response := <-waiter:
		if response.GetError() != nil {
			return nil, &ProviderError{Code: response.GetError().GetCode(), Message: response.GetError().GetMessage()}
		}
		return response, nil
	}
}

func (client *Client) sendFrame(channel uint16, typ uint8, payload []byte) error {
	frame, err := providerproto.EncodeFrame(channel, typ, payload)
	if err != nil {
		return err
	}
	client.sendMu.Lock()
	defer client.sendMu.Unlock()
	return client.conn.Send(frame)
}

func (client *Client) readLoop() {
	defer close(client.done)
	for {
		frame, err := client.conn.Recv()
		if err != nil {
			client.fail(err)
			return
		}
		channel, typ, payload, err := providerproto.DecodeFrame(frame)
		if err != nil {
			client.fail(err)
			return
		}
		if channel != 0 {
			client.deliverStream(channel, StreamFrame{Type: typ, Payload: payload})
			continue
		}
		switch typ {
		case providerproto.TypeResponse:
			response, err := providerproto.DecodeResponsePayload(payload)
			if err != nil {
				client.fail(err)
				return
			}
			client.mu.Lock()
			waiter := client.waiters[response.GetId()]
			client.mu.Unlock()
			if waiter != nil {
				waiter <- response
			}
		case providerproto.TypeEvent:
			event, err := providerproto.DecodeEventPayload(payload)
			if err != nil {
				client.fail(err)
				return
			}
			client.publishEvent(event.GetTerminalEvent())
		case providerproto.TypeHello:
			client.mu.Lock()
			select {
			case <-client.hello:
			default:
				close(client.hello)
			}
			client.mu.Unlock()
		default:
			client.fail(fmt.Errorf("provider client: unsupported control frame type %d", typ))
			return
		}
	}
}

func (client *Client) deliverStream(channel uint16, frame StreamFrame) {
	client.mu.Lock()
	stream := client.streams[channel]
	if stream == nil {
		if len(client.pending[channel]) >= 256 {
			client.mu.Unlock()
			client.fail(fmt.Errorf("provider client: pending stream frame capacity is exhausted"))
			return
		}
		client.pending[channel] = append(client.pending[channel], frame)
		client.mu.Unlock()
		return
	}
	client.mu.Unlock()
	select {
	case stream <- frame:
	default:
		client.fail(fmt.Errorf("provider client: stream consumer fell behind"))
	}
}

// Stream 注册一个 attachment channel 的帧消费者；返回 stop 解除注册。
// bootstrap/ready 之前的帧会被暂存，保证调用方可以先注册再发送 BootstrapDone。
func (client *Client) Stream(channel uint16) (<-chan StreamFrame, func()) {
	stream := make(chan StreamFrame, 64)
	client.mu.Lock()
	pending := client.pending[channel]
	delete(client.pending, channel)
	client.streams[channel] = stream
	client.mu.Unlock()
	go func() {
		for _, frame := range pending {
			stream <- frame
		}
	}()
	var once sync.Once
	return stream, func() {
		once.Do(func() {
			client.mu.Lock()
			if client.streams[channel] == stream {
				delete(client.streams, channel)
			}
			client.mu.Unlock()
		})
	}
}

// SendBootstrapDone 在 attachment channel 上启动 PTY 输出。
func (client *Client) SendBootstrapDone(channel uint16) error {
	return client.sendFrame(channel, providerproto.StreamBootstrapDone, nil)
}

// SendStreamClose 停止 attachment channel 的 PTY 输出，但保留 resource。
func (client *Client) SendStreamClose(channel uint16) error {
	return client.sendFrame(channel, providerproto.StreamClosed, nil)
}

// Attach 建立 attachment；调用方用 token 前两字节得到 channel 后 Stream()。
func (client *Client) Attach(ctx context.Context, command *providerv1.TerminalAttachCommand) (*providerv1.TerminalAttachResult, error) {
	response, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_TerminalAttach{TerminalAttach: command}}
	})
	if err != nil {
		return nil, err
	}
	return response.GetTerminalAttach(), nil
}

// Detach 释放 attachment resource。
func (client *Client) Detach(ctx context.Context, token []byte) error {
	_, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_TerminalDetach{TerminalDetach: &providerv1.TerminalDetachCommand{OpaqueToken: token}}}
	})
	return err
}

// Input 向 attachment 写入 bytes。
func (client *Client) Input(ctx context.Context, token, data []byte) error {
	_, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_TerminalInput{TerminalInput: &providerv1.TerminalInputCommand{OpaqueToken: token, Data: data}}}
	})
	return err
}

// Resize 协调 resize ownership 并返回 authoritative control。
func (client *Client) Resize(ctx context.Context, token []byte, size *providerv1.Size, policy providerv1.ResizePolicy, takeOwnership bool, expectedOwnerEpoch uint64) (*providerv1.TerminalResizeResult, error) {
	response, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_TerminalResize{TerminalResize: &providerv1.TerminalResizeCommand{
			OpaqueToken: token, Size: size, ResizePolicy: policy, TakeOwnership: takeOwnership, ExpectedOwnerEpoch: expectedOwnerEpoch,
		}}}
	})
	if err != nil {
		return nil, err
	}
	return response.GetTerminalResize(), nil
}

// ResizeLock 修改 owner size lock。
func (client *Client) ResizeLock(ctx context.Context, token []byte, locked bool) (*providerv1.TerminalResizeResult, error) {
	response, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_TerminalResizeLock{TerminalResizeLock: &providerv1.TerminalResizeLockCommand{
			OpaqueToken: token, Locked: locked,
		}}}
	})
	if err != nil {
		return nil, err
	}
	return response.GetTerminalResizeLock(), nil
}

func (client *Client) publishEvent(event *providerv1.TerminalEvent) {
	if event == nil {
		return
	}
	client.eventMu.Lock()
	subscribers := make([]chan *providerv1.TerminalEvent, 0, len(client.eventSubscribers))
	for _, subscriber := range client.eventSubscribers {
		subscribers = append(subscribers, subscriber)
	}
	client.eventMu.Unlock()
	for _, subscriber := range subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

// Events 注册 provider terminal event 订阅；ctx 取消时退订。
func (client *Client) Events(ctx context.Context) <-chan *providerv1.TerminalEvent {
	if ctx == nil {
		ctx = context.Background()
	}
	out := make(chan *providerv1.TerminalEvent, 64)
	client.eventMu.Lock()
	client.nextEventSub++
	id := client.nextEventSub
	client.eventSubscribers[id] = out
	client.eventMu.Unlock()
	go func() {
		<-ctx.Done()
		client.eventMu.Lock()
		delete(client.eventSubscribers, id)
		client.eventMu.Unlock()
	}()
	return out
}

// HistoryWindow 查询 authoritative history window。
func (client *Client) HistoryWindow(ctx context.Context, command *providerv1.HistoryWindowCommand) (*providerv1.HistoryWindowResult, error) {
	response, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_HistoryWindow{HistoryWindow: command}}
	})
	if err != nil {
		return nil, err
	}
	return response.GetHistoryWindow(), nil
}

// HistoryCopy 复制 frozen history token 的文本。
func (client *Client) HistoryCopy(ctx context.Context, command *providerv1.HistoryCopyCommand) (*providerv1.HistoryCopyResult, error) {
	response, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_HistoryCopy{HistoryCopy: command}}
	})
	if err != nil {
		return nil, err
	}
	return response.GetHistoryCopy(), nil
}

// HistorySearch 搜索 frozen history token。
func (client *Client) HistorySearch(ctx context.Context, command *providerv1.HistorySearchCommand) (*providerv1.HistorySearchResult, error) {
	response, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_HistorySearch{HistorySearch: command}}
	})
	if err != nil {
		return nil, err
	}
	return response.GetHistorySearch(), nil
}

// HistoryRelease 释放 frozen history token。
func (client *Client) HistoryRelease(ctx context.Context, terminalID string, token string) error {
	_, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_HistoryRelease{HistoryRelease: &providerv1.HistoryReleaseCommand{
			Terminal: &providerv1.TerminalRef{TerminalId: terminalID}, Token: token,
		}}}
	})
	return err
}

// HistoryBacklogStatus 返回 history backlog 诊断投影。
func (client *Client) HistoryBacklogStatus(ctx context.Context, terminalID string) (*providerv1.HistoryBacklogStatusResult, error) {
	response, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_HistoryBacklogStatus{HistoryBacklogStatus: &providerv1.HistoryBacklogStatusCommand{
			Terminal: &providerv1.TerminalRef{TerminalId: terminalID},
		}}}
	})
	if err != nil {
		return nil, err
	}
	return response.GetHistoryBacklogStatus(), nil
}

// LiveScreenNext 等待 observed revision 之后的 native screen delta。
func (client *Client) LiveScreenNext(ctx context.Context, terminalID string, observedRevision uint64) (*providerv1.NativeScreenResult, error) {
	response, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_LiveScreenNext{LiveScreenNext: &providerv1.LiveScreenNextCommand{
			Terminal: &providerv1.TerminalRef{TerminalId: terminalID}, ObservedRevision: observedRevision,
		}}}
	})
	if err != nil {
		return nil, err
	}
	return response.GetLiveScreen(), nil
}

// EventSubscribe 建立 provider terminal event 订阅。
func (client *Client) EventSubscribe(ctx context.Context, command *providerv1.EventSubscribeCommand) (*providerv1.EventSubscriptionResult, error) {
	response, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_EventSubscribe{EventSubscribe: command}}
	})
	if err != nil {
		return nil, err
	}
	return response.GetEventSubscribe(), nil
}

// EventRelease 释放 provider terminal event 订阅。
func (client *Client) EventRelease(ctx context.Context, token []byte) error {
	_, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_EventRelease{EventRelease: &providerv1.EventSubscriptionReleaseCommand{
			OpaqueToken: token,
		}}}
	})
	return err
}

func (client *Client) fail(err error) {
	client.mu.Lock()
	if client.doneErr == nil {
		client.doneErr = err
	}
	waiters := make([]chan *providerv1.Response, 0, len(client.waiters))
	for id, waiter := range client.waiters {
		waiters = append(waiters, waiter)
		delete(client.waiters, id)
	}
	client.mu.Unlock()
	for _, waiter := range waiters {
		close(waiter)
	}
}

// Create 创建 terminal。
func (client *Client) Create(ctx context.Context, spec *providerv1.TerminalCreateSpec) (*providerv1.TerminalInfo, error) {
	response, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_TerminalCreate{TerminalCreate: spec}}
	})
	if err != nil {
		return nil, err
	}
	return response.GetTerminalCreate(), nil
}

// List 返回 terminal inventory。
func (client *Client) List(ctx context.Context) ([]*providerv1.TerminalInfo, error) {
	response, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_TerminalList{TerminalList: &providerv1.TerminalListCommand{}}}
	})
	if err != nil {
		return nil, err
	}
	return response.GetTerminalList().GetTerminals(), nil
}

// Get 返回单个 terminal 快照。
func (client *Client) Get(ctx context.Context, terminalID string) (*providerv1.TerminalInfo, error) {
	response, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_TerminalGet{TerminalGet: &providerv1.TerminalRef{TerminalId: terminalID}}}
	})
	if err != nil {
		return nil, err
	}
	return response.GetTerminalGet().GetTerminal(), nil
}

// Restart 重启 terminal 进程。
func (client *Client) Restart(ctx context.Context, terminalID string) error {
	_, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_TerminalRestart{TerminalRestart: &providerv1.TerminalRef{TerminalId: terminalID}}}
	})
	return err
}

// Kill 终止 terminal 进程。
func (client *Client) Kill(ctx context.Context, terminalID string) error {
	_, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_TerminalKill{TerminalKill: &providerv1.TerminalRef{TerminalId: terminalID}}}
	})
	return err
}

// Remove 删除 terminal record。
func (client *Client) Remove(ctx context.Context, terminalID string) error {
	_, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_TerminalRemove{TerminalRemove: &providerv1.TerminalRef{TerminalId: terminalID}}}
	})
	return err
}

// SetMetadata 原子更新 name/tags。
func (client *Client) SetMetadata(ctx context.Context, terminalID, name string, tags map[string]string) error {
	_, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_TerminalSetMetadata{TerminalSetMetadata: &providerv1.TerminalSetMetadataCommand{
			Terminal: &providerv1.TerminalRef{TerminalId: terminalID}, Name: name, Tags: tags,
		}}}
	})
	return err
}

// SetTags 替换 tags。
func (client *Client) SetTags(ctx context.Context, terminalID string, tags map[string]string) error {
	_, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_TerminalSetTags{TerminalSetTags: &providerv1.TerminalSetTagsCommand{
			Terminal: &providerv1.TerminalRef{TerminalId: terminalID}, Tags: tags,
		}}}
	})
	return err
}

// TerminalDefaults 返回 owning pool 机器的 shell/cwd 默认值。
func (client *Client) TerminalDefaults(ctx context.Context) (*providerv1.TerminalDefaults, error) {
	response, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_TerminalDefaults{TerminalDefaults: &providerv1.TerminalDefaultsCommand{}}}
	})
	if err != nil {
		return nil, err
	}
	return response.GetTerminalDefaults(), nil
}

// ListDirectories 返回 path completion 窗口。
func (client *Client) ListDirectories(ctx context.Context, prefix string, limit int) (*providerv1.PathListDirectoriesResult, error) {
	response, err := client.request(ctx, func(id uint64) *providerv1.Request {
		return &providerv1.Request{Id: id, Command: &providerv1.Request_PathListDirectories{PathListDirectories: &providerv1.PathListDirectoriesCommand{
			Prefix: prefix, Limit: int32(limit),
		}}}
	})
	if err != nil {
		return nil, err
	}
	return response.GetPathListDirectories(), nil
}

package pool

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	"github.com/anytty/anytty/internal/providerproto"
	poolprovider "github.com/anytty/anytty/pool/provider"
	"github.com/anytty/anytty/proto/access/wire"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
)

// Info 返回 pool provider 身份。EndpointID 取本地 provider socket 路径，仅诊断用；
// access-visible endpoint 身份仍由 access/server 的 session stamp 决定。
func (provider *TerminalProvider) Info(context.Context) (terminalprovider.Info, error) {
	if provider == nil {
		return terminalprovider.Info{}, terminalprovider.ErrUnavailable
	}
	return terminalprovider.Info{
		Kind:       "pool",
		Version:    fmt.Sprintf("provider/v%d", providerproto.Version),
		EndpointID: provider.socket,
	}, nil
}

// Capabilities 声明 pool provider 支持全部能力面。
func (*TerminalProvider) Capabilities() terminalprovider.Capabilities {
	return terminalprovider.Capabilities{
		Lifecycle: true, Metadata: true, Attach: true, ResizeLock: true,
		History: true, Live: true, Events: true, Paths: true,
	}
}

// readyClient 返回可用的 provider client；provider 已关闭时返回 typed unavailable。
func (provider *TerminalProvider) readyClient() (*poolprovider.Client, error) {
	if provider == nil || provider.client == nil {
		return nil, fmt.Errorf("%w: provider is closed", terminalprovider.ErrUnavailable)
	}
	return provider.client, nil
}

// typedProviderError 把 provider wire 错误投影为 terminal 领域 typed error；
// 同时保留原始 ProviderError（双 %w），调用方可按 sentinel 分类或按 wire code
// 细分；其余错误（bad request/forbidden/stale resource/exhausted/internal）原样
// 透传，保留原始 code。
func typedProviderError(err error) error {
	if err == nil {
		return nil
	}
	var providerErr *poolprovider.ProviderError
	if !errors.As(err, &providerErr) {
		return err
	}
	// 所有 provider wire 错误都带上 typed code，access/server 据此复刻既有
	// ApiErrorCode 映射（不 import pool/provider 实现包）。
	coded := &terminalprovider.CodedError{
		Code:      providerErr.Code,
		Message:   providerErr.Message,
		Retryable: providerErr.Code == providerproto.ErrorExhausted || providerErr.Code == providerproto.ErrorUnavailable,
	}
	switch providerErr.Code {
	case providerproto.ErrorNotFound:
		return fmt.Errorf("%w: %w", terminalprovider.ErrNotFound, coded)
	case providerproto.ErrorConflict:
		return fmt.Errorf("%w: %w", terminalprovider.ErrConflict, coded)
	case providerproto.ErrorUnavailable:
		return fmt.Errorf("%w: %w", terminalprovider.ErrUnavailable, coded)
	default:
		return fmt.Errorf("%w: %w", terminalprovider.ErrUnavailable, coded)
	}
}

// ---- lifecycle ---------------------------------------------------------------

// Create 创建 terminal 并返回 authoritative 快照。
func (provider *TerminalProvider) Create(ctx context.Context, spec *providerv1.TerminalCreateSpec) (*providerv1.TerminalInfo, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	info, err := client.Create(ctx, spec)
	if err != nil {
		return nil, typedProviderError(err)
	}
	return info, nil
}

// List 返回 terminal inventory。
func (provider *TerminalProvider) List(ctx context.Context) ([]*providerv1.TerminalInfo, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	items, err := client.List(ctx)
	if err != nil {
		return nil, typedProviderError(err)
	}
	return items, nil
}

// Get 返回单个 terminal 快照。
func (provider *TerminalProvider) Get(ctx context.Context, terminalID string) (*providerv1.TerminalInfo, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	info, err := client.Get(ctx, terminalID)
	if err != nil {
		return nil, typedProviderError(err)
	}
	return info, nil
}

// Restart 重启 terminal 进程；provider wire 只回 ack，adapter 重新读取快照。
func (provider *TerminalProvider) Restart(ctx context.Context, terminalID string) (*providerv1.TerminalInfo, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	if err := client.Restart(ctx, terminalID); err != nil {
		return nil, typedProviderError(err)
	}
	info, err := client.Get(ctx, terminalID)
	if err != nil {
		return nil, typedProviderError(err)
	}
	return info, nil
}

// Kill 终止 terminal 进程（保留 record）。
func (provider *TerminalProvider) Kill(ctx context.Context, terminalID string) error {
	client, err := provider.readyClient()
	if err != nil {
		return err
	}
	return typedProviderError(client.Kill(ctx, terminalID))
}

// Remove 删除 terminal record。
func (provider *TerminalProvider) Remove(ctx context.Context, terminalID string) error {
	client, err := provider.readyClient()
	if err != nil {
		return err
	}
	return typedProviderError(client.Remove(ctx, terminalID))
}

// SetMetadata 按 patch 更新 name/tags。provider wire 的 SetMetadata 是整体更新，
// 这里先读取当前快照补齐 nil 字段，提交后再读取 authoritative 结果。
func (provider *TerminalProvider) SetMetadata(ctx context.Context, terminalID string, patch terminalprovider.MetadataPatch) (*providerv1.TerminalInfo, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	current, err := client.Get(ctx, terminalID)
	if err != nil {
		return nil, typedProviderError(err)
	}
	name := current.GetName()
	if patch.Name != nil {
		name = *patch.Name
	}
	tags := current.GetTags()
	if patch.Tags != nil {
		tags = *patch.Tags
	}
	if err := client.SetMetadata(ctx, terminalID, name, tags); err != nil {
		return nil, typedProviderError(err)
	}
	info, err := client.Get(ctx, terminalID)
	if err != nil {
		return nil, typedProviderError(err)
	}
	return info, nil
}

// SetTags 按 patch 增量更新 tags。provider wire 的 SetTags 是整体替换，
// 这里以当前快照为基底应用 Remove/Set，提交后读取 authoritative 结果。
func (provider *TerminalProvider) SetTags(ctx context.Context, terminalID string, patch terminalprovider.TagsPatch) (*providerv1.TerminalInfo, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	current, err := client.Get(ctx, terminalID)
	if err != nil {
		return nil, typedProviderError(err)
	}
	tags := make(map[string]string, len(current.GetTags())+len(patch.Set))
	if !patch.Replace {
		for key, value := range current.GetTags() {
			tags[key] = value
		}
	}
	for _, key := range patch.Remove {
		delete(tags, key)
	}
	for key, value := range patch.Set {
		tags[key] = value
	}
	if err := client.SetTags(ctx, terminalID, tags); err != nil {
		return nil, typedProviderError(err)
	}
	info, err := client.Get(ctx, terminalID)
	if err != nil {
		return nil, typedProviderError(err)
	}
	return info, nil
}

// ---- attachments -------------------------------------------------------------

// Attach 建立 attachment。typed Attachment 的 Stream 在 WaitReady 或首次读写时
// 完成 bootstrap/ready，避免 consumer 未就绪时 PTY 输出淹没 stream 队列。
func (provider *TerminalProvider) Attach(ctx context.Context, request terminalprovider.AttachRequest) (terminalprovider.Attachment, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	if request.TerminalID == "" {
		return nil, errors.New("provider/pool: attach terminal id is required")
	}
	result, err := client.Attach(ctx, &providerv1.TerminalAttachCommand{
		Terminal:     &providerv1.TerminalRef{TerminalId: request.TerminalID},
		Mode:         request.Mode,
		ResizePolicy: request.ResizePolicy,
		SurfaceId:    request.SurfaceID,
		ViewId:       request.ViewID,
	})
	if err != nil {
		return nil, typedProviderError(err)
	}
	handle := result.GetAttachment()
	if handle == nil || len(handle.GetOpaqueToken()) < 2 {
		return nil, errors.New("provider/pool: attachment handle from provider is malformed")
	}
	return &poolAttachment{
		client:  client,
		handle:  handle,
		size:    handle.GetSize(),
		epoch:   handle.GetEpoch(),
		control: result.GetResizeControl(),
	}, nil
}

// poolAttachment 是 typed Attachment 的 pool 实现。
type poolAttachment struct {
	client *poolprovider.Client
	handle *providerv1.AttachmentHandle

	mu      sync.Mutex
	size    *providerv1.Size
	epoch   uint64
	control *providerv1.ResizeControl
	duplex  *attachmentDuplex
}

// Handle 返回 attach 时的 attachment handle。
func (attachment *poolAttachment) Handle() *providerv1.AttachmentHandle {
	return attachment.handle
}

// Size 返回最近一次 authoritative size。
func (attachment *poolAttachment) Size() *providerv1.Size {
	attachment.mu.Lock()
	defer attachment.mu.Unlock()
	return attachment.size
}

// OwnerEpoch 返回最近一次已知的 owner epoch（resize fence 用）。
func (attachment *poolAttachment) OwnerEpoch() uint64 {
	attachment.mu.Lock()
	defer attachment.mu.Unlock()
	return attachment.epoch
}

// ResizeControl 返回最近一次 authoritative resize control（初始为 attach 时值）。
func (attachment *poolAttachment) ResizeControl() *providerv1.ResizeControl {
	attachment.mu.Lock()
	defer attachment.mu.Unlock()
	return attachment.control
}

// Stream 返回 PTY 双向字节流；同一 attachment 复用一个未关闭的 Duplex，
// 客户端 stop 之后重新 bootstrap 会得到新的 Duplex。
func (attachment *poolAttachment) Stream() terminalprovider.Duplex {
	attachment.mu.Lock()
	defer attachment.mu.Unlock()
	if attachment.duplex == nil || attachment.duplex.closed.Load() {
		attachment.duplex = &attachmentDuplex{
			client: attachment.client,
			token:  append([]byte(nil), attachment.handle.GetOpaqueToken()...),
		}
	}
	return attachment.duplex
}

// Resize 协调 size ownership 并刷新 size/epoch 快照。
func (attachment *poolAttachment) Resize(ctx context.Context, size *providerv1.Size, policy providerv1.ResizePolicy, takeOwnership bool, expectedEpoch uint64) (*providerv1.TerminalResizeResult, error) {
	if policy == providerv1.ResizePolicy_RESIZE_POLICY_UNSPECIFIED {
		policy = attachment.handle.GetResizePolicy()
	}
	result, err := attachment.client.Resize(ctx, attachment.handle.GetOpaqueToken(), size, policy, takeOwnership, expectedEpoch)
	if err != nil {
		return nil, typedProviderError(err)
	}
	attachment.mu.Lock()
	if result.GetSize() != nil {
		attachment.size = result.GetSize()
	}
	if control := result.GetResizeControl(); control != nil {
		attachment.control = control
	}
	if ownership := result.GetResizeControl().GetResizeOwnership(); ownership != nil && ownership.GetEpoch() != 0 {
		attachment.epoch = ownership.GetEpoch()
	}
	attachment.mu.Unlock()
	return result, nil
}

// ResizeLock 修改 owner size lock 并刷新 size/epoch 快照。
func (attachment *poolAttachment) ResizeLock(ctx context.Context, locked bool) (*providerv1.TerminalResizeResult, error) {
	result, err := attachment.client.ResizeLock(ctx, attachment.handle.GetOpaqueToken(), locked)
	if err != nil {
		return nil, typedProviderError(err)
	}
	attachment.mu.Lock()
	if result.GetSize() != nil {
		attachment.size = result.GetSize()
	}
	if control := result.GetResizeControl(); control != nil {
		attachment.control = control
	}
	if ownership := result.GetResizeControl().GetResizeOwnership(); ownership != nil && ownership.GetEpoch() != 0 {
		attachment.epoch = ownership.GetEpoch()
	}
	attachment.mu.Unlock()
	return result, nil
}

// Detach 关闭 PTY 流并释放 attachment resource。
func (attachment *poolAttachment) Detach(ctx context.Context) error {
	attachment.mu.Lock()
	duplex := attachment.duplex
	attachment.duplex = nil
	attachment.mu.Unlock()
	if duplex != nil {
		_ = duplex.Close()
	}
	return typedProviderError(attachment.client.Detach(ctx, attachment.handle.GetOpaqueToken()))
}

// attachmentDuplex 把 provider attachment stream 投影成纯 PTY 字节流：
// bootstrap/ready 在 WaitReady 或首次读写时内部完成，StreamClosed/StreamSyncLost
// 映射为 typed 错误，Send 走 provider TerminalInput（PTY 输入）。
type attachmentDuplex struct {
	client *poolprovider.Client
	token  []byte

	openOnce sync.Once
	frames   <-chan poolprovider.StreamFrame
	stop     func()
	openErr  error

	closeOnce sync.Once
	closed    atomic.Bool
}

// WaitReady 等待 attachment stream 的 bootstrap/ready 完成。幂等。
func (duplex *attachmentDuplex) WaitReady(ctx context.Context) error {
	if duplex.closed.Load() {
		return errors.New("provider/pool: attachment stream is closed")
	}
	return duplex.open(ctx)
}

// Receive 返回下一段 PTY 输出；流结束返回 io.EOF。
func (duplex *attachmentDuplex) Receive(ctx context.Context) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if duplex.closed.Load() {
		return nil, io.EOF
	}
	if err := duplex.open(ctx); err != nil {
		return nil, err
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case frame, ok := <-duplex.frames:
			if !ok {
				return nil, io.EOF
			}
			switch frame.Type {
			case providerproto.StreamPTYOutput:
				return frame.Payload, nil
			case providerproto.StreamClosed:
				exitCode := -1
				if decoded, decodeErr := wire.DecodeClosedPayload(frame.Payload); decodeErr == nil {
					exitCode = decoded
				}
				return nil, &terminalprovider.StreamClosedError{ExitCode: exitCode}
			case providerproto.StreamSyncLost:
				dropped, _ := wire.DecodeSyncLostPayload(frame.Payload)
				return nil, &terminalprovider.StreamSyncLostError{DroppedBytes: dropped}
			default:
				// ready 等流控帧对调用方不可见。
				continue
			}
		}
	}
}

// Send 把字节写入 PTY 输入。首次发送会先完成 output stream 的 bootstrap，
// 避免输入早于输出订阅导致回显丢失。
func (duplex *attachmentDuplex) Send(ctx context.Context, data []byte) error {
	if duplex.closed.Load() {
		return errors.New("provider/pool: attachment stream is closed")
	}
	if err := duplex.open(ctx); err != nil {
		return err
	}
	return typedProviderError(duplex.client.Input(ctx, duplex.token, data))
}

// Close 停止当前 PTY 输出流（保留 attachment resource；释放走 Detach）。幂等。
func (duplex *attachmentDuplex) Close() error {
	var closeErr error
	duplex.closeOnce.Do(func() {
		if duplex.frames != nil {
			if len(duplex.token) >= 2 {
				channel := binary.BigEndian.Uint16(duplex.token[:2])
				closeErr = duplex.client.SendStreamClose(channel)
			}
			if duplex.stop != nil {
				duplex.stop()
			}
		}
		duplex.closed.Store(true)
	})
	return closeErr
}

func (duplex *attachmentDuplex) open(ctx context.Context) error {
	duplex.openOnce.Do(func() {
		token := duplex.token
		if len(token) < 2 {
			duplex.openErr = errors.New("provider/pool: attachment resource token is malformed")
			return
		}
		channel := binary.BigEndian.Uint16(token[:2])
		if channel == 0 {
			duplex.openErr = errors.New("provider/pool: attachment stream channel is missing")
			return
		}
		frames, stop := duplex.client.Stream(channel)
		if err := duplex.client.SendBootstrapDone(channel); err != nil {
			stop()
			duplex.openErr = err
			return
		}
		if err := awaitStreamReady(frames, attachmentStreamReadyTimeout); err != nil {
			stop()
			duplex.openErr = err
			return
		}
		duplex.frames = frames
		duplex.stop = stop
	})
	return duplex.openErr
}

// ---- history/live ------------------------------------------------------------

// HistoryWindow 读取 history window 并签发 frozen token。
func (provider *TerminalProvider) HistoryWindow(ctx context.Context, request terminalprovider.HistoryRequest) (*providerv1.HistoryWindowResult, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	result, err := client.HistoryWindow(ctx, request.Command())
	if err != nil {
		return nil, typedProviderError(err)
	}
	return result, nil
}

// HistoryCopy 复制 frozen history token 的文本。
func (provider *TerminalProvider) HistoryCopy(ctx context.Context, request terminalprovider.HistoryCopyRequest) (*providerv1.HistoryCopyResult, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	terminalID := request.TerminalID
	if terminalID == "" && request.Window != nil {
		terminalID = request.Window.TerminalID
	}
	var window *providerv1.HistoryWindowCommand
	if request.Window != nil {
		window = request.Window.Command()
		window.Terminal = &providerv1.TerminalRef{TerminalId: terminalID}
	}
	result, err := client.HistoryCopy(ctx, &providerv1.HistoryCopyCommand{
		Terminal: &providerv1.TerminalRef{TerminalId: terminalID},
		Window:   window,
		MaxLines: request.MaxLines,
		MaxBytes: request.MaxBytes,
	})
	if err != nil {
		return nil, typedProviderError(err)
	}
	return result, nil
}

// HistorySearch 搜索 frozen history token。
func (provider *TerminalProvider) HistorySearch(ctx context.Context, request terminalprovider.HistorySearchRequest) (*providerv1.HistorySearchResult, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	result, err := client.HistorySearch(ctx, request.Command())
	if err != nil {
		return nil, typedProviderError(err)
	}
	return result, nil
}

// HistoryRelease 释放 frozen history token。
func (provider *TerminalProvider) HistoryRelease(ctx context.Context, terminalID, token string) error {
	client, err := provider.readyClient()
	if err != nil {
		return err
	}
	return typedProviderError(client.HistoryRelease(ctx, terminalID, token))
}

// HistoryBacklogStatus 返回 history backlog 诊断投影。
func (provider *TerminalProvider) HistoryBacklogStatus(ctx context.Context, terminalID string) (*providerv1.HistoryBacklogStatusResult, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	status, err := client.HistoryBacklogStatus(ctx, terminalID)
	if err != nil {
		return nil, typedProviderError(err)
	}
	return status, nil
}

// LiveScreen 等待 observedRevision 之后的 native screen delta。
func (provider *TerminalProvider) LiveScreen(ctx context.Context, terminalID string, observedRevision uint64) (*providerv1.NativeScreenResult, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	result, err := client.LiveScreenNext(ctx, terminalID, observedRevision)
	if err != nil {
		return nil, typedProviderError(err)
	}
	return result, nil
}

// ---- events/paths ------------------------------------------------------------

// Subscribe 建立 provider event 订阅并返回 subscription token 与 initial events。
func (provider *TerminalProvider) Subscribe(ctx context.Context, command *providerv1.EventSubscribeCommand) (*providerv1.EventSubscriptionResult, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	result, err := client.EventSubscribe(ctx, command)
	if err != nil {
		return nil, typedProviderError(err)
	}
	return result, nil
}

// Events 返回该 provider session 的 raw terminal event 流。
func (provider *TerminalProvider) Events(ctx context.Context) (<-chan *providerv1.TerminalEvent, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	return client.Events(ctx), nil
}

// EventRelease 释放 Subscribe 返回的 provider event 订阅 token。
func (provider *TerminalProvider) EventRelease(ctx context.Context, token []byte) error {
	client, err := provider.readyClient()
	if err != nil {
		return err
	}
	return typedProviderError(client.EventRelease(ctx, token))
}

// TerminalDefaults 返回 owning pool 机器的 shell/cwd 默认值。
func (provider *TerminalProvider) TerminalDefaults(ctx context.Context) (*providerv1.TerminalDefaults, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	defaults, err := client.TerminalDefaults(ctx)
	if err != nil {
		return nil, typedProviderError(err)
	}
	return defaults, nil
}

// ListDirectories 返回 path completion 窗口；limit<=0 表示用 provider 默认窗口。
func (provider *TerminalProvider) ListDirectories(ctx context.Context, path string, limit int32) (*providerv1.PathListDirectoriesResult, error) {
	client, err := provider.readyClient()
	if err != nil {
		return nil, err
	}
	result, err := client.ListDirectories(ctx, path, int(limit))
	if err != nil {
		return nil, typedProviderError(err)
	}
	return result, nil
}

package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	apimapping "github.com/anytty/anytty/api_mapping"
	daemonprovider "github.com/anytty/anytty/daemon/provider"
	"github.com/anytty/anytty/internal/providerproto"
	"github.com/anytty/anytty/proto/access/apipb"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
	"google.golang.org/protobuf/proto"
)

// TerminalDialTimeout bounds one provider session establishment (dial + Hello).
const TerminalDialTimeout = 5 * time.Second

// TerminalProvider adapts the access terminal routing to the daemon provider
// protocol. access/server keeps owning client channels/tokens and stream
// bridging; this adapter translates apipb commands to typed provider calls and
// provider results back to apipb envelopes.
type TerminalProvider struct {
	socket string
	client *daemonprovider.Client

	ctx    context.Context
	cancel context.CancelFunc

	events chan *apipb.EventEnvelope

	eventMu  sync.Mutex
	nextSub  uint64
	subs     map[string]*adapterSubscription
	pumpOnce sync.Once
}

var _ terminalprovider.Provider = (*TerminalProvider)(nil)

// DialTerminal 建立一条 daemon provider 会话：owner-only unix dial + provider Hello。
func DialTerminal(ctx context.Context, socket string) (*TerminalProvider, error) {
	path := strings.TrimSpace(socket)
	if path == "" {
		return nil, errors.New("provider/daemon: socket path is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	dialCtx, cancelDial := context.WithTimeout(ctx, TerminalDialTimeout)
	defer cancelDial()
	client, err := daemonprovider.Dial(dialCtx, path)
	if err != nil {
		return nil, fmt.Errorf("provider/daemon: %w", err)
	}
	pumpCtx, cancel := context.WithCancel(context.Background())
	provider := &TerminalProvider{
		socket: path,
		client: client,
		ctx:    pumpCtx,
		cancel: cancel,
		events: make(chan *apipb.EventEnvelope, 256),
		subs:   make(map[string]*adapterSubscription),
	}
	return provider, nil
}

// Socket 返回 provider socket 路径（诊断用）。
func (provider *TerminalProvider) Socket() string {
	if provider == nil {
		return ""
	}
	return provider.socket
}

// Execute 把一个 apipb application command 翻译为 provider 调用。
func (provider *TerminalProvider) Execute(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	if provider == nil || provider.client == nil {
		return nil, errors.New("provider/daemon: provider is closed")
	}
	if err := validateTerminalCommand(command); err != nil {
		return apiErrorResult(command, apimapping.ErrorToProto(err, false)), nil
	}
	result, err := provider.execute(ctx, command)
	if err != nil {
		return providerErrorResult(command, err), nil
	}
	return result, nil
}

func (provider *TerminalProvider) execute(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	endpointID := command.GetContext().GetSession().GetEndpointId()
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_TerminalDefaults:
		defaults, err := provider.client.TerminalDefaults(ctx)
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_TerminalDefaults{TerminalDefaults: terminalDefaultsToAPI(defaults)}), nil
	case *apipb.CommandEnvelope_TerminalCreate:
		created, err := provider.client.Create(ctx, terminalCreateSpecToProvider(value.TerminalCreate.GetTerminal()))
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_TerminalCreate{TerminalCreate: &apipb.TerminalCreateResult{
			Terminal: terminalInfoToAPI(endpointID, created),
		}}), nil
	case *apipb.CommandEnvelope_TerminalList:
		items, err := provider.client.List(ctx)
		if err != nil {
			return nil, err
		}
		list := &apipb.TerminalListResult{Terminals: make([]*apipb.TerminalInfo, 0, len(items))}
		for _, item := range items {
			list.Terminals = append(list.Terminals, terminalInfoToAPI(endpointID, item))
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_TerminalList{TerminalList: list}), nil
	case *apipb.CommandEnvelope_TerminalGet:
		info, err := provider.client.Get(ctx, value.TerminalGet.GetTerminal().GetTerminalId())
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_TerminalGet{TerminalGet: &apipb.TerminalGetResult{
			Terminal: terminalInfoToAPI(endpointID, info),
		}}), nil
	case *apipb.CommandEnvelope_TerminalRestart:
		if err := provider.client.Restart(ctx, value.TerminalRestart.GetTerminal().GetTerminalId()); err != nil {
			return nil, err
		}
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_TerminalKill:
		if err := provider.client.Kill(ctx, value.TerminalKill.GetTerminal().GetTerminalId()); err != nil {
			return nil, err
		}
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_TerminalRemove:
		if err := provider.client.Remove(ctx, value.TerminalRemove.GetTerminal().GetTerminalId()); err != nil {
			return nil, err
		}
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_TerminalSetMetadata:
		if err := provider.client.SetMetadata(ctx, value.TerminalSetMetadata.GetTerminal().GetTerminalId(), value.TerminalSetMetadata.GetName(), value.TerminalSetMetadata.GetTags()); err != nil {
			return nil, err
		}
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_TerminalSetTags:
		if err := provider.client.SetTags(ctx, value.TerminalSetTags.GetTerminal().GetTerminalId(), value.TerminalSetTags.GetTags()); err != nil {
			return nil, err
		}
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_TerminalAttach:
		return provider.attach(ctx, command, value.TerminalAttach)
	case *apipb.CommandEnvelope_TerminalDetach:
		if err := provider.client.Detach(ctx, value.TerminalDetach.GetAttachment().GetOpaqueToken()); err != nil {
			return nil, err
		}
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_TerminalInput:
		if err := provider.client.Input(ctx, value.TerminalInput.GetAttachment().GetOpaqueToken(), value.TerminalInput.GetData()); err != nil {
			return nil, err
		}
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_TerminalResize:
		resized, err := provider.client.Resize(ctx,
			value.TerminalResize.GetAttachment().GetOpaqueToken(),
			sizeToProviderFromAPI(value.TerminalResize.GetSize()),
			resizePolicyToProvider(value.TerminalResize.GetResizePolicy()),
			value.TerminalResize.GetTakeOwnership(),
			value.TerminalResize.GetExpectedOwnerEpoch(),
		)
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_TerminalResize{TerminalResize: &apipb.TerminalResizeResult{
			Size: sizeToAPI(resized.GetSize()), Resized: resized.GetResized(), ResizeControl: resizeControlToAPI(resized.GetResizeControl()),
		}}), nil
	case *apipb.CommandEnvelope_TerminalResizeLock:
		resized, err := provider.client.ResizeLock(ctx, value.TerminalResizeLock.GetAttachment().GetOpaqueToken(), value.TerminalResizeLock.GetLocked())
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_TerminalResize{TerminalResize: &apipb.TerminalResizeResult{
			Size: sizeToAPI(resized.GetSize()), Resized: resized.GetResized(), ResizeControl: resizeControlToAPI(resized.GetResizeControl()),
		}}), nil
	case *apipb.CommandEnvelope_PathListDirectories:
		directories, err := provider.client.ListDirectories(ctx, value.PathListDirectories.GetPrefix(), int(value.PathListDirectories.GetLimit()))
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_PathListDirectories{PathListDirectories: pathDirectoriesToAPI(directories)}), nil
	case *apipb.CommandEnvelope_HistoryWindow:
		window, err := provider.client.HistoryWindow(ctx, historyWindowRequestToProvider(value.HistoryWindow))
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_HistoryWindow{HistoryWindow: historyWindowToAPI(endpointID, window)}), nil
	case *apipb.CommandEnvelope_HistoryCopy:
		copied, err := provider.client.HistoryCopy(ctx, historyCopyCommandToProvider(value.HistoryCopy))
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_HistoryCopy{HistoryCopy: historyCopyResultToAPI(copied)}), nil
	case *apipb.CommandEnvelope_HistorySearch:
		found, err := provider.client.HistorySearch(ctx, historySearchCommandToProvider(value.HistorySearch))
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_HistorySearch{HistorySearch: historySearchResultToAPI(endpointID, found)}), nil
	case *apipb.CommandEnvelope_HistoryRelease:
		if err := provider.client.HistoryRelease(ctx, value.HistoryRelease.GetTerminal().GetTerminalId(), value.HistoryRelease.GetToken()); err != nil {
			return nil, err
		}
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_HistoryBacklogStatus:
		status, err := provider.client.HistoryBacklogStatus(ctx, value.HistoryBacklogStatus.GetTerminal().GetTerminalId())
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_HistoryBacklogStatus{HistoryBacklogStatus: historyBacklogToAPI(endpointID, status)}), nil
	case *apipb.CommandEnvelope_LiveScreenNext:
		snapshot, err := provider.client.LiveScreenNext(ctx, value.LiveScreenNext.GetTerminal().GetTerminalId(), value.LiveScreenNext.GetObservedRevision())
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_LiveScreen{LiveScreen: nativeScreenToAPI(endpointID, snapshot)}), nil
	case *apipb.CommandEnvelope_EventSubscribe:
		return provider.subscribeEvents(ctx, command, value.EventSubscribe)
	case *apipb.CommandEnvelope_ReleaseResource:
		return provider.releaseResource(ctx, command, value.ReleaseResource)
	case *apipb.CommandEnvelope_CancelOperation:
		return apiErrorResult(command, &apipb.ApiError{
			Code: apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE, Message: "operation cancellation is unavailable", Retryable: true,
		}), nil
	default:
		return apiErrorResult(command, &apipb.ApiError{
			Code: apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST, Message: "command is unsupported by the terminal provider",
		}), nil
	}
}

func (provider *TerminalProvider) attach(ctx context.Context, command *apipb.CommandEnvelope, attach *apipb.TerminalAttachCommand) (*apipb.ResultEnvelope, error) {
	endpointID := command.GetContext().GetSession().GetEndpointId()
	session := command.GetContext().GetSession()
	result, err := provider.client.Attach(ctx, &providerv1.TerminalAttachCommand{
		Terminal:     &providerv1.TerminalRef{TerminalId: attach.GetTerminal().GetTerminalId()},
		Mode:         attachmentModeToProvider(attach.GetMode()),
		ResizePolicy: resizePolicyToProvider(attach.GetResizePolicy()),
		SurfaceId:    attach.GetSurfaceId(),
		ViewId:       attach.GetViewId(),
	})
	if err != nil {
		return nil, err
	}
	handle := result.GetAttachment()
	attachment := &apipb.AttachmentHandle{
		Resource: &apipb.ResourceHandle{
			OpaqueToken: append([]byte(nil), handle.GetOpaqueToken()...),
			Kind:        apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT,
			Session:     cloneSessionStamp(session),
			Generation:  1,
		},
		Terminal:  &apipb.TerminalRef{EndpointId: endpointID, TerminalId: handle.GetTerminalId()},
		Operation: cloneOperationStamp(attach.GetOperation()),
		SurfaceId: handle.GetSurfaceId(),
		ViewId:    handle.GetViewId(),
	}
	return resultEnvelope(command, &apipb.ResultEnvelope_TerminalAttach{TerminalAttach: &apipb.TerminalAttachResult{
		Attachment:    attachment,
		Mode:          attachmentModeToAPI(handle.GetMode()),
		ResizePolicy:  resizePolicyToAPI(handle.GetResizePolicy()),
		Size:          sizeToAPI(handle.GetSize()),
		ResizeControl: resizeControlToAPI(result.GetResizeControl()),
	}}), nil
}

func (provider *TerminalProvider) releaseResource(ctx context.Context, command *apipb.CommandEnvelope, release *apipb.ReleaseResourceCommand) (*apipb.ResultEnvelope, error) {
	token := release.GetResource().GetOpaqueToken()
	if subscription := provider.subscriptionForToken(token); subscription != nil {
		if !subscription.synthetic {
			if err := provider.client.EventRelease(ctx, token); err != nil {
				return nil, err
			}
		}
		provider.removeSubscription(token)
		return acknowledgeEnvelope(command), nil
	}
	if err := provider.client.Detach(ctx, token); err != nil {
		return nil, err
	}
	return acknowledgeEnvelope(command), nil
}

// OpenStream 打开 attachment 的 PTY 输出流：bootstrap/ready 后返回有界帧流。
func (provider *TerminalProvider) OpenStream(ctx context.Context, resource *apipb.ResourceHandle) (terminalprovider.Stream, error) {
	if provider == nil || provider.client == nil {
		return nil, errors.New("provider/daemon: provider is closed")
	}
	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
	token := resource.GetOpaqueToken()
	if len(token) < 2 {
		return nil, errors.New("provider/daemon: attachment resource token is malformed")
	}
	channel := uint16(token[0])<<8 | uint16(token[1])
	frames, stop := provider.client.Stream(channel)
	if resource.GetKind() == apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT {
		if err := provider.client.SendBootstrapDone(channel); err != nil {
			stop()
			return nil, err
		}
		if err := awaitStreamReady(frames, attachmentStreamReadyTimeout); err != nil {
			stop()
			return nil, err
		}
	}
	return &attachmentStream{client: provider.client, channel: channel, frames: frames, stop: stop}, nil
}

// Events 返回 provider terminal event 的 apipb 投影流。
func (provider *TerminalProvider) Events(ctx context.Context) (<-chan *apipb.EventEnvelope, error) {
	if provider == nil || provider.client == nil {
		return nil, errors.New("provider/daemon: provider is closed")
	}
	provider.pumpOnce.Do(func() { go provider.runEventPump() })
	return provider.events, nil
}

// Done 在 provider 连接终止时关闭。
func (provider *TerminalProvider) Done() <-chan struct{} {
	if provider == nil || provider.client == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return provider.client.Done()
}

// Err 返回 provider 连接终止原因。
func (provider *TerminalProvider) Err() error {
	if provider == nil || provider.client == nil {
		return errors.New("provider/daemon: provider is closed")
	}
	return provider.client.Err()
}

// Close 释放 provider 连接与事件泵。
func (provider *TerminalProvider) Close() error {
	if provider == nil {
		return nil
	}
	if provider.cancel != nil {
		provider.cancel()
	}
	if provider.client == nil {
		return nil
	}
	return provider.client.Close()
}

const attachmentStreamReadyTimeout = 5 * time.Second

func awaitStreamReady(frames <-chan daemonprovider.StreamFrame, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return errors.New("provider/daemon: terminal attachment stream ready timed out")
		case frame, ok := <-frames:
			if !ok {
				return errors.New("provider/daemon: terminal attachment stream closed before ready")
			}
			if frame.Type == providerproto.StreamReady {
				return nil
			}
			if frame.Type == providerproto.StreamClosed {
				return errors.New("provider/daemon: terminal attachment stream closed before ready")
			}
		}
	}
}

type attachmentStream struct {
	client  *daemonprovider.Client
	channel uint16
	frames  <-chan daemonprovider.StreamFrame
	stop    func()
	once    sync.Once
}

func (stream *attachmentStream) Receive(ctx context.Context) (uint8, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	case frame, ok := <-stream.frames:
		if !ok {
			return 0, nil, nil
		}
		return frame.Type, frame.Payload, nil
	}
}

func (stream *attachmentStream) Send(context.Context, uint8, []byte) error {
	return errors.New("provider/daemon: terminal attachment stream is receive-only; input uses TerminalInputCommand")
}

func (stream *attachmentStream) Close() error {
	var err error
	stream.once.Do(func() {
		err = stream.client.SendStreamClose(stream.channel)
		stream.stop()
	})
	return err
}

// ---- result/error envelopes --------------------------------------------------

func resultEnvelope(command *apipb.CommandEnvelope, result interface{}) *apipb.ResultEnvelope {
	envelope := &apipb.ResultEnvelope{
		RequestId:     command.GetContext().GetRequestId(),
		OriginSession: cloneSessionStamp(command.GetContext().GetSession()),
	}
	switch value := result.(type) {
	case *apipb.ResultEnvelope:
		envelope.Result = value.GetResult()
	case *apipb.ResultEnvelope_TerminalDefaults:
		envelope.Result = value
	case *apipb.ResultEnvelope_TerminalCreate:
		envelope.Result = value
	case *apipb.ResultEnvelope_TerminalList:
		envelope.Result = value
	case *apipb.ResultEnvelope_TerminalGet:
		envelope.Result = value
	case *apipb.ResultEnvelope_TerminalAttach:
		envelope.Result = value
	case *apipb.ResultEnvelope_TerminalResize:
		envelope.Result = value
	case *apipb.ResultEnvelope_PathListDirectories:
		envelope.Result = value
	case *apipb.ResultEnvelope_HistoryWindow:
		envelope.Result = value
	case *apipb.ResultEnvelope_HistoryCopy:
		envelope.Result = value
	case *apipb.ResultEnvelope_HistorySearch:
		envelope.Result = value
	case *apipb.ResultEnvelope_HistoryBacklogStatus:
		envelope.Result = value
	case *apipb.ResultEnvelope_LiveScreen:
		envelope.Result = value
	case *apipb.ResultEnvelope_EventSubscription:
		envelope.Result = value
	case *apipb.ResultEnvelope_Acknowledge:
		envelope.Result = value
	case *apipb.ResultEnvelope_Error:
		envelope.Result = value
	}
	return envelope
}

func acknowledgeEnvelope(command *apipb.CommandEnvelope) *apipb.ResultEnvelope {
	return resultEnvelope(command, &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}})
}

func apiErrorResult(command *apipb.CommandEnvelope, apiError *apipb.ApiError) *apipb.ResultEnvelope {
	return resultEnvelope(command, &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_Error{Error: apiError}})
}

func providerErrorResult(command *apipb.CommandEnvelope, err error) *apipb.ResultEnvelope {
	var providerErr *daemonprovider.ProviderError
	if !errors.As(err, &providerErr) {
		return apiErrorResult(command, &apipb.ApiError{
			Code: apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE, Message: err.Error(), Retryable: true,
		})
	}
	apiError := &apipb.ApiError{Message: providerErr.Message}
	switch providerErr.Code {
	case providerproto.ErrorBadRequest:
		apiError.Code = apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST
	case providerproto.ErrorForbidden:
		apiError.Code = apipb.ApiErrorCode_API_ERROR_CODE_FORBIDDEN
	case providerproto.ErrorNotFound:
		apiError.Code = apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND
	case providerproto.ErrorConflict:
		apiError.Code = apipb.ApiErrorCode_API_ERROR_CODE_CONFLICT
	case providerproto.ErrorStaleResource:
		apiError.Code = apipb.ApiErrorCode_API_ERROR_CODE_STALE_RESOURCE
	case providerproto.ErrorExhausted:
		apiError.Code = apipb.ApiErrorCode_API_ERROR_CODE_RESOURCE_EXHAUSTED
		apiError.Retryable = true
	case providerproto.ErrorUnavailable:
		apiError.Code = apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE
		apiError.Retryable = true
	default:
		apiError.Code = apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL
	}
	return apiErrorResult(command, apiError)
}

func validateTerminalCommand(command *apipb.CommandEnvelope) error {
	if command == nil {
		return errors.New("command is required")
	}
	switch command.GetCommand().(type) {
	case *apipb.CommandEnvelope_TerminalDefaults,
		*apipb.CommandEnvelope_TerminalCreate,
		*apipb.CommandEnvelope_TerminalList,
		*apipb.CommandEnvelope_TerminalGet,
		*apipb.CommandEnvelope_TerminalRestart,
		*apipb.CommandEnvelope_TerminalKill,
		*apipb.CommandEnvelope_TerminalRemove,
		*apipb.CommandEnvelope_TerminalSetMetadata,
		*apipb.CommandEnvelope_TerminalSetTags,
		*apipb.CommandEnvelope_TerminalAttach,
		*apipb.CommandEnvelope_TerminalDetach,
		*apipb.CommandEnvelope_TerminalInput,
		*apipb.CommandEnvelope_TerminalResize,
		*apipb.CommandEnvelope_TerminalResizeLock,
		*apipb.CommandEnvelope_PathListDirectories:
		return apimapping.ValidateTerminalCommand(command)
	case *apipb.CommandEnvelope_HistoryWindow,
		*apipb.CommandEnvelope_HistoryCopy,
		*apipb.CommandEnvelope_HistoryRelease,
		*apipb.CommandEnvelope_HistoryBacklogStatus,
		*apipb.CommandEnvelope_HistorySearch,
		*apipb.CommandEnvelope_LiveScreenNext:
		return apimapping.ValidateHistoryLiveCommand(command)
	case *apipb.CommandEnvelope_EventSubscribe:
		return apimapping.ValidateEventSubscribeCommand(command)
	default:
		return nil
	}
}

func cloneSessionStamp(stamp *apipb.EndpointSessionStamp) *apipb.EndpointSessionStamp {
	if stamp == nil {
		return nil
	}
	return proto.Clone(stamp).(*apipb.EndpointSessionStamp)
}

func cloneOperationStamp(stamp *apipb.OperationStamp) *apipb.OperationStamp {
	if stamp == nil {
		return nil
	}
	return proto.Clone(stamp).(*apipb.OperationStamp)
}

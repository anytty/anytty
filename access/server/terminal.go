package server

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"

	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	"github.com/anytty/anytty/access/server/terminalmap"
	apimapping "github.com/anytty/anytty/api_mapping"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
	"google.golang.org/protobuf/proto"
)

// executeTerminal 是 terminal family 的 typed 执行路径：先把 apipb command
// 投影为 providerv1 DTO，调用 terminal.Provider 的 typed 方法，再把结果投影回
// 客户端看到的 apipb envelope。access-issued attachment token 在这里解析为
// typed Attachment；provider token 不离开 access。
func (session *session) executeTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	provider, err := session.ensureProvider(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateTerminalCommand(command); err != nil {
		return nil, &routeError{apiError: apimapping.ErrorToProto(err, false)}
	}
	endpointID := command.GetContext().GetSession().GetEndpointId()
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_TerminalDefaults:
		defaults, err := provider.TerminalDefaults(ctx)
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_TerminalDefaults{TerminalDefaults: terminalmap.TerminalDefaultsToAPI(defaults)}), nil
	case *apipb.CommandEnvelope_TerminalCreate:
		created, err := provider.Create(ctx, terminalmap.TerminalCreateSpecToProvider(value.TerminalCreate.GetTerminal()))
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_TerminalCreate{TerminalCreate: &apipb.TerminalCreateResult{
			Terminal: terminalmap.TerminalInfoToAPI(endpointID, created),
		}}), nil
	case *apipb.CommandEnvelope_TerminalList:
		items, err := provider.List(ctx)
		if err != nil {
			return nil, err
		}
		list := &apipb.TerminalListResult{Terminals: make([]*apipb.TerminalInfo, 0, len(items))}
		for _, item := range items {
			list.Terminals = append(list.Terminals, terminalmap.TerminalInfoToAPI(endpointID, item))
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_TerminalList{TerminalList: list}), nil
	case *apipb.CommandEnvelope_TerminalGet:
		info, err := provider.Get(ctx, value.TerminalGet.GetTerminal().GetTerminalId())
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_TerminalGet{TerminalGet: &apipb.TerminalGetResult{
			Terminal: terminalmap.TerminalInfoToAPI(endpointID, info),
		}}), nil
	case *apipb.CommandEnvelope_TerminalRestart:
		if _, err := provider.Restart(ctx, value.TerminalRestart.GetTerminal().GetTerminalId()); err != nil {
			return nil, err
		}
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_TerminalKill:
		if err := provider.Kill(ctx, value.TerminalKill.GetTerminal().GetTerminalId()); err != nil {
			return nil, err
		}
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_TerminalRemove:
		if err := provider.Remove(ctx, value.TerminalRemove.GetTerminal().GetTerminalId()); err != nil {
			return nil, err
		}
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_TerminalSetMetadata:
		name := value.TerminalSetMetadata.GetName()
		tags := value.TerminalSetMetadata.GetTags()
		if _, err := provider.SetMetadata(ctx, value.TerminalSetMetadata.GetTerminal().GetTerminalId(), terminalprovider.MetadataPatch{
			Name: &name, Tags: &tags,
		}); err != nil {
			return nil, err
		}
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_TerminalSetTags:
		// 客户端 terminal.set-tags 是整体替换：走 provider wire 的 SetTags，
		// Replace 保证旧 tag 不会被合并保留。
		if _, err := provider.SetTags(ctx, value.TerminalSetTags.GetTerminal().GetTerminalId(), terminalprovider.TagsPatch{
			Set: value.TerminalSetTags.GetTags(), Replace: true,
		}); err != nil {
			return nil, err
		}
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_TerminalAttach:
		return session.attachTerminal(ctx, provider, command, value.TerminalAttach)
	case *apipb.CommandEnvelope_TerminalDetach:
		binding, err := session.attachmentBinding(value.TerminalDetach.GetAttachment().GetOpaqueToken())
		if err != nil {
			return nil, err
		}
		if err := binding.attachment.Detach(ctx); err != nil {
			return nil, err
		}
		session.retireBinding(binding)
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_TerminalInput:
		binding, err := session.attachmentBinding(value.TerminalInput.GetAttachment().GetOpaqueToken())
		if err != nil {
			return nil, err
		}
		if err := binding.attachment.Stream().Send(ctx, value.TerminalInput.GetData()); err != nil {
			return nil, err
		}
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_TerminalResize:
		binding, err := session.attachmentBinding(value.TerminalResize.GetAttachment().GetOpaqueToken())
		if err != nil {
			return nil, err
		}
		resized, err := binding.attachment.Resize(ctx,
			terminalmap.SizeFromAPI(value.TerminalResize.GetSize()),
			terminalmap.ResizePolicyToProvider(value.TerminalResize.GetResizePolicy()),
			value.TerminalResize.GetTakeOwnership(),
			value.TerminalResize.GetExpectedOwnerEpoch(),
		)
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_TerminalResize{TerminalResize: &apipb.TerminalResizeResult{
			Size: terminalmap.SizeToAPI(resized.GetSize()), Resized: resized.GetResized(), ResizeControl: terminalmap.ResizeControlToAPI(resized.GetResizeControl()),
		}}), nil
	case *apipb.CommandEnvelope_TerminalResizeLock:
		binding, err := session.attachmentBinding(value.TerminalResizeLock.GetAttachment().GetOpaqueToken())
		if err != nil {
			return nil, err
		}
		resized, err := binding.attachment.ResizeLock(ctx, value.TerminalResizeLock.GetLocked())
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_TerminalResize{TerminalResize: &apipb.TerminalResizeResult{
			Size: terminalmap.SizeToAPI(resized.GetSize()), Resized: resized.GetResized(), ResizeControl: terminalmap.ResizeControlToAPI(resized.GetResizeControl()),
		}}), nil
	case *apipb.CommandEnvelope_PathListDirectories:
		directories, err := provider.ListDirectories(ctx, value.PathListDirectories.GetPrefix(), value.PathListDirectories.GetLimit())
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_PathListDirectories{PathListDirectories: terminalmap.PathDirectoriesToAPI(directories)}), nil
	case *apipb.CommandEnvelope_HistoryWindow:
		window, err := provider.HistoryWindow(ctx, terminalprovider.HistoryRequestFromCommand(terminalmap.HistoryWindowRequestToProvider(value.HistoryWindow)))
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_HistoryWindow{HistoryWindow: terminalmap.HistoryWindowToAPI(endpointID, window)}), nil
	case *apipb.CommandEnvelope_HistoryCopy:
		copied, err := provider.HistoryCopy(ctx, terminalprovider.HistoryCopyRequestFromCommand(terminalmap.HistoryCopyCommandToProvider(value.HistoryCopy)))
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_HistoryCopy{HistoryCopy: terminalmap.HistoryCopyResultToAPI(copied)}), nil
	case *apipb.CommandEnvelope_HistorySearch:
		found, err := provider.HistorySearch(ctx, terminalprovider.HistorySearchRequestFromCommand(terminalmap.HistorySearchCommandToProvider(value.HistorySearch)))
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_HistorySearch{HistorySearch: terminalmap.HistorySearchResultToAPI(endpointID, found)}), nil
	case *apipb.CommandEnvelope_HistoryRelease:
		if err := provider.HistoryRelease(ctx, value.HistoryRelease.GetTerminal().GetTerminalId(), value.HistoryRelease.GetToken()); err != nil {
			return nil, err
		}
		return acknowledgeEnvelope(command), nil
	case *apipb.CommandEnvelope_HistoryBacklogStatus:
		status, err := provider.HistoryBacklogStatus(ctx, value.HistoryBacklogStatus.GetTerminal().GetTerminalId())
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_HistoryBacklogStatus{HistoryBacklogStatus: terminalmap.HistoryBacklogToAPI(endpointID, status)}), nil
	case *apipb.CommandEnvelope_LiveScreenNext:
		snapshot, err := provider.LiveScreen(ctx, value.LiveScreenNext.GetTerminal().GetTerminalId(), value.LiveScreenNext.GetObservedRevision())
		if err != nil {
			return nil, err
		}
		return resultEnvelope(command, &apipb.ResultEnvelope_LiveScreen{LiveScreen: terminalmap.NativeScreenToAPI(endpointID, snapshot)}), nil
	case *apipb.CommandEnvelope_EventSubscribe:
		return session.executeEventSubscribe(ctx, provider, command, value.EventSubscribe)
	case *apipb.CommandEnvelope_ReleaseResource:
		return session.executeReleaseResource(ctx, provider, command, value.ReleaseResource)
	case *apipb.CommandEnvelope_CancelOperation:
		return apiErrorEnvelope(command, &apipb.ApiError{
			Code: apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE, Message: "operation cancellation is unavailable", Retryable: true,
		}), nil
	default:
		return apiErrorEnvelope(command, &apipb.ApiError{
			Code: apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST, Message: "command is unsupported by the terminal provider",
		}), nil
	}
}

// attachTerminal 建立 typed attachment，并把 provider resource 重签为
// access-issued channel/token 后发布给客户端。
func (session *session) attachTerminal(ctx context.Context, provider terminalprovider.Provider, command *apipb.CommandEnvelope, attach *apipb.TerminalAttachCommand) (*apipb.ResultEnvelope, error) {
	endpointID := command.GetContext().GetSession().GetEndpointId()
	attachment, err := provider.Attach(ctx, terminalprovider.AttachRequest{
		TerminalID:   attach.GetTerminal().GetTerminalId(),
		Mode:         terminalmap.AttachmentModeToProvider(attach.GetMode()),
		ResizePolicy: terminalmap.ResizePolicyToProvider(attach.GetResizePolicy()),
		SurfaceID:    attach.GetSurfaceId(),
		ViewID:       attach.GetViewId(),
	})
	if err != nil {
		return nil, err
	}
	handle := attachment.Handle()
	if handle == nil {
		return nil, errors.New("access/server: provider attachment handle is missing")
	}
	envelope := resultEnvelope(command, &apipb.ResultEnvelope_TerminalAttach{TerminalAttach: &apipb.TerminalAttachResult{
		Attachment: &apipb.AttachmentHandle{
			Resource: &apipb.ResourceHandle{
				OpaqueToken: append([]byte(nil), handle.GetOpaqueToken()...),
				Kind:        apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT,
				Session:     cloneSessionStamp(command.GetContext().GetSession()),
				Generation:  1,
			},
			Terminal:  &apipb.TerminalRef{EndpointId: endpointID, TerminalId: handle.GetTerminalId()},
			Operation: cloneOperationStamp(attach.GetOperation()),
			SurfaceId: handle.GetSurfaceId(),
			ViewId:    handle.GetViewId(),
		},
		Mode:          terminalmap.AttachmentModeToAPI(handle.GetMode()),
		ResizePolicy:  terminalmap.ResizePolicyToAPI(handle.GetResizePolicy()),
		Size:          terminalmap.SizeToAPI(handle.GetSize()),
		ResizeControl: terminalmap.ResizeControlToAPI(attachment.ResizeControl()),
	}})
	if err := session.publishAttachmentBinding(envelope.GetTerminalAttach().GetAttachment(), attachment); err != nil {
		_ = attachment.Detach(ctx)
		return nil, err
	}
	return envelope, nil
}

// publishAttachmentBinding 为 provider attachment 分配 access client channel/
// token 并登记 binding；provider token 只保存在 provider Attachment 内。
func (session *session) publishAttachmentBinding(handle *apipb.AttachmentHandle, attachment terminalprovider.Attachment) error {
	resource := handle.GetResource()
	token := make([]byte, accessTokenBytes)
	if _, err := rand.Read(token); err != nil {
		return fmt.Errorf("access/server: allocate resource token: %w", err)
	}
	session.streamsMu.Lock()
	clientChannel, err := session.allocateChannelLocked()
	if err != nil {
		session.streamsMu.Unlock()
		return err
	}
	binary.BigEndian.PutUint16(token[:2], clientChannel)
	binding := &streamBinding{
		clientChannel: clientChannel,
		kind:          resource.GetKind(),
		accessToken:   token,
		attachment:    attachment,
	}
	session.channels[clientChannel] = binding
	session.tokens[string(token)] = binding
	session.streamsMu.Unlock()
	resource.OpaqueToken = token
	return nil
}

// attachmentBinding 按 access token 取回 attachment binding。
func (session *session) attachmentBinding(token []byte) (*streamBinding, error) {
	if len(token) == 0 {
		return nil, fmt.Errorf("%w: attachment token is required", terminalprovider.ErrNotFound)
	}
	session.streamsMu.Lock()
	binding := session.tokens[string(token)]
	session.streamsMu.Unlock()
	if binding == nil || binding.local != nil || binding.attachment == nil {
		return nil, fmt.Errorf("%w: attachment resource is not bound", terminalprovider.ErrNotFound)
	}
	return binding, nil
}

// executeEventSubscribe 建立 provider terminal event 订阅（storage-only 订阅
// 只在 access 本地保留合成 token），并接管 storage 变更广播。
func (session *session) executeEventSubscribe(ctx context.Context, provider terminalprovider.Provider, command *apipb.CommandEnvelope, subscribe *apipb.EventSubscribeCommand) (*apipb.ResultEnvelope, error) {
	sessionStamp := command.GetContext().GetSession()
	terminalID := subscribe.GetTerminal().GetTerminalId()
	types, terminalEvents := terminalmap.ProviderEventTypes(subscribe.GetTypes())
	subscription := &eventSubscription{
		session:        cloneSessionStamp(sessionStamp),
		endpointID:     sessionStamp.GetEndpointId(),
		terminalID:     terminalID,
		types:          types,
		terminalEvents: terminalEvents,
	}
	if terminalEvents {
		result, err := provider.Subscribe(ctx, &providerv1.EventSubscribeCommand{TerminalId: terminalID, Types: types})
		if err != nil {
			return nil, err
		}
		subscription.token = append([]byte(nil), result.GetOpaqueToken()...)
		session.addEventSubscription(subscription)
		for _, event := range result.GetInitialEvents() {
			session.dispatchTerminalEvent(subscription, event)
		}
	} else {
		// storage-only 订阅由 access 本地处理；这里只保留可 release 的 token。
		subscription.token = terminalmap.SyntheticSubscriptionToken()
		subscription.synthetic = true
		session.addEventSubscription(subscription)
	}
	if len(subscription.token) == 0 {
		return nil, errors.New("access/server: event subscription token is empty")
	}
	envelope := resultEnvelope(command, &apipb.ResultEnvelope_EventSubscription{EventSubscription: &apipb.EventSubscriptionResult{
		Subscription: &apipb.ResourceHandle{
			OpaqueToken: append([]byte(nil), subscription.token...),
			Kind:        apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION,
			Session:     cloneSessionStamp(sessionStamp),
			Generation:  1,
		},
	}})
	session.registerStorageSubscription(command, envelope)
	return envelope, nil
}

// executeReleaseResource 释放 provider event 订阅或 attachment resource。
func (session *session) executeReleaseResource(ctx context.Context, provider terminalprovider.Provider, command *apipb.CommandEnvelope, release *apipb.ReleaseResourceCommand) (*apipb.ResultEnvelope, error) {
	resource := release.GetResource()
	token := resource.GetOpaqueToken()
	if subscription := session.eventSubscriptionForToken(token); subscription != nil {
		if !subscription.synthetic {
			if err := provider.EventRelease(ctx, token); err != nil {
				return nil, err
			}
		}
		session.removeEventSubscription(token)
		session.releaseLocalSubscription(resource)
		return acknowledgeEnvelope(command), nil
	}
	binding, err := session.attachmentBinding(token)
	if err != nil {
		return nil, err
	}
	if err := binding.attachment.Detach(ctx); err != nil {
		return nil, err
	}
	session.retireBinding(binding)
	return acknowledgeEnvelope(command), nil
}

// addEventSubscription 登记一个订阅投影；同 token 覆盖旧登记。
func (session *session) addEventSubscription(subscription *eventSubscription) {
	session.subscriptionsMu.Lock()
	if session.subscriptions == nil {
		session.subscriptions = make(map[string]*eventSubscription)
	}
	session.subscriptions[string(subscription.token)] = subscription
	session.subscriptionsMu.Unlock()
}

func (session *session) eventSubscriptionForToken(token []byte) *eventSubscription {
	if len(token) == 0 {
		return nil
	}
	session.subscriptionsMu.Lock()
	defer session.subscriptionsMu.Unlock()
	return session.subscriptions[string(token)]
}

func (session *session) removeEventSubscription(token []byte) {
	session.subscriptionsMu.Lock()
	delete(session.subscriptions, string(token))
	session.subscriptionsMu.Unlock()
}

// releaseProviderSubscriptions 在 provider 连接终止时丢弃 provider-backed 订阅；
// storage-only（synthetic）订阅继续由 access 本地广播。
func (session *session) releaseProviderSubscriptions() {
	session.subscriptionsMu.Lock()
	for token, subscription := range session.subscriptions {
		if !subscription.synthetic {
			delete(session.subscriptions, token)
		}
	}
	session.subscriptionsMu.Unlock()
}

// dispatchProviderEvent 把一个 raw provider event 分发给所有命中的订阅。
func (session *session) dispatchProviderEvent(event *providerv1.TerminalEvent) {
	if event == nil {
		return
	}
	session.subscriptionsMu.Lock()
	matched := make([]*eventSubscription, 0, len(session.subscriptions))
	for _, subscription := range session.subscriptions {
		if !subscription.terminalEvents {
			continue
		}
		if !terminalmap.EventMatchesFilter(subscription.terminalID, subscription.types, event) {
			continue
		}
		matched = append(matched, subscription)
	}
	session.subscriptionsMu.Unlock()
	for _, subscription := range matched {
		session.dispatchTerminalEvent(subscription, event)
	}
}

// dispatchTerminalEvent 把 typed event 投影为订阅 envelope 并发送 event frame。
func (session *session) dispatchTerminalEvent(subscription *eventSubscription, event *providerv1.TerminalEvent) {
	envelope := terminalmap.TerminalEventToEnvelope(subscription.endpointID, subscription.session, subscription.token, event)
	if envelope == nil {
		return
	}
	payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(envelope)
	if err != nil {
		return
	}
	_ = session.sendFrame(0, wire.TypeEvent, payload)
}

// validateTerminalCommand 复刻 legacy passthrough 的校验分派：terminal 面用
// ValidateTerminalCommand，history/live 用 ValidateHistoryLiveCommand，event
// subscribe 用 ValidateEventSubscribeCommand；release/cancel 无额外校验。
func validateTerminalCommand(command *apipb.CommandEnvelope) error {
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

// resultEnvelope 构造带 client correlation 的结果信封；result 必须是 apipb
// ResultEnvelope oneof 包装类型。
func resultEnvelope(command *apipb.CommandEnvelope, result interface{}) *apipb.ResultEnvelope {
	envelope := newResultEnvelope(command)
	switch value := result.(type) {
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
	}
	return envelope
}

func acknowledgeEnvelope(command *apipb.CommandEnvelope) *apipb.ResultEnvelope {
	envelope := newResultEnvelope(command)
	envelope.Result = &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}
	return envelope
}

func apiErrorEnvelope(command *apipb.CommandEnvelope, apiError *apipb.ApiError) *apipb.ResultEnvelope {
	envelope := newResultEnvelope(command)
	envelope.Result = &apipb.ResultEnvelope_Error{Error: apiError}
	return envelope
}

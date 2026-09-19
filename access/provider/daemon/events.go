package daemon

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/anytty/anytty/proto/access/apipb"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
)

// adapterSubscription 是 access 会话上的一个 event subscription。
// terminalEvents=false 表示只订阅 access-local 事件（storage），不向 provider 订阅。
type adapterSubscription struct {
	token          []byte
	session        *apipb.EndpointSessionStamp
	endpointID     string
	terminalID     string
	types          []providerv1.TerminalEventType
	terminalEvents bool
	synthetic      bool
}

func (provider *TerminalProvider) subscribeEvents(ctx context.Context, command *apipb.CommandEnvelope, subscribe *apipb.EventSubscribeCommand) (*apipb.ResultEnvelope, error) {
	session := command.GetContext().GetSession()
	terminalID := subscribe.GetTerminal().GetTerminalId()
	types, terminalEvents := providerEventTypes(subscribe.GetTypes())
	subscription := &adapterSubscription{
		session:        cloneSessionStamp(session),
		endpointID:     session.GetEndpointId(),
		terminalID:     terminalID,
		types:          types,
		terminalEvents: terminalEvents,
	}
	if terminalEvents {
		result, err := provider.client.EventSubscribe(ctx, &providerv1.EventSubscribeCommand{
			TerminalId: terminalID,
			Types:      types,
		})
		if err != nil {
			return nil, err
		}
		subscription.token = append([]byte(nil), result.GetOpaqueToken()...)
		provider.addSubscription(subscription)
		for _, event := range result.GetInitialEvents() {
			provider.enqueueEnvelope(subscription, event)
		}
	} else {
		// storage-only 订阅由 access 本地处理；这里只保留可 release 的 token。
		subscription.token = syntheticSubscriptionToken()
		subscription.synthetic = true
		provider.addSubscription(subscription)
	}
	if len(subscription.token) == 0 {
		return nil, fmt.Errorf("provider/daemon: event subscription token is empty")
	}
	return resultEnvelope(command, &apipb.ResultEnvelope_EventSubscription{EventSubscription: &apipb.EventSubscriptionResult{
		Subscription: &apipb.ResourceHandle{
			OpaqueToken: append([]byte(nil), subscription.token...),
			Kind:        apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION,
			Session:     cloneSessionStamp(session),
			Generation:  1,
		},
	}}), nil
}

func (provider *TerminalProvider) addSubscription(subscription *adapterSubscription) {
	provider.eventMu.Lock()
	provider.nextSub++
	provider.subs[string(subscription.token)] = subscription
	provider.eventMu.Unlock()
}

func (provider *TerminalProvider) subscriptionForToken(token []byte) *adapterSubscription {
	provider.eventMu.Lock()
	defer provider.eventMu.Unlock()
	return provider.subs[string(token)]
}

func (provider *TerminalProvider) removeSubscription(token []byte) {
	provider.eventMu.Lock()
	delete(provider.subs, string(token))
	provider.eventMu.Unlock()
}

func (provider *TerminalProvider) runEventPump() {
	events := provider.client.Events(provider.ctx)
	for {
		select {
		case <-provider.ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			provider.dispatchProviderEvent(event)
		}
	}
}

func (provider *TerminalProvider) dispatchProviderEvent(event *providerv1.TerminalEvent) {
	if event == nil {
		return
	}
	provider.eventMu.Lock()
	envelopes := make([]*apipb.EventEnvelope, 0, len(provider.subs))
	for _, subscription := range provider.subs {
		if !subscriptionMatchesEvent(subscription, event) {
			continue
		}
		envelopes = append(envelopes, buildEventEnvelope(subscription, event))
	}
	provider.eventMu.Unlock()
	for _, envelope := range envelopes {
		select {
		case provider.events <- envelope:
		default:
		}
	}
}

func (provider *TerminalProvider) enqueueEnvelope(subscription *adapterSubscription, event *providerv1.TerminalEvent) {
	if event == nil {
		return
	}
	envelope := buildEventEnvelope(subscription, event)
	if envelope == nil {
		return
	}
	select {
	case provider.events <- envelope:
	default:
	}
}

func subscriptionMatchesEvent(subscription *adapterSubscription, event *providerv1.TerminalEvent) bool {
	if !subscription.terminalEvents {
		return false
	}
	if subscription.terminalID != "" && subscription.terminalID != event.GetTerminalId() {
		return false
	}
	if len(subscription.types) == 0 {
		return true
	}
	for _, typ := range subscription.types {
		if typ == event.GetType() {
			return true
		}
	}
	return false
}

func buildEventEnvelope(subscription *adapterSubscription, event *providerv1.TerminalEvent) *apipb.EventEnvelope {
	envelope := &apipb.EventEnvelope{
		EventId:           fmt.Sprintf("terminal.%s-%d", strings.ToLower(event.GetType().String()), event.GetTimestampUnixNano()),
		TimestampUnixNano: event.GetTimestampUnixNano(),
		ApiVersion:        &apipb.ApiVersion{Major: 1},
		OriginSession:     cloneSessionStamp(subscription.session),
		Subscription: &apipb.ResourceHandle{
			OpaqueToken: append([]byte(nil), subscription.token...),
			Kind:        apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION,
			Session:     cloneSessionStamp(subscription.session),
			Generation:  1,
		},
	}
	if event.GetTerminal() == nil {
		return nil
	}
	lifecycle := &apipb.TerminalLifecycleEvent{
		Terminal: terminalInfoToAPI(subscription.endpointID, event.GetTerminal()),
	}
	if projection := event.GetAttachment(); projection != nil {
		lifecycle.AttachmentProjection = true
		lifecycle.ResizeControl = resizeControlToAPI(projection.GetResizeControl())
		lifecycle.ResizeEpoch = projection.GetResizeEpoch()
	}
	envelope.Event = &apipb.EventEnvelope_TerminalLifecycle{TerminalLifecycle: lifecycle}
	return envelope
}

// providerEventTypes 把 apipb 事件类型映射为 provider terminal 事件类型。
// 返回 terminalEvents=false 表示该订阅不包含 terminal 事件（例如 storage-only）。
func providerEventTypes(types []apipb.ApplicationEventType) ([]providerv1.TerminalEventType, bool) {
	if len(types) == 0 {
		// 空 types 表示订阅全部事件；terminal 面由 provider 提供。
		return nil, true
	}
	out := make([]providerv1.TerminalEventType, 0, len(types))
	for _, typ := range types {
		switch typ {
		case apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE:
			out = append(out,
				providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_CREATED,
				providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_EXITED,
				providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_METADATA_CHANGED,
				providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_REMOVED,
				providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_CHANGED,
			)
		}
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

func syntheticSubscriptionToken() []byte {
	token := make([]byte, 10)
	token[0] = 'e'
	token[1] = 'v'
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		binary.BigEndian.PutUint64(raw, uint64(time.Now().UnixNano()))
	}
	copy(token[2:], raw)
	return token
}

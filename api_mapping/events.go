package apimapping

import (
	"fmt"

	corev2 "github.com/anytty/anytty/pool/core"
	"github.com/anytty/anytty/proto/access/apipb"
	"google.golang.org/protobuf/proto"
)

// ValidateEventSubscribeCommand 校验事件类型、terminal fence 与 storage filter。
func ValidateEventSubscribeCommand(command *apipb.CommandEnvelope) error {
	if err := ValidateRequestContext(RequestContextForCommand(command)); err != nil {
		return err
	}
	value, ok := command.GetCommand().(*apipb.CommandEnvelope_EventSubscribe)
	if !ok || value.EventSubscribe == nil {
		return validation("command", "event subscribe command is required")
	}
	if value.EventSubscribe.GetTerminal() != nil {
		if err := validateTerminalRefForContext(value.EventSubscribe.GetTerminal(), RequestContextForCommand(command)); err != nil {
			return err
		}
	}
	for _, eventType := range value.EventSubscribe.GetTypes() {
		if eventType == apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_UNSPECIFIED {
			return validation("event_subscribe.types", "unspecified event type is not allowed")
		}
		switch eventType {
		case apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE,
			apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_STORAGE_CHANGED:
		default:
			return validation("event_subscribe.types", "event type is not supported")
		}
	}
	return nil
}

// EventFilterFromProto 映射为 core event broker filter。
// storage 变更由 access 本地终结并按 storage filter 直接过滤（access/server），
// 因而这里只投影 pool broker 的 terminal 维度。
func EventFilterFromProto(command *apipb.EventSubscribeCommand) corev2.EventFilter {
	filter := corev2.EventFilter{TerminalID: command.GetTerminal().GetTerminalId()}
	for _, eventType := range command.GetTypes() {
		switch eventType {
		case apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE:
			filter.Types = append(filter.Types, corev2.EventTerminalCreated, corev2.EventTerminalExited, corev2.EventTerminalMetadataChanged, corev2.EventTerminalRemoved, corev2.EventTerminalChanged)
		}
	}
	return filter
}

// EventSubscriptionToProto 把 core subscription token 绑定到当前 Proto session。
func EventSubscriptionToProto(session *apipb.EndpointSessionStamp, token []byte) *apipb.EventSubscriptionResult {
	return &apipb.EventSubscriptionResult{Subscription: &apipb.ResourceHandle{OpaqueToken: cloneBytes(token), Kind: apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION, Session: cloneSessionStamp(session), Generation: 1}}
}

// EncodeEventEnvelope 把 core-native event 转为公共 Proto event frame payload。
// storage 变更帧由 access/server 本地直接编码，不经过 pool event。
func EncodeEventEnvelope(endpointID string, session *apipb.EndpointSessionStamp, subscriptionToken []byte, event corev2.Event) ([]byte, error) {
	envelope := &apipb.EventEnvelope{EventId: fmt.Sprintf("%s-%d", event.Type, event.Timestamp.UnixNano()), TimestampUnixNano: event.Timestamp.UnixNano(), ApiVersion: &apipb.ApiVersion{Major: 1}, OriginSession: cloneSessionStamp(session), Subscription: &apipb.ResourceHandle{OpaqueToken: cloneBytes(subscriptionToken), Kind: apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION, Session: cloneSessionStamp(session), Generation: 1}}
	switch {
	case event.Terminal != nil:
		attachmentCount := 0
		if event.Attachment != nil {
			attachmentCount = event.Attachment.AttachmentCount
		}
		terminal, err := TerminalInfoToProto(endpointID, *event.Terminal, attachmentCount)
		if err != nil {
			return nil, err
		}
		lifecycle := &apipb.TerminalLifecycleEvent{Terminal: terminal}
		if event.Attachment != nil {
			lifecycle.AttachmentProjection = true
			lifecycle.ResizeControl = resizeControlToProto(event.Attachment.ResizeControl)
			lifecycle.ResizeEpoch = event.Attachment.ResizeEpoch
		}
		envelope.Event = &apipb.EventEnvelope_TerminalLifecycle{TerminalLifecycle: lifecycle}
	default:
		return nil, fmt.Errorf("unsupported application event %q", event.Type)
	}
	return proto.MarshalOptions{Deterministic: true}.Marshal(envelope)
}

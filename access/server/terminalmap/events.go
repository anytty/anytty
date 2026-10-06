package terminalmap

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/anytty/anytty/proto/access/apipb"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
	"google.golang.org/protobuf/proto"
)

// TerminalEventToEnvelope 把 typed provider terminal event 投影为客户端可见的
// apipb application event envelope。endpointID/session/token 是该订阅在 access
// 会话上的身份快照；provider 侧身份（socket 路径等）不会出现在 envelope 中。
// event 不携带 terminal 快照时返回 nil。
func TerminalEventToEnvelope(endpointID string, session *apipb.EndpointSessionStamp, token []byte, event *providerv1.TerminalEvent) *apipb.EventEnvelope {
	if event == nil {
		return nil
	}
	envelope := &apipb.EventEnvelope{
		EventId:           fmt.Sprintf("terminal.%s-%d", strings.ToLower(event.GetType().String()), event.GetTimestampUnixNano()),
		TimestampUnixNano: event.GetTimestampUnixNano(),
		ApiVersion:        &apipb.ApiVersion{Major: 1},
		OriginSession:     cloneSessionStamp(session),
		Subscription: &apipb.ResourceHandle{
			OpaqueToken: append([]byte(nil), token...),
			Kind:        apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION,
			Session:     cloneSessionStamp(session),
			Generation:  1,
		},
	}
	if event.GetTerminal() == nil {
		return nil
	}
	lifecycle := &apipb.TerminalLifecycleEvent{
		Terminal: TerminalInfoToAPI(endpointID, event.GetTerminal()),
	}
	if projection := event.GetAttachment(); projection != nil {
		lifecycle.AttachmentProjection = true
		lifecycle.ResizeControl = ResizeControlToAPI(projection.GetResizeControl())
		lifecycle.ResizeEpoch = projection.GetResizeEpoch()
	}
	envelope.Event = &apipb.EventEnvelope_TerminalLifecycle{TerminalLifecycle: lifecycle}
	return envelope
}

// EventMatchesFilter 报告 typed event 是否命中订阅的 terminal/type 过滤；
// 空 terminalID 匹配全部 terminal，空 types 匹配全部类型。
func EventMatchesFilter(terminalID string, types []providerv1.TerminalEventType, event *providerv1.TerminalEvent) bool {
	if event == nil {
		return false
	}
	if terminalID != "" && terminalID != event.GetTerminalId() {
		return false
	}
	if len(types) == 0 {
		return true
	}
	for _, typ := range types {
		if typ == event.GetType() {
			return true
		}
	}
	return false
}

// ProviderEventTypes 把 apipb 事件类型映射为 provider terminal 事件类型。
// 返回 terminalEvents=false 表示该订阅不包含 terminal 事件（例如 storage-only）。
func ProviderEventTypes(types []apipb.ApplicationEventType) ([]providerv1.TerminalEventType, bool) {
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

// SyntheticSubscriptionToken 给 access-local（storage-only）订阅生成不透明 token，
// 保持与 provider token 一致的不透明性；access 按 token 释放本地订阅。
func SyntheticSubscriptionToken() []byte {
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

func cloneSessionStamp(stamp *apipb.EndpointSessionStamp) *apipb.EndpointSessionStamp {
	if stamp == nil {
		return nil
	}
	return proto.Clone(stamp).(*apipb.EndpointSessionStamp)
}

package terminalmap

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/proto/access/apipb"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
)

// TestTerminalEventToEnvelopeKeepsLegacyShape 锁定客户端可见的 event envelope 形状：
// event id 前缀/时间戳、订阅 token、origin session、terminal 快照与 attachment 投影。
func TestTerminalEventToEnvelopeKeepsLegacyShape(t *testing.T) {
	session := &apipb.EndpointSessionStamp{EndpointId: "local", RouteId: "route-1", Generation: 3}
	token := []byte("ev-token")
	event := &providerv1.TerminalEvent{
		Type:              providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_CREATED,
		TerminalId:        "term-1",
		TimestampUnixNano: 12345,
		Terminal: &providerv1.TerminalInfo{
			TerminalId: "term-1", State: "running", Size: &providerv1.Size{Cols: 80, Rows: 24},
		},
		Attachment: &providerv1.TerminalAttachmentProjection{
			AttachmentCount: 2, ResizeEpoch: 7,
			ResizeControl: &providerv1.ResizeControl{CanResize: true, SizeLocked: true},
		},
	}
	envelope := TerminalEventToEnvelope("local", session, token, event)
	if envelope == nil {
		t.Fatal("event envelope is nil")
	}
	if !strings.HasPrefix(envelope.GetEventId(), "terminal.") || !strings.HasSuffix(envelope.GetEventId(), "-12345") {
		t.Fatalf("event id = %q", envelope.GetEventId())
	}
	if envelope.GetTimestampUnixNano() != 12345 || envelope.GetApiVersion().GetMajor() != 1 {
		t.Fatalf("envelope metadata = %#v", envelope)
	}
	if string(envelope.GetSubscription().GetOpaqueToken()) != string(token) ||
		envelope.GetSubscription().GetKind() != apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION {
		t.Fatalf("subscription handle = %#v", envelope.GetSubscription())
	}
	if envelope.GetOriginSession().GetRouteId() != "route-1" {
		t.Fatalf("origin session = %#v", envelope.GetOriginSession())
	}
	lifecycle := envelope.GetTerminalLifecycle()
	if lifecycle == nil || lifecycle.GetTerminal().GetRef().GetTerminalId() != "term-1" ||
		lifecycle.GetTerminal().GetRef().GetEndpointId() != "local" ||
		lifecycle.GetTerminal().GetState() != apipb.TerminalState_TERMINAL_STATE_RUNNING {
		t.Fatalf("lifecycle terminal = %#v", lifecycle)
	}
	if !lifecycle.GetAttachmentProjection() || lifecycle.GetResizeEpoch() != 7 ||
		!lifecycle.GetResizeControl().GetSizeLocked() {
		t.Fatalf("lifecycle attachment projection = %#v", lifecycle)
	}
}

// TestProviderEventTypesFiltersTerminalFace 锁定 apipb 事件类型 → provider 类型的
// 映射与 storage-only 判定。
func TestProviderEventTypesFiltersTerminalFace(t *testing.T) {
	types, terminalEvents := ProviderEventTypes(nil)
	if !terminalEvents || len(types) != 0 {
		t.Fatalf("empty types = %#v terminalEvents=%v", types, terminalEvents)
	}
	types, terminalEvents = ProviderEventTypes([]apipb.ApplicationEventType{
		apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE,
	})
	if !terminalEvents || len(types) != 5 {
		t.Fatalf("terminal types = %#v terminalEvents=%v", types, terminalEvents)
	}
	types, terminalEvents = ProviderEventTypes([]apipb.ApplicationEventType{
		apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_STORAGE_CHANGED,
	})
	if terminalEvents || len(types) != 0 {
		t.Fatalf("storage-only types = %#v terminalEvents=%v", types, terminalEvents)
	}
	if token := SyntheticSubscriptionToken(); len(token) < 2 || token[0] != 'e' || token[1] != 'v' {
		t.Fatalf("synthetic subscription token = %q", token)
	}
}

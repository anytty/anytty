package apimapping

import (
	"testing"
	"time"

	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

func TestEventFilterPreservesUnspecifiedStorageScope(t *testing.T) {
	filter := EventFilterFromProto(&apipb.EventSubscribeCommand{Types: []apipb.ApplicationEventType{
		apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE,
	}})
	if filter.StorageScope != "" || filter.StorageAppID != "" || filter.StorageOwnerID != "" || filter.StorageKeyPrefix != "" {
		t.Fatalf("terminal lifecycle filter gained storage constraints: %#v", filter)
	}
}

func TestEncodeEventEnvelopePreservesAttachmentCountAndResizeOwner(t *testing.T) {
	encoded, err := EncodeEventEnvelope("west", &apipb.EndpointSessionStamp{EndpointId: "west", RouteId: "local", Generation: 3}, []byte("subscription"), corev2.Event{
		Type:      corev2.EventTerminalMetadataChanged,
		Timestamp: time.Unix(12, 34).UTC(),
		Terminal: &corev2.TerminalInfo{
			ID: "term-1", Name: "shell", Command: []string{"sh"}, Size: corev2.Size{Cols: 80, Rows: 24}, State: corev2.TerminalStateRunning,
		},
		Attachment: &corev2.TerminalAttachmentProjection{
			AttachmentCount: 2,
			ResizeEpoch:     7,
			ResizeControl: &corev2.TerminalResizeControl{
				CanResize: true, Reason: corev2.TerminalResizeReasonOwner,
				SurfaceID: "surface-a", OwnerSurfaceID: "surface-a", OwnerViewID: "pane:two",
				ResizeOwnership: &corev2.TerminalResizeOwnership{
					OwnerAttachmentID: "attachment-2", OwnerSurfaceID: "surface-a", OwnerViewID: "pane:two",
					Size: corev2.Size{Cols: 80, Rows: 24}, Epoch: 7,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("encode event: %v", err)
	}
	var envelope apipb.EventEnvelope
	if err := proto.Unmarshal(encoded, &envelope); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	lifecycle := envelope.GetTerminalLifecycle()
	if lifecycle == nil || !lifecycle.GetAttachmentProjection() || lifecycle.GetTerminal().GetAttachmentCount() != 2 {
		t.Fatalf("attachment projection was lost: %#v", lifecycle)
	}
	if lifecycle.GetResizeEpoch() != 7 {
		t.Fatalf("attachment projection epoch was lost: %#v", lifecycle)
	}
	control := lifecycle.GetResizeControl()
	if control.GetOwnerSurfaceId() != "surface-a" || control.GetOwnerViewId() != "pane:two" || control.GetOwnership().GetEpoch() != 7 {
		t.Fatalf("resize owner projection was lost: %#v", control)
	}
}

func TestEncodeEventEnvelopePreservesOwnerlessProjection(t *testing.T) {
	encoded, err := EncodeEventEnvelope("west", &apipb.EndpointSessionStamp{EndpointId: "west"}, []byte("subscription"), corev2.Event{
		Type:      corev2.EventTerminalMetadataChanged,
		Timestamp: time.Unix(13, 0).UTC(),
		Terminal: &corev2.TerminalInfo{
			ID: "term-1", Name: "shell", Command: []string{"sh"}, Size: corev2.Size{Cols: 80, Rows: 24}, State: corev2.TerminalStateRunning,
		},
		Attachment: &corev2.TerminalAttachmentProjection{
			AttachmentCount: 1,
			ResizeEpoch:     8,
			ResizeControl:   &corev2.TerminalResizeControl{Reason: corev2.TerminalResizeReasonFollower},
		},
	})
	if err != nil {
		t.Fatalf("encode event: %v", err)
	}
	var envelope apipb.EventEnvelope
	if err := proto.Unmarshal(encoded, &envelope); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	lifecycle := envelope.GetTerminalLifecycle()
	if lifecycle.GetResizeEpoch() != 8 || lifecycle.GetResizeControl() == nil || lifecycle.GetResizeControl().GetReason() != apipb.ResizeControlReason_RESIZE_CONTROL_REASON_FOLLOWER || lifecycle.GetResizeControl().GetOwnership() != nil {
		t.Fatalf("ownerless projection was not encoded completely: %#v", lifecycle)
	}
}

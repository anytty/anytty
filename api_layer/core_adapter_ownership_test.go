package apilayer

import (
	"context"
	"testing"
	"time"

	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

type retentionTestApplicationPort struct {
	corev2.ApplicationSessionPort
	eventEncoder      corev2.ApplicationEventEncoder
	attachTransaction corev2.TerminalAttachmentTransaction
}

func (port *retentionTestApplicationPort) ApplicationEventSubscribe(_ context.Context, _ corev2.EventFilter, encoder corev2.ApplicationEventEncoder) ([]byte, error) {
	port.eventEncoder = encoder
	return []byte("event-token"), nil
}

func (port *retentionTestApplicationPort) ApplicationTerminalAttach(context.Context, corev2.TerminalAttachmentRequest) (corev2.TerminalAttachmentTransaction, error) {
	return port.attachTransaction, nil
}

type retentionTestAttachmentTransaction struct {
	result        corev2.TerminalAttachment
	resultCalls   int
	commitCalls   int
	rollbackCalls int
}

func (transaction *retentionTestAttachmentTransaction) Result() corev2.TerminalAttachment {
	transaction.resultCalls++
	return transaction.result
}

func (transaction *retentionTestAttachmentTransaction) Commit(context.Context) error {
	transaction.commitCalls++
	return nil
}

func (transaction *retentionTestAttachmentTransaction) Rollback(context.Context) error {
	transaction.rollbackCalls++
	return nil
}

func TestCoreApplicationAdapterEventEncoderOwnsOriginAfterSubscribe(t *testing.T) {
	port := &retentionTestApplicationPort{}
	adapter := &coreApplicationAdapter{port: port}
	origin := &apipb.EndpointSessionStamp{EndpointId: "endpoint-before", RouteId: "route-before", Generation: 7}
	result, err := adapter.EventSubscribe(context.Background(), origin, &apipb.EventSubscribeCommand{
		Types: []apipb.ApplicationEventType{apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_STORAGE_CHANGED},
	})
	if err != nil || result.GetSubscription().GetSession().GetGeneration() != 7 || port.eventEncoder == nil {
		t.Fatalf("subscribe result=%#v encoder=%v err=%v", result, port.eventEncoder != nil, err)
	}

	origin.EndpointId = "endpoint-after"
	origin.RouteId = "route-after"
	origin.Generation = 99

	type encodeResult struct {
		payload []byte
		err     error
	}
	encoded := make(chan encodeResult, 1)
	go func() {
		payload, encodeErr := port.eventEncoder(corev2.Event{
			Type: corev2.EventStorageChanged,
			Storage: &corev2.StorageChanged{
				AppID: "app", Scope: corev2.StorageScopePublic, Key: "key", Version: 11, Op: "put",
			},
			Timestamp: time.Unix(123, 456),
		}, []byte("async-token"))
		encoded <- encodeResult{payload: payload, err: encodeErr}
	}()

	var outcome encodeResult
	select {
	case outcome = <-encoded:
	case <-time.After(time.Second):
		t.Fatal("asynchronous event encode did not complete")
	}
	if outcome.err != nil {
		t.Fatal(outcome.err)
	}
	event := &apipb.EventEnvelope{}
	if err := proto.Unmarshal(outcome.payload, event); err != nil {
		t.Fatal(err)
	}
	if event.GetOriginSession().GetEndpointId() != "endpoint-before" ||
		event.GetOriginSession().GetRouteId() != "route-before" || event.GetOriginSession().GetGeneration() != 7 ||
		event.GetSubscription().GetSession().GetEndpointId() != "endpoint-before" ||
		event.GetStorageChanged().GetKey().GetAppId() != "app" {
		t.Fatalf("async event aliases borrowed origin: %#v", event)
	}
}

func TestCoreAttachmentTransactionOwnsProjectionInputsAndDelegatesRollback(t *testing.T) {
	coreTransaction := &retentionTestAttachmentTransaction{result: corev2.TerminalAttachment{
		Token: []byte("attachment-token"), TerminalID: "term-before",
		Mode: corev2.TerminalAttachmentModeCollaborator, ResizePolicy: corev2.TerminalResizePolicyOwner,
		SurfaceID: "surface-before", ViewID: "view-before", Size: corev2.Size{Cols: 80, Rows: 24},
	}}
	adapter := &coreApplicationAdapter{port: &retentionTestApplicationPort{attachTransaction: coreTransaction}}
	origin := &apipb.EndpointSessionStamp{EndpointId: "endpoint-before", RouteId: "route-before", Generation: 7}
	command := &apipb.TerminalAttachCommand{
		Terminal: &apipb.TerminalRef{EndpointId: "endpoint-before", TerminalId: "term-before"},
		Operation: &apipb.OperationStamp{
			Session:     &apipb.EndpointSessionStamp{EndpointId: "endpoint-before", RouteId: "route-before", Generation: 7},
			OperationId: "operation-before",
		},
		Mode: apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		SurfaceId: "surface-before", ViewId: "view-before",
	}
	transaction, err := adapter.TerminalAttach(context.Background(), origin, command)
	if err != nil {
		t.Fatal(err)
	}

	origin.EndpointId = "endpoint-after"
	origin.RouteId = "route-after"
	origin.Generation = 99
	command.Terminal.EndpointId = "endpoint-after"
	command.Terminal.TerminalId = "term-after"
	command.Operation.Session.EndpointId = "endpoint-after"
	command.Operation.Session.Generation = 99
	command.Operation.OperationId = "operation-after"
	command.Mode = apipb.AttachmentMode_ATTACHMENT_MODE_OBSERVER
	command.ResizePolicy = apipb.ResizePolicy_RESIZE_POLICY_OBSERVER
	command.SurfaceId = "surface-after"
	command.ViewId = "view-after"

	result := transaction.Result()
	attachment := result.GetAttachment()
	if attachment.GetResource().GetSession().GetEndpointId() != "endpoint-before" ||
		attachment.GetResource().GetSession().GetRouteId() != "route-before" || attachment.GetResource().GetSession().GetGeneration() != 7 ||
		attachment.GetTerminal().GetEndpointId() != "endpoint-before" || attachment.GetTerminal().GetTerminalId() != "term-before" ||
		attachment.GetOperation().GetOperationId() != "operation-before" || attachment.GetOperation().GetSession().GetGeneration() != 7 ||
		attachment.GetSurfaceId() != "surface-before" || attachment.GetViewId() != "view-before" ||
		result.GetMode() != apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR || result.GetResizePolicy() != apipb.ResizePolicy_RESIZE_POLICY_OWNER {
		t.Fatalf("attachment transaction aliases borrowed projection inputs: %#v", result)
	}
	if err := transaction.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if coreTransaction.resultCalls != 1 || coreTransaction.rollbackCalls != 1 || coreTransaction.commitCalls != 0 {
		t.Fatalf("core transaction calls: result=%d commit=%d rollback=%d", coreTransaction.resultCalls, coreTransaction.commitCalls, coreTransaction.rollbackCalls)
	}
}

package protocoladapter

import (
	"context"
	"errors"
	"testing"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/port"
	"google.golang.org/protobuf/proto"
)

func TestTerminalAdapterRejectsStaleInputBeforeAttachmentLookup(t *testing.T) {
	client := &stampedTerminalClient{}
	adapter, err := NewProtocolTerminalServiceAdapter(client, clientruntime.EndpointSessionStamp{EndpointID: "west", RouteID: "ssh", Generation: 8})
	if err != nil {
		t.Fatal(err)
	}
	err = adapter.SendInput(context.Background(), port.TerminalInputRequest{
		EndpointID: "west", TerminalID: "term-1", Channel: 7, OperationID: "input-1",
		Session: &apipb.EndpointSessionStamp{EndpointId: "west", RouteId: "ssh", Generation: 7}, Bytes: []byte("x"),
	})
	if clientruntime.CodeOf(err) != clientruntime.ErrorStaleSession || clientruntime.WasAttempted(err) {
		t.Fatalf("stale input error = %#v", err)
	}
	if client.attachmentCalls != 0 || client.executeCalls != 0 || client.validateCalls != 0 {
		t.Fatalf("stale stamp crossed adapter guard: validate=%d attachment=%d execute=%d", client.validateCalls, client.attachmentCalls, client.executeCalls)
	}
}

func TestTerminalAdapterRejectsStaleBoundOperationsBeforeAttachmentLookup(t *testing.T) {
	stale := &apipb.EndpointSessionStamp{EndpointId: "west", RouteId: "ssh", Generation: 7}
	tests := []struct {
		name string
		run  func(ProtocolTerminalServiceAdapter) error
	}{
		{name: "detach-cleanup", run: func(adapter ProtocolTerminalServiceAdapter) error {
			return adapter.Detach(context.Background(), port.TerminalDetachRequest{EndpointID: "west", TerminalID: "term-1", Channel: 7, Session: stale, OperationID: "cleanup:attach-1"})
		}},
		{name: "resize", run: func(adapter ProtocolTerminalServiceAdapter) error {
			_, err := adapter.Resize(context.Background(), port.TerminalResizeRequest{EndpointID: "west", TerminalID: "term-1", Channel: 7, Cols: 100, Rows: 30, Session: stale, OperationID: "resize-1"})
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &stampedTerminalClient{}
			adapter, err := NewProtocolTerminalServiceAdapter(client, clientruntime.EndpointSessionStamp{EndpointID: "west", RouteID: "ssh", Generation: 8})
			if err != nil {
				t.Fatal(err)
			}
			err = test.run(adapter)
			if clientruntime.CodeOf(err) != clientruntime.ErrorStaleSession || clientruntime.WasAttempted(err) {
				t.Fatalf("stale operation error = %#v", err)
			}
			if client.attachmentCalls != 0 || client.executeCalls != 0 || client.validateCalls != 0 {
				t.Fatalf("stale operation crossed adapter guard: validate=%d attachment=%d execute=%d", client.validateCalls, client.attachmentCalls, client.executeCalls)
			}
		})
	}
}

func TestTerminalAdapterRejectsReplacedGenerationBeforeAttach(t *testing.T) {
	client := &stampedTerminalClient{validateErr: &clientruntime.Error{Code: clientruntime.ErrorStaleSession, Message: "generation replaced", Attempted: false}}
	adapter, err := NewProtocolTerminalServiceAdapter(client, clientruntime.EndpointSessionStamp{EndpointID: "west", RouteID: "ssh", Generation: 8})
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.Attach(context.Background(), port.TerminalAttachRequest{EndpointID: "west", TerminalID: "term-1", OperationID: "attach-1"})
	if clientruntime.CodeOf(err) != clientruntime.ErrorStaleSession || clientruntime.WasAttempted(err) || !errors.Is(err, client.validateErr) {
		t.Fatalf("replaced attach error = %#v", err)
	}
	if client.validateCalls != 1 || client.attachmentCalls != 0 || client.executeCalls != 0 {
		t.Fatalf("replaced attach calls: validate=%d attachment=%d execute=%d", client.validateCalls, client.attachmentCalls, client.executeCalls)
	}
}

func TestTerminalAdapterCarriesInputOperationIdentityIntoProto(t *testing.T) {
	client := &stampedTerminalClient{}
	stamp := clientruntime.EndpointSessionStamp{EndpointID: "west", RouteID: "ssh", Generation: 8}
	adapter, err := NewProtocolTerminalServiceAdapter(client, stamp)
	if err != nil {
		t.Fatal(err)
	}
	err = adapter.SendInput(context.Background(), port.TerminalInputRequest{
		EndpointID: "west", TerminalID: "term-1", Channel: 7, OperationID: "paste:view-1:19",
		Session: &apipb.EndpointSessionStamp{EndpointId: "west", RouteId: "ssh", Generation: 8}, Bytes: []byte("payload"),
	})
	if err != nil {
		t.Fatal(err)
	}
	operation := client.command.GetTerminalInput().GetOperation()
	if operation.GetOperationId() != "paste:view-1:19" || operation.GetSession().GetEndpointId() != "west" || operation.GetSession().GetRouteId() != "ssh" || operation.GetSession().GetGeneration() != 8 {
		t.Fatalf("unexpected Proto operation %#v", operation)
	}
	if client.validateCalls != 1 || client.attachmentCalls != 1 || client.executeCalls != 1 {
		t.Fatalf("valid input calls: validate=%d attachment=%d execute=%d", client.validateCalls, client.attachmentCalls, client.executeCalls)
	}
}

type stampedTerminalClient struct {
	validateCalls   int
	attachmentCalls int
	executeCalls    int
	command         *apipb.CommandEnvelope
	validateErr     error
}

func (client *stampedTerminalClient) ValidateApplicationSession(clientruntime.EndpointSessionStamp) error {
	client.validateCalls++
	return client.validateErr
}

func (client *stampedTerminalClient) ExecuteApplication(_ context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	client.executeCalls++
	client.command = proto.Clone(command).(*apipb.CommandEnvelope)
	return &apipb.ResultEnvelope{
		RequestId: command.GetContext().GetRequestId(), OriginSession: proto.Clone(command.GetContext().GetSession()).(*apipb.EndpointSessionStamp),
		Result: &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}},
	}, nil
}

func (client *stampedTerminalClient) ApplicationAttachmentChannel(*apipb.ResourceHandle) (uint16, bool) {
	return 7, true
}

func (client *stampedTerminalClient) ApplicationAttachment(uint16) (*apipb.ResourceHandle, bool) {
	client.attachmentCalls++
	return &apipb.ResourceHandle{Kind: apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT}, true
}

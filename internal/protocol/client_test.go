package protocol

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport/memory"
	"google.golang.org/protobuf/proto"
)

func TestClientExecutesGeneratedApplicationEnvelope(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientTransport, serverTransport := memory.NewPair()
	defer clientTransport.Close()
	defer serverTransport.Close()

	serverDone := make(chan error, 1)
	go func() { serverDone <- serveApplicationList(serverTransport) }()
	client := NewClient(clientTransport)
	defer client.Close()
	if err := client.Hello(ctx, Hello{Version: wire.Version, Client: "test"}); err != nil {
		t.Fatal(err)
	}
	command := &apipb.CommandEnvelope{
		Context: &apipb.RequestContext{RequestId: "list-1", ApiVersion: &apipb.ApiVersion{Major: 1}, Session: testSessionStamp()},
		Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}},
	}
	result, err := client.ExecuteApplication(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.GetTerminalList().GetTerminals()[0].GetRef().GetTerminalId(); got != "term-1" {
		t.Fatalf("terminal id = %q", got)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
}

func TestClientRetainsLateFileOpenResultAfterContextCancellation(t *testing.T) {
	clientTransport, serverTransport := memory.NewPair()
	defer clientTransport.Close()
	defer serverTransport.Close()
	client := NewClient(clientTransport)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	helloDone := make(chan error, 1)
	go func() { helloDone <- client.Hello(ctx, Hello{Version: wire.Version, Client: "test"}) }()
	if err := expectHelloAndRespond(serverTransport); err != nil {
		t.Fatal(err)
	}
	if err := <-helloDone; err != nil {
		t.Fatal(err)
	}

	requestSeen := make(chan Request, 1)
	releaseResponse := make(chan struct{})
	serverDone := make(chan error, 1)
	go func() {
		channel, typ, payload, err := receiveTestFrame(serverTransport)
		if err != nil || channel != 0 || typ != wire.TypeRequest {
			serverDone <- fmt.Errorf("file request frame channel=%d type=%d err=%v", channel, typ, err)
			return
		}
		request, err := DecodeRequestPayload(payload)
		if err != nil {
			serverDone <- err
			return
		}
		requestSeen <- request
		<-releaseResponse
		var command apipb.CommandEnvelope
		if err := proto.Unmarshal(request.Params, &command); err != nil {
			serverDone <- err
			return
		}
		result, err := proto.Marshal(&apipb.ResultEnvelope{RequestId: command.GetContext().GetRequestId(), OriginSession: command.GetContext().GetSession(), Result: &apipb.ResultEnvelope_FileTransferOpen{FileTransferOpen: &apipb.FileTransferOpenResult{Transfer: &apipb.FileTransferHandle{
			Resource: &apipb.ResourceHandle{Kind: apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER, OpaqueToken: []byte("resource"), Session: command.GetContext().GetSession(), Generation: 1},
			Resume:   &apipb.FileUploadResumeHandle{OpaqueToken: []byte("resume")},
		}}}})
		if err != nil {
			serverDone <- err
			return
		}
		response, err := EncodeResponsePayload(Response{ID: request.ID, Result: result})
		if err == nil {
			err = sendTestFrame(serverTransport, 0, wire.TypeResponse, response)
		}
		serverDone <- err
	}()

	requestCtx, requestCancel := context.WithCancel(context.Background())
	resultCh := make(chan *apipb.ResultEnvelope, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := client.ExecuteApplicationTerminal(requestCtx, &apipb.CommandEnvelope{
			Context: &apipb.RequestContext{RequestId: "upload-open", ApiVersion: &apipb.ApiVersion{Major: 1}, Session: testSessionStamp()},
			Command: &apipb.CommandEnvelope_FileUploadOpen{FileUploadOpen: &apipb.FileUploadOpenCommand{Path: "/tmp/demo", Size: 8, Overwrite: true, Operation: &apipb.OperationStamp{Session: testSessionStamp(), OperationId: "upload-open"}}},
		})
		resultCh <- result
		errCh <- err
	}()
	<-requestSeen
	requestCancel()
	close(releaseResponse)
	if err := <-errCh; err != nil {
		t.Fatalf("late file result was discarded: %v", err)
	}
	if result := <-resultCh; string(result.GetFileTransferOpen().GetTransfer().GetResume().GetOpaqueToken()) != "resume" {
		t.Fatalf("late file result = %#v", result)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
}

func TestClientPublishesGeneratedApplicationEvents(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientTransport, serverTransport := memory.NewPair()
	defer clientTransport.Close()
	defer serverTransport.Close()
	client := NewClient(clientTransport)
	defer client.Close()

	helloDone := make(chan error, 1)
	go func() { helloDone <- client.Hello(ctx, Hello{Version: wire.Version, Client: "test"}) }()
	if err := expectHelloAndRespond(serverTransport); err != nil {
		t.Fatal(err)
	}
	if err := <-helloDone; err != nil {
		t.Fatal(err)
	}
	events, err := client.ApplicationEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	event := &apipb.EventEnvelope{Event: &apipb.EventEnvelope_TerminalLifecycle{TerminalLifecycle: &apipb.TerminalLifecycleEvent{
		Terminal: &apipb.TerminalInfo{Ref: &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-1"}},
	}}}
	payload, err := proto.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if err := sendTestFrame(serverTransport, 0, wire.TypeEvent, payload); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-events:
		if got.GetTerminalLifecycle().GetTerminal().GetRef().GetTerminalId() != "term-1" {
			t.Fatalf("unexpected event %#v", got)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestClientFailAllDoesNotBlockOnCompletedWaiter(t *testing.T) {
	completed := make(chan result, 1)
	completed <- result{payload: []byte("done")}
	client := &Client{
		waiters:                     map[uint64]*responseWaiter{1: {ch: completed, delivered: true}},
		abandonedWaiters:            make(map[uint64]struct{}),
		streams:                     make(map[uint16]*clientStream),
		pending:                     make(map[uint16][]StreamFrame),
		reused:                      make(map[uint16][]StreamFrame),
		dropped:                     make(map[uint16]struct{}),
		applicationEventSubscribers: make(map[uint64]chan *apipb.EventEnvelope),
		helloCh:                     make(chan result, 1),
	}
	done := make(chan struct{})
	go func() {
		client.failAll(io.EOF)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("failAll blocked behind a completed waiter")
	}
	if got := <-completed; string(got.payload) != "done" || got.err != nil {
		t.Fatalf("completed waiter result=%#v", got)
	}
}

func testSessionStamp() *apipb.EndpointSessionStamp {
	return &apipb.EndpointSessionStamp{EndpointId: "local", RouteId: "memory", Generation: 1}
}

func serveApplicationList(tr *memory.Transport) error {
	if err := expectHelloAndRespond(tr); err != nil {
		return err
	}
	channel, typ, payload, err := receiveTestFrame(tr)
	if err != nil || channel != 0 || typ != wire.TypeRequest {
		return fmt.Errorf("application request frame channel=%d type=%d err=%v", channel, typ, err)
	}
	request, err := DecodeRequestPayload(payload)
	if err != nil || request.Method != "api.execute" {
		return fmt.Errorf("application request method=%q err=%v", request.Method, err)
	}
	var command apipb.CommandEnvelope
	if err := proto.Unmarshal(request.Params, &command); err != nil || command.GetTerminalList() == nil {
		return fmt.Errorf("decode application command: %v", err)
	}
	result, err := proto.Marshal(&apipb.ResultEnvelope{RequestId: command.GetContext().GetRequestId(), OriginSession: command.GetContext().GetSession(), Result: &apipb.ResultEnvelope_TerminalList{TerminalList: &apipb.TerminalListResult{Terminals: []*apipb.TerminalInfo{{Ref: &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-1"}}}}}})
	if err != nil {
		return err
	}
	response, err := EncodeResponsePayload(Response{ID: request.ID, Result: result})
	if err != nil {
		return err
	}
	return sendTestFrame(tr, 0, wire.TypeResponse, response)
}

func expectHelloAndRespond(tr *memory.Transport) error {
	channel, typ, payload, err := receiveTestFrame(tr)
	if err != nil || channel != 0 || typ != wire.TypeHello {
		return fmt.Errorf("hello frame channel=%d type=%d err=%v", channel, typ, err)
	}
	if _, err := DecodeHelloPayload(payload); err != nil {
		return err
	}
	response, err := EncodeHelloPayload(Hello{Version: wire.Version, Server: "test"})
	if err != nil {
		return err
	}
	return sendTestFrame(tr, 0, wire.TypeHello, response)
}

func sendTestFrame(tr *memory.Transport, channel uint16, typ uint8, payload []byte) error {
	frame, err := wire.EncodeFrame(channel, typ, payload)
	if err != nil {
		return err
	}
	return tr.Send(frame)
}

func receiveTestFrame(tr *memory.Transport) (uint16, uint8, []byte, error) {
	frame, err := tr.Recv()
	if err != nil {
		return 0, 0, nil, err
	}
	return wire.DecodeFrame(frame)
}

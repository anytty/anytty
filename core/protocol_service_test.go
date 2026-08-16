package core

import (
	"context"
	"strings"
	"testing"
	"time"

	clientendpoint "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport/memory"
)

func TestProtocolServiceExecutesGeneratedTerminalAPI(t *testing.T) {
	server, application, _, closeClient := newApplicationProtocolHarness(t)
	defer closeClient()
	created, err := application.TerminalCreate(context.Background(), &apipb.TerminalCreateCommand{Terminal: &apipb.TerminalCreateSpec{
		TerminalId: "term-api", Name: "proto", Command: []string{"shell"}, Size: &apipb.TerminalSize{Cols: 12, Rows: 4},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if created.GetTerminal().GetRef().GetTerminalId() != "term-api" {
		t.Fatalf("unexpected create result %#v", created)
	}
	listed, err := application.TerminalList(context.Background(), &apipb.TerminalListCommand{})
	if err != nil || len(listed.GetTerminals()) != 1 {
		t.Fatalf("list=%#v err=%v", listed, err)
	}
	if err := server.IngestOutput(context.Background(), "term-api", "hello\r\n"); err != nil {
		t.Fatal(err)
	}
	got, err := application.TerminalGet(context.Background(), &apipb.TerminalGetCommand{Terminal: &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-api"}})
	if err != nil || got.GetTerminal().GetState() != apipb.TerminalState_TERMINAL_STATE_RUNNING {
		t.Fatalf("get=%#v err=%v", got, err)
	}
}

func TestProtocolServiceBindsAttachmentResourceToStreamAndRoutesInputResize(t *testing.T) {
	server, application, client, closeClient := newApplicationProtocolHarness(t)
	defer closeClient()
	if _, err := application.TerminalCreate(context.Background(), &apipb.TerminalCreateCommand{Terminal: &apipb.TerminalCreateSpec{
		TerminalId: "term-attach", Command: []string{"shell"}, Size: &apipb.TerminalSize{Cols: 12, Rows: 4},
	}}); err != nil {
		t.Fatal(err)
	}
	attached, err := application.TerminalAttach(context.Background(), &apipb.TerminalAttachCommand{
		Terminal: &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-attach"}, Mode: apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR,
		ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER, SurfaceId: "surface", ViewId: "view",
	})
	if err != nil {
		t.Fatal(err)
	}
	resource := attached.GetAttachment().GetResource()
	if _, ok := client.ApplicationAttachmentChannel(resource); !ok {
		t.Fatal("attachment resource was not bound to a stream channel")
	}
	if err := application.TerminalInput(context.Background(), &apipb.TerminalInputCommand{Attachment: resource, Data: []byte("pwd\n")}); err != nil {
		t.Fatal(err)
	}
	resized, err := application.TerminalResize(context.Background(), &apipb.TerminalResizeCommand{
		Attachment: resource, Size: &apipb.TerminalSize{Cols: 20, Rows: 5}, ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER,
	})
	if err != nil || resized.GetSize().GetCols() != 20 {
		t.Fatalf("resize=%#v err=%v", resized, err)
	}
	process := serverProcessForProtoTest(t, server, "term-attach")
	inputs, resizes, _, _ := process.snapshot()
	if len(inputs) == 0 || string(inputs[0]) != "pwd\n" || len(resizes) == 0 {
		t.Fatalf("inputs=%q resizes=%#v", inputs, resizes)
	}
}

func TestProtocolServiceReturnsTypedNotFoundError(t *testing.T) {
	_, application, _, closeClient := newApplicationProtocolHarness(t)
	defer closeClient()
	_, err := application.TerminalGet(context.Background(), &apipb.TerminalGetCommand{Terminal: &apipb.TerminalRef{EndpointId: "local", TerminalId: "missing"}})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing terminal error = %v", err)
	}
}

func newApplicationProtocolHarness(t *testing.T) (*Server, *clientruntime.ApplicationSession, *protocol.Client, func()) {
	t.Helper()
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	server.cfg.applicationFactory = applicationTestExecutorFactory
	application, client, closeClient := newApplicationProtocolClient(t, server, fullDaemonTransportScope(), 1)
	return server, application, client, closeClient
}

func newApplicationProtocolClient(t *testing.T, server *Server, scope TransportScope, generation uint64) (*clientruntime.ApplicationSession, *protocol.Client, func()) {
	t.Helper()
	clientTransport, serverTransport := memory.NewPair()
	errCh := make(chan error, 1)
	go func() {
		errCh <- newProtocolSession(server, serverTransport, scope).run(context.Background())
	}()
	client := protocol.NewClient(clientTransport)
	if err := client.Hello(context.Background(), protocol.Hello{Version: wire.Version, Client: "test"}); err != nil {
		t.Fatal(err)
	}
	application, err := clientruntime.NewApplicationSession(clientruntime.EndpointSessionStamp{
		EndpointID: clientendpoint.EndpointID("local"), RouteID: clientendpoint.RouteID("memory"), Generation: clientruntime.SessionGeneration(generation),
	}, client)
	if err != nil {
		t.Fatal(err)
	}
	return application, client, func() {
		_ = client.Close()
		select {
		case err := <-errCh:
			if err != nil && !strings.Contains(err.Error(), "EOF") {
				t.Fatalf("server session returned error: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("server session did not stop")
		}
	}
}

func serverProcessForProtoTest(t *testing.T, server *Server, terminalID string) *recordingProcess {
	t.Helper()
	terminal, err := server.Terminal(terminalID)
	if err != nil {
		t.Fatal(err)
	}
	process, ok := terminal.process.(*recordingProcess)
	if !ok {
		t.Fatalf("process type %T", terminal.process)
	}
	return process
}

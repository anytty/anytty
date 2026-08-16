package apilayer

import (
	"context"
	"strings"
	"testing"
	"time"

	clientendpoint "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport/memory"
)

func TestCoreApplicationAdapterRunsTerminalProtoE2E(t *testing.T) {
	clientTransport, serverTransport := memory.NewPair()
	server := corev2.NewServer(corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory))
	errCh := make(chan error, 1)
	go func() { errCh <- server.ServeTransport(context.Background(), serverTransport) }()
	client := protocol.NewClient(clientTransport)
	defer func() {
		_ = client.Close()
		select {
		case err := <-errCh:
			if err != nil && !strings.Contains(err.Error(), "EOF") {
				t.Fatalf("serve transport: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("server transport did not stop")
		}
	}()
	if err := client.Hello(context.Background(), protocol.Hello{Version: wire.Version, Client: "api-layer-core-e2e"}); err != nil {
		t.Fatal(err)
	}
	application, err := clientruntime.NewApplicationSession(clientruntime.EndpointSessionStamp{EndpointID: clientendpoint.EndpointID("local"), RouteID: clientendpoint.RouteID("memory"), Generation: 1}, client)
	if err != nil {
		t.Fatal(err)
	}
	created, err := application.TerminalCreate(context.Background(), &apipb.TerminalCreateCommand{Terminal: &apipb.TerminalCreateSpec{TerminalId: "term-e2e", Command: testIdleTerminalCommand(), Size: &apipb.TerminalSize{Cols: 80, Rows: 24}}})
	if err != nil {
		t.Fatal(err)
	}
	if created.GetTerminal().GetRef().GetTerminalId() != "term-e2e" {
		t.Fatalf("created=%#v", created)
	}
	attached, err := application.TerminalAttach(context.Background(), &apipb.TerminalAttachCommand{Terminal: &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-e2e"}, Mode: apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER, SurfaceId: "surface", ViewId: "view"})
	if err != nil {
		t.Fatal(err)
	}
	resource := attached.GetAttachment().GetResource()
	if len(resource.GetOpaqueToken()) == 0 || resource.GetSession().GetGeneration() != 1 {
		t.Fatalf("attachment=%#v", attached)
	}
	if err := application.TerminalDetach(context.Background(), &apipb.TerminalDetachCommand{Attachment: resource}); err != nil {
		t.Fatal(err)
	}
}

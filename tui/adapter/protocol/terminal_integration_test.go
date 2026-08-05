package protocoladapter

import (
	"context"
	"io"
	"testing"
	"time"

	apilayer "github.com/anytty/anytty/api_layer"
	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport/memory"
	. "github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

func TestProtocolTerminalServiceAdapterWithRealProtocolClient(t *testing.T) {
	clientTransport, serverTransport := memory.NewPair()
	server := corev2.NewServer(corev2.WithApplicationExecutorFactory(apilayer.CoreApplicationExecutorFactory))
	if _, err := server.RegisterTerminal(corev2.TerminalRecord{ID: "term-1", Command: testIdleTerminalCommand(), Size: corev2.Size{Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	errCh := make(chan error, 1)
	go func() { errCh <- server.ServeTransport(context.Background(), serverTransport) }()
	client := protocol.NewClient(clientTransport)
	defer func() { _ = client.Close() }()
	if err := client.Hello(context.Background(), protocol.Hello{Version: wire.Version, Client: "tui-v3-test"}); err != nil {
		t.Fatalf("hello: %v", err)
	}

	application, err := clientruntime.NewApplicationSession(clientruntime.EndpointSessionStamp{EndpointID: endpoint.EndpointID("local"), RouteID: endpoint.RouteID("memory"), Generation: 1}, client)
	if err != nil {
		t.Fatal(err)
	}
	adapter := ProtocolTerminalServiceAdapter{Client: client, Application: application}
	attached, err := adapter.Attach(context.Background(), TerminalAttachRequest{EndpointID: "local", TerminalID: "term-1", Cols: 80, Rows: 24, ResizePolicy: state.TerminalResizeRoleOwner, SurfaceID: "surface-1", ViewID: "view-1", OperationID: "attach-1"})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	if attached.Channel == 0 || !attached.CanResize {
		t.Fatalf("unexpected attach result %#v", attached)
	}
	if err := adapter.SendInput(context.Background(), TerminalInputRequest{EndpointID: "local", TerminalID: "term-1", Channel: attached.Channel, SurfaceID: "surface-1", ViewID: "view-1", Session: attached.Session, OperationID: "input-1", Bytes: []byte("x")}); err != nil {
		t.Fatalf("input: %v", err)
	}
	if _, err := adapter.Resize(context.Background(), TerminalResizeRequest{EndpointID: "local", TerminalID: "term-1", Channel: attached.Channel, Cols: 100, Rows: 40, SurfaceID: "surface-1", ViewID: "view-1", Session: attached.Session, OperationID: "resize-1"}); err != nil {
		t.Fatalf("resize: %v", err)
	}
	info, err := server.GetTerminal("term-1")
	if err != nil {
		t.Fatal(err)
	}
	if info.Size != (corev2.Size{Cols: 100, Rows: 40}) {
		t.Fatalf("unexpected resize size %#v", info.Size)
	}
	_ = clientTransport.Close()
	select {
	case err := <-errCh:
		if err != nil && err != io.EOF {
			t.Fatalf("server returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop")
	}
}

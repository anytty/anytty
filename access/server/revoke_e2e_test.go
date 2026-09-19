package server_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	daemonprovider "github.com/anytty/anytty/access/provider/daemon"
	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	"github.com/anytty/anytty/access/provider/terminal/tmux"
	accessserver "github.com/anytty/anytty/access/server"
	"github.com/anytty/anytty/access/sessions"
	corev2 "github.com/anytty/anytty/daemon/core"
	providercore "github.com/anytty/anytty/daemon/provider"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
	"github.com/anytty/anytty/shared/transport/memory"
)

// TestGrantRevokeClosesProviderSession 覆盖 Phase 1 kick 契约：
// remoteauth 会话登记后，CloseGrant 关闭客户端 transport，access session 退出，
// provider 连接与附件桥接随之释放。
func TestGrantRevokeClosesProviderSession(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	daemonSocket := filepath.Join(dir, "anyttyd.sock")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := corev2.NewServer(
		corev2.WithSocketPath(daemonSocket),
		corev2.WithHistoryDisabled(),
	)
	defer func() { _ = server.Shutdown(context.Background()) }()
	providerServer, err := providercore.New(server, providercore.Config{Socket: daemonSocket})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = providerServer.Shutdown(context.Background()) }()
	go func() { _ = providerServer.ListenAndServe(ctx) }()
	waitForSocket(t, daemonSocket)

	registry := sessions.NewRegistry()
	defer registry.CloseAll()
	access, err := accessserver.New(accessserver.Config{
		Socket:  filepath.Join(dir, "unused.sock"),
		Files:   accessserverTestFiles(t),
		Tracker: registry,
		Provider: func(dialCtx context.Context) (terminalprovider.Provider, error) {
			return daemonprovider.DialTerminal(dialCtx, daemonSocket)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = access.Close() }()

	clientTransport, serverTransport := memory.NewPair()
	sessionCtx := sessions.WithRemoteSessionInfo(ctx, sessions.RemoteSessionInfo{
		GrantID: "grant-e2e", ExpiresAt: time.Now().Add(time.Hour),
	})
	sessionDone := make(chan error, 1)
	go func() { sessionDone <- access.ServeTransport(sessionCtx, serverTransport) }()

	client := internalprotocol.NewClient(clientTransport)
	defer func() { _ = client.Close() }()
	if err := client.Hello(ctx, internalprotocol.Hello{Version: wire.Version, Client: "revoke-e2e"}); err != nil {
		t.Fatal(err)
	}
	application, err := clientruntime.NewApplicationSession(clientruntime.EndpointSessionStamp{
		EndpointID: clientendpoint.EndpointID("local"),
		RouteID:    clientendpoint.RouteID("access-revoke-test"),
		Generation: 1,
	}, client)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.TerminalCreate(ctx, &apipb.TerminalCreateCommand{Terminal: &apipb.TerminalCreateSpec{
		TerminalId: "term-revoke", Command: []string{"/bin/cat"}, Size: &apipb.TerminalSize{Cols: 20, Rows: 5},
	}}); err != nil {
		t.Fatal(err)
	}
	attached, err := application.TerminalAttach(ctx, &apipb.TerminalAttachCommand{
		Terminal:     &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-revoke"},
		Mode:         apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR,
		ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		SurfaceId:    "surface-revoke",
		ViewId:       "view-revoke",
	})
	if err != nil {
		t.Fatal(err)
	}
	channel, ok := client.ApplicationAttachmentChannel(attached.GetAttachment().GetResource())
	if !ok {
		t.Fatal("attachment resource was not bound")
	}
	frames, stopStream := client.Stream(channel)
	defer stopStream()
	if err := client.SendAttachmentReady(channel); err != nil {
		t.Fatal(err)
	}
	waitForStreamReady(t, frames)

	if closed := registry.CloseGrant("grant-e2e"); closed != 1 {
		t.Fatalf("CloseGrant closed %d sessions, want 1", closed)
	}
	select {
	case <-sessionDone:
	case <-time.After(5 * time.Second):
		t.Fatal("access session did not stop after grant revoke")
	}
	waitForStreamClosed(t, frames)
}

func TestTmuxProviderReturnsUnsupported(t *testing.T) {
	t.Parallel()
	provider := tmux.Unsupported()
	if _, err := provider.Execute(context.Background(), &apipb.CommandEnvelope{}); err != terminalprovider.ErrUnsupported {
		t.Fatalf("tmux execute error = %v, want ErrUnsupported", err)
	}
	if _, err := provider.OpenStream(context.Background(), &apipb.ResourceHandle{}); err != terminalprovider.ErrUnsupported {
		t.Fatalf("tmux open stream error = %v, want ErrUnsupported", err)
	}
	if _, err := provider.Events(context.Background()); err != terminalprovider.ErrUnsupported {
		t.Fatalf("tmux events error = %v, want ErrUnsupported", err)
	}
}

func waitForStreamClosed(t *testing.T, frames <-chan internalprotocol.StreamFrame) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case frame, ok := <-frames:
			if !ok {
				return
			}
			if frame.Type == wire.TypeClosed || frame.Type == wire.TypeSyncLost {
				return
			}
		case <-deadline:
			t.Fatal("attachment stream did not close after grant revoke")
		}
	}
}

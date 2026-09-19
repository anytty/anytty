package server_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	"github.com/anytty/anytty/access/files"
	daemonprovider "github.com/anytty/anytty/access/provider/daemon"
	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	accessserver "github.com/anytty/anytty/access/server"
	corev2 "github.com/anytty/anytty/daemon/core"
	providercore "github.com/anytty/anytty/daemon/provider"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
)

// TestTerminalRoutingThroughAccessReachesDaemonPTY 覆盖 Phase 1 e2e：
// client → access(.access) → provider → daemon PTY，包含 create/list/attach/input/detach/kill。
func TestTerminalRoutingThroughAccessReachesDaemonPTY(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	daemonSocket := filepath.Join(dir, "anyttyd.sock")
	accessSocket := filepath.Join(dir, "anyttyd.sock.access")

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

	access, err := accessserver.New(accessserver.Config{
		Socket: accessSocket,
		Files:  accessserverTestFiles(t),
		Provider: func(dialCtx context.Context) (terminalprovider.Provider, error) {
			return daemonprovider.DialTerminal(dialCtx, daemonSocket)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = access.Close() }()
	go func() { _ = access.Serve(ctx) }()
	waitForSocket(t, accessSocket)

	connection, err := unixtransport.DialContext(ctx, accessSocket)
	if err != nil {
		t.Fatal(err)
	}
	client := internalprotocol.NewClient(connection)
	defer func() { _ = client.Close() }()
	if err := client.Hello(ctx, internalprotocol.Hello{Version: wire.Version, Client: "access-e2e"}); err != nil {
		t.Fatal(err)
	}
	application, err := clientruntime.NewApplicationSession(clientruntime.EndpointSessionStamp{
		EndpointID: clientendpoint.EndpointID("local"),
		RouteID:    clientendpoint.RouteID("access-test"),
		Generation: 1,
	}, client)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := application.TerminalCreate(ctx, &apipb.TerminalCreateCommand{Terminal: &apipb.TerminalCreateSpec{
		TerminalId: "term-e2e", Command: []string{"/bin/cat"}, Size: &apipb.TerminalSize{Cols: 20, Rows: 5},
	}}); err != nil {
		t.Fatalf("terminal create through access: %v", err)
	}
	list, err := application.TerminalList(ctx, &apipb.TerminalListCommand{})
	if err != nil {
		t.Fatalf("terminal list through access: %v", err)
	}
	if len(list.GetTerminals()) != 1 || list.GetTerminals()[0].GetRef().GetTerminalId() != "term-e2e" {
		t.Fatalf("terminal list projection = %#v", list.GetTerminals())
	}

	attached, err := application.TerminalAttach(ctx, &apipb.TerminalAttachCommand{
		Terminal:     &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-e2e"},
		Mode:         apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR,
		ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		SurfaceId:    "surface-e2e",
		ViewId:       "view-e2e",
	})
	if err != nil {
		t.Fatalf("terminal attach through access: %v", err)
	}
	attachment := attached.GetAttachment()
	if attachment.GetResource().GetKind() != apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT {
		t.Fatalf("attachment resource = %#v", attachment.GetResource())
	}
	channel, ok := client.ApplicationAttachmentChannel(attachment.GetResource())
	if !ok {
		t.Fatal("access did not bind attachment resource to a client channel")
	}
	frames, stopStream := client.Stream(channel)
	defer stopStream()
	if err := client.SendAttachmentReady(channel); err != nil {
		t.Fatal(err)
	}
	waitForStreamReady(t, frames)

	payload := []byte("access-terminal-e2e\n")
	if err := application.TerminalInput(ctx, &apipb.TerminalInputCommand{Attachment: attachment.GetResource(), Data: payload}); err != nil {
		t.Fatalf("terminal input through access: %v", err)
	}
	if output := waitForPTYOutput(t, frames, "access-terminal-e2e"); output == "" {
		t.Fatal("PTY output did not reach client through access bridge")
	}

	if err := application.TerminalDetach(ctx, &apipb.TerminalDetachCommand{Attachment: attachment.GetResource()}); err != nil {
		t.Fatalf("terminal detach through access: %v", err)
	}
	terminalRef := &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-e2e"}
	if err := application.TerminalKill(ctx, &apipb.TerminalKillCommand{Terminal: terminalRef}); err != nil {
		t.Fatalf("terminal kill through access: %v", err)
	}
	if err := application.TerminalRemove(ctx, &apipb.TerminalRemoveCommand{Terminal: terminalRef}); err != nil {
		t.Fatalf("terminal remove through access: %v", err)
	}
	empty, err := application.TerminalList(ctx, &apipb.TerminalListCommand{})
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.GetTerminals()) != 0 {
		t.Fatalf("terminal list after remove = %#v", empty.GetTerminals())
	}
}

func waitForSocket(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("socket %s did not appear", path)
}

func waitForStreamReady(t *testing.T, frames <-chan internalprotocol.StreamFrame) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case frame, ok := <-frames:
			if !ok {
				t.Fatal("attachment stream closed before ready")
			}
			if frame.Type == wire.TypeStreamReady {
				return
			}
			if frame.Type == wire.TypeError {
				t.Fatalf("attachment stream error: %s", string(frame.Payload))
			}
		case <-deadline:
			t.Fatal("timed out waiting for attachment stream ready")
		}
	}
}

func waitForPTYOutput(t *testing.T, frames <-chan internalprotocol.StreamFrame, needle string) string {
	t.Helper()
	deadline := time.After(5 * time.Second)
	sb := &strings.Builder{}
	for {
		select {
		case frame, ok := <-frames:
			if !ok {
				t.Fatal("attachment stream closed before PTY output")
			}
			if frame.Type == wire.TypePTYOutput {
				sb.Write(frame.Payload)
				if strings.Contains(sb.String(), needle) {
					return sb.String()
				}
			}
		case <-deadline:
			t.Fatalf("timed out waiting for PTY output %q; got %q", needle, sb.String())
		}
	}
}

// accessserverTestFiles 给测试注入临时 transfer 目录，避免污染用户 state 目录。
func accessserverTestFiles(t *testing.T) files.Config {
	t.Helper()
	return files.Config{TransferDir: filepath.Join(t.TempDir(), "transfers")}
}

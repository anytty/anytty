package provider_test

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/internal/providerproto"
	"github.com/anytty/anytty/pool/core"
	"github.com/anytty/anytty/pool/provider"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
)

func startProvider(t *testing.T) (*core.Server, *provider.Server, string) {
	t.Helper()
	coreServer := core.NewServer(core.WithHistoryStorageDir(t.TempDir()))
	t.Cleanup(func() { _ = coreServer.Shutdown(context.Background()) })
	socketPath := filepath.Join(t.TempDir(), "provider.sock")
	server, err := provider.New(coreServer, provider.Config{Socket: socketPath})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); _ = server.Shutdown(context.Background()) })
	go func() { _ = server.ListenAndServe(ctx) }()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			return coreServer, server, socketPath
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("provider socket did not appear")
	return nil, nil, ""
}

// TestProviderTerminalLifecycle 覆盖 Slice A1 的全部 terminal lifecycle 操作。
func TestProviderTerminalLifecycle(t *testing.T) {
	t.Parallel()
	_, _, socketPath := startProvider(t)
	ctx := context.Background()
	client, err := provider.Dial(ctx, socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	created, err := client.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-a1", Name: "provider-a1", Command: []string{"/bin/cat"},
		Size: &providerv1.Size{Cols: 40, Rows: 10}, Tags: map[string]string{"role": "test"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.GetTerminalId() != "term-a1" || created.GetName() != "provider-a1" || created.GetSize().GetCols() != 40 {
		t.Fatalf("created = %#v", created)
	}
	if _, err := client.Create(ctx, &providerv1.TerminalCreateSpec{TerminalId: "term-a1", Command: []string{"/bin/cat"}}); providerCode(err) != providerproto.ErrorConflict {
		t.Fatalf("duplicate create err = %v", err)
	}
	if _, err := client.Create(ctx, &providerv1.TerminalCreateSpec{Command: []string{"/bin/cat"}}); providerCode(err) != providerproto.ErrorBadRequest {
		t.Fatalf("missing id err = %v", err)
	}

	list, err := client.List(ctx)
	if err != nil || len(list) != 1 || list[0].GetTerminalId() != "term-a1" {
		t.Fatalf("list = %#v err=%v", list, err)
	}
	got, err := client.Get(ctx, "term-a1")
	if err != nil || got.GetName() != "provider-a1" || got.GetTags()["role"] != "test" {
		t.Fatalf("get = %#v err=%v", got, err)
	}

	if err := client.SetMetadata(ctx, "term-a1", "renamed-a1", map[string]string{"role": "renamed"}); err != nil {
		t.Fatal(err)
	}
	if got, err = client.Get(ctx, "term-a1"); err != nil || got.GetName() != "renamed-a1" || got.GetTags()["role"] != "renamed" {
		t.Fatalf("after set metadata = %#v err=%v", got, err)
	}
	if err := client.SetTags(ctx, "term-a1", map[string]string{"only": "tags"}); err != nil {
		t.Fatal(err)
	}
	if got, err = client.Get(ctx, "term-a1"); err != nil || got.GetName() != "renamed-a1" || got.GetTags()["only"] != "tags" {
		t.Fatalf("after set tags = %#v err=%v", got, err)
	}

	if err := client.Restart(ctx, "term-a1"); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if err := client.Kill(ctx, "term-a1"); err != nil {
		t.Fatalf("kill: %v", err)
	}
	if err := client.Remove(ctx, "term-a1"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := client.Get(ctx, "term-a1"); providerCode(err) != providerproto.ErrorNotFound {
		t.Fatalf("get removed err = %v", err)
	}
	if err := client.Kill(ctx, "term-a1"); providerCode(err) != providerproto.ErrorNotFound {
		t.Fatalf("kill removed err = %v", err)
	}
}

// TestProviderDefaultsAndPathDirectories 覆盖 path.defaults / path.list_directories。
func TestProviderDefaultsAndPathDirectories(t *testing.T) {
	t.Parallel()
	_, _, socketPath := startProvider(t)
	ctx := context.Background()
	client, err := provider.Dial(ctx, socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	defaults, err := client.TerminalDefaults(ctx)
	if err != nil || defaults.GetCwd() == "" || len(defaults.GetCommand()) == 0 || defaults.GetPlatform() == "" {
		t.Fatalf("defaults = %#v err=%v", defaults, err)
	}
	base := t.TempDir()
	for _, name := range []string{"alpha", "beta"} {
		if err := os.Mkdir(filepath.Join(base, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	directories, err := client.ListDirectories(ctx, base+string(filepath.Separator), 10)
	if err != nil || len(directories.GetEntries()) != 2 {
		t.Fatalf("directories = %#v err=%v", directories, err)
	}
	missing, err := client.ListDirectories(ctx, filepath.Join(base, "missing")+string(filepath.Separator), 10)
	if err != nil || !missing.GetMissing() {
		t.Fatalf("missing directories = %#v err=%v", missing, err)
	}
}

// TestProviderRejectsBadHandshakeAndUnknownCommand 覆盖协议层错误。
func TestProviderRejectsBadHandshakeAndUnknownCommand(t *testing.T) {
	t.Parallel()
	_, _, socketPath := startProvider(t)
	ctx := context.Background()

	// 版本不匹配的 Hello 必须得到 typed bad request。
	badConn, err := unixtransport.DialContext(ctx, socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer badConn.Close()
	helloPayload, err := providerproto.EncodeHelloPayload(&providerv1.Hello{Version: 99})
	if err != nil {
		t.Fatal(err)
	}
	if err := sendRawFrame(badConn, 0, providerproto.TypeHello, helloPayload); err != nil {
		t.Fatal(err)
	}
	response := recvRawResponse(t, badConn)
	if response.GetError().GetCode() != providerproto.ErrorBadRequest {
		t.Fatalf("bad hello response = %#v", response)
	}

	// Hello 之前的 request 必须 fail closed。
	earlyConn, err := unixtransport.DialContext(ctx, socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer earlyConn.Close()
	requestPayload, err := providerproto.EncodeRequestPayload(&providerv1.Request{
		Id: 1, Command: &providerv1.Request_TerminalList{TerminalList: &providerv1.TerminalListCommand{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sendRawFrame(earlyConn, 0, providerproto.TypeRequest, requestPayload); err != nil {
		t.Fatal(err)
	}
	response = recvRawResponse(t, earlyConn)
	if response.GetError().GetCode() != providerproto.ErrorBadRequest {
		t.Fatalf("pre-hello request response = %#v", response)
	}

	// 已完成 Hello 的连接上，缺少 id/command 的请求必须得到 typed bad request。
	goodConn, err := unixtransport.DialContext(ctx, socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer goodConn.Close()
	helloPayload, err = providerproto.EncodeHelloPayload(&providerv1.Hello{Version: providerproto.Version})
	if err != nil {
		t.Fatal(err)
	}
	if err := sendRawFrame(goodConn, 0, providerproto.TypeHello, helloPayload); err != nil {
		t.Fatal(err)
	}
	if _, typ, _, err := recvRawFrame(goodConn); err != nil || typ != providerproto.TypeHello {
		t.Fatalf("hello ack type=%d err=%v", typ, err)
	}
	malformed, err := providerproto.EncodeRequestPayload(&providerv1.Request{Id: 7})
	if err != nil {
		t.Fatal(err)
	}
	if err := sendRawFrame(goodConn, 0, providerproto.TypeRequest, malformed); err != nil {
		t.Fatal(err)
	}
	response = recvRawResponse(t, goodConn)
	if response.GetId() != 7 || response.GetError().GetCode() != providerproto.ErrorBadRequest {
		t.Fatalf("missing command response = %#v", response)
	}
}

func sendRawFrame(connection *unixtransport.Transport, channel uint16, typ uint8, payload []byte) error {
	frame, err := providerproto.EncodeFrame(channel, typ, payload)
	if err != nil {
		return err
	}
	return connection.Send(frame)
}

func recvRawResponse(t *testing.T, connection *unixtransport.Transport) *providerv1.Response {
	t.Helper()
	_, typ, payload, err := recvRawFrame(connection)
	if err != nil {
		t.Fatal(err)
	}
	if typ != providerproto.TypeResponse {
		t.Fatalf("frame type = %d, want response", typ)
	}
	response, err := providerproto.DecodeResponsePayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func recvRawFrame(connection *unixtransport.Transport) (uint16, uint8, []byte, error) {
	raw, err := connection.Recv()
	if err != nil {
		return 0, 0, nil, err
	}
	return providerproto.DecodeFrame(raw)
}

func providerCode(err error) uint32 {
	var providerErr *provider.ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Code
	}
	return 0
}

// TestProviderAttachmentStreamInputAndResize 覆盖 Slice A2：
// attach stream bootstrap/ready、input、PTY 输出桥接、resize ownership、detach。
func TestProviderAttachmentStreamInputAndResize(t *testing.T) {
	t.Parallel()
	_, _, socketPath := startProvider(t)
	ctx := context.Background()
	client, err := provider.Dial(ctx, socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if _, err := client.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-attach", Command: []string{"/bin/cat"}, Size: &providerv1.Size{Cols: 20, Rows: 5},
	}); err != nil {
		t.Fatal(err)
	}
	attached, err := client.Attach(ctx, &providerv1.TerminalAttachCommand{
		Terminal:     &providerv1.TerminalRef{TerminalId: "term-attach"},
		Mode:         providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR,
		ResizePolicy: providerv1.ResizePolicy_RESIZE_POLICY_OWNER,
		SurfaceId:    "surface-a2", ViewId: "view-a2",
	})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	token := attached.GetAttachment().GetOpaqueToken()
	if len(token) < 2 || attached.GetResizeControl().GetReason() != providerv1.ResizeReason_RESIZE_REASON_OWNER {
		t.Fatalf("attach result = %#v", attached)
	}
	channel := binary.BigEndian.Uint16(token[:2])
	stream, stop := client.Stream(channel)
	defer stop()
	if err := client.SendBootstrapDone(channel); err != nil {
		t.Fatal(err)
	}
	waitForStreamFrame(t, stream, providerproto.StreamReady, 5*time.Second)

	if err := client.Input(ctx, token, []byte("provider-a2-echo\n")); err != nil {
		t.Fatalf("input: %v", err)
	}
	var output string
	deadline := time.After(5 * time.Second)
	for !strings.Contains(output, "provider-a2-echo") {
		select {
		case frame := <-stream:
			if frame.Type == providerproto.StreamPTYOutput {
				output += string(frame.Payload)
			}
		case <-deadline:
			t.Fatalf("timed out waiting for PTY output, got %q", output)
		}
	}

	resized, err := client.Resize(ctx, token, &providerv1.Size{Cols: 30, Rows: 8}, providerv1.ResizePolicy_RESIZE_POLICY_OWNER, false, 0)
	if err != nil {
		t.Fatalf("resize: %v", err)
	}
	if !resized.GetResized() || !resized.GetResizeControl().GetCanResize() {
		t.Fatalf("resize result = %#v", resized)
	}
	info, err := client.Get(ctx, "term-attach")
	if err != nil || info.GetSize().GetCols() != 30 || info.GetSize().GetRows() != 8 {
		t.Fatalf("resized info = %#v err=%v", info, err)
	}

	if err := client.Detach(ctx, token); err != nil {
		t.Fatalf("detach: %v", err)
	}
	if _, err := client.Resize(ctx, token, &providerv1.Size{Cols: 40, Rows: 9}, providerv1.ResizePolicy_RESIZE_POLICY_OWNER, false, 0); providerCode(err) != providerproto.ErrorNotFound {
		t.Fatalf("resize after detach err = %v", err)
	}
}

func waitForStreamFrame(t *testing.T, stream <-chan provider.StreamFrame, want uint8, timeout time.Duration) provider.StreamFrame {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case frame := <-stream:
			if frame.Type == want {
				return frame
			}
		case <-deadline:
			t.Fatalf("timed out waiting for stream frame type %d", want)
		}
	}
}

// TestProviderHistoryLifecycle 覆盖 A3 history window/copy/release/backlog 与 token 生命周期。
func TestProviderHistoryLifecycle(t *testing.T) {
	t.Parallel()
	_, _, socketPath := startProvider(t)
	ctx := context.Background()
	client, err := provider.Dial(ctx, socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if _, err := client.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-history", Command: []string{"/bin/sh", "-c", "printf 'history-alpha\\nhistory-beta\\n'; sleep 30"},
		Size: &providerv1.Size{Cols: 80, Rows: 24},
	}); err != nil {
		t.Fatal(err)
	}
	window, err := client.HistoryWindow(ctx, &providerv1.HistoryWindowCommand{
		Terminal: &providerv1.TerminalRef{TerminalId: "term-history"},
		Mode:     providerv1.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST,
		Cols:     80, Limit: 50,
	})
	if err != nil {
		t.Fatalf("history window: %v", err)
	}
	token := window.GetToken()
	if token == "" || len(window.GetRows()) == 0 {
		t.Fatalf("history window = %#v", window)
	}
	copied, err := client.HistoryCopy(ctx, &providerv1.HistoryCopyCommand{
		Terminal: &providerv1.TerminalRef{TerminalId: "term-history"},
		Window: &providerv1.HistoryWindowCommand{
			Terminal: &providerv1.TerminalRef{TerminalId: "term-history"}, Token: token, Cols: 80,
		},
	})
	if err != nil {
		t.Fatalf("history copy: %v", err)
	}
	if !strings.Contains(copied.GetText(), "history-alpha") {
		t.Fatalf("history copy text = %q", copied.GetText())
	}
	if _, err := client.HistorySearch(ctx, &providerv1.HistorySearchCommand{
		Terminal: &providerv1.TerminalRef{TerminalId: "term-history"}, Token: token, Cols: 80, Limit: 10,
		Query: "history-beta", Mode: providerv1.HistorySearchMode_HISTORY_SEARCH_MODE_TEXT,
	}); err != nil {
		t.Fatalf("history search: %v", err)
	}
	backlog, err := client.HistoryBacklogStatus(ctx, "term-history")
	if err != nil || backlog.GetTerminal().GetTerminalId() != "term-history" {
		t.Fatalf("history backlog = %#v err=%v", backlog, err)
	}
	if err := client.HistoryRelease(ctx, "term-history", token); err != nil {
		t.Fatalf("history release: %v", err)
	}
	if err := client.HistoryRelease(ctx, "term-history", token); providerCode(err) != providerproto.ErrorStaleResource {
		t.Fatalf("second release err = %v", err)
	}
	if _, err := client.HistoryWindow(ctx, &providerv1.HistoryWindowCommand{
		Terminal: &providerv1.TerminalRef{TerminalId: "term-history"}, Mode: providerv1.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDER, Token: token, Cols: 80,
	}); providerCode(err) != providerproto.ErrorStaleResource {
		t.Fatalf("stale window err = %v", err)
	}
}

// TestProviderLiveScreenNext 覆盖 native screen delta 与 baseline 协议。
func TestProviderLiveScreenNext(t *testing.T) {
	t.Parallel()
	_, _, socketPath := startProvider(t)
	ctx := context.Background()
	client, err := provider.Dial(ctx, socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-live", Command: []string{"/bin/sh", "-c", "printf 'live-surface\\n'; sleep 30"},
		Size: &providerv1.Size{Cols: 40, Rows: 10},
	}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := client.LiveScreenNext(ctx, "term-live", 0)
	if err != nil {
		t.Fatalf("live screen next: %v", err)
	}
	if snapshot == nil || snapshot.GetTerminal().GetTerminalId() != "term-live" {
		t.Fatalf("live screen = %#v", snapshot)
	}
	if snapshot.GetLiveRevision() == 0 {
		t.Fatalf("live revision = 0")
	}
}

// TestProviderEventSubscription 覆盖 terminal event 订阅、fan-out 与 release。
func TestProviderEventSubscription(t *testing.T) {
	t.Parallel()
	_, _, socketPath := startProvider(t)
	ctx := context.Background()
	client, err := provider.Dial(ctx, socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	eventCtx, cancelEvents := context.WithCancel(ctx)
	defer cancelEvents()
	events := client.Events(eventCtx)
	subscribed, err := client.EventSubscribe(ctx, &providerv1.EventSubscribeCommand{
		Types: []providerv1.TerminalEventType{providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_CREATED},
	})
	if err != nil {
		t.Fatalf("event subscribe: %v", err)
	}
	token := subscribed.GetOpaqueToken()
	if len(token) == 0 {
		t.Fatalf("subscription token is empty")
	}
	if _, err := client.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-events", Command: []string{"/bin/cat"}, Size: &providerv1.Size{Cols: 20, Rows: 5},
	}); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case event := <-events:
			if event.GetTerminalId() != "term-events" {
				continue
			}
			if event.GetType() != providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_CREATED {
				t.Fatalf("event type = %v", event.GetType())
			}
			goto released
		case <-deadline:
			t.Fatal("timed out waiting for terminal created event")
		}
	}
released:
	if err := client.EventRelease(ctx, token); err != nil {
		t.Fatalf("event release: %v", err)
	}
	if err := client.EventRelease(ctx, token); providerCode(err) != providerproto.ErrorNotFound {
		t.Fatalf("second release err = %v", err)
	}
	if _, err := client.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-events-2", Command: []string{"/bin/cat"}, Size: &providerv1.Size{Cols: 20, Rows: 5},
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.GetTerminalId() == "term-events-2" {
			t.Fatalf("released subscription received event %#v", event)
		}
	case <-time.After(300 * time.Millisecond):
	}
}

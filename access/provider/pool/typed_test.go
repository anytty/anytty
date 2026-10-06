package pool_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	adapter "github.com/anytty/anytty/access/provider/pool"
	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	"github.com/anytty/anytty/pool/core"
	provider "github.com/anytty/anytty/pool/provider"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
)

// TestAdapterTypedFacade 覆盖 typed 契约面：Info/Capabilities、生命周期与
// metadata、typed 错误投影、attach PTY 双向流、history 与 detach。
func TestAdapterTypedFacade(t *testing.T) {
	socketPath := startAdapterProvider(t)
	ctx := context.Background()
	terminal, err := adapter.DialTerminal(ctx, socketPath)
	if err != nil {
		t.Fatalf("dial adapter: %v", err)
	}
	defer func() { _ = terminal.Close() }()

	info, err := terminal.Info(ctx)
	if err != nil || info.Kind != "pool" {
		t.Fatalf("info = %#v err=%v", info, err)
	}
	capabilities := terminal.Capabilities()
	if !capabilities.Lifecycle || !capabilities.Metadata || !capabilities.Attach || !capabilities.History {
		t.Fatalf("capabilities = %#v", capabilities)
	}

	created, err := terminal.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-typed", Name: "typed", Command: []string{"/bin/cat"},
		Size: &providerv1.Size{Cols: 20, Rows: 5},
	})
	if err != nil || created.GetTerminalId() != "term-typed" {
		t.Fatalf("create = %#v err=%v", created, err)
	}
	if _, err := terminal.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-typed", Command: []string{"/bin/cat"},
	}); !errors.Is(err, terminalprovider.ErrConflict) {
		t.Fatalf("duplicate create error = %v, want ErrConflict", err)
	}

	items, err := terminal.List(ctx)
	if err != nil || len(items) != 1 {
		t.Fatalf("list = %#v err=%v", items, err)
	}
	if _, err := terminal.Get(ctx, "missing"); !errors.Is(err, terminalprovider.ErrNotFound) {
		t.Fatalf("missing get error = %v, want ErrNotFound", err)
	}

	renamed := "typed-renamed"
	updated, err := terminal.SetMetadata(ctx, "term-typed", terminalprovider.MetadataPatch{Name: &renamed})
	if err != nil || updated.GetName() != renamed {
		t.Fatalf("set metadata = %#v err=%v", updated, err)
	}
	updated, err = terminal.SetTags(ctx, "term-typed", terminalprovider.TagsPatch{Set: map[string]string{"role": "typed"}})
	if err != nil || updated.GetTags()["role"] != "typed" {
		t.Fatalf("set tags = %#v err=%v", updated, err)
	}
	// Replace=true 是客户端 terminal.set-tags 的整体替换语义：旧 tag 不保留。
	updated, err = terminal.SetTags(ctx, "term-typed", terminalprovider.TagsPatch{Set: map[string]string{"replace": "yes"}, Replace: true})
	if err != nil || len(updated.GetTags()) != 1 || updated.GetTags()["replace"] != "yes" {
		t.Fatalf("replace tags = %#v err=%v", updated, err)
	}
	updated, err = terminal.SetTags(ctx, "term-typed", terminalprovider.TagsPatch{Remove: []string{"replace"}})
	if err != nil || len(updated.GetTags()) != 0 {
		t.Fatalf("remove tags = %#v err=%v", updated, err)
	}

	attachment, err := terminal.Attach(ctx, terminalprovider.AttachRequest{
		TerminalID:   "term-typed",
		Mode:         providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR,
		ResizePolicy: providerv1.ResizePolicy_RESIZE_POLICY_OWNER,
		SurfaceID:    "surface-typed", ViewID: "view-typed",
	})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	if attachment.Handle().GetTerminalId() != "term-typed" || attachment.OwnerEpoch() == 0 {
		t.Fatalf("attachment handle = %#v epoch=%d", attachment.Handle(), attachment.OwnerEpoch())
	}

	duplex := attachment.Stream()
	if err := duplex.Send(ctx, []byte("typed-facade\n")); err != nil {
		t.Fatalf("stream send: %v", err)
	}
	if output := readDuplexUntil(t, duplex, "typed-facade"); output == "" {
		t.Fatal("typed attachment stream returned no output")
	}

	resized, err := attachment.Resize(ctx, &providerv1.Size{Cols: 30, Rows: 10}, providerv1.ResizePolicy_RESIZE_POLICY_OWNER, true, 0)
	if err != nil || resized.GetSize().GetCols() != 30 {
		t.Fatalf("resize = %#v err=%v", resized, err)
	}
	if size := attachment.Size(); size.GetCols() != 30 || size.GetRows() != 10 {
		t.Fatalf("attachment size after resize = %#v", size)
	}
	locked, err := attachment.ResizeLock(ctx, true)
	if err != nil || !locked.GetResizeControl().GetSizeLocked() {
		t.Fatalf("resize lock = %#v err=%v", locked, err)
	}
	if _, err := attachment.ResizeLock(ctx, false); err != nil {
		t.Fatalf("resize unlock: %v", err)
	}

	window, err := terminal.HistoryWindow(ctx, terminalprovider.HistoryRequest{
		TerminalID: "term-typed",
		Mode:       providerv1.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST,
		Cols:       30, Limit: 10,
	})
	if err != nil || window.GetToken() == "" {
		t.Fatalf("history window = %#v err=%v", window, err)
	}
	copied := readHistoryCopyUntil(t, terminal, window.GetToken(), "typed-facade")
	if !strings.Contains(copied, "typed-facade") {
		t.Fatalf("history copy = %q", copied)
	}
	found, err := terminal.HistorySearch(ctx, terminalprovider.HistorySearchRequest{
		TerminalID: "term-typed", Token: window.GetToken(), Query: "typed-facade", Cols: 30, Limit: 10,
	})
	if err != nil || !found.GetFound() {
		t.Fatalf("history search = %#v err=%v", found, err)
	}
	if err := terminal.HistoryRelease(ctx, "term-typed", window.GetToken()); err != nil {
		t.Fatalf("history release: %v", err)
	}
	backlog, err := terminal.HistoryBacklogStatus(ctx, "term-typed")
	if err != nil || !backlog.GetHistoryEnabled() {
		t.Fatalf("history backlog = %#v err=%v", backlog, err)
	}

	events, err := terminal.Events(ctx)
	if err != nil || events == nil {
		t.Fatalf("typed events = %v err=%v", events, err)
	}
	subscription, err := terminal.Subscribe(ctx, &providerv1.EventSubscribeCommand{TerminalId: "term-typed"})
	if err != nil || len(subscription.GetOpaqueToken()) == 0 {
		t.Fatalf("subscribe = %#v err=%v", subscription, err)
	}
	if err := terminal.EventRelease(ctx, subscription.GetOpaqueToken()); err != nil {
		t.Fatalf("event release: %v", err)
	}

	if err := attachment.Detach(ctx); err != nil {
		t.Fatalf("detach: %v", err)
	}

	restarted, err := terminal.Restart(ctx, "term-typed")
	if err != nil || restarted.GetTerminalId() != "term-typed" {
		t.Fatalf("restart = %#v err=%v", restarted, err)
	}
	if err := terminal.Kill(ctx, "term-typed"); err != nil {
		t.Fatalf("kill: %v", err)
	}
	if err := terminal.Remove(ctx, "term-typed"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := terminal.Get(ctx, "term-typed"); !errors.Is(err, terminalprovider.ErrNotFound) {
		t.Fatalf("get removed error = %v, want ErrNotFound", err)
	}
}

// TestAdapterTypedStreamCloseReportsExitCode 验证 provider 结束 PTY 流时 typed
// Duplex 以 StreamClosedError 汇报退出码，且之前的输出已经送达。
func TestAdapterTypedStreamCloseReportsExitCode(t *testing.T) {
	socketPath := startAdapterProvider(t)
	ctx := context.Background()
	terminal, err := adapter.DialTerminal(ctx, socketPath)
	if err != nil {
		t.Fatalf("dial adapter: %v", err)
	}
	defer func() { _ = terminal.Close() }()

	if _, err := terminal.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-close", Name: "close", Command: []string{"/bin/sh", "-c", "sleep 0.3; printf 'exit-marker\\n'; exit 7"},
		Size: &providerv1.Size{Cols: 20, Rows: 5},
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	attachment, err := terminal.Attach(ctx, terminalprovider.AttachRequest{TerminalID: "term-close"})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}

	duplex := attachment.Stream()
	receiveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var output strings.Builder
	for {
		chunk, err := duplex.Receive(receiveCtx)
		if err != nil {
			var closed *terminalprovider.StreamClosedError
			if !errors.As(err, &closed) {
				t.Fatalf("stream receive after %q: %v", output.String(), err)
			}
			if closed.ExitCode != 7 {
				t.Fatalf("stream close exit code = %d, want 7", closed.ExitCode)
			}
			break
		}
		output.Write(chunk)
	}
	if !strings.Contains(output.String(), "exit-marker") {
		t.Fatalf("stream output = %q, want exit-marker", output.String())
	}
}

// readDuplexUntil 读取 attachment 输出直到出现 marker 或超时。
func readDuplexUntil(t *testing.T, duplex terminalprovider.Duplex, marker string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var output strings.Builder
	for {
		chunk, err := duplex.Receive(ctx)
		if err != nil {
			t.Fatalf("stream receive after %q: %v", output.String(), err)
		}
		output.Write(chunk)
		if strings.Contains(output.String(), marker) {
			return output.String()
		}
	}
}

// readHistoryCopyUntil 轮询 HistoryCopy 直到 marker 落盘可读或超时。
func readHistoryCopyUntil(t *testing.T, terminal *adapter.TerminalProvider, token, marker string) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		result, err := terminal.HistoryCopy(context.Background(), terminalprovider.HistoryCopyRequest{
			TerminalID: "term-typed",
			Window:     &terminalprovider.HistoryRequest{TerminalID: "term-typed", Token: token, Cols: 30},
		})
		if err == nil {
			last = result.GetText()
			if strings.Contains(last, marker) {
				return last
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return last
}

func startAdapterProvider(t *testing.T) string {
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
			return socketPath
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("provider socket did not appear")
	return ""
}

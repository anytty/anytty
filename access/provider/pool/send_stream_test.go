package pool

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	"github.com/anytty/anytty/internal/providerproto"
	"github.com/anytty/anytty/pool/core"
	poolprovider "github.com/anytty/anytty/pool/provider"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
	"github.com/anytty/anytty/shared/transport"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
)

// frameSpyTransport 记录客户端发出的每个 provider frame（channel + type），
// 再把帧交给真实 transport。测试用它观察 typed facade 到底发了哪些流控帧。
type frameSpyTransport struct {
	inner transport.Transport

	mu   sync.Mutex
	sent [][2]uint32
}

func (spy *frameSpyTransport) Send(frame []byte) error {
	if channel, typ, _, err := providerproto.DecodeFrame(frame); err == nil {
		spy.mu.Lock()
		spy.sent = append(spy.sent, [2]uint32{uint32(channel), uint32(typ)})
		spy.mu.Unlock()
	}
	return spy.inner.Send(frame)
}

func (spy *frameSpyTransport) Recv() ([]byte, error)     { return spy.inner.Recv() }
func (spy *frameSpyTransport) Close() error             { return spy.inner.Close() }
func (spy *frameSpyTransport) Done() <-chan struct{}    { return spy.inner.Done() }
func (spy *frameSpyTransport) reset()                   { spy.mu.Lock(); spy.sent = nil; spy.mu.Unlock() }

func (spy *frameSpyTransport) countType(typ uint8) int {
	spy.mu.Lock()
	defer spy.mu.Unlock()
	count := 0
	for _, entry := range spy.sent {
		if uint8(entry[1]) == typ {
			count++
		}
	}
	return count
}

// TestSendInputDoesNotBootstrapRawPTYStream 锁住一条回归：
// 写 PTY 输入不允许顺带 bootstrap 原始 PTY 输出流。
//
// 只轮询 canonical screen 的客户端（移动端）从不消费原始输出。bootstrap 之后
// provider 会持续推送 PTY bytes，没人读就会填满 provider 的 stream 缓冲并让
// 整条 provider 会话失败（"stream consumer fell behind"），所有终端一起掉线。
func TestSendInputDoesNotBootstrapRawPTYStream(t *testing.T) {
	socketPath := startSendSpyProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	connection, err := unixtransport.DialContext(ctx, socketPath)
	if err != nil {
		t.Fatalf("dial provider socket: %v", err)
	}
	spy := &frameSpyTransport{inner: connection}
	client := poolprovider.NewClient(spy)
	if err := client.Hello(ctx); err != nil {
		t.Fatalf("provider hello: %v", err)
	}
	defer func() { _ = client.Close() }()

	provider := &TerminalProvider{socket: socketPath, client: client, cancel: func() {}}
	if _, err := provider.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-send-spy",
		Command:    []string{"/bin/cat"},
		Size:       &providerv1.Size{Cols: 40, Rows: 10},
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelCleanup()
		_ = provider.Kill(cleanupCtx, "term-send-spy")
		_ = provider.Remove(cleanupCtx, "term-send-spy")
	})

	attachment, err := provider.Attach(ctx, terminalprovider.AttachRequest{
		TerminalID:   "term-send-spy",
		Mode:         providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR,
		ResizePolicy: providerv1.ResizePolicy_RESIZE_POLICY_OWNER,
		SurfaceID:    "surface-send-spy",
		ViewID:       "view-send-spy",
	})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	defer func() { _ = attachment.Detach(context.Background()) }()

	duplex := attachment.Stream()
	spy.reset()
	if err := duplex.Send(ctx, []byte("spy-input\n")); err != nil {
		t.Fatalf("send input: %v", err)
	}
	// 给发送协程一点时间把帧写到 transport 上。
	time.Sleep(100 * time.Millisecond)
	if count := spy.countType(providerproto.StreamBootstrapDone); count != 0 {
		t.Fatalf("input write sent %d bootstrap frames; input must not open the raw PTY stream", count)
	}

	// 探针自检：显式请求原始输出时必须出现 bootstrap，否则上面的断言没有意义。
	spy.reset()
	receiveDone := make(chan struct{})
	go func() {
		defer close(receiveDone)
		readCtx, cancelRead := context.WithTimeout(ctx, 3*time.Second)
		defer cancelRead()
		for {
			payload, err := duplex.Receive(readCtx)
			if err != nil {
				return
			}
			if len(payload) > 0 {
				return
			}
		}
	}()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && spy.countType(providerproto.StreamBootstrapDone) == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if spy.countType(providerproto.StreamBootstrapDone) == 0 {
		t.Fatal("explicit raw stream open did not send bootstrap; the spy is not observing client frames")
	}
	<-receiveDone
}

// startSendSpyProvider 启动一个真实 pool provider 服务，返回其 unix socket。
func startSendSpyProvider(t *testing.T) string {
	t.Helper()
	coreServer := core.NewServer(core.WithHistoryStorageDir(t.TempDir()))
	t.Cleanup(func() { _ = coreServer.Shutdown(context.Background()) })
	socketPath := filepath.Join(t.TempDir(), "provider.sock")
	server, err := poolprovider.New(coreServer, poolprovider.Config{Socket: socketPath})
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

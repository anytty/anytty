package server_test

import (
	"context"
	"path/filepath"
	"sort"
	"testing"
	"time"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	poolprovider "github.com/anytty/anytty/access/provider/pool"
	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	accessserver "github.com/anytty/anytty/access/server"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	corev2 "github.com/anytty/anytty/pool/core"
	providercore "github.com/anytty/anytty/pool/provider"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
)

// latencyEnv 是一条进程内的完整链路：client → access(.access) → provider → pool PTY。
// 用来量栈内开销，不含网络与渲染。
type latencyEnv struct {
	ctx      context.Context
	client   *internalprotocol.Client
	app      *clientruntime.ApplicationSession
	terminal *apipb.TerminalRef
	attach   *apipb.AttachmentHandle
}

func newLatencyEnv(tb testing.TB, command []string, cols, rows uint32) *latencyEnv {
	tb.Helper()
	dir := tb.TempDir()
	daemonSocket := filepath.Join(dir, "pool.sock")
	accessSocket := filepath.Join(dir, "pool.sock.access")

	ctx, cancel := context.WithCancel(context.Background())
	tb.Cleanup(cancel)

	coreServer := corev2.NewServer(
		corev2.WithSocketPath(daemonSocket),
		corev2.WithHistoryDisabled(),
	)
	tb.Cleanup(func() { _ = coreServer.Shutdown(context.Background()) })
	providerServer, err := providercore.New(coreServer, providercore.Config{Socket: daemonSocket})
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { _ = providerServer.Shutdown(context.Background()) })
	go func() { _ = providerServer.ListenAndServe(ctx) }()
	waitForSocket(tb, daemonSocket)

	access, err := accessserver.New(accessserver.Config{
		Socket: accessSocket,
		Files:  accessserverTestFiles(tb),
		Provider: func(dialCtx context.Context) (terminalprovider.Provider, error) {
			return poolprovider.DialTerminal(dialCtx, daemonSocket)
		},
	})
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { _ = access.Close() })
	go func() { _ = access.Serve(ctx) }()
	waitForSocket(tb, accessSocket)

	connection, err := unixtransport.DialContext(ctx, accessSocket)
	if err != nil {
		tb.Fatal(err)
	}
	client := internalprotocol.NewClient(connection)
	tb.Cleanup(func() { _ = client.Close() })
	if err := client.Hello(ctx, internalprotocol.Hello{Version: wire.Version, Client: "latency-bench"}); err != nil {
		tb.Fatal(err)
	}
	application, err := clientruntime.NewApplicationSession(clientruntime.EndpointSessionStamp{
		EndpointID: clientendpoint.EndpointID("local"),
		RouteID:    clientendpoint.RouteID("latency-bench"),
		Generation: 1,
	}, client)
	if err != nil {
		tb.Fatal(err)
	}

	ref := &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-bench"}
	if _, err := application.TerminalCreate(ctx, &apipb.TerminalCreateCommand{Terminal: &apipb.TerminalCreateSpec{
		TerminalId: ref.GetTerminalId(),
		Command:    command,
		Size:       &apipb.TerminalSize{Cols: cols, Rows: rows},
	}}); err != nil {
		tb.Fatalf("create terminal: %v", err)
	}
	tb.Cleanup(func() {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelCleanup()
		_ = application.TerminalKill(cleanupCtx, &apipb.TerminalKillCommand{Terminal: ref})
		_ = application.TerminalRemove(cleanupCtx, &apipb.TerminalRemoveCommand{Terminal: ref})
	})

	attached, err := application.TerminalAttach(ctx, &apipb.TerminalAttachCommand{
		Terminal:     ref,
		Mode:         apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR,
		ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		SurfaceId:    "surface-latency",
		ViewId:       "view-latency",
	})
	if err != nil {
		tb.Fatalf("attach terminal: %v", err)
	}
	attachment := attached.GetAttachment()
	if _, ok := client.ApplicationAttachmentChannel(attachment.GetResource()); !ok {
		tb.Fatal("access did not bind the attachment resource to a client channel")
	}
	return &latencyEnv{
		ctx:      ctx,
		client:   client,
		app:      application,
		terminal: ref,
		attach:   attachment,
	}
}

// BenchmarkInputToScreenLatency 量「写输入 → PTY 回显 → canonical screen 可见」。
//
// 这是移动端渲染路径的栈内成本：不开原始 PTY 流，只长轮询屏幕 delta。
// 本机 unix socket，因此数字里不含网络 RTT 与客户端渲染开销。
func BenchmarkInputToScreenLatency(b *testing.B) {
	env := newLatencyEnv(b, []string{"/bin/cat"}, 120, 40)

	observed, err := env.currentRevision()
	if err != nil {
		b.Fatal(err)
	}
	// 预热：跑一轮让 provider/access 的 goroutine 都起来。
	if err := env.typeAndWait(&observed); err != nil {
		b.Fatal(err)
	}

	acks := make([]time.Duration, 0, b.N)
	totals := make([]time.Duration, 0, b.N)
	for i := 0; i < b.N; i++ {
		start := time.Now()
		if err := env.app.TerminalInput(env.ctx, &apipb.TerminalInputCommand{
			Attachment: env.attach.GetResource(),
			Data:       []byte("a"),
		}); err != nil {
			b.Fatalf("input: %v", err)
		}
		acks = append(acks, time.Since(start))
		screen, err := env.app.LiveScreenNext(env.ctx, &apipb.LiveScreenNextCommand{
			Terminal:         env.terminal,
			ObservedRevision: observed,
		})
		if err != nil {
			b.Fatalf("live screen: %v", err)
		}
		if screen.GetLiveRevision() <= observed {
			b.Fatalf("revision did not advance: observed=%d got=%d", observed, screen.GetLiveRevision())
		}
		observed = screen.GetLiveRevision()
		totals = append(totals, time.Since(start))
	}
	sort.Slice(acks, func(i, j int) bool { return acks[i] < acks[j] })
	sort.Slice(totals, func(i, j int) bool { return totals[i] < totals[j] })
	b.ReportMetric(percentileMicros(acks, 0.50), "us/ack-p50")
	b.ReportMetric(percentileMicros(acks, 0.95), "us/ack-p95")
	b.ReportMetric(percentileMicros(totals, 0.50), "us/screen-p50")
	b.ReportMetric(percentileMicros(totals, 0.95), "us/screen-p95")
}

// BenchmarkInputAckLatency 单独量输入应答：客户端写输入 → access 有序队列 →
// provider 写 PTY → 应答回到客户端。它把「传输/队列」开销和屏幕合成分开看。
func BenchmarkInputAckLatency(b *testing.B) {
	env := newLatencyEnv(b, []string{"/bin/cat"}, 120, 40)
	if _, err := env.currentRevision(); err != nil {
		b.Fatal(err)
	}
	samples := make([]time.Duration, 0, b.N)
	for i := 0; i < b.N; i++ {
		start := time.Now()
		if err := env.app.TerminalInput(env.ctx, &apipb.TerminalInputCommand{
			Attachment: env.attach.GetResource(),
			Data:       []byte("a"),
		}); err != nil {
			b.Fatalf("input: %v", err)
		}
		samples = append(samples, time.Since(start))
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	b.ReportMetric(percentileMicros(samples, 0.50), "us/p50")
	b.ReportMetric(percentileMicros(samples, 0.95), "us/p95")
}

func (env *latencyEnv) currentRevision() (uint64, error) {
	screen, err := env.app.LiveScreenNext(env.ctx, &apipb.LiveScreenNextCommand{
		Terminal:         env.terminal,
		ObservedRevision: 0,
	})
	if err != nil {
		return 0, err
	}
	return screen.GetLiveRevision(), nil
}

func (env *latencyEnv) typeAndWait(observed *uint64) error {
	if err := env.app.TerminalInput(env.ctx, &apipb.TerminalInputCommand{
		Attachment: env.attach.GetResource(),
		Data:       []byte("a"),
	}); err != nil {
		return err
	}
	screen, err := env.app.LiveScreenNext(env.ctx, &apipb.LiveScreenNextCommand{
		Terminal:         env.terminal,
		ObservedRevision: *observed,
	})
	if err != nil {
		return err
	}
	*observed = screen.GetLiveRevision()
	return nil
}

func percentileMicros(sorted []time.Duration, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1) * p)
	return float64(sorted[index].Microseconds())
}

package server_test

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	accessserver "github.com/anytty/anytty/access/server"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
	"github.com/anytty/anytty/shared/transport/memory"
)

// blockingProvider 在 access in-flight 预算测试中占住请求槽，直到 release 关闭。
type blockingProvider struct {
	entered atomic.Int32
	release chan struct{}
}

var _ terminalprovider.Provider = (*blockingProvider)(nil)

func (*blockingProvider) Info(context.Context) (terminalprovider.Info, error) {
	return terminalprovider.Info{Kind: "blocking-test"}, nil
}

func (*blockingProvider) Capabilities() terminalprovider.Capabilities {
	return terminalprovider.Capabilities{Lifecycle: true}
}

func (*blockingProvider) Create(context.Context, *providerv1.TerminalCreateSpec) (*providerv1.TerminalInfo, error) {
	return nil, terminalprovider.ErrUnsupported
}

func (provider *blockingProvider) List(ctx context.Context) ([]*providerv1.TerminalInfo, error) {
	provider.entered.Add(1)
	select {
	case <-provider.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return nil, nil
}

func (*blockingProvider) Get(context.Context, string) (*providerv1.TerminalInfo, error) {
	return nil, terminalprovider.ErrUnsupported
}

func (*blockingProvider) Restart(context.Context, string) (*providerv1.TerminalInfo, error) {
	return nil, terminalprovider.ErrUnsupported
}

func (*blockingProvider) Kill(context.Context, string) error { return terminalprovider.ErrUnsupported }

func (*blockingProvider) Remove(context.Context, string) error {
	return terminalprovider.ErrUnsupported
}

func (*blockingProvider) SetMetadata(context.Context, string, terminalprovider.MetadataPatch) (*providerv1.TerminalInfo, error) {
	return nil, terminalprovider.ErrUnsupported
}

func (*blockingProvider) SetTags(context.Context, string, terminalprovider.TagsPatch) (*providerv1.TerminalInfo, error) {
	return nil, terminalprovider.ErrUnsupported
}

func (*blockingProvider) Attach(context.Context, terminalprovider.AttachRequest) (terminalprovider.Attachment, error) {
	return nil, terminalprovider.ErrUnsupported
}

func (*blockingProvider) HistoryWindow(context.Context, terminalprovider.HistoryRequest) (*providerv1.HistoryWindowResult, error) {
	return nil, terminalprovider.ErrUnsupported
}

func (*blockingProvider) HistoryCopy(context.Context, terminalprovider.HistoryCopyRequest) (*providerv1.HistoryCopyResult, error) {
	return nil, terminalprovider.ErrUnsupported
}

func (*blockingProvider) HistorySearch(context.Context, terminalprovider.HistorySearchRequest) (*providerv1.HistorySearchResult, error) {
	return nil, terminalprovider.ErrUnsupported
}

func (*blockingProvider) HistoryRelease(context.Context, string, string) error {
	return terminalprovider.ErrUnsupported
}

func (*blockingProvider) HistoryBacklogStatus(context.Context, string) (*providerv1.HistoryBacklogStatusResult, error) {
	return nil, terminalprovider.ErrUnsupported
}

func (*blockingProvider) LiveScreen(context.Context, string, uint64) (*providerv1.NativeScreenResult, error) {
	return nil, terminalprovider.ErrUnsupported
}

func (*blockingProvider) Subscribe(context.Context, *providerv1.EventSubscribeCommand) (*providerv1.EventSubscriptionResult, error) {
	return nil, terminalprovider.ErrUnsupported
}

func (*blockingProvider) Events(context.Context) (<-chan *providerv1.TerminalEvent, error) {
	return nil, terminalprovider.ErrUnsupported
}

func (*blockingProvider) EventRelease(context.Context, []byte) error {
	return terminalprovider.ErrUnsupported
}

func (*blockingProvider) TerminalDefaults(context.Context) (*providerv1.TerminalDefaults, error) {
	return nil, terminalprovider.ErrUnsupported
}

func (*blockingProvider) ListDirectories(context.Context, string, int32) (*providerv1.PathListDirectoriesResult, error) {
	return nil, terminalprovider.ErrUnsupported
}

func (*blockingProvider) Done() <-chan struct{} {
	closed := make(chan struct{})
	close(closed)
	return closed
}
func (*blockingProvider) Err() error   { return nil }
func (*blockingProvider) Close() error { return nil }

// TestAccessSessionInFlightBudgetExhaustsAndReleases 迁移旧 protocol 的
// per-connection in-flight 预算语义：超过上限返回 429，完成后槽位复用。
func TestAccessSessionInFlightBudgetExhaustsAndReleases(t *testing.T) {
	t.Parallel()
	provider := &blockingProvider{release: make(chan struct{})}
	access, err := accessserver.New(accessserver.Config{
		Socket:   filepath.Join(t.TempDir(), "unused.sock"),
		Provider: func(context.Context) (terminalprovider.Provider, error) { return provider, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = access.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clientTransport, serverTransport := memory.NewPair()
	go func() { _ = access.ServeTransport(ctx, serverTransport) }()

	client := internalprotocol.NewClient(clientTransport)
	defer func() { _ = client.Close() }()
	if err := client.Hello(ctx, internalprotocol.Hello{Version: wire.Version, Client: "access-budget-e2e"}); err != nil {
		t.Fatal(err)
	}
	application, err := clientruntime.NewApplicationSession(clientruntime.EndpointSessionStamp{
		EndpointID: clientendpoint.EndpointID("local"),
		RouteID:    clientendpoint.RouteID("budget-test"),
		Generation: 1,
	}, client)
	if err != nil {
		t.Fatal(err)
	}

	const budget = 64
	var wait sync.WaitGroup
	errs := make(chan error, budget)
	for index := 0; index < budget; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := application.TerminalList(ctx, &apipb.TerminalListCommand{})
			errs <- err
		}()
	}
	deadline := time.Now().Add(5 * time.Second)
	for provider.entered.Load() < budget && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := provider.entered.Load(); got != budget {
		t.Fatalf("provider entered %d requests, want %d", got, budget)
	}

	if _, err := application.TerminalList(ctx, &apipb.TerminalListCommand{}); err == nil || !strings.Contains(err.Error(), "capacity is exhausted") {
		t.Fatalf("excess request error = %v, want exhausted", err)
	}

	close(provider.release)
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("in-flight request failed: %v", err)
		}
	}
	if _, err := application.TerminalList(ctx, &apipb.TerminalListCommand{}); err != nil {
		t.Fatalf("request after release failed: %v", err)
	}
}

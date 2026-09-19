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
	"github.com/anytty/anytty/shared/transport/memory"
)

// blockingProvider 在 access in-flight 预算测试中占住请求槽，直到 release 关闭。
type blockingProvider struct {
	entered atomic.Int32
	release chan struct{}
}

var _ terminalprovider.Provider = (*blockingProvider)(nil)

func (provider *blockingProvider) Execute(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	provider.entered.Add(1)
	select {
	case <-provider.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return &apipb.ResultEnvelope{
		RequestId:     command.GetContext().GetRequestId(),
		OriginSession: command.GetContext().GetSession(),
		Result:        &apipb.ResultEnvelope_TerminalList{TerminalList: &apipb.TerminalListResult{}},
	}, nil
}

func (*blockingProvider) OpenStream(context.Context, *apipb.ResourceHandle) (terminalprovider.Stream, error) {
	return nil, terminalprovider.ErrUnsupported
}
func (*blockingProvider) Events(context.Context) (<-chan *apipb.EventEnvelope, error) {
	closed := make(chan *apipb.EventEnvelope)
	close(closed)
	return closed, nil
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

package local

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
)

func TestDialerDoesNotStartSecondDaemonAfterHandshakeFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.sock")
	listener, err := unixtransport.NewListener(path)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	accepted := make(chan struct{})
	go func() {
		transport, acceptErr := listener.Accept(context.Background())
		if acceptErr != nil {
			return
		}
		close(accepted)
		<-ctx.Done()
		_ = transport.Close()
	}()

	target, ok := endpoint.DefaultRegistry().DefaultEndpoint()
	if !ok {
		t.Fatal("default local endpoint is unavailable")
	}
	owner := clientruntime.NewSessionOwner()
	defer owner.Close()
	attempt, err := owner.BeginRouteAttempt(target, endpoint.DefaultLocalRouteID, clientruntime.ConnectIntentInteractive)
	if err != nil {
		t.Fatalf("begin local attempt: %v", err)
	}
	var started atomic.Bool
	dialer := NewDialer(Options{
		SocketOverride: path,
		Start: func(context.Context, string) error {
			started.Store(true)
			return nil
		},
	})
	if _, err := dialer.Connect(ctx, attempt); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("connect error = %v, want deadline exceeded", err)
	}
	if started.Load() {
		t.Fatal("handshake failure started a second daemon")
	}
	select {
	case <-accepted:
	default:
		t.Fatal("test server did not accept the transport connection")
	}
}

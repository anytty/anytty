package memory

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/anytty/anytty/shared/transport"
)

func TestListenerDialAcceptRoundTrip(t *testing.T) {
	listener := NewListener("memory://test")
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	accepted := make(chan transport.Transport, 1)
	go func() {
		conn, err := listener.Accept(ctx)
		if err == nil {
			accepted <- conn
		}
	}()

	client, err := listener.Dial(ctx)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer client.Close()

	var server transport.Transport
	select {
	case server = <-accepted:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for accepted transport")
	}
	defer server.Close()

	if listener.Addr() != "memory://test" {
		t.Fatalf("unexpected listener addr: %s", listener.Addr())
	}

	if err := client.Send([]byte("ping")); err != nil {
		t.Fatalf("client send failed: %v", err)
	}
	got, err := server.Recv()
	if err != nil {
		t.Fatalf("server recv failed: %v", err)
	}
	if !bytes.Equal(got, []byte("ping")) {
		t.Fatalf("unexpected payload: %q", string(got))
	}
}

func TestListenerCloseUnblocksAccept(t *testing.T) {
	listener := NewListener("memory://test")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := listener.Accept(ctx)
		done <- err
	}()

	time.Sleep(20 * time.Millisecond)
	if err := listener.Close(); err != nil {
		t.Fatalf("listener close failed: %v", err)
	}

	select {
	case err := <-done:
		if !errors.Is(err, transport.ErrListenerClosed) {
			t.Fatalf("expected ErrListenerClosed, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for accept to unblock")
	}
}

func TestListenerAcceptContextCanceled(t *testing.T) {
	listener := NewListener("memory://test")
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := listener.Accept(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestListenerDialContextCanceled(t *testing.T) {
	listener := NewListener("memory://test")
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := listener.Dial(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled from dial, got %v", err)
	}
}

func TestListenerDialAfterCloseReturnsErr(t *testing.T) {
	listener := NewListener("memory://test")
	if err := listener.Close(); err != nil {
		t.Fatalf("listener close failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := listener.Dial(ctx); !errors.Is(err, transport.ErrListenerClosed) {
		t.Fatalf("expected ErrListenerClosed from dial, got %v", err)
	}
}

func TestTransportDoneAndEOFAfterClose(t *testing.T) {
	client, server := NewPair()
	_ = server

	if err := client.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	select {
	case <-client.Done():
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for done channel")
	}

	if err := client.Send([]byte("x")); !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF from send after close, got %v", err)
	}
	if _, err := client.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF from recv after close, got %v", err)
	}
}

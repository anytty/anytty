package core

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestBrowserReceiveWindowBoundsCreditAndAcknowledgements(t *testing.T) {
	window := newBrowserReceiveWindow(64)
	defer window.close()
	if err := window.recordSent(64); err != nil {
		t.Fatal(err)
	}
	if _, err := window.available(32, 10*time.Millisecond); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("full window: %v", err)
	}
	for _, ack := range [][2]int64{{65, 65}, {32, 64}, {-1, -1}} {
		if err := window.acknowledge(ack[0], ack[1]); err == nil {
			t.Fatalf("invalid credit accepted: %v", ack)
		}
	}
	if err := window.acknowledge(32, 32); err != nil {
		t.Fatal(err)
	}
	if err := window.acknowledge(32, 32); err == nil {
		t.Fatal("duplicate credit accepted")
	}
	if err := window.acknowledge(32, 0); err != nil {
		t.Fatal(err)
	}
	if available, err := window.available(64, time.Second); err != nil || available != 32 {
		t.Fatalf("credit=%d err=%v", available, err)
	}
	if err := window.recordSent(33); err == nil {
		t.Fatal("window exceeded")
	}
}

func TestBrowserReceiveWindowCloseUnblocksWaiter(t *testing.T) {
	window := newBrowserReceiveWindow(1)
	if err := window.recordSent(1); err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	go func() { _, err := window.available(1, time.Minute); finished <- err }()
	window.close()
	select {
	case err := <-finished:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("close error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("closed window retained sender")
	}
}

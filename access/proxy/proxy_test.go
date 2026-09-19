package proxy

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestReceiveWindowBoundsCreditAndAcknowledgements(t *testing.T) {
	window := newReceiveWindow(64)
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

func TestReceiveWindowCloseUnblocksWaiter(t *testing.T) {
	window := newReceiveWindow(1)
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

func TestUploadQueueBoundsInflightAndWraps(t *testing.T) {
	queue := newUploadQueue(8)
	defer queue.close()
	if err := queue.enqueue([]byte("abcdefgh")); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 5)
	if n, err := queue.peek(buffer); err != nil || n != 5 || string(buffer) != "abcde" {
		t.Fatalf("peek %q %d %v", buffer, n, err)
	}
	if err := queue.enqueue([]byte("x")); err == nil {
		t.Fatal("in-flight bytes freed credit before write completion")
	}
	queue.consume(5)
	if err := queue.enqueue([]byte("ijklm")); err != nil {
		t.Fatal(err)
	}
	buffer = make([]byte, 8)
	if n, err := queue.peek(buffer); err != nil || n != 8 || string(buffer) != "fghijklm" {
		t.Fatalf("wrap %q %d %v", buffer, n, err)
	}
	queue.consume(8)
	finished := make(chan error, 1)
	go func() { _, err := queue.peek(buffer); finished <- err }()
	queue.close()
	select {
	case err := <-finished:
		if err != io.EOF {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("queue close did not release worker")
	}
}

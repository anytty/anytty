package server

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
)

// TestPTYForwarderKeepsReadingWhileSendBlocks 验证背压隔离：
// 发送侧卡住时读取侧必须继续接收，并在恢复后按序送达、合并成更少的帧。
//
// 回归：逐帧同步转发会让 provider 的 stream 缓冲在客户端背压时溢出，
// provider 以 "stream consumer fell behind" 关闭整条连接，所有终端掉线。
func TestPTYForwarderKeepsReadingWhileSendBlocks(t *testing.T) {
	// 无缓冲：每次喂块都必须被读取侧取走，才能证明读取没有停在发送上。
	chunks := make(chan []byte)
	receive := func(ctx context.Context) ([]byte, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case chunk := <-chunks:
			return chunk, nil
		}
	}

	release := make(chan struct{})
	var mu sync.Mutex
	var delivered []byte
	frames := 0
	send := func(payload []byte) error {
		<-release // 第一次发送卡住，模拟客户端链路背压
		mu.Lock()
		delivered = append(delivered, payload...)
		frames++
		mu.Unlock()
		return nil
	}

	f := &ptyForwarder{
		receive:     receive,
		send:        send,
		batchWindow: time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go f.Run(ctx)

	const total = 300
	const chunkSize = 1024
	want := make([]byte, 0, total*chunkSize)
	for i := 0; i < total; i++ {
		chunk := bytes.Repeat([]byte{byte(i % 251)}, chunkSize)
		want = append(want, chunk...)
		select {
		case chunks <- chunk:
		case <-time.After(2 * time.Second):
			t.Fatalf("reader stalled after %d chunks while send was blocked", i)
		}
	}

	close(release)

	deadline := time.Now().Add(5 * time.Second)
	for {
		mu.Lock()
		n := len(delivered)
		mu.Unlock()
		if n == len(want) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("delivered %d/%d bytes after backpressure cleared", n, len(want))
		}
		time.Sleep(5 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if !bytes.Equal(delivered, want) {
		t.Fatal("coalesced output does not preserve byte order")
	}
	if frames >= total {
		t.Fatalf("expected coalescing, got %d frames for %d chunks", frames, total)
	}
}

// TestPTYForwarderReportsOverflowWithoutHanging 验证积压超过上限时以 overflow
// 结束，而不是无限占用内存或卡住调用方。
func TestPTYForwarderReportsOverflowWithoutHanging(t *testing.T) {
	chunks := make(chan []byte, 64)
	receive := func(ctx context.Context) ([]byte, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case chunk := <-chunks:
			return chunk, nil
		}
	}
	blocked := make(chan struct{})
	send := func([]byte) error {
		<-blocked // 模拟客户端背压
		return nil
	}

	ended := make(chan ptyForwardEnd, 1)
	f := &ptyForwarder{
		receive:     receive,
		send:        send,
		maxPending:  4 << 10,
		batchWindow: time.Millisecond,
		onEnd:       func(end ptyForwardEnd) { ended <- end },
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go f.Run(ctx)

	// 第一帧会被发送侧取走并卡住，其余帧全部进入积压，很快超过 4KB。
	for i := 0; i < 32; i++ {
		chunks <- bytes.Repeat([]byte{'x'}, 1024)
	}
	// 积压超限会在读取侧立刻停止读取；正在发送的那一帧仍可能阻塞，
	// 释放它之后泵才会把 overflow 上报给调用方。
	time.Sleep(50 * time.Millisecond)
	close(blocked)

	select {
	case end := <-ended:
		if !end.Overflow {
			t.Fatalf("end = %#v, want overflow", end)
		}
		if end.Err != nil {
			t.Fatalf("overflow end carries unexpected error: %v", end.Err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("forwarder did not end after pending budget was exceeded (pending=%d)", f.pendingBytes())
	}
}

// TestPTYForwarderProjectsProviderClose 验证 provider 结束原因被原样上报，
// 且结束前已缓冲的字节仍然送达。
func TestPTYForwarderProjectsProviderClose(t *testing.T) {
	var step int
	var mu sync.Mutex
	receive := func(ctx context.Context) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		step++
		if step == 1 {
			return []byte("last-output"), nil
		}
		return nil, &terminalprovider.StreamClosedError{ExitCode: 7}
	}

	var delivered bytes.Buffer
	ended := make(chan ptyForwardEnd, 1)
	f := &ptyForwarder{
		receive:     receive,
		send:        func(payload []byte) error { _, err := delivered.Write(payload); return err },
		batchWindow: time.Millisecond,
		onEnd:       func(end ptyForwardEnd) { ended <- end },
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go f.Run(ctx)

	select {
	case end := <-ended:
		var closed *terminalprovider.StreamClosedError
		if !errors.As(end.Err, &closed) {
			t.Fatalf("end error = %v, want StreamClosedError", end.Err)
		}
		if closed.ExitCode != 7 {
			t.Fatalf("exit code = %d, want 7", closed.ExitCode)
		}
		if got := delivered.String(); got != "last-output" {
			t.Fatalf("delivered = %q, want buffered bytes before close", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("forwarder did not report provider close")
	}
}

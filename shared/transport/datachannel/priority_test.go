package datachannel

import (
	"errors"
	"testing"
	"time"
)

// TestControlPriorityBypassesSendWatermark 锁住一条回归：
// 输入应答/错误这类小帧不能等在屏幕积压后面。
//
// 这条 DataChannel 是有序单流；等待水位意味着控制帧要排在已经入队的数据
// 之后。0.5MB/s 的链路上，一次屏幕突发就能把回显压到数秒后（"点了没反应"）。
func TestControlPriorityBypassesSendWatermark(t *testing.T) {
	clientChannel, serverChannel := newFakeChannelPair()
	clientChannel.setBufferedAmount(1 << 20) // 远高于任何水位
	transport := New(clientChannel)
	peer := New(serverChannel)
	defer transport.Close()
	defer peer.Close()
	transport.drainTimeout = 5 * time.Second

	started := time.Now()
	if err := transport.SendWithPriority([]byte("input-ack"), PriorityControl); err != nil {
		t.Fatalf("control send: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("control frame waited %s behind the send watermark", elapsed)
	}
	if got := clientChannel.sendCallCount(); got != 1 {
		t.Fatalf("channel send calls = %d, want 1", got)
	}
}

// TestBulkPriorityYieldsBeforeNormal 验证 bulk 流量比普通流量更早让路，
// 只有普通帧还有空间时 bulk 才继续写。
func TestBulkPriorityYieldsBeforeNormal(t *testing.T) {
	clientChannel, serverChannel := newFakeChannelPair()
	// 高于 bulk 水位、低于 normal 水位。
	clientChannel.setBufferedAmount(uint64(bulkSendBufferHigh) + 1)
	transport := New(clientChannel)
	peer := New(serverChannel)
	defer transport.Close()
	defer peer.Close()
	transport.drainTimeout = 10 * time.Millisecond

	if err := transport.SendWithPriority([]byte("file"), PriorityBulk); !errors.Is(err, ErrBackpressure) {
		t.Fatalf("bulk send error = %v, want ErrBackpressure", err)
	}
	if err := transport.SendWithPriority([]byte("screen"), PriorityNormal); err != nil {
		t.Fatalf("normal send below normal watermark: %v", err)
	}
}

// TestNormalPriorityStillWaitsForWatermark 保证普通帧的背压语义没被优先级改动破坏。
func TestNormalPriorityStillWaitsForWatermark(t *testing.T) {
	clientChannel, serverChannel := newFakeChannelPair()
	clientChannel.setBufferedAmount(defaultSendBufferHigh + 1)
	transport := New(clientChannel)
	peer := New(serverChannel)
	defer transport.Close()
	defer peer.Close()
	transport.drainTimeout = 10 * time.Millisecond

	if err := transport.Send([]byte("screen")); !errors.Is(err, ErrBackpressure) {
		t.Fatalf("normal send error = %v, want ErrBackpressure", err)
	}
	// 背压不是连接故障：transport 仍必须可用。
	clientChannel.setBufferedAmount(0)
	if err := transport.Send([]byte("screen-again")); err != nil {
		t.Fatalf("send after backpressure: %v", err)
	}
}

func (channel *fakeChannel) sendCallCount() int {
	channel.mu.Lock()
	defer channel.mu.Unlock()
	return channel.sendCalls
}

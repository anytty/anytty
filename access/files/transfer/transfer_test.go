package transfer

import (
	"testing"
	"time"
)

func TestAdaptPicksWindowByLatency(t *testing.T) {
	lan := Adapt(1*time.Millisecond, 0)
	if lan.ChunkBytes != 64<<10 || lan.WindowBytes != 4<<20 {
		t.Fatalf("lan policy = %#v", lan)
	}
	wan := Adapt(200*time.Millisecond, 0)
	if wan.ChunkBytes != 8<<10 || wan.WindowBytes != 512<<10 {
		t.Fatalf("wan policy = %#v", wan)
	}
	if wan.ChunkBytes > lan.ChunkBytes {
		t.Fatalf("wan chunk %d must not exceed lan chunk %d", wan.ChunkBytes, lan.ChunkBytes)
	}
	// 低带宽链路必须收到不小于 2×BDP 的窗口，同时受硬上限约束。
	lowBandwidth := Adapt(50*time.Millisecond, 1<<20)
	bdp := int64(50*time.Millisecond) * (1 << 20) / int64(time.Second)
	if int64(lowBandwidth.WindowBytes) < bdp {
		t.Fatalf("window %d below BDP %d", lowBandwidth.WindowBytes, bdp)
	}
	huge := Adapt(time.Nanosecond, 1<<40)
	if huge.WindowBytes > MaxWindowBytes {
		t.Fatalf("window %d exceeds hard cap", huge.WindowBytes)
	}
	normalized := Policy{ChunkBytes: 1, WindowBytes: 1 << 30}.Normalized()
	if normalized.ChunkBytes != MinChunkBytes || normalized.WindowBytes != MaxWindowBytes {
		t.Fatalf("normalized = %#v", normalized)
	}
	narrow := Policy{ChunkBytes: 128 << 10, WindowBytes: 64 << 10}.Normalized()
	if narrow.WindowBytes < narrow.ChunkBytes {
		t.Fatalf("window must cover one chunk: %#v", narrow)
	}
}

func TestCoalescerMergesWithinWindowAndEmitsCompletion(t *testing.T) {
	coalescer := NewCoalescer(250 * time.Millisecond)
	start := time.Unix(1700000000, 0)
	progress, emit := coalescer.Observe(0, 1000, start)
	if !emit || progress.Transferred != 0 {
		t.Fatalf("first observe must emit: %#v emit=%v", progress, emit)
	}
	_, emit = coalescer.Observe(100, 1000, start.Add(50*time.Millisecond))
	if emit {
		t.Fatal("observe inside window must be merged")
	}
	progress, emit = coalescer.Observe(400, 1000, start.Add(300*time.Millisecond))
	if !emit {
		t.Fatal("observe after window must emit")
	}
	if progress.RateBytesPerSec <= 0 || progress.RemainingSeconds <= 0 {
		t.Fatalf("progress must carry rate/remaining: %#v", progress)
	}
	_, emit = coalescer.Observe(500, 1000, start.Add(350*time.Millisecond))
	if emit {
		t.Fatal("observe inside window must be merged")
	}
	progress, emit = coalescer.Observe(1000, 1000, start.Add(360*time.Millisecond))
	if !emit || progress.State != StateCompleted || progress.RemainingSeconds != 0 {
		t.Fatalf("completion must always emit: %#v emit=%v", progress, emit)
	}
}

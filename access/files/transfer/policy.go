package transfer

import "time"

// 窗口/分片边界。它们是单条连接上的内存与 RTT 权衡，不是协议常量：
// 客户端只通过 file transfer handle 的 chunk/window 字段感知。
const (
	MinChunkBytes  = 4 << 10
	MaxChunkBytes  = 256 << 10
	MinWindowBytes = 64 << 10
	MaxWindowBytes = 8 << 20
)

// Policy 描述一次传输使用的分片与窗口大小。
type Policy struct {
	ChunkBytes  int
	WindowBytes int
}

// DefaultPolicy 返回同机/低延迟默认：大窗口、大分片。
func DefaultPolicy() Policy {
	return Policy{ChunkBytes: 64 << 10, WindowBytes: 4 << 20}
}

// Normalized 把越界值收敛到硬边界，保证任何链路下内存占用有界。
func (policy Policy) Normalized() Policy {
	if policy.ChunkBytes < MinChunkBytes {
		policy.ChunkBytes = MinChunkBytes
	}
	if policy.ChunkBytes > MaxChunkBytes {
		policy.ChunkBytes = MaxChunkBytes
	}
	if policy.WindowBytes < MinWindowBytes {
		policy.WindowBytes = MinWindowBytes
	}
	if policy.WindowBytes > MaxWindowBytes {
		policy.WindowBytes = MaxWindowBytes
	}
	if policy.WindowBytes < policy.ChunkBytes {
		policy.WindowBytes = policy.ChunkBytes
	}
	return policy
}

// Adapt 按 RTT 与估算带宽选择窗口/分片：
//   - 局域网（RTT≤2ms）用 64KiB/4MiB；
//   - 高延迟链路缩小分片，避免单帧放大延迟；
//   - 已知带宽时窗口按 2×BDP 收敛，保证吞吐而不无限占用内存。
func Adapt(rtt time.Duration, bandwidthBytesPerSec int64) Policy {
	if rtt < 0 {
		rtt = 0
	}
	var policy Policy
	switch {
	case rtt <= 2*time.Millisecond:
		policy = Policy{ChunkBytes: 64 << 10, WindowBytes: 4 << 20}
	case rtt <= 30*time.Millisecond:
		policy = Policy{ChunkBytes: 32 << 10, WindowBytes: 2 << 20}
	case rtt <= 120*time.Millisecond:
		policy = Policy{ChunkBytes: 16 << 10, WindowBytes: 1 << 20}
	default:
		policy = Policy{ChunkBytes: 8 << 10, WindowBytes: 512 << 10}
	}
	if bandwidthBytesPerSec > 0 {
		bdp := int64(rtt) * bandwidthBytesPerSec / int64(time.Second)
		if bdp <= 0 {
			bdp = 1
		}
		target := bdp * 2
		if target > int64(MaxWindowBytes) {
			target = MaxWindowBytes
		}
		if target > 0 && target < int64(policy.WindowBytes) {
			policy.WindowBytes = int(target)
		}
	}
	return policy.Normalized()
}

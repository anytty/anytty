package files

import (
	"sync"
	"time"

	"github.com/anytty/anytty/access/files/transfer"
)

// 历史 pool 默认值：在所有测量缺失时保持客户端已适应的窗口/分片。
const (
	baseWindowBytes = 1 << 20
	baseChunkBytes  = 64 << 10
)

// estimator 从下载 ack 节奏估计 RTT 与吞吐，并据此选择窗口/分片。
// 它只下调历史默认值（不放大），避免改变已上架客户端的既有假设。
type estimator struct {
	mu            sync.Mutex
	rtt           time.Duration
	bandwidth     int64
	lastSendAt    time.Time
	lastSentBytes int64
	lastAckAt     time.Time
	lastAckOffset int64
	samples       int
}

func newEstimator() *estimator {
	return &estimator{}
}

// observeDownloadSend 记录一次数据发送，供下一次 ack 估计 RTT。
func (estimator *estimator) observeDownloadSend(sentBytes int64) {
	estimator.mu.Lock()
	estimator.lastSendAt = time.Now()
	estimator.lastSentBytes = sentBytes
	estimator.mu.Unlock()
}

// observeAck 记录一次客户端 ack，更新 RTT/吞吐 EWMA。
func (estimator *estimator) observeAck(offset int64) {
	now := time.Now()
	estimator.mu.Lock()
	defer estimator.mu.Unlock()
	if !estimator.lastSendAt.IsZero() && estimator.lastSentBytes == offset {
		if rtt := now.Sub(estimator.lastSendAt); rtt > 0 {
			if estimator.rtt == 0 {
				estimator.rtt = rtt
			} else {
				estimator.rtt = (estimator.rtt*7 + rtt) / 8
			}
		}
	}
	if !estimator.lastAckAt.IsZero() && offset > estimator.lastAckOffset {
		elapsed := now.Sub(estimator.lastAckAt)
		if elapsed > 0 {
			bandwidth := int64(float64(offset-estimator.lastAckOffset) / elapsed.Seconds())
			if estimator.bandwidth == 0 {
				estimator.bandwidth = bandwidth
			} else {
				estimator.bandwidth = (estimator.bandwidth*7 + bandwidth) / 8
			}
		}
	}
	estimator.lastAckAt = now
	estimator.lastAckOffset = offset
	estimator.samples++
}

// policy 返回当前链路策略；未测量时保持 pool 默认值。
func (estimator *estimator) policy() transfer.Policy {
	estimator.mu.Lock()
	defer estimator.mu.Unlock()
	if estimator.samples < 2 {
		return transfer.Policy{ChunkBytes: baseChunkBytes, WindowBytes: baseWindowBytes}
	}
	policy := transfer.Adapt(estimator.rtt, estimator.bandwidth).Normalized()
	// 只允许收敛到不高于历史默认值，避免放大已上架客户端的内存/窗口假设。
	if policy.WindowBytes > baseWindowBytes {
		policy.WindowBytes = baseWindowBytes
	}
	if policy.ChunkBytes > baseChunkBytes {
		policy.ChunkBytes = baseChunkBytes
	}
	return policy
}

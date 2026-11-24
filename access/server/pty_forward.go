package server

import (
	"context"
	"sync"
	"time"
)

// PTY 泵的默认参数。
const (
	// defaultPTYBatchWindow 是小块合并窗口：同一突发里的多个 PTY 小块合成一帧发。
	defaultPTYBatchWindow = 3 * time.Millisecond
	// defaultPTYMaxBatchBytes 是单帧合并上限。
	defaultPTYMaxBatchBytes = 64 << 10
	// defaultPTYMaxPendingBytes 是读取侧允许积压的上限；超过后以 sync-lost
	// 结束桥接，而不是拖死整条 provider 连接。
	defaultPTYMaxPendingBytes = 4 << 20
)

// ptyForwardEnd 描述泵结束的原因：Err 来自 provider 一侧（关闭/同步丢失），
// Overflow 表示客户端太慢、积压超过上限。
type ptyForwardEnd struct {
	Err      error
	Overflow bool
}

// ptyForwarder 把 provider 的 PTY 字节泵成客户端 stream 帧。
//
// 读取与发送必须解耦：provider 侧的 stream 缓冲只有几十帧，而客户端链路
// 随时可能背压（例如滑动触发的 resize 让 TUI 全屏重绘）。逐帧同步转发会让
// 一次背压立刻填满 provider 缓冲，provider 随后以
// "stream consumer fell behind" 关闭整条连接，所有终端一起掉线。
//
// 这里读取协程只做接收和追加，发送协程按批合并后写出；字节顺序不变，
// 正常情况下不丢字节，只有积压超过上限才以 sync-lost 主动结束，
// 让客户端重新同步。
type ptyForwarder struct {
	receive func(context.Context) ([]byte, error)
	send    func([]byte) error
	onEnd   func(ptyForwardEnd)

	batchWindow time.Duration
	maxBatch    int
	maxPending  int

	mu       sync.Mutex
	pending  []byte
	readErr  error
	overflow bool
}

// Run 启动读取协程并按批发送，直到 provider 结束、积压溢出或 ctx 取消。
// ctx 取消导致的正常退出不回调 onEnd。
func (forwarder *ptyForwarder) Run(ctx context.Context) {
	if forwarder == nil || forwarder.receive == nil || forwarder.send == nil {
		return
	}
	window := forwarder.batchWindow
	if window <= 0 {
		window = defaultPTYBatchWindow
	}
	maxBatch := forwarder.maxBatch
	if maxBatch <= 0 {
		maxBatch = defaultPTYMaxBatchBytes
	}
	maxPending := forwarder.maxPending
	if maxPending <= 0 {
		maxPending = defaultPTYMaxPendingBytes
	}

	signal := make(chan struct{}, 1)
	nudge := func() {
		select {
		case signal <- struct{}{}:
		default:
		}
	}

	// 读取协程只做接收和追加，永远不等待发送。
	go func() {
		for {
			payload, err := forwarder.receive(ctx)
			if !forwarder.record(payload, err, maxPending) {
				nudge()
				return
			}
			nudge()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-signal:
		}
		// 合并窗口：让同一突发里的后续小块一起发出。
		timer := time.NewTimer(window)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		batch, end, ok := forwarder.takeBatch(maxBatch)
		if !ok {
			continue
		}
		if end != nil {
			if forwarder.onEnd != nil {
				forwarder.onEnd(*end)
			}
			return
		}
		if err := forwarder.send(batch); err != nil {
			return
		}
		// 还有积压或读取已结束：继续下一轮，避免信号丢失。
		nudge()
	}
}

// record 追加一块 PTY 输出；返回 false 表示读取侧应当停止。
// 超过积压上限时标记 overflow，由发送侧以 sync-lost 结束桥接。
func (forwarder *ptyForwarder) record(payload []byte, err error, maxPending int) bool {
	forwarder.mu.Lock()
	defer forwarder.mu.Unlock()
	if err != nil {
		forwarder.readErr = err
		return false
	}
	if len(forwarder.pending)+len(payload) > maxPending {
		forwarder.overflow = true
		return false
	}
	forwarder.pending = append(forwarder.pending, payload...)
	return true
}

// takeBatch 取出下一批待发送字节，或返回结束原因。
// ok=false 表示暂时没有数据，调用方应等待下一次信号。
func (forwarder *ptyForwarder) takeBatch(maxBatch int) ([]byte, *ptyForwardEnd, bool) {
	forwarder.mu.Lock()
	defer forwarder.mu.Unlock()
	if len(forwarder.pending) > 0 {
		if len(forwarder.pending) > maxBatch {
			batch := append([]byte(nil), forwarder.pending[:maxBatch]...)
			forwarder.pending = append([]byte(nil), forwarder.pending[maxBatch:]...)
			return batch, nil, true
		}
		batch := forwarder.pending
		forwarder.pending = nil
		return batch, nil, true
	}
	if forwarder.readErr != nil {
		return nil, &ptyForwardEnd{Err: forwarder.readErr}, true
	}
	if forwarder.overflow {
		return nil, &ptyForwardEnd{Overflow: true}, true
	}
	return nil, nil, false
}

// pendingBytes 仅供测试观察积压量。
func (forwarder *ptyForwarder) pendingBytes() int {
	forwarder.mu.Lock()
	defer forwarder.mu.Unlock()
	return len(forwarder.pending)
}

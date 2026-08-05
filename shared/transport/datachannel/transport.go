// Package datachannel 提供面向可靠有序 DataChannel 的 anytty protocol transport。
package datachannel

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/anytty/anytty/proto/wire"
)

const (
	defaultReceiveQueueCapacity = 256
	maxReceiveQueuedBytes       = 8 << 20
	maxLogicalFrameSize         = wire.MaxEncodedFrameSize
	defaultSendBufferHigh       = 512 * 1024
	defaultSendBufferLow        = 128 * 1024
	defaultDrainTimeout         = 30 * time.Second
	defaultDrainPoll            = time.Millisecond
)

// ErrReceiveQueueExhausted reports that the bounded pre-auth/protocol receive queue was exceeded.
var ErrReceiveQueueExhausted = errors.New("transport/datachannel: receive queue exhausted")

// Channel 是 WebRTC 实现需要适配的可靠有序消息通道。
// 该接口只暴露 protocol frame 传输和 buffered amount 背压，不承担信令、身份验证、grant 校验或 terminal lifecycle。
type Channel interface {
	SetMessageHandler(func([]byte))
	SetCloseHandler(func())
	BufferedAmount() uint64
	SetBufferedAmountLowThreshold(uint64)
	SetBufferedAmountLowHandler(func())
	Send([]byte) error
	Close() error
}

// Transport 把一个已经协商的可靠有序 DataChannel 投影为 message transport。
// 同一实例可以先承载 remote auth envelope，再在 CapabilityAccepted 后承载 anytty frame；调用方在授权成功前不得交给 protocol client 或 core-v2。
type Transport struct {
	channel          Channel
	recvMu           sync.Mutex
	recvQueue        [][]byte
	recvQueuedBytes  int
	recvClosed       bool
	recvErr          error
	recvNotify       chan struct{}
	drainCh          chan struct{}
	done             chan struct{}
	sendMu           sync.Mutex
	channelCloseOnce sync.Once
	closeErr         error
	drainTimeout     time.Duration
}

// New 创建 DataChannel message transport。
// channel 必须保证消息可靠且有序；该构造函数不判断 auth/protocol 阶段、不创建 peer connection，也不提供旧 remote/session-token fallback。
func New(channel Channel) *Transport {
	transport := &Transport{
		channel:      channel,
		recvQueue:    make([][]byte, 0, defaultReceiveQueueCapacity),
		recvNotify:   make(chan struct{}, 1),
		drainCh:      make(chan struct{}, 1),
		done:         make(chan struct{}),
		drainTimeout: defaultDrainTimeout,
	}
	if channel == nil {
		transport.closeDone()
		return transport
	}
	channel.SetMessageHandler(transport.handleMessage)
	channel.SetCloseHandler(transport.closeDone)
	channel.SetBufferedAmountLowThreshold(defaultSendBufferLow)
	channel.SetBufferedAmountLowHandler(func() {
		select {
		case transport.drainCh <- struct{}{}:
		default:
		}
	})
	return transport
}

// Send 发送一个完整 anytty protocol frame。
// 当 DataChannel 缓冲超过高水位时等待低水位通知；超时或关闭会失败，不允许丢帧或切换到其他 transport。
func (transport *Transport) Send(frame []byte) error {
	if len(frame) > maxLogicalFrameSize {
		return wire.ErrFrameTooLarge
	}
	if transport == nil || transport.channel == nil {
		return io.EOF
	}
	for transport.channel.BufferedAmount() > defaultSendBufferHigh {
		timer := time.NewTimer(defaultDrainTimeout)
		select {
		case <-transport.drainCh:
			if !timer.Stop() {
				<-timer.C
			}
		case <-transport.done:
			if !timer.Stop() {
				<-timer.C
			}
			return io.EOF
		case <-timer.C:
			return context.DeadlineExceeded
		}
	}
	transport.sendMu.Lock()
	defer transport.sendMu.Unlock()
	select {
	case <-transport.done:
		return io.EOF
	default:
	}
	return transport.channel.Send(append([]byte(nil), frame...))
}

// OutboundBufferedAmount 返回 DataChannel 尚未发出的字节数，供上层批量流量主动预留控制帧空间。
func (transport *Transport) OutboundBufferedAmount() uint64 {
	if transport == nil || transport.channel == nil {
		return 0
	}
	return transport.channel.BufferedAmount()
}

// Drain 等待已经成功交给 DataChannel 的 outbound message 被底层发送完毕。
// 它只用于 pairing 等“发送最终响应后必须关闭 transport”的协议边界；普通 session 关闭仍走 Close，不能因 drain 阻塞 teardown。
func (transport *Transport) Drain(ctx context.Context) error {
	if transport == nil || transport.channel == nil {
		return io.EOF
	}
	if ctx == nil {
		ctx = context.Background()
	}
	drainCtx, cancel := context.WithTimeout(ctx, transport.drainTimeout)
	defer cancel()
	ticker := time.NewTicker(defaultDrainPoll)
	defer ticker.Stop()
	for {
		transport.sendMu.Lock()
		buffered := transport.channel.BufferedAmount()
		transport.sendMu.Unlock()
		if buffered == 0 {
			return nil
		}
		select {
		case <-drainCtx.Done():
			return drainCtx.Err()
		case <-transport.done:
			return io.EOF
		case <-ticker.C:
		}
	}
}

// Recv 接收一个完整 anytty protocol frame。
// DataChannel 正常关闭时返回 io.EOF；本地接收边界关闭会返回对应的稳定错误。
// 接收内容是独立副本，不能被底层实现后续复用或修改。
func (transport *Transport) Recv() ([]byte, error) {
	if transport == nil {
		return nil, io.EOF
	}
	for {
		transport.recvMu.Lock()
		if transport.recvClosed {
			err := transport.recvErr
			transport.recvMu.Unlock()
			if err != nil {
				return nil, err
			}
			return nil, io.EOF
		}
		if len(transport.recvQueue) > 0 {
			frame := transport.recvQueue[0]
			copy(transport.recvQueue, transport.recvQueue[1:])
			last := len(transport.recvQueue) - 1
			transport.recvQueue[last] = nil
			transport.recvQueue = transport.recvQueue[:last]
			transport.recvQueuedBytes -= len(frame)
			transport.recvMu.Unlock()
			return frame, nil
		}
		transport.recvMu.Unlock()
		select {
		case <-transport.done:
			continue
		case <-transport.recvNotify:
		}
	}
}

// Close 关闭 DataChannel 并解除所有等待中的 Send/Recv。
// 关闭只结束当前 transport，不修改 endpoint registry、reducer state 或 daemon terminal lifecycle。
func (transport *Transport) Close() error {
	if transport == nil {
		return nil
	}
	transport.closeDone()
	transport.closeChannel()
	transport.sendMu.Lock()
	//lint:ignore SA2001 sendMu is an intentional wait barrier for an in-flight Channel.Send.
	transport.sendMu.Unlock()
	return transport.closeErr
}

// Done 返回当前 DataChannel transport 的生命周期结束信号。
func (transport *Transport) Done() <-chan struct{} {
	if transport == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	return transport.done
}

func (transport *Transport) closeDone() {
	transport.closeReceive(nil)
}

func (transport *Transport) handleMessage(payload []byte) {
	if len(payload) > maxLogicalFrameSize {
		transport.closeForReceiveOverflow(wire.ErrFrameTooLarge)
		return
	}
	transport.recvMu.Lock()
	if transport.recvClosed {
		transport.recvMu.Unlock()
		return
	}
	if len(transport.recvQueue) >= defaultReceiveQueueCapacity || len(payload) > maxReceiveQueuedBytes-transport.recvQueuedBytes {
		transport.closeReceiveLocked(ErrReceiveQueueExhausted)
		transport.recvMu.Unlock()
		transport.closeChannel()
		return
	}
	transport.recvQueuedBytes += len(payload)
	frame := append([]byte(nil), payload...)
	transport.recvQueue = append(transport.recvQueue, frame)
	transport.recvMu.Unlock()
	select {
	case transport.recvNotify <- struct{}{}:
	default:
	}
}

func (transport *Transport) closeForReceiveOverflow(err error) {
	transport.closeReceive(err)
	transport.closeChannel()
}

func (transport *Transport) closeReceive(err error) {
	transport.recvMu.Lock()
	transport.closeReceiveLocked(err)
	transport.recvMu.Unlock()
}

func (transport *Transport) closeReceiveLocked(err error) {
	if transport.recvClosed {
		return
	}
	transport.recvClosed = true
	transport.recvErr = err
	for index := range transport.recvQueue {
		transport.recvQueue[index] = nil
	}
	transport.recvQueue = nil
	transport.recvQueuedBytes = 0
	close(transport.done)
}

func (transport *Transport) closeChannel() {
	transport.channelCloseOnce.Do(func() {
		if transport.channel != nil {
			transport.closeErr = transport.channel.Close()
		}
	})
}

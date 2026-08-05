// Package webrtc adapts Pion primitives to remote-v2 transport contracts.
package webrtc

import (
	"sync"

	"github.com/anytty/anytty/shared/transport/datachannel"
	"github.com/pion/webrtc/v4"
)

var _ datachannel.Channel = (*Channel)(nil)

const (
	preHandlerMessageLimit = 32
	preHandlerByteLimit    = 1 << 20
)

// Channel 把 Pion DataChannel 适配为共享 datachannel.Channel。
// 它不创建 peer connection、不验证 grant，也不决定 core-v2 scope；端到端授权属于 daemon DataChannel handler。
type Channel struct {
	channel        pionDataChannel
	dispatchMu     sync.Mutex
	mu             sync.Mutex
	closed         bool
	closeHandlers  []func()
	messageHandler func([]byte)
	pending        [][]byte
	pendingBytes   int
}

type pionDataChannel interface {
	OnClose(func())
	OnMessage(func(webrtc.DataChannelMessage))
	OnBufferedAmountLow(func())
	BufferedAmount() uint64
	SetBufferedAmountLowThreshold(uint64)
	Send([]byte) error
	Close() error
}

// NewChannel 创建 Pion DataChannel adapter。
// 调用方必须确保 DataChannel 使用可靠有序模式，并在 OnOpen 后才交给 protocol session。
func NewChannel(channel *webrtc.DataChannel) *Channel {
	return newChannel(channel)
}

func newChannel(channel pionDataChannel) *Channel {
	adapter := &Channel{channel: channel}
	channel.OnClose(adapter.notifyClosed)
	// Pion 可能在上层 WaitReady 返回前收到 daemon 的 DeviceHello。adapter 必须从创建时就接管消息，
	// 否则可靠有序 DataChannel 的首帧会在 transport 注册 handler 前被静默丢弃。
	channel.OnMessage(adapter.receive)
	return adapter
}

// SetMessageHandler 注册完整 DataChannel message 的接收处理器。
func (channel *Channel) SetMessageHandler(handler func([]byte)) {
	if handler == nil {
		return
	}
	channel.dispatchMu.Lock()
	defer channel.dispatchMu.Unlock()
	channel.mu.Lock()
	if channel.closed {
		channel.mu.Unlock()
		return
	}
	channel.messageHandler = handler
	pending := channel.pending
	channel.pending = nil
	channel.pendingBytes = 0
	channel.mu.Unlock()
	for _, payload := range pending {
		handler(payload)
	}
}

// receive 串行投递 Pion message；handler 尚未装配时只缓存有界握手窗口，溢出即关闭当前 channel。
func (channel *Channel) receive(message webrtc.DataChannelMessage) {
	payload := append([]byte(nil), message.Data...)
	channel.dispatchMu.Lock()
	channel.mu.Lock()
	if channel.closed {
		channel.mu.Unlock()
		channel.dispatchMu.Unlock()
		return
	}
	handler := channel.messageHandler
	if handler == nil {
		if len(channel.pending) >= preHandlerMessageLimit || channel.pendingBytes+len(payload) > preHandlerByteLimit {
			channel.mu.Unlock()
			channel.dispatchMu.Unlock()
			channel.notifyClosed()
			_ = channel.channel.Close()
			return
		}
		channel.pending = append(channel.pending, payload)
		channel.pendingBytes += len(payload)
		channel.mu.Unlock()
		channel.dispatchMu.Unlock()
		return
	}
	channel.mu.Unlock()
	handler(payload)
	channel.dispatchMu.Unlock()
}

// SetCloseHandler 注册底层 DataChannel 关闭处理器。
func (channel *Channel) SetCloseHandler(handler func()) {
	if handler == nil {
		return
	}
	channel.mu.Lock()
	if channel.closed {
		channel.mu.Unlock()
		handler()
		return
	}
	channel.closeHandlers = append(channel.closeHandlers, handler)
	channel.mu.Unlock()
}

// BufferedAmount 返回 Pion 当前待发送字节数。
func (channel *Channel) BufferedAmount() uint64 {
	return channel.channel.BufferedAmount()
}

// SetBufferedAmountLowThreshold 配置共享 transport 的发送低水位。
func (channel *Channel) SetBufferedAmountLowThreshold(threshold uint64) {
	channel.channel.SetBufferedAmountLowThreshold(threshold)
}

// SetBufferedAmountLowHandler 注册发送缓冲降到低水位时的处理器。
func (channel *Channel) SetBufferedAmountLowHandler(handler func()) {
	channel.channel.OnBufferedAmountLow(handler)
}

// Send 发送一个完整 DataChannel message；内容由当前 remote-auth/protocol 状态机解释。
func (channel *Channel) Send(payload []byte) error {
	err := channel.channel.Send(payload)
	if err != nil {
		// 可靠有序 DataChannel 的单帧发送失败后不能继续承载 protocol correlation；
		// 立即发布 transport 终止，让 binding 淘汰精确 session generation。
		channel.notifyClosed()
	}
	return err
}

// Close 关闭当前 Pion DataChannel。
func (channel *Channel) Close() error {
	return channel.channel.Close()
}

func (channel *Channel) notifyClosed() {
	channel.mu.Lock()
	if channel.closed {
		channel.mu.Unlock()
		return
	}
	channel.closed = true
	handlers := append([]func(){}, channel.closeHandlers...)
	channel.closeHandlers = nil
	channel.mu.Unlock()
	for _, handler := range handlers {
		handler()
	}
}

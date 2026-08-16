package transport

import (
	"context"
	"errors"
)

var ErrListenerClosed = errors.New("transport: listener closed")

type Transport interface {
	Send(frame []byte) error
	Recv() ([]byte, error)
	Close() error
	Done() <-chan struct{}
}

// OutboundBufferReporter 是支持底层发送队列观测的可选能力。批量传输可以据此主动让出队列空间，
// 但控制协议不能依赖该值判断传输是否成功。
type OutboundBufferReporter interface {
	OutboundBufferedAmount() uint64
}

type Listener interface {
	Accept(ctx context.Context) (Transport, error)
	Close() error
	Addr() string
}

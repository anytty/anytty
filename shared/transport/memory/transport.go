package memory

import (
	"context"
	"io"
	"sync"

	"github.com/anytty/anytty/shared/transport"
)

// Transport 是同进程测试用 frame transport。
// 它的 truth source 是 NewPair 创建的双向 channel 和两端 done channel，只用于 harness，不代表真实网络 backpressure。
type Transport struct {
	incoming <-chan []byte
	outgoing chan<- []byte
	done     chan struct{}
	peerDone <-chan struct{}
	closeFn  func()
	once     sync.Once
}

// NewPair 创建一对内存 frame transport。
// 这个包只用于 harness 和跨模块 protocol 测试；生命周期真值由两端 done channel 表达，
// 不关闭数据 channel，避免 send-on-closed-channel panic 掩盖真实 transport 关闭语义。
func NewPair() (*Transport, *Transport) {
	aToB := make(chan []byte, 256)
	bToA := make(chan []byte, 256)
	aDone := make(chan struct{})
	bDone := make(chan struct{})

	a := &Transport{
		incoming: bToA,
		outgoing: aToB,
		done:     aDone,
		peerDone: bDone,
	}
	b := &Transport{
		incoming: aToB,
		outgoing: bToA,
		done:     bDone,
		peerDone: aDone,
	}
	a.closeFn = func() {
		close(aDone)
	}
	b.closeFn = func() {
		close(bDone)
	}
	return a, b
}

// Send 发送一个完整 frame 到对端。
// 失败条件只来自本端关闭、对端关闭或调用方阻塞上下文之外的 channel backpressure；
// 不用 panic/recover 做控制流，避免测试 transport 吃掉生命周期 bug。
func (t *Transport) Send(frame []byte) error {
	select {
	case <-t.done:
		return io.EOF
	case <-t.peerDone:
		return io.EOF
	default:
	}
	data := append([]byte(nil), frame...)
	select {
	case <-t.done:
		return io.EOF
	case <-t.peerDone:
		return io.EOF
	case t.outgoing <- data:
		return nil
	}
}

// Recv 接收对端发送的完整 frame。
// 本端关闭立即返回 io.EOF；对端关闭时先排空其成功 Send 的 frame，再返回 EOF。
// 返回的 buffer 所有权交给接收方，peerDone 不能越过已经入队的数据成为第二份顺序真值。
func (t *Transport) Recv() ([]byte, error) {
	select {
	case <-t.done:
		return nil, io.EOF
	default:
	}

	// 先取已经入队的 frame，避免 peerDone 与 incoming 同时就绪时随机丢弃 write-before-close 数据。
	select {
	case frame, ok := <-t.incoming:
		if !ok {
			return nil, io.EOF
		}
		// Send already clones before enqueuing, so the receiver owns this buffer
		// and can consume it directly without another copy.
		return frame, nil
	default:
	}

	select {
	case <-t.done:
		return nil, io.EOF
	case frame, ok := <-t.incoming:
		if !ok {
			return nil, io.EOF
		}
		return frame, nil
	case <-t.peerDone:
		// Close 只发布“不会再有新 frame”；已经成功 Send 的 frame 仍必须先交付。
		select {
		case frame, ok := <-t.incoming:
			if !ok {
				return nil, io.EOF
			}
			return frame, nil
		default:
			return nil, io.EOF
		}
	}
}

// Close 关闭本端 memory transport。
// 它只关闭本端 done channel；对端通过 peerDone 观察生命周期，不依赖关闭共享数据 channel。
func (t *Transport) Close() error {
	t.once.Do(t.closeFn)
	return nil
}

// Done 返回本端关闭信号。
// protocol harness 用它区分正常阻塞 Recv 和 transport 生命周期结束。
func (t *Transport) Done() <-chan struct{} {
	return t.done
}

// Listener 是 memory transport 的测试 listener。
// 它不代表真实网络监听，只用于 harness 在同进程内建立一对 frame transport。
type Listener struct {
	ch   chan transport.Transport
	done chan struct{}
	addr string
	once sync.Once
}

// NewListener 创建一个同进程 memory listener。
// addr 只用于测试诊断展示，不参与路由或 daemon identity。
func NewListener(addr string) *Listener {
	return &Listener{
		ch:   make(chan transport.Transport),
		done: make(chan struct{}),
		addr: addr,
	}
}

// Accept 等待 memory Dial 提交的 server 端 transport。
// 取消、listener 关闭和成功接收是唯一三种结果；不会启动额外 goroutine。
func (l *Listener) Accept(ctx context.Context) (transport.Transport, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-l.done:
		return nil, transport.ErrListenerClosed
	case conn, ok := <-l.ch:
		if !ok {
			return nil, transport.ErrListenerClosed
		}
		return conn, nil
	}
}

// Close 关闭 listener，并解除阻塞中的 Accept/Dial。
func (l *Listener) Close() error {
	l.once.Do(func() {
		close(l.done)
	})
	return nil
}

// Addr 返回测试 listener 的诊断地址。
func (l *Listener) Addr() string {
	return l.addr
}

// Dial 创建 client/server memory transport pair，并把 server 端交给 Accept。
// 该调用必须显式传入 ctx；listener 已关闭或 ctx 取消时返回错误，不能 panic 或永久阻塞。
func (l *Listener) Dial(ctx context.Context) (transport.Transport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, server := NewPair()
	select {
	case <-ctx.Done():
		_ = client.Close()
		_ = server.Close()
		return nil, ctx.Err()
	case <-l.done:
		_ = client.Close()
		_ = server.Close()
		return nil, transport.ErrListenerClosed
	case l.ch <- server:
		return client, nil
	}
}

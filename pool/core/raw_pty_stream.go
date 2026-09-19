package core

import (
	"context"
	"errors"
	"io"
	"sync"
)

const rawPTYStreamQueueChunks = 16

var errRawPTYStreamOverflow = errors.New("raw PTY stream consumer fell behind")

// rawPTYSubscription 是单个 attachment 的有界、有序 PTY byte stream。
// 队列溢出会关闭订阅并报告 dropped bytes；调用方不得把它当成可继续解析的流。
type rawPTYSubscription struct {
	chunks   chan []byte
	closedCh chan struct{}

	mu           sync.Mutex
	terminalErr  error
	droppedBytes uint64
	exitCode     *int
	closed       bool
}

func newRawPTYSubscription() *rawPTYSubscription {
	return &rawPTYSubscription{
		chunks:   make(chan []byte, rawPTYStreamQueueChunks),
		closedCh: make(chan struct{}),
	}
}

func (subscription *rawPTYSubscription) receive(ctx context.Context) ([]byte, error) {
	if subscription == nil {
		return nil, io.EOF
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case chunk, ok := <-subscription.chunks:
		if ok {
			return chunk, nil
		}
		subscription.mu.Lock()
		err := subscription.terminalErr
		subscription.mu.Unlock()
		if err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
}

func (subscription *rawPTYSubscription) termination() (droppedBytes uint64, exitCode *int) {
	if subscription == nil {
		return 0, nil
	}
	subscription.mu.Lock()
	defer subscription.mu.Unlock()
	if subscription.exitCode != nil {
		code := *subscription.exitCode
		exitCode = &code
	}
	return subscription.droppedBytes, exitCode
}

// close 只能在 owning broadcaster 的锁内调用。
func (subscription *rawPTYSubscription) close(err error, droppedBytes uint64, exitCode *int) {
	if subscription == nil {
		return
	}
	subscription.mu.Lock()
	if subscription.closed {
		subscription.mu.Unlock()
		return
	}
	subscription.closed = true
	subscription.terminalErr = err
	subscription.droppedBytes = droppedBytes
	if exitCode != nil {
		code := *exitCode
		subscription.exitCode = &code
	}
	close(subscription.chunks)
	close(subscription.closedCh)
	subscription.mu.Unlock()
}

type rawPTYBroadcaster struct {
	mu          sync.Mutex
	nextID      uint64
	subscribers map[uint64]*rawPTYSubscription
	closed      bool
	exitCode    *int
}

func newRawPTYBroadcaster() *rawPTYBroadcaster {
	return &rawPTYBroadcaster{subscribers: make(map[uint64]*rawPTYSubscription)}
}

func (broadcaster *rawPTYBroadcaster) subscribe(ctx context.Context) *rawPTYSubscription {
	if ctx == nil {
		ctx = context.Background()
	}
	subscription := newRawPTYSubscription()
	broadcaster.mu.Lock()
	if broadcaster.closed {
		subscription.close(nil, 0, broadcaster.exitCode)
		broadcaster.mu.Unlock()
		return subscription
	}
	broadcaster.nextID++
	id := broadcaster.nextID
	broadcaster.subscribers[id] = subscription
	broadcaster.mu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
			broadcaster.unsubscribe(id, ctx.Err())
		case <-subscription.closedCh:
		}
	}()
	return subscription
}

// publish 保留 PTY read chunk 的字节内容和顺序，但不让慢客户端反压 PTY reader。
// 单个客户端超过有界队列后会被明确终止，其他客户端和 terminal 继续运行。
func (broadcaster *rawPTYBroadcaster) publish(raw []byte) {
	if broadcaster == nil || len(raw) == 0 {
		return
	}
	broadcaster.mu.Lock()
	defer broadcaster.mu.Unlock()
	if broadcaster.closed {
		return
	}
	for id, subscription := range broadcaster.subscribers {
		chunk := append([]byte(nil), raw...)
		select {
		case subscription.chunks <- chunk:
		default:
			delete(broadcaster.subscribers, id)
			subscription.close(errRawPTYStreamOverflow, uint64(len(raw)), nil)
		}
	}
}

func (broadcaster *rawPTYBroadcaster) close(exitCode *int) {
	if broadcaster == nil {
		return
	}
	broadcaster.mu.Lock()
	defer broadcaster.mu.Unlock()
	if broadcaster.closed {
		return
	}
	broadcaster.closed = true
	if exitCode != nil {
		code := *exitCode
		broadcaster.exitCode = &code
	}
	for id, subscription := range broadcaster.subscribers {
		delete(broadcaster.subscribers, id)
		subscription.close(nil, 0, broadcaster.exitCode)
	}
}

func (broadcaster *rawPTYBroadcaster) unsubscribe(id uint64, err error) {
	if broadcaster == nil {
		return
	}
	broadcaster.mu.Lock()
	defer broadcaster.mu.Unlock()
	subscription := broadcaster.subscribers[id]
	if subscription == nil {
		return
	}
	delete(broadcaster.subscribers, id)
	subscription.close(err, 0, nil)
}

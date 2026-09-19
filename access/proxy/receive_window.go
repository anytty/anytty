package proxy

import (
	"context"
	"errors"
	"io"
	"math"
	"sync"
	"time"
)

// receiveWindow 只对“已被目标 socket 消费”的字节返还 credit；每个资源独立，
// 窗口耗尽不阻塞 protocol receive。
type receiveWindow struct {
	mu                          sync.Mutex
	maximum, sent, acknowledged int64
	changed                     chan struct{}
	done                        chan struct{}
	closeOnce                   sync.Once
}

func newReceiveWindow(size uint32) *receiveWindow {
	return &receiveWindow{maximum: int64(size), changed: make(chan struct{}, 1), done: make(chan struct{})}
}

func (window *receiveWindow) available(limit int, timeout time.Duration) (int, error) {
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case <-window.done:
			return 0, io.EOF
		default:
		}
		window.mu.Lock()
		credit := window.maximum - (window.sent - window.acknowledged)
		window.mu.Unlock()
		if credit > 0 {
			return min(limit, int(credit)), nil
		}
		if timer == nil {
			timer = time.NewTimer(timeout)
		}
		select {
		case <-window.done:
			return 0, io.EOF
		case <-timer.C:
			return 0, context.DeadlineExceeded
		case <-window.changed:
		}
	}
}

func (window *receiveWindow) recordSent(count int) error {
	window.mu.Lock()
	defer window.mu.Unlock()
	if count < 0 || int64(count) > math.MaxInt64-window.sent ||
		int64(count) > window.maximum-(window.sent-window.acknowledged) {
		return errors.New("browser receive window exceeded")
	}
	window.sent += int64(count)
	return nil
}

func (window *receiveWindow) acknowledge(offset, credit int64) error {
	window.mu.Lock()
	defer window.mu.Unlock()
	if offset < window.acknowledged || offset > window.sent || credit != offset-window.acknowledged {
		return errors.New("invalid browser receive acknowledgement")
	}
	window.acknowledged = offset
	if credit > 0 {
		select {
		case window.changed <- struct{}{}:
		default:
		}
	}
	return nil
}

func (window *receiveWindow) close() { window.closeOnce.Do(func() { close(window.done) }) }

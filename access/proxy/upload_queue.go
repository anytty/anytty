package proxy

import (
	"errors"
	"io"
	"sync"
)

// uploadQueue 是固定 ring：阻塞的目标写入不会释放 credit，也不会在接收端
// 继续到达时累积 per-frame allocation。
type uploadQueue struct {
	mu         sync.Mutex
	buffer     []byte
	head, size int
	closed     bool
	changed    chan struct{}
}

func newUploadQueue(capacity int) *uploadQueue {
	return &uploadQueue{buffer: make([]byte, capacity), changed: make(chan struct{}, 1)}
}

func (q *uploadQueue) enqueue(payload []byte) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return io.EOF
	}
	if len(payload) > len(q.buffer)-q.size {
		return errors.New("browser upload exceeds negotiated credit")
	}
	tail := (q.head + q.size) % len(q.buffer)
	first := copy(q.buffer[tail:], payload)
	copy(q.buffer, payload[first:])
	q.size += len(payload)
	select {
	case q.changed <- struct{}{}:
	default:
	}
	return nil
}

func (q *uploadQueue) peek(buffer []byte) (int, error) {
	for {
		q.mu.Lock()
		if q.closed {
			q.mu.Unlock()
			return 0, io.EOF
		}
		if q.size > 0 {
			count := min(len(buffer), q.size)
			first := copy(buffer[:count], q.buffer[q.head:min(q.head+count, len(q.buffer))])
			copy(buffer[first:count], q.buffer[:count-first])
			q.mu.Unlock()
			return count, nil
		}
		q.mu.Unlock()
		<-q.changed
	}
}

func (q *uploadQueue) consume(count int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.head = (q.head + count) % len(q.buffer)
	q.size -= count
}

func (q *uploadQueue) close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	select {
	case q.changed <- struct{}{}:
	default:
	}
}

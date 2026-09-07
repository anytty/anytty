package core

import (
	"errors"
	"io"
	"sync"
)

// The fixed ring includes the chunk being written, so a blocked target cannot
// free credit or accumulate per-frame allocations while the receiver continues.
type browserUploadQueue struct {
	mu         sync.Mutex
	buffer     []byte
	head, size int
	closed     bool
	changed    chan struct{}
}

func newBrowserUploadQueue(capacity int) *browserUploadQueue {
	return &browserUploadQueue{buffer: make([]byte, capacity), changed: make(chan struct{}, 1)}
}

func (q *browserUploadQueue) enqueue(payload []byte) error {
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

func (q *browserUploadQueue) peek(buffer []byte) (int, error) {
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

func (q *browserUploadQueue) consume(count int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.head = (q.head + count) % len(q.buffer)
	q.size -= count
}

func (q *browserUploadQueue) close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	select {
	case q.changed <- struct{}{}:
	default:
	}
}

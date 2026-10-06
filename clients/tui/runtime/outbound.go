package runtime

import (
	"errors"
	"io"
	"sync"

	wire "github.com/anytty/anytty/proto/ui"
	gproto "google.golang.org/protobuf/proto"
)

// outputClass controls the bounded host -> program queue. Control and
// reliable input frames are never discarded. State/stream data may replace a
// queued frame of the same kind when a slow program falls behind.
type outputClass uint8

const (
	outputControl outputClass = iota
	outputReliable
	outputCoalescible
)

type outboundFrame struct {
	typ      wire.Type
	bytes    []byte
	class    outputClass
	coalesce string
}

type outboundWriter struct {
	w       io.Writer
	max     uint32
	limit   int
	mu      sync.Mutex
	cond    *sync.Cond
	queue   []outboundFrame
	closed  bool
	done    chan struct{}
	closeMu sync.Once
}

func newOutboundWriter(w io.Writer, max uint32, limit int) *outboundWriter {
	if limit < 1 {
		limit = 1
	}
	o := &outboundWriter{w: w, max: max, limit: limit, done: make(chan struct{})}
	o.cond = sync.NewCond(&o.mu)
	go o.run()
	return o
}

func (o *outboundWriter) enqueue(t wire.Type, m gproto.Message, class outputClass, key string) error {
	frame, err := wire.Marshal(t, m, o.max)
	if err != nil {
		return err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	for !o.closed && len(o.queue) >= o.limit {
		if class == outputCoalescible {
			for i := len(o.queue) - 1; i >= 0; i-- {
				if o.queue[i].class == outputCoalescible && o.queue[i].coalesce == key {
					o.queue[i].bytes = frame
					return nil
				}
			}
			// Preserve a newer state snapshot even when another coalescible kind
			// occupies the bounded queue. Reliable/control frames are never
			// evicted; if the queue contains only those, this transient update
			// may be dropped and the next state change will retry it.
			for i := range o.queue {
				if o.queue[i].class == outputCoalescible {
					o.queue[i] = outboundFrame{typ: t, bytes: frame, class: class, coalesce: key}
					return nil
				}
			}
			return nil
		}
		o.cond.Wait()
	}
	if o.closed {
		return io.ErrClosedPipe
	}
	o.queue = append(o.queue, outboundFrame{typ: t, bytes: frame, class: class, coalesce: key})
	o.cond.Signal()
	return nil
}

func (o *outboundWriter) run() {
	defer close(o.done)
	for {
		o.mu.Lock()
		for len(o.queue) == 0 && !o.closed {
			o.cond.Wait()
		}
		if len(o.queue) == 0 && o.closed {
			o.mu.Unlock()
			return
		}
		frame := o.queue[0]
		o.queue[0] = outboundFrame{}
		o.queue = o.queue[1:]
		o.cond.Broadcast()
		o.mu.Unlock()
		_, _ = writeFull(o.w, frame.bytes)
	}
}

func writeFull(w io.Writer, p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		n, err := w.Write(p)
		total += n
		p = p[n:]
		if err != nil {
			return total, err
		}
		if n == 0 {
			return total, errors.New("runtime: zero-length outbound write")
		}
	}
	return total, nil
}

func (o *outboundWriter) close() {
	o.closeMu.Do(func() {
		o.mu.Lock()
		o.closed = true
		o.queue = nil
		o.cond.Broadcast()
		o.mu.Unlock()
		<-o.done
	})
}

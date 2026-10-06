package runtime

import (
	"bytes"
	"sync"
	"testing"
	"time"

	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

type gatedWriter struct {
	mu      sync.Mutex
	buf     bytes.Buffer
	started chan struct{}
	release chan struct{}
}

func newGatedWriter() *gatedWriter {
	return &gatedWriter{started: make(chan struct{}, 1), release: make(chan struct{})}
}

func (w *gatedWriter) Write(p []byte) (int, error) {
	select {
	case w.started <- struct{}{}:
	default:
	}
	<-w.release
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *gatedWriter) Bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]byte(nil), w.buf.Bytes()...)
}

func TestOutboundWriterCoalescesAndPreservesControl(t *testing.T) {
	w := newGatedWriter()
	o := newOutboundWriter(w, 1<<20, 1)
	defer o.close()
	if err := o.enqueue(wire.TypeEvent, &pb.Event{Event: &pb.Event_Notice{Notice: &pb.NoticeEvent{Message: "one"}}}, outputCoalescible, "notice"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.started:
	case <-time.After(time.Second):
		t.Fatal("writer did not start")
	}
	// The first frame is in the blocked writer, so the bounded queue contains
	// one notice. The next notice replaces it instead of growing the queue.
	if err := o.enqueue(wire.TypeEvent, &pb.Event{Event: &pb.Event_Notice{Notice: &pb.NoticeEvent{Message: "two"}}}, outputCoalescible, "notice"); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- o.enqueue(wire.TypeResponse, &pb.Response{RequestId: 7, Epoch: 1, Ok: true}, outputControl, "response")
	}()
	select {
	case err := <-done:
		t.Fatalf("control enqueue returned before capacity was available: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(w.release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("control enqueue did not complete after writer resumed")
	}

	var types []wire.Type
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		types = types[:0]
		dec := wire.NewDecoder(bytes.NewReader(w.Bytes()), wire.RoleProgram, 0)
		for {
			typ, _, err := dec.Decode()
			if err != nil {
				break
			}
			types = append(types, typ)
		}
		if len(types) >= 3 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if len(types) != 3 || types[0] != wire.TypeEvent || types[1] != wire.TypeEvent || types[2] != wire.TypeResponse {
		t.Fatalf("written frame types = %v, want two events then response", types)
	}
}

func TestSessionOutputQueueBoundsSlowProgram(t *testing.T) {
	w := newGatedWriter()
	s := NewSession(Options{ViewID: "bench", Cols: 80, Rows: 24, OutputQueue: 4}, bytes.NewReader(nil), w)
	if err := s.SendHello(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.started:
	case <-time.After(time.Second):
		t.Fatal("queued writer did not start")
	}
	start := time.Now()
	for i := 0; i < 10000; i++ {
		if err := s.SendNotice("info", "notice"); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("10k coalescible notices took %v with a blocked reader", elapsed)
	}
	close(w.release)
	s.Close()
}

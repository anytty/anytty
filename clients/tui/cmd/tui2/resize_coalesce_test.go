package main

import (
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/clients/tui/runtime"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/ui/protobuf"
)

// coalescingPTY records resize calls and blocks until released, so a burst can
// be observed while one resize is in flight.
type coalescingPTY struct {
	*fakePTY
	mu       sync.Mutex
	sizes    [][2]int
	entered  chan struct{}
	release  chan struct{}
	blocking bool
}

func (p *coalescingPTY) Resize(cols, rows int) error {
	p.mu.Lock()
	p.sizes = append(p.sizes, [2]int{cols, rows})
	entered := p.entered
	release := p.release
	blocking := p.blocking
	p.mu.Unlock()
	if blocking && entered != nil {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
	}
	return p.fakePTY.Resize(cols, rows)
}

func (p *coalescingPTY) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.sizes)
}

// A drag emits one resize per frame, but the host must not run one blocking
// round-trip per frame: while a resize is in flight newer requests coalesce and
// only the last size is applied.
func TestResizeCoalescesDuringDrag(t *testing.T) {
	proc := &coalescingPTY{
		fakePTY: newFakePTY(), entered: make(chan struct{}, 8),
		release: make(chan struct{}), blocking: true,
	}
	handler := runtime.NewTerminalHandler(runtime.TerminalOptions{Cols: 80, Rows: 24, NewPTY: func(pty.Config) pty.PTY { return proc }})
	defer handler.Close()
	out, _ := handler.Handle(runtime.Request{Method: runtime.Method{Name: "terminal.attach"}, Params: &protobuf.MethodParams{Endpoint: "local", Id: "r"}})
	if !out.OK {
		t.Fatal(out.Error)
	}
	term, _ := handler.TerminalBySource(runtime.SourceID("local", "r"))
	host := &Host{}

	// First resize enters and blocks.
	host.applyResize(term, term.SourceID(), 100, 30)
	select {
	case <-proc.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first resize never started")
	}
	// A drag burst while the first is in flight must coalesce, not queue.
	for cols := 101; cols <= 120; cols++ {
		host.applyResize(term, term.SourceID(), cols, 30)
	}
	if got := proc.count(); got != 1 {
		t.Fatalf("resize calls while in flight = %d, want 1 (coalesced)", got)
	}
	close(proc.release)
	// Exactly one follow-up applies the newest size.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		proc.mu.Lock()
		sizes := append([][2]int(nil), proc.sizes...)
		proc.mu.Unlock()
		if len(sizes) >= 2 {
			if sizes[len(sizes)-1] != [2]int{120, 30} {
				t.Fatalf("final coalesced size = %v, want [120 30] (all sizes=%v)", sizes[len(sizes)-1], sizes)
			}
			if len(sizes) > 2 {
				t.Fatalf("burst produced %d resize calls, want at most 2: %v", len(sizes), sizes)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("coalesced follow-up resize never ran")
}

var _ = apipb.TerminalRef{}

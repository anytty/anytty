package main

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/clients/tui/render"
	"github.com/anytty/anytty/clients/tui/runtime"
	"github.com/anytty/anytty/clients/tui/sdk"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// logCollector is a goroutine-safe sink for host diagnostics (Options.Logf).
type logCollector struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (c *logCollector) logf(format string, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprintf(&c.buf, format+"\n", args...)
}

func (c *logCollector) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

// memPipe is an unbounded in-memory byte stream for the fake TTY and the
// fake program pipes.
type memPipe struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
}

func newMemPipe() *memPipe {
	p := &memPipe{}
	p.cond = sync.NewCond(&p.mu)
	return p
}

func (p *memPipe) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0, io.ErrClosedPipe
	}
	p.buf = append(p.buf, b...)
	p.cond.Broadcast()
	return len(b), nil
}

func (p *memPipe) Read(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(p.buf) == 0 && !p.closed {
		p.cond.Wait()
	}
	if len(p.buf) == 0 {
		return 0, io.EOF
	}
	n := copy(b, p.buf)
	p.buf = p.buf[n:]
	return n, nil
}

func (p *memPipe) Close() error {
	p.mu.Lock()
	p.closed = true
	p.cond.Broadcast()
	p.mu.Unlock()
	return nil
}

// syncBuffer is a goroutine-safe output sink for the host TTY writes.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// fakePTY is an injectable PTY: emitted bytes become terminal output, written
// bytes are recorded as program input and the size is observable.
type fakePTY struct {
	mu     sync.Mutex
	input  bytes.Buffer
	outR   *io.PipeReader
	outW   *io.PipeWriter
	cols   int
	rows   int
	closed bool
}

func newFakePTY() *fakePTY {
	r, w := io.Pipe()
	return &fakePTY{outR: r, outW: w, cols: 80, rows: 24}
}

func (p *fakePTY) Start() error { return nil }
func (p *fakePTY) Read(b []byte) (int, error) {
	return p.outR.Read(b)
}
func (p *fakePTY) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.input.Write(b)
}
func (p *fakePTY) Resize(cols, rows int) error {
	p.mu.Lock()
	p.cols, p.rows = cols, rows
	p.mu.Unlock()
	return nil
}
func (p *fakePTY) Size() (int, int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cols, p.rows, nil
}
func (p *fakePTY) ExitCode() int { return -1 }
func (p *fakePTY) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.closed {
		p.closed = true
		_ = p.outW.Close()
	}
	return nil
}
func (p *fakePTY) emit(text string) { _, _ = p.outW.Write([]byte(text)) }
func (p *fakePTY) written() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.input.String()
}
func (p *fakePTY) size() [2]int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return [2]int{p.cols, p.rows}
}

// fakeProcess is the injectable layout program process; the test drives it
// through an sdk.Client on the other end of the pipes.
type fakeProcess struct {
	stdin   io.Writer
	stdout  io.Reader
	stopped chan struct{}
	once    sync.Once
}

func (p *fakeProcess) Stdin() io.Writer  { return p.stdin }
func (p *fakeProcess) Stdout() io.Reader { return p.stdout }
func (p *fakeProcess) Stop()             { p.once.Do(func() { close(p.stopped) }) }

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

// TestHostFrameRoundTrip drives one local session end to end with an
// injected TTY, PTY and layout program: create a terminal, paint its output,
// type into it, open the Ctrl-Q overlay and quit cleanly.
func TestHostFrameRoundTrip(t *testing.T) {
	programIn := newMemPipe()
	programOut := newMemPipe()
	proc := &fakeProcess{stdin: programIn, stdout: programOut, stopped: make(chan struct{})}

	// sourceID and ptyBox are written on the host/client goroutines and read
	// by the test goroutine, so every access goes through stateMu.
	var stateMu sync.Mutex
	var ptyBox *fakePTY
	var sourceID string
	state := func() (string, *fakePTY) {
		stateMu.Lock()
		defer stateMu.Unlock()
		return sourceID, ptyBox
	}

	var client *sdk.Client
	handlers := sdk.Handlers{
		Hello: func(h *pb.Hello) {
			if h.GetEpoch() != 1 || h.GetViewId() != viewID {
				t.Errorf("hello = %+v", h)
			}
			_, _ = client.Emit("terminal.create", &pb.MethodParams{Endpoint: "local"}, func(resp *pb.Response) {
				if !resp.GetOk() {
					t.Errorf("create failed: %s", resp.GetError())
					return
				}
				id := "terminal:" + resp.GetData().GetEndpoint() + ":" + resp.GetData().GetId()
				stateMu.Lock()
				sourceID = id
				stateMu.Unlock()
				tree := sdk.Row(
					sdk.Terminal(id).
						ID("slot-1").
						Width(78).
						Height(18).
						Input("key", "paste", "wheel").
						Props(map[string]string{
							"chrome.border":       "fg:#565f89",
							"chrome.border_focus": "fg:#565f89",
						}).
						Focused(true),
				)
				if err := client.Commit(tree.Build(), sdk.Keys{Claim: []string{"ctrl-p"}}); err != nil {
					t.Errorf("commit: %v", err)
				}
			})
		},
	}
	client = sdk.New(programIn, programOut, handlers)
	go func() { _ = client.Loop() }()

	input := newMemPipe()
	out := &syncBuffer{}
	host := NewHost(Options{
		Shell: []string{"fake-shell"},
		In:    input,
		Out:   out,
		NewProcess: func([]string) (process, error) {
			return proc, nil
		},
		NewPTY: func(pty.Config) pty.PTY {
			box := newFakePTY()
			stateMu.Lock()
			ptyBox = box
			stateMu.Unlock()
			return box
		},
		Cols: 80,
		Rows: 20,
		Tick: 5 * time.Millisecond,
	})

	done := make(chan error, 1)
	go func() { done <- host.Run() }()

	waitFor(t, "layout program hello", func() bool { return client.Hello() != nil })
	waitFor(t, "terminal created", func() bool {
		id, box := state()
		return id != "" && box != nil
	})
	waitFor(t, "view applied", func() bool { return client.Rev() >= 1 })

	_, box := state()
	box.emit("hello from pty\r\n")
	waitFor(t, "frame with pty output", func() bool { return bytes.Contains([]byte(out.String()), []byte("hello from pty")) })
	// The program-declared chrome.border_focus prop survives the
	// kernel/runtime pass-through and reaches the framebuffer as the explicit
	// SGR (#565f89), instead of the host fallback accent.
	waitFor(t, "program chrome props reach the component", func() bool {
		return bytes.Contains([]byte(out.String()), []byte("\x1b[38;2;86;95;137m"))
	})
	_, box = state()
	waitFor(t, "pty resized to the solved rect", func() bool { return box.size() == [2]int{76, 16} })

	if _, err := input.Write([]byte("hi")); err != nil {
		t.Fatalf("write input: %v", err)
	}
	waitFor(t, "keystrokes routed to the focused pty", func() bool { return box.written() == "hi" })

	if _, err := input.Write([]byte{0x11}); err != nil {
		t.Fatalf("write ctrl-q: %v", err)
	}
	waitFor(t, "host quit confirmation", func() bool { return bytes.Contains([]byte(out.String()), []byte("Quit tui2?")) })

	if _, err := input.Write([]byte{'\r'}); err != nil {
		t.Fatalf("write enter: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("host did not exit after the confirmation")
	}
	if !bytes.Contains([]byte(out.String()), []byte(render.ExitScreen())) {
		t.Fatal("exit must restore the terminal")
	}
}

// TestHostResizeFollowsProgramReflow pins the SIGWINCH chain end to end: the
// host's external resize reaches the program as a resize event, the program
// recommits a view sized for the new viewport and the PTY winsize follows the
// solved content rect (rect minus the component inset).
func TestHostResizeFollowsProgramReflow(t *testing.T) {
	programIn := newMemPipe()
	programOut := newMemPipe()
	proc := &fakeProcess{stdin: programIn, stdout: programOut, stopped: make(chan struct{})}

	var stateMu sync.Mutex
	var ptyBox *fakePTY
	var sourceID string
	var client *sdk.Client

	commitSlot := func(cols, rows int) {
		tree := sdk.Row(
			sdk.Terminal(sourceID).
				ID("slot-1").
				Width(cols).
				Height(rows).
				Input("key", "paste", "wheel").
				Focused(true),
		)
		if err := client.Commit(tree.Build(), sdk.Keys{Claim: []string{"ctrl-p"}}); err != nil {
			t.Errorf("commit: %v", err)
		}
	}

	handlers := sdk.Handlers{
		Hello: func(h *pb.Hello) {
			_, _ = client.Emit("terminal.create", &pb.MethodParams{Endpoint: "local"}, func(resp *pb.Response) {
				if !resp.GetOk() {
					t.Errorf("create failed: %s", resp.GetError())
					return
				}
				stateMu.Lock()
				sourceID = "terminal:" + resp.GetData().GetEndpoint() + ":" + resp.GetData().GetId()
				stateMu.Unlock()
				commitSlot(int(h.GetCols()), int(h.GetRows()))
			})
		},
		Resize: func(cols, rows int) { commitSlot(cols, rows) },
	}
	client = sdk.New(programIn, programOut, handlers)
	go func() { _ = client.Loop() }()

	host := NewHost(Options{
		Shell: []string{"fake-shell"},
		In:    newMemPipe(),
		Out:   &syncBuffer{},
		NewProcess: func([]string) (process, error) {
			return proc, nil
		},
		NewPTY: func(pty.Config) pty.PTY {
			box := newFakePTY()
			stateMu.Lock()
			ptyBox = box
			stateMu.Unlock()
			return box
		},
		Cols: 60,
		Rows: 20,
		Tick: 5 * time.Millisecond,
	})
	done := make(chan error, 1)
	go func() { done <- host.Run() }()

	waitFor(t, "initial PTY size", func() bool {
		stateMu.Lock()
		defer stateMu.Unlock()
		return ptyBox != nil && ptyBox.size() == [2]int{58, 18}
	})

	// The external TTY shrinks; only the host knows (SIGWINCH -> Host.Resize).
	host.Resize(40, 12)
	waitFor(t, "PTY follows the resized view", func() bool {
		stateMu.Lock()
		defer stateMu.Unlock()
		return ptyBox != nil && ptyBox.size() == [2]int{38, 10}
	})

	host.quit()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("host did not exit")
	}
}

// TestHostRestartsImmediatelyExitingProgram pins the failure path of a bad
// -shell command: the program exits right after start, the diagnostic goes to
// the log sink (never to the frame-only terminal stream) and the host keeps
// applying the restart policy instead of silently spinning.
func TestHostRestartsImmediatelyExitingProgram(t *testing.T) {
	var mu sync.Mutex
	starts := 0
	out := &syncBuffer{}
	logs := &logCollector{}
	host := NewHost(Options{
		Shell: []string{"python3", "missing-shell.py"},
		In:    newMemPipe(),
		Out:   out,
		Logf:  logs.logf,
		NewProcess: func([]string) (process, error) {
			mu.Lock()
			starts++
			mu.Unlock()
			return &fakeProcess{stdin: io.Discard, stdout: bytes.NewReader(nil), stopped: make(chan struct{})}, nil
		},
		NewPTY:      func(pty.Config) pty.PTY { return newFakePTY() },
		RestartWait: 5 * time.Millisecond,
		Tick:        5 * time.Millisecond,
	})
	done := make(chan error, 1)
	go func() { done <- host.Run() }()

	waitFor(t, "immediate-exit diagnostic in the log", func() bool {
		return bytes.Contains([]byte(logs.String()), []byte("exited immediately"))
	})
	waitFor(t, "program restarted per policy", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return starts >= 2
	})
	if bytes.Contains([]byte(out.String()), []byte("exited immediately")) {
		t.Fatal("host diagnostics must never be written to the terminal stream")
	}

	host.quit()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("host did not exit")
	}
}

func TestSnapshotSourcesStableOrder(t *testing.T) {
	host := NewHost(Options{
		Shell: []string{"fake-shell"},
		In:    newMemPipe(),
		Out:   &syncBuffer{},
		NewProcess: func([]string) (process, error) {
			return &fakeProcess{stdin: newMemPipe(), stdout: newMemPipe(), stopped: make(chan struct{})}, nil
		},
		NewPTY: func(pty.Config) pty.PTY { return newFakePTY() },
	})
	defer host.handler.Close()

	for i := 0; i < 3; i++ {
		outcome, pending := host.gate.Handle(runtime.Request{
			Method: runtime.Method{Name: "terminal.create"},
			Params: &pb.MethodParams{Endpoint: "local"},
		})
		if !outcome.OK || pending {
			t.Fatalf("create %d = %+v pending=%v", i, outcome, pending)
		}
	}
	first, _ := host.snapshotSources()
	if len(first) != 3 {
		t.Fatalf("sources = %d, want 3", len(first))
	}
	second, changed := host.snapshotSources()
	if changed {
		t.Fatal("an unchanged snapshot must not be republished")
	}
	for i := range first {
		if first[i].GetId() != second[i].GetId() {
			t.Fatalf("source order changed: %v vs %v", first[i].GetId(), second[i].GetId())
		}
	}
}

func TestHostEscDeniesProgramQuit(t *testing.T) {
	programIn := newMemPipe()
	programOut := newMemPipe()
	proc := &fakeProcess{stdin: programIn, stdout: programOut, stopped: make(chan struct{})}
	var ptyBox *fakePTY

	var client *sdk.Client
	quitCh := make(chan *pb.Response, 1)
	handlers := sdk.Handlers{
		Hello: func(h *pb.Hello) {
			_, _ = client.Emit("system.quit", &pb.MethodParams{}, func(resp *pb.Response) {
				quitCh <- resp
			})
		},
	}
	client = sdk.New(programIn, programOut, handlers)
	go func() { _ = client.Loop() }()

	input := newMemPipe()
	out := &syncBuffer{}
	host := NewHost(Options{
		Shell: []string{"fake-shell"},
		In:    input,
		Out:   out,
		NewProcess: func([]string) (process, error) {
			return proc, nil
		},
		NewPTY: func(pty.Config) pty.PTY {
			ptyBox = newFakePTY()
			return ptyBox
		},
		Cols: 60,
		Rows: 16,
		Tick: 5 * time.Millisecond,
	})
	done := make(chan error, 1)
	go func() { done <- host.Run() }()

	waitFor(t, "confirmation overlay", func() bool {
		return bytes.Contains([]byte(out.String()), []byte("Layout program requests quit?"))
	})
	if _, err := input.Write([]byte{0x1b}); err != nil {
		t.Fatalf("write esc: %v", err)
	}
	var quitResponse *pb.Response
	select {
	case quitResponse = <-quitCh:
	case <-time.After(3 * time.Second):
		t.Fatal("no response to the denied system.quit")
	}
	if quitResponse.GetOk() || quitResponse.GetError() != "denied by user" {
		t.Fatalf("quit response = %+v", quitResponse)
	}
	if host.isQuitting() {
		t.Fatal("deny must not quit the host")
	}
	host.quit()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("host did not exit")
	}
}

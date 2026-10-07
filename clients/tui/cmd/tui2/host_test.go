package main

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/clients/tui/render"
	"github.com/anytty/anytty/clients/tui/runtime"
	"github.com/anytty/anytty/clients/tui/runtime/keys"
	"github.com/anytty/anytty/clients/tui/sdk"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// logCollector is a goroutine-safe sink for host diagnostics (Options.Logf).
type logCollector struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func TestFrameFlushDelayUsesFixedBurstBudget(t *testing.T) {
	now := time.Unix(100, 0)
	if got := frameFlushDelay(time.Time{}, now); got != 0 {
		t.Fatalf("idle frame delay = %s, want immediate", got)
	}
	last := now.Add(-3 * time.Millisecond)
	if got, want := frameFlushDelay(last, now), 5*time.Millisecond; got != want {
		t.Fatalf("burst frame delay = %s, want %s", got, want)
	}
	if got := frameFlushDelay(now.Add(-frameCoalesceWindow), now); got != 0 {
		t.Fatalf("expired frame delay = %s, want immediate", got)
	}
}

func TestInteractionRecentHasBoundedUrgencyWindow(t *testing.T) {
	now := time.Unix(100, 0)
	h := &Host{}
	if h.interactionRecent(now) {
		t.Fatal("unset interaction must not be urgent")
	}
	h.lastInteraction.Store(now.Add(-interactionUrgencyWindow + time.Nanosecond).UnixNano())
	if !h.interactionRecent(now) {
		t.Fatal("recent interaction should be urgent")
	}
	h.lastInteraction.Store(now.Add(-interactionUrgencyWindow - time.Nanosecond).UnixNano())
	if h.interactionRecent(now) {
		t.Fatal("expired interaction should use the coalesced path")
	}
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

func TestCleanupOwnedTerminalsOnlyRemovesEphemeralCreates(t *testing.T) {
	proc := newFakePTY()
	host := NewHost(Options{
		Cols:   80,
		Rows:   24,
		NewPTY: func(pty.Config) pty.PTY { return proc },
	})
	defer host.endpoints.Close()
	ephemeral := true
	outcome, pending := host.gate.create(runtime.Request{
		Epoch:  1,
		Method: runtime.Method{Name: "terminal.create"},
		Params: &pb.MethodParams{Endpoint: "local", Ephemeral: &ephemeral},
	})
	if pending || !outcome.OK {
		t.Fatalf("ephemeral create = %+v pending=%v", outcome, pending)
	}
	host.cleanupOwnedTerminals()
	host.mu.Lock()
	remaining := len(host.tracked)
	host.mu.Unlock()
	if remaining != 0 {
		t.Fatalf("owned terminals remaining after cleanup = %d", remaining)
	}
	if _, ok := host.handler.Terminal("term-1"); ok {
		t.Fatal("ephemeral terminal remained in the handler after cleanup")
	}
	proc.mu.Lock()
	closed := proc.closed
	proc.mu.Unlock()
	if !closed {
		t.Fatal("ephemeral PTY was not closed")
	}
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

type stoppablePipeProcess struct {
	stdin  *io.PipeWriter
	stdout *io.PipeReader
	once   sync.Once
}

func (p *stoppablePipeProcess) Stdin() io.Writer  { return p.stdin }
func (p *stoppablePipeProcess) Stdout() io.Reader { return p.stdout }
func (p *stoppablePipeProcess) Stop() {
	p.once.Do(func() {
		_ = p.stdin.Close()
		_ = p.stdout.Close()
	})
}

func TestStopProgramReleasesBlockedOutputWriter(t *testing.T) {
	programReader, programWriter := io.Pipe()
	proc := &stoppablePipeProcess{stdin: programWriter, stdout: programReader}
	session := runtime.NewSession(runtime.Options{OutputQueue: 1}, bytes.NewReader(nil), programWriter)
	if err := session.SendHello(); err != nil {
		t.Fatal(err)
	}
	host := &Host{proc: proc, session: session}
	done := make(chan struct{})
	go func() {
		host.stopProgram()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stopProgram waited on a blocked session writer")
	}
}

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

// TestHostRawWheelReplay follows the real host boundary rather than calling
// Session.Input directly: raw SGR bytes pass through the host parser and
// pointer preparation, the focused program receives the wheel and asks the
// host to scroll, then a child that enables mouse tracking must still leave
// later wheels with the history reducer while that viewport is frozen.
func TestHostRawWheelReplay(t *testing.T) {
	programIn := newMemPipe()
	programOut := newMemPipe()
	proc := &fakeProcess{stdin: programIn, stdout: programOut, stopped: make(chan struct{})}
	input := newMemPipe()
	hostOut := &syncBuffer{}

	var stateMu sync.Mutex
	var sourceID, terminalID string
	var ptyBox *fakePTY
	var term *runtime.Terminal
	deltas := make(chan int, 8)
	var client *sdk.Client
	state := func() (string, *fakePTY) {
		stateMu.Lock()
		defer stateMu.Unlock()
		return sourceID, ptyBox
	}

	handlers := sdk.Handlers{
		Hello: func(h *pb.Hello) {
			_, _ = client.Emit("terminal.create", &pb.MethodParams{Endpoint: "local"}, func(resp *pb.Response) {
				if !resp.GetOk() {
					t.Errorf("create failed: %s", resp.GetError())
					return
				}
				data := resp.GetData()
				stateMu.Lock()
				terminalID = data.GetId()
				sourceID = "terminal:" + data.GetEndpoint() + ":" + data.GetId()
				stateMu.Unlock()
				tree := sdk.Row(sdk.Terminal(sourceID).ID("term").Width(78).Height(18).
					Input("key", "wheel").Focused(true))
				if err := client.Commit(tree.Build(), sdk.Keys{}); err != nil {
					t.Errorf("commit: %v", err)
				}
			})
		},
		Wheel: func(w *pb.WheelEvent) {
			deltas <- int(w.GetDelta())
			_, _ = client.Emit("terminal.scroll", &pb.MethodParams{
				Endpoint: "local", Id: terminalID, Delta: w.GetDelta(), Rows: 16, View: "term",
			}, nil)
		},
	}
	client = sdk.New(programIn, programOut, handlers)
	go func() { _ = client.Loop() }()

	host := NewHost(Options{
		Shell: []string{"fake-shell"}, In: input, Out: hostOut,
		NewProcess: func([]string) (process, error) { return proc, nil },
		NewPTY: func(pty.Config) pty.PTY {
			box := newFakePTY()
			stateMu.Lock()
			ptyBox = box
			stateMu.Unlock()
			return box
		},
		Cols: 80, Rows: 20, Tick: 5 * time.Millisecond,
	})
	done := make(chan error, 1)
	go func() { done <- host.Run() }()
	defer func() {
		select {
		case <-done:
		default:
			proc.Stop()
			host.stopProgram()
		}
	}()

	waitFor(t, "layout hello", func() bool { return client.Hello() != nil })
	waitFor(t, "terminal view", func() bool {
		id, box := state()
		return id != "" && box != nil && client.Rev() >= 1
	})
	id, box := state()
	term, ok := host.handler.TerminalBySource(id)
	if !ok {
		t.Fatalf("terminal %q not found", id)
	}

	// Seed enough local scrollback for terminal.scroll(+1) to pin a history
	// viewport. The child is initially not tracking mouse input.
	var seed strings.Builder
	for i := 0; i < 32; i++ {
		seed.WriteString("line-")
		seed.WriteString(fmt.Sprint(i))
		seed.WriteString("\r\n")
	}
	box.emit(seed.String())
	if _, err := input.Write([]byte("\x1b[<64;4;4M")); err != nil {
		t.Fatalf("write first wheel: %v", err)
	}
	select {
	case got := <-deltas:
		if got != 1 {
			t.Fatalf("first wheel delta = %d, want +1", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("program did not receive first wheel")
	}
	waitFor(t, "history viewport", func() bool { return term.HistoryActive("term") && term.Offset("term") > 0 })

	// Codex-like children can leave DEC tracking enabled while the host is in
	// copy/history mode. The second, downward wheel must remain a program
	// event, and must never inject ESC[<65 into the child PTY.
	box.emit("\x1b[?1000h\x1b[?1006h")
	waitFor(t, "mouse tracking enabled", func() bool { return term.Modes().MouseTracking() })
	if _, err := input.Write([]byte("\x1b[<65;4;4M")); err != nil {
		t.Fatalf("write second wheel: %v", err)
	}
	select {
	case got := <-deltas:
		if got != -1 {
			t.Fatalf("second wheel delta = %d, want -1", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("history did not receive second wheel")
	}
	waitFor(t, "live bottom after downward wheel", func() bool { return !term.HistoryActive("term") && term.Offset("term") == 0 })
	// Once live is restored, tracking belongs to the child again. A downward
	// wheel must be available to Codex/other mouse-aware terminal programs.
	if _, err := input.Write([]byte("\x1b[<65;4;4M")); err != nil {
		t.Fatalf("write bottom wheel: %v", err)
	}
	waitFor(t, "live tracking wheel reaches PTY", func() bool {
		got := box.written()
		return bytes.Contains([]byte(got), []byte("\x1b[<65")) || bytes.Contains([]byte(got), []byte("\x1b[M"))
	})
	if got := box.written(); !bytes.Contains([]byte(got), []byte("\x1b[<65")) && !bytes.Contains([]byte(got), []byte("\x1b[M")) {
		t.Fatalf("live tracking wheel did not reach PTY: %q", got)
	}

	// Quit the host cleanly so the test also exercises the same input pump's
	// lifecycle instead of leaving the fake program goroutine behind.
	_, _ = input.Write([]byte{0x11})
	waitFor(t, "quit confirmation", func() bool { return bytes.Contains([]byte(hostOut.String()), []byte("Quit tui2?")) })
	_, _ = input.Write([]byte{'\r'})
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("host run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("host did not stop")
	}
}

// TestHostPerViewScrollIsolation binds two terminal boxes ("pane-a"/"pane-b")
// to ONE terminal source and drives a scroll for pane-a through the real host
// boundary (the program's terminal.scroll carries the box id as its view). The
// shared runtime.Terminal must keep one frozen viewport per pane, so pane-a
// freezes while its sibling pane-b stays live.
func TestHostPerViewScrollIsolation(t *testing.T) {
	programIn := newMemPipe()
	programOut := newMemPipe()
	proc := &fakeProcess{stdin: programIn, stdout: programOut, stopped: make(chan struct{})}
	input := newMemPipe()
	hostOut := &syncBuffer{}

	var stateMu sync.Mutex
	var sourceID, terminalID string
	var ptyBox *fakePTY
	var client *sdk.Client
	state := func() (string, string, *fakePTY) {
		stateMu.Lock()
		defer stateMu.Unlock()
		return sourceID, terminalID, ptyBox
	}

	handlers := sdk.Handlers{
		Hello: func(h *pb.Hello) {
			_, _ = client.Emit("terminal.create", &pb.MethodParams{Endpoint: "local"}, func(resp *pb.Response) {
				if !resp.GetOk() {
					t.Errorf("create failed: %s", resp.GetError())
					return
				}
				stateMu.Lock()
				terminalID = resp.GetData().GetId()
				sourceID = "terminal:" + resp.GetData().GetEndpoint() + ":" + resp.GetData().GetId()
				src := sourceID
				stateMu.Unlock()
				// Two panes over one source; both commit their own view == box id.
				tree := sdk.Row(
					sdk.Terminal(src).ID("pane-a").Width(30).Height(10).Input("key", "a"),
					sdk.Terminal(src).ID("pane-b").Width(30).Height(10).Input("key", "b"),
				)
				if err := client.Commit(tree.Build(), sdk.Keys{}); err != nil {
					t.Errorf("commit: %v", err)
				}
			})
		},
	}
	client = sdk.New(programIn, programOut, handlers)
	go func() { _ = client.Loop() }()

	host := NewHost(Options{
		Shell: []string{"fake-shell"}, In: input, Out: hostOut,
		NewProcess: func([]string) (process, error) { return proc, nil },
		NewPTY: func(pty.Config) pty.PTY {
			box := newFakePTY()
			stateMu.Lock()
			ptyBox = box
			stateMu.Unlock()
			return box
		},
		Cols: 80, Rows: 24, Tick: 5 * time.Millisecond,
	})
	done := make(chan error, 1)
	go func() { done <- host.Run() }()
	defer func() {
		select {
		case <-done:
		default:
			proc.Stop()
			host.stopProgram()
		}
	}()

	waitFor(t, "layout hello", func() bool { return client.Hello() != nil })
	waitFor(t, "terminal view", func() bool {
		id, tid, box := state()
		return id != "" && tid != "" && box != nil && client.Rev() >= 1
	})
	id, terminalID, box := state()
	term, ok := host.handler.TerminalBySource(id)
	if !ok {
		t.Fatalf("terminal %q not found", id)
	}

	// Seed local scrollback so a +1 scroll can pin a frozen viewport.
	var seed strings.Builder
	for i := 0; i < 32; i++ {
		seed.WriteString("line-")
		seed.WriteString(fmt.Sprint(i))
		seed.WriteString("\r\n")
	}
	box.emit(seed.String())

	// Drive the same scroll path the real shell uses: the program emits
	// terminal.scroll with view == the pane id.
	if _, err := client.Emit("terminal.scroll", &pb.MethodParams{
		Endpoint: "local", Id: terminalID, Delta: 1, Rows: 8, View: "pane-a",
	}, nil); err != nil {
		t.Fatalf("emit scroll: %v", err)
	}
	waitFor(t, "pane-a frozen viewport", func() bool { return term.Offset("pane-a") > 0 })
	if got := term.Offset("pane-b"); got != 0 {
		t.Fatalf("pane-b offset = %d, want 0 (a sibling pane must stay live)", got)
	}
	if !term.HistoryActive("pane-a") {
		t.Fatal("pane-a must be frozen after its own scroll")
	}
	if term.HistoryActive("pane-b") {
		t.Fatal("pane-b must stay live when only pane-a scrolls")
	}

	_, _ = input.Write([]byte{0x11})
	waitFor(t, "quit confirmation", func() bool { return bytes.Contains([]byte(hostOut.String()), []byte("Quit tui2?")) })
	_, _ = input.Write([]byte{'\r'})
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("host run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("host did not stop")
	}
}

// TestHostCopyModeProgramCursorReachesFrame pins the GAP 2 host half: when the
// layout program declares a cursor on a terminal box for a frozen copy session
// (view.go term.Cursor(st.cursorRow, st.cursorCol, "block")), the composited
// host frame must carry that program cursor even though the pane is not
// focused and its PTY cursor is not involved. The legacy copy cursor was an
// always-visible hardware block cursor (render.copyHistoryCursor ->
// terminalhost.FrameSink), so entering copy mode must keep the white cursor.
func TestHostCopyModeProgramCursorReachesFrame(t *testing.T) {
	programIn := newMemPipe()
	programOut := newMemPipe()
	proc := &fakeProcess{stdin: programIn, stdout: programOut, stopped: make(chan struct{})}
	input := newMemPipe()
	hostOut := &syncBuffer{}

	var stateMu sync.Mutex
	var sourceID string
	var ptyBox *fakePTY
	var client *sdk.Client
	state := func() (string, *fakePTY) {
		stateMu.Lock()
		defer stateMu.Unlock()
		return sourceID, ptyBox
	}

	// copyCursor is the position the program declares while a copy session is
	// open; it is nil in live mode so the pane carries no program cursor.
	var copyCursor *pb.Cursor
	var copySel string
	commit := func() {
		stateMu.Lock()
		id := sourceID
		stateMu.Unlock()
		box := sdk.Terminal(id).ID("pane-a").Pos(0, 0).Width(20).Height(6).
			Input("key", "paste", "wheel").
			Props(map[string]string{
				"chrome.inset":      "0",
				"chrome.owner":      "1",
				"copy.style.cursor": "reverse",
			}).
			// The live PTY cursor only shows on the focused placement, so keep
			// the pane focused: the copy program cursor must win over it.
			Focused(true)
		if copyCursor != nil {
			box.Cursor(int(copyCursor.GetRow()), int(copyCursor.GetCol()), copyCursor.GetShape())
			// The real v3shell also sends the copy overlay props (copy.cursor /
			// copy.sel / styles). The overlay is a redundant repaint; the frame
			// cursor must remain the single hardware cursor at the copy cell.
			box.Props(map[string]string{
				"copy.cursor": "2,7",
				"copy.sel":    copySel,
			})
		}
		if err := client.Commit(sdk.Stack(box).Build(), sdk.Keys{}); err != nil {
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
				commit()
			})
		},
	}
	client = sdk.New(programIn, programOut, handlers)
	go func() { _ = client.Loop() }()

	host := NewHost(Options{
		Shell: []string{"fake-shell"}, In: input, Out: hostOut,
		NewProcess: func([]string) (process, error) { return proc, nil },
		NewPTY: func(pty.Config) pty.PTY {
			box := newFakePTY()
			stateMu.Lock()
			ptyBox = box
			stateMu.Unlock()
			return box
		},
		Cols: 40, Rows: 12,
	})
	done := make(chan error, 1)
	go func() { done <- host.Run() }()
	defer func() {
		select {
		case <-done:
		default:
			proc.Stop()
			host.stopProgram()
		}
	}()

	waitFor(t, "layout hello", func() bool { return client.Hello() != nil })
	waitFor(t, "terminal view", func() bool {
		id, box := state()
		return id != "" && box != nil && client.Rev() >= 1
	})

	// Live mode before copy: the focused pane shows the PTY's own hardware
	// cursor through the Placement path (the compositor's fallback), so the
	// "small white cursor" already exists and copy mode only swaps its source.
	id, box := state()
	term, ok := host.handler.TerminalBySource(id)
	if !ok {
		t.Fatalf("terminal %q not found", id)
	}
	var seed strings.Builder
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&seed, "line-%d\r\n", i)
	}
	box.emit(seed.String())
	waitFor(t, "live PTY cursor visible", func() bool {
		_, _, visible := host.currentSession().ComposeFrame(host.placements(), nil).Cursor()
		return visible
	})

	// Enter copy mode at the live bottom (offset 0): the program declares the
	// block cursor at (2,7). It must be visible in the composited host frame
	// regardless of the frozen offset.
	copyCursor = &pb.Cursor{Row: 2, Col: 7, Shape: "block"}
	copySel = ""
	commit()
	waitFor(t, "copy commit", func() bool { return client.Rev() >= 2 })
	waitFor(t, "copy cursor in frame", func() bool {
		frame := host.currentSession().ComposeFrame(host.placements(), nil)
		x, y, visible := frame.Cursor()
		return visible && x == 7 && y == 2 && frame.CursorShape() == "block"
	})
	frame := host.currentSession().ComposeFrame(host.placements(), nil)
	x, y, visible := frame.Cursor()
	if !visible || x != 7 || y != 2 {
		t.Fatalf("copy frame cursor = (%d,%d) visible=%v, want (7,2)", x, y, visible)
	}
	if frame.CursorShape() != "block" {
		t.Fatalf("copy frame cursor shape = %q, want block", frame.CursorShape())
	}

	// Scroll the frozen window: the copy cursor must stay visible through the
	// per-pane history path too.
	if _, err := client.Emit("terminal.scroll", &pb.MethodParams{
		Endpoint: "local", Id: strings.TrimPrefix(id, "terminal:local:"), Delta: 1, Rows: 4, View: "pane-a",
	}, nil); err != nil {
		t.Fatalf("emit scroll: %v", err)
	}
	waitFor(t, "pane-a frozen", func() bool { return term.Offset("pane-a") > 0 })
	waitFor(t, "copy cursor visible while scrolled", func() bool {
		x, y, visible := host.currentSession().ComposeFrame(host.placements(), nil).Cursor()
		return visible && x == 7 && y == 2
	})

	// Leave copy mode: no program cursor, and a live (unfocused) pane must not
	// leave a stale visible cursor behind.
	copyCursor = nil
	commit()
	waitFor(t, "cursor hidden after copy", func() bool {
		_, _, visible := host.currentSession().ComposeFrame(host.placements(), nil).Cursor()
		return !visible
	})

	_, _ = input.Write([]byte{0x11})
	waitFor(t, "quit confirmation", func() bool { return bytes.Contains([]byte(hostOut.String()), []byte("Quit tui2?")) })
	_, _ = input.Write([]byte{'\r'})
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("host run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("host did not stop")
	}
}

// TestHostCodexLikePersistentTUIPassthroughTrace runs a terminal child that
// enables DEC mouse tracking and records the raw wheel bytes written to its
// PTY. It deliberately has a persistent history capability as a remote
// terminal would, but the child still owns the wheel gesture while live.
// This is the boundary that must remain transparent for Codex/OpenCode-like
// TUIs; the layout program must not receive a synthetic history event here.
func TestHostCodexLikePersistentTUIPassthroughTrace(t *testing.T) {
	programIn := newMemPipe()
	programOut := newMemPipe()
	proc := &fakeProcess{stdin: programIn, stdout: programOut, stopped: make(chan struct{})}
	input := newMemPipe()
	hostOut := &syncBuffer{}

	var stateMu sync.Mutex
	var sourceID string
	var ptyBox *traceHistoryPTY
	wheelEvents := make(chan int, 8)
	var client *sdk.Client
	state := func() (string, *traceHistoryPTY) {
		stateMu.Lock()
		defer stateMu.Unlock()
		return sourceID, ptyBox
	}

	handlers := sdk.Handlers{
		Hello: func(h *pb.Hello) {
			_, _ = client.Emit("terminal.create", &pb.MethodParams{Endpoint: "local"}, func(resp *pb.Response) {
				if !resp.GetOk() {
					t.Errorf("create failed: %s", resp.GetError())
					return
				}
				data := resp.GetData()
				stateMu.Lock()
				sourceID = "terminal:" + data.GetEndpoint() + ":" + data.GetId()
				stateMu.Unlock()
				tree := sdk.Row(sdk.Terminal(sourceID).ID("term").Width(78).Height(18).
					Input("key", "wheel").Focused(true))
				if err := client.Commit(tree.Build(), sdk.Keys{}); err != nil {
					t.Errorf("commit: %v", err)
				}
			})
		},
		Wheel: func(w *pb.WheelEvent) { wheelEvents <- int(w.GetDelta()) },
	}
	client = sdk.New(programIn, programOut, handlers)
	go func() { _ = client.Loop() }()

	host := NewHost(Options{
		Shell: []string{"fake-shell"}, In: input, Out: hostOut,
		NewProcess: func([]string) (process, error) { return proc, nil },
		NewPTY: func(pty.Config) pty.PTY {
			box := &traceHistoryPTY{fakePTY: newFakePTY(), backend: newTraceHistoryBackend()}
			stateMu.Lock()
			ptyBox = box
			stateMu.Unlock()
			return box
		},
		Cols: 80, Rows: 20, Tick: 5 * time.Millisecond,
	})
	done := make(chan error, 1)
	go func() { done <- host.Run() }()
	defer func() {
		select {
		case <-done:
		default:
			proc.Stop()
			host.stopProgram()
		}
	}()

	waitFor(t, "layout hello", func() bool { return client.Hello() != nil })
	waitFor(t, "persistent terminal view", func() bool {
		id, box := state()
		return id != "" && box != nil && client.Rev() >= 1
	})
	_, box := state()
	box.emit("\x1b[?1000h\x1b[?1006h")
	waitFor(t, "mouse tracking enabled", func() bool {
		term, ok := host.handler.TerminalBySource(sourceID)
		return ok && term.Modes().MouseTracking()
	})

	var trace []string
	for _, tc := range []struct {
		name  string
		seq   string
		delta int
	}{
		{name: "up", seq: "\x1b[<64;4;4M", delta: 1},
		{name: "down", seq: "\x1b[<65;4;4M", delta: -1},
	} {
		before := box.written()
		if _, err := input.Write([]byte(tc.seq)); err != nil {
			t.Fatalf("write %s wheel: %v", tc.name, err)
		}
		select {
		case got := <-wheelEvents:
			trace = append(trace, fmt.Sprintf("%s -> layout program delta=%d", tc.name, got))
			t.Fatalf("Codex-like TUI wheel was diverted to layout: trace=%s", strings.Join(trace, " | "))
		case <-time.After(100 * time.Millisecond):
		}
		waitFor(t, tc.name+" wheel reaches child PTY", func() bool { return len(box.written()) > len(before) })
		trace = append(trace, fmt.Sprintf("%s -> child PTY bytes=%q", tc.name, box.written()[len(before):]))
	}
	t.Logf("Codex-like passthrough trace: %s", strings.Join(trace, " | "))

	_, _ = input.Write([]byte{0x11})
	waitFor(t, "quit confirmation", func() bool { return bytes.Contains([]byte(hostOut.String()), []byte("Quit tui2?")) })
	_, _ = input.Write([]byte{'\r'})
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("host run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("host did not stop")
	}
}

func TestHostTerminalPointerUsesContentCoordinatesAndKeepsDrag(t *testing.T) {
	session := runtime.NewSession(runtime.Options{ViewID: "v", Cols: 80, Rows: 24}, bytes.NewReader(nil), io.Discard)
	if err := session.SetSources([]*pb.Source{{Id: "terminal:local:main", Kind: "terminal", Attached: true}}); err != nil {
		t.Fatal(err)
	}
	if err := session.HandleView(&pb.View{
		Epoch: 1,
		Rev:   1,
		Root: &pb.Box{Id: "root", Children: []*pb.Box{{
			Id: "term", Focused: true, Input: []string{"mouse", "wheel"},
			Content: &pb.Content{Self: "terminal:local:main"},
		}}},
	}); err != nil {
		t.Fatal(err)
	}
	host := &Host{}
	press := keys.Event{Kind: keys.KindMouse, Action: keys.ActionPress, Button: keys.ButtonLeft, X: 2, Y: 2}
	host.preparePointer(session, &press)
	if !press.HitFocused || press.PTYX != 1 || press.PTYY != 1 {
		t.Fatalf("press = %+v, want first content cell", press)
	}
	release := keys.Event{Kind: keys.KindMouse, Action: keys.ActionRelease, Button: keys.ButtonLeft, X: 100, Y: 40}
	host.preparePointer(session, &release)
	if !release.HitFocused || release.PTYX != 78 || release.PTYY != 22 {
		t.Fatalf("release = %+v, want clamped terminal cell", release)
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

// visibleObserverCount mirrors the program-side pane badge formula: the daemon
// total with this client's single shared attachment replaced by the local pane
// count. It is the assertion helper for the host tests below (the program
// itself computes it in clients/tui/examples/v3shell).
func visibleObserverCount(daemon, localPanes int) int {
	if localPanes <= 0 {
		return maxInt(0, daemon)
	}
	if daemon <= 0 {
		return localPanes
	}
	return maxInt(0, daemon+localPanes-1)
}

// TestTestBinaryDetection pins the guard that keeps the observer-count polling
// ticker out of unit tests: inside `go test` the testing package has registered
// its flags, so testBinary must be true (otherwise tests would start a 3s list
// loop against a dead endpoint).
func TestTestBinaryDetection(t *testing.T) {
	if !testBinary() {
		t.Fatal("testBinary() must detect the go test binary")
	}
}

// TestSnapshotSourcesCarriesDaemonAttachmentCount pins the host side of the
// observer-count badge: the daemon attachment_count is republished on the
// source, a change republishes the snapshot, and the program-side formula turns
// (daemon=2, 2 local panes) into 3 and (daemon=0, 1 local pane) into 1.
func TestSnapshotSourcesCarriesDaemonAttachmentCount(t *testing.T) {
	// daemonCount stands in for the manager's cached inventory; each call builds
	// a fresh source, exactly like Manager.Sources does after a re-list.
	daemonCount := int32(2)
	host := &Host{daemonSourceFn: func() []*pb.Source {
		return []*pb.Source{{
			Id: "terminal:hs:term-1", Kind: "terminal", Title: "term-1",
			Endpoint: "hs", TerminalId: "term-1", AttachmentCount: daemonCount,
		}}
	}}

	items, changed := host.snapshotSources()
	if !changed || len(items) != 1 {
		t.Fatalf("first snapshot changed=%v items=%d", changed, len(items))
	}
	if got := items[0].GetAttachmentCount(); got != 2 {
		t.Fatalf("published attachment_count = %d, want 2", got)
	}
	// The program sees daemon 2 + two local panes - this client's shared seat.
	if got := visibleObserverCount(int(items[0].GetAttachmentCount()), 2); got != 3 {
		t.Fatalf("visible observers = %d, want 3 (2 + 2 - 1)", got)
	}
	// An unchanged daemon count must not republish.
	if _, changed := host.snapshotSources(); changed {
		t.Fatal("an unchanged attachment_count must not republish")
	}
	// A daemon count change republishes.
	daemonCount = 3
	items, changed = host.snapshotSources()
	if !changed {
		t.Fatal("a changed attachment_count must republish")
	}
	if got := items[0].GetAttachmentCount(); got != 3 {
		t.Fatalf("republished attachment_count = %d, want 3", got)
	}

	// A single local pane with no daemon count still shows one observer.
	if got := visibleObserverCount(0, 1); got != 1 {
		t.Fatalf("visible observers = %d, want 1 (daemon 0, one local pane)", got)
	}
}

func TestAttachedSourceTitlePreservesDaemonName(t *testing.T) {
	const sourceID = "terminal:hs:autopush-id"
	if got := attachedSourceTitle(sourceID,
		map[string]string{sourceID: "autopush"},
		map[string]string{sourceID: "MIX-TERM"},
		"terminal-id",
	); got != "autopush" {
		t.Fatalf("daemon title = %q, want autopush", got)
	}
	if got := attachedSourceTitle(sourceID, nil,
		map[string]string{sourceID: "autopush"},
		"terminal-id",
	); got != "autopush" {
		t.Fatalf("previous title = %q, want autopush", got)
	}
	if got := attachedSourceTitle(sourceID, nil, nil, "terminal-id"); got != "terminal-id" {
		t.Fatalf("fallback title = %q, want terminal-id", got)
	}
}

func TestTrackedSourceTitlePrefersExplicitRename(t *testing.T) {
	term := &runtime.Terminal{}
	tracked := &trackedTerminal{term: term, title: "renamed"}
	if got := trackedSourceTitle("terminal:local:id", tracked, nil, map[string]string{"terminal:local:id": "old"}); got != "renamed" {
		t.Fatalf("tracked title = %q, want renamed", got)
	}
}

func TestHostClipboardPasteAndDetachLifecycle(t *testing.T) {
	var proc *fakePTY
	host := NewHost(Options{
		Shell:                []string{"fake-shell"},
		In:                   newMemPipe(),
		Out:                  &syncBuffer{},
		ClipboardHistoryPath: filepath.Join(t.TempDir(), "clipboard.json"),
		ClipboardRead:        func() (string, error) { return "one\ntwo", nil },
		NewProcess: func([]string) (process, error) {
			return &fakeProcess{stdin: newMemPipe(), stdout: newMemPipe(), stopped: make(chan struct{})}, nil
		},
		NewPTY: func(pty.Config) pty.PTY {
			proc = newFakePTY()
			return proc
		},
	})
	defer host.handler.Close()
	params := &pb.MethodParams{Endpoint: "local", Id: "clip-test"}
	if outcome, pending := host.gate.Handle(runtime.Request{Method: runtime.Method{Name: "terminal.attach"}, Params: params}); !outcome.OK || pending {
		t.Fatalf("attach = %+v pending=%v", outcome, pending)
	}
	outcome, pending := host.gate.Handle(runtime.Request{Method: runtime.Method{Name: "clipboard.paste"}, Params: params})
	if !outcome.OK || pending {
		t.Fatalf("paste = %+v pending=%v", outcome, pending)
	}
	proc.mu.Lock()
	got := proc.input.String()
	proc.mu.Unlock()
	if got != "one\rtwo" {
		t.Fatalf("pasted bytes = %q, want CR-normalized text", got)
	}
	outcome, pending = host.gate.Handle(runtime.Request{Method: runtime.Method{Name: "terminal.detach"}, Params: params})
	if !outcome.OK || pending {
		t.Fatalf("detach = %+v pending=%v", outcome, pending)
	}
	if _, ok := host.handler.TerminalBySource(runtime.SourceID("local", "clip-test")); ok {
		t.Fatal("detached terminal still accepts input")
	}
	term, ok := host.handler.TerminalAt("local", "clip-test")
	if !ok || term.Exited() {
		t.Fatal("detach must preserve the live terminal for reconnect")
	}
	outcome, pending = host.gate.Handle(runtime.Request{Method: runtime.Method{Name: "terminal.reconnect"}, Params: params})
	if !outcome.OK || pending {
		t.Fatalf("reconnect = %+v pending=%v", outcome, pending)
	}
	if _, ok := host.handler.TerminalBySource(runtime.SourceID("local", "clip-test")); !ok {
		t.Fatal("reconnected terminal was not routable")
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
		return bytes.Contains([]byte(out.String()), []byte("Quit tui2?"))
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

// countingPTY records how many Resize calls reached the single source PTY so a
// test can prove follower panes never reflow it.
type countingPTY struct {
	*fakePTY
	mu      sync.Mutex
	resizes int
}

func newCountingPTY() *countingPTY { return &countingPTY{fakePTY: newFakePTY()} }

func (p *countingPTY) Resize(cols, rows int) error {
	p.mu.Lock()
	p.resizes++
	p.mu.Unlock()
	return p.fakePTY.Resize(cols, rows)
}

func (p *countingPTY) resizeCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.resizes
}

// TestHostTerminalResizeFollowsSingleOwnerPerSource pins the legacy
// owner/follower resize semantics for one terminal source shown by two panes of
// different widths: only the per-source owner pane drives the single PTY size,
// so the terminal settles at the focused pane's content rect instead of
// oscillating between both panes every frame. Moving focus hands ownership over
// and the PTY follows the new owner.
func TestHostTerminalResizeFollowsSingleOwnerPerSource(t *testing.T) {
	programIn := newMemPipe()
	programOut := newMemPipe()
	proc := &fakeProcess{stdin: programIn, stdout: programOut, stopped: make(chan struct{})}

	var stateMu sync.Mutex
	var ptyBox *countingPTY
	var sourceID string
	var client *sdk.Client

	// Both panes reference the same source (so one shared PTY); only the
	// focused pane is the owner. pane-a is wider than pane-b so the two sizes
	// are unambiguous (content = box minus the default 1-cell border).
	commit := func(focusA bool) {
		stateMu.Lock()
		src := sourceID
		stateMu.Unlock()
		a := sdk.Terminal(src).ID("pane-a").Width(20).Height(8).
			Input("key", "paste").Focused(focusA)
		b := sdk.Terminal(src).ID("pane-b").Width(12).Height(8).
			Input("key", "paste").Focused(!focusA)
		if err := client.Commit(sdk.Row(a, b).Build(), sdk.Keys{Claim: []string{"ctrl-p"}}); err != nil {
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
				commit(true)
			})
		},
	}
	client = sdk.New(programIn, programOut, handlers)
	go func() { _ = client.Loop() }()

	input := newMemPipe()
	host := NewHost(Options{
		Shell: []string{"fake-shell"},
		In:    input,
		Out:   &syncBuffer{},
		NewProcess: func([]string) (process, error) {
			return proc, nil
		},
		NewPTY: func(pty.Config) pty.PTY {
			box := newCountingPTY()
			stateMu.Lock()
			ptyBox = box
			stateMu.Unlock()
			return box
		},
		Cols: 40,
		Rows: 10,
		Tick: 5 * time.Millisecond,
	})

	done := make(chan error, 1)
	go func() { done <- host.Run() }()
	defer func() {
		host.quit()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("host did not exit")
		}
	}()

	box := func() *countingPTY {
		stateMu.Lock()
		defer stateMu.Unlock()
		return ptyBox
	}

	// The focused owner pane-a drives the PTY: content width 20-2, height 8-2.
	waitFor(t, "PTY follows the focused owner pane", func() bool {
		p := box()
		return p != nil && p.size() == [2]int{18, 6}
	})

	// Follower pane-b must never reflow the PTY, so after settling every sample
	// stays at the owner size and exactly one resize was issued.
	samples := 4
	for i := 0; i < samples; i++ {
		p := box()
		if got := p.size(); got != [2]int{18, 6} {
			t.Fatalf("sample %d after settle = %v, want the focused owner size [18 6]", i, got)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := box().resizeCount(); got != 1 {
		t.Fatalf("owner-only resize calls = %d, want 1 (followers must not reflow the PTY)", got)
	}

	// Handing focus to pane-b transfers ownership; the terminal resizes to the
	// new owner's rect (content 12-2 x 8-2).
	commit(false)
	waitFor(t, "PTY follows the newly focused owner", func() bool {
		p := box()
		return p != nil && p.size() == [2]int{10, 6}
	})
	for i := 0; i < samples; i++ {
		p := box()
		if got := p.size(); got != [2]int{10, 6} {
			t.Fatalf("follower sample %d after focus switch = %v, want [10 6]", i, got)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestHostTerminalResizeOwnerFollowsDeclaredOwnerProp pins the manual resize
// ownership contract: the program declares the owner with chrome.owner=1 on one
// box, and the host resizes the single PTY to THAT box's content rect even
// though focus sits on a different box. Focus must never outrank the declared
// owner (the legacy panel.take_owner model), so the size is stable and does not
// oscillate toward the focused follower.
func TestHostTerminalResizeOwnerFollowsDeclaredOwnerProp(t *testing.T) {
	programIn := newMemPipe()
	programOut := newMemPipe()
	proc := &fakeProcess{stdin: programIn, stdout: programOut, stopped: make(chan struct{})}

	var stateMu sync.Mutex
	var ptyBox *countingPTY
	var sourceID string
	var client *sdk.Client

	// pane-a declares chrome.owner=1 (content 18x6) but pane-b is focused
	// (content 10x6). The declared owner must win over focus.
	commit := func() {
		stateMu.Lock()
		src := sourceID
		stateMu.Unlock()
		a := sdk.Terminal(src).ID("pane-a").Width(20).Height(8).
			Props(map[string]string{"chrome.owner": "1"}).
			Input("key", "paste").Focused(false)
		b := sdk.Terminal(src).ID("pane-b").Width(12).Height(8).
			Input("key", "paste").Focused(true)
		if err := client.Commit(sdk.Row(a, b).Build(), sdk.Keys{Claim: []string{"ctrl-p"}}); err != nil {
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
				commit()
			})
		},
	}
	client = sdk.New(programIn, programOut, handlers)
	go func() { _ = client.Loop() }()

	input := newMemPipe()
	host := NewHost(Options{
		Shell: []string{"fake-shell"},
		In:    input,
		Out:   &syncBuffer{},
		NewProcess: func([]string) (process, error) {
			return proc, nil
		},
		NewPTY: func(pty.Config) pty.PTY {
			box := newCountingPTY()
			stateMu.Lock()
			ptyBox = box
			stateMu.Unlock()
			return box
		},
		Cols: 40,
		Rows: 10,
		Tick: 5 * time.Millisecond,
	})

	done := make(chan error, 1)
	go func() { done <- host.Run() }()
	defer func() {
		host.quit()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("host did not exit")
		}
	}()

	box := func() *countingPTY {
		stateMu.Lock()
		defer stateMu.Unlock()
		return ptyBox
	}

	// The DECLARED owner pane-a (not the focused pane-b) drives the PTY.
	waitFor(t, "PTY follows the declared owner, not focus", func() bool {
		p := box()
		return p != nil && p.size() == [2]int{18, 6}
	})

	// The focused follower must never reflow the PTY: after settling every
	// sample stays at the declared owner size and exactly one resize was issued.
	samples := 4
	for i := 0; i < samples; i++ {
		p := box()
		if got := p.size(); got != [2]int{18, 6} {
			t.Fatalf("sample %d after settle = %v, want the declared owner size [18 6]", i, got)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := box().resizeCount(); got != 1 {
		t.Fatalf("declared-owner-only resize calls = %d, want 1 (focus must not resize)", got)
	}
}

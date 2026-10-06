package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/history"
	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/clients/tui/render/ansi"
	"github.com/anytty/anytty/clients/tui/runtime"
	"github.com/anytty/anytty/clients/tui/runtime/keys"
	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/proto/access/apipb"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
	"github.com/charmbracelet/x/term"
)

const (
	realPTYChildEnv        = "ANYTTY_TUI2_REAL_PTY_CHILD"
	realPTYLogEnv          = "ANYTTY_TUI2_REAL_PTY_LOG"
	realPTYScrollChildEnv  = "ANYTTY_TUI2_REAL_PTY_SCROLL_CHILD"
	realPTYChunkedChildEnv = "ANYTTY_TUI2_REAL_PTY_CHUNKED_CHILD"
)

// TestTui2RealPTYChild is run in a real PTY by TestHostRealPTYWheelPassthrough.
// Keeping the child in the test binary makes the regression independent of a
// prebuilt helper and exercises the same raw terminal boundary as Codex-like
// full-screen TUIs.
func TestTui2RealPTYChild(t *testing.T) {
	if os.Getenv(realPTYChildEnv) != "1" {
		return
	}
	logPath := os.Getenv(realPTYLogEnv)
	trace, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("open child trace: %v", err)
	}
	defer trace.Close()
	state, err := term.MakeRaw(os.Stdin.Fd())
	if err != nil {
		t.Fatalf("raw child PTY: %v", err)
	}
	defer term.Restore(os.Stdin.Fd(), state)
	if os.Getenv(realPTYScrollChildEnv) == "1" {
		if os.Getenv(realPTYChunkedChildEnv) == "1" {
			runRealPTYChunkedScrollChild(trace)
			return
		}
		runRealPTYScrollChild(trace)
		return
	}
	_, _ = io.WriteString(os.Stdout, "\x1b[?1049h\x1b[?25l\x1b[?1000h\x1b[?1006h\x1b[2J\x1b[Hreal tui child")
	defer io.WriteString(os.Stdout, "\x1b[?1006l\x1b[?1000l\x1b[?25h\x1b[?1049l")
	buf := make([]byte, 4096)
	for {
		n, readErr := os.Stdin.Read(buf)
		if n > 0 {
			if _, err := trace.Write(buf[:n]); err != nil {
				return
			}
		}
		if readErr != nil {
			return
		}
	}
}

// runRealPTYChunkedScrollChild deliberately splits a synchronized redraw
// across multiple PTY writes. A full-screen app can leave the parser showing
// the cleared screen for a short time between these writes; the host must not
// publish that intermediate screen merely because another wheel arrives.
func runRealPTYChunkedScrollChild(trace io.Writer) {
	const maxOffset = 3
	offset := 0
	paint := func() {
		_, _ = io.WriteString(os.Stdout, "\x1b[?2026h\x1b[?1049h\x1b[?25l\x1b[?1000h\x1b[?1006h\x1b[2J\x1b[H")
		_, _ = fmt.Fprintf(trace, "stage=clear offset=%d\n", offset)
		time.Sleep(35 * time.Millisecond)
		var b strings.Builder
		fmt.Fprintf(&b, "codex-probe offset=%d\r\n", offset)
		for i := 0; i < 10; i++ {
			fmt.Fprintf(&b, "history-line-%02d\r\n", i+offset)
		}
		_, _ = io.WriteString(os.Stdout, b.String())
		_, _ = fmt.Fprintf(trace, "stage=content offset=%d\n", offset)
		time.Sleep(35 * time.Millisecond)
		_, _ = io.WriteString(os.Stdout, "\x1b[?2026l")
		_, _ = fmt.Fprintf(trace, "stage=end offset=%d\n", offset)
	}
	paint()
	parser := keys.NewParser()
	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			_, _ = fmt.Fprintf(trace, "raw=%q\n", buf[:n])
			for _, ev := range parser.Feed(buf[:n]) {
				if ev.Kind != keys.KindWheel {
					continue
				}
				before := offset
				offset += ev.Delta
				if offset < 0 {
					offset = 0
				}
				if offset > maxOffset {
					offset = maxOffset
				}
				_, _ = fmt.Fprintf(trace, "wheel delta=%d before=%d after=%d\n", ev.Delta, before, offset)
				paint()
			}
		}
		if err != nil {
			return
		}
	}
}

// outputTraceBuffer keeps the exact host writes as well as their concatenated
// output. Replaying each write through the ANSI parser lets the regression
// inspect the screen after every host flush, rather than only its final state.
type outputTraceBuffer struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	writes [][]byte
}

func (b *outputTraceBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	b.buf.Write(p)
	b.writes = append(b.writes, append([]byte(nil), p...))
	b.mu.Unlock()
	return len(p), nil
}

func (b *outputTraceBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (b *outputTraceBuffer) WritesFrom(index int) [][]byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	if index < 0 {
		index = 0
	}
	if index > len(b.writes) {
		index = len(b.writes)
	}
	out := make([][]byte, len(b.writes)-index)
	for i := range out {
		out[i] = append([]byte(nil), b.writes[index+i]...)
	}
	return out
}

func replayHostFrames(writes [][]byte, cols, rows int) []ansi.Screen {
	parser := ansi.New(cols, rows)
	frames := make([]ansi.Screen, 0, len(writes))
	for _, write := range writes {
		parser.Write(write)
		frames = append(frames, parser.Screen())
	}
	return frames
}

func probeOffset(screen ansi.Screen) (string, bool) {
	for y := range screen.Lines {
		line := screen.Text(y)
		if at := strings.Index(line, "codex-probe offset="); at >= 0 {
			end := strings.IndexByte(line[at:], '\r')
			if end < 0 {
				end = len(line) - at
			}
			return line[at : at+end], true
		}
	}
	return "", false
}

// runRealPTYScrollChild is a tiny Codex-like full-screen child. It owns a
// bounded scroll position and repaints the complete viewport after every wheel
// report. The test host must pass the reports through unchanged; if it turns a
// live wheel into host history traffic, the trace exposes the divergence.
func runRealPTYScrollChild(trace io.Writer) {
	const maxOffset = 3
	offset := 0
	paint := func() {
		var b strings.Builder
		// Codex/OpenCode commonly wrap a full redraw in DEC synchronized
		// output. Keep this in the probe so the host's parser notification
		// boundary is exercised together with mouse forwarding.
		b.WriteString("\x1b[?2026h\x1b[?1049h\x1b[?25l\x1b[?1000h\x1b[?1006h\x1b[2J\x1b[H")
		fmt.Fprintf(&b, "codex-probe offset=%d\r\n", offset)
		for i := 0; i < 10; i++ {
			fmt.Fprintf(&b, "history-line-%02d\r\n", i+offset)
		}
		b.WriteString("\x1b[?2026l")
		_, _ = io.WriteString(os.Stdout, b.String())
		_, _ = fmt.Fprintf(trace, "paint offset=%d\n", offset)
	}
	paint()
	parser := keys.NewParser()
	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			_, _ = fmt.Fprintf(trace, "raw=%q\n", buf[:n])
			for _, ev := range parser.Feed(buf[:n]) {
				if ev.Kind != keys.KindWheel {
					continue
				}
				before := offset
				offset += ev.Delta
				if offset < 0 {
					offset = 0
				}
				if offset > maxOffset {
					offset = maxOffset
				}
				_, _ = fmt.Fprintf(trace, "wheel delta=%d before=%d after=%d\n", ev.Delta, before, offset)
				paint()
			}
		}
		if err != nil {
			return
		}
	}
}

type realHistoryPTY struct {
	pty.PTY
	backend history.Backend
}

func (p *realHistoryPTY) HistoryBackend(context.Context) (history.Backend, *apipb.TerminalRef, error) {
	return p.backend, &apipb.TerminalRef{TerminalId: "real"}, nil
}

func TestHostRealPTYWheelPassthrough(t *testing.T) {
	if os.Getenv(realPTYChildEnv) == "1" {
		t.Skip("child process mode")
	}
	programIn := newMemPipe()
	programOut := newMemPipe()
	proc := &fakeProcess{stdin: programIn, stdout: programOut, stopped: make(chan struct{})}
	input := newMemPipe()
	hostOut := &syncBuffer{}
	tracePath := filepath.Join(t.TempDir(), "child-input.log")

	var stateMu sync.Mutex
	var sourceID string
	var client *sdk.Client
	client = sdk.New(programIn, programOut, sdk.Handlers{
		Hello: func(*pb.Hello) {
			_, _ = client.Emit("terminal.create", &pb.MethodParams{
				Endpoint: "local", Argv: []string{"ignored"},
			}, func(resp *pb.Response) {
				if !resp.GetOk() {
					t.Errorf("create failed: %s", resp.GetError())
					return
				}
				d := resp.GetData()
				stateMu.Lock()
				sourceID = runtime.SourceID(d.GetEndpoint(), d.GetId())
				stateMu.Unlock()
				root := sdk.Row(sdk.Terminal(sourceID).ID("term").Flex(1).
					Input("key", "paste", "wheel").Focused(true))
				if err := client.Commit(root.Build(), sdk.Keys{}); err != nil {
					t.Errorf("commit: %v", err)
				}
			})
		},
	})
	go func() { _ = client.Loop() }()

	host := NewHost(Options{
		Shell: []string{"fake-shell"}, In: input, Out: hostOut,
		NewProcess: func([]string) (process, error) { return proc, nil },
		NewPTY: func(cfg pty.Config) pty.PTY {
			env := append(os.Environ(), realPTYChildEnv+"=1", realPTYLogEnv+"="+tracePath)
			return &realHistoryPTY{
				PTY: pty.New(pty.Config{
					Argv: []string{os.Args[0], "-test.run=TestTui2RealPTYChild"},
					Env:  env, Cols: cfg.Cols, Rows: cfg.Rows,
				}),
				backend: newTraceHistoryBackend(),
			}
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
	var currentSourceID string
	waitFor(t, "real PTY view", func() bool {
		stateMu.Lock()
		currentSourceID = sourceID
		stateMu.Unlock()
		return currentSourceID != "" && client.Rev() >= 1
	})
	termSource, ok := host.handler.TerminalBySource(currentSourceID)
	if !ok {
		t.Fatalf("terminal %q not found", currentSourceID)
	}
	waitFor(t, "real child mouse tracking", func() bool { return termSource.Modes().MouseTracking() })

	for i := 0; i < 8; i++ {
		if _, err := input.Write([]byte("\x1b[<64;4;4M\x1b[<65;4;4M")); err != nil {
			t.Fatalf("write wheel burst: %v", err)
		}
	}
	waitFor(t, "real child receives wheel burst", func() bool {
		data, err := os.ReadFile(tracePath)
		return err == nil && bytes.Count(data, []byte("\x1b[<64;3;3M")) >= 8 && bytes.Count(data, []byte("\x1b[<65;3;3M")) >= 8
	})
	data, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	if got := bytes.Count(data, []byte("\x1b[<64;3;3M")); got != 8 {
		t.Fatalf("real child up wheel count = %d, want 8; trace=%q", got, data)
	}
	if got := bytes.Count(data, []byte("\x1b[<65;3;3M")); got != 8 {
		t.Fatalf("real child down wheel count = %d, want 8; trace=%q", got, data)
	}

	_, _ = input.Write([]byte{0x11})
	waitFor(t, "quit confirmation", func() bool { return bytes.Contains([]byte(hostOut.String()), []byte("Quit tui2?")) })
	_, _ = input.Write([]byte{'\r'})
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("host run: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("host did not stop")
	}
}

// TestHostRealPTYChildOwnScrollBoundary drives a real full-screen child that
// implements its own finite scrollback. It sends more upward wheel reports
// after the child reaches its oldest row. Every report must arrive in the same
// order and the child's bounded offset must stay at maxOffset; an alternating
// offset here would prove that the host is injecting a reverse wheel or
// taking ownership of the live child unexpectedly.
func TestHostRealPTYChildOwnScrollBoundary(t *testing.T) {
	if os.Getenv(realPTYChildEnv) == "1" {
		t.Skip("child process mode")
	}
	programIn := newMemPipe()
	programOut := newMemPipe()
	proc := &fakeProcess{stdin: programIn, stdout: programOut, stopped: make(chan struct{})}
	input := newMemPipe()
	hostOut := &syncBuffer{}
	tracePath := filepath.Join(t.TempDir(), "child-scroll.log")

	var stateMu sync.Mutex
	var sourceID string
	var client *sdk.Client
	client = sdk.New(programIn, programOut, sdk.Handlers{
		Hello: func(*pb.Hello) {
			_, _ = client.Emit("terminal.create", &pb.MethodParams{Endpoint: "local"}, func(resp *pb.Response) {
				if !resp.GetOk() {
					t.Errorf("create failed: %s", resp.GetError())
					return
				}
				d := resp.GetData()
				stateMu.Lock()
				sourceID = runtime.SourceID(d.GetEndpoint(), d.GetId())
				stateMu.Unlock()
				root := sdk.Row(sdk.Terminal(sourceID).ID("term").Width(78).Height(18).
					Input("key", "wheel").Focused(true))
				if err := client.Commit(root.Build(), sdk.Keys{}); err != nil {
					t.Errorf("commit: %v", err)
				}
			})
		},
	})
	go func() { _ = client.Loop() }()

	host := NewHost(Options{
		Shell: []string{"fake-shell"}, In: input, Out: hostOut,
		NewProcess: func([]string) (process, error) { return proc, nil },
		NewPTY: func(cfg pty.Config) pty.PTY {
			env := append(os.Environ(), realPTYChildEnv+"=1", realPTYScrollChildEnv+"=1", realPTYLogEnv+"="+tracePath)
			return &realHistoryPTY{
				PTY:     pty.New(pty.Config{Argv: []string{os.Args[0], "-test.run=TestTui2RealPTYChild"}, Env: env, Cols: cfg.Cols, Rows: cfg.Rows}),
				backend: newTraceHistoryBackend(),
			}
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
	waitFor(t, "real scroll child view", func() bool {
		stateMu.Lock()
		defer stateMu.Unlock()
		return sourceID != "" && client.Rev() >= 1
	})
	termSource, ok := host.handler.TerminalBySource(sourceID)
	if !ok {
		t.Fatalf("terminal %q not found", sourceID)
	}
	waitFor(t, "real scroll child mouse tracking", func() bool { return termSource.Modes().MouseTracking() })

	// Fill the child's finite scroll range, then keep sending the same
	// direction at the oldest boundary. Reverse at the boundary and exercise
	// the live bottom as well; the host must preserve all twenty events.
	for i := 0; i < 10; i++ {
		if _, err := input.Write([]byte("\x1b[<64;4;4M")); err != nil {
			t.Fatalf("write wheel %d: %v", i, err)
		}
	}
	for i := 0; i < 10; i++ {
		if _, err := input.Write([]byte("\x1b[<65;4;4M")); err != nil {
			t.Fatalf("write reverse wheel %d: %v", i, err)
		}
	}
	waitFor(t, "child receives all own-scroll wheels", func() bool {
		data, err := os.ReadFile(tracePath)
		return err == nil && bytes.Count(data, []byte("wheel delta=1")) >= 10 && bytes.Count(data, []byte("wheel delta=-1")) >= 10
	})
	data, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	var offsets []int
	for _, line := range lines {
		if !strings.HasPrefix(line, "wheel delta=") {
			continue
		}
		var delta, before, after int
		if _, err := fmt.Sscanf(line, "wheel delta=%d before=%d after=%d", &delta, &before, &after); err != nil {
			t.Fatalf("malformed child trace %q: %v", line, err)
		}
		if delta == 1 {
			offsets = append(offsets, after)
			continue
		}
		if delta != -1 {
			t.Fatalf("child received malformed wheel: %q", line)
		}
		// The reverse half is checked separately below so a direction change
		// cannot be hidden by sorting or coalescing the trace.
		offsets = append(offsets, -after-1)
	}
	if len(offsets) != 20 {
		t.Fatalf("child wheel trace count=%d, want 20; trace=%q", len(offsets), data)
	}
	for i := 0; i < 10; i++ {
		want := i + 1
		if want > 3 {
			want = 3
		}
		if offsets[i] != want {
			t.Fatalf("child upward offset[%d]=%d, want %d; trace=%q", i, offsets[i], want, data)
		}
	}
	for i := 10; i < 20; i++ {
		after := 3 - (i - 9)
		if after < 0 {
			after = 0
		}
		want := -after - 1 // encoded as -after-1 below
		if offsets[i] != want {
			t.Fatalf("child downward offset[%d]=%d, want encoded %d; trace=%q", i, offsets[i], want, data)
		}
	}

	// A live mouse-aware child is rendered through a regular cell diff. A
	// physical terminal scroll here would apply the wheel twice and is the
	// visual mechanism behind the reported bottom flicker.
	output := hostOut.String()
	t.Logf("child boundary offsets=%v; raw +1=%d -1=%d; host physical S=%d T=%d", offsets,
		bytes.Count(data, []byte("wheel delta=1")), bytes.Count(data, []byte("wheel delta=-1")),
		strings.Count(output, "\x1b[S"), strings.Count(output, "\x1b[T"))
	if strings.Contains(output, "\x1b[S") || strings.Contains(output, "\x1b[T") {
		t.Fatalf("live child frame used physical scroll: output=%q", output)
	}

	_, _ = input.Write([]byte{0x11})
	waitFor(t, "quit confirmation", func() bool { return bytes.Contains([]byte(hostOut.String()), []byte("Quit tui2?")) })
	_, _ = input.Write([]byte{'\r'})
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("host run: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("host did not stop")
	}
}

// TestHostRealPTYChunkedRedrawDoesNotLeakIntermediateBlankFrames uses a
// child that sends DEC 2026 begin/clear, content, and DEC 2026 end in separate
// writes with 35 ms gaps. A wheel is injected during each clear interval, so
// the host's input-triggered flush path is exercised while the parser contains
// an incomplete frame. The replay must keep showing the previous or the new
// labelled viewport; a blank screen is an intermediate redraw leak.
func TestHostRealPTYChunkedRedrawDoesNotLeakIntermediateBlankFrames(t *testing.T) {
	if os.Getenv(realPTYChildEnv) == "1" {
		t.Skip("child process mode")
	}
	programIn := newMemPipe()
	programOut := newMemPipe()
	proc := &fakeProcess{stdin: programIn, stdout: programOut, stopped: make(chan struct{})}
	input := newMemPipe()
	output := &outputTraceBuffer{}
	tracePath := filepath.Join(t.TempDir(), "child-chunked.log")

	var stateMu sync.Mutex
	var sourceID string
	var client *sdk.Client
	client = sdk.New(programIn, programOut, sdk.Handlers{
		Hello: func(*pb.Hello) {
			_, _ = client.Emit("terminal.create", &pb.MethodParams{Endpoint: "local"}, func(resp *pb.Response) {
				if !resp.GetOk() {
					t.Errorf("create failed: %s", resp.GetError())
					return
				}
				data := resp.GetData()
				stateMu.Lock()
				sourceID = runtime.SourceID(data.GetEndpoint(), data.GetId())
				stateMu.Unlock()
				root := sdk.Row(sdk.Terminal(sourceID).ID("term").Width(78).Height(18).
					Input("key", "wheel").Focused(true))
				if err := client.Commit(root.Build(), sdk.Keys{}); err != nil {
					t.Errorf("commit: %v", err)
				}
			})
		},
	})
	go func() { _ = client.Loop() }()

	host := NewHost(Options{
		Shell: []string{"fake-shell"}, In: input, Out: output,
		NewProcess: func([]string) (process, error) { return proc, nil },
		NewPTY: func(cfg pty.Config) pty.PTY {
			env := append(os.Environ(), realPTYChildEnv+"=1", realPTYScrollChildEnv+"=1",
				realPTYChunkedChildEnv+"=1", realPTYLogEnv+"="+tracePath)
			return &realHistoryPTY{
				PTY: pty.New(pty.Config{
					Argv: []string{os.Args[0], "-test.run=TestTui2RealPTYChild"},
					Env:  env, Cols: cfg.Cols, Rows: cfg.Rows,
				}),
				backend: newTraceHistoryBackend(),
			}
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
	waitFor(t, "real chunked child view", func() bool {
		stateMu.Lock()
		defer stateMu.Unlock()
		return sourceID != "" && client.Rev() >= 1
	})
	termSource, ok := host.handler.TerminalBySource(sourceID)
	if !ok {
		t.Fatalf("terminal %q not found", sourceID)
	}
	waitFor(t, "real chunked child mouse tracking", func() bool { return termSource.Modes().MouseTracking() })
	waitFor(t, "initial chunked redraw", func() bool {
		data, err := os.ReadFile(tracePath)
		return err == nil && bytes.Contains(data, []byte("stage=end offset=0"))
	})
	// Allow the host's normal frame loop to flush the complete initial frame;
	// all subsequent replay checks start after this stable baseline.
	time.Sleep(50 * time.Millisecond)
	baselineWrites := len(output.WritesFrom(0))

	// Send the first wheel to enter offset 1. While that redraw is in its
	// clear interval, send another wheel; repeat through the oldest boundary.
	for i := 0; i < 6; i++ {
		if _, err := input.Write([]byte("\x1b[<64;4;4M")); err != nil {
			t.Fatalf("write chunked wheel %d: %v", i, err)
		}
		wantOffset := i + 1
		if wantOffset > 3 {
			wantOffset = 3
		}
		pattern := []byte(fmt.Sprintf("stage=clear offset=%d", wantOffset))
		clearCount := 1
		if wantOffset == 3 {
			clearCount = i - 1
		}
		waitFor(t, fmt.Sprintf("chunked clear offset %d", wantOffset), func() bool {
			data, err := os.ReadFile(tracePath)
			return err == nil && bytes.Count(data, pattern) >= clearCount
		})
		// Keep the next input inside the 35 ms clear interval. The child is
		// blocked in paint(), while host input remains independently routable.
		if i < 5 {
			continue
		}
	}
	waitFor(t, "chunked redraws complete", func() bool {
		data, err := os.ReadFile(tracePath)
		return err == nil && bytes.Count(data, []byte("stage=end offset=3")) >= 4
	})
	time.Sleep(100 * time.Millisecond)

	frames := replayHostFrames(output.WritesFrom(baselineWrites), 80, 20)
	if len(frames) == 0 {
		t.Fatal("host emitted no frames after chunked redraw input")
	}
	var labels []string
	for index, frame := range frames {
		label, ok := probeOffset(frame)
		if !ok {
			trace, _ := os.ReadFile(tracePath)
			t.Fatalf("host frame %d leaked a blank/intermediate child viewport; writes=%d baseline=%d trace=%q screen=%q",
				index, len(frames), baselineWrites, trace, frame.TextLines())
		}
		labels = append(labels, label)
	}
	t.Logf("chunked host replay frames=%d labels=%v", len(frames), labels)

	_, _ = input.Write([]byte{0x11})
	waitFor(t, "quit confirmation", func() bool { return bytes.Contains([]byte(output.String()), []byte("Quit tui2?")) })
	_, _ = input.Write([]byte{'\r'})
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("host run: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("host did not stop")
	}
}

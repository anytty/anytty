package runtime

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/runtime/keys"
	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/app"
	"github.com/anytty/anytty/clients/tui/sdk/builder"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// programSource is a real layout program driven through the real SDK: one
// commit on HELLO, then one commit per KEY, alternating one leaf so the host
// exercises the delta path. It is built as a separate executable so the
// benchmark measures the two-process OS-pipe path, not an in-process call.
const programSource = `package main

import (
	"os"

	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/app"
	"github.com/anytty/anytty/clients/tui/sdk/builder"
	"github.com/anytty/anytty/proto/ui/protobuf"
)

type model struct{ n int }

func (m *model) Init() app.Cmd { return nil }

func (m *model) Update(msg app.Msg) app.Cmd {
	switch msg.(type) {
	case app.HelloMsg:
		m.n++
	case app.KeyMsg:
		m.n++
	}
	return nil
}

func (m *model) View() *protobuf.Box {
	col := builder.Col()
	for i := 0; i < 40; i++ {
		col.Child(builder.Row(builder.Text("pane"), builder.Text(m.leaf(i))))
	}
	return col.Build()
}

func (m *model) leaf(i int) string {
	if i == 20 {
		return "cell-" + itoa(m.n)
	}
	return "cell-" + itoa(i)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}

func main() {
	client := sdk.New(os.Stdin, os.Stdout, sdk.Handlers{})
	_ = app.Run(client, &model{})
}
`

func buildLayoutProgram(tb testing.TB) string {
	tb.Helper()
	dir := tb.TempDir()
	src := filepath.Join(dir, "prog.go")
	if err := os.WriteFile(src, []byte(programSource), 0o644); err != nil {
		tb.Fatal(err)
	}
	exe := filepath.Join(dir, "prog")
	if out, err := exec.Command("go", "build", "-o", exe, src).CombinedOutput(); err != nil {
		tb.Fatalf("build layout program: %v\n%s", err, out)
	}
	return exe
}

// BenchmarkTwoProcessKeyToFrame is the two-process round trip for one key:
// host Input (encode EVENT, write pipe) -> program Update+Commit (encode
// VIEW_DELTA, write pipe) -> host decodes, applies, composes a frame.
func BenchmarkTwoProcessKeyToFrame(b *testing.B) {
	if os.Getenv("ANYTTY_E2E_BENCH") != "1" {
		b.Skip("set ANYTTY_E2E_BENCH=1 to run the real two-process benchmark")
	}
	exe := buildLayoutProgram(b)
	cmd := exec.Command(exe)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		b.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		b.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		b.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()

	committed := make(chan struct{}, 1)
	hostOut := &countingWriter{w: stdin}
	hostIn := &countingReader{r: stdout}
	s := NewSession(Options{
		ViewID: "bench", Cols: 120, Rows: 40,
		Limits: DefaultLimits(),
		OnView: func() {
			select {
			case committed <- struct{}{}:
			default:
			}
		},
	}, hostIn, hostOut)

	serveDone := make(chan error, 1)
	go func() { serveDone <- s.Serve() }()
	if err := s.SendHello(); err != nil {
		b.Fatal(err)
	}

	// Initial full VIEW.
	waitCommit := func() {
		select {
		case <-committed:
		case <-time.After(5 * time.Second):
			b.Fatal("timed out waiting for a committed view")
		}
	}
	waitCommit()
	_ = serveDone

	key := keys.Event{Kind: keys.KindKey, Key: "a", Char: "a"}
	frameSink := io.Discard
	hostOut.n, hostIn.n = 0, 0
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Input(key); err != nil {
			b.Fatal(err)
		}
		waitCommit()
		if data := s.FrameBytes(nil, nil); len(data) > 0 {
			_, _ = frameSink.Write(data)
		}
	}
	b.ReportMetric(float64(hostOut.n)/float64(b.N), "host_to_prog_B/key")
	b.ReportMetric(float64(hostIn.n)/float64(b.N), "prog_to_host_B/key")
}

// benchModel is the same program logic as programSource, in process.
type benchModel struct{ n int }

func (m *benchModel) Init() app.Cmd { return nil }
func (m *benchModel) Update(msg app.Msg) app.Cmd {
	switch msg.(type) {
	case app.HelloMsg:
		m.n++
	case app.KeyMsg:
		m.n++
	}
	return nil
}
func (m *benchModel) View() *pb.Box {
	col := builder.Col()
	for i := 0; i < 40; i++ {
		text := "cell-" + itoaBench(i)
		if i == 20 {
			text = "cell-" + itoaBench(m.n)
		}
		col.Child(builder.Row(builder.Text("pane"), builder.Text(text)))
	}
	return col.Build()
}

func itoaBench(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

// BenchmarkInProcessKeyToFrame is the same host path with the program logic in
// this process over in-memory pipes. The gap to BenchmarkTwoProcessKeyToFrame
// is the two-process (OS pipe + scheduling) overhead.
func BenchmarkInProcessKeyToFrame(b *testing.B) {
	if os.Getenv("ANYTTY_E2E_BENCH") != "1" {
		b.Skip("set ANYTTY_E2E_BENCH=1 to run the comparison benchmark")
	}
	hostIn, progOut := io.Pipe()
	progIn, hostOut := io.Pipe()

	go func() {
		_ = app.Run(sdk.New(progIn, progOut, sdk.Handlers{}), &benchModel{})
	}()

	committed := make(chan struct{}, 1)
	s := NewSession(Options{
		ViewID: "bench", Cols: 120, Rows: 40, Limits: DefaultLimits(),
		OnView: func() {
			select {
			case committed <- struct{}{}:
			default:
			}
		},
	}, hostIn, hostOut)
	go func() { _ = s.Serve() }()
	if err := s.SendHello(); err != nil {
		b.Fatal(err)
	}
	waitCommit := func() {
		select {
		case <-committed:
		case <-time.After(5 * time.Second):
			b.Fatal("timed out waiting for a committed view")
		}
	}
	waitCommit()

	key := keys.Event{Kind: keys.KindKey, Key: "a", Char: "a"}
	sink := io.Discard
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Input(key); err != nil {
			b.Fatal(err)
		}
		waitCommit()
		if data := s.FrameBytes(nil, nil); len(data) > 0 {
			_, _ = sink.Write(data)
		}
	}
}

var _ = io.Discard
var _ = bytes.MinRead

// BenchmarkHostFrameCompose isolates the host's per-frame composition cost
// (kernel layout + frame diff) with no program, no wire and no pipe: it is the
// floor both the single-process and two-process designs share.
func BenchmarkHostFrameCompose(b *testing.B) {
	root := benchTree(40)
	s := NewSession(Options{ViewID: "bench", Cols: 120, Rows: 40, Limits: DefaultLimits()},
		bytes.NewReader(nil), io.Discard)
	if err := s.HandleView(&pb.View{Epoch: 1, Rev: 1, Root: root}); err != nil {
		b.Fatal(err)
	}
	// Prime the previous-frame diff, then alternate one leaf so every frame is
	// a real change (not the no-op early return).
	s.FrameBytes(nil, nil)
	b.ReportAllocs()
	b.ResetTimer()
	rev := uint64(1)
	for i := 0; i < b.N; i++ {
		rev++
		patch := &pb.ViewDelta{Epoch: 1, Rev: rev, RevBase: rev - 1, Patches: []*pb.Patch{{
			Op: "replace", Path: []uint32{20, 0},
			Box: &pb.Box{Content: &pb.Content{Text: "cell-" + itoaBench(i)}},
		}}}
		if err := s.HandleViewDelta(patch); err != nil {
			b.Fatal(err)
		}
		_ = s.FrameBytes(nil, nil)
	}
}

// BenchmarkHostFrameComposeStatic measures FrameBytes on an unchanged tree:
// the host still composes the full frame and diffs it, so this is the floor
// the frame loop pays per wake even when nothing visible changed.
func BenchmarkHostFrameComposeStatic(b *testing.B) {
	root := benchTree(40)
	s := NewSession(Options{ViewID: "bench", Cols: 120, Rows: 40, Limits: DefaultLimits()},
		bytes.NewReader(nil), io.Discard)
	if err := s.HandleView(&pb.View{Epoch: 1, Rev: 1, Root: root}); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.FrameBytes(nil, nil)
	}
}

// countingWriter counts bytes written (wire bytes host -> program).
type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	c.n += int64(len(p))
	return c.w.Write(p)
}

// countingReader counts bytes read (wire bytes program -> host).
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

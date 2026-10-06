package endpoint

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/clients/tui/render/ansi"
	"github.com/anytty/anytty/proto/access/apipb"
)

func testManager(t *testing.T) *Manager {
	t.Helper()
	m := NewManager(Options{
		DialTimeout: time.Second,
		CallTimeout: time.Second,
		BackoffMin:  20 * time.Millisecond,
		BackoffMax:  60 * time.Millisecond,
		Dial:        dialRawSession,
	})
	t.Cleanup(func() { _ = m.Close() })
	return m
}

func registerDaemon(t *testing.T, m *Manager, d *fakeDaemon, name string) {
	t.Helper()
	if err := m.Register(Config{Name: name, Kind: KindDaemon, Socket: d.socket}); err != nil {
		t.Fatalf("register endpoint: %v", err)
	}
	waitFor(t, 3*time.Second, func() bool { return m.Health(name) == HealthOK }, "endpoint health ok")
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// readCollector drains a RemotePTY in the background.
type readCollector struct {
	mu  sync.Mutex
	buf []byte
	err error
}

func startReader(p *RemotePTY) *readCollector {
	c := &readCollector{}
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := p.Read(buf)
			c.mu.Lock()
			if n > 0 {
				c.buf = append(c.buf, buf[:n]...)
			}
			if err != nil {
				c.err = err
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()
		}
	}()
	return c
}

func (c *readCollector) text() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return string(c.buf)
}

func (c *readCollector) readErr() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func waitText(t *testing.T, c *readCollector, needle string) {
	t.Helper()
	waitFor(t, 3*time.Second, func() bool { return strings.Contains(c.text(), needle) }, "output "+needle)
}

func createTerminal(t *testing.T, m *Manager, d *fakeDaemon, endpoint string) string {
	t.Helper()
	info, err := m.Create(context.Background(), endpoint, &apipb.TerminalCreateSpec{
		Name:    "demo",
		Command: []string{"sh"},
		Size:    &apipb.TerminalSize{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create terminal: %v", err)
	}
	id := info.GetRef().GetTerminalId()
	if id == "" {
		t.Fatal("create returned an empty terminal id")
	}
	if d.terminal(id) == nil {
		t.Fatalf("terminal %s not registered on the fake daemon", id)
	}
	return id
}

func TestListCreateAttachInputResizeKill(t *testing.T) {
	d := newFakeDaemon(t)
	m := testManager(t)
	registerDaemon(t, m, d, "dev")
	id := createTerminal(t, m, d, "dev")

	statuses := m.Snapshot()
	if len(statuses) != 1 || statuses[0].ID != id || statuses[0].Health != HealthOK || statuses[0].Attached {
		t.Fatalf("snapshot after create = %+v", statuses)
	}

	p := m.NewRemotePTY(pty.Config{Endpoint: "dev", ID: id, Cols: 80, Rows: 24})
	if err := p.Start(); err != nil {
		t.Fatalf("remote pty start: %v", err)
	}
	reader := startReader(p)

	if _, err := p.Write([]byte("echo hello\r")); err != nil {
		t.Fatalf("write: %v", err)
	}
	waitText(t, reader, "hello")

	if _, err := p.Write([]byte("stty size\r")); err != nil {
		t.Fatalf("write stty: %v", err)
	}
	waitText(t, reader, "24 80")

	if err := p.Resize(100, 30); err != nil {
		t.Fatalf("resize: %v", err)
	}
	waitFor(t, time.Second, func() bool {
		size := d.terminal(id).sizeProto()
		return size.GetCols() == 100 && size.GetRows() == 30
	}, "daemon terminal resize")
	if _, err := p.Write([]byte("stty size\r")); err != nil {
		t.Fatalf("write stty after resize: %v", err)
	}
	waitText(t, reader, "30 100")

	if err := m.Kill(context.Background(), "dev", id); err != nil {
		t.Fatalf("kill: %v", err)
	}
	waitFor(t, 3*time.Second, func() bool { return p.Exited() }, "remote pty exit")
	if code := p.ExitCode(); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	// 远端退出后读取应终止；错误观测可能有毫秒级延迟，这里等待而不是立即断言（避免时序 flake）。
	waitFor(t, time.Second, func() bool { return reader.readErr() != nil }, "remote read terminates after kill")
	waitFor(t, time.Second, func() bool {
		for _, status := range m.Snapshot() {
			if status.ID == id {
				return status.Exited
			}
		}
		return false
	}, "snapshot exit state")
}

func TestReconnectResubscribesAndSeedsSnapshot(t *testing.T) {
	d := newFakeDaemon(t)
	m := testManager(t)
	var noticeMu sync.Mutex
	var notices []string
	m.SetOnNotice(func(level, message string) {
		noticeMu.Lock()
		notices = append(notices, level+":"+message)
		noticeMu.Unlock()
	})
	registerDaemon(t, m, d, "dev")
	id := createTerminal(t, m, d, "dev")

	p := m.NewRemotePTY(pty.Config{Endpoint: "dev", ID: id, Cols: 80, Rows: 24})
	if err := p.Start(); err != nil {
		t.Fatalf("remote pty start: %v", err)
	}
	reader := startReader(p)
	if _, err := p.Write([]byte("echo first\r")); err != nil {
		t.Fatalf("write: %v", err)
	}
	waitText(t, reader, "first")

	d.disconnect()
	waitFor(t, 3*time.Second, func() bool { return m.Health("dev") == HealthOffline }, "health offline")
	waitFor(t, 3*time.Second, func() bool { return m.Health("dev") == HealthOK }, "health back online")
	waitFor(t, 3*time.Second, func() bool {
		noticeMu.Lock()
		defer noticeMu.Unlock()
		return len(notices) >= 2
	}, "offline/online notices")
	noticeMu.Lock()
	joined := strings.Join(notices, "\n")
	noticeMu.Unlock()
	if !strings.Contains(joined, "offline") || !strings.Contains(joined, "connected") {
		t.Fatalf("notices = %q", joined)
	}
	if p.Exited() {
		t.Fatal("remote pty must stay alive across a reconnect")
	}
	// The reconnect must re-attach and re-seed the authoritative snapshot:
	// the first output appears again without replaying a stream.
	waitFor(t, 3*time.Second, func() bool { return strings.Count(reader.text(), "first") >= 2 }, "snapshot re-seed")
	if _, err := p.Write([]byte("echo again\r")); err != nil {
		t.Fatalf("write after reconnect: %v", err)
	}
	waitText(t, reader, "again")

	// Manager close detaches but must not kill the daemon terminal.
	if err := m.Close(); err != nil {
		t.Fatalf("manager close: %v", err)
	}
	if d.terminalCount() != 1 {
		t.Fatalf("daemon terminal count after close = %d, want 1", d.terminalCount())
	}
	waitFor(t, time.Second, func() bool { return p.Closed() }, "remote pty closed")
	waitFor(t, time.Second, func() bool { return reader.readErr() != nil }, "reader observes close")
	if err := reader.readErr(); !errors.Is(err, pty.ErrClosed) {
		t.Fatalf("read error after close = %v, want ErrClosed", err)
	}
}

func TestUnsupportedModesAndErrors(t *testing.T) {
	d := newFakeDaemon(t)
	m := testManager(t)
	registerDaemon(t, m, d, "dev")

	if err := m.Register(Config{Name: "webrtc", Kind: KindDaemon, Socket: d.socket, ConnectMode: ConnectDirectWebRTC}); err != nil {
		t.Fatalf("register webrtc endpoint: %v", err)
	}
	// The production dialer is the shared layer: a direct endpoint without
	// registry-provided locators stays listed/offline with a readable error.
	sharedM := NewManager(Options{DialTimeout: 200 * time.Millisecond, CallTimeout: 200 * time.Millisecond, BackoffMin: 20 * time.Millisecond, BackoffMax: 60 * time.Millisecond, RegistryPath: hermeticRegistryPath(t)})
	t.Cleanup(func() { _ = sharedM.Close() })
	if err := sharedM.Register(Config{Name: "webrtc", Kind: KindDaemon, Socket: d.socket, ConnectMode: ConnectDirectWebRTC}); err != nil {
		t.Fatalf("register shared-webrtc endpoint: %v", err)
	}
	waitFor(t, 3*time.Second, func() bool { return sharedM.Health("webrtc") == HealthOffline }, "webrtc endpoint offline")
	p := sharedM.NewRemotePTY(pty.Config{Endpoint: "webrtc", ID: "x", Cols: 80, Rows: 24})
	if err := p.Start(); err == nil || !strings.Contains(err.Error(), "signaling") {
		t.Fatalf("webrtc start error = %v; want readable missing-route error", err)
	}

	if err := m.Register(Config{Name: "bad", Kind: KindDaemon}); err == nil {
		t.Fatal("registering a daemon endpoint without a socket must fail")
	} else if !strings.Contains(err.Error(), "socket") {
		t.Fatalf("missing socket error = %v", err)
	}
	if err := m.Register(Config{Name: "tcp-no-address", Kind: KindDaemon, ConnectMode: ConnectDirectTCP}); err == nil {
		t.Fatal("registering a tcp endpoint without an address must fail")
	} else if !strings.Contains(err.Error(), "address") {
		t.Fatalf("missing tcp address error = %v", err)
	}

	missing := m.NewRemotePTY(pty.Config{Endpoint: "dev", ID: "nope", Cols: 80, Rows: 24})
	err := missing.Start()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("attach missing terminal error = %v", err)
	}
}

func TestResizeOwnerConflictIsReadable(t *testing.T) {
	d := newFakeDaemon(t)
	m := testManager(t)
	registerDaemon(t, m, d, "dev")
	id := createTerminal(t, m, d, "dev")

	p := m.NewRemotePTY(pty.Config{Endpoint: "dev", ID: id, Cols: 80, Rows: 24})
	if err := p.Start(); err != nil {
		t.Fatalf("remote pty start: %v", err)
	}
	defer func() { _ = p.Close() }()

	terminal := d.terminal(id)
	terminal.mu.Lock()
	terminal.owner = &fakeAttach{terminal: terminal}
	terminal.epoch++
	terminal.mu.Unlock()

	err := p.Resize(90, 20)
	if err == nil || !strings.Contains(err.Error(), "resize") || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("resize conflict error = %v", err)
	}
}

func TestRenderScreenSnapshotFeedsParser(t *testing.T) {
	screen := &apipb.NativeScreenResult{
		Size:        &apipb.TerminalSize{Cols: 10, Rows: 3},
		FullReplace: true,
		RowReplacements: []*apipb.ScreenRowReplace{
			{RowIndex: 0, Row: &apipb.ScreenRow{Cells: []*apipb.ScreenCell{
				{Content: "hi", Width: 2, Style: &apipb.CellStyle{Bold: true, Foreground: "ansi:2"}},
				{Content: "x", Width: 1, Style: &apipb.CellStyle{Foreground: "idx:5"}},
				{Content: "y", Width: 1, Style: &apipb.CellStyle{Foreground: "#ff0000"}},
			}}},
			{RowIndex: 1, Row: &apipb.ScreenRow{Cells: []*apipb.ScreenCell{
				{Content: "ok", Width: 2, Style: &apipb.CellStyle{Underline: true}},
			}}},
		},
		Cursor: &apipb.TerminalCursor{Row: 1, Col: 2, Visible: true, Shape: apipb.CursorShape_CURSOR_SHAPE_BAR},
		Modes:  &apipb.TerminalModes{BracketedPaste: true, MouseSgr: true, AlternateScreen: true, AutoWrap: true},
	}
	rendered := renderScreenSnapshot(screen)
	for _, want := range []string{"\x1b[?1049h", "\x1b[?1006h", "\x1b[?2004h", "38;5;5", "38;2;255;0;0", "\x1b[6 q", "\x1b[2;3H", "\x1b[?25h"} {
		if !strings.Contains(string(rendered), want) {
			t.Fatalf("snapshot output missing %q: %q", want, rendered)
		}
	}
	parser := ansi.New(10, 3)
	parser.Write([]byte("\x1b[?2026h"))
	parser.Write(rendered)
	snapshot := parser.Screen()
	if parser.Modes().SynchronizedOutput {
		t.Fatal("snapshot must close a synchronized-output batch from the old attachment")
	}
	if got := strings.TrimRight(snapshot.Text(0), " "); got != "hixy" {
		t.Fatalf("parser row 0 = %q, want %q", got, "hixy")
	}
	if got := strings.TrimRight(snapshot.Text(1), " "); got != "ok" {
		t.Fatalf("parser row 1 = %q, want %q", got, "ok")
	}
	if modes := parser.Modes(); !modes.BracketPaste || !modes.MouseSGR || !modes.AltScreen {
		t.Fatalf("parser modes = %+v", modes)
	}
	if got := snapshot.CursorShape; got != "bar" {
		t.Fatalf("snapshot cursor shape = %q, want bar", got)
	}
}

func TestRenderScreenSnapshotPreservesBlankBackgroundRows(t *testing.T) {
	gray := &apipb.CellStyle{Background: "idx:236"}
	screen := &apipb.NativeScreenResult{
		Size:        &apipb.TerminalSize{Cols: 6, Rows: 3},
		FullReplace: true,
		Cursor:      &apipb.TerminalCursor{Row: 1, Col: 0, Visible: true},
		RowReplacements: []*apipb.ScreenRowReplace{
			{RowIndex: 0, Row: &apipb.ScreenRow{Cells: []*apipb.ScreenCell{{Width: 6, Style: gray}}}},
			{RowIndex: 1, Row: &apipb.ScreenRow{Cells: []*apipb.ScreenCell{
				{Content: "x", Width: 1, Style: gray},
				{Width: 5, Style: gray},
			}}},
			{RowIndex: 2, Row: &apipb.ScreenRow{TailFill: gray}},
		},
	}
	parser := ansi.New(6, 3)
	parser.Write(renderScreenSnapshot(screen))
	snapshot := parser.Screen()
	for _, cell := range []struct{ x, y int }{{0, 0}, {5, 0}, {1, 1}, {5, 1}, {0, 2}, {5, 2}} {
		got := snapshot.CellAt(cell.x, cell.y)
		if string(got.Style) != "ansi:48;5;236" {
			t.Fatalf("blank cell (%d,%d) style = %q, want gray background", cell.x, cell.y, got.Style)
		}
	}
	if got := strings.TrimRight(snapshot.Text(1), " "); got != "x" {
		t.Fatalf("blank background changed logical text: %q", got)
	}
	parser.Write([]byte("z"))
	snapshot = parser.Screen()
	if got := snapshot.CellAt(0, 0).Style; got != "ansi:48;5;236" {
		t.Fatalf("snapshot reset changed existing row style: %q", got)
	}
	if got := snapshot.CellAt(0, 2).Style; got != "ansi:48;5;236" {
		t.Fatalf("snapshot reset changed tail-fill row style: %q", got)
	}
	if got := snapshot.CellAt(0, 1).Text; got != "z" {
		t.Fatalf("post-snapshot PTY text = %q, want z", got)
	}
}

func TestRemotePTYSeedSnapshotDropsStaleBufferedOutput(t *testing.T) {
	// A reconnect snapshot is the new stream generation's synchronization
	// point. Bytes left by the old attachment must not be replayed after it.
	p := &RemotePTY{buf: []byte("stale-old-attachment")}
	p.cond = sync.NewCond(&p.mu)
	reset := false
	p.SetSnapshotReset(func() { reset = true })
	p.seedSnapshot(&apipb.NativeScreenResult{
		Size:        &apipb.TerminalSize{Cols: 5, Rows: 1},
		FullReplace: true,
		RowReplacements: []*apipb.ScreenRowReplace{{
			RowIndex: 0,
			Row:      &apipb.ScreenRow{Cells: []*apipb.ScreenCell{{Content: "fresh", Width: 5}}},
		}},
	})
	got := make([]byte, 256)
	n, err := p.Read(got)
	if err != nil {
		t.Fatalf("read seeded snapshot: %v", err)
	}
	got = got[:n]
	if bytes.Contains(got, []byte("stale-old-attachment")) {
		t.Fatalf("reconnect snapshot replayed stale bytes: %q", got)
	}
	if !bytes.Contains(got, []byte("fresh")) {
		t.Fatalf("reconnect snapshot omitted fresh screen: %q", got)
	}
	if !reset {
		t.Fatal("reconnect snapshot did not reset the previous parser generation")
	}
}

func TestRemotePTYSnapshotResetWaitsUntilSnapshotRead(t *testing.T) {
	// The pump can have an old chunk in hand when a reconnect publishes the
	// snapshot. Resetting from seedSnapshot would let that old chunk land in a
	// freshly reset parser. The reset hook must run on the first snapshot Read,
	// after the pump has returned from the old Read.
	p := &RemotePTY{buf: []byte("old-output")}
	p.cond = sync.NewCond(&p.mu)
	reset := false
	p.SetSnapshotReset(func() { reset = true })
	buf := make([]byte, 64)
	n, err := p.Read(buf)
	if err != nil || string(buf[:n]) != "old-output" {
		t.Fatalf("read old output = %q, err=%v", buf[:n], err)
	}
	if reset {
		t.Fatal("snapshot reset ran before the authoritative snapshot was read")
	}
	p.seedSnapshot(&apipb.NativeScreenResult{
		Size:        &apipb.TerminalSize{Cols: 5, Rows: 1},
		FullReplace: true,
		RowReplacements: []*apipb.ScreenRowReplace{{
			RowIndex: 0,
			Row:      &apipb.ScreenRow{Cells: []*apipb.ScreenCell{{Content: "fresh", Width: 5}}},
		}},
	})
	n, err = p.Read(buf)
	if err != nil || n == 0 {
		t.Fatalf("read fresh snapshot = %q, err=%v", buf[:n], err)
	}
	if !reset {
		t.Fatal("snapshot reset did not run before fresh bytes were returned")
	}
}

func TestRemotePTYSeedSnapshotWaitsForInFlightReadBarrier(t *testing.T) {
	// Exercise the narrower race where Read has copied old bytes but the
	// runtime has not committed that chunk to its parser yet. A reconnect must
	// wait for that parser transaction before running the reset hook.
	p := &RemotePTY{buf: []byte("old-output")}
	p.cond = sync.NewCond(&p.mu)
	readReturned := make(chan struct{})
	allowReadDone := make(chan struct{})
	readDone := make(chan struct{})
	buf := make([]byte, 64)
	go func() {
		n, err := p.Read(buf)
		if err != nil || string(buf[:n]) != "old-output" {
			t.Errorf("read old output = %q, err=%v", buf[:n], err)
		}
		close(readReturned)
		<-allowReadDone
		p.ReadDone()
		close(readDone)
	}()
	select {
	case <-readReturned:
	case <-time.After(time.Second):
		t.Fatal("old read did not return")
	}
	barrierStarted := make(chan struct{})
	barrierRelease := make(chan struct{})
	barrierFinished := make(chan struct{})
	p.SetSnapshotBarrier(func() {
		close(barrierStarted)
		<-barrierRelease
		close(barrierFinished)
	})
	go p.seedSnapshot(&apipb.NativeScreenResult{
		Size:        &apipb.TerminalSize{Cols: 5, Rows: 1},
		FullReplace: true,
		RowReplacements: []*apipb.ScreenRowReplace{{
			RowIndex: 0,
			Row:      &apipb.ScreenRow{Cells: []*apipb.ScreenCell{{Content: "fresh", Width: 5}}},
		}},
	})
	select {
	case <-barrierStarted:
	case <-time.After(time.Second):
		t.Fatal("snapshot did not observe the in-flight old read")
	}
	select {
	case <-barrierFinished:
		t.Fatal("snapshot barrier finished before parser transaction release")
	default:
	}
	close(allowReadDone)
	select {
	case <-readDone:
	case <-time.After(time.Second):
		t.Fatal("old read transaction did not finish")
	}
	close(barrierRelease)
	select {
	case <-barrierFinished:
	case <-time.After(time.Second):
		t.Fatal("snapshot barrier did not finish")
	}
}

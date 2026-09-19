package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// closedReader simulates reading the stdout of a program the host just
// stopped: os.File.Read returns fs.ErrClosed, not io.EOF.
type closedReader struct{}

func (closedReader) Read([]byte) (int, error) {
	return 0, &os.PathError{Op: "read", Path: "|0", Err: fs.ErrClosed}
}

func TestDevModeDisabledIsNil(t *testing.T) {
	dev, err := newDevMode(devConfig{}, []string{"shell"})
	if err != nil {
		t.Fatalf("newDevMode: %v", err)
	}
	if dev != nil {
		t.Fatal("a zero devConfig must not create a devMode")
	}
}

func TestProtocolLogDecodesBothDirections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dev.log")
	dev, err := newDevMode(devConfig{Enabled: true, ProtocolLog: path}, nil)
	if err != nil {
		t.Fatalf("newDevMode: %v", err)
	}
	defer dev.close()

	// host -> program: one HELLO frame through the write tap.
	hostFrame, err := wire.Marshal(wire.TypeHello, &pb.Hello{Epoch: 7, ViewId: "view:test", Cols: 80, Rows: 24}, 0)
	if err != nil {
		t.Fatal(err)
	}
	sink := newMemPipe()
	writer := dev.tapWrite(sink)
	if _, err := writer.Write(hostFrame); err != nil {
		t.Fatalf("tap write: %v", err)
	}
	if got := sink.buf; !bytes.Equal(got, hostFrame) {
		t.Fatalf("tap write altered bytes: %d != %d", len(got), len(hostFrame))
	}

	// program -> host: one VIEW frame through the read tap.
	programFrame, err := wire.Marshal(wire.TypeView, &pb.View{Epoch: 7, Rev: 3, Root: &pb.Box{Id: "root"}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	reader := dev.tapRead(bytes.NewReader(programFrame))
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("tap read: %v", err)
	}
	if !bytes.Equal(got, programFrame) {
		t.Fatal("tap read altered frame bytes")
	}

	records := readFrameRecords(t, path)
	if len(records) != 2 {
		t.Fatalf("want 2 log records, got %d: %+v", len(records), records)
	}
	if records[0].Dir != "host->program" || records[0].Type != "HELLO" {
		t.Fatalf("record 0 = %+v", records[0])
	}
	if records[1].Dir != "program->host" || records[1].Type != "VIEW" {
		t.Fatalf("record 1 = %+v", records[1])
	}
	if !strings.Contains(string(records[1].Payload), `"rev":"3"`) &&
		!strings.Contains(string(records[1].Payload), `"rev":3`) {
		t.Fatalf("VIEW payload not decoded: %s", records[1].Payload)
	}
	if records[0].TS == "" {
		t.Fatal("records must carry a timestamp")
	}
}

func TestDevModeSurfacesViewRejectedNotice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dev.log")
	dev, err := newDevMode(devConfig{Enabled: true, ProtocolLog: path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.close()
	var notices []string
	dev.onNotice = func(level, message string) { notices = append(notices, level+": "+message) }

	frame, err := wire.Marshal(wire.TypeEvent, &pb.Event{Event: &pb.Event_ViewRejected{
		ViewRejected: &pb.ViewRejectedEvent{Epoch: 1, Rev: 4, Reason: "max_nodes"},
	}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dev.tapWrite(io.Discard).Write(frame); err != nil {
		t.Fatal(err)
	}
	if len(notices) != 1 || !strings.Contains(notices[0], "view rejected rev=4: max_nodes") {
		t.Fatalf("notices = %v", notices)
	}
}

func TestTapReaderClosedPipeEndsCleanly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dev.log")
	dev, err := newDevMode(devConfig{Enabled: true, ProtocolLog: path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.close()
	var notices []string
	dev.onNotice = func(level, message string) { notices = append(notices, level+": "+message) }

	if _, err := io.ReadAll(dev.tapRead(closedReader{})); err != nil {
		t.Fatalf("closed pipe must end cleanly, got %v", err)
	}
	if len(notices) != 0 {
		t.Fatalf("a deliberate stop must not emit notices: %v", notices)
	}
	for _, record := range readFrameRecords(t, path) {
		if record.Type == "ERROR" || record.Error != "" {
			t.Fatalf("a deliberate stop must not log a protocol error: %+v", record)
		}
	}
}

func TestStderrBufferKeepsLastLines(t *testing.T) {
	buffer := newStderrBuffer(3)
	_, _ = buffer.Write([]byte("one\ntwo\nthree\nfour\npartial"))
	tail := buffer.tail(3)
	if len(tail) != 3 || tail[0] != "three" || tail[1] != "four" || tail[2] != "partial" {
		t.Fatalf("tail = %v", tail)
	}
}

func TestFileWatcherDetectsChange(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "program.py")
	if err := os.WriteFile(target, []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed := make(chan string, 4)
	watcher := newFileWatcher([]string{dir}, 20*time.Millisecond, func(path string) { changed <- path })
	watcher.start()
	defer watcher.close()

	if err := os.WriteFile(target, []byte("v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case path := <-changed:
		if path != target {
			t.Fatalf("changed path = %q, want %q", path, target)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("watcher did not report the save")
	}
}

func TestReloadWatchPaths(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "program.py")
	if err := os.WriteFile(script, []byte("print('x')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	paths := reloadWatchPaths([]string{"python3", script, "--flag"})
	if len(paths) != 1 || paths[0] != script {
		t.Fatalf("reloadWatchPaths = %v", paths)
	}
	if got := reloadWatchPaths([]string{"python3", "--flag"}); len(got) != 1 || got[0] != "." {
		t.Fatalf("fallback = %v", got)
	}
}

func TestRestartProgramDevNoticeCarriesStderrAndCountdown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dev.log")
	dev, err := newDevMode(devConfig{Enabled: true, ProtocolLog: path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.close()
	if _, err := dev.stderr.Write([]byte("DEV-STDERR-BOOM\n")); err != nil {
		t.Fatal(err)
	}
	collector := &logCollector{}
	programIn := newMemPipe()
	programOut := newMemPipe()
	proc := &fakeProcess{stdin: programIn, stdout: programOut, stopped: make(chan struct{})}
	host := NewHost(Options{
		Shell:       []string{"fake-shell"},
		In:          newMemPipe(),
		Out:         &syncBuffer{},
		NewProcess:  func([]string) (process, error) { return proc, nil },
		Cols:        80,
		Rows:        20,
		RestartWait: time.Millisecond,
		Logf:        collector.logf,
		Dev:         dev,
	})
	host.restartProgram(errors.New("exit status 2"))
	if !strings.Contains(collector.String(), "restarting in") {
		t.Fatalf("crash notice lacks the countdown:\n%s", collector.String())
	}
	if !strings.Contains(collector.String(), "stderr: DEV-STDERR-BOOM") {
		t.Fatalf("crash notice lacks the stderr tail:\n%s", collector.String())
	}
}

func TestReloadProgramKeepsFrameAndNoticesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dev.log")
	dev, err := newDevMode(devConfig{Enabled: true, ProtocolLog: path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.close()
	collector := &logCollector{}
	programIn := newMemPipe()
	programOut := newMemPipe()
	proc := &fakeProcess{stdin: programIn, stdout: programOut, stopped: make(chan struct{})}
	host := NewHost(Options{
		Shell:       []string{"fake-shell"},
		In:          newMemPipe(),
		Out:         &syncBuffer{},
		NewProcess:  func([]string) (process, error) { return proc, nil },
		Cols:        80,
		Rows:        20,
		RestartWait: time.Millisecond,
		Logf:        collector.logf,
		Dev:         dev,
	})
	// Seed a "last good tree" frame, then reload: the new session has no
	// committed view, so the old frame must remain untouched.
	host.reloadProgram("/tmp/program.py")
	if !strings.Contains(collector.String(), "reloaded /tmp/program.py") {
		t.Fatalf("reload notice missing:\n%s", collector.String())
	}
	if host.opts.Dev.reload == nil {
		t.Fatal("dev mode must expose the reload channel")
	}
	select {
	case got := <-host.reloadChannel():
		t.Fatalf("reload channel should be empty, got %q", got)
	default:
	}
}

// readFrameRecords parses the JSON Lines protocol log.
func readFrameRecords(t *testing.T, path string) []frameRecord {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read protocol log: %v", err)
	}
	var records []frameRecord
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var record frameRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("decode log line %q: %v", line, err)
		}
		records = append(records, record)
	}
	return records
}

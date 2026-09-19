package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
	"github.com/anytty/anytty/shared/userdirs"
	"google.golang.org/protobuf/encoding/protojson"
)

// devProtocolLogName is the default frame log written by -dev when
// -protocol-log is not given.
const devProtocolLogName = "tui2-dev.log"

// devWatchInterval is the polling period of the hot-reload file watcher.
const devWatchInterval = 200 * time.Millisecond

// devConfig is the parsed -dev request from the command line. A zero value
// disables every instrumentation, so the production host is untouched.
type devConfig struct {
	// Enabled is -dev: default protocol log, stderr capture, dev notices.
	Enabled bool
	// ProtocolLog is -protocol-log <file>; empty means the -dev default.
	ProtocolLog string
	// Watch is -watch <path>[,...]: files/directories that trigger a reload.
	Watch []string
	// ReloadOnSave is -reload-on-save: derive the watch list from the layout
	// program arguments.
	ReloadOnSave bool
}

// active reports whether any instrumentation was requested.
func (c devConfig) active() bool {
	return c.Enabled || c.ProtocolLog != "" || len(c.Watch) > 0 || c.ReloadOnSave
}

// devMode carries the tui2 -dev instrumentation: a decoded frame log, layout
// program stderr capture, a polling file watcher and the reload channel the
// frame loop selects on. Nil means production behavior.
type devMode struct {
	enabled  bool
	log      *protocolLog
	stderr   *stderrBuffer
	watcher  *fileWatcher
	reload   chan string
	onNotice func(level, message string)

	noticeMu sync.Mutex
	lastErr  string
}

// newDevMode opens the dev instrumentation for one host run. argv is the
// resolved layout program command line (used by -reload-on-save). It returns
// (nil, nil) when no dev flag is set, so callers can pass the result
// straight into Options.Dev.
func newDevMode(cfg devConfig, argv []string) (*devMode, error) {
	if !cfg.active() {
		return nil, nil
	}
	dev := &devMode{enabled: cfg.Enabled, reload: make(chan string, 1)}
	if cfg.Enabled {
		dev.stderr = newStderrBuffer(64)
	}
	logPath := strings.TrimSpace(cfg.ProtocolLog)
	if logPath == "" && cfg.Enabled {
		logPath = filepath.Join(userdirs.StateHome(), "anytty", devProtocolLogName)
	}
	if logPath != "" {
		log, err := openProtocolLog(logPath)
		if err != nil {
			return nil, err
		}
		dev.log = log
	}
	paths := append([]string(nil), cfg.Watch...)
	if cfg.ReloadOnSave {
		paths = append(paths, reloadWatchPaths(argv)...)
	}
	if len(paths) > 0 {
		dev.watcher = newFileWatcher(uniquePaths(paths), devWatchInterval, func(path string) {
			select {
			case dev.reload <- path:
			default:
			}
		})
		dev.watcher.start()
	}
	return dev, nil
}

// close releases the log file and stops the watcher.
func (d *devMode) close() {
	if d == nil {
		return
	}
	if d.watcher != nil {
		d.watcher.close()
	}
	if d.log != nil {
		d.log.close()
	}
}

// startProcess launches the layout program while capturing its stderr into
// the dev ring buffer (dev mode surfaces the tail in crash notices).
func (d *devMode) startProcess(argv []string) (process, error) {
	return startExecProcessTo(argv, d.stderr)
}

// tapRead wraps the program -> host pipe: every complete frame is decoded
// into the frame log (and diagnosed) before the session decoder sees it.
func (d *devMode) tapRead(r io.Reader) io.Reader {
	if d == nil || d.log == nil {
		return r
	}
	return &tapReader{src: r, dev: d}
}

// tapWrite wraps the host -> program pipe the same way.
func (d *devMode) tapWrite(w io.Writer) io.Writer {
	if d == nil || d.log == nil {
		return w
	}
	return &tapWriter{dst: w, dev: d}
}

// stderrTail returns the last captured layout program stderr lines.
func (d *devMode) stderrTail() []string {
	if d == nil || d.stderr == nil {
		return nil
	}
	return d.stderr.tail(3)
}

// notice forwards a dev diagnostic to the host notice queue (the layout
// program renders it in its status line), deduplicating repeated protocol
// errors so a broken program cannot flood the footer.
func (d *devMode) notice(level, message string) {
	if d == nil || !d.enabled || d.onNotice == nil {
		return
	}
	d.noticeMu.Lock()
	if level == "warning" && strings.HasPrefix(message, "protocol error:") {
		if d.lastErr == message {
			d.noticeMu.Unlock()
			return
		}
		d.lastErr = message
	}
	d.noticeMu.Unlock()
	d.onNotice(level, message)
}

// reloadWatchPaths derives the -reload-on-save watch list from the resolved
// layout program argv: every existing file/directory argument (the script
// itself, a source directory), or the working directory when nothing exists.
func reloadWatchPaths(argv []string) []string {
	var paths []string
	for _, arg := range argv {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if _, err := os.Stat(arg); err == nil {
			paths = append(paths, arg)
		}
	}
	if len(paths) == 0 {
		paths = append(paths, ".")
	}
	return paths
}

func uniquePaths(paths []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	return out
}

// protocolLog writes one JSON line per protocol frame, both directions, with
// a timestamp, direction and the decoded protobuf payload.
type protocolLog struct {
	mu  sync.Mutex
	f   *os.File
	enc *json.Encoder
}

// frameRecord is one JSON line of the -protocol-log file.
type frameRecord struct {
	TS      string          `json:"ts"`
	Dir     string          `json:"dir"`
	Type    string          `json:"type"`
	Bytes   int             `json:"bytes"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

func openProtocolLog(path string) (*protocolLog, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("protocol log %s: %w", path, err)
		}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("protocol log %s: %w", path, err)
	}
	encoder := json.NewEncoder(file)
	// Keep "host->program" readable (no \u003e HTML escaping).
	encoder.SetEscapeHTML(false)
	return &protocolLog{f: file, enc: encoder}, nil
}

func (l *protocolLog) close() {
	if l == nil || l.f == nil {
		return
	}
	_ = l.f.Close()
	l.f = nil
}

func (l *protocolLog) write(record frameRecord) {
	if l == nil || l.f == nil {
		return
	}
	record.TS = time.Now().Format(time.RFC3339Nano)
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = l.enc.Encode(record)
}

// logFrame decodes one complete raw frame (u32 length | u8 type | payload)
// and writes it as a readable JSON line. Decode failures are logged with an
// error field so the file stays a faithful trace.
func (d *devMode) logFrame(dir string, frame []byte) {
	if d == nil || d.log == nil {
		return
	}
	record := frameRecord{Dir: dir, Bytes: len(frame)}
	if len(frame) < 5 {
		record.Type = "UNKNOWN"
		record.Error = "short frame"
		d.log.write(record)
		return
	}
	t := wire.Type(frame[4])
	record.Type = t.String()
	legal := wire.RoleProgram.CanReceive(t)
	if dir == "program->host" {
		legal = wire.RoleHost.CanReceive(t)
	}
	if !legal {
		record.Error = fmt.Sprintf("illegal %s frame in %s direction", t, dir)
		d.log.write(record)
		d.notice("warning", "protocol error: "+record.Error)
		return
	}
	message, err := wire.UnmarshalPayload(t, frame[5:])
	if err != nil {
		record.Error = err.Error()
		d.log.write(record)
		d.notice("warning", "protocol error: "+err.Error())
		return
	}
	if payload, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(message); err == nil {
		record.Payload = payload
	} else {
		record.Error = err.Error()
	}
	d.log.write(record)
	// view_rejected is the host answer to an invalid VIEW; in dev mode it is
	// surfaced as a notice on top of the protocol log line.
	if event, ok := message.(*pb.Event); ok && event.GetViewRejected() != nil {
		rejected := event.GetViewRejected()
		d.notice("warning", fmt.Sprintf("view rejected rev=%d: %s", rejected.GetRev(), rejected.GetReason()))
	}
}

// tapReader turns a program -> host byte stream into a frame-logging reader:
// complete frames are decoded for the log and handed to the session decoder
// byte for byte. Oversize/zero-length frames keep their protocol semantics.
type tapReader struct {
	src io.Reader
	dev *devMode
	buf []byte
	err error
}

func (t *tapReader) Read(p []byte) (int, error) {
	for len(t.buf) == 0 {
		if t.err != nil {
			return 0, t.err
		}
		frame, err := readRawFrame(t.src)
		if err != nil {
			// A deliberately stopped program closes its stdout: that is the
			// same clean end as EOF, not a protocol error (the reload notice
			// must stay the last word in the footer).
			if errors.Is(err, fs.ErrClosed) || errors.Is(err, io.ErrClosedPipe) {
				err = io.EOF
			}
			if !errors.Is(err, io.EOF) {
				t.dev.logFrameError("program->host", err)
			}
			t.err = err
			return 0, err
		}
		t.dev.logFrame("program->host", frame)
		t.buf = frame
	}
	n := copy(p, t.buf)
	t.buf = t.buf[n:]
	return n, nil
}

// logFrameError records a frame-level protocol failure (zero-length,
// oversize, truncation) in the frame log and surfaces it as a dev notice.
func (d *devMode) logFrameError(dir string, err error) {
	if d == nil {
		return
	}
	if d.log != nil {
		d.log.write(frameRecord{Dir: dir, Type: "ERROR", Error: err.Error()})
	}
	d.notice("warning", "protocol error: "+err.Error())
}

// tapWriter accumulates host -> program writes and logs every complete frame
// before forwarding the original bytes to the program pipe.
type tapWriter struct {
	dst     io.Writer
	dev     *devMode
	pending []byte
}

func (t *tapWriter) Write(p []byte) (int, error) {
	t.pending = append(t.pending, p...)
	for {
		frame, rest, ok := splitRawFrame(t.pending)
		if !ok {
			break
		}
		t.dev.logFrame("host->program", frame)
		t.pending = rest
	}
	return t.dst.Write(p)
}

// readRawFrame reads one complete frame, returning the raw bytes including
// its length prefix. The session's own decoder enforces the size limit, so
// the tap mirrors it: an oversize frame is drained and reported with the
// same *wire.Error the session expects.
func readRawFrame(r io.Reader) ([]byte, error) {
	var prefix [4]byte
	if _, err := io.ReadFull(r, prefix[:]); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(prefix[:])
	if length == 0 {
		return nil, &wire.Error{Kind: wire.KindZeroLength}
	}
	if length > wire.DefaultMaxMessageBytes {
		var typeByte [1]byte
		if _, err := io.ReadFull(r, typeByte[:]); err != nil {
			return nil, err
		}
		if _, err := io.CopyN(io.Discard, r, int64(length)-1); err != nil {
			return nil, err
		}
		return nil, &wire.Error{Kind: wire.KindOversize, Type: wire.Type(typeByte[0]), Limit: wire.DefaultMaxMessageBytes}
	}
	frame := make([]byte, 4+length)
	copy(frame, prefix[:])
	if _, err := io.ReadFull(r, frame[4:]); err != nil {
		return nil, err
	}
	return frame, nil
}

// splitRawFrame extracts one complete frame from a write buffer without
// consuming it on failure.
func splitRawFrame(buffer []byte) (frame, rest []byte, ok bool) {
	if len(buffer) < 4 {
		return nil, buffer, false
	}
	length := binary.BigEndian.Uint32(buffer[:4])
	if length == 0 || uint64(length) > uint64(wire.DefaultMaxMessageBytes) {
		return nil, buffer, false
	}
	if len(buffer) < int(4+length) {
		return nil, buffer, false
	}
	return buffer[:4+length], buffer[4+length:], true
}

// stderrBuffer keeps the last n complete lines written by the layout
// program's stderr without ever blocking the child.
type stderrBuffer struct {
	mu      sync.Mutex
	max     int
	lines   []string
	partial string
}

func newStderrBuffer(max int) *stderrBuffer {
	if max <= 0 {
		max = 64
	}
	return &stderrBuffer{max: max}
}

func (b *stderrBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.partial += string(p)
	for {
		index := strings.IndexByte(b.partial, '\n')
		if index < 0 {
			break
		}
		b.lines = append(b.lines, strings.TrimRight(b.partial[:index], "\r"))
		b.partial = b.partial[index+1:]
		if len(b.lines) > b.max {
			b.lines = b.lines[len(b.lines)-b.max:]
		}
	}
	return len(p), nil
}

// tail returns up to n most recent stderr lines (including an unterminated
// trailing line).
func (b *stderrBuffer) tail(n int) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	lines := append([]string(nil), b.lines...)
	if strings.TrimSpace(b.partial) != "" {
		lines = append(lines, strings.TrimRight(b.partial, "\r"))
	}
	lines = trimEmptyLines(lines)
	if n > 0 && len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

func trimEmptyLines(lines []string) []string {
	out := lines[:0]
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}

// fileWatcher polls a path list and reports the first changed path. Polling
// is deliberate: it has no kernel-watch limits and behaves the same on every
// platform a layout program may run on.
type fileWatcher struct {
	paths    []string
	interval time.Duration
	changed  func(path string)

	stopCh chan struct{}
	doneCh chan struct{}
	once   sync.Once
}

func newFileWatcher(paths []string, interval time.Duration, changed func(path string)) *fileWatcher {
	if interval <= 0 {
		interval = devWatchInterval
	}
	return &fileWatcher{
		paths:    paths,
		interval: interval,
		changed:  changed,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func (w *fileWatcher) start() {
	// Snapshot the baseline synchronously so a change written right after
	// start() cannot be mistaken for the initial state.
	previous := w.fingerprint()
	go w.run(previous)
}

func (w *fileWatcher) close() {
	if w == nil {
		return
	}
	w.once.Do(func() { close(w.stopCh) })
	select {
	case <-w.doneCh:
	case <-time.After(2 * time.Second):
	}
}

func (w *fileWatcher) run(previous map[string]string) {
	defer close(w.doneCh)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			current := w.fingerprint()
			if changed, ok := firstDifference(previous, current); ok {
				previous = current
				if w.changed != nil {
					w.changed(changed)
				}
			}
		}
	}
}

const watchMaxEntries = 20000

// watchHashMaxFile bounds content hashing so a same-size save is detected
// even where the filesystem timestamp granularity makes mtime identical; the
// budget bounds the per-poll cost on large trees.
const (
	watchHashMaxFile = 1 << 20
	watchHashBudget  = 8 << 20
)

// fingerprint snapshots path -> "mtime:size[:hash]". Directories are walked
// so a save inside a source tree triggers a reload; small files also hash
// their content, which catches editors that preserve size and timestamp.
func (w *fileWatcher) fingerprint() map[string]string {
	out := map[string]string{}
	budget := watchHashBudget
	for _, path := range w.paths {
		info, err := os.Stat(path)
		if err != nil {
			out[path] = "missing"
			continue
		}
		if !info.IsDir() {
			out[path] = watchStamp(path, info, &budget)
			continue
		}
		_ = filepath.WalkDir(path, func(entry string, dir os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if dir.IsDir() {
				return nil
			}
			info, err := dir.Info()
			if err != nil {
				return nil
			}
			out[entry] = watchStamp(entry, info, &budget)
			if len(out) >= watchMaxEntries {
				return filepath.SkipAll
			}
			return nil
		})
	}
	return out
}

// watchStamp builds one fingerprint entry, hashing small file contents when
// the remaining budget allows it.
func watchStamp(path string, info os.FileInfo, budget *int) string {
	stamp := fmt.Sprintf("%d:%d", info.ModTime().UnixNano(), info.Size())
	if info.Size() > watchHashMaxFile || *budget < int(info.Size()) {
		return stamp
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return stamp
	}
	*budget -= len(data)
	hash := fnv.New64a()
	_, _ = hash.Write(data)
	return fmt.Sprintf("%s:%x", stamp, hash.Sum64())
}

// firstDifference reports the lexicographically first changed/added/removed
// path between two fingerprints.
func firstDifference(previous, current map[string]string) (string, bool) {
	changed := make([]string, 0, 4)
	for path, stamp := range current {
		if previous[path] != stamp {
			changed = append(changed, path)
		}
	}
	for path := range previous {
		if _, ok := current[path]; !ok {
			changed = append(changed, path)
		}
	}
	if len(changed) == 0 {
		return "", false
	}
	sort.Strings(changed)
	return changed[0], true
}

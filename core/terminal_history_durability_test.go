package core

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/core/history/linehist"
)

type durabilityTerminalLineStorage struct {
	mu          sync.Mutex
	lines       []linehist.Line
	appendCalls int
	syncCalls   int
	closeCalls  int
	appendErr   error
	syncErrs    []error
	closeErr    error
}

func (storage *durabilityTerminalLineStorage) AppendLines(lines []linehist.Line) error {
	if len(lines) == 0 {
		return nil
	}
	storage.mu.Lock()
	defer storage.mu.Unlock()
	storage.appendCalls++
	if storage.appendErr != nil {
		err := storage.appendErr
		storage.appendErr = nil
		return err
	}
	storage.lines = append(storage.lines, lines...)
	return nil
}

func (*durabilityTerminalLineStorage) AppendBoundary() error { return nil }

func (storage *durabilityTerminalLineStorage) LineCount() int {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	return len(storage.lines)
}

func (*durabilityTerminalLineStorage) Base() int { return 0 }

func (storage *durabilityTerminalLineStorage) Lines(start int, end int) ([]linehist.Line, error) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	return append([]linehist.Line(nil), storage.lines[start:end]...), nil
}

func (storage *durabilityTerminalLineStorage) Sync() error {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	storage.syncCalls++
	if len(storage.syncErrs) == 0 {
		return nil
	}
	err := storage.syncErrs[0]
	storage.syncErrs = storage.syncErrs[1:]
	return err
}

// Production CompressedLineFile.Close performs one final file.Sync. This fake
// counts that close-owned sync so tests catch an extra Terminal-level Sync.
func (storage *durabilityTerminalLineStorage) Close() error {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	storage.closeCalls++
	storage.syncCalls++
	return storage.closeErr
}

func (storage *durabilityTerminalLineStorage) counts() (appendCalls int, syncCalls int, closeCalls int) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	return storage.appendCalls, storage.syncCalls, storage.closeCalls
}

func (storage *durabilityTerminalLineStorage) snapshotLines() []linehist.Line {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	return append([]linehist.Line(nil), storage.lines...)
}

func newDurabilityHistoryServer(storage *durabilityTerminalLineStorage) (*Server, *recordingProcessFactory) {
	factory := newRecordingProcessFactory()
	server := NewServer(
		WithProcessFactory(factory),
		WithHistoryStoreFactory(func(id string) (history.HistoryStore, error) {
			return linehist.NewStore(id, linehist.NewEngine(storage)), nil
		}),
	)
	return server, factory
}

func TestTerminalFlushHistoryWaitsForConsumerThenSyncsExactlyOnce(t *testing.T) {
	storage := &durabilityTerminalLineStorage{}
	server, factory := newDurabilityHistoryServer(storage)
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	const terminalID = "term-history-durability-fence"
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID: terminalID, Command: []string{"shell"}, Size: Size{Cols: 80, Rows: 20},
	}); err != nil {
		t.Fatal(err)
	}
	terminal, err := server.Terminal(terminalID)
	if err != nil {
		t.Fatal(err)
	}
	if err := terminal.lineHistory.AppendLifecycleLines([]string{"append one"}); err != nil {
		t.Fatal(err)
	}
	if err := terminal.lineHistory.AppendLifecycleLines([]string{"append two"}); err != nil {
		t.Fatal(err)
	}
	appendCalls, syncCalls, _ := storage.counts()
	if appendCalls < 3 || syncCalls != 0 {
		t.Fatalf("before Flush append calls=%d sync calls=%d, want multiple appends and no sync", appendCalls, syncCalls)
	}
	lineCountBefore := storage.LineCount()

	terminal.tapOpMu.Lock()
	locked := true
	defer func() {
		if locked {
			terminal.tapOpMu.Unlock()
		}
	}()
	factory.process(terminalID).emitOutput("current hot screen")
	terminal.queueMu.Lock()
	buffer := terminal.outputBuffer
	terminal.queueMu.Unlock()
	if buffer == nil {
		t.Fatal("history output buffer is not installed")
	}
	waitForOutputCondition(t, func() bool {
		buffer.mu.Lock()
		defer buffer.mu.Unlock()
		return buffer.consumers[terminalOutputConsumerHistory].inFlight != nil
	}, "history consumer did not enter the blocked ingest")

	flushed := make(chan error, 1)
	go func() { flushed <- terminal.FlushHistory(context.Background()) }()
	waitForOutputCondition(t, func() bool {
		buffer.mu.Lock()
		defer buffer.mu.Unlock()
		return buffer.flushWaiters[terminalOutputConsumerHistory] == 1
	}, "FlushHistory did not wait at the history consumer fence")
	_, syncCalls, _ = storage.counts()
	if syncCalls != 0 {
		t.Fatalf("Sync ran before the consumer fence completed: %d", syncCalls)
	}
	terminal.tapOpMu.Unlock()
	locked = false
	select {
	case err := <-flushed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("FlushHistory did not finish")
	}
	_, syncCalls, _ = storage.counts()
	if syncCalls != 1 {
		t.Fatalf("explicit Flush sync calls = %d, want exactly 1", syncCalls)
	}
	if got := storage.LineCount(); got != lineCountBefore {
		t.Fatalf("Flush sealed the current hot screen: line count=%d want=%d", got, lineCountBefore)
	}
}

func TestTerminalFlushHistorySyncErrorIsRetryable(t *testing.T) {
	wantErr := errors.New("transient history sync failure")
	storage := &durabilityTerminalLineStorage{syncErrs: []error{wantErr, nil}}
	server, _ := newDurabilityHistoryServer(storage)
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	const terminalID = "term-history-sync-retry"
	if _, err := server.RegisterTerminal(TerminalRecord{ID: terminalID, Command: []string{"shell"}}); err != nil {
		t.Fatal(err)
	}
	terminal, err := server.Terminal(terminalID)
	if err != nil {
		t.Fatal(err)
	}
	if err := terminal.FlushHistory(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("first FlushHistory error = %v, want %v", err, wantErr)
	}
	if err := terminal.FlushHistory(context.Background()); err != nil {
		t.Fatalf("retry FlushHistory error = %v", err)
	}
	_, syncCalls, _ := storage.counts()
	if syncCalls != 2 {
		t.Fatalf("retry sync calls = %d, want 2", syncCalls)
	}
	if status := terminal.HistoryBacklogStatus(); status.Unavailable {
		t.Fatalf("transient Sync error became permanently unavailable: %#v", status)
	}
}

func TestTerminalFlushHistoryMakesSmallPendingBlockRecoverable(t *testing.T) {
	dir := t.TempDir()
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()), WithHistoryStorageDir(dir))
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	const terminalID = "term-history-small-pending"
	if _, err := server.RegisterTerminal(TerminalRecord{ID: terminalID, Command: []string{"shell"}}); err != nil {
		t.Fatal(err)
	}
	terminal, err := server.Terminal(terminalID)
	if err != nil {
		t.Fatal(err)
	}
	const marker = "small pending explicit flush"
	if err := terminal.lineHistory.AppendLifecycleLines([]string{marker}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, terminalID+".logical-lines.bin")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Fatalf("small pending history reached disk before FlushHistory: size=%d", info.Size())
	}
	if err := terminal.FlushHistory(context.Background()); err != nil {
		t.Fatal(err)
	}

	reopened, err := linehist.OpenCompressedLineFile(dir, terminalID, linehist.CompressedLineFileOptions{
		MaxBytes: DefaultHistoryMaxBytesPerTerminal, Compression: HistoryCompressionZstd, CompressionLevel: HistoryCompressionLevelFast,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	lines, err := reopened.Lines(0, reopened.LineCount())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, line := range lines {
		if linehist.LineText(line) == marker {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("new file handle did not recover %q from %v", marker, lines)
	}
}

func TestHistoryReadsSyncOnlyWhenCreatingFrozenSnapshot(t *testing.T) {
	storage := &durabilityTerminalLineStorage{}
	server, _ := newDurabilityHistoryServer(storage)
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	const terminalID = "term-history-read-fence"
	if _, err := server.RegisterTerminal(TerminalRecord{ID: terminalID, Command: []string{"shell"}}); err != nil {
		t.Fatal(err)
	}
	terminal, err := server.Terminal(terminalID)
	if err != nil {
		t.Fatal(err)
	}
	if err := terminal.lineHistory.AppendLifecycleLines([]string{"durable snapshot"}); err != nil {
		t.Fatal(err)
	}

	if _, err := server.TerminalHistoryWindow(context.Background(), terminalID, history.HistoryWindowRequest{Limit: 10}); err != nil {
		t.Fatal(err)
	}
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	if _, err := server.TerminalHistoryCopy(context.Background(), terminalID, history.HistoryCopyRequest{TerminalID: terminalID}); !errors.Is(err, history.ErrHistoryInvalidMutation) {
		t.Fatalf("tokenless server copy error = %v, want %v", err, history.ErrHistoryInvalidMutation)
	}
	if _, err := session.ApplicationHistoryCopy(context.Background(), history.HistoryCopyRequest{TerminalID: terminalID}); !errors.Is(err, history.ErrHistoryInvalidMutation) {
		t.Fatalf("tokenless application copy error = %v, want %v", err, history.ErrHistoryInvalidMutation)
	}
	_, syncCalls, _ := storage.counts()
	if syncCalls != 0 {
		t.Fatalf("live history read and rejected copy sync calls = %d, want 0", syncCalls)
	}

	snapshot, err := session.ApplicationHistoryWindow(context.Background(), history.HistoryWindowRequest{
		TerminalID: terminalID,
		Mode:       history.HistoryWindowModeLatest,
		Limit:      10,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, syncCalls, _ = storage.counts()
	if syncCalls != 1 {
		t.Fatalf("history freeze sync calls = %d, want 1", syncCalls)
	}

	if _, err := session.ApplicationHistoryWindow(context.Background(), history.HistoryWindowRequest{
		TerminalID: terminalID,
		Mode:       history.HistoryWindowModeOldest,
		Token:      snapshot.Token,
		Limit:      10,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := session.ApplicationHistoryCopy(context.Background(), history.HistoryCopyRequest{
		TerminalID: terminalID,
		Token:      snapshot.Token,
	}); err != nil {
		t.Fatal(err)
	}
	_, syncCalls, _ = storage.counts()
	if syncCalls != 1 {
		t.Fatalf("frozen history reads sync calls = %d, want 1", syncCalls)
	}
}

func TestTerminalCloseDrainsPayloadAlreadyInSharedBufferAndSyncsOnce(t *testing.T) {
	storage := &durabilityTerminalLineStorage{}
	server, factory := newDurabilityHistoryServer(storage)
	const terminalID = "term-history-close-durable"
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID: terminalID, Command: []string{"shell"}, Size: Size{Cols: 80, Rows: 20},
	}); err != nil {
		t.Fatal(err)
	}
	terminal, err := server.Terminal(terminalID)
	if err != nil {
		t.Fatal(err)
	}
	terminal.tapOpMu.Lock()
	locked := true
	defer func() {
		if locked {
			terminal.tapOpMu.Unlock()
		}
	}()
	// The durability contract starts after this payload has entered the shared
	// buffer; bytes produced after RemoveTerminal starts are future output.
	factory.process(terminalID).emitOutput("close durable payload")
	terminal.queueMu.Lock()
	buffer := terminal.outputBuffer
	terminal.queueMu.Unlock()
	waitForOutputCondition(t, func() bool {
		buffer.mu.Lock()
		defer buffer.mu.Unlock()
		return buffer.consumers[terminalOutputConsumerHistory].inFlight != nil
	}, "history consumer did not enter the blocked close ingest")

	closed := make(chan error, 1)
	go func() { closed <- server.RemoveTerminal(terminalID) }()
	waitForOutputCondition(t, func() bool {
		buffer.mu.Lock()
		defer buffer.mu.Unlock()
		return buffer.flushWaiters[terminalOutputConsumerHistory] == 1
	}, "normal close did not wait at the history consumer fence")
	terminal.tapOpMu.Unlock()
	locked = false
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("normal close did not finish")
	}

	found := false
	for _, line := range storage.snapshotLines() {
		if strings.Contains(linehist.LineText(line), "close durable payload") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("normal close did not persist fenced history: %v", storage.snapshotLines())
	}
	_, syncCalls, closeCalls := storage.counts()
	if syncCalls != 1 || closeCalls != 1 {
		t.Fatalf("normal close sync calls=%d close calls=%d, want one each", syncCalls, closeCalls)
	}
}

func TestRemoveTerminalReturnsJoinedCloseErrorsAndStillPublishesRemoved(t *testing.T) {
	sealErr := errors.New("seal lifecycle tail")
	closeErr := errors.New("close history storage")
	processErr := errors.New("close terminal process")
	storage := &durabilityTerminalLineStorage{closeErr: closeErr}
	server, factory := newDurabilityHistoryServer(storage)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := server.Events(ctx, EventFilter{Types: []EventType{EventTerminalCreated, EventTerminalRemoved}})
	const terminalID = "term-history-remove-errors"
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID: terminalID, Command: []string{"shell"}, Size: Size{Cols: 80, Rows: 20},
	}); err != nil {
		t.Fatal(err)
	}
	assertEvent(t, events, EventTerminalCreated, terminalID)
	if err := server.IngestOutput(context.Background(), terminalID, "hot tail"); err != nil {
		t.Fatal(err)
	}
	storage.mu.Lock()
	storage.appendErr = sealErr
	storage.mu.Unlock()
	factory.process(terminalID).setCloseError(processErr)

	err := server.RemoveTerminal(terminalID)
	for _, want := range []error{sealErr, closeErr, processErr} {
		if !errors.Is(err, want) {
			t.Fatalf("RemoveTerminal error = %v, missing %v", err, want)
		}
	}
	assertEvent(t, events, EventTerminalRemoved, terminalID)
	if _, err := server.GetTerminal(terminalID); !errors.Is(err, ErrTerminalNotFound) {
		t.Fatalf("removed registry entry is still present: %v", err)
	}
	_, syncCalls, closeCalls := storage.counts()
	if syncCalls != 1 || closeCalls != 1 {
		t.Fatalf("normal close sync calls=%d close calls=%d, want one each", syncCalls, closeCalls)
	}
}

func TestServerShutdownReceivesJoinedTerminalCloseErrors(t *testing.T) {
	sealErr := errors.New("shutdown seal lifecycle tail")
	closeErr := errors.New("shutdown close history storage")
	processErr := errors.New("shutdown close terminal process")
	storage := &durabilityTerminalLineStorage{closeErr: closeErr}
	server, factory := newDurabilityHistoryServer(storage)
	const terminalID = "term-history-shutdown-errors"
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID: terminalID, Command: []string{"shell"}, Size: Size{Cols: 80, Rows: 20},
	}); err != nil {
		t.Fatal(err)
	}
	if err := server.IngestOutput(context.Background(), terminalID, "hot shutdown tail"); err != nil {
		t.Fatal(err)
	}
	storage.mu.Lock()
	storage.appendErr = sealErr
	storage.mu.Unlock()
	factory.process(terminalID).setCloseError(processErr)

	err := server.Shutdown(context.Background())
	for _, want := range []error{sealErr, closeErr, processErr} {
		if !errors.Is(err, want) {
			t.Fatalf("Shutdown error = %v, missing %v", err, want)
		}
	}
	_, syncCalls, closeCalls := storage.counts()
	if syncCalls != 1 || closeCalls != 1 {
		t.Fatalf("shutdown sync calls=%d close calls=%d, want one each", syncCalls, closeCalls)
	}
}

func TestProcessExitSealFailureMarksHistoryUnavailableWithoutClosingProcess(t *testing.T) {
	sealErr := errors.New("process exit seal failure")
	storage := &durabilityTerminalLineStorage{}
	server, factory := newDurabilityHistoryServer(storage)
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	const terminalID = "term-history-exit-seal-error"
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID: terminalID, Command: []string{"shell"}, Size: Size{Cols: 80, Rows: 20},
	}); err != nil {
		t.Fatal(err)
	}
	if err := server.IngestOutput(context.Background(), terminalID, "process exit hot tail"); err != nil {
		t.Fatal(err)
	}
	storage.mu.Lock()
	storage.appendErr = sealErr
	storage.mu.Unlock()
	process := factory.process(terminalID)
	process.exit(0)
	waitForTerminalState(t, server, terminalID, TerminalStateExited)
	terminal, err := server.Terminal(terminalID)
	if err != nil {
		t.Fatal(err)
	}
	status := terminal.HistoryBacklogStatus()
	if !status.Unavailable || !strings.Contains(status.UnavailableReason, sealErr.Error()) {
		t.Fatalf("process-exit seal failure was silent: %#v", status)
	}
	_, _, killed, closed := process.snapshot()
	if killed || closed {
		t.Fatalf("history seal failure changed process lifecycle: killed=%v closed=%v", killed, closed)
	}
}

func TestTerminalRestartLogsOldProcessCloseErrorAfterCommittingNewGeneration(t *testing.T) {
	const sensitiveCloseError = "process close failed: path=/private/session/token command=deploy secret=TOP-SECRET-DO-NOT-LOG"
	closeErr := errors.New(sensitiveCloseError)
	storage := &durabilityTerminalLineStorage{}
	factory := newRecordingProcessFactory()
	var logs bytes.Buffer
	server := NewServer(
		WithProcessFactory(factory),
		WithLogger(slog.New(slog.NewTextHandler(&logs, nil))),
		WithHistoryStoreFactory(func(id string) (history.HistoryStore, error) {
			return linehist.NewStore(id, linehist.NewEngine(storage)), nil
		}),
	)
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	const terminalID = "term-restart-close-error"
	if _, err := server.RegisterTerminal(TerminalRecord{ID: terminalID, Command: []string{"shell"}}); err != nil {
		t.Fatal(err)
	}
	eventCtx, cancelEvents := context.WithCancel(context.Background())
	defer cancelEvents()
	events := server.Events(eventCtx, EventFilter{
		TerminalID: terminalID,
		Types:      []EventType{EventTerminalChanged, EventTerminalLiveInvalidated},
	})
	old := factory.process(terminalID)
	old.setCloseError(closeErr)
	var committedEvents []Event
	old.setCloseHook(func() {
		for len(committedEvents) < 2 {
			select {
			case event := <-events:
				committedEvents = append(committedEvents, event)
			default:
				return
			}
		}
	})
	if err := server.RestartTerminal(context.Background(), terminalID); err != nil {
		t.Fatalf("committed restart returned old-generation cleanup error: %v", err)
	}
	if len(committedEvents) != 2 || committedEvents[0].Type != EventTerminalChanged || committedEvents[1].Type != EventTerminalLiveInvalidated {
		t.Fatalf("restart events were not committed before old process cleanup: %#v", committedEvents)
	}
	if !committedEvents[0].LifecycleKnown || committedEvents[0].Terminal == nil || committedEvents[0].Terminal.State != TerminalStateRunning {
		t.Fatalf("restart lifecycle event = %#v", committedEvents[0])
	}
	if committedEvents[1].Live == nil || committedEvents[1].Live.Revision == 0 {
		t.Fatalf("restart live event = %#v", committedEvents[1])
	}
	current := factory.process(terminalID)
	if current == old {
		t.Fatal("restart did not install the new process generation")
	}
	info, err := server.GetTerminal(terminalID)
	if err != nil || info.State != TerminalStateRunning {
		t.Fatalf("new process generation was not published: info=%#v err=%v", info, err)
	}
	if specs := factory.spawnedSpecs(terminalID); len(specs) != 2 {
		t.Fatalf("successful restart triggered retry spawn semantics: %d total spawns", len(specs))
	}
	current.emitOutput("new generation watcher active")
	outputEvent := assertEventValue(t, events, EventTerminalLiveInvalidated, terminalID)
	if outputEvent.Live == nil || outputEvent.Live.Revision <= committedEvents[1].Live.Revision {
		t.Fatalf("new generation watcher did not advance live output: %#v", outputEvent)
	}
	logText := logs.String()
	for _, want := range []string{
		"close previous terminal process after restart failed",
		"terminal_id=" + terminalID,
		"state_before=running",
		"error_kind=process_close_failed",
	} {
		if !strings.Contains(logText, want) {
			t.Fatalf("restart cleanup log missing %q: %s", want, logText)
		}
	}
	if strings.Contains(logText, sensitiveCloseError) {
		t.Fatal("restart cleanup log exposed the old process Close error")
	}
}

package linehist

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type durabilityLineStorage struct {
	lines      []Line
	appendErr  error
	syncErr    error
	closeErr   error
	linesCalls atomic.Int32
	linesSeen  atomic.Int32
	syncCalls  atomic.Int32
	closeCalls atomic.Int32
}

func (storage *durabilityLineStorage) AppendLines(lines []Line) error {
	if storage.appendErr != nil {
		return storage.appendErr
	}
	storage.lines = append(storage.lines, lines...)
	return nil
}

func (storage *durabilityLineStorage) AppendBoundary() error { return nil }
func (storage *durabilityLineStorage) LineCount() int        { return len(storage.lines) }
func (storage *durabilityLineStorage) Base() int             { return 0 }
func (storage *durabilityLineStorage) Lines(start int, end int) ([]Line, error) {
	storage.linesCalls.Add(1)
	return append([]Line(nil), storage.lines[start:end]...), nil
}
func (storage *durabilityLineStorage) VisitLines(start int, end int, reverse bool, visit func(index int, line Line) bool) error {
	storage.linesCalls.Add(1)
	if reverse {
		for index := end - 1; index >= start; index-- {
			storage.linesSeen.Add(1)
			if !visit(index, storage.lines[index]) {
				break
			}
		}
		return nil
	}
	for index := start; index < end; index++ {
		storage.linesSeen.Add(1)
		if !visit(index, storage.lines[index]) {
			break
		}
	}
	return nil
}
func (storage *durabilityLineStorage) Sync() error {
	storage.syncCalls.Add(1)
	return storage.syncErr
}
func (storage *durabilityLineStorage) Close() error {
	storage.closeCalls.Add(1)
	return storage.closeErr
}

type blockingIngestGate struct {
	entered chan struct{}
	release chan struct{}
}

func (gate *blockingIngestGate) Lock() {
	close(gate.entered)
	<-gate.release
}

func (*blockingIngestGate) Unlock() {}

func TestEngineSyncForwardsExactlyOnce(t *testing.T) {
	storage := &durabilityLineStorage{}
	engine := NewEngine(storage)
	if err := engine.Sync(); err != nil {
		t.Fatal(err)
	}
	if got := storage.syncCalls.Load(); got != 1 {
		t.Fatalf("storage sync calls = %d, want 1", got)
	}
}

func TestStoreSyncUsesExistingIngestGate(t *testing.T) {
	storage := &durabilityLineStorage{}
	store := NewStore("sync-gate", NewEngine(storage))
	gate := &blockingIngestGate{entered: make(chan struct{}), release: make(chan struct{})}
	store.Bind(nil, gate)
	done := make(chan error, 1)
	go func() { done <- store.Sync() }()

	select {
	case <-gate.entered:
	case <-time.After(time.Second):
		t.Fatal("Store.Sync did not acquire the ingest gate")
	}
	if got := storage.syncCalls.Load(); got != 0 {
		t.Fatalf("storage Sync ran before the ingest gate was acquired: %d", got)
	}
	close(gate.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := storage.syncCalls.Load(); got != 1 {
		t.Fatalf("storage sync calls = %d, want 1", got)
	}
}

func TestEngineCloseJoinsTailSealAndStorageCloseErrors(t *testing.T) {
	sealErr := errors.New("seal open tail")
	closeErr := errors.New("close line storage")
	storage := &durabilityLineStorage{appendErr: sealErr, closeErr: closeErr}
	engine := NewEngine(storage)
	engine.asm.open = []Run{{Text: "tail"}}

	err := engine.Close()
	if !errors.Is(err, sealErr) || !errors.Is(err, closeErr) {
		t.Fatalf("Engine.Close error = %v, want joined seal and close errors", err)
	}
	if got := storage.closeCalls.Load(); got != 1 {
		t.Fatalf("storage close calls = %d, want 1", got)
	}
}

func TestCompressedLineFileSyncMakesSmallPendingBlockRecoverableBeforeClose(t *testing.T) {
	dir := t.TempDir()
	file, err := OpenCompressedLineFile(dir, "small-pending", CompressedLineFileOptions{Compression: compressionNone})
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	want := Line{Runs: []Run{{Text: "durable small pending"}}, HardEnd: true}
	if err := file.AppendLines([]Line{want}); err != nil {
		t.Fatal(err)
	}
	info, err := file.file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Fatalf("small append reached the file before Sync: size=%d", info.Size())
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenCompressedLineFile(dir, "small-pending", CompressedLineFileOptions{Compression: compressionNone})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := reopened.Lines(0, reopened.LineCount())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || LineText(got[0]) != LineText(want) {
		t.Fatalf("recovered lines = %#v, want %#v", got, []Line{want})
	}
}

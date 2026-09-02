package core

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/shared/transport"
)

func TestServerOptions(t *testing.T) {
	server := NewServer(
		WithSocketPath("/tmp/core-test.sock"),
		WithDefaultSize(100, 30),
		WithHistoryStorageDir("/tmp/core-history"),
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		WithListenerFactory(func(string) (transport.Listener, error) {
			return newFakeListener("unused"), nil
		}),
	)
	if server.SocketPath() != "/tmp/core-test.sock" {
		t.Fatalf("unexpected socket path %q", server.SocketPath())
	}
	if server.DefaultSize() != (Size{Cols: 100, Rows: 30}) {
		t.Fatalf("unexpected default size %#v", server.DefaultSize())
	}
	if server.HistoryStorageDir() != "/tmp/core-history" {
		t.Fatalf("unexpected history storage dir %q", server.HistoryStorageDir())
	}
	if got := server.TerminalOutputBufferConfig(); got != DefaultTerminalOutputBufferConfig() {
		t.Fatalf("unexpected default terminal output buffer %#v", got)
	} else if got.Overflow != TerminalOutputOverflowBlock || got.CapacityBytes != 32<<20 {
		t.Fatalf("production output buffer defaults changed: %#v", got)
	}
	if got := server.TerminalOutputResidentBudget(); got != DefaultTerminalOutputResidentBudgetBytes {
		t.Fatalf("unexpected default terminal output resident budget %d", got)
	}
	if got := server.TerminalResourceSamplingConfig(); got != DefaultTerminalResourceSamplingConfig() || got.Interval != 500*time.Millisecond || got.MaxSamples != 512 {
		t.Fatalf("unexpected default terminal resource sampling config %#v", got)
	}
	if got := server.HistoryStorageConfig(); got.MaxBytesPerTerminal != DefaultHistoryMaxBytesPerTerminal || got.Compression != HistoryCompressionZstd || got.CompressionLevel != HistoryCompressionLevelFast {
		t.Fatalf("unexpected default history storage config %#v", got)
	}
}

func TestServerHistoryStorageConfigOption(t *testing.T) {
	server := NewServer(WithHistoryStorageConfig(HistoryStorageConfig{
		MaxBytesPerTerminal: 64 << 20,
		MaxAge:              14 * 24 * time.Hour,
		Compression:         HistoryCompressionS2,
		CompressionLevel:    HistoryCompressionLevelBest,
	}))
	if got := server.HistoryStorageConfig(); got.MaxBytesPerTerminal != 64<<20 || got.MaxAge != 14*24*time.Hour || got.Compression != HistoryCompressionS2 || got.CompressionLevel != HistoryCompressionLevelBest {
		t.Fatalf("unexpected history storage config %#v", got)
	}
}

func TestServerTerminalOutputBufferConfigOption(t *testing.T) {
	server := NewServer(
		WithTerminalOutputBufferConfig(TerminalOutputBufferConfig{Overflow: TerminalOutputOverflowBlock, CapacityBytes: 7 << 20}),
		WithTerminalOutputResidentBudget(19<<20),
	)
	if got := server.TerminalOutputBufferConfig(); got.Overflow != TerminalOutputOverflowBlock || got.CapacityBytes != 7<<20 {
		t.Fatalf("unexpected terminal output buffer config %#v", got)
	}
	if got := server.TerminalOutputResidentBudget(); got != 19<<20 {
		t.Fatalf("unexpected resident output budget %d", got)
	}
	fallback := NewServer(WithTerminalOutputResidentBudget(MinTerminalOutputResidentBudgetBytes - 1))
	if got := fallback.TerminalOutputResidentBudget(); got != DefaultTerminalOutputResidentBudgetBytes {
		t.Fatalf("unsafe aggregate budget did not fall back to default: %d", got)
	}
}

func TestServerTerminalResourceSamplingConfigOption(t *testing.T) {
	server := NewServer(WithTerminalResourceSamplingConfig(TerminalResourceSamplingConfig{
		Interval:   750 * time.Millisecond,
		MaxSamples: 1024,
	}))
	if got := server.TerminalResourceSamplingConfig(); got.Interval != 750*time.Millisecond || got.MaxSamples != 1024 {
		t.Fatalf("unexpected terminal resource sampling config %#v", got)
	}
	fallback := NewServer(WithTerminalResourceSamplingConfig(TerminalResourceSamplingConfig{
		Interval:   50 * time.Millisecond,
		MaxSamples: 64,
	}))
	if got := fallback.TerminalResourceSamplingConfig(); got != DefaultTerminalResourceSamplingConfig() {
		t.Fatalf("unsafe terminal resource sampling config did not fall back to default: %#v", got)
	}
}

func TestTerminalHistoryBacklogStatusExposesOutputBufferConfig(t *testing.T) {
	server := NewServer(
		WithProcessFactory(newRecordingProcessFactory()),
		WithTerminalOutputBufferConfig(TerminalOutputBufferConfig{Overflow: TerminalOutputOverflowBlock, CapacityBytes: 9 << 20}),
		WithTerminalOutputResidentBudget(23<<20),
	)
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-r2-output-buffer", Command: []string{"shell"}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	status, err := server.TerminalHistoryBacklogStatus("term-r2-output-buffer")
	if err != nil {
		t.Fatalf("history backlog status: %v", err)
	}
	if status.OutputBufferPolicy != TerminalOutputOverflowBlock || status.BufferCapacityBytes != 9<<20 || status.AggregateBudgetBytes != 23<<20 {
		t.Fatalf("status lost output buffer config: %#v", status)
	}
}

func TestServerHistoryDisabledSkipsStoreCreationAndReturnsDisabled(t *testing.T) {
	called := false
	server := NewServer(
		WithProcessFactory(newRecordingProcessFactory()),
		WithHistoryStoreFactory(func(string) (history.HistoryStore, error) {
			called = true
			return nil, nil
		}),
		WithHistoryDisabled(),
	)
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-history-disabled",
		Command: []string{"shell"},
		Size:    Size{Cols: 20, Rows: 3},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if called {
		t.Fatal("history disabled must not create a history store")
	}
	if err := server.IngestOutput(context.Background(), "term-history-disabled", "old\r\nlatest-tail"); err != nil {
		t.Fatalf("ingest output: %v", err)
	}
	rows, err := server.LiveRows("term-history-disabled")
	if err != nil {
		t.Fatalf("live rows: %v", err)
	}
	if got := strings.Join(rows, "\n"); !strings.Contains(got, "latest-tail") {
		t.Fatalf("history disabled must still update native live screen, got %q", got)
	}
	if _, err := server.TerminalHistoryWindow(context.Background(), "term-history-disabled", history.HistoryWindowRequest{
		TerminalID: "term-history-disabled",
		Mode:       history.HistoryWindowModeLatest,
		Limit:      5,
		Cols:       20,
	}); !errors.Is(err, ErrHistoryDisabled) {
		t.Fatalf("expected ErrHistoryDisabled, got %v", err)
	}
}

func TestServerRegistryPublishesEvents(t *testing.T) {
	server := NewServer(WithListenerFactory(func(string) (transport.Listener, error) {
		return newFakeListener("unused"), nil
	}), WithProcessFactory(newRecordingProcessFactory()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := server.Events(ctx, EventFilter{})
	info, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-2",
		Name:    "demo",
		Command: []string{"sh"},
	})
	if err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if info.Size != server.DefaultSize() {
		t.Fatalf("expected default size, got %#v", info.Size)
	}
	if got, err := server.GetTerminal("term-2"); err != nil || got.ID != "term-2" || got.Name != "demo" {
		t.Fatalf("get terminal got=%#v err=%v", got, err)
	}
	assertEvent(t, events, EventTerminalCreated, "term-2")
	if err := server.RemoveTerminal("term-2"); err != nil {
		t.Fatalf("remove terminal: %v", err)
	}
	assertEvent(t, events, EventTerminalRemoved, "term-2")
	if _, err := server.GetTerminal("term-2"); !errors.Is(err, ErrTerminalNotFound) {
		t.Fatalf("expected ErrTerminalNotFound, got %v", err)
	}
}

func TestServerListTerminalsAddsResourceProjection(t *testing.T) {
	sampledAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	factory := &resourceProcessFactory{usage: TerminalResourceUsage{
		PID:            4321,
		CPUPercentX100: 1234,
		MemoryBytes:    64 * 1024 * 1024,
		SampledAt:      sampledAt,
	}}
	server := NewServer(WithProcessFactory(factory))
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-resource", Command: []string{"sh"}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	t.Cleanup(func() { _ = server.RemoveTerminal("term-resource") })
	items := server.ListTerminals()
	if len(items) != 1 {
		t.Fatalf("expected one terminal, got %#v", items)
	}
	if got := items[0].Resources; got.PID != 4321 || got.CPUPercentX100 != 1234 || got.MemoryBytes != 64*1024*1024 || !got.SampledAt.Equal(sampledAt) {
		t.Fatalf("list should include resource projection, got %#v", got)
	}
	if got := items[0].ResourceHistory; len(got) != 1 || got[0] != items[0].Resources {
		t.Fatalf("list should include previously sampled resource history, got %#v", got)
	}
	info, err := server.GetTerminal("term-resource")
	if err != nil {
		t.Fatalf("get terminal: %v", err)
	}
	if !info.Resources.SampledAt.IsZero() {
		t.Fatalf("resource projection must not be written back to registry: %#v", info.Resources)
	}
}

func TestTerminalResourceHistoryIsBoundedAndResetsForNewProcess(t *testing.T) {
	const historyLimit = 513
	terminal := &Terminal{resourceSampling: TerminalResourceSamplingConfig{Interval: time.Second, MaxSamples: historyLimit}}
	base := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	for index := 0; index < historyLimit+5; index++ {
		terminal.recordResourceUsage(TerminalResourceUsage{PID: 101, CPUPercentX100: index, SampledAt: base.Add(time.Duration(index) * time.Second)})
	}
	latest, history := terminal.ResourceSnapshot()
	if len(history) != historyLimit || history[0].CPUPercentX100 != 5 || latest.CPUPercentX100 != historyLimit+4 {
		t.Fatalf("unexpected bounded history: latest=%#v history=%#v", latest, history)
	}
	history[0].CPUPercentX100 = -1
	_, cloned := terminal.ResourceSnapshot()
	if cloned[0].CPUPercentX100 == -1 {
		t.Fatal("resource snapshot aliases terminal-owned history")
	}
	terminal.recordResourceUsage(TerminalResourceUsage{PID: 202, CPUPercentX100: 7, SampledAt: base.Add(2 * time.Minute)})
	latest, history = terminal.ResourceSnapshot()
	if len(history) != 1 || latest.PID != 202 {
		t.Fatalf("new process should reset old resource history: latest=%#v history=%#v", latest, history)
	}
}

func TestServerRegisterTerminalCarriesCreateOptionsToProcessSpec(t *testing.T) {
	factory := newRecordingProcessFactory()
	server := NewServer(WithProcessFactory(factory))
	info, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-peer",
		Command: []string{"sh"},
		Size:    Size{Cols: 90, Rows: 30},
		Options: TerminalCreateOptions{
			Dir:                "/tmp/anytty-peer",
			Env:                []string{"ANYTTY_PEER=1", "ANYTTY_REGION=local"},
			ScrollbackSize:     123,
			ScrollbackMaxBytes: 4567,
			ScrollbackMaxAge:   2 * time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if info.CWD != "/tmp/anytty-peer" || info.LiveCWD != "/tmp/anytty-peer" {
		t.Fatalf("create cwd must enter terminal info, got %#v", info)
	}
	specs := factory.spawnedSpecs("term-peer")
	if len(specs) != 1 {
		t.Fatalf("expected one process spawn, got %#v", specs)
	}
	spec := specs[0]
	if spec.Dir != "/tmp/anytty-peer" || spec.Size != (Size{Cols: 90, Rows: 30}) {
		t.Fatalf("process spec lost dir/size: %#v", spec)
	}
	if got := strings.Join(spec.Env, "\x00"); !strings.Contains(got, "ANYTTY_PEER=1") || !strings.Contains(got, "ANYTTY_REGION=local") {
		t.Fatalf("process spec lost env: %#v", spec.Env)
	}
	if spec.ScrollbackSize != 123 || spec.ScrollbackMaxBytes != 4567 || spec.ScrollbackMaxAge != 2*time.Hour {
		t.Fatalf("process spec lost scrollback contract: %#v", spec)
	}
}

func TestServerRegistryValidatesRecords(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{Command: []string{"sh"}}); !errors.Is(err, ErrInvalidTerminalID) {
		t.Fatalf("expected ErrInvalidTerminalID, got %v", err)
	}
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-1"}); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("expected ErrInvalidCommand, got %v", err)
	}
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-1", Command: []string{"sh"}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-1", Command: []string{"sh"}}); !errors.Is(err, ErrDuplicateTerminal) {
		t.Fatalf("expected ErrDuplicateTerminal, got %v", err)
	}
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-2", Name: "term-1", Command: []string{"sh"}}); !errors.Is(err, ErrDuplicateTerminal) {
		t.Fatalf("expected duplicate terminal name to return ErrDuplicateTerminal, got %v", err)
	}
}

func TestServerMetadataRejectsDuplicateTerminalName(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-1", Name: "alpha", Command: []string{"sh"}}); err != nil {
		t.Fatalf("register first terminal: %v", err)
	}
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-2", Name: "beta", Command: []string{"sh"}}); err != nil {
		t.Fatalf("register second terminal: %v", err)
	}
	if _, err := server.SetMetadata(context.Background(), "term-2", "alpha", nil); !errors.Is(err, ErrDuplicateTerminal) {
		t.Fatalf("expected duplicate metadata name to return ErrDuplicateTerminal, got %v", err)
	}
	info, err := server.GetTerminal("term-2")
	if err != nil {
		t.Fatalf("get terminal: %v", err)
	}
	if info.Name != "beta" {
		t.Fatalf("duplicate rename must not mutate metadata, got %#v", info)
	}
}

func TestServerStorageVersioningListAndEvents(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := server.Events(ctx, EventFilter{
		Types:            []EventType{EventStorageChanged},
		StorageAppID:     "tui",
		StorageScope:     StorageScopePublic,
		StorageOwnerID:   "workspace-main",
		StorageKeyPrefix: "workbench/",
	})

	entry, err := server.StoragePut(context.Background(), StoragePutRequest{
		AppID:   "tui",
		Scope:   StorageScopePublic,
		OwnerID: "workspace-main",
		Key:     "workbench/root",
		Value:   []byte(`{"workspace":"main"}`),
	})
	if err != nil {
		t.Fatalf("storage put: %v", err)
	}
	if entry.Version != 1 || string(entry.Value) != `{"workspace":"main"}` {
		t.Fatalf("unexpected first entry %#v", entry)
	}
	event := assertEventValue(t, events, EventStorageChanged, "")
	if event.Storage == nil || event.Storage.AppID != "tui" || event.Storage.Key != "workbench/root" || event.Storage.Op != StorageOpPut || event.Storage.Version != 1 {
		t.Fatalf("unexpected storage event %#v", event)
	}

	if _, err := server.StoragePut(context.Background(), StoragePutRequest{
		AppID:           "tui",
		Scope:           StorageScopePublic,
		OwnerID:         "workspace-main",
		Key:             "workbench/root",
		Value:           []byte("stale"),
		CheckVersion:    true,
		ExpectedVersion: 99,
	}); !errors.Is(err, ErrStorageVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
	entry, err = server.StoragePut(context.Background(), StoragePutRequest{
		AppID:           "tui",
		Scope:           StorageScopePublic,
		OwnerID:         "workspace-main",
		Key:             "workbench/root",
		Value:           []byte(`{"workspace":"renamed"}`),
		CheckVersion:    true,
		ExpectedVersion: 1,
	})
	if err != nil {
		t.Fatalf("versioned storage put: %v", err)
	}
	if entry.Version != 2 {
		t.Fatalf("expected version 2, got %#v", entry)
	}

	_, err = server.StoragePut(context.Background(), StoragePutRequest{AppID: "tui", Scope: StorageScopePublic, OwnerID: "workspace-main", Key: "other/root", Value: []byte("ignored")})
	if err != nil {
		t.Fatalf("put other key: %v", err)
	}
	list := server.StorageList(context.Background(), "tui", StorageScopePublic, "workspace-main", "workbench/")
	if len(list) != 1 || list[0].Key != "workbench/root" || list[0].Version != 2 {
		t.Fatalf("unexpected storage list %#v", list)
	}
	deleted, err := server.StorageDelete(context.Background(), StorageDeleteRequest{
		AppID:           "tui",
		Scope:           StorageScopePublic,
		OwnerID:         "workspace-main",
		Key:             "workbench/root",
		CheckVersion:    true,
		ExpectedVersion: 2,
	})
	if err != nil {
		t.Fatalf("storage delete: %v", err)
	}
	if !deleted.Deleted || deleted.Version != 3 {
		t.Fatalf("unexpected delete result %#v", deleted)
	}
}

func TestServerListenAndShutdownUseListenerFactory(t *testing.T) {
	listener := newFakeListener("/tmp/core-v2.sock")
	factoryCalled := make(chan string, 1)
	server := NewServer(
		WithSocketPath("/tmp/core-v2.sock"),
		WithListenerFactory(func(path string) (transport.Listener, error) {
			factoryCalled <- path
			return listener, nil
		}),
	)
	events := server.Events(context.Background(), EventFilter{Types: []EventType{EventServerListening}})
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe(context.Background())
	}()
	select {
	case path := <-factoryCalled:
		if path != "/tmp/core-v2.sock" {
			t.Fatalf("listener factory path = %q", path)
		}
	case <-time.After(time.Second):
		t.Fatal("listener factory was not called")
	}
	assertEvent(t, events, EventServerListening, "")
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("listen returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("listen did not stop")
	}
	if !listener.closed() {
		t.Fatal("expected listener to be closed")
	}
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-1", Command: []string{"sh"}}); !errors.Is(err, ErrServerClosed) {
		t.Fatalf("expected ErrServerClosed after shutdown, got %v", err)
	}
}

func TestServerAcceptedTransportClosesOnShutdown(t *testing.T) {
	listener := newFakeListener("/tmp/core-v2.sock")
	conn := newFakeTransport()
	server := NewServer(WithListenerFactory(func(string) (transport.Listener, error) {
		return listener, nil
	}))
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe(context.Background())
	}()
	listener.accept(conn)
	listener.waitAccepted(t)
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	select {
	case <-conn.Done():
	case <-time.After(time.Second):
		t.Fatal("accepted transport was not closed")
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("listen returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("listen did not stop")
	}
}

func TestCoreV2DoesNotImportLegacyRuntime(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	var offenders []string
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if isLegacyRuntimeImport(importPath) {
				offenders = append(offenders, path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk core-v2: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("core-v2 must not import legacy runtime: %v", offenders)
	}
}

func isLegacyRuntimeImport(importPath string) bool {
	for _, legacy := range []string{
		"github.com/anytty/anytty/legacy-core",
		"github.com/anytty/anytty/tuiv2",
	} {
		if importPath == legacy || strings.HasPrefix(importPath, legacy+"/") {
			return true
		}
	}
	return false
}

func assertEvent(t *testing.T, events <-chan Event, typ EventType, terminalID string) {
	t.Helper()
	_ = assertEventValue(t, events, typ, terminalID)
}

func assertEventValue(t *testing.T, events <-chan Event, typ EventType, terminalID string) Event {
	t.Helper()
	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("event channel closed")
		}
		if event.Type != typ || event.TerminalID != terminalID {
			t.Fatalf("unexpected event %#v", event)
		}
		return event
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s event", typ)
	}
	return Event{}
}

type resourceProcessFactory struct {
	usage TerminalResourceUsage
}

func (factory *resourceProcessFactory) Spawn(_ context.Context, spec ProcessSpec) (TerminalProcess, error) {
	return &resourceProcess{
		recordingProcess: &recordingProcess{
			id:       spec.TerminalID,
			outputCh: make(chan []byte, 16),
			waitCh:   make(chan ProcessExit, 1),
		},
		usage: factory.usage,
	}, nil
}

type resourceProcess struct {
	*recordingProcess
	usage TerminalResourceUsage
}

func (process *resourceProcess) ResourceUsage() (TerminalResourceUsage, bool) {
	return process.usage, true
}

type fakeListener struct {
	addr     string
	acceptCh chan transport.Transport
	accepted chan struct{}
	done     chan struct{}
}

func newFakeListener(addr string) *fakeListener {
	return &fakeListener{
		addr:     addr,
		acceptCh: make(chan transport.Transport, 8),
		accepted: make(chan struct{}, 8),
		done:     make(chan struct{}),
	}
}

func (listener *fakeListener) Accept(ctx context.Context) (transport.Transport, error) {
	select {
	case conn := <-listener.acceptCh:
		listener.accepted <- struct{}{}
		return conn, nil
	case <-listener.done:
		return nil, transport.ErrListenerClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (listener *fakeListener) Close() error {
	select {
	case <-listener.done:
	default:
		close(listener.done)
	}
	return nil
}

func (listener *fakeListener) Addr() string {
	return listener.addr
}

func (listener *fakeListener) accept(conn transport.Transport) {
	listener.acceptCh <- conn
}

func (listener *fakeListener) waitAccepted(t *testing.T) {
	t.Helper()
	select {
	case <-listener.accepted:
	case <-time.After(time.Second):
		t.Fatal("listener did not accept transport")
	}
}

func (listener *fakeListener) closed() bool {
	select {
	case <-listener.done:
		return true
	default:
		return false
	}
}

type fakeTransport struct {
	done chan struct{}
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{done: make(chan struct{})}
}

func (transport *fakeTransport) Send([]byte) error {
	return nil
}

func (transport *fakeTransport) Recv() ([]byte, error) {
	<-transport.done
	return nil, io.EOF
}

func (transport *fakeTransport) Close() error {
	select {
	case <-transport.done:
	default:
		close(transport.done)
	}
	return nil
}

func (transport *fakeTransport) Done() <-chan struct{} {
	return transport.done
}

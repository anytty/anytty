package core

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/shared/transport"
	"github.com/anytty/anytty/shared/transport/memory"
)

func TestServerShutdownDeadlineDuringBlockedSpawnKeepsCoordinatorAlive(t *testing.T) {
	factory := newShutdownProcessFactory()
	server := NewServer(WithProcessFactory(factory), WithHistoryDisabled())
	registerDone := make(chan error, 1)
	go func() {
		_, err := server.RegisterTerminal(TerminalRecord{ID: "blocked-spawn", Command: []string{"shell"}})
		registerDone <- err
	}()
	awaitShutdownSignal(t, factory.entered, "process spawn did not start")

	if err := server.Shutdown(expiredShutdownContext(t)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown deadline error = %v", err)
	}
	if !server.closed.Load() {
		t.Fatal("shutdown did not close admission before waiting for lifecycle")
	}
	lateAdmission := make(chan error, 1)
	go func() {
		_, err := server.RegisterTerminal(TerminalRecord{ID: "late", Command: []string{"shell"}})
		lateAdmission <- err
	}()
	if err := awaitShutdownResult(t, lateAdmission, "late terminal admission blocked on lifecycle"); !errors.Is(err, ErrServerClosed) {
		t.Fatalf("late terminal admission error = %v", err)
	}
	assertShutdownPending(t, server.shutdownDone)

	completed := make(chan error, 1)
	go func() { completed <- server.Shutdown(context.Background()) }()
	close(factory.release)
	if err := awaitShutdownResult(t, registerDone, "blocked terminal registration did not finish"); err != nil {
		t.Fatalf("registration that owned lifecycle before shutdown failed: %v", err)
	}
	if err := awaitShutdownResult(t, completed, "shutdown coordinator did not finish"); err != nil {
		t.Fatalf("completed shutdown error = %v", err)
	}
	if got := factory.process.closeCalls.Load(); got != 1 {
		t.Fatalf("spawned process close calls = %d, want 1", got)
	}
}

func TestServerConcurrentShutdownWaitersShareOneResult(t *testing.T) {
	injected := errors.New("transport close failed")
	server := NewServer(WithHistoryDisabled())
	raw := newShutdownCountingTransport(injected)
	tracked, err := server.beginTrackedTransport(raw)
	if err != nil {
		t.Fatal(err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	contexts := []context.Context{expiredShutdownContext(t), canceled, expiredShutdownContext(t), canceled}
	start := make(chan struct{})
	results := make(chan error, len(contexts))
	var wait sync.WaitGroup
	for _, ctx := range contexts {
		wait.Add(1)
		go func(ctx context.Context) {
			defer wait.Done()
			<-start
			results <- server.Shutdown(ctx)
		}(ctx)
	}
	close(start)
	wait.Wait()
	close(results)
	for err := range results {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("incomplete shutdown waiter error = %v", err)
		}
	}
	awaitShutdownSignal(t, raw.done, "shutdown did not close tracked transport")
	assertShutdownPending(t, server.shutdownDone)

	server.finishTrackedTransport(tracked)
	if err := server.Shutdown(context.Background()); !errors.Is(err, injected) {
		t.Fatalf("final shutdown error = %v, want %v", err, injected)
	}
	if err := server.Shutdown(canceled); !errors.Is(err, injected) {
		t.Fatalf("completed shutdown lost priority to canceled context: %v", err)
	}
	if got := raw.closeCalls.Load(); got != 1 {
		t.Fatalf("transport close calls = %d, want 1", got)
	}
}

func TestServerShutdownTracksBlockedGrantActiveBeforeStoreReturns(t *testing.T) {
	now := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	clock := newGrantTestClock(now)
	service := newGrantAccessTestService(map[string]time.Time{"blocked-grant": expiresAt})
	gate := service.gateActive("blocked-grant")
	timers := &grantManualTimerFactory{}
	server := newGrantTransportTestServer(service, clock, timers.afterFunc)
	raw := newGrantCountingTransport()
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.ServeScopedTransport(context.Background(), raw, grantTestScope("blocked-grant", expiresAt))
	}()
	waitGrantStage(t, gate.entered)

	server.mu.Lock()
	trackedBeforeStoreReturn := len(server.transports)
	server.mu.Unlock()
	if trackedBeforeStoreReturn != 1 {
		t.Fatalf("tracked transports before GrantActive return = %d, want 1", trackedBeforeStoreReturn)
	}
	if err := server.Shutdown(expiredShutdownContext(t)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown deadline error = %v", err)
	}
	awaitShutdownSignal(t, raw.done, "shutdown did not close transport blocked in GrantActive")
	assertShutdownPending(t, server.shutdownDone)

	gate.open()
	if err := awaitShutdownResult(t, serveDone, "GrantActive transport root did not finish"); !errors.Is(err, ErrServerClosed) {
		t.Fatalf("blocked GrantActive serve error = %v", err)
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("completed shutdown error = %v", err)
	}
	assertGrantServerEmpty(t, server)
	if got := timers.count(); got != 0 {
		t.Fatalf("grant timers after shutdown = %d, want 0", got)
	}
}

func TestServerShutdownOwnsLateListenerFromBlockedFactory(t *testing.T) {
	factoryEntered := make(chan struct{})
	factoryRelease := make(chan struct{})
	listener := newShutdownListener("late-listener")
	server := NewServer(
		WithHistoryDisabled(),
		WithListenerFactory(func(string) (transport.Listener, error) {
			close(factoryEntered)
			<-factoryRelease
			return listener, nil
		}),
	)
	listening := server.Events(context.Background(), EventFilter{Types: []EventType{EventServerListening}})
	listenDone := make(chan error, 1)
	go func() { listenDone <- server.ListenAndServe(context.Background()) }()
	awaitShutdownSignal(t, factoryEntered, "listener factory did not start")

	if err := server.Shutdown(expiredShutdownContext(t)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown deadline error = %v", err)
	}
	assertShutdownPending(t, server.shutdownDone)
	close(factoryRelease)
	if err := awaitShutdownResult(t, listenDone, "late listener root did not finish"); !errors.Is(err, ErrServerClosed) {
		t.Fatalf("late listener error = %v", err)
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("completed shutdown error = %v", err)
	}
	if got := listener.closeCalls.Load(); got != 1 {
		t.Fatalf("late listener close calls = %d, want 1", got)
	}
	if got := listener.acceptCalls.Load(); got != 0 {
		t.Fatalf("late listener Accept calls = %d, want 0", got)
	}
	if event, ok := <-listening; ok {
		t.Fatalf("late listener published listening event: %#v", event)
	}
}

func TestServerShutdownWinsBetweenAcceptAndTransportAdmission(t *testing.T) {
	raw := newShutdownCountingTransport(nil)
	listener := newGatedAcceptListener(raw)
	server := NewServer(WithHistoryDisabled(), WithListenerFactory(func(string) (transport.Listener, error) {
		return listener, nil
	}))
	listening := server.Events(context.Background(), EventFilter{Types: []EventType{EventServerListening}})
	listenDone := make(chan error, 1)
	go func() { listenDone <- server.ListenAndServe(context.Background()) }()
	assertEvent(t, listening, EventServerListening, "")
	awaitShutdownSignal(t, listener.acceptEntered, "listener did not enter Accept")

	if err := server.Shutdown(expiredShutdownContext(t)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown deadline error = %v", err)
	}
	awaitShutdownSignal(t, listener.closed, "coordinator did not close registered listener")
	assertShutdownPending(t, server.shutdownDone)
	close(listener.acceptRelease)
	if err := awaitShutdownResult(t, listenDone, "listener root did not reject post-shutdown connection"); err != nil {
		t.Fatalf("listener root error = %v", err)
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("completed shutdown error = %v", err)
	}
	awaitShutdownSignal(t, raw.done, "post-shutdown accepted connection was orphaned")
	if got := raw.closeCalls.Load(); got != 1 {
		t.Fatalf("post-shutdown connection close calls = %d, want 1", got)
	}
	if got := listener.closeCalls.Load(); got != 1 {
		t.Fatalf("registered listener close calls = %d, want 1", got)
	}
}

func TestServerShutdownClosesTransportAdmittedAfterAccept(t *testing.T) {
	raw := newShutdownCountingTransport(nil)
	listener := newSingleTransportListener(raw)
	server := NewServer(WithHistoryDisabled(), WithListenerFactory(func(string) (transport.Listener, error) {
		return listener, nil
	}))
	listenDone := make(chan error, 1)
	go func() { listenDone <- server.ListenAndServe(context.Background()) }()
	awaitShutdownSignal(t, raw.recvEntered, "accepted transport did not enter protocol session")

	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error = %v", err)
	}
	if err := awaitShutdownResult(t, listenDone, "listener root did not finish"); err != nil {
		t.Fatalf("listener root error = %v", err)
	}
	if got := raw.closeCalls.Load(); got != 1 {
		t.Fatalf("admitted transport close calls = %d, want 1", got)
	}
	if got := listener.closeCalls.Load(); got != 1 {
		t.Fatalf("listener close calls = %d, want 1", got)
	}
}

func TestServerShutdownWaitsForExecutorIgnoringCancellation(t *testing.T) {
	executor := &blockingProtocolExecutor{started: make(chan struct{}), release: make(chan struct{})}
	server := NewServer(
		WithHistoryDisabled(),
		WithApplicationExecutorFactory(func(ApplicationSessionPort) ApplicationExecutor { return executor }),
	)
	client, daemon := memory.NewPair()
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.ServeTransport(context.Background(), daemon) }()
	completeServerProtocolHello(t, client)
	payload, err := protocol.EncodeApplicationCommand(&apipb.CommandEnvelope{})
	if err != nil {
		t.Fatal(err)
	}
	sendProtocolRequest(t, client, protocol.Request{ID: 1, Method: "api.execute", Params: payload})
	awaitShutdownSignal(t, executor.started, "executor did not start")

	if err := server.Shutdown(expiredShutdownContext(t)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown deadline error = %v", err)
	}
	assertShutdownPending(t, server.shutdownDone)
	close(executor.release)
	if err := awaitShutdownResult(t, serveDone, "transport root did not wait for executor"); err == nil {
		t.Fatal("closed transport root unexpectedly returned nil")
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("completed shutdown error = %v", err)
	}
	if got := len(server.protocolRequestSlots); got != 0 {
		t.Fatalf("server request slots after shutdown = %d, want 0", got)
	}
	_ = client.Close()
}

func expiredShutdownContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(1, 0))
	t.Cleanup(cancel)
	return ctx
}

func assertShutdownPending(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
		t.Fatal("shutdown coordinator completed before its owner was released")
	default:
	}
}

func awaitShutdownSignal(t *testing.T, signal <-chan struct{}, failure string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal(failure)
	}
}

func awaitShutdownResult[T any](t *testing.T, result <-chan T, failure string) T {
	t.Helper()
	select {
	case value := <-result:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal(failure)
		var zero T
		return zero
	}
}

type shutdownProcessFactory struct {
	entered chan struct{}
	release chan struct{}
	process *shutdownCountingProcess
}

func newShutdownProcessFactory() *shutdownProcessFactory {
	return &shutdownProcessFactory{
		entered: make(chan struct{}),
		release: make(chan struct{}),
		process: &shutdownCountingProcess{wait: make(chan ProcessExit)},
	}
}

func (factory *shutdownProcessFactory) Spawn(context.Context, ProcessSpec) (TerminalProcess, error) {
	close(factory.entered)
	<-factory.release
	return factory.process, nil
}

type shutdownCountingProcess struct {
	closeCalls atomic.Int32
	wait       chan ProcessExit
	waitOnce   sync.Once
}

func (*shutdownCountingProcess) Input([]byte) error               { return nil }
func (*shutdownCountingProcess) Resize(Size) error                { return nil }
func (*shutdownCountingProcess) Output() <-chan []byte            { return nil }
func (*shutdownCountingProcess) CancelOutput()                    {}
func (*shutdownCountingProcess) Kill() error                      { return nil }
func (process *shutdownCountingProcess) Wait() <-chan ProcessExit { return process.wait }
func (process *shutdownCountingProcess) Close() error {
	process.closeCalls.Add(1)
	process.waitOnce.Do(func() { close(process.wait) })
	return nil
}

type shutdownCountingTransport struct {
	closeCalls  atomic.Int32
	closeErr    error
	done        chan struct{}
	recvEntered chan struct{}
	recvOnce    sync.Once
	closeOnce   sync.Once
}

func newShutdownCountingTransport(closeErr error) *shutdownCountingTransport {
	return &shutdownCountingTransport{closeErr: closeErr, done: make(chan struct{}), recvEntered: make(chan struct{})}
}

func (*shutdownCountingTransport) Send([]byte) error { return nil }
func (connection *shutdownCountingTransport) Recv() ([]byte, error) {
	connection.recvOnce.Do(func() { close(connection.recvEntered) })
	<-connection.done
	return nil, io.EOF
}
func (connection *shutdownCountingTransport) Close() error {
	connection.closeCalls.Add(1)
	connection.closeOnce.Do(func() { close(connection.done) })
	return connection.closeErr
}
func (connection *shutdownCountingTransport) Done() <-chan struct{} { return connection.done }

type shutdownListener struct {
	addr        string
	closed      chan struct{}
	closeOnce   sync.Once
	closeCalls  atomic.Int32
	acceptCalls atomic.Int32
}

func newShutdownListener(addr string) *shutdownListener {
	return &shutdownListener{addr: addr, closed: make(chan struct{})}
}

func (listener *shutdownListener) Accept(context.Context) (transport.Transport, error) {
	listener.acceptCalls.Add(1)
	<-listener.closed
	return nil, transport.ErrListenerClosed
}
func (listener *shutdownListener) Close() error {
	listener.closeCalls.Add(1)
	listener.closeOnce.Do(func() { close(listener.closed) })
	return nil
}
func (listener *shutdownListener) Addr() string { return listener.addr }

type gatedAcceptListener struct {
	connection    transport.Transport
	acceptEntered chan struct{}
	acceptRelease chan struct{}
	closed        chan struct{}
	closeOnce     sync.Once
	closeCalls    atomic.Int32
}

func newGatedAcceptListener(connection transport.Transport) *gatedAcceptListener {
	return &gatedAcceptListener{
		connection: connection, acceptEntered: make(chan struct{}), acceptRelease: make(chan struct{}), closed: make(chan struct{}),
	}
}

func (listener *gatedAcceptListener) Accept(context.Context) (transport.Transport, error) {
	close(listener.acceptEntered)
	<-listener.acceptRelease
	return listener.connection, nil
}
func (listener *gatedAcceptListener) Close() error {
	listener.closeCalls.Add(1)
	listener.closeOnce.Do(func() { close(listener.closed) })
	return nil
}
func (*gatedAcceptListener) Addr() string { return "gated-accept" }

type singleTransportListener struct {
	connection transport.Transport
	returned   bool
	closed     chan struct{}
	closeOnce  sync.Once
	closeCalls atomic.Int32
}

func newSingleTransportListener(connection transport.Transport) *singleTransportListener {
	return &singleTransportListener{connection: connection, closed: make(chan struct{})}
}

func (listener *singleTransportListener) Accept(context.Context) (transport.Transport, error) {
	if !listener.returned {
		listener.returned = true
		return listener.connection, nil
	}
	<-listener.closed
	return nil, transport.ErrListenerClosed
}
func (listener *singleTransportListener) Close() error {
	listener.closeCalls.Add(1)
	listener.closeOnce.Do(func() { close(listener.closed) })
	return nil
}
func (*singleTransportListener) Addr() string { return "single-transport" }

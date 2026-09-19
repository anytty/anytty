package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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
	factory := newShutdownProcessFactory()
	injected := errors.New("terminal close failed")
	factory.process.closeErr = injected
	server := NewServer(WithProcessFactory(factory), WithHistoryDisabled())
	registerDone := make(chan error, 1)
	go func() {
		_, err := server.RegisterTerminal(TerminalRecord{ID: "blocked-spawn", Command: []string{"shell"}})
		registerDone <- err
	}()
	awaitShutdownSignal(t, factory.entered, "process spawn did not start")

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
	assertShutdownPending(t, server.shutdownDone)

	close(factory.release)
	if err := awaitShutdownResult(t, registerDone, "blocked terminal registration did not finish"); err != nil {
		t.Fatalf("registration that owned lifecycle before shutdown failed: %v", err)
	}
	if err := server.Shutdown(context.Background()); !errors.Is(err, injected) {
		t.Fatalf("final shutdown error = %v, want %v", err, injected)
	}
	if err := server.Shutdown(canceled); !errors.Is(err, injected) {
		t.Fatalf("completed shutdown lost priority to canceled context: %v", err)
	}
	if got := factory.process.closeCalls.Load(); got != 1 {
		t.Fatalf("process close calls = %d, want 1", got)
	}
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
	closeErr   error
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
	return process.closeErr
}

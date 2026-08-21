package core

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/core/history/linehist"
	vterm "github.com/anytty/anytty/vterm/vterm"
)

func TestTerminalHistoryDeltaQueuePreservesOrderAndFlushFence(t *testing.T) {
	queue := newTerminalHistoryDeltaQueue(TerminalOutputBufferConfig{})
	var (
		mu      sync.Mutex
		applied []uint64
	)
	go queue.Run(func(tx vterm.TerminalSemanticTransaction) error {
		mu.Lock()
		applied = append(applied, tx.Seq)
		mu.Unlock()
		return nil
	}, nil, nil)

	for _, seq := range []uint64{11, 12, 13} {
		if !queue.Enqueue(vterm.TerminalSemanticTransaction{Seq: seq}, 1) {
			t.Fatalf("enqueue transaction %d", seq)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := queue.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	queue.Seal()
	queue.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(applied) != 3 || applied[0] != 11 || applied[1] != 12 || applied[2] != 13 {
		t.Fatalf("applied transactions out of order: %v", applied)
	}
}

func TestTerminalHistoryDeltaQueueOverflowIsNonBlockingAndPersistsGap(t *testing.T) {
	queue := newTerminalHistoryDeltaQueue(TerminalOutputBufferConfig{
		CapacityBytes: MinTerminalOutputBufferCapacityBytes,
		Overflow:      TerminalOutputOverflowBlock,
	})
	applyStarted := make(chan struct{})
	releaseApply := make(chan struct{})
	var gapCount int
	go queue.Run(func(vterm.TerminalSemanticTransaction) error {
		close(applyStarted)
		<-releaseApply
		return nil
	}, func() error {
		gapCount++
		return nil
	}, nil)

	if !queue.Enqueue(vterm.TerminalSemanticTransaction{Seq: 1}, 16) {
		t.Fatal("enqueue first transaction")
	}
	select {
	case <-applyStarted:
	case <-time.After(time.Second):
		t.Fatal("history worker did not start")
	}

	huge := vterm.TerminalSemanticTransaction{Seq: 2, EvictedRows: []vterm.TerminalSemanticScrollOut{{
		Cells: make([]vterm.TerminalSemanticCell, 2048),
	}}}
	started := time.Now()
	if !queue.Enqueue(huge, 64<<10) {
		t.Fatal("overflow enqueue was rejected")
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("overflow enqueue blocked live parser for %v", elapsed)
	}

	close(releaseApply)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := queue.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	queue.Seal()
	queue.Wait()
	if gapCount != 1 {
		t.Fatalf("persistent gap count = %d, want 1", gapCount)
	}
	status := queue.Status()
	if status.GapCount != 1 || status.DroppedBytes != 64<<10 || status.ResidentBytes != 0 {
		t.Fatalf("overflow status = %#v", status)
	}
}

func TestSlowHistoryIOCannotDelayLiveInvalidation(t *testing.T) {
	storage := &durabilityTerminalLineStorage{}
	factory := newRecordingProcessFactory()
	server := NewServer(
		WithProcessFactory(factory),
		WithHistoryStoreFactory(func(id string) (history.HistoryStore, error) {
			return linehist.NewStore(id, linehist.NewEngine(storage)), nil
		}),
	)
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	const terminalID = "term-single-vterm-slow-history"
	events := server.Events(context.Background(), EventFilter{
		TerminalID: terminalID,
		Types:      []EventType{EventTerminalLiveInvalidated},
	})
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID: terminalID, Command: []string{"shell"}, Size: Size{Cols: 20, Rows: 2},
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
	factory.process(terminalID).emitOutput("one\r\ntwo\r\nthree\r\n")
	queue := terminal.currentHistoryDeltaQueue()
	waitForOutputCondition(t, func() bool {
		queue.mu.Lock()
		defer queue.mu.Unlock()
		return queue.inFlight != nil
	}, "typed history worker did not reach blocked IO")

	select {
	case event := <-events:
		if event.Live == nil || event.Live.Revision == 0 {
			t.Fatalf("live invalidation = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("live invalidation waited for history IO")
	}
	terminal.tapOpMu.Unlock()
	locked = false
	if err := terminal.waitForHistory(context.Background()); err != nil {
		t.Fatal(err)
	}
}

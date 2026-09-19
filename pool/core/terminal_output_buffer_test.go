package core

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func testOutputConfig(policy TerminalOutputOverflowPolicy) TerminalOutputBufferConfig {
	return TerminalOutputBufferConfig{CapacityBytes: MinTerminalOutputBufferCapacityBytes, Overflow: policy}
}

func TestTerminalOutputBufferStoresPayloadOnceForTwoConsumers(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, true)
	payload := []byte(strings.Repeat("x", int(MinTerminalOutputBufferCapacityBytes)))
	if !buffer.Write(payload) {
		t.Fatal("write rejected")
	}
	status := buffer.Status(terminalOutputConsumerHistory)
	if status.ResidentBytes != int64(len(payload)) || status.AggregateResidentBytes != int64(len(payload)) {
		t.Fatalf("payload must be accounted once, status=%#v", status)
	}
	buffer.Close()
	buffer.Wait()
}

func TestTerminalOutputBufferChunksOversizedWriteInOrder(t *testing.T) {
	budget := newTerminalOutputResidentBudget(2 * MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, false)
	payload := bytes.Repeat([]byte("abcdef"), 12_000)
	var mu sync.Mutex
	var chunks [][]byte
	go buffer.Run(terminalOutputConsumerLive, func(chunk []byte) error {
		mu.Lock()
		chunks = append(chunks, append([]byte(nil), chunk...))
		mu.Unlock()
		return nil
	}, nil)
	if !buffer.Write(payload) {
		t.Fatal("oversized write rejected")
	}
	buffer.Seal()
	buffer.Wait()
	buffer.Close()

	mu.Lock()
	defer mu.Unlock()
	if got := bytes.Join(chunks, nil); !bytes.Equal(got, payload) {
		t.Fatalf("chunked payload changed: got=%d want=%d", len(got), len(payload))
	}
	for _, chunk := range chunks {
		if len(chunk) > terminalOutputChunkBytes {
			t.Fatalf("chunk exceeded limit: %d", len(chunk))
		}
	}
}

func TestTerminalOutputBufferBatchesOnlyAlreadyQueuedSmallWrites(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, false)
	for _, payload := range [][]byte{[]byte("ab"), []byte("cd"), []byte("ef")} {
		if !buffer.Write(payload) {
			t.Fatal("queued write rejected")
		}
	}

	seen := make(chan string, 2)
	go buffer.Run(terminalOutputConsumerLive, func(payload []byte) error {
		seen <- string(payload)
		return nil
	}, nil)
	if err := buffer.Flush(context.Background(), terminalOutputConsumerLive); err != nil {
		t.Fatal(err)
	}
	buffer.Seal()
	buffer.Wait()
	buffer.Close()

	if got := receiveOutputTest(t, seen, "batched payload"); got != "abcdef" {
		t.Fatalf("queued payload batch = %q, want %q", got, "abcdef")
	}
	select {
	case extra := <-seen:
		t.Fatalf("queued small writes should use one ingest call, extra=%q", extra)
	default:
	}
}

func TestTerminalOutputBufferBlockResumesAfterBothConsumersAdvance(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, true)
	payload := []byte(strings.Repeat("a", int(MinTerminalOutputBufferCapacityBytes)))
	if !buffer.Write(payload) {
		t.Fatal("initial write rejected")
	}
	started := make(chan struct{})
	written := make(chan bool, 1)
	go func() {
		close(started)
		written <- buffer.Write([]byte("next"))
	}()
	<-started
	waitForOutputCondition(t, func() bool {
		buffer.mu.Lock()
		defer buffer.mu.Unlock()
		return buffer.localWaiters == 1
	}, "local capacity waiter")

	liveSeen := make(chan struct{}, 8)
	historySeen := make(chan struct{}, 8)
	go buffer.Run(terminalOutputConsumerLive, func([]byte) error {
		liveSeen <- struct{}{}
		return nil
	}, nil)
	go buffer.Run(terminalOutputConsumerHistory, func([]byte) error {
		historySeen <- struct{}{}
		return nil
	}, nil)
	for i := 0; i < int(MinTerminalOutputBufferCapacityBytes)/terminalOutputChunkBytes; i++ {
		receiveOutputTest(t, liveSeen, "live chunk")
		receiveOutputTest(t, historySeen, "history chunk")
	}
	if !receiveOutputTest(t, written, "blocked write") {
		t.Fatal("blocked write rejected after consumers advanced")
	}
	buffer.Seal()
	buffer.Wait()
	status := buffer.Status(terminalOutputConsumerHistory)
	buffer.Close()
	if status.ResidentBytes != 0 || status.AggregateResidentBytes != 0 {
		t.Fatalf("block accounting did not drain: %#v", status)
	}
}

func TestTerminalOutputBufferDropKeepsFastLiveAndStartsHistoryEpoch(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowDrop), budget, true)
	live := make(chan string, 8)
	go buffer.Run(terminalOutputConsumerLive, func(payload []byte) error {
		live <- string(payload)
		return nil
	}, nil)

	first := []byte(strings.Repeat("a", int(MinTerminalOutputBufferCapacityBytes)))
	if !buffer.Write(first) {
		t.Fatal("first write rejected")
	}
	for i := 0; i < len(first)/terminalOutputChunkBytes; i++ {
		receiveOutputTest(t, live, "initial live chunk")
	}
	if !buffer.Write([]byte("tail")) {
		t.Fatal("drop policy rejected fast-consumer write")
	}
	if got := receiveOutputTest(t, live, "live tail"); got != "tail" {
		t.Fatalf("fast live consumer got %q", got)
	}

	history := make(chan string, 4)
	gapDone := make(chan terminalOutputGap, 1)
	go buffer.Run(terminalOutputConsumerHistory, func(payload []byte) error {
		history <- string(payload)
		return nil
	}, func(gap terminalOutputGap) error {
		gapDone <- gap
		return nil
	})
	gap := receiveOutputTest(t, gapDone, "history gap")
	if gap.Epoch != 1 || gap.DroppedBytes == 0 {
		t.Fatalf("invalid history gap: %#v", gap)
	}
	if err := buffer.Flush(context.Background(), terminalOutputConsumerHistory); err != nil {
		t.Fatalf("flush history gap: %v", err)
	}
	if !buffer.Write([]byte("new-epoch")) {
		t.Fatal("write after history gap rejected")
	}
	if got := receiveOutputTest(t, history, "new history epoch"); got != "new-epoch" {
		t.Fatalf("history parser crossed gap: %q", got)
	}
	buffer.Seal()
	buffer.Wait()
	status := buffer.Status(terminalOutputConsumerHistory)
	buffer.Close()
	if status.GapCount != 1 || status.DroppedBytes == 0 || status.ResidentBytes != 0 {
		t.Fatalf("drop diagnostics incorrect: %#v", status)
	}
}

func TestTerminalOutputBufferDropDuringGapCallbackStartsSecondTransition(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowDrop), budget, true)
	live := make(chan string, 16)
	go buffer.Run(terminalOutputConsumerLive, func(payload []byte) error {
		live <- string(payload)
		return nil
	}, nil)
	if !buffer.Write(bytes.Repeat([]byte("x"), int(MinTerminalOutputBufferCapacityBytes))) {
		t.Fatal("initial write rejected")
	}
	for i := 0; i < int(MinTerminalOutputBufferCapacityBytes)/terminalOutputChunkBytes; i++ {
		receiveOutputTest(t, live, "initial live chunk")
	}
	if !buffer.Write([]byte("trigger")) {
		t.Fatal("trigger write rejected")
	}
	receiveOutputTest(t, live, "live trigger")

	firstGapStarted := make(chan terminalOutputGap, 1)
	releaseFirstGap := make(chan struct{})
	secondGapDone := make(chan terminalOutputGap, 1)
	var gapCalls int
	var gapMu sync.Mutex
	history := make(chan string, 1)
	go buffer.Run(terminalOutputConsumerHistory, func(payload []byte) error {
		history <- string(payload)
		return nil
	}, func(gap terminalOutputGap) error {
		gapMu.Lock()
		gapCalls++
		call := gapCalls
		gapMu.Unlock()
		if call == 1 {
			firstGapStarted <- gap
			<-releaseFirstGap
		} else {
			secondGapDone <- gap
		}
		return nil
	})
	receiveOutputTest(t, firstGapStarted, "first gap callback")
	if !buffer.Write([]byte("during-gap")) {
		t.Fatal("write during gap callback rejected")
	}
	receiveOutputTest(t, live, "live output during history gap")
	if !buffer.Write([]byte("during-gap-2")) {
		t.Fatal("second write during gap callback rejected")
	}
	receiveOutputTest(t, live, "second live output during history gap")
	close(releaseFirstGap)
	second := receiveOutputTest(t, secondGapDone, "second gap callback")
	if second.Epoch != 2 || second.DroppedBytes != uint64(len("during-gap")+len("during-gap-2")) {
		t.Fatalf("concurrent drop was lost or folded into delivered gap: %#v", second)
	}
	if err := buffer.Flush(context.Background(), terminalOutputConsumerHistory); err != nil {
		t.Fatalf("flush second history gap: %v", err)
	}
	if !buffer.Write([]byte("after-gaps")) {
		t.Fatal("write after second gap rejected")
	}
	if got := receiveOutputTest(t, history, "history after second gap"); got != "after-gaps" {
		t.Fatalf("history parser saw unexpected payload %q", got)
	}
	buffer.Seal()
	buffer.Wait()
	status := buffer.Status(terminalOutputConsumerHistory)
	buffer.Close()
	if status.GapCount != 2 || status.Epoch != 2 {
		t.Fatalf("gap transitions were not coalesced per callback boundary: %#v", status)
	}
}

func TestTerminalOutputBufferDropReclaimsBehindInFlightHistoryWithoutLosingLive(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowDrop), budget, true)
	liveSeen := make(chan byte, 8)
	liveGap := make(chan terminalOutputGap, 1)
	go buffer.Run(terminalOutputConsumerLive, func(payload []byte) error {
		liveSeen <- payload[0]
		return nil
	}, func(gap terminalOutputGap) error {
		liveGap <- gap
		return ErrTerminalOutputSyncLost
	})
	historyStarted := make(chan struct{})
	releaseHistory := make(chan struct{})
	historyGap := make(chan terminalOutputGap, 1)
	go buffer.Run(terminalOutputConsumerHistory, func([]byte) error {
		select {
		case <-historyStarted:
		default:
			close(historyStarted)
		}
		<-releaseHistory
		return nil
	}, func(gap terminalOutputGap) error {
		historyGap <- gap
		return nil
	})
	writeChunk := func(marker byte) {
		t.Helper()
		if !buffer.Write(bytes.Repeat([]byte{marker}, terminalOutputChunkBytes)) {
			t.Fatalf("write %q rejected", marker)
		}
		if got := receiveOutputTest(t, liveSeen, "fast live chunk"); got != marker {
			t.Fatalf("live chunk=%q want=%q", got, marker)
		}
	}
	writeChunk('a')
	receiveOutputTest(t, historyStarted, "in-flight history")
	writeChunk('b')
	writeChunk('c')
	writeChunk('d')
	writeChunk('e')
	select {
	case gap := <-liveGap:
		t.Fatalf("fast live consumer lost sync: %#v", gap)
	default:
	}
	close(releaseHistory)
	gap := receiveOutputTest(t, historyGap, "history gap behind in-flight node")
	if gap.DroppedBytes == 0 {
		t.Fatal("slow history consumer did not receive a gap")
	}
	buffer.Seal()
	buffer.Wait()
	buffer.Close()
}

func TestTerminalOutputBufferConsumerFailureReleasesCapacity(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, true)
	wantErr := errors.New("history ingest failed")
	failureCalled := make(chan struct{}, 1)
	go buffer.Run(terminalOutputConsumerLive, func([]byte) error { return nil }, nil)
	go buffer.Run(terminalOutputConsumerHistory, func([]byte) error {
		failureCalled <- struct{}{}
		return wantErr
	}, nil)
	if !buffer.Write(bytes.Repeat([]byte("x"), int(MinTerminalOutputBufferCapacityBytes))) {
		t.Fatal("write rejected")
	}
	receiveOutputTest(t, failureCalled, "history failure")
	if err := buffer.Flush(context.Background(), terminalOutputConsumerHistory); !errors.Is(err, wantErr) {
		t.Fatalf("history failure not observable: %v", err)
	}
	if !buffer.Write([]byte("still-live")) {
		t.Fatal("failed history consumer pinned capacity")
	}
	buffer.Seal()
	buffer.Wait()
	buffer.Close()
}

func TestTerminalOutputBufferConsumerFailureReportsFailedAndQueuedRange(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, true)
	releaseFailure := make(chan struct{})
	failureStarted := make(chan struct{})
	failure := make(chan terminalOutputConsumerFailure, 1)
	go buffer.Run(terminalOutputConsumerLive, func([]byte) error { return nil }, nil)
	go buffer.run(terminalOutputConsumerHistory, func([]byte) error {
		close(failureStarted)
		<-releaseFailure
		return errors.New("history write failed")
	}, nil, func(result terminalOutputConsumerFailure) {
		failure <- result
	})
	payload := bytes.Repeat([]byte("x"), 3*terminalOutputChunkBytes)
	written := make(chan bool, 1)
	go func() { written <- buffer.Write(payload) }()
	receiveOutputTest(t, failureStarted, "history failure callback")
	if !receiveOutputTest(t, written, "queued failure range write") {
		t.Fatal("write rejected")
	}
	close(releaseFailure)
	result := receiveOutputTest(t, failure, "typed consumer failure")
	var outputErr *TerminalOutputError
	if !errors.As(result.Err, &outputErr) {
		t.Fatalf("consumer failure was not typed: %v", result.Err)
	}
	if outputErr.DroppedBytes != uint64(len(payload)) || outputErr.Epoch != 1 || result.DuringGap {
		t.Fatalf("consumer failure range=%#v result=%#v", outputErr, result)
	}
	status := buffer.Status(terminalOutputConsumerHistory)
	if status.DroppedBytes != uint64(len(payload)) || status.GapCount != 1 {
		t.Fatalf("consumer failure diagnostics=%#v", status)
	}
	buffer.Seal()
	buffer.Wait()
	buffer.Close()
}

func TestTerminalOutputBufferFlushWaitsForFailureHandling(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, false)
	wantErr := errors.New("live ingest failed")
	failureStarted := make(chan struct{})
	releaseFailure := make(chan struct{})
	go buffer.run(terminalOutputConsumerLive, func([]byte) error {
		return wantErr
	}, nil, func(terminalOutputConsumerFailure) {
		close(failureStarted)
		<-releaseFailure
	})
	if !buffer.Write([]byte("failed output")) {
		t.Fatal("write rejected")
	}
	receiveOutputTest(t, failureStarted, "failure handler")

	flushed := make(chan error, 1)
	go func() {
		flushed <- buffer.Flush(context.Background(), terminalOutputConsumerLive)
	}()
	waitForOutputCondition(t, func() bool {
		buffer.mu.Lock()
		defer buffer.mu.Unlock()
		return buffer.consumers[terminalOutputConsumerLive].failurePending &&
			buffer.flushWaiters[terminalOutputConsumerLive] == 1
	}, "flush waiting for failure handler")
	select {
	case err := <-flushed:
		t.Fatalf("flush returned before failure handling completed: %v", err)
	default:
	}

	close(releaseFailure)
	if err := receiveOutputTest(t, flushed, "handled failure flush"); !errors.Is(err, wantErr) {
		t.Fatalf("flush error=%v want=%v", err, wantErr)
	}
	buffer.Seal()
	buffer.Wait()
	buffer.Close()
}

func TestTerminalOutputResidentBudgetBlocksAcrossBuffers(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	first := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, false)
	second := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, false)
	if !first.Write(bytes.Repeat([]byte("x"), int(MinTerminalOutputBufferCapacityBytes))) {
		t.Fatal("first write rejected")
	}
	written := make(chan bool, 1)
	go func() { written <- second.Write([]byte("second")) }()
	waitForOutputCondition(t, func() bool {
		budget.mu.Lock()
		defer budget.mu.Unlock()
		return budget.waiters == 1
	}, "aggregate budget waiter")
	firstSeen := make(chan struct{}, 8)
	go first.Run(terminalOutputConsumerLive, func([]byte) error {
		firstSeen <- struct{}{}
		return nil
	}, nil)
	receiveOutputTest(t, firstSeen, "aggregate budget release")
	if !receiveOutputTest(t, written, "aggregate-blocked write") {
		t.Fatal("aggregate-blocked write rejected")
	}
	first.Seal()
	first.Wait()
	first.Close()
	go second.Run(terminalOutputConsumerLive, func([]byte) error { return nil }, nil)
	second.Seal()
	second.Wait()
	second.Close()
	resident, limit := budget.status()
	if resident != 0 || limit != MinTerminalOutputBufferCapacityBytes {
		t.Fatalf("aggregate accounting leaked: resident=%d limit=%d", resident, limit)
	}
	if status := second.Status(terminalOutputConsumerLive); status.WaitNanos <= 0 {
		t.Fatalf("aggregate wait was not included in diagnostics: %#v", status)
	}
}

func TestTerminalOutputResidentBudgetCloseRejectsBlockedWriteWithoutGap(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	first := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, false)
	second := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, false)
	if !first.Write(bytes.Repeat([]byte("x"), int(MinTerminalOutputBufferCapacityBytes))) {
		t.Fatal("first write rejected")
	}

	written := make(chan bool, 1)
	go func() { written <- second.Write([]byte("blocked")) }()
	waitForOutputCondition(t, func() bool {
		budget.mu.Lock()
		defer budget.mu.Unlock()
		return budget.waiters == 1
	}, "aggregate budget waiter")

	budget.close()
	if receiveOutputTest(t, written, "budget-close writer") {
		t.Fatal("block writer accepted output after aggregate budget closed")
	}
	status := second.Status(terminalOutputConsumerLive)
	if status.DroppedBytes != 0 || status.GapCount != 0 || status.ResidentBytes != 0 {
		t.Fatalf("block writer converted budget shutdown into loss: %#v", status)
	}
	first.Close()
	first.Wait()
	second.Close()
	second.Wait()
}

func TestTerminalOutputDropNeverWaitsBehindBlockBufferAggregateBudget(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	blockBuffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, false)
	if !blockBuffer.Write(bytes.Repeat([]byte("b"), int(MinTerminalOutputBufferCapacityBytes))) {
		t.Fatal("block buffer write rejected")
	}

	dropBuffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowDrop), budget, false)
	gapSeen := make(chan terminalOutputGap, 1)
	unexpectedPayload := make(chan []byte, 1)
	go dropBuffer.Run(terminalOutputConsumerLive, func([]byte) error {
		unexpectedPayload <- []byte("payload")
		return nil
	}, func(gap terminalOutputGap) error {
		gapSeen <- gap
		return ErrTerminalOutputSyncLost
	})
	written := make(chan bool, 1)
	go func() { written <- dropBuffer.Write([]byte("must-drop")) }()
	if !receiveOutputTest(t, written, "drop write behind block budget") {
		t.Fatal("drop buffer stopped accepting upstream output")
	}
	gap := receiveOutputTest(t, gapSeen, "aggregate gap")
	if gap.DroppedBytes != uint64(len("must-drop")) {
		t.Fatalf("aggregate gap bytes=%d", gap.DroppedBytes)
	}
	dropBuffer.Wait()
	err := dropBuffer.ConsumerError(terminalOutputConsumerLive)
	if !errors.Is(err, ErrTerminalOutputSyncLost) {
		t.Fatalf("aggregate loss did not become typed unavailable state: %v", err)
	}
	select {
	case <-unexpectedPayload:
		t.Fatal("aggregate-rejected payload reached live parser")
	default:
	}
	status := dropBuffer.Status(terminalOutputConsumerLive)
	if status.AggregateResidentBytes > status.AggregateBudgetBytes || status.GapCount != 1 {
		t.Fatalf("aggregate diagnostics invalid: %#v", status)
	}
	dropBuffer.Close()
	blockBuffer.Close()
	blockBuffer.Wait()
}

func TestTerminalOutputBufferSerializesConcurrentWritersWithinCapacity(t *testing.T) {
	budget := newTerminalOutputResidentBudget(2 * MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, false)
	const writers = 12
	started := make(chan struct{}, writers)
	results := make(chan bool, writers)
	payload := bytes.Repeat([]byte("w"), int(MinTerminalOutputBufferCapacityBytes/2))
	for i := 0; i < writers; i++ {
		go func() {
			started <- struct{}{}
			results <- buffer.Write(payload)
		}()
	}
	for i := 0; i < writers; i++ {
		receiveOutputTest(t, started, "concurrent writer start")
	}
	for i := 0; i < 2; i++ {
		if !receiveOutputTest(t, results, "capacity-filling writer") {
			t.Fatal("capacity-filling writer rejected")
		}
	}
	status := buffer.Status(terminalOutputConsumerLive)
	if status.ResidentBytes != MinTerminalOutputBufferCapacityBytes || status.ResidentBytes > status.CapacityBytes {
		t.Fatalf("concurrent writers exceeded local capacity: %#v", status)
	}
	buffer.Close()
	buffer.Wait()
	for i := 2; i < writers; i++ {
		if receiveOutputTest(t, results, "closed concurrent writer") {
			t.Fatal("writer appended after buffer close")
		}
	}
}

func TestTerminalOutputBufferSealUnblocksWriter(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, false)
	if !buffer.Write(bytes.Repeat([]byte("x"), int(MinTerminalOutputBufferCapacityBytes))) {
		t.Fatal("initial write rejected")
	}
	done := make(chan bool, 1)
	go func() { done <- buffer.Write([]byte("blocked")) }()
	buffer.Seal()
	if receiveOutputTest(t, done, "sealed writer") {
		t.Fatal("sealed buffer accepted blocked write")
	}
	buffer.Close()
	buffer.Wait()
}

func TestTerminalOutputBufferCloseBeforeRunMakesWaitFinite(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, true)
	if !buffer.Write([]byte("unread")) {
		t.Fatal("write rejected")
	}
	buffer.Close()
	waited := make(chan struct{})
	go func() {
		buffer.Wait()
		close(waited)
	}()
	receiveOutputTest(t, waited, "close-before-run wait")
	status := buffer.Status(terminalOutputConsumerHistory)
	if status.ResidentBytes != 0 || status.PendingGapBytes != uint64(len("unread")) {
		t.Fatalf("close did not release unread payload as a gap: %#v", status)
	}
}

func TestTerminalOutputBufferFlushCancellationUsesSingleNotifier(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, false)
	if !buffer.Write([]byte("pending")) {
		t.Fatal("write rejected")
	}
	ctx, cancel := context.WithCancel(context.Background())
	flushed := make(chan error, 1)
	go func() { flushed <- buffer.Flush(ctx, terminalOutputConsumerLive) }()
	for i := 0; i < 128; i++ {
		buffer.mu.Lock()
		buffer.cond.Broadcast()
		buffer.mu.Unlock()
	}
	cancel()
	if err := receiveOutputTest(t, flushed, "cancelled flush"); !errors.Is(err, context.Canceled) {
		t.Fatalf("flush cancellation error=%v", err)
	}
	buffer.Close()
	buffer.Wait()
}

func TestTerminalOutputBufferConcurrentFlushWaitersComplete(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, false)
	started := make(chan struct{})
	release := make(chan struct{})
	go buffer.Run(terminalOutputConsumerLive, func([]byte) error {
		close(started)
		<-release
		return nil
	}, nil)
	if !buffer.Write([]byte("pending")) {
		t.Fatal("write rejected")
	}
	receiveOutputTest(t, started, "in-flight ingest")
	first := make(chan error, 1)
	second := make(chan error, 1)
	go func() { first <- buffer.Flush(context.Background(), terminalOutputConsumerLive) }()
	go func() { second <- buffer.Flush(context.Background(), terminalOutputConsumerLive) }()
	close(release)
	if err := receiveOutputTest(t, first, "first flush"); err != nil {
		t.Fatalf("first flush: %v", err)
	}
	if err := receiveOutputTest(t, second, "second flush"); err != nil {
		t.Fatalf("second flush: %v", err)
	}
	buffer.Seal()
	buffer.Wait()
	buffer.Close()
}

func TestTerminalOutputBufferFlushDoesNotWaitForFutureOutput(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, false)
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondStarted := make(chan struct{})
	releaseSecond := make(chan struct{})
	go buffer.Run(terminalOutputConsumerLive, func(payload []byte) error {
		switch string(payload) {
		case "first":
			close(firstStarted)
			<-releaseFirst
		case "second":
			close(secondStarted)
			<-releaseSecond
		}
		return nil
	}, nil)
	if !buffer.Write([]byte("first")) {
		t.Fatal("first write rejected")
	}
	receiveOutputTest(t, firstStarted, "first in-flight output")
	flushed := make(chan error, 1)
	go func() { flushed <- buffer.Flush(context.Background(), terminalOutputConsumerLive) }()
	waitForOutputCondition(t, func() bool {
		buffer.mu.Lock()
		defer buffer.mu.Unlock()
		return buffer.flushWaiters[terminalOutputConsumerLive] == 1
	}, "flush target capture")
	if !buffer.Write([]byte("second")) {
		t.Fatal("future write rejected")
	}
	close(releaseFirst)
	receiveOutputTest(t, secondStarted, "future in-flight output")
	if err := receiveOutputTest(t, flushed, "flush fence"); err != nil {
		t.Fatalf("flush target failed: %v", err)
	}
	close(releaseSecond)
	buffer.Seal()
	buffer.Wait()
	buffer.Close()
}

func TestTerminalOutputBufferRepeatedRunDoesNotFinishActiveConsumer(t *testing.T) {
	budget := newTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes)
	buffer := newTerminalOutputBuffer(testOutputConfig(TerminalOutputOverflowBlock), budget, false)
	started := make(chan struct{})
	release := make(chan struct{})
	go buffer.Run(terminalOutputConsumerLive, func([]byte) error {
		close(started)
		<-release
		return nil
	}, nil)
	if !buffer.Write([]byte("in-flight")) {
		t.Fatal("write rejected")
	}
	receiveOutputTest(t, started, "first runner")
	secondReturned := make(chan struct{})
	go func() {
		buffer.Run(terminalOutputConsumerLive, nil, nil)
		close(secondReturned)
	}()
	receiveOutputTest(t, secondReturned, "duplicate runner return")
	select {
	case <-buffer.consumers[terminalOutputConsumerLive].done:
		t.Fatal("duplicate Run marked the active consumer done")
	default:
	}
	close(release)
	buffer.Seal()
	buffer.Wait()
	buffer.Close()
}

func receiveOutputTest[T any](t *testing.T, channel <-chan T, description string) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
		var zero T
		return zero
	}
}

func waitForOutputCondition(t *testing.T, condition func() bool, description string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", description)
		}
		time.Sleep(time.Millisecond)
	}
}

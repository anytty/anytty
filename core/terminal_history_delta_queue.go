package core

import (
	"context"
	"sync"

	vterm "github.com/anytty/anytty/vterm/vterm"
)

type terminalHistoryDeltaNode struct {
	seq           uint64
	sourceBytes   uint64
	residentBytes int64
	tx            vterm.TerminalSemanticTransaction
	next          *terminalHistoryDeltaNode
}

// terminalHistoryDeltaQueue decouples the single live parser from history IO.
// Enqueue never waits for the worker. When its memory bound is reached, queued
// deltas collapse into an ordered gap that the worker persists before later
// transactions; the live emulator remains authoritative and is never reset.
type terminalHistoryDeltaQueue struct {
	mu       sync.Mutex
	changed  chan struct{}
	done     chan struct{}
	doneOnce sync.Once

	capacity int64
	policy   TerminalOutputOverflowPolicy
	head     *terminalHistoryDeltaNode
	tail     *terminalHistoryDeltaNode
	inFlight *terminalHistoryDeltaNode
	resident int64

	nextSeq            uint64
	completedSeq       uint64
	pendingGapBytes    uint64
	pendingGapThrough  uint64
	gapInFlightBytes   uint64
	gapInFlightThrough uint64
	droppedBytes       uint64
	gapCount           uint64
	epoch              uint64
	started            bool
	sealed             bool
	closed             bool
	unavailable        error
	flushWaiters       int
}

// terminalHistoryViewGate freezes the emulator, drains every delta produced by
// that emulator state, and then serializes the cold/hot snapshot with history
// mutation. The worker never needs liveOpMu, so slow history IO cannot block
// live parsing outside an explicit query or durability fence.
type terminalHistoryViewGate struct {
	terminal *Terminal
}

func (gate *terminalHistoryViewGate) Lock() {
	if gate == nil || gate.terminal == nil {
		return
	}
	gate.terminal.liveOpMu.Lock()
	queue := gate.terminal.currentHistoryDeltaQueue()
	if queue != nil {
		_ = queue.Flush(context.Background())
	}
	gate.terminal.tapOpMu.Lock()
}

func (gate *terminalHistoryViewGate) Unlock() {
	if gate == nil || gate.terminal == nil {
		return
	}
	gate.terminal.tapOpMu.Unlock()
	gate.terminal.liveOpMu.Unlock()
}

func newTerminalHistoryDeltaQueue(config TerminalOutputBufferConfig) *terminalHistoryDeltaQueue {
	config = config.normalized()
	return &terminalHistoryDeltaQueue{
		capacity: config.CapacityBytes,
		policy:   config.Overflow,
		changed:  make(chan struct{}),
		done:     make(chan struct{}),
	}
}

func (queue *terminalHistoryDeltaQueue) Enqueue(tx vterm.TerminalSemanticTransaction, sourceBytes int) bool {
	if queue == nil {
		return false
	}
	if sourceBytes < 0 {
		sourceBytes = 0
	}
	if sourceBytes == 0 {
		sourceBytes = 1
	}
	resident := historyTransactionResidentBytes(tx)
	queue.mu.Lock()
	defer queue.mu.Unlock()
	if queue.closed || queue.sealed || queue.unavailable != nil {
		return false
	}
	queue.nextSeq++
	seq := queue.nextSeq
	if queue.resident+resident > queue.capacity {
		queue.dropQueuedLocked()
	}
	if queue.resident+resident > queue.capacity {
		queue.addGapLocked(seq, uint64(sourceBytes))
		queue.notifyLocked()
		return true
	}
	node := &terminalHistoryDeltaNode{
		seq: seq, sourceBytes: uint64(sourceBytes), residentBytes: resident, tx: tx,
	}
	if queue.tail == nil {
		queue.head = node
	} else {
		queue.tail.next = node
	}
	queue.tail = node
	queue.resident += resident
	queue.notifyLocked()
	return true
}

func (queue *terminalHistoryDeltaQueue) Run(apply func(vterm.TerminalSemanticTransaction) error, gap func() error, failed func(error)) {
	if queue == nil {
		return
	}
	queue.mu.Lock()
	if queue.started || queue.closed {
		queue.mu.Unlock()
		return
	}
	queue.started = true
	queue.mu.Unlock()
	defer queue.finish()

	for {
		node, gapBytes, gapThrough, ok := queue.next()
		if !ok {
			return
		}
		var err error
		if gapBytes > 0 {
			if gap != nil {
				err = gap()
			}
			queue.completeGap(gapThrough, err)
		} else {
			if apply != nil {
				err = apply(node.tx)
			}
			queue.completeNode(node, err)
		}
		if err != nil {
			if failed != nil {
				failed(err)
			}
			return
		}
	}
}

func (queue *terminalHistoryDeltaQueue) next() (*terminalHistoryDeltaNode, uint64, uint64, bool) {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	for !queue.closed && queue.unavailable == nil && queue.pendingGapBytes == 0 && queue.head == nil && !queue.sealed {
		changed := queue.changed
		queue.mu.Unlock()
		<-changed
		queue.mu.Lock()
	}
	if queue.closed || queue.unavailable != nil {
		return nil, 0, 0, false
	}
	if queue.pendingGapBytes > 0 {
		queue.gapInFlightBytes = queue.pendingGapBytes
		queue.gapInFlightThrough = queue.pendingGapThrough
		queue.pendingGapBytes = 0
		queue.pendingGapThrough = 0
		return nil, queue.gapInFlightBytes, queue.gapInFlightThrough, true
	}
	if queue.head == nil {
		return nil, 0, 0, false
	}
	node := queue.head
	queue.head = node.next
	if queue.head == nil {
		queue.tail = nil
	}
	node.next = nil
	queue.inFlight = node
	return node, 0, 0, true
}

func (queue *terminalHistoryDeltaQueue) completeNode(node *terminalHistoryDeltaNode, err error) {
	queue.mu.Lock()
	if queue.inFlight == node {
		queue.inFlight = nil
		queue.resident -= node.residentBytes
		if queue.resident < 0 {
			queue.resident = 0
		}
		if err == nil && node.seq > queue.completedSeq {
			queue.completedSeq = node.seq
		}
	}
	if err != nil {
		queue.failLocked(err, node.seq, node.sourceBytes)
	}
	queue.notifyLocked()
	queue.mu.Unlock()
}

func (queue *terminalHistoryDeltaQueue) completeGap(through uint64, err error) {
	queue.mu.Lock()
	bytes := queue.gapInFlightBytes
	queue.gapInFlightBytes = 0
	queue.gapInFlightThrough = 0
	if err == nil {
		queue.epoch++
		if through > queue.completedSeq {
			queue.completedSeq = through
		}
	} else {
		queue.failLocked(err, through, bytes)
	}
	queue.notifyLocked()
	queue.mu.Unlock()
}

func (queue *terminalHistoryDeltaQueue) failLocked(err error, through uint64, bytes uint64) {
	if queue.unavailable == nil {
		queue.unavailable = err
	}
	if bytes > 0 {
		queue.droppedBytes += bytes
	}
	queue.dropQueuedLocked()
	if through > queue.completedSeq {
		queue.completedSeq = through
	}
}

func (queue *terminalHistoryDeltaQueue) dropQueuedLocked() {
	for node := queue.head; node != nil; node = node.next {
		queue.addGapLocked(node.seq, node.sourceBytes)
		queue.resident -= node.residentBytes
	}
	if queue.resident < 0 {
		queue.resident = 0
	}
	queue.head = nil
	queue.tail = nil
}

func (queue *terminalHistoryDeltaQueue) addGapLocked(through uint64, bytes uint64) {
	if bytes == 0 {
		return
	}
	if queue.pendingGapBytes == 0 {
		queue.gapCount++
	}
	queue.pendingGapBytes += bytes
	queue.droppedBytes += bytes
	if through > queue.pendingGapThrough {
		queue.pendingGapThrough = through
	}
}

func (queue *terminalHistoryDeltaQueue) Flush(ctx context.Context) error {
	if queue == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	queue.mu.Lock()
	target := queue.nextSeq
	queue.flushWaiters++
	defer func() {
		queue.flushWaiters--
		queue.mu.Unlock()
	}()
	for queue.completedSeq < target && queue.unavailable == nil && !queue.closed {
		changed := queue.changed
		queue.mu.Unlock()
		select {
		case <-ctx.Done():
			queue.mu.Lock()
			return ctx.Err()
		case <-changed:
			queue.mu.Lock()
		}
	}
	return queue.unavailable
}

func (queue *terminalHistoryDeltaQueue) Seal() {
	if queue == nil {
		return
	}
	queue.mu.Lock()
	queue.sealed = true
	queue.notifyLocked()
	queue.mu.Unlock()
}

func (queue *terminalHistoryDeltaQueue) Close() {
	if queue == nil {
		return
	}
	queue.mu.Lock()
	queue.closed = true
	queue.dropQueuedLocked()
	started := queue.started
	queue.notifyLocked()
	queue.mu.Unlock()
	if !started {
		queue.finish()
	}
}

func (queue *terminalHistoryDeltaQueue) Wait() {
	if queue != nil {
		<-queue.done
	}
}

func (queue *terminalHistoryDeltaQueue) finish() {
	queue.doneOnce.Do(func() { close(queue.done) })
}

func (queue *terminalHistoryDeltaQueue) Status() terminalOutputBufferStatus {
	if queue == nil {
		return terminalOutputBufferStatus{Closed: true}
	}
	queue.mu.Lock()
	defer queue.mu.Unlock()
	status := terminalOutputBufferStatus{
		Policy:          queue.policy,
		CapacityBytes:   queue.capacity,
		ResidentBytes:   queue.resident,
		DroppedBytes:    queue.droppedBytes,
		GapCount:        queue.gapCount,
		Epoch:           queue.epoch,
		PendingGapBytes: queue.pendingGapBytes + queue.gapInFlightBytes,
		Closed:          queue.closed || (queue.sealed && queue.head == nil && queue.inFlight == nil),
	}
	if queue.unavailable != nil {
		status.Unavailable = true
		status.UnavailableReason = queue.unavailable.Error()
	}
	return status
}

func (queue *terminalHistoryDeltaQueue) notifyLocked() {
	close(queue.changed)
	queue.changed = make(chan struct{})
}

func historyTransactionResidentBytes(tx vterm.TerminalSemanticTransaction) int64 {
	const (
		transactionOverhead = int64(256)
		rowOverhead         = int64(96)
		cellOverhead        = int64(128)
		runOverhead         = int64(96)
	)
	total := transactionOverhead
	for _, row := range tx.EvictedRows {
		total += rowOverhead
		for _, cell := range row.Cells {
			total += cellOverhead + int64(len(cell.Content)+len(cell.LinkURL)+len(cell.LinkParams)+len(cell.Style.FG)+len(cell.Style.BG)+len(cell.Style.LinkURL)+len(cell.Style.LinkParams))
		}
		for _, run := range row.Runs {
			total += runOverhead + int64(len(run.Text)+len(run.Style.FG)+len(run.Style.BG)+len(run.Style.LinkURL)+len(run.Style.LinkParams))
		}
	}
	return total
}
